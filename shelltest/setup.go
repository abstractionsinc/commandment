package shelltest

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

type setupSessionState uint8

const (
	setupSessionActive setupSessionState = iota + 1
	setupSessionClosed
)

type setupSession struct {
	mu   sync.Mutex
	uses sync.WaitGroup

	workspace   string
	executable  string
	version     string
	integration []byte
	reserved    []string

	files       []setupFile
	environment []setupEnvironment
	args        []string
	failure     error

	state setupSessionState
	inUse bool
}

type setupPlan struct {
	files       []setupFile
	environment []setupEnvironment
	args        []string
}

type setupFile struct {
	name    string
	content []byte
}

type setupEnvironment struct {
	name  string
	value string
}

func runSetup(prepare PrepareFunc, workspace string, verified verifiedRuntime, generated generatedCompletion) (setupPlan, error) {
	if prepare == nil {
		return setupPlan{}, nil
	}

	session := &setupSession{
		workspace:   workspace,
		executable:  verified.executable,
		version:     verified.version,
		integration: generated.integration,
		reserved:    generated.reserved,
		state:       setupSessionActive,
	}

	var callbackErr error
	func() {
		defer func() {
			session.close()
		}()
		callbackErr = prepare(Setup{session: session})
	}()
	if callbackErr != nil {
		return setupPlan{}, &failureError{
			subject: "prepare target",
			cause:   callbackErr,
		}
	}
	if session.failure != nil {
		return setupPlan{}, session.failure
	}

	plan := setupPlan{
		files:       session.files,
		environment: session.environment,
		args:        session.args,
	}
	return plan, nil
}

func (session *setupSession) close() {
	session.mu.Lock()
	if session.state == setupSessionClosed {
		session.mu.Unlock()
		return
	}
	crossedBoundary := session.inUse
	session.state = setupSessionClosed
	session.mu.Unlock()

	session.uses.Wait()
	if crossedBoundary && session.failure == nil {
		session.failure = &failureError{subject: "Setup operation crossed Prepare callback boundary"}
	}
	session.integration = nil
	session.reserved = nil
}

// Executable returns the exact resolved executable used for both the version probe and the real shell process.
func (setup Setup) Executable() string {
	session := setup.acquire()
	defer setup.release(session)
	return session.executable
}

// Version returns the verified semantic runtime version derived from the exact probe.
func (setup Setup) Version() string {
	session := setup.acquire()
	defer setup.release(session)
	return session.version
}

// IntegrationFile declares materialization of the exact production-generated completion integration at name and returns its future absolute path.
func (setup Setup) IntegrationFile(name string) string {
	session := setup.acquire()
	defer setup.release(session)
	return session.declareFile(name, session.integration)
}

// File declares one case-local test-owned text file and returns its future absolute path. The file is materialized only after PrepareFunc returns.
func (setup Setup) File(name, contents string) string {
	session := setup.acquire()
	defer setup.release(session)
	if !utf8.ValidString(contents) {
		session.recordFailure(&failureError{subject: "test file contents are not valid UTF-8"})
		return ""
	}
	return session.declareFile(name, []byte(contents))
}

// Env declares one shell-runtime environment binding. Repeating the same value is idempotent; a conflicting duplicate invalidates setup.
func (setup Setup) Env(name, value string) {
	session := setup.acquire()
	defer setup.release(session)
	if !validEnvironmentName(name) || strings.IndexByte(value, 0) >= 0 {
		session.recordFailure(&failureError{
			subject: "invalid environment declaration",
			detail:  strconv.Quote(name),
		})
		return
	}
	key := environmentKey(name)
	for _, existing := range session.environment {
		if environmentKey(existing.name) != key {
			continue
		}
		if existing.value != value {
			session.recordFailure(&failureError{
				subject: "conflicting environment declaration",
				detail:  strconv.Quote(name),
			})
		}
		return
	}
	session.environment = append(session.environment, setupEnvironment{
		name:  name,
		value: value,
	})
}

// Args appends provider-owned shell startup arguments. It cannot replace argv[0].
func (setup Setup) Args(args ...string) {
	session := setup.acquire()
	defer setup.release(session)
	for _, arg := range args {
		if strings.IndexByte(arg, 0) >= 0 {
			session.recordFailure(&failureError{subject: "shell startup argument contains NUL"})
			return
		}
	}
	session.args = append(session.args, args...)
}

func (setup Setup) acquire() *setupSession {
	if setup.session == nil {
		panic("shelltest: invalid Setup")
	}
	session := setup.session
	session.mu.Lock()
	if session.state != setupSessionActive {
		session.mu.Unlock()
		panic("shelltest: Setup used outside Prepare callback")
	}
	if session.inUse {
		session.mu.Unlock()
		panic("shelltest: Setup used concurrently")
	}
	session.inUse = true
	session.uses.Add(1)
	session.mu.Unlock()
	return session
}

func (setup Setup) release(session *setupSession) {
	session.mu.Lock()
	session.inUse = false
	session.mu.Unlock()
	session.uses.Done()
}

func (session *setupSession) declareFile(name string, content []byte) string {
	clean, err := validateSetupFileName(name)
	if err != nil {
		session.recordFailure(err)
		return ""
	}
	for _, existing := range session.files {
		if setupFileNamesCollide(existing.name, clean) {
			session.recordFailure(&failureError{
				subject: "setup file collision",
				detail:  strconv.Quote(existing.name) + " and " + strconv.Quote(clean),
			})
			return ""
		}
	}
	for _, reserved := range session.reserved {
		if setupFileNamesCollide(reserved, clean) {
			session.recordFailure(&failureError{
				subject: "setup file collides with framework path",
				detail:  strconv.Quote(clean),
			})
			return ""
		}
	}

	absolute := filepath.Join(session.workspace, clean)
	file := setupFile{
		name:    clean,
		content: bytes.Clone(content),
	}
	session.files = append(session.files, file)
	return absolute
}

func (session *setupSession) recordFailure(err error) {
	if session.failure == nil {
		session.failure = err
	}
}

func validateSetupFileName(name string) (string, error) {
	if name == "" || strings.IndexByte(name, 0) >= 0 || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", &failureError{
			subject: "invalid setup file name",
			detail:  strconv.Quote(name),
		}
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", &failureError{
			subject: "invalid setup file name",
			detail:  strconv.Quote(name),
		}
	}
	return clean, nil
}

func setupFileNamesCollide(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if left == right {
		return true
	}
	separator := string(filepath.Separator)
	return strings.HasPrefix(left, right+separator) || strings.HasPrefix(right, left+separator)
}

func materializeSetup(workspace string, plan setupPlan) error {
	for _, file := range plan.files {
		if err := createWorkspaceFile(workspace, file.name, file.content, privateFileMode); err != nil {
			return &failureError{
				subject: "materialize setup file",
				detail:  strconv.Quote(file.name),
				cause:   err,
			}
		}
	}
	return nil
}
