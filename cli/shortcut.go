package cli

import (
	"context"
	"errors"
)

var (
	errInvalidShortcutArgumentSplice = errors.New("invocation: invalid shortcut argument splice")
	errInvalidShortcut               = errors.New("invocation: invalid shortcut")
	errUnknownShortcutEntry          = errors.New("invocation: unknown shortcut entry")
	errUnknownShortcutTarget         = errors.New("invocation: unknown shortcut target")
	errInvalidShortcutDefault        = errors.New("invocation: invalid shortcut default")
)

type selectedShortcut struct {
	entry         CommandEntryID
	target        CommandID
	definition    shortcutDefinition
	input         InputOriginID
	selectorIndex int
}

func shortcutSettlementPossible(operation *invocation) bool {
	if operation == nil || operation.analysis == nil {
		return false
	}
	model := operation.analysis.effectiveModel
	if len(model.Providers()) != 0 {
		return true
	}
	for _, record := range model.CommandEntries() {
		if _, ok := record.Shortcut(); ok {
			return true
		}
	}
	return false
}

func selectedShortcutFromPrefix(operation *invocation, prefix grammarPrefixResult, expanded map[InputOriginID]struct{}, boundary, limit int) (selectedShortcut, bool) {
	if operation == nil || operation.analysis == nil {
		return selectedShortcut{}, false
	}
	for _, step := range prefix.Grammar.Selection.Steps {
		if !step.Entry.present() || !step.Input.present() {
			continue
		}
		if _, ok := expanded[step.Input]; ok {
			continue
		}
		record, ok := operation.analysis.effectiveModel.CommandEntry(step.Entry)
		if !ok {
			panic("cli: accepted shortcut selection has no command entry")
		}
		definition, ok := record.Shortcut()
		if !ok {
			continue
		}
		selectorIndex := argumentIndexForOrigin(operation, step.Input)
		if selectorIndex < 0 || selectorIndex >= boundary || selectorIndex >= limit {
			panic("cli: accepted shortcut selection has invalid input origin")
		}
		selection := selectedShortcut{
			entry:         step.Entry,
			target:        record.target,
			definition:    definition,
			input:         step.Input,
			selectorIndex: selectorIndex,
		}
		return selection, true
	}
	return selectedShortcut{}, false
}

// settleCommittedShortcuts applies selected shortcuts in lexical order before
// ordinary invocation grammar is allowed to settle beyond each selecting token.
// This keeps fixed input on the same grammar path as user input and preserves
// enclosing shortcut append suffixes while nested shortcuts are expanded.
func settleCommittedShortcuts(ctx context.Context, operation *invocation) error {
	if operation == nil || operation.analysis == nil {
		return errInvalidShortcut
	}
	if !shortcutSettlementPossible(operation) {
		return nil
	}
	expanded := make(map[InputOriginID]struct{})
	trailingAppend := 0
	for {
		settled, nextTrailingAppend, err := settleNextCommittedShortcut(ctx, operation, expanded, trailingAppend)
		if err != nil {
			return err
		}
		trailingAppend = nextTrailingAppend
		if !settled {
			return nil
		}
	}
}

func settleNextCommittedShortcut(ctx context.Context, operation *invocation, expanded map[InputOriginID]struct{}, trailingAppend int) (bool, int, error) {
	residualEnd := len(operation.Args()) - trailingAppend
	if residualEnd < 0 {
		return false, trailingAppend, errInvalidShortcutArgumentSplice
	}
	for boundary := 1; boundary <= residualEnd; boundary++ {
		prefix := analyzeGrammarPrefix(
			ctx,
			operation,
			boundary,
			operation.namespaceFor(ctx),
			operation.conditionFor(ctx),
			operation.defaultCommandFor(ctx),
		)
		if _, supplyErr := operation.supplyFailure(); supplyErr != nil {
			// The caller owns the already-recorded supply failure; shortcut settlement
			// only stops before adding further semantic transitions.
			return false, trailingAppend, nil //nolint:nilerr
		}
		selection, ok := selectedShortcutFromPrefix(operation, prefix, expanded, boundary, residualEnd)
		if !ok {
			continue
		}
		prepend := selection.definition.Args()
		appendArgs := selection.definition.AppendArgs()
		_, err := operation.ApplyShortcut(
			selection.selectorIndex,
			residualEnd,
			selection.entry,
			selection.target,
			prepend,
			appendArgs,
			selection.definition.Defaults(),
			selection.definition.Source().Sensitive(),
		)
		if err != nil {
			return false, trailingAppend, err
		}
		expanded[selection.input] = struct{}{}

		prependStart := selection.selectorIndex + 1
		prependStop := prependStart + len(prepend)
		settledPrependStop, err := settleResponseRange(
			ctx,
			operation,
			operation.analysis,
			prependStart,
			prependStop,
			-1,
		)
		if err != nil {
			return false, trailingAppend, err
		}
		prependDelta := settledPrependStop - prependStop
		appendStart := residualEnd + len(prepend) + prependDelta
		appendStop := appendStart + len(appendArgs)
		settledAppendStop, err := settleResponseRange(
			ctx,
			operation,
			operation.analysis,
			appendStart,
			appendStop,
			-1,
		)
		if err != nil {
			return false, trailingAppend, err
		}
		return true, trailingAppend + settledAppendStop - appendStart, nil
	}
	return false, trailingAppend, nil
}

func argumentIndexForOrigin(operation *invocation, origin InputOriginID) int {
	if operation == nil {
		return -1
	}
	for index := range operation.Args() {
		ref, ok := operation.ArgumentOrigin(index)
		if ok && ref.ID() == origin {
			return index
		}
	}
	return -1
}

// ApplyShortcut commits one selected shortcut as a single operation transition:
// generated argv/provenance and its fallback defaults cannot be published
// independently.
func (operation *invocation) ApplyShortcut(selectorIndex, residualEnd int, entry CommandEntryID, target CommandID, prepend, appendArgs []string, defaults []shortcutDefault, sensitive bool) (int, error) {
	if operation == nil || !entry.present() || !target.present() {
		return 0, errInvalidShortcut
	}
	if _, ok := operation.analysis.effectiveModel.CommandEntry(entry); !ok {
		return 0, errUnknownShortcutEntry
	}
	if _, ok := operation.analysis.effectiveModel.Command(target); !ok {
		return 0, errUnknownShortcutTarget
	}
	for _, value := range defaults {
		if !value.Valid() {
			return 0, errInvalidShortcutDefault
		}
	}
	inserted, err := operation.spliceShortcutArguments(
		selectorIndex,
		residualEnd,
		entry,
		prepend,
		appendArgs,
		sensitive,
	)
	if err != nil {
		return 0, err
	}
	operation.addShortcutDefaults(entry, target, defaults)
	return inserted, nil
}

// spliceShortcutArguments applies one selected shortcut's fixed decoded input
// to the operation-local grammar view. Prepend tokens follow the selecting
// shortcut token. Append tokens are inserted at residualEnd, after the complete
// residual user-input slice and before any already-settled enclosing shortcut
// append suffix. Every fixed token is generated from one shortcut-owned source
// and retains the selecting token as its parent origin.
func (operation *invocation) spliceShortcutArguments(selectorIndex, residualEnd int, entry CommandEntryID, prepend, appendArgs []string, sensitive bool) (int, error) {
	if operation == nil || selectorIndex < 0 || selectorIndex >= len(operation.analysis.args) || residualEnd < selectorIndex+1 ||
		residualEnd > len(operation.analysis.args) ||
		!entry.present() {
		return 0, errInvalidShortcutArgumentSplice
	}
	if len(prepend) == 0 && len(appendArgs) == 0 {
		return 0, nil
	}
	parent := operation.analysis.provenance.argumentOrigins[selectorIndex]
	tokens := make([]string, 0, len(prepend)+len(appendArgs))
	tokens = append(tokens, prepend...)
	tokens = append(tokens, appendArgs...)
	source := operation.addShortcutSource(entry, tokens, sensitive)
	origins := make([]originRef, len(tokens))
	for i, token := range tokens {
		spanRefValue := spanRef{
			Source: source,
			Entry:  i,
			Start:  0,
			End:    len(token),
		}
		origins[i] = operation.addOrigin(spanRefValue, []originRef{parent}, sensitive)
	}

	prependOrigins := origins[:len(prepend)]
	appendOrigins := origins[len(prepend):]
	args := make([]string, 0, len(operation.analysis.args)+len(tokens))
	args = append(args, operation.analysis.args[:selectorIndex+1]...)
	args = append(args, prepend...)
	args = append(args, operation.analysis.args[selectorIndex+1:residualEnd]...)
	args = append(args, appendArgs...)
	args = append(args, operation.analysis.args[residualEnd:]...)
	argumentOrigins := make([]originRef, 0, len(operation.analysis.provenance.argumentOrigins)+len(origins))
	argumentOrigins = append(argumentOrigins, operation.analysis.provenance.argumentOrigins[:selectorIndex+1]...)
	argumentOrigins = append(argumentOrigins, prependOrigins...)
	argumentOrigins = append(argumentOrigins, operation.analysis.provenance.argumentOrigins[selectorIndex+1:residualEnd]...)
	argumentOrigins = append(argumentOrigins, appendOrigins...)
	argumentOrigins = append(argumentOrigins, operation.analysis.provenance.argumentOrigins[residualEnd:]...)

	for i := range operation.analysis.provenance.responseIncludes {
		position := operation.analysis.provenance.responseIncludes[i].BeforeArgument
		if position >= selectorIndex+1 {
			position += len(prepend)
		}
		adjustedResidualEnd := residualEnd + len(prepend)
		if position >= adjustedResidualEnd {
			position += len(appendArgs)
		}
		operation.analysis.provenance.responseIncludes[i].BeforeArgument = position
	}
	operation.analysis.args = args
	operation.analysis.provenance.argumentOrigins = argumentOrigins
	return len(tokens), nil
}

// ShortcutInputEntries returns the detached fixed-token snapshot for one
// shortcut source. Other source kinds have no entries here.
func (operation *invocation) ShortcutInputEntries(id InputSourceID) []string {
	if operation == nil || operation.analysis.provenance.shortcutInputEntries == nil {
		return nil
	}
	return append([]string(nil), operation.analysis.provenance.shortcutInputEntries[id]...)
}

// CommandShortcutDeclaration identifies one staged static command shortcut. It supports typed
// defaults and replacement while its authoring scope is active. The zero value is invalid; copies
// identify the same declaration, and retained, concurrent, or cross-scope use is builder lifecycle
// misuse.
type CommandShortcutDeclaration[C any] struct {
	state    *declarationSession
	shortcut shortcutDraftRef
}

// Default adds a typed default for destination to this statically declared shortcut and returns
// the same shortcut declaration for further authoring.
func (shortcut CommandShortcutDeclaration[C]) Default[O, T any](destination destinationLike[O, T], value T, options ...SourceOption) CommandShortcutDeclaration[C] {
	return commandShortcutDefault(shortcut, destination, value, options)
}

// ReplacedBy declares that this shortcut is replaced by target within the same authoring scope.
func (shortcut CommandShortcutDeclaration[C]) ReplacedBy(target CommandShortcutDeclaration[C]) CommandShortcutDeclaration[C] {
	commandShortcutReplacement(shortcut, target)
	return shortcut
}

type commandShortcutDirectiveKind uint8

const (
	commandShortcutDirectiveInvalid commandShortcutDirectiveKind = iota
	commandShortcutDirectiveDefault
	commandShortcutDirectiveReplacement
)

// CommandShortcut describes a static shortcut for the current command. Its effective input is
// Args, then residual user input, then AppendArgs; each string is one decoded argument token. A
// shortcut must contribute fixed input or a typed default rather than acting as a name-only alias.
type CommandShortcut struct {
	Name string

	Description Description
	Examples    []Example

	Args       []string
	AppendArgs []string

	Hidden bool

	Release ReleaseLifecycle
}

// CommandShortcutBinding identifies one shortcut staged by the active provider batch. It may add
// destination or flag defaults and declare replacement while that batch is active. The zero value
// is invalid; copies identify the same staged shortcut and must not outlive the provider callback.
type CommandShortcutBinding struct {
	session  *declarationSession
	shortcut shortcutRef
}

// Default adds a typed destination default to the provider-contributed shortcut and returns the
// binding for further batch authoring.
func (binding CommandShortcutBinding) Default[O, T any](destination destinationLike[O, T], value T, options ...SourceOption) CommandShortcutBinding {
	return providerShortcutTypedDefault(binding, destination, value, options)
}

// DefaultFlag adds raw values for the named flag spelling to the provider-contributed shortcut and
// returns the binding for further batch authoring.
func (binding CommandShortcutBinding) DefaultFlag(spelling string, values []string, options ...SourceOption) CommandShortcutBinding {
	return providerShortcutFlagDefault(binding, spelling, values, options)
}

// ReplacedBy declares that this provider-contributed shortcut is replaced by target. Both bindings
// must belong to the same active provider batch.
func (binding CommandShortcutBinding) ReplacedBy(target CommandShortcutBinding) CommandShortcutBinding {
	providerShortcutReplacedBy(binding, target)
	return binding
}

func declareCommandShortcut[C any](command *CommandBuilder[C], shortcut CommandShortcut) CommandShortcutDeclaration[C] {
	if command == nil || command.session == nil {
		return CommandShortcutDeclaration[C]{}
	}
	staged := shortcutDraft{
		name:        shortcut.Name,
		description: shortcut.Description,
		examples:    shortcut.Examples,
		args:        shortcut.Args,
		appendArgs:  shortcut.AppendArgs,
		hidden:      shortcut.Hidden,
		release:     shortcut.Release,
		valid:       shortcut.Name != "",
	}
	state := stageCommandShortcut(command.session, command.profile, staged)
	return CommandShortcutDeclaration[C]{
		state:    command.session,
		shortcut: state,
	}
}

func commandShortcutDefault[C, O, T any](shortcut CommandShortcutDeclaration[C], destination destinationLike[O, T], value T, options []SourceOption) CommandShortcutDeclaration[C] {
	if shortcut.state == nil || !shortcut.shortcut.valid() || destination == nil {
		return shortcut
	}
	stageCommandShortcutDefault(shortcut.state, shortcut.shortcut, destination.destinationHandle(), value, options)
	return shortcut
}

func commandShortcutReplacement[C any](shortcut, target CommandShortcutDeclaration[C]) {
	if shortcut.state == nil || !shortcut.shortcut.valid() || !target.shortcut.valid() {
		return
	}
	stageCommandShortcutReplacement(shortcut.state, shortcut.shortcut, target.shortcut)
}

func stageCommandShortcut(session *declarationSession, profile grammarProfileCaseRef, shortcut shortcutDraft) shortcutDraftRef {
	if session == nil {
		return shortcutDraftRef{}
	}

	var ref shortcutDraftRef
	session.use(func(draft *declarationDraft, command commandRef) {
		record := shortcutDraft{
			position:    draft.nextPosition(),
			command:     command,
			profile:     profile,
			name:        shortcut.name,
			description: shortcut.description,
			examples:    append([]Example(nil), shortcut.examples...),
			args:        append([]string(nil), shortcut.args...),
			appendArgs:  append([]string(nil), shortcut.appendArgs...),
			hidden:      shortcut.hidden,
			release:     shortcut.release,
			valid:       shortcut.valid,
		}
		ref = draft.shortcuts.append(draft, record)
	})
	return ref
}

func stageCommandShortcutDefault(session *declarationSession, shortcut shortcutDraftRef, destination *destinationHandle, value any, options []SourceOption) {
	if session == nil || !shortcut.valid() || destination == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		relation, relationOK := captureAuthoringDestinationIdentity(draft, destination)
		owner, ownerOK := draft.shortcuts.resolve(draft, shortcut)
		profile := grammarProfileCaseRef{}
		if ownerOK {
			profile = owner.profile
		}
		record := shortcutDirectiveDraft{
			position:      draft.nextPosition(),
			command:       command,
			profile:       profile,
			shortcut:      shortcut,
			kind:          commandShortcutDirectiveDefault,
			destination:   relation,
			value:         snapshotMutableValue(value),
			sourceOptions: append([]SourceOption(nil), options...),
			valid:         ownerOK && relationOK,
		}
		draft.shortcutDirectives.append(draft, record)
	})
}

func stageCommandShortcutReplacement(session *declarationSession, shortcut, replacement shortcutDraftRef) {
	if session == nil || !shortcut.valid() || !replacement.valid() {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		owner, ownerOK := draft.shortcuts.resolve(draft, shortcut)
		_, replacementOK := draft.shortcuts.resolve(draft, replacement)
		profile := grammarProfileCaseRef{}
		if ownerOK {
			profile = owner.profile
		}
		record := shortcutDirectiveDraft{
			position:    draft.nextPosition(),
			command:     command,
			profile:     profile,
			shortcut:    shortcut,
			kind:        commandShortcutDirectiveReplacement,
			replacement: replacement,
			valid:       ownerOK && replacementOK,
		}
		draft.shortcutDirectives.append(draft, record)
	})
}

func stageStaticShortcuts(draft *declarationCandidate, input declarationPlan, source *declarationDraft, state *staticAcceptanceState) error {
	defaults := make([][]shortcutDefault, len(input.records))
	replacements := make([]int, len(input.records))
	for index := range replacements {
		replacements[index] = -1
	}
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordShortcutDirective {
			continue
		}
		directive := source.shortcutDirectives.entries[record.familyIndex].value
		shortcutIndex, indexed := input.indexes.shortcuts[directive.shortcut]
		if !indexed {
			panic("cli: validated shortcut directive has no shortcut record")
		}
		slot := state.mustSlot(shortcutIndex)
		if !slot.shortcutEntry.ID().present() {
			panic("cli: accepted shortcut entry identity is unavailable")
		}
		switch directive.kind {
		case commandShortcutDirectiveDefault:
			id := state.mustValueID(directive.destination)
			definition, defined := draft.ExistingDestinationDefinition(id)
			if !defined || !validEffectiveDomainContribution(definition, directive.value) {
				return errStaticDeclarationRejected
			}
			source, ok := producerSource(directive.sourceOptions)
			if !ok {
				return errStaticDeclarationRejected
			}
			defaults[shortcutIndex] = append(
				defaults[shortcutIndex],
				acceptShortcutDestinationDefault(id, directive.value, source),
			)
		case commandShortcutDirectiveReplacement:
			targetIndex, indexed := input.indexes.shortcuts[directive.replacement]
			if !indexed {
				panic("cli: validated shortcut replacement has no target record")
			}
			target := state.mustSlot(targetIndex)
			if !target.shortcutEntry.ID().present() {
				panic("cli: accepted shortcut replacement identity is unavailable")
			}
			if replacements[shortcutIndex] >= 0 {
				return errStaticDeclarationRejected
			}
			replacements[shortcutIndex] = targetIndex
		default:
			return errStaticDeclarationRejected
		}
	}

	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordShortcut {
			continue
		}
		entry := state.slots[index].shortcutEntry
		if !entry.ID().present() {
			panic("cli: accepted shortcut entry identity is unavailable")
		}
		definition := acceptShortcutDefinition(
			source.shortcuts.entries[record.familyIndex].value.args,
			source.shortcuts.entries[record.familyIndex].value.appendArgs,
			defaults[index],
			snapshotProducerSource(nil, false),
		)
		if !definition.HasContribution() {
			return errStaticDeclarationRejected
		}
		position := input.positionFor(index)
		draft.CommandEntryShortcut(entry, definition, position)
		if targetIndex := replacements[index]; targetIndex >= 0 {
			target := state.slots[targetIndex].shortcutEntry
			if !target.ID().present() {
				panic("cli: accepted shortcut replacement identity is unavailable")
			}
			draft.CommandEntryReplacement(entry, target, position)
		}
	}
	return nil
}

type shortcutDefaultKind uint8

const (
	shortcutDefaultKindUnknown shortcutDefaultKind = iota
	shortcutDefaultKindDestination
	shortcutDefaultKindFlag
)

type shortcutDefault struct {
	kind        shortcutDefaultKind
	destination ValueID
	value       any
	spelling    string
	values      []string
	source      semanticProducerSource
}

func acceptShortcutDestinationDefault(destination ValueID, value any, source semanticProducerSource) shortcutDefault {
	mustDestination(destination)
	result := shortcutDefault{
		kind:        shortcutDefaultKindDestination,
		destination: destination,
		value:       value,
		source:      source,
	}

	return result
}

func acceptShortcutFlagDefault(spelling string, values []string, source semanticProducerSource) shortcutDefault {
	if spelling == "" || len(values) == 0 {
		panic("semantic: invalid shortcut flag default")
	}
	result := shortcutDefault{
		kind:     shortcutDefaultKindFlag,
		spelling: spelling,
		values:   append([]string(nil), values...),
		source:   source,
	}

	return result
}

func (value shortcutDefault) Valid() bool {
	switch value.kind {
	case shortcutDefaultKindDestination:
		return value.destination.present() && value.spelling == "" && len(value.values) == 0
	case shortcutDefaultKindFlag:
		return !value.destination.present() && value.spelling != "" && len(value.values) != 0
	default:
		return false
	}
}

func (value shortcutDefault) Kind() shortcutDefaultKind { return value.kind }

func (value shortcutDefault) Destination() ValueID { return value.destination }

func (value shortcutDefault) Value() any { return value.value }

func (value shortcutDefault) Spelling() string { return value.spelling }

func (value shortcutDefault) Values() []string { return append([]string(nil), value.values...) }

func (value shortcutDefault) Source() semanticProducerSource {
	return snapshotProducerSource(value.source.Annotations(), value.source.Sensitive())
}

type shortcutDefinition struct {
	args       []string
	appendArgs []string
	defaults   []shortcutDefault
	source     semanticProducerSource
}

func acceptShortcutDefinition(args, appendArgs []string, defaults []shortcutDefault, source semanticProducerSource) shortcutDefinition {
	definition := shortcutDefinition{
		args:       append([]string(nil), args...),
		appendArgs: append([]string(nil), appendArgs...),
		defaults:   append([]shortcutDefault(nil), defaults...),
		source:     snapshotProducerSource(source.Annotations(), source.Sensitive()),
	}
	for _, value := range definition.defaults {
		if !value.Valid() {
			panic("semantic: invalid shortcut definition")
		}
	}
	return definition
}

func (definition shortcutDefinition) Args() []string {
	return append([]string(nil), definition.args...)
}

func (definition shortcutDefinition) AppendArgs() []string {
	return append([]string(nil), definition.appendArgs...)
}

func (definition shortcutDefinition) Defaults() []shortcutDefault {
	return append([]shortcutDefault(nil), definition.defaults...)
}

func (definition shortcutDefinition) Source() semanticProducerSource {
	return snapshotProducerSource(definition.source.Annotations(), definition.source.Sensitive())
}

func (definition shortcutDefinition) HasContribution() bool {
	return len(definition.args) != 0 || len(definition.appendArgs) != 0 || len(definition.defaults) != 0
}
