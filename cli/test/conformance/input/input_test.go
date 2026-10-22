package conformance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestNormalizeAnalysisLimitsPreservesAnalysisLimitsInvariant(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t)
	limits := cli.AnalysisLimits{
		Diagnostics: 7,
		ResponseFiles: cli.ResponseFileLimits{
			Depth:             2,
			SourceBytes:       3,
			TotalSourceBytes:  4,
			TotalTokens:       5,
			TokenBytes:        6,
			IncludeReferences: 7,
		},
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Limits: limits,
	}).Require(t)
	completionRequest := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
		Limits: limits,
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t)
}

func TestNormalizeAnalysisLimitsRejectsEveryNegativeField(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	cases := []cli.AnalysisLimits{
		{
			Diagnostics: -1,
		},
		{ResponseFiles: cli.ResponseFileLimits{
			Depth: -1,
		}},
		{ResponseFiles: cli.ResponseFileLimits{
			SourceBytes: -1,
		}},
		{ResponseFiles: cli.ResponseFileLimits{
			TotalSourceBytes: -1,
		}},
		{ResponseFiles: cli.ResponseFileLimits{
			TotalTokens: -1,
		}},
		{ResponseFiles: cli.ResponseFileLimits{
			TokenBytes: -1,
		}},
		{ResponseFiles: cli.ResponseFileLimits{
			IncludeReferences: -1,
		}},
	}
	for index, limits := range cases {
		if _, err := app.Inspect(t.Context(), cli.InspectionRequest{
			Limits: limits,
		}); err == nil {
			t.Fatalf("case %d accepted negative analysis limit: %#v", index, limits)
		}
		completionRequest := cli.CompletionRequest{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
			},
			Limits: limits,
		}
		if _, err := app.Complete(t.Context(), completionRequest); err == nil {
			t.Fatalf("completion case %d accepted negative analysis limit: %#v", index, limits)
		}
	}
}

var _ = errors.Is

type inputTestOrderedErrors []error

func (errs inputTestOrderedErrors) Error() string { return "ordered failures" }

func (errs inputTestOrderedErrors) Unwrap() []error { return append([]error(nil), errs...) }
func inputMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

type (
	inputRejectionValueConfig struct{ Value string }
	emptyMultiUnwrapperError  struct{}
)

func (emptyMultiUnwrapperError) Error() string { return "test error" }

func (emptyMultiUnwrapperError) Unwrap() []error { return nil }

type cyclicInputError struct{}

func (*cyclicInputError) Error() string { return "cyclic input failure" }

func (err *cyclicInputError) Unwrap() error { return err }

func TestRejectionOptionPreservesInputSemantics(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--unknown"},
	}).Require(
		t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnknownFlag).Present()),
	)
}

func TestMalformedRejectionMessageOptionsYieldParserFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		rejection cli.InputRejection
	}{
		{
			name: "expected value repeated across options",
			rejection: cli.Rejection.Message("invalid-value", "invalid value",
				cli.Rejection.Expected("one"),
				cli.Rejection.Expected("one"),
			),
		},
		{
			name:      "duplicate suggestions within option",
			rejection: cli.Rejection.Message("invalid-value", "invalid value", cli.Rejection.Suggest("one", "one")),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			type config = inputRejectionValueConfig
			app := clitest.Compile(t, cli.Definition[config]{
				Name: "tool",
				Declare: func(root *cli.CommandBuilder[config]) {
					value := root.Values.String(&root.Config.Value)
					root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						form.Tokens(value, "value", cli.TokenLanguage[string]{
							Name:   "value",
							Count:  cli.Count.Exactly(1),
							Parse:  func(context.Context, []string) (string, error) { return "", test.rejection },
							Format: func(value string) []string { return []string{value} },
						})
					})
				},
			})
			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: []string{"bad"},
			})
			proof.Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present()),
			)
		})
	}
}

func TestCollectInputRejections(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       error
		truncated bool
	}{
		{
			name: "zero children",
			err:  emptyMultiUnwrapperError{},
		},
		{
			name: "nil child",
			err:  inputTestOrderedErrors{nil},
		},
		{
			name:      "cyclic wrapper",
			err:       &cyclicInputError{},
			truncated: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			type config = inputRejectionValueConfig
			app := clitest.Compile(t, cli.Definition[config]{
				Name: "tool",
				Declare: func(root *cli.CommandBuilder[config]) {
					value := root.Values.String(&root.Config.Value)
					root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
						form.Tokens(value, "value", cli.TokenLanguage[string]{
							Name:   "value",
							Count:  cli.Count.Exactly(1),
							Parse:  func(context.Context, []string) (string, error) { return "", test.err },
							Format: func(value string) []string { return []string{value} },
						})
					})
				},
			})
			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
				Args: []string{"bad"},
			})
			failureExpectations := []clitest.DiagnosticReportExpectation{
				clitest.Diagnostic(cli.DiagnosticInputParserFailure).Present(),
			}
			if test.truncated {
				failureExpectations = append(failureExpectations, clitest.DiagnosticsTruncated())
			}
			proof.Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(failureExpectations...),
			)
		})
	}
}

type inputTagsLabelsConfig struct {
	Tags   []string
	Labels map[string]string
}

type inputScalarSeqMapConfig struct {
	Scalar string
	Seq    []string
	Map    map[string]string
}

func TestEnvironmentAuthoringLowersExactGranularitySensitivityAndPrefixSyntax(t *testing.T) {
	t.Parallel()
	type config = inputTagsLabelsConfig
	app := clitest.Compile(t, cli.Definition[config]{
		Name: "environment",
		Declare: func(root *cli.CommandBuilder[config]) {
			tags := root.Values.Sequence(&root.Config.Tags, cli.Value.String())
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			root.Values.Environment(tags, "TOOL_TAGS").Sensitive().Elements(",")
			root.Values.Environment(labels, "TOOL_LABELS").WholeKeyed(";")
			root.Values.EnvironmentPrefix(labels, "APP_LABEL_").Sensitive()
		},
	})
	root := clitest.RootCommand()
	tags := root.Environment("TOOL_TAGS")
	labels := root.Environment("TOOL_LABELS")
	prefix := root.Environment("APP_LABEL_")
	clitest.Schema(app).Require(t,
		root.EnvironmentInputsExactly(tags, labels, prefix),
		tags.Delimiter(","), tags.Sensitive(), tags.NotPrefix(),
		labels.Delimiter(";"), labels.NotPrefix(),
		prefix.Prefix(), prefix.Sensitive(),
	)
}

func TestEnvironmentAuthoringRejectsIncompatibleGranularityAndOverlappingSelectors(t *testing.T) {
	t.Parallel()
	type config = inputScalarSeqMapConfig
	tests := []struct {
		name    string
		declare func(*cli.CommandBuilder[config])
	}{
		{
			name: "scalar adapter",
			declare: func(root *cli.CommandBuilder[config]) {
				d := root.Values.String(&root.Config.Scalar)
				root.Values.Environment(d, "VALUE").Elements(",")
			},
		},
		{
			name: "sequence missing adapter",
			declare: func(root *cli.CommandBuilder[config]) {
				d := root.Values.Sequence(&root.Config.Seq, cli.Value.String())
				root.Values.Environment(d, "VALUES")
			},
		},
		{
			name: "overlapping prefix and exact",
			declare: func(root *cli.CommandBuilder[config]) {
				d := root.Values.Keyed(&root.Config.Map, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
				root.Values.EnvironmentPrefix(d, "APP_").Sensitive()
				root.Values.Environment(d, "APP_LABEL").KeyedEntries(",")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			clitest.CompileFails(t, cli.Definition[config]{
				Name:    "bad-env",
				Declare: test.declare,
			}).Require(t,
				clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
			)
		})
	}
}

func TestInputRejectionSnapshotsStructuredDetailAndRedactsSensitiveError(t *testing.T) {
	t.Parallel()
	expected := []string{"one", ""}
	suggestions := []string{"two"}
	replacement := []string{"fixed"}
	rejection := cli.Rejection.Message(
		"bad-value",
		"private message",
		cli.Rejection.Across(0, 1, 1, 2),
		cli.Rejection.Expected(expected...),
		cli.Rejection.Suggest(suggestions...),
		cli.Rejection.ReplaceWith(replacement...),
		cli.Rejection.Hint("private hint"),
		cli.Rejection.Sensitive(),
	)
	if got := rejection.Error(); got != "cli: input rejected" {
		t.Fatalf("Error() = %q, want redacted stable text", got)
	}
	expected[0], suggestions[0], replacement[0] = "mutated", "mutated", "mutated"
	inspection, err := inputTokenLanguageInspection(
		t.Context(),
		t,
		func(context.Context, []string) (string, error) { return "", rejection },
		"abc",
		"xy",
	)
	if err == nil || !inputInspectionHasDiagnosticCode(inspection, cli.DiagnosticInvalidValue) {
		t.Fatalf("structured rejection lost after caller mutation: err %v diagnostics %#v", err, inspection.Diagnostics)
	}
	diagnosticProof := clitest.Diagnostics(err)
	diagnosticProof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidValue).Present())
}

type inputDetachmentValueConfig struct{ Value string }

type inputDetachmentNameDefaultedRequiredInOutErrLocaleConfig struct {
	Name      string
	Defaulted string
	Required  string
	In        io.Reader
	Out       io.Writer
	Err       io.Writer
	Locale    cli.Locale
}

type inputDetachmentRequiredConfig struct{ Required string }

func TestCompletionInputDetachesOperationOwnedContainers(t *testing.T) {
	t.Parallel()
	type config = inputDetachmentValueConfig
	var calls int
	app := inputMustCompilePublic(t, cli.Definition[config]{
		Name: "completion-preparation",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Value).CompleteBy(func(_ context.Context, _ *config, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				calls++
				contribution := cli.CompletionContribution{
					Candidates: []cli.CompletionCandidate{{
						Value:       "alpha",
						Description: "candidate",
					}},
				}
				return contribution, nil
			}).
				Long("value")
		},
	})
	args := []string{"--value", "a"}
	env := cli.Env{{
		Name:  "A",
		Value: "1",
	}, {
		Name:  "A",
		Value: "2",
	}}
	fields := []cli.CompletionMatchField{cli.CompletionMatchFieldValue, cli.CompletionMatchFieldDescription}
	defaults := []cli.DefaultProvider{
		cli.DefaultsFor(func(context.Context, *cli.DefaultSupply[config]) error { return nil }),
	}
	facts := []cli.FactProvider{cli.FactsFor(func(context.Context, *cli.FactSupply[config]) error { return nil })}
	result, err := func() (cli.Completion, error) {
		completionRequest := cli.CompletionRequest{
			Args: args,
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     1,
			},
			Env:      env,
			Defaults: defaults,
			Facts:    facts,
			Client: cli.CompletionClient{
				Match: cli.CompletionMatchOptions{
					Fields: fields,
				},
			},
		}
		proof := clitest.Complete(t.Context(), app, completionRequest)
		proof.Require(t, clitest.CompletionItem("alpha").Present())
		return proof.Completion()
	}()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if calls != 1 || len(result.Items) != 1 || result.Items[0].Value != "alpha" {
		t.Fatalf("completion = %#v calls=%d", result, calls)
	}
	args[0] = "mutated"
	env[0].Name = "MUTATED"
	fields[0] = cli.CompletionMatchFieldDetail
	defaults[0] = nil
	facts[0] = nil
	if result.Items[0].Value != "alpha" {
		t.Fatalf("completion result changed after caller mutation: %#v", result)
	}
}

func TestCompletionInputRejectsInvalidOperationInputBeforeUse(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "completion-invalid",
	})
	cases := []struct {
		name string
		in   cli.CompletionRequest
	}{
		{
			name: "provider",
			in: cli.CompletionRequest{
				Defaults: []cli.DefaultProvider{cli.DefaultsFor[struct{}](nil)},
			},
		},
		{
			name: "environment",
			in: cli.CompletionRequest{
				Env: cli.Env{{
					Name:  "",
					Value: "x",
				}},
			},
		},
		{
			name: "limits",
			in: cli.CompletionRequest{
				Limits: cli.AnalysisLimits{
					Diagnostics: -1,
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			clitest.Complete(t.Context(), app, tc.in).Require(t,
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidCompletionRequest).Present()),
			)
		})
	}
}

func TestRunInputDetachesContainersAndBorrowsHostIntegrations(t *testing.T) {
	t.Parallel()
	type config = inputDetachmentNameDefaultedRequiredInOutErrLocaleConfig
	args := []string{"--name", "cli"}
	env := cli.Env{{
		Name:  "MODE",
		Value: "dev",
	}, {
		Name:  "MODE",
		Value: "",
	}}
	stdin := bytes.NewBufferString("input")
	stdout := io.Discard
	stderr := io.Discard
	defaultCalls, factCalls := 0, 0
	var defaulted cli.Destination[config, string]
	var required cli.Destination[config, string]
	var defaults []cli.DefaultProvider
	var facts []cli.FactProvider
	defaults = []cli.DefaultProvider{
		cli.DefaultsFor[config](func(_ context.Context, supply *cli.DefaultSupply[config]) error {
			defaultCalls++
			supply.Set(defaulted, "from-default-provider")
			args[0] = "--mutated"
			env[0].Value = "mutated"
			defaults[0] = nil
			facts[0] = nil
			return nil
		}),
	}
	facts = []cli.FactProvider{cli.FactsFor[config](func(_ context.Context, supply *cli.FactSupply[config]) error {
		factCalls++
		supply.Set(required, "fact")
		return nil
	})}
	var got config
	definition := cli.Definition[config]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Flags.String(&root.Config.Name).Long("name")
			defaulted = root.Values.String(&root.Config.Defaulted)
			root.Rule.Require(defaulted)
			required = root.Facts.Require(&root.Config.Required)
			root.Access.Stdin(&root.Config.In)
			root.Access.Stdout(&root.Config.Out)
			root.Access.Stderr(&root.Config.Err)
			root.Access.Locale(&root.Config.Locale)
		},
		Run: func(_ context.Context, cfg *config) error { got = *cfg; return nil },
	}
	app := inputMustCompilePublic(t, definition)
	runRequest := cli.RunRequest{
		Args:     args,
		Env:      env,
		Defaults: defaults,
		Facts:    facts,
		Stdin:    stdin,
		Stdout:   stdout,
		Stderr:   stderr,
		Locale:   "en-US",
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Name != "cli" || got.Defaulted != "from-default-provider" || got.Required != "fact" {
		t.Fatalf("snapshot values=%#v", got)
	}
	if got.In != stdin || got.Out != stdout || got.Err != stderr || got.Locale != "en-US" {
		t.Fatalf("borrowed integrations=%#v", got)
	}
	if defaultCalls != 1 || factCalls != 1 {
		t.Fatalf("provider calls defaults=%d facts=%d", defaultCalls, factCalls)
	}
}

func TestInspectionInputRejectsInvalidProviderPreflightBeforeSemanticUse(t *testing.T) {
	t.Parallel()
	type config = inputDetachmentRequiredConfig
	var required cli.Destination[config, string]
	app := clitest.Compile(t, cli.Definition[config]{
		Name:    "tool",
		Declare: func(root *cli.CommandBuilder[config]) { required = root.Facts.Require(&root.Config.Required) },
	})
	validDefault := cli.DefaultsFor[config](func(context.Context, *cli.DefaultSupply[config]) error { return nil })
	validFact := cli.FactsFor[config](
		func(_ context.Context, s *cli.FactSupply[config]) error { s.Set(required, "ok"); return nil },
	)
	cases := []cli.InspectionRequest{
		{
			Defaults: []cli.DefaultProvider{validDefault, nil},
			Facts:    []cli.FactProvider{validFact},
		},
		{
			Defaults: []cli.DefaultProvider{cli.DefaultsFor[config](nil)},
			Facts:    []cli.FactProvider{validFact},
		},
		{
			Facts: []cli.FactProvider{validFact, nil},
		},
		{
			Facts: []cli.FactProvider{cli.FactsFor[config](nil)},
		},
	}
	for i, request := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			t.Parallel()
			clitest.Inspect(t.Context(), app, request).Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
			)
		})
	}
}

func TestInspectionInputRejectsMalformedEnvironment(t *testing.T) {
	t.Parallel()
	app := clitest.Compile(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "BAD=NAME",
			Value: "x",
		}},
	}).Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
	)
}

func TestEnvFromEnvironPreservesInvocationSemantics(t *testing.T) {
	t.Parallel()
	env, err := cli.EnvFromEnviron([]string{"B=2", "A=1=tail"})
	if err != nil || len(env) != 2 || env[1].Name != "A" || env[1].Value != "1=tail" {
		t.Fatalf("EnvFromEnviron = %#v, %v", env, err)
	}
	if _, err := cli.EnvFromEnviron([]string{"broken"}); err == nil {
		t.Fatal("malformed entry accepted")
	}
}

func TestEnvFromMapPreservesInvocationSemantics(t *testing.T) {
	t.Parallel()
	env := cli.EnvFromMap(map[string]string{"B": "2", "A": "1"})
	if len(env) != 2 || env[0].Name != "A" || env[1].Name != "B" {
		t.Fatalf("EnvFromMap = %#v", env)
	}
}

func TestInvalidEnvironmentEntryErrorErrorPreservesInvocationSemantics(t *testing.T) {
	t.Parallel()
	_, err := cli.EnvFromEnviron([]string{"broken"})
	if err == nil || err.Error() != "cli: malformed environment entry" {
		t.Fatalf("error = %v", err)
	}
}

type inputTokenLanguageRejectionValueConfig struct{ Value string }

func inputTokenLanguageInspection(ctx context.Context, t *testing.T, parse func(context.Context, []string) (string, error), args ...string) (cli.Inspection, error) {
	t.Helper()
	type config = inputTokenLanguageRejectionValueConfig
	definition := cli.Definition[config]{
		Name: "rejection",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			value := root.Values.String(&root.Config.Value)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.Tokens(value, "value", cli.TokenLanguage[string]{
					Name:   "external-rejection",
					Count:  cli.Count.Exactly(len(args)),
					Parse:  parse,
					Format: func(value string) []string { return []string{value} },
				})
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
	return app.Inspect(ctx, request)
}

func inputInspectionHasDiagnosticCode(inspection cli.Inspection, code cli.DiagnosticCode) bool {
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestMatchEnvironmentProducersPreservesProducerThenDuplicateEntryOrder(t *testing.T) {
	t.Parallel()
	type config struct{ Modes []string }
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			modes := root.Values.Sequence(&root.Config.Modes, cli.Value.String())
			root.Values.Environment(modes, "MODE").Elements(",")
			root.Values.Environment(modes, "OLD_MODE").Sensitive().Elements(",")
		},
	}
	app := clitest.Compile(t, definition)
	mode := clitest.RootCommand().Environment("MODE")
	old := clitest.RootCommand().Environment("OLD_MODE")
	value := mode.Value()
	c0 := value.CandidateAt(0)
	c1 := value.CandidateAt(1)
	c2 := value.CandidateAt(2)
	c3 := value.CandidateAt(3)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Env: cli.Env{
			{
				Name:  "OLD_MODE",
				Value: "reference-first",
			},
			{
				Name:  "MODE",
				Value: "one",
			},
			{
				Name:  "MODE",
				Value: "",
			},
			{
				Name:  "OLD_MODE",
				Value: "reference-last",
			},
		},
	}).Require(t,
		clitest.InspectionReady(),
		value.CandidatesExactly(c0, c1, c2, c3),
		c0.Declaration(mode),
		c0.ElementCount(1),
		c0.ElementAt(0).InputText("one"),
		c1.Declaration(mode),
		c1.ElementCount(0),
		c2.Declaration(old),
		c2.ElementCount(1),
		c2.ElementAt(0).Sensitive(),
		c2.Sensitive(),
		c3.Declaration(old),
		c3.ElementCount(1),
		c3.ElementAt(0).Sensitive(),
		c3.Sensitive(),
	)
}
