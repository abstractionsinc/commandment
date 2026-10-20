package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestTypedDeclarationLocatorLabels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  evaluationField
		want string
	}{
		{"command entry", commandEntryLocatorLabel(CommandEntryByID(1).state), "command entry id 1"},
		{"action entry", actionEntryLocatorLabel(ActionEntryByID(2).state), "action entry id 2"},
		{"flag", flagLocatorLabel(FlagByID(3).state), "flag id 3"},
		{"flag spelling", flagSpellingLocatorLabel(FlagSpellingByID(4).state), "flag spelling id 4"},
		{"argument form", argumentFormLocatorLabel(ArgumentFormByID(5).state), "argument form id 5"},
		{"argument term", argumentTermLocatorLabel(ArgumentTermByID(6).state), "argument term id 6"},
		{"environment", environmentInputLocatorLabel(EnvironmentInputByID(7).state), "environment input id 7"},
		{"fact", factLocatorLabel(FactByID(8).state), "fact value id 8"},
		{"value", valueLocatorLabel(ValueByID(9).state), "value id 9"},
		{"value domain entry", valueDomainEntryLocatorLabel(ValueDomainEntryByID(10).state), "value-domain entry id 10"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := renderEvaluationField(tc.got); got != tc.want {
				t.Fatalf("label = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTypedFlagResolutionPreservesClosure(t *testing.T) {
	t.Parallel()

	commandID := cli.CommandID(1)
	firstFlagID := cli.FlagID(2)
	secondFlagID := cli.FlagID(3)
	firstSpellingID := cli.FlagSpellingID(4)
	secondSpellingID := cli.FlagSpellingID(5)

	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			commandID: {ID: commandID, Flags: []cli.FlagID{firstFlagID, secondFlagID}},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{
			firstFlagID:  {ID: firstFlagID, Command: commandID, Spellings: []cli.FlagSpellingID{firstSpellingID}},
			secondFlagID: {ID: secondFlagID, Command: commandID, Spellings: []cli.FlagSpellingID{secondSpellingID}},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
			firstSpellingID:  {ID: firstSpellingID, Flag: firstFlagID, Spelling: cli.FlagSpelling{Text: "--mode"}},
			secondSpellingID: {ID: secondSpellingID, Flag: secondFlagID, Spelling: cli.FlagSpelling{Text: "--mode"}},
		},
	}

	if resolution := resolveSchemaFlagLocator(FlagByID(firstFlagID).state, schema); resolution.status != locatorResolutionFound || resolution.value.ID != firstFlagID {
		t.Fatalf("by ID resolution = %#v", resolution)
	}
	if resolution := resolveSchemaFlagLocator(RootCommand().Flag("--mode").state, schema); resolution.status != locatorResolutionAmbiguous {
		t.Fatalf("ambiguous human coordinate = %#v", resolution)
	}
	if resolution := resolveSchemaFlagLocator(RootCommand().FlagAt(-1).state, schema); resolution.status != locatorResolutionInvalid {
		t.Fatalf("negative schema index = %#v", resolution)
	}

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {ID: commandID, Flags: []cli.FlagID{firstFlagID}, NamespaceCompleteness: cli.CommandNamespaceIncomplete},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			firstFlagID: {ID: firstFlagID, Command: commandID, Spellings: []cli.FlagSpellingID{firstSpellingID}},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			firstSpellingID: {ID: firstSpellingID, Flag: firstFlagID, Spelling: cli.FlagSpelling{Text: "--known"}},
		},
	}
	if resolution := resolveInspectionFlagLocator(RootCommand().Flag("--missing").state, inspection); resolution.status != locatorResolutionNotProvable {
		t.Fatalf("incomplete inspection absence = %#v", resolution)
	}
	inspection.Commands[commandID] = cli.CommandInspection{ID: commandID, Flags: []cli.FlagID{firstFlagID}, NamespaceCompleteness: cli.CommandNamespaceComplete}
	if resolution := resolveInspectionFlagLocator(RootCommand().Flag("--missing").state, inspection); resolution.status != locatorResolutionAbsent {
		t.Fatalf("closed inspection absence = %#v", resolution)
	}
}

func TestTypedDeclarationExistenceTruthTable(t *testing.T) {
	t.Parallel()

	flagID := cli.FlagID(1)
	schema := cli.Schema{Flags: map[cli.FlagID]cli.FlagSchema{flagID: {ID: flagID}}}

	exists := requireSchemaEvaluator(t, FlagByID(flagID).Exists())
	absent := requireSchemaEvaluator(t, FlagByID(flagID).DoesNotExist())
	missing := requireSchemaEvaluator(t, FlagByID(9).Exists())
	invalidAbsent := requireSchemaEvaluator(t, (FlagLocator{}).DoesNotExist())

	if got := exists.evaluateSchema(&schema); got.status != evaluationProved {
		t.Fatalf("found + exists = %#v", got)
	}
	if got := absent.evaluateSchema(&schema); got.status != evaluationDisproved {
		t.Fatalf("found + absent = %#v", got)
	}
	if got := missing.evaluateSchema(&schema); got.status != evaluationDisproved {
		t.Fatalf("absent + exists = %#v", got)
	}
	if got := requireSchemaEvaluator(t, FlagByID(9).DoesNotExist()).evaluateSchema(&schema); got.status != evaluationProved {
		t.Fatalf("absent + absent = %#v", got)
	}
	if got := invalidAbsent.evaluateSchema(&schema); got.status != evaluationDisproved {
		t.Fatalf("invalid locator + absent = %#v", got)
	}
}

func TestTypedDeclarationInspectionAbsenceRequiresClosure(t *testing.T) {
	t.Parallel()

	flagID := cli.FlagID(7)
	expectation := requireInspectionEvaluator(t, FlagByID(flagID).DoesNotExist())

	if got := expectation.evaluateInspection(cli.Inspection{Truncated: true}, nil); got.status != evaluationNotProvable {
		t.Fatalf("truncated absence = %#v", got)
	}
	if got := expectation.evaluateInspection(cli.Inspection{}, nil); got.status != evaluationProved {
		t.Fatalf("closed absence = %#v", got)
	}
}

func TestTypedCommandEntryResolutionPreservesLocalClosure(t *testing.T) {
	t.Parallel()

	commandID := cli.CommandID(1)
	canonicalID := cli.CommandEntryID(10)
	aliasID := cli.CommandEntryID(11)
	locator := RootCommand().Alias("serve")

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				CanonicalEntry:        canonicalID,
				CommandEntries:        []cli.CommandEntryID{canonicalID, aliasID},
				NamespaceCompleteness: cli.CommandNamespaceIncomplete,
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{
			canonicalID: {ID: canonicalID, Parent: commandID, Kind: cli.CommandEntryCanonical, Name: "root"},
			aliasID:     {ID: aliasID, Parent: commandID, Kind: cli.CommandEntryAlias, Name: "serve"},
		},
	}
	if resolution := resolveInspectionCommandEntryLocator(locator.state, inspection); resolution.status != locatorResolutionFound || resolution.value.ID != aliasID {
		t.Fatalf("positive partial resolution = %#v", resolution)
	}

	delete(inspection.CommandEntries, aliasID)
	if resolution := resolveInspectionCommandEntryLocator(locator.state, inspection); resolution.status != locatorResolutionNotProvable {
		t.Fatalf("missing detail in incomplete namespace = %#v", resolution)
	}

	inspection.Commands[commandID] = cli.CommandInspection{
		ID:                    commandID,
		CanonicalEntry:        canonicalID,
		CommandEntries:        []cli.CommandEntryID{canonicalID},
		NamespaceCompleteness: cli.CommandNamespaceComplete,
	}
	if resolution := resolveInspectionCommandEntryLocator(locator.state, inspection); resolution.status != locatorResolutionAbsent {
		t.Fatalf("closed alias absence = %#v", resolution)
	}
}

func TestTypedDeclarationIdentityResolution(t *testing.T) {
	t.Parallel()

	flagID := cli.FlagID(2)
	entryID := cli.CommandEntryID(3)
	schema := &cli.Schema{
		Flags:          map[cli.FlagID]cli.FlagSchema{flagID: {ID: flagID}},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{entryID: {ID: entryID}},
	}

	cases := []struct {
		name string
		loc  DeclarationIdentityLocator
		want cli.DeclarationIdentity
	}{
		{"flag", FlagByID(flagID), cli.DeclarationIdentity{Kind: cli.DeclarationKindFlag, Flag: flagID}},
		{"command entry", CommandEntryByID(entryID), cli.DeclarationIdentity{Kind: cli.DeclarationKindCommandEntry, CommandEntry: entryID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, status, detail := resolveExpectedDeclarationIdentity(tc.loc, schema, nil, cli.Inspection{})
			if status != locatorResolutionFound || got != tc.want || !emptyEvaluationField(detail) {
				t.Fatalf("identity = %#v, %v, %q; want %#v", got, status, detail, tc.want)
			}
		})
	}
}

func TestTypedDeclarationMetadataExpectations(t *testing.T) {
	t.Parallel()

	flagID := cli.FlagID(1)
	schema := &cli.Schema{Flags: map[cli.FlagID]cli.FlagSchema{
		flagID: {
			ID:             flagID,
			Origin:         cli.DeclarationOrigin(0),
			Hidden:         true,
			Sensitive:      true,
			Required:       true,
			LifecycleState: cli.LifecycleState(0),
		},
	}}

	expectations := []SchemaExpectation{
		FlagByID(flagID).Exists(),
		FlagByID(flagID).Hidden(),
		FlagByID(flagID).Sensitive(),
		FlagByID(flagID).Required(),
	}
	for i, expectation := range expectations {
		evaluator, ok := expectation.(schemaEvaluator)
		if !ok {
			t.Fatalf("expectation %d lacks schema evaluator", i)
		}
		if got := evaluator.evaluateSchema(schema); got.status != evaluationProved {
			t.Fatalf("expectation %d = %#v", i, got)
		}
	}
}

func TestTypedEffectiveDeclarationExpectations(t *testing.T) {
	t.Parallel()

	flagID := cli.FlagID(1)
	reference := cli.Reference{Flags: map[cli.FlagID]cli.FlagReference{
		flagID: {ID: flagID, Available: true, Visible: false},
	}}

	available := requireReferenceEvaluator(t, FlagByID(flagID).Available())
	notVisible := requireReferenceEvaluator(t, FlagByID(flagID).NotVisible())
	if got := available.evaluateReference(reference, nil); got.status != evaluationProved {
		t.Fatalf("available = %#v", got)
	}
	if got := notVisible.evaluateReference(reference, nil); got.status != evaluationProved {
		t.Fatalf("not visible = %#v", got)
	}
}
