# Getting Started

Build one `alienctl` command, give one typed value three input sources, add a rule, and prove which source won without running the handler.

You will finish with this command.

```text
$ ALIEN_BEAM_STRENGTH=2 go run . abduct --beam-strength 4 --dry-run cow
would abduct cow with beam strength 4
```

## Prepare an empty module

Commandment requires Go 1.27.1 and is pre-v1. This source snapshot has no release tag, so use an exact local checkout for now.

```sh
mkdir alienctl
cd alienctl
go mod init example.com/alienctl
go mod edit -require=github.com/abstractionsinc/commandment@v0.0.0
go mod edit -replace=github.com/abstractionsinc/commandment=/absolute/path/to/commandment
```

Replace `/absolute/path/to/commandment` with the absolute path of this repository's Commandment module. Remove the local `replace` and use an exact tag once a release is published.

## Get a command running

Create `main.go`.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type abductConfig struct {
	Target       string
	BeamStrength int
	DryRun       bool
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[abductConfig]{
			Name: "abduct",
			Declare: func(cb *cli.CommandBuilder[abductConfig]) {
				cb.Arguments.String(&cb.Config.Target, "target")
				cb.Flags.Int(&cb.Config.BeamStrength).Long("beam-strength")
				cb.Flags.Bool(&cb.Config.DryRun).Long("dry-run")
			},
			Run: func(_ context.Context, cfg *abductConfig) error {
				verb := "abducting"
				if cfg.DryRun {
					verb = "would abduct"
				}
				fmt.Printf("%s %s with beam strength %d\n", verb, cfg.Target, cfg.BeamStrength)
				return nil
			},
		}
		cb.Subcommands.Command(cmd)
	},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.RunFromProcess(context.Background()); err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
```

Run it.

```sh
go mod tidy
go run . abduct --beam-strength 3 --dry-run cow
```

```text
would abduct cow with beam strength 3
```

The `definition` is authored source input. Its declaration callback receives a callback-scoped `cli.CommandBuilder`, and `cli.Compile` validates the declaration and returns an immutable `cli.Application`. Keep one-use declaration callbacks lexical; do not retain a builder after its callback or use it concurrently.

## Let one value come from three places

`BeamStrength` currently comes only from a flag. Replace that declaration with one destination and three producers.

```go
beamStrength := cb.Values.Int(&cb.Config.BeamStrength)
cb.Values.Default(beamStrength, 1)
cb.Values.Environment(beamStrength, "ALIEN_BEAM_STRENGTH")
cb.Flags.ValueFor(beamStrength).Long("beam-strength")
```

The destination is still one `int` in `abductConfig`. A static default, environment input, and flag can each contribute a candidate to it.

```sh
ALIEN_BEAM_STRENGTH=2 go run . abduct --beam-strength 4 cow
```

```text
abducting cow with beam strength 4
```

The flag wins under the default precedence. Remove it and the environment supplies `2`; remove both and the static default supplies `1`. The handler still reads only `cfg.BeamStrength`.

## Validate the resolved value

Add a range rule after the producers.

```go
cb.Rule.Between(beamStrength, 1, 10)
```

A rule is blocking validation policy over Commandment semantic state. An out-of-range value is rejected before the handler runs, so the handler does not need a second range check.

## Prove why the value is 4

Create `main_test.go`.

```go
package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

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

Run the test with ordinary Go tooling.

```sh
go test ./...
```

The test compiles the same `definition` and asks the public `Inspect` operation why the value resolved to `4`. Handler output could show that the application received `4`; it would not prove that the flag contribution won resolution.

## What you built

One declaration now owns the command grammar, the `BeamStrength` destination, its three producers, the range rule, and the handler. The compiled application uses that same model for execution and inspection, while `clitest` proves the semantic result without maintaining another parser or precedence model.

Continue with the [Commandment Guide](02-commandment-guide.md) for declaration lifetime, static and invocation-time state, operation boundaries, and the rest of the programming model. Use the [examples corpus](../examples/README.md) when you want a complete runnable scenario for one focused capability.
