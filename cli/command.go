package cli

// CommandAlias describes one lexical alternate spelling for its owning canonical command. It has
// no independent command identity or behavior. A zero Description inherits canonical presentation;
// Hidden and Release may narrow the alias without broadening the canonical command.
type CommandAlias struct {
	Name string

	Description Description

	Hidden bool

	Release ReleaseLifecycle
}

// CommandAliases is an ordered collection of lexical aliases for one canonical command.
type CommandAliases []CommandAlias

// Aliases returns a CommandAliases value containing names in caller order.
func Aliases(names ...string) CommandAliases {
	aliases := make(CommandAliases, len(names))
	for i, name := range names {
		aliases[i] = CommandAlias{
			Name: name,
		}
	}
	return aliases
}

// CommandBuilder is the callback-scoped declaration surface for one command. Config points at the
// command configuration instance, and the capability fields stage declarations for that command.
// Its zero value is invalid; retaining the builder or a capability derived from it, using one
// concurrently, or using one after the declaration callback returns is builder lifecycle misuse.
type CommandBuilder[C any] struct {
	Config *C

	Access      accessCapability[C]
	Dialect     Dialect
	Flags       Flags[C]
	Facts       Facts[C]
	Arguments   Arguments[C]
	Values      Values[C]
	When        Conditions
	Rule        ruleCapability[C]
	Advisory    advisoryCapability[C]
	Notice      noticeCapability[C]
	Profiles    GrammarProfiles[C]
	Signals     Signals[C]
	Subcommands Subcommands[C]

	session *declarationSession
	profile grammarProfileCaseRef
}

// AvailableWhen makes the command structurally available only when condition holds.
func (command *CommandBuilder[C]) AvailableWhen(condition Condition) *CommandBuilder[C] {
	if command == nil {
		panicBuilderMisuse()
	}
	stageAvailabilityCondition(command.session, command.profile, authoringCommandConditionTarget{
		reference: command.session.command,
	}, condition)
	return command
}

// VisibleWhen makes the command visible only when condition holds.
func (command *CommandBuilder[C]) VisibleWhen(condition Condition) *CommandBuilder[C] {
	if command == nil {
		panicBuilderMisuse()
	}
	stageVisibilityCondition(command.session, command.profile, authoringCommandConditionTarget{
		reference: command.session.command,
	}, condition)
	return command
}

// Shortcut declares a static shortcut in the current command scope and returns its declaration for
// defaults or replacement.
func (command *CommandBuilder[C]) Shortcut(shortcut CommandShortcut) CommandShortcutDeclaration[C] {
	return declareCommandShortcut(command, shortcut)
}

type commandDeclaration interface {
	commandDeclaration()
	declareInto(session *declarationSession)
}

// Commands is a heterogeneous list of command declarations that can be added to a Subcommands
// surface in order.
type Commands []commandDeclaration

func (Command[C]) commandDeclaration() {}

func (command Command[C]) declareInto(session *declarationSession) {
	if session == nil {
		return
	}
	subcommands := Subcommands[struct{}]{
		session: session,
	}
	declareSubcommand(&subcommands, command)
}

// Subcommands declares child commands, dynamic command providers, and subcommand help groups for
// one command. It is a callback-scoped declaration capability; its zero value and retained,
// concurrent, or post-callback use are invalid.
type Subcommands[C any] struct {
	Config *C

	session   *declarationSession
	profile   grammarProfileCaseRef
	helpGroup helpGroupRef
}

// Command declares one child command and returns its declaration handle in the owning parent
// declaration scope.
func (subcommands *Subcommands[C]) Command[T any](cmd Command[T]) CommandDeclaration[C] {
	return declareSubcommand(subcommands, cmd)
}

// Commands declares each non-nil command in the slice in slice order.
func (subcommands *Subcommands[C]) Commands(commands Commands) {
	if subcommands == nil || subcommands.session == nil {
		panicBuilderMisuse()
	}
	for _, command := range commands {
		if command == nil {
			continue
		}
		command.declareInto(subcommands.session)
	}
}

// CommandsFrom declares a dynamic command provider evaluated with the supplied dependencies.
func (subcommands *Subcommands[C]) CommandsFrom(provider CommandProvider[C], dependencies ...Dependency) {
	declareCommandProvider(subcommands, provider, dependencies)
}

// HelpGroup declares a titled group of subcommands through the callback.
func (subcommands *Subcommands[C]) HelpGroup(title string, declare func(*Subcommands[C])) helpGroupDeclaration {
	return declareSubcommandHelpGroup(subcommands, title, declare)
}

// CommandTarget is the set of values that can identify a command when authoring aliases,
// shortcuts, and related contributions.
type CommandTarget interface {
	commandTarget()
}

// CommandPath names a command target by an ordered root-relative sequence of command names, from
// the root toward the target.
type CommandPath []string

func (CommandPath) commandTarget() {}

func commandBuilder[C any](config *C, session *declarationSession, profile grammarProfileCaseRef) *CommandBuilder[C] {
	var profileSession *declarationSession
	if !profile.valid() {
		profileSession = session
	}

	argumentAuthoring := &argumentAuthoringState{}

	builder := CommandBuilder[C]{
		Config: config,
		Access: accessCapability[C]{
			config:  config,
			session: session,
			profile: profile,
		},
		Dialect: Dialect{
			session: session,
			profile: profile,
		},
		Flags: Flags[C]{
			Config:  config,
			session: session,
			profile: profile,
		},
		Facts: Facts[C]{
			Config:  config,
			session: session,
			profile: profile,
		},
		Arguments: Arguments[C]{
			Config:    config,
			session:   session,
			profile:   profile,
			authoring: argumentAuthoring,
		},
		Values: Values[C]{
			Config:  config,
			session: session,
			profile: profile,
		},
		When: Conditions{
			session: session,
			profile: profile,
		},
		Rule: ruleCapability[C]{
			ruleScope: ruleScope[C]{
				session: session,
				profile: profile,
				Compare: ruleComparisons[C]{
					session: session,
					profile: profile,
				},
			},
		},
		Advisory: advisoryCapability[C]{
			session: session,
			profile: profile,
		},
		Notice: noticeCapability[C]{
			session: session,
			profile: profile,
		},
		Profiles: GrammarProfiles[C]{
			Config:  config,
			session: profileSession,
		},
		Signals: Signals[C]{
			Config: config,
			Match: signalMatches[C]{
				session: session,
				profile: profile,
			},
			Action: signalActions[C]{
				config:  config,
				session: session,
				profile: profile,
			},
			session: session,
			profile: profile,
		},
		Subcommands: Subcommands[C]{
			Config:  config,
			session: session,
			profile: profile,
		},
		session: session,
		profile: profile,
	}

	return &builder
}

func declareSubcommand[C, T any](subcommands *Subcommands[C], command Command[T]) CommandDeclaration[C] {
	if subcommands == nil || subcommands.session == nil {
		panicBuilderMisuse()
	}

	draft, parentCommand, resume := subcommands.session.suspend()
	defer resume()

	contribution := draft.beginContribution()
	defer contribution.rollback()

	parent := stagedCommandParent{
		command: parentCommand,
	}
	child := stageCommand(draft, parent, command)
	staged := draft.commands.mustResolve(draft, child)
	staged.profile = subcommands.profile
	staged.helpGroup = subcommands.helpGroup
	declaration := CommandDeclaration[C]{
		session: subcommands.session,
		command: child,
	}

	if command.Declare != nil {
		childSession := subcommands.session.prepareChild(child)
		builder := commandBuilder(new(T), childSession, subcommands.profile)
		if cause := childSession.invoke(func() {
			command.Declare(builder)
		}); cause != nil {
			abortBuilder(cause)
		}
	}

	if contribution.rejected() {
		declaration.command = commandRef{}
		return declaration
	}

	contribution.commit()
	return declaration
}

// CommandDeclaration identifies one staged command entry and may be used as a command target or
// replacement source. C is the configuration type of the owning parent declaration scope, not the
// child command. It is callback-scoped: the zero value is invalid, copies identify the same
// declaration, and retained, concurrent, or cross-scope use is builder lifecycle misuse.
type CommandDeclaration[C any] struct {
	session *declarationSession
	command commandRef
}

func (CommandDeclaration[C]) commandTarget() {}

type commandDefaultState uint8

const (
	commandDefaultUnspecified commandDefaultState = iota
	commandDefaultUnconditional
	commandDefaultConditional
)

type commandDefaultDraft struct {
	condition Condition
	state     commandDefaultState
}

// Default declares this exact staged child as an unconditional default of its owning parent scope
// and returns the same declaration.
func (declaration CommandDeclaration[C]) Default() CommandDeclaration[C] {
	value := commandDefaultDraft{
		state: commandDefaultUnconditional,
	}
	return stageCommandDefault(declaration, value)
}

// DefaultWhen declares this exact staged child as a default of its owning parent scope when
// condition holds and returns the same declaration.
func (declaration CommandDeclaration[C]) DefaultWhen(condition Condition) CommandDeclaration[C] {
	value := commandDefaultDraft{
		condition: condition,
		state:     commandDefaultConditional,
	}
	return stageCommandDefault(declaration, value)
}

func stageCommandDefault[C any](declaration CommandDeclaration[C], value commandDefaultDraft) CommandDeclaration[C] {
	if !declaration.command.valid() {
		return declaration
	}
	declaration.session.use(func(draft *declarationDraft, _ commandRef) {
		command, ok := draft.commands.resolve(draft, declaration.command)
		if !ok || command.defaults.state != commandDefaultUnspecified {
			draft.reject(draft.nextPosition(), errInvalidAuthoringTransaction)
			return
		}
		command.defaults = value
	})
	return declaration
}

// AvailableWhen makes the command declaration structurally available only when condition holds.
func (declaration CommandDeclaration[C]) AvailableWhen(condition Condition) CommandDeclaration[C] {
	if !declaration.command.valid() {
		return declaration
	}
	stageAvailabilityCondition(declaration.session, grammarProfileCaseRef{}, authoringCommandConditionTarget{
		reference: declaration.command,
	}, condition)
	return declaration
}

// VisibleWhen makes the command declaration visible only when condition holds.
func (declaration CommandDeclaration[C]) VisibleWhen(condition Condition) CommandDeclaration[C] {
	if !declaration.command.valid() {
		return declaration
	}
	stageVisibilityCondition(declaration.session, grammarProfileCaseRef{}, authoringCommandConditionTarget{
		reference: declaration.command,
	}, condition)
	return declaration
}

func stageStaticCommandIdentity(input declarationPlan, source *declarationDraft, draft *declarationCandidate, state *staticAcceptanceState) error {
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind == declarationRecordShortcut {
			if record.commandIndex < 0 || record.commandIndex >= len(input.records) {
				panic("cli: validated shortcut has invalid command index")
			}
			target := state.mustCommand(record.commandIndex)
			commandRecord := input.records[record.commandIndex]
			if commandRecord.kind != declarationRecordCommand {
				panic("cli: validated shortcut owner is not a command")
			}
			parent := state.mustCommand(commandRecord.commandIndex)
			position := input.positionFor(index)
			entry := draft.CommandEntry(parent, target, position)
			draft.CommandEntryGrammar(
				entry,
				CommandEntryShortcut,
				source.shortcuts.entries[record.familyIndex].value.name,
				true,
			)
			draft.CommandEntryPresentation(
				entry,
				semanticPresentation(
					source.shortcuts.entries[record.familyIndex].value.description,
					source.shortcuts.entries[record.familyIndex].value.examples,
					source.shortcuts.entries[record.familyIndex].value.hidden,
				),
				position,
			)
			draft.CommandEntryLifecycle(
				entry,
				source.shortcuts.entries[record.familyIndex].value.release,
				position,
			)
			state.slots[index].shortcutEntry = entry
			state.slots[record.commandIndex].commandEntries = append(
				state.slots[record.commandIndex].commandEntries,
				entry,
			)
			continue
		}
		if record.kind != declarationRecordCommand {
			continue
		}
		parent := state.mustCommand(record.commandIndex)
		position := input.positionFor(index)
		value := source.commands.entries[record.familyIndex].value
		configuration := semanticCommandConfiguration{}
		if value.makeConfig != nil {
			configuration = acceptCommandConfiguration(value.makeConfig)
		}
		command := draft.Command(parent, position, configuration)
		state.slots[index].command = command
		commandPresentation := semanticPresentation(
			value.description,
			value.examples,
			value.hidden,
		)
		draft.CommandPresentation(command, commandPresentation, position)
		entry := draft.CommandEntry(parent, command, position)
		draft.CommandEntryGrammar(
			entry,
			CommandEntryCanonical,
			value.name,
			true,
		)
		draft.CommandEntryPresentation(entry, commandPresentation, position)
		draft.CommandEntryLifecycle(
			entry,
			value.release,
			position,
		)
		state.slots[index].commandEntries = append(state.slots[index].commandEntries, entry)
		for _, alias := range value.aliases {
			aliasEntry := draft.CommandEntry(parent, command, position)
			draft.CommandEntryGrammar(aliasEntry, CommandEntryAlias, alias.Name, true)
			draft.CommandEntryPresentation(
				aliasEntry,
				semanticPresentation(alias.Description, nil, alias.Hidden),
				position,
			)
			draft.CommandEntryLifecycle(aliasEntry, alias.Release, position)
			state.slots[index].commandEntries = append(state.slots[index].commandEntries, aliasEntry)
		}
		switch value.defaults.state {
		case commandDefaultUnspecified:
		case commandDefaultUnconditional, commandDefaultConditional:
			state.slots[index].defaultRef = draft.Default(parent, command, position)
		default:
			return errStaticDeclarationRejected
		}
	}
	return nil
}

// commandEntryGrammar contains recognition facts only. Canonical topology and
// identity remain owned by CommandEntryRecord.
type commandEntryGrammar struct {
	id       CommandEntryID
	parent   CommandID
	target   CommandID
	kind     CommandEntryKind
	name     string
	profile  GrammarProfileID
	position declarationPosition
	enabled  bool
}

// commandEntryGrammarInput is the complete immutable payload of one command-entry grammar record.
type commandEntryGrammarInput struct {
	ID       CommandEntryID
	Parent   CommandID
	Target   CommandID
	Kind     CommandEntryKind
	Name     string
	Position declarationPosition
	Enabled  bool
}

func validCommandEntryGrammarInput(input commandEntryGrammarInput) bool {
	return input.ID.present() && input.Parent.present() && input.Target.present() && input.Kind != CommandEntryUnknown &&
		input.Name != ""
}

func acceptCommandEntryGrammar(input commandEntryGrammarInput) commandEntryGrammar {
	if !validCommandEntryGrammarInput(input) {
		panic("semantic: invalid command-entry grammar")
	}
	grammar := commandEntryGrammar{
		id:       input.ID,
		parent:   input.Parent,
		target:   input.Target,
		kind:     input.Kind,
		name:     input.Name,
		position: input.Position,
		enabled:  input.Enabled,
	}

	return grammar
}

type staticCommandParent struct {
	id      CommandID
	present bool
}

// commandRecord is the canonical static identity/topology record for one
// command. Product-specific grammar, policy, presentation, and execution facts
// attach to the same command identity in later semantic partitions.
type commandRecord struct {
	meta          recordMeta[CommandID]
	parent        staticCommandParent
	configuration semanticCommandConfiguration
	presentation  presentation
	execution     commandExecution
	access        executionAccess
}

func acceptRootCommandRecord(id CommandID, position declarationPosition, configuration semanticCommandConfiguration) commandRecord {
	if !configuration.Valid() {
		panic("semantic: invalid command configuration")
	}
	return commandRecord{
		meta:          acceptRecordMeta(id, position),
		configuration: configuration,
	}
}

func acceptCommandRecord(id, parent CommandID, position declarationPosition, configuration semanticCommandConfiguration) commandRecord {
	if !parent.present() {
		panic("semantic: invalid parent command identity")
	}
	return commandRecord{
		meta: acceptRecordMeta(id, position),
		parent: staticCommandParent{
			id:      parent,
			present: true,
		},
		configuration: configuration,
	}
}

func (record commandRecord) Parent() (CommandID, bool) {
	return record.parent.id, record.parent.present
}

func (record commandRecord) Configuration() (semanticCommandConfiguration, bool) {
	return record.configuration, record.configuration.Valid()
}

type commandEntryRecord struct {
	meta         recordMeta[CommandEntryID]
	parent       CommandID
	target       CommandID
	presentation presentation
	lifecycle    ReleaseLifecycle
	shortcut     *shortcutDefinition
	replacement  CommandEntryID
}

func acceptCommandEntryRecord(id CommandEntryID, parent, target CommandID, position declarationPosition) commandEntryRecord {
	mustCommand(parent)
	mustCommand(target)
	record := commandEntryRecord{
		meta:   acceptRecordMeta(id, position),
		parent: parent,
		target: target,
	}

	return record
}

func (record commandEntryRecord) Shortcut() (shortcutDefinition, bool) {
	if record.shortcut == nil {
		return shortcutDefinition{}, false
	}
	return acceptShortcutDefinition(
		record.shortcut.Args(),
		record.shortcut.AppendArgs(),
		record.shortcut.Defaults(),
		record.shortcut.Source(),
	), true
}

func (record commandEntryRecord) Replacement() (CommandEntryID, bool) {
	return record.replacement, record.replacement.present()
}

// Command describes one child command before compilation. During compilation, Declare receives a
// fresh callback-scoped CommandBuilder whose Config belongs to that command. Name, aliases,
// presentation, release state, middleware, and Run belong to the reusable command declaration; an
// empty Name is invalid. Parent-relative default selection is authored on CommandDeclaration.
type Command[C any] struct {
	Name    string
	Aliases CommandAliases

	Description Description
	Examples    []Example

	Hidden  bool
	Release ReleaseLifecycle

	Declare func(*CommandBuilder[C])

	Middleware []Middleware[C]
	Run        Handler[C]
}
