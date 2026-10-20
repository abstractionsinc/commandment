package cli

import (
	"cmp"
	"context"
	"errors"
	"slices"
)

// completeArgumentCursor resolves every viable positional target independently.
// The incomplete cursor contribution never enters value analysis. Each target
// is evaluated into an isolated local result and only then merged, preserving
// target-local fallback/atomicity while retaining valid peer targets on failure.
func (operation *completionOperation) completeArgumentCursor(ctx context.Context, model *effectiveModel, reference Reference, prefix grammarPrefixResult, result Completion, analysis argumentCompletionAnalysis) (Completion, error) {
	snapshot := operation
	if !analysis.Settled {
		result.Partial = true
		return result, nil
	}
	if analysis.Ambiguous {
		result.Partial = true
		return result, operationDiagnosticFailure(snapshot.invocation,
			DiagnosticAmbiguousCompletionTarget,
			"ambiguous completion target",
			nil,
		)
	}
	if len(analysis.Targets) == 0 {
		return result, nil
	}

	type targetWork struct {
		targetIndex int
		projection  projectedArgumentCompletionTarget
		prefix      grammarPrefixResult
	}

	work := make([]targetWork, 0, len(analysis.Targets))
	for _, semanticTarget := range analysis.Targets {
		projection, ok := projectArgumentCompletionTarget(model, reference, prefix.Current, semanticTarget)
		if !ok {
			continue
		}
		targetIndex := len(result.Targets)
		result.Targets = append(result.Targets, projection.public)
		targetPrefix := prefix
		if semanticTarget.Allocation.Form.present() {
			targetPrefix.Grammar.Arguments = semanticTarget.Allocation
		}
		work = append(work, targetWork{
			targetIndex: targetIndex,
			projection:  projection,
			prefix:      targetPrefix,
		})
	}
	slices.SortStableFunc(work, func(left, right targetWork) int {
		return cmp.Or(
			cmp.Compare(left.projection.term.meta.position, right.projection.term.meta.position),
			cmp.Compare(left.targetIndex, right.targetIndex),
		)
	})

	baseReplace := result.Replace
	var failures []error
	for _, targetWork := range work {
		semanticTarget := targetWork.projection.semantic
		projection := targetWork.projection
		targetIndex := targetWork.targetIndex

		local := Completion{
			Locale:    result.Locale,
			Targets:   []CompletionTarget{projection.public},
			Selection: cloneCommandSelection(result.Selection),
			Replace:   result.Replace,
		}
		targetPrefix := targetWork.prefix
		var targetErr error
		switch semanticTarget.Kind {
		case ArgumentTermLiteral:
			local = completeLiteralArgumentTarget(local, snapshot, projection)
		default:
			local, targetErr = operation.completeSemanticArgumentTarget(
				ctx,
				model,
				reference,
				targetPrefix,
				local,
				projection,
				len(analysis.Targets) == 1,
			)
		}
		if targetErr != nil {
			result.Partial = true
			result.Diagnostics = append(result.Diagnostics, local.Diagnostics...)
			result.DiagnosticsTruncated = result.DiagnosticsTruncated || local.DiagnosticsTruncated
			failures = append(failures, targetErr)
			continue
		}
		if local.Delegation != nil {
			// Delegation is legal only after one authoritative transfer owner and
			// is exclusive with every semantic/native result in this operation.
			if len(analysis.Targets) != 1 || len(result.Items) != 0 || len(result.NativeRequests) != 0 ||
				len(result.Targets) != 1 {
				result.Partial = true
				failures = append(
					failures,
					operationDiagnosticFailure(snapshot.invocation,
						DiagnosticUnsupportedCompletionDelegation,
						"completion delegation is not exclusive",
						nil,
					),
				)
				continue
			}
			result.Delegation = cloneCompletionDelegation(local.Delegation)
			return result, errors.Join(failures...)
		}
		var mergeErr error
		result, mergeErr = mergeArgumentTargetResult(result, local, targetIndex, baseReplace)
		if mergeErr != nil {
			result.Partial = true
			failures = append(
				failures,
				operationDiagnosticFailure(snapshot.invocation,
					DiagnosticInvalidCompletionContribution,
					"incompatible completion target contribution",
					mergeErr,
				),
			)
		}
	}
	return result, errors.Join(failures...)
}

type projectedArgumentCompletionTarget struct {
	semantic   argumentCompletionTarget
	public     CompletionTarget
	term       argumentTermRecord
	form       argumentFormRecord
	publicTerm ArgumentTermReference
}

func projectArgumentCompletionTarget(model *effectiveModel, reference Reference, command CommandID, target argumentCompletionTarget) (projectedArgumentCompletionTarget, bool) {
	term, ok := model.ArgumentTerm(target.Term)
	if !ok {
		return projectedArgumentCompletionTarget{}, false
	}
	form, ok := model.ArgumentForm(target.Form)
	if !ok || form.command != command {
		return projectedArgumentCompletionTarget{}, false
	}
	publicTerm, ok := reference.ArgumentTerms[target.Term]
	if !ok || !publicTerm.Available {
		return projectedArgumentCompletionTarget{}, false
	}
	publicForm, ok := reference.ArgumentForms[target.Form]
	if !ok || !publicForm.Available {
		return projectedArgumentCompletionTarget{}, false
	}
	publicCommand, ok := reference.Commands[command]
	if !ok {
		return projectedArgumentCompletionTarget{}, false
	}
	kind := CompletionTargetArgument
	switch target.Kind {
	case ArgumentTermLiteral:
		kind = CompletionTargetLiteral
		if publicTerm.EndOfOptions {
			kind = CompletionTargetEndOfOptions
		}
	case ArgumentTermTokens:
		kind = CompletionTargetTokenLanguage
	case ArgumentTermRemainder:
		kind = CompletionTargetRemainder
	case ArgumentTermPassThrough:
		kind = CompletionTargetPassThrough
	case ArgumentTermValue:
	default:
		return projectedArgumentCompletionTarget{}, false
	}
	result := projectedArgumentCompletionTarget{
		semantic: target,
		public: CompletionTarget{
			Kind:           kind,
			Command:        publicCommand.ID,
			CommandPath:    append([]string(nil), publicCommand.Path...),
			ValueID:        publicTerm.Destination,
			ArgumentFormID: publicForm.ID,
			TermIndex:      target.TermIndex,
			Name:           publicTerm.Name,
			Spelling:       publicTerm.Literal,
			Occurrence:     target.Element,
		},
		term:       term,
		form:       form,
		publicTerm: publicTerm,
	}
	return result, true
}

func completeLiteralArgumentTarget(local Completion, snapshot *completionOperation, projection projectedArgumentCompletionTarget) Completion {
	term := projection.publicTerm
	if term.Literal == "" {
		return local
	}
	candidate := CompletionCandidate{
		Value:          term.Literal,
		DisplayText:    term.Literal,
		Description:    term.Description,
		Kind:           CompletionCandidateKeyword,
		LifecycleState: term.LifecycleState,
	}
	matcher := prepareCompletionMatcher(snapshot.input, completionMatchQuery(snapshot))
	scratch := completionMatchScratch{}
	match, ok := matcher.matchCandidate(candidate, &scratch)
	if !ok {
		return local
	}
	item, ok := normalizeValueCompletionCandidate(candidate, snapshot.input.client, nil)
	if !ok {
		return local
	}
	item.Declaration = DeclarationIdentity{
		Kind:         DeclarationKindArgumentTerm,
		ArgumentTerm: term.ID,
	}
	item.TargetIndexes = []int{0}
	item.Match = match
	local.Items = append(local.Items, item)
	return local
}

func (operation *completionOperation) completeSemanticArgumentTarget(ctx context.Context, model *effectiveModel, reference Reference, prefix grammarPrefixResult, local Completion, projection projectedArgumentCompletionTarget, authoritativeSingleTarget bool) (Completion, error) {
	snapshot := operation
	invocation := operation.invocation
	target := projection.semantic
	publicTarget := projection.public
	term := projection.term
	form := projection.form
	completion, hasCompletion := term.Completion()
	needsSupplies := false
	if target.Kind == ArgumentTermValue {
		needsSupplies = completionDomainsNeedInvocationSupplies(model, target.Destination)
	}
	if hasCompletion {
		switch completion.Kind() {
		case completionDeclarationKindValue,
			completionDeclarationKindTokenLanguage,
			completionDeclarationKindDelegation:
			needsSupplies = true
		case completionDeclarationKindUnknown, completionDeclarationKindNative:
		}
	}
	if needsSupplies {
		if code, err := invocation.finalizeSupplies(ctx, prefix.Grammar.Selection); err != nil {
			return local, operationDiagnosticFailure(
				snapshot.invocation,
				code,
				"completion invocation value supply failed",
				err,
			)
		}
	}

	if target.Kind == ArgumentTermValue {
		effectiveModel, domainReference, outcomes, err := snapshot.settleCompletionTargetDomains(ctx, prefix.Grammar, target.Destination)
		if err != nil {
			diagnostics, _ := dynamicDomainMaterializationDiagnostics(outcomes)
			local.Diagnostics = append(local.Diagnostics, diagnostics...)
			local.Partial = true
			return local, err
		}
		if domainReference.Commands != nil {
			model = effectiveModel
			reference = domainReference
			local.Locale = reference.Locale
			local = appendCompletionDomainCandidates(local, snapshot, reference, target.Destination)
		}
	}
	if !hasCompletion {
		return local, nil
	}

	if completion.Kind() == completionDeclarationKindNative {
		nativeKind, nativeMode, ok := completion.Native()
		if ok && completionNativeSupported(snapshot.input.client.Capabilities.Native, nativeKind) &&
			nativeCompletionTierActive(nativeMode, len(local.Items), 0) {
			request4 := CompletionNativeRequest{
				Kind:        nativeKind,
				Mode:        nativeMode,
				TargetIndex: 0,
			}
			local.NativeRequests = append(local.NativeRequests, request4)
		}
		return local, nil
	}

	callback, _ := snapshot.invocation.argumentCompletionCallback(term.meta.id)
	if !callback.present() {
		panic("cli: accepted argument completion has no callback")
	}

	callbackCtx, callbackValues, ok := operation.completionCallbackContext(
		ctx,
		model,
		reference,
		prefix,
		local.Selection,
		completion.Dependencies(),
	)
	if !ok {
		local.Partial = true
		return local, completionDependencyFailure(snapshot.invocation, model, callbackValues, completion.Dependencies())
	}
	configuration, ok := snapshot.invocation.analysisConfiguration(form.command, callbackValues.Destinations)
	if !ok {
		panic("cli: accepted completion callback has no analysis configuration")
	}
	currentText := snapshot.invocation.Args()[snapshot.input.analysisCursorArgument]
	currentReplace := CompletionRange{
		Start: 0,
		End:   len(currentText),
	}

	switch completion.Kind() {
	case completionDeclarationKindValue:
		request := ValueCompletionRequest{
			Target:           publicTarget,
			Selection:        cloneCommandSelection(local.Selection),
			Text:             currentText,
			Cursor:           snapshot.input.cursor.Byte,
			Replace:          currentReplace,
			WorkingDirectory: snapshot.invocation.input.workingDirectory,
			Trigger:          snapshot.input.trigger,
			Limit:            snapshot.input.limit,
		}
		contribution, err := callback.value(callbackCtx, configuration, request)
		if err != nil {
			return local, operationDiagnosticFailure(
				snapshot.invocation,
				DiagnosticCompletionFailure,
				"completion callback failed",
				err,
			)
		}
		applied, err := applyValueCompletionContribution(local, snapshot, contribution, completion)
		if err != nil {
			return local, operationDiagnosticFailure(snapshot.invocation,
				DiagnosticInvalidCompletionContribution,
				"invalid completion contribution",
				err,
			)
		}
		return applied, nil

	case completionDeclarationKindTokenLanguage:
		tokens := make([]string, 0, len(target.PriorIndices))
		args := snapshot.invocation.Args()
		for _, index := range target.PriorIndices {
			if index < 0 || index >= len(args) {
				panic("cli: accepted token-language completion target index is out of range")
			}
			tokens = append(tokens, args[index])
		}
		request2 := TokenLanguageCompletionRequest{
			Target:           publicTarget,
			Selection:        cloneCommandSelection(local.Selection),
			Tokens:           tokens,
			Text:             currentText,
			Cursor:           snapshot.input.cursor.Byte,
			Replace:          currentReplace,
			WorkingDirectory: snapshot.invocation.input.workingDirectory,
			Trigger:          snapshot.input.trigger,
			Limit:            snapshot.input.limit,
		}
		contribution, err := callback.tokenLanguage(callbackCtx, configuration, request2)
		if err != nil {
			return local, operationDiagnosticFailure(
				snapshot.invocation,
				DiagnosticCompletionFailure,
				"completion callback failed",
				err,
			)
		}
		applied, err := applyValueCompletionContribution(local, snapshot, contribution, completion)
		if err != nil {
			return local, operationDiagnosticFailure(snapshot.invocation,
				DiagnosticInvalidCompletionContribution,
				"invalid completion contribution",
				err,
			)
		}
		return applied, nil

	case completionDeclarationKindDelegation:
		if !authoritativeSingleTarget ||
			(target.Kind != ArgumentTermRemainder && target.Kind != ArgumentTermPassThrough) {
			return local, operationDiagnosticFailure(snapshot.invocation,
				DiagnosticUnsupportedCompletionDelegation,
				"completion delegation lacks one authoritative transfer owner",
				nil,
			)
		}
		args := make([]string, 0, len(target.PriorIndices)+1)
		operationArgs := snapshot.invocation.Args()
		for _, index := range target.PriorIndices {
			if index < 0 || index >= len(operationArgs) {
				panic("cli: accepted delegation completion target index is out of range")
			}
			args = append(args, operationArgs[index])
		}
		args = append(args, currentText)
		request3 := CompletionDelegationRequest{
			Target:    publicTarget,
			Selection: cloneCommandSelection(local.Selection),
			Args:      append([]string(nil), args...),
			Cursor: CompletionCursor{
				Argument: len(args) - 1,
				Byte:     snapshot.input.cursor.Byte,
			},
		}
		delegation, err := callback.delegation(callbackCtx, configuration, request3)
		if err != nil {
			return local, operationDiagnosticFailure(
				snapshot.invocation,
				DiagnosticCompletionFailure,
				"completion delegation failed",
				err,
			)
		}
		if !validCompletionDelegation(delegation) {
			return local, operationDiagnosticFailure(snapshot.invocation,
				DiagnosticInvalidCompletionContribution,
				"invalid completion delegation",
				errInvalidCompletionContribution,
			)
		}
		local.Delegation = cloneCompletionDelegation(&delegation)
		return local, nil
	default:
		panic("cli: accepted completion declaration has invalid kind")
	}
}

func mergeArgumentTargetResult(result, local Completion, targetIndex int, baseReplace CompletionSpan) (Completion, error) {
	staged := stageCompletion(result)
	if local.Replace != baseReplace {
		if result.Replace != baseReplace && result.Replace != local.Replace {
			return result, errInvalidCompletionContribution
		}
		staged.Replace = local.Replace
	}
	for _, item := range local.Items {
		item.TargetIndexes = []int{targetIndex}
		staged.Items = append(staged.Items, item)
	}
	for _, request := range local.NativeRequests {
		request.TargetIndex = targetIndex
		staged.NativeRequests = append(staged.NativeRequests, request)
	}
	for _, group := range local.Groups {
		found := false
		for _, existing := range staged.Groups {
			if existing.ID != group.ID {
				continue
			}
			found = true
			if existing != group {
				return result, errInvalidCompletionContribution
			}
			break
		}
		if !found {
			staged.Groups = append(staged.Groups, group)
		}
	}
	staged.Messages = append(staged.Messages, local.Messages...)
	if local.Spacing != CompletionSpacingDefault {
		if staged.Spacing != CompletionSpacingDefault && staged.Spacing != local.Spacing {
			return result, errInvalidCompletionContribution
		}
		staged.Spacing = local.Spacing
	}
	staged.Truncated = staged.Truncated || local.Truncated
	staged.RefreshOnInput = staged.RefreshOnInput || local.RefreshOnInput
	staged.Diagnostics = append(staged.Diagnostics, local.Diagnostics...)
	staged.DiagnosticsTruncated = staged.DiagnosticsTruncated || local.DiagnosticsTruncated
	return staged, nil
}

func validCompletionDelegation(delegation CompletionDelegation) bool {
	if delegation.Cursor.Argument < 0 || delegation.Cursor.Argument >= len(delegation.Args) {
		return false
	}
	return delegation.Cursor.Byte >= 0 && delegation.Cursor.Byte <= len(delegation.Args[delegation.Cursor.Argument])
}

func cloneCompletionDelegation(delegation *CompletionDelegation) *CompletionDelegation {
	if delegation == nil {
		return nil
	}
	clone := *delegation
	clone.Args = append([]string(nil), delegation.Args...)
	return &clone
}

func (operation *completionOperation) settleCompletionTargetDomains(ctx context.Context, grammar grammarResult, destination ValueID) (*effectiveModel, Reference, map[int]dynamicDomainOutcome, error) {
	if operation == nil || operation.invocation == nil || operation.invocation.analysis == nil {
		panic("cli: invalid completion target domain demand")
	}

	model := operation.invocation.analysis.effectiveModel
	if len(dynamicDomainIndexesForDestination(model, destination)) == 0 {
		return model, Reference{}, nil, nil
	}

	outcomes := operation.invocation.materializeDynamicDomainsForDestination(ctx, grammar, destination)
	if len(outcomes) != 0 {
		diagnostics, cause := dynamicDomainMaterializationDiagnostics(outcomes)
		if len(diagnostics) != 0 || cause != nil {
			return operation.invocation.analysis.effectiveModel, Reference{}, outcomes, dynamicDomainDiagnosticFailure(
				operation.invocation,
				outcomes,
				cause,
			)
		}
	}

	model = operation.invocation.analysis.effectiveModel
	visibility := ReferenceVisibilityPublic
	if operation.input.visibility == CompletionVisibilityAllInvocable {
		visibility = ReferenceVisibilityAll
	}
	referenceOptionsValue := ReferenceOptions{
		Locale:     operation.invocation.input.locale,
		Visibility: visibility,
		Order:      ReferenceOrderDeclaration,
	}
	reference, err := projectOperationReference(operation.invocation, referenceOptionsValue)
	if err != nil {
		return model, Reference{}, outcomes, err
	}

	return model, reference, outcomes, nil
}

func completionDomainsNeedInvocationSupplies(model *effectiveModel, destination ValueID) bool {
	domains := model.ValueDomains()
	for _, index := range dynamicDomainIndexesForDestination(model, destination) {
		if index < 0 || index >= len(domains) {
			continue
		}
		if _, dynamic := domains[index].DynamicDefinition(); dynamic {
			return true
		}
	}
	return false
}

func appendCompletionDomainCandidates(result Completion, snapshot *completionOperation, reference Reference, destination ValueID) Completion {
	matcher := prepareCompletionMatcher(snapshot.input, completionMatchQuery(snapshot))
	scratch := completionMatchScratch{}
	for _, domain := range reference.ValueDomains {
		used := false
		for _, use := range domain.Uses {
			if use.Destination == destination {
				used = true
				break
			}
		}
		if !used {
			continue
		}
		for _, entryID := range domain.Entries {
			entry, ok := reference.ValueDomainEntries[entryID]
			if !ok || !entry.Available || !entry.Visible {
				continue
			}
			candidate := CompletionCandidate{
				Value:          entry.Name,
				DisplayText:    entry.Name,
				Description:    entry.Description.Short,
				Kind:           CompletionCandidateValue,
				LifecycleState: entry.LifecycleState,
			}
			match, ok := matcher.matchCandidate(candidate, &scratch)
			if !ok {
				continue
			}
			item, ok := normalizeValueCompletionCandidate(candidate, snapshot.input.client, nil)
			if !ok {
				continue
			}
			item.Declaration = DeclarationIdentity{
				Kind:             DeclarationKindValueDomainEntry,
				ValueDomainEntry: entry.ID,
			}
			item.TargetIndexes = []int{0}
			item.Match = match
			result.Items = append(result.Items, item)
		}
	}
	return result
}
