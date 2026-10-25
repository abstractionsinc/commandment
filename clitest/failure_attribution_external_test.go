package clitest_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

const (
	failureAttributionCaseEnv  = "COMMANDMENT_CLITEST_FAILURE_ATTRIBUTION_CASE"
	failureAttributionChildArg = "commandment-clitest-failure-attribution-child"
)

func failureAttributionHelperCase() (string, bool) {
	evaluationCase := os.Getenv(failureAttributionCaseEnv)
	if evaluationCase == "" {
		return "", false
	}
	if slices.Contains(os.Args[1:], failureAttributionChildArg) {
		return evaluationCase, true
	}
	return "", false
}

func failureAttributionChildEnv(evaluationCase string) []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(name, failureAttributionCaseEnv) {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, failureAttributionCaseEnv+"="+evaluationCase)
}

func failureUXExternalCases() []string {
	return []string{
		"inspection", "completion", "schema", "reference", "diagnostics", "localization", "conformance",
		"compile", "compile-fails", "help-presentation", "resolve-declaration",
		"scalar", "presence", "aggregate", "failed-prerequisite", "many-roots",
		"ordered-collection", "ambiguous-diagnostic", "truncated-diagnostic", "nested-diagnostic", "diagnostic-producer-kind",
		"sensitive-control", "sensitive-rejection-expected-empty", "sensitive-rejection-expected-secret", "rejection-spans", "huge-control", "deep-nesting", "repeated-subjects",
		"direct-id", "nested-subtest",
	}
}

func TestPublicExpectationEntrypointsPreserveConsumerAttribution(t *testing.T) {
	t.Parallel()
	if evaluationCase, child := failureAttributionHelperCase(); child {
		runPublicExpectationAttributionFailure(t, evaluationCase)
		return
	}

	callLines := failureAttributionCallLines(t)
	productionFrame := regexp.MustCompile(`(?:^|[/\\])clitest[/\\]([^/\\:]+\.go):(\d+):`)
	for _, evaluationCase := range failureUXExternalCases() {
		t.Run(evaluationCase, func(t *testing.T) {
			t.Parallel()
			text := runFailureAttributionCase(t, evaluationCase)
			if !strings.Contains(text, "clitest: expectation failed") || !strings.Contains(text, "expected:") {
				t.Fatalf("%s failure did not use the clitest failure-report substrate:\n%s", evaluationCase, text)
			}
			if strings.Contains(text, "actual:") {
				t.Fatalf("%s failure used deprecated actual label:\n%s", evaluationCase, text)
			}
			if !strings.Contains(text, "observed:") && !strings.Contains(text, "not provable:") {
				t.Fatalf("%s failure exposed neither observed nor not-provable truth:\n%s", evaluationCase, text)
			}
			want := "failure_attribution_external_test.go:" + strconv.Itoa(callLines[evaluationCase]) + ":"
			if !strings.Contains(text, want) {
				t.Fatalf("%s failure was not attributed to exact consumer call %s\n%s", evaluationCase, want, text)
			}
			if evaluationCase == "compile" && !strings.Contains(text, "invalid-declaration") {
				t.Fatalf("compile failure omitted structured diagnostic code:\n%s", text)
			}
			if evaluationCase == "compile-fails" && !strings.Contains(text, "structured diagnostic rejection") {
				t.Fatalf("CompileFails success failure omitted expected boundary:\n%s", text)
			}
			switch evaluationCase {
			case "schema":
				if !strings.Contains(text, "item 0: command is absent") {
					t.Fatalf("schema collection failure omitted separated item coordinate:\n%s", text)
				}
			case "help-presentation":
				if !strings.Contains(text, "help usage") || !strings.Contains(text, "index 0=") {
					t.Fatalf("help-presentation failure omitted structured collection delta:\n%s", text)
				}
			case "resolve-declaration":
				if !strings.Contains(text, "declaration locator") || !strings.Contains(text, "resolve exactly") {
					t.Fatalf("declaration acquisition failure omitted semantic boundary:\n%s", text)
				}
			case "scalar":
				if !strings.Contains(text, "locale") || !strings.Contains(text, `"fr"`) {
					t.Fatalf("scalar failure omitted expected locale fact:\n%s", text)
				}
			case "presence":
				if !strings.Contains(text, `command path "missing"`) || !strings.Contains(text, "exists=true") {
					t.Fatalf("presence failure omitted semantic subject or predicate:\n%s", text)
				}
			case "aggregate":
				if strings.Count(text, "clitest: expectation failed") != 1 ||
					!strings.Contains(text, `1. command path "missing-a"`) ||
					!strings.Contains(text, `2. command path "missing-b"`) {
					t.Fatalf("aggregate failure did not preserve one report and root order:\n%s", text)
				}
				repeated := runFailureAttributionCase(t, evaluationCase)
				if normalizeGoTestOutput(text) != normalizeGoTestOutput(repeated) {
					t.Fatalf(
						"representative public failure output was nondeterministic:\nfirst:\n%s\nsecond:\n%s",
						text,
						repeated,
					)
				}
			case "failed-prerequisite":
				if strings.Count(text, "clitest: expectation failed") != 1 ||
					!strings.Contains(text, "inspection operation") ||
					strings.Contains(text, "derivative") {
					t.Fatalf("failed prerequisite did not suppress derivative expectation noise:\n%s", text)
				}
			case "many-roots":
				if !strings.Contains(text, "additional failure(s) omitted") ||
					!strings.Contains(text, `1. command path "missing-00"`) ||
					strings.Contains(text, `command path "missing-39"`) {
					t.Fatalf("many-root failure was not deterministically bounded:\n%s", text)
				}
			case "ordered-collection":
				if !strings.Contains(text, "length=256") || !strings.Contains(text, "index 127=") {
					t.Fatalf("ordered collection failure omitted focused delta:\n%s", text)
				}
			case "ambiguous-diagnostic":
				if !strings.Contains(text, "matched 2 diagnostics") {
					t.Fatalf("ambiguous diagnostic failure omitted ambiguity:\n%s", text)
				}
			case "truncated-diagnostic":
				if !strings.Contains(text, "not provable:") || !strings.Contains(text, "truncated") {
					t.Fatalf("truncated diagnostic failure asserted false absence:\n%s", text)
				}
			case "diagnostic-producer-kind":
				if !strings.Contains(text, "producer kind") || !strings.Contains(text, "environment") {
					t.Fatalf("diagnostic producer-kind failure omitted structured producer-kind fact:\n%s", text)
				}
			case "sensitive-control":
				for _, secret := range []string{"expected-secret", "observed-secret"} {
					if strings.Contains(text, secret) {
						t.Fatalf("sensitive failure leaked %q:\n%s", secret, text)
					}
				}
				if !strings.Contains(text, "<redacted>") {
					t.Fatalf("sensitive failure omitted redaction marker:\n%s", text)
				}
			case "sensitive-rejection-expected-empty":
				assertSensitiveRejectionExpectedFailure(t, text)
			case "sensitive-rejection-expected-secret":
				assertSensitiveRejectionExpectedFailure(t, text, "caller-secret")
			case "rejection-spans":
				for _, want := range []string{"rejection span", "source=present", "start={offset="} {
					if !strings.Contains(text, want) {
						t.Fatalf("rejection-span failure omitted semantic detail %q:\n%s", want, text)
					}
				}
				if strings.Contains(text, "InputSpan{") || strings.Contains(text, "value:") {
					t.Fatalf("rejection-span failure exposed Go representation:\n%s", text)
				}
			case "huge-control":
				if strings.ContainsRune(text, '\x1b') || !strings.Contains(text, `\n`) ||
					!strings.Contains(text, "rendered bytes omitted") ||
					len(text) > 40*1024 {
					t.Fatalf("huge/control failure was unsafe or unbounded (len=%d):\n%s", len(text), text)
				}
			case "deep-nesting":
				if !strings.Contains(text, "nested locator") || len(text) > 40*1024 {
					t.Fatalf("deeply nested subject was not semantically bounded (len=%d):\n%s", len(text), text)
				}
			case "repeated-subjects":
				if !strings.Contains(text, `command path "left" child command "same"`) ||
					!strings.Contains(text, `command path "right" child command "same"`) {
					t.Fatalf("repeated subjects lost parent coordinates:\n%s", text)
				}
			case "direct-id":
				if !strings.Contains(text, "command id 999") {
					t.Fatalf("direct-ID failure lost caller-selected disambiguator:\n%s", text)
				}
			}
			for _, match := range productionFrame.FindAllStringSubmatch(text, -1) {
				if !strings.HasSuffix(match[1], "_test.go") {
					t.Fatalf(
						"%s failure exposed production clitest frame %q as a location:\n%s",
						evaluationCase,
						match[0],
						text,
					)
				}
			}
		})
	}
}

func assertSensitiveRejectionExpectedFailure(t *testing.T, text string, callerSecrets ...string) {
	t.Helper()

	for _, want := range []string{
		"expected: expected fragments = <redacted>",
		"not provable: observed expected fragments = <redacted>",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("sensitive rejection failure omitted %q:\n%s", want, text)
		}
	}
	for _, secret := range []string{"alpha", "beta"} {
		if strings.Contains(text, secret) {
			t.Fatalf("sensitive rejection failure leaked observed fragment %q:\n%s", secret, text)
		}
	}
	for _, secret := range callerSecrets {
		if strings.Contains(text, secret) {
			t.Fatalf("sensitive rejection failure leaked caller material %q:\n%s", secret, text)
		}
	}
}

func runFailureAttributionCase(t *testing.T, evaluationCase string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	args := []string{"-test.run=^TestPublicExpectationEntrypointsPreserveConsumerAttribution$", "-test.v"}
	if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
		args = append(args, "-test.gocoverdir="+coverDir)
	}
	args = append(args, failureAttributionChildArg)
	//nolint:gosec // The test deliberately launches the current test binary with test-owned arguments.
	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	cmd.Env = failureAttributionChildEnv(evaluationCase)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s expectation helper exceeded test-owned watchdog: %v\n%s", evaluationCase, ctx.Err(), output)
	}
	if err == nil {
		t.Fatalf("%s expectation helper unexpectedly passed:\n%s", evaluationCase, output)
	}
	return string(output)
}

var (
	goTestTiming   = regexp.MustCompile(`\([0-9]+(?:\.[0-9]+)?s\)`)
	goTestCoverage = regexp.MustCompile(`(?m)^coverage: [0-9]+(?:\.[0-9]+)?% of statements\n?`)
)

func normalizeGoTestOutput(text string) string {
	text = goTestTiming.ReplaceAllString(text, "(duration)")
	return goTestCoverage.ReplaceAllString(text, "")
}

func failureAttributionCallLines(t *testing.T) map[string]int {
	t.Helper()
	path := "failure_attribution_external_test.go"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	result := make(map[string]int)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "runPublicExpectationAttributionFailure" {
			continue
		}
		inspectAttributionSwitch := func(node ast.Node) bool {
			switchStatement, ok := node.(*ast.SwitchStmt)
			if !ok {
				return true
			}
			tag, ok := switchStatement.Tag.(*ast.Ident)
			if !ok || tag.Name != "evaluationCase" {
				return true
			}
			for _, statement := range switchStatement.Body.List {
				clause, ok := statement.(*ast.CaseClause)
				if !ok || len(clause.List) != 1 {
					continue
				}
				literal, ok := clause.List[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				name, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("decode attribution expectation case %s: %v", literal.Value, err)
				}
				line := 0
				inspectConsumerCall := func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok || !failureAttributionConsumerCall(call) {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						t.Fatalf("attribution consumer function = %T, want *ast.SelectorExpr", call.Fun)
					}
					line = fset.Position(selector.Sel.Pos()).Line
					return true
				}
				for _, bodyStatement := range clause.Body {
					ast.Inspect(bodyStatement, inspectConsumerCall)
				}
				if line != 0 {
					result[name] = line
				}
			}
			return false
		}
		ast.Inspect(function.Body, inspectAttributionSwitch)
	}

	for _, name := range failureUXExternalCases() {
		if result[name] == 0 {
			t.Fatalf("missing structural attribution consumer for %q", name)
		}
	}
	return result
}

func failureAttributionConsumerCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	consumesT := false
	for _, argument := range call.Args {
		identifier, ok := argument.(*ast.Ident)
		if ok && identifier.Name == "t" {
			consumesT = true
			break
		}
	}
	if !consumesT {
		return false
	}
	if selector.Sel.Name == "Require" {
		return true
	}
	packageName, ok := selector.X.(*ast.Ident)
	return ok && packageName.Name == "clitest"
}

func runPublicExpectationAttributionFailure(t *testing.T, evaluationCase string) { //nolint:thelper // Intentionally remains a consumer frame so attribution is tested at each public call.
	definition := cli.Definition[struct{}]{
		Name: "tool",
		Description: cli.Description{
			Short: "known source",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	switch evaluationCase {
	case "inspection":
		clitest.Inspect(ctx, app, cli.InspectionRequest{}).
			Require(t, clitest.InspectionRejected())
	case "completion":
		clitest.Complete(ctx, app, cli.CompletionRequest{}).
			Require(t, clitest.CompletionPartial())
	case "schema":
		clitest.Schema(app).Require(t, clitest.CommandsExactly(clitest.Command("missing")))
	case "reference":
		clitest.Reference(app, cli.ReferenceOptions{}).
			Require(t, clitest.CommandsExactly(clitest.Command("missing")))
	case "diagnostics":
		clitest.Diagnostics(errors.New("ordinary application error")).
			Require(t, clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticUnknownFlag)))
	case "localization":
		policy := clitest.LocalizationPolicy{
			RequireCompleteLocales: []cli.Locale{"fr"},
		}
		clitest.RequireLocalization(t, app, policy)
	case "conformance":
		expectations := []clitest.SchemaExpectation{
			clitest.CommandsExactly(clitest.Command("missing")),
		}
		schemaCase := clitest.SchemaCase{
			Name:         "schema",
			Expectations: expectations,
		}
		clitest.Conformance{schemaCase}.Require(t, ctx, app)
	case "compile":
		declare := func(root *cli.CommandBuilder[struct{}]) { root.Arguments.Form(nil) }
		definition := cli.Definition[struct{}]{
			Name:    "invalid-compile",
			Declare: declare,
		}
		clitest.Compile(t, definition)
	case "compile-fails":
		definition := cli.Definition[struct{}]{
			Name: "unexpected-success",
		}
		clitest.CompileFails(t, definition)
	case "help-presentation":
		help := cli.HelpPage{
			Usage: []string{"observed usage"},
		}
		clitest.RequireHelpPresentation(t, help, clitest.HelpUsageExactly("expected usage"))
	case "resolve-declaration":
		clitest.ResolveDeclaration(
			t,
			clitest.Schema(app),
			clitest.Command("missing").CanonicalEntry(),
		)
	case "scalar":
		clitest.Reference(app, cli.ReferenceOptions{}).Require(t, clitest.EffectiveLocale("fr"))
	case "presence":
		clitest.Schema(app).Require(t, clitest.Command("missing").Exists())
	case "aggregate":
		clitest.Schema(app).
			Require(t, clitest.Command("missing-a").Exists(), clitest.Command("missing-b").Exists())
	case "failed-prerequisite":
		request := cli.InspectionRequest{
			Args: []string{"--unknown"},
		}
		clitest.Inspect(ctx, app, request).Require(t, clitest.Command("derivative").Exists())
	case "many-roots":
		expectations := make([]clitest.SchemaExpectation, 40)
		for i := range expectations {
			expectations[i] = clitest.Command(fmt.Sprintf("missing-%02d", i)).Exists()
		}
		clitest.Schema(app).Require(t, expectations...)
	case "ordered-collection":
		declare := func(root *cli.CommandBuilder[struct{}]) {
			for i := range 256 {
				child := cli.Command[struct{}]{
					Name: fmt.Sprintf("child-%02d", i),
				}
				root.Subcommands.Command(child)
			}
		}
		definition := cli.Definition[struct{}]{
			Name:    "collection",
			Declare: declare,
		}
		collectionApp := clitest.Compile(t, definition)
		want := make([]clitest.CommandLocator, 256)
		for i := range want {
			want[i] = clitest.Command(fmt.Sprintf("child-%02d", i))
		}
		want[127], want[128] = want[128], want[127]
		clitest.Schema(collectionApp).
			Require(t, clitest.RootCommand().ChildrenExactly(want...))
	case "ambiguous-diagnostic":
		firstRequest := cli.InspectionRequest{
			Args: []string{"--unknown-a"},
		}
		_, first := app.Inspect(ctx, firstRequest)
		secondRequest := cli.InspectionRequest{
			Args: []string{"--unknown-b"},
		}
		_, second := app.Inspect(ctx, secondRequest)
		clitest.Diagnostics(errors.Join(first, second)).
			Require(t, clitest.Diagnostic(cli.DiagnosticUnknownFlag).Present())
	case "truncated-diagnostic":
		validate := func(context.Context, *struct{}, cli.Dependencies) ([]cli.Violation, error) {
			violations := []cli.Violation{{
				Kind: cli.ViolationInvalid,
			}, {
				Kind: cli.ViolationInvalid,
			}}
			return violations, nil
		}
		declare := func(root *cli.CommandBuilder[struct{}]) { root.Rule.Validate("many", validate) }
		run := func(context.Context, *struct{}) error { return nil }
		truncatedDefinition := cli.Definition[struct{}]{
			Name:    "truncated",
			Declare: declare,
			Run:     run,
		}
		truncatedApp := clitest.Compile(t, truncatedDefinition)
		runRequest := cli.RunRequest{
			Limits: cli.AnalysisLimits{
				Diagnostics: 1,
			},
		}
		failure := truncatedApp.Run(ctx, runRequest)
		clitest.Diagnostics(failure).
			Require(t, clitest.Diagnostic(cli.DiagnosticInvalidValue).Absent())
	case "nested-diagnostic":
		request := cli.InspectionRequest{
			Args: []string{"--unknown"},
		}
		_, failure := app.Inspect(ctx, request)
		expectedSpan := cli.InputSpan{
			Entry: 99,
		}
		clitest.Diagnostics(failure).Require(t, clitest.DiagnosticAt(0).Span(expectedSpan))
	case "diagnostic-producer-kind":
		request := cli.InspectionRequest{
			Args: []string{"unexpected"},
		}
		_, failure := app.Inspect(ctx, request)
		clitest.Diagnostics(failure).
			Require(t, clitest.DiagnosticAt(0).ProducerKind(cli.ValueProducerKindEnvironment))
	case "sensitive-control":
		type secretConfig struct{ Secret string }
		completeSecret := func(context.Context, *secretConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
			message := cli.CompletionMessage{
				Kind:      cli.CompletionMessageInformation,
				Text:      "observed-secret\n\t\x1b",
				Sensitive: true,
			}
			contribution := cli.CompletionContribution{
				Messages: []cli.CompletionMessage{message},
			}
			return contribution, nil
		}
		declare := func(root *cli.CommandBuilder[secretConfig]) {
			root.Flags.String(&root.Config.Secret).CompleteBy(completeSecret).Long("secret")
		}
		secretDefinition := cli.Definition[secretConfig]{
			Name:    "secret",
			Declare: declare,
		}
		secretApp := clitest.Compile(t, secretDefinition)
		request := cli.CompletionRequest{
			Args: []string{"--secret", ""},
			Cursor: cli.CompletionCursor{
				Argument: 1,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					Messages: true,
				},
			},
		}
		expectedMessage := cli.CompletionMessage{
			Kind:      cli.CompletionMessageWarning,
			Text:      "expected-secret\n\t\x1b",
			Sensitive: true,
		}
		clitest.Complete(ctx, secretApp, request).Require(t, clitest.CompletionMessagesExactly(expectedMessage))
	case "sensitive-rejection-expected-empty":
		diagnostics, _ := inputRejectionObservation(t, true)
		diagnostics.Require(
			t,
			clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection().ExpectedExactly(),
		)
	case "sensitive-rejection-expected-secret":
		diagnostics, _ := inputRejectionObservation(t, true)
		diagnostics.Require(
			t,
			clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection().ExpectedExactly("caller-secret"),
		)
	case "rejection-spans":
		diagnostics, span := inputRejectionObservation(t, false)
		span.Start.Offset++
		diagnostics.Require(
			t,
			clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection().SpansExactly(span),
		)
	case "huge-control":
		name := "\n\t\x1b[31m" + strings.Repeat("x", 64*1024)
		clitest.Schema(app).Require(t, clitest.Command(name).Exists())
	case "deep-nesting":
		locator := clitest.RootCommand()
		for i := range 128 {
			locator = locator.Command(fmt.Sprintf("level-%03d", i))
		}
		clitest.Schema(app).Require(t, locator.Exists())
	case "repeated-subjects":
		left := cli.Command[struct{}]{
			Name: "left",
		}
		right := cli.Command[struct{}]{
			Name: "right",
		}
		declare := func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(left)
			root.Subcommands.Command(right)
		}
		repeatedDefinition := cli.Definition[struct{}]{
			Name:    "repeated",
			Declare: declare,
		}
		repeatedApp := clitest.Compile(t, repeatedDefinition)
		clitest.Schema(repeatedApp).Require(t,
			clitest.Command("left").Command("same").Exists(),
			clitest.Command("right").Command("same").Exists(),
		)
	case "direct-id":
		clitest.Schema(app).Require(t, clitest.CommandByID(cli.CommandID(999)).Exists())
	case "nested-subtest":
		t.Run("consumer-subtest", func(t *testing.T) {
			t.Parallel()
			clitest.Schema(app).
				Require(t, clitest.Command("subtest-missing").Exists())
		})
	default:
		t.Fatalf("unknown evaluation case %q", evaluationCase)
	}
}
