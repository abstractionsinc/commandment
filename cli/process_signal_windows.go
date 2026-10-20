//go:build windows

package cli

import (
	"errors"
	"os"
	"syscall"
)

const windowsStatusControlCExit uint32 = 0xC000013A

func processPlatformSignalDescriptors() []SignalDescriptor[os.Signal] { return nil }

var errUnsupportedProcessReraiseSignal = errors.New("cli: process reraise on Windows supports only os.Interrupt")

func processSignalReraise(signal os.Signal) error {
	if signal != os.Interrupt {
		return errUnsupportedProcessReraiseSignal
	}
	syscall.ExitProcess(windowsStatusControlCExit)
	return nil
}
