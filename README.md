# Commandment

Commandment is a declarative CLI framework for Go. Application code declares commands, typed values, their input sources, and policy in ordinary Go. `Compile` validates that declaration and returns an immutable `Application` that execution, help, completion, inspection, and tooling can use without rebuilding the same CLI facts in parallel.

Different operations can use different representations. The important part is that a shared semantic fact has one authority. `Declarative` here means that ordinary Go declares those relationships and compilation accepts them into application state. It does not mean YAML, struct tags, or avoiding ordinary Go control flow.

```text
Definition -> Compile -> immutable Application
```

**RC1 community review** · **Go 1.27.1** · **module `github.com/abstractionsinc/commandment`**

Commandment is in RC1 review before final v1.0.0. The review is happening against the current [public source on GitHub](https://github.com/abstractionsinc/commandment). [Getting Started](docs/01-getting-started.md) shows how to build against that checkout.

## Meet `alienctl`

```console
$ ./alienctl scan mars
scanning mars

$ ALIEN_BEAM_STRENGTH=2 ./alienctl abduct --beam-strength 4 cow
abducting cow with beam strength 4
```

`BeamStrength` has a static default of `1`. The environment can contribute `2` through `ALIEN_BEAM_STRENGTH`. The flag contributes `4`. The accepted value must be between `1` and `10`.

The command line is textual, but the application state is typed Go.

```go
type scanConfig struct {
	World string
}

type abductConfig struct {
	Target       string
	BeamStrength int
}
```

Each command owns the configuration its handler actually needs. Commandment does not require one root struct containing every descendant field, and its authoring model does not depend on reflection-driven struct or tag decoding.

The `abduct` declaration gives one typed destination several relationships.

```go
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
```

The default, environment input, flag, and range rule are not separate definitions of `BeamStrength`. They are relationships around the same `int` destination. Resolution chooses the effective contribution, and the handler receives the result as an ordinary `int`.

`int` is only a built-in case. Application-defined Go types can provide a `ValueCodec[T]` for their textual representation while the rest of the CLI continues to refer to the same typed destination.

## Compile the declaration

The root definition composes the commands, then compilation establishes the application boundary.

```go
app, err := cli.Compile(definition)
if err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

if err := app.RunFromProcess(context.Background()); err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(cli.ExitCode(err))
}
```

`Compile` validates the definition and returns an immutable `Application`. Execution, help, completion, inspection, and tooling all work from that compiled result.

For the `abduct` invocation above, resolution has three candidate values and one effective result.

```text
static default  -> 1
environment     -> 2
flag            -> 4
resolved value  -> 4
```

Order matters where Commandment deliberately gives it semantic meaning. It does not become precedence merely because one callback or setup step happened to run later than another.

## Help and completion use the same grammar

The compiled application already knows the command grammar well enough to render root help.

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
  --help  Show help.
  -h      Show help.
```

There is no second command tree for that output. The accepted grammar already contains the commands and flag spellings that help presents.

Completion reads the same grammar. With incomplete input `ab`, the application can determine what may come next.

```go
result, err := app.Complete(context.Background(), cli.CompletionRequest{
	Args:   []string{"ab"},
	Cursor: cli.CompletionCursor{Argument: 0, Byte: 2},
})
```

For this application, the result contains `abduct`.

```text
abduct
```

Shell integration adapts semantic completion to a concrete shell. It does not define another command language.

## Prove why `4` won

Printing `4` proves that the handler received `4`. It does not prove which input source won resolution.

Commandment retains that causal information. `Inspect` can analyze explicit invocation input without running the selected handler, and `clitest` can assert the resulting semantic observation directly.

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

The test proves both the effective value and the winning producer. It does not recreate Commandment's precedence rules or infer them from terminal output. It asks the compiled application what happened.

The same application that executed the command can still explain why the result exists.

## What grows from the compiled model

Real CLIs often need more than primitive flags and a static command tree. The same model supports richer typed values and dynamic command structure without giving the same CLI facts a second owner.

The compiled application can also feed help, diagnostics, inspection, schema, and other tooling. Those surfaces can look different while staying consistent about the CLI facts they share.

Shell, process, and host integrations sit around the same application boundary. Persistence, service topology, dependency injection, and business logic stay in ordinary application code.

[Documentation](docs/README.md) owns the complete programming model and the detailed guides for these areas.

## When Commandment fits

If routing, flags, help, and completion are most of the problem, a smaller command-tree library may be the better fit.

Commandment starts to earn its additional machinery when the CLI itself owns richer semantics that must stay coherent across resolution, policy, completion, diagnostics, tooling, and tests. The design is useful when you need to ask the compiled CLI not only what value it produced, but why that value won and how the same accepted meaning should appear through other operations.

Commandment owns CLI semantics. It is not a general replacement for application configuration, persistence, network clients, or business logic.

## Start building

Start with [Getting Started](docs/01-getting-started.md) to build, run, and semantically test the first application against the current source. [Documentation](docs/README.md) teaches the complete public programming model. [Examples](examples/README.md) provides focused runnable `alienctl` scenarios. Use GoDoc on the exported packages when you need exact local API contracts.

If you find an API or behavior that should change before v1.0.0, [open an issue](https://github.com/abstractionsinc/commandment/issues).
