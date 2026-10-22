package cli

import (
	"errors"
	"slices"
	"time"
)

type signalEndpointAcceptanceInput struct {
	key     any
	binding configurationBinding
	adapt   signalEndpointAdapter
}

type signalMatchAcceptanceInput struct {
	lineage         any
	kind            signalSelectorKind
	signals         []Signal
	excludedSignals []Signal
	origins         []SignalOrigin
	excludedOrigins []SignalOrigin
	excludedSender  authoringSignalTarget
	hasSender       bool
	phases          []SignalPhase
	conditions      []signalConditionClause
	minimum         int
	maximum         int
	aggregation     SignalAggregation
	deferred        bool
	continueMatch   bool
	hint            string
}

type signalActionAcceptanceInput struct {
	kind        signalActionKind
	name        string
	handler     signalHandlerCallback
	target      authoringSignalTarget
	signal      Signal
	signalValue destinationIdentity
	endpoint    *signalEndpointAcceptanceInput
	delayed     bool
	delay       time.Duration
	delayValue  destinationIdentity
	exitCode    int
}

type signalPlanAcceptanceInput struct {
	actions []signalActionAcceptanceInput
}

type signalPolicyAcceptanceInput struct {
	match *signalMatchAcceptanceInput
	plan  *signalPlanAcceptanceInput
}

func staticSignalPolicyAcceptanceInput(source *declarationDraft, value signalPolicyDraft) (signalPolicyAcceptanceInput, bool) {
	if source == nil || !validSignalPolicyDraft(source, value) {
		return signalPolicyAcceptanceInput{}, false
	}
	match := value.match
	acceptedMatch := &signalMatchAcceptanceInput{
		lineage:         match.lineage,
		kind:            match.kind,
		signals:         match.signals,
		excludedSignals: match.excludedSignals,
		origins:         match.origins,
		excludedOrigins: match.excludedOrigins,
		excludedSender:  match.excludedSender,
		hasSender:       match.hasSender,
		phases:          match.phases,
		conditions:      match.conditions,
		minimum:         match.minimum,
		maximum:         match.maximum,
		aggregation:     match.aggregation,
		deferred:        match.deferred,
		continueMatch:   match.continueMatch,
		hint:            match.hint,
	}
	actions := make([]signalActionAcceptanceInput, len(value.plan.actions))
	for index, action := range value.plan.actions {
		var signalValue destinationIdentity
		if action.signalValue != nil {
			signalValue = action.signalValue.currentIdentity()
		}
		var delayValue destinationIdentity
		if action.delayValue != nil {
			delayValue = action.delayValue.currentIdentity()
		}
		var endpoint *signalEndpointAcceptanceInput
		if action.endpoint.valid() {
			value, ok := source.signalEndpoints.resolve(source, action.endpoint)
			if !ok {
				panic("cli: validated signal endpoint cannot resolve")
			}
			endpoint = &signalEndpointAcceptanceInput{
				key:     action.endpoint,
				binding: value.binding,
				adapt:   value.adapt,
			}
		}
		actions[index] = signalActionAcceptanceInput{
			kind:        action.kind,
			name:        action.name,
			handler:     action.handler,
			target:      action.target,
			signal:      action.signal,
			signalValue: signalValue,
			endpoint:    endpoint,
			delayed:     action.delayed,
			delay:       action.delay,
			delayValue:  delayValue,
			exitCode:    action.exitCode,
		}
	}
	input := &signalPlanAcceptanceInput{
		actions: actions,
	}
	policyInput := signalPolicyAcceptanceInput{
		match: acceptedMatch,
		plan:  input,
	}
	return policyInput, true
}

func batchSignalPolicyAcceptanceInput(declarations *batchDeclarations, value batchSignalPolicy) signalPolicyAcceptanceInput {
	if declarations == nil || value.match == nil || value.plan == nil {
		panic("cli: validated provider signal policy is incomplete")
	}
	match := value.match
	acceptedMatch := &signalMatchAcceptanceInput{
		lineage:         match.lineage,
		kind:            match.kind,
		signals:         match.signals,
		excludedSignals: match.excludedSignals,
		origins:         match.origins,
		excludedOrigins: match.excludedOrigins,
		excludedSender:  match.excludedSender,
		hasSender:       match.hasSender,
		phases:          match.phases,
		conditions:      match.conditions,
		minimum:         match.minimum,
		maximum:         match.maximum,
		aggregation:     match.aggregation,
		deferred:        match.deferred,
		continueMatch:   match.continueMatch,
		hint:            match.hint,
	}
	actions := make([]signalActionAcceptanceInput, len(value.plan.actions))
	for index, action := range value.plan.actions {
		var endpoint *signalEndpointAcceptanceInput
		if action.endpoint.valid() {
			endpointIndex := action.endpoint.sliceIndex()
			if endpointIndex < 0 || endpointIndex >= len(declarations.signalEndpoints) {
				panic("cli: validated provider signal endpoint cannot resolve")
			}
			value := declarations.signalEndpoints[endpointIndex]
			endpoint = &signalEndpointAcceptanceInput{
				key:     action.endpoint,
				binding: value.binding,
				adapt:   value.adapt,
			}
		}
		actions[index] = signalActionAcceptanceInput{
			kind:        action.kind,
			name:        action.name,
			handler:     action.handler,
			target:      action.target,
			signal:      action.signal,
			signalValue: action.signalValue,
			endpoint:    endpoint,
			delayed:     action.delayed,
			delay:       action.delay,
			delayValue:  action.delayValue,
			exitCode:    action.exitCode,
		}
	}
	input := &signalPlanAcceptanceInput{
		actions: actions,
	}
	policyInput := signalPolicyAcceptanceInput{
		match: acceptedMatch,
		plan:  input,
	}
	return policyInput
}

var errInvalidSignalPolicyDeclaration = errors.New("cli: invalid signal policy declaration")

func prepareSignalPolicyDefinition(draft *declarationCandidate, policyValue signalPolicyAcceptanceInput, command acceptanceCommandRef, resolver destinationAcceptanceResolver, endpoints map[any]signalEndpointRef, position declarationPosition) (signalPolicyDefinition, error) {
	match := policyValue.match
	plan := policyValue.plan
	phases, ok := prepareSignalPhases(match.phases)
	if !ok {
		return signalPolicyDefinition{}, errInvalidSignalPolicyDeclaration
	}
	origins, ok := prepareSignalOrigins(match.origins)
	if !ok {
		return signalPolicyDefinition{}, errInvalidSignalPolicyDeclaration
	}
	excludedOrigins, ok := prepareSignalOrigins(match.excludedOrigins)
	if !ok {
		return signalPolicyDefinition{}, errInvalidSignalPolicyDeclaration
	}
	input := signalSelectorInput{
		Kind:            match.kind,
		Signals:         semanticSignalIDs(match.signals),
		ExcludedSignals: semanticSignalIDs(match.excludedSignals),
		Origins:         origins,
		ExcludedOrigins: excludedOrigins,
		ExcludedSender:  match.excludedSender.name,
		Phases:          phases,
	}
	selector := acceptSignalSelectorDefinition(input)
	if match.hasSender {
		selector = selector.WithExcludedSenderTarget(acceptSignalTargetDefinition(
			match.excludedSender.name, nil, match.excludedSender.matchesSender,
		))
	}
	if !selector.Valid() {
		return signalPolicyDefinition{}, errInvalidSignalPolicyDeclaration
	}

	activation, hasActivation, ok := prepareSignalActivation(match.conditions, resolver)
	if !ok {
		return signalPolicyDefinition{}, errInvalidSignalPolicyDeclaration
	}

	requiredSource := SignalSourceCapabilities(0)
	if len(origins) != 0 || len(excludedOrigins) != 0 {
		requiredSource |= SignalSourceCapabilityOrigin
	}
	if match.hasSender {
		requiredSource |= SignalSourceCapabilitySender
	}
	requiredHost := SignalHostCapabilities(0)
	actions := make([]signalActionDefinition, 0, len(plan.actions))
	for _, action := range plan.actions {
		definition, host, err := prepareSignalAction(draft, command, action, resolver, endpoints, position)
		if err != nil {
			return signalPolicyDefinition{}, err
		}
		if action.kind == signalActionKindReraise {
			requiredSource |= SignalSourceCapabilityReraise
		}
		requiredHost |= host
		actions = append(actions, definition)
	}

	candidate := signalPolicyCandidate{
		Match: semanticSignalPolicyMatch{
			Selector:         selector,
			Occurrence:       acceptSignalOccurrenceRange(match.minimum, match.maximum),
			Aggregation:      match.aggregation,
			Deferred:         match.deferred,
			ContinueMatching: match.continueMatch,
			Hint:             match.hint,
		},
		Actions: actions,
		Activation: signalPolicyActivation{
			Condition: activation,
			Present:   hasActivation,
		},
		Requirements: signalPolicyRequirements{
			Source: requiredSource,
			Host:   requiredHost,
		},
	}
	definition := acceptSignalPolicyDefinition(candidate)
	if !definition.Valid() {
		return signalPolicyDefinition{}, errInvalidSignalPolicyDeclaration
	}
	return definition, nil
}

func prepareSignalActivation(clauses []signalConditionClause, resolver destinationAcceptanceResolver) (condition, bool, bool) {
	if len(clauses) == 0 {
		return condition{}, false, true
	}
	children := make([]condition, 0, len(clauses))
	for _, clause := range clauses {
		acceptedCondition, ok := prepareCondition(clause.condition, resolver)
		if !ok {
			return condition{}, false, false
		}
		if clause.negated {
			acceptedCondition = acceptNegatedCondition(negatedConditionInput{
				Children: []condition{acceptedCondition},
			})
		}
		children = append(children, acceptedCondition)
	}
	if len(children) == 1 {
		return children[0], true, true
	}
	input := conditionGroupInput{
		Kind:     conditionKindAll,
		Children: children,
	}
	return acceptConditionGroup(input), true, true
}

func prepareSignalAction(draft *declarationCandidate, command acceptanceCommandRef, action signalActionAcceptanceInput, resolver destinationAcceptanceResolver, endpoints map[any]signalEndpointRef, position declarationPosition) (signalActionDefinition, SignalHostCapabilities, error) {
	if !action.kind.Valid() {
		return signalActionDefinition{}, 0, errInvalidSignalPolicyDeclaration
	}
	var signalValue ValueID
	if action.signalValue != nil {
		id, ok := resolver.ValueID(action.signalValue)
		if !ok {
			return signalActionDefinition{}, 0, errInvalidSignalPolicyDeclaration
		}
		signalValue = id
	}
	var delayValue ValueID
	if action.delayValue != nil {
		id, ok := resolver.ValueID(action.delayValue)
		if !ok {
			return signalActionDefinition{}, 0, errInvalidSignalPolicyDeclaration
		}
		delayValue = id
	}
	var endpoint SignalEndpointID
	if action.kind == signalActionKindEmit {
		if action.endpoint == nil || !action.endpoint.binding.Valid() {
			return signalActionDefinition{}, 0, errInvalidSignalPolicyDeclaration
		}
		ref, exists := endpoints[action.endpoint.key]
		if !exists {
			ref = draft.SignalEndpoint(command, position)
			draft.SignalEndpointBinding(ref, action.endpoint.binding, action.endpoint.adapt, position)
			endpoints[action.endpoint.key] = ref
		}
		endpoint = ref.ID()
	}

	var target signalTargetDefinition
	if action.kind == signalActionKindForward || action.kind == signalActionKindSend {
		target = acceptSignalTargetDefinition(
			action.target.name,
			action.target.deliver,
			action.target.matchesSender,
		)
	}
	var host SignalHostCapabilities
	switch action.kind {
	case signalActionKindAbort:
		host |= SignalHostCapabilityAbort
	case signalActionKindUnknown,
		signalActionKindReraise,
		signalActionKindIgnore,
		signalActionKindCancel,
		signalActionKindHandle,
		signalActionKindForward,
		signalActionKindSend,
		signalActionKindEmit:
	}
	candidate := signalActionCandidate{
		Kind:    action.kind,
		Name:    action.name,
		Handler: action.handler,
		Target:  target,
		Delivery: signalActionDelivery{
			Signal: action.signal,
			Value:  signalValue,
		},
		Endpoint: endpoint,
		Delay: signalActionDelay{
			Present:  action.delayed,
			Duration: action.delay,
			Value:    delayValue,
		},
		ExitCode:  action.exitCode,
		Sensitive: false,
	}
	definition := acceptSignalActionDefinition(candidate)
	if !definition.Valid() {
		return signalActionDefinition{}, 0, errInvalidSignalPolicyDeclaration
	}
	return definition, host, nil
}

func prepareSignalOrigins(values []SignalOrigin) ([]SignalOrigin, bool) {
	result := make([]SignalOrigin, len(values))
	for i, value := range values {
		switch value {
		case SignalOriginTerminal:
			result[i] = SignalOriginTerminal
		case SignalOriginProcess:
			result[i] = SignalOriginProcess
		case SignalOriginKernel:
			result[i] = SignalOriginKernel
		case SignalOriginChild:
			result[i] = SignalOriginChild
		case SignalOriginRuntime:
			result[i] = SignalOriginRuntime
		default:
			return nil, false
		}
	}
	return result, true
}

func prepareSignalPhases(values []SignalPhase) ([]SignalPhase, bool) {
	if len(values) == 0 {
		return []SignalPhase{SignalPhaseExecution}, true
	}
	result := make([]SignalPhase, len(values))
	for i, value := range values {
		switch value {
		case SignalPhaseResolution:
			result[i] = SignalPhaseResolution
		case SignalPhaseExecution:
			result[i] = SignalPhaseExecution
		case SignalPhaseCleanup:
			result[i] = SignalPhaseCleanup
		default:
			return nil, false
		}
	}
	return result, true
}

func semanticSignalIDs(values []Signal) []Signal {
	return slices.Clone(values)
}

func stageStaticSignals(draft *declarationCandidate, input declarationPlan, source *declarationDraft, state *staticAcceptanceState) error {
	if draft == nil {
		panic("cli: nil static signal acceptance candidate")
	}
	counters := make(map[any]signalCounterRef)
	endpoints := make(map[any]signalEndpointRef)

	for index := range len(input.records) {
		record := input.records[index]
		position := input.positionFor(index)
		command := acceptanceCommandRef{}
		if record.kind == declarationRecordSignalClear || record.kind == declarationRecordSignalPolicy {
			command = state.mustCommand(record.commandIndex)
		}
		profile, hasProfile := staticSignalProfile(record, state)
		switch record.kind {
		case declarationRecordSignalClear:
			draft.SignalPolicyScope(command, profile, true, position)
		case declarationRecordSignalPolicy:
			policyValue, valid := staticSignalPolicyAcceptanceInput(source, source.signalPolicies.entries[record.familyIndex].value)
			if !valid {
				return errStaticDeclarationRejected
			}
			if policyValue.match == nil || policyValue.plan == nil || policyValue.match.lineage == nil || len(policyValue.plan.actions) == 0 {
				panic("cli: accepted static signal policy is incomplete")
			}
			policy := draft.SignalPolicy(command, position)
			if hasProfile {
				draft.SignalPolicyProfile(policy, profile, position)
			}
			counter, exists := counters[policyValue.match.lineage]
			if !exists {
				counter = draft.SignalCounter(policy, position)
				counters[policyValue.match.lineage] = counter
			} else {
				draft.SignalPolicyCounter(policy, counter, position)
			}

			definition, err := prepareSignalPolicyDefinition(draft, policyValue, command, state, endpoints, position)
			if err != nil {
				return err
			}
			draft.SignalPolicyDefinition(policy, definition, position)
		case declarationRecordInvalid,
			declarationRecordDestination,
			declarationRecordValueDirective,
			declarationRecordDialectDirective,
			declarationRecordFlag,
			declarationRecordFlagSpelling,
			declarationRecordFlagSpellingDirective,
			declarationRecordArgumentForm,
			declarationRecordArgumentTerm,
			declarationRecordArgumentFormSelection,
			declarationRecordCommand,
			declarationRecordConditionPolicy,
			declarationRecordLifecycleDirective,
			declarationRecordGrammarProfileSwitch,
			declarationRecordGrammarProfileCase,
			declarationRecordRule,
			declarationRecordRuleDirective,
			declarationRecordAdvisory,
			declarationRecordAdvisoryDirective,
			declarationRecordNoticeSuppression,
			declarationRecordShortcut,
			declarationRecordShortcutDirective,
			declarationRecordProvider,
			declarationRecordHelpGroup,
			declarationRecordReplacement,
			declarationRecordAccess:
		}
	}
	return nil
}

func staticSignalProfile(record declarationPlanRecord, state *staticAcceptanceState) (grammarProfileRef, bool) {
	if record.profileIndex < 0 {
		return grammarProfileRef{}, false
	}
	return state.mustProfile(record.profileIndex), true
}

func (draft *declarationCandidate) validateSignalReferences(index referenceValidationIndex, rejections []rejection) []rejection {
	commands := index.commands
	destinations := index.destinations
	flags := index.flags
	commandEntries := index.commandEntries
	profiles := index.profiles
	endpoints := index.endpoints
	counters := index.counters
	for _, record := range draft.input.SignalEndpoints {
		if _, ok := commands[record.command]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}
	for _, record := range draft.input.SignalPolicies {
		if _, ok := commands[record.command]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		if record.meta.profile.present() {
			profile, ok := profiles[record.meta.profile]
			if !ok || profile.command != record.command {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		definition, defined := record.Definition()
		if !defined {
			continue
		}
		counter, hasCounter := record.Counter()
		if !hasCounter {
			rejections = append(rejections, semanticRejection(record.meta.position))
		} else if _, ok := counters[counter]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		if activation, ok := definition.Activation(); ok {
			for _, dependency := range activation.Dependencies() {
				destination, exists := destinations[dependency]
				if !exists || !commandSameOrAncestor(commands, destination.command, record.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
		for _, action := range definition.Actions() {
			if ValueID, ok := action.SignalValue(); ok {
				destination, exists := destinations[ValueID]
				if !exists || !commandSameOrAncestor(commands, destination.command, record.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
			if ValueID, ok := action.DelayValue(); ok {
				destination, exists := destinations[ValueID]
				if !exists || !commandSameOrAncestor(commands, destination.command, record.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
			if endpointID, ok := action.Endpoint(); ok {
				endpoint, exists := endpoints[endpointID]
				if !exists || endpoint.command != record.command {
					rejections = append(rejections, semanticRejection(record.meta.position))
				} else if _, bound := endpoint.Binding(); !bound {
					rejections = append(rejections, semanticRejection(endpoint.meta.position))
				}
			}
		}
	}
	for _, scope := range draft.input.SignalPolicyScopes {
		if _, ok := commands[scope.Command()]; !ok {
			rejections = append(rejections, semanticRejection(scope.Position()))
			continue
		}
		if scope.Profile().present() {
			profile, ok := profiles[scope.Profile()]
			if !ok || profile.command != scope.Command() {
				rejections = append(rejections, semanticRejection(scope.Position()))
			}
		}
	}
	for _, group := range draft.input.HelpGroups {
		if !group.Valid() {
			rejections = append(rejections, semanticRejection(group.position))
			continue
		}
		if _, ok := commands[group.command]; !ok {
			rejections = append(rejections, semanticRejection(group.position))
		}
		if group.profile.present() {
			profile, ok := profiles[group.profile]
			if !ok || profile.command != group.command {
				rejections = append(rejections, semanticRejection(group.position))
			}
		}
		switch group.family {
		case helpGroupFamilyCommand:
			for _, entry := range group.CommandEntries() {
				switch entry.Kind() {
				case HelpCommandGroupEntryCommandEntry:
					record, ok := commandEntries[entry.CommandEntry()]
					if !ok || record.parent != group.command {
						rejections = append(rejections, semanticRejection(group.position))
					}
				case HelpCommandGroupEntryCommandProvider:
					found := false
					for _, provider := range draft.input.Providers {
						if provider.meta.id == entry.Provider() && provider.parent == group.command {
							found = true
							break
						}
					}
					if !found {
						rejections = append(rejections, semanticRejection(group.position))
					}
				default:
					rejections = append(rejections, semanticRejection(group.position))
				}
			}
		case helpGroupFamilyFlag:
			for _, flag := range group.Flags() {
				record, ok := flags[flag]
				if !ok || record.command != group.command {
					rejections = append(rejections, semanticRejection(group.position))
				}
			}
		case helpGroupFamilyUnknown:
		}
	}
	return rejections
}
