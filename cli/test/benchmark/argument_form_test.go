package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

type argumentFormFixture struct {
	app        cli.Application
	inspection cli.InspectionRequest
	completion cli.CompletionRequest
	forms      int
	terms      int
	items      int
	exactCount bool
}

// BenchmarkInspectArgumentForms separates form-count growth from term-count growth at the public Inspect boundary.
func BenchmarkInspectArgumentForms(b *testing.B) {
	for _, forms := range [...]int{1, 4, 16, 64} {
		fixture := literalArgumentFormFixture(b, forms, 4)
		preflightArgumentFormInspection(b, fixture)
		b.Run(fmt.Sprintf("Forms/N=%d/Terms=4", forms), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, fixture.app, fixture.inspection)
		})
	}
	for _, terms := range [...]int{1, 4, 16, 64} {
		fixture := literalArgumentFormFixture(b, 4, terms)
		preflightArgumentFormInspection(b, fixture)
		b.Run(fmt.Sprintf("Forms=4/Terms/N=%d", terms), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, fixture.app, fixture.inspection)
		})
	}
}

// BenchmarkCompleteArgumentForms measures the same declaration dimensions through completion, including exact-count token growth.
func BenchmarkCompleteArgumentForms(b *testing.B) {
	for _, forms := range [...]int{1, 4, 16, 64} {
		fixture := literalArgumentFormFixture(b, forms, 4)
		preflightArgumentFormCompletion(b, fixture)
		b.Run(fmt.Sprintf("Forms/N=%d/Terms=4", forms), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, fixture.app, fixture.completion)
		})
	}
	for _, terms := range [...]int{1, 4, 16, 64} {
		fixture := literalArgumentFormFixture(b, 4, terms)
		preflightArgumentFormCompletion(b, fixture)
		b.Run(fmt.Sprintf("Forms=4/Terms/N=%d", terms), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, fixture.app, fixture.completion)
		})
	}
	for _, tokens := range [...]int{1, 4, 16, 64} {
		fixture := exactCountArgumentFormFixture(b, tokens)
		preflightArgumentFormCompletion(b, fixture)
		b.Run(fmt.Sprintf("ExactCount/Tokens=%d", tokens), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, fixture.app, fixture.completion)
		})
	}
}

func isolatedArgumentFormDefinition(definition cli.Definition[inputConfig]) cli.Definition[inputConfig] {
	definition.Help = cli.HelpConfig{Disabled: true}
	definition.Version = cli.VersionConfig{}
	definition.ShellCompletion = cli.ShellCompletion{Disabled: true}
	definition.Run = func(context.Context, *inputConfig) error { return nil }
	return definition
}

func literalArgumentFormFixture(tb testing.TB, forms, terms int) argumentFormFixture {
	tb.Helper()
	declaration := cli.Definition[inputConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[inputConfig]) {
			for formIndex := range forms {
				index := formIndex
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[inputConfig]) {
					for termIndex := range terms {
						literal := fmt.Sprintf("term-%03d", termIndex)
						if termIndex == terms-1 {
							literal = fmt.Sprintf("final-%03d", index)
						}
						form.Literal(literal)
					}
				})
			}
		},
	}
	definition := isolatedArgumentFormDefinition(declaration)
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("argument-form literal compile: %v", err)
	}
	args := make([]string, terms)
	for termIndex := range terms {
		args[termIndex] = fmt.Sprintf("term-%03d", termIndex)
	}
	args[terms-1] = fmt.Sprintf("final-%03d", forms-1)
	completionArgs := append([]string(nil), args...)
	completionArgs[terms-1] = "final-"
	return argumentFormFixture{
		app:        app,
		inspection: cli.InspectionRequest{Args: args},
		completion: cli.CompletionRequest{
			Args: completionArgs,
			Cursor: cli.CompletionCursor{
				Argument: terms - 1,
				Byte:     len("final-"),
			},
		},
		forms: forms,
		terms: terms,
		items: forms,
	}
}

func exactCountArgumentFormFixture(tb testing.TB, tokens int) argumentFormFixture {
	tb.Helper()
	declaration := cli.Definition[inputConfig]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[inputConfig]) {
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[inputConfig]) {
				form.Strings(&form.Config.Strings, "items", cli.Count.Exactly(tokens))
			})
		},
	}
	definition := isolatedArgumentFormDefinition(declaration)
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("argument-form exact-count compile: %v", err)
	}
	args := make([]string, tokens)
	for index := range tokens {
		args[index] = fmt.Sprintf("value-%03d", index)
	}
	completionArgs := append([]string(nil), args...)
	completionArgs[tokens-1] = ""
	return argumentFormFixture{
		app:        app,
		inspection: cli.InspectionRequest{Args: args},
		completion: cli.CompletionRequest{
			Args: completionArgs,
			Cursor: cli.CompletionCursor{
				Argument: tokens - 1,
			},
		},
		forms:      1,
		terms:      1,
		exactCount: true,
	}
}

func preflightArgumentFormInspection(tb testing.TB, fixture argumentFormFixture) {
	tb.Helper()
	root := clitest.RootCommand()
	selected := root.ArgumentFormAt(fixture.forms - 1)
	clitest.Inspect(tb.Context(), fixture.app, fixture.inspection).Require(tb,
		clitest.InspectionReady(),
		selected.Selected(),
		root.MatchingArgumentFormsExactly(selected),
	)
}

func preflightArgumentFormCompletion(tb testing.TB, fixture argumentFormFixture) {
	tb.Helper()
	proof := clitest.Complete(tb.Context(), fixture.app, fixture.completion)
	target := clitest.CompletionTargetAt(0)
	if fixture.exactCount {
		proof.Require(tb,
			clitest.CompletionTargetsExactly(target),
			target.Kind(cli.CompletionTargetArgument),
			target.TermIndex(0),
			target.Occurrence(fixture.completion.Cursor.Argument),
			target.OnSelectedCommand(),
			clitest.CompletionItemsExactly(),
		)
		return
	}
	targets := make([]clitest.CompletionTargetLocator, fixture.forms)
	items := make([]clitest.CompletionItemLocator, fixture.items)
	for index := range targets {
		targets[index] = clitest.CompletionTargetAt(index)
		items[index] = clitest.CompletionItem(fmt.Sprintf("final-%03d", index))
	}
	proof.Require(tb,
		clitest.CompletionTargetsExactly(targets...),
		clitest.CompletionItemsExactly(items...),
	)
	for index, target := range targets {
		proof.Require(tb,
			target.Kind(cli.CompletionTargetLiteral),
			target.TermIndex(fixture.terms-1),
			target.OnSelectedCommand(),
			items[index].TargetsExactly(target),
		)
	}
}
