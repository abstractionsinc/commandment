# Commandment

Commandment models a CLI once, as typed Go. Compile that declaration into an immutable application, and the same semantic model drives execution, help, completion, schema, diagnostics, and semantic tests instead of maintaining parallel command trees and configuration rules.

Commandment's public authoring API uses Go 1.27 generic methods so commands keep their own configuration types while typed destinations stay typed across arguments, flags, environment input, defaults, policy, and execution. The authoring model does not depend on reflection-driven struct or tag decoding.

## Declare the CLI once

This is a complete `alienctl` application with two independently typed commands.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type scanConfig struct {
	World string
}

type abductConfig struct {
	Target       string
	BeamStrength int
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		scanCmd := cli.Command[scanConfig]{
			Name: "scan",
			Declare: func(cb *cli.CommandBuilder[scanConfig]) {
				cb.Arguments.String(&cb.Config.World, "world")
			},
			Run: func(_ context.Context, cfg *scanConfig) error {
				fmt.Printf("scanning %s\n", cfg.World)
				return nil
			},
		}
		cb.Subcommands.Command(scanCmd)

		abductCmd := cli.Command[abductConfig]{
			Name: "abduct",
			Declare: func(cb *cli.CommandBuilder[abductConfig]) {
				cb.Arguments.String(&cb.Config.Target, "target")

				beamStrength := cb.Values.Int(&cb.Config.BeamStrength)
				cb.Values.Default(beamStrength, 1)
				cb.Values.Environment(beamStrength, "ALIEN_BEAM_STRENGTH")
				cb.Flags.ValueFor(beamStrength).Long("beam-strength")
				cb.Rule.Between(beamStrength, 1, 10)
			},
			Run: func(_ context.Context, cfg *abductConfig) error {
				fmt.Printf("abducting %s with beam strength %d\n", cfg.Target, cfg.BeamStrength)
				return nil
			},
		}
		cb.Subcommands.Command(abductCmd)
	},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.RunFromProcess(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
```

Build it once and use it as a CLI.

```sh
go build -o alienctl .
./alienctl scan mars
ALIEN_BEAM_STRENGTH=2 ./alienctl abduct --beam-strength 4 cow
```

```text
scanning mars
abducting cow with beam strength 4
```

The root `CommandBuilder[struct{}]` composes both `Command[scanConfig]` and `Command[abductConfig]`. Go 1.27 generic methods let command-local configuration stay command-local instead of collapsing the application into one root configuration type.

Inside `abduct`, one `int` destination is shared by a static default, environment input, a flag, and a rule. The flag contributes `4`, the environment contributes `2`, the default contributes `1`, and the handler receives the resolved value as an ordinary `int`.

`Compile` validates the static declaration and publishes an immutable, reusable `Application` only if the declaration is valid.

## Prove the same semantic fact

Commandment did not merely decode strings into a struct. The compiled application retained enough meaning to prove why `BeamStrength` became `4`.

```go
func TestBeamStrengthFlagWins(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	beamStrength := clitest.Command("abduct").Flag("--beam-strength").Value()
	request := cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "ALIEN_BEAM_STRENGTH",
			Value: "2",
		}},
		Args: []string{"abduct", "--beam-strength", "4", "cow"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		beamStrength.Text("4"),
		beamStrength.ContributedBy(cli.ValueProducerKindFlag),
	)
}
```

A handler-output assertion could prove that the application eventually printed `4`. This proves that the flag contribution won resolution. `clitest` asks the semantic observation that owns the fact instead of inferring it from terminal text.

## The same model drives the rest of the CLI

| Area                  | Capabilities                                                                                                                                                                                     |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Grammar               | Nested commands, aliases, shortcuts, default relationships, alternate argument forms, optional and repeated input, remainder/pass-through input, grammar profiles, and dynamic command providers |
| Resolution            | Arguments, flags, environment input, static and invocation defaults, host facts, derivations, dependencies, static and dynamic value domains, and provenance                                     |
| Policy                | Conditions, availability, visibility, blocking rules, advisories, notices, suppression, lifecycle, and declaration replacement                                                                   |
| Interaction           | Structured help, command and flag groups, localization, version presentation, semantic completion, native delegation, and shell integration                                                      |
| Tooling & diagnostics | schema, reference, structured diagnostics and repairs, and invocation inspection                                                                                                                 |
| Runtime               | Explicit host input, middleware, response files, signals, application errors, and process exit policy                                                                                            |
| Verification          | `clitest`, real-shell `shelltest`, ordinary Go tests, and focused `clibench` helpers                                                                                                             |

Commandment is designed for CLIs where grammar, configuration, policy, completion, presentation, tooling, and tests need to agree on the same application model.

## Completion reaches real shells

Completion uses the same semantic subjects as execution. A completer receives the semantic request and contributes structured candidates.

```go
type visitConfig struct {
	World    string
	Manifest string
}

func completeWorld(_ context.Context, _ *visitConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	worlds := []string{"europa", "ganymede", "io"}
	candidates := make([]cli.CompletionCandidate, 0, len(worlds))
	for _, world := range worlds {
		if strings.HasPrefix(world, request.Text) {
			candidates = append(candidates, cli.CompletionCandidate{Value: world})
		}
	}
	contribution := cli.CompletionContribution{Candidates: candidates}
	return contribution, nil
}
```

This excerpt from the `visitConfig` command declaration attaches that semantic completer to one value and delegates another value to the native client for filesystem completion.

```go
world := cb.Arguments.String(&cb.Config.World, "world")
world.CompleteBy(completeWorld)

manifest := cb.Flags.String(&cb.Config.Manifest)
manifest.CompleteWith(cli.CompletionNativeFiles).Long("manifest")
```

Dynamic value domains and invocation-time command providers can participate in the same completion operation. Matching and ranking remain semantic application behavior; shell integrations adapt that behavior to real clients.

| Shell                           | Built-in target |
| ------------------------------- | --------------- |
| Bash                            | 3.2+            |
| Zsh                             | 4.0.1+          |
| Fish                            | 3.x+            |
| Windows PowerShell / PowerShell | 5.1+ / 7+       |
| Elvish                          | 0.11+           |
| Nushell                         | 0.72+           |

Load the Bash integration into the current shell:

```sh
export PATH="$PWD:$PATH"
source <(./alienctl completion bash)
```

Elvish and Nushell can also generate syntax for an explicit supported target version when their integration syntax differs across releases. Custom shell providers adapt the same completion engine instead of defining another command model. `shelltest` exercises shell completion through real shell runtimes.

## Help is another projection of the declaration

A substantial CLI can organize presentation without maintaining a second help tree. Inside the root declaration, two groups are enough to make that organization explicit:

```go
cb.Subcommands.HelpGroup("Field operations", func(commands *cli.Subcommands[struct{}]) {
	commands.Command(scanCmd)
	commands.Command(abductCmd)
}).OrderByName()

cb.Subcommands.HelpGroup("Fleet operations", func(commands *cli.Subcommands[struct{}]) {
	commands.Command(fleetCmd)
	commands.Command(outpostCmd)
}).OrderByName()
```

The resulting root help remains normal CLI output:

```sh
./alienctl --help
```

```text
Usage:
  alienctl [flags] <command>

Field operations:
  abduct  Acquire one specimen.
  scan    Scan a nearby world.

Fleet operations:
  fleet    Manage the fleet.
  outpost  Manage remote outposts.

Commands:
  help        Show help.
  completion  Generate shell completion source

Flags:
  --help  Show help.
  -h      Show help.
```

`Flags.HelpGroup` organizes flags the same way. The presentation model also carries descriptions, examples, lifecycle information, localization, and release/version data. A custom renderer receives a structured `cli.HelpPage` instead of rediscovering the CLI from another registry.

## The application is machine-readable

The compiled CLI is available as detached static data.

```go
schema := app.Schema()
```

Tooling can understand the CLI without scraping `--help` or maintaining another command registry. Structured diagnostics and repairs expose typed failure evidence, while `Inspect` provides the invocation-specific view when a tool needs to explain concrete input without executing the selected handler.

## The same model reaches the process boundary

The complete application above uses `RunFromProcess` to adapt the current process. Hosts that own invocation state can instead call `Run` with explicit caller-owned state instead of reading process globals implicitly. Response-file loading, signal integration, notice delivery, and middleware remain explicit integration boundaries around the same invocation lifecycle.

Application-owned failures remain ordinary Go errors and can carry explicit process status.

```go
return cli.WithExitCode(errLandingDenied, 7)
```

At the executable boundary, `cli.ExitCode(err)` converts the returned error into process status without requiring application erors to implement a framework interface.

## Scale it like ordinary Go

Command-local configuration stays with the command that consumes it. Cohesive packages can own command declarations, policy, execution, and business dependencies without turning the repository into one global framework object.

Static topology remains lexical at useful assembly boundaries. Command providers are for namespaces that are genuinely invocation-dependent, not for hiding ordinary package composition. The compiled `Application` is immutable, while persistence, network clients, service topology, dependency injection, and business logic remain ordinary Go concerns.

[Building Large Applications](docs/12-building-large-applications.md) covers composition in depth. [Adopting Commandment](docs/13-adopting-commandment.md) covers mature existing CLIs.

## Start here

Commandment requires Go 1.27.1. The source is pre-v1 and has no release tag, so [Getting Started](docs/01-getting-started.md) uses the current local-module workflow.

- [Getting Started](docs/01-getting-started.md) - build, run, and semantically test a first application.
- [Documentation](docs/README.md) - learn the complete public programming model.
- [Examples](examples/README.md) - copy focused executable `alienctl` scenarios.
- GoDoc for `cli`, `clitest`, `clibench`, and `shelltest` - look up exact API contracts.

## Dedication

Commandment is dedicated to my first protégé, Eitan.

This project started as a way for me to teach him Go. I was so focused on what he was learning from me that I didn't realize how much I was learning from him too.

Thank you, Eitan :)
