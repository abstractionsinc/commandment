package cli

import (
	"errors"
	"slices"
)

// Compile validates and compiles definition into a reusable Application. Static
// declaration failures are returned as structured frontend errors discoverable
// through Diagnostics; no partially compiled Application is published.
func Compile[C any](definition Definition[C]) (Application, error) {
	var compiled Application
	var draft applicationDraft
	defer draft.discard()

	root, release, err := compileRoot(&draft, definition)
	if err != nil {
		return compiled, compileDeclarationError(err)
	}
	if err := invokeDeclare(&draft, root, definition.Declare); err != nil {
		return compiled, compileDeclarationError(err)
	}
	if err := compileProcessSignals(&draft); err != nil {
		return compiled, compileDeclarationError(err)
	}
	if err := compileHelp(&draft, root, definition.Help); err != nil {
		return compiled, compileDeclarationError(err)
	}
	if err := compileVersion(&draft, root, definition.Version, release); err != nil {
		return compiled, compileDeclarationError(err)
	}
	shellCompletion, err := compileShellCompletion(&draft, root, definition.ShellCompletion)
	if err != nil {
		return compiled, compileDeclarationError(err)
	}

	identity := draft.identity
	program, callbacks, signalSources, err := draft.commit()
	if err != nil {
		return compiled, compileDeclarationError(err)
	}
	compiled = Application{
		identity:        identity,
		program:         program,
		callbacks:       callbacks,
		shellCompletion: shellCompletion,
		signalSources:   signalSources,
	}
	return compiled, nil
}

func compileDeclarationError(cause error) error {
	if errors.Is(cause, errBuilderMisuse) {
		return projectDiagnosticFailure(DiagnosticBuilderMisuse, "builder misuse", cause)
	}
	return projectDiagnosticFailure(DiagnosticInvalidDeclaration, "invalid declaration", cause)
}

type applicationDraft struct {
	closed   bool
	identity *applicationIdentity

	name                  string
	description           Description
	examples              []Example
	release               ReleaseScheme
	localization          localizationSnapshot
	frameworkLocalization localizationSnapshot
	middleware            []commandMiddleware

	declarations declarationDraft

	signalSources            []compiledSignalSource
	nextSignalID             uint64
	invalidSignalDeclaration bool
}

func (draft *applicationDraft) discard() {
	if draft == nil || draft.closed {
		return
	}
	*draft = applicationDraft{
		closed: true,
	}
}

func compileRoot[C any](draft *applicationDraft, definition Definition[C]) (commandRef, ReleaseScheme, error) {
	var root commandRef
	var release ReleaseScheme
	if draft == nil || definition.Name == "" {
		return root, release, errInvalidAuthoringTransaction
	}

	release, err := compileRelease(definition.Release)
	if err != nil {
		return root, release, err
	}
	localization, err := snapshotLocalization(definition.Localization)
	if err != nil {
		return root, release, err
	}
	frameworkLocalization, err := snapshotFrameworkLocalization(definition.FrameworkTranslations)
	if err != nil {
		return root, release, err
	}
	middleware, err := compileMiddleware(definition.Middleware)
	if err != nil {
		return root, release, err
	}

	draft.identity = &applicationIdentity{}
	draft.name = definition.Name
	draft.description = definition.Description
	draft.examples = slices.Clone(definition.Examples)
	draft.release = release
	draft.localization = localization
	draft.frameworkLocalization = frameworkLocalization
	draft.middleware = middleware
	draft.declarations.scope = identityScope{
		application: draft.identity,
	}
	root = stageRoot(draft, definition.Run)

	return root, release, nil
}

func invokeDeclare[C any](draft *applicationDraft, command commandRef, declare func(*CommandBuilder[C])) error {
	if draft == nil {
		return errInvalidAuthoringTransaction
	}
	if declare == nil {
		return nil
	}
	session := draft.prepareSession(command)
	builder := commandBuilder(new(C), session, grammarProfileCaseRef{})
	if cause := session.invoke(func() { declare(builder) }); cause != nil {
		return cause
	}
	return draft.declarations.rejectionError()
}

func compileProcessSignals(draft *applicationDraft) error {
	if draft == nil || draft.invalidSignalDeclaration {
		return errInvalidSignalSourceDeclaration
	}
	if _, ok := registerProcessSignalSource(draft); !ok {
		return errInvalidSignalSourceDeclaration
	}
	return nil
}

func compileHelp(draft *applicationDraft, root commandRef, help HelpConfig) error {
	if draft == nil {
		return errInvalidAuthoringTransaction
	}
	if _, ok := draft.declarations.commands.resolve(&draft.declarations, root); !ok {
		return errInvalidAuthoringTransaction
	}
	entries, err := captureHelpActionEntries(help.ReplaceEntries)
	if err != nil {
		return err
	}
	draft.declarations.actions.helpDisabled = help.Disabled
	draft.declarations.actions.helpWhenEmpty = help.WhenEmpty
	draft.declarations.actions.helpRenderer = help.Renderer
	if draft.declarations.actions.helpRenderer == nil {
		draft.declarations.actions.helpRenderer = DefaultHelp(DefaultHelpConfig{})
	}
	stageIntrinsicActionEntries(&draft.declarations, entries)
	return nil
}

func compileVersion(draft *applicationDraft, root commandRef, version VersionConfig, release ReleaseScheme) error {
	if draft == nil {
		return errInvalidAuthoringTransaction
	}
	if _, ok := draft.declarations.commands.resolve(&draft.declarations, root); !ok {
		return errInvalidAuthoringTransaction
	}
	text, present, err := intrinsicVersionText(version, release)
	if err != nil {
		return err
	}
	draft.declarations.actions.versionText = text
	draft.declarations.actions.versionPresent = present
	draft.declarations.actions.versionDisabled = version.Disabled
	draft.declarations.actions.versionWhenEmpty = version.WhenEmpty
	if present {
		entries, err := captureVersionActionEntries(version.ReplaceEntries)
		if err != nil {
			return err
		}
		stageIntrinsicActionEntries(&draft.declarations, entries)
	}
	return nil
}

func compileShellCompletion(draft *applicationDraft, root commandRef, config ShellCompletion) (shellCompletionCatalog, error) {
	if draft == nil {
		return shellCompletionCatalog{}, errInvalidAuthoringTransaction
	}
	catalog, err := snapshotShellCompletion(config)
	if err != nil {
		return shellCompletionCatalog{}, err
	}
	if err := stageShellCompletion(&draft.declarations, root, catalog); err != nil {
		return shellCompletionCatalog{}, err
	}
	return catalog, nil
}

func (draft *applicationDraft) commit() (staticProgram, declarationCallbacks, []compiledSignalSource, error) {
	if draft == nil || draft.closed {
		return staticProgram{}, declarationCallbacks{}, nil, errInvalidAuthoringTransaction
	}
	if err := draft.declarations.rejectionError(); err != nil {
		return staticProgram{}, declarationCallbacks{}, nil, err
	}
	input, err := draft.declarations.declarationPlan()
	if err != nil {
		return staticProgram{}, declarationCallbacks{}, nil, err
	}
	root, ok := draft.declarations.commands.resolve(&draft.declarations, draft.declarations.root)
	if !ok {
		return staticProgram{}, declarationCallbacks{}, nil, errInvalidAuthoringTransaction
	}
	metadata := snapshotApplicationMetadata(
		draft.name,
		draft.description,
		draft.examples,
		draft.release,
		draft.localization,
		draft.frameworkLocalization,
	)
	configuration := acceptCommandConfiguration(root.makeConfig)
	execution := acceptCommandExecution(root.middleware, root.handler)
	prepared, err := prepareStaticCommit(
		input,
		&draft.declarations,
		metadata,
		configuration,
		execution,
	)
	if err != nil {
		return staticProgram{}, declarationCallbacks{}, nil, err
	}
	accepted := prepared.program
	callbacks := prepared.callbacks
	catalog := freezeCompiledSignalCatalog(draft.signalSources)
	description, err := signalSourceDescription(catalog.sources)
	if err != nil {
		return staticProgram{}, declarationCallbacks{}, nil, err
	}
	if err := validateCompiledSignalSourcePolicies(accepted, description); err != nil {
		return staticProgram{}, declarationCallbacks{}, nil, err
	}

	// No-failure publication region: every fallible acceptance/projection check is complete.
	prepared.publishDestinations()
	prepared.shellCompletion.publishIdentity()
	draft.signalSources = slices.Clone(catalog.sources)
	draft.closed = true

	return accepted, callbacks, slices.Clone(draft.signalSources), nil
}
