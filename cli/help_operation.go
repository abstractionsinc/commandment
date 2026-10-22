package cli

import (
	"cmp"
	"context"
	"io"
	"maps"
	"slices"
)

type helpRenderingError struct{ cause error }

func (err *helpRenderingError) Error() string { return "render help: " + err.cause.Error() }

func (err *helpRenderingError) Unwrap() error { return err.cause }

type helpRequest struct {
	target CommandID

	selection CommandSelection

	requestedPath []string
	profiles      map[CommandID]GrammarProfileID
}

type helpAnalysis struct {
	request helpRequest

	availability map[conditionTarget]conditionOutcome
	defaults     map[CommandDefaultID]conditionOutcome
}

type helpTargetState struct {
	available bool
	visible   bool
}

type helpFrontier struct {
	targets map[conditionTarget]struct{}
	flags   []flagGrammar
	forms   []argumentFormGrammar
}

type helpRelevantFacts struct {
	conditions  []conditionPolicyRecord
	defaults    []defaultRecord
	localeFlags []flagRecord
	helpGroups  []helpGroupRecord
}

type helpRequirementSet struct {
	items   []dependencyRequirement
	byIndex map[ValueID]int
}

func (set *helpRequirementSet) add(dependency dependencyRequirement) {
	if index, ok := set.byIndex[dependency.Destination()]; ok {
		if set.items[index].Optional() && !dependency.Optional() {
			set.items[index] = acceptDependencyRequirement(dependency.Destination(), false)
		}
		return
	}
	if set.byIndex == nil {
		set.byIndex = make(map[ValueID]int)
	}
	set.byIndex[dependency.Destination()] = len(set.items)
	set.items = append(set.items, dependency)
}

func executeHelpAction(ctx context.Context, snapshot *runState, operation *invocation, model *effectiveModel, grammar grammarResult) error {
	if snapshot == nil || operation == nil || operation != snapshot.invocation {
		panic("cli: selected Help action has invalid invocation custody")
	}
	actions, ok := model.IntrinsicActions()
	if !ok {
		panic("cli: selected Help action has no accepted intrinsic actions")
	}
	help := actions.Help()
	if !help.Valid() || help.Disabled() {
		panic("cli: selected Help action has no enabled Help renderer")
	}
	request := deriveHelpRequest(model, grammar)
	return operation.showHelp(ctx, snapshot.stdout, grammar, request, help.Render)
}

func deriveHelpRequest(model *effectiveModel, grammar grammarResult) helpRequest {
	if !grammar.Selection.Selected.present() || grammar.Selection.ActionKind != intrinsicActionHelp {
		panic("cli: Help request requires an authoritative Help selection")
	}
	profiles := make(map[CommandID]GrammarProfileID, len(grammar.SelectedProfiles))
	maps.Copy(profiles, grammar.SelectedProfiles)
	request := helpRequest{
		target:        grammar.Selection.Selected,
		selection:     projectGrammarSelection(grammar.Selection),
		requestedPath: helpRequestedPath(model, grammar),
		profiles:      profiles,
	}
	return request
}

func (operation *invocation) showHelp(ctx context.Context, stdout io.Writer, grammar grammarResult, request helpRequest, renderer helpRender) error {
	if stdout == nil {
		panic("cli: Help selected without an output writer")
	}
	if renderer == nil {
		panic("cli: accepted Help action has no Help renderer")
	}

	analysis, err := operation.analyzeHelp(ctx, grammar, request)
	if err != nil {
		return err
	}

	values := operationDependencySeed(operation.analysis)
	locale, err := operation.selectHelpLocale(grammar, values)
	if err != nil {
		return err
	}
	operation.FinalizeFrameworkPresentationLocale(string(locale))

	page, err := operation.buildHelpPage(grammar, analysis, locale)
	if err != nil {
		return err
	}

	renderCtx := withInvocationObservationContext(ctx, page.Locale, page.Selection, true)
	renderCtx = withProfile(renderCtx, profileForWriter(stdout, destinationStdoutInformational))
	text, err := renderer(renderCtx, page)
	if err != nil {
		failure := &helpRenderingError{
			cause: err,
		}
		return operationDiagnosticFailure(
			operation,
			DiagnosticHelpRenderingFailure,
			"help rendering failure",
			failure,
		)
	}
	return writeFrameworkOutput(operation, stdout, text, "help")
}

func (operation *invocation) analyzeHelp(ctx context.Context, grammar grammarResult, request helpRequest) (helpAnalysis, error) {
	var result helpAnalysis
	if operation == nil || operation.analysis == nil {
		panic("cli: Help analysis requires an invocation analysis")
	}
	if request.target != grammar.Selection.Selected || request.selection.Selected != grammar.Selection.Selected {
		panic("cli: Help request does not match authoritative grammar selection")
	}

	if err := operation.settleHelpNamespace(ctx, grammar, request); err != nil {
		return result, err
	}

	model := operation.analysis.effectiveModel
	frontier, ok := buildHelpFrontier(model, grammar)
	if !ok {
		panic("cli: authoritative Help target has no Help frontier")
	}
	availability := helpGrammarAvailability(grammar)
	facts := collectHelpRelevantFacts(model, grammar, frontier, operation.input.locale == "")
	requirements := helpSemanticRequirements(model, facts, grammar, frontier, availability)
	values, err := operation.settleDependencies(ctx, grammar, requirements)
	if err != nil {
		return result, err
	}

	states := evaluateHelpTargetStates(model, facts.conditions, grammar, frontier, availability, values)
	if _, ok := states[commandConditionTarget(request.target)]; !ok {
		panic("cli: completed Help analysis lost its authoritative target")
	}
	defaults := evaluateHelpDefaultOutcomes(facts.defaults, grammar, request.target, values)

	result = helpAnalysis{
		request:      request,
		availability: availability,
		defaults:     defaults,
	}
	return result, nil
}

func (operation *invocation) settleHelpNamespace(ctx context.Context, grammar grammarResult, request helpRequest) error {
	model := operation.analysis.effectiveModel
	providers := effectiveProviderRecords(providerRecordsForOwner(model, request.target), request.target, request.profiles)
	if len(providers) == 0 {
		return nil
	}
	dependencies := providerDependencyUnion(providers)
	if len(dependencies) != 0 {
		if _, err := operation.settleDependencies(ctx, grammar, dependencies); err != nil {
			return err
		}
	}

	namespace := operation.namespace(ctx, grammarNamespaceDemand{
		Command: request.target,
		Grammar: grammar,
	})
	if code, err := operation.supplyFailure(); err != nil {
		return operationDiagnosticFailure(operation, code, "invocation value supply failed", err)
	}
	namespaces := operation.providerNamespaces()
	providerDiagnostics, providerCause := providerMaterializationDiagnostics(namespaces)
	if len(providerDiagnostics) != 0 {
		return providerDiagnosticFailure(operation, namespaces, providerCause)
	}
	if !namespace.Complete {
		outcome, ok := namespaces[request.target]
		if !ok {
			panic("cli: incomplete Help provider namespace has no outcome")
		}
		blocked := make([]ValueID, 0)
		for _, provider := range outcome.results {
			for _, destination := range provider.blockedBy {
				blocked = appendUniqueDestinationID(blocked, destination)
			}
		}
		if len(blocked) == 0 {
			panic("cli: incomplete Help provider namespace has no blocking dependency")
		}
		return blockedDependencyFailure(operation, operation.analysis.effectiveModel, operationDependencySeed(operation.analysis), blocked)
	}
	return nil
}

func helpGrammarAvailability(grammar grammarResult) map[conditionTarget]conditionOutcome {
	var availability map[conditionTarget]conditionOutcome
	for _, observation := range grammar.Conditions {
		if observation.Effect != ConditionEffectAvailability {
			continue
		}
		if availability == nil {
			availability = make(map[conditionTarget]conditionOutcome)
		}
		availability[observation.Target] = observation.Outcome
	}
	return availability
}

func buildHelpFrontier(model *effectiveModel, grammar grammarResult) (helpFrontier, bool) {
	grammarModel, ok := model.EffectiveGrammar()
	if !ok {
		return helpFrontier{}, false
	}
	frontier := helpFrontier{
		targets: make(map[conditionTarget]struct{}),
	}
	target := grammar.Selection.Selected
	frontier.targets[commandConditionTarget(target)] = struct{}{}

	for _, entry := range grammarModel.CommandEntries(target) {
		if !entry.enabled || !structuralProfileActive(grammar.SelectedProfiles, target, entry.profile) {
			continue
		}
		frontier.targets[commandConditionTarget(entry.target)] = struct{}{}
	}
	for _, command := range grammar.Selection.Path {
		for _, flag := range grammarModel.Flags(command) {
			if !flag.Enabled() || !structuralProfileActive(grammar.SelectedProfiles, command, flag.profile) {
				continue
			}
			if command != target && flag.placement != FlagPlacementSubtree {
				continue
			}
			frontier.flags = append(frontier.flags, flag)
			frontier.targets[flagConditionTarget(flag.id)] = struct{}{}
		}
	}
	for _, form := range grammarModel.ArgumentForms(target) {
		if !form.enabled || !structuralProfileActive(grammar.SelectedProfiles, target, form.profile) {
			continue
		}
		frontier.forms = append(frontier.forms, form)
		frontier.targets[argumentFormConditionTarget(form.id)] = struct{}{}
		for _, term := range form.Terms() {
			if !term.Enabled() {
				continue
			}
			frontier.targets[argumentTermConditionTarget(term.id)] = struct{}{}
		}
	}
	return frontier, true
}

func collectHelpRelevantFacts(model *effectiveModel, grammar grammarResult, frontier helpFrontier, includeLocaleSelector bool) helpRelevantFacts {
	owners := make(map[CommandID]struct{})
	for target := range frontier.targets {
		owner := structuralConditionOwner(model, target)
		if owner.present() {
			owners[owner] = struct{}{}
		}
	}
	conditionOwners := make([]CommandID, 0, len(owners))
	for owner := range owners {
		conditionOwners = append(conditionOwners, owner)
	}
	conditions := model.ConditionPoliciesForCommandsOrdered(conditionOwners)

	defaults := model.DefaultsForParent(grammar.Selection.Selected)
	var localeFlags []flagRecord
	if includeLocaleSelector {
		for _, command := range grammar.Selection.Path {
			localeFlags = append(localeFlags, model.FlagsForCommand(command)...)
		}
		slices.SortStableFunc(localeFlags, func(left, right flagRecord) int {
			return cmp.Compare(left.meta.position, right.meta.position)
		})
	}
	result := helpRelevantFacts{
		conditions:  conditions,
		defaults:    defaults,
		localeFlags: localeFlags,
		helpGroups:  model.HelpGroupsForCommand(grammar.Selection.Selected),
	}

	return result
}

func helpSemanticRequirements(model *effectiveModel, facts helpRelevantFacts, grammar grammarResult, frontier helpFrontier, availability map[conditionTarget]conditionOutcome) []dependencyRequirement {
	var requirements helpRequirementSet
	for _, record := range facts.conditions {
		if _, relevant := frontier.targets[record.target]; !relevant {
			continue
		}
		owner := structuralConditionOwner(model, record.target)
		if !structuralProfileActive(grammar.SelectedProfiles, owner, record.profile) {
			continue
		}
		if record.effect == ConditionEffectAvailability {
			if _, alreadySettled := availability[record.target]; alreadySettled {
				continue
			}
		} else if record.effect != ConditionEffectVisibility {
			continue
		}
		for _, dependency := range record.condition.Dependencies() {
			requirements.add(acceptDependencyRequirement(dependency, false))
		}
	}
	for _, record := range facts.defaults {
		if record.parent != grammar.Selection.Selected ||
			!structuralProfileActive(grammar.SelectedProfiles, record.parent, record.meta.profile) {
			continue
		}
		condition, present := record.Condition()
		if !present {
			continue
		}
		for _, dependency := range condition.Dependencies() {
			requirements.add(acceptDependencyRequirement(dependency, false))
		}
	}
	for _, flag := range facts.localeFlags {
		if !flag.localeSelector ||
			!structuralProfileActive(grammar.SelectedProfiles, flag.command, flag.meta.profile) {
			continue
		}
		requirements.add(acceptDependencyRequirement(flag.destination, false))
	}
	return requirements.items
}

func evaluateHelpTargetStates(model *effectiveModel, conditions []conditionPolicyRecord, grammar grammarResult, frontier helpFrontier, availability map[conditionTarget]conditionOutcome, values valueAnalysis) map[conditionTarget]helpTargetState {
	states := make(map[conditionTarget]helpTargetState, len(frontier.targets))
	for target := range frontier.targets {
		states[target] = helpTargetState{
			available: true,
			visible:   true,
		}
	}
	for _, record := range conditions {
		state, relevant := states[record.target]
		if !relevant {
			continue
		}
		owner := structuralConditionOwner(model, record.target)
		if !structuralProfileActive(grammar.SelectedProfiles, owner, record.profile) {
			continue
		}
		var outcome conditionOutcome
		switch record.effect {
		case ConditionEffectAvailability:
			var ok bool
			outcome, ok = availability[record.target]
			if !ok {
				outcome = evaluateCondition(record.condition, values)
			}
		case ConditionEffectVisibility:
			outcome = evaluateCondition(record.condition, values)
		default:
			continue
		}
		switch outcome {
		case conditionOutcomeSatisfied:
		case conditionOutcomeUnsatisfied:
			if record.effect == ConditionEffectAvailability {
				state.available = false
				state.visible = false
			} else {
				state.visible = false
			}
		case conditionOutcomeBlocked,
			conditionOutcomeUnsupported,
			conditionOutcomeUnknown:
			panic("cli: completed Help condition analysis is incomplete")
		default:
			panic("cli: completed Help condition analysis has invalid outcome")
		}
		states[record.target] = state
	}
	return states
}

func evaluateHelpDefaultOutcomes(defaults []defaultRecord, grammar grammarResult, target CommandID, values valueAnalysis) map[CommandDefaultID]conditionOutcome {
	var result map[CommandDefaultID]conditionOutcome
	for _, record := range defaults {
		if record.parent != target || !structuralProfileActive(grammar.SelectedProfiles, target, record.meta.profile) {
			continue
		}
		outcome := conditionOutcomeSatisfied
		if condition, present := record.Condition(); present {
			outcome = evaluateCondition(condition, values)
		}
		switch outcome {
		case conditionOutcomeSatisfied, conditionOutcomeUnsatisfied:
		case conditionOutcomeBlocked,
			conditionOutcomeUnsupported,
			conditionOutcomeUnknown:
			panic("cli: completed Help default analysis is incomplete")
		default:
			panic("cli: completed Help default analysis has invalid outcome")
		}
		if result == nil {
			result = make(map[CommandDefaultID]conditionOutcome)
		}
		result[record.meta.id] = outcome
	}
	return result
}

func helpDefaultTargets(model *effectiveModel, target CommandID, profiles map[CommandID]GrammarProfileID, outcomes map[CommandDefaultID]conditionOutcome) map[CommandID]struct{} {
	var result map[CommandID]struct{}
	for _, record := range model.DefaultsForParent(target) {
		if !structuralProfileActive(profiles, target, record.meta.profile) {
			continue
		}
		outcome, ok := outcomes[record.meta.id]
		if !ok {
			panic("cli: completed Help analysis is missing a default outcome")
		}
		if outcome != conditionOutcomeSatisfied {
			continue
		}
		if result == nil {
			result = make(map[CommandID]struct{})
		}
		result[record.target] = struct{}{}
	}
	return result
}

func (operation *invocation) selectHelpLocale(grammar grammarResult, values valueAnalysis) (Locale, error) {
	model := operation.analysis.effectiveModel
	selector := Locale("")
	if operation.input.locale == "" {
		var err error
		selector, err = resolvedLocaleSelector(model, grammar, values)
		if err != nil {
			return "", operationDiagnosticFailure(
				operation,
				DiagnosticInvalidValue,
				"invalid locale selector",
				err,
			)
		}
	}
	locale, err := selectEffectiveLocale(
		string(operation.input.locale),
		string(selector),
		model.Application().localization.SourceLocale(),
	)
	if err != nil {
		return "", operationDiagnosticFailure(
			operation,
			DiagnosticInvalidValue,
			"invalid invocation locale",
			err,
		)
	}
	return Locale(locale), nil
}
