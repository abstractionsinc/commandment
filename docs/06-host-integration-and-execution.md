# Host Integration and Execution

A compiled Commandment application owns CLI semantics. The host still owns the process, external resources, application dependencies, and business effects.

```text
Commandment                    host application
    grammar                        process state
    resolution                     files and services
    policy                         business dependencies
    execution admission            handler effects
    observations                    application errors
```

Typed integration points connect those responsibilities without forcing handlers to rebuild the frontend from globals.

## Choose RunFromProcess or Run deliberately

`Application.RunFromProcess` is the normal process adapter. It gathers the supported process inputs and runs the same invocation lifecycle as `Run`.

```go
func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.RunFromProcess(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
```

Use `Application.Run` when the caller owns invocation inputs explicitly. A `RunRequest` can supply arguments, environment, working directory, response-file access, invocation defaults, facts, streams, signals, notice delivery, locale, and analysis limits.

Because `Run` does not silently fill missing host capabilities from the process, it suits embedded applications, controlled hosts, tests, and tools. `RunFromProcess` is only the process adapter around the same application semantics.

## Host context is explicit

Environment input, invocation defaults, and facts all come from outside the command line, but they have different meanings.

Environment values feed ordinary environment producers declared by the application, and `RunRequest.Env` supplies the invocation snapshot. Invocation defaults provide overridable fallback candidates for declared destinations. Facts are trusted typed host context outside ordinary CLI grammar.

Do not use a fact as a disguised default or an invocation default where the application requires trusted host context. The [host-context examples](../examples/host-context/) show both roles.

## Execution access exposes required resources to handlers

Command configuration should hold semantic values owned by the command. When execution also needs supported host resources or resolved values from another scope, declare that need through execution access instead of reading package globals.

The execution surface includes resources such as standard streams and working directory together with supported semantic relationships. The [execution examples](../examples/execution/) show complete uses.

Execution access is deliberately narrow rather than a general dependency injection system; application services unrelated to CLI semantics remain ordinary Go dependencies.

## Handlers receive admitted typed configuration

By the time Commandment calls a handler, command selection and grammar analysis have succeeded, required destinations have settled, blocking policy has passed, execution access is ready, and applicable middleware and signal policy have entered the execution lifecycle.

The handler should therefore read typed configuration and perform application work. It should not reparse arguments, reread environment variables, rerun precedence, or repeat frontend validation.

Execution admission is a one-way handoff. Before admission, Commandment owns semantic preparation. After admission, same-scope middleware and the selected handler receive the invocation-local configuration under ordinary Go ownership. Later application mutation does not rewrite finalized resolution, provenance, policy outcomes, signal operands, or detached observations. Distinct invocations do not share configuration storage, although application-owned referents stored inside configuration keep their ordinary Go sharing semantics.

## Middleware composes application execution

Middleware wraps admitted execution with typed command configuration and a `cli.MiddlewareNext` continuation. Use it for behavior that genuinely surrounds execution, such as tracing, authorization around an admitted action, or an application transaction boundary.

Do not move parsing, resolution, availability, rules, or frontend diagnostics into middleware. Those concerns already have semantic owners before admission.

Definition-level middleware applies at the application root and command middleware belongs to the command scope it surrounds. `MiddlewareNext` is callback-scoped, one-shot, and synchronous. Call it once and let downstream settle, or intentionally stop the chain with a non-nil application error. Do not retain it, transfer it to detached work, call it concurrently, or silently return nil without admitting downstream.

GoDoc owns the exact ordering and protocol contracts.

## Application errors remain ordinary Go errors

Once middleware or a handler runs, application errors remain application-owned.

```go
err := app.Run(ctx, request)
if errors.Is(err, errNotAuthorized) {
	// Application-owned handling.
}
```

Commandment does not manufacture a frontend diagnostic because application code returned an error. Use `cli.Diagnostics(err)` for Commandment-owned diagnostics that may be present in an error tree, and use `errors.Is` or `errors.As` for application error identity.

Use `cli.WithExitCode(err, code)` when an application execution error should select an explicit exit status. `cli.ExitCode` queries the status represented by the returned error at the process boundary; an arbitrary `ExitCode() int` method on an application error does not opt into Commandment exit policy.

If middleware admits downstream and both layers return independent failures, Commandment preserves the independent branches while normal Go wrapping remains meaningful. Commandment also does not recover arbitrary application panics or roll configuration back after application mutation.

## Response files split grammar from host loading

Response-file recognition belongs to grammar, while the host owns loading. With response files enabled in the dialect, `RunFromProcess` uses the ordinary filesystem adapter; explicit operations provide `cli.ResponseFileAccess` when they need another loader.

Loaded content re-enters normal grammar and retains its input-source and origin relationships. Tools and embedded hosts can therefore provide their own source acquisition without changing Commandment token semantics.

Use `clitest.ResponseFiles` when the public response-file host protocol is the semantic test subject. See the [response-files collection](../examples/response-files/) for complete programs.

## Signals combine declared policy with host delivery

Signal integration keeps the signal source, signal occurrence, signal selector, signal policy, signal endpoint, and resulting signal actions distinct. The application declares what a signal means; the host supplies actual process delivery or another accepted source.

The [interrupt scenario](../examples/signals/interrupt/) proves real process-interrupt behavior. The [custom-source scenario](../examples/signals/custom-source/) shows an application-defined typed signal source.

Ordinary custom sources publish a bounded descriptor set when opened. Watched sources support integrations whose watched set can change under the public revision and nonretroactivity rules. Use that more complex contract only when the external integration genuinely changes its watch set.

Signal aggregation, phase selection, overflow, source failure, reraising, and action sequencing remain part of the declared signal model. Use a deterministic `clitest.SignalSource` when the public source protocol is under test and a real process test when OS signal behavior is the contract.

## Cancellation has an owner

Caller context cancellation, signal policy, source failure, and application callback errors are different paths. Keep them distinguishable in application code and tests rather than rewriting them into one generic failure.

[Troubleshooting](11-troubleshooting.md) shows which public observations help classify a difficult cancellation or signal problem.

## Keep the process shell thin

For most executables, `main` should compile the package-level definition, run the process adapter, report the returned error, and preserve its exit status. Do not rebuild CLI configuration around the application in `main`.

If configuration is part of the CLI, declare it. If it is application infrastructure, construct it in ordinary Go and connect it at the appropriate application boundary. [Building Large Applications](12-building-large-applications.md) shows that composition across packages.

## Test the boundary that owns the fact

Use `clitest.Inspect` for pre-execution semantic state, `clitest` host fixtures for Commandment-owned response-file or signal-source protocols, ordinary Go tests for handler and middleware effects, subprocess tests for exit status and process behavior, and `shelltest` for real shell completion integration.


Continue with [Operations and Presentation](07-operations-and-presentation.md) for the non-executing questions a compiled application can answer.
