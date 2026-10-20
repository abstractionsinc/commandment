package shelltest

import "github.com/abstractionsinc/commandment/cli"

// RuntimeVersionParser derives the semantic runtime version from bounded probe output.
type RuntimeVersionParser func(string) (string, error)

// Runtime declares one exact shell runtime and its version probe. Its zero value is invalid:
// Executable and VersionArgs must be nonempty and ParseVersion must be non-nil. VersionArgs is
// snapshotted when an operation retains the runtime declaration.
type Runtime struct {
	Executable   string
	VersionArgs  []string
	ParseVersion RuntimeVersionParser
}

// PrepareFunc declares provider-owned shell setup before the real shell starts.
type PrepareFunc func(Setup) error

// InterpretFunc maps one settled shell observation into provider-owned semantics.
type InterpretFunc[R any] func(Observation) (R, error)

// Target binds one production completion provider to its runtime and adapter behavior. Its zero
// value is invalid: Provider and Interpret must be non-nil. Prepare may be nil. Operations snapshot
// mutable Runtime inputs they retain so later caller mutation cannot change an in-flight proof.
type Target[R any] struct {
	Provider  cli.ShellCompletionProvider
	Runtime   Runtime
	Prepare   PrepareFunc
	Interpret InterpretFunc[R]
}

// Setup is a callback-scoped declarative capability supplied to PrepareFunc. Its zero value is
// invalid. Setup values are noncomparable, must not be retained after PrepareFunc returns, and
// must not be used concurrently.
type Setup struct {
	_       [0]func()
	session *setupSession
}

// Observation is a callback-scoped immutable settled view supplied to InterpretFunc. Its zero value
// is invalid. Observation values are noncomparable and must not be retained after InterpretFunc
// returns. Slice-returning methods return defensive copies.
type Observation struct {
	_     [0]func()
	state *observationState
}

// Input is editor text split at the completion cursor. Its zero value is valid empty editor input with the cursor at the start.
type Input struct {
	BeforeCursor string
	AfterCursor  string
}
