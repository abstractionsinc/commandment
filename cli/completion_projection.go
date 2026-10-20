package cli

const completionReferenceScopeCapacityFactor = 2

// completionStaticProjector projects only the detached presentation facts that
// can become static Completion output. It is operation-local and derives every
// answer from the canonical semantic view, effective grammar, release law, and
// localization owner; it retains no projected state across operations.
type completionStaticProjector struct {
	operation               *invocation
	model                   *effectiveModel
	grammar                 grammarView
	visibility              ReferenceVisibility
	scheme                  ReleaseScheme
	release                 string
	locale                  string
	applicationLocalization localizationSnapshot
	commands                map[CommandID]completionStaticCommand
	grammarScopes           map[CommandID]*completionStaticGrammarScope
}

type completionStaticGrammarScope struct {
	commandEntriesResolved bool
	commandEntries         map[CommandEntryID]commandEntryGrammar
	canonicalEntries       map[CommandID]commandEntryGrammar
	actionCommandsResolved bool
	actionCommands         map[ActionEntryID]actionCommandGrammar
	actionFlagsResolved    bool
	actionFlags            map[ActionEntryID]semanticActionFlagGrammar
	flagsResolved          bool
	flagSpellings          map[FlagSpellingID]string
}

type completionStaticCommand struct {
	resolved  bool
	available bool
	visible   bool
	parent    CommandID
	name      string
}

func completionResultLocale(model *effectiveModel, requested Locale) (Locale, error) {
	if !model.Valid() {
		return "", errInvalidStaticProjectionInput
	}
	localization := model.Application().localization
	locale, err := selectEffectiveLocale(string(requested), "", localization.SourceLocale())
	if err != nil {
		return "", errInvalidStaticProjectionInput
	}
	return Locale(locale), nil
}

func prepareCompletionStaticProjector(operation *invocation, visibility ReferenceVisibility, requested Locale) (*completionStaticProjector, error) {
	if operation == nil || (visibility != ReferenceVisibilityPublic && visibility != ReferenceVisibilityAll) {
		return nil, errInvalidStaticProjectionInput
	}
	model := operation.analysis.effectiveModel
	grammar, ok := operation.analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		panic("cli: accepted completion operation has no effective grammar")
	}
	locale, err := completionResultLocale(model, requested)
	if err != nil {
		return nil, err
	}
	scheme := model.Application().release
	release := ""
	if scheme.valid() {
		release = scheme.current()
	}
	projector := &completionStaticProjector{
		operation:               operation,
		model:                   model,
		grammar:                 grammar,
		visibility:              visibility,
		scheme:                  scheme,
		release:                 release,
		locale:                  string(locale),
		applicationLocalization: model.Application().localization,
		commands:                make(map[CommandID]completionStaticCommand),
		grammarScopes:           make(map[CommandID]*completionStaticGrammarScope),
	}
	return projector, nil
}

func (projector *completionStaticProjector) localizationForCommand(command CommandID) localizationSnapshot {
	if localization := projector.operation.LocalizationForCommand(command); localization.Enabled() {
		return localization
	}
	return projector.applicationLocalization
}

func (projector *completionStaticProjector) localizationForEntry(entry CommandEntryID, parent CommandID) localizationSnapshot {
	if localization := projector.operation.LocalizationForCommandEntry(entry); localization.Enabled() {
		return localization
	}
	return projector.localizationForCommand(parent)
}

func (projector *completionStaticProjector) lifecycle(lifecycle ReleaseLifecycle) (LifecycleState, error) {
	return referenceLifecycleState(projector.scheme, releaseLifecycle(lifecycle), projector.release)
}

func (projector *completionStaticProjector) grammarScope(command CommandID) *completionStaticGrammarScope {
	scope := projector.grammarScopes[command]
	if scope == nil {
		scope = &completionStaticGrammarScope{}
		projector.grammarScopes[command] = scope
	}
	return scope
}

func (projector *completionStaticProjector) resolveCommandEntries(command CommandID) *completionStaticGrammarScope {
	scope := projector.grammarScope(command)
	if scope.commandEntriesResolved {
		return scope
	}
	scope.commandEntriesResolved = true
	entries := projector.grammar.CommandEntries(command)
	for _, entry := range entries {
		if scope.commandEntries == nil {
			scope.commandEntries = make(map[CommandEntryID]commandEntryGrammar, len(entries))
		}
		scope.commandEntries[entry.id] = entry
		if entry.kind == CommandEntryCanonical {
			if scope.canonicalEntries == nil {
				scope.canonicalEntries = make(map[CommandID]commandEntryGrammar)
			}
			scope.canonicalEntries[entry.target] = entry
		}
	}
	return scope
}

func (projector *completionStaticProjector) resolveActionCommands(command CommandID) *completionStaticGrammarScope {
	scope := projector.grammarScope(command)
	if scope.actionCommandsResolved {
		return scope
	}
	scope.actionCommandsResolved = true
	actions := projector.grammar.ActionCommands(command)
	for _, action := range actions {
		if scope.actionCommands == nil {
			scope.actionCommands = make(map[ActionEntryID]actionCommandGrammar, len(actions))
		}
		scope.actionCommands[action.id] = action
	}
	return scope
}

func (projector *completionStaticProjector) resolveActionFlags(command CommandID) *completionStaticGrammarScope {
	scope := projector.grammarScope(command)
	if scope.actionFlagsResolved {
		return scope
	}
	scope.actionFlagsResolved = true
	actions := projector.grammar.ActionFlags(command)
	for _, action := range actions {
		if scope.actionFlags == nil {
			scope.actionFlags = make(map[ActionEntryID]semanticActionFlagGrammar, len(actions))
		}
		scope.actionFlags[action.id] = action
	}
	return scope
}

func (projector *completionStaticProjector) resolveFlagSpellings(command CommandID) *completionStaticGrammarScope {
	scope := projector.grammarScope(command)
	if scope.flagsResolved {
		return scope
	}
	scope.flagsResolved = true
	flags := projector.grammar.Flags(command)
	for _, flag := range flags {
		for _, spelling := range flag.Spellings() {
			if scope.flagSpellings == nil {
				scope.flagSpellings = make(map[FlagSpellingID]string)
			}
			scope.flagSpellings[spelling.id] = spelling.text
		}
	}
	return scope
}

func (projector *completionStaticProjector) canonicalEntry(command CommandID) (commandEntryGrammar, bool) {
	record, ok := projector.model.Command(command)
	if !ok {
		return commandEntryGrammar{}, false
	}
	parent, ok := record.Parent()
	if !ok {
		return commandEntryGrammar{}, false
	}
	entry, ok := projector.resolveCommandEntries(parent).canonicalEntries[command]
	return entry, ok
}

func (projector *completionStaticProjector) command(command CommandID) (completionStaticCommand, bool) {
	if cached, ok := projector.commands[command]; ok && cached.resolved {
		return cached, true
	}
	record, ok := projector.model.Command(command)
	if !ok {
		return completionStaticCommand{}, false
	}
	state := completionStaticCommand{
		resolved:  true,
		available: true,
		visible:   true,
	}
	parent, hasParent := record.Parent()
	if !hasParent {
		state.visible = projector.visibility == ReferenceVisibilityAll || !record.presentation.hidden
		projector.commands[command] = state
		return state, true
	}
	parentState, ok := projector.command(parent)
	if !ok {
		return completionStaticCommand{}, false
	}
	entry, ok := projector.canonicalEntry(command)
	if !ok {
		return completionStaticCommand{}, false
	}
	entryRecord, ok := projector.model.CommandEntry(entry.id)
	if !ok {
		return completionStaticCommand{}, false
	}
	lifecycle, err := projector.lifecycle(entryRecord.lifecycle)
	if err != nil {
		return completionStaticCommand{}, false
	}
	state.available = parentState.available && referenceAvailable(lifecycle)
	state.visible = state.available && parentState.visible &&
		(projector.visibility == ReferenceVisibilityAll || !record.presentation.hidden)
	state.parent = parent
	state.name = entry.name
	projector.commands[command] = state
	return state, true
}

func (projector *completionStaticProjector) commandPath(command CommandID) ([]string, bool) {
	if _, ok := projector.command(command); !ok {
		return nil, false
	}
	depth := 0
	for current := command; current.present(); {
		state, ok := projector.commands[current]
		if !ok || !state.resolved {
			return nil, false
		}
		if state.name != "" {
			depth++
		}
		current = state.parent
	}
	path := make([]string, depth)
	index := depth - 1
	for current := command; current.present(); {
		state := projector.commands[current]
		if state.name != "" {
			path[index] = state.name
			index--
		}
		current = state.parent
	}
	return path, true
}

func (projector *completionStaticProjector) commandEntryGrammar(record commandEntryRecord) (commandEntryGrammar, bool) {
	entry, ok := projector.resolveCommandEntries(record.parent).commandEntries[record.meta.id]
	return entry, ok
}

func (projector *completionStaticProjector) commandItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	if candidate.Kind == completionSyntaxCandidateCommandEntry {
		return projector.commandEntryItem(client, candidate, targetIndex)
	}
	return projector.actionSyntaxItem(client, candidate, targetIndex)
}

func (projector *completionStaticProjector) commandEntryItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	record, ok := projector.model.CommandEntry(candidate.CommandEntry)
	if !ok {
		return CompletionItem{}, false
	}
	entry, ok := projector.commandEntryGrammar(record)
	if !ok {
		return CompletionItem{}, false
	}
	projection, err := effectiveCommandEntryProjection(
		projector.model,
		projector.grammar,
		projector.scheme,
		projector.release,
		projector.visibility,
		record.meta.id,
	)
	if err != nil || !projection.visible {
		return CompletionItem{}, false
	}
	state := projection.lifecycle
	kind := CompletionCandidateCommand
	switch entry.kind {
	case CommandEntryAlias:
		kind = CompletionCandidateCommandAlias
	case CommandEntryShortcut:
		kind = CompletionCandidateCommandShortcut
	case CommandEntryCanonical:
	default:
		return CompletionItem{}, false
	}
	item := CompletionItem{
		Value:          entry.name,
		DisplayText:    entry.name,
		Kind:           kind,
		SortText:       entry.name,
		LifecycleState: state,
		Declaration: DeclarationIdentity{
			Kind:         DeclarationKindCommandEntry,
			CommandEntry: record.meta.id,
		},
	}
	if client.Capabilities.Descriptions {
		description := record.presentation.description
		localization := projector.localizationForEntry(record.meta.id, record.parent)
		if (entry.kind == CommandEntryAlias || entry.kind == CommandEntryShortcut) &&
			description.Short == "" &&
			description.Long == "" {
			if target, ok := projector.model.Command(record.target); ok {
				description = target.presentation.description
				localization = projector.localizationForCommand(record.target)
			}
		}
		item.Description = referenceText(localization, projector.locale, description.Short)
	}
	if !client.Capabilities.CandidateKinds {
		item.Kind = CompletionCandidateUnknown
	}
	item.TargetIndexes = []int{targetIndex}
	item.Match = emptyPrefixCompletionMatch(client)
	return item, true
}

func (projector *completionStaticProjector) flagItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	if candidate.Kind == completionSyntaxCandidateFlagSpelling {
		return projector.flagSpellingItem(client, candidate, targetIndex)
	}
	return projector.actionSyntaxItem(client, candidate, targetIndex)
}

func (projector *completionStaticProjector) actionSyntaxItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	record, ok := projector.model.ActionEntry(candidate.ActionEntry)
	if !ok {
		return CompletionItem{}, false
	}
	parent, ok := projector.command(record.parent)
	if !ok {
		return CompletionItem{}, false
	}
	state, err := projector.lifecycle(record.lifecycle)
	if err != nil || !referenceAvailable(state) || !parent.available || !parent.visible ||
		(projector.visibility == ReferenceVisibilityPublic && record.presentation.hidden) {
		return CompletionItem{}, false
	}
	var value string
	switch candidate.Kind {
	case completionSyntaxCandidateCommandAction:
		action, exists := projector.resolveActionCommands(record.parent).actionCommands[record.meta.id]
		if !exists {
			return CompletionItem{}, false
		}
		value = action.name
	case completionSyntaxCandidateFlagAction:
		action, exists := projector.resolveActionFlags(record.parent).actionFlags[record.meta.id]
		if !exists {
			return CompletionItem{}, false
		}
		value = action.text
	default:
		return CompletionItem{}, false
	}
	return projector.actionEntryItem(client, record, state, value, targetIndex)
}

func (projector *completionStaticProjector) actionEntryItem(client CompletionClient, record actionEntryRecord, state LifecycleState, value string, targetIndex int) (CompletionItem, bool) {
	if value == "" {
		return CompletionItem{}, false
	}
	item := CompletionItem{
		Value:          value,
		DisplayText:    value,
		Kind:           CompletionCandidateAction,
		SortText:       value,
		LifecycleState: state,
		Declaration: DeclarationIdentity{
			Kind:        DeclarationKindActionEntry,
			ActionEntry: record.meta.id,
		},
	}
	if client.Capabilities.Descriptions {
		item.Description = referenceText(
			projector.localizationForCommand(record.parent),
			projector.locale,
			record.presentation.description.Short,
		)
	}
	if !client.Capabilities.CandidateKinds {
		item.Kind = CompletionCandidateUnknown
	}
	item.TargetIndexes = []int{targetIndex}
	item.Match = emptyPrefixCompletionMatch(client)
	return item, true
}

func (projector *completionStaticProjector) flagSpellingItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	spellingRecord, ok := projector.model.FlagSpelling(candidate.FlagSpelling)
	if !ok {
		return CompletionItem{}, false
	}
	flagRecord, ok := projector.model.Flag(spellingRecord.flag)
	if !ok {
		return CompletionItem{}, false
	}
	parent, ok := projector.command(flagRecord.command)
	if !ok {
		return CompletionItem{}, false
	}
	flagState, err := projector.lifecycle(flagRecord.lifecycle)
	if err != nil {
		return CompletionItem{}, false
	}
	spellingState, err := projector.lifecycle(spellingRecord.lifecycle)
	if err != nil || !referenceAvailable(flagState) || !referenceAvailable(spellingState) || !parent.available ||
		!parent.visible ||
		(projector.visibility == ReferenceVisibilityPublic && (flagRecord.presentation.hidden || spellingRecord.hidden)) {
		return CompletionItem{}, false
	}
	text, ok := projector.resolveFlagSpellings(flagRecord.command).flagSpellings[spellingRecord.meta.id]
	if !ok || text == "" {
		return CompletionItem{}, false
	}
	item := CompletionItem{
		Value:          text,
		DisplayText:    text,
		Kind:           CompletionCandidateFlag,
		SortText:       text,
		LifecycleState: spellingState,
		Declaration: DeclarationIdentity{
			Kind:         DeclarationKindFlagSpelling,
			FlagSpelling: spellingRecord.meta.id,
		},
	}
	if client.Capabilities.Descriptions {
		item.Description = referenceText(
			projector.localizationForCommand(flagRecord.command),
			projector.locale,
			flagRecord.presentation.description.Short,
		)
	}
	if !client.Capabilities.CandidateKinds {
		item.Kind = CompletionCandidateUnknown
	}
	item.TargetIndexes = []int{targetIndex}
	item.Match = emptyPrefixCompletionMatch(client)
	return item, true
}

func completionStaticProjectionNeedsReference(effectiveModel *effectiveModel, grammar grammarResult) bool {
	if !effectiveModel.Valid() {
		return true
	}
	selected := make(map[CommandID]struct{}, len(grammar.Selection.Path))
	scopes := make([]CommandID, 0, len(grammar.Selection.Path)*completionReferenceScopeCapacityFactor)
	seenScopes := make(map[CommandID]struct{}, len(grammar.Selection.Path)*completionReferenceScopeCapacityFactor)
	addScope := func(command CommandID) {
		if !command.present() {
			return
		}
		if _, seen := seenScopes[command]; seen {
			return
		}
		seenScopes[command] = struct{}{}
		scopes = append(scopes, command)
	}
	for _, command := range grammar.Selection.Path {
		selected[command] = struct{}{}
		addScope(command)
	}

	effective, ok := effectiveModel.EffectiveGrammar()
	if !ok {
		return true
	}
	for _, parent := range grammar.Selection.Path {
		for _, entry := range effective.CommandEntries(parent) {
			if !entry.enabled || !structuralProfileActive(grammar.SelectedProfiles, parent, entry.profile) {
				continue
			}
			target, ok := effectiveModel.Command(entry.target)
			if !ok {
				continue
			}
			targetParent, ok := target.Parent()
			if ok && targetParent == parent {
				addScope(target.meta.id)
			}
		}
	}

	for _, command := range scopes {
		for _, record := range effectiveModel.ConditionPoliciesForCommand(command) {
			if !structuralConditionActive(effectiveModel, selected, record.target) {
				continue
			}
			owner := structuralConditionOwner(effectiveModel, record.target)
			if structuralProfileActive(grammar.SelectedProfiles, owner, record.profile) {
				return true
			}
		}
	}
	return false
}
