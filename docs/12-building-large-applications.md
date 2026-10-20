# Building Large Applications

A large Commandment application should not be a small example stretched into one enormous declaration callback; it should scale by semantic ownership.

```text
process entry
    -> compile one root definition
    -> run one immutable application

application assembly
    -> compose command-owned declarations
    -> connect truly shared policy and integration

command packages
    -> own command configuration, grammar, values, policy, and behavior

host/application packages
    -> own services, process adapters, storage, network clients, and business logic
```

These are ownership rules, not a prescribed repository tree. Source structure should make the semantic owner easy to find without letting callback-scoped declaration capabilities escape their lifetime.

## Keep root assembly visible

The root definition normally owns application metadata, root configuration, declarations that genuinely belong to the root, and composition of top-level commands.

Do not put every child flag, handler, domain fetch, and helper in the root merely because `Definition.Declare` starts compilation. Do not hide the opposite extreme behind `declareEverything(cb)` and a forest of one-use helpers either. A reader should be able to see the important topology where the application is assembled.

## Let each command own its configuration

Give a command the configuration its handler actually needs.

```go
type fleetConfig struct {
	Region string
}

type inspectConfig struct {
	Format string
}
```

Command-owned configuration keeps destinations, grammar, policy, middleware, and handlers aligned with the scope that uses them. It also prevents unrelated commands from sharing one universal mutable bag of fields.

Application services and host dependencies do not become CLI values merely because they cross a package boundary. Keep ordinary application dependencies in ordinary Go.

## Extract declaration helpers only when they earn a boundary

A declaration helper is useful when it represents a reusable declaration pattern or hides real integration complexity. It may receive callback-scoped capabilities synchronously and use them before returning, but it must not retain, return, or concurrently use those capabilities.

Durable immutable values can cross that boundary when their GoDoc contract permits ordinary reuse, so a helper can construct and return reusable value languages, domains, resolution policies, release schemes, and other public values.

```text
callback-scoped declaration capability
    -> use synchronously inside the owning callback

durable immutable value or policy
    -> construct and reuse according to its public contract
```

Do not add wrapper types merely to move declaration capabilities across packages.

## Prefer lexical declaration for static topology

If the command tree is statically known, keep the important topology lexical at its assembly boundary. Cross-package APIs should expose only the ordinary Go boundary that genuinely earns itself.

A constructor is justified when it binds real per-instance services, provider state, durable declaration values, or another lifetime relationship. A one-use `Command()` factory or mutable package-global command registry is not justified merely because the source file is long.

Use a command provider when the namespace itself is invocation-dependent or externally materialized, not as a source-organization technique.

## Use command providers only for dynamic namespaces

A command provider publishes an accepted batch when its namespace is demanded. Good uses include commands discovered from an external system or another namespace whose members cannot be known statically. Avoid providers whose only purpose is shortening a file or delaying an import.

Accepted provider output becomes ordinary Commandment commands and relationships. Dynamic origin should not create a second command architecture.

See the [dynamic commands scenario](../examples/commands/dynamic/).

## Keep shared policy semantic and narrow

Share a value or policy when the meaning genuinely repeats, not merely because several declarations look similar. A giant `common` package that owns every repeated flag and rule hides command ownership and couples otherwise independent scopes.

The strongest shared abstractions usually have application-domain names rather than framework-mechanics names.

## Keep host concerns at the host boundary

Use the owner of the information to choose the boundary.

- User-selected CLI meaning belongs to command/value declarations.
- Trusted invocation context belongs to facts.
- Overridable host fallback belongs to invocation defaults.
- Commandment execution resources belong to execution access.
- Application/business dependencies remain ordinary Go.
- Process adaptation belongs at the executable or host boundary.

Keeping these boundaries prevents command configuration from becoming a service locator.

## Put cross-cutting execution in middleware only when it is execution

Tracing around admitted handler execution, authorization of an admitted action, or an application transaction boundary can belong in middleware. Grammar, precedence, availability, rules, and diagnostics do not; those owners must remain visible to `Inspect`, `Complete`, help, and tests before execution begins.

## Let tooling consume the compiled application

Reference generation, IDE/editor integration, completion, inspection utilities, compatibility audits, and semantic tests should compile the same definition and use `Schema`, `Reference`, `Inspect`, or `Complete` for the question they need to answer.

Do not build another command registry for tooling.

## Test in layers

Use `clitest` for Commandment-owned semantics such as selection, resolution/provenance, policy, diagnostics, completion, Schema, Reference, and Inspection. Use ordinary Go tests for handlers, middleware effects, collaborators, application errors, and domain logic. Use subprocess tests for process-visible contracts such as exit status and streams, and `shelltest` only for real shell completion integration.

These layers keep semantic failures explainable without turning every test into a giant black-box process test or a white-box declaration test.

## Design for ordinary evolution

A healthy architecture keeps common changes local. Adding a command or producer, extracting a real reusable declaration fragment, adding shared immutable policy, introducing a genuine dynamic namespace, changing lifecycle, supplying a host fact/default, or adding a semantic expectation should not require plumbing through unrelated packages.

Moving a command deserves special care because it changes the canonical command path by default. Update semantic locators and invocation paths to the new path. If the old path is a compatibility contract, preserve it deliberately with the supported command-entry and lifecycle model rather than treating the source move as behavior-neutral.

If routine changes repeatedly expand one global configuration type or require unrelated package edits, revisit ownership before adding more abstraction.

## One possible application shape

A real application may end up with a structure like this.

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

The names are not the rule. The useful properties are that process adaptation stays thin, application assembly is easy to find, command packages own their semantic configuration, genuinely dynamic provider code is isolated, host/application services remain ordinary Go, and semantic tests exercise the public compiled application.

Commandment's public examples intentionally stay flat inside each scenario leaf. Pedagogical examples and production application layout have different jobs.

## Test the architecture itself

Before accepting a large-app structure, ask whether adding, moving, or deprecating one command changes a small semantic owner without violating declaration-capability lifetime. Then ask whether tools can query the same compiled command/value/policy model the executable uses or whether another registry has appeared on the side.

If either answer is no, revisit the ownership boundary.

For an existing mature CLI, continue with [Adopting Commandment](13-adopting-commandment.md).
