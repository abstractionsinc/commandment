package cli

import (
	"slices"
	"strconv"
)

func effectiveDestinationSensitive(model *effectiveModel, destination ValueID) (bool, error) {
	record, ok := model.Destination(destination)
	if !ok {
		return false, errInvalidModelTopology
	}
	return record.sensitive, nil
}

func projectEffectiveRuleOperand(model *effectiveModel, destination ValueID, operand any) (RuleOperand, error) {
	record, ok := model.Destination(destination)
	if !ok {
		return RuleOperand{}, errInvalidModelTopology
	}
	return projectRuleOperandRecord(record, operand)
}

func projectEffectiveRuleFailure(model *effectiveModel, id RuleID, definition ruleDefinition) (RuleFailure, error) {
	kind := definition.Kind()
	if !kind.valid() || !id.present() {
		return RuleFailure{}, errInvalidModelTopology
	}
	result := RuleFailure{
		RuleID:   RuleID(id.uint32()),
		RuleKind: kind,
		RuleCode: RuleCode(definition.Code()),
		Subjects: destinationValueIDs(definition.Subjects()),
	}
	if definition.Kind() == RuleMatch {
		result.Pattern = definition.Pattern()
	}
	if definition.Kind() == RuleItemCount || definition.Kind() == RuleEntryCount {
		result.Count = countRange(definition.Count())
	}
	subjects := definition.Subjects()
	if len(definition.Operands()) != 0 {
		if len(subjects) == 0 {
			return RuleFailure{}, errInvalidModelTopology
		}
		for _, operand := range definition.Operands() {
			projected, err := projectEffectiveRuleOperand(model, subjects[0], operand)
			if err != nil {
				return RuleFailure{}, err
			}
			result.Operands = append(result.Operands, projected)
		}
	}
	return result, nil
}

const (
	noticeSeverityInformationCode uint8 = iota + 1
	noticeSeverityWarningCode
)

func projectConditionEffect(effect ConditionEffect) (ConditionEffect, bool) {
	switch effect {
	case ConditionEffectAvailability:
		return ConditionEffectAvailability, true
	case ConditionEffectVisibility:
		return ConditionEffectVisibility, true
	default:
		return ConditionEffectUnknown, false
	}
}

func projectConditionExpressionWith(value condition, sensitive func(ValueID) (bool, error), projectLiteral func(ValueID, any, bool) (ValueLiteral, error), projectChild func(condition) (ConditionExpression, error)) (ConditionExpression, error) {
	if !value.Valid() {
		return ConditionExpression{}, errInvalidModelTopology
	}
	result := ConditionExpression{
		Destination: value.Destination(),
	}
	switch value.Kind() {
	case conditionKindResolved:
		result.Kind = ConditionExpressionResolved
	case conditionKindEqual:
		result.Kind = ConditionExpressionEqual
	case conditionKindNotEqual:
		result.Kind = ConditionExpressionNotEqual
	case conditionKindIn:
		result.Kind = ConditionExpressionIn
	case conditionKindNotIn:
		result.Kind = ConditionExpressionNotIn
	case conditionKindGreaterThan:
		result.Kind = ConditionExpressionGreaterThan
	case conditionKindAtLeast:
		result.Kind = ConditionExpressionAtLeast
	case conditionKindLessThan:
		result.Kind = ConditionExpressionLessThan
	case conditionKindAtMost:
		result.Kind = ConditionExpressionAtMost
	case conditionKindMatches:
		result.Kind = ConditionExpressionMatch
		result.Pattern = value.Pattern()
	case conditionKindAll:
		result.Kind = ConditionExpressionAll
		result.Destination = 0
	case conditionKindAny:
		result.Kind = ConditionExpressionAny
		result.Destination = 0
	case conditionKindNot:
		result.Kind = ConditionExpressionNot
		result.Destination = 0
	default:
		return ConditionExpression{}, errInvalidModelTopology
	}
	if value.Destination().present() {
		sensitiveValue, err := sensitive(value.Destination())
		if err != nil {
			return ConditionExpression{}, err
		}
		result.Sensitive = sensitiveValue
		for _, operand := range value.Operands() {
			literal, err := projectLiteral(value.Destination(), operand, sensitiveValue)
			if err != nil {
				return ConditionExpression{}, err
			}
			result.Operands = append(result.Operands, literal)
		}
	}
	for _, child := range value.Children() {
		projected, err := projectChild(child)
		if err != nil {
			return ConditionExpression{}, err
		}
		result.Children = append(result.Children, projected)
		result.Sensitive = result.Sensitive || projected.Sensitive
	}
	return result, nil
}

func projectPolicyConditionExpressionWith(value policyCondition, sensitive func(ValueID) (bool, error), projectLiteral func(ValueID, any, bool) (ValueLiteral, error), projectChild func(policyCondition) (ConditionExpression, error)) (ConditionExpression, error) {
	if !value.Valid() {
		return ConditionExpression{}, errInvalidModelTopology
	}
	result := ConditionExpression{
		Destination: value.Destination(),
	}
	switch value.Kind() {
	case policyConditionResolved:
		result.Kind = ConditionExpressionResolved
	case policyConditionEqual:
		result.Kind = ConditionExpressionEqual
	case policyConditionNotEqual:
		result.Kind = ConditionExpressionNotEqual
	case policyConditionIn:
		result.Kind = ConditionExpressionIn
	case policyConditionNotIn:
		result.Kind = ConditionExpressionNotIn
	case policyConditionGreaterThan:
		result.Kind = ConditionExpressionGreaterThan
	case policyConditionAtLeast:
		result.Kind = ConditionExpressionAtLeast
	case policyConditionLessThan:
		result.Kind = ConditionExpressionLessThan
	case policyConditionAtMost:
		result.Kind = ConditionExpressionAtMost
	case policyConditionMatches:
		result.Kind = ConditionExpressionMatch
		result.Pattern = value.Pattern()
	case policyConditionContributedBy:
		result.Kind = ConditionExpressionContributedBy
		for _, producerKind := range value.ProducerKinds() {
			if !producerKind.valid() {
				return ConditionExpression{}, errInvalidModelTopology
			}
			result.ProducerKinds = append(result.ProducerKinds, producerKind)
		}
	case policyConditionAll:
		result.Kind = ConditionExpressionAll
		result.Destination = 0
	case policyConditionAny:
		result.Kind = ConditionExpressionAny
		result.Destination = 0
	case policyConditionNot:
		result.Kind = ConditionExpressionNot
		result.Destination = 0
	default:
		return ConditionExpression{}, errInvalidModelTopology
	}
	if value.Destination().present() {
		sensitiveValue, err := sensitive(value.Destination())
		if err != nil {
			return ConditionExpression{}, err
		}
		result.Sensitive = sensitiveValue
		for _, operand := range value.Operands() {
			literal, err := projectLiteral(value.Destination(), operand, sensitiveValue)
			if err != nil {
				return ConditionExpression{}, err
			}
			result.Operands = append(result.Operands, literal)
		}
	}
	for _, child := range value.Children() {
		projected, err := projectChild(child)
		if err != nil {
			return ConditionExpression{}, err
		}
		result.Children = append(result.Children, projected)
		result.Sensitive = result.Sensitive || projected.Sensitive
	}
	return result, nil
}

func projectRuleOperandRecord(record destinationRecord, operand any) (RuleOperand, error) {
	definition, ok := record.Definition()
	if !ok {
		return RuleOperand{}, errInvalidModelTopology
	}
	sensitive := record.sensitive
	result := RuleOperand{
		Sensitive: sensitive,
	}

	if valueType, ok := definition.ValueType(); ok && valueType.Kind == ValueKindCount {
		literal := ValueLiteral{
			Type: valueType,
		}
		if sensitive {
			literal.Sensitive = true
			literal.Redacted = true
			result.Literal = &literal
			return result, nil
		}
		value, ok := operand.(int)
		if !ok {
			return RuleOperand{}, errInvalidModelTopology
		}
		literal.Text = strconv.Itoa(value)
		result.Literal = &literal
		return result, nil
	}

	language, valueType, representable, err := ruleOperandLanguage(definition)
	if err != nil {
		return RuleOperand{}, err
	}
	if !representable {
		return result, nil
	}
	literal := ValueLiteral{
		Type: valueType,
	}
	if sensitive {
		literal.Sensitive = true
		literal.Redacted = true
		result.Literal = &literal
		return result, nil
	}
	text, err := language.Format(operand)
	if err != nil {
		return RuleOperand{}, errInvalidModelTopology
	}
	literal.Text = text
	result.Literal = &literal
	return result, nil
}

func ruleOperandLanguage(definition destinationDefinition) (valueLanguage, ValueType, bool, error) {
	switch definition.Kind() {
	case destinationKindScalar:
		language, languageOK := definition.Language()
		valueType, typeOK := definition.ValueType()
		if !languageOK || !typeOK {
			return valueLanguage{}, ValueType{}, false, errInvalidModelTopology
		}
		if language.DynamicDomain() {
			return valueLanguage{}, ValueType{}, false, nil
		}
		return language, valueType, true, nil
	case destinationKindSequence:
		language, languageOK := definition.Language()
		valueType, typeOK := definition.ElementType()
		if !languageOK || !typeOK {
			return valueLanguage{}, ValueType{}, false, errInvalidModelTopology
		}
		if language.DynamicDomain() {
			return valueLanguage{}, ValueType{}, false, nil
		}
		return language, valueType, true, nil
	case destinationKindKeyed:
		language, languageOK := definition.KeyedLanguage()
		valueType, typeOK := definition.KeyType()
		if !languageOK || !typeOK {
			return valueLanguage{}, ValueType{}, false, errInvalidModelTopology
		}
		return language.Key(), valueType, true, nil
	case destinationKindOpaque,
		destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		return valueLanguage{}, ValueType{}, false, nil
	default:
		return valueLanguage{}, ValueType{}, false, errInvalidModelTopology
	}
}

func projectNoticeSeverity(severity uint8) (NoticeSeverity, bool) {
	switch severity {
	case noticeSeverityInformationCode:
		return NoticeSeverityInformation, true
	case noticeSeverityWarningCode:
		return NoticeSeverityWarning, true
	default:
		return NoticeSeverityUnknown, false
	}
}

func projectNoticeMatch(match noticeMatch) (NoticeMatch, bool) {
	if !match.Valid() {
		return NoticeMatch{}, false
	}
	severity := NoticeSeverityUnknown
	if match.Severity != 0 {
		var ok bool
		severity, ok = projectNoticeSeverity(match.Severity)
		if !ok {
			return NoticeMatch{}, false
		}
	}
	result := NoticeMatch{
		All:          match.All,
		Severity:     severity,
		Code:         NoticeCode(match.Code),
		AdvisoryCode: AdvisoryCode(match.AdvisoryCode),
	}
	return result, true
}

func destinationValueIDs(values []ValueID) []ValueID {
	return slices.Clone(values)
}
