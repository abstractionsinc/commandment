package cli

import (
	"errors"
	"maps"
)

type declarationAcceptanceSlot struct {
	command        acceptanceCommandRef
	commandEntries []commandEntryRef
	shortcutEntry  commandEntryRef
	defaultRef     defaultRef
	destination    acceptanceDestinationRef
	flag           acceptanceFlagRef
	spelling       acceptanceFlagSpellingRef
	environment    environmentInputRef
	argumentForm   argumentFormRef
	argumentTerm   argumentTermRef
	profile        grammarProfileRef
	provider       providerRef
	rule           ruleRef
	advisory       advisoryRef
	dialect        dialectInput
	spellingConfig spellingAcceptanceConfig
	helpGroupKey   helpGroupAcceptanceKey
	helpGroupSet   bool
}

type staticAcceptanceState struct {
	staticIndexes   declarationPlanIndex
	slots           []declarationAcceptanceSlot
	root            acceptanceCommandRef
	rootDialect     dialectInput
	domains         map[*valueDomainState]valueDomainRef
	destinationRefs map[int]destinationRef
	destinations    map[destinationRef]acceptanceDestinationRef
	actionEntries   []ActionEntryID
}

func prepareStaticAcceptanceState(input declarationPlan, root acceptanceCommandRef) *staticAcceptanceState {
	refs := make(map[int]destinationRef, len(input.destinationRefs))
	maps.Copy(refs, input.destinationRefs)
	state := &staticAcceptanceState{
		staticIndexes:   input.indexes,
		slots:           make([]declarationAcceptanceSlot, len(input.records)),
		root:            root,
		domains:         make(map[*valueDomainState]valueDomainRef),
		destinationRefs: refs,
		destinations:    make(map[destinationRef]acceptanceDestinationRef, len(refs)),
	}
	return state
}

func (state *staticAcceptanceState) slot(index int) (*declarationAcceptanceSlot, bool) {
	if state == nil || index < 0 || index >= len(state.slots) {
		return nil, false
	}
	return &state.slots[index], true
}

func (state *staticAcceptanceState) mustSlot(index int) *declarationAcceptanceSlot {
	slot, ok := state.slot(index)
	if !ok {
		panic("cli: static acceptance slot is unavailable")
	}
	return slot
}

func (state *staticAcceptanceState) mustCommand(index int) acceptanceCommandRef {
	command, ok := state.command(index)
	if !ok {
		panic("cli: static acceptance command is unavailable")
	}
	return command
}

func (state *staticAcceptanceState) mustDestination(index int) acceptanceDestinationRef {
	destination, ok := state.destination(index)
	if !ok {
		panic("cli: static acceptance destination is unavailable")
	}
	return destination
}

func (state *staticAcceptanceState) mustProfile(index int) grammarProfileRef {
	profile, ok := state.profile(index)
	if !ok {
		panic("cli: static acceptance grammar profile is unavailable")
	}
	return profile
}

func (state *staticAcceptanceState) mustDialectForIndexes(commandIndex, profileIndex int) dialectInput {
	dialect, ok := state.dialectForIndexes(commandIndex, profileIndex)
	if !ok {
		panic("cli: static acceptance dialect is unavailable")
	}
	return dialect
}

func (state *staticAcceptanceState) mustAcceptanceDestinationRef(draft *declarationCandidate, identity destinationIdentity) acceptanceDestinationRef {
	destination, ok := state.acceptanceDestinationRef(draft, identity)
	if !ok {
		panic("cli: static acceptance destination identity is unavailable")
	}
	return destination
}

func (state *staticAcceptanceState) mustValueID(identity destinationIdentity) ValueID {
	id, ok := state.ValueID(identity)
	if !ok {
		panic("cli: static acceptance destination identity is unavailable")
	}
	return id
}

func (state *staticAcceptanceState) authoredCommandIndex(ref commandRef) (int, bool) {
	if state == nil || !ref.valid() {
		return 0, false
	}
	index, ok := state.staticIndexes.commands[ref]
	return index, ok
}

func (state *staticAcceptanceState) command(index int) (acceptanceCommandRef, bool) {
	if state == nil {
		return acceptanceCommandRef{}, false
	}
	if index == -1 {
		return state.root, state.root.ID().present()
	}
	slot, ok := state.slot(index)
	if !ok {
		return acceptanceCommandRef{}, false
	}
	return slot.command, slot.command.ID().present()
}

func (state *staticAcceptanceState) destination(index int) (acceptanceDestinationRef, bool) {
	slot, ok := state.slot(index)
	if !ok {
		return acceptanceDestinationRef{}, false
	}
	return slot.destination, slot.destination.ID().present()
}

func (state *staticAcceptanceState) ValueID(identity destinationIdentity) (ValueID, bool) {
	if state == nil {
		return ValueID(0), false
	}
	switch identity := identity.(type) {
	case stagedDestinationIdentity:
		ref, ok := state.destinations[identity.ref]
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

func (state *staticAcceptanceState) acceptanceDestinationRef(draft *declarationCandidate, identity destinationIdentity) (acceptanceDestinationRef, bool) {
	if state == nil || draft == nil {
		return acceptanceDestinationRef{}, false
	}
	switch identity := identity.(type) {
	case stagedDestinationIdentity:
		ref, ok := state.destinations[identity.ref]
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

func (state *staticAcceptanceState) profile(index int) (grammarProfileRef, bool) {
	slot, ok := state.slot(index)
	if !ok {
		return grammarProfileRef{}, false
	}
	return slot.profile, slot.profile.ID().present()
}

func (state *staticAcceptanceState) dialectForIndexes(commandIndex, profileIndex int) (dialectInput, bool) {
	if state == nil {
		return dialectInput{}, false
	}
	if profileIndex >= 0 {
		slot, ok := state.slot(profileIndex)
		if !ok {
			return dialectInput{}, false
		}
		return slot.dialect, validSemanticDialectInput(slot.dialect)
	}
	if commandIndex == -1 {
		return state.rootDialect, validSemanticDialectInput(state.rootDialect)
	}
	slot, ok := state.slot(commandIndex)
	if !ok {
		return dialectInput{}, false
	}
	return slot.dialect, validSemanticDialectInput(slot.dialect)
}

func (state *staticAcceptanceState) profileForIndex(profileIndex int) (grammarProfileRef, bool) {
	if profileIndex < 0 {
		return grammarProfileRef{}, false
	}
	return state.profile(profileIndex)
}

func stageStaticDeclarations(input declarationPlan, source *declarationDraft, metadata applicationMetadata, rootInitial dialectInput, draft *declarationCandidate, state *staticAcceptanceState, rootExecution commandExecution) error {
	if draft == nil || state == nil || !state.root.ID().present() || len(state.slots) != len(input.records) {
		panic("cli: invalid static acceptance state")
	}
	if err := stageStaticCommandIdentity(input, source, draft, state); err != nil {
		return err
	}
	rootDialect, err := stageStaticGrammarProfiles(input, source, rootInitial, draft, state)
	if err != nil {
		return err
	}

	for index := range len(input.records) {
		record := input.records[index]
		position := input.positionFor(index)
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
			destinationValue, valid := staticDestinationAcceptanceInput(source.destinations.entries[record.familyIndex].value)
			if !valid {
				return errStaticDeclarationRejected
			}
			destination, err := establishStaticDestination(input, metadata, draft, state, index, destinationValue)
			if err != nil {
				return err
			}
			if err := prepareIntrinsicDestinationProducer(draft, state, destinationValue, destination, position); err != nil {
				return err
			}
		case declarationRecordFlag:
			flagValue, valid := input.flagInput(source, record)
			if !valid {
				return errStaticDeclarationRejected
			}
			destination := state.mustDestination(flagValue.destinationIndex)
			if input.records[flagValue.destinationIndex].commandIndex != record.commandIndex {
				panic("cli: validated static flag destination belongs to another command")
			}
			flag := draft.Flag(owner, destination, position)
			state.slots[index].flag = flag
			if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
				draft.FlagProfile(flag, profile, position)
			}
			draft.FlagLifecycle(flag, flagReleaseLifecycle(flagValue), position)
			flagPresentation, valueName, required, localeSelector := flagPresentation(flagValue)
			draft.FlagPresentation(flag, flagPresentation, valueName, required, localeSelector, position)
			if flagValue.completionSet {
				completion, callback, valid := prepareAuthoringCompletion(flagValue.completion, state)
				if !valid {
					return errStaticDeclarationRejected
				}
				draft.FlagCompletion(flag, completion, callback, position)
			}
			destinationValue, valid := staticDestinationAcceptanceInput(source.destinations.entries[input.records[flagValue.destinationIndex].familyIndex].value)
			if !valid {
				return errStaticDeclarationRejected
			}
			grammar, valid := prepareFlagGrammar(flagValue, destinationValue, dialect)
			if !valid {
				return errStaticDeclarationRejected
			}
			draft.FlagGrammar(flag, grammar)
		case declarationRecordFlagSpelling:
			spellingValue := source.flagSpellings.entries[record.familyIndex].value
			flagIndex, indexed := input.indexes.flags[spellingValue.flag]
			if !indexed {
				panic("cli: validated static flag spelling has no flag record")
			}
			flag := state.mustSlot(flagIndex).flag
			if !flag.ID().present() || input.records[flagIndex].commandIndex != record.commandIndex {
				panic("cli: validated static flag spelling has unavailable flag identity")
			}
			spelling := draft.FlagSpelling(flag, position)
			state.slots[index].spelling = spelling
			flagValue, valid := input.flagInput(source, input.records[flagIndex])
			if !valid {
				return errStaticDeclarationRejected
			}
			producer := flagValue.producer
			mode, forms, sensitive, valid := prepareFlagSpellingGrammar(
				spellingValue.kind,
				producer,
				state.slots[index].spellingConfig,
			)
			if !valid {
				return errStaticDeclarationRejected
			}
			kind, valid := prepareFlagSpellingKind(spellingValue.kind)
			if !valid {
				return errStaticDeclarationRejected
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
				panic("cli: accepted static destination definition is unavailable")
			}
			if state.slots[index].spellingConfig.fixed {
				if !validEffectiveDomainContribution(definition, state.slots[index].spellingConfig.fixedValue) {
					return errStaticDeclarationRejected
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
					return errStaticDeclarationRejected
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
			directive := source.flagSpellingDirectives.entries[record.familyIndex].value
			spellingIndex, indexed := input.indexes.flagSpellings[directive.spelling]
			if !indexed {
				panic("cli: validated flag-spelling directive has no spelling record")
			}
			spellingSlot := state.mustSlot(spellingIndex)
			if !spellingSlot.spelling.ID().present() || input.records[spellingIndex].commandIndex != record.commandIndex {
				panic("cli: validated flag-spelling directive has unavailable spelling identity")
			}
		case declarationRecordArgumentForm:
			if !source.argumentForms.entries[record.familyIndex].value.valid {
				return errStaticDeclarationRejected
			}
			form := draft.ArgumentForm(owner, position)
			state.slots[index].argumentForm = form
			if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
				draft.ArgumentFormProfile(form, profile, position)
			}
			draft.ArgumentFormGrammar(form)
		case declarationRecordArgumentTerm:
			termValue, valid := input.argumentTermInput(source, record)
			if !valid {
				return errStaticDeclarationRejected
			}
			form := state.mustSlot(termValue.formIndex).argumentForm
			if !form.ID().present() || input.records[termValue.formIndex].commandIndex != record.commandIndex {
				panic("cli: validated static argument term has unavailable form identity")
			}
			var term argumentTermRef
			if termValue.kind == argumentTermLiteral {
				term = draft.ArgumentTerm(form, position)
			} else {
				destination := state.mustDestination(termValue.destinationIndex)
				if input.records[termValue.destinationIndex].commandIndex != record.commandIndex {
					panic("cli: validated static argument destination belongs to another command")
				}
				term = draft.ArgumentValueTerm(form, destination, position)
			}
			state.slots[index].argumentTerm = term
			lifecycle, present := argumentReleaseLifecycle(termValue)
			if present {
				draft.ArgumentTermLifecycle(term, lifecycle, position)
			}
			draft.ArgumentTermPresentation(term, argumentPresentation(termValue), termValue.name, argumentSourceSensitive(termValue), position)
			if termValue.completionSet {
				completion, callback, valid := prepareAuthoringCompletion(termValue.completion, state)
				if !valid {
					return errStaticDeclarationRejected
				}
				draft.ArgumentTermCompletion(term, completion, callback, position)
			}
			destinationValue := destinationAcceptanceInput{}
			if termValue.destinationIndex >= 0 {
				var valid bool
				destinationValue, valid = staticDestinationAcceptanceInput(source.destinations.entries[input.records[termValue.destinationIndex].familyIndex].value)
				if !valid {
					return errStaticDeclarationRejected
				}
			}
			if !prepareArgumentTermGrammar(draft, term, termValue, destinationValue) {
				return errStaticDeclarationRejected
			}
		case declarationRecordArgumentFormSelection:
			// Applied after all form and destination identities exist.
		case declarationRecordConditionPolicy:
			// Applied after all declaration and destination identities exist.
		case declarationRecordLifecycleDirective:
			// Applied after all exact declaration identities exist.
		case declarationRecordValueDirective:
			directive := source.valueDirectives.entries[record.familyIndex].value
			destination := state.mustAcceptanceDestinationRef(draft, directive.destination)
			destinationDefinition, defined := draft.existingDestinationDefinitionForRef(destination)
			if !defined || !destinationDefinition.Valid() {
				panic("cli: accepted static destination definition is unavailable")
			}
			switch directive.kind {
			case valueDirectiveEnvironment, valueDirectiveEnvironmentPrefix:
				environmentInput, valid := staticEnvironmentInputAcceptanceInput(directive)
				if !valid {
					return errStaticDeclarationRejected
				}
				syntax, valid := prepareEnvironmentInputSyntax(environmentInput, destinationDefinition)
				if !valid {
					return errStaticDeclarationRejected
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
				policy, valid := prepareResolutionPolicy(directive.policies, destinationDefinition)
				if !valid {
					return errStaticDeclarationRejected
				}
				draft.DestinationResolution(destination, policy, position)
			case valueDirectiveDefault:
				definition, ok := draft.existingDestinationDefinitionForRef(destination)
				if !ok {
					panic("cli: accepted static destination definition is unavailable")
				}
				if !validEffectiveDomainContribution(definition, directive.value) {
					return errStaticDeclarationRejected
				}
				source := snapshotProducerSource(nil, directive.sensitive)
				producer := acceptStaticValueProducer(
					declaredProducerKindDefault,
					directive.value,
					source,
				)
				if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
					producer.profile = profile.ID()
				}
				draft.DestinationProducer(destination, producer, nil, position)
			case valueDirectiveComputedDefault, valueDirectiveOptionalComputedDefault:
				kind := declaredProducerKindComputedDefault
				if directive.kind == valueDirectiveOptionalComputedDefault {
					kind = declaredProducerKindOptionalComputedDefault
				}
				producer, callback, valid := prepareExecutableValueProducer(kind, directive.computation, state)
				if !valid {
					return errStaticDeclarationRejected
				}
				producer.source = producerSourceWithSensitive(producer.source, directive.sensitive)
				if profile, profiled := state.profileForIndex(record.profileIndex); profiled {
					producer.profile = profile.ID()
				}
				draft.DestinationProducer(destination, producer, callback, position)
			default:
				return errStaticDeclarationRejected
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
			panic("cli: validated static declaration record kind is unsupported")
		}
	}

	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordGrammarProfileCase {
			continue
		}
		profileCase := source.grammarProfileCases.entries[record.familyIndex].value
		switchIndex, indexed := input.indexes.grammarProfileSwitches[profileCase.switchRef]
		if !indexed {
			panic("cli: validated grammar profile case has no switch record")
		}
		switchRecord := input.records[switchIndex]
		switchDraft := source.grammarProfileSwitches.entries[switchRecord.familyIndex].value
		selector := state.mustAcceptanceDestinationRef(draft, switchDraft.selector)
		profile := state.mustProfile(index)
		dialect := state.mustSlot(index).dialect
		if !validSemanticDialectInput(dialect) {
			panic("cli: accepted grammar profile has invalid dialect")
		}
		draft.GrammarProfileCase(
			profile,
			selector,
			profileCase.value,
			switchDraft.equal,
			profileCase.defaultCase,
			dialect,
			input.positionFor(index),
		)
	}

	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordArgumentFormSelection {
			continue
		}
		selection := source.argumentFormSelections.entries[record.familyIndex].value
		if !selection.valid || selection.destination == nil {
			panic("cli: validated argument-form selection is invalid")
		}
		formIndex, indexed := input.indexes.argumentForms[selection.form]
		if !indexed {
			panic("cli: validated argument-form selection has no form record")
		}
		form := state.mustSlot(formIndex).argumentForm
		if !form.ID().present() {
			panic("cli: accepted argument form identity is unavailable")
		}
		destination := state.mustAcceptanceDestinationRef(draft, selection.destination)
		definition, ok := draft.existingDestinationDefinitionForRef(destination)
		if !ok {
			panic("cli: accepted static destination definition is unavailable")
		}
		if !validEffectiveDomainContribution(definition, selection.value) {
			return errStaticDeclarationRejected
		}
		draft.ArgumentFormSelection(form, destination, selection.value, input.positionFor(index))
	}

	stageStaticExecution(input, source, draft, state, rootExecution)

	if err := stageStaticConditions(input, source, draft, state); err != nil {
		return err
	}

	if err := stageStaticShortcuts(draft, input, source, state); err != nil {
		return err
	}
	if err := stageStaticProviderDefinitions(draft, input, source, state); err != nil {
		return err
	}

	if err := stageStaticPolicies(draft, input, source, state); err != nil {
		return err
	}
	if err := stageStaticSignals(draft, input, source, state); err != nil {
		return err
	}

	if err := stageStaticHelpGroups(draft, input, source, state); err != nil {
		return err
	}

	if err := stageStaticLifecycles(input, source, draft, state); err != nil {
		return err
	}

	if err := stageStaticReplacements(draft, input, source, state); err != nil {
		return err
	}

	actions := source.actions
	actionDefinition, ok := actions.semanticDefinition()
	if !ok {
		return errStaticDeclarationRejected
	}
	draft.IntrinsicActions(actionDefinition)
	actionIDs, err := prepareIntrinsicActions(
		draft,
		state.root,
		rootDialect,
		actions,
	)
	if err != nil {
		return err
	}
	state.actionEntries = actionIDs

	return nil
}

var errStaticDeclarationRejected = errors.New("cli: static declarations rejected")

type spellingAcceptanceConfig struct {
	forms             []FlagValueForm
	formsSet          bool
	fixed             bool
	implicit          bool
	sensitive         bool
	hidden            bool
	hasRelease        bool
	lifecycle         ReleaseLifecycle
	fixedValue        any
	implicitVal       any
	fixedSensitive    bool
	implicitSensitive bool
}

// rootCommandDialectInitial returns the authoritative base dialect for the root
// command. These values are explicit product semantics; they are not inferred
// from Go zero values, enum declaration order, tests, or private constructors.
func rootCommandDialectInitial() dialectInput {
	input := dialectInput{
		CommandMatching:      NameMatchingExact,
		CommandNormalization: NameNormalizationExact,
		FlagMatching:         NameMatchingExact,
		FlagNormalization:    NameNormalizationExact,
		FlagOrder:            FlagOrderBeforeArguments,
		ShortClustering:      ShortFlagClusteringDisabled,
		DashlessFlags:        DashlessFlagModeDisabled,
		DefaultFlagPlacement: FlagPlacementCommand,
		EndOfOptions:         []string{"--"},
		ResponseFiles:        false,
	}

	return input
}

// staticCommitPreparation is the complete prepared static commit state. Every fallible
// acceptance and proof step finishes before its destination identities publish.
type staticCommitPreparation struct {
	program                staticProgram
	callbacks              declarationCallbacks
	shellCompletion        shellCompletionCatalog
	destinationAcceptances []destinationAcceptance
}

func (acceptance *staticCommitPreparation) publishDestinations() {
	if acceptance == nil {
		panic("cli: nil static commit preparation")
	}
	for _, destination := range acceptance.destinationAcceptances {
		destination.publish()
	}
}

func prepareStaticCommit(input declarationPlan, source *declarationDraft, metadata applicationMetadata, configuration semanticCommandConfiguration, execution commandExecution) (staticCommitPreparation, error) {
	if !configuration.Valid() {
		panic("cli: accepted root command has invalid configuration")
	}
	rootInitial := rootCommandDialectInitial()
	if !validSemanticDialectInput(rootInitial) {
		panic("cli: root command dialect is invalid")
	}
	candidate, identities := prepareStaticCandidate()
	candidate.Application(metadata)
	root := candidate.RootCommand(0, configuration)
	candidate.CommandPresentation(root, snapshotPresentation(metadata.description, metadata.Examples(), false), 0)
	state := prepareStaticAcceptanceState(input, root)
	if err := collectStaticSpellingConfigs(input, source, state); err != nil {
		return staticCommitPreparation{}, err
	}
	if err := stageStaticDeclarations(input, source, metadata, rootInitial, candidate, state, execution); err != nil {
		return staticCommitPreparation{}, err
	}
	compiledShell, ok := acceptedShellCompletion(input, source, state)
	if !ok {
		panic("cli: staged shell-completion identities cannot be accepted")
	}
	if !staticCallbacksValid(candidate) {
		panic("cli: static callback custody is inconsistent")
	}
	callbacks := *candidate.callbacks
	model, err := acceptStaticDeclarations(candidate, identities, acceptAcceptanceDecision(acceptanceBoundaryStatic, nil))
	if err != nil {
		return staticCommitPreparation{}, err
	}
	acceptances := prepareStaticDestinationAcceptances(source, state, model)
	result := staticCommitPreparation{
		program:                model,
		callbacks:              callbacks,
		shellCompletion:        compiledShell,
		destinationAcceptances: acceptances,
	}
	return result, nil
}

func prepareStaticDestinationAcceptances(source *declarationDraft, state *staticAcceptanceState, model staticProgram) []destinationAcceptance {
	if source == nil || state == nil || !model.Valid() {
		panic("cli: invalid static destination publication state")
	}
	pending := make([]destinationAcceptance, 0, len(state.destinationRefs))
	scope := source.scope
	for orderIndex, ref := range state.destinationRefs {
		slot := state.mustSlot(orderIndex)
		if !slot.destination.ID().present() {
			panic("cli: accepted static destination identity is unavailable")
		}
		record, ok := source.destinations.resolve(source, ref)
		if !ok {
			panic("cli: accepted static destination draft cannot resolve")
		}
		id := slot.destination.ID()
		accepted, ok := model.Destination(id)
		if !ok || accepted.meta.id != id {
			panic("cli: accepted static destination cannot resolve")
		}
		definition, ok := accepted.Definition()
		if !ok {
			panic("cli: accepted static destination has no definition")
		}
		if record.intrinsic {
			if record.handle != nil {
				panic("cli: intrinsic destination unexpectedly has a public handle")
			}
			continue
		}
		if record.handle == nil {
			panic("cli: accepted destination has no public handle")
		}
		record.handle.snapshot = definition.Snapshot
		identity := acceptedDestinationIdentity{
			scope: scope,
			id:    id,
		}
		pending = append(pending, prepareDestinationAcceptance(record.handle, identity))
	}
	return pending
}

// acceptanceBoundary identifies the acceptance phase whose authoritative facts are in
// use. The same validators can serve static and provider-batch acceptance
// without pretending their available fact universes are identical.
type acceptanceBoundary uint8

const (
	acceptanceBoundaryInvalid acceptanceBoundary = iota
	acceptanceBoundaryStatic
	acceptanceBoundaryProviderBatch
)

// acceptanceDecision is one immutable shared-acceptance decision. It is implementation
// control state only, not a public compiler report or semantic finding model.
type acceptanceDecision struct {
	boundary   acceptanceBoundary
	rejections []rejection
}

// acceptAcceptanceDecision combines deterministic rejection groups produced by the applicable
// shared acceptance services. No group can make a rejected candidate partly
// accepted.
func acceptAcceptanceDecision(boundary acceptanceBoundary, groups ...[]rejection) acceptanceDecision {
	if boundary != acceptanceBoundaryStatic && boundary != acceptanceBoundaryProviderBatch {
		panic("accept: invalid acceptance boundary")
	}
	var rejections []rejection
	for _, group := range groups {
		rejections = append(rejections, group...)
	}
	return acceptanceDecision{
		boundary:   boundary,
		rejections: normalizeRejections(rejections),
	}
}

func (decision acceptanceDecision) Boundary() acceptanceBoundary { return decision.boundary }

func (decision acceptanceDecision) Accepted() bool {
	return (decision.boundary == acceptanceBoundaryStatic || decision.boundary == acceptanceBoundaryProviderBatch) &&
		len(decision.rejections) == 0
}

func (decision acceptanceDecision) Rejections() []rejection {
	return append([]rejection(nil), decision.rejections...)
}

// publicationAcceptance owns one operation-local declaration publication lifecycle. Its
// concrete type is private; invocation owns the surrounding transaction.
type publicationAcceptance struct {
	draft *publicationDraft
}

func preparePublicationAcceptance(input declarationRecordInput, grammar grammarModel, metadata applicationMetadata, ids operationIDs) *publicationAcceptance {
	return &publicationAcceptance{draft: preparePublicationDraft(input, grammar, metadata, ids)}
}

func (acceptance *publicationAcceptance) Draft() *declarationCandidate {
	if acceptance == nil || acceptance.draft == nil {
		return nil
	}
	return &acceptance.draft.declarationCandidate
}

func (acceptance *publicationAcceptance) Preview(decision acceptanceDecision) (publicationPreview, error) {
	if acceptance == nil || acceptance.draft == nil {
		return publicationPreview{}, errDraftClosed
	}
	return acceptance.draft.preview(decision)
}

func (acceptance *publicationAcceptance) Accept(decision acceptanceDecision) (publicationDelta, error) {
	if acceptance == nil || acceptance.draft == nil {
		return publicationDelta{}, errDraftClosed
	}
	publication, err := acceptance.draft.acceptPublication(decision)
	if err != nil {
		return publicationDelta{}, err
	}
	acceptance.draft = nil
	return publication, nil
}
