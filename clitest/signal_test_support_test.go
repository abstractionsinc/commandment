package clitest

import (
	"context"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func testSemanticSignals(t *testing.T, names ...string) []cli.Signal {
	t.Helper()
	got := make([]cli.Signal, len(names))
	descriptors := make([]cli.SignalDescriptor[string], len(names))
	for index, name := range names {
		descriptors[index] = cli.SignalDescriptor[string]{
			Signal:   name,
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}
	}
	receive := func(ctx context.Context) (cli.SignalNotification[string], error) {
		<-ctx.Done()
		return cli.SignalNotification[string]{}, ctx.Err()
	}
	open := func(context.Context) (cli.SignalStream[string], error) {
		stream := cli.SignalStream[string]{Receive: receive}
		return stream, nil
	}
	signalSource := cli.SignalSource[string]{
		Signals: descriptors,
		Open:    open,
	}
	declare := func(root *cli.CommandBuilder[struct{}]) {
		source := root.Signals.Source(signalSource)
		for index, name := range names {
			got[index] = source.Signal(name)
		}
	}
	definition := cli.Definition[struct{}]{
		Name:    "signal-test",
		Declare: declare,
	}
	_, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("build semantic signal test identities: %v", err)
	}
	return got
}

func testSemanticSignal(t *testing.T) cli.Signal {
	t.Helper()
	return testSemanticSignals(t, "interrupt")[0]
}
