// Package clitest provides semantic unit-test expectations for public Commandment behavior.
//
// A clitest test starts from a real cli.Application operation or an
// already-observed public Commandment artifact. Inspect, Complete, Schema, and
// Reference each perform one public operation and retain that exact observation.
// Diagnostics retains the DiagnosticReport exposed by an existing error. Compile
// is test setup, and CompileFails acquires a structured compilation failure;
// neither creates a second compiler or assertion language.
//
// Locators describe public semantic subjects. Expectations describe facts to
// evaluate about those subjects. A locator carries locator coordinates, not
// observed application state, and human locators use canonical semantic identity
// rather than parser recognition, prefix matching, or fuzzy selection. Require
// evaluates the retained observation against expectations; reusing one result
// for several Require calls never reruns the underlying operation.
//
// A positive expectation may be satisfied by truthful partial output when the fact
// is present. Negative, exhaustive, and exact-collection expectations require a
// complete relevant observation. When incomplete or truncated output cannot
// establish the requested conclusion, clitest reports the expectation as
// not provable rather than inferring absence or exactness.
//
// Use Compile when a definition should compile successfully and CompileFails
// when structured compilation diagnostics are the subject of the test. A typical
// invocation test is:
//
//	root := clitest.RootCommand()
//	scan := root.Command("scan")
//	clitest.Inspect(ctx, app, cli.InspectionRequest{Args: []string{"scan"}}).Require(t,
//		clitest.InspectionReady(),
//		clitest.SelectionPathExactly(root, scan),
//	)
//
// The SchemaResult.Require example shows the same Require shape with Schema.
//
// Result accessors return semantically exact detached structural snapshots of the
// retained public cli observations when ordinary Go is the clearer assertion tool.
// Mutating a returned snapshot cannot affect the observation used by Require, and
// fallible-operation accessors return the exact retained opaque error unchanged.
// The package intentionally does not wrap cli.Application.Run or
// cli.Application.RunFromProcess; application business effects, application-owned
// errors, generic equality, panic checks, process behavior, fuzzing, race tests,
// benchmarks, and private implementation mechanics remain ordinary Go testing
// concerns.
//
// Reusable expectations, conformance cases, and fixture configuration snapshot
// retained mutable caller input when later mutation would otherwise change an
// already-configured expectation or route. Conformance composes the same
// expectation vocabulary through ordinary testing subtests. ResponseFiles and
// SignalSource are bounded test fixtures for real public cli host contracts, not
// general mocking or host-virtualization frameworks. Parallel tests remain
// subject to the concurrency contracts of the underlying cli operation and
// caller-owned collaborators.
package clitest
