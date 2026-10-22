package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

type accessorSentinelError struct{}

func (*accessorSentinelError) Error() string { return "accessor sentinel" }

func TestRawAccessorsDetachNestedMutableObservationState(t *testing.T) {
	t.Parallel()

	commandID := cli.CommandID(1)
	grammarProfileID := cli.GrammarProfileID(2)
	diagnosticCode := cli.DiagnosticCode("snapshot")

	inspectionResult := InspectionResult{
		state: &inspectionResultState{
			observation: cli.Inspection{
				Status: cli.InspectionStatusReady,
				Commands: map[cli.CommandID]cli.CommandInspection{
					commandID: {
						ID:   commandID,
						Name: "root",
						Path: []string{"root"},
					},
				},
			},
		},
	}
	inspection, err := inspectionResult.Inspection()
	if err != nil {
		t.Fatalf("Inspection() error = %v", err)
	}
	command := inspection.Commands[commandID]
	command.Path[0] = "mutated"
	inspection.Commands[commandID] = command
	delete(inspection.Commands, commandID)
	retainedInspection, err := inspectionResult.Inspection()
	if err != nil || retainedInspection.Commands[commandID].Path[0] != "root" {
		t.Fatalf("retained Inspection = %#v, %v", retainedInspection.Commands[commandID], err)
	}
	inspectionResult.Require(t, CommandByID(commandID).Exists())

	completionResult := CompletionResult{
		state: &completionResultState{
			observation: cli.Completion{
				Items: []cli.CompletionItem{{
					Value:         "alpha",
					Replacement:   []string{"alpha"},
					SearchTerms:   []string{"alpha", "a"},
					TargetIndexes: []int{0},
					Match: cli.CompletionMatch{
						Ranges: []cli.CompletionRange{{
							Start: 0,
							End:   1,
						}},
					},
				}},
			},
		},
	}
	completion, err := completionResult.Completion()
	if err != nil {
		t.Fatalf("Completion() error = %v", err)
	}
	completion.Items[0].Replacement[0] = "mutated"
	completion.Items[0].SearchTerms[0] = "mutated"
	completion.Items[0].TargetIndexes[0] = 99
	completion.Items[0].Match.Ranges[0].End = 99
	retainedCompletion, err := completionResult.Completion()
	if err != nil || retainedCompletion.Items[0].Replacement[0] != "alpha" || retainedCompletion.Items[0].SearchTerms[0] != "alpha" || retainedCompletion.Items[0].TargetIndexes[0] != 0 || retainedCompletion.Items[0].Match.Ranges[0].End != 1 {
		t.Fatalf("retained Completion = %#v, %v", retainedCompletion.Items[0], err)
	}
	completionResult.Require(t, CompletionItem("alpha").Present())

	schemaResult := SchemaResult{
		state: &schemaResultState{
			observation: cli.Schema{
				Commands: map[cli.CommandID]cli.CommandSchema{
					commandID: {
						ID:   commandID,
						Name: "root",
						Path: []string{"root"},
						Dialect: &cli.CommandDialect{
							EndOfOptions: []string{"--"},
						},
					},
				},
			},
		},
	}
	schema := schemaResult.Schema()
	schemaCommand := schema.Commands[commandID]
	schemaCommand.Path[0] = "mutated"
	schemaCommand.Dialect.EndOfOptions[0] = "mutated"
	schema.Commands[commandID] = schemaCommand
	delete(schema.Commands, commandID)
	retainedSchema := schemaResult.Schema()
	if retainedSchema.Commands[commandID].Path[0] != "root" || retainedSchema.Commands[commandID].Dialect.EndOfOptions[0] != "--" {
		t.Fatalf("retained Schema = %#v", retainedSchema.Commands[commandID])
	}
	schemaResult.Require(t, CommandByID(commandID).Exists())

	referenceResult := ReferenceResult{
		state: &referenceResultState{
			observation: cli.Reference{
				Commands: map[cli.CommandID]cli.CommandReference{
					commandID: {
						ID:   commandID,
						Name: "root",
						Path: []string{"root"},
						ProfileUsage: map[cli.GrammarProfileID][]string{
							grammarProfileID: {"root --flag"},
						},
					},
				},
			},
		},
	}
	reference, err := referenceResult.Reference()
	if err != nil {
		t.Fatalf("Reference() error = %v", err)
	}
	referenceCommand := reference.Commands[commandID]
	referenceCommand.Path[0] = "mutated"
	referenceCommand.ProfileUsage[grammarProfileID][0] = "mutated"
	reference.Commands[commandID] = referenceCommand
	delete(reference.Commands, commandID)
	retainedReference, err := referenceResult.Reference()
	if err != nil || retainedReference.Commands[commandID].Path[0] != "root" || retainedReference.Commands[commandID].ProfileUsage[grammarProfileID][0] != "root --flag" {
		t.Fatalf("retained Reference = %#v, %v", retainedReference.Commands[commandID], err)
	}
	referenceResult.Require(t, CommandByID(commandID).Exists())

	diagnosticsResult := DiagnosticsResult{
		state: &diagnosticsResultState{
			observation: cli.DiagnosticReport{
				Diagnostics: []cli.Diagnostic{{
					Code:        diagnosticCode,
					CommandPath: []string{"root"},
					Span: &cli.InputSpan{
						Start: cli.InputPosition{
							Offset: 1,
						},
						End: cli.InputPosition{
							Offset: 2,
						},
					},
					Origins: []cli.InputOriginID{{}},
				}},
			},
		},
	}
	report := diagnosticsResult.Report()
	report.Diagnostics[0].CommandPath[0] = "mutated"
	report.Diagnostics[0].Span.Start.Offset = 99
	report.Diagnostics[0].Origins = append(report.Diagnostics[0].Origins, cli.InputOriginID{})
	retainedReport := diagnosticsResult.Report()
	if retainedReport.Diagnostics[0].CommandPath[0] != "root" || retainedReport.Diagnostics[0].Span.Start.Offset != 1 || len(retainedReport.Diagnostics[0].Origins) != 1 {
		t.Fatalf("retained DiagnosticReport = %#v", retainedReport.Diagnostics[0])
	}
	diagnosticsResult.Require(t, Diagnostic(diagnosticCode).Present())
}

func TestRawAccessorsPreserveExactOpaqueErrorIdentity(t *testing.T) {
	t.Parallel()

	sentinel := &accessorSentinelError{}
	inspection := InspectionResult{
		state: &inspectionResultState{
			err: sentinel,
		},
	}
	completion := CompletionResult{
		state: &completionResultState{
			err: sentinel,
		},
	}
	reference := ReferenceResult{
		state: &referenceResultState{
			err: sentinel,
		},
	}

	if _, err := inspection.Inspection(); err != sentinel { //nolint:errorlint,err113 // Raw accessors must preserve exact opaque error identity.
		t.Fatalf("Inspection() error = %p, want %p", err, sentinel)
	}
	if _, err := completion.Completion(); err != sentinel { //nolint:errorlint,err113 // Raw accessors must preserve exact opaque error identity.
		t.Fatalf("Completion() error = %p, want %p", err, sentinel)
	}
	if _, err := reference.Reference(); err != sentinel { //nolint:errorlint,err113 // Raw accessors must preserve exact opaque error identity.
		t.Fatalf("Reference() error = %p, want %p", err, sentinel)
	}
}
