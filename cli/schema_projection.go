package cli

import (
	"cmp"
	"slices"
)

func projectEffectiveLifecycleState(model *effectiveModel, lifecycle ReleaseLifecycle) (LifecycleState, error) {
	state, err := evaluateLifecycle(model.Application().release, lifecycle, "")
	if err != nil {
		return LifecycleStateUnknown, err
	}
	switch state {
	case LifecycleStateNotIntroduced:
		return LifecycleStateNotIntroduced, nil
	case LifecycleStateCurrent:
		return LifecycleStateCurrent, nil
	case LifecycleStateDeprecated:
		return LifecycleStateDeprecated, nil
	case LifecycleStateRemoved:
		return LifecycleStateRemoved, nil
	default:
		return LifecycleStateUnknown, errInvalidModelTopology
	}
}

func projectEffectiveValueLiteral(model *effectiveModel, destination ValueID, value any, sensitive bool) (ValueLiteral, error) {
	record, ok := model.Destination(destination)
	if !ok {
		return ValueLiteral{}, errInvalidModelTopology
	}
	def, ok := record.Definition()
	if !ok {
		return ValueLiteral{}, errInvalidModelTopology
	}
	vt, ok := def.ValueType()
	if !ok {
		return ValueLiteral{}, errInvalidModelTopology
	}
	language, ok := def.Language()
	if !ok {
		return ValueLiteral{}, errInvalidModelTopology
	}
	if sensitive {
		valueLiteral := ValueLiteral{
			Type:      vt,
			Sensitive: true,
			Redacted:  true,
		}
		return valueLiteral, nil
	}
	text, err := language.Format(value)
	if err != nil {
		return ValueLiteral{}, errInvalidModelTopology
	}
	literal := ValueLiteral{
		Type: vt,
		Text: text,
	}
	return literal, nil
}

func projectApplicationSchema(model staticProgram) (*Schema, error) {
	grammar, available := model.EffectiveGrammar()
	if !available {
		return nil, errInvalidModelTopology
	}
	records := model.Records()
	projector, err := prepareSchemaProjector(
		model.Valid(),
		model.Application(),
		records,
		grammar,
		analyzeRecordReachability(records),
	)
	if err != nil {
		return nil, err
	}
	return projector.project()
}

func releaseLifecycle(v ReleaseLifecycle) ReleaseLifecycle {
	releaseLifecycle := ReleaseLifecycle{
		IntroducedIn: v.IntroducedIn,
		DeprecatedIn: v.DeprecatedIn,
		RemovedIn:    v.RemovedIn,
		Reason:       v.Reason,
	}

	return releaseLifecycle
}

func countRange(v semanticCountRange) CountRange {
	maximum, bounded := v.Maximum()
	countRange := CountRange{
		minimum:   v.Minimum(),
		maximum:   maximum,
		unbounded: !bounded,
		valid:     true,
	}

	return countRange
}

func projectCompletion(v completionDeclaration, present bool) (CompletionDeclarationCapabilities, error) {
	if !present {
		return CompletionDeclarationCapabilities{}, nil
	}
	r := CompletionDeclarationCapabilities{
		SourceSensitive: v.Source().Sensitive(),
		Dependencies:    dependencyIDs(v.Dependencies()),
	}
	switch v.Kind() {
	case completionDeclarationKindValue, completionDeclarationKindTokenLanguage:
		r.Dynamic = true
	case completionDeclarationKindDelegation:
		r.Dynamic = true
		r.Delegated = true
	case completionDeclarationKindNative:
	default:
		return r, errInvalidModelTopology
	}
	if k, m, ok := v.Native(); ok {
		r.Native = &CompletionNativeIntent{
			Kind: k,
			Mode: m,
		}
	}
	return r, nil
}

func projectStaticDefaultRecord(record destinationRecord, value any, sensitive bool) (ValueStaticDefault, error) {
	definition, ok := record.Definition()
	if !ok {
		return ValueStaticDefault{}, errInvalidModelTopology
	}
	result := ValueStaticDefault{
		Sensitive: sensitive,
	}
	switch definition.Kind() {
	case destinationKindScalar:
		semanticType, hasType := definition.ValueType()
		language, hasLanguage := definition.Language()
		if !hasType || !hasLanguage {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		result.HasRepresentation = true
		result.Redacted = sensitive
		literal := ValueLiteral{
			Type:      semanticType,
			Sensitive: sensitive,
			Redacted:  sensitive,
		}
		if !sensitive {
			text, err := language.Format(value)
			if err != nil {
				return ValueStaticDefault{}, errInvalidModelTopology
			}
			literal.Text = text
		}
		result.Scalar = &literal
		return result, nil
	case destinationKindSequence:
		elementType, ok := definition.ElementType()
		if !ok {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		language, ok := definition.Language()
		if !ok {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		result.HasRepresentation = true
		result.Redacted = sensitive
		sequence := &ValueSequenceLiteral{
			ElementType: elementType,
		}
		result.Sequence = sequence
		if sensitive {
			return result, nil
		}
		elements, ok := definition.UnpackSequence(value)
		if !ok {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		for _, element := range elements {
			text, err := language.Format(element)
			if err != nil {
				return ValueStaticDefault{}, errInvalidModelTopology
			}
			sequence.Elements = append(sequence.Elements, ValueLiteral{
				Type: elementType,
				Text: text,
			})
		}
		return result, nil
	case destinationKindKeyed:
		keyType, ok := definition.KeyType()
		if !ok {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		entryValueType, ok := definition.KeyedValueType()
		if !ok {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		language, ok := definition.KeyedLanguage()
		if !ok {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		result.HasRepresentation = true
		result.Redacted = sensitive
		keyed := &ValueKeyedLiteral{
			KeyType:   keyType,
			ValueType: entryValueType,
		}
		result.Keyed = keyed
		if sensitive {
			return result, nil
		}
		entries, ok := definition.UnpackKeyed(value)
		if !ok {
			return ValueStaticDefault{}, errInvalidModelTopology
		}
		type formattedEntry struct {
			keyText   string
			key       any
			valueText string
		}
		formatted := make([]formattedEntry, 0, len(entries))
		seen := make(map[string]any, len(entries))
		for _, entry := range entries {
			keyText, err := language.Key().Format(entry.Key)
			if err != nil {
				return ValueStaticDefault{}, errInvalidModelTopology
			}
			if prior, exists := seen[keyText]; exists && prior != entry.Key {
				return ValueStaticDefault{}, errInvalidModelTopology
			}
			seen[keyText] = entry.Key
			valueText, err := language.Value().Format(entry.Value)
			if err != nil {
				return ValueStaticDefault{}, errInvalidModelTopology
			}
			entry2 := formattedEntry{
				keyText:   keyText,
				key:       entry.Key,
				valueText: valueText,
			}
			formatted = append(formatted, entry2)
		}
		slices.SortStableFunc(formatted, func(left, right formattedEntry) int { return cmp.Compare(left.keyText, right.keyText) })
		for _, entry := range formatted {
			keyed.Entries = append(keyed.Entries, ValueKeyedEntry{
				Key: ValueLiteral{
					Type: keyType,
					Text: entry.keyText,
				},
				Value: ValueLiteral{
					Type: entryValueType,
					Text: entry.valueText,
				},
			})
		}
		return result, nil
	case destinationKindOpaque:
		// An ordinary opaque destination can own a valid typed runtime default
		// without owning a detached CLI language or canonical formatter.
		return result, nil
	case destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		return ValueStaticDefault{}, errInvalidModelTopology
	default:
		return ValueStaticDefault{}, errInvalidModelTopology
	}
}

func projectValueDestinationKind(kind destinationKind) (ValueDestinationKind, bool) {
	switch kind {
	case destinationKindScalar:
		return ValueDestinationScalar, true
	case destinationKindSequence:
		return ValueDestinationSequence, true
	case destinationKindKeyed:
		return ValueDestinationKeyed, true
	case destinationKindOpaque:
		return ValueDestinationOpaque, true
	case destinationKindDerived:
		return ValueDestinationDerived, true
	case destinationKindOptionalDerived:
		return ValueDestinationOptionalDerived, true
	case destinationKindFactRequired:
		return ValueDestinationFactRequired, true
	case destinationKindFactResolved:
		return ValueDestinationFactResolved, true
	default:
		return ValueDestinationUnknown, false
	}
}

func dependencyIDs(v []dependencyRequirement) []ValueID {
	out := make([]ValueID, len(v))
	for i, d := range v {
		out[i] = d.Destination()
	}
	return out
}

func appendUniqueValueIDs(base []ValueID, more ...ValueID) []ValueID {
	seen := map[ValueID]struct{}{}
	for _, v := range base {
		seen[v] = struct{}{}
	}
	for _, v := range more {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			base = append(base, v)
		}
	}
	return base
}

func invocationAction(v intrinsicAction) InvocationAction {
	switch v {
	case intrinsicActionHelp:
		return InvocationActionHelp
	case intrinsicActionVersion:
		return InvocationActionVersion
	default:
		return InvocationActionUnknown
	}
}

func flagValueForms(v []flagValueForm) []FlagValueForm {
	out := make([]FlagValueForm, len(v))
	for i, x := range v {
		out[i] = FlagValueForm(x)
	}
	return out
}

func valueDomainEntryKind(v domainEntryKind) ValueDomainEntryKind {
	switch v {
	case domainEntryKindCanonical:
		return ValueDomainEntryCanonical
	case domainEntryKindAlias:
		return ValueDomainEntryAlias
	default:
		return ValueDomainEntryUnknown
	}
}

func valueCandidateContribution(v environmentContribution) (ValueCandidateContribution, bool) {
	switch v {
	case environmentContributionScalar:
		return ValueCandidateContributionWhole, true
	case environmentContributionSequenceElements:
		return ValueCandidateContributionSequenceSegment, true
	case environmentContributionWholeSequence:
		return ValueCandidateContributionWhole, true
	case environmentContributionKeyedEntries:
		return ValueCandidateContributionKeyedEntry, true
	case environmentContributionWholeKeyed:
		return ValueCandidateContributionKeyedReplacement, true
	default:
		return ValueCandidateContributionUnknown, false
	}
}

func cloneExecutionAccess(access ExecutionAccess) ExecutionAccess {
	access.Values = append([]ValueID(nil), access.Values...)
	access.OptionalValues = append([]ValueID(nil), access.OptionalValues...)
	access.ValueStates = append([]ValueID(nil), access.ValueStates...)
	return access
}
