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

// Static value declaration truth and Inspection observation/effective truth are deliberately disjoint.
func TestValueExpectationAxesPreserveOperationApplicability(t *testing.T) {
	t.Parallel()
	for name, expectation := range map[string]any{
		"Exists":            clitest.ValueByID(1).Exists(),
		"DoesNotExist":      clitest.ValueByID(1).DoesNotExist(),
		"DeclaredSensitive": clitest.ValueByID(1).DeclaredSensitive(),
	} {
		if _, ok := expectation.(clitest.StaticDeclarationExpectation); !ok {
			t.Fatalf("%s no longer implements StaticDeclarationExpectation", name)
		}
		if _, ok := expectation.(clitest.InspectionExpectation); ok {
			t.Fatalf("%s unexpectedly implements InspectionExpectation", name)
		}
	}

	for name, expectation := range map[string]any{
		"Observed":    clitest.ValueByID(1).Observed(),
		"NotObserved": clitest.ValueByID(1).NotObserved(),
		"Sensitive":   clitest.ValueByID(1).Sensitive(),
	} {
		if _, ok := expectation.(clitest.InspectionExpectation); !ok {
			t.Fatalf("%s no longer implements InspectionExpectation", name)
		}
		if _, ok := expectation.(clitest.StaticDeclarationExpectation); ok {
			t.Fatalf("%s unexpectedly implements StaticDeclarationExpectation", name)
		}
	}
}

// TokenAt is the positive public spelling for an authoritative token-order coordinate.
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
			return &applicationOwnedError{
				cause: sentinel,
			}
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
