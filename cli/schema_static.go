package cli

import (
	"cmp"
	"errors"
	"slices"
	"strings"
)

var errInvalidModelTopology = errors.New("application: invalid model topology")

type staticTopology struct {
	records staticRecords
	grammar grammarView

	root         CommandID
	commandOrder []CommandID

	commands          map[CommandID]int
	destinations      map[ValueID]int
	flags             map[FlagID]int
	flagSpellings     map[FlagSpellingID]int
	argumentForms     map[ArgumentFormID]int
	argumentTerms     map[ArgumentTermID]int
	commandEntries    map[CommandEntryID]int
	actionEntries     map[ActionEntryID]int
	defaults          map[CommandDefaultID]int
	providers         map[CommandProviderID]int
	rules             map[RuleID]int
	advisories        map[advisoryID]int
	signalPolicies    map[SignalPolicyID]int
	domainEntries     map[ValueDomainEntryID]int
	environmentInputs map[EnvironmentInputID]int
	grammarProfiles   map[GrammarProfileID]int
	signalEndpoints   map[SignalEndpointID]int
	signalCounters    map[SignalCounterID]int

	entryGrammar         map[CommandEntryID]commandEntryGrammar
	actionCommandGrammar map[ActionEntryID]actionCommandGrammar
	actionFlagGrammar    map[ActionEntryID]semanticActionFlagGrammar
	flagGrammar          map[FlagID]flagGrammar
	spellingGrammar      map[FlagSpellingID]flagSpellingGrammar
	formGrammar          map[ArgumentFormID]argumentFormGrammar
	termGrammar          map[ArgumentTermID]argumentTermGrammar

	childrenByCommand           map[CommandID][]CommandID
	destinationsByCommand       map[CommandID][]ValueID
	flagsByCommand              map[CommandID][]FlagID
	formsByCommand              map[CommandID][]ArgumentFormID
	entriesByParent             map[CommandID][]CommandEntryID
	actionEntriesByParent       map[CommandID][]ActionEntryID
	defaultsByParent            map[CommandID][]CommandDefaultID
	providersByParent           map[CommandID][]CommandProviderID
	rulesByCommand              map[CommandID][]RuleID
	advisoriesByCommand         map[CommandID][]advisoryID
	signalPoliciesByCommand     map[CommandID][]SignalPolicyID
	profilesByCommand           map[CommandID][]GrammarProfileID
	conditionPoliciesByCommand  map[CommandID][]int
	noticeSuppressionsByCommand map[CommandID][]int
	helpGroupsByCommand         map[CommandID][]int
	termsByForm                 map[ArgumentFormID][]ArgumentTermID
	spellingsByFlag             map[FlagID][]FlagSpellingID
	environmentByDestination    map[ValueID][]EnvironmentInputID
	canonicalEntries            map[CommandID]CommandEntryID
	commandPaths                map[CommandID][]string
	endpointPolicies            map[SignalEndpointID][]SignalPolicyID
	counterPolicies             map[SignalCounterID][]SignalPolicyID
}

type staticArtifactSource struct {
	*staticTopology

	application applicationMetadata
	analysis    staticAnalysis
}

type schemaProjector struct {
	*staticArtifactSource
}

func prepareStaticTopology(records staticRecords, grammar grammarView) (*staticTopology, error) {
	if grammar == nil || len(records.commands) == 0 {
		return nil, errInvalidModelTopology
	}
	topology := &staticTopology{
		records:                     records,
		grammar:                     grammar,
		commands:                    make(map[CommandID]int, len(records.commands)),
		destinations:                make(map[ValueID]int, len(records.destinations)),
		flags:                       make(map[FlagID]int, len(records.flags)),
		flagSpellings:               make(map[FlagSpellingID]int, len(records.flagSpellings)),
		argumentForms:               make(map[ArgumentFormID]int, len(records.argumentForms)),
		argumentTerms:               make(map[ArgumentTermID]int, len(records.argumentTerms)),
		commandEntries:              make(map[CommandEntryID]int, len(records.commandEntries)),
		actionEntries:               make(map[ActionEntryID]int, len(records.actionEntries)),
		defaults:                    make(map[CommandDefaultID]int, len(records.defaults)),
		providers:                   make(map[CommandProviderID]int, len(records.providers)),
		rules:                       make(map[RuleID]int, len(records.rules)),
		advisories:                  make(map[advisoryID]int, len(records.advisories)),
		signalPolicies:              make(map[SignalPolicyID]int, len(records.signalPolicies)),
		domainEntries:               make(map[ValueDomainEntryID]int, len(records.domainEntries)),
		environmentInputs:           make(map[EnvironmentInputID]int, len(records.environmentInputs)),
		grammarProfiles:             make(map[GrammarProfileID]int, len(records.grammarProfiles)),
		signalEndpoints:             make(map[SignalEndpointID]int, len(records.signalEndpoints)),
		signalCounters:              make(map[SignalCounterID]int, len(records.signalCounters)),
		entryGrammar:                make(map[CommandEntryID]commandEntryGrammar),
		actionCommandGrammar:        make(map[ActionEntryID]actionCommandGrammar),
		actionFlagGrammar:           make(map[ActionEntryID]semanticActionFlagGrammar),
		flagGrammar:                 make(map[FlagID]flagGrammar),
		spellingGrammar:             make(map[FlagSpellingID]flagSpellingGrammar),
		formGrammar:                 make(map[ArgumentFormID]argumentFormGrammar),
		termGrammar:                 make(map[ArgumentTermID]argumentTermGrammar),
		childrenByCommand:           make(map[CommandID][]CommandID),
		destinationsByCommand:       make(map[CommandID][]ValueID),
		flagsByCommand:              make(map[CommandID][]FlagID),
		formsByCommand:              make(map[CommandID][]ArgumentFormID),
		entriesByParent:             make(map[CommandID][]CommandEntryID),
		actionEntriesByParent:       make(map[CommandID][]ActionEntryID),
		defaultsByParent:            make(map[CommandID][]CommandDefaultID),
		providersByParent:           make(map[CommandID][]CommandProviderID),
		rulesByCommand:              make(map[CommandID][]RuleID),
		advisoriesByCommand:         make(map[CommandID][]advisoryID),
		signalPoliciesByCommand:     make(map[CommandID][]SignalPolicyID),
		profilesByCommand:           make(map[CommandID][]GrammarProfileID),
		conditionPoliciesByCommand:  make(map[CommandID][]int),
		noticeSuppressionsByCommand: make(map[CommandID][]int),
		helpGroupsByCommand:         make(map[CommandID][]int),
		termsByForm:                 make(map[ArgumentFormID][]ArgumentTermID),
		spellingsByFlag:             make(map[FlagID][]FlagSpellingID),
		environmentByDestination:    make(map[ValueID][]EnvironmentInputID),
		canonicalEntries:            make(map[CommandID]CommandEntryID),
		commandPaths:                make(map[CommandID][]string, len(records.commands)),
		endpointPolicies:            make(map[SignalEndpointID][]SignalPolicyID),
		counterPolicies:             make(map[SignalCounterID][]SignalPolicyID),
	}
	if err := topology.indexAndValidate(); err != nil {
		return nil, err
	}
	return topology, nil
}

func validateAcceptedTopology(records staticRecords, grammar grammarView) error {
	_, err := prepareStaticTopology(records, grammar)
	return err
}

func prepareStaticArtifactSource(valid bool, application applicationMetadata, records staticRecords, grammar grammarView, analysis staticAnalysis) (*staticArtifactSource, error) {
	if !valid {
		return nil, errInvalidModelTopology
	}
	topology, err := prepareStaticTopology(records, grammar)
	if err != nil {
		return nil, err
	}
	return &staticArtifactSource{
		staticTopology: topology,
		application:    application,
		analysis:       analysis,
	}, nil
}

func prepareSchemaProjector(valid bool, application applicationMetadata, records staticRecords, grammar grammarView, analysis staticAnalysis) (*schemaProjector, error) {
	source, err := prepareStaticArtifactSource(valid, application, records, grammar, analysis)
	if err != nil {
		return nil, err
	}
	return &schemaProjector{staticArtifactSource: source}, nil
}

func putRecordIndex[I comparable](index map[I]int, id I, position int) bool {
	if _, exists := index[id]; exists {
		return false
	}
	index[id] = position
	return true
}

func (p *staticTopology) indexAndValidate() error {
	rootCount := 0
	for index, record := range p.records.commands {
		if !record.meta.id.present() || !putRecordIndex(p.commands, record.meta.id, index) {
			return errInvalidModelTopology
		}
		p.commandOrder = append(p.commandOrder, record.meta.id)
		if parent, ok := record.Parent(); ok {
			p.childrenByCommand[parent] = append(p.childrenByCommand[parent], record.meta.id)
		} else {
			p.root = record.meta.id
			rootCount++
		}
	}
	if rootCount != 1 {
		return errInvalidModelTopology
	}
	for _, record := range p.records.commands {
		if parent, ok := record.Parent(); ok {
			if _, exists := p.commands[parent]; !exists {
				return errInvalidModelTopology
			}
		}
		if _, ok := p.grammar.Dialect(record.meta.id); !ok {
			return errInvalidModelTopology
		}
		for _, entry := range p.grammar.CommandEntries(record.meta.id) {
			if _, duplicate := p.entryGrammar[entry.id]; duplicate {
				return errInvalidModelTopology
			}
			p.entryGrammar[entry.id] = entry
		}
		for _, entry := range p.grammar.ActionCommands(record.meta.id) {
			if _, duplicate := p.actionCommandGrammar[entry.id]; duplicate {
				return errInvalidModelTopology
			}
			p.actionCommandGrammar[entry.id] = entry
		}
		for _, entry := range p.grammar.ActionFlags(record.meta.id) {
			if _, duplicate := p.actionFlagGrammar[entry.id]; duplicate {
				return errInvalidModelTopology
			}
			p.actionFlagGrammar[entry.id] = entry
		}
		for _, flag := range p.grammar.Flags(record.meta.id) {
			if _, duplicate := p.flagGrammar[flag.id]; duplicate {
				return errInvalidModelTopology
			}
			p.flagGrammar[flag.id] = flag
			for _, spelling := range flag.Spellings() {
				if _, duplicate := p.spellingGrammar[spelling.id]; duplicate {
					return errInvalidModelTopology
				}
				p.spellingGrammar[spelling.id] = spelling
			}
		}
		for _, form := range p.grammar.ArgumentForms(record.meta.id) {
			if _, duplicate := p.formGrammar[form.id]; duplicate {
				return errInvalidModelTopology
			}
			p.formGrammar[form.id] = form
			for _, term := range form.Terms() {
				if _, duplicate := p.termGrammar[term.id]; duplicate {
					return errInvalidModelTopology
				}
				p.termGrammar[term.id] = term
			}
		}
	}

	for index, record := range p.records.destinations {
		if _, ok := p.commands[record.command]; !ok || !putRecordIndex(p.destinations, record.meta.id, index) {
			return errInvalidModelTopology
		}
		if _, ok := record.Definition(); !ok {
			return errInvalidModelTopology
		}
		p.destinationsByCommand[record.command] = append(p.destinationsByCommand[record.command], record.meta.id)
	}
	for index, record := range p.records.actionEntries {
		if _, ok := p.commands[record.parent]; !ok || !putRecordIndex(p.actionEntries, record.meta.id, index) {
			return errInvalidModelTopology
		}
		commandGrammar, hasCommandGrammar := p.actionCommandGrammar[record.meta.id]
		flagGrammar, hasFlagGrammar := p.actionFlagGrammar[record.meta.id]
		if hasCommandGrammar == hasFlagGrammar {
			return errInvalidModelTopology
		}
		if hasCommandGrammar && (commandGrammar.parent != record.parent || commandGrammar.position != record.meta.position) {
			return errInvalidModelTopology
		}
		if hasFlagGrammar && (flagGrammar.parent != record.parent || flagGrammar.position != record.meta.position) {
			return errInvalidModelTopology
		}
		p.actionEntriesByParent[record.parent] = append(p.actionEntriesByParent[record.parent], record.meta.id)
	}
	for index, record := range p.records.commandEntries {
		if _, ok := p.commands[record.parent]; !ok {
			return errInvalidModelTopology
		}
		if _, ok := p.commands[record.target]; !ok || !putRecordIndex(p.commandEntries, record.meta.id, index) {
			return errInvalidModelTopology
		}
		grammar, ok := p.entryGrammar[record.meta.id]
		if !ok || grammar.parent != record.parent || grammar.target != record.target || grammar.position != record.meta.position {
			return errInvalidModelTopology
		}
		if grammar.kind == CommandEntryCanonical {
			if _, duplicate := p.canonicalEntries[record.target]; duplicate {
				return errInvalidModelTopology
			}
			p.canonicalEntries[record.target] = record.meta.id
		}
		p.entriesByParent[record.parent] = append(p.entriesByParent[record.parent], record.meta.id)
	}
	for index, record := range p.records.flags {
		destination, ok := p.destination(record.destination)
		if _, commandExists := p.commands[record.command]; !commandExists || !ok || destination.command != record.command || !putRecordIndex(p.flags, record.meta.id, index) {
			return errInvalidModelTopology
		}
		grammar, ok := p.flagGrammar[record.meta.id]
		if !ok || grammar.command != record.command || grammar.destination != record.destination || grammar.position != record.meta.position {
			return errInvalidModelTopology
		}
		p.flagsByCommand[record.command] = append(p.flagsByCommand[record.command], record.meta.id)
	}
	for index, record := range p.records.flagSpellings {
		if _, ok := p.flags[record.flag]; !ok || !putRecordIndex(p.flagSpellings, record.meta.id, index) {
			return errInvalidModelTopology
		}
		grammar, ok := p.spellingGrammar[record.meta.id]
		if !ok || grammar.flag != record.flag || grammar.position != record.meta.position {
			return errInvalidModelTopology
		}
		p.spellingsByFlag[record.flag] = append(p.spellingsByFlag[record.flag], record.meta.id)
	}
	for index, record := range p.records.argumentTerms {
		if !putRecordIndex(p.argumentTerms, record.meta.id, index) {
			return errInvalidModelTopology
		}
		destination, hasDestination := record.Destination()
		if hasDestination {
			if _, ok := p.destinations[destination]; !ok {
				return errInvalidModelTopology
			}
		}
		grammar, ok := p.termGrammar[record.meta.id]
		grammarDestination, grammarHasDestination := grammar.Destination()
		if !ok || grammar.position != record.meta.position || grammarHasDestination != hasDestination || (hasDestination && grammarDestination != destination) {
			return errInvalidModelTopology
		}
		p.termsByForm[record.form] = append(p.termsByForm[record.form], record.meta.id)
	}
	for form, terms := range p.termsByForm {
		sortByDeclarationPosition(terms, func(id ArgumentTermID) declarationPosition {
			record, _ := p.argumentTerm(id)
			return record.meta.position
		})
		p.termsByForm[form] = terms
	}
	for index, record := range p.records.argumentForms {
		if _, ok := p.commands[record.command]; !ok || !putRecordIndex(p.argumentForms, record.meta.id, index) {
			return errInvalidModelTopology
		}
		grammar, ok := p.formGrammar[record.meta.id]
		if !ok || grammar.command != record.command || grammar.position != record.meta.position {
			return errInvalidModelTopology
		}
		terms := p.termsByForm[record.meta.id]
		grammarTerms := grammar.Terms()
		if len(grammarTerms) != len(terms) {
			return errInvalidModelTopology
		}
		for index, term := range grammarTerms {
			if term.id != terms[index] {
				return errInvalidModelTopology
			}
		}
		p.formsByCommand[record.command] = append(p.formsByCommand[record.command], record.meta.id)
	}
	for _, record := range p.records.argumentTerms {
		form, ok := p.argumentForm(record.form)
		if !ok {
			return errInvalidModelTopology
		}
		if destination, present := record.Destination(); present {
			destinationRecord, _ := p.destination(destination)
			if destinationRecord.command != form.command {
				return errInvalidModelTopology
			}
		}
	}
	if len(p.flagGrammar) != len(p.records.flags) || len(p.spellingGrammar) != len(p.records.flagSpellings) || len(p.formGrammar) != len(p.records.argumentForms) || len(p.termGrammar) != len(p.records.argumentTerms) {
		return errInvalidModelTopology
	}

	for index, record := range p.records.defaults {
		if _, ok := p.commands[record.parent]; !ok {
			return errInvalidModelTopology
		}
		if _, ok := p.commands[record.target]; !ok || !putRecordIndex(p.defaults, record.meta.id, index) {
			return errInvalidModelTopology
		}
		p.defaultsByParent[record.parent] = append(p.defaultsByParent[record.parent], record.meta.id)
	}
	for index, record := range p.records.providers {
		if _, ok := p.commands[record.parent]; !ok || !record.definition.Valid() || !putRecordIndex(p.providers, record.meta.id, index) {
			return errInvalidModelTopology
		}
		p.providersByParent[record.parent] = append(p.providersByParent[record.parent], record.meta.id)
	}
	for index, record := range p.records.rules {
		if _, ok := p.commands[record.command]; !ok || !record.definition.Valid() || !putRecordIndex(p.rules, record.meta.id, index) {
			return errInvalidModelTopology
		}
		p.rulesByCommand[record.command] = append(p.rulesByCommand[record.command], record.meta.id)
	}
	for index, record := range p.records.advisories {
		if _, ok := p.commands[record.command]; !ok || !record.definition.Valid() || !putRecordIndex(p.advisories, record.meta.id, index) {
			return errInvalidModelTopology
		}
		p.advisoriesByCommand[record.command] = append(p.advisoriesByCommand[record.command], record.meta.id)
	}
	for index, record := range p.records.signalPolicies {
		if _, ok := p.commands[record.command]; !ok || !putRecordIndex(p.signalPolicies, record.meta.id, index) {
			return errInvalidModelTopology
		}
		p.signalPoliciesByCommand[record.command] = append(p.signalPoliciesByCommand[record.command], record.meta.id)
		if definition, present := record.Definition(); present {
			counter, hasCounter := record.Counter()
			if !hasCounter || !definition.Valid() {
				return errInvalidModelTopology
			}
			p.counterPolicies[counter] = append(p.counterPolicies[counter], record.meta.id)
			for _, action := range definition.Actions() {
				if endpoint, ok := action.Endpoint(); ok {
					p.endpointPolicies[endpoint] = append(p.endpointPolicies[endpoint], record.meta.id)
				}
			}
		} else if counter, present := record.Counter(); present {
			p.counterPolicies[counter] = append(p.counterPolicies[counter], record.meta.id)
		}
	}
	for index, record := range p.records.domainEntries {
		if !putRecordIndex(p.domainEntries, record.meta.id, index) {
			return errInvalidModelTopology
		}
	}
	for _, record := range p.records.valueDomains {
		for _, entry := range record.Entries() {
			if _, ok := p.domainEntries[entry]; !ok {
				return errInvalidModelTopology
			}
		}
		for _, use := range record.Uses() {
			if _, ok := p.destinations[use.destination]; !ok {
				return errInvalidModelTopology
			}
		}
	}
	for index, record := range p.records.environmentInputs {
		destination, ok := p.destination(record.destination)
		if _, commandExists := p.commands[record.command]; !commandExists || !ok || destination.command != record.command || !putRecordIndex(p.environmentInputs, record.meta.id, index) {
			return errInvalidModelTopology
		}
		p.environmentByDestination[record.destination] = append(p.environmentByDestination[record.destination], record.meta.id)
	}
	for index, record := range p.records.grammarProfiles {
		if _, ok := p.commands[record.command]; !ok || !putRecordIndex(p.grammarProfiles, record.meta.id, index) {
			return errInvalidModelTopology
		}
		selector, ok := record.Selector()
		if !ok {
			return errInvalidModelTopology
		}
		destination, ok := p.destination(selector)
		if !ok || destination.command != record.command || destination.meta.profile.present() {
			return errInvalidModelTopology
		}
		if !record.defaultCase {
			value, present := record.Value()
			definition, defined := destination.Definition()
			if !present || !defined || definition.Kind() != destinationKindScalar {
				return errInvalidModelTopology
			}
			language, present := definition.Language()
			if !present {
				return errInvalidModelTopology
			}
			if _, err := language.Format(value); err != nil {
				return errInvalidModelTopology
			}
		}
		if dialect, present := record.Dialect(); !present || !dialectValid(dialect) {
			return errInvalidModelTopology
		}
		p.profilesByCommand[record.command] = append(p.profilesByCommand[record.command], record.meta.id)
	}
	for index, record := range p.records.signalEndpoints {
		if _, ok := p.commands[record.command]; !ok || !putRecordIndex(p.signalEndpoints, record.meta.id, index) {
			return errInvalidModelTopology
		}
	}
	for index, record := range p.records.signalCounters {
		if _, ok := p.signalPolicies[record.policy]; !ok || !putRecordIndex(p.signalCounters, record.meta.id, index) {
			return errInvalidModelTopology
		}
	}

	for _, scope := range p.records.signalPolicyScopes {
		if _, ok := p.commands[scope.Command()]; !ok {
			return errInvalidModelTopology
		}
		if profileID := scope.Profile(); profileID.present() {
			profile, ok := p.grammarProfile(profileID)
			if !ok || profile.command != scope.Command() {
				return errInvalidModelTopology
			}
		}
	}
	for index, group := range p.records.helpGroups {
		if _, ok := p.commands[group.command]; !ok || !group.Valid() {
			return errInvalidModelTopology
		}
		if group.profile.present() {
			profile, ok := p.grammarProfile(group.profile)
			if !ok || profile.command != group.command {
				return errInvalidModelTopology
			}
		}
		switch group.family {
		case helpGroupFamilyUnknown:
			return errInvalidModelTopology
		case helpGroupFamilyCommand:
			for _, entry := range group.CommandEntries() {
				switch entry.Kind() {
				case HelpCommandGroupEntryCommandEntry:
					record, ok := p.commandEntry(entry.CommandEntry())
					if !ok || record.parent != group.command {
						return errInvalidModelTopology
					}
				case HelpCommandGroupEntryCommandProvider:
					record, ok := p.provider(entry.Provider())
					if !ok || record.parent != group.command {
						return errInvalidModelTopology
					}
				default:
					return errInvalidModelTopology
				}
			}
		case helpGroupFamilyFlag:
			for _, flagID := range group.Flags() {
				record, ok := p.flag(flagID)
				if !ok || record.command != group.command {
					return errInvalidModelTopology
				}
			}
		default:
			return errInvalidModelTopology
		}
		p.helpGroupsByCommand[group.command] = append(p.helpGroupsByCommand[group.command], index)
	}
	for index, record := range p.records.conditionPolicies {
		command, ok := p.conditionTargetCommand(record.target)
		if !record.condition.Valid() || !ok {
			return errInvalidModelTopology
		}
		p.conditionPoliciesByCommand[command] = append(p.conditionPoliciesByCommand[command], index)
	}
	for index, record := range p.records.noticeSuppressions {
		if _, ok := p.commands[record.command]; !ok || !record.match.Valid() || !record.condition.Valid() {
			return errInvalidModelTopology
		}
		p.noticeSuppressionsByCommand[record.command] = append(p.noticeSuppressionsByCommand[record.command], index)
	}

	for _, ids := range p.childrenByCommand {
		p.sortCommands(ids)
	}
	for flag, ids := range p.spellingsByFlag {
		sortByDeclarationPosition(ids, func(id FlagSpellingID) declarationPosition {
			record, _ := p.flagSpelling(id)
			return record.meta.position
		})
		p.spellingsByFlag[flag] = ids
	}
	for destination, ids := range p.environmentByDestination {
		sortByDeclarationPosition(ids, func(id EnvironmentInputID) declarationPosition {
			record, _ := p.environmentInput(id)
			return record.meta.position
		})
		p.environmentByDestination[destination] = ids
	}
	for command, ids := range p.profilesByCommand {
		sortByDeclarationPosition(ids, func(id GrammarProfileID) declarationPosition {
			record, _ := p.grammarProfile(id)
			return record.meta.position
		})
		p.profilesByCommand[command] = ids
	}
	p.sortCommands(p.commandOrder)
	for _, id := range p.commandOrder {
		if _, err := p.commandPath(id); err != nil {
			return err
		}
	}
	return nil
}

func dialectValid(value dialect) bool {
	return value.FlagOrder().valid() && value.ShortClustering().valid() && value.FlagMatching().valid() && value.CommandMatching().valid() && value.FlagNormalization().valid() && value.CommandNormalization().valid() && value.DashlessFlags().valid()
}

func (p *staticTopology) conditionTargetCommand(target conditionTarget) (CommandID, bool) {
	switch target.Kind() {
	case conditionTargetCommand:
		_, ok := p.commands[target.Command()]
		return target.Command(), ok
	case conditionTargetFlag:
		record, ok := p.flag(target.Flag())
		return record.command, ok
	case conditionTargetArgumentForm:
		record, ok := p.argumentForm(target.ArgumentForm())
		return record.command, ok
	case conditionTargetArgumentTerm:
		record, ok := p.argumentTerm(target.ArgumentTerm())
		if !ok {
			return 0, false
		}
		form, ok := p.argumentForm(record.form)
		return form.command, ok
	default:
		return 0, false
	}
}

func (p *staticTopology) sortCommands(ids []CommandID) {
	sortByDeclarationPosition(ids, func(id CommandID) declarationPosition {
		record, _ := p.command(id)
		return record.meta.position
	})
}

func sortByDeclarationPosition[T comparable](values []T, position func(T) declarationPosition) {
	positions := make(map[T]declarationPosition, len(values))
	for _, value := range values {
		positions[value] = position(value)
	}
	slices.SortStableFunc(values, func(left, right T) int {
		return cmp.Compare(positions[left], positions[right])
	})
}

func (p *staticTopology) commandPath(id CommandID) ([]string, error) {
	if path, ok := p.commandPaths[id]; ok {
		return slices.Clone(path), nil
	}
	record, ok := p.command(id)
	if !ok {
		return nil, errInvalidModelTopology
	}
	if id == p.root {
		p.commandPaths[id] = []string{}
		return []string{}, nil
	}
	entryID, ok := p.canonicalEntries[id]
	if !ok {
		return nil, errInvalidModelTopology
	}
	entry, ok := p.commandEntry(entryID)
	if !ok {
		return nil, errInvalidModelTopology
	}
	parent, present := record.Parent()
	if !present {
		return nil, errInvalidModelTopology
	}
	parentPath, err := p.commandPath(parent)
	if err != nil {
		return nil, err
	}
	grammar := p.entryGrammar[entry.meta.id]
	path := slices.Clone(parentPath)
	path = append(path, grammar.name)
	p.commandPaths[id] = slices.Clone(path)
	return path, nil
}

func (p *staticTopology) command(id CommandID) (commandRecord, bool) {
	index, ok := p.commands[id]
	if !ok {
		return commandRecord{}, false
	}
	return p.records.commands[index], true
}

func (p *staticTopology) destination(id ValueID) (destinationRecord, bool) {
	index, ok := p.destinations[id]
	if !ok {
		return destinationRecord{}, false
	}
	return p.records.destinations[index], true
}

func (p *staticTopology) flag(id FlagID) (flagRecord, bool) {
	index, ok := p.flags[id]
	if !ok {
		return flagRecord{}, false
	}
	return p.records.flags[index], true
}

func (p *staticTopology) flagSpelling(id FlagSpellingID) (flagSpellingRecord, bool) {
	index, ok := p.flagSpellings[id]
	if !ok {
		return flagSpellingRecord{}, false
	}
	return p.records.flagSpellings[index], true
}

func (p *staticTopology) argumentForm(id ArgumentFormID) (argumentFormRecord, bool) {
	index, ok := p.argumentForms[id]
	if !ok {
		return argumentFormRecord{}, false
	}
	return p.records.argumentForms[index], true
}

func (p *staticTopology) argumentTerm(id ArgumentTermID) (argumentTermRecord, bool) {
	index, ok := p.argumentTerms[id]
	if !ok {
		return argumentTermRecord{}, false
	}
	return p.records.argumentTerms[index], true
}

func (p *staticTopology) commandEntry(id CommandEntryID) (commandEntryRecord, bool) {
	index, ok := p.commandEntries[id]
	if !ok {
		return commandEntryRecord{}, false
	}
	return p.records.commandEntries[index], true
}

func (p *staticTopology) provider(id CommandProviderID) (providerRecord, bool) {
	index, ok := p.providers[id]
	if !ok {
		return providerRecord{}, false
	}
	return p.records.providers[index], true
}

func (p *staticTopology) grammarProfile(id GrammarProfileID) (grammarProfileRecord, bool) {
	index, ok := p.grammarProfiles[id]
	if !ok {
		return grammarProfileRecord{}, false
	}
	return p.records.grammarProfiles[index], true
}

func (p *staticTopology) environmentInput(id EnvironmentInputID) (environmentInputRecord, bool) {
	index, ok := p.environmentInputs[id]
	if !ok {
		return environmentInputRecord{}, false
	}
	return p.records.environmentInputs[index], true
}

func (p *schemaProjector) newSchema() *Schema {
	result := &Schema{
		Localization:       publicLocalization(p.application.localization),
		Root:               p.root,
		CommandOrder:       slices.Clone(p.commandOrder),
		Commands:           make(map[CommandID]CommandSchema, len(p.records.commands)),
		CommandEntries:     make(map[CommandEntryID]CommandEntrySchema, len(p.records.commandEntries)),
		ActionEntries:      make(map[ActionEntryID]ActionEntrySchema, len(p.records.actionEntries)),
		CommandDefaults:    make(map[CommandDefaultID]CommandDefaultSchema, len(p.records.defaults)),
		CommandProviders:   make(map[CommandProviderID]CommandProviderSchema, len(p.records.providers)),
		Flags:              make(map[FlagID]FlagSchema, len(p.records.flags)),
		FlagSpellings:      make(map[FlagSpellingID]FlagSpellingSchema, len(p.records.flagSpellings)),
		ArgumentForms:      make(map[ArgumentFormID]ArgumentFormSchema, len(p.records.argumentForms)),
		ArgumentTerms:      make(map[ArgumentTermID]ArgumentTermSchema, len(p.records.argumentTerms)),
		Values:             make(map[ValueID]ValueSchema, len(p.records.destinations)),
		Facts:              make(map[ValueID]FactSchema),
		EnvironmentInputs:  make(map[EnvironmentInputID]EnvironmentInputSchema, len(p.records.environmentInputs)),
		ValueDomains:       make([]ValueDomainSchema, 0, len(p.records.valueDomains)),
		ValueDomainEntries: make(map[ValueDomainEntryID]ValueDomainEntrySchema, len(p.records.domainEntries)),
		SignalPolicies:     make(map[SignalPolicyID]SignalPolicySchema, len(p.records.signalPolicies)),
		SignalEndpoints:    make(map[SignalEndpointID]SignalEndpoint, len(p.records.signalEndpoints)),
	}
	if p.application.release.valid() {
		result.EvaluatedRelease = p.application.release.current()
	}
	return result
}

func (p *schemaProjector) project() (*Schema, error) {
	result := p.newSchema()
	if err := p.projectCommands(result); err != nil {
		return nil, err
	}
	if err := p.projectCommandEntries(result); err != nil {
		return nil, err
	}
	if err := p.projectActionsAndGrammar(result); err != nil {
		return nil, err
	}
	if err := p.projectValuesAndDomains(result); err != nil {
		return nil, err
	}
	if err := p.projectStaticRelations(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *staticArtifactSource) lifecycleState(lifecycle ReleaseLifecycle) (LifecycleState, error) {
	state, err := evaluateLifecycle(p.application.release, lifecycle, "")
	if err != nil {
		return LifecycleStateUnknown, err
	}
	switch state {
	case LifecycleStateNotIntroduced, LifecycleStateCurrent, LifecycleStateDeprecated, LifecycleStateRemoved:
		return state, nil
	default:
		return LifecycleStateUnknown, errInvalidModelTopology
	}
}

func projectExecutionAccessRecord(access executionAccess) ExecutionAccess {
	result := ExecutionAccess{}
	for _, value := range access.Values() {
		result.Values = append(result.Values, value.Source())
	}
	for _, value := range access.OptionalValues() {
		result.OptionalValues = append(result.OptionalValues, value.Source())
	}
	for _, value := range access.ValueStates() {
		result.ValueStates = append(result.ValueStates, value.Source())
	}
	_, result.Locale = access.Locale()
	_, result.CommandSelection = access.CommandSelection()
	_, result.WorkingDirectory = access.WorkingDirectory()
	_, result.Stdin = access.Stdin()
	_, result.Stdout = access.Stdout()
	_, result.Stderr = access.Stderr()
	return result
}

func projectPublicDialectRecord(value dialect) (CommandDialect, error) {
	if !dialectValid(value) {
		return CommandDialect{}, errInvalidModelTopology
	}
	dialect := CommandDialect{
		FlagOrder:            value.FlagOrder(),
		ShortFlagClustering:  value.ShortClustering(),
		FlagMatching:         value.FlagMatching(),
		CommandMatching:      value.CommandMatching(),
		FlagNormalization:    value.FlagNormalization(),
		CommandNormalization: value.CommandNormalization(),
		DashlessFlags:        value.DashlessFlags(),
		EndOfOptions:         value.EndOfOptions(),
		ResponseFiles:        value.ResponseFiles(),
	}
	return dialect, nil
}

func (p *schemaProjector) projectCommands(result *Schema) error {
	for _, id := range p.commandOrder {
		if err := p.projectCommand(
			result,
			id,
			slices.Clone(p.flagsByCommand[id]),
			slices.Clone(p.entriesByParent[id]),
		); err != nil {
			return err
		}
	}
	return nil
}

func (p *schemaProjector) projectCommand(result *Schema, id CommandID, flags []FlagID, entries []CommandEntryID) error {
	record, _ := p.command(id)
	path, err := p.commandPath(id)
	if err != nil {
		return err
	}
	name := p.application.name
	canonicalEntry := CommandEntryID(0)
	state := LifecycleStateCurrent
	if id != p.root {
		entryID, ok := p.canonicalEntries[id]
		if !ok {
			return errInvalidModelTopology
		}
		entry, _ := p.commandEntry(entryID)
		grammar := p.entryGrammar[entryID]
		canonicalEntry = entryID
		name = grammar.name
		state, err = p.lifecycleState(entry.lifecycle)
		if err != nil {
			return err
		}
	}
	dialect, _ := p.grammar.Dialect(id)
	publicDialect, err := projectPublicDialectRecord(dialect)
	if err != nil {
		return err
	}
	profiles, present, err := p.projectGrammarProfileSet(id)
	if err != nil {
		return err
	}
	var profileSet *GrammarProfileSet
	if present {
		profileSet = &profiles
	}
	presentation := record.presentation
	result.Commands[id] = CommandSchema{
		ID:               id,
		CanonicalEntry:   canonicalEntry,
		Parent:           record.parent.id,
		Children:         slices.Clone(p.childrenByCommand[id]),
		Origin:           DeclarationOriginApplication,
		Name:             name,
		Path:             path,
		Description:      presentation.description,
		Examples:         presentation.Examples(),
		Hidden:           presentation.hidden,
		Runnable:         record.execution.Runnable(),
		LifecycleState:   state,
		Access:           projectExecutionAccessRecord(record.access),
		Dialect:          &publicDialect,
		Profiles:         profileSet,
		Help:             p.projectHelp(id),
		Flags:            flags,
		ArgumentForms:    slices.Clone(p.formsByCommand[id]),
		SignalPolicies:   slices.Clone(p.signalPoliciesByCommand[id]),
		CommandEntries:   entries,
		ActionEntries:    slices.Clone(p.actionEntriesByParent[id]),
		CommandDefaults:  slices.Clone(p.defaultsByParent[id]),
		CommandProviders: slices.Clone(p.providersByParent[id]),
	}
	return nil
}

func (p *schemaProjector) projectCommandEntries(result *Schema) error {
	for _, record := range p.records.commandEntries {
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		grammar := p.entryGrammar[record.meta.id]
		public := CommandEntrySchema{
			ID:               record.meta.id,
			Parent:           record.parent,
			Kind:             grammar.kind,
			Origin:           DeclarationOriginApplication,
			DeclarationIndex: int(record.meta.position),
			Profile:          record.meta.profile,
			Name:             grammar.name,
			Description:      record.presentation.description,
			Examples:         record.presentation.Examples(),
			Hidden:           record.presentation.hidden,
			Release:          releaseLifecycle(record.lifecycle),
			LifecycleState:   state,
		}
		if shortcut, ok := record.Shortcut(); ok {
			public.Args = shortcut.Args()
			public.AppendArgs = shortcut.AppendArgs()
			public.Defaults, err = projectShortcutDefaults(shortcut, p.destination)
			if err != nil {
				return err
			}
		}
		public.TargetPath = p.pathCommandIDs(record.target)
		result.CommandEntries[record.meta.id] = public
	}
	return nil
}

func (p *staticArtifactSource) pathCommandIDs(target CommandID) []CommandID {
	var reverse []CommandID
	for target.present() {
		reverse = append(reverse, target)
		if target == p.root {
			break
		}
		record, ok := p.command(target)
		if !ok {
			return nil
		}
		parent, ok := record.Parent()
		if !ok {
			return nil
		}
		target = parent
	}
	result := make([]CommandID, len(reverse))
	for index := range reverse {
		result[len(reverse)-1-index] = reverse[index]
	}
	return result
}

func projectShortcutDefaults(shortcut shortcutDefinition, destination func(ValueID) (destinationRecord, bool)) ([]CommandShortcutDefaultSchema, error) {
	var result []CommandShortcutDefaultSchema
	for _, value := range shortcut.Defaults() {
		if !value.Valid() {
			return nil, errInvalidModelTopology
		}
		sensitive := value.Source().Sensitive()
		projected := CommandShortcutDefaultSchema{Sensitive: sensitive}
		switch value.Kind() {
		case shortcutDefaultKindDestination:
			record, ok := destination(value.Destination())
			if !ok {
				return nil, errInvalidModelTopology
			}
			values, err := projectShortcutDefaultRecord(record, value.Value(), sensitive)
			if err != nil {
				return nil, err
			}
			projected.Destination = value.Destination()
			projected.Values = values
		case shortcutDefaultKindFlag:
			projected.Spelling = value.Spelling()
			for _, text := range value.Values() {
				literal := ValueLiteral{
					Type:      ValueType{Kind: ValueKindString},
					Text:      text,
					Sensitive: sensitive,
				}
				if sensitive {
					literal.Text = ""
					literal.Redacted = true
				}
				projected.Values = append(projected.Values, literal)
			}
		default:
			return nil, errInvalidModelTopology
		}
		result = append(result, projected)
	}
	return result, nil
}

func projectShortcutDefaultRecord(record destinationRecord, value any, sensitive bool) ([]ValueLiteral, error) {
	definition, ok := record.Definition()
	if !ok || definition.Kind() != destinationKindScalar {
		return nil, errInvalidModelTopology
	}
	literal, err := projectValueLiteralRecord(record, value, sensitive)
	if err != nil {
		return nil, err
	}
	return []ValueLiteral{literal}, nil
}

func projectValueLiteralRecord(record destinationRecord, value any, sensitive bool) (ValueLiteral, error) {
	definition, ok := record.Definition()
	if !ok {
		return ValueLiteral{}, errInvalidModelTopology
	}
	valueType, ok := definition.ValueType()
	if !ok {
		return ValueLiteral{}, errInvalidModelTopology
	}
	language, ok := definition.Language()
	if !ok {
		return ValueLiteral{}, errInvalidModelTopology
	}
	if sensitive {
		return ValueLiteral{
			Type:      valueType,
			Sensitive: true,
			Redacted:  true,
		}, nil
	}
	text, err := language.Format(value)
	if err != nil {
		return ValueLiteral{}, errInvalidModelTopology
	}
	return ValueLiteral{
		Type: valueType,
		Text: text,
	}, nil
}

func (p *staticArtifactSource) projectGrammarProfileSet(command CommandID) (GrammarProfileSet, bool, error) {
	ids := p.profilesByCommand[command]
	if len(ids) == 0 {
		return GrammarProfileSet{}, false, nil
	}
	set := GrammarProfileSet{Profiles: make(map[GrammarProfileID]GrammarProfile, len(ids))}
	var defaultID GrammarProfileID
	for _, id := range ids {
		record, ok := p.grammarProfile(id)
		if !ok {
			return GrammarProfileSet{}, false, errInvalidModelTopology
		}
		selector, _ := record.Selector()
		if set.Selector == 0 {
			set.Selector = selector
		} else if set.Selector != selector {
			return GrammarProfileSet{}, false, errInvalidModelTopology
		}
		dialect, _ := record.Dialect()
		publicDialect, err := projectPublicDialectRecord(dialect)
		if err != nil {
			return GrammarProfileSet{}, false, err
		}
		value := ""
		if !record.defaultCase {
			raw, _ := record.Value()
			destination, _ := p.destination(selector)
			definition, _ := destination.Definition()
			language, _ := definition.Language()
			value, err = language.Format(raw)
			if err != nil {
				return GrammarProfileSet{}, false, errInvalidModelTopology
			}
		}
		item := GrammarProfile{
			ID:           record.meta.id,
			Value:        value,
			Origin:       DeclarationOriginApplication,
			Reachability: p.analysis.GrammarProfile(record.meta.id),
			Dialect:      publicDialect,
		}
		set.Profiles[item.ID] = item
		if record.defaultCase {
			if set.Default != 0 {
				return GrammarProfileSet{}, false, errInvalidModelTopology
			}
			set.Default = item.ID
			defaultID = item.ID
		} else {
			set.Cases = append(set.Cases, item.ID)
		}
	}
	if set.Selector == 0 {
		return GrammarProfileSet{}, false, errInvalidModelTopology
	}
	if set.Default != 0 {
		set.DefaultRedundant = p.analysis.GrammarProfile(defaultID) == ReachabilityUnreachable
	}
	return set, true, nil
}

func (p *staticArtifactSource) projectHelp(command CommandID) HelpSchema {
	var result HelpSchema
	for _, position := range p.helpGroupsByCommand[command] {
		group := p.records.helpGroups[position]
		switch group.family {
		case helpGroupFamilyUnknown:
			panic("cli: accepted help group has unknown family")
		case helpGroupFamilyCommand:
			projected := HelpCommandGroupSchema{
				Kind:    HelpGroupDeclared,
				Title:   group.title,
				Order:   group.order,
				Profile: group.profile,
			}
			for _, entry := range group.CommandEntries() {
				item := HelpCommandGroupEntrySchema{Profile: group.profile}
				switch entry.Kind() {
				case HelpCommandGroupEntryCommandEntry:
					item.Kind = HelpCommandGroupEntryCommandEntry
					item.CommandEntry = entry.CommandEntry()
				case HelpCommandGroupEntryCommandProvider:
					item.Kind = HelpCommandGroupEntryCommandProvider
					item.Provider = entry.Provider()
				default:
					continue
				}
				projected.Entries = append(projected.Entries, item)
			}
			result.CommandGroups = append(result.CommandGroups, projected)
		case helpGroupFamilyFlag:
			projected := HelpFlagGroupSchema{
				Kind:    HelpGroupDeclared,
				Title:   group.title,
				Order:   group.order,
				Profile: group.profile,
				Flags:   group.Flags(),
			}
			result.FlagGroups = append(result.FlagGroups, projected)
		}
	}
	return result
}

func (p *schemaProjector) projectActionsAndGrammar(result *Schema) error {
	for _, record := range p.records.actionEntries {
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		public := ActionEntrySchema{
			ID:               record.meta.id,
			Parent:           record.parent,
			Origin:           record.origin,
			DeclarationIndex: int(record.meta.position),
			Description:      record.presentation.description,
			Hidden:           record.presentation.hidden,
			Release:          releaseLifecycle(record.lifecycle),
			LifecycleState:   state,
		}
		if grammar, ok := p.actionCommandGrammar[record.meta.id]; ok {
			public.Kind = ActionEntryCommand
			public.Action = invocationAction(grammar.action)
			public.Name = grammar.name
			public.Placement = FlagPlacementUnknown
		} else if grammar, ok := p.actionFlagGrammar[record.meta.id]; ok {
			public.Kind = ActionEntryFlag
			public.Action = invocationAction(grammar.action)
			public.Placement = grammar.placement
			switch grammar.kind {
			case FlagSpellingLong:
				public.Long = grammar.text
			case FlagSpellingShort:
				runes := []rune(grammar.text)
				if len(runes) != 2 || runes[0] != '-' {
					return errInvalidModelTopology
				}
				public.Short = runes[1]
			case FlagSpellingLiteral:
				public.Literal = grammar.text
			default:
				return errInvalidModelTopology
			}
		} else {
			return errInvalidModelTopology
		}
		result.ActionEntries[record.meta.id] = public
	}

	for _, record := range p.records.flags {
		destination, _ := p.destination(record.destination)
		definition, _ := destination.Definition()
		valueType, hasValueType := definition.ValueType()
		if !hasValueType {
			return errInvalidModelTopology
		}
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		completion, present := record.Completion()
		publicCompletion, err := projectCompletion(completion, present)
		if err != nil {
			return err
		}
		grammar := p.flagGrammar[record.meta.id]
		result.Flags[record.meta.id] = FlagSchema{
			ID:                  record.meta.id,
			Command:             record.command,
			Origin:              DeclarationOriginApplication,
			Profile:             record.meta.profile,
			Destination:         record.destination,
			Spellings:           slices.Clone(p.spellingsByFlag[record.meta.id]),
			ValueName:           record.valueName,
			Description:         record.presentation.description.Short,
			Type:                valueType,
			Occurrences:         countRange(grammar.occurrences),
			ValuesPerOccurrence: countRange(grammar.valuesPerOccurrence),
			Placement:           grammar.placement,
			Required:            record.required,
			Hidden:              record.presentation.hidden,
			Sensitive:           destination.sensitive,
			LocaleSelector:      record.localeSelector,
			Completion:          publicCompletion,
			Release:             releaseLifecycle(record.lifecycle),
			LifecycleState:      state,
		}
	}
	for _, record := range p.records.flagSpellings {
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		grammar := p.spellingGrammar[record.meta.id]
		publicSpelling := FlagSpelling{
			Kind:       grammar.kind,
			Text:       grammar.text,
			ValueMode:  grammar.mode,
			ValueForms: flagValueForms(grammar.Forms()),
			Sensitive:  grammar.sensitive,
		}
		flag, _ := p.flag(record.flag)
		if value, present, sensitive := grammar.FixedValue(); present {
			destination, _ := p.destination(flag.destination)
			literal, err := projectValueLiteralRecord(destination, value, grammar.sensitive || sensitive)
			if err != nil {
				return err
			}
			publicSpelling.FixedValue = &literal
		}
		if value, present, sensitive := grammar.ImplicitValue(); present {
			destination, _ := p.destination(flag.destination)
			literal, err := projectValueLiteralRecord(destination, value, grammar.sensitive || sensitive)
			if err != nil {
				return err
			}
			publicSpelling.ImplicitValue = &literal
		}
		result.FlagSpellings[record.meta.id] = FlagSpellingSchema{
			ID:             record.meta.id,
			Flag:           record.flag,
			Origin:         DeclarationOriginApplication,
			Spelling:       publicSpelling,
			Hidden:         record.hidden,
			Release:        releaseLifecycle(record.lifecycle),
			LifecycleState: state,
		}
	}

	for _, record := range p.records.argumentForms {
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		result.ArgumentForms[record.meta.id] = ArgumentFormSchema{
			ID:             record.meta.id,
			Command:        record.command,
			Origin:         DeclarationOriginApplication,
			Profile:        record.meta.profile,
			Terms:          slices.Clone(p.termsByForm[record.meta.id]),
			Release:        releaseLifecycle(record.lifecycle),
			LifecycleState: state,
			Reachability:   p.analysis.ArgumentForm(record.meta.id),
		}
	}
	for _, record := range p.records.argumentTerms {
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		grammar := p.termGrammar[record.meta.id]
		public := ArgumentTermSchema{
			ID:              record.meta.id,
			Form:            record.form,
			Origin:          DeclarationOriginApplication,
			Kind:            grammar.kind,
			Name:            record.name,
			Literal:         grammar.literal,
			Description:     record.presentation.description.Short,
			Count:           countRange(grammar.count),
			EndOfOptions:    grammar.endOfOptions,
			Hidden:          record.presentation.hidden,
			SourceSensitive: record.sourceSensitive,
			HyphenValues:    grammar.hyphenValues,
			Release:         releaseLifecycle(record.lifecycle),
			LifecycleState:  state,
		}
		public.Optional = grammar.count.Minimum() == 0
		public.Greedy = grammar.allocation == semanticArgumentAllocationGreedy
		public.Minimal = grammar.allocation == semanticArgumentAllocationMinimal
		if destinationID, present := record.Destination(); present {
			destination, _ := p.destination(destinationID)
			definition, _ := destination.Definition()
			valueType, ok := definition.ValueType()
			if !ok {
				return errInvalidModelTopology
			}
			public.Destination = destinationID
			public.Type = valueType
			public.Sensitive = destination.sensitive
			if public.Kind == ArgumentTermValue && valueType.Kind == ValueKindSequence {
				public.Kind = ArgumentTermSequence
			}
		}
		if grammar.tokenName != "" {
			public.TokenLanguage = &TokenLanguageSyntax{Fragments: grammar.TokenSyntax()}
		}
		completion, present := record.Completion()
		public.Completion, err = projectCompletion(completion, present)
		if err != nil {
			return err
		}
		result.ArgumentTerms[record.meta.id] = public
	}
	return nil
}

func projectDestinationSchemaRecord(record destinationRecord, environmentInputs []EnvironmentInputID) (ValueSchema, FactSchema, bool, error) {
	definition, ok := record.Definition()
	if !ok {
		return ValueSchema{}, FactSchema{}, false, errInvalidModelTopology
	}
	kind := definition.Kind()
	valueType, hasValueType := definition.ValueType()
	isFact := kind == destinationKindFactRequired || kind == destinationKindFactResolved
	fact := FactSchema{}
	if isFact {
		fact.ID = record.meta.id
		for _, producer := range record.Producers() {
			if producer.kind == declaredProducerKindFact {
				fact.HasResolver = true
				fact.ResolverSensitive = producer.source.Sensitive()
				fact.Dependencies = dependencyIDs(producer.Dependencies())
			}
		}
	}
	publicKind, ok := projectValueDestinationKind(kind)
	if !ok {
		return ValueSchema{}, FactSchema{}, false, errInvalidModelTopology
	}
	value := ValueSchema{
		ID:                record.meta.id,
		Command:           record.command,
		Origin:            DeclarationOriginApplication,
		Profile:           record.meta.profile,
		Kind:              publicKind,
		HasType:           hasValueType,
		Sensitive:         record.sensitive,
		EnvironmentInputs: append([]EnvironmentInputID(nil), environmentInputs...),
	}
	if hasValueType {
		value.Type = valueType
	}
	switch kind {
	case destinationKindDerived, destinationKindOptionalDerived:
		value.Derived = true
	case destinationKindScalar, destinationKindSequence, destinationKindKeyed, destinationKindOpaque, destinationKindFactRequired, destinationKindFactResolved:
	default:
		return ValueSchema{}, FactSchema{}, false, errInvalidModelTopology
	}
	for _, producer := range record.Producers() {
		switch producer.kind {
		case declaredProducerKindFact:
			if !isFact {
				return ValueSchema{}, FactSchema{}, false, errInvalidModelTopology
			}
			value.Dependencies = appendUniqueValueIDs(value.Dependencies, dependencyIDs(producer.Dependencies())...)
		case declaredProducerKindDefault:
			staticValue, present := producer.Static()
			if !present {
				return ValueSchema{}, FactSchema{}, false, errInvalidModelTopology
			}
			projected, err := projectStaticDefaultRecord(record, staticValue, record.sensitive || producer.source.Sensitive())
			if err != nil {
				return ValueSchema{}, FactSchema{}, false, err
			}
			value.HasStaticDefault = true
			value.StaticDefault = projected
		case declaredProducerKindComputedDefault, declaredProducerKindOptionalComputedDefault:
			value.HasComputedDefault = true
			value.ComputedDefaultSensitive = value.ComputedDefaultSensitive || producer.source.Sensitive()
			value.Dependencies = appendUniqueValueIDs(value.Dependencies, dependencyIDs(producer.Dependencies())...)
		case declaredProducerKindDerived, declaredProducerKindOptionalDerived:
			value.Derived = true
			value.DerivationSensitive = value.DerivationSensitive || producer.source.Sensitive()
			value.Dependencies = appendUniqueValueIDs(value.Dependencies, dependencyIDs(producer.Dependencies())...)
		default:
			return ValueSchema{}, FactSchema{}, false, errInvalidModelTopology
		}
	}
	return value, fact, isFact, nil
}

func (p *schemaProjector) projectValuesAndDomains(result *Schema) error {
	for _, record := range p.records.destinations {
		value, fact, isFact, err := projectDestinationSchemaRecord(record, p.environmentByDestination[record.meta.id])
		if err != nil {
			return err
		}
		result.Values[record.meta.id] = value
		command := result.Commands[record.command]
		if isFact {
			result.Facts[record.meta.id] = fact
			command.Facts = append(command.Facts, record.meta.id)
		} else {
			command.Values = append(command.Values, record.meta.id)
		}
		result.Commands[record.command] = command
	}

	for _, record := range p.records.environmentInputs {
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		contribution, ok := valueCandidateContribution(record.contribution)
		if !ok {
			return errInvalidModelTopology
		}
		result.EnvironmentInputs[record.meta.id] = EnvironmentInputSchema{
			ID:             record.meta.id,
			Command:        record.command,
			Destination:    record.destination,
			Origin:         DeclarationOriginApplication,
			Profile:        record.meta.profile,
			Name:           record.selector,
			Prefix:         record.prefix,
			Contribution:   contribution,
			Delimiter:      record.delimiter,
			Sensitive:      record.sensitive,
			Release:        releaseLifecycle(record.lifecycle),
			LifecycleState: state,
		}
	}
	for _, record := range p.records.valueDomains {
		public := ValueDomainSchema{Entries: record.Entries()}
		for _, use := range record.Uses() {
			public.Uses = append(public.Uses, ValueDomainUse{
				Destination: use.destination,
				Kind:        use.kind,
			})
		}
		result.ValueDomains = append(result.ValueDomains, public)
	}
	for _, record := range p.records.domainEntries {
		state, err := p.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		result.ValueDomainEntries[record.meta.id] = ValueDomainEntrySchema{
			ID:             record.meta.id,
			Kind:           valueDomainEntryKind(record.kind),
			Origin:         DeclarationOriginApplication,
			Name:           record.name,
			Canonical:      record.canonical,
			Description:    Description{Short: record.description},
			Hidden:         record.hidden,
			Release:        releaseLifecycle(record.lifecycle),
			LifecycleState: state,
		}
	}
	return nil
}

func (p *schemaProjector) projectStaticRelations(result *Schema) error {
	for _, record := range p.records.defaults {
		public := CommandDefaultSchema{
			ID:               record.meta.id,
			Parent:           record.parent,
			Target:           record.target,
			Origin:           DeclarationOriginApplication,
			DeclarationIndex: int(record.meta.position),
			Profile:          record.meta.profile,
			Reachability:     p.analysis.Default(record.meta.id),
		}
		if condition, present := record.Condition(); present {
			projected, err := p.projectConditionExpression(condition)
			if err != nil {
				return err
			}
			public.Condition = &projected
		}
		result.CommandDefaults[record.meta.id] = public
	}
	for _, record := range p.records.providers {
		result.CommandProviders[record.meta.id] = CommandProviderSchema{
			ID:               record.meta.id,
			Parent:           record.parent,
			DeclarationIndex: int(record.meta.position),
			Profile:          record.meta.profile,
			Dependencies:     dependencyIDs(record.definition.Dependencies()),
		}
	}
	if err := p.projectPolicies(result); err != nil {
		return err
	}
	if err := p.projectSignals(result); err != nil {
		return err
	}
	return p.projectReplacements(result)
}

func (p *staticArtifactSource) destinationSensitive(id ValueID) (bool, error) {
	record, ok := p.destination(id)
	if !ok {
		return false, errInvalidModelTopology
	}
	return record.sensitive, nil
}

func (p *staticArtifactSource) projectConditionExpression(value condition) (ConditionExpression, error) {
	return projectConditionExpressionWith(value, p.destinationSensitive, func(destination ValueID, operand any, sensitive bool) (ValueLiteral, error) {
		record, ok := p.destination(destination)
		if !ok {
			return ValueLiteral{}, errInvalidModelTopology
		}
		return projectValueLiteralRecord(record, operand, sensitive)
	}, p.projectConditionExpression)
}

func (p *staticArtifactSource) projectPolicyConditionExpression(value policyCondition) (ConditionExpression, error) {
	return projectPolicyConditionExpressionWith(value, p.destinationSensitive, func(destination ValueID, operand any, sensitive bool) (ValueLiteral, error) {
		record, ok := p.destination(destination)
		if !ok {
			return ValueLiteral{}, errInvalidModelTopology
		}
		return projectValueLiteralRecord(record, operand, sensitive)
	}, p.projectPolicyConditionExpression)
}

func (p *staticArtifactSource) projectConditionTarget(target conditionTarget) (DeclarationIdentity, CommandID, error) {
	switch target.Kind() {
	case conditionTargetFlag:
		record, ok := p.flag(target.Flag())
		if !ok {
			return DeclarationIdentity{}, 0, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind: DeclarationKindFlag,
			Flag: target.Flag(),
		}, record.command, nil
	case conditionTargetArgumentForm:
		record, ok := p.argumentForm(target.ArgumentForm())
		if !ok {
			return DeclarationIdentity{}, 0, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:         DeclarationKindArgumentForm,
			ArgumentForm: target.ArgumentForm(),
		}, record.command, nil
	case conditionTargetArgumentTerm:
		record, ok := p.argumentTerm(target.ArgumentTerm())
		if !ok {
			return DeclarationIdentity{}, 0, errInvalidModelTopology
		}
		form, ok := p.argumentForm(record.form)
		if !ok {
			return DeclarationIdentity{}, 0, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:         DeclarationKindArgumentTerm,
			ArgumentTerm: target.ArgumentTerm(),
		}, form.command, nil
	case conditionTargetCommand:
		if _, ok := p.command(target.Command()); !ok {
			return DeclarationIdentity{}, 0, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:    DeclarationKindCommand,
			Command: target.Command(),
		}, target.Command(), nil
	default:
		return DeclarationIdentity{}, 0, errInvalidModelTopology
	}
}

func (p *schemaProjector) projectPolicies(result *Schema) error {
	for _, commandID := range p.commandOrder {
		command := result.Commands[commandID]
		for _, position := range p.conditionPoliciesByCommand[commandID] {
			record := p.records.conditionPolicies[position]
			identity, owner, err := p.projectConditionTarget(record.target)
			if err != nil || owner != commandID {
				return errInvalidModelTopology
			}
			expression, err := p.projectConditionExpression(record.condition)
			if err != nil {
				return err
			}
			effect, ok := projectConditionEffect(record.effect)
			if !ok {
				return errInvalidModelTopology
			}
			condition := ConditionSchema{
				Target:       identity,
				Effect:       effect,
				Expression:   expression,
				Reachability: p.analysis.Condition(record.target, record.effect, record.position),
				Profile:      record.profile,
			}
			command.Conditions = append(command.Conditions, condition)
		}
		for _, id := range p.rulesByCommand[commandID] {
			position, ok := p.rules[id]
			if !ok {
				return errInvalidModelTopology
			}
			projected, err := p.projectRule(p.records.rules[position])
			if err != nil {
				return err
			}
			command.Rules = append(command.Rules, projected)
		}
		for _, id := range p.advisoriesByCommand[commandID] {
			position, ok := p.advisories[id]
			if !ok {
				return errInvalidModelTopology
			}
			projected, err := p.projectAdvisory(p.records.advisories[position])
			if err != nil {
				return err
			}
			command.Advisories = append(command.Advisories, projected)
		}
		for _, position := range p.noticeSuppressionsByCommand[commandID] {
			record := p.records.noticeSuppressions[position]
			expression, err := p.projectConditionExpression(record.condition)
			if err != nil {
				return err
			}
			match, ok := projectNoticeMatch(record.match)
			if !ok {
				return errInvalidModelTopology
			}
			suppression := NoticeSuppressionSchema{
				Command:          record.command,
				Origin:           DeclarationOriginApplication,
				DeclarationIndex: int(record.position),
				Profile:          record.profile,
				Match:            match,
				Condition:        expression,
				Reachability:     p.analysis.NoticeSuppression(record.command, record.position),
			}
			command.NoticeSuppressions = append(command.NoticeSuppressions, suppression)
		}
		result.Commands[commandID] = command
	}
	return nil
}

func (p *staticArtifactSource) projectRule(record ruleRecord) (RuleSchema, error) {
	definition := record.definition
	kind := definition.Kind()
	if !kind.valid() {
		return RuleSchema{}, errInvalidModelTopology
	}
	result := RuleSchema{
		ID:               record.meta.id,
		DeclarationIndex: int(record.meta.position),
		Profile:          record.meta.profile,
		Kind:             kind,
		Code:             RuleCode(definition.Code()),
		Subjects:         destinationValueIDs(definition.Subjects()),
		Dependencies:     dependencyIDs(definition.Dependencies()),
		Reachability:     p.analysis.Rule(record.meta.id),
	}
	if kind == RuleMatch {
		result.Pattern = definition.Pattern()
	}
	if kind == RuleItemCount || kind == RuleEntryCount {
		result.Count = countRange(definition.Count())
	}
	subjects := definition.Subjects()
	if len(definition.Operands()) != 0 {
		if len(subjects) == 0 {
			return RuleSchema{}, errInvalidModelTopology
		}
		sensitive, err := p.destinationSensitive(subjects[0])
		if err != nil {
			return RuleSchema{}, err
		}
		result.Sensitive = sensitive
		for _, operand := range definition.Operands() {
			destination, _ := p.destination(subjects[0])
			projected, err := projectRuleOperandRecord(destination, operand)
			if err != nil {
				return RuleSchema{}, err
			}
			result.Operands = append(result.Operands, projected)
		}
	}
	for _, subject := range subjects {
		sensitive, err := p.destinationSensitive(subject)
		if err != nil {
			return RuleSchema{}, err
		}
		result.Sensitive = result.Sensitive || sensitive
	}
	if activation, present := definition.Activation(); present {
		projected, err := p.projectPolicyConditionExpression(activation)
		if err != nil {
			return RuleSchema{}, err
		}
		result.Activation = &projected
		result.Sensitive = result.Sensitive || projected.Sensitive
	}
	return result, nil
}

func (p *staticArtifactSource) projectAdvisory(record advisoryRecord) (AdvisorySchema, error) {
	definition := record.definition
	kind := definition.Kind()
	if !kind.valid() {
		return AdvisorySchema{}, errInvalidModelTopology
	}
	severity, ok := projectNoticeSeverity(definition.Severity())
	if !ok {
		return AdvisorySchema{}, errInvalidModelTopology
	}
	result := AdvisorySchema{
		Code:             AdvisoryCode(definition.Code()),
		Severity:         severity,
		Message:          definition.Message(),
		Reason:           definition.Reason(),
		Hint:             definition.Hint(),
		Command:          record.command,
		Origin:           DeclarationOriginApplication,
		DeclarationIndex: int(record.meta.position),
		Profile:          record.meta.profile,
		Kind:             kind,
		Reachability:     p.analysis.Advisory(record.meta.id),
		Subjects:         destinationValueIDs(definition.Subjects()),
		Dependencies:     dependencyIDs(definition.Dependencies()),
	}
	if activation, present := definition.Activation(); present {
		projected, err := p.projectConditionExpression(activation)
		if err != nil {
			return AdvisorySchema{}, err
		}
		result.Activation = &projected
	}
	return result, nil
}

func (p *schemaProjector) projectSignals(result *Schema) error {
	for _, record := range p.records.signalEndpoints {
		command, ok := result.Commands[record.command]
		if !ok {
			return errInvalidModelTopology
		}
		result.SignalEndpoints[record.meta.id] = SignalEndpoint{
			ID:       record.meta.id,
			Command:  record.command,
			Origin:   command.Origin,
			Capacity: SignalBacklogCapacity,
			Policies: slices.Clone(p.endpointPolicies[record.meta.id]),
		}
	}
	for _, record := range p.records.signalPolicies {
		definition, present := record.Definition()
		if !present || !definition.Valid() {
			return errInvalidModelTopology
		}
		command := result.Commands[record.command]
		projected, err := p.projectSignalPolicy(record, definition, command.Origin)
		if err != nil {
			return err
		}
		result.SignalPolicies[record.meta.id] = projected
	}
	return nil
}

func (p *staticArtifactSource) projectSignalPolicy(record signalPolicyRecord, definition signalPolicyDefinition, origin DeclarationOrigin) (SignalPolicySchema, error) {
	selector, err := projectSignalSelector(definition.Selector())
	if err != nil {
		return SignalPolicySchema{}, err
	}
	aggregation := definition.Aggregation()
	if !aggregation.valid() {
		return SignalPolicySchema{}, errInvalidModelTopology
	}
	occurrence := definition.Occurrence()
	result := SignalPolicySchema{
		ID:       record.meta.id,
		Command:  record.command,
		Origin:   origin,
		Profile:  record.meta.profile,
		Selector: selector,
		Occurrence: SignalOccurrenceRange{
			Minimum: occurrence.Minimum(),
			Maximum: occurrence.Maximum(),
		},
		Aggregation:                aggregation,
		Deferred:                   definition.Deferred(),
		ContinueMatching:           definition.ContinueMatching(),
		Hint:                       definition.Hint(),
		RequiredSourceCapabilities: definition.RequiredSourceCapabilities(),
		RequiredHostCapabilities:   definition.RequiredHostCapabilities(),
		Reachability:               p.analysis.SignalPolicy(record.meta.id),
	}
	if counter, present := record.Counter(); present {
		result.CounterID = counter
	}
	for _, action := range definition.Actions() {
		projected, err := p.projectSignalAction(action)
		if err != nil {
			return SignalPolicySchema{}, err
		}
		result.Actions = append(result.Actions, projected)
		result.Sensitive = result.Sensitive || projected.Sensitive || projected.SignalRedacted || projected.DelayRedacted
	}
	if activation, present := definition.Activation(); present {
		projected, err := p.projectConditionExpression(activation)
		if err != nil {
			return SignalPolicySchema{}, err
		}
		result.Activation = &projected
		result.Sensitive = result.Sensitive || projected.Sensitive
	}
	return result, nil
}

func (p *staticArtifactSource) projectSignalAction(action signalActionDefinition) (SignalAction, error) {
	result, err := projectSignalActionBase(action)
	if err != nil {
		return SignalAction{}, err
	}
	if destination, present := action.SignalValue(); present {
		result.SignalValueID = destination
		sensitive, err := p.destinationSensitive(destination)
		if err != nil {
			return SignalAction{}, err
		}
		result.SignalRedacted = sensitive
		result.Sensitive = result.Sensitive || sensitive
	}
	if destination, present := action.DelayValue(); present {
		result.DelayValueID = destination
		sensitive, err := p.destinationSensitive(destination)
		if err != nil {
			return SignalAction{}, err
		}
		result.DelayRedacted = sensitive
		result.Sensitive = result.Sensitive || sensitive
	}
	return result, nil
}

func (p *schemaProjector) projectReplacements(result *Schema) error {
	for _, relation := range p.records.replacements {
		target, targetLifecycle, err := p.projectReplacementIdentity(relation.target)
		if err != nil {
			return err
		}
		_, sourceLifecycle, err := p.projectReplacementIdentity(relation.source)
		if err != nil {
			return err
		}
		presentation, err := p.replacementPresentation(relation.target)
		if err != nil {
			return err
		}
		replacement := &DeclarationReplacement{
			Target:       target,
			Presentation: presentation,
			TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
				p.application.release,
				sourceLifecycle,
				targetLifecycle,
			),
		}
		if err := setGenericReplacement(result, relation.source, replacement); err != nil {
			return err
		}
	}

	for _, record := range p.records.commandEntries {
		targetID, present := record.Replacement()
		if !present {
			continue
		}
		target, ok := p.commandEntry(targetID)
		if !ok {
			return errInvalidModelTopology
		}
		presentation, err := p.replacementPresentation(commandEntryReplacementIdentity(targetID))
		if err != nil {
			return err
		}
		value := result.CommandEntries[record.meta.id]
		value.Replacement = &DeclarationReplacement{
			Target: DeclarationIdentity{
				Kind:         DeclarationKindCommandEntry,
				CommandEntry: targetID,
			},
			Presentation: presentation,
			TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
				p.application.release,
				record.lifecycle,
				target.lifecycle,
			),
		}
		result.CommandEntries[record.meta.id] = value
	}
	for _, record := range p.records.actionEntries {
		targetID, present := record.Replacement()
		if !present {
			continue
		}
		target, ok := p.actionEntry(targetID)
		if !ok {
			return errInvalidModelTopology
		}
		presentation, err := p.actionEntryReplacementPresentation(target)
		if err != nil {
			return err
		}
		value := result.ActionEntries[record.meta.id]
		value.Replacement = &DeclarationReplacement{
			Target: DeclarationIdentity{
				Kind:        DeclarationKindActionEntry,
				ActionEntry: targetID,
			},
			Presentation: presentation,
			TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
				p.application.release,
				record.lifecycle,
				target.lifecycle,
			),
		}
		result.ActionEntries[record.meta.id] = value
	}
	for _, record := range p.records.domainEntries {
		targetID, present := record.Replacement()
		if !present {
			continue
		}
		target, ok := p.domainEntry(targetID)
		if !ok {
			return errInvalidModelTopology
		}
		presentation, err := p.domainReplacementPresentation(target)
		if err != nil {
			return err
		}
		value := result.ValueDomainEntries[record.meta.id]
		value.Replacement = &DeclarationReplacement{
			Target: DeclarationIdentity{
				Kind:             DeclarationKindValueDomainEntry,
				ValueDomainEntry: targetID,
			},
			Presentation: presentation,
			TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
				p.application.release,
				record.lifecycle,
				target.lifecycle,
			),
		}
		result.ValueDomainEntries[record.meta.id] = value
	}
	return nil
}

func (p *staticArtifactSource) projectReplacementIdentity(identity declarationReplacementIdentity) (DeclarationIdentity, ReleaseLifecycle, error) {
	switch identity.Kind() {
	case declarationReplacementKindCommandEntry:
		record, ok := p.commandEntry(identity.CommandEntry())
		if !ok {
			return DeclarationIdentity{}, ReleaseLifecycle{}, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:         DeclarationKindCommandEntry,
			CommandEntry: identity.CommandEntry(),
		}, record.lifecycle, nil
	case declarationReplacementKindFlag:
		record, ok := p.flag(identity.Flag())
		if !ok {
			return DeclarationIdentity{}, ReleaseLifecycle{}, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind: DeclarationKindFlag,
			Flag: identity.Flag(),
		}, record.lifecycle, nil
	case declarationReplacementKindFlagSpelling:
		record, ok := p.flagSpelling(identity.FlagSpelling())
		if !ok {
			return DeclarationIdentity{}, ReleaseLifecycle{}, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:         DeclarationKindFlagSpelling,
			FlagSpelling: identity.FlagSpelling(),
		}, record.lifecycle, nil
	case declarationReplacementKindArgumentForm:
		record, ok := p.argumentForm(identity.ArgumentForm())
		if !ok {
			return DeclarationIdentity{}, ReleaseLifecycle{}, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:         DeclarationKindArgumentForm,
			ArgumentForm: identity.ArgumentForm(),
		}, record.lifecycle, nil
	case declarationReplacementKindArgumentTerm:
		record, ok := p.argumentTerm(identity.ArgumentTerm())
		if !ok {
			return DeclarationIdentity{}, ReleaseLifecycle{}, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:         DeclarationKindArgumentTerm,
			ArgumentTerm: identity.ArgumentTerm(),
		}, record.lifecycle, nil
	case declarationReplacementKindEnvironmentInput:
		record, ok := p.environmentInput(identity.EnvironmentInput())
		if !ok {
			return DeclarationIdentity{}, ReleaseLifecycle{}, errInvalidModelTopology
		}
		return DeclarationIdentity{
			Kind:             DeclarationKindEnvironmentInput,
			EnvironmentInput: identity.EnvironmentInput(),
		}, record.lifecycle, nil
	default:
		return DeclarationIdentity{}, ReleaseLifecycle{}, errInvalidModelTopology
	}
}

func (p *staticArtifactSource) replacementPresentation(identity declarationReplacementIdentity) (string, error) {
	switch identity.Kind() {
	case declarationReplacementKindCommandEntry:
		record, ok := p.commandEntry(identity.CommandEntry())
		if !ok {
			return "", errInvalidModelTopology
		}
		grammar := p.entryGrammar[record.meta.id]
		if grammar.kind == CommandEntryCanonical {
			path, ok := p.commandPaths[record.target]
			if !ok {
				return "", errInvalidModelTopology
			}
			return strings.Join(path, " "), nil
		}
		parent, ok := p.commandPaths[record.parent]
		if !ok {
			return "", errInvalidModelTopology
		}
		return strings.Join(append(slices.Clone(parent), grammar.name), " "), nil
	case declarationReplacementKindFlag:
		ids := p.spellingsByFlag[identity.Flag()]
		if len(ids) == 0 {
			return "", errInvalidModelTopology
		}
		grammar, ok := p.spellingGrammar[ids[0]]
		if !ok {
			return "", errInvalidModelTopology
		}
		return grammar.text, nil
	case declarationReplacementKindFlagSpelling:
		grammar, ok := p.spellingGrammar[identity.FlagSpelling()]
		if !ok {
			return "", errInvalidModelTopology
		}
		return grammar.text, nil
	case declarationReplacementKindArgumentForm:
		if _, ok := p.argumentForm(identity.ArgumentForm()); !ok {
			return "", errInvalidModelTopology
		}
		terms := p.termsByForm[identity.ArgumentForm()]
		parts := make([]string, 0, len(terms))
		for _, termID := range terms {
			record, _ := p.argumentTerm(termID)
			grammar := p.termGrammar[termID]
			if grammar.kind == ArgumentTermLiteral {
				parts = append(parts, grammar.literal)
			} else {
				parts = append(parts, record.name)
			}
		}
		return "(" + strings.Join(parts, " ") + ")", nil
	case declarationReplacementKindArgumentTerm:
		record, ok := p.argumentTerm(identity.ArgumentTerm())
		if !ok {
			return "", errInvalidModelTopology
		}
		grammar := p.termGrammar[record.meta.id]
		if grammar.kind == ArgumentTermLiteral {
			return grammar.literal, nil
		}
		return record.name, nil
	case declarationReplacementKindEnvironmentInput:
		record, ok := p.environmentInput(identity.EnvironmentInput())
		if !ok {
			return "", errInvalidModelTopology
		}
		if record.prefix {
			return record.selector + "*", nil
		}
		return record.selector, nil
	default:
		return "", errInvalidModelTopology
	}
}

func (p *staticArtifactSource) actionEntryReplacementPresentation(record actionEntryRecord) (string, error) {
	if grammar, ok := p.actionCommandGrammar[record.meta.id]; ok {
		return grammar.name, nil
	}
	if grammar, ok := p.actionFlagGrammar[record.meta.id]; ok {
		return grammar.text, nil
	}
	return "", errInvalidModelTopology
}

func (p *staticArtifactSource) domainReplacementPresentation(record domainEntryRecord) (string, error) {
	if record.kind == domainEntryKindCanonical {
		return record.name, nil
	}
	canonical, ok := p.domainEntry(record.canonical)
	if !ok || canonical.kind != domainEntryKindCanonical {
		return "", errInvalidModelTopology
	}
	return canonical.name, nil
}

func (p *staticArtifactSource) actionEntry(id ActionEntryID) (actionEntryRecord, bool) {
	index, ok := p.actionEntries[id]
	if !ok {
		return actionEntryRecord{}, false
	}
	return p.records.actionEntries[index], true
}

func (p *staticArtifactSource) domainEntry(id ValueDomainEntryID) (domainEntryRecord, bool) {
	index, ok := p.domainEntries[id]
	if !ok {
		return domainEntryRecord{}, false
	}
	return p.records.domainEntries[index], true
}
