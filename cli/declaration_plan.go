package cli

import (
	"cmp"
	"errors"
	"slices"
)

var errInvalidAuthoringTransaction = errors.New("cli: invalid authoring transaction")

type declarationPlanRecord struct {
	commandIndex          int
	profileIndex          int
	commandHelpGroupIndex int
	flagHelpGroupIndex    int

	kind        declarationRecordKind
	familyIndex int
	position    declarationPosition
}

type declarationPlanIndex struct {
	commands               map[commandRef]int
	destinations           map[destinationRef]int
	valueDirectives        map[valueDirectiveRef]int
	flags                  map[flagRef]int
	flagSpellings          map[flagSpellingRef]int
	argumentForms          map[argumentFormDraftRef]int
	argumentTerms          map[argumentTermDraftRef]int
	grammarProfileSwitches map[grammarProfileSwitchRef]int
	grammarProfileCases    map[grammarProfileCaseRef]int
	rules                  map[ruleDraftRef]int
	advisories             map[advisoryDraftRef]int
	shortcuts              map[shortcutDraftRef]int
	helpGroups             map[helpGroupRef]int
}

type declarationPlan struct {
	records         []declarationPlanRecord
	indexes         declarationPlanIndex
	destinationRefs map[int]destinationRef
	finalPosition   declarationPosition
}

func (input declarationPlan) positionFor(index int) declarationPosition {
	if index < 0 {
		return 0
	}
	if index < len(input.records) {
		return input.records[index].position
	}

	position := input.finalPosition
	for current := len(input.records); current <= index; current++ {
		position++
		if position == 0 {
			panic("cli: declaration position exhausted")
		}
	}
	return position
}

func declarationRefAt[T any](draft *declarationDraft, family *declarationFamily[T], index int) draftRef[T] {
	if draft == nil || family == nil || index < 0 || index >= len(family.entries) {
		return draftRef[T]{}
	}
	entry := family.entries[index]
	return draftRef[T]{
		draft:      draft.identity,
		index:      index,
		generation: entry.generation,
	}
}

func (draft *declarationDraft) declarationPlan() (declarationPlan, error) {
	var input declarationPlan
	if draft == nil || draft.identity == nil {
		return input, errInvalidAuthoringTransaction
	}

	records := make([]declarationPlanRecord, 0,
		len(draft.commands.entries)+len(draft.destinations.entries)+len(draft.flags.entries)+len(draft.flagSpellings.entries)+
			len(draft.valueDirectives.entries)+len(draft.dialectDirectives.entries)+len(draft.flagSpellingDirectives.entries)+
			len(draft.argumentForms.entries)+len(draft.argumentTerms.entries)+len(draft.argumentFormSelections.entries)+
			len(draft.conditionPolicies.entries)+len(draft.lifecycleDirectives.entries)+len(draft.grammarProfileSwitches.entries)+
			len(draft.grammarProfileCases.entries)+len(draft.rules.entries)+len(draft.ruleDirectives.entries)+
			len(draft.advisories.entries)+len(draft.advisoryDirectives.entries)+len(draft.noticeSuppressions.entries)+
			len(draft.shortcuts.entries)+len(draft.shortcutDirectives.entries)+len(draft.providers.entries)+
			len(draft.signalClears.entries)+len(draft.signalPolicies.entries)+len(draft.helpGroups.entries)+
			len(draft.replacements.entries)+len(draft.accesses.entries))

	for index, entry := range draft.commands.entries {
		ref := declarationRefAt(draft, &draft.commands, index)
		if ref == draft.root {
			continue
		}
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordCommand,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.destinations.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordDestination,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.valueDirectives.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordValueDirective,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.dialectDirectives.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordDialectDirective,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.flags.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordFlag,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.flagSpellings.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordFlagSpelling,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.flagSpellingDirectives.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordFlagSpellingDirective,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.argumentForms.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordArgumentForm,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.argumentTerms.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordArgumentTerm,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.argumentFormSelections.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordArgumentFormSelection,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.conditionPolicies.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordConditionPolicy,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.lifecycleDirectives.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordLifecycleDirective,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.grammarProfileSwitches.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordGrammarProfileSwitch,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.grammarProfileCases.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordGrammarProfileCase,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.rules.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordRule,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.ruleDirectives.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordRuleDirective,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.advisories.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordAdvisory,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.advisoryDirectives.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordAdvisoryDirective,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.noticeSuppressions.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordNoticeSuppression,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.shortcuts.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordShortcut,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.shortcutDirectives.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordShortcutDirective,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.providers.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordProvider,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.signalClears.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordSignalClear,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.signalPolicies.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordSignalPolicy,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.helpGroups.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordHelpGroup,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.replacements.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordReplacement,
			familyIndex: index,
			position:    entry.value.position,
		})
	}
	for index, entry := range draft.accesses.entries {
		records = append(records, declarationPlanRecord{
			kind:        declarationRecordAccess,
			familyIndex: index,
			position:    entry.value.position,
		})
	}

	slices.SortFunc(records, func(left, right declarationPlanRecord) int {
		if order := cmp.Compare(left.position, right.position); order != 0 {
			return order
		}
		if left.kind != right.kind {
			if left.kind == declarationRecordDestination {
				return -1
			}
			if right.kind == declarationRecordDestination {
				return 1
			}
			return cmp.Compare(left.kind, right.kind)
		}
		return cmp.Compare(left.familyIndex, right.familyIndex)
	})
	for index, record := range records {
		if record.position == 0 {
			return input, errInvalidAuthoringTransaction
		}
		if index == 0 {
			continue
		}
		previous := records[index-1]
		if previous.position > record.position {
			return input, errInvalidAuthoringTransaction
		}
		if previous.position == record.position && !validSharedDeclarationPosition(previous.kind, record.kind) {
			return input, errInvalidAuthoringTransaction
		}
	}

	indexes := declarationPlanIndex{
		commands:               make(map[commandRef]int),
		destinations:           make(map[destinationRef]int),
		valueDirectives:        make(map[valueDirectiveRef]int),
		flags:                  make(map[flagRef]int),
		flagSpellings:          make(map[flagSpellingRef]int),
		argumentForms:          make(map[argumentFormDraftRef]int),
		argumentTerms:          make(map[argumentTermDraftRef]int),
		grammarProfileSwitches: make(map[grammarProfileSwitchRef]int),
		grammarProfileCases:    make(map[grammarProfileCaseRef]int),
		rules:                  make(map[ruleDraftRef]int),
		advisories:             make(map[advisoryDraftRef]int),
		shortcuts:              make(map[shortcutDraftRef]int),
		helpGroups:             make(map[helpGroupRef]int),
	}
	if draft.root.valid() {
		indexes.commands[draft.root] = -1
	}
	for order, record := range records {
		switch record.kind {
		case declarationRecordCommand:
			indexes.commands[declarationRefAt(draft, &draft.commands, record.familyIndex)] = order
		case declarationRecordDestination:
			indexes.destinations[declarationRefAt(draft, &draft.destinations, record.familyIndex)] = order
		case declarationRecordValueDirective:
			indexes.valueDirectives[declarationRefAt(draft, &draft.valueDirectives, record.familyIndex)] = order
		case declarationRecordFlag:
			indexes.flags[declarationRefAt(draft, &draft.flags, record.familyIndex)] = order
		case declarationRecordFlagSpelling:
			indexes.flagSpellings[declarationRefAt(draft, &draft.flagSpellings, record.familyIndex)] = order
		case declarationRecordArgumentForm:
			indexes.argumentForms[declarationRefAt(draft, &draft.argumentForms, record.familyIndex)] = order
		case declarationRecordArgumentTerm:
			indexes.argumentTerms[declarationRefAt(draft, &draft.argumentTerms, record.familyIndex)] = order
		case declarationRecordGrammarProfileSwitch:
			indexes.grammarProfileSwitches[declarationRefAt(draft, &draft.grammarProfileSwitches, record.familyIndex)] = order
		case declarationRecordGrammarProfileCase:
			indexes.grammarProfileCases[declarationRefAt(draft, &draft.grammarProfileCases, record.familyIndex)] = order
		case declarationRecordRule:
			indexes.rules[declarationRefAt(draft, &draft.rules, record.familyIndex)] = order
		case declarationRecordAdvisory:
			indexes.advisories[declarationRefAt(draft, &draft.advisories, record.familyIndex)] = order
		case declarationRecordShortcut:
			indexes.shortcuts[declarationRefAt(draft, &draft.shortcuts, record.familyIndex)] = order
		case declarationRecordHelpGroup:
			indexes.helpGroups[declarationRefAt(draft, &draft.helpGroups, record.familyIndex)] = order
		case declarationRecordInvalid:
			panic("cli: invalid declaration acceptance record")
		case declarationRecordDialectDirective,
			declarationRecordFlagSpellingDirective,
			declarationRecordArgumentFormSelection,
			declarationRecordConditionPolicy,
			declarationRecordLifecycleDirective,
			declarationRecordRuleDirective,
			declarationRecordAdvisoryDirective,
			declarationRecordNoticeSuppression,
			declarationRecordShortcutDirective,
			declarationRecordProvider,
			declarationRecordSignalClear,
			declarationRecordSignalPolicy,
			declarationRecordReplacement,
			declarationRecordAccess:
			// These families carry no cross-record reference that needs a global acceptance index.
		}
	}

	input.records = records
	input.indexes = indexes
	input.destinationRefs = make(map[int]destinationRef, len(indexes.destinations))
	input.finalPosition = draft.position
	for ref, index := range indexes.destinations {
		input.destinationRefs[index] = ref
	}

	for order := range input.records {
		record, ok := draft.acceptanceRecord(input.records[order], indexes)
		if !ok {
			return declarationPlan{}, errInvalidAuthoringTransaction
		}
		input.records[order] = record
	}

	if err := validateDeclarationPlan(draft, input); err != nil {
		return declarationPlan{}, err
	}
	return input, nil
}

func validSharedDeclarationPosition(first, second declarationRecordKind) bool {
	if first != declarationRecordDestination {
		return false
	}
	return second == declarationRecordFlag || second == declarationRecordArgumentTerm
}

func (draft *declarationDraft) acceptanceRecord(record declarationPlanRecord, indexes declarationPlanIndex) (declarationPlanRecord, bool) {
	record.commandIndex = -1
	record.profileIndex = -1
	record.commandHelpGroupIndex = -1
	record.flagHelpGroupIndex = -1

	setContext := func(command commandRef, profile grammarProfileCaseRef, commandHelp, flagHelp helpGroupRef) bool {
		var ok bool
		record.commandIndex, ok = declarationCommandIndex(draft, indexes, command)
		if !ok {
			return false
		}
		if profile.valid() {
			record.profileIndex, ok = indexes.grammarProfileCases[profile]
			if !ok {
				return false
			}
		}
		if commandHelp.valid() {
			record.commandHelpGroupIndex, ok = indexes.helpGroups[commandHelp]
			if !ok {
				return false
			}
		}
		if flagHelp.valid() {
			record.flagHelpGroupIndex, ok = indexes.helpGroups[flagHelp]
			if !ok {
				return false
			}
		}
		return true
	}

	switch record.kind {
	case declarationRecordCommand:
		command := draft.commands.entries[record.familyIndex].value
		switch parent := command.parent.(type) {
		case stagedCommandParent:
			if !setContext(parent.command, command.profile, command.helpGroup, helpGroupRef{}) {
				return declarationPlanRecord{}, false
			}
		case acceptedCommandParent:
			record.commandIndex = -1
			if command.profile.valid() {
				profileIndex, ok := indexes.grammarProfileCases[command.profile]
				if !ok {
					return declarationPlanRecord{}, false
				}
				record.profileIndex = profileIndex
			}
			if command.helpGroup.valid() {
				helpIndex, ok := indexes.helpGroups[command.helpGroup]
				if !ok {
					return declarationPlanRecord{}, false
				}
				record.commandHelpGroupIndex = helpIndex
			}
		default:
			return declarationPlanRecord{}, false
		}
	case declarationRecordFlagSpelling:
		value := draft.flagSpellings.entries[record.familyIndex].value
		flag, ok := draft.flags.resolve(draft, value.flag)
		if !ok || !setContext(flag.command, flag.profile, helpGroupRef{}, flag.helpGroup) {
			return declarationPlanRecord{}, false
		}
	case declarationRecordGrammarProfileSwitch, declarationRecordGrammarProfileCase:
		var command commandRef
		if record.kind == declarationRecordGrammarProfileSwitch {
			command = draft.grammarProfileSwitches.entries[record.familyIndex].value.command
		} else {
			command = draft.grammarProfileCases.entries[record.familyIndex].value.command
		}
		if !setContext(command, grammarProfileCaseRef{}, helpGroupRef{}, helpGroupRef{}) {
			return declarationPlanRecord{}, false
		}
	default:
		command, profile, commandHelp, flagHelp, ok := draft.acceptanceContext(record)
		if !ok || !setContext(command, profile, commandHelp, flagHelp) {
			return declarationPlanRecord{}, false
		}
	}

	return record, true
}

func (draft *declarationDraft) acceptanceContext(record declarationPlanRecord) (commandRef, grammarProfileCaseRef, helpGroupRef, helpGroupRef, bool) {
	switch record.kind {
	case declarationRecordDestination:
		value := draft.destinations.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordValueDirective:
		value := draft.valueDirectives.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordDialectDirective:
		value := draft.dialectDirectives.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordFlag:
		value := draft.flags.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, value.helpGroup, true
	case declarationRecordFlagSpellingDirective:
		value := draft.flagSpellingDirectives.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordArgumentForm:
		value := draft.argumentForms.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordArgumentTerm:
		value := draft.argumentTerms.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordArgumentFormSelection:
		value := draft.argumentFormSelections.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordConditionPolicy:
		value := draft.conditionPolicies.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordLifecycleDirective:
		value := draft.lifecycleDirectives.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordRule:
		value := draft.rules.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordRuleDirective:
		value := draft.ruleDirectives.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordAdvisory:
		value := draft.advisories.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordAdvisoryDirective:
		value := draft.advisoryDirectives.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordNoticeSuppression:
		value := draft.noticeSuppressions.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordShortcut:
		value := draft.shortcuts.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordShortcutDirective:
		value := draft.shortcutDirectives.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordProvider:
		value := draft.providers.entries[record.familyIndex].value
		return value.command, value.profile, value.helpGroup, helpGroupRef{}, true
	case declarationRecordSignalClear:
		value := draft.signalClears.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordSignalPolicy:
		value := draft.signalPolicies.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordHelpGroup:
		value := draft.helpGroups.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordReplacement:
		value := draft.replacements.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	case declarationRecordAccess:
		value := draft.accesses.entries[record.familyIndex].value
		return value.command, value.profile, helpGroupRef{}, helpGroupRef{}, true
	default:
		return commandRef{}, grammarProfileCaseRef{}, helpGroupRef{}, helpGroupRef{}, false
	}
}

func (input declarationPlan) conditionTargetIndex(target authoringConditionTarget) (int, bool) {
	switch target := target.(type) {
	case authoringCommandConditionTarget:
		index, ok := input.indexes.commands[target.reference]
		return index, ok
	case authoringFlagConditionTarget:
		index, ok := input.indexes.flags[target.reference]
		return index, ok
	case authoringArgumentFormConditionTarget:
		index, ok := input.indexes.argumentForms[target.reference]
		return index, ok
	case authoringArgumentTermConditionTarget:
		index, ok := input.indexes.argumentTerms[target.reference]
		return index, ok
	default:
		return 0, false
	}
}

func (input declarationPlan) lifecycleTargetIndex(target authoringLifecycleTarget) (int, bool) {
	switch target := target.(type) {
	case authoringEnvironmentLifecycleTarget:
		index, ok := input.indexes.valueDirectives[target.reference]
		return index, ok
	case authoringArgumentFormLifecycleTarget:
		index, ok := input.indexes.argumentForms[target.reference]
		return index, ok
	case authoringArgumentTermLifecycleTarget:
		index, ok := input.indexes.argumentTerms[target.reference]
		return index, ok
	default:
		return 0, false
	}
}

func validateDeclarationPlan(draft *declarationDraft, input declarationPlan) error {
	if draft == nil || !draft.scope.valid() {
		return errInvalidAuthoringTransaction
	}
	if draft.scope.operation == nil && !draft.root.valid() {
		return errInvalidAuthoringTransaction
	}
	if draft.scope.operation != nil && draft.root.valid() {
		return errInvalidAuthoringTransaction
	}
	for index, record := range input.records {
		if record.commandHelpGroupIndex >= index || record.flagHelpGroupIndex >= index || record.commandIndex >= index || record.profileIndex >= index {
			return errInvalidAuthoringTransaction
		}
		if record.commandIndex < -1 || record.profileIndex < -1 || record.commandHelpGroupIndex < -1 || record.flagHelpGroupIndex < -1 {
			return errInvalidAuthoringTransaction
		}
		if record.commandIndex >= 0 && input.records[record.commandIndex].kind != declarationRecordCommand {
			return errInvalidAuthoringTransaction
		}
		if record.profileIndex >= 0 && input.records[record.profileIndex].kind != declarationRecordGrammarProfileCase {
			return errInvalidAuthoringTransaction
		}
		if !validDeclarationPlanRecord(draft, input, index, record) {
			return errInvalidAuthoringTransaction
		}
	}
	return nil
}

func validDeclarationPlanRecord(draft *declarationDraft, input declarationPlan, index int, record declarationPlanRecord) bool {
	validIndex := func(value int, kind declarationRecordKind) bool {
		return value >= 0 && value < index && input.records[value].kind == kind
	}
	validRef := func(value int, ok bool, kind declarationRecordKind) bool {
		return ok && validIndex(value, kind)
	}
	validDestination := func(identity destinationIdentity) bool {
		switch identity := identity.(type) {
		case stagedDestinationIdentity:
			position, ok := plannedDestinationIndex(input, identity.ref)
			return ok && validIndex(position, declarationRecordDestination)
		case acceptedDestinationIdentity:
			return identity.id.present() && draft.scope.includes(identity.scope)
		default:
			return false
		}
	}

	switch record.kind {
	case declarationRecordCommand:
		value := draft.commands.entries[record.familyIndex].value
		if value.name == "" {
			return false
		}
		if value.intrinsic {
			return value.makeConfig == nil && len(value.middleware) == 0 && value.handler == nil
		}
		return value.makeConfig != nil
	case declarationRecordDialectDirective, declarationRecordArgumentForm,
		declarationRecordSignalClear, declarationRecordHelpGroup:
		return true
	case declarationRecordShortcut:
		value := draft.shortcuts.entries[record.familyIndex].value
		return value.valid && record.commandIndex >= 0
	case declarationRecordDestination:
		value := draft.destinations.entries[record.familyIndex].value
		if value.intrinsic {
			return value.valid && value.storage == nil && !value.binding.Valid() && value.handle == nil
		}
		return value.valid && value.storage != nil && value.binding.Valid() && value.handle != nil
	case declarationRecordValueDirective:
		value := draft.valueDirectives.entries[record.familyIndex].value
		return value.valid && validDestination(value.destination)
	case declarationRecordAccess:
		value := draft.accesses.entries[record.familyIndex].value
		return value.valid && value.target.Valid() && (value.source == nil || validDestination(value.source))
	case declarationRecordFlag:
		value := draft.flags.entries[record.familyIndex].value
		position, ok := input.indexes.destinations[value.destination]
		if !value.valid || !validRef(position, ok, declarationRecordDestination) {
			return false
		}
		return input.records[position].commandIndex == record.commandIndex
	case declarationRecordFlagSpelling:
		value := draft.flagSpellings.entries[record.familyIndex].value
		position, ok := input.indexes.flags[value.flag]
		return value.valid && validRef(position, ok, declarationRecordFlag)
	case declarationRecordFlagSpellingDirective:
		value := draft.flagSpellingDirectives.entries[record.familyIndex].value
		position, ok := input.indexes.flagSpellings[value.spelling]
		return value.valid && validRef(position, ok, declarationRecordFlagSpelling)
	case declarationRecordArgumentTerm:
		value := draft.argumentTerms.entries[record.familyIndex].value
		formPosition, formOK := input.indexes.argumentForms[value.form]
		if !value.valid || !validRef(formPosition, formOK, declarationRecordArgumentForm) {
			return false
		}
		if input.records[formPosition].commandIndex != record.commandIndex {
			return false
		}
		if value.kind == argumentTermLiteral {
			return true
		}
		destinationPosition, destinationOK := input.indexes.destinations[value.destination]
		if !validRef(destinationPosition, destinationOK, declarationRecordDestination) {
			return false
		}
		return input.records[destinationPosition].commandIndex == record.commandIndex
	case declarationRecordArgumentFormSelection:
		value := draft.argumentFormSelections.entries[record.familyIndex].value
		formPosition, formOK := input.indexes.argumentForms[value.form]
		return value.valid && validRef(formPosition, formOK, declarationRecordArgumentForm) && validDestination(value.destination)
	case declarationRecordConditionPolicy:
		value := draft.conditionPolicies.entries[record.familyIndex].value
		return value.valid && value.condition.state != nil && value.condition.state.owner == draft.identity
	case declarationRecordGrammarProfileSwitch:
		value := draft.grammarProfileSwitches.entries[record.familyIndex].value
		return value.valid && validDestination(value.selector)
	case declarationRecordGrammarProfileCase:
		value := draft.grammarProfileCases.entries[record.familyIndex].value
		position, ok := input.indexes.grammarProfileSwitches[value.switchRef]
		return value.valid && validRef(position, ok, declarationRecordGrammarProfileSwitch)
	case declarationRecordLifecycleDirective:
		return draft.lifecycleDirectives.entries[record.familyIndex].value.valid
	case declarationRecordRule:
		value := draft.rules.entries[record.familyIndex].value
		if !value.valid {
			return false
		}
		for _, destination := range value.destinations {
			if !validDestination(destination) {
				return false
			}
		}
		return true
	case declarationRecordRuleDirective:
		value := draft.ruleDirectives.entries[record.familyIndex].value
		rulePosition, ruleOK := input.indexes.rules[value.rule]
		return value.valid && validRef(rulePosition, ruleOK, declarationRecordRule) && validDestination(value.destination)
	case declarationRecordAdvisory:
		return draft.advisories.entries[record.familyIndex].value.valid
	case declarationRecordAdvisoryDirective:
		value := draft.advisoryDirectives.entries[record.familyIndex].value
		advisoryPosition, advisoryOK := input.indexes.advisories[value.advisory]
		return value.valid && validRef(advisoryPosition, advisoryOK, declarationRecordAdvisory) && validDestination(value.destination)
	case declarationRecordNoticeSuppression:
		value := draft.noticeSuppressions.entries[record.familyIndex].value
		return value.valid && value.condition.state != nil && value.condition.state.owner == draft.identity
	case declarationRecordShortcutDirective:
		value := draft.shortcutDirectives.entries[record.familyIndex].value
		shortcutPosition, shortcutOK := input.indexes.shortcuts[value.shortcut]
		if !value.valid || !validRef(shortcutPosition, shortcutOK, declarationRecordShortcut) ||
			(value.destination != nil && !validDestination(value.destination)) {
			return false
		}
		if !value.replacement.valid() {
			return true
		}
		replacementPosition, replacementOK := input.indexes.shortcuts[value.replacement]
		return validRef(replacementPosition, replacementOK, declarationRecordShortcut)
	case declarationRecordProvider:
		return draft.providers.entries[record.familyIndex].value.valid
	case declarationRecordSignalPolicy:
		return validSignalPolicyDraft(draft, draft.signalPolicies.entries[record.familyIndex].value)
	case declarationRecordReplacement:
		value := draft.replacements.entries[record.familyIndex].value
		if !value.valid {
			return false
		}
		sourcePosition, sourceOK := declarationReplacementTargetIndex(value.source, input.indexes)
		targetPosition, targetOK := declarationReplacementTargetIndex(value.target, input.indexes)
		return sourceOK && targetOK && sourcePosition >= 0 && sourcePosition < index && targetPosition >= 0 && targetPosition < index
	default:
		return false
	}
}

func declarationCommandIndex(draft *declarationDraft, indexes declarationPlanIndex, command commandRef) (int, bool) {
	if command.valid() {
		index, ok := indexes.commands[command]
		return index, ok
	}
	if draft != nil && !draft.root.valid() {
		return -1, true
	}
	return 0, false
}

func declarationReplacementTargetIndex(target authoringReplacementTarget, indexes declarationPlanIndex) (int, bool) {
	switch target.kind {
	case authoringReplacementCommand:
		index, ok := indexes.commands[target.command]
		return index, ok
	case authoringReplacementFlag:
		index, ok := indexes.flags[target.flag]
		return index, ok
	case authoringReplacementArgument, authoringReplacementArgumentLiteral:
		index, ok := indexes.argumentTerms[target.argument]
		return index, ok
	case authoringReplacementFlagSpelling:
		index, ok := indexes.flagSpellings[target.spelling]
		return index, ok
	case authoringReplacementEnvironment:
		index, ok := indexes.valueDirectives[target.environment]
		return index, ok
	case authoringReplacementArgumentForm:
		index, ok := indexes.argumentForms[target.argumentForm]
		return index, ok
	default:
		return 0, false
	}
}

func plannedDestinationIndex(input declarationPlan, ref destinationRef) (int, bool) {
	index, ok := input.indexes.destinations[ref]
	return index, ok
}

func validSignalPolicyDraft(draft *declarationDraft, value signalPolicyDraft) bool {
	return validSignalPolicy(draft, value.match, value.plan, value.valid)
}

func validSignalPolicy(draft *declarationDraft, match *authoringSignalMatch, plan *signalActionPlanState, valid bool) bool {
	if !valid || match == nil || plan == nil || !match.valid ||
		!plan.valid || plan.pendingDelay || len(plan.actions) == 0 {
		return false
	}
	owner := match.owner
	if owner == nil || plan.owner != owner {
		return false
	}
	for _, clause := range match.conditions {
		if clause.condition.state == nil || !clause.condition.state.valid || clause.condition.state.owner != owner {
			return false
		}
	}
	phases := match.phases
	if len(phases) == 0 {
		phases = []SignalPhase{SignalPhaseExecution}
	}
	hasResolution := false
	executionOnly := len(phases) == 1 && phases[0] == SignalPhaseExecution
	for _, phase := range phases {
		if phase == SignalPhaseResolution {
			hasResolution = true
		}
	}
	if hasResolution && match.hasSender {
		return false
	}
	for _, action := range plan.actions {
		if !action.valid || !action.kind.Valid() {
			return false
		}
		if action.signalValue != nil && !stagedDestinationOwnedBy(action.signalValue, owner) {
			return false
		}
		if action.delayValue != nil && !stagedDestinationOwnedBy(action.delayValue, owner) {
			return false
		}
		if action.delayed && action.delayValue == nil && action.delay <= 0 {
			return false
		}
		if hasResolution && action.kind != signalActionKindIgnore && action.kind != signalActionKindCancel {
			return false
		}
		if action.kind == signalActionKindEmit {
			if draft == nil || !action.endpoint.valid() || !action.endpoint.ownedBy(owner) || !executionOnly || action.delayed {
				return false
			}
			endpoint, ok := draft.signalEndpoints.resolve(draft, action.endpoint)
			if !ok || !endpoint.valid || !endpoint.binding.Valid() || endpoint.adapt == nil {
				return false
			}
		}
	}
	return true
}

func validEmbeddedDestinationDraft(destination destinationDraft) bool {
	if !destination.valid || destination.storage == nil {
		return false
	}
	switch destination.kind {
	case destinationDeclarationValue, destinationDeclarationSequence, destinationDeclarationKeyed:
		return destination.language != nil
	case destinationDeclarationOpaque:
		return destination.language == nil && destination.computation.evaluate == nil
	default:
		return false
	}
}
