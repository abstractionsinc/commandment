package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// ConditionExpressionLocator identifies one public condition-expression node reached through a
// declaration relationship. Its child and operand locators use public ordered coordinates; it
// carries locator coordinates rather than observed state, and its zero value is invalid.
type ConditionExpressionLocator struct {
	_     [0]func()
	state *conditionExpressionLocatorState
}

// ConditionLocator identifies one structural Condition for clitest semantic
// expectation. Its zero value is invalid.
type ConditionLocator struct {
	_     [0]func()
	state *conditionLocatorState
}

// ConditionAt returns a locator for the condition at index in the located command's authoritative
// public condition order.
func (l CommandLocator) ConditionAt(index int) ConditionLocator {
	return conditionIndexLocator(l, index)
}

// ConditionsExactly returns an expectation that checks the located command's conditions are
// exactly the supplied locators in authoritative public order. Exactness is not inferred from an
// incomplete observation.
func (l CommandLocator) ConditionsExactly(values ...ConditionLocator) DeclarationExpectation {
	expectation := commandConditionsExpectation{
		command: l.state,
		items:   conditionLocatorStates(values),
	}
	return expectation
}

// Expression returns a locator that follows the public expression relationship from the located
// condition. The returned locator carries locator coordinates, not observed state.
func (locator ConditionLocator) Expression() ConditionExpressionLocator {
	state := &conditionExpressionLocatorState{
		mode:      conditionExpressionLocatorCondition,
		condition: locator.state,
	}
	conditionExpressionLocator := ConditionExpressionLocator{
		state: state,
	}
	return conditionExpressionLocator
}

// Exists returns an expectation that checks the located Condition exists in the relevant
// public declaration observation.
func (locator ConditionLocator) Exists() DeclarationExpectation {
	value := conditionExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located Condition is absent from a
// complete relevant public declaration domain; incomplete observation cannot manufacture
// absence.
func (locator ConditionLocator) DoesNotExist() DeclarationExpectation {
	value := conditionExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Effect returns an expectation that checks the Condition has the requested declared effect.
func (locator ConditionLocator) Effect(v cli.ConditionEffect) DeclarationExpectation {
	value := conditionEffectExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Reachability returns an expectation that checks the Condition has the requested declared
// reachability.
func (locator ConditionLocator) Reachability(v cli.Reachability) DeclarationExpectation {
	value := conditionReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Outcome returns an expectation that checks the Condition's invocation-time
// EvaluationOutcome.
func (locator ConditionLocator) Outcome(outcome cli.EvaluationOutcome) InspectionExpectation {
	return conditionOutcomeExpectation(locator, outcome)
}

// Target returns an expectation that checks the Condition targets the exact heterogeneous
// declaration identified by the supplied locator.
func (locator ConditionLocator) Target(v DeclarationIdentityLocator) DeclarationExpectation {
	value := conditionTargetExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Sensitive returns an expectation that checks the inspected Condition is sensitive; mismatch
// reporting preserves redaction.
func (locator ConditionLocator) Sensitive() InspectionExpectation {
	return conditionSensitiveExpectation(locator)
}

// ChildAt returns a locator for the child at the supplied public ordered index of the located
// condition expression. It does not search for or silently choose a first match.
func (locator ConditionExpressionLocator) ChildAt(index int) ConditionExpressionLocator {
	state := &conditionExpressionLocatorState{
		mode:   conditionExpressionLocatorChild,
		parent: locator.state,
		index:  index,
	}
	conditionExpressionLocator := ConditionExpressionLocator{
		state: state,
	}
	return conditionExpressionLocator
}

// OperandAt returns a locator for the operand at the supplied public ordered index of the
// located condition expression. It does not search for or silently choose a first match.
func (locator ConditionExpressionLocator) OperandAt(index int) DeclarationValueLiteralLocator {
	state := &declarationValueLiteralLocatorState{
		mode:      declarationValueLiteralLocatorConditionOperand,
		condition: locator.state,
		index:     index,
	}
	declarationValueLiteralLocator := DeclarationValueLiteralLocator{
		state: state,
	}
	return declarationValueLiteralLocator
}

// Exists returns a declaration expectation that checks the condition-expression relationship is
// present. Presence may be satisfied by truthful partial output when the subject is available.
func (locator ConditionExpressionLocator) Exists() DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:  locator.state,
		property: conditionExpressionExpectationExists,
		exists:   true,
	}
	return expectation
}

// DoesNotExist returns a declaration expectation that checks the condition-expression
// relationship is absent. Omission in a truncated or otherwise incomplete observation is not
// treated as expectation of absence.
func (locator ConditionExpressionLocator) DoesNotExist() DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:  locator.state,
		property: conditionExpressionExpectationExists,
	}
	return expectation
}

// Kind returns a reusable declaration expectation that checks the public kind exposed by the
// located condition expression.
func (locator ConditionExpressionLocator) Kind(expected cli.ConditionExpressionKind) DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:  locator.state,
		property: conditionExpressionExpectationKind,
		kind:     expected,
	}
	return expectation
}

// Pattern returns a reusable declaration expectation that checks the public pattern exposed by
// the located condition expression.
func (locator ConditionExpressionLocator) Pattern(expected string) DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:  locator.state,
		property: conditionExpressionExpectationPattern,
		pattern:  expected,
	}
	return expectation
}

// ProducerKindsExactly returns a reusable declaration expectation that checks the complete producer-kind
// list exactly in authoritative public order.
func (locator ConditionExpressionLocator) ProducerKindsExactly(expected ...cli.ValueProducerKind) DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:       locator.state,
		property:      conditionExpressionExpectationProducerKinds,
		producerKinds: slices.Clone(expected),
	}
	return expectation
}

// ChildCount returns a reusable declaration expectation that checks the child count exposed by
// the located condition expression.
func (locator ConditionExpressionLocator) ChildCount(expected int) DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:  locator.state,
		property: conditionExpressionExpectationChildCount,
		count:    expected,
	}
	return expectation
}

// OperandCount returns a reusable declaration expectation that checks the operand count exposed
// by the located condition expression.
func (locator ConditionExpressionLocator) OperandCount(expected int) DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:  locator.state,
		property: conditionExpressionExpectationOperandCount,
		count:    expected,
	}
	return expectation
}

// Sensitive returns a reusable declaration expectation that checks the located condition
// expression is marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator ConditionExpressionLocator) Sensitive() DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:   locator.state,
		property:  conditionExpressionExpectationSensitive,
		sensitive: true,
	}
	return expectation
}

// NotSensitive returns a reusable declaration expectation that checks the located condition
// expression is not marked sensitive.
func (locator ConditionExpressionLocator) NotSensitive() DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:  locator.state,
		property: conditionExpressionExpectationSensitive,
	}
	return expectation
}

// Destination returns a declaration expectation that checks the expression's public destination
// relationship resolves to the same value identity as the expected ValueLocator; textual
// similarity is not substituted for identity.
func (locator ConditionExpressionLocator) Destination(expected ValueLocator) DeclarationExpectation {
	expectation := conditionExpressionExpectation{
		locator:     locator.state,
		property:    conditionExpressionExpectationDestination,
		destination: expected.state,
	}
	return expectation
}

type conditionExpressionLocatorMode uint8

const (
	conditionExpressionLocatorInvalid conditionExpressionLocatorMode = iota
	conditionExpressionLocatorCommandDefault
	conditionExpressionLocatorCondition
	conditionExpressionLocatorRuleActivation
	conditionExpressionLocatorAdvisoryActivation
	conditionExpressionLocatorNoticeSuppression
	conditionExpressionLocatorSignalPolicyActivation
	conditionExpressionLocatorChild
)

type conditionExpressionLocatorState struct {
	mode              conditionExpressionLocatorMode
	commandDefault    *commandDefaultLocatorState
	condition         *conditionLocatorState
	rule              *ruleLocatorState
	advisory          *advisoryLocatorState
	noticeSuppression *noticeSuppressionLocatorState
	signalPolicy      *signalPolicyLocatorState
	parent            *conditionExpressionLocatorState
	index             int
}

type conditionExpressionExpectationProperty uint8

const (
	conditionExpressionExpectationInvalid conditionExpressionExpectationProperty = iota
	conditionExpressionExpectationExists
	conditionExpressionExpectationKind
	conditionExpressionExpectationPattern
	conditionExpressionExpectationProducerKinds
	conditionExpressionExpectationChildCount
	conditionExpressionExpectationOperandCount
	conditionExpressionExpectationSensitive
	conditionExpressionExpectationDestination
)

type conditionExpressionExpectation struct {
	locator       *conditionExpressionLocatorState
	property      conditionExpressionExpectationProperty
	exists        bool
	kind          cli.ConditionExpressionKind
	pattern       string
	producerKinds []cli.ValueProducerKind
	count         int
	sensitive     bool
	destination   *valueLocatorState
}

type commandConditionsExpectation struct {
	command *commandLocatorState
	items   []*conditionLocatorState
}

type conditionLocatorState struct {
	command *commandLocatorState
	index   int
}

type conditionExistenceExpectationValue struct {
	locator  *conditionLocatorState
	expected bool
}

type conditionEffectExpectationValue struct {
	locator  *conditionLocatorState
	expected cli.ConditionEffect
}

type conditionReachabilityExpectationValue struct {
	locator  *conditionLocatorState
	expected cli.Reachability
}

type conditionTargetExpectationValue struct {
	locator  *conditionLocatorState
	expected DeclarationIdentityLocator
}

type conditionOutcomeExpectationValue struct {
	locator  *conditionLocatorState
	expected cli.EvaluationOutcome
}

type conditionSensitiveExpectationValue struct {
	locator *conditionLocatorState
}

func conditionFromPointer(value *cli.ConditionExpression) (cli.ConditionExpression, locatorResolutionStatus, evaluationField) {
	if value == nil {
		return cli.ConditionExpression{}, locatorResolutionAbsent, evaluationText("condition expression is absent")
	}
	return *value, locatorResolutionFound, evaluationText("")
}

func conditionExpressionLabel(state *conditionExpressionLocatorState) evaluationField {
	if state == nil {
		return evaluationText("condition expression")
	}
	switch state.mode {
	case conditionExpressionLocatorCommandDefault:
		return qualifySubject(commandDefaultLocatorLabel(state.commandDefault), evaluationText("condition expression"))
	case conditionExpressionLocatorCondition:
		return qualifySubject(conditionLocatorLabel(state.condition), evaluationText("condition expression"))
	case conditionExpressionLocatorRuleActivation:
		return qualifySubject(ruleLocatorLabel(state.rule), evaluationText("activation"))
	case conditionExpressionLocatorAdvisoryActivation:
		return qualifySubject(advisoryLocatorLabel(state.advisory), evaluationText("activation"))
	case conditionExpressionLocatorNoticeSuppression:
		return qualifySubject(noticeSuppressionLocatorLabel(state.noticeSuppression), evaluationText("condition expression"))
	case conditionExpressionLocatorSignalPolicyActivation:
		return qualifySubject(signalPolicyLocatorLabel(state.signalPolicy), evaluationText("activation"))
	case conditionExpressionLocatorChild:
		return qualifySubject(conditionExpressionLabel(state.parent), subjectAt(semanticSubjectConditionChild, state.index))
	default:
		return evaluationText("condition expression")
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveSchemaConditionExpressionLocator(state *conditionExpressionLocatorState, schema *cli.Schema) locatorResolution[cli.ConditionExpression] {
	var resolution locatorResolution[cli.ConditionExpression]
	label := conditionExpressionLabel(state)
	if state == nil || state.mode == conditionExpressionLocatorInvalid {
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	var value cli.ConditionExpression
	var status locatorResolutionStatus
	var detail evaluationField
	switch state.mode {
	case conditionExpressionLocatorCommandDefault:
		resolved := resolveSchemaCommandDefaultLocator(state.commandDefault, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Condition)
	case conditionExpressionLocatorCondition:
		resolved := resolveSchemaConditionLocator(state.condition, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = resolved.value.Expression, locatorResolutionFound, evaluationText("")
	case conditionExpressionLocatorRuleActivation:
		resolved := resolveSchemaRuleLocator(state.rule, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorAdvisoryActivation:
		resolved := resolveSchemaAdvisoryLocator(state.advisory, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorNoticeSuppression:
		resolved := resolveSchemaNoticeSuppressionLocator(state.noticeSuppression, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = resolved.value.Condition, locatorResolutionFound, evaluationText("")
	case conditionExpressionLocatorSignalPolicyActivation:
		resolved := resolveSchemaSignalPolicyLocator(state.signalPolicy, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorChild:
		parent := resolveSchemaConditionExpressionLocator(state.parent, schema)
		return resolveConditionExpressionChild(state, parent)
	default:
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ConditionExpression]{
		status:  status,
		value:   value,
		subject: label,
		detail:  detail,
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveReferenceConditionExpressionLocator(state *conditionExpressionLocatorState, reference cli.Reference) locatorResolution[cli.ConditionExpression] {
	var resolution locatorResolution[cli.ConditionExpression]
	label := conditionExpressionLabel(state)
	if state == nil || state.mode == conditionExpressionLocatorInvalid {
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	var value cli.ConditionExpression
	var status locatorResolutionStatus
	var detail evaluationField
	switch state.mode {
	case conditionExpressionLocatorCommandDefault:
		resolved := resolveReferenceCommandDefaultLocator(state.commandDefault, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Condition)
	case conditionExpressionLocatorCondition:
		resolved := resolveReferenceConditionLocator(state.condition, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = resolved.value.Expression, locatorResolutionFound, evaluationText("")
	case conditionExpressionLocatorRuleActivation:
		resolved := resolveReferenceRuleLocator(state.rule, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorAdvisoryActivation:
		resolved := resolveReferenceAdvisoryLocator(state.advisory, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorNoticeSuppression:
		resolved := resolveReferenceNoticeSuppressionLocator(state.noticeSuppression, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = resolved.value.Condition, locatorResolutionFound, evaluationText("")
	case conditionExpressionLocatorSignalPolicyActivation:
		resolved := resolveReferenceSignalPolicyLocator(state.signalPolicy, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorChild:
		parent := resolveReferenceConditionExpressionLocator(state.parent, reference)
		return resolveConditionExpressionChild(state, parent)
	default:
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ConditionExpression]{
		status:  status,
		value:   value,
		subject: label,
		detail:  detail,
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveInspectionConditionExpressionLocator(state *conditionExpressionLocatorState, inspection cli.Inspection) locatorResolution[cli.ConditionExpression] {
	var resolution locatorResolution[cli.ConditionExpression]
	label := conditionExpressionLabel(state)
	if state == nil || state.mode == conditionExpressionLocatorInvalid {
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	var value cli.ConditionExpression
	var status locatorResolutionStatus
	var detail evaluationField
	switch state.mode {
	case conditionExpressionLocatorCommandDefault:
		resolved := resolveInspectionCommandDefaultLocator(state.commandDefault, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Condition)
	case conditionExpressionLocatorCondition:
		resolved := resolveInspectionConditionLocator(state.condition, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = resolved.value.Expression, locatorResolutionFound, evaluationText("")
	case conditionExpressionLocatorRuleActivation:
		resolved := resolveInspectionRuleLocator(state.rule, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorAdvisoryActivation:
		resolved := resolveInspectionAdvisoryLocator(state.advisory, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorNoticeSuppression:
		resolved := resolveInspectionNoticeSuppressionLocator(state.noticeSuppression, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = resolved.value.Condition, locatorResolutionFound, evaluationText("")
	case conditionExpressionLocatorSignalPolicyActivation:
		resolved := resolveInspectionSignalPolicyLocator(state.signalPolicy, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ConditionExpression]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		value, status, detail = conditionFromPointer(resolved.value.Activation)
	case conditionExpressionLocatorChild:
		parent := resolveInspectionConditionExpressionLocator(state.parent, inspection)
		return resolveConditionExpressionChild(state, parent)
	default:
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ConditionExpression]{
		status:  status,
		value:   value,
		subject: label,
		detail:  detail,
	}
	return resolution
}

//nolint:dupl // Condition-expression children keep their own typed bounds and absence proof.
func resolveConditionExpressionChild(state *conditionExpressionLocatorState, parent locatorResolution[cli.ConditionExpression]) locatorResolution[cli.ConditionExpression] {
	var resolution locatorResolution[cli.ConditionExpression]
	label := conditionExpressionLabel(state)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	if state.index < 0 {
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectConditionChild, observationInvalidLocator),
		}
		return resolution
	}
	if state.index >= len(parent.value.Children) {
		resolution = locatorResolution[cli.ConditionExpression]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectConditionChild, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ConditionExpression]{
		status:  locatorResolutionFound,
		value:   parent.value.Children[state.index],
		subject: label,
	}
	return resolution
}

//nolint:dupl // Condition operand literals keep their own typed bounds and absence proof.
func resolveConditionOperandLiteral(state *declarationValueLiteralLocatorState, parent locatorResolution[cli.ConditionExpression]) locatorResolution[cli.ValueLiteral] {
	var resolution locatorResolution[cli.ValueLiteral]
	label := declarationValueLiteralLabel(state)
	if parent.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  parent.status,
			subject: label,
			detail:  parent.detail,
		}
		return resolution
	}
	if state.index < 0 {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectConditionOperand, observationInvalidLocator),
		}
		return resolution
	}
	if state.index >= len(parent.value.Operands) {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectConditionOperand, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueLiteral]{
		status:  locatorResolutionFound,
		value:   parent.value.Operands[state.index],
		subject: label,
	}
	return resolution
}

func conditionLocatorLabel(state *conditionLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectCondition
	}
	return qualifySubject(commandLocatorLabel(state.command), subjectAt(semanticSubjectCondition, state.index))
}

//nolint:dupl // Distinct typed policy families intentionally retain direct resolution logic.
func resolveInspectionConditionLocator(state *conditionLocatorState, inspection cli.Inspection) locatorResolution[cli.ConditionInspection] {
	var resolution locatorResolution[cli.ConditionInspection]
	label := conditionLocatorLabel(state)
	if state == nil || state.command == nil || state.index < 0 {
		resolution = locatorResolution[cli.ConditionInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCondition, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ConditionInspection]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index < len(command.value.Conditions) {
		resolution = locatorResolution[cli.ConditionInspection]{
			status:  locatorResolutionFound,
			value:   command.value.Conditions[state.index],
			subject: label,
		}
		return resolution
	}
	if detail, incomplete := policyInspectionNamespaceIncomplete(command.value, inspection); incomplete {
		resolution = locatorResolution[cli.ConditionInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.ConditionInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectCondition, observationAbsent),
	}
	return resolution
}

func resolveSchemaConditionLocator(state *conditionLocatorState, schema *cli.Schema) locatorResolution[cli.ConditionSchema] {
	var resolution locatorResolution[cli.ConditionSchema]
	label := conditionLocatorLabel(state)
	if schema == nil || state == nil || state.command == nil || state.index < 0 {
		resolution = locatorResolution[cli.ConditionSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCondition, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ConditionSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index >= len(command.value.Conditions) {
		resolution = locatorResolution[cli.ConditionSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCondition, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ConditionSchema]{
		status:  locatorResolutionFound,
		value:   command.value.Conditions[state.index],
		subject: label,
	}
	return resolution
}

func resolveReferenceConditionLocator(state *conditionLocatorState, reference cli.Reference) locatorResolution[cli.ConditionSchema] {
	var resolution locatorResolution[cli.ConditionSchema]
	label := conditionLocatorLabel(state)
	if state == nil || state.command == nil || state.index < 0 {
		resolution = locatorResolution[cli.ConditionSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCondition, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.ConditionSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	if state.index >= len(command.value.Conditions) {
		resolution = locatorResolution[cli.ConditionSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectCondition, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ConditionSchema]{
		status:  locatorResolutionFound,
		value:   command.value.Conditions[state.index],
		subject: label,
	}
	return resolution
}

func conditionLocatorStates(values []ConditionLocator) []*conditionLocatorState {
	states := make([]*conditionLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

func compareConditionStatesSchema(command cli.CommandSchema, states []*conditionLocatorState, schema *cli.Schema, label evaluationField) evaluationResult {
	if len(command.Conditions) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("conditions count %d", len(states))), evaluationText(fmt.Sprintf("conditions count %d", len(command.Conditions))))
	}
	for index, state := range states {
		if state == nil || state.index != index || !validateSchemaPolicyParent(state.command, command, schema) {
			return disprovedResult(label, evaluationText("ordered condition locators"), evaluationText(fmt.Sprintf("invalid condition locator at position %d", index)))
		}
	}
	return provedResult()
}

func compareConditionStatesReference(command cli.CommandReference, states []*conditionLocatorState, reference cli.Reference, label evaluationField) evaluationResult {
	if len(command.Conditions) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("conditions count %d", len(states))), evaluationText(fmt.Sprintf("conditions count %d", len(command.Conditions))))
	}
	for index, state := range states {
		if state == nil || state.index != index || !validateReferencePolicyParent(state.command, command, reference) {
			return disprovedResult(label, evaluationText("ordered condition locators"), evaluationText(fmt.Sprintf("invalid condition locator at position %d", index)))
		}
	}
	return provedResult()
}

func compareConditionStatesInspection(command cli.CommandInspection, states []*conditionLocatorState, inspection cli.Inspection, label evaluationField) evaluationResult {
	if len(command.Conditions) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("conditions count %d", len(states))), evaluationText(fmt.Sprintf("conditions count %d", len(command.Conditions))))
	}
	for index, state := range states {
		if state == nil || state.index != index {
			return disprovedResult(label, evaluationText("ordered condition locators"), evaluationText(fmt.Sprintf("invalid condition locator at position %d", index)))
		}
		if outcome, ok := validateInspectionPolicyParent(state.command, command, inspection, label, evaluationText("ordered condition locators")); !ok {
			return outcome
		}
	}
	return provedResult()
}

func conditionIndexLocator(command CommandLocator, index int) ConditionLocator {
	state := &conditionLocatorState{
		command: command.state,
		index:   index,
	}

	locator := ConditionLocator{
		state: state,
	}
	return locator
}

func conditionOutcomeExpectation(locator ConditionLocator, outcome cli.EvaluationOutcome) InspectionExpectation {
	value := conditionOutcomeExpectationValue{
		locator:  locator.state,
		expected: outcome,
	}
	return value
}

func conditionSensitiveExpectation(locator ConditionLocator) InspectionExpectation {
	value := conditionSensitiveExpectationValue{
		locator: locator.state,
	}
	return value
}

func (conditionExpressionExpectation) schemaExpectation() {}

func (conditionExpressionExpectation) referenceExpectation() {}

func (conditionExpressionExpectation) inspectionExpectation() {}

func (expectation conditionExpressionExpectation) evaluateResolved(resolution locatorResolution[cli.ConditionExpression], truncated bool) evaluationResult {
	subject := evaluationText("condition expression")

	if expectation.property == conditionExpressionExpectationExists {
		if expectation.exists {
			if resolution.status == locatorResolutionFound {
				return provedResult()
			}
			return missingResult(subject, evaluationText("exists"), resolution.detail, truncated)
		}
		switch resolution.status {
		case locatorResolutionAbsent:
			if truncated {
				return notProvableResult(subject, semanticRequirementAbsent, observed(semanticSubjectInspection, observationTruncated))
			}
			return provedResult()
		case locatorResolutionFound:
			return disprovedResult(subject, semanticRequirementAbsent, evaluationText("present"))
		case locatorResolutionNotProvable:
			return notProvableResult(subject, semanticRequirementAbsent, resolution.detail)
		default:
			return disprovedResult(subject, semanticRequirementAbsent, evaluationText("invalid or ambiguous locator"))
		}
	}

	if resolution.status != locatorResolutionFound {
		return missingResult(subject, expectation.propertyLabel(), resolution.detail, truncated)
	}

	value := resolution.value
	switch expectation.property {
	case conditionExpressionExpectationKind:
		return equalFieldResult(subject, evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(value.Kind)), value.Kind == expectation.kind)
	case conditionExpressionExpectationPattern:
		return equalFieldResult(subject, evaluationText("pattern"), evaluationText(expectation.pattern), evaluationText(value.Pattern), value.Pattern == expectation.pattern)
	case conditionExpressionExpectationProducerKinds:
		if slices.Equal(value.ProducerKinds, expectation.producerKinds) {
			return provedResult()
		}
		return orderedCollectionMismatch(subject, "producer kinds", expectation.producerKinds, value.ProducerKinds)
	case conditionExpressionExpectationChildCount:
		return equalFieldResult(subject, evaluationText("child-count"), evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(value.Children))), len(value.Children) == expectation.count)
	case conditionExpressionExpectationOperandCount:
		return equalFieldResult(subject, evaluationText("operand-count"), evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(value.Operands))), len(value.Operands) == expectation.count)
	case conditionExpressionExpectationSensitive:
		return equalFieldResult(subject, semanticRequirementSensitive, evaluationBool(expectation.sensitive), evaluationBool(value.Sensitive), value.Sensitive == expectation.sensitive)
	default:
		panic("clitest: invalid condition-expression expectation")
	}
}

func (expectation conditionExpressionExpectation) propertyLabel() evaluationField {
	switch expectation.property {
	case conditionExpressionExpectationKind:
		return evaluationText("kind")
	case conditionExpressionExpectationPattern:
		return evaluationText("pattern")
	case conditionExpressionExpectationProducerKinds:
		return evaluationText("producer kinds")
	case conditionExpressionExpectationChildCount:
		return evaluationText("child-count")
	case conditionExpressionExpectationOperandCount:
		return evaluationText("operand-count")
	case conditionExpressionExpectationSensitive:
		return semanticRequirementSensitive
	case conditionExpressionExpectationDestination:
		return evaluationText("destination")
	default:
		return evaluationText("condition expression")
	}
}

func (expectation conditionExpressionExpectation) evaluateDestination(resolution locatorResolution[cli.ConditionExpression], expected cli.ValueID, expectedStatus locatorResolutionStatus, truncated bool) evaluationResult {
	subject := evaluationText("condition expression")
	if resolution.status != locatorResolutionFound {
		return missingResult(subject, evaluationText("destination"), resolution.detail, truncated)
	}
	if expectedStatus != locatorResolutionFound {
		return relationLocatorFailureOutcome("condition expression", "destination", "expected value locator", expectedStatus)
	}
	if resolution.value.Destination == expected {
		return provedResult()
	}
	return disprovedResult(subject, evaluationText("destination matches expected value identity"), evaluationText("destination has different value identity"))
}

func (expectation conditionExpressionExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaConditionExpressionLocator(expectation.locator, schema)
	if expectation.property != conditionExpressionExpectationDestination {
		return expectation.evaluateResolved(resolution, false)
	}
	expected, status, _ := resolveSchemaValueID(expectation.destination, schema)
	return expectation.evaluateDestination(resolution, expected, status, false)
}

func (expectation conditionExpressionExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceConditionExpressionLocator(expectation.locator, reference)
	if expectation.property != conditionExpressionExpectationDestination {
		return expectation.evaluateResolved(resolution, false)
	}
	expected, status, _ := resolveReferenceValueID(expectation.destination, reference)
	return expectation.evaluateDestination(resolution, expected, status, false)
}

func (expectation conditionExpressionExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionConditionExpressionLocator(expectation.locator, inspection)
	if expectation.property != conditionExpressionExpectationDestination {
		return expectation.evaluateResolved(resolution, inspection.Truncated)
	}
	expected, status, _ := resolveInspectionValueID(expectation.destination, inspection)
	return expectation.evaluateDestination(resolution, expected, status, inspection.Truncated)
}

func (commandConditionsExpectation) schemaExpectation() {}

func (commandConditionsExpectation) referenceExpectation() {}

func (commandConditionsExpectation) inspectionExpectation() {}

func (expectation commandConditionsExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("conditions exactly"), command.detail)
	}
	return compareConditionStatesSchema(command.value, expectation.items, schema, qualifySubject(command.subject, evaluationText("conditions")))
}

func (expectation commandConditionsExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("conditions exactly"), command.detail)
	}
	return compareConditionStatesReference(command.value, expectation.items, reference, qualifySubject(command.subject, evaluationText("conditions")))
}

func (expectation commandConditionsExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	label, failure, ok := exactPolicyCollectionInspection(command, inspection, "conditions")
	if !ok {
		return failure
	}
	return compareConditionStatesInspection(command.value, expectation.items, inspection, label)
}

func (conditionExistenceExpectationValue) schemaExpectation() {}

func (conditionExistenceExpectationValue) referenceExpectation() {}

func (conditionExistenceExpectationValue) inspectionExpectation() {}

func (expectation conditionExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaConditionLocator(expectation.locator, schema), expectation.expected)
}

func (expectation conditionExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceConditionLocator(expectation.locator, reference), expectation.expected)
}

func (expectation conditionExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionConditionLocator(expectation.locator, inspection), expectation.expected)
}

func (conditionEffectExpectationValue) schemaExpectation() {}

func (conditionEffectExpectationValue) referenceExpectation() {}

func (conditionEffectExpectationValue) inspectionExpectation() {}

func (expectation conditionEffectExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaConditionLocator(expectation.locator, schema)
	actual := resolution.value.Effect
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("effect %v", expectation.expected)), evaluationText(fmt.Sprintf("effect %v", actual)), actual == expectation.expected)
}

func (expectation conditionEffectExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceConditionLocator(expectation.locator, reference)
	actual := resolution.value.Effect
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("effect %v", expectation.expected)), evaluationText(fmt.Sprintf("effect %v", actual)), actual == expectation.expected)
}

func (expectation conditionEffectExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionConditionLocator(expectation.locator, inspection)
	actual := resolution.value.Effect
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("effect %v", expectation.expected)), evaluationText(fmt.Sprintf("effect %v", actual)), actual == expectation.expected)
}

func (conditionReachabilityExpectationValue) schemaExpectation() {}

func (conditionReachabilityExpectationValue) referenceExpectation() {}

func (conditionReachabilityExpectationValue) inspectionExpectation() {}

func (expectation conditionReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaConditionLocator(expectation.locator, schema)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation conditionReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceConditionLocator(expectation.locator, reference)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation conditionReachabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionConditionLocator(expectation.locator, inspection)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (conditionTargetExpectationValue) schemaExpectation() {}

func (conditionTargetExpectationValue) referenceExpectation() {}

func (conditionTargetExpectationValue) inspectionExpectation() {}

func (expectation conditionTargetExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaConditionLocator(expectation.locator, schema)
	return declarationIdentityRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Target, expectation.expected, schema, nil, cli.Inspection{})
}

func (expectation conditionTargetExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceConditionLocator(expectation.locator, reference)
	return declarationIdentityRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Target, expectation.expected, nil, &reference, cli.Inspection{})
}

func (expectation conditionTargetExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionConditionLocator(expectation.locator, inspection)
	return declarationIdentityRelationResult(resolution.status, resolution.subject, resolution.detail, resolution.value.Target, expectation.expected, nil, nil, inspection)
}

func (conditionOutcomeExpectationValue) inspectionExpectation() {}

//nolint:dupl // Condition and rule outcome families remain direct typed evaluators.
func (expectation conditionOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionConditionLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Outcome)), resolution.value.Outcome == expectation.expected)
	}
	if resolution.value.Outcome == expectation.expected {
		return provedResult()
	}
	if resolution.value.Sensitive {
		return disprovedResult(
			qualifySubject(resolution.subject, evaluationText("outcome")),
			evaluationText("redacted expected condition property"),
			evaluationText("redacted observed condition property"),
		)
	}
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Outcome)), resolution.value.Outcome == expectation.expected)
}

func (conditionSensitiveExpectationValue) inspectionExpectation() {}

func (expectation conditionSensitiveExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionConditionLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, semanticRequirementSensitive, evaluationText(""), false)
	}
	if resolution.value.Sensitive {
		return provedResult()
	}
	return disprovedResult(resolution.subject, semanticRequirementSensitive, evaluationText("sensitive=false"))
}
