package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestStaticDeclarationExpectationsUsePublicArtifacts(t *testing.T) {
	t.Parallel()
	command := cli.CommandID(1)
	target := cli.CommandID(2)
	entry := cli.CommandEntryID(3)
	action := cli.ActionEntryID(4)
	flag := cli.FlagID(5)
	spelling := cli.FlagSpellingID(6)
	form := cli.ArgumentFormID(7)
	term := cli.ArgumentTermID(8)
	env := cli.EnvironmentInputID(9)
	fact := cli.ValueID(10)
	replacement := &cli.DeclarationReplacement{
		Target: cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: cli.FlagID(99),
		},
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{target: {
			ID:   target,
			Path: []string{"target"},
		}},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{
			entry: {
				ID:             entry,
				Kind:           cli.CommandEntryAlias,
				TargetPath:     []cli.CommandID{target},
				Hidden:         true,
				LifecycleState: cli.LifecycleStateDeprecated,
			},
		},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntrySchema{
			action: {
				ID:             action,
				Kind:           cli.ActionEntryFlag,
				Hidden:         true,
				LifecycleState: cli.LifecycleStateDeprecated,
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{
			flag: {
				ID:             flag,
				Required:       true,
				Hidden:         true,
				Sensitive:      true,
				LifecycleState: cli.LifecycleStateDeprecated,
				Replacement:    replacement,
				Spellings:      []cli.FlagSpellingID{spelling},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
			spelling: {
				ID: spelling,
				Spelling: cli.FlagSpelling{
					Kind: cli.FlagSpellingLong,
				},
				LifecycleState: cli.LifecycleStateCurrent,
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{
			form: {
				ID:             form,
				Terms:          []cli.ArgumentTermID{term},
				LifecycleState: cli.LifecycleStateCurrent,
				Reachability:   cli.ReachabilityReachable,
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
			term: {
				ID:             term,
				Kind:           cli.ArgumentTermValue,
				Hidden:         true,
				Sensitive:      true,
				LifecycleState: cli.LifecycleStateCurrent,
			},
		},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{
			env: {
				ID:             env,
				Sensitive:      true,
				LifecycleState: cli.LifecycleStateCurrent,
			},
		},
		Facts: map[cli.ValueID]cli.FactSchema{fact: {
			ID:          fact,
			HasResolver: true,
		}},
	}
	exps := []SchemaExpectation{
		CommandEntryByID(entry).Exists(),
		CommandEntryByID(entry).Kind(cli.CommandEntryAlias),
		CommandEntryByID(entry).Hidden(),
		CommandEntryByID(entry).Targets(CommandByID(target)),
		ActionEntryByID(action).Kind(cli.ActionEntryFlag),
		FlagByID(flag).Required(),
		FlagByID(flag).Sensitive(),
		FlagByID(flag).ReplacedBy(FlagByID(99)),
		FlagByID(flag).SpellingsExactly(FlagSpellingByID(spelling)),
		FlagSpellingByID(spelling).Kind(cli.FlagSpellingLong),
		ArgumentFormByID(form).TermsExactly(ArgumentTermByID(term)),
		ArgumentFormByID(form).Reachability(cli.ReachabilityReachable),
		ArgumentTermByID(term).Kind(cli.ArgumentTermValue),
		ArgumentTermByID(term).Sensitive(),
		EnvironmentInputByID(env).Sensitive(),
		FactByID(fact).HasResolver(),
	}
	(SchemaResult{
		state: &schemaResultState{
			observation: *schema,
		},
	}).Require(t, exps...)

	_ = command
}

func TestStaticDeclarationEffectiveStateUsesReferenceAndInspectionOwners(t *testing.T) {
	t.Parallel()
	flagID := cli.FlagID(41)
	factID := cli.ValueID(42)
	replacement := &cli.DeclarationReplacement{
		Target: cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: 99,
		},
	}
	flagSchema := cli.FlagSchema{
		ID:             flagID,
		Hidden:         true,
		Sensitive:      true,
		LifecycleState: cli.LifecycleStateDeprecated,
		Replacement:    replacement,
	}
	schema := cli.Schema{
		Flags: map[cli.FlagID]cli.FlagSchema{flagID: flagSchema},
	}
	reference := cli.Reference{
		Flags: map[cli.FlagID]cli.FlagReference{
			flagID: {
				ID:             flagID,
				Hidden:         true,
				Sensitive:      true,
				LifecycleState: cli.LifecycleStateDeprecated,
				Replacement:    replacement,
				Available:      true,
				Visible:        false,
			},
		},
		Facts: map[cli.ValueID]cli.FactReference{
			factID: {
				FactSchema: cli.FactSchema{
					ID: factID,
				},
				Available: true,
			},
		},
	}
	inspection := cli.Inspection{
		Flags: map[cli.FlagID]cli.FlagInspection{
			flagID: {
				ID:             flagID,
				Hidden:         true,
				Sensitive:      true,
				LifecycleState: cli.LifecycleStateDeprecated,
				Replacement:    replacement,
				Available:      true,
				Visible:        false,
				Present:        true,
			},
		},
	}

	flag := FlagByID(flagID)
	(SchemaResult{
		state: &schemaResultState{
			observation: schema,
		},
	}).Require(t,
		flag.Exists(),
		flag.Hidden(),
		flag.Sensitive(),
		flag.Lifecycle(cli.LifecycleStateDeprecated),
		flag.ReplacedBy(FlagByID(99)))
	(ReferenceResult{
		state: &referenceResultState{
			observation: reference,
			err:         nil,
		},
	}).Require(t,
		flag.Exists(),
		flag.Hidden(),
		flag.Sensitive(),
		flag.Lifecycle(cli.LifecycleStateDeprecated),
		flag.ReplacedBy(FlagByID(99)),
		flag.Available(),
		flag.NotVisible(),
		FactByID(factID).Available())
	(InspectionResult{
		state: &inspectionResultState{
			observation: inspection,
			err:         nil,
		},
	}).Require(t,
		flag.Exists(),
		flag.Hidden(),
		flag.Sensitive(),
		flag.Lifecycle(cli.LifecycleStateDeprecated),
		flag.ReplacedBy(FlagByID(99)),
		flag.Available(),
		flag.NotVisible(),
		flag.Present())
}

func TestCommandStaticPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	expectation := CommandByID(commandID).Hidden()
	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{commandID: {
			ID:     commandID,
			Hidden: true,
		}},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{commandID: {
			ID:     commandID,
			Hidden: false,
		}},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{commandID: {
			ID:     commandID,
			Hidden: true,
		}},
	}
	if got := schemaExpectationOutcome(t, expectation, &schema); got.status != evaluationProved {
		t.Fatalf("schema property = %#v", got)
	}
	if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationDisproved {
		t.Fatalf("reference mismatch = %#v", got)
	}
	if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
		t.Fatalf("inspection property = %#v", got)
	}
}

func TestSchemaCommandResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	rootID, childID := cli.CommandID(1), cli.CommandID(2)
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			rootID: {
				ID:       rootID,
				Path:     []string{},
				Children: []cli.CommandID{childID},
			},
			childID: {
				ID:   childID,
				Name: "child",
				Path: []string{"child"},
			},
		},
	}
	t.Run("command child name: semantic result", func(t *testing.T) {
		t.Parallel()
		if got, ok, _ := resolveSchemaCommand(RootCommand().Command("child").state, schema); !ok || got.ID != childID {
			t.Fatalf("child command = %#v, %v", got, ok)
		}
		if _, ok, _ := resolveSchemaCommand(RootCommand().Command("missing").state, schema); ok {
			t.Fatal("missing child unexpectedly resolved")
		}
	})
	t.Run("command path: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		if _, ok, _ := resolveSchemaCommand((*commandLocatorState)(nil), schema); ok {
			t.Fatal("nil locator unexpectedly resolved")
		}
		if _, ok, _ := resolveSchemaCommand(Command("child").state, schema); !ok {
			t.Fatal("unique path did not resolve")
		}
		ambiguous := &cli.Schema{
			Commands: map[cli.CommandID]cli.CommandSchema{
				1: {
					ID:   1,
					Path: []string{"dup"},
				},
				2: {
					ID:   2,
					Path: []string{"dup"},
				},
			},
		}
		if _, ok, _ := resolveSchemaCommand(Command("dup").state, ambiguous); ok {
			t.Fatal("ambiguous path unexpectedly resolved")
		}
	})
}

func TestReferenceCommandResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	rootID, childID := cli.CommandID(1), cli.CommandID(2)
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			rootID: {
				ID:       rootID,
				Path:     []string{},
				Children: []cli.CommandID{childID},
			},
			childID: {
				ID:   childID,
				Name: "child",
				Path: []string{"child"},
			},
		},
	}
	t.Run("command child name: semantic result", func(t *testing.T) {
		t.Parallel()
		if got, ok, _ := resolveReferenceCommand(
			RootCommand().Command("child").state,
			reference,
		); !ok ||
			got.ID != childID {
			t.Fatalf("child command = %#v, %v", got, ok)
		}
		if _, ok, _ := resolveReferenceCommand(RootCommand().Command("missing").state, reference); ok {
			t.Fatal("missing child unexpectedly resolved")
		}
	})
	t.Run("command path: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		if _, ok, _ := resolveReferenceCommand((*commandLocatorState)(nil), reference); ok {
			t.Fatal("nil locator unexpectedly resolved")
		}
		if _, ok, _ := resolveReferenceCommand(Command("child").state, reference); !ok {
			t.Fatal("unique path did not resolve")
		}
		ambiguous := cli.Reference{
			Commands: map[cli.CommandID]cli.CommandReference{
				1: {
					ID:   1,
					Path: []string{"dup"},
				},
				2: {
					ID:   2,
					Path: []string{"dup"},
				},
			},
		}
		if _, ok, _ := resolveReferenceCommand(Command("dup").state, ambiguous); ok {
			t.Fatal("ambiguous path unexpectedly resolved")
		}
	})
}

func TestCommandInspectionPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	expectation := commandAvailabilityExpectation(CommandByID(commandID), true)
	t.Run("commandinspectionproperty unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		if got := inspectionExpectationOutcome(t, expectation, cli.Inspection{}); got.status != evaluationNotProvable {
			t.Fatalf("missing command = %#v", got)
		}
		if got := inspectionExpectationOutcome(t, expectation, cli.Inspection{
			Truncated: true,
		}); got.status != evaluationNotProvable {
			t.Fatalf("truncated command = %#v", got)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{commandID: {
				ID:        commandID,
				Available: false,
			}},
		}
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationDisproved {
			t.Fatalf("availability mismatch = %#v", got)
		}
		inspection.Commands[commandID] = cli.CommandInspection{
			ID:        commandID,
			Available: true,
		}
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("availability match = %#v", got)
		}
	})
}

func requireStaticValueSchemaProperty(t *testing.T, expectation StaticDeclarationExpectation, value cli.ValueSchema, label string) {
	t.Helper()
	schema := cli.Schema{
		Values: map[cli.ValueID]cli.ValueSchema{value.ID: value},
	}
	reference := cli.Reference{
		Values: map[cli.ValueID]cli.ValueReference{
			value.ID: {
				ValueSchema: value,
			},
		},
	}
	if got := schemaExpectationOutcome(t, expectation, &schema); got.status != evaluationProved {
		t.Fatalf("schema %s = %#v", label, got)
	}
	if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
		t.Fatalf("reference %s = %#v", label, got)
	}
	t.Run("absent declaration: semantic absence", func(t *testing.T) {
		t.Parallel()
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationDisproved {
			t.Fatalf("absent schema value = %#v", got)
		}
		if got := referenceExpectationOutcome(t, expectation, cli.Reference{}); got.status != evaluationDisproved {
			t.Fatalf("absent reference value = %#v", got)
		}
	})
}

func TestValueSensitiveDeclarationPreservesSensitivity(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(1)
	expectation := ValueByID(valueID).DeclaredSensitive()
	value := cli.ValueSchema{
		ID:        valueID,
		Sensitive: true,
	}
	requireStaticValueSchemaProperty(t, expectation, value, "sensitive")
}

func TestSchemaValueResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(3)
	schema := &cli.Schema{
		Values: map[cli.ValueID]cli.ValueSchema{valueID: {
			ID: valueID,
		}},
	}
	resolved := resolveSchemaValueLocator(ValueByID(valueID).state, schema)
	if resolved.status != locatorResolutionFound || resolved.value.ID != valueID {
		t.Fatalf("value = %#v, %v", resolved.value, resolved.status)
	}
	invalid := resolveSchemaValueLocator(ValueLocator{}.state, schema)
	if invalid.status != locatorResolutionInvalid {
		t.Fatalf("zero value locator = %v", invalid.status)
	}
}

func TestReferenceValueResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(3)
	reference := cli.Reference{
		Values: map[cli.ValueID]cli.ValueReference{valueID: {ValueSchema: cli.ValueSchema{
			ID: valueID,
		}}},
	}
	resolved := resolveReferenceValueLocator(ValueByID(valueID).state, reference)
	if resolved.status != locatorResolutionFound || resolved.value.ID != valueID {
		t.Fatalf("value = %#v, %v", resolved.value, resolved.status)
	}
	invalid := resolveReferenceValueLocator(ValueLocator{}.state, reference)
	if invalid.status != locatorResolutionInvalid {
		t.Fatalf("zero value locator = %v", invalid.status)
	}
}

func TestValueRequiredPreservesSemanticStatus(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(1)
	expectation := ValueByID(valueID).Required()
	value := cli.ValueSchema{
		ID:       valueID,
		Required: true,
	}
	requireStaticValueSchemaProperty(t, expectation, value, "required")
}

func TestValueReferencePropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(1)
	expectation := ValueByID(valueID).Available()
	reference := cli.Reference{
		Values: map[cli.ValueID]cli.ValueReference{valueID: {
			ValueSchema: cli.ValueSchema{
				ID: valueID,
			},
			Available: false,
		}},
	}
	if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationDisproved {
		t.Fatalf("availability mismatch = %#v", got)
	}
	reference.Values[valueID] = cli.ValueReference{
		ID:        valueID,
		Available: true,
	}
	if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
		t.Fatalf("availability match = %#v", got)
	}
	if got := referenceExpectationOutcome(t, expectation, cli.Reference{}); got.status != evaluationDisproved {
		t.Fatalf("missing value = %#v", got)
	}
}

func TestCommandResolutionInternalStateFailsClosed(t *testing.T) {
	t.Parallel()

	if _, found, _ := resolveSchemaCommand(&commandLocatorState{}, &cli.Schema{}); found {
		t.Fatal("schema resolver accepted an unsupported command locator kind")
	}
	if _, found, _ := resolveReferenceCommand((*commandLocatorState)(nil), cli.Reference{}); found {
		t.Fatal("reference resolver accepted a nil command locator")
	}
	if _, found, _ := resolveReferenceCommand(&commandLocatorState{}, cli.Reference{}); found {
		t.Fatal("reference resolver accepted an unsupported command locator kind")
	}
}

func TestCommandStaticResolutionRejectsMissingReferenceAndMissingChildParents(t *testing.T) {
	t.Parallel()
	expectation := CommandByID(1).Hidden()
	if got := referenceExpectationOutcome(t, expectation, cli.Reference{}); got.status != evaluationDisproved {
		t.Fatalf("missing reference command = %#v", got)
	}

	child := CommandByID(99).Command("child").state
	if _, ok, _ := resolveSchemaCommand(child, &cli.Schema{}); ok {
		t.Fatal("schema child resolved without parent")
	}
	if _, ok, _ := resolveReferenceCommand(child, cli.Reference{}); ok {
		t.Fatal("reference child resolved without parent")
	}
}
