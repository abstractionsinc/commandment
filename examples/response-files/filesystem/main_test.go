package main

import (
	"os"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestFilesystemResponseFileSuppliesArgument(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("crew.args")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	load := cli.ResponseFileRequest{
		Path: "crew.args",
		Base: ".",
	}
	file := cli.ResponseFile{
		Key:         "crew.args",
		Location:    "crew.args",
		IncludeBase: ".",
		Content:     content,
	}
	files := clitest.ResponseFiles{
		Loads: map[cli.ResponseFileRequest]cli.ResponseFile{load: file},
	}
	request := cli.InspectionRequest{
		Args:             []string{"hail", "@crew.args"},
		WorkingDirectory: ".",
		ResponseFiles:    files.Access(),
	}
	app := clitest.Compile(t, definition)
	name := clitest.Command("hail").ArgumentFormAt(0).Term("name")
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		name.Value().Text("Zorblax"),
	)
}
