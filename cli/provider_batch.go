package cli

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"
)

var errInvalidProviderBatch = errors.New("cli: invalid provider batch")

type providerBatchRejectionError struct {
	cause error
}

func (err providerBatchRejectionError) Error() string { return "cli: invalid provider batch" }

func (err providerBatchRejectionError) Unwrap() error { return err.cause }

func (err providerBatchRejectionError) Is(target error) bool {
	return target == errInvalidProviderBatch
}

func rejectProviderBatch(cause error) error {
	if cause == nil {
		return errInvalidProviderBatch
	}
	failure := providerBatchRejectionError{
		cause: cause,
	}
	return failure
}

type providerCommandTarget interface {
	providerCommandTarget()
}

type stagedCommandTarget struct {
	command commandRef
}

func (stagedCommandTarget) providerCommandTarget() {}

type pathTarget struct {
	path CommandPath
}

func (pathTarget) providerCommandTarget() {}

func (pathTarget) batchCommandTarget() {}

type batchContributionDraft interface {
	batchContributionDraft()
}

type commandContributionDraft struct {
	position declarationPosition
	command  commandRef
	source   []SourceOption
}

func (commandContributionDraft) batchContributionDraft() {}

type aliasContributionDraft struct {
	position declarationPosition
	target   providerCommandTarget
	alias    CommandAlias
	source   []SourceOption
}

func (aliasContributionDraft) batchContributionDraft() {}

type shortcutDefaultDraft interface {
	shortcutDefaultDraft()
}

type destinationDefaultDraft struct {
	destination destinationIdentity
	value       any
	source      []SourceOption
}

func (destinationDefaultDraft) shortcutDefaultDraft() {}

type shortcutContributionDraft struct {
	position declarationPosition
	target   providerCommandTarget
	shortcut CommandShortcut
	source   []SourceOption
	defaults []shortcutDefaultDraft

	replacement shortcutRef
}

func (*shortcutContributionDraft) batchContributionDraft() {}

type shortcutRef uint32

func shortcutRefAt(index int) shortcutRef {
	if index < 0 || index >= math.MaxUint32 {
		panic("cli: invalid shortcut index")
	}
	return shortcutRef(index + 1)
}

func (ref shortcutRef) valid() bool { return ref != 0 }

func (ref shortcutRef) index() int {
	if !ref.valid() {
		panic("cli: invalid shortcut reference")
	}
	return int(ref - 1)
}

type batchRef[T any] struct {
	index uint32
}

func batchRefAt[T any](index int) batchRef[T] {
	if index < 0 || index >= math.MaxUint32 {
		panic("cli: invalid provider batch reference")
	}
	ref := batchRef[T]{
		index: uint32(index + 1),
	}
	return ref
}

func (ref batchRef[T]) valid() bool { return ref.index != 0 }

func (ref batchRef[T]) sliceIndex() int {
	if !ref.valid() {
		panic("cli: invalid provider batch reference")
	}
	return int(ref.index - 1)
}

type batchCommand struct {
	name        string
	aliases     []CommandAlias
	description Description
	examples    []Example
	hidden      bool
	defaults    commandDefaultDraft
	release     ReleaseLifecycle

	makeConfig func() any
	middleware []commandMiddleware
	handler    commandHandler
}

type batchDestinationDependency struct {
	destination batchDestinationTarget
	optional    bool
}

type batchValueComputation struct {
	dependencies []batchDestinationDependency
	source       semanticProducerSource
}

type batchCompletion struct {
	kind         completionDeclarationKind
	dependencies []batchDestinationDependency
	source       semanticProducerSource
	nativeKind   CompletionNativeKind
	nativeMode   CompletionNativeMode
	hasNative    bool
}

type batchDynamicValueDomain struct {
	commandIndex int
	dependencies []batchDestinationDependency
}

type batchValueDomain struct {
	entries []domainEntrySemantics
	dynamic *batchDynamicValueDomain
}

type batchDomainUse struct {
	domain batchValueDomainRef
	kind   ValueDomainUseKind
}

type batchDestination struct {
	sensitive   bool
	computation batchValueComputation
	binding     configurationBinding
	domainUses  []batchDomainUse
	definition  destinationDefinition
}

type batchFlag struct {
	destination batchDestinationRef
	producer    flagProducerKind

	completion          batchCompletion
	completionSet       bool
	tokenLanguage       any
	tokenCount          CountRange
	occurrences         CountRange
	occurrencesSet      bool
	valuesPerOccurrence CountRange
	valuesSet           bool
	delimiter           string
	delimiterSet        bool
	terminator          string
	terminatorSet       bool
	hyphenValues        HyphenValueMode
	hyphenValuesSet     bool
	placement           FlagPlacement
	placementSet        bool
	required            bool
	requiredSet         bool
	valueName           string
	valueNameSet        bool
	description         string
	descriptionSet      bool
	hidden              bool
	hiddenSet           bool
	localeSelector      bool
	localeSelectorSet   bool
	release             ReleaseLifecycle
	releaseSet          bool
}

type batchFlagSpelling struct {
	flag batchFlagRef
	kind flagSpellingKind
	text string
}

type batchArgumentTerm struct {
	form        batchArgumentFormRef
	destination batchDestinationRef

	kind               argumentTermKind
	name               string
	count              CountRange
	tokenLanguage      any
	literal            string
	completion         batchCompletion
	completionSet      bool
	optional           bool
	optionalSet        bool
	allocation         argumentAllocationPreference
	allocationSet      bool
	hyphenValues       HyphenValueMode
	hyphenValuesSet    bool
	description        string
	descriptionSet     bool
	hidden             bool
	hiddenSet          bool
	release            ReleaseLifecycle
	releaseSet         bool
	sourceSensitive    bool
	sourceSensitiveSet bool
	endOfOptions       bool
	endOfOptionsSet    bool
}

type batchProvider struct {
	dependencies []batchDestinationDependency
}

type batchValueDirective struct {
	kind             valueDirectiveKind
	destination      destinationIdentity
	value            any
	computation      batchValueComputation
	policies         []ResolutionPolicy
	name             string
	sensitive        bool
	environmentShape environmentInputShape
	delimiter        string
}

type batchFlagSpellingDirective struct {
	spelling      batchFlagSpellingRef
	kind          flagSpellingDirectiveKind
	forms         []FlagValueForm
	value         any
	sourceOptions []SourceOption
	lifecycle     ReleaseLifecycle
}

type batchArgumentForm struct{}

type batchArgumentFormSelection struct {
	form        batchArgumentFormRef
	destination destinationIdentity
	value       any
}

type batchConditionTargetKind uint8

const (
	batchConditionTargetInvalid batchConditionTargetKind = iota
	batchConditionTargetCommand
	batchConditionTargetFlag
	batchConditionTargetArgumentForm
	batchConditionTargetArgumentTerm
)

type batchConditionTarget struct {
	kind         batchConditionTargetKind
	command      batchCommandRef
	flag         batchFlagRef
	argumentForm batchArgumentFormRef
	argumentTerm batchArgumentTermRef
}

type batchConditionPolicy struct {
	target    batchConditionTarget
	effect    ConditionEffect
	condition Condition
}

type batchLifecycleTargetKind uint8

const (
	batchLifecycleTargetInvalid batchLifecycleTargetKind = iota
	batchLifecycleTargetEnvironment
	batchLifecycleTargetArgumentForm
	batchLifecycleTargetArgumentTerm
)

type batchLifecycleTarget struct {
	kind         batchLifecycleTargetKind
	environment  batchValueDirectiveRef
	argumentForm batchArgumentFormRef
	argumentTerm batchArgumentTermRef
}

type batchLifecycleDirective struct {
	target    batchLifecycleTarget
	lifecycle ReleaseLifecycle
}

type batchGrammarProfileSwitch struct {
	selector destinationIdentity
	equal    valueEquality
}

type batchGrammarProfileCase struct {
	switchRef   batchGrammarProfileSwitchRef
	value       any
	defaultCase bool
}

type batchRule struct {
	kind             RuleKind
	code             RuleCode
	activation       RuleCondition
	destinations     []destinationIdentity
	operands         []any
	count            CountRange
	pattern          string
	callbackKind     authoringRuleCallbackKind
	callback         authoringRuleCallback
	builtinPredicate authoringBuiltinRulePredicate
}

type batchRuleDirective struct {
	rule        batchRuleRef
	kind        ruleDirectiveKind
	destination destinationIdentity
	optional    bool
}

type batchAdvisory struct {
	kind      AdvisoryKind
	advisory  Advisory
	condition Condition
	callback  authoringAdvisoryCallback
}

type batchAdvisoryDirective struct {
	advisory    batchAdvisoryRef
	kind        advisoryDirectiveKind
	destination destinationIdentity
	optional    bool
}

type batchNoticeSuppression struct {
	condition Condition
	match     NoticeMatch
}

type batchShortcut struct {
	name        string
	description Description
	examples    []Example
	args        []string
	appendArgs  []string
	hidden      bool
	release     ReleaseLifecycle
}

type batchShortcutDirective struct {
	shortcut      batchShortcutRef
	kind          commandShortcutDirectiveKind
	destination   destinationIdentity
	value         any
	sourceOptions []SourceOption
	replacement   batchShortcutRef
}

type batchSignalClear struct{}

type batchSignalCounterLineage struct{}

type batchSignalEndpoint struct {
	binding configurationBinding
	adapt   signalEndpointAdapter
}

type batchSignalMatch struct {
	lineage         *batchSignalCounterLineage
	kind            signalSelectorKind
	signals         []Signal
	excludedSignals []Signal
	origins         []SignalOrigin
	excludedOrigins []SignalOrigin
	excludedSender  authoringSignalTarget
	hasSender       bool
	phases          []SignalPhase
	conditions      []signalConditionClause
	minimum         int
	maximum         int
	aggregation     SignalAggregation
	deferred        bool
	continueMatch   bool
	hint            string
}

type batchSignalAction struct {
	kind        signalActionKind
	name        string
	handler     signalHandlerCallback
	target      authoringSignalTarget
	signal      Signal
	signalValue destinationIdentity
	endpoint    batchSignalEndpointRef
	delayed     bool
	delay       time.Duration
	delayValue  destinationIdentity
	exitCode    int
}

type batchSignalActionPlan struct {
	actions []batchSignalAction
}

type batchSignalPolicy struct {
	match *batchSignalMatch
	plan  *batchSignalActionPlan
}

type batchHelpGroup struct {
	family helpGroupFamily
	title  string
	order  HelpGroupOrder
}

type batchReplacementTarget struct {
	kind  authoringReplacementKind
	index int
}

type batchReplacement struct {
	source batchReplacementTarget
	target batchReplacementTarget
}

type (
	batchCommandRef              = batchRef[batchCommand]
	batchDestinationRef          = batchRef[batchDestination]
	batchValueDirectiveRef       = batchRef[batchValueDirective]
	batchFlagRef                 = batchRef[batchFlag]
	batchFlagSpellingRef         = batchRef[batchFlagSpelling]
	batchArgumentFormRef         = batchRef[batchArgumentForm]
	batchArgumentTermRef         = batchRef[batchArgumentTerm]
	batchGrammarProfileSwitchRef = batchRef[batchGrammarProfileSwitch]
	batchRuleRef                 = batchRef[batchRule]
	batchAdvisoryRef             = batchRef[batchAdvisory]
	batchShortcutRef             = batchRef[batchShortcut]
	batchProviderRef             = batchRef[batchProvider]
	batchValueDomainRef          = batchRef[batchValueDomain]
	batchSignalEndpointRef       = batchRef[batchSignalEndpoint]
)

type batchCommandTarget interface {
	batchCommandTarget()
}

type localCommandTarget struct {
	command batchCommandRef
}

func (localCommandTarget) batchCommandTarget() {}

type batchDestinationTarget interface {
	batchDestinationTarget()
}

type localDestinationTarget struct {
	destination batchDestinationRef
}

func (localDestinationTarget) batchDestinationTarget() {}

func (localDestinationTarget) destinationIdentity() {}

func (acceptedDestinationIdentity) batchDestinationTarget() {}

type batchContribution interface {
	batchContribution()
}

type commandContribution struct {
	command batchCommandRef
	source  []SourceOption
}

func (commandContribution) batchContribution() {}

type aliasContribution struct {
	target batchCommandTarget
	alias  CommandAlias
	source []SourceOption
}

func (aliasContribution) batchContribution() {}

type batchShortcutDefault interface {
	batchShortcutDefault()
}

type destinationDefault struct {
	destination batchDestinationTarget
	value       any
	source      []SourceOption
}

func (destinationDefault) batchShortcutDefault() {}

type flagDefault struct {
	spelling string
	values   []string
	source   []SourceOption
}

func (flagDefault) shortcutDefaultDraft() {}

func (flagDefault) batchShortcutDefault() {}

type shortcutContribution struct {
	target   batchCommandTarget
	shortcut CommandShortcut
	source   []SourceOption
	defaults []batchShortcutDefault

	replacement shortcutRef
}

func (shortcutContribution) batchContribution() {}

type batchDeclarationOrderEntry struct {
	commandIndex          int
	profileIndex          int
	commandHelpGroupIndex int
	flagHelpGroupIndex    int
	kind                  declarationRecordKind
	familyIndex           int
	destination           batchDestinationRef
}

type batchDeclarations struct {
	order []batchDeclarationOrderEntry

	commands      []batchCommand
	destinations  []batchDestination
	flags         []batchFlag
	flagSpellings []batchFlagSpelling
	argumentTerms []batchArgumentTerm
	providers     []batchProvider
	valueDomains  []batchValueDomain

	valueDirectives        []batchValueDirective
	dialectDirectives      []dialectDirectiveDefinition
	flagSpellingDirectives []batchFlagSpellingDirective
	argumentForms          []batchArgumentForm
	argumentFormSelections []batchArgumentFormSelection
	conditionPolicies      []batchConditionPolicy
	lifecycleDirectives    []batchLifecycleDirective
	grammarProfileSwitches []batchGrammarProfileSwitch
	grammarProfileCases    []batchGrammarProfileCase
	rules                  []batchRule
	ruleDirectives         []batchRuleDirective
	advisories             []batchAdvisory
	advisoryDirectives     []batchAdvisoryDirective
	noticeSuppressions     []batchNoticeSuppression
	shortcuts              []batchShortcut
	shortcutDirectives     []batchShortcutDirective
	signalEndpoints        []batchSignalEndpoint
	signalClears           []batchSignalClear
	signalPolicies         []batchSignalPolicy
	helpGroups             []batchHelpGroup
	replacements           []batchReplacement
	accesses               []accessDefinition
}

type batchDestinationBinding struct {
	handle      *destinationHandle
	destination batchDestinationRef
}

type batchValueProducerKey struct {
	destination batchDestinationRef
	index       int
}

type batchCompletionCallbacks struct {
	flags     map[batchFlagRef]completionCallback
	arguments map[batchArgumentTermRef]completionCallback
}

type batchCallbacks struct {
	valueProducers   map[batchValueProducerKey]valueProducer
	completions      batchCompletionCallbacks
	valueDomains     []valueDomainProvider
	commandProviders map[batchProviderRef]commandProviderFunc
}

type providerBatchDraft struct {
	provider CommandProviderID
	parent   CommandID
	scope    identityScope
	release  ReleaseScheme

	declarations  declarationDraft
	contributions []batchContributionDraft

	localization *localizationSnapshot
}

type providerBatch struct {
	provider CommandProviderID
	parent   CommandID
	scope    identityScope

	declarations  batchDeclarations
	contributions []batchContribution
	callbacks     batchCallbacks
	bindings      []batchDestinationBinding

	localization *localizationSnapshot
}

func captureProviderBatch[C any](ctx context.Context, call providerCall, provider CommandProvider[C]) (providerBatch, error) {
	var result providerBatch
	if provider == nil {
		return result, rejectProviderBatch(errInvalidProviderBatch)
	}
	if !call.provider.present() || !call.parent.present() || !call.scope.valid() || call.scope.operation == nil {
		panic("cli: invalid provider call identity")
	}
	configuration, ok := call.config.(*C)
	if !ok || configuration == nil {
		panic("cli: invalid provider configuration")
	}

	draft := providerBatchDraft{
		provider: call.provider,
		parent:   call.parent,
		scope:    call.scope,
		release:  call.release,
	}
	session := draft.prepareSession()
	batch := CommandBatch[C]{
		Config:       configuration,
		Dependencies: call.dependencies,
		session:      session,
	}
	var callbackErr error
	cause := session.invoke(func() {
		callbackErr = provider(ctx, &batch)
	})
	if cause != nil {
		return result, cause
	}
	if callbackErr != nil {
		return result, callbackErr
	}
	return draft.freeze()
}

func resolveProviderCommandTarget(session *declarationSession, target CommandTarget) (providerCommandTarget, bool) {
	if session == nil || target == nil {
		return nil, false
	}
	resolved, ok := target.(interface {
		providerCommandTargetFor(session *declarationSession) (providerCommandTarget, bool)
	})
	if !ok {
		return nil, false
	}
	return resolved.providerCommandTargetFor(session)
}

func (path CommandPath) providerCommandTargetFor(_ *declarationSession) (providerCommandTarget, bool) {
	if len(path) == 0 || slices.Contains(path, "") {
		return nil, false
	}
	target := pathTarget{
		path: slices.Clone(path),
	}
	return target, true
}

func (declaration CommandDeclaration[C]) providerCommandTargetFor(session *declarationSession) (providerCommandTarget, bool) {
	if session == nil || declaration.session != session || !declaration.command.valid() {
		return nil, false
	}
	target := stagedCommandTarget{
		command: declaration.command,
	}
	return target, true
}

func providerBatchCommand[C, T any](batch *CommandBatch[C], command Command[T], source []SourceOption) CommandDeclaration[C] {
	if batch == nil || batch.session == nil {
		panicBuilderMisuse()
	}

	declarations, transaction, resume := batch.session.suspendProvider()
	defer resume()

	contribution := declarations.beginContribution()
	declaration := CommandDeclaration[C]{
		session: batch.session,
	}
	defer contribution.rollback()

	parent := acceptedCommandParent{
		command: transaction.parent,
	}
	child := stageCommand(declarations, parent, command)
	declaration.command = child

	if command.Declare != nil {
		childSession := batch.session.prepareChild(child)
		builder := commandBuilder(new(T), childSession, grammarProfileCaseRef{})
		if cause := childSession.invoke(func() {
			command.Declare(builder)
		}); cause != nil {
			abortBuilder(cause)
		}
	}
	if contribution.rejected() {
		declaration.command = commandRef{}
		return declaration
	}

	contribution.commit()
	staged := declarations.commands.mustResolve(declarations, child)
	contributionDraft := commandContributionDraft{
		position: staged.position,
		command:  child,
		source:   slices.Clone(source),
	}
	transaction.contributions = append(transaction.contributions, contributionDraft)
	return declaration
}

func providerBatchLocalization[C any](batch *CommandBatch[C], localization Localization) {
	if batch == nil || batch.session == nil {
		panicBuilderMisuse()
	}
	batch.session.useProvider(func(declarations *declarationDraft, transaction *providerBatchDraft, _ commandRef) {
		position := declarations.nextPosition()
		if transaction.localization != nil {
			declarations.reject(position, errInvalidProviderBatch)
			return
		}
		snapshot, err := snapshotLocalization(localization)
		if err != nil || !snapshot.Enabled() {
			declarations.reject(position, errInvalidProviderBatch)
			return
		}
		transaction.localization = &snapshot
	})
}

func providerBatchAlias[C any](batch *CommandBatch[C], target CommandTarget, alias CommandAlias, source []SourceOption) {
	if batch == nil || batch.session == nil {
		panicBuilderMisuse()
	}
	batch.session.useProvider(func(declarations *declarationDraft, transaction *providerBatchDraft, _ commandRef) {
		position := declarations.nextPosition()
		resolved, ok := resolveProviderCommandTarget(batch.session, target)
		if !ok {
			declarations.reject(position, errInvalidProviderBatch)
			return
		}
		contribution := aliasContributionDraft{
			position: position,
			target:   resolved,
			alias:    alias,
			source:   slices.Clone(source),
		}
		if alias.Name == "" {
			declarations.reject(position, errInvalidProviderBatch)
		}
		transaction.contributions = append(transaction.contributions, contribution)
	})
}

func providerBatchShortcut[C any](batch *CommandBatch[C], target CommandTarget, shortcut CommandShortcut, source []SourceOption) CommandShortcutBinding {
	if batch == nil || batch.session == nil {
		panicBuilderMisuse()
	}
	binding := CommandShortcutBinding{
		session: batch.session,
	}
	batch.session.useProvider(func(declarations *declarationDraft, transaction *providerBatchDraft, _ commandRef) {
		position := declarations.nextPosition()
		resolved, ok := resolveProviderCommandTarget(batch.session, target)
		if !ok {
			declarations.reject(position, errInvalidProviderBatch)
			return
		}
		contribution := &shortcutContributionDraft{
			position: position,
			target:   resolved,
			shortcut: snapshotProviderShortcut(shortcut),
			source:   slices.Clone(source),
		}
		if shortcut.Name == "" {
			declarations.reject(position, errInvalidProviderBatch)
		}
		binding.shortcut = shortcutRefAt(len(transaction.contributions))
		transaction.contributions = append(transaction.contributions, contribution)
	})
	return binding
}

func providerShortcutTypedDefault[O, T any](binding CommandShortcutBinding, destination destinationLike[O, T], value T, source []SourceOption) CommandShortcutBinding {
	if binding.session == nil {
		panicBuilderMisuse()
	}
	binding.session.useProvider(func(declarations *declarationDraft, transaction *providerBatchDraft, _ commandRef) {
		if !binding.shortcut.valid() {
			return
		}
		shortcut, ok := providerShortcut(transaction, binding.shortcut)
		if !ok {
			panic("cli: issued shortcut reference cannot resolve")
		}
		if destination == nil {
			declarations.reject(shortcut.position, errInvalidProviderBatch)
			return
		}
		handle := destination.destinationHandle()
		identity, ok := captureAuthoringDestinationIdentity(declarations, handle)
		if !ok {
			declarations.reject(shortcut.position, errInvalidProviderBatch)
			return
		}
		detached, ok := handle.snapshotValue(value)
		if !ok {
			declarations.reject(shortcut.position, errInvalidProviderBatch)
			return
		}
		destinationDefaultDraftValue := destinationDefaultDraft{
			destination: identity,
			value:       detached,
			source:      slices.Clone(source),
		}
		shortcut.defaults = append(shortcut.defaults, destinationDefaultDraftValue)
	})
	return binding
}

func providerShortcutFlagDefault(binding CommandShortcutBinding, spelling string, values []string, source []SourceOption) CommandShortcutBinding {
	if binding.session == nil {
		panicBuilderMisuse()
	}
	binding.session.useProvider(func(declarations *declarationDraft, transaction *providerBatchDraft, _ commandRef) {
		if !binding.shortcut.valid() {
			return
		}
		shortcut, ok := providerShortcut(transaction, binding.shortcut)
		if !ok {
			panic("cli: issued shortcut reference cannot resolve")
		}
		if spelling == "" || len(values) == 0 {
			declarations.reject(shortcut.position, errInvalidProviderBatch)
			return
		}
		flagDefaultValue := flagDefault{
			spelling: spelling,
			values:   slices.Clone(values),
			source:   slices.Clone(source),
		}
		shortcut.defaults = append(shortcut.defaults, flagDefaultValue)
	})
	return binding
}

func providerShortcut(batch *providerBatchDraft, ref shortcutRef) (*shortcutContributionDraft, bool) {
	if batch == nil || !ref.valid() {
		return nil, false
	}
	index := ref.index()
	if index >= len(batch.contributions) {
		return nil, false
	}
	shortcut, ok := batch.contributions[index].(*shortcutContributionDraft)
	return shortcut, ok && shortcut != nil
}

func providerShortcutReplacedBy(binding, target CommandShortcutBinding) {
	if binding.session == nil {
		panicBuilderMisuse()
	}
	binding.session.useProvider(func(declarations *declarationDraft, transaction *providerBatchDraft, _ commandRef) {
		if !binding.shortcut.valid() {
			return
		}
		shortcut, ok := providerShortcut(transaction, binding.shortcut)
		if !ok {
			panic("cli: issued shortcut reference cannot resolve")
		}
		if target.session != binding.session || !target.shortcut.valid() {
			declarations.reject(shortcut.position, errInvalidProviderBatch)
			return
		}
		if _, ok := providerShortcut(transaction, target.shortcut); !ok {
			declarations.reject(shortcut.position, errInvalidProviderBatch)
			return
		}
		shortcut.replacement = target.shortcut
	})
}

func snapshotProviderShortcut(shortcut CommandShortcut) CommandShortcut {
	shortcut.Examples = slices.Clone(shortcut.Examples)
	shortcut.Args = slices.Clone(shortcut.Args)
	shortcut.AppendArgs = slices.Clone(shortcut.AppendArgs)
	return shortcut
}

func (declarations *batchDeclarations) appendRecord(record declarationPlanRecord, destination batchDestinationRef) {
	if declarations == nil || record.familyIndex < 0 {
		panic("cli: invalid provider batch declaration")
	}
	entry := batchDeclarationOrderEntry{
		commandIndex:          record.commandIndex,
		profileIndex:          record.profileIndex,
		commandHelpGroupIndex: record.commandHelpGroupIndex,
		flagHelpGroupIndex:    record.flagHelpGroupIndex,
		kind:                  record.kind,
		familyIndex:           record.familyIndex,
		destination:           destination,
	}
	if entry.destination.valid() && entry.destination.sliceIndex() >= len(declarations.destinations) {
		panic("cli: provider destination has invalid batch coordinate")
	}
	declarations.order = append(declarations.order, entry)
}

func (draft *providerBatchDraft) freezeCallbacks(input declarationPlan, domainSources []*valueDomainState) batchCallbacks {
	if draft == nil {
		panic("cli: nil provider batch draft")
	}

	callbacks := batchCallbacks{
		valueProducers: make(map[batchValueProducerKey]valueProducer),
		completions: batchCompletionCallbacks{
			flags:     make(map[batchFlagRef]completionCallback),
			arguments: make(map[batchArgumentTermRef]completionCallback),
		},
		commandProviders: make(map[batchProviderRef]commandProviderFunc),
	}
	producerOrdinals := make(map[batchDestinationRef]int)

	for _, record := range input.records {
		switch record.kind {
		case declarationRecordDestination:
			destinationRef := batchRefAt[batchDestination](record.familyIndex)
			destination := draft.declarations.destinations.entries[record.familyIndex].value
			switch destination.kind {
			case destinationDeclarationDerived, destinationDeclarationOptionalDerived, destinationDeclarationFactResolved:
				ordinal := producerOrdinals[destinationRef]
				callbacks.valueProducers[batchValueProducerKey{
					destination: destinationRef,
					index:       ordinal,
				}] = freezeBatchValueProducer(destination.computation.evaluate)
				producerOrdinals[destinationRef] = ordinal + 1
			case destinationDeclarationInvalid, destinationDeclarationValue, destinationDeclarationSequence, destinationDeclarationKeyed, destinationDeclarationOpaque, destinationDeclarationFactRequired:
			}

		case declarationRecordFlag:
			flag := draft.declarations.flags.entries[record.familyIndex].value
			if flag.completionSet && flag.completion.callback.present() {
				callbacks.completions.flags[batchRefAt[batchFlag](record.familyIndex)] = flag.completion.callback
			}

		case declarationRecordArgumentTerm:
			argument := draft.declarations.argumentTerms.entries[record.familyIndex].value
			if argument.completionSet && argument.completion.callback.present() {
				callbacks.completions.arguments[batchRefAt[batchArgumentTerm](record.familyIndex)] = argument.completion.callback
			}

		case declarationRecordValueDirective:
			directive := draft.declarations.valueDirectives.entries[record.familyIndex].value
			destination, ok := providerBatchDestinationRef(draft, directive.destination)
			switch directive.kind {
			case valueDirectiveDefault:
				if !ok {
					panic("cli: provider default destination is not batch-local")
				}
				producerOrdinals[destination]++
			case valueDirectiveComputedDefault, valueDirectiveOptionalComputedDefault:
				if !ok {
					panic("cli: provider computed default destination is not batch-local")
				}
				ordinal := producerOrdinals[destination]
				callbacks.valueProducers[batchValueProducerKey{
					destination: destination,
					index:       ordinal,
				}] = freezeBatchValueProducer(directive.computation.evaluate)
				producerOrdinals[destination] = ordinal + 1
			case valueDirectiveInvalid, valueDirectiveResolution, valueDirectiveEnvironment, valueDirectiveEnvironmentPrefix:
			}

		case declarationRecordProvider:
			provider := draft.declarations.providers.entries[record.familyIndex].value
			if provider.provide == nil {
				panic("cli: provider declaration has no callback")
			}
			callbacks.commandProviders[batchRefAt[batchProvider](record.familyIndex)] = provider.provide

		case declarationRecordInvalid, declarationRecordDialectDirective, declarationRecordFlagSpelling, declarationRecordFlagSpellingDirective, declarationRecordArgumentForm, declarationRecordArgumentFormSelection, declarationRecordCommand, declarationRecordConditionPolicy, declarationRecordLifecycleDirective, declarationRecordGrammarProfileSwitch, declarationRecordGrammarProfileCase, declarationRecordRule, declarationRecordRuleDirective, declarationRecordAdvisory, declarationRecordAdvisoryDirective, declarationRecordNoticeSuppression, declarationRecordShortcut, declarationRecordShortcutDirective, declarationRecordSignalClear, declarationRecordSignalPolicy, declarationRecordHelpGroup, declarationRecordReplacement, declarationRecordAccess:
		}
	}

	callbacks.valueDomains = make([]valueDomainProvider, len(domainSources))
	for index, state := range domainSources {
		callbacks.valueDomains[index] = freezeBatchValueDomainProvider(state)
	}

	return callbacks
}

func providerBatchDestinationRef(draft *providerBatchDraft, identity destinationIdentity) (batchDestinationRef, bool) {
	if draft == nil {
		return batchDestinationRef{}, false
	}
	staged, ok := identity.(stagedDestinationIdentity)
	if !ok {
		return batchDestinationRef{}, false
	}
	if _, ok := draft.declarations.destinations.resolve(&draft.declarations, staged.ref); !ok {
		return batchDestinationRef{}, false
	}
	return batchRefAt[batchDestination](staged.ref.index), true
}

func freezeBatchValueProducer(evaluate authoringValueEvaluator) valueProducer {
	if evaluate == nil {
		panic("cli: provider value producer has no callback")
	}
	return func(ctx context.Context, configuration any, dependencies Dependencies) (valueEvaluation, error) {
		value, present, err := evaluate(ctx, configuration, dependencies)
		if err != nil {
			return valueEvaluation{}, err
		}
		if !present {
			return valueOmitted(), nil
		}
		return valuePresent(value), nil
	}
}

func freezeBatchValueDomainProvider(state *valueDomainState) valueDomainProvider {
	if state == nil {
		panic("cli: provider value domain has no state")
	}
	if state.dynamic == nil {
		return nil
	}
	evaluate := state.dynamic.evaluate
	if evaluate == nil {
		panic("cli: provider dynamic value domain has no callback")
	}
	return func(ctx context.Context, configuration, rawDependencies, output any) error {
		dependencies, ok := rawDependencies.(Dependencies)
		if !ok {
			panic("cli: invalid provider value-domain dependencies")
		}
		evaluation, ok := output.(*dynamicDomainEvaluation)
		if !ok || evaluation == nil {
			panic("cli: invalid provider value-domain output")
		}
		captured, err := evaluate(ctx, configuration, dependencies)
		if err != nil {
			return err
		}
		evaluation.state = captured
		return nil
	}
}
