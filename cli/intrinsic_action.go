package cli

import (
	"context"
	"strings"
	"sync"
)

// HelpConfig configures the built-in Help action. Its zero value supplies standard Help entries, with
// --help and -h authored for subtree placement, the default renderer, and automatic structural Help;
// disabling it removes the CLI routes without removing direct structured Help/rendering capability.
type HelpConfig struct {
	Disabled  bool
	WhenEmpty bool
	Renderer  HelpRenderer

	ReplaceEntries func(*HelpActionEntries)
}

// VersionConfig configures the built-in Version action. Definition.Release is the sole release/version
// authority; Commit optionally adds single-line build metadata and CommitLength limits its displayed
// Unicode-code-point prefix.
type VersionConfig struct {
	Disabled     bool
	WhenEmpty    bool
	Commit       string
	CommitLength int

	ReplaceEntries func(*VersionActionEntries)
}

type actionFlag struct {
	long        string
	short       rune
	literal     string
	placement   FlagPlacement
	description Description
	hidden      bool
	release     ReleaseLifecycle
}

type actionCommand struct {
	name        string
	description Description
	hidden      bool
	release     ReleaseLifecycle
}

// HelpActionEntries declares the replacement entry set for the built-in Help action. Entries are staged
// in callback order; returned private handles own their entry-local presentation/lifecycle methods.
// The zero value and retained, concurrent, or post-callback use are invalid.
type HelpActionEntries struct{ session *actionEntrySession }

// VersionActionEntries declares the replacement entry set for the built-in Version action. Help and
// Version entry target families are intentionally distinct so cross-action replacement cannot type
// check.
type VersionActionEntries struct{ session *actionEntrySession }

type helpEntryTarget interface {
	helpEntryTarget() *actionEntryDeclarationState
}

type (
	helpFlagEntry    struct{ state *actionEntryDeclarationState }
	helpCommandEntry struct{ state *actionEntryDeclarationState }
)

type versionEntryTarget interface {
	versionEntryTarget() *actionEntryDeclarationState
}

type (
	versionFlagEntry    struct{ state *actionEntryDeclarationState }
	versionCommandEntry struct{ state *actionEntryDeclarationState }
)

func (entries *HelpActionEntries) Long(name string) helpFlagEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	flag := actionFlag{
		long: name,
	}
	entry := helpFlagEntry{
		state: declareActionFlag(entries.session, flag),
	}
	return entry
}

func (entries *HelpActionEntries) Short(name rune) helpFlagEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	flag := actionFlag{
		short: name,
	}
	entry := helpFlagEntry{
		state: declareActionFlag(entries.session, flag),
	}
	return entry
}

func (entries *HelpActionEntries) Literal(spelling string) helpFlagEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	flag := actionFlag{
		literal: spelling,
	}
	entry := helpFlagEntry{
		state: declareActionFlag(entries.session, flag),
	}
	return entry
}

func (entries *HelpActionEntries) Command(name string) helpCommandEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	command := actionCommand{
		name: name,
	}
	entry := helpCommandEntry{
		state: declareActionCommand(entries.session, command),
	}
	return entry
}

func (entry helpFlagEntry) helpEntryTarget() *actionEntryDeclarationState { return entry.state }

func (entry helpCommandEntry) helpEntryTarget() *actionEntryDeclarationState { return entry.state }

func (entry versionFlagEntry) versionEntryTarget() *actionEntryDeclarationState { return entry.state }

func (entry versionCommandEntry) versionEntryTarget() *actionEntryDeclarationState {
	return entry.state
}

func (entry helpFlagEntry) Hidden() helpFlagEntry {
	setActionFlagHidden(entry.state)
	return entry
}

func (entry helpFlagEntry) Description(description Description) helpFlagEntry {
	setActionFlagDescription(entry.state, description)
	return entry
}

func (entry helpFlagEntry) Placement(placement FlagPlacement) helpFlagEntry {
	setActionFlagPlacement(entry.state, placement)
	return entry
}

func (entry helpFlagEntry) Release(release ReleaseLifecycle) helpFlagEntry {
	setActionFlagRelease(entry.state, release)
	return entry
}

func (entry helpFlagEntry) ReplacedBy(target helpEntryTarget) helpFlagEntry {
	replaceActionEntry(entry.state, actionTargetState(target))
	return entry
}

func (entry helpCommandEntry) Hidden() helpCommandEntry {
	setActionCommandHidden(entry.state)
	return entry
}

func (entry helpCommandEntry) Description(description Description) helpCommandEntry {
	setActionCommandDescription(entry.state, description)
	return entry
}

func (entry helpCommandEntry) Release(release ReleaseLifecycle) helpCommandEntry {
	setActionCommandRelease(entry.state, release)
	return entry
}

func (entry helpCommandEntry) ReplacedBy(target helpEntryTarget) helpCommandEntry {
	replaceActionEntry(entry.state, actionTargetState(target))
	return entry
}

func (entries *VersionActionEntries) Long(name string) versionFlagEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	flag := actionFlag{
		long: name,
	}
	entry := versionFlagEntry{
		state: declareActionFlag(entries.session, flag),
	}
	return entry
}

func (entries *VersionActionEntries) Short(name rune) versionFlagEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	flag := actionFlag{
		short: name,
	}
	entry := versionFlagEntry{
		state: declareActionFlag(entries.session, flag),
	}
	return entry
}

func (entries *VersionActionEntries) Literal(spelling string) versionFlagEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	flag := actionFlag{
		literal: spelling,
	}
	entry := versionFlagEntry{
		state: declareActionFlag(entries.session, flag),
	}
	return entry
}

func (entries *VersionActionEntries) Command(name string) versionCommandEntry {
	if entries == nil {
		panicBuilderMisuse()
	}
	command := actionCommand{
		name: name,
	}
	entry := versionCommandEntry{
		state: declareActionCommand(entries.session, command),
	}
	return entry
}

func (entry versionFlagEntry) Hidden() versionFlagEntry {
	setActionFlagHidden(entry.state)
	return entry
}

func (entry versionFlagEntry) Description(description Description) versionFlagEntry {
	setActionFlagDescription(entry.state, description)
	return entry
}

func (entry versionFlagEntry) Placement(placement FlagPlacement) versionFlagEntry {
	setActionFlagPlacement(entry.state, placement)
	return entry
}

func (entry versionFlagEntry) Release(release ReleaseLifecycle) versionFlagEntry {
	setActionFlagRelease(entry.state, release)
	return entry
}

func (entry versionFlagEntry) ReplacedBy(target versionEntryTarget) versionFlagEntry {
	replaceActionEntry(entry.state, versionActionTargetState(target))
	return entry
}

func (entry versionCommandEntry) Hidden() versionCommandEntry {
	setActionCommandHidden(entry.state)
	return entry
}

func (entry versionCommandEntry) Description(description Description) versionCommandEntry {
	setActionCommandDescription(entry.state, description)
	return entry
}

func (entry versionCommandEntry) Release(release ReleaseLifecycle) versionCommandEntry {
	setActionCommandRelease(entry.state, release)
	return entry
}

func (entry versionCommandEntry) ReplacedBy(target versionEntryTarget) versionCommandEntry {
	replaceActionEntry(entry.state, versionActionTargetState(target))
	return entry
}

func actionTargetState(target helpEntryTarget) *actionEntryDeclarationState {
	if target == nil {
		return nil
	}
	return target.helpEntryTarget()
}

func versionActionTargetState(target versionEntryTarget) *actionEntryDeclarationState {
	if target == nil {
		return nil
	}
	return target.versionEntryTarget()
}

type authoringActionEntryKind uint8

const (
	authoringActionEntryInvalid authoringActionEntryKind = iota
	authoringActionEntryFlag
	authoringActionEntryCommand
)

type authoringActionEntry struct {
	kind        authoringActionEntryKind
	action      intrinsicAction
	origin      DeclarationOrigin
	flag        actionFlag
	command     actionCommand
	replacement int
	position    declarationPosition
}

type actionEntryDeclarationScope struct {
	action  intrinsicAction
	entries []authoringActionEntry
	valid   bool
}

type actionEntrySessionState uint8

const (
	actionEntrySessionPrepared actionEntrySessionState = iota
	actionEntrySessionActive
	actionEntrySessionClosed
)

type actionEntrySession struct {
	mu   sync.Mutex
	uses sync.WaitGroup

	draft *actionEntryDeclarationScope

	state actionEntrySessionState
	inUse bool
}

func (session *actionEntrySession) activate() {
	if session == nil {
		panic("cli: nil action-entry session")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state != actionEntrySessionPrepared {
		panic("cli: invalid action-entry session transition")
	}
	if session.draft == nil {
		panic("cli: prepared action-entry session has no draft")
	}
	session.state = actionEntrySessionActive
}

func (session *actionEntrySession) close() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	retainedUse := session.inUse
	session.state = actionEntrySessionClosed
	session.mu.Unlock()

	session.uses.Wait()

	session.mu.Lock()
	session.inUse = false
	session.draft = nil
	session.mu.Unlock()

	return retainedUse
}

func (session *actionEntrySession) invoke(callback func()) (cause error) {
	session.activate()
	defer func() {
		recovered := recover()
		retainedUse := session.close()

		switch recovered := recovered.(type) {
		case nil:
		case builderAbort:
			cause = recovered.cause
		default:
			panic(recovered)
		}

		if cause == nil && retainedUse {
			cause = errBuilderMisuse
		}
	}()

	callback()
	return nil
}

func (session *actionEntrySession) acquire() (*actionEntryDeclarationScope, func()) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	if session.state != actionEntrySessionActive || session.inUse {
		session.mu.Unlock()
		panicBuilderMisuse()
	}
	if session.draft == nil {
		session.mu.Unlock()
		panic("cli: active action-entry session has no draft")
	}
	session.inUse = true
	session.uses.Add(1)
	draft := session.draft
	session.mu.Unlock()

	release := func() {
		session.mu.Lock()
		session.inUse = false
		session.mu.Unlock()
		session.uses.Done()
	}

	return draft, release
}

func (session *actionEntrySession) use(callback func(*actionEntryDeclarationScope)) {
	draft, release := session.acquire()
	defer release()
	callback(draft)
}

type actionEntryField uint8

const (
	actionEntryFieldHidden actionEntryField = iota + 1
	actionEntryFieldDescription
	actionEntryFieldPlacement
	actionEntryFieldRelease
	actionEntryFieldReplacement
)

type actionEntryDeclarationState struct {
	session    *actionEntrySession
	index      int
	configured map[actionEntryField]struct{}
}

type authoringIntrinsicActions struct {
	entries []authoringActionEntry

	helpDisabled  bool
	helpWhenEmpty bool
	helpRenderer  HelpRenderer

	versionText      string
	versionPresent   bool
	versionDisabled  bool
	versionWhenEmpty bool
}

func declareActionFlag(session *actionEntrySession, flag actionFlag) *actionEntryDeclarationState {
	if session == nil {
		panicBuilderMisuse()
	}
	var state *actionEntryDeclarationState
	session.use(func(draft *actionEntryDeclarationScope) {
		if !draft.valid || !validActionFlag(flag) {
			draft.valid = false
			return
		}
		index := len(draft.entries)
		entry := authoringActionEntry{
			kind:        authoringActionEntryFlag,
			action:      draft.action,
			origin:      DeclarationOriginApplication,
			flag:        flag,
			replacement: -1,
		}
		draft.entries = append(draft.entries, entry)
		state = &actionEntryDeclarationState{
			session:    session,
			index:      index,
			configured: map[actionEntryField]struct{}{},
		}
	})
	return state
}

func declareActionCommand(session *actionEntrySession, command actionCommand) *actionEntryDeclarationState {
	if session == nil {
		panicBuilderMisuse()
	}
	var state *actionEntryDeclarationState
	session.use(func(draft *actionEntryDeclarationScope) {
		if !draft.valid || !validActionCommand(command) {
			draft.valid = false
			return
		}
		index := len(draft.entries)
		entry := authoringActionEntry{
			kind:        authoringActionEntryCommand,
			action:      draft.action,
			origin:      DeclarationOriginApplication,
			command:     command,
			replacement: -1,
		}
		draft.entries = append(draft.entries, entry)
		state = &actionEntryDeclarationState{
			session:    session,
			index:      index,
			configured: map[actionEntryField]struct{}{},
		}
	})
	return state
}

func mutateActionEntry(state *actionEntryDeclarationState, field actionEntryField, mutate func(*authoringActionEntry) bool) {
	if state == nil {
		return
	}
	if state.session == nil {
		panicBuilderMisuse()
	}
	state.session.use(func(draft *actionEntryDeclarationScope) {
		if !draft.valid || state.index < 0 || state.index >= len(draft.entries) {
			return
		}
		if _, duplicate := state.configured[field]; duplicate {
			draft.valid = false
			return
		}
		entry := draft.entries[state.index]
		if !mutate(&entry) {
			draft.valid = false
			return
		}
		state.configured[field] = struct{}{}
		draft.entries[state.index] = entry
	})
}

func setActionFlagHidden(state *actionEntryDeclarationState) {
	mutateActionEntry(state, actionEntryFieldHidden, func(entry *authoringActionEntry) bool {
		if entry.kind != authoringActionEntryFlag {
			return false
		}
		entry.flag.hidden = true
		return true
	})
}

func setActionFlagDescription(state *actionEntryDeclarationState, description Description) {
	mutateActionEntry(state, actionEntryFieldDescription, func(entry *authoringActionEntry) bool {
		if entry.kind != authoringActionEntryFlag {
			return false
		}
		entry.flag.description = description
		return true
	})
}

func setActionFlagPlacement(state *actionEntryDeclarationState, placement FlagPlacement) {
	mutateActionEntry(state, actionEntryFieldPlacement, func(entry *authoringActionEntry) bool {
		if entry.kind != authoringActionEntryFlag || !validActionPlacement(placement) {
			return false
		}
		entry.flag.placement = placement
		return true
	})
}

func setActionFlagRelease(state *actionEntryDeclarationState, release ReleaseLifecycle) {
	mutateActionEntry(state, actionEntryFieldRelease, func(entry *authoringActionEntry) bool {
		if entry.kind != authoringActionEntryFlag {
			return false
		}
		entry.flag.release = release
		return true
	})
}

func setActionCommandHidden(state *actionEntryDeclarationState) {
	mutateActionEntry(state, actionEntryFieldHidden, func(entry *authoringActionEntry) bool {
		if entry.kind != authoringActionEntryCommand {
			return false
		}
		entry.command.hidden = true
		return true
	})
}

func setActionCommandDescription(state *actionEntryDeclarationState, description Description) {
	mutateActionEntry(state, actionEntryFieldDescription, func(entry *authoringActionEntry) bool {
		if entry.kind != authoringActionEntryCommand {
			return false
		}
		entry.command.description = description
		return true
	})
}

func setActionCommandRelease(state *actionEntryDeclarationState, release ReleaseLifecycle) {
	mutateActionEntry(state, actionEntryFieldRelease, func(entry *authoringActionEntry) bool {
		if entry.kind != authoringActionEntryCommand {
			return false
		}
		entry.command.release = release
		return true
	})
}

func replaceActionEntry(source, target *actionEntryDeclarationState) {
	if source == nil {
		return
	}
	if source.session == nil {
		panicBuilderMisuse()
	}
	source.session.use(func(draft *actionEntryDeclarationScope) {
		if !draft.valid || source.index < 0 || source.index >= len(draft.entries) {
			return
		}
		if _, duplicate := source.configured[actionEntryFieldReplacement]; duplicate {
			draft.valid = false
			return
		}
		if target == nil || target.session != source.session || target.index < 0 || target.index >= len(draft.entries) ||
			target.index == source.index {
			draft.valid = false
			return
		}
		entry := draft.entries[source.index]
		entry.replacement = target.index
		source.configured[actionEntryFieldReplacement] = struct{}{}
		draft.entries[source.index] = entry
	})
}

func stageIntrinsicActionEntries(draft *declarationDraft, entries []authoringActionEntry) {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	offset := len(draft.actions.entries)
	for _, entry := range entries {
		staged := entry
		staged.position = draft.nextPosition()
		if staged.replacement >= 0 {
			staged.replacement += offset
		}
		draft.actions.entries = append(draft.actions.entries, staged)
	}
}

func intrinsicVersionText(version VersionConfig, release ReleaseScheme) (string, bool, error) {
	configured := version.Disabled || version.WhenEmpty || version.Commit != "" || version.CommitLength != 0 ||
		version.ReplaceEntries != nil
	if version.CommitLength < 0 || (version.CommitLength > 0 && version.Commit == "") ||
		strings.ContainsAny(version.Commit, "\r\n") {
		return "", false, errInvalidAuthoringTransaction
	}
	if !release.valid() {
		if configured {
			return "", false, errInvalidAuthoringTransaction
		}
		return "", false, nil
	}
	text := release.current()
	if version.Commit == "" {
		return text, true, nil
	}
	commit := version.Commit
	if version.CommitLength > 0 {
		runes := []rune(commit)
		if len(runes) > version.CommitLength {
			commit = string(runes[:version.CommitLength])
		}
	}
	return text + " (" + commit + ")", true, nil
}

const (
	standardHelpActionDescription    = "Show help."
	standardVersionActionDescription = "Show version."
)

func captureHelpActionEntries(replace func(*HelpActionEntries)) ([]authoringActionEntry, error) {
	if replace == nil {
		description := Description{
			Short: standardHelpActionDescription,
		}
		flag := actionFlag{
			long:        "help",
			placement:   FlagPlacementSubtree,
			description: description,
		}
		returnedDescription := Description{
			Short: standardHelpActionDescription,
		}
		actionFlag := actionFlag{
			short:       'h',
			placement:   FlagPlacementSubtree,
			description: returnedDescription,
		}
		description2 := Description{
			Short: standardHelpActionDescription,
		}
		command := actionCommand{
			name:        "help",
			description: description2,
		}
		return []authoringActionEntry{
			{
				kind:        authoringActionEntryFlag,
				action:      intrinsicActionHelp,
				origin:      DeclarationOriginFrontend,
				flag:        flag,
				replacement: -1,
			},
			{
				kind:        authoringActionEntryFlag,
				action:      intrinsicActionHelp,
				origin:      DeclarationOriginFrontend,
				flag:        actionFlag,
				replacement: -1,
			},
			{
				kind:        authoringActionEntryCommand,
				action:      intrinsicActionHelp,
				origin:      DeclarationOriginFrontend,
				command:     command,
				replacement: -1,
			},
		}, nil
	}
	scope := &actionEntryDeclarationScope{
		action: intrinsicActionHelp,
		valid:  true,
	}
	session := &actionEntrySession{
		draft: scope,
		state: actionEntrySessionPrepared,
	}
	if cause := session.invoke(func() {
		replace(&HelpActionEntries{
			session: session,
		})
	}); cause != nil {
		return nil, cause
	}
	return closeActionEntryScope(scope)
}

func captureVersionActionEntries(replace func(*VersionActionEntries)) ([]authoringActionEntry, error) {
	if replace == nil {
		description := Description{
			Short: standardVersionActionDescription,
		}
		flag := actionFlag{
			long:        "version",
			description: description,
		}
		return []authoringActionEntry{
			{
				kind:        authoringActionEntryFlag,
				action:      intrinsicActionVersion,
				origin:      DeclarationOriginFrontend,
				flag:        flag,
				replacement: -1,
			},
		}, nil
	}
	scope := &actionEntryDeclarationScope{
		action: intrinsicActionVersion,
		valid:  true,
	}
	session := &actionEntrySession{
		draft: scope,
		state: actionEntrySessionPrepared,
	}
	if cause := session.invoke(func() {
		replace(&VersionActionEntries{
			session: session,
		})
	}); cause != nil {
		return nil, cause
	}
	return closeActionEntryScope(scope)
}

func closeActionEntryScope(scope *actionEntryDeclarationScope) ([]authoringActionEntry, error) {
	if scope == nil {
		return nil, errInvalidAuthoringTransaction
	}
	if !scope.valid || !validActionReplacementGraph(scope.entries) {
		return nil, errInvalidAuthoringTransaction
	}
	result := make([]authoringActionEntry, len(scope.entries))
	copy(result, scope.entries)
	return result, nil
}

func validActionReplacementGraph(entries []authoringActionEntry) bool {
	for source := range entries {
		target := entries[source].replacement
		if target < 0 {
			continue
		}
		if target >= len(entries) || target == source {
			return false
		}
		seen := map[int]struct{}{source: {}}
		for target >= 0 {
			if target >= len(entries) {
				return false
			}
			if _, duplicate := seen[target]; duplicate {
				return false
			}
			seen[target] = struct{}{}
			target = entries[target].replacement
		}
	}
	return true
}

func validActionPlacement(placement FlagPlacement) bool {
	switch placement {
	case FlagPlacementUnknown, FlagPlacementCommand, FlagPlacementSubtree:
		return true
	default:
		return false
	}
}

func validActionFlag(flag actionFlag) bool {
	forms := 0
	if flag.long != "" {
		forms++
		if !validLongFlagName("--" + flag.long) {
			return false
		}
	}
	if flag.short != 0 {
		forms++
		if !validShortFlagText("-" + string(flag.short)) {
			return false
		}
	}
	if flag.literal != "" {
		forms++
		if !validLiteralFlagSpelling(flag.literal) {
			return false
		}
	}
	return forms == 1 && validActionPlacement(flag.placement)
}

func validActionCommand(command actionCommand) bool {
	return command.name != "" && !containsForbiddenDeclarationText(command.name)
}

func containsForbiddenDeclarationText(text string) bool {
	for _, r := range text {
		if r == '\x00' || r == '\r' || r == '\n' {
			return true
		}
	}
	return false
}

func actionFlagGrammar(flag actionFlag) (FlagSpellingKind, string, bool) {
	switch {
	case flag.long != "":
		return FlagSpellingLong, "--" + flag.long, true
	case flag.short != 0:
		return FlagSpellingShort, "-" + string(flag.short), true
	case flag.literal != "":
		return FlagSpellingLiteral, flag.literal, true
	default:
		return FlagSpellingUnknown, "", false
	}
}

func actionEnabled(actions authoringIntrinsicActions, action intrinsicAction) bool {
	switch action {
	case intrinsicActionHelp:
		return !actions.helpDisabled
	case intrinsicActionVersion:
		return actions.versionPresent && !actions.versionDisabled
	default:
		return false
	}
}

func (actions authoringIntrinsicActions) semanticDefinition() (intrinsicActions, bool) {
	if actions.helpRenderer == nil {
		return intrinsicActions{}, false
	}
	helpRenderer := actions.helpRenderer
	help := acceptHelpActionDefinition(
		actions.helpDisabled,
		actions.helpWhenEmpty,
		func(ctx context.Context, rawPage any) (string, error) {
			page, ok := rawPage.(HelpPage)
			if !ok {
				panic("cli: invalid help renderer input")
			}
			return helpRenderer(ctx, page)
		},
	)
	return acceptIntrinsicActions(
		help,
		acceptVersionActionDefinition(
			actions.versionText,
			actions.versionPresent,
			actions.versionDisabled,
			actions.versionWhenEmpty,
		),
	), true
}

func prepareIntrinsicActions(draft *declarationCandidate, root acceptanceCommandRef, rootDialect dialectInput, actions authoringIntrinsicActions) ([]ActionEntryID, error) {
	refs := make([]actionEntryRef, len(actions.entries))
	ids := make([]ActionEntryID, len(actions.entries))
	for index, entry := range actions.entries {
		position := entry.position
		ref := draft.ActionEntry(root, position)
		if !entry.origin.valid() {
			return nil, errStaticDeclarationRejected
		}
		draft.ActionEntryOrigin(ref, entry.origin, position)
		refs[index] = ref
		ids[index] = ref.ID()
		enabled := actionEnabled(actions, entry.action)
		switch entry.kind {
		case authoringActionEntryFlag:
			kind, text, ok := actionFlagGrammar(entry.flag)
			if !ok {
				return nil, errStaticDeclarationRejected
			}
			if entry.flag.placement == FlagPlacementUnknown {
				if rootDialect.DefaultFlagPlacement != FlagPlacementCommand &&
					rootDialect.DefaultFlagPlacement != FlagPlacementSubtree {
					return nil, errStaticDeclarationRejected
				}
				draft.ActionFlagGrammarDefaulted(
					ref,
					entry.action,
					kind,
					text,
					rootDialect.DefaultFlagPlacement,
					enabled,
				)
			} else {
				placement, ok := prepareFlagPlacement(entry.flag.placement)
				if !ok {
					return nil, errStaticDeclarationRejected
				}
				draft.ActionFlagGrammar(ref, entry.action, kind, text, placement, enabled)
			}
			draft.ActionEntryPresentation(
				ref,
				semanticPresentation(entry.flag.description, nil, entry.flag.hidden),
				position,
			)
			draft.ActionEntryLifecycle(ref, entry.flag.release, position)
		case authoringActionEntryCommand:
			draft.ActionCommandGrammar(ref, entry.action, entry.command.name, enabled)
			draft.ActionEntryPresentation(
				ref,
				semanticPresentation(entry.command.description, nil, entry.command.hidden),
				position,
			)
			draft.ActionEntryLifecycle(ref, entry.command.release, position)
		default:
			return nil, errStaticDeclarationRejected
		}
	}
	for source, entry := range actions.entries {
		if entry.replacement < 0 {
			continue
		}
		if entry.replacement >= len(refs) {
			return nil, errStaticDeclarationRejected
		}
		draft.ActionEntryReplacement(
			refs[source], refs[entry.replacement],
			actions.entries[source].position,
		)
	}
	return ids, nil
}

type intrinsicAction uint8

const (
	intrinsicActionInvalid intrinsicAction = iota
	intrinsicActionHelp
	intrinsicActionVersion
)

type actionCommandGrammar struct {
	id       ActionEntryID
	parent   CommandID
	action   intrinsicAction
	name     string
	position declarationPosition
	enabled  bool
}

// actionCommandGrammarInput is the complete immutable payload of one intrinsic action command.
type actionCommandGrammarInput struct {
	ID       ActionEntryID
	Parent   CommandID
	Action   intrinsicAction
	Name     string
	Position declarationPosition
	Enabled  bool
}

func validActionCommandGrammarInput(input actionCommandGrammarInput) bool {
	return input.ID.present() && input.Parent.present() && input.Action != intrinsicActionInvalid && input.Name != ""
}

func acceptActionCommandGrammar(input actionCommandGrammarInput) actionCommandGrammar {
	if !validActionCommandGrammarInput(input) {
		panic("semantic: invalid action-command grammar")
	}
	grammar := actionCommandGrammar{
		id:       input.ID,
		parent:   input.Parent,
		action:   input.Action,
		name:     input.Name,
		position: input.Position,
		enabled:  input.Enabled,
	}

	return grammar
}

type semanticActionFlagGrammar struct {
	id               ActionEntryID
	parent           CommandID
	action           intrinsicAction
	kind             FlagSpellingKind
	text             string
	placement        FlagPlacement
	defaultPlacement bool
	position         declarationPosition
	enabled          bool
}

// actionFlagGrammarInput is the complete immutable payload of one intrinsic action flag.
type actionFlagGrammarInput struct {
	ID        ActionEntryID
	Parent    CommandID
	Action    intrinsicAction
	Kind      FlagSpellingKind
	Text      string
	Placement FlagPlacement
	Position  declarationPosition
	Enabled   bool
}

func validActionFlagGrammarInput(input actionFlagGrammarInput) bool {
	return input.ID.present() && input.Parent.present() && input.Action != intrinsicActionInvalid &&
		input.Kind != FlagSpellingUnknown && input.Text != "" && input.Placement != FlagPlacementUnknown
}

func acceptActionFlagGrammar(input actionFlagGrammarInput) semanticActionFlagGrammar {
	if !validActionFlagGrammarInput(input) {
		panic("semantic: invalid action-flag grammar")
	}
	grammar := semanticActionFlagGrammar{
		id:        input.ID,
		parent:    input.Parent,
		action:    input.Action,
		kind:      input.Kind,
		text:      input.Text,
		placement: input.Placement,
		position:  input.Position,
		enabled:   input.Enabled,
	}

	return grammar
}

func acceptDefaultedActionFlagGrammar(input actionFlagGrammarInput) semanticActionFlagGrammar {
	entry := acceptActionFlagGrammar(input)
	entry.defaultPlacement = true
	return entry
}

// helpRender is the erased application-lifetime Help renderer. The facade owns
// the concrete public HelpPage type and validates the erased request before the
// callback is invoked.
type helpRender func(context.Context, any) (string, error)

type helpActionDefinition struct {
	disabled  bool
	whenEmpty bool
	render    helpRender
}

func acceptHelpActionDefinition(disabled, whenEmpty bool, render helpRender) helpActionDefinition {
	definition := helpActionDefinition{
		disabled:  disabled,
		whenEmpty: whenEmpty,
		render:    render,
	}

	return definition
}

func (definition helpActionDefinition) Valid() bool { return definition.render != nil }

func (definition helpActionDefinition) Disabled() bool { return definition.disabled }

func (definition helpActionDefinition) WhenEmpty() bool { return definition.whenEmpty }

func (definition helpActionDefinition) Render(ctx context.Context, page any) (string, error) {
	if definition.render == nil {
		panic("semantic: invalid help action definition")
	}
	return definition.render(ctx, page)
}

type versionActionDefinition struct {
	text      string
	present   bool
	disabled  bool
	whenEmpty bool
}

func acceptVersionActionDefinition(text string, present, disabled, whenEmpty bool) versionActionDefinition {
	definition := versionActionDefinition{
		text:      text,
		present:   present,
		disabled:  disabled,
		whenEmpty: whenEmpty,
	}

	return definition
}

func (definition versionActionDefinition) Text() string { return definition.text }

func (definition versionActionDefinition) Disabled() bool { return definition.disabled }

func (definition versionActionDefinition) Present() bool { return definition.present }

// intrinsicActions is the immutable static behavior retained for frontend-owned actions.
type intrinsicActions struct {
	help    helpActionDefinition
	version versionActionDefinition
	valid   bool
}

func acceptIntrinsicActions(help helpActionDefinition, version versionActionDefinition) intrinsicActions {
	if !help.Valid() {
		panic("semantic: invalid intrinsic help action")
	}
	result := intrinsicActions{
		help:    help,
		version: version,
		valid:   true,
	}

	return result
}

func (actions intrinsicActions) Valid() bool { return actions.valid && actions.help.Valid() }

func (actions intrinsicActions) Help() helpActionDefinition { return actions.help }

func (actions intrinsicActions) Version() versionActionDefinition { return actions.version }

type actionEntryRecord struct {
	meta         recordMeta[ActionEntryID]
	parent       CommandID
	origin       DeclarationOrigin
	presentation presentation
	lifecycle    ReleaseLifecycle
	replacement  ActionEntryID
}

func acceptActionEntryRecord(id ActionEntryID, parent CommandID, position declarationPosition) actionEntryRecord {
	mustCommand(parent)
	record := actionEntryRecord{
		meta:   acceptRecordMeta(id, position),
		parent: parent,
	}
	return record
}

func (record actionEntryRecord) Replacement() (ActionEntryID, bool) {
	return record.replacement, record.replacement.present()
}
