package cli

import (
	"context"
	"regexp"
	"slices"
	"strings"
)

func prepareCondition(authored Condition, resolver destinationAcceptanceResolver) (condition, bool) {
	if authored.state == nil || !authored.state.valid {
		return condition{}, false
	}
	state := authored.state
	kind, valid := prepareConditionKind(state.kind)
	if !valid {
		return condition{}, false
	}
	var destination ValueID
	if state.destination != nil {
		id, ok := resolver.ValueID(state.destination)
		if !ok {
			return condition{}, false
		}
		destination = id
	}
	children := make([]condition, len(state.children))
	for index, child := range state.children {
		acceptedCondition, ok := prepareCondition(Condition{
			state: child,
		}, resolver)
		if !ok {
			return condition{}, false
		}
		children[index] = acceptedCondition
	}
	if state.kind == conditionMatches {
		if _, err := regexp.Compile(state.pattern); err != nil {
			return condition{}, false
		}
	}
	switch kind {
	case conditionKindResolved:
		input := resolvedConditionInput{
			Destination: destination,
			Predicate:   state.predicate,
		}
		return acceptResolvedCondition(input), true
	case conditionKindEqual,
		conditionKindNotEqual,
		conditionKindGreaterThan,
		conditionKindAtLeast,
		conditionKindLessThan,
		conditionKindAtMost:
		input := comparisonConditionInput{
			Kind:        kind,
			Destination: destination,
			Operands:    state.operands,
			Predicate:   state.predicate,
		}
		return acceptComparisonCondition(input), true
	case conditionKindIn, conditionKindNotIn:
		input := membershipConditionInput{
			Kind:        kind,
			Destination: destination,
			Operands:    state.operands,
			Predicate:   state.predicate,
		}
		return acceptMembershipCondition(input), true
	case conditionKindMatches:
		input := patternConditionInput{
			Destination: destination,
			Pattern:     state.pattern,
			Predicate:   state.predicate,
		}
		return acceptPatternCondition(input), true
	case conditionKindAll, conditionKindAny:
		input := conditionGroupInput{
			Kind:      kind,
			Children:  children,
			Predicate: state.predicate,
		}
		return acceptConditionGroup(input), true
	case conditionKindNot:
		input := negatedConditionInput{
			Children:  children,
			Predicate: state.predicate,
		}
		return acceptNegatedCondition(input), true
	default:
		return condition{}, false
	}
}

func prepareConditionKind(kind conditionKind) (semanticConditionKind, bool) {
	switch kind {
	case conditionResolved:
		return conditionKindResolved, true
	case conditionEqual:
		return conditionKindEqual, true
	case conditionNotEqual:
		return conditionKindNotEqual, true
	case conditionIn:
		return conditionKindIn, true
	case conditionNotIn:
		return conditionKindNotIn, true
	case conditionGreaterThan:
		return conditionKindGreaterThan, true
	case conditionAtLeast:
		return conditionKindAtLeast, true
	case conditionLessThan:
		return conditionKindLessThan, true
	case conditionAtMost:
		return conditionKindAtMost, true
	case conditionMatches:
		return conditionKindMatches, true
	case conditionAll:
		return conditionKindAll, true
	case conditionAny:
		return conditionKindAny, true
	case conditionNot:
		return conditionKindNot, true
	default:
		return conditionKindUnknown, false
	}
}

func appendUniqueDestination(values []ValueID, value ValueID) []ValueID {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func appendUniqueDependency(values []dependencyRequirement, value dependencyRequirement) []dependencyRequirement {
	for _, existing := range values {
		if existing.Destination() == value.Destination() {
			if existing.Optional() == value.Optional() {
				return values
			}
			// Same dependency cannot be simultaneously required and optional in one
			// exact declaration; retain the stricter required edge.
			if !value.Optional() {
				for i := range values {
					if values[i].Destination() == value.Destination() {
						values[i] = value
						return values
					}
				}
			}
			return values
		}
	}
	return append(values, value)
}

func prepareRuleCondition(condition RuleCondition, resolver destinationAcceptanceResolver) (policyCondition, bool) {
	state := ruleConditionStateOf(condition)
	if state == nil || !state.valid {
		return policyCondition{}, false
	}
	switch state.kind {
	case ruleConditionStructural:
		structural, valid := prepareCondition(Condition{
			state: state.structural,
		}, resolver)
		if !valid {
			return policyCondition{}, false
		}
		return policyConditionFromStructural(structural), true
	case ruleConditionContributedBy:
		id, ok := resolver.ValueID(state.destination)
		if !ok {
			return policyCondition{}, false
		}
		producerKinds := make([]ValueProducerKind, len(state.producerKinds))
		for i, producerKind := range state.producerKinds {
			mapped, valid := semanticValueProducerKind(producerKind)
			if !valid {
				return policyCondition{}, false
			}
			producerKinds[i] = mapped
		}
		input := contributedByPolicyConditionInput{
			Destination:   id,
			ProducerKinds: producerKinds,
		}
		return acceptContributedByPolicyCondition(input), true
	case ruleConditionAll, ruleConditionAny, ruleConditionNot:
		children := make([]policyCondition, len(state.children))
		for i, child := range state.children {
			acceptedCondition, valid := prepareRuleCondition(validationRuleCondition{
				state: child,
			}, resolver)
			if !valid {
				return policyCondition{}, false
			}
			children[i] = acceptedCondition
		}
		kind := policyConditionAll
		switch state.kind {
		case ruleConditionAny:
			kind = policyConditionAny
		case ruleConditionNot:
			kind = policyConditionNot
		case ruleConditionAll:
		default:
			return policyCondition{}, false
		}
		if kind == policyConditionNot {
			input := negatedPolicyConditionInput{
				Children: children,
			}
			return acceptNegatedPolicyCondition(input), true
		}
		input := policyConditionGroupInput{
			Kind:     kind,
			Children: children,
		}
		return acceptPolicyConditionGroup(input), true
	default:
		return policyCondition{}, false
	}
}

func policyConditionFromStructural(condition condition) policyCondition {
	children := condition.Children()
	policyChildren := make([]policyCondition, len(children))
	for i, child := range children {
		policyChildren[i] = policyConditionFromStructural(child)
	}
	predicate := func(value any) bool {
		matched, ok := condition.TestResolvedValue(value)
		return ok && matched
	}
	switch condition.Kind() {
	case conditionKindResolved:
		input := resolvedPolicyConditionInput{
			Destination: condition.Destination(),
		}
		return acceptResolvedPolicyCondition(input)
	case conditionKindEqual,
		conditionKindNotEqual,
		conditionKindGreaterThan,
		conditionKindAtLeast,
		conditionKindLessThan,
		conditionKindAtMost:
		kind := policyConditionEqual
		conditionKind := condition.Kind()
		switch conditionKind {
		case conditionKindNotEqual:
			kind = policyConditionNotEqual
		case conditionKindGreaterThan:
			kind = policyConditionGreaterThan
		case conditionKindAtLeast:
			kind = policyConditionAtLeast
		case conditionKindLessThan:
			kind = policyConditionLessThan
		case conditionKindAtMost:
			kind = policyConditionAtMost
		default:
		}
		input := comparisonPolicyConditionInput{
			Kind:        kind,
			Destination: condition.Destination(),
			Operands:    condition.Operands(),
			Predicate:   predicate,
		}
		return acceptComparisonPolicyCondition(input)
	case conditionKindIn, conditionKindNotIn:
		kind := policyConditionIn
		if condition.Kind() == conditionKindNotIn {
			kind = policyConditionNotIn
		}
		input := membershipPolicyConditionInput{
			Kind:        kind,
			Destination: condition.Destination(),
			Operands:    condition.Operands(),
			Predicate:   predicate,
		}
		return acceptMembershipPolicyCondition(input)
	case conditionKindMatches:
		input := patternPolicyConditionInput{
			Destination: condition.Destination(),
			Pattern:     condition.Pattern(),
			Predicate:   predicate,
		}
		return acceptPatternPolicyCondition(input)
	case conditionKindAll, conditionKindAny:
		kind := policyConditionAll
		if condition.Kind() == conditionKindAny {
			kind = policyConditionAny
		}
		input := policyConditionGroupInput{
			Kind:     kind,
			Children: policyChildren,
		}
		return acceptPolicyConditionGroup(input)
	case conditionKindNot:
		input := negatedPolicyConditionInput{
			Children: policyChildren,
		}
		return acceptNegatedPolicyCondition(input)
	default:
		panic("semantic: invalid structural condition")
	}
}

func semanticValueProducerKind(producerKind ValueProducerKind) (ValueProducerKind, bool) {
	switch producerKind {
	case ValueProducerKindFlag:
		return ValueProducerKindFlag, true
	case ValueProducerKindArgument:
		return ValueProducerKindArgument, true
	case ValueProducerKindRemainder:
		return ValueProducerKindRemainder, true
	case ValueProducerKindPassThrough:
		return ValueProducerKindPassThrough, true
	case ValueProducerKindShortcutDefault:
		return ValueProducerKindShortcutDefault, true
	case ValueProducerKindEnvironment:
		return ValueProducerKindEnvironment, true
	case ValueProducerKindInvocationDefault:
		return ValueProducerKindInvocationDefault, true
	case ValueProducerKindComputedDefault:
		return ValueProducerKindComputedDefault, true
	case ValueProducerKindStaticDefault:
		return ValueProducerKindStaticDefault, true
	case ValueProducerKindInitial:
		return ValueProducerKindInitial, true
	case ValueProducerKindSuppliedFact:
		return ValueProducerKindSuppliedFact, true
	case ValueProducerKindResolvedFact:
		return ValueProducerKindResolvedFact, true
	case ValueProducerKindDerived:
		return ValueProducerKindDerived, true
	default:
		return ValueProducerKindUnknown, false
	}
}

func prepareNoticeMatch(match NoticeMatch) (noticeMatch, bool) {
	if !validNoticeMatch(match) {
		return noticeMatch{}, false
	}
	result := noticeMatch{
		All:          match.All,
		Severity:     uint8(match.Severity),
		Code:         string(match.Code),
		AdvisoryCode: string(match.AdvisoryCode),
	}
	return result, true
}

func stageStaticPolicies(draft *declarationCandidate, input declarationPlan, source *declarationDraft, state *staticAcceptanceState) error {
	if draft == nil || state == nil {
		panic("cli: invalid static policy acceptance state")
	}
	ruleSubjects := make([][]ValueID, len(input.records))
	ruleDependencies := make([][]dependencyRequirement, len(input.records))
	ruleLookups := make([]resolvedDestinationDependencies, len(input.records))
	advisorySubjects := make([][]ValueID, len(input.records))
	advisoryDependencies := make([][]dependencyRequirement, len(input.records))
	advisoryLookups := make([]resolvedDestinationDependencies, len(input.records))

	for index := range len(input.records) {
		record := input.records[index]
		kind := record.kind
		switch kind {
		case declarationRecordRuleDirective:
			directive := source.ruleDirectives.entries[record.familyIndex].value
			ruleIndex, indexed := input.indexes.rules[directive.rule]
			if !indexed || ruleIndex >= index || input.records[ruleIndex].commandIndex != record.commandIndex ||
				input.records[ruleIndex].profileIndex != record.profileIndex {
				return errStaticDeclarationRejected
			}
			id := state.mustValueID(directive.destination)
			addDestinationRelationLookup(&ruleLookups[ruleIndex], directive.destination, id)
			switch directive.kind {
			case ruleDirectiveSubject:
				ruleSubjects[ruleIndex] = appendUniqueDestination(ruleSubjects[ruleIndex], id)
			case ruleDirectiveDependency:
				ruleDependencies[ruleIndex] = appendUniqueDependency(
					ruleDependencies[ruleIndex],
					acceptDependencyRequirement(id, directive.optional),
				)
			case ruleDirectiveGroupDependency:
				ruleSubjects[ruleIndex] = appendUniqueDestination(ruleSubjects[ruleIndex], id)
				ruleDependencies[ruleIndex] = appendUniqueDependency(
					ruleDependencies[ruleIndex],
					acceptDependencyRequirement(id, directive.optional),
				)
			default:
				return errStaticDeclarationRejected
			}
		case declarationRecordAdvisoryDirective:
			directive := source.advisoryDirectives.entries[record.familyIndex].value
			advisoryIndex, indexed := input.indexes.advisories[directive.advisory]
			if !indexed || advisoryIndex >= index ||
				input.records[advisoryIndex].commandIndex != record.commandIndex ||
				input.records[advisoryIndex].profileIndex != record.profileIndex {
				return errStaticDeclarationRejected
			}
			id := state.mustValueID(directive.destination)
			addDestinationRelationLookup(&advisoryLookups[advisoryIndex], directive.destination, id)
			switch directive.kind {
			case advisoryDirectiveSubject:
				advisorySubjects[advisoryIndex] = appendUniqueDestination(advisorySubjects[advisoryIndex], id)
			case advisoryDirectiveDependency:
				advisoryDependencies[advisoryIndex] = appendUniqueDependency(
					advisoryDependencies[advisoryIndex],
					acceptDependencyRequirement(id, directive.optional),
				)
			default:
				return errStaticDeclarationRejected
			}
		default:
			continue
		}
	}

	for index := range len(input.records) {
		record := input.records[index]
		position := input.positionFor(index)
		kind := record.kind
		switch kind {
		case declarationRecordRule:
			ref := state.slots[index].rule
			if !ref.ID().present() {
				panic("cli: accepted rule identity is unavailable")
			}
			ruleValue := source.rules.entries[record.familyIndex].value
			if !validRuleDraft(ruleValue, record.commandIndex, record.profileIndex) {
				return errStaticDeclarationRejected
			}
			subjects := append([]ValueID(nil), ruleSubjects[index]...)
			dependencies := append([]dependencyRequirement(nil), ruleDependencies[index]...)
			lookup := &ruleLookups[index]
			for _, relation := range ruleValue.destinations {
				id := state.mustValueID(relation)
				addDestinationRelationLookup(lookup, relation, id)
				subjects = appendUniqueDestination(subjects, id)
				dependencies = appendUniqueDependency(dependencies, acceptDependencyRequirement(id, false))
			}
			var activation *policyCondition
			if ruleValue.activation != nil {
				condition, valid := prepareRuleCondition(ruleValue.activation, state)
				if !valid {
					return errStaticDeclarationRejected
				}
				activation = &condition
			}
			callbackKind := ruleCallbackNone
			var callback ruleCallback
			if ruleValue.callback != nil {
				switch ruleValue.callbackKind {
				case authoringRuleCallbackCheck:
					callbackKind = ruleCallbackCheck
				case authoringRuleCallbackValidate:
					callbackKind = ruleCallbackValidate
				default:
					panic("cli: validated custom rule has invalid callback kind")
				}
				erased := ruleValue.callback
				frozen := freezeResolvedDestinationDependencies(ruleLookups[index])
				callback = func(ctx context.Context, configuration any, dependencies Dependencies) (ruleCallbackResult, error) {
					ctx = contextWithValueStateKeyAliases(ctx, frozen.keys)
					return erased(ctx, configuration, dependencies)
				}
			}
			count := semanticCountRange{}
			if ruleValue.count.valid {
				if ruleValue.count.unbounded {
					count = unboundedCount(ruleValue.count.minimum)
				} else {
					count = boundedCount(ruleValue.count.minimum, ruleValue.count.maximum)
				}
			}
			var builtinPredicate ruleBuiltinPredicate
			if ruleValue.builtinPredicate != nil {
				erased := ruleValue.builtinPredicate
				builtinPredicate = func(values []any) bool { return erased(values) }
			}
			var definition ruleDefinition
			if ruleValue.kind == RuleCustom {
				input := customRuleDefinitionInput{
					Code:         string(ruleValue.code),
					Subjects:     subjects,
					Dependencies: dependencies,
					Activation:   activation,
					Callback:     callback,
				}
				switch callbackKind {
				case ruleCallbackCheck:
					definition = acceptCustomRuleCheckDefinition(input)
				case ruleCallbackValidate:
					definition = acceptCustomRuleValidateDefinition(input)
				default:
					panic("semantic: invalid Rule definition")
				}
			} else {
				input := builtinRuleDefinitionInput{
					Kind:         ruleValue.kind,
					Subjects:     subjects,
					Dependencies: dependencies,
					Operands:     ruleValue.operands,
					Activation:   activation,
					Count:        count,
					Pattern:      ruleValue.pattern,
					Predicate:    builtinPredicate,
				}
				definition = acceptBuiltinRuleDefinition(input)
			}
			draft.RuleDefinition(ref, definition, position)
		case declarationRecordAdvisory:
			ref := state.slots[index].advisory
			if !ref.ID().present() {
				panic("cli: accepted advisory identity is unavailable")
			}
			advisoryValue := source.advisories.entries[record.familyIndex].value
			if !validAdvisoryDraft(advisoryValue, record.commandIndex, record.profileIndex) {
				return errStaticDeclarationRejected
			}
			dependencies := append([]dependencyRequirement(nil), advisoryDependencies[index]...)
			if advisoryValue.kind != AdvisoryKindDetector && len(dependencies) != 0 {
				return errStaticDeclarationRejected
			}
			var activation *condition
			if advisoryValue.kind == AdvisoryKindConditional {
				condition, valid := prepareCondition(advisoryValue.condition, state)
				if !valid {
					return errStaticDeclarationRejected
				}
				activation = &condition
			}
			var callback advisoryCallback
			if advisoryValue.callback != nil {
				erased := advisoryValue.callback
				frozen := freezeResolvedDestinationDependencies(advisoryLookups[index])
				callback = func(ctx context.Context, configuration any, dependencies Dependencies) (bool, error) {
					ctx = contextWithValueStateKeyAliases(ctx, frozen.keys)
					return erased(ctx, configuration, dependencies)
				}
			}
			content := advisoryContent{
				Code:     string(advisoryValue.advisory.Code),
				Severity: uint8(advisoryValue.advisory.Severity),
				Message:  advisoryValue.advisory.Message,
				Reason:   advisoryValue.advisory.Reason,
				Hint:     advisoryValue.advisory.Hint,
			}
			var definition advisoryDefinition
			switch advisoryValue.kind {
			case AdvisoryKindAlways:
				if activation != nil || len(dependencies) != 0 || callback != nil {
					panic("semantic: invalid Advisory definition")
				}
				definition = acceptAlwaysAdvisoryDefinition(content, advisorySubjects[index])
			case AdvisoryKindConditional:
				condition := condition{}
				if activation != nil {
					condition = *activation
				}
				if len(dependencies) != 0 || callback != nil {
					panic("semantic: invalid Advisory definition")
				}
				definition = acceptConditionalAdvisoryDefinition(content, condition, advisorySubjects[index])
			case AdvisoryKindDetector:
				if activation != nil {
					panic("semantic: invalid Advisory definition")
				}
				definition = acceptDetectorAdvisoryDefinition(
					content, advisorySubjects[index], dependencies, callback,
				)
			default:
				panic("semantic: invalid Advisory definition")
			}
			draft.AdvisoryDefinition(ref, definition, position)
		case declarationRecordNoticeSuppression:
			owner := state.mustCommand(record.commandIndex)
			condition, valid := prepareCondition(source.noticeSuppressions.entries[record.familyIndex].value.condition, state)
			if !valid {
				return errStaticDeclarationRejected
			}
			match, valid := prepareNoticeMatch(source.noticeSuppressions.entries[record.familyIndex].value.match)
			if !valid {
				return errStaticDeclarationRejected
			}
			suppressionIndex := draft.NoticeSuppression(owner, match, condition, position)
			if record.profileIndex >= 0 {
				draft.NoticeSuppressionProfile(suppressionIndex, state.mustProfile(record.profileIndex), position)
			}
		default:
			continue
		}
	}
	return nil
}

func (draft *declarationCandidate) validatePolicyReferences(index referenceValidationIndex, rejections []rejection) []rejection {
	commands := index.commands
	destinations := index.destinations
	flags := index.flags
	forms := index.forms
	type conditionPolicyKey struct {
		target conditionTarget
		effect ConditionEffect
	}
	seenConditionPolicies := make(map[conditionPolicyKey][]GrammarProfileID)
	for _, record := range draft.input.ConditionPolicies {
		target := record.target
		validTarget := false
		switch target.Kind() {
		case conditionTargetCommand:
			_, validTarget = commands[target.Command()]
		case conditionTargetFlag:
			_, validTarget = flags[target.Flag()]
		case conditionTargetArgumentForm:
			_, validTarget = forms[target.ArgumentForm()]
		case conditionTargetArgumentTerm:
			for _, term := range draft.input.ArgumentTerms {
				if term.meta.id == target.ArgumentTerm() {
					validTarget = true
					break
				}
			}
		case conditionTargetUnknown:
		}
		if !validTarget {
			rejections = append(rejections, semanticRejection(record.position))
			continue
		}
		key := conditionPolicyKey{
			target: target,
			effect: record.effect,
		}
		for _, priorProfile := range seenConditionPolicies[key] {
			if grammarProfilesOverlap(priorProfile, record.profile) {
				rejections = append(rejections, semanticRejection(record.position))
				break
			}
		}
		seenConditionPolicies[key] = append(seenConditionPolicies[key], record.profile)
		for _, dependency := range record.condition.Dependencies() {
			if _, ok := destinations[dependency]; !ok {
				rejections = append(rejections, semanticRejection(record.position))
			}
		}
		if !structuralAvailabilityPhaseLegal(record, commands, destinations, flags, forms, draft.input.ArgumentTerms) {
			rejections = append(rejections, semanticRejection(record.position))
		}
	}
	for _, record := range draft.input.Providers {
		if _, ok := commands[record.parent]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		definition := record.definition
		if !definition.Valid() {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		seen := make(map[ValueID]struct{})
		for _, dependency := range definition.Dependencies() {
			destination, ok := destinations[dependency.Destination()]
			if !ok || !commandSameOrAncestor(commands, destination.command, record.parent) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
			if _, duplicate := seen[dependency.Destination()]; duplicate {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
			seen[dependency.Destination()] = struct{}{}
		}
	}
	for _, record := range draft.input.Rules {
		if _, ok := commands[record.command]; !ok || !record.definition.Valid() {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		definition := record.definition
		if definition.Kind() == RuleCustom && !validRuleCode(definition.Code()) {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		for _, subject := range definition.Subjects() {
			destination, ok := destinations[subject]
			if !ok || !commandSameOrAncestor(commands, destination.command, record.command) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		for _, dependency := range definition.Dependencies() {
			destination, ok := destinations[dependency.Destination()]
			if !ok || !commandSameOrAncestor(commands, destination.command, record.command) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		if activation, ok := definition.Activation(); ok {
			for _, dependency := range activation.Dependencies() {
				destination, exists := destinations[dependency]
				if !exists || !commandSameOrAncestor(commands, destination.command, record.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
	}
	advisoryCodes := make(map[CommandID]map[string]struct{})
	for _, record := range draft.input.Advisories {
		if _, ok := commands[record.command]; !ok || !record.definition.Valid() {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		definition := record.definition
		severity := NoticeSeverityUnknown
		switch declaredSeverity := NoticeSeverity(definition.Severity()); declaredSeverity {
		case NoticeSeverityUnknown:
		case NoticeSeverityInformation, NoticeSeverityWarning:
			severity = declaredSeverity
		}
		advisoryDeclarationValue := advisoryDeclaration{
			Code:     definition.Code(),
			Severity: severity,
			Message:  definition.Message(),
		}
		if !validateAdvisoryDeclaration(advisoryDeclarationValue) {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		codes := advisoryCodes[record.command]
		if codes == nil {
			codes = make(map[string]struct{})
			advisoryCodes[record.command] = codes
		}
		if _, duplicate := codes[definition.Code()]; duplicate {
			rejections = append(rejections, semanticRejection(record.meta.position))
		} else {
			codes[definition.Code()] = struct{}{}
		}
		for _, subject := range definition.Subjects() {
			destination, ok := destinations[subject]
			if !ok || !commandSameOrAncestor(commands, destination.command, record.command) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		for _, dependency := range definition.Dependencies() {
			destination, ok := destinations[dependency.Destination()]
			if !ok || !commandSameOrAncestor(commands, destination.command, record.command) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		if activation, ok := definition.Activation(); ok {
			for _, dependency := range activation.Dependencies() {
				destination, exists := destinations[dependency]
				if !exists || !commandSameOrAncestor(commands, destination.command, record.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
	}
	for _, record := range draft.input.NoticeSuppressions {
		if _, ok := commands[record.command]; !ok || !record.match.Valid() || !record.condition.Valid() {
			rejections = append(rejections, semanticRejection(record.position))
			continue
		}
		for _, dependency := range record.condition.Dependencies() {
			destination, exists := destinations[dependency]
			if !exists || !commandSameOrAncestor(commands, destination.command, record.command) {
				rejections = append(rejections, semanticRejection(record.position))
			}
		}
	}
	return rejections
}

// policyConditionKind extends the common structural Condition vocabulary with the
// validation-only ContributedBy predicate. It is immutable model data; callback
// execution and evaluation state remain operation-local.
type policyConditionKind uint8

const (
	policyConditionUnknown policyConditionKind = iota
	policyConditionResolved
	policyConditionEqual
	policyConditionNotEqual
	policyConditionIn
	policyConditionNotIn
	policyConditionGreaterThan
	policyConditionAtLeast
	policyConditionLessThan
	policyConditionAtMost
	policyConditionMatches
	policyConditionContributedBy
	policyConditionAll
	policyConditionAny
	policyConditionNot
)

type policyCondition struct {
	kind          policyConditionKind
	destination   ValueID
	operands      []any
	pattern       string
	producerKinds []ValueProducerKind
	children      []policyCondition
	predicate     func(any) bool
}

// resolvedPolicyConditionInput is the payload of one resolved policy condition.
type resolvedPolicyConditionInput struct {
	Destination ValueID
	Predicate   func(any) bool
}

// comparisonPolicyConditionInput is the payload shared by single-operand
// comparison policy-condition variants.
type comparisonPolicyConditionInput struct {
	Kind        policyConditionKind
	Destination ValueID
	Operands    []any
	Predicate   func(any) bool
}

// membershipPolicyConditionInput is the payload shared by set-membership
// policy-condition variants.
type membershipPolicyConditionInput struct {
	Kind        policyConditionKind
	Destination ValueID
	Operands    []any
	Predicate   func(any) bool
}

// patternPolicyConditionInput is the payload of a pattern-match policy condition.
type patternPolicyConditionInput struct {
	Destination ValueID
	Pattern     string
	Predicate   func(any) bool
}

// contributedByPolicyConditionInput is the payload of a producer-kind membership condition.
type contributedByPolicyConditionInput struct {
	Destination   ValueID
	ProducerKinds []ValueProducerKind
	Predicate     func(any) bool
}

// policyConditionGroupInput is the payload shared by all/any policy composition.
type policyConditionGroupInput struct {
	Kind      policyConditionKind
	Children  []policyCondition
	Predicate func(any) bool
}

// negatedPolicyConditionInput is the payload of a not policy condition.
type negatedPolicyConditionInput struct {
	Children  []policyCondition
	Predicate func(any) bool
}

func acceptResolvedPolicyCondition(input resolvedPolicyConditionInput) policyCondition {
	policyConditionValue := policyCondition{
		kind:        policyConditionResolved,
		destination: input.Destination,
		predicate:   input.Predicate,
	}
	return mustPolicyCondition(policyConditionValue)
}

func acceptComparisonPolicyCondition(input comparisonPolicyConditionInput) policyCondition {
	policyConditionValue := policyCondition{
		kind:        input.Kind,
		destination: input.Destination,
		operands:    input.Operands,
		predicate:   input.Predicate,
	}
	return mustPolicyCondition(policyConditionValue)
}

func acceptMembershipPolicyCondition(input membershipPolicyConditionInput) policyCondition {
	policyConditionValue := policyCondition{
		kind:        input.Kind,
		destination: input.Destination,
		operands:    input.Operands,
		predicate:   input.Predicate,
	}
	return mustPolicyCondition(policyConditionValue)
}

func acceptPatternPolicyCondition(input patternPolicyConditionInput) policyCondition {
	policyConditionValue := policyCondition{
		kind:        policyConditionMatches,
		destination: input.Destination,
		pattern:     input.Pattern,
		predicate:   input.Predicate,
	}
	return mustPolicyCondition(policyConditionValue)
}

func acceptContributedByPolicyCondition(input contributedByPolicyConditionInput) policyCondition {
	policyConditionValue := policyCondition{
		kind:          policyConditionContributedBy,
		destination:   input.Destination,
		producerKinds: input.ProducerKinds,
		predicate:     input.Predicate,
	}
	return mustPolicyCondition(policyConditionValue)
}

func acceptPolicyConditionGroup(input policyConditionGroupInput) policyCondition {
	policyConditionValue := policyCondition{
		kind:      input.Kind,
		children:  input.Children,
		predicate: input.Predicate,
	}
	return mustPolicyCondition(policyConditionValue)
}

func acceptNegatedPolicyCondition(input negatedPolicyConditionInput) policyCondition {
	policyConditionValue := policyCondition{
		kind:      policyConditionNot,
		children:  input.Children,
		predicate: input.Predicate,
	}
	return mustPolicyCondition(policyConditionValue)
}

func mustPolicyCondition(condition policyCondition) policyCondition {
	condition.operands = append([]any(nil), condition.operands...)
	condition.producerKinds = append([]ValueProducerKind(nil), condition.producerKinds...)
	condition.children = append([]policyCondition(nil), condition.children...)
	if !condition.Valid() {
		panic("semantic: invalid policy condition")
	}
	return condition
}

func (condition policyCondition) Valid() bool {
	simple := func(operandCount int) bool {
		return condition.destination.present() && len(condition.operands) == operandCount && condition.pattern == "" &&
			len(condition.producerKinds) == 0 &&
			len(condition.children) == 0
	}
	switch condition.kind {
	case policyConditionResolved:
		return simple(0)
	case policyConditionEqual,
		policyConditionNotEqual,
		policyConditionGreaterThan,
		policyConditionAtLeast,
		policyConditionLessThan,
		policyConditionAtMost:
		return simple(1)
	case policyConditionIn, policyConditionNotIn:
		return condition.destination.present() && condition.pattern == "" && len(condition.producerKinds) == 0 &&
			len(condition.children) == 0
	case policyConditionMatches:
		return condition.destination.present() && len(condition.operands) == 0 && len(condition.producerKinds) == 0 &&
			len(condition.children) == 0
	case policyConditionContributedBy:
		if !condition.destination.present() || len(condition.operands) != 0 || condition.pattern != "" ||
			len(condition.producerKinds) == 0 ||
			len(condition.children) != 0 {
			return false
		}
		seen := make(map[ValueProducerKind]struct{}, len(condition.producerKinds))
		for _, producerKind := range condition.producerKinds {
			if !producerKind.valid() {
				return false
			}
			if _, duplicate := seen[producerKind]; duplicate {
				return false
			}
			seen[producerKind] = struct{}{}
		}
		return true
	case policyConditionAll, policyConditionAny:
		if condition.destination.present() || len(condition.operands) != 0 || condition.pattern != "" ||
			len(condition.producerKinds) != 0 ||
			len(condition.children) == 0 {
			return false
		}
		for _, child := range condition.children {
			if !child.Valid() {
				return false
			}
		}
		return true
	case policyConditionNot:
		return !condition.destination.present() && len(condition.operands) == 0 && condition.pattern == "" &&
			len(condition.producerKinds) == 0 &&
			len(condition.children) == 1 &&
			condition.children[0].Valid()
	default:
		return false
	}
}

func (condition policyCondition) Kind() policyConditionKind { return condition.kind }

func (condition policyCondition) Destination() ValueID { return condition.destination }

func (condition policyCondition) Operands() []any { return append([]any(nil), condition.operands...) }

func (condition policyCondition) Pattern() string { return condition.pattern }

func (condition policyCondition) ProducerKinds() []ValueProducerKind {
	return append([]ValueProducerKind(nil), condition.producerKinds...)
}

func (condition policyCondition) Children() []policyCondition {
	return append([]policyCondition(nil), condition.children...)
}

func (condition policyCondition) CanEvaluate() bool {
	if !condition.Valid() {
		return false
	}
	switch condition.kind {
	case policyConditionResolved, policyConditionContributedBy:
		return true
	case policyConditionEqual, policyConditionNotEqual, policyConditionIn, policyConditionNotIn,
		policyConditionGreaterThan, policyConditionAtLeast, policyConditionLessThan, policyConditionAtMost, policyConditionMatches:
		return condition.predicate != nil
	case policyConditionAll, policyConditionAny, policyConditionNot:
		for _, child := range condition.children {
			if !child.CanEvaluate() {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (condition policyCondition) TestResolvedValue(value any) (bool, bool) {
	if !condition.CanEvaluate() || condition.predicate == nil {
		return false, false
	}
	return condition.predicate(value), true
}

func (condition policyCondition) Dependencies() []ValueID {
	if !condition.Valid() {
		return nil
	}
	seen := make(map[ValueID]struct{})
	var result []ValueID
	var visit func(policyCondition)
	visit = func(current policyCondition) {
		if current.destination.present() {
			if _, exists := seen[current.destination]; !exists {
				seen[current.destination] = struct{}{}
				result = append(result, current.destination)
			}
		}
		for _, child := range current.children {
			visit(child)
		}
	}
	visit(condition)
	return result
}

// NoticeSeverity is the private acceptance vocabulary for application Advisory
// declaration validation. The public projection is adapted at the cli boundary.

// advisoryDeclaration is the static subset needed to validate one application
// Advisory before it enters the canonical semantic model.
type advisoryDeclaration struct {
	Code     string
	Severity NoticeSeverity
	Message  string
}

// noticeMatchDeclaration is the static subset needed to validate Notice
// suppression shape. A nonzero narrow match is conjunctive; All is exclusive.
type noticeMatchDeclaration struct {
	All           bool
	Severity      NoticeSeverity
	FrameworkCode string
	AdvisoryCode  string
}

func validRuleCode(code string) bool { return validPolicyCode(code) }

func validAdvisoryCode(code string) bool { return validPolicyCode(code) }

func validPolicyCode(code string) bool {
	if len(code) == 0 || len(code) > 64 {
		return false
	}
	segmentStart := true
	for i := range len(code) {
		c := code[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			segmentStart = false
		case c == '-':
			if segmentStart || i == len(code)-1 {
				return false
			}
			segmentStart = true
		default:
			return false
		}
	}
	return !segmentStart
}

func validateAdvisoryDeclaration(advisory advisoryDeclaration) bool {
	return validAdvisoryCode(advisory.Code) &&
		(advisory.Severity == NoticeSeverityInformation || advisory.Severity == NoticeSeverityWarning) &&
		strings.TrimSpace(advisory.Message) != ""
}

func validateNoticeMatch(match noticeMatchDeclaration) bool {
	narrow := match.Severity != NoticeSeverityUnknown || match.FrameworkCode != "" || match.AdvisoryCode != ""
	if match.All {
		return !narrow
	}
	if !narrow {
		return false
	}
	if match.Severity != NoticeSeverityUnknown && match.Severity != NoticeSeverityInformation &&
		match.Severity != NoticeSeverityWarning {
		return false
	}
	if match.AdvisoryCode != "" && !validAdvisoryCode(match.AdvisoryCode) {
		return false
	}
	return true
}
