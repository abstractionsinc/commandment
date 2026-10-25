package cli

import (
	"context"
	"maps"
)

type grammarScanState struct {
	current            CommandID
	flagsEnabled       bool
	route              grammarRoute
	occurrences        map[FlagID]int
	positional         []int
	pendingFlagValue   *flagOccurrence
	actionIncompatible int
	ownershipCertain   bool
}

// analyzeGrammar performs command/flag recognition, fixed positional
// allocation, and the conversion-aware form classification required to commit
// one positional form. Conversion cannot revise the structural allocations.
func analyzeGrammar(ctx context.Context, operation *invocation) grammarResult {
	if operation == nil {
		panic("invocation: nil operation")
	}
	result, _ := analysisAnalyzeGrammarAt(
		ctx,
		operation,
		operation.analysis,
		operation.Args(),
		true,
		-1,
		nil,
		nil,
		nil,
		nil,
	)
	return result
}

func analyzeCommittedGrammar(ctx context.Context, operation *invocation) (grammarResult, error) {
	if operation == nil || operation.analysis == nil {
		panic("invocation: nil committed grammar operation")
	}
	if _, err := settleResponseRange(ctx, operation, operation.analysis, 0, -1, -1); err != nil {
		return grammarResult{}, err
	}
	if err := settleCommittedShortcuts(ctx, operation); err != nil {
		return grammarResult{}, err
	}
	result, _ := analysisAnalyzeGrammarAt(
		ctx,
		operation,
		operation.analysis,
		operation.Args(),
		true,
		-1,
		operation.namespaceFor(ctx),
		operation.conditionFor(ctx),
		operation.defaultCommandFor(ctx),
		operation.valueLanguage(ctx),
	)
	operation.EnableShortcutDefaults()
	return result, nil
}

func analyzeGrammarArguments(ctx context.Context, operation *invocation, args []string, namespace grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc, valueLanguages grammarValueLanguageDemandFunc) grammarResult {
	if operation == nil {
		panic("invocation: nil operation")
	}
	seed := operationDependencySeed(operation.analysis)
	defer storeOperationDependencySeed(operation.analysis, seed)
	result, _ := analysisAnalyzeGrammarAt(
		ctx,
		operation,
		operation.analysis,
		append([]string(nil), args...),
		true,
		-1,
		namespace,
		condition,
		defaults,
		valueLanguages,
	)
	return result
}

func analyzeGrammarPrefix(ctx context.Context, operation *invocation, argument int, namespace grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) grammarPrefixResult {
	return analyzeGrammarPrefixSkipping(
		ctx,
		operation,
		argument,
		-1,
		namespace,
		condition,
		defaults,
	)
}

// analyzeGrammarPrefixSkipping evaluates one lexical
// prefix while leaving one argument in that prefix unowned. Completion uses
// this only when later fixed input must be interpreted without converting the
// cursor-owned incomplete token into complete grammar input.
func analyzeGrammarPrefixSkipping(ctx context.Context, operation *invocation, argument, skipArgument int, namespace grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) grammarPrefixResult {
	if operation == nil {
		panic("invocation: nil operation")
	}
	args := operation.Args()
	if argument < 0 || argument > len(args) {
		panic("invocation: grammar prefix argument out of range")
	}
	if skipArgument >= argument {
		skipArgument = -1
	}
	return analyzeGrammarPrefixArgumentsSkipping(
		ctx,
		operation,
		args[:argument],
		skipArgument,
		namespace,
		condition,
		defaults,
	)
}

// analyzeGrammarArgumentsPrefixSkipping is the
// provisional-argument counterpart of analyzeGrammarPrefixSkipping.
// The supplied slice is detached before analysis.
func analyzeGrammarArgumentsPrefixSkipping(ctx context.Context, operation *invocation, args []string, skipArgument int, namespace grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) grammarPrefixResult {
	if operation == nil {
		panic("invocation: nil operation")
	}
	seed := operationDependencySeed(operation.analysis)
	defer storeOperationDependencySeed(operation.analysis, seed)
	if skipArgument >= len(args) {
		skipArgument = -1
	}
	return analyzeGrammarPrefixArgumentsSkipping(
		ctx,
		operation,
		append([]string(nil), args...),
		skipArgument,
		namespace,
		condition,
		defaults,
	)
}

// analyzeGrammarCompletionTail consumes the complete
// operation argument stream while leaving one incomplete cursor argument
// unowned. It exists so Completion can settle fixed input that follows the
// cursor through the ordinary recognizer without relocating that fixed input
// ahead of residual user input. Final positional allocation remains deferred.
func analyzeGrammarCompletionTail(ctx context.Context, operation *invocation, cursorArgument int, namespace grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) grammarPrefixResult {
	if operation == nil {
		panic("invocation: nil operation")
	}
	seed := operationDependencySeed(operation.analysis)
	defer storeOperationDependencySeed(operation.analysis, seed)
	args := operation.Args()
	if cursorArgument < 0 || cursorArgument >= len(args) {
		panic("invocation: completion cursor argument out of range")
	}
	return analyzeGrammarPrefixArgumentsSkipping(ctx, operation, args, cursorArgument, namespace, condition, defaults)
}

func analyzeGrammarPrefixArgumentsSkipping(ctx context.Context, operation *invocation, args []string, skipArgument int, namespace grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) grammarPrefixResult {
	result, state := analysisAnalyzeGrammarAt(
		ctx,
		operation,
		operation.analysis,
		args,
		false,
		skipArgument,
		namespace,
		condition,
		defaults,
		nil,
	)
	var occurrences map[FlagID]int
	if len(state.occurrences) != 0 {
		occurrences = make(map[FlagID]int, len(state.occurrences))
		maps.Copy(occurrences, state.occurrences)
	}
	value := grammarPrefixResult{
		Grammar:            result,
		Current:            state.current,
		FlagsEnabled:       state.flagsEnabled,
		FlagOccurrences:    occurrences,
		PendingFlagValue:   cloneFlagOccurrence(state.pendingFlagValue),
		PositionalIndices:  append([]int(nil), state.positional...),
		route:              state.route,
		actionIncompatible: state.actionIncompatible,
	}

	return value
}

func terminalMissingFlagValue(issues []grammarIssue) bool {
	for _, issue := range issues {
		if issue.Kind == grammarIssueMissingFlagValue {
			return true
		}
	}
	return false
}

type grammarScanner struct {
	operation *invocation
	analysis  *analysisState
	model     grammarView
	result    grammarResult

	current            CommandID
	flagsEnabled       bool
	route              grammarRoute
	occurrences        map[FlagID]int
	positional         []int
	actionIncompatible int
	ownershipCertain   bool
	pendingFlagValue   *flagOccurrence
	demandedNamespaces map[CommandID]struct{}
	index              int
	rescanIndex        int
	rescanConditionLen int

	flagFrontier      selectedPathFlagFrontier
	flagFrontierValid bool
}

func prepareGrammarScanner(operation *invocation, analysis *analysisState, args []string, condition grammarConditionDemandFunc) *grammarScanner {
	if operation == nil || analysis == nil {
		panic("invocation: nil operation")
	}
	model, ok := analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		panic("invocation: application model has no grammar")
	}
	scanner := &grammarScanner{
		operation: operation,
		analysis:  analysis,
		model:     model,
		result: grammarResult{
			Selection: commandSelection{
				Root:     model.Root(),
				Selected: model.Root(),
				Path:     []CommandID{model.Root()},
			},
		},
		current:            model.Root(),
		flagsEnabled:       true,
		route:              grammarRouteSelecting,
		actionIncompatible: -1,
		ownershipCertain:   true,
		rescanIndex:        -1,
	}
	scanner.ensureOwnership(args)
	if enabled, settled := structuralAvailabilityAt(
		operation,
		analysis,
		&scanner.result,
		condition,
		commandConditionTarget(scanner.current),
		-1,
	); !settled {
		scanner.ownershipCertain = false
	} else if !enabled {
		grammarIssueValue := grammarIssue{
			Kind:            grammarIssueUnavailableCommand,
			TokenIndex:      -1,
			Command:         scanner.current,
			ConditionTarget: commandConditionTarget(scanner.current),
		}
		appendGrammarIssue(&scanner.result, operation.input.limits.Diagnostics, grammarIssueValue)
		scanner.ownershipCertain = false
	}
	return scanner
}

func (scanner *grammarScanner) ensureOwnership(args []string) {
	if len(scanner.result.Ownership) >= len(args) {
		return
	}
	start := len(scanner.result.Ownership)
	scanner.result.Ownership = append(scanner.result.Ownership, make([]tokenOwnership, len(args)-start)...)
	for index := start; index < len(args); index++ {
		origin, _ := scanner.analysis.argumentOrigin(index)
		scanner.result.Ownership[index].Origin = origin.ID()
	}
}

func (scanner *grammarScanner) refreshOrigins() {
	for index := range scanner.result.Ownership {
		origin, ok := scanner.analysis.argumentOrigin(index)
		if ok {
			scanner.result.Ownership[index].Origin = origin.ID()
		}
	}
}

func cloneGrammarScannerResult(result grammarResult) grammarResult {
	cloned := grammarDemandSnapshot(result)
	if len(result.tokenOverrides) != 0 {
		cloned.tokenOverrides = make(map[int][]tokenObservation, len(result.tokenOverrides))
		for index, observations := range result.tokenOverrides {
			cloned.tokenOverrides[index] = append([]tokenObservation(nil), observations...)
		}
	}
	return cloned
}

func (scanner *grammarScanner) clone() grammarScanner {
	cloned := *scanner
	cloned.result = cloneGrammarScannerResult(scanner.result)
	if len(scanner.occurrences) != 0 {
		cloned.occurrences = make(map[FlagID]int, len(scanner.occurrences))
		maps.Copy(cloned.occurrences, scanner.occurrences)
	}
	cloned.positional = append([]int(nil), scanner.positional...)
	cloned.pendingFlagValue = cloneFlagOccurrence(scanner.pendingFlagValue)
	cloned.flagFrontier = scanner.flagFrontier.clone()
	if len(scanner.demandedNamespaces) != 0 {
		cloned.demandedNamespaces = make(map[CommandID]struct{}, len(scanner.demandedNamespaces))
		for command := range scanner.demandedNamespaces {
			cloned.demandedNamespaces[command] = struct{}{}
		}
	}
	return cloned
}

type commandNamespaceSettlement struct {
	model             grammarView
	view              grammarView
	dialect           dialect
	match             childMatch
	namespaceComplete bool
	changed           bool
	settled           bool
}

func settleCurrentCommandNamespace(operation *invocation, analysis *analysisState, result *grammarResult, condition grammarConditionDemandFunc, model, view grammarView, current CommandID, dialect dialect, token string, tokenIndex int, materialize grammarNamespaceDemandFunc, demandedNamespaces *map[CommandID]struct{}, matchChildPolicy func(grammarView, CommandID, dialect, string) childMatch) commandNamespaceSettlement {
	settlement := commandNamespaceSettlement{
		model:             model,
		view:              view,
		dialect:           dialect,
		namespaceComplete: true,
		settled:           true,
	}
	settlement.view, settlement.settled = applyCommandAvailabilityAt(
		operation, analysis, result, condition, settlement.view, current, settlement.dialect, token, tokenIndex,
	)
	if !settlement.settled {
		return settlement
	}
	settlement.match = matchChildPolicy(settlement.view, current, settlement.dialect, token)
	if materialize == nil {
		return settlement
	}
	providers := effectiveProviderRecords(
		analysis.effectiveModel.ProvidersForParent(current),
		current,
		result.SelectedProfiles,
	)
	if len(providers) == 0 {
		return settlement
	}
	if _, alreadyDemanded := (*demandedNamespaces)[current]; alreadyDemanded {
		return settlement
	}
	if *demandedNamespaces == nil {
		*demandedNamespaces = make(map[CommandID]struct{})
	}
	(*demandedNamespaces)[current] = struct{}{}
	state := materialize(grammarNamespaceDemand{
		Command: current,
		Grammar: grammarDemandSnapshot(*result),
	})
	settlement.namespaceComplete = state.Complete
	if !state.Complete {
		result.IncompleteNamespaces = appendUniqueCommandID(result.IncompleteNamespaces, current)
	}
	if !state.Changed {
		return settlement
	}

	refreshed, ok := analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		grammarIssueValue := grammarIssue{
			Kind:       grammarIssueMissingDialect,
			TokenIndex: tokenIndex,
			Command:    current,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
		settlement.settled = false
		return settlement
	}
	settlement.model = refreshed
	settlement.changed = true
	settlement.view = grammarOperationSource{
		operation: operation,
		analysis:  analysis,
	}.effectiveScope(settlement.model, result.SelectedProfiles)
	settlement.dialect, ok = settlement.view.Dialect(current)
	if !ok {
		grammarIssueValue2 := grammarIssue{
			Kind:       grammarIssueMissingDialect,
			TokenIndex: tokenIndex,
			Command:    current,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue2)
		settlement.settled = false
		return settlement
	}
	settlement.view, settlement.settled = applyCommandAvailabilityAt(
		operation, analysis, result, condition, settlement.view, current, settlement.dialect, token, tokenIndex,
	)
	if !settlement.settled {
		return settlement
	}
	settlement.match = matchChildPolicy(settlement.view, current, settlement.dialect, token)
	return settlement
}

type flagTokenScanState struct {
	route              grammarRoute
	occurrences        map[FlagID]int
	positional         []int
	actionIncompatible int
	pendingFlagValue   *flagOccurrence
	index              int
	rescanIndex        int
	rescanConditionLen int
	flagFrontierValid  bool
}

type flagTokenScanOutcome struct {
	state              flagTokenScanState
	continueScan       bool
	stopScan           bool
	ownershipUncertain bool
}

func scanFlagToken(ctx context.Context, operation *invocation, analysis *analysisState, args []string, result *grammarResult, current CommandID, view grammarView, flagFrontier selectedPathFlagFrontier, dialect dialect, complete, streaming bool, conditionLenBeforeToken int, state flagTokenScanState) flagTokenScanOutcome {
	if dialect.ShortClustering() == ShortFlagClusteringEnabled {
		cluster := matchShortClusterFrontier(view, flagFrontier, dialect, args[state.index])
		if cluster.recognized {
			next, certain := acceptShortCluster(operation, analysis, args, result, current, view, result.Selection.Path, dialect, state.index, cluster, &state.occurrences, state.route == grammarRouteArguments)
			if !certain {
				result := flagTokenScanOutcome{
					state:              state,
					stopScan:           true,
					ownershipUncertain: true,
				}

				return result
			}
			state.index = next
			outcome := flagTokenScanOutcome{
				state:        state,
				continueScan: true,
			}
			return outcome
		}
	}

	match := matchFlagSetsFrontier(view, flagFrontier, dialect, args[state.index])
	if match.ambiguous {
		grammarIssueValue := grammarIssue{
			Kind:       grammarIssueAmbiguousFlag,
			TokenIndex: state.index,
			Command:    current,
			Flags:      match.candidates,
			Actions:    match.actions,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
		value := flagTokenScanOutcome{
			state:              state,
			stopScan:           true,
			ownershipUncertain: true,
		}

		return value
	}
	if !match.found {
		outcome := flagTokenScanOutcome{
			state: state,
		}
		return outcome
	}
	if kind := flagMatchLifecycleIssue(analysis, match); kind != grammarIssueNone {
		grammarIssueValue2 := grammarIssue{
			Kind:       kind,
			TokenIndex: state.index,
			Command:    current,
			Flags:      match.candidates,
			Actions:    match.actions,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue2)
		value := flagTokenScanOutcome{
			state:              state,
			stopScan:           true,
			ownershipUncertain: true,
		}

		return value
	}
	if !match.enabled {
		grammarIssueValue3 := grammarIssue{
			Kind:       grammarIssueUnavailableFlag,
			TokenIndex: state.index,
			Command:    current,
			Flags:      match.candidates,
			Actions:    match.actions,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue3)
		value := flagTokenScanOutcome{
			state:              state,
			stopScan:           true,
			ownershipUncertain: true,
		}

		return value
	}
	if match.action.id.present() {
		ownership2 := tokenOwnership{
			Role:        tokenRoleAction,
			Command:     current,
			ActionEntry: match.action.id,
			Match:       match.match,
		}
		claimToken(result, state.index, ownership2)
		result.Selection.Action = match.action.id
		result.Selection.ActionKind = match.action.action
		if state.route == grammarRouteArguments && len(state.positional) != 0 && state.actionIncompatible < 0 {
			state.actionIncompatible = state.positional[0]
		}
		state.positional = nil
		state.route = grammarRouteTerminalAction
		state.index++
		outcome := flagTokenScanOutcome{
			state:        state,
			continueScan: true,
		}
		return outcome
	}

	count := state.occurrences[match.flag.id]
	if maximum, bounded := match.flag.occurrences.Maximum(); bounded && count >= maximum {
		grammarIssueValue4 := grammarIssue{
			Kind:       grammarIssueTooManyFlagOccurrences,
			TokenIndex: state.index,
			Command:    current,
			Flags:      []FlagSpellingID{match.spelling.id},
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue4)
		value := flagTokenScanOutcome{
			state:              state,
			stopScan:           true,
			ownershipUncertain: true,
		}

		return value
	}
	occurrence, next, issue, issueIndex := planFlagOccurrence(args, state.index, current, view, result.Selection.Path, dialect, match, state.route == grammarRouteArguments)
	if issue == grammarIssueMissingFlagValue && !complete && next == len(args) {
		if streaming {
			state.rescanIndex = state.index
			state.rescanConditionLen = conditionLenBeforeToken
			outcome := flagTokenScanOutcome{
				state:    state,
				stopScan: true,
			}
			return outcome
		}
		occurrence.Occurrence = count
		claimFlagOccurrence(result, current, match, occurrence, count)
		state.pendingFlagValue = cloneFlagOccurrence(&occurrence)
		state.index = next
		outcome := flagTokenScanOutcome{
			state:        state,
			continueScan: true,
		}
		return outcome
	}
	if issue != grammarIssueNone {
		grammarIssueValue5 := grammarIssue{
			Kind:       issue,
			TokenIndex: issueIndex,
			Command:    current,
			Flags:      []FlagSpellingID{match.spelling.id},
		}
		if !appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue5) {
			result := flagTokenScanOutcome{
				state:              state,
				stopScan:           true,
				ownershipUncertain: true,
			}

			return result
		}
		if recoverableFlagOccurrenceIssue(issue, next, args, view, result.Selection.Path, dialect) {
			claimRejectedFlagOccurrence(result, state.index, current, match, occurrence, count)
			if grammarDiagnosticLimitReached(result, operation.input.limits.Diagnostics) {
				result.Truncated = true
				value := flagTokenScanOutcome{
					state:              state,
					stopScan:           true,
					ownershipUncertain: true,
				}

				return value
			}
			state.index = next
			outcome := flagTokenScanOutcome{
				state:        state,
				continueScan: true,
			}
			return outcome
		}
		value := flagTokenScanOutcome{
			state:              state,
			stopScan:           true,
			ownershipUncertain: true,
		}

		return value
	}

	occurrence.Occurrence = count
	claimFlagOccurrence(result, current, match, occurrence, count)
	if occurrence.TerminatorIndex >= 0 {
		ownership := tokenOwnership{
			Role:         tokenRoleFlagTerminator,
			Command:      current,
			Flag:         match.flag.id,
			FlagSpelling: match.spelling.id,
			Destination:  match.flag.destination,
			Form:         match.form,
			Occurrence:   count,
		}
		claimToken(result, occurrence.TerminatorIndex, ownership)
	}
	if state.occurrences == nil {
		state.occurrences = make(map[FlagID]int)
	}
	state.occurrences[match.flag.id] = count + 1
	result.Occurrences = append(result.Occurrences, occurrence)
	state.index = next
	if grammarProfileSelectorAt(analysis, current) == match.flag.destination && grammarProfilePendingAt(analysis, current, result.SelectedProfiles) {
		if !resolveGrammarProfileAt(ctx, operation, analysis, result, view, args, current, occurrence.TokenIndex) {
			result := flagTokenScanOutcome{
				state:              state,
				stopScan:           true,
				ownershipUncertain: true,
			}

			return result
		}
		state.flagFrontierValid = false
	}
	outcome := flagTokenScanOutcome{
		state:        state,
		continueScan: true,
	}
	return outcome
}

func claimFlagOccurrence(result *grammarResult, current CommandID, match flagMatch, occurrence flagOccurrence, count int) {
	ownership := tokenOwnership{
		Role:         tokenRoleFlag,
		Command:      current,
		Flag:         match.flag.id,
		FlagSpelling: match.spelling.id,
		Destination:  match.flag.destination,
		Match:        match.match,
		Form:         match.form,
		Occurrence:   count,
	}
	claimToken(result, occurrence.TokenIndex, ownership)
	for element, valueIndex := range occurrence.ValueIndices {
		ownership2 := tokenOwnership{
			Role:         tokenRoleFlagValue,
			Command:      current,
			Flag:         match.flag.id,
			FlagSpelling: match.spelling.id,
			Destination:  match.flag.destination,
			Form:         match.form,
			Occurrence:   count,
			Element:      element,
		}
		claimToken(result, valueIndex, ownership2)
	}
}

type commandRouteScanState struct {
	model             grammarView
	current           CommandID
	route             grammarRoute
	index             int
	flagsEnabled      bool
	flagFrontier      selectedPathFlagFrontier
	flagFrontierValid bool
}

type commandRouteScanOutcome struct {
	state        commandRouteScanState
	continueScan bool
	stopScan     bool
}

func scanHelpPathToken(operation *invocation, analysis *analysisState, result *grammarResult, condition grammarConditionDemandFunc, view grammarView, dialect dialect, token string, materialize grammarNamespaceDemandFunc, demandedNamespaces *map[CommandID]struct{}, state commandRouteScanState) commandRouteScanOutcome {
	settlement := settleCurrentCommandNamespace(
		operation, analysis, result, condition, state.model, view, state.current, dialect, token, state.index,
		materialize, demandedNamespaces, matchCommandChild,
	)
	state.model = settlement.model
	if settlement.changed {
		state.flagFrontierValid = false
	}
	if !settlement.settled {
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	match := settlement.match
	if match.ambiguous {
		grammarIssueValue2 := grammarIssue{
			Kind:       grammarIssueAmbiguousName,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue2)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	if !match.found {
		if !settlement.namespaceComplete {
			outcome := commandRouteScanOutcome{
				state:    state,
				stopScan: true,
			}
			return outcome
		}
		grammarIssueValue5 := grammarIssue{
			Kind:       grammarIssueUnexpectedInput,
			TokenIndex: state.index,
			Command:    state.current,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue5)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	if kind := commandMatchLifecycleIssue(analysis, match); kind != grammarIssueNone {
		grammarIssueValue := grammarIssue{
			Kind:       kind,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
			Actions:    match.actions,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	if !match.enabled {
		grammarIssueValue3 := grammarIssue{
			Kind:       grammarIssueBlockedExplicitEntry,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue3)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	markDefaultsNotEvaluatedAt(analysis, result, state.current)
	ownership := tokenOwnership{
		Role:         tokenRoleCommand,
		Command:      state.current,
		CommandEntry: match.entry.id,
		Match:        match.match,
	}
	claimToken(result, state.index, ownership)
	previous := state.current
	contribution, ok := commandEntrySelectionPath(analysis, settlement.view, state.current, match.entry)
	if !ok {
		grammarIssueValue4 := grammarIssue{
			Kind:       grammarIssueBlockedExplicitEntry,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue4)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	state.current = match.entry.target
	result.Selection.Selected = state.current
	result.Selection.Path = append(result.Selection.Path, contribution...)
	if state.flagFrontierValid {
		for _, command := range contribution {
			state.flagFrontier.extend(settlement.view, command)
		}
	}
	stepCommands := make([]CommandID, 1, len(contribution)+1)
	stepCommands[0] = previous
	stepCommands = append(stepCommands, contribution...)
	selectionStepValue := selectionStep{
		Entry:    match.entry.id,
		Input:    result.Ownership[state.index].Origin,
		Commands: stepCommands,
	}
	result.Selection.Steps = append(result.Selection.Steps, selectionStepValue)
	state.index++
	outcome := commandRouteScanOutcome{
		state:        state,
		continueScan: true,
	}
	return outcome
}

func scanSelectingToken(operation *invocation, analysis *analysisState, result *grammarResult, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc, view grammarView, dialect dialect, token string, materialize grammarNamespaceDemandFunc, demandedNamespaces *map[CommandID]struct{}, state commandRouteScanState) commandRouteScanOutcome {
	settlement := settleCurrentCommandNamespace(
		operation, analysis, result, condition, state.model, view, state.current, dialect, token, state.index,
		materialize, demandedNamespaces, matchChild,
	)
	state.model = settlement.model
	if settlement.changed {
		state.flagFrontierValid = false
	}
	if !settlement.settled {
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	match := settlement.match
	if match.ambiguous {
		grammarIssueValue := grammarIssue{
			Kind:       grammarIssueAmbiguousName,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
			Actions:    match.actions,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	if match.found {
		return acceptSelectingMatch(operation, analysis, result, settlement.view, match, state)
	}
	if !settlement.namespaceComplete {
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	next, selected := selectDefaultCommandAt(operation, analysis, result, defaults, condition, settlement.view, state.current, state.index)
	if !selected {
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	if next == state.current {
		outcome := commandRouteScanOutcome{
			state: state,
		}
		return outcome
	}
	state.current = next
	if state.flagFrontierValid {
		state.flagFrontier.extend(settlement.view, state.current)
	}
	outcome := commandRouteScanOutcome{
		state:        state,
		continueScan: true,
	}
	return outcome
}

func acceptSelectingMatch(operation *invocation, analysis *analysisState, result *grammarResult, view grammarView, match childMatch, state commandRouteScanState) commandRouteScanOutcome {
	if kind := commandMatchLifecycleIssue(analysis, match); kind != grammarIssueNone {
		grammarIssueValue := grammarIssue{
			Kind:       kind,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
			Actions:    match.actions,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	if !match.enabled {
		grammarIssueValue2 := grammarIssue{
			Kind:       grammarIssueBlockedExplicitEntry,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
			Actions:    match.actions,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue2)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	if match.action.present() {
		markDefaultsNotEvaluatedAt(analysis, result, state.current)
		ownership := tokenOwnership{
			Role:        tokenRoleAction,
			Command:     state.current,
			ActionEntry: match.action,
			Match:       match.match,
		}
		claimToken(result, state.index, ownership)
		result.Selection.Action = match.action
		result.Selection.ActionKind = match.actionKind
		state.index++
		state.route = grammarRouteTerminalAction
		if match.actionKind == intrinsicActionHelp {
			state.route = grammarRouteHelpPath
		}
		outcome := commandRouteScanOutcome{
			state:        state,
			continueScan: true,
		}
		return outcome
	}
	markDefaultsNotEvaluatedAt(analysis, result, state.current)
	ownership2 := tokenOwnership{
		Role:         tokenRoleCommand,
		Command:      state.current,
		CommandEntry: match.entry.id,
		Match:        match.match,
	}
	claimToken(result, state.index, ownership2)
	previous := state.current
	contribution, ok := commandEntrySelectionPath(analysis, view, state.current, match.entry)
	if !ok {
		grammarIssueValue3 := grammarIssue{
			Kind:       grammarIssueBlockedExplicitEntry,
			TokenIndex: state.index,
			Command:    state.current,
			Entries:    match.entries,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue3)
		outcome := commandRouteScanOutcome{
			state:    state,
			stopScan: true,
		}
		return outcome
	}
	state.current = match.entry.target
	result.Selection.Selected = state.current
	result.Selection.Path = append(result.Selection.Path, contribution...)
	if state.flagFrontierValid {
		for _, command := range contribution {
			state.flagFrontier.extend(view, command)
		}
	}
	stepCommands := make([]CommandID, 1, len(contribution)+1)
	stepCommands[0] = previous
	stepCommands = append(stepCommands, contribution...)
	selectionStepValue := selectionStep{
		Entry:    match.entry.id,
		Input:    result.Ownership[state.index].Origin,
		Commands: stepCommands,
	}
	result.Selection.Steps = append(result.Selection.Steps, selectionStepValue)
	state.index++
	outcome := commandRouteScanOutcome{
		state:        state,
		continueScan: true,
	}
	return outcome
}

func (scanner *grammarScanner) scan(ctx context.Context, args []string, complete bool, skipArgument int, materialize grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc, streaming bool) {
	scanner.ensureOwnership(args)
	operation := scanner.operation
	model := scanner.model
	result := scanner.result
	current := scanner.current
	flagsEnabled := scanner.flagsEnabled
	route := scanner.route
	occurrences := scanner.occurrences
	positional := scanner.positional
	ownershipCertain := scanner.ownershipCertain
	pendingFlagValue := scanner.pendingFlagValue
	demandedNamespaces := scanner.demandedNamespaces
	actionIncompatible := scanner.actionIncompatible
	rescanIndex := scanner.rescanIndex
	rescanConditionLen := scanner.rescanConditionLen
	index := scanner.index
	flagFrontier := scanner.flagFrontier
	flagFrontierValid := scanner.flagFrontierValid
scanLoop:
	for ownershipCertain && index < len(args) {
		if index == rescanIndex {
			if rescanConditionLen >= 0 && rescanConditionLen <= len(result.Conditions) {
				result.Conditions = result.Conditions[:rescanConditionLen]
			}
			rescanIndex = -1
		}
		if index == skipArgument {
			index++
			continue
		}
		view := grammarOperationSource{
			operation: operation,
			analysis:  scanner.analysis,
		}.effectiveScope(model, result.SelectedProfiles)
		dialect, ok := view.Dialect(current)
		if !ok {
			grammarIssueValue3 := grammarIssue{
				Kind:       grammarIssueMissingDialect,
				TokenIndex: index,
				Command:    current,
			}
			appendGrammarIssue(&result, operation.input.limits.Diagnostics, grammarIssueValue3)
			ownershipCertain = false
			break
		}
		token := args[index]
		conditionLenBeforeToken := len(result.Conditions)
		if !flagFrontierValid {
			flagFrontier = buildSelectedPathFlagFrontier(view, result.Selection.Path)
			flagFrontierValid = true
		}
		var settled bool
		view, settled = applyFlagAvailabilityWithFrontierAt(operation, scanner.analysis, &result, condition, view, flagFrontier, dialect, token, index)
		if !settled {
			ownershipCertain = false
			break
		}
		if grammarProfilePendingAt(scanner.analysis, current, result.SelectedProfiles) && !profilePrefixCanConsumeAt(scanner.analysis, view, result.Selection.Path, dialect, token) {
			if !resolveGrammarProfileAt(ctx, operation, scanner.analysis, &result, view, args, current, index) {
				ownershipCertain = false
				break
			}
			flagFrontierValid = false
			continue
		}
		if route == grammarRouteTerminalAction {
			if action, match, ok := terminalActionAt(scanner.analysis, view, flagFrontier, dialect, current, token); ok {
				if result.Selection.ActionSelected() {
					appendActionConflictIssue(
						&result,
						operation.input.limits.Diagnostics,
						current,
						firstActionInput(result.Ownership, result.Selection.Action),
						result.Selection.Action,
					)
					result.Selection.Action = 0
					result.Selection.ActionKind = intrinsicActionInvalid
					result.Selection.ActionTrigger = actionTriggerNone
				}
				ownership := tokenOwnership{
					Role:        tokenRoleAction,
					Command:     current,
					ActionEntry: action,
					Match:       match,
				}
				claimToken(&result, index, ownership)
				appendActionConflictIssue(&result, operation.input.limits.Diagnostics, current, index, action)
			} else if actionIncompatible < 0 {
				actionIncompatible = index
			}
			index++
			continue
		}
		if route == grammarRouteHelpPath {
			scanState := commandRouteScanState{
				model:             model,
				current:           current,
				route:             route,
				index:             index,
				flagFrontier:      flagFrontier,
				flagFrontierValid: flagFrontierValid,
			}
			outcome := scanHelpPathToken(operation, scanner.analysis, &result, condition, view, dialect, token, materialize, &demandedNamespaces, scanState)
			model = outcome.state.model
			current = outcome.state.current
			index = outcome.state.index
			flagFrontier = outcome.state.flagFrontier
			flagFrontierValid = outcome.state.flagFrontierValid
			if outcome.continueScan {
				continue
			}
			ownershipCertain = false
			break
		}
		if flagsEnabled && isEndOfOptions(dialect, token) {
			claimToken(&result, index, tokenOwnership{
				Role:    tokenRoleEndOfOptions,
				Command: current,
			})
			flagsEnabled = false
			index++
			continue
		}

		flagAllowed := flagsEnabled && (route != grammarRouteArguments || dialect.FlagOrder() == FlagOrderInterspersed)
		if flagAllowed {
			scanState2 := flagTokenScanState{
				route:              route,
				occurrences:        occurrences,
				positional:         positional,
				actionIncompatible: actionIncompatible,
				pendingFlagValue:   pendingFlagValue,
				index:              index,
				rescanIndex:        rescanIndex,
				rescanConditionLen: rescanConditionLen,
				flagFrontierValid:  flagFrontierValid,
			}
			outcome := scanFlagToken(ctx, operation, scanner.analysis, args, &result, current, view, flagFrontier, dialect, complete, streaming, conditionLenBeforeToken, scanState2)
			route = outcome.state.route
			occurrences = outcome.state.occurrences
			positional = outcome.state.positional
			actionIncompatible = outcome.state.actionIncompatible
			pendingFlagValue = outcome.state.pendingFlagValue
			index = outcome.state.index
			rescanIndex = outcome.state.rescanIndex
			rescanConditionLen = outcome.state.rescanConditionLen
			flagFrontierValid = outcome.state.flagFrontierValid
			if outcome.stopScan {
				if outcome.ownershipUncertain {
					ownershipCertain = false
				}
				break scanLoop
			}
			if outcome.continueScan {
				continue
			}
		}

		if flagsEnabled {
			known := matchSelectedFlagsFrontier(view, flagFrontier, dialect, token)
			if known.ambiguous {
				grammarIssueValue := grammarIssue{
					Kind:       grammarIssueAmbiguousFlag,
					TokenIndex: index,
					Command:    current,
					Flags:      known.candidates,
				}
				appendGrammarIssue(&result, operation.input.limits.Diagnostics, grammarIssueValue)
				ownershipCertain = false
				break
			}
			if known.found {
				grammarIssueValue2 := grammarIssue{
					Kind:       grammarIssueMisplacedFlag,
					TokenIndex: index,
					Command:    current,
					Flags:      []FlagSpellingID{known.spelling.id},
				}
				appendGrammarIssue(&result, operation.input.limits.Diagnostics, grammarIssueValue2)
				ownershipCertain = false
				break
			}
		}

		if flagAllowed && isUnknownHyphenFlagToken(token) && !positionalCanOwnToken(view.ArgumentForms(current), token, false) {
			grammarIssueValue4 := grammarIssue{
				Kind:       grammarIssueUnknownFlag,
				TokenIndex: index,
				Command:    current,
			}
			if !appendGrammarIssue(&result, operation.input.limits.Diagnostics, grammarIssueValue4) {
				ownershipCertain = false
				break
			}
			if isUnknownAssignedFlagToken(token) {
				// Assigned unknown input is lexically confined to this one token.
				// The rejected token remains represented by its structured issue;
				// later token ownership is still deterministic.
				if grammarDiagnosticLimitReached(&result, operation.input.limits.Diagnostics) && index+1 < len(args) {
					result.Truncated = true
					ownershipCertain = false
					break
				}
				index++
				continue
			}
			// A bare unknown flag may take following text, so the frontend
			// cannot recover without guessing ownership.
			ownershipCertain = false
			break
		}

		if route == grammarRouteSelecting {
			scanState3 := commandRouteScanState{
				model:             model,
				current:           current,
				route:             route,
				index:             index,
				flagsEnabled:      flagsEnabled,
				flagFrontier:      flagFrontier,
				flagFrontierValid: flagFrontierValid,
			}
			outcome := scanSelectingToken(operation, scanner.analysis, &result, condition, defaults, view, dialect, token, materialize, &demandedNamespaces, scanState3)
			model = outcome.state.model
			current = outcome.state.current
			route = outcome.state.route
			index = outcome.state.index
			flagFrontier = outcome.state.flagFrontier
			flagFrontierValid = outcome.state.flagFrontierValid
			if outcome.stopScan {
				ownershipCertain = false
				break
			}
			if outcome.continueScan {
				continue
			}
		}

		route = grammarRouteArguments
		positional = append(positional, index)
		index++
	}

	scanner.model = model
	scanner.result = result
	scanner.current = current
	scanner.flagsEnabled = flagsEnabled
	scanner.route = route
	scanner.occurrences = occurrences
	scanner.positional = positional
	scanner.ownershipCertain = ownershipCertain
	scanner.pendingFlagValue = pendingFlagValue
	scanner.demandedNamespaces = demandedNamespaces
	scanner.actionIncompatible = actionIncompatible
	scanner.index = index
	scanner.rescanIndex = rescanIndex
	scanner.rescanConditionLen = rescanConditionLen
	scanner.flagFrontier = flagFrontier
	scanner.flagFrontierValid = flagFrontierValid
}

func terminalActionAt(analysis *analysisState, view grammarView, frontier selectedPathFlagFrontier, dialect dialect, current CommandID, token string) (ActionEntryID, nameMatchKind, bool) {
	flag := matchActionFlagsFrontier(frontier, dialect, token)
	if flag.found && !flag.ambiguous && flag.enabled && flagMatchLifecycleIssue(analysis, flag) == grammarIssueNone {
		return flag.action.id, flag.match, true
	}
	command := matchChild(view, current, dialect, token)
	if command.found && !command.ambiguous && command.action.present() && command.enabled && commandMatchLifecycleIssue(analysis, command) == grammarIssueNone {
		return command.action, command.match, true
	}
	return 0, nameMatchKindNone, false
}

func firstActionInput(tokens []tokenOwnership, action ActionEntryID) int {
	for index, token := range tokens {
		if token.ActionEntry == action {
			return index
		}
	}
	return -1
}

func appendActionConflictIssue(result *grammarResult, limit int, command CommandID, input int, action ActionEntryID) {
	if input < 0 || !action.present() {
		return
	}
	issue := grammarIssue{
		Kind:       grammarIssueActionConflict,
		TokenIndex: input,
		Command:    command,
		Actions:    []ActionEntryID{action},
	}
	appendGrammarIssue(result, limit, issue)
}

func commandEntrySelectionPath(analysis *analysisState, grammar grammarView, parent CommandID, entry commandEntryGrammar) ([]CommandID, bool) {
	if entry.kind != CommandEntryShortcut {
		return []CommandID{entry.target}, true
	}
	path, err := canonicalCommandPath(analysis.effectiveModel, grammar, parent, entry.target)
	if err != nil || len(path) == 0 {
		return nil, false
	}
	commands := make([]CommandID, len(path))
	for i, step := range path {
		commands[i] = step.Command
	}
	return commands, true
}

func (scanner *grammarScanner) state() grammarScanState {
	state := grammarScanState{
		current:            scanner.current,
		flagsEnabled:       scanner.flagsEnabled,
		route:              scanner.route,
		occurrences:        scanner.occurrences,
		positional:         append([]int(nil), scanner.positional...),
		pendingFlagValue:   cloneFlagOccurrence(scanner.pendingFlagValue),
		actionIncompatible: scanner.actionIncompatible,
		ownershipCertain:   scanner.ownershipCertain,
	}

	return state
}

func grammarPrefixFromScanner(scanner *grammarScanner, args []string) grammarPrefixResult {
	result := finalizedGrammarResult(args, cloneGrammarScannerResult(scanner.result))
	var occurrences map[FlagID]int
	if len(scanner.occurrences) != 0 {
		occurrences = make(map[FlagID]int, len(scanner.occurrences))
		maps.Copy(occurrences, scanner.occurrences)
	}
	value := grammarPrefixResult{
		Grammar:            result,
		Current:            scanner.current,
		FlagsEnabled:       scanner.flagsEnabled,
		FlagOccurrences:    occurrences,
		PendingFlagValue:   cloneFlagOccurrence(scanner.pendingFlagValue),
		PositionalIndices:  append([]int(nil), scanner.positional...),
		route:              scanner.route,
		actionIncompatible: scanner.actionIncompatible,
	}

	return value
}

// grammarPrefixCursor incrementally advances the same canonical grammar scanner
// used by AnalyzeGrammar*. It is operation/call-local and exists only to avoid
// replaying already-settled prefixes at response-recognition boundaries.
type grammarPrefixCursor struct {
	scanner      *grammarScanner
	args         []string
	skipArgument int
	materialize  grammarNamespaceDemandFunc
	condition    grammarConditionDemandFunc
	defaults     grammarDefaultDemandFunc
}

func prepareOperationGrammarPrefixCursorAt(ctx context.Context, operation *invocation, analysis *analysisState, argument, skipArgument int, materialize grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) *grammarPrefixCursor {
	if operation == nil || analysis == nil {
		panic("invocation: nil operation")
	}
	if argument < 0 || argument > len(analysis.args) {
		panic("invocation: grammar prefix argument out of range")
	}
	args := append([]string(nil), analysis.args[:argument]...)
	detached := append([]string(nil), args...)
	scanner := prepareGrammarScanner(operation, analysis, detached, condition)
	cursor := &grammarPrefixCursor{
		scanner:      scanner,
		args:         detached,
		skipArgument: skipArgument,
		materialize:  materialize,
		condition:    condition,
		defaults:     defaults,
	}
	scanner.scan(ctx, detached, false, skipArgument, materialize, condition, defaults, true)
	return cursor
}

func prepareGrammarArgumentsPrefixCursor(ctx context.Context, operation *invocation, args []string, skipArgument int, materialize grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) *grammarPrefixCursor {
	detached := append([]string(nil), args...)
	scanner := prepareGrammarScanner(operation, operation.analysis, detached, condition)
	cursor := &grammarPrefixCursor{
		scanner:      scanner,
		args:         detached,
		skipArgument: skipArgument,
		materialize:  materialize,
		condition:    condition,
		defaults:     defaults,
	}
	scanner.scan(ctx, detached, false, skipArgument, materialize, condition, defaults, true)
	return cursor
}

func (cursor *grammarPrefixCursor) Prefix(ctx context.Context) grammarPrefixResult {
	if cursor == nil || cursor.scanner == nil {
		panic("invocation: nil grammar prefix cursor")
	}
	if cursor.scanner.index == len(cursor.args) {
		return grammarPrefixFromScanner(cursor.scanner, cursor.args)
	}
	preview := cursor.scanner.clone()
	preview.scan(
		ctx,
		cursor.args,
		false,
		cursor.skipArgument,
		cursor.materialize,
		cursor.condition,
		cursor.defaults,
		false,
	)
	return grammarPrefixFromScanner(&preview, cursor.args)
}

func (cursor *grammarPrefixCursor) Append(ctx context.Context, text string) {
	if cursor == nil || cursor.scanner == nil {
		panic("invocation: nil grammar prefix cursor")
	}
	cursor.args = append(cursor.args, text)
	cursor.scanner.scan(
		ctx,
		cursor.args,
		false,
		cursor.skipArgument,
		cursor.materialize,
		cursor.condition,
		cursor.defaults,
		true,
	)
}

// Ready reports whether the settled prefix has enough canonical grammar truth
// for response recognition. It deliberately exposes no mutable scanner state.
func (cursor *grammarPrefixCursor) Ready() bool {
	if cursor == nil || cursor.scanner == nil {
		return false
	}
	return cursor.scanner.ownershipCertain && len(cursor.scanner.result.Issues) == 0 &&
		len(cursor.scanner.result.IncompleteNamespaces) == 0
}

// Dialect returns the current operation-effective dialect without cloning the
// growing prefix result. Callers must check Ready first.
func (cursor *grammarPrefixCursor) Dialect() (dialect, bool) {
	if !cursor.Ready() || cursor.scanner.route == grammarRouteTerminalAction {
		return dialect{}, false
	}
	grammar, ok := grammarOperationSource{
		operation: cursor.scanner.operation,
		analysis:  cursor.scanner.analysis,
	}.scope(cursor.scanner.result.SelectedProfiles)
	if !ok {
		return dialect{}, false
	}
	return grammar.Dialect(cursor.scanner.current)
}

func (cursor *grammarPrefixCursor) RefreshCommittedOrigins() {
	if cursor == nil || cursor.scanner == nil {
		return
	}
	cursor.scanner.refreshOrigins()
}

func analysisAnalyzeGrammarAt(ctx context.Context, operation *invocation, analysis *analysisState, args []string, complete bool, skipArgument int, materialize grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc, valueLanguages grammarValueLanguageDemandFunc) (grammarResult, grammarScanState) {
	scanner := prepareGrammarScanner(operation, analysis, args, condition)
	scanner.scan(ctx, args, complete, skipArgument, materialize, condition, defaults, false)

	result := scanner.result
	model := scanner.model
	current := scanner.current
	route := scanner.route
	ownershipCertain := scanner.ownershipCertain
	demandedNamespaces := scanner.demandedNamespaces
	positional := scanner.positional
	occurrences := scanner.occurrences

	if ownershipCertain && complete && grammarProfilePendingAt(scanner.analysis, current, result.SelectedProfiles) {
		view := grammarOperationSource{
			operation: operation,
			analysis:  analysis,
		}.effectiveScope(model, result.SelectedProfiles)
		if !resolveGrammarProfileAt(ctx, operation, analysis, &result, view, args, current, len(args)) {
			ownershipCertain = false
		}
	}
	if ownershipCertain && complete && !result.Selection.Action.present() && route == grammarRouteSelecting {
		current, ownershipCertain = settleDefaultChainAtBoundaryAt(
			ctx,
			operation,
			analysis, &result, &model, args, current, len(args), materialize, condition, defaults, &demandedNamespaces,
		)
		if ownershipCertain {
			activateAutomaticHelpAtBoundaryAt(operation, analysis, &result, current, len(args) == 0)
		}
	}

	scanner.result = result
	scanner.model = model
	scanner.current = current
	scanner.ownershipCertain = ownershipCertain
	scanner.demandedNamespaces = demandedNamespaces
	state := scanner.state()
	if !ownershipCertain || !complete {
		return finalizedGrammarResult(args, result), state
	}
	if terminalMissingFlagValue(result.Issues) {
		return finalizedGrammarResult(args, result), state
	}
	if result.Selection.ActionSelected() || route == grammarRouteTerminalAction {
		if route == grammarRouteTerminalAction && scanner.actionIncompatible >= 0 {
			grammarIssueValue := grammarIssue{
				Kind:       grammarIssueUnexpectedInput,
				TokenIndex: scanner.actionIncompatible,
				Command:    current,
			}
			appendGrammarIssue(&result, operation.input.limits.Diagnostics, grammarIssueValue)
		}
		return finalizedGrammarResult(args, result), state
	}
	view := grammarOperationSource{
		operation: operation,
		analysis:  analysis,
	}.effectiveScope(model, result.SelectedProfiles)
	view, settled := applyArgumentAvailabilityAt(
		operation,
		analysis,
		&result,
		condition,
		view,
		current,
		firstIndex(positional),
	)
	if !settled {
		state.ownershipCertain = false
		return finalizedGrammarResult(args, result), state
	}
	allocation, formEvaluations, issue := allocateArgumentForm(
		ctx,
		operation,
		analysis,
		result,
		valueLanguages,
		view.ArgumentForms(current),
		args,
		positional,
	)
	result.ArgumentForms = formEvaluations
	if issue.Kind == grammarIssueNoArgumentForm &&
		!issue.ArgumentTerm.present() &&
		issue.TokenIndex == firstIndex(positional) &&
		issue.TokenIndex >= 0 && issue.TokenIndex < len(args) &&
		commandHasSelectableChildren(analysis.effectiveModel, view, current) &&
		!positionalCanOwnToken(view.ArgumentForms(current), args[issue.TokenIndex], !scanner.flagsEnabled) {
		issue.Kind = grammarIssueUnknownCommand
	}
	if issue.Kind == grammarIssueUnavailableArgument {
		if lifecycleKind := unavailableArgumentLifecycleIssue(
			analysis,
			view.ArgumentForms(current),
			issue.Forms,
		); lifecycleKind != grammarIssueNone {
			issue.Kind = lifecycleKind
		}
	}
	if issue.Kind != grammarIssueNone {
		issue.Command = current
		appendGrammarIssue(&result, operation.input.limits.Diagnostics, issue)
		return finalizedGrammarResult(args, result), state
	}
	result.Arguments = allocation
	claimArgumentAllocation(&result, view, allocation)
	view, settled = applyRequiredFlagAvailabilityAt(
		operation,
		analysis,
		&result,
		condition,
		view,
		result.Selection.Path,
		len(args),
	)
	if !settled {
		state.ownershipCertain = false
		return finalizedGrammarResult(args, result), state
	}
	appendMissingFlagOccurrences(&result, operation.input.limits.Diagnostics, view, result.Selection.Path, occurrences)
	return finalizedGrammarResult(args, result), state
}

func commandHasSelectableChildren(model *effectiveModel, view grammarView, command CommandID) bool {
	for _, entry := range view.CommandEntries(command) {
		target, ok := model.Command(entry.target)
		if !ok {
			continue
		}
		if target.origin != DeclarationOriginFrontend {
			return true
		}
	}
	return false
}

type tokenRole uint8

const (
	tokenRoleUnowned tokenRole = iota
	tokenRoleCommand
	tokenRoleAction
	tokenRoleFlag
	tokenRoleFlagValue
	tokenRoleFlagTerminator
	tokenRolePositional
	tokenRoleLiteral
	tokenRoleTokenLanguage
	tokenRoleEndOfOptions
	tokenRoleResponseFile
	tokenRoleRemainder
	tokenRolePassThrough
)

type nameMatchKind uint8

const (
	nameMatchKindNone nameMatchKind = iota
	nameMatchKindExact
	nameMatchKindUniquePrefix
)

// tokenOwnership is the private single-assignment ownership result for one
// invocation argument token. Attached flag values remain owned by the flag
// token; separate flag values receive TokenRoleFlagValue.
type tokenOwnership struct {
	Role         tokenRole
	Command      CommandID
	CommandEntry CommandEntryID
	ActionEntry  ActionEntryID
	Flag         FlagID
	FlagSpelling FlagSpellingID
	ArgumentForm ArgumentFormID
	ArgumentTerm ArgumentTermID
	Destination  ValueID
	Origin       InputOriginID
	Match        nameMatchKind
	Form         flagValueForm
	Occurrence   int
	Element      int
}

// tokenObservation is the smallest lexical unit that receives independent
// semantic ownership. Physical input entries remain separate so compound
// entries such as short-flag clusters can project several authoritative token
// observations without fabricating one owner for the original argv entry.
type tokenObservation struct {
	Text          string
	Origin        InputOriginID
	PhysicalIndex int
	Start         int
	End           int
	Ownership     tokenOwnership
}

type selectionStep struct {
	Entry    CommandEntryID
	Default  CommandDefaultID
	Input    InputOriginID
	Commands []CommandID
}

type actionTrigger uint8

const (
	actionTriggerNone actionTrigger = iota
	actionTriggerEmpty
	actionTriggerStructural
)

type commandSelection struct {
	Root          CommandID
	Selected      CommandID
	Path          []CommandID
	Steps         []selectionStep
	Action        ActionEntryID
	ActionKind    intrinsicAction
	ActionTrigger actionTrigger
}

func (selection commandSelection) ActionSelected() bool {
	return selection.ActionKind != intrinsicActionInvalid
}

type flagOccurrence struct {
	Occurrence      int
	Flag            FlagID
	Spelling        FlagSpellingID
	Command         CommandID
	Destination     ValueID
	TokenIndex      int
	ValueIndices    []int
	TerminatorIndex int
	Attached        string
	Match           nameMatchKind
	Form            flagValueForm
}

type argumentAllocation struct {
	Form  ArgumentFormID
	Terms []termAllocation
}

type argumentFormEvaluationStatus uint8

const (
	argumentFormEvaluationUnknown argumentFormEvaluationStatus = iota
	argumentFormEvaluationNoMatch
	argumentFormEvaluationRejected
	argumentFormEvaluationAccepted
	argumentFormEvaluationAmbiguousAllocation
	argumentFormEvaluationFailed
)

type argumentFormEvaluation struct {
	Form       ArgumentFormID
	Status     argumentFormEvaluationStatus
	Allocation []termAllocation
	Err        error
}

type termAllocation struct {
	Term    ArgumentTermID
	Indices []int
}

type grammarIssueKind uint8

const (
	grammarIssueNone grammarIssueKind = iota
	grammarIssueMissingDialect
	grammarIssueAmbiguousName
	grammarIssueBlockedExplicitEntry
	grammarIssueAmbiguousFlag
	grammarIssueMisplacedFlag
	grammarIssueUnknownFlag
	grammarIssueUnknownCommand
	grammarIssueMissingFlagValue
	grammarIssuePrematureFlagTerminator
	grammarIssueTooManyFlagOccurrences
	grammarIssueMissingFlagOccurrence
	grammarIssueAmbiguousOwnership
	grammarIssueNoArgumentForm
	grammarIssueAmbiguousArgumentForm
	grammarIssueAmbiguousFormAllocation
	grammarIssueRejectedArgumentForm
	grammarIssueArgumentFormFailure
	grammarIssueUnexpectedInput
	grammarIssueActionConflict
	grammarIssueProfileSelection
	grammarIssueUnavailableCommand
	grammarIssueUnavailableFlag
	grammarIssueUnavailableArgument
	grammarIssueNotIntroducedCommand
	grammarIssueNotIntroducedFlag
	grammarIssueNotIntroducedArgument
	grammarIssueRemovedCommand
	grammarIssueRemovedFlag
	grammarIssueRemovedArgument
	grammarIssueConditionEvaluation
	grammarIssueAmbiguousDefault
	grammarIssueBlockedDefault
)

type grammarIssue struct {
	Kind             grammarIssueKind
	TokenIndex       int
	Command          CommandID
	Entries          []CommandEntryID
	Actions          []ActionEntryID
	Flags            []FlagSpellingID
	FlagDeclarations []FlagID
	Forms            []ArgumentFormID
	ArgumentTerm     ArgumentTermID
	Defaults         []CommandDefaultID
	ConditionTarget  conditionTarget
	Err              error
}

// grammarNamespaceMaterialization reports the result of one demand for a
// command's dynamic child namespace. Changed means the operation-effective
// application model may have advanced and grammar must refresh its model.
// Complete means no provider output capable of changing this namespace remains
// unresolved for this operation.
type grammarNamespaceMaterialization struct {
	Changed  bool
	Complete bool
}

// grammarNamespaceDemand is the authoritative structural prefix available at
// the instant one dynamic child namespace becomes demanded. Grammar contains
// only ownership already committed before the unresolved child token; it does
// not include or guess ownership for that token or later input.
type grammarNamespaceDemand struct {
	Command CommandID
	Grammar grammarResult
}

// grammarNamespaceDemandFunc is an operation-owned, at-most-once-capable
// namespace demand hook. Grammar invokes it while command selection is open
// whenever applicable providers must settle before authoritative recognition
// of the immediate namespace.
type grammarNamespaceDemandFunc func(grammarNamespaceDemand) grammarNamespaceMaterialization

// grammarConditionDemand identifies one structural availability condition at
// the exact grammar frontier before its target may influence recognition or
// ownership. The demand function must resolve only dependencies legal at that
// frontier; it must not reparse or revise already-owned input.
type grammarConditionDemand struct {
	Policy  conditionPolicyRecord
	Grammar grammarResult
}

// grammarConditionDemandFunc evaluates one operation-local structural
// availability condition. Satisfied means the declaration participates,
// Unsatisfied means it is known but unavailable, and Blocked/Unsupported are
// analysis failures rather than false conditions.
type grammarConditionDemandFunc func(grammarConditionDemand) conditionOutcome

// grammarDefaultDemand identifies one effective command-default relationship at
// the exact selection point before parent positional ownership. The demand function
// may settle only the condition dependencies legal at that frontier.
type grammarDefaultDemand struct {
	Default defaultRecord
	Grammar grammarResult
}

// grammarDefaultDemandFunc evaluates a conditional default relationship.
// Unconditional defaults are handled directly by grammar and never call it.
type grammarDefaultDemandFunc func(grammarDefaultDemand) conditionOutcome

// grammarValueLanguageDemand identifies a destination whose invocation-scoped
// scalar language must be materialized after one positional form's structural
// allocation is frozen and before that allocation is classified by conversion.
type grammarValueLanguageDemand struct {
	Destination ValueID
	Grammar     grammarResult
}

// grammarValueLanguageMaterialization reports whether the operation-effective
// destination language is ready. Err preserves domain failure/cancellation.
type grammarValueLanguageMaterialization struct {
	Ready bool
	Err   error
}

// grammarValueLanguageDemandFunc realizes an invocation-scoped destination
// language without changing the already-frozen positional allocation.
type grammarValueLanguageDemandFunc func(grammarValueLanguageDemand) grammarValueLanguageMaterialization

type defaultSelectionStatus uint8

const (
	defaultSelectionUnknown defaultSelectionStatus = iota
	defaultSelectionNotEvaluated
	defaultSelectionNone
	defaultSelectionSelected
	defaultSelectionAmbiguous
	defaultSelectionBlocked
)

type defaultSelection struct {
	Status     defaultSelectionStatus
	Candidates []CommandDefaultID
	Applicable []CommandDefaultID
	Blocked    []CommandDefaultID
	Selected   CommandDefaultID
}

type grammarConditionObservation struct {
	Target  conditionTarget
	Effect  ConditionEffect
	Outcome conditionOutcome
}

type grammarResult struct {
	Selection            commandSelection
	Ownership            []tokenOwnership
	Tokens               []tokenObservation
	Occurrences          []flagOccurrence
	Arguments            argumentAllocation
	ArgumentForms        []argumentFormEvaluation
	Issues               []grammarIssue
	IncompleteNamespaces []CommandID
	SelectedProfiles     map[CommandID]GrammarProfileID
	Conditions           []grammarConditionObservation
	DefaultSelections    map[CommandID]defaultSelection
	DefaultConditions    map[CommandDefaultID]conditionOutcome
	Truncated            bool

	tokenOverrides map[int][]tokenObservation
}

type grammarRoute uint8

const (
	grammarRouteSelecting grammarRoute = iota
	grammarRouteHelpPath
	grammarRouteArguments
	grammarRouteTerminalAction
)

// grammarPrefixResult is the structural state after consuming a strict argv
// prefix without applying end-of-invocation completeness requirements. It is
// used by incomplete-input operations such as Completion so they share the
// ordinary grammar recognizer rather than reconstructing command/flag state.
type grammarPrefixResult struct {
	Grammar           grammarResult
	Current           CommandID
	FlagsEnabled      bool
	FlagOccurrences   map[FlagID]int
	PendingFlagValue  *flagOccurrence
	PositionalIndices []int

	route              grammarRoute
	actionIncompatible int
}

func (prefix grammarPrefixResult) Selecting() bool {
	return prefix.route == grammarRouteSelecting
}

func (prefix grammarPrefixResult) HelpPath() bool {
	return prefix.route == grammarRouteHelpPath
}

func (prefix grammarPrefixResult) Arguments() bool {
	return prefix.route == grammarRouteArguments
}

func (prefix grammarPrefixResult) TerminalAction() bool {
	return prefix.route == grammarRouteTerminalAction
}

func (prefix grammarPrefixResult) CommandPathOpen() bool {
	return prefix.Selecting() || prefix.HelpPath()
}

func adoptGrammarPrefixRoute(prefix, source grammarPrefixResult) grammarPrefixResult {
	prefix.route = source.route
	return prefix
}

// argumentCompletionTarget is one operation-effective argument term that can
// own the incomplete cursor token after the already-consumed positional
// prefix. PriorIndices are exactly the completed tokens owned by this term on
// that viable path; bounded token-language completion uses them as its only
// prior token view.
type argumentCompletionTarget struct {
	Form         ArgumentFormID
	Term         ArgumentTermID
	TermIndex    int
	Kind         ArgumentTermKind
	Destination  ValueID
	Element      int
	PriorIndices []int

	// Allocation contains the complete positional tokens on this viable
	// completion path. It deliberately excludes the incomplete cursor token.
	// Completion uses it only when fixed trailing input makes post-cursor
	// positional ownership semantically relevant.
	Allocation argumentAllocation
}

type argumentCompletionAnalysis struct {
	Grammar   grammarResult
	Targets   []argumentCompletionTarget
	Ambiguous bool
	Settled   bool
}

// completionDefaultAnalysis is the effective command scope reached when the
// cursor token does not have an authoritative explicit command/action owner.
// It follows the same namespace/profile/default selection chain as ordinary
// grammar without claiming the incomplete cursor token.
type completionDefaultAnalysis struct {
	Prefix          grammarPrefixResult
	StructuralOwner bool
	Settled         bool
}

// completionSyntaxCandidateKind identifies one statically recognized declaration kind
// that can remain syntactically viable at a cursor prefix. Presentation,
// lifecycle/visibility filtering, client matching, and completion callbacks are
// intentionally downstream concerns.
type completionSyntaxCandidateKind uint8

const (
	completionSyntaxCandidateUnknown completionSyntaxCandidateKind = iota
	completionSyntaxCandidateCommandEntry
	completionSyntaxCandidateCommandAction
	completionSyntaxCandidateFlagSpelling
	completionSyntaxCandidateFlagAction
)

type completionSyntaxCandidate struct {
	Kind         completionSyntaxCandidateKind
	CommandEntry CommandEntryID
	ActionEntry  ActionEntryID
	FlagSpelling FlagSpellingID
	Position     declarationPosition
}

type completionSyntaxCandidates struct {
	CommandEntries []CommandEntryID
	CommandActions []ActionEntryID
	FlagSpellings  []FlagSpellingID
	FlagActions    []ActionEntryID

	CommandOrder []completionSyntaxCandidate
	FlagOrder    []completionSyntaxCandidate
}

func checkedUint32FromInt(value int) uint32 {
	if value < 0 {
		panic("invocation: negative integer conversion")
	}
	converted := uint64(value)
	if converted > uint64(^uint32(0)) {
		panic("invocation: integer conversion overflow")
	}
	return uint32(converted)
}

func checkedUint64FromInt(value int) uint64 {
	if value < 0 {
		panic("invocation: negative integer conversion")
	}
	return uint64(value)
}
