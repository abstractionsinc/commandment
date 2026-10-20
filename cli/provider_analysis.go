package cli

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
)

var errProviderBatchRejected = errors.New("cli: provider batch rejected")

type providerContextError struct {
	cause error
}

func (err *providerContextError) Error() string {
	return "cli: provider context: " + err.cause.Error()
}

func (err *providerContextError) Unwrap() error { return err.cause }

type providerOutcomeStatus uint8

const (
	providerOutcomeUnknown providerOutcomeStatus = iota
	providerOutcomeReady
	providerOutcomeFailed
	providerOutcomeCanceled
	providerOutcomeBuilderMisuse
	providerOutcomeRejected
	providerOutcomeBlocked
)

type providerOutcome struct {
	provider  CommandProviderID
	status    providerOutcomeStatus
	err       error
	batch     providerBatch
	blockedBy []ValueID
}

type providerNamespaceOutcome struct {
	// changed records whether the accepted publication for this namespace added
	// at least one provider-owned command entry to operation-effective grammar.
	changed  bool
	complete bool
	results  []providerOutcome
}

type providerSlotPhase uint8

const (
	providerSlotUnstarted providerSlotPhase = iota
	providerSlotRunning
	providerSlotSettled
)

type providerNamespaceSlot struct {
	mu sync.Mutex

	phase      providerSlotPhase
	done       chan struct{}
	outcome    providerNamespaceOutcome
	panicked   bool
	panicValue any
}

type providerState struct {
	mu sync.Mutex

	namespaces map[CommandID]*providerNamespaceSlot
}

func (state *providerState) namespace(owner CommandID) *providerNamespaceSlot {
	if state == nil {
		panic("cli: nil provider state")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.namespaces == nil {
		state.namespaces = make(map[CommandID]*providerNamespaceSlot)
	}
	slot := state.namespaces[owner]
	if slot == nil {
		slot = &providerNamespaceSlot{}
		state.namespaces[owner] = slot
	}
	return slot
}

func (operation *invocation) materializeProviderNamespaceAt(ctx context.Context, analysis *analysisState, demand grammarNamespaceDemand) providerNamespaceOutcome {
	if operation == nil || analysis == nil || !operation.signalDescription.Valid() {
		panic("cli: invalid provider demand")
	}
	release := analysis.work.enter()
	defer release()

	slot := analysis.providers.namespace(demand.Command)

	slot.mu.Lock()
	leader := false
	switch slot.phase {
	case providerSlotUnstarted:
		slot.phase = providerSlotRunning
		slot.done = make(chan struct{})
		leader = true
	case providerSlotRunning:
	case providerSlotSettled:
	default:
		slot.mu.Unlock()
		panic("cli: invalid provider slot phase")
	}
	done := slot.done
	slot.mu.Unlock()

	if leader {
		var outcome providerNamespaceOutcome
		var commit *providerCommit
		var panicked bool
		var panicValue any
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					panicked = true
					panicValue = recovered
				}
			}()
			outcome, commit = operation.evaluateProviderNamespaceAt(ctx, analysis, demand.Command, demand.Grammar)
		}()

		switch {
		case panicked:
			settleProviderSlot(slot, outcome, true, panicValue)
		case commit != nil && analysis == operation.analysis:
			operation.publishProviderCommit(slot, *commit)
		case commit != nil:
			operation.publishProviderCommitPrivate(analysis, slot, *commit)
		default:
			settleProviderSlot(slot, outcome, false, nil)
		}
	} else if done != nil {
		<-done
	}

	slot.mu.Lock()
	outcome := cloneProviderNamespaceOutcome(slot.outcome)
	panicked := slot.panicked
	panicValue := slot.panicValue
	slot.mu.Unlock()
	if panicked {
		panic(panicValue)
	}
	return outcome
}

func settleProviderSlot(slot *providerNamespaceSlot, outcome providerNamespaceOutcome, panicked bool, panicValue any) {
	if slot == nil {
		panic("cli: nil provider slot")
	}
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if slot.phase != providerSlotRunning || slot.done == nil {
		panic("cli: provider slot is not running")
	}
	slot.outcome = cloneProviderNamespaceOutcome(outcome)
	slot.panicked = panicked
	slot.panicValue = panicValue
	slot.phase = providerSlotSettled
	close(slot.done)
}

func cloneProviderNamespaceOutcome(result providerNamespaceOutcome) providerNamespaceOutcome {
	cloned := result
	cloned.results = make([]providerOutcome, len(result.results))
	for index := range result.results {
		cloned.results[index] = result.results[index]
		cloned.results[index].blockedBy = append([]ValueID(nil), result.results[index].blockedBy...)
	}
	return cloned
}

func (result providerNamespaceOutcome) matchesProviderSet(providers []providerRecord) bool {
	if len(result.results) != len(providers) {
		return false
	}
	for _, provider := range providers {
		matched := false
		for _, settled := range result.results {
			if settled.provider == provider.meta.id {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func acceptedProviderEntries(publications []providerPublication) map[CommandProviderID][]CommandEntryID {
	result := make(map[CommandProviderID][]CommandEntryID)
	for _, publication := range publications {
		for provider, entries := range publication.ProviderEntries() {
			result[provider] = append(result[provider], entries...)
		}
	}
	return result
}

func (state *providerState) outcomes() map[CommandID]providerNamespaceOutcome {
	if state == nil {
		return nil
	}
	state.mu.Lock()
	slots := make(map[CommandID]*providerNamespaceSlot, len(state.namespaces))
	maps.Copy(slots, state.namespaces)
	state.mu.Unlock()

	outcomes := make(map[CommandID]providerNamespaceOutcome, len(slots))
	for owner, slot := range slots {
		slot.mu.Lock()
		if slot.phase == providerSlotSettled {
			outcomes[owner] = cloneProviderNamespaceOutcome(slot.outcome)
		}
		slot.mu.Unlock()
	}
	return outcomes
}

func (operation *invocation) evaluateProviderNamespaceAt(ctx context.Context, analysis *analysisState, owner CommandID, grammar grammarResult) (providerNamespaceOutcome, *providerCommit) {
	model := analysis.effectiveModel
	providers := effectiveProviderRecords(
		model.ProvidersForParent(owner),
		owner,
		grammar.SelectedProfiles,
	)
	if len(providers) == 0 {
		return providerNamespaceOutcome{
			complete: true,
		}, nil
	}

	results := make([]providerOutcome, len(providers))
	allDependencies := providerDependencyUnion(providers)
	dependencyAnalysis := dependencyAnalysis{
		Ready: true,
		Values: valueAnalysis{
			Destinations: make(map[ValueID]destinationResolution),
			Candidates:   make(map[valueCandidateRef]valueCandidateObservation),
		},
	}
	if len(allDependencies) != 0 && ctx.Err() == nil {
		dependencyAnalysis = analyzeDependenciesAt(ctx, operation, analysis, grammar, allDependencies)
	}
	panics := make([]any, len(providers))
	var evaluations sync.WaitGroup
	for index, record := range providers {
		results[index] = providerOutcome{
			provider: record.meta.id,
			status:   providerOutcomeBlocked,
		}
		definition := record.definition
		if err := ctx.Err(); err != nil {
			results[index].status = providerOutcomeCanceled
			results[index].err = err
			continue
		}
		configuration, configurationReady := operation.analysisConfigurationAt(analysis, owner, dependencyAnalysis.Values.Destinations)
		if !configurationReady {
			panic("cli: accepted provider command has no analysis configuration")
		}
		dependencies, ready := dependenciesFromValueAnalysis(operation.scope, model, definition.Dependencies(), dependencyAnalysis.Values)
		if !ready {
			results[index].blockedBy = blockedDependencyIDs(definition.Dependencies(), dependencyAnalysis.Values)
			if len(results[index].blockedBy) == 0 {
				panic("cli: provider dependency analysis is not ready without a blocking dependency")
			}
			continue
		}
		callback, callbackOK := analysis.commandProviderCallback(record.meta.id)
		if !callbackOK {
			panic("cli: accepted command provider has no callback")
		}

		// Provider callbacks are independent evaluations over the same settled
		// operation-local configuration/dependency snapshot. Execute eligible
		// callbacks concurrently, but retain each result in provider declaration
		// order so scheduling can never become contribution or diagnostic order.
		evaluations.Add(1)
		go func(index int, record providerRecord, callback commandProviderFunc, configuration any, dependencies Dependencies) {
			defer evaluations.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					panics[index] = recovered
				}
			}()
			call := providerCall{
				provider:     record.meta.id,
				parent:       record.parent,
				scope:        operation.scope,
				release:      model.Application().release,
				config:       configuration,
				dependencies: dependencies,
			}
			batch, err := callback(ctx, call)
			if err != nil {
				switch {
				case callbackReturnedCancellation(ctx, err):
					results[index].status = providerOutcomeCanceled
				case errors.Is(err, errBuilderMisuse):
					results[index].status = providerOutcomeBuilderMisuse
				case errors.Is(err, errInvalidProviderBatch):
					results[index].status = providerOutcomeRejected
				default:
					results[index].status = providerOutcomeFailed
				}
				results[index].err = err
				return
			}
			if err := validateProviderSignalSourcePolicies(
				batch.declarations,
				operation.signalDescription,
			); err != nil {
				results[index].status = providerOutcomeRejected
				results[index].err = err
				return
			}
			results[index].status = providerOutcomeReady
			results[index].batch = batch
		}(
			index,
			record,
			callback,
			configuration,
			dependencies,
		)
	}
	evaluations.Wait()
	for _, recovered := range panics {
		if recovered != nil {
			// Callback completion order is not semantic order. Preserve the ordinary
			// provider panic contract by rethrowing the first panicking provider in
			// declaration order before any peer settlement or publication occurs.
			panic(recovered)
		}
	}

	settleProviderLexicalBatches(operation, analysis, owner, providers, results, grammar.SelectedProfiles)
	active := make([]int, 0, len(results))
	for index := range results {
		if results[index].status == providerOutcomeReady {
			active = append(active, index)
		}
	}
	changed := false
	var commit *providerCommit
	if len(active) != 0 {
		var err error
		changed, commit, err = operation.prepareProviderProgramsAt(
			ctx,
			analysis,
			model,
			owner,
			providers,
			results,
			active,
			grammar.SelectedProfiles,
		)
		if err != nil {
			changed = false
			commit = nil
			for _, index := range active {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					results[index].status = providerOutcomeCanceled
				} else {
					results[index].status = providerOutcomeRejected
				}
				results[index].err = err
			}
		}
	}

	complete := true
	for _, result := range results {
		if result.status != providerOutcomeReady {
			complete = false
			break
		}
	}
	outcome := providerNamespaceOutcome{
		changed:  changed,
		complete: complete,
		results:  results,
	}
	if commit != nil {
		commit.outcome = cloneProviderNamespaceOutcome(outcome)
	}

	return outcome, commit
}

func providerDependencyUnion(providers []providerRecord) []dependencyRequirement {
	var result []dependencyRequirement
	indexByDestination := make(map[ValueID]int)
	for _, provider := range providers {
		for _, dependency := range provider.definition.Dependencies() {
			if index, ok := indexByDestination[dependency.Destination()]; ok {
				if result[index].Optional() && !dependency.Optional() {
					result[index] = acceptDependencyRequirement(dependency.Destination(), false)
				}
				continue
			}
			indexByDestination[dependency.Destination()] = len(result)
			result = append(result, dependency)
		}
	}
	return result
}

func effectiveProviderRecords(providers []providerRecord, owner CommandID, selected map[CommandID]GrammarProfileID) []providerRecord {
	result := make([]providerRecord, 0, len(providers))
	for _, record := range providers {
		if !record.meta.profile.present() || selected[owner] == record.meta.profile {
			result = append(result, record)
		}
	}
	return result
}

func providerRecordsForOwner(model *effectiveModel, owner CommandID) []providerRecord {
	result := model.ProvidersForParent(owner)
	slices.SortStableFunc(result, func(left, right providerRecord) int { return cmp.Compare(left.meta.position, right.meta.position) })
	return result
}

func (operation *invocation) prepareProviderProgramsAt(ctx context.Context, analysis *analysisState, model *effectiveModel, ownerID CommandID, providers []providerRecord, results []providerOutcome, active []int, selected map[CommandID]GrammarProfileID) (bool, *providerCommit, error) {
	for len(active) != 0 {
		identities, acceptance, ok := operation.prepareProviderPublication(analysis, ownerID, nil)
		if !ok {
			panic("cli: accepted provider owner cannot begin publication")
		}
		draft := acceptance.Draft()
		owner := draft.CommandByID(ownerID)
		ownerDialect := providerOwnerDialectAt(operation, analysis, ownerID, selected)

		coordinatesByProvider := make(map[CommandProviderID]providerPublicationCoordinates, len(active))
		stagingRejected := false
		for _, index := range active {
			provider := providers[index]
			batch := results[index].batch
			state := prepareProviderPublicationState(batch.declarations, owner, provider.meta.position)
			collectProviderSpellingConfigs(&batch.declarations, state)
			if err := stageProviderDeclarations(
				&batch.declarations,
				model.Application(),
				ownerDialect,
				draft,
				state,
			); err != nil {
				if !errors.Is(err, errProviderBatchRejected) {
					panic("cli: validated provider batch cannot stage")
				}
				results[index].status = providerOutcomeRejected
				results[index].err = err
				stagingRejected = true
				break
			}
			coordinatesByProvider[provider.meta.id] = providerCoordinatesFromState(state, batch.declarations)
		}
		if stagingRejected {
			active = active[:0]
			for index := range results {
				if results[index].status == providerOutcomeReady {
					active = append(active, index)
				}
			}
			continue
		}
		decision := acceptAcceptanceDecision(acceptanceBoundaryProviderBatch, nil)
		preview, err := acceptance.Preview(decision)
		if err != nil {
			return false, nil, &publicationOperationError{
				operation: "preview provider publication",
				cause:     err,
			}
		}

		// Target settlement is provider-batch local even though targets may name
		// canonical commands contributed by peers. Reject only the batch whose
		// target/default relationship is invalid, then rebuild the canonical
		// preview so dependent peer targets see the surviving namespace.
		rejected := false
		for _, index := range active {
			provider := providers[index]
			batch := results[index].batch
			coordinates := coordinatesByProvider[provider.meta.id]
			for _, raw := range batch.contributions {
				var target batchCommandTarget
				switch contribution := raw.(type) {
				case commandContribution:
					continue
				case aliasContribution:
					target = contribution.target
				case shortcutContribution:
					target = contribution.target
					if _, ok := providerShortcutDefinition(preview, coordinates, contribution); !ok {
						results[index].status = providerOutcomeRejected
						results[index].err = errProviderBatchRejected
						rejected = true
						break
					}
				default:
					panic("cli: validated provider contribution has invalid kind")
				}
				if rejected {
					break
				}
				if _, ok := resolveProviderContributionTarget(preview, ownerID, coordinates, target); !ok {
					results[index].status = providerOutcomeRejected
					results[index].err = errProviderBatchRejected
					rejected = true
					break
				}
			}
		}
		if rejected {
			active = active[:0]
			for index := range results {
				if results[index].status == providerOutcomeReady {
					active = append(active, index)
				}
			}
			continue
		}

		entryByContribution := make(map[providerContributionRef]commandEntryRef)
		for _, index := range active {
			provider := providers[index]
			batch := results[index].batch
			coordinates := coordinatesByProvider[provider.meta.id]
			for contributionIndex, raw := range batch.contributions {
				var target batchCommandTarget
				switch contribution := raw.(type) {
				case commandContribution:
					continue
				case aliasContribution:
					target = contribution.target
				case shortcutContribution:
					target = contribution.target
				default:
					panic("cli: validated provider contribution has invalid kind")
				}
				targetID, ok := resolveProviderContributionTarget(preview, ownerID, coordinates, target)
				if !ok {
					panic("cli: settled provider contribution target cannot resolve")
				}
				targetRef := draft.CommandByID(targetID)
				position := provider.meta.position
				entry := draft.CommandEntry(owner, targetRef, position)
				switch contribution := raw.(type) {
				case aliasContribution:
					draft.CommandEntryGrammar(entry, CommandEntryAlias, contribution.alias.Name, true)
					draft.CommandEntryPresentation(
						entry,
						semanticPresentation(contribution.alias.Description, nil, contribution.alias.Hidden),
						position,
					)
					draft.CommandEntryLifecycle(entry, contribution.alias.Release, position)
				case shortcutContribution:
					draft.CommandEntryGrammar(entry, CommandEntryShortcut, contribution.shortcut.Name, true)
					draft.CommandEntryPresentation(
						entry,
						semanticPresentation(
							contribution.shortcut.Description,
							contribution.shortcut.Examples,
							contribution.shortcut.Hidden,
						),
						position,
					)
					draft.CommandEntryLifecycle(entry, contribution.shortcut.Release, position)
					definition, ok := providerShortcutDefinition(preview, coordinates, contribution)
					if !ok {
						panic("cli: settled provider shortcut cannot be rebuilt")
					}
					draft.CommandEntryShortcut(entry, definition, position)
				}
				entryByContribution[providerContributionRef{
					provider: provider.meta.id,
					index:    contributionIndex,
				}] = entry
			}
		}

		for _, index := range active {
			provider := providers[index]
			for contributionIndex, raw := range results[index].batch.contributions {
				contribution, ok := raw.(shortcutContribution)
				if !ok || !contribution.replacement.valid() {
					continue
				}
				entry, entryOK := entryByContribution[providerContributionRef{
					provider: provider.meta.id,
					index:    contributionIndex,
				}]
				replacement, replacementOK := entryByContribution[providerContributionRef{
					provider: provider.meta.id,
					index:    contribution.replacement.index(),
				}]
				if !entryOK || !replacementOK {
					panic("cli: validated provider shortcut replacement cannot resolve")
				}
				draft.CommandEntryReplacement(entry, replacement, provider.meta.position)
			}
		}

		commandLocalization, entryLocalization, providerEntries := providerPublicationMetadata(
			active,
			providers,
			results,
			coordinatesByProvider,
			entryByContribution,
		)
		callbacks := declarationCallbacks{}
		acceptances := make([]destinationAcceptance, 0)
		for _, index := range active {
			provider := providers[index]
			batch := results[index].batch
			coordinates := coordinatesByProvider[provider.meta.id]
			providerCallbacks := providerDeclarationCallbacks(batch, coordinates, preview)
			callbacks.merge(providerCallbacks)
			providerAcceptances := prepareProviderDestinationAcceptances(batch, coordinates, preview)
			acceptances = append(acceptances, providerAcceptances...)
		}
		if err := ctx.Err(); err != nil {
			return false, nil, &providerContextError{cause: err}
		}
		accepted, err := acceptance.Accept(decision)
		if err != nil {
			return false, nil, &publicationOperationError{
				operation: "accept provider publication",
				cause:     err,
			}
		}
		providerPublication := freezeProviderPublication(
			accepted.Declarations(),
			accepted.Grammar(),
			commandLocalization,
			entryLocalization,
			providerEntries,
		)
		commit := &providerCommit{
			analysis:     analysis,
			identities:   identities,
			publication:  providerPublication,
			callbacks:    callbacks,
			destinations: acceptances,
		}
		for _, entries := range providerEntries {
			if len(entries) != 0 {
				return true, commit, nil
			}
		}
		return false, commit, nil
	}
	return false, nil, nil
}

func providerPublicationMetadata(active []int, providers []providerRecord, results []providerOutcome, coordinatesByProvider map[CommandProviderID]providerPublicationCoordinates, entryByContribution map[providerContributionRef]commandEntryRef) (map[CommandID]localizationSnapshot, map[CommandEntryID]localizationSnapshot, map[CommandProviderID][]CommandEntryID) {
	commandLocalization := make(map[CommandID]localizationSnapshot)
	entryLocalization := make(map[CommandEntryID]localizationSnapshot)
	providerEntries := make(map[CommandProviderID][]CommandEntryID, len(active))
	for _, index := range active {
		provider := providers[index]
		batch := results[index].batch
		coordinates := coordinatesByProvider[provider.meta.id]
		entries := make([]CommandEntryID, 0)
		for recordIndex := range len(batch.declarations.order) {
			entries = append(entries, coordinates.commandEntries[recordIndex]...)
		}
		for contributionIndex := range batch.contributions {
			if entry, ok := entryByContribution[providerContributionRef{
				provider: provider.meta.id,
				index:    contributionIndex,
			}]; ok {
				entries = append(entries, entry.ID())
			}
		}
		providerEntries[provider.meta.id] = append([]CommandEntryID(nil), entries...)
		if batch.localization == nil {
			continue
		}
		// Localization is validated after the complete declaration/contribution
		// candidate has been staged but before the no-fail effective-state commit.
		// Invalid localization can therefore consume neither visible declarations
		// nor the operation identity frontier.
		if !batch.localization.Enabled() {
			panic("cli: validated provider localization cannot be published")
		}
		localization := *batch.localization
		for recordIndex := range len(batch.declarations.order) {
			if command, ok := coordinates.command(recordIndex); ok {
				commandLocalization[command] = localization
			}
		}
		for _, entry := range entries {
			entryLocalization[entry] = localization
		}
	}
	return commandLocalization, entryLocalization, providerEntries
}

type providerContributionRef struct {
	provider CommandProviderID
	index    int
}

func settleProviderLexicalBatches(operation *invocation, analysis *analysisState, owner CommandID, providers []providerRecord, results []providerOutcome, selected map[CommandID]GrammarProfileID) {
	grammar, ok := grammarOperationSource{
		operation: operation,
		analysis:  analysis,
	}.scope(selected)
	if !ok {
		panic("cli: provider settlement has no effective grammar")
	}
	dialect := effectiveProviderDialect(grammar, owner, selected)
	mode := dialect.CommandNormalization()
	if !mode.valid() {
		panic("cli: provider settlement owner has invalid command normalization")
	}
	static := make([]string, 0)
	for _, entry := range grammar.CommandEntries(owner) {
		if !entry.profile.present() || selected[owner] == entry.profile {
			static = append(static, entry.name)
		}
	}
	for _, action := range grammar.ActionCommands(owner) {
		static = append(static, action.name)
	}
	var lexical []providerLexicalEntry[CommandProviderID]
	for index, provider := range providers {
		if results[index].status != providerOutcomeReady {
			continue
		}
		for _, contribution := range results[index].batch.contributions {
			name := providerContributionName(results[index].batch, contribution)
			lexical = append(lexical, providerLexicalEntry[CommandProviderID]{
				Provider: provider.meta.id,
				Name:     name,
			})
		}
	}
	dispositions := settleProviderLexicalNamespace(mode, static, lexical)
	for index, provider := range providers {
		if results[index].status != providerOutcomeReady {
			continue
		}
		if disposition, exists := dispositions[provider.meta.id]; exists &&
			disposition != providerLexicalAccepted {
			results[index].status = providerOutcomeRejected
			results[index].err = errProviderBatchRejected
		}
	}
}

func providerContributionName(batch providerBatch, contribution batchContribution) string {
	var name string
	switch contribution := contribution.(type) {
	case commandContribution:
		name = batch.declarations.commands[contribution.command.sliceIndex()].name
	case aliasContribution:
		name = contribution.alias.Name
	case shortcutContribution:
		name = contribution.shortcut.Name
	default:
		panic("cli: validated provider contribution has invalid kind")
	}
	if name == "" {
		panic("cli: validated provider contribution has empty name")
	}
	return name
}

func effectiveProviderDialect(grammar grammarView, owner CommandID, selected map[CommandID]GrammarProfileID) dialect {
	if profile := selected[owner]; profile.present() {
		dialect, ok := grammar.ProfileDialect(profile)
		if !ok {
			panic("cli: provider owner profile has no effective dialect")
		}
		return dialect
	}
	dialect, ok := grammar.Dialect(owner)
	if !ok {
		panic("cli: provider owner has no effective dialect")
	}
	return dialect
}

func providerOwnerDialectAt(operation *invocation, analysis *analysisState, owner CommandID, selected map[CommandID]GrammarProfileID) dialectInput {
	grammar, ok := grammarOperationSource{
		operation: operation,
		analysis:  analysis,
	}.scope(selected)
	if !ok {
		panic("cli: provider publication has no effective grammar")
	}
	dialect := effectiveProviderDialect(grammar, owner, selected)
	input := dialectInput{
		CommandMatching:      dialect.CommandMatching(),
		CommandNormalization: dialect.CommandNormalization(),
		FlagMatching:         dialect.FlagMatching(),
		FlagNormalization:    dialect.FlagNormalization(),
		FlagOrder:            dialect.FlagOrder(),
		ShortClustering:      dialect.ShortClustering(),
		DashlessFlags:        dialect.DashlessFlags(),
		DefaultFlagPlacement: dialect.DefaultFlagPlacement(),
		EndOfOptions:         dialect.EndOfOptions(),
		ResponseFiles:        dialect.ResponseFiles(),
	}
	if !validSemanticDialectInput(input) {
		panic("cli: provider publication owner has invalid dialect")
	}
	return input
}

func providerCoordinatesFromState(state *providerPublicationState, declarations batchDeclarations) providerPublicationCoordinates {
	coordinates := providerPublicationCoordinates{
		commands:          make([]CommandID, len(state.slots)),
		commandEntries:    make([][]CommandEntryID, len(state.slots)),
		batchCommands:     make(map[batchCommandRef]CommandID),
		batchDestinations: make(map[batchDestinationRef]ValueID),
		batchFlags:        make(map[batchFlagRef]FlagID),
		batchArguments:    make(map[batchArgumentTermRef]ArgumentTermID),
		batchProviders:    make(map[batchProviderRef]CommandProviderID),
	}
	for index, slot := range state.slots {
		if slot.command.ID().present() {
			coordinates.commands[index] = slot.command.ID()
		}
		if len(slot.commandEntries) != 0 {
			entries := make([]CommandEntryID, 0, len(slot.commandEntries))
			for _, entry := range slot.commandEntries {
				if entry.ID().present() {
					entries = append(entries, entry.ID())
				}
			}
			coordinates.commandEntries[index] = entries
		}
	}
	for staged, accepted := range state.destinations {
		if accepted.ID().present() {
			coordinates.batchDestinations[staged] = accepted.ID()
		}
	}
	for recordIndex, entry := range declarations.order {
		slot := state.mustSlot(recordIndex)
		switch entry.kind {
		case declarationRecordCommand:
			if slot.command.ID().present() {
				coordinates.batchCommands[batchRefAt[batchCommand](entry.familyIndex)] = slot.command.ID()
			}
		case declarationRecordFlag:
			if slot.flag.ID().present() {
				coordinates.batchFlags[batchRefAt[batchFlag](entry.familyIndex)] = slot.flag.ID()
			}
		case declarationRecordArgumentTerm:
			if slot.argumentTerm.ID().present() {
				coordinates.batchArguments[batchRefAt[batchArgumentTerm](entry.familyIndex)] = slot.argumentTerm.ID()
			}
		case declarationRecordProvider:
			if slot.provider.ID().present() {
				coordinates.batchProviders[batchRefAt[batchProvider](entry.familyIndex)] = slot.provider.ID()
			}
		default:
		}
	}
	return coordinates
}

func providerDeclarationCallbacks(batch providerBatch, coordinates providerPublicationCoordinates, preview publicationPreview) declarationCallbacks {
	if len(batch.callbacks.valueDomains) != len(batch.declarations.valueDomains) {
		panic("cli: validated provider value-domain callback custody is misaligned")
	}
	expectedProducerCallbacks := 0
	for ref, id := range coordinates.batchDestinations {
		destination, ok := preview.Destination(id)
		if !ok || destination.meta.id != id {
			panic("cli: published provider destination cannot be resolved for callback custody")
		}
		for index, producer := range destination.Producers() {
			key := batchValueProducerKey{
				destination: ref,
				index:       index,
			}
			callback, exists := batch.callbacks.valueProducers[key]
			if producer.kind == declaredProducerKindDefault {
				if exists {
					panic("cli: validated provider static producer has callback custody")
				}
				continue
			}
			expectedProducerCallbacks++
			if !exists || callback == nil {
				panic("cli: validated provider value-producer callback custody is incomplete")
			}
		}
	}
	if len(batch.callbacks.valueProducers) != expectedProducerCallbacks {
		panic("cli: validated provider value-producer callback custody is misaligned")
	}

	callbacks := declarationCallbacks{
		valueProducers: make(map[valueProducerKey]valueProducer, len(batch.callbacks.valueProducers)),
		completions: completionCallbacks{
			flags:     make(map[FlagID]completionCallback, len(batch.callbacks.completions.flags)),
			arguments: make(map[ArgumentTermID]completionCallback, len(batch.callbacks.completions.arguments)),
		},
		valueDomains:     append([]valueDomainProvider(nil), batch.callbacks.valueDomains...),
		commandProviders: make(map[CommandProviderID]commandProviderFunc, len(batch.callbacks.commandProviders)),
	}
	for key, callback := range batch.callbacks.valueProducers {
		destination := coordinates.mustBatchDestination(key.destination)
		if callback == nil || key.index < 0 {
			panic("cli: validated provider value-producer callback cannot be published")
		}
		callbacks.valueProducers[valueProducerKey{
			destination: destination,
			index:       key.index,
		}] = callback
	}
	for ref, callback := range batch.callbacks.completions.flags {
		id := coordinates.mustBatchFlag(ref)
		if !callback.present() {
			panic("cli: validated provider completion callback cannot be published")
		}
		callbacks.completions.flags[id] = callback
	}
	for ref, callback := range batch.callbacks.completions.arguments {
		id := coordinates.mustBatchArgument(ref)
		if !callback.present() {
			panic("cli: validated provider completion callback cannot be published")
		}
		callbacks.completions.arguments[id] = callback
	}
	for ref, callback := range batch.callbacks.commandProviders {
		id := coordinates.mustBatchProvider(ref)
		if callback == nil {
			panic("cli: validated provider callback cannot be published")
		}
		callbacks.commandProviders[id] = callback
	}
	return callbacks
}

func prepareProviderDestinationAcceptances(batch providerBatch, coordinates providerPublicationCoordinates, preview publicationPreview) []destinationAcceptance {
	acceptances := make([]destinationAcceptance, 0, len(batch.bindings))
	for _, binding := range batch.bindings {
		id := coordinates.mustBatchDestination(binding.destination)
		if binding.handle == nil {
			panic("cli: validated provider destination binding cannot be published")
		}
		record, ok := preview.Destination(id)
		if !ok || record.meta.id != id {
			panic("cli: published provider destination cannot be resolved")
		}
		definition, ok := record.Definition()
		if !ok || !definition.Valid() {
			panic("cli: published provider destination has no definition")
		}
		binding.handle.snapshot = definition.Snapshot
		acceptance := prepareDestinationAcceptance(binding.handle, acceptedDestinationIdentity{
			scope: batch.scope,
			id:    id,
		})
		acceptances = append(acceptances, acceptance)
	}
	return acceptances
}

func resolveProviderContributionTarget(preview publicationPreview, owner CommandID, coordinates providerPublicationCoordinates, target batchCommandTarget) (CommandID, bool) {
	switch target := target.(type) {
	case localCommandTarget:
		return coordinates.mustBatchCommand(target.command), true
	case pathTarget:
		return resolveCanonicalProviderPath(preview, owner, target.path)
	default:
		panic("cli: validated provider contribution has invalid command target")
	}
}

func resolveCanonicalProviderPath(preview publicationPreview, owner CommandID, path []string) (CommandID, bool) {
	if len(path) == 0 {
		return CommandID(0), false
	}
	grammar, ok := preview.Grammar()
	if !ok {
		return CommandID(0), false
	}
	current := owner
	for _, part := range path {
		dialect, ok := grammar.Dialect(current)
		if !ok {
			return CommandID(0), false
		}
		mode := dialect.CommandNormalization()
		if !mode.valid() {
			return CommandID(0), false
		}
		name, ok := normalizeName(mode, part)
		if !ok || name == "" {
			return CommandID(0), false
		}
		var target CommandID
		matches := 0
		for _, entry := range grammar.CommandEntries(current) {
			if entry.kind != CommandEntryCanonical {
				continue
			}
			candidate, valid := normalizeName(mode, entry.name)
			if valid && candidate == name {
				target = entry.target
				matches++
			}
		}
		if matches != 1 {
			return CommandID(0), false
		}
		current = target
	}
	return current, true
}

func providerShortcutDestinationID(coordinates providerPublicationCoordinates, target batchDestinationTarget) ValueID {
	switch target := target.(type) {
	case localDestinationTarget:
		return coordinates.mustBatchDestination(target.destination)
	case acceptedDestinationIdentity:
		if !target.id.present() || !target.scope.valid() {
			panic("cli: validated provider shortcut destination has invalid identity")
		}
		return target.id
	default:
		panic("cli: validated provider shortcut destination has invalid kind")
	}
}

func providerShortcutDefinition(preview publicationPreview, coordinates providerPublicationCoordinates, contribution shortcutContribution) (shortcutDefinition, bool) {
	defaults := make([]shortcutDefault, 0, len(contribution.defaults))
	for _, value := range contribution.defaults {
		switch value := value.(type) {
		case destinationDefault:
			source, ok := producerSource(value.source)
			if !ok {
				panic("cli: validated provider shortcut default has invalid source")
			}
			id := providerShortcutDestinationID(coordinates, value.destination)
			record, ok := preview.Destination(id)
			if !ok {
				panic("cli: validated provider shortcut destination is absent from publication")
			}
			definition, ok := record.Definition()
			if !ok || !definition.Valid() {
				panic("cli: validated provider shortcut destination has no definition")
			}
			if !validEffectiveDomainContribution(definition, value.value) {
				switch value.destination.(type) {
				case acceptedDestinationIdentity:
					return shortcutDefinition{}, false
				case localDestinationTarget:
					panic("cli: validated local provider shortcut default is outside its destination domain")
				default:
					panic("cli: validated provider shortcut default has invalid destination kind")
				}
			}
			defaults = append(defaults, acceptShortcutDestinationDefault(id, value.value, source))
		case flagDefault:
			source, ok := producerSource(value.source)
			if !ok || value.spelling == "" || len(value.values) == 0 {
				panic("cli: validated provider flag default is invalid")
			}
			defaults = append(defaults, acceptShortcutFlagDefault(value.spelling, value.values, source))
		default:
			panic("cli: validated provider shortcut default has invalid kind")
		}
	}
	source, ok := producerSource(contribution.source)
	if !ok {
		panic("cli: validated provider shortcut has invalid source")
	}
	definition := acceptShortcutDefinition(
		contribution.shortcut.Args,
		contribution.shortcut.AppendArgs,
		defaults,
		source,
	)
	if !definition.HasContribution() {
		panic("cli: validated provider shortcut has no contribution")
	}
	return definition, true
}
