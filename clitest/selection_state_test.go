package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestAdditionalSelectionExpectations(t *testing.T) {
	t.Parallel()
	origin := cli.InputOriginID{}
	inspection := cli.Inspection{
		Selection: cli.InvocationSelection{
			Command: cli.CommandSelection{
				Path: []cli.CommandID{1, 2},
				Steps: []cli.CommandSelectionStep{{
					Kind:    cli.CommandSelectionStepDefault,
					Default: 11,
				}},
			},
			Action: cli.InvocationActionSelection{
				Selected: cli.InvocationActionVersion,
				Trigger:  cli.InvocationActionTriggerEntry,
				Entry:    12,
				Input:    origin,
			},
		},
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:      1,
				Path:    []string{},
				Profile: 21,
			},
			2: {
				ID:   2,
				Path: []string{"serve"},
				ArgumentFormSelection: cli.ArgumentFormSelection{
					Selected: 31,
				},
			},
		},
		Origins: map[cli.InputOriginID]cli.InputOrigin{origin: {
			ID: origin,
		}},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
			31: {
				ID:      31,
				Command: 2,
				Outcome: cli.ArgumentFormOutcomeSelected,
			},
		},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultInspection{
			11: {
				ID:               11,
				ConditionOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
	}

	assertInspectionExpectation(t, InvocationAction(cli.InvocationActionVersion).Selected(), inspection)
	assertInspectionExpectation(t, InvocationAction(cli.InvocationActionVersion).Trigger(cli.InvocationActionTriggerEntry), inspection)
	assertInspectionExpectation(t, InvocationAction(cli.InvocationActionVersion).SelectedEntry(ActionEntryByID(12)), inspection)
	assertInspectionExpectation(t, InvocationAction(cli.InvocationActionVersion).TriggeredBy(InputOriginByID(origin)), inspection)
	assertInspectionExpectation(t, ActionEntryByID(12).Selected(), inspection)
	assertInspectionExpectation(t, GrammarProfileByID(21).Selected(), inspection)
	assertInspectionExpectation(t, ArgumentFormByID(31).Outcome(cli.ArgumentFormOutcomeSelected), inspection)
	assertInspectionExpectation(t, ArgumentFormByID(31).Selected(), inspection)
	assertInspectionExpectation(t, CommandDefaultByID(11).ConditionOutcome(cli.EvaluationOutcomeSatisfied), inspection)
	assertInspectionExpectation(t, CommandDefaultByID(11).Selected(), inspection)
}

func TestCommandSelectionDetailExpectations(t *testing.T) {
	t.Parallel()
	root := cli.CommandID(1)
	def1, def2 := cli.CommandDefaultID(10), cli.CommandDefaultID(11)
	form1, form2 := cli.ArgumentFormID(20), cli.ArgumentFormID(21)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			root: {
				ID:              root,
				Path:            []string{"root"},
				CommandDefaults: []cli.CommandDefaultID{def1, def2},
				ArgumentForms:   []cli.ArgumentFormID{form1, form2},
				DefaultSelection: cli.DefaultCommandSelection{
					Outcome:    cli.DefaultCommandSelectionSelected,
					Candidates: []cli.CommandDefaultID{def1, def2},
					Applicable: []cli.CommandDefaultID{def1},
					Blocked:    []cli.CommandDefaultID{def2},
					Selected:   def1,
				},
				ArgumentFormSelection: cli.ArgumentFormSelection{
					Outcome:  cli.ArgumentFormSelectionSelected,
					Selected: form1,
					Matches:  []cli.ArgumentFormID{form1},
					Blocked:  []cli.ArgumentFormID{form2},
				},
			},
		},
	}
	locator := CommandByID(root)
	state := &inspectionResultState{
		observation: inspection,
		err:         nil,
	}
	(InspectionResult{
		state: state,
	}).Require(t,
		locator.DefaultSelectionOutcome(cli.DefaultCommandSelectionSelected),
		locator.DefaultCandidatesExactly(CommandDefaultByID(def1), CommandDefaultByID(def2)),
		locator.ApplicableDefaultsExactly(CommandDefaultByID(def1)),
		locator.BlockedDefaultsExactly(CommandDefaultByID(def2)),
		locator.ArgumentFormSelectionOutcome(cli.ArgumentFormSelectionSelected),
		locator.MatchingArgumentFormsExactly(ArgumentFormByID(form1)),
		locator.BlockedArgumentFormsExactly(ArgumentFormByID(form2)),
		locator.FailedArgumentFormsExactly())
}

func TestCommandEntrySelectionUsesInspectionAndCompletion(t *testing.T) {
	t.Parallel()
	entryID := cli.CommandEntryID(71)
	selection := cli.CommandSelection{
		Steps: []cli.CommandSelectionStep{{
			Kind:  cli.CommandSelectionStepEntry,
			Entry: entryID,
		}},
	}
	expectation := CommandEntryByID(entryID).Selected()
	(InspectionResult{
		state: &inspectionResultState{
			observation: cli.Inspection{
				Selection: cli.InvocationSelection{
					Command: selection,
				},
			},
			err: nil,
		},
	}).Require(t,

		expectation)
	(CompletionResult{
		state: &completionResultState{
			observation: cli.Completion{
				Selection: selection,
			},
			err: nil,
		},
	}).Require(t,

		expectation)
}

func TestSelectionNegativePartialIdentityAndCompletionLaws(t *testing.T) {
	t.Parallel()
	rootID := cli.CommandID(1)
	serveID := cli.CommandID(2)
	dupAID := cli.CommandID(3)
	dupBID := cli.CommandID(4)
	entryID := cli.CommandEntryID(10)
	defaultID := cli.CommandDefaultID(11)
	formID := cli.ArgumentFormID(12)
	profileID := cli.GrammarProfileID(13)
	originID := cli.InputOriginID{}

	commands := map[cli.CommandID]cli.CommandInspection{
		rootID: {
			ID:                    rootID,
			Children:              []cli.CommandID{serveID, dupAID, dupBID},
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		},
		serveID: {
			ID:      serveID,
			Name:    "serve",
			Path:    []string{"serve"},
			Profile: profileID,
			ArgumentFormSelection: cli.ArgumentFormSelection{
				Selected: formID,
			},
		},
		dupAID: {
			ID:   dupAID,
			Name: "dup",
			Path: []string{"dup"},
		},
		dupBID: {
			ID:   dupBID,
			Name: "dup",
			Path: []string{"dup"},
		},
	}
	selection := cli.CommandSelection{
		Path: []cli.CommandID{rootID, serveID},
		Steps: []cli.CommandSelectionStep{
			{
				Kind:     cli.CommandSelectionStepEntry,
				Entry:    entryID,
				Input:    originID,
				Commands: []cli.CommandID{serveID},
			},
			{
				Kind:    cli.CommandSelectionStepDefault,
				Default: defaultID,
			},
		},
	}
	inspection := cli.Inspection{
		Selection: cli.InvocationSelection{
			Command: selection,
		},
		Commands: commands,
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{
			entryID: {
				ID:         entryID,
				Parent:     rootID,
				TargetPath: []cli.CommandID{serveID},
			},
		},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultInspection{
			defaultID: {
				ID:               defaultID,
				Parent:           rootID,
				Target:           serveID,
				ConditionOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
		Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {
			ID: originID,
		}},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
			formID: {
				ID:      formID,
				Command: serveID,
				Outcome: cli.ArgumentFormOutcomeSelected,
			},
		},
	}

	for _, test := range []struct {
		name        string
		expectation InspectionExpectation
	}{
		{
			name:        "unselected command dischecks not-selected",
			expectation: CommandByID(serveID).NotSelected(),
		},
		{
			name:        "missing command dischecks selected",
			expectation: CommandByID(99).Selected(),
		},
		{
			name:        "entry absent from steps",
			expectation: CommandEntryByID(99).Selected(),
		},
		{
			name:        "entry present dischecks not-selected",
			expectation: CommandEntryByID(entryID).NotSelected(),
		},
		{
			name:        "profile absent",
			expectation: GrammarProfileByID(99).Selected(),
		},
		{
			name:        "form absent",
			expectation: ArgumentFormByID(99).Selected(),
		},
		{
			name:        "form present dischecks not-selected",
			expectation: ArgumentFormByID(formID).NotSelected(),
		},
		{
			name:        "default present dischecks not-selected",
			expectation: CommandDefaultByID(defaultID).NotSelected(),
		},
		{
			name: "default condition mismatch",
			expectation: CommandDefaultByID(defaultID).ConditionOutcome(
				cli.EvaluationOutcomeUnsatisfied,
			),
		},
	} {
		name := test.name
		expectation := test.expectation
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationDisproved {
			t.Fatalf("%s = %#v, want disproved", name, got)
		}
	}

	if got := inspectionExpectationOutcome(
		t,
		RootCommand().Command("dup").Selected(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("ambiguous child selection = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		RootCommand().Command("missing").Selected(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("missing child selection = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		SelectionPathExactly(RootCommand(), Command("other")),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("path identity mismatch = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		SelectionStepsExactly(CommandEntryByID(entryID).SelectionStep()),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("step cardinality mismatch = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(t, SelectionStepsExactly(
		CommandEntryByID(entryID).SelectionStep().From(InputOriginByID(originID)).Selects(CommandByID(serveID)),
		CommandDefaultByID(defaultID).SelectionStep(),
	), inspection); got.status != evaluationProved {
		t.Fatalf("selection step identity/order = %#v, want proved", got)
	}

	completion := cli.Completion{
		Selection: selection,
		Origins: map[cli.InputOriginID]cli.InputOrigin{originID: {
			ID: originID,
		}},
	}
	for _, test := range []struct {
		name        string
		expectation CompletionExpectation
	}{
		{
			name:        "default selected",
			expectation: CommandDefaultByID(defaultID).Selected(),
		},
		{
			name:        "default not selected",
			expectation: CommandDefaultByID(99).NotSelected(),
		},
		{
			name:        "entry selected",
			expectation: CommandEntryByID(entryID).Selected(),
		},
		{
			name:        "entry not selected",
			expectation: CommandEntryByID(99).NotSelected(),
		},
	} {
		name := test.name
		expectation := test.expectation
		evaluator, ok := expectation.(interface {
			evaluateCompletion(completion cli.Completion, err error) evaluationResult
		})
		if !ok {
			t.Fatalf("completion expectation %T does not expose completion evaluation", expectation)
		}
		got := evaluator.evaluateCompletion(completion, nil)
		if got.status != evaluationProved {
			t.Fatalf("completion %s = %#v, want proved", name, got)
		}
	}
	defaultNotSelected, ok := CommandDefaultByID(defaultID).NotSelected().(interface {
		evaluateCompletion(completion cli.Completion, err error) evaluationResult
	})
	if !ok {
		t.Fatal("CommandDefault NotSelected expectation does not expose completion evaluation")
	}
	if got := defaultNotSelected.evaluateCompletion(completion, nil); got.status != evaluationDisproved {
		t.Fatalf("completion selected default negative = %#v, want disproved", got)
	}
	entryNotSelected, ok := CommandEntryByID(entryID).NotSelected().(interface {
		evaluateCompletion(completion cli.Completion, err error) evaluationResult
	})
	if !ok {
		t.Fatal("CommandEntry NotSelected expectation does not expose completion evaluation")
	}
	if got := entryNotSelected.evaluateCompletion(completion, nil); got.status != evaluationDisproved {
		t.Fatalf("completion selected entry negative = %#v, want disproved", got)
	}

	partial := cli.Inspection{
		Selection: cli.InvocationSelection{
			Command: cli.CommandSelection{
				Path: []cli.CommandID{rootID},
			},
		},
		Commands: map[cli.CommandID]cli.CommandInspection{rootID: {
			ID:       rootID,
			Children: []cli.CommandID{serveID},
		}},
	}
	if got := inspectionExpectationOutcome(
		t,
		RootCommand().Command("serve").Selected(),
		partial,
	); got.status != evaluationNotProvable {
		t.Fatalf("missing child details = %#v, want not provable", got)
	}
	partial.Truncated = true
	if got := inspectionExpectationOutcome(
		t,
		ArgumentFormByID(formID).Selected(),
		partial,
	); got.status != evaluationNotProvable {
		t.Fatalf("truncated argument form = %#v, want not provable", got)
	}

	var zeroCommand CommandLocator
	if got := inspectionExpectationOutcome(
		t,
		zeroCommand.Selected(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("zero command locator = %#v, want deterministic disproval", got)
	}
	var zeroEntry CommandEntryLocator
	if got := inspectionExpectationOutcome(
		t,
		zeroEntry.Selected(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("zero entry locator = %#v, want deterministic disproval", got)
	}
}

func TestArgumentFormSelectionRequiresObservableOwningCommand(t *testing.T) {
	t.Parallel()

	formID := cli.ArgumentFormID(1)
	commandID := cli.CommandID(2)
	partial := cli.Inspection{
		Truncated: true,
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
			formID: {
				ID:      formID,
				Command: commandID,
				Outcome: cli.ArgumentFormOutcomeNoMatch,
			},
		},
	}

	for _, expectation := range []InspectionExpectation{
		ArgumentFormByID(formID).Selected(),
		ArgumentFormByID(formID).NotSelected(),
	} {
		if outcome := inspectionExpectationOutcome(
			t,
			expectation,
			partial,
		); outcome.status != evaluationNotProvable {
			t.Fatalf("%T = %#v, want not provable", expectation, outcome)
		}
	}
}

func TestCommandEntrySelectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	entryID := cli.CommandEntryID(10)
	selection := cli.CommandSelection{
		Steps: []cli.CommandSelectionStep{{
			Kind:  cli.CommandSelectionStepEntry,
			Entry: entryID,
		}},
	}

	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).EntryAt(0).state
		outcome := commandEntrySelectionExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("incomplete command-entry lookup = %#v", outcome)
		}
	})
	t.Run("command-entry selection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).EntryAt(0).state
		outcome := commandEntrySelectionExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status != evaluationNotProvable || renderEvaluationField(outcome.subject) != "command-entry selection" {
			t.Fatalf("incomplete command-entry selection = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCommandEntrySelection(entryID, false, selection)
		if outcome.status != evaluationDisproved {
			t.Fatalf("selected entry negative = %#v", outcome)
		}
	})
	t.Run("commandentryselection unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(99).EntryAt(0).state
		outcome := commandEntrySelectionExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("unresolved command-entry selection = %#v", outcome)
		}
	})
	t.Run("commandentryselection unresolved: semantic result", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Command: selection,
			},
			Commands: map[cli.CommandID]cli.CommandInspection{
				1: {
					ID:                    1,
					NamespaceCompleteness: cli.CommandNamespaceComplete,
					CommandEntries:        []cli.CommandEntryID{entryID},
				},
			},
			CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{entryID: {
				ID:     entryID,
				Parent: 1,
			}},
		}
		state := CommandByID(1).EntryAt(0).state
		outcome := commandEntrySelectionExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(inspection, nil)
		if outcome.status != evaluationProved {
			t.Fatalf("resolved command-entry selection = %#v", outcome)
		}
	})
	t.Run("command-entry selection: semantic result", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).EntryAt(0).state
		outcome := commandEntrySelectionExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateCompletion(cli.Completion{
			Selection: selection,
		}, nil)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("completion relational command-entry selection = %#v", outcome)
		}
	})
}

func TestSelectionCommandResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	state := CommandByID(1).Command("missing").Command("child").state
	commands := map[cli.CommandID]cli.CommandInspection{1: {
		ID:                    1,
		NamespaceCompleteness: cli.CommandNamespaceComplete,
	}}
	_, status, actual := resolveSelectionCommand(state, commands)
	if status != locatorResolutionAbsent || renderEvaluationField(actual) != "child command is absent" {
		t.Fatalf("nested command resolution = status %v, actual %q", status, actual)
	}
}

func TestChildCommandResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	_, status, actual := resolveInspectionCommandChild(1, "child", map[cli.CommandID]cli.CommandInspection{})
	if status != locatorResolutionNotProvable || renderEvaluationField(actual) != "parent command details are absent" {
		t.Fatalf("missing parent = status %v, actual %q", status, actual)
	}
}

func TestCommandPathResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("deterministic match ordering: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := map[cli.CommandID]cli.CommandInspection{
			2: {
				ID: 2,
			}, 1: {
				ID: 1,
			},
		}
		_, status, actual := resolveInspectionCommandPath([]string{"x"}, inspection)
		if status != locatorResolutionAmbiguous || renderEvaluationField(actual) != "matches root-command IDs; count=2" {
			t.Fatalf("sorted roots = status %v, actual %q", status, actual)
		}
	})
	t.Run("ambiguous root: ambiguous identity", func(t *testing.T) {
		t.Parallel()
		inspection := map[cli.CommandID]cli.CommandInspection{1: {
			ID: 1,
		}, 2: {
			ID: 2,
		}}
		_, status, _ := resolveInspectionCommandPath(nil, inspection)
		if status != locatorResolutionAmbiguous {
			t.Fatalf("ambiguous roots = %v", status)
		}
	})
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		inspection := map[cli.CommandID]cli.CommandInspection{1: {
			ID: 1,
		}}
		id, status, actual := resolveInspectionCommandPath(nil, inspection)
		if id != 1 || status != locatorResolutionFound || renderEvaluationField(actual) != "" {
			t.Fatalf("root path = id %v status %v actual %q", id, status, actual)
		}
	})
	t.Run("command details are absent during path traversal: incomplete observation", func(t *testing.T) {
		t.Parallel()
		commands := map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:                    1,
				Children:              []cli.CommandID{2},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
			2: {
				ID:   2,
				Name: "serve",
				Path: []string{"serve"},
			},
		}
		delete(commands, 2)
		_, status, actual := resolveInspectionCommandPath([]string{"serve"}, commands)
		if status != locatorResolutionNotProvable ||
			renderEvaluationField(actual) != "child command details needed for name lookup are absent" {
			t.Fatalf("missing traversal detail = status %v actual %q", status, actual)
		}
	})
	t.Run(
		"command relationship topology contradicts observed command path: incomplete observation",
		func(t *testing.T) {
			t.Parallel()
			commands := map[cli.CommandID]cli.CommandInspection{
				1: {
					ID:                    1,
					Children:              []cli.CommandID{2},
					NamespaceCompleteness: cli.CommandNamespaceComplete,
				},
				2: {
					ID:   2,
					Name: "serve",
					Path: []string{"different"},
				},
			}
			_, status, actual := resolveInspectionCommandPath([]string{"serve"}, commands)
			if status != locatorResolutionNotProvable ||
				renderEvaluationField(actual) != "command relationship topology contradicts observed command path" {
				t.Fatalf("contradictory path = status %v actual %q", status, actual)
			}
		},
	)
}

func TestSelectionStateSelectionPathPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("selection path: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateSelectionPath(
			[]*commandLocatorState{CommandByID(1).state},
			cli.CommandSelection{},
			nil,
		)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.subject) != "selection path" {
			t.Fatalf("path cardinality = %#v", outcome)
		}
	})
	t.Run("selection path: incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).Command("child").state
		selection := cli.CommandSelection{
			Path: []cli.CommandID{2},
		}
		inspection := map[cli.CommandID]cli.CommandInspection{1: {
			ID:       1,
			Children: []cli.CommandID{2},
		}}
		outcome := evaluateSelectionPath([]*commandLocatorState{state}, selection, inspection)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("incomplete path = %#v", outcome)
		}
	})
	t.Run("expected subject unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		selection := cli.CommandSelection{
			Path: []cli.CommandID{2},
		}
		outcome := evaluateSelectionPath([]*commandLocatorState{CommandByID(1).state}, selection, nil)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "command 2" {
			t.Fatalf("path identity mismatch = %#v", outcome)
		}
	})
}

func TestSelectionStepsPreserveTypedSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		outcome := matchSelectionStep(nil, cli.CommandSelectionStep{}, nil, nil, nil, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("invalid step = %#v", outcome)
		}
	})

	t.Run("exact entry", func(t *testing.T) {
		t.Parallel()
		step := CommandEntryByID(7).SelectionStep()
		selectionStep := cli.CommandSelectionStep{
			Kind:  cli.CommandSelectionStepEntry,
			Entry: 7,
		}
		outcome := matchSelectionStep(step.state, selectionStep, nil, nil, nil, nil)
		if outcome.status != evaluationProved {
			t.Fatalf("entry step = %#v", outcome)
		}
	})

	t.Run("human entry needs observable lookup domain", func(t *testing.T) {
		t.Parallel()
		step := CommandByID(1).Alias("legacy").SelectionStep()
		outcome := matchSelectionStep(step.state, cli.CommandSelectionStep{
			Kind: cli.CommandSelectionStepEntry,
		}, nil, nil, nil, nil)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("unobservable human entry = %#v", outcome)
		}
	})

	t.Run("exact default", func(t *testing.T) {
		t.Parallel()
		step := CommandDefaultByID(9).SelectionStep()
		selectionStep := cli.CommandSelectionStep{
			Kind:    cli.CommandSelectionStepDefault,
			Default: 9,
		}
		outcome := matchSelectionStep(step.state, selectionStep, nil, nil, nil, nil)
		if outcome.status != evaluationProved {
			t.Fatalf("default step = %#v", outcome)
		}
	})

	t.Run("origin and selected command order", func(t *testing.T) {
		t.Parallel()
		origin := cli.InputOriginID{}
		step := CommandEntryByID(7).SelectionStep().From(InputOriginByID(origin)).Selects(CommandByID(2), CommandByID(3))
		observed := cli.CommandSelectionStep{
			Kind:     cli.CommandSelectionStepEntry,
			Entry:    7,
			Input:    origin,
			Commands: []cli.CommandID{2, 3},
		}
		origins := map[cli.InputOriginID]cli.InputOrigin{origin: {
			ID: origin,
		}}
		outcome := matchSelectionStep(step.state, observed, nil, nil, nil, origins)
		if outcome.status != evaluationProved {
			t.Fatalf("exact step = %#v", outcome)
		}

		observed.Commands = []cli.CommandID{3, 2}
		firstOrigins := map[cli.InputOriginID]cli.InputOrigin{origin: {
			ID: origin,
		}}
		outcome = matchSelectionStep(step.state, observed, nil, nil, nil, firstOrigins)
		if outcome.status != evaluationDisproved {
			t.Fatalf("reordered commands = %#v", outcome)
		}
	})

	t.Run("aggregate step order", func(t *testing.T) {
		t.Parallel()
		states := []*selectionStepLocatorState{
			CommandEntryByID(7).SelectionStep().state,
			CommandDefaultByID(9).SelectionStep().state,
		}
		selection := cli.CommandSelection{
			Steps: []cli.CommandSelectionStep{
				{
					Kind:  cli.CommandSelectionStepEntry,
					Entry: 7,
				},
				{
					Kind:    cli.CommandSelectionStepDefault,
					Default: 9,
				},
			},
		}
		if outcome := evaluateSelectionSteps(states, selection, nil, nil, nil, nil); outcome.status != evaluationProved {
			t.Fatalf("selection steps = %#v", outcome)
		}
	})
}

func TestOriginIDResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	zero := cli.InputOriginID{}
	other := zero
	t.Run("invalid origin locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		_, status, actual := resolveSelectionOriginID(&inputOriginLocatorState{}, nil)
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid origin locator" {
			t.Fatalf("invalid origin = %v %q", status, actual)
		}
	})
	t.Run("input origin parent index: matching observation", func(t *testing.T) {
		t.Parallel()
		state := InputOriginByID(zero).ParentAt(0).state
		origins := map[cli.InputOriginID]cli.InputOrigin{zero: {
			ID:      zero,
			Parents: []cli.InputOriginID{other},
		}}
		id, status, _ := resolveSelectionOriginID(
			state, origins,
		)
		if status != locatorResolutionFound || id != other {
			t.Fatalf("parent origin = %v %v", id, status)
		}
	})
	t.Run("input origin parent index: incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := InputOriginByID(zero).ParentAt(0).state
		_, status, actual := resolveSelectionOriginID(state, nil)
		if status != locatorResolutionNotProvable || renderEvaluationField(actual) != "operation does not expose origin relationships" {
			t.Fatalf("origin relationships = %v %q", status, actual)
		}
	})
	t.Run("input origin parent index: semantic absence", func(t *testing.T) {
		t.Parallel()
		state := InputOriginByID(zero).ParentAt(0).state
		_, status, actual := resolveSelectionOriginID(state, map[cli.InputOriginID]cli.InputOrigin{})
		if status != locatorResolutionAbsent || renderEvaluationField(actual) != "parent origin is absent" {
			t.Fatalf("absent parent origin = %v %q", status, actual)
		}
	})
	t.Run("input origin parent index: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		state := InputOriginByID(zero).ParentAt(1).state
		origins := map[cli.InputOriginID]cli.InputOrigin{zero: {
			ID: zero,
		}}
		_, status, actual := resolveSelectionOriginID(state, origins)
		if status != locatorResolutionAbsent || renderEvaluationField(actual) != "origin parent relationship is absent" {
			t.Fatalf("missing parent edge = %v %q", status, actual)
		}
	})
}

func TestActionSelectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("invocation action: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		outcome := actionSelectionExpectationValue{
			selected: true,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid action locator" {
			t.Fatalf("invalid action = %#v", outcome)
		}
	})
	t.Run("invocation action: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Selected: cli.InvocationActionVersion,
				},
			},
		}
		outcome := actionSelectionExpectationValue{
			locator:  actionLocator(cli.InvocationActionHelp).state,
			selected: true,
		}.evaluateInspection(inspection, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("action mismatch = %#v", outcome)
		}
	})
}

func TestActionPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	t.Run("invocation action: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		expectation := actionTriggerExpectation(InvocationActionLocator{}, cli.InvocationActionTriggerEntry)
		outcome := inspectionExpectationOutcome(t, expectation, cli.Inspection{})
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid action locator" {
			t.Fatalf("invalid action property = %#v", outcome)
		}
	})
	t.Run("invocation action: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expectation := actionTriggerExpectation(actionLocator(cli.InvocationActionHelp), cli.InvocationActionTriggerEntry)
		inspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Selected: cli.InvocationActionVersion,
				},
			},
		}
		outcome := inspectionExpectationOutcome(t, expectation, inspection)
		if outcome.status != evaluationDisproved {
			t.Fatalf("action subject mismatch = %#v", outcome)
		}
		firstInspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Selected: cli.InvocationActionHelp,
				},
			},
		}
		outcome = inspectionExpectationOutcome(t, expectation, firstInspection)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
			t.Fatalf("action property mismatch = %#v", outcome)
		}
	})
}

func TestActionEntryRejectsInvalidCoordinates(t *testing.T) {
	t.Parallel()

	t.Run("invalid action-entry locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		expectation := actionEntryExpectation(actionLocator(cli.InvocationActionHelp), ActionEntryLocator{})
		inspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Selected: cli.InvocationActionHelp,
				},
			},
		}
		outcome := inspectionExpectationOutcome(t, expectation, inspection)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid action entry locator" {
			t.Fatalf("invalid action entry = %#v", outcome)
		}
	})
	t.Run("unresolved relational entry", func(t *testing.T) {
		t.Parallel()
		entry := CommandByID(1).ActionEntryAt(0)
		expectation := actionEntryExpectation(actionLocator(cli.InvocationActionHelp), entry)
		inspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Selected: cli.InvocationActionHelp,
				},
			},
			Commands: map[cli.CommandID]cli.CommandInspection{1: {
				ID: 1,
			}},
		}
		outcome := inspectionExpectationOutcome(t, expectation, inspection)
		if outcome.status == evaluationProved {
			t.Fatalf("unresolved action entry unexpectedly matched")
		}
	})
}

func TestActionEntryRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	expectation := actionEntryExpectation(actionLocator(cli.InvocationActionHelp), ActionEntryByID(1))
	inspection := cli.Inspection{
		Selection: cli.InvocationSelection{
			Action: cli.InvocationActionSelection{
				Selected: cli.InvocationActionHelp,
				Entry:    2,
			},
		},
	}
	outcome := inspectionExpectationOutcome(t, expectation, inspection)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "action entry 2" {
		t.Fatalf("action entry mismatch = %#v", outcome)
	}
}

func TestActionOriginRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	origin := cli.InputOriginID{}
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		unresolved := actionOriginExpectation(actionLocator(cli.InvocationActionHelp), InputOriginByID(origin).ParentAt(0))
		firstInspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Selected: cli.InvocationActionHelp,
				},
			},
			Truncated: true,
		}
		outcome := inspectionExpectationOutcome(t, unresolved, firstInspection)
		if outcome.status == evaluationProved {
			t.Fatalf("missing origin graph unexpectedly matched")
		}

		definition := cli.Definition[struct{}]{
			Name:    "action-origin",
			Release: cli.Release.SemVer("1.0.0"),
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}
		request := cli.InspectionRequest{
			Args: []string{"--version", "unexpected"},
		}
		inspection, _ := app.Inspect(t.Context(), request)
		if inspection.Selection.Action.Input == origin {
			t.Fatal("fixture did not establish a distinct action input origin")
		}
		expectation := actionOriginExpectation(actionLocator(cli.InvocationActionHelp), InputOriginByID(origin))
		outcome = inspectionExpectationOutcome(t, expectation, inspection)
		if outcome.status != evaluationDisproved {
			t.Fatalf("origin mismatch = %#v", outcome)
		}
	})
}

func TestActionEntrySelectedPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	entryID := cli.ActionEntryID(1)
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).ActionEntryAt(0).state
		outcome := actionEntrySelectedExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("incomplete action entry = %#v", outcome)
		}
	})
	t.Run("action entry: incomplete observation", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).ActionEntryAt(0).state
		outcome := actionEntrySelectedExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status != evaluationNotProvable || renderEvaluationField(outcome.subject) != "action entry" {
			t.Fatalf("incomplete action entry subject = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Entry: 2,
				},
			},
		}
		outcome := actionEntrySelectedExpectationValue{
			locator:  ActionEntryByID(entryID).state,
			selected: true,
		}.evaluateInspection(inspection, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("action entry mismatch = %#v", outcome)
		}
	})
	t.Run("actionentryselected unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(99).ActionEntryAt(0).state
		outcome := actionEntrySelectedExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("unresolved action entry = %#v", outcome)
		}
	})
	t.Run("actionentryselected unresolved: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		outcome := actionEntrySelectedExpectationValue{
			locator:  &actionEntryLocatorState{},
			selected: true,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("invalid resolved action entry = %#v", outcome)
		}
	})
	t.Run("action entry: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		outcome := actionEntrySelectedExpectationValue{
			selected: true,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid action entry locator" {
			t.Fatalf("nil action entry = %#v", outcome)
		}
	})
	t.Run("action entry: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Entry: entryID,
				},
			},
		}
		outcome := actionEntrySelectedExpectationValue{
			locator:  ActionEntryByID(entryID).state,
			selected: false,
		}.evaluateInspection(inspection, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("selected action entry negative = %#v", outcome)
		}
	})
	t.Run("relational action entry: matching observation", func(t *testing.T) {
		t.Parallel()
		const entryID cli.ActionEntryID = 11
		state := CommandByID(1).ActionEntryAt(0).state
		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{
				1: {
					ID:            1,
					ActionEntries: []cli.ActionEntryID{entryID},
				},
			},
			ActionEntries: map[cli.ActionEntryID]cli.ActionEntryInspection{entryID: {
				ID: entryID,
			}},
			Selection: cli.InvocationSelection{
				Action: cli.InvocationActionSelection{
					Entry: entryID,
				},
			},
		}
		outcome := actionEntrySelectedExpectationValue{
			locator:  state,
			selected: true,
		}.evaluateInspection(inspection, nil)
		if outcome.status != evaluationProved {
			t.Fatalf("relational action entry = %#v", outcome)
		}
	})
}

func TestGrammarProfileSelectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("grammar profile: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		outcome := grammarProfileSelectionExpectationValue{
			locator:  &grammarProfileLocatorState{},
			selected: true,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("invalid grammar-profile locator = %#v", outcome)
		}
		outcome = grammarProfileSelectionExpectationValue{
			selected: true,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("nil grammar-profile locator = %#v", outcome)
		}
	})
}

func TestArgumentFormResolutionPreservesResolutionOutcome(t *testing.T) {
	t.Parallel()

	formID := cli.ArgumentFormID(1)
	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).ArgumentFormAt(0).state
		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{
				1: {
					ID:            1,
					ArgumentForms: []cli.ArgumentFormID{formID},
				},
			},
			ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{formID: {
				ID:      formID,
				Command: 1,
			}},
		}
		resolution := resolveInspectionArgumentFormLocator(state, inspection)
		if resolution.status != locatorResolutionFound || resolution.value.ID != formID {
			t.Fatalf("resolved form = %#v", resolution)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		state := CommandByID(1).ArgumentFormAt(0).state
		resolution := resolveInspectionArgumentFormLocator(state, cli.Inspection{
			Truncated: true,
		})
		if resolution.status != locatorResolutionNotProvable {
			t.Fatalf("unresolved form = %#v", resolution)
		}
	})
	t.Run("expected subject unresolved: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		resolution := resolveInspectionArgumentFormLocator(&argumentFormLocatorState{}, cli.Inspection{})
		if resolution.status != locatorResolutionInvalid {
			t.Fatalf("invalid form = %#v", resolution)
		}
	})
	t.Run("invalid argument-form locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		resolution := resolveInspectionArgumentFormLocator(nil, cli.Inspection{})
		if resolution.status != locatorResolutionInvalid || renderEvaluationField(resolution.detail) != "invalid argument-form locator" {
			t.Fatalf("nil form = %#v", resolution)
		}
	})
}

func TestArgumentFormInspectionPreservesSemanticOutcome(t *testing.T) {
	t.Parallel()

	formID := cli.ArgumentFormID(1)
	expectation := argumentFormOutcomeExpectation(ArgumentFormByID(formID), cli.ArgumentFormOutcomeSelected)
	t.Run("argument form: incomplete observation", func(t *testing.T) {
		t.Parallel()
		outcome := inspectionExpectationOutcome(t, expectation, cli.Inspection{
			Truncated: true,
		})
		if outcome.status != evaluationNotProvable {
			t.Fatalf("truncated form = %#v", outcome)
		}
	})
	t.Run("argumentforminspection unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := inspectionExpectationOutcome(t, expectation, cli.Inspection{})
		if outcome.status != evaluationDisproved {
			t.Fatalf("absent form = %#v", outcome)
		}
	})
	t.Run("argument form: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
				formID: {
					ID:      formID,
					Outcome: cli.ArgumentFormOutcomeNoMatch,
				},
			},
		}
		outcome := inspectionExpectationOutcome(t, expectation, inspection)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
			t.Fatalf("form status mismatch = %#v", outcome)
		}
	})
}

func TestArgumentFormSelectedPreservesSemanticOutcome(t *testing.T) {
	t.Parallel()

	formID := cli.ArgumentFormID(1)
	inspection := cli.Inspection{
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{formID: {
			ID:      formID,
			Command: 2,
		}},
	}
	outcome := argumentFormSelectedExpectationValue{
		locator:  ArgumentFormByID(formID).state,
		selected: true,
	}.evaluateInspection(inspection, nil)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "owning command is unavailable" {
		t.Fatalf("unavailable owning command = %#v", outcome)
	}
}

func TestSelectionStateCommandDefaultSelectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	outcome := evaluateCommandDefaultSelection(nil, true, cli.CommandSelection{})
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "invalid command-default locator" {
		t.Fatalf("invalid command default = %#v", outcome)
	}
}

func TestCommandDefaultConditionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	defaultID := cli.CommandDefaultID(1)
	t.Run("commanddefaultcondition unresolved: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		outcome := commandDefaultConditionExpectationValue{
			locator: &commandDefaultLocatorState{},
			outcome: cli.EvaluationOutcomeSatisfied,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationDisproved {
			t.Fatalf("invalid default locator = %#v", outcome)
		}
	})
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		outcome := commandDefaultConditionExpectationValue{
			locator: CommandDefaultByID(defaultID).state,
			outcome: cli.EvaluationOutcomeSatisfied,
		}.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("truncated default = %#v", outcome)
		}
	})
	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		outcome := commandDefaultConditionExpectationValue{
			locator: CommandDefaultByID(defaultID).state,
			outcome: cli.EvaluationOutcomeSatisfied,
		}.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status != evaluationNotProvable || renderEvaluationField(outcome.observed) != "inspection is truncated" {
			t.Fatalf("truncated default detail = %#v", outcome)
		}
	})
	t.Run("absent command default: semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := commandDefaultConditionExpectationValue{
			locator: CommandDefaultByID(defaultID).state,
			outcome: cli.EvaluationOutcomeSatisfied,
		}.evaluateInspection(cli.Inspection{}, nil)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "command default is absent" {
			t.Fatalf("absent default = %#v", outcome)
		}
	})
}

func TestSelectionInternalStateFailsClosed(t *testing.T) {
	t.Parallel()

	if _, status, _ := resolveSelectionCommand(&commandLocatorState{}, nil); status != locatorResolutionInvalid {
		t.Fatalf("command status = %v, want invalid", status)
	}
	if got := matchSelectionStep(&selectionStepLocatorState{}, cli.CommandSelectionStep{}, nil, nil, nil, nil); got.status != evaluationDisproved {
		t.Fatalf("selection step = %#v", got)
	}
	wrong := &inputOriginLocatorState{}
	if _, status, _ := resolveSelectionOriginID(wrong, nil); status != locatorResolutionInvalid {
		t.Fatalf("origin status = %v, want invalid", status)
	}
	expectation := commandDefaultConditionExpectationValue{
		locator: &commandDefaultLocatorState{},
		outcome: cli.EvaluationOutcomeSatisfied,
	}
	if got := expectation.evaluateInspection(cli.Inspection{}, nil); got.status != evaluationDisproved {
		t.Fatalf("command default condition = %#v", got)
	}
}

func TestSelectionStateConservesNestedResolutionFailures(t *testing.T) {
	t.Parallel()

	rootID, childID := cli.CommandID(1), cli.CommandID(2)
	commands := map[cli.CommandID]cli.CommandInspection{
		rootID: {
			ID:                    rootID,
			Children:              []cli.CommandID{childID},
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		},
		childID: {
			ID:   childID,
			Name: "child",
			Path: []string{"child"},
		},
	}
	if got, status, _ := resolveInspectionCommandPath([]string{"child"}, commands); status != locatorResolutionFound || got != childID {
		t.Fatalf("resolved path = %d, %v", got, status)
	}

	step := cli.CommandSelectionStep{
		Kind: cli.CommandSelectionStepDefault,
	}
	state := &selectionStepLocatorState{
		kind:           selectionStepLocatorDefault,
		commandDefault: &commandDefaultLocatorState{},
	}
	if got := matchSelectionStep(state, step, commands, nil, map[cli.CommandDefaultID]cli.CommandDefaultInspection{}, nil); got.status != evaluationDisproved {
		t.Fatalf("unresolved default = %#v", got)
	}

	state = &selectionStepLocatorState{
		kind:           selectionStepLocatorDefault,
		commandDefault: CommandDefaultByID(1).state,
		origin:         &inputOriginLocatorState{},
	}
	selectionStep := cli.CommandSelectionStep{
		Kind:    cli.CommandSelectionStepDefault,
		Default: 1,
	}
	firstInspection := map[cli.CommandDefaultID]cli.CommandDefaultInspection{1: {
		ID: 1,
	}}
	if got := matchSelectionStep(state, selectionStep, commands, nil, firstInspection, nil); got.status != evaluationDisproved {
		t.Fatalf("invalid configured origin = %#v", got)
	}

	state = &selectionStepLocatorState{
		kind:           selectionStepLocatorDefault,
		commandDefault: CommandDefaultByID(1).state,
		commands:       []*commandLocatorState{{}},
	}
	firstSelectionStep := cli.CommandSelectionStep{
		Kind:     cli.CommandSelectionStepDefault,
		Default:  1,
		Commands: []cli.CommandID{1},
	}
	secondInspection := map[cli.CommandDefaultID]cli.CommandDefaultInspection{1: {
		ID: 1,
	}}
	if got := matchSelectionStep(state, firstSelectionStep, commands, nil, secondInspection, nil); got.status != evaluationDisproved {
		t.Fatalf("invalid selected command = %#v", got)
	}

	if _, status, _ := resolveSelectionOriginID(nil, nil); status != locatorResolutionInvalid {
		t.Fatalf("nil origin status = %v", status)
	}
	if _, status, _ := resolveSelectionOriginID(&inputOriginLocatorState{
		mode: inputOriginLocatorParentIndex,
	}, nil); status != locatorResolutionInvalid {
		t.Fatalf("invalid origin parent status = %v", status)
	}

	originExpectation := actionOriginExpectation(InvocationAction(cli.InvocationActionHelp), InputOriginLocator{})
	inspection := cli.Inspection{
		Selection: cli.InvocationSelection{
			Action: cli.InvocationActionSelection{
				Selected: cli.InvocationActionHelp,
			},
		},
	}
	if got := inspectionExpectationOutcome(t, originExpectation, inspection); got.status != evaluationDisproved {
		t.Fatalf("invalid action origin = %#v", got)
	}

	selected := actionEntrySelectedExpectationValue{
		locator:  &actionEntryLocatorState{},
		selected: true,
	}
	if got := selected.evaluateInspection(cli.Inspection{}, nil); got.status != evaluationDisproved {
		t.Fatalf("invalid action entry = %#v", got)
	}
}
