package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestTokenPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated: true,
			Tokens:    []cli.TokenInspection{{}},
		}
		got := inspectionExpectationOutcome(t, TokenAt(1).Role(cli.TokenRole(1)), inspection)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v, want not provable", got.status)
		}
	})
	t.Run("tokenproperty unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Tokens: []cli.TokenInspection{{}},
		}
		got := inspectionExpectationOutcome(t, TokenAt(1).Role(cli.TokenRole(1)), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v, want unsatisfied", got.status)
		}
	})
	t.Run("redacted value: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Tokens: []cli.TokenInspection{{
				Role:     cli.TokenRole(1),
				Redacted: true,
			}},
		}
		got := inspectionExpectationOutcome(t, TokenAt(0).Role(cli.TokenRole(2)), inspection)
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.expected), "redacted") ||
			!strings.Contains(renderEvaluationField(got.observed), "redacted") {
			t.Fatalf("outcome = %#v, want redacted mismatch", got)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Tokens: []cli.TokenInspection{{Role: cli.TokenRole(1)}},
		}
		got := inspectionExpectationOutcome(t, TokenAt(0).Role(cli.TokenRole(2)), inspection)
		if got.status != evaluationDisproved || strings.Contains(renderEvaluationField(got.expected), "redacted") {
			t.Fatalf("outcome = %#v, want ordinary mismatch", got)
		}
	})
}

func TestTokenAtResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("token: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		_, status, _, actual := resolveTokenAt(nil, cli.Inspection{})
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid token locator" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		_, status, _, actual := resolveTokenAt(tokenIndexLocator(1).state, cli.Inspection{
			Truncated: true,
		})
		if status != locatorResolutionNotProvable || renderEvaluationField(actual) != "inspection is truncated" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
}

func TestTokenOwnerPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		owners := []TokenOwner{
			CommandEntryByID(cli.CommandEntryID(1)),
			ActionEntryByID(cli.ActionEntryID(2)),
			FlagSpellingByID(cli.FlagSpellingID(3)),
			ArgumentFormByID(cli.ArgumentFormID(4)),
			ArgumentTermByID(cli.ArgumentTermID(5)),
			EnvironmentInputByID(cli.EnvironmentInputID(6)),
			ValueDomainEntryByID(cli.ValueDomainEntryID(7)),
			FactByID(cli.ValueID(8)),
		}
		for _, owner := range owners {
			expectation, ok := tokenOwnerExpectation(TokenAt(0), owner).(tokenOwnerExpectationValue)
			if !ok {
				t.Fatal("tokenOwnerExpectation must return tokenOwnerExpectationValue")
			}
			if expectation.owner == nil {
				t.Fatalf("owner %T was not snapshotted", owner)
			}
		}
	})
}

func TestTokenOwnerPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, TokenAt(1).OwnedBy(ValueByID(2)), cli.Inspection{
			Truncated: true,
		})
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("tokenowner unresolved: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := tokenOwnerExpectationValue{
			token: nil,
			owner: ValueByID(2),
		}.evaluateInspection(cli.Inspection{}, nil)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("tokenowner unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		owner := RootCommand().Flag("--missing").Spelling("-m")
		inspection := cli.Inspection{
			Tokens: []cli.TokenInspection{{}},
			Commands: map[cli.CommandID]cli.CommandInspection{
				1: {
					ID:                    1,
					Path:                  []string{},
					NamespaceCompleteness: cli.CommandNamespaceComplete,
				},
			},
		}
		got := inspectionExpectationOutcome(t, TokenAt(0).OwnedBy(owner), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("command id: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Tokens:   []cli.TokenInspection{{}},
			Commands: map[cli.CommandID]cli.CommandInspection{},
		}
		got := inspectionExpectationOutcome(t, TokenAt(0).OwnedBy(Command("missing")), inspection)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("command id: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		commands := map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:                    1,
				Path:                  []string{},
				Children:              []cli.CommandID{2},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
			2: {
				ID:                    2,
				Parent:                1,
				Name:                  "other",
				Path:                  []string{"other"},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		}
		inspection := cli.Inspection{
			Tokens:   []cli.TokenInspection{{}},
			Commands: commands,
		}
		got := inspectionExpectationOutcome(t, TokenAt(0).OwnedBy(Command("missing")), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("value id: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Tokens: []cli.TokenInspection{{
				Destination: 1,
				Redacted:    true,
			}},
		}
		got := inspectionExpectationOutcome(t, TokenAt(0).OwnedBy(ValueByID(2)), inspection)
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.expected), "redacted") {
			t.Fatalf("outcome = %#v", got)
		}
		firstInspection := cli.Inspection{
			Tokens: []cli.TokenInspection{{Destination: 1}},
		}
		got = inspectionExpectationOutcome(t, TokenAt(0).OwnedBy(ValueByID(2)), firstInspection)
		if got.status != evaluationDisproved || strings.Contains(renderEvaluationField(got.expected), "redacted") {
			t.Fatalf("outcome = %#v", got)
		}
	})
}

func TestExecutionHandlerPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("executionhandler unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		commands := map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:                    1,
				Path:                  []string{},
				Children:              nil,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		}
		got := inspectionExpectationOutcome(t, ExecutionAdmission().Handler(Command("missing")), cli.Inspection{
			Commands: commands,
		})
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("executionhandler mismatch: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			ExecutionAdmission: cli.ExecutionAdmissionInspection{
				HandlerCommand: 1,
			},
		}
		got := inspectionExpectationOutcome(t, ExecutionAdmission().Handler(CommandByID(2)), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
}

func TestMiddlewarePropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			ExecutionAdmission().MiddlewareAt(1).Scope(cli.MiddlewareScope(1)),
			cli.Inspection{
				Truncated: true,
			},
		)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("middlewareproperty unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			ExecutionAdmission().MiddlewareAt(1).Scope(cli.MiddlewareScope(1)),
			cli.Inspection{},
		)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			ExecutionAdmission: cli.ExecutionAdmissionInspection{
				Middleware: []cli.MiddlewareInspection{{Scope: cli.MiddlewareScope(1)}},
			},
		}
		got := inspectionExpectationOutcome(
			t,
			ExecutionAdmission().MiddlewareAt(0).Scope(cli.MiddlewareScope(2)), inspection,
		)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
}

func TestMiddlewareResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("middleware: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		_, status, _, actual := resolveMiddleware(nil, cli.Inspection{})
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid middleware locator" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		_, status, _, _ := resolveMiddleware(ExecutionAdmission().MiddlewareAt(1).state, cli.Inspection{
			Truncated: true,
		})
		if status != locatorResolutionNotProvable {
			t.Fatalf("status = %v", status)
		}
	})
	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		_, status, _, actual := resolveMiddleware(ExecutionAdmission().MiddlewareAt(1).state, cli.Inspection{
			Truncated: true,
		})
		if status != locatorResolutionNotProvable || renderEvaluationField(actual) != "inspection is truncated" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("middleware is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		_, status, _, actual := resolveMiddleware(ExecutionAdmission().MiddlewareAt(1).state, cli.Inspection{})
		if status != locatorResolutionAbsent || renderEvaluationField(actual) != "middleware is absent" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
}

func TestMiddlewareCommandPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("configured command: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, ExecutionAdmission().MiddlewareAt(1).Command(CommandByID(1)), cli.Inspection{
			Truncated: true,
		})
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("middlewarecommand unresolved: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, ExecutionAdmission().MiddlewareAt(1).Command(CommandByID(1)), cli.Inspection{})
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("middlewarecommand unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			ExecutionAdmission: cli.ExecutionAdmissionInspection{
				Middleware: []cli.MiddlewareInspection{{}},
			},
			Commands: map[cli.CommandID]cli.CommandInspection{
				1: {
					ID:                    1,
					Path:                  []string{},
					NamespaceCompleteness: cli.CommandNamespaceComplete,
				},
			},
		}
		got := inspectionExpectationOutcome(t, ExecutionAdmission().MiddlewareAt(0).Command(Command("missing")), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("middlewarecommand mismatch: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			ExecutionAdmission: cli.ExecutionAdmissionInspection{
				Middleware: []cli.MiddlewareInspection{{Command: 1}},
			},
		}
		got := inspectionExpectationOutcome(t, ExecutionAdmission().MiddlewareAt(0).Command(CommandByID(2)), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
}

func TestExecutionAdmissionMiddlewareExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	t.Run("execution admission middleware: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			ExecutionAdmission().MiddlewareExactly(ExecutionAdmission().MiddlewareAt(0)),
			cli.Inspection{},
		)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("expected subject unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			ExecutionAdmission: cli.ExecutionAdmissionInspection{
				Middleware: []cli.MiddlewareInspection{{}},
			},
		}
		got := inspectionExpectationOutcome(t, ExecutionAdmission().MiddlewareExactly(MiddlewareLocator{}), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("invalid locator status = %v", got.status)
		}
		inspection.ExecutionAdmission.Middleware = []cli.MiddlewareInspection{{}, {}}
		got = inspectionExpectationOutcome(
			t,
			ExecutionAdmission().MiddlewareExactly(ExecutionAdmission().MiddlewareAt(1), ExecutionAdmission().MiddlewareAt(0)),
			inspection,
		)
		if got.status != evaluationDisproved {
			t.Fatalf("wrong order status = %v", got.status)
		}
	})
}

func TestSignalPolicyIDResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("invalid signal-policy locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		_, status, actual := resolveTypedSignalPolicyID(nil, cli.Inspection{})
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid signal-policy locator" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("execution admission signal policy index: incomplete observation", func(t *testing.T) {
		t.Parallel()
		_, status, actual := resolveTypedSignalPolicyID(ExecutionAdmission().SignalPolicyAt(1).state, cli.Inspection{
			Truncated: true,
		})
		if status != locatorResolutionNotProvable || renderEvaluationField(actual) != "inspection is truncated" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("execution admission signal policy index: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		state := &signalPolicyLocatorState{}
		_, status, actual := resolveTypedSignalPolicyID(state, cli.Inspection{})
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid signal-policy locator" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("command signal policy relation: resolves against inspection", func(t *testing.T) {
		t.Parallel()
		const policyID cli.SignalPolicyID = 17
		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{
				1: {
					ID:             1,
					Path:           nil,
					SignalPolicies: []cli.SignalPolicyID{policyID},
				},
			},
		}
		id, status, actual := resolveTypedSignalPolicyID(RootCommand().SignalPolicyAt(0).state, inspection)
		if status != locatorResolutionFound || renderEvaluationField(actual) != "" || id != policyID {
			t.Fatalf("id/status/detail = %v, %v, %q", id, status, actual)
		}
	})
}

func TestSignalEndpointIDResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("invalid signal-endpoint locator: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		_, status, actual := resolveTypedSignalEndpointID(nil, cli.Inspection{})
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid signal-endpoint locator" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("execution admission signal endpoint index: incomplete observation", func(t *testing.T) {
		t.Parallel()
		_, status, actual := resolveTypedSignalEndpointID(ExecutionAdmission().SignalEndpointAt(1).state, cli.Inspection{
			Truncated: true,
		})
		if status != locatorResolutionNotProvable || renderEvaluationField(actual) != "inspection is truncated" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("execution admission signal endpoint index: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		state := &signalEndpointLocatorState{}
		_, status, actual := resolveTypedSignalEndpointID(state, cli.Inspection{})
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid signal-endpoint locator" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
}

func requireExecutionAdmissionExactMembershipMismatch(t *testing.T, subject string, empty, invalid, mismatch InspectionExpectation, inspection cli.Inspection) {
	t.Helper()
	t.Run(subject+": semantic mismatch", func(t *testing.T) {
		t.Parallel()
		if got := inspectionExpectationOutcome(t, empty, cli.Inspection{}); got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("expected subject unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		if got := inspectionExpectationOutcome(t, invalid, inspection); got.status != evaluationDisproved {
			t.Fatalf("invalid locator status = %v", got.status)
		}
		if got := inspectionExpectationOutcome(t, mismatch, inspection); got.status != evaluationDisproved {
			t.Fatalf("mismatch status = %v", got.status)
		}
	})
}

func TestExecutionAdmissionSignalPoliciesExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{ExecutionAdmission: cli.ExecutionAdmissionInspection{SignalPolicies: []cli.SignalPolicyID{1}}}
	requireExecutionAdmissionExactMembershipMismatch(
		t,
		"execution admission signal policies",
		ExecutionAdmission().SignalPoliciesExactly(SignalPolicyByID(1)),
		ExecutionAdmission().SignalPoliciesExactly(SignalPolicyLocator{}),
		ExecutionAdmission().SignalPoliciesExactly(SignalPolicyByID(2)), inspection,
	)
}

func TestExecutionAdmissionSignalEndpointsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{ExecutionAdmission: cli.ExecutionAdmissionInspection{SignalEndpoints: []cli.SignalEndpointID{1}}}
	requireExecutionAdmissionExactMembershipMismatch(
		t,
		"execution admission signal endpoints",
		ExecutionAdmission().SignalEndpointsExactly(SignalEndpointByID(1)),
		ExecutionAdmission().SignalEndpointsExactly(SignalEndpointLocator{}),
		ExecutionAdmission().SignalEndpointsExactly(SignalEndpointByID(2)), inspection,
	)
}

func TestSignalPolicyInspectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			ExecutionAdmission().SignalPolicyAt(1).Status(cli.SignalPolicyStatus(1)),
			cli.Inspection{
				Truncated: true,
			},
		)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("signalpolicyinspection unresolved: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated: true,
			ExecutionAdmission: cli.ExecutionAdmissionInspection{
				SignalPolicies: []cli.SignalPolicyID{1},
			},
		}
		got := inspectionExpectationOutcome(t, SignalPolicyByID(1).Status(cli.SignalPolicyStatus(1)), inspection)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated:      true,
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{},
		}
		got := inspectionExpectationOutcome(t, SignalPolicyByID(1).Status(cli.SignalPolicyStatus(1)), inspection)
		if got.status != evaluationNotProvable || renderEvaluationField(got.observed) != "inspection is truncated" {
			t.Fatalf("outcome = %#v", got)
		}
	})
	t.Run("signal policy is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, SignalPolicyByID(1).Status(cli.SignalPolicyStatus(1)), cli.Inspection{})
		if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "signal policy is absent" {
			t.Fatalf("outcome = %#v", got)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
				1: {
					ID:     1,
					Status: cli.SignalPolicyStatus(1),
				},
			},
		}
		got := inspectionExpectationOutcome(t, SignalPolicyByID(1).Status(cli.SignalPolicyStatus(2)), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("redacted value: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
				1: {
					ID:        1,
					Status:    cli.SignalPolicyStatus(1),
					Sensitive: true,
				},
			},
		}
		got := inspectionExpectationOutcome(t, SignalPolicyByID(1).Status(cli.SignalPolicyStatus(2)), inspection)
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.expected), "redacted") {
			t.Fatalf("outcome = %#v", got)
		}
	})
}

func TestSignalEndpointInspectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			ExecutionAdmission().SignalEndpointAt(1).Status(cli.SignalEndpointStatus(1)),
			cli.Inspection{
				Truncated: true,
			},
		)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("signalendpointinspection unresolved: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated: true,
			ExecutionAdmission: cli.ExecutionAdmissionInspection{
				SignalEndpoints: []cli.SignalEndpointID{1},
			},
		}
		got := inspectionExpectationOutcome(t, SignalEndpointByID(1).Status(cli.SignalEndpointStatus(1)), inspection)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			SignalEndpointByID(1).Status(cli.SignalEndpointStatus(1)),
			cli.Inspection{
				Truncated: true,
			},
		)
		if got.status != evaluationNotProvable || renderEvaluationField(got.observed) != "inspection is truncated" {
			t.Fatalf("outcome = %#v", got)
		}
	})
	t.Run("signal endpoint is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			SignalEndpointByID(1).Status(cli.SignalEndpointStatus(1)),
			cli.Inspection{},
		)
		if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "signal endpoint is absent" {
			t.Fatalf("outcome = %#v", got)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
				1: {Status: cli.SignalEndpointStatus(1)},
			},
		}
		got := inspectionExpectationOutcome(t, SignalEndpointByID(1).Status(cli.SignalEndpointStatus(2)), inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
	})
}

func TestSignalEndpointPoliciesExactPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	t.Run("exact authoritative policy collection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			ExecutionAdmission().SignalEndpointAt(1).EffectivePoliciesExactly(),
			cli.Inspection{
				Truncated: true,
			},
		)
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("signalendpointpoliciesexact unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, SignalEndpointLocator{}.EffectivePoliciesExactly(), cli.Inspection{})
		if got.status != evaluationDisproved {
			t.Fatalf("status = %v", got.status)
		}
		inspection := cli.Inspection{
			SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
				1: {EffectivePolicies: []cli.SignalPolicyID{1}},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{},
		}
		got = inspectionExpectationOutcome(
			t,
			SignalEndpointByID(1).EffectivePoliciesExactly(SignalPolicyLocator{}),
			inspection,
		)
		if got.status != evaluationDisproved {
			t.Fatalf("policy resolution status = %v", got.status)
		}
		inspection.SignalEndpoints[1] = cli.SignalEndpointInspection{
			EffectivePolicies: []cli.SignalPolicyID{1},
		}
		got = inspectionExpectationOutcome(
			t,
			SignalEndpointByID(1).EffectivePoliciesExactly(SignalPolicyByID(2)),
			inspection,
		)
		if got.status != evaluationDisproved {
			t.Fatalf("policy mismatch status = %v", got.status)
		}
	})
	t.Run("signalendpointpoliciesexact unresolved: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, SignalEndpointByID(1).EffectivePoliciesExactly(), cli.Inspection{
			Truncated: true,
		})
		if got.status != evaluationNotProvable {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, SignalEndpointByID(1).EffectivePoliciesExactly(), cli.Inspection{
			Truncated: true,
		})
		if got.status != evaluationNotProvable {
			t.Fatalf("absent endpoint status = %v", got.status)
		}
		inspection := cli.Inspection{
			Truncated:       true,
			SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{1: {}},
		}
		got = inspectionExpectationOutcome(t, SignalEndpointByID(1).EffectivePoliciesExactly(), inspection)
		if got.status != evaluationNotProvable {
			t.Fatalf("present endpoint status = %v", got.status)
		}
	})
	t.Run("existing signal endpoint: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(t, SignalEndpointByID(1).EffectivePoliciesExactly(), cli.Inspection{})
		if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "signal endpoint is absent" {
			t.Fatalf("outcome = %#v", got)
		}
	})
}

func TestExecutionAndSignalIntegrationFactoryRootsAreValid(t *testing.T) {
	t.Parallel()

	admission := ExecutionAdmission()
	if admission.state == nil {
		t.Fatal("ExecutionAdmission() returned an invalid root locator")
	}
	if middleware := admission.MiddlewareAt(0); middleware.state == nil || !middleware.state.valid {
		t.Fatalf("ExecutionAdmission().MiddlewareAt(0) = %#v, want valid typed coordinate", middleware.state)
	}
	if policy := admission.SignalPolicyAt(0); policy.state == nil || policy.state.mode != signalPolicyLocatorExecutionAdmissionIndex {
		t.Fatalf("ExecutionAdmission().SignalPolicyAt(0) = %#v, want execution-admission-relative coordinate", policy.state)
	}
	if endpoint := admission.SignalEndpointAt(0); endpoint.state == nil || endpoint.state.mode != signalEndpointLocatorExecutionAdmissionIndex {
		t.Fatalf("ExecutionAdmission().SignalEndpointAt(0) = %#v, want execution-admission-relative coordinate", endpoint.state)
	}

	integration := SignalIntegration()
	if integration.state == nil {
		t.Fatal("SignalIntegration() returned an invalid root locator")
	}
}

func TestExecutionAndSignalIntegrationLiteralZeroLocatorsAreInvalid(t *testing.T) {
	t.Parallel()

	if child := (ExecutionAdmissionLocator{}).MiddlewareAt(0); child.state != nil {
		t.Fatalf("zero ExecutionAdmissionLocator child = %#v, want invalid zero locator", child.state)
	}
	if child := (ExecutionAdmissionLocator{}).SignalPolicyAt(0); child.state != nil {
		t.Fatalf("zero ExecutionAdmissionLocator signal policy = %#v, want invalid zero locator", child.state)
	}
	if child := (ExecutionAdmissionLocator{}).SignalEndpointAt(0); child.state != nil {
		t.Fatalf("zero ExecutionAdmissionLocator signal endpoint = %#v, want invalid zero locator", child.state)
	}

	if got := inspectionExpectationOutcome(t, (ExecutionAdmissionLocator{}).Ready(), cli.Inspection{}); got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid execution admission locator" {
		t.Fatalf("zero ExecutionAdmissionLocator expectation = %#v", got)
	}
	if got := inspectionExpectationOutcome(t, (SignalIntegrationLocator{}).Required(), cli.Inspection{}); got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid signal-integration locator" {
		t.Fatalf("zero SignalIntegrationLocator expectation = %#v", got)
	}
}

func TestInputExecutionInternalStateFailsClosed(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Tokens: []cli.TokenInspection{{}},
	}
	if got := (tokenOwnerExpectationValue{
		token: TokenAt(0).state,
	}).evaluateInspection(
		inspection,
		nil,
	); got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid token-owner locator" {
		t.Fatalf("nil owner = %#v", got)
	}
}
