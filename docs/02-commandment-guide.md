# Commandment Guide

[Getting Started](01-getting-started.md) leaves you with a working command, typed multi-source resolution, one rule, and one semantic test. This guide explains the ownership model that keeps those pieces coherent as the CLI grows.

```text
authoring
    definition
      -> declarations through callback-scoped capabilities
      -> commit points
      -> Compile

compiled state
    immutable application / static program

one operation
    explicit request + host capabilities
      -> analysis
      -> invocation-time materialization when needed
      -> effective invocation model
      -> detached observation and/or execution
```

GoDoc remains the owner of exact signatures, zero/nil behavior, and option-level contracts.

## Definition, declaration, and application are different things

A definition is the root authored input supplied to `cli.Compile`. A declaration is semantic program content admitted through a declaration callback, such as a command, destination, flag, argument term, rule, signal policy, or provider. The application is the immutable compiled program returned by `cli.Compile`.

`Run`, `Inspect`, `Complete`, `Schema`, and `Reference` operate on the application. They do not reopen the definition as a mutable runtime registry.

## Declaration capabilities have a lifetime

`cli.CommandBuilder` and the capabilities reachable from it belong to one declaration callback and scope. They have an invalid zero value, may be passed only through synchronous work inside the active scope, and may not be retained, used concurrently, or reused in another declaration scope.

Canonical Commandment source keeps one-use declaration callbacks lexical because the nesting shows semantic ownership.

```go
var definition = cli.Definition[struct{}]{
	Name: "alienctl",
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cmd := cli.Command[scanConfig]{
			Name: "scan",
			Declare: func(cb *cli.CommandBuilder[scanConfig]) {
				// scan owns these declarations
			},
		}
		cb.Subcommands.Command(cmd)
	},
}
```

Extract a helper when it represents a reusable declaration fragment or real behavior such as a provider, completion algorithm, handler, or host integration. Do not extract `declareRoot`, `declareScan`, and similar one-use callbacks merely to shorten the literal. [Building Large Applications](12-building-large-applications.md) applies this rule across packages.

## Inert expressions and commit points keep authoring declarative

Some authoring values describe relationships before they become part of the semantic program. For example, `cb.When.Equal(...)` creates an inert condition; applying that condition through `AvailableWhen`, `VisibleWhen`, an advisory, or another supported owner gives it semantic effect.

A commit point is the declaration operation that accepts staged authoring into the program. The exact commit point differs by family. In practice, compose declarations inside their callback, use returned declaration values only for supported relationships, and let `Compile` produce the durable application.

## The static program is immutable

Successful compilation accepts the static declarations and produces one immutable application. Repeated and concurrent operations own independent operation state.

Some structure is intentionally invocation-scoped, so a command provider can publish a command batch when its namespace is demanded, a dynamic value domain can materialize legal values from resolved dependencies, and the host can supply invocation defaults or facts. Those contributions belong to that operation and do not mutate the static application.

Static and dynamic declarations still use the same semantic model. A command-provider-produced command is an ordinary command after its batch is accepted; only its origin and lifetime differ.

## Commands own scopes and configuration follows ownership

Give each command the configuration type its handler actually needs.

```go
type rootConfig struct {
	Locale string
}

type deployConfig struct {
	Region string
	DryRun bool
}
```

The root does not need every descendant field. A child command owns its configuration, grammar, policy, middleware, execution access, signals, and child declarations unless a real cross-command relationship says otherwise.

## Grammar describes the language users type

Command routing, positional forms, flags, flag spellings, command shortcuts, grammar profiles, dialect, token ownership, and end-of-options behavior belong to grammar. Handlers should not rediscover them from raw strings.

A command and a command entry are distinct. A canonical command has semantic identity, while a namespace can contain different entries that select it. A lexical alias is another immediate spelling. A command shortcut can target a canonical path and contribute bounded input. A default relationship selects an immediate child without consuming input. An action entry selects a bounded frontend action such as help or version.

A flag and a flag spelling are separate for the same reason. One flag can recognize `--mode` and `-m` without creating two application values.

[Commands and Grammar](03-commands-and-grammar.md) covers the authoring shapes and exact distinctions.

## Values form a typed graph

A destination is a typed semantic node. Producers create candidates for it, and resolution tiers plus the destination's resolution policy determine the effective value.

Commandment keeps several relationships distinct because tools and tests often need the difference.

- An input source is where invocation material was read from.
- A producer is the semantic mechanism that can create a candidate.
- A producer kind classifies that mechanism on the candidate.
- A resolution tier is a precedence stratum.
- A value contribution is an accepted contribution to the destination.
- Provenance is the causal history of a contribution or effective value.

For example, process environment is an input source. An environment-input declaration is the producer, its candidate records the environment producer kind, and that candidate participates at a resolution tier.

Dependencies make value relationships explicit because derivations, dynamic domains, and fact resolvers can declare the destinations they require, which lets static analysis and invocation resolution reason about scope, ordering, and cycles. [Values and Resolution](04-values-and-resolution.md) owns that model.

## CLI input, host context, and policy stay separate

Facts are typed host-supplied invocation inputs outside ordinary CLI grammar. Invocation defaults are host-supplied fallback contributions for destinations. Execution access exposes selected resolved values or supported host resources such as standard streams and working directory to execution configuration.

Policy keeps different responsibilities separate through distinct owners. Conditions describe activation, availability controls structural participation, visibility controls presentation, rules are blocking validation, advisories are nonfatal semantic outcomes, and notice suppression controls delivery without erasing the underlying fact. Lifecycle and declaration replacement describe release evolution.

Use [Policy and Lifecycle](05-policy-and-lifecycle.md) and [Host Integration and Execution](06-host-integration-and-execution.md) when those concerns become concrete.

## An invocation produces an effective model

`Run`, `Inspect`, and `Complete` take explicit operation input. Analysis selects commands, determines token ownership, materializes demanded dynamic structure, resolves destinations, evaluates policy, selects an invocation action, and produces the operation-specific result.

The effective invocation model is operation-owned semantic state. It is neither the immutable static program nor application business state.

Before execution admission, Commandment owns semantic preparation. After admission, same-scope middleware and the selected handler receive the invocation-local configuration under ordinary Go ownership. Later application mutation does not rewrite finalized resolution, provenance, policy outcomes, or detached observations.

## Choose the operation by the question

| Question | Operation |
| --- | --- |
| Execute an accepted invocation | `Run` / `RunFromProcess` |
| Explain one invocation without executing application code | `Inspect` |
| Reason about incomplete input | `Complete` |
| Read accepted static semantic structure | `Schema` |
| Build static reader-oriented reference data | `Reference` |

Do not stretch one operation into another's job. `Schema` does not answer invocation-time provider or resolution questions, `Inspect` is unnecessary for purely static declarations, and completion clients should not scrape help text. Detached artifacts are caller-owned snapshots; mutating them does not mutate the application.

[Operations and Presentation](07-operations-and-presentation.md) covers these operations and the presentation surfaces built from them.

## Partial results are evidence, not guesses

Some operations can return the most complete valid partial result together with failure or truncation state. Presence in a partial result can still be proved, but absence usually cannot.

If a partial completion contains `europa`, that proves presence. If `ganymede` is missing from a truncated result, it does not prove absence. The same rule applies to incomplete provider namespaces, inspection data, diagnostics, and exact collection claims. `clitest` preserves the difference between disproved and not provable.

## Presentation derives from semantics

Help, localization, version presentation, completion display metadata, diagnostics, notices, Schema, Reference, and Inspection project the semantic model rather than redefine it.

Application-authored text has localization ownership and an effective locale. A custom help renderer receives a structured `cli.HelpPage` instead of rediscovering command selection or translations. `Definition.Release` remains the release authority used by lifecycle and version presentation.

Completion follows the same ownership rule. Custom completion belongs to the value or grammar declaration that owns the target, while native-client requests and completion delegation preserve external ownership boundaries. Shell completion integration adapts a shell/editor protocol to the semantic completion operation; it does not maintain another command model.

## Keep application responsibilities outside Commandment

Commandment owns the CLI semantic program. Persistence, network clients, service topology, domain entities, general dependency injection, transaction boundaries, and application-specific orchestration remain ordinary Go unless they enter through an explicit Commandment host contract.

Application errors remain ordinary Go errors. Commandment diagnostics describe Commandment-owned failures, while `errors.Is` and `errors.As` remain the normal tools for application-owned error identity. Exit status is process policy and does not turn an application error into a Commandment diagnostic.

## Continue by responsibility

Use the [documentation index](README.md) to choose the focused guide for the next problem. The [examples corpus](../examples/README.md) provides complete programs after you understand the concept, and GoDoc owns exact local contracts throughout.
