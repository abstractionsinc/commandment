package cli

import "context"

func providerDestinationConstraintFailure(identity destinationIdentity) error {
	switch identity.(type) {
	case localDestinationTarget:
		panic("cli: validated local provider destination cannot fail publication")
	case acceptedDestinationIdentity:
		return errProviderBatchRejected
	default:
		panic("cli: validated provider destination has invalid publication identity")
	}
}

func resolveBatchDependencies(dependencies []batchDestinationDependency, state *providerPublicationState) ([]dependencyRequirement, bool) {
	if state == nil {
		return nil, len(dependencies) == 0
	}
	result := make([]dependencyRequirement, 0, len(dependencies))
	seen := make(map[ValueID]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		id, ok := state.batchValueID(dependency.destination)
		if !ok || !id.present() {
			return nil, false
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, false
		}
		seen[id] = struct{}{}
		result = append(result, acceptDependencyRequirement(id, dependency.optional))
	}

	return result, true
}

func prepareBatchCompletion(completion batchCompletion, state *providerPublicationState) (completionDeclaration, bool) {
	dependencies, ok := resolveBatchDependencies(completion.dependencies, state)
	if !ok {
		return completionDeclaration{}, false
	}
	candidate := completionDeclarationCandidate{
		Kind: completion.kind,
		Callback: completionCallbackCandidate{
			Dependencies: dependencies,
			Source:       completion.source,
		},
		Native: completionNativeCandidate{
			Kind:    completion.nativeKind,
			Mode:    completion.nativeMode,
			Present: completion.hasNative,
		},
	}
	definition := acceptDetachedCompletionDeclaration(candidate)
	return definition, definition.Valid()
}

func prepareBatchValueProducer(kind declaredProducerKind, computation batchValueComputation, state *providerPublicationState) (valueProducerRecord, bool) {
	dependencies, ok := resolveBatchDependencies(computation.dependencies, state)
	if !ok {
		return valueProducerRecord{}, false
	}
	producer := acceptValueProducer(kind, dependencies, computation.source)
	return producer, producer.Valid()
}

func prepareBatchIntrinsicDestinationProducer(draft *declarationCandidate, state *providerPublicationState, value batchDestination, destination acceptanceDestinationRef, position declarationPosition) bool {
	var kind declaredProducerKind
	switch value.definition.Kind() {
	case destinationKindDerived:
		kind = declaredProducerKindDerived
	case destinationKindOptionalDerived:
		kind = declaredProducerKindOptionalDerived
	case destinationKindFactResolved:
		kind = declaredProducerKindFact
	default:
		return true
	}
	producer, ok := prepareBatchValueProducer(kind, value.computation, state)
	if !ok {
		return false
	}
	draft.DestinationProducer(destination, producer, nil, position)
	return true
}

func batchDestinationAcceptanceInput(value batchDestination) destinationAcceptanceInput {
	input := destinationAcceptanceInput{
		sensitive:          value.sensitive,
		binding:            value.binding,
		bindingRequired:    true,
		definition:         value.definition,
		definitionComplete: true,
		definitionFinal:    true,
	}

	return input
}

func batchFlagAcceptanceInput(state *providerPublicationState, value batchFlag) flagAcceptanceInput {
	input := flagAcceptanceInput{
		destinationIndex:    state.mustRecordIndex(declarationRecordDestination, value.destination.sliceIndex()),
		producer:            value.producer,
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

	return input
}

func batchArgumentTermAcceptanceInput(state *providerPublicationState, value batchArgumentTerm) argumentTermAcceptanceInput {
	destinationIndex := -1
	if value.destination.valid() {
		destinationIndex = state.mustRecordIndex(declarationRecordDestination, value.destination.sliceIndex())
	}
	input := argumentTermAcceptanceInput{
		formIndex:          state.mustRecordIndex(declarationRecordArgumentForm, value.form.sliceIndex()),
		destinationIndex:   destinationIndex,
		kind:               value.kind,
		name:               value.name,
		count:              value.count,
		tokenLanguage:      value.tokenLanguage,
		literal:            value.literal,
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

	return input
}

func stageProviderDestinationDomains(draft *declarationCandidate, state *providerPublicationState, declarations *batchDeclarations, uses []batchDomainUse, destination acceptanceDestinationRef, position declarationPosition, metadata applicationMetadata) {
	for _, use := range uses {
		if !use.domain.valid() || use.kind == ValueDomainUseUnknown {
			panic("cli: validated provider batch cannot be published")
		}
		domain := stageProviderValueDomain(draft, state, declarations, use.domain, metadata)
		draft.ValueDomainUse(domain, destination, use.kind, position)
	}
}

func stageProviderValueDomain(draft *declarationCandidate, state *providerPublicationState, declarations *batchDeclarations, ref batchValueDomainRef, metadata applicationMetadata) valueDomainRef {
	if draft == nil || state == nil || declarations == nil || !ref.valid() {
		panic("cli: validated provider batch cannot be published")
	}
	if domain, exists := state.domains[ref]; exists {
		return domain
	}
	index := ref.sliceIndex()
	if index != state.nextDomain || index >= len(declarations.valueDomains) {
		panic("cli: validated provider value-domain order cannot be published")
	}
	value := declarations.valueDomains[index]
	var domain valueDomainRef
	if value.dynamic == nil {
		entries, err := prepareBatchDomainEntries(draft, value, metadata)
		if err != nil {
			panic("cli: validated provider value domain cannot be published")
		}
		domain = draft.ValueDomain(entries)
	} else {
		domain = stageProviderDynamicValueDomain(draft, state, value.dynamic)
	}
	state.domains[ref] = domain
	state.nextDomain++
	return domain
}

func stageProviderDynamicValueDomain(draft *declarationCandidate, state *providerPublicationState, value *batchDynamicValueDomain) valueDomainRef {
	if value == nil || value.commandIndex < -1 {
		panic("cli: validated provider batch cannot be published")
	}
	owner := state.mustCommand(value.commandIndex)
	dependencies, ok := resolveBatchDependencies(value.dependencies, state)
	if !ok {
		panic("cli: validated provider batch cannot be published")
	}
	definition := acceptDynamicDomainDefinition(owner.ID(), dependencies)
	return draft.DynamicValueDomain(definition, nil)
}

func providerReplacementRef(target batchReplacementTarget, state *providerPublicationState) replacementRef {
	replacement, ok := replacementRefForAcceptedSlot(target.kind, state.mustSlot(target.index))
	if !ok {
		panic("cli: validated provider replacement coordinate cannot resolve")
	}
	return replacement
}

func establishProviderDestination(declarations *batchDeclarations, metadata applicationMetadata, draft *declarationCandidate, state *providerPublicationState, index int, value batchDestination) acceptanceDestinationRef {
	if index < 0 || index >= len(declarations.order) {
		panic("cli: validated provider batch cannot be published")
	}
	record := declarations.order[index]
	owner := state.mustCommand(record.commandIndex)
	input := batchDestinationAcceptanceInput(value)
	definition, valid := prepareDestinationDefinitionForRelease(input, metadata.release)
	if !valid {
		panic("cli: validated provider destination cannot be published")
	}
	if !value.binding.Valid() {
		panic("cli: validated provider batch cannot be published")
	}
	position := state.position
	destination := draft.Destination(owner, position, definition, value.binding)
	draft.DestinationSensitive(destination, value.sensitive, position)
	state.slots[index].destination = destination
	if staged, ok := state.destinationRefs[index]; ok {
		state.destinations[staged] = destination
	}
	if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
		draft.DestinationProfile(destination, profile, position)
	}
	stageProviderDestinationDomains(draft, state, declarations, value.domainUses, destination, position, metadata)
	return destination
}

func stageProviderDeclarations(declarations *batchDeclarations, metadata applicationMetadata, rootInitial dialectInput, draft *declarationCandidate, state *providerPublicationState) error {
	if draft == nil || state == nil || !state.root.ID().present() || len(state.slots) != len(declarations.order) {
		panic("cli: validated provider batch cannot be published")
	}
	stageProviderCommandIdentity(declarations, draft, state)
	stageProviderGrammarProfiles(declarations, rootInitial, draft, state)

	for index := range len(declarations.order) {
		record := declarations.order[index]
		position := state.position
		owner := state.mustCommand(record.commandIndex)
		dialect := state.mustDialectForIndexes(record.commandIndex, record.profileIndex)
		switch record.kind {
		case declarationRecordCommand:
			// Allocated in the command-tree pass above.
		case declarationRecordAccess:
			// Applied after every destination identity is available.
		case declarationRecordShortcut, declarationRecordShortcutDirective:
			// Entry identity is allocated in the command-entry pass; fixed input,
			// defaults, and replacement are attached after destinations exist.
		case declarationRecordDestination:
			value := declarations.destinations[record.familyIndex]
			destination := establishProviderDestination(declarations, metadata, draft, state, index, value)
			if !prepareBatchIntrinsicDestinationProducer(draft, state, value, destination, position) {
				panic("cli: validated provider destination producer cannot be published")
			}
		case declarationRecordFlag:
			flagValue := batchFlagAcceptanceInput(state, declarations.flags[record.familyIndex])
			destination := state.mustDestination(flagValue.destinationIndex)
			if declarations.order[flagValue.destinationIndex].commandIndex != record.commandIndex {
				panic("cli: validated provider flag destination belongs to another command")
			}
			flag := draft.Flag(owner, destination, position)
			state.slots[index].flag = flag
			if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
				draft.FlagProfile(flag, profile, position)
			}
			draft.FlagLifecycle(flag, flagReleaseLifecycle(flagValue), position)
			flagPresentation, valueName, required, localeSelector := flagPresentation(flagValue)
			draft.FlagPresentation(flag, flagPresentation, valueName, required, localeSelector, position)
			if declarations.flags[record.familyIndex].completionSet {
				completion, valid := prepareBatchCompletion(declarations.flags[record.familyIndex].completion, state)
				if !valid {
					panic("cli: validated provider batch cannot be published")
				}
				draft.FlagCompletion(flag, completion, completionCallback{}, position)
			}
			destinationValue := batchDestinationAcceptanceInput(declarations.destinations[declarations.order[flagValue.destinationIndex].familyIndex])
			grammar, valid := prepareFlagGrammar(flagValue, destinationValue, dialect)
			if !valid {
				panic("cli: validated provider batch cannot be published")
			}
			draft.FlagGrammar(flag, grammar)
		case declarationRecordFlagSpelling:
			spellingValue := declarations.flagSpellings[record.familyIndex]
			flagIndex := state.mustRecordIndex(declarationRecordFlag, spellingValue.flag.sliceIndex())
			flag := state.mustSlot(flagIndex).flag
			if !flag.ID().present() || declarations.order[flagIndex].commandIndex != record.commandIndex {
				panic("cli: validated provider flag spelling owner cannot resolve")
			}
			spelling := draft.FlagSpelling(flag, position)
			state.slots[index].spelling = spelling
			flagValue := batchFlagAcceptanceInput(state, declarations.flags[declarations.order[flagIndex].familyIndex])
			producer := flagValue.producer
			mode, forms, sensitive, valid := prepareFlagSpellingGrammar(
				spellingValue.kind,
				producer,
				state.slots[index].spellingConfig,
			)
			if !valid {
				panic("cli: validated provider batch cannot be published")
			}
			kind, valid := prepareFlagSpellingKind(spellingValue.kind)
			if !valid {
				panic("cli: validated provider batch cannot be published")
			}
			draft.FlagSpellingGrammarWithKind(
				spelling,
				kind,
				spellingValue.text,
				mode,
				forms,
				sensitive,
			)
			definitionRef := state.mustDestination(flagValue.destinationIndex)
			definition, definitionOK := draft.existingDestinationDefinitionForRef(definitionRef)
			if !definitionOK || !definition.Valid() {
				panic("cli: validated provider destination definition is unavailable")
			}
			if state.slots[index].spellingConfig.fixed {
				if !validEffectiveDomainContribution(definition, state.slots[index].spellingConfig.fixedValue) {
					panic("cli: validated provider batch cannot be published")
				}
				draft.FlagSpellingFixedValue(
					spelling,
					state.slots[index].spellingConfig.fixedValue,
					state.slots[index].spellingConfig.fixedSensitive,
				)
			} else if producer == flagProducerBool {
				draft.FlagSpellingFixedValue(spelling, true, false)
			}
			if state.slots[index].spellingConfig.implicit {
				if !validEffectiveDomainContribution(definition, state.slots[index].spellingConfig.implicitVal) {
					panic("cli: validated provider batch cannot be published")
				}
				draft.FlagSpellingImplicitValue(
					spelling,
					state.slots[index].spellingConfig.implicitVal,
					state.slots[index].spellingConfig.implicitSensitive,
				)
			}
			if state.slots[index].spellingConfig.hidden {
				draft.FlagSpellingHidden(spelling, position)
			}
			if state.slots[index].spellingConfig.hasRelease {
				draft.FlagSpellingLifecycle(
					spelling,
					state.slots[index].spellingConfig.lifecycle,
					position,
				)
			}
		case declarationRecordFlagSpellingDirective:
			directive := declarations.flagSpellingDirectives[record.familyIndex]
			spellingIndex := state.mustRecordIndex(declarationRecordFlagSpelling, directive.spelling.sliceIndex())
			spellingSlot := state.mustSlot(spellingIndex)
			if !spellingSlot.spelling.ID().present() || declarations.order[spellingIndex].commandIndex != record.commandIndex {
				panic("cli: validated provider spelling directive owner cannot resolve")
			}
		case declarationRecordArgumentForm:
			form := draft.ArgumentForm(owner, position)
			state.slots[index].argumentForm = form
			if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
				draft.ArgumentFormProfile(form, profile, position)
			}
			draft.ArgumentFormGrammar(form)
		case declarationRecordArgumentTerm:
			termValue := batchArgumentTermAcceptanceInput(state, declarations.argumentTerms[record.familyIndex])
			form := state.mustSlot(termValue.formIndex).argumentForm
			if !form.ID().present() || declarations.order[termValue.formIndex].commandIndex != record.commandIndex {
				panic("cli: validated provider argument form cannot resolve")
			}
			var term argumentTermRef
			if termValue.kind == argumentTermLiteral {
				term = draft.ArgumentTerm(form, position)
			} else {
				destination := state.mustDestination(termValue.destinationIndex)
				if declarations.order[termValue.destinationIndex].commandIndex != record.commandIndex {
					panic("cli: validated provider argument destination belongs to another command")
				}
				term = draft.ArgumentValueTerm(form, destination, position)
			}
			state.slots[index].argumentTerm = term
			lifecycle, present := argumentReleaseLifecycle(termValue)
			if present {
				draft.ArgumentTermLifecycle(term, lifecycle, position)
			}
			draft.ArgumentTermPresentation(term, argumentPresentation(termValue), termValue.name, argumentSourceSensitive(termValue), position)
			if declarations.argumentTerms[record.familyIndex].completionSet {
				completion, valid := prepareBatchCompletion(declarations.argumentTerms[record.familyIndex].completion, state)
				if !valid {
					panic("cli: validated provider batch cannot be published")
				}
				draft.ArgumentTermCompletion(term, completion, completionCallback{}, position)
			}
			destinationValue := destinationAcceptanceInput{}
			if termValue.destinationIndex >= 0 {
				destinationValue = batchDestinationAcceptanceInput(declarations.destinations[declarations.order[termValue.destinationIndex].familyIndex])
			}
			if !prepareArgumentTermGrammar(draft, term, termValue, destinationValue) {
				panic("cli: validated provider batch cannot be published")
			}
		case declarationRecordArgumentFormSelection:
			// Applied after all form and destination identities exist.
		case declarationRecordConditionPolicy:
			// Applied after all declaration and destination identities exist.
		case declarationRecordLifecycleDirective:
			// Applied after all exact declaration identities exist.
		case declarationRecordValueDirective:
			recordValue := declarations.valueDirectives[record.familyIndex]
			destination, exists := state.acceptanceDestinationRef(draft, recordValue.destination)
			if !exists || !destination.ID().present() {
				panic("cli: validated provider batch cannot be published")
			}
			destinationDefinition, defined := draft.existingDestinationDefinitionForRef(destination)
			if !defined || !destinationDefinition.Valid() {
				panic("cli: validated provider batch cannot be published")
			}
			switch recordValue.kind {
			case valueDirectiveEnvironment, valueDirectiveEnvironmentPrefix:
				environmentInput := environmentInputAcceptanceInput{
					kind:             recordValue.kind,
					name:             recordValue.name,
					sensitive:        recordValue.sensitive,
					environmentShape: recordValue.environmentShape,
					delimiter:        recordValue.delimiter,
				}
				syntax, valid := prepareEnvironmentInputSyntax(environmentInput, destinationDefinition)
				if !valid {
					return providerDestinationConstraintFailure(recordValue.destination)
				}
				environment := draft.EnvironmentInputWithSyntax(
					owner,
					destination,
					position,
					syntax.selector,
					syntax.prefix,
					syntax.contribution,
					syntax.delimiter,
					syntax.sensitive,
				)
				state.slots[index].environment = environment
				if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
					draft.EnvironmentInputProfile(environment, profile, position)
				}
			case valueDirectiveResolution:
				policy, valid := prepareResolutionPolicy(recordValue.policies, destinationDefinition)
				if !valid {
					return providerDestinationConstraintFailure(recordValue.destination)
				}
				draft.DestinationResolution(destination, policy, position)
			case valueDirectiveDefault:
				definition, ok := draft.existingDestinationDefinitionForRef(destination)
				if !ok {
					panic("cli: validated provider destination has no definition")
				}
				if !validEffectiveDomainContribution(definition, recordValue.value) {
					return providerDestinationConstraintFailure(recordValue.destination)
				}
				source := snapshotProducerSource(nil, recordValue.sensitive)
				producer := acceptStaticValueProducer(
					declaredProducerKindDefault,
					recordValue.value,
					source,
				)
				if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
					producer.profile = profile.ID()
				}
				draft.DestinationProducer(destination, producer, nil, position)
			case valueDirectiveComputedDefault, valueDirectiveOptionalComputedDefault:
				kind := declaredProducerKindComputedDefault
				if recordValue.kind == valueDirectiveOptionalComputedDefault {
					kind = declaredProducerKindOptionalComputedDefault
				}
				producer, valid := prepareBatchValueProducer(kind, recordValue.computation, state)
				if !valid {
					panic("cli: validated provider batch cannot be published")
				}
				producer.source = producerSourceWithSensitive(producer.source, recordValue.sensitive)
				if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
					producer.profile = profile.ID()
				}
				draft.DestinationProducer(destination, producer, nil, position)
			default:
				panic("cli: validated provider batch cannot be published")
			}
		case declarationRecordRule:
			rule := draft.Rule(owner, position)
			state.slots[index].rule = rule
			if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
				draft.RuleProfile(rule, profile, position)
			}
		case declarationRecordRuleDirective:
			// Applied after all destination and Rule identities exist.
		case declarationRecordAdvisory:
			advisory := draft.Advisory(owner, position)
			state.slots[index].advisory = advisory
			if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
				draft.AdvisoryProfile(advisory, profile, position)
			}
		case declarationRecordAdvisoryDirective, declarationRecordNoticeSuppression:
			// Applied after all destination identities exist.
		case declarationRecordProvider:
			provider := draft.Provider(owner, position)
			state.slots[index].provider = provider
			if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
				draft.ProviderProfile(provider, profile, position)
			}
		case declarationRecordSignalClear, declarationRecordSignalPolicy:
			// Applied after destination, condition, and profile identities exist.
		case declarationRecordHelpGroup:
			// Membership is applied after command-entry, provider, and flag identities exist.
		case declarationRecordReplacement:
			// Replacement relationships are applied after every replaceable declaration identity exists.
		case declarationRecordGrammarProfileSwitch, declarationRecordGrammarProfileCase:
			// Profile switch/case identity and dialect are finalized after selector destinations exist.
		case declarationRecordDialectDirective:
			// Already folded into the command/profile fully effective dialect above.
		default:
			panic("cli: validated provider batch cannot be published")
		}
	}

	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordGrammarProfileCase {
			continue
		}
		recordValue := declarations.grammarProfileCases[record.familyIndex]
		switchIndex := state.mustRecordIndex(declarationRecordGrammarProfileSwitch, recordValue.switchRef.sliceIndex())
		if switchIndex < 0 || switchIndex >= len(declarations.order) {
			panic("cli: validated provider batch cannot be published")
		}
		switchRecord := declarations.order[switchIndex]
		profileSwitch := declarations.grammarProfileSwitches[switchRecord.familyIndex]
		selector, ok := state.acceptanceDestinationRef(draft, profileSwitch.selector)
		profile := state.mustProfile(index)
		dialect := state.mustSlot(index).dialect
		if !ok || !validSemanticDialectInput(dialect) || !selector.ID().present() {
			panic("cli: validated provider grammar profile cannot be published")
		}
		if recordValue.defaultCase {
			draft.GrammarProfileDefaultCase(
				profile,
				selector,
				recordValue.value,
				profileSwitch.equal,
				dialect,
				state.position,
			)
		} else {
			draft.GrammarProfileCase(
				profile,
				selector,
				recordValue.value,
				profileSwitch.equal,
				dialect,
				state.position,
			)
		}
	}

	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordArgumentFormSelection {
			continue
		}
		recordValue := declarations.argumentFormSelections[record.familyIndex]
		if recordValue.destination == nil {
			panic("cli: validated provider batch cannot be published")
		}
		formIndex := state.mustRecordIndex(declarationRecordArgumentForm, recordValue.form.sliceIndex())
		form := state.mustSlot(formIndex).argumentForm
		if !form.ID().present() {
			panic("cli: validated provider argument form cannot be published")
		}
		destination, ok := state.acceptanceDestinationRef(draft, recordValue.destination)
		if !ok || !destination.ID().present() {
			panic("cli: validated provider batch cannot be published")
		}
		definition, ok := draft.existingDestinationDefinitionForRef(destination)
		if !ok {
			panic("cli: validated provider form-selection destination has no definition")
		}
		if !validEffectiveDomainContribution(definition, recordValue.value) {
			return providerDestinationConstraintFailure(recordValue.destination)
		}
		draft.ArgumentFormSelection(form, destination, recordValue.value, state.position)
	}

	stageProviderExecution(declarations, draft, state)

	stageProviderConditions(declarations, draft, state)

	stageProviderShortcuts(draft, declarations, state)
	stageProviderDefinitions(draft, declarations, state)

	stageProviderPolicies(draft, declarations, state)
	stageProviderSignals(draft, declarations, state)

	stageProviderHelpGroups(draft, declarations, state)

	stageProviderLifecycles(declarations, draft, state)

	stageProviderReplacements(draft, declarations, state)

	if state.nextDomain != len(declarations.valueDomains) {
		panic("cli: validated provider value-domain publication is incomplete")
	}
	return nil
}

func stageProviderCommandIdentity(declarations *batchDeclarations, draft *declarationCandidate, state *providerPublicationState) {
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind == declarationRecordShortcut {
			if record.commandIndex < 0 || record.commandIndex >= len(declarations.order) {
				panic("cli: validated provider shortcut has invalid command coordinate")
			}
			target := state.mustCommand(record.commandIndex)
			commandRecord := declarations.order[record.commandIndex]
			if commandRecord.kind != declarationRecordCommand {
				panic("cli: validated provider batch cannot be published")
			}
			parent := state.mustCommand(commandRecord.commandIndex)
			shortcut := declarations.shortcuts[record.familyIndex]
			position := state.position
			entry := draft.CommandEntry(parent, target, position)
			draft.CommandEntryGrammar(entry, CommandEntryShortcut, shortcut.name, true)
			draft.CommandEntryPresentation(
				entry,
				semanticPresentation(shortcut.description, shortcut.examples, shortcut.hidden),
				position,
			)
			draft.CommandEntryLifecycle(entry, shortcut.release, position)
			state.slots[index].shortcutEntry = entry
			state.slots[record.commandIndex].commandEntries = append(
				state.slots[record.commandIndex].commandEntries,
				entry,
			)
			continue
		}
		if record.kind != declarationRecordCommand {
			continue
		}
		parent := state.mustCommand(record.commandIndex)
		position := state.position
		value := declarations.commands[record.familyIndex]
		configuration := acceptCommandConfiguration(value.makeConfig)
		command := draft.Command(parent, position, configuration)
		state.slots[index].command = command
		commandPresentation := semanticPresentation(value.description, value.examples, value.hidden)
		draft.CommandPresentation(command, commandPresentation, position)
		entry := draft.CommandEntry(parent, command, position)
		draft.CommandEntryGrammar(entry, CommandEntryCanonical, value.name, true)
		draft.CommandEntryPresentation(entry, commandPresentation, position)
		draft.CommandEntryLifecycle(entry, value.release, position)
		state.slots[index].commandEntries = append(state.slots[index].commandEntries, entry)
		for _, alias := range value.aliases {
			aliasEntry := draft.CommandEntry(parent, command, position)
			draft.CommandEntryGrammar(aliasEntry, CommandEntryAlias, alias.Name, true)
			draft.CommandEntryPresentation(
				aliasEntry,
				semanticPresentation(alias.Description, nil, alias.Hidden),
				position,
			)
			draft.CommandEntryLifecycle(aliasEntry, alias.Release, position)
			state.slots[index].commandEntries = append(state.slots[index].commandEntries, aliasEntry)
		}
		switch value.defaults.state {
		case commandDefaultUnspecified:
		case commandDefaultUnconditional, commandDefaultConditional:
			state.slots[index].defaultRef = draft.Default(parent, command, position)
		default:
			panic("cli: validated provider batch cannot be published")
		}
	}
}

func stageProviderGrammarProfiles(declarations *batchDeclarations, rootInitial dialectInput, draft *declarationCandidate, state *providerPublicationState) dialectInput {
	rootDialect := applyProviderDialectDirectives(rootInitial, declarations, -1, -1)
	state.rootDialect = rootDialect
	draft.Dialect(state.root, rootDialect)
	for index := range len(declarations.order) {
		record := declarations.order[index]
		position := state.position
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
				panic("cli: validated provider grammar profile has no base dialect")
			}
			owner := state.mustCommand(record.commandIndex)
			dialect := applyProviderDialectDirectives(base, declarations, record.commandIndex, index)
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
				panic("cli: validated provider batch cannot be published")
			}
			dialect := applyProviderDialectDirectives(base, declarations, index, -1)
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

	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordShortcut || record.profileIndex < 0 {
			continue
		}
		entry := state.slots[index].shortcutEntry
		profile := state.mustProfile(record.profileIndex)
		if !entry.ID().present() {
			panic("cli: validated provider shortcut entry cannot resolve")
		}
		draft.CommandEntryProfile(entry, profile, state.position)
	}
	return rootDialect
}

func stageProviderExecution(declarations *batchDeclarations, draft *declarationCandidate, state *providerPublicationState) {
	if state == nil {
		panic("cli: validated provider batch cannot be published")
	}
	rootAccess := make([]accessDefinition, 0)
	accessByCommand := make([][]accessDefinition, len(declarations.order))
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordAccess {
			continue
		}
		if record.commandIndex == -1 {
			rootAccess = append(rootAccess, declarations.accesses[record.familyIndex])
			continue
		}
		if record.commandIndex < 0 || record.commandIndex >= len(declarations.order) {
			panic("cli: validated provider batch cannot be published")
		}
		accessByCommand[record.commandIndex] = append(
			accessByCommand[record.commandIndex],
			declarations.accesses[record.familyIndex],
		)
	}
	if len(rootAccess) != 0 {
		access, valid := prepareExecutionAccess(rootAccess, state)
		if !valid {
			panic("cli: validated provider batch cannot be published")
		}
		draft.CommandAccess(state.root, access, 0)
	}
	for index, declarations := range accessByCommand {
		if len(declarations) == 0 {
			continue
		}
		command := state.mustCommand(index)
		access, valid := prepareExecutionAccess(declarations, state)
		if !valid {
			panic("cli: validated provider batch cannot be published")
		}
		draft.CommandAccess(command, access, state.position)
	}
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordCommand {
			continue
		}
		command := state.mustCommand(index)
		value := declarations.commands[record.familyIndex]
		execution := acceptCommandExecution(value.middleware, value.handler)
		draft.CommandExecution(command, execution, state.position)
	}
}

func stageProviderConditions(declarations *batchDeclarations, draft *declarationCandidate, state *providerPublicationState) {
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordCommand {
			continue
		}
		value := declarations.commands[record.familyIndex]
		if value.defaults.state != commandDefaultConditional {
			continue
		}
		defaultRef := state.slots[index].defaultRef
		if !defaultRef.ID().present() {
			panic("cli: validated provider batch cannot be published")
		}
		condition, valid := prepareCondition(value.defaults.condition, state)
		if !valid {
			panic("cli: validated provider batch cannot be published")
		}
		draft.DefaultCondition(defaultRef, condition, state.position)
	}
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordConditionPolicy {
			continue
		}
		value := declarations.conditionPolicies[record.familyIndex]
		condition, valid := prepareCondition(value.condition, state)
		if !valid {
			panic("cli: validated provider batch cannot be published")
		}
		position := state.position
		conditionPolicyIndex := draft.ConditionPolicyCount()
		switch value.target.kind {
		case batchConditionTargetCommand:
			targetIndex := state.mustRecordIndex(declarationRecordCommand, value.target.command.sliceIndex())
			command := state.mustCommand(targetIndex)
			draft.CommandCondition(command, value.effect, condition, position)
		case batchConditionTargetFlag:
			targetIndex := state.mustRecordIndex(declarationRecordFlag, value.target.flag.sliceIndex())
			slot := state.mustSlot(targetIndex)
			if !slot.flag.ID().present() {
				panic("cli: validated provider condition flag target cannot resolve")
			}
			draft.FlagCondition(slot.flag, value.effect, condition, position)
		case batchConditionTargetArgumentForm:
			targetIndex := state.mustRecordIndex(declarationRecordArgumentForm, value.target.argumentForm.sliceIndex())
			slot := state.mustSlot(targetIndex)
			if !slot.argumentForm.ID().present() {
				panic("cli: validated provider condition argument-form target cannot resolve")
			}
			draft.ArgumentFormCondition(slot.argumentForm, value.effect, condition, position)
		case batchConditionTargetArgumentTerm:
			targetIndex := state.mustRecordIndex(declarationRecordArgumentTerm, value.target.argumentTerm.sliceIndex())
			slot := state.mustSlot(targetIndex)
			if !slot.argumentTerm.ID().present() {
				panic("cli: validated provider condition argument target cannot resolve")
			}
			draft.ArgumentTermCondition(slot.argumentTerm, value.effect, condition, position)
		default:
			panic("cli: validated provider batch cannot be published")
		}
		if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
			draft.ConditionPolicyProfile(conditionPolicyIndex, profile, position)
		}
	}
}

func stageProviderLifecycles(declarations *batchDeclarations, draft *declarationCandidate, state *providerPublicationState) {
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordLifecycleDirective {
			continue
		}
		value := declarations.lifecycleDirectives[record.familyIndex]
		position := state.position
		switch value.target.kind {
		case batchLifecycleTargetEnvironment:
			targetIndex := state.mustRecordIndex(declarationRecordValueDirective, value.target.environment.sliceIndex())
			slot := state.mustSlot(targetIndex)
			if !slot.environment.ID().present() {
				panic("cli: validated provider lifecycle environment target cannot resolve")
			}
			draft.EnvironmentInputLifecycle(slot.environment, value.lifecycle, position)
		case batchLifecycleTargetArgumentForm:
			targetIndex := state.mustRecordIndex(declarationRecordArgumentForm, value.target.argumentForm.sliceIndex())
			slot := state.mustSlot(targetIndex)
			if !slot.argumentForm.ID().present() {
				panic("cli: validated provider lifecycle argument-form target cannot resolve")
			}
			draft.ArgumentFormLifecycle(slot.argumentForm, value.lifecycle, position)
		case batchLifecycleTargetArgumentTerm:
			targetIndex := state.mustRecordIndex(declarationRecordArgumentTerm, value.target.argumentTerm.sliceIndex())
			slot := state.mustSlot(targetIndex)
			if !slot.argumentTerm.ID().present() {
				panic("cli: validated provider lifecycle argument target cannot resolve")
			}
			draft.ArgumentTermLifecycle(slot.argumentTerm, value.lifecycle, position)
		default:
			panic("cli: validated provider batch cannot be published")
		}
	}
}

func collectProviderSpellingConfigs(declarations *batchDeclarations, state *providerPublicationState) {
	if state == nil || len(state.slots) != len(declarations.order) {
		panic("cli: validated provider batch cannot be published")
	}
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordFlagSpellingDirective {
			continue
		}
		directive := declarations.flagSpellingDirectives[record.familyIndex]
		spellingIndex := state.mustRecordIndex(declarationRecordFlagSpelling, directive.spelling.sliceIndex())
		if spellingIndex < 0 || spellingIndex >= len(state.slots) {
			panic("cli: validated provider batch cannot be published")
		}
		acceptedDirective := spellingDirectiveAcceptanceInput{
			kind:          directive.kind,
			forms:         directive.forms,
			value:         directive.value,
			sourceOptions: directive.sourceOptions,
			lifecycle:     directive.lifecycle,
		}
		config, ok := applySpellingDirective(state.slots[spellingIndex].spellingConfig, acceptedDirective)
		if !ok {
			panic("cli: validated provider spelling directive cannot be applied")
		}
		state.slots[spellingIndex].spellingConfig = config
	}
}

func applyProviderDialectDirectives(initial dialectInput, declarations *batchDeclarations, commandIndex, profileIndex int) dialectInput {
	result := initial
	endOfOptionsConfigured := false
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordDialectDirective ||
			record.commandIndex != commandIndex || record.profileIndex != profileIndex {
			continue
		}
		directive := declarations.dialectDirectives[record.familyIndex]
		var ok bool
		result, endOfOptionsConfigured, ok = applyDialectDirective(result, directive, endOfOptionsConfigured)
		if !ok {
			panic("cli: validated provider dialect directive cannot be applied")
		}
	}
	if !validSemanticDialectInput(result) {
		panic("cli: validated provider batch cannot be published")
	}
	return result
}

func stageProviderHelpGroups(draft *declarationCandidate, declarations *batchDeclarations, state *providerPublicationState) {
	groups := make(map[helpGroupAcceptanceKey]*helpGroupAcceptanceState)
	groupOrder := make([]helpGroupAcceptanceKey, 0)
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordHelpGroup {
			continue
		}
		command := state.mustCommand(record.commandIndex)
		profileID := GrammarProfileID(0)
		if record.profileIndex >= 0 {
			profile := state.mustProfile(record.profileIndex)
			profileID = profile.ID()
		}
		value := declarations.helpGroups[record.familyIndex]
		key := helpGroupAcceptanceKey{
			command: command.ID(),
			profile: profileID,
			family:  value.family,
			title:   value.title,
		}
		state.slots[index].helpGroupKey = key
		state.slots[index].helpGroupSet = true
		if existing := groups[key]; existing != nil {
			if existing.order != value.order {
				panic("cli: validated provider batch cannot be published")
			}
			continue
		}
		groups[key] = &helpGroupAcceptanceState{
			order:    value.order,
			position: state.position,
		}
		groupOrder = append(groupOrder, key)
	}

	for index := range len(declarations.order) {
		record := declarations.order[index]
		stageProviderCommandHelpGroupMembership(index, record, declarations, state, groups)
		stageProviderFlagHelpGroupMembership(index, record, declarations, state, groups)
	}

	publishHelpGroups(draft, groupOrder, groups)
}

func stageProviderCommandHelpGroupMembership(index int, record batchDeclarationOrderEntry, declarations *batchDeclarations, state *providerPublicationState, groups map[helpGroupAcceptanceKey]*helpGroupAcceptanceState) {
	if record.commandHelpGroupIndex < 0 {
		return
	}
	groupIndex := record.commandHelpGroupIndex
	if groupIndex >= len(declarations.order) || !state.slots[groupIndex].helpGroupSet {
		panic("cli: validated provider batch cannot be published")
	}
	key := state.slots[groupIndex].helpGroupKey
	group := groups[key]
	if group == nil || key.family != helpGroupFamilyCommand {
		panic("cli: validated provider batch cannot be published")
	}
	switch record.kind {
	case declarationRecordCommand:
		entries := state.slots[index].commandEntries
		if len(entries) == 0 {
			panic("cli: validated provider batch cannot be published")
		}
		for _, entry := range entries {
			if !entry.ID().present() {
				panic("cli: validated provider batch cannot be published")
			}
			group.entries = append(group.entries, acceptHelpCommandEntry(entry.ID()))
		}
	case declarationRecordProvider:
		provider := state.slots[index].provider
		if !provider.ID().present() {
			panic("cli: validated provider batch cannot be published")
		}
		group.entries = append(group.entries, acceptHelpProviderEntry(provider.ID()))
	case declarationRecordHelpGroup:
	default:
		panic("cli: validated provider batch cannot be published")
	}
}

func stageProviderFlagHelpGroupMembership(index int, record batchDeclarationOrderEntry, declarations *batchDeclarations, state *providerPublicationState, groups map[helpGroupAcceptanceKey]*helpGroupAcceptanceState) {
	if record.flagHelpGroupIndex < 0 {
		return
	}
	groupIndex := record.flagHelpGroupIndex
	if groupIndex >= len(declarations.order) || !state.slots[groupIndex].helpGroupSet {
		panic("cli: validated provider batch cannot be published")
	}
	key := state.slots[groupIndex].helpGroupKey
	group := groups[key]
	if group == nil || key.family != helpGroupFamilyFlag {
		panic("cli: validated provider batch cannot be published")
	}
	switch record.kind {
	case declarationRecordFlag:
		flag := state.slots[index].flag
		if !flag.ID().present() {
			panic("cli: validated provider batch cannot be published")
		}
		group.flags = append(group.flags, flag.ID())
	case declarationRecordFlagSpelling, declarationRecordFlagSpellingDirective, declarationRecordHelpGroup:
	default:
		panic("cli: validated provider batch cannot be published")
	}
}

func stageProviderPolicies(draft *declarationCandidate, declarations *batchDeclarations, state *providerPublicationState) {
	if draft == nil || state == nil {
		panic("cli: validated provider batch cannot be published")
	}
	ruleSubjects := make([][]ValueID, len(declarations.order))
	ruleDependencies := make([][]dependencyRequirement, len(declarations.order))
	ruleLookups := make([]resolvedDestinationDependencies, len(declarations.order))
	advisorySubjects := make([][]ValueID, len(declarations.order))
	advisoryDependencies := make([][]dependencyRequirement, len(declarations.order))
	advisoryLookups := make([]resolvedDestinationDependencies, len(declarations.order))

	for index := range len(declarations.order) {
		record := declarations.order[index]
		kind := record.kind
		switch kind {
		case declarationRecordRuleDirective:
			recordValue := declarations.ruleDirectives[record.familyIndex]
			ruleIndex := state.mustRecordIndex(declarationRecordRule, recordValue.rule.sliceIndex())
			if ruleIndex < 0 || ruleIndex >= index || declarations.order[ruleIndex].commandIndex != record.commandIndex ||
				declarations.order[ruleIndex].profileIndex != record.profileIndex {
				panic("cli: validated provider batch cannot be published")
			}
			id, ok := state.ValueID(recordValue.destination)
			if !ok {
				panic("cli: validated provider batch cannot be published")
			}
			addDestinationRelationLookup(&ruleLookups[ruleIndex], recordValue.destination, id)
			switch recordValue.kind {
			case ruleDirectiveSubject:
				ruleSubjects[ruleIndex] = appendUniqueDestination(ruleSubjects[ruleIndex], id)
			case ruleDirectiveDependency:
				ruleDependencies[ruleIndex] = appendUniqueDependency(
					ruleDependencies[ruleIndex],
					acceptDependencyRequirement(id, recordValue.optional),
				)
			case ruleDirectiveGroupDependency:
				ruleSubjects[ruleIndex] = appendUniqueDestination(ruleSubjects[ruleIndex], id)
				ruleDependencies[ruleIndex] = appendUniqueDependency(
					ruleDependencies[ruleIndex],
					acceptDependencyRequirement(id, recordValue.optional),
				)
			default:
				panic("cli: validated provider batch cannot be published")
			}
		case declarationRecordAdvisoryDirective:
			recordValue := declarations.advisoryDirectives[record.familyIndex]
			advisoryIndex := state.mustRecordIndex(declarationRecordAdvisory, recordValue.advisory.sliceIndex())
			if advisoryIndex < 0 || advisoryIndex >= index ||
				declarations.order[advisoryIndex].commandIndex != record.commandIndex ||
				declarations.order[advisoryIndex].profileIndex != record.profileIndex {
				panic("cli: validated provider batch cannot be published")
			}
			id, ok := state.ValueID(recordValue.destination)
			if !ok {
				panic("cli: validated provider batch cannot be published")
			}
			addDestinationRelationLookup(&advisoryLookups[advisoryIndex], recordValue.destination, id)
			switch recordValue.kind {
			case advisoryDirectiveSubject:
				advisorySubjects[advisoryIndex] = appendUniqueDestination(advisorySubjects[advisoryIndex], id)
			case advisoryDirectiveDependency:
				advisoryDependencies[advisoryIndex] = appendUniqueDependency(
					advisoryDependencies[advisoryIndex],
					acceptDependencyRequirement(id, recordValue.optional),
				)
			default:
				panic("cli: validated provider batch cannot be published")
			}
		default:
			continue
		}
	}

	for index := range len(declarations.order) {
		record := declarations.order[index]
		position := state.position
		kind := record.kind
		switch kind {
		case declarationRecordRule:
			ref := state.slots[index].rule
			if !ref.ID().present() {
				panic("cli: validated provider batch cannot be published")
			}
			recordValue := declarations.rules[record.familyIndex]
			subjects := append([]ValueID(nil), ruleSubjects[index]...)
			dependencies := append([]dependencyRequirement(nil), ruleDependencies[index]...)
			lookup := &ruleLookups[index]
			for _, relation := range recordValue.destinations {
				id, exists := state.ValueID(relation)
				if !exists {
					panic("cli: validated provider batch cannot be published")
				}
				addDestinationRelationLookup(lookup, relation, id)
				subjects = appendUniqueDestination(subjects, id)
				dependencies = appendUniqueDependency(dependencies, acceptDependencyRequirement(id, false))
			}
			var activation *policyCondition
			if recordValue.activation != nil {
				condition, valid := prepareRuleCondition(recordValue.activation, state)
				if !valid {
					panic("cli: validated provider batch cannot be published")
				}
				activation = &condition
			}
			callbackKind := ruleCallbackNone
			var callback ruleCallback
			if recordValue.callback != nil {
				switch recordValue.callbackKind {
				case authoringRuleCallbackCheck:
					callbackKind = ruleCallbackCheck
				case authoringRuleCallbackValidate:
					callbackKind = ruleCallbackValidate
				default:
					panic("cli: validated provider batch cannot be published")
				}
				erased := recordValue.callback
				frozen := freezeResolvedDestinationDependencies(ruleLookups[index])
				callback = func(ctx context.Context, configuration any, dependencies Dependencies) (ruleCallbackResult, error) {
					ctx = contextWithValueStateKeyAliases(ctx, frozen.keys)
					return erased(ctx, configuration, dependencies)
				}
			}
			count := semanticCountRange{}
			if recordValue.count.valid {
				if recordValue.count.unbounded {
					count = unboundedCount(recordValue.count.minimum)
				} else {
					count = boundedCount(recordValue.count.minimum, recordValue.count.maximum)
				}
			}
			var builtinPredicate ruleBuiltinPredicate
			if recordValue.builtinPredicate != nil {
				erased := recordValue.builtinPredicate
				builtinPredicate = func(values []any) bool { return erased(values) }
			}
			var definition ruleDefinition
			if recordValue.kind == RuleCustom {
				input := customRuleDefinitionInput{
					Code:         string(recordValue.code),
					Subjects:     subjects,
					Dependencies: dependencies,
					Activation:   activation,
					Callback:     callback,
				}
				switch callbackKind {
				case ruleCallbackCheck:
					definition = acceptCustomRuleCheckDefinition(input)
				case ruleCallbackValidate:
					definition = acceptCustomRuleValidateDefinition(input)
				default:
					panic("semantic: invalid Rule definition")
				}
			} else {
				input := builtinRuleDefinitionInput{
					Kind:         recordValue.kind,
					Subjects:     subjects,
					Dependencies: dependencies,
					Operands:     recordValue.operands,
					Activation:   activation,
					Count:        count,
					Pattern:      recordValue.pattern,
					Predicate:    builtinPredicate,
				}
				definition = acceptBuiltinRuleDefinition(input)
			}
			draft.RuleDefinition(ref, definition, position)
		case declarationRecordAdvisory:
			ref := state.slots[index].advisory
			if !ref.ID().present() {
				panic("cli: validated provider batch cannot be published")
			}
			recordValue := declarations.advisories[record.familyIndex]
			dependencies := append([]dependencyRequirement(nil), advisoryDependencies[index]...)
			if recordValue.kind != AdvisoryKindDetector && len(dependencies) != 0 {
				panic("cli: validated provider batch cannot be published")
			}
			var activation *condition
			if recordValue.kind == AdvisoryKindConditional {
				condition, valid := prepareCondition(recordValue.condition, state)
				if !valid {
					panic("cli: validated provider batch cannot be published")
				}
				activation = &condition
			}
			var callback advisoryCallback
			if recordValue.callback != nil {
				erased := recordValue.callback
				frozen := freezeResolvedDestinationDependencies(advisoryLookups[index])
				callback = func(ctx context.Context, configuration any, dependencies Dependencies) (bool, error) {
					ctx = contextWithValueStateKeyAliases(ctx, frozen.keys)
					return erased(ctx, configuration, dependencies)
				}
			}
			content := advisoryContent{
				Code:     string(recordValue.advisory.Code),
				Severity: uint8(recordValue.advisory.Severity),
				Message:  recordValue.advisory.Message,
				Reason:   recordValue.advisory.Reason,
				Hint:     recordValue.advisory.Hint,
			}
			var definition advisoryDefinition
			switch recordValue.kind {
			case AdvisoryKindAlways:
				if activation != nil || len(dependencies) != 0 || callback != nil {
					panic("semantic: invalid Advisory definition")
				}
				definition = acceptAlwaysAdvisoryDefinition(content, advisorySubjects[index])
			case AdvisoryKindConditional:
				condition := condition{}
				if activation != nil {
					condition = *activation
				}
				if len(dependencies) != 0 || callback != nil {
					panic("semantic: invalid Advisory definition")
				}
				definition = acceptConditionalAdvisoryDefinition(content, condition, advisorySubjects[index])
			case AdvisoryKindDetector:
				if activation != nil {
					panic("semantic: invalid Advisory definition")
				}
				definition = acceptDetectorAdvisoryDefinition(
					content, advisorySubjects[index], dependencies, callback,
				)
			default:
				panic("semantic: invalid Advisory definition")
			}
			draft.AdvisoryDefinition(ref, definition, position)
		case declarationRecordNoticeSuppression:
			owner := state.mustCommand(record.commandIndex)
			recordValue := declarations.noticeSuppressions[record.familyIndex]
			condition, valid := prepareCondition(recordValue.condition, state)
			if !valid {
				panic("cli: validated provider batch cannot be published")
			}
			match, valid := prepareNoticeMatch(recordValue.match)
			if !valid {
				panic("cli: validated provider batch cannot be published")
			}
			suppressionIndex := draft.NoticeSuppression(owner, match, condition, position)
			if record.profileIndex >= 0 {
				profile := state.mustProfile(record.profileIndex)
				draft.NoticeSuppressionProfile(suppressionIndex, profile, position)
			}
		default:
			continue
		}
	}
}

func stageProviderDefinitions(draft *declarationCandidate, declarations *batchDeclarations, state *providerPublicationState) {
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordProvider {
			continue
		}
		provider := state.slots[index].provider
		if !provider.ID().present() {
			panic("cli: validated provider batch cannot be published")
		}
		value := declarations.providers[record.familyIndex]
		dependencies := make([]dependencyRequirement, 0, len(value.dependencies))
		for _, dependency := range value.dependencies {
			id, ok := state.batchValueID(dependency.destination)
			if !ok {
				panic("cli: validated provider batch cannot be published")
			}
			dependencies = append(dependencies, acceptDependencyRequirement(id, dependency.optional))
		}
		definition := acceptProviderDefinition(dependencies)
		draft.ProviderDefinition(provider, definition, nil, state.position)
	}
}

func stageProviderReplacements(draft *declarationCandidate, declarations *batchDeclarations, state *providerPublicationState) {
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordReplacement {
			continue
		}
		replacement := declarations.replacements[record.familyIndex]
		source := providerReplacementRef(replacement.source, state)
		target := providerReplacementRef(replacement.target, state)
		draft.DeclarationReplacement(source, target, state.position)
	}
}

func stageProviderShortcuts(draft *declarationCandidate, declarations *batchDeclarations, state *providerPublicationState) {
	defaults := make([][]shortcutDefault, len(declarations.order))
	replacements := make([]int, len(declarations.order))
	for index := range replacements {
		replacements[index] = -1
	}
	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordShortcutDirective {
			continue
		}
		recordValue := declarations.shortcutDirectives[record.familyIndex]
		shortcutIndex := state.mustRecordIndex(declarationRecordShortcut, recordValue.shortcut.sliceIndex())
		slot := state.mustSlot(shortcutIndex)
		if !slot.shortcutEntry.ID().present() {
			panic("cli: validated provider shortcut directive owner cannot resolve")
		}
		switch recordValue.kind {
		case commandShortcutDirectiveDefault:
			id, ok := state.ValueID(recordValue.destination)
			if !ok {
				panic("cli: validated provider batch cannot be published")
			}
			definition, defined := draft.ExistingDestinationDefinition(id)
			if !defined || !validEffectiveDomainContribution(definition, recordValue.value) {
				panic("cli: validated provider batch cannot be published")
			}
			source, ok := producerSource(recordValue.sourceOptions)
			if !ok {
				panic("cli: validated provider batch cannot be published")
			}
			defaults[shortcutIndex] = append(
				defaults[shortcutIndex],
				acceptShortcutDestinationDefault(id, recordValue.value, source),
			)
		case commandShortcutDirectiveReplacement:
			if !recordValue.replacement.valid() {
				panic("cli: validated provider batch cannot be published")
			}
			targetIndex := state.mustRecordIndex(declarationRecordShortcut, recordValue.replacement.sliceIndex())
			target := state.mustSlot(targetIndex)
			if !target.shortcutEntry.ID().present() || replacements[shortcutIndex] >= 0 {
				panic("cli: validated provider shortcut replacement cannot resolve")
			}
			replacements[shortcutIndex] = targetIndex
		default:
			panic("cli: validated provider batch cannot be published")
		}
	}

	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordShortcut {
			continue
		}
		entry := state.slots[index].shortcutEntry
		if !entry.ID().present() {
			panic("cli: validated provider batch cannot be published")
		}
		value := declarations.shortcuts[record.familyIndex]
		definition := acceptShortcutDefinition(
			value.args,
			value.appendArgs,
			defaults[index],
			snapshotProducerSource(nil, false),
		)
		if !definition.HasContribution() {
			panic("cli: validated provider batch cannot be published")
		}
		position := state.position
		draft.CommandEntryShortcut(entry, definition, position)
		if targetIndex := replacements[index]; targetIndex >= 0 {
			target := state.slots[targetIndex].shortcutEntry
			if !target.ID().present() {
				panic("cli: validated provider batch cannot be published")
			}
			draft.CommandEntryReplacement(entry, target, position)
		}
	}
}

func stageProviderSignals(draft *declarationCandidate, declarations *batchDeclarations, state *providerPublicationState) {
	if draft == nil {
		panic("cli: validated provider batch cannot be published")
	}
	counters := make(map[any]signalCounterRef)
	endpoints := make(map[any]signalEndpointRef)

	for index := range len(declarations.order) {
		record := declarations.order[index]
		if record.kind != declarationRecordSignalClear && record.kind != declarationRecordSignalPolicy {
			continue
		}
		position := state.position
		command := state.mustCommand(record.commandIndex)
		var profile grammarProfileRef
		hasProfile := record.profileIndex >= 0
		if hasProfile {
			profile = state.mustProfile(record.profileIndex)
		}
		switch record.kind {
		case declarationRecordSignalClear:
			draft.SignalPolicyScope(command, profile, true, position)
		case declarationRecordSignalPolicy:
			policyValue := batchSignalPolicyAcceptanceInput(declarations, declarations.signalPolicies[record.familyIndex])
			if policyValue.match == nil || policyValue.plan == nil || policyValue.match.lineage == nil || len(policyValue.plan.actions) == 0 {
				panic("cli: validated provider batch cannot be published")
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
				panic("cli: validated provider signal policy cannot be published")
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
}

type batchRecordCoordinate struct {
	kind        declarationRecordKind
	familyIndex int
}

type providerPublicationState struct {
	slots           []declarationAcceptanceSlot
	root            acceptanceCommandRef
	rootDialect     dialectInput
	domains         map[batchValueDomainRef]valueDomainRef
	nextDomain      int
	destinationRefs map[int]batchDestinationRef
	destinations    map[batchDestinationRef]acceptanceDestinationRef
	recordIndexes   map[batchRecordCoordinate]int
	position        declarationPosition
}

func prepareProviderPublicationState(declarations batchDeclarations, root acceptanceCommandRef, position declarationPosition) *providerPublicationState {
	if !root.ID().present() || position == 0 {
		panic("cli: invalid provider publication frontier")
	}
	state := &providerPublicationState{
		slots:           make([]declarationAcceptanceSlot, len(declarations.order)),
		root:            root,
		domains:         make(map[batchValueDomainRef]valueDomainRef, len(declarations.valueDomains)),
		destinationRefs: make(map[int]batchDestinationRef, len(declarations.destinations)),
		destinations:    make(map[batchDestinationRef]acceptanceDestinationRef, len(declarations.destinations)),
		recordIndexes:   make(map[batchRecordCoordinate]int, len(declarations.order)),
		position:        position,
	}
	for index, entry := range declarations.order {
		if entry.kind == declarationRecordInvalid || entry.familyIndex < 0 {
			panic("cli: validated provider publication record is invalid")
		}
		coordinate := batchRecordCoordinate{
			kind:        entry.kind,
			familyIndex: entry.familyIndex,
		}
		if _, duplicate := state.recordIndexes[coordinate]; duplicate {
			panic("cli: validated provider publication record is ambiguous")
		}
		state.recordIndexes[coordinate] = index
		if entry.destination.valid() {
			state.destinationRefs[index] = entry.destination
		}
	}
	return state
}

func (state *providerPublicationState) mustRecordIndex(kind declarationRecordKind, familyIndex int) int {
	if state == nil || kind == declarationRecordInvalid || familyIndex < 0 {
		panic("cli: invalid provider publication record coordinate")
	}
	index, ok := state.recordIndexes[batchRecordCoordinate{
		kind:        kind,
		familyIndex: familyIndex,
	}]
	if !ok {
		panic("cli: validated provider publication record cannot resolve")
	}
	return index
}

func (state *providerPublicationState) mustSlot(index int) *declarationAcceptanceSlot {
	if state == nil || index < 0 || index >= len(state.slots) {
		panic("cli: validated provider publication coordinate cannot resolve")
	}
	return &state.slots[index]
}

func (state *providerPublicationState) mustCommand(index int) acceptanceCommandRef {
	if state == nil {
		panic("cli: nil provider publication state")
	}
	if index == -1 {
		if !state.root.ID().present() {
			panic("cli: validated provider root command is unavailable")
		}
		return state.root
	}
	command := state.mustSlot(index).command
	if !command.ID().present() {
		panic("cli: validated provider command coordinate cannot resolve")
	}
	return command
}

func (state *providerPublicationState) mustDestination(index int) acceptanceDestinationRef {
	destination := state.mustSlot(index).destination
	if !destination.ID().present() {
		panic("cli: validated provider destination coordinate cannot resolve")
	}
	return destination
}

func (state *providerPublicationState) batchValueID(target batchDestinationTarget) (ValueID, bool) {
	if state == nil {
		return ValueID(0), false
	}
	switch target := target.(type) {
	case localDestinationTarget:
		ref, ok := state.destinations[target.destination]
		if !ok || !ref.ID().present() {
			return ValueID(0), false
		}
		return ref.ID(), true
	case acceptedDestinationIdentity:
		return target.id, target.id.present() && target.scope.valid()
	default:
		return ValueID(0), false
	}
}

func (state *providerPublicationState) ValueID(identity destinationIdentity) (ValueID, bool) {
	if state == nil {
		return ValueID(0), false
	}
	switch identity := identity.(type) {
	case localDestinationTarget:
		ref, ok := state.destinations[identity.destination]
		if !ok || !ref.ID().present() {
			return ValueID(0), false
		}
		return ref.ID(), true
	case acceptedDestinationIdentity:
		return identity.id, identity.id.present() && identity.scope.valid()
	default:
		return ValueID(0), false
	}
}

func (state *providerPublicationState) acceptanceDestinationRef(draft *declarationCandidate, identity destinationIdentity) (acceptanceDestinationRef, bool) {
	if state == nil || draft == nil {
		return acceptanceDestinationRef{}, false
	}
	switch identity := identity.(type) {
	case localDestinationTarget:
		ref, ok := state.destinations[identity.destination]
		return ref, ok && ref.ID().present()
	case acceptedDestinationIdentity:
		if !identity.id.present() || !identity.scope.valid() {
			return acceptanceDestinationRef{}, false
		}
		ref := draft.DestinationByID(identity.id)
		return ref, ref.ID().present()
	default:
		return acceptanceDestinationRef{}, false
	}
}

func (state *providerPublicationState) mustProfile(index int) grammarProfileRef {
	profile := state.mustSlot(index).profile
	if !profile.ID().present() {
		panic("cli: validated provider profile coordinate cannot resolve")
	}
	return profile
}

func (state *providerPublicationState) profileForIndex(index int) (grammarProfileRef, bool) {
	if index < 0 {
		return grammarProfileRef{}, false
	}
	return state.mustProfile(index), true
}

func (state *providerPublicationState) mustDialectForIndexes(commandIndex, profileIndex int) dialectInput {
	if state == nil {
		panic("cli: nil provider publication state")
	}
	var dialect dialectInput
	switch {
	case profileIndex >= 0:
		dialect = state.mustSlot(profileIndex).dialect
	case commandIndex == -1:
		dialect = state.rootDialect
	default:
		dialect = state.mustSlot(commandIndex).dialect
	}
	if !validSemanticDialectInput(dialect) {
		panic("cli: validated provider dialect coordinate cannot resolve")
	}
	return dialect
}

type providerPublicationCoordinates struct {
	commands          []CommandID
	commandEntries    [][]CommandEntryID
	batchCommands     map[batchCommandRef]CommandID
	batchDestinations map[batchDestinationRef]ValueID
	batchFlags        map[batchFlagRef]FlagID
	batchArguments    map[batchArgumentTermRef]ArgumentTermID
	batchProviders    map[batchProviderRef]CommandProviderID
}

func (coordinates providerPublicationCoordinates) command(index int) (CommandID, bool) {
	if index < 0 || index >= len(coordinates.commands) || !coordinates.commands[index].present() {
		return CommandID(0), false
	}
	return coordinates.commands[index], true
}

func (coordinates providerPublicationCoordinates) mustBatchCommand(ref batchCommandRef) CommandID {
	id, ok := coordinates.batchCommands[ref]
	if !ok || !id.present() {
		panic("cli: provider publication command coordinate cannot resolve")
	}
	return id
}

func (coordinates providerPublicationCoordinates) mustBatchDestination(ref batchDestinationRef) ValueID {
	id, ok := coordinates.batchDestinations[ref]
	if !ok || !id.present() {
		panic("cli: provider publication destination coordinate cannot resolve")
	}
	return id
}

func (coordinates providerPublicationCoordinates) mustBatchFlag(ref batchFlagRef) FlagID {
	id, ok := coordinates.batchFlags[ref]
	if !ok || !id.present() {
		panic("cli: provider publication flag coordinate cannot resolve")
	}
	return id
}

func (coordinates providerPublicationCoordinates) mustBatchArgument(ref batchArgumentTermRef) ArgumentTermID {
	id, ok := coordinates.batchArguments[ref]
	if !ok || !id.present() {
		panic("cli: provider publication argument coordinate cannot resolve")
	}
	return id
}

func (coordinates providerPublicationCoordinates) mustBatchProvider(ref batchProviderRef) CommandProviderID {
	id, ok := coordinates.batchProviders[ref]
	if !ok || !id.present() {
		panic("cli: provider publication provider coordinate cannot resolve")
	}
	return id
}
