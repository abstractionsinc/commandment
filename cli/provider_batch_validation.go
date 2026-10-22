package cli

import (
	"regexp"
	"slices"
)

type providerDialectContext struct {
	command int
	profile int
}

func validProviderBatchDeclarations(draft *providerBatchDraft, input declarationPlan) bool {
	if draft == nil || draft.localization != nil && !draft.localization.Enabled() {
		return false
	}

	spellingConfigs := make(map[flagSpellingRef]spellingAcceptanceConfig)
	dialectContexts := make(map[providerDialectContext]bool)
	profileCases := make(map[grammarProfileSwitchRef][]grammarProfileCaseDraft)

	for _, record := range input.records {
		switch record.kind {
		case declarationRecordCommand:
			value := draft.declarations.commands.entries[record.familyIndex].value
			if value.name == "" || value.parent == nil || value.makeConfig == nil || !validProviderCommandParent(draft, input, record, value.parent) || !validProviderLifecycle(draft.release, value.release) {
				return false
			}
			for _, alias := range value.aliases {
				if alias.Name == "" || !validProviderLifecycle(draft.release, alias.Release) {
					return false
				}
			}

		case declarationRecordDestination:
			value := draft.declarations.destinations.entries[record.familyIndex].value
			if value.handle == nil || !value.binding.Valid() {
				return false
			}
			destination, ok := staticDestinationAcceptanceInput(value)
			if !ok {
				return false
			}
			if _, ok := prepareDestinationDefinitionForRelease(destination, draft.release); !ok {
				return false
			}
			if !validProviderDomainUses(input, draft.scope, draft.release, record.commandIndex, value.domainUses) {
				return false
			}
			if !validProviderComputationDependencies(&draft.declarations, input, draft.scope, record.commandIndex, value.computation) {
				return false
			}

		case declarationRecordValueDirective:
			value := draft.declarations.valueDirectives.entries[record.familyIndex].value
			if !value.valid || !validProviderDestinationVisibility(input, draft.scope, value.destination, record.commandIndex) {
				return false
			}
			if !validProviderComputationDependencies(&draft.declarations, input, draft.scope, record.commandIndex, value.computation) {
				return false
			}
			definition, local, ok := providerLocalDestinationDefinition(draft, value.destination)
			if !ok || local && !validProviderValueDirective(value, definition) {
				return false
			}

		case declarationRecordDialectDirective:
			value := draft.declarations.dialectDirectives.entries[record.familyIndex].value
			directive, ok := staticDialectDirectiveDefinition(value)
			if !ok {
				return false
			}
			context := providerDialectContext{
				command: record.commandIndex,
				profile: record.profileIndex,
			}
			_, configured, ok := applyDialectDirective(rootCommandDialectInitial(), directive, dialectContexts[context])
			if !ok {
				return false
			}
			dialectContexts[context] = configured

		case declarationRecordFlag:
			value := draft.declarations.flags.entries[record.familyIndex].value
			if value.releaseSet && !validProviderLifecycle(draft.release, value.release) {
				return false
			}
			destinationIndex, ok := input.indexes.destinations[value.destination]
			if !ok || input.records[destinationIndex].commandIndex != record.commandIndex {
				return false
			}
			flag, ok := input.flagInput(&draft.declarations, record)
			if !ok {
				return false
			}
			destination, ok := staticDestinationAcceptanceInput(draft.declarations.destinations.entries[input.records[destinationIndex].familyIndex].value)
			if !ok {
				return false
			}
			if _, ok := prepareFlagGrammar(flag, destination, rootCommandDialectInitial()); !ok {
				return false
			}
			if !validProviderCompletionDependencies(&draft.declarations, input, draft.scope, record.commandIndex, flag.completion, flag.completionSet) {
				return false
			}

		case declarationRecordFlagSpelling:
			value := draft.declarations.flagSpellings.entries[record.familyIndex].value
			flagIndex, ok := input.indexes.flags[value.flag]
			if !ok || input.records[flagIndex].commandIndex != record.commandIndex || !validProviderFlagSpelling(value) {
				return false
			}

		case declarationRecordFlagSpellingDirective:
			value := draft.declarations.flagSpellingDirectives.entries[record.familyIndex].value
			if value.kind == flagSpellingDirectiveRelease && !validProviderLifecycle(draft.release, value.lifecycle) {
				return false
			}
			directive, ok := staticSpellingDirectiveAcceptanceInput(value)
			if !ok {
				return false
			}
			config, ok := applySpellingDirective(spellingConfigs[value.spelling], directive)
			if !ok {
				return false
			}
			spellingConfigs[value.spelling] = config

		case declarationRecordArgumentForm:
			if !draft.declarations.argumentForms.entries[record.familyIndex].value.valid {
				return false
			}

		case declarationRecordArgumentTerm:
			value := draft.declarations.argumentTerms.entries[record.familyIndex].value
			if value.releaseSet && !validProviderLifecycle(draft.release, value.release) {
				return false
			}
			term, ok := input.argumentTermInput(&draft.declarations, record)
			if !ok {
				return false
			}
			formIndex, ok := input.indexes.argumentForms[value.form]
			if !ok || input.records[formIndex].commandIndex != record.commandIndex {
				return false
			}
			if value.destination.valid() {
				destinationIndex, ok := input.indexes.destinations[value.destination]
				if !ok || input.records[destinationIndex].commandIndex != record.commandIndex {
					return false
				}
			}
			if !validProviderCompletionDependencies(&draft.declarations, input, draft.scope, record.commandIndex, term.completion, term.completionSet) {
				return false
			}

		case declarationRecordArgumentFormSelection:
			value := draft.declarations.argumentFormSelections.entries[record.familyIndex].value
			if !value.valid {
				return false
			}
			formIndex, ok := input.indexes.argumentForms[value.form]
			if !ok || input.records[formIndex].commandIndex != record.commandIndex || !validProviderDestinationVisibility(input, draft.scope, value.destination, record.commandIndex) {
				return false
			}
			definition, local, ok := providerLocalDestinationDefinition(draft, value.destination)
			if !ok || local && !validEffectiveDomainContribution(definition, value.value) {
				return false
			}

		case declarationRecordConditionPolicy:
			value := draft.declarations.conditionPolicies.entries[record.familyIndex].value
			if !validProviderConditionPolicy(input, draft.scope, record, value) {
				return false
			}

		case declarationRecordLifecycleDirective:
			value := draft.declarations.lifecycleDirectives.entries[record.familyIndex].value
			if !validProviderLifecycle(draft.release, value.lifecycle) || !validProviderLifecycleDirective(input, record, value) {
				return false
			}

		case declarationRecordGrammarProfileSwitch:
			value := draft.declarations.grammarProfileSwitches.entries[record.familyIndex].value
			if !value.valid || value.equal == nil || !validProviderDestinationVisibility(input, draft.scope, value.selector, record.commandIndex) {
				return false
			}

		case declarationRecordGrammarProfileCase:
			value := draft.declarations.grammarProfileCases.entries[record.familyIndex].value
			if !value.valid {
				return false
			}
			profileCases[value.switchRef] = append(profileCases[value.switchRef], value)

		case declarationRecordRule:
			value := draft.declarations.rules.entries[record.familyIndex].value
			if !validRuleDraft(value, record.commandIndex, record.profileIndex) || !validProviderRuleDestinations(input, draft.scope, record.commandIndex, value) || !validProviderRuleCondition(input, draft.scope, record.commandIndex, value.activation) {
				return false
			}

		case declarationRecordRuleDirective:
			value := draft.declarations.ruleDirectives.entries[record.familyIndex].value
			if !validProviderRuleDirective(input, draft.scope, record, value) {
				return false
			}

		case declarationRecordAdvisory:
			value := draft.declarations.advisories.entries[record.familyIndex].value
			if !validAdvisoryDraft(value, record.commandIndex, record.profileIndex) || !validProviderCondition(input, draft.scope, record.commandIndex, value.condition.state) {
				return false
			}

		case declarationRecordAdvisoryDirective:
			value := draft.declarations.advisoryDirectives.entries[record.familyIndex].value
			if !validProviderAdvisoryDirective(input, draft.scope, record, value) {
				return false
			}

		case declarationRecordNoticeSuppression:
			value := draft.declarations.noticeSuppressions.entries[record.familyIndex].value
			if !value.valid || !validNoticeMatch(value.match) || !validProviderCondition(input, draft.scope, record.commandIndex, value.condition.state) {
				return false
			}

		case declarationRecordShortcut:
			value := draft.declarations.shortcuts.entries[record.familyIndex].value
			if !validProviderShortcut(value) || !validProviderLifecycle(draft.release, value.release) {
				return false
			}

		case declarationRecordShortcutDirective:
			value := draft.declarations.shortcutDirectives.entries[record.familyIndex].value
			if !validProviderShortcutDirective(&draft.declarations, input, draft.scope, record, value) {
				return false
			}

		case declarationRecordProvider:
			value := draft.declarations.providers.entries[record.familyIndex].value
			if !value.valid || !validProviderDependencies(input, draft.scope, record.commandIndex, value.dependencies) {
				return false
			}

		case declarationRecordSignalClear:
			if !draft.declarations.signalClears.entries[record.familyIndex].value.valid {
				return false
			}

		case declarationRecordSignalPolicy:
			value := draft.declarations.signalPolicies.entries[record.familyIndex].value
			if !validSignalPolicyDraft(&draft.declarations, value) || !validProviderSignalRelations(input, draft.scope, record.commandIndex, value) {
				return false
			}

		case declarationRecordHelpGroup:
			value := draft.declarations.helpGroups.entries[record.familyIndex].value
			if !validProviderHelpGroup(value) {
				return false
			}

		case declarationRecordReplacement:
			value := draft.declarations.replacements.entries[record.familyIndex].value
			if !validProviderReplacement(&draft.declarations, input, value) {
				return false
			}

		case declarationRecordAccess:
			value := draft.declarations.accesses.entries[record.familyIndex].value
			if !validProviderAccess(input, draft.scope, record.commandIndex, value) {
				return false
			}

		case declarationRecordInvalid:
			return false
		}
	}

	for _, record := range input.records {
		if record.kind != declarationRecordFlagSpelling {
			continue
		}
		spelling := draft.declarations.flagSpellings.entries[record.familyIndex].value
		flagIndex, ok := input.indexes.flags[spelling.flag]
		if !ok {
			return false
		}
		flag := draft.declarations.flags.entries[input.records[flagIndex].familyIndex].value
		config := spellingConfigs[declarationRefAt(&draft.declarations, &draft.declarations.flagSpellings, record.familyIndex)]
		if _, _, _, ok := prepareFlagSpellingGrammar(spelling.kind, flag.producer, config); !ok {
			return false
		}
		if config.fixed || config.implicit {
			destinationIndex, ok := input.indexes.destinations[flag.destination]
			if !ok {
				return false
			}
			destination, ok := staticDestinationAcceptanceInput(draft.declarations.destinations.entries[input.records[destinationIndex].familyIndex].value)
			if !ok {
				return false
			}
			definition, ok := prepareDestinationDefinitionForRelease(destination, draft.release)
			if !ok || config.fixed && !validEffectiveDomainContribution(definition, config.fixedValue) || config.implicit && !validEffectiveDomainContribution(definition, config.implicitVal) {
				return false
			}
		}
	}

	return validProviderGrammarProfileCases(draft, profileCases) &&
		validProviderHelpGroups(&draft.declarations, input) &&
		validProviderShortcutDeclarations(&draft.declarations, input) &&
		validProviderReplacementGraph(&draft.declarations, input)
}

func validProviderValueDomainState(state *valueDomainState, release ReleaseScheme) bool {
	if !validValueDomainState(state) {
		return false
	}
	if state.dynamic != nil {
		return true
	}

	effective := make([]bool, len(state.entries))
	for index, entry := range state.entries {
		if err := validateLifecycle(release, entry.lifecycle); err != nil {
			return false
		}
		LifecycleState, err := evaluateLifecycle(release, entry.lifecycle, "")
		if err != nil {
			return false
		}
		effective[index] = LifecycleState == LifecycleStateCurrent || LifecycleState == LifecycleStateDeprecated

		switch entry.kind {
		case domainDeclarationValue:
		case domainDeclarationAlias:
			if entry.target < 0 || entry.target >= index || !effective[entry.target] || state.entries[entry.target].kind != domainDeclarationValue {
				return false
			}
		default:
			return false
		}
	}

	for index, entry := range state.entries {
		if entry.replacement < 0 {
			continue
		}
		if entry.replacement >= len(state.entries) || entry.replacement == index ||
			state.entries[entry.replacement].kind != domainDeclarationValue {
			return false
		}
	}

	return true
}

func validProviderLifecycle(scheme ReleaseScheme, lifecycle ReleaseLifecycle) bool {
	return validateLifecycle(scheme, lifecycle) == nil
}

func providerLocalDestinationDefinition(draft *providerBatchDraft, identity destinationIdentity) (destinationDefinition, bool, bool) {
	if draft == nil {
		return destinationDefinition{}, false, false
	}
	switch identity := identity.(type) {
	case stagedDestinationIdentity:
		destination, ok := draft.declarations.destinations.resolve(&draft.declarations, identity.ref)
		if !ok {
			return destinationDefinition{}, true, false
		}
		input, ok := staticDestinationAcceptanceInput(*destination)
		if !ok {
			return destinationDefinition{}, true, false
		}
		definition, ok := prepareDestinationDefinitionForRelease(input, draft.release)
		return definition, true, ok
	case acceptedDestinationIdentity:
		return destinationDefinition{}, false, identity.id.present() && draft.scope.includes(identity.scope)
	default:
		return destinationDefinition{}, false, false
	}
}

func validProviderValueDirective(value valueDirectiveDraft, definition destinationDefinition) bool {
	switch value.kind {
	case valueDirectiveEnvironment, valueDirectiveEnvironmentPrefix:
		input, ok := staticEnvironmentInputAcceptanceInput(value)
		if !ok {
			return false
		}
		_, ok = prepareEnvironmentInputSyntax(input, definition)
		return ok
	case valueDirectiveResolution:
		_, ok := prepareResolutionPolicy(value.policies, definition)
		return ok
	case valueDirectiveDefault:
		return validEffectiveDomainContribution(definition, value.value)
	case valueDirectiveComputedDefault, valueDirectiveOptionalComputedDefault:
		return true
	default:
		return false
	}
}

func validProviderCommandParent(draft *providerBatchDraft, input declarationPlan, record declarationPlanRecord, parent commandParent) bool {
	switch parent := parent.(type) {
	case stagedCommandParent:
		index, ok := input.indexes.commands[parent.command]
		return ok && index == record.commandIndex
	case acceptedCommandParent:
		return record.commandIndex == -1 && parent.command == draft.parent
	default:
		return false
	}
}

func validProviderRuleDirective(input declarationPlan, scope identityScope, record declarationPlanRecord, value ruleDirectiveDraft) bool {
	if !value.valid || !validProviderDestinationVisibility(input, scope, value.destination, record.commandIndex) {
		return false
	}
	ruleIndex, ok := input.indexes.rules[value.rule]
	if !ok || ruleIndex < 0 || ruleIndex >= len(input.records) {
		return false
	}
	rule := input.records[ruleIndex]
	if rule.kind != declarationRecordRule || rule.commandIndex != record.commandIndex || rule.profileIndex != record.profileIndex {
		return false
	}
	switch value.kind {
	case ruleDirectiveSubject:
		return !value.optional
	case ruleDirectiveDependency, ruleDirectiveGroupDependency:
		return true
	default:
		return false
	}
}

func validProviderAdvisoryDirective(input declarationPlan, scope identityScope, record declarationPlanRecord, value advisoryDirectiveDraft) bool {
	if !value.valid || !validProviderDestinationVisibility(input, scope, value.destination, record.commandIndex) {
		return false
	}
	advisoryIndex, ok := input.indexes.advisories[value.advisory]
	if !ok || advisoryIndex < 0 || advisoryIndex >= len(input.records) {
		return false
	}
	advisory := input.records[advisoryIndex]
	if advisory.kind != declarationRecordAdvisory || advisory.commandIndex != record.commandIndex || advisory.profileIndex != record.profileIndex {
		return false
	}
	switch value.kind {
	case advisoryDirectiveSubject:
		return !value.optional
	case advisoryDirectiveDependency:
		return true
	default:
		return false
	}
}

func validProviderHelpGroups(draft *declarationDraft, input declarationPlan) bool {
	if draft == nil {
		return false
	}
	type key struct {
		command int
		profile int
		family  helpGroupFamily
		title   string
	}
	orders := make(map[key]HelpGroupOrder)
	for index, record := range input.records {
		if record.kind == declarationRecordHelpGroup {
			value := draft.helpGroups.entries[record.familyIndex].value
			current := key{
				command: record.commandIndex,
				profile: record.profileIndex,
				family:  value.family,
				title:   value.title,
			}
			if order, exists := orders[current]; exists && order != value.order {
				return false
			}
			orders[current] = value.order
		}
		if record.commandHelpGroupIndex >= 0 {
			if !validProviderHelpGroupMembership(draft, input, index, record.commandHelpGroupIndex, helpGroupFamilyCommand) {
				return false
			}
			switch record.kind {
			case declarationRecordCommand, declarationRecordProvider:
			default:
				return false
			}
		}
		if record.flagHelpGroupIndex >= 0 {
			if !validProviderHelpGroupMembership(draft, input, index, record.flagHelpGroupIndex, helpGroupFamilyFlag) {
				return false
			}
			switch record.kind {
			case declarationRecordFlag, declarationRecordFlagSpelling:
			default:
				return false
			}
		}
	}
	return true
}

func validProviderHelpGroupMembership(draft *declarationDraft, input declarationPlan, memberIndex, groupIndex int, family helpGroupFamily) bool {
	if memberIndex < 0 || memberIndex >= len(input.records) || groupIndex < 0 || groupIndex >= memberIndex {
		return false
	}
	member := input.records[memberIndex]
	groupRecord := input.records[groupIndex]
	if groupRecord.kind != declarationRecordHelpGroup || groupRecord.commandIndex != member.commandIndex || groupRecord.profileIndex != member.profileIndex {
		return false
	}
	group := draft.helpGroups.entries[groupRecord.familyIndex].value
	return group.valid && group.family == family
}

func validProviderConditionPolicy(input declarationPlan, scope identityScope, record declarationPlanRecord, value conditionPolicyDraft) bool {
	if !value.valid || value.effect != ConditionEffectAvailability && value.effect != ConditionEffectVisibility || !validProviderCondition(input, scope, record.commandIndex, value.condition.state) {
		return false
	}
	targetIndex, ok := input.conditionTargetIndex(value.target)
	if !ok || targetIndex < 0 || targetIndex >= len(input.records) {
		return false
	}
	target := input.records[targetIndex]
	if target.profileIndex != record.profileIndex {
		return false
	}
	switch value.target.(type) {
	case authoringCommandConditionTarget:
		return target.kind == declarationRecordCommand && targetIndex == record.commandIndex
	case authoringFlagConditionTarget:
		return target.kind == declarationRecordFlag && target.commandIndex == record.commandIndex
	case authoringArgumentFormConditionTarget:
		return target.kind == declarationRecordArgumentForm && target.commandIndex == record.commandIndex
	case authoringArgumentTermConditionTarget:
		return target.kind == declarationRecordArgumentTerm && target.commandIndex == record.commandIndex
	default:
		return false
	}
}

func validProviderLifecycleDirective(input declarationPlan, record declarationPlanRecord, value lifecycleDirectiveDraft) bool {
	if !value.valid {
		return false
	}
	targetIndex, ok := input.lifecycleTargetIndex(value.target)
	if !ok || targetIndex < 0 || targetIndex >= len(input.records) {
		return false
	}
	target := input.records[targetIndex]
	if target.commandIndex != record.commandIndex || target.profileIndex != record.profileIndex {
		return false
	}
	switch value.target.(type) {
	case authoringEnvironmentLifecycleTarget:
		return target.kind == declarationRecordValueDirective
	case authoringArgumentFormLifecycleTarget:
		return target.kind == declarationRecordArgumentForm
	case authoringArgumentTermLifecycleTarget:
		return target.kind == declarationRecordArgumentTerm
	default:
		return false
	}
}

func validProviderShortcut(value shortcutDraft) bool {
	return value.valid && value.name != ""
}

func validProviderShortcutDirective(draft *declarationDraft, input declarationPlan, scope identityScope, record declarationPlanRecord, value shortcutDirectiveDraft) bool {
	if draft == nil || !value.valid {
		return false
	}
	shortcutIndex, ok := input.indexes.shortcuts[value.shortcut]
	if !ok || shortcutIndex < 0 || shortcutIndex >= len(input.records) {
		return false
	}
	shortcut := input.records[shortcutIndex]
	if shortcut.kind != declarationRecordShortcut || shortcut.commandIndex != record.commandIndex || shortcut.profileIndex != record.profileIndex {
		return false
	}
	switch value.kind {
	case commandShortcutDirectiveDefault:
		if value.destination == nil || value.replacement.valid() || !validProviderDestinationVisibility(input, scope, value.destination, record.commandIndex) {
			return false
		}
		if _, ok := producerSource(value.sourceOptions); !ok {
			return false
		}
		switch identity := value.destination.(type) {
		case acceptedDestinationIdentity:
			return identity.id.present() && scope.includes(identity.scope)
		case stagedDestinationIdentity:
			destinationIndex, ok := input.indexes.destinations[identity.ref]
			if !ok || destinationIndex < 0 || destinationIndex >= len(input.records) {
				return false
			}
			destinationRecord := input.records[destinationIndex]
			if destinationRecord.kind != declarationRecordDestination {
				return false
			}
			destination, ok := staticDestinationAcceptanceInput(draft.destinations.entries[destinationRecord.familyIndex].value)
			if !ok {
				return false
			}
			definition, ok := prepareDestinationDefinition(destination)
			return ok && validEffectiveDomainContribution(definition, value.value)
		default:
			return false
		}
	case commandShortcutDirectiveReplacement:
		if value.destination != nil || !value.replacement.valid() || len(value.sourceOptions) != 0 {
			return false
		}
		replacementIndex, ok := input.indexes.shortcuts[value.replacement]
		if !ok || replacementIndex < 0 || replacementIndex >= len(input.records) || replacementIndex == shortcutIndex {
			return false
		}
		replacement := input.records[replacementIndex]
		return replacement.kind == declarationRecordShortcut && replacement.commandIndex == record.commandIndex
	default:
		return false
	}
}

func validProviderShortcutDeclarations(draft *declarationDraft, input declarationPlan) bool {
	if draft == nil {
		return false
	}
	defaultCount := make(map[int]int)
	replacements := make(map[int]int)
	for _, record := range input.records {
		if record.kind != declarationRecordShortcutDirective {
			continue
		}
		value := draft.shortcutDirectives.entries[record.familyIndex].value
		source, ok := input.indexes.shortcuts[value.shortcut]
		if !ok {
			return false
		}
		switch value.kind {
		case commandShortcutDirectiveDefault:
			defaultCount[source]++
		case commandShortcutDirectiveReplacement:
			target, ok := input.indexes.shortcuts[value.replacement]
			if !ok {
				return false
			}
			if _, duplicate := replacements[source]; duplicate {
				return false
			}
			replacements[source] = target
		default:
			return false
		}
	}
	for index, record := range input.records {
		if record.kind != declarationRecordShortcut {
			continue
		}
		value := draft.shortcuts.entries[record.familyIndex].value
		if len(value.args) == 0 && len(value.appendArgs) == 0 && defaultCount[index] == 0 {
			return false
		}
	}
	for start := range replacements {
		seen := make(map[int]struct{})
		for current := start; ; {
			if _, duplicate := seen[current]; duplicate {
				return false
			}
			seen[current] = struct{}{}
			next, ok := replacements[current]
			if !ok {
				break
			}
			current = next
		}
	}
	return true
}

func validProviderHelpGroup(value helpGroupDraft) bool {
	return value.valid && value.family.Valid() && value.title != "" && value.order.valid()
}

type providerReplacementSubject struct {
	key      batchReplacementTarget
	command  int
	profile  int
	flag     int
	termKind argumentTermKind
}

func validProviderReplacement(draft *declarationDraft, input declarationPlan, value replacementDraft) bool {
	if draft == nil || !value.valid {
		return false
	}
	source, sourceOK := resolveProviderReplacementSubject(draft, input, value.source)
	target, targetOK := resolveProviderReplacementSubject(draft, input, value.target)
	if !sourceOK || !targetOK || source.key == target.key || !validProviderReplacementProfile(source.profile, target.profile) {
		return false
	}
	switch source.key.kind {
	case authoringReplacementCommand:
		return target.key.kind == authoringReplacementCommand && source.command == target.command
	case authoringReplacementFlag:
		return source.command == target.command && (target.key.kind == authoringReplacementFlag || target.key.kind == authoringReplacementArgument && target.termKind != argumentTermLiteral && target.termKind != argumentTermInvalid)
	case authoringReplacementFlagSpelling:
		return target.key.kind == authoringReplacementFlagSpelling && source.flag >= 0 && source.flag == target.flag
	case authoringReplacementArgument, authoringReplacementArgumentLiteral:
		if source.termKind == argumentTermLiteral {
			return target.key.kind == authoringReplacementArgumentLiteral && target.termKind == argumentTermLiteral && source.command == target.command
		}
		return source.termKind != argumentTermInvalid && source.command == target.command && (target.key.kind == authoringReplacementFlag || target.key.kind == authoringReplacementArgument && target.termKind != argumentTermLiteral && target.termKind != argumentTermInvalid)
	case authoringReplacementArgumentForm:
		return target.key.kind == authoringReplacementArgumentForm && source.command == target.command
	case authoringReplacementEnvironment:
		return target.key.kind == authoringReplacementEnvironment && source.command == target.command
	default:
		return false
	}
}

func resolveProviderReplacementSubject(draft *declarationDraft, input declarationPlan, target authoringReplacementTarget) (providerReplacementSubject, bool) {
	index, ok := declarationReplacementTargetIndex(target, input.indexes)
	if !ok || index < 0 || index >= len(input.records) {
		return providerReplacementSubject{}, false
	}
	record := input.records[index]
	subject := providerReplacementSubject{
		key: batchReplacementTarget{
			kind:  target.kind,
			index: index,
		},
		command: record.commandIndex,
		profile: record.profileIndex,
		flag:    -1,
	}
	switch target.kind {
	case authoringReplacementCommand:
		return subject, record.kind == declarationRecordCommand
	case authoringReplacementFlag:
		return subject, record.kind == declarationRecordFlag
	case authoringReplacementFlagSpelling:
		if record.kind != declarationRecordFlagSpelling {
			return providerReplacementSubject{}, false
		}
		value := draft.flagSpellings.entries[record.familyIndex].value
		flagIndex, ok := input.indexes.flags[value.flag]
		if !ok {
			return providerReplacementSubject{}, false
		}
		subject.flag = flagIndex
		return subject, true
	case authoringReplacementArgument, authoringReplacementArgumentLiteral:
		if record.kind != declarationRecordArgumentTerm {
			return providerReplacementSubject{}, false
		}
		subject.termKind = draft.argumentTerms.entries[record.familyIndex].value.kind
		if target.kind == authoringReplacementArgumentLiteral {
			return subject, subject.termKind == argumentTermLiteral
		}
		return subject, subject.termKind != argumentTermLiteral && subject.termKind != argumentTermInvalid
	case authoringReplacementEnvironment:
		return subject, record.kind == declarationRecordValueDirective
	case authoringReplacementArgumentForm:
		return subject, record.kind == declarationRecordArgumentForm
	default:
		return providerReplacementSubject{}, false
	}
}

func validProviderReplacementProfile(source, target int) bool {
	if source < 0 {
		return target < 0
	}
	return target < 0 || source == target
}

func validProviderReplacementGraph(draft *declarationDraft, input declarationPlan) bool {
	if draft == nil {
		return false
	}
	edges := make(map[batchReplacementTarget]batchReplacementTarget)
	for _, record := range input.records {
		if record.kind != declarationRecordReplacement {
			continue
		}
		value := draft.replacements.entries[record.familyIndex].value
		source, sourceOK := resolveProviderReplacementSubject(draft, input, value.source)
		target, targetOK := resolveProviderReplacementSubject(draft, input, value.target)
		if !sourceOK || !targetOK {
			return false
		}
		if _, duplicate := edges[source.key]; duplicate {
			return false
		}
		edges[source.key] = target.key
	}
	for start := range edges {
		seen := make(map[batchReplacementTarget]struct{})
		for current := start; ; {
			if _, duplicate := seen[current]; duplicate {
				return false
			}
			seen[current] = struct{}{}
			next, ok := edges[current]
			if !ok {
				break
			}
			current = next
		}
	}
	return true
}

func validProviderFlagSpelling(value flagSpellingDraft) bool {
	if !value.valid {
		return false
	}
	switch value.kind {
	case flagSpellingLong:
		return validLongFlagName(value.text)
	case flagSpellingShort:
		return validShortFlagText(value.text)
	case flagSpellingLiteral:
		return validLiteralFlagSpelling(value.text)
	default:
		return false
	}
}

func validProviderGrammarProfileCases(draft *providerBatchDraft, groups map[grammarProfileSwitchRef][]grammarProfileCaseDraft) bool {
	for ref, cases := range groups {
		switchValue, ok := draft.declarations.grammarProfileSwitches.resolve(&draft.declarations, ref)
		if !ok || !switchValue.valid || switchValue.equal == nil {
			return false
		}
		defaultSeen := false
		for index, current := range cases {
			if !current.valid {
				return false
			}
			if current.defaultCase {
				if defaultSeen {
					return false
				}
				defaultSeen = true
				continue
			}
			for prior := range index {
				if !cases[prior].defaultCase && switchValue.equal(cases[prior].value, current.value) {
					return false
				}
			}
		}
	}
	return true
}

func validProviderComputationDependencies(draft *declarationDraft, input declarationPlan, scope identityScope, commandIndex int, computation authoringValueComputation) bool {
	frozen, ok := freezeAuthoringValueComputation(draft, computation)
	if !ok {
		return false
	}
	return validProviderDependencies(input, scope, commandIndex, frozen.dependencies)
}

func validProviderCompletionDependencies(draft *declarationDraft, input declarationPlan, scope identityScope, commandIndex int, completion authoringCompletion, present bool) bool {
	if !present {
		return true
	}
	frozen, ok := freezeAuthoringCompletion(draft, completion)
	if !ok {
		return false
	}
	if _, ok := producerSource(frozen.options); !ok {
		return false
	}
	return validProviderDependencies(input, scope, commandIndex, frozen.dependencies)
}

func validProviderDependencies(input declarationPlan, scope identityScope, commandIndex int, dependencies []authoringDestinationDependency) bool {
	seen := make(map[destinationIdentity]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		if !validProviderDestinationVisibility(input, scope, dependency.destination, commandIndex) {
			return false
		}
		if _, duplicate := seen[dependency.destination]; duplicate {
			return false
		}
		seen[dependency.destination] = struct{}{}
	}
	return true
}

func validProviderDestinationVisibility(input declarationPlan, scope identityScope, identity destinationIdentity, commandIndex int) bool {
	owner, local, valid := providerDestinationOwner(input, scope, identity)
	if !valid || !local {
		return valid
	}
	return providerCommandSameOrAncestor(input, owner, commandIndex)
}

func providerDestinationOwner(input declarationPlan, scope identityScope, identity destinationIdentity) (int, bool, bool) {
	switch identity := identity.(type) {
	case stagedDestinationIdentity:
		index, ok := input.indexes.destinations[identity.ref]
		if !ok || index < 0 || index >= len(input.records) {
			return 0, true, false
		}
		return input.records[index].commandIndex, true, true
	case acceptedDestinationIdentity:
		return -1, false, identity.id.present() && scope.includes(identity.scope)
	default:
		return 0, false, false
	}
}

func providerCommandSameOrAncestor(input declarationPlan, ancestor, command int) bool {
	if ancestor == -1 {
		return true
	}
	for current := command; current >= 0; {
		if current == ancestor {
			return true
		}
		if current >= len(input.records) || input.records[current].kind != declarationRecordCommand {
			return false
		}
		current = input.records[current].commandIndex
	}
	return false
}

func validProviderDomainUses(input declarationPlan, scope identityScope, release ReleaseScheme, commandIndex int, uses []authoringDomainUse) bool {
	for _, use := range uses {
		if !validProviderValueDomainState(use.state, release) {
			return false
		}
		if dynamic := use.state.dynamic; dynamic != nil {
			if dynamic.evaluate == nil || !validProviderDependencies(input, scope, commandIndex, dynamic.dependencies) {
				return false
			}
			if dynamic.ownerCommand.valid() {
				owner, ok := input.indexes.commands[dynamic.ownerCommand]
				if !ok || !providerCommandSameOrAncestor(input, owner, commandIndex) {
					return false
				}
			}
		}
	}
	return true
}

func validProviderCondition(input declarationPlan, scope identityScope, commandIndex int, state *conditionState) bool {
	if state == nil {
		return true
	}
	if !state.valid {
		return false
	}
	if _, ok := prepareConditionKind(state.kind); !ok {
		return false
	}
	if state.destination != nil && !validProviderDestinationVisibility(input, scope, state.destination, commandIndex) {
		return false
	}
	for _, child := range state.children {
		if !validProviderCondition(input, scope, commandIndex, child) {
			return false
		}
	}
	switch state.kind {
	case conditionResolved:
		return state.destination != nil && len(state.operands) == 0 && state.pattern == "" && len(state.children) == 0
	case conditionEqual, conditionNotEqual, conditionGreaterThan, conditionAtLeast, conditionLessThan, conditionAtMost:
		return state.destination != nil && len(state.operands) == 1 && state.pattern == "" && len(state.children) == 0
	case conditionIn, conditionNotIn:
		return state.destination != nil && state.pattern == "" && len(state.children) == 0
	case conditionMatches:
		if state.destination == nil || len(state.operands) != 0 || len(state.children) != 0 {
			return false
		}
		_, err := regexp.Compile(state.pattern)
		return err == nil
	case conditionAll, conditionAny:
		return state.destination == nil && len(state.operands) == 0 && state.pattern == "" && len(state.children) != 0
	case conditionNot:
		return state.destination == nil && len(state.operands) == 0 && state.pattern == "" && len(state.children) == 1
	default:
		return false
	}
}

func validProviderRuleCondition(input declarationPlan, scope identityScope, commandIndex int, condition RuleCondition) bool {
	state := ruleConditionStateOf(condition)
	return validProviderRuleConditionState(input, scope, commandIndex, state)
}

func validProviderRuleConditionState(input declarationPlan, scope identityScope, commandIndex int, state *ruleConditionState) bool {
	if state == nil {
		return true
	}
	if !state.valid {
		return false
	}
	if state.structural != nil && !validProviderCondition(input, scope, commandIndex, state.structural) {
		return false
	}
	if state.destination != nil && !validProviderDestinationVisibility(input, scope, state.destination, commandIndex) {
		return false
	}
	for _, child := range state.children {
		if !validProviderRuleConditionState(input, scope, commandIndex, child) {
			return false
		}
	}
	return true
}

func validProviderRuleDestinations(input declarationPlan, scope identityScope, commandIndex int, value ruleDraft) bool {
	for _, destination := range value.destinations {
		if !validProviderDestinationVisibility(input, scope, destination, commandIndex) {
			return false
		}
	}
	return true
}

func validProviderSignalRelations(input declarationPlan, scope identityScope, commandIndex int, value signalPolicyDraft) bool {
	for _, clause := range value.match.conditions {
		if !validProviderCondition(input, scope, commandIndex, clause.condition.state) {
			return false
		}
	}
	for _, action := range value.plan.actions {
		if action.signalValue != nil && !validProviderDestinationVisibility(input, scope, action.signalValue.currentIdentity(), commandIndex) {
			return false
		}
		if action.delayValue != nil && !validProviderDestinationVisibility(input, scope, action.delayValue.currentIdentity(), commandIndex) {
			return false
		}
	}
	return true
}

func validProviderAccess(input declarationPlan, scope identityScope, commandIndex int, value accessDraft) bool {
	if !value.valid || !value.target.Valid() {
		return false
	}
	switch value.kind {
	case accessDeclarationValue,
		accessDeclarationOptionalValue,
		accessDeclarationValueState,
		accessDeclarationLocale,
		accessDeclarationCommandSelection,
		accessDeclarationWorkingDirectory,
		accessDeclarationStdin,
		accessDeclarationStdout,
		accessDeclarationStderr:
	case accessDeclarationInvalid:
		return false
	default:
		return false
	}
	if value.source == nil {
		return value.kind != accessDeclarationValue && value.kind != accessDeclarationOptionalValue && value.kind != accessDeclarationValueState
	}
	owner, local, valid := providerDestinationOwner(input, scope, value.source)
	if !valid || !local {
		return valid
	}
	if !providerCommandSameOrAncestor(input, owner, commandIndex) {
		return false
	}
	if value.kind == accessDeclarationValue || value.kind == accessDeclarationOptionalValue {
		return owner != commandIndex
	}
	return true
}

func (draft *providerBatchDraft) validate(input declarationPlan) error {
	if draft == nil {
		panic("cli: nil provider batch draft")
	}
	if !draft.provider.present() || !draft.parent.present() || !draft.scope.valid() || draft.scope.operation == nil {
		panic("cli: invalid provider batch identity")
	}
	if !validProviderBatchDeclarations(draft, input) {
		return errInvalidProviderBatch
	}
	if !validProviderBatchCallbackCustody(&draft.declarations, input) {
		return errInvalidProviderBatch
	}
	if !validProviderBatchContributions(draft, input) {
		return errInvalidProviderBatch
	}
	return nil
}

func validProviderBatchCallbackCustody(draft *declarationDraft, input declarationPlan) bool {
	if draft == nil {
		return false
	}
	for _, record := range input.records {
		switch record.kind {
		case declarationRecordDestination:
			value := draft.destinations.entries[record.familyIndex].value
			if !validProviderDestinationCallbackCustody(draft, value) {
				return false
			}
		case declarationRecordFlag:
			value := draft.flags.entries[record.familyIndex].value
			if value.completionSet {
				completion, ok := freezeAuthoringCompletion(draft, value.completion)
				if !ok || !completion.callback.validFor(completion.kind) {
					return false
				}
			}
		case declarationRecordArgumentTerm:
			value := draft.argumentTerms.entries[record.familyIndex].value
			if value.completionSet {
				completion, ok := freezeAuthoringCompletion(draft, value.completion)
				if !ok || !completion.callback.validFor(completion.kind) {
					return false
				}
			}
		case declarationRecordValueDirective:
			value := draft.valueDirectives.entries[record.familyIndex].value
			switch value.kind {
			case valueDirectiveComputedDefault, valueDirectiveOptionalComputedDefault:
				computation, ok := freezeAuthoringValueComputation(draft, value.computation)
				if !ok || computation.evaluate == nil {
					return false
				}
				if _, ok := value.destination.(stagedDestinationIdentity); !ok {
					return false
				}
			default:
			}
		case declarationRecordProvider:
			value := draft.providers.entries[record.familyIndex].value
			if !value.valid || value.provide == nil {
				return false
			}
		case declarationRecordInvalid:
			return false
		case declarationRecordDialectDirective,
			declarationRecordFlagSpelling,
			declarationRecordFlagSpellingDirective,
			declarationRecordArgumentForm,
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
			declarationRecordSignalClear,
			declarationRecordSignalPolicy,
			declarationRecordHelpGroup,
			declarationRecordReplacement,
			declarationRecordAccess:
			// These declaration families retain no executable callback custody here.
		}
	}
	return true
}

func validProviderDestinationCallbackCustody(draft *declarationDraft, value destinationDraft) bool {
	switch value.kind {
	case destinationDeclarationDerived, destinationDeclarationOptionalDerived, destinationDeclarationFactResolved:
		computation, ok := freezeAuthoringValueComputation(draft, value.computation)
		if !ok || computation.evaluate == nil {
			return false
		}
	default:
	}
	for _, use := range value.domainUses {
		if !validValueDomainState(use.state) {
			return false
		}
		if use.state.dynamic != nil && use.state.dynamic.evaluate == nil {
			return false
		}
	}
	return true
}

func validProviderBatchContributions(draft *providerBatchDraft, input declarationPlan) bool {
	if draft == nil {
		return false
	}
	replacements := make(map[int]int)
	for contributionIndex, contribution := range draft.contributions {
		switch contribution := contribution.(type) {
		case commandContributionDraft:
			command, ok := draft.declarations.commands.resolve(&draft.declarations, contribution.command)
			if !ok || command.parent == nil || !validProviderSourceOptions(contribution.source) {
				return false
			}
		case aliasContributionDraft:
			if contribution.alias.Name == "" || !validProviderLifecycle(draft.release, contribution.alias.Release) || !validProviderAliasTarget(draft, contribution.target) || !validProviderSourceOptions(contribution.source) {
				return false
			}
		case *shortcutContributionDraft:
			if contribution == nil || contribution.shortcut.Name == "" || !validProviderLifecycle(draft.release, contribution.shortcut.Release) || !validProviderShortcutTarget(draft, contribution.target) || !validProviderSourceOptions(contribution.source) {
				return false
			}
			if len(contribution.shortcut.Args) == 0 && len(contribution.shortcut.AppendArgs) == 0 && len(contribution.defaults) == 0 {
				return false
			}
			if contribution.replacement.valid() {
				replacementIndex := contribution.replacement.index()
				if replacementIndex >= len(draft.contributions) || replacementIndex == contributionIndex {
					return false
				}
				if _, ok := draft.contributions[replacementIndex].(*shortcutContributionDraft); !ok {
					return false
				}
				replacements[contributionIndex] = replacementIndex
			}
			for _, defaultValue := range contribution.defaults {
				switch defaultValue := defaultValue.(type) {
				case destinationDefaultDraft:
					if !validProviderDestinationIdentity(defaultValue.destination, &draft.declarations, input) || !validProviderSourceOptions(defaultValue.source) {
						return false
					}
					if identity, ok := defaultValue.destination.(stagedDestinationIdentity); ok {
						destination, resolved := draft.declarations.destinations.resolve(&draft.declarations, identity.ref)
						if !resolved {
							return false
						}
						input, valid := staticDestinationAcceptanceInput(*destination)
						if !valid {
							return false
						}
						definition, valid := prepareDestinationDefinitionForRelease(input, draft.release)
						if !valid || !validEffectiveDomainContribution(definition, defaultValue.value) {
							return false
						}
					}
				case flagDefault:
					if defaultValue.spelling == "" || len(defaultValue.values) == 0 || !validProviderSourceOptions(defaultValue.source) {
						return false
					}
				default:
					return false
				}
			}
		default:
			return false
		}
	}
	for start := range replacements {
		seen := make(map[int]struct{})
		for current := start; ; {
			if _, duplicate := seen[current]; duplicate {
				return false
			}
			seen[current] = struct{}{}
			next, ok := replacements[current]
			if !ok {
				break
			}
			current = next
		}
	}
	return true
}

func validProviderSourceOptions(options []SourceOption) bool {
	_, ok := producerSource(options)
	return ok
}

func validProviderDestinationIdentity(identity destinationIdentity, draft *declarationDraft, input declarationPlan) bool {
	switch identity := identity.(type) {
	case stagedDestinationIdentity:
		if draft == nil {
			return false
		}
		if _, ok := draft.destinations.resolve(draft, identity.ref); !ok {
			return false
		}
		_, ok := plannedDestinationIndex(input, identity.ref)
		return ok
	case acceptedDestinationIdentity:
		return identity.id.present() && draft.scope.includes(identity.scope)
	default:
		return false
	}
}

func validProviderAliasTarget(draft *providerBatchDraft, target providerCommandTarget) bool {
	switch target := target.(type) {
	case stagedCommandTarget:
		return validProviderStagedCommandTarget(draft, target)
	case pathTarget:
		return len(target.path) == 1 && target.path[0] != ""
	default:
		return false
	}
}

func validProviderShortcutTarget(draft *providerBatchDraft, target providerCommandTarget) bool {
	switch target := target.(type) {
	case stagedCommandTarget:
		return validProviderStagedCommandTarget(draft, target)
	case pathTarget:
		return len(target.path) != 0 && !slices.Contains(target.path, "")
	default:
		return false
	}
}

func validProviderStagedCommandTarget(draft *providerBatchDraft, target stagedCommandTarget) bool {
	if draft == nil || !target.command.valid() {
		return false
	}
	_, ok := draft.declarations.commands.resolve(&draft.declarations, target.command)
	return ok
}
