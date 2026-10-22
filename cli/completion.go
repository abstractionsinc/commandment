package cli

import (
	"context"
	"errors"
)

var errInvalidCompletionAdapterInvocation = errors.New("cli: invalid completion adapter invocation")

type authoringCompletion struct {
	kind         completionDeclarationKind
	callback     completionCallback
	pending      []destinationDependency
	dependencies []authoringDestinationDependency
	options      []SourceOption
	nativeKind   CompletionNativeKind
	nativeMode   CompletionNativeMode
	hasNative    bool
	valid        bool
}

// flagCompletion is the declaration-local continuation returned by semantic flag completion.
// It retains the flag value type while exposing only the operations that refine that same
// completion plus the spelling/dependency operations required to continue authoring the same flag.
type flagCompletion[C, T any] struct {
	state *declarationSession
	flag  flagRef
}

type flagCompletedDeclaration[C, T any] struct {
	state *declarationSession
	flag  flagRef
}

// argumentCompletion is the declaration-local continuation returned by semantic positional
// completion.
type argumentCompletion[C any] struct {
	state *declarationSession
	term  argumentTermDraftRef
}

type argumentCompletedDeclaration[C any] struct {
	state *declarationSession
	term  argumentTermDraftRef
}

// argumentDelegationCompletion is the declaration-local continuation returned by positional
// completion delegation.
type argumentDelegationCompletion[C any] struct {
	state *declarationSession
	term  argumentTermDraftRef
}

// CompleteBy attaches semantic value completion to this flag declaration.
func (flag FlagDeclaration[C, T]) CompleteBy(complete ValueCompleter[C], options ...SourceOption) flagCompletion[C, T] {
	setFlagAuthoringCompletion(flag.state, flag.flag, valueCompletionAuthoring(complete, nil, nil, options))
	completion := flagCompletion[C, T]{
		state: flag.state,
		flag:  flag.flag,
	}
	return completion
}

// CompleteTokensBy attaches token-language completion to this flag declaration.
func (flag FlagDeclaration[C, T]) CompleteTokensBy(complete TokenLanguageCompleter[C], options ...SourceOption) flagCompletion[C, T] {
	setFlagAuthoringCompletion(flag.state, flag.flag, valueCompletionAuthoring(nil, complete, nil, options))
	completion := flagCompletion[C, T]{
		state: flag.state,
		flag:  flag.flag,
	}
	return completion
}

// CompleteWith attaches client-native completion to this flag declaration.
func (flag FlagDeclaration[C, T]) CompleteWith(native CompletionNativeKind) FlagDeclaration[C, T] {
	setFlagAuthoringCompletion(flag.state, flag.flag, nativeCompletionAuthoring(native))
	return flag
}

// From replaces the dependencies observed by this flag completion callback.
func (completion flagCompletion[C, T]) From(dependencies ...Dependency) flagCompletion[C, T] {
	setFlagCompletionDependencies(completion.state, completion.flag, dependencies)
	return completion
}

// FallbackTo adds client-native completion only when semantic completion contributes no match.
func (completion flagCompletion[C, T]) FallbackTo(native CompletionNativeKind) flagCompletedDeclaration[C, T] {
	composeFlagCompletion(completion.state, completion.flag, native, CompletionNativeOnNoMatch)
	return flagCompletedDeclaration[C, T](completion)
}

// CombineWith adds client-native completion alongside semantic completion.
func (completion flagCompletion[C, T]) CombineWith(native CompletionNativeKind) flagCompletedDeclaration[C, T] {
	composeFlagCompletion(completion.state, completion.flag, native, CompletionNativeAlways)
	return flagCompletedDeclaration[C, T](completion)
}

// Long continues spelling authoring for the same flag after completion configuration.
func (completion flagCompletion[C, T]) Long(name string) FlagSpellingDeclaration[C, T] {
	return longFlagSpelling(completion.flagDeclaration(), name)
}

// Short continues spelling authoring for the same flag after completion configuration.
func (completion flagCompletion[C, T]) Short(name rune) FlagSpellingDeclaration[C, T] {
	return shortFlagSpelling(completion.flagDeclaration(), name)
}

// Literal continues spelling authoring for the same flag after completion configuration.
func (completion flagCompletion[C, T]) Literal(spelling string) FlagSpellingDeclaration[C, T] {
	return literalFlagSpelling(completion.flagDeclaration(), spelling)
}

// OptionalDependency identifies the flag destination as an optional dependency.
func (completion flagCompletion[C, T]) OptionalDependency() Dependency {
	return completion.flagDeclaration().OptionalDependency()
}

func (completion flagCompletion[C, T]) flagDeclaration() FlagDeclaration[C, T] {
	var handle *destinationHandle
	if completion.state != nil && completion.flag.valid() {
		completion.state.use(func(draft *declarationDraft, _ commandRef) {
			flag, ok := draft.flags.resolve(draft, completion.flag)
			if !ok {
				panic("cli: issued flag draft cannot resolve")
			}
			destination, ok := draft.destinations.resolve(draft, flag.destination)
			if !ok {
				panic("cli: issued flag destination cannot resolve")
			}
			handle = destination.handle
		})
	}
	destination := Destination[C, T]{
		handle: handle,
	}
	declaration := FlagDeclaration[C, T]{
		state:       completion.state,
		destination: destination,
		flag:        completion.flag,
	}
	return declaration
}

// Long continues spelling authoring for the same flag after native composition.
func (completion flagCompletedDeclaration[C, T]) Long(name string) FlagSpellingDeclaration[C, T] {
	return longFlagSpelling(completion.flagDeclaration(), name)
}

// Short continues spelling authoring for the same flag after native composition.
func (completion flagCompletedDeclaration[C, T]) Short(name rune) FlagSpellingDeclaration[C, T] {
	return shortFlagSpelling(completion.flagDeclaration(), name)
}

// Literal continues spelling authoring for the same flag after native composition.
func (completion flagCompletedDeclaration[C, T]) Literal(spelling string) FlagSpellingDeclaration[C, T] {
	return literalFlagSpelling(completion.flagDeclaration(), spelling)
}

// OptionalDependency identifies the flag destination as an optional dependency.
func (completion flagCompletedDeclaration[C, T]) OptionalDependency() Dependency {
	return completion.flagDeclaration().OptionalDependency()
}

func (completion flagCompletedDeclaration[C, T]) flagDeclaration() FlagDeclaration[C, T] {
	return flagCompletion[C, T](completion).flagDeclaration()
}

// CompleteBy attaches semantic value completion to this positional declaration.
func (argument ArgumentDeclaration[C, T]) CompleteBy(complete ValueCompleter[C], options ...SourceOption) argumentCompletion[C] {
	setArgumentAuthoringCompletion(argument.state, argument.term, valueCompletionAuthoring(complete, nil, nil, options))
	completion := argumentCompletion[C]{
		state: argument.state,
		term:  argument.term,
	}
	return completion
}

// CompleteTokensBy attaches token-language completion to this positional declaration.
func (argument ArgumentDeclaration[C, T]) CompleteTokensBy(complete TokenLanguageCompleter[C], options ...SourceOption) argumentCompletion[C] {
	setArgumentAuthoringCompletion(argument.state, argument.term, valueCompletionAuthoring(nil, complete, nil, options))
	completion := argumentCompletion[C]{
		state: argument.state,
		term:  argument.term,
	}
	return completion
}

// CompleteWith attaches client-native completion to this positional declaration.
func (argument ArgumentDeclaration[C, T]) CompleteWith(native CompletionNativeKind) ArgumentDeclaration[C, T] {
	setArgumentAuthoringCompletion(argument.state, argument.term, nativeCompletionAuthoring(native))
	return argument
}

// DelegateCompletionBy attaches semantic completion delegation to this positional declaration.
func (argument ArgumentDeclaration[C, T]) DelegateCompletionBy(delegate CompletionDelegator[C], options ...SourceOption) argumentDelegationCompletion[C] {
	setArgumentAuthoringCompletion(argument.state, argument.term, delegationCompletionAuthoring(delegate, nil, options))
	completion := argumentDelegationCompletion[C]{
		state: argument.state,
		term:  argument.term,
	}
	return completion
}

// From replaces the dependencies observed by this positional completion callback.
func (completion argumentCompletion[C]) From(dependencies ...Dependency) argumentCompletion[C] {
	setArgumentCompletionDependencies(completion.state, completion.term, dependencies)
	return completion
}

// FallbackTo adds client-native completion only when semantic completion contributes no match.
func (completion argumentCompletion[C]) FallbackTo(native CompletionNativeKind) argumentCompletedDeclaration[C] {
	composeArgumentCompletion(completion.state, completion.term, native, CompletionNativeOnNoMatch)
	return argumentCompletedDeclaration[C](completion)
}

// CombineWith adds client-native completion alongside semantic completion.
func (completion argumentCompletion[C]) CombineWith(native CompletionNativeKind) argumentCompletedDeclaration[C] {
	composeArgumentCompletion(completion.state, completion.term, native, CompletionNativeAlways)
	return argumentCompletedDeclaration[C](completion)
}

// From replaces the dependencies observed by this positional delegation callback.
func (completion argumentDelegationCompletion[C]) From(dependencies ...Dependency) argumentDelegationCompletion[C] {
	setArgumentCompletionDependencies(completion.state, completion.term, dependencies)
	return completion
}

func setFlagAuthoringCompletion(state *declarationSession, flag flagRef, completion authoringCompletion) {
	mutateFlag(state, flag, func(record *flagDraft) {
		if record.completionSet {
			record.valid = false
			return
		}
		record.completionSet = true
		record.completion = snapshotAuthoringCompletion(completion)
		record.valid = record.valid && completion.valid
	})
}

func setArgumentAuthoringCompletion(state *declarationSession, term argumentTermDraftRef, completion authoringCompletion) {
	mutateArgumentTerm(state, term, func(record *argumentTermDraft) {
		if record.completionSet {
			record.valid = false
			return
		}
		record.completionSet = true
		record.completion = snapshotAuthoringCompletion(completion)
		record.valid = record.valid && completion.valid
	})
}

func setFlagCompletionDependencies(state *declarationSession, flag flagRef, dependencies []Dependency) {
	mutateFlag(state, flag, func(record *flagDraft) {
		if !record.completionSet || record.completion.hasNative && !record.completion.callback.present() {
			record.valid = false
			return
		}
		pending, ok := completionDependencies(dependencies)
		record.completion.pending = pending
		record.completion.dependencies = nil
		record.completion.valid = record.completion.valid && ok
		record.valid = record.valid && ok
	})
}

func setArgumentCompletionDependencies(state *declarationSession, term argumentTermDraftRef, dependencies []Dependency) {
	mutateArgumentTerm(state, term, func(record *argumentTermDraft) {
		if !record.completionSet || !record.completion.callback.present() {
			record.valid = false
			return
		}
		pending, ok := completionDependencies(dependencies)
		record.completion.pending = pending
		record.completion.dependencies = nil
		record.completion.valid = record.completion.valid && ok
		record.valid = record.valid && ok
	})
}

func composeFlagCompletion(state *declarationSession, flag flagRef, native CompletionNativeKind, mode CompletionNativeMode) {
	mutateFlag(state, flag, func(record *flagDraft) {
		if !record.completionSet || !record.completion.callback.present() || record.completion.hasNative {
			record.valid = false
			return
		}
		ok := native.valid() && mode.valid()
		record.completion.nativeKind = native
		record.completion.nativeMode = mode
		record.completion.hasNative = true
		record.completion.valid = record.completion.valid && ok
		record.valid = record.valid && ok
	})
}

func composeArgumentCompletion(state *declarationSession, term argumentTermDraftRef, native CompletionNativeKind, mode CompletionNativeMode) {
	mutateArgumentTerm(state, term, func(record *argumentTermDraft) {
		if !record.completionSet || !record.completion.callback.present() || record.completion.hasNative {
			record.valid = false
			return
		}
		ok := native.valid() && mode.valid()
		record.completion.nativeKind = native
		record.completion.nativeMode = mode
		record.completion.hasNative = true
		record.completion.valid = record.completion.valid && ok
		record.valid = record.valid && ok
	})
}

func completionDependencies(values []Dependency) ([]destinationDependency, bool) {
	return dependencyDescriptors(values)
}

func valueCompletionAuthoring[C any](complete ValueCompleter[C], tokens TokenLanguageCompleter[C], dependencies []Dependency, options []SourceOption) authoringCompletion {
	deps, ok := completionDependencies(dependencies)
	if !ok || (complete == nil) == (tokens == nil) {
		return authoringCompletion{}
	}
	result := authoringCompletion{
		pending: deps,
		options: append([]SourceOption(nil), options...),
		valid:   true,
	}
	if complete != nil {
		result.kind = completionDeclarationKindValue
		result.callback.value = func(ctx context.Context, rawConfiguration any, request ValueCompletionRequest) (CompletionContribution, error) {
			configuration, ok := rawConfiguration.(*C)
			if !ok {
				return CompletionContribution{}, errInvalidCompletionAdapterInvocation
			}
			return complete(ctx, configuration, request)
		}
		return result
	}
	result.kind = completionDeclarationKindTokenLanguage
	result.callback.tokenLanguage = func(ctx context.Context, rawConfiguration any, request TokenLanguageCompletionRequest) (CompletionContribution, error) {
		configuration, ok := rawConfiguration.(*C)
		if !ok {
			return CompletionContribution{}, errInvalidCompletionAdapterInvocation
		}
		return tokens(ctx, configuration, request)
	}
	return result
}

func delegationCompletionAuthoring[C any](delegate CompletionDelegator[C], dependencies []Dependency, options []SourceOption) authoringCompletion {
	deps, ok := completionDependencies(dependencies)
	if !ok || delegate == nil {
		return authoringCompletion{}
	}
	callback := completionCallback{}
	callback.delegation = func(ctx context.Context, rawConfiguration any, request CompletionDelegationRequest) (CompletionDelegation, error) {
		configuration, ok := rawConfiguration.(*C)
		if !ok {
			return CompletionDelegation{}, errInvalidCompletionAdapterInvocation
		}
		return delegate(ctx, configuration, request)
	}
	completion := authoringCompletion{
		kind:     completionDeclarationKindDelegation,
		callback: callback,
		pending:  deps,
		options:  append([]SourceOption(nil), options...),
		valid:    true,
	}

	return completion
}

func nativeCompletionAuthoring(native CompletionNativeKind) authoringCompletion {
	completion := authoringCompletion{
		kind:       completionDeclarationKindNative,
		nativeKind: native,
		nativeMode: CompletionNativeAlways,
		hasNative:  true,
		valid:      native.valid(),
	}

	return completion
}

func snapshotAuthoringCompletion(completion authoringCompletion) authoringCompletion {
	completion.pending = append([]destinationDependency(nil), completion.pending...)
	completion.dependencies = append([]authoringDestinationDependency(nil), completion.dependencies...)
	completion.options = append([]SourceOption(nil), completion.options...)
	return completion
}

func freezeAuthoringCompletion(draft *declarationDraft, completion authoringCompletion) (authoringCompletion, bool) {
	completion = snapshotAuthoringCompletion(completion)
	if draft == nil || !completion.valid {
		return completion, false
	}
	dependencies, ok := freezeAuthoringDestinationDependencies(draft, completion.pending)
	completion.pending = nil
	completion.dependencies = dependencies
	completion.valid = completion.valid && ok
	return completion, completion.valid
}

func prepareAuthoringCompletion(completion authoringCompletion, resolver destinationAcceptanceResolver) (completionDeclaration, completionCallback, bool) {
	if !completion.valid {
		return completionDeclaration{}, completionCallback{}, false
	}
	source, ok := producerSource(completion.options)
	if !ok || len(completion.pending) != 0 {
		return completionDeclaration{}, completionCallback{}, false
	}
	resolved, ok := resolveDestinationDependencies(completion.dependencies, resolver)
	if !ok {
		return completionDeclaration{}, completionCallback{}, false
	}
	callback := completion.callback
	if callback.present() && len(resolved.keys) != 0 {
		frozen := freezeResolvedDestinationDependencies(resolved)
		callback = callback.withValueStateKeyAliases(frozen.keys)
	}
	candidate := completionDeclarationCandidate{
		Kind: completion.kind,
		Callback: completionCallbackCandidate{
			Callback:     callback,
			Dependencies: resolved.requirements,
			Source:       source,
		},
		Native: completionNativeCandidate{
			Kind:    completion.nativeKind,
			Mode:    completion.nativeMode,
			Present: completion.hasNative,
		},
	}
	if !callback.present() && completion.kind != completionDeclarationKindNative {
		return completionDeclaration{}, completionCallback{}, false
	}
	definition := acceptCompletionDeclaration(candidate)
	return definition, callback, definition.Valid()
}

// CompletionRequest describes one semantic analysis of an incomplete invocation. Args is the
// complete argument-vector representation for the request, Cursor uses byte coordinates, and the
// remaining fields provide host inputs, client capabilities, visibility, locale, and analysis
// bounds without transferring grammar or execution authority to the client.
type CompletionRequest struct {
	// Args is the exact execution-argument lane for direct callers. It is mutually exclusive
	// with ObservedInput.
	Args []string

	// ObservedInput is the completion-time observation lane for shell/editor clients that cannot
	// truthfully claim execution-equivalent argv. It is mutually exclusive with Args/Cursor.
	ObservedInput *CompletionInputObservation

	Cursor  CompletionCursor
	Trigger CompletionTrigger

	Env Env

	WorkingDirectory string
	ResponseFiles    ResponseFileAccess

	Defaults []DefaultProvider
	Facts    []FactProvider

	Locale     Locale
	Visibility CompletionVisibility
	Client     CompletionClient

	Limits AnalysisLimits

	Limit int
}

// CompletionInputFidelity classifies how faithfully one observed completion word represents the
// corresponding execution-time argument value. It is independent of argument-boundary certainty.
type CompletionInputFidelity uint8

const (
	// CompletionInputFidelityUnknown is invalid for an observed word.
	CompletionInputFidelityUnknown CompletionInputFidelity = iota
	// CompletionInputFidelityExact means the observed value is execution-equivalent for that word.
	CompletionInputFidelityExact
	// CompletionInputFidelityLexical means the value is a shell/editor lexical observation.
	CompletionInputFidelityLexical
	// CompletionInputFidelityExpanded means the value reflects some native expansion but is not
	// established as execution-equivalent.
	CompletionInputFidelityExpanded
)

// CompletionArgvTopology classifies whether observed word boundaries/cardinality are established
// to map one-to-one to execution argv.
type CompletionArgvTopology uint8

const (
	// CompletionArgvTopologyUnknown is invalid for an observed input.
	CompletionArgvTopologyUnknown CompletionArgvTopology = iota
	// CompletionArgvTopologyExact establishes one-to-one argv topology for the entire observation.
	CompletionArgvTopologyExact
	// CompletionArgvTopologyPrefixExact establishes one-to-one argv topology only for
	// ExactArgvPrefix leading words; later word positions are observation-space only.
	CompletionArgvTopologyPrefixExact
	// CompletionArgvTopologyUncertain establishes no execution-argv topology beyond the empty prefix.
	CompletionArgvTopologyUncertain
)

// CompletionCursorSpace identifies the byte-coordinate system used by an observed completion
// cursor. The semantic engine never silently converts between these spaces.
type CompletionCursorSpace uint8

const (
	// CompletionCursorSpaceUnknown is invalid.
	CompletionCursorSpaceUnknown CompletionCursorSpace = iota
	// CompletionCursorSpaceArgumentBytes uses decoded exact-argument UTF-8 byte coordinates.
	CompletionCursorSpaceArgumentBytes
	// CompletionCursorSpaceObservedWordBytes uses UTF-8 byte coordinates in the reported observed word.
	CompletionCursorSpaceObservedWordBytes
	// CompletionCursorSpaceSourceBytes uses bytes in the shell/editor source buffer and cannot be
	// treated as decoded-argument coordinates without adapter-established conversion.
	CompletionCursorSpaceSourceBytes
)

// CompletionObservedWord is one shell/editor completion-time word observation.
type CompletionObservedWord struct {
	Value    string
	Fidelity CompletionInputFidelity
}

// CompletionObservedCursor locates the cursor in the observed-input coordinate system.
type CompletionObservedCursor struct {
	Word  int
	Byte  int
	Space CompletionCursorSpace
}

// CompletionInputObservation preserves completion-time input truth without fabricating exact argv.
// ExactArgvPrefix is the number of leading observed words whose one-to-one argv topology is
// independently established when Topology is CompletionArgvTopologyPrefixExact.
type CompletionInputObservation struct {
	Words           []CompletionObservedWord
	Topology        CompletionArgvTopology
	ExactArgvPrefix int
	Cursor          CompletionObservedCursor
}

// CompletionCursor identifies the cursor-owned argument and byte offset within that decoded
// argument.
type CompletionCursor struct {
	Argument int
	Byte     int
}

// CompletionVisibility selects the declaration visibility view used for completion. Including
// hidden-but-invocable declarations does not change their declared hidden state or semantic
// visibility.
type CompletionVisibility uint8

const (
	// CompletionVisibilityPublic selects ordinary public completion, excluding declarations hidden
	// from public presentation.
	CompletionVisibilityPublic CompletionVisibility = iota

	// CompletionVisibilityAllInvocable includes hidden-but-invocable declarations without changing
	// their hidden or visibility state.
	CompletionVisibilityAllInvocable
)

// CompletionTriggerKind classifies why a completion request was issued.
type CompletionTriggerKind uint8

const (
	// CompletionTriggerInvoked means completion was explicitly invoked.
	CompletionTriggerInvoked CompletionTriggerKind = iota

	// CompletionTriggerCharacter means completion was triggered by a specific input character recorded
	// in CompletionTrigger.Character.
	CompletionTriggerCharacter

	// CompletionTriggerRefresh means the client requested refresh of an existing completion view.
	CompletionTriggerRefresh
)

// CompletionTrigger records the request trigger. Character is meaningful only for
// CompletionTriggerCharacter.
type CompletionTrigger struct {
	Kind      CompletionTriggerKind
	Character rune
}

// CompletionShell records first-class facts about the shell that is actually running the
// completion client. Family and Version are runtime client metadata; they are not a Completion
// protocol generation and are normally populated by shell integration rather than application
// configuration.
type CompletionShell struct {
	Family  ShellFamily
	Version string
}

// CompletionClient describes the external completion consumer and its matching, ranking, and
// output capabilities. Name, Version, and Shell are client metadata; they are not Completion
// transport generation or semantic grammar state.
type CompletionClient struct {
	Name         string
	Version      string
	Shell        CompletionShell
	Match        CompletionMatchOptions
	Ranking      CompletionRankingMode
	Capabilities CompletionClientCapabilities
}

// CompletionNativeCapabilities declares which client-side native enumeration categories the
// consumer can perform. Native enumeration remains owned by the shell, editor, or completion
// client.
type CompletionNativeCapabilities struct {
	Paths                bool
	Files                bool
	Directories          bool
	Executables          bool
	Users                bool
	Hosts                bool
	EnvironmentVariables bool
}

// CompletionClientCapabilities declares which structured completion features the consumer can
// accept. Capability negotiation may shape presentation data such as match ranges, but it does not
// redefine recognition or semantic candidate generation.
type CompletionClientCapabilities struct {
	Descriptions     bool
	Details          bool
	Groups           bool
	CandidateKinds   bool
	MatchRanges      bool
	CommitCharacters bool
	Messages         bool
	Delegation       bool
	Native           CompletionNativeCapabilities
	RefreshOnInput   bool
}

// CompletionMatchMode selects the maximum textual match class admitted after semantic candidate
// production. The strength ladder is Exact, Prefix, Substring, Fuzzy, Approximate; Unfiltered
// performs no textual filtering.
type CompletionMatchMode uint8

const (
	// CompletionMatchModePrefix admits Exact and Prefix matches.
	CompletionMatchModePrefix CompletionMatchMode = iota

	// CompletionMatchModeExact admits only whole-field Exact matches.
	CompletionMatchModeExact

	// CompletionMatchModeSubstring admits matches through Substring.
	CompletionMatchModeSubstring

	// CompletionMatchModeFuzzy admits matches through ordered noncontiguous Fuzzy matching.
	CompletionMatchModeFuzzy

	// CompletionMatchModeApproximate admits matches through the bounded Approximate class after all
	// stronger classes fail.
	CompletionMatchModeApproximate

	// CompletionMatchModeUnfiltered performs no server-side textual filtering and publishes an unknown
	// match descriptor for retained candidates.
	CompletionMatchModeUnfiltered
)

// CompletionCaseMode selects locale-independent client matching case behavior. Smart uses the
// original query: any Unicode uppercase scalar selects sensitive comparison, otherwise insensitive
// comparison.
type CompletionCaseMode uint8

const (
	// CompletionCaseSmart chooses sensitive matching when the original query contains any Unicode
	// uppercase scalar, otherwise insensitive matching.
	CompletionCaseSmart CompletionCaseMode = iota

	// CompletionCaseSensitive performs matching without case folding.
	CompletionCaseSensitive
	// CompletionCaseInsensitive uses locale-independent Unicode Simple Case Folding.
	CompletionCaseInsensitive
)

// CompletionMatchField identifies a candidate field used for client matching. Without an explicit
// field list, matching checks Value, DisplayText, then SearchTerm; explicit fields retain caller
// preference order.
type CompletionMatchField uint8

const (
	// CompletionMatchFieldUnknown is the zero/unknown match-field classification.
	CompletionMatchFieldUnknown CompletionMatchField = iota
	// CompletionMatchFieldValue matches the candidate insertion Value.
	CompletionMatchFieldValue
	// CompletionMatchFieldDisplayText matches candidate DisplayText.
	CompletionMatchFieldDisplayText
	// CompletionMatchFieldSearchTerm matches SearchTerms in increasing slice order and publishes the
	// selected term index in FieldIndex.
	CompletionMatchFieldSearchTerm
	// CompletionMatchFieldDescription matches candidate Description when explicitly requested.
	CompletionMatchFieldDescription
	// CompletionMatchFieldDetail matches candidate Detail when explicitly requested.
	CompletionMatchFieldDetail
)

// CompletionMatchOptions configures post-generation client matching. Fields are
// preference-ordered, and IgnoreDiacritics uses canonical NFD diacritic removal while preserving
// ranges in original UTF-8 byte coordinates.
type CompletionMatchOptions struct {
	Mode CompletionMatchMode
	Case CompletionCaseMode

	Fields []CompletionMatchField

	IgnoreDiacritics bool
}

// CompletionRankingMode selects deterministic ordering after matching. Ranking may reorder
// eligible candidates and populate Score, but cannot change grammar recognition, matching
// eligibility, target ownership, fallback tiers, or exact declaration identity.
type CompletionRankingMode uint8

const (
	// CompletionRankingModeRelevance orders deterministically by semantic relevance and reports the named
	// relevance algorithm.
	CompletionRankingModeRelevance CompletionRankingMode = iota

	// CompletionRankingModeSource preserves established source and semantic contribution order.
	CompletionRankingModeSource

	// CompletionRankingModeLexical orders deterministically by lexical sort text, falling back to
	// candidate Value.
	CompletionRankingModeLexical

	// CompletionRankingModeUnranked requests no ranking transform and preserves the already-established
	// candidate order.
	CompletionRankingModeUnranked
)

// CompletionRankingAlgorithm identifies the public ranking algorithm applied when the selected
// ranking mode has one. The empty value reports no named algorithm.
type CompletionRankingAlgorithm string

const (
	// CompletionRankingAlgorithmUnknown reports that no public ranking algorithm is named.
	CompletionRankingAlgorithmUnknown CompletionRankingAlgorithm = ""
	// CompletionRankingAlgorithmRelevance identifies Commandment relevance ranking.
	CompletionRankingAlgorithmRelevance CompletionRankingAlgorithm = "relevance"
)

// CompletionTargetKind classifies the semantic owner of the incomplete input at the cursor. Its
// zero value is the unknown/unset classification.
type CompletionTargetKind uint8

const (
	// CompletionTargetUnknown is the zero/unknown target classification.
	CompletionTargetUnknown CompletionTargetKind = iota
	// CompletionTargetCommand identifies command-name completion.
	CompletionTargetCommand
	// CompletionTargetFlag identifies flag-spelling completion.
	CompletionTargetFlag
	// CompletionTargetFlagValue identifies completion of a flag value.
	CompletionTargetFlagValue
	// CompletionTargetArgument identifies positional argument completion.
	CompletionTargetArgument
	// CompletionTargetLiteral identifies a literal argument term.
	CompletionTargetLiteral
	// CompletionTargetTokenLanguage identifies a bounded token-language term whose completer owns only
	// that language input.
	CompletionTargetTokenLanguage
	// CompletionTargetRemainder identifies the authoritative remainder owner.
	CompletionTargetRemainder
	// CompletionTargetPassThrough identifies the authoritative pass-through owner.
	CompletionTargetPassThrough
	// CompletionTargetEndOfOptions identifies the end-of-options boundary.
	CompletionTargetEndOfOptions
	// CompletionTargetResponseFile identifies an incomplete response-file reference; the marker
	// remains Commandment-owned while only the path portion is completed.
	CompletionTargetResponseFile
)

// CompletionTarget identifies one viable semantic completion target and the declaration, spelling,
// form, term, and occurrence facts needed to preserve target ownership through callbacks,
// fallback, native intent, deduplication, and ranking.
type CompletionTarget struct {
	Kind CompletionTargetKind

	Command CommandID

	CommandPath []string

	ValueID        ValueID
	ArgumentFormID ArgumentFormID
	TermIndex      int

	Name     string
	Spelling string

	FlagValueForm FlagValueForm

	Occurrence int
}

// CompletionCandidate is application-authored suggestion content. A nil Replacement uses Value
// as one decoded replacement entry; a nonnil Replacement is an ordered structural replacement
// sequence. Candidate text marked Sensitive is not exposed, and this value does not establish
// exact declaration provenance by itself.
type CompletionCandidate struct {
	Value       string
	Replacement []string

	DisplayText string

	Description string

	Detail string

	Kind CompletionCandidateKind

	GroupID string

	SearchTerms []string

	SortText string

	Priority int

	Recommended bool

	FallbackOnly bool

	LifecycleState LifecycleState

	Spacing CompletionSpacing

	CommitCharacters []rune

	Sensitive bool
}

// CompletionItem is a candidate observation produced by Commandment. Declaration, when
// nonzero, identifies the exact declaration projected by the item; TargetIndexes retain target
// relationships, Match records the selected client match, and Required/Conditional preserve
// composition status. Different nonzero Declaration identities are not equivalent merely
// because candidate text matches.
type CompletionItem struct {
	CompletionCandidate

	Declaration DeclarationIdentity

	TargetIndexes []int

	Match CompletionMatch

	Required bool

	Conditional bool
}

// CompletionMatchKind classifies the textual match selected for an eligible item. Its zero value
// is also the descriptor used for an original empty query or unfiltered matching.
type CompletionMatchKind uint8

const (
	// CompletionMatchKindUnknown is the zero/unknown match classification.
	CompletionMatchKindUnknown CompletionMatchKind = iota
	// CompletionMatchKindExact records whole-field equality.
	CompletionMatchKindExact
	// CompletionMatchKindPrefix records a prefix match selected after Exact was unavailable.
	CompletionMatchKindPrefix
	// CompletionMatchKindSubstring records a substring match selected after stronger classes were
	// unavailable.
	CompletionMatchKindSubstring
	// CompletionMatchKindFuzzy records ordered noncontiguous subsequence matching.
	CompletionMatchKindFuzzy
	// CompletionMatchKindApproximate records bounded edit-distance matching selected only after
	// stronger classes failed.
	CompletionMatchKindApproximate
)

// CompletionMatch records the match selected by Commandment, including field, strength,
// ranking score, field index, original-field UTF-8 byte ranges, and approximate edit count.
// Matching owns every field except Score; ranking may set Score but cannot revise eligibility
// or alignment.
type CompletionMatch struct {
	Kind  CompletionMatchKind
	Field CompletionMatchField

	FieldIndex int

	Score int64

	Ranges []CompletionRange

	Edits int
}

// CompletionRange is a half-open UTF-8 byte range. Completion matching ranges address the original
// selected candidate field rather than normalized comparison text.
type CompletionRange struct {
	Start int
	End   int
}

// CompletionSpan identifies a half-open byte range within one decoded argument entry.
type CompletionSpan struct {
	Argument int
	Start    int
	End      int
}

// ValueCompletionRequest is the canonical client-independent request for one semantic
// value-completion callback. It exposes only the target, finalized prior selection/dependencies,
// incomplete text and byte range, host working directory, trigger, and bound needed by that
// callback.
type ValueCompletionRequest struct {
	Target    CompletionTarget
	Selection CommandSelection

	Text string

	Cursor int

	Replace CompletionRange

	WorkingDirectory string

	Trigger CompletionTrigger

	Limit int
}

// CompletionContribution is one uncommitted semantic completion callback result. It may contribute
// candidates, groups, messages, replacement, spacing, native intent, truncation, and refresh
// behavior for its target; if the callback returns an error, accompanying contribution data is
// ignored.
type CompletionContribution struct {
	Replace *CompletionRange

	Candidates []CompletionCandidate
	Groups     []CompletionGroup
	Messages   []CompletionMessage

	Spacing CompletionSpacing
	Native  *CompletionNativeIntent

	Truncated bool

	RefreshOnInput bool
}

// ValueCompleter performs semantic value completion for one target. It may be invoked
// speculatively and concurrently across viable targets, so it must be deterministic for its
// declared inputs, side-effect-free for speculative analysis, cancellation-aware, and
// read-only with respect to Commandment configuration.
type ValueCompleter[C any] func(context.Context, *C, ValueCompletionRequest) (CompletionContribution, error)

// CompletionCandidateKind classifies candidate presentation/application intent. Candidate kind
// does not establish exact declaration provenance; CompletionItem.Declaration owns that
// Commandment-established relationship.
type CompletionCandidateKind string

const (
	// CompletionCandidateUnknown is the empty/unknown candidate kind.
	CompletionCandidateUnknown CompletionCandidateKind = ""

	// CompletionCandidateValue classifies a semantic value candidate.
	CompletionCandidateValue CompletionCandidateKind = "value"
	// CompletionCandidateCommand classifies a canonical command candidate.
	CompletionCandidateCommand CompletionCandidateKind = "command"
	// CompletionCandidateCommandAlias classifies a lexical command-alias candidate.
	CompletionCandidateCommandAlias CompletionCandidateKind = "command-alias"
	// CompletionCandidateCommandShortcut classifies a command-shortcut candidate.
	CompletionCandidateCommandShortcut CompletionCandidateKind = "command-shortcut"
	// CompletionCandidateAction classifies an intrinsic-action candidate.
	CompletionCandidateAction CompletionCandidateKind = "action"
	// CompletionCandidateFlag classifies a flag candidate.
	CompletionCandidateFlag CompletionCandidateKind = "flag"
	// CompletionCandidateKeyword classifies a keyword candidate, including literal argument terms
	// where applicable.
	CompletionCandidateKeyword CompletionCandidateKind = "keyword"
	// CompletionCandidateFile classifies a file candidate.
	CompletionCandidateFile CompletionCandidateKind = "file"
	// CompletionCandidateDirectory classifies a directory candidate.
	CompletionCandidateDirectory CompletionCandidateKind = "directory"
	// CompletionCandidateExecutable classifies an executable candidate.
	CompletionCandidateExecutable CompletionCandidateKind = "executable"
	// CompletionCandidateResource classifies a resource candidate.
	CompletionCandidateResource CompletionCandidateKind = "resource"
	// CompletionCandidateNamespace classifies a namespace candidate.
	CompletionCandidateNamespace CompletionCandidateKind = "namespace"
	// CompletionCandidateUser classifies a user-name candidate.
	CompletionCandidateUser CompletionCandidateKind = "user"
	// CompletionCandidateHost classifies a host-name candidate.
	CompletionCandidateHost CompletionCandidateKind = "host"
	// CompletionCandidateEnvironmentVariable classifies an environment-variable-name candidate.
	CompletionCandidateEnvironmentVariable CompletionCandidateKind = "environment-variable"
	// CompletionCandidateURL classifies a URL candidate.
	CompletionCandidateURL CompletionCandidateKind = "url"
)

// CompletionGroup describes one presentation group for completion items. Group order and Priority
// participate only in completion presentation/ranking; Sensitive prevents unsafe exposure of group
// text.
type CompletionGroup struct {
	ID       string
	Label    string
	Priority int

	Sensitive bool
}

// CompletionMessageKind classifies a structured completion message. Its zero value is the unknown
// classification.
type CompletionMessageKind uint8

const (
	// CompletionMessageUnknown is the zero/unknown message classification.
	CompletionMessageUnknown CompletionMessageKind = iota
	// CompletionMessageInformation marks informational completion presentation.
	CompletionMessageInformation
	// CompletionMessageWarning marks warning completion presentation.
	CompletionMessageWarning
	// CompletionMessageError marks error completion presentation; it does not by itself mean the
	// Completion operation returned an error.
	CompletionMessageError
)

// CompletionMessage carries structured completion presentation emitted alongside candidates.
// Sensitive messages must not be exposed by library-generated presentation.
type CompletionMessage struct {
	Kind      CompletionMessageKind
	Text      string
	Sensitive bool
}

// CompletionSpacing describes how a client should separate an applied completion from following
// input.
type CompletionSpacing uint8

const (
	// CompletionSpacingDefault requests the client default spacing behavior.
	CompletionSpacingDefault CompletionSpacing = iota
	// CompletionSpacingAppendSpace requests a following space after applying the completion.
	CompletionSpacingAppendSpace
	// CompletionSpacingNoSpace requests no automatically appended space.
	CompletionSpacingNoSpace
)

// CompletionNativeKind identifies an operating-system enumeration category requested from the
// completion client. Commandment describes the intent; the client performs enumeration.
type CompletionNativeKind uint8

const (
	// CompletionNativeUnknown is the zero/unknown native category.
	CompletionNativeUnknown CompletionNativeKind = iota
	// CompletionNativePaths requests client-side path enumeration.
	CompletionNativePaths
	// CompletionNativeFiles requests client-side file enumeration.
	CompletionNativeFiles
	// CompletionNativeDirectories requests client-side directory enumeration.
	CompletionNativeDirectories
	// CompletionNativeExecutables requests client-side executable enumeration.
	CompletionNativeExecutables
	// CompletionNativeUsers requests client-side user-name enumeration.
	CompletionNativeUsers
	// CompletionNativeHosts requests client-side host-name enumeration.
	CompletionNativeHosts
	// CompletionNativeEnvironmentVariables requests client-side environment-variable-name enumeration.
	CompletionNativeEnvironmentVariables
)

func (kind CompletionNativeKind) valid() bool {
	return kind >= CompletionNativePaths && kind <= CompletionNativeEnvironmentVariables
}

// CompletionNativeMode controls whether client-native completion is a target-specific no-match
// fallback or is always combined with semantic output.
type CompletionNativeMode uint8

const (
	// CompletionNativeModeUnknown is the zero/unknown native composition mode.
	CompletionNativeModeUnknown CompletionNativeMode = iota

	// CompletionNativeOnNoMatch activates native completion only for the target-specific semantic
	// no-match fallback condition.
	CompletionNativeOnNoMatch

	// CompletionNativeAlways combines native intent with that target's semantic completion output.
	CompletionNativeAlways
)

func (mode CompletionNativeMode) valid() bool {
	return mode == CompletionNativeOnNoMatch || mode == CompletionNativeAlways
}

// CompletionNativeIntent is callback-authored client-native completion intent for one semantic
// target.
type CompletionNativeIntent struct {
	Kind CompletionNativeKind
	Mode CompletionNativeMode
}

// CompletionNativeRequest is native completion intent produced by Commandment for an exact
// completion target. It preserves composition mode and required/conditional status for the
// client; the client remains responsible for enumeration.
type CompletionNativeRequest struct {
	Kind CompletionNativeKind
	Mode CompletionNativeMode

	TargetIndex int

	Required    bool
	Conditional bool
}

// CompletionDelegator transfers completion ownership to an external completion system only after
// the effective grammar establishes one authoritative remainder or pass-through owner.
type CompletionDelegator[C any] func(context.Context, *C, CompletionDelegationRequest) (CompletionDelegation, error)

// CompletionDelegationRequest is the canonical client-independent handoff request for delegated
// completion. It preserves the owning target, prior command selection, argument vector, and
// cursor.
type CompletionDelegationRequest struct {
	Target    CompletionTarget
	Selection CommandSelection

	Args []string

	Cursor CompletionCursor
}

// CompletionDelegation is the structured handoff to an external completion system. Delegated
// completion is exclusive with semantic candidates and native requests in the final result;
// Sensitive marks handoff arguments that must not leak through diagnostics, tracing, generated
// shell source, or incidental logging.
type CompletionDelegation struct {
	Args   []string
	Cursor CompletionCursor

	Sensitive bool
}

// Completion is the detached semantic result of one completion analysis. It preserves targets and
// selection, replacement coordinates, items and groups, native or delegated handoff, source/origin
// closure for diagnostics and repairs, deterministic ranking metadata, and the effective locale.
// Partial reports incomplete semantic analysis; Truncated reports omitted completion results under
// completion limits; DiagnosticsTruncated independently reports diagnostic-analysis truncation.
// Caller mutation of the detached result does not change the Application.
type Completion struct {
	Locale    Locale
	Targets   []CompletionTarget
	Selection CommandSelection
	Replace   CompletionSpan

	Items  []CompletionItem
	Groups []CompletionGroup

	Spacing  CompletionSpacing
	Messages []CompletionMessage

	NativeRequests []CompletionNativeRequest

	Delegation *CompletionDelegation

	Sources map[InputSourceID]InputSource
	Origins map[InputOriginID]InputOrigin

	Diagnostics          []Diagnostic
	Repairs              []InputRepair
	DiagnosticsTruncated bool

	RankingAlgorithm CompletionRankingAlgorithm

	Partial bool

	Truncated bool

	RefreshOnInput bool
}

type acceptanceCallbackCustody uint8

const (
	acceptanceCallbackCustodyInvalid acceptanceCallbackCustody = iota
	acceptanceCallbacksStaged
	acceptanceCallbacksDetached
)

func (custody acceptanceCallbackCustody) detached() bool {
	switch custody {
	case acceptanceCallbacksStaged:
		return false
	case acceptanceCallbacksDetached:
		return true
	default:
		panic("cli: invalid acceptance callback custody")
	}
}

type completionDeclarationKind uint8

const (
	completionDeclarationKindUnknown completionDeclarationKind = iota
	completionDeclarationKindValue
	completionDeclarationKindTokenLanguage
	completionDeclarationKindDelegation
	completionDeclarationKindNative
)

func (kind completionDeclarationKind) Valid() bool {
	return kind >= completionDeclarationKindValue && kind <= completionDeclarationKindNative
}

type completionCallback struct {
	value         func(context.Context, any, ValueCompletionRequest) (CompletionContribution, error)
	tokenLanguage func(context.Context, any, TokenLanguageCompletionRequest) (CompletionContribution, error)
	delegation    func(context.Context, any, CompletionDelegationRequest) (CompletionDelegation, error)
}

func (callback completionCallback) present() bool {
	return callback.value != nil || callback.tokenLanguage != nil || callback.delegation != nil
}

func (callback completionCallback) validFor(kind completionDeclarationKind) bool {
	switch kind {
	case completionDeclarationKindValue:
		return callback.value != nil && callback.tokenLanguage == nil && callback.delegation == nil
	case completionDeclarationKindTokenLanguage:
		return callback.value == nil && callback.tokenLanguage != nil && callback.delegation == nil
	case completionDeclarationKindDelegation:
		return callback.value == nil && callback.tokenLanguage == nil && callback.delegation != nil
	case completionDeclarationKindNative:
		return !callback.present()
	case completionDeclarationKindUnknown:
		return false
	default:
		return false
	}
}

func (callback completionCallback) withValueStateKeyAliases(aliases map[destinationRef]ValueID) completionCallback {
	if len(aliases) == 0 {
		return callback
	}
	if callback.value != nil {
		base := callback.value
		callback.value = func(ctx context.Context, configuration any, request ValueCompletionRequest) (CompletionContribution, error) {
			return base(contextWithValueStateKeyAliases(ctx, aliases), configuration, request)
		}
	}
	if callback.tokenLanguage != nil {
		base := callback.tokenLanguage
		callback.tokenLanguage = func(ctx context.Context, configuration any, request TokenLanguageCompletionRequest) (CompletionContribution, error) {
			return base(contextWithValueStateKeyAliases(ctx, aliases), configuration, request)
		}
	}
	if callback.delegation != nil {
		base := callback.delegation
		callback.delegation = func(ctx context.Context, configuration any, request CompletionDelegationRequest) (CompletionDelegation, error) {
			return base(contextWithValueStateKeyAliases(ctx, aliases), configuration, request)
		}
	}
	return callback
}

type completionDeclaration struct {
	kind         completionDeclarationKind
	dependencies []dependencyRequirement
	source       semanticProducerSource
	nativeKind   CompletionNativeKind
	nativeMode   CompletionNativeMode
	hasNative    bool
}

// completionCallbackCandidate owns the callback-backed portion of a raw
// completion declaration. It intentionally permits combinations that later
// CompletionDeclaration.Valid rejects.
type completionCallbackCandidate struct {
	Callback     completionCallback
	Dependencies []dependencyRequirement
	Source       semanticProducerSource
}

// completionNativeCandidate owns native completion or native-fallback state.
// Present distinguishes absent native state from malformed present state whose
// kind or mode is invalid.
type completionNativeCandidate struct {
	Kind    CompletionNativeKind
	Mode    CompletionNativeMode
	Present bool
}

// completionDeclarationCandidate is the exact malformed-capable construction
// subject for CompletionDeclaration. Acceptance remains owned by Valid and the
// existing Draft attachment boundary.
type completionDeclarationCandidate struct {
	Kind     completionDeclarationKind
	Callback completionCallbackCandidate
	Native   completionNativeCandidate
}

func acceptCompletionDeclaration(candidate completionDeclarationCandidate) completionDeclaration {
	return acceptCompletionDeclarationWithCustody(candidate, acceptanceCallbacksStaged)
}

func acceptDetachedCompletionDeclaration(candidate completionDeclarationCandidate) completionDeclaration {
	return acceptCompletionDeclarationWithCustody(candidate, acceptanceCallbacksDetached)
}

func acceptCompletionDeclarationWithCustody(candidate completionDeclarationCandidate, custody acceptanceCallbackCustody) completionDeclaration {
	detached := custody.detached()
	callbackPresent := candidate.Callback.Callback.present()
	native := candidate.Kind == completionDeclarationKindNative
	if native && (callbackPresent || !candidate.Native.Present) {
		return completionDeclaration{}
	}
	if !native && !candidate.Kind.Valid() {
		return completionDeclaration{}
	}
	if !native && callbackPresent && !candidate.Callback.Callback.validFor(candidate.Kind) {
		return completionDeclaration{}
	}
	if !native && !callbackPresent && !detached {
		return completionDeclaration{}
	}
	if candidate.Native.Present {
		if candidate.Kind == completionDeclarationKindDelegation || !candidate.Native.Kind.valid() || !candidate.Native.Mode.valid() {
			return completionDeclaration{}
		}
	} else if candidate.Native.Kind != CompletionNativeUnknown || candidate.Native.Mode != CompletionNativeModeUnknown {
		return completionDeclaration{}
	}

	result := completionDeclaration{
		kind:         candidate.Kind,
		dependencies: append([]dependencyRequirement(nil), candidate.Callback.Dependencies...),
		source:       candidate.Callback.Source,
		nativeKind:   candidate.Native.Kind,
		nativeMode:   candidate.Native.Mode,
		hasNative:    candidate.Native.Present,
	}

	return result
}

func (definition completionDeclaration) Valid() bool {
	if !definition.kind.Valid() {
		return false
	}
	if definition.kind == completionDeclarationKindNative && !definition.hasNative {
		return false
	}
	if definition.hasNative {
		if definition.kind == completionDeclarationKindDelegation || !definition.nativeKind.valid() ||
			!definition.nativeMode.valid() {
			return false
		}
	} else if definition.nativeKind != CompletionNativeUnknown || definition.nativeMode != CompletionNativeModeUnknown {
		return false
	}
	seen := make(map[ValueID]struct{}, len(definition.dependencies))
	for _, dependency := range definition.dependencies {
		if !dependency.Destination().present() {
			return false
		}
		if _, duplicate := seen[dependency.Destination()]; duplicate {
			return false
		}
		seen[dependency.Destination()] = struct{}{}
	}
	return true
}

func (definition completionDeclaration) Kind() completionDeclarationKind { return definition.kind }

func (definition completionDeclaration) Dependencies() []dependencyRequirement {
	return append([]dependencyRequirement(nil), definition.dependencies...)
}

func (definition completionDeclaration) Source() semanticProducerSource { return definition.source }

func (definition completionDeclaration) Native() (CompletionNativeKind, CompletionNativeMode, bool) {
	return definition.nativeKind, definition.nativeMode, definition.hasNative
}
