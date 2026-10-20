package cli

import (
	"context"
	"errors"
	"sync"
)

// analysisConfiguration assembles one fresh callback-local configuration from
// the semantic snapshot currently authorized for observation. The returned
// configuration is never retained as Run execution state.
func (operation *invocation) analysisConfiguration(command CommandID, values map[ValueID]destinationResolution) (any, bool) {
	if operation == nil {
		return nil, false
	}
	return operation.analysisConfigurationAt(operation.analysis, command, values)
}

func (operation *invocation) analysisConfigurationAt(analysis *analysisState, command CommandID, values map[ValueID]destinationResolution) (any, bool) {
	if operation == nil || analysis == nil || !command.present() {
		return nil, false
	}
	model := analysis.effectiveModel
	record, ok := model.Command(command)
	if !ok {
		return nil, false
	}
	factory, ok := record.Configuration()
	if !ok {
		return nil, false
	}
	configuration := factory.New()
	if configuration == nil {
		return nil, false
	}
	for _, destination := range model.DestinationsForCommand(command) {
		resolution, resolved := values[destination.meta.id]
		if !resolved || resolution.State != destinationStateResolved {
			continue
		}
		binding, ok := destination.Binding()
		if !ok {
			return nil, false
		}
		definition, ok := destination.Definition()
		if !ok {
			return nil, false
		}
		value, ok := definition.Snapshot(resolution.Value)
		if !ok || !binding.Write(configuration, value) {
			return nil, false
		}
	}
	return configuration, true
}

func (operation *invocation) analysisConfigurationForDependenciesAt(analysis *analysisState, command CommandID, dependencies []dependencyRequirement, values map[ValueID]destinationResolution) (any, bool) {
	if operation == nil || analysis == nil || !command.present() {
		return nil, false
	}
	model := analysis.effectiveModel
	record, ok := model.Command(command)
	if !ok {
		return nil, false
	}
	factory, ok := record.Configuration()
	if !ok {
		return nil, false
	}
	configuration := factory.New()
	if configuration == nil {
		return nil, false
	}
	for _, dependency := range dependencies {
		destination, ok := model.Destination(dependency.Destination())
		if !ok {
			return nil, false
		}
		if destination.command != command {
			continue
		}
		resolution, resolved := values[destination.meta.id]
		if !resolved || resolution.State != destinationStateResolved {
			continue
		}
		binding, ok := destination.Binding()
		if !ok {
			return nil, false
		}
		definition, ok := destination.Definition()
		if !ok {
			return nil, false
		}
		value, ok := definition.Snapshot(resolution.Value)
		if !ok || !binding.Write(configuration, value) {
			return nil, false
		}
	}
	return configuration, true
}

func (operation *invocation) valueLanguage(ctx context.Context) grammarValueLanguageDemandFunc {
	return operation.dynamicDomainValueLanguage(ctx)
}

func (operation *invocation) providerNamespaces() map[CommandID]providerNamespaceOutcome {
	return operation.analysis.providers.outcomes()
}

func (operation *invocation) dynamicDomainResults() map[int]dynamicDomainOutcome {
	return operation.analysis.domains.outcomes()
}

func (operation *invocation) namespace(ctx context.Context, demand grammarNamespaceDemand) grammarNamespaceMaterialization {
	return operation.materializeGrammarNamespace(ctx, demand)
}

func (operation *invocation) condition(ctx context.Context, demand grammarConditionDemand) conditionOutcome {
	return operation.evaluateGrammarCondition(ctx, demand)
}

func (operation *invocation) supplyDefaults(ctx context.Context, selection commandSelection) bool {
	analysis := operation.analysis
	if analysis.supplyErr != nil {
		return false
	}
	code, err := analysis.supplies.supplyDefaults(ctx, operation, analysis, selection)
	if err != nil {
		analysis.supplyCode, analysis.supplyErr = code, err
		return false
	}
	return true
}

func (operation *invocation) materializeDynamicDomains(ctx context.Context, grammar grammarResult, demanded []int) map[int]dynamicDomainOutcome {
	for _, index := range demanded {
		operation.materializeDynamicDomain(ctx, grammar, index)
	}
	return operation.analysis.domains.outcomes()
}

func (operation *invocation) supplyFacts(ctx context.Context, path []CommandID) bool {
	return operation.supplyFactsAt(ctx, operation.analysis, path)
}

func (operation *invocation) supplyFactsAt(ctx context.Context, analysis *analysisState, path []CommandID) bool {
	if analysis.supplyErr != nil {
		return false
	}
	code, err := analysis.supplies.supplyFacts(ctx, operation, analysis, path)
	if err != nil {
		analysis.supplyCode, analysis.supplyErr = code, err
		return false
	}
	return true
}

func (operation *invocation) materializeGrammarNamespace(ctx context.Context, demand grammarNamespaceDemand) grammarNamespaceMaterialization {
	return operation.materializeGrammarNamespaceAt(ctx, operation.analysis, demand)
}

func (operation *invocation) materializeGrammarNamespaceAt(ctx context.Context, analysis *analysisState, demand grammarNamespaceDemand) grammarNamespaceMaterialization {
	if !operation.supplyFactsAt(ctx, analysis, demand.Grammar.Selection.Path) {
		return grammarNamespaceMaterialization{}
	}
	materialized := operation.materializeProviderNamespaceAt(ctx, analysis, demand)
	return grammarNamespaceMaterialization{
		Changed:  materialized.changed,
		Complete: materialized.complete,
	}
}

func (operation *invocation) evaluateGrammarCondition(ctx context.Context, demand grammarConditionDemand) conditionOutcome {
	return operation.evaluateGrammarConditionAt(ctx, operation.analysis, demand)
}

func (operation *invocation) evaluateGrammarConditionAt(ctx context.Context, analysis *analysisState, demand grammarConditionDemand) conditionOutcome {
	if !operation.supplyFactsAt(ctx, analysis, demand.Grammar.Selection.Path) {
		return conditionOutcomeUnsupported
	}
	dependencies := demand.Policy.condition.Dependencies()
	requirements := make([]dependencyRequirement, len(dependencies))
	for index, destination := range dependencies {
		requirements[index] = acceptDependencyRequirement(destination, false)
	}
	values := valueAnalysis{
		Destinations: map[ValueID]destinationResolution{},
		Candidates:   map[valueCandidateRef]valueCandidateObservation{},
	}
	if len(requirements) != 0 {
		values = analyzeDependenciesAt(ctx, operation, analysis, demand.Grammar, requirements).Values
	}
	return evaluateCondition(demand.Policy.condition, values)
}

func (operation *invocation) evaluateDefaultCommandAt(ctx context.Context, analysis *analysisState, demand grammarDefaultDemand) conditionOutcome {
	if !operation.supplyFactsAt(ctx, analysis, demand.Grammar.Selection.Path) {
		return conditionOutcomeUnsupported
	}
	condition, conditional := demand.Default.Condition()
	if !conditional {
		return conditionOutcomeSatisfied
	}
	dependencies := condition.Dependencies()
	requirements := make([]dependencyRequirement, len(dependencies))
	for index, destination := range dependencies {
		requirements[index] = acceptDependencyRequirement(destination, false)
	}
	values := valueAnalysis{
		Destinations: map[ValueID]destinationResolution{},
		Candidates:   map[valueCandidateRef]valueCandidateObservation{},
	}
	if len(requirements) != 0 {
		values = analyzeDependenciesAt(ctx, operation, analysis, demand.Grammar, requirements).Values
	}
	return evaluateCondition(condition, values)
}

func (operation *invocation) finalizeSupplies(ctx context.Context, selection commandSelection) (DiagnosticCode, error) {
	analysis := operation.analysis
	if analysis.supplyErr != nil {
		return analysis.supplyCode, analysis.supplyErr
	}
	return analysis.supplies.supplyAll(ctx, operation, analysis, selection)
}

func (operation *invocation) namespaceFor(ctx context.Context) grammarNamespaceDemandFunc {
	return operation.namespaceForAt(ctx, operation.analysis)
}

func (operation *invocation) namespaceForAt(ctx context.Context, analysis *analysisState) grammarNamespaceDemandFunc {
	return func(demand grammarNamespaceDemand) grammarNamespaceMaterialization {
		return operation.materializeGrammarNamespaceAt(ctx, analysis, demand)
	}
}

func (operation *invocation) conditionFor(ctx context.Context) grammarConditionDemandFunc {
	return operation.conditionForAt(ctx, operation.analysis)
}

func (operation *invocation) conditionForAt(ctx context.Context, analysis *analysisState) grammarConditionDemandFunc {
	return func(demand grammarConditionDemand) conditionOutcome {
		return operation.evaluateGrammarConditionAt(ctx, analysis, demand)
	}
}

func (operation *invocation) defaultCommandFor(ctx context.Context) grammarDefaultDemandFunc {
	return operation.defaultCommandForAt(ctx, operation.analysis)
}

func (operation *invocation) defaultCommandForAt(ctx context.Context, analysis *analysisState) grammarDefaultDemandFunc {
	return func(demand grammarDefaultDemand) conditionOutcome {
		return operation.evaluateDefaultCommandAt(ctx, analysis, demand)
	}
}

func (operation *invocation) supplyFailure() (DiagnosticCode, error) {
	return operation.analysis.supplyCode, operation.analysis.supplyErr
}

type operationIdentityState struct {
	inputSource identityCounter
	inputOrigin identityCounter
}

type operationIdentities struct {
	state *operationIdentityState
}

func allocateOperationIdentities() operationIdentities {
	return operationIdentities{state: &operationIdentityState{}}
}

func (ids operationIdentities) mustState() *operationIdentityState {
	if ids.state == nil {
		panic("semantic: invalid operation identity scope")
	}
	return ids.state
}

func (ids operationIdentities) AllocateInputSourceID() InputSourceID {
	return InputSourceID{value: ids.mustState().inputSource.allocate()}
}

func (ids operationIdentities) AllocateInputOriginID() InputOriginID {
	return InputOriginID{value: ids.mustState().inputOrigin.allocate()}
}

type operationIDs struct {
	declarationIdentities
	operationIdentities
}

func allocateOperationIDs(baseline staticIdentityBaseline) operationIDs {
	return operationIDs{
		declarationIdentities: restoreDeclarationIdentities(baseline),
		operationIdentities:   allocateOperationIdentities(),
	}
}

func (ids operationIDs) ForkDeclarations() operationIDs {
	declaration := *ids.declarationIdentities.mustState()
	declaration.sealed = false
	return operationIDs{
		declarationIdentities: declarationIdentities{state: &declaration},
		operationIdentities:   ids.operationIdentities,
	}
}

func (ids operationIDs) CommitDeclarations(fork operationIDs) {
	target := ids.mutableState()
	source := fork.declarationIdentities.mustState()
	if source.sealed {
		panic("semantic: sealed declaration identity transaction")
	}
	*target = *source
}

var errInvalidAnalysisLimits = errors.New("cli: invalid analysis limits")

// normalizeAnalysisLimits resolves the one operation-wide public limit set.
// Zero chooses the package default, positive values are exact overrides, and
// any negative field rejects the operation before analysis begins.
func normalizeAnalysisLimits(input AnalysisLimits) (AnalysisLimits, error) {
	if input.Diagnostics < 0 ||
		input.ResponseFiles.Depth < 0 ||
		input.ResponseFiles.SourceBytes < 0 ||
		input.ResponseFiles.TotalSourceBytes < 0 ||
		input.ResponseFiles.TotalTokens < 0 ||
		input.ResponseFiles.TokenBytes < 0 ||
		input.ResponseFiles.IncludeReferences < 0 {
		return AnalysisLimits{}, errInvalidAnalysisLimits
	}
	if input.Diagnostics == 0 {
		input.Diagnostics = DefaultDiagnosticLimit
	}
	limits := analysisResponseFileLimits{
		Depth:             input.ResponseFiles.Depth,
		SourceBytes:       input.ResponseFiles.SourceBytes,
		TotalSourceBytes:  input.ResponseFiles.TotalSourceBytes,
		TotalTokens:       input.ResponseFiles.TotalTokens,
		TokenBytes:        input.ResponseFiles.TokenBytes,
		IncludeReferences: input.ResponseFiles.IncludeReferences,
	}
	response, _ := normalizeResponseFileLimits(limits)
	input.ResponseFiles = ResponseFileLimits(response)
	return input, nil
}

// AnalysisLimits bounds invocation analysis work. A zero field selects the
// package default, a positive field is an exact override, and any negative
// field makes the operation invalid before analysis begins.
type AnalysisLimits struct {
	Diagnostics   int
	ResponseFiles ResponseFileLimits
}

// DefaultDiagnosticLimit is the diagnostic bound used when
// AnalysisLimits.Diagnostics is zero. A report is marked truncated only when
// eligible diagnostic work is actually omitted by the bound.
const DefaultDiagnosticLimit = 20

type shortcutState struct {
	defaults []shortcutDefaultContribution
	enabled  bool
}

type grammarMemo struct {
	mu                       sync.RWMutex
	commandLexicalNamespaces map[lexicalNamespaceKey]*commandLexicalNamespace
	flagLexicalNamespaces    map[lexicalNamespaceKey]*flagLexicalNamespace
	flagFrontiers            map[flagFrontierCacheKey]*flagFrontierSnapshot
}

type analysisState struct {
	ids            operationIDs
	effectiveModel *effectiveModel
	callbacks      declarationCallbacks

	provenance invocationProvenance
	responses  *responseFileSession
	shortcuts  shortcutState

	args          []string
	valueSupplies []valueSupply
	supplies      invocationSupplies

	providers providerState
	domains   dynamicDomainState

	dependencyMu     sync.Mutex
	dependencyValues valueAnalysis
	conversions      conversionMemo
	grammar          grammarMemo
	work             analysisWorkGate

	pendingDestinationAcceptances []destinationAcceptance
	nextValueCandidate            int

	supplyCode DiagnosticCode
	supplyErr  error
}

// beginAnalysis establishes the one mutable semantic authority for an invocation.
func (operation *invocation) beginAnalysis() {
	if operation == nil {
		panic("invocation: nil operation")
	}
	if operation.analysis != nil {
		panic("invocation: analysis already started")
	}

	responses, err := prepareResponseFileSession(analysisResponseFileLimits(operation.input.limits.ResponseFiles))
	if err != nil {
		panic("cli: normalized response-file limits rejected")
	}
	analysis := &analysisState{
		ids:            allocateOperationIDs(operation.program.IdentityBaseline()),
		effectiveModel: prepareEffectiveModel(operation.program),
		callbacks:      cloneDeclarationCallbacks(operation.callbacks),
		responses:      responses,
		args:           append([]string(nil), operation.input.args...),
	}
	analysis.supplies = invocationSupplies{
		defaults: append([]DefaultProvider(nil), operation.input.defaults...),
		facts:    append([]FactProvider(nil), operation.input.facts...),
		factDone: make(map[CommandID]struct{}),
	}
	operation.analysis = analysis
	operation.initializeInputSources()
}

func (analysis *analysisState) argsSnapshot() []string {
	if analysis == nil {
		return nil
	}
	return append([]string(nil), analysis.args...)
}

func (operation *invocation) Args() []string {
	if operation == nil {
		return nil
	}
	return operation.analysis.argsSnapshot()
}

type analysisWorkGate struct {
	mu sync.Mutex

	active int
	frozen bool
}

func (gate *analysisWorkGate) enter() func() {
	if gate == nil {
		panic("cli: nil analysis work gate")
	}

	gate.mu.Lock()
	if gate.frozen {
		gate.mu.Unlock()
		panic("cli: semantic work entered frozen analysis snapshot")
	}
	gate.active++
	gate.mu.Unlock()

	released := false
	return func() {
		gate.mu.Lock()
		defer gate.mu.Unlock()

		if released || gate.active <= 0 {
			panic("cli: invalid analysis work release")
		}

		released = true
		gate.active--
	}
}

func (gate *analysisWorkGate) freezeForResponse() {
	if gate == nil {
		panic("cli: nil analysis work gate")
	}

	gate.mu.Lock()
	defer gate.mu.Unlock()

	if gate.frozen || gate.active != 0 {
		panic("cli: response fork requires quiescent analysis snapshot")
	}

	gate.frozen = true
}

func (gate *analysisWorkGate) quiescent() bool {
	if gate == nil {
		return false
	}

	gate.mu.Lock()
	defer gate.mu.Unlock()

	return gate.active == 0
}

// callbackReturnedCancellation reports whether a materialization callback's
// returned error itself carries cancellation provenance. Ambient context state
// is not causal evidence for classifying an independent callback failure.
func callbackReturnedCancellation(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if isOnlyCause(err, context.Canceled) || isOnlyCause(err, context.DeadlineExceeded) {
		return true
	}
	if ctx == nil {
		return false
	}
	cause := context.Cause(ctx)
	return cause != nil && isOnlyCause(err, cause)
}
