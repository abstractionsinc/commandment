package cli

import (
	"context"
	"errors"
	"slices"
)

type declarationRecordKind uint8

const (
	declarationRecordInvalid declarationRecordKind = iota
	declarationRecordDestination
	declarationRecordValueDirective
	declarationRecordDialectDirective
	declarationRecordFlag
	declarationRecordFlagSpelling
	declarationRecordFlagSpellingDirective
	declarationRecordArgumentForm
	declarationRecordArgumentTerm
	declarationRecordArgumentFormSelection
	declarationRecordCommand
	declarationRecordConditionPolicy
	declarationRecordLifecycleDirective
	declarationRecordGrammarProfileSwitch
	declarationRecordGrammarProfileCase
	declarationRecordRule
	declarationRecordRuleDirective
	declarationRecordAdvisory
	declarationRecordAdvisoryDirective
	declarationRecordNoticeSuppression
	declarationRecordShortcut
	declarationRecordShortcutDirective
	declarationRecordProvider
	declarationRecordSignalClear
	declarationRecordSignalPolicy
	declarationRecordHelpGroup
	declarationRecordReplacement
	declarationRecordAccess
)

var errInvalidMiddlewareDeclaration = errors.New("cli: invalid middleware declaration")

type draftIdentity struct {
	_ byte
}

type draftRef[T any] struct {
	draft      *draftIdentity
	index      int
	generation uint64
}

func (ref draftRef[T]) valid() bool {
	return ref.draft != nil && ref.index >= 0 && ref.generation != 0
}

func (ref draftRef[T]) issued() bool {
	return ref.valid()
}

func (ref draftRef[T]) ownedBy(owner *draftIdentity) bool {
	return owner != nil && ref.valid() && ref.draft == owner
}

type draftEntry[T any] struct {
	generation uint64
	value      T
}

type declarationFamily[T any] struct {
	entries []draftEntry[T]
}

func (family *declarationFamily[T]) append(draft *declarationDraft, value T) draftRef[T] {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	if draft.identity == nil {
		draft.identity = &draftIdentity{}
	}
	draft.generation++
	if draft.generation == 0 {
		panic("cli: declaration reference generation exhausted")
	}
	ref := draftRef[T]{
		draft:      draft.identity,
		index:      len(family.entries),
		generation: draft.generation,
	}
	family.entries = append(family.entries, draftEntry[T]{
		generation: ref.generation,
		value:      value,
	})
	return ref
}

func (family *declarationFamily[T]) resolve(draft *declarationDraft, ref draftRef[T]) (*T, bool) {
	if draft == nil || !ref.valid() || ref.draft != draft.identity || ref.index >= len(family.entries) {
		return nil, false
	}
	entry := &family.entries[ref.index]
	if entry.generation != ref.generation {
		return nil, false
	}
	return &entry.value, true
}

func (family *declarationFamily[T]) mustResolve(draft *declarationDraft, ref draftRef[T]) *T {
	value, ok := family.resolve(draft, ref)
	if !ok {
		panic("cli: issued declaration reference cannot resolve")
	}
	return value
}

type (
	commandRef      = draftRef[commandDraft]
	destinationRef  = draftRef[destinationDraft]
	flagRef         = draftRef[flagDraft]
	flagSpellingRef = draftRef[flagSpellingDraft]
)

type declarationDraft struct {
	identity   *draftIdentity
	scope      identityScope
	position   declarationPosition
	generation uint64

	root commandRef

	commands      declarationFamily[commandDraft]
	destinations  declarationFamily[destinationDraft]
	flags         declarationFamily[flagDraft]
	flagSpellings declarationFamily[flagSpellingDraft]

	valueDirectives        declarationFamily[valueDirectiveDraft]
	dialectDirectives      declarationFamily[dialectDirectiveDraft]
	flagSpellingDirectives declarationFamily[flagSpellingDirectiveDraft]
	argumentForms          declarationFamily[argumentFormDraft]
	argumentTerms          declarationFamily[argumentTermDraft]
	argumentFormSelections declarationFamily[argumentFormSelectionDraft]
	conditionPolicies      declarationFamily[conditionPolicyDraft]
	lifecycleDirectives    declarationFamily[lifecycleDirectiveDraft]
	grammarProfileSwitches declarationFamily[grammarProfileSwitchDraft]
	grammarProfileCases    declarationFamily[grammarProfileCaseDraft]
	rules                  declarationFamily[ruleDraft]
	ruleDirectives         declarationFamily[ruleDirectiveDraft]
	advisories             declarationFamily[advisoryDraft]
	advisoryDirectives     declarationFamily[advisoryDirectiveDraft]
	noticeSuppressions     declarationFamily[noticeSuppressionDraft]
	shortcuts              declarationFamily[shortcutDraft]
	shortcutDirectives     declarationFamily[shortcutDirectiveDraft]
	providers              declarationFamily[commandProviderDraft]
	signalEndpoints        declarationFamily[signalEndpointDraft]
	signalClears           declarationFamily[signalClearDraft]
	signalPolicies         declarationFamily[signalPolicyDraft]
	helpGroups             declarationFamily[helpGroupDraft]
	replacements           declarationFamily[replacementDraft]
	accesses               declarationFamily[accessDraft]

	actions         authoringIntrinsicActions
	shellCompletion shellCompletionDraft

	rejections []declarationRejection
}

type declarationRejection struct {
	position declarationPosition
	cause    error
}

type declarationRejectionError struct {
	rejections []declarationRejection
}

func (err declarationRejectionError) Error() string { return "cli: declarations rejected" }

func (err declarationRejectionError) Unwrap() []error {
	causes := make([]error, len(err.rejections))
	for index, rejection := range err.rejections {
		causes[index] = rejection.cause
	}
	return causes
}

func (draft *declarationDraft) nextPosition() declarationPosition {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	draft.position++
	if draft.position == 0 {
		panic("cli: declaration position exhausted")
	}
	return draft.position
}

func (draft *declarationDraft) reject(position declarationPosition, cause error) {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	if cause == nil {
		panic("cli: nil declaration rejection")
	}
	draft.rejections = append(draft.rejections, declarationRejection{
		position: position,
		cause:    cause,
	})
}

func (draft *declarationDraft) rejectionError() error {
	if draft == nil || len(draft.rejections) == 0 {
		return nil
	}
	failure := declarationRejectionError{
		rejections: slices.Clone(draft.rejections),
	}
	return failure
}

type declarationContribution struct {
	draft          *declarationDraft
	snapshot       declarationDraft
	rejectionCount int
	closed         bool
}

func (draft *declarationDraft) beginContribution() declarationContribution {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	result := declarationContribution{
		draft:          draft,
		snapshot:       *draft,
		rejectionCount: len(draft.rejections),
	}

	return result
}

func (contribution *declarationContribution) rejected() bool {
	return contribution == nil || contribution.draft == nil || len(contribution.draft.rejections) != contribution.rejectionCount
}

func (contribution *declarationContribution) commit() {
	if contribution == nil {
		panic("cli: nil declaration contribution")
	}
	if contribution.closed {
		panic("cli: declaration contribution already closed")
	}
	contribution.closed = true
}

func (contribution *declarationContribution) rollback() {
	if contribution == nil || contribution.closed || contribution.draft == nil {
		return
	}
	draft := contribution.draft
	position := draft.position
	generation := draft.generation
	rejections := draft.rejections
	*draft = contribution.snapshot
	draft.position = position
	draft.generation = generation
	draft.rejections = rejections
	contribution.closed = true
}

type commandParent interface {
	commandParent()
}

type stagedCommandParent struct {
	command commandRef
}

func (stagedCommandParent) commandParent() {}

type acceptedCommandParent struct {
	command CommandID
}

func (acceptedCommandParent) commandParent() {}

type commandDraft struct {
	position  declarationPosition
	parent    commandParent
	profile   grammarProfileCaseRef
	helpGroup helpGroupRef

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
	intrinsic  bool
}

type destinationDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef
	handle   *destinationHandle

	kind               destinationDeclarationKind
	storage            any
	language           any
	sensitive          bool
	computation        authoringValueComputation
	binding            configurationBinding
	domainUses         []authoringDomainUse
	definition         destinationDefinition
	definitionComplete bool
	intrinsic          bool
	valid              bool
}

type flagDraft struct {
	position    declarationPosition
	command     commandRef
	profile     grammarProfileCaseRef
	helpGroup   helpGroupRef
	destination destinationRef

	producer            flagProducerKind
	completion          authoringCompletion
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
	valid               bool
}

type flagSpellingDraft struct {
	position declarationPosition
	flag     flagRef

	kind  flagSpellingKind
	text  string
	valid bool
}

func compileMiddleware[C any](middleware []Middleware[C]) ([]commandMiddleware, error) {
	result := make([]commandMiddleware, len(middleware))
	for index, middleware := range middleware {
		if middleware == nil {
			return nil, errInvalidMiddlewareDeclaration
		}
		typed := middleware
		result[index] = func(ctx context.Context, configuration any, next func(context.Context) error) error {
			config, ok := configuration.(*C)
			if !ok || config == nil {
				panic("cli: invalid middleware configuration")
			}
			return typed(ctx, config, next)
		}
	}
	return result, nil
}

func compileHandler[C any](handler Handler[C]) commandHandler {
	if handler == nil {
		return nil
	}
	typed := handler
	return func(ctx context.Context, configuration any) error {
		config, ok := configuration.(*C)
		if !ok || config == nil {
			panic("cli: invalid command handler configuration")
		}
		return typed(ctx, config)
	}
}

func stageRoot[C any](draft *applicationDraft, run Handler[C]) commandRef {
	if draft == nil {
		panic("cli: nil application draft")
	}
	handler := compileHandler(run)
	position := draft.declarations.nextPosition()
	root := commandDraft{
		position: position,
		name:     draft.name,
		makeConfig: func() any {
			return new(C)
		},
		middleware: slices.Clone(draft.middleware),
		handler:    handler,
	}
	ref := draft.declarations.commands.append(&draft.declarations, root)
	draft.declarations.root = ref
	return ref
}

func stageCommand[C any](draft *declarationDraft, parent commandParent, command Command[C]) commandRef {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	position := draft.nextPosition()
	middleware, err := compileMiddleware(command.Middleware)
	if err != nil {
		draft.reject(position, err)
	}
	handler := compileHandler(command.Run)
	staged := commandDraft{
		position:    position,
		parent:      parent,
		name:        command.Name,
		aliases:     slices.Clone(command.Aliases),
		description: command.Description,
		examples:    slices.Clone(command.Examples),
		hidden:      command.Hidden,
		release:     command.Release,
		makeConfig: func() any {
			return new(C)
		},
		middleware: middleware,
		handler:    handler,
	}
	ref := draft.commands.append(draft, staged)
	if command.Name == "" {
		draft.reject(position, errInvalidAuthoringTransaction)
	}
	return ref
}

// The declaration draft stores callback-time semantics directly. Each record
// has one mutable owner: its declarationFamily entry. Immutable application
// acceptance happens only in applicationDraft.commit.

type valueDirectiveDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	kind             valueDirectiveKind
	destination      destinationIdentity
	value            any
	computation      authoringValueComputation
	policies         []ResolutionPolicy
	name             string
	sensitive        bool
	environmentShape environmentInputShape
	delimiter        string
	valid            bool
}

type dialectDirectiveDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	kind          dialectDirectiveKind
	flagOrder     FlagOrder
	clustering    ShortFlagClustering
	matching      NameMatching
	normalization NameNormalization
	dashless      DashlessFlagMode
	tokens        []string
	enabled       bool
	placement     FlagPlacement
	valid         bool
}

type flagSpellingDirectiveDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef
	spelling flagSpellingRef

	kind          flagSpellingDirectiveKind
	forms         []FlagValueForm
	value         any
	sourceOptions []SourceOption
	lifecycle     ReleaseLifecycle
	valid         bool
}

type argumentFormDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef
	valid    bool
}

type argumentTermDraft struct {
	position    declarationPosition
	command     commandRef
	profile     grammarProfileCaseRef
	form        argumentFormDraftRef
	destination destinationRef

	kind               argumentTermKind
	name               string
	count              CountRange
	tokenLanguage      any
	literal            string
	completion         authoringCompletion
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
	valid              bool
}

type argumentFormSelectionDraft struct {
	position    declarationPosition
	command     commandRef
	profile     grammarProfileCaseRef
	form        argumentFormDraftRef
	destination destinationIdentity
	value       any
	valid       bool
}

type conditionPolicyDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	target    authoringConditionTarget
	effect    ConditionEffect
	condition Condition
	valid     bool
}

type lifecycleDirectiveDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	target    authoringLifecycleTarget
	lifecycle ReleaseLifecycle
	valid     bool
}

type grammarProfileSwitchDraft struct {
	position declarationPosition
	command  commandRef

	selector destinationIdentity
	equal    valueEquality
	valid    bool
}

type grammarProfileCaseDraft struct {
	position  declarationPosition
	command   commandRef
	switchRef grammarProfileSwitchRef

	value       any
	defaultCase bool
	valid       bool
}

type ruleDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

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
	valid            bool
}

type ruleDirectiveDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef
	rule     ruleDraftRef

	kind        ruleDirectiveKind
	destination destinationIdentity
	optional    bool
	valid       bool
}

type advisoryDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	kind      AdvisoryKind
	advisory  Advisory
	condition Condition
	callback  authoringAdvisoryCallback
	valid     bool
}

type advisoryDirectiveDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef
	advisory advisoryDraftRef

	kind        advisoryDirectiveKind
	destination destinationIdentity
	optional    bool
	valid       bool
}

type noticeSuppressionDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	condition Condition
	match     NoticeMatch
	valid     bool
}

type shortcutDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	name        string
	description Description
	examples    []Example
	args        []string
	appendArgs  []string
	hidden      bool
	release     ReleaseLifecycle
	valid       bool
}

type shortcutDirectiveDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef
	shortcut shortcutDraftRef

	kind          commandShortcutDirectiveKind
	destination   destinationIdentity
	value         any
	sourceOptions []SourceOption
	replacement   shortcutDraftRef
	valid         bool
}

type commandProviderDraft struct {
	position  declarationPosition
	command   commandRef
	profile   grammarProfileCaseRef
	helpGroup helpGroupRef

	dependencies []authoringDestinationDependency
	provide      commandProviderFunc
	valid        bool
}

type signalEndpointDraft struct {
	binding configurationBinding
	adapt   signalEndpointAdapter
	valid   bool
}

type signalClearDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef
	valid    bool
}

type signalPolicyDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	match *authoringSignalMatch
	plan  *signalActionPlanState
	valid bool
}

type helpGroupDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	family helpGroupFamily
	title  string
	order  HelpGroupOrder
	valid  bool
}

type replacementDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	source authoringReplacementTarget
	target authoringReplacementTarget
	valid  bool
}

type accessDraft struct {
	position declarationPosition
	command  commandRef
	profile  grammarProfileCaseRef

	kind    accessDeclarationKind
	source  destinationIdentity
	target  configurationBinding
	prepare executionAccessPrepare
	valid   bool
}

type (
	valueDirectiveRef       = draftRef[valueDirectiveDraft]
	argumentFormDraftRef    = draftRef[argumentFormDraft]
	argumentTermDraftRef    = draftRef[argumentTermDraft]
	grammarProfileSwitchRef = draftRef[grammarProfileSwitchDraft]
	grammarProfileCaseRef   = draftRef[grammarProfileCaseDraft]
	ruleDraftRef            = draftRef[ruleDraft]
	advisoryDraftRef        = draftRef[advisoryDraft]
	shortcutDraftRef        = draftRef[shortcutDraft]
	signalEndpointDraftRef  = draftRef[signalEndpointDraft]
	helpGroupRef            = draftRef[helpGroupDraft]
)
