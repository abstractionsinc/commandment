package benchmark

import "github.com/abstractionsinc/commandment/cli"

// These few config shapes intentionally bound generic instantiation cost in this large external
// benchmark package. Reuse a shape only when its storage is truthful for the workload; do not
// add unrelated state merely to avoid another config type.

type emptyConfig struct{}

type stringConfig struct {
	Value  string
	Values [256]string
}

type flagConfig struct {
	Flags  [256]bool
	Value  string
	Locale cli.Locale
}

type helpConfig struct {
	Flags   [64]bool
	Values  [257]string
	Visible string
}

type inputConfig struct {
	Value   string
	Target  bool
	Strings []string
	Locale  cli.Locale
}

type declarationPlacement string

const (
	placementUnrelated declarationPlacement = "Unrelated"
	placementSelected  declarationPlacement = "Selected"
)
