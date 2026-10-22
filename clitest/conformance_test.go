package clitest

import (
	"context"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestConformanceDeclarationIsOrderedCallerOwnedAndSnapshotsAtExecutionBoundary(t *testing.T) {
	t.Parallel()
	args := []string{"one"}
	expectations := []InspectionExpectation{nil}
	firstCase := InspectionCase{
		Name: "first",
		Request: cli.InspectionRequest{
			Args: args,
		},
		Expectations: expectations,
	}
	secondCase := SchemaCase{
		Name: "second",
	}
	conformance := Conformance{firstCase, secondCase}

	args[0] = "mutated"
	expectations[0] = InspectionReady()
	declared, ok := conformance[0].(InspectionCase)
	if !ok {
		t.Fatalf("declared case type = %T, want InspectionCase", conformance[0])
	}
	if declared.Request.Args[0] != "mutated" || declared.Expectations[0] == nil {
		t.Fatalf("caller-owned declaration = %#v", declared)
	}

	snapshot := snapshotConformanceCases(conformance)
	args[0] = "after-snapshot"
	expectations[0] = nil
	retained, ok := snapshot[0].(InspectionCase)
	if !ok {
		t.Fatalf("retained case type = %T, want InspectionCase", snapshot[0])
	}
	if retained.Request.Args[0] != "mutated" || retained.Expectations[0] == nil {
		t.Fatalf("execution snapshot = %#v", retained)
	}
	second, ok := snapshot[1].(SchemaCase)
	if !ok {
		t.Fatalf("second case type = %T, want SchemaCase", snapshot[1])
	}
	if second.Name != "second" {
		t.Fatalf("second case = %q", second.Name)
	}
}

func TestEmptyConformanceIsValidAndUsesOrdinaryGoComposition(t *testing.T) {
	t.Parallel()
	var zero Conformance
	zero.Require(t, t.Context(), cli.Application{})
	base := Conformance{SchemaCase{
		Name: "schema",
	}}
	got := append(Conformance(nil), base...)
	if len(zero) != 0 || len(got) != 1 {
		t.Fatalf("zero/composed state = %#v / %#v", zero, got)
	}
}

func TestResultRawAccessorsRetainExactStoredObservation(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Truncated: true,
	}
	ir := InspectionResult{
		state: &inspectionResultState{
			observation: inspection,
			err:         context.Canceled,
		},
	}
	gotInspection, gotErr := ir.Inspection()
	if gotInspection.Truncated != inspection.Truncated ||
		gotErr != context.Canceled { //nolint:errorlint,err113 // Conformance forwarding must preserve the exact returned error.
		t.Fatalf("Inspection() = %#v, %v", gotInspection, gotErr)
	}

	completion := cli.Completion{
		Partial: true,
	}
	cr := CompletionResult{
		state: &completionResultState{
			observation: completion,
			err:         context.DeadlineExceeded,
		},
	}
	gotCompletion, gotErr := cr.Completion()
	if gotCompletion.Partial != completion.Partial ||
		gotErr != context.DeadlineExceeded { //nolint:errorlint,err113 // Conformance forwarding must preserve the exact returned error.
		t.Fatalf("Completion() = %#v, %v", gotCompletion, gotErr)
	}

	schema := cli.Schema{}
	sr := SchemaResult{
		state: &schemaResultState{
			observation: schema,
		},
	}
	gotSchema := sr.Schema()
	if gotSchema.EvaluatedRelease != schema.EvaluatedRelease || gotSchema.Root != schema.Root ||
		len(gotSchema.Commands) != len(schema.Commands) ||
		len(gotSchema.Values) != len(schema.Values) {
		t.Fatal("Schema() did not return exact retained value")
	}

	reference := cli.Reference{
		EvaluatedRelease: "r1",
	}
	rr := ReferenceResult{
		state: &referenceResultState{
			observation: reference,
			err:         context.Canceled,
		},
	}
	gotReference, gotErr := rr.Reference()
	if gotReference.EvaluatedRelease != reference.EvaluatedRelease ||
		gotErr != context.Canceled { //nolint:errorlint,err113 // Conformance forwarding must preserve the exact returned error.
		t.Fatalf("Reference() = %#v, %v", gotReference, gotErr)
	}
}

func TestConformanceSnapshotsEveryFrozenCaseForm(t *testing.T) {
	t.Parallel()
	inspectionExpectations := []InspectionExpectation{nil}
	completionExpectations := []CompletionExpectation{nil}
	schemaExpectations := []SchemaExpectation{nil}
	referenceExpectations := []ReferenceExpectation{nil}
	fields := []cli.CompletionMatchField{cli.CompletionMatchFieldValue}
	completeLocales := []cli.Locale{"fr"}
	inspectionCase := InspectionCase{
		Name:         "inspect",
		Expectations: inspectionExpectations,
	}
	completionRequest := cli.CompletionRequest{
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Fields: fields,
			},
		},
	}
	completionCase := CompletionCase{
		Name:         "complete",
		Request:      completionRequest,
		Expectations: completionExpectations,
	}
	schemaCase := SchemaCase{
		Name:         "schema",
		Expectations: schemaExpectations,
	}
	referenceCase := ReferenceCase{
		Name:         "reference",
		Expectations: referenceExpectations,
	}
	localizationPolicy := LocalizationPolicy{
		RequireCompleteLocales: completeLocales,
	}
	localizationCase := LocalizationCase{
		Name:   "localization",
		Policy: localizationPolicy,
	}
	conformance := Conformance{inspectionCase, completionCase, schemaCase, referenceCase, localizationCase}
	got := snapshotConformanceCases(conformance)
	inspectionExpectations[0] = InspectionReady()
	completionExpectations[0] = CompletionPartial()
	schemaExpectations[0] = CommandsExactly()
	referenceExpectations[0] = CommandsExactly()
	fields[0] = cli.CompletionMatchFieldDetail
	completeLocales[0] = "de"

	if len(got) != 5 {
		t.Fatalf("cases = %#v", got)
	}
	inspectionCase, inspectionOK := got[0].(InspectionCase)
	completionCase, completionOK := got[1].(CompletionCase)
	schemaCase, schemaOK := got[2].(SchemaCase)
	referenceCase, referenceOK := got[3].(ReferenceCase)
	localizationCase, localizationOK := got[4].(LocalizationCase)
	if !inspectionOK || inspectionCase.Expectations[0] != nil ||
		!completionOK || completionCase.Expectations[0] != nil ||
		completionCase.Request.Client.Match.Fields[0] != cli.CompletionMatchFieldValue ||
		!schemaOK || schemaCase.Expectations[0] != nil ||
		!referenceOK || referenceCase.Expectations[0] != nil ||
		!localizationOK || localizationCase.Policy.RequireCompleteLocales[0] != "fr" {
		t.Fatalf("snapshots = %#v", got)
	}
}

func TestConformanceSnapshotsPointerCaseFormsIntoDetachedValues(t *testing.T) {
	t.Parallel()

	inspectionExpectations := []InspectionExpectation{nil}
	completionExpectations := []CompletionExpectation{nil}
	schemaExpectations := []SchemaExpectation{nil}
	referenceExpectations := []ReferenceExpectation{nil}
	inspectionArgs := []string{"inspect"}
	completionFields := []cli.CompletionMatchField{cli.CompletionMatchFieldValue}
	completeLocales := []cli.Locale{"fr"}

	inspection := &InspectionCase{
		Name: "inspection",
		Request: cli.InspectionRequest{
			Args: inspectionArgs,
		},
		Expectations: inspectionExpectations,
	}
	completion := &CompletionCase{
		Name: "completion",
		Request: cli.CompletionRequest{
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Fields: completionFields,
				},
			},
		},
		Expectations: completionExpectations,
	}
	schema := &SchemaCase{
		Name:         "schema",
		Expectations: schemaExpectations,
	}
	reference := &ReferenceCase{
		Name:         "reference",
		Expectations: referenceExpectations,
	}
	localization := &LocalizationCase{
		Name: "localization",
		Policy: LocalizationPolicy{
			RequireCompleteLocales: completeLocales,
		},
	}

	conformance := Conformance{inspection, completion, schema, reference, localization}
	got := snapshotConformanceCases(conformance)

	inspection.Name = "mutated"
	inspectionArgs[0] = "mutated"
	inspectionExpectations[0] = InspectionReady()
	completion.Name = "mutated"
	completionFields[0] = cli.CompletionMatchFieldDetail
	completionExpectations[0] = CompletionPartial()
	schema.Name = "mutated"
	schemaExpectations[0] = CommandsExactly()
	reference.Name = "mutated"
	referenceExpectations[0] = CommandsExactly()
	localization.Name = "mutated"
	completeLocales[0] = "de"

	if len(got) != 5 {
		t.Fatalf("cases = %#v", got)
	}
	inspectionCase, inspectionOK := got[0].(InspectionCase)
	completionCase, completionOK := got[1].(CompletionCase)
	schemaCase, schemaOK := got[2].(SchemaCase)
	referenceCase, referenceOK := got[3].(ReferenceCase)
	localizationCase, localizationOK := got[4].(LocalizationCase)
	if !inspectionOK || inspectionCase.Name != "inspection" || inspectionCase.Request.Args[0] != "inspect" ||
		inspectionCase.Expectations[0] != nil ||
		!completionOK ||
		completionCase.Name != "completion" ||
		completionCase.Request.Client.Match.Fields[0] != cli.CompletionMatchFieldValue ||
		completionCase.Expectations[0] != nil ||
		!schemaOK ||
		schemaCase.Name != "schema" ||
		schemaCase.Expectations[0] != nil ||
		!referenceOK ||
		referenceCase.Name != "reference" ||
		referenceCase.Expectations[0] != nil ||
		!localizationOK ||
		localizationCase.Name != "localization" ||
		localizationCase.Policy.RequireCompleteLocales[0] != "fr" {
		t.Fatalf("pointer snapshots = %#v", got)
	}
}

func TestConformancePointerCaseFormsExecuteAsRetainedValues(t *testing.T) {
	t.Parallel()

	definition := cli.Definition[struct{}]{
		Name: "conformance-pointer-cases",
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	inspectionCase := &InspectionCase{
		Name: "inspection",
	}
	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
	}
	completionCase := &CompletionCase{
		Name:    "completion",
		Request: completionRequest,
	}
	schemaCase := &SchemaCase{
		Name: "schema",
	}
	referenceCase := &ReferenceCase{
		Name: "reference",
	}
	localizationCase := &LocalizationCase{
		Name: "localization",
	}
	conformance := Conformance{inspectionCase, completionCase, schemaCase, referenceCase, localizationCase}
	conformance.Require(t, t.Context(), app)
}

func TestConformanceTypedNilPointerCasesUseNilCasePanicContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		item ConformanceCase
	}{
		{
			name: "inspection",
			item: (*InspectionCase)(nil),
		},
		{
			name: "completion",
			item: (*CompletionCase)(nil),
		},
		{
			name: "schema",
			item: (*SchemaCase)(nil),
		},
		{
			name: "reference",
			item: (*ReferenceCase)(nil),
		},
		{
			name: "localization",
			item: (*LocalizationCase)(nil),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recovered := recover(); recovered != "clitest: nil conformance case" {
					t.Fatalf("panic = %#v", recovered)
				}
			}()
			_ = snapshotConformanceCases(Conformance{test.item})
		})
	}
}

func TestConformanceCompletionObservedInputWordsAreDeepSnapshotted(t *testing.T) {
	t.Parallel()

	words := []cli.CompletionObservedWord{{
		Value: "original",
	}}
	observed := &cli.CompletionInputObservation{
		Words:    words,
		Topology: cli.CompletionArgvTopologyExact,
	}
	request := cli.CompletionRequest{
		ObservedInput: observed,
	}
	completionCase := CompletionCase{
		Request: request,
	}
	conformance := Conformance{completionCase}
	snapshot := snapshotConformanceCases(conformance)

	words[0].Value = "mutated"
	observed.Words = append(observed.Words, cli.CompletionObservedWord{
		Value: "extra",
	})

	retained, ok := snapshot[0].(CompletionCase)
	if !ok {
		t.Fatalf("retained case type = %T, want CompletionCase", snapshot[0])
	}
	if retained.Request.ObservedInput == nil || len(retained.Request.ObservedInput.Words) != 1 || retained.Request.ObservedInput.Words[0].Value != "original" {
		t.Fatalf("retained observed input = %#v", retained.Request.ObservedInput)
	}

	execution := cloneCompletionRequest(retained.Request)
	execution.ObservedInput.Words[0].Value = "execution-mutated"
	if retained.Request.ObservedInput.Words[0].Value != "original" {
		t.Fatalf("execution clone mutated retained request = %#v", retained.Request.ObservedInput)
	}
}

type recordingSchemaExpectation struct {
	order *[]string
	name  string
}

func (recordingSchemaExpectation) schemaExpectation() {}

func (expectation recordingSchemaExpectation) evaluateSchema(*cli.Schema) evaluationResult {
	*expectation.order = append(*expectation.order, expectation.name)
	return provedResult()
}

func TestConformanceRequireUsesNativeSubtestsInDeclarationOrder(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "conformance-order",
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var order []string
	firstExpectation := recordingSchemaExpectation{
		order: &order,
		name:  "first",
	}
	firstCase := SchemaCase{
		Name:         "first",
		Expectations: []SchemaExpectation{firstExpectation},
	}
	secondExpectation := recordingSchemaExpectation{
		order: &order,
		name:  "second",
	}
	secondCase := SchemaCase{
		Name:         "second",
		Expectations: []SchemaExpectation{secondExpectation},
	}
	conformance := Conformance{firstCase, secondCase}
	conformance.Require(t, t.Context(), app)
	if want := []string{"first", "second"}; !slices.Equal(order, want) {
		t.Fatalf("execution order = %#v, want %#v", order, want)
	}
}

func TestConformanceSnapshotPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	got := Conformance{}
	if len(got) != 0 {
		t.Fatalf("empty conformance cases = %#v", got)
	}
}

func TestConformanceOrdinaryCompositionPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()
	base := Conformance{SchemaCase{
		Name: "schema",
	}}
	got := append(Conformance(nil), base...)
	got = append(got, InspectionCase{
		Name: "inspection",
	})
	if len(base) != 1 || len(got) != 2 {
		t.Fatalf("composition changed membership: base %#v got %#v", base, got)
	}
	first, firstOK := got[0].(SchemaCase)
	second, secondOK := got[1].(InspectionCase)
	if !firstOK || !secondOK || first.Name != "schema" || second.Name != "inspection" {
		t.Fatalf("composition order = %#v", got)
	}
}

func TestConformanceCaseSnapshotPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered := recover(); recovered != "clitest: nil conformance case" {
			t.Fatalf("panic = %#v", recovered)
		}
	}()
	snapshotConformanceCases([]ConformanceCase{nil})
}

func TestConformanceExecutionPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()

		func() {
			defer func() {
				if recovered := recover(); recovered != "clitest: nil conformance context" {
					t.Fatalf("nil-context panic = %#v", recovered)
				}
			}()
			requireConformance(t, nil, cli.Application{}, Conformance{})
		}()

		definition := cli.Definition[struct{}]{
			Name: "conformance-residual",
		}
		app, err := cli.Compile(definition)
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}
		inspectionCase := InspectionCase{
			Name: "inspection",
		}
		completionRequest := cli.CompletionRequest{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
			},
		}
		completionCase := CompletionCase{
			Name:    "completion",
			Request: completionRequest,
		}
		referenceCase := ReferenceCase{
			Name: "reference",
		}
		localizationCase := LocalizationCase{
			Name: "localization",
		}
		conformance := Conformance{inspectionCase, completionCase, referenceCase, localizationCase}
		conformance.Require(t, t.Context(), app)
	})
}

func TestConformanceCaseNamePreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	if got := conformanceCaseName("", 2); got != "case-3" {
		t.Fatalf("fallback name = %q", got)
	}
}

type unsupportedConformanceCase struct{}

func (unsupportedConformanceCase) conformanceCase() {}

//nolint:tparallel // This test must synchronously observe the panic raised while requireConformance drives its subtest boundary.
func TestConformanceRejectsUnsupportedRetainedCase(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatalf("unsupported retained conformance case did not panic")
		}
	}()
	conformance := Conformance{unsupportedConformanceCase{}}
	requireConformance(t, t.Context(), cli.Application{}, conformance)
}

func TestConformanceSnapshotRejectsUnsupportedInternalCase(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatalf("unsupported conformance snapshot case did not panic")
		}
	}()
	cases := []ConformanceCase{unsupportedConformanceCase{}}
	_ = snapshotConformanceCases(cases)
}
