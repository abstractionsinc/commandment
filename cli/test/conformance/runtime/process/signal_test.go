package conformance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type authoringSignalDelayConfig struct{ Delay time.Duration }

type authoringSignalEventsConfig struct{ Events <-chan cli.SignalEvent }

type authoringSignalGateConfig struct{ Gate string }

type authoringSignalSignalConfig struct{ Signal cli.Signal }

func authoringSignalPassiveSource() cli.SignalSource[string] {
	return cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   "sent",
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
		Open: func(_ context.Context) (cli.SignalStream[string], error) {
			return cli.SignalStream[string]{
				Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
					<-ctx.Done()
					return cli.SignalNotification[string]{}, ctx.Err()
				},
			}, nil
		},
	}
}

func declareAuthoringSignalFixture(root *cli.CommandBuilder[authoringSignalEventsConfig]) {
	root.Signals.Policy(
		root.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each().Hint("interrupt"),
		root.Signals.Action.Ignore(),
	)
	root.Signals.Policy(
		root.Signals.Match.Terminate().During(cli.SignalPhaseCleanup).Coalesce(),
		root.Signals.Action.Abort(17),
	)
	events := root.Signals.Endpoint(&root.Config.Events)
	root.Signals.Policy(
		root.Signals.Match.Shutdown().During(cli.SignalPhaseExecution).Latest(),
		root.Signals.Action.Emit(events),
	)
}

func TestAuthoringSignalSignalMatchPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorWhenPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorUnlessPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorExceptOriginPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorAtPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorAtMostPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorAtLeastPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorBetweenPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorContinueMatchingPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalPolicyMatchContinueMatchingPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorHintPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalPolicyMatchCoalescePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalPolicyMatchLatestPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalPolicyMatchDeferredPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalPolicyMatchHintPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalAggregationMatchDeferredPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalAggregationMatchContinueMatchingPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestAuthoringSignalSignalPlanPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalActionsSendPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalActionsSendFromPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalActionsReraisePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceThenCancelPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceThenHandlePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceThenForwardPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceThenSendPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceThenSendFromPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceThenEmitPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceThenReraisePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceAfterPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSequenceAfterFromPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalationCancelPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalationHandlePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalationForwardPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalationSendPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalationSendFromPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalationAbortPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalationReraisePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestNewSignalSelectorPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestCloneSignalMatchPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalMatchImplPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorConditionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorExceptPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorOriginsPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorSenderPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSelectorPhasesPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalOccurrencePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalAggregationPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalMatchModifierPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestCloneSignalPlanPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestAppendSignalActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEscalatePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSendPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSendFromPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalReraisePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSequenceAppendPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSequenceAfterPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestEscalationCompletePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestEraseSignalHandlerPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestEraseSignalTargetPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalTargetActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestStageSignalClearPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestStageSignalPolicyPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalPlanImplPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestDestinationForStoragePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestStoragePointerPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestStorageDeclaredAsDestinationPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestStorageDeclaredByRecorderPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestValidSignalListPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestValidSignalOriginsPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestValidSignalPhasesPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalCancelActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalReraiseActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalForwardActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSendActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalSendFromActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestDestinationForPlanStoragePreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestSignalEmitActionPreservesSignalAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = authoringSignalEventsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: declareAuthoringSignalFixture,
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Exists())
}

func TestWhenPublishesSignalCondition(t *testing.T) {
	t.Parallel()
	type config = authoringSignalGateConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			gate := root.Values.String(&root.Config.Gate)
			root.Signals.Policy(
				root.Signals.Match.Interrupt().When(root.When.Resolved(gate)).Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.Activation().Kind(cli.ConditionExpressionResolved),
		policy.Activation().Destination(clitest.RootCommand().ValueAt(0)),
	)
}

func TestUnlessPublishesNegatedSignalCondition(t *testing.T) {
	t.Parallel()
	type config = authoringSignalGateConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			gate := root.Values.String(&root.Config.Gate)
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Unless(root.When.Resolved(gate)).Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	activation := clitest.RootCommand().SignalPolicyAt(0).Activation()
	clitest.Schema(app).Require(t,
		activation.Kind(cli.ConditionExpressionNot),
		activation.ChildCount(1),
		activation.ChildAt(0).Kind(cli.ConditionExpressionResolved),
		activation.ChildAt(0).Destination(clitest.RootCommand().ValueAt(0)),
	)
}

func TestExceptOriginExcludesSignalOrigin(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().ExceptOrigin(cli.SignalOriginKernel).Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	clitest.Schema(app).
		Require(t, clitest.RootCommand().SignalPolicyAt(0).Selector().ExcludedOriginsExactly(cli.SignalOriginKernel))
}

func TestAtPublishesExactOccurrence(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().At(2).Each(), root.Signals.Action.Ignore())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Occurrence(cli.SignalOccurrenceRange{
		Minimum: 2,
		Maximum: 2,
	}))
}

func TestAtMostPublishesOccurrenceRange(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().AtMost(2).Each(), root.Signals.Action.Ignore())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Occurrence(cli.SignalOccurrenceRange{
		Minimum: 1,
		Maximum: 2,
	}))
}

func TestAtLeastPublishesOccurrenceRange(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().AtLeast(2).Each(), root.Signals.Action.Ignore())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Occurrence(cli.SignalOccurrenceRange{
		Minimum: 2,
		Maximum: 0,
	}))
}

func TestBetweenPublishesOccurrenceRange(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().Between(2, 4).Each(), root.Signals.Action.Ignore())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Occurrence(cli.SignalOccurrenceRange{
		Minimum: 2,
		Maximum: 4,
	}))
}

func TestContinueMatchingPublishesContinuation(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().ContinueMatching().Each(), root.Signals.Action.Ignore())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).ContinuesMatching())
}

func TestHintPublishesSignalHint(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Hint("interrupt hint").Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Hint("interrupt hint"))
}

func TestCoalescePublishesCoalescedAggregation(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().AtLeast(1).Coalesce(), root.Signals.Action.Ignore())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Aggregation(cli.SignalAggregationCoalesced))
}

func TestLatestPublishesLatestAggregation(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().AtLeast(1).Latest(), root.Signals.Action.Ignore())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Aggregation(cli.SignalAggregationLatest))
}

func TestDeferredPublishesDeferredPolicy(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().AtLeast(1).Deferred().Each(),
				root.Signals.Action.Ignore(),
			)
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).Deferred())
}

func TestSendPublishesSignalAction(t *testing.T) {
	t.Parallel()
	target := cli.SignalTarget[struct{}]{
		Name:    "target",
		Deliver: func(context.Context, *struct{}, cli.SignalDelivery) error { return nil },
	}
	var sent cli.Signal
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			sent = root.Signals.Source(authoringSignalPassiveSource()).Signal("sent")
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Send(target, sent))
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(1),
		policy.ActionAt(0).Kind(cli.SignalActionSend),
		policy.ActionAt(0).Name("target"),
		policy.ActionAt(0).Signal(sent),
	)
}

func TestSendFromPublishesDynamicSignalAction(t *testing.T) {
	t.Parallel()
	type config = authoringSignalSignalConfig
	target := cli.SignalTarget[config]{
		Name:    "target",
		Deliver: func(context.Context, *config, cli.SignalDelivery) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			signal := root.Values.Destination(&root.Config.Signal, cli.Value.Assign[cli.Signal]())
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.SendFrom(target, signal),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(1),
		policy.ActionAt(0).Kind(cli.SignalActionSend),
		policy.ActionAt(0).Name("target"),
		policy.ActionAt(0).SignalValue(clitest.RootCommand().ValueAt(0)),
	)
}

func TestReraisePublishesSignalAction(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Reraise())
		},
	})
	clitest.Schema(app).Require(t, clitest.RootCommand().SignalPolicyAt(0).ActionAt(0).Kind(cli.SignalActionReraise))
}

func TestThenHandleAppendsSignalAction(t *testing.T) {
	t.Parallel()
	handler := cli.SignalHandler[struct{}]{
		Name:   "handle",
		Handle: func(context.Context, *struct{}, cli.SignalEvent) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Cancel().ThenHandle(handler))
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionHandle),
		policy.ActionAt(1).Name("handle"),
	)
}

func TestThenForwardAppendsSignalAction(t *testing.T) {
	t.Parallel()
	target := cli.SignalTarget[struct{}]{
		Name:    "target",
		Deliver: func(context.Context, *struct{}, cli.SignalDelivery) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Cancel().ThenForward(target))
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionForward),
		policy.ActionAt(1).Name("target"),
	)
}

func TestThenSendAppendsSignalAction(t *testing.T) {
	t.Parallel()
	target := cli.SignalTarget[struct{}]{
		Name:    "target",
		Deliver: func(context.Context, *struct{}, cli.SignalDelivery) error { return nil },
	}
	var sent cli.Signal
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			sent = root.Signals.Source(authoringSignalPassiveSource()).Signal("sent")
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().ThenSend(target, sent),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionSend),
		policy.ActionAt(1).Name("target"),
		policy.ActionAt(1).Signal(sent),
	)
}

func TestThenSendFromAppendsDynamicSignalAction(t *testing.T) {
	t.Parallel()
	type config = authoringSignalSignalConfig
	target := cli.SignalTarget[config]{
		Name:    "target",
		Deliver: func(context.Context, *config, cli.SignalDelivery) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			signal := root.Values.Destination(&root.Config.Signal, cli.Value.Assign[cli.Signal]())
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().ThenSendFrom(target, signal),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionSend),
		policy.ActionAt(1).Name("target"),
		policy.ActionAt(1).SignalValue(clitest.RootCommand().ValueAt(0)),
	)
}

func TestThenReraiseAppendsSignalAction(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(root.Signals.Match.Interrupt().Each(), root.Signals.Action.Cancel().ThenReraise())
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionReraise),
	)
}

func TestAfterAppliesDelayToFollowingSignalAction(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().After(time.Second).Abort(2),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(0).Immediate(),
		policy.ActionAt(1).Kind(cli.SignalActionAbort),
		policy.ActionAt(1).Delayed(),
		policy.ActionAt(1).Delay(time.Second),
		policy.ActionAt(1).ExitCode(2),
	)
}

func TestAfterFromAppliesDynamicDelayToFollowingSignalAction(t *testing.T) {
	t.Parallel()
	type config = authoringSignalDelayConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			delay := root.Values.Destination(&root.Config.Delay, cli.Value.Assign[time.Duration]())
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().AfterFrom(delay).Abort(2),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionAbort),
		policy.ActionAt(1).Delayed(),
		policy.ActionAt(1).DelayValue(clitest.RootCommand().ValueAt(0)),
		policy.ActionAt(1).ExitCode(2),
	)
}

func TestCancelPublishesSignalAction(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().After(time.Second).Cancel(),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(0).Immediate(),
		policy.ActionAt(1).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Delayed(),
		policy.ActionAt(1).Delay(time.Second),
	)
}

func TestHandlePublishesSignalAction(t *testing.T) {
	t.Parallel()
	handler := cli.SignalHandler[struct{}]{
		Name:   "handle",
		Handle: func(context.Context, *struct{}, cli.SignalEvent) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().After(time.Second).Handle(handler),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionHandle),
		policy.ActionAt(1).Name("handle"),
		policy.ActionAt(1).Delayed(),
		policy.ActionAt(1).Delay(time.Second),
	)
}

func TestForwardPublishesSignalAction(t *testing.T) {
	t.Parallel()
	target := cli.SignalTarget[struct{}]{
		Name:    "target",
		Deliver: func(context.Context, *struct{}, cli.SignalDelivery) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().After(time.Second).Forward(target),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionForward),
		policy.ActionAt(1).Name("target"),
		policy.ActionAt(1).Delayed(),
		policy.ActionAt(1).Delay(time.Second),
	)
}

func TestAbortPublishesSignalAction(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Signals.Policy(
				root.Signals.Match.Interrupt().Each(),
				root.Signals.Action.Cancel().After(time.Second).Abort(7),
			)
		},
	})
	policy := clitest.RootCommand().SignalPolicyAt(0)
	clitest.Schema(app).Require(t,
		policy.ActionCount(2),
		policy.ActionAt(0).Kind(cli.SignalActionCancel),
		policy.ActionAt(1).Kind(cli.SignalActionAbort),
		policy.ActionAt(1).Delayed(),
		policy.ActionAt(1).Delay(time.Second),
		policy.ActionAt(1).ExitCode(7),
	)
}

type signalConcurrencyConfig struct{}

func TestProviderSignalPolicyInspectionDetachesAcrossOperations(t *testing.T) {
	t.Parallel()
	type config = signalConcurrencyConfig
	source := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   signalSignalNativeInterrupt,
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
		Open: func(context.Context) (cli.SignalStream[string], error) {
			panic("provider signal source opened during Inspect")
		},
	}
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			interrupt := root.Signals.Source(source).Signal(signalSignalNativeInterrupt)
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				batch.Command(cli.Command[config]{
					Name: "remote",
					Declare: func(child *cli.CommandBuilder[config]) {
						child.Signals.Policy(
							child.Signals.Match.On(interrupt).During(cli.SignalPhaseExecution).Each(),
							child.Signals.Action.Ignore(),
						)
					},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: []string{"remote"}})
		proof.Require(t, clitest.InspectionReady())
		return proof.Inspection()
	}()
	if err != nil || len(first.ExecutionAdmission.SignalPolicies) != 1 {
		t.Fatalf("first provider signal policy = %#v err=%v", first.ExecutionAdmission, err)
	}
	id := first.ExecutionAdmission.SignalPolicies[0]
	delete(first.SignalPolicies, id)
	first.ExecutionAdmission.SignalPolicies[0] = 0
	second, err := app.Inspect(t.Context(), cli.InspectionRequest{Args: []string{"remote"}})
	if err != nil || len(second.ExecutionAdmission.SignalPolicies) != 1 {
		t.Fatalf("second provider signal policy = %#v err=%v", second.ExecutionAdmission, err)
	}
	if _, ok := second.SignalPolicies[second.ExecutionAdmission.SignalPolicies[0]]; !ok {
		t.Fatalf("second provider signal policy is dangling: %#v", second.SignalPolicies)
	}
}

type declarativeSignalConfig struct{}

const (
	signalSignalNativeInterrupt        = "interrupt"
	signalSignalNativeMutated          = "mutated"
	signalSignalNativeDynamic          = "dynamic"
	signalSignalNativeTerminate        = "terminate"
	signalSignalNativeFirst            = "first"
	signalSignalAlienctlName           = "alienctl"
	signalSignalRunFromProcessChildEnv = "COMMANDMENT_SIGNAL_RUN_FROM_PROCESS_CHILD"
)

func exactStringSignal(name string, capabilities cli.SignalSourceCapabilities) cli.SignalDescriptor[string] {
	return cli.SignalDescriptor[string]{
		Signal:       name,
		Capabilities: capabilities,
		Fidelity:     cli.SignalSourceDeliveryFidelityExact,
	}
}

func TestSignalSourceScopedIdentityIsStableDistinctAndClosed(t *testing.T) {
	t.Parallel()
	passive := func(context.Context) (cli.SignalStream[string], error) {
		return cli.SignalStream[string]{
			Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
				<-ctx.Done()
				return cli.SignalNotification[string]{}, ctx.Err()
			},
		}, nil
	}
	var firstA, firstAgain, secondA cli.Signal
	app := clitest.Compile(t, cli.Definition[declarativeSignalConfig]{
		Name: "identity",
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			first := root.Signals.Source(cli.SignalSource[string]{
				Signals: []cli.SignalDescriptor[string]{exactStringSignal("a", 0)},
				Open:    passive,
			})
			second := root.Signals.Source(cli.SignalSource[string]{
				Signals: []cli.SignalDescriptor[string]{exactStringSignal("a", 0)},
				Open:    passive,
			})
			firstA = first.Signal("a")
			firstAgain = first.Signal("a")
			secondA = second.Signal("a")
			root.Signals.Policy(root.Signals.Match.On(firstA).Each(), root.Signals.Action.Ignore())
		},
	})
	_ = app
	if firstA != firstAgain {
		t.Fatal("repeated lookup in one source registration changed semantic identity")
	}
	if firstA == secondA {
		t.Fatal("equal native values in distinct source registrations shared semantic identity")
	}

	_, err := cli.Compile(cli.Definition[declarativeSignalConfig]{
		Name: "unknown",
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			source := root.Signals.Source(cli.SignalSource[string]{
				Signals: []cli.SignalDescriptor[string]{exactStringSignal("known", 0)},
				Open:    passive,
			})
			_ = source.Signal("unknown")
		},
	})
	if err == nil {
		t.Fatal("Compile accepted lookup of an undeclared source-domain signal")
	}
	clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestSignalSourceDescriptorsAreSnapshottedAndInspectionIsStatic(t *testing.T) {
	t.Parallel()
	var opens atomic.Int32
	descriptors := []cli.SignalDescriptor[string]{exactStringSignal(signalSignalNativeInterrupt, 0)}
	var semantic cli.Signal
	app := clitest.Compile(t, cli.Definition[declarativeSignalConfig]{
		Name: "snapshot",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			registered := root.Signals.Source(cli.SignalSource[string]{
				Signals: descriptors,
				Open: func(context.Context) (cli.SignalStream[string], error) {
					opens.Add(1)
					return cli.SignalStream[string]{}, fmt.Errorf(
						"must not open during inspect: %w",
						errors.ErrUnsupported,
					)
				},
			})
			semantic = registered.Signal(signalSignalNativeInterrupt)
			descriptors[0].Signal = signalSignalNativeMutated
			root.Signals.Policy(root.Signals.Match.On(semantic).Each(), root.Signals.Action.Ignore())
		},
	})
	inspection, err := app.Inspect(t.Context(), cli.InspectionRequest{})
	if err != nil || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf("Inspect() = status %v err %v", inspection.Status, err)
	}
	if opens.Load() != 0 {
		t.Fatalf("Inspect opened static signal source %d times", opens.Load())
	}
	if len(inspection.ExecutionAdmission.SignalPolicies) != 1 ||
		!slices.Contains(inspection.SignalSource.ResolvedSignals, semantic) {
		t.Fatalf("static signal projection lost snapshotted semantic identity: %#v", inspection.SignalSource)
	}
}

func TestSignalSourceReraiseDeclarationValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source cli.SignalSource[string]
	}{
		{
			name: "advertised without callback",
			source: cli.SignalSource[string]{
				Signals: []cli.SignalDescriptor[string]{exactStringSignal("a", cli.SignalSourceCapabilityReraise)},
				Open: func(context.Context) (cli.SignalStream[string], error) {
					return cli.SignalStream[string]{
						Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
							<-ctx.Done()
							return cli.SignalNotification[string]{}, ctx.Err()
						},
					}, nil
				},
			},
		},
		{
			name: "callback without advertisement",
			source: cli.SignalSource[string]{
				Signals: []cli.SignalDescriptor[string]{exactStringSignal("a", 0)},
				Open: func(context.Context) (cli.SignalStream[string], error) {
					return cli.SignalStream[string]{
						Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
							<-ctx.Done()
							return cli.SignalNotification[string]{}, ctx.Err()
						},
					}, nil
				},
				Reraise: func(context.Context, string) error { return nil },
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := cli.Compile(cli.Definition[declarativeSignalConfig]{
				Name: "invalid-source",
				Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
					root.Signals.Source(test.source)
				},
			})
			if err == nil {
				t.Fatal("Compile accepted invalid Reraise source declaration")
			}
			clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
		})
	}
}

func TestSignalReraisePolicyRejectsMixedCapabilitiesAtCompile(t *testing.T) {
	t.Parallel()
	source := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{
			exactStringSignal("reraisable", cli.SignalSourceCapabilityReraise),
			exactStringSignal("plain", 0),
		},
		Open: func(context.Context) (cli.SignalStream[string], error) {
			return cli.SignalStream[string]{Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
				<-ctx.Done()
				return cli.SignalNotification[string]{}, ctx.Err()
			}}, nil
		},
		Reraise: func(context.Context, string) error { return nil },
	}
	_, err := cli.Compile(cli.Definition[declarativeSignalConfig]{
		Name: "mixed-reraise",
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			registered := root.Signals.Source(source)
			root.Signals.Policy(
				root.Signals.Match.On(registered.Signal("reraisable"), registered.Signal("plain")).Each(),
				root.Signals.Action.Reraise(),
			)
		},
	})
	if err == nil {
		t.Fatal("Compile accepted Reraise selector containing a non-reraisable signal")
	}
	clitest.Diagnostics(err).Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestProviderReraisePolicyRejectsAtomicBatchBeforeActivation(t *testing.T) {
	t.Parallel()
	source := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{
			exactStringSignal("reraisable", cli.SignalSourceCapabilityReraise),
			exactStringSignal("plain", 0),
		},
		Open: func(context.Context) (cli.SignalStream[string], error) {
			return cli.SignalStream[string]{Receive: func(ctx context.Context) (cli.SignalNotification[string], error) {
				<-ctx.Done()
				return cli.SignalNotification[string]{}, ctx.Err()
			}}, nil
		},
		Reraise: func(context.Context, string) error { return nil },
	}
	app := clitest.Compile(t, cli.Definition[declarativeSignalConfig]{
		Name: "provider-reraise",
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			registered := root.Signals.Source(source)
			plain := registered.Signal("plain")
			root.Subcommands.CommandsFrom(
				func(_ context.Context, batch *cli.CommandBatch[declarativeSignalConfig]) error {
					batch.Command(cli.Command[declarativeSignalConfig]{
						Name: signalSignalNativeDynamic,
						Declare: func(command *cli.CommandBuilder[declarativeSignalConfig]) {
							command.Signals.Policy(
								command.Signals.Match.On(plain).Each(),
								command.Signals.Action.Reraise(),
							)
						},
					})
					return nil
				},
			)
		},
	})
	inspection, err := app.Inspect(t.Context(), cli.InspectionRequest{})
	if err == nil || inspection.Status != cli.InspectionStatusRejected {
		t.Fatalf("Inspect() = status %v err %v, want rejected provider batch", inspection.Status, err)
	}
	if len(inspection.Selection.Command.Path) != 1 {
		t.Fatalf("rejected provider batch changed active command path: %#v", inspection.Selection.Command)
	}
}

func TestOrdinarySignalSourceRunsAndReraisesNativeDomainValue(t *testing.T) {
	t.Parallel()
	source, control := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{exactStringSignal("reload", cli.SignalSourceCapabilityReraise)},
	})
	var semantic cli.Signal
	app := clitest.Compile(t, cli.Definition[declarativeSignalConfig]{
		Name: "ordinary-reraise",
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			registered := root.Signals.Source(source)
			semantic = registered.Signal("reload")
			root.Signals.Policy(
				root.Signals.Match.On(semantic).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Reraise(),
			)
		},
		Run: func(ctx context.Context, _ *declarativeSignalConfig) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})
	done := make(chan error, 1)
	go func() { done <- app.Run(t.Context(), cli.RunRequest{}) }()
	select {
	case <-control.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("ordinary signal source did not open")
	}
	if err := control.Notify(
		cli.SignalNotification[string]{
			Signal: "reload",
			Origin: cli.SignalOriginRuntime,
		},
	); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-control.Reraises():
		if got != "reload" {
			t.Fatalf("Reraise native value = %q, want reload", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("source-native Reraise was not observed")
	}
	err := <-done
	var termination *cli.SignalTermination
	if !errors.As(err, &termination) || termination.Signal != semantic {
		t.Fatalf("Run() error = %#v, want semantic SignalTermination", err)
	}
	select {
	case <-control.Closed():
	case <-time.After(2 * time.Second):
		t.Fatal("ordinary signal source did not close")
	}
}

func TestAdvancedSignalSourceUsesTypedWatchAndSameReraiseLaw(t *testing.T) {
	t.Parallel()
	source, control := clitest.SignalWatchSource(clitest.SignalWatchSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{
			exactStringSignal(signalSignalNativeTerminate, cli.SignalSourceCapabilityReraise),
		},
	})
	var semantic cli.Signal
	app := clitest.Compile(t, cli.Definition[declarativeSignalConfig]{
		Name: "advanced-reraise",
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			registered := root.Signals.Source(source)
			semantic = registered.Signal(signalSignalNativeTerminate)
			root.Signals.Policy(
				root.Signals.Match.On(semantic).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Reraise(),
			)
		},
		Run: func(ctx context.Context, _ *declarativeSignalConfig) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})
	done := make(chan error, 1)
	go func() { done <- app.Run(t.Context(), cli.RunRequest{}) }()
	var watch clitest.SignalWatchState[string]
	select {
	case watch = <-control.WatchStates():
	case <-time.After(2 * time.Second):
		t.Fatal("advanced signal source did not subscribe")
	}
	if watch.Revision == 0 || !slices.Equal(watch.Signals, []string{signalSignalNativeTerminate}) {
		t.Fatalf("initial watch = %#v", watch)
	}
	if err := control.Notify(
		cli.SignalNotification[string]{
			Signal: signalSignalNativeTerminate,
			Origin: cli.SignalOriginRuntime,
		},
	); err != nil {
		t.Fatal(err)
	}
	select {
	case got, ok := <-control.Reraises():
		if !ok {
			t.Fatal("Reraise history closed without value")
		}
		if got != signalSignalNativeTerminate {
			t.Fatalf("Reraise native value = %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("advanced source-native Reraise was not observed")
	}
	err := <-done
	var termination *cli.SignalTermination
	if !errors.As(err, &termination) || termination.Signal != semantic {
		t.Fatalf("Run() error = %#v, want semantic SignalTermination", err)
	}
	select {
	case <-control.Closed():
	case <-time.After(2 * time.Second):
		t.Fatal("advanced signal source did not close")
	}
}

func TestMultipleDeclaredSignalSourcesShareOneInvocationRuntime(t *testing.T) {
	t.Parallel()
	firstSource, firstControl := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{exactStringSignal(signalSignalNativeFirst, 0)},
	})
	secondSource, secondControl := clitest.SignalSource(clitest.SignalSourceOptions[int]{
		Signals: []cli.SignalDescriptor[int]{{Signal: 2, Fidelity: cli.SignalSourceDeliveryFidelityExact}},
	})
	seen := make(chan cli.Signal, 2)
	app := clitest.Compile(t, cli.Definition[declarativeSignalConfig]{
		Name: "multi-source",
		Declare: func(root *cli.CommandBuilder[declarativeSignalConfig]) {
			first := root.Signals.Source(firstSource).Signal(signalSignalNativeFirst)
			second := root.Signals.Source(secondSource).Signal(2)
			handler := cli.SignalHandler[declarativeSignalConfig]{
				Name: "record",
				Handle: func(_ context.Context, _ *declarativeSignalConfig, event cli.SignalEvent) error {
					seen <- event.Signal
					return nil
				},
			}
			root.Signals.Policy(
				root.Signals.Match.On(first).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Handle(handler),
			)
			root.Signals.Policy(
				root.Signals.Match.On(second).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Handle(handler),
			)
		},
		Run: func(ctx context.Context, _ *declarativeSignalConfig) error {
			<-ctx.Done()
			return nil
		},
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx, cli.RunRequest{}) }()
	for _, opened := range []<-chan struct{}{firstControl.Opened(), secondControl.Opened()} {
		select {
		case <-opened:
		case <-time.After(2 * time.Second):
			t.Fatal("declared signal source was not activated")
		}
	}
	if err := firstControl.Notify(cli.SignalNotification[string]{Signal: signalSignalNativeFirst}); err != nil {
		t.Fatal(err)
	}
	if err := secondControl.Notify(cli.SignalNotification[int]{Signal: 2}); err != nil {
		t.Fatal(err)
	}
	observed := make([]cli.Signal, 0, 2)
	deadline := time.After(2 * time.Second)
	for len(observed) < 2 {
		select {
		case signal := <-seen:
			observed = append(observed, signal)
		case runErr := <-done:
			chain := []string{}
			for current := runErr; current != nil; current = errors.Unwrap(current) {
				chain = append(chain, fmt.Sprintf("%T: %s", current, current.Error()))
			}
			t.Fatalf("Run exited before both source notifications were dispatched: %#v", chain)
		case <-deadline:
			t.Fatalf("canonical dispatcher did not observe both source domains: %#v", observed)
		}
	}
	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() after caller cancellation = %v", err)
	}
}

func TestAlienctlDeclarativeSignalSourceEmitRunsFromProcessWithoutBinding(t *testing.T) {
	t.Parallel()
	if os.Getenv(signalSignalRunFromProcessChildEnv) == "1" {
		runAlienctlDeclarativeSignalSourceEmitFromProcess(t)
		return
	}

	cmd := exec.CommandContext(
		t.Context(),
		os.Args[0],
		"-test.run=^TestAlienctlDeclarativeSignalSourceEmitRunsFromProcessWithoutBinding$",
	)
	cmd.Env = append(os.Environ(), signalSignalRunFromProcessChildEnv+"=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("RunFromProcess declarative source child failed: %v\n%s", err, output)
	}
}

func runAlienctlDeclarativeSignalSourceEmitFromProcess(t *testing.T) {
	t.Helper()
	type config struct {
		Events <-chan cli.SignalEvent
	}

	source, control := clitest.SignalSource(clitest.SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{exactStringSignal("alarm", 0)},
	})
	var alarm cli.Signal
	app := clitest.Compile(t, cli.Definition[config]{
		Name: signalSignalAlienctlName,
		Declare: func(root *cli.CommandBuilder[config]) {
			alien := root.Signals.Source(source)
			alarm = alien.Signal("alarm")
			events := root.Signals.Endpoint(&root.Config.Events)
			root.Signals.Policy(
				root.Signals.Match.On(alarm).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Emit(events),
			)
			// The process integration is declared by Commandment itself and enters the
			// same canonical runtime when RunFromProcess is selected.
			root.Signals.Policy(
				root.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Ignore(),
			)
		},
		Run: func(ctx context.Context, cfg *config) error {
			select {
			case event := <-cfg.Events:
				if event.Signal != alarm {
					return fmt.Errorf("alienctl received the wrong semantic signal: %w", errors.ErrUnsupported)
				}
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})

	oldArgs := os.Args
	os.Args = []string{signalSignalAlienctlName}
	defer func() { os.Args = oldArgs }()

	done := make(chan error, 1)
	go func() { done <- app.RunFromProcess(t.Context()) }()
	select {
	case <-control.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("Definition-owned source was not activated by RunFromProcess")
	}
	if err := control.Notify(
		cli.SignalNotification[string]{
			Signal: "alarm",
			Origin: cli.SignalOriginRuntime,
		},
	); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunFromProcess() = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunFromProcess did not deliver the declarative Signal endpoint")
	}
	select {
	case <-control.Closed():
	case <-time.After(2 * time.Second):
		t.Fatal("Definition-owned source was not closed after RunFromProcess")
	}
}

type signalConfig struct{}

func TestSignalPolicyPreparationIgnoresUnrelatedCommandPolicies(t *testing.T) {
	t.Parallel()
	source := cli.SignalSource[string]{
		Signals: []cli.SignalDescriptor[string]{{
			Signal:   "signal",
			Fidelity: cli.SignalSourceDeliveryFidelityExact,
		}},
		Open: func(context.Context) (cli.SignalStream[string], error) {
			panic("signal source opened during Inspect")
		},
	}
	app := clitest.Compile(t, cli.Definition[signalConfig]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[signalConfig]) {
			sig := root.Signals.Source(source).Signal("signal")
			root.Signals.Policy(
				root.Signals.Match.On(sig).During(cli.SignalPhaseExecution).Each(),
				root.Signals.Action.Ignore(),
			)
			root.Subcommands.Command(cli.Command[signalConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[signalConfig]) {
					target.Signals.ClearInherited()
					target.Signals.Policy(
						target.Signals.Match.On(sig).During(cli.SignalPhaseExecution).Each(),
						target.Signals.Action.Ignore(),
					)
				},
			})
			root.Subcommands.Command(cli.Command[signalConfig]{
				Name: "bulk",
				Declare: func(bulk *cli.CommandBuilder[signalConfig]) {
					for range 64 {
						bulk.Signals.Policy(
							bulk.Signals.Match.On(sig).During(cli.SignalPhaseExecution).Each(),
							bulk.Signals.Action.Ignore(),
						)
					}
				},
			})
		},
	})
	target := clitest.RootCommand().Command("target")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: []string{"target"}}).Require(
		t,
		clitest.InspectionReady(),
		clitest.SelectionPathExactly(clitest.RootCommand(), target),
		clitest.ExecutionAdmission().SignalPoliciesExactly(target.SignalPolicyAt(0)),
	)
}

func TestDeferredSignalWindowsRemainSeparateAcrossExecutionAndCleanup(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		sourceOptions := clitest.SignalSourceOptions[string]{
			Signals: []cli.SignalDescriptor[string]{exactStringSignal("reload", 0)},
		}
		source, control := clitest.SignalSource(sourceOptions)
		executionEntered := make(chan struct{})
		allowHandlerReturn := make(chan struct{})
		cleanupEntered := make(chan struct{})
		allowRelease := make(chan struct{})
		acknowledged := make(chan cli.SignalEvent, 2)
		deferred := make(chan cli.SignalEvent, 2)
		middleware := func(ctx context.Context, _ *struct{}, next cli.MiddlewareNext) error {
			release := cli.DeferSignalActions(ctx)
			err := next(ctx)
			close(cleanupEntered)
			<-allowRelease
			release()
			return err
		}
		acknowledge := func(_ context.Context, _ *struct{}, event cli.SignalEvent) error {
			acknowledged <- event
			return nil
		}
		recordDeferred := func(_ context.Context, _ *struct{}, event cli.SignalEvent) error {
			deferred <- event
			return nil
		}
		declare := func(root *cli.CommandBuilder[struct{}]) {
			registered := root.Signals.Source(source)
			signal := registered.Signal("reload")
			ack := cli.SignalHandler[struct{}]{
				Name:   "acknowledge",
				Handle: acknowledge,
			}
			record := cli.SignalHandler[struct{}]{
				Name:   "record-deferred",
				Handle: recordDeferred,
			}
			root.Signals.Policy(
				root.Signals.Match.On(signal).
					During(cli.SignalPhaseExecution, cli.SignalPhaseCleanup).
					Each().
					ContinueMatching(),
				root.Signals.Action.Handle(ack),
			)
			root.Signals.Policy(
				root.Signals.Match.On(signal).
					During(cli.SignalPhaseExecution, cli.SignalPhaseCleanup).
					Deferred().
					Each(),
				root.Signals.Action.Handle(record),
			)
		}
		run := func(context.Context, *struct{}) error {
			close(executionEntered)
			<-allowHandlerReturn
			return nil
		}
		definition := cli.Definition[struct{}]{
			Name:       "alienctl",
			Middleware: []cli.Middleware[struct{}]{middleware},
			Declare:    declare,
			Run:        run,
		}
		app := clitest.Compile(t, definition)
		request := cli.RunRequest{}
		done := make(chan error, 1)

		go func() {
			done <- app.Run(t.Context(), request)
		}()
		<-control.Opened()
		<-executionEntered
		if err := control.Notify(cli.SignalNotification[string]{Signal: "reload"}); err != nil {
			t.Fatalf("Execution Notify() error = %v", err)
		}
		firstAck := <-acknowledged
		if firstAck.Phase != cli.SignalPhaseExecution {
			t.Fatalf("first acknowledged phase = %v, want Execution", firstAck.Phase)
		}
		close(allowHandlerReturn)
		<-cleanupEntered
		if err := control.Notify(cli.SignalNotification[string]{Signal: "reload"}); err != nil {
			t.Fatalf("Cleanup Notify() error = %v", err)
		}
		secondAck := <-acknowledged
		if secondAck.Phase != cli.SignalPhaseCleanup {
			t.Fatalf("second acknowledged phase = %v, want Cleanup", secondAck.Phase)
		}
		close(allowRelease)

		if err := <-done; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		first := <-deferred
		second := <-deferred
		if first.Phase != cli.SignalPhaseExecution || first.BatchSize != 1 {
			t.Fatalf("first deferred event = %#v, want Execution singleton", first)
		}
		if second.Phase != cli.SignalPhaseCleanup || second.BatchSize != 1 {
			t.Fatalf("second deferred event = %#v, want Cleanup singleton", second)
		}
		synctest.Wait()
		select {
		case extra := <-deferred:
			t.Fatalf("unexpected third deferred event = %#v", extra)
		default:
		}
	})
}

func TestUnreleasedDeferredSignalWindowIsDiscardedAtShutdown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		sourceOptions := clitest.SignalSourceOptions[string]{
			Signals: []cli.SignalDescriptor[string]{exactStringSignal("reload", 0)},
		}
		source, control := clitest.SignalSource(sourceOptions)
		executionEntered := make(chan struct{})
		allowHandlerReturn := make(chan struct{})
		acknowledged := make(chan cli.SignalEvent, 1)
		deferred := make(chan cli.SignalEvent, 1)
		middleware := func(ctx context.Context, _ *struct{}, next cli.MiddlewareNext) error {
			_ = cli.DeferSignalActions(ctx)
			return next(ctx)
		}
		acknowledge := func(_ context.Context, _ *struct{}, event cli.SignalEvent) error {
			acknowledged <- event
			return nil
		}
		recordDeferred := func(_ context.Context, _ *struct{}, event cli.SignalEvent) error {
			deferred <- event
			return nil
		}
		declare := func(root *cli.CommandBuilder[struct{}]) {
			signal := root.Signals.Source(source).Signal("reload")
			ack := cli.SignalHandler[struct{}]{Name: "acknowledge", Handle: acknowledge}
			record := cli.SignalHandler[struct{}]{Name: "record-deferred", Handle: recordDeferred}
			root.Signals.Policy(
				root.Signals.Match.On(signal).
					During(cli.SignalPhaseExecution).
					Each().
					ContinueMatching(),
				root.Signals.Action.Handle(ack),
			)
			root.Signals.Policy(
				root.Signals.Match.On(signal).
					During(cli.SignalPhaseExecution).
					Deferred().
					Each(),
				root.Signals.Action.Handle(record),
			)
		}
		run := func(context.Context, *struct{}) error {
			close(executionEntered)
			<-allowHandlerReturn
			return nil
		}
		definition := cli.Definition[struct{}]{
			Name:       "alienctl",
			Middleware: []cli.Middleware[struct{}]{middleware},
			Declare:    declare,
			Run:        run,
		}
		app := clitest.Compile(t, definition)
		request := cli.RunRequest{}
		done := make(chan error, 1)

		go func() {
			done <- app.Run(t.Context(), request)
		}()
		<-control.Opened()
		<-executionEntered
		if err := control.Notify(cli.SignalNotification[string]{Signal: "reload"}); err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
		ack := <-acknowledged
		if ack.Phase != cli.SignalPhaseExecution {
			t.Fatalf("acknowledged phase = %v, want Execution", ack.Phase)
		}
		close(allowHandlerReturn)
		if err := <-done; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		synctest.Wait()
		select {
		case event := <-deferred:
			t.Fatalf("unreleased deferred event dispatched = %#v", event)
		default:
		}
	})
}

func TestUnadmittedDelayedSignalActionIsDiscardedAtShutdown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		sourceOptions := clitest.SignalSourceOptions[string]{
			Signals: []cli.SignalDescriptor[string]{exactStringSignal("reload", 0)},
		}
		source, control := clitest.SignalSource(sourceOptions)
		executionEntered := make(chan struct{})
		allowHandlerReturn := make(chan struct{})
		acknowledged := make(chan cli.SignalEvent, 1)
		var delayedCalls atomic.Int32
		acknowledge := func(_ context.Context, _ *struct{}, event cli.SignalEvent) error {
			acknowledged <- event
			return nil
		}
		immediate := func(context.Context, *struct{}, cli.SignalEvent) error {
			return nil
		}
		delayed := func(context.Context, *struct{}, cli.SignalEvent) error {
			delayedCalls.Add(1)
			return nil
		}
		declare := func(root *cli.CommandBuilder[struct{}]) {
			signal := root.Signals.Source(source).Signal("reload")
			ack := cli.SignalHandler[struct{}]{Name: "acknowledge", Handle: acknowledge}
			immediateHandler := cli.SignalHandler[struct{}]{Name: "immediate", Handle: immediate}
			delayedHandler := cli.SignalHandler[struct{}]{Name: "delayed", Handle: delayed}
			root.Signals.Policy(
				root.Signals.Match.On(signal).
					During(cli.SignalPhaseExecution).
					Each().
					ContinueMatching(),
				root.Signals.Action.Handle(ack),
			)
			root.Signals.Policy(
				root.Signals.Match.On(signal).
					During(cli.SignalPhaseExecution).
					Each(),
				root.Signals.Action.Handle(immediateHandler).
					After(time.Hour).
					Handle(delayedHandler),
			)
		}
		run := func(context.Context, *struct{}) error {
			close(executionEntered)
			<-allowHandlerReturn
			return nil
		}
		definition := cli.Definition[struct{}]{
			Name:    "alienctl",
			Declare: declare,
			Run:     run,
		}
		app := clitest.Compile(t, definition)
		request := cli.RunRequest{}
		done := make(chan error, 1)

		go func() {
			done <- app.Run(t.Context(), request)
		}()
		<-control.Opened()
		<-executionEntered
		if err := control.Notify(cli.SignalNotification[string]{Signal: "reload"}); err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
		<-acknowledged
		close(allowHandlerReturn)
		if err := <-done; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		synctest.Wait()
		if got := delayedCalls.Load(); got != 0 {
			t.Fatalf("delayed calls after shutdown = %d, want 0", got)
		}
	})
}

func TestResolutionCancelStopsPostRuleAdvisoryAnalysis(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		sourceOptions := clitest.SignalSourceOptions[string]{
			Signals: []cli.SignalDescriptor[string]{exactStringSignal("interrupt", 0)},
		}
		source, control := clitest.SignalSource(sourceOptions)
		ruleEntered := make(chan struct{})
		var advisoryCalls atomic.Int32
		var handlerCalls atomic.Int32
		checkRule := func(ctx context.Context, _ *struct{}, _ cli.Dependencies) (bool, error) {
			close(ruleEntered)
			<-ctx.Done()
			return true, nil
		}
		detectAdvisory := func(context.Context, *struct{}, cli.Dependencies) (bool, error) {
			advisoryCalls.Add(1)
			return true, nil
		}
		advisory := cli.Advisory{
			Code:     "post-cancel",
			Severity: cli.NoticeSeverityInformation,
			Message:  "must remain cold",
		}
		declare := func(root *cli.CommandBuilder[struct{}]) {
			registered := root.Signals.Source(source)
			root.Signals.Policy(
				root.Signals.Match.On(registered.Signal("interrupt")).
					During(cli.SignalPhaseResolution).
					Each(),
				root.Signals.Action.Cancel(),
			)
			root.Rule.Check("wait-for-cancel", checkRule)
			root.Advisory.Detect(detectAdvisory, advisory)
		}
		run := func(context.Context, *struct{}) error {
			handlerCalls.Add(1)
			return nil
		}
		definition := cli.Definition[struct{}]{
			Name:    "alienctl",
			Declare: declare,
			Run:     run,
		}
		app := clitest.Compile(t, definition)
		request := cli.RunRequest{}
		done := make(chan error, 1)

		go func() {
			done <- app.Run(t.Context(), request)
		}()
		<-control.Opened()
		<-ruleEntered
		if err := control.Notify(cli.SignalNotification[string]{Signal: "interrupt"}); err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
		err := <-done
		if _, ok := errors.AsType[*cli.SignalCancellation](err); !ok {
			t.Fatalf("Run() error = %v, want SignalCancellation", err)
		}
		if got := advisoryCalls.Load(); got != 0 {
			t.Fatalf("Advisory calls after Resolution cancellation = %d, want 0", got)
		}
		if got := handlerCalls.Load(); got != 0 {
			t.Fatalf("Run handler calls after Resolution cancellation = %d, want 0", got)
		}
	})
}
