package cli

import (
	"context"
	"slices"
	"time"
)

// SignalMatch is an immutable declared signal-selection expression. Its zero value is not a valid
// policy match; selection is finalized before an effective policy activates.
type SignalMatch[C any] interface {
	signalMatch(config *C)
}

type signalSelector[C any] struct {
	state *authoringSignalMatch
}

func (signalSelector[C]) signalMatch(*C) {}

type signalPolicyMatch[C any] struct {
	state *authoringSignalMatch
}

func (signalPolicyMatch[C]) signalMatch(*C) {}

type signalAggregationMatch[C any] struct {
	state *authoringSignalMatch
}

func (signalAggregationMatch[C]) signalMatch(*C) {}

type signalMatches[C any] struct {
	session *declarationSession
	profile grammarProfileCaseRef
}

// Interrupt starts a selector for the source's interrupt semantic signal class.
func (matches signalMatches[C]) Interrupt() signalSelector[C] {
	return stageSignalSelector(matches, semanticSignalSelectorInterrupt(), nil)
}

// Terminate starts a selector for the source's terminate semantic signal class.
func (matches signalMatches[C]) Terminate() signalSelector[C] {
	return stageSignalSelector(matches, semanticSignalSelectorTerminate(), nil)
}

// Shutdown starts a selector for the source's shutdown semantic signal class.
func (matches signalMatches[C]) Shutdown() signalSelector[C] {
	return stageSignalSelector(matches, semanticSignalSelectorShutdown(), nil)
}

// On starts a selector for the exact supplied signals.
func (matches signalMatches[C]) On(selected ...Signal) signalSelector[C] {
	return stageSignalSelector(matches, semanticSignalSelectorListed(), selected)
}

// Asynchronous starts a selector for the source's asynchronous semantic signal class.
func (matches signalMatches[C]) Asynchronous() signalSelector[C] {
	return stageSignalSelector(matches, semanticSignalSelectorAsynchronous(), nil)
}

// When restricts this selector to events whose declared Condition is satisfied.
func (selector signalSelector[C]) When(condition Condition) signalSelector[C] {
	return signalSelectorCondition(selector, condition, false)
}

// Unless restricts this selector to events whose declared Condition is not satisfied.
func (selector signalSelector[C]) Unless(condition Condition) signalSelector[C] {
	return signalSelectorCondition(selector, condition, true)
}

// Except excludes the supplied exact signals from this selector.
func (selector signalSelector[C]) Except(excluded ...Signal) signalSelector[C] {
	return signalSelectorExcept(selector, excluded)
}

// Origin restricts matching to the supplied source-origin classes.
func (selector signalSelector[C]) Origin(allowed ...SignalOrigin) signalSelector[C] {
	return signalSelectorOrigins(selector, allowed, false)
}

// ExceptOrigin excludes the supplied source-origin classes.
func (selector signalSelector[C]) ExceptOrigin(excluded ...SignalOrigin) signalSelector[C] {
	return signalSelectorOrigins(selector, excluded, true)
}

// ExceptSender excludes events whose sender is target; phase legality still depends on whether
// application-owned target behavior is available.
func (selector signalSelector[C]) ExceptSender(target SignalTarget[C]) signalSelector[C] {
	return signalSelectorSender(selector, target)
}

// During restricts matching to the supplied semantic signal phases.
func (selector signalSelector[C]) During(phases ...SignalPhase) signalSelector[C] {
	return signalSelectorPhases(selector, phases)
}

// At fixes the occurrence selector to exactly count for this policy lineage.
func (selector signalSelector[C]) At(count int) signalPolicyMatch[C] {
	return signalOccurrence(selector, count, count)
}

// AtMost fixes the occurrence selector to occurrences up to count.
func (selector signalSelector[C]) AtMost(count int) signalPolicyMatch[C] {
	return signalOccurrence(selector, 1, count)
}

// AtLeast fixes the occurrence selector to occurrences from count onward.
func (selector signalSelector[C]) AtLeast(count int) signalPolicyMatch[C] {
	return signalOccurrence(selector, count, 0)
}

// Between fixes the occurrence selector to the inclusive occurrence range.
func (selector signalSelector[C]) Between(minimum, maximum int) signalPolicyMatch[C] {
	return signalOccurrence(selector, minimum, maximum)
}

// Each selects per-event aggregation for fixed signal routing windows.
func (selector signalSelector[C]) Each() signalAggregationMatch[C] {
	return signalAggregation(selector, semanticSignalAggregationEach())
}

// Coalesce selects coalesced-window aggregation, preserving the full occurrence range while using
// the first notification as the representative.
func (selector signalSelector[C]) Coalesce() signalAggregationMatch[C] {
	return signalAggregation(selector, semanticSignalAggregationCoalesced())
}

// Latest selects latest-window aggregation, preserving the full occurrence range while using the
// last notification as the representative.
func (selector signalSelector[C]) Latest() signalAggregationMatch[C] {
	return signalAggregation(selector, semanticSignalAggregationLatest())
}

// Deferred marks this policy for dynamic action-dispatch deferral; source acceptance, counting, and
// nondeferred policies continue normally.
func (selector signalSelector[C]) Deferred() signalPolicyMatch[C] {
	return signalPolicyMatchModifier[C](selector, true, false, nil)
}

// ContinueMatching allows later matching policies for the same event only after this policy's
// immediate action sequence succeeds.
func (selector signalSelector[C]) ContinueMatching() signalPolicyMatch[C] {
	return signalPolicyMatchModifier[C](selector, false, true, nil)
}

// ContinueMatching allows later matching policies only after this policy's immediate sequence
// succeeds.
func (match signalPolicyMatch[C]) ContinueMatching() signalPolicyMatch[C] {
	return signalPolicyMatchModifier[C](match, false, true, nil)
}

// Hint attaches ordinary localizable source text to the signal policy without changing matching or
// action semantics.
func (selector signalSelector[C]) Hint(hint string) signalPolicyMatch[C] {
	return signalPolicyMatchModifier[C](selector, false, false, &hint)
}

// Each selects per-event aggregation for the already occurrence-qualified match.
func (match signalPolicyMatch[C]) Each() signalAggregationMatch[C] {
	return signalAggregation(match, semanticSignalAggregationEach())
}

// Coalesce selects coalesced-window aggregation for the already occurrence-qualified match.
func (match signalPolicyMatch[C]) Coalesce() signalAggregationMatch[C] {
	return signalAggregation(match, semanticSignalAggregationCoalesced())
}

// Latest selects latest-window aggregation for the already occurrence-qualified match.
func (match signalPolicyMatch[C]) Latest() signalAggregationMatch[C] {
	return signalAggregation(match, semanticSignalAggregationLatest())
}

// Deferred marks the occurrence-qualified policy for dynamic action-dispatch deferral.
func (match signalPolicyMatch[C]) Deferred() signalPolicyMatch[C] {
	return signalPolicyMatchModifier[C](match, true, false, nil)
}

// Hint attaches ordinary localizable source text without altering the qualified match.
func (match signalPolicyMatch[C]) Hint(hint string) signalPolicyMatch[C] {
	return signalPolicyMatchModifier[C](match, false, false, &hint)
}

// Deferred marks this aggregated policy for dynamic action-dispatch deferral.
func (match signalAggregationMatch[C]) Deferred() signalAggregationMatch[C] {
	return signalAggregationMatchModifier[C](match, true, false, nil)
}

// ContinueMatching allows later matching policies only after this policy's immediate sequence
// succeeds.
func (match signalAggregationMatch[C]) ContinueMatching() signalAggregationMatch[C] {
	return signalAggregationMatchModifier[C](match, false, true, nil)
}

// Hint attaches ordinary localizable source text without changing aggregation or selection.
func (match signalAggregationMatch[C]) Hint(hint string) signalAggregationMatch[C] {
	return signalAggregationMatchModifier[C](match, false, false, &hint)
}

type signalConditionClause struct {
	condition Condition
	negated   bool
}

type authoringSignalTarget struct {
	name          string
	deliver       signalTargetDeliverCallback
	matchesSender signalTargetMatchesSenderCallback
	validName     bool
}

type signalCounterLineage struct{}

type authoringSignalMatch struct {
	owner           *draftIdentity
	lineage         *signalCounterLineage
	kind            signalSelectorKind
	signals         []Signal
	excludedSignals []Signal
	origins         []SignalOrigin
	excludedOrigins []SignalOrigin
	excludedSender  authoringSignalTarget
	hasSender       bool
	phases          []SignalPhase
	conditions      []signalConditionClause
	minimum         int
	maximum         int
	aggregation     SignalAggregation
	deferred        bool
	deferredSet     bool
	continueMatch   bool
	continueSet     bool
	hint            string
	hintSet         bool
	phasesSet       bool
	valid           bool
}

type authoringSignalAction struct {
	kind        signalActionKind
	name        string
	handler     signalHandlerCallback
	target      authoringSignalTarget
	signal      Signal
	signalValue *destinationHandle
	endpoint    signalEndpointDraftRef
	delayed     bool
	delay       time.Duration
	delayValue  *destinationHandle
	exitCode    int
	valid       bool
}

type signalActionPlanState struct {
	config       any
	owner        *draftIdentity
	actions      []authoringSignalAction
	pendingDelay bool
	delay        time.Duration
	delayValue   *destinationHandle
	valid        bool
}

func stageSignalSelector[C any](matches signalMatches[C], kind signalSelectorKind, selected []Signal) signalSelector[C] {
	state := &authoringSignalMatch{
		lineage:     &signalCounterLineage{},
		kind:        kind,
		minimum:     1,
		aggregation: SignalAggregationEach,
	}
	if matches.session != nil {
		matches.session.use(func(draft *declarationDraft, _ commandRef) {
			state.owner = draft.identity
		})
	}
	state.signals = append([]Signal(nil), selected...)
	state.valid = state.owner != nil && kind.Valid() && validSignalList(state.signals)
	if kind == signalSelectorKindListed && len(state.signals) == 0 {
		state.valid = false
	}
	return signalSelector[C]{
		state: state,
	}
}

func cloneSignalMatch(input *authoringSignalMatch) *authoringSignalMatch {
	if input == nil {
		return nil
	}
	cloned := *input
	cloned.signals = append([]Signal(nil), input.signals...)
	cloned.excludedSignals = append([]Signal(nil), input.excludedSignals...)
	cloned.origins = append([]SignalOrigin(nil), input.origins...)
	cloned.excludedOrigins = append([]SignalOrigin(nil), input.excludedOrigins...)
	cloned.phases = append([]SignalPhase(nil), input.phases...)
	cloned.conditions = append([]signalConditionClause(nil), input.conditions...)
	return &cloned
}

func signalMatchImpl[C any](match SignalMatch[C]) *authoringSignalMatch {
	switch match := match.(type) {
	case signalSelector[C]:
		if match.state != nil {
			return match.state
		}
	case signalPolicyMatch[C]:
		if match.state != nil {
			return match.state
		}
	case signalAggregationMatch[C]:
		if match.state != nil {
			return match.state
		}
	}
	return nil
}

func signalSelectorCondition[C any](selector signalSelector[C], condition Condition, negated bool) signalSelector[C] {
	impl := cloneSignalMatch(signalMatchImpl[C](selector))
	if impl == nil {
		return signalSelector[C]{}
	}
	if condition.state == nil || !condition.state.valid || condition.state.owner != impl.owner {
		impl.valid = false
	} else {
		impl.conditions = append(impl.conditions, signalConditionClause{
			condition: condition,
			negated:   negated,
		})
	}
	return signalSelector[C]{
		state: impl,
	}
}

func signalSelectorExcept[C any](selector signalSelector[C], excluded []Signal) signalSelector[C] {
	impl := cloneSignalMatch(signalMatchImpl[C](selector))
	if impl == nil {
		return signalSelector[C]{}
	}
	if !validSignalList(excluded) {
		impl.valid = false
	} else {
		impl.excludedSignals = append(impl.excludedSignals, excluded...)
	}
	return signalSelector[C]{
		state: impl,
	}
}

func signalSelectorOrigins[C any](selector signalSelector[C], origins []SignalOrigin, excluded bool) signalSelector[C] {
	impl := cloneSignalMatch(signalMatchImpl[C](selector))
	if impl == nil {
		return signalSelector[C]{}
	}
	switch {
	case !validSignalOrigins(origins):
		impl.valid = false
	case excluded:
		impl.excludedOrigins = append(impl.excludedOrigins, origins...)
	default:
		impl.origins = append(impl.origins, origins...)
	}
	return signalSelector[C]{
		state: impl,
	}
}

func signalSelectorSender[C any](selector signalSelector[C], target SignalTarget[C]) signalSelector[C] {
	impl := cloneSignalMatch(signalMatchImpl[C](selector))
	if impl == nil {
		return signalSelector[C]{}
	}
	erased := eraseSignalTarget(target)
	if !erased.validName || erased.matchesSender == nil || impl.hasSender {
		impl.valid = false
	} else {
		impl.excludedSender = erased
		impl.hasSender = true
	}
	return signalSelector[C]{
		state: impl,
	}
}

func signalSelectorPhases[C any](selector signalSelector[C], phases []SignalPhase) signalSelector[C] {
	impl := cloneSignalMatch(signalMatchImpl[C](selector))
	if impl == nil {
		return signalSelector[C]{}
	}
	if impl.phasesSet || !validSignalPhases(phases) {
		impl.valid = false
	} else {
		impl.phases = normalizedSignalPhases(phases)
		impl.phasesSet = true
	}
	return signalSelector[C]{
		state: impl,
	}
}

func signalOccurrence[C any](selector signalSelector[C], minimum, maximum int) signalPolicyMatch[C] {
	impl := cloneSignalMatch(signalMatchImpl[C](selector))
	if impl == nil {
		return signalPolicyMatch[C]{}
	}
	if minimum < 1 || maximum < 0 || maximum != 0 && maximum < minimum {
		impl.valid = false
	} else {
		impl.minimum = minimum
		impl.maximum = maximum
	}
	return signalPolicyMatch[C]{
		state: impl,
	}
}

func signalAggregation[C any](match SignalMatch[C], aggregation SignalAggregation) signalAggregationMatch[C] {
	impl := cloneSignalMatch(signalMatchImpl(match))
	if impl == nil {
		return signalAggregationMatch[C]{}
	}
	impl.aggregation = aggregation
	return signalAggregationMatch[C]{
		state: impl,
	}
}

func signalMatchModifier[C any](match SignalMatch[C], deferred, continueMatching bool, hint *string) *authoringSignalMatch {
	impl := cloneSignalMatch(signalMatchImpl(match))
	if impl == nil {
		return nil
	}
	if deferred {
		if impl.deferredSet {
			impl.valid = false
		} else {
			impl.deferred = true
			impl.deferredSet = true
		}
	}
	if continueMatching {
		if impl.continueSet {
			impl.valid = false
		} else {
			impl.continueMatch = true
			impl.continueSet = true
		}
	}
	if hint != nil {
		if impl.hintSet || containsForbiddenDeclarationText(*hint) {
			impl.valid = false
		} else {
			impl.hint = *hint
			impl.hintSet = true
		}
	}
	return impl
}

func signalPolicyMatchModifier[C any](match SignalMatch[C], deferred, continueMatching bool, hint *string) signalPolicyMatch[C] {
	return signalPolicyMatch[C]{
		state: signalMatchModifier(match, deferred, continueMatching, hint),
	}
}

func signalAggregationMatchModifier[C any](match SignalMatch[C], deferred, continueMatching bool, hint *string) signalAggregationMatch[C] {
	return signalAggregationMatch[C]{
		state: signalMatchModifier(match, deferred, continueMatching, hint),
	}
}

func stageSignalClear[C any](signals *Signals[C]) {
	if signals == nil || signals.session == nil {
		return
	}

	signals.session.use(func(draft *declarationDraft, command commandRef) {
		record := signalClearDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  signals.profile,
			valid:    true,
		}
		draft.signalClears.append(draft, record)
	})
}

func stageSignalPolicy[C any](signals *Signals[C], match SignalMatch[C], plan SignalActionPlan[C]) {
	if signals == nil || signals.session == nil {
		return
	}

	m := cloneSignalMatch(signalMatchImpl(match))
	p := signalActionPlanStateOf(plan)
	signals.session.use(func(draft *declarationDraft, command commandRef) {
		valid := m != nil && p != nil && m.valid && p.valid && !p.pendingDelay &&
			m.owner == draft.identity && p.owner == draft.identity && len(p.actions) > 0
		record := signalPolicyDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  signals.profile,
			match:    m,
			plan:     p,
			valid:    valid,
		}
		draft.signalPolicies.append(draft, record)
	})
}

func validSignalList(values []Signal) bool {
	seen := map[Signal]struct{}{}
	for _, signal := range values {
		if !signal.valid() {
			return false
		}
		if _, ok := seen[signal]; ok {
			return false
		}
		seen[signal] = struct{}{}
	}
	return true
}

func validSignalOrigins(values []SignalOrigin) bool {
	seen := map[SignalOrigin]struct{}{}
	for _, v := range values {
		if v < SignalOriginTerminal || v > SignalOriginRuntime {
			return false
		}
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}

func validSignalPhases(values []SignalPhase) bool {
	if len(values) == 0 {
		return false
	}
	seen := map[SignalPhase]struct{}{}
	for _, v := range values {
		if v < SignalPhaseResolution || v > SignalPhaseCleanup {
			return false
		}
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}

func normalizedSignalPhases(values []SignalPhase) []SignalPhase {
	result := make([]SignalPhase, 0, len(values))
	for _, phase := range []SignalPhase{SignalPhaseResolution, SignalPhaseExecution, SignalPhaseCleanup} {
		if slices.Contains(values, phase) {
			result = append(result, phase)
		}
	}
	return result
}

func semanticSignalSelectorInterrupt() signalSelectorKind {
	return signalSelectorKindInterrupt
}

func semanticSignalSelectorTerminate() signalSelectorKind {
	return signalSelectorKindTerminate
}

func semanticSignalSelectorListed() signalSelectorKind {
	return signalSelectorKindListed
}

func semanticSignalSelectorAsynchronous() signalSelectorKind {
	return signalSelectorKindAsynchronous
}

func semanticSignalSelectorShutdown() signalSelectorKind {
	return signalSelectorKindShutdown
}

func semanticSignalAggregationEach() SignalAggregation {
	return SignalAggregationEach
}

func semanticSignalAggregationCoalesced() SignalAggregation {
	return SignalAggregationCoalesced
}

func semanticSignalAggregationLatest() SignalAggregation {
	return SignalAggregationLatest
}

type signalPolicyRecord struct {
	meta       recordMeta[SignalPolicyID]
	command    CommandID
	counter    SignalCounterID
	definition signalPolicyDefinition
}

func acceptSignalPolicyRecord(id SignalPolicyID, command CommandID, position declarationPosition) signalPolicyRecord {
	mustCommand(command)
	return signalPolicyRecord{
		meta:    acceptRecordMeta(id, position),
		command: command,
	}
}

func (record signalPolicyRecord) Definition() (signalPolicyDefinition, bool) {
	return record.definition, record.definition.Valid()
}

func (record signalPolicyRecord) Counter() (SignalCounterID, bool) {
	return record.counter, record.counter.present()
}

type signalCounterRecord struct {
	meta   recordMeta[SignalCounterID]
	policy SignalPolicyID
}

func acceptSignalCounterRecord(id SignalCounterID, policy SignalPolicyID, position declarationPosition) signalCounterRecord {
	if !policy.present() {
		panic("semantic: invalid signal-policy identity")
	}
	return signalCounterRecord{
		meta:   acceptRecordMeta(id, position),
		policy: policy,
	}
}

type signalSelectorKind uint8

const (
	signalSelectorKindUnknown signalSelectorKind = iota
	signalSelectorKindInterrupt
	signalSelectorKindTerminate
	signalSelectorKindListed
	signalSelectorKindAsynchronous
	signalSelectorKindShutdown
)

func (kind signalSelectorKind) Valid() bool {
	return kind >= signalSelectorKindInterrupt && kind <= signalSelectorKindShutdown
}

type signalSelectorDefinition struct {
	kind                 signalSelectorKind
	signals              []Signal
	excludedSignals      []Signal
	origins              []SignalOrigin
	excludedOrigins      []SignalOrigin
	excludedSender       string
	excludedSenderTarget signalTargetDefinition
	phases               []SignalPhase
}

type signalSelectorInput struct {
	Kind            signalSelectorKind
	Signals         []Signal
	ExcludedSignals []Signal
	Origins         []SignalOrigin
	ExcludedOrigins []SignalOrigin
	ExcludedSender  string
	Phases          []SignalPhase
}

func acceptSignalSelectorDefinition(input signalSelectorInput) signalSelectorDefinition {
	definition := signalSelectorDefinition{
		kind:            input.Kind,
		signals:         append([]Signal(nil), input.Signals...),
		excludedSignals: append([]Signal(nil), input.ExcludedSignals...),
		origins:         append([]SignalOrigin(nil), input.Origins...),
		excludedOrigins: append([]SignalOrigin(nil), input.ExcludedOrigins...),
		excludedSender:  input.ExcludedSender,
		phases:          append([]SignalPhase(nil), input.Phases...),
	}

	return definition
}

func (selector signalSelectorDefinition) WithExcludedSenderTarget(target signalTargetDefinition) signalSelectorDefinition {
	if selector.excludedSender == "" || target.Name() != selector.excludedSender || target.MatchesSender() == nil {
		selector.kind = 0
		return selector
	}
	selector.excludedSenderTarget = target
	return selector
}

func (selector signalSelectorDefinition) Valid() bool {
	if !selector.kind.Valid() || selector.kind == signalSelectorKindListed && len(selector.signals) == 0 ||
		len(selector.phases) == 0 {
		return false
	}
	if selector.kind != signalSelectorKindListed && len(selector.signals) != 0 {
		return false
	}
	if !validSignalSet(selector.signals) || !validSignalSet(selector.excludedSignals) {
		return false
	}
	seenOrigins := make(map[SignalOrigin]struct{}, len(selector.origins)+len(selector.excludedOrigins))
	for _, values := range [][]SignalOrigin{selector.origins, selector.excludedOrigins} {
		local := make(map[SignalOrigin]struct{}, len(values))
		for _, origin := range values {
			if origin < SignalOriginTerminal || origin > SignalOriginRuntime {
				return false
			}
			if _, duplicate := local[origin]; duplicate {
				return false
			}
			local[origin] = struct{}{}
			seenOrigins[origin] = struct{}{}
		}
	}
	seenPhases := make(map[SignalPhase]struct{}, len(selector.phases))
	for _, phase := range selector.phases {
		if !phase.valid() {
			return false
		}
		if _, duplicate := seenPhases[phase]; duplicate {
			return false
		}
		seenPhases[phase] = struct{}{}
	}
	return true
}

func (selector signalSelectorDefinition) Kind() signalSelectorKind { return selector.kind }

// Signals returns the selector's explicitly included semantic signal identities.
func (selector signalSelectorDefinition) Signals() []Signal {
	return append([]Signal(nil), selector.signals...)
}

// ExcludedSignals returns the selector's explicitly excluded semantic signal identities.
func (selector signalSelectorDefinition) ExcludedSignals() []Signal {
	return append([]Signal(nil), selector.excludedSignals...)
}

func (selector signalSelectorDefinition) Origins() []SignalOrigin {
	return append([]SignalOrigin(nil), selector.origins...)
}

func (selector signalSelectorDefinition) ExcludedOrigins() []SignalOrigin {
	return append([]SignalOrigin(nil), selector.excludedOrigins...)
}

func (selector signalSelectorDefinition) ExcludedSender() string { return selector.excludedSender }

func (selector signalSelectorDefinition) ExcludedSenderTarget() (signalTargetDefinition, bool) {
	return selector.excludedSenderTarget, selector.excludedSenderTarget.MatchesSender() != nil
}

func (selector signalSelectorDefinition) Phases() []SignalPhase {
	return append([]SignalPhase(nil), selector.phases...)
}

func validSignalSet(values []Signal) bool {
	seen := make(map[Signal]struct{}, len(values))
	for _, signal := range values {
		if !validSignal(signal) {
			return false
		}
		if _, duplicate := seen[signal]; duplicate {
			return false
		}
		seen[signal] = struct{}{}
	}
	return true
}

type signalOccurrenceRange struct{ minimum, maximum int }

func acceptSignalOccurrenceRange(minimum, maximum int) signalOccurrenceRange {
	if minimum < 1 || maximum < 0 || maximum != 0 && maximum < minimum {
		panic("semantic: invalid signal occurrence range")
	}
	return signalOccurrenceRange{
		minimum: minimum,
		maximum: maximum,
	}
}

func (r signalOccurrenceRange) Valid() bool {
	return r.minimum >= 1 && r.maximum >= 0 && (r.maximum == 0 || r.maximum >= r.minimum)
}

func (r signalOccurrenceRange) Minimum() int { return r.minimum }

func (r signalOccurrenceRange) Maximum() int { return r.maximum }

type (
	signalHandlerCallback             func(context.Context, any, any) error
	signalTargetDeliverCallback       func(context.Context, any, any) error
	signalTargetMatchesSenderCallback func(context.Context, any, SignalSender) (bool, error)
)

type signalTargetDefinition struct {
	name          string
	deliver       signalTargetDeliverCallback
	matchesSender signalTargetMatchesSenderCallback
}

func acceptSignalTargetDefinition(name string, deliver signalTargetDeliverCallback, matches signalTargetMatchesSenderCallback) signalTargetDefinition {
	definition := signalTargetDefinition{
		name:          name,
		deliver:       deliver,
		matchesSender: matches,
	}

	return definition
}

func (t signalTargetDefinition) ValidForDelivery() bool { return t.name != "" && t.deliver != nil }

func (t signalTargetDefinition) Name() string { return t.name }

func (t signalTargetDefinition) Deliver() signalTargetDeliverCallback { return t.deliver }

func (t signalTargetDefinition) MatchesSender() signalTargetMatchesSenderCallback {
	return t.matchesSender
}

type signalActionKind uint8

const (
	signalActionKindUnknown signalActionKind = iota
	signalActionKindIgnore
	signalActionKindCancel
	signalActionKindHandle
	signalActionKindForward
	signalActionKindSend
	signalActionKindAbort
	signalActionKindReraise
	signalActionKindEmit
)

func (kind signalActionKind) Valid() bool {
	return kind >= signalActionKindIgnore && kind <= signalActionKindEmit
}

type signalActionDefinition struct {
	kind        signalActionKind
	name        string
	handler     signalHandlerCallback
	target      signalTargetDefinition
	signal      Signal
	signalValue ValueID
	endpoint    SignalEndpointID
	delayed     bool
	delay       time.Duration
	delayValue  ValueID
	exitCode    int
	sensitive   bool
}

type signalActionDelivery struct {
	Signal Signal
	Value  ValueID
}

type signalActionDelay struct {
	Present  bool
	Duration time.Duration
	Value    ValueID
}

type signalActionCandidate struct {
	Kind      signalActionKind
	Name      string
	Handler   signalHandlerCallback
	Target    signalTargetDefinition
	Delivery  signalActionDelivery
	Endpoint  SignalEndpointID
	Delay     signalActionDelay
	ExitCode  int
	Sensitive bool
}

func acceptSignalActionDefinition(candidate signalActionCandidate) signalActionDefinition {
	definition := signalActionDefinition{
		kind:        candidate.Kind,
		name:        candidate.Name,
		handler:     candidate.Handler,
		target:      candidate.Target,
		signal:      candidate.Delivery.Signal,
		signalValue: candidate.Delivery.Value,
		endpoint:    candidate.Endpoint,
		delayed:     candidate.Delay.Present,
		delay:       candidate.Delay.Duration,
		delayValue:  candidate.Delay.Value,
		exitCode:    candidate.ExitCode,
		sensitive:   candidate.Sensitive,
	}

	return definition
}

func (a signalActionDefinition) Valid() bool {
	if !a.kind.Valid() {
		return false
	}
	if a.delayed {
		if (a.delayValue.present() && a.delay != 0) || (!a.delayValue.present() && a.delay <= 0) {
			return false
		}
	} else if a.delay != 0 || a.delayValue.present() {
		return false
	}
	switch a.kind {
	case signalActionKindIgnore, signalActionKindCancel, signalActionKindReraise:
		return a.name == "" && a.handler == nil && !a.target.ValidForDelivery() && !a.signal.valid() &&
			!a.signalValue.present() &&
			!a.endpoint.present()
	case signalActionKindHandle:
		return a.name != "" && a.handler != nil && !a.target.ValidForDelivery() && !a.signal.valid() &&
			!a.signalValue.present() &&
			!a.endpoint.present()
	case signalActionKindForward:
		return a.target.ValidForDelivery() && a.name == a.target.Name() && a.handler == nil && !a.signal.valid() &&
			!a.signalValue.present() &&
			!a.endpoint.present()
	case signalActionKindSend:
		return a.target.ValidForDelivery() && a.name == a.target.Name() && a.handler == nil && !a.endpoint.present() &&
			((a.signal.valid() && validSignal(a.signal) && !a.signalValue.present()) ||
				(!a.signal.valid() && a.signalValue.present()))
	case signalActionKindAbort:
		return a.name == "" && a.handler == nil && !a.target.ValidForDelivery() && !a.signal.valid() &&
			!a.signalValue.present() &&
			!a.endpoint.present()
	case signalActionKindEmit:
		return !a.delayed && a.name == "" && a.handler == nil && !a.target.ValidForDelivery() && !a.signal.valid() &&
			!a.signalValue.present() &&
			a.endpoint.present()
	default:
		return false
	}
}

func (a signalActionDefinition) Kind() signalActionKind { return a.kind }

func (a signalActionDefinition) Name() string { return a.name }

func (a signalActionDefinition) Handler() signalHandlerCallback { return a.handler }

func (a signalActionDefinition) Target() signalTargetDefinition { return a.target }

// Signal returns the signal used by a send action.
func (a signalActionDefinition) Signal() Signal { return a.signal }

func (a signalActionDefinition) SignalValue() (ValueID, bool) {
	return a.signalValue, a.signalValue.present()
}

func (a signalActionDefinition) Endpoint() (SignalEndpointID, bool) {
	return a.endpoint, a.endpoint.present()
}

func (a signalActionDefinition) Delayed() bool { return a.delayed }

func (a signalActionDefinition) Delay() time.Duration { return a.delay }

func (a signalActionDefinition) DelayValue() (ValueID, bool) {
	return a.delayValue, a.delayValue.present()
}

func (a signalActionDefinition) ExitCode() int { return a.exitCode }

func (a signalActionDefinition) Sensitive() bool { return a.sensitive }

type signalPolicyDefinition struct {
	selector         signalSelectorDefinition
	occurrence       signalOccurrenceRange
	aggregation      SignalAggregation
	deferred         bool
	continueMatching bool
	hint             string
	actions          []signalActionDefinition
	activation       condition
	hasActivation    bool
	requiredSource   SignalSourceCapabilities
	requiredHost     SignalHostCapabilities
}

type semanticSignalPolicyMatch struct {
	Selector         signalSelectorDefinition
	Occurrence       signalOccurrenceRange
	Aggregation      SignalAggregation
	Deferred         bool
	ContinueMatching bool
	Hint             string
}

type signalPolicyActivation struct {
	Condition condition
	Present   bool
}

type signalPolicyRequirements struct {
	Source SignalSourceCapabilities
	Host   SignalHostCapabilities
}

type signalPolicyCandidate struct {
	Match        semanticSignalPolicyMatch
	Actions      []signalActionDefinition
	Activation   signalPolicyActivation
	Requirements signalPolicyRequirements
}

func acceptSignalPolicyDefinition(candidate signalPolicyCandidate) signalPolicyDefinition {
	definition := signalPolicyDefinition{
		selector:         candidate.Match.Selector,
		occurrence:       candidate.Match.Occurrence,
		aggregation:      candidate.Match.Aggregation,
		deferred:         candidate.Match.Deferred,
		continueMatching: candidate.Match.ContinueMatching,
		hint:             candidate.Match.Hint,
		actions:          append([]signalActionDefinition(nil), candidate.Actions...),
		activation:       candidate.Activation.Condition,
		hasActivation:    candidate.Activation.Present,
		requiredSource:   candidate.Requirements.Source,
		requiredHost:     candidate.Requirements.Host,
	}

	return definition
}

func (d signalPolicyDefinition) Valid() bool {
	if !d.selector.Valid() || !d.occurrence.Valid() || !d.aggregation.valid() || len(d.actions) == 0 ||
		d.requiredSource&^signalSourceCapabilityMask != 0 || d.requiredHost&^signalHostCapabilityMask != 0 {
		return false
	}
	if d.hasActivation && !d.activation.Valid() {
		return false
	}
	for _, action := range d.actions {
		if !action.Valid() {
			return false
		}
	}
	return true
}

func (d signalPolicyDefinition) Selector() signalSelectorDefinition { return d.selector }

func (d signalPolicyDefinition) Occurrence() signalOccurrenceRange { return d.occurrence }

func (d signalPolicyDefinition) Aggregation() SignalAggregation { return d.aggregation }

func (d signalPolicyDefinition) Deferred() bool { return d.deferred }

func (d signalPolicyDefinition) ContinueMatching() bool { return d.continueMatching }

func (d signalPolicyDefinition) Hint() string { return d.hint }

func (d signalPolicyDefinition) Actions() []signalActionDefinition {
	return append([]signalActionDefinition(nil), d.actions...)
}

func (d signalPolicyDefinition) Activation() (condition, bool) { return d.activation, d.hasActivation }

func (d signalPolicyDefinition) RequiredSourceCapabilities() SignalSourceCapabilities {
	return d.requiredSource
}

func (d signalPolicyDefinition) RequiredHostCapabilities() SignalHostCapabilities {
	return d.requiredHost
}

type signalPolicyScope struct {
	command        CommandID
	profile        GrammarProfileID
	clearInherited bool
	position       declarationPosition
}

func acceptSignalPolicyScope(command CommandID, profile GrammarProfileID, clearInherited bool, position declarationPosition) signalPolicyScope {
	mustCommand(command)
	result := signalPolicyScope{
		command:        command,
		profile:        profile,
		clearInherited: clearInherited,
		position:       position,
	}

	return result
}

func (s signalPolicyScope) Command() CommandID { return s.command }

func (s signalPolicyScope) Profile() GrammarProfileID { return s.profile }

func (s signalPolicyScope) ClearInherited() bool { return s.clearInherited }

func (s signalPolicyScope) Position() declarationPosition { return s.position }
