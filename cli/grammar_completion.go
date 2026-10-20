package cli

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

func completionCursorHasFlagOwner(model grammarView, path []CommandID, dialect dialect, cursorText string) bool {
	if isEndOfOptions(dialect, cursorText) {
		return true
	}
	if dialect.ShortClustering() == ShortFlagClusteringEnabled && matchShortCluster(model, path, dialect, cursorText).recognized {
		return true
	}
	return matchFlag(model, path, dialect, cursorText).found
}

// analyzeCompletionSyntax projects the active grammar window for completion
// from an analyzeGrammarPrefix result. It uses the same placement and action
// rules as ordinary recognition and does not match the incomplete token text.
func analyzeCompletionSyntax(operation *invocation, prefix grammarPrefixResult) completionSyntaxCandidates {
	if operation == nil {
		panic("invocation: nil operation")
	}
	model, ok := operation.analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		panic("invocation: application model has no grammar")
	}
	model = grammarOperationSource{
		operation: operation,
		analysis:  operation.analysis,
	}.effectiveScope(model, prefix.Grammar.SelectedProfiles)
	result := completionSyntaxCandidates{}
	if prefix.Selecting() || prefix.HelpPath() {
		for _, entry := range model.CommandEntries(prefix.Current) {
			if entry.enabled {
				result.CommandEntries = append(result.CommandEntries, entry.id)
				completionSyntaxCandidateValue := completionSyntaxCandidate{
					Kind:         completionSyntaxCandidateCommandEntry,
					CommandEntry: entry.id,
					Position:     entry.position,
				}
				result.CommandOrder = append(result.CommandOrder, completionSyntaxCandidateValue)
			}
		}
		if prefix.Selecting() {
			for _, action := range model.ActionCommands(prefix.Current) {
				if action.enabled {
					result.CommandActions = append(result.CommandActions, action.id)
					completionSyntaxCandidateValue2 := completionSyntaxCandidate{
						Kind:        completionSyntaxCandidateCommandAction,
						ActionEntry: action.id,
						Position:    action.position,
					}
					result.CommandOrder = append(result.CommandOrder, completionSyntaxCandidateValue2)
				}
			}
		}
		slices.SortStableFunc(result.CommandOrder, func(left, right completionSyntaxCandidate) int {
			return cmp.Compare(left.Position, right.Position)
		})
	}
	if prefix.HelpPath() || prefix.TerminalAction() {
		return result
	}
	dialect, ok := model.Dialect(prefix.Current)
	if !ok {
		return result
	}
	flagAllowed := prefix.FlagsEnabled &&
		(!prefix.Arguments() || dialect.FlagOrder() == FlagOrderInterspersed)
	if !flagAllowed {
		return result
	}
	for _, flag := range activeFlags(model, prefix.Grammar.Selection.Path) {
		if maximum, bounded := flag.occurrences.Maximum(); bounded && prefix.FlagOccurrences[flag.id] >= maximum {
			continue
		}
		for _, spelling := range flag.Spellings() {
			result.FlagSpellings = append(result.FlagSpellings, spelling.id)
			completionSyntaxCandidateValue3 := completionSyntaxCandidate{
				Kind:         completionSyntaxCandidateFlagSpelling,
				FlagSpelling: spelling.id,
				Position:     spelling.position,
			}
			result.FlagOrder = append(result.FlagOrder, completionSyntaxCandidateValue3)
		}
	}
	for _, action := range activeActionFlags(model, prefix.Grammar.Selection.Path) {
		if action.enabled {
			result.FlagActions = append(result.FlagActions, action.id)
			completionSyntaxCandidateValue4 := completionSyntaxCandidate{
				Kind:        completionSyntaxCandidateFlagAction,
				ActionEntry: action.id,
				Position:    action.position,
			}
			result.FlagOrder = append(result.FlagOrder, completionSyntaxCandidateValue4)
		}
	}
	slices.SortStableFunc(result.FlagOrder, func(left, right completionSyntaxCandidate) int {
		return cmp.Compare(left.Position, right.Position)
	})
	return result
}

// analyzeCompletionDefaultScope settles the no-explicit-owner branch at the
// completion cursor. The caller may project incomplete command/flag candidates
// from each prefix in parallel. StructuralOwner reports that the current token
// already has a recognized command/action/flag/end-of-options owner, in which
// case defaults and positional ownership cannot compete with it.
func analyzeCompletionDefaultScope(ctx context.Context, operation *invocation, prefix grammarPrefixResult, tokenIndex int, cursorText string, materialize grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc) completionDefaultAnalysis {
	analysis := completionDefaultAnalysis{
		Prefix:  cloneGrammarPrefix(prefix),
		Settled: true,
	}
	if operation == nil || !prefix.Selecting() || prefix.PendingFlagValue != nil {
		return analysis
	}
	model, ok := operation.analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		analysis.Settled = false
		return analysis
	}
	demandedNamespaces := make(map[CommandID]struct{})
	current := prefix.Current
	for {
		view := grammarOperationSource{
			operation: operation,
			analysis:  operation.analysis,
		}.effectiveScope(model, analysis.Prefix.Grammar.SelectedProfiles)
		dialect, ok := view.Dialect(current)
		if !ok {
			analysis.Settled = false
			return analysis
		}

		var settled bool
		view, settled = applyFlagAvailability(
			operation,
			&analysis.Prefix.Grammar,
			condition,
			view,
			analysis.Prefix.Grammar.Selection.Path,
			dialect,
			cursorText,
			tokenIndex,
		)
		if !settled {
			analysis.Settled = false
			return analysis
		}
		if grammarProfilePending(operation, current, analysis.Prefix.Grammar.SelectedProfiles) {
			if profilePrefixCanConsume(
				operation,
				view,
				analysis.Prefix.Grammar.Selection.Path,
				dialect,
				cursorText,
			) {
				analysis.StructuralOwner = true
				return analysis
			}

			selector := grammarProfileSelector(operation, current)
			beforeGrammar := grammarDemandSnapshot(analysis.Prefix.Grammar)
			beforeSeed := operationDependencySeed(operation.analysis)
			if !resolveGrammarProfile(
				ctx,
				operation,
				&analysis.Prefix.Grammar,
				view,
				operation.Args(),
				current,
				tokenIndex,
			) {
				afterSeed := operationDependencySeed(operation.analysis)
				resolution, present := afterSeed.Destinations[selector]
				if !present || resolution.State == destinationStateAbsent ||
					resolution.State == destinationStateInactive {
					analysis.Prefix.Grammar = beforeGrammar
					storeOperationDependencySeed(operation.analysis, beforeSeed)
					analysis.Prefix.Current = current
					return analysis
				}
				analysis.Settled = false
				return analysis
			}
			continue
		}

		if analysis.Prefix.FlagsEnabled && completionCursorHasFlagOwner(view, analysis.Prefix.Grammar.Selection.Path, dialect, cursorText) {
			analysis.StructuralOwner = true
			return analysis
		}

		if _, demanded := demandedNamespaces[current]; !demanded && materialize != nil {
			demandedNamespaces[current] = struct{}{}
			state := materialize(grammarNamespaceDemand{
				Command: current,
				Grammar: grammarDemandSnapshot(analysis.Prefix.Grammar),
			})
			if state.Changed {
				refreshed, ok := operation.analysis.effectiveModel.EffectiveGrammar()
				if !ok {
					analysis.Settled = false
					return analysis
				}
				model = refreshed
			}
			if !state.Complete {
				analysis.Prefix.Grammar.IncompleteNamespaces = appendUniqueCommandID(
					analysis.Prefix.Grammar.IncompleteNamespaces,
					current,
				)
				analysis.Settled = false
				return analysis
			}
		}

		view = grammarOperationSource{
			operation: operation,
			analysis:  operation.analysis,
		}.effectiveScope(model, analysis.Prefix.Grammar.SelectedProfiles)
		dialect, ok = view.Dialect(current)
		if !ok {
			analysis.Settled = false
			return analysis
		}
		view, settled = applyCommandAvailability(
			operation,
			&analysis.Prefix.Grammar,
			condition,
			view,
			current,
			dialect,
			cursorText,
			tokenIndex,
		)
		if !settled {
			analysis.Settled = false
			return analysis
		}
		if match := matchChild(view, current, dialect, cursorText); match.found {
			analysis.StructuralOwner = true
			return analysis
		}

		next, selected := selectDefaultCommand(
			operation,
			&analysis.Prefix.Grammar,
			defaults,
			condition,
			view,
			current,
			tokenIndex,
		)
		if !selected {
			analysis.Settled = false
			return analysis
		}
		if next == current {
			analysis.Prefix.Current = current
			return analysis
		}
		current = next
		analysis.Prefix.Current = current
	}
}

// analyzeArgumentCompletion determines positional cursor ownership from the
// same effective grammar used by ordinary invocation analysis. It does not
// claim the incomplete token. Several distinct terms may remain viable; a
// single term reached through incompatible prior allocations is reported as
// ambiguous because its callback-visible token history would otherwise be
// guessed.
func analyzeArgumentCompletion(operation *invocation, prefix grammarPrefixResult, cursorText string, materialize grammarConditionDemandFunc) argumentCompletionAnalysis {
	result := argumentCompletionAnalysis{
		Grammar: grammarDemandSnapshot(prefix.Grammar),
		Settled: true,
	}
	if operation == nil || prefix.PendingFlagValue != nil || prefix.HelpPath() || prefix.TerminalAction() {
		return result
	}
	model, ok := operation.analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		result.Settled = false
		return result
	}
	model = grammarOperationSource{
		operation: operation,
		analysis:  operation.analysis,
	}.effectiveScope(model, result.Grammar.SelectedProfiles)
	model, settled := applyArgumentAvailability(
		operation,
		&result.Grammar,
		materialize,
		model,
		prefix.Current,
		len(operation.Args())-1,
	)
	if !settled {
		result.Settled = false
		return result
	}
	dialect, ok := model.Dialect(prefix.Current)
	if !ok {
		result.Settled = false
		return result
	}
	for _, form := range model.ArgumentForms(prefix.Current) {
		if !form.enabled {
			continue
		}
		targets := argumentFormCompletionTargets(
			form,
			operation.Args(),
			prefix.PositionalIndices,
			cursorText,
			prefix.FlagsEnabled,
			prefix.Arguments(),
			dialect,
		)
		for _, target := range targets {
			duplicate := -1
			for i := range result.Targets {
				if result.Targets[i].Form == target.Form && result.Targets[i].Term == target.Term {
					duplicate = i
					break
				}
			}
			if duplicate < 0 {
				result.Targets = append(result.Targets, target)
				continue
			}
			if !equalArgumentCompletionPrior(result.Targets[duplicate], target) {
				result.Ambiguous = true
			}
		}
	}
	return result
}

// analyzeArgumentCompletionAt determines cursor ownership when the supplied
// prefix contains complete positional tokens on both sides of the incomplete
// cursor. This is the same Completion allocation model as AnalyzeArgumentCompletion;
// the explicit cursor coordinate preserves lexical order for fixed trailing input.
func analyzeArgumentCompletionAt(operation *invocation, prefix grammarPrefixResult, cursorArgument int, cursorText string, materialize grammarConditionDemandFunc) argumentCompletionAnalysis {
	result := argumentCompletionAnalysis{
		Grammar: grammarDemandSnapshot(prefix.Grammar),
		Settled: true,
	}
	if operation == nil || cursorArgument < 0 || cursorArgument >= len(operation.Args()) ||
		prefix.PendingFlagValue != nil ||
		prefix.HelpPath() ||
		prefix.TerminalAction() {
		return result
	}
	model, ok := operation.analysis.effectiveModel.EffectiveGrammar()
	if !ok {
		result.Settled = false
		return result
	}
	model = grammarOperationSource{
		operation: operation,
		analysis:  operation.analysis,
	}.effectiveScope(model, result.Grammar.SelectedProfiles)
	model, settled := applyArgumentAvailability(
		operation,
		&result.Grammar,
		materialize,
		model,
		prefix.Current,
		cursorArgument,
	)
	if !settled {
		result.Settled = false
		return result
	}
	dialect, ok := model.Dialect(prefix.Current)
	if !ok {
		result.Settled = false
		return result
	}
	for _, form := range model.ArgumentForms(prefix.Current) {
		if !form.enabled {
			continue
		}
		allocations := argumentFormCompletionAllocationsAt(
			form,
			operation.Args(),
			prefix.PositionalIndices,
			cursorArgument,
			cursorText,
			prefix.FlagsEnabled,
			prefix.Arguments(),
			dialect,
		)
		for _, allocation := range allocations {
			target := allocation.target
			target.Allocation = allocation.allocation
			duplicate := -1
			for i := range result.Targets {
				if result.Targets[i].Form == target.Form && result.Targets[i].Term == target.Term {
					duplicate = i
					break
				}
			}
			if duplicate < 0 {
				result.Targets = append(result.Targets, target)
				continue
			}
			if !equalArgumentCompletionPrior(result.Targets[duplicate], target) ||
				!equalArgumentAllocation(result.Targets[duplicate].Allocation, target.Allocation) {
				result.Ambiguous = true
			}
		}
	}
	return result
}

type completionArgumentToken struct {
	index  int
	cursor bool
}

func argumentFormCompletionAllocationsAt(form argumentFormGrammar, args []string, indices []int, cursorArgument int, cursor string, flagsEnabled, argumentsStarted bool, dialect dialect) []argumentCompletionAllocation {
	declared := form.Terms()
	terms := make([]argumentTermGrammar, 0, len(declared))
	termIndexes := make([]int, 0, len(declared))
	for index, term := range declared {
		if term.Enabled() {
			terms = append(terms, term)
			termIndexes = append(termIndexes, index)
		}
	}

	tokens := make([]completionArgumentToken, 0, len(indices)+1)
	inserted := false
	for _, index := range indices {
		if index == cursorArgument {
			continue
		}
		if !inserted && index > cursorArgument {
			tokens = append(tokens, completionArgumentToken{
				index:  cursorArgument,
				cursor: true,
			})
			inserted = true
		}
		tokens = append(tokens, completionArgumentToken{
			index: index,
		})
	}
	if !inserted {
		tokens = append(tokens, completionArgumentToken{
			index:  cursorArgument,
			cursor: true,
		})
	}

	var candidates []argumentCompletionAllocation
	var visit func(int, int, []int, []termAllocation, *argumentCompletionTarget)
	visit = func(termIndex, tokenIndex int, counts []int, allocations []termAllocation, target *argumentCompletionTarget) {
		if tokenIndex == len(tokens) {
			if target == nil {
				return
			}
			copyTarget := *target
			copyTarget.PriorIndices = append([]int(nil), target.PriorIndices...)
			candidate := argumentCompletionAllocation{
				target: copyTarget,
				counts: append([]int(nil), counts...),
				allocation: argumentAllocation{
					Form:  form.id,
					Terms: cloneTermAllocations(allocations),
				},
			}
			candidates = append(candidates, candidate)
			return
		}
		if termIndex >= len(terms) {
			return
		}

		term := terms[termIndex]
		if term.kind == ArgumentTermLiteral {
			token := tokens[tokenIndex]
			if token.cursor {
				if !strings.HasPrefix(term.literal, cursor) {
					return
				}
				nextTarget := &argumentCompletionTarget{
					Form:      form.id,
					Term:      term.id,
					TermIndex: termIndexes[termIndex],
					Kind:      term.kind,
				}
				visit(termIndex+1, tokenIndex+1, appendCompletionAllocationCount(counts, 1), allocations, nextTarget)
				return
			}
			if token.index < 0 || token.index >= len(args) || args[token.index] != term.literal {
				return
			}
			visit(
				termIndex+1,
				tokenIndex+1,
				appendCompletionAllocationCount(counts, 1),
				appendCompletionTermAllocation(allocations, term.id, []int{token.index}),
				target,
			)
			return
		}

		remaining := len(tokens) - tokenIndex
		maximum, bounded := term.count.Maximum()
		if !bounded || maximum > remaining {
			maximum = remaining
		}
		for count := 0; count <= maximum; count++ {
			segment := tokens[tokenIndex : tokenIndex+count]
			cursorOffset := -1
			complete := make([]int, 0, len(segment))
			prior := make([]int, 0, len(segment))
			for offset, token := range segment {
				if token.cursor {
					cursorOffset = offset
					continue
				}
				complete = append(complete, token.index)
				if cursorOffset < 0 {
					prior = append(prior, token.index)
				}
			}

			transfer := term.kind == ArgumentTermRemainder ||
				term.kind == ArgumentTermPassThrough
			if transfer && count != remaining {
				continue
			}
			if cursorOffset < 0 && count < term.count.Minimum() {
				continue
			}
			if cursorOffset >= 0 &&
				!argumentCompletionTermCanOwnCursor(term, cursor, flagsEnabled, argumentsStarted, dialect) {
				continue
			}

			nextTarget := target
			if cursorOffset >= 0 {
				if target != nil {
					continue
				}
				destination, _ := term.Destination()
				nextTarget = &argumentCompletionTarget{
					Form:         form.id,
					Term:         term.id,
					TermIndex:    termIndexes[termIndex],
					Kind:         term.kind,
					Destination:  destination,
					Element:      cursorOffset,
					PriorIndices: append([]int(nil), prior...),
				}
			}
			nextAllocations := allocations
			if len(complete) != 0 {
				nextAllocations = appendCompletionTermAllocation(allocations, term.id, complete)
			}
			visit(
				termIndex+1,
				tokenIndex+count,
				appendCompletionAllocationCount(counts, count),
				nextAllocations,
				nextTarget,
			)
		}
	}
	visit(0, 0, nil, nil, nil)
	if len(candidates) == 0 {
		return nil
	}

	active := append([]argumentCompletionAllocation(nil), candidates...)
	for termIndex, term := range terms {
		minimum, maximum, differs := completionAllocationRange(active, termIndex)
		if !differs {
			continue
		}
		switch term.allocation {
		case semanticArgumentAllocationGreedy:
			active = filterCompletionAllocations(active, termIndex, maximum)
		case semanticArgumentAllocationMinimal:
			active = filterCompletionAllocations(active, termIndex, minimum)
		default:
			return active
		}
		if len(active) <= 1 {
			break
		}
	}
	return active
}

func appendCompletionTermAllocation(allocations []termAllocation, term ArgumentTermID, indices []int) []termAllocation {
	result := cloneTermAllocations(allocations)
	result = append(result, termAllocation{
		Term:    term,
		Indices: append([]int(nil), indices...),
	})
	return result
}

func cloneTermAllocations(allocations []termAllocation) []termAllocation {
	result := make([]termAllocation, len(allocations))
	for i, allocation := range allocations {
		result[i] = termAllocation{
			Term:    allocation.Term,
			Indices: append([]int(nil), allocation.Indices...),
		}
	}
	return result
}

func equalArgumentAllocation(left, right argumentAllocation) bool {
	if left.Form != right.Form || len(left.Terms) != len(right.Terms) {
		return false
	}
	for i := range left.Terms {
		if left.Terms[i].Term != right.Terms[i].Term || len(left.Terms[i].Indices) != len(right.Terms[i].Indices) {
			return false
		}
		for j := range left.Terms[i].Indices {
			if left.Terms[i].Indices[j] != right.Terms[i].Indices[j] {
				return false
			}
		}
	}
	return true
}

func argumentFormCompletionTargets(form argumentFormGrammar, args []string, indices []int, cursor string, flagsEnabled, argumentsStarted bool, dialect dialect) []argumentCompletionTarget {
	declared := form.Terms()
	terms := make([]argumentTermGrammar, 0, len(declared))
	termIndexes := make([]int, 0, len(declared))
	for index, term := range declared {
		if term.Enabled() {
			terms = append(terms, term)
			termIndexes = append(termIndexes, index)
		}
	}
	type key struct{ term, token int }
	visiting := make(map[key]struct{})
	var candidates []argumentCompletionAllocation
	var visit func(int, int, []int)
	visit = func(termIndex, tokenIndex int, counts []int) {
		if termIndex >= len(terms) {
			return
		}
		k := key{termIndex, tokenIndex}
		if _, ok := visiting[k]; ok {
			return
		}
		visiting[k] = struct{}{}
		defer delete(visiting, k)
		term := terms[termIndex]
		if term.kind == ArgumentTermLiteral {
			if tokenIndex < len(indices) {
				if args[indices[tokenIndex]] == term.literal {
					visit(termIndex+1, tokenIndex+1, appendCompletionAllocationCount(counts, 1))
				}
				return
			}
			if strings.HasPrefix(term.literal, cursor) {
				candidates = append(candidates, argumentCompletionAllocation{
					target: argumentCompletionTarget{
						Form:      form.id,
						Term:      term.id,
						TermIndex: termIndexes[termIndex],
						Kind:      term.kind,
					},
					counts: appendCompletionAllocationCount(counts, 1),
				})
			}
			return
		}

		remaining := len(indices) - tokenIndex
		minimum := term.count.Minimum()
		maximum, bounded := term.count.Maximum()
		if !bounded || maximum > remaining {
			maximum = remaining
		}
		for count := 0; count <= maximum; count++ {
			if count > remaining {
				break
			}
			transfer := term.kind == ArgumentTermRemainder ||
				term.kind == ArgumentTermPassThrough
			if count >= minimum && (!transfer || count == remaining) {
				visit(termIndex+1, tokenIndex+count, appendCompletionAllocationCount(counts, count))
			}
			if tokenIndex+count != len(indices) ||
				!argumentCompletionTermCanOwnCursor(term, cursor, flagsEnabled, argumentsStarted, dialect) {
				continue
			}
			if boundedMaximum, bounded := term.count.Maximum(); bounded && count >= boundedMaximum {
				continue
			}
			destination, _ := term.Destination()
			candidates = append(candidates, argumentCompletionAllocation{
				target: argumentCompletionTarget{
					Form:         form.id,
					Term:         term.id,
					TermIndex:    termIndexes[termIndex],
					Kind:         term.kind,
					Destination:  destination,
					Element:      count,
					PriorIndices: append([]int(nil), indices[tokenIndex:tokenIndex+count]...),
				},
				counts: appendCompletionAllocationCount(counts, count+1),
			})
		}
	}
	visit(0, 0, nil)
	if len(candidates) == 0 {
		return nil
	}

	// Completion applies the same declaration-order structural ownership rule
	// as complete-input allocation. The cursor token participates in the count
	// vector for its prospective owner. This is the only way a Greedy/Minimal
	// declaration can decide whether the incomplete token belongs to an earlier
	// variable term or to a later term without introducing a completion-only
	// ownership rule.
	active := append([]argumentCompletionAllocation(nil), candidates...)
	for termIndex, term := range terms {
		minimum, maximum, differs := completionAllocationRange(active, termIndex)
		if !differs {
			continue
		}
		switch term.allocation {
		case semanticArgumentAllocationGreedy:
			active = filterCompletionAllocations(active, termIndex, maximum)
		case semanticArgumentAllocationMinimal:
			active = filterCompletionAllocations(active, termIndex, minimum)
		default:
			// In incomplete-input analysis, an unspecified first differing
			// allocation remains several viable cursor targets rather than an
			// invocation rejection. Target identity preserves that uncertainty.
			// Ambiguous is reserved for one target identity reached through
			// incompatible prior ownership histories, which a callback request
			// cannot represent without guessing.
			return completionAllocationTargets(active)
		}
		if len(active) <= 1 {
			break
		}
	}
	return completionAllocationTargets(active)
}

func appendCompletionAllocationCount(counts []int, count int) []int {
	result := make([]int, len(counts)+1)
	copy(result, counts)
	result[len(counts)] = count
	return result
}

type argumentCompletionAllocation struct {
	target     argumentCompletionTarget
	counts     []int
	allocation argumentAllocation
}

func completionAllocationRange(candidates []argumentCompletionAllocation, termIndex int) (int, int, bool) {
	set := false
	minimum, maximum := 0, 0
	for _, candidate := range candidates {
		if termIndex >= len(candidate.counts) {
			continue
		}
		count := candidate.counts[termIndex]
		if !set {
			minimum, maximum, set = count, count, true
			continue
		}
		if count < minimum {
			minimum = count
		}
		if count > maximum {
			maximum = count
		}
	}
	return minimum, maximum, set && minimum != maximum
}

func filterCompletionAllocations(candidates []argumentCompletionAllocation, termIndex, count int) []argumentCompletionAllocation {
	filtered := candidates[:0]
	for _, candidate := range candidates {
		if termIndex < len(candidate.counts) && candidate.counts[termIndex] == count {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func completionAllocationTargets(candidates []argumentCompletionAllocation) []argumentCompletionTarget {
	result := make([]argumentCompletionTarget, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.target)
	}
	return result
}

func argumentCompletionTermCanOwnCursor(term argumentTermGrammar, cursor string, flagsEnabled, argumentsStarted bool, dialect dialect) bool {
	if term.kind == ArgumentTermLiteral {
		return strings.HasPrefix(term.literal, cursor)
	}
	if !flagsEnabled || argumentsStarted || !isUnknownHyphenFlagToken(cursor) {
		return true
	}
	// When the incomplete token is flag-shaped, positional ownership is viable
	// only under the same hyphen-value rule ordinary grammar uses before the
	// positional stream begins. Interspersed flag order does not grant a value
	// term ownership of an otherwise flag-owned spelling.
	_ = dialect
	return hyphenValueAllowed(term.hyphenValues, cursor)
}

func equalArgumentCompletionPrior(left, right argumentCompletionTarget) bool {
	if left.Element != right.Element || len(left.PriorIndices) != len(right.PriorIndices) {
		return false
	}
	for i := range left.PriorIndices {
		if left.PriorIndices[i] != right.PriorIndices[i] {
			return false
		}
	}
	return true
}
