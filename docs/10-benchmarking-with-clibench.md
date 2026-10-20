# Benchmarking with clibench

Use `clibench` when the timed subject is exactly one successful Commandment operation over a reusable workload.

```go
func BenchmarkInspect(b *testing.B) {
	request := cli.InspectionRequest{Args: []string{"abduct", "--beam-strength", "3", "cow"}}
	clibench.Inspect(b, app, request)
}
```

This measures repeated successful `app.Inspect` calls. Application construction, rendering, process startup, and other setup remain outside the measurement unless you put them inside the timed path yourself.

## Choose the operation before the helper

`clibench` has helpers for `Compile`, `Run`, `Inspect`, `Complete`, `Schema`, and `Reference`. Each helper repeats that public operation inside `b.Loop()`, treats an unexpected error as benchmark failure, and discards the successful result.

Choose the helper from the work that belongs inside the timed loop. Use `clibench.Complete` to isolate semantic completion; benchmark a larger path directly when the claim also includes rendering, transport, process startup, or application work.

## clibench keeps one narrow boundary

For supported operations, the helper keeps this shape consistent.

```text
reusable fixture -> one successful public operation -> discard result
```

`Run`, `Inspect`, and `Complete` reuse one benchmark-lifetime `b.Context()`. `Schema` and `Reference` need no context. `Compile` recompiles the supplied `cli.Definition` on every iteration.

Establish that the definition, application, request, and expected-success path are correct before interpreting benchmark numbers. The workload must also mean the same thing on every iteration; a helper cannot make stateful or consuming input reusable.

## Let testing.B own benchmark structure

Use ordinary sub-benchmarks for the coordinates you want to compare.

```go
func BenchmarkInspect(b *testing.B) {
	cases := []struct {
		name    string
		request cli.InspectionRequest
	}{
		{
			name:    "short",
			request: cli.InspectionRequest{Args: []string{"abduct", "cow"}},
		},
		{
			name:    "with-flags",
			request: cli.InspectionRequest{Args: []string{"abduct", "--beam-strength", "3", "cow"}},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			clibench.Inspect(b, app, tc.request)
		})
	}
}
```

`testing.B` owns benchmark hierarchy, names, timer/tooling behavior, profiles, statistics, and the rest of the benchmark lifecycle. `clibench` owns only the repeated successful Commandment call.

## Use testing.B directly when lifecycle is part of the question

Write the loop yourself for fresh per-iteration state or context, expected failures, custom concurrency, process startup, end-to-end executable cost, application-specific work around the Commandment operation, or cold/one-shot behavior.

Do not force a workload through `clibench` merely because a helper exists. The helper is useful only when its boundary matches the claim.

## Interpret the number narrowly

A `clibench` result describes the benchmark you wrote on the Go toolchain and environment that ran it. It does not by itself establish end-to-end latency, cold-start cost, production throughput, cross-machine behavior, or a general performance promise.

Before acting on a result, confirm that the helper isolates the intended operation, setup is timed or untimed deliberately, the workload remains equivalent across iterations, unexpected failure is truly a failure for the case, and both sides of a comparison use the same meaningful boundary.

Use GoDoc for exact helper signatures. Return to [Operations and Presentation](07-operations-and-presentation.md) when the operation boundary is unclear, and use [Troubleshooting](11-troubleshooting.md) before interpreting a benchmark whose operation is already returning an unexpected semantic result.
