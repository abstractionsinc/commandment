# Commandment

A declarative CLI framework for Go.

Declare your CLI once. Compile it into one model for execution, Help, completion, diagnostics, tooling, host integration, and tests.

[Getting Started](docs/01-getting-started.md) | [Documentation](docs/README.md) | [Examples](examples/README.md)

RC2 | Go 1.27.1 | module `github.com/abstractionsinc/commandment`

Commandment compiles one Go declaration into an immutable application. The same declarations drive execution, Help, completion, diagnostics, tooling, host integration, and tests.

```text
Definition -> Compile -> Application
                            |
                            +-> Run
                            +-> Inspect
                            +-> Complete
                            +-> Schema
                            +-> Reference
                            +-> Help and version
                            +-> shell integration
                            `-> clitest
```

Handlers and business logic remain ordinary Go.

## Contents

- [Build a real CLI](#build-a-real-cli)
- [Give values one typed meaning](#give-values-one-typed-meaning)
- [Shape the command language](#shape-the-command-language)
- [Conditions, rules, diagnostics, and notices](#conditions-rules-diagnostics-and-notices)
- [Help that stays correct](#help-that-stays-correct)
- [Evolve a CLI across releases](#evolve-a-cli-across-releases)
- [Localize presentation](#localize-presentation)
- [Completion](#completion)
- [Inspect and describe the application](#inspect-and-describe-the-application)
- [Test Commandment applications with clitest](#test-commandment-applications-with-clitest)
- [Test shell integrations with shelltest](#test-shell-integrations-with-shelltest)
- [Benchmark Commandment applications with clibench](#benchmark-commandment-applications-with-clibench)
- [Build commands from runtime data](#build-commands-from-runtime-data)
- [Supply values from the host](#supply-values-from-the-host)
- [Keep execution ordinary Go](#keep-execution-ordinary-go)
- [Response files](#response-files)
- [Signals](#signals)
- [Scale the source](#scale-the-source)
- [When Commandment fits](#when-commandment-fits)
- [Start building](#start-building)

## Build a real CLI

A useful Commandment program is ordinary Go. It has typed configuration, a declaration, a handler, one compile step, and a thin process entrypoint.

This `alienctl` command abducts a target. Beam strength can come from a default, the environment, or a flag. The accepted value must stay between `1` and `10`.

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

func runAbduct(_ context.Context, config *abductConfig) error {
	verb := "abducting"
	if config.DryRun {
		verb = "would abduct"
	}
	fmt.Printf("%s %s with beam strength %d\n", verb, config.Target, config.BeamStrength)
	return nil
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(root *cli.CommandBuilder[struct{}]) {
		command := cli.Command[abductConfig]{
			Name: "abduct",
			Declare: func(command *cli.CommandBuilder[abductConfig]) {
				command.Arguments.String(&command.Config.Target, "target")

				beamStrength := command.Values.Int(&command.Config.BeamStrength)
				command.Values.Default(beamStrength, 1)
				command.Values.Environment(beamStrength, "ALIEN_BEAM_STRENGTH")
				command.Flags.ValueFor(beamStrength).Long("beam-strength")
				command.Rule.Between(beamStrength, 1, 10)

				command.Flags.Bool(&command.Config.DryRun).Long("dry-run")
			},
			Run: runAbduct,
		}
		root.Subcommands.Command(command)
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

The same destination accepts every source declared for it.

```console
$ ./alienctl abduct cow
abducting cow with beam strength 1

$ ALIEN_BEAM_STRENGTH=2 ./alienctl abduct cow
abducting cow with beam strength 2

$ ALIEN_BEAM_STRENGTH=2 ./alienctl abduct --beam-strength 4 --dry-run cow
would abduct cow with beam strength 4
```

The rule runs before application code.

```console
$ ./alienctl abduct --beam-strength 20 cow
alienctl: --beam-strength must be between 1 and 10
```

The handler never parses the flag, reads the environment, chooses which source wins, or repeats the range check. It receives the admitted typed configuration for the invocation.

`cli.Compile` freezes the declaration into an immutable `cli.Application`. Runtime operations keep their own invocation state rather than turning the definition into a mutable registry.

[`Getting Started`](docs/01-getting-started.md) builds a complete application from an empty Go module.

## Give values one typed meaning

A Commandment value is not a flag field or an environment field. It is one typed destination that can receive contributions from several producers.

```go
beamStrength := command.Values.Int(&command.Config.BeamStrength)
command.Values.Default(beamStrength, 1)
command.Values.Environment(beamStrength, "ALIEN_BEAM_STRENGTH")
command.Flags.ValueFor(beamStrength).Long("beam-strength")
```

The destination owns resolution. Application code reads one `int` regardless of where the winning contribution came from.

```go
command.Values.ResolveWith(beamStrength, cli.Value.KeepLast())
```

Built-in resolution policies include `KeepFirst`, `KeepLast`, `RejectMultipleTiers`, and `CombineTiers`. Absence is also represented separately from a resolved Go zero value, so tools do not have to guess whether `0`, `""`, or `false` means "not supplied."

### Use your own Go types

Commandment does not need built-in knowledge of every type an application may use. `ValueCodec[T]` teaches the value system how one scalar type parses from text, formats back to text, and is snapshotted when Commandment retains it.

```go
import "net/netip"

type relayConfig struct {
	Address netip.Addr
}

addressCodec := cli.ValueCodec[netip.Addr]{
	Name:     "ip-address",
	Parse:    netip.ParseAddr,
	Format:   netip.Addr.String,
	Snapshot: cli.Value.Assign[netip.Addr](),
}

address := command.Values.Value(&command.Config.Address, addressCodec)
command.Flags.ValueFor(address).Long("address")
```

```console
$ ./alienctl relay --address 203.0.113.7
relaying through 203.0.113.7
```

Custom codecs implement the same `ValueLanguage[T]` contract used by Commandment's built-in scalar languages. That is the extension point for application-specific configuration types without bolting a second parsing system onto the CLI. Arguments, flags, domains, completion, diagnostics, artifacts, and tests can continue to work with the typed value.

Token languages and keyed value languages cover different shapes when one semantic value owns several tokens or key/value syntax. Command routing and token ownership remain grammar concerns rather than being hidden inside a scalar codec.

### Limit values to real choices

A finite domain makes the legal choices part of the value itself.

```go
declareApproaches := func(values *cli.ValueDomainBuilder[string]) {
	values.Value("orbit", "orbit")
	values.Value("land", "land")
	values.Value("flyby", "flyby")
}

approaches := cli.Value.Domain(cli.Value.Assign[string](), declareApproaches)
approach := command.Values.Value(&command.Config.Approach, approaches)
command.Values.Default(approach, "orbit")
command.Flags.ValueFor(approach).Long("approach")
```

```console
$ ./alienctl visit europa
visiting europa by orbit

$ ./alienctl visit --approach land europa
visiting europa by land
```

The domain is not only validation data. Help, completion, inspection, diagnostics, and tests can all see the same accepted values.

See [`values/choices`](examples/values/choices/).

### Let one value control another value's choices

A domain can depend on values already resolved for this invocation. `alienctl visit` uses the selected planetary system to decide which moons are legal.

```go
declareSystems := func(values *cli.ValueDomainBuilder[string]) {
	values.Value("jovian", "jovian")
	values.Value("saturnian", "saturnian")
}

systems := cli.Value.Domain(cli.Value.Assign[string](), declareSystems)
system := command.Values.Value(&command.Config.System, systems)
command.Flags.ValueFor(system).Long("system")
command.Rule.Require(system)

declareMoons := func(_ context.Context, _ *visitConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
	resolvedSystem, _ := dependencies.Get(system)
	switch resolvedSystem {
	case "jovian":
		domain.Value("europa", "europa")
		domain.Value("ganymede", "ganymede")
	case "saturnian":
		domain.Value("titan", "titan")
		domain.Value("enceladus", "enceladus")
	}
	return nil
}

moons := command.Values.Domain(declareMoons, system)
moon := command.Values.Value(&command.Config.Moon, moons)
command.Arguments.ValueFor(moon, "moon")
```

```console
$ ./alienctl visit --system jovian europa
visiting europa in the jovian system

$ ./alienctl visit --system saturnian titan
visiting titan in the saturnian system
```

Passing `system` to `Values.Domain` declares the dependency. Commandment settles that value before building the moon domain, then uses the resulting domain everywhere else that needs to understand the value.

See [`values/dynamic-choices`](examples/values/dynamic-choices/).

### Compute configuration from resolved input

Some application values are derived from other resolved values rather than supplied directly by the user.

```go
deriveSummary := func(_ context.Context, _ *abductConfig, dependencies cli.Dependencies) (string, error) {
	resolvedTarget, _ := dependencies.Get(target)
	resolvedBeamStrength, _ := dependencies.Get(beamStrength)
	summary := fmt.Sprintf("abducting %s with beam strength %d", resolvedTarget, resolvedBeamStrength)
	return summary, nil
}

summary := cli.Value.ComputedBy(deriveSummary).
	From(target, beamStrength)

command.Values.Derive(&command.Config.Summary, summary)
```

The handler receives the finished `Summary`. It does not need to reproduce the dependency or recompute the value after Commandment has already resolved the inputs.

`.From(...)` also tells Commandment which values the computation depends on. That relationship can be validated for scope and reachability instead of being hidden in callback code.

### Protect sensitive values

Mark the destination sensitive when the semantic value itself should be redacted from Commandment-owned presentation and observations.

```go
token := command.Values.String(&command.Config.Token).Sensitive()
command.Flags.ValueFor(token).Long("token")
```

A single producer can be sensitive without making every producer for the destination sensitive.

```go
token := command.Values.String(&command.Config.Token)
command.Values.Environment(token, "ALIEN_TOKEN").Sensitive()
command.Flags.ValueFor(token).Long("token")
```

Flag spellings and implicit or fixed values can carry the same producer-level sensitivity when that is the right owner.

Sensitivity controls Commandment's presentation, provenance, and observation behavior. It is not encryption and it does not replace application authorization.

[`Values and Resolution`](docs/04-values-and-resolution.md) covers producers, resolution tiers, value languages, domains, dependencies, provenance, sensitivity, and structured repair in full.

## Shape the command language

Commandment treats commands, positional forms, flag spellings, aliases, shortcuts, defaults, profiles, and dialect choices as part of the CLI language rather than handler-side parsing conventions.

### Nested commands

```go
fleet := cli.Command[struct{}]{
	Name: "fleet",
	Declare: func(fleet *cli.CommandBuilder[struct{}]) {
		status := cli.Command[struct{}]{
			Name: "status",
			Run: func(context.Context, *struct{}) error {
				fmt.Println("fleet status: 3 ships ready")
				return nil
			},
		}
		fleet.Subcommands.Command(status)

		launch := cli.Command[launchConfig]{
			Name: "launch",
			Declare: func(launch *cli.CommandBuilder[launchConfig]) {
				launch.Arguments.String(&launch.Config.Ship, "ship")
			},
			Run: runLaunch,
		}
		fleet.Subcommands.Command(launch)
	},
}
root.Subcommands.Command(fleet)
```

```console
$ ./alienctl fleet status
fleet status: 3 ships ready

$ ./alienctl fleet launch scout-7
launching scout-7
```

See [`commands/nested`](examples/commands/nested/).

### Aliases and shortcuts

An alias is another spelling for the same command.

```go
scan := cli.Command[scanConfig]{
	Name:    "scan",
	Aliases: cli.Aliases("survey"),
	Declare: func(scan *cli.CommandBuilder[scanConfig]) {
		scan.Arguments.String(&scan.Config.World, "world")
	},
}
```

```console
$ ./alienctl scan mars
scanning mars

$ ./alienctl survey mars
scanning mars
```

A shortcut is a distinct entry that targets a canonical command and can contribute bounded input.

```go
navigate := cli.Command[navigateConfig]{
	Name: "navigate",
	Declare: func(navigate *cli.CommandBuilder[navigateConfig]) {
		navigate.Arguments.String(&navigate.Config.Destination, "destination")

		home := cli.CommandShortcut{
			Name: "home",
			Args: []string{"earth"},
		}
		navigate.Shortcut(home)
	},
}
```

```console
$ ./alienctl navigate mars
navigating to mars

$ ./alienctl home
navigating to earth
```

Aliases change spelling. Shortcuts change the invocation shape. Keeping them distinct lets Help, lifecycle, inspection, and tests preserve the difference.

See [`commands/aliases`](examples/commands/aliases/) and [`commands/shortcuts`](examples/commands/shortcuts/).

### Default commands

A parent can select one immediate child when the user does not name a child explicitly.

```go
status := cli.Command[struct{}]{
	Name: "status",
	Run:  runFleetStatus,
}
fleet.Subcommands.Command(status).Default()
```

```console
$ ./alienctl fleet
fleet status: 3 ships ready

$ ./alienctl fleet status
fleet status: 3 ships ready
```

The default belongs to this exact parent-to-child relationship. The reusable command value does not have to carry default policy with it.

Conditional defaults use the same declaration handle through `DefaultWhen`.

See [`commands/default`](examples/commands/default/).

### Alternate positional forms

Some commands really do accept more than one positional grammar.

```go
navigate.Arguments.Form(func(form *cli.ArgumentFormBuilder[navigateConfig]) {
	form.String(&form.Config.World, "world")
})

navigate.Arguments.Form(func(form *cli.ArgumentFormBuilder[navigateConfig]) {
	form.Int(&form.Config.X, "x")
	form.Int(&form.Config.Y, "y")
	form.Int(&form.Config.Z, "z")
})
```

```console
$ ./alienctl navigate mars
navigating to mars

$ ./alienctl navigate 12 8 4
navigating to coordinates 12,8,4
```

Commandment keeps the two forms distinct instead of flattening them into ambiguous optional-token notation.

Optional arguments use the same grammar model.

```console
$ ./alienctl status
mothership status: nominal

$ ./alienctl status scout-7
scout-7 status: nominal
```

See [`arguments/forms`](examples/arguments/forms/) and [`arguments/optional`](examples/arguments/optional/).

### Pass-through arguments

A wrapper command can deliberately hand the rest of the command line to another consumer.

```go
forwarded := relay.Values.Sequence(&relay.Config.Command, cli.Value.String())
relay.Arguments.Form(func(form *cli.ArgumentFormBuilder[relayConfig]) {
	form.String(&form.Config.Rover, "rover")
	form.PassThrough(forwarded, "command", cli.Count.OneOrMore())
})
```

```console
$ ./alienctl relay rover-7 -- echo hello earth
relaying to rover-7: echo hello earth
```

Everything after `--` belongs to the pass-through term. Commandment does not reinterpret those tokens as `alienctl` flags or commands.

See [`arguments/pass-through`](examples/arguments/pass-through/).

### Flag spellings and counting

One semantic flag can accept several spellings.

```go
scanRange := scan.Flags.Int(&scan.Config.Range)
scanRange.Long("range")
scanRange.Short('r')
scan.Rule.Require(scanRange)
```

```console
$ ./alienctl scan --range 5 mars
scanning mars at range 5

$ ./alienctl scan -r 5 mars
scanning mars at range 5
```

Repeated counting flags are first-class too.

```go
scan.Flags.Count(&scan.Config.Verbosity).Short('v')
```

```console
$ ./alienctl scan -v -v -v mars
scanning mars with verbosity 3
```

See [`flags/spellings`](examples/flags/spellings/) and [`flags/counting`](examples/flags/counting/).

### Grammar profiles

One command can expose mutually exclusive grammar variants selected by an early typed value.

```go
configureProfiles := func(cases *cli.GrammarProfileCases[transmitConfig, string]) {
	cases.Case("brief", func(*cli.CommandBuilder[transmitConfig]) {})
	cases.Case("verbose", func(transmit *cli.CommandBuilder[transmitConfig]) {
		transmit.Flags.Bool(&transmit.Config.Verbose).
			Occurrences(cli.Count.Optional()).
			Long("verbose")
	})
}

transmit.Profiles.Switch(profile, configureProfiles)
```

The selected profile remains visible to analysis, completion, Schema, Reference, and tests. Use separate commands instead when the alternatives are really different operations rather than different grammar for one operation.

### CLI recognition rules

`Dialect` changes how a command recognizes CLI syntax. Use it when compatibility or product requirements genuinely need a different language.

For example, short flags do not cluster by default. A command that deliberately accepts clustered spellings can opt in.

```go
scan.Dialect.ShortFlagClustering(cli.ShortFlagClusteringEnabled)
```

```console
$ ./alienctl scan -vvv mars
scanning mars with verbosity 3
```

The dialect also owns flag ordering, name matching and normalization, dashless short flags, end-of-options tokens, response-file recognition, and default flag placement.

Changing dialect is a grammar decision. It changes what user input means, so the default should remain in place unless the CLI actually needs another recognition policy.

[`Commands and Grammar`](docs/03-commands-and-grammar.md) covers the complete language model.

## Conditions, rules, diagnostics, and notices

A CLI should reject bad input before application code has to reason about it. Commandment keeps the grammar decision, the semantic failure, the human error, and the structured diagnostic connected to the same declaration.

### Conditional syntax

`--cloak-level` only makes sense when stealth mode is enabled. It does not need to be accepted everywhere and rejected later in the handler.

```go
stealth := approach.Flags.Bool(&approach.Config.Stealth).
	Occurrences(cli.Count.Optional())
stealth.Long("stealth")

cloakLevel := approach.Flags.Int(&approach.Config.CloakLevel)
cloakLevel.
	AvailableWhen(approach.When.Equal(stealth, true)).
	Long("cloak-level")
```

```console
$ ./alienctl approach --stealth --cloak-level 3 earth
approaching earth under cloak level 3

$ ./alienctl approach --cloak-level 3 earth
alienctl: flag --cloak-level is unavailable
  suggestion: --cloak-level
```

Availability changes whether the declaration participates in the effective CLI. Help, completion, inspection, diagnostics, and execution all see the same result.

Visibility is separate. A declaration can remain valid while being omitted from ordinary presentation.

See [`policies/conditions`](examples/policies/conditions/).

### Rules and diagnostics

Rules validate settled semantic values before the handler runs.

```go
speed := approach.Flags.Int(&approach.Config.Speed)
speed.Long("speed")
approach.Rule.Require(speed)
approach.Rule.Between(speed, 1, 5)
```

The user gets a direct CLI error without application code formatting it.

```console
$ ./alienctl approach earth
alienctl: --speed is required

$ ./alienctl approach --speed 9 earth
alienctl: --speed must be between 1 and 5
```

`cli.WriteError` owns the normal writer-aware presentation of Commandment failures.

```go
if err := app.RunFromProcess(context.Background()); err != nil {
	_ = cli.WriteError(os.Stderr, err)
	os.Exit(cli.ExitCode(err))
}
```

The same error keeps structured diagnostics for programs that need more than rendered text.

```go
request := cli.RunRequest{
	Args: []string{"approach", "--speed", "9", "earth"},
}
err := app.Run(context.Background(), request)
report := cli.Diagnostics(err)

for _, diagnostic := range report.Diagnostics {
	fmt.Println(diagnostic.Code)
}
```

```text
rule-violation
```

A tool can work with diagnostic identity, declaration relationships, safe input summaries, source spans, rejection details, repairs, and causal errors without scraping the terminal message.

Application errors remain ordinary Go errors. `Diagnostics` returns an empty report when an error carries no Commandment diagnostic.

### Structured repairs

A parser can reject input and attach a machine-readable replacement at the same time.

```go
parseDockCode := func(_ context.Context, tokens []string) (string, error) {
	code := tokens[0]
	if code == "A1" {
		return code, nil
	}
	rejection := cli.Rejection.Message(
		"dock-code",
		"invalid dock code",
		cli.Rejection.At(0, 0, len(code)),
		cli.Rejection.ReplaceWith("A1"),
	)
	return "", rejection
}

formatDockCode := func(code string) []string {
	return []string{code}
}

dockCode := cli.TokenLanguage[string]{
	Name:   "dock-code",
	Count:  cli.Count.Exactly(1),
	Parse:  parseDockCode,
	Format: formatDockCode,
}
```

The terminal still gets a normal error.

```console
$ ./alienctl dock ZZ
alienctl: invalid dock code
  input: ZZ
```

The diagnostic report also contains an `InputRepair` pointing at the original source and replacing `ZZ` with `A1`.

```go
report := cli.Diagnostics(err)
diagnostic := report.Diagnostics[0]
repair := report.Repairs[diagnostic.RepairIndexes[0]]
edit := repair.Sources[0].TextEdits[0]

fmt.Println(edit.Replacement[0].Literal)
```

```text
A1
```

Commandment does not silently apply the repair. A terminal, editor, IDE, or another client decides how to present or use it.

### Advisories and notices

Not every important fact should fail the invocation. An advisory records a nonfatal semantic outcome.

```go
advisory := cli.Advisory{
	Code:     "reckless-approach",
	Severity: cli.NoticeSeverityWarning,
	Message:  "reckless approach enabled",
}

approach.Advisory.
	When(approach.When.Equal(reckless, true), advisory).
	Subject(reckless)
```

The host decides how notices are delivered.

```go
notify := func(_ context.Context, notice cli.Notice) error {
	_, err := fmt.Fprintf(os.Stderr, "warning: %s\n", notice.Message)
	return err
}

request := cli.RunRequest{
	Args:   os.Args[1:],
	Notify: notify,
}
err := app.Run(context.Background(), request)
```

```console
$ ./alienctl approach --reckless earth
warning: reckless approach enabled
approaching earth
```

Notice suppression changes delivery without erasing the advisory that produced the notice.

```go
approach.Notice.SuppressWhen(
	approach.When.Equal(quiet, true),
	cli.NoticeMatch{
		AdvisoryCode: advisory.Code,
	},
)
```

```console
$ ./alienctl approach --reckless --quiet earth
approaching earth
```

The advisory is still part of the semantic result. The matching notice is simply not delivered for that invocation.

See [`policies/advisories`](examples/policies/advisories/) and [`Troubleshooting`](docs/11-troubleshooting.md).

## Help that stays correct

Commandment renders default Help from the accepted application model. There is no second usage grammar to keep synchronized with the command declaration.

A realistic `alienctl` definition can produce this root page.

```console
$ ./alienctl --help
Navigate expedition craft across nearby worlds.

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
```

The declarations behind that page are ordinary command metadata and grammar.

```go
navigate := cli.Command[navigateConfig]{
	Name:    "navigate",
	Aliases: cli.Aliases("travel"),
	Description: cli.Description{
		Long: "Navigate the expedition to a destination.",
	},
	Examples: []cli.Example{
		{
			Description: "Travel to Europa by warp corridor.",
			CommandLine: "alienctl navigate europa --approach=warp",
		},
	},
	Declare: func(navigate *cli.CommandBuilder[navigateConfig]) {
		navigate.Arguments.String(&navigate.Config.Destination, "destination").
			Description("World or station to navigate to.")

		home := cli.CommandShortcut{
			Name: "home",
			Description: cli.Description{
				Short: "Navigate directly to Earth.",
			},
			Args: []string{"earth"},
		}
		navigate.Shortcut(home)
	},
}
```

Descriptions, groups, aliases, shortcuts, flags, arguments, domains, lifecycle state, sensitive input, examples, and inherited scope all remain available to nested Help.

```console
$ ./alienctl navigate --help
Navigate the expedition to a destination.

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
```

The implicit token is sensitive, so Help knows the optional value exists without printing the hidden value.

The visible shortcut resolves through Help too.

```console
$ ./alienctl home --help
Navigate the expedition to a destination.

Usage:
  alienctl navigate [flags] <destination>
...
```

See [`presentation/help`](examples/presentation/help/).

### Custom Help

Applications that need another presentation can replace the renderer without rebuilding command selection or Help semantics.

```go
renderHelp := func(_ context.Context, page cli.HelpPage) (string, error) {
	text := fmt.Sprintf(
		"%s %s\n%s\n",
		strings.ToUpper(page.ApplicationName),
		page.Name,
		page.Description.Short,
	)
	return text, nil
}

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Help: cli.HelpConfig{
		Renderer: renderHelp,
	},
}
```

```console
$ ./alienctl help scan
ALIENCTL scan
scan a nearby world
```

The custom renderer receives a structured `cli.HelpPage`. It does not have to rediscover which command was selected or reconstruct the command grammar.

See [`presentation/custom-help`](examples/presentation/custom-help/).

## Evolve a CLI across releases

Commandment can treat release evolution as application semantics instead of scattered version strings and deprecation prose.

The definition owns the current release and the comparison scheme.

```go
var definition = cli.Definition[struct{}]{
	Name:    "alienctl",
	Release: cli.Release.SemVer("2.0.0"),
	Version: cli.VersionConfig{},
}
```

The built-in version action uses that authority.

```console
$ ./alienctl --version
2.0.0
```

Declarations can carry the release history of the CLI surface itself. A renamed flag can say when the old spelling became deprecated and identify the spelling that replaced it.

```go
speed := approach.Flags.Int(&approach.Config.Speed).
	Description("Set approach speed.")

legacy := speed.Long("velocity").Release(cli.ReleaseLifecycle{
	IntroducedIn: "1.0.0",
	DeprecatedIn: "2.0.0",
	Reason:       "velocity was renamed",
})
current := speed.Long("speed")
legacy.ReplacedBy(current)
```

At release `2.0.0`, Help can explain both facts without another hand-maintained deprecation table.

```console
$ ./alienctl approach --help
Usage:
  alienctl approach [flags]

Flags:
  --velocity VALUE [deprecated 2.0.0: velocity was renamed; use --speed],
  --velocity=VALUE [deprecated 2.0.0: velocity was renamed; use --speed],
  --speed VALUE, --speed=VALUE
    Set approach speed.

Global flags:
  --help  Show help.
  -h      Show help.
```

`IntroducedIn`, `DeprecatedIn`, and `RemovedIn` define lifecycle boundaries. `ReplacedBy` is a separate relationship that names where users should move. The same model applies to commands, flag spellings, arguments and forms, environment inputs, shortcuts, and finite-domain entries where those relationships are supported.

Commandment provides semantic-version and explicit ordered release schemes. Version presentation, Help, inspection, Schema, Reference, and tests can all use the same release authority instead of learning release history independently.

See [`presentation/version`](examples/presentation/version/) and [`Policy and Lifecycle`](docs/05-policy-and-lifecycle.md).

## Localize presentation

Application-authored presentation text can be translated without creating a second command model.

```console
$ ./alienctl help scan
scan nearby worlds

$ ./alienctl --locale fr help scan
analyser les mondes proches
```

The localization bundle maps application text into another locale.

```go
const scanDescription = "scan nearby worlds"

var definition = cli.Definition[rootConfig]{
	Name: "alienctl",
	Localization: cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{
			{
				Locale: "fr",
				Translations: cli.Translations{
					scanDescription: "analyser les mondes proches",
				},
			},
		},
	},
}
```

The application then declares how the user selects the effective locale.

```go
locale := root.Values.String(&root.Config.Locale)
root.Flags.ValueFor(locale).
	LocaleSelector().
	Occurrences(cli.Count.Optional()).
	Long("locale")
```

`LocaleSelector()` marks that value as the invocation's locale selector. Commandment can then resolve the effective locale before it renders Help and other Commandment-owned localized presentation.

See [`presentation/localization`](examples/presentation/localization/).

## Completion

Completion is a first-class application operation. Commandment understands incomplete input with the same commands, grammar, values, conditions, domains, and presentation metadata used everywhere else.

### Complete application values

A value can contribute its own candidates.

```go
completeWorld := func(_ context.Context, _ *visitConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	worlds := []string{"europa", "ganymede", "io"}
	var candidates []cli.CompletionCandidate
	for _, world := range worlds {
		if strings.HasPrefix(world, request.Text) {
			candidate := cli.CompletionCandidate{
				Value: world,
			}
			candidates = append(candidates, candidate)
		}
	}
	contribution := cli.CompletionContribution{
		Candidates: candidates,
	}
	return contribution, nil
}

world := visit.Arguments.String(&visit.Config.World, "world")
world.CompleteBy(completeWorld)
```

Programmatic clients call `Application.Complete` directly.

```go
request := cli.CompletionRequest{
	Args: []string{"visit", "eu"},
	Cursor: cli.CompletionCursor{
		Argument: 1,
		Byte:     2,
	},
}
completion, err := app.Complete(context.Background(), request)
```

```text
europa
```

See [`completion/values`](examples/completion/values/).

### Use native file completion

Files are often better completed by the shell or editor that already knows the local filesystem and its presentation conventions.

```go
manifest := upload.Flags.String(&upload.Config.Manifest)
manifest.CompleteWith(cli.CompletionNativeFiles).Long("manifest")
```

A completion client that advertises native file support receives a native-file request instead of a fake list generated by Commandment.

```text
native completion: files
```

The client capability decides whether that delegation is available. The semantic result still records what Commandment requested.

See [`completion/native-files`](examples/completion/native-files/).

### Use completion in real shells

The zero `ShellCompletion` configuration enables six built-in providers.

```text
Bash
Zsh
Fish
PowerShell
Elvish
Nushell
```

Every normal application gets the intrinsic `completion` command.

```sh
go build -o alienctl .
export PATH="$PWD:$PATH"
source <(alienctl completion bash)
```

At a Bash prompt, ordinary command completion uses the compiled application.

```text
alienctl sc<TAB>
```

becomes

```text
alienctl scan
```

The shell integration adapts the shell/editor protocol to `Application.Complete`. The application does not maintain a second registry of commands or values for shell completion.

Version-sensitive providers can also generate integration for an explicit target version through `ShellCompletionScriptForVersion`.

### Add another shell

Applications can replace or extend the built-in provider catalog.

```go
generateXonsh := func(generation cli.ShellCompletionGeneration) (string, error) {
	return generateXonshScript(generation.ApplicationName), nil
}

xonsh := cli.Shell.Custom("xonsh", generateXonsh)

var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	ShellCompletion: cli.ShellCompletion{
		Shells: []cli.ShellCompletionProvider{
			cli.Shell.Bash,
			xonsh,
		},
	},
}
```

The same intrinsic command can now generate the custom provider.

```console
$ ./alienctl completion xonsh
...
```

`Application.ShellCompletionScript` exposes the same provider catalog directly for hosts that want the generated artifact without running the intrinsic command.

[`Operations and Presentation`](docs/07-operations-and-presentation.md) covers completion targets, candidate groups, matching, ranking, partial results, delegation, client capabilities, and shell generation in depth.

## Inspect and describe the application

A compiled `cli.Application` is useful without executing a handler. Commandment exposes different observations for different jobs instead of forcing every tool through rendered output.

| Need | Operation |
| --- | --- |
| Explain one invocation | `Inspect` |
| Complete partial input | `Complete` |
| Read the accepted static program | `Schema` |
| Build reader-oriented static reference data | `Reference` |

### Inspect one invocation

`Inspect` runs Commandment's analysis without running middleware or the selected handler.

```go
request := cli.InspectionRequest{
	Args: []string{"scan", "--range", "5", "mars"},
}
inspection, err := app.Inspect(context.Background(), request)
```

The inspection contains the selected command, token ownership, argument form, value state, candidates, producers, domains, conditions, rules, advisories, notices, diagnostics, execution admission, signal policy, and invocation-sensitive dynamic structure when those facts are relevant.

A small tool can pull out exactly the facts it cares about.

```text
command: scan
range: 5
world: mars
```

The value history is still there after resolution. Once a tool has the destination ID from Schema, it can follow the winning candidate back to the producer that contributed it.

```go
value := inspection.Values[rangeID]
candidate := inspection.ValueCandidates[value.Contributors[0]]
producer := value.Producers[candidate.ProducerIndex]

fmt.Printf("value: %s\n", value.Representation.Text)
if producer.Kind == cli.ValueProducerKindFlag {
	fmt.Println("source: flag")
}
```

```text
value: 5
source: flag
```

The handler only needs the final `int`. Inspection retains enough structure for tooling to answer why that `int` won.

See [`observations/inspect`](examples/observations/inspect/).

### Read the accepted static program

`Schema` describes what compilation accepted without inventing invocation-time state.

```go
schema := app.Schema()
for _, commandID := range schema.CommandOrder {
	if commandID == schema.Root {
		continue
	}
	fmt.Println(schema.Commands[commandID].Name)
}
```

```text
scan
status
completion
```

Schema includes static commands and entries, flags and spellings, argument forms, values, producers, environment inputs, conditions, rules, domains, providers, lifecycle, signals, and other accepted relationships. Dynamic provider results stay dynamic rather than being misrepresented as static structure.

See [`observations/schema`](examples/observations/schema/).

### Build reference data

`Reference` projects the static application for reader-facing tooling.

```go
reference, err := app.Reference(cli.ReferenceOptions{})
if err != nil {
	return err
}
for _, commandID := range reference.CommandOrder {
	if commandID == reference.Root {
		continue
	}
	command := reference.Commands[commandID]
	fmt.Printf("%-11s%s\n", command.Name, command.Description.Short)
}
```

```text
scan       scan nearby worlds
status     show fleet status
completion Generate shell completion source
```

Use Schema when a consumer needs the accepted static semantic program. Use Reference when a consumer needs Commandment's reader-oriented static projection.

See [`observations/reference`](examples/observations/reference/).

Schema, Reference, Inspection, Completion, and Diagnostic reports are detached snapshots. Callers can retain or transform them without mutating the compiled application.

## Test Commandment applications with clitest

Commandment ships `clitest` as a first-class testing library for Commandment applications. It makes the library's public semantic observations easy to assert from ordinary Go tests.

The opening beam-strength example can verify both the final value and the producer that won.

```go
func TestBeamStrengthFlagWins(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	beamStrength := clitest.Command("abduct").Flag("--beam-strength").Value()
	request := cli.InspectionRequest{
		Env: cli.Env{
			{
				Name:  "ALIEN_BEAM_STRENGTH",
				Value: "2",
			},
		},
		Args: []string{"abduct", "--beam-strength", "4", "cow"},
	}

	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		beamStrength.Text("4"),
		beamStrength.ContributedBy(cli.ValueProducerKindFlag),
	)
}
```

An output assertion can prove that the handler saw `4`. `clitest` can also prove that the flag contribution won over the environment contribution because that fact belongs to Commandment's semantic result.

The package follows the same public operations used by applications and tooling.

| What the test needs | `clitest` entry point |
| --- | --- |
| A definition should compile | `Compile` |
| A definition should fail compilation | `CompileFails` |
| Invocation meaning before execution | `Inspect` |
| Completion for partial input | `Complete` |
| Accepted static structure | `Schema` |
| Reader-oriented reference data | `Reference` |
| Structured Commandment diagnostics | `Diagnostics` |

Locators identify subjects using recognizable coordinates.

```go
beamStrength := clitest.Command("abduct").Flag("--beam-strength").Value()

proof.Require(
	t,
	beamStrength.Text("4"),
	beamStrength.ContributedBy(cli.ValueProducerKindFlag),
)
```

Expected failures stay specific too.

```go
diagnostics := clitest.CompileFails(t, badDefinition)
diagnostics.Require(
	t,
	clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
)
```

`clitest.Conformance` can package semantic checks that genuinely need to run against several applications or reusable components. `clitest.ResponseFiles` and `clitest.SignalSource` provide deterministic fixtures for Commandment-owned host protocols.

Application behavior is still ordinary Go. Use normal `testing`, `errors.Is`, `errors.As`, subprocess tests, fuzzing, race tests, and application-specific assertions for facts that Commandment does not own.

See [`Testing with clitest`](docs/08-testing-with-clitest.md).

## Test shell integrations with shelltest

Commandment ships completion for Bash, Zsh, Fish, PowerShell, Elvish, and Nushell. The repository tests those integrations through the public `shelltest` package.

Applications can use the same package when they add a shell provider of their own and want to prove it through a real shell or editor instead of stopping at `Application.Complete`.

A `shelltest.Target[T]` binds the provider to the runtime and the setup needed for that integration.

```go
runtime := shelltest.Runtime{
	Executable:   bash,
	VersionArgs:  []string{"--version"},
	ParseVersion: parseBashVersion,
}

target := shelltest.Target[string]{
	Provider:  cli.Shell.Bash,
	Runtime:   runtime,
	Prepare:   prepareBash,
	Interpret: interpretBash,
}
```

Then drive the real editor path.

```go
input := shelltest.Input{
	BeforeCursor: "alienctl re",
}
screen := shelltest.Complete(t, app, target, input)
```

That proof includes generated integration source, the shell process, quoting, editor state, protocol transport, and the bridge back to Commandment.

Use `clitest.Complete` when the semantic completion result is the contract. Use `shelltest` when the shell integration itself is the contract.

See [`Testing Shell Completion with shelltest`](docs/09-testing-shell-completion-with-shelltest.md) and the runnable [`completion/shell`](examples/completion/shell/) scenario.

## Benchmark Commandment applications with clibench

`clibench` provides small `testing.B` helpers for Commandment's public operations.

```go
func BenchmarkInspect(b *testing.B) {
	request := cli.InspectionRequest{
		Args: []string{"abduct", "--beam-strength", "3", "cow"},
	}
	clibench.Inspect(b, app, request)
}
```

Helpers are available for `Compile`, `Run`, `Inspect`, `Complete`, `Schema`, and `Reference`. Each helper repeats exactly that successful operation inside `b.Loop()`.

Go's `testing.B` still owns benchmark structure, sub-benchmarks, profiles, timing policy, statistics, and application-specific setup. Use a normal benchmark loop when process startup, fresh state, expected failures, custom concurrency, or application work belongs inside the measurement.

See [`Benchmarking with clibench`](docs/10-benchmarking-with-clibench.md).

## Build commands from runtime data

Some command namespaces really are dynamic. An application may discover remote environments, plugins, tenants, devices, or other commands from invocation-time state.

`alienctl outpost` uses `--system` to decide which outpost command exists for the current invocation.

```go
system := outpost.Flags.String(&outpost.Config.System)
system.Long("system")
outpost.Rule.Require(system)

provideOutposts := func(_ context.Context, batch *cli.CommandBatch[outpostConfig]) error {
	switch batch.Config.System {
	case "jovian":
		europa := cli.Command[struct{}]{
			Name: "europa",
			Run: func(context.Context, *struct{}) error {
				fmt.Println("contacting europa outpost")
				return nil
			},
		}
		batch.Command(europa)
	case "saturnian":
		titan := cli.Command[struct{}]{
			Name: "titan",
			Run: func(context.Context, *struct{}) error {
				fmt.Println("contacting titan outpost")
				return nil
			},
		}
		batch.Command(titan)
	}
	return nil
}

outpost.Subcommands.CommandsFrom(provideOutposts, system)
```

```console
$ ./alienctl outpost --system jovian europa
contacting europa outpost

$ ./alienctl outpost --system saturnian titan
contacting titan outpost
```

The runtime namespace is also visible to Help.

```console
$ ./alienctl outpost --system jovian --help
Usage:
  alienctl outpost [flags] <command>

Commands:
  europa

Flags:
  --system VALUE, --system=VALUE

Global flags:
  --help  Show help.
  -h      Show help.
```

Passing `system` to `CommandsFrom` declares the prerequisite. Commandment resolves it before invoking the provider. `CommandBatch` publishes the result atomically, so an invalid or failed provider does not leak half a namespace.

Once accepted, provider-produced commands are normal commands. Selection, Help, completion, inspection, aliases, shortcuts, defaults, localization, lifecycle, and policy do not need parallel dynamic implementations.

See [`commands/dynamic`](examples/commands/dynamic/).

## Supply values from the host

Not every value comes from command-line text. Commandment keeps host-provided fallback input and trusted host context separate because they mean different things.

### Invocation defaults

An invocation default is fallback input. A stronger ordinary producer can replace it.

```go
func newStatusProgram() (cli.Definition[struct{}], cli.DefaultProvider) {
	var ship cli.Destination[statusConfig, string]
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			status := cli.Command[statusConfig]{
				Name: "status",
				Declare: func(status *cli.CommandBuilder[statusConfig]) {
					ship = status.Values.String(&status.Config.Ship)
					status.Flags.ValueFor(ship).Long("ship")
				},
				Run: runStatus,
			}
			root.Subcommands.Command(status)
		},
	}

	supplyDefault := func(_ context.Context, supply *cli.DefaultSupply[statusConfig]) error {
		supply.Set(ship, "scout-7")
		return nil
	}
	defaults := cli.DefaultsFor[statusConfig](supplyDefault)
	return definition, defaults
}
```

The host supplies the provider with the run request.

```go
request := cli.RunRequest{
	Args:     os.Args[1:],
	Defaults: []cli.DefaultProvider{defaults},
}
err := app.Run(context.Background(), request)
```

```console
$ ./alienctl status
scout-7 status: nominal

$ ./alienctl status --ship voyager
voyager status: nominal
```

The flag overrides the host fallback because both contribute to the same destination.

See [`host-context/defaults`](examples/host-context/defaults/).

### Facts

A fact is trusted typed context supplied by the host outside ordinary CLI input.

```go
func newContactProgram() (cli.Definition[struct{}], cli.FactProvider) {
	var ship cli.Destination[contactConfig, string]
	definition := cli.Definition[struct{}]{
		Name: "alienctl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			contact := cli.Command[contactConfig]{
				Name: "contact",
				Declare: func(contact *cli.CommandBuilder[contactConfig]) {
					ship = contact.Facts.Require(&contact.Config.Ship)
					contact.Arguments.String(&contact.Config.Target, "target")
				},
				Run: runContact,
			}
			root.Subcommands.Command(contact)
		},
	}

	supplyFacts := func(_ context.Context, supply *cli.FactSupply[contactConfig]) error {
		supply.Set(ship, "mothership")
		return nil
	}
	facts := cli.FactsFor[contactConfig](supplyFacts)
	return definition, facts
}
```

```go
request := cli.RunRequest{
	Args:  os.Args[1:],
	Facts: []cli.FactProvider{facts},
}
```

```console
$ ./alienctl contact martian
mothership contacting martian
```

The user supplied `martian`. The host supplied the current ship. Facts can participate in dependencies, sensitivity, inspection, and testing without pretending the user typed them.

See [`host-context/facts`](examples/host-context/facts/).

## Keep execution ordinary Go

Once an invocation is admitted, handlers and middleware remain normal Go. Commandment gives execution explicit access to the values and resources it needs without turning the framework into a general dependency-injection container.

### Access standard streams and resolved values

A handler can receive stdout through its configuration.

```go
type scanConfig struct {
	World  string
	Output io.Writer
}

scan := cli.Command[scanConfig]{
	Name: "scan",
	Declare: func(scan *cli.CommandBuilder[scanConfig]) {
		scan.Arguments.String(&scan.Config.World, "world")
		scan.Access.Stdout(&scan.Config.Output)
	},
	Run: func(_ context.Context, config *scanConfig) error {
		_, err := fmt.Fprintf(config.Output, "scanning %s\n", config.World)
		return err
	},
}
```

Execution access can also move an ancestor-owned resolved value into the selected command configuration without resolving it again.

```go
ship := root.Values.String(&root.Config.Ship)
root.Values.Default(ship, "mothership")
root.Flags.ValueFor(ship).Long("ship")

scan.Access.Value(ship, &scan.Config.Ship)
```

Command selection, value state, effective locale, working directory, stdin, stdout, and stderr use the same explicit access model where applicable.

### Middleware

Middleware wraps admitted execution.

```go
func scanMiddleware(ctx context.Context, config *scanConfig, next cli.MiddlewareNext) error {
	if _, err := fmt.Fprintln(config.Output, "preparing sensors"); err != nil {
		return err
	}
	if err := next(ctx); err != nil {
		return err
	}
	_, err := fmt.Fprintln(config.Output, "recording scan")
	return err
}

scan := cli.Command[scanConfig]{
	Name:       "scan",
	Middleware: []cli.Middleware[scanConfig]{scanMiddleware},
	Run:        runScan,
}
```

```console
$ ./alienctl scan mars
preparing sensors
scanning mars
recording scan
```

Parsing, resolution, availability, rules, and diagnostics happen before middleware. Middleware is for execution concerns, not for rebuilding the CLI model around the handler.

See [`execution/middleware`](examples/execution/middleware/).

### Application errors and exit codes

Application errors keep normal Go identity.

```go
var errLandingDenied = errors.New("landing denied: earth is protected")

func runLand(_ context.Context, config *landConfig) error {
	if config.World == "earth" {
		return cli.WithExitCode(errLandingDenied, 7)
	}
	fmt.Printf("landing on %s\n", config.World)
	return nil
}
```

```console
$ ./alienctl land earth
landing denied: earth is protected
```

The process exits with status `7`. `errors.Is` and `errors.As` still work on the original application error. Exit status is process policy, not a reason to turn application failures into Commandment diagnostics.

See [`execution/errors`](examples/execution/errors/).

### Run from the process or supply the host explicitly

`RunFromProcess` is the convenient adapter for a normal executable.

```go
err := app.RunFromProcess(context.Background())
```

Use `Run` when a host needs to provide the invocation explicitly.

```go
request := cli.RunRequest{
	Args:          args,
	Env:           env,
	Defaults:      defaults,
	Facts:         facts,
	Notify:        notify,
	ResponseFiles: responseFiles,
}
err := app.Run(ctx, request)
```

That boundary supports embedded applications, alternate process adapters, daemon hosts, and explicit capability ownership without making the compiled application a global singleton.

See [`Host Integration and Execution`](docs/06-host-integration-and-execution.md).

## Response files

Response files let users move arguments into another input source while preserving ordinary Commandment grammar and provenance.

Enable response-file recognition in the command dialect.

```go
root.Dialect.ResponseFiles(true)
root.Arguments.String(&root.Config.Name, "name")
```

If `crew.args` contains

```text
Zorblax
```

then the normal process host can load it from the filesystem.

```console
$ ./alienctl hail @crew.args
hail Zorblax
```

See [`response-files/filesystem`](examples/response-files/filesystem/).

An embedded host can provide another loader instead of granting filesystem access.

```go
loadCrew := func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
	if request.Path != "crew" {
		return cli.ResponseFile{}, errUnknownResponseSource
	}
	response := cli.ResponseFile{
		Key:      "crew",
		Location: "crew",
		Content:  []byte("Zorblax"),
	}
	return response, nil
}

request := cli.RunRequest{
	Args: os.Args[1:],
	ResponseFiles: cli.ResponseFileAccess{
		Load: loadCrew,
	},
}
```

```console
$ ./alienctl hail @crew
hail Zorblax
```

The included text re-enters normal grammar with response-file source history. It does not become an opaque application string.

See [`response-files/custom-loader`](examples/response-files/custom-loader/).

## Signals

Commandment brings signals into the application model instead of leaving every command to wire its own `os.Signal` loop.

A signal can come from the operating system, but the abstraction is broader than operating-system signals. Commands can match signal sources, choose the phases where they matter, select or aggregate occurrences, and declare what should happen.

### Handle Ctrl-C inside the command

A command can route an interrupt into a typed endpoint that application code receives through its configuration.

```go
type listenConfig struct {
	Events <-chan cli.SignalEvent
}

listen := cli.Command[listenConfig]{
	Name: "listen",
	Declare: func(listen *cli.CommandBuilder[listenConfig]) {
		events := listen.Signals.Endpoint(&listen.Config.Events)
		interrupt := listen.Signals.Match.
			Interrupt().
			During(cli.SignalPhaseExecution).
			Each().
			Hint("interrupt")

		listen.Signals.Policy(
			interrupt,
			listen.Signals.Action.Emit(events),
		)
	},
	Run: func(ctx context.Context, config *listenConfig) error {
		fmt.Println("listening for signals")
		select {
		case <-config.Events:
			fmt.Println("signal received")
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	},
}
```

```console
$ ./alienctl listen
listening for signals
```

Press Ctrl-C.

```text
signal received
```

The process interrupt enters Commandment's signal model, matches the declared execution-phase policy, becomes a structured `SignalEvent`, and reaches the handler through the endpoint. The command can shut down cleanly without owning the process signal loop itself.

See [`signals/interrupt`](examples/signals/interrupt/).

### Signals are not limited to the operating system

Applications can register their own typed signal sources.

```go
descriptor := cli.SignalDescriptor[string]{
	Signal:   "alarm",
	Fidelity: cli.SignalSourceDeliveryFidelityExact,
}

source := cli.SignalSource[string]{
	Signals: []cli.SignalDescriptor[string]{descriptor},
	Open:    openAlarm,
}

registered := root.Signals.Source(source)
alarm := registered.Signal("alarm")
```

A command can route the custom signal through the same policy and endpoint model.

```go
events := watch.Signals.Endpoint(&watch.Config.Events)
match := watch.Signals.Match.
	On(alarm).
	During(cli.SignalPhaseExecution).
	Each().
	Hint("alarm")

watch.Signals.Policy(match, watch.Signals.Action.Emit(events))
```

```console
$ ./alienctl watch
received alarm
```

The source could represent a runtime event, supervisory control channel, host notification, or another application-defined control plane. Application code can use the resulting events to coordinate reload, shutdown, restart, or any other control flow it owns.

See [`signals/custom-source`](examples/signals/custom-source/).

### Choose what each signal does

Signal policies are not limited to endpoint emission. Action plans can ignore a match, cancel primary execution, call an application handler, forward or send a signal to a declared target, emit a structured event, abort with an exit code, or reraise to the host.

Actions can be sequenced.

```go
plan := command.Signals.Action.
	Emit(events).
	ThenCancel()
```

A sequence can also declare delayed escalation.

```go
plan := command.Signals.Action.
	Handle(beginGracefulShutdown).
	After(5 * time.Second).
	Abort(1)
```

That makes signal behavior part of the command's semantics instead of a collection of unrelated goroutines and process hooks. Inspection, Schema, diagnostics, and `clitest` can observe the same declared policy.

[`Host Integration and Execution`](docs/06-host-integration-and-execution.md) covers source identity, sender metadata, delivery fidelity, phases, selection, aggregation, action sequencing, endpoints, forwarding, cancellation, reraising, and failure behavior.

## Scale the source

A larger Commandment application can use ordinary Go package ownership. It does not need a second internal framework around Commandment.

A repository might grow into a shape like this.

```text
cmd/alienctl/
    main.go

internal/fleet/
    command.go
    run.go
    run_test.go

internal/worlds/
    command.go
    provider.go

internal/host/
    facts.go
    services.go

app.go
app_test.go
```

The exact file names are not important. The useful properties are.

- Process adaptation stays thin.
- Application assembly is easy to find.
- Commands own the configuration and declarations their handlers need.
- One-use declaration callbacks remain close to the declaration they describe.
- Helpers are extracted for real reuse or a real operation, not merely to hide source length.
- Command providers are used for genuinely dynamic namespaces rather than source organization.
- Host services remain ordinary Go and enter through explicit host boundaries when Commandment needs them.
- Middleware owns cross-cutting execution only when the concern really belongs to execution.
- Tooling consumes the compiled application instead of maintaining another registry.
- Application behavior uses ordinary Go tests, Commandment semantics use `clitest`, and real shell integrations use `shelltest`.

[`Building Large Applications`](docs/12-building-large-applications.md) covers package ownership, extraction, static and dynamic structure, host integration, and testing at application scale.

[`Adopting Commandment`](docs/13-adopting-commandment.md) applies the same model to existing CLIs where grammar, configuration behavior, Help, completion, errors, and other observable contracts already matter.

## When Commandment fits

If the whole job is a small command tree with a few flags, a smaller CLI library may be the better choice.

Commandment becomes more useful as the CLI needs more of these capabilities to agree with one another.

```text
commands and grammar
        |
        v
typed value resolution
        |
        v
conditions, rules, lifecycle
        |
        v
execution admission

same compiled application
        |
        +-> Help and localization
        +-> completion and shell integration
        +-> diagnostics and repairs
        +-> Inspect
        +-> Schema and Reference
        +-> clitest
        `-> host integration and signals
```

That includes CLIs with multi-source typed configuration, nontrivial precedence, conditional grammar, finite or runtime-dependent domains, dynamic commands, lifecycle policy, structured diagnostics, rich Help, custom completion, host-supplied context, response files, signals, or tooling that needs structured information instead of rendered text.

Commandment stops at the CLI boundary. Persistence, network clients, databases, service topology, domain entities, and general dependency injection remain application concerns unless they participate through an explicit CLI or host relationship.

## Start building

Commandment is currently RC2 before `v1.0.0`. The module requires Go 1.27.1. This source snapshot has no release tag, so the public guides use an exact local checkout rather than pretending a stable module version already exists.

Start with [`Getting Started`](docs/01-getting-started.md) for the complete cold-start path.

Continue with the guide that matches the problem you are solving.

- [`Commandment Guide`](docs/02-commandment-guide.md) explains the mental model and declaration lifetime.
- [`Commands and Grammar`](docs/03-commands-and-grammar.md) covers commands, entries, positional forms, profiles, dialect, token ownership, and providers.
- [`Values and Resolution`](docs/04-values-and-resolution.md) covers producers, tiers, languages, domains, dependencies, provenance, sensitivity, and repair.
- [`Policy and Lifecycle`](docs/05-policy-and-lifecycle.md) covers conditions, availability, visibility, rules, advisories, notices, lifecycle, and replacement.
- [`Host Integration and Execution`](docs/06-host-integration-and-execution.md) covers `Run`, host context, access, middleware, response files, signals, and application errors.
- [`Operations and Presentation`](docs/07-operations-and-presentation.md) covers Inspect, Complete, Schema, Reference, Diagnostics, Help, localization, version, and shell integration.
- [`Testing with clitest`](docs/08-testing-with-clitest.md) covers semantic testing of Commandment applications.
- [`Testing Shell Completion with shelltest`](docs/09-testing-shell-completion-with-shelltest.md) covers real shell and editor integration tests.
- [`Benchmarking with clibench`](docs/10-benchmarking-with-clibench.md) covers the benchmark helpers.
- [`Troubleshooting`](docs/11-troubleshooting.md) starts from structured observations when something behaves unexpectedly.
- [`Building Large Applications`](docs/12-building-large-applications.md) covers application-scale composition.
- [`Adopting Commandment`](docs/13-adopting-commandment.md) covers migration of an existing CLI.
- [`examples/`](examples/README.md) contains focused runnable `alienctl` programs.
- GoDoc owns exact API contracts.

If an API or behavior should change before `v1.0.0`, [open an issue](https://github.com/abstractionsinc/commandment/issues).
