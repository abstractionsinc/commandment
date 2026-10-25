package conformance

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type diagnosticsHeadTailConfig struct {
	Head string
	Tail []string
}

type diagnosticsValueConfig struct{ Value string }

type diagnosticTokenLanguageRejectionValueConfig struct{ Value string }

func TestSensitiveInputRejectionErrorDoesNotExposeMessage(t *testing.T) {
	t.Parallel()
	rejection := cli.Rejection.Message("bad-value", "secret input detail", cli.Rejection.Sensitive())
	if got := rejection.Error(); got != "cli: input rejected" {
		t.Fatalf("Error() = %q, want redacted stable text", got)
	}
	if errors.Is(rejection, errors.New("secret input detail")) {
		t.Fatal("sensitive rejection unexpectedly exposes an ordinary wrapped secret error")
	}
	_, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) { return "", rejection },
		"secret",
	)
	diagnosticProof := clitest.Diagnostics(err)
	diagnosticProof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidValue).Present())
}

func TestInputRejectionRejectsMalformedConstruction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  cli.InputRejection
	}{
		{
			name: "empty code",
			got:  cli.Rejection.Message("", "message"),
		},
		{
			name: "uppercase code",
			got:  cli.Rejection.Message("Bad", "message"),
		},
		{
			name: "leading code hyphen",
			got:  cli.Rejection.Message("-bad", "message"),
		},
		{
			name: "trailing code hyphen",
			got:  cli.Rejection.Message("bad-", "message"),
		},
		{
			name: "double code hyphen",
			got:  cli.Rejection.Message("bad--value", "message"),
		},
		{
			name: "invalid code byte",
			got:  cli.Rejection.Message("bad_value", "message"),
		},
		{
			name: "empty nonsensitive message",
			got:  cli.Rejection.Message("code", ""),
		},
		{
			name: "repeated range",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.At(0, 0, 0), cli.Rejection.At(0, 0, 0)),
		},
		{
			name: "reversed range",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.Across(1, 0, 0, 0)),
		},
		{
			name: "replacement without range",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.ReplaceWith("x")),
		},
		{
			name: "repeated replacement",
			got: cli.Rejection.Message(
				"code",
				"message",
				cli.Rejection.At(0, 0, 0),
				cli.Rejection.ReplaceWith("x"),
				cli.Rejection.ReplaceWith("y"),
			),
		},
		{
			name: "empty hint",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.Hint("")),
		},
		{
			name: "repeated hint",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.Hint("x"), cli.Rejection.Hint("y")),
		},
		{
			name: "duplicate expected",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.Expected("x", "x")),
		},
		{
			name: "duplicate suggested across options",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.Suggest("x"), cli.Rejection.Suggest("x")),
		},
		{
			name: "repeated sensitive",
			got:  cli.Rejection.Message("code", "message", cli.Rejection.Sensitive(), cli.Rejection.Sensitive()),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			inspection, err := diagnosticTokenLanguageInspection(
				t.Context(),
				t,
				func(context.Context, []string) (string, error) { return "", test.got },
				"x",
			)
			if err == nil || !diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInputParserFailure) {
				t.Fatalf(
					"malformed rejection was accepted as structured rejection: err %v diagnostics %#v",
					err,
					inspection.Diagnostics,
				)
			}
			diagnosticProof := clitest.Diagnostics(err)
			diagnosticProof.Require(t, clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present())
		})
	}
}

func TestNormalizeInputRejectionValidatesParserLocalByteRange(t *testing.T) {
	t.Parallel()
	valid := cli.Rejection.Message("bad", "bad input", cli.Rejection.Across(0, 1, 1, 2), cli.Rejection.ReplaceWith())
	inspection, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) { return "", valid },
		"abc",
		"xy",
	)
	if err == nil || !diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInvalidValue) {
		t.Fatalf("valid parser-local range = err %v diagnostics %#v", err, inspection.Diagnostics)
	}
	validProof := clitest.Diagnostics(err)
	validProof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidValue).Present())
	for _, args := range [][]string{{"a", "x"}, {"abc"}} {
		inspection, err = diagnosticTokenLanguageInspection(
			t.Context(),
			t,
			func(context.Context, []string) (string, error) { return "", valid },
			args...,
		)
		if err == nil || !diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInputParserFailure) {
			t.Fatalf(
				"out-of-range parser-local rejection %#v = err %v diagnostics %#v",
				args,
				err,
				inspection.Diagnostics,
			)
		}
		outOfRangeProof := clitest.Diagnostics(err)
		outOfRangeProof.Require(t, clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present())
	}
}

func TestEvaluateTokenLanguageClassifiesAcceptedAndStructuredRejectedOutcomes(t *testing.T) {
	t.Parallel()
	var parserSaw []string
	inspection, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(_ context.Context, tokens []string) (string, error) {
			parserSaw = append([]string(nil), tokens...)
			tokens[0] = "mutated by parser"
			return "accepted", nil
		},
		"original",
	)
	if err != nil || inspection.Status != cli.InspectionStatusReady {
		t.Fatalf(
			"accepted Inspect = status %v, err %v, diagnostics %#v",
			inspection.Status,
			err,
			inspection.Diagnostics,
		)
	}
	acceptedProof := clitest.Diagnostics(err)
	acceptedProof.Require(t, clitest.DiagnosticsExactly())
	if len(parserSaw) != 1 || parserSaw[0] != "original" || len(inspection.Tokens) != 1 ||
		inspection.Tokens[0].Text != "original" {
		t.Fatalf("parser snapshot/input projection = saw %#v, tokens %#v", parserSaw, inspection.Tokens)
	}

	first := cli.Rejection.Message("first", "first", cli.Rejection.At(0, 0, 1))
	second := cli.Rejection.Message("second", "second", cli.Rejection.At(1, 0, 1))
	inspection, err = diagnosticTokenLanguageInspection(t.Context(), t, func(context.Context, []string) (string, error) {
		return "ignored", fmt.Errorf("outer: %w", errors.Join(first, fmt.Errorf("inner: %w", second)))
	}, "a", "b")
	if err == nil || !diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInvalidValue) {
		t.Fatalf("structured rejection = err %v, diagnostics %#v", err, inspection.Diagnostics)
	}
	structuredProof := clitest.Diagnostics(err)
	structuredProof.Require(t,
		clitest.DiagnosticsExactly(clitest.DiagnosticAt(0), clitest.DiagnosticAt(1)),
		clitest.DiagnosticAt(0).Code(cli.DiagnosticInvalidValue),
		clitest.DiagnosticAt(1).Code(cli.DiagnosticInvalidValue),
	)

	canceledCtx, cancel := context.WithCancel(t.Context())
	inspection, err = diagnosticTokenLanguageInspection(canceledCtx, t, func(ctx context.Context, _ []string) (string, error) {
		cancel()
		<-ctx.Done()
		return "", ctx.Err()
	}, "x")
	if err == nil || !errors.Is(err, context.Canceled) ||
		!diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInputParserFailure) {
		t.Fatalf("canceled = err %v, diagnostics %#v", err, inspection.Diagnostics)
	}
	canceledProof := clitest.Diagnostics(err)
	canceledProof.Require(t, clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present())
}

func TestEvaluateTokenLanguageTreatsMixedMalformedAndOperationalErrorTreesAsFailure(t *testing.T) {
	t.Parallel()
	rejection := cli.Rejection.Message("bad", "bad", cli.Rejection.At(0, 0, 1))
	operational := errors.New("backend failed")
	inspection, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) {
			return "", errors.Join(rejection, operational)
		},
		"x",
	)
	if err == nil || !errors.Is(err, operational) ||
		!diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInputParserFailure) {
		t.Fatalf("mixed error = err %v, diagnostics %#v", err, inspection.Diagnostics)
	}
	mixedProof := clitest.Diagnostics(err)
	mixedProof.Require(t, clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present())

	malformed := cli.Rejection.Message("bad", "bad", cli.Rejection.At(1, 0, 1))
	inspection, err = diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) { return "", malformed },
		"x",
	)
	if err == nil || !diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInputParserFailure) {
		t.Fatalf("malformed rejection = err %v, diagnostics %#v", err, inspection.Diagnostics)
	}
	malformedProof := clitest.Diagnostics(err)
	malformedProof.Require(t, clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present())

	var typedNil *cli.InputRejection
	inspection, err = diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) { return "", typedNil },
		"x",
	)
	if err == nil || !diagnosticInspectionHasDiagnosticCode(inspection, cli.DiagnosticInputParserFailure) {
		t.Fatalf("typed-nil rejection = err %v, diagnostics %#v", err, inspection.Diagnostics)
	}
	typedNilProof := clitest.Diagnostics(err)
	typedNilProof.Require(t, clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present())
}

func TestEvaluateTokenLanguageRejectsNilParser(t *testing.T) {
	t.Parallel()
	type config = diagnosticsValueConfig
	definition := cli.Definition[config]{
		Name: "nil-parser",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			language := cli.TokenLanguage[string]{
				Name:   "nil-parser",
				Count:  cli.Count.Exactly(1),
				Format: func(value string) []string { return []string{value} },
			}
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language)
			})
		},
	}
	_, err := cli.Compile(definition)
	if err == nil {
		t.Fatal("token language with nil parser compiled")
	}
	diagnosticProof := clitest.Diagnostics(err)
	diagnosticProof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestTokenLanguageRuntimePreservesStructuredRejection(t *testing.T) {
	t.Parallel()
	type config = diagnosticsValueConfig
	var calls atomic.Int32
	language := cli.TokenLanguage[string]{
		Name:  "runtime-value",
		Count: cli.Count.Exactly(1),
		Parse: func(context.Context, []string) (string, error) {
			calls.Add(1)
			return "", cli.Rejection.Message("token-value", "invalid token value", cli.Rejection.At(0, 0, 1))
		},
		Format: func(value string) []string { return []string{value} },
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language)
			})
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"bad"},
	}
	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()))
	if got := calls.Load(); got != 1 {
		t.Fatalf("Parse calls = %d, want 1", got)
	}
}

func TestTokenLanguageRuntimePreservesParserFailure(t *testing.T) {
	t.Parallel()
	type config = diagnosticsValueConfig
	sentinel := errors.New("parser exploded")
	var calls atomic.Int32
	language := cli.TokenLanguage[string]{
		Name:  "runtime-value",
		Count: cli.Count.Exactly(1),
		Parse: func(context.Context, []string) (string, error) {
			calls.Add(1)
			return "", sentinel
		},
		Format: func(value string) []string { return []string{value} },
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language)
			})
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"bad"},
	}
	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present()))
	if got := calls.Load(); got != 1 {
		t.Fatalf("Parse calls = %d, want 1", got)
	}
	_, inspectErr := proof.Inspection()
	if !errors.Is(inspectErr, sentinel) {
		t.Fatalf("Inspect() error = %v, want errors.Is(..., %v)", inspectErr, sentinel)
	}
}

func TestTokenLanguageRuntimePreservesCancellation(t *testing.T) {
	t.Parallel()
	type config = diagnosticsValueConfig
	var calls atomic.Int32
	language := cli.TokenLanguage[string]{
		Name:  "runtime-value",
		Count: cli.Count.Exactly(1),
		Parse: func(ctx context.Context, _ []string) (string, error) {
			calls.Add(1)
			return "", ctx.Err()
		},
		Format: func(value string) []string { return []string{value} },
	}
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language)
			})
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"bad"},
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	proof := clitest.Inspect(ctx, app, request)
	proof.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present()))
	if got := calls.Load(); got != 1 {
		t.Fatalf("Parse calls = %d, want 1", got)
	}
	_, inspectErr := proof.Inspection()
	if !errors.Is(inspectErr, context.Canceled) {
		t.Fatalf("Inspect() error = %v, want errors.Is(..., %v)", inspectErr, context.Canceled)
	}
}

func TestTokenLanguageRejectionDoesNotRetryAlternateStructuralAllocation(t *testing.T) {
	t.Parallel()
	type config = diagnosticsHeadTailConfig
	var calls atomic.Int32
	var received [][]string
	definition := cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			head := root.Values.String(&root.Config.Head)
			tail := root.Values.Sequence(&root.Config.Tail, cli.Value.String())
			language := cli.TokenLanguage[string]{
				Name:  "head",
				Count: cli.Count.Between(1, 2),
				Parse: func(_ context.Context, tokens []string) (string, error) {
					calls.Add(1)
					received = append(received, append([]string(nil), tokens...))
					if len(tokens) == 2 {
						return "", cli.Rejection.Message(
							"fixed-allocation-reject",
							"greedy allocation rejected",
							cli.Rejection.At(0, 0, len(tokens[0])),
						)
					}
					return tokens[0], nil
				},
				Format: func(value string) []string { return []string{value} },
			}
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(head, "head", language).Greedy()
				form.Sequence(tail, "tail", cli.Count.Between(1, 2))
			})
		},
	}
	app := clitest.Compile(t, definition)
	request := cli.InspectionRequest{
		Args: []string{"one", "two", "three"},
	}
	proof := clitest.Inspect(t.Context(), app, request)
	proof.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()))
	if got := calls.Load(); got != 1 {
		t.Fatalf("Parse calls = %d, want no retry", got)
	}
	if !slices.EqualFunc(received, [][]string{{"one", "two"}}, slices.Equal) {
		t.Fatalf("Parse allocations = %#v, want only the greedy frozen allocation", received)
	}
}

func diagnosticTokenLanguageInspection(ctx context.Context, t *testing.T, parse func(context.Context, []string) (string, error), args ...string) (cli.Inspection, error) {
	t.Helper()
	type config = diagnosticTokenLanguageRejectionValueConfig
	definition := cli.Definition[config]{
		Name: "rejection",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			language := cli.TokenLanguage[string]{
				Name:   "external-rejection",
				Count:  cli.Count.Exactly(len(args)),
				Parse:  parse,
				Format: func(value string) []string { return []string{value} },
			}
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language)
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	request := cli.InspectionRequest{
		Args: append([]string(nil), args...),
	}
	inspection, err := app.Inspect(ctx, request)
	return inspection, err //nolint:wrapcheck // Preserve the exact operation error graph for diagnostic assertions.
}

func diagnosticInspectionHasDiagnosticCode(inspection cli.Inspection, code cli.DiagnosticCode) bool {
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestInputRejectionDiagnosticsPreserveJoinedStructureAndDetach(t *testing.T) {
	t.Parallel()
	first := cli.Rejection.Message(
		"bad-first",
		"first token is invalid",
		cli.Rejection.At(0, 0, 1),
		cli.Rejection.Expected("A"),
	)
	second := cli.Rejection.Message(
		"bad-second",
		"second token is invalid",
		cli.Rejection.At(1, 0, 1),
		cli.Rejection.Expected("B"),
	)
	_, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) {
			return "", errors.Join(first, second)
		},
		"x",
		"y",
	)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 2 {
		t.Fatalf("Diagnostics = %d, want 2", len(report.Diagnostics))
	}
	wantCodes := []cli.InputRejectionCode{"bad-first", "bad-second"}
	wantInputs := []string{"x", "y"}
	for index, diagnostic := range report.Diagnostics {
		if diagnostic.Rejection == nil {
			t.Fatalf("Diagnostic %d rejection = nil", index)
		}
		if diagnostic.Rejection.Code != wantCodes[index] {
			t.Fatalf("Diagnostic %d rejection code = %q", index, diagnostic.Rejection.Code)
		}
		if diagnostic.Input != wantInputs[index] || diagnostic.Span == nil || len(diagnostic.Rejection.Spans) != 1 {
			t.Fatalf("Diagnostic %d = %#v", index, diagnostic)
		}
		if diagnostic.Span.Entry != index || diagnostic.Rejection.Spans[0].Entry != index {
			t.Fatalf("Diagnostic %d spans = primary %#v, rejection %#v", index, diagnostic.Span, diagnostic.Rejection.Spans)
		}
	}

	report.Diagnostics[0].Rejection.Expected[0] = "mutated"
	report.Diagnostics[0].Rejection.Spans[0].Start.Offset = 99
	fresh := cli.Diagnostics(err)
	if fresh.Diagnostics[0].Rejection.Expected[0] != "A" || fresh.Diagnostics[0].Rejection.Spans[0].Start.Offset != 0 {
		t.Fatalf("Diagnostics() shared mutable rejection state: %#v", fresh.Diagnostics[0].Rejection)
	}
}

func TestSensitiveInputRejectionRedactsContentButPreservesSafeStructure(t *testing.T) {
	t.Parallel()
	_, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) {
			return "", cli.Rejection.Message(
				"secret-code",
				"secret message",
				cli.Rejection.At(0, 0, 1),
				cli.Rejection.Expected("secret-expected"),
				cli.Rejection.Suggest("secret-suggestion"),
				cli.Rejection.ReplaceWith("secret-replacement"),
				cli.Rejection.Hint("secret-hint"),
				cli.Rejection.Sensitive(),
			)
		},
		"secret-input",
	)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Rejection == nil {
		t.Fatalf("Diagnostics = %#v", report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if !diagnostic.Sensitive || !diagnostic.InputRedacted || diagnostic.Input != "" || diagnostic.Message != "" || diagnostic.Hint != "" {
		t.Fatalf("sensitive Diagnostic leaked content: %#v", diagnostic)
	}
	if diagnostic.Rejection.Code != "secret-code" || len(diagnostic.Rejection.Spans) != 1 || len(diagnostic.Rejection.Expected) != 0 {
		t.Fatalf("sensitive rejection structure = %#v", diagnostic.Rejection)
	}
	if len(diagnostic.Suggestions) != 0 || len(diagnostic.RepairIndexes) != 0 || len(report.Repairs) != 0 {
		t.Fatalf("sensitive rejection leaked suggestions/repair: diagnostic %#v repairs %#v", diagnostic, report.Repairs)
	}
	for _, secret := range []string{"secret-input", "secret message", "secret-expected", "secret-suggestion", "secret-replacement", "secret-hint"} {
		if strings.Contains(fmt.Sprintf("%+v", err), secret) || strings.Contains(diagnosticErrorText(t, err), secret) {
			t.Fatalf("sensitive presentation leaked %q", secret)
		}
	}
}

func contextSensitiveRejection() error {
	return cli.Rejection.Message(
		"context-sensitive",
		"message-secret",
		cli.Rejection.At(0, 0, 1),
		cli.Rejection.Expected("expected-secret"),
		cli.Rejection.Suggest("suggestion-secret"),
		cli.Rejection.ReplaceWith("replacement-secret"),
		cli.Rejection.Hint("hint-secret"),
	)
}

func contextSensitiveTokenLanguage() cli.TokenLanguage[string] {
	language := cli.TokenLanguage[string]{
		Name:  "rejected-token",
		Count: cli.Count.Exactly(1),
		Parse: func(context.Context, []string) (string, error) {
			return "", contextSensitiveRejection()
		},
		Format: func(value string) []string { return []string{value} },
	}
	return language
}

func contextSensitiveValueCodec() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "rejected-value",
		Parse: func(string) (string, error) {
			return "", contextSensitiveRejection()
		},
		Format:   func(value string) string { return value },
		Snapshot: cli.Value.Assign[string](),
	}
	return codec
}

func requireSensitiveRunFailure[C any](t *testing.T, definition cli.Definition[C], request cli.RunRequest, expectRejection bool) {
	t.Helper()
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), request)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 {
		t.Fatalf("Diagnostics = %#v", report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if !diagnostic.Sensitive || !diagnostic.InputRedacted || diagnostic.Input != "" ||
		diagnostic.Message != "" || diagnostic.Hint != "" || len(diagnostic.Suggestions) != 0 {
		t.Fatalf("sensitive Diagnostic = %#v", diagnostic)
	}
	if expectRejection {
		if diagnostic.Rejection == nil || diagnostic.Rejection.Code != "context-sensitive" ||
			len(diagnostic.Rejection.Expected) != 0 || len(diagnostic.Rejection.Spans) != 1 {
			t.Fatalf("sensitive rejection = %#v", diagnostic.Rejection)
		}
	} else if diagnostic.Rejection != nil {
		t.Fatalf("ordinary value failure rejection = %#v", diagnostic.Rejection)
	}
	if len(diagnostic.RepairIndexes) != 0 || len(report.Repairs) != 0 {
		t.Fatalf("sensitive rejection repairs = diagnostic %#v report %#v", diagnostic, report.Repairs)
	}
	debugText := fmt.Sprintf("%+v", err)
	humanText := diagnosticErrorText(t, err)
	for _, secret := range []string{
		"PRIVATE_INPUT",
		"PRIVATE_VALUE",
		"message-secret",
		"expected-secret",
		"suggestion-secret",
		"replacement-secret",
		"hint-secret",
	} {
		if strings.Contains(debugText, secret) {
			t.Fatalf("sensitive debug presentation leaked %q: %q", secret, debugText)
		}
		if strings.Contains(humanText, secret) {
			t.Fatalf("sensitive human presentation leaked %q: %q", secret, humanText)
		}
	}
}

func TestSensitiveDestinationRedactsFlagRejection(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	language := contextSensitiveTokenLanguage()
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value).Sensitive()
			root.Flags.Tokens(value, language).Long("value")
		},
	}
	request := cli.RunRequest{
		Args: []string{"--value", "PRIVATE_INPUT"},
	}
	requireSensitiveRunFailure(t, definition, request, true)
}

func TestSensitiveDestinationRedactsArgumentRejection(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	language := contextSensitiveTokenLanguage()
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value).Sensitive()
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language)
			})
		},
	}
	request := cli.RunRequest{
		Args: []string{"PRIVATE_INPUT"},
	}
	requireSensitiveRunFailure(t, definition, request, true)
}

func TestSensitiveArgumentDeclarationRedactsRejection(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	language := contextSensitiveTokenLanguage()
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language).Sensitive()
			})
		},
	}
	request := cli.RunRequest{
		Args: []string{"PRIVATE_INPUT"},
	}
	requireSensitiveRunFailure(t, definition, request, true)
}

func TestSensitiveFlagSpellingRedactsRejection(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	language := contextSensitiveTokenLanguage()
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Flags.Tokens(value, language).Long("value").Sensitive()
		},
	}
	request := cli.RunRequest{
		Args: []string{"--value", "PRIVATE_INPUT"},
	}
	requireSensitiveRunFailure(t, definition, request, true)
}

func TestSensitiveEnvironmentInputRedactsValueFailure(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	codec := contextSensitiveValueCodec()
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.Value(&root.Config.Value, codec)
			root.Values.Environment(value, "ALIEN_SECRET").Sensitive()
		},
	}
	request := cli.RunRequest{
		Env: cli.Env{
			{
				Name:  "ALIEN_SECRET",
				Value: "PRIVATE_VALUE",
			},
		},
	}
	requireSensitiveRunFailure(t, definition, request, false)
}

func TestSensitiveResponseSourceRedactsRejection(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	language := contextSensitiveTokenLanguage()
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", language)
			})
		},
	}
	request := cli.RunRequest{
		Args: []string{"@secret"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				response := cli.ResponseFile{
					Key:       "secret",
					Revision:  cli.Revision("r1"),
					Content:   []byte("PRIVATE_INPUT"),
					Sensitive: true,
				}
				return response, nil
			},
		},
	}
	requireSensitiveRunFailure(t, definition, request, true)
}

func TestSensitiveParserInputOutsideRejectionRangeRedactsDerivedContent(t *testing.T) {
	t.Parallel()
	const privateInput = "PRIVATE_PARSER_INPUT"
	type config struct{ Value string }
	language := cli.TokenLanguage[string]{
		Name:  "pair",
		Count: cli.Count.Exactly(2),
		Parse: func(_ context.Context, tokens []string) (string, error) {
			rejection := cli.Rejection.Message(
				"mixed-origin",
				"saw "+tokens[0],
				cli.Rejection.At(1, 0, len(tokens[1])),
				cli.Rejection.Expected("expected-"+tokens[0]),
				cli.Rejection.Suggest("suggestion-"+tokens[0]),
				cli.Rejection.ReplaceWith("replacement-"+tokens[0]),
				cli.Rejection.Hint("hint-"+tokens[0]),
			)
			return "", rejection
		},
		Format: func(value string) []string { return []string{value} },
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "pair", language)
			})
		},
	}
	request := cli.RunRequest{
		Args: []string{"@secret", "PUBLIC_INPUT"},
		ResponseFiles: cli.ResponseFileAccess{
			Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
				response := cli.ResponseFile{
					Key:       "secret",
					Revision:  cli.Revision("r1"),
					Content:   []byte(privateInput),
					Sensitive: true,
				}
				return response, nil
			},
		},
	}
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), request)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 {
		t.Fatalf("Diagnostics = %#v", report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if !diagnostic.Sensitive || !diagnostic.InputRedacted || diagnostic.Input != "" ||
		diagnostic.Message != "" || diagnostic.Hint != "" || len(diagnostic.Suggestions) != 0 {
		t.Fatalf("sensitive Diagnostic = %#v", diagnostic)
	}
	if diagnostic.Rejection == nil || diagnostic.Rejection.Code != "mixed-origin" ||
		len(diagnostic.Rejection.Expected) != 0 || len(diagnostic.Rejection.Spans) != 1 {
		t.Fatalf("sensitive rejection = %#v", diagnostic.Rejection)
	}
	if len(diagnostic.Origins) != 1 {
		t.Fatalf("Diagnostic origins = %#v, want one ranged origin", diagnostic.Origins)
	}
	origin, ok := report.Origins[diagnostic.Origins[0]]
	if !ok {
		t.Fatalf("Diagnostic origin %v missing from report", diagnostic.Origins[0])
	}
	source, ok := report.Sources[origin.Span.Source]
	if !ok || source.Kind != cli.InputKindArguments || source.Sensitive {
		t.Fatalf("ranged origin source = %#v, want nonsensitive argv source", source)
	}
	if diagnostic.InputSource != source.ID || diagnostic.Rejection.Spans[0].Source != source.ID {
		t.Fatalf("ranged attribution = source %v input %v span %v", source.ID, diagnostic.InputSource, diagnostic.Rejection.Spans[0].Source)
	}
	if len(diagnostic.RepairIndexes) != 0 || len(report.Repairs) != 0 {
		t.Fatalf("sensitive rejection repairs = diagnostic %#v report %#v", diagnostic, report.Repairs)
	}
	debugText := fmt.Sprintf("%+v", err)
	humanText := diagnosticErrorText(t, err)
	for _, leaked := range []string{
		privateInput,
		"saw " + privateInput,
		"expected-" + privateInput,
		"suggestion-" + privateInput,
		"replacement-" + privateInput,
		"hint-" + privateInput,
	} {
		if strings.Contains(debugText, leaked) {
			t.Fatalf("sensitive debug presentation leaked %q: %q", leaked, debugText)
		}
		if strings.Contains(humanText, leaked) {
			t.Fatalf("sensitive human presentation leaked %q: %q", leaked, humanText)
		}
	}
}

func rejectedSubjectTokenLanguage(rejection error) cli.TokenLanguage[string] {
	language := cli.TokenLanguage[string]{
		Name:   "rejected-token",
		Count:  cli.Count.Exactly(1),
		Parse:  func(context.Context, []string) (string, error) { return "", rejection },
		Format: func(value string) []string { return []string{value} },
	}
	return language
}

func rejectedSubjectValueCodec(rejection error) cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name:     "rejected-value",
		Parse:    func(string) (string, error) { return "", rejection },
		Format:   func(value string) string { return value },
		Snapshot: cli.Value.Assign[string](),
	}
	return codec
}

func requireDiagnosticDeclarationSubject[C any](t *testing.T, definition cli.Definition[C], request cli.RunRequest, kind cli.DeclarationKind, subject string) {
	t.Helper()
	app := clitest.Compile(t, definition)
	err := app.Run(t.Context(), request)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 {
		t.Fatalf("Diagnostics = %#v", report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if diagnostic.Declaration.Kind != kind || diagnostic.Subject != subject {
		t.Fatalf("Diagnostic declaration/subject = %#v / %q, want kind %v / %q", diagnostic.Declaration, diagnostic.Subject, kind, subject)
	}
}

func TestRejectedFlagValueDiagnosticUsesExactSpelling(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	rejection := cli.Rejection.Message("candidate-rejection", "rejected")
	language := rejectedSubjectTokenLanguage(rejection)
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			flag := root.Flags.Tokens(value, language)
			flag.Long("first")
			flag.Long("second")
		},
	}
	request := cli.RunRequest{
		Args: []string{"--second", "bad"},
	}
	requireDiagnosticDeclarationSubject(t, definition, request, cli.DeclarationKindFlagSpelling, "--second")
}

func TestRejectedEnvironmentValueDiagnosticUsesExactProducer(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	rejection := cli.Rejection.Message("candidate-rejection", "rejected")
	codec := rejectedSubjectValueCodec(rejection)
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.Value(&root.Config.Value, codec)
			root.Values.Environment(value, "ALIEN_FIRST")
			root.Values.Environment(value, "ALIEN_SECOND")
		},
	}
	request := cli.RunRequest{
		Env: cli.Env{
			{
				Name:  "ALIEN_SECOND",
				Value: "bad",
			},
		},
	}
	requireDiagnosticDeclarationSubject(t, definition, request, cli.DeclarationKindEnvironmentInput, "ALIEN_SECOND")
}

func TestRejectedArgumentValueDiagnosticUsesExactTerm(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	rejection := cli.Rejection.Message("candidate-rejection", "rejected")
	language := rejectedSubjectTokenLanguage(rejection)
	definition := cli.Definition[config]{
		Name: "alienctl",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Flags.Tokens(value, language).Long("value")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "argument", language)
			})
		},
	}
	request := cli.RunRequest{
		Args: []string{"bad"},
	}
	requireDiagnosticDeclarationSubject(t, definition, request, cli.DeclarationKindArgumentTerm, "<argument>")
}

func TestInputRejectionReplacementProducesDetachedArgvRepair(t *testing.T) {
	t.Parallel()
	_, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) {
			return "", cli.Rejection.Message(
				"dock-code",
				"invalid dock code",
				cli.Rejection.At(0, 0, 2),
				cli.Rejection.ReplaceWith("A1"),
			)
		},
		"ZZ",
	)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 || len(report.Repairs) != 1 {
		t.Fatalf("Diagnostics/Repairs = %d/%d, want 1/1", len(report.Diagnostics), len(report.Repairs))
	}
	diagnostic := report.Diagnostics[0]
	if len(diagnostic.RepairIndexes) != 1 || diagnostic.RepairIndexes[0] != 0 {
		t.Fatalf("RepairIndexes = %v, want [0]", diagnostic.RepairIndexes)
	}
	repair := report.Repairs[0]
	if len(repair.Sources) != 1 || len(repair.Sources[0].TextEdits) != 1 {
		t.Fatalf("repair = %#v", repair)
	}
	edit := repair.Sources[0].TextEdits[0]
	if len(edit.Replacement) != 1 || edit.Replacement[0].Literal != "A1" || edit.Replacement[0].Copy != nil {
		t.Fatalf("replacement = %#v, want literal A1", edit.Replacement)
	}

	report.Repairs[0].Sources[0].TextEdits[0].Replacement[0].Literal = "mutated"
	fresh := cli.Diagnostics(err)
	if fresh.Repairs[0].Sources[0].TextEdits[0].Replacement[0].Literal != "A1" {
		t.Fatalf("Diagnostics() shared mutable repair state: %#v", fresh.Repairs[0])
	}
}

func TestInputRejectionAcrossResponseSourcesPreservesExactSpansWithoutUnsoundRepair(t *testing.T) {
	t.Parallel()
	type config struct{ Pair string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			value := root.Values.String(&root.Config.Pair)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				language := cli.TokenLanguage[string]{
					Name:  "pair",
					Count: cli.Count.Exactly(2),
					Parse: func(context.Context, []string) (string, error) {
						return "", cli.Rejection.Message(
							"bad-pair",
							"pair is invalid",
							cli.Rejection.Across(0, 1, 1, 1),
							cli.Rejection.ReplaceWith("A", "B"),
						)
					},
					Format: func(value string) []string { return []string{value} },
				}
				form.Tokens(value, "pair", language)
			})
		},
	}
	app := clitest.Compile(t, definition)
	responseFiles := cli.ResponseFileAccess{
		Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
			response := cli.ResponseFile{
				Key:      "one",
				Location: "virtual:one",
				Content:  []byte("beta"),
			}
			return response, nil
		},
	}
	request := cli.InspectionRequest{
		Args:          []string{"alpha", "@one"},
		ResponseFiles: responseFiles,
	}
	_, err := app.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Rejection == nil {
		t.Fatalf("Diagnostics = %#v", report.Diagnostics)
	}
	spans := report.Diagnostics[0].Rejection.Spans
	if len(spans) != 2 || spans[0].Source == spans[1].Source {
		t.Fatalf("Rejection.Spans = %#v, want two cross-source spans", spans)
	}
	if len(report.Diagnostics[0].RepairIndexes) != 0 || len(report.Repairs) != 0 {
		t.Fatalf("cross-source replacement produced unsound repair: diagnostic %#v repairs %#v", report.Diagnostics[0], report.Repairs)
	}
}

func TestInputRejectionDeletionProducesEmptyReplacement(t *testing.T) {
	t.Parallel()
	_, err := diagnosticTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) {
			return "", cli.Rejection.Message(
				"remove-input",
				"input is not allowed",
				cli.Rejection.At(0, 0, 2),
				cli.Rejection.ReplaceWith(),
			)
		},
		"ZZ",
	)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 || len(report.Repairs) != 1 {
		t.Fatalf("Diagnostics/Repairs = %d/%d, want 1/1", len(report.Diagnostics), len(report.Repairs))
	}
	edit := report.Repairs[0].Sources[0].TextEdits[0]
	if len(edit.Replacement) != 0 {
		t.Fatalf("deletion replacement = %#v, want empty replacement", edit.Replacement)
	}
}

func TestInputRejectionResponseFileReplacementUsesVerifiedSource(t *testing.T) {
	t.Parallel()
	type config struct{ Code string }
	definition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			value := root.Values.String(&root.Config.Code)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				language := cli.TokenLanguage[string]{
					Name:  "code",
					Count: cli.Count.Exactly(1),
					Parse: func(context.Context, []string) (string, error) {
						return "", cli.Rejection.Message(
							"dock-code",
							"invalid dock code",
							cli.Rejection.At(0, 0, 2),
							cli.Rejection.ReplaceWith("A1"),
						)
					},
					Format: func(value string) []string { return []string{value} },
				}
				form.Tokens(value, "code", language)
			})
		},
	}
	app := clitest.Compile(t, definition)
	responseFiles := cli.ResponseFileAccess{
		Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
			response := cli.ResponseFile{
				Key:      "one",
				Location: "virtual:one",
				Content:  []byte("ZZ"),
			}
			return response, nil
		},
	}
	request := cli.InspectionRequest{
		Args:          []string{"@one"},
		ResponseFiles: responseFiles,
	}
	_, err := app.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 || len(report.Repairs) != 1 {
		t.Fatalf("Diagnostics/Repairs = %d/%d, want 1/1", len(report.Diagnostics), len(report.Repairs))
	}
	repairSource := report.Repairs[0].Sources[0]
	if len(repairSource.TextEdits) != 1 {
		t.Fatalf("repair source = %#v, want one text edit", repairSource)
	}
	edit := repairSource.TextEdits[0]
	if edit.Span.Entry != -1 || len(edit.Replacement) != 1 || edit.Replacement[0].Literal != "A1" {
		t.Fatalf("response-file edit = %#v, want flat-text replacement A1", edit)
	}
	source, ok := report.Sources[repairSource.Source]
	if !ok || source.Kind != cli.InputKindResponseFile || source.Revision == (cli.InputRevision{}) {
		t.Fatalf("response-file source = %#v, present %t, want verified revision", source, ok)
	}
}

func TestInputRejectionPresentationUsesDeclarationLocalizationOwner(t *testing.T) {
	t.Parallel()
	const (
		message = "invalid dock code"
		hint    = "choose a registered dock"
	)
	type config struct{ Value string }
	declareRejectedValue := func(command *cli.CommandBuilder[config]) {
		value := command.Values.String(&command.Config.Value)
		command.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
			language := cli.TokenLanguage[string]{
				Name:  "dock-code",
				Count: cli.Count.Exactly(1),
				Parse: func(context.Context, []string) (string, error) {
					return "", cli.Rejection.Message(
						"dock-code",
						message,
						cli.Rejection.At(0, 0, 2),
						cli.Rejection.Hint(hint),
					)
				},
				Format: func(value string) []string { return []string{value} },
			}
			form.Tokens(value, "dock", language)
		})
	}
	applicationLocalization := cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				message: "code de quai invalide (application)",
				hint:    "choisissez un quai enregistré (application)",
			},
		}},
	}
	applicationDefinition := cli.Definition[config]{
		Name:         "alienctl",
		Localization: applicationLocalization,
		Declare:      declareRejectedValue,
	}
	application := clitest.Compile(t, applicationDefinition)

	assert := func(label string, app cli.Application, locale cli.Locale, wantMessage, wantHint string, args ...string) {
		t.Helper()
		request := cli.InspectionRequest{
			Args:   args,
			Locale: locale,
		}
		_, err := app.Inspect(t.Context(), request)
		if err == nil {
			t.Fatalf("%s Inspect() error = nil", label)
		}
		report := cli.Diagnostics(err)
		if len(report.Diagnostics) != 1 {
			t.Fatalf("%s Diagnostics = %#v", label, report.Diagnostics)
		}
		diagnostic := report.Diagnostics[0]
		if diagnostic.Message != wantMessage || diagnostic.Hint != wantHint {
			t.Fatalf("%s rejection presentation = message %q hint %q", label, diagnostic.Message, diagnostic.Hint)
		}
	}

	assert(
		"application target locale",
		application,
		"fr",
		"code de quai invalide (application)",
		"choisissez un quai enregistré (application)",
		"ZZ",
	)
	assert("application C locale", application, "C", message, hint, "ZZ")
	assert("application POSIX locale", application, "POSIX", message, hint, "ZZ")

	providerDefinition := cli.Definition[config]{
		Name:         "alienctl",
		Localization: applicationLocalization,
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				localization := cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							message: "code de quai invalide (fournisseur)",
							hint:    "choisissez un quai enregistré (fournisseur)",
						},
					}},
				}
				batch.Localization(localization)
				command := cli.Command[config]{
					Name:    "remote",
					Declare: declareRejectedValue,
				}
				batch.Command(command)
				return nil
			})
		},
	}
	provider := clitest.Compile(t, providerDefinition)
	assert(
		"provider target locale",
		provider,
		"fr",
		"code de quai invalide (fournisseur)",
		"choisissez un quai enregistré (fournisseur)",
		"remote",
		"ZZ",
	)
}

func TestInputRejectionDiagnosticsHonorLimitAndPreserveSameSourceAcrossSpans(t *testing.T) {
	t.Parallel()
	type config struct{ Value string }
	joinedDefinition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				language := cli.TokenLanguage[string]{
					Name:  "joined",
					Count: cli.Count.Exactly(2),
					Parse: func(context.Context, []string) (string, error) {
						first := cli.Rejection.Message("first", "first", cli.Rejection.At(0, 0, 1))
						second := cli.Rejection.Message("second", "second", cli.Rejection.At(1, 0, 1))
						return "", errors.Join(first, second)
					},
					Format: func(value string) []string { return []string{value} },
				}
				form.Tokens(value, "value", language)
			})
		},
	}
	joined := clitest.Compile(t, joinedDefinition)
	inspectionRequest := cli.InspectionRequest{
		Args: []string{"x", "y"},
		Limits: cli.AnalysisLimits{
			Diagnostics: 1,
		},
	}
	inspection, err := joined.Inspect(t.Context(), inspectionRequest)
	if err == nil {
		t.Fatal("Inspect() error = nil")
	}
	report := cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 || !report.Truncated || !inspection.Truncated {
		t.Fatalf("limited rejection report = diagnostics %d report truncated %v inspection truncated %v", len(report.Diagnostics), report.Truncated, inspection.Truncated)
	}
	if report.Diagnostics[0].Rejection == nil || report.Diagnostics[0].Rejection.Code != "first" {
		t.Fatalf("limited rejection = %#v", report.Diagnostics[0])
	}

	runRequest := cli.RunRequest{
		Args: []string{"x", "y"},
		Limits: cli.AnalysisLimits{
			Diagnostics: 1,
		},
	}
	runErr := joined.Run(t.Context(), runRequest)
	if runErr == nil {
		t.Fatal("Run() error = nil")
	}
	runReport := cli.Diagnostics(runErr)
	if len(runReport.Diagnostics) != 1 || !runReport.Truncated || runReport.Diagnostics[0].Rejection == nil ||
		runReport.Diagnostics[0].Rejection.Code != "first" {
		t.Fatalf("limited Run rejection report = %#v", runReport)
	}

	acrossDefinition := cli.Definition[config]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				language := cli.TokenLanguage[string]{
					Name:  "across",
					Count: cli.Count.Exactly(2),
					Parse: func(context.Context, []string) (string, error) {
						return "", cli.Rejection.Message("across", "across", cli.Rejection.Across(0, 1, 1, 1))
					},
					Format: func(value string) []string { return []string{value} },
				}
				form.Tokens(value, "value", language)
			})
		},
	}
	across := clitest.Compile(t, acrossDefinition)
	request := cli.InspectionRequest{
		Args: []string{"ab", "cd"},
	}
	_, err = across.Inspect(t.Context(), request)
	if err == nil {
		t.Fatal("Across Inspect() error = nil")
	}
	report = cli.Diagnostics(err)
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Rejection == nil {
		t.Fatalf("Across Diagnostics = %#v", report.Diagnostics)
	}
	spans := report.Diagnostics[0].Rejection.Spans
	if len(spans) != 2 || spans[0].Source != spans[1].Source || spans[0].Entry != 0 || spans[0].Start.Offset != 1 || spans[0].End.Offset != 2 || spans[1].Entry != 1 || spans[1].Start.Offset != 0 || spans[1].End.Offset != 1 {
		t.Fatalf("Across spans = %#v", spans)
	}
}
