package cli

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

// errInvalidShellCompletion reports an invalid compiled shell-completion catalog.
var errInvalidShellCompletion = errors.New("application: invalid shell completion catalog")

// shellCompletionProvider is one snapshotted provider in the compiled catalog.
type shellCompletionProvider struct {
	name      string
	generator ShellCompletionGenerator
}

// acceptShellCompletionProvider validates and freezes one compiled provider.
func acceptShellCompletionProvider(name string, generator ShellCompletionGenerator) shellCompletionProvider {
	if name == "" || generator == nil {
		panic("application: invalid shell completion provider")
	}
	provider := shellCompletionProvider{
		name:      name,
		generator: generator,
	}
	return provider
}

// Name returns the provider catalog name.
func (provider shellCompletionProvider) Name() string { return provider.name }

// Valid reports whether the provider is structurally usable.
func (provider shellCompletionProvider) Valid() bool {
	return provider.name != "" && provider.generator != nil
}

// Generate renders an artifact from the snapshotted provider.
func (provider shellCompletionProvider) Generate(generation ShellCompletionGeneration) (string, error) {
	if !provider.Valid() {
		return "", errInvalidShellCompletion
	}
	return provider.generator(generation)
}

// shellCompletionIdentity is the accepted identity of the synthesized shell-completion declarations.
type shellCompletionIdentity struct {
	command     CommandID
	shell       ValueID
	shellTerm   ArgumentTermID
	version     ValueID
	versionFlag FlagID
}

type shellCompletionCatalog struct {
	valid     bool
	disabled  bool
	providers []shellCompletionProvider

	identity *shellCompletionIdentity
	accepted shellCompletionIdentity
}

// freezeDisabledShellCompletionCatalog freezes a valid disabled catalog.
func freezeDisabledShellCompletionCatalog() shellCompletionCatalog {
	catalog := shellCompletionCatalog{
		valid:    true,
		disabled: true,
	}
	return catalog
}

// freezeShellCompletionCatalog snapshots a non-empty provider catalog.
func freezeShellCompletionCatalog(providers []shellCompletionProvider) shellCompletionCatalog {
	copyProviders := append([]shellCompletionProvider(nil), providers...)
	if len(copyProviders) == 0 {
		panic("application: shell completion catalog requires providers")
	}
	for _, provider := range copyProviders {
		if !provider.Valid() {
			panic("application: invalid shell completion provider")
		}
	}
	catalog := shellCompletionCatalog{
		valid:     true,
		providers: copyProviders,
		identity:  &shellCompletionIdentity{},
	}
	return catalog
}

// Valid reports whether the catalog was constructed successfully.
func (catalog shellCompletionCatalog) Valid() bool { return catalog.valid }

// Disabled reports whether the intrinsic is intentionally absent.
func (catalog shellCompletionCatalog) Disabled() bool { return catalog.valid && catalog.disabled }

// Providers returns a defensive copy of the compiled provider order.
func (catalog shellCompletionCatalog) Providers() []shellCompletionProvider {
	return append([]shellCompletionProvider(nil), catalog.providers...)
}

// Provider resolves one compiled provider by its exact catalog name.
func (catalog shellCompletionCatalog) Provider(name string) (shellCompletionProvider, bool) {
	for _, provider := range catalog.providers {
		if provider.name == name {
			return provider, true
		}
	}
	return shellCompletionProvider{}, false
}

// Selected reports whether command is the synthesized shell-completion intrinsic.
func (catalog shellCompletionCatalog) Selected(command CommandID) bool {
	identity := catalog.publishedIdentity()
	return catalog.valid && !catalog.disabled && identity.command.present() && identity.command == command
}

// ShellDestination returns the synthesized shell-provider destination.
func (catalog shellCompletionCatalog) ShellDestination() ValueID {
	return catalog.publishedIdentity().shell
}

// VersionDestination returns the synthesized target-version destination.
func (catalog shellCompletionCatalog) VersionDestination() ValueID {
	return catalog.publishedIdentity().version
}

// VersionFlag returns the synthesized target-version flag.
func (catalog shellCompletionCatalog) VersionFlag() FlagID {
	return catalog.publishedIdentity().versionFlag
}

func (catalog shellCompletionCatalog) publishedIdentity() shellCompletionIdentity {
	if catalog.identity == nil {
		return shellCompletionIdentity{}
	}

	return *catalog.identity
}

func (catalog shellCompletionCatalog) publishIdentity() {
	if !catalog.valid || catalog.disabled {
		return
	}
	if catalog.identity == nil {
		panic("cli: shell completion identity has no publication cell")
	}
	if catalog.identity.command.present() {
		panic("cli: shell completion identity already published")
	}
	if !catalog.accepted.command.present() || !catalog.accepted.shell.present() || !catalog.accepted.shellTerm.present() || !catalog.accepted.version.present() || !catalog.accepted.versionFlag.present() {
		panic("cli: shell completion identity is incomplete")
	}

	*catalog.identity = catalog.accepted
}

type shellCompletionDraft struct {
	catalog shellCompletionCatalog

	command     commandRef
	shell       destinationRef
	shellTerm   argumentTermDraftRef
	version     destinationRef
	versionFlag flagRef
	valid       bool
}

func stageShellCompletion(draft *declarationDraft, root commandRef, catalog shellCompletionCatalog) error {
	if draft == nil || !catalog.Valid() || draft.shellCompletion.valid {
		return errInvalidAuthoringTransaction
	}
	if _, ok := draft.commands.resolve(draft, root); !ok {
		return errInvalidAuthoringTransaction
	}
	staged := shellCompletionDraft{
		catalog: catalog,
		valid:   true,
	}
	if catalog.Disabled() {
		draft.shellCompletion = staged
		return nil
	}
	providers := catalog.Providers()
	if len(providers) == 0 {
		return errInvalidAuthoringTransaction
	}

	commandPosition := draft.nextPosition()
	command := commandDraft{
		position: commandPosition,
		parent: stagedCommandParent{
			command: root,
		},
		name: "completion",
		description: Description{
			Short: "Generate shell completion source",
		},
		intrinsic: true,
	}
	commandRef := draft.commands.append(draft, command)

	dialect := dialectDirectiveDraft{
		position:  draft.nextPosition(),
		command:   commandRef,
		kind:      dialectDirectiveFlagOrder,
		flagOrder: FlagOrderInterspersed,
		valid:     true,
	}
	draft.dialectDirectives.append(draft, dialect)

	domain := Value.Domain(Value.Assign[string](), func(builder *ValueDomainBuilder[string]) {
		for _, provider := range providers {
			builder.Value(provider.Name(), provider.Name())
		}
	})
	shellDefinition, shellDefinitionComplete, shellUses := scalarDestinationDefinition[string](domain)
	if !shellDefinitionComplete || !shellDefinition.Valid() {
		return errInvalidAuthoringTransaction
	}
	shellPosition := draft.nextPosition()
	shellDestination := destinationDraft{
		position:           shellPosition,
		command:            commandRef,
		kind:               destinationDeclarationValue,
		domainUses:         shellUses,
		definition:         shellDefinition,
		definitionComplete: true,
		intrinsic:          true,
		valid:              true,
	}
	shellRef := draft.destinations.append(draft, shellDestination)

	formPosition := draft.nextPosition()
	formRef := draft.argumentForms.append(draft, argumentFormDraft{
		position: formPosition,
		command:  commandRef,
		valid:    true,
	})
	termPosition := draft.nextPosition()
	shellTerm := argumentTermDraft{
		position:       termPosition,
		command:        commandRef,
		form:           formRef,
		destination:    shellRef,
		kind:           argumentTermValue,
		name:           "shell",
		count:          Count.Exactly(1),
		description:    "Shell completion provider",
		descriptionSet: true,
		valid:          true,
	}
	termRef := draft.argumentTerms.append(draft, shellTerm)

	versionDefinition, versionDefinitionComplete, versionUses := scalarDestinationDefinition[string](Value.String())
	if !versionDefinitionComplete || !versionDefinition.Valid() || len(versionUses) != 0 {
		return errInvalidAuthoringTransaction
	}
	versionPosition := draft.nextPosition()
	versionDestination := destinationDraft{
		position:           versionPosition,
		command:            commandRef,
		kind:               destinationDeclarationValue,
		definition:         versionDefinition,
		definitionComplete: true,
		intrinsic:          true,
		valid:              true,
	}
	versionRef := draft.destinations.append(draft, versionDestination)

	flagPosition := draft.nextPosition()
	versionFlag := flagDraft{
		position:       flagPosition,
		command:        commandRef,
		destination:    versionRef,
		producer:       flagProducerScalar,
		occurrences:    Count.Optional(),
		occurrencesSet: true,
		placement:      FlagPlacementCommand,
		placementSet:   true,
		valueName:      "VERSION",
		valueNameSet:   true,
		description:    "Target shell syntax version",
		descriptionSet: true,
		valid:          true,
	}
	flagRef := draft.flags.append(draft, versionFlag)
	spellingPosition := draft.nextPosition()
	versionSpelling := flagSpellingDraft{
		position: spellingPosition,
		flag:     flagRef,
		kind:     flagSpellingLong,
		text:     "--shell-version",
		valid:    true,
	}
	draft.flagSpellings.append(draft, versionSpelling)

	staged.command = commandRef
	staged.shell = shellRef
	staged.shellTerm = termRef
	staged.version = versionRef
	staged.versionFlag = flagRef
	draft.shellCompletion = staged
	return nil
}

func acceptedShellCompletion(input declarationPlan, source *declarationDraft, state *staticAcceptanceState) (shellCompletionCatalog, bool) {
	if source == nil || state == nil || !source.shellCompletion.valid || !source.shellCompletion.catalog.Valid() {
		return shellCompletionCatalog{}, false
	}
	staged := source.shellCompletion
	catalog := staged.catalog
	if catalog.Disabled() {
		return catalog, true
	}
	commandIndex, commandOK := input.indexes.commands[staged.command]
	shellIndex, shellOK := input.indexes.destinations[staged.shell]
	termIndex, termOK := input.indexes.argumentTerms[staged.shellTerm]
	versionIndex, versionOK := input.indexes.destinations[staged.version]
	flagIndex, flagOK := input.indexes.flags[staged.versionFlag]
	if !commandOK || !shellOK || !termOK || !versionOK || !flagOK {
		return shellCompletionCatalog{}, false
	}
	command, commandOK := state.command(commandIndex)
	shell, shellOK := state.destination(shellIndex)
	termSlot, termOK := state.slot(termIndex)
	version, versionOK := state.destination(versionIndex)
	flagSlot, flagOK := state.slot(flagIndex)
	if !commandOK || !shellOK || !termOK || !versionOK || !flagOK ||
		!termSlot.argumentTerm.ID().present() || !flagSlot.flag.ID().present() {
		return shellCompletionCatalog{}, false
	}
	catalog.accepted = shellCompletionIdentity{
		command:     command.ID(),
		shell:       shell.ID(),
		shellTerm:   termSlot.argumentTerm.ID(),
		version:     version.ID(),
		versionFlag: flagSlot.flag.ID(),
	}
	return catalog, true
}

// ShellFamily identifies the runtime shell family participating in Completion. Built-in
// integrations use the six well-known constants below; custom semantic clients may use another
// nonempty family name without creating a second completion protocol.
type ShellFamily string

const (
	ShellFamilyBash       ShellFamily = "bash"
	ShellFamilyZsh        ShellFamily = "zsh"
	ShellFamilyFish       ShellFamily = "fish"
	ShellFamilyPowerShell ShellFamily = "powershell"
	ShellFamilyElvish     ShellFamily = "elvish"
	ShellFamilyNushell    ShellFamily = "nushell"
)

// ShellCompletion configures the Definition-owned public shell-completion exporter.
// The zero value enables the six built-in providers in canonical order. Disabled is
// the sole disabled representation and requires Shells to be nil.
type ShellCompletion struct {
	Disabled bool
	Shells   []ShellCompletionProvider
}

// ShellCompletionProvider is one immutable shell artifact provider declaration.
// Built-ins are used as method values such as cli.Shell.Bash; custom providers are
// created with cli.Shell.Custom.
type ShellCompletionProvider func() shellCompletionDescriptor

// ShellCompletionGenerator generates one shell completion integration source artifact.
type ShellCompletionGenerator func(ShellCompletionGeneration) (string, error)

// ShellCompletionGeneration is the complete public generation context available
// to built-in and custom shell providers.
type ShellCompletionGeneration struct {
	ApplicationName       string
	TargetVersion         string
	ActivationEnvironment string
	ActivationValue       string
}

type shellCompletionDescriptor struct {
	name             string
	generator        ShellCompletionGenerator
	family           ShellFamily
	versionSensitive bool
}

func (descriptor shellCompletionDescriptor) valid() bool {
	return validShellCompletionProviderName(descriptor.name) && descriptor.generator != nil
}

type shellNamespace struct {
	_ [0]func()
}

// Shell is the sealed namespace of built-in shell-completion providers.
var Shell shellNamespace

func (shellNamespace) Bash() shellCompletionDescriptor {
	return builtInShellCompletionDescriptor(ShellFamilyBash, false)
}

func (shellNamespace) Zsh() shellCompletionDescriptor {
	return builtInShellCompletionDescriptor(ShellFamilyZsh, false)
}

func (shellNamespace) Fish() shellCompletionDescriptor {
	return builtInShellCompletionDescriptor(ShellFamilyFish, false)
}

func (shellNamespace) PowerShell() shellCompletionDescriptor {
	return builtInShellCompletionDescriptor(ShellFamilyPowerShell, false)
}

func (shellNamespace) Elvish() shellCompletionDescriptor {
	return builtInShellCompletionDescriptor(ShellFamilyElvish, true)
}

func (shellNamespace) Nushell() shellCompletionDescriptor {
	return builtInShellCompletionDescriptor(ShellFamilyNushell, true)
}

// Custom returns a provider for name backed by generator. Invalid names or a nil
// generator produce an invalid provider that Compile/direct generation rejects.
func (shellNamespace) Custom(name string, generator ShellCompletionGenerator) ShellCompletionProvider {
	if !validShellCompletionProviderName(name) || generator == nil {
		return func() shellCompletionDescriptor { return shellCompletionDescriptor{} }
	}
	return func() shellCompletionDescriptor {
		descriptor := shellCompletionDescriptor{
			name:      name,
			generator: generator,
		}
		return descriptor
	}
}

func builtInShellCompletionDescriptor(family ShellFamily, versionSensitive bool) shellCompletionDescriptor {
	descriptor := shellCompletionDescriptor{
		name:             string(family),
		family:           family,
		versionSensitive: versionSensitive,
	}
	descriptor.generator = func(generation ShellCompletionGeneration) (string, error) {
		return generateBuiltInShellCompletionScript(family, generation)
	}
	return descriptor
}

func defaultShellCompletionProviders() []ShellCompletionProvider {
	return []ShellCompletionProvider{
		Shell.Bash,
		Shell.Zsh,
		Shell.Fish,
		Shell.PowerShell,
		Shell.Elvish,
		Shell.Nushell,
	}
}

func validShellCompletionProviderName(name string) bool {
	if name == "" || !utf8.ValidString(name) || name[0] == '-' {
		return false
	}
	for _, r := range name {
		if r == 0 || r == '\r' || r == '\n' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func snapshotShellCompletion(config ShellCompletion) (shellCompletionCatalog, error) {
	if config.Disabled {
		if config.Shells != nil {
			return shellCompletionCatalog{}, errInvalidShellCompletionProvider
		}
		return freezeDisabledShellCompletionCatalog(), nil
	}
	providers := config.Shells
	if providers == nil {
		providers = defaultShellCompletionProviders()
	} else if len(providers) == 0 {
		return shellCompletionCatalog{}, errInvalidShellCompletionProvider
	}
	compiled := make([]shellCompletionProvider, 0, len(providers))
	seen := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		if provider == nil {
			return shellCompletionCatalog{}, errInvalidShellCompletionProvider
		}
		descriptor := provider()
		if !descriptor.valid() {
			return shellCompletionCatalog{}, errInvalidShellCompletionProvider
		}
		if _, exists := seen[descriptor.name]; exists {
			return shellCompletionCatalog{}, errInvalidShellCompletionProvider
		}
		seen[descriptor.name] = struct{}{}
		compiled = append(compiled, acceptShellCompletionProvider(descriptor.name, descriptor.generator))
	}
	return freezeShellCompletionCatalog(compiled), nil
}
