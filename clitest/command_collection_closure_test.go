package clitest

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestInspectionCommandCollectionsRequireRelevantNamespaceClosure(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	serveID := cli.CommandID(2)
	entryID := cli.CommandEntryID(10)

	closed := cli.Inspection{
		CommandOrder: []cli.CommandID{rootID, serveID},
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				Children:              []cli.CommandID{serveID},
				CommandEntries:        []cli.CommandEntryID{entryID},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
			serveID: {
				ID:                    serveID,
				Name:                  "serve",
				Path:                  []string{"serve"},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{
			entryID: {
				ID:     entryID,
				Parent: rootID,
				Name:   "serve",
			},
		},
	}
	root := RootCommand()
	serve := root.Command("serve")

	for _, expectation := range []InspectionExpectation{
		CommandsExactly(root, serve),
		root.ChildrenExactly(serve),
		root.EntriesExactly(CommandEntryByID(entryID)),
	} {
		if outcome := inspectionExpectationOutcome(t, expectation, closed); outcome.status != evaluationProved {
			t.Fatalf("closed %T = %#v, want proved", expectation, outcome)
		}
	}

	incomplete := closed
	incomplete.Commands = make(map[cli.CommandID]cli.CommandInspection, len(closed.Commands))
	maps.Copy(incomplete.Commands, closed.Commands)
	rootInspection := incomplete.Commands[rootID]
	rootInspection.NamespaceCompleteness = cli.CommandNamespaceIncomplete
	incomplete.Commands[rootID] = rootInspection

	for _, expectation := range []InspectionExpectation{
		CommandsExactly(root, serve),
		root.ChildrenExactly(serve),
		root.EntriesExactly(CommandEntryByID(entryID)),
	} {
		if outcome := inspectionExpectationOutcome(
			t,
			expectation,
			incomplete,
		); outcome.status != evaluationNotProvable {
			t.Fatalf("incomplete %T = %#v, want not provable", expectation, outcome)
		}
	}
}

func TestCommandsExactlyRequiresExactlyOneObservableRoot(t *testing.T) {
	t.Parallel()

	if outcome := inspectionExpectationOutcome(
		t,
		CommandsExactly(),
		cli.Inspection{},
	); outcome.status != evaluationNotProvable {
		t.Fatalf("zero-root CommandsExactly = %#v, want not provable", outcome)
	}

	firstID := cli.CommandID(1)
	secondID := cli.CommandID(2)
	multipleRoots := cli.Inspection{
		CommandOrder: []cli.CommandID{firstID, secondID},
		Commands: map[cli.CommandID]cli.CommandInspection{
			firstID: {
				ID:                    firstID,
				Path:                  []string{},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
			secondID: {
				ID:                    secondID,
				Path:                  []string{},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
	}
	if outcome := inspectionExpectationOutcome(
		t,
		CommandsExactly(CommandByID(firstID), CommandByID(secondID)),
		multipleRoots,
	); outcome.status != evaluationNotProvable {
		t.Fatalf("multi-root CommandsExactly = %#v, want not provable", outcome)
	}

	oneRoot := cli.Inspection{
		CommandOrder: []cli.CommandID{firstID},
		Commands: map[cli.CommandID]cli.CommandInspection{
			firstID: {
				ID:                    firstID,
				Path:                  []string{},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
	}
	if outcome := inspectionExpectationOutcome(
		t,
		CommandsExactly(CommandByID(firstID)),
		oneRoot,
	); outcome.status != evaluationProved {
		t.Fatalf("one-root CommandsExactly = %#v, want proved", outcome)
	}
}

func TestInspectionCommandEntryLookupRequiresNamespaceClosureForExtensibleEntries(t *testing.T) {
	t.Parallel()

	rootID := cli.CommandID(1)
	canonicalID := cli.CommandEntryID(10)
	aliasID := cli.CommandEntryID(11)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				CanonicalEntry:        canonicalID,
				CommandEntries:        []cli.CommandEntryID{canonicalID, aliasID},
				NamespaceCompleteness: cli.CommandNamespaceIncomplete,
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{
			canonicalID: {
				ID:     canonicalID,
				Parent: rootID,
				Kind:   cli.CommandEntryCanonical,
				Name:   "tool",
			},
			aliasID: {
				ID:     aliasID,
				Parent: rootID,
				Kind:   cli.CommandEntryAlias,
				Name:   "alias",
			},
		},
	}
	root := RootCommand()

	for _, test := range []struct {
		name    string
		locator CommandEntryLocator
	}{
		{
			name:    "canonical",
			locator: root.CanonicalEntry(),
		},
		{
			name:    "observed alias",
			locator: root.Alias("alias"),
		},
	} {
		name := test.name
		locator := test.locator
		resolved := resolveInspectionCommandEntryLocator(locator.state, inspection)
		if resolved.status != locatorResolutionFound {
			t.Fatalf("%s = status %v actual %q, want found", name, resolved.status, renderEvaluationField(resolved.detail))
		}
	}
	for _, test := range []struct {
		name    string
		locator CommandEntryLocator
	}{
		{
			name:    "missing alias",
			locator: root.Alias("missing"),
		},
	} {
		name := test.name
		locator := test.locator
		resolved := resolveInspectionCommandEntryLocator(locator.state, inspection)
		if resolved.status != locatorResolutionNotProvable {
			t.Fatalf("%s = status %v actual %q, want not provable", name, resolved.status, renderEvaluationField(resolved.detail))
		}
	}
	if resolved := resolveInspectionCommandEntryLocator(root.EntryAt(0).state, inspection); resolved.status != locatorResolutionFound || resolved.value.ID != canonicalID {
		t.Fatalf("observed entry index = status %v value %v actual %q, want found canonical entry", resolved.status, resolved.value.ID, renderEvaluationField(resolved.detail))
	}

	closed := inspection
	closed.Commands = map[cli.CommandID]cli.CommandInspection{rootID: inspection.Commands[rootID]}
	rootInspection := closed.Commands[rootID]
	rootInspection.NamespaceCompleteness = cli.CommandNamespaceComplete
	closed.Commands[rootID] = rootInspection
	missing := resolveInspectionCommandEntryLocator(root.Alias("missing").state, closed)
	if missing.status != locatorResolutionAbsent {
		t.Fatalf("closed missing alias = status %v actual %q, want absent", missing.status, renderEvaluationField(missing.detail))
	}
	entry := resolveInspectionCommandEntryLocator(root.EntryAt(0).state, closed)
	if entry.status != locatorResolutionFound {
		t.Fatalf("closed entry index = status %v actual %q, want found", entry.status, renderEvaluationField(entry.detail))
	}
}

func TestPartialProviderInspectionDoesNotCloseCommandAbsenceOrExactness(t *testing.T) {
	t.Parallel()

	type config struct{}
	providerFailure := errors.New("provider failure")
	survivorCommand := cli.Command[config]{
		Name: "survivor",
	}
	discardedCommand := cli.Command[config]{
		Name: "discarded",
	}
	//nolint:unparam // The public callback contract fixes this result signature.
	provideSurvivor := func(_ context.Context, batch *cli.CommandBatch[config]) error {
		batch.Command(survivorCommand)
		return nil
	}
	provideDiscarded := func(_ context.Context, batch *cli.CommandBatch[config]) error {
		batch.Command(discardedCommand)
		return providerFailure
	}
	declare := func(root *cli.CommandBuilder[config]) {
		root.Subcommands.CommandsFrom(provideSurvivor)
		root.Subcommands.CommandsFrom(provideDiscarded)
	}
	definition := cli.Definition[config]{
		Name:    "tool",
		Declare: declare,
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	request := cli.InspectionRequest{
		Args: []string{"survivor"},
	}
	inspection, err := app.Inspect(t.Context(), request)
	if !errors.Is(err, providerFailure) {
		t.Fatalf("Inspect() error = %v, want provider failure", err)
	}

	root := RootCommand()
	survivor := root.Command("survivor")
	discarded := root.Command("discarded")
	for _, expectation := range []InspectionExpectation{
		root.NamespaceCompleteness(cli.CommandNamespaceIncomplete),
		survivor.Exists(),
	} {
		if outcome := inspectionExpectationOutcome(
			t,
			expectation,
			inspection,
		); outcome.status != evaluationProved {
			t.Fatalf("positive partial %T = %#v, want proved", expectation, outcome)
		}
	}
	for _, expectation := range []InspectionExpectation{
		discarded.DoesNotExist(),
		root.ChildrenExactly(survivor),
		CommandsExactly(root, survivor),
	} {
		if outcome := inspectionExpectationOutcome(
			t,
			expectation,
			inspection,
		); outcome.status != evaluationNotProvable {
			t.Fatalf("negative/exact partial %T = %#v, want not provable", expectation, outcome)
		}
	}
}
