package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

//nolint:dupl // Schema and Reference fixtures intentionally mirror the same public command relationships without a shared semantic model.
func TestCommandCollectionsAcrossStaticAndInspectionArtifacts(t *testing.T) {
	t.Parallel()
	root, child := cli.CommandID(1), cli.CommandID(2)
	entry, action := cli.CommandEntryID(3), cli.ActionEntryID(4)
	def, provider := cli.CommandDefaultID(5), cli.CommandProviderID(6)
	flag, fact, value := cli.FlagID(7), cli.ValueID(8), cli.ValueID(9)
	form, policy := cli.ArgumentFormID(10), cli.SignalPolicyID(11)
	commandSchema := cli.CommandSchema{
		ID:               root,
		Path:             []string{},
		Children:         []cli.CommandID{child},
		CommandEntries:   []cli.CommandEntryID{entry},
		ActionEntries:    []cli.ActionEntryID{action},
		CommandDefaults:  []cli.CommandDefaultID{def},
		CommandProviders: []cli.CommandProviderID{provider},
		Flags:            []cli.FlagID{flag},
		Facts:            []cli.ValueID{fact},
		Values:           []cli.ValueID{value},
		ArgumentForms:    []cli.ArgumentFormID{form},
		SignalPolicies:   []cli.SignalPolicyID{policy},
	}
	schema := &cli.Schema{
		CommandOrder: []cli.CommandID{root, child},
		Commands: map[cli.CommandID]cli.CommandSchema{
			root: commandSchema,
			child: {
				ID:   child,
				Path: []string{"child"},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{entry: {
			ID: entry,
		}},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntrySchema{action: {
			ID: action,
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultSchema{def: {
			ID: def,
		}},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderSchema{provider: {
			ID: provider,
		}},
		Flags: map[cli.FlagID]cli.FlagSchema{flag: {
			ID: flag,
		}},
		Facts: map[cli.ValueID]cli.FactSchema{fact: {
			ID: fact,
		}},
		Values: map[cli.ValueID]cli.ValueSchema{value: {
			ID: value,
		}},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{form: {
			ID: form,
		}},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{policy: {
			ID: policy,
		}},
	}
	commandRef := cli.CommandReference{
		ID:               root,
		Path:             []string{},
		Children:         commandSchema.Children,
		CommandEntries:   commandSchema.CommandEntries,
		ActionEntries:    commandSchema.ActionEntries,
		CommandDefaults:  commandSchema.CommandDefaults,
		CommandProviders: commandSchema.CommandProviders,
		Flags:            commandSchema.Flags,
		Facts:            commandSchema.Facts,
		Values:           commandSchema.Values,
		ArgumentForms:    commandSchema.ArgumentForms,
		SignalPolicies:   commandSchema.SignalPolicies,
	}
	reference := cli.Reference{
		CommandOrder: []cli.CommandID{root, child},
		Commands: map[cli.CommandID]cli.CommandReference{
			root: commandRef,
			child: {
				ID:   child,
				Path: []string{"child"},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryReference{entry: {
			ID: entry,
		}},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntryReference{action: {
			ID: action,
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultReference{def: {
			ID: def,
		}},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderReference{provider: {
			ID: provider,
		}},
		Flags: map[cli.FlagID]cli.FlagReference{flag: {
			ID: flag,
		}},
		Facts: map[cli.ValueID]cli.FactReference{fact: {
			ID: fact,
		}},
		Values: map[cli.ValueID]cli.ValueReference{value: {
			ID: value,
		}},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormReference{form: {
			ID: form,
		}},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{policy: {
			ID: policy,
		}},
	}
	commandInspection := cli.CommandInspection{
		ID:                    root,
		Path:                  []string{},
		NamespaceCompleteness: cli.CommandNamespaceComplete,
		Children:              commandSchema.Children,
		CommandEntries:        commandSchema.CommandEntries,
		ActionEntries:         commandSchema.ActionEntries,
		CommandDefaults:       commandSchema.CommandDefaults,
		CommandProviders:      commandSchema.CommandProviders,
		Flags:                 commandSchema.Flags,
		Facts:                 commandSchema.Facts,
		Values:                commandSchema.Values,
		ArgumentForms:         commandSchema.ArgumentForms,
		SignalPolicies:        commandSchema.SignalPolicies,
	}
	inspection := cli.Inspection{
		CommandOrder: []cli.CommandID{root, child},
		Commands: map[cli.CommandID]cli.CommandInspection{
			root: commandInspection,
			child: {
				ID:                    child,
				Path:                  []string{"child"},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{entry: {
			ID: entry,
		}},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntryInspection{action: {
			ID: action,
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultInspection{def: {
			ID: def,
		}},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{provider: {
			ID: provider,
		}},
		Flags: map[cli.FlagID]cli.FlagInspection{flag: {
			ID: flag,
		}},
		Facts: map[cli.ValueID]cli.FactInspection{fact: {
			ID: fact,
		}},
		Values: map[cli.ValueID]cli.ValueInspection{value: {
			ID: value,
		}},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{form: {
			ID: form,
		}},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{policy: {
			ID: policy,
		}},
	}
	expectations := []DeclarationExpectation{
		CommandsExactly(RootCommand(), Command("child")),
		RootCommand().ChildrenExactly(Command("child")),
		RootCommand().EntriesExactly(CommandEntryByID(entry)),
		RootCommand().ActionEntriesExactly(ActionEntryByID(action)),
		RootCommand().DefaultsExactly(CommandDefaultByID(def)),
		RootCommand().ProvidersExactly(CommandProviderByID(provider)),
		RootCommand().FlagsExactly(FlagByID(flag)),
		RootCommand().FactsExactly(FactByID(fact)),
		RootCommand().ValuesExactly(ValueByID(value)),
		RootCommand().ArgumentFormsExactly(ArgumentFormByID(form)),
		RootCommand().SignalPoliciesExactly(SignalPolicyByID(policy)),
	}
	for _, expectation := range expectations {
		(SchemaResult{
			state: &schemaResultState{
				observation: *schema,
			},
		}).Require(t, expectation)
	}
	for _, expectation := range expectations {
		(ReferenceResult{
			state: &referenceResultState{
				observation: reference,
				err:         nil,
			},
		}).Require(t, expectation)
	}
	for _, expectation := range expectations {
		(InspectionResult{
			state: &inspectionResultState{
				observation: inspection,
				err:         nil,
			},
		}).Require(t, expectation)
	}
}

func TestDeclarationLocatorConstructorsPreserveTypedImmutableCoordinates(t *testing.T) {
	t.Parallel()

	root := RootCommand()
	canonical := root.CanonicalEntry()
	if canonical.state == nil || canonical.state.mode != commandEntryLocatorCanonical || canonical.state.command != root.state {
		t.Fatalf("canonical command-entry state = %#v", canonical.state)
	}
	shortcut := root.Shortcut("short")
	if shortcut.state == nil || shortcut.state.mode != commandEntryLocatorShortcut || shortcut.state.command != root.state || shortcut.state.name != "short" {
		t.Fatalf("shortcut command-entry state = %#v", shortcut.state)
	}
	action := root.ActionEntryAt(4)
	if action.state == nil || action.state.mode != actionEntryLocatorCommandIndex || action.state.command != root.state || action.state.index != 4 {
		t.Fatalf("action-entry state = %#v", action.state)
	}
	profile := root.GrammarProfile("prod")
	if profile.state == nil || profile.state.mode != grammarProfileLocatorValue || profile.state.command != root.state || profile.state.value != "prod" {
		t.Fatalf("grammar-profile state = %#v", profile.state)
	}
	flag := root.FlagAt(6)
	if flag.state == nil || flag.state.mode != flagLocatorIndex || flag.state.command != root.state || flag.state.index != 6 {
		t.Fatalf("flag state = %#v", flag.state)
	}
	spelling := flag.Spelling("--flag")
	if spelling.state == nil || spelling.state.mode != flagSpellingLocatorText || spelling.state.flag != flag.state || spelling.state.text != "--flag" {
		t.Fatalf("flag-spelling state = %#v", spelling.state)
	}
	form := root.ArgumentFormAt(7)
	if form.state == nil || form.state.mode != argumentFormLocatorCommandIndex || form.state.command != root.state || form.state.index != 7 {
		t.Fatalf("argument-form state = %#v", form.state)
	}
	term := form.Literal("fixed")
	if term.state == nil || term.state.mode != argumentTermLocatorFormLiteral || term.state.form != form.state || term.state.text != "fixed" {
		t.Fatalf("argument-term state = %#v", term.state)
	}
	environment := root.EnvironmentAt(8)
	if environment.state == nil || environment.state.mode != environmentInputLocatorCommandIndex || environment.state.command != root.state || environment.state.index != 8 {
		t.Fatalf("environment-input state = %#v", environment.state)
	}
	fact := root.FactAt(9)
	if fact.state == nil || fact.state.mode != factLocatorCommandIndex || fact.state.command != root.state || fact.state.index != 9 {
		t.Fatalf("fact state = %#v", fact.state)
	}
	value := root.ValueAt(10)
	if value.state == nil || value.state.mode != valueLocatorCommandIndex || value.state.command != root.state || value.state.index != 10 {
		t.Fatalf("value state = %#v", value.state)
	}
	domain := ValueByID(11).Domain(cli.ValueDomainUseDestination)
	entry := domain.EntryAt(12)
	if entry.state == nil || entry.state.mode != valueDomainEntryLocatorIndex || entry.state.domain != domain.state || entry.state.index != 12 {
		t.Fatalf("value-domain-entry state = %#v", entry.state)
	}
}

func TestDeclarationLocatorPredicatesPreserveSchemaTruth(t *testing.T) {
	t.Parallel()

	replacement := &cli.DeclarationReplacement{
		Target: cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: 999,
		},
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID:       1,
			Runnable: false,
		}},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{2: {
			ID:          2,
			Replacement: replacement,
		}},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntrySchema{
			3: {
				ID:          3,
				Hidden:      true,
				Replacement: replacement,
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{4: {
			ID:          4,
			Replacement: replacement,
		}},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{5: {
			ID:          5,
			Replacement: replacement,
		}},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{6: {
			ID:          6,
			Replacement: replacement,
		}},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
			7: {
				ID:          7,
				Hidden:      true,
				Replacement: replacement,
			},
		},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{8: {
			ID:          8,
			Replacement: replacement,
		}},
	}

	for _, expectation := range []SchemaExpectation{
		CommandByID(1001).DoesNotExist(),
		CommandEntryByID(1002).DoesNotExist(),
		ActionEntryByID(1003).DoesNotExist(),
		GrammarProfileByID(1004).DoesNotExist(),
		FlagByID(1005).DoesNotExist(),
		FlagSpellingByID(1006).DoesNotExist(),
		ArgumentFormByID(1007).DoesNotExist(),
		ArgumentTermByID(1008).DoesNotExist(),
		EnvironmentInputByID(1009).DoesNotExist(),
	} {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("missing %T = %+v", expectation, got)
		}
	}
	if got := schemaExpectationOutcome(t, CommandByID(1).NotRunnable(), schema); got.status != evaluationProved {
		t.Fatalf("NotRunnable = %+v", got)
	}
	for _, expectation := range []SchemaExpectation{
		ActionEntryByID(3).Exists(),
		ActionEntryByID(3).Hidden(),
		FlagSpellingByID(5).Exists(),
		ArgumentFormByID(6).Exists(),
		ArgumentTermByID(7).Exists(),
		ArgumentTermByID(7).Hidden(),
		EnvironmentInputByID(8).Exists(),
	} {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("present %T = %+v", expectation, got)
		}
	}
	for _, expectation := range []SchemaExpectation{
		CommandEntryByID(2).ReplacedBy(FlagByID(999)),
		ActionEntryByID(3).ReplacedBy(FlagByID(999)),
		FlagByID(4).ReplacedBy(FlagByID(999)),
		FlagSpellingByID(5).ReplacedBy(FlagByID(999)),
		ArgumentFormByID(6).ReplacedBy(FlagByID(999)),
		ArgumentTermByID(7).ReplacedBy(FlagByID(999)),
		EnvironmentInputByID(8).ReplacedBy(FlagByID(999)),
	} {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("replacement %T = %+v", expectation, got)
		}
	}
}

func TestEffectiveDeclarationPolarityPredicatesPreserveObservationTruth(t *testing.T) {
	t.Parallel()

	reference := cli.Reference{
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryReference{
			1: {
				ID:        1,
				Available: true,
				Visible:   true,
			},
			2: {
				ID:        2,
				Available: false,
				Visible:   false,
			},
		},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntryReference{
			3: {
				ID:        3,
				Available: true,
				Visible:   true,
			},
			4: {
				ID:        4,
				Available: false,
				Visible:   false,
			},
		},
		Flags: map[cli.FlagID]cli.FlagReference{
			5: {
				ID:        5,
				Available: true,
				Visible:   true,
			},
			6: {
				ID:        6,
				Available: false,
				Visible:   false,
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingReference{
			7: {
				ID:        7,
				Available: true,
				Visible:   true,
			},
			8: {
				ID:        8,
				Available: false,
				Visible:   false,
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormReference{
			9: {
				ID:        9,
				Available: true,
				Visible:   true,
			},
			10: {
				ID:        10,
				Available: false,
				Visible:   false,
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermReference{
			11: {
				ID:        11,
				Available: true,
				Visible:   true,
			},
			12: {
				ID:        12,
				Available: false,
				Visible:   false,
			},
		},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputReference{
			13: {
				EnvironmentInputSchema: cli.EnvironmentInputSchema{
					ID: 13,
				},
				Available: true,
			},
			14: {
				EnvironmentInputSchema: cli.EnvironmentInputSchema{
					ID: 14,
				},
				Available: false,
			},
		},
	}
	inspection := cli.Inspection{
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{
			1: {
				ID:        1,
				Available: true,
				Visible:   true,
			},
			2: {
				ID:        2,
				Available: false,
				Visible:   false,
			},
		},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntryInspection{
			3: {
				ID:        3,
				Available: true,
				Visible:   true,
			},
			4: {
				ID:        4,
				Available: false,
				Visible:   false,
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			5: {
				ID:        5,
				Available: true,
				Visible:   true,
				Present:   true,
			},
			6: {
				ID:        6,
				Available: false,
				Visible:   false,
				Present:   false,
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			7: {
				ID:        7,
				Available: true,
				Visible:   true,
				Present:   true,
			},
			8: {
				ID:        8,
				Available: false,
				Visible:   false,
				Present:   false,
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
			9: {
				ID:        9,
				Available: true,
				Visible:   true,
			},
			10: {
				ID:        10,
				Available: false,
				Visible:   false,
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermInspection{
			11: {
				ID:        11,
				Available: true,
				Visible:   true,
				Present:   true,
			},
			12: {
				ID:        12,
				Available: false,
				Visible:   false,
				Present:   false,
			},
		},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputInspection{
			13: {
				EnvironmentInputSchema: cli.EnvironmentInputSchema{
					ID: 13,
				},
				Available: true,
			},
			14: {
				EnvironmentInputSchema: cli.EnvironmentInputSchema{
					ID: 14,
				},
				Available: false,
			},
		},
	}

	referenceExpectations := []ReferenceExpectation{
		CommandEntryByID(
			1,
		).Available(),
		CommandEntryByID(2).Unavailable(),
		CommandEntryByID(1).Visible(),
		CommandEntryByID(2).NotVisible(),
		ActionEntryByID(
			3,
		).Available(),
		ActionEntryByID(4).Unavailable(),
		ActionEntryByID(3).Visible(),
		ActionEntryByID(4).NotVisible(),
		FlagByID(5).Available(),
		FlagByID(6).Unavailable(),
		FlagByID(5).Visible(),
		FlagByID(6).NotVisible(),
		FlagSpellingByID(
			7,
		).Available(),
		FlagSpellingByID(8).Unavailable(),
		FlagSpellingByID(7).Visible(),
		FlagSpellingByID(8).NotVisible(),
		ArgumentFormByID(
			9,
		).Available(),
		ArgumentFormByID(10).Unavailable(),
		ArgumentFormByID(9).Visible(),
		ArgumentFormByID(10).NotVisible(),
		ArgumentTermByID(
			11,
		).Available(),
		ArgumentTermByID(12).Unavailable(),
		ArgumentTermByID(11).Visible(),
		ArgumentTermByID(12).NotVisible(),
		EnvironmentInputByID(13).Available(),
		EnvironmentInputByID(14).Unavailable(),
	}
	for _, expectation := range referenceExpectations {
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference %T = %+v", expectation, got)
		}
	}
	for _, expectation := range []InspectionExpectation{
		CommandEntryByID(
			1,
		).Available(), CommandEntryByID(2).Unavailable(), CommandEntryByID(1).Visible(), CommandEntryByID(2).NotVisible(),
		ActionEntryByID(
			3,
		).Available(), ActionEntryByID(4).Unavailable(), ActionEntryByID(3).Visible(), ActionEntryByID(4).NotVisible(),
		FlagByID(5).Available(), FlagByID(6).Unavailable(), FlagByID(5).Visible(), FlagByID(6).NotVisible(),
		FlagSpellingByID(
			7,
		).Available(), FlagSpellingByID(8).Unavailable(), FlagSpellingByID(7).Visible(), FlagSpellingByID(8).NotVisible(),
		ArgumentFormByID(9).Available(), ArgumentFormByID(
			10,
		).Unavailable(), ArgumentFormByID(9).Visible(), ArgumentFormByID(10).NotVisible(),
		ArgumentTermByID(
			11,
		).Available(), ArgumentTermByID(12).Unavailable(), ArgumentTermByID(11).Visible(), ArgumentTermByID(12).NotVisible(),
		EnvironmentInputByID(13).Available(), EnvironmentInputByID(14).Unavailable(),
		FlagByID(5).Present(), FlagByID(6).Absent(),
		FlagSpellingByID(7).Present(), FlagSpellingByID(8).Absent(),
		ArgumentTermByID(11).Present(), ArgumentTermByID(12).Absent(),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("inspection %T = %+v", expectation, got)
		}
	}
}

func TestSelectionNegationPredicatesPreserveInspectionTruth(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:      1,
				Path:    []string{"root"},
				Profile: 99,
				ArgumentFormSelection: cli.ArgumentFormSelection{
					Selected: 0,
				},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
			6: {
				ID:      6,
				Command: 1,
				Outcome: cli.ArgumentFormOutcomeNoMatch,
			},
		},
	}
	for _, expectation := range []InspectionExpectation{
		CommandByID(1).NotSelected(),
		CommandEntryByID(2).NotSelected(),
		InvocationAction(cli.InvocationActionHelp).NotSelected(),
		ActionEntryByID(3).NotSelected(),
		GrammarProfileByID(4).NotSelected(),
		ArgumentFormByID(6).NotSelected(),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("not-selected %T = %+v", expectation, got)
		}
	}
}

func TestCommandInspectionPolarityPredicatesPreserveObservationTruth(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:        1,
				Available: true,
				Visible:   true,
			},
			2: {
				ID:        2,
				Available: false,
				Visible:   false,
			},
		},
	}
	for _, expectation := range []InspectionExpectation{
		CommandByID(1).Available(),
		CommandByID(2).Unavailable(),
		CommandByID(1).Visible(),
		CommandByID(2).NotVisible(),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("command inspection %T = %+v", expectation, got)
		}
	}
}
