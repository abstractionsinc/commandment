package cli

// ReplacementTarget is the statically compatible flag/argument declaration family for C.
type ReplacementTarget[C any] interface {
	replacementTarget(config *C)
	authoringReplacementCarrier
}

func (FlagDeclaration[C, T]) replacementTarget(*C) {}

func (ArgumentDeclaration[C, T]) replacementTarget(*C) {}

// ReplacedBy declares that this command is replaced by target in the same owning parent
// declaration scope.
func (source CommandDeclaration[C]) ReplacedBy(target CommandDeclaration[C]) CommandDeclaration[C] {
	replaceDeclaration(source, target)
	return source
}

// ReplacedBy declares that this flag declaration is replaced by a compatible flag/argument target.
func (source FlagDeclaration[C, T]) ReplacedBy(target ReplacementTarget[C]) FlagDeclaration[C, T] {
	replaceDeclaration(source, target)
	return source
}

// ReplacedBy declares that this positional argument is replaced by a compatible flag/argument target.
func (source ArgumentDeclaration[C, T]) ReplacedBy(target ReplacementTarget[C]) ArgumentDeclaration[C, T] {
	replaceDeclaration(source, target)
	return source
}

// ReplacedBy declares that this spelling is replaced by target for the same typed flag value.
func (source FlagSpellingDeclaration[C, T]) ReplacedBy(target FlagSpellingDeclaration[C, T]) FlagSpellingDeclaration[C, T] {
	replaceDeclaration(source, target)
	return source
}

// ReplacedBy declares that this exact environment-input declaration is replaced by a target with
// the same configuration and value type. Prefix inputs join this private family in their owner wave.
func (source EnvironmentInputDeclaration[C, T]) ReplacedBy(target environmentReplacementTarget[C, T]) EnvironmentInputDeclaration[C, T] {
	replaceDeclaration(source, target)
	return source
}

func (source EnvironmentPrefixDeclaration[C, K, V]) ReplacedBy(target environmentReplacementTarget[C, map[K]V]) EnvironmentPrefixDeclaration[C, K, V] {
	replaceDeclaration(source, target)
	return source
}

// ReplacedBy declares that this argument form is replaced by target in the same configuration scope.
func (source ArgumentFormDeclaration[C]) ReplacedBy(target ArgumentFormDeclaration[C]) ArgumentFormDeclaration[C] {
	replaceDeclaration(source, target)
	return source
}

// ReplacedBy declares that this literal argument is replaced by target in the same configuration scope.
func (source ArgumentLiteralDeclaration[C]) ReplacedBy(target ArgumentLiteralDeclaration[C]) ArgumentLiteralDeclaration[C] {
	replaceDeclaration(source, target)
	return source
}

// ReplacedBy declares that this canonical domain value is replaced by target in the same value domain.
func (source DomainValueDeclaration[T]) ReplacedBy(target DomainValueDeclaration[T]) DomainValueDeclaration[T] {
	source.use(func(domain *valueDomainState, entry *valueDomainEntry) {
		if target.session != source.session || target.index < 0 || target.index >= len(domain.entries) ||
			target.index == source.index || domain.entries[target.index].kind != domainDeclarationValue {
			entry.valid = false
			domain.issues = append(domain.issues, "invalid domain replacement target")
			return
		}
		entry.replacement = target.index
	})
	return source
}

// ReplacedBy declares that this domain alias is replaced by the canonical target value in the same value domain.
func (source DomainAliasDeclaration[T]) ReplacedBy(target DomainValueDeclaration[T]) DomainAliasDeclaration[T] {
	source.use(func(domain *valueDomainState, entry *valueDomainEntry) {
		if target.session != source.session || target.index < 0 || target.index >= len(domain.entries) ||
			domain.entries[target.index].kind != domainDeclarationValue {
			entry.valid = false
			domain.issues = append(domain.issues, "invalid domain replacement target")
			return
		}
		entry.replacement = target.index
	})
	return source
}

func replacementRefForTarget(target authoringReplacementTarget, plan declarationPlan, state *staticAcceptanceState) (replacementRef, bool) {
	index, indexed := declarationReplacementTargetIndex(target, plan.indexes)
	if !indexed {
		return replacementRef{}, false
	}
	slot, ok := state.slot(index)
	if !ok {
		return replacementRef{}, false
	}
	return replacementRefForAcceptedSlot(target.kind, slot)
}

func replacementRefForAcceptedSlot(kind authoringReplacementKind, slot *declarationAcceptanceSlot) (replacementRef, bool) {
	if slot == nil {
		return replacementRef{}, false
	}
	switch kind {
	case authoringReplacementInvalid:
		return replacementRef{}, false
	case authoringReplacementCommand:
		if len(slot.commandEntries) == 0 || !slot.commandEntries[0].ID().present() {
			return replacementRef{}, false
		}
		return replacementCommandEntry(slot.commandEntries[0]), true
	case authoringReplacementFlag:
		if !slot.flag.ID().present() {
			return replacementRef{}, false
		}
		return replacementFlag(slot.flag), true
	case authoringReplacementArgument:
		if !slot.argumentTerm.ID().present() {
			return replacementRef{}, false
		}
		return replacementArgumentTerm(slot.argumentTerm), true
	case authoringReplacementFlagSpelling:
		if !slot.spelling.ID().present() {
			return replacementRef{}, false
		}
		return replacementFlagSpelling(slot.spelling), true
	case authoringReplacementEnvironment:
		if !slot.environment.ID().present() {
			return replacementRef{}, false
		}
		return replacementEnvironmentInput(slot.environment), true
	case authoringReplacementArgumentForm:
		if !slot.argumentForm.ID().present() {
			return replacementRef{}, false
		}
		return replacementArgumentForm(slot.argumentForm), true
	case authoringReplacementArgumentLiteral:
		if !slot.argumentTerm.ID().present() {
			return replacementRef{}, false
		}
		return replacementArgumentTerm(slot.argumentTerm), true
	default:
		return replacementRef{}, false
	}
}

type authoringReplacementKind uint8

const (
	authoringReplacementInvalid authoringReplacementKind = iota
	authoringReplacementCommand
	authoringReplacementFlag
	authoringReplacementArgument
	authoringReplacementFlagSpelling
	authoringReplacementEnvironment
	authoringReplacementArgumentForm
	authoringReplacementArgumentLiteral
)

type authoringReplacementTarget struct {
	kind    authoringReplacementKind
	session *declarationSession

	command      commandRef
	flag         flagRef
	argument     argumentTermDraftRef
	spelling     flagSpellingRef
	environment  valueDirectiveRef
	argumentForm argumentFormDraftRef
}

func (target authoringReplacementTarget) valid() bool {
	if target.kind == authoringReplacementInvalid || target.session == nil {
		return false
	}

	switch target.kind {
	case authoringReplacementCommand:
		return target.command.valid()
	case authoringReplacementFlag:
		return target.flag.valid()
	case authoringReplacementArgument, authoringReplacementArgumentLiteral:
		return target.argument.valid()
	case authoringReplacementFlagSpelling:
		return target.spelling.valid()
	case authoringReplacementEnvironment:
		return target.environment.valid()
	case authoringReplacementArgumentForm:
		return target.argumentForm.valid()
	default:
		return false
	}
}

type authoringReplacementCarrier interface {
	authoringReplacementTarget() authoringReplacementTarget
}

type environmentReplacementTarget[C, T any] interface {
	authoringReplacementCarrier
	environmentReplacementTarget(config *C, value *T)
}

func (EnvironmentInputDeclaration[C, T]) environmentReplacementTarget(*C, *T) {}

func (EnvironmentPrefixDeclaration[C, K, V]) environmentReplacementTarget(*C, *map[K]V) {}

func (source CommandDeclaration[C]) authoringReplacementTarget() authoringReplacementTarget {
	target := authoringReplacementTarget{
		kind:    authoringReplacementCommand,
		session: source.session,
		command: source.command,
	}
	return target
}

func (source FlagDeclaration[C, T]) authoringReplacementTarget() authoringReplacementTarget {
	ref := source.flag
	target := authoringReplacementTarget{
		kind:    authoringReplacementFlag,
		session: source.state,
		flag:    ref,
	}
	return target
}

func (source ArgumentDeclaration[C, T]) authoringReplacementTarget() authoringReplacementTarget {
	ref := source.term
	target := authoringReplacementTarget{
		kind:     authoringReplacementArgument,
		session:  source.state,
		argument: ref,
	}
	return target
}

func (source FlagSpellingDeclaration[C, T]) authoringReplacementTarget() authoringReplacementTarget {
	ref := source.spelling
	target := authoringReplacementTarget{
		kind:     authoringReplacementFlagSpelling,
		session:  source.state,
		spelling: ref,
	}
	return target
}

func (source EnvironmentInputDeclaration[C, T]) authoringReplacementTarget() authoringReplacementTarget {
	ref := source.producer
	target := authoringReplacementTarget{
		kind:        authoringReplacementEnvironment,
		session:     source.state,
		environment: ref,
	}
	return target
}

func (source EnvironmentPrefixDeclaration[C, K, V]) authoringReplacementTarget() authoringReplacementTarget {
	ref := source.producer
	target := authoringReplacementTarget{
		kind:        authoringReplacementEnvironment,
		session:     source.state,
		environment: ref,
	}
	return target
}

func (source ArgumentFormDeclaration[C]) authoringReplacementTarget() authoringReplacementTarget {
	ref := source.form
	target := authoringReplacementTarget{
		kind:         authoringReplacementArgumentForm,
		session:      source.state,
		argumentForm: ref,
	}
	return target
}

func (source ArgumentLiteralDeclaration[C]) authoringReplacementTarget() authoringReplacementTarget {
	ref := source.term
	target := authoringReplacementTarget{
		kind:     authoringReplacementArgumentLiteral,
		session:  source.state,
		argument: ref,
	}
	return target
}

func replaceDeclaration(source, target authoringReplacementCarrier) {
	sourceTarget := source.authoringReplacementTarget()
	targetTarget := target.authoringReplacementTarget()
	replacement := replacementDraft{
		source: sourceTarget,
		target: targetTarget,
		valid: sourceTarget.valid() && targetTarget.valid() &&
			sourceTarget.session == targetTarget.session,
	}
	stageDeclarationReplacement(sourceTarget.session, replacement)
}

func stageDeclarationReplacement(session *declarationSession, replacement replacementDraft) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		if !replacementTargetOwnedBy(replacement.source, draft.identity) ||
			!replacementTargetOwnedBy(replacement.target, draft.identity) {
			replacement.valid = false
		}

		profile := replacementTargetProfile(draft, replacement.source)
		record := replacementDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  profile,
			source:   replacement.source,
			target:   replacement.target,
			valid:    replacement.valid,
		}
		draft.replacements.append(draft, record)
	})
}

func replacementTargetOwnedBy(target authoringReplacementTarget, owner *draftIdentity) bool {
	switch target.kind {
	case authoringReplacementInvalid:
		return false
	case authoringReplacementCommand:
		return target.command.ownedBy(owner)
	case authoringReplacementFlag:
		return target.flag.ownedBy(owner)
	case authoringReplacementArgument, authoringReplacementArgumentLiteral:
		return target.argument.ownedBy(owner)
	case authoringReplacementFlagSpelling:
		return target.spelling.ownedBy(owner)
	case authoringReplacementEnvironment:
		return target.environment.ownedBy(owner)
	case authoringReplacementArgumentForm:
		return target.argumentForm.ownedBy(owner)
	default:
		return false
	}
}

func replacementTargetProfile(draft *declarationDraft, target authoringReplacementTarget) grammarProfileCaseRef {
	if draft == nil {
		return grammarProfileCaseRef{}
	}

	switch target.kind {
	case authoringReplacementInvalid:
		return grammarProfileCaseRef{}
	case authoringReplacementCommand:
		if record, ok := draft.commands.resolve(draft, target.command); ok {
			return record.profile
		}
	case authoringReplacementFlag:
		if record, ok := draft.flags.resolve(draft, target.flag); ok {
			return record.profile
		}
	case authoringReplacementArgument, authoringReplacementArgumentLiteral:
		if record, ok := draft.argumentTerms.resolve(draft, target.argument); ok {
			return record.profile
		}
	case authoringReplacementFlagSpelling:
		if spelling, ok := draft.flagSpellings.resolve(draft, target.spelling); ok {
			if flag, ok := draft.flags.resolve(draft, spelling.flag); ok {
				return flag.profile
			}
		}
	case authoringReplacementEnvironment:
		if record, ok := draft.valueDirectives.resolve(draft, target.environment); ok {
			return record.profile
		}
	case authoringReplacementArgumentForm:
		if record, ok := draft.argumentForms.resolve(draft, target.argumentForm); ok {
			return record.profile
		}
	}

	return grammarProfileCaseRef{}
}

func stageStaticReplacements(draft *declarationCandidate, input declarationPlan, source *declarationDraft, state *staticAcceptanceState) error {
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordReplacement {
			continue
		}
		replacement := source.replacements.entries[record.familyIndex].value
		if !replacement.valid {
			panic("cli: validated replacement is invalid")
		}
		sourceRef, ok := replacementRefForTarget(replacement.source, input, state)
		if !ok {
			panic("cli: validated replacement source identity is unavailable")
		}
		targetRef, ok := replacementRefForTarget(replacement.target, input, state)
		if !ok {
			panic("cli: validated replacement target identity is unavailable")
		}
		source, target := sourceRef, targetRef
		draft.DeclarationReplacement(source, target, input.positionFor(index))
	}
	return nil
}

// declarationReplacementKind is the bounded private identity family used only
// for exact declaration replacement relationships. It is not a universal
// declaration registry or lookup key.
type declarationReplacementKind uint8

const (
	declarationReplacementKindUnknown declarationReplacementKind = iota
	declarationReplacementKindCommandEntry
	declarationReplacementKindFlag
	declarationReplacementKindFlagSpelling
	declarationReplacementKindArgumentForm
	declarationReplacementKindArgumentTerm
	declarationReplacementKindEnvironmentInput
)

type declarationReplacementIdentity struct {
	kind             declarationReplacementKind
	commandEntry     CommandEntryID
	flag             FlagID
	flagSpelling     FlagSpellingID
	argumentForm     ArgumentFormID
	argumentTerm     ArgumentTermID
	environmentInput EnvironmentInputID
}

func commandEntryReplacementIdentity(id CommandEntryID) declarationReplacementIdentity {
	if !id.present() {
		panic("semantic: invalid command-entry replacement identity")
	}
	identity := declarationReplacementIdentity{
		kind:         declarationReplacementKindCommandEntry,
		commandEntry: id,
	}
	return identity
}

func flagReplacementIdentity(id FlagID) declarationReplacementIdentity {
	if !id.present() {
		panic("semantic: invalid flag replacement identity")
	}
	identity := declarationReplacementIdentity{
		kind: declarationReplacementKindFlag,
		flag: id,
	}
	return identity
}

func flagSpellingReplacementIdentity(id FlagSpellingID) declarationReplacementIdentity {
	if !id.present() {
		panic("semantic: invalid flag-spelling replacement identity")
	}
	identity := declarationReplacementIdentity{
		kind:         declarationReplacementKindFlagSpelling,
		flagSpelling: id,
	}
	return identity
}

func argumentFormReplacementIdentity(id ArgumentFormID) declarationReplacementIdentity {
	if !id.present() {
		panic("semantic: invalid argument-form replacement identity")
	}
	identity := declarationReplacementIdentity{
		kind:         declarationReplacementKindArgumentForm,
		argumentForm: id,
	}
	return identity
}

func argumentTermReplacementIdentity(id ArgumentTermID) declarationReplacementIdentity {
	if !id.present() {
		panic("semantic: invalid argument-term replacement identity")
	}
	identity := declarationReplacementIdentity{
		kind:         declarationReplacementKindArgumentTerm,
		argumentTerm: id,
	}
	return identity
}

func environmentInputReplacementIdentity(id EnvironmentInputID) declarationReplacementIdentity {
	if !id.present() {
		panic("semantic: invalid environment-input replacement identity")
	}
	identity := declarationReplacementIdentity{
		kind:             declarationReplacementKindEnvironmentInput,
		environmentInput: id,
	}
	return identity
}

func (identity declarationReplacementIdentity) Valid() bool {
	switch identity.kind {
	case declarationReplacementKindCommandEntry:
		return identity.commandEntry.present()
	case declarationReplacementKindFlag:
		return identity.flag.present()
	case declarationReplacementKindFlagSpelling:
		return identity.flagSpelling.present()
	case declarationReplacementKindArgumentForm:
		return identity.argumentForm.present()
	case declarationReplacementKindArgumentTerm:
		return identity.argumentTerm.present()
	case declarationReplacementKindEnvironmentInput:
		return identity.environmentInput.present()
	default:
		return false
	}
}

func (identity declarationReplacementIdentity) Kind() declarationReplacementKind {
	return identity.kind
}

func (identity declarationReplacementIdentity) CommandEntry() CommandEntryID {
	return identity.commandEntry
}

func (identity declarationReplacementIdentity) Flag() FlagID { return identity.flag }

func (identity declarationReplacementIdentity) FlagSpelling() FlagSpellingID {
	return identity.flagSpelling
}

func (identity declarationReplacementIdentity) ArgumentForm() ArgumentFormID {
	return identity.argumentForm
}

func (identity declarationReplacementIdentity) ArgumentTerm() ArgumentTermID {
	return identity.argumentTerm
}

func (identity declarationReplacementIdentity) EnvironmentInput() EnvironmentInputID {
	return identity.environmentInput
}

type declarationReplacementRecord struct {
	source   declarationReplacementIdentity
	target   declarationReplacementIdentity
	position declarationPosition
}

func acceptDeclarationReplacementRecord(source, target declarationReplacementIdentity, position declarationPosition) declarationReplacementRecord {
	if !source.Valid() || !target.Valid() || source == target {
		panic("semantic: invalid declaration replacement")
	}
	record := declarationReplacementRecord{
		source:   source,
		target:   target,
		position: position,
	}

	return record
}
