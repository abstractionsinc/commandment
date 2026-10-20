package cli

// Definition describes an application before compilation. Compile snapshots its metadata and
// invokes Declare with a fresh callback-scoped CommandBuilder whose Config is the application
// configuration instance; the zero value does not describe a valid named.
type Definition[C any] struct {
	Name string

	Description Description
	Release     ReleaseScheme
	Examples    []Example

	Help            HelpConfig
	Version         VersionConfig
	ShellCompletion ShellCompletion

	Declare func(*CommandBuilder[C])

	Localization          Localization
	FrameworkTranslations []TranslationBundle

	Middleware []Middleware[C]
	Run        Handler[C]
}
