# Policy and Lifecycle

Commandment keeps several policy questions separate because selection, completion, help, inspection, diagnostics, and execution need to observe the same distinctions. Structural participation, presentation visibility, blocking validation, nonfatal advice, notice delivery, and release evolution are not one generic `enabled` or validation state.

## Conditions describe activation

A condition is an inert expression over declared semantic state.

```go
hazardous := cb.When.Equal(unsafe, true)
```

Constructing the condition does not change the application. It gains semantic effect when an owning declaration consumes it through `AvailableWhen`, `VisibleWhen`, an advisory, notice suppression, or another condition-bearing capability.

Keep conditions inside their valid declaration scope. They are declarative relationships, not mutable runtime booleans.

## Availability controls structural participation

Availability answers whether a declaration may participate in the effective model for an invocation.

```go
hazardous := cb.When.Equal(unsafe, true)
override.AvailableWhen(hazardous)
```

An unavailable declaration is semantically unavailable, not merely hidden. Parsing, completion, help/reference projections, inspection, and diagnostics can all depend on that fact. Do not accept the input and reject it later in a handler when the real rule is availability.

## Visibility controls presentation

Visibility answers whether presentation should expose an otherwise available declaration.

```go
cb.VisibleWhen(cb.When.Not(hazardous))
```

Use visibility when a declaration remains valid but should not normally be advertised. Use availability when it must not participate at all.

```text
availability -> may this declaration participate?
visibility   -> should presentation expose it?
```

## Invocable, runnable, execution admission, and reachability are different

A command can be reachable in the command structure without being runnable. It can be invocable according to its command role and still fail execution admission because required analysis or policy did not succeed. Static reachability from the root is another property again.

Application code rarely manipulates these states directly, but tooling and tests must not flatten them into one generic flag. `Schema`, `Reference`, `Inspection`, diagnostics, and `clitest` preserve the distinctions where they matter.

## Rules are blocking semantic validation

A rule is a declared blocking constraint over settled semantic values.

```go
cb.Rule.Between(beamStrength, 1, 10)
```

Put a constraint in Commandment when it is part of the CLI contract, such as an accepted range, a relationship between values, or a required combination that must hold before execution. Keep application business validation in application code when the CLI model does not own the check.

A rule violation is structured semantic failure. Use the resulting violation and diagnostic relationships rather than parsing rendered text.

## Advisories are nonfatal semantic outcomes

An advisory records a semantic fact worth communicating without rejecting the invocation.

```go
cb.Advisory.When(hazardous, advisory).Subject(unsafe)
```

An advisory is not a logging hook. Inspection, presentation, tests, and notice delivery can all observe the same semantic outcome.

A notice is the user-facing notification derived from semantic state. It is not another name for the advisory itself.

## Notice suppression controls delivery, not truth

Notice suppression prevents a matching notice from being delivered under a declared condition without erasing the underlying advisory.

```go
cb.Notice.SuppressWhen(hazardous, cli.NoticeMatch{AdvisoryCode: advisory.Code})
```

Use suppression for delivery policy while tools and tests continue to observe the semantic fact that produced the notice.

The [conditions](../examples/policies/conditions/) and [advisories](../examples/policies/advisories/) scenarios show conditional availability, advisories, and notice suppression in complete CLIs.

## Lifecycle describes release evolution

Lifecycle records when one exact declaration is introduced, deprecated, or removed according to the application's release scheme.

```go
scanCmd := cli.Command[scanConfig]{
	Name: "scan",
	Release: cli.ReleaseLifecycle{
		DeprecatedIn: "2.0.0",
	},
}
```

The application release scheme defines how release coordinates are compared. Commandment provides semantic-version and explicit ordered schemes through `cli.Release`.

Lifecycle is declaration metadata, so do not implement it by checking version strings in handlers or by maintaining unrelated presentation-only version state. GoDoc owns the exact lifecycle fields and zero-value rules.

## Declaration replacement is a separate relationship

Declaration replacement says that one declaration supersedes another.

```go
scan.ReplacedBy(probe)
```

Replacement and lifecycle answer different questions. Lifecycle says where a declaration stands in release evolution; replacement says which declaration users should move to. Do not infer replacement from aliases, similar names, help text, or deprecation prose.

The examples corpus does not need a contrived executable solely to demonstrate metadata relationships that are already taught here and specified in GoDoc.

## Keep policy with its semantic owner

Place a rule, condition, availability relation, advisory, notice policy, or lifecycle declaration near the command, value, or declaration it qualifies unless reuse is genuinely semantic. Moving all policy into one global package usually separates the policy from the values and scopes it governs.

Reusable immutable policy values can be appropriate when the same semantic policy truly repeats. Callback-scoped declaration capabilities still remain inside their valid scope. [Building Large Applications](12-building-large-applications.md) covers that composition boundary.

## Test the semantic property, not its rendering

Use `clitest.Inspect` for invocation-specific availability, rule outcomes, advisories, notices, lifecycle state, and execution admission. Use `clitest.Schema` for static declaration relationships and `clitest.Reference` when the reference projection itself is the contract. Application-owned behavior after execution begins stays with ordinary Go tests.

A visible string is usually a presentation consequence rather than the policy fact you need to prove.

Continue with [Host Integration and Execution](06-host-integration-and-execution.md) when the question moves from declared policy to process and host behavior.
