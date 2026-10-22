package cli

import (
	"context"
	"maps"
)

// ValueProducerKind classifies the semantic producer that created a value candidate. A
// resolved aggregate may have several contributing producer kinds, so producer-kind truth remains on
// producer/candidate history rather than one aggregate ValueState classifier.
type ValueProducerKind uint8

func (kind ValueProducerKind) valid() bool {
	return kind >= ValueProducerKindFlag && kind <= ValueProducerKindDerived
}

const (
	// ValueProducerKindUnknown is the zero value and does not identify a producer kind.
	ValueProducerKindUnknown ValueProducerKind = iota
	// ValueProducerKindFlag identifies explicit input from a flag.
	ValueProducerKindFlag
	// ValueProducerKindArgument identifies explicit input from a positional argument.
	ValueProducerKindArgument
	// ValueProducerKindRemainder identifies input captured by an argument remainder.
	ValueProducerKindRemainder
	// ValueProducerKindPassThrough identifies input captured by a pass-through destination.
	ValueProducerKindPassThrough
	// ValueProducerKindShortcutDefault identifies a fallback default contributed by a command shortcut.
	ValueProducerKindShortcutDefault
	// ValueProducerKindEnvironment identifies a candidate produced from the invocation environment snapshot.
	ValueProducerKindEnvironment
	// ValueProducerKindInvocationDefault identifies a candidate supplied by an invocation DefaultProvider.
	ValueProducerKindInvocationDefault
	// ValueProducerKindComputedDefault identifies a candidate produced by a declared computed default.
	ValueProducerKindComputedDefault
	// ValueProducerKindStaticDefault identifies a candidate from a declared static default.
	ValueProducerKindStaticDefault
	// ValueProducerKindInitial identifies a destination initial value.
	ValueProducerKindInitial
	// ValueProducerKindSuppliedFact identifies a fact supplied through invocation fact supply.
	ValueProducerKindSuppliedFact
	// ValueProducerKindResolvedFact identifies a fact produced by declared fact resolution.
	ValueProducerKindResolvedFact
	// ValueProducerKindDerived identifies a value produced by declared derivation.
	ValueProducerKindDerived
)

// ResolutionTier classifies candidate precedence for value resolution. It is distinct
// from ValueProducerKind: kind describes the producer, while tier describes precedence.
type ResolutionTier uint8

const (
	// ResolutionTierUnknown is the zero value and does not identify precedence.
	ResolutionTierUnknown ResolutionTier = iota
	// ResolutionTierExplicit is the highest explicit invocation-input precedence tier.
	ResolutionTierExplicit
	// ResolutionTierShortcutDefault is the command-shortcut fallback-default tier.
	ResolutionTierShortcutDefault
	// ResolutionTierEnvironment is the tier populated from the invocation environment snapshot.
	ResolutionTierEnvironment
	// ResolutionTierInvocationDefault is the tier populated by invocation DefaultProviders.
	ResolutionTierInvocationDefault
	// ResolutionTierComputedDefault is the declared computed-default tier.
	ResolutionTierComputedDefault
	// ResolutionTierStaticDefault is the declared static-default tier.
	ResolutionTierStaticDefault
	// ResolutionTierInitial is the destination initial-value tier.
	ResolutionTierInitial
)

// ValueProducerOutcome records what happened when one producer participated in
// resolution, including inactivity, omission, contribution, failure, and blocking.
type ValueProducerOutcome uint8

const (
	// ValueProducerOutcomeUnknown is the zero value and records no producer outcome.
	ValueProducerOutcomeUnknown ValueProducerOutcome = iota
	// ValueProducerOutcomeInactive means the producer was not active for the resolved.
	ValueProducerOutcomeInactive
	// ValueProducerOutcomeSkipped means analysis intentionally skipped producer evaluation.
	ValueProducerOutcomeSkipped
	// ValueProducerOutcomeOmitted means the producer evaluated successfully but emitted no candidate.
	ValueProducerOutcomeOmitted
	// ValueProducerOutcomeContributed means the producer emitted one or more semantic contributions, including valid empty aggregate contribution where applicable.
	ValueProducerOutcomeContributed
	// ValueProducerOutcomeFailed means producer evaluation failed.
	ValueProducerOutcomeFailed
	// ValueProducerOutcomeBlocked means the producer could not proceed because required semantic state was blocked.
	ValueProducerOutcomeBlocked
)

// ResolutionState records the finalized state of one destination: inactive, absent,
// resolved, failed, or blocked. Unknown is reserved for insufficient/unavailable state.
type ResolutionState uint8

const (
	// ResolutionStateUnknown is the zero value and represents unavailable or insufficient state.
	ResolutionStateUnknown ResolutionState = iota
	// ResolutionStateInactive means the destination did not participate in the effective.
	ResolutionStateInactive
	// ResolutionStateAbsent means the destination was active but resolved to semantic absence.
	ResolutionStateAbsent
	// ResolutionStateResolved means resolution produced a finalized value.
	ResolutionStateResolved
	// ResolutionStateFailed means resolution failed.
	ResolutionStateFailed
	// ResolutionStateBlocked means resolution could not proceed because prerequisite semantic state was blocked.
	ResolutionStateBlocked
)

// ValueCandidateDisposition records a candidate's terminal role in canonical value
// history. It distinguishes effective, no-effect, suppressed, conflicting, rejected,
// and unresolved candidates rather than flattening them into selected/not-selected.
type ValueCandidateDisposition uint8

const (
	// ValueCandidateDispositionUnknown is the zero value and records no terminal candidate disposition.
	ValueCandidateDispositionUnknown ValueCandidateDisposition = iota
	// ValueCandidateDispositionEffective means the candidate contributes to the finalized result.
	ValueCandidateDispositionEffective
	// ValueCandidateDispositionNoEffect means a successful candidate had no semantic effect, such as an empty segment contribution.
	ValueCandidateDispositionNoEffect
	// ValueCandidateDispositionSuppressed means precedence or another semantic rule prevented the candidate from taking effect.
	ValueCandidateDispositionSuppressed
	// ValueCandidateDispositionConflicting means the candidate participates in a semantic conflict.
	ValueCandidateDispositionConflicting
	// ValueCandidateDispositionRejected means the candidate itself was invalid and rejected.
	ValueCandidateDispositionRejected
	// ValueCandidateDispositionUnresolved records terminal candidate history after resolution failed elsewhere; it is distinct from Unknown.
	ValueCandidateDispositionUnresolved
)

// ValueProducer is detached producer history for a ValueState. Origins and Sensitive
// preserve producer-level provenance/redaction, and Outcome records whether the
// producer actually contributed or failed/blocked/omitted.
type ValueProducer struct {
	Kind      ValueProducerKind
	Tier      ResolutionTier
	Outcome   ValueProducerOutcome
	Sensitive bool
	Origins   []InputOriginID
}

// ValueCandidate is a detached view of one retained candidate in canonical candidate
// order. It preserves disposition, contribution granularity, provenance, declaration
// identity, relationships, and redaction without exposing mutable configuration. Values
// referring to the same candidate in the same detached ValueState snapshot compare equal;
// comparison does not define semantic equality across distinct snapshots.
type ValueCandidate struct{ data valueCandidateData }

type valueCandidateData struct {
	id       valueCandidateRef
	snapshot *valueStateSnapshot
}

// ValueState is finalized detached observational state for one destination. Its zero
// value is unavailable/unknown. It never aliases execution configuration; sensitivity
// and candidate/provenance history remain valid even after application code mutates
// handed-off configuration.
type ValueState struct {
	_    [0]func()
	data valueStateData
}

type valueStateData struct{ snapshot *valueStateSnapshot }

// ValueStateFromContext returns finalized state for destination when the owning
// analysis callback is authorized to observe it. The lookup is observational only: it
// does not create a dependency, alter scheduling, or authorize undeclared access.
func ValueStateFromContext[C, T any](ctx context.Context, destination destinationLike[C, T]) (ValueState, bool) {
	return valueStateFromContext(ctx, destinationHandleOf(destination))
}

// State returns the destination's finalized ResolutionState, or Unknown for a zero
// or unavailable ValueState.
func (state ValueState) State() ResolutionState { return state.data.state() }

// Candidates returns retained candidates in canonical candidate order.
func (state ValueState) Candidates() []ValueCandidate { return state.data.candidates(false) }

// Contributors returns, in canonical order, the candidates that compose the resolved
// result. Aggregate values may have more than one contributor.
func (state ValueState) Contributors() []ValueCandidate { return state.data.candidates(true) }

// Producers returns detached producer history in canonical producer order.
func (state ValueState) Producers() []ValueProducer { return state.data.producers() }

// Sensitive reports whether the effective destination value is protected by the
// frontend redaction contract. It does not imply encryption or restrict application
// access to its own configuration.
func (state ValueState) Sensitive() bool { return state.data.sensitive() }

// ProducerKind returns the candidate producer kind, or ValueProducerKindUnknown when the
// candidate is unavailable.
func (candidate ValueCandidate) ProducerKind() ValueProducerKind {
	return candidate.data.producerKind()
}

// Tier returns the candidate producer's ResolutionTier, or ResolutionTierUnknown when
// the candidate is unavailable.
func (candidate ValueCandidate) Tier() ResolutionTier { return candidate.data.tier() }

// Disposition returns the candidate's terminal resolution disposition.
func (candidate ValueCandidate) Disposition() ValueCandidateDisposition {
	return candidate.data.disposition()
}

// Contribution returns the explicit contribution granularity represented by the
// candidate; it is not inferred from the Go container shape.
func (candidate ValueCandidate) Contribution() ValueCandidateContribution {
	return candidate.data.contribution()
}

// Declaration returns the producer declaration identity when one exists. It is
// distinct from domain-entry match identity and candidate-local identity.
func (candidate ValueCandidate) Declaration() DeclarationIdentity {
	return candidate.data.declaration()
}

// Sensitive reports whether this candidate is protected by the frontend redaction
// contract.
func (candidate ValueCandidate) Sensitive() bool { return candidate.data.sensitive() }

// InputSource returns the artifact-local source that directly produced this candidate,
// when the candidate has one.
func (candidate ValueCandidate) InputSource() (InputSourceID, bool) {
	return candidate.data.inputSource()
}

// Origins returns the candidate's provenance origins in canonical order.
func (candidate ValueCandidate) Origins() []InputOriginID { return candidate.data.origins() }

// Input returns retained source text only when it is present and not redacted. A
// protected or absent representation returns ok=false rather than a placeholder secret.
func (candidate ValueCandidate) Input() (string, bool) { return candidate.data.input() }

// InputRedacted reports that source text exists but was intentionally withheld by the
// redaction contract.
func (candidate ValueCandidate) InputRedacted() bool { return candidate.data.inputRedacted() }

// Representation returns the candidate's canonical detached value representation when
// available. The representation is semantic text/tokens, not replay argv or arbitrary
// application memory.
func (candidate ValueCandidate) Representation() (ValueRepresentation, bool) {
	return candidate.data.representation()
}

// DomainEntry returns the finite-domain entry matched by this candidate, if any.
func (candidate ValueCandidate) DomainEntry() ValueDomainEntryID { return candidate.data.domainEntry() }

// Suppression returns structured information explaining why a suppressed candidate
// did not become effective.
func (candidate ValueCandidate) Suppression() ValueCandidateSuppression {
	return candidate.data.suppression()
}

// SuppressedBy returns suppressing candidates in their recorded deterministic order.
func (candidate ValueCandidate) SuppressedBy() []ValueCandidate { return candidate.data.related(false) }

// ConflictsWith returns conflicting candidates in their recorded deterministic order.
func (candidate ValueCandidate) ConflictsWith() []ValueCandidate { return candidate.data.related(true) }

type valueStateSnapshot struct {
	value            ValueInspection
	candidateOrder   []valueCandidateRef
	contributors     []valueCandidateRef
	candidates       map[valueCandidateRef]ValueCandidateInspection
	candidateSources map[valueCandidateRef]InputSourceID
	suppressedBy     map[valueCandidateRef][]valueCandidateRef
	conflictsWith    map[valueCandidateRef][]valueCandidateRef
}

type valueStateContextKey struct{}

type valueStateContext struct {
	states  map[uint32]ValueState
	aliases map[destinationRef]uint32
}

func contextWithValueStates(ctx context.Context, states map[uint32]ValueState) context.Context {
	if ctx == nil || len(states) == 0 {
		return ctx
	}
	copyStates := make(map[uint32]ValueState, len(states))
	maps.Copy(copyStates, states)
	stateContext := valueStateContext{
		states: copyStates,
	}
	return context.WithValue(ctx, valueStateContextKey{}, stateContext)
}

func contextWithValueStateKeyAliases(ctx context.Context, aliases map[destinationRef]ValueID) context.Context {
	if ctx == nil || len(aliases) == 0 {
		return ctx
	}
	values, _ := ctx.Value(valueStateContextKey{}).(valueStateContext)
	copyAliases := make(map[destinationRef]uint32, len(values.aliases)+len(aliases))
	maps.Copy(copyAliases, values.aliases)
	for key, id := range aliases {
		if key.valid() && id.present() {
			copyAliases[key] = id.uint32()
		}
	}
	values.aliases = copyAliases
	return context.WithValue(ctx, valueStateContextKey{}, values)
}

func valueStateFromContext(ctx context.Context, destination *destinationHandle) (ValueState, bool) {
	if ctx == nil || destination == nil {
		return ValueState{}, false
	}
	values, ok := ctx.Value(valueStateContextKey{}).(valueStateContext)
	if !ok {
		return ValueState{}, false
	}
	id := uint32(0)
	if values.aliases != nil {
		if identity, ok := destination.currentIdentity().(stagedDestinationIdentity); ok {
			id = values.aliases[identity.ref]
		}
	}
	if id == 0 {
		if identity, ok := acceptedDestinationHandleIdentity(destination); ok {
			id = identity.id.uint32()
		}
	}
	if id == 0 {
		return ValueState{}, false
	}
	state, ok := values.states[id]
	return state, ok
}

func policyContextProjector(model *effectiveModel, analysis valueAnalysis) analysisPolicyContextProjector {
	return func(ctx context.Context, destinations []ValueID) context.Context {
		states := make(map[uint32]ValueState, len(destinations))
		for _, destination := range destinations {
			state, ok := prepareValueState(model, analysis, destination)
			if !ok {
				continue
			}
			states[destination.uint32()] = state
		}
		return contextWithValueStates(ctx, states)
	}
}

func prepareValueState(model *effectiveModel, analysis valueAnalysis, destination ValueID) (ValueState, bool) {
	resolved, ok := analysis.Destinations[destination]
	if !ok {
		return ValueState{}, false
	}
	value, present, complete := projectDestinationValue(model, destination, resolved, nil)
	if !present || !complete {
		return ValueState{}, false
	}

	candidates := make(map[valueCandidateRef]ValueCandidateInspection, len(resolved.Candidates))
	candidateSources := make(map[valueCandidateRef]InputSourceID, len(resolved.Candidates))
	suppressedBy := make(map[valueCandidateRef][]valueCandidateRef, len(resolved.Candidates))
	conflictsWith := make(map[valueCandidateRef][]valueCandidateRef, len(resolved.Candidates))
	for _, ref := range resolved.Candidates {
		observed, ok := analysis.Candidates[ref]
		if !ok || observed.Meta.Destination != destination {
			return ValueState{}, false
		}
		candidate, ok := projectObservedValueCandidate(model, observed, nil)
		if !ok {
			return ValueState{}, false
		}
		candidates[ref] = candidate
		suppressedBy[ref] = append([]valueCandidateRef(nil), observed.Meta.SuppressedBy...)
		conflictsWith[ref] = append([]valueCandidateRef(nil), observed.Meta.ConflictsWith...)
		if observed.InputSource.present() {
			candidateSources[ref] = observed.InputSource
		}
	}
	snapshot := &valueStateSnapshot{
		value:            value,
		candidateOrder:   append([]valueCandidateRef(nil), resolved.Candidates...),
		contributors:     append([]valueCandidateRef(nil), resolved.Contributors...),
		candidates:       candidates,
		candidateSources: candidateSources,
		suppressedBy:     suppressedBy,
		conflictsWith:    conflictsWith,
	}
	data := valueStateData{
		snapshot: snapshot,
	}
	state := ValueState{
		data: data,
	}
	return state, true
}

func (data valueStateData) state() ResolutionState {
	if data.snapshot == nil {
		return ResolutionStateUnknown
	}
	return data.snapshot.value.State
}

func (data valueStateData) candidates(contributors bool) []ValueCandidate {
	if data.snapshot == nil {
		return nil
	}
	refs := data.snapshot.candidateOrder
	if contributors {
		refs = data.snapshot.contributors
	}
	result := make([]ValueCandidate, 0, len(refs))
	for _, ref := range refs {
		if _, ok := data.snapshot.candidates[ref]; ok {
			result = append(result, ValueCandidate{
				data: valueCandidateData{
					id:       ref,
					snapshot: data.snapshot,
				},
			})
		}
	}
	return result
}

func (data valueStateData) producers() []ValueProducer {
	if data.snapshot == nil {
		return nil
	}
	result := make([]ValueProducer, len(data.snapshot.value.Producers))
	for i, producer := range data.snapshot.value.Producers {
		result[i] = ValueProducer{
			Kind:      producer.Kind,
			Tier:      producer.Tier,
			Outcome:   producer.Outcome,
			Sensitive: producer.Sensitive,
			Origins:   append([]InputOriginID(nil), producer.Origins...),
		}
	}
	return result
}

func (data valueStateData) sensitive() bool {
	return data.snapshot != nil && data.snapshot.value.Sensitive
}

func (data valueCandidateData) inspection() (ValueCandidateInspection, bool) {
	if data.snapshot == nil {
		return ValueCandidateInspection{}, false
	}
	value, ok := data.snapshot.candidates[data.id]
	return value, ok
}

func (data valueCandidateData) producerKind() ValueProducerKind {
	value, ok := data.inspection()
	if !ok || value.ProducerIndex < 0 || value.ProducerIndex >= len(data.snapshot.value.Producers) {
		return ValueProducerKindUnknown
	}
	return data.snapshot.value.Producers[value.ProducerIndex].Kind
}

func (data valueCandidateData) tier() ResolutionTier {
	value, ok := data.inspection()
	if !ok || value.ProducerIndex < 0 || value.ProducerIndex >= len(data.snapshot.value.Producers) {
		return ResolutionTierUnknown
	}
	return data.snapshot.value.Producers[value.ProducerIndex].Tier
}

func (data valueCandidateData) disposition() ValueCandidateDisposition {
	value, _ := data.inspection()
	return value.Disposition
}

func (data valueCandidateData) contribution() ValueCandidateContribution {
	value, _ := data.inspection()
	return value.Contribution
}

func (data valueCandidateData) declaration() DeclarationIdentity {
	value, _ := data.inspection()
	return value.Declaration
}

func (data valueCandidateData) sensitive() bool {
	value, _ := data.inspection()
	return value.Sensitive
}

func (data valueCandidateData) inputSource() (InputSourceID, bool) {
	if data.snapshot == nil {
		return InputSourceID{}, false
	}
	value, ok := data.snapshot.candidateSources[data.id]
	return value, ok
}

func (data valueCandidateData) origins() []InputOriginID {
	value, _ := data.inspection()
	return append([]InputOriginID(nil), value.Origins...)
}

func (data valueCandidateData) input() (string, bool) {
	value, ok := data.inspection()
	if !ok || value.Input == nil || value.Input.Redacted {
		return "", false
	}
	return value.Input.Text, true
}

func (data valueCandidateData) inputRedacted() bool {
	value, _ := data.inspection()
	return value.Input != nil && value.Input.Redacted
}

func (data valueCandidateData) representation() (ValueRepresentation, bool) {
	value, ok := data.inspection()
	if !ok || value.Representation == nil {
		return ValueRepresentation{}, false
	}
	return *value.Representation, true
}

func (data valueCandidateData) domainEntry() ValueDomainEntryID {
	value, _ := data.inspection()
	return value.DomainEntry
}

func (data valueCandidateData) suppression() ValueCandidateSuppression {
	value, _ := data.inspection()
	return value.Suppression
}

func (data valueCandidateData) related(conflicts bool) []ValueCandidate {
	if _, ok := data.inspection(); !ok {
		return nil
	}
	refs := data.snapshot.suppressedBy[data.id]
	if conflicts {
		refs = data.snapshot.conflictsWith[data.id]
	}
	result := make([]ValueCandidate, 0, len(refs))
	for _, ref := range refs {
		if _, ok := data.snapshot.candidates[ref]; ok {
			result = append(result, ValueCandidate{
				data: valueCandidateData{
					id:       ref,
					snapshot: data.snapshot,
				},
			})
		}
	}
	return result
}
