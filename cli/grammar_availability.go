package cli

import (
	"context"
	"maps"
	"slices"
)

// effectiveScope exposes one operation's selected-profile and lifecycle grammar
// through bounded namespace reads. It never reconstructs a complete GrammarModel;
// the underlying application grammar facet is captured at one accepted-provider
// generation so parser caches cannot cross publication frontiers.
func (source grammarOperationSource) effectiveScope(model grammarView, selected map[CommandID]GrammarProfileID) grammarView {
	generation := 0
	if source.analysis != nil && source.analysis.effectiveModel != nil {
		if captured, ok := source.analysis.effectiveModel.GrammarGeneration(model); ok {
			generation = captured
		}
	}
	return analysisEffectiveGrammar(source, model, generation, selected)
}

// structuralAvailabilityAt settles one effective availability condition exactly
// once for this grammar operation. Unsatisfied preserves declaration identity
// while disabling participation; blocked/unsupported is an analysis failure.
func structuralAvailabilityAt(operation *invocation, analysis *analysisState, grammar *grammarResult, materialize grammarConditionDemandFunc, target conditionTarget, tokenIndex int) (bool, bool) {
	policy, ok := effectiveAvailabilityPolicyAt(analysis, target, grammar.SelectedProfiles)
	if !ok {
		return true, true
	}
	for _, observation := range grammar.Conditions {
		if observation.Target == target && observation.Effect == ConditionEffectAvailability {
			switch observation.Outcome {
			case conditionOutcomeSatisfied:
				return true, true
			case conditionOutcomeUnsatisfied:
				return false, true
			default:
				return false, false
			}
		}
	}
	outcome := conditionOutcomeUnsupported
	if materialize != nil {
		outcome = materialize(grammarConditionDemand{
			Policy:  policy,
			Grammar: grammarDemandSnapshot(*grammar),
		})
	}
	observation2 := grammarConditionObservation{
		Target:  target,
		Effect:  ConditionEffectAvailability,
		Outcome: outcome,
	}
	grammar.Conditions = append(grammar.Conditions, observation2)
	switch outcome {
	case conditionOutcomeSatisfied:
		return true, true
	case conditionOutcomeUnsatisfied:
		return false, true
	default:
		grammarIssueValue := grammarIssue{
			Kind:            grammarIssueConditionEvaluation,
			TokenIndex:      tokenIndex,
			Command:         analysisConditionTargetCommandAt(analysis, target),
			ConditionTarget: target,
		}
		appendGrammarIssue(grammar, operation.input.limits.Diagnostics, grammarIssueValue)
		return false, false
	}
}

func effectiveAvailabilityPolicyAt(analysis *analysisState, target conditionTarget, selected map[CommandID]GrammarProfileID) (conditionPolicyRecord, bool) {
	owner, records := availabilityPolicyRecordsAt(analysis, target)
	for _, record := range records {
		if record.target != target || record.effect != ConditionEffectAvailability {
			continue
		}
		if !record.profile.present() || selected[owner] == record.profile {
			return record, true
		}
	}
	return conditionPolicyRecord{}, false
}

func availabilityPolicyRecordsAt(analysis *analysisState, target conditionTarget) (CommandID, []conditionPolicyRecord) {
	owner := analysisConditionTargetCommandAt(analysis, target)
	if owner.present() {
		// A valid owner makes the scoped read authoritative even when it is
		// empty. Falling back on an empty scoped result would make unrelated
		// sibling policies participate in ordinary operation analysis.
		return owner, analysis.effectiveModel.ConditionPoliciesForCommand(owner)
	}

	// An accepted application always has the declaration record required to
	// recover the owner. Missing ownership is invalid operation-effective semantics; never
	// broaden the query to unrelated policies.
	return CommandID(0), nil
}

func analysisConditionTargetCommandAt(analysis *analysisState, target conditionTarget) CommandID {
	switch target.Kind() {
	case conditionTargetCommand:
		return target.Command()
	case conditionTargetFlag:
		if record, ok := analysis.effectiveModel.Flag(target.Flag()); ok {
			return record.command
		}
	case conditionTargetArgumentForm:
		if record, ok := analysis.effectiveModel.ArgumentForm(target.ArgumentForm()); ok {
			return record.command
		}
	case conditionTargetArgumentTerm:
		if term, ok := analysis.effectiveModel.ArgumentTerm(target.ArgumentTerm()); ok {
			if form, exists := analysis.effectiveModel.ArgumentForm(term.form); exists {
				return form.command
			}
		}
	case conditionTargetUnknown:
	}
	return CommandID(0)
}

func applyCommandAvailability(operation *invocation, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, command CommandID, dialect dialect, token string, tokenIndex int) (grammarView, bool) {
	return applyCommandAvailabilityAt(operation, operation.analysis, grammar, materialize, model, command, dialect, token, tokenIndex)
}

func applyCommandAvailabilityAt(operation *invocation, analysis *analysisState, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, command CommandID, dialect dialect, token string, tokenIndex int) (grammarView, bool) {
	// Settle only lexical competitors for this token. Each shortcut keeps its
	// canonical target path ordered so availability stops at the same first
	// unreachable command as ordinary canonical traversal.
	match := matchCommandChild(model, command, dialect, token)
	if !match.found {
		return model, true
	}

	entries := make(map[CommandEntryID]commandEntryGrammar, len(match.entries))
	for _, entry := range model.CommandEntries(command) {
		entries[entry.id] = entry
	}
	paths := make([][]CommandID, 0, len(match.entries))
	for _, id := range match.entries {
		entry, ok := entries[id]
		if !ok {
			grammarIssueValue := grammarIssue{
				Kind:       grammarIssueBlockedExplicitEntry,
				TokenIndex: tokenIndex,
				Command:    command,
				Entries:    match.entries,
			}
			appendGrammarIssue(grammar, operation.input.limits.Diagnostics, grammarIssueValue)
			return model, false
		}
		if entry.kind != CommandEntryShortcut {
			paths = append(paths, []CommandID{entry.target})
			continue
		}
		path, err := canonicalCommandPath(analysis.effectiveModel, model, command, entry.target)
		if err != nil || len(path) == 0 {
			grammarIssueValue2 := grammarIssue{
				Kind:       grammarIssueBlockedExplicitEntry,
				TokenIndex: tokenIndex,
				Command:    command,
				Entries:    match.entries,
			}
			appendGrammarIssue(grammar, operation.input.limits.Diagnostics, grammarIssueValue2)
			return model, false
		}
		commands := make([]CommandID, len(path))
		for index, step := range path {
			commands[index] = step.Command
		}
		paths = append(paths, commands)
	}
	if len(paths) == 0 {
		return model, true
	}

	states := make(map[CommandID]bool, len(paths))
	for _, path := range paths {
		enabled := true
		for _, target := range path {
			targetEnabled, settled := structuralAvailabilityAt(
				operation,
				analysis,
				grammar,
				materialize,
				commandConditionTarget(target),
				tokenIndex,
			)
			if !settled {
				return model, false
			}
			if !targetEnabled {
				enabled = false
				break
			}
		}
		terminal := path[len(path)-1]
		if previous, exists := states[terminal]; exists {
			states[terminal] = previous && enabled
		} else {
			states[terminal] = enabled
		}
	}
	availability := availabilityGrammar{
		base:     model,
		commands: states,
	}
	return availability, true
}

func applyFlagAvailability(operation *invocation, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, path []CommandID, dialect dialect, token string, tokenIndex int) (grammarView, bool) {
	return applyFlagAvailabilityAt(operation, operation.analysis, grammar, materialize, model, path, dialect, token, tokenIndex)
}

func applyFlagAvailabilityAt(operation *invocation, analysis *analysisState, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, path []CommandID, dialect dialect, token string, tokenIndex int) (grammarView, bool) {
	frontier := buildSelectedPathFlagFrontier(model, path)
	return applyFlagAvailabilityWithFrontierAt(
		operation,
		analysis,
		grammar,
		materialize,
		model,
		frontier,
		dialect,
		token,
		tokenIndex,
	)
}

func addShortClusterCandidateFlags(candidateIDs map[FlagID]struct{}, cluster shortClusterMatch) {
	if !cluster.recognized {
		return
	}
	for _, member := range cluster.members {
		candidateIDs[member.flag.id] = struct{}{}
	}
	for _, flag := range cluster.candidateFlags {
		candidateIDs[flag] = struct{}{}
	}
}

func applyFlagAvailabilityWithFrontierAt(operation *invocation, analysis *analysisState, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, frontier selectedPathFlagFrontier, dialect dialect, token string, tokenIndex int) (grammarView, bool) {
	if len(frontier.active) == 0 {
		return model, true
	}
	candidateIDs := make(map[FlagID]struct{})
	if dialect.ShortClustering() == ShortFlagClusteringEnabled {
		addShortClusterCandidateFlags(candidateIDs, matchShortClusterFrontier(model, frontier, dialect, token))
	}
	match := matchFlagsFrontier(model, frontier, dialect, token)
	if match.found {
		for _, flag := range match.candidateFlags {
			candidateIDs[flag] = struct{}{}
		}
	}
	if len(candidateIDs) == 0 {
		return model, true
	}
	states := make(map[FlagID]bool, len(candidateIDs))
	for id := range candidateIDs {
		enabled, settled := structuralAvailabilityAt(
			operation,
			analysis,
			grammar,
			materialize,
			flagConditionTarget(id),
			tokenIndex,
		)
		if !settled {
			return model, false
		}
		states[id] = enabled
	}
	availability := availabilityGrammar{
		base:  model,
		flags: states,
	}
	return availability, true
}

func applyRequiredFlagAvailabilityAt(operation *invocation, analysis *analysisState, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, path []CommandID, tokenIndex int) (grammarView, bool) {
	// Requiredness is itself a grammar consequence. An unused flag whose
	// availability is conditional must settle before missing-occurrence
	// diagnostics are emitted; an unavailable declaration is not a missing one.
	targets := make(map[FlagID]bool)
	for _, command := range path {
		for _, flag := range model.Flags(command) {
			if !flag.Enabled() || flag.occurrences.Minimum() == 0 {
				continue
			}
			enabled, settled := structuralAvailabilityAt(
				operation,
				analysis,
				grammar,
				materialize,
				flagConditionTarget(flag.id),
				tokenIndex,
			)
			if !settled {
				return model, false
			}
			targets[flag.id] = enabled
		}
	}
	if len(targets) == 0 {
		return model, true
	}
	availability := availabilityGrammar{
		base:  model,
		flags: targets,
	}
	return availability, true
}

func applyArgumentAvailability(operation *invocation, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, command CommandID, tokenIndex int) (grammarView, bool) {
	return applyArgumentAvailabilityAt(operation, operation.analysis, grammar, materialize, model, command, tokenIndex)
}

func applyArgumentAvailabilityAt(operation *invocation, analysis *analysisState, grammar *grammarResult, materialize grammarConditionDemandFunc, model grammarView, command CommandID, tokenIndex int) (grammarView, bool) {
	formStates := make(map[ArgumentFormID]bool)
	termStates := make(map[ArgumentTermID]bool)
	for _, form := range model.ArgumentForms(command) {
		enabled, settled := structuralAvailabilityAt(
			operation,
			analysis,
			grammar,
			materialize,
			argumentFormConditionTarget(form.id),
			tokenIndex,
		)
		if !settled {
			return model, false
		}
		formStates[form.id] = enabled
		for _, term := range form.Terms() {
			termEnabled, termSettled := structuralAvailabilityAt(
				operation,
				analysis,
				grammar,
				materialize,
				argumentTermConditionTarget(term.id),
				tokenIndex,
			)
			if !termSettled {
				return model, false
			}
			termStates[term.id] = termEnabled
		}
	}
	if len(formStates) == 0 && len(termStates) == 0 {
		return model, true
	}
	result := availabilityGrammar{
		base:          model,
		argumentForms: formStates,
		argumentTerms: termStates,
	}
	return result, true
}

func flagAvailabilityDependsOnProfileAt(analysis *analysisState, flag FlagID) bool {
	record, ok := analysis.effectiveModel.Flag(flag)
	var policies []conditionPolicyRecord
	if ok {
		policies = analysis.effectiveModel.ConditionPoliciesForCommand(record.command)
	} else {
		policies = analysis.effectiveModel.ConditionPolicies()
	}
	for _, policy := range policies {
		if policy.effect == ConditionEffectAvailability &&
			policy.target.Kind() == conditionTargetFlag &&
			policy.target.Flag() == flag &&
			policy.profile.present() {
			return true
		}
	}
	return false
}

func grammarProfileRecordsAt(analysis *analysisState, command CommandID) []grammarProfileRecord {
	return analysis.effectiveModel.GrammarProfilesForCommand(command)
}

func grammarProfilePendingAt(analysis *analysisState, command CommandID, selected map[CommandID]GrammarProfileID) bool {
	return len(grammarProfileRecordsAt(analysis, command)) != 0 && !selected[command].present()
}

func grammarProfilePending(operation *invocation, command CommandID, selected map[CommandID]GrammarProfileID) bool {
	return grammarProfilePendingAt(operation.analysis, command, selected)
}

func grammarProfileSelectorAt(analysis *analysisState, command CommandID) ValueID {
	for _, profile := range grammarProfileRecordsAt(analysis, command) {
		if selector, ok := profile.Selector(); ok {
			return selector
		}
	}
	return ValueID(0)
}

func grammarProfileSelector(operation *invocation, command CommandID) ValueID {
	return grammarProfileSelectorAt(operation.analysis, command)
}

func profilePrefixShortClusterCanConsumeAt(analysis *analysisState, cluster shortClusterMatch) (bool, bool) {
	if !cluster.recognized {
		return false, false
	}
	if len(cluster.ambiguous) != 0 || cluster.issueAt >= 0 {
		return true, true // Let the ordinary recognizer publish the exact rejection.
	}
	for _, member := range cluster.members {
		if flagAvailabilityDependsOnProfileAt(analysis, member.flag.id) || !profilePrefixFlagFixed(member.flag, member.spelling, member.suffix != "") {
			return false, true
		}
	}
	return true, true
}

func profilePrefixCanConsume(operation *invocation, model grammarView, path []CommandID, dialect dialect, token string) bool {
	return profilePrefixCanConsumeAt(operation.analysis, model, path, dialect, token)
}

func profilePrefixCanConsumeAt(analysis *analysisState, model grammarView, path []CommandID, dialect dialect, token string) bool {
	if isEndOfOptions(dialect, token) {
		return false
	}
	if dialect.ShortClustering() == ShortFlagClusteringEnabled {
		if consumable, recognized := profilePrefixShortClusterCanConsumeAt(analysis, matchShortCluster(model, path, dialect, token)); recognized {
			return consumable
		}
	}
	match := matchFlag(model, path, dialect, token)
	if match.found && match.flag.id.present() && flagAvailabilityDependsOnProfileAt(analysis, match.flag.id) {
		return false
	}
	if !match.found || match.action.id.present() {
		return false
	}
	if match.ambiguous {
		return true // Structural ambiguity is diagnosed before profile selection.
	}
	return profilePrefixFlagFixed(
		match.flag,
		match.spelling,
		match.form == flagValueFormAssigned || match.form == flagValueFormJoined,
	)
}

func profilePrefixFlagFixed(flag flagGrammar, spelling flagSpellingGrammar, attached bool) bool {
	if spelling.mode == FlagValueModeNone {
		return true
	}
	if spelling.mode != FlagValueModeRequired {
		return false
	}
	if flag.tokenLanguageName != "" {
		return false
	}
	if _, ok := flag.Delimiter(); ok {
		return false
	}
	if _, ok := flag.Terminator(); ok {
		return false
	}
	minimum := flag.valuesPerOccurrence.Minimum()
	maximum, bounded := flag.valuesPerOccurrence.Maximum()
	if !bounded || minimum != maximum {
		return false
	}
	if attached {
		return minimum == 1
	}
	return true
}

// resolveGrammarProfile resolves the selector through the ordinary scalar
// candidate precedence using only the producer families authorized before
// profile-dependent ownership: already-owned explicit flag input, environment,
// and a static default. The resulting selector observation is stored as the
// operation dependency seed so downstream value analysis does not re-resolve it.
func resolveGrammarProfile(ctx context.Context, operation *invocation, grammar *grammarResult, model grammarView, args []string, command CommandID, tokenIndex int) bool {
	return resolveGrammarProfileAt(ctx, operation, operation.analysis, grammar, model, args, command, tokenIndex)
}

func resolveGrammarProfileAt(ctx context.Context, operation *invocation, analysisState *analysisState, grammar *grammarResult, model grammarView, args []string, command CommandID, tokenIndex int) bool {
	profiles := grammarProfileRecordsAt(analysisState, command)
	if len(profiles) == 0 {
		return true
	}
	selector := grammarProfileSelectorAt(analysisState, command)
	record, ok := analysisState.effectiveModel.Destination(selector)
	if !ok {
		grammarIssueValue := grammarIssue{
			Kind:       grammarIssueProfileSelection,
			TokenIndex: tokenIndex,
			Command:    command,
		}
		appendGrammarIssue(grammar, operation.input.limits.Diagnostics, grammarIssueValue)
		return false
	}
	analysis := operationDependencySeed(analysisState)
	if existing, exists := analysis.Destinations[selector]; !exists {
		selectedCommands := make(map[CommandID]struct{}, len(grammar.Selection.Path))
		for _, selectedCommand := range grammar.Selection.Path {
			selectedCommands[selectedCommand] = struct{}{}
		}
		include := map[ValueID]struct{}{selector: {}}
		plans := make(map[ValueID][]valueCandidatePlan)
		builder := valuePlanBuilder{
			state:    analysisState,
			profiles: grammar.SelectedProfiles,
			selected: selectedCommands,
			include:  include,
			plans:    plans,
			analysis: &analysis,
		}
		builder.appendGrammarValueCandidatePlansFromArgs(ctx, operation, model, args, *grammar)
		builder.appendEnvironmentValueCandidatePlans(operation)
		resolveDestination(
			ctx,
			operation,
			analysisState,
			prepareValueSettlementNode(record, grammar.SelectedProfiles),
			plans[selector],
			&analysis,
		)
		storeOperationDependencySeed(analysisState, analysis)
	} else {
		_ = existing
	}
	resolution, ok := analysis.Destinations[selector]
	if !ok || resolution.State != destinationStateResolved {
		grammarIssueValue2 := grammarIssue{
			Kind:       grammarIssueProfileSelection,
			TokenIndex: tokenIndex,
			Command:    command,
		}
		appendGrammarIssue(grammar, operation.input.limits.Diagnostics, grammarIssueValue2)
		return false
	}
	var fallback GrammarProfileID
	for _, profile := range profiles {
		if profile.defaultCase {
			fallback = profile.meta.id
			continue
		}
		_, present := profile.Value()
		if present && profile.MatchesValue(resolution.Value) {
			if grammar.SelectedProfiles == nil {
				grammar.SelectedProfiles = make(map[CommandID]GrammarProfileID)
			}
			grammar.SelectedProfiles[command] = profile.meta.id
			return true
		}
	}
	if fallback.present() {
		if grammar.SelectedProfiles == nil {
			grammar.SelectedProfiles = make(map[CommandID]GrammarProfileID)
		}
		grammar.SelectedProfiles[command] = fallback
		return true
	}
	grammarIssueValue3 := grammarIssue{
		Kind:       grammarIssueProfileSelection,
		TokenIndex: tokenIndex,
		Command:    command,
	}
	appendGrammarIssue(grammar, operation.input.limits.Diagnostics, grammarIssueValue3)
	return false
}

func grammarDemandSnapshot(result grammarResult) grammarResult {
	snapshot := result
	snapshot.Selection.Path = append([]CommandID(nil), result.Selection.Path...)
	snapshot.Selection.Steps = append([]selectionStep(nil), result.Selection.Steps...)
	for index := range snapshot.Selection.Steps {
		snapshot.Selection.Steps[index].Commands = append([]CommandID(nil), result.Selection.Steps[index].Commands...)
	}
	snapshot.Ownership = append([]tokenOwnership(nil), result.Ownership...)
	snapshot.Occurrences = append([]flagOccurrence(nil), result.Occurrences...)
	for index := range snapshot.Occurrences {
		snapshot.Occurrences[index].ValueIndices = append([]int(nil), result.Occurrences[index].ValueIndices...)
	}
	snapshot.Issues = append([]grammarIssue(nil), result.Issues...)
	snapshot.Conditions = append([]grammarConditionObservation(nil), result.Conditions...)
	for index := range snapshot.Issues {
		snapshot.Issues[index].Entries = append([]CommandEntryID(nil), result.Issues[index].Entries...)
		snapshot.Issues[index].Actions = append([]ActionEntryID(nil), result.Issues[index].Actions...)
		snapshot.Issues[index].Flags = append([]FlagSpellingID(nil), result.Issues[index].Flags...)
		snapshot.Issues[index].FlagDeclarations = append([]FlagID(nil), result.Issues[index].FlagDeclarations...)
		snapshot.Issues[index].Forms = append([]ArgumentFormID(nil), result.Issues[index].Forms...)
	}
	snapshot.IncompleteNamespaces = append([]CommandID(nil), result.IncompleteNamespaces...)
	snapshot.SelectedProfiles = nil
	if len(result.SelectedProfiles) != 0 {
		snapshot.SelectedProfiles = make(
			map[CommandID]GrammarProfileID,
			len(result.SelectedProfiles),
		)
		maps.Copy(snapshot.SelectedProfiles, result.SelectedProfiles)
	}
	snapshot.DefaultSelections = nil
	if len(result.DefaultSelections) != 0 {
		snapshot.DefaultSelections = make(map[CommandID]defaultSelection, len(result.DefaultSelections))
		for command, selection := range result.DefaultSelections {
			selection.Candidates = append([]CommandDefaultID(nil), selection.Candidates...)
			selection.Applicable = append([]CommandDefaultID(nil), selection.Applicable...)
			selection.Blocked = append([]CommandDefaultID(nil), selection.Blocked...)
			snapshot.DefaultSelections[command] = selection
		}
	}
	snapshot.DefaultConditions = nil
	if len(result.DefaultConditions) != 0 {
		snapshot.DefaultConditions = make(map[CommandDefaultID]conditionOutcome, len(result.DefaultConditions))
		maps.Copy(snapshot.DefaultConditions, result.DefaultConditions)
	}
	snapshot.Tokens = nil
	snapshot.Arguments = argumentAllocation{}
	snapshot.ArgumentForms = nil
	snapshot.tokenOverrides = nil
	return snapshot
}

func appendUniqueCommandID(values []CommandID, value CommandID) []CommandID {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func cloneFlagOccurrence(occurrence *flagOccurrence) *flagOccurrence {
	if occurrence == nil {
		return nil
	}
	cloned := *occurrence
	cloned.ValueIndices = append([]int(nil), occurrence.ValueIndices...)
	return &cloned
}

func activateAutomaticHelpAtBoundaryAt(operation *invocation, analysis *analysisState, result *grammarResult, current CommandID, emptyInput bool) {
	if operation == nil || result == nil || result.Selection.ActionSelected() {
		return
	}
	actions, ok := analysis.effectiveModel.IntrinsicActions()
	if !ok {
		return
	}
	help := actions.Help()
	if !help.Valid() || help.Disabled() {
		return
	}
	if emptyInput && help.WhenEmpty() {
		result.Selection.ActionKind = intrinsicActionHelp
		result.Selection.ActionTrigger = actionTriggerEmpty
		return
	}
	command, ok := analysis.effectiveModel.Command(current)
	if !ok || command.execution.Runnable() {
		return
	}
	result.Selection.ActionKind = intrinsicActionHelp
	result.Selection.ActionTrigger = actionTriggerStructural
}

func materializeDefaultBoundaryNamespaceAt(operation *invocation, analysis *analysisState, result *grammarResult, model *grammarView, current CommandID, tokenIndex int, materialize grammarNamespaceDemandFunc, demandedNamespaces *map[CommandID]struct{}) bool {
	if materialize == nil {
		return true
	}
	if _, demanded := (*demandedNamespaces)[current]; demanded {
		return true
	}
	if *demandedNamespaces == nil {
		*demandedNamespaces = make(map[CommandID]struct{})
	}
	(*demandedNamespaces)[current] = struct{}{}
	state := materialize(grammarNamespaceDemand{
		Command: current,
		Grammar: grammarDemandSnapshot(*result),
	})
	if state.Changed {
		refreshed, ok := analysis.effectiveModel.EffectiveGrammar()
		if !ok {
			grammarIssueValue := grammarIssue{
				Kind:       grammarIssueMissingDialect,
				TokenIndex: tokenIndex,
				Command:    current,
			}
			appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
			return false
		}
		*model = refreshed
	}
	if state.Complete {
		return true
	}
	result.IncompleteNamespaces = appendUniqueCommandID(result.IncompleteNamespaces, current)
	return false
}

func settleDefaultChainAtBoundaryAt(ctx context.Context, operation *invocation, analysis *analysisState, result *grammarResult, model *grammarView, args []string, current CommandID, tokenIndex int, materialize grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc, demandedNamespaces *map[CommandID]struct{}) (CommandID, bool) {
	for {
		if grammarProfilePendingAt(analysis, current, result.SelectedProfiles) {
			view := grammarOperationSource{
				operation: operation,
				analysis:  analysis,
			}.effectiveScope(*model, result.SelectedProfiles)
			if !resolveGrammarProfileAt(ctx, operation, analysis, result, view, args, current, tokenIndex) {
				return current, false
			}
		}

		if !materializeDefaultBoundaryNamespaceAt(operation, analysis, result, model, current, tokenIndex, materialize, demandedNamespaces) {
			return current, false
		}

		view := grammarOperationSource{
			operation: operation,
			analysis:  analysis,
		}.effectiveScope(*model, result.SelectedProfiles)
		next, selected := selectDefaultCommandAt(
			operation,
			analysis,
			result,
			defaults,
			condition,
			view,
			current,
			tokenIndex,
		)
		if !selected {
			return current, false
		}
		if next == current {
			return current, true
		}
		current = next
	}
}

func cloneGrammarPrefix(prefix grammarPrefixResult) grammarPrefixResult {
	cloned := prefix
	cloned.Grammar = grammarDemandSnapshot(prefix.Grammar)
	if len(prefix.FlagOccurrences) != 0 {
		cloned.FlagOccurrences = make(map[FlagID]int, len(prefix.FlagOccurrences))
		maps.Copy(cloned.FlagOccurrences, prefix.FlagOccurrences)
	}
	cloned.PendingFlagValue = cloneFlagOccurrence(prefix.PendingFlagValue)
	cloned.PositionalIndices = append([]int(nil), prefix.PositionalIndices...)
	return cloned
}

func markDefaultsNotEvaluatedAt(analysis *analysisState, result *grammarResult, command CommandID) {
	if _, exists := result.DefaultSelections[command]; exists {
		return
	}
	selection := defaultSelection{
		Status: defaultSelectionNotEvaluated,
	}
	for _, record := range analysis.effectiveModel.DefaultsForParent(command) {
		if analysisGrammarProfileApplies(result.SelectedProfiles, command, record.meta.profile) {
			selection.Candidates = append(selection.Candidates, record.meta.id)
		}
	}
	if result.DefaultSelections == nil {
		result.DefaultSelections = make(map[CommandID]defaultSelection)
	}
	result.DefaultSelections[command] = selection
}

func evaluateCommandDefaultsAt(analysis *analysisState, result *grammarResult, materialize grammarDefaultDemandFunc, command CommandID) defaultSelection {
	if existing, ok := result.DefaultSelections[command]; ok {
		return existing
	}
	selection := defaultSelection{
		Status: defaultSelectionNone,
	}
	for _, record := range analysis.effectiveModel.DefaultsForParent(command) {
		if !analysisGrammarProfileApplies(result.SelectedProfiles, command, record.meta.profile) {
			continue
		}
		selection.Candidates = append(selection.Candidates, record.meta.id)
		outcome := conditionOutcomeSatisfied
		if _, conditional := record.Condition(); conditional {
			if materialize == nil {
				outcome = conditionOutcomeUnsupported
			} else {
				outcome = materialize(grammarDefaultDemand{
					Default: record,
					Grammar: grammarDemandSnapshot(*result),
				})
			}
			if result.DefaultConditions == nil {
				result.DefaultConditions = make(map[CommandDefaultID]conditionOutcome)
			}
			result.DefaultConditions[record.meta.id] = outcome
		}
		switch outcome {
		case conditionOutcomeSatisfied:
			selection.Applicable = append(selection.Applicable, record.meta.id)
		case conditionOutcomeUnsatisfied:
		case conditionOutcomeBlocked, conditionOutcomeUnsupported, conditionOutcomeUnknown:
			selection.Blocked = append(selection.Blocked, record.meta.id)
		}
	}
	switch {
	case len(selection.Applicable) > 1:
		selection.Status = defaultSelectionAmbiguous
	case len(selection.Blocked) != 0:
		selection.Status = defaultSelectionBlocked
	case len(selection.Applicable) == 1:
		selection.Status = defaultSelectionSelected
		selection.Selected = selection.Applicable[0]
	default:
		selection.Status = defaultSelectionNone
	}
	if result.DefaultSelections == nil {
		result.DefaultSelections = make(map[CommandID]defaultSelection)
	}
	result.DefaultSelections[command] = selection
	return selection
}

func selectDefaultCommand(operation *invocation, result *grammarResult, materialize grammarDefaultDemandFunc, condition grammarConditionDemandFunc, model grammarView, command CommandID, tokenIndex int) (CommandID, bool) {
	return selectDefaultCommandAt(operation, operation.analysis, result, materialize, condition, model, command, tokenIndex)
}

func selectDefaultCommandAt(operation *invocation, analysis *analysisState, result *grammarResult, materialize grammarDefaultDemandFunc, condition grammarConditionDemandFunc, model grammarView, command CommandID, tokenIndex int) (CommandID, bool) {
	selection := evaluateCommandDefaultsAt(analysis, result, materialize, command)
	switch selection.Status {
	case defaultSelectionNone:
		return command, true
	case defaultSelectionAmbiguous:
		grammarIssueValue := grammarIssue{
			Kind:       grammarIssueAmbiguousDefault,
			TokenIndex: tokenIndex,
			Command:    command,
			Defaults:   append([]CommandDefaultID(nil), selection.Applicable...),
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
		return command, false
	case defaultSelectionBlocked:
		grammarIssueValue2 := grammarIssue{
			Kind:       grammarIssueBlockedDefault,
			TokenIndex: tokenIndex,
			Command:    command,
			Defaults:   append([]CommandDefaultID(nil), selection.Blocked...),
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue2)
		return command, false
	case defaultSelectionSelected:
		record, ok := analysis.effectiveModel.Default(selection.Selected)
		if !ok {
			grammarIssueValue3 := grammarIssue{
				Kind:       grammarIssueBlockedDefault,
				TokenIndex: tokenIndex,
				Command:    command,
				Defaults:   []CommandDefaultID{selection.Selected},
			}
			appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue3)
			return command, false
		}
		next := record.target
		enabled, settled := structuralAvailabilityAt(
			operation,
			analysis,
			result,
			condition,
			commandConditionTarget(next),
			tokenIndex,
		)
		if !settled {
			return command, false
		}
		if !enabled || !defaultTargetLifecycleCurrentAt(analysis, model, command, next) {
			selection.Status = defaultSelectionBlocked
			selection.Blocked = appendUniqueDefaultID(selection.Blocked, selection.Selected)
			selection.Selected = CommandDefaultID(0)
			result.DefaultSelections[command] = selection
			grammarIssueValue4 := grammarIssue{
				Kind:       grammarIssueBlockedDefault,
				TokenIndex: tokenIndex,
				Command:    command,
				Defaults:   []CommandDefaultID{record.meta.id},
			}
			appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue4)
			return command, false
		}
		previous := command
		result.Selection.Selected = next
		result.Selection.Path = append(result.Selection.Path, next)
		result.Selection.Steps = append(result.Selection.Steps, selectionStep{
			Default:  record.meta.id,
			Commands: []CommandID{previous, next},
		})
		return next, true
	default:
		return command, false
	}
}

func appendUniqueDefaultID(values []CommandDefaultID, id CommandDefaultID) []CommandDefaultID {
	if slices.Contains(values, id) {
		return values
	}
	return append(values, id)
}

func defaultTargetLifecycleCurrentAt(analysis *analysisState, model grammarView, parent, target CommandID) bool {
	found := false
	for _, entry := range model.CommandEntries(parent) {
		if entry.kind != CommandEntryCanonical || entry.target != target {
			continue
		}
		found = true
		switch commandEntryLifecycleState(analysis, entry.id) {
		case LifecycleStateCurrent, LifecycleStateDeprecated:
			return true
		case LifecycleStateUnknown,
			LifecycleStateNotIntroduced,
			LifecycleStateRemoved:
		}
	}
	return !found
}

func finalizedGrammarResult(args []string, result grammarResult) grammarResult {
	finalizeTokenObservations(args, &result)
	return result
}

func appendMissingFlagOccurrences(result *grammarResult, limit int, model grammarView, path []CommandID, occurrences map[FlagID]int) {
	for _, command := range path {
		for _, flag := range model.Flags(command) {
			if !flag.Enabled() || occurrences[flag.id] >= flag.occurrences.Minimum() {
				continue
			}
			grammarIssueValue := grammarIssue{
				Kind:             grammarIssueMissingFlagOccurrence,
				TokenIndex:       -1,
				Command:          command,
				FlagDeclarations: []FlagID{flag.id},
			}
			if !appendGrammarIssue(result, limit, grammarIssueValue) {
				return
			}
		}
	}
}

func appendGrammarIssue(result *grammarResult, limit int, issue grammarIssue) bool {
	if limit > 0 && len(result.Issues) >= limit {
		result.Truncated = true
		return false
	}
	result.Issues = append(result.Issues, issue)
	return true
}

func grammarDiagnosticLimitReached(result *grammarResult, limit int) bool {
	return limit > 0 && len(result.Issues) >= limit
}

func claimToken(result *grammarResult, index int, ownership tokenOwnership) {
	if index < 0 || index >= len(result.Ownership) {
		panic("invocation: token ownership index out of range")
	}
	if result.Ownership[index].Role != tokenRoleUnowned {
		panic("invocation: token ownership assigned twice")
	}
	ownership.Origin = result.Ownership[index].Origin
	result.Ownership[index] = ownership
}

func isEndOfOptions(dialect dialect, token string) bool {
	return slices.Contains(dialect.EndOfOptions(), token)
}

func finalizeTokenObservations(args []string, result *grammarResult) {
	observations := make([]tokenObservation, 0, len(result.Ownership))
	for index, ownership := range result.Ownership {
		if override := result.tokenOverrides[index]; len(override) != 0 {
			observations = append(observations, override...)
			continue
		}
		text := args[index]
		observation := tokenObservation{
			Text:          text,
			Origin:        ownership.Origin,
			PhysicalIndex: index,
			Start:         0,
			End:           len(text),
			Ownership:     ownership,
		}
		observations = append(observations, observation)
	}
	result.Tokens = observations
	result.tokenOverrides = nil
}

type lifecycleOwner struct {
	index int
	state LifecycleState
	life  ReleaseLifecycle
}

// lifecycleOwnedCommandGrammar applies the release-time temporal owner rule only
// to one command lexical namespace after profile selection. Inactive winning
// owners remain present so recognition can emit precise lifecycle diagnostics.
func lifecycleOwnedCommandGrammar(analysis *analysisState, model grammarView, selected map[CommandID]GrammarProfileID, command CommandID) ([]commandEntryGrammar, []actionCommandGrammar) {
	if analysis == nil || model == nil {
		return nil, nil
	}
	dialect, ok := selectedGrammarDialect(model, selected, command)
	if !ok {
		return nil, nil
	}
	mode := dialect.CommandNormalization()
	type member struct {
		entry  *commandEntryGrammar
		action *actionCommandGrammar
		life   ReleaseLifecycle
	}
	groups := make(map[string][]member)
	entries := model.CommandEntries(command)
	canonicalLifecycles := canonicalCommandEntryLifecycles(analysis, entries)
	profileEntries := make([]commandEntryGrammar, 0, len(entries))
	for i := range entries {
		if entries[i].profile.present() && selected[command] != entries[i].profile {
			continue
		}
		profileEntries = append(profileEntries, entries[i])
	}
	actions := model.ActionCommands(command)
	for i := range profileEntries {
		name, valid := normalizeName(mode, profileEntries[i].name)
		if !valid {
			continue
		}
		groups[name] = append(groups[name], member{
			entry: &profileEntries[i],
			life: effectiveCommandEntryLifecycleWithCanonical(
				analysis,
				model,
				profileEntries[i],
				canonicalLifecycles,
			),
		})
	}
	for i := range actions {
		name, valid := normalizeName(mode, actions[i].name)
		if !valid {
			continue
		}
		groups[name] = append(groups[name], member{
			action: &actions[i],
			life:   actionLifecycle(analysis, actions[i].id),
		})
	}
	entryKeep := make(map[CommandEntryID]struct{})
	actionKeep := make(map[ActionEntryID]struct{})
	for _, group := range groups {
		owners := make([]lifecycleOwner, 0, len(group))
		for i, item := range group {
			state, _ := evaluateLifecycle(analysis.effectiveModel.Application().release, item.life, "")
			lifecycleOwnerValue := lifecycleOwner{
				index: i,
				state: state,
				life:  item.life,
			}
			owners = append(owners, lifecycleOwnerValue)
		}
		for _, index := range chooseLifecycleOwners(analysis.effectiveModel.Application().release, owners) {
			item := group[index]
			if item.entry != nil {
				entryKeep[item.entry.id] = struct{}{}
			} else {
				actionKeep[item.action.id] = struct{}{}
			}
		}
	}
	keptEntries := make([]commandEntryGrammar, 0, len(profileEntries))
	for _, entry := range profileEntries {
		if _, ok := entryKeep[entry.id]; ok {
			keptEntries = append(keptEntries, entry)
		}
	}
	keptActions := make([]actionCommandGrammar, 0, len(actions))
	for _, action := range actions {
		if _, ok := actionKeep[action.id]; ok {
			keptActions = append(keptActions, action)
		}
	}
	return keptEntries, keptActions
}

// lifecycleOwnedFlagGrammar applies the same lexical-owner rule to one
// command's flag spelling/action namespace.
func lifecycleOwnedFlagGrammar(analysis *analysisState, model grammarView, selected map[CommandID]GrammarProfileID, command CommandID) ([]flagGrammar, []semanticActionFlagGrammar) {
	if analysis == nil || model == nil {
		return nil, nil
	}
	dialect, ok := selectedGrammarDialect(model, selected, command)
	if !ok {
		return nil, nil
	}
	mode := dialect.FlagNormalization()
	type member struct {
		spelling *flagSpellingGrammar
		action   *semanticActionFlagGrammar
		life     ReleaseLifecycle
	}
	groups := make(map[string][]member)
	flags := model.Flags(command)
	profileFlags := make([]flagGrammar, 0, len(flags))
	for _, flag := range flags {
		if flag.profile.present() && selected[command] != flag.profile {
			continue
		}
		profileFlags = append(profileFlags, flag)
	}
	for fi := range profileFlags {
		spellings := profileFlags[fi].Spellings()
		for si := range spellings {
			name, valid := normalizeName(mode, spellings[si].text)
			if !valid {
				continue
			}
			groups[name] = append(groups[name], member{
				spelling: &spellings[si],
				life:     effectiveFlagSpellingLifecycle(analysis, profileFlags[fi].id, spellings[si].id),
			})
		}
	}
	actions := model.ActionFlags(command)
	for i := range actions {
		name, valid := normalizeName(mode, actions[i].text)
		if !valid {
			continue
		}
		groups[name] = append(groups[name], member{
			action: &actions[i],
			life:   actionLifecycle(analysis, actions[i].id),
		})
	}
	spellingKeep := make(map[FlagSpellingID]struct{})
	actionKeep := make(map[ActionEntryID]struct{})
	for _, group := range groups {
		owners := make([]lifecycleOwner, 0, len(group))
		for i, item := range group {
			state, _ := evaluateLifecycle(analysis.effectiveModel.Application().release, item.life, "")
			lifecycleOwnerValue := lifecycleOwner{
				index: i,
				state: state,
				life:  item.life,
			}
			owners = append(owners, lifecycleOwnerValue)
		}
		for _, index := range chooseLifecycleOwners(analysis.effectiveModel.Application().release, owners) {
			item := group[index]
			if item.spelling != nil {
				spellingKeep[item.spelling.id] = struct{}{}
			} else {
				actionKeep[item.action.id] = struct{}{}
			}
		}
	}
	keptFlags := make([]flagGrammar, 0, len(profileFlags))
	for _, flag := range profileFlags {
		var spellings []flagSpellingGrammar
		for _, spelling := range flag.Spellings() {
			if _, ok := spellingKeep[spelling.id]; ok {
				spellings = append(spellings, spelling)
			}
		}
		if len(spellings) != 0 {
			flag.spellings = append([]flagSpellingGrammar(nil), spellings...)
			keptFlags = append(keptFlags, flag)
		}
	}
	keptActions := make([]semanticActionFlagGrammar, 0, len(actions))
	for _, action := range actions {
		if _, ok := actionKeep[action.id]; ok {
			keptActions = append(keptActions, action)
		}
	}
	return keptFlags, keptActions
}

func selectedGrammarDialect(model grammarView, selected map[CommandID]GrammarProfileID, command CommandID) (dialect, bool) {
	if profile := selected[command]; profile.present() {
		if dialect, ok := model.ProfileDialect(profile); ok {
			return dialect, true
		}
	}
	return model.Dialect(command)
}

func chooseLifecycleOwners(scheme ReleaseScheme, owners []lifecycleOwner) []int {
	if len(owners) <= 1 {
		if len(owners) == 1 {
			return []int{owners[0].index}
		}
		return nil
	}
	var active, removed, future []lifecycleOwner
	for _, owner := range owners {
		switch owner.state {
		case LifecycleStateCurrent, LifecycleStateDeprecated:
			active = append(active, owner)
		case LifecycleStateRemoved:
			removed = append(removed, owner)
		case LifecycleStateNotIntroduced:
			future = append(future, owner)
		default:
			active = append(active, owner) // invalid should have been rejected; preserve ambiguity fail-closed.
		}
	}
	if len(active) != 0 {
		return ownerIndices(active)
	}
	if len(removed) != 0 {
		slices.SortStableFunc(removed, func(left, right lifecycleOwner) int {
			return -compareReleaseBoundary(scheme, left.life.RemovedIn, right.life.RemovedIn)
		})
		best := removed[0]
		result := []int{best.index}
		for _, candidate := range removed[1:] {
			if compareReleaseBoundary(scheme, candidate.life.RemovedIn, best.life.RemovedIn) == 0 {
				result = append(result, candidate.index)
			}
		}
		return result
	}
	slices.SortStableFunc(future, func(left, right lifecycleOwner) int {
		return compareReleaseBoundary(scheme, left.life.IntroducedIn, right.life.IntroducedIn)
	})
	if len(future) == 0 {
		return nil
	}
	best := future[0]
	result := []int{best.index}
	for _, candidate := range future[1:] {
		if compareReleaseBoundary(scheme, candidate.life.IntroducedIn, best.life.IntroducedIn) == 0 {
			result = append(result, candidate.index)
		}
	}
	return result
}

func ownerIndices(values []lifecycleOwner) []int {
	result := make([]int, len(values))
	for i, v := range values {
		result[i] = v.index
	}
	return result
}

func compareReleaseBoundary(scheme ReleaseScheme, left, right string) int {
	if left == right {
		return 0
	}
	if left == "" {
		return -1
	}
	if right == "" {
		return 1
	}
	cmp, err := scheme.compare(left, right)
	if err != nil {
		return 0
	}
	return cmp
}

func effectiveCommandEntryLifecycle(analysis *analysisState, model grammarView, entry commandEntryGrammar) ReleaseLifecycle {
	return effectiveCommandEntryLifecycleWithCanonical(analysis, model, entry, nil)
}

func effectiveCommandEntryLifecycleWithCanonical(analysis *analysisState, model grammarView, entry commandEntryGrammar, canonicalLifecycles map[CommandID]ReleaseLifecycle) ReleaseLifecycle {
	record, ok := analysis.effectiveModel.CommandEntry(entry.id)
	if !ok {
		lifecycle := ReleaseLifecycle{
			IntroducedIn: "\x00",
		}
		return lifecycle
	}
	life := record.lifecycle
	switch entry.kind {
	case CommandEntryCanonical:
		return life
	case CommandEntryAlias:
		if canonical, ok := canonicalLifecycles[entry.target]; ok {
			return intersectEffectiveReleaseLifecycles(
				analysis.effectiveModel.Application().release,
				life,
				canonical,
			)
		}
		if canonical, ok := canonicalEntryLifecycle(analysis, model, entry.target); ok {
			return intersectEffectiveReleaseLifecycles(
				analysis.effectiveModel.Application().release,
				life,
				canonical,
			)
		}
		lifecycle := ReleaseLifecycle{
			IntroducedIn: "\x00",
		}
		return lifecycle
	case CommandEntryShortcut:
		path, err := canonicalCommandPath(analysis.effectiveModel, model, entry.parent, entry.target)
		if err != nil || len(path) == 0 {
			lifecycle := ReleaseLifecycle{
				IntroducedIn: "\x00",
			}
			return lifecycle
		}
		canonical, err := intersectCanonicalCommandPathLifecycle(
			analysis.effectiveModel,
			analysis.effectiveModel.Application().release,
			path,
		)
		if err != nil {
			lifecycle := ReleaseLifecycle{
				IntroducedIn: "\x00",
			}
			return lifecycle
		}
		return intersectEffectiveReleaseLifecycles(analysis.effectiveModel.Application().release, life, canonical)
	default:
		lifecycle := ReleaseLifecycle{
			IntroducedIn: "\x00",
		}
		return lifecycle
	}
}

func canonicalCommandEntryLifecycles(analysis *analysisState, entries []commandEntryGrammar) map[CommandID]ReleaseLifecycle {
	result := make(map[CommandID]ReleaseLifecycle)
	for _, entry := range entries {
		if entry.kind != CommandEntryCanonical {
			continue
		}
		if _, exists := result[entry.target]; exists {
			continue
		}
		if record, ok := analysis.effectiveModel.CommandEntry(entry.id); ok {
			result[entry.target] = record.lifecycle
		}
	}
	return result
}

func canonicalEntryLifecycle(analysis *analysisState, model grammarView, target CommandID) (ReleaseLifecycle, bool) {
	if model == nil {
		return ReleaseLifecycle{}, false
	}
	command, ok := analysis.effectiveModel.Command(target)
	if !ok {
		return ReleaseLifecycle{}, false
	}
	parent, ok := command.Parent()
	if !ok {
		return ReleaseLifecycle{}, false
	}
	for _, entry := range model.CommandEntries(parent) {
		if entry.target == target && entry.kind == CommandEntryCanonical {
			if record, exists := analysis.effectiveModel.CommandEntry(entry.id); exists {
				return record.lifecycle, true
			}
		}
	}
	return ReleaseLifecycle{}, false
}

func actionLifecycle(analysis *analysisState, id ActionEntryID) ReleaseLifecycle {
	if r, ok := analysis.effectiveModel.ActionEntry(id); ok {
		return r.lifecycle
	}
	return ReleaseLifecycle{}
}

func effectiveFlagSpellingLifecycle(analysis *analysisState, flag FlagID, spelling FlagSpellingID) ReleaseLifecycle {
	fr, fok := analysis.effectiveModel.Flag(flag)
	sr, sok := analysis.effectiveModel.FlagSpelling(spelling)
	if !fok || !sok {
		return ReleaseLifecycle{}
	}
	return intersectEffectiveReleaseLifecycles(
		analysis.effectiveModel.Application().release,
		fr.lifecycle,
		sr.lifecycle,
	)
}

func commandEntryLifecycleState(analysis *analysisState, id CommandEntryID) LifecycleState {
	grammar, ok := analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		return LifecycleStateUnknown
	}
	record, ok := analysis.effectiveModel.CommandEntry(id)
	if !ok {
		return LifecycleStateUnknown
	}
	for _, entry := range grammar.CommandEntries(record.parent) {
		if entry.id == id {
			state, _ := evaluateLifecycle(
				analysis.effectiveModel.Application().release,
				effectiveCommandEntryLifecycle(analysis, grammar, entry),
				"",
			)
			return state
		}
	}
	return LifecycleStateUnknown
}

func actionEntryLifecycleState(analysis *analysisState, id ActionEntryID) LifecycleState {
	state, _ := evaluateLifecycle(
		analysis.effectiveModel.Application().release,
		actionLifecycle(analysis, id),
		"",
	)
	return state
}

func flagSpellingLifecycleState(analysis *analysisState, flag FlagID, spelling FlagSpellingID) LifecycleState {
	state, _ := evaluateLifecycle(
		analysis.effectiveModel.Application().release,
		effectiveFlagSpellingLifecycle(analysis, flag, spelling),
		"",
	)
	return state
}

func argumentFormLifecycleState(analysis *analysisState, id ArgumentFormID) LifecycleState {
	if r, ok := analysis.effectiveModel.ArgumentForm(id); ok {
		state, _ := evaluateLifecycle(analysis.effectiveModel.Application().release, r.lifecycle, "")
		return state
	}
	return LifecycleStateUnknown
}

func argumentTermLifecycleState(analysis *analysisState, id ArgumentTermID) LifecycleState {
	if r, ok := analysis.effectiveModel.ArgumentTerm(id); ok {
		state, _ := evaluateLifecycle(analysis.effectiveModel.Application().release, r.lifecycle, "")
		return state
	}
	return LifecycleStateUnknown
}

func commandMatchLifecycleIssue(analysis *analysisState, match childMatch) grammarIssueKind {
	state := LifecycleStateCurrent
	switch {
	case match.lifecycleKnown:
		state = match.lifecycle
	case match.action.present():
		state = actionEntryLifecycleState(analysis, match.action)
	case match.entry.id.present():
		state = commandEntryLifecycleState(analysis, match.entry.id)
	}
	switch state {
	case LifecycleStateNotIntroduced:
		return grammarIssueNotIntroducedCommand
	case LifecycleStateRemoved:
		return grammarIssueRemovedCommand
	case LifecycleStateUnknown:
		return grammarIssueBlockedExplicitEntry
	default:
		return grammarIssueNone
	}
}

func flagMatchLifecycleIssue(analysis *analysisState, match flagMatch) grammarIssueKind {
	if match.action.id.present() {
		switch actionEntryLifecycleState(analysis, match.action.id) {
		case LifecycleStateNotIntroduced:
			return grammarIssueNotIntroducedFlag
		case LifecycleStateRemoved:
			return grammarIssueRemovedFlag
		default:
			return grammarIssueNone
		}
	}
	return flagSpellingLifecycleIssue(analysis, match.flag.id, match.spelling.id)
}

func flagSpellingLifecycleIssue(analysis *analysisState, flag FlagID, spelling FlagSpellingID) grammarIssueKind {
	switch flagSpellingLifecycleState(analysis, flag, spelling) {
	case LifecycleStateNotIntroduced:
		return grammarIssueNotIntroducedFlag
	case LifecycleStateRemoved:
		return grammarIssueRemovedFlag
	default:
		return grammarIssueNone
	}
}

func unavailableArgumentLifecycleIssue(analysis *analysisState, forms []argumentFormGrammar, ids []ArgumentFormID) grammarIssueKind {
	selected := make(map[ArgumentFormID]struct{}, len(ids))
	for _, id := range ids {
		selected[id] = struct{}{}
	}
	kind := grammarIssueNone
	for _, form := range forms {
		if _, ok := selected[form.id]; !ok {
			continue
		}
		state := argumentFormLifecycleState(analysis, form.id)
		if state == LifecycleStateCurrent || state == LifecycleStateDeprecated {
			state = LifecycleStateUnknown
			for _, term := range form.Terms() {
				termState := argumentTermLifecycleState(analysis, term.id)
				if termState == LifecycleStateNotIntroduced ||
					termState == LifecycleStateRemoved {
					state = termState
					break
				}
			}
		}
		var candidate grammarIssueKind
		switch state {
		case LifecycleStateNotIntroduced:
			candidate = grammarIssueNotIntroducedArgument
		case LifecycleStateRemoved:
			candidate = grammarIssueRemovedArgument
		default:
			return grammarIssueNone
		}
		if kind == grammarIssueNone {
			kind = candidate
		} else if kind != candidate {
			return grammarIssueNone
		}
	}
	return kind
}
