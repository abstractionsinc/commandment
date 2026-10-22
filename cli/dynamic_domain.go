package cli

import (
	"context"
	"errors"
	"maps"
	"math"
	"slices"
	"sync"
)

type dynamicDomainOutcomeStatus uint8

const (
	dynamicDomainOutcomeUnknown dynamicDomainOutcomeStatus = iota
	dynamicDomainOutcomeReady
	dynamicDomainOutcomeInvalid
	dynamicDomainOutcomeFailed
	dynamicDomainOutcomeBlocked
	dynamicDomainOutcomeCanceled
)

type dynamicDomainOutcome struct {
	index     int
	status    dynamicDomainOutcomeStatus
	err       error
	entries   []ValueDomainEntryID
	blockedBy []ValueID
}

type dynamicDomainSlotPhase uint8

const (
	dynamicDomainSlotUnstarted dynamicDomainSlotPhase = iota
	dynamicDomainSlotRunning
	dynamicDomainSlotSettled
)

type dynamicDomainSlot struct {
	mu sync.Mutex

	phase      dynamicDomainSlotPhase
	done       chan struct{}
	outcome    dynamicDomainOutcome
	panicked   bool
	panicValue any
}

type dynamicDomainState struct {
	mu sync.Mutex

	slots map[int]*dynamicDomainSlot
}

func (state *dynamicDomainState) slot(index int) *dynamicDomainSlot {
	if state == nil {
		panic("cli: nil dynamic-domain state")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.slots == nil {
		state.slots = make(map[int]*dynamicDomainSlot)
	}
	slot := state.slots[index]
	if slot == nil {
		slot = &dynamicDomainSlot{}
		state.slots[index] = slot
	}
	return slot
}

func (operation *invocation) materializeDynamicDomain(ctx context.Context, grammar grammarResult, index int) dynamicDomainOutcome {
	if operation == nil || operation.analysis == nil {
		panic("cli: invalid dynamic-domain demand")
	}
	return operation.materializeDynamicDomainAt(ctx, operation.analysis, grammar, index)
}

func (operation *invocation) materializeDynamicDomainAt(ctx context.Context, analysis *analysisState, grammar grammarResult, index int) dynamicDomainOutcome {
	if operation == nil || analysis == nil {
		panic("cli: invalid dynamic-domain demand")
	}
	release := analysis.work.enter()
	defer release()

	slot := analysis.domains.slot(index)

	slot.mu.Lock()
	leader := false
	switch slot.phase {
	case dynamicDomainSlotUnstarted:
		slot.phase = dynamicDomainSlotRunning
		slot.done = make(chan struct{})
		leader = true
	case dynamicDomainSlotRunning:
	case dynamicDomainSlotSettled:
	default:
		slot.mu.Unlock()
		panic("cli: invalid dynamic-domain slot phase")
	}
	done := slot.done
	slot.mu.Unlock()

	if leader {
		var outcome dynamicDomainOutcome
		var commit *dynamicDomainCommit
		var panicked bool
		var panicValue any
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					panicked = true
					panicValue = recovered
				}
			}()
			outcome, commit = operation.evaluateDynamicDomainAt(ctx, analysis, grammar, index)
		}()

		switch {
		case panicked:
			settleDynamicDomainSlot(slot, outcome, true, panicValue)
		case commit != nil && analysis == operation.analysis:
			operation.publishDynamicDomainCommit(slot, *commit)
		case commit != nil:
			operation.publishDynamicDomainCommitPrivate(analysis, slot, *commit)
		default:
			settleDynamicDomainSlot(slot, outcome, false, nil)
		}
	} else if done != nil {
		<-done
	}

	slot.mu.Lock()
	outcome := cloneDynamicDomainOutcome(slot.outcome)
	panicked := slot.panicked
	panicValue := slot.panicValue
	slot.mu.Unlock()
	if panicked {
		panic(panicValue)
	}
	return outcome
}

func settleDynamicDomainSlot(slot *dynamicDomainSlot, outcome dynamicDomainOutcome, panicked bool, panicValue any) {
	if slot == nil {
		panic("cli: nil dynamic-domain slot")
	}
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if slot.phase != dynamicDomainSlotRunning || slot.done == nil {
		panic("cli: dynamic-domain slot is not running")
	}
	slot.outcome = cloneDynamicDomainOutcome(outcome)
	slot.panicked = panicked
	slot.panicValue = panicValue
	slot.phase = dynamicDomainSlotSettled
	close(slot.done)
}

func cloneDynamicDomainOutcome(outcome dynamicDomainOutcome) dynamicDomainOutcome {
	cloned := outcome
	cloned.entries = append([]ValueDomainEntryID(nil), outcome.entries...)
	cloned.blockedBy = append([]ValueID(nil), outcome.blockedBy...)
	return cloned
}

func (operation *invocation) dynamicDomainValueLanguage(ctx context.Context) grammarValueLanguageDemandFunc {
	return func(demand grammarValueLanguageDemand) grammarValueLanguageMaterialization {
		index, ok := dynamicDomainIndexForDestination(operation.analysis.effectiveModel, demand.Destination)
		if !ok {
			return grammarValueLanguageMaterialization{}
		}
		outcome := operation.materializeDynamicDomain(ctx, demand.Grammar, index)
		if outcome.status == dynamicDomainOutcomeReady {
			materialization := grammarValueLanguageMaterialization{
				Ready: true,
			}
			return materialization
		}
		materialization := grammarValueLanguageMaterialization{
			Err: outcome.err,
		}
		return materialization
	}
}

func dynamicDomainIndexesForDestination(model *effectiveModel, destination ValueID) []int {
	if model == nil {
		panic("cli: nil effective model for dynamic-domain demand")
	}

	domains := model.ValueDomains()
	indexes := make([]int, 0, 1)
	for index, domain := range domains {
		for _, use := range domain.Uses() {
			if use.destination == destination {
				indexes = append(indexes, index)
				break
			}
		}
	}

	return indexes
}

func (operation *invocation) materializeDynamicDomainsForDestination(ctx context.Context, grammar grammarResult, destination ValueID) map[int]dynamicDomainOutcome {
	if operation == nil || operation.analysis == nil {
		panic("cli: invalid destination dynamic-domain demand")
	}

	model := operation.analysis.effectiveModel
	indexes := dynamicDomainIndexesForDestination(model, destination)
	var outcomes map[int]dynamicDomainOutcome
	for _, index := range indexes {
		domain, ok := model.ValueDomain(index)
		if !ok {
			panic("cli: demanded value domain cannot resolve")
		}
		if _, dynamic := domain.DynamicDefinition(); !dynamic {
			continue
		}
		if outcomes == nil {
			outcomes = make(map[int]dynamicDomainOutcome)
		}
		outcomes[index] = operation.materializeDynamicDomain(ctx, grammar, index)
		model = operation.analysis.effectiveModel
	}

	return outcomes
}

func (state *dynamicDomainState) outcomes() map[int]dynamicDomainOutcome {
	if state == nil {
		return nil
	}
	state.mu.Lock()
	slots := make(map[int]*dynamicDomainSlot, len(state.slots))
	maps.Copy(slots, state.slots)
	state.mu.Unlock()

	outcomes := make(map[int]dynamicDomainOutcome, len(slots))
	for index, slot := range slots {
		slot.mu.Lock()
		if slot.phase == dynamicDomainSlotSettled {
			outcomes[index] = cloneDynamicDomainOutcome(slot.outcome)
		}
		slot.mu.Unlock()
	}
	return outcomes
}

func (operation *invocation) evaluateDynamicDomainAt(ctx context.Context, analysis *analysisState, grammar grammarResult, index int) (dynamicDomainOutcome, *dynamicDomainCommit) {
	result := dynamicDomainOutcome{
		index:  index,
		status: dynamicDomainOutcomeBlocked,
	}
	effectiveModel := analysis.effectiveModel
	domain, ok := effectiveModel.ValueDomain(index)
	if !ok {
		panic("cli: demanded dynamic value domain cannot resolve")
	}
	model := effectiveModel
	definition, dynamic := domain.DynamicDefinition()
	if !dynamic {
		result.status = dynamicDomainOutcomeReady
		result.entries = domain.Entries()
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		result.status = dynamicDomainOutcomeCanceled
		result.err = err
		return result, nil
	}
	dependencyAnalysis := analyzeDependenciesAt(ctx, operation, analysis, grammar, definition.Dependencies())
	configuration, ok := operation.analysisConfigurationAt(analysis, definition.Owner(), dependencyAnalysis.Values.Destinations)
	if !ok {
		panic("cli: accepted dynamic-domain owner has no analysis configuration")
	}
	dependencies, ready := dependenciesFromValueAnalysis(operation.scope, effectiveModel, definition.Dependencies(), dependencyAnalysis.Values)
	if !ready {
		result.blockedBy = blockedDependencyIDs(definition.Dependencies(), dependencyAnalysis.Values)
		if len(result.blockedBy) == 0 {
			panic("cli: dynamic-domain dependency analysis is not ready without a blocking dependency")
		}
		return result, nil
	}
	callback, callbackOK := analysis.valueDomainCallback(index)
	if !callbackOK {
		panic("cli: accepted dynamic value domain has no callback")
	}
	evaluation := &dynamicDomainEvaluation{}
	if err := callback(ctx, configuration, dependencies, evaluation); err != nil {
		switch {
		case callbackReturnedCancellation(ctx, err):
			result.status = dynamicDomainOutcomeCanceled
		case errors.Is(err, errInvalidDynamicValueDomain):
			result.status = dynamicDomainOutcomeInvalid
		default:
			result.status = dynamicDomainOutcomeFailed
		}
		result.err = err
		return result, nil
	}
	if !validValueDomainState(evaluation.state) || evaluation.state.dynamic != nil {
		panic("cli: accepted dynamic value-domain callback returned invalid state without an error")
	}

	identities, acceptance, localDomainIndex, scopeOK := operation.prepareDynamicDomainPublication(analysis, index)
	if !scopeOK {
		panic("cli: accepted dynamic value domain cannot begin publication")
	}
	draft := acceptance.Draft()
	entries, err := prepareDomainEntries(draft, evaluation.state, model.Application())
	if err != nil {
		result.status = dynamicDomainOutcomeInvalid
		result.err = errInvalidDynamicValueDomain
		return result, nil
	}
	domainRef := draft.ValueDomainByIndex(localDomainIndex)
	if index < 0 || uint64(index) >= uint64(math.MaxUint32) {
		panic("cli: accepted dynamic value-domain index is out of range")
	}
	draft.ValueDomainEntries(domainRef, entries, declarationPosition(uint32(index)+1))
	language, ok := eraseEffectiveDomainValueLanguage(evaluation.state, model.Application().release)
	if !ok {
		result.status = dynamicDomainOutcomeInvalid
		result.err = errInvalidDynamicValueDomain
		return result, nil
	}
	for _, use := range domain.Uses() {
		record, ok := model.Destination(use.destination)
		if !ok {
			panic("cli: accepted dynamic value-domain destination cannot resolve")
		}
		definition, ok := record.Definition()
		if !ok {
			panic("cli: accepted dynamic value-domain destination has no definition")
		}
		effective, ok := materializedValueDomainDefinition(definition, use.kind, language)
		if !ok {
			panic("cli: accepted dynamic value-domain use cannot materialize")
		}
		draft.DestinationDefinition(draft.DestinationByID(record.meta.id), effective, record.meta.position)
	}

	if err := ctx.Err(); err != nil {
		result.status = dynamicDomainOutcomeCanceled
		result.err = err
		return result, nil
	}
	accepted, err := acceptance.Accept(acceptAcceptanceDecision(acceptanceBoundaryProviderBatch))
	if err != nil {
		result.status = dynamicDomainOutcomeInvalid
		result.err = &publicationOperationError{
			operation: "accept dynamic-domain publication",
			cause:     err,
		}
		return result, nil
	}
	publication := freezeDynamicDomainPublication(index, language, accepted.Declarations().DomainEntries())
	result.status = dynamicDomainOutcomeReady
	result.entries = make([]ValueDomainEntryID, len(entries))
	for entryIndex := range entries {
		result.entries[entryIndex] = entries[entryIndex].ID()
	}
	commit := &dynamicDomainCommit{
		analysis:    analysis,
		identities:  identities,
		publication: publication,
		outcome:     cloneDynamicDomainOutcome(result),
	}
	return result, commit
}

func dynamicDomainIndexForDestination(effectiveModel *effectiveModel, destination ValueID) (int, bool) {
	if !effectiveModel.Valid() {
		return -1, false
	}
	found := -1
	for _, index := range effectiveModel.ValueDomainIndexesForDestination(destination) {
		domain, ok := effectiveModel.ValueDomain(index)
		if !ok {
			continue
		}
		if _, dynamic := domain.DynamicDefinition(); !dynamic {
			continue
		}
		if found >= 0 && found != index {
			return -1, false
		}
		found = index
	}
	return found, found >= 0
}

func demandedDynamicDomainsForValueAnalysis(operation *invocation, grammar grammarResult) []int {
	if operation == nil {
		return nil
	}
	effectiveModel := operation.analysis.effectiveModel
	selected := make(map[CommandID]struct{}, len(grammar.Selection.Path))
	for _, command := range grammar.Selection.Path {
		selected[command] = struct{}{}
	}

	neededSet := make(map[ValueID]struct{})
	neededOrder := make([]ValueID, 0, len(grammar.Occurrences)+len(grammar.Arguments.Terms))
	addNeeded := func(destination ValueID) {
		if !destination.present() {
			return
		}
		if _, exists := neededSet[destination]; exists {
			return
		}
		neededSet[destination] = struct{}{}
		neededOrder = append(neededOrder, destination)
	}

	for _, occurrence := range grammar.Occurrences {
		addNeeded(occurrence.Destination)
	}
	for _, allocation := range grammar.Arguments.Terms {
		if len(allocation.Indices) == 0 {
			continue
		}
		term, ok := effectiveModel.ArgumentTerm(allocation.Term)
		if !ok {
			continue
		}
		if destination, ok := term.Destination(); ok {
			addNeeded(destination)
		}
	}
	if matches, err := operation.MatchEnvironmentProducers(); err == nil {
		for _, match := range matches {
			addNeeded(match.Destination)
		}
	}
	for _, command := range grammar.Selection.Path {
		for _, destination := range effectiveModel.DestinationsForCommand(command) {
			if len(destination.Producers()) != 0 {
				addNeeded(destination.meta.id)
			}
		}
	}

	demanded := make(map[int]struct{})
	result := make([]int, 0)
	for _, destination := range neededOrder {
		for _, index := range effectiveModel.ValueDomainIndexesForDestination(destination) {
			if _, exists := demanded[index]; exists {
				continue
			}
			domain, ok := effectiveModel.ValueDomain(index)
			if !ok {
				continue
			}
			definition, dynamic := domain.DynamicDefinition()
			if !dynamic {
				continue
			}
			if _, ok := selected[definition.Owner()]; !ok {
				continue
			}
			demanded[index] = struct{}{}
			result = append(result, index)
		}
	}

	// Preserve the previous whole-family scan's global value-domain index order
	// while making ordering work proportional only to the demanded result set.
	slices.Sort(result)
	return result
}

func dynamicDomainMaterializationDiagnosticCode(status dynamicDomainOutcomeStatus) (DiagnosticCode, bool) {
	switch status {
	case dynamicDomainOutcomeInvalid:
		return DiagnosticInvalidValueDomain, true
	case dynamicDomainOutcomeFailed:
		return DiagnosticValueDomainProvisionFailure, true
	case dynamicDomainOutcomeCanceled:
		return DiagnosticValueDomainProvisionCancellation, true
	default:
		return "", false
	}
}

func dynamicDomainMaterializationDiagnostics(results map[int]dynamicDomainOutcome) ([]Diagnostic, error) {
	indexes := make([]int, 0, len(results))
	for index := range results {
		indexes = append(indexes, index)
	}
	slices.Sort(indexes)
	diagnostics := make([]Diagnostic, 0, len(indexes))
	var cause error
	for _, index := range indexes {
		outcome := results[index]
		code, ok := dynamicDomainMaterializationDiagnosticCode(outcome.status)
		if !ok {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Code: code,
		})
		if cause == nil && outcome.err != nil {
			cause = outcome.err
		}
	}
	return diagnostics, cause
}

func dynamicDomainDiagnosticFailure(operation *invocation, results map[int]dynamicDomainOutcome, cause error) error {
	indexes := make([]int, 0, len(results))
	for index := range results {
		indexes = append(indexes, index)
	}
	slices.Sort(indexes)
	payload := operationDiagnosticPayload(operation, false)
	for _, index := range indexes {
		outcome := results[index]
		code, ok := dynamicDomainMaterializationDiagnosticCode(outcome.status)
		if !ok {
			continue
		}
		diagnostic := diagnostic{
			Code: string(code),
		}
		if domain, ok := operation.analysis.effectiveModel.ValueDomain(index); ok {
			if definition, dynamic := domain.DynamicDefinition(); dynamic {
				diagnostic.Command = definition.Owner()
			}
		}
		payload.Diagnostics = append(payload.Diagnostics, diagnostic)
	}
	return retainDiagnosticFailure(payload, cause)
}

func applyDynamicDomainMaterialization(result *Inspection, operation *invocation, model *effectiveModel, locale Locale, results map[int]dynamicDomainOutcome) {
	domains := model.ValueDomains()
	for index, domain := range domains {
		definition, dynamic := domain.DynamicDefinition()
		if !dynamic || index >= len(result.ValueDomains) {
			continue
		}
		public := result.ValueDomains[index]
		public.InvocationScoped = true
		public.Command = CommandID(definition.Owner().uint32())
		public.Dependencies = make([]ValueID, len(definition.Dependencies()))
		for dependencyIndex, dependency := range definition.Dependencies() {
			public.Dependencies[dependencyIndex] = ValueID(dependency.Destination().uint32())
		}
		outcome, demanded := results[index]
		if !demanded {
			public.Outcome = ValueDomainOutcomeNotNeeded
		} else {
			switch outcome.status {
			case dynamicDomainOutcomeReady:
				public.Outcome = ValueDomainOutcomeReady
			case dynamicDomainOutcomeInvalid:
				public.Outcome = ValueDomainOutcomeInvalid
			case dynamicDomainOutcomeFailed:
				public.Outcome = ValueDomainOutcomeFailed
			case dynamicDomainOutcomeBlocked:
				public.Outcome = ValueDomainOutcomeBlocked
			case dynamicDomainOutcomeCanceled:
				public.Outcome = ValueDomainOutcomeCanceled
			default:
				public.Outcome = ValueDomainOutcomeUnknown
			}
		}

		public.Entries = public.Entries[:0]
		for _, entryID := range domain.Entries() {
			entry, ok := model.DomainEntry(entryID)
			if !ok {
				panic("cli: effective dynamic domain entry is absent")
			}
			projected, available := projectInspectionDynamicDomainEntry(result, operation, model, definition.Owner(), locale, entry)
			result.ValueDomainEntries[projected.ID] = projected
			if available {
				public.Entries = append(public.Entries, projected.ID)
			}
		}
		result.ValueDomains[index] = public
	}
}
