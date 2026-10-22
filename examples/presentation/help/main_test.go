package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestDefaultHelpPresentation(t *testing.T) {
	t.Parallel()

	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var root bytes.Buffer
	rootRequest := cli.RunRequest{
		Args:   []string{"--help"},
		Stdout: &root,
	}
	if err := app.Run(t.Context(), rootRequest); err != nil {
		t.Fatalf("Run(--help) error = %v", err)
	}
	rootWant := `Navigate expedition craft across nearby worlds.

Usage:
  alienctl [flags] <command>

Navigation:
  navigate  Navigate the expedition to a destination.
  travel    Navigate the expedition to a destination.
            [alias]
  home      Navigate directly to Earth.
            [shortcut]

Commands:
  help        Show help.
  completion  Generate shell completion source

Flags:
  --verbose  Show navigation detail.
  --help     Show help.
  -h         Show help.
  --version  Show version.

Examples:
Return the expedition home.
  alienctl home
`
	if root.String() != rootWant {
		t.Fatalf("root help = %q, want %q", root.String(), rootWant)
	}

	var navigate bytes.Buffer
	navigateRequest := cli.RunRequest{
		Args:   []string{"navigate", "--help"},
		Stdout: &navigate,
	}
	if err := app.Run(t.Context(), navigateRequest); err != nil {
		t.Fatalf("Run(navigate --help) error = %v", err)
	}
	navigateWant := `Navigate the expedition to a destination.

Usage:
  alienctl navigate [flags] <destination>

Arguments:
  <destination>  World or station to navigate to.

Flags:
  --approach METHOD, --approach=METHOD
    Choose the navigation approach.
    METHOD:
      orbit - Enter a stable orbit.
      teleport - Use the legacy transporter.
        [deprecated 2: transporter navigation is being retired; use warp]
      warp - Use a controlled warp corridor.
      cruise (alias for orbit)
  --token[=TOKEN]
    Authenticate the navigation link.

Global flags:
  --verbose  Show navigation detail.
  --help     Show help.
  -h         Show help.

Examples:
Travel to Europa by warp corridor.
  alienctl navigate europa --approach=warp
`
	if navigate.String() != navigateWant {
		t.Fatalf("navigate help = %q, want %q", navigate.String(), navigateWant)
	}

	var shortcut bytes.Buffer
	shortcutRequest := cli.RunRequest{
		Args:   []string{"help", "home"},
		Stdout: &shortcut,
	}
	if err := app.Run(t.Context(), shortcutRequest); err != nil {
		t.Fatalf("Run(help home) error = %v", err)
	}
	if shortcut.String() != navigateWant {
		t.Fatalf("shortcut help = %q, want canonical target help %q", shortcut.String(), navigateWant)
	}

	var shortcutFlag bytes.Buffer
	shortcutFlagRequest := cli.RunRequest{
		Args:   []string{"home", "--help"},
		Stdout: &shortcutFlag,
	}
	if err := app.Run(t.Context(), shortcutFlagRequest); err != nil {
		t.Fatalf("Run(home --help) error = %v", err)
	}
	if shortcutFlag.String() != navigateWant {
		t.Fatalf("shortcut flag help = %q, want canonical target help %q", shortcutFlag.String(), navigateWant)
	}
}
