package cli

type argumentTermAcceptanceInput struct {
	formIndex          int
	destinationIndex   int
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
}

func (input declarationPlan) argumentTermInput(source *declarationDraft, record declarationPlanRecord) (argumentTermAcceptanceInput, bool) {
	if source == nil || record.kind != declarationRecordArgumentTerm {
		return argumentTermAcceptanceInput{}, false
	}
	value := source.argumentTerms.entries[record.familyIndex].value
	formIndex, formOK := input.indexes.argumentForms[value.form]
	destinationIndex := -1
	destinationOK := value.kind == argumentTermLiteral
	if value.destination.valid() {
		destinationIndex, destinationOK = input.indexes.destinations[value.destination]
	}
	if !value.valid || !formOK || !destinationOK {
		return argumentTermAcceptanceInput{}, false
	}
	result := argumentTermAcceptanceInput{
		formIndex:          formIndex,
		destinationIndex:   destinationIndex,
		kind:               value.kind,
		name:               value.name,
		count:              value.count,
		tokenLanguage:      value.tokenLanguage,
		literal:            value.literal,
		completion:         snapshotAuthoringCompletion(value.completion),
		completionSet:      value.completionSet,
		optional:           value.optional,
		optionalSet:        value.optionalSet,
		allocation:         value.allocation,
		allocationSet:      value.allocationSet,
		hyphenValues:       value.hyphenValues,
		hyphenValuesSet:    value.hyphenValuesSet,
		description:        value.description,
		descriptionSet:     value.descriptionSet,
		hidden:             value.hidden,
		hiddenSet:          value.hiddenSet,
		release:            value.release,
		releaseSet:         value.releaseSet,
		sourceSensitive:    value.sourceSensitive,
		sourceSensitiveSet: value.sourceSensitiveSet,
		endOfOptions:       value.endOfOptions,
		endOfOptionsSet:    value.endOfOptionsSet,
	}
	if result.completionSet {
		frozen, complete := freezeAuthoringCompletion(source, result.completion)
		if !complete {
			return argumentTermAcceptanceInput{}, false
		}
		result.completion = frozen
	}
	return result, true
}

func argumentSourceSensitive(value argumentTermAcceptanceInput) bool {
	return value.sourceSensitive
}

// Arguments declares positional grammar for one command. Declarations made directly on Arguments
// belong to the implicit argument form; Form opens an explicit form. It is valid only during the
// owning command declaration callback; its zero value and retained, concurrent, or post-callback
// use are invalid.
type Arguments[C any] struct {
	Config *C

	session   *declarationSession
	profile   grammarProfileCaseRef
	authoring *argumentAuthoringState
}

// ArgumentFormBuilder declares the terms of one explicit positional argument form. Config is the
// owning command configuration. The builder is callback-scoped; its zero value and retained,
// concurrent, or post-callback use are invalid.
type ArgumentFormBuilder[C any] struct {
	Config *C

	session *declarationSession
	profile grammarProfileCaseRef
	form    argumentFormDraftRef
}

// Form starts one explicit positional argument form. The callback is authoring-scoped, and its
// terms are recorded in declaration order.
func (arguments *Arguments[C]) Form(declare func(*ArgumentFormBuilder[C])) ArgumentFormDeclaration[C] {
	return declareArgumentForm(arguments, declare)
}

// Value declares a positional value parsed by language and bound to storage.
func (arguments *Arguments[C]) Value[T any](storage *T, name string, language ValueLanguage[T]) ArgumentDeclaration[C, T] {
	return declareArgumentValue(implicitArgumentForm(arguments), storage, name, language)
}

// ValueFor declares a positional value that contributes to an existing typed destination.
func (arguments *Arguments[C]) ValueFor[T any](destination destinationLike[C, T], name string) ArgumentDeclaration[C, T] {
	return declareArgumentValueFor(implicitArgumentForm(arguments), destination, name)
}

// Sequence declares a positional sequence whose accepted element count is constrained by count and
// whose values preserve input order.
func (arguments *Arguments[C]) Sequence[T any](destination destinationLike[C, []T], name string, count CountRange) ArgumentDeclaration[C, []T] {
	return declareArgumentSequence(implicitArgumentForm(arguments), destination, name, count)
}

// Tokens declares a positional value parsed from a TokenLanguage.
func (arguments *Arguments[C]) Tokens[T any](destination destinationLike[C, T], name string, language TokenLanguage[T]) ArgumentDeclaration[C, T] {
	return declareArgumentTokens(implicitArgumentForm(arguments), destination, name, language)
}

// Literal declares a fixed positional token that produces no destination value.
func (arguments *Arguments[C]) Literal(token string) ArgumentLiteralDeclaration[C] {
	return declareArgumentLiteral(implicitArgumentForm(arguments), token)
}

// Remainder declares a positional term that captures remaining tokens into the destination,
// subject to count.
func (arguments *Arguments[C]) Remainder(destination destinationLike[C, []string], name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareArgumentRemainder(implicitArgumentForm(arguments), destination, name, count)
}

// PassThrough declares a positional term that passes consumed tokens through to the destination,
// subject to count.
func (arguments *Arguments[C]) PassThrough(destination destinationLike[C, []string], name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareArgumentPassThrough(implicitArgumentForm(arguments), destination, name, count)
}

// String declares a string positional value using the built-in string language.
func (arguments *Arguments[C]) String(storage *string, name string) ArgumentDeclaration[C, string] {
	return declareArgumentString(implicitArgumentForm(arguments), storage, name)
}

// Int declares an integer positional value using the built-in integer language.
func (arguments *Arguments[C]) Int(storage *int, name string) ArgumentDeclaration[C, int] {
	return declareArgumentInt(implicitArgumentForm(arguments), storage, name)
}

// Strings declares an ordered sequence of string positional values constrained by count.
func (arguments *Arguments[C]) Strings(storage *[]string, name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareArgumentStrings(implicitArgumentForm(arguments), storage, name, count)
}

// Value declares a positional value parsed by language and bound to storage.
func (form *ArgumentFormBuilder[C]) Value[T any](storage *T, name string, language ValueLanguage[T]) ArgumentDeclaration[C, T] {
	return declareArgumentValue(form, storage, name, language)
}

// ValueFor declares a positional value that contributes to an existing typed destination.
func (form *ArgumentFormBuilder[C]) ValueFor[T any](destination destinationLike[C, T], name string) ArgumentDeclaration[C, T] {
	return declareArgumentValueFor(form, destination, name)
}

// Sequence declares a positional sequence whose accepted element count is constrained by count and
// whose values preserve input order.
func (form *ArgumentFormBuilder[C]) Sequence[T any](destination destinationLike[C, []T], name string, count CountRange) ArgumentDeclaration[C, []T] {
	return declareArgumentSequence(form, destination, name, count)
}

// Tokens declares a positional value parsed from a TokenLanguage.
func (form *ArgumentFormBuilder[C]) Tokens[T any](destination destinationLike[C, T], name string, language TokenLanguage[T]) ArgumentDeclaration[C, T] {
	return declareArgumentTokens(form, destination, name, language)
}

// Literal declares a fixed positional token that produces no destination value.
func (form *ArgumentFormBuilder[C]) Literal(token string) ArgumentLiteralDeclaration[C] {
	return declareArgumentLiteral(form, token)
}

// Remainder declares a positional term that captures remaining tokens into the destination,
// subject to count.
func (form *ArgumentFormBuilder[C]) Remainder(destination destinationLike[C, []string], name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareArgumentRemainder(form, destination, name, count)
}

// PassThrough declares a positional term that passes consumed tokens through to the destination,
// subject to count.
func (form *ArgumentFormBuilder[C]) PassThrough(destination destinationLike[C, []string], name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareArgumentPassThrough(form, destination, name, count)
}

// String declares a string positional value using the built-in string language.
func (form *ArgumentFormBuilder[C]) String(storage *string, name string) ArgumentDeclaration[C, string] {
	return declareArgumentString(form, storage, name)
}

// Int declares an integer positional value using the built-in integer language.
func (form *ArgumentFormBuilder[C]) Int(storage *int, name string) ArgumentDeclaration[C, int] {
	return declareArgumentInt(form, storage, name)
}

// Strings declares an ordered sequence of string positional values constrained by count.
func (form *ArgumentFormBuilder[C]) Strings(storage *[]string, name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareArgumentStrings(form, storage, name, count)
}

// HyphenValueMode selects how a value-producing flag or positional argument may consume tokens
// that begin with a hyphen.
type HyphenValueMode uint8

const (
	// HyphenValueUnknown is the unspecified hyphen-value mode and is not a valid explicit policy.
	HyphenValueUnknown HyphenValueMode = iota
	// HyphenValueDisallow prevents a value-producing term from consuming hyphen-prefixed tokens as
	// values.
	HyphenValueDisallow
	// HyphenValueNegativeNumbers permits hyphen-prefixed tokens only when they are valid negative
	// numbers for the supported signed-integer language.
	HyphenValueNegativeNumbers
	// HyphenValueAllow permits hyphen-prefixed tokens to be consumed as values.
	HyphenValueAllow
)

type argumentAuthoringState struct {
	mode         argumentAuthoringMode
	implicitForm argumentFormDraftRef
}

type argumentAuthoringMode uint8

const (
	argumentAuthoringModeUnset argumentAuthoringMode = iota
	argumentAuthoringModeImplicit
	argumentAuthoringModeExplicit
)

func declareArgumentForm[C any](arguments *Arguments[C], declare func(*ArgumentFormBuilder[C])) ArgumentFormDeclaration[C] {
	if arguments == nil || arguments.session == nil || arguments.authoring == nil {
		return ArgumentFormDeclaration[C]{}
	}

	validMode := arguments.authoring.mode != argumentAuthoringModeImplicit
	if arguments.authoring.mode == argumentAuthoringModeUnset {
		arguments.authoring.mode = argumentAuthoringModeExplicit
	}

	var form argumentFormDraftRef
	arguments.session.use(func(draft *declarationDraft, command commandRef) {
		form = stageArgumentFormOnDraft(arguments.session, arguments.profile, draft, command, declare != nil && validMode)
	})

	result := ArgumentFormDeclaration[C]{
		state: arguments.session,
		form:  form,
	}
	if declare == nil {
		return result
	}

	draft, command, resume := arguments.session.suspend()
	defer resume()

	contribution := draft.beginContribution()
	defer contribution.rollback()

	child := arguments.session.prepareChild(command)
	builder := ArgumentFormBuilder[C]{
		Config:  arguments.Config,
		session: child,
		profile: arguments.profile,
		form:    form,
	}
	if cause := child.invoke(func() {
		declare(&builder)
	}); cause != nil {
		abortBuilder(cause)
	}
	if contribution.rejected() {
		return result
	}

	contribution.commit()
	return result
}

func implicitArgumentForm[C any](arguments *Arguments[C]) *ArgumentFormBuilder[C] {
	if arguments == nil || arguments.session == nil || arguments.authoring == nil {
		return nil
	}

	if !arguments.authoring.implicitForm.valid() {
		validMode := arguments.authoring.mode != argumentAuthoringModeExplicit
		if arguments.authoring.mode == argumentAuthoringModeUnset {
			arguments.authoring.mode = argumentAuthoringModeImplicit
		}
		arguments.session.use(func(draft *declarationDraft, command commandRef) {
			arguments.authoring.implicitForm = stageArgumentFormOnDraft(arguments.session, arguments.profile, draft, command, validMode)
		})
	}

	builder := ArgumentFormBuilder[C]{
		Config:  arguments.Config,
		session: arguments.session,
		profile: arguments.profile,
		form:    arguments.authoring.implicitForm,
	}

	return &builder
}

func declareAttachedArgument[C, T any](form *ArgumentFormBuilder[C], destination destinationLike[C, T], kind argumentTermKind, name string, count CountRange, tokenLanguage any) ArgumentDeclaration[C, T] {
	handle := destinationHandleOf(destination)
	valid := form != nil && form.session != nil && form.form.valid() && handle != nil && count.valid
	session := formState(form)
	formRef := formRecord(form)
	termDraft := argumentTermDraft{
		kind:          kind,
		name:          name,
		count:         count,
		tokenLanguage: tokenLanguage,
		valid:         valid,
	}
	term, handle := stageAttachedArgumentTerm(session, form.profile, formRef, termDraft, handle)
	returnedDestination := Destination[C, T]{
		handle: handle,
	}
	declaration := ArgumentDeclaration[C, T]{
		state:       session,
		destination: returnedDestination,
		term:        term,
	}
	return declaration
}

func declareEmbeddedArgument[C, T any](form *ArgumentFormBuilder[C], storage *T, kind argumentTermKind, destinationKind destinationDeclarationKind, language any, name string, count CountRange) ArgumentDeclaration[C, T] {
	var bindingOK bool
	var binding configurationBinding
	if form != nil {
		binding, bindingOK = bindConfigurationStorage(form.Config, storage)
	}
	destination := destinationDraft{
		kind:     destinationKind,
		storage:  storage,
		language: language,
		binding:  binding,
		valid:    storage != nil && bindingOK,
	}
	valid := form != nil && form.session != nil && form.form.valid() && count.valid && validEmbeddedDestinationDraft(destination)
	session := formState(form)
	formRef := formRecord(form)
	termDraft := argumentTermDraft{
		kind:  kind,
		name:  name,
		count: count,
		valid: valid,
	}
	term, handle := stageEmbeddedArgumentTerm(session, form.profile, formRef, termDraft, destination)
	returnedDestination := Destination[C, T]{
		handle: handle,
	}
	declaration := ArgumentDeclaration[C, T]{
		state:       session,
		destination: returnedDestination,
		term:        term,
	}
	return declaration
}

func declareArgumentValue[C, T any](form *ArgumentFormBuilder[C], storage *T, name string, language ValueLanguage[T]) ArgumentDeclaration[C, T] {
	normalized, err := normalizeValueLanguage(language)
	if err != nil {
		return declareEmbeddedArgument(form, storage, argumentTermValue, destinationDeclarationValue, nil, name, Count.Exactly(1))
	}
	definition, definitionComplete, domainUses := scalarDestinationDefinition(language)
	var binding configurationBinding
	var bindingOK bool
	if form != nil {
		binding, bindingOK = bindConfigurationStorage(form.Config, storage)
	}
	destination := destinationDraft{
		kind:               destinationDeclarationValue,
		storage:            storage,
		language:           normalized,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK,
	}
	valid := form != nil && form.session != nil && form.form.valid() && validEmbeddedDestinationDraft(destination)
	session := formState(form)
	formRef := formRecord(form)
	termDraft := argumentTermDraft{
		kind:  argumentTermValue,
		name:  name,
		count: Count.Exactly(1),
		valid: valid,
	}
	term, handle := stageEmbeddedArgumentTerm(session, form.profile, formRef, termDraft, destination)
	returnedDestination := Destination[C, T]{
		handle: handle,
	}
	declaration := ArgumentDeclaration[C, T]{
		state:       session,
		destination: returnedDestination,
		term:        term,
	}
	return declaration
}

func declareArgumentValueFor[C, T any](form *ArgumentFormBuilder[C], destination destinationLike[C, T], name string) ArgumentDeclaration[C, T] {
	return declareAttachedArgument(form, destination, argumentTermValue, name, Count.Exactly(1), nil)
}

func declareArgumentSequence[C, T any](form *ArgumentFormBuilder[C], destination destinationLike[C, []T], name string, count CountRange) ArgumentDeclaration[C, []T] {
	return declareAttachedArgument(form, destination, argumentTermSequence, name, count, nil)
}

func declareArgumentTokens[C, T any](form *ArgumentFormBuilder[C], destination destinationLike[C, T], name string, language TokenLanguage[T]) ArgumentDeclaration[C, T] {
	normalized, err := normalizeTokenLanguage(language)
	if err != nil {
		return declareAttachedArgument(form, destination, argumentTermTokens, name, CountRange{}, nil)
	}
	return declareAttachedArgument(form, destination, argumentTermTokens, name, normalized.count, normalized)
}

func declareArgumentLiteral[C any](form *ArgumentFormBuilder[C], token string) ArgumentLiteralDeclaration[C] {
	state := formState(form)
	record := formRecord(form)
	termDraft := argumentTermDraft{
		kind:    argumentTermLiteral,
		literal: token,
		count:   Count.Exactly(1),
		valid:   state != nil && record.valid(),
	}
	term := stageLiteralArgumentTerm(state, form.profile, record, termDraft)
	declaration := ArgumentLiteralDeclaration[C]{
		state: state,
		term:  term,
	}
	return declaration
}

func declareArgumentRemainder[C any](form *ArgumentFormBuilder[C], destination destinationLike[C, []string], name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareAttachedArgument(form, destination, argumentTermRemainder, name, count, nil)
}

func declareArgumentPassThrough[C any](form *ArgumentFormBuilder[C], destination destinationLike[C, []string], name string, count CountRange) ArgumentDeclaration[C, []string] {
	return declareAttachedArgument(form, destination, argumentTermPassThrough, name, count, nil)
}

func declareArgumentString[C any](form *ArgumentFormBuilder[C], storage *string, name string) ArgumentDeclaration[C, string] {
	language, err := normalizeValueLanguage[string](Value.String())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedArgument(
		form,
		storage,
		argumentTermValue,
		destinationDeclarationValue,
		language,
		name,
		Count.Exactly(1),
	)
}

func declareArgumentInt[C any](form *ArgumentFormBuilder[C], storage *int, name string) ArgumentDeclaration[C, int] {
	language, err := normalizeValueLanguage[int](Value.Int())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedArgument(
		form,
		storage,
		argumentTermValue,
		destinationDeclarationValue,
		language,
		name,
		Count.Exactly(1),
	)
}

func declareArgumentStrings[C any](form *ArgumentFormBuilder[C], storage *[]string, name string, count CountRange) ArgumentDeclaration[C, []string] {
	language, err := normalizeValueLanguage[string](Value.String())
	if err != nil {
		panic(err)
	}
	return declareEmbeddedArgument(
		form,
		storage,
		argumentTermSequence,
		destinationDeclarationSequence,
		language,
		name,
		count,
	)
}

func formState[C any](form *ArgumentFormBuilder[C]) *declarationSession {
	if form == nil {
		return nil
	}
	return form.session
}

func formRecord[C any](form *ArgumentFormBuilder[C]) argumentFormDraftRef {
	if form == nil {
		return argumentFormDraftRef{}
	}
	return form.form
}

// AvailableWhen sets the structural availability condition for this argument form; unavailable
// forms are excluded before downstream selection and execution.
func (form ArgumentFormDeclaration[C]) AvailableWhen(condition Condition) ArgumentFormDeclaration[C] {
	if !form.form.valid() {
		return form
	}
	stageAvailabilityCondition(form.state, grammarProfileCaseRef{}, authoringArgumentFormConditionTarget{
		reference: form.form,
	}, condition)
	return form
}

// VisibleWhen sets the presentation visibility condition for this argument form without changing
// whether an otherwise available form may be selected.
func (form ArgumentFormDeclaration[C]) VisibleWhen(condition Condition) ArgumentFormDeclaration[C] {
	if !form.form.valid() {
		return form
	}
	stageVisibilityCondition(form.state, grammarProfileCaseRef{}, authoringArgumentFormConditionTarget{
		reference: form.form,
	}, condition)
	return form
}

// Selects declares the destination value selected when this argument form is the effective form.
func (form ArgumentFormDeclaration[C]) Selects[T any](destination destinationLike[C, T], value T) ArgumentFormDeclaration[C] {
	if !form.form.valid() || form.state == nil {
		return form
	}
	stageArgumentFormSelection(form.state, form.form, destinationHandleOf(destination), value)
	return form
}

func mutateArgumentTerm(session *declarationSession, ref argumentTermDraftRef, mutate func(*argumentTermDraft)) {
	if session == nil || !ref.issued() {
		return
	}

	session.use(func(draft *declarationDraft, _ commandRef) {
		if !ref.ownedBy(draft.identity) {
			panic("cli: issued argument reference changed owner")
		}
		record, ok := draft.argumentTerms.resolve(draft, ref)
		if !ok {
			panic("cli: issued argument draft cannot resolve")
		}
		mutate(record)
	})
}

func (declaration ArgumentDeclaration[C, T]) destinationFor(*C, *T) {}

func (declaration ArgumentDeclaration[C, T]) destinationHandle() *destinationHandle {
	return declaration.destination.destinationHandle()
}

// ArgumentDeclaration identifies one staged value-producing positional argument and preserves its
// typed Destination. Visibility, availability, replacement, and destination relationships are
// configured through this declaration. The zero value is invalid; copies identify the same
// declaration, and retained, concurrent, or cross-scope use is builder lifecycle misuse.
type ArgumentDeclaration[C, T any] struct {
	state       *declarationSession
	destination Destination[C, T]
	term        argumentTermDraftRef
}

// ArgumentFormDeclaration identifies one staged explicit positional argument form and may
// participate in replacement relationships within its owning scope. The zero value is invalid;
// copies identify the same declaration, and retained, concurrent, or cross-scope use is builder
// lifecycle misuse.
type ArgumentFormDeclaration[C any] struct {
	state *declarationSession
	form  argumentFormDraftRef
}

// ArgumentLiteralDeclaration identifies one staged literal positional term and may participate in
// replacement relationships within its owning scope. The zero value is invalid; copies identify
// the same declaration, and retained, concurrent, or cross-scope use is builder lifecycle misuse.
type ArgumentLiteralDeclaration[C any] struct {
	state *declarationSession
	term  argumentTermDraftRef
}

// Destination returns the typed destination produced by this positional argument.
func (declaration ArgumentDeclaration[C, T]) Destination() Destination[C, T] {
	return declaration.destination
}

// VisibleWhen makes the flag visible only when condition holds without changing the declaration
// identity.
func (declaration FlagDeclaration[C, T]) VisibleWhen(condition Condition) FlagDeclaration[C, T] {
	if !declaration.flag.valid() {
		panicBuilderMisuse()
	}
	stageVisibilityCondition(declaration.state, grammarProfileCaseRef{}, authoringFlagConditionTarget{
		reference: declaration.flag,
	}, condition)
	return declaration
}

// AvailableWhen makes the positional argument structurally available only when condition holds.
func (declaration ArgumentDeclaration[C, T]) AvailableWhen(condition Condition) ArgumentDeclaration[C, T] {
	if !declaration.term.valid() {
		panicBuilderMisuse()
	}
	stageAvailabilityCondition(declaration.state, grammarProfileCaseRef{}, authoringArgumentTermConditionTarget{
		reference: declaration.term,
	}, condition)
	return declaration
}

// VisibleWhen makes the positional argument visible only when condition holds without changing its
// declaration identity.
func (declaration ArgumentDeclaration[C, T]) VisibleWhen(condition Condition) ArgumentDeclaration[C, T] {
	if !declaration.term.valid() {
		panicBuilderMisuse()
	}
	stageVisibilityCondition(declaration.state, grammarProfileCaseRef{}, authoringArgumentTermConditionTarget{
		reference: declaration.term,
	}, condition)
	return declaration
}

// Greedy selects greedy token allocation for the positional term.
func (declaration ArgumentDeclaration[C, T]) Greedy() ArgumentDeclaration[C, T] {
	return declaration.argumentAllocation(argumentAllocationGreedy)
}

// Minimal selects minimal token allocation for the positional term.
func (declaration ArgumentDeclaration[C, T]) Minimal() ArgumentDeclaration[C, T] {
	return declaration.argumentAllocation(argumentAllocationMinimal)
}

func (declaration ArgumentDeclaration[C, T]) argumentAllocation(allocation argumentAllocationPreference) ArgumentDeclaration[C, T] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.allocationSet {
			term.valid = false
			return
		}
		term.allocationSet = true
		term.allocation = allocation
	})
	return declaration
}

// HyphenValues sets how hyphen-prefixed tokens may be consumed by this positional term.
func (declaration ArgumentDeclaration[C, T]) HyphenValues(mode HyphenValueMode) ArgumentDeclaration[C, T] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.hyphenValuesSet {
			term.valid = false
			return
		}
		term.hyphenValuesSet = true
		term.hyphenValues = mode
	})
	return declaration
}

// Description sets presentation text for the positional term.
func (declaration ArgumentDeclaration[C, T]) Description(text string) ArgumentDeclaration[C, T] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.descriptionSet {
			term.valid = false
			return
		}
		term.descriptionSet = true
		term.description = text
	})
	return declaration
}

// Hidden omits the positional term from ordinary presentation.
func (declaration ArgumentDeclaration[C, T]) Hidden() ArgumentDeclaration[C, T] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.hiddenSet {
			term.valid = false
			return
		}
		term.hiddenSet = true
		term.hidden = true
	})
	return declaration
}

// Release attaches the declared release lifecycle to the positional term.
func (declaration ArgumentDeclaration[C, T]) Release(lifecycle ReleaseLifecycle) ArgumentDeclaration[C, T] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.releaseSet {
			term.valid = false
			return
		}
		term.releaseSet = true
		term.release = lifecycle
	})
	return declaration
}

// Sensitive marks values produced by this positional term as sensitive.
func (declaration ArgumentDeclaration[C, T]) Sensitive() ArgumentDeclaration[C, T] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.sourceSensitiveSet {
			term.valid = false
			return
		}
		term.sourceSensitiveSet = true
		term.sourceSensitive = true
	})
	return declaration
}

// EndOfOptions makes this literal transfer subsequent tokens out of option recognition.
func (declaration ArgumentLiteralDeclaration[C]) EndOfOptions() ArgumentLiteralDeclaration[C] {
	mutateArgumentTerm(declaration.state, declaration.term, func(term *argumentTermDraft) {
		if term.endOfOptionsSet {
			term.valid = false
			return
		}
		term.endOfOptionsSet = true
		term.endOfOptions = true
	})
	return declaration
}

type argumentAllocationPreference uint8

const (
	argumentAllocationUnspecified argumentAllocationPreference = iota
	argumentAllocationGreedy
	argumentAllocationMinimal
)

type argumentTermKind uint8

const (
	argumentTermInvalid argumentTermKind = iota
	argumentTermValue
	argumentTermSequence
	argumentTermTokens
	argumentTermLiteral
	argumentTermRemainder
	argumentTermPassThrough
)

func stageArgumentFormLifecycle(session *declarationSession, target argumentFormDraftRef, lifecycle ReleaseLifecycle) {
	if session == nil || !target.valid() {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		profile := grammarProfileCaseRef{}
		if source, ok := draft.argumentForms.resolve(draft, target); ok {
			profile = source.profile
		}
		record := lifecycleDirectiveDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  profile,
			target: authoringArgumentFormLifecycleTarget{
				reference: target,
			},
			lifecycle: lifecycle,
			valid:     target.ownedBy(draft.identity),
		}
		draft.lifecycleDirectives.append(draft, record)
	})
}

func stageArgumentTermLifecycle(session *declarationSession, target argumentTermDraftRef, lifecycle ReleaseLifecycle) {
	if session == nil || !target.valid() {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		profile := grammarProfileCaseRef{}
		if source, ok := draft.argumentTerms.resolve(draft, target); ok {
			profile = source.profile
		}
		record := lifecycleDirectiveDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  profile,
			target: authoringArgumentTermLifecycleTarget{
				reference: target,
			},
			lifecycle: lifecycle,
			valid:     target.ownedBy(draft.identity),
		}
		draft.lifecycleDirectives.append(draft, record)
	})
}

func stageArgumentFormOnDraft(session *declarationSession, profile grammarProfileCaseRef, draft *declarationDraft, command commandRef, valid bool) argumentFormDraftRef {
	if session == nil || draft == nil {
		return argumentFormDraftRef{}
	}
	if session.draft != draft {
		panic("cli: declaration session draft mismatch")
	}

	record := argumentFormDraft{
		position: draft.nextPosition(),
		command:  command,
		profile:  profile,
		valid:    valid,
	}
	return draft.argumentForms.append(draft, record)
}

func stageArgumentTermOnDraft(draft *declarationDraft, command commandRef, profile grammarProfileCaseRef, form argumentFormDraftRef, position declarationPosition, term argumentTermDraft, destination destinationRef) argumentTermDraftRef {
	if draft == nil {
		panic("cli: nil declaration draft")
	}
	term.position = position
	term.command = command
	term.profile = profile
	term.form = form
	term.destination = destination
	term.completion = snapshotAuthoringCompletion(term.completion)
	return draft.argumentTerms.append(draft, term)
}

func stageAttachedArgumentTerm(session *declarationSession, profile grammarProfileCaseRef, form argumentFormDraftRef, term argumentTermDraft, destination *destinationHandle) (argumentTermDraftRef, *destinationHandle) {
	if session == nil || !form.valid() {
		return argumentTermDraftRef{}, destination
	}

	var ref argumentTermDraftRef
	session.use(func(draft *declarationDraft, command commandRef) {
		valid := term.valid && form.ownedBy(draft.identity)
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
		term.valid = valid
		ref = stageArgumentTermOnDraft(draft, command, profile, form, draft.nextPosition(), term, destinationRef)
	})
	return ref, destination
}

func stageEmbeddedArgumentTerm(session *declarationSession, profile grammarProfileCaseRef, form argumentFormDraftRef, term argumentTermDraft, destination destinationDraft) (argumentTermDraftRef, *destinationHandle) {
	if session == nil || !form.valid() {
		return argumentTermDraftRef{}, nil
	}

	var ref argumentTermDraftRef
	var handle *destinationHandle
	session.use(func(draft *declarationDraft, command commandRef) {
		position := draft.nextPosition()
		valid := term.valid && form.ownedBy(draft.identity)
		handle = stageDestinationRecordOnDraft(session, draft, command, profile, position, destination)
		var destinationRef destinationRef
		if handle == nil {
			valid = false
		} else if identity, ok := handle.currentIdentity().(stagedDestinationIdentity); ok {
			destinationRef = identity.ref
		} else {
			valid = false
		}
		term.valid = valid
		ref = stageArgumentTermOnDraft(draft, command, profile, form, position, term, destinationRef)
	})
	return ref, handle
}

func stageLiteralArgumentTerm(session *declarationSession, profile grammarProfileCaseRef, form argumentFormDraftRef, term argumentTermDraft) argumentTermDraftRef {
	if session == nil || !form.valid() {
		return argumentTermDraftRef{}
	}

	var ref argumentTermDraftRef
	session.use(func(draft *declarationDraft, command commandRef) {
		term.valid = term.valid && form.ownedBy(draft.identity)
		ref = stageArgumentTermOnDraft(draft, command, profile, form, draft.nextPosition(), term, destinationRef{})
	})
	return ref
}

func stageArgumentFormSelection(session *declarationSession, form argumentFormDraftRef, destination *destinationHandle, value any) {
	if session == nil || !form.valid() || destination == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		formRecord, formOK := draft.argumentForms.resolve(draft, form)
		relation, relationOK := captureAuthoringDestinationIdentity(draft, destination)
		profile := grammarProfileCaseRef{}
		if formOK {
			profile = formRecord.profile
		}
		record := argumentFormSelectionDraft{
			position:    draft.nextPosition(),
			command:     command,
			profile:     profile,
			form:        form,
			destination: relation,
			value:       snapshotMutableValue(value),
			valid:       formOK && relationOK,
		}
		draft.argumentFormSelections.append(draft, record)
	})
}

type argumentTermGrammar struct {
	id             ArgumentTermID
	kind           ArgumentTermKind
	disabled       bool
	destination    ValueID
	hasDestination bool
	count          semanticCountRange
	literal        string
	allocation     semanticArgumentAllocationPreference
	endOfOptions   bool
	hyphenValues   HyphenValueMode
	tokenName      string
	tokenSyntax    []string
	tokenLanguage  tokenLanguage
	position       declarationPosition
}

type argumentTermGrammarInput struct {
	ID             ArgumentTermID
	Kind           ArgumentTermKind
	Destination    ValueID
	HasDestination bool
	Count          semanticCountRange
	Literal        string
	Allocation     semanticArgumentAllocationPreference
	EndOfOptions   bool
	HyphenValues   HyphenValueMode
	TokenName      string
	TokenSyntax    []string
	TokenLanguage  tokenLanguage
	Position       declarationPosition
}

func validArgumentTermGrammarInput(input argumentTermGrammarInput) bool {
	if !input.ID.present() || input.Kind == ArgumentTermUnknown || !input.Count.valid() {
		return false
	}
	hyphen := input.HyphenValues
	if hyphen == HyphenValueUnknown {
		hyphen = HyphenValueDisallow
	}
	if hyphen != HyphenValueDisallow && hyphen != HyphenValueNegativeNumbers && hyphen != HyphenValueAllow {
		return false
	}
	if input.Allocation != semanticArgumentAllocationUnspecified && input.Allocation != semanticArgumentAllocationGreedy &&
		input.Allocation != semanticArgumentAllocationMinimal {
		return false
	}
	switch input.Kind {
	case ArgumentTermLiteral:
		if input.HasDestination || input.Literal == "" || input.Count.Minimum() != 1 {
			return false
		}
		maximum, bounded := input.Count.Maximum()
		return bounded && maximum == 1 && input.Allocation == semanticArgumentAllocationUnspecified &&
			hyphen == HyphenValueDisallow
	case ArgumentTermValue, ArgumentTermRemainder, ArgumentTermPassThrough:
		return input.HasDestination && input.Destination.present() && input.Literal == "" && !input.EndOfOptions &&
			input.TokenName == "" && len(input.TokenSyntax) == 0
	case ArgumentTermTokens:
		if !input.HasDestination || !input.Destination.present() || input.Literal != "" || input.EndOfOptions ||
			input.TokenName == "" {
			return false
		}
		return !input.TokenLanguage.Valid() || input.TokenLanguage.Name() == input.TokenName
	default:
		return false
	}
}

func acceptArgumentTermGrammar(input argumentTermGrammarInput) argumentTermGrammar {
	if !validArgumentTermGrammarInput(input) {
		panic("semantic: invalid argument term")
	}
	hyphen := input.HyphenValues
	if hyphen == HyphenValueUnknown {
		hyphen = HyphenValueDisallow
	}
	tokenSyntax := append([]string(nil), input.TokenSyntax...)
	grammar := argumentTermGrammar{
		id:             input.ID,
		kind:           input.Kind,
		destination:    input.Destination,
		hasDestination: input.HasDestination,
		count:          input.Count,
		literal:        input.Literal,
		allocation:     input.Allocation,
		endOfOptions:   input.EndOfOptions,
		hyphenValues:   hyphen,
		tokenName:      input.TokenName,
		tokenSyntax:    tokenSyntax,
		tokenLanguage:  input.TokenLanguage,
		position:       input.Position,
	}

	return grammar
}

func (term argumentTermGrammar) Enabled() bool { return !term.disabled }

func (term argumentTermGrammar) Destination() (ValueID, bool) {
	return term.destination, term.hasDestination
}

func (term argumentTermGrammar) TokenLanguage() (tokenLanguage, bool) {
	return term.tokenLanguage, term.tokenLanguage.Valid()
}

func (term argumentTermGrammar) TokenSyntax() []string {
	return append([]string(nil), term.tokenSyntax...)
}

type argumentFormGrammar struct {
	id       ArgumentFormID
	command  CommandID
	profile  GrammarProfileID
	terms    []argumentTermGrammar
	position declarationPosition
	enabled  bool
}

type argumentFormGrammarInput struct {
	ID       ArgumentFormID
	Command  CommandID
	Terms    []argumentTermGrammar
	Position declarationPosition
	Enabled  bool
}

func acceptArgumentFormGrammar(input argumentFormGrammarInput) argumentFormGrammar {
	if !input.ID.present() || !input.Command.present() {
		panic("semantic: invalid argument form grammar")
	}
	grammar := argumentFormGrammar{
		id:       input.ID,
		command:  input.Command,
		terms:    append([]argumentTermGrammar(nil), input.Terms...),
		position: input.Position,
		enabled:  input.Enabled,
	}

	return grammar
}

func (form argumentFormGrammar) Terms() []argumentTermGrammar {
	return append([]argumentTermGrammar(nil), form.terms...)
}

type argumentFormSelection struct {
	destination ValueID
	value       any
}

func acceptArgumentFormSelection(destination ValueID, value any) argumentFormSelection {
	mustDestination(destination)
	selection := argumentFormSelection{
		destination: destination,
		value:       value,
	}
	return selection
}

type argumentFormRecord struct {
	meta       recordMeta[ArgumentFormID]
	command    CommandID
	selections []argumentFormSelection
	lifecycle  ReleaseLifecycle
}

func acceptArgumentFormRecord(id ArgumentFormID, command CommandID, position declarationPosition) argumentFormRecord {
	mustCommand(command)
	record := argumentFormRecord{
		meta:    acceptRecordMeta(id, position),
		command: command,
	}
	return record
}

func (record argumentFormRecord) Selections() []argumentFormSelection {
	return append([]argumentFormSelection(nil), record.selections...)
}

type argumentDestination struct {
	id      ValueID
	present bool
}

type argumentTermRecord struct {
	meta            recordMeta[ArgumentTermID]
	form            ArgumentFormID
	destination     argumentDestination
	presentation    presentation
	name            string
	sourceSensitive bool
	lifecycle       ReleaseLifecycle
	completion      completionDeclaration
}

func acceptArgumentTermRecord(id ArgumentTermID, form ArgumentFormID, position declarationPosition) argumentTermRecord {
	if !form.present() {
		panic("semantic: invalid argument-form identity")
	}
	record := argumentTermRecord{
		meta: acceptRecordMeta(id, position),
		form: form,
	}
	return record
}

func acceptArgumentValueTermRecord(id ArgumentTermID, form ArgumentFormID, destination ValueID, position declarationPosition) argumentTermRecord {
	mustDestination(destination)
	record := acceptArgumentTermRecord(id, form, position)
	record.destination = argumentDestination{
		id:      destination,
		present: true,
	}
	return record
}

func (record argumentTermRecord) Completion() (completionDeclaration, bool) {
	return record.completion, record.completion.Valid()
}

func (record argumentTermRecord) Destination() (ValueID, bool) {
	return record.destination.id, record.destination.present
}
