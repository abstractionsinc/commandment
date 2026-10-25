package conformance

import (
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSynthesizedShellCompletionDeclarationsPreserveFrontendDeclarationOriginAcrossArtifacts(t *testing.T) {
	t.Parallel()

	definition := cli.Definition[struct{}]{
		Name: "alienctl",
	}
	app := clitest.Compile(t, definition)

	schema := app.Schema()
	completion := schemaCommandByName(t, schema.Commands, "completion")
	if root := schema.Commands[schema.Root]; root.Origin != cli.DeclarationOriginApplication {
		t.Fatalf("root origin = %v", root.Origin)
	}
	reference, err := app.Reference(cli.ReferenceOptions{})
	if err != nil {
		t.Fatalf("Reference() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"completion", "bash"},
	}
	inspection, err := app.Inspect(t.Context(), request)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	want := cli.DeclarationOriginFrontend
	artifacts := [...]string{"schema", "reference", "inspection"}
	assertOrigins := func(subject string, got ...cli.DeclarationOrigin) {
		t.Helper()
		for index, origin := range got {
			if origin != want {
				t.Fatalf("%s %s origin = %v, want %v", artifacts[index], subject, origin, want)
			}
		}
	}
	command := schema.Commands[completion.ID]
	assertOrigins("command", command.Origin, reference.Commands[completion.ID].Origin, inspection.Commands[completion.ID].Origin)
	entryID := command.CanonicalEntry
	assertOrigins("command entry", schema.CommandEntries[entryID].Origin, reference.CommandEntries[entryID].Origin, inspection.CommandEntries[entryID].Origin)
	for _, id := range command.Flags {
		assertOrigins(fmt.Sprintf("flag %v", id), schema.Flags[id].Origin, reference.Flags[id].Origin, inspection.Flags[id].Origin)
		for _, spellingID := range schema.Flags[id].Spellings {
			assertOrigins(fmt.Sprintf("flag spelling %v", spellingID), schema.FlagSpellings[spellingID].Origin, reference.FlagSpellings[spellingID].Origin, inspection.FlagSpellings[spellingID].Origin)
		}
	}
	for _, id := range command.ArgumentForms {
		assertOrigins(fmt.Sprintf("argument form %v", id), schema.ArgumentForms[id].Origin, reference.ArgumentForms[id].Origin, inspection.ArgumentForms[id].Origin)
		for _, termID := range schema.ArgumentForms[id].Terms {
			assertOrigins(fmt.Sprintf("argument term %v", termID), schema.ArgumentTerms[termID].Origin, reference.ArgumentTerms[termID].Origin, inspection.ArgumentTerms[termID].Origin)
		}
	}
	for _, id := range command.Values {
		assertOrigins(fmt.Sprintf("value %v", id), schema.Values[id].Origin, reference.Values[id].Origin)
	}
	for id := range schema.ValueDomainEntries {
		assertOrigins(fmt.Sprintf("value-domain entry %v", id), schema.ValueDomainEntries[id].Origin, reference.ValueDomainEntries[id].Origin, inspection.ValueDomainEntries[id].Origin)
	}
}

func schemaCommandByName(t testing.TB, commands map[cli.CommandID]cli.CommandSchema, name string) cli.CommandSchema {
	t.Helper()
	for _, command := range commands {
		if command.Name == name {
			return command
		}
	}
	t.Fatalf("command %q is absent", name)
	return cli.CommandSchema{}
}
