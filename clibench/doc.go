// Package clibench provides narrow testing.B helpers for benchmarking one
// successful public Commandment operation over a reusable workload.
//
// Each helper executes exactly one supported operation per B.Loop iteration and
// fails the benchmark on an unexpected operation error. Inspect, Complete,
// Schema, Reference, and Compile discard successful results. Run, Inspect, and
// Complete reuse one benchmark-lifetime context from B.Context; Schema and
// Reference are context-free, and Compile recompiles the supplied definition on
// every iteration.
//
// Establish correctness before calling the helper, and keep the workload
// semantically equivalent across iterations. Use testing.B directly when the
// measurement requires fresh state or context per iteration, expected failures,
// custom concurrency, process or startup work, or application-specific work
// around the Commandment operation.
//
// A benchmark with an already-compiled app can keep setup outside the timed
// helper:
//
//	func BenchmarkInspect(b *testing.B) {
//		request := cli.InspectionRequest{Args: []string{"scan"}}
//		clibench.Inspect(b, app, request)
//	}
//
// Benchmark hierarchy, coordinates, timing policy, profiles, statistics, and
// other Go benchmarking mechanics remain owned by testing.B and Go tooling.
package clibench
