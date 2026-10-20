package cli

import (
	"context"
	"errors"
	"slices"
)

const completionHeapBranchingFactor = 2

var errInvalidCompletionContribution = errors.New("cli: invalid completion contribution")

func (operation *completionOperation) completionCallbackContext(ctx context.Context, model *effectiveModel, reference Reference, prefix grammarPrefixResult, selection CommandSelection, dependencies []dependencyRequirement) (context.Context, valueAnalysis, bool) {
	snapshot := operation
	ctx = withInvocationObservationContext(ctx, reference.Locale, selection, true)
	values := analyzeValues(ctx, snapshot.invocation, prefix.Grammar)
	if values.Unsupported || completionValueIssuesBlockCallback(values.Issues, dependencies) {
		return ctx, values, false
	}
	states := make(map[uint32]ValueState, len(dependencies))
	for _, dependency := range dependencies {
		resolution, ok := values.Destinations[dependency.Destination()]
		if !ok {
			if dependency.Optional() {
				continue
			}
			return ctx, values, false
		}
		if !dependency.Optional() && resolution.State != destinationStateResolved {
			return ctx, values, false
		}
		state, ok := prepareValueState(model, values, dependency.Destination())
		if ok {
			states[dependency.Destination().uint32()] = state
		}
	}
	return contextWithValueStates(ctx, states), values, true
}

func completionDependencyFailure(operation *invocation, model *effectiveModel, values valueAnalysis, dependencies []dependencyRequirement) error {
	diagnostics := runValueDiagnostics(model, values)
	if len(diagnostics) != 0 {
		return valueDiagnosticFailure(operation, model, values, diagnostics, false)
	}

	blocked := blockedDependencyIDs(dependencies, values)
	if len(blocked) == 0 {
		return nil
	}
	return blockedDependencyFailure(operation, model, values, blocked)
}

func completionValueIssuesBlockCallback(issues []valueIssue, dependencies []dependencyRequirement) bool {
	for _, issue := range issues {
		matched := false
		for _, dependency := range dependencies {
			if dependency.Destination() != issue.Destination {
				continue
			}
			matched = true
			if !dependency.Optional() {
				return true
			}
		}
		if !matched {
			return true
		}
	}
	return false
}

func applyValueCompletionContribution(result Completion, snapshot *completionOperation, contribution CompletionContribution, declaration completionDeclaration) (Completion, error) {
	staged := stageCompletion(result)
	if contribution.Replace != nil {
		current := snapshot.invocation.Args()[snapshot.input.analysisCursorArgument]
		if contribution.Replace.Start < 0 || contribution.Replace.End < contribution.Replace.Start ||
			contribution.Replace.End > len(current) {
			return result, errInvalidCompletionContribution
		}
		staged.Replace.Start = contribution.Replace.Start
		staged.Replace.End = contribution.Replace.End
	}
	if contribution.Spacing > CompletionSpacingNoSpace {
		return result, errInvalidCompletionContribution
	}

	groups, groupIDs, ok := normalizeCompletionGroups(contribution.Groups, snapshot.input.client.Capabilities.Groups)
	if !ok {
		return result, errInvalidCompletionContribution
	}
	ordinary := make([]CompletionItem, 0, len(contribution.Candidates))
	fallback := make([]CompletionItem, 0, len(contribution.Candidates))
	matcher := prepareCompletionMatcher(snapshot.input, completionMatchQuery(snapshot))
	scratch := completionMatchScratch{}
	for _, candidate := range contribution.Candidates {
		match, matched := matcher.matchCandidate(candidate, &scratch)
		if !matched {
			continue
		}
		item, ok := normalizeValueCompletionCandidate(candidate, snapshot.input.client, groupIDs)
		if !ok {
			return result, errInvalidCompletionContribution
		}
		item.TargetIndexes = []int{0}
		item.Match = match
		if item.FallbackOnly {
			fallback = append(fallback, item)
		} else {
			ordinary = append(ordinary, item)
		}
	}
	selected := ordinary
	if len(selected) == 0 && len(staged.Items) == 0 {
		selected = fallback
	}
	staged.Items = append(staged.Items, selected...)
	staged.Groups = groups
	staged.Spacing = contribution.Spacing
	staged.Truncated = result.Truncated || contribution.Truncated
	if snapshot.input.client.Capabilities.Messages {
		messages, ok := normalizeCompletionMessages(contribution.Messages)
		if !ok {
			return result, errInvalidCompletionContribution
		}
		staged.Messages = messages
	}
	if contribution.RefreshOnInput && snapshot.input.client.Capabilities.RefreshOnInput {
		staged.RefreshOnInput = true
	}

	declaredKind, declaredMode, hasDeclaredNative := declaration.Native()
	if contribution.Native != nil && hasDeclaredNative {
		// Native intent has one semantic home. A callback contribution cannot
		// add a second Native request to a declaration that already owns one.
		return result, errInvalidCompletionContribution
	}
	if contribution.Native != nil {
		kind := contribution.Native.Kind
		mode := contribution.Native.Mode
		if !kind.valid() || !mode.valid() {
			return result, errInvalidCompletionContribution
		}
		if completionNativeSupported(snapshot.input.client.Capabilities.Native, kind) &&
			nativeCompletionTierActive(mode, len(staged.Items)+len(ordinary), len(fallback)) {
			request := CompletionNativeRequest{
				Kind:        contribution.Native.Kind,
				Mode:        contribution.Native.Mode,
				TargetIndex: 0,
			}
			staged.NativeRequests = append(staged.NativeRequests, request)
		}
	} else if hasDeclaredNative && completionNativeSupported(snapshot.input.client.Capabilities.Native, declaredKind) && nativeCompletionTierActive(declaredMode, len(staged.Items)+len(ordinary), len(fallback)) {
		request2 := CompletionNativeRequest{
			Kind:        declaredKind,
			Mode:        declaredMode,
			TargetIndex: 0,
		}
		staged.NativeRequests = append(staged.NativeRequests, request2)
	}

	return staged, nil
}

func stageCompletion(result Completion) Completion {
	staged := result
	staged.Items = append([]CompletionItem(nil), result.Items...)
	staged.Groups = append([]CompletionGroup(nil), result.Groups...)
	staged.Messages = append([]CompletionMessage(nil), result.Messages...)
	staged.NativeRequests = append([]CompletionNativeRequest(nil), result.NativeRequests...)
	return staged
}

func finalizeCompletionItems(result *Completion, input completionInput) {
	if result == nil {
		return
	}
	if input.client.Ranking == CompletionRankingModeRelevance {
		result.RankingAlgorithm = CompletionRankingAlgorithmRelevance
	}
	limit := input.limit
	if input.client.Ranking == CompletionRankingModeSource ||
		input.client.Ranking == CompletionRankingModeUnranked {
		if limit > 0 && len(result.Items) > limit {
			items := make([]CompletionItem, limit)
			copy(items, result.Items[:limit])
			result.Items = items
			result.Truncated = true
		}
		return
	}
	if limit <= 0 || limit >= len(result.Items) {
		applyStaticCompletionRanking(result.Items, input.client.Ranking)
		return
	}
	result.Items = stableCompletionTopK(result.Items, input.client.Ranking, limit)
	result.Truncated = true
}

func stableCompletionTopK(items []CompletionItem, mode CompletionRankingMode, limit int) []CompletionItem {
	if limit <= 0 || limit >= len(items) {
		result := append([]CompletionItem(nil), items...)
		applyStaticCompletionRanking(result, mode)
		return result
	}
	heap := make([]int, 0, limit)
	for index := range items {
		if len(heap) < limit {
			heap = append(heap, index)
			for child := len(heap) - 1; child > 0; {
				parent := (child - 1) / completionHeapBranchingFactor
				if !completionItemIndexLess(items, mode, heap[parent], heap[child]) {
					break
				}
				heap[parent], heap[child] = heap[child], heap[parent]
				child = parent
			}
			continue
		}
		if !completionItemIndexLess(items, mode, index, heap[0]) {
			continue
		}
		heap[0] = index
		completionTopKSiftDown(items, mode, heap, 0)
	}
	slices.SortFunc(heap, func(left, right int) int { return completionItemIndexCompare(items, mode, left, right) })
	result := make([]CompletionItem, limit)
	for i, index := range heap {
		result[i] = items[index]
	}
	return result
}

func completionTopKSiftDown(items []CompletionItem, mode CompletionRankingMode, heap []int, parent int) {
	for {
		left := parent*completionHeapBranchingFactor + 1
		if left >= len(heap) {
			return
		}
		worst := left
		right := left + 1
		if right < len(heap) && completionItemIndexLess(items, mode, heap[left], heap[right]) {
			worst = right
		}
		if !completionItemIndexLess(items, mode, heap[parent], heap[worst]) {
			return
		}
		heap[parent], heap[worst] = heap[worst], heap[parent]
		parent = worst
	}
}

func applyResponseFileCompletionContribution(result Completion, snapshot *completionOperation, contribution CompletionContribution, path string) (Completion, error) {
	staged := stageCompletion(result)
	if contribution.Replace != nil {
		if contribution.Replace.Start < 0 || contribution.Replace.End < contribution.Replace.Start ||
			contribution.Replace.End > len(path) {
			return result, errInvalidCompletionContribution
		}
		staged.Replace.Start = contribution.Replace.Start + 1
		staged.Replace.End = contribution.Replace.End + 1
	}
	if contribution.Spacing > CompletionSpacingNoSpace {
		return result, errInvalidCompletionContribution
	}

	groups, groupIDs, ok := normalizeCompletionGroups(contribution.Groups, snapshot.input.client.Capabilities.Groups)
	if !ok {
		return result, errInvalidCompletionContribution
	}
	ordinary := make([]CompletionItem, 0, len(contribution.Candidates))
	fallback := make([]CompletionItem, 0, len(contribution.Candidates))
	matcher := prepareCompletionMatcher(snapshot.input, path)
	scratch := completionMatchScratch{}
	for _, candidate := range contribution.Candidates {
		match, matched := matcher.matchCandidate(candidate, &scratch)
		if !matched {
			continue
		}
		item, ok := normalizeValueCompletionCandidate(candidate, snapshot.input.client, groupIDs)
		if !ok {
			return result, errInvalidCompletionContribution
		}
		item.TargetIndexes = []int{0}
		item.Match = match
		if item.FallbackOnly {
			fallback = append(fallback, item)
		} else {
			ordinary = append(ordinary, item)
		}
	}
	selected := ordinary
	if len(selected) == 0 {
		selected = fallback
	}
	staged.Items = append(staged.Items, selected...)
	staged.Groups = groups
	staged.Spacing = contribution.Spacing
	staged.Truncated = result.Truncated || contribution.Truncated
	if snapshot.input.client.Capabilities.Messages {
		messages, ok := normalizeCompletionMessages(contribution.Messages)
		if !ok {
			return result, errInvalidCompletionContribution
		}
		staged.Messages = messages
	}
	if contribution.RefreshOnInput && snapshot.input.client.Capabilities.RefreshOnInput {
		staged.RefreshOnInput = true
	}
	if contribution.Native != nil {
		kind := contribution.Native.Kind
		mode := contribution.Native.Mode
		if !kind.valid() || !mode.valid() {
			return result, errInvalidCompletionContribution
		}
		if completionNativeSupported(snapshot.input.client.Capabilities.Native, kind) &&
			nativeCompletionTierActive(mode, len(ordinary), len(fallback)) {
			request := CompletionNativeRequest{
				Kind:        contribution.Native.Kind,
				Mode:        contribution.Native.Mode,
				TargetIndex: 0,
			}
			staged.NativeRequests = append(staged.NativeRequests, request)
		}
	}

	return staged, nil
}

func nativeCompletionTierActive(mode CompletionNativeMode, ordinary, fallback int) bool {
	switch mode {
	case CompletionNativeAlways:
		return true
	case CompletionNativeOnNoMatch:
		return ordinary == 0 && fallback == 0
	default:
		return false
	}
}

func normalizeCompletionGroups(groups []CompletionGroup, expose bool) ([]CompletionGroup, map[string]struct{}, bool) {
	ids := make(map[string]struct{}, len(groups))
	result := make([]CompletionGroup, 0, len(groups))
	for _, group := range groups {
		if group.ID == "" {
			return nil, nil, false
		}
		if _, duplicate := ids[group.ID]; duplicate {
			return nil, nil, false
		}
		ids[group.ID] = struct{}{}
		if group.Sensitive {
			group.Label = ""
		}
		if expose {
			result = append(result, group)
		}
	}
	return result, ids, true
}

func normalizeValueCompletionCandidate(candidate CompletionCandidate, client CompletionClient, groups map[string]struct{}) (CompletionItem, bool) {
	if !validCompletionCandidateKind(candidate.Kind) || candidate.Spacing > CompletionSpacingNoSpace {
		return CompletionItem{}, false
	}
	if candidate.GroupID != "" {
		if _, ok := groups[candidate.GroupID]; !ok {
			return CompletionItem{}, false
		}
		if !client.Capabilities.Groups {
			candidate.GroupID = ""
		}
	}
	candidate.Replacement = append([]string(nil), candidate.Replacement...)
	candidate.SearchTerms = append([]string(nil), candidate.SearchTerms...)
	candidate.CommitCharacters = append([]rune(nil), candidate.CommitCharacters...)
	if candidate.Sensitive {
		candidate.Value = ""
		candidate.Replacement = nil
		candidate.DisplayText = ""
		candidate.Description = ""
		candidate.Detail = ""
		candidate.SearchTerms = nil
		candidate.SortText = ""
		candidate.CommitCharacters = nil
	}
	if !client.Capabilities.Descriptions {
		candidate.Description = ""
	}
	if !client.Capabilities.Details {
		candidate.Detail = ""
	}
	if !client.Capabilities.CandidateKinds {
		candidate.Kind = CompletionCandidateUnknown
	}
	if !client.Capabilities.CommitCharacters {
		candidate.CommitCharacters = nil
	}
	return CompletionItem{
		CompletionCandidate: candidate,
	}, true
}

func validCompletionCandidateKind(kind CompletionCandidateKind) bool {
	switch kind {
	case CompletionCandidateUnknown,
		CompletionCandidateValue,
		CompletionCandidateCommand,
		CompletionCandidateCommandAlias,
		CompletionCandidateCommandShortcut,
		CompletionCandidateAction,
		CompletionCandidateFlag,
		CompletionCandidateKeyword,
		CompletionCandidateFile,
		CompletionCandidateDirectory,
		CompletionCandidateExecutable,
		CompletionCandidateResource,
		CompletionCandidateNamespace,
		CompletionCandidateUser,
		CompletionCandidateHost,
		CompletionCandidateEnvironmentVariable,
		CompletionCandidateURL:
		return true
	default:
		return false
	}
}

func normalizeCompletionMessages(messages []CompletionMessage) ([]CompletionMessage, bool) {
	result := make([]CompletionMessage, len(messages))
	for i, message := range messages {
		if message.Kind < CompletionMessageInformation || message.Kind > CompletionMessageError {
			return nil, false
		}
		if message.Sensitive {
			message.Text = ""
		}
		result[i] = message
	}
	return result, true
}
