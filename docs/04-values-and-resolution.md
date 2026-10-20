# Values and Resolution

Use this guide when typed configuration can come from several places, depend on other values, use a finite domain, or retain evidence about how the effective value was produced.

```text
producers -> candidates -> resolution policy -> value / resolution state
                                              |-> provenance
                                              |-> policy
                                              `-> dependencies / derivations
```

The central unit is a destination. A flag or environment input can produce a candidate for a destination, but neither is the destination itself.

## Name the application value once

The `BeamStrength` example declares one destination and attaches three producers.

```go
beamStrength := cb.Values.Int(&cb.Config.BeamStrength)
cb.Values.Default(beamStrength, 1)
cb.Values.Environment(beamStrength, "ALIEN_BEAM_STRENGTH")
cb.Flags.ValueFor(beamStrength).Long("beam-strength")
```

The handler still sees one `int`. Commandment keeps the candidate and provenance information needed to explain how that value settled.

`cb.Values` can declare scalar destinations, sequences, keyed maps, and destinations with application-supplied snapshot behavior. A flag or positional argument can create the destination directly when only one producer owns it and no later relationship needs a name for the destination.

## Producers create candidates

A producer is a semantic mechanism that can create a value candidate. Normal producer families include positional arguments, flags, environment input, static and invocation defaults, facts, derivations, command-shortcut fallback defaults, and equivalent producers inside accepted command-provider output.

Producer kind is the classifier recorded on the candidate, while input source identifies where invocation material was read. Process environment is an input source; an environment-input declaration is the producer; its candidate records `ValueProducerKindEnvironment` and participates at a resolution tier.

Keep these relationships distinct when inspection, diagnostics, repair, or tests need to explain the result.

## Resolution tiers express precedence

A resolution tier is a precedence stratum. The destination's resolution policy determines how accepted contributions settle within and across tiers.

Built-in policies from `cli.Value` include `KeepFirst()`, `KeepLast()`, `RejectMultipleTiers()`, and `CombineTiers(order)`.

```go
beamStrength := cb.Values.Int(&cb.Config.BeamStrength)
cb.Values.ResolveWith(beamStrength, cli.Value.KeepLast())
```

`ResolutionTierOrder` matters when a combine policy needs an explicit direction. Keep precedence in the destination model rather than reimplementing it in a handler.

[Getting Started](01-getting-started.md) demonstrates the ordinary default/environment/flag precedence and proves the winning producer with `clitest.Inspect`.

## Resolution state is not a Go zero value

A destination can be inactive, absent, resolved, failed, blocked, or otherwise not yet established according to the public resolution-state model. That state is separate from the Go zero value stored in the configuration field.

Execution normally reads the final typed configuration after execution admission. Tools and tests that need to know whether a destination resolved, why it failed, or which candidates were observed should use `Inspect` or another public observation that exposes resolution state. Do not infer semantic absence from `0`, `""`, or another Go zero value.

## Value languages own typed scalar meaning

A value language defines the typed scalar language used by value-producing declarations. Built-ins from `cli.Value` cover common scalar types, and a static finite domain is also a value language.

```go
baseDomain := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
	values.Value("sol", "sol")
	values.Value("vega", "vega")
})
base := cb.Values.Value(&cb.Config.Base, baseDomain)
```

Use `ValueCodec[T]` for application-specific scalar syntax. A token language solves a different problem by owning the bounded token sequence assigned to one grammar term, while a keyed value language owns typed key/value parsing for map-shaped destinations.

Keep command routing and token ownership outside scalar codecs. These language abstractions let parsing, formatting, domains, completion, diagnostics, and artifacts share one typed meaning.

## Static and dynamic value domains

A value domain is a finite semantic set attached to a value language. Use a static domain when legal values are known at compile time and an invocation-scoped dynamic domain when legal values depend on already-resolved invocation state.

```go
regions := cb.Values.Domain(
	func(_ context.Context, _ *deployConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
		base, _ := dependencies.Get(baseDestination)
		domain.Value(base+"-north", base+"-north")
		domain.Value(base+"-south", base+"-south")
		return nil
	},
	baseDestination,
)
```

Explicit dependencies let Commandment order evaluation, retain outcome state, expose the materialized domain in inspection, and use the same domain for completion. Do not fetch and validate a finite CLI domain again in the handler.

See [value choices](../examples/values/choices/) and [dynamic choices](../examples/values/dynamic-choices/).

## Dependencies make derived meaning explicit

A dependency says that one semantic evaluator requires another destination or value.

```go
summary := cli.Value.ComputedBy(
	func(_ context.Context, _ *abductConfig, dependencies cli.Dependencies) (string, error) {
		target, _ := dependencies.Get(targetDestination)
		beamStrength, _ := dependencies.Get(beamStrengthDestination)
		return fmt.Sprintf("abducting %s with beam strength %d", target, beamStrength), nil
	},
).From(targetDestination, beamStrengthDestination)

cb.Values.Derive(&cb.Config.Summary, summary)
```

`.From(...)` is part of the semantic graph, not commentary for the callback. Static analysis and invocation resolution can use it to validate scope, phase, and reachability. Prefer explicit dependencies over hidden reads from unrelated mutable configuration.

## Facts are host context, not CLI input

A fact is typed invocation context supplied outside ordinary CLI grammar.

```go
clearance := cb.Facts.Require(&cb.Config.Clearance).Sensitive()
```

The host supplies the destination through a `cli.FactProvider`, commonly built with `cli.FactsFor`. Facts participate in sensitivity, dependencies, resolution, and observations without pretending the user typed them.

A resolved fact can depend on other destinations.

```go
greeting := cb.Facts.Resolve(&cb.Config.Greeting, resolveGreeting)
greeting.From(clearance, species)
```

Invocation defaults are a separate host input. They provide fallback candidates for ordinary destinations and participate in precedence. Use `cli.DefaultsFor` for that role rather than modeling every host-supplied value as a fact.

See the [facts scenario](../examples/host-context/facts/).

## Provenance preserves causal evidence

Provenance is the structured causal history of a contribution or effective value. Inspection can relate a resolved value to its producer and producer kind, resolution tier, input source, origin, exposed source span or revision, and applicable shortcut/provider relationships.

Use those structured relationships when a tool needs to answer where a value came from. Provenance is evidence, not another producer or input source, and should not be flattened into an application-specific string when the public artifact already carries the relation.

## Sensitivity protects Commandment presentation

Mark a destination or producer sensitive when Commandment-owned observations or presentation must not expose the raw material. Sensitivity participates in provenance propagation and redaction, but it does not encrypt the value or create an authorization boundary.

Application code that legitimately receives the value owns its later handling. Tests and diagnostics should prefer structured identity over forcing sensitive raw values into failure text.

## Repair is structured input correction

A diagnostic can publish an `InputRepair`, a machine-actionable suggestion over observed input sources and revisions. Repairs can describe edits or replacements without reducing them to a human hint string.

Commandment does not silently apply a repair. The consuming tool decides whether and how to present or apply it. Use the structured repair rather than parsing diagnostic prose to rediscover edits that are already represented.

[Diagnostics and troubleshooting](11-troubleshooting.md) starts from the public diagnostic report.

## What execution should read

Handlers normally read command configuration after execution admission. If execution also needs an ancestor value, optional destination, resolution-state relationship, effective locale, command selection, standard stream, or another supported host resource, declare that requirement through execution access rather than reaching into globals or rerunning analysis.

[Host Integration and Execution](06-host-integration-and-execution.md) owns that boundary.

## How to test value behavior

Use `clitest.Inspect` for invocation-specific value state, candidates, producer kinds, provenance, facts, domains, and dependencies. Use `clitest.Schema` for purely static declaration structure.

Do not duplicate Commandment's resolution algorithm in test code. Ask the public observation that already records the result. See [Getting Started](01-getting-started.md), [facts](../examples/host-context/facts/), and the [values collection](../examples/values/) for complete examples.
