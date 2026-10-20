# Commandment documentation

Start with the [root README](../README.md) for the product tour, then use this index to find the guide that owns your next problem. The [examples](../examples/README.md) provide complete focused programs, while GoDoc owns exact API contracts.

## First application

1. [Getting Started](01-getting-started.md) builds, runs, and semantically tests a first `alienctl` application.
2. [Commandment Guide](02-commandment-guide.md) explains definitions, declaration capabilities, commit points, the static program, invocation operations, and ownership.
3. Continue with the focused guide for the feature you are adding.

## Authoring and runtime

| Need | Read |
| --- | --- |
| Commands, aliases, shortcuts, defaults, arguments, flags, grammar, and command providers | [Commands and Grammar](03-commands-and-grammar.md) |
| Destinations, producers, defaults, facts, derivations, resolution, domains, provenance, and repair | [Values and Resolution](04-values-and-resolution.md) |
| Conditions, availability, visibility, rules, advisories, notices, reachability, and lifecycle | [Policy and Lifecycle](05-policy-and-lifecycle.md) |
| Process input, execution access, middleware, response files, signals, errors, and exit status | [Host Integration and Execution](06-host-integration-and-execution.md) |
| Run, Inspect, Complete, Schema, Reference, help, localization, version, and shell completion | [Operations and Presentation](07-operations-and-presentation.md) |

## Verification and diagnosis

| Need | Read |
| --- | --- |
| Prove Commandment-owned semantic facts | [Testing with clitest](08-testing-with-clitest.md) |
| Prove generated completion in a real shell or editor | [Testing shell completion with shelltest](09-testing-shell-completion-with-shelltest.md) |
| Benchmark one supported successful operation | [Benchmarking with clibench](10-benchmarking-with-clibench.md) |
| Find the public observation that owns a wrong result | [Troubleshooting](11-troubleshooting.md) |

## Larger applications

[Building Large Applications](12-building-large-applications.md) covers package ownership, reusable declarations and policy, dynamic composition, host integration, and layered tests.

[Adopting Commandment](13-adopting-commandment.md) starts from the observable contracts of an existing CLI and shows how to replace its frontend without accidental framework-driven drift.

## Examples and GoDoc

The [examples corpus](../examples/README.md) is the canonical executable consumer-code imitation surface. Each scenario is a small `alienctl` program with a focused test.

Use GoDoc for exact signatures, zero and nil behavior, callback lifetime, ordering, option fields, snapshotting, concurrency, and failure contracts.
