package cli

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type flagAcceptanceInput struct {
	destinationIndex    int
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
}

type spellingDirectiveAcceptanceInput struct {
	kind          flagSpellingDirectiveKind
	forms         []FlagValueForm
	value         any
	sourceOptions []SourceOption
	lifecycle     ReleaseLifecycle
}

func staticSpellingDirectiveAcceptanceInput(value flagSpellingDirectiveDraft) (spellingDirectiveAcceptanceInput, bool) {
	if !value.valid {
		return spellingDirectiveAcceptanceInput{}, false
	}
	input := spellingDirectiveAcceptanceInput{
		kind:          value.kind,
		forms:         value.forms,
		value:         value.value,
		sourceOptions: value.sourceOptions,
		lifecycle:     value.lifecycle,
	}

	return input, true
}

func (input declarationPlan) flagInput(source *declarationDraft, record declarationPlanRecord) (flagAcceptanceInput, bool) {
	if source == nil || record.kind != declarationRecordFlag {
		return flagAcceptanceInput{}, false
	}
	value := source.flags.entries[record.familyIndex].value
	destinationIndex, ok := input.indexes.destinations[value.destination]
	if !value.valid || !ok {
		return flagAcceptanceInput{}, false
	}
	result := flagAcceptanceInput{
		destinationIndex:    destinationIndex,
		producer:            value.producer,
		completion:          snapshotAuthoringCompletion(value.completion),
		completionSet:       value.completionSet,
		tokenLanguage:       value.tokenLanguage,
		tokenCount:          value.tokenCount,
		occurrences:         value.occurrences,
		occurrencesSet:      value.occurrencesSet,
		valuesPerOccurrence: value.valuesPerOccurrence,
		valuesSet:           value.valuesSet,
		delimiter:           value.delimiter,
		delimiterSet:        value.delimiterSet,
		terminator:          value.terminator,
		terminatorSet:       value.terminatorSet,
		hyphenValues:        value.hyphenValues,
		hyphenValuesSet:     value.hyphenValuesSet,
		placement:           value.placement,
		placementSet:        value.placementSet,
		required:            value.required,
		requiredSet:         value.requiredSet,
		valueName:           value.valueName,
		valueNameSet:        value.valueNameSet,
		description:         value.description,
		descriptionSet:      value.descriptionSet,
		hidden:              value.hidden,
		hiddenSet:           value.hiddenSet,
		localeSelector:      value.localeSelector,
		localeSelectorSet:   value.localeSelectorSet,
		release:             value.release,
		releaseSet:          value.releaseSet,
	}
	if result.completionSet {
		frozen, complete := freezeAuthoringCompletion(source, result.completion)
		if !complete {
			return flagAcceptanceInput{}, false
		}
		result.completion = frozen
	}
	return result, true
}

func mutateFlag(session *declarationSession, ref flagRef, mutate func(*flagDraft)) {
	if session == nil || !ref.issued() {
		return
	}

	session.use(func(draft *declarationDraft, _ commandRef) {
		if !ref.ownedBy(draft.identity) {
			panic("cli: issued flag reference changed owner")
		}
		record, ok := draft.flags.resolve(draft, ref)
		if !ok {
			panic("cli: issued flag draft cannot resolve")
		}
		mutate(record)
	})
}

func (declaration FlagDeclaration[C, T]) destinationFor(*C, *T) {}

func (declaration FlagDeclaration[C, T]) destinationHandle() *destinationHandle {
	return declaration.destination.destinationHandle()
}

// FlagDeclaration identifies one staged flag and preserves its typed Destination. Spellings,
// conditions, and replacement relationships are configured through this declaration. The zero
// value is invalid; copies identify the same declaration, and retained, concurrent, or cross-scope
// use is builder lifecycle misuse.
type FlagDeclaration[C, T any] struct {
	state       *declarationSession
	destination Destination[C, T]
	flag        flagRef
}

// FlagSpellingDeclaration identifies one spelling of a staged flag. Value forms, implicit or fixed
// values, sensitivity, visibility, and replacement are configured through this declaration. The
// zero value is invalid; copies identify the same declaration, and retained, concurrent, or
// cross-scope use is builder lifecycle misuse.
type FlagSpellingDeclaration[C, T any] struct {
	state    *declarationSession
	spelling flagSpellingRef
}

// Destination returns the typed destination produced by this flag.
func (declaration FlagDeclaration[C, T]) Destination() Destination[C, T] {
	return declaration.destination
}

// AvailableWhen makes the flag structurally available only when condition holds.
func (declaration FlagDeclaration[C, T]) AvailableWhen(condition Condition) FlagDeclaration[C, T] {
	if !declaration.flag.valid() {
		panicBuilderMisuse()
	}
	stageAvailabilityCondition(declaration.state, grammarProfileCaseRef{}, authoringFlagConditionTarget{
		reference: declaration.flag,
	}, condition)
	return declaration
}

// Occurrences sets the accepted number of occurrences of this flag.
func (declaration FlagDeclaration[C, T]) Occurrences(count CountRange) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.occurrencesSet {
			flag.valid = false
			return
		}
		flag.occurrencesSet = true
		flag.occurrences = count
	})
	return declaration
}

// ValuesPerOccurrence sets the accepted value count for each sequence or keyed flag occurrence.
func (declaration FlagDeclaration[C, T]) ValuesPerOccurrence(count CountRange) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.valuesSet {
			flag.valid = false
			return
		}
		flag.valuesSet = true
		flag.valuesPerOccurrence = count
	})
	return declaration
}

// DelimitedBy enables delimiter-separated values for a sequence or keyed flag.
func (declaration FlagDeclaration[C, T]) DelimitedBy(delimiter string) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.delimiterSet {
			flag.valid = false
			return
		}
		flag.delimiterSet = true
		flag.delimiter = delimiter
	})
	return declaration
}

// TerminatedBy sets the token that terminates value collection for this flag.
func (declaration FlagDeclaration[C, T]) TerminatedBy(token string) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.terminatorSet {
			flag.valid = false
			return
		}
		flag.terminatorSet = true
		flag.terminator = token
	})
	return declaration
}

// HyphenValues sets how hyphen-prefixed tokens may be consumed as flag values.
func (declaration FlagDeclaration[C, T]) HyphenValues(mode HyphenValueMode) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.hyphenValuesSet {
			flag.valid = false
			return
		}
		flag.hyphenValuesSet = true
		flag.hyphenValues = mode
	})
	return declaration
}

// Placement sets whether the flag belongs to its declaring command or subtree.
func (declaration FlagDeclaration[C, T]) Placement(placement FlagPlacement) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.placementSet {
			flag.valid = false
			return
		}
		flag.placementSet = true
		flag.placement = placement
	})
	return declaration
}

// Required marks the flag as structurally required.
func (declaration FlagDeclaration[C, T]) Required() FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.requiredSet {
			flag.valid = false
			return
		}
		flag.requiredSet = true
		flag.required = true
	})
	return declaration
}

// ValueName sets the presentation name used for the flag value.
func (declaration FlagDeclaration[C, T]) ValueName(name string) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.valueNameSet {
			flag.valid = false
			return
		}
		flag.valueNameSet = true
		flag.valueName = name
	})
	return declaration
}

// Description sets presentation text for the flag.
func (declaration FlagDeclaration[C, T]) Description(text string) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.descriptionSet {
			flag.valid = false
			return
		}
		flag.descriptionSet = true
		flag.description = text
	})
	return declaration
}

// Hidden omits the flag from ordinary presentation.
func (declaration FlagDeclaration[C, T]) Hidden() FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.hiddenSet {
			flag.valid = false
			return
		}
		flag.hiddenSet = true
		flag.hidden = true
	})
	return declaration
}

// LocaleSelector marks this flag destination as the invocation locale selector.
func (declaration FlagDeclaration[C, T]) LocaleSelector() FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.localeSelectorSet {
			flag.valid = false
			return
		}
		flag.localeSelectorSet = true
		flag.localeSelector = true
	})
	return declaration
}

// Release attaches the declared release lifecycle to this flag.
func (declaration FlagDeclaration[C, T]) Release(lifecycle ReleaseLifecycle) FlagDeclaration[C, T] {
	mutateFlag(declaration.state, declaration.flag, func(flag *flagDraft) {
		if flag.releaseSet {
			flag.valid = false
			return
		}
		flag.releaseSet = true
		flag.release = lifecycle
	})
	return declaration
}

// Optional makes this positional term optional in its argument form.
func (declaration ArgumentDeclaration[C, T]) Optional() ArgumentDeclaration[C, T] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.optionalSet {
			term.valid = false
			return
		}
		term.optionalSet = true
		term.optional = true
	})
	return declaration
}

// FlagValueForm selects a concrete token form that a flag spelling accepts when it carries a
// value.
type FlagValueForm uint8

const (
	// FlagValueFormUnknown is the unspecified flag-value form.
	FlagValueFormUnknown FlagValueForm = iota
	// FlagValueFormSeparate accepts the flag value as a separate following token.
	FlagValueFormSeparate
	// FlagValueFormJoined accepts the flag value joined directly to the spelling.
	FlagValueFormJoined
	// FlagValueFormAssigned accepts the flag value in an assignment form.
	FlagValueFormAssigned
)

// FlagValueMode describes whether a flag spelling takes no value, requires a value, or accepts an
// optional value.
type FlagValueMode uint8

const (
	// FlagValueModeUnknown is the unspecified flag-value mode.
	FlagValueModeUnknown FlagValueMode = iota
	// FlagValueModeNone indicates that the spelling carries no explicit value.
	FlagValueModeNone
	// FlagValueModeRequired indicates that the spelling requires an explicit value.
	FlagValueModeRequired
	// FlagValueModeOptional indicates that the spelling may omit its explicit value.
	FlagValueModeOptional
)

// Flags declares typed flag destinations and spellings for one command. Each declaration returns a
// FlagDeclaration for spelling-specific configuration. The surface is callback-scoped; its zero
// value and retained, concurrent, or post-callback use are invalid.
type Flags[C any] struct {
	Config *C

	session   *declarationSession
	profile   grammarProfileCaseRef
	helpGroup helpGroupRef
}

// Bool declares a boolean flag destination using the built-in boolean language.
func (flags *Flags[C]) Bool(storage *bool) FlagDeclaration[C, bool] {
	return declareBoolFlag(flags, storage)
}

// Count declares a counting flag whose destination accumulates occurrences.
func (flags *Flags[C]) Count(storage *int) FlagDeclaration[C, int] {
	return declareCountFlag(flags, storage)
}

// Int declares an integer flag destination using the built-in integer language.
func (flags *Flags[C]) Int(storage *int) FlagDeclaration[C, int] {
	return declareIntFlag(flags, storage)
}

// Uint declares an unsigned-integer flag destination using the built-in unsigned language.
func (flags *Flags[C]) Uint(storage *uint) FlagDeclaration[C, uint] {
	return declareUintFlag(flags, storage)
}

// String declares a string flag destination using the built-in string language.
func (flags *Flags[C]) String(storage *string) FlagDeclaration[C, string] {
	return declareStringFlag(flags, storage)
}

// Strings declares a sequence of string flag values, preserving accepted value order.
func (flags *Flags[C]) Strings(storage *[]string) FlagDeclaration[C, []string] {
	return declareStringsFlag(flags, storage)
}

// Value declares a scalar flag destination using language.
func (flags *Flags[C]) Value[T any](storage *T, language ValueLanguage[T]) FlagDeclaration[C, T] {
	return declareValueFlag(flags, storage, language)
}

// ValueFor declares a scalar flag that contributes to an existing typed destination.
func (flags *Flags[C]) ValueFor[T any](destination destinationLike[C, T]) FlagDeclaration[C, T] {
	return declareAttachedFlag(flags, destination, flagProducerScalar, nil, CountRange{})
}

// Sequence declares a flag that contributes an ordered sequence to an existing slice destination.
func (flags *Flags[C]) Sequence[T any](destination destinationLike[C, []T]) FlagDeclaration[C, []T] {
	return declareAttachedFlag(flags, destination, flagProducerSequence, nil, CountRange{})
}

// Tokens declares a flag whose value is parsed from a TokenLanguage into an existing destination.
func (flags *Flags[C]) Tokens[T any](destination destinationLike[C, T], language TokenLanguage[T]) FlagDeclaration[C, T] {
	return declareTokenFlag(flags, destination, language)
}

// Keyed declares a flag that contributes keyed values to an existing map destination.
func (flags *Flags[C]) Keyed[K comparable, V any](destination destinationLike[C, map[K]V]) FlagDeclaration[C, map[K]V] {
	return declareAttachedFlag(flags, destination, flagProducerKeyed, nil, CountRange{})
}

// HelpGroup declares a titled group of flags through the callback; group membership follows
// declarations made by that callback.
func (flags *Flags[C]) HelpGroup(title string, declare func(*Flags[C])) helpGroupDeclaration {
	return declareFlagHelpGroup(flags, title, declare)
}

// Long adds a --name spelling to the flag and returns that spelling for spelling-specific
// configuration.
func (declaration FlagDeclaration[C, T]) Long(name string) FlagSpellingDeclaration[C, T] {
	return longFlagSpelling(declaration, name)
}

// Short adds a single-rune short spelling to the flag and returns that spelling for
// spelling-specific configuration.
func (declaration FlagDeclaration[C, T]) Short(name rune) FlagSpellingDeclaration[C, T] {
	return shortFlagSpelling(declaration, name)
}

// Literal adds the exact spelling to the flag and returns that spelling for spelling-specific
// configuration.
func (declaration FlagDeclaration[C, T]) Literal(spelling string) FlagSpellingDeclaration[C, T] {
	return literalFlagSpelling(declaration, spelling)
}

// ValueForms restricts this spelling to the supplied explicit flag-value forms.
func (spelling FlagSpellingDeclaration[C, T]) ValueForms(forms ...FlagValueForm) FlagSpellingDeclaration[C, T] {
	return configureFlagSpellingValueForms(spelling, forms...)
}

// FixedValue makes this spelling contribute value without parsing an explicit flag value.
func (spelling FlagSpellingDeclaration[C, T]) FixedValue(value T, options ...SourceOption) FlagSpellingDeclaration[C, T] {
	return configureFlagSpellingFixedValue(spelling, value, options...)
}

// ImplicitValue supplies value when this spelling is used without an explicit value where
// optional-value grammar permits it.
func (spelling FlagSpellingDeclaration[C, T]) ImplicitValue(value T, options ...SourceOption) FlagSpellingDeclaration[C, T] {
	return configureFlagSpellingImplicitValue(spelling, value, options...)
}

// Sensitive marks values supplied through this spelling as sensitive for downstream presentation
// and observation rules.
func (spelling FlagSpellingDeclaration[C, T]) Sensitive() FlagSpellingDeclaration[C, T] {
	return configureFlagSpellingSensitive(spelling)
}

// Hidden omits this spelling from ordinary presentation while preserving the spelling in grammar.
func (spelling FlagSpellingDeclaration[C, T]) Hidden() FlagSpellingDeclaration[C, T] {
	return configureFlagSpellingHidden(spelling)
}

func declareEmbeddedFlag[C, T any](flags *Flags[C], storage *T, producer flagProducerKind, destinationKind destinationDeclarationKind, language any) FlagDeclaration[C, T] {
	binding, bindingOK := bindConfigurationStorage(flags.Config, storage)
	destinationValue := destinationDraft{
		kind:     destinationKind,
		storage:  storage,
		language: language,
		binding:  binding,
		valid:    storage != nil && bindingOK,
	}
	if destinationKind == destinationDeclarationValue || destinationKind == destinationDeclarationSequence ||
		destinationKind == destinationDeclarationKeyed {
		destinationValue.valid = destinationValue.valid && language != nil
	}
	flagDraft := flagDraft{
		producer: producer,
		valid:    destinationValue.valid,
	}
	flag, destination := stageEmbeddedFlag(flags.session, flags.profile, flags.helpGroup, flagDraft, destinationValue)
	flagDeclaration := FlagDeclaration[C, T]{
		state: flags.session,
		destination: Destination[C, T]{
			handle: destination,
		},
		flag: flag,
	}

	return flagDeclaration
}

func declareAttachedFlag[C, T any](flags *Flags[C], destination destinationLike[C, T], producer flagProducerKind, tokenLanguage any, tokenCount CountRange) FlagDeclaration[C, T] {
	state := destinationHandleOf(destination)
	valid := state != nil && flags != nil && flags.session != nil
	flagDraft := flagDraft{
		producer:      producer,
		tokenLanguage: tokenLanguage,
		tokenCount:    tokenCount,
		valid:         valid,
	}
	flag, state := stageAttachedFlag(flags.session, flags.profile, flags.helpGroup, flagDraft, state)
	flagDeclaration := FlagDeclaration[C, T]{
		state: flags.session,
		destination: Destination[C, T]{
			handle: state,
		},
		flag: flag,
	}

	return flagDeclaration
}

func declareBoolFlag[C any](flags *Flags[C], storage *bool) FlagDeclaration[C, bool] {
	language, err := normalizeValueLanguage[bool](Value.Bool())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedFlag(flags, storage, flagProducerBool, destinationDeclarationValue, language)
}

func declareCountFlag[C any](flags *Flags[C], storage *int) FlagDeclaration[C, int] {
	binding, bindingOK := bindConfigurationStorage(flags.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationOpaque,
		storage:            storage,
		binding:            binding,
		definition:         acceptCountDestinationDefinition(),
		definitionComplete: true,
		valid:              storage != nil && bindingOK,
	}
	flagDraft := flagDraft{
		producer: flagProducerCount,
		valid:    destinationValue.valid,
	}
	flag, destination := stageEmbeddedFlag(flags.session, flags.profile, flags.helpGroup, flagDraft, destinationValue)
	flagDeclaration := FlagDeclaration[C, int]{
		state: flags.session,
		destination: Destination[C, int]{
			handle: destination,
		},
		flag: flag,
	}

	return flagDeclaration
}

func declareIntFlag[C any](flags *Flags[C], storage *int) FlagDeclaration[C, int] {
	language, err := normalizeValueLanguage[int](Value.Int())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedFlag(flags, storage, flagProducerScalar, destinationDeclarationValue, language)
}

func declareUintFlag[C any](flags *Flags[C], storage *uint) FlagDeclaration[C, uint] {
	language, err := normalizeValueLanguage[uint](Value.Uint())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedFlag(flags, storage, flagProducerScalar, destinationDeclarationValue, language)
}

func declareStringFlag[C any](flags *Flags[C], storage *string) FlagDeclaration[C, string] {
	language, err := normalizeValueLanguage[string](Value.String())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedFlag(flags, storage, flagProducerScalar, destinationDeclarationValue, language)
}

func declareStringsFlag[C any](flags *Flags[C], storage *[]string) FlagDeclaration[C, []string] {
	language, err := normalizeValueLanguage[string](Value.String())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedFlag(flags, storage, flagProducerSequence, destinationDeclarationSequence, language)
}

func declareValueFlag[C, T any](flags *Flags[C], storage *T, language ValueLanguage[T]) FlagDeclaration[C, T] {
	normalized, err := normalizeValueLanguage(language)
	if err != nil {
		return declareEmbeddedFlag(flags, storage, flagProducerScalar, destinationDeclarationValue, nil)
	}
	definition, definitionComplete, domainUses := scalarDestinationDefinition(language)
	binding, bindingOK := bindConfigurationStorage(flags.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationValue,
		storage:            storage,
		language:           normalized,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK,
	}
	flagDraft := flagDraft{
		producer: flagProducerScalar,
		valid:    destinationValue.valid,
	}
	flag, destination := stageEmbeddedFlag(flags.session, flags.profile, flags.helpGroup, flagDraft, destinationValue)
	flagDeclaration := FlagDeclaration[C, T]{
		state: flags.session,
		destination: Destination[C, T]{
			handle: destination,
		},
		flag: flag,
	}

	return flagDeclaration
}

func declareTokenFlag[C, T any](flags *Flags[C], destination destinationLike[C, T], language TokenLanguage[T]) FlagDeclaration[C, T] {
	normalized, err := normalizeTokenLanguage(language)
	if err != nil {
		return declareAttachedFlag(flags, destination, flagProducerTokens, nil, CountRange{})
	}
	return declareAttachedFlag(flags, destination, flagProducerTokens, normalized, normalized.count)
}

func declareFlagSpelling[C, T any](flag FlagDeclaration[C, T], kind flagSpellingKind, text string, valid bool) FlagSpellingDeclaration[C, T] {
	spellingValue := flagSpellingDraft{
		kind:  kind,
		text:  text,
		valid: valid,
	}
	spelling := stageFlagSpelling(flag.state, flag.flag, spellingValue)
	return FlagSpellingDeclaration[C, T]{
		state:    flag.state,
		spelling: spelling,
	}
}

func longFlagSpelling[C, T any](flag FlagDeclaration[C, T], name string) FlagSpellingDeclaration[C, T] {
	return declareFlagSpelling(flag, flagSpellingLong, "--"+name, validLongFlagName("--"+name))
}

func shortFlagSpelling[C, T any](flag FlagDeclaration[C, T], name rune) FlagSpellingDeclaration[C, T] {
	text := "-" + string(name)
	return declareFlagSpelling(flag, flagSpellingShort, text, validShortFlagText(text))
}

func literalFlagSpelling[C, T any](flag FlagDeclaration[C, T], spelling string) FlagSpellingDeclaration[C, T] {
	return declareFlagSpelling(flag, flagSpellingLiteral, spelling, validLiteralFlagSpelling(spelling))
}

func configureFlagSpellingValueForms[C, T any](spelling FlagSpellingDeclaration[C, T], forms ...FlagValueForm) FlagSpellingDeclaration[C, T] {
	directive := flagSpellingDirectiveDraft{
		kind:  flagSpellingDirectiveValueForms,
		forms: append([]FlagValueForm(nil), forms...),
		valid: true,
	}
	stageFlagSpellingDirective(spelling.state, spelling.spelling, directive)
	return spelling
}

func configureFlagSpellingFixedValue[C, T any](spelling FlagSpellingDeclaration[C, T], value T, options ...SourceOption) FlagSpellingDeclaration[C, T] {
	directive := flagSpellingDirectiveDraft{
		kind:          flagSpellingDirectiveFixedValue,
		value:         value,
		sourceOptions: append([]SourceOption(nil), options...),
		valid:         true,
	}
	stageFlagSpellingDirective(spelling.state, spelling.spelling, directive)
	return spelling
}

func configureFlagSpellingImplicitValue[C, T any](spelling FlagSpellingDeclaration[C, T], value T, options ...SourceOption) FlagSpellingDeclaration[C, T] {
	directive := flagSpellingDirectiveDraft{
		kind:          flagSpellingDirectiveImplicitValue,
		value:         value,
		sourceOptions: append([]SourceOption(nil), options...),
		valid:         true,
	}
	stageFlagSpellingDirective(spelling.state, spelling.spelling, directive)
	return spelling
}

func configureFlagSpellingSensitive[C, T any](spelling FlagSpellingDeclaration[C, T]) FlagSpellingDeclaration[C, T] {
	stageFlagSpellingDirective(spelling.state, spelling.spelling, flagSpellingDirectiveDraft{
		kind:  flagSpellingDirectiveSensitive,
		valid: true,
	})
	return spelling
}

func configureFlagSpellingHidden[C, T any](spelling FlagSpellingDeclaration[C, T]) FlagSpellingDeclaration[C, T] {
	stageFlagSpellingDirective(spelling.state, spelling.spelling, flagSpellingDirectiveDraft{
		kind:  flagSpellingDirectiveHidden,
		valid: true,
	})
	return spelling
}

func configureFlagSpellingRelease[C, T any](spelling FlagSpellingDeclaration[C, T], lifecycle ReleaseLifecycle) FlagSpellingDeclaration[C, T] {
	directive := flagSpellingDirectiveDraft{
		kind:      flagSpellingDirectiveRelease,
		lifecycle: lifecycle,
		valid:     true,
	}
	stageFlagSpellingDirective(spelling.state, spelling.spelling, directive)
	return spelling
}

func validLongFlagName(text string) bool {
	if !strings.HasPrefix(text, "--") || len(text) <= 2 {
		return false
	}
	name := text[2:]
	if strings.HasPrefix(name, "-") || strings.ContainsRune(name, '\x00') || strings.ContainsRune(name, '=') {
		return false
	}
	return strings.IndexFunc(name, unicode.IsSpace) < 0
}

func validShortFlagText(text string) bool {
	if !strings.HasPrefix(text, "-") || strings.HasPrefix(text, "--") {
		return false
	}
	name := text[1:]
	r, size := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError && size == 0 || size != len(name) || r == 0 || r == '-' || r == '=' || unicode.IsSpace(r) {
		return false
	}
	return true
}

func validLiteralFlagSpelling(text string) bool {
	return text != "" && !strings.ContainsRune(text, '\x00')
}

type flagProducerKind uint8

const (
	flagProducerInvalid flagProducerKind = iota
	flagProducerBool
	flagProducerCount
	flagProducerScalar
	flagProducerSequence
	flagProducerTokens
	flagProducerKeyed
)

type flagSpellingKind uint8

const (
	flagSpellingInvalid flagSpellingKind = iota
	flagSpellingLong
	flagSpellingShort
	flagSpellingLiteral
)

type flagSpellingDirectiveKind uint8

const (
	flagSpellingDirectiveInvalid flagSpellingDirectiveKind = iota
	flagSpellingDirectiveValueForms
	flagSpellingDirectiveFixedValue
	flagSpellingDirectiveImplicitValue
	flagSpellingDirectiveSensitive
	flagSpellingDirectiveHidden
	flagSpellingDirectiveRelease
)

func stageFlagOnDraft(draft *declarationDraft, command commandRef, profile grammarProfileCaseRef, helpGroup helpGroupRef, position declarationPosition, flag flagDraft, destination destinationRef) flagRef {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	flag.position = position
	flag.command = command
	flag.profile = profile
	flag.helpGroup = helpGroup
	flag.destination = destination
	flag.completion = snapshotAuthoringCompletion(flag.completion)
	ref := draft.flags.append(draft, flag)
	if !flag.valid {
		draft.reject(position, errInvalidAuthoringTransaction)
	}
	return ref
}

func stageAttachedFlag(session *declarationSession, profile grammarProfileCaseRef, helpGroup helpGroupRef, flag flagDraft, destination *destinationHandle) (flagRef, *destinationHandle) {
	if session == nil {
		return flagRef{}, destination
	}

	var ref flagRef
	session.use(func(draft *declarationDraft, command commandRef) {
		valid := flag.valid
		var destinationRef destinationRef
		if destination == nil {
			valid = false
		} else {
			identity, ok := destination.currentIdentity().(stagedDestinationIdentity)
			if !ok || !identity.ref.ownedBy(draft.identity) {
				valid = false
			} else {
				destinationRef = identity.ref
			}
		}
		flag.valid = valid
		ref = stageFlagOnDraft(draft, command, profile, helpGroup, draft.nextPosition(), flag, destinationRef)
	})
	return ref, destination
}

func stageEmbeddedFlag(session *declarationSession, profile grammarProfileCaseRef, helpGroup helpGroupRef, flag flagDraft, destination destinationDraft) (flagRef, *destinationHandle) {
	if session == nil {
		return flagRef{}, nil
	}

	var ref flagRef
	var handle *destinationHandle
	session.use(func(draft *declarationDraft, command commandRef) {
		position := draft.nextPosition()
		valid := flag.valid
		handle = stageDestinationRecordOnDraft(session, draft, command, profile, position, destination)
		var destinationRef destinationRef
		if handle == nil {
			valid = false
		} else if identity, ok := handle.currentIdentity().(stagedDestinationIdentity); ok {
			destinationRef = identity.ref
		} else {
			valid = false
		}
		flag.valid = valid
		ref = stageFlagOnDraft(draft, command, profile, helpGroup, position, flag, destinationRef)
	})
	return ref, handle
}

func stageFlagSpelling(session *declarationSession, flag flagRef, spellingValue flagSpellingDraft) flagSpellingRef {
	if session == nil || !flag.valid() {
		return flagSpellingRef{}
	}

	var ref flagSpellingRef
	session.use(func(draft *declarationDraft, _ commandRef) {
		valid := spellingValue.valid && flag.ownedBy(draft.identity)
		record := flagSpellingDraft{
			position: draft.nextPosition(),
			flag:     flag,
			kind:     spellingValue.kind,
			text:     spellingValue.text,
			valid:    valid,
		}
		ref = draft.flagSpellings.append(draft, record)
	})
	return ref
}

func stageFlagSpellingDirective(session *declarationSession, spelling flagSpellingRef, directive flagSpellingDirectiveDraft) {
	if session == nil || !spelling.valid() {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		spellingRecord, ok := draft.flagSpellings.resolve(draft, spelling)
		if !ok {
			return
		}
		flagRecord, flagOK := draft.flags.resolve(draft, spellingRecord.flag)
		profile := grammarProfileCaseRef{}
		if flagOK {
			profile = flagRecord.profile
		}
		record := flagSpellingDirectiveDraft{
			position:      draft.nextPosition(),
			command:       command,
			profile:       profile,
			spelling:      spelling,
			kind:          directive.kind,
			forms:         append([]FlagValueForm(nil), directive.forms...),
			value:         snapshotMutableValue(directive.value),
			sourceOptions: append([]SourceOption(nil), directive.sourceOptions...),
			lifecycle:     directive.lifecycle,
			valid:         directive.valid && flagOK,
		}
		draft.flagSpellingDirectives.append(draft, record)
	})
}

func collectStaticSpellingConfigs(input declarationPlan, source *declarationDraft, state *staticAcceptanceState) error {
	if state == nil || len(state.slots) != len(input.records) {
		panic("cli: invalid static spelling acceptance state")
	}
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordFlagSpellingDirective {
			continue
		}
		directive := source.flagSpellingDirectives.entries[record.familyIndex].value
		spellingIndex, ok := input.indexes.flagSpellings[directive.spelling]
		if !ok {
			panic("cli: validated flag-spelling directive has no spelling record")
		}
		accepted, ok := staticSpellingDirectiveAcceptanceInput(directive)
		if !ok {
			return errStaticDeclarationRejected
		}
		config, ok := applySpellingDirective(state.slots[spellingIndex].spellingConfig, accepted)
		if !ok {
			return errStaticDeclarationRejected
		}
		state.slots[spellingIndex].spellingConfig = config
	}
	return nil
}

type flagValueForm uint8

const (
	flagValueFormInvalid flagValueForm = iota
	flagValueFormSeparate
	flagValueFormJoined
	flagValueFormAssigned
)

// HyphenValueMode controls ownership of otherwise-unrecognized hyphen-leading
// separate values while ordinary flag recognition is still active. Exact known
// flags and active end-of-options tokens remain hard lexical boundaries.

// semanticArgumentAllocationPreference resolves otherwise-ambiguous structural
// allocations in declaration order. Unspecified intentionally remains the zero
// value because no preference is the normal grammar default.
type semanticArgumentAllocationPreference uint8

const (
	semanticArgumentAllocationUnspecified semanticArgumentAllocationPreference = iota
	semanticArgumentAllocationGreedy
	semanticArgumentAllocationMinimal
)

type flagSpellingGrammar struct {
	id                FlagSpellingID
	flag              FlagID
	kind              FlagSpellingKind
	text              string
	forms             []flagValueForm
	mode              FlagValueMode
	sensitive         bool
	fixedValue        any
	hasFixedValue     bool
	fixedSensitive    bool
	fixedSnapshot     func(any) (any, bool)
	implicitValue     any
	hasImplicitValue  bool
	implicitSensitive bool
	implicitSnapshot  func(any) (any, bool)
	position          declarationPosition
}

// flagSpellingGrammarInput is the complete immutable payload of one flag spelling grammar.
type flagSpellingGrammarInput struct {
	ID        FlagSpellingID
	Flag      FlagID
	Kind      FlagSpellingKind
	Text      string
	Mode      FlagValueMode
	Forms     []flagValueForm
	Sensitive bool
	Position  declarationPosition
}

func validFlagSpellingGrammarInput(input flagSpellingGrammarInput) bool {
	if !input.ID.present() || !input.Flag.present() || !validFlagSpellingText(input.Kind, input.Text) ||
		input.Mode == FlagValueModeUnknown {
		return false
	}
	if input.Mode == FlagValueModeNone {
		if len(input.Forms) != 0 {
			return false
		}
	} else if len(input.Forms) == 0 {
		return false
	}
	seen := map[flagValueForm]struct{}{}
	for _, form := range input.Forms {
		if form == flagValueFormInvalid {
			return false
		}
		if _, exists := seen[form]; exists {
			return false
		}
		seen[form] = struct{}{}
	}
	return true
}

func validFlagSpellingText(kind FlagSpellingKind, text string) bool {
	switch kind {
	case FlagSpellingLong:
		if !strings.HasPrefix(text, "--") || len(text) <= 2 {
			return false
		}
		name := text[2:]
		return !strings.HasPrefix(name, "-") && !strings.ContainsRune(name, '\x00') &&
			!strings.ContainsRune(name, '=') && strings.IndexFunc(name, unicode.IsSpace) < 0
	case FlagSpellingShort:
		if !strings.HasPrefix(text, "-") || strings.HasPrefix(text, "--") {
			return false
		}
		name := text[1:]
		r, size := utf8.DecodeRuneInString(name)
		return (r != utf8.RuneError || size != 0) && size == len(name) && r != 0 && r != '-' && r != '=' &&
			!unicode.IsSpace(r)
	case FlagSpellingLiteral:
		return text != "" && !strings.ContainsRune(text, '\x00')
	default:
		return false
	}
}

func acceptFlagSpellingGrammarWithKind(input flagSpellingGrammarInput) flagSpellingGrammar {
	if !validFlagSpellingGrammarInput(input) {
		panic("semantic: invalid flag-spelling grammar")
	}
	copied := append([]flagValueForm(nil), input.Forms...)
	grammar := flagSpellingGrammar{
		id:        input.ID,
		flag:      input.Flag,
		kind:      input.Kind,
		text:      input.Text,
		forms:     copied,
		mode:      input.Mode,
		sensitive: input.Sensitive,
		position:  input.Position,
	}

	return grammar
}

func (spelling flagSpellingGrammar) Forms() []flagValueForm {
	return append([]flagValueForm(nil), spelling.forms...)
}

func (spelling flagSpellingGrammar) FixedValue() (any, bool, bool) {
	if !spelling.hasFixedValue || spelling.fixedSnapshot == nil {
		return nil, false, spelling.fixedSensitive
	}
	value, ok := spelling.fixedSnapshot(spelling.fixedValue)
	return value, ok, spelling.fixedSensitive
}

func (spelling flagSpellingGrammar) ImplicitValue() (any, bool, bool) {
	if !spelling.hasImplicitValue || spelling.implicitSnapshot == nil {
		return nil, false, spelling.implicitSensitive
	}
	value, ok := spelling.implicitSnapshot(spelling.implicitValue)
	return value, ok, spelling.implicitSensitive
}

// WithFixedValue returns spelling with one already-typed fixed value captured
// through the destination's snapshot policy.

// WithImplicitValue returns spelling with one already-typed implicit value
// captured through the destination's snapshot policy.

func (spelling flagSpellingGrammar) Allows(form flagValueForm) bool {
	return slices.Contains(spelling.forms, form)
}

type flagGrammar struct {
	id                  FlagID
	command             CommandID
	destination         ValueID
	profile             GrammarProfileID
	disabled            bool
	placement           FlagPlacement
	occurrences         semanticCountRange
	valuesPerOccurrence semanticCountRange
	delimiter           string
	terminator          string
	hyphenValues        HyphenValueMode
	tokenLanguageName   string
	tokenLanguage       tokenLanguage
	spellings           []flagSpellingGrammar
	position            declarationPosition
}

type flagGrammarInput struct {
	ID                  FlagID
	Command             CommandID
	Destination         ValueID
	Placement           FlagPlacement
	Occurrences         semanticCountRange
	ValuesPerOccurrence semanticCountRange
	Delimiter           string
	Terminator          string
	HyphenValues        HyphenValueMode
	TokenLanguageName   string
	TokenLanguage       tokenLanguage
	Spellings           []flagSpellingGrammar
	Position            declarationPosition
}

func validFlagGrammarInput(input flagGrammarInput) bool {
	if !input.ID.present() || !input.Command.present() || !input.Destination.present() ||
		input.Placement == FlagPlacementUnknown || !input.Occurrences.valid() ||
		!input.ValuesPerOccurrence.valid() || len(input.Spellings) == 0 {
		return false
	}
	hyphen := input.HyphenValues
	if hyphen == HyphenValueUnknown {
		hyphen = HyphenValueDisallow
	}
	if hyphen != HyphenValueDisallow && hyphen != HyphenValueNegativeNumbers && hyphen != HyphenValueAllow {
		return false
	}
	if input.Delimiter != "" && input.ValuesPerOccurrence.Minimum() < 1 {
		return false
	}
	for _, spelling := range input.Spellings {
		if spelling.flag != input.ID {
			return false
		}
	}
	return !input.TokenLanguage.Valid() || input.TokenLanguageName == "" ||
		input.TokenLanguage.Name() == input.TokenLanguageName
}

func acceptFlagGrammar(input flagGrammarInput) flagGrammar {
	if !validFlagGrammarInput(input) {
		panic("semantic: invalid flag grammar")
	}
	hyphen := input.HyphenValues
	if hyphen == HyphenValueUnknown {
		hyphen = HyphenValueDisallow
	}
	spellings := append([]flagSpellingGrammar(nil), input.Spellings...)
	grammar := flagGrammar{
		id:                  input.ID,
		command:             input.Command,
		destination:         input.Destination,
		placement:           input.Placement,
		occurrences:         input.Occurrences,
		valuesPerOccurrence: input.ValuesPerOccurrence,
		delimiter:           input.Delimiter,
		terminator:          input.Terminator,
		hyphenValues:        hyphen,
		tokenLanguageName:   input.TokenLanguageName,
		tokenLanguage:       input.TokenLanguage,
		spellings:           spellings,
		position:            input.Position,
	}

	return grammar
}

func (flag flagGrammar) Enabled() bool { return !flag.disabled }

func (flag flagGrammar) Delimiter() (string, bool) { return flag.delimiter, flag.delimiter != "" }

func (flag flagGrammar) Terminator() (string, bool) { return flag.terminator, flag.terminator != "" }

func (flag flagGrammar) TokenLanguage() (tokenLanguage, bool) {
	return flag.tokenLanguage, flag.tokenLanguage.Valid()
}

func (flag flagGrammar) Spellings() []flagSpellingGrammar {
	return append([]flagSpellingGrammar(nil), flag.spellings...)
}

// flagRecord preserves exact flag declaration identity separately from the
// destination identity that owns value semantics.
type flagRecord struct {
	meta           recordMeta[FlagID]
	command        CommandID
	destination    ValueID
	presentation   presentation
	valueName      string
	required       bool
	localeSelector bool
	lifecycle      ReleaseLifecycle
	completion     completionDeclaration
}

func acceptFlagRecord(id FlagID, command CommandID, destination ValueID, position declarationPosition) flagRecord {
	mustCommand(command)
	mustDestination(destination)
	record := flagRecord{
		meta:        acceptRecordMeta(id, position),
		command:     command,
		destination: destination,
	}

	return record
}

func (record flagRecord) Completion() (completionDeclaration, bool) {
	return record.completion, record.completion.Valid()
}

type flagSpellingRecord struct {
	meta      recordMeta[FlagSpellingID]
	flag      FlagID
	hidden    bool
	lifecycle ReleaseLifecycle
}

func acceptFlagSpellingRecord(id FlagSpellingID, flag FlagID, position declarationPosition) flagSpellingRecord {
	if !flag.present() {
		panic("semantic: invalid flag identity")
	}
	return flagSpellingRecord{
		meta: acceptRecordMeta(id, position),
		flag: flag,
	}
}
