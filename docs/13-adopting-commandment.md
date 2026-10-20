# Adopting Commandment

Replacing the frontend of a mature CLI is a compatibility project before it is a framework project.

Start with the behavior that users, scripts, shell integrations, and operators already depend on. Decide what must remain compatible, what should change deliberately, and what was only an implementation detail. Then design the Commandment application around those decisions.

This method applies whether the existing CLI is small, kubectl-scale, hand-written, or built on another library.

## Inventory the language users type

Record the public grammar before designing the new source:

- root and nested command paths;
- lexical aliases and command shortcuts;
- default relationships;
- flag spellings and counting/value behavior;
- positional argument forms;
- pass-through and end-of-options behavior;
- command and flag normalization rules;
- response-file recognition;
- shell-visible completion behavior where grammar affects it.

Model the intended language with Commandment commands, flags, argument forms, grammar profiles, dialect, command entries, and token ownership. Do not preserve an old implementation merely because it produced the existing grammar.

If the migration also changes the CLI language, record that as a product change and prove it separately from the framework replacement.

## Inventory values and resolution

For each meaningful configuration value, record its type, accepted sources, defaults, precedence, repeated-input behavior, environment names, policy, dynamic legal values, host-supplied context, and sensitivity requirements.

Map those responsibilities onto destinations, producers, resolution policy, value domains, facts, invocation defaults, dependencies or derivations, rules, and sensitivity. The new application should remove duplicated merge logic rather than move it into new handlers.

When precedence is user-visible or operationally important, compatibility tests should prove both the effective value and its provenance.

## Inventory presentation and completion

Capture the presentation behavior users rely on:

- help organization and descriptions;
- examples;
- visibility;
- lifecycle and declaration-replacement presentation;
- localization and locale selection;
- version output;
- reference generation;
- semantic completion;
- completion delegation;
- generated shell completion integration.

Commandment should become the semantic authority for these projections. Do not retain a second command model solely because the old implementation used it for help or completion.

Exact whitespace is a compatibility requirement only when the product intentionally makes it one. Prefer semantic compatibility when presentation may legitimately improve.

## Inventory process and host integration

Record how the executable interacts with its host:

- process arguments and environment;
- working directory;
- standard streams;
- response files;
- signals and cancellation;
- exit status and application-error mapping;
- host facts and defaults;
- locale;
- external services and business dependencies.

Keep Commandment-owned frontend behavior separate from ordinary application infrastructure. Migration is a good opportunity to remove package globals that existed only because the old frontend lacked an explicit host boundary, but it is not a reason to force every service into Commandment configuration.

## Inventory compatibility and lifecycle

Record declarations that are new, deprecated, scheduled for removal, replaced, or retained as compatibility syntax.

Model intentional lifecycle and declaration replacement explicitly. Do not keep an internal spelling simply because it once existed, and do not silently remove a widely used spelling because the new declaration is cleaner.

## Record real performance requirements

A mature CLI may have release requirements for startup latency, completion latency, memory use, or another measurable property. Keep those requirements only when they are real, and assign each one to the proof boundary that can measure it.

Use `clibench` when the requirement is exactly one supported Commandment operation over a reusable workload. End-to-end startup or shell latency needs a broader benchmark. An application's performance target is not automatically a generic Commandment promise.

## Design the target ownership

Use the inventory to decide where the new application owns each responsibility.

```text
command package
    command configuration
    grammar
    values
    command policy
    command handler or middleware

application assembly
    root definition
    top-level composition
    truly shared semantic values and policies

host and application packages
    services
    fact and default providers
    process adapters
    external integrations
```

Do not preserve old package boundaries automatically. Preserve the intended public behavior and redesign internal ownership where the old structure no longer earns itself.

[Building Large Applications](12-building-large-applications.md) develops these composition rules.

## Build a compatibility corpus

Choose the smallest public proof for each intended invariant.

Use `clitest` for Commandment-owned semantics such as selection, value resolution and provenance, availability, visibility, policy, diagnostics, repair, completion, schema/reference relationships, and lifecycle state.

Use ordinary Go tests for application behavior. Use subprocess tests for process-visible contracts such as exit status, streams, and end-to-end output when that output is intentionally stable. Use `shelltest` for real supported shell completion behavior.

Do not make one golden test prove unrelated boundaries.

## Migrate in supported increments

The useful increment depends on the existing executable. It might be a command subtree, a value-resolution family, a process adapter, or another coherent user-visible unit.

Do not invent a side-by-side Commandment and legacy runtime if the product does not support one. When both frontends cannot safely coexist in one process, migrate behind source and test boundaries, then switch the executable once the compatibility corpus is green.

## Classify differences deliberately

When the new application differs from the old one, classify the difference before changing either implementation or expectation.

```text
BUG
    intended compatibility changed accidentally

INTENTIONAL REDESIGN
    product behavior is deliberately changing

OLD IMPLEMENTATION DETAIL
    internal behavior was never a public contract

UNSUPPORTED LEGACY BEHAVIOR
    Commandment does not represent it and product owners must decide
```

This prevents a migration from preserving every historical accident merely because a snapshot captured it or changing user behavior accidentally because another shape is easier to express in the new framework.

## Re-prove the external boundaries

Before cutover, rerun the evidence at the boundary that owns each contract:

- semantic compatibility;
- application behavior;
- process behavior;
- real-shell behavior;
- explicit performance acceptance checks.

Then review the compiled application with `Schema`, `Reference`, and `Inspect`. Those operations can expose duplicate commands, missing lifecycle, unexpected source precedence, or presentation drift more clearly than declaration source alone.

## What a large migration should achieve

A successful migration leaves one compiled semantic model instead of separate parser, help, completion, and test descriptions. Commands own typed configuration, and resolution is explicit rather than repeated as merge logic. Dynamic namespaces use command providers only when they are genuinely dynamic. Host integration remains separate from CLI semantics, lifecycle is explicit, semantic tests explain semantic failures, and process or shell tests remain only where those boundaries matter.

That is a stronger result than making every old command run. It preserves the contracts worth preserving while replacing accumulated frontend architecture with one coherent model.

For subsystem authoring, return to the [documentation index](README.md). For migration failures, start with [Troubleshooting](11-troubleshooting.md).
