package shelltest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	runtimeProbeOutputLimit = 1 << 20
	runtimeProbeTimeout     = 10 * time.Second
)

type verifiedRuntime struct {
	executable  string
	version     string
	fingerprint executableFingerprint
}

type executableFingerprint struct {
	info   os.FileInfo
	size   int64
	mode   os.FileMode
	digest [sha256.Size]byte
}

func verifyRuntime(parent context.Context, workspace string, runtime Runtime) (verifiedRuntime, error) {
	resolved, err := exec.LookPath(runtime.Executable)
	if err != nil {
		return verifiedRuntime{}, &failureError{
			subject: "resolve runtime executable",
			detail:  strconv.Quote(runtime.Executable),
			cause:   err,
		}
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return verifiedRuntime{}, &failureError{
			subject: "resolve absolute runtime executable",
			detail:  strconv.Quote(resolved),
			cause:   err,
		}
	}

	fingerprint, err := fingerprintExecutable(resolved)
	if err != nil {
		return verifiedRuntime{}, err
	}

	probeEnvironment, err := frameworkEnvironment(workspace)
	if err != nil {
		return verifiedRuntime{}, err
	}
	ctx, cancel := context.WithTimeout(parent, runtimeProbeTimeout)
	defer cancel()

	//nolint:gosec // Runtime.Executable is the explicitly supplied shell runtime under test.
	command := exec.CommandContext(ctx, resolved, runtime.VersionArgs...)
	command.Dir = workspace
	command.Env = probeEnvironment
	output := &boundedOutput{limit: runtimeProbeOutputLimit}
	command.Stdout = output
	command.Stderr = output
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return verifiedRuntime{}, &failureError{
				subject: "probe runtime version",
				cause:   ctx.Err(),
			}
		}
		return verifiedRuntime{}, &failureError{
			subject: "probe runtime version",
			detail:  boundedFailureDetail("output=" + strconv.Quote(output.String())),
			cause:   err,
		}
	}
	if output.exceeded {
		return verifiedRuntime{}, &failureError{
			subject: "probe runtime version output exceeded byte limit",
			detail:  strconv.Itoa(runtimeProbeOutputLimit),
		}
	}

	probedFingerprint, err := fingerprintExecutable(resolved)
	if err != nil {
		return verifiedRuntime{}, err
	}
	if !fingerprint.equal(probedFingerprint) {
		return verifiedRuntime{}, &failureError{subject: "runtime executable changed during version probe"}
	}

	probeOutput := output.String()
	version, err := runtime.ParseVersion(probeOutput)
	if err != nil {
		return verifiedRuntime{}, &failureError{
			subject: "parse runtime version",
			cause:   err,
		}
	}
	if version == "" {
		return verifiedRuntime{}, &failureError{subject: "parse runtime version returned an empty token"}
	}
	if !strings.Contains(probeOutput, version) {
		return verifiedRuntime{}, &failureError{
			subject: "parsed runtime version was not observed verbatim in probe output",
			detail:  boundedFailureDetail(strconv.Quote(version)),
		}
	}

	verified := verifiedRuntime{
		executable:  resolved,
		version:     version,
		fingerprint: fingerprint,
	}
	return verified, nil
}

func fingerprintExecutable(path string) (executableFingerprint, error) {
	//nolint:gosec // path is the exact executable spelling selected by exec.LookPath for runtime verification.
	file, err := os.Open(path)
	if err != nil {
		return executableFingerprint{}, &failureError{
			subject: "open runtime executable",
			detail:  strconv.Quote(path),
			cause:   err,
		}
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return executableFingerprint{}, &failureError{
			subject: "stat runtime executable",
			detail:  strconv.Quote(path),
			cause:   err,
		}
	}
	if !info.Mode().IsRegular() {
		return executableFingerprint{}, &failureError{
			subject: "runtime executable is not a regular file",
			detail:  strconv.Quote(path),
		}
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return executableFingerprint{}, &failureError{
			subject: "hash runtime executable",
			detail:  strconv.Quote(path),
			cause:   err,
		}
	}
	var digest [sha256.Size]byte
	hash.Sum(digest[:0])

	fingerprint := executableFingerprint{
		info:   info,
		size:   info.Size(),
		mode:   info.Mode(),
		digest: digest,
	}
	return fingerprint, nil
}

func (left executableFingerprint) equal(right executableFingerprint) bool {
	if left.info == nil || right.info == nil {
		return false
	}
	return os.SameFile(left.info, right.info) && left.size == right.size && left.mode == right.mode && left.digest == right.digest
}

func verifyRuntimeExecutable(runtime verifiedRuntime) error {
	fingerprint, err := fingerprintExecutable(runtime.executable)
	if err != nil {
		return err
	}
	if !runtime.fingerprint.equal(fingerprint) {
		return &failureError{subject: "runtime executable changed after version verification"}
	}
	return nil
}

type boundedOutput struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (output *boundedOutput) Write(data []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()

	remaining := output.limit - output.buffer.Len()
	if remaining <= 0 {
		output.exceeded = true
		return len(data), nil
	}
	written := len(data)
	if len(data) > remaining {
		output.exceeded = true
		data = data[:remaining]
	}
	_, _ = output.buffer.Write(data)
	return written, nil
}

func (output *boundedOutput) String() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.buffer.String()
}
