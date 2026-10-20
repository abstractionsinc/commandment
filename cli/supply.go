package cli

import (
	"context"
	"errors"
)

// DefaultProvider supplies invocation-scoped default candidates during analysis. Providers are
// created with DefaultsFor and are applied only to matching command configuration scopes;
// provider failure is reported as a Commandment default-supply failure.
type DefaultProvider interface{ defaultProvider() }

// FactProvider supplies invocation-scoped fact candidates during analysis. Providers are
// created with FactsFor and are applied only to matching command configuration scopes;
// provider failure is reported as a Commandment fact-supply failure.
type FactProvider interface{ factProvider() }

type typedDefaultProvider[C any] struct {
	provide func(context.Context, *DefaultSupply[C]) error
}

func (typedDefaultProvider[C]) defaultProvider() {}

type typedFactProvider[C any] struct {
	provide func(context.Context, *FactSupply[C]) error
}

func (typedFactProvider[C]) factProvider() {}

// DefaultsFor creates a typed DefaultProvider whose callback receives the matching
// command configuration and selection for the current.
func DefaultsFor[C any](provide func(context.Context, *DefaultSupply[C]) error) DefaultProvider {
	return typedDefaultProvider[C]{
		provide: provide,
	}
}

// FactsFor creates a typed FactProvider whose callback receives the matching
// command configuration for the current.
func FactsFor[C any](provide func(context.Context, *FactSupply[C]) error) FactProvider {
	return typedFactProvider[C]{
		provide: provide,
	}
}

// DefaultSupply is the callback-local surface for contributing invocation defaults.
// Config is the matching command configuration and Selection is a detached view
// of the selected command path for this.
type DefaultSupply[C any] struct {
	Config    *C
	Selection CommandSelection
	state     *defaultsState
}

type defaultsState struct{ supply *invocationSupplyState }

// Set contributes value as an invocation default for destination. Mutable values
// are snapshotted when staged; source options control producer metadata such as
// sensitivity.
func (defaults *DefaultSupply[C]) Set[T any](destination destinationLike[C, T], value T, options ...SourceOption) {
	stageInvocationSupply(defaults.state, destinationHandleOf(destination), value, nil, options)
}

// Load contributes a lazily evaluated invocation default for destination. The
// loader runs with the operation context, and a returned value must satisfy the
// destination's accepted type/value contract.
func (defaults *DefaultSupply[C]) Load[T any](destination destinationLike[C, T], load func(context.Context) (T, error), options ...SourceOption) {
	var erased func(context.Context) (any, error)
	if load != nil {
		erased = func(ctx context.Context) (any, error) { return load(ctx) }
	}
	stageInvocationSupply(defaults.state, destinationHandleOf(destination), nil, erased, options)
}

// FactSupply is the callback-local surface for supplying required or resolved facts.
// Config is the matching command configuration for the current.
type FactSupply[C any] struct {
	Config *C
	state  *factSupplyState
}

type factSupplyState struct{ supply *invocationSupplyState }

// Set supplies value for a fact destination in the current command scope. The
// contributed value is snapshotted and participates in normal provenance and
// sensitivity tracking.
func (supply *FactSupply[C]) Set[T any](destination destinationLike[C, T], value T, options ...SourceOption) {
	stageInvocationFact(supply.state, destinationHandleOf(destination), value, options)
}

var errInvalidInvocationSupply = errors.New("cli: invalid invocation supply")

type invocationSupplyState struct {
	operation *invocation
	analysis  *analysisState
	command   CommandID
	kind      valueSupplyKind
	staged    []valueSupply
	invalid   bool
}

func stageInvocationSupply(state *defaultsState, destination *destinationHandle, value any, load func(context.Context) (any, error), options []SourceOption) {
	if state == nil || state.supply == nil {
		return
	}
	stageValueSupply(state.supply, destination, value, load, options, valueSupplyDefault)
}

func stageInvocationFact(state *factSupplyState, destination *destinationHandle, value any, options []SourceOption) {
	if state == nil || state.supply == nil {
		return
	}
	stageValueSupply(state.supply, destination, value, nil, options, valueSupplyFact)
}

func stageValueSupply(state *invocationSupplyState, destination *destinationHandle, value any, load func(context.Context) (any, error), options []SourceOption, kind valueSupplyKind) {
	identity, identityOK := acceptedDestinationHandleIdentity(destination)
	if state == nil || state.invalid || kind != state.kind || !identityOK ||
		state.operation == nil || state.analysis == nil || !state.operation.scope.includes(identity.scope) {
		if state != nil {
			state.invalid = true
		}
		return
	}
	model := state.analysis.effectiveModel
	record, ok := model.Destination(identity.id)
	if !ok || record.command != state.command || record.meta.profile.present() {
		state.invalid = true
		return
	}
	definition, ok := record.Definition()
	if !ok {
		state.invalid = true
		return
	}
	if kind == valueSupplyFact {
		if definition.Kind() != destinationKindFactRequired &&
			definition.Kind() != destinationKindFactResolved {
			state.invalid = true
			return
		}
	} else if definition.Kind() == destinationKindFactRequired || definition.Kind() == destinationKindFactResolved || definition.Kind() == destinationKindDerived || definition.Kind() == destinationKindOptionalDerived {
		state.invalid = true
		return
	}
	source, ok := producerSource(options)
	if !ok {
		state.invalid = true
		return
	}
	binding, ok := record.Binding()
	if !ok {
		state.invalid = true
		return
	}
	configuration, ok := state.operation.analysisConfigurationAt(state.analysis, state.command, nil)
	if !ok {
		state.invalid = true
		return
	}
	var evaluatorOK bool
	if load == nil {
		load, evaluatorOK = snapshotValueSupplyEvaluator(definition, binding, configuration, value)
	} else {
		load = validatedValueSupplyEvaluator(definition, binding, configuration, load)
		evaluatorOK = true
	}
	if !evaluatorOK {
		state.invalid = true
		return
	}
	valueSupplyValue := valueSupply{
		Kind:        kind,
		Destination: identity.id,
		Order:       uint64(len(state.staged) + 1),
		Sensitive:   source.Sensitive(),
		Evaluate:    load,
	}
	state.staged = append(state.staged, valueSupplyValue)
}

func snapshotValueSupplyEvaluator(definition destinationDefinition, binding configurationBinding, configuration, value any) (func(context.Context) (any, error), bool) {
	if !binding.Accepts(configuration, value) {
		return nil, false
	}
	captured, ok := definition.Snapshot(value)
	if !ok {
		return nil, false
	}
	return func(context.Context) (any, error) {
		snapshot, snapshotOK := definition.Snapshot(captured)
		if !snapshotOK {
			return nil, errInvalidInvocationSupply
		}
		return snapshot, nil
	}, true
}

func validatedValueSupplyEvaluator(definition destinationDefinition, binding configurationBinding, configuration any, load func(context.Context) (any, error)) func(context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		resolved, err := load(ctx)
		if err != nil {
			return nil, err
		}
		if !binding.Accepts(configuration, resolved) {
			return nil, errInvalidInvocationSupply
		}
		snapshot, ok := definition.Snapshot(resolved)
		if !ok {
			return nil, errInvalidInvocationSupply
		}
		return snapshot, nil
	}
}

type defaultProviderRunner interface {
	DefaultProvider
	validDefaultProvider() bool
	runDefaultProvider(ctx context.Context, operation *invocation, analysis *analysisState, command CommandID, selection CommandSelection) ([]valueSupply, bool, error)
}

type factProviderRunner interface {
	FactProvider
	validFactProvider() bool
	runFactProvider(ctx context.Context, operation *invocation, analysis *analysisState, command CommandID) ([]valueSupply, bool, error)
}

func (provider typedDefaultProvider[C]) runDefaultProvider(ctx context.Context, operation *invocation, analysis *analysisState, command CommandID, selection CommandSelection) ([]valueSupply, bool, error) {
	seed := operationDependencySeed(analysis)
	configuration, ok := operation.analysisConfigurationAt(analysis, command, seed.Destinations)
	if !ok {
		return nil, false, nil
	}
	typed, ok := configuration.(*C)
	if !ok {
		return nil, false, nil
	}
	state := &invocationSupplyState{
		operation: operation,
		analysis:  analysis,
		command:   command,
		kind:      valueSupplyDefault,
	}
	supply := &DefaultSupply[C]{
		Config:    typed,
		Selection: cloneCommandSelection(selection),
		state: &defaultsState{
			supply: state,
		},
	}
	if err := provider.provide(ctx, supply); err != nil {
		return nil, true, err
	}
	if state.invalid {
		return nil, true, errInvalidInvocationSupply
	}
	return append([]valueSupply(nil), state.staged...), true, nil
}

func (provider typedFactProvider[C]) runFactProvider(ctx context.Context, operation *invocation, analysis *analysisState, command CommandID) ([]valueSupply, bool, error) {
	seed := operationDependencySeed(analysis)
	configuration, ok := operation.analysisConfigurationAt(analysis, command, seed.Destinations)
	if !ok {
		return nil, false, nil
	}
	typed, ok := configuration.(*C)
	if !ok {
		return nil, false, nil
	}
	state := &invocationSupplyState{
		operation: operation,
		analysis:  analysis,
		command:   command,
		kind:      valueSupplyFact,
	}
	supply := &FactSupply[C]{
		Config: typed,
		state: &factSupplyState{
			supply: state,
		},
	}
	if err := provider.provide(ctx, supply); err != nil {
		return nil, true, err
	}
	if state.invalid {
		return nil, true, errInvalidInvocationSupply
	}
	return append([]valueSupply(nil), state.staged...), true, nil
}

type invocationSupplies struct {
	defaults     []DefaultProvider
	facts        []FactProvider
	factDone     map[CommandID]struct{}
	defaultsDone bool
	order        uint64
}

func (runner *invocationSupplies) commit(analysis *analysisState, contributions []valueSupply) {
	for i := range contributions {
		runner.order++
		contributions[i].Order = runner.order
	}
	analysis.addValueSupplies(contributions)
}

// supplyFacts settles fact supply for newly selected scopes at most once.
// It is safe to call during grammar analysis before a scope's structural policy
// is evaluated; no default supply is run at this early frontier.
func (runner *invocationSupplies) supplyFacts(ctx context.Context, operation *invocation, analysis *analysisState, path []CommandID) (DiagnosticCode, error) {
	if runner == nil || operation == nil || analysis == nil {
		panic("cli: invalid invocation fact supply")
	}
	release := analysis.work.enter()
	defer release()

	for _, command := range path {
		if _, ok := runner.factDone[command]; ok {
			continue
		}
		for _, provider := range runner.facts {
			providerRunner, ok := provider.(factProviderRunner)
			if !ok {
				return DiagnosticFactSupplyFailure, errInvalidInvocationSupply
			}
			contributions, applicable, err := providerRunner.runFactProvider(ctx, operation, analysis, command)
			if err != nil {
				return DiagnosticFactSupplyFailure, err
			}
			if applicable {
				runner.commit(analysis, contributions)
			}
		}
		runner.factDone[command] = struct{}{}
	}
	return "", nil
}

func (runner *invocationSupplies) supplyDefaults(ctx context.Context, operation *invocation, analysis *analysisState, selection commandSelection) (DiagnosticCode, error) {
	if runner == nil || operation == nil || analysis == nil {
		panic("cli: invalid invocation default supply")
	}
	release := analysis.work.enter()
	defer release()

	if runner.defaultsDone {
		return "", nil
	}
	publicSelection := projectGrammarSelection(selection)
	for _, provider := range runner.defaults {
		providerRunner, ok := provider.(defaultProviderRunner)
		if !ok {
			return DiagnosticDefaultSupplyFailure, errInvalidInvocationSupply
		}
		for _, command := range selection.Path {
			contributions, applicable, err := providerRunner.runDefaultProvider(ctx, operation, analysis, command, publicSelection)
			if err != nil {
				return DiagnosticDefaultSupplyFailure, err
			}
			if applicable {
				runner.commit(analysis, contributions)
			}
		}
	}
	runner.defaultsDone = true
	return "", nil
}

func (runner *invocationSupplies) supplyAll(ctx context.Context, operation *invocation, analysis *analysisState, selection commandSelection) (DiagnosticCode, error) {
	if code, err := runner.supplyFacts(ctx, operation, analysis, selection.Path); err != nil {
		return code, err
	}
	return runner.supplyDefaults(ctx, operation, analysis, selection)
}

// valueSupplyKind identifies one invocation-owned candidate source.
// These values are private implementation vocabulary; public projection uses
// the existing ValueProducerKind classifications.
type valueSupplyKind uint8

const (
	valueSupplyInvalid valueSupplyKind = iota
	valueSupplyDefault
	valueSupplyFact
)

// valueSupply is one atomically accepted invocation provider output.
// Evaluate is operation-local and invoked at most once by value analysis.
type valueSupply struct {
	Kind        valueSupplyKind
	Destination ValueID
	Order       uint64
	Sensitive   bool
	Evaluate    func(context.Context) (any, error)
}

func (contribution valueSupply) valid() bool {
	return contribution.Kind != valueSupplyInvalid &&
		contribution.Destination.present() &&
		contribution.Order != 0 &&
		contribution.Evaluate != nil
}

func (analysis *analysisState) addValueSupplies(contributions []valueSupply) {
	if analysis == nil {
		panic("cli: nil analysis")
	}
	for _, contribution := range contributions {
		if !contribution.valid() {
			panic("cli: invalid invocation contribution")
		}
		if _, ok := analysis.effectiveModel.Destination(contribution.Destination); !ok {
			panic("cli: invocation contribution references unknown destination")
		}
	}
	analysis.valueSupplies = append(analysis.valueSupplies, contributions...)
}

func (analysis *analysisState) valueSupplySnapshot() []valueSupply {
	if analysis == nil || len(analysis.valueSupplies) == 0 {
		return nil
	}
	return append([]valueSupply(nil), analysis.valueSupplies...)
}

// shortcutDefaultContribution is one operation-local fallback contribution
// activated by a selected command shortcut. It retains shortcut identity and
// the canonical target used to resolve provider textual flag defaults without
// creating synthetic grammar input.
type shortcutDefaultContribution struct {
	Entry   CommandEntryID
	Target  CommandID
	Default shortcutDefault
	Order   uint64
}

// addShortcutDefaults commits the immutable fallback defaults owned by one
// selected shortcut. Fixed shortcut arguments are grammar input; these defaults
// are deliberately separate resolution candidates and therefore consume no
// input token.
func (operation *invocation) addShortcutDefaults(entry CommandEntryID, target CommandID, defaults []shortcutDefault) {
	if operation == nil || !entry.present() || !target.present() {
		panic("invocation: invalid shortcut default contribution")
	}
	if _, ok := operation.analysis.effectiveModel.CommandEntry(entry); !ok {
		panic("invocation: unknown shortcut default entry")
	}
	if _, ok := operation.analysis.effectiveModel.Command(target); !ok {
		panic("invocation: unknown shortcut default target")
	}
	for _, value := range defaults {
		if !value.Valid() {
			panic("invocation: invalid shortcut default")
		}
		shortcutDefaultContributionValue := shortcutDefaultContribution{
			Entry:   entry,
			Target:  target,
			Default: value,
			Order:   uint64(len(operation.analysis.shortcuts.defaults) + 1),
		}
		operation.analysis.shortcuts.defaults = append(operation.analysis.shortcuts.defaults, shortcutDefaultContributionValue)
	}
}

// EnableShortcutDefaults opens shortcut fallback contributions only after
// structural command/default selection and token ownership have settled.
// Shortcut defaults are resolution input; they must never influence grammar
// conditions or default-command selection.
func (operation *invocation) EnableShortcutDefaults() {
	if operation == nil {
		panic("invocation: nil operation")
	}
	operation.analysis.shortcuts.enabled = true
}

func (analysis *analysisState) shortcutDefaultsActive() bool {
	return analysis != nil && analysis.shortcuts.enabled
}

func (analysis *analysisState) shortcutDefaultSnapshot() []shortcutDefaultContribution {
	if analysis == nil || len(analysis.shortcuts.defaults) == 0 {
		return nil
	}
	return append([]shortcutDefaultContribution(nil), analysis.shortcuts.defaults...)
}

// FactResolver resolves a fact from read-only configuration and explicitly declared dependencies
// during invocation analysis; returning an error is resolution failure.
type FactResolver[C, T any] func(context.Context, *C, Dependencies) (T, error)

type factResolution[C, T any] struct {
	resolve      FactResolver[C, T]
	dependencies []Dependency
	options      []SourceOption
}

// resolvedFactDeclaration is the inferred owner for a resolved fact.
type resolvedFactDeclaration[C, T any] struct {
	state       *declarationSession
	destination Destination[C, T]
	resolution  factResolution[C, T]
}

func (declaration resolvedFactDeclaration[C, T]) dependencyTarget() (*destinationHandle, bool) {
	return declaration.destination.destinationHandle(), false
}

func (declaration resolvedFactDeclaration[C, T]) destinationFor(*C, *T) {}

func (declaration resolvedFactDeclaration[C, T]) destinationHandle() *destinationHandle {
	return declaration.destination.destinationHandle()
}

func (declaration resolvedFactDeclaration[C, T]) Destination() Destination[C, T] {
	return declaration.destination
}

func (declaration resolvedFactDeclaration[C, T]) OptionalDependency() Dependency {
	return destinationDependency{
		handle:   declaration.destination.destinationHandle(),
		optional: true,
	}
}

func (declaration resolvedFactDeclaration[C, T]) Sensitive() resolvedFactDeclaration[C, T] {
	setDestinationSensitive(declaration.destination.handle, true)
	return declaration
}

func (declaration resolvedFactDeclaration[C, T]) From(dependencies ...Dependency) resolvedFactDeclaration[C, T] {
	declaration.resolution.dependencies = append([]Dependency(nil), dependencies...)
	handle := declaration.destination.handle
	if handle == nil || handle.session == nil {
		return declaration
	}
	handle.session.use(func(draft *declarationDraft, _ commandRef) {
		computation, ok := eraseFactResolution(declaration.resolution)
		if ok {
			computation, ok = freezeAuthoringValueComputation(draft, computation)
		}
		identity, staged := handle.currentIdentity().(stagedDestinationIdentity)
		if !staged || !identity.ref.ownedBy(draft.identity) {
			panic("cli: issued destination reference changed owner")
		}
		record, resolved := draft.destinations.resolve(draft, identity.ref)
		if !resolved || record.handle != handle {
			panic("cli: issued destination draft cannot resolve")
		}
		record.computation = computation
		record.definitionComplete = ok
		record.valid = record.valid && ok
	})
	return declaration
}

// Facts declares required and resolved typed facts for the owning command configuration.
type Facts[C any] struct {
	Config *C

	session *declarationSession
	profile grammarProfileCaseRef
}

func (facts *Facts[C]) Require[T any](storage *T) Destination[C, T] {
	binding, bindingOK := bindConfigurationStorage(facts.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationFactRequired,
		storage:            storage,
		binding:            binding,
		definition:         typedAssignmentDestinationDefinition[T](destinationDeclarationFactRequired),
		definitionComplete: true,
		valid:              storage != nil && bindingOK,
	}
	state := stageDestination(
		facts.session,
		facts.profile,
		destinationValue,
	)
	return Destination[C, T]{handle: state}
}

func (facts *Facts[C]) Resolve[T any](storage *T, resolver FactResolver[C, T], options ...SourceOption) resolvedFactDeclaration[C, T] {
	binding, bindingOK := bindConfigurationStorage(facts.Config, storage)
	resolution := factResolution[C, T]{
		resolve: resolver,
		options: append([]SourceOption(nil), options...),
	}
	computation, computationOK := eraseFactResolution(resolution)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationFactResolved,
		storage:            storage,
		computation:        computation,
		binding:            binding,
		definition:         typedAssignmentDestinationDefinition[T](destinationDeclarationFactResolved),
		definitionComplete: computationOK,
		valid:              storage != nil && bindingOK && resolver != nil && computationOK,
	}
	state := stageDestination(
		facts.session,
		facts.profile,
		destinationValue,
	)
	result := resolvedFactDeclaration[C, T]{
		state:       facts.session,
		destination: Destination[C, T]{handle: state},
		resolution:  resolution,
	}

	return result
}

func eraseFactResolution[C, T any](resolution factResolution[C, T]) (authoringValueComputation, bool) {
	dependencies, ok := dependencyDescriptors(resolution.dependencies)
	source, sourceOK := producerSource(resolution.options)
	if !ok || !sourceOK || resolution.resolve == nil {
		return authoringValueComputation{}, false
	}
	result := authoringValueComputation{
		pending: dependencies,
		source:  source,
		evaluate: func(ctx context.Context, raw any, dependencies Dependencies) (any, bool, error) {
			config, ok := raw.(*C)
			if !ok || config == nil {
				panic("cli: invalid value computation configuration")
			}
			value, err := resolution.resolve(ctx, config, dependencies)
			if err != nil {
				return nil, false, err
			}
			return value, true, nil
		},
	}
	return result, true
}

type defaultRecord struct {
	meta      recordMeta[CommandDefaultID]
	parent    CommandID
	target    CommandID
	condition condition
}

func acceptDefaultRecord(id CommandDefaultID, parent, target CommandID, position declarationPosition) defaultRecord {
	mustCommand(parent)
	mustCommand(target)
	record := defaultRecord{
		meta:   acceptRecordMeta(id, position),
		parent: parent,
		target: target,
	}

	return record
}

func (record defaultRecord) Condition() (condition, bool) {
	return record.condition, record.condition.Valid()
}
