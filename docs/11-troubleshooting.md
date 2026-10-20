# Troubleshooting

Start with the first public observation that contradicts what you expected. If you have an error, check for structured diagnostics. If the operation succeeded but produced the wrong meaning, inspect that operation's structured result before moving outward to presentation or host integration.

## Start with the error

Use `cli.Diagnostics` for Commandment-owned failure evidence.

```go
report := cli.Diagnostics(err)
for _, diagnostic := range report.Diagnostics {
	_ = diagnostic.Code
}
```

Prefer diagnostic code and typed subject, source, origin, and repair relationships over parsing `Message`, `Reason`, or `Hint` text. An empty report means the error carries no Commandment diagnostic payload; middleware and handler errors remain ordinary Go errors and should be handled with `errors.Is`, `errors.As`, and application contracts.

Compilation failures have no application to inspect. Start with the compile error and its diagnostic report, or use `clitest.CompileFails` when a test intentionally proves a structured compile rejection.

## Find the observation that owns the wrong fact

| Symptom | Start here |
| --- | --- |
| Definition does not compile | `cli.Diagnostics(err)` from `cli.Compile` |
| Selection, grammar, resolution, policy, or command-provider result is wrong | `app.Inspect` with the same invocation inputs |
| Completion result is wrong | `app.Complete` |
| Static declaration structure is wrong | `app.Schema()` |
| Reference projection is wrong | `app.Reference(...)` |
| Help presentation is wrong | the `cli.HelpPage` passed to the renderer |
| `Run` reached application code and failed | ordinary Go error handling |
| `clitest` reports a failed proof | the retained public observation and its completeness state |

See the [diagnostics scenario](../examples/observations/diagnostics/) for a complete runnable failure boundary.

## The invocation selected or resolved the wrong thing

Call `app.Inspect` with the same arguments, environment, locale, response-file access, defaults, facts, and other public request inputs that reproduced the problem.

For selection, start with the selected command and selection state, then descend into token ownership, argument forms, or origins only if needed.

For a value problem, work backward from the destination. Check the resolved representation, observed candidates, winning contribution, and finally the input source or origin when provenance is needed. This path works for flags, environment input, defaults, and other producers without treating each source as a separate configuration system.

For a rule failure, inspect the settled value and structured diagnostic. For a command-provider problem, inspect the provider outcome, affected namespace, and diagnostics. Provider materialization is demand-driven, so a namespace that an operation never needed is not evidence that the provider failed.

## Separate semantics from presentation

When visible output is wrong, check whether the structured data is already wrong.

For completion, inspect the returned `cli.Completion` first. If its target, replacement span, items, native requests/delegation, diagnostics, and completeness state are correct but the real shell behaves differently, move outward to the shell/editor integration.

For help, inspect the `cli.HelpPage` passed to the renderer. If the page is correct and the rendered string is wrong, the renderer is the next owner. Apply the same rule to localization before changing presentation code.

`Schema` and `Reference` are static projections. Inspect the artifact itself rather than inferring it from runtime help or an invocation.

## Missing evidence is not always absence

Inspection can retain the most complete valid partial artifact after analysis failure, and completion has independent partial and truncation state. Command-provider-backed namespaces also record whether the relevant demand was resolved completely.

A fact present in valid partial output may still prove presence. Absence, exhaustive membership, and exact collection claims require evidence complete enough to prove them.

If something appears missing, find why analysis stopped before declaring the fact false. Provider/dependency failure, response-file failure, analysis limits, cancellation, or another structured diagnostic may explain the incomplete observation.

`clitest` preserves the same distinction, where `disproved` means the retained observation establishes the requested fact as false and `not provable` means the observation cannot establish the requested negative or exact claim.

## Follow the input Commandment actually saw

Response-file and repair problems are easier to diagnose from source/origin relationships than from the original shell string alone.

Reproduce response-file failures with the same `cli.ResponseFileAccess`, then inspect diagnostics and the source/origin chain around the failed include or loaded source. For deterministic semantic tests, use `clitest.ResponseFiles`; the fixture models Commandment's public response-file host contract, not filesystem normalization or shell behavior.

If a diagnostic publishes an `InputRepair`, inspect the repair and its source snapshots directly. Repairs are structured suggestions and are not applied automatically. Do not reconstruct a machine repair by parsing hint text.

## Run reached application code and failed

Once middleware or a handler runs, application errors remain ordinary Go errors.

```go
err := app.Run(ctx, request)
if errors.Is(err, errExpectedBusinessFailure) {
	// Application-owned error handling.
}
```

`cli.Diagnostics(err)` is still safe because a Commandment-owned branch may exist in the error tree, but ordinary application failures do not receive synthetic diagnostics.

If the question is whether execution was ready before application code ran, use `Inspect`. For cancellation and signal problems, separate caller cancellation, source failure, declared signal policy/endpoint behavior, and application callback errors before changing code. Use `clitest.SignalSource` for deterministic public-protocol tests and a real process test when OS signal behavior is the contract.

## clitest failed a proof

A `clitest` result retains one exact public observation; additional `Require` calls do not rerun the operation. Read the failure against that retained result, and use `Inspection()`, `Completion()`, `Schema()`, `Reference()`, or `Report()` when ordinary Go inspection is useful.

Keep `disproved` and `not provable` separate. Weakening a negative assertion until it passes is not a substitute for proving that the observed domain was complete.

## Produce a useful bug report

Reduce the problem to the smallest public contradiction you can reproduce. Include the Commandment revision and Go version, the smallest relevant `cli.Definition`, the exact public operation and request, the returned error, relevant diagnostics, and the smallest observation fragment that shows the mismatch. Include partial/truncation state and any host fixtures or process/shell prerequisites that affect the result.

A report built from public inputs and observations is both easier to reproduce and less likely to depend on an implementation detail that is unrelated to the defect.
