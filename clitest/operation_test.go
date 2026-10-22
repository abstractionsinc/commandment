package clitest

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

func TestExecutionAndSignalExpectationsUseInspectionPlan(t *testing.T) {
	t.Parallel()
	interrupt := testSemanticSignal(t)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:   1,
				Path: []string{},
			},
			2: {
				ID:   2,
				Path: []string{"serve"},
			},
		},
		ExecutionAdmission: cli.ExecutionAdmissionInspection{
			Status:         cli.ExecutionAdmissionReady,
			HandlerCommand: 2,
			Middleware: []cli.MiddlewareInspection{
				{
					Scope: cli.MiddlewareScopeApplication,
				},
				{
					Scope:            cli.MiddlewareScopeCommand,
					Command:          2,
					DeclarationIndex: 4,
				},
			},
			SignalPolicies:  []cli.SignalPolicyID{5},
			SignalEndpoints: []cli.SignalEndpointID{6},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
			5: {
				ID:                5,
				Status:            cli.SignalPolicyStatusReady,
				ActivationOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
			6: {
				Status:            cli.SignalEndpointStatusReady,
				EffectivePolicies: []cli.SignalPolicyID{5},
			},
		},
		SignalSource: cli.SignalSourceInspection{
			Status:                     cli.SignalIntegrationStatusReady,
			Required:                   true,
			Supplied:                   true,
			ResolvedSignals:            []cli.Signal{interrupt},
			RequiredSourceCapabilities: cli.SignalSourceCapabilityOrigin,
			RequiredHostCapabilities:   cli.SignalHostCapabilityAbort,
		},
	}

	assertInspectionExpectation(t, ExecutionAdmission().Ready(), inspection)
	assertInspectionExpectation(t, ExecutionAdmission().Handler(Command("serve")), inspection)
	assertInspectionExpectation(
		t,
		ExecutionAdmission().MiddlewareExactly(ExecutionAdmission().MiddlewareAt(0), ExecutionAdmission().MiddlewareAt(1)),
		inspection,
	)
	assertInspectionExpectation(t, ExecutionAdmission().MiddlewareAt(1).Scope(cli.MiddlewareScopeCommand), inspection)
	assertInspectionExpectation(t, ExecutionAdmission().MiddlewareAt(1).Command(Command("serve")), inspection)
	assertInspectionExpectation(t, ExecutionAdmission().SignalPoliciesExactly(SignalPolicyByID(5)), inspection)
	assertInspectionExpectation(t, ExecutionAdmission().SignalEndpointsExactly(SignalEndpointByID(6)), inspection)
	assertInspectionExpectation(t, SignalPolicyByID(5).Status(cli.SignalPolicyStatusReady), inspection)
	assertInspectionExpectation(t, SignalPolicyByID(5).ActivationOutcome(cli.EvaluationOutcomeSatisfied), inspection)
	assertInspectionExpectation(t, SignalEndpointByID(6).Status(cli.SignalEndpointStatusReady), inspection)
	assertInspectionExpectation(t, SignalEndpointByID(6).EffectivePoliciesExactly(SignalPolicyByID(5)), inspection)
	assertInspectionExpectation(t, SignalIntegration().Status(cli.SignalIntegrationStatusReady), inspection)
	assertInspectionExpectation(t, SignalIntegration().Required(), inspection)
	assertInspectionExpectation(t, SignalIntegration().Supplied(), inspection)
	assertInspectionExpectation(t, SignalIntegration().ResolvedSignalsExactly(interrupt), inspection)
	assertInspectionExpectation(
		t,
		SignalIntegration().RequiredSourceCapabilities(cli.SignalSourceCapabilityOrigin),
		inspection,
	)
	assertInspectionExpectation(t, SignalIntegration().RequiredHostCapabilities(cli.SignalHostCapabilityAbort), inspection)
}

func TestSignalIntegrationExpectationsOwnAsymmetricSemanticFields(t *testing.T) {
	t.Parallel()
	signals := testSemanticSignals(t, "resolved", "missing")
	resolved := signals[0]
	missing := signals[1]
	inspection := cli.Inspection{
		SignalSource: cli.SignalSourceInspection{
			Status:                     cli.SignalIntegrationStatusIncompatible,
			Required:                   true,
			Supplied:                   false,
			ResolvedSignals:            []cli.Signal{resolved},
			MissingSignals:             []cli.Signal{missing},
			RequiredSourceCapabilities: cli.SignalSourceCapabilityOrigin,
			MissingSourceCapabilities:  cli.SignalSourceCapabilitySender | cli.SignalSourceCapabilityReraise,
			RequiredHostCapabilities:   cli.SignalHostCapabilityAbort,
		},
	}

	matching := []InspectionExpectation{
		SignalIntegration().Status(cli.SignalIntegrationStatusIncompatible),
		SignalIntegration().Required(),
		SignalIntegration().ResolvedSignalsExactly(resolved),
		SignalIntegration().MissingSignalsExactly(missing),
		SignalIntegration().RequiredSourceCapabilities(cli.SignalSourceCapabilityOrigin),
		SignalIntegration().MissingSourceCapabilities(cli.SignalSourceCapabilitySender | cli.SignalSourceCapabilityReraise),
		SignalIntegration().RequiredHostCapabilities(cli.SignalHostCapabilityAbort),
	}
	for i, expectation := range matching {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("matching SignalIntegration expectation %d = %#v", i, got)
		}
	}

	if got := inspectionExpectationOutcome(
		t,
		SignalIntegration().Supplied(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("Supplied against required-only source = %#v, want disproved", got)
	}

	opposite := cli.Inspection{
		SignalSource: cli.SignalSourceInspection{
			Required: false,
			Supplied: true,
		},
	}
	if got := inspectionExpectationOutcome(
		t,
		SignalIntegration().Supplied(),
		opposite,
	); got.status != evaluationProved {
		t.Fatalf("Supplied against supplied-only source = %#v", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		SignalIntegration().Required(),
		opposite,
	); got.status != evaluationDisproved {
		t.Fatalf("Required against supplied-only source = %#v, want disproved", got)
	}
	mismatching := []InspectionExpectation{
		SignalIntegration().ResolvedSignalsExactly(missing),
		SignalIntegration().MissingSignalsExactly(resolved),
		SignalIntegration().RequiredSourceCapabilities(cli.SignalSourceCapabilitySender),
		SignalIntegration().MissingSourceCapabilities(cli.SignalSourceCapabilityOrigin),
		SignalIntegration().RequiredSourceCapabilities(cli.SignalSourceCapabilityReraise),
		SignalIntegration().MissingHostCapabilities(cli.SignalHostCapabilityAbort),
	}
	for i, expectation := range mismatching {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationDisproved {
			t.Fatalf("swapped SignalIntegration expectation %d = %#v, want disproved", i, got)
		}
	}

	blocked := inspection
	blocked.SignalSource.Status = cli.SignalIntegrationStatusMissing
	if got := inspectionExpectationOutcome(
		t,
		SignalIntegration().Status(cli.SignalIntegrationStatusMissing),
		blocked,
	); got.status != evaluationProved {
		t.Fatalf("missing SignalIntegration status = %#v", got)
	}
}

func TestExactExecutionCollectionIsNotProvableFromTruncatedInspection(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Truncated: true,
		ExecutionAdmission: cli.ExecutionAdmissionInspection{
			Middleware: []cli.MiddlewareInspection{{}},
		},
	}
	expectation := ExecutionAdmission().MiddlewareExactly(ExecutionAdmission().MiddlewareAt(0))
	evaluator, ok := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("MiddlewareExactly expectation does not expose inspection evaluation")
	}
	if outcome := evaluator.evaluateInspection(inspection, nil); outcome.status != evaluationNotProvable {
		t.Fatalf("status = %v, want not provable", outcome.status)
	}
}

func TestDeclaredAndEffectiveSensitivityUseDifferentAuthorities(t *testing.T) {
	t.Parallel()
	value := ValueByID(4)
	schema := &cli.Schema{
		Values: map[cli.ValueID]cli.ValueSchema{4: {
			ID:        4,
			Sensitive: true,
		}},
	}
	declared, ok := value.DeclaredSensitive().(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("DeclaredSensitive expectation does not expose schema evaluation")
	}
	if got := declared.evaluateSchema(schema); got.status != evaluationProved {
		t.Fatalf("DeclaredSensitive = %#v, want proved", got)
	}

	effectiveFalse := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{4: {
			ID:        4,
			Sensitive: false,
		}},
	}
	if got := inspectionExpectationOutcome(
		t,
		value.Sensitive(),
		effectiveFalse,
	); got.status != evaluationDisproved {
		t.Fatalf("effective Sensitive false = %#v, want disproved", got)
	}
	effectiveTrue := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{4: {
			ID:        4,
			Sensitive: true,
		}},
	}
	if got := inspectionExpectationOutcome(t, value.Sensitive(), effectiveTrue); got.status != evaluationProved {
		t.Fatalf("effective Sensitive true = %#v, want proved", got)
	}
}

func TestCommandArgumentConvenienceRequiresExactlyOneForm(t *testing.T) {
	t.Parallel()
	root := RootCommand()
	name := root.Argument("environment")
	literal := root.ArgumentLiteral("--")

	oneFormSchema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			1: {
				ID:            1,
				Path:          nil,
				ArgumentForms: []cli.ArgumentFormID{10},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{
			10: {
				ID:      10,
				Command: 1,
				Terms:   []cli.ArgumentTermID{20, 21},
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
			20: {
				ID:   20,
				Form: 10,
				Name: "environment",
			},
			21: {
				ID:      21,
				Form:    10,
				Literal: "--",
			},
		},
	}
	for _, test := range []struct {
		label   string
		locator ArgumentTermLocator
	}{
		{
			label:   "name",
			locator: name,
		},
		{
			label:   "literal",
			locator: literal,
		},
	} {
		label := test.label
		locator := test.locator
		resolved := resolveSchemaArgumentTermLocator(locator.state, oneFormSchema)
		if resolved.status != locatorResolutionFound {
			t.Fatalf("%s one-form resolution = %#v, %v, %q", label, resolved.value, resolved.status, renderEvaluationField(resolved.detail))
		}
	}
	oneFormReference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			1: {
				ID:            1,
				Path:          nil,
				ArgumentForms: []cli.ArgumentFormID{10},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormReference{
			10: {
				ID:      10,
				Command: 1,
				Terms:   []cli.ArgumentTermID{20, 21},
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermReference{
			20: {
				ID:   20,
				Form: 10,
				Name: "environment",
			},
			21: {
				ID:      21,
				Form:    10,
				Literal: "--",
			},
		},
	}
	for _, test := range []struct {
		label   string
		locator ArgumentTermLocator
	}{
		{
			label:   "name",
			locator: name,
		},
		{
			label:   "literal",
			locator: literal,
		},
	} {
		label := test.label
		locator := test.locator
		resolved := resolveReferenceArgumentTermLocator(locator.state, oneFormReference)
		if resolved.status != locatorResolutionFound {
			t.Fatalf("%s one-form reference resolution = %#v, %v, %q", label, resolved.value, resolved.status, renderEvaluationField(resolved.detail))
		}
	}
	if resolved := resolveSchemaArgumentTermLocator(root.Argument("missing").state, oneFormSchema); resolved.status != locatorResolutionAbsent {
		t.Fatalf("one-form missing status = %v, want absent", resolved.status)
	}

	zeroForm := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID:   1,
			Path: nil,
		}},
	}
	if resolved := resolveSchemaArgumentTermLocator(name.state, zeroForm); resolved.status != locatorResolutionAbsent {
		t.Fatalf("zero-form status = %v, want absent", resolved.status)
	}

	multiForm := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			1: {
				ID:            1,
				Path:          nil,
				ArgumentForms: []cli.ArgumentFormID{10, 11},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{
			10: {
				ID:      10,
				Command: 1,
				Terms:   []cli.ArgumentTermID{20},
			},
			11: {
				ID:      11,
				Command: 1,
				Terms:   []cli.ArgumentTermID{22},
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
			20: {
				ID:   20,
				Form: 10,
				Name: "environment",
			},
			22: {
				ID:   22,
				Form: 11,
				Name: "other",
			},
		},
	}
	if resolved := resolveSchemaArgumentTermLocator(name.state, multiForm); resolved.status != locatorResolutionAmbiguous ||
		!strings.Contains(renderEvaluationField(resolved.detail), "select one explicitly") {
		t.Fatalf("multi-form convenience = %v %q, want ambiguous", resolved.status, renderEvaluationField(resolved.detail))
	}
	if resolved := resolveSchemaArgumentTermLocator(root.ArgumentFormAt(0).Term("environment").state, multiForm); resolved.status != locatorResolutionFound ||
		resolved.value.ID != 20 {
		t.Fatalf("explicit form resolution = %#v %v %q", resolved.value, resolved.status, renderEvaluationField(resolved.detail))
	}
}

func TestCommandArgumentConveniencePreservesInspectionCompleteness(t *testing.T) {
	t.Parallel()
	root := RootCommand()
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:            1,
				Path:          nil,
				ArgumentForms: []cli.ArgumentFormID{10},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormInspection{
			10: {
				ID:      10,
				Command: 1,
				Terms:   []cli.ArgumentTermID{20},
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermInspection{20: {
			ID:   20,
			Form: 10,
			Name: "environment",
		}},
	}
	if resolved := resolveInspectionArgumentTermLocator(root.Argument("environment").state, inspection); resolved.status != locatorResolutionFound ||
		resolved.value.ID != 20 {
		t.Fatalf("inspection one-form resolution = %#v %v %q", resolved.value, resolved.status, renderEvaluationField(resolved.detail))
	}
	inspection.Truncated = true
	if resolved := resolveInspectionArgumentTermLocator(root.Argument("missing").state, inspection); resolved.status != locatorResolutionNotProvable ||
		!strings.Contains(renderEvaluationField(resolved.detail), "truncated") {
		t.Fatalf("truncated missing term = %v %q, want not provable", resolved.status, renderEvaluationField(resolved.detail))
	}
	if got := inspectionExpectationOutcome(
		t,
		root.Argument("missing").Value().StateIfObserved(cli.ResolutionStateResolved),
		inspection,
	); got.status != evaluationNotProvable {
		t.Fatalf("StateIfObserved relational incompleteness = %#v, want not provable", got)
	}
}

func TestFailsWithRejectsBareAnyErrorExpectation(t *testing.T) {
	t.Parallel()
	err := invalidDeclarationFailure(t)
	expectation, ok := FailsWith().(failureExpectationValue)
	if !ok {
		t.Fatal("FailsWith must return failureExpectationValue")
	}
	if got := expectation.evaluateFailure("reference", err); got.status != evaluationDisproved {
		t.Fatalf("bare FailsWith = %#v, want disproved", got)
	}
}

func TestOperationGlobalExpectationsReadOnlyAuthoritativeFields(t *testing.T) {
	t.Parallel()
	locale := cli.Locale("en-US")
	inspectionLocale, ok := EffectiveLocale(locale).(effectiveLocaleExpectationValue)
	if !ok {
		t.Fatal("EffectiveLocale must return effectiveLocaleExpectationValue")
	}
	if got := inspectionLocale.evaluateInspection(cli.Inspection{
		Locale: locale,
	}, nil); got.status != evaluationProved {
		t.Fatalf("inspection locale = %#v", got)
	}
	completionLocale, ok := EffectiveLocale(locale).(effectiveLocaleExpectationValue)
	if !ok {
		t.Fatal("EffectiveLocale must return effectiveLocaleExpectationValue")
	}
	if got := completionLocale.evaluateCompletion(cli.Completion{
		Locale: locale,
	}, nil); got.status != evaluationProved {
		t.Fatalf("completion locale = %#v", got)
	}
	referenceLocale, ok := EffectiveLocale(locale).(effectiveLocaleExpectationValue)
	if !ok {
		t.Fatal("EffectiveLocale must return effectiveLocaleExpectationValue")
	}
	if got := referenceLocale.evaluateReference(cli.Reference{
		Locale: locale,
	}, nil); got.status != evaluationProved {
		t.Fatalf("reference locale = %#v", got)
	}

	release, ok := EvaluatedRelease("v2").(releaseExpectationValue)
	if !ok {
		t.Fatal("EvaluatedRelease must return releaseExpectationValue")
	}
	if got := release.evaluateSchema(&cli.Schema{
		EvaluatedRelease: "v2",
	}); got.status != evaluationProved {
		t.Fatalf("schema release = %#v", got)
	}
	if got := release.evaluateReference(cli.Reference{
		EvaluatedRelease: "v2",
	}, nil); got.status != evaluationProved {
		t.Fatalf("reference release = %#v", got)
	}
	if got := release.evaluateInspection(cli.Inspection{
		EvaluatedRelease: "v2",
	}, nil); got.status != evaluationProved {
		t.Fatalf("inspection release = %#v", got)
	}
}

func TestPublicOperationExpectationSpine(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	var providerCalls atomic.Int64
	var completionCalls atomic.Int64
	run := func(context.Context, *config) error { return nil }
	//nolint:unparam // The public callback contract fixes this result signature.
	completeValue := func(context.Context, *config, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		completionCalls.Add(1)
		contribution := cli.CompletionContribution{
			Candidates: []cli.CompletionCandidate{{
				Value: "alpha",
			}},
		}
		return contribution, nil
	}
	advisory := cli.Advisory{
		Code:     "expectation",
		Severity: cli.NoticeSeverityInformation,
		Message:  "expectation",
	}
	//nolint:unparam // The public callback contract fixes this result signature.
	provideCommands := func(_ context.Context, batch *cli.CommandBatch[config]) error {
		providerCalls.Add(1)
		child := cli.Command[config]{
			Name: "child",
			Run:  run,
		}
		batch.Command(child)
		return nil
	}
	declare := func(root *cli.CommandBuilder[config]) {
		root.Flags.String(&root.Config.Value).CompleteBy(completeValue).Long("value")
		root.Advisory.Always(advisory)
		root.Subcommands.CommandsFrom(provideCommands)
	}
	definition := cli.Definition[config]{
		Name:    "expectation-spine",
		Run:     run,
		Declare: declare,
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	inspectionRequest := cli.InspectionRequest{
		Args: []string{"child"},
	}
	inspectionResult := Inspect(t.Context(), app, inspectionRequest)
	inspection, err := inspectionResult.Inspection()
	if err != nil || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf("Inspect = status %v err %v", inspection.Status, err)
	}
	if providerCalls.Load() != 1 {
		t.Fatalf("provider calls for one Inspect = %d, want 1", providerCalls.Load())
	}
	selected := inspection.Selection.Command.Selected
	if selected == 0 {
		t.Fatalf("selection = %#v", inspection.Selection)
	}
	inspectionResult.Require(t, InspectionReady(), CommandByID(selected).Selected(), ExecutionAdmission().Ready())
	if providerCalls.Load() != 1 {
		t.Fatalf("expectation reran provider: %d", providerCalls.Load())
	}

	// A later operation can materialize fresh operation-local state without
	// changing the retained observation from the first result.
	firstSelection := inspection.Selection.Command.Selected
	freshInspectionRequest := cli.InspectionRequest{
		Args: []string{"child"},
	}
	_, _ = Inspect(t.Context(), app, freshInspectionRequest).Inspection()
	retained, _ := inspectionResult.Inspection()
	if retained.Selection.Command.Selected != firstSelection {
		t.Fatalf("retained inspection changed across later operation: %#v", retained.Selection)
	}

	valueInspectionRequest := cli.InspectionRequest{
		Args: []string{"--value", "alpha"},
	}
	valueResult := Inspect(t.Context(), app, valueInspectionRequest)
	valueInspection, err := valueResult.Inspection()
	if err != nil {
		t.Fatalf("Inspect(value): %v", err)
	}
	var valueID cli.ValueID
	for _, flag := range valueInspection.Flags {
		for _, spellingID := range flag.Spellings {
			if valueInspection.FlagSpellings[spellingID].Spelling.Text == "--value" {
				valueID = flag.Destination
			}
		}
	}
	if valueID == 0 {
		t.Fatal("value destination not projected")
	}
	valueResult.Require(
		t,
		ValueByID(valueID).Resolved(),
		Notice(cli.NoticeAdvisory).Advisory("expectation").Present(),
		TokenAt(0).MatchedSpelling("--value"),
	)

	completionRequest := cli.CompletionRequest{
		Args: []string{"--value", ""},
		Cursor: cli.CompletionCursor{
			Argument: 1,
		},
	}
	completionResult := Complete(t.Context(), app, completionRequest)
	completion, err := completionResult.Completion()
	if err != nil || completionCalls.Load() != 1 {
		t.Fatalf("Complete = calls %d err %v result %#v", completionCalls.Load(), err, completion)
	}
	completionResult.Require(t, CompletionItem("alpha").Present())
	if completionCalls.Load() != 1 {
		t.Fatalf("expectation reran completion callback: %d", completionCalls.Load())
	}

	schemaResult := Schema(app)
	if len(schemaResult.Schema().Commands) == 0 {
		t.Fatal("Schema returned no commands")
	}
	schemaResult.Require(t, RootCommand().Exists())

	referenceResult := Reference(app, cli.ReferenceOptions{})
	if _, err := referenceResult.Reference(); err != nil {
		t.Fatalf("Reference: %v", err)
	}
	referenceResult.Require(t, RootCommand().Exists())
}

func TestResponseFilesIntegratesWithInspectAndComplete(t *testing.T) {
	t.Parallel()
	type config struct{}
	definition := cli.Definition[config]{
		Name: "response-integration",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var requests ResponseFileRequests
	files := ResponseFiles{
		Requests: &requests,
	}
	responseRequest := cli.ResponseFileRequest{
		Path: "args",
		Base: "/work",
	}
	responseFile := cli.ResponseFile{
		Key:      "args",
		Revision: cli.Revision("r1"),
	}
	files.Loads = map[cli.ResponseFileRequest]cli.ResponseFile{responseRequest: responseFile}
	responseInspectionRequest := cli.InspectionRequest{
		Args:             []string{"@args"},
		WorkingDirectory: "/work",
		ResponseFiles:    files.Access(),
	}
	inspectResult := Inspect(t.Context(), app, responseInspectionRequest)
	inspection, err := inspectResult.Inspection()
	if err != nil || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf("Inspect response file = status %v err %v", inspection.Status, err)
	}
	expectedLoadRequests := []cli.ResponseFileRequest{responseRequest}
	if got := requests.Loads(); !slices.Equal(got, expectedLoadRequests) {
		t.Fatalf("load requests = %#v", got)
	}

	request := cli.ResponseFileCompletionRequest{
		Path:   "fi",
		Base:   "/work",
		Cursor: 2,
		Replace: cli.CompletionRange{
			Start: 0,
			End:   2,
		},
		Trigger: cli.CompletionTrigger{
			Kind: cli.CompletionTriggerInvoked,
		},
	}
	replace := cli.CompletionRange{
		Start: 0,
		End:   2,
	}
	contribution := cli.CompletionContribution{
		Replace: &replace,
		Candidates: []cli.CompletionCandidate{{
			Value: "file.txt",
			Kind:  cli.CompletionCandidateFile,
		}},
	}
	files.Completions = map[cli.ResponseFileCompletionRequest]cli.CompletionContribution{request: contribution}
	responseCompletionRequest := cli.CompletionRequest{
		Args: []string{"@fi"},
		Cursor: cli.CompletionCursor{
			Argument: 0,
			Byte:     3,
		},
		WorkingDirectory: "/work",
		ResponseFiles:    files.Access(),
	}
	completeResult := Complete(t.Context(), app, responseCompletionRequest)
	completion, err := completeResult.Completion()
	if err != nil || len(completion.Items) != 1 || completion.Items[0].Value != "file.txt" {
		t.Fatalf("Complete response file = %#v err %v", completion, err)
	}
	expectedCompletionRequests := []cli.ResponseFileCompletionRequest{request}
	if got := requests.Completions(); !slices.Equal(got, expectedCompletionRequests) {
		t.Fatalf("completion requests = %#v", got)
	}

	missing := ResponseFiles{}
	missingResponseRequest := cli.InspectionRequest{
		Args:             []string{"@missing"},
		WorkingDirectory: "/work",
		ResponseFiles:    missing.Access(),
	}
	partialResult := Inspect(t.Context(), app, missingResponseRequest)
	partial, partialErr := partialResult.Inspection()
	if partialErr == nil {
		t.Fatalf("missing response file unexpectedly succeeded: %#v", partial)
	}
	report := cli.Diagnostics(partialErr)
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != cli.DiagnosticResponseFileLoadFailure {
		t.Fatalf("missing response file diagnostics = %#v err %v", report, partialErr)
	}
}

func TestSignalSourceIntegratesWithPublicSignalRuntime(t *testing.T) {
	t.Parallel()
	type config struct{ Events <-chan cli.SignalEvent }
	descriptor := cli.SignalDescriptor[string]{
		Signal:   fixtureSignalReload,
		Fidelity: cli.SignalSourceDeliveryFidelityExact,
	}
	sourceOptions := SignalSourceOptions[string]{
		Signals: []cli.SignalDescriptor[string]{descriptor},
	}
	source, control := SignalSource(sourceOptions)
	received := make(chan cli.SignalEvent, 1)
	run := func(ctx context.Context, configuration *config) error {
		select {
		case event := <-configuration.Events:
			received <- event
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-time.After(2 * time.Second):
			return errors.New("signal endpoint delivery timed out")
		}
	}
	declare := func(root *cli.CommandBuilder[config]) {
		registered := root.Signals.Source(source)
		endpoint := root.Signals.Endpoint(&root.Config.Events)
		root.Signals.Policy(
			root.Signals.Match.On(registered.Signal(fixtureSignalReload)).During(cli.SignalPhaseExecution).Each(),
			root.Signals.Action.Emit(endpoint),
		)
	}
	definition := cli.Definition[config]{
		Name:    "signal-integration",
		Run:     run,
		Declare: declare,
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	runDone := make(chan error, 1)
	go func() { runDone <- app.Run(t.Context(), cli.RunRequest{}) }()
	select {
	case <-control.Opened():
	case <-time.After(2 * time.Second):
		t.Fatal("SignalSource was not opened for execution")
	}
	value := cli.SignalNotification[string]{
		Signal:     fixtureSignalReload,
		Origin:     cli.SignalOriginRuntime,
		ObservedAt: time.Unix(10, 0),
	}
	if err := control.Notify(value); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete after signal endpoint delivery")
	}
	select {
	case event := <-received:
		if event.BatchSize != 1 || event.FirstOccurrence != 1 || event.LastOccurrence != 1 {
			t.Fatalf("emitted signal event = %#v", event)
		}
	default:
		t.Fatal("Run completed without exposing the routed signal event")
	}
	select {
	case <-control.Closed():
	case <-time.After(2 * time.Second):
		t.Fatal("SignalSource did not close after Run")
	}
}

func TestHostFixturesAreInstanceLocal(t *testing.T) {
	t.Parallel()
	var firstRequests ResponseFileRequests
	var secondRequests ResponseFileRequests
	firstFiles := ResponseFiles{
		Requests: &firstRequests,
	}
	secondFiles := ResponseFiles{
		Requests: &secondRequests,
	}
	firstRequest := cli.ResponseFileRequest{
		Path: "first",
	}
	secondRequest := cli.ResponseFileRequest{
		Path: "second",
	}
	firstFile := cli.ResponseFile{
		Content: []byte("one"),
	}
	secondFile := cli.ResponseFile{
		Content: []byte("two"),
	}
	firstFiles.Loads = map[cli.ResponseFileRequest]cli.ResponseFile{firstRequest: firstFile}
	secondFiles.Loads = map[cli.ResponseFileRequest]cli.ResponseFile{secondRequest: secondFile}

	_, firstSignal := SignalSource(SignalSourceOptions[string]{})
	_, secondSignal := SignalSource(SignalSourceOptions[string]{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		file, err := firstFiles.Access().Load(t.Context(), firstRequest)
		if err != nil || string(file.Content) != "one" {
			t.Errorf("first fixture = %q, %v", file.Content, err)
		}
	}()
	go func() {
		defer wg.Done()
		file, err := secondFiles.Access().Load(t.Context(), secondRequest)
		if err != nil || string(file.Content) != "two" {
			t.Errorf("second fixture = %q, %v", file.Content, err)
		}
	}()
	wg.Wait()
	if len(firstRequests.Loads()) != 1 || len(secondRequests.Loads()) != 1 {
		t.Fatalf("fixture logs crossed: first=%#v second=%#v", firstRequests.Loads(), secondRequests.Loads())
	}
	if firstSignal == secondSignal {
		t.Fatal("distinct SignalSource fixtures share identity")
	}
}

func isCommandmentPrivateImport(importPath string) bool {
	return strings.HasPrefix(importPath, "github.com/abstractionsinc/commandment/cli/internal/")
}

func TestCommandmentPrivateImportGuardrailTracksLivePrivateNamespace(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		importPath string
		private    bool
	}{
		{
			name:       "live private application",
			importPath: "github.com/abstractionsinc/commandment/cli/internal/application",
			private:    true,
		},
		{
			name:       "live private invocation",
			importPath: "github.com/abstractionsinc/commandment/cli/internal/invocation",
			private:    true,
		},
		{
			name:       "public cli",
			importPath: "github.com/abstractionsinc/commandment/cli",
			private:    false,
		},
		{
			name:       "public clitest",
			importPath: "github.com/abstractionsinc/commandment/clitest",
			private:    false,
		},
		{
			name:       "retired root internal namespace",
			importPath: "github.com/abstractionsinc/commandment/internal/application",
			private:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := isCommandmentPrivateImport(test.importPath); got != test.private {
				t.Fatalf("isCommandmentPrivateImport(%q) = %t, want %t", test.importPath, got, test.private)
			}
		})
	}
}

func TestProductionClitestImportsNoCommandmentInternalPackages(t *testing.T) {
	t.Parallel()
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if isCommandmentPrivateImport(importPath) {
				t.Fatalf("production clitest file %s imports private authority %s", path, importPath)
			}
		}
	}
}

func TestExecutionSignalAndInputExpectationsMatchInspection(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		ExecutionAdmission: cli.ExecutionAdmissionInspection{
			Status:          cli.ExecutionAdmissionBlocked,
			SignalPolicies:  []cli.SignalPolicyID{5},
			SignalEndpoints: []cli.SignalEndpointID{6},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
			5: {
				ID:     5,
				Status: cli.SignalPolicyStatusReady,
			},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
			6: {
				Status: cli.SignalEndpointStatusReady,
			},
		},
		Tokens: []cli.TokenInspection{{
			Redacted: true,
		}},
	}
	checks := []InspectionExpectation{
		ExecutionAdmission().Status(cli.ExecutionAdmissionBlocked),
		ExecutionAdmission().Blocked(),
		ExecutionAdmission().SignalPolicyAt(0).Status(cli.SignalPolicyStatusReady),
		ExecutionAdmission().SignalEndpointAt(0).Status(cli.SignalEndpointStatusReady),
		TokenAt(0).Redacted(),
	}
	for index, expectation := range checks {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("execution/input expectation %d = %#v", index, got)
		}
	}

	if got := inspectionExpectationOutcome(
		t,
		ExecutionAdmission().Status(cli.ExecutionAdmissionReady),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("execution admission status mismatch = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		ExecutionAdmission().SignalPolicyAt(1).Status(cli.SignalPolicyStatusReady),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("missing signal-policy coordinate = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		ExecutionAdmission().SignalEndpointAt(1).Status(cli.SignalEndpointStatusReady),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("missing signal-endpoint coordinate = %#v, want disproved", got)
	}
}

func TestInspectionIncompleteExpectationMatchesOnlyIncompleteStatus(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Status: cli.InspectionStatusIncomplete,
	}
	if got := inspectionExpectationOutcome(
		t,
		InspectionIncomplete(),
		inspection,
	); got.status != evaluationProved {
		t.Fatalf("incomplete inspection = %#v", got)
	}
	inspection.Status = cli.InspectionStatusReady
	if got := inspectionExpectationOutcome(
		t,
		InspectionIncomplete(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("ready inspection under incomplete expectation = %#v, want disproved", got)
	}
}
