package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCompletionReturnsMatchingWorld(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	request := cli.CompletionRequest{
		Args: []string{"visit", "eu"},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     2,
		},
	}
	clitest.Complete(t.Context(), app, request).Require(t,
		clitest.CompletionItem("europa").Present(),
		clitest.CompletionItem("ganymede").Absent(),
	)
}
