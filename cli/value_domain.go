package cli

import (
	"context"
	"errors"
	"strings"
	"sync"
)

var errInvalidValueDomainDeclaration = errors.New("cli: invalid value-domain declaration")

func prepareValueDomain(draft *declarationCandidate, value *valueDomainState, metadata applicationMetadata, state *staticAcceptanceState) (valueDomainRef, error) {
	if !validValueDomainState(value) {
		return valueDomainRef{}, errInvalidValueDomainDeclaration
	}
	if value.dynamic != nil {
		return prepareDynamicValueDomain(draft, value.dynamic, state)
	}
	entries, err := prepareDomainEntries(draft, value, metadata)
	if err != nil {
		return valueDomainRef{}, err
	}
	return draft.ValueDomain(entries), nil
}

func prepareDynamicValueDomain(draft *declarationCandidate, domain *dynamicDomainDraft, state *staticAcceptanceState) (valueDomainRef, error) {
	ownerIndex := domain.ownerCommandIndex
	if domain.ownerCommand.valid() {
		var ok bool
		ownerIndex, ok = state.authoredCommandIndex(domain.ownerCommand)
		if !ok {
			return valueDomainRef{}, errInvalidValueDomainDeclaration
		}
	}
	owner, ok := state.command(ownerIndex)
	if !ok || !owner.ID().present() {
		return valueDomainRef{}, errInvalidValueDomainDeclaration
	}
	resolved, ok := resolveDestinationDependencies(domain.dependencies, state)
	if !ok {
		return valueDomainRef{}, errInvalidValueDomainDeclaration
	}
	callback := func(ctx context.Context, configuration, rawDependencies, output any) error {
		ctx = contextWithValueStateKeyAliases(ctx, resolved.keys)
		dependencies, ok := rawDependencies.(Dependencies)
		if !ok {
			return errInvalidValueDomainDeclaration
		}
		evaluation, ok := output.(*dynamicDomainEvaluation)
		if !ok || evaluation == nil {
			return errInvalidValueDomainDeclaration
		}
		captured, err := domain.evaluate(ctx, configuration, dependencies)
		if err != nil {
			return err
		}
		evaluation.state = captured
		return nil
	}
	definition := acceptDynamicDomainDefinition(owner.ID(), resolved.requirements)
	return draft.DynamicValueDomain(definition, callback), nil
}

type domainEntrySemantics struct {
	kind        domainDeclarationKind
	name        string
	value       any
	target      int
	description string
	hidden      bool
	lifecycle   ReleaseLifecycle
	replacement int
}

func acceptDomainEntrySemantics(entry valueDomainEntry) (domainEntrySemantics, bool) {
	if !entry.valid {
		return domainEntrySemantics{}, false
	}
	semantics := domainEntrySemantics{
		kind:        entry.kind,
		name:        entry.name,
		value:       entry.value,
		target:      entry.target,
		description: entry.description,
		hidden:      entry.hidden,
		lifecycle:   entry.lifecycle,
		replacement: entry.replacement,
	}
	return semantics, true
}

func prepareDomainEntries(draft *declarationCandidate, state *valueDomainState, metadata applicationMetadata) ([]domainEntryRef, error) {
	if !validValueDomainState(state) || state.dynamic != nil {
		return nil, errInvalidValueDomainDeclaration
	}
	entries := make([]domainEntrySemantics, len(state.entries))
	for index, entry := range state.entries {
		var ok bool
		entries[index], ok = acceptDomainEntrySemantics(entry)
		if !ok {
			return nil, errInvalidValueDomainDeclaration
		}
	}

	return prepareDomainEntryRecords(draft, entries, metadata)
}

func prepareBatchDomainEntries(draft *declarationCandidate, domain batchValueDomain, metadata applicationMetadata) ([]domainEntryRef, error) {
	if domain.dynamic != nil {
		return nil, errInvalidValueDomainDeclaration
	}

	return prepareDomainEntryRecords(draft, domain.entries, metadata)
}

func prepareDomainEntryRecords(draft *declarationCandidate, entries []domainEntrySemantics, metadata applicationMetadata) ([]domainEntryRef, error) {
	effective := make([]bool, len(entries))
	for index, entry := range entries {
		lifecycle := entry.lifecycle
		if err := validateLifecycle(metadata.release, lifecycle); err != nil {
			return nil, errInvalidValueDomainDeclaration
		}
		LifecycleState, err := evaluateLifecycle(metadata.release, lifecycle, "")
		if err != nil {
			return nil, errInvalidValueDomainDeclaration
		}
		effective[index] = LifecycleState == LifecycleStateCurrent || LifecycleState == LifecycleStateDeprecated
	}
	for index, entry := range entries {
		if entry.kind != domainDeclarationAlias || !effective[index] {
			continue
		}
		if entry.target < 0 || entry.target >= len(entries) || !effective[entry.target] || entries[entry.target].kind != domainDeclarationValue {
			return nil, errInvalidValueDomainDeclaration
		}
	}

	accepted := make([]domainEntryRef, len(entries))
	for index, entry := range entries {
		position := declarationPosition(index + 1)
		lifecycle := entry.lifecycle
		switch entry.kind {
		case domainDeclarationValue:
			accepted[index] = draft.DomainValue(position, entry.name, entry.value, entry.description, entry.hidden, lifecycle)
		case domainDeclarationAlias:
			if entry.target < 0 || entry.target >= index || !accepted[entry.target].ID().present() {
				return nil, errInvalidValueDomainDeclaration
			}
			accepted[index] = draft.DomainAlias(position, entry.name, accepted[entry.target], entry.hidden, lifecycle)
		default:
			return nil, errInvalidValueDomainDeclaration
		}
	}
	for index, entry := range entries {
		if entry.replacement < 0 {
			continue
		}
		if entry.replacement >= len(accepted) || entry.replacement == index || entries[entry.replacement].kind != domainDeclarationValue || !accepted[index].ID().present() || !accepted[entry.replacement].ID().present() {
			return nil, errInvalidValueDomainDeclaration
		}
		draft.DomainReplacement(accepted[index], accepted[entry.replacement], declarationPosition(index+1))
	}

	return accepted, nil
}

func prepareDestinationDomains(draft *declarationCandidate, state *staticAcceptanceState, destinationValue destinationAcceptanceInput, destination acceptanceDestinationRef, position declarationPosition, metadata applicationMetadata) error {
	for _, use := range destinationValue.domainUses {
		if use.state == nil || use.kind == ValueDomainUseUnknown {
			return errInvalidValueDomainDeclaration
		}
		domain, ok := state.domains[use.state]
		if !ok {
			var err error
			domain, err = prepareValueDomain(draft, use.state, metadata, state)
			if err != nil {
				return err
			}
			state.domains[use.state] = domain
		}
		draft.ValueDomainUse(domain, destination, use.kind, position)
	}
	return nil
}

func prepareEffectiveDomainValueLanguage(state *valueDomainState, scheme ReleaseScheme) (valueLanguage, bool) {
	return eraseEffectiveDomainValueLanguage(state, scheme)
}

// validEffectiveDomainContribution validates the portions of a statically known
// typed contribution that are governed by finite-domain membership. Non-domain
// language components remain outside this check.
func validEffectiveDomainContribution(definition destinationDefinition, contribution any) bool {
	switch definition.Kind() {
	case destinationKindScalar:
		language, ok := definition.Language()
		if !ok || language.Kind() != valueLanguageKindDomain {
			return ok
		}
		if language.DynamicDomain() {
			return true
		}
		_, err := language.Format(contribution)
		return err == nil

	case destinationKindSequence:
		language, ok := definition.Language()
		if !ok || language.Kind() != valueLanguageKindDomain {
			return ok
		}
		if language.DynamicDomain() {
			return true
		}
		values, ok := definition.UnpackSequence(contribution)
		if !ok {
			return false
		}
		for _, value := range values {
			if _, err := language.Format(value); err != nil {
				return false
			}
		}
		return true

	case destinationKindKeyed:
		keyed, ok := definition.KeyedLanguage()
		if !ok {
			return false
		}
		validateKey := keyed.Key().Kind() == valueLanguageKindDomain && !keyed.Key().DynamicDomain()
		validateValue := keyed.Value().Kind() == valueLanguageKindDomain && !keyed.Value().DynamicDomain()
		if !validateKey && !validateValue {
			return true
		}
		entries, ok := definition.UnpackKeyed(contribution)
		if !ok {
			return false
		}
		for _, entry := range entries {
			if validateKey {
				if _, err := keyed.Key().Format(entry.Key); err != nil {
					return false
				}
			}
			if validateValue {
				if _, err := keyed.Value().Format(entry.Value); err != nil {
					return false
				}
			}
		}
		return true
	default:
		return true
	}
}

// DomainValueDeclaration identifies one staged canonical value in a ValueDomain. The zero value is
// invalid; copies identify the same declaration, and retained, concurrent, or cross-scope use is
// builder lifecycle misuse.
type DomainValueDeclaration[T comparable] struct {
	session *valueDomainSession[T]
	index   int
}

// DomainAliasDeclaration identifies one staged ValueDomain alias and its relationship to a
// canonical value. The zero value is invalid; copies identify the same declaration, and retained,
// concurrent, or cross-scope use is builder lifecycle misuse.
type DomainAliasDeclaration[T comparable] struct {
	session *valueDomainSession[T]
	index   int
}

// ValueDomain is a finite value language over comparable values constructed by Value.Domain. The declaration
// callback fixes its canonical values, aliases, visibility, release lifecycle, and replacements.
// Its zero value is not a valid value language.
type ValueDomain[T comparable] struct {
	_     [0]func()
	state *valueDomainState
}

type valueDomainState struct {
	entries  []valueDomainEntry
	issues   []string
	dynamic  *dynamicDomainDraft
	snapshot func(any) (any, bool)
	equal    valueEquality
}

type domainDeclarationKind uint8

const (
	domainDeclarationUnknown domainDeclarationKind = iota
	domainDeclarationValue
	domainDeclarationAlias
)

type valueDomainEntry struct {
	kind        domainDeclarationKind
	name        string
	value       any
	target      int
	description string
	hidden      bool
	lifecycle   ReleaseLifecycle
	replacement int
	valid       bool
}

type valueDomainSessionState uint8

const (
	valueDomainSessionPrepared valueDomainSessionState = iota
	valueDomainSessionActive
	valueDomainSessionClosed
)

type valueDomainSession[T comparable] struct {
	mu   sync.Mutex
	uses sync.WaitGroup

	draft *valueDomainState

	state valueDomainSessionState
	inUse bool
}

func (session *valueDomainSession[T]) activate() {
	if session == nil {
		panic("cli: nil value-domain session")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state != valueDomainSessionPrepared {
		panic("cli: invalid value-domain session transition")
	}
	if session.draft == nil {
		panic("cli: prepared value-domain session has no draft")
	}
	session.state = valueDomainSessionActive
}

func (session *valueDomainSession[T]) close() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	retainedUse := session.inUse
	session.state = valueDomainSessionClosed
	session.mu.Unlock()

	session.uses.Wait()

	session.mu.Lock()
	session.inUse = false
	session.draft = nil
	session.mu.Unlock()

	return retainedUse
}

func (session *valueDomainSession[T]) invoke(callback func()) (cause error) {
	session.activate()
	defer func() {
		recovered := recover()
		retainedUse := session.close()

		switch recovered := recovered.(type) {
		case nil:
		case builderAbort:
			cause = recovered.cause
		default:
			panic(recovered)
		}

		if cause == nil && retainedUse {
			cause = errBuilderMisuse
		}
	}()

	callback()
	return nil
}

func (session *valueDomainSession[T]) acquire() (*valueDomainState, func()) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	if session.state != valueDomainSessionActive || session.inUse {
		session.mu.Unlock()
		panicBuilderMisuse()
	}
	if session.draft == nil {
		session.mu.Unlock()
		panic("cli: active value-domain session has no draft")
	}
	session.inUse = true
	session.uses.Add(1)
	draft := session.draft
	session.mu.Unlock()

	release := func() {
		session.mu.Lock()
		session.inUse = false
		session.mu.Unlock()
		session.uses.Done()
	}

	return draft, release
}

func (session *valueDomainSession[T]) use(callback func(*valueDomainState)) {
	draft, release := session.acquire()
	defer release()
	callback(draft)
}

func (ValueDomain[T]) valueLanguageFor(*T) {}

// ValueDomainBuilder declares the canonical values and aliases of one ValueDomain. It is valid only
// while the Value.Domain callback is active; its zero value and retained, concurrent, or
// post-callback use are invalid.
type ValueDomainBuilder[T comparable] struct {
	session *valueDomainSession[T]
}

// Value declares a canonical named value in the domain.
func (domain *ValueDomainBuilder[T]) Value(name string, value T) DomainValueDeclaration[T] {
	return declareDomainValue(domain, name, value)
}

// Alias declares name as an alias of target in the same domain.
func (domain *ValueDomainBuilder[T]) Alias(name string, target DomainValueDeclaration[T]) DomainAliasDeclaration[T] {
	return declareDomainAlias(domain, name, target)
}

func snapshotDomainEntries[T any](state *valueDomainState) ([]domainLanguageEntry[T], bool) {
	if !validValueDomainState(state) || state.dynamic != nil {
		return nil, false
	}
	result := make([]domainLanguageEntry[T], 0, len(state.entries))
	for _, entry := range state.entries {
		switch entry.kind {
		case domainDeclarationValue:
			value, ok := entry.value.(T)
			if !ok {
				return nil, false
			}
			composite := domainLanguageEntry[T]{
				name:      entry.name,
				value:     value,
				canonical: true,
			}
			result = append(result, composite)
		case domainDeclarationAlias:
			if entry.target < 0 || entry.target >= len(state.entries) {
				return nil, false
			}
			canonical := state.entries[entry.target]
			if canonical.kind != domainDeclarationValue {
				return nil, false
			}
			value, ok := canonical.value.(T)
			if !ok {
				return nil, false
			}
			result = append(result, domainLanguageEntry[T]{
				name:  entry.name,
				value: value,
			})
		default:
			return nil, false
		}
	}
	return result, true
}

func eraseDomainValueLanguage[T any](state *valueDomainState) (valueLanguage, bool) {
	if validValueDomainState(state) && state.dynamic != nil {
		return acceptDynamicDomainValueLanguage(), true
	}
	entries, ok := snapshotDomainEntries[T](state)
	if !ok {
		return valueLanguage{}, false
	}
	language := acceptValueLanguage(
		valueLanguageKindDomain,
		"",
		func(text string) (any, error) {
			for _, entry := range entries {
				if entry.name == text {
					return entry.value, nil
				}
			}
			return nil, errInvalidValueLanguage
		},
		func(value any) (string, bool) {
			typed, ok := value.(T)
			if !ok {
				return "", false
			}
			for _, entry := range entries {
				if entry.canonical && state.equal.Equal(entry.value, typed) {
					return entry.name, true
				}
			}
			return "", false
		},
	)
	if state.snapshot == nil {
		return valueLanguage{}, false
	}
	language = language.WithSnapshot(state.snapshot)
	return language, language.Valid()
}

func (domain ValueDomain[T]) valueDomainStateFor() *valueDomainState {
	return domain.state
}

func (ValueDomain[T]) valueDomainFor(*T) {}

func captureValueDomain[T comparable](snapshot func(T) T, declare func(*ValueDomainBuilder[T])) (ValueDomain[T], error) {
	state := &valueDomainState{
		equal: safeComparableValueEquality[T](),
	}
	if snapshot == nil {
		state.issues = append(state.issues, "missing snapshot policy")
	} else {
		state.snapshot = func(value any) (any, bool) {
			typed, ok := value.(T)
			if !ok {
				return nil, false
			}
			return snapshot(typed), true
		}
	}
	if declare == nil {
		state.issues = append(state.issues, "missing declaration callback")
		domain := ValueDomain[T]{
			state: state,
		}
		return domain, nil
	}
	session := valueDomainSession[T]{
		draft: state,
		state: valueDomainSessionPrepared,
	}
	builder := &ValueDomainBuilder[T]{
		session: &session,
	}
	cause := session.invoke(func() { declare(builder) })
	domain := ValueDomain[T]{
		state: state,
	}
	return domain, cause
}

func stageValueDomain[T comparable](snapshot func(T) T, declare func(*ValueDomainBuilder[T])) ValueDomain[T] {
	domain, cause := captureValueDomain(snapshot, declare)
	if cause != nil {
		abortBuilder(cause)
	}
	return domain
}

func declareDomainValue[T comparable](builder *ValueDomainBuilder[T], name string, value T) DomainValueDeclaration[T] {
	if builder == nil || builder.session == nil {
		panicBuilderMisuse()
	}
	declaration := DomainValueDeclaration[T]{
		session: builder.session,
		index:   -1,
	}
	builder.session.use(func(state *valueDomainState) {
		if state.snapshot != nil {
			var valid bool
			value, valid = snapshotDomainValue(state.snapshot, value)
			if !valid {
				state.issues = append(state.issues, "invalid snapshot result")
			}
		}
		entry := valueDomainEntry{
			kind:        domainDeclarationValue,
			name:        name,
			value:       value,
			target:      -1,
			replacement: -1,
			valid:       true,
		}
		if !validDomainName(name) {
			entry.valid = false
			state.issues = append(state.issues, "invalid canonical value name")
		}
		if !state.equal.Equal(value, value) {
			entry.valid = false
			state.issues = append(state.issues, "unstable canonical typed value")
		}
		for _, prior := range state.entries {
			if prior.name == name {
				entry.valid = false
				state.issues = append(state.issues, "duplicate domain name")
			}
			if prior.kind == domainDeclarationValue && state.equal.Equal(prior.value, value) {
				entry.valid = false
				state.issues = append(state.issues, "duplicate canonical typed value")
			}
		}
		declaration.index = len(state.entries)
		state.entries = append(state.entries, entry)
	})
	return declaration
}

func declareDomainAlias[T comparable](builder *ValueDomainBuilder[T], name string, target DomainValueDeclaration[T]) DomainAliasDeclaration[T] {
	if builder == nil || builder.session == nil {
		panicBuilderMisuse()
	}
	declaration := DomainAliasDeclaration[T]{
		session: builder.session,
		index:   -1,
	}
	builder.session.use(func(state *valueDomainState) {
		entry := valueDomainEntry{
			kind:        domainDeclarationAlias,
			name:        name,
			target:      -1,
			replacement: -1,
			valid:       true,
		}
		if !validDomainName(name) {
			entry.valid = false
			state.issues = append(state.issues, "invalid alias name")
		}
		if target.session != builder.session || target.index < 0 || target.index >= len(state.entries) ||
			state.entries[target.index].kind != domainDeclarationValue {
			entry.valid = false
			state.issues = append(state.issues, "invalid alias target")
		} else {
			entry.target = target.index
		}
		for _, prior := range state.entries {
			if prior.name == name {
				entry.valid = false
				state.issues = append(state.issues, "duplicate domain name")
			}
		}
		declaration.index = len(state.entries)
		state.entries = append(state.entries, entry)
	})
	return declaration
}

func validDomainName(name string) bool {
	return name != "" && !strings.ContainsRune(name, '\x00')
}

func snapshotDomainValue[T comparable](snapshot func(any) (any, bool), value T) (T, bool) {
	snap, ok := snapshot(value)
	if !ok {
		return value, false
	}
	typed, ok := snap.(T)
	if !ok {
		return value, false
	}
	return typed, true
}

func safeComparableValueEquality[T comparable]() valueEquality {
	return func(left, right any) bool {
		leftTyped, leftOK := left.(T)
		rightTyped, rightOK := right.(T)
		if !leftOK || !rightOK {
			return false
		}
		equal := false
		func() {
			defer func() { _ = recover() }()
			equal = leftTyped == rightTyped
		}()
		return equal
	}
}

func validValueDomainState(state *valueDomainState) bool {
	if state == nil || len(state.issues) != 0 {
		return false
	}
	if state.dynamic != nil {
		return state.dynamic.evaluate != nil && state.dynamic.ownerCommandIndex >= -1 && len(state.entries) == 0
	}
	if state.equal == nil {
		return false
	}
	for _, entry := range state.entries {
		if !entry.valid {
			return false
		}
	}
	return true
}

func (declaration DomainValueDeclaration[T]) Description(text string) DomainValueDeclaration[T] {
	declaration.use(func(_ *valueDomainState, entry *valueDomainEntry) { entry.description = text })
	return declaration
}

func (declaration DomainValueDeclaration[T]) Hidden() DomainValueDeclaration[T] {
	declaration.use(func(_ *valueDomainState, entry *valueDomainEntry) { entry.hidden = true })
	return declaration
}

func (declaration DomainAliasDeclaration[T]) Hidden() DomainAliasDeclaration[T] {
	declaration.use(func(_ *valueDomainState, entry *valueDomainEntry) { entry.hidden = true })
	return declaration
}

func (declaration DomainValueDeclaration[T]) use(callback func(*valueDomainState, *valueDomainEntry)) {
	if declaration.session == nil {
		panicBuilderMisuse()
	}
	declaration.session.use(func(state *valueDomainState) {
		if declaration.index < 0 || declaration.index >= len(state.entries) {
			panic("cli: issued value-domain reference cannot resolve")
		}
		entry := &state.entries[declaration.index]
		if entry.kind != domainDeclarationValue {
			panic("cli: issued value-domain reference has wrong kind")
		}
		callback(state, entry)
	})
}

func (declaration DomainAliasDeclaration[T]) use(callback func(*valueDomainState, *valueDomainEntry)) {
	if declaration.session == nil {
		panicBuilderMisuse()
	}
	declaration.session.use(func(state *valueDomainState) {
		if declaration.index < 0 || declaration.index >= len(state.entries) {
			panic("cli: issued value-domain reference cannot resolve")
		}
		entry := &state.entries[declaration.index]
		if entry.kind != domainDeclarationAlias {
			panic("cli: issued value-domain reference has wrong kind")
		}
		callback(state, entry)
	})
}

func contributionResult[T any](contribution ValueContribution[T]) (T, bool, bool) {
	switch contribution.state {
	case contributionValue:
		return contribution.value, true, true
	case contributionOmitted:
		var zero T
		return zero, false, true
	default:
		var zero T
		return zero, false, false
	}
}

func validateDynamicValueDomain(domainIndex int, domain valueDomainRecord, commands map[CommandID]commandRecord, destinations map[ValueID]destinationRecord, flags map[FlagID]flagRecord, profiles map[GrammarProfileID]grammarProfileRecord) []rejection {
	definition, dynamic := domain.DynamicDefinition()
	if !dynamic {
		return nil
	}
	order := dependencyOrdinal(domainIndex)
	reject := func(rejections []rejection) []rejection {
		candidateRejection := rejection{
			Kind:  rejectionStructuralInvalidity,
			Order: order,
		}
		return append(rejections, candidateRejection)
	}
	var rejections []rejection
	if _, ok := commands[definition.Owner()]; !ok {
		rejections = reject(rejections)
	}
	for _, dependency := range definition.Dependencies() {
		destination, ok := destinations[dependency.Destination()]
		if !ok || !commandSameOrAncestor(commands, destination.command, definition.Owner()) {
			rejections = reject(rejections)
		}
	}
	dynamicDestinations := make(map[ValueID]struct{}, len(domain.Uses()))
	for _, use := range domain.Uses() {
		dynamicDestinations[use.destination] = struct{}{}
	}
	for _, flag := range flags {
		if flag.localeSelector {
			if _, dynamicDestination := dynamicDestinations[flag.destination]; dynamicDestination {
				rejections = reject(rejections)
			}
		}
	}
	for _, profile := range profiles {
		selector, present := profile.Selector()
		if !present {
			continue
		}
		if _, dynamicDestination := dynamicDestinations[selector]; dynamicDestination {
			rejections = reject(rejections)
		}
	}
	return rejections
}

func validateValueDomains(commands map[CommandID]commandRecord, destinations map[ValueID]destinationRecord, flags map[FlagID]flagRecord, profiles map[GrammarProfileID]grammarProfileRecord, entries []domainEntryRecord, domains []valueDomainRecord) []rejection {
	entryByID := make(map[ValueDomainEntryID]domainEntryRecord, len(entries))
	for _, entry := range entries {
		entryByID[entry.meta.id] = entry
	}
	membership := make(map[ValueDomainEntryID]int, len(entries))
	var rejections []rejection
	for domainIndex, domain := range domains {
		rejections = append(rejections, validateDynamicValueDomain(domainIndex, domain, commands, destinations, flags, profiles)...)
		seenEntries := make(map[ValueDomainEntryID]struct{})
		for _, id := range domain.Entries() {
			entry, ok := entryByID[id]
			if !ok {
				rejections = append(rejections, rejection{
					Kind:  rejectionStructuralInvalidity,
					Order: uint64(domainIndex + 1),
				})
				continue
			}
			if _, duplicate := seenEntries[id]; duplicate {
				rejections = append(rejections, semanticRejection(entry.meta.position))
				continue
			}
			seenEntries[id] = struct{}{}
			if prior, exists := membership[id]; exists && prior != domainIndex {
				rejections = append(rejections, semanticRejection(entry.meta.position))
				continue
			}
			membership[id] = domainIndex
		}
		type domainUseKey struct {
			destination ValueID
			kind        ValueDomainUseKind
		}
		seenUses := make(map[domainUseKey]struct{})
		for _, use := range domain.Uses() {
			destination, ok := destinations[use.destination]
			if !ok || use.kind == ValueDomainUseUnknown {
				rejections = append(rejections, rejection{
					Kind:  rejectionStructuralInvalidity,
					Order: uint64(domainIndex + 1),
				})
				continue
			}
			if definition, dynamic := domain.DynamicDefinition(); dynamic &&
				!commandSameOrAncestor(commands, definition.Owner(), destination.command) {
				rejections = append(rejections, rejection{
					Kind:  rejectionStructuralInvalidity,
					Order: uint64(domainIndex + 1),
				})
				continue
			}
			key := domainUseKey(use)
			if _, duplicate := seenUses[key]; duplicate {
				rejections = append(rejections, rejection{
					Kind:  rejectionStructuralInvalidity,
					Order: uint64(domainIndex + 1),
				})
				continue
			}
			seenUses[key] = struct{}{}
		}
	}
	for _, entry := range entries {
		if entry.kind == domainEntryKindAlias {
			canonical, ok := entryByID[entry.canonical]
			entryDomain, entryMember := membership[entry.meta.id]
			canonicalDomain, canonicalMember := membership[entry.canonical]
			if !ok || canonical.kind != domainEntryKindCanonical || !entryMember || !canonicalMember ||
				canonicalDomain != entryDomain {
				rejections = append(rejections, semanticRejection(entry.meta.position))
			}
		}
		if replacement, ok := entry.Replacement(); ok {
			_, exists := entryByID[replacement]
			entryDomain, entryMember := membership[entry.meta.id]
			targetDomain, targetMember := membership[replacement]
			if !exists || !entryMember || !targetMember || targetDomain != entryDomain {
				rejections = append(rejections, semanticRejection(entry.meta.position))
			}
		}
	}
	return rejections
}

// dynamicDomainEvaluator is the erased callback retained by one static
// invocation-scoped ValueDomain definition. The owning frontend adapter supplies
// typed configuration/dependency views and receives the canonical captured
// domain program through output.
type dynamicDomainEvaluator func(context.Context, any, any, any) error

type dynamicDomainDefinition struct {
	owner        CommandID
	dependencies []dependencyRequirement
}

func acceptDynamicDomainDefinition(owner CommandID, dependencies []dependencyRequirement) dynamicDomainDefinition {
	mustCommand(owner)
	definition := dynamicDomainDefinition{
		owner:        owner,
		dependencies: append([]dependencyRequirement(nil), dependencies...),
	}

	return definition
}

func (definition dynamicDomainDefinition) Valid() bool {
	return definition.owner.present()
}

func (definition dynamicDomainDefinition) Owner() CommandID { return definition.owner }

func (definition dynamicDomainDefinition) Dependencies() []dependencyRequirement {
	return append([]dependencyRequirement(nil), definition.dependencies...)
}

type domainEntryKind uint8

const (
	domainEntryKindUnknown domainEntryKind = iota
	domainEntryKindCanonical
	domainEntryKindAlias
)

type domainEntryRecord struct {
	meta        recordMeta[ValueDomainEntryID]
	kind        domainEntryKind
	name        string
	canonical   ValueDomainEntryID
	value       any
	description string
	hidden      bool
	lifecycle   ReleaseLifecycle
	replacement ValueDomainEntryID
}

func acceptDomainEntryRecord(id ValueDomainEntryID, position declarationPosition) domainEntryRecord {
	record := domainEntryRecord{
		meta: acceptRecordMeta(id, position),
	}
	return record
}

// domainValueInput is the complete immutable payload of one canonical domain value.
type domainValueInput struct {
	ID          ValueDomainEntryID
	Position    declarationPosition
	Name        string
	Value       any
	Description string
	Hidden      bool
	Lifecycle   ReleaseLifecycle
}

func acceptDomainValueRecord(input domainValueInput) domainEntryRecord {
	record := domainEntryRecord{
		meta:        acceptRecordMeta(input.ID, input.Position),
		kind:        domainEntryKindCanonical,
		name:        input.Name,
		canonical:   input.ID,
		value:       input.Value,
		description: input.Description,
		hidden:      input.Hidden,
		lifecycle:   input.Lifecycle,
	}

	return record
}

// domainAliasInput is the complete immutable payload of one domain alias.
type domainAliasInput struct {
	ID        ValueDomainEntryID
	Position  declarationPosition
	Name      string
	Canonical ValueDomainEntryID
	Hidden    bool
	Lifecycle ReleaseLifecycle
}

func acceptDomainAliasRecord(input domainAliasInput) domainEntryRecord {
	if !input.Canonical.present() {
		panic("semantic: invalid canonical domain entry")
	}
	record := domainEntryRecord{
		meta:      acceptRecordMeta(input.ID, input.Position),
		kind:      domainEntryKindAlias,
		name:      input.Name,
		canonical: input.Canonical,
		hidden:    input.Hidden,
		lifecycle: input.Lifecycle,
	}

	return record
}

func (record domainEntryRecord) Value() (any, bool) {
	return record.value, record.kind == domainEntryKindCanonical
}

func (record domainEntryRecord) Replacement() (ValueDomainEntryID, bool) {
	return record.replacement, record.replacement.present()
}

type valueDomainUseRecord struct {
	destination ValueID
	kind        ValueDomainUseKind
}

func acceptValueDomainUseRecord(destination ValueID, kind ValueDomainUseKind) valueDomainUseRecord {
	mustDestination(destination)
	switch kind {
	case ValueDomainUseDestination,
		ValueDomainUseSequenceElement,
		ValueDomainUseMapKey,
		ValueDomainUseMapValue:
	default:
		panic("semantic: invalid value domain use")
	}
	record := valueDomainUseRecord{
		destination: destination,
		kind:        kind,
	}
	return record
}

type valueDomainRecord struct {
	entries []ValueDomainEntryID
	uses    []valueDomainUseRecord
	dynamic dynamicDomainDefinition
}

func acceptDynamicValueDomainRecord(definition dynamicDomainDefinition, uses []valueDomainUseRecord) valueDomainRecord {
	if !definition.Valid() {
		panic("semantic: invalid dynamic value domain")
	}
	record := acceptValueDomainRecord(nil, uses)
	record.dynamic = definition
	return record
}

func acceptValueDomainRecord(entries []ValueDomainEntryID, uses []valueDomainUseRecord) valueDomainRecord {
	for _, entry := range entries {
		if !entry.present() {
			panic("semantic: invalid value domain entry")
		}
	}
	for _, use := range uses {
		if !use.destination.present() || use.kind == ValueDomainUseUnknown {
			panic("semantic: invalid value domain use")
		}
	}
	record := valueDomainRecord{
		entries: append([]ValueDomainEntryID(nil), entries...),
		uses:    append([]valueDomainUseRecord(nil), uses...),
	}
	return record
}

func (record valueDomainRecord) Entries() []ValueDomainEntryID {
	return append([]ValueDomainEntryID(nil), record.entries...)
}

func (record valueDomainRecord) DynamicDefinition() (dynamicDomainDefinition, bool) {
	return record.dynamic, record.dynamic.Valid()
}

func (record valueDomainRecord) Uses() []valueDomainUseRecord {
	return append([]valueDomainUseRecord(nil), record.uses...)
}

var errDomainContributionMembership = errors.New("cli: typed contribution is not an effective domain member")

// enforceEffectiveDomainMembership applies the finite-domain membership law to
// resolved typed candidates after the operation-effective domain language is
// known. Textual candidates have already crossed the same language through
// Parse; this closes the equivalent path for defaults, derivations, facts, and
// other typed contributions that do not parse source text.
func enforceEffectiveDomainMembership(model *effectiveModel, analysis *valueAnalysis) {
	if analysis == nil {
		return
	}
	for id, resolution := range analysis.Destinations {
		if resolution.State != destinationStateResolved {
			continue
		}
		record, ok := model.Destination(id)
		if !ok {
			continue
		}
		definition, ok := record.Definition()
		if !ok || validEffectiveDomainContribution(definition, resolution.Value) {
			continue
		}
		for _, candidateID := range resolution.Contributors {
			candidate, ok := analysis.Candidates[candidateID]
			if !ok {
				continue
			}
			candidate.Meta.Disposition = ValueCandidateDispositionRejected
			analysis.Candidates[candidateID] = candidate
		}
		resolution.State = destinationStateFailed
		resolution.Value = nil
		resolution.Contributors = nil
		resolution.Err = errDomainContributionMembership
		analysis.Destinations[id] = resolution
		valueIssueValue := valueIssue{
			Kind:        valueIssueInvalidInput,
			Destination: id,
			Err:         errDomainContributionMembership,
		}
		analysis.Issues = append(analysis.Issues, valueIssueValue)
	}
}

var errInvalidDynamicValueDomain = errors.New("cli: invalid dynamic value domain")

type dynamicDomainDraft struct {
	ownerCommand      commandRef
	ownerCommandIndex int
	dependencies      []authoringDestinationDependency
	evaluate          func(context.Context, any, Dependencies) (*valueDomainState, error)
}

type dynamicDomainEvaluation struct {
	state *valueDomainState
}

func stageDynamicValueDomain[C any, T comparable](values *Values[C], provide func(context.Context, *C, Dependencies, *ValueDomainBuilder[T]) error, dependencies []Dependency) ValueDomain[T] {
	state := &valueDomainState{
		snapshot: func(value any) (any, bool) {
			typed, ok := value.(T)
			return typed, ok
		},
	}
	if values == nil || values.session == nil || provide == nil {
		state.issues = append(state.issues, "invalid dynamic domain declaration")
		domain := ValueDomain[T]{
			state: state,
		}
		return domain
	}
	descriptors, ok := dependencyDescriptors(dependencies)
	if !ok {
		state.issues = append(state.issues, "invalid dynamic domain dependency")
		domain := ValueDomain[T]{
			state: state,
		}
		return domain
	}
	var frozen []authoringDestinationDependency
	var frozenOK bool
	var owner commandRef
	values.session.use(func(draft *declarationDraft, command commandRef) {
		owner = command
		frozen, frozenOK = freezeAuthoringDestinationDependencies(draft, descriptors)
	})
	if !frozenOK {
		state.issues = append(state.issues, "invalid dynamic domain dependency provenance")
		domain := ValueDomain[T]{
			state: state,
		}
		return domain
	}
	dynamic := &dynamicDomainDraft{
		ownerCommand: owner,
		dependencies: frozen,
	}
	dynamic.evaluate = func(ctx context.Context, raw any, resolved Dependencies) (*valueDomainState, error) {
		configuration, ok := raw.(*C)
		if !ok || configuration == nil {
			panic("cli: invalid dynamic value-domain configuration")
		}
		var callbackErr error
		domain, cause := captureValueDomain(func(value T) T { return value }, func(builder *ValueDomainBuilder[T]) {
			callbackErr = provide(ctx, configuration, resolved, builder)
		})
		if cause != nil {
			return nil, cause
		}
		if callbackErr != nil {
			return nil, callbackErr
		}
		if !validValueDomainState(domain.state) {
			return nil, errInvalidDynamicValueDomain
		}
		return domain.state, nil
	}
	state.dynamic = dynamic
	domain := ValueDomain[T]{
		state: state,
	}
	return domain
}
