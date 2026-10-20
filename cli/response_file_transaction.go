package cli

import (
	"errors"
	"maps"
)

func (parent *analysisState) forkResponse(operation *invocation) *analysisState {
	if parent == nil || operation == nil {
		panic("cli: invalid response analysis parent")
	}
	if operation.analysis != parent {
		panic("cli: response fork lost current analysis authority")
	}

	parent.work.freezeForResponse()
	parent.assertSettledResponseFork()

	child := &analysisState{
		ids:                cloneOperationIDsForResponse(parent.ids),
		effectiveModel:     cloneEffectiveModelForResponse(parent.effectiveModel),
		callbacks:          cloneDeclarationCallbacks(parent.callbacks),
		provenance:         cloneInvocationProvenanceForResponse(parent.provenance),
		responses:          cloneResponseFileSessionForResponse(parent.responses),
		shortcuts:          cloneShortcutStateForResponse(parent.shortcuts),
		args:               append([]string(nil), parent.args...),
		valueSupplies:      append([]valueSupply(nil), parent.valueSupplies...),
		dependencyValues:   valueAnalysis{},
		nextValueCandidate: parent.nextValueCandidate,
		supplyCode:         parent.supplyCode,
		supplyErr:          parent.supplyErr,
	}
	child.supplies = cloneInvocationSuppliesForResponse(parent.supplies)
	cloneProviderStateForResponse(&parent.providers, &child.providers)
	cloneDynamicDomainStateForResponse(&parent.domains, &child.domains)
	child.dependencyValues = cloneValueAnalysisForResponse(parent.dependencyValues, parent, child)
	cloneConversionMemoForResponse(&parent.conversions, &child.conversions)
	cloneGrammarMemoForResponse(&parent.grammar, &child.grammar)

	return child
}

func (parent *analysisState) assertSettledResponseFork() {
	if parent == nil || !parent.work.quiescent() {
		panic("cli: response fork requires settled analysis")
	}
	if len(parent.pendingDestinationAcceptances) != 0 {
		panic("cli: authoritative response parent has unpublished destination acceptance")
	}
	assertProviderStateSettledForResponse(&parent.providers)
	assertDynamicDomainStateSettledForResponse(&parent.domains)
	assertConversionMemoSettledForResponse(&parent.conversions)
}

type preparedResponseCommit struct {
	parent *analysisState
	child  *analysisState

	destinations []destinationAcceptance
}

func prepareResponseCommit(parent, child *analysisState) preparedResponseCommit {
	if parent == nil || child == nil || parent == child || !child.work.quiescent() {
		panic("cli: invalid response commit")
	}

	destinations := append([]destinationAcceptance(nil), child.pendingDestinationAcceptances...)
	child.pendingDestinationAcceptances = nil

	commit := preparedResponseCommit{
		parent:       parent,
		child:        child,
		destinations: destinations,
	}

	return commit
}

func (operation *invocation) publishResponse(commit preparedResponseCommit) {
	if operation == nil || commit.parent == nil || commit.child == nil || commit.parent == commit.child {
		panic("cli: invalid response publication")
	}

	operation.publication.mu.Lock()
	defer operation.publication.mu.Unlock()

	if operation.analysis != commit.parent {
		panic("cli: response publication lost its parent authority")
	}

	operation.analysis = commit.child
	for _, acceptance := range commit.destinations {
		acceptance.publish()
	}
}

func cloneOperationIDsForResponse(ids operationIDs) operationIDs {
	declaration := *ids.declarationIdentities.mustState()
	operation := *ids.operationIdentities.mustState()

	return operationIDs{
		declarationIdentities: declarationIdentities{
			state: &declaration,
		},
		operationIdentities: operationIdentities{
			state: &operation,
		},
	}
}

func cloneEffectiveModelForResponse(parent *effectiveModel) *effectiveModel {
	if parent == nil {
		panic("cli: nil effective model")
	}

	parent.mu.RLock()
	defer parent.mu.RUnlock()

	child := &effectiveModel{
		base:      parent.base,
		providers: append([]providerPublication(nil), parent.providers...),
		domains:   append([]dynamicDomainPublication(nil), parent.domains...),
	}
	child.resetDerivedLocked()

	return child
}

func cloneInvocationProvenanceForResponse(parent invocationProvenance) invocationProvenance {
	child := invocationProvenance{
		sources:              append([]inputSourceRecord(nil), parent.sources...),
		origins:              append([]inputOriginRecord(nil), parent.origins...),
		argumentSource:       parent.argumentSource,
		environmentSource:    parent.environmentSource,
		argumentOrigins:      append([]originRef(nil), parent.argumentOrigins...),
		environmentOrigins:   append([]originRef(nil), parent.environmentOrigins...),
		responseIncludes:     append([]responseIncludeObservation(nil), parent.responseIncludes...),
		shortcutInputEntries: make(map[InputSourceID][]string, len(parent.shortcutInputEntries)),
	}
	for index := range child.origins {
		child.origins[index].parents = append([]InputOriginID(nil), parent.origins[index].parents...)
	}
	for source, entries := range parent.shortcutInputEntries {
		child.shortcutInputEntries[source] = append([]string(nil), entries...)
	}
	if parent.shortcutInputEntries == nil {
		child.shortcutInputEntries = nil
	}

	return child
}

func cloneResponseFileSessionForResponse(parent *responseFileSession) *responseFileSession {
	if parent == nil {
		panic("cli: nil response-file session")
	}

	child := &responseFileSession{
		limits:       parent.limits,
		sourceByKey:  make(map[string]responseFileSnapshot, len(parent.sourceByKey)),
		uniqueBytes:  parent.uniqueBytes,
		tokenCount:   parent.tokenCount,
		includeCount: parent.includeCount,
	}
	for key, snapshot := range parent.sourceByKey {
		child.sourceByKey[key] = cloneResponseSnapshot(snapshot)
	}

	return child
}

func cloneShortcutStateForResponse(parent shortcutState) shortcutState {
	child := shortcutState{
		defaults: make([]shortcutDefaultContribution, len(parent.defaults)),
		enabled:  parent.enabled,
	}
	for index, contribution := range parent.defaults {
		child.defaults[index] = contribution
		child.defaults[index].Default.value = snapshotMutableValue(contribution.Default.value)
		child.defaults[index].Default.values = append([]string(nil), contribution.Default.values...)
	}
	if parent.defaults == nil {
		child.defaults = nil
	}

	return child
}

func cloneInvocationSuppliesForResponse(parent invocationSupplies) invocationSupplies {
	child := invocationSupplies{
		defaults:     append([]DefaultProvider(nil), parent.defaults...),
		facts:        append([]FactProvider(nil), parent.facts...),
		factDone:     make(map[CommandID]struct{}, len(parent.factDone)),
		defaultsDone: parent.defaultsDone,
		order:        parent.order,
	}
	maps.Copy(child.factDone, parent.factDone)
	if parent.factDone == nil {
		child.factDone = nil
	}

	return child
}

//nolint:dupl // Provider and dynamic-domain slots are distinct typed semantic owners with intentionally parallel fork rules.
func cloneProviderStateForResponse(parent, child *providerState) {
	if parent == nil || child == nil {
		panic("cli: invalid provider response-fork state")
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if parent.namespaces != nil {
		child.namespaces = make(map[CommandID]*providerNamespaceSlot, len(parent.namespaces))
	}
	for command, slot := range parent.namespaces {
		if slot == nil {
			panic("cli: nil provider response-fork slot")
		}
		slot.mu.Lock()
		switch slot.phase {
		case providerSlotUnstarted:
			child.namespaces[command] = &providerNamespaceSlot{}
		case providerSlotSettled:
			done := make(chan struct{})
			close(done)
			child.namespaces[command] = &providerNamespaceSlot{
				phase:      providerSlotSettled,
				done:       done,
				outcome:    cloneProviderNamespaceOutcome(slot.outcome),
				panicked:   slot.panicked,
				panicValue: slot.panicValue,
			}
		case providerSlotRunning:
			slot.mu.Unlock()
			panic("cli: running provider slot during response fork")
		default:
			slot.mu.Unlock()
			panic("cli: invalid provider slot during response fork")
		}
		slot.mu.Unlock()
	}
}

func assertProviderStateSettledForResponse(parent *providerState) {
	if parent == nil {
		panic("cli: nil provider response-fork state")
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()
	for _, slot := range parent.namespaces {
		if slot == nil {
			panic("cli: nil provider response-fork slot")
		}
		slot.mu.Lock()
		phase := slot.phase
		slot.mu.Unlock()
		if phase == providerSlotRunning {
			panic("cli: running provider slot during response fork")
		}
	}
}

//nolint:dupl // Provider and dynamic-domain slots are distinct typed semantic owners with intentionally parallel fork rules.
func cloneDynamicDomainStateForResponse(parent, child *dynamicDomainState) {
	if parent == nil || child == nil {
		panic("cli: invalid dynamic-domain response-fork state")
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if parent.slots != nil {
		child.slots = make(map[int]*dynamicDomainSlot, len(parent.slots))
	}
	for index, slot := range parent.slots {
		if slot == nil {
			panic("cli: nil dynamic-domain response-fork slot")
		}
		slot.mu.Lock()
		switch slot.phase {
		case dynamicDomainSlotUnstarted:
			child.slots[index] = &dynamicDomainSlot{}
		case dynamicDomainSlotSettled:
			done := make(chan struct{})
			close(done)
			child.slots[index] = &dynamicDomainSlot{
				phase:      dynamicDomainSlotSettled,
				done:       done,
				outcome:    cloneDynamicDomainOutcome(slot.outcome),
				panicked:   slot.panicked,
				panicValue: slot.panicValue,
			}
		case dynamicDomainSlotRunning:
			slot.mu.Unlock()
			panic("cli: running dynamic-domain slot during response fork")
		default:
			slot.mu.Unlock()
			panic("cli: invalid dynamic-domain slot during response fork")
		}
		slot.mu.Unlock()
	}
}

func assertDynamicDomainStateSettledForResponse(parent *dynamicDomainState) {
	if parent == nil {
		panic("cli: nil dynamic-domain response-fork state")
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()
	for _, slot := range parent.slots {
		if slot == nil {
			panic("cli: nil dynamic-domain response-fork slot")
		}
		slot.mu.Lock()
		phase := slot.phase
		slot.mu.Unlock()
		if phase == dynamicDomainSlotRunning {
			panic("cli: running dynamic-domain slot during response fork")
		}
	}
}

func cloneValueAnalysisForResponse(parent valueAnalysis, parentOwner, childOwner *analysisState) valueAnalysis {
	child := cloneValueAnalysis(parent)
	remap := func(ref valueCandidateRef) valueCandidateRef {
		if !ref.valid() || ref.owner != parentOwner {
			panic("cli: invalid response-fork candidate reference")
		}
		return valueCandidateRef{
			owner: childOwner,
			index: ref.index,
		}
	}

	remappedCandidates := make(map[valueCandidateRef]valueCandidateObservation, len(child.Candidates))
	for ref, candidate := range child.Candidates {
		newRef := remap(ref)
		candidate.Meta.ID = remap(candidate.Meta.ID)
		for index, suppressed := range candidate.Meta.SuppressedBy {
			candidate.Meta.SuppressedBy[index] = remap(suppressed)
		}
		for index, conflict := range candidate.Meta.ConflictsWith {
			candidate.Meta.ConflictsWith[index] = remap(conflict)
		}
		remappedCandidates[newRef] = candidate
	}
	child.Candidates = remappedCandidates
	for id, resolution := range child.Destinations {
		for index, ref := range resolution.Candidates {
			resolution.Candidates[index] = remap(ref)
		}
		for index, ref := range resolution.Contributors {
			resolution.Contributors[index] = remap(ref)
		}
		for index, component := range resolution.Elements {
			resolution.Elements[index].Candidate = remap(component.Candidate)
		}
		for index, component := range resolution.Entries {
			resolution.Entries[index].Candidate = remap(component.Candidate)
		}
		child.Destinations[id] = resolution
	}

	return child
}

func cloneConversionMemoForResponse(parent, child *conversionMemo) {
	if parent == nil || child == nil {
		panic("cli: invalid conversion response-fork memo")
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if parent.results != nil {
		child.results = make(map[conversionEvaluationKey]*frozenConversionEvaluation, len(parent.results))
	}
	for key, frozen := range parent.results {
		if frozen == nil || frozen.done == nil {
			panic("cli: invalid conversion memo during response fork")
		}
		select {
		case <-frozen.done:
		default:
			panic("cli: running conversion during response fork")
		}
		done := make(chan struct{})
		close(done)
		child.results[key] = &frozenConversionEvaluation{
			done:       done,
			outcome:    cloneConversionOutcomeForResponse(frozen.outcome),
			panicked:   frozen.panicked,
			panicValue: frozen.panicValue,
		}
	}
}

func assertConversionMemoSettledForResponse(parent *conversionMemo) {
	if parent == nil {
		panic("cli: nil conversion response-fork memo")
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()
	for _, frozen := range parent.results {
		if frozen == nil || frozen.done == nil {
			panic("cli: invalid conversion memo during response fork")
		}
		select {
		case <-frozen.done:
		default:
			panic("cli: running conversion during response fork")
		}
	}
}

func cloneConversionOutcomeForResponse(parent conversionOutcome) conversionOutcome {
	child := parent.Clone()
	if parent.tokenRepresentation == nil {
		return child
	}
	tokens := parent.tokenRepresentation.tokensSnapshot()
	child.tokenRepresentation = &frozenTokenRepresentation{
		tokens: append([]string(nil), tokens...),
	}
	return child
}

func cloneGrammarMemoForResponse(parent, child *grammarMemo) {
	if parent == nil || child == nil {
		panic("cli: invalid grammar response-fork memo")
	}
	parent.mu.RLock()
	defer parent.mu.RUnlock()

	if parent.commandLexicalNamespaces != nil {
		child.commandLexicalNamespaces = make(map[lexicalNamespaceKey]*commandLexicalNamespace, len(parent.commandLexicalNamespaces))
		for key, namespace := range parent.commandLexicalNamespaces {
			child.commandLexicalNamespaces[key] = cloneCommandLexicalNamespaceForResponse(namespace)
		}
	}
	if parent.flagLexicalNamespaces != nil {
		child.flagLexicalNamespaces = make(map[lexicalNamespaceKey]*flagLexicalNamespace, len(parent.flagLexicalNamespaces))
		for key, namespace := range parent.flagLexicalNamespaces {
			child.flagLexicalNamespaces[key] = cloneFlagLexicalNamespaceForResponse(namespace)
		}
	}
	if parent.flagFrontiers != nil {
		child.flagFrontiers = make(map[flagFrontierCacheKey]*flagFrontierSnapshot, len(parent.flagFrontiers))
		cloned := make(map[*flagFrontierSnapshot]*flagFrontierSnapshot, len(parent.flagFrontiers))
		for key, frontier := range parent.flagFrontiers {
			copyKey := key
			copyKey.parent = cloneFlagFrontierGraphForResponse(key.parent, cloned)
			child.flagFrontiers[copyKey] = cloneFlagFrontierGraphForResponse(frontier, cloned)
		}
	}
}

func cloneFlagFrontierGraphForResponse(frontier *flagFrontierSnapshot, cloned map[*flagFrontierSnapshot]*flagFrontierSnapshot) *flagFrontierSnapshot {
	if frontier == nil {
		return nil
	}
	if existing := cloned[frontier]; existing != nil {
		return existing
	}
	copyFrontier := cloneFlagFrontierSnapshotForResponse(frontier)
	cloned[frontier] = copyFrontier
	return copyFrontier
}

func cloneCommandLexicalNamespaceForResponse(parent *commandLexicalNamespace) *commandLexicalNamespace {
	if parent == nil {
		return nil
	}
	child := &commandLexicalNamespace{
		entries:      append([]commandEntryGrammar(nil), parent.entries...),
		actions:      append([]actionCommandGrammar(nil), parent.actions...),
		entryMembers: append([]commandLexicalMember(nil), parent.entryMembers...),
		members:      append([]commandLexicalMember(nil), parent.members...),
		exactEntries: cloneStringIntSlicesForResponse(parent.exactEntries),
		exact:        cloneStringIntSlicesForResponse(parent.exact),
	}
	return child
}

func cloneFlagLexicalNamespaceForResponse(parent *flagLexicalNamespace) *flagLexicalNamespace {
	if parent == nil {
		return nil
	}
	return &flagLexicalNamespace{
		flags:   append([]flagGrammar(nil), parent.flags...),
		actions: append([]semanticActionFlagGrammar(nil), parent.actions...),
	}
}

func cloneFlagFrontierSnapshotForResponse(parent *flagFrontierSnapshot) *flagFrontierSnapshot {
	if parent == nil {
		return nil
	}
	return &flagFrontierSnapshot{
		active:        append([]flagGrammar(nil), parent.active...),
		actions:       append([]semanticActionFlagGrammar(nil), parent.actions...),
		selected:      append([]flagGrammar(nil), parent.selected...),
		activeIndex:   cloneFlagMatchIndexForResponse(parent.activeIndex),
		actionIndex:   cloneActionFlagMatchIndexForResponse(parent.actionIndex),
		selectedIndex: cloneFlagMatchIndexForResponse(parent.selectedIndex),
	}
}

func cloneFlagMatchIndexForResponse(parent flagMatchIndex) flagMatchIndex {
	child := flagMatchIndex{
		flags:      append([]flagGrammar(nil), parent.flags...),
		candidates: append([]flagLexicalCandidate(nil), parent.candidates...),
		exactPlain: cloneStringIntSlicesForResponse(parent.exactPlain),
		attached:   make(map[string][]flagAttachedCandidate, len(parent.attached)),
		shortExact: cloneStringIntSlicesForResponse(parent.shortExact),
	}
	for key, values := range parent.attached {
		child.attached[key] = append([]flagAttachedCandidate(nil), values...)
	}
	if parent.attached == nil {
		child.attached = nil
	}
	return child
}

func cloneActionFlagMatchIndexForResponse(parent actionFlagMatchIndex) actionFlagMatchIndex {
	return actionFlagMatchIndex{
		actions:    append([]semanticActionFlagGrammar(nil), parent.actions...),
		candidates: append([]actionFlagLexicalCandidate(nil), parent.candidates...),
		exact:      cloneStringIntSlicesForResponse(parent.exact),
	}
}

func cloneStringIntSlicesForResponse(parent map[string][]int) map[string][]int {
	if parent == nil {
		return nil
	}
	child := make(map[string][]int, len(parent))
	for key, values := range parent {
		child[key] = append([]int(nil), values...)
	}
	return child
}

var (
	errInvalidArgumentRewrite              = errors.New("invocation: invalid argument rewrite")
	errInvalidResponseExpansionSplice      = errors.New("invocation: invalid response expansion splice")
	errInvalidResponseExpansionTokenOrigin = errors.New("invocation: invalid response expansion token origin")
	errInvalidResponseExpansionEntryOrigin = errors.New("invocation: invalid response expansion entry origin")
)

// responseIncludeObservation is an operation-local lexical observation for one
// successfully recognized include directive. BeforeArgument is the position in
// the current flattened grammar-input view immediately before which the include
// occurred; it is projection metadata, not a flattened-argv public artifact.
type responseIncludeObservation struct {
	Spelling       string
	Origin         originRef
	BeforeArgument int
}

// RewriteArgument replaces one prepared lexical entry without changing its
// provenance. It is used only for response-file literal-marker escaping after
// the selected command dialect has authoritatively enabled response syntax.
func (operation *invocation) RewriteArgument(index int, text string) error {
	if operation == nil {
		return errInvalidArgumentRewrite
	}
	return operation.analysis.rewriteArgument(index, text)
}

func (parent *analysisState) rewriteArgument(index int, text string) error {
	if parent == nil || index < 0 || index >= len(parent.args) {
		return errInvalidArgumentRewrite
	}
	parent.args[index] = text
	return nil
}

func (parent *analysisState) spliceResponseExpansion(operation *invocation, index int, spelling string, expansion committedResponseExpansion) error {
	if operation == nil || index < 0 || index >= len(parent.args) || spelling == "" {
		return errInvalidResponseExpansionSplice
	}
	includeOrigin := parent.provenance.argumentOrigins[index]
	for _, token := range expansion.Tokens {
		if token.Origin.operation != operation || !token.Origin.id.present() || !parent.hasOrigin(token.Origin.id) {
			return errInvalidResponseExpansionTokenOrigin
		}
	}
	for _, entry := range expansion.Entries {
		if entry.Origin.operation != operation || !entry.Origin.id.present() || !parent.hasOrigin(entry.Origin.id) {
			return errInvalidResponseExpansionEntryOrigin
		}
	}

	delta := len(expansion.Tokens) - 1
	for observation := range parent.provenance.responseIncludes {
		if parent.provenance.responseIncludes[observation].BeforeArgument > index {
			parent.provenance.responseIncludes[observation].BeforeArgument += delta
		}
	}
	observation2 := responseIncludeObservation{
		Spelling:       spelling,
		Origin:         includeOrigin,
		BeforeArgument: index,
	}
	parent.provenance.responseIncludes = append(parent.provenance.responseIncludes, observation2)
	emitted := 0
	for _, entry := range expansion.Entries {
		if entry.Include {
			observation3 := responseIncludeObservation{
				Spelling:       entry.Spelling,
				Origin:         entry.Origin,
				BeforeArgument: index + emitted,
			}
			parent.provenance.responseIncludes = append(parent.provenance.responseIncludes, observation3)
			continue
		}
		emitted++
	}

	args := make([]string, 0, len(parent.args)-1+len(expansion.Tokens))
	args = append(args, parent.args[:index]...)
	for _, token := range expansion.Tokens {
		args = append(args, token.Text)
	}
	args = append(args, parent.args[index+1:]...)

	origins := make([]originRef, 0, len(parent.provenance.argumentOrigins)-1+len(expansion.Tokens))
	origins = append(origins, parent.provenance.argumentOrigins[:index]...)
	for _, token := range expansion.Tokens {
		origins = append(origins, token.Origin)
	}
	origins = append(origins, parent.provenance.argumentOrigins[index+1:]...)

	parent.args = args
	parent.provenance.argumentOrigins = origins
	return nil
}

// ResponseIncludes returns the successful include observations in lexical commit
// order. The returned slice is detached from operation mutation.
func (operation *invocation) ResponseIncludes() []responseIncludeObservation {
	if operation == nil {
		return nil
	}
	return append([]responseIncludeObservation(nil), operation.analysis.provenance.responseIncludes...)
}
