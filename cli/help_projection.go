package cli

import (
	"cmp"
	"slices"
)

type helpProjection struct {
	operation    *invocation
	model        *effectiveModel
	grammarModel grammarView
	grammar      grammarResult
	states       map[conditionTarget]helpTargetState
	locale       Locale
	scheme       ReleaseScheme
	release      string
	target       CommandID
	scope        []string
	frontier     helpFrontier
	defaults     map[CommandID]bool
	providers    helpProviderOrigins
	helpGroups   []helpGroupRecord
	page         HelpPage
}

type helpProviderOrigins struct {
	entries  map[CommandEntryID]struct{}
	batches  map[CommandProviderID][]CommandEntryID
	commands map[CommandID]struct{}
}

func (operation *invocation) buildHelpPage(grammar grammarResult, analysis helpAnalysis, locale Locale) (HelpPage, error) {
	if operation == nil || operation.analysis == nil {
		panic("cli: Help projection requires an invocation analysis")
	}
	if analysis.request.target != grammar.Selection.Selected {
		panic("cli: Help projection request does not match authoritative grammar")
	}
	model := operation.analysis.effectiveModel
	grammarModel, ok := model.EffectiveGrammar()
	if !ok {
		panic("cli: completed Help analysis has no effective grammar")
	}
	target := analysis.request.target
	targetRecord, ok := model.Command(target)
	if !ok {
		panic("cli: authoritative Help target cannot resolve")
	}
	frontier, ok := buildHelpFrontier(model, grammar)
	if !ok {
		panic("cli: authoritative Help target has no Help frontier")
	}
	facts := collectHelpRelevantFacts(model, grammar, frontier, operation.input.locale == "")
	values := operationDependencySeed(operation.analysis)
	states := evaluateHelpTargetStates(model, facts.conditions, grammar, frontier, analysis.availability, values)
	targetState, ok := states[commandConditionTarget(target)]
	if !ok || !targetState.available {
		panic("cli: authoritative Help target is unavailable after completed analysis")
	}

	app := model.Application()
	scheme := app.release
	release := ""
	if scheme.valid() {
		release = scheme.current()
	}
	scope := commandPathNames(model, target)
	version := ""
	if actions, ok := model.IntrinsicActions(); ok && actions.Version().Present() {
		version = actions.Version().Text()
	}
	page := HelpPage{
		Kind:               HelpPageCommand,
		Selection:          analysis.request.selection,
		ApplicationName:    app.name,
		Name:               helpCommandName(model, target),
		Version:            version,
		SelectedPath:       append([]string(nil), scope...),
		RequestedPath:      append([]string(nil), analysis.request.requestedPath...),
		EvaluatedRelease:   release,
		Runnable:           targetRecord.execution.Runnable(),
		FlagSpellings:      make(map[FlagSpellingID]HelpFlagSpelling),
		ValueDomainEntries: make(map[ValueDomainEntryID]HelpValueDomainEntry),
		Locale:             locale,
		Labels:             defaultHelpLabels(app.frameworkLocalization, locale),
	}
	projection := helpProjection{
		operation:    operation,
		model:        model,
		grammarModel: grammarModel,
		grammar:      grammar,
		states:       states,
		locale:       locale,
		scheme:       scheme,
		release:      release,
		target:       target,
		scope:        scope,
		frontier:     frontier,
		defaults:     helpDefaultTargets(model, target, analysis.request.profiles, analysis.defaults),
		helpGroups:   facts.helpGroups,
		page:         page,
	}
	projection.providers = projection.providerOrigins()
	if err := projection.projectTargetPresentation(targetRecord); err != nil {
		return HelpPage{}, err
	}
	projection.page.CommandGroups = projection.commandGroups()
	argumentForms, err := projection.argumentForms()
	if err != nil {
		return HelpPage{}, err
	}
	projection.page.ArgumentForms = argumentForms
	flagGroups, err := projection.flagGroups()
	if err != nil {
		return HelpPage{}, err
	}
	projection.page.FlagGroups = flagGroups
	projection.page.Usage = buildHelpUsage(projection.page)
	return projection.page, nil
}

func (projection *helpProjection) projectTargetPresentation(targetRecord commandRecord) error {
	if projection.target == projection.grammarModel.Root() {
		app := projection.model.Application()
		projection.page.Kind = HelpPageApplication
		projection.page.Name = app.name
		projection.page.Description = localizedDescription(app.localization, projection.locale, app.description)
		projection.page.Examples = localizedExamples(app.localization, projection.locale, app.Examples())
		projection.page.LifecycleState = LifecycleStateCurrent
		return nil
	}

	localization := projection.operation.LocalizationForCommand(projection.target)
	projection.page.Description = localizedDescription(localization, projection.locale, targetRecord.presentation.description)
	projection.page.Examples = localizedExamples(localization, projection.locale, targetRecord.presentation.Examples())
	canonical, ok := helpCanonicalEntry(projection.model, projection.grammarModel, projection.target)
	if !ok {
		panic("cli: authoritative Help target has no canonical command entry")
	}
	entryRecord, ok := projection.model.CommandEntry(canonical.id)
	if !ok {
		panic("cli: authoritative Help canonical command entry cannot resolve")
	}
	state, err := helpLifecycleState(projection.scheme, entryRecord.lifecycle, projection.release)
	if err != nil {
		return err
	}
	if !referenceAvailable(state) {
		panic("cli: authoritative Help target is unavailable for the evaluated release")
	}
	projection.page.Release = localizedReleaseLifecycle(
		projection.operation.LocalizationForCommandEntry(canonical.id),
		projection.locale,
		entryRecord.lifecycle,
	)
	projection.page.LifecycleState = state
	return nil
}

func helpLifecycleState(scheme ReleaseScheme, lifecycle ReleaseLifecycle, release string) (LifecycleState, error) {
	if !scheme.valid() {
		if lifecycle.IntroducedIn == "" && lifecycle.DeprecatedIn == "" && lifecycle.RemovedIn == "" {
			return LifecycleStateCurrent, nil
		}
		panic("cli: accepted Help lifecycle requires an unavailable release scheme")
	}
	return referenceLifecycleState(scheme, releaseLifecycle(lifecycle), release)
}

func localizedReleaseLifecycle(localization localizationSnapshot, locale Locale, lifecycle ReleaseLifecycle) ReleaseLifecycle {
	result := releaseLifecycle(lifecycle)
	result.Reason = referenceText(localization, string(locale), result.Reason)
	return result
}

func localizedDescription(localization localizationSnapshot, locale Locale, description Description) Description {
	return Description{
		Short: referenceText(localization, string(locale), description.Short),
		Long:  referenceText(localization, string(locale), description.Long),
	}
}

func localizedExamples(localization localizationSnapshot, locale Locale, examples []Example) []Example {
	result := make([]Example, len(examples))
	for index, example := range examples {
		result[index] = Example{
			Description: referenceText(localization, string(locale), example.Description),
			CommandLine: example.CommandLine,
		}
	}
	return result
}

func helpCanonicalEntry(model *effectiveModel, grammarModel grammarView, command CommandID) (commandEntryGrammar, bool) {
	record, ok := model.Command(command)
	if !ok {
		return commandEntryGrammar{}, false
	}
	parent, ok := record.Parent()
	if !ok {
		return commandEntryGrammar{}, false
	}
	for _, entry := range grammarModel.CommandEntries(parent) {
		if entry.target == command && entry.kind == CommandEntryCanonical {
			return entry, true
		}
	}
	return commandEntryGrammar{}, false
}

func helpCommandName(model *effectiveModel, command CommandID) string {
	grammarModel, ok := model.EffectiveGrammar()
	if !ok || command == grammarModel.Root() {
		return model.Application().name
	}
	if entry, ok := helpCanonicalEntry(model, grammarModel, command); ok {
		return entry.name
	}
	return ""
}

func helpCommandPathIDs(model *effectiveModel, command CommandID) []CommandID {
	grammarModel, ok := model.EffectiveGrammar()
	if !ok {
		panic("cli: completed Help projection cannot resolve required declaration state")
	}
	var reversed []CommandID
	current := command
	for {
		reversed = append(reversed, current)
		if current == grammarModel.Root() {
			break
		}
		record, ok := model.Command(current)
		if !ok {
			panic("cli: completed Help projection cannot resolve required declaration state")
		}
		parent, ok := record.Parent()
		if !ok {
			panic("cli: completed Help projection cannot resolve required declaration state")
		}
		current = parent
	}
	result := make([]CommandID, len(reversed))
	for i := range reversed {
		result[len(reversed)-1-i] = reversed[i]
	}
	return result
}

func helpRequestedPath(model *effectiveModel, grammar grammarResult) []string {
	var result []string
	for _, step := range grammar.Selection.Steps {
		if !step.Entry.present() {
			continue
		}
		grammarModel, ok := model.EffectiveGrammar()
		if !ok {
			continue
		}
		record, ok := model.CommandEntry(step.Entry)
		if !ok {
			continue
		}
		for _, entry := range grammarModel.CommandEntries(record.parent) {
			if entry.id == step.Entry {
				result = append(result, entry.name)
				break
			}
		}
	}
	return result
}

func (projection *helpProjection) providerOrigins() helpProviderOrigins {
	origins := helpProviderOrigins{
		entries:  make(map[CommandEntryID]struct{}),
		batches:  make(map[CommandProviderID][]CommandEntryID),
		commands: make(map[CommandID]struct{}),
	}
	for _, entry := range projection.grammarModel.CommandEntries(projection.target) {
		provider, ok := projection.model.ProviderForCommandEntry(entry.id)
		if !ok {
			continue
		}
		origins.entries[entry.id] = struct{}{}
		origins.batches[provider] = append(origins.batches[provider], entry.id)
	}
	for _, command := range projection.grammar.Selection.Path {
		if command == projection.grammarModel.Root() {
			continue
		}
		entry, ok := helpCanonicalEntry(projection.model, projection.grammarModel, command)
		if !ok {
			continue
		}
		if _, ok := projection.model.ProviderForCommandEntry(entry.id); ok {
			origins.commands[command] = struct{}{}
		}
	}
	return origins
}

type helpCommandCandidate struct {
	entry    CommandEntryID
	action   ActionEntryID
	position declarationPosition
	value    HelpCommand
}

func (projection *helpProjection) commandGroups() []HelpCommandGroup {
	candidates := make(map[CommandEntryID]helpCommandCandidate)
	var candidateOrder []CommandEntryID
	for _, entry := range projection.grammarModel.CommandEntries(projection.target) {
		if !entry.enabled || !structuralProfileActive(projection.grammar.SelectedProfiles, projection.target, entry.profile) {
			continue
		}
		state := projection.states[commandConditionTarget(entry.target)]
		if !state.available || !state.visible {
			continue
		}
		record, ok := projection.model.CommandEntry(entry.id)
		if !ok {
			panic("cli: completed Help projection cannot resolve required declaration state")
		}
		entryProjection, err := effectiveCommandEntryProjection(
			projection.model,
			projection.grammarModel,
			projection.scheme,
			projection.release,
			ReferenceVisibilityPublic,
			entry.id,
		)
		if err != nil || !entryProjection.visible {
			continue
		}
		life := entryProjection.lifecycle
		presentation := record.presentation
		description := presentation.description
		localization := projection.operation.LocalizationForCommandEntry(entry.id)
		if description.Short == "" && description.Long == "" {
			if command, ok := projection.model.Command(entry.target); ok {
				description = command.presentation.description
				localization = projection.operation.LocalizationForCommand(entry.target)
			}
		}
		if entry.kind == CommandEntryCanonical {
			if command, ok := projection.model.Command(entry.target); ok && command.presentation.hidden {
				continue
			}
		}
		path := helpCommandPathIDs(projection.model, entry.target)
		origin := DeclarationOriginApplication
		if _, provider := projection.providers.entries[entry.id]; provider {
			origin = DeclarationOriginProvider
		}
		candidate := HelpCommand{
			Kind:           helpCommandKind(entry.kind),
			Origin:         origin,
			CommandEntry:   entry.id,
			TargetPath:     path,
			Name:           entry.name,
			Description:    localizedDescription(localization, projection.locale, description),
			Default:        projection.defaults[entry.target],
			Runnable:       projection.commandRunnable(entry.target),
			HasSubcommands: projection.hasSubcommands(entry.target),
			Release: localizedReleaseLifecycle(
				projection.operation.LocalizationForCommandEntry(entry.id),
				projection.locale,
				record.lifecycle,
			),
			LifecycleState: life,
		}
		candidates[entry.id] = helpCommandCandidate{
			entry:    entry.id,
			position: entry.position,
			value:    candidate,
		}
		candidateOrder = append(candidateOrder, entry.id)
	}

	var actions []helpCommandCandidate
	for _, action := range projection.grammarModel.ActionCommands(projection.target) {
		if !action.enabled {
			continue
		}
		record, ok := projection.model.ActionEntry(action.id)
		if !ok || record.presentation.hidden {
			continue
		}
		life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
		if err != nil || !referenceAvailable(life) {
			continue
		}
		localization := projection.actionLocalization(record)
		helpCommandCandidateValue := helpCommandCandidate{
			action:   action.id,
			position: action.position,
			value: HelpCommand{
				Kind:           HelpCommandAction,
				Origin:         record.origin,
				ActionEntry:    action.id,
				Name:           action.name,
				Description:    localizedDescription(localization, projection.locale, record.presentation.description),
				Release:        localizedReleaseLifecycle(localization, projection.locale, record.lifecycle),
				LifecycleState: life,
			},
		}
		actions = append(actions, helpCommandCandidateValue)
	}

	assigned := make(map[CommandEntryID]struct{})
	var groups []HelpCommandGroup
	for _, group := range projection.helpGroups {
		if group.command != projection.target || group.family != helpGroupFamilyCommand ||
			!structuralProfileActive(projection.grammar.SelectedProfiles, projection.target, group.profile) {
			continue
		}
		public := HelpCommandGroup{
			Kind: HelpGroupDeclared,
			Classification: HelpGroupClassification{
				CommandPath: append([]string(nil), projection.scope...),
				Profile:     group.profile,
				SourceTitle: group.title,
			},
			Title: referenceText(
				projection.operation.LocalizationForCommand(projection.target),
				string(projection.locale),
				group.title,
			),
			Order: group.order,
		}
		for _, item := range group.CommandEntries() {
			switch item.Kind() {
			case HelpCommandGroupEntryCommandEntry:
				candidate, ok := candidates[item.CommandEntry()]
				if !ok {
					continue
				}
				if _, used := assigned[candidate.entry]; used {
					continue
				}
				public.Commands = append(public.Commands, candidate.value)
				assigned[candidate.entry] = struct{}{}
			case HelpCommandGroupEntryCommandProvider:
				for _, entryID := range projection.providers.batches[item.Provider()] {
					candidate, ok := candidates[entryID]
					if !ok {
						continue
					}
					if _, used := assigned[entryID]; used {
						continue
					}
					public.Commands = append(public.Commands, candidate.value)
					assigned[entryID] = struct{}{}
				}
			case HelpCommandGroupEntryUnknown:
			}
		}
		if public.Order == HelpGroupOrderName {
			slices.SortStableFunc(public.Commands, func(left, right HelpCommand) int {
				return cmp.Compare(left.Name, right.Name)
			})
		}
		if len(public.Commands) != 0 {
			groups = append(groups, public)
		}
	}

	var remaining []helpCommandCandidate
	for _, id := range candidateOrder {
		if _, used := assigned[id]; !used {
			remaining = append(remaining, candidates[id])
		}
	}
	remaining = append(remaining, actions...)
	slices.SortStableFunc(remaining, func(left, right helpCommandCandidate) int { return cmp.Compare(left.position, right.position) })
	if len(remaining) != 0 {
		group := HelpCommandGroup{
			Kind: HelpGroupDefault,
			Classification: HelpGroupClassification{
				CommandPath: append([]string(nil), projection.scope...),
			},
			Title: projection.page.Labels.Commands,
			Order: HelpGroupOrderDeclaration,
		}
		for _, candidate := range remaining {
			group.Commands = append(group.Commands, candidate.value)
		}
		groups = append(groups, group)
	}
	return groups
}

func helpCommandKind(kind CommandEntryKind) HelpCommandKind {
	switch kind {
	case CommandEntryCanonical:
		return HelpCommandCanonical
	case CommandEntryAlias:
		return HelpCommandAlias
	case CommandEntryShortcut:
		return HelpCommandShortcut
	default:
		return HelpCommandUnknown
	}
}

func (projection *helpProjection) commandRunnable(command CommandID) bool {
	record, ok := projection.model.Command(command)
	return ok && record.execution.Runnable()
}

func (projection *helpProjection) hasSubcommands(command CommandID) bool {
	for _, entry := range projection.grammarModel.CommandEntries(command) {
		if entry.enabled && structuralProfileActive(projection.grammar.SelectedProfiles, command, entry.profile) {
			return true
		}
	}
	providers := providerRecordsForOwner(projection.model, command)
	return len(effectiveProviderRecords(providers, command, projection.grammar.SelectedProfiles)) != 0
}

func (projection *helpProjection) actionLocalization(record actionEntryRecord) localizationSnapshot {
	if record.origin == DeclarationOriginFrontend {
		return localizationSnapshot{}
	}
	return projection.operation.LocalizationForCommand(record.parent)
}

func (projection *helpProjection) argumentForms() ([]HelpArgumentForm, error) {
	result := make([]HelpArgumentForm, 0, len(projection.frontier.forms))
	for _, form := range projection.frontier.forms {
		state := projection.states[argumentFormConditionTarget(form.id)]
		if !state.available || !state.visible {
			continue
		}
		record, ok := projection.model.ArgumentForm(form.id)
		if !ok {
			panic("cli: completed Help projection cannot resolve required declaration state")
		}
		life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
		if err != nil || !referenceAvailable(life) {
			continue
		}
		public := HelpArgumentForm{
			ID: form.id,
		}
		for _, term := range form.Terms() {
			if !term.Enabled() {
				continue
			}
			termState := projection.states[argumentTermConditionTarget(term.id)]
			if !termState.available || !termState.visible {
				continue
			}
			termRecord, ok := projection.model.ArgumentTerm(term.id)
			if !ok || termRecord.presentation.hidden {
				continue
			}
			termLife, err := helpLifecycleState(projection.scheme, termRecord.lifecycle, projection.release)
			if err != nil || !referenceAvailable(termLife) {
				continue
			}
			owner := record.command
			localization := projection.operation.LocalizationForCommand(owner)
			kind := term.kind
			typ, sequence := projection.argumentTermValueType(term)
			if sequence {
				kind = ArgumentTermSequence
			}
			publicTerm := HelpArgumentTerm{
				ID:      term.id,
				Kind:    kind,
				Name:    termRecord.name,
				Literal: term.literal,
				Description: referenceText(
					localization,
					string(projection.locale),
					termRecord.presentation.description.Short,
				),
				Type:            typ,
				Count:           countRange(term.count),
				Optional:        term.count.Minimum() == 0,
				Greedy:          term.allocation == semanticArgumentAllocationGreedy,
				Minimal:         term.allocation == semanticArgumentAllocationMinimal,
				EndOfOptions:    term.endOfOptions,
				Hidden:          false,
				Sensitive:       projection.termSensitive(term),
				SourceSensitive: termRecord.sourceSensitive,
				HyphenValues:    term.hyphenValues,
				Release:         localizedReleaseLifecycle(localization, projection.locale, termRecord.lifecycle),
				LifecycleState:  termLife,
			}
			if term.kind == ArgumentTermTokens {
				publicTerm.TokenLanguage = &TokenLanguageSyntax{
					Fragments: term.TokenSyntax(),
				}
			}
			if destination, present := term.Destination(); present {
				uses, err := projection.valueDomainUses(destination)
				if err != nil {
					return nil, err
				}
				publicTerm.ValueDomainUses = uses
			}
			public.Terms = append(public.Terms, publicTerm)
		}
		result = append(result, public)
	}
	return result, nil
}

func (projection *helpProjection) argumentTermValueType(term argumentTermGrammar) (ValueType, bool) {
	destination, present := term.Destination()
	if !present {
		return ValueType{}, false
	}
	destinationRecord, ok := projection.model.Destination(destination)
	if !ok {
		panic("cli: completed Help projection cannot resolve argument destination semantics")
	}
	definition, ok := destinationRecord.Definition()
	if !ok {
		panic("cli: completed Help projection cannot resolve argument destination semantics")
	}
	valueTypeDef, present := definition.ValueType()
	if !present {
		return ValueType{}, false
	}
	sequence := term.kind == ArgumentTermValue &&
		valueTypeDef.Kind == ValueKindSequence
	return valueTypeDef, sequence
}

func (projection *helpProjection) termSensitive(term argumentTermGrammar) bool {
	destination, ok := term.Destination()
	if !ok {
		return false
	}
	record, ok := projection.model.Destination(destination)
	return ok && record.sensitive
}

type helpFlagCandidate struct {
	flag     FlagID
	action   ActionEntryID
	command  CommandID
	scope    int
	position declarationPosition
	value    HelpFlag
}

func (projection *helpProjection) flagGroups() ([]HelpFlagGroup, error) {
	local := make(map[FlagID]helpFlagCandidate)
	var localOrder []FlagID
	var localActions []helpFlagCandidate
	var inherited []helpFlagCandidate
	pathRank := make(map[CommandID]int, len(projection.grammar.Selection.Path))
	for index, command := range projection.grammar.Selection.Path {
		pathRank[command] = index
	}
	for _, flag := range projection.frontier.flags {
		id := flag.id
		state := projection.states[flagConditionTarget(id)]
		if !state.available || !state.visible {
			continue
		}
		record, ok := projection.model.Flag(id)
		if !ok || record.presentation.hidden {
			continue
		}
		life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
		if err != nil || !referenceAvailable(life) {
			continue
		}
		spellings, err := projection.flagSpellings(flag)
		if err != nil {
			return nil, err
		}
		if len(spellings) == 0 {
			continue
		}
		domainUses, err := projection.valueDomainUses(record.destination)
		if err != nil {
			return nil, err
		}
		ownerPath := commandPathNames(projection.model, flag.command)
		origin := DeclarationOriginApplication
		if _, provider := projection.providers.commands[flag.command]; provider {
			origin = DeclarationOriginProvider
		}
		candidate := helpFlagCandidate{
			flag:     id,
			command:  flag.command,
			scope:    pathRank[flag.command],
			position: flag.position,
			value: HelpFlag{
				Origin:        origin,
				Flag:          id,
				FlagSpellings: spellings,
				ValueName:     record.valueName,
				Description: referenceText(
					projection.operation.LocalizationForCommand(flag.command),
					string(projection.locale),
					record.presentation.description.Short,
				),
				OwnerPath:           ownerPath,
				Occurrences:         countRange(flag.occurrences),
				ValuesPerOccurrence: countRange(flag.valuesPerOccurrence),
				Placement:           flag.placement,
				Required:            record.required,
				Sensitive:           projection.flagSensitive(record),
				LocaleSelector:      record.localeSelector,
				Release: localizedReleaseLifecycle(
					projection.operation.LocalizationForCommand(flag.command),
					projection.locale,
					record.lifecycle,
				),
				LifecycleState:  life,
				ValueDomainUses: domainUses,
			},
		}
		if flag.command == projection.target {
			local[id] = candidate
			localOrder = append(localOrder, id)
		} else {
			inherited = append(inherited, candidate)
		}
	}
	for _, command := range projection.grammar.Selection.Path {
		for _, action := range projection.grammarModel.ActionFlags(command) {
			if !action.enabled || (command != projection.target && action.placement != FlagPlacementSubtree) {
				continue
			}
			record, ok := projection.model.ActionEntry(action.id)
			if !ok || record.presentation.hidden {
				continue
			}
			life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
			if err != nil || !referenceAvailable(life) {
				continue
			}
			localization := projection.actionLocalization(record)
			candidate := helpFlagCandidate{
				action:   action.id,
				command:  command,
				scope:    pathRank[command],
				position: action.position,
				value: HelpFlag{
					Origin:      record.origin,
					ActionEntry: action.id,
					ActionSpelling: FlagSpelling{
						Kind:      action.kind,
						Text:      action.text,
						ValueMode: FlagValueModeNone,
					},
					Description: referenceText(
						localization,
						string(projection.locale),
						record.presentation.description.Short,
					),
					OwnerPath:           commandPathNames(projection.model, command),
					Occurrences:         Count.Optional(),
					ValuesPerOccurrence: Count.Exactly(0),
					Placement:           action.placement,
					Release:             localizedReleaseLifecycle(localization, projection.locale, record.lifecycle),
					LifecycleState:      life,
				},
			}
			if command == projection.target {
				localActions = append(localActions, candidate)
			} else {
				inherited = append(inherited, candidate)
			}
		}
	}

	assigned := make(map[FlagID]struct{})
	var result []HelpFlagGroup
	for _, group := range projection.helpGroups {
		if group.command != projection.target || group.family != helpGroupFamilyFlag ||
			!structuralProfileActive(projection.grammar.SelectedProfiles, projection.target, group.profile) {
			continue
		}
		public := HelpFlagGroup{
			Kind: HelpGroupDeclared,
			Classification: HelpGroupClassification{
				CommandPath: append([]string(nil), projection.scope...),
				Profile:     group.profile,
				SourceTitle: group.title,
			},
			Title: referenceText(
				projection.operation.LocalizationForCommand(projection.target),
				string(projection.locale),
				group.title,
			),
			Order: group.order,
		}
		for _, id := range group.Flags() {
			candidate, ok := local[id]
			if !ok {
				continue
			}
			if _, used := assigned[id]; used {
				continue
			}
			public.Flags = append(public.Flags, candidate.value)
			assigned[id] = struct{}{}
		}
		if public.Order == HelpGroupOrderName {
			slices.SortStableFunc(public.Flags, func(left, right HelpFlag) int {
				return cmp.Compare(
					helpFlagSortName(left, projection.page.FlagSpellings),
					helpFlagSortName(right, projection.page.FlagSpellings),
				)
			})
		}
		if len(public.Flags) != 0 {
			result = append(result, public)
		}
	}
	var remaining []helpFlagCandidate
	for _, id := range localOrder {
		if _, used := assigned[id]; !used {
			remaining = append(remaining, local[id])
		}
	}
	remaining = append(remaining, localActions...)
	slices.SortStableFunc(remaining, func(left, right helpFlagCandidate) int { return cmp.Compare(left.position, right.position) })
	if len(remaining) != 0 {
		group := HelpFlagGroup{
			Kind: HelpGroupDefault,
			Classification: HelpGroupClassification{
				CommandPath: append([]string(nil), projection.scope...),
			},
			Title: projection.page.Labels.Flags,
			Order: HelpGroupOrderDeclaration,
		}
		for _, candidate := range remaining {
			group.Flags = append(group.Flags, candidate.value)
		}
		result = append(result, group)
	}
	slices.SortStableFunc(inherited, func(left, right helpFlagCandidate) int {
		return cmp.Or(
			cmp.Compare(left.scope, right.scope),
			cmp.Compare(left.position, right.position),
		)
	})
	if len(inherited) != 0 {
		group := HelpFlagGroup{
			Kind: HelpGroupInherited,
			Classification: HelpGroupClassification{
				CommandPath: append([]string(nil), projection.scope...),
			},
			Title: projection.page.Labels.GlobalFlags,
			Order: HelpGroupOrderDeclaration,
		}
		for _, candidate := range inherited {
			group.Flags = append(group.Flags, candidate.value)
		}
		result = append(result, group)
	}
	return result, nil
}

func (projection *helpProjection) flagSensitive(flag flagRecord) bool {
	record, ok := projection.model.Destination(flag.destination)
	return ok && record.sensitive
}

func (projection *helpProjection) flagSpellings(flag flagGrammar) ([]FlagSpellingID, error) {
	grammarByID := make(map[FlagSpellingID]flagSpellingGrammar, len(flag.Spellings()))
	for _, spelling := range flag.Spellings() {
		grammarByID[spelling.id] = spelling
	}
	replacements := make(map[FlagSpellingID]FlagSpellingID)
	for _, replacement := range projection.model.ReplacementsForCommand(flag.command) {
		if replacement.source.Kind() != declarationReplacementKindFlagSpelling ||
			replacement.target.Kind() != declarationReplacementKindFlagSpelling {
			continue
		}
		source := replacement.source.FlagSpelling()
		target := replacement.target.FlagSpelling()
		if source.present() && target.present() {
			replacements[source] = target
		}
	}
	result := make([]FlagSpellingID, 0, len(flag.Spellings()))
	for _, spelling := range flag.Spellings() {
		record, ok := projection.model.FlagSpelling(spelling.id)
		if !ok {
			panic("cli: completed Help projection cannot resolve required declaration state")
		}
		life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
		if err != nil {
			return nil, err
		}
		if !referenceAvailable(life) || record.hidden {
			continue
		}
		if err := projection.ensureFlagSpelling(
			flag,
			grammarByID,
			replacements,
			spelling.id,
		); err != nil {
			return nil, err
		}
		result = append(result, spelling.id)
	}
	return result, nil
}

func (projection *helpProjection) ensureFlagSpelling(flag flagGrammar, grammarByID map[FlagSpellingID]flagSpellingGrammar, replacements map[FlagSpellingID]FlagSpellingID, id FlagSpellingID) error {
	publicID := id
	if _, exists := projection.page.FlagSpellings[publicID]; exists {
		return nil
	}
	record, ok := projection.model.FlagSpelling(id)
	if !ok || record.flag != flag.id {
		panic("cli: completed Help projection cannot resolve required declaration state")
	}
	spelling, ok := grammarByID[id]
	if !ok || spelling.flag != flag.id {
		panic("cli: completed Help projection cannot resolve required declaration state")
	}
	life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
	if err != nil {
		return err
	}
	publicSpelling := FlagSpelling{
		Kind:       spelling.kind,
		Text:       spelling.text,
		ValueMode:  spelling.mode,
		ValueForms: flagValueForms(spelling.Forms()),
		Sensitive:  spelling.sensitive,
	}
	if value, present, sensitive := spelling.FixedValue(); present {
		literal, err := projectEffectiveValueLiteral(projection.model, flag.destination, value, sensitive)
		if err != nil {
			return err
		}
		publicSpelling.FixedValue = &literal
	}
	if value, present, sensitive := spelling.ImplicitValue(); present {
		literal, err := projectEffectiveValueLiteral(projection.model, flag.destination, value, sensitive)
		if err != nil {
			return err
		}
		publicSpelling.ImplicitValue = &literal
	}
	replacement := replacements[id]
	localization := projection.operation.LocalizationForCommand(flag.command)
	projection.page.FlagSpellings[publicID] = HelpFlagSpelling{
		ID:             publicID,
		Spelling:       publicSpelling,
		Release:        localizedReleaseLifecycle(localization, projection.locale, record.lifecycle),
		LifecycleState: life,
		Replacement:    replacement,
	}
	if replacement.present() {
		if err := projection.ensureFlagSpelling(
			flag,
			grammarByID,
			replacements,
			replacement,
		); err != nil {
			return err
		}
	}
	return nil
}

func (projection *helpProjection) valueDomainUses(destination ValueID) ([]HelpValueDomainUse, error) {
	if !destination.present() {
		return nil, nil
	}
	localization := projection.model.Application().localization
	if record, ok := projection.model.Destination(destination); ok {
		localization = projection.operation.LocalizationForCommand(record.command)
	}
	var result []HelpValueDomainUse
	for _, domain := range projection.model.ValueDomains() {
		for _, use := range domain.Uses() {
			if use.destination != destination {
				continue
			}
			public := HelpValueDomainUse{
				Kind: use.kind,
			}
			for _, entryID := range domain.Entries() {
				record, ok := projection.model.DomainEntry(entryID)
				if !ok {
					panic("cli: completed Help projection cannot resolve required declaration state")
				}
				life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
				if err != nil {
					return nil, err
				}
				if !referenceAvailable(life) || record.hidden {
					continue
				}
				if err := projection.ensureValueDomainEntry(entryID, localization); err != nil {
					return nil, err
				}
				public.Entries = append(public.Entries, entryID)
			}
			result = append(result, public)
		}
	}
	return result, nil
}

func (projection *helpProjection) ensureValueDomainEntry(id ValueDomainEntryID, localization localizationSnapshot) error {
	publicID := id
	if _, exists := projection.page.ValueDomainEntries[publicID]; exists {
		return nil
	}
	record, ok := projection.model.DomainEntry(id)
	if !ok {
		panic("cli: completed Help projection cannot resolve required declaration state")
	}
	life, err := helpLifecycleState(projection.scheme, record.lifecycle, projection.release)
	if err != nil {
		return err
	}
	canonical := record.canonical
	canonicalRecord, ok := projection.model.DomainEntry(canonical)
	if !ok {
		panic("cli: completed Help projection cannot resolve required declaration state")
	}
	replacement, hasReplacement := record.Replacement()
	publicReplacement := ValueDomainEntryID(0)
	if hasReplacement {
		publicReplacement = replacement
	}
	projection.page.ValueDomainEntries[publicID] = HelpValueDomainEntry{
		ID:             publicID,
		Kind:           valueDomainEntryKind(record.kind),
		Name:           record.name,
		Canonical:      canonical,
		Description:    referenceText(localization, string(projection.locale), canonicalRecord.description),
		Release:        localizedReleaseLifecycle(localization, projection.locale, record.lifecycle),
		LifecycleState: life,
		Replacement:    publicReplacement,
	}
	if canonical != id {
		if err := projection.ensureValueDomainEntry(canonical, localization); err != nil {
			return err
		}
	}
	if hasReplacement {
		if err := projection.ensureValueDomainEntry(replacement, localization); err != nil {
			return err
		}
	}
	return nil
}

func helpFlagSortName(flag HelpFlag, spellings map[FlagSpellingID]HelpFlagSpelling) string {
	if flag.ActionEntry != 0 {
		return flag.ActionSpelling.Text
	}
	for _, kind := range []FlagSpellingKind{FlagSpellingLong, FlagSpellingLiteral, FlagSpellingShort} {
		for _, id := range flag.FlagSpellings {
			spelling, ok := spellings[id]
			if ok && spelling.Spelling.Kind == kind {
				return spelling.Spelling.Text
			}
		}
	}
	return ""
}
