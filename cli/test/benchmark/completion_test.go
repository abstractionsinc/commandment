package benchmark

import (
	"fmt"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

func BenchmarkComplete(b *testing.B) {
	definition := cli.Definition[inputConfig]{
		Name:            "tool",
		Help:            cli.HelpConfig{Disabled: true},
		ShellCompletion: cli.ShellCompletion{Disabled: true},
		Declare: func(root *cli.CommandBuilder[inputConfig]) {
			root.Subcommands.Command(cli.Command[inputConfig]{
				Name: "deploy",
			})
			root.Subcommands.Command(cli.Command[inputConfig]{
				Name: "status",
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		b.Fatalf("Complete compile preflight: %v", err)
	}
	request := cli.CompletionRequest{
		Args: []string{""},
		Cursor: cli.CompletionCursor{
			Argument: 0,
		},
	}
	clitest.Complete(b.Context(), app, request).Require(b,
		clitest.CompletionItem("deploy").Present(),
		clitest.CompletionItem("status").Present(),
	)
	if b.Failed() {
		return
	}
	b.ReportAllocs()
	clibench.Complete(b, app, request)
}

func BenchmarkCompleteSelectedPath(b *testing.B) {
	for _, depth := range [...]int{1, 4, 16, 64} {
		definition := cli.Definition[inputConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[inputConfig]) {
				declareCompletionDepth(root, depth, 0)
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Complete selected-path compile: %v", err)
		}
		args := make([]string, 0, depth+1)
		for level := range depth {
			args = append(args, fmt.Sprintf("selected-%03d", level))
		}
		args = append(args, "")
		request := cli.CompletionRequest{
			Args: args,
			Cursor: cli.CompletionCursor{
				Argument: depth,
			},
		}
		proof := clitest.Complete(b.Context(), app, request)
		proof.Require(b, clitest.CompletionItemsExactly(clitest.CompletionItem("--target")))
		if b.Failed() {
			return
		}
		result, err := proof.Completion()
		if err != nil {
			b.Fatalf("Complete selected-path preflight: %v", err)
		}
		if len(result.Selection.Path) != depth+1 {
			b.Fatalf("Complete selected-path preflight: path=%d want=%d", len(result.Selection.Path), depth+1)
		}
		b.Run(fmt.Sprintf("Depth/N=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, app, request)
		})
	}
}

// BenchmarkCompleteUnrelatedSubtree holds the selected completion path fixed while unrelated command structure grows.
func BenchmarkCompleteUnrelatedSubtree(b *testing.B) {
	for _, unrelated := range [...]int{16, 64, 256, 1024} {
		definition := cli.Definition[inputConfig]{
			Name:            "tool",
			Help:            cli.HelpConfig{Disabled: true},
			ShellCompletion: cli.ShellCompletion{Disabled: true},
			Declare: func(root *cli.CommandBuilder[inputConfig]) {
				root.Subcommands.Command(cli.Command[inputConfig]{
					Name: "target",
				})
				root.Subcommands.Command(cli.Command[inputConfig]{
					Name: "bulk",
					Declare: func(bulk *cli.CommandBuilder[inputConfig]) {
						for index := range unrelated {
							bulk.Subcommands.Command(cli.Command[inputConfig]{
								Name: fmt.Sprintf("n%04d", index),
							})
						}
					},
				})
			},
		}
		app, err := cli.Compile(definition)
		if err != nil {
			b.Fatalf("Complete projection compile: %v", err)
		}
		request := cli.CompletionRequest{
			Args: []string{""},
			Cursor: cli.CompletionCursor{
				Argument: 0,
			},
			Client: cli.CompletionClient{
				Capabilities: cli.CompletionClientCapabilities{
					CandidateKinds: true,
				},
			},
		}
		clitest.Complete(b.Context(), app, request).Require(b,
			clitest.CompletionItemsExactly(
				clitest.CompletionItem("bulk"),
				clitest.CompletionItem("target"),
			),
		)
		if b.Failed() {
			return
		}
		b.Run(fmt.Sprintf("UnrelatedSubtree/N=%d", unrelated), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, app, request)
		})
	}
}

func declareCompletionDepth(parent *cli.CommandBuilder[inputConfig], depth, level int) {
	if level == depth {
		parent.Flags.Bool(&parent.Config.Target).Long("target")
		return
	}
	parent.Subcommands.Command(cli.Command[inputConfig]{
		Name: fmt.Sprintf("selected-%03d", level),
		Declare: func(child *cli.CommandBuilder[inputConfig]) {
			declareCompletionDepth(child, depth, level+1)
		},
	})
	parent.Subcommands.Command(cli.Command[inputConfig]{
		Name: fmt.Sprintf("sibling-%03d", level),
	})
}
