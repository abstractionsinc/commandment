package cli

import (
	"cmp"
	"regexp"
	"slices"
)

// conditionOutcome is operation-local evaluation state. Blocked means the
// condition's truth cannot be selected because an upstream destination failed
// or was blocked; unsupported means this candidate cannot preserve an exact
// typed leaf relationship from the compiled application view.
type conditionOutcome uint8

const (
	conditionOutcomeUnknown conditionOutcome = iota
	conditionOutcomeSatisfied
	conditionOutcomeUnsatisfied
	conditionOutcomeBlocked
	conditionOutcomeUnsupported
)

func evaluateCondition(condition condition, values valueAnalysis) conditionOutcome {
	if !condition.Valid() || !condition.CanEvaluate() {
		return conditionOutcomeUnsupported
	}
	switch condition.Kind() {
	case conditionKindResolved:
		return destinationConditionState(condition.Destination(), values, nil)
	case conditionKindEqual,
		conditionKindNotEqual,
		conditionKindIn,
		conditionKindNotIn,
		conditionKindGreaterThan,
		conditionKindAtLeast,
		conditionKindLessThan,
		conditionKindAtMost,
		conditionKindMatches:
		return destinationConditionState(condition.Destination(), values, func(value any) bool {
			matched, ok := condition.TestResolvedValue(value)
			return ok && matched
		})
	case conditionKindAll:
		blocked := false
		for _, child := range condition.Children() {
			outcome := evaluateCondition(child, values)
			switch outcome {
			case conditionOutcomeUnsatisfied:
				return conditionOutcomeUnsatisfied
			case conditionOutcomeBlocked:
				blocked = true
			case conditionOutcomeUnsupported:
				return conditionOutcomeUnsupported
			case conditionOutcomeSatisfied:
			default:
				return conditionOutcomeUnsupported
			}
		}
		if blocked {
			return conditionOutcomeBlocked
		}
		return conditionOutcomeSatisfied
	case conditionKindAny:
		blocked := false
		for _, child := range condition.Children() {
			outcome := evaluateCondition(child, values)
			switch outcome {
			case conditionOutcomeSatisfied:
				return conditionOutcomeSatisfied
			case conditionOutcomeBlocked:
				blocked = true
			case conditionOutcomeUnsupported:
				return conditionOutcomeUnsupported
			case conditionOutcomeUnsatisfied:
			default:
				return conditionOutcomeUnsupported
			}
		}
		if blocked {
			return conditionOutcomeBlocked
		}
		return conditionOutcomeUnsatisfied
	case conditionKindNot:
		children := condition.Children()
		if len(children) != 1 {
			return conditionOutcomeUnsupported
		}
		switch evaluateCondition(children[0], values) {
		case conditionOutcomeSatisfied:
			return conditionOutcomeUnsatisfied
		case conditionOutcomeUnsatisfied:
			return conditionOutcomeSatisfied
		case conditionOutcomeBlocked:
			return conditionOutcomeBlocked
		default:
			return conditionOutcomeUnsupported
		}
	default:
		return conditionOutcomeUnsupported
	}
}

func evaluatePolicyCondition(condition policyCondition, values valueAnalysis) conditionOutcome {
	if !condition.Valid() || !condition.CanEvaluate() {
		return conditionOutcomeUnsupported
	}
	switch condition.Kind() {
	case policyConditionResolved:
		return destinationConditionState(condition.Destination(), values, nil)
	case policyConditionContributedBy:
		resolution, ok := values.Destinations[condition.Destination()]
		if !ok {
			return conditionOutcomeUnsupported
		}
		switch resolution.State {
		case destinationStateResolved:
			accepted := make(map[ValueProducerKind]struct{}, len(condition.ProducerKinds()))
			for _, producerKind := range condition.ProducerKinds() {
				accepted[producerKind] = struct{}{}
			}
			for _, candidateID := range resolution.Contributors {
				candidate, exists := values.Candidates[candidateID]
				if !exists {
					return conditionOutcomeUnsupported
				}
				if _, matches := accepted[candidate.Meta.ProducerKind]; matches {
					return conditionOutcomeSatisfied
				}
			}
			return conditionOutcomeUnsatisfied
		case destinationStateFailed, destinationStateBlocked:
			return conditionOutcomeBlocked
		case destinationStateAbsent, destinationStateInactive:
			return conditionOutcomeUnsatisfied
		default:
			return conditionOutcomeUnsupported
		}
	case policyConditionEqual,
		policyConditionNotEqual,
		policyConditionIn,
		policyConditionNotIn,
		policyConditionGreaterThan,
		policyConditionAtLeast,
		policyConditionLessThan,
		policyConditionAtMost,
		policyConditionMatches:
		return destinationPolicyConditionState(condition.Destination(), values, condition)
	case policyConditionAll:
		blocked := false
		for _, child := range condition.Children() {
			outcome := evaluatePolicyCondition(child, values)
			switch outcome {
			case conditionOutcomeUnsatisfied:
				return conditionOutcomeUnsatisfied
			case conditionOutcomeBlocked:
				blocked = true
			case conditionOutcomeUnsupported:
				return conditionOutcomeUnsupported
			case conditionOutcomeSatisfied:
			default:
				return conditionOutcomeUnsupported
			}
		}
		if blocked {
			return conditionOutcomeBlocked
		}
		return conditionOutcomeSatisfied
	case policyConditionAny:
		blocked := false
		for _, child := range condition.Children() {
			outcome := evaluatePolicyCondition(child, values)
			switch outcome {
			case conditionOutcomeSatisfied:
				return conditionOutcomeSatisfied
			case conditionOutcomeBlocked:
				blocked = true
			case conditionOutcomeUnsupported:
				return conditionOutcomeUnsupported
			case conditionOutcomeUnsatisfied:
			default:
				return conditionOutcomeUnsupported
			}
		}
		if blocked {
			return conditionOutcomeBlocked
		}
		return conditionOutcomeUnsatisfied
	case policyConditionNot:
		children := condition.Children()
		if len(children) != 1 {
			return conditionOutcomeUnsupported
		}
		switch evaluatePolicyCondition(children[0], values) {
		case conditionOutcomeSatisfied:
			return conditionOutcomeUnsatisfied
		case conditionOutcomeUnsatisfied:
			return conditionOutcomeSatisfied
		case conditionOutcomeBlocked:
			return conditionOutcomeBlocked
		default:
			return conditionOutcomeUnsupported
		}
	default:
		return conditionOutcomeUnsupported
	}
}

func destinationConditionState(destination ValueID, values valueAnalysis, predicate func(any) bool) conditionOutcome {
	resolution, ok := values.Destinations[destination]
	if !ok {
		return conditionOutcomeUnsupported
	}
	switch resolution.State {
	case destinationStateResolved:
		if predicate == nil || predicate(resolution.Value) {
			return conditionOutcomeSatisfied
		}
		return conditionOutcomeUnsatisfied
	case destinationStateAbsent, destinationStateInactive:
		return conditionOutcomeUnsatisfied
	case destinationStateFailed, destinationStateBlocked:
		return conditionOutcomeBlocked
	default:
		return conditionOutcomeUnsupported
	}
}

func destinationPolicyConditionState(destination ValueID, values valueAnalysis, condition policyCondition) conditionOutcome {
	return destinationConditionState(destination, values, func(value any) bool {
		matched, ok := condition.TestResolvedValue(value)
		return ok && matched
	})
}

// Condition is an inert immutable structural expression over declared semantic state. Constructing
// or combining it commits no policy; ownership, dependencies, and phase legality are validated when
// it is consumed.
type Condition struct {
	state *conditionState
}

type conditionKind uint8

const (
	conditionUnknown conditionKind = iota
	conditionResolved
	conditionEqual
	conditionNotEqual
	conditionIn
	conditionNotIn
	conditionGreaterThan
	conditionAtLeast
	conditionLessThan
	conditionAtMost
	conditionMatches
	conditionAll
	conditionAny
	conditionNot
)

type conditionState struct {
	kind        conditionKind
	owner       *draftIdentity
	destination destinationIdentity
	operands    []any
	pattern     string
	children    []*conditionState
	predicate   func(any) bool
	valid       bool
}

// RuleCondition is an activation expression for Rule authoring.
type RuleCondition interface {
	ruleCondition()
}

func (Condition) ruleCondition() {}

type validationRuleCondition struct{ state *ruleConditionState }

func (validationRuleCondition) ruleCondition() {}

// Conditions constructs inert structural conditions for its owning declaration scope. The surface
// is callback-scoped: its zero value has no valid owner, and retained, concurrent, post-callback,
// or cross-scope use is builder lifecycle misuse.
type Conditions struct {
	session *declarationSession
	profile grammarProfileCaseRef
}

// Resolved returns a condition that is true when destination has resolved semantic state.
func (conditions Conditions) Resolved[C, T any](destination destinationLike[C, T]) Condition {
	return stageConditionLeaf(conditions.session, conditionResolved, destination)
}

// Equal returns a condition comparing destination with value for equality.
func (conditions Conditions) Equal[C any, T comparable](destination destinationLike[C, T], value T) Condition {
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionEqual, destination, value),
		func(observed T) bool { return observed == value },
	)
}

// NotEqual returns a condition comparing destination with value for inequality.
func (conditions Conditions) NotEqual[C any, T comparable](destination destinationLike[C, T], value T) Condition {
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionNotEqual, destination, value),
		func(observed T) bool { return observed != value },
	)
}

// In returns a condition that matches when destination equals one of values.
func (conditions Conditions) In[C any, T comparable](destination destinationLike[C, T], values ...T) Condition {
	owned := append([]T(nil), values...)
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionIn, destination, owned...),
		func(observed T) bool {
			return slices.Contains(owned, observed)
		},
	)
}

// NotIn returns a condition that matches when destination equals none of values.
func (conditions Conditions) NotIn[C any, T comparable](destination destinationLike[C, T], values ...T) Condition {
	owned := append([]T(nil), values...)
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionNotIn, destination, owned...),
		func(observed T) bool {
			return !slices.Contains(owned, observed)
		},
	)
}

// GreaterThan returns an ordered condition requiring destination to be greater than value.
func (conditions Conditions) GreaterThan[C any, T cmp.Ordered](destination destinationLike[C, T], value T) Condition {
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionGreaterThan, destination, value),
		func(observed T) bool { return observed > value },
	)
}

// AtLeast returns an ordered condition requiring destination to be greater than or equal to value.
func (conditions Conditions) AtLeast[C any, T cmp.Ordered](destination destinationLike[C, T], value T) Condition {
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionAtLeast, destination, value),
		func(observed T) bool { return observed >= value },
	)
}

// LessThan returns an ordered condition requiring destination to be less than value.
func (conditions Conditions) LessThan[C any, T cmp.Ordered](destination destinationLike[C, T], value T) Condition {
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionLessThan, destination, value),
		func(observed T) bool { return observed < value },
	)
}

// AtMost returns an ordered condition requiring destination to be less than or equal to value.
func (conditions Conditions) AtMost[C any, T cmp.Ordered](destination destinationLike[C, T], value T) Condition {
	return conditionWithPredicate(
		stageConditionLeaf(conditions.session, conditionAtMost, destination, value),
		func(observed T) bool { return observed <= value },
	)
}

// Matches returns a string condition using Go regular-expression semantics; an invalid pattern
// makes the declaration invalid when consumed.
func (conditions Conditions) Matches[C any](destination destinationLike[C, string], pattern string) Condition {
	condition := stageConditionLeaf(conditions.session, conditionMatches, destination)
	if condition.state != nil {
		condition.state.pattern = pattern
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			condition.state.valid = false
		} else {
			condition.state.predicate = func(value any) bool {
				observed, ok := value.(string)
				return ok && compiled.MatchString(observed)
			}
		}
	}
	return condition
}

// All returns the logical conjunction of the supplied conditions.
func (conditions Conditions) All(first Condition, rest ...Condition) Condition {
	return stageConditionGroup(conditions.session, conditionAll, first, rest...)
}

// Any returns the logical disjunction of the supplied conditions.
func (conditions Conditions) Any(first Condition, rest ...Condition) Condition {
	return stageConditionGroup(conditions.session, conditionAny, first, rest...)
}

// Not returns the logical negation of condition.
func (conditions Conditions) Not(condition Condition) Condition {
	return stageConditionGroup(conditions.session, conditionNot, condition)
}

func stageConditionLeaf[C, T any](owner *declarationSession, kind conditionKind, destination destinationLike[C, T], operands ...T) Condition {
	state := &conditionState{
		kind: kind,
	}
	if owner == nil || destination == nil {
		condition := Condition{
			state: state,
		}
		return condition
	}
	rawDestination := destination.destinationHandle()
	owner.useCondition(func(draft *declarationDraft) {
		state.owner = draft.identity
		state.destination, state.valid = captureAuthoringDestinationIdentity(draft, rawDestination)
	})
	state.operands = make([]any, len(operands))
	for index := range operands {
		state.operands[index] = operands[index]
	}
	condition := Condition{
		state: state,
	}
	return condition
}

func conditionWithPredicate[T any](condition Condition, predicate func(T) bool) Condition {
	if condition.state != nil && predicate != nil {
		condition.state.predicate = func(value any) bool {
			typed, ok := value.(T)
			return ok && predicate(typed)
		}
	}
	return condition
}

func stageConditionGroup(owner *declarationSession, kind conditionKind, first Condition, rest ...Condition) Condition {
	state := &conditionState{
		kind: kind,
	}
	if owner != nil {
		owner.useCondition(func(draft *declarationDraft) {
			state.owner = draft.identity
		})
	}
	state.valid = state.owner != nil
	conditions := make([]Condition, 0, len(rest)+1)
	conditions = append(conditions, first)
	conditions = append(conditions, rest...)
	if kind == conditionNot && len(conditions) != 1 {
		state.valid = false
	}
	state.children = make([]*conditionState, len(conditions))
	for index, condition := range conditions {
		state.children[index] = condition.state
		if condition.state == nil || !condition.state.valid || condition.state.owner != state.owner {
			state.valid = false
		}
	}
	returnedCondition := Condition{
		state: state,
	}
	return returnedCondition
}

type authoringConditionTarget interface {
	authoringConditionTarget()
}

type authoringCommandConditionTarget struct {
	reference commandRef
}

type authoringFlagConditionTarget struct {
	reference flagRef
}

type authoringArgumentFormConditionTarget struct {
	reference argumentFormDraftRef
}

type authoringArgumentTermConditionTarget struct {
	reference argumentTermDraftRef
}

func (authoringCommandConditionTarget) authoringConditionTarget() {}

func (authoringFlagConditionTarget) authoringConditionTarget() {}

func (authoringArgumentFormConditionTarget) authoringConditionTarget() {}

func (authoringArgumentTermConditionTarget) authoringConditionTarget() {}

func stageConditionPolicy(session *declarationSession, profile grammarProfileCaseRef, target authoringConditionTarget, effect ConditionEffect, condition Condition) {
	if session == nil {
		panicBuilderMisuse()
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		valid := condition.state != nil && condition.state.valid && condition.state.owner == draft.identity
		if valid {
			switch target := target.(type) {
			case authoringCommandConditionTarget:
				valid = target.reference.ownedBy(draft.identity)
			case authoringFlagConditionTarget:
				valid = target.reference.ownedBy(draft.identity)
			case authoringArgumentFormConditionTarget:
				valid = target.reference.ownedBy(draft.identity)
			case authoringArgumentTermConditionTarget:
				valid = target.reference.ownedBy(draft.identity)
			default:
				valid = false
			}
		}

		if !profile.valid() {
			switch target := target.(type) {
			case authoringFlagConditionTarget:
				if record, ok := draft.flags.resolve(draft, target.reference); ok {
					profile = record.profile
				}
			case authoringArgumentFormConditionTarget:
				if record, ok := draft.argumentForms.resolve(draft, target.reference); ok {
					profile = record.profile
				}
			case authoringArgumentTermConditionTarget:
				if record, ok := draft.argumentTerms.resolve(draft, target.reference); ok {
					profile = record.profile
				}
			}
		}

		record := conditionPolicyDraft{
			position:  draft.nextPosition(),
			command:   command,
			profile:   profile,
			target:    target,
			effect:    effect,
			condition: condition,
			valid:     valid,
		}
		draft.conditionPolicies.append(draft, record)
	})
}

func stageAvailabilityCondition(session *declarationSession, profile grammarProfileCaseRef, target authoringConditionTarget, condition Condition) {
	stageConditionPolicy(session, profile, target, ConditionEffectAvailability, condition)
}

func stageVisibilityCondition(session *declarationSession, profile grammarProfileCaseRef, target authoringConditionTarget, condition Condition) {
	stageConditionPolicy(session, profile, target, ConditionEffectVisibility, condition)
}

func stageStaticConditions(input declarationPlan, source *declarationDraft, draft *declarationCandidate, state *staticAcceptanceState) error {
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordCommand {
			continue
		}
		value := source.commands.entries[record.familyIndex].value
		if value.defaults.state != commandDefaultConditional {
			continue
		}
		defaultRef := state.slots[index].defaultRef
		if !defaultRef.ID().present() {
			panic("cli: conditional command default identity is unavailable")
		}
		condition, valid := prepareCondition(value.defaults.condition, state)
		if !valid {
			return errStaticDeclarationRejected
		}
		draft.DefaultCondition(defaultRef, condition, input.positionFor(index))
	}
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordConditionPolicy {
			continue
		}
		policy := source.conditionPolicies.entries[record.familyIndex].value
		condition, valid := prepareCondition(policy.condition, state)
		if !valid {
			return errStaticDeclarationRejected
		}
		position := input.positionFor(index)
		conditionPolicyIndex := draft.ConditionPolicyCount()
		targetIndex, indexed := input.conditionTargetIndex(policy.target)
		if !indexed {
			return errStaticDeclarationRejected
		}
		switch policy.target.(type) {
		case authoringCommandConditionTarget:
			draft.CommandCondition(state.mustCommand(targetIndex), policy.effect, condition, position)
		case authoringFlagConditionTarget:
			slot := state.mustSlot(targetIndex)
			if !slot.flag.ID().present() {
				panic("cli: condition target flag identity is unavailable")
			}
			draft.FlagCondition(slot.flag, policy.effect, condition, position)
		case authoringArgumentFormConditionTarget:
			slot := state.mustSlot(targetIndex)
			if !slot.argumentForm.ID().present() {
				panic("cli: condition target argument-form identity is unavailable")
			}
			draft.ArgumentFormCondition(slot.argumentForm, policy.effect, condition, position)
		case authoringArgumentTermConditionTarget:
			slot := state.mustSlot(targetIndex)
			if !slot.argumentTerm.ID().present() {
				panic("cli: condition target argument-term identity is unavailable")
			}
			draft.ArgumentTermCondition(slot.argumentTerm, policy.effect, condition, position)
		default:
			return errStaticDeclarationRejected
		}
		if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
			draft.ConditionPolicyProfile(conditionPolicyIndex, profile, position)
		}
	}
	return nil
}

// semanticConditionKind identifies one immutable structured condition expression kind.
// Public projection owns its own detached ConditionExpression representation.
type semanticConditionKind uint8

const (
	conditionKindUnknown semanticConditionKind = iota
	conditionKindResolved
	conditionKindEqual
	conditionKindNotEqual
	conditionKindIn
	conditionKindNotIn
	conditionKindGreaterThan
	conditionKindAtLeast
	conditionKindLessThan
	conditionKindAtMost
	conditionKindMatches
	conditionKindAll
	conditionKindAny
	conditionKindNot
)

// condition is an immutable expression over exact semantic destination identity.
// Operands are already snapshotted by the typed public authoring boundary;
// immutable declarations do not rediscover their type or destination.
type condition struct {
	kind        semanticConditionKind
	destination ValueID
	operands    []any
	pattern     string
	children    []condition
	predicate   func(any) bool
}

// resolvedConditionInput is the payload of one resolved-state condition.
type resolvedConditionInput struct {
	Destination ValueID
	Predicate   func(any) bool
}

// comparisonConditionInput is the payload shared by single-operand comparison conditions.
type comparisonConditionInput struct {
	Kind        semanticConditionKind
	Destination ValueID
	Operands    []any
	Predicate   func(any) bool
}

// membershipConditionInput is the payload shared by set-membership conditions.
// Operands remain a slice because empty membership sets are valid.
type membershipConditionInput struct {
	Kind        semanticConditionKind
	Destination ValueID
	Operands    []any
	Predicate   func(any) bool
}

// patternConditionInput is the payload of a pattern-match condition.
type patternConditionInput struct {
	Destination ValueID
	Pattern     string
	Predicate   func(any) bool
}

// conditionGroupInput is the payload shared by all/any composition variants.
type conditionGroupInput struct {
	Kind      semanticConditionKind
	Children  []condition
	Predicate func(any) bool
}

// negatedConditionInput is the payload of a not condition. Children remains a
// slice so construction preserves the existing validation/panic boundary.
type negatedConditionInput struct {
	Children  []condition
	Predicate func(any) bool
}

func acceptResolvedCondition(input resolvedConditionInput) condition {
	conditionValue := condition{
		kind:        conditionKindResolved,
		destination: input.Destination,
		predicate:   input.Predicate,
	}
	return mustCondition(conditionValue)
}

func acceptComparisonCondition(input comparisonConditionInput) condition {
	conditionValue := condition{
		kind:        input.Kind,
		destination: input.Destination,
		operands:    input.Operands,
		predicate:   input.Predicate,
	}
	return mustCondition(conditionValue)
}

func acceptMembershipCondition(input membershipConditionInput) condition {
	conditionValue := condition{
		kind:        input.Kind,
		destination: input.Destination,
		operands:    input.Operands,
		predicate:   input.Predicate,
	}
	return mustCondition(conditionValue)
}

func acceptPatternCondition(input patternConditionInput) condition {
	conditionValue := condition{
		kind:        conditionKindMatches,
		destination: input.Destination,
		pattern:     input.Pattern,
		predicate:   input.Predicate,
	}
	return mustCondition(conditionValue)
}

func acceptConditionGroup(input conditionGroupInput) condition {
	conditionValue := condition{
		kind:      input.Kind,
		children:  input.Children,
		predicate: input.Predicate,
	}
	return mustCondition(conditionValue)
}

func acceptNegatedCondition(input negatedConditionInput) condition {
	conditionValue := condition{
		kind:      conditionKindNot,
		children:  input.Children,
		predicate: input.Predicate,
	}
	return mustCondition(conditionValue)
}

func mustCondition(value condition) condition {
	value.operands = append([]any(nil), value.operands...)
	value.children = append([]condition(nil), value.children...)
	if !value.Valid() {
		panic("semantic: invalid condition")
	}
	return value
}

func (value condition) Valid() bool {
	switch value.kind {
	case conditionKindResolved:
		return value.destination.present() && len(value.operands) == 0 && value.pattern == "" &&
			len(value.children) == 0
	case conditionKindEqual,
		conditionKindNotEqual,
		conditionKindGreaterThan,
		conditionKindAtLeast,
		conditionKindLessThan,
		conditionKindAtMost:
		return value.destination.present() && len(value.operands) == 1 && value.pattern == "" &&
			len(value.children) == 0
	case conditionKindIn, conditionKindNotIn:
		return value.destination.present() && value.pattern == "" && len(value.children) == 0
	case conditionKindMatches:
		return value.destination.present() && len(value.operands) == 0 && len(value.children) == 0
	case conditionKindAll, conditionKindAny:
		if value.destination.present() || len(value.operands) != 0 || value.pattern != "" ||
			len(value.children) == 0 {
			return false
		}
		for _, child := range value.children {
			if !child.Valid() {
				return false
			}
		}
		return true
	case conditionKindNot:
		return !value.destination.present() && len(value.operands) == 0 && value.pattern == "" &&
			len(value.children) == 1 &&
			value.children[0].Valid()
	default:
		return false
	}
}

func (value condition) Kind() semanticConditionKind { return value.kind }

func (value condition) Destination() ValueID { return value.destination }

func (value condition) Operands() []any { return append([]any(nil), value.operands...) }

func (value condition) Pattern() string { return value.pattern }

func (value condition) Children() []condition {
	return append([]condition(nil), value.children...)
}

// CanEvaluate reports whether operation-time evaluation can preserve the exact
// typed semantics of every leaf. Resolved leaves and logical composition need
// no erased value predicate; typed relational leaves do.
func (value condition) CanEvaluate() bool {
	if !value.Valid() {
		return false
	}
	switch value.kind {
	case conditionKindResolved:
		return true
	case conditionKindEqual, conditionKindNotEqual, conditionKindIn, conditionKindNotIn,
		conditionKindGreaterThan, conditionKindAtLeast, conditionKindLessThan, conditionKindAtMost, conditionKindMatches:
		return value.predicate != nil
	case conditionKindAll, conditionKindAny, conditionKindNot:
		for _, child := range value.children {
			if !child.CanEvaluate() {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// TestResolvedValue applies one typed leaf predicate to a successfully
// resolved value. It is valid only for non-Resolved leaf kinds.
func (value condition) TestResolvedValue(resolved any) (bool, bool) {
	if !value.CanEvaluate() || value.predicate == nil {
		return false, false
	}
	return value.predicate(resolved), true
}

func (value condition) Dependencies() []ValueID {
	if !value.Valid() {
		return nil
	}
	seen := make(map[ValueID]struct{})
	var result []ValueID
	var visit func(condition)
	visit = func(current condition) {
		if current.destination.present() {
			if _, ok := seen[current.destination]; !ok {
				seen[current.destination] = struct{}{}
				result = append(result, current.destination)
			}
		}
		for _, child := range current.children {
			visit(child)
		}
	}
	visit(value)
	return result
}

type conditionTargetKind uint8

const (
	conditionTargetUnknown conditionTargetKind = iota
	conditionTargetCommand
	conditionTargetFlag
	conditionTargetArgumentForm
	conditionTargetArgumentTerm
)

// conditionTarget is a closed private exact-declaration coordinate. It does not
// create a public universal declaration identity or a string lookup authority.
type conditionTarget struct {
	kind    conditionTargetKind
	command CommandID
	flag    FlagID
	form    ArgumentFormID
	term    ArgumentTermID
}

func commandConditionTarget(id CommandID) conditionTarget {
	mustCommand(id)
	target := conditionTarget{
		kind:    conditionTargetCommand,
		command: id,
	}
	return target
}

func flagConditionTarget(id FlagID) conditionTarget {
	if !id.present() {
		panic("semantic: invalid flag condition target")
	}
	target := conditionTarget{
		kind: conditionTargetFlag,
		flag: id,
	}
	return target
}

func argumentFormConditionTarget(id ArgumentFormID) conditionTarget {
	if !id.present() {
		panic("semantic: invalid argument-form condition target")
	}
	target := conditionTarget{
		kind: conditionTargetArgumentForm,
		form: id,
	}
	return target
}

func argumentTermConditionTarget(id ArgumentTermID) conditionTarget {
	if !id.present() {
		panic("semantic: invalid argument-term condition target")
	}
	target := conditionTarget{
		kind: conditionTargetArgumentTerm,
		term: id,
	}
	return target
}

func (target conditionTarget) Valid() bool {
	switch target.kind {
	case conditionTargetCommand:
		return target.command.present()
	case conditionTargetFlag:
		return target.flag.present()
	case conditionTargetArgumentForm:
		return target.form.present()
	case conditionTargetArgumentTerm:
		return target.term.present()
	default:
		return false
	}
}

func (target conditionTarget) Kind() conditionTargetKind { return target.kind }

func (target conditionTarget) Command() CommandID { return target.command }

func (target conditionTarget) Flag() FlagID { return target.flag }

func (target conditionTarget) ArgumentForm() ArgumentFormID { return target.form }

func (target conditionTarget) ArgumentTerm() ArgumentTermID { return target.term }

type conditionPolicyRecord struct {
	target    conditionTarget
	effect    ConditionEffect
	condition condition
	position  declarationPosition
	profile   GrammarProfileID
}

func acceptConditionPolicyRecord(target conditionTarget, effect ConditionEffect, condition condition, position declarationPosition) conditionPolicyRecord {
	if !target.Valid() || !condition.Valid() ||
		(effect != ConditionEffectAvailability && effect != ConditionEffectVisibility) {
		panic("semantic: invalid condition policy")
	}
	record := conditionPolicyRecord{
		target:    target,
		effect:    effect,
		condition: condition,
		position:  position,
	}

	return record
}

// structuralAvailabilityPhaseLegal enforces the topology constraints that can
// be decided without looking at invocation token order. Runtime analysis may
// still report a blocked condition when an otherwise legal dependency has not
// resolved before the target's actual use frontier; that is distinct from a
// declaration-time dependency cycle.
func structuralAvailabilityPhaseLegal(record conditionPolicyRecord, commands map[CommandID]commandRecord, destinations map[ValueID]destinationRecord, flags map[FlagID]flagRecord, forms map[ArgumentFormID]argumentFormRecord, terms []argumentTermRecord) bool {
	if record.effect != ConditionEffectAvailability {
		return true
	}
	target := record.target
	owner, targetProfile, ok := conditionTargetProfile(target, commands, flags, forms, terms)
	if !ok {
		return false
	}

	// Positional form contributions do not exist until after form selection.
	// Any availability decision that is itself needed before that selection may
	// not depend on a destination whose value can be changed by a competing form.
	formProduces := func(destination ValueID) bool {
		for _, form := range forms {
			if form.command != owner || !grammarProfilesOverlap(targetProfile, form.meta.profile) {
				continue
			}
			for _, selection := range form.Selections() {
				if selection.destination == destination {
					return true
				}
			}
			for _, term := range terms {
				if term.form != form.meta.id {
					continue
				}
				if value, present := term.Destination(); present && value == destination {
					return true
				}
			}
		}
		return false
	}

	for _, dependency := range record.condition.Dependencies() {
		destination, exists := destinations[dependency]
		if !exists {
			return false
		}
		switch target.Kind() {
		case conditionTargetCommand:
			// The root is already selected when its availability is settled; a
			// child is not. Therefore a child can observe only destinations in
			// its already-selected parent/ancestor path. Root-local non-grammar
			// state (environment, Facts, static/computed producers) is legal.
			command := target.Command()
			if parent, hasParent := commands[command].Parent(); hasParent {
				if !commandSameOrAncestor(commands, destination.command, parent) {
					return false
				}
			} else if destination.command != command {
				return false
			}
			// Positional ownership is established only after command selection
			// closes, so it can never feed command availability.
			if positionalDestinationInCommand(dependency, destination.command, forms, terms) {
				return false
			}
			// Root availability also precedes flag recognition in its own scope.
			if _, hasParent := commands[command].Parent(); !hasParent &&
				flagDestinationInCommand(dependency, command, flags) {
				return false
			}
		case conditionTargetFlag:
			flag := flags[target.Flag()]
			if dependency == flag.destination || formProduces(dependency) {
				return false
			}
		case conditionTargetArgumentForm, conditionTargetArgumentTerm:
			if formProduces(dependency) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func flagDestinationInCommand(destination ValueID, command CommandID, flags map[FlagID]flagRecord) bool {
	for _, flag := range flags {
		if flag.command == command && flag.destination == destination {
			return true
		}
	}
	return false
}

func positionalDestinationInCommand(destination ValueID, command CommandID, forms map[ArgumentFormID]argumentFormRecord, terms []argumentTermRecord) bool {
	for _, form := range forms {
		if form.command != command {
			continue
		}
		for _, selection := range form.Selections() {
			if selection.destination == destination {
				return true
			}
		}
		for _, term := range terms {
			if term.form != form.meta.id {
				continue
			}
			if value, present := term.Destination(); present && value == destination {
				return true
			}
		}
	}
	return false
}
