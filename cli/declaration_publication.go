package cli

import (
	"cmp"
	"maps"
	"slices"
)

// declarationDelta is the immutable typed declaration contribution accepted
// from one operation-local publication. It contains only records introduced or
// replaced by that publication; unchanged compiled records remain owned by the
// base declarations.
type declarationDelta struct {
	commands           []commandRecord
	destinations       []destinationRecord
	flags              []flagRecord
	flagSpellings      []flagSpellingRecord
	argumentForms      []argumentFormRecord
	argumentTerms      []argumentTermRecord
	commandEntries     []commandEntryRecord
	actionEntries      []actionEntryRecord
	defaults           []defaultRecord
	providers          []providerRecord
	rules              []ruleRecord
	advisories         []advisoryRecord
	signalPolicies     []signalPolicyRecord
	domainEntries      []domainEntryRecord
	valueDomains       []valueDomainRecord
	environmentInputs  []environmentInputRecord
	grammarProfiles    []grammarProfileRecord
	signalEndpoints    []signalEndpointRecord
	signalCounters     []signalCounterRecord
	conditionPolicies  []conditionPolicyRecord
	noticeSuppressions []noticeSuppressionRecord
	signalPolicyScopes []signalPolicyScope
	helpGroups         []helpGroupRecord
	replacements       []declarationReplacementRecord
}

// freezeDeclarationDelta snapshots only the supplied declaration records. The
// input carrier is reused to keep the canonical family list in one place; the
// resulting delta is not StaticRecords and builds no whole-model indexes.
func freezeDeclarationDelta(input declarationRecordInput) declarationDelta {
	result := declarationDelta{
		commands:           append([]commandRecord(nil), input.Commands...),
		destinations:       append([]destinationRecord(nil), input.Destinations...),
		flags:              append([]flagRecord(nil), input.Flags...),
		flagSpellings:      append([]flagSpellingRecord(nil), input.FlagSpellings...),
		argumentForms:      append([]argumentFormRecord(nil), input.ArgumentForms...),
		argumentTerms:      append([]argumentTermRecord(nil), input.ArgumentTerms...),
		commandEntries:     append([]commandEntryRecord(nil), input.CommandEntries...),
		actionEntries:      append([]actionEntryRecord(nil), input.ActionEntries...),
		defaults:           append([]defaultRecord(nil), input.Defaults...),
		providers:          append([]providerRecord(nil), input.Providers...),
		rules:              append([]ruleRecord(nil), input.Rules...),
		advisories:         append([]advisoryRecord(nil), input.Advisories...),
		signalPolicies:     append([]signalPolicyRecord(nil), input.SignalPolicies...),
		domainEntries:      append([]domainEntryRecord(nil), input.DomainEntries...),
		valueDomains:       cloneValueDomainRecords(input.ValueDomains),
		environmentInputs:  append([]environmentInputRecord(nil), input.EnvironmentInputs...),
		grammarProfiles:    append([]grammarProfileRecord(nil), input.GrammarProfiles...),
		signalEndpoints:    append([]signalEndpointRecord(nil), input.SignalEndpoints...),
		signalCounters:     append([]signalCounterRecord(nil), input.SignalCounters...),
		conditionPolicies:  append([]conditionPolicyRecord(nil), input.ConditionPolicies...),
		noticeSuppressions: append([]noticeSuppressionRecord(nil), input.NoticeSuppressions...),
		signalPolicyScopes: append([]signalPolicyScope(nil), input.SignalPolicyScopes...),
		helpGroups:         append([]helpGroupRecord(nil), input.HelpGroups...),
		replacements:       append([]declarationReplacementRecord(nil), input.Replacements...),
	}

	return result
}

func cloneValueDomainRecords(records []valueDomainRecord) []valueDomainRecord {
	result := make([]valueDomainRecord, len(records))
	for i, record := range records {
		result[i] = cloneValueDomainRecord(record)
	}
	return result
}

func cloneValueDomainRecord(record valueDomainRecord) valueDomainRecord {
	if dynamic, ok := record.DynamicDefinition(); ok {
		result := acceptDynamicValueDomainRecord(dynamic, record.Uses())
		result.entries = record.Entries()
		return result
	}
	return acceptValueDomainRecord(record.Entries(), record.Uses())
}

func (delta declarationDelta) Commands() []commandRecord {
	return append([]commandRecord(nil), delta.commands...)
}

func (delta declarationDelta) Destinations() []destinationRecord {
	return append([]destinationRecord(nil), delta.destinations...)
}

func (delta declarationDelta) Flags() []flagRecord { return append([]flagRecord(nil), delta.flags...) }

func (delta declarationDelta) FlagSpellings() []flagSpellingRecord {
	return append([]flagSpellingRecord(nil), delta.flagSpellings...)
}

func (delta declarationDelta) ArgumentForms() []argumentFormRecord {
	return append([]argumentFormRecord(nil), delta.argumentForms...)
}

func (delta declarationDelta) ArgumentTerms() []argumentTermRecord {
	return append([]argumentTermRecord(nil), delta.argumentTerms...)
}

func (delta declarationDelta) CommandEntries() []commandEntryRecord {
	return append([]commandEntryRecord(nil), delta.commandEntries...)
}

func (delta declarationDelta) ActionEntries() []actionEntryRecord {
	return append([]actionEntryRecord(nil), delta.actionEntries...)
}

func (delta declarationDelta) Defaults() []defaultRecord {
	return append([]defaultRecord(nil), delta.defaults...)
}

func (delta declarationDelta) Providers() []providerRecord {
	return append([]providerRecord(nil), delta.providers...)
}

func (delta declarationDelta) Rules() []ruleRecord { return append([]ruleRecord(nil), delta.rules...) }

func (delta declarationDelta) Advisories() []advisoryRecord {
	return append([]advisoryRecord(nil), delta.advisories...)
}

func (delta declarationDelta) SignalPolicies() []signalPolicyRecord {
	return append([]signalPolicyRecord(nil), delta.signalPolicies...)
}

func (delta declarationDelta) DomainEntries() []domainEntryRecord {
	return append([]domainEntryRecord(nil), delta.domainEntries...)
}

func (delta declarationDelta) ValueDomains() []valueDomainRecord {
	return cloneValueDomainRecords(delta.valueDomains)
}

func (delta declarationDelta) ValueDomain(index int) (valueDomainRecord, bool) {
	if index < 0 || index >= len(delta.valueDomains) {
		return valueDomainRecord{}, false
	}
	return cloneValueDomainRecord(delta.valueDomains[index]), true
}

func (delta declarationDelta) EnvironmentInputs() []environmentInputRecord {
	return append([]environmentInputRecord(nil), delta.environmentInputs...)
}

func (delta declarationDelta) GrammarProfiles() []grammarProfileRecord {
	return append([]grammarProfileRecord(nil), delta.grammarProfiles...)
}

func (delta declarationDelta) SignalEndpoints() []signalEndpointRecord {
	return append([]signalEndpointRecord(nil), delta.signalEndpoints...)
}

func (delta declarationDelta) SignalCounters() []signalCounterRecord {
	return append([]signalCounterRecord(nil), delta.signalCounters...)
}

func (delta declarationDelta) ConditionPolicies() []conditionPolicyRecord {
	return append([]conditionPolicyRecord(nil), delta.conditionPolicies...)
}

func (delta declarationDelta) NoticeSuppressions() []noticeSuppressionRecord {
	return append([]noticeSuppressionRecord(nil), delta.noticeSuppressions...)
}

func (delta declarationDelta) SignalPolicyScopes() []signalPolicyScope {
	return append([]signalPolicyScope(nil), delta.signalPolicyScopes...)
}

func (delta declarationDelta) HelpGroups() []helpGroupRecord {
	return append([]helpGroupRecord(nil), delta.helpGroups...)
}

func (delta declarationDelta) Replacements() []declarationReplacementRecord {
	return append([]declarationReplacementRecord(nil), delta.replacements...)
}

func (delta declarationDelta) Empty() bool {
	return len(delta.commands) == 0 && len(delta.destinations) == 0 && len(delta.flags) == 0 &&
		len(delta.flagSpellings) == 0 && len(delta.argumentForms) == 0 && len(delta.argumentTerms) == 0 &&
		len(delta.commandEntries) == 0 && len(delta.actionEntries) == 0 && len(delta.defaults) == 0 &&
		len(delta.providers) == 0 && len(delta.rules) == 0 && len(delta.advisories) == 0 &&
		len(delta.signalPolicies) == 0 && len(delta.domainEntries) == 0 && len(delta.valueDomains) == 0 &&
		len(delta.environmentInputs) == 0 && len(delta.grammarProfiles) == 0 && len(delta.signalEndpoints) == 0 &&
		len(delta.signalCounters) == 0 && len(delta.conditionPolicies) == 0 && len(delta.noticeSuppressions) == 0 &&
		len(delta.signalPolicyScopes) == 0 && len(delta.helpGroups) == 0 && len(delta.replacements) == 0
}

func cloneDialectMap[K comparable](source map[K]dialect) map[K]dialect {
	cloned := make(map[K]dialect, len(source))
	for id, dialect := range source {
		cloned[id] = cloneDialect(dialect)
	}
	return cloned
}

// grammarDelta is the immutable grammar contribution carried by an accepted
// provider publication. Grammar objects are retained verbatim, including the
// executable TokenLanguage carriers.
type grammarDelta struct {
	dialects        map[CommandID]dialect
	profileDialects map[GrammarProfileID]dialect
	profileCommands map[GrammarProfileID]CommandID
	profileOrder    []GrammarProfileID
	commandEntries  []commandEntryGrammar
	actionCommands  []actionCommandGrammar
	actionFlags     []semanticActionFlagGrammar
	flags           []flagGrammar
	argumentForms   []argumentFormGrammar
}

func freezeGrammarDelta(input grammarInput) grammarDelta {
	delta := grammarDelta{
		dialects:        cloneDialectMap(input.Dialects),
		profileDialects: cloneDialectMap(input.ProfileDialects),
		profileCommands: make(map[GrammarProfileID]CommandID, len(input.ProfileCommands)),
		profileOrder:    append([]GrammarProfileID(nil), input.ProfileOrder...),
		commandEntries:  append([]commandEntryGrammar(nil), input.CommandEntries...),
		actionCommands:  append([]actionCommandGrammar(nil), input.ActionCommands...),
		actionFlags:     append([]semanticActionFlagGrammar(nil), input.ActionFlags...),
		flags:           append([]flagGrammar(nil), input.Flags...),
		argumentForms:   append([]argumentFormGrammar(nil), input.ArgumentForms...),
	}
	maps.Copy(delta.profileCommands, input.ProfileCommands)
	return delta
}

func (delta grammarDelta) Input() grammarInput {
	input := grammarInput{
		Dialects:        cloneDialectMap(delta.dialects),
		ProfileDialects: cloneDialectMap(delta.profileDialects),
		ProfileCommands: make(map[GrammarProfileID]CommandID, len(delta.profileCommands)),
		ProfileOrder:    append([]GrammarProfileID(nil), delta.profileOrder...),
		CommandEntries:  append([]commandEntryGrammar(nil), delta.commandEntries...),
		ActionCommands:  append([]actionCommandGrammar(nil), delta.actionCommands...),
		ActionFlags:     append([]semanticActionFlagGrammar(nil), delta.actionFlags...),
		Flags:           append([]flagGrammar(nil), delta.flags...),
		ArgumentForms:   append([]argumentFormGrammar(nil), delta.argumentForms...),
	}
	maps.Copy(input.ProfileCommands, delta.profileCommands)
	return input
}

func (delta grammarDelta) Dialect(command CommandID) (dialect, bool) {
	dialect, ok := delta.dialects[command]
	return cloneDialect(dialect), ok
}

func (delta grammarDelta) ProfileDialect(profile GrammarProfileID) (dialect, bool) {
	dialect, ok := delta.profileDialects[profile]
	return cloneDialect(dialect), ok
}

func (delta grammarDelta) Profiles(command CommandID) []GrammarProfileID {
	var result []GrammarProfileID
	for _, profile := range delta.profileOrder {
		if delta.profileCommands[profile] == command {
			result = append(result, profile)
		}
	}
	return result
}

func (delta grammarDelta) CommandEntries(parent CommandID) []commandEntryGrammar {
	var result []commandEntryGrammar
	for _, entry := range delta.commandEntries {
		if entry.parent == parent {
			result = append(result, entry)
		}
	}
	return result
}

func (delta grammarDelta) ActionCommands(parent CommandID) []actionCommandGrammar {
	var result []actionCommandGrammar
	for _, action := range delta.actionCommands {
		if action.parent == parent {
			result = append(result, action)
		}
	}
	return result
}

func (delta grammarDelta) ActionFlags(command CommandID) []semanticActionFlagGrammar {
	var result []semanticActionFlagGrammar
	for _, action := range delta.actionFlags {
		if action.parent == command {
			result = append(result, action)
		}
	}
	return result
}

func (delta grammarDelta) Flags(command CommandID) []flagGrammar {
	var result []flagGrammar
	for _, flag := range delta.flags {
		if flag.command == command {
			result = append(result, flag)
		}
	}
	return result
}

func (delta grammarDelta) ArgumentForms(command CommandID) []argumentFormGrammar {
	var result []argumentFormGrammar
	for _, form := range delta.argumentForms {
		if form.command == command {
			result = append(result, form)
		}
	}
	return result
}

func (delta grammarDelta) Empty() bool {
	return len(delta.dialects) == 0 && len(delta.profileDialects) == 0 && len(delta.profileOrder) == 0 &&
		len(delta.commandEntries) == 0 && len(delta.actionCommands) == 0 && len(delta.actionFlags) == 0 &&
		len(delta.flags) == 0 && len(delta.argumentForms) == 0
}

// providerPublication is one immutable accepted provider settlement wave. The
// declaration and grammar deltas, provider-owned localization, and provider
// batch ownership are committed as one operation-local fact.
type providerPublication struct {
	declarations        declarationDelta
	grammar             grammarDelta
	commandLocalization map[CommandID]localizationSnapshot
	entryLocalization   map[CommandEntryID]localizationSnapshot
	providerEntries     map[CommandProviderID][]CommandEntryID
}

func freezeProviderPublication(declarations declarationDelta, grammar grammarDelta, commandLocalization map[CommandID]localizationSnapshot, entryLocalization map[CommandEntryID]localizationSnapshot, providerEntries map[CommandProviderID][]CommandEntryID) providerPublication {
	publication := providerPublication{
		declarations:        declarations,
		grammar:             grammar,
		commandLocalization: make(map[CommandID]localizationSnapshot, len(commandLocalization)),
		entryLocalization:   make(map[CommandEntryID]localizationSnapshot, len(entryLocalization)),
		providerEntries:     make(map[CommandProviderID][]CommandEntryID, len(providerEntries)),
	}
	maps.Copy(publication.commandLocalization, commandLocalization)
	maps.Copy(publication.entryLocalization, entryLocalization)
	for provider, owned := range providerEntries {
		publication.providerEntries[provider] = append([]CommandEntryID(nil), owned...)
	}
	return publication
}

func (publication providerPublication) Declarations() declarationDelta {
	return publication.declarations
}

func (publication providerPublication) Grammar() grammarDelta { return publication.grammar }

func (publication providerPublication) CommandLocalization() map[CommandID]localizationSnapshot {
	result := make(map[CommandID]localizationSnapshot, len(publication.commandLocalization))
	maps.Copy(result, publication.commandLocalization)
	return result
}

func (publication providerPublication) EntryLocalization() map[CommandEntryID]localizationSnapshot {
	result := make(map[CommandEntryID]localizationSnapshot, len(publication.entryLocalization))
	maps.Copy(result, publication.entryLocalization)
	return result
}

func (publication providerPublication) ProviderEntries() map[CommandProviderID][]CommandEntryID {
	result := make(map[CommandProviderID][]CommandEntryID, len(publication.providerEntries))
	for provider, entries := range publication.providerEntries {
		result[provider] = append([]CommandEntryID(nil), entries...)
	}
	return result
}

// dynamicDomainPublication is one immutable accepted operation-local domain
// materialization. Canonical publication truth is only the demanded domain
// coordinate, its accepted executable value language snapshot, and the accepted
// domain entries. Effective destination definitions are derived from these facts
// plus the canonical ValueDomainUse and original destination definition held by
// the declarations; they are never independent publication authority.
type dynamicDomainPublication struct {
	domain   int
	language valueLanguage
	entries  []domainEntryRecord
}

func freezeDynamicDomainPublication(domain int, language valueLanguage, entries []domainEntryRecord) dynamicDomainPublication {
	if domain < 0 || !language.Valid() || language.DynamicDomain() {
		panic("semantic: invalid dynamic-domain publication")
	}
	result := dynamicDomainPublication{
		domain:   domain,
		language: language,
		entries:  append([]domainEntryRecord(nil), entries...),
	}

	return result
}

func (publication dynamicDomainPublication) Domain() int { return publication.domain }

func (publication dynamicDomainPublication) Language() valueLanguage { return publication.language }

func (publication dynamicDomainPublication) Entries() []domainEntryRecord {
	return append([]domainEntryRecord(nil), publication.entries...)
}

// materializedValueDomainDefinition derives one effective destination
// definition from an accepted dynamic-domain language. It is deliberately pure:
// callers may discard and rebuild the result without changing semantic truth.
func materializedValueDomainDefinition(definition destinationDefinition, kind ValueDomainUseKind, language valueLanguage) (destinationDefinition, bool) {
	if !language.Valid() || language.DynamicDomain() {
		return destinationDefinition{}, false
	}
	switch definition.Kind() {
	case destinationKindScalar:
		if kind != ValueDomainUseDestination {
			return destinationDefinition{}, false
		}
		return definition.WithLanguage(language), true
	case destinationKindSequence:
		if kind != ValueDomainUseSequenceElement {
			return destinationDefinition{}, false
		}
		return definition.WithLanguage(language), true
	case destinationKindKeyed:
		keyed, ok := definition.KeyedLanguage()
		if !ok {
			return destinationDefinition{}, false
		}
		key := keyed.Key()
		value := keyed.Value()
		switch kind {
		case ValueDomainUseMapKey:
			key = language
		case ValueDomainUseMapValue:
			value = language
		default:
			return destinationDefinition{}, false
		}
		var effective keyedValueLanguage
		if keyOnly, present := keyed.KeyOnly(); present {
			if value.Kind() == valueLanguageKindDomain {
				if _, err := value.Format(keyOnly); err != nil {
					return destinationDefinition{}, false
				}
			}
			effective = acceptKeyedValueLanguageWithKeyOnly(key, value, keyed.Separator(), keyOnly)
		} else {
			effective = acceptKeyedValueLanguage(key, value, keyed.Separator())
		}
		return definition.WithKeyedLanguage(effective), true
	default:
		return destinationDefinition{}, false
	}
}

func snapshotPublicationInput(input declarationRecordInput) declarationRecordInput {
	result := declarationRecordInput{
		IntrinsicActions:   input.IntrinsicActions,
		Commands:           append([]commandRecord(nil), input.Commands...),
		Destinations:       append([]destinationRecord(nil), input.Destinations...),
		Flags:              append([]flagRecord(nil), input.Flags...),
		FlagSpellings:      append([]flagSpellingRecord(nil), input.FlagSpellings...),
		ArgumentForms:      append([]argumentFormRecord(nil), input.ArgumentForms...),
		ArgumentTerms:      append([]argumentTermRecord(nil), input.ArgumentTerms...),
		CommandEntries:     append([]commandEntryRecord(nil), input.CommandEntries...),
		ActionEntries:      append([]actionEntryRecord(nil), input.ActionEntries...),
		Defaults:           append([]defaultRecord(nil), input.Defaults...),
		Providers:          append([]providerRecord(nil), input.Providers...),
		Rules:              append([]ruleRecord(nil), input.Rules...),
		Advisories:         append([]advisoryRecord(nil), input.Advisories...),
		SignalPolicies:     append([]signalPolicyRecord(nil), input.SignalPolicies...),
		DomainEntries:      append([]domainEntryRecord(nil), input.DomainEntries...),
		EnvironmentInputs:  append([]environmentInputRecord(nil), input.EnvironmentInputs...),
		GrammarProfiles:    append([]grammarProfileRecord(nil), input.GrammarProfiles...),
		SignalEndpoints:    append([]signalEndpointRecord(nil), input.SignalEndpoints...),
		SignalCounters:     append([]signalCounterRecord(nil), input.SignalCounters...),
		ConditionPolicies:  append([]conditionPolicyRecord(nil), input.ConditionPolicies...),
		NoticeSuppressions: append([]noticeSuppressionRecord(nil), input.NoticeSuppressions...),
		SignalPolicyScopes: append([]signalPolicyScope(nil), input.SignalPolicyScopes...),
		HelpGroups:         append([]helpGroupRecord(nil), input.HelpGroups...),
		Replacements:       append([]declarationReplacementRecord(nil), input.Replacements...),
	}
	result.ValueDomains = make([]valueDomainRecord, len(input.ValueDomains))
	for index, record := range input.ValueDomains {
		if dynamic, ok := record.DynamicDefinition(); ok {
			copyRecord := acceptDynamicValueDomainRecord(dynamic, record.Uses())
			copyRecord.entries = record.Entries()
			result.ValueDomains[index] = copyRecord
		} else {
			result.ValueDomains[index] = acceptValueDomainRecord(record.Entries(), record.Uses())
		}
	}
	return result
}

func publicationCounts(input declarationRecordInput) publicationRecordCounts {
	result := publicationRecordCounts{
		commands:           len(input.Commands),
		destinations:       len(input.Destinations),
		flags:              len(input.Flags),
		flagSpellings:      len(input.FlagSpellings),
		argumentForms:      len(input.ArgumentForms),
		argumentTerms:      len(input.ArgumentTerms),
		commandEntries:     len(input.CommandEntries),
		actionEntries:      len(input.ActionEntries),
		defaults:           len(input.Defaults),
		providers:          len(input.Providers),
		rules:              len(input.Rules),
		advisories:         len(input.Advisories),
		signalPolicies:     len(input.SignalPolicies),
		domainEntries:      len(input.DomainEntries),
		valueDomains:       len(input.ValueDomains),
		environmentInputs:  len(input.EnvironmentInputs),
		grammarProfiles:    len(input.GrammarProfiles),
		signalEndpoints:    len(input.SignalEndpoints),
		signalCounters:     len(input.SignalCounters),
		conditionPolicies:  len(input.ConditionPolicies),
		noticeSuppressions: len(input.NoticeSuppressions),
		signalPolicyScopes: len(input.SignalPolicyScopes),
		helpGroups:         len(input.HelpGroups),
		replacements:       len(input.Replacements),
	}

	return result
}

func publicationGrammarBaseCounts(grammar grammarModel) publicationGrammarCounts {
	result := publicationGrammarCounts{
		commandEntries: len(grammar.AllCommandEntries()),
		actionCommands: len(grammar.AllActionCommands()),
		actionFlags:    len(grammar.AllActionFlags()),
		flags:          len(grammar.AllFlags()),
		argumentForms:  len(grammar.AllArgumentForms()),
	}

	return result
}

func publicationTail[T any](values []T, offset int) []T {
	if offset < 0 || offset > len(values) {
		panic("application: invalid publication boundary")
	}
	return append([]T(nil), values[offset:]...)
}

func (workspace *publicationDraft) declarationDelta() declarationDelta {
	base := workspace.base
	input2 := declarationRecordInput{
		Commands:           publicationTail(workspace.input.Commands, base.commands),
		Destinations:       publicationTail(workspace.input.Destinations, base.destinations),
		Flags:              publicationTail(workspace.input.Flags, base.flags),
		FlagSpellings:      publicationTail(workspace.input.FlagSpellings, base.flagSpellings),
		ArgumentForms:      publicationTail(workspace.input.ArgumentForms, base.argumentForms),
		ArgumentTerms:      publicationTail(workspace.input.ArgumentTerms, base.argumentTerms),
		CommandEntries:     publicationTail(workspace.input.CommandEntries, base.commandEntries),
		ActionEntries:      publicationTail(workspace.input.ActionEntries, base.actionEntries),
		Defaults:           publicationTail(workspace.input.Defaults, base.defaults),
		Providers:          publicationTail(workspace.input.Providers, base.providers),
		Rules:              publicationTail(workspace.input.Rules, base.rules),
		Advisories:         publicationTail(workspace.input.Advisories, base.advisories),
		SignalPolicies:     publicationTail(workspace.input.SignalPolicies, base.signalPolicies),
		DomainEntries:      publicationTail(workspace.input.DomainEntries, base.domainEntries),
		ValueDomains:       publicationTail(workspace.input.ValueDomains, base.valueDomains),
		EnvironmentInputs:  publicationTail(workspace.input.EnvironmentInputs, base.environmentInputs),
		GrammarProfiles:    publicationTail(workspace.input.GrammarProfiles, base.grammarProfiles),
		SignalEndpoints:    publicationTail(workspace.input.SignalEndpoints, base.signalEndpoints),
		SignalCounters:     publicationTail(workspace.input.SignalCounters, base.signalCounters),
		ConditionPolicies:  publicationTail(workspace.input.ConditionPolicies, base.conditionPolicies),
		NoticeSuppressions: publicationTail(workspace.input.NoticeSuppressions, base.noticeSuppressions),
		SignalPolicyScopes: publicationTail(workspace.input.SignalPolicyScopes, base.signalPolicyScopes),
		HelpGroups:         publicationTail(workspace.input.HelpGroups, base.helpGroups),
		Replacements:       publicationTail(workspace.input.Replacements, base.replacements),
	}
	return freezeDeclarationDelta(input2)
}

func (workspace *publicationDraft) grammarDelta(grammar grammarModel) grammarDelta {
	base := workspace.baseGrammar
	input := grammarInput{
		Dialects:        make(map[CommandID]dialect),
		ProfileDialects: make(map[GrammarProfileID]dialect),
		ProfileCommands: make(map[GrammarProfileID]CommandID),
		CommandEntries:  publicationTail(grammar.AllCommandEntries(), base.commandEntries),
		ActionCommands:  publicationTail(grammar.AllActionCommands(), base.actionCommands),
		ActionFlags:     publicationTail(grammar.AllActionFlags(), base.actionFlags),
		Flags:           publicationTail(grammar.AllFlags(), base.flags),
		ArgumentForms:   publicationTail(grammar.AllArgumentForms(), base.argumentForms),
	}
	// Grammar mutation can target declarations that predate this publication.
	// Emit every command/profile dialect whose effective value differs from the
	// publication baseline, not only dialects belonging to newly declared rows.
	for _, command := range workspace.input.Commands {
		dialect, ok := grammar.Dialect(command.meta.id)
		if !ok {
			continue
		}
		baseDialect, baseOK := workspace.baseGrammarModel.Dialect(command.meta.id)
		if !baseOK || !baseDialect.Equal(dialect) {
			input.Dialects[command.meta.id] = dialect
		}
	}
	for _, profile := range workspace.input.GrammarProfiles {
		dialect, ok := grammar.ProfileDialect(profile.meta.id)
		if !ok {
			continue
		}
		baseDialect, baseOK := workspace.baseGrammarModel.ProfileDialect(profile.meta.id)
		if !baseOK || !baseDialect.Equal(dialect) {
			input.ProfileDialects[profile.meta.id] = dialect
			input.ProfileCommands[profile.meta.id] = profile.command
			input.ProfileOrder = append(input.ProfileOrder, profile.meta.id)
		}
	}
	return freezeGrammarDelta(input)
}

// publicationValidationScope is a disposable, bounded validation input for one
// runtime declaration transaction. It is not operation truth and cannot act as
// an effective-declarations or static-declarations replacement.
type publicationValidationScope struct {
	Records     declarationRecordInput
	Grammar     grammarModel
	Application applicationMetadata
}

// ProviderPublicationScope selects only the effective command namespace and
// external relationships that can affect one provider publication.
func (model *effectiveModel) ProviderPublicationScope(owner CommandID, externalDestinations []ValueID) (publicationValidationScope, bool) {
	if model == nil || !owner.present() {
		return publicationValidationScope{}, false
	}
	if grammar, ok := model.base.Grammar(); !ok || !grammar.Valid() {
		return publicationValidationScope{}, false
	}
	model.mu.RLock()
	defer model.mu.RUnlock()

	selected := make(map[CommandID]struct{})
	addAncestors := func(command CommandID) {
		for command.present() {
			if _, exists := selected[command]; exists {
				return
			}
			selected[command] = struct{}{}
			record, exists := model.commandLocked(command)
			if !exists {
				return
			}
			parent, exists := record.Parent()
			if !exists {
				return
			}
			command = parent
		}
	}
	var addDescendants func(CommandID)
	addDescendants = func(command CommandID) {
		if _, exists := selected[command]; !exists {
			selected[command] = struct{}{}
		}
		for _, child := range model.childrenForCommandLocked(command) {
			if _, exists := selected[child]; exists {
				continue
			}
			addDescendants(child)
		}
	}
	addAncestors(owner)
	addDescendants(owner)
	for _, destination := range externalDestinations {
		if record, exists := model.destinationLocked(destination); exists {
			addAncestors(record.command)
		}
	}

	// The selected provider namespace owns the records that can collide with or
	// satisfy this publication. Entry/default targets outside that namespace are
	// validation references only: their command ancestry and exact referenced
	// relationships must resolve, but their owned descendants/records must not be
	// imported into this transaction.
	validationCommands := copyCommandSet(selected)
	input := model.scopeRecordInputLocked(selected)
	input = model.closeValidationRecordReferencesLocked(input, selected, validationCommands)
	grammarInput := model.scopeGrammarInputLocked(selected, input.GrammarProfiles)
	model.addValidationGrammarDialectsLocked(&grammarInput, selected, validationCommands)
	grammarModel := acceptGrammarModel(grammarInput)
	result := publicationValidationScope{
		Records:     input,
		Grammar:     grammarModel,
		Application: model.base.Application(),
	}
	return result, true
}

// DynamicDomainPublicationScope builds the bounded validation input for one
// demanded domain and returns its compact local domain coordinate.
func (model *effectiveModel) DynamicDomainPublicationScope(index int) (publicationValidationScope, int, bool) {
	if model == nil {
		return publicationValidationScope{}, -1, false
	}
	if grammar, ok := model.base.Grammar(); !ok || !grammar.Valid() {
		return publicationValidationScope{}, -1, false
	}
	model.mu.RLock()
	defer model.mu.RUnlock()
	domain, ok := model.valueDomainLocked(index)
	if !ok {
		return publicationValidationScope{}, -1, false
	}
	definition, dynamic := domain.DynamicDefinition()
	if !dynamic {
		return publicationValidationScope{}, -1, false
	}
	selected := make(map[CommandID]struct{})
	addAncestors := func(command CommandID) {
		for command.present() {
			if _, exists := selected[command]; exists {
				return
			}
			selected[command] = struct{}{}
			record, exists := model.commandLocked(command)
			if !exists {
				return
			}
			parent, exists := record.Parent()
			if !exists {
				return
			}
			command = parent
		}
	}
	addAncestors(definition.Owner())
	for _, use := range domain.Uses() {
		if destination, exists := model.destinationLocked(use.destination); exists {
			addAncestors(destination.command)
		}
	}
	for _, dependency := range definition.Dependencies() {
		if destination, exists := model.destinationLocked(dependency.Destination()); exists {
			addAncestors(destination.command)
		}
	}

	// Dynamic-domain publication mutates only the demanded domain and its use
	// destinations; it does not mutate command namespace structure. Keep owned
	// records bounded to the owner/use/dependency ancestry above, while adding
	// reference-only command context for command entries and exact destination
	// relationships required by that ancestry. This makes the disposable graph
	// without importing records or descendants owned by unrelated targets.
	input, validationCommands := model.dynamicDomainPublicationRecordInputLocked(selected, domain)
	grammarInput := model.dynamicDomainPublicationGrammarInputLocked(input, validationCommands)
	grammarModel := acceptGrammarModel(grammarInput)
	result := publicationValidationScope{
		Records:     input,
		Grammar:     grammarModel,
		Application: model.base.Application(),
	}
	return result, 0, true
}

// dynamicDomainPublicationRecordInputLocked builds the validation substrate for
// exactly one demanded dynamic domain. Unlike the generic provider scope, it
// does not import declaration families merely because a selected command owns
// them. It closes only relationships that can affect the demanded publication.
func (model *effectiveModel) dynamicDomainPublicationRecordInputLocked(selected map[CommandID]struct{}, domain valueDomainRecord) (declarationRecordInput, map[CommandID]struct{}) {
	actions, _ := model.base.IntrinsicActions()
	input := declarationRecordInput{
		IntrinsicActions: actions,
		ValueDomains:     []valueDomainRecord{cloneValueDomainRecord(domain)},
	}
	for _, id := range domain.Entries() {
		if entry, ok := model.domainEntryLocked(id); ok {
			input.DomainEntries = append(input.DomainEntries, entry)
		}
	}

	validationCommands := copyCommandSet(selected)
	addAncestors := func(command CommandID) {
		for command.present() {
			if _, exists := validationCommands[command]; exists {
				return
			}
			validationCommands[command] = struct{}{}
			record, exists := model.commandLocked(command)
			if !exists {
				return
			}
			parent, exists := record.Parent()
			if !exists {
				return
			}
			command = parent
		}
	}

	destinationSet := make(map[ValueID]struct{})
	profileSet := make(map[GrammarProfileID]struct{})
	var addDestination func(ValueID)
	addProfile := func(id GrammarProfileID) {
		if !id.present() {
			return
		}
		if _, exists := profileSet[id]; exists {
			return
		}
		record, exists := model.grammarProfiles[id]
		if !exists {
			record, exists = model.base.GrammarProfile(id)
		}
		if !exists {
			return
		}
		profileSet[id] = struct{}{}
		input.GrammarProfiles = append(input.GrammarProfiles, record)
		addAncestors(record.command)
		if selector, ok := record.Selector(); ok {
			addDestination(selector)
		}
	}
	addDestination = func(id ValueID) {
		if _, exists := destinationSet[id]; exists {
			return
		}
		record, exists := model.destinationLocked(id)
		if !exists {
			return
		}
		destinationSet[id] = struct{}{}
		input.Destinations = append(input.Destinations, record)
		addAncestors(record.command)
		addProfile(record.meta.profile)
		for _, producer := range record.Producers() {
			addProfile(producer.profile)
			for _, dependency := range producer.Dependencies() {
				addDestination(dependency.Destination())
			}
		}
	}

	// Preserve the already-established command-entry reference closure for the
	// selected ancestry, but do not import records owned by reference-only targets.
	for command := range selected {
		for _, entry := range model.commandEntriesForParentLocked(command) {
			input.CommandEntries = append(input.CommandEntries, entry)
			addAncestors(entry.target)
			addProfile(entry.meta.profile)
			if shortcut, ok := entry.Shortcut(); ok {
				for _, value := range shortcut.Defaults() {
					if value.Kind() == shortcutDefaultKindDestination {
						addDestination(value.Destination())
					}
				}
			}
		}
	}
	definition, _ := domain.DynamicDefinition()
	for _, use := range domain.Uses() {
		addDestination(use.destination)
	}
	for _, dependency := range definition.Dependencies() {
		addDestination(dependency.Destination())
	}

	// Command access relationships are validation inputs even when their source
	// destinations are outside the demanded domain's direct use/dependency set.
	processedCommands := make(map[CommandID]struct{})
	for len(processedCommands) != len(validationCommands) {
		for command := range validationCommands {
			if _, done := processedCommands[command]; done {
				continue
			}
			processedCommands[command] = struct{}{}
			record, exists := model.commandLocked(command)
			if !exists {
				continue
			}
			access := record.access
			for _, relationship := range access.Values() {
				addDestination(relationship.Source())
			}
			for _, relationship := range access.OptionalValues() {
				addDestination(relationship.Source())
			}
			for _, relationship := range access.ValueStates() {
				addDestination(relationship.Source())
			}
		}
	}

	// Retain only grammar declarations whose destinations participate in this
	// exact validation closure. This preserves selector/locale restrictions and
	// grammar validation without importing unrelated same-command values.
	for command := range selected {
		for _, flag := range model.flagsForCommandLocked(command) {
			if _, relevant := destinationSet[flag.destination]; !relevant {
				continue
			}
			input.Flags = append(input.Flags, flag)
			input.FlagSpellings = append(input.FlagSpellings, model.flagSpellingsForFlagLocked(flag.meta.id)...)
			addProfile(flag.meta.profile)
			if completion, ok := flag.Completion(); ok {
				for _, dependency := range completion.Dependencies() {
					addDestination(dependency.Destination())
				}
			}
		}
		for _, form := range model.formsForCommandLocked(command) {
			terms := model.termsForFormLocked(form.meta.id)
			relevant := false
			for _, selection := range form.Selections() {
				if _, ok := destinationSet[selection.destination]; ok {
					relevant = true
					break
				}
			}
			if !relevant {
				for _, term := range terms {
					if destination, ok := term.Destination(); ok {
						if _, hit := destinationSet[destination]; hit {
							relevant = true
							break
						}
					}
				}
			}
			if !relevant {
				continue
			}
			input.ArgumentForms = append(input.ArgumentForms, form)
			input.ArgumentTerms = append(input.ArgumentTerms, terms...)
			addProfile(form.meta.profile)
			for _, selection := range form.Selections() {
				addDestination(selection.destination)
			}
			for _, term := range terms {
				if destination, ok := term.Destination(); ok {
					addDestination(destination)
				}
				if completion, ok := term.Completion(); ok {
					for _, dependency := range completion.Dependencies() {
						addDestination(dependency.Destination())
					}
				}
			}
		}
		for _, environment := range model.environmentForCommandLocked(command) {
			if _, relevant := destinationSet[environment.destination]; !relevant {
				continue
			}
			input.EnvironmentInputs = append(input.EnvironmentInputs, environment)
			addProfile(environment.meta.profile)
		}
	}

	commands := make([]commandRecord, 0, len(validationCommands))
	for command := range validationCommands {
		if record, ok := model.commandLocked(command); ok {
			commands = append(commands, record)
		}
	}
	slices.SortStableFunc(commands, func(left, right commandRecord) int { return cmp.Compare(left.meta.position, right.meta.position) })
	input.Commands = commands
	sortRecordInput(&input)
	return input, validationCommands
}

func (model *effectiveModel) dynamicDomainPublicationGrammarInputLocked(input declarationRecordInput, validationCommands map[CommandID]struct{}) grammarInput {
	grammar, ok := model.effectiveGrammarLocked()
	if !ok {
		panic("invocation: semantic view base has no grammar")
	}
	result := grammarInput{
		Root:            grammar.Root(),
		Dialects:        make(map[CommandID]dialect),
		ProfileDialects: make(map[GrammarProfileID]dialect),
		ProfileCommands: make(map[GrammarProfileID]CommandID),
	}
	for _, commandRecord := range input.Commands {
		command := commandRecord.meta.id
		if _, required := validationCommands[command]; !required {
			continue
		}
		if dialect, ok := grammar.Dialect(command); ok {
			result.Dialects[command] = dialect
		}
	}
	entrySet := make(map[CommandEntryID]struct{}, len(input.CommandEntries))
	for _, record := range input.CommandEntries {
		entrySet[record.meta.id] = struct{}{}
	}
	for _, commandRecord := range input.Commands {
		command := commandRecord.meta.id
		for _, entry := range grammar.CommandEntries(command) {
			if _, ok := entrySet[entry.id]; ok {
				result.CommandEntries = append(result.CommandEntries, entry)
			}
		}
	}
	flagSet := make(map[FlagID]struct{}, len(input.Flags))
	for _, record := range input.Flags {
		flagSet[record.meta.id] = struct{}{}
	}
	for _, commandRecord := range input.Commands {
		command := commandRecord.meta.id
		for _, flag := range grammar.Flags(command) {
			if _, ok := flagSet[flag.id]; ok {
				result.Flags = append(result.Flags, flag)
			}
		}
	}
	formSet := make(map[ArgumentFormID]struct{}, len(input.ArgumentForms))
	for _, record := range input.ArgumentForms {
		formSet[record.meta.id] = struct{}{}
	}
	for _, commandRecord := range input.Commands {
		command := commandRecord.meta.id
		for _, form := range grammar.ArgumentForms(command) {
			if _, ok := formSet[form.id]; ok {
				result.ArgumentForms = append(result.ArgumentForms, form)
			}
		}
	}
	for _, profile := range input.GrammarProfiles {
		if dialect, ok := grammar.ProfileDialect(profile.meta.id); ok {
			result.ProfileDialects[profile.meta.id] = dialect
			result.ProfileCommands[profile.meta.id] = profile.command
			result.ProfileOrder = append(result.ProfileOrder, profile.meta.id)
		}
	}
	return result
}

func copyCommandSet(input map[CommandID]struct{}) map[CommandID]struct{} {
	result := make(map[CommandID]struct{}, len(input))
	for command := range input {
		result[command] = struct{}{}
	}
	return result
}

// closeValidationRecordReferencesLocked expands only exact relationship
// context needed to validate records already owned by selected. Commands added
// here are reference-only: their owned declaration families never enter input.
func (model *effectiveModel) closeValidationRecordReferencesLocked(input declarationRecordInput, selected, validationCommands map[CommandID]struct{}) declarationRecordInput {
	addValidationAncestors := func(command CommandID) {
		for command.present() {
			if _, exists := validationCommands[command]; exists {
				return
			}
			validationCommands[command] = struct{}{}
			record, exists := model.commandLocked(command)
			if !exists {
				return
			}
			parent, exists := record.Parent()
			if !exists {
				return
			}
			command = parent
		}
	}
	for _, entry := range input.CommandEntries {
		addValidationAncestors(entry.target)
	}
	for _, record := range input.Defaults {
		addValidationAncestors(record.target)
	}

	destinationSet := make(map[ValueID]struct{}, len(input.Destinations))
	for _, record := range input.Destinations {
		destinationSet[record.meta.id] = struct{}{}
	}
	profileSet := make(map[GrammarProfileID]struct{}, len(input.GrammarProfiles))
	for _, record := range input.GrammarProfiles {
		profileSet[record.meta.id] = struct{}{}
	}
	addValidationProfile := func(profile GrammarProfileID) {
		if !profile.present() {
			return
		}
		if _, exists := profileSet[profile]; exists {
			return
		}
		record, exists := model.grammarProfiles[profile]
		if !exists {
			record, exists = model.base.GrammarProfile(profile)
		}
		if !exists {
			return
		}
		profileSet[profile] = struct{}{}
		input.GrammarProfiles = append(input.GrammarProfiles, record)
		addValidationAncestors(record.command)
	}
	var addValidationDestination func(ValueID)
	addValidationDestination = func(destination ValueID) {
		if _, exists := destinationSet[destination]; exists {
			return
		}
		record, exists := model.destinationLocked(destination)
		if !exists {
			return
		}
		destinationSet[destination] = struct{}{}
		input.Destinations = append(input.Destinations, record)
		addValidationAncestors(record.command)
		addValidationProfile(record.meta.profile)
		for _, producer := range record.Producers() {
			addValidationProfile(producer.profile)
			for _, dependency := range producer.Dependencies() {
				addValidationDestination(dependency.Destination())
			}
		}
	}
	for _, entry := range input.CommandEntries {
		definition, shortcut := entry.Shortcut()
		if !shortcut {
			continue
		}
		for _, value := range definition.Defaults() {
			if value.Kind() == shortcutDefaultKindDestination {
				addValidationDestination(value.Destination())
			}
		}
	}
	processedCommands := make(map[CommandID]struct{}, len(validationCommands))
	for len(processedCommands) != len(validationCommands) {
		for command := range validationCommands {
			if _, processed := processedCommands[command]; processed {
				continue
			}
			processedCommands[command] = struct{}{}
			record, exists := model.commandLocked(command)
			if !exists {
				continue
			}
			access := record.access
			for _, relationship := range access.Values() {
				addValidationDestination(relationship.Source())
			}
			for _, relationship := range access.OptionalValues() {
				addValidationDestination(relationship.Source())
			}
			for _, relationship := range access.ValueStates() {
				addValidationDestination(relationship.Source())
			}
		}
	}

	if len(validationCommands) != len(selected) {
		commands := make([]commandRecord, 0, len(validationCommands))
		for command := range validationCommands {
			if record, exists := model.commandLocked(command); exists {
				commands = append(commands, record)
			}
		}
		slices.SortStableFunc(commands, func(left, right commandRecord) int { return cmp.Compare(left.meta.position, right.meta.position) })
		input.Commands = commands
	}
	sortRecordInput(&input)
	return input
}

func (model *effectiveModel) addValidationGrammarDialectsLocked(input *grammarInput, selected, validationCommands map[CommandID]struct{}) {
	grammar, ok := model.effectiveGrammarLocked()
	if !ok {
		return
	}
	for command := range validationCommands {
		if _, owner := selected[command]; owner {
			continue
		}
		if dialect, exists := grammar.Dialect(command); exists {
			input.Dialects[command] = dialect
		}
	}
}

func (model *effectiveModel) scopeRecordInputLocked(selected map[CommandID]struct{}) declarationRecordInput {
	actions, _ := model.base.IntrinsicActions()
	input := declarationRecordInput{
		IntrinsicActions: actions,
	}
	commands := make([]commandRecord, 0, len(selected))
	for command := range selected {
		if record, ok := model.commandLocked(command); ok {
			commands = append(commands, record)
		}
	}
	slices.SortStableFunc(commands, func(left, right commandRecord) int { return cmp.Compare(left.meta.position, right.meta.position) })
	input.Commands = commands

	flagSet := make(map[FlagID]struct{})
	formSet := make(map[ArgumentFormID]struct{})
	policySet := make(map[SignalPolicyID]struct{})
	destinationSet := make(map[ValueID]struct{})
	for _, command := range commands {
		id := command.meta.id
		for _, record := range model.destinationsForCommandLocked(id) {
			input.Destinations = append(input.Destinations, record)
			destinationSet[record.meta.id] = struct{}{}
		}
		for _, record := range model.flagsForCommandLocked(id) {
			input.Flags = append(input.Flags, record)
			flagSet[record.meta.id] = struct{}{}
		}
		for _, record := range model.formsForCommandLocked(id) {
			input.ArgumentForms = append(input.ArgumentForms, record)
			formSet[record.meta.id] = struct{}{}
		}
		input.CommandEntries = append(input.CommandEntries, model.commandEntriesForParentLocked(id)...)
		input.ActionEntries = append(input.ActionEntries, model.actionEntriesForParentLocked(id)...)
		input.Defaults = append(input.Defaults, model.defaultsForParentLocked(id)...)
		input.Providers = append(input.Providers, model.providersForParentLocked(id)...)
		input.Rules = append(input.Rules, model.rulesForCommandLocked(id)...)
		input.Advisories = append(input.Advisories, model.advisoriesForCommandLocked(id)...)
		for _, record := range model.signalPoliciesForCommandLocked(id) {
			input.SignalPolicies = append(input.SignalPolicies, record)
			policySet[record.meta.id] = struct{}{}
		}
		input.EnvironmentInputs = append(input.EnvironmentInputs, model.environmentForCommandLocked(id)...)
		input.GrammarProfiles = append(input.GrammarProfiles, model.profilesForCommandLocked(id)...)
		input.SignalEndpoints = append(input.SignalEndpoints, model.endpointsForCommandLocked(id)...)
		input.ConditionPolicies = append(input.ConditionPolicies, model.conditionsForCommandLocked(id)...)
		input.NoticeSuppressions = append(input.NoticeSuppressions, model.noticeForCommandLocked(id)...)
		input.SignalPolicyScopes = append(input.SignalPolicyScopes, model.signalScopesForCommandLocked(id)...)
		input.HelpGroups = append(input.HelpGroups, model.helpGroupsForCommandLocked(id)...)
	}
	for flag := range flagSet {
		input.FlagSpellings = append(input.FlagSpellings, model.flagSpellingsForFlagLocked(flag)...)
	}
	for form := range formSet {
		input.ArgumentTerms = append(input.ArgumentTerms, model.termsForFormLocked(form)...)
	}
	for policy := range policySet {
		input.SignalCounters = append(input.SignalCounters, model.countersForPolicyLocked(policy)...)
	}

	domainSet := make(map[int]struct{})
	for destination := range destinationSet {
		for _, index := range model.valueDomainIndexesForDestinationLocked(destination) {
			domainSet[index] = struct{}{}
		}
	}
	domainIndexes := make([]int, 0, len(domainSet))
	for index := range domainSet {
		domainIndexes = append(domainIndexes, index)
	}
	slices.Sort(domainIndexes)
	entrySet := make(map[ValueDomainEntryID]struct{})
	for _, index := range domainIndexes {
		domain, ok := model.valueDomainLocked(index)
		if !ok {
			continue
		}
		input.ValueDomains = append(input.ValueDomains, domain)
		for _, entry := range domain.Entries() {
			entrySet[entry] = struct{}{}
		}
	}
	for entry := range entrySet {
		if record, ok := model.domainEntryLocked(entry); ok {
			input.DomainEntries = append(input.DomainEntries, record)
		}
	}

	seenReplacements := make(map[declarationReplacementRecord]struct{})
	for command := range selected {
		for _, replacement := range model.replacementsForCommandLocked(command) {
			if _, seen := seenReplacements[replacement]; seen {
				continue
			}
			seenReplacements[replacement] = struct{}{}
			input.Replacements = append(input.Replacements, replacement)
		}
	}
	sortRecordInput(&input)
	return input
}

func (model *effectiveModel) scopeGrammarInputLocked(selected map[CommandID]struct{}, profiles []grammarProfileRecord) grammarInput {
	grammar, ok := model.effectiveGrammarLocked()
	if !ok {
		panic("invocation: semantic view base has no grammar")
	}
	input := grammarInput{
		Root:            grammar.Root(),
		Dialects:        make(map[CommandID]dialect),
		ProfileDialects: make(map[GrammarProfileID]dialect),
		ProfileCommands: make(map[GrammarProfileID]CommandID),
	}
	commands := make([]CommandID, 0, len(selected))
	positions := make(map[CommandID]declarationPosition, len(selected))
	for command := range selected {
		record, _ := model.commandLocked(command)
		commands = append(commands, command)
		positions[command] = record.meta.position
	}
	slices.SortStableFunc(commands, func(left, right CommandID) int {
		return cmp.Compare(positions[left], positions[right])
	})
	for _, command := range commands {
		if dialect, ok := grammar.Dialect(command); ok {
			input.Dialects[command] = dialect
		}
		input.CommandEntries = append(input.CommandEntries, grammar.CommandEntries(command)...)
		input.ActionCommands = append(input.ActionCommands, grammar.ActionCommands(command)...)
		input.ActionFlags = append(input.ActionFlags, grammar.ActionFlags(command)...)
		input.Flags = append(input.Flags, grammar.Flags(command)...)
		input.ArgumentForms = append(input.ArgumentForms, grammar.ArgumentForms(command)...)
	}
	slices.SortStableFunc(profiles, func(left, right grammarProfileRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	for _, profile := range profiles {
		if dialect, ok := grammar.ProfileDialect(profile.meta.id); ok {
			input.ProfileDialects[profile.meta.id] = dialect
			input.ProfileCommands[profile.meta.id] = profile.command
			input.ProfileOrder = append(input.ProfileOrder, profile.meta.id)
		}
	}
	return input
}

func sortRecordInput(input *declarationRecordInput) {
	slices.SortStableFunc(input.Destinations, func(left, right destinationRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.Flags, func(left, right flagRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.FlagSpellings, func(left, right flagSpellingRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.ArgumentForms, func(left, right argumentFormRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.ArgumentTerms, func(left, right argumentTermRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.CommandEntries, func(left, right commandEntryRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.ActionEntries, func(left, right actionEntryRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.Defaults, func(left, right defaultRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.Providers, func(left, right providerRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.Rules, func(left, right ruleRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.Advisories, func(left, right advisoryRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.SignalPolicies, func(left, right signalPolicyRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.DomainEntries, func(left, right domainEntryRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.EnvironmentInputs, func(left, right environmentInputRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.GrammarProfiles, func(left, right grammarProfileRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.SignalEndpoints, func(left, right signalEndpointRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.SignalCounters, func(left, right signalCounterRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	slices.SortStableFunc(input.ConditionPolicies, func(left, right conditionPolicyRecord) int {
		return cmp.Compare(left.position, right.position)
	})
	slices.SortStableFunc(input.NoticeSuppressions, func(left, right noticeSuppressionRecord) int {
		return cmp.Compare(left.position, right.position)
	})
	slices.SortStableFunc(input.SignalPolicyScopes, func(left, right signalPolicyScope) int {
		return cmp.Compare(left.Position(), right.Position())
	})
	slices.SortStableFunc(input.HelpGroups, func(left, right helpGroupRecord) int {
		return cmp.Compare(left.position, right.position)
	})
	slices.SortStableFunc(input.Replacements, func(left, right declarationReplacementRecord) int {
		return cmp.Compare(left.position, right.position)
	})
}
