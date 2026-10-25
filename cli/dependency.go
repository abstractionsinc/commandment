package cli

import (
	"context"
	"maps"
	"slices"
	"strings"
)

// dependencyAnalysis is the bounded value-resolution closure available to a
// dependency-bearing analysis callback before the full invocation value phase.
// It resolves only the requested destinations and their declared producer
// predecessors from ownership already committed in grammar.
type dependencyAnalysis struct {
	Values    valueAnalysis
	Ready     bool
	BlockedBy []ValueID
}

// analyzeDependencies resolves one declared dependency set without evaluating
// unrelated destinations. Required dependencies must resolve. Optional
// dependencies may be absent or inactive, but failure/blocking/invalidity is
// never converted into absence.
func analyzeDependencies(ctx context.Context, operation *invocation, grammar grammarResult, requirements []dependencyRequirement) dependencyAnalysis {
	if operation == nil || operation.analysis == nil {
		panic("invocation: nil operation")
	}
	return analyzeDependenciesAt(ctx, operation, operation.analysis, grammar, requirements)
}

func analyzeDependenciesAt(ctx context.Context, operation *invocation, state *analysisState, grammar grammarResult, requirements []dependencyRequirement) dependencyAnalysis {
	if operation == nil || state == nil {
		panic("invocation: nil operation analysis")
	}
	release := state.work.enter()
	defer release()

	if len(requirements) == 0 {
		analysis := dependencyAnalysis{
			Values: operationDependencySeed(state),
			Ready:  true,
		}
		return analysis
	}
	if len(grammar.Issues) != 0 {
		analysis := operationDependencySeed(state)
		analysis.Unsupported = true
		result := dependencyAnalysis{
			Values:    analysis,
			BlockedBy: dependencyRequirementIDs(requirements),
		}
		return result
	}
	view := state.effectiveModel
	model, ok := view.EffectiveGrammar()
	if !ok {
		analysis := operationDependencySeed(state)
		analysis.Unsupported = true
		result := dependencyAnalysis{
			Values:    analysis,
			BlockedBy: dependencyRequirementIDs(requirements),
		}
		return result
	}

	// The dependency seed is operation-private acceleration. Mutate it under one
	// operation-local critical section so each incremental dependency settlement
	// reuses already-settled destinations without cloning the whole growing seed.
	// The returned ValueAnalysis is still a detached snapshot, but it is bounded
	// to the exact transitive closure requested by this call rather than leaking
	// unrelated cached destinations from earlier dependency demands.
	state.dependencyMu.Lock()
	defer state.dependencyMu.Unlock()
	analysis := &state.dependencyValues
	ensureValueAnalysisMaps(analysis)

	selected := make(map[CommandID]struct{}, len(grammar.Selection.Path))
	for _, command := range grammar.Selection.Path {
		selected[command] = struct{}{}
	}
	needed := make(map[ValueID]struct{})
	nodes := make([]valueSettlementNode, 0, len(requirements))
	queue := make([]ValueID, 0, len(requirements))
	for _, requirement := range requirements {
		if !requirement.Destination().present() {
			analysis.Unsupported = true
			continue
		}
		queue = append(queue, requirement.Destination())
	}
	for len(queue) != 0 {
		id := queue[0]
		queue = queue[1:]
		if _, exists := needed[id]; exists {
			continue
		}
		record, ok := view.Destination(id)
		if !ok {
			analysis.Unsupported = true
			continue
		}
		if _, ok := selected[record.command]; !ok ||
			!analysisGrammarProfileApplies(grammar.SelectedProfiles, record.command, record.meta.profile) {
			continue
		}
		needed[id] = struct{}{}
		node := prepareValueSettlementNode(record, grammar.SelectedProfiles)
		nodes = append(nodes, node)
		for _, producer := range node.producers {
			for _, dependency := range producer.dependencies {
				queue = append(queue, dependency.Destination())
			}
		}
	}
	plans := make(map[ValueID][]valueCandidatePlan)
	builder := valuePlanBuilder{
		state:    state,
		profiles: grammar.SelectedProfiles,
		selected: selected,
		include:  needed,
		plans:    plans,
		analysis: analysis,
	}
	builder.appendGrammarValueCandidatePlans(ctx, operation, model, grammar)
	builder.appendEnvironmentValueCandidatePlans(operation)
	builder.appendValueSupplyPlans(ctx)
	builder.appendShortcutDefaultPlans(model)
	settleValueDestinations(ctx, operation, state, model, grammar, nodes, plans, analysis)

	result := dependencyAnalysis{
		Values: cloneValueAnalysisDestinations(*analysis, needed),
		Ready:  !analysis.Unsupported,
	}
	for _, requirement := range requirements {
		resolution, ok := analysis.Destinations[requirement.Destination()]
		if !ok {
			result.Ready = false
			result.BlockedBy = appendUniqueDestinationID(result.BlockedBy, requirement.Destination())
			continue
		}
		switch resolution.State {
		case destinationStateResolved:
		case destinationStateAbsent, destinationStateInactive:
			if !requirement.Optional() {
				result.Ready = false
				result.BlockedBy = appendUniqueDestinationID(result.BlockedBy, requirement.Destination())
			}
		default:
			result.Ready = false
			result.BlockedBy = appendUniqueDestinationID(result.BlockedBy, requirement.Destination())
		}
	}
	return result
}

func (operation *invocation) settleDependencies(ctx context.Context, grammar grammarResult, requirements []dependencyRequirement) (valueAnalysis, error) {
	if operation == nil || operation.analysis == nil {
		panic("cli: dependency settlement requires an invocation analysis")
	}
	if len(requirements) == 0 {
		return valueAnalysis{}, nil
	}

	model := operation.analysis.effectiveModel
	destinations, domains := dependencyDemandClosure(model, grammar, requirements)
	if dependencyDestinationsNeedFacts(model, destinations) && !operation.supplyFacts(ctx, grammar.Selection.Path) {
		return valueAnalysis{}, operation.supplyDiagnosticFailure()
	}
	if dependencyDestinationsNeedDefaults(model, destinations) && !operation.supplyDefaults(ctx, grammar.Selection) {
		return valueAnalysis{}, operation.supplyDiagnosticFailure()
	}

	materializeDependencyDomains(ctx, operation, grammar, domains)
	outcomes := operation.dynamicDomainResults()
	domainDiagnostics, domainCause := dynamicDomainMaterializationDiagnostics(outcomes)
	if len(domainDiagnostics) != 0 {
		return valueAnalysis{}, dynamicDomainDiagnosticFailure(operation, outcomes, domainCause)
	}
	if !dynamicDomainOutcomesComplete(outcomes, domains) {
		panic("cli: settled dependency domains are incomplete without a diagnostic failure")
	}

	model = operation.analysis.effectiveModel
	dependencyAnalysis := analyzeDependencies(ctx, operation, grammar, requirements)
	values := dependencyAnalysis.Values
	enforceEffectiveDomainMembership(model, &values)
	diagnostics, valueTruncated := runValueDiagnostics(operation, model, values)
	if len(diagnostics) != 0 {
		return valueAnalysis{}, valueDiagnosticFailure(operation, model, values, diagnostics, grammar.Truncated || valueTruncated)
	}
	if !dependencyAnalysis.Ready || valueAnalysisIncomplete(values) {
		blocked := dependencyAnalysis.BlockedBy
		if len(blocked) == 0 {
			blocked = blockedDependencyIDs(requirements, values)
		}
		if len(blocked) == 0 {
			panic("cli: dependency settlement is incomplete without a blocking dependency")
		}
		return valueAnalysis{}, blockedDependencyFailure(operation, model, values, blocked)
	}
	return values, nil
}

func (operation *invocation) supplyDiagnosticFailure() error {
	if operation == nil {
		panic("cli: nil invocation supply failure")
	}
	code, err := operation.supplyFailure()
	if err == nil {
		panic("cli: incomplete invocation supply has no failure")
	}
	return operationDiagnosticFailure(operation, code, "invocation value supply failed", err)
}

func dependencyDemandClosure(model *effectiveModel, grammar grammarResult, requirements []dependencyRequirement) (map[ValueID]struct{}, []int) {
	destinations := make(map[ValueID]struct{})
	queue := make([]ValueID, 0, len(requirements))
	addDestination := func(id ValueID) {
		if !id.present() {
			return
		}
		if _, exists := destinations[id]; exists {
			return
		}
		destinations[id] = struct{}{}
		queue = append(queue, id)
	}
	for _, requirement := range requirements {
		addDestination(requirement.Destination())
	}

	demanded := make(map[int]struct{})
	for len(queue) != 0 {
		destination := queue[0]
		queue = queue[1:]
		record, ok := model.Destination(destination)
		if ok {
			for _, producer := range record.Producers() {
				if !grammarProfileApplies(grammar.SelectedProfiles, record.command, producer.profile) {
					continue
				}
				for _, dependency := range producer.Dependencies() {
					addDestination(dependency.Destination())
				}
			}
		}
		for _, index := range model.ValueDomainIndexesForDestination(destination) {
			domain, ok := model.ValueDomain(index)
			if !ok {
				panic("cli: accepted dependency destination references an absent value domain")
			}
			definition, dynamic := domain.DynamicDefinition()
			if !dynamic {
				continue
			}
			if _, exists := demanded[index]; exists {
				continue
			}
			demanded[index] = struct{}{}
			for _, dependency := range definition.Dependencies() {
				addDestination(dependency.Destination())
			}
		}
	}

	domains := make([]int, 0, len(demanded))
	for index := range demanded {
		domains = append(domains, index)
	}
	slices.Sort(domains)
	return destinations, domains
}

func dependencyDestinationsNeedFacts(model *effectiveModel, destinations map[ValueID]struct{}) bool {
	for destination := range destinations {
		record, ok := model.Destination(destination)
		if !ok {
			continue
		}
		definition, ok := record.Definition()
		if !ok {
			continue
		}
		switch definition.Kind() {
		case destinationKindFactRequired, destinationKindFactResolved:
			return true
		case destinationKindUnknown,
			destinationKindScalar,
			destinationKindSequence,
			destinationKindKeyed,
			destinationKindOpaque,
			destinationKindDerived,
			destinationKindOptionalDerived:
		}
	}
	return false
}

func dependencyDestinationsNeedDefaults(model *effectiveModel, destinations map[ValueID]struct{}) bool {
	for destination := range destinations {
		record, ok := model.Destination(destination)
		if !ok {
			continue
		}
		definition, ok := record.Definition()
		if !ok {
			continue
		}
		switch definition.Kind() {
		case destinationKindScalar,
			destinationKindSequence,
			destinationKindKeyed,
			destinationKindOpaque:
			return true
		case destinationKindUnknown,
			destinationKindDerived,
			destinationKindOptionalDerived,
			destinationKindFactRequired,
			destinationKindFactResolved:
		}
	}
	return false
}

func materializeDependencyDomains(ctx context.Context, operation *invocation, grammar grammarResult, demanded []int) {
	if len(demanded) == 0 {
		return
	}
	model := operation.analysis.effectiveModel
	demandedSet := make(map[int]struct{}, len(demanded))
	for _, index := range demanded {
		demandedSet[index] = struct{}{}
	}
	const (
		dependencyDomainVisiting uint8 = iota + 1
		dependencyDomainComplete
	)
	state := make(map[int]uint8, len(demanded))
	var visit func(int)
	visit = func(index int) {
		switch state[index] {
		case dependencyDomainVisiting:
			panic("cli: accepted dynamic value-domain dependencies contain a cycle")
		case dependencyDomainComplete:
			return
		}
		state[index] = dependencyDomainVisiting
		domain, ok := model.ValueDomain(index)
		if !ok {
			panic("cli: demanded dependency value domain cannot resolve")
		}
		definition, dynamic := domain.DynamicDefinition()
		if !dynamic {
			panic("cli: demanded dependency value domain is not dynamic")
		}
		predecessors := make(map[int]struct{})
		for _, dependency := range definition.Dependencies() {
			for _, predecessor := range model.ValueDomainIndexesForDestination(dependency.Destination()) {
				if predecessor == index {
					continue
				}
				if _, needed := demandedSet[predecessor]; needed {
					predecessors[predecessor] = struct{}{}
				}
			}
		}
		ordered := make([]int, 0, len(predecessors))
		for predecessor := range predecessors {
			ordered = append(ordered, predecessor)
		}
		slices.Sort(ordered)
		for _, predecessor := range ordered {
			visit(predecessor)
		}
		operation.materializeDynamicDomain(ctx, grammar, index)
		state[index] = dependencyDomainComplete
	}
	for _, index := range demanded {
		visit(index)
	}
}

func blockedDependencyIDs(requirements []dependencyRequirement, values valueAnalysis) []ValueID {
	blocked := make([]ValueID, 0, len(requirements))
	for _, requirement := range requirements {
		id := requirement.Destination()
		resolution, ok := values.Destinations[id]
		if !ok {
			blocked = appendUniqueDestinationID(blocked, id)
			continue
		}
		switch resolution.State {
		case destinationStateResolved:
		case destinationStateAbsent, destinationStateInactive:
			if !requirement.Optional() {
				blocked = appendUniqueDestinationID(blocked, id)
			}
		default:
			blocked = appendUniqueDestinationID(blocked, id)
		}
	}
	return blocked
}

func dependencyRequirementIDs(requirements []dependencyRequirement) []ValueID {
	result := make([]ValueID, 0, len(requirements))
	for _, requirement := range requirements {
		result = appendUniqueDestinationID(result, requirement.Destination())
	}
	return result
}

func appendUniqueDestinationID(values []ValueID, value ValueID) []ValueID {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

// grammarProfileApplies reports whether a profile-scoped declaration is eligible
// under the grammar profile selected for its owning command. Profile-free
// declarations are always active. All pre-analysis demand walkers must use this
// same eligibility law rather than reconstructing profile checks locally.
func grammarProfileApplies(selected map[CommandID]GrammarProfileID, command CommandID, profile GrammarProfileID) bool {
	if !profile.present() {
		return true
	}
	return selected[command] == profile
}

func analysisGrammarProfileApplies(selected map[CommandID]GrammarProfileID, command CommandID, profile GrammarProfileID) bool {
	return grammarProfileApplies(selected, command, profile)
}

func activeInputProducerDestinations(state *analysisState, model grammarView, grammar grammarResult) map[ValueID]struct{} {
	var active map[ValueID]struct{}
	for _, command := range grammar.Selection.Path {
		for _, environment := range state.effectiveModel.EnvironmentInputsForCommand(command) {
			if !analysisGrammarProfileApplies(grammar.SelectedProfiles, environment.command, environment.meta.profile) {
				continue
			}
			if active == nil {
				active = make(map[ValueID]struct{})
			}
			active[environment.destination] = struct{}{}
		}
		for _, flag := range model.Flags(command) {
			if !analysisGrammarProfileApplies(grammar.SelectedProfiles, flag.command, flag.profile) {
				continue
			}
			if active == nil {
				active = make(map[ValueID]struct{})
			}
			active[flag.destination] = struct{}{}
		}
	}
	if grammar.Arguments.Form.present() {
		for _, form := range model.ArgumentForms(grammar.Selection.Selected) {
			if form.id != grammar.Arguments.Form {
				continue
			}
			for _, term := range form.Terms() {
				destination, ok := term.Destination()
				if !ok {
					continue
				}
				if active == nil {
					active = make(map[ValueID]struct{})
				}
				active[destination] = struct{}{}
			}
		}
	}
	return active
}

func destinationProducerActive(active map[ValueID]struct{}, record destinationRecord, hasActiveProducer bool) bool {
	if hasActiveProducer {
		return true
	}
	_, ok := active[record.meta.id]
	return ok
}

func (builder *valuePlanBuilder) appendValueSupplyPlans(ctx context.Context) {
	profiles := builder.profiles
	selected := builder.selected
	include := builder.include
	plans := builder.plans
	analysis := builder.analysis
	for _, contribution := range builder.state.valueSupplySnapshot() {
		record, ok := builder.state.effectiveModel.Destination(contribution.Destination)
		if !ok {
			analysis.Unsupported = true
			continue
		}
		if _, ok := selected[record.command]; !ok ||
			!analysisGrammarProfileApplies(profiles, record.command, record.meta.profile) {
			continue
		}
		if include != nil {
			if _, wanted := include[record.meta.id]; !wanted {
				continue
			}
		}
		var producerKind ValueProducerKind
		tier := ResolutionTierUnknown
		switch contribution.Kind {
		case valueSupplyDefault:
			producerKind, tier = ValueProducerKindInvocationDefault, ResolutionTierInvocationDefault
		case valueSupplyFact:
			producerKind = ValueProducerKindSuppliedFact
		default:
			analysis.Unsupported = true
			continue
		}
		evaluate := contribution.Evaluate
		plan := valueCandidatePlan{
			producerKind:  producerKind,
			tier:          tier,
			order:         contribution.Order,
			sensitive:     contribution.Sensitive,
			producerIndex: -1,
			whole:         func() (any, error) { return evaluate(ctx) },
		}
		destination := record.meta.id
		plans[destination] = append(plans[destination], plan)
	}
}

func (builder *valuePlanBuilder) appendShortcutDefaultPlans(model grammarView) {
	profiles := builder.profiles
	selected := builder.selected
	include := builder.include
	plans := builder.plans
	analysis := builder.analysis
	if !builder.state.shortcutDefaultsActive() {
		return
	}
	for _, contribution := range builder.state.shortcutDefaultSnapshot() {
		value := contribution.Default
		var destination ValueID
		plan := valueCandidatePlan{
			producerIndex: -1,
			producerKind:  ValueProducerKindShortcutDefault,
			tier:          ResolutionTierShortcutDefault,
			order:         contribution.Order,
			sensitive:     value.Source().Sensitive(),
		}
		switch value.Kind() {
		case shortcutDefaultKindDestination:
			destination = value.Destination()
			record, ok := builder.state.effectiveModel.Destination(destination)
			if !ok || !shortcutDefaultDestinationSelected(record, profiles, selected, include) {
				continue
			}
			definition, ok := record.Definition()
			if !ok {
				analysis.Unsupported = true
				continue
			}
			var valid bool
			plan, valid = typedFixedPlan(plan, definition, value.Value())
			if !valid {
				analysis.Unsupported = true
				continue
			}
		case shortcutDefaultKindFlag:
			path, ok := analysisCanonicalCommandPath(builder.state, contribution.Target)
			if !ok {
				analysis.Unsupported = true
				continue
			}
			dialect, ok := model.Dialect(contribution.Target)
			if !ok {
				analysis.Unsupported = true
				continue
			}
			match := matchSelectedPathFlag(model, path, dialect, value.Spelling())
			if !match.found || match.ambiguous || match.match != nameMatchKindExact || !match.flag.id.present() ||
				!match.spelling.id.present() {
				analysis.Unsupported = true
				continue
			}
			destination = match.flag.destination
			record, ok := builder.state.effectiveModel.Destination(destination)
			if !ok || !shortcutDefaultDestinationSelected(record, profiles, selected, include) {
				continue
			}
			definition, ok := record.Definition()
			if !ok {
				analysis.Unsupported = true
				continue
			}
			plan.sensitive = plan.sensitive || match.spelling.sensitive
			plan.declaration = valueCandidateDeclaration{
				Flag:         match.flag.id,
				FlagSpelling: match.spelling.id,
			}
			texts := value.Values()
			plan.input, plan.hasInput = strings.Join(texts, "\x00"), true
			switch definition.Kind() {
			case destinationKindScalar:
				if len(texts) != 1 {
					analysis.Unsupported = true
					continue
				}
				language, ok := definition.Language()
				if !ok {
					analysis.Unsupported = true
					continue
				}
				text := texts[0]
				plan.whole = func() (any, error) { return language.Parse(text) }
			case destinationKindSequence:
				language, ok := definition.Language()
				if !ok {
					analysis.Unsupported = true
					continue
				}
				copied := append([]string(nil), texts...)
				plan.sequence = func() ([]any, error) { return parseSequence(language, copied) }
			case destinationKindKeyed:
				language, ok := definition.KeyedLanguage()
				if !ok {
					analysis.Unsupported = true
					continue
				}
				inputs := make([]valueCandidateComponentInput, len(texts))
				for i, text := range texts {
					inputs[i] = valueCandidateComponentInput{
						text:    text,
						present: true,
					}
				}
				entries, ok := keyedEntryPlans(language, inputs)
				if !ok {
					analysis.Unsupported = true
					continue
				}
				plan.keyedBatch = true
				plan.keyedEntries = entries
			default:
				analysis.Unsupported = true
				continue
			}
		default:
			analysis.Unsupported = true
			continue
		}
		plans[destination] = append(plans[destination], plan)
	}
}

func shortcutDefaultDestinationSelected(record destinationRecord, profiles map[CommandID]GrammarProfileID, selected map[CommandID]struct{}, include map[ValueID]struct{}) bool {
	if _, ok := selected[record.command]; !ok ||
		!analysisGrammarProfileApplies(profiles, record.command, record.meta.profile) {
		return false
	}
	if include != nil {
		_, ok := include[record.meta.id]
		return ok
	}
	return true
}

func analysisCanonicalCommandPath(state *analysisState, target CommandID) ([]CommandID, bool) {
	if state == nil || !target.present() {
		return nil, false
	}
	var reversed []CommandID
	seen := make(map[CommandID]struct{})
	current := target
	for current.present() {
		if _, duplicate := seen[current]; duplicate {
			return nil, false
		}
		seen[current] = struct{}{}
		record, ok := state.effectiveModel.Command(current)
		if !ok {
			return nil, false
		}
		reversed = append(reversed, current)
		parent, ok := record.Parent()
		if !ok {
			break
		}
		current = parent
	}
	path := make([]CommandID, len(reversed))
	for i := range reversed {
		path[len(reversed)-1-i] = reversed[i]
	}
	return path, true
}

// Dependency is a declared destination dependency for analysis callbacks and computations. Optional
// tolerates resolved absence; it does not hide failure or blocking.
type Dependency interface {
	dependencyTarget() (*destinationHandle, bool)
}

func (destination Destination[C, T]) dependencyTarget() (*destinationHandle, bool) {
	return destination.destinationHandle(), false
}

func (declaration FlagDeclaration[C, T]) dependencyTarget() (*destinationHandle, bool) {
	return declaration.destination.destinationHandle(), false
}

func (declaration ArgumentDeclaration[C, T]) dependencyTarget() (*destinationHandle, bool) {
	return declaration.destination.destinationHandle(), false
}

// OptionalDependency returns a dependency that tolerates resolved absence of this destination.
func (destination Destination[C, T]) OptionalDependency() Dependency {
	dependency := destinationDependency{
		handle:   destination.destinationHandle(),
		optional: true,
	}
	return dependency
}

// OptionalDependency returns a dependency that tolerates resolved absence of the flag destination.
func (declaration FlagDeclaration[C, T]) OptionalDependency() Dependency {
	dependency := destinationDependency{
		handle:   declaration.destination.destinationHandle(),
		optional: true,
	}
	return dependency
}

// OptionalDependency returns a dependency that tolerates resolved absence of the argument destination.
func (declaration ArgumentDeclaration[C, T]) OptionalDependency() Dependency {
	dependency := destinationDependency{
		handle:   declaration.destination.destinationHandle(),
		optional: true,
	}
	return dependency
}

type destinationDependency struct {
	handle   *destinationHandle
	optional bool
}

func (dependency destinationDependency) dependencyTarget() (*destinationHandle, bool) {
	return dependency.handle, dependency.optional
}

// Dependencies is the immutable detached historical view authorized for one callback invocation.
type Dependencies struct {
	_      [0]func()
	values *dependencyValues
}

type dependencyValues struct {
	scope         identityScope
	byDestination map[ValueID]any
}

// Get returns a detached value only when destination belongs to this dependency view's identity
// universe and was explicitly authorized for the callback.
func (dependencies Dependencies) Get[C, T any](destination destinationLike[C, T]) (T, bool) {
	var zero T
	if dependencies.values == nil || destination == nil {
		return zero, false
	}
	handle := destination.destinationHandle()
	if handle == nil {
		return zero, false
	}
	identity, ok := handle.currentIdentity().(acceptedDestinationIdentity)
	if !ok || !dependencies.values.scope.includes(identity.scope) {
		return zero, false
	}
	value, ok := dependencies.values.byDestination[identity.id]
	if !ok {
		return zero, false
	}
	if handle.snapshot == nil {
		panic("cli: accepted destination has no snapshot policy")
	}
	detached, ok := handle.snapshotValue(value)
	if !ok {
		panic("cli: accepted dependency value failed canonical snapshot")
	}
	typed, ok := detached.(T)
	if !ok {
		panic("cli: accepted dependency value has wrong Go type")
	}
	return typed, true
}

func dependencyDescriptors(input []Dependency) ([]destinationDependency, bool) {
	result := make([]destinationDependency, len(input))
	for i, dependency := range input {
		if dependency == nil {
			return nil, false
		}
		handle, optional := dependency.dependencyTarget()
		if handle == nil {
			return nil, false
		}
		result[i] = destinationDependency{
			handle:   handle,
			optional: optional,
		}
	}
	return result, true
}

func freezeAuthoringDestinationDependencies(draft *declarationDraft, pending []destinationDependency) ([]authoringDestinationDependency, bool) {
	result := make([]authoringDestinationDependency, len(pending))
	for index, dependency := range pending {
		relation, ok := captureAuthoringDestinationIdentity(draft, dependency.handle)
		if !ok {
			return nil, false
		}
		result[index] = authoringDestinationDependency{
			destination: relation,
			optional:    dependency.optional,
		}
	}
	return result, true
}

type destinationAcceptanceResolver interface {
	ValueID(identity destinationIdentity) (ValueID, bool)
}

type resolvedDestinationDependencies struct {
	requirements []dependencyRequirement
	keys         map[destinationRef]ValueID
	ids          map[ValueID]struct{}
}

func resolveDestinationDependencies(dependencies []authoringDestinationDependency, resolver destinationAcceptanceResolver) (resolvedDestinationDependencies, bool) {
	if resolver == nil {
		return resolvedDestinationDependencies{}, len(dependencies) == 0
	}
	result := resolvedDestinationDependencies{
		requirements: make([]dependencyRequirement, 0, len(dependencies)),
		keys:         make(map[destinationRef]ValueID),
		ids:          make(map[ValueID]struct{}),
	}
	for _, dependency := range dependencies {
		id, ok := resolver.ValueID(dependency.destination)
		if !ok || !id.present() {
			return resolvedDestinationDependencies{}, false
		}
		if _, duplicate := result.ids[id]; duplicate {
			return resolvedDestinationDependencies{}, false
		}
		result.ids[id] = struct{}{}
		if staged, ok := dependency.destination.(stagedDestinationIdentity); ok {
			result.keys[staged.ref] = id
		}
		result.requirements = append(result.requirements, acceptDependencyRequirement(id, dependency.optional))
	}
	return result, true
}

func addDestinationRelationLookup(resolved *resolvedDestinationDependencies, identity destinationIdentity, id ValueID) {
	if resolved == nil || !id.present() {
		return
	}
	if resolved.keys == nil {
		resolved.keys = make(map[destinationRef]ValueID)
	}
	if resolved.ids == nil {
		resolved.ids = make(map[ValueID]struct{})
	}
	resolved.ids[id] = struct{}{}
	if staged, ok := identity.(stagedDestinationIdentity); ok {
		resolved.keys[staged.ref] = id
	}
}

func freezeResolvedDestinationDependencies(resolved resolvedDestinationDependencies) resolvedDestinationDependencies {
	frozen := resolvedDestinationDependencies{
		requirements: append([]dependencyRequirement(nil), resolved.requirements...),
		keys:         make(map[destinationRef]ValueID, len(resolved.keys)),
		ids:          make(map[ValueID]struct{}, len(resolved.ids)),
	}
	maps.Copy(frozen.keys, resolved.keys)
	for id := range resolved.ids {
		frozen.ids[id] = struct{}{}
	}
	return frozen
}

type authoringDestinationDependency struct {
	destination destinationIdentity
	optional    bool
}

func dependencyOrdinal(index int) uint64 {
	if index < 0 {
		panic("application: negative dependency index")
	}
	return uint64(index) + 1
}

// dependenciesFromValueAnalysis constructs the exact immutable callback view for one declared
// dependency set. Producer-predecessor values resolved only to establish the requested
// destinations remain unavailable unless they were themselves explicitly declared.
func dependenciesFromValueAnalysis(scope identityScope, model *effectiveModel, requirements []dependencyRequirement, values valueAnalysis) (Dependencies, bool) {
	if !scope.valid() || scope.operation == nil || !model.Valid() {
		panic("cli: invalid dependency view scope")
	}
	ready := !values.Unsupported
	byDestination := make(map[ValueID]any, len(requirements))
	for _, requirement := range requirements {
		id := requirement.Destination()
		resolution, ok := values.Destinations[id]
		if !ok {
			ready = false
			continue
		}
		switch resolution.State {
		case destinationStateResolved:
			record, exists := model.Destination(id)
			if !exists {
				panic("cli: resolved dependency destination is absent from effective model")
			}
			definition, exists := record.Definition()
			if !exists {
				panic("cli: resolved dependency destination has no definition")
			}
			detached, snapshotOK := definition.Snapshot(resolution.Value)
			if !snapshotOK {
				panic("cli: resolved dependency value failed canonical snapshot")
			}
			byDestination[id] = detached
		case destinationStateAbsent, destinationStateInactive:
			if !requirement.Optional() {
				ready = false
			}
		default:
			ready = false
		}
	}
	view := Dependencies{
		values: &dependencyValues{
			scope:         scope,
			byDestination: byDestination,
		},
	}
	return view, ready
}

// dependenciesFromAuthorizedValueAnalysis constructs an immutable view for a callback whose
// authorized read set is already established by its semantic owner. Only resolved values are
// present; unresolved optional values remain unavailable through Dependencies.Get.
func dependenciesFromAuthorizedValueAnalysis(scope identityScope, model *effectiveModel, authorized []ValueID, values valueAnalysis) Dependencies {
	if !scope.valid() || scope.operation == nil || !model.Valid() {
		panic("cli: invalid dependency view scope")
	}
	byDestination := make(map[ValueID]any, len(authorized))
	for _, id := range authorized {
		resolution, ok := values.Destinations[id]
		if !ok || resolution.State != destinationStateResolved {
			continue
		}
		record, exists := model.Destination(id)
		if !exists {
			panic("cli: resolved dependency destination is absent from effective model")
		}
		definition, exists := record.Definition()
		if !exists {
			panic("cli: resolved dependency destination has no definition")
		}
		detached, snapshotOK := definition.Snapshot(resolution.Value)
		if !snapshotOK {
			panic("cli: resolved dependency value failed canonical snapshot")
		}
		byDestination[id] = detached
	}
	snapshot := &dependencyValues{
		scope:         scope,
		byDestination: byDestination,
	}
	dependencies := Dependencies{
		values: snapshot,
	}
	return dependencies
}
