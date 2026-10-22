//go:build windows

package shelltest

import (
	"context"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type terminalProcess struct {
	input         *os.File
	output        *os.File
	console       windows.Handle
	process       windows.Handle
	job           windows.Handle
	doneCh        chan struct{}
	waitErr       error
	terminateOnce sync.Once
	closeOnce     sync.Once
}

func startTerminalProcess(parent context.Context, executable string, args []string, workspace string, environment []string) (*terminalProcess, error) {
	var inputRead windows.Handle
	var inputWrite windows.Handle
	if err := windows.CreatePipe(&inputRead, &inputWrite, nil, 0); err != nil {
		failure := failureError{
			subject: "create ConPTY input pipe",
			cause:   err,
		}
		return nil, failure
	}
	defer func() {
		if inputRead != 0 {
			_ = windows.CloseHandle(inputRead)
		}
		if inputWrite != 0 {
			_ = windows.CloseHandle(inputWrite)
		}
	}()

	var outputRead windows.Handle
	var outputWrite windows.Handle
	if err := windows.CreatePipe(&outputRead, &outputWrite, nil, 0); err != nil {
		failure := failureError{
			subject: "create ConPTY output pipe",
			cause:   err,
		}
		return nil, failure
	}
	defer func() {
		if outputRead != 0 {
			_ = windows.CloseHandle(outputRead)
		}
		if outputWrite != 0 {
			_ = windows.CloseHandle(outputWrite)
		}
	}()

	var console windows.Handle
	size := windows.Coord{
		X: int16(terminalColumns),
		Y: int16(terminalRows),
	}
	if err := windows.CreatePseudoConsole(size, inputRead, outputWrite, 0, &console); err != nil {
		failure := failureError{
			subject: "create ConPTY",
			cause:   err,
		}
		return nil, failure
	}
	defer func() {
		if console != 0 {
			windows.ClosePseudoConsole(console)
		}
	}()

	if err := windows.CloseHandle(inputRead); err != nil {
		failure := failureError{
			subject: "release ConPTY input endpoint",
			cause:   err,
		}
		return nil, failure
	}
	inputRead = 0
	if err := windows.CloseHandle(outputWrite); err != nil {
		failure := failureError{
			subject: "release ConPTY output endpoint",
			cause:   err,
		}
		return nil, failure
	}
	outputWrite = 0

	attributes, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		failure := failureError{
			subject: "allocate ConPTY process attributes",
			cause:   err,
		}
		return nil, failure
	}
	defer attributes.Delete()
	if err := attributes.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, unsafe.Pointer(&console), unsafe.Sizeof(console)); err != nil {
		failure := failureError{
			subject: "attach ConPTY process attribute",
			cause:   err,
		}
		return nil, failure
	}

	job, err := createTerminalJob()
	if err != nil {
		return nil, err
	}
	defer func() {
		if job != 0 {
			_ = windows.CloseHandle(job)
		}
	}()

	executableUTF16, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		failure := failureError{
			subject: "encode shell executable",
			cause:   err,
		}
		return nil, failure
	}
	argv := make([]string, 0, len(args)+1)
	argv = append(argv, executable)
	argv = append(argv, args...)
	commandLineUTF16, err := windows.UTF16FromString(windows.ComposeCommandLine(argv))
	if err != nil {
		failure := failureError{
			subject: "encode shell command line",
			cause:   err,
		}
		return nil, failure
	}
	workspaceUTF16, err := windows.UTF16PtrFromString(workspace)
	if err != nil {
		failure := failureError{
			subject: "encode shell working directory",
			cause:   err,
		}
		return nil, failure
	}
	environmentUTF16, err := windowsEnvironmentBlock(environment)
	if err != nil {
		return nil, err
	}

	startup := windows.StartupInfoEx{
		StartupInfo: windows.StartupInfo{
			Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{})),
		},
		ProcThreadAttributeList: attributes.List(),
	}
	var processInfo windows.ProcessInformation
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_SUSPENDED | windows.CREATE_UNICODE_ENVIRONMENT)
	if err := windows.CreateProcess(executableUTF16, &commandLineUTF16[0], nil, nil, false, flags, &environmentUTF16[0], workspaceUTF16, &startup.StartupInfo, &processInfo); err != nil {
		failure := failureError{
			subject: "start ConPTY shell process",
			cause:   err,
		}
		return nil, failure
	}
	thread := processInfo.Thread
	defer func() {
		if thread != 0 {
			_ = windows.CloseHandle(thread)
		}
		if processInfo.Process != 0 {
			_ = windows.CloseHandle(processInfo.Process)
		}
	}()

	if err := windows.AssignProcessToJobObject(job, processInfo.Process); err != nil {
		failure := failureError{
			subject: "assign shell process tree to job",
			cause:   err,
		}
		return nil, failure
	}
	if _, err := windows.ResumeThread(thread); err != nil {
		failure := failureError{
			subject: "resume shell process",
			cause:   err,
		}
		return nil, failure
	}
	if err := windows.CloseHandle(thread); err != nil {
		failure := failureError{
			subject: "close shell primary thread handle",
			cause:   err,
		}
		return nil, failure
	}
	thread = 0

	input := os.NewFile(uintptr(inputWrite), "shelltest-conpty-input")
	output := os.NewFile(uintptr(outputRead), "shelltest-conpty-output")
	if input == nil || output == nil {
		failure := failureError{
			subject: "construct ConPTY stream handles",
		}
		return nil, failure
	}
	inputWrite = 0
	outputRead = 0

	process := &terminalProcess{
		input:   input,
		output:  output,
		console: console,
		process: processInfo.Process,
		job:     job,
		doneCh:  make(chan struct{}),
	}
	console = 0
	processInfo.Process = 0
	job = 0

	go process.wait(parent)
	return process, nil
}

func createTerminalJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		failure := failureError{
			subject: "create shell process job",
			cause:   err,
		}
		return 0, failure
	}
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		_ = windows.CloseHandle(job)
		failure := failureError{
			subject: "configure shell process job",
			cause:   err,
		}
		return 0, failure
	}
	return job, nil
}

func windowsEnvironmentBlock(environment []string) ([]uint16, error) {
	type environmentEntry struct {
		value string
		key   string
	}

	ordered := make([]environmentEntry, len(environment))
	for index, entry := range environment {
		ordered[index] = environmentEntry{
			value: entry,
			key:   strings.ToUpper(entry),
		}
	}
	slices.SortStableFunc(ordered, func(left, right environmentEntry) int {
		return strings.Compare(left.key, right.key)
	})

	var block []uint16
	for _, orderedEntry := range ordered {
		entry := orderedEntry.value
		if strings.IndexByte(entry, 0) >= 0 {
			failure := failureError{
				subject: "invalid shell environment entry",
			}
			return nil, failure
		}
		block = append(block, utf16.Encode([]rune(entry))...)
		block = append(block, 0)
	}
	block = append(block, 0)
	if len(block) == 1 {
		block = append(block, 0)
	}
	return block, nil
}

func (process *terminalProcess) wait(parent context.Context) {
	stopCancellation := context.AfterFunc(parent, process.terminate)
	defer stopCancellation()

	_, err := windows.WaitForSingleObject(process.process, windows.INFINITE)
	if err == nil {
		var exitCode uint32
		if codeErr := windows.GetExitCodeProcess(process.process, &exitCode); codeErr != nil {
			err = codeErr
		} else if exitCode != 0 {
			err = failureError{
				subject: "shell process exited with nonzero status",
				detail:  strconv.FormatUint(uint64(exitCode), 10),
			}
		}
	}
	process.waitErr = err
	close(process.doneCh)
}

func (process *terminalProcess) Read(data []byte) (int, error)  { return process.output.Read(data) }
func (process *terminalProcess) Write(data []byte) (int, error) { return process.input.Write(data) }
func (process *terminalProcess) done() <-chan struct{}          { return process.doneCh }

func (process *terminalProcess) exited() bool {
	select {
	case <-process.doneCh:
		return true
	default:
		return false
	}
}

func (process *terminalProcess) waitError() error {
	<-process.doneCh
	return process.waitErr
}

func (process *terminalProcess) terminate() {
	process.terminateOnce.Do(func() {
		if process.job != 0 {
			_ = windows.CloseHandle(process.job)
			process.job = 0
		}
		_ = process.input.Close()
	})
}

func (process *terminalProcess) closeTerminal() {
	process.closeOnce.Do(func() {
		if process.console != 0 {
			windows.ClosePseudoConsole(process.console)
			process.console = 0
		}
		_ = process.output.Close()
		if process.process != 0 {
			_ = windows.CloseHandle(process.process)
			process.process = 0
		}
	})
}

var _ io.ReadWriter = (*terminalProcess)(nil)
