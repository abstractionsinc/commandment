package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCompletionDelegatesFilesToTheClient(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	client := cli.CompletionClient{
		Capabilities: cli.CompletionClientCapabilities{
			Native: cli.CompletionNativeCapabilities{Files: true},
		},
	}
	request := cli.CompletionRequest{
		Args:   []string{"upload", "--manifest", ""},
		Cursor: cli.CompletionCursor{Argument: 2},
		Client: client,
	}
	clitest.Complete(t.Context(), app, request).Require(t,
		clitest.CompletionItemsExactly(),
		clitest.CompletionNativeRequestsExactly(clitest.CompletionNativeRequestAt(0)),
		clitest.CompletionNativeRequestAt(0).Kind(cli.CompletionNativeFiles),
	)
}
