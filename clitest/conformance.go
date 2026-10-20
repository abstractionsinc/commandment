package clitest

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

// ConformanceCase is the closed set of operation and localization cases that a
// Conformance can compose. It reuses clitest's ordinary expectation vocabulary rather
// than defining a second assertion language.
type ConformanceCase interface{ conformanceCase() }

// Conformance is an ordered reusable declaration of clitest cases. The caller
// owns the declaration until Require snapshots it. The zero value is valid and
// contains no cases. It retains no cross-run result state, performs no hidden
// replay, and introduces no test runner beyond ordinary testing subtests.
type Conformance []ConformanceCase

// InspectionCase describes one reusable Inspect expectation case. Require snapshots
// its retained request and expectation input before execution.
type InspectionCase struct {
	// Name is the subtest name. If empty, Conformance.Require assigns a
	// deterministic case-N name.
	Name string
	// Request is the Inspection request for the case. Conformance snapshots its
	// retained mutable input before execution.
	Request cli.InspectionRequest
	// Expectations are evaluated in declaration order. Conformance snapshots the
	// slice before execution.
	Expectations []InspectionExpectation
}

// CompletionCase describes one reusable Complete expectation case. Require snapshots
// its retained request and expectation input before execution.
type CompletionCase struct {
	// Name is the subtest name. If empty, Conformance.Require assigns a
	// deterministic case-N name.
	Name string
	// Request is the Completion request for the case. Conformance snapshots its
	// retained mutable input before execution.
	Request cli.CompletionRequest
	// Expectations are evaluated in declaration order. Conformance snapshots the
	// slice before execution.
	Expectations []CompletionExpectation
}

// SchemaCase describes one reusable Schema test case. Require snapshots its
// expectation slice before execution.
type SchemaCase struct {
	// Name is the subtest name. If empty, Conformance.Require assigns a
	// deterministic case-N name.
	Name string
	// Expectations are evaluated in declaration order. Conformance snapshots the
	// slice before execution.
	Expectations []SchemaExpectation
}

// ReferenceCase describes one reusable Reference test case. Require snapshots
// its expectation slice before execution.
type ReferenceCase struct {
	// Name is the subtest name. If empty, Conformance.Require assigns a
	// deterministic case-N name.
	Name string
	// Options are passed to the Reference operation for this case.
	Options cli.ReferenceOptions
	// Expectations are evaluated in declaration order. Conformance snapshots the
	// slice before execution.
	Expectations []ReferenceExpectation
}

// LocalizationCase composes the existing whole-application localization audit
// into Conformance without redefining localization semantics.
type LocalizationCase struct {
	// Name is the subtest name. If empty, Conformance.Require assigns a
	// deterministic case-N name.
	Name string
	// Policy is the whole-application localization policy for the case. Conformance
	// snapshots retained mutable policy input.
	Policy LocalizationPolicy
}

func (InspectionCase) conformanceCase() {}

func (SchemaCase) conformanceCase() {}

func (ReferenceCase) conformanceCase() {}

// Require snapshots the declared cases, then runs them as ordinary testing
// subtests in declaration order using the same Inspect, Complete, Schema,
// Reference, and localization expectation paths as direct tests. It does not
// call t.Parallel or create a competing test lifecycle.
//
//nolint:revive // *testing.T intentionally leads this public test-helper API by Go testing convention.
func (conformance Conformance) Require(t *testing.T, ctx context.Context, app cli.Application) {
	t.Helper()
	cases := snapshotConformanceCases(conformance)
	requireConformance(t, ctx, app, cases)
}

func snapshotConformanceCases(cases []ConformanceCase) []ConformanceCase {
	result := make([]ConformanceCase, len(cases))
	for i, item := range cases {
		switch value := item.(type) {
		case *InspectionCase:
			if value == nil {
				panic("clitest: nil conformance case")
			}
			item = *value
		case *CompletionCase:
			if value == nil {
				panic("clitest: nil conformance case")
			}
			item = *value
		case *SchemaCase:
			if value == nil {
				panic("clitest: nil conformance case")
			}
			item = *value
		case *ReferenceCase:
			if value == nil {
				panic("clitest: nil conformance case")
			}
			item = *value
		case *LocalizationCase:
			if value == nil {
				panic("clitest: nil conformance case")
			}
			item = *value
		}

		switch value := item.(type) {
		case InspectionCase:
			value.Request = cloneInspectionRequest(value.Request)
			value.Expectations = slices.Clone(value.Expectations)
			result[i] = value
		case CompletionCase:
			value.Request = cloneCompletionRequest(value.Request)
			value.Expectations = slices.Clone(value.Expectations)
			result[i] = value
		case SchemaCase:
			value.Expectations = slices.Clone(value.Expectations)
			result[i] = value
		case ReferenceCase:
			value.Expectations = slices.Clone(value.Expectations)
			result[i] = value
		case LocalizationCase:
			value.Policy = cloneLocalizationPolicy(value.Policy)
			result[i] = value
		case nil:
			panic("clitest: nil conformance case")
		default:
			panic(fmt.Sprintf("clitest: unsupported conformance case %T", item))
		}
	}
	return result
}

//nolint:revive // *testing.T intentionally leads this private test helper by Go testing convention.
func requireConformance(t *testing.T, ctx context.Context, app cli.Application, cases []ConformanceCase) {
	t.Helper()
	if ctx == nil {
		panic("clitest: nil conformance context")
	}
	for i, item := range cases {
		switch value := item.(type) {
		case InspectionCase:
			t.Run(conformanceCaseName(value.Name, i), func(t *testing.T) {
				t.Helper()
				Inspect(ctx, app, cloneInspectionRequest(value.Request)).Require(t, value.Expectations...)
			})
		case CompletionCase:
			t.Run(conformanceCaseName(value.Name, i), func(t *testing.T) {
				t.Helper()
				Complete(ctx, app, cloneCompletionRequest(value.Request)).Require(t, value.Expectations...)
			})
		case SchemaCase:
			t.Run(conformanceCaseName(value.Name, i), func(t *testing.T) {
				t.Helper()
				Schema(app).Require(t, value.Expectations...)
			})
		case ReferenceCase:
			t.Run(conformanceCaseName(value.Name, i), func(t *testing.T) {
				t.Helper()
				Reference(app, value.Options).Require(t, value.Expectations...)
			})
		case LocalizationCase:
			t.Run(conformanceCaseName(value.Name, i), func(t *testing.T) {
				t.Helper()
				RequireLocalization(t, app, cloneLocalizationPolicy(value.Policy))
			})
		default:
			panic(fmt.Sprintf("clitest: invalid retained conformance case %T", item))
		}
	}
}

func conformanceCaseName(name string, index int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("case-%d", index+1)
}

func cloneInspectionRequest(input cli.InspectionRequest) cli.InspectionRequest {
	input.Args = append([]string(nil), input.Args...)
	input.Env = append(cli.Env(nil), input.Env...)
	input.Defaults = append([]cli.DefaultProvider(nil), input.Defaults...)
	input.Facts = append([]cli.FactProvider(nil), input.Facts...)
	return input
}

func cloneCompletionRequest(input cli.CompletionRequest) cli.CompletionRequest {
	input.Args = append([]string(nil), input.Args...)
	input.Env = append(cli.Env(nil), input.Env...)
	input.Defaults = append([]cli.DefaultProvider(nil), input.Defaults...)
	input.Facts = append([]cli.FactProvider(nil), input.Facts...)
	input.Client.Match.Fields = append([]cli.CompletionMatchField(nil), input.Client.Match.Fields...)
	if input.ObservedInput != nil {
		observed := *input.ObservedInput
		observed.Words = append([]cli.CompletionObservedWord(nil), input.ObservedInput.Words...)
		input.ObservedInput = &observed
	}
	return input
}

func cloneLocalizationPolicy(input LocalizationPolicy) LocalizationPolicy {
	input.RequireCompleteLocales = slices.Clone(input.RequireCompleteLocales)
	input.RequireExactLocales = slices.Clone(input.RequireExactLocales)
	input.RuntimeSources = slices.Clone(input.RuntimeSources)
	input.AllowedSharedSources = slices.Clone(input.AllowedSharedSources)
	return input
}
