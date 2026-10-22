package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// RuleOperandLocator identifies one public Rule operand by its authoritative operand index. It
// follows the Rule's exposed operand relationship, does not create independent semantic
// identity, and is invalid when zero-valued.
type RuleOperandLocator struct {
	rule  RuleLocator
	index int
}

// RuleLocator identifies one Rule for clitest semantic expectation. Its zero value is
// invalid.
type RuleLocator struct {
	_     [0]func()
	state *ruleLocatorState
}

// Rule returns a locator for the uniquely resolved rule classified by code under the located
// command; a code is not substituted for rule identity when several rules share it.
func (l CommandLocator) Rule(code cli.RuleCode) RuleLocator {
	return ruleCodeLocator(l, code)
}

// RuleAt returns a locator for the rule at index in the located command's authoritative public
// rule order.
func (l CommandLocator) RuleAt(index int) RuleLocator { return ruleIndexLocator(l, index) }

// RulesExactly returns an expectation that checks the located command's rules are exactly the
// supplied locators in authoritative public order. Exactness is not inferred from an incomplete
// observation.
func (l CommandLocator) RulesExactly(values ...RuleLocator) DeclarationExpectation {
	expectation := commandRulesExpectation{
		command: l.state,
		items:   ruleLocatorStates(values),
	}
	return expectation
}

// Activation returns a locator that follows the public activation relationship from the located
// Rule. The returned locator carries locator coordinates, not observed state.
func (locator RuleLocator) Activation() ConditionExpressionLocator {
	state := &conditionExpressionLocatorState{
		mode: conditionExpressionLocatorRuleActivation,
		rule: locator.state,
	}
	conditionExpressionLocator := ConditionExpressionLocator{
		state: state,
	}
	return conditionExpressionLocator
}

// Pattern returns a reusable declaration expectation that checks the public pattern exposed by
// the located Rule.
func (locator RuleLocator) Pattern(expected string) DeclarationExpectation {
	value := rulePatternExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// Count returns a reusable declaration expectation that checks the count exposed by the located
// Rule.
func (locator RuleLocator) Count(expected cli.CountRange) DeclarationExpectation {
	value := ruleCountExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// OperandCount returns a reusable declaration expectation that checks the operand count exposed
// by the located Rule.
func (locator RuleLocator) OperandCount(expected int) DeclarationExpectation {
	value := ruleOperandCountExpectationValue{
		locator:  locator.state,
		expected: expected,
	}
	return value
}

// OperandAt returns a locator for the operand at the supplied public ordered index of the
// located Rule. It does not search for or silently choose a first match.
func (locator RuleLocator) OperandAt(index int) RuleOperandLocator {
	ruleOperandLocator := RuleOperandLocator{
		rule:  locator,
		index: index,
	}
	return ruleOperandLocator
}

// Exists returns an expectation that checks the located Rule exists in the relevant public
// declaration observation.
func (locator RuleLocator) Exists() DeclarationExpectation {
	value := ruleExistenceExpectationValue{
		locator:  locator.state,
		expected: true,
	}
	return value
}

// DoesNotExist returns an expectation that checks the located Rule is absent from a complete
// relevant public declaration domain; incomplete observation cannot manufacture absence.
func (locator RuleLocator) DoesNotExist() DeclarationExpectation {
	value := ruleExistenceExpectationValue{
		locator:  locator.state,
		expected: false,
	}
	return value
}

// Kind returns an expectation that checks the Rule has the requested declared RuleKind.
func (locator RuleLocator) Kind(v cli.RuleKind) DeclarationExpectation {
	value := ruleKindExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Code returns an expectation that checks the Rule carries the requested RuleCode; the code is
// metadata, not Rule identity.
func (locator RuleLocator) Code(v cli.RuleCode) DeclarationExpectation {
	value := ruleCodeExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Reachability returns an expectation that checks the Rule has the requested declared
// reachability.
func (locator RuleLocator) Reachability(v cli.Reachability) DeclarationExpectation {
	value := ruleReachabilityExpectationValue{
		locator:  locator.state,
		expected: v,
	}
	return value
}

// Outcome returns an expectation that checks the Rule's invocation-time EvaluationOutcome.
func (locator RuleLocator) Outcome(outcome cli.EvaluationOutcome) InspectionExpectation {
	return ruleOutcomeExpectation(locator, outcome)
}

// ActivationOutcome returns an expectation that checks the Rule's optional activation
// evaluation outcome.
func (locator RuleLocator) ActivationOutcome(outcome cli.EvaluationOutcome) InspectionExpectation {
	return ruleActivationOutcomeExpectation(locator, outcome)
}

// SubjectsExactly returns an expectation that checks the Rule declares exactly the requested
// value subjects in authoritative order.
func (locator RuleLocator) SubjectsExactly(values ...ValueLocator) DeclarationExpectation {
	states := make([]*valueLocatorState, len(values))
	for i, value := range values {
		states[i] = value.state
	}
	expectation := ruleValuesExpectationValue{
		locator:  locator.state,
		expected: states,
	}
	return expectation
}

// DependenciesExactly returns an expectation that checks the Rule declares exactly the
// requested value dependencies in authoritative order.
func (locator RuleLocator) DependenciesExactly(values ...ValueLocator) DeclarationExpectation {
	states := make([]*valueLocatorState, len(values))
	for i, value := range values {
		states[i] = value.state
	}
	expectation := ruleValuesExpectationValue{
		locator:      locator.state,
		expected:     states,
		dependencies: true,
	}
	return expectation
}

// Sensitive returns an expectation that checks the Rule is declared sensitive; expectation failures
// do not expose sensitive Rule data.
func (locator RuleLocator) Sensitive() DeclarationExpectation {
	value := ruleSensitiveExpectationValue{
		locator: locator.state,
	}
	return value
}

// Sensitive returns a reusable declaration expectation that checks the located Rule operand is
// marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator RuleOperandLocator) Sensitive() DeclarationExpectation {
	expectation := ruleOperandSensitiveExpectation{
		rule:  locator.rule.state,
		index: locator.index,
	}
	return expectation
}

// Literal returns a locator that follows the public literal relationship from the located Rule
// operand. The returned locator carries locator coordinates, not observed state.
func (locator RuleOperandLocator) Literal() DeclarationValueLiteralLocator {
	return ruleOperandLiteral(locator)
}

// NoLiteral returns a reusable declaration expectation that checks the located Rule operand
// explicitly has no literal.
func (locator RuleOperandLocator) NoLiteral() DeclarationExpectation {
	expectation := ruleOperandNoLiteralExpectation{
		locator: ruleOperandLiteral(locator).state,
	}
	return expectation
}

type ruleOperandSensitiveExpectation struct {
	rule  *ruleLocatorState
	index int
}

type ruleOperandNoLiteralExpectation struct {
	locator *declarationValueLiteralLocatorState
}

type commandRulesExpectation struct {
	command *commandLocatorState
	items   []*ruleLocatorState
}

type ruleLocatorMode uint8

const (
	ruleLocatorInvalid ruleLocatorMode = iota
	ruleLocatorCode
	ruleLocatorIndex
)

type ruleLocatorState struct {
	mode ruleLocatorMode

	command *commandLocatorState
	code    cli.RuleCode
	index   int
}

type ruleExistenceExpectationValue struct {
	locator  *ruleLocatorState
	expected bool
}

type ruleKindExpectationValue struct {
	locator  *ruleLocatorState
	expected cli.RuleKind
}

type ruleCodeExpectationValue struct {
	locator  *ruleLocatorState
	expected cli.RuleCode
}

type ruleReachabilityExpectationValue struct {
	locator  *ruleLocatorState
	expected cli.Reachability
}

type ruleSensitiveExpectationValue struct{ locator *ruleLocatorState }

type ruleValuesExpectationValue struct {
	locator      *ruleLocatorState
	expected     []*valueLocatorState
	dependencies bool
}

type rulePatternExpectationValue struct {
	locator  *ruleLocatorState
	expected string
}

type ruleCountExpectationValue struct {
	locator  *ruleLocatorState
	expected cli.CountRange
}

type ruleOperandCountExpectationValue struct {
	locator  *ruleLocatorState
	expected int
}

type ruleOutcomeExpectationValue struct {
	locator  *ruleLocatorState
	expected cli.EvaluationOutcome
}

type ruleActivationOutcomeExpectationValue struct {
	locator  *ruleLocatorState
	expected cli.EvaluationOutcome
}

func ruleOperandLiteral(locator RuleOperandLocator) DeclarationValueLiteralLocator {
	state := &declarationValueLiteralLocatorState{
		mode:  declarationValueLiteralLocatorRuleOperand,
		rule:  locator.rule.state,
		index: locator.index,
	}
	declarationValueLiteralLocator := DeclarationValueLiteralLocator{
		state: state,
	}
	return declarationValueLiteralLocator
}

func resolveRuleOperandLiteral(state *declarationValueLiteralLocatorState, status locatorResolutionStatus, detail evaluationField, operands []cli.RuleOperand) locatorResolution[cli.ValueLiteral] {
	var resolution locatorResolution[cli.ValueLiteral]
	label := declarationValueLiteralLabel(state)
	if status != locatorResolutionFound {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  status,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	if state.index < 0 {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectRuleOperand, observationInvalidLocator),
		}
		return resolution
	}
	if state.index >= len(operands) {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectRuleOperand, observationAbsent),
		}
		return resolution
	}
	return resolveOptionalLiteral(label, operands[state.index].Literal, semanticSubjectRuleOperandLiteral)
}

func ruleLocatorLabel(state *ruleLocatorState) evaluationField {
	if state == nil {
		return semanticSubjectRule
	}
	switch state.mode {
	case ruleLocatorCode:
		return qualifySubject(commandLocatorLabel(state.command), subjectNamed(semanticSubjectRule, string(state.code)))
	case ruleLocatorIndex:
		return qualifySubject(commandLocatorLabel(state.command), subjectAt(semanticSubjectRule, state.index))
	default:
		return semanticSubjectRule
	}
}

func resolveInspectionRuleLocator(state *ruleLocatorState, inspection cli.Inspection) locatorResolution[cli.RuleInspection] {
	var resolution locatorResolution[cli.RuleInspection]
	label := ruleLocatorLabel(state)
	if state == nil || state.command == nil || state.mode == ruleLocatorInvalid {
		resolution = locatorResolution[cli.RuleInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectRule, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveInspectionCommandLocator(state.command, inspection)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.RuleInspection]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	switch state.mode {
	case ruleLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.RuleInspection]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectRule, observationInvalidLocator),
			}
			return resolution
		}
		if state.index < len(command.value.Rules) {
			resolution = locatorResolution[cli.RuleInspection]{
				status:  locatorResolutionFound,
				value:   command.value.Rules[state.index],
				subject: label,
			}
			return resolution
		}
	case ruleLocatorCode:
		matches := make([]int, 0, 1)
		for index, rule := range command.value.Rules {
			if rule.Code == state.code {
				matches = append(matches, index)
			}
		}
		if len(matches) == 1 {
			resolution = locatorResolution[cli.RuleInspection]{
				status:  locatorResolutionFound,
				value:   command.value.Rules[matches[0]],
				subject: label,
			}
			return resolution
		}
		if len(matches) > 1 {
			slices.Sort(matches)
			resolution = locatorResolution[cli.RuleInspection]{
				status:  locatorResolutionAmbiguous,
				subject: label,
				detail: collectionCardinalityField{
					name:  "matches rule-code indexes",
					count: len(matches),
				},
			}
			return resolution
		}
	default:
		resolution = locatorResolution[cli.RuleInspection]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectRule, observationInvalidLocator),
		}
		return resolution
	}
	if detail, incomplete := policyInspectionNamespaceIncomplete(command.value, inspection); incomplete {
		resolution = locatorResolution[cli.RuleInspection]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  detail,
		}
		return resolution
	}
	resolution = locatorResolution[cli.RuleInspection]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  observed(semanticSubjectRule, observationAbsent),
	}
	return resolution
}

func resolveSchemaRuleLocator(state *ruleLocatorState, schema *cli.Schema) locatorResolution[cli.RuleSchema] {
	var resolution locatorResolution[cli.RuleSchema]
	label := ruleLocatorLabel(state)
	if schema == nil || state == nil || state.command == nil || state.mode == ruleLocatorInvalid {
		resolution = locatorResolution[cli.RuleSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectRule, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.RuleSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	return selectSchemaRuleLocator(state, command.value.Rules, label)
}

func resolveReferenceRuleLocator(state *ruleLocatorState, reference cli.Reference) locatorResolution[cli.RuleSchema] {
	var resolution locatorResolution[cli.RuleSchema]
	label := ruleLocatorLabel(state)
	if state == nil || state.command == nil || state.mode == ruleLocatorInvalid {
		resolution = locatorResolution[cli.RuleSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectRule, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = locatorResolution[cli.RuleSchema]{
			status:  command.status,
			subject: label,
			detail:  command.detail,
		}
		return resolution
	}
	return selectSchemaRuleLocator(state, command.value.Rules, label)
}

func selectSchemaRuleLocator(state *ruleLocatorState, values []cli.RuleSchema, label evaluationField) locatorResolution[cli.RuleSchema] {
	var resolution locatorResolution[cli.RuleSchema]
	switch state.mode {
	case ruleLocatorIndex:
		if state.index < 0 {
			resolution = locatorResolution[cli.RuleSchema]{
				status:  locatorResolutionInvalid,
				subject: label,
				detail:  observed(semanticSubjectRule, observationInvalidLocator),
			}
			return resolution
		}
		if state.index < len(values) {
			resolution = locatorResolution[cli.RuleSchema]{
				status:  locatorResolutionFound,
				value:   values[state.index],
				subject: label,
			}
			return resolution
		}
		resolution = locatorResolution[cli.RuleSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectRule, observationAbsent),
		}
		return resolution
	case ruleLocatorCode:
		matches := make([]int, 0, 1)
		for index, value := range values {
			if value.Code == state.code {
				matches = append(matches, index)
			}
		}
		if len(matches) == 1 {
			resolution = locatorResolution[cli.RuleSchema]{
				status:  locatorResolutionFound,
				value:   values[matches[0]],
				subject: label,
			}
			return resolution
		}
		if len(matches) > 1 {
			slices.Sort(matches)
			resolution = locatorResolution[cli.RuleSchema]{
				status:  locatorResolutionAmbiguous,
				subject: label,
				detail: collectionCardinalityField{
					name:  "matches rule-code indexes",
					count: len(matches),
				},
			}
			return resolution
		}
		resolution = locatorResolution[cli.RuleSchema]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(semanticSubjectRule, observationAbsent),
		}
		return resolution
	default:
		resolution = locatorResolution[cli.RuleSchema]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectRule, observationInvalidLocator),
		}
		return resolution
	}
}

func ruleLocatorStates(values []RuleLocator) []*ruleLocatorState {
	states := make([]*ruleLocatorState, len(values))
	for index, value := range values {
		states[index] = value.state
	}
	return states
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func compareRuleStatesSchema(command cli.CommandSchema, states []*ruleLocatorState, schema *cli.Schema, label evaluationField) evaluationResult {
	if len(command.Rules) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("rules count %d", len(states))), evaluationText(fmt.Sprintf("rules count %d", len(command.Rules))))
	}
	for index, state := range states {
		if state == nil || !validateSchemaPolicyParent(state.command, command, schema) {
			return disprovedResult(label, evaluationText("ordered rule locators"), evaluationText(fmt.Sprintf("invalid rule locator at position %d", index)))
		}
		switch state.mode {
		case ruleLocatorIndex:
			if state.index != index {
				return disprovedResult(label, evaluationText("ordered rule locators"), evaluationText(fmt.Sprintf("rule index %d at position %d", state.index, index)))
			}
		case ruleLocatorCode:
			matches := 0
			matched := -1
			for candidate := range command.Rules {
				if command.Rules[candidate].Code == state.code {
					matches++
					matched = candidate
				}
			}
			if matches != 1 || matched != index {
				field := collectionCardinalityField{
					name:  semanticNameMatchesRuleCodeRecords,
					count: matches,
				}
				return disprovedResult(label, evaluationText("ordered unambiguous rule locators"), field)
			}
		default:
			return disprovedResult(label, evaluationText("rule index/code locators"), evaluationText(fmt.Sprintf("invalid rule locator at position %d", index)))
		}
	}
	return provedResult()
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func compareRuleStatesReference(command cli.CommandReference, states []*ruleLocatorState, reference cli.Reference, label evaluationField) evaluationResult {
	if len(command.Rules) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("rules count %d", len(states))), evaluationText(fmt.Sprintf("rules count %d", len(command.Rules))))
	}
	for index, state := range states {
		if state == nil || !validateReferencePolicyParent(state.command, command, reference) {
			return disprovedResult(label, evaluationText("ordered rule locators"), evaluationText(fmt.Sprintf("invalid rule locator at position %d", index)))
		}
		switch state.mode {
		case ruleLocatorIndex:
			if state.index != index {
				return disprovedResult(label, evaluationText("ordered rule locators"), evaluationText(fmt.Sprintf("rule index %d at position %d", state.index, index)))
			}
		case ruleLocatorCode:
			matches := 0
			matched := -1
			for candidate := range command.Rules {
				if command.Rules[candidate].Code == state.code {
					matches++
					matched = candidate
				}
			}
			if matches != 1 || matched != index {
				field := collectionCardinalityField{
					name:  semanticNameMatchesRuleCodeRecords,
					count: matches,
				}
				return disprovedResult(label, evaluationText("ordered unambiguous rule locators"), field)
			}
		default:
			return disprovedResult(label, evaluationText("rule index/code locators"), evaluationText(fmt.Sprintf("invalid rule locator at position %d", index)))
		}
	}
	return provedResult()
}

func compareRuleStatesInspection(command cli.CommandInspection, states []*ruleLocatorState, inspection cli.Inspection, label evaluationField) evaluationResult {
	if len(command.Rules) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("rules count %d", len(states))), evaluationText(fmt.Sprintf("rules count %d", len(command.Rules))))
	}
	for index, state := range states {
		if state == nil {
			return disprovedResult(label, evaluationText("ordered rule locators"), evaluationText(fmt.Sprintf("invalid rule locator at position %d", index)))
		}
		if outcome, ok := validateInspectionPolicyParent(state.command, command, inspection, label, evaluationText("ordered rule locators")); !ok {
			return outcome
		}
		switch state.mode {
		case ruleLocatorIndex:
			if state.index != index {
				return disprovedResult(label, evaluationText("ordered rule locators"), evaluationText(fmt.Sprintf("rule index %d at position %d", state.index, index)))
			}
		case ruleLocatorCode:
			matches := 0
			matched := -1
			for candidate := range command.Rules {
				if command.Rules[candidate].Code == state.code {
					matches++
					matched = candidate
				}
			}
			if matches != 1 || matched != index {
				field := collectionCardinalityField{
					name:  semanticNameMatchesRuleCodeRecords,
					count: matches,
				}
				return disprovedResult(label, evaluationText("ordered unambiguous rule locators"), field)
			}
		default:
			return disprovedResult(label, evaluationText("rule index/code locators"), evaluationText(fmt.Sprintf("invalid rule locator at position %d", index)))
		}
	}
	return provedResult()
}

func ruleCodeLocator(command CommandLocator, code cli.RuleCode) RuleLocator {
	state := &ruleLocatorState{
		mode:    ruleLocatorCode,
		command: command.state,
		code:    code,
	}

	locator := RuleLocator{
		state: state,
	}
	return locator
}

func ruleIndexLocator(command CommandLocator, index int) RuleLocator {
	state := &ruleLocatorState{
		mode:    ruleLocatorIndex,
		command: command.state,
		index:   index,
	}

	locator := RuleLocator{
		state: state,
	}
	return locator
}

func ruleOutcomeExpectation(locator RuleLocator, outcome cli.EvaluationOutcome) InspectionExpectation {
	value := ruleOutcomeExpectationValue{
		locator:  locator.state,
		expected: outcome,
	}
	return value
}

func ruleActivationOutcomeExpectation(locator RuleLocator, outcome cli.EvaluationOutcome) InspectionExpectation {
	value := ruleActivationOutcomeExpectationValue{
		locator:  locator.state,
		expected: outcome,
	}
	return value
}

func (ruleOperandSensitiveExpectation) schemaExpectation() {}

func (ruleOperandSensitiveExpectation) referenceExpectation() {}

func (ruleOperandSensitiveExpectation) inspectionExpectation() {}

func (expectation ruleOperandSensitiveExpectation) evaluateSchemaResolution(rule locatorResolution[cli.RuleSchema]) evaluationResult {
	if rule.status != locatorResolutionFound {
		return expectedRelationFailure(rule.status, semanticSubjectRuleOperand, semanticRequirementSensitive, rule.detail)
	}
	if expectation.index < 0 {
		return disprovedResult(semanticSubjectRuleOperand, semanticRequirementSensitive, observed(semanticSubjectRuleOperand, observationInvalidLocator))
	}
	if expectation.index >= len(rule.value.Operands) {
		return disprovedResult(semanticSubjectRuleOperand, semanticRequirementSensitive, observed(semanticSubjectRuleOperand, observationAbsent))
	}
	return equalFieldResult(semanticSubjectRuleOperand, semanticRequirementSensitive, evaluationBool(true), evaluationBool(rule.value.Operands[expectation.index].Sensitive), rule.value.Operands[expectation.index].Sensitive)
}

func (expectation ruleOperandSensitiveExpectation) evaluateInspectionResolution(rule locatorResolution[cli.RuleInspection]) evaluationResult {
	if rule.status != locatorResolutionFound {
		return expectedRelationFailure(rule.status, semanticSubjectRuleOperand, semanticRequirementSensitive, rule.detail)
	}
	if expectation.index < 0 {
		return disprovedResult(semanticSubjectRuleOperand, semanticRequirementSensitive, observed(semanticSubjectRuleOperand, observationInvalidLocator))
	}
	if expectation.index >= len(rule.value.Operands) {
		return disprovedResult(semanticSubjectRuleOperand, semanticRequirementSensitive, observed(semanticSubjectRuleOperand, observationAbsent))
	}
	return equalFieldResult(semanticSubjectRuleOperand, semanticRequirementSensitive, evaluationBool(true), evaluationBool(rule.value.Operands[expectation.index].Sensitive), rule.value.Operands[expectation.index].Sensitive)
}

func (expectation ruleOperandSensitiveExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return expectation.evaluateSchemaResolution(resolveSchemaRuleLocator(expectation.rule, schema))
}

func (expectation ruleOperandSensitiveExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return expectation.evaluateSchemaResolution(resolveReferenceRuleLocator(expectation.rule, reference))
}

func (expectation ruleOperandSensitiveExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return expectation.evaluateInspectionResolution(resolveInspectionRuleLocator(expectation.rule, inspection))
}

func (ruleOperandNoLiteralExpectation) schemaExpectation() {}

func (ruleOperandNoLiteralExpectation) referenceExpectation() {}

func (ruleOperandNoLiteralExpectation) inspectionExpectation() {}

func (expectation ruleOperandNoLiteralExpectation) evaluate(resolution locatorResolution[cli.ValueLiteral], truncated bool) evaluationResult {
	subject := evaluationText("rule operand")
	expected := evaluationText("literal absent")
	switch resolution.status {
	case locatorResolutionAbsent:
		if truncated {
			return notProvableResult(subject, expected, observed(semanticSubjectInspection, observationTruncated))
		}
		return provedResult()
	case locatorResolutionFound:
		return disprovedResult(subject, expected, evaluationText("literal present"))
	case locatorResolutionNotProvable:
		return notProvableResult(subject, expected, resolution.detail)
	default:
		return disprovedResult(subject, expected, observed(semanticSubjectLocator, observationInvalidLocator))
	}
}

func (expectation ruleOperandNoLiteralExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return expectation.evaluate(resolveSchemaDeclarationValueLiteralLocator(expectation.locator, schema), false)
}

func (expectation ruleOperandNoLiteralExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return expectation.evaluate(resolveReferenceDeclarationValueLiteralLocator(expectation.locator, reference), false)
}

func (expectation ruleOperandNoLiteralExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return expectation.evaluate(resolveInspectionDeclarationValueLiteralLocator(expectation.locator, inspection), inspection.Truncated)
}

func (commandRulesExpectation) schemaExpectation() {}

func (commandRulesExpectation) referenceExpectation() {}

func (commandRulesExpectation) inspectionExpectation() {}

func (expectation commandRulesExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.command, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("rules exactly"), command.detail)
	}
	return compareRuleStatesSchema(command.value, expectation.items, schema, qualifySubject(command.subject, evaluationText("rules")))
}

func (expectation commandRulesExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.command, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("rules exactly"), command.detail)
	}
	return compareRuleStatesReference(command.value, expectation.items, reference, qualifySubject(command.subject, evaluationText("rules")))
}

func (expectation commandRulesExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.command, inspection)
	label, failure, ok := exactPolicyCollectionInspection(command, inspection, "rules")
	if !ok {
		return failure
	}
	return compareRuleStatesInspection(command.value, expectation.items, inspection, label)
}

func (ruleExistenceExpectationValue) schemaExpectation() {}

func (ruleExistenceExpectationValue) referenceExpectation() {}

func (ruleExistenceExpectationValue) inspectionExpectation() {}

func (expectation ruleExistenceExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	return declarationExistenceResult(resolveSchemaRuleLocator(expectation.locator, schema), expectation.expected)
}

func (expectation ruleExistenceExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return declarationExistenceResult(resolveReferenceRuleLocator(expectation.locator, reference), expectation.expected)
}

func (expectation ruleExistenceExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return declarationExistenceResult(resolveInspectionRuleLocator(expectation.locator, inspection), expectation.expected)
}

func (ruleKindExpectationValue) schemaExpectation() {}

func (ruleKindExpectationValue) referenceExpectation() {}

func (ruleKindExpectationValue) inspectionExpectation() {}

func (expectation ruleKindExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation ruleKindExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (expectation ruleKindExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	actual := resolution.value.Kind
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("kind %v", expectation.expected)), evaluationText(fmt.Sprintf("kind %v", actual)), actual == expectation.expected)
}

func (ruleCodeExpectationValue) schemaExpectation() {}

func (ruleCodeExpectationValue) referenceExpectation() {}

func (ruleCodeExpectationValue) inspectionExpectation() {}

func (expectation ruleCodeExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	actual := resolution.value.Code
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, subjectNamed(evaluationText("code"), string(expectation.expected)), subjectNamed(evaluationText("code"), string(actual)), actual == expectation.expected)
}

func (expectation ruleCodeExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	actual := resolution.value.Code
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, subjectNamed(evaluationText("code"), string(expectation.expected)), subjectNamed(evaluationText("code"), string(actual)), actual == expectation.expected)
}

func (expectation ruleCodeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	actual := resolution.value.Code
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, subjectNamed(evaluationText("code"), string(expectation.expected)), subjectNamed(evaluationText("code"), string(actual)), actual == expectation.expected)
}

func (ruleReachabilityExpectationValue) schemaExpectation() {}

func (ruleReachabilityExpectationValue) referenceExpectation() {}

func (ruleReachabilityExpectationValue) inspectionExpectation() {}

func (expectation ruleReachabilityExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation ruleReachabilityExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (expectation ruleReachabilityExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	actual := resolution.value.Reachability
	return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, evaluationText(fmt.Sprintf("reachability %v", expectation.expected)), evaluationText(fmt.Sprintf("reachability %v", actual)), actual == expectation.expected)
}

func (ruleSensitiveExpectationValue) schemaExpectation() {}

func (ruleSensitiveExpectationValue) referenceExpectation() {}

func (ruleSensitiveExpectationValue) inspectionExpectation() {}

func (expectation ruleSensitiveExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("sensitive"), true, resolution.value.Sensitive)
}

func (expectation ruleSensitiveExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("sensitive"), true, resolution.value.Sensitive)
}

func (expectation ruleSensitiveExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	return resolvedBoolProperty(resolution.status, resolution.subject, resolution.detail, evaluationText("sensitive"), true, resolution.value.Sensitive)
}

func (ruleValuesExpectationValue) schemaExpectation() {}

func (ruleValuesExpectationValue) referenceExpectation() {}

func (ruleValuesExpectationValue) inspectionExpectation() {}

func (expectation ruleValuesExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, expectation.valuesField(), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsSchema(expectation.expected, schema)
	return expectedValueIDsResult(resolution.subject, expectation.ruleSchemaValues(resolution.value), expected, status, detail, expectation.valuesName())
}

func (expectation ruleValuesExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, expectation.valuesField(), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsReference(expectation.expected, reference)
	return expectedValueIDsResult(resolution.subject, expectation.ruleSchemaValues(resolution.value), expected, status, detail, expectation.valuesName())
}

func (expectation ruleValuesExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return declarationResolvedComparison(resolution.status, resolution.subject, resolution.detail, expectation.valuesField(), evaluationText(""), false)
	}
	expected, status, detail := resolveExpectedValueIDsInspection(expectation.expected, inspection)
	return expectedValueIDsResult(resolution.subject, expectation.actualInspection(resolution.value), expected, status, detail, expectation.valuesName())
}

func (expectation ruleValuesExpectationValue) valuesName() string {
	if expectation.dependencies {
		return semanticNameDependencies
	}
	return "subjects"
}

func (expectation ruleValuesExpectationValue) valuesField() evaluationField {
	return evaluationText("exact " + expectation.valuesName())
}

func (expectation ruleValuesExpectationValue) ruleSchemaValues(rule cli.RuleSchema) []cli.ValueID {
	if expectation.dependencies {
		return rule.Dependencies
	}
	return rule.Subjects
}

func (expectation ruleValuesExpectationValue) actualInspection(rule cli.RuleInspection) []cli.ValueID {
	if expectation.dependencies {
		return rule.Dependencies
	}
	return rule.Subjects
}

func (rulePatternExpectationValue) schemaExpectation() {}

func (rulePatternExpectationValue) referenceExpectation() {}

func (rulePatternExpectationValue) inspectionExpectation() {}

func (expectation rulePatternExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("pattern"), evaluationText(expectation.expected), evaluationText(resolution.value.Pattern), resolution.value.Pattern == expectation.expected)
}

func (expectation rulePatternExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("pattern"), evaluationText(expectation.expected), evaluationText(resolution.value.Pattern), resolution.value.Pattern == expectation.expected)
}

func (expectation rulePatternExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("pattern"), evaluationText(expectation.expected), evaluationText(resolution.value.Pattern), resolution.value.Pattern == expectation.expected)
}

func (ruleCountExpectationValue) schemaExpectation() {}

func (ruleCountExpectationValue) referenceExpectation() {}

func (ruleCountExpectationValue) inspectionExpectation() {}

func (expectation ruleCountExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	field := countRangeField{
		value: expectation.expected,
	}
	observedField := countRangeField{
		value: resolution.value.Count,
	}
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("count"), field, observedField, resolution.value.Count == expectation.expected)
}

func (expectation ruleCountExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	field := countRangeField{
		value: expectation.expected,
	}
	observedField := countRangeField{
		value: resolution.value.Count,
	}
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("count"), field, observedField, resolution.value.Count == expectation.expected)
}

func (expectation ruleCountExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	field := countRangeField{
		value: expectation.expected,
	}
	observedField := countRangeField{
		value: resolution.value.Count,
	}
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("count"), field, observedField, resolution.value.Count == expectation.expected)
}

func (ruleOperandCountExpectationValue) schemaExpectation() {}

func (ruleOperandCountExpectationValue) referenceExpectation() {}

func (ruleOperandCountExpectationValue) inspectionExpectation() {}

func (expectation ruleOperandCountExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolution := resolveSchemaRuleLocator(expectation.locator, schema)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("operand-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(resolution.value.Operands))), len(resolution.value.Operands) == expectation.expected)
}

func (expectation ruleOperandCountExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolution := resolveReferenceRuleLocator(expectation.locator, reference)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("operand-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(resolution.value.Operands))), len(resolution.value.Operands) == expectation.expected)
}

func (expectation ruleOperandCountExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("operand-count"), evaluationText(strconv.Itoa(expectation.expected)), evaluationText(strconv.Itoa(len(resolution.value.Operands))), len(resolution.value.Operands) == expectation.expected)
}

func (ruleOutcomeExpectationValue) inspectionExpectation() {}

//nolint:dupl // Rule outcome variants remain direct typed evaluators.
func (expectation ruleOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Outcome)), resolution.value.Outcome == expectation.expected)
	}
	if resolution.value.Outcome == expectation.expected {
		return provedResult()
	}
	if resolution.value.Sensitive {
		return disprovedResult(
			qualifySubject(resolution.subject, evaluationText("outcome")),
			evaluationText("redacted expected rule property"),
			evaluationText("redacted observed rule property"),
		)
	}
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.Outcome)), resolution.value.Outcome == expectation.expected)
}

func (ruleActivationOutcomeExpectationValue) inspectionExpectation() {}

//nolint:dupl // Rule outcome variants remain direct typed evaluators.
func (expectation ruleActivationOutcomeExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolution := resolveInspectionRuleLocator(expectation.locator, inspection)
	if resolution.status != locatorResolutionFound {
		return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("activation-outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.ActivationOutcome)), resolution.value.ActivationOutcome == expectation.expected)
	}
	if resolution.value.ActivationOutcome == expectation.expected {
		return provedResult()
	}
	if resolution.value.Sensitive {
		return disprovedResult(
			qualifySubject(resolution.subject, evaluationText("activation-outcome")),
			evaluationText("redacted expected rule property"),
			evaluationText("redacted observed rule property"),
		)
	}
	return policyPropertyResult(resolution.status, resolution.subject, resolution.detail, evaluationText("activation-outcome"), evaluationText(fmt.Sprint(expectation.expected)), evaluationText(fmt.Sprint(resolution.value.ActivationOutcome)), resolution.value.ActivationOutcome == expectation.expected)
}
