//go:build !windows

package cli

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func processPlatformSignalDescriptors() []SignalDescriptor[os.Signal] {
	return []SignalDescriptor[os.Signal]{
		{
			Signal:       syscall.SIGTERM,
			Classes:      SignalClassTerminate,
			Reserved:     true,
			Capabilities: SignalSourceCapabilityOrigin | SignalSourceCapabilityReraise,
			Fidelity:     SignalSourceDeliveryFidelityCoalesced,
		},
		{
			Signal:       syscall.SIGHUP,
			Classes:      SignalClassTerminate,
			Capabilities: SignalSourceCapabilityOrigin | SignalSourceCapabilityReraise,
			Fidelity:     SignalSourceDeliveryFidelityCoalesced,
		},
		{
			Signal:       syscall.SIGQUIT,
			Classes:      SignalClassTerminate,
			Capabilities: SignalSourceCapabilityOrigin | SignalSourceCapabilityReraise,
			Fidelity:     SignalSourceDeliveryFidelityCoalesced,
		},
	}
}

var errUnsupportedProcessReraiseSignal = errors.New("cli: process reraise requires a native Unix signal")

func processSignalReraise(processSignal os.Signal) error {
	native, ok := processSignal.(syscall.Signal)
	if !ok || native <= 0 {
		return errUnsupportedProcessReraiseSignal
	}
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		failure := &processSignalError{
			operation: "find current process for signal reraise",
			cause:     err,
		}
		return failure
	}
	// Reset before signaling so the terminal reraise cannot be recaptured by
	// Commandment's own os/signal subscription.
	signal.Reset(native)
	if err := process.Signal(native); err != nil {
		failure := &processSignalError{
			operation: "reraise process signal",
			cause:     err,
		}
		return failure
	}

	// A successful process signal reports delivery, not process termination.
	// Reraise is terminal even when the host integration could otherwise return,
	// so do not let RunFromProcess race native signal termination with ordinary
	// unwinding after the signal has been accepted by the kernel.
	select {}
}
