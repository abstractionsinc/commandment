package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestCommandEntrySelectionStepLocatorPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()

		if got := commandEntrySelectionStepLocator(CommandEntryLocator{}); got.state != nil {
			t.Fatalf("zero command-entry selection step = %#v, want zero locator", got.state)
		}
		entry := CommandByID(1).Alias("legacy")
		got := commandEntrySelectionStepLocator(entry)
		if got.state == nil || got.state.kind != selectionStepLocatorEntry || got.state.entry == nil ||
			got.state.entry.mode != commandEntryLocatorAlias || got.state.entry.name != "legacy" {
			t.Fatalf("typed command-entry selection step = %#v", got.state)
		}
	})
}

func TestCommandDefaultSelectionStepsPreservesSelectionState(t *testing.T) {
	t.Parallel()

	step := commandDefaultSelectionStepLocator(CommandByID(1).DefaultAt(0))
	if step.state == nil || step.state.kind != selectionStepLocatorDefault || step.state.commandDefault == nil ||
		step.state.commandDefault.mode != commandDefaultLocatorIndex {
		t.Fatalf("command-default selection step = %#v", step.state)
	}
}

func TestSelectionStepFromPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()

		if got := selectionStepFrom(SelectionStepLocator{}, InputOriginByID(cli.InputOriginID{})); got.state != nil {
			t.Fatalf("zero selection step = %#v, want zero locator", got.state)
		}
	})
	t.Run("invalid selection-step origin locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		got := selectionStepFrom(commandEntrySelectionStepLocator(CommandEntryByID(1)), InputOriginLocator{})
		if got.state != nil {
			t.Fatalf("selection step with invalid origin = %#v, want zero locator", got.state)
		}
	})
}

func TestSelectionStepSelectsPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()

		commands := []CommandLocator{CommandByID(1)}
		if got := selectionStepSelects(SelectionStepLocator{}, commands); got.state != nil {
			t.Fatalf("zero selection step = %#v, want zero locator", got.state)
		}
	})
	t.Run("invalid selection-step command locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		commands := []CommandLocator{{}}
		got := selectionStepSelects(commandEntrySelectionStepLocator(CommandEntryByID(1)), commands)
		if got.state != nil {
			t.Fatalf("selection step with invalid command = %#v, want zero locator", got.state)
		}
	})
}

func TestCompletionTargetOnPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()

		if got := completionTargetOn(CompletionTargetLocator{}, CommandByID(1)); got.state != nil {
			t.Fatalf("zero completion target = %#v, want zero locator", got.state)
		}
	})
	t.Run("invalid completion-target subject locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		got := completionTargetOn(completionTargetIndexLocator(0), CommandLocator{})
		if got.state == nil || !got.state.subjectSet {
			t.Fatalf("completion target with invalid subject = %#v", got.state)
		}
		completion := cli.Completion{Targets: []cli.CompletionTarget{{}}}
		resolved := resolveCompletionTarget(got.state, completion)
		if resolved.status != locatorResolutionInvalid {
			t.Fatalf("completion target with invalid subject resolved = %#v", resolved)
		}
	})
}

func TestCompletionTargetResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("expected subject unresolved: semantic absence", func(t *testing.T) {
		t.Parallel()

		result := cli.Completion{
			Items: []cli.CompletionItem{{
				Value: "one",
			}},
		}
		missingItem := completionItemTargetAt(completionItemIndexLocator(4), 0)
		if got := resolveCompletionTarget(missingItem.state, result); got.status != locatorResolutionAbsent {
			t.Fatalf("target through absent item = %#v, want absent", got)
		}
		missingRelation := completionItemTargetAt(completionItemIndexLocator(0), 0)
		if got := resolveCompletionTarget(
			missingRelation.state,
			result,
		); got.status != locatorResolutionAbsent ||
			renderEvaluationField(got.actual) != "item target relationship is absent" {
			t.Fatalf("absent item target relationship = %#v", got)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()

		request := completionNativeRequestIndexLocator(2)
		target := completionNativeRequestTarget(request)
		if got := resolveCompletionTarget(target.state, cli.Completion{}); got.status != locatorResolutionAbsent {
			t.Fatalf("target through absent native request = %#v, want absent", got)
		}
	})
	t.Run("completion target kind: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		if got := resolveCompletionTarget(&completionTargetLocatorState{}, cli.Completion{}); got.status != locatorResolutionInvalid {
			t.Fatalf("invalid completion target kind = %#v", got)
		}
	})
}

func TestCompletionTargetMatchesSubjectPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()

		if matched, status := completionTargetMatchesSubject(cli.CompletionTarget{}, nil); matched || status != locatorResolutionInvalid {
			t.Fatalf("nil target subject = %t/%v, want false/invalid", matched, status)
		}
	})
	t.Run("declaration kind argument form: semantic result", func(t *testing.T) {
		t.Parallel()

		if matched, status := completionTargetMatchesSubject(cli.CompletionTarget{}, FlagByID(1)); matched || status != locatorResolutionNotProvable {
			t.Fatalf("unsupported declaration target subject = %t/%v, want false/not-provable", matched, status)
		}
	})
}

func TestCompletionItemResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("completion item: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		if got := resolveCompletionItem(nil, cli.Completion{}); got.status != locatorResolutionInvalid {
			t.Fatalf("nil completion item locator = %#v", got)
		}
	})
	t.Run("completion item value: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		if got := resolveCompletionItem(&completionItemLocatorState{}, cli.Completion{}); got.status != locatorResolutionInvalid {
			t.Fatalf("wrong-kind completion item locator = %#v", got)
		}
	})
}

func TestCompletionGroupResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("completion group: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		if got := resolveCompletionGroup(nil, cli.Completion{}); got.status != locatorResolutionInvalid {
			t.Fatalf("nil completion group locator = %#v", got)
		}
	})
	t.Run("completion group id: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		if got := resolveCompletionGroup(&completionGroupLocatorState{}, cli.Completion{}); got.status != locatorResolutionInvalid {
			t.Fatalf("wrong-kind completion group locator = %#v", got)
		}
	})
}

func TestCompletionNativeRequestResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("completion native request: invalid coordinates", func(t *testing.T) {
		t.Parallel()

		if got := resolveCompletionNativeRequest(nil, cli.Completion{}); got.status != locatorResolutionInvalid {
			t.Fatalf("nil native-request locator = %#v", got)
		}
	})
	t.Run("completion native request is absent: semantic absence", func(t *testing.T) {
		t.Parallel()

		if got := resolveCompletionNativeRequest(
			completionNativeRequestIndexLocator(3).state,
			cli.Completion{},
		); got.status != locatorResolutionAbsent {
			t.Fatalf("absent native request = %#v, want absent", got)
		}
	})
}

func TestCompletionResolutionInternalStateFailsClosed(t *testing.T) {
	t.Parallel()

	t.Run("zero command default selection step", func(t *testing.T) {
		t.Parallel()

		if got := commandDefaultSelectionStepLocator(CommandDefaultLocator{}); got.state != nil {
			t.Fatalf("selection step state = %#v, want nil", got.state)
		}
	})

	t.Run("complete observation reason", func(t *testing.T) {
		t.Parallel()

		if got := completionObservationIncompleteReason(cli.Completion{}); got != "completion observation is complete" {
			t.Fatalf("reason = %q", got)
		}
	})

	t.Run("item target outside target set", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			Items: []cli.CompletionItem{{TargetIndexes: []int{0}}},
		}
		got := resolveCompletionTarget(completionItemTargetAt(completionItemIndexLocator(0), 0).state, completion)
		if got.status != locatorResolutionInvalid {
			t.Fatalf("status = %v, want invalid", got.status)
		}
	})

	t.Run("native target outside target set", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			NativeRequests: []cli.CompletionNativeRequest{{TargetIndex: 0}},
		}
		got := resolveCompletionTarget(completionNativeRequestTarget(completionNativeRequestIndexLocator(0)).state, completion)
		if got.status != locatorResolutionInvalid {
			t.Fatalf("status = %v, want invalid", got.status)
		}
	})

	t.Run("found resolution outcome", func(t *testing.T) {
		t.Parallel()

		if got := resolutionOutcome(
			locatorResolutionFound,
			evaluationText("subject"),
			evaluationText("expected"),
			evaluationText("actual"),
		); got.status != evaluationProved {
			t.Fatalf("outcome = %#v, want satisfied", got)
		}
	})
}

func TestCompletionTargetSubjectRejectsUnsupportedLocatorKinds(t *testing.T) {
	t.Parallel()
	if matched, status := completionTargetMatchesSubject(cli.CompletionTarget{}, FlagByID(1)); matched || status != locatorResolutionNotProvable {
		t.Fatalf("unsupported target subject = matched %t, status %v", matched, status)
	}
}
