package cli

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
)

func prepareExecutableValueProducer(kind declaredProducerKind, computation authoringValueComputation, resolver destinationAcceptanceResolver) (valueProducerRecord, valueProducer, bool) {
	if len(computation.pending) != 0 {
		return valueProducerRecord{}, nil, false
	}
	resolved, ok := resolveDestinationDependencies(computation.dependencies, resolver)
	if !ok {
		return valueProducerRecord{}, nil, false
	}
	if computation.evaluate == nil {
		return valueProducerRecord{}, nil, false
	}
	evaluator := func(ctx context.Context, configuration any, dependencies Dependencies) (valueEvaluation, error) {
		ctx = contextWithValueStateKeyAliases(ctx, resolved.keys)
		value, present, err := computation.evaluate(ctx, configuration, dependencies)
		if err != nil {
			return valueEvaluation{}, err
		}
		if !present {
			return valueOmitted(), nil
		}
		return valuePresent(value), nil
	}
	return acceptValueProducer(kind, resolved.requirements, computation.source), evaluator, true
}

func prepareIntrinsicDestinationProducer(draft *declarationCandidate, resolver destinationAcceptanceResolver, destinationValue destinationAcceptanceInput, destination acceptanceDestinationRef, position declarationPosition) error {
	definition, ok := prepareDestinationDefinition(destinationValue)
	if !ok {
		return errInvalidValueDomainDeclaration
	}
	var kind declaredProducerKind
	switch definition.Kind() {
	case destinationKindDerived:
		kind = declaredProducerKindDerived
	case destinationKindOptionalDerived:
		kind = declaredProducerKindOptionalDerived
	case destinationKindFactResolved:
		kind = declaredProducerKindFact
	default:
		return nil
	}
	producer, callback, ok := prepareExecutableValueProducer(kind, destinationValue.computation, resolver)
	if !ok {
		return errInvalidValueDomainDeclaration
	}
	draft.DestinationProducer(destination, producer, callback, position)
	return nil
}

func prepareResolutionPolicy(policies []ResolutionPolicy, destination destinationDefinition) (semanticResolutionPolicy, bool) {
	if len(policies) == 0 {
		return semanticResolutionPolicy{}, false
	}
	if destination.Kind() != destinationKindScalar &&
		destination.Kind() != destinationKindSequence &&
		destination.Kind() != destinationKindKeyed {
		return semanticResolutionPolicy{}, false
	}
	var result semanticResolutionPolicy
	for _, policy := range policies {
		if !policy.valid {
			return semanticResolutionPolicy{}, false
		}
		var addition semanticResolutionPolicy
		var ok bool
		switch policy.kind {
		case resolutionPolicyKeepFirst:
			if destination.Kind() == destinationKindSequence {
				return semanticResolutionPolicy{}, false
			}
			addition, ok = withDuplicateResolution(addition, duplicateResolutionKeepFirst)
		case resolutionPolicyKeepLast:
			if destination.Kind() == destinationKindSequence {
				return semanticResolutionPolicy{}, false
			}
			addition, ok = withDuplicateResolution(addition, duplicateResolutionKeepLast)
		case resolutionPolicyRejectMultipleTiers:
			addition, ok = withRejectMultipleTiers(addition)
		case resolutionPolicyCombineTiers:
			if destination.Kind() != destinationKindSequence {
				return semanticResolutionPolicy{}, false
			}
			switch policy.order {
			case ResolutionTierLowerFirst:
				addition, ok = withTierCombination(
					addition,
					ResolutionTierLowerFirst,
				)
			case ResolutionTierHigherFirst:
				addition, ok = withTierCombination(
					addition,
					ResolutionTierHigherFirst,
				)
			default:
				return semanticResolutionPolicy{}, false
			}
		default:
			return semanticResolutionPolicy{}, false
		}
		if !ok {
			return semanticResolutionPolicy{}, false
		}
		result, ok = mergeResolutionPolicies(result, addition)
		if !ok {
			return semanticResolutionPolicy{}, false
		}
	}
	return result, true
}

// Values declares destinations, producers, defaults, derivations, resolution policy, environment
// input, and dynamic value domains for the owning command configuration. It is a callback-scoped
// declaration capability; its zero value and retained, concurrent, or post-callback use are invalid.
type Values[C any] struct {
	Config *C

	session *declarationSession
	profile grammarProfileCaseRef
}

// Value declares a scalar destination parsed according to language and bound to storage.
func (values *Values[C]) Value[T any](storage *T, language ValueLanguage[T]) Destination[C, T] {
	_, languageErr := normalizeValueLanguage(language)
	definition, definitionComplete, domainUses := scalarDestinationDefinition(language)
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationValue,
		storage:            storage,
		language:           language,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK && languageErr == nil,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, T]{
		handle: state,
	}
}

// String declares a string destination using the built-in string value language.
func (values *Values[C]) String(storage *string) Destination[C, string] {
	language := Value.String()
	definition, definitionComplete, domainUses := scalarDestinationDefinition(language)
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationValue,
		storage:            storage,
		language:           language,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, string]{
		handle: state,
	}
}

// Int declares an int destination using the built-in integer value language.
func (values *Values[C]) Int(storage *int) Destination[C, int] {
	language := Value.Int()
	definition, definitionComplete, domainUses := scalarDestinationDefinition(language)
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationValue,
		storage:            storage,
		language:           language,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, int]{
		handle: state,
	}
}

// Uint declares a uint destination using the built-in unsigned-integer value language.
func (values *Values[C]) Uint(storage *uint) Destination[C, uint] {
	language := Value.Uint()
	definition, definitionComplete, domainUses := scalarDestinationDefinition(language)
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationValue,
		storage:            storage,
		language:           language,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, uint]{
		handle: state,
	}
}

// Bool declares a bool destination using the built-in Boolean value language.
func (values *Values[C]) Bool(storage *bool) Destination[C, bool] {
	language := Value.Bool()
	definition, definitionComplete, domainUses := scalarDestinationDefinition(language)
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationValue,
		storage:            storage,
		language:           language,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, bool]{
		handle: state,
	}
}

// Sequence declares an ordered sequence destination whose element values use the supplied element language and preserve
// semantic contribution order.
func (values *Values[C]) Sequence[T any](storage *[]T, element ValueLanguage[T]) Destination[C, []T] {
	_, languageErr := normalizeValueLanguage(element)
	definition, definitionComplete, domainUses := sequenceDestinationDefinition(element)
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationSequence,
		storage:            storage,
		language:           element,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              storage != nil && bindingOK && languageErr == nil,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, []T]{
		handle: state,
	}
}

// Keyed declares a keyed destination using the supplied key/value language.
func (values *Values[C]) Keyed[K comparable, V any](storage *map[K]V, language KeyValueLanguage[K, V]) Destination[C, map[K]V] {
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	valid := storage != nil && bindingOK && language.state != nil && language.state.valid
	definition, definitionComplete, domainUses := keyedDestinationDefinition(language)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationKeyed,
		storage:            storage,
		language:           language,
		binding:            binding,
		definition:         definition,
		definitionComplete: definitionComplete,
		domainUses:         domainUses,
		valid:              valid,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, map[K]V]{
		handle: state,
	}
}

// Destination declares an opaque typed destination for semantic producers that do not parse CLI
// text directly.
func (values *Values[C]) Destination[T any](storage *T, snapshot func(T) T) Destination[C, T] {
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	definition := mustOpaqueDestinationDefinition(destinationDeclarationOpaque)
	if snapshot != nil {
		definition = definition.WithSnapshot(func(value any) (any, bool) {
			typed, ok := value.(T)
			if !ok {
				return nil, false
			}
			return snapshot(typed), true
		})
	}
	destinationValue := destinationDraft{
		kind:               destinationDeclarationOpaque,
		storage:            storage,
		binding:            binding,
		definition:         definition,
		definitionComplete: snapshot != nil,
		valid:              storage != nil && bindingOK && snapshot != nil,
	}
	state := stageDestination(values.session, values.profile, destinationValue)
	return Destination[C, T]{
		handle: state,
	}
}

// staticDefaultDeclaration is the inferred fluent owner for one static default producer.
type staticDefaultDeclaration struct {
	state    *declarationSession
	producer valueDirectiveRef
}

func (declaration staticDefaultDeclaration) Sensitive() staticDefaultDeclaration {
	mutateValueDirective(declaration.state, declaration.producer, func(producer *valueDirectiveDraft) {
		producer.sensitive = true
	})
	return declaration
}

// Default adds a static default contribution for destination.
func (values *Values[C]) Default[T any](destination destinationLike[C, T], value T) staticDefaultDeclaration {
	state := destinationHandleOf(destination)
	if state != nil && state.snapshot != nil {
		if snapshot, ok := state.snapshotValue(value); ok {
			if typed, typedOK := snapshot.(T); typedOK {
				value = typed
			}
		}
	}
	directive := valueDirectiveDraft{
		kind:  valueDirectiveDefault,
		value: value,
		valid: state != nil,
	}
	producer := stageValueDirective(
		values.session,
		values.profile,
		directive,
		state,
	)
	return staticDefaultDeclaration{
		state:    values.session,
		producer: producer,
	}
}

// ResolveWith sets the explicit resolution policy sequence for destination.
func (values *Values[C]) ResolveWith[T any](destination destinationLike[C, T], policies ...ResolutionPolicy) {
	state := destinationHandleOf(destination)
	directive := valueDirectiveDraft{
		kind:     valueDirectiveResolution,
		policies: append([]ResolutionPolicy(nil), policies...),
		valid:    state != nil && len(policies) != 0,
	}
	stageValueDirective(values.session, values.profile, directive, state)
}

// Domain declares a dynamic finite value domain produced from read-only configuration and the
// listed dependencies.
func (values *Values[C]) Domain[T comparable](provide func(context.Context, *C, Dependencies, *ValueDomainBuilder[T]) error, dependencies ...Dependency) ValueDomain[T] {
	return stageDynamicValueDomain(values, provide, dependencies)
}

func mutateValueDirective(session *declarationSession, ref valueDirectiveRef, mutate func(*valueDirectiveDraft)) {
	if session == nil || !ref.issued() {
		return
	}

	session.use(func(draft *declarationDraft, _ commandRef) {
		if !ref.ownedBy(draft.identity) {
			panic("cli: issued value directive reference changed owner")
		}
		record, ok := draft.valueDirectives.resolve(draft, ref)
		if !ok {
			panic("cli: issued value directive draft cannot resolve")
		}
		mutate(record)
	})
}

type resolutionPolicyKind uint8

const (
	resolutionPolicyUnknown resolutionPolicyKind = iota
	resolutionPolicyKeepFirst
	resolutionPolicyKeepLast
	resolutionPolicyRejectMultipleTiers
	resolutionPolicyCombineTiers
)

// ResolutionPolicy is an immutable value-resolution policy constructed by Value.
// Its zero value is invalid.
type ResolutionPolicy struct {
	kind  resolutionPolicyKind
	order ResolutionTierOrder
	valid bool
}

// ResolutionTierOrder controls the precedence direction used when combining values across
// resolution tiers. Its zero value is invalid.
type ResolutionTierOrder uint8

// ResolutionTierLowerFirst combines values from lower precedence toward higher
// precedence. ResolutionTierHigherFirst combines them from higher precedence toward
// lower precedence.
const (
	resolutionTierOrderUnknown ResolutionTierOrder = iota
	ResolutionTierLowerFirst
	ResolutionTierHigherFirst
)

// valueConstructors is the stateless namespace behind Value and constructs value languages, domains,
// keyed languages, and resolution policies.
type valueConstructors struct{}

// Value provides built-in value languages, value domains, keyed languages, and resolution
// policies.
var Value valueConstructors

// Bool returns the built-in boolean value language.
func (valueConstructors) Bool() ValueLanguage[bool] {
	return builtinValueLanguage[bool]{
		name: builtinValueLanguageNameBool,
	}
}

// Int returns the built-in signed-integer value language.
func (valueConstructors) Int() ValueLanguage[int] {
	return builtinValueLanguage[int]{
		name: builtinValueLanguageNameInt,
	}
}

// Uint returns the built-in unsigned-integer value language.
func (valueConstructors) Uint() ValueLanguage[uint] {
	return builtinValueLanguage[uint]{
		name: builtinValueLanguageNameUint,
	}
}

// String returns the built-in string value language.
func (valueConstructors) String() ValueLanguage[string] {
	return builtinValueLanguage[string]{
		name: builtinValueLanguageNameString,
	}
}

// Assign returns the identity snapshot policy for immutable or value-semantic T values.
func (valueConstructors) Assign[T any]() func(T) T {
	return func(value T) T { return value }
}

// Domain declares and returns a finite comparable value domain using the callback-scoped builder.
func (valueConstructors) Domain[T comparable](snapshot func(T) T, declare func(*ValueDomainBuilder[T])) ValueDomain[T] {
	return stageValueDomain(snapshot, declare)
}

// Keyed constructs a keyed language from the key and value scalar languages.
func (valueConstructors) Keyed[K comparable, V any](key ValueLanguage[K], value ValueLanguage[V]) KeyValueLanguage[K, V] {
	return stageKeyValueLanguage(key, value)
}

func stageKeyValueLanguage[K comparable, V any](key ValueLanguage[K], value ValueLanguage[V]) KeyValueLanguage[K, V] {
	return KeyValueLanguage[K, V]{state: &keyValueLanguageState{
		key:       key,
		value:     value,
		separator: "=",
		valid:     key != nil && value != nil,
	}}
}

// KeepFirst selects the first contributing source value when multiple resolution tiers contribute a value.
func (valueConstructors) KeepFirst() ResolutionPolicy {
	return ResolutionPolicy{
		kind:  resolutionPolicyKeepFirst,
		valid: true,
	}
}

// KeepLast selects the last contributing source value when multiple resolution tiers contribute a value.
func (valueConstructors) KeepLast() ResolutionPolicy {
	return ResolutionPolicy{
		kind:  resolutionPolicyKeepLast,
		valid: true,
	}
}

// RejectMultipleTiers rejects resolution when more than one resolution tier contributes a value.
func (valueConstructors) RejectMultipleTiers() ResolutionPolicy {
	return ResolutionPolicy{
		kind:  resolutionPolicyRejectMultipleTiers,
		valid: true,
	}
}

// CombineTiers combines values from all contributing resolution tiers in the declared precedence order.
func (valueConstructors) CombineTiers(order ResolutionTierOrder) ResolutionPolicy {
	valid := order == ResolutionTierLowerFirst || order == ResolutionTierHigherFirst
	result := ResolutionPolicy{
		kind:  resolutionPolicyCombineTiers,
		order: order,
		valid: valid,
	}

	return result
}

func opaqueDestinationDefinition(kind destinationDeclarationKind) (destinationDefinition, bool) {
	switch kind {
	case destinationDeclarationOpaque:
		return acceptOpaqueDestinationDefinition(destinationKindOpaque), true
	case destinationDeclarationDerived:
		return acceptOpaqueDestinationDefinition(destinationKindDerived), true
	case destinationDeclarationOptionalDerived:
		return acceptOpaqueDestinationDefinition(destinationKindOptionalDerived), true
	case destinationDeclarationFactRequired:
		return acceptOpaqueDestinationDefinition(destinationKindFactRequired), true
	case destinationDeclarationFactResolved:
		return acceptOpaqueDestinationDefinition(destinationKindFactResolved), true
	default:
		return destinationDefinition{}, false
	}
}

func mustOpaqueDestinationDefinition(kind destinationDeclarationKind) destinationDefinition {
	definition, ok := opaqueDestinationDefinition(kind)
	if !definition.Valid() {
		panic("cli: invalid destination definition")
	}
	_ = ok
	return definition
}

func typedAssignmentDestinationDefinition[T any](kind destinationDeclarationKind) destinationDefinition {
	return mustOpaqueDestinationDefinition(kind).WithSnapshot(func(value any) (any, bool) {
		typed, ok := value.(T)
		if !ok {
			return nil, false
		}
		return typed, true
	})
}

func scalarDestinationDefinition[T any](language ValueLanguage[T]) (destinationDefinition, bool, []authoringDomainUse) {
	normalized, err := normalizeValueLanguage(language)
	if err != nil {
		return destinationDefinition{}, false, nil
	}
	erased, complete := eraseNormalizedValueLanguage(normalized)
	if !complete {
		return destinationDefinition{}, false, nil
	}
	var uses []authoringDomainUse
	if normalized.kind == normalizedValueLanguageDomain {
		uses = []authoringDomainUse{{
			state: normalized.domain,
			kind:  ValueDomainUseDestination,
		}}
	}
	return acceptScalarDestinationDefinition(erased), true, uses
}

func sequenceDestinationDefinition[T any](language ValueLanguage[T]) (destinationDefinition, bool, []authoringDomainUse) {
	normalized, err := normalizeValueLanguage(language)
	if err != nil {
		return destinationDefinition{}, false, nil
	}
	erased, complete := eraseNormalizedValueLanguage(normalized)
	if !complete {
		return destinationDefinition{}, false, nil
	}
	var uses []authoringDomainUse
	if normalized.kind == normalizedValueLanguageDomain {
		uses = []authoringDomainUse{{
			state: normalized.domain,
			kind:  ValueDomainUseSequenceElement,
		}}
	}
	definition := acceptSequenceDestinationDefinition(erased).WithSequenceShape(
		func(values []any) (any, bool) {
			result := make([]T, len(values))
			for i, value := range values {
				typed, ok := value.(T)
				if !ok {
					return nil, false
				}
				result[i] = typed
			}
			return result, true
		},
		func(value any) ([]any, bool) {
			typed, ok := value.([]T)
			if !ok {
				return nil, false
			}
			result := make([]any, len(typed))
			for i := range typed {
				result[i] = typed[i]
			}
			return result, true
		},
	).WithSnapshot(func(value any) (any, bool) {
		typed, ok := value.([]T)
		if !ok {
			return nil, false
		}
		result := make([]T, len(typed))
		for i := range typed {
			result[i] = normalized.snapshot(typed[i])
		}
		return result, true
	})
	return definition, true, uses
}

func keyedDestinationDefinition[K comparable, V any](language KeyValueLanguage[K, V]) (destinationDefinition, bool, []authoringDomainUse) {
	if language.state == nil || !language.state.valid {
		return destinationDefinition{}, false, nil
	}
	key, ok := language.state.key.(ValueLanguage[K])
	if !ok {
		return destinationDefinition{}, false, nil
	}
	value, ok := language.state.value.(ValueLanguage[V])
	if !ok {
		return destinationDefinition{}, false, nil
	}
	keyNormalized, err := normalizeValueLanguage(key)
	if err != nil {
		return destinationDefinition{}, false, nil
	}
	valueNormalized, err := normalizeValueLanguage(value)
	if err != nil {
		return destinationDefinition{}, false, nil
	}
	keyErased, keyComplete := eraseNormalizedValueLanguage(keyNormalized)
	valueErased, valueComplete := eraseNormalizedValueLanguage(valueNormalized)
	if !keyComplete || !valueComplete {
		return destinationDefinition{}, false, nil
	}
	keyed := acceptKeyedValueLanguage(keyErased, valueErased, language.state.separator)
	if language.state.keyOnly {
		keyOnly, ok := language.state.keyOnlyValue.(V)
		if !ok {
			return destinationDefinition{}, false, nil
		}
		keyed = acceptKeyedValueLanguageWithKeyOnly(keyErased, valueErased, language.state.separator, keyOnly)
	}
	uses := make([]authoringDomainUse, 0, keyedDestinationDomainUseCapacity)
	if keyNormalized.kind == normalizedValueLanguageDomain {
		uses = append(uses, authoringDomainUse{
			state: keyNormalized.domain,
			kind:  ValueDomainUseMapKey,
		})
	}
	if valueNormalized.kind == normalizedValueLanguageDomain {
		uses = append(uses, authoringDomainUse{
			state: valueNormalized.domain,
			kind:  ValueDomainUseMapValue,
		})
	}
	definition := acceptKeyedDestinationDefinition(keyed).WithKeyedShape(
		func(entries []keyedValueEntry) (any, bool) {
			result := make(map[K]V, len(entries))
			for _, entry := range entries {
				key, ok := entry.Key.(K)
				if !ok {
					return nil, false
				}
				value, ok := entry.Value.(V)
				if !ok {
					return nil, false
				}
				result[key] = value
			}
			return result, true
		},
		func(value any) ([]keyedValueEntry, bool) {
			typed, ok := value.(map[K]V)
			if !ok {
				return nil, false
			}
			type orderedEntry struct {
				text  string
				entry keyedValueEntry
			}
			ordered := make([]orderedEntry, 0, len(typed))
			seen := make(map[string]K, len(typed))
			for key, item := range typed {
				text, err := keyErased.Format(key)
				if err != nil {
					return nil, false
				}
				if previous, exists := seen[text]; exists && previous != key {
					return nil, false
				}
				seen[text] = key
				ordered = append(ordered, orderedEntry{
					text: text,
					entry: keyedValueEntry{
						Key:   key,
						Value: item,
					},
				})
			}
			slices.SortFunc(ordered, func(left, right orderedEntry) int { return cmp.Compare(left.text, right.text) })
			result := make([]keyedValueEntry, len(ordered))
			for i := range ordered {
				result[i] = ordered[i].entry
			}
			return result, true
		},
		func(left, right any) bool {
			leftKey, leftOK := left.(K)
			rightKey, rightOK := right.(K)
			return leftOK && rightOK && leftKey == rightKey
		},
	).WithSnapshot(func(value any) (any, bool) {
		typed, ok := value.(map[K]V)
		if !ok {
			return nil, false
		}
		result := make(map[K]V, len(typed))
		for key, item := range typed {
			snapshotKey := keyNormalized.snapshot(key)
			snapshotValue := valueNormalized.snapshot(item)
			result[snapshotKey] = snapshotValue
		}
		return result, true
	})
	return definition, true, uses
}

type valueDirectiveKind uint8

const (
	valueDirectiveInvalid valueDirectiveKind = iota
	valueDirectiveDefault
	valueDirectiveComputedDefault
	valueDirectiveOptionalComputedDefault
	valueDirectiveResolution
	valueDirectiveEnvironment
	valueDirectiveEnvironmentPrefix
)

func stageValueDirective(session *declarationSession, profile grammarProfileCaseRef, directive valueDirectiveDraft, destination *destinationHandle) valueDirectiveRef {
	if session == nil || destination == nil {
		return valueDirectiveRef{}
	}

	var ref valueDirectiveRef
	session.use(func(draft *declarationDraft, command commandRef) {
		relation, relationOK := captureAuthoringDestinationIdentity(draft, destination)
		frozen, computationOK := freezeAuthoringValueComputation(draft, directive.computation)
		record := valueDirectiveDraft{
			position:         draft.nextPosition(),
			command:          command,
			profile:          profile,
			kind:             directive.kind,
			destination:      relation,
			value:            snapshotMutableValue(directive.value),
			computation:      frozen,
			policies:         append([]ResolutionPolicy(nil), directive.policies...),
			name:             directive.name,
			sensitive:        directive.sensitive,
			environmentShape: directive.environmentShape,
			delimiter:        directive.delimiter,
			valid:            directive.valid && relationOK && computationOK,
		}
		ref = draft.valueDirectives.append(draft, record)
	})
	return ref
}

func (draft *declarationCandidate) validateDestinationProducers(commands map[CommandID]commandRecord, destinations map[ValueID]destinationRecord, ordered []destinationRecord) []rejection {
	var rejections []rejection
	type producerEdge struct {
		target      ValueID
		constraints map[CommandID]GrammarProfileID
	}
	adjacency := make(map[ValueID][]producerEdge, len(ordered))
	profiles := make(map[GrammarProfileID]grammarProfileRecord, len(draft.input.GrammarProfiles))
	for _, profile := range draft.input.GrammarProfiles {
		profiles[profile.meta.id] = profile
	}
	activationProfiles := draft.commandActivationProfiles(commands)
	constraintsFor := func(command CommandID, profile GrammarProfileID) (map[CommandID]GrammarProfileID, bool) {
		constraints := make(map[CommandID]GrammarProfileID)
		add := func(candidate GrammarProfileID) bool {
			if !candidate.present() {
				return true
			}
			record, ok := profiles[candidate]
			if !ok {
				return false
			}
			if prior, exists := constraints[record.command]; exists && prior != candidate {
				return false
			}
			constraints[record.command] = candidate
			return true
		}
		for _, active := range activationProfiles[command] {
			if !add(active) {
				return nil, false
			}
		}
		if !add(profile) {
			return nil, false
		}
		return constraints, true
	}
	for _, record := range ordered {
		for _, producer := range record.Producers() {
			ownerProfile := producer.profile
			if !ownerProfile.present() {
				ownerProfile = record.meta.profile
			}
			constraints, constraintsOK := constraintsFor(record.command, ownerProfile)
			if !constraintsOK {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
			seen := make(map[ValueID]struct{})
			for _, dependency := range producer.Dependencies() {
				target, ok := destinations[dependency.Destination()]
				if !ok {
					rejections = append(rejections, semanticRejection(record.meta.position))
					continue
				}
				if _, duplicate := seen[target.meta.id]; duplicate {
					rejections = append(rejections, semanticRejection(record.meta.position))
					continue
				}
				seen[target.meta.id] = struct{}{}
				if !commandSameOrAncestor(commands, target.command, record.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
					continue
				}
				if constraintsOK {
					adjacency[record.meta.id] = append(adjacency[record.meta.id], producerEdge{
						target:      target.meta.id,
						constraints: constraints,
					})
				}
			}
		}
	}

	mergeConstraints := func(base, addition map[CommandID]GrammarProfileID) (map[CommandID]GrammarProfileID, bool) {
		result := make(map[CommandID]GrammarProfileID, len(base)+len(addition))
		maps.Copy(result, base)
		for command, profile := range addition {
			if prior, exists := result[command]; exists && prior != profile {
				return nil, false
			}
			result[command] = profile
		}
		return result, true
	}
	var cycleFrom func(start, current ValueID, path map[ValueID]struct{}, constraints map[CommandID]GrammarProfileID) bool
	cycleFrom = func(start, current ValueID, path map[ValueID]struct{}, constraints map[CommandID]GrammarProfileID) bool {
		for _, edge := range adjacency[current] {
			merged, compatible := mergeConstraints(constraints, edge.constraints)
			if !compatible {
				continue
			}
			if edge.target == start {
				return true
			}
			if _, seen := path[edge.target]; seen {
				continue
			}
			path[edge.target] = struct{}{}
			if cycleFrom(start, edge.target, path, merged) {
				return true
			}
			delete(path, edge.target)
		}
		return false
	}
	for _, record := range ordered {
		path := map[ValueID]struct{}{record.meta.id: {}}
		if cycleFrom(record.meta.id, record.meta.id, path, nil) {
			rejections = append(rejections, semanticRejection(record.meta.position))
			break
		}
	}
	return rejections
}

var (
	errInvalidValueLanguageInput = errors.New("semantic: invalid value language input")
	errValueLanguageTypeMismatch = errors.New("semantic: value language type mismatch")
)

// valueEquality is semantic equality captured before typed authoring values are erased.
// It is retained with the semantic record so declaration validation and runtime selection
// use the same typed relation rather than rediscovering equality from any.
type valueEquality func(any, any) bool

// comparableValueEquality captures == for T while T is still known.
func comparableValueEquality[T comparable]() valueEquality {
	return func(left, right any) bool {
		leftTyped, leftOK := left.(T)
		rightTyped, rightOK := right.(T)
		return leftOK && rightOK && leftTyped == rightTyped
	}
}

// Equal compares two erased values using the typed equality relation captured
// at authoring time.
func (equal valueEquality) Equal(left, right any) bool {
	return equal != nil && equal(left, right)
}

type destinationKind uint8

const (
	destinationKindUnknown destinationKind = iota
	destinationKindScalar
	destinationKindSequence
	destinationKindKeyed
	destinationKindOpaque
	destinationKindDerived
	destinationKindOptionalDerived
	destinationKindFactRequired
	destinationKindFactResolved
)

func acceptValueType(kind ValueKind, name string) ValueType {
	if kind == ValueKindUnknown || (kind != ValueKindCustom && name != "") {
		panic("semantic: invalid value type")
	}
	return ValueType{
		Kind: kind,
		Name: name,
	}
}

func (valueType ValueType) valid() bool {
	return valueType.Kind != ValueKindUnknown && (valueType.Kind == ValueKindCustom || valueType.Name == "")
}

type valueLanguageKind uint8

const (
	valueLanguageKindUnknown valueLanguageKind = iota
	valueLanguageKindBuiltin
	valueLanguageKindCodec
	valueLanguageKindDomain
)

// valueLanguage is the immutable, destination-bound erased form of one typed
// public value language. Its callbacks are executable semantics retained from
// the already-established typed authoring relationship; they are never used to
// discover destination identity.
type valueLanguage struct {
	kind     valueLanguageKind
	name     string
	parse    func(string) (any, error)
	evaluate func(string) conversionOutcome
	format   func(any) (string, bool)
	snapshot func(any) (any, bool)
	dynamic  bool
}

func acceptValueLanguage(kind valueLanguageKind, name string, parse func(string) (any, error), format func(any) (string, bool)) valueLanguage {
	if kind == valueLanguageKindUnknown || parse == nil || format == nil {
		panic("semantic: invalid value language")
	}
	evaluate := func(text string) conversionOutcome {
		value, err := parse(text)
		if err != nil {
			if kind == valueLanguageKindBuiltin || kind == valueLanguageKindDomain {
				return rejectedConversion(err)
			}
			return failedConversion(err)
		}
		return acceptedConversion(value)
	}
	result := valueLanguage{
		kind:     kind,
		name:     name,
		parse:    parse,
		evaluate: evaluate,
		format:   format,
		snapshot: func(value any) (any, bool) { return snapshotMutableValue(value), true },
	}

	return result
}

func acceptValueLanguageWithEvaluation(kind valueLanguageKind, name string, evaluate func(string) conversionOutcome, format func(any) (string, bool)) valueLanguage {
	if kind == valueLanguageKindUnknown || evaluate == nil || format == nil {
		panic("semantic: invalid value language")
	}
	parse := func(text string) (any, error) {
		outcome := evaluate(text)
		if value, ok := outcome.Value(); ok {
			return value, nil
		}
		return nil, outcome.Err()
	}
	result := valueLanguage{
		kind:     kind,
		name:     name,
		parse:    parse,
		evaluate: evaluate,
		format:   format,
		snapshot: func(value any) (any, bool) { return snapshotMutableValue(value), true },
	}

	return result
}

func acceptDynamicDomainValueLanguage() valueLanguage {
	return valueLanguage{
		kind:    valueLanguageKindDomain,
		dynamic: true,
	}
}

func (language valueLanguage) Valid() bool {
	return language.kind != valueLanguageKindUnknown &&
		((language.parse != nil && language.evaluate != nil && language.format != nil && language.snapshot != nil) ||
			(language.kind == valueLanguageKindDomain && language.dynamic))
}

func (language valueLanguage) DynamicDomain() bool {
	return language.kind == valueLanguageKindDomain && language.dynamic
}

// WithSnapshot returns a copy of language using snapshot at semantic retention boundaries.
func (language valueLanguage) WithSnapshot(snapshot func(any) (any, bool)) valueLanguage {
	if !language.Valid() || snapshot == nil {
		return valueLanguage{}
	}
	language.snapshot = snapshot
	return language
}

// Snapshot detaches value according to the language's typed snapshot policy.
func (language valueLanguage) Snapshot(value any) (any, bool) {
	if !language.Valid() || language.snapshot == nil {
		return nil, false
	}
	return language.snapshot(value)
}

func (language valueLanguage) Kind() valueLanguageKind { return language.kind }

func (language valueLanguage) Name() string { return language.name }

func (language valueLanguage) Parse(text string) (any, error) {
	if !language.Valid() || language.parse == nil {
		return nil, errInvalidValueLanguageInput
	}
	return language.parse(text)
}

func (language valueLanguage) Evaluate(text string) conversionOutcome {
	if !language.Valid() || language.evaluate == nil {
		return failedConversion(errInvalidValueLanguageInput)
	}
	return language.evaluate(text).Clone()
}

func (language valueLanguage) Format(value any) (string, error) {
	if !language.Valid() || language.format == nil {
		return "", errInvalidValueLanguageInput
	}
	text, ok := language.format(value)
	if !ok {
		return "", errValueLanguageTypeMismatch
	}
	return text, nil
}

func valueTypeForLanguage(language valueLanguage) ValueType {
	switch language.Kind() {
	case valueLanguageKindBuiltin:
		switch language.Name() {
		case "bool":
			return acceptValueType(ValueKindBool, "")
		case "int":
			return acceptValueType(ValueKindInt, "")
		case "uint":
			return acceptValueType(ValueKindUint, "")
		case "string":
			return acceptValueType(ValueKindString, "")
		default:
			panic("semantic: unsupported builtin value language")
		}
	case valueLanguageKindCodec:
		return acceptValueType(ValueKindCustom, language.Name())
	case valueLanguageKindDomain:
		// A finite domain is a custom CLI-level value whose canonical domain
		// identity, rather than a codec name, is carried separately by the
		// destination-domain relationship.
		return acceptValueType(ValueKindCustom, "")
	default:
		return ValueType{}
	}
}

type keyedValueLanguage struct {
	key          valueLanguage
	value        valueLanguage
	separator    string
	keyOnly      bool
	keyOnlyValue any
}

func acceptKeyedValueLanguage(key, value valueLanguage, separator string) keyedValueLanguage {
	if !key.Valid() || !value.Valid() || separator == "" {
		panic("semantic: invalid keyed value language")
	}
	result := keyedValueLanguage{
		key:       key,
		value:     value,
		separator: separator,
	}

	return result
}

func acceptKeyedValueLanguageWithKeyOnly(key, value valueLanguage, separator string, keyOnlyValue any) keyedValueLanguage {
	language := acceptKeyedValueLanguage(key, value, separator)
	language.keyOnly = true
	language.keyOnlyValue = keyOnlyValue
	return language
}

func (language keyedValueLanguage) Valid() bool {
	return language.key.Valid() && language.value.Valid() && language.separator != ""
}

func (language keyedValueLanguage) Key() valueLanguage { return language.key }

func (language keyedValueLanguage) Value() valueLanguage { return language.value }

func (language keyedValueLanguage) Separator() string { return language.separator }

func (language keyedValueLanguage) KeyOnly() (any, bool) {
	return language.keyOnlyValue, language.keyOnly
}

type duplicateResolutionPolicy uint8

const (
	duplicateResolutionDefault duplicateResolutionPolicy = iota
	duplicateResolutionKeepFirst
	duplicateResolutionKeepLast
)

// semanticResolutionPolicy is the normalized set of independent destination policy
// dimensions. Its zero value is the default policy.
type semanticResolutionPolicy struct {
	duplicate           duplicateResolutionPolicy
	rejectMultipleTiers bool
	combination         ResolutionTierOrder
}

func (policy semanticResolutionPolicy) Duplicate() duplicateResolutionPolicy { return policy.duplicate }

func (policy semanticResolutionPolicy) RejectMultipleTiers() bool {
	return policy.rejectMultipleTiers
}

func (policy semanticResolutionPolicy) Combination() ResolutionTierOrder {
	return policy.combination
}

func withDuplicateResolution(policy semanticResolutionPolicy, duplicate duplicateResolutionPolicy) (semanticResolutionPolicy, bool) {
	if duplicate != duplicateResolutionKeepFirst && duplicate != duplicateResolutionKeepLast {
		return semanticResolutionPolicy{}, false
	}
	if policy.duplicate != duplicateResolutionDefault {
		return semanticResolutionPolicy{}, false
	}
	policy.duplicate = duplicate
	return policy, true
}

func withRejectMultipleTiers(policy semanticResolutionPolicy) (semanticResolutionPolicy, bool) {
	if policy.rejectMultipleTiers {
		return semanticResolutionPolicy{}, false
	}
	policy.rejectMultipleTiers = true
	return policy, true
}

func withTierCombination(policy semanticResolutionPolicy, order ResolutionTierOrder) (semanticResolutionPolicy, bool) {
	if order != ResolutionTierLowerFirst && order != ResolutionTierHigherFirst {
		return semanticResolutionPolicy{}, false
	}
	if policy.combination != resolutionTierOrderUnknown {
		return semanticResolutionPolicy{}, false
	}
	policy.combination = order
	return policy, true
}

func mergeResolutionPolicies(base, addition semanticResolutionPolicy) (semanticResolutionPolicy, bool) {
	result := base
	var ok bool
	if addition.duplicate != duplicateResolutionDefault {
		result, ok = withDuplicateResolution(result, addition.duplicate)
		if !ok {
			return semanticResolutionPolicy{}, false
		}
	}
	if addition.rejectMultipleTiers {
		result, ok = withRejectMultipleTiers(result)
		if !ok {
			return semanticResolutionPolicy{}, false
		}
	}
	if addition.combination != resolutionTierOrderUnknown {
		result, ok = withTierCombination(result, addition.combination)
		if !ok {
			return semanticResolutionPolicy{}, false
		}
	}
	return result, true
}

// keyedValueEntry is one keyed value pair in a keyed destination value.
type keyedValueEntry struct {
	Key   any
	Value any
}

// destinationDefinition preserves exact typed collection construction at the
// authoring boundary before the heterogeneous semantic record is erased. The
// closures are shape adapters only: they never discover destination identity
// or infer semantics from reflection.
type destinationDefinition struct {
	kind       destinationKind
	valueType  ValueType
	language   valueLanguage
	keyed      keyedValueLanguage
	resolution semanticResolutionPolicy

	packSequence   func([]any) (any, bool)
	unpackSequence func(any) ([]any, bool)
	packKeyed      func([]keyedValueEntry) (any, bool)
	unpackKeyed    func(any) ([]keyedValueEntry, bool)
	equalKey       func(any, any) bool
	snapshot       func(any) (any, bool)
}

func acceptScalarDestinationDefinition(language valueLanguage) destinationDefinition {
	if !language.Valid() {
		panic("semantic: invalid scalar destination language")
	}
	definition := destinationDefinition{
		kind:      destinationKindScalar,
		valueType: valueTypeForLanguage(language),
		language:  language,
		snapshot:  language.Snapshot,
	}

	return definition
}

func acceptSequenceDestinationDefinition(language valueLanguage) destinationDefinition {
	if !language.Valid() {
		panic("semantic: invalid sequence destination language")
	}
	definition := destinationDefinition{
		kind:      destinationKindSequence,
		valueType: acceptValueType(ValueKindSequence, ""),
		language:  language,
		snapshot:  func(value any) (any, bool) { return snapshotMutableValue(value), true },
	}

	return definition
}

func acceptKeyedDestinationDefinition(language keyedValueLanguage) destinationDefinition {
	if !language.Valid() {
		panic("semantic: invalid keyed destination language")
	}
	definition := destinationDefinition{
		kind:      destinationKindKeyed,
		valueType: acceptValueType(ValueKindKeyed, ""),
		keyed:     language,
		snapshot:  func(value any) (any, bool) { return snapshotMutableValue(value), true },
	}

	return definition
}

func acceptCountDestinationDefinition() destinationDefinition {
	definition := destinationDefinition{
		kind:      destinationKindOpaque,
		valueType: acceptValueType(ValueKindCount, ""),
		snapshot:  func(value any) (any, bool) { return snapshotMutableValue(value), true },
	}

	return definition
}

func acceptOpaqueDestinationDefinition(kind destinationKind) destinationDefinition {
	switch kind {
	case destinationKindOpaque,
		destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		return destinationDefinition{
			kind:     kind,
			snapshot: func(value any) (any, bool) { return snapshotMutableValue(value), true },
		}
	default:
		panic("semantic: invalid opaque destination kind")
	}
}

// WithSnapshot returns a copy of definition using snapshot at semantic retention boundaries.
func (definition destinationDefinition) WithSnapshot(snapshot func(any) (any, bool)) destinationDefinition {
	if !definition.Valid() || snapshot == nil {
		panic("semantic: invalid destination snapshot")
	}
	definition.snapshot = snapshot
	return definition
}

// Snapshot detaches a destination value according to its typed ownership policy.
func (definition destinationDefinition) Snapshot(value any) (any, bool) {
	if !definition.Valid() || definition.snapshot == nil {
		return nil, false
	}
	return definition.snapshot(value)
}

func (definition destinationDefinition) WithSequenceShape(pack func([]any) (any, bool), unpack func(any) ([]any, bool)) destinationDefinition {
	if definition.kind != destinationKindSequence || pack == nil || unpack == nil {
		panic("semantic: invalid sequence destination shape")
	}
	definition.packSequence = pack
	definition.unpackSequence = unpack
	return definition
}

func (definition destinationDefinition) WithKeyedShape(pack func([]keyedValueEntry) (any, bool), unpack func(any) ([]keyedValueEntry, bool), equal func(any, any) bool) destinationDefinition {
	if definition.kind != destinationKindKeyed || pack == nil || unpack == nil || equal == nil {
		panic("semantic: invalid keyed destination shape")
	}
	definition.packKeyed = pack
	definition.unpackKeyed = unpack
	definition.equalKey = equal
	return definition
}

func (definition destinationDefinition) PackSequence(values []any) (any, bool) {
	if definition.kind != destinationKindSequence || definition.packSequence == nil {
		return nil, false
	}
	return definition.packSequence(append([]any(nil), values...))
}

func (definition destinationDefinition) UnpackSequence(value any) ([]any, bool) {
	if definition.kind != destinationKindSequence || definition.unpackSequence == nil {
		return nil, false
	}
	values, ok := definition.unpackSequence(value)
	return append([]any(nil), values...), ok
}

func (definition destinationDefinition) PackKeyed(entries []keyedValueEntry) (any, bool) {
	if definition.kind != destinationKindKeyed || definition.packKeyed == nil {
		return nil, false
	}
	entriesCopy := append([]keyedValueEntry(nil), entries...)
	return definition.packKeyed(entriesCopy)
}

func (definition destinationDefinition) UnpackKeyed(value any) ([]keyedValueEntry, bool) {
	if definition.kind != destinationKindKeyed || definition.unpackKeyed == nil {
		return nil, false
	}
	entries, ok := definition.unpackKeyed(value)
	return append([]keyedValueEntry(nil), entries...), ok
}

func (definition destinationDefinition) EqualKey(left, right any) bool {
	return definition.kind == destinationKindKeyed && definition.equalKey != nil && definition.equalKey(left, right)
}

// WithLanguage replaces the release-effective CLI language while preserving the
// exact collection shape relationship captured before erasure.
func (definition destinationDefinition) WithLanguage(language valueLanguage) destinationDefinition {
	if !language.Valid() || (definition.kind != destinationKindScalar && definition.kind != destinationKindSequence) {
		panic("semantic: invalid destination language replacement")
	}
	definition.language = language
	if definition.kind == destinationKindScalar {
		definition.valueType = valueTypeForLanguage(language)
		definition.snapshot = language.Snapshot
	}
	return definition
}

func (definition destinationDefinition) WithKeyedLanguage(language keyedValueLanguage) destinationDefinition {
	if !language.Valid() || definition.kind != destinationKindKeyed {
		panic("semantic: invalid keyed destination language replacement")
	}
	definition.keyed = language
	return definition
}

func (definition destinationDefinition) Resolution() semanticResolutionPolicy {
	return definition.resolution
}

func (definition destinationDefinition) Valid() bool {
	switch definition.kind {
	case destinationKindScalar, destinationKindSequence:
		return definition.language.Valid()
	case destinationKindKeyed:
		return definition.keyed.Valid()
	case destinationKindOpaque,
		destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		return true
	default:
		return false
	}
}

func (definition destinationDefinition) Kind() destinationKind { return definition.kind }

func (definition destinationDefinition) ValueType() (ValueType, bool) {
	return definition.valueType, definition.valueType.valid()
}

func (definition destinationDefinition) Language() (valueLanguage, bool) {
	return definition.language, definition.kind == destinationKindScalar || definition.kind == destinationKindSequence
}

func (definition destinationDefinition) KeyedLanguage() (keyedValueLanguage, bool) {
	return definition.keyed, definition.kind == destinationKindKeyed
}

func (definition destinationDefinition) ElementType() (ValueType, bool) {
	if definition.kind != destinationKindSequence || !definition.language.Valid() {
		return ValueType{}, false
	}
	valueType := valueTypeForLanguage(definition.language)
	return valueType, valueType.valid()
}

func (definition destinationDefinition) KeyType() (ValueType, bool) {
	if definition.kind != destinationKindKeyed || !definition.keyed.Valid() {
		return ValueType{}, false
	}
	valueType := valueTypeForLanguage(definition.keyed.Key())
	return valueType, valueType.valid()
}

func (definition destinationDefinition) KeyedValueType() (ValueType, bool) {
	if definition.kind != destinationKindKeyed || !definition.keyed.Valid() {
		return ValueType{}, false
	}
	valueType := valueTypeForLanguage(definition.keyed.Value())
	return valueType, valueType.valid()
}

type sourceAnnotationKind uint8

const (
	sourceAnnotationKindUnknown sourceAnnotationKind = iota
	sourceAnnotationKindKey
	sourceAnnotationKindLocation
	sourceAnnotationKindText
)

type sourceAnnotation struct {
	kind  sourceAnnotationKind
	value string
}

func acceptSourceAnnotation(kind sourceAnnotationKind, value string) sourceAnnotation {
	if kind != sourceAnnotationKindKey && kind != sourceAnnotationKindLocation && kind != sourceAnnotationKindText {
		panic("semantic: invalid source annotation")
	}
	return sourceAnnotation{
		kind:  kind,
		value: value,
	}
}

type semanticProducerSource struct {
	annotations []sourceAnnotation
	sensitive   bool
}

func snapshotProducerSource(annotations []sourceAnnotation, sensitive bool) semanticProducerSource {
	return semanticProducerSource{
		annotations: append([]sourceAnnotation(nil), annotations...),
		sensitive:   sensitive,
	}
}

func (source semanticProducerSource) Annotations() []sourceAnnotation {
	return append([]sourceAnnotation(nil), source.annotations...)
}

func (source semanticProducerSource) Sensitive() bool { return source.sensitive }

type declaredProducerKind uint8

const (
	declaredProducerKindUnknown declaredProducerKind = iota
	declaredProducerKindDefault
	declaredProducerKindComputedDefault
	declaredProducerKindOptionalComputedDefault
	declaredProducerKindDerived
	declaredProducerKindOptionalDerived
	declaredProducerKindFact
)

type dependencyRequirement struct {
	destination ValueID
	optional    bool
}

func acceptDependencyRequirement(destination ValueID, optional bool) dependencyRequirement {
	mustDestination(destination)
	return dependencyRequirement{
		destination: destination,
		optional:    optional,
	}
}

func (requirement dependencyRequirement) Destination() ValueID { return requirement.destination }

func (requirement dependencyRequirement) Optional() bool { return requirement.optional }

type valueEvaluation struct {
	value   any
	present bool
}

func valuePresent(value any) valueEvaluation {
	return valueEvaluation{
		value:   value,
		present: true,
	}
}

func valueOmitted() valueEvaluation { return valueEvaluation{} }

func (evaluation valueEvaluation) Value() (any, bool) { return evaluation.value, evaluation.present }

type valueProducer func(context.Context, any, Dependencies) (valueEvaluation, error)

// valueProducerRecord is an immutable canonical producer plan. The authoring
// boundary has already erased the exact typed callback relationship.
type valueProducerRecord struct {
	kind         declaredProducerKind
	dependencies []dependencyRequirement
	static       any
	source       semanticProducerSource
	profile      GrammarProfileID
}

func acceptStaticValueProducer(kind declaredProducerKind, value any, source semanticProducerSource) valueProducerRecord {
	if kind != declaredProducerKindDefault {
		panic("semantic: invalid static value producer")
	}
	record := valueProducerRecord{
		kind:   kind,
		static: value,
		source: source,
	}

	return record
}

func acceptValueProducer(kind declaredProducerKind, dependencies []dependencyRequirement, source semanticProducerSource) valueProducerRecord {
	switch kind {
	case declaredProducerKindComputedDefault, declaredProducerKindOptionalComputedDefault,
		declaredProducerKindDerived, declaredProducerKindOptionalDerived, declaredProducerKindFact:
	default:
		panic("semantic: invalid executable value producer")
	}
	record := valueProducerRecord{
		kind:         kind,
		dependencies: append([]dependencyRequirement(nil), dependencies...),
		source:       source,
	}

	return record
}

func (producer valueProducerRecord) Valid() bool {
	if producer.kind == declaredProducerKindDefault {
		return true
	}
	switch producer.kind {
	case declaredProducerKindComputedDefault, declaredProducerKindOptionalComputedDefault,
		declaredProducerKindDerived, declaredProducerKindOptionalDerived, declaredProducerKindFact:
		return true
	default:
		return false
	}
}

func (producer valueProducerRecord) Dependencies() []dependencyRequirement {
	return append([]dependencyRequirement(nil), producer.dependencies...)
}

func (producer valueProducerRecord) Static() (any, bool) {
	return producer.static, producer.kind == declaredProducerKindDefault
}

// snapshotMutableValue detaches the small set of framework-owned mutable representations that
// cross erased internal boundaries without a destination snapshot. User value semantics are not
// discovered here: arbitrary slices/maps must carry the typed snapshot policy captured by their
// ValueLanguage or DestinationDefinition before erasure.
func snapshotMutableValue(value any) any {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case map[string]string:
		result := make(map[string]string, len(typed))
		maps.Copy(result, typed)
		return result
	default:
		return value
	}
}
