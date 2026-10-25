# Testing with clitest

Use `clitest` when a test needs to prove what Commandment understood rather than only what a handler printed or returned.

```go
func TestBeamStrengthFlagWins(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	beamStrength := clitest.Command("abduct").Flag("--beam-strength").Value()
	request := cli.InspectionRequest{
		Env: cli.Env{{
			Name:  "ALIEN_BEAM_STRENGTH",
			Value: "2",
		}},
		Args: []string{"abduct", "--beam-strength", "4", "cow"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		beamStrength.Text("4"),
		beamStrength.ContributedBy(cli.ValueProducerKindFlag),
	)
}
```

A handler-output assertion could prove that the application received `4`. It would not prove why the value resolved to `4`. This test asks Commandment's public inspection result for the semantic fact directly.

## Start with the question

Choose the entry point that owns the fact.

| Question | Use |
| --- | --- |
| Should this definition compile? | `clitest.Compile` |
| Why should this definition fail to compile? | `clitest.CompileFails` |
| What does this invocation mean before execution? | `clitest.Inspect` |
| What should completion return for partial input? | `clitest.Complete` |
| What static structure did compilation produce? | `clitest.Schema` |
| What reference view is exposed for these options? | `clitest.Reference` |
| What Commandment diagnostics are carried by this error? | `clitest.Diagnostics` |

`Compile` calls the real compiler and fails the test if compilation does not succeed, while `CompileFails` owns the inverse case. The observation families return retained results whose `Require` methods check Commandment-specific facts. If you already have a public `cli.HelpPage`, use `clitest.RequireHelpPresentation` for structured help-presentation checks.

## Find the subject and state the fact

Most assertions first locate a semantic subject and then state what should be true about it.

```go
beamStrength := clitest.Command("abduct").Flag("--beam-strength").Value()
proof.Require(t,
	beamStrength.Text("4"),
	beamStrength.ContributedBy(cli.ValueProducerKindFlag),
)
```

The locator follows public relationships from the command to its exact flag spelling and destination; it does not contain the observed value. Prefer recognizable coordinates such as command paths and flag spellings. ID- and order-based locators exist where those are the public coordinates, and GoDoc owns the complete locator/expectation surface.

## Make expected failures specific

Do not accept any non-nil error when the test cares about one Commandment rejection.

```go
diagnostics := clitest.CompileFails(t, badDefinition)
diagnostics.Require(t,
	clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present(),
)
```

`CompileFails` requires structured Commandment diagnostics; for fallible analysis operations, use `clitest.FailsWith(...)` and the diagnostic expectations that identify the failure under test.

Application-owned errors remain ordinary Go, so `clitest` does not wrap `Application.Run` or `RunFromProcess`; use `errors.Is`, `errors.As`, and application-specific assertions there.

Failure reports preserve Commandment sensitivity and redaction instead of printing protected values for convenience. Rejection details follow the public Diagnostic relationship directly:

```go
clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection().Code(code)
clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection().ExpectedExactly("orbit", "land")
clitest.Diagnostic(cli.DiagnosticInvalidValue).Rejection().SpansExactly(span)
```

For a sensitive Diagnostic with a rejection relationship, exact rejection expected fragments are deliberately not provable: redacted content is not equivalent to an empty collection, and `clitest` will not echo either caller-supplied or observed secret material in the failure report. Safe rejection code and span structure remain provable.

## Reuse one observation for several checks

A result retains the public observation, so several assertions do not rerun the operation.

```go
proof := clitest.Inspect(t.Context(), app, request)
proof.Require(t, clitest.InspectionReady())
proof.Require(t, clitest.Command("abduct").Selected())
```

When ordinary Go is clearer, read the retained public value through `Inspection()`, `Completion()`, `Schema()`, `Reference()`, or `Report()`. Accessors return detached snapshots and preserve the exact retained error for fallible operations.

## Do not infer absence from partial output

Presence and absence need different evidence. An item found in a partial completion proves presence; an item missing from a partial or truncated completion does not prove absence.

The same rule applies to exact collections and other incomplete observations. `clitest` reports a fact as not provable when the retained result cannot justify it. Prove completeness explicitly when completeness is part of the contract, using the applicable incomplete/partial/truncated expectations.

## Use fixtures for Commandment-owned host protocols

`clitest.ResponseFiles` provides deterministic declaration data for the public response-file host protocol. `Access()` snapshots that data into `cli.ResponseFileAccess`, and request observations are available when the load or completion request itself matters. Filesystem semantics remain ordinary Go concerns.

`clitest.SignalSource` returns an ordinary `cli.SignalSource[K]` plus a separate test control. Pass the source into the application and use the control to observe lifecycle, inject notifications or failure, and inspect reraises. `clitest.SignalWatchSource` provides the corresponding watched-state protocol for integrations that actually need it.

Use process tests when the process is the contract. These fixtures model Commandment host protocols rather than acting as a general mocking layer.

## Share repeated semantic checks with Conformance

Use `Conformance` when the same Commandment behavior must hold across several applications or reusable components.

```go
checks := clitest.Conformance{
	clitest.SchemaCase{
		Name: "root command",
		Expectations: []clitest.SchemaExpectation{
			clitest.RootCommand().Name("alienctl"),
		},
	},
	clitest.InspectionCase{
		Name:    "abduct is selectable",
		Request: cli.InspectionRequest{Args: []string{"abduct", "cow"}},
		Expectations: []clitest.InspectionExpectation{
			clitest.InspectionReady(),
			clitest.Command("abduct").Selected(),
		},
	},
}

checks.Require(t, t.Context(), app)
```

`Conformance.Require` runs ordinary Go subtests in declaration order. Compose reusable case slices with ordinary Go when dynamic composition is genuinely needed. Do not use Conformance as a replacement for normal table tests or general assertions.

## Leave ordinary Go in charge of ordinary Go problems

Use `clitest` for Commandment-owned semantic facts. Ordinary Go testing remains the owner of generic equality, handler effects, application collaborators, application error identity, panic checks, subprocess behavior, race/concurrency orchestration, fuzzing, private mechanics, and exact rendered output when presentation itself is the contract.

`clitest` saves you from rebuilding Commandment semantics in tests; it does not replace package `testing`.

See [Getting Started](01-getting-started.md) for the opening test, the [examples corpus](../examples/README.md) for focused tests beside complete programs, and [Operations and Presentation](07-operations-and-presentation.md) for the public operations those tests observe.
