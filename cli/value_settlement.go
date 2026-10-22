package cli

import (
	"context"
	"errors"
)

var errUnsupportedValueProducerKind = errors.New("invocation: unsupported value producer kind")

const producerResolutionOrderBase uint64 = 1_000_000

type valueSettlementProducer struct {
	producer     valueProducerRecord
	dependencies []dependencyRequirement
	order        int
}

type valueSettlementNode struct {
	record        destinationRecord
	producers     []valueSettlementProducer
	prerequisites []ValueID
}

func prepareValueSettlementNode(record destinationRecord, profiles map[CommandID]GrammarProfileID) valueSettlementNode {
	node := valueSettlementNode{
		record: record,
	}
	var seen map[ValueID]struct{}
	for order, producer := range record.Producers() {
		if !analysisGrammarProfileApplies(profiles, record.command, producer.profile) {
			continue
		}
		dependencies := producer.Dependencies()
		valueSettlementProducerValue := valueSettlementProducer{
			producer:     producer,
			dependencies: dependencies,
			order:        order,
		}
		node.producers = append(node.producers, valueSettlementProducerValue)
		for _, dependency := range dependencies {
			id := dependency.Destination()
			if seen == nil {
				seen = make(map[ValueID]struct{})
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			node.prerequisites = append(node.prerequisites, id)
		}
	}
	return node
}

func settleValueDestinations(ctx context.Context, operation *invocation, state *analysisState, model grammarView, grammar grammarResult, nodes []valueSettlementNode, plans map[ValueID][]valueCandidatePlan, analysis *valueAnalysis) {
	if len(nodes) == 0 {
		return
	}
	activeInputProducers := activeInputProducerDestinations(state, model, grammar)
	hasPrerequisites := false
	for i := range nodes {
		if len(nodes[i].prerequisites) != 0 {
			hasPrerequisites = true
			break
		}
	}
	if !hasPrerequisites {
		for i := range nodes {
			node := nodes[i]
			id := node.record.meta.id
			if _, terminal := analysis.Destinations[id]; terminal {
				continue
			}
			resolveDestination(ctx, operation, state, node, plans[id], analysis)
			resolved := analysis.Destinations[id]
			if resolved.State == destinationStateAbsent &&
				!destinationProducerActive(activeInputProducers, node.record, len(node.producers) != 0) {
				resolved.State = destinationStateInactive
				analysis.Destinations[id] = resolved
			}
		}
		return
	}
	indexes := make(map[ValueID]int, len(nodes))
	for i := range nodes {
		indexes[nodes[i].record.meta.id] = i
	}
	reverse := make([][]int, len(nodes))
	unmet := make([]int, len(nodes))
	settled := make([]bool, len(nodes))
	for i := range nodes {
		id := nodes[i].record.meta.id
		if _, terminal := analysis.Destinations[id]; terminal {
			settled[i] = true
			continue
		}
		for _, pre := range nodes[i].prerequisites {
			if _, terminal := analysis.Destinations[pre]; terminal {
				continue
			}
			pi, local := indexes[pre]
			unmet[i]++
			if local {
				reverse[pi] = append(reverse[pi], i)
			}
		}
	}
	ready := make([]int, 0, len(nodes))
	for i := range nodes {
		if !settled[i] && unmet[i] == 0 {
			ready = append(ready, i)
		}
	}
	for at := 0; at < len(ready); at++ {
		i := ready[at]
		if settled[i] {
			continue
		}
		node := nodes[i]
		id := node.record.meta.id
		resolveDestination(ctx, operation, state, node, plans[id], analysis)
		resolved := analysis.Destinations[id]
		if resolved.State == destinationStateAbsent &&
			!destinationProducerActive(activeInputProducers, node.record, len(node.producers) != 0) {
			resolved.State = destinationStateInactive
			analysis.Destinations[id] = resolved
		}
		settled[i] = true
		for _, dep := range reverse[i] {
			if unmet[dep] > 0 {
				unmet[dep]--
			}
			if unmet[dep] == 0 && !settled[dep] {
				ready = append(ready, dep)
			}
		}
	}
	for i := range nodes {
		if !settled[i] {
			analysis.Unsupported = true
			analysis.Issues = append(analysis.Issues, valueIssue{
				Kind:        valueIssueUnsupported,
				Destination: nodes[i].record.meta.id,
			})
		}
	}
}

func resolveDestination(ctx context.Context, operation *invocation, state *analysisState, node valueSettlementNode, plans []valueCandidatePlan, analysis *valueAnalysis) {
	record := node.record
	definition, ok := record.Definition()
	if !ok {
		analysis.Unsupported = true
		analysis.Destinations[record.meta.id] = destinationResolution{
			Destination: record.meta.id,
			State:       destinationStateInvalid,
		}
		return
	}
	allPlans := append([]valueCandidatePlan(nil), plans...)
	producerObservations := make([]producerObservation, 0, len(plans)+len(node.producers))
	for index := range allPlans {
		allPlans[index].sensitive = allPlans[index].sensitive || state.originsSensitive(allPlans[index].origins)
		allPlans[index].producerIndex = len(producerObservations)
		observation2 := producerObservation{
			ProducerKind: allPlans[index].producerKind,
			Tier:         allPlans[index].tier,
			Outcome:      producerOutcomeSkipped,
			Sensitive:    allPlans[index].sensitive,
			Origins:      append([]InputOriginID(nil), allPlans[index].origins...),
		}
		producerObservations = append(producerObservations, observation2)
	}
	for _, pp := range node.producers {
		observation, plan, hasPlan := producerValueCandidatePlan(
			ctx,
			operation,
			state,
			record,
			pp.producer,
			pp.order,
			pp.dependencies,
			analysis.Destinations,
			producerResolutionOrderBase+checkedUint64FromInt(pp.order),
		)
		producerIndex := len(producerObservations)
		producerObservations = append(producerObservations, observation)
		if hasPlan {
			plan.producerIndex = producerIndex
			allPlans = append(allPlans, plan)
		}
		if observation.Err != nil {
			analysis.Destinations[record.meta.id] = destinationResolution{
				Destination: record.meta.id,
				State:       destinationStateFailed,
				Producers:   producerObservations,
				Err:         observation.Err,
			}
			valueIssueValue := valueIssue{
				Kind:        valueIssueProducerFailure,
				Destination: record.meta.id,
				Err:         observation.Err,
			}
			analysis.Issues = append(analysis.Issues, valueIssueValue)
			return
		}
		if len(observation.BlockedBy) != 0 {
			analysis.Destinations[record.meta.id] = destinationResolution{
				Destination: record.meta.id,
				State:       destinationStateBlocked,
				Producers:   producerObservations,
				BlockedBy:   append([]ValueID(nil), observation.BlockedBy...),
			}
			analysis.Issues = append(analysis.Issues, valueIssue{
				Kind:        valueIssueProducerBlocked,
				Destination: record.meta.id,
			})
			return
		}
	}
	policy := definition.Resolution()
	switch definition.Kind() {
	case destinationKindScalar, destinationKindOpaque:
		resolveScalarDestination(state, record, policy, allPlans, producerObservations, analysis)
	case destinationKindSequence:
		resolveSequenceDestination(state, record, definition, policy, allPlans, producerObservations, analysis)
	case destinationKindKeyed:
		resolveKeyedDestination(state, record, definition, policy, allPlans, producerObservations, analysis)
	case destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		resolveExclusiveDestination(state, record, definition, allPlans, producerObservations, analysis)
	default:
		analysis.Unsupported = true
		analysis.Destinations[record.meta.id] = destinationResolution{
			Destination: record.meta.id,
			State:       destinationStateInvalid,
			Producers:   producerObservations,
		}
	}
}

func producerValueCandidatePlan(ctx context.Context, operation *invocation, state *analysisState, record destinationRecord, producer valueProducerRecord, producerOrdinal int, dependencies []dependencyRequirement, settled map[ValueID]destinationResolution, order uint64) (producerObservation, valueCandidatePlan, bool) {
	observation := producerObservation{
		Kind:    producer.kind,
		Outcome: producerOutcomeSkipped,
	}
	sensitive := producer.source.Sensitive()
	var blockedBy []ValueID
	for _, dependency := range dependencies {
		observation.Dependencies = append(observation.Dependencies, dependency.Destination())
		resolution, ok := settled[dependency.Destination()]
		if ok && resolution.State == destinationStateResolved && resolution.Sensitive {
			sensitive = true
		}
		if !ok || resolution.State != destinationStateResolved {
			if !dependency.Optional() {
				blockedBy = append(blockedBy, dependency.Destination())
			}
		}
	}
	observation.Sensitive = sensitive
	if len(blockedBy) != 0 {
		switch producer.kind {
		case declaredProducerKindDerived,
			declaredProducerKindOptionalDerived,
			declaredProducerKindFact:
			observation.Outcome = producerOutcomeBlocked
			observation.BlockedBy = append([]ValueID(nil), blockedBy...)
			return observation, valueCandidatePlan{}, false
		case declaredProducerKindUnknown,
			declaredProducerKindDefault,
			declaredProducerKindComputedDefault,
			declaredProducerKindOptionalComputedDefault:
		}
	}
	configuration, _ := operation.analysisConfigurationForDependenciesAt(state, record.command, dependencies, settled)
	var producerKind ValueProducerKind
	var tier ResolutionTier
	switch producer.kind {
	case declaredProducerKindDefault:
		producerKind, tier = ValueProducerKindStaticDefault, ResolutionTierStaticDefault
	case declaredProducerKindComputedDefault, declaredProducerKindOptionalComputedDefault:
		producerKind, tier = ValueProducerKindComputedDefault, ResolutionTierComputedDefault
	case declaredProducerKindDerived, declaredProducerKindOptionalDerived:
		producerKind = ValueProducerKindDerived
	case declaredProducerKindFact:
		producerKind = ValueProducerKindResolvedFact
	default:
		observation.Err = errUnsupportedValueProducerKind
		return observation, valueCandidatePlan{}, false
	}
	observation.ProducerKind, observation.Tier = producerKind, tier
	tracker := &producerTracker{}
	evaluate := func() (any, error) {
		tracker.called = true
		if len(blockedBy) != 0 {
			tracker.blockedBy = append([]ValueID(nil), blockedBy...)
			failure := producerBlockedError{
				destinations: append([]ValueID(nil), blockedBy...),
			}
			return nil, failure
		}
		var result valueEvaluation
		var err error
		if producer.kind == declaredProducerKindDefault {
			value, present := producer.Static()
			if !present {
				panic("cli: static value producer has no value")
			}
			result = valuePresent(value)
		} else {
			callback, ok := state.valueProducerCallback(record.meta.id, producerOrdinal)
			if !ok {
				panic("cli: accepted value producer has no callback")
			}
			dependencyView, ready := dependenciesFromValueAnalysis(
				operation.scope,
				state.effectiveModel,
				dependencies,
				valueAnalysis{
					Destinations: settled,
				},
			)
			if !ready {
				panic("cli: value producer dependencies are not ready")
			}
			result, err = callback(ctx, configuration, dependencyView)
		}
		if err != nil {
			tracker.err = err
			return nil, err
		}
		value, present := result.Value()
		tracker.present = present
		if !present {
			return nil, errValueCandidateOmitted
		}
		return value, nil
	}
	if producer.kind == declaredProducerKindDefault {
		observation.Outcome = producerOutcomeContributed
	}
	plan := valueCandidatePlan{
		producerKind:  producerKind,
		tier:          tier,
		order:         order,
		sensitive:     sensitive,
		producerIndex: -1,
		tracker:       tracker,
		whole:         evaluate,
	}
	return observation, plan, true
}

func resolveScalarDestination(state *analysisState, record destinationRecord, policy semanticResolutionPolicy, plans []valueCandidatePlan, producers []producerObservation, analysis *valueAnalysis) {
	inputs := make([]scalarValueCandidateInput[any], 0, len(plans))
	memo := make([]*memoValue, len(plans))
	for i, plan := range plans {
		if plan.whole == nil {
			analysis.Unsupported = true
			continue
		}
		memo[i] = memoizeValue(plan.whole)
		composite := scalarValueCandidateInput[any]{
			ProducerKind: plan.producerKind,
			Tier:         plan.tier,
			Order:        plan.order,
			Sensitive:    plan.sensitive,
			Origins:      plan.origins,
			Evaluate:     memo[i].get,
		}
		inputs = append(inputs, composite)
	}
	resolver := prepareScalarResolver[any](
		state,
		record.meta.id,
		policy.Duplicate(),
		policy.RejectMultipleTiers(),
		inputs,
	)
	resolved := resolver.Resolve()
	result := destinationResolution{
		Destination: record.meta.id,
		State:       resolved.State,
		Sensitive:   record.sensitive || resolved.Sensitive,
		Producers:   producers,
		Err:         resolved.Err,
	}
	if blocked, ok := errors.AsType[producerBlockedError](resolved.Err); ok {
		result.State = destinationStateBlocked
		result.BlockedBy = append([]ValueID(nil), blocked.destinations...)
	}
	projectValueCandidateMetas(plans, memo, resolved.Candidates, analysis, &result)
	if resolved.State == destinationStateResolved {
		result.Value = snapshotMutableValue(resolved.Value)
	}
	if resolved.Err != nil {
		kind := resolutionIssueKind(resolved.Err)
		if result.State == destinationStateBlocked {
			kind = valueIssueProducerBlocked
		}
		valueIssueValue := valueIssue{
			Kind:        kind,
			Destination: record.meta.id,
			Err:         resolved.Err,
		}
		analysis.Issues = append(analysis.Issues, valueIssueValue)
	}
	result.Producers = finalizeProducerOutcomes(record.meta.id, result.Producers, plans, analysis.Candidates)
	analysis.Destinations[record.meta.id] = result
}

func resolveSequenceDestination(state *analysisState, record destinationRecord, definition destinationDefinition, policy semanticResolutionPolicy, plans []valueCandidatePlan, producers []producerObservation, analysis *valueAnalysis) {
	inputs := make([]sequenceValueCandidateInput[any], 0, len(plans))
	keptPlans := make([]valueCandidatePlan, 0, len(plans))
	keptMemos := make([]*memoSequence, 0, len(plans))
	for _, plan := range plans {
		var evaluate func() ([]any, error)
		kind := sequenceContributionSegment
		switch {
		case plan.sequence != nil:
			evaluate = plan.sequence
			if plan.sequenceWhole {
				kind = sequenceContributionWhole
			}
		case plan.whole != nil:
			kind = sequenceContributionWhole
			whole := memoizeValue(plan.whole)
			evaluate = func() ([]any, error) {
				value, err := whole.get()
				if err != nil {
					return nil, err
				}
				values, ok := definition.UnpackSequence(value)
				if !ok {
					return nil, errValueLanguageTypeMismatch
				}
				return values, nil
			}
		default:
			analysis.Unsupported = true
			continue
		}
		memo := memoizeSequence(evaluate)
		composite := sequenceValueCandidateInput[any]{
			ProducerKind: plan.producerKind,
			Tier:         plan.tier,
			Order:        plan.order,
			Sensitive:    plan.sensitive,
			Origins:      plan.origins,
			Kind:         kind,
			Evaluate:     memo.get,
		}
		inputs = append(inputs, composite)
		keptPlans = append(keptPlans, plan)
		keptMemos = append(keptMemos, memo)
	}
	resolver := prepareSequenceResolver[any](state, record.meta.id, policy.Combination(), inputs)
	resolved := resolver.Resolve()
	result := destinationResolution{
		Destination: record.meta.id,
		State:       resolved.State,
		Sensitive:   record.sensitive || resolved.Sensitive,
		Producers:   producers,
		Err:         resolved.Err,
	}
	if blocked, ok := errors.AsType[producerBlockedError](resolved.Err); ok {
		result.State = destinationStateBlocked
		result.BlockedBy = append([]ValueID(nil), blocked.destinations...)
	}
	projectSequenceValueCandidateMetas(keptPlans, keptMemos, resolved.Candidates, analysis, &result)
	if resolved.State == destinationStateResolved {
		result.Elements = sequenceComponentReferences(resolved.Candidates, analysis.Candidates, policy.Combination())
		packed, ok := definition.PackSequence(resolved.Value)
		if !ok {
			analysis.Unsupported = true
			result.State = destinationStateInvalid
		} else {
			result.Value = snapshotMutableValue(packed)
		}
	}
	if resolved.Err != nil {
		kind := resolutionIssueKind(resolved.Err)
		if result.State == destinationStateBlocked {
			kind = valueIssueProducerBlocked
		}
		valueIssueValue := valueIssue{
			Kind:        kind,
			Destination: record.meta.id,
			Err:         resolved.Err,
		}
		analysis.Issues = append(analysis.Issues, valueIssueValue)
	}
	result.Producers = finalizeProducerOutcomes(record.meta.id, result.Producers, plans, analysis.Candidates)
	analysis.Destinations[record.meta.id] = result
}

func resolveKeyedDestination(state *analysisState, record destinationRecord, definition destinationDefinition, policy semanticResolutionPolicy, plans []valueCandidatePlan, producers []producerObservation, analysis *valueAnalysis) {
	inputs := make([]keyedBatchInput, 0, len(plans))
	keptPlans := make([]valueCandidatePlan, 0, len(plans))
	for _, plan := range plans {
		input := keyedBatchInput{
			ProducerKind: plan.producerKind,
			Tier:         plan.tier,
			Order:        plan.order,
			Sensitive:    plan.sensitive,
			Origins:      append([]InputOriginID(nil), plan.origins...),
		}
		switch {
		case plan.keyedReplacement:
			input.Kind = keyedBatchReplacement
			entries := append([]keyedEntryPlan(nil), plan.keyedEntries...)
			input.Whole = func() ([]keyedValueEntry, error) {
				return evaluateKeyedEntryPlans(entries)
			}
		case plan.keyedBatch:
			input.Kind = keyedBatchEntries
			for _, entry := range plan.keyedEntries {
				input.Entries = append(input.Entries, keyedBatchEntryInput{
					Key:   entry.key,
					Value: entry.value,
				})
			}
		case plan.key != nil && plan.value != nil:
			input.Kind = keyedBatchEntries
			input.Entries = []keyedBatchEntryInput{{
				Key:   plan.key,
				Value: plan.value,
			}}
		case plan.whole != nil:
			input.Kind = keyedBatchReplacement
			whole := memoizeValue(plan.whole)
			input.Whole = func() ([]keyedValueEntry, error) {
				value, err := whole.get()
				if err != nil {
					return nil, err
				}
				entries, ok := definition.UnpackKeyed(value)
				if !ok {
					return nil, errValueLanguageTypeMismatch
				}
				return entries, nil
			}
		default:
			analysis.Unsupported = true
			continue
		}
		inputs = append(inputs, input)
		keptPlans = append(keptPlans, plan)
	}

	resolved := resolveKeyedBatches(state, record.meta.id, policy.Duplicate(), definition.EqualKey, inputs)
	result := destinationResolution{
		Destination: record.meta.id,
		State:       resolved.State,
		Sensitive:   record.sensitive || resolved.Sensitive,
		Producers:   producers,
		Err:         resolved.Err,
	}
	if blocked, ok := errors.AsType[producerBlockedError](resolved.Err); ok {
		result.State = destinationStateBlocked
		result.BlockedBy = append([]ValueID(nil), blocked.destinations...)
	}
	projectKeyedBatchValueCandidates(keptPlans, resolved.Candidates, analysis, &result)
	installResolvedKeyedDestination(definition, resolved, analysis, &result)
	if resolved.Err != nil {
		kind := resolutionIssueKind(resolved.Err)
		if result.State == destinationStateBlocked {
			kind = valueIssueProducerBlocked
		}
		valueIssueValue := valueIssue{
			Kind:        kind,
			Destination: record.meta.id,
			Err:         resolved.Err,
		}
		analysis.Issues = append(analysis.Issues, valueIssueValue)
	}
	result.Producers = finalizeProducerOutcomes(record.meta.id, result.Producers, plans, analysis.Candidates)
	analysis.Destinations[record.meta.id] = result
}

func installResolvedKeyedDestination(definition destinationDefinition, resolved keyedBatchResolution, analysis *valueAnalysis, result *destinationResolution) {
	if resolved.State != destinationStateResolved {
		return
	}
	for _, ref := range resolved.EntryRefs {
		if ref.CandidateIndex < 0 || ref.CandidateIndex >= len(resolved.Candidates) {
			analysis.Unsupported = true
			continue
		}
		candidate := resolved.Candidates[ref.CandidateIndex]
		result.Entries = append(result.Entries, valueCandidateComponentReference{
			Candidate: candidate.Meta.ID,
			Index:     ref.EntryIndex,
		})
	}
	packed, ok := definition.PackKeyed(resolved.Entries)
	if !ok {
		analysis.Unsupported = true
		result.State = destinationStateInvalid
		return
	}
	result.Value = snapshotMutableValue(packed)
}

func evaluateKeyedEntryPlans(entries []keyedEntryPlan) ([]keyedValueEntry, error) {
	result := make([]keyedValueEntry, len(entries))
	for i, entry := range entries {
		if entry.key == nil || entry.value == nil {
			return nil, errValueLanguageTypeMismatch
		}
		key, err := entry.key()
		if err != nil {
			return nil, err
		}
		value, err := entry.value()
		if err != nil {
			return nil, err
		}
		result[i] = keyedValueEntry{
			Key:   key,
			Value: value,
		}
	}
	return result, nil
}

func resolveExclusiveDestination(state *analysisState, record destinationRecord, definition destinationDefinition, plans []valueCandidatePlan, producers []producerObservation, analysis *valueAnalysis) {
	if definition.Kind() == destinationKindFactRequired ||
		definition.Kind() == destinationKindFactResolved {
		resolveFactDestination(state, record, definition, plans, producers, analysis)
		return
	}
	if len(plans) == 0 {
		state := destinationStateAbsent
		if definition.Kind() == destinationKindDerived {
			state = destinationStateBlocked
		}
		analysis.Destinations[record.meta.id] = destinationResolution{
			Destination: record.meta.id,
			State:       state,
			Producers:   finalizeProducerOutcomes(record.meta.id, producers, plans, analysis.Candidates),
		}
		return
	}
	if len(plans) != 1 || plans[0].whole == nil {
		analysis.Unsupported = true
		analysis.Destinations[record.meta.id] = destinationResolution{
			Destination: record.meta.id,
			State:       destinationStateInvalid,
			Producers:   producers,
		}
		return
	}
	plan := plans[0]
	meta := exclusiveValueCandidateMeta(state, record.meta.id, plan)
	value, err := plan.whole()
	result := destinationResolution{
		Destination: record.meta.id,
		Producers:   producers,
	}
	observation := valueCandidateObservation{
		Meta:          meta,
		ProducerIndex: plan.producerIndex,
		Declaration:   plan.declaration,
	}
	switch {
	case errors.Is(err, errValueCandidateOmitted):
		meta.Disposition = ValueCandidateDispositionNoEffect
		observation.Meta = meta
		result.State = destinationStateAbsent
		if definition.Kind() == destinationKindDerived {
			result.State = destinationStateBlocked
		}
	case err != nil:
		meta.Disposition = ValueCandidateDispositionRejected
		observation.Meta = meta
		result.State, result.Err = destinationStateFailed, err
		valueIssueValue := valueIssue{
			Kind:        valueIssueProducerFailure,
			Destination: record.meta.id,
			Err:         err,
		}
		analysis.Issues = append(analysis.Issues, valueIssueValue)
	default:
		meta.Disposition = ValueCandidateDispositionEffective
		observation.Meta, observation.Value, observation.HasValue = meta, snapshotMutableValue(value), true
		result.State, result.Value, result.Sensitive = destinationStateResolved, snapshotMutableValue(
			value,
		), record.sensitive ||
			plan.sensitive
	}
	analysis.Candidates[meta.ID] = observation
	result.Candidates = []valueCandidateRef{meta.ID}
	if meta.Disposition == ValueCandidateDispositionEffective {
		result.Contributors = []valueCandidateRef{meta.ID}
	}
	result.Producers = finalizeProducerOutcomes(record.meta.id, producers, plans, analysis.Candidates)
	analysis.Destinations[record.meta.id] = result
}

func resolveFactDestination(state *analysisState, record destinationRecord, definition destinationDefinition, plans []valueCandidatePlan, producers []producerObservation, analysis *valueAnalysis) {
	var supplied, resolved []valueCandidatePlan
	for _, plan := range plans {
		switch plan.producerKind {
		case ValueProducerKindSuppliedFact:
			supplied = append(supplied, plan)
		case ValueProducerKindResolvedFact:
			resolved = append(resolved, plan)
		default:
			analysis.Unsupported = true
		}
	}
	result := destinationResolution{
		Destination: record.meta.id,
		Producers:   producers,
	}
	if len(supplied) > 1 || (len(supplied) == 0 && len(resolved) > 1) {
		conflicting := supplied
		if len(conflicting) == 0 {
			conflicting = resolved
		}
		ids := make([]valueCandidateRef, len(conflicting))
		metas := make([]valueCandidateMeta, len(conflicting))
		for i, plan := range conflicting {
			metas[i] = exclusiveValueCandidateMeta(state, record.meta.id, plan)
			ids[i] = metas[i].ID
		}
		for i, plan := range conflicting {
			meta := metas[i]
			meta.Disposition = ValueCandidateDispositionConflicting
			for j, id := range ids {
				if i != j {
					meta.ConflictsWith = append(meta.ConflictsWith, id)
				}
			}
			analysis.Candidates[meta.ID] = valueCandidateObservation{
				Meta:          meta,
				ProducerIndex: plan.producerIndex,
				Declaration:   plan.declaration,
			}
			result.Candidates = append(result.Candidates, meta.ID)
		}
		for _, plan := range resolved {
			if len(supplied) == 0 {
				break
			}
			meta := exclusiveValueCandidateMeta(state, record.meta.id, plan)
			meta.Disposition, meta.Suppression = ValueCandidateDispositionSuppressed, ValueCandidateSuppressionPrecedence
			meta.SuppressedBy = append([]valueCandidateRef(nil), ids...)
			analysis.Candidates[meta.ID] = valueCandidateObservation{
				Meta:          meta,
				ProducerIndex: plan.producerIndex,
				Declaration:   plan.declaration,
			}
			result.Candidates = append(result.Candidates, meta.ID)
		}
		result.State, result.Err = destinationStateFailed, errSameTierConflict
		valueIssueValue := valueIssue{
			Kind:        valueIssueResolutionConflict,
			Destination: record.meta.id,
			Err:         result.Err,
		}
		analysis.Issues = append(analysis.Issues, valueIssueValue)
		result.Producers = finalizeProducerOutcomes(record.meta.id, producers, plans, analysis.Candidates)
		analysis.Destinations[record.meta.id] = result
		return
	}

	var selected *valueCandidatePlan
	if len(supplied) == 1 {
		selected = &supplied[0]
	} else if len(resolved) == 1 {
		selected = &resolved[0]
	}
	if selected == nil {
		result.State = destinationStateAbsent
		if definition.Kind() == destinationKindFactRequired {
			result.State = destinationStateBlocked
		}
		result.Producers = finalizeProducerOutcomes(record.meta.id, producers, plans, analysis.Candidates)
		analysis.Destinations[record.meta.id] = result
		return
	}

	meta := exclusiveValueCandidateMeta(state, record.meta.id, *selected)
	value, err := selected.whole()
	observation := valueCandidateObservation{
		Meta:          meta,
		ProducerIndex: selected.producerIndex,
		Declaration:   selected.declaration,
	}
	switch {
	case errors.Is(err, errValueCandidateOmitted):
		meta.Disposition = ValueCandidateDispositionNoEffect
		result.State = destinationStateAbsent
		if definition.Kind() == destinationKindFactRequired {
			result.State = destinationStateBlocked
		}
	case err != nil:
		meta.Disposition = ValueCandidateDispositionRejected
		result.State, result.Err = destinationStateFailed, err
		valueIssueValue2 := valueIssue{
			Kind:        valueIssueProducerFailure,
			Destination: record.meta.id,
			Err:         err,
		}
		analysis.Issues = append(analysis.Issues, valueIssueValue2)
	default:
		meta.Disposition = ValueCandidateDispositionEffective
		observation.Value, observation.HasValue = snapshotMutableValue(value), true
		result.State, result.Value, result.Sensitive = destinationStateResolved, snapshotMutableValue(
			value,
		), record.sensitive ||
			selected.sensitive
		result.Contributors = []valueCandidateRef{meta.ID}
	}
	observation.Meta = meta
	analysis.Candidates[meta.ID] = observation
	result.Candidates = append(result.Candidates, meta.ID)
	if len(supplied) == 1 {
		for _, plan := range resolved {
			suppressed := exclusiveValueCandidateMeta(state, record.meta.id, plan)
			suppressed.Disposition, suppressed.Suppression = ValueCandidateDispositionSuppressed, ValueCandidateSuppressionPrecedence
			suppressed.SuppressedBy = []valueCandidateRef{meta.ID}
			analysis.Candidates[suppressed.ID] = valueCandidateObservation{
				Meta:          suppressed,
				ProducerIndex: plan.producerIndex,
				Declaration:   plan.declaration,
			}
			result.Candidates = append(result.Candidates, suppressed.ID)
		}
	}
	result.Producers = finalizeProducerOutcomes(record.meta.id, producers, plans, analysis.Candidates)
	analysis.Destinations[record.meta.id] = result
}

func exclusiveValueCandidateMeta(state *analysisState, destination ValueID, plan valueCandidatePlan) valueCandidateMeta {
	result := valueCandidateMeta{
		ID:           state.allocateValueCandidateRef(),
		Destination:  destination,
		ProducerKind: plan.producerKind,
		Tier:         plan.tier,
		Order:        plan.order,
		Shape:        contributionShapeWhole,
		Sensitive:    plan.sensitive,
		Origins:      append([]InputOriginID(nil), plan.origins...),
		Disposition:  ValueCandidateDispositionUnresolved,
	}

	return result
}

func projectValueCandidateMetas(plans []valueCandidatePlan, memos []*memoValue, metas []valueCandidateMeta, analysis *valueAnalysis, result *destinationResolution) {
	for _, meta := range metas {
		planIndex := findPlanIndex(plans, meta.ProducerKind, meta.Tier, meta.Order)
		if planIndex < 0 {
			continue
		}
		plan := plans[planIndex]
		observation := valueCandidateObservation{
			Meta:          meta,
			ProducerIndex: plan.producerIndex,
			Declaration:   plan.declaration,
			Input:         plan.input,
			HasInput:      plan.hasInput,
			InputSource:   plan.inputSource,
		}
		if planIndex < len(memos) && memos[planIndex] != nil && memos[planIndex].done {
			observation.Value, observation.HasValue = snapshotMutableValue(
				memos[planIndex].value,
			), memos[planIndex].err == nil
			if observation.HasValue && plan.tokenLanguage.Valid() && plan.tokenRepresentation != nil {
				tokens, available, err := plan.tokenRepresentation()
				if err != nil {
					analysis.Unsupported = true
				} else if available {
					observation.TokenRepresentation = tokens
					observation.HasTokenRepresentation = true
				}
			}
		}
		analysis.Candidates[meta.ID] = observation
		result.Candidates = append(result.Candidates, meta.ID)
		if meta.Disposition == ValueCandidateDispositionEffective {
			result.Contributors = append(result.Contributors, meta.ID)
		}
	}
}

func projectSequenceValueCandidateMetas(plans []valueCandidatePlan, memos []*memoSequence, metas []valueCandidateMeta, analysis *valueAnalysis, result *destinationResolution) {
	for _, meta := range metas {
		index := findPlanIndex(plans, meta.ProducerKind, meta.Tier, meta.Order)
		if index < 0 {
			continue
		}
		plan := plans[index]
		observation := valueCandidateObservation{
			Meta:          meta,
			ProducerIndex: plan.producerIndex,
			Declaration:   plan.declaration,
			Input:         plan.input,
			HasInput:      plan.hasInput,
			InputSource:   plan.inputSource,
		}
		for _, input := range plan.elementInputs {
			observation2 := valueCandidateComponentObservation{
				Input:     input.text,
				HasInput:  input.present,
				Origins:   append([]InputOriginID(nil), input.origins...),
				Sensitive: plan.sensitive,
			}
			observation.Elements = append(observation.Elements, observation2)
		}
		if index < len(memos) && memos[index] != nil && memos[index].done && memos[index].err == nil {
			for elementIndex, value := range memos[index].value {
				if elementIndex >= len(observation.Elements) {
					observation.Elements = append(observation.Elements, valueCandidateComponentObservation{
						Sensitive: plan.sensitive,
					})
				}
				observation.Elements[elementIndex].Value = snapshotMutableValue(value)
				observation.Elements[elementIndex].HasValue = true
			}
		}
		analysis.Candidates[meta.ID] = observation
		result.Candidates = append(result.Candidates, meta.ID)
		if meta.Disposition == ValueCandidateDispositionEffective {
			result.Contributors = append(result.Contributors, meta.ID)
		}
	}
}

func projectKeyedBatchValueCandidates(plans []valueCandidatePlan, candidates []keyedBatchValueCandidateResult, analysis *valueAnalysis, result *destinationResolution) {
	for _, candidate := range candidates {
		if candidate.BatchIndex < 0 || candidate.BatchIndex >= len(plans) {
			continue
		}
		plan := plans[candidate.BatchIndex]
		observation := valueCandidateObservation{
			Meta:          candidate.Meta,
			ProducerIndex: plan.producerIndex,
			Declaration:   plan.declaration,
			Input:         plan.input,
			HasInput:      plan.hasInput,
			InputSource:   plan.inputSource,
		}
		if candidate.EntryIndex >= 0 {
			entry := valueCandidateEntryObservation{}
			if candidate.EntryIndex < len(plan.keyedEntries) {
				raw := plan.keyedEntries[candidate.EntryIndex]
				entry.Key = componentObservation(raw.keyInput, candidate.Key, candidate.HasKey, plan.sensitive)
				entry.Value = componentObservation(raw.valueInput, candidate.Value, candidate.HasValue, plan.sensitive)
			} else {
				entry.Key = componentObservation(plan.keyInput, candidate.Key, candidate.HasKey, plan.sensitive)
				entry.Value = componentObservation(plan.valueInput, candidate.Value, candidate.HasValue, plan.sensitive)
			}
			observation.Entries = []valueCandidateEntryObservation{entry}
		} else {
			for entryIndex, typed := range candidate.Entries {
				var keyInput, valueInput valueCandidateComponentInput
				if entryIndex < len(plan.keyedEntries) {
					keyInput = plan.keyedEntries[entryIndex].keyInput
					valueInput = plan.keyedEntries[entryIndex].valueInput
				}
				observation.Entries = append(observation.Entries, valueCandidateEntryObservation{
					Key:   componentObservation(keyInput, typed.Key, true, plan.sensitive),
					Value: componentObservation(valueInput, typed.Value, true, plan.sensitive),
				})
			}
		}
		analysis.Candidates[candidate.Meta.ID] = observation
		result.Candidates = append(result.Candidates, candidate.Meta.ID)
		if candidate.Meta.Disposition == ValueCandidateDispositionEffective {
			result.Contributors = append(result.Contributors, candidate.Meta.ID)
		}
	}
}

func componentObservation(input valueCandidateComponentInput, value any, present, sensitive bool) valueCandidateComponentObservation {
	result := valueCandidateComponentObservation{
		Input:     input.text,
		HasInput:  input.present,
		Value:     snapshotMutableValue(value),
		HasValue:  present,
		Origins:   append([]InputOriginID(nil), input.origins...),
		Sensitive: sensitive,
	}

	return result
}

func sequenceComponentReferences(metas []valueCandidateMeta, candidates map[valueCandidateRef]valueCandidateObservation, order ResolutionTierOrder) []valueCandidateComponentReference {
	groups := make([][]valueCandidateMeta, 0)
	for _, meta := range metas {
		if meta.Disposition != ValueCandidateDispositionEffective {
			continue
		}
		if len(groups) == 0 || groups[len(groups)-1][0].Tier != meta.Tier {
			groups = append(groups, []valueCandidateMeta{meta})
		} else {
			groups[len(groups)-1] = append(groups[len(groups)-1], meta)
		}
	}
	if order == ResolutionTierLowerFirst {
		for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
			groups[i], groups[j] = groups[j], groups[i]
		}
	}
	var result []valueCandidateComponentReference
	for _, group := range groups {
		for _, meta := range group {
			observation := candidates[meta.ID]
			for index := range observation.Elements {
				result = append(result, valueCandidateComponentReference{
					Candidate: meta.ID,
					Index:     index,
				})
			}
		}
	}
	return result
}

func finalizeProducerOutcomes(destination ValueID, producers []producerObservation, plans []valueCandidatePlan, candidates map[valueCandidateRef]valueCandidateObservation) []producerObservation {
	result := append([]producerObservation(nil), producers...)
	for _, plan := range plans {
		if plan.producerIndex < 0 || plan.producerIndex >= len(result) {
			continue
		}
		producer := &result[plan.producerIndex]
		if plan.tracker == nil {
			for _, candidate := range candidates {
				if candidate.Meta.Destination == destination && candidate.ProducerIndex == plan.producerIndex {
					producer.Outcome = producerOutcomeContributed
					break
				}
			}
			continue
		}
		if producer.Kind == declaredProducerKindDefault {
			producer.Outcome = producerOutcomeContributed
			continue
		}
		if !plan.tracker.called {
			producer.Outcome = producerOutcomeSkipped
			continue
		}
		if len(plan.tracker.blockedBy) != 0 {
			producer.Outcome = producerOutcomeBlocked
			producer.BlockedBy = append([]ValueID(nil), plan.tracker.blockedBy...)
			continue
		}
		if plan.tracker.err != nil {
			producer.Outcome = producerOutcomeFailed
			producer.Err = plan.tracker.err
			continue
		}
		if !plan.tracker.present {
			producer.Outcome = producerOutcomeOmitted
			continue
		}
		producer.Outcome = producerOutcomeContributed
	}
	return result
}
