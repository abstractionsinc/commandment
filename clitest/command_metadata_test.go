package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestCommandMetadataExpectationsObservePublicArtifactsExactly(t *testing.T) {
	t.Parallel()

	dialect := cli.CommandDialect{
		FlagOrder:            cli.FlagOrderInterspersed,
		ShortFlagClustering:  cli.ShortFlagClusteringEnabled,
		FlagMatching:         cli.NameMatchingUniquePrefix,
		CommandMatching:      cli.NameMatchingExact,
		FlagNormalization:    cli.NameNormalizationASCIICaseFold,
		CommandNormalization: cli.NameNormalizationExact,
		DashlessFlags:        cli.DashlessFlagModeInitialCluster,
		EndOfOptions:         []string{"--", "---"},
		ResponseFiles:        true,
	}
	access := cli.ExecutionAccess{
		Values:           []cli.ValueID{11},
		OptionalValues:   []cli.ValueID{12},
		ValueStates:      []cli.ValueID{13},
		Locale:           true,
		CommandSelection: true,
		WorkingDirectory: true,
		Stdin:            true,
		Stdout:           true,
		Stderr:           true,
	}
	command := cli.CommandSchema{
		ID:      1,
		Origin:  cli.DeclarationOriginProvider,
		Dialect: &dialect,
		Access:  access,
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: command},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{1: {
			ID:      1,
			Origin:  cli.DeclarationOriginProvider,
			Dialect: &dialect,
			Access:  access,
		}},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:      1,
			Origin:  cli.DeclarationOriginProvider,
			Dialect: &dialect,
			Access:  access,
		}},
	}

	expectedAccess := ExecutionAccess{
		Values:           []ValueLocator{ValueByID(11)},
		OptionalValues:   []ValueLocator{ValueByID(12)},
		ValueStates:      []ValueLocator{ValueByID(13)},
		Locale:           true,
		CommandSelection: true,
		WorkingDirectory: true,
		Stdin:            true,
		Stdout:           true,
		Stderr:           true,
	}
	expectations := []DeclarationExpectation{
		CommandByID(1).EffectiveDialect(dialect),
		CommandByID(1).ExecutionAccessExactly(expectedAccess),
		CommandByID(1).Origin(cli.DeclarationOriginProvider),
	}
	for _, expectation := range expectations {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("schema expectation %T = %+v", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference expectation %T = %+v", expectation, got)
		}
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("inspection expectation %T = %+v", expectation, got)
		}
	}
}

func TestCommandMetadataExpectationsSnapshotCallerInputsAndAttributeMismatch(t *testing.T) {
	t.Parallel()

	dialect := cli.CommandDialect{
		EndOfOptions:  []string{"--"},
		ResponseFiles: true,
	}
	expectation := CommandByID(1).EffectiveDialect(dialect)
	dialect.EndOfOptions[0] = "mutated"

	actual := cli.CommandDialect{
		EndOfOptions:  []string{"--"},
		ResponseFiles: true,
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:      1,
			Dialect: &actual,
		}},
	}
	if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
		t.Fatalf("dialect expectation borrowed caller memory: %+v", got)
	}

	wrong := actual
	wrong.ResponseFiles = false
	inspection.Commands[1] = cli.CommandInspection{
		ID:      1,
		Dialect: &wrong,
	}
	got := inspectionExpectationOutcome(t, CommandByID(1).EffectiveDialect(actual), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "ResponseFiles") {
		t.Fatalf("dialect mismatch attribution = %+v", got)
	}
}

func TestExecutionAccessExactlySnapshotsCallerLocatorSlices(t *testing.T) {
	t.Parallel()

	values := []ValueLocator{ValueByID(11)}
	optional := []ValueLocator{ValueByID(12)}
	states := []ValueLocator{ValueByID(13)}
	expectation := CommandByID(1).ExecutionAccessExactly(
		ExecutionAccess{
			Values:         values,
			OptionalValues: optional,
			ValueStates:    states,
		},
	)
	values[0] = ValueByID(91)
	optional[0] = ValueByID(92)
	states[0] = ValueByID(93)

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID: 1,
			Access: cli.ExecutionAccess{
				Values:         []cli.ValueID{11},
				OptionalValues: []cli.ValueID{12},
				ValueStates:    []cli.ValueID{13},
			},
		}},
	}
	if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
		t.Fatalf("execution-access exact expectation borrowed caller locator slice: %+v", got)
	}
}

func TestCommandMetadataInspectionRefusesFalseCertaintyOnTruncationAndAmbiguity(t *testing.T) {
	t.Parallel()

	truncated := cli.Inspection{
		Commands:  map[cli.CommandID]cli.CommandInspection{},
		Truncated: true,
	}
	for _, expectation := range []DeclarationExpectation{
		CommandByID(9).Origin(cli.DeclarationOriginApplication),
		CommandByID(9).EffectiveDialect(cli.CommandDialect{}),
		CommandByID(9).ExecutionAccessExactly(ExecutionAccess{}),
	} {
		if got := inspectionExpectationOutcome(t, expectation, truncated); got.status != evaluationNotProvable {
			t.Fatalf("truncated missing command for %T = %+v, want not provable", expectation, got)
		}
	}

	ambiguous := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
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
	got := inspectionExpectationOutcome(t, Command("dup").Origin(cli.DeclarationOriginApplication), ambiguous)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "matches") {
		t.Fatalf("ambiguous command = %+v", got)
	}
}

func TestCommandMetadataRejectsMissingDialectInvalidAccessAndFieldMismatches(t *testing.T) {
	t.Parallel()

	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID: 1,
		}},
	}
	if got := schemaExpectationOutcome(
		t,
		CommandByID(1).EffectiveDialect(cli.CommandDialect{}),
		schema,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "effective dialect") {
		t.Fatalf("missing dialect = %+v", got)
	}
	if got := schemaExpectationOutcome(
		t,
		CommandByID(2).Origin(cli.DeclarationOriginApplication),
		schema,
	); got.status != evaluationDisproved {
		t.Fatalf("missing command origin = %+v", got)
	}
	if got := schemaExpectationOutcome(
		t,
		CommandByID(1).ExecutionAccessExactly(ExecutionAccess{
			Values: []ValueLocator{{}},
		}),
		schema,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "execution access") {
		t.Fatalf("invalid access locator = %+v", got)
	}

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID: 1,
			Access: cli.ExecutionAccess{
				Locale: true,
			},
			Origin: cli.DeclarationOriginFrontend,
		}},
	}
	if got := inspectionExpectationOutcome(
		t,
		CommandByID(1).ExecutionAccessExactly(ExecutionAccess{}),
		inspection,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "Locale") {
		t.Fatalf("access mismatch = %+v", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		CommandByID(1).Origin(cli.DeclarationOriginProvider),
		inspection,
	); got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.expected), "origin") ||
		!strings.Contains(renderEvaluationField(got.observed), "origin") {
		t.Fatalf("command origin mismatch = %+v", got)
	}
}

func TestCommandMetadataExpectationsRejectMissingCommands(t *testing.T) {
	t.Parallel()

	dialect := cli.CommandDialect{}
	if got := schemaExpectationOutcome(
		t,
		CommandByID(99).EffectiveDialect(dialect),
		&cli.Schema{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "command is absent") {
		t.Fatalf("schema missing effective dialect = %+v", got)
	}
	if got := referenceExpectationOutcome(
		t,
		CommandByID(99).EffectiveDialect(dialect),
		cli.Reference{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "command is absent") {
		t.Fatalf("reference missing effective dialect = %+v", got)
	}

	access := ExecutionAccess{
		Values: []ValueLocator{ValueByID(1)},
	}
	if got := schemaExpectationOutcome(
		t,
		CommandByID(99).ExecutionAccessExactly(access),
		&cli.Schema{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "command is absent") {
		t.Fatalf("schema missing execution access command = %+v", got)
	}
	if got := referenceExpectationOutcome(
		t,
		CommandByID(99).ExecutionAccessExactly(access),
		cli.Reference{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "command is absent") {
		t.Fatalf("reference missing execution access command = %+v", got)
	}
}

func TestExpectedExecutionAccessRejectsUnresolvableRelationshipsByClass(t *testing.T) {
	t.Parallel()

	missingValue := FlagByID(999).Value().state
	cases := []struct {
		name  string
		state *expectedExecutionAccessState
	}{
		{
			name: "required values",
			state: &expectedExecutionAccessState{
				values: []*valueLocatorState{missingValue},
			},
		},
		{
			name: "optional values",
			state: &expectedExecutionAccessState{
				optionalValues: []*valueLocatorState{missingValue},
			},
		},
		{
			name: "value states",
			state: &expectedExecutionAccessState{
				valueStates: []*valueLocatorState{missingValue},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, status, detail := expectedExecutionAccessSchema(
				tc.state,
				&cli.Schema{},
			); status != locatorResolutionAbsent ||
				renderEvaluationField(detail) == "" {
				t.Fatalf("schema status/detail = %v %q", status, detail)
			}
			if _, status, detail := expectedExecutionAccessReference(
				tc.state,
				cli.Reference{},
			); status != locatorResolutionAbsent ||
				renderEvaluationField(detail) == "" {
				t.Fatalf("reference status/detail = %v %q", status, detail)
			}
			if _, status, detail := expectedExecutionAccessInspection(
				tc.state,
				cli.Inspection{},
			); status != locatorResolutionAbsent ||
				renderEvaluationField(detail) == "" {
				t.Fatalf("inspection status/detail = %v %q", status, detail)
			}
		})
	}
}

func TestExecutionAccessExactlySurfacesUnresolvableExpectedRelationships(t *testing.T) {
	t.Parallel()

	commandID := cli.CommandID(1)
	expected := ExecutionAccess{
		Values: []ValueLocator{FlagByID(999).Value()},
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{commandID: {
			ID: commandID,
		}},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{commandID: {
			ID: commandID,
		}},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{commandID: {
			ID: commandID,
		}},
	}

	if got := schemaExpectationOutcome(
		t,
		CommandByID(commandID).ExecutionAccessExactly(expected),
		schema,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "execution access") {
		t.Fatalf("schema unresolved expected access = %+v", got)
	}
	if got := referenceExpectationOutcome(
		t,
		CommandByID(commandID).ExecutionAccessExactly(expected),
		reference,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "execution access") {
		t.Fatalf("reference unresolved expected access = %+v", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		CommandByID(commandID).ExecutionAccessExactly(expected),
		inspection,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "execution access") {
		t.Fatalf("inspection unresolved expected access = %+v", got)
	}
}
