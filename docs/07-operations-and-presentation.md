# Operations and Presentation

Compilation produces one immutable application that can answer several different questions.

```text
Run       -> execute an accepted invocation
Inspect   -> explain one invocation without application execution
Complete  -> reason about partial input
Schema    -> expose static compiled structure
Reference -> expose reader-facing static reference information
```

Help, localization, version presentation, diagnostics, completion integration, and developer tooling should consume those semantics instead of maintaining parallel descriptions of the CLI.

## Run executes application behavior

`Application.Run` analyzes explicit invocation input and, after execution admission, runs middleware and the selected handler. Use `RunFromProcess` when the process is the host and `Run` when the caller already owns the request inputs.

If you only need to know what Commandment would understand, use `Inspect` rather than executing the application.

## Inspect explains one invocation

`Application.Inspect` takes explicit invocation input without running application middleware or handlers. It can expose selection, resolution state and provenance, provider/domain outcomes, policy results, diagnostics, execution admission, and other invocation observations.

Use it for questions such as which command would win, why a value resolved to `4`, which producer contributed it, whether a declaration was available or visible, whether a rule blocked execution, or what dynamic command/domain materialized.

Inspection can preserve valid partial evidence when analysis fails. Presence may still be provable from a partial artifact, but exact absence requires enough completeness to support that claim.

See the [observations collection](../examples/observations/) and [Troubleshooting](11-troubleshooting.md).

## Complete reasons about partial input

`Application.Complete` analyzes editor state rather than a complete invocation.

A completion target is the semantic subject currently being completed. A completion candidate is a proposed semantic completion, items are finalized client-facing results, completion matching describes how candidate text relates to observed input, completion ranking orders accepted candidates, and completion delegation asks a native or external completion mechanism to own part of the result.

Values and token languages can provide semantic completion callbacks or native completion requests where supported. Dynamic command providers and value domains can participate when the operation materializes the namespace or domain they own.

Semantic completion does not prove shell completion integration. Quoting, editor behavior, generated adapter source, protocol transport, and runtime behavior belong to the shell boundary. Use [Testing Shell Completion with shelltest](09-testing-shell-completion-with-shelltest.md) for that proof.

The [completion collection](../examples/completion/) shows semantic candidates, native file delegation, and shell integration.

## Schema exposes the static program

`Application.Schema()` returns detached static structure from the compiled application. Use it for accepted command and declaration identity, static grammar, destination/dependency structure, lifecycle/presentation metadata, provider declarations, and other static relationships.

Schema is not an invocation trace. Command-provider-produced batches and invocation-scoped dynamic-domain contents remain operation-specific observations. The returned schema is caller-owned data and mutating it does not mutate the application.

## Reference is reader-facing static projection

`Application.Reference(options)` returns a detached reader-oriented static projection.

```text
Schema    -> structural compiled truth for tools
Reference -> reader-facing static reference
```

Reference generators should consume `Reference` rather than scrape rendered help or rebuild reference structure from internal declarations. The [observations collection](../examples/observations/) shows both static projections.

## Diagnostics are structured failure evidence

When Commandment owns a failure, `cli.Diagnostics(err)` extracts its structured diagnostic report. Prefer diagnostic code, subject, source/origin, repair, and other typed relationships over parsing human `Message`, `Reason`, or `Hint` text.

Handler and middleware errors may have no Commandment diagnostic because the application owns those failures. A diagnostic can also point to an `InputRepair`, which is structured correction data rather than a hint string; Commandment does not apply repairs automatically.

See the [diagnostics scenario](../examples/observations/diagnostics/) and [Troubleshooting](11-troubleshooting.md).

## Help is presentation over the semantic model

The built-in help action and direct help rendering consume structured presentation derived from the application. `cli.HelpConfig` configures the frontend and a custom renderer receives a `cli.HelpPage` rather than rediscovering commands, flags, lifecycle, localization, or groups from another registry.

Disabling help as a CLI route does not erase the underlying structured presentation/reference capability. Descriptions, examples, grouping, action entries, lifecycle state, finite-domain presentation, flag spellings, and localized labels remain inputs to presentation rather than strings owned by one printer.

The [presentation collection](../examples/presentation/) covers custom help, localization, and version behavior.

## Localization has explicit ownership

Application-authored text and Commandment-owned framework phrases have different localization owners. The application supplies its default locale and translation bundles for application text, while framework translations remain separate.

Translation identity follows authored source text rather than ad hoc renderer keys. The effective locale belongs to an operation rather than package-global mutable state, and invocation-time command-provider output follows the provider-localization rules instead of mutating the static catalog.

Renderers consume resolved localized presentation without selecting locale again or maintaining a shadow translation map for Commandment semantics.

## Version presentation follows release authority

The application release scheme is lifecycle comparison authority, and version presentation projects the application's release/version configuration. Do not maintain unrelated version strings for help, lifecycle, and generated reference unless the product truly has separate concepts.

## Shell completion integration is an adapter layer

`Application.Complete` is shell-neutral, while a shell completion provider adapts a supported shell/editor protocol to that operation and can generate integration source for a specific supported target/version when required. Built-in providers live under `cli.Shell`; custom providers should adapt the same semantic completion engine rather than introduce another one.

Direct callers with exact execution arguments use `CompletionRequest.Args`. Clients that observe shell/editor words without execution-equivalent argv use `ObservedInput`, which preserves word fidelity, argv-topology certainty, and cursor coordinate space. Do not manufacture decoded argv or translate coordinate spaces just to fit the simpler request lane.

```text
Application.Complete          semantic partial-input analysis
shell completion provider     shell-specific adapter/generation authority
shell completion generation   emitted integration source
shell completion integration  installed client/runtime behavior
shelltest                     real-shell proof frontend
```

The [shell completion scenario](../examples/completion/shell/) demonstrates generation and real-shell verification.

## Choose the narrowest operation that owns the question

Use `Inspect` for one complete invocation, `Complete` for partial editor input, `Schema` for static structural truth, `Reference` for static reader-facing reference, and `Run` for application effects. Tools then stay on public observations instead of coupling to implementation internals.

Continue with [Testing with clitest](08-testing-with-clitest.md) for semantic verification of those operations.
