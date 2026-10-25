package clitest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var (
	_ clitest.CompletionExpectation = clitest.CompletionItem("serve").Declaration(clitest.CommandEntryByID(1))
	_ clitest.CompletionExpectation = clitest.CompletionItem("--help").Declaration(clitest.ActionEntryByID(2))
	_ clitest.CompletionExpectation = clitest.CompletionItem("--verbose").Declaration(clitest.FlagSpellingByID(3))
	_ clitest.CompletionExpectation = clitest.CompletionItem("start").Declaration(clitest.ArgumentTermByID(4))
	_ clitest.CompletionExpectation = clitest.CompletionItem("prod").Declaration(clitest.ValueDomainEntryByID(5))

	_ clitest.ConformanceCase = clitest.InspectionCase{}
	_ clitest.ConformanceCase = (*clitest.InspectionCase)(nil)
	_ clitest.ConformanceCase = clitest.CompletionCase{}
	_ clitest.ConformanceCase = (*clitest.CompletionCase)(nil)
	_ clitest.ConformanceCase = clitest.SchemaCase{}
	_ clitest.ConformanceCase = (*clitest.SchemaCase)(nil)
	_ clitest.ConformanceCase = clitest.ReferenceCase{}
	_ clitest.ConformanceCase = (*clitest.ReferenceCase)(nil)
	_ clitest.ConformanceCase = clitest.LocalizationCase{}
	_ clitest.ConformanceCase = (*clitest.LocalizationCase)(nil)
	_                         = clitest.Conformance{
		clitest.InspectionCase{},
		clitest.CompletionCase{},
		clitest.SchemaCase{},
		clitest.ReferenceCase{},
		clitest.LocalizationCase{},
	}

	_ clitest.DiagnosticSubject       = clitest.FlagByID(cli.FlagID(1))
	_ clitest.ValueProducerLocator    = clitest.ValueByID(cli.ValueID(1)).ProducerAt(0)
	_ clitest.CompletionTargetLocator = clitest.CompletionItemAt(0).TargetAt(0)
	_ clitest.HelpCommandGroupEntry   = clitest.CommandEntryByID(cli.CommandEntryID(1))
	_ clitest.HelpCommandGroupEntry   = clitest.RootCommand().ProviderAt(0)

	_ func(clitest.SchemaResult, testing.TB, ...clitest.SchemaExpectation) = clitest.SchemaResult.Require
	_ func() clitest.InspectionExpectation                                 = clitest.InspectionReady
	_ func(cli.FlagID) clitest.FlagLocator                                 = clitest.FlagByID
	_ func(cli.ValueID) clitest.ValueLocator                               = clitest.ValueByID
	_ func(clitest.ArgumentFormLocator) clitest.InspectionExpectation      = clitest.ArgumentFormLocator.Selected
	_ func(clitest.InvocationActionLocator) clitest.InspectionExpectation  = clitest.InvocationActionLocator.Selected
	_ func(clitest.FactLocator) clitest.ReferenceExpectation               = clitest.FactLocator.Available
)

func TestHelpCommandGroupEntryIsClosedToLegalLocatorFamilies(t *testing.T) {
	t.Parallel()

	var unrelated any = clitest.FlagByID(cli.FlagID(1))
	if _, ok := unrelated.(clitest.HelpCommandGroupEntry); ok {
		t.Fatal("unrelated locator unexpectedly implements HelpCommandGroupEntry")
	}
}

func TestConformancePointerCaseFormsAreAdmittedAtPublicBoundary(t *testing.T) {
	t.Parallel()
	cases := []clitest.ConformanceCase{
		&clitest.InspectionCase{},
		&clitest.CompletionCase{},
		&clitest.SchemaCase{},
		&clitest.ReferenceCase{},
		&clitest.LocalizationCase{},
	}
	_ = clitest.Conformance(cases)
}

type consumerInspectionMatcher struct{}

//nolint:unused // Deliberately mirrors the closed package-private method spelling from an external package.
func (consumerInspectionMatcher) inspectionExpectation() {}

func TestConsumerCannotImplementClosedExpectationInterface(t *testing.T) {
	t.Parallel()
	var candidate any = consumerInspectionMatcher{}
	if _, ok := candidate.(clitest.InspectionExpectation); ok {
		t.Fatal("consumer-defined matcher unexpectedly implements closed InspectionExpectation")
	}
}

func TestClosedExpectationFamiliesPreserveOperationApplicability(t *testing.T) {
	t.Parallel()
	var inspectionOnly any = clitest.InspectionReady()
	if _, ok := inspectionOnly.(clitest.CompletionExpectation); ok {
		t.Fatal("InspectionReady unexpectedly applies to Completion")
	}

	var selection any = clitest.RootCommand().Selected()
	if _, ok := selection.(clitest.InspectionExpectation); !ok {
		t.Fatal("selection expectation lost Inspection applicability")
	}
	if _, ok := selection.(clitest.CompletionExpectation); !ok {
		t.Fatal("selection expectation lost Completion applicability")
	}
}

// TestValueExpectationAxesPreserveOperationApplicability keeps static declaration truth disjoint from Inspection observation and effective truth.
func TestValueExpectationAxesPreserveOperationApplicability(t *testing.T) {
	t.Parallel()
	staticCases := []struct {
		name        string
		expectation any
	}{
		{
			name:        "Exists",
			expectation: clitest.ValueByID(1).Exists(),
		},
		{
			name:        "DoesNotExist",
			expectation: clitest.ValueByID(1).DoesNotExist(),
		},
		{
			name:        "DeclaredSensitive",
			expectation: clitest.ValueByID(1).DeclaredSensitive(),
		},
	}
	for _, test := range staticCases {
		name := test.name
		expectation := test.expectation
		if _, ok := expectation.(clitest.StaticDeclarationExpectation); !ok {
			t.Fatalf("%s no longer implements StaticDeclarationExpectation", name)
		}
		if _, ok := expectation.(clitest.InspectionExpectation); ok {
			t.Fatalf("%s unexpectedly implements InspectionExpectation", name)
		}
	}

	inspectionCases := []struct {
		name        string
		expectation any
	}{
		{
			name:        "Observed",
			expectation: clitest.ValueByID(1).Observed(),
		},
		{
			name:        "NotObserved",
			expectation: clitest.ValueByID(1).NotObserved(),
		},
		{
			name:        "Sensitive",
			expectation: clitest.ValueByID(1).Sensitive(),
		},
	}
	for _, test := range inspectionCases {
		name := test.name
		expectation := test.expectation
		if _, ok := expectation.(clitest.InspectionExpectation); !ok {
			t.Fatalf("%s no longer implements InspectionExpectation", name)
		}
		if _, ok := expectation.(clitest.StaticDeclarationExpectation); ok {
			t.Fatalf("%s unexpectedly implements StaticDeclarationExpectation", name)
		}
	}
}

// TestTokenAtPublicCoordinate preserves TokenAt as the positive public spelling for an authoritative token-order coordinate.
func TestTokenAtPublicCoordinate(t *testing.T) {
	t.Parallel()
	var expectation any = clitest.TokenAt(0).Role(cli.TokenRoleCommand)
	if _, ok := expectation.(clitest.InspectionExpectation); !ok {
		t.Fatal("TokenAt coordinate did not produce an InspectionExpectation")
	}
}

type applicationOwnedError struct{ cause error }

func (err *applicationOwnedError) Error() string { return "application-owned: " + err.cause.Error() }
func (err *applicationOwnedError) Unwrap() error { return err.cause }

func TestApplicationOwnedEffectsAndErrorsRemainOrdinaryGo(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("business sentinel")
	effects := 0
	definition := cli.Definition[struct{}]{
		Name: "ordinary-go-boundary",
		Run: func(context.Context, *struct{}) error {
			effects++
			failure := &applicationOwnedError{
				cause: sentinel,
			}
			return failure
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	got := app.Run(t.Context(), cli.RunRequest{})
	if effects != 1 {
		t.Fatalf("application effects = %d, want 1", effects)
	}
	if !errors.Is(got, sentinel) {
		t.Fatalf("errors.Is(%v, sentinel) = false", got)
	}
	var typed *applicationOwnedError
	if !errors.As(got, &typed) ||
		typed.cause != sentinel { //nolint:errorlint,err113 // The wrapper cause must retain the exact sentinel identity.
		t.Fatalf("errors.As(%v) = %#v", got, typed)
	}
	if report := clitest.Diagnostics(got).Report(); len(report.Diagnostics) != 0 {
		t.Fatalf("clitest invented Commandment diagnostics for business error: %#v", report)
	}
}

func TestLocalizationAuditExcludesSynthesizedFrontendCompletionSources(t *testing.T) {
	t.Parallel()

	const source = "known source"
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Description: cli.Description{
			Short: source,
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{
				{
					Locale: "fr",
					Translations: cli.Translations{
						source: "source connue",
					},
				},
			},
		},
	}
	app := clitest.Compile(t, definition)
	policy := clitest.LocalizationPolicy{
		RequireCompleteLocales: []cli.Locale{"fr"},
	}

	clitest.RequireLocalization(t, app, policy)
}

type inputRejectionConfig struct{ Value string }

func inputRejectionObservation(t *testing.T, sensitive bool) (clitest.DiagnosticsResult, cli.InputSpan) {
	t.Helper()

	parse := func(_ context.Context, tokens []string) (string, error) {
		if tokens[0] == "ok" {
			return tokens[0], nil
		}
		if sensitive {
			return "", cli.Rejection.Message(
				"choice",
				"sensitive choice",
				cli.Rejection.At(0, 1, 2),
				cli.Rejection.Expected("alpha", "beta"),
				cli.Rejection.Sensitive(),
			)
		}
		return "", cli.Rejection.Message(
			"choice",
			"invalid choice",
			cli.Rejection.At(0, 1, 2),
			cli.Rejection.Expected("alpha", "beta"),
		)
	}
	format := func(value string) []string { return []string{value} }
	declare := func(root *cli.CommandBuilder[inputRejectionConfig]) {
		value := root.Values.String(&root.Config.Value)
		language := cli.TokenLanguage[string]{
			Name:   "choice",
			Count:  cli.Count.Exactly(1),
			Parse:  parse,
			Format: format,
		}
		root.Arguments.Form(func(form *cli.ArgumentFormBuilder[inputRejectionConfig]) {
			form.Tokens(value, "choice", language)
		})
	}
	definition := cli.Definition[inputRejectionConfig]{
		Name:    "tool",
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"bad"},
	}
	inspection, err := app.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("Inspect() error = nil, want structured rejection")
	}
	if len(inspection.Tokens) != 1 {
		t.Fatalf("Inspection token count = %d, want 1", len(inspection.Tokens))
	}
	origin, ok := inspection.Origins[inspection.Tokens[0].Origin]
	if !ok {
		t.Fatalf("Inspection token origin %v is absent", inspection.Tokens[0].Origin)
	}
	expectedSpan := cli.InputSpan{
		Source: origin.Span.Source,
		Entry:  origin.Span.Entry,
		Start: cli.InputPosition{
			Offset: origin.Span.Start.Offset + 1,
		},
		End: cli.InputPosition{
			Offset: origin.Span.Start.Offset + 2,
		},
	}

	diagnostics := clitest.Diagnostics(err)
	return diagnostics, expectedSpan
}

func TestDiagnosticRejectionRelationPreservesStructuredExactness(t *testing.T) {
	t.Parallel()

	diagnostics, expectedSpan := inputRejectionObservation(t, false)
	locator := clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection()
	diagnostics.Require(
		t,
		locator.Code(cli.InputRejectionCode("choice")),
		locator.ExpectedExactly("alpha", "beta"),
		locator.SpansExactly(expectedSpan),
	)

	sensitiveDiagnostics, sensitiveSpan := inputRejectionObservation(t, true)
	sensitiveDiagnostics.Require(
		t,
		locator.Code(cli.InputRejectionCode("choice")),
		locator.SpansExactly(sensitiveSpan),
	)

	definition := cli.Definition[struct{}]{
		Name: "tool",
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"--unknown"},
	}
	_, unknownFlagErr := app.Inspect(t.Context(), request)
	if unknownFlagErr == nil {
		t.Fatal("Inspect() error = nil, want unknown flag")
	}
	clitest.Diagnostics(unknownFlagErr).Require(
		t,
		clitest.Diagnostic(cli.DiagnosticUnknownFlag).Rejection().DoesNotExist(),
	)
}

func TestDiagnosticRejectionExpectedExactlySnapshotsCallerSlice(t *testing.T) {
	t.Parallel()

	expected := []string{"alpha", "beta"}
	expectation := clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection().ExpectedExactly(expected...)
	expected[0] = "mutated"
	diagnostics, _ := inputRejectionObservation(t, false)

	diagnostics.Require(t, expectation)
}

func TestDiagnosticsResultReportDetachesInputRejectionDetail(t *testing.T) {
	t.Parallel()

	diagnostics, expectedSpan := inputRejectionObservation(t, false)
	first := diagnostics.Report()
	if len(first.Diagnostics) != 1 || first.Diagnostics[0].Rejection == nil {
		t.Fatalf("first Report() = %#v", first)
	}
	first.Diagnostics[0].Rejection.Code = "mutated"
	first.Diagnostics[0].Rejection.Expected[0] = "mutated"
	first.Diagnostics[0].Rejection.Spans[0].Start.Offset = 99
	first.Diagnostics[0].Rejection.Expected = append(first.Diagnostics[0].Rejection.Expected, "extra")

	second := diagnostics.Report()
	observed := second.Diagnostics[0].Rejection
	if observed.Code != "choice" || len(observed.Expected) != 2 || observed.Expected[0] != "alpha" || observed.Expected[1] != "beta" {
		t.Fatalf("second Report() rejection = %#v", observed)
	}
	if len(observed.Spans) != 1 || observed.Spans[0] != expectedSpan {
		t.Fatalf("second Report() spans = %#v, want %#v", observed.Spans, []cli.InputSpan{expectedSpan})
	}
}
