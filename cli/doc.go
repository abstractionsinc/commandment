// Package cli defines, compiles, analyzes, and runs Commandment command-line applications.
//
// Applications are declared with [Definition] and [CommandBuilder], compiled with [Compile]
// into immutable [Application] values, and operated on through Run, Inspect, Complete, Schema,
// and Reference. CommandBuilder and the declaration capabilities reachable from it are
// callback-scoped authoring values and must not be retained after the callback that supplies
// them returns.
//
// Invocation inputs such as arguments, environment, host access, locale, and analysis limits
// are supplied explicitly by the relevant operation request. [Application.RunFromProcess]
// adapts process state for a normal executable; callers that own invocation inputs directly
// can use [Application.Run].
//
// Schema and Reference expose detached static observations, while Inspect and Complete analyze
// an invocation without mutating the compiled Application. [Diagnostics] extracts structured
// Commandment diagnostics from errors that carry them. Detached observations are caller-owned
// data and may be inspected or modified without changing the Application.
//
// The package keeps declaration, analysis, presentation, completion, and execution on one
// semantic model while leaving application business logic and application-owned dependencies
// in ordinary Go code.

package cli
