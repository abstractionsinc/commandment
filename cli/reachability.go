package cli

// StaticReachability is compiler-owned classification for one legal semantic
// effect. It is deliberately internal: public artifacts project the conclusion
// without exposing a proof/finding object or solver API.

type conditionReachabilityKey struct {
	target   conditionTarget
	effect   ConditionEffect
	position declarationPosition
}

type suppressionReachabilityKey struct {
	command  CommandID
	position declarationPosition
}

// staticAnalysis is a read-only compiler projection of sound reachability
// conclusions for already-accepted canonical records. Its maps are private and
// keyed by existing semantic identity/relationship coordinates; analysis never
// manufactures proof or finding identity.
type staticAnalysis struct {
	profiles     map[GrammarProfileID]Reachability
	defaults     map[CommandDefaultID]Reachability
	forms        map[ArgumentFormID]Reachability
	rules        map[RuleID]Reachability
	advisories   map[advisoryID]Reachability
	signals      map[SignalPolicyID]Reachability
	conditions   map[conditionReachabilityKey]Reachability
	suppressions map[suppressionReachabilityKey]Reachability
}

func (a staticAnalysis) GrammarProfile(id GrammarProfileID) Reachability {
	return lookupReachability(a.profiles, id)
}

func (a staticAnalysis) Default(id CommandDefaultID) Reachability {
	return lookupReachability(a.defaults, id)
}

func (a staticAnalysis) ArgumentForm(id ArgumentFormID) Reachability {
	return lookupReachability(a.forms, id)
}

func (a staticAnalysis) Rule(id RuleID) Reachability {
	return lookupReachability(a.rules, id)
}

func (a staticAnalysis) Advisory(id advisoryID) Reachability {
	return lookupReachability(a.advisories, id)
}

func (a staticAnalysis) SignalPolicy(id SignalPolicyID) Reachability {
	return lookupReachability(a.signals, id)
}

func (a staticAnalysis) Condition(target conditionTarget, effect ConditionEffect, position declarationPosition) Reachability {
	if a.conditions == nil {
		return ReachabilityUnknown
	}
	key := conditionReachabilityKey{
		target:   target,
		effect:   effect,
		position: position,
	}
	return a.conditions[key]
}

func (a staticAnalysis) NoticeSuppression(command CommandID, position declarationPosition) Reachability {
	if a.suppressions == nil {
		return ReachabilityUnknown
	}
	key := suppressionReachabilityKey{
		command:  command,
		position: position,
	}
	return a.suppressions[key]
}

func lookupReachability[I comparable](values map[I]Reachability, id I) Reachability {
	if values == nil {
		return ReachabilityUnknown
	}
	return values[id]
}

// analyzeRecordReachability reuses the accepted proof vocabulary over
// an immutable accepted model. It executes no application callback. The same
// function therefore applies to the static application model and to an
// operation-local model after an accepted provider batch has committed.
func analyzeRecordReachability(records staticRecords) staticAnalysis {
	return analyzeReachability(
		proofUniverseFromSlices(records.domainEntries, records.valueDomains),
		records.grammarProfiles,
		records.conditionPolicies,
		records.argumentForms,
		records.defaults,
		records.rules,
		records.advisories,
		records.signalPolicies,
		records.noticeSuppressions,
	)
}

func emptyStaticAnalysis() staticAnalysis {
	result := staticAnalysis{
		profiles:     make(map[GrammarProfileID]Reachability),
		defaults:     make(map[CommandDefaultID]Reachability),
		forms:        make(map[ArgumentFormID]Reachability),
		rules:        make(map[RuleID]Reachability),
		advisories:   make(map[advisoryID]Reachability),
		signals:      make(map[SignalPolicyID]Reachability),
		conditions:   make(map[conditionReachabilityKey]Reachability),
		suppressions: make(map[suppressionReachabilityKey]Reachability),
	}
	return result
}

func analyzeReachability(universe universe, profiles []grammarProfileRecord, conditionPolicies []conditionPolicyRecord, forms []argumentFormRecord, defaults []defaultRecord, rules []ruleRecord, advisories []advisoryRecord, signals []signalPolicyRecord, suppressions []noticeSuppressionRecord) staticAnalysis {
	result := emptyStaticAnalysis()

	profilesByCommand := make(map[CommandID][]grammarProfileRecord)
	for _, profile := range profiles {
		profilesByCommand[profile.command] = append(profilesByCommand[profile.command], profile)
	}
	for _, profile := range profiles {
		condition, ok := grammarProfileCondition(profile, profilesByCommand[profile.command])
		result.profiles[profile.meta.id] = classifyOwnReachability(condition, ok, universe)
	}

	profileGate := func(id GrammarProfileID) Reachability {
		if !id.present() {
			return ReachabilityReachable
		}
		return result.GrammarProfile(id)
	}

	availabilityByForm := make(map[ArgumentFormID]conditionPolicyRecord)
	for _, policy := range conditionPolicies {
		own, ok := semanticLowerCondition(policy.condition)
		key := conditionReachabilityKey{
			target:   policy.target,
			effect:   policy.effect,
			position: policy.position,
		}
		result.conditions[key] = classifyReachability(own, ok, universe, profileGate(policy.profile))
		if policy.effect == ConditionEffectAvailability && policy.target.Kind() == conditionTargetArgumentForm {
			availabilityByForm[policy.target.ArgumentForm()] = policy
		}
	}

	for _, form := range forms {
		if policy, ok := availabilityByForm[form.meta.id]; ok {
			condition, lowered := semanticLowerCondition(policy.condition)
			result.forms[form.meta.id] = classifyReachability(condition, lowered, universe, profileGate(form.meta.profile))
			continue
		}
		result.forms[form.meta.id] = inheritReachability(ReachabilityReachable, profileGate(form.meta.profile))
	}

	for _, record := range defaults {
		condition := always()
		ok := true
		if declared, conditional := record.Condition(); conditional {
			condition, ok = semanticLowerCondition(declared)
		}
		result.defaults[record.meta.id] = classifyReachability(condition, ok, universe, profileGate(record.meta.profile))
	}

	for _, record := range rules {
		condition := always()
		ok := true
		if activation, present := record.definition.Activation(); present {
			condition, ok = proofPolicyCondition(activation)
		}
		result.rules[record.meta.id] = classifyReachability(condition, ok, universe, profileGate(record.meta.profile))
	}

	for _, record := range advisories {
		condition := always()
		ok := true
		if activation, present := record.definition.Activation(); present {
			condition, ok = semanticLowerCondition(activation)
		}
		result.advisories[record.meta.id] = classifyReachability(condition, ok, universe, profileGate(record.meta.profile))
	}

	for _, record := range signals {
		definition, valid := record.Definition()
		if !valid {
			result.signals[record.meta.id] = ReachabilityUnknown
			continue
		}
		condition := always()
		ok := true
		if activation, present := definition.Activation(); present {
			condition, ok = semanticLowerCondition(activation)
		}
		result.signals[record.meta.id] = classifyReachability(condition, ok, universe, profileGate(record.meta.profile))
	}

	for _, record := range suppressions {
		condition, ok := semanticLowerCondition(record.condition)
		key := suppressionReachabilityKey{
			command:  record.command,
			position: record.position,
		}
		result.suppressions[key] = classifyReachability(condition, ok, universe, profileGate(record.profile))
	}

	return result
}

func classifyOwnReachability(condition proofCondition, lowered bool, universe universe) Reachability {
	if !lowered {
		return ReachabilityUnknown
	}
	switch satisfiable(condition, universe) {
	case proved:
		return ReachabilityReachable
	case disproved:
		return ReachabilityUnreachable
	default:
		return ReachabilityUnknown
	}
}

func classifyReachability(condition proofCondition, lowered bool, universe universe, enclosing Reachability) Reachability {
	return inheritReachability(classifyOwnReachability(condition, lowered, universe), enclosing)
}

func inheritReachability(own, enclosing Reachability) Reachability {
	// An independently impossible effect remains independently unreachable even
	// when its enclosing profile is also impossible.
	if own == ReachabilityUnreachable {
		return own
	}
	switch enclosing {
	case ReachabilityUnreachable, ReachabilityInherited:
		return ReachabilityInherited
	case ReachabilityUnknown:
		return ReachabilityUnknown
	case ReachabilityReachable:
		return own
	default:
		return ReachabilityUnknown
	}
}

func proofPolicyCondition(condition policyCondition) (proofCondition, bool) {
	if !condition.Valid() {
		return proofCondition{}, false
	}
	scalarOperands := func() ([]scalar, bool) {
		operands := condition.Operands()
		result := make([]scalar, len(operands))
		for index, operand := range operands {
			scalar, ok := lowerScalar(operand)
			if !ok {
				return nil, false
			}
			result[index] = scalar
		}
		return result, true
	}
	switch condition.Kind() {
	case policyConditionResolved:
		return resolved(condition.Destination()), true
	case policyConditionEqual, policyConditionNotEqual,
		policyConditionIn, policyConditionNotIn,
		policyConditionGreaterThan, policyConditionAtLeast,
		policyConditionLessThan, policyConditionAtMost:
		operands, ok := scalarOperands()
		if !ok {
			return proofCondition{}, false
		}
		switch condition.Kind() {
		case policyConditionEqual:
			return equal(condition.Destination(), operands[0]), true
		case policyConditionNotEqual:
			return notEqual(condition.Destination(), operands[0]), true
		case policyConditionIn:
			return in(condition.Destination(), operands...), true
		case policyConditionNotIn:
			return notIn(condition.Destination(), operands...), true
		case policyConditionGreaterThan:
			return greaterThan(condition.Destination(), operands[0]), true
		case policyConditionAtLeast:
			return atLeast(condition.Destination(), operands[0]), true
		case policyConditionLessThan:
			return lessThan(condition.Destination(), operands[0]), true
		case policyConditionAtMost:
			return atMost(condition.Destination(), operands[0]), true
		case policyConditionUnknown,
			policyConditionResolved,
			policyConditionMatches,
			policyConditionContributedBy,
			policyConditionAll,
			policyConditionAny,
			policyConditionNot:
		}
	case policyConditionMatches:
		return matches(condition.Destination(), condition.Pattern())
	case policyConditionContributedBy:
		// Source provenance is invocation state, not static theorem input.
		return proofCondition{}, false
	case policyConditionAll, policyConditionAny:
		children := condition.Children()
		lowered := make([]proofCondition, len(children))
		for index, child := range children {
			value, ok := proofPolicyCondition(child)
			if !ok {
				return proofCondition{}, false
			}
			lowered[index] = value
		}
		if condition.Kind() == policyConditionAll {
			return all(lowered[0], lowered[1:]...), true
		}
		return anyCondition(lowered[0], lowered[1:]...), true
	case policyConditionNot:
		child, ok := proofPolicyCondition(condition.Children()[0])
		if !ok {
			return proofCondition{}, false
		}
		return not(child), true
	case policyConditionUnknown:
	}
	return proofCondition{}, false
}
