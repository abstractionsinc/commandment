package cli

import (
	"errors"
	"io"
)

func prepareExecutionAccess(declarations []accessDefinition, resolver destinationAcceptanceResolver) (executionAccess, bool) {
	var values []executionAccessValue
	var optionalValues []executionAccessValue
	var valueStates []executionAccessValue
	var access executionAccess
	for _, declaration := range declarations {
		if !declaration.target.Valid() {
			return executionAccess{}, false
		}
		switch declaration.kind {
		case accessDeclarationValue, accessDeclarationOptionalValue, accessDeclarationValueState:
			if resolver == nil {
				return executionAccess{}, false
			}
			source, ok := resolver.ValueID(declaration.source)
			if !ok || !source.present() {
				return executionAccess{}, false
			}
			binding := acceptExecutionAccessValue(source, declaration.target, declaration.prepare)
			switch declaration.kind {
			case accessDeclarationValue:
				values = append(values, binding)
			case accessDeclarationOptionalValue:
				optionalValues = append(optionalValues, binding)
			case accessDeclarationValueState:
				valueStates = append(valueStates, binding)
			case accessDeclarationInvalid,
				accessDeclarationLocale,
				accessDeclarationCommandSelection,
				accessDeclarationWorkingDirectory,
				accessDeclarationStdin,
				accessDeclarationStdout,
				accessDeclarationStderr:
			}
		case accessDeclarationInvalid,
			accessDeclarationLocale,
			accessDeclarationCommandSelection,
			accessDeclarationWorkingDirectory,
			accessDeclarationStdin,
			accessDeclarationStdout,
			accessDeclarationStderr:
		}
	}
	access = acceptExecutionAccess(values, optionalValues, valueStates)
	for _, declaration := range declarations {
		switch declaration.kind {
		case accessDeclarationLocale:
			access = access.WithLocale(declaration.target)
		case accessDeclarationCommandSelection:
			access = access.WithCommandSelection(declaration.target)
		case accessDeclarationWorkingDirectory:
			access = access.WithWorkingDirectory(declaration.target)
		case accessDeclarationStdin:
			access = access.WithStdin(declaration.target)
		case accessDeclarationStdout:
			access = access.WithStdout(declaration.target)
		case accessDeclarationStderr:
			access = access.WithStderr(declaration.target)
		case accessDeclarationInvalid,
			accessDeclarationValue,
			accessDeclarationOptionalValue,
			accessDeclarationValueState:
		}
	}
	return access, true
}

type accessDefinition struct {
	kind    accessDeclarationKind
	source  destinationIdentity
	target  configurationBinding
	prepare executionAccessPrepare
}

// OptionalValue distinguishes a present value from semantic absence; Present remains true even when
// Value is a Go zero value or nil.
type OptionalValue[T any] struct {
	Value   T
	Present bool
}

// ExecutionAccess is the detached declaration summary of passive access a command requires before
// application execution. Value ID slices retain deterministic canonical order within their
// categories; its zero value means no declared access.
type ExecutionAccess struct {
	Values         []ValueID
	OptionalValues []ValueID
	ValueStates    []ValueID

	Locale           bool
	CommandSelection bool
	WorkingDirectory bool

	Stdin  bool
	Stdout bool
	Stderr bool
}

// accessCapability is the command-scoped fluent surface behind CommandBuilder.Access. Its methods
// declare values or invocation facilities to populate into the command configuration atomically
// before middleware or handlers begin.
type accessCapability[C any] struct {
	config  *C
	session *declarationSession
	profile grammarProfileCaseRef
}

type accessDeclarationKind uint8

const (
	accessDeclarationInvalid accessDeclarationKind = iota
	accessDeclarationValue
	accessDeclarationOptionalValue
	accessDeclarationValueState
	accessDeclarationLocale
	accessDeclarationCommandSelection
	accessDeclarationWorkingDirectory
	accessDeclarationStdin
	accessDeclarationStdout
	accessDeclarationStderr
)

// Value declares a required execution dependency on source and snapshots its finalized value into
// target before application execution.
func (access accessCapability[C]) Value[O, T any](source destinationLike[O, T], target *T) {
	prepare := func(raw any, present bool) (any, bool) {
		if !present {
			return nil, false
		}
		typed, ok := raw.(T)
		if !ok {
			return nil, false
		}
		return snapshotMutableValue(typed), true
	}
	stageTypedAccess(access.config, access.session, access.profile, accessDeclarationValue, destinationHandleOf(source), target, prepare)
}

// OptionalValue declares absence-tolerant access to source. Resolved absence produces Present
// false; a real value produces Present true, while source failure or blocking remains failure or
// blocking.
func (access accessCapability[C]) OptionalValue[O, T any](source destinationLike[O, T], target *OptionalValue[T]) {
	prepare := func(raw any, present bool) (any, bool) {
		if !present {
			return OptionalValue[T]{}, true
		}
		typed, ok := raw.(T)
		if !ok {
			return nil, false
		}
		snapshot, ok := snapshotMutableValue(typed).(T)
		if !ok {
			return nil, false
		}
		value := OptionalValue[T]{
			Value:   snapshot,
			Present: true,
		}
		return value, true
	}
	stageTypedAccess(
		access.config,
		access.session,
		access.profile,
		accessDeclarationOptionalValue,
		destinationHandleOf(source),
		target,
		prepare,
	)
}

// ValueState declares observational access to the finalized state of source without creating a
// resolution dependency or changing scheduling.
func (access accessCapability[C]) ValueState[O, T any](source destinationLike[O, T], target *OptionalValue[ValueState]) {
	prepare := func(raw any, present bool) (any, bool) {
		if !present {
			return OptionalValue[ValueState]{}, true
		}
		state, ok := raw.(ValueState)
		if !ok {
			return nil, false
		}
		value := OptionalValue[ValueState]{
			Value:   state,
			Present: true,
		}
		return value, true
	}
	stageTypedAccess(
		access.config,
		access.session,
		access.profile,
		accessDeclarationValueState,
		destinationHandleOf(source),
		target,
		prepare,
	)
}

// Locale populates target with the finalized effective invocation locale before application
// execution.
func (access accessCapability[C]) Locale(target *Locale) {
	stageFacilityAccess(access.config, access.session, access.profile, accessDeclarationLocale, target)
}

// CommandSelection populates target with a detached snapshot of the finalized effective command
// selection.
func (access accessCapability[C]) CommandSelection(target *CommandSelection) {
	stageFacilityAccess(access.config, access.session, access.profile, accessDeclarationCommandSelection, target)
}

// WorkingDirectory populates target with the effective invocation-relative working directory.
func (access accessCapability[C]) WorkingDirectory(target *string) {
	stageFacilityAccess(access.config, access.session, access.profile, accessDeclarationWorkingDirectory, target)
}

// Stdin populates target with the effective borrowed invocation input facility; it does not
// strengthen the facility's existing nil or lifetime guarantees.
func (access accessCapability[C]) Stdin(target *io.Reader) {
	stageFacilityAccess(access.config, access.session, access.profile, accessDeclarationStdin, target)
}

// Stdout populates target with the effective borrowed invocation output facility; it does not
// strengthen the facility's existing nil or lifetime guarantees.
func (access accessCapability[C]) Stdout(target *io.Writer) {
	stageFacilityAccess(access.config, access.session, access.profile, accessDeclarationStdout, target)
}

// Stderr populates target with the effective borrowed invocation error-output facility; it does not
// strengthen the facility's existing nil or lifetime guarantees.
func (access accessCapability[C]) Stderr(target *io.Writer) {
	stageFacilityAccess(access.config, access.session, access.profile, accessDeclarationStderr, target)
}

func stageTypedAccess[C, T any](config *C, session *declarationSession, profile grammarProfileCaseRef, kind accessDeclarationKind, source *destinationHandle, target *T, prepare executionAccessPrepare) {
	binding, ok := bindConfigurationStorage(config, target)
	access := accessDraft{
		kind:    kind,
		target:  binding,
		prepare: prepare,
		valid:   ok && source != nil && prepare != nil,
	}
	stageAccess(session, profile, access, source)
}

func stageFacilityAccess[C, T any](config *C, session *declarationSession, profile grammarProfileCaseRef, kind accessDeclarationKind, target *T) {
	binding, ok := bindConfigurationStorage(config, target)
	access := accessDraft{
		kind:   kind,
		target: binding,
		valid:  ok,
	}
	stageAccess(session, profile, access, nil)
}

func stageAccess(session *declarationSession, profile grammarProfileCaseRef, access accessDraft, source *destinationHandle) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		if source != nil {
			identity, ok := captureAuthoringDestinationIdentity(draft, source)
			access.source = identity
			access.valid = access.valid && ok
		}

		record := accessDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  profile,
			kind:     access.kind,
			source:   access.source,
			target:   access.target,
			prepare:  access.prepare,
			valid:    access.valid,
		}
		draft.accesses.append(draft, record)
	})
}

func captureAuthoringDestinationIdentity(draft *declarationDraft, destination *destinationHandle) (destinationIdentity, bool) {
	if draft == nil || destination == nil {
		return nil, false
	}
	identity := destination.currentIdentity()
	switch identity := identity.(type) {
	case stagedDestinationIdentity:
		if _, ok := draft.destinations.resolve(draft, identity.ref); !ok {
			return nil, false
		}
		return identity, true
	case acceptedDestinationIdentity:
		if !identity.id.present() || !draft.scope.includes(identity.scope) {
			return nil, false
		}
		return identity, true
	default:
		return nil, false
	}
}

// executionAccessPrepare adapts one exact erased execution-access value relationship.
// Source semantic identity and target typed storage binding are both fixed at
// compilation; no runtime lookup rediscovers either relationship.
type executionAccessPrepare func(value any, present bool) (any, bool)

type executionAccessValue struct {
	source  ValueID
	target  configurationBinding
	prepare executionAccessPrepare
}

func acceptExecutionAccessValue(source ValueID, target configurationBinding, prepare executionAccessPrepare) executionAccessValue {
	if !source.present() || !target.Valid() || prepare == nil {
		panic("semantic: invalid execution access value")
	}
	result := executionAccessValue{
		source:  source,
		target:  target,
		prepare: prepare,
	}

	return result
}

func (value executionAccessValue) Source() ValueID { return value.source }

func (value executionAccessValue) Target() configurationBinding { return value.target }

func (value executionAccessValue) Prepare(raw any, present bool) (any, bool) {
	if !value.Valid() {
		return nil, false
	}
	return value.prepare(raw, present)
}

func (value executionAccessValue) Valid() bool {
	return value.source.present() && value.target.Valid() && value.prepare != nil
}

// executionAccess is the immutable private declaration/realization plan for one
// command. Facility bindings intentionally remain separate because no public
// heterogeneous declaration order exists across access categories.
type executionAccess struct {
	values         []executionAccessValue
	optionalValues []executionAccessValue
	valueStates    []executionAccessValue

	locale           configurationBinding
	commandSelection configurationBinding
	workingDirectory configurationBinding
	stdin            configurationBinding
	stdout           configurationBinding
	stderr           configurationBinding
	invalid          bool
}

func acceptExecutionAccess(values, optionalValues, valueStates []executionAccessValue) executionAccess {
	access := executionAccess{
		values:         append([]executionAccessValue(nil), values...),
		optionalValues: append([]executionAccessValue(nil), optionalValues...),
		valueStates:    append([]executionAccessValue(nil), valueStates...),
	}
	bindings := access.bindings()
	for index, binding := range bindings {
		for _, prior := range bindings[:index] {
			if semanticConfigurationBindingsOverlap(prior, binding) {
				access.invalid = true
			}
		}
	}
	return access
}

func (access executionAccess) WithLocale(binding configurationBinding) executionAccess {
	if access.locale.Valid() || access.overlaps(binding) {
		access.invalid = true
	}
	access.locale = binding
	return access
}

func (access executionAccess) WithCommandSelection(binding configurationBinding) executionAccess {
	if access.commandSelection.Valid() || access.overlaps(binding) {
		access.invalid = true
	}
	access.commandSelection = binding
	return access
}

func (access executionAccess) WithWorkingDirectory(binding configurationBinding) executionAccess {
	if access.workingDirectory.Valid() || access.overlaps(binding) {
		access.invalid = true
	}
	access.workingDirectory = binding
	return access
}

func (access executionAccess) WithStdin(binding configurationBinding) executionAccess {
	if access.stdin.Valid() || access.overlaps(binding) {
		access.invalid = true
	}
	access.stdin = binding
	return access
}

func (access executionAccess) WithStdout(binding configurationBinding) executionAccess {
	if access.stdout.Valid() || access.overlaps(binding) {
		access.invalid = true
	}
	access.stdout = binding
	return access
}

func (access executionAccess) WithStderr(binding configurationBinding) executionAccess {
	if access.stderr.Valid() || access.overlaps(binding) {
		access.invalid = true
	}
	access.stderr = binding
	return access
}

func (access executionAccess) Values() []executionAccessValue {
	return append([]executionAccessValue(nil), access.values...)
}

func (access executionAccess) OptionalValues() []executionAccessValue {
	return append([]executionAccessValue(nil), access.optionalValues...)
}

func (access executionAccess) ValueStates() []executionAccessValue {
	return append([]executionAccessValue(nil), access.valueStates...)
}

func (access executionAccess) Locale() (configurationBinding, bool) {
	return access.locale, access.locale.Valid()
}

func (access executionAccess) CommandSelection() (configurationBinding, bool) {
	return access.commandSelection, access.commandSelection.Valid()
}

func (access executionAccess) WorkingDirectory() (configurationBinding, bool) {
	return access.workingDirectory, access.workingDirectory.Valid()
}

func (access executionAccess) Stdin() (configurationBinding, bool) {
	return access.stdin, access.stdin.Valid()
}

func (access executionAccess) Stdout() (configurationBinding, bool) {
	return access.stdout, access.stdout.Valid()
}

func (access executionAccess) Stderr() (configurationBinding, bool) {
	return access.stderr, access.stderr.Valid()
}

func (access executionAccess) Valid() bool { return !access.invalid }

const executionFacilityBindingCount = 6

func (access executionAccess) bindings() []configurationBinding {
	bindings := make([]configurationBinding, 0, len(access.values)+len(access.optionalValues)+len(access.valueStates)+executionFacilityBindingCount)
	for _, value := range access.values {
		bindings = append(bindings, value.Target())
	}
	for _, value := range access.optionalValues {
		bindings = append(bindings, value.Target())
	}
	for _, value := range access.valueStates {
		bindings = append(bindings, value.Target())
	}
	for _, binding := range []configurationBinding{access.locale, access.commandSelection, access.workingDirectory, access.stdin, access.stdout, access.stderr} {
		if binding.Valid() {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

func (access executionAccess) overlaps(binding configurationBinding) bool {
	for _, existing := range access.bindings() {
		if semanticConfigurationBindingsOverlap(existing, binding) {
			return true
		}
	}
	return false
}

func (access executionAccess) Bindings() []configurationBinding {
	return append([]configurationBinding(nil), access.bindings()...)
}

func (access executionAccess) Empty() bool {
	return len(access.values) == 0 && len(access.optionalValues) == 0 && len(access.valueStates) == 0 &&
		!access.locale.Valid() && !access.commandSelection.Valid() && !access.workingDirectory.Valid() &&
		!access.stdin.Valid() && !access.stdout.Valid() && !access.stderr.Valid()
}

var errExecutionAccessPreparation = errors.New("cli: execution access preparation failed")

type executionAccessWrite struct {
	configuration any
	binding       configurationBinding
	value         any
}

// realizeExecutionAccess prepares every declared passive execution projection
// from finalized invocation truth before mutating application-visible
// configuration storage. Only after the complete selected-path plan is
// prepared successfully are the staged writes committed.
func realizeExecutionAccess(snapshot *runState, model *effectiveModel, grammar grammarResult, values valueAnalysis) (Locale, CommandSelection, map[uint32]ValueState, error) {
	selection := projectGrammarSelection(grammar.Selection)
	selector, err := resolvedLocaleSelector(model, grammar, values)
	if err != nil {
		return "", CommandSelection{}, nil, errExecutionAccessPreparation
	}
	locale, err := selectEffectiveLocale(
		string(snapshot.invocation.input.locale),
		string(selector),
		model.Application().localization.SourceLocale(),
	)
	if err != nil {
		return "", CommandSelection{}, nil, errExecutionAccessPreparation
	}

	workingDirectory := snapshot.invocation.input.workingDirectory

	states := make(map[uint32]ValueState)
	writes := make([]executionAccessWrite, 0)
	for _, command := range grammar.Selection.Path {
		record, ok := model.Command(command)
		if !ok {
			return "", CommandSelection{}, nil, errExecutionAccessPreparation
		}
		configuration, ok := snapshot.configurations.configuration(command)
		if !ok {
			return "", CommandSelection{}, nil, errExecutionAccessPreparation
		}

		for _, destination := range model.DestinationsForCommand(command) {
			resolution, resolved := values.Destinations[destination.meta.id]
			if !resolved || resolution.State != destinationStateResolved {
				continue
			}
			binding, ok := destination.Binding()
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			definition, ok := destination.Definition()
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			value, ok := definition.Snapshot(resolution.Value)
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			writes = append(writes, executionAccessWrite{
				configuration: configuration,
				binding:       binding,
				value:         value,
			})
		}

		access := record.access
		if access.Empty() {
			continue
		}

		for _, item := range access.Values() {
			resolution, ok := values.Destinations[item.Source()]
			if !ok || resolution.State != destinationStateResolved {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			value, ok := item.Prepare(resolution.Value, true)
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			executionAccessWriteValue := executionAccessWrite{
				configuration: configuration,
				binding:       item.Target(),
				value:         value,
			}
			writes = append(writes, executionAccessWriteValue)
		}

		for _, item := range access.OptionalValues() {
			resolution, ok := values.Destinations[item.Source()]
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			var raw any
			present := false
			switch resolution.State {
			case destinationStateResolved:
				raw, present = resolution.Value, true
			case destinationStateAbsent, destinationStateInactive:
				// Resolved semantic absence is the only tolerated missing value.
			case destinationStateFailed,
				destinationStateBlocked,
				destinationStateInvalid:
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			default:
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			value, ok := item.Prepare(raw, present)
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			executionAccessWriteValue2 := executionAccessWrite{
				configuration: configuration,
				binding:       item.Target(),
				value:         value,
			}
			writes = append(writes, executionAccessWriteValue2)
		}

		for _, item := range access.ValueStates() {
			state, ok := prepareValueState(model, values, item.Source())
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			value, ok := item.Prepare(state, true)
			if !ok {
				return "", CommandSelection{}, nil, errExecutionAccessPreparation
			}
			executionAccessWriteValue3 := executionAccessWrite{
				configuration: configuration,
				binding:       item.Target(),
				value:         value,
			}
			writes = append(writes, executionAccessWriteValue3)
			states[item.Source().uint32()] = state
		}

		if binding, ok := access.Locale(); ok {
			writes = appendExecutionFacilityWrite(writes, configuration, binding, Locale(locale))
		}
		if binding, ok := access.CommandSelection(); ok {
			writes = appendExecutionFacilityWrite(writes, configuration, binding, cloneCommandSelection(selection))
		}
		if binding, ok := access.WorkingDirectory(); ok {
			writes = appendExecutionFacilityWrite(writes, configuration, binding, workingDirectory)
		}
		if binding, ok := access.Stdin(); ok {
			writes = appendExecutionFacilityWrite(writes, configuration, binding, snapshot.stdin)
		}
		if binding, ok := access.Stdout(); ok {
			writes = appendExecutionFacilityWrite(writes, configuration, binding, snapshot.stdout)
		}
		if binding, ok := access.Stderr(); ok {
			writes = appendExecutionFacilityWrite(writes, configuration, binding, snapshot.stderr)
		}
	}

	for _, write := range writes {
		if !write.binding.Accepts(write.configuration, write.value) {
			return "", CommandSelection{}, nil, errExecutionAccessPreparation
		}
	}
	for _, write := range writes {
		if !write.binding.Write(write.configuration, write.value) {
			panic("cli: validated execution access write failed")
		}
	}
	return Locale(locale), selection, states, nil
}

func appendExecutionFacilityWrite(writes []executionAccessWrite, configuration any, binding configurationBinding, value any) []executionAccessWrite {
	executionAccessWriteValue := executionAccessWrite{
		configuration: configuration,
		binding:       binding,
		value:         value,
	}
	return append(writes, executionAccessWriteValue)
}
