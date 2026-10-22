package cli

import "sync"

type destinationAcceptanceInput struct {
	kind               destinationDeclarationKind
	language           any
	sensitive          bool
	computation        authoringValueComputation
	binding            configurationBinding
	bindingRequired    bool
	domainUses         []authoringDomainUse
	definition         destinationDefinition
	definitionComplete bool
	definitionFinal    bool
}

func staticDestinationAcceptanceInput(value destinationDraft) (destinationAcceptanceInput, bool) {
	if !value.valid || !value.intrinsic && value.storage == nil {
		return destinationAcceptanceInput{}, false
	}
	input := destinationAcceptanceInput{
		kind:               value.kind,
		language:           value.language,
		sensitive:          value.sensitive,
		computation:        value.computation,
		binding:            value.binding,
		bindingRequired:    !value.intrinsic,
		domainUses:         value.domainUses,
		definition:         value.definition,
		definitionComplete: value.definitionComplete,
	}

	return input, true
}

func prepareDestinationDefinitionForRelease(destinationValue destinationAcceptanceInput, scheme ReleaseScheme) (destinationDefinition, bool) {
	definition, valid := prepareDestinationDefinition(destinationValue)
	if !valid {
		return definition, false
	}
	if destinationValue.definitionFinal || len(destinationValue.domainUses) == 0 {
		return definition, true
	}

	switch definition.Kind() {
	case destinationKindScalar, destinationKindSequence:
		if len(destinationValue.domainUses) != 1 {
			return destinationDefinition{}, false
		}
		use := destinationValue.domainUses[0]
		want := ValueDomainUseDestination
		if definition.Kind() == destinationKindSequence {
			want = ValueDomainUseSequenceElement
		}
		if use.kind != want {
			return destinationDefinition{}, false
		}
		language, ok := prepareEffectiveDomainValueLanguage(use.state, scheme)
		if !ok {
			return destinationDefinition{}, false
		}
		return definition.WithLanguage(language), true

	case destinationKindKeyed:
		keyed, ok := definition.KeyedLanguage()
		if !ok {
			return destinationDefinition{}, false
		}
		key := keyed.Key()
		value := keyed.Value()
		seenKey := false
		seenValue := false
		for _, use := range destinationValue.domainUses {
			language, ok := prepareEffectiveDomainValueLanguage(use.state, scheme)
			if !ok {
				return destinationDefinition{}, false
			}
			switch use.kind {
			case ValueDomainUseMapKey:
				if seenKey {
					return destinationDefinition{}, false
				}
				seenKey = true
				key = language
			case ValueDomainUseMapValue:
				if seenValue {
					return destinationDefinition{}, false
				}
				seenValue = true
				value = language
			default:
				return destinationDefinition{}, false
			}
		}
		var effective keyedValueLanguage
		if keyOnly, present := keyed.KeyOnly(); present {
			if value.Kind() == valueLanguageKindDomain {
				if _, err := value.Format(keyOnly); err != nil {
					return destinationDefinition{}, false
				}
			}
			effective = acceptKeyedValueLanguageWithKeyOnly(key, value, keyed.Separator(), keyOnly)
		} else {
			effective = acceptKeyedValueLanguage(key, value, keyed.Separator())
		}
		return definition.WithKeyedLanguage(effective), true
	default:
		return destinationDefinition{}, false
	}
}

func prepareDestinationDefinition(destinationValue destinationAcceptanceInput) (destinationDefinition, bool) {
	if destinationValue.definition.Valid() && destinationValue.definitionComplete {
		return destinationValue.definition, true
	}
	switch destinationValue.kind {
	case destinationDeclarationValue, destinationDeclarationSequence:
		var language valueLanguage
		var ok bool
		switch typed := destinationValue.language.(type) {
		case normalizedValueLanguage[bool]:
			language, ok = eraseNormalizedValueLanguage(typed)
		case normalizedValueLanguage[int]:
			language, ok = eraseNormalizedValueLanguage(typed)
		case normalizedValueLanguage[uint]:
			language, ok = eraseNormalizedValueLanguage(typed)
		case normalizedValueLanguage[string]:
			language, ok = eraseNormalizedValueLanguage(typed)
		case builtinValueLanguage[bool]:
			normalized, err := normalizeValueLanguage[bool](typed)
			if err == nil {
				language, ok = eraseNormalizedValueLanguage(normalized)
			}
		case builtinValueLanguage[int]:
			normalized, err := normalizeValueLanguage[int](typed)
			if err == nil {
				language, ok = eraseNormalizedValueLanguage(normalized)
			}
		case builtinValueLanguage[uint]:
			normalized, err := normalizeValueLanguage[uint](typed)
			if err == nil {
				language, ok = eraseNormalizedValueLanguage(normalized)
			}
		case builtinValueLanguage[string]:
			normalized, err := normalizeValueLanguage[string](typed)
			if err == nil {
				language, ok = eraseNormalizedValueLanguage(normalized)
			}
		default:
			return destinationDefinition{}, false
		}
		if !ok {
			return destinationDefinition{}, false
		}
		if destinationValue.kind == destinationDeclarationSequence {
			return acceptSequenceDestinationDefinition(language), true
		}
		return acceptScalarDestinationDefinition(language), true
	case destinationDeclarationOpaque:
		return acceptOpaqueDestinationDefinition(destinationKindOpaque), true
	case destinationDeclarationFactRequired:
		return acceptOpaqueDestinationDefinition(destinationKindFactRequired), true
	case destinationDeclarationKeyed:
		// Keyed destinations are normalized when the exact generic key/value
		// relationship is still available at authoring time. Reaching this fallback
		// means that relationship was not preserved completely.
		return destinationDefinition{}, false
	case destinationDeclarationDerived:
		return acceptOpaqueDestinationDefinition(destinationKindDerived), true
	case destinationDeclarationOptionalDerived:
		return acceptOpaqueDestinationDefinition(destinationKindOptionalDerived), true
	case destinationDeclarationFactResolved:
		return acceptOpaqueDestinationDefinition(destinationKindFactResolved), true
	default:
		return destinationDefinition{}, false
	}
}

func destinationUsesBuiltinSignedInteger(destination destinationAcceptanceInput) bool {
	definition, ok := prepareDestinationDefinition(destination)
	if !ok {
		return false
	}
	language, ok := definition.Language()
	return ok && language.Kind() == valueLanguageKindBuiltin && language.Name() == builtinValueLanguageNameInt
}

func destinationHandleOf[C, T any](destination destinationLike[C, T]) *destinationHandle {
	if destination == nil {
		return nil
	}
	return destination.destinationHandle()
}

// Destination is the typed callback-scoped handle for a Commandment-managed configuration value.
// C is the owning command configuration type and T is the complete value type. Its zero value is
// invalid; copies identify the same staged destination, and retained, concurrent, or cross-scope
// use is builder lifecycle misuse.
type Destination[C, T any] struct {
	handle *destinationHandle
}

type destinationLike[C, T any] interface {
	destinationFor(config *C, value *T)
	destinationHandle() *destinationHandle
}

func (destination Destination[C, T]) destinationFor(*C, *T) {}

func (destination Destination[C, T]) destinationHandle() *destinationHandle {
	return destination.handle
}

// Sensitive marks this destination as sensitive without changing producer provenance.
func (destination Destination[C, T]) Sensitive() Destination[C, T] {
	setDestinationSensitive(destination.handle, true)
	return destination
}

func setDestinationSensitive(handle *destinationHandle, sensitive bool) {
	if handle == nil || handle.session == nil {
		return
	}

	handle.session.use(func(draft *declarationDraft, _ commandRef) {
		identity, ok := handle.currentIdentity().(stagedDestinationIdentity)
		if !ok {
			return
		}
		if !identity.ref.ownedBy(draft.identity) {
			panic("cli: issued destination reference changed owner")
		}

		record, ok := draft.destinations.resolve(draft, identity.ref)
		if !ok || record.handle != handle {
			panic("cli: issued destination draft cannot resolve")
		}
		record.sensitive = sensitive
	})
}

type destinationDeclarationKind uint8

const (
	destinationDeclarationInvalid destinationDeclarationKind = iota
	destinationDeclarationValue
	destinationDeclarationSequence
	destinationDeclarationKeyed
	destinationDeclarationOpaque
	destinationDeclarationDerived
	destinationDeclarationOptionalDerived
	destinationDeclarationFactRequired
	destinationDeclarationFactResolved
)

func stageDestinationRecordOnDraft(session *declarationSession, draft *declarationDraft, command commandRef, profile grammarProfileCaseRef, position declarationPosition, destinationValue destinationDraft) *destinationHandle {
	if session == nil || draft == nil || session.draft != draft {
		return nil
	}
	if frozen, ok := freezeAuthoringValueComputation(draft, destinationValue.computation); ok {
		destinationValue.computation = frozen
	} else {
		destinationValue.valid = false
	}
	handle := &destinationHandle{
		session:  session,
		snapshot: destinationValue.definition.snapshot,
	}
	record := destinationDraft{
		position:           position,
		command:            command,
		profile:            profile,
		handle:             handle,
		kind:               destinationValue.kind,
		storage:            destinationValue.storage,
		language:           destinationValue.language,
		sensitive:          destinationValue.sensitive,
		computation:        destinationValue.computation,
		binding:            destinationValue.binding,
		domainUses:         append([]authoringDomainUse(nil), destinationValue.domainUses...),
		definition:         destinationValue.definition,
		definitionComplete: destinationValue.definitionComplete,
		valid:              destinationValue.valid,
	}
	ref := draft.destinations.append(draft, record)
	handle.setStagedIdentity(ref)
	return handle
}

func stageDestination(session *declarationSession, profile grammarProfileCaseRef, destinationValue destinationDraft) *destinationHandle {
	if session == nil {
		return nil
	}

	var handle *destinationHandle
	session.use(func(draft *declarationDraft, command commandRef) {
		handle = stageDestinationRecordOnDraft(session, draft, command, profile, draft.nextPosition(), destinationValue)
	})
	return handle
}

func establishStaticDestination(input declarationPlan, metadata applicationMetadata, draft *declarationCandidate, state *staticAcceptanceState, index int, destinationValue destinationAcceptanceInput) (acceptanceDestinationRef, error) {
	if index < 0 || index >= len(input.records) {
		panic("cli: static destination record index is invalid")
	}
	record := input.records[index]
	owner := state.mustCommand(record.commandIndex)
	definition, valid := prepareDestinationDefinitionForRelease(destinationValue, metadata.release)
	if !valid {
		return acceptanceDestinationRef{}, errStaticDeclarationRejected
	}
	if destinationValue.bindingRequired && !destinationValue.binding.Valid() {
		panic("cli: validated static destination has no configuration binding")
	}
	position := input.positionFor(index)
	destination := draft.Destination(owner, position, definition, destinationValue.binding)
	draft.DestinationSensitive(destination, destinationValue.sensitive, position)
	state.slots[index].destination = destination
	if staged, ok := state.destinationRefs[index]; ok {
		state.destinations[staged] = destination
	}
	if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
		draft.DestinationProfile(destination, profile, position)
	}
	if err := prepareDestinationDomains(draft, state, destinationValue, destination, position, metadata); err != nil {
		return acceptanceDestinationRef{}, err
	}
	return destination, nil
}

// destinationRecord owns one exact typed semantic destination independently
// from any declaration that contributes grammar or presentation for it.
type destinationRecord struct {
	meta       recordMeta[ValueID]
	command    CommandID
	definition destinationDefinition
	binding    configurationBinding
	producers  []valueProducerRecord
	sensitive  bool
}

func acceptDestinationRecord(id ValueID, command CommandID, position declarationPosition, definition destinationDefinition, binding configurationBinding) destinationRecord {
	mustCommand(command)
	if !definition.Valid() {
		panic("semantic: invalid destination definition")
	}
	record := destinationRecord{
		meta:       acceptRecordMeta(id, position),
		command:    command,
		definition: definition,
		binding:    binding,
	}
	return record
}

func (record destinationRecord) Definition() (destinationDefinition, bool) {
	return record.definition, record.definition.Valid()
}

func (record destinationRecord) Binding() (configurationBinding, bool) {
	return record.binding, record.binding.Valid()
}

func (record destinationRecord) Producers() []valueProducerRecord {
	return append([]valueProducerRecord(nil), record.producers...)
}

type destinationIdentity interface{ destinationIdentity() }

type stagedDestinationIdentity struct{ ref destinationRef }

func (stagedDestinationIdentity) destinationIdentity() {}

type acceptedDestinationIdentity struct {
	scope identityScope
	id    ValueID
}

func (acceptedDestinationIdentity) destinationIdentity() {}

type destinationHandle struct {
	mu       sync.RWMutex
	session  *declarationSession
	identity destinationIdentity
	snapshot func(any) (any, bool)
}

func (handle *destinationHandle) currentIdentity() destinationIdentity {
	if handle == nil {
		return nil
	}
	handle.mu.RLock()
	identity := handle.identity
	handle.mu.RUnlock()
	return identity
}

func (handle *destinationHandle) setStagedIdentity(ref destinationRef) {
	if handle == nil || !ref.valid() {
		panic("cli: invalid staged destination")
	}
	handle.mu.Lock()
	defer handle.mu.Unlock()
	if handle.identity != nil {
		panic("cli: destination identity already initialized")
	}
	handle.identity = stagedDestinationIdentity{
		ref: ref,
	}
}

func (handle *destinationHandle) snapshotValue(value any) (any, bool) {
	if handle == nil || handle.snapshot == nil {
		return nil, false
	}
	return handle.snapshot(value)
}

type destinationAcceptance struct {
	handle   *destinationHandle
	identity acceptedDestinationIdentity
}

func prepareDestinationAcceptance(handle *destinationHandle, identity acceptedDestinationIdentity) destinationAcceptance {
	if handle == nil || !identity.scope.valid() || !identity.id.present() {
		panic("cli: invalid destination acceptance")
	}
	if _, ok := handle.currentIdentity().(stagedDestinationIdentity); !ok {
		panic("cli: destination identity is not staged")
	}
	acceptance := destinationAcceptance{
		handle:   handle,
		identity: identity,
	}
	return acceptance
}

func (acceptance destinationAcceptance) publish() {
	if acceptance.handle == nil {
		panic("cli: unprepared destination acceptance")
	}
	acceptance.handle.mu.Lock()
	acceptance.handle.identity = acceptance.identity
	acceptance.handle.mu.Unlock()
}

func acceptedDestinationHandleIdentity(handle *destinationHandle) (acceptedDestinationIdentity, bool) {
	if handle == nil {
		return acceptedDestinationIdentity{}, false
	}
	identity, ok := handle.currentIdentity().(acceptedDestinationIdentity)
	return identity, ok && identity.scope.valid() && identity.id.present()
}

func stagedDestinationOwnedBy(handle *destinationHandle, owner *draftIdentity) bool {
	if handle == nil || owner == nil {
		return false
	}
	identity, ok := handle.currentIdentity().(stagedDestinationIdentity)
	return ok && identity.ref.ownedBy(owner)
}
