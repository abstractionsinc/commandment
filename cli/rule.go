package cli

import (
	"cmp"
	"context"
	"regexp"
	"slices"
)

// RuleDeclaration identifies one staged custom Rule. Subject grants structured diagnostic and
// provenance authority; DependsOn declares callback data and scheduling dependencies. The zero
// value is invalid; copies identify the same Rule and are valid only in the owning declaration
// callback.
type RuleDeclaration[C any] struct {
	state *declarationSession
	rule  ruleDraftRef
}

// Subject declares that destination is a semantic subject of this custom Rule, authorizing
// structured diagnostic, provenance, and state observation for that destination.
func (declaration RuleDeclaration[C]) Subject[D, T any](destination destinationLike[D, T]) RuleDeclaration[C] {
	stageRuleSubjectDirective(declaration.state, declaration.rule, destinationHandleOf(destination))
	return declaration
}

// DependsOn declares a data dependency required to schedule and evaluate this custom Rule.
func (declaration RuleDeclaration[C]) DependsOn(dependency Dependency) RuleDeclaration[C] {
	stageRuleDirective(declaration.state, declaration.rule, ruleDirectiveDependency, dependency)
	return declaration
}

type ruleGroup struct {
	session *declarationSession
	rule    ruleDraftRef
}

// And appends dependency to this presence group while preserving the group's original Rule
// declaration order.
func (group ruleGroup) And(dependency Dependency) ruleGroup {
	if group.session != nil {
		stageRuleDirective(group.session, group.rule, ruleDirectiveGroupDependency, dependency)
	}
	return group
}

// ruleScope is the fluent built-in/custom Rule authoring surface carrying an optional activation
// expression; When and Unless return a new activated scope.
type ruleScope[C any] struct {
	session    *declarationSession
	profile    grammarProfileCaseRef
	activation RuleCondition
	Compare    ruleComparisons[C]
}

// ruleCapability extends the Rule authoring scope with activation-expression constructors.
type ruleCapability[C any] struct {
	ruleScope[C]
}

// ruleComparisons is the Compare surface for cross-destination Rule relationships under the same
// activation scope.
type ruleComparisons[C any] struct {
	session    *declarationSession
	profile    grammarProfileCaseRef
	activation RuleCondition
}

// ContributedBy returns a Rule activation condition matching destination provenance against the
// listed producer kinds.
func (rule ruleCapability[C]) ContributedBy[D, T any](destination destinationLike[D, T], first ValueProducerKind, rest ...ValueProducerKind) RuleCondition {
	return contributedByRuleCondition(rule.session, destination, first, rest...)
}

// All returns the conjunction of Rule activation conditions.
func (rule ruleCapability[C]) All(first RuleCondition, rest ...RuleCondition) RuleCondition {
	return combineRuleConditions(rule.session, ruleConditionAll, first, rest...)
}

// Any returns the disjunction of Rule activation conditions.
func (rule ruleCapability[C]) Any(first RuleCondition, rest ...RuleCondition) RuleCondition {
	return combineRuleConditions(rule.session, ruleConditionAny, first, rest...)
}

// Not returns the negation of a Rule activation condition.
func (rule ruleCapability[C]) Not(condition RuleCondition) RuleCondition {
	return combineRuleConditions(rule.session, ruleConditionNot, condition)
}

// When returns a Rule scope activated only when condition is satisfied.
func (scope ruleScope[C]) When(condition RuleCondition) ruleScope[C] {
	return activatedRuleScope(scope, condition, false)
}

// Unless returns a Rule scope activated only when condition is not satisfied.
func (scope ruleScope[C]) Unless(condition RuleCondition) ruleScope[C] {
	return activatedRuleScope(scope, condition, true)
}

// Check declares a custom Rule whose predicate yields one generic invalid violation on false, nil;
// a non-nil error is Rule evaluation failure.
func (scope ruleScope[C]) Check(code RuleCode, check RulePredicate[C]) RuleDeclaration[C] {
	return stageCustomRule(scope, code, authoringRuleCallbackCheck, eraseRuleCheck(check))
}

// Validate declares a custom Rule whose validator yields an ordered set of structured violations;
// any non-nil error discards returned violations.
func (scope ruleScope[C]) Validate(code RuleCode, validate RuleValidator[C]) RuleDeclaration[C] {
	return stageCustomRule(scope, code, authoringRuleCallbackValidate, eraseRuleValidate(validate))
}

// Require declares that destination must be semantically present; Go zero values remain present
// when resolution succeeded.
func (scope ruleScope[C]) Require[D, T any](destination destinationLike[D, T]) {
	stageBuiltinRule(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleRequire(),
		[]*destinationHandle{destinationHandleOf(destination)},
		CountRange{},
	)
}

// Forbid declares that destination must be semantically absent.
func (scope ruleScope[C]) Forbid[D, T any](destination destinationLike[D, T]) {
	stageBuiltinRule(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleForbid(),
		[]*destinationHandle{destinationHandleOf(destination)},
		CountRange{},
	)
}

// NotEmpty requires a resolved string destination to be nonempty.
func (scope ruleScope[C]) NotEmpty[D any](destination destinationLike[D, string]) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleNotEmpty(),
		[]*destinationHandle{destinationHandleOf(destination)},
		nil,
		CountRange{},
		"",
		unaryRulePredicate(func(value string) bool { return value != "" }),
	)
}

// In requires destination to equal one of the declared values.
func (scope ruleScope[C]) In[D any, T comparable](destination destinationLike[D, T], first T, rest ...T) {
	values := append([]any{first}, anySlice(rest)...)
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleIn(),
		[]*destinationHandle{destinationHandleOf(destination)},
		values,
		CountRange{},
		"",
		unaryRulePredicate(func(value T) bool {
			return slices.Contains(append([]T{first}, rest...), value)
		}),
	)
}

// NotIn requires destination to equal none of the declared values.
func (scope ruleScope[C]) NotIn[D any, T comparable](destination destinationLike[D, T], first T, rest ...T) {
	values := append([]any{first}, anySlice(rest)...)
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleNotIn(),
		[]*destinationHandle{destinationHandleOf(destination)},
		values,
		CountRange{},
		"",
		unaryRulePredicate(func(value T) bool {
			return !slices.Contains(append([]T{first}, rest...), value)
		}),
	)
}

// Equal requires destination to equal value.
func (scope ruleScope[C]) Equal[D any, T comparable](destination destinationLike[D, T], value T) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleEqual(),
		[]*destinationHandle{destinationHandleOf(destination)},
		[]any{value},
		CountRange{},
		"",
		unaryRulePredicate(func(observed T) bool { return observed == value }),
	)
}

// NotEqual requires destination to differ from value.
func (scope ruleScope[C]) NotEqual[D any, T comparable](destination destinationLike[D, T], value T) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleNotEqual(),
		[]*destinationHandle{destinationHandleOf(destination)},
		[]any{value},
		CountRange{},
		"",
		unaryRulePredicate(func(observed T) bool { return observed != value }),
	)
}

// Between requires destination to lie inclusively between minimum and maximum.
func (scope ruleScope[C]) Between[D any, T cmp.Ordered](destination destinationLike[D, T], minimum, maximum T) {
	if minimum > maximum {
		stageInvalidRule(scope.session, scope.profile)
		return
	}
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleBetween(),
		[]*destinationHandle{destinationHandleOf(destination)},
		[]any{minimum, maximum},
		CountRange{},
		"",
		unaryRulePredicate(func(value T) bool { return value >= minimum && value <= maximum }),
	)
}

// AtLeast requires destination to be greater than or equal to minimum.
func (scope ruleScope[C]) AtLeast[D any, T cmp.Ordered](destination destinationLike[D, T], minimum T) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleAtLeast(),
		[]*destinationHandle{destinationHandleOf(destination)},
		[]any{minimum},
		CountRange{},
		"",
		unaryRulePredicate(func(value T) bool { return value >= minimum }),
	)
}

// AtMost requires destination to be less than or equal to maximum.
func (scope ruleScope[C]) AtMost[D any, T cmp.Ordered](destination destinationLike[D, T], maximum T) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleAtMost(),
		[]*destinationHandle{destinationHandleOf(destination)},
		[]any{maximum},
		CountRange{},
		"",
		unaryRulePredicate(func(value T) bool { return value <= maximum }),
	)
}

// GreaterThan requires destination to be strictly greater than minimum.
func (scope ruleScope[C]) GreaterThan[D any, T cmp.Ordered](destination destinationLike[D, T], minimum T) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleGreaterThan(),
		[]*destinationHandle{destinationHandleOf(destination)},
		[]any{minimum},
		CountRange{},
		"",
		unaryRulePredicate(func(value T) bool { return value > minimum }),
	)
}

// LessThan requires destination to be strictly less than maximum.
func (scope ruleScope[C]) LessThan[D any, T cmp.Ordered](destination destinationLike[D, T], maximum T) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleLessThan(),
		[]*destinationHandle{destinationHandleOf(destination)},
		[]any{maximum},
		CountRange{},
		"",
		unaryRulePredicate(func(value T) bool { return value < maximum }),
	)
}

// Match requires a string destination to match pattern using Go regular-expression semantics.
func (scope ruleScope[C]) Match[D any](destination destinationLike[D, string], pattern string) {
	func() {
		compiled, err := regexp.Compile(pattern)
		var predicate authoringBuiltinRulePredicate
		if err == nil {
			predicate = unaryRulePredicate(func(value string) bool { return compiled.MatchString(value) })
		}
		stageBuiltinRuleWithPredicate(
			scope.session,
			scope.profile,
			scope.activation,
			semanticRuleMatch(),
			[]*destinationHandle{destinationHandleOf(destination)},
			nil,
			CountRange{},
			pattern,
			predicate,
		)
	}()
}

// ItemCount requires the sequence destination's item count to satisfy count.
func (scope ruleScope[C]) ItemCount[D, T any](destination destinationLike[D, []T], count CountRange) {
	stageBuiltinRule(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleItemCount(),
		[]*destinationHandle{destinationHandleOf(destination)},
		count,
	)
}

// Contains requires the sequence destination to contain every declared value.
func (scope ruleScope[C]) Contains[D any, T comparable](destination destinationLike[D, []T], first T, rest ...T) {
	values := append([]any{first}, anySlice(rest)...)
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleContains(),
		[]*destinationHandle{destinationHandleOf(destination)},
		values,
		CountRange{},
		"",
		unaryRulePredicate(func(sequence []T) bool {
			for _, required := range append([]T{first}, rest...) {
				found := slices.Contains(sequence, required)
				if !found {
					return false
				}
			}
			return true
		}),
	)
}

// EachIn requires every sequence element to belong to the declared allowed set.
func (scope ruleScope[C]) EachIn[D any, T comparable](destination destinationLike[D, []T], first T, rest ...T) {
	values := append([]any{first}, anySlice(rest)...)
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleEachIn(),
		[]*destinationHandle{destinationHandleOf(destination)},
		values,
		CountRange{},
		"",
		unaryRulePredicate(func(sequence []T) bool {
			allowed := append([]T{first}, rest...)
			for _, value := range sequence {
				found := slices.Contains(allowed, value)
				if !found {
					return false
				}
			}
			return true
		}),
	)
}

// Unique requires sequence elements to be pairwise unique.
func (scope ruleScope[C]) Unique[D any, T comparable](destination destinationLike[D, []T]) {
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleUnique(),
		[]*destinationHandle{destinationHandleOf(destination)},
		nil,
		CountRange{},
		"",
		unaryRulePredicate(func(sequence []T) bool {
			seen := make(map[T]struct{}, len(sequence))
			for _, value := range sequence {
				if _, ok := seen[value]; ok {
					return false
				}
				seen[value] = struct{}{}
			}
			return true
		}),
	)
}

// EntryCount requires the keyed destination's entry count to satisfy count.
func (scope ruleScope[C]) EntryCount[D any, K comparable, V any](destination destinationLike[D, map[K]V], count CountRange) {
	stageBuiltinRule(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleEntryCount(),
		[]*destinationHandle{destinationHandleOf(destination)},
		count,
	)
}

// HasKeys requires the keyed destination to contain every declared key.
func (scope ruleScope[C]) HasKeys[D any, K comparable, V any](destination destinationLike[D, map[K]V], first K, rest ...K) {
	values := append([]any{first}, anySlice(rest)...)
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleHasKeys(),
		[]*destinationHandle{destinationHandleOf(destination)},
		values,
		CountRange{},
		"",
		unaryRulePredicate(func(keyed map[K]V) bool {
			for _, key := range append([]K{first}, rest...) {
				if _, ok := keyed[key]; !ok {
					return false
				}
			}
			return true
		}),
	)
}

// KeysIn requires every key in the keyed destination to belong to the declared allowed set.
func (scope ruleScope[C]) KeysIn[D any, K comparable, V any](destination destinationLike[D, map[K]V], first K, rest ...K) {
	values := append([]any{first}, anySlice(rest)...)
	stageBuiltinRuleWithPredicate(
		scope.session,
		scope.profile,
		scope.activation,
		semanticRuleKeysIn(),
		[]*destinationHandle{destinationHandleOf(destination)},
		values,
		CountRange{},
		"",
		unaryRulePredicate(func(keyed map[K]V) bool {
			allowed := append([]K{first}, rest...)
			for key := range keyed {
				found := slices.Contains(allowed, key)
				if !found {
					return false
				}
			}
			return true
		}),
	)
}

// ExactlyOne starts a heterogeneous presence group requiring exactly one dependency to be present;
// And appends more members.
func (scope ruleScope[C]) ExactlyOne(first, second Dependency) ruleGroup {
	return stageRuleGroup(scope.session, scope.profile, scope.activation, semanticRuleExactlyOne(), first, second)
}

// AllOrNone starts a heterogeneous presence group requiring either all dependencies or none to be
// present; And appends more members.
func (scope ruleScope[C]) AllOrNone(first, second Dependency) ruleGroup {
	return stageRuleGroup(scope.session, scope.profile, scope.activation, semanticRuleAllOrNone(), first, second)
}

// AtMostOne starts a heterogeneous presence group allowing at most one present dependency; And
// appends more members.
func (scope ruleScope[C]) AtMostOne(first, second Dependency) ruleGroup {
	return stageRuleGroup(scope.session, scope.profile, scope.activation, semanticRuleAtMostOne(), first, second)
}

// AtLeastOne starts a heterogeneous presence group requiring at least one present dependency; And
// appends more members.
func (scope ruleScope[C]) AtLeastOne(first, second Dependency) ruleGroup {
	return stageRuleGroup(scope.session, scope.profile, scope.activation, semanticRuleAtLeastOne(), first, second)
}

// Equal requires the two destinations to resolve to equal values.
func (compare ruleComparisons[C]) Equal[L, R any, T comparable](left destinationLike[L, T], right destinationLike[R, T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleEqual(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right T) bool { return left == right }),
	)
}

// NotEqual requires the two destinations to resolve to different values.
func (compare ruleComparisons[C]) NotEqual[L, R any, T comparable](left destinationLike[L, T], right destinationLike[R, T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleNotEqual(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right T) bool { return left != right }),
	)
}

// LessThan requires left to be strictly less than right.
func (compare ruleComparisons[C]) LessThan[L, R any, T cmp.Ordered](left destinationLike[L, T], right destinationLike[R, T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleLessThan(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right T) bool { return left < right }),
	)
}

// AtMost requires left to be less than or equal to right.
func (compare ruleComparisons[C]) AtMost[L, R any, T cmp.Ordered](left destinationLike[L, T], right destinationLike[R, T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleAtMost(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right T) bool { return left <= right }),
	)
}

// GreaterThan requires left to be strictly greater than right.
func (compare ruleComparisons[C]) GreaterThan[L, R any, T cmp.Ordered](left destinationLike[L, T], right destinationLike[R, T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleGreaterThan(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right T) bool { return left > right }),
	)
}

// AtLeast requires left to be greater than or equal to right.
func (compare ruleComparisons[C]) AtLeast[L, R any, T cmp.Ordered](left destinationLike[L, T], right destinationLike[R, T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleAtLeast(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right T) bool { return left >= right }),
	)
}

// In requires the scalar value destination to occur in the sequence collection destination.
func (compare ruleComparisons[C]) In[L, R any, T comparable](value destinationLike[L, T], collection destinationLike[R, []T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleIn(),
		[]*destinationHandle{destinationHandleOf(value), destinationHandleOf(collection)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(value T, collection []T) bool {
			return slices.Contains(collection, value)
		}),
	)
}

// NotIn requires the scalar value destination not to occur in the sequence collection destination.
func (compare ruleComparisons[C]) NotIn[L, R any, T comparable](value destinationLike[L, T], collection destinationLike[R, []T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleNotIn(),
		[]*destinationHandle{destinationHandleOf(value), destinationHandleOf(collection)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(value T, collection []T) bool {
			return !slices.Contains(collection, value)
		}),
	)
}

// SubsetOf requires every value in left to occur in right.
func (compare ruleComparisons[C]) SubsetOf[L, R any, T comparable](left destinationLike[L, []T], right destinationLike[R, []T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleSubsetOf(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right []T) bool {
			set := make(map[T]struct{}, len(right))
			for _, value := range right {
				set[value] = struct{}{}
			}
			for _, value := range left {
				if _, ok := set[value]; !ok {
					return false
				}
			}
			return true
		}),
	)
}

// Disjoint requires left and right to share no values.
func (compare ruleComparisons[C]) Disjoint[L, R any, T comparable](left destinationLike[L, []T], right destinationLike[R, []T]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleDisjoint(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left, right []T) bool {
			set := make(map[T]struct{}, len(right))
			for _, value := range right {
				set[value] = struct{}{}
			}
			for _, value := range left {
				if _, ok := set[value]; ok {
					return false
				}
			}
			return true
		}),
	)
}

// KeysSubsetOf requires every key in left to occur in right.
func (compare ruleComparisons[C]) KeysSubsetOf[L, R any, K comparable, LV, RV any](left destinationLike[L, map[K]LV], right destinationLike[R, map[K]RV]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleKeysSubsetOf(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left map[K]LV, right map[K]RV) bool {
			for key := range left {
				if _, ok := right[key]; !ok {
					return false
				}
			}
			return true
		}),
	)
}

// KeysDisjoint requires left and right to share no keys.
func (compare ruleComparisons[C]) KeysDisjoint[L, R any, K comparable, LV, RV any](left destinationLike[L, map[K]LV], right destinationLike[R, map[K]RV]) {
	stageBuiltinRuleWithPredicate(
		compare.session,
		compare.profile,
		compare.activation,
		semanticRuleKeysDisjoint(),
		[]*destinationHandle{destinationHandleOf(left), destinationHandleOf(right)},
		nil,
		CountRange{},
		"",
		binaryRulePredicate(func(left map[K]LV, right map[K]RV) bool {
			for key := range left {
				if _, ok := right[key]; ok {
					return false
				}
			}
			return true
		}),
	)
}

type ruleConditionKind uint8

const (
	ruleConditionInvalid ruleConditionKind = iota
	ruleConditionStructural
	ruleConditionContributedBy
	ruleConditionAll
	ruleConditionAny
	ruleConditionNot
)

type ruleConditionState struct {
	kind          ruleConditionKind
	owner         *draftIdentity
	structural    *conditionState
	destination   destinationIdentity
	producerKinds []ValueProducerKind
	children      []*ruleConditionState
	valid         bool
}

func ruleConditionStateOf(condition RuleCondition) *ruleConditionState {
	switch condition := condition.(type) {
	case Condition:
		state := &ruleConditionState{
			kind:       ruleConditionStructural,
			structural: condition.state,
		}
		if condition.state != nil {
			state.owner = condition.state.owner
			state.valid = condition.state.valid
		}
		return state
	case validationRuleCondition:
		return condition.state
	default:
		return nil
	}
}

func contributedByRuleCondition[D, T any](state *declarationSession, destination destinationLike[D, T], first ValueProducerKind, rest ...ValueProducerKind) RuleCondition {
	result := &ruleConditionState{
		kind: ruleConditionContributedBy,
	}
	if state != nil && destination != nil {
		rawDestination := destination.destinationHandle()
		state.use(func(draft *declarationDraft, _ commandRef) {
			result.owner = draft.identity
			result.destination, result.valid = captureAuthoringDestinationIdentity(draft, rawDestination)
		})
	}
	result.producerKinds = append([]ValueProducerKind{first}, rest...)
	result.valid = result.valid && validValueProducerKinds(result.producerKinds)
	return validationRuleCondition{
		state: result,
	}
}

func combineRuleConditions(state *declarationSession, kind ruleConditionKind, first RuleCondition, rest ...RuleCondition) RuleCondition {
	result := &ruleConditionState{
		kind: kind,
	}
	if state != nil {
		state.use(func(draft *declarationDraft, _ commandRef) {
			result.owner = draft.identity
		})
	}
	conditions := append([]RuleCondition{first}, rest...)
	if kind == ruleConditionNot && len(conditions) != 1 {
		result.valid = false
		return validationRuleCondition{
			state: result,
		}
	}
	result.valid = result.owner != nil && len(conditions) > 0
	result.children = make([]*ruleConditionState, len(conditions))
	for i, condition := range conditions {
		child := ruleConditionStateOf(condition)
		result.children[i] = child
		if child == nil || !child.valid || child.owner != result.owner {
			result.valid = false
		}
	}
	return validationRuleCondition{
		state: result,
	}
}

func validValueProducerKinds(producerKinds []ValueProducerKind) bool {
	if len(producerKinds) == 0 {
		return false
	}
	seen := make(map[ValueProducerKind]struct{}, len(producerKinds))
	for _, producerKind := range producerKinds {
		if producerKind < ValueProducerKindFlag || producerKind > ValueProducerKindDerived {
			return false
		}
		if _, duplicate := seen[producerKind]; duplicate {
			return false
		}
		seen[producerKind] = struct{}{}
	}
	return true
}

func activatedRuleScope[C any](scope ruleScope[C], condition RuleCondition, unless bool) ruleScope[C] {
	if scope.session == nil {
		return scope
	}
	if unless {
		condition = combineRuleConditions(scope.session, ruleConditionNot, condition)
	}
	conditionState := ruleConditionStateOf(condition)
	if conditionState == nil || !conditionState.valid {
		scope.activation = validationRuleCondition{
			state: &ruleConditionState{
				owner: conditionStateOwner(scope.session),
			},
		}
		return scope
	}
	if scope.activation != nil {
		condition = combineRuleConditions(scope.session, ruleConditionAll, scope.activation, condition)
	}
	scope.activation = condition
	scope.Compare.activation = condition
	return scope
}

func conditionStateOwner(session *declarationSession) *draftIdentity {
	if session == nil || session.draft == nil {
		return nil
	}
	return session.draft.identity
}

func stageInvalidRule(session *declarationSession, profile grammarProfileCaseRef) {
	stageRule(session, profile, ruleDraft{})
}

func stageRule(session *declarationSession, profile grammarProfileCaseRef, rule ruleDraft) ruleDraftRef {
	if session == nil {
		return ruleDraftRef{}
	}

	var ref ruleDraftRef
	session.use(func(draft *declarationDraft, command commandRef) {
		ref = stageRuleInDraft(draft, command, profile, rule)
	})
	return ref
}

func stageRuleInDraft(draft *declarationDraft, command commandRef, profile grammarProfileCaseRef, rule ruleDraft) ruleDraftRef {
	if draft == nil {
		return ruleDraftRef{}
	}

	rule.valid = rule.valid && draft.identity != nil
	record := ruleDraft{
		position:         draft.nextPosition(),
		command:          command,
		profile:          profile,
		kind:             rule.kind,
		code:             rule.code,
		activation:       rule.activation,
		destinations:     append([]destinationIdentity(nil), rule.destinations...),
		operands:         append([]any(nil), rule.operands...),
		count:            rule.count,
		pattern:          rule.pattern,
		callbackKind:     rule.callbackKind,
		callback:         rule.callback,
		builtinPredicate: rule.builtinPredicate,
		valid:            rule.valid,
	}
	return draft.rules.append(draft, record)
}

func stageBuiltinRule(session *declarationSession, profile grammarProfileCaseRef, activation RuleCondition, kind RuleKind, destinations []*destinationHandle, count CountRange) {
	stageBuiltinRuleWithPredicate(session, profile, activation, kind, destinations, nil, count, "", nil)
}

func stageBuiltinRuleWithPredicate(session *declarationSession, profile grammarProfileCaseRef, activation RuleCondition, kind RuleKind, destinations []*destinationHandle, operands []any, count CountRange, pattern string, predicate authoringBuiltinRulePredicate) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		valid := kind.valid() && kind != RuleCustom && len(destinations) > 0
		relations := make([]destinationIdentity, len(destinations))
		if valid {
			for index, destination := range destinations {
				if destination == nil {
					valid = false
					break
				}
				relation, ok := captureAuthoringDestinationIdentity(draft, destination)
				if !ok {
					valid = false
					break
				}
				relations[index] = relation
			}
		}
		if activation != nil {
			condition := ruleConditionStateOf(activation)
			valid = valid && condition != nil && condition.valid && condition.owner == draft.identity
		}
		if kind == RuleMatch {
			_, err := regexp.Compile(pattern)
			valid = valid && err == nil
		}
		if kind == RuleItemCount || kind == RuleEntryCount {
			valid = valid && count.valid
		}
		rule := ruleDraft{
			kind:             kind,
			activation:       activation,
			destinations:     relations,
			operands:         operands,
			count:            count,
			pattern:          pattern,
			builtinPredicate: predicate,
			valid:            valid,
		}
		stageRuleInDraft(draft, command, profile, rule)
	})
}

func unaryRulePredicate[T any](predicate func(T) bool) authoringBuiltinRulePredicate {
	if predicate == nil {
		return nil
	}
	return func(values []any) bool {
		if len(values) != 1 {
			return false
		}
		value, ok := values[0].(T)
		return ok && predicate(value)
	}
}

const binaryRuleOperandCount = 2

func binaryRulePredicate[L, R any](predicate func(L, R) bool) authoringBuiltinRulePredicate {
	if predicate == nil {
		return nil
	}
	return func(values []any) bool {
		if len(values) != binaryRuleOperandCount {
			return false
		}
		left, leftOK := values[0].(L)
		right, rightOK := values[1].(R)
		return leftOK && rightOK && predicate(left, right)
	}
}

func stageCustomRule[C any](scope ruleScope[C], code RuleCode, callbackKind authoringRuleCallbackKind, callback authoringRuleCallback) RuleDeclaration[C] {
	valid := scope.session != nil && validRuleCode(string(code)) && callback != nil &&
		(callbackKind == authoringRuleCallbackCheck || callbackKind == authoringRuleCallbackValidate)
	if scope.activation != nil && scope.session != nil {
		condition := ruleConditionStateOf(scope.activation)
		valid = valid && condition != nil && condition.valid && condition.owner == conditionStateOwner(scope.session)
	}
	rule := ruleDraft{
		kind:         RuleCustom,
		code:         code,
		activation:   scope.activation,
		callbackKind: callbackKind,
		callback:     callback,
		valid:        valid,
	}
	ref := stageRule(scope.session, scope.profile, rule)
	return RuleDeclaration[C]{
		state: scope.session,
		rule:  ref,
	}
}

func eraseRuleCheck[C any](check RulePredicate[C]) authoringRuleCallback {
	if check == nil {
		return nil
	}
	return func(ctx context.Context, raw any, dependencies Dependencies) (ruleCallbackResult, error) {
		config, ok := raw.(*C)
		if !ok || config == nil {
			panic("cli: invalid rule callback configuration")
		}
		value, err := check(ctx, config, dependencies)
		return ruleCallbackResult{
			Check: value,
		}, err
	}
}

func eraseRuleValidate[C any](validate RuleValidator[C]) authoringRuleCallback {
	if validate == nil {
		return nil
	}
	return func(ctx context.Context, raw any, dependencies Dependencies) (ruleCallbackResult, error) {
		config, ok := raw.(*C)
		if !ok || config == nil {
			panic("cli: invalid rule callback configuration")
		}
		violations, err := validate(ctx, config, dependencies)
		if err != nil {
			return ruleCallbackResult{}, err
		}
		result := ruleCallbackResult{
			Violations: make([]ruleViolation, len(violations)),
		}
		for i, violation := range violations {
			kind := ruleViolationUnknown
			switch violation.Kind {
			case ViolationInvalid:
				kind = ruleViolationInvalid
			case ViolationConflict:
				kind = ruleViolationConflict
			case ViolationDuplicate:
				kind = ruleViolationDuplicate
			case ViolationUnknown:
			}
			origins := make([]uint32, len(violation.Origins))
			for j, origin := range violation.Origins {
				origins[j] = origin.value
			}
			result.Violations[i] = ruleViolation{
				Kind:    kind,
				Origins: origins,
			}
		}
		return result, nil
	}
}

func stageRuleSubjectDirective(session *declarationSession, rule ruleDraftRef, destination *destinationHandle) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		owner, ownerOK := draft.rules.resolve(draft, rule)
		directive := ruleDirectiveDraft{
			rule: rule,
			kind: ruleDirectiveSubject,
		}
		if ownerOK && destination != nil {
			directive.destination, directive.valid = captureAuthoringDestinationIdentity(draft, destination)
		}
		profile := grammarProfileCaseRef{}
		if ownerOK {
			profile = owner.profile
		}
		record := ruleDirectiveDraft{
			position:    draft.nextPosition(),
			command:     command,
			profile:     profile,
			rule:        rule,
			kind:        directive.kind,
			destination: directive.destination,
			optional:    directive.optional,
			valid:       directive.valid,
		}
		draft.ruleDirectives.append(draft, record)
	})
}

func stageRuleDirective(session *declarationSession, rule ruleDraftRef, kind ruleDirectiveKind, dependency Dependency) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		owner, ownerOK := draft.rules.resolve(draft, rule)
		directive := ruleDirectiveDraft{
			rule: rule,
			kind: kind,
		}
		if ownerOK {
			destination, optional := dependencyTarget(dependency)
			directive.optional = optional
			directive.destination, directive.valid = captureAuthoringDestinationIdentity(draft, destination)
		}
		profile := grammarProfileCaseRef{}
		if ownerOK {
			profile = owner.profile
		}
		record := ruleDirectiveDraft{
			position:    draft.nextPosition(),
			command:     command,
			profile:     profile,
			rule:        rule,
			kind:        directive.kind,
			destination: directive.destination,
			optional:    directive.optional,
			valid:       directive.valid,
		}
		draft.ruleDirectives.append(draft, record)
	})
}

func stageRuleGroup(state *declarationSession, profile grammarProfileCaseRef, activation RuleCondition, kind RuleKind, first, second Dependency) ruleGroup {
	rule := ruleDraft{
		kind:       kind,
		activation: activation,
		valid:      true,
	}
	ref := stageRule(state, profile, rule)
	if ref.issued() {
		stageRuleDirective(state, ref, ruleDirectiveGroupDependency, first)
		stageRuleDirective(state, ref, ruleDirectiveGroupDependency, second)
	}
	return ruleGroup{
		session: state,
		rule:    ref,
	}
}

func dependencyTarget(dependency Dependency) (*destinationHandle, bool) {
	if dependency == nil {
		return nil, false
	}
	return dependency.dependencyTarget()
}

func validRuleDraft(value ruleDraft, commandIndex, profileIndex int) bool {
	_ = commandIndex
	_ = profileIndex
	if !value.valid || !value.kind.valid() {
		return false
	}
	if value.kind == RuleCustom {
		return validRuleCode(string(value.code)) && value.callback != nil &&
			(value.callbackKind == authoringRuleCallbackCheck || value.callbackKind == authoringRuleCallbackValidate)
	}
	return value.code == "" && value.callback == nil && value.callbackKind == authoringRuleCallbackNone
}

func validAdvisoryDraft(value advisoryDraft, commandIndex, profileIndex int) bool {
	_ = commandIndex
	_ = profileIndex
	if !value.valid {
		return false
	}
	if !value.advisory.Severity.valid() {
		return false
	}
	advisoryDeclarationValue := advisoryDeclaration{
		Code:     string(value.advisory.Code),
		Severity: value.advisory.Severity,
		Message:  value.advisory.Message,
	}
	if !validateAdvisoryDeclaration(advisoryDeclarationValue) {
		return false
	}
	switch value.kind {
	case AdvisoryKindAlways:
		return value.condition.state == nil && value.callback == nil
	case AdvisoryKindConditional:
		return value.condition.state != nil && value.condition.state.valid && value.callback == nil
	case AdvisoryKindDetector:
		return value.condition.state == nil && value.callback != nil
	default:
		return false
	}
}

func anySlice[T any](values []T) []any {
	result := make([]any, len(values))
	for i := range values {
		result[i] = values[i]
	}
	return result
}

func semanticRuleRequire() RuleKind { return RuleRequire }

func semanticRuleForbid() RuleKind { return RuleForbid }

func semanticRuleNotEmpty() RuleKind { return RuleNotEmpty }

func semanticRuleIn() RuleKind { return RuleIn }

func semanticRuleNotIn() RuleKind { return RuleNotIn }

func semanticRuleEqual() RuleKind { return RuleEqual }

func semanticRuleNotEqual() RuleKind { return RuleNotEqual }

func semanticRuleBetween() RuleKind { return RuleBetween }

func semanticRuleAtLeast() RuleKind { return RuleAtLeast }

func semanticRuleAtMost() RuleKind { return RuleAtMost }

func semanticRuleGreaterThan() RuleKind { return RuleGreaterThan }

func semanticRuleLessThan() RuleKind { return RuleLessThan }

func semanticRuleMatch() RuleKind { return RuleMatch }

func semanticRuleItemCount() RuleKind { return RuleItemCount }

func semanticRuleContains() RuleKind { return RuleContains }

func semanticRuleEachIn() RuleKind { return RuleEachIn }

func semanticRuleUnique() RuleKind { return RuleUnique }

func semanticRuleEntryCount() RuleKind { return RuleEntryCount }

func semanticRuleHasKeys() RuleKind { return RuleHasKeys }

func semanticRuleKeysIn() RuleKind { return RuleKeysIn }

func semanticRuleSubsetOf() RuleKind { return RuleSubsetOf }

func semanticRuleDisjoint() RuleKind { return RuleDisjoint }

func semanticRuleKeysSubsetOf() RuleKind { return RuleKeysSubsetOf }

func semanticRuleKeysDisjoint() RuleKind { return RuleKeysDisjoint }

func semanticRuleExactlyOne() RuleKind { return RuleExactlyOne }

func semanticRuleAllOrNone() RuleKind { return RuleAllOrNone }

func semanticRuleAtMostOne() RuleKind { return RuleAtMostOne }

func semanticRuleAtLeastOne() RuleKind { return RuleAtLeastOne }

type authoringRuleCallbackKind uint8

const (
	authoringRuleCallbackNone authoringRuleCallbackKind = iota
	authoringRuleCallbackCheck
	authoringRuleCallbackValidate
)

type authoringRuleCallback func(context.Context, any, Dependencies) (ruleCallbackResult, error)

type authoringBuiltinRulePredicate func([]any) bool

type ruleDirectiveKind uint8

const (
	ruleDirectiveInvalid ruleDirectiveKind = iota
	ruleDirectiveSubject
	ruleDirectiveDependency
	ruleDirectiveGroupDependency
)

type ruleCallbackKind uint8

const (
	ruleCallbackNone ruleCallbackKind = iota
	ruleCallbackCheck
	ruleCallbackValidate
)

type ruleViolationKind uint8

const (
	ruleViolationUnknown ruleViolationKind = iota
	ruleViolationInvalid
	ruleViolationConflict
	ruleViolationDuplicate
)

type ruleViolation struct {
	Kind    ruleViolationKind
	Origins []uint32
}

type ruleCallbackResult struct {
	Check      bool
	Violations []ruleViolation
}

type (
	ruleCallback         func(context.Context, any, Dependencies) (ruleCallbackResult, error)
	ruleBuiltinPredicate func([]any) bool
)

type ruleDefinition struct {
	kind             RuleKind
	code             string
	subjects         []ValueID
	dependencies     []dependencyRequirement
	operands         []any
	activation       *policyCondition
	count            semanticCountRange
	pattern          string
	callbackKind     ruleCallbackKind
	callback         ruleCallback
	builtinPredicate ruleBuiltinPredicate
}

type builtinRuleDefinitionInput struct {
	Kind         RuleKind
	Subjects     []ValueID
	Dependencies []dependencyRequirement
	Operands     []any
	Activation   *policyCondition
	Count        semanticCountRange
	Pattern      string
	Predicate    ruleBuiltinPredicate
}

type customRuleDefinitionInput struct {
	Code         string
	Subjects     []ValueID
	Dependencies []dependencyRequirement
	Activation   *policyCondition
	Callback     ruleCallback
}

func acceptBuiltinRuleDefinition(input builtinRuleDefinitionInput) ruleDefinition {
	definition := ruleDefinition{
		kind:             input.Kind,
		subjects:         input.Subjects,
		dependencies:     input.Dependencies,
		operands:         input.Operands,
		activation:       input.Activation,
		count:            input.Count,
		pattern:          input.Pattern,
		builtinPredicate: input.Predicate,
	}
	return finalizeRuleDefinition(definition)
}

func acceptCustomRuleCheckDefinition(input customRuleDefinitionInput) ruleDefinition {
	return acceptCustomRuleDefinition(input, ruleCallbackCheck)
}

func acceptCustomRuleValidateDefinition(input customRuleDefinitionInput) ruleDefinition {
	return acceptCustomRuleDefinition(input, ruleCallbackValidate)
}

func acceptCustomRuleDefinition(input customRuleDefinitionInput, callbackKind ruleCallbackKind) ruleDefinition {
	definition := ruleDefinition{
		kind:         RuleCustom,
		code:         input.Code,
		subjects:     input.Subjects,
		dependencies: input.Dependencies,
		activation:   input.Activation,
		callbackKind: callbackKind,
		callback:     input.Callback,
	}
	return finalizeRuleDefinition(definition)
}

func finalizeRuleDefinition(definition ruleDefinition) ruleDefinition {
	definition.subjects = append([]ValueID(nil), definition.subjects...)
	definition.dependencies = append([]dependencyRequirement(nil), definition.dependencies...)
	definition.operands = append([]any(nil), definition.operands...)
	if definition.activation != nil {
		activationCopy := *definition.activation
		definition.activation = &activationCopy
	}
	if !definition.Valid() {
		panic("semantic: invalid Rule definition")
	}
	return definition
}

const binaryRuleSubjectCount = 2

func (definition ruleDefinition) Valid() bool {
	if !definition.kind.valid() {
		return false
	}
	for _, id := range definition.subjects {
		if !id.present() {
			return false
		}
	}
	for _, dependency := range definition.dependencies {
		if !dependency.Destination().present() {
			return false
		}
	}
	if definition.activation != nil && !definition.activation.Valid() {
		return false
	}
	if definition.kind == RuleCustom {
		return definition.code != "" && len(definition.operands) == 0 && definition.pattern == "" &&
			(definition.callbackKind == ruleCallbackCheck || definition.callbackKind == ruleCallbackValidate) && definition.callback != nil
	}
	if definition.code != "" || definition.callbackKind != ruleCallbackNone || definition.callback != nil {
		return false
	}

	oneSubject := len(definition.subjects) == 1
	twoSubjects := len(definition.subjects) == binaryRuleSubjectCount
	groupSubjects := len(definition.subjects) >= binaryRuleSubjectCount
	noOperands := len(definition.operands) == 0
	noPattern := definition.pattern == ""

	switch definition.kind {
	case RuleRequire, RuleForbid, RuleNotEmpty, RuleUnique:
		return oneSubject && noOperands && noPattern
	case RuleIn, RuleNotIn:
		// Literal membership uses one subject plus a nonempty operand set. The
		// comparison form uses two subjects and no literal operands.
		return noPattern && ((oneSubject && len(definition.operands) > 0) || (twoSubjects && noOperands))
	case RuleEqual, RuleNotEqual, RuleAtLeast, RuleAtMost, RuleGreaterThan, RuleLessThan:
		// Scalar forms compare one subject with one literal. Rule.Compare uses
		// two destinations and therefore carries no literal operand.
		return noPattern && ((oneSubject && len(definition.operands) == 1) || (twoSubjects && noOperands))
	case RuleBetween:
		return oneSubject && len(definition.operands) == 2 && noPattern
	case RuleMatch:
		return oneSubject && noOperands
	case RuleItemCount, RuleEntryCount:
		return oneSubject && noOperands && noPattern && definition.count.valid()
	case RuleContains, RuleEachIn, RuleHasKeys, RuleKeysIn:
		return oneSubject && len(definition.operands) > 0 && noPattern
	case RuleSubsetOf, RuleDisjoint, RuleKeysSubsetOf, RuleKeysDisjoint:
		return twoSubjects && noOperands && noPattern
	case RuleExactlyOne, RuleAllOrNone, RuleAtMostOne, RuleAtLeastOne:
		return groupSubjects && noOperands && noPattern
	default:
		return false
	}
}

func (definition ruleDefinition) Kind() RuleKind { return definition.kind }

func (definition ruleDefinition) Code() string { return definition.code }

func (definition ruleDefinition) Subjects() []ValueID {
	return append([]ValueID(nil), definition.subjects...)
}

func (definition ruleDefinition) Dependencies() []dependencyRequirement {
	return append([]dependencyRequirement(nil), definition.dependencies...)
}

func (definition ruleDefinition) Operands() []any { return append([]any(nil), definition.operands...) }

func (definition ruleDefinition) Activation() (policyCondition, bool) {
	if definition.activation == nil {
		return policyCondition{}, false
	}
	return *definition.activation, true
}

func (definition ruleDefinition) Count() semanticCountRange { return definition.count }

func (definition ruleDefinition) Pattern() string { return definition.pattern }

func (definition ruleDefinition) CallbackKind() ruleCallbackKind { return definition.callbackKind }

func (definition ruleDefinition) BuiltinPredicate() (ruleBuiltinPredicate, bool) {
	return definition.builtinPredicate, definition.builtinPredicate != nil
}

func (definition ruleDefinition) Evaluate(ctx context.Context, config any, dependencies Dependencies) (ruleCallbackResult, error) {
	if definition.callback == nil {
		panic("semantic: Rule has no callback")
	}
	result, err := definition.callback(ctx, config, dependencies)
	result.Violations = append([]ruleViolation(nil), result.Violations...)
	return result, err
}

type ruleRecord struct {
	meta       recordMeta[RuleID]
	command    CommandID
	definition ruleDefinition
}

func acceptRuleRecord(id RuleID, command CommandID, position declarationPosition) ruleRecord {
	mustCommand(command)
	return ruleRecord{
		meta:    acceptRecordMeta(id, position),
		command: command,
	}
}

// RuleID is the exact identity of an accepted Rule declaration. Zero is invalid.
// Static IDs are stable within one immutable Application; provider-produced IDs are
// operation-local. RuleID is distinct from RuleCode and declaration order.
type RuleID uint32

// RuleCode is a custom Rule classification, not Rule identity. Distinct Rules may
// legally share the same code.
type RuleCode string

// RulePredicate evaluates a custom Check rule over read-only analysis state.
// true, nil satisfies the rule; false, nil produces one generic invalid violation;
// any non-nil error is a rule-evaluation failure.
type RulePredicate[C any] func(context.Context, *C, Dependencies) (bool, error)

// RuleValidator evaluates a custom Validate rule and returns its structured
// violations. A non-nil error is evaluation failure rather than a violation.
type RuleValidator[C any] func(context.Context, *C, Dependencies) ([]Violation, error)

// ViolationKind classifies a structured custom Rule violation.
type ViolationKind uint8

const (
	// ViolationUnknown is the zero value and does not classify a violation.
	ViolationUnknown ViolationKind = iota
	// ViolationInvalid identifies generic custom semantic invalidity.
	ViolationInvalid
	// ViolationConflict identifies incompatible semantic pieces or authorized origins.
	ViolationConflict
	// ViolationDuplicate identifies duplicate origins in canonical original-then-duplicate order.
	ViolationDuplicate
)

// Violation is one custom Rule violation. Origins identify the source occurrences
// implicated by the violation in their canonical semantic order.
type Violation struct {
	Kind    ViolationKind
	Origins []InputOriginID
}
