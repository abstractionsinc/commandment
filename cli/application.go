package cli

import "context"

// Application is an immutable compiled Commandment program. Its zero value is
// invalid; construct one with Compile. A compiled Application is reusable, and
// each Run, Inspect, Complete, Schema, or Reference operation owns independent
// operation state over the same accepted declarations.
type Application struct {
	_ [0]func()

	identity        *applicationIdentity
	program         staticProgram
	callbacks       declarationCallbacks
	shellCompletion shellCompletionCatalog
	signalSources   []compiledSignalSource
}

func (app Application) valid() bool {
	return app.identity != nil && app.program.Valid()
}

func (app Application) mustStaticProgram() staticProgram {
	if !app.valid() {
		panic("cli: invalid Application")
	}
	return app.program
}

func requireContext(ctx context.Context) context.Context {
	if ctx == nil {
		panic("cli: nil context")
	}
	return ctx
}

func (app Application) signalCatalog() compiledSignalCatalog {
	return freezeCompiledSignalCatalog(app.signalSources)
}

// RunFromProcess runs app using the current process integration and the same
// invocation lifecycle as Run.
func (app Application) RunFromProcess(ctx context.Context) error {
	ctx = requireContext(ctx)
	return runApplicationFromProcess(ctx, app)
}

// Run executes one invocation using the explicit host state in request.
func (app Application) Run(ctx context.Context, request RunRequest) error {
	ctx = requireContext(ctx)
	model := app.mustStaticProgram()
	return runApplication(ctx, model, app.callbacks, app.identity, app.shellCompletion, app.signalCatalog().available(false), request)
}

// Complete analyzes one completion request against app and returns its detached completion result.
func (app Application) Complete(ctx context.Context, request CompletionRequest) (Completion, error) {
	ctx = requireContext(ctx)
	model := app.mustStaticProgram()
	result, err := completeApplicationWithSignals(ctx, model, app.callbacks, app.identity, app.signalCatalog().available(false), request)
	canonicalizeCompletionResult(&result)
	return result, err
}

// ShellCompletionScript returns the generated source artifact for shell.
func (app Application) ShellCompletionScript(shell ShellCompletionProvider) (string, error) {
	if shell == nil {
		return "", errInvalidShellCompletionProvider
	}
	model := app.mustStaticProgram()
	return generateShellCompletionScript(model.Application().name, shell(), "")
}

// ShellCompletionScriptForVersion returns the generated source artifact targeting a specific shell syntax version.
func (app Application) ShellCompletionScriptForVersion(shell ShellCompletionProvider, version string) (string, error) {
	if shell == nil || version == "" {
		return "", errInvalidShellCompletionProvider
	}
	model := app.mustStaticProgram()
	return generateShellCompletionScript(model.Application().name, shell(), version)
}

// Inspect analyzes one explicit invocation request without executing application handlers.
func (app Application) Inspect(ctx context.Context, request InspectionRequest) (Inspection, error) {
	ctx = requireContext(ctx)
	model := app.mustStaticProgram()
	return inspectApplicationWithSignals(ctx, model, app.callbacks, app.identity, app.signalCatalog().available(false), request)
}

// Schema returns the detached static schema of app.
func (app Application) Schema() Schema {
	model := app.mustStaticProgram()
	schema, _ := projectApplicationSchema(model)
	return *schema
}

// Reference returns a detached public reference projection of app according to options.
func (app Application) Reference(options ReferenceOptions) (Reference, error) {
	return projectApplicationReference(app.mustStaticProgram(), options)
}

// applicationMetadata is the canonical application-root metadata partition.
// It is application-lifetime immutable state owned by the application declarations, not
// a facade sidecar or detached artifact authority.
type applicationMetadata struct {
	name                  string
	description           Description
	examples              []Example
	release               ReleaseScheme
	localization          localizationSnapshot
	frameworkLocalization localizationSnapshot
}

func snapshotApplicationMetadata(name string, description Description, examples []Example, release ReleaseScheme, localization, frameworkLocalization localizationSnapshot) applicationMetadata {
	metadata := applicationMetadata{
		name:                  name,
		description:           description,
		examples:              append([]Example(nil), examples...),
		release:               release,
		localization:          localization,
		frameworkLocalization: frameworkLocalization,
	}

	return metadata
}

func (metadata applicationMetadata) Examples() []Example {
	return append([]Example(nil), metadata.examples...)
}
