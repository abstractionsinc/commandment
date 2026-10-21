# Commandment

Commandment is a declarative CLI framework for Go. Application code declares commands, typed values, input sources, policy, and selected runtime behavior in ordinary Go. `Compile` validates that declaration and returns an immutable `Application` that execution, help, completion, structured tooling, and semantic tests can all use.

Shared CLI facts have one semantic authority. Help, completion, tooling, and execution can use different representations without redefining those facts. `Declarative` here means those relationships are authored in ordinary Go and validated by `Compile`. Handlers and application logic remain ordinary Go.

```text
Definition -> Compile -> immutable Application
```

**RC1 community review** · **Go 1.27.1** · **module `github.com/abstractionsinc/commandment`**

Commandment is in RC1 review before final v1.0.0. The review is happening against the current [public source on GitHub](https://github.com/abstractionsinc/commandment). [Getting Started](docs/01-getting-started.md) shows how to build against that checkout.

## Meet `alienctl`

```console
$ ./alienctl --profile survey scan mars
scanning mars with profile survey

$ ALIEN_BEAM_STRENGTH=2 ./alienctl --profile field abduct --beam-strength 4 cow
abducting cow with beam strength 4 using profile field
```

The root owns the shared `--profile` value. `scan` owns a `string` world, while `abduct` owns a target and an `int` beam strength. Both commands explicitly receive the finalized root profile in their own typed handler configuration. The beam strength can come from a static default, the environment, or a flag, and it must remain between `1` and `10`.

## Build a Commandment application

A `Definition[T]` describes the root application. Its `Declare` callback receives a `CommandBuilder[T]`, where that scope declares its CLI behavior. Child commands are `Command[T]` values with their own configuration type, declaration callback, and optional handler.

```text
Definition[rootConfig]
├── --profile
├── Command[scanConfig]   <- accesses profile
└── Command[abductConfig] <- accesses profile
```

This is the complete program.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type rootConfig struct {
	Profile string
}

type scanConfig struct {
	Profile string
	World   string
}

type abductConfig struct {
	Profile      string
	Target       string
	BeamStrength int
}

var definition = cli.Definition[rootConfig]{
	Name: "alienctl",
	Declare: func(root *cli.CommandBuilder[rootConfig]) {
		profile := root.Values.String(&root.Config.Profile)
		root.Values.Default(profile, "default")
		root.Flags.ValueFor(profile).Long("profile")

		scanCmd := cli.Command[scanConfig]{
			Name: "scan",
			Declare: func(scan *cli.CommandBuilder[scanConfig]) {
				scan.Access.Value(profile, &scan.Config.Profile)
				scan.Arguments.String(&scan.Config.World, "world")
			},
			Run: func(_ context.Context, cfg *scanConfig) error {
				fmt.Printf("scanning %s with profile %s\n", cfg.World, cfg.Profile)
				return nil
			},
		}
		root.Subcommands.Command(scanCmd)

		abductCmd := cli.Command[abductConfig]{
			Name: "abduct",
			Declare: func(abduct *cli.CommandBuilder[abductConfig]) {
				abduct.Access.Value(profile, &abduct.Config.Profile)
				abduct.Arguments.String(&abduct.Config.Target, "target")

				beamStrength := abduct.Values.Int(&abduct.Config.BeamStrength)
				abduct.Values.Default(beamStrength, 1)
				abduct.Values.Environment(beamStrength, "ALIEN_BEAM_STRENGTH")
				abduct.Flags.ValueFor(beamStrength).Long("beam-strength")
				abduct.Rule.Between(beamStrength, 1, 10)
			},
			Run: func(_ context.Context, cfg *abductConfig) error {
				fmt.Printf("abducting %s with beam strength %d using profile %s\n", cfg.Target, cfg.BeamStrength, cfg.Profile)
				return nil
			},
		}
		root.Subcommands.Command(abductCmd)
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

`Definition[rootConfig]` owns `--profile` once at the root. `scanConfig` and `abductConfig` still own the state specific to their handlers. Each child uses `Access.Value` to receive the finalized root profile in its execution config. The profile still has one semantic owner instead of being resolved independently in every command.

Inside a `Declare` callback, `Config` refers to that command's typed configuration. The builder declares values, flags, arguments, child commands, and other CLI behavior. The root builder composes the application. `Run` receives resolved configuration as ordinary Go. `Compile` validates the declaration and returns the immutable application. `RunFromProcess` supplies current process state to that same application.

## Model typed input once

`BeamStrength` is one typed destination. Its default, environment input, flag, and range rule all attach to that destination.

```go
beamStrength := abduct.Values.Int(&abduct.Config.BeamStrength)
abduct.Values.Default(beamStrength, 1)
abduct.Values.Environment(beamStrength, "ALIEN_BEAM_STRENGTH")
abduct.Flags.ValueFor(beamStrength).Long("beam-strength")
abduct.Rule.Between(beamStrength, 1, 10)
```

The default can contribute `1`, the environment can contribute `2`, and the flag can contribute `4`. Commandment resolves those sources to one effective value before applying the range rule. The handler does not parse strings, read the environment, reproduce precedence, or run another range check. It reads `cfg.BeamStrength` as an `int`.

Arguments bind to typed configuration the same way. `abduct.Arguments.String(&abduct.Config.Target, "target")` binds the positional `<target>` term directly to the command configuration.

Typed destinations are not limited to Commandment's built-in scalar types. A `ValueCodec[T]` defines how one textual token becomes an application-defined Go type.

```go
type relayConfig struct {
	Address netip.Addr
}

addressCodec := cli.ValueCodec[netip.Addr]{
	Name:     "ip-address",
	Parse:    netip.ParseAddr,
	Format:   netip.Addr.String,
	Snapshot: cli.Value.Assign[netip.Addr](),
}
address := cb.Values.Value(&cb.Config.Address, addressCodec)
cb.Flags.ValueFor(address).Long("address")
```

Here `netip.Addr` comes from the standard library `net/netip` package. With that declaration, CLI text such as `203.0.113.7` becomes a `netip.Addr`. The rest of the CLI can work with that typed destination without converting back and forth through application strings.

## Compile once, use the application

Compilation is the boundary between authoring and use.

```text
Definition
    │
    └── Compile
          │
          └── Application
                ├── run
                ├── help
                ├── completion
                └── Schema
```

The same `alienctl` application can render its root help.

```console
$ ./alienctl --help
Usage:
  alienctl [flags] <command>

Commands:
  scan
  abduct
  help        Show help.
  completion  Generate shell completion source

Flags:
  --profile VALUE, --profile=VALUE
  --help
    Show help.
  -h
    Show help.
```

Completion uses the accepted grammar too. Given incomplete input `ab`, the application returns `abduct`.

```go
result, err := app.Complete(context.Background(), cli.CompletionRequest{
	Args:   []string{"ab"},
	Cursor: cli.CompletionCursor{Argument: 0, Byte: 2},
})
```

```text
abduct
```

Shell adapters turn that completion result into shell-specific integration. Help and completion are expected CLI features, but Commandment does not maintain separate command descriptions just to produce them.

## Use the CLI as structured data

`Application.Schema()` returns a detached static description of the compiled CLI. A tool can inspect command, flag, value, environment, provider, domain, and signal relationships without scraping help text or mutating the application.

For the same `alienctl` program, this code follows the environment input back to its typed destination and owning command.

```go
schema := app.Schema()
for _, input := range schema.EnvironmentInputs {
	if input.Name != "ALIEN_BEAM_STRENGTH" {
		continue
	}
	value := schema.Values[input.Destination]
	command := schema.Commands[value.Command]
	fmt.Printf("%s owns %s, default=%t\n", command.Name, input.Name, value.HasStaticDefault)
}
```

```text
abduct owns ALIEN_BEAM_STRENGTH, default=true
```

Schema is intentionally static. It describes what compilation accepted, including dynamic command-provider declarations, without executing those providers or folding invocation-dependent commands into the static schema.

`Application.Reference(...)` provides a presentation-oriented view for documentation and reference tooling.

## Let resolved input shape the command namespace

Some CLIs cannot know every command from static source alone. The available namespace may depend on resolved input.

The dynamic `alienctl` example uses `--system` to decide which outpost command exists.

```go
type outpostConfig struct {
	System string
}

outpostCmd := cli.Command[outpostConfig]{
	Name: "outpost",
	Declare: func(cb *cli.CommandBuilder[outpostConfig]) {
		system := cb.Flags.String(&cb.Config.System)
		system.Long("system")
		cb.Rule.Require(system)

		cb.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[outpostConfig]) error {
			switch batch.Config.System {
			case "jovian":
				europaCmd := cli.Command[struct{}]{
					Name: "europa",
					Run: func(context.Context, *struct{}) error {
						fmt.Println("contacting europa outpost")
						return nil
					},
				}
				batch.Command(europaCmd)
			case "saturnian":
				titanCmd := cli.Command[struct{}]{
					Name: "titan",
					Run: func(context.Context, *struct{}) error {
						fmt.Println("contacting titan outpost")
						return nil
					},
				}
				batch.Command(titanCmd)
			}
			return nil
		}, system)
	},
}
```

`outpost` is static. Its children are not. Passing `system` to `CommandsFrom` declares that dependency, so Commandment resolves `--system` before asking the provider which child commands are available. The returned `CommandBatch` is then used for command selection, help, completion, and anything else that needs the current namespace.

```console
$ ./alienctl outpost --system jovian europa
contacting europa outpost
```

A provider batch is atomic. If the provider fails, none of the commands in that batch become available.

## Declare signal behavior with the command

Signal handling often lives beside the CLI as separate process setup. Commandment can keep selected signal behavior with the command that uses it.

```go
type listenConfig struct {
	Events <-chan cli.SignalEvent
}

listenCmd := cli.Command[listenConfig]{
	Name: "listen",
	Declare: func(cb *cli.CommandBuilder[listenConfig]) {
		events := cb.Signals.Endpoint(&cb.Config.Events)
		interrupt := cb.Signals.Match.Interrupt().During(cli.SignalPhaseExecution).Each().Hint("interrupt")
		cb.Signals.Policy(interrupt, cb.Signals.Action.Emit(events))
	},
	Run: func(_ context.Context, cfg *listenConfig) error {
		event := <-cfg.Events
		fmt.Println(event.Signal)
		return nil
	},
}
```

The declaration routes matching interrupt events into the typed channel that the handler receives through its own configuration. The signal policy stays with the command instead of being hidden in separate startup wiring.

Commandment does not need to own every process concern. Process behavior that is not part of the CLI can stay in ordinary application code.

## Test CLI semantics with `clitest`

Ordinary Go tests are still the right place for your application behavior. `clitest` is for behavior owned by Commandment itself, such as compiled structure, resolution, provenance, diagnostics, and operation results.

The `BeamStrength` test can assert both the effective value and the producer that supplied it.

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

A stdout assertion could prove that the handler printed `4`. This test proves the Commandment-owned semantic fact that the flag supplied the effective value, without reproducing Commandment's resolution logic in test code.

Use `shelltest` for real-shell completion tests and `clibench` for focused operation benchmarks built on ordinary `testing.B`.

## Where the model goes next

The deeper documentation continues in three directions.

- **Richer CLI semantics** include value domains, derivations, facts, conditions, lifecycle, response files, grammar profiles, localization, and related policy.
- **Observation and tooling** include diagnostics and repairs, deeper schema/reference use, direct inspection, custom presentation, and generated artifacts.
- **Runtime and integration** include host facts and defaults, middleware, signal sources, shell adapters, application errors, and larger application composition.

[Documentation](docs/README.md) covers these topics in depth. [Examples](examples/README.md) provides focused runnable `alienctl` scenarios when a complete program helps.

## When Commandment fits

If routing, flags, help, and completion are most of the problem, a smaller command-tree library may be the better fit.

Commandment starts to earn its additional machinery when typed resolution, policy, dynamic grammar, tooling, process behavior, and tests all need to agree about the same CLI. The value is not that every part uses one literal representation. It is that shared CLI facts keep one owner as more parts of the application use them.

Commandment owns CLI semantics. Persistence, service topology, dependency injection, network clients, and business logic remain ordinary application concerns unless they need to participate directly in the CLI.

## Start building

Start with [Getting Started](docs/01-getting-started.md) to build your first application from scratch. Use the [documentation index](docs/README.md) for the complete programming model, [examples](examples/README.md) for focused runnable scenarios, and GoDoc for exact API details.

If you find an API or behavior that should change before v1.0.0, [open an issue](https://github.com/abstractionsinc/commandment/issues).
