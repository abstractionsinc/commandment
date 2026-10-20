package cli

import "context"

// valueIssueKind identifies one operation-local failure discovered after
// structural ownership is fixed. These failures never cause grammar
// backtracking or alternate token ownership.
type valueIssueKind uint8

const (
	valueIssueNone valueIssueKind = iota
	valueIssueInvalidInput
	valueIssueResolutionConflict
	valueIssueProducerFailure
	valueIssueProducerBlocked
	valueIssueUnsupported
)

type valueIssue struct {
	Kind        valueIssueKind
	Destination ValueID
	Origins     []InputOriginID
	Err         error
}

// valueCandidateRef identifies the exact accepted declaration that
// contributed a candidate without depending on public artifact types.
type valueCandidateRef struct {
	owner *analysisState
	index int
}

func (ref valueCandidateRef) valid() bool {
	return ref.owner != nil && ref.index >= 0
}

func (analysis *analysisState) allocateValueCandidateRef() valueCandidateRef {
	if analysis == nil {
		panic("invocation: nil analysis state")
	}
	ref := valueCandidateRef{
		owner: analysis,
		index: analysis.nextValueCandidate,
	}
	analysis.nextValueCandidate++
	if analysis.nextValueCandidate < 0 {
		panic("invocation: value candidate reference exhausted")
	}
	return ref
}

type valueCandidateDeclaration struct {
	Flag             FlagID
	FlagSpelling     FlagSpellingID
	ArgumentTerm     ArgumentTermID
	EnvironmentInput EnvironmentInputID
}

type valueCandidateComponentObservation struct {
	Input     string
	HasInput  bool
	Value     any
	HasValue  bool
	Origins   []InputOriginID
	Sensitive bool
}

type valueCandidateEntryObservation struct {
	Key   valueCandidateComponentObservation
	Value valueCandidateComponentObservation
}

type valueCandidateObservation struct {
	Meta          valueCandidateMeta
	ProducerIndex int
	Declaration   valueCandidateDeclaration

	Input                  string
	HasInput               bool
	Value                  any
	HasValue               bool
	Key                    any
	HasKey                 bool
	Elements               []valueCandidateComponentObservation
	Entries                []valueCandidateEntryObservation
	InputSource            InputSourceID
	TokenRepresentation    []string
	HasTokenRepresentation bool
}

type producerOutcome uint8

const (
	producerOutcomeUnknown producerOutcome = iota
	producerOutcomeInactive
	producerOutcomeSkipped
	producerOutcomeOmitted
	producerOutcomeContributed
	producerOutcomeFailed
	producerOutcomeBlocked
)

type producerTracker struct {
	called    bool
	present   bool
	err       error
	blockedBy []ValueID
}

type producerBlockedError struct {
	destinations []ValueID
}

func (err producerBlockedError) Error() string {
	return "value producer blocked by unresolved dependency"
}

type producerObservation struct {
	Kind         declaredProducerKind
	ProducerKind ValueProducerKind
	Tier         ResolutionTier
	Outcome      producerOutcome
	Sensitive    bool
	Dependencies []ValueID
	BlockedBy    []ValueID
	Origins      []InputOriginID
	Err          error
}

type valueCandidateComponentReference struct {
	Candidate valueCandidateRef
	Index     int
}

type destinationResolution struct {
	Destination ValueID
	State       analysisDestinationState
	Value       any
	Sensitive   bool

	Candidates   []valueCandidateRef
	Contributors []valueCandidateRef
	Elements     []valueCandidateComponentReference
	Entries      []valueCandidateComponentReference
	Producers    []producerObservation
	BlockedBy    []ValueID
	Err          error
}

type valueAnalysis struct {
	Destinations map[ValueID]destinationResolution
	Candidates   map[valueCandidateRef]valueCandidateObservation
	Issues       []valueIssue
	Unsupported  bool
}

// analyzeValues resolves the selected/effective destination graph after
// grammar ownership. It intentionally consumes the already-established
// physical/semantic ownership result rather than reparsing input. The first
// implementation wave covers ordinary flag/argument/environment candidates
// and accepted static/computed/fact/derived producer plans plus atomically
// accepted invocation-supplied defaults/facts. Profile-dependent grammars remain
// unsupported until selected-profile ownership is established before parsing;
// it is never guessed from Go type identity.
func analyzeValues(ctx context.Context, operation *invocation, grammar grammarResult) valueAnalysis {
	if operation == nil || operation.analysis == nil {
		panic("invocation: nil operation")
	}
	return analyzeValuesAt(ctx, operation, operation.analysis, grammar)
}

func analyzeValuesAt(ctx context.Context, operation *invocation, state *analysisState, grammar grammarResult) valueAnalysis {
	if operation == nil || state == nil {
		panic("invocation: nil operation analysis")
	}
	release := state.work.enter()
	defer release()

	analysis := operationDependencySeed(state)
	if len(grammar.Issues) != 0 {
		return analysis
	}
	view := state.effectiveModel
	model, ok := view.EffectiveGrammar()
	if !ok {
		analysis.Unsupported = true
		return analysis
	}

	selected := make(map[CommandID]struct{}, len(grammar.Selection.Path))
	for _, command := range grammar.Selection.Path {
		selected[command] = struct{}{}
	}

	plans := make(map[ValueID][]valueCandidatePlan)
	builder := valuePlanBuilder{
		state:    state,
		profiles: grammar.SelectedProfiles,
		selected: selected,
		plans:    plans,
		analysis: &analysis,
	}
	builder.appendGrammarValueCandidatePlans(ctx, operation, model, grammar)
	builder.appendEnvironmentValueCandidatePlans(operation)
	builder.appendValueSupplyPlans(ctx)
	builder.appendShortcutDefaultPlans(model)

	records := view.destinationsForCommandsOrdered(grammar.Selection.Path)
	nodes := make([]valueSettlementNode, 0, len(records))
	for _, record := range records {
		if !analysisGrammarProfileApplies(grammar.SelectedProfiles, record.command, record.meta.profile) {
			continue
		}
		nodes = append(nodes, prepareValueSettlementNode(record, grammar.SelectedProfiles))
	}
	settleValueDestinations(ctx, operation, state, model, grammar, nodes, plans, &analysis)
	return analysis
}

func operationDependencySeed(state *analysisState) valueAnalysis {
	state.dependencyMu.Lock()
	defer state.dependencyMu.Unlock()
	return cloneValueAnalysis(state.dependencyValues)
}

func storeOperationDependencySeed(state *analysisState, analysis valueAnalysis) {
	state.dependencyMu.Lock()
	defer state.dependencyMu.Unlock()
	state.dependencyValues = cloneValueAnalysis(analysis)
}

func cloneValueAnalysis(analysis valueAnalysis) valueAnalysis {
	cloned := valueAnalysis{
		Destinations: make(map[ValueID]destinationResolution, len(analysis.Destinations)),
		Candidates:   make(map[valueCandidateRef]valueCandidateObservation, len(analysis.Candidates)),
		Issues:       append([]valueIssue(nil), analysis.Issues...),
		Unsupported:  analysis.Unsupported,
	}
	for id, resolution := range analysis.Destinations {
		cloned.Destinations[id] = cloneDestinationResolution(resolution)
	}
	for id, candidate := range analysis.Candidates {
		cloned.Candidates[id] = cloneValueCandidateObservation(candidate)
	}
	for index := range cloned.Issues {
		cloned.Issues[index].Origins = append([]InputOriginID(nil), analysis.Issues[index].Origins...)
	}
	return cloned
}

func ensureValueAnalysisMaps(analysis *valueAnalysis) {
	if analysis.Destinations == nil {
		analysis.Destinations = make(map[ValueID]destinationResolution)
	}
	if analysis.Candidates == nil {
		analysis.Candidates = make(map[valueCandidateRef]valueCandidateObservation)
	}
}

func cloneValueAnalysisDestinations(analysis valueAnalysis, destinations map[ValueID]struct{}) valueAnalysis {
	cloned := valueAnalysis{
		Destinations: make(map[ValueID]destinationResolution, len(destinations)),
		Candidates:   make(map[valueCandidateRef]valueCandidateObservation),
		Unsupported:  analysis.Unsupported,
	}
	candidateIDs := make(map[valueCandidateRef]struct{})
	for id := range destinations {
		resolution, ok := analysis.Destinations[id]
		if !ok {
			continue
		}
		copyResolution := cloneDestinationResolution(resolution)
		cloned.Destinations[id] = copyResolution
		for _, candidate := range resolution.Candidates {
			candidateIDs[candidate] = struct{}{}
		}
		for _, candidate := range resolution.Contributors {
			candidateIDs[candidate] = struct{}{}
		}
		for _, element := range resolution.Elements {
			candidateIDs[element.Candidate] = struct{}{}
		}
		for _, entry := range resolution.Entries {
			candidateIDs[entry.Candidate] = struct{}{}
		}
	}
	for id := range candidateIDs {
		if candidate, ok := analysis.Candidates[id]; ok {
			cloned.Candidates[id] = cloneValueCandidateObservation(candidate)
		}
	}
	for _, issue := range analysis.Issues {
		if _, ok := destinations[issue.Destination]; !ok {
			continue
		}
		copyIssue := issue
		copyIssue.Origins = append([]InputOriginID(nil), issue.Origins...)
		cloned.Issues = append(cloned.Issues, copyIssue)
	}
	return cloned
}

func cloneDestinationResolution(resolution destinationResolution) destinationResolution {
	copyResolution := resolution
	copyResolution.Value = snapshotMutableValue(resolution.Value)
	copyResolution.Candidates = append([]valueCandidateRef(nil), resolution.Candidates...)
	copyResolution.Contributors = append([]valueCandidateRef(nil), resolution.Contributors...)
	copyResolution.Elements = append([]valueCandidateComponentReference(nil), resolution.Elements...)
	copyResolution.Entries = append([]valueCandidateComponentReference(nil), resolution.Entries...)
	copyResolution.Producers = append([]producerObservation(nil), resolution.Producers...)
	for index := range copyResolution.Producers {
		copyResolution.Producers[index].Dependencies = append(
			[]ValueID(nil),
			resolution.Producers[index].Dependencies...,
		)
		copyResolution.Producers[index].BlockedBy = append(
			[]ValueID(nil),
			resolution.Producers[index].BlockedBy...,
		)
		copyResolution.Producers[index].Origins = append(
			[]InputOriginID(nil),
			resolution.Producers[index].Origins...,
		)
	}
	copyResolution.BlockedBy = append([]ValueID(nil), resolution.BlockedBy...)
	return copyResolution
}

func cloneValueCandidateObservation(candidate valueCandidateObservation) valueCandidateObservation {
	copyCandidate := candidate
	copyCandidate.Value = snapshotMutableValue(candidate.Value)
	copyCandidate.Key = snapshotMutableValue(candidate.Key)
	copyCandidate.TokenRepresentation = append([]string(nil), candidate.TokenRepresentation...)
	copyCandidate.Elements = append([]valueCandidateComponentObservation(nil), candidate.Elements...)
	for index := range copyCandidate.Elements {
		copyCandidate.Elements[index].Value = snapshotMutableValue(candidate.Elements[index].Value)
		copyCandidate.Elements[index].Origins = append(
			[]InputOriginID(nil),
			candidate.Elements[index].Origins...,
		)
	}
	copyCandidate.Entries = append([]valueCandidateEntryObservation(nil), candidate.Entries...)
	for index := range copyCandidate.Entries {
		copyCandidate.Entries[index].Key.Value = snapshotMutableValue(candidate.Entries[index].Key.Value)
		copyCandidate.Entries[index].Key.Origins = append(
			[]InputOriginID(nil),
			candidate.Entries[index].Key.Origins...,
		)
		copyCandidate.Entries[index].Value.Value = snapshotMutableValue(
			candidate.Entries[index].Value.Value,
		)
		copyCandidate.Entries[index].Value.Origins = append(
			[]InputOriginID(nil),
			candidate.Entries[index].Value.Origins...,
		)
	}
	return copyCandidate
}
