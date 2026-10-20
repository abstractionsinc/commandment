# Commands and Grammar

Use this guide for the language a user types, including command selection, positional forms, flags and spellings, grammar profiles, dialect, and invocation-time command namespaces. Grammar belongs in the declaration, not in a handler-side parser.

## Commands and command entries are not the same thing

A command is a semantic CLI scope. A command entry is one immediate routing member in a command namespace.

A canonical command can be selected through several entry relationships. Its canonical entry names the command directly, a lexical alias gives that immediate command another spelling, and a command shortcut targets a canonical path while optionally contributing bounded input or fallback defaults. A default relationship is different again because it selects one immediate child when no explicit child entry wins without consuming command input.

```go
scanCmd := cli.Command[scanConfig]{
	Name:    "scan",
	Aliases: cli.Aliases("survey"),
}
scan := cb.Subcommands.Command(scanCmd).Default()

probeCmd := cli.Command[probeConfig]{
	Name: "probe",
	Declare: func(cb *cli.CommandBuilder[probeConfig]) {
		shortcut := cli.CommandShortcut{
			Name: "quick-probe",
			Args: []string{"--mode", "quick"},
		}
		cb.Shortcut(shortcut)
	},
}
probe := cb.Subcommands.Command(probeCmd)
scan.ReplacedBy(probe)
```

Use an alias when only recognition spelling changes. Use a shortcut when the entry represents a useful invocation shape with fixed input or fallback defaults. Use a default relationship when omission should select a child. Declaration replacement records lifecycle and does not silently redirect execution.

The [commands collection](../examples/commands/) shows these relationships in complete CLIs.

## Keep child configuration with the child

A nested command receives its own `cli.CommandBuilder[C]` and configuration instance.

```go
type probeConfig struct {
	Mode string
}

type statusConfig struct{}

cmd := cli.Command[probeConfig]{
	Name: "probe",
	Declare: func(cb *cli.CommandBuilder[probeConfig]) {
		statusCmd := cli.Command[statusConfig]{Name: "status"}
		cb.Subcommands.Command(statusCmd)
	},
}
```

Do not turn the root configuration into a union of every descendant field merely because the command tree is nested. Keep command and configuration ownership aligned unless a real cross-command relationship requires otherwise.

## Flags own destinations and spellings own syntax

A flag and a flag spelling are separate semantic subjects.

```go
mode := cb.Flags.String(&cb.Config.Mode)
mode.Long("mode")
mode.Short('m')
```

`mode` is one flag and one destination. `--mode` and `-m` are two recognized spellings, not two application values. The distinction remains visible to lifecycle, diagnostics, help, completion, and inspection.

Commandment provides scalar flag constructors and typed forms for existing destinations, sequences, tokens, and keyed values. Occurrence counts, values per occurrence, delimiters, terminators, placement, requiredness, visibility, sensitivity, lifecycle, and spelling-specific forms belong to the flag or spelling that owns them. Use those refinements only for real CLI language requirements.

## Positional arguments can have explicit forms

Simple commands can declare positional terms directly.

```go
cb.Arguments.String(&cb.Config.Target, "target")
```

When a command has genuinely different positional shapes, declare argument forms instead of branching on raw tokens in the handler.

```go
cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[transmitConfig]) {
	form.Literal("to")
	form.String(&form.Config.Target, "target")
	form.Strings(&form.Config.Words, "message", cli.Count.OneOrMore())
})

raw := cb.Values.Sequence(&cb.Config.Raw, cli.Value.String())
cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[transmitConfig]) {
	form.Literal("raw")
	form.Remainder(raw, "payload", cli.Count.OneOrMore())
})
```

Argument terms can own scalar values, bounded sequences, token-language values, literals, remainders, and pass-through input. Optionality and allocation belong to the grammar so the handler receives the typed result rather than reconstructing which form matched.

The [argument forms](../examples/arguments/forms/) and [pass-through](../examples/arguments/pass-through/) scenarios show the complete runtime shapes.

### Remainder versus pass-through

A remainder is still Commandment-owned positional grammar. Commandment owns the tokens and contributes them to the destination.

Pass-through deliberately preserves a bounded token sequence for another application-owned consumer. It creates an ownership boundary for one term rather than disabling the rest of Commandment grammar. Use GoDoc for the exact occurrence and allocation contracts.

## Grammar profiles represent mutually exclusive alternatives

A grammar profile is a statically declared partition selected under bounded deterministic rules. Use one when a single command legitimately has alternate grammar selected by an early value.

```go
cb.Profiles.Switch(profile, func(cases *cli.GrammarProfileCases[transmitConfig, string]) {
	cases.Case("brief", func(*cli.CommandBuilder[transmitConfig]) {})
	cases.Case("verbose", func(cb *cli.CommandBuilder[transmitConfig]) {
		cb.Flags.Bool(&cb.Config.Verbose).Occurrences(cli.Count.Optional()).Long("verbose")
	})
})
```

A profile is part of declared grammar, so analysis, completion, artifacts, and tests can observe it. Use separate commands instead when the alternatives have different semantic ownership or application behavior.

## Dialect changes recognition policy

A dialect is a declared bundle of lexical and grammar policy. `cb.Dialect` controls decisions such as flag ordering, short-flag clustering, command and flag matching/normalization, dashless short flags, end-of-options tokens, response-file recognition, and default flag placement.

Those settings change the CLI language rather than its presentation. Keep the default dialect unless compatibility or product requirements justify a change, especially when the change affects a large command surface.

## Token ownership is semantic

Analysis records which grammar element owns each input token. Diagnostics, completion, repairs, and selection reasoning can use that ownership later.

Do not parse the same raw argument strings again in a handler. Once declared grammar owns the tokens, reparsing creates a second interpretation that other Commandment operations cannot observe. The declared end-of-options boundary follows the same rule.

## Value language and token language solve different problems

A value language converts one scalar value between text and its typed representation. Built-ins cover common scalar types, and `ValueCodec[T]` defines an application-specific scalar language.

A token language owns the bounded token sequence assigned to one grammar term. Use `TokenLanguage[T]` only when one semantic value genuinely has structured multi-token syntax. Neither abstraction should become a hidden command parser; routing, option recognition, form selection, and token ownership remain command grammar responsibilities.

[Values and Resolution](04-values-and-resolution.md) covers value languages inside the value graph.

## Command providers materialize a namespace when demanded

Keep command trees static unless the set of commands genuinely depends on invocation-time state. `Subcommands.CommandsFrom` declares a command provider for that case.

```go
cb.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[struct{}]) error {
	cmd := cli.Command[remoteConfig]{
		Name: "remote",
		Run:  runRemote,
	}
	batch.Command(cmd)
	return nil
})
```

The callback publishes through a `CommandBatch`. Acceptance is atomic, so invalid provider output rejects the batch instead of exposing half a namespace. Accepted provider-local aliases, shortcuts, localization, and commands use the ordinary Commandment semantics for those concepts.

Because materialization is demand-driven and invocation-scoped, detached observations preserve command-namespace completeness so callers can distinguish an unobserved namespace from a complete namespace that lacks an entry.

Use a provider because the namespace is genuinely dynamic, not to organize source files. See the [dynamic commands scenario](../examples/commands/dynamic/).

## Response-file recognition is grammar and loading is host integration

`cb.Dialect.ResponseFiles(true)` allows response-file syntax. It does not grant filesystem access.

The operation host supplies `cli.ResponseFileAccess`, while `RunFromProcess` supplies the ordinary process filesystem adapter. Loaded content re-enters normal grammar and retains response-file input-source and origin relationships.

[Host Integration and Execution](06-host-integration-and-execution.md) covers loading, and the [response-files collection](../examples/response-files/) shows complete examples.

## Observe grammar through the right operation

Use `Schema` for accepted static command and grammar structure, `Inspect` for what won in one invocation, and `Complete` for incomplete input. The operations project one semantic model but answer different questions.

GoDoc owns the exact method families and zero/lifetime behavior for commands, builders, arguments, flags, dialect, grammar profiles, and command batches.
