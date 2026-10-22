package cli

// GrammarProfiles declares grammar-profile switches for one command, selected by typed
// destinations. It is a callback-scoped declaration capability; its zero value and retained,
// concurrent, or post-callback use are invalid.
type GrammarProfiles[C any] struct {
	Config *C

	session *declarationSession
}

// GrammarProfileCases declares the cases of one grammar-profile switch. Config is the owning
// command configuration. The value is callback-scoped; its zero value and retained, concurrent,
// or post-callback use are invalid.
type GrammarProfileCases[C any, T comparable] struct {
	Config *C

	session   *declarationSession
	switchRef grammarProfileSwitchRef
}

// Switch declares a grammar-profile switch selected by destination and populates its cases through
// the callback.
func (profiles *GrammarProfiles[C]) Switch[T comparable](selector destinationLike[C, T], declare func(*GrammarProfileCases[C, T])) {
	stageGrammarProfileSwitch(profiles, selector, declare)
}

// Case declares a profile selected when the switch destination equals value.
func (cases *GrammarProfileCases[C, T]) Case(value T, declare func(*CommandBuilder[C])) {
	stageGrammarProfileCase(cases, value, declare)
}

// Default declares the fallback profile used when no explicit case matches.
func (cases *GrammarProfileCases[C, T]) Default(declare func(*CommandBuilder[C])) {
	stageGrammarProfileDefaultCase(cases, declare)
}

func stageGrammarProfileSwitch[C any, T comparable](profiles *GrammarProfiles[C], selector destinationLike[C, T], declare func(*GrammarProfileCases[C, T])) {
	if profiles == nil || profiles.session == nil {
		return
	}

	selectorHandle := destinationHandleOf(selector)
	switchRef, valid := stageProfileSwitch(profiles.session, selectorHandle, comparableValueEquality[T](), declare != nil)
	if !valid || declare == nil {
		return
	}

	cases := GrammarProfileCases[C, T]{
		Config:    profiles.Config,
		session:   profiles.session,
		switchRef: switchRef,
	}
	declare(&cases)
}

func stageGrammarProfileCase[C any, T comparable](cases *GrammarProfileCases[C, T], value T, declare func(*CommandBuilder[C])) {
	if cases == nil || cases.session == nil || !cases.switchRef.valid() {
		return
	}

	session := cases.session
	draft, command, resume := session.suspend()
	defer resume()

	contribution := draft.beginContribution()
	defer contribution.rollback()

	profile, valid := stageProfileCaseOnDraft(draft, command, cases.switchRef, value, declare != nil)
	if !valid || declare == nil {
		return
	}

	child := session.prepareChild(command)
	builder := commandBuilder(cases.Config, child, profile)
	if cause := child.invoke(func() {
		declare(builder)
	}); cause != nil {
		abortBuilder(cause)
	}
	if contribution.rejected() {
		return
	}

	contribution.commit()
}

func stageGrammarProfileDefaultCase[C any, T comparable](cases *GrammarProfileCases[C, T], declare func(*CommandBuilder[C])) {
	if cases == nil || cases.session == nil || !cases.switchRef.valid() {
		return
	}

	session := cases.session
	draft, command, resume := session.suspend()
	defer resume()

	contribution := draft.beginContribution()
	defer contribution.rollback()

	var zero T
	profile, valid := stageDefaultProfileCaseOnDraft(draft, command, cases.switchRef, zero, declare != nil)
	if !valid || declare == nil {
		return
	}

	child := session.prepareChild(command)
	builder := commandBuilder(cases.Config, child, profile)
	if cause := child.invoke(func() {
		declare(builder)
	}); cause != nil {
		abortBuilder(cause)
	}
	if contribution.rejected() {
		return
	}

	contribution.commit()
}

func stageProfileSwitch(session *declarationSession, selector *destinationHandle, equal valueEquality, callbackPresent bool) (grammarProfileSwitchRef, bool) {
	if session == nil || selector == nil {
		return grammarProfileSwitchRef{}, false
	}

	var ref grammarProfileSwitchRef
	valid := false
	session.use(func(draft *declarationDraft, command commandRef) {
		relation, relationOK := captureAuthoringDestinationIdentity(draft, selector)
		valid = callbackPresent && relationOK && equal != nil
		record := grammarProfileSwitchDraft{
			position: draft.nextPosition(),
			command:  command,
			selector: relation,
			equal:    equal,
			valid:    valid,
		}
		ref = draft.grammarProfileSwitches.append(draft, record)
		if !valid {
			draft.reject(record.position, errInvalidAuthoringTransaction)
		}
	})

	return ref, valid
}

func stageProfileCaseOnDraft(draft *declarationDraft, command commandRef, switchRef grammarProfileSwitchRef, value any, callbackPresent bool) (grammarProfileCaseRef, bool) {
	record := grammarProfileCaseDraft{
		command:   command,
		switchRef: switchRef,
		value:     value,
	}
	return stageGrammarProfileCaseRecord(draft, record, callbackPresent)
}

func stageDefaultProfileCaseOnDraft(draft *declarationDraft, command commandRef, switchRef grammarProfileSwitchRef, value any, callbackPresent bool) (grammarProfileCaseRef, bool) {
	record := grammarProfileCaseDraft{
		command:     command,
		switchRef:   switchRef,
		value:       value,
		defaultCase: true,
	}
	return stageGrammarProfileCaseRecord(draft, record, callbackPresent)
}

func stageGrammarProfileCaseRecord(draft *declarationDraft, record grammarProfileCaseDraft, callbackPresent bool) (grammarProfileCaseRef, bool) {
	if draft == nil || !record.switchRef.valid() {
		return grammarProfileCaseRef{}, false
	}

	_, switchOK := draft.grammarProfileSwitches.resolve(draft, record.switchRef)
	valid := callbackPresent && switchOK
	record.position = draft.nextPosition()
	record.valid = valid
	ref := draft.grammarProfileCases.append(draft, record)
	if !valid {
		draft.reject(record.position, errInvalidAuthoringTransaction)
	}

	return ref, valid
}

func stageStaticGrammarProfiles(input declarationPlan, source *declarationDraft, rootInitial dialectInput, draft *declarationCandidate, state *staticAcceptanceState) (dialectInput, error) {
	rootDialect, err := applyStaticDialectDirectives(rootInitial, input, source, -1, -1)
	if err != nil {
		return dialectInput{}, err
	}
	state.rootDialect = rootDialect
	draft.Dialect(state.root, rootDialect)
	for index := range len(input.records) {
		record := input.records[index]
		kind := record.kind
		if kind == declarationRecordGrammarProfileSwitch && !source.grammarProfileSwitches.entries[record.familyIndex].value.valid {
			panic("cli: validated grammar profile switch is invalid")
		}
		if kind == declarationRecordGrammarProfileCase && !source.grammarProfileCases.entries[record.familyIndex].value.valid {
			panic("cli: validated grammar profile case is invalid")
		}
	}
	for index := range len(input.records) {
		record := input.records[index]
		position := input.positionFor(index)
		switch record.kind {
		case declarationRecordGrammarProfileCase:
			var base dialectInput
			var ok bool
			if record.commandIndex == -1 {
				base, ok = state.rootDialect, true
			} else {
				ownerSlot := state.mustSlot(record.commandIndex)
				base, ok = ownerSlot.dialect, validSemanticDialectInput(ownerSlot.dialect)
			}
			if !ok {
				panic("cli: static grammar profile owner has invalid dialect")
			}
			owner := state.mustCommand(record.commandIndex)
			dialect, err := applyStaticDialectDirectives(base, input, source, record.commandIndex, index)
			if err != nil {
				return dialectInput{}, err
			}
			state.slots[index].profile = draft.GrammarProfile(owner, position)
			state.slots[index].dialect = dialect
		case declarationRecordCommand:
			var base dialectInput
			var ok bool
			switch {
			case record.profileIndex >= 0:
				profileSlot := state.mustSlot(record.profileIndex)
				base, ok = profileSlot.dialect, validSemanticDialectInput(profileSlot.dialect)
			case record.commandIndex == -1:
				base, ok = state.rootDialect, true
			default:
				parentSlot := state.mustSlot(record.commandIndex)
				base, ok = parentSlot.dialect, validSemanticDialectInput(parentSlot.dialect)
			}
			if !ok {
				panic("cli: static command has invalid inherited dialect")
			}
			dialect, err := applyStaticDialectDirectives(base, input, source, index, -1)
			if err != nil {
				return dialectInput{}, err
			}
			state.slots[index].dialect = dialect
			command := state.mustCommand(index)
			draft.Dialect(command, dialect)
			if record.profileIndex >= 0 {
				profile := state.mustProfile(record.profileIndex)
				for _, entry := range state.slots[index].commandEntries {
					draft.CommandEntryProfile(entry, profile, position)
				}
				if def := state.slots[index].defaultRef; def.ID().present() {
					draft.DefaultProfile(def, profile, position)
				}
			}
		default:
			continue
		}
	}

	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordShortcut || record.profileIndex < 0 {
			continue
		}
		entry := state.slots[index].shortcutEntry
		profile := state.mustProfile(record.profileIndex)
		if !entry.ID().present() {
			panic("cli: accepted shortcut entry identity is unavailable")
		}
		draft.CommandEntryProfile(entry, profile, input.positionFor(index))
	}
	return rootDialect, nil
}

func (draft *declarationCandidate) GrammarProfileCase(profile grammarProfileRef, selector acceptanceDestinationRef, value any, equal valueEquality, dialect dialectInput, position declarationPosition) {
	draft.mustOpen()
	if !profile.OwnedBy(draft.owner) || !selector.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	profileIndex, profileRecord, profileOK := draft.grammarProfileRecordForRef(profile)
	_, selectorRecord, selectorOK := draft.destinationRecordForRef(selector)
	if !profileOK || !selectorOK || profileRecord.command != selectorRecord.command {
		draft.reject(position)
		return
	}
	if equal == nil {
		panic("semantic: invalid grammar-profile case")
	}
	profileRecord.selector = selector.ID()
	profileRecord.value = value
	profileRecord.equal = equal
	profileRecord.dialect = acceptDialect(dialect)
	draft.input.GrammarProfiles[profileIndex] = profileRecord
}

func (draft *declarationCandidate) GrammarProfileDefaultCase(profile grammarProfileRef, selector acceptanceDestinationRef, value any, equal valueEquality, dialect dialectInput, position declarationPosition) {
	draft.mustOpen()
	if !profile.OwnedBy(draft.owner) || !selector.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	profileIndex, profileRecord, profileOK := draft.grammarProfileRecordForRef(profile)
	_, selectorRecord, selectorOK := draft.destinationRecordForRef(selector)
	if !profileOK || !selectorOK || profileRecord.command != selectorRecord.command {
		draft.reject(position)
		return
	}
	if equal == nil {
		panic("semantic: invalid grammar-profile case")
	}
	profileRecord.selector = selector.ID()
	profileRecord.value = value
	profileRecord.equal = equal
	profileRecord.defaultCase = true
	profileRecord.dialect = acceptDialect(dialect)
	draft.input.GrammarProfiles[profileIndex] = profileRecord
}

func (draft *declarationCandidate) DestinationProfile(destination acceptanceDestinationRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !destination.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.destinationRecordForRef(destination)
	if !ok || record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.Destinations[index] = record
}

func (draft *declarationCandidate) FlagProfile(flag acceptanceFlagRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !flag.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.flagRecordForRef(flag)
	if !ok || record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.Flags[index] = record
}

func (draft *declarationCandidate) ArgumentFormProfile(form argumentFormRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !form.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.argumentFormRecordForRef(form)
	if !ok || record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.ArgumentForms[index] = record
}

func (draft *declarationCandidate) CommandEntryProfile(entry commandEntryRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandEntryRecordForRef(entry)
	if !ok || record.parent != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.CommandEntries[index] = record
}

func (draft *declarationCandidate) DefaultProfile(def defaultRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !def.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.defaultRecordForRef(def)
	if !ok || record.parent != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.Defaults[index] = record
}

func (draft *declarationCandidate) EnvironmentInputProfile(input environmentInputRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !input.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.environmentInputRecordForRef(input)
	if !ok || record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.EnvironmentInputs[index] = record
}

func (draft *declarationCandidate) grammarProfileCommand(profile grammarProfileRef) CommandID {
	_, record, ok := draft.grammarProfileRecordForRef(profile)
	if !ok {
		return CommandID(0)
	}
	return record.command
}

func (draft *declarationCandidate) ConditionPolicyCount() int {
	draft.mustOpen()
	return len(draft.input.ConditionPolicies)
}

func (draft *declarationCandidate) ProviderProfile(provider providerRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !provider.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.providerRecordForRef(provider)
	if !ok || record.parent != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.Providers[index] = record
}

func (draft *declarationCandidate) RuleProfile(rule ruleRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !rule.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.ruleRecordForRef(rule)
	if !ok || record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.Rules[index] = record
}

func (draft *declarationCandidate) AdvisoryProfile(advisory advisoryRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !advisory.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.advisoryRecordForRef(advisory)
	if !ok || record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.Advisories[index] = record
}

func (draft *declarationCandidate) SignalPolicyProfile(policy signalPolicyRef, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !policy.OwnedBy(draft.owner) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.signalPolicyRecordForRef(policy)
	if !ok || record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.meta.profile = profile.ID()
	draft.input.SignalPolicies[index] = record
}

// replacementSubject is a bounded declaration-validation view over declaration
// families that participate in the generic replacement relationship. It is
// never published as semantic identity or used as a lookup authority.
type replacementSubject struct {
	identity  declarationReplacementIdentity
	command   CommandID
	flag      FlagID
	termKind  ArgumentTermKind
	profile   GrammarProfileID
	lifecycle ReleaseLifecycle
	position  declarationPosition
}

func validReplacementProfile(source, target GrammarProfileID) bool {
	if !source.present() {
		return !target.present()
	}
	return !target.present() || source == target
}

func validReplacementLifecycle(scheme ReleaseScheme, source, target ReleaseLifecycle) bool {
	if !scheme.valid() || validateLifecycle(scheme, source) != nil || validateLifecycle(scheme, target) != nil {
		return false
	}
	start := source.DeprecatedIn
	if start == "" {
		start = source.RemovedIn
	}
	if start == "" {
		return false
	}
	if target.IntroducedIn != "" {
		cmp, err := scheme.compare(target.IntroducedIn, start)
		if err != nil || cmp > 0 {
			return false
		}
	}
	if source.RemovedIn == "" {
		return target.RemovedIn == ""
	}
	if target.RemovedIn != "" {
		cmp, err := scheme.compare(target.RemovedIn, source.RemovedIn)
		if err != nil || cmp < 0 {
			return false
		}
	}
	return true
}

func subjectPosition(source declarationReplacementIdentity, relations []declarationReplacementRecord) declarationPosition {
	for _, relation := range relations {
		if relation.source == source {
			return relation.position
		}
	}
	return 0
}

type grammarProfileRecord struct {
	meta        recordMeta[GrammarProfileID]
	command     CommandID
	selector    ValueID
	value       any
	equal       valueEquality
	defaultCase bool
	dialect     dialect
}

func acceptGrammarProfileRecord(id GrammarProfileID, command CommandID, position declarationPosition) grammarProfileRecord {
	mustCommand(command)
	record := grammarProfileRecord{
		meta:    acceptRecordMeta(id, position),
		command: command,
	}
	return record
}

func (record grammarProfileRecord) Selector() (ValueID, bool) {
	return record.selector, record.selector.present()
}

func (record grammarProfileRecord) Value() (any, bool) {
	return record.value, !record.defaultCase && record.selector.present()
}

// ValuesEqual compares two erased selector values through the typed equality
// relation captured by this grammar profile.
func (record grammarProfileRecord) ValuesEqual(left, right any) bool {
	return record.equal.Equal(left, right)
}

// MatchesValue reports whether value matches one of this profile's cases using
// the same captured equality relation used at validation time.
func (record grammarProfileRecord) MatchesValue(value any) bool {
	return !record.defaultCase && record.selector.present() && record.equal.Equal(record.value, value)
}

func (record grammarProfileRecord) Dialect() (dialect, bool) {
	return record.dialect, record.dialect.CommandMatching() != NameMatchingUnknown
}
