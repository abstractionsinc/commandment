package cli

import "context"

type formMatch struct {
	form       argumentFormGrammar
	allocation []termAllocation
}

type argumentFormClassification struct {
	form          ArgumentFormID
	allocation    []termAllocation
	status        argumentFormEvaluationStatus
	rejectionTerm ArgumentTermID
	err           error
}

func reduceArgumentFormClassifications(classifications []argumentFormClassification, tokenIndex int) (argumentAllocation, grammarIssue) {
	accepted := make([]argumentFormClassification, 0, len(classifications))
	var rejected []ArgumentFormID
	var failed []ArgumentFormID
	var firstRejected error
	var firstRejectedTerm ArgumentTermID
	firstRejectedSet := false
	var firstFailure error
	for _, classification := range classifications {
		switch classification.status {
		case argumentFormEvaluationAccepted:
			accepted = append(accepted, classification)
		case argumentFormEvaluationRejected:
			rejected = append(rejected, classification.form)
			if !firstRejectedSet {
				firstRejectedSet = true
				firstRejected = classification.err
				firstRejectedTerm = classification.rejectionTerm
			}
		case argumentFormEvaluationFailed:
			failed = append(failed, classification.form)
			if firstFailure == nil {
				firstFailure = classification.err
			}
		case argumentFormEvaluationUnknown, argumentFormEvaluationNoMatch, argumentFormEvaluationAmbiguousAllocation:
		}
	}
	if len(failed) != 0 {
		result := grammarIssue{
			Kind:       grammarIssueArgumentFormFailure,
			TokenIndex: tokenIndex,
			Forms:      failed,
			Err:        firstFailure,
		}
		return argumentAllocation{}, result
	}
	if len(accepted) > 1 {
		ids := make([]ArgumentFormID, len(accepted))
		for i, classification := range accepted {
			ids[i] = classification.form
		}
		result := grammarIssue{
			Kind:       grammarIssueAmbiguousArgumentForm,
			TokenIndex: tokenIndex,
			Forms:      ids,
		}
		return argumentAllocation{}, result
	}
	if len(accepted) == 1 {
		allocation := argumentAllocation{
			Form:  accepted[0].form,
			Terms: cloneTermAllocations(accepted[0].allocation),
		}
		return allocation, grammarIssue{}
	}
	if len(rejected) != 0 {
		result := grammarIssue{
			Kind:         grammarIssueRejectedArgumentForm,
			TokenIndex:   tokenIndex,
			Forms:        rejected,
			ArgumentTerm: firstRejectedTerm,
			Err:          firstRejected,
		}
		return argumentAllocation{}, result
	}
	issue := grammarIssue{
		Kind:       grammarIssueNoArgumentForm,
		TokenIndex: tokenIndex,
	}
	return argumentAllocation{}, issue
}

func allocateArgumentForm(ctx context.Context, operation *invocation, analysis *analysisState, grammar grammarResult, valueLanguages grammarValueLanguageDemandFunc, forms []argumentFormGrammar, args []string, indices []int) (argumentAllocation, []argumentFormEvaluation, grammarIssue) {
	var matches []formMatch
	evaluations := make([]argumentFormEvaluation, 0, len(forms))
	for _, form := range forms {
		if !form.enabled {
			continue
		}
		allocation, match, isAmbiguous := matchForm(form, args, indices)
		if isAmbiguous {
			argumentFormEvaluationValue2 := argumentFormEvaluation{
				Form:       form.id,
				Status:     argumentFormEvaluationAmbiguousAllocation,
				Allocation: cloneTermAllocations(allocation),
			}
			evaluations = append(evaluations, argumentFormEvaluationValue2)
			continue
		}
		if match {
			matches = append(matches, formMatch{
				form:       form,
				allocation: allocation,
			})
		} else {
			evaluations = append(evaluations, argumentFormEvaluation{
				Form:   form.id,
				Status: argumentFormEvaluationNoMatch,
			})
		}
	}
	if len(matches) == 0 {
		return argumentAllocation{}, evaluations, noArgumentFormMatchIssue(forms, args, indices, evaluations)
	}

	classifications := make([]argumentFormClassification, 0, len(matches))
	for _, match := range matches {
		status, rejectionTerm, err := evaluateArgumentForm(
			ctx,
			operation,
			analysis,
			grammar,
			valueLanguages,
			match.form,
			match.allocation,
			args,
		)
		argumentFormEvaluationValue := argumentFormEvaluation{
			Form:       match.form.id,
			Status:     status,
			Allocation: cloneTermAllocations(match.allocation),
			Err:        err,
		}
		evaluations = append(evaluations, argumentFormEvaluationValue)
		argumentFormClassificationValue := argumentFormClassification{
			form:          match.form.id,
			allocation:    match.allocation,
			status:        status,
			rejectionTerm: rejectionTerm,
			err:           err,
		}
		classifications = append(classifications, argumentFormClassificationValue)
	}
	allocation, issue := reduceArgumentFormClassifications(classifications, firstIndex(indices))
	return allocation, evaluations, issue
}

func noArgumentFormMatchIssue(forms []argumentFormGrammar, args []string, indices []int, evaluations []argumentFormEvaluation) grammarIssue {
	var unavailable []ArgumentFormID
	for _, form := range forms {
		if form.enabled && allArgumentTermsEnabled(form) {
			continue
		}
		probe := form
		probe.enabled = true
		terms := probe.Terms()
		for index := range terms {
			terms[index].disabled = false
		}
		probe.terms = append([]argumentTermGrammar(nil), terms...)
		if _, match, ambiguous := matchForm(probe, args, indices); match || ambiguous {
			unavailable = append(unavailable, form.id)
		}
	}
	if len(unavailable) != 0 {
		result := grammarIssue{
			Kind:       grammarIssueUnavailableArgument,
			TokenIndex: firstIndex(indices),
			Forms:      unavailable,
		}

		return result
	}
	if len(indices) == 0 && len(forms) == 0 {
		return grammarIssue{}
	}
	var ambiguous []ArgumentFormID
	for _, evaluation := range evaluations {
		if evaluation.Status == argumentFormEvaluationAmbiguousAllocation {
			ambiguous = append(ambiguous, evaluation.Form)
		}
	}
	if len(ambiguous) != 0 {
		result := grammarIssue{
			Kind:       grammarIssueAmbiguousFormAllocation,
			TokenIndex: firstIndex(indices),
			Forms:      ambiguous,
		}

		return result
	}
	issue := grammarIssue{
		Kind:       grammarIssueNoArgumentForm,
		TokenIndex: firstIndex(indices),
	}
	return issue
}

func evaluateArgumentForm(ctx context.Context, operation *invocation, analysis *analysisState, grammar grammarResult, valueLanguages grammarValueLanguageDemandFunc, form argumentFormGrammar, allocation []termAllocation, args []string) (argumentFormEvaluationStatus, ArgumentTermID, error) {
	terms := make(map[ArgumentTermID]argumentTermGrammar, len(form.Terms()))
	for _, term := range form.Terms() {
		terms[term.id] = term
	}
	rejected := false
	var rejection error
	var rejectionTerm ArgumentTermID
	rejectionSet := false
	var failure error
	for _, owned := range allocation {
		term, ok := terms[owned.Term]
		if !ok {
			return argumentFormEvaluationFailed, ArgumentTermID(0), errInvalidValueLanguageInput
		}
		texts := make([]string, len(owned.Indices))
		for i, index := range owned.Indices {
			if index < 0 || index >= len(args) {
				return argumentFormEvaluationFailed, ArgumentTermID(0), errInvalidValueLanguageInput
			}
			texts[i] = args[index]
		}
		switch term.kind {
		case ArgumentTermLiteral,
			ArgumentTermRemainder,
			ArgumentTermPassThrough:
			continue
		case ArgumentTermTokens:
			language, ok := term.TokenLanguage()
			if !ok {
				return argumentFormEvaluationFailed, ArgumentTermID(0), errInvalidValueLanguageInput
			}
			outcome := operation.evaluateArgumentTokenLanguage(ctx, analysis, term.id, language, owned.Indices, texts)
			switch outcome.Kind() {
			case conversionOutcomeAccepted:
			case conversionOutcomeRejected:
				rejected = true
				if !rejectionSet {
					rejectionSet = true
					rejection = outcome.Err()
					rejectionTerm = owned.Term
				}
			default:
				if failure == nil {
					failure = outcome.Err()
				}
			}
		case ArgumentTermValue:
			if len(texts) == 0 {
				continue
			}
			destination, ok := term.Destination()
			if !ok {
				continue
			}
			definition, ok := analysisDestinationDefinition(analysis, destination)
			if !ok {
				continue
			}
			language, ok := definition.Language()
			if !ok {
				continue
			}
			language, ready, materializationErr := materializeArgumentValueLanguage(
				analysis, grammar, valueLanguages, destination, language,
			)
			if materializationErr != nil {
				if failure == nil {
					failure = materializationErr
				}
				continue
			}
			if !ready {
				continue
			}
			for i, text := range texts {
				outcome := operation.evaluateValueLanguage(analysis, destination, language, text, owned.Indices[i])
				switch outcome.Kind() {
				case conversionOutcomeAccepted:
				case conversionOutcomeRejected:
					rejected = true
					if !rejectionSet {
						rejectionSet = true
						rejection = outcome.Err()
						rejectionTerm = owned.Term
					}
				default:
					if failure == nil {
						failure = outcome.Err()
					}
				}
			}
		case ArgumentTermUnknown, ArgumentTermSequence:
		}
	}
	if failure != nil {
		return argumentFormEvaluationFailed, ArgumentTermID(0), failure
	}
	if rejected {
		return argumentFormEvaluationRejected, rejectionTerm, rejection
	}
	return argumentFormEvaluationAccepted, ArgumentTermID(0), nil
}

func materializeArgumentValueLanguage(analysis *analysisState, grammar grammarResult, materialize grammarValueLanguageDemandFunc, destination ValueID, language valueLanguage) (valueLanguage, bool, error) {
	if !language.DynamicDomain() {
		return language, true, nil
	}
	if materialize == nil {
		return language, false, nil
	}
	state := materialize(grammarValueLanguageDemand{
		Destination: destination,
		Grammar:     grammarDemandSnapshot(grammar),
	})
	if state.Err != nil {
		return language, false, state.Err
	}
	if !state.Ready {
		return language, false, errInvalidValueLanguageInput
	}
	definition, ok := analysisDestinationDefinition(analysis, destination)
	if !ok {
		return language, false, errInvalidValueLanguageInput
	}
	materialized, ok := definition.Language()
	if !ok || materialized.DynamicDomain() {
		return language, false, errInvalidValueLanguageInput
	}
	return materialized, true, nil
}

func allArgumentTermsEnabled(form argumentFormGrammar) bool {
	for _, term := range form.Terms() {
		if !term.Enabled() {
			return false
		}
	}
	return true
}

func firstIndex(indices []int) int {
	if len(indices) == 0 {
		return -1
	}
	return indices[0]
}

type allocationKey struct{ term, token int }

type allocationState struct {
	matched    bool
	ambiguous  bool
	allocation []termAllocation
}

type countedAllocation struct {
	count int
	state allocationState
}

func matchForm(form argumentFormGrammar, args []string, indices []int) ([]termAllocation, bool, bool) {
	declared := form.Terms()
	terms := make([]argumentTermGrammar, 0, len(declared))
	for _, term := range declared {
		if term.Enabled() {
			terms = append(terms, term)
		}
	}
	memo := make(map[allocationKey]allocationState)
	var visit func(int, int) allocationState
	visit = func(termIndex, tokenIndex int) allocationState {
		key := allocationKey{
			term:  termIndex,
			token: tokenIndex,
		}
		if cached, ok := memo[key]; ok {
			return cached
		}
		if termIndex == len(terms) {
			state := allocationState{
				matched: tokenIndex == len(indices),
			}
			memo[key] = state
			return state
		}

		term := terms[termIndex]
		if term.kind == ArgumentTermLiteral {
			if tokenIndex >= len(indices) || args[indices[tokenIndex]] != term.literal {
				memo[key] = allocationState{}
				return allocationState{}
			}
			next := visit(termIndex+1, tokenIndex+1)
			if !next.matched {
				memo[key] = allocationState{}
				return allocationState{}
			}
			state := prependAllocation(term, indices[tokenIndex:tokenIndex+1], next)
			memo[key] = state
			return state
		}

		minimum := term.count.Minimum()
		maximum, bounded := term.count.Maximum()
		remaining := len(indices) - tokenIndex
		if !bounded || maximum > remaining {
			maximum = remaining
		}
		if minimum > maximum {
			memo[key] = allocationState{}
			return allocationState{}
		}

		var candidates []countedAllocation
		for count := minimum; count <= maximum; count++ {
			if (term.kind == ArgumentTermRemainder || term.kind == ArgumentTermPassThrough) &&
				count != remaining {
				continue
			}
			next := visit(termIndex+1, tokenIndex+count)
			if next.matched {
				candidates = append(candidates, countedAllocation{
					count: count,
					state: next,
				})
			}
		}
		if len(candidates) == 0 {
			memo[key] = allocationState{}
			return allocationState{}
		}

		selected := candidates[0]
		switch term.allocation {
		case semanticArgumentAllocationGreedy:
			for _, candidate := range candidates[1:] {
				if candidate.count > selected.count {
					selected = candidate
				}
			}
		case semanticArgumentAllocationMinimal:
			for _, candidate := range candidates[1:] {
				if candidate.count < selected.count {
					selected = candidate
				}
			}
		default:
			if len(candidates) > 1 {
				state := prependAllocation(term, indices[tokenIndex:tokenIndex+selected.count], selected.state)
				state.ambiguous = true
				memo[key] = state
				return state
			}
		}

		state := prependAllocation(term, indices[tokenIndex:tokenIndex+selected.count], selected.state)
		memo[key] = state
		return state
	}

	state := visit(0, 0)
	return state.allocation, state.matched, state.ambiguous
}

func prependAllocation(term argumentTermGrammar, indices []int, tail allocationState) allocationState {
	allocation := make([]termAllocation, 1, 1+len(tail.allocation))
	allocation[0] = termAllocation{
		Term:    term.id,
		Indices: append([]int(nil), indices...),
	}
	allocation = append(allocation, tail.allocation...)
	state := allocationState{
		matched:    true,
		ambiguous:  tail.ambiguous,
		allocation: allocation,
	}

	return state
}

func claimArgumentAllocation(result *grammarResult, model grammarView, allocation argumentAllocation) {
	if !allocation.Form.present() {
		return
	}
	var form argumentFormGrammar
	for _, candidate := range model.ArgumentForms(result.Selection.Selected) {
		if candidate.id == allocation.Form {
			form = candidate
			break
		}
	}
	termByID := make(map[ArgumentTermID]argumentTermGrammar)
	for _, term := range form.Terms() {
		termByID[term.id] = term
	}
	for _, termAllocation := range allocation.Terms {
		term := termByID[termAllocation.Term]
		role := tokenRolePositional
		switch term.kind {
		case ArgumentTermLiteral:
			role = tokenRoleLiteral
			if term.endOfOptions {
				role = tokenRoleEndOfOptions
			}
		case ArgumentTermTokens:
			role = tokenRoleTokenLanguage
		case ArgumentTermRemainder:
			role = tokenRoleRemainder
		case ArgumentTermPassThrough:
			role = tokenRolePassThrough
		case ArgumentTermUnknown, ArgumentTermSequence, ArgumentTermValue:
		}
		destination, _ := term.Destination()
		for element, index := range termAllocation.Indices {
			ownership := tokenOwnership{
				Role:         role,
				Command:      result.Selection.Selected,
				ArgumentForm: allocation.Form,
				ArgumentTerm: term.id,
				Destination:  destination,
				Element:      element,
			}
			claimToken(result, index, ownership)
		}
	}
}
