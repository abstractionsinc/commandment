package cli

import "slices"

func freezeProviderDestinationIdentity(draft *providerBatchDraft, identity destinationIdentity) destinationIdentity {
	if draft == nil {
		panic("cli: nil provider batch draft")
	}
	switch identity := identity.(type) {
	case nil:
		return nil
	case stagedDestinationIdentity:
		if _, ok := draft.declarations.destinations.resolve(&draft.declarations, identity.ref); !ok {
			panic("cli: provider destination identity cannot freeze")
		}
		return localDestinationTarget{destination: batchRefAt[batchDestination](identity.ref.index)}
	case acceptedDestinationIdentity:
		return identity
	case localDestinationTarget:
		return identity
	default:
		panic("cli: unsupported provider destination identity")
	}
}

func freezeProviderDestinationTarget(draft *providerBatchDraft, identity destinationIdentity) batchDestinationTarget {
	frozen := freezeProviderDestinationIdentity(draft, identity)
	target, ok := frozen.(batchDestinationTarget)
	if !ok {
		panic("cli: provider destination target cannot freeze")
	}
	return target
}

func freezeProviderDependencies(draft *providerBatchDraft, input []authoringDestinationDependency) []batchDestinationDependency {
	result := make([]batchDestinationDependency, len(input))
	for index, dependency := range input {
		result[index] = batchDestinationDependency{
			destination: freezeProviderDestinationTarget(draft, dependency.destination),
			optional:    dependency.optional,
		}
	}
	return result
}

func freezeProviderComputation(draft *providerBatchDraft, input authoringValueComputation) batchValueComputation {
	frozen, ok := freezeAuthoringValueComputation(&draft.declarations, input)
	if !ok {
		panic("cli: invalid provider value computation reached freeze")
	}
	computation := batchValueComputation{
		dependencies: freezeProviderDependencies(draft, frozen.dependencies),
		source:       snapshotProducerSource(frozen.source.Annotations(), frozen.source.Sensitive()),
	}

	return computation
}

func freezeProviderCompletion(draft *providerBatchDraft, input authoringCompletion) batchCompletion {
	frozen, ok := freezeAuthoringCompletion(&draft.declarations, input)
	if !ok {
		panic("cli: invalid provider completion reached freeze")
	}
	source, ok := producerSource(frozen.options)
	if !ok {
		panic("cli: invalid provider completion source reached freeze")
	}
	completion := batchCompletion{
		kind:         frozen.kind,
		dependencies: freezeProviderDependencies(draft, frozen.dependencies),
		source:       source,
		nativeKind:   frozen.nativeKind,
		nativeMode:   frozen.nativeMode,
		hasNative:    frozen.hasNative,
	}

	return completion
}

func freezeProviderCondition(draft *providerBatchDraft, input Condition) Condition {
	if input.state == nil {
		return Condition{}
	}
	return Condition{state: freezeProviderConditionState(draft, input.state)}
}

func freezeProviderConditionState(draft *providerBatchDraft, input *conditionState) *conditionState {
	if input == nil {
		return nil
	}
	state := *input
	state.owner = nil
	state.destination = freezeProviderDestinationIdentity(draft, input.destination)
	state.operands = append([]any(nil), input.operands...)
	state.children = make([]*conditionState, len(input.children))
	for index, child := range input.children {
		state.children[index] = freezeProviderConditionState(draft, child)
	}
	return &state
}

func freezeProviderRuleCondition(draft *providerBatchDraft, input RuleCondition) RuleCondition {
	switch input := input.(type) {
	case nil:
		return nil
	case Condition:
		return freezeProviderCondition(draft, input)
	case validationRuleCondition:
		return validationRuleCondition{state: freezeProviderRuleConditionState(draft, input.state)}
	default:
		return input
	}
}

func freezeProviderRuleConditionState(draft *providerBatchDraft, input *ruleConditionState) *ruleConditionState {
	if input == nil {
		return nil
	}
	state := *input
	state.owner = nil
	state.structural = freezeProviderConditionState(draft, input.structural)
	state.destination = freezeProviderDestinationIdentity(draft, input.destination)
	state.producerKinds = slices.Clone(input.producerKinds)
	state.children = make([]*ruleConditionState, len(input.children))
	for index, child := range input.children {
		state.children[index] = freezeProviderRuleConditionState(draft, child)
	}
	return &state
}

type providerValueDomainFreeze struct {
	bySource map[*valueDomainState]batchValueDomainRef
	sources  []*valueDomainState
	frozen   []batchValueDomain
}

func freezeProviderValueDomain(draft *providerBatchDraft, input declarationPlan, state *valueDomainState, domains *providerValueDomainFreeze) batchValueDomainRef {
	if state == nil {
		panic("cli: nil provider value domain reached freeze")
	}
	if domains == nil || domains.bySource == nil {
		panic("cli: invalid provider value-domain freeze")
	}
	if ref, ok := domains.bySource[state]; ok {
		return ref
	}
	domain := batchValueDomain{
		entries: make([]domainEntrySemantics, len(state.entries)),
	}
	for index, entry := range state.entries {
		var ok bool
		domain.entries[index], ok = acceptDomainEntrySemantics(entry)
		if !ok {
			panic("cli: invalid provider value-domain entry reached freeze")
		}
	}
	if state.dynamic != nil {
		owner := state.dynamic.ownerCommandIndex
		if state.dynamic.ownerCommand.valid() {
			var ok bool
			owner, ok = input.indexes.commands[state.dynamic.ownerCommand]
			if !ok {
				panic("cli: provider dynamic-domain owner cannot freeze")
			}
		}
		domain.dynamic = &batchDynamicValueDomain{
			commandIndex: owner,
			dependencies: freezeProviderDependencies(draft, state.dynamic.dependencies),
		}
	}
	ref := batchRefAt[batchValueDomain](len(domains.frozen))
	domains.bySource[state] = ref
	domains.sources = append(domains.sources, state)
	domains.frozen = append(domains.frozen, domain)
	return ref
}

func freezeProviderDomainUses(draft *providerBatchDraft, input declarationPlan, uses []authoringDomainUse, domains *providerValueDomainFreeze) []batchDomainUse {
	result := make([]batchDomainUse, len(uses))
	for index, use := range uses {
		result[index] = batchDomainUse{
			domain: freezeProviderValueDomain(draft, input, use.state, domains),
			kind:   use.kind,
		}
	}
	return result
}

func freezeProviderCommand(draft *providerBatchDraft, value commandDraft) batchCommand {
	defaults := value.defaults
	if defaults.state == commandDefaultConditional {
		defaults.condition = freezeProviderCondition(draft, defaults.condition)
	}
	command := batchCommand{
		name:        value.name,
		aliases:     slices.Clone(value.aliases),
		description: value.description,
		examples:    slices.Clone(value.examples),
		hidden:      value.hidden,
		defaults:    defaults,
		release:     value.release,
		makeConfig:  value.makeConfig,
		middleware:  slices.Clone(value.middleware),
		handler:     value.handler,
	}

	return command
}

func freezeProviderDestination(draft *providerBatchDraft, input declarationPlan, value destinationDraft, domains *providerValueDomainFreeze) batchDestination {
	accepted, ok := staticDestinationAcceptanceInput(value)
	if !ok {
		panic("cli: validated provider destination cannot freeze")
	}
	definition, ok := prepareDestinationDefinitionForRelease(accepted, draft.release)
	if !ok {
		panic("cli: validated provider destination cannot freeze")
	}
	destination := batchDestination{
		sensitive:   value.sensitive,
		computation: freezeProviderComputation(draft, value.computation),
		binding:     value.binding,
		domainUses:  freezeProviderDomainUses(draft, input, value.domainUses, domains),
		definition:  definition,
	}

	return destination
}

func freezeProviderValueDirective(draft *providerBatchDraft, value valueDirectiveDraft) batchValueDirective {
	directive := batchValueDirective{
		kind:             value.kind,
		destination:      freezeProviderDestinationIdentity(draft, value.destination),
		value:            snapshotMutableValue(value.value),
		computation:      freezeProviderComputation(draft, value.computation),
		policies:         slices.Clone(value.policies),
		name:             value.name,
		sensitive:        value.sensitive,
		environmentShape: value.environmentShape,
		delimiter:        value.delimiter,
	}

	return directive
}

func freezeProviderDialectDirective(value dialectDirectiveDraft) dialectDirectiveDefinition {
	directive := dialectDirectiveDefinition{
		kind:          value.kind,
		flagOrder:     value.flagOrder,
		clustering:    value.clustering,
		matching:      value.matching,
		normalization: value.normalization,
		dashless:      value.dashless,
		tokens:        slices.Clone(value.tokens),
		enabled:       value.enabled,
		placement:     value.placement,
	}

	return directive
}

func freezeProviderFlag(draft *providerBatchDraft, value flagDraft) batchFlag {
	var completion batchCompletion
	if value.completionSet {
		completion = freezeProviderCompletion(draft, value.completion)
	}
	flag := batchFlag{
		destination:         batchRefAt[batchDestination](value.destination.index),
		producer:            value.producer,
		completion:          completion,
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

	return flag
}

func freezeProviderFlagSpelling(value flagSpellingDraft) batchFlagSpelling {
	return batchFlagSpelling{
		flag: batchRefAt[batchFlag](value.flag.index),
		kind: value.kind,
		text: value.text,
	}
}

func freezeProviderFlagSpellingDirective(value flagSpellingDirectiveDraft) batchFlagSpellingDirective {
	directive := batchFlagSpellingDirective{
		spelling:      batchRefAt[batchFlagSpelling](value.spelling.index),
		kind:          value.kind,
		forms:         slices.Clone(value.forms),
		value:         snapshotMutableValue(value.value),
		sourceOptions: slices.Clone(value.sourceOptions),
		lifecycle:     value.lifecycle,
	}

	return directive
}

func freezeProviderArgumentTerm(draft *providerBatchDraft, value argumentTermDraft) batchArgumentTerm {
	destination := batchDestinationRef{}
	if value.destination.valid() {
		destination = batchRefAt[batchDestination](value.destination.index)
	}
	var completion batchCompletion
	if value.completionSet {
		completion = freezeProviderCompletion(draft, value.completion)
	}
	term := batchArgumentTerm{
		form:               batchRefAt[batchArgumentForm](value.form.index),
		destination:        destination,
		kind:               value.kind,
		name:               value.name,
		count:              value.count,
		tokenLanguage:      value.tokenLanguage,
		literal:            value.literal,
		completion:         completion,
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

	return term
}

func freezeProviderConditionTarget(target authoringConditionTarget) batchConditionTarget {
	switch target := target.(type) {
	case authoringCommandConditionTarget:
		return batchConditionTarget{
			kind:    batchConditionTargetCommand,
			command: batchRefAt[batchCommand](target.reference.index),
		}
	case authoringFlagConditionTarget:
		return batchConditionTarget{
			kind: batchConditionTargetFlag,
			flag: batchRefAt[batchFlag](target.reference.index),
		}
	case authoringArgumentFormConditionTarget:
		return batchConditionTarget{
			kind:         batchConditionTargetArgumentForm,
			argumentForm: batchRefAt[batchArgumentForm](target.reference.index),
		}
	case authoringArgumentTermConditionTarget:
		return batchConditionTarget{
			kind:         batchConditionTargetArgumentTerm,
			argumentTerm: batchRefAt[batchArgumentTerm](target.reference.index),
		}
	default:
		return batchConditionTarget{}
	}
}

func freezeProviderLifecycleTarget(target authoringLifecycleTarget) batchLifecycleTarget {
	switch target := target.(type) {
	case authoringEnvironmentLifecycleTarget:
		return batchLifecycleTarget{
			kind:        batchLifecycleTargetEnvironment,
			environment: batchRefAt[batchValueDirective](target.reference.index),
		}
	case authoringArgumentFormLifecycleTarget:
		return batchLifecycleTarget{
			kind:         batchLifecycleTargetArgumentForm,
			argumentForm: batchRefAt[batchArgumentForm](target.reference.index),
		}
	case authoringArgumentTermLifecycleTarget:
		return batchLifecycleTarget{
			kind:         batchLifecycleTargetArgumentTerm,
			argumentTerm: batchRefAt[batchArgumentTerm](target.reference.index),
		}
	default:
		return batchLifecycleTarget{}
	}
}

func freezeProviderRule(draft *providerBatchDraft, value ruleDraft) batchRule {
	destinations := make([]destinationIdentity, len(value.destinations))
	for index, destination := range value.destinations {
		destinations[index] = freezeProviderDestinationIdentity(draft, destination)
	}
	rule := batchRule{
		kind:             value.kind,
		code:             value.code,
		activation:       freezeProviderRuleCondition(draft, value.activation),
		destinations:     destinations,
		operands:         append([]any(nil), value.operands...),
		count:            value.count,
		pattern:          value.pattern,
		callbackKind:     value.callbackKind,
		callback:         value.callback,
		builtinPredicate: value.builtinPredicate,
	}

	return rule
}

func freezeProviderShortcut(value shortcutDraft) batchShortcut {
	shortcut := batchShortcut{
		name:        value.name,
		description: value.description,
		examples:    slices.Clone(value.examples),
		args:        slices.Clone(value.args),
		appendArgs:  slices.Clone(value.appendArgs),
		hidden:      value.hidden,
		release:     value.release,
	}

	return shortcut
}

func freezeProviderReplacementTarget(target authoringReplacementTarget, indexes declarationPlanIndex) batchReplacementTarget {
	index, ok := declarationReplacementTargetIndex(target, indexes)
	if !ok {
		return batchReplacementTarget{
			kind:  target.kind,
			index: -1,
		}
	}
	return batchReplacementTarget{
		kind:  target.kind,
		index: index,
	}
}

func freezeProviderSignalDestination(draft *providerBatchDraft, handle *destinationHandle) destinationIdentity {
	if handle == nil {
		return nil
	}
	return freezeProviderDestinationIdentity(draft, handle.currentIdentity())
}

func freezeProviderSignalPolicy(draft *providerBatchDraft, value signalPolicyDraft, lineages map[*signalCounterLineage]*batchSignalCounterLineage) batchSignalPolicy {
	if !validSignalPolicyDraft(&draft.declarations, value) {
		panic("cli: invalid provider signal policy reached freeze")
	}

	match := value.match
	lineage := lineages[match.lineage]
	if lineage == nil {
		lineage = &batchSignalCounterLineage{}
		lineages[match.lineage] = lineage
	}
	conditions := make([]signalConditionClause, len(match.conditions))
	for index, clause := range match.conditions {
		conditions[index] = signalConditionClause{
			condition: freezeProviderCondition(draft, clause.condition),
			negated:   clause.negated,
		}
	}
	frozenMatch := &batchSignalMatch{
		lineage:         lineage,
		kind:            match.kind,
		signals:         slices.Clone(match.signals),
		excludedSignals: slices.Clone(match.excludedSignals),
		origins:         slices.Clone(match.origins),
		excludedOrigins: slices.Clone(match.excludedOrigins),
		excludedSender:  match.excludedSender,
		hasSender:       match.hasSender,
		phases:          slices.Clone(match.phases),
		conditions:      conditions,
		minimum:         match.minimum,
		maximum:         match.maximum,
		aggregation:     match.aggregation,
		deferred:        match.deferred,
		continueMatch:   match.continueMatch,
		hint:            match.hint,
	}

	plan := value.plan
	actions := make([]batchSignalAction, len(plan.actions))
	for index, action := range plan.actions {
		var endpoint batchSignalEndpointRef
		if action.endpoint.valid() {
			if _, ok := draft.declarations.signalEndpoints.resolve(&draft.declarations, action.endpoint); !ok {
				panic("cli: provider signal endpoint cannot freeze")
			}
			endpoint = batchRefAt[batchSignalEndpoint](action.endpoint.index)
		}
		actions[index] = batchSignalAction{
			kind:        action.kind,
			name:        action.name,
			handler:     action.handler,
			target:      action.target,
			signal:      action.signal,
			signalValue: freezeProviderSignalDestination(draft, action.signalValue),
			endpoint:    endpoint,
			delayed:     action.delayed,
			delay:       action.delay,
			delayValue:  freezeProviderSignalDestination(draft, action.delayValue),
			exitCode:    action.exitCode,
		}
	}
	frozenPlan := &batchSignalActionPlan{
		actions: actions,
	}

	policy := batchSignalPolicy{
		match: frozenMatch,
		plan:  frozenPlan,
	}

	return policy
}

func (draft *providerBatchDraft) freeze() (providerBatch, error) {
	var batch providerBatch
	if draft == nil {
		panic("cli: nil provider batch draft")
	}
	if err := draft.declarations.rejectionError(); err != nil {
		return batch, rejectProviderBatch(err)
	}
	input, err := draft.declarations.declarationPlan()
	if err != nil {
		return batch, rejectProviderBatch(err)
	}
	if err := draft.validate(input); err != nil {
		return batch, rejectProviderBatch(err)
	}
	declarations, bindings, domainSources := draft.freezeDeclarations(input)
	callbacks := draft.freezeCallbacks(input, domainSources)
	contributions := draft.freezeContributions()
	var localization *localizationSnapshot
	if draft.localization != nil {
		snapshot := cloneLocalizationSnapshot(*draft.localization)
		localization = &snapshot
	}
	batch = providerBatch{
		provider:      draft.provider,
		parent:        draft.parent,
		scope:         draft.scope,
		declarations:  declarations,
		contributions: contributions,
		callbacks:     callbacks,
		bindings:      bindings,
		localization:  localization,
	}
	return batch, nil
}

func (draft *providerBatchDraft) freezeDeclarations(input declarationPlan) (batchDeclarations, []batchDestinationBinding, []*valueDomainState) {
	if draft == nil {
		panic("cli: nil provider batch draft")
	}

	domains := providerValueDomainFreeze{
		bySource: make(map[*valueDomainState]batchValueDomainRef),
	}
	signalLineages := make(map[*signalCounterLineage]*batchSignalCounterLineage)
	declarations := batchDeclarations{
		order:                  make([]batchDeclarationOrderEntry, 0, len(input.records)),
		commands:               make([]batchCommand, len(draft.declarations.commands.entries)),
		destinations:           make([]batchDestination, len(draft.declarations.destinations.entries)),
		flags:                  make([]batchFlag, len(draft.declarations.flags.entries)),
		flagSpellings:          make([]batchFlagSpelling, len(draft.declarations.flagSpellings.entries)),
		argumentTerms:          make([]batchArgumentTerm, len(draft.declarations.argumentTerms.entries)),
		providers:              make([]batchProvider, len(draft.declarations.providers.entries)),
		valueDirectives:        make([]batchValueDirective, len(draft.declarations.valueDirectives.entries)),
		dialectDirectives:      make([]dialectDirectiveDefinition, len(draft.declarations.dialectDirectives.entries)),
		flagSpellingDirectives: make([]batchFlagSpellingDirective, len(draft.declarations.flagSpellingDirectives.entries)),
		argumentForms:          make([]batchArgumentForm, len(draft.declarations.argumentForms.entries)),
		argumentFormSelections: make([]batchArgumentFormSelection, len(draft.declarations.argumentFormSelections.entries)),
		conditionPolicies:      make([]batchConditionPolicy, len(draft.declarations.conditionPolicies.entries)),
		lifecycleDirectives:    make([]batchLifecycleDirective, len(draft.declarations.lifecycleDirectives.entries)),
		grammarProfileSwitches: make([]batchGrammarProfileSwitch, len(draft.declarations.grammarProfileSwitches.entries)),
		grammarProfileCases:    make([]batchGrammarProfileCase, len(draft.declarations.grammarProfileCases.entries)),
		rules:                  make([]batchRule, len(draft.declarations.rules.entries)),
		ruleDirectives:         make([]batchRuleDirective, len(draft.declarations.ruleDirectives.entries)),
		advisories:             make([]batchAdvisory, len(draft.declarations.advisories.entries)),
		advisoryDirectives:     make([]batchAdvisoryDirective, len(draft.declarations.advisoryDirectives.entries)),
		noticeSuppressions:     make([]batchNoticeSuppression, len(draft.declarations.noticeSuppressions.entries)),
		shortcuts:              make([]batchShortcut, len(draft.declarations.shortcuts.entries)),
		shortcutDirectives:     make([]batchShortcutDirective, len(draft.declarations.shortcutDirectives.entries)),
		signalEndpoints:        make([]batchSignalEndpoint, len(draft.declarations.signalEndpoints.entries)),
		signalClears:           make([]batchSignalClear, len(draft.declarations.signalClears.entries)),
		signalPolicies:         make([]batchSignalPolicy, len(draft.declarations.signalPolicies.entries)),
		helpGroups:             make([]batchHelpGroup, len(draft.declarations.helpGroups.entries)),
		replacements:           make([]batchReplacement, len(draft.declarations.replacements.entries)),
		accesses:               make([]accessDefinition, len(draft.declarations.accesses.entries)),
	}

	destinationForOrder := make([]batchDestinationRef, len(input.records))
	for index, entry := range draft.declarations.signalEndpoints.entries {
		value := entry.value
		if !value.valid || !value.binding.Valid() || value.adapt == nil {
			panic("cli: invalid provider signal endpoint reached freeze")
		}
		declarations.signalEndpoints[index] = batchSignalEndpoint{
			binding: value.binding,
			adapt:   value.adapt,
		}
	}

	bindings := make([]batchDestinationBinding, 0, len(draft.declarations.destinations.entries))
	for orderIndex, ref := range input.destinationRefs {
		if ref.index < 0 || ref.index >= len(declarations.destinations) {
			panic("cli: provider destination has invalid typed coordinate")
		}
		destinationForOrder[orderIndex] = batchRefAt[batchDestination](ref.index)
	}
	for index, entry := range draft.declarations.destinations.entries {
		declarations.destinations[index] = freezeProviderDestination(draft, input, entry.value, &domains)
		if entry.value.handle == nil {
			panic("cli: provider destination missing handle")
		}
		bindings = append(bindings, batchDestinationBinding{
			handle:      entry.value.handle,
			destination: batchRefAt[batchDestination](index),
		})
	}

	for orderIndex, record := range input.records {
		switch record.kind {
		case declarationRecordCommand:
			declarations.commands[record.familyIndex] = freezeProviderCommand(draft, draft.declarations.commands.entries[record.familyIndex].value)
		case declarationRecordDestination:
			// Destinations were frozen above so their handles and bindings stay aligned.
		case declarationRecordValueDirective:
			value := draft.declarations.valueDirectives.entries[record.familyIndex].value
			declarations.valueDirectives[record.familyIndex] = freezeProviderValueDirective(draft, value)
		case declarationRecordDialectDirective:
			value := draft.declarations.dialectDirectives.entries[record.familyIndex].value
			declarations.dialectDirectives[record.familyIndex] = freezeProviderDialectDirective(value)
		case declarationRecordFlag:
			declarations.flags[record.familyIndex] = freezeProviderFlag(draft, draft.declarations.flags.entries[record.familyIndex].value)
		case declarationRecordFlagSpelling:
			declarations.flagSpellings[record.familyIndex] = freezeProviderFlagSpelling(draft.declarations.flagSpellings.entries[record.familyIndex].value)
		case declarationRecordFlagSpellingDirective:
			value := draft.declarations.flagSpellingDirectives.entries[record.familyIndex].value
			declarations.flagSpellingDirectives[record.familyIndex] = freezeProviderFlagSpellingDirective(value)
		case declarationRecordArgumentForm:
			declarations.argumentForms[record.familyIndex] = batchArgumentForm{}
		case declarationRecordArgumentTerm:
			declarations.argumentTerms[record.familyIndex] = freezeProviderArgumentTerm(draft, draft.declarations.argumentTerms.entries[record.familyIndex].value)
		case declarationRecordArgumentFormSelection:
			value := draft.declarations.argumentFormSelections.entries[record.familyIndex].value
			declarations.argumentFormSelections[record.familyIndex] = batchArgumentFormSelection{
				form:        batchRefAt[batchArgumentForm](value.form.index),
				destination: freezeProviderDestinationIdentity(draft, value.destination),
				value:       snapshotMutableValue(value.value),
			}
		case declarationRecordConditionPolicy:
			value := draft.declarations.conditionPolicies.entries[record.familyIndex].value
			declarations.conditionPolicies[record.familyIndex] = batchConditionPolicy{
				target:    freezeProviderConditionTarget(value.target),
				effect:    value.effect,
				condition: freezeProviderCondition(draft, value.condition),
			}
		case declarationRecordLifecycleDirective:
			value := draft.declarations.lifecycleDirectives.entries[record.familyIndex].value
			declarations.lifecycleDirectives[record.familyIndex] = batchLifecycleDirective{
				target:    freezeProviderLifecycleTarget(value.target),
				lifecycle: value.lifecycle,
			}
		case declarationRecordGrammarProfileSwitch:
			value := draft.declarations.grammarProfileSwitches.entries[record.familyIndex].value
			declarations.grammarProfileSwitches[record.familyIndex] = batchGrammarProfileSwitch{
				selector: freezeProviderDestinationIdentity(draft, value.selector),
				equal:    value.equal,
			}
		case declarationRecordGrammarProfileCase:
			value := draft.declarations.grammarProfileCases.entries[record.familyIndex].value
			declarations.grammarProfileCases[record.familyIndex] = batchGrammarProfileCase{
				switchRef:   batchRefAt[batchGrammarProfileSwitch](value.switchRef.index),
				value:       snapshotMutableValue(value.value),
				defaultCase: value.defaultCase,
			}
		case declarationRecordRule:
			declarations.rules[record.familyIndex] = freezeProviderRule(draft, draft.declarations.rules.entries[record.familyIndex].value)
		case declarationRecordRuleDirective:
			value := draft.declarations.ruleDirectives.entries[record.familyIndex].value
			declarations.ruleDirectives[record.familyIndex] = batchRuleDirective{
				rule:        batchRefAt[batchRule](value.rule.index),
				kind:        value.kind,
				destination: freezeProviderDestinationIdentity(draft, value.destination),
				optional:    value.optional,
			}
		case declarationRecordAdvisory:
			value := draft.declarations.advisories.entries[record.familyIndex].value
			declarations.advisories[record.familyIndex] = batchAdvisory{
				kind:      value.kind,
				advisory:  value.advisory,
				condition: freezeProviderCondition(draft, value.condition),
				callback:  value.callback,
			}
		case declarationRecordAdvisoryDirective:
			value := draft.declarations.advisoryDirectives.entries[record.familyIndex].value
			declarations.advisoryDirectives[record.familyIndex] = batchAdvisoryDirective{
				advisory:    batchRefAt[batchAdvisory](value.advisory.index),
				kind:        value.kind,
				destination: freezeProviderDestinationIdentity(draft, value.destination),
				optional:    value.optional,
			}
		case declarationRecordNoticeSuppression:
			value := draft.declarations.noticeSuppressions.entries[record.familyIndex].value
			declarations.noticeSuppressions[record.familyIndex] = batchNoticeSuppression{
				condition: freezeProviderCondition(draft, value.condition),
				match:     value.match,
			}
		case declarationRecordShortcut:
			declarations.shortcuts[record.familyIndex] = freezeProviderShortcut(draft.declarations.shortcuts.entries[record.familyIndex].value)
		case declarationRecordShortcutDirective:
			value := draft.declarations.shortcutDirectives.entries[record.familyIndex].value
			var replacement batchShortcutRef
			if value.replacement.valid() {
				replacement = batchRefAt[batchShortcut](value.replacement.index)
			}
			declarations.shortcutDirectives[record.familyIndex] = batchShortcutDirective{
				shortcut:      batchRefAt[batchShortcut](value.shortcut.index),
				kind:          value.kind,
				destination:   freezeProviderDestinationIdentity(draft, value.destination),
				value:         snapshotMutableValue(value.value),
				sourceOptions: slices.Clone(value.sourceOptions),
				replacement:   replacement,
			}
		case declarationRecordProvider:
			value := draft.declarations.providers.entries[record.familyIndex].value
			declarations.providers[record.familyIndex] = batchProvider{
				dependencies: freezeProviderDependencies(draft, value.dependencies),
			}
		case declarationRecordSignalClear:
			declarations.signalClears[record.familyIndex] = batchSignalClear{}
		case declarationRecordSignalPolicy:
			value := draft.declarations.signalPolicies.entries[record.familyIndex].value
			declarations.signalPolicies[record.familyIndex] = freezeProviderSignalPolicy(draft, value, signalLineages)
		case declarationRecordHelpGroup:
			value := draft.declarations.helpGroups.entries[record.familyIndex].value
			declarations.helpGroups[record.familyIndex] = batchHelpGroup{
				family: value.family,
				title:  value.title,
				order:  value.order,
			}
		case declarationRecordReplacement:
			value := draft.declarations.replacements.entries[record.familyIndex].value
			source := freezeProviderReplacementTarget(value.source, input.indexes)
			target := freezeProviderReplacementTarget(value.target, input.indexes)
			if source.index < 0 || target.index < 0 {
				panic("cli: validated provider replacement cannot freeze")
			}
			declarations.replacements[record.familyIndex] = batchReplacement{
				source: source,
				target: target,
			}
		case declarationRecordAccess:
			value := draft.declarations.accesses.entries[record.familyIndex].value
			declarations.accesses[record.familyIndex] = accessDefinition{
				kind:    value.kind,
				source:  freezeProviderDestinationIdentity(draft, value.source),
				target:  value.target,
				prepare: value.prepare,
			}
		default:
			panic("cli: invalid provider declaration record")
		}
		declarations.appendRecord(record, destinationForOrder[orderIndex])
	}
	declarations.valueDomains = domains.frozen
	return declarations, bindings, domains.sources
}

func (draft *providerBatchDraft) freezeContributions() []batchContribution {
	result := make([]batchContribution, len(draft.contributions))
	for index, contribution := range draft.contributions {
		switch contribution := contribution.(type) {
		case commandContributionDraft:
			command := draft.freezeCommandRef(contribution.command)
			result[index] = commandContribution{
				command: command,
				source:  slices.Clone(contribution.source),
			}
		case aliasContributionDraft:
			result[index] = aliasContribution{
				target: draft.freezeCommandTarget(contribution.target),
				alias:  contribution.alias,
				source: slices.Clone(contribution.source),
			}
		case *shortcutContributionDraft:
			if contribution == nil {
				panic("cli: nil provider shortcut contribution")
			}
			defaults := make([]batchShortcutDefault, len(contribution.defaults))
			for defaultIndex, value := range contribution.defaults {
				switch value := value.(type) {
				case destinationDefaultDraft:
					defaults[defaultIndex] = destinationDefault{
						destination: draft.freezeDestinationTarget(value.destination),
						value:       value.value,
						source:      slices.Clone(value.source),
					}
				case flagDefault:
					defaults[defaultIndex] = flagDefault{
						spelling: value.spelling,
						values:   slices.Clone(value.values),
						source:   slices.Clone(value.source),
					}
				default:
					panic("cli: invalid provider shortcut default")
				}
			}
			result[index] = shortcutContribution{
				target:      draft.freezeCommandTarget(contribution.target),
				shortcut:    snapshotProviderShortcut(contribution.shortcut),
				source:      slices.Clone(contribution.source),
				defaults:    defaults,
				replacement: contribution.replacement,
			}
		default:
			panic("cli: invalid provider contribution")
		}
	}
	return result
}

func (draft *providerBatchDraft) freezeCommandRef(ref commandRef) batchCommandRef {
	if draft == nil {
		panic("cli: nil provider batch draft")
	}
	if _, ok := draft.declarations.commands.resolve(&draft.declarations, ref); !ok {
		panic("cli: provider command reference cannot freeze")
	}
	return batchRefAt[batchCommand](ref.index)
}

func (draft *providerBatchDraft) freezeCommandTarget(target providerCommandTarget) batchCommandTarget {
	switch target := target.(type) {
	case stagedCommandTarget:
		return localCommandTarget{command: draft.freezeCommandRef(target.command)}
	case pathTarget:
		return pathTarget{path: slices.Clone(target.path)}
	default:
		panic("cli: invalid provider command target")
	}
}

func (draft *providerBatchDraft) freezeDestinationTarget(target destinationIdentity) batchDestinationTarget {
	switch target := target.(type) {
	case stagedDestinationIdentity:
		if _, ok := draft.declarations.destinations.resolve(&draft.declarations, target.ref); !ok {
			panic("cli: provider destination target cannot freeze")
		}
		return localDestinationTarget{destination: batchRefAt[batchDestination](target.ref.index)}
	case acceptedDestinationIdentity:
		return target
	default:
		panic("cli: invalid provider destination target")
	}
}
