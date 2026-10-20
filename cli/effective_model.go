package cli

import (
	"cmp"
	"slices"
	"sync"
)

func (operation *invocation) ProviderPublications() []providerPublication {
	if operation == nil || operation.analysis == nil || operation.analysis.effectiveModel == nil {
		return nil
	}
	return operation.analysis.effectiveModel.ProviderPublications()
}

// LocalizationForCommand resolves presentation ownership from the nearest
// independently localized provider scope, falling back to application-owned
// localization.
func (operation *invocation) LocalizationForCommand(command CommandID) localizationSnapshot {
	if operation == nil || operation.analysis.effectiveModel == nil || !command.present() {
		return localizationSnapshot{}
	}
	if localization := operation.analysis.effectiveModel.ProviderLocalizationForCommand(command); localization.Enabled() {
		return localization
	}
	return operation.program.Application().localization
}

// LocalizationForCommandEntry preserves provider ownership for alias/shortcut
// presentation contributed directly into an existing command namespace.
func (operation *invocation) LocalizationForCommandEntry(entry CommandEntryID) localizationSnapshot {
	if operation == nil || operation.analysis.effectiveModel == nil || !entry.present() {
		return localizationSnapshot{}
	}
	if localization := operation.analysis.effectiveModel.ProviderLocalizationForCommandEntry(entry); localization.Enabled() {
		return localization
	}
	return operation.program.Application().localization
}

type providerValueDomainLocator struct {
	publication int
	domain      int
}

type providerConditionPolicyRecord struct {
	record      conditionPolicyRecord
	publication int
	declaration int
}

type providerNoticeSuppressionRecord struct {
	record      noticeSuppressionRecord
	publication int
	declaration int
}

// effectiveModel is the operation-local read authority for effective semantic
// state. Canonical truth is the immutable base model plus accepted immutable
// provider and dynamic-domain publications. All operation-local maps below are
// delta/override accelerators and are rebuildable from those facts. Grammar
// reads compose the immutable base grammar with accepted provider deltas only
// at the bounded namespace requested by the caller.
type effectiveModel struct {
	mu sync.RWMutex

	base                 staticProgram
	providers            []providerPublication
	domains              []dynamicDomainPublication
	grammar              providerGrammarIndex
	providerValueDomains []providerValueDomainLocator

	commands             map[CommandID]commandRecord
	destinations         map[ValueID]destinationRecord
	flags                map[FlagID]flagRecord
	flagSpellings        map[FlagSpellingID]flagSpellingRecord
	argumentForms        map[ArgumentFormID]argumentFormRecord
	argumentTerms        map[ArgumentTermID]argumentTermRecord
	commandEntries       map[CommandEntryID]commandEntryRecord
	actionEntries        map[ActionEntryID]actionEntryRecord
	defaults             map[CommandDefaultID]defaultRecord
	providerRecords      map[CommandProviderID]providerRecord
	rules                map[RuleID]ruleRecord
	advisories           map[advisoryID]advisoryRecord
	signalPolicies       map[SignalPolicyID]signalPolicyRecord
	domainEntries        map[ValueDomainEntryID]domainEntryRecord
	environmentInputs    map[EnvironmentInputID]environmentInputRecord
	grammarProfiles      map[GrammarProfileID]grammarProfileRecord
	signalEndpoints      map[SignalEndpointID]signalEndpointRecord
	signalCounters       map[SignalCounterID]signalCounterRecord
	valueDomainOverrides map[int]valueDomainRecord

	providersByOwner          map[CommandID][]providerRecord
	domainsByDestination      map[ValueID][]int
	childrenByCommand         map[CommandID][]CommandID
	destinationsByCommand     map[CommandID][]destinationRecord
	flagsByCommand            map[CommandID][]flagRecord
	flagSpellingsByFlag       map[FlagID][]flagSpellingRecord
	formsByCommand            map[CommandID][]argumentFormRecord
	termsByForm               map[ArgumentFormID][]argumentTermRecord
	entriesByParent           map[CommandID][]commandEntryRecord
	actionsByParent           map[CommandID][]actionEntryRecord
	defaultsByParent          map[CommandID][]defaultRecord
	rulesByCommand            map[CommandID][]ruleRecord
	advisoriesByCommand       map[CommandID][]advisoryRecord
	signalPoliciesByCommand   map[CommandID][]signalPolicyRecord
	environmentByCommand      map[CommandID][]environmentInputRecord
	profilesByCommand         map[CommandID][]grammarProfileRecord
	endpointsByCommand        map[CommandID][]signalEndpointRecord
	countersByPolicy          map[SignalPolicyID][]signalCounterRecord
	conditionsByCommand       map[CommandID][]providerConditionPolicyRecord
	noticeByCommand           map[CommandID][]providerNoticeSuppressionRecord
	signalScopesByCommand     map[CommandID][]signalPolicyScope
	helpGroupsByCommand       map[CommandID][]helpGroupRecord
	replacementsByCommand     map[CommandID][]declarationReplacementRecord
	providerLocalization      map[CommandID]localizationSnapshot
	providerEntryLocalization map[CommandEntryID]localizationSnapshot
	providerByCommandEntry    map[CommandEntryID]CommandProviderID
}

func prepareEffectiveModel(base staticProgram) *effectiveModel {
	if !base.Valid() {
		panic("application: invalid semantic view base")
	}
	model := &effectiveModel{
		base: base,
	}
	model.resetDerivedLocked()
	return model
}

func (model *effectiveModel) Application() applicationMetadata {
	return model.base.Application()
}

func (model *effectiveModel) Valid() bool { return model != nil && model.base.Valid() }

func (model *effectiveModel) schemaSnapshot() (staticRecords, grammarView, bool) {
	if model == nil {
		return staticRecords{}, nil, false
	}
	model.mu.RLock()
	defer model.mu.RUnlock()

	grammar, ok := model.effectiveGrammarLocked()
	if !ok {
		return staticRecords{}, nil, false
	}
	return model.recordsSnapshotLocked(), grammar, true
}

func (model *effectiveModel) recordsSnapshotLocked() staticRecords {
	intrinsicActions, _ := model.base.IntrinsicActions()
	input := declarationRecordInput{
		IntrinsicActions: intrinsicActions,
		Commands: appendProviderFamily(
			model.base.Commands(),
			model.providers,
			func(delta declarationDelta) []commandRecord { return delta.Commands() },
		),
		Destinations: appendProviderFamily(
			model.base.Destinations(),
			model.providers,
			func(delta declarationDelta) []destinationRecord { return delta.Destinations() },
		),
		Flags: appendProviderFamily(
			model.base.Flags(),
			model.providers,
			func(delta declarationDelta) []flagRecord { return delta.Flags() },
		),
		FlagSpellings: appendProviderFamily(
			model.base.FlagSpellings(),
			model.providers,
			func(delta declarationDelta) []flagSpellingRecord { return delta.FlagSpellings() },
		),
		ArgumentForms: appendProviderFamily(
			model.base.ArgumentForms(),
			model.providers,
			func(delta declarationDelta) []argumentFormRecord { return delta.ArgumentForms() },
		),
		ArgumentTerms: appendProviderFamily(
			model.base.ArgumentTerms(),
			model.providers,
			func(delta declarationDelta) []argumentTermRecord { return delta.ArgumentTerms() },
		),
		CommandEntries: appendProviderFamily(
			model.base.CommandEntries(),
			model.providers,
			func(delta declarationDelta) []commandEntryRecord { return delta.CommandEntries() },
		),
		ActionEntries: appendProviderFamily(
			model.base.ActionEntries(),
			model.providers,
			func(delta declarationDelta) []actionEntryRecord { return delta.ActionEntries() },
		),
		Defaults: appendProviderFamily(
			model.base.Defaults(),
			model.providers,
			func(delta declarationDelta) []defaultRecord { return delta.Defaults() },
		),
		Providers: appendProviderFamily(
			model.base.Providers(),
			model.providers,
			func(delta declarationDelta) []providerRecord { return delta.Providers() },
		),
		Rules: appendProviderFamily(
			model.base.Rules(),
			model.providers,
			func(delta declarationDelta) []ruleRecord { return delta.Rules() },
		),
		Advisories: appendProviderFamily(
			model.base.Advisories(),
			model.providers,
			func(delta declarationDelta) []advisoryRecord { return delta.Advisories() },
		),
		SignalPolicies: appendProviderFamily(
			model.base.SignalPolicies(),
			model.providers,
			func(delta declarationDelta) []signalPolicyRecord { return delta.SignalPolicies() },
		),
		DomainEntries: appendProviderFamily(
			model.base.DomainEntries(),
			model.providers,
			func(delta declarationDelta) []domainEntryRecord { return delta.DomainEntries() },
		),
		EnvironmentInputs: appendProviderFamily(
			model.base.EnvironmentInputs(),
			model.providers,
			func(delta declarationDelta) []environmentInputRecord { return delta.EnvironmentInputs() },
		),
		GrammarProfiles: appendProviderFamily(
			model.base.GrammarProfiles(),
			model.providers,
			func(delta declarationDelta) []grammarProfileRecord { return delta.GrammarProfiles() },
		),
		SignalEndpoints: appendProviderFamily(
			model.base.SignalEndpoints(),
			model.providers,
			func(delta declarationDelta) []signalEndpointRecord { return delta.SignalEndpoints() },
		),
		SignalCounters: appendProviderFamily(
			model.base.SignalCounters(),
			model.providers,
			func(delta declarationDelta) []signalCounterRecord { return delta.SignalCounters() },
		),
		ConditionPolicies: appendProviderFamily(
			model.base.ConditionPolicies(),
			model.providers,
			func(delta declarationDelta) []conditionPolicyRecord { return delta.ConditionPolicies() },
		),
		NoticeSuppressions: appendProviderFamily(
			model.base.NoticeSuppressions(),
			model.providers,
			func(delta declarationDelta) []noticeSuppressionRecord { return delta.NoticeSuppressions() },
		),
		SignalPolicyScopes: appendProviderFamily(
			model.base.SignalPolicyScopes(),
			model.providers,
			func(delta declarationDelta) []signalPolicyScope { return delta.SignalPolicyScopes() },
		),
		HelpGroups: appendProviderFamily(
			model.base.HelpGroups(),
			model.providers,
			func(delta declarationDelta) []helpGroupRecord { return delta.HelpGroups() },
		),
		Replacements: appendProviderFamily(
			model.base.Replacements(),
			model.providers,
			func(delta declarationDelta) []declarationReplacementRecord { return delta.Replacements() },
		),
	}
	for index, record := range input.Destinations {
		if effective, ok := model.destinations[record.meta.id]; ok {
			input.Destinations[index] = effective
		}
	}
	for _, publication := range model.domains {
		input.DomainEntries = append(input.DomainEntries, publication.Entries()...)
	}
	count := model.valueDomainCountLocked()
	input.ValueDomains = make([]valueDomainRecord, 0, count)
	for index := range count {
		record, ok := model.valueDomainLocked(index)
		if ok {
			input.ValueDomains = append(input.ValueDomains, record)
		}
	}

	return freezeStaticRecords(input)
}

func (model *effectiveModel) IdentityBaseline() staticIdentityBaseline {
	return model.base.IdentityBaseline()
}

func (model *effectiveModel) ProviderPublications() []providerPublication {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return append([]providerPublication(nil), model.providers...)
}

func (model *effectiveModel) CommitProvider(publication providerPublication) {
	model.mu.Lock()
	defer model.mu.Unlock()
	model.providers = append(model.providers, publication)
	model.applyProviderDerivedLocked(publication, len(model.providers))
}

func (model *effectiveModel) applyProviderDerivedLocked(publication providerPublication, frontier int) {
	delta := publication.Declarations()
	domains := delta.ValueDomains()
	domainBase := model.valueDomainCountLocked()
	publicationIndex := frontier - 1
	model.applyDeclarationDeltaLocked(delta)
	model.grammar.add(publication.Grammar(), frontier)
	model.indexDeclarationDeltaScopesLocked(delta, publicationIndex)
	for command, localization := range publication.CommandLocalization() {
		if localization.Enabled() {
			if model.providerLocalization == nil {
				model.providerLocalization = make(map[CommandID]localizationSnapshot)
			}
			model.providerLocalization[command] = localization
		}
	}
	for entry, localization := range publication.EntryLocalization() {
		if localization.Enabled() {
			if model.providerEntryLocalization == nil {
				model.providerEntryLocalization = make(map[CommandEntryID]localizationSnapshot)
			}
			model.providerEntryLocalization[entry] = localization
		}
	}
	for provider, entries := range publication.ProviderEntries() {
		for _, entry := range entries {
			if model.providerByCommandEntry == nil {
				model.providerByCommandEntry = make(map[CommandEntryID]CommandProviderID)
			}
			model.providerByCommandEntry[entry] = provider
		}
	}
	for _, record := range delta.Providers() {
		if model.providersByOwner == nil {
			model.providersByOwner = make(map[CommandID][]providerRecord)
		}
		model.providersByOwner[record.parent] = append(model.providersByOwner[record.parent], record)
	}
	model.indexValueDomainsLocked(domains, domainBase)
	for domain := range domains {
		model.providerValueDomains = append(model.providerValueDomains, providerValueDomainLocator{
			publication: publicationIndex,
			domain:      domain,
		})
	}
}

func (model *effectiveModel) CommitDynamicDomain(publication dynamicDomainPublication) {
	model.mu.Lock()
	defer model.mu.Unlock()
	model.domains = append(model.domains, publication)
	model.applyDynamicDomainDerivedLocked(publication)
}

func (model *effectiveModel) applyDynamicDomainDerivedLocked(publication dynamicDomainPublication) {
	index := publication.Domain()
	if index < 0 || index >= model.valueDomainCountLocked() {
		panic("application: dynamic-domain publication outside semantic view")
	}
	entries := publication.Entries()
	entryIDs := make([]ValueDomainEntryID, len(entries))
	for i, record := range entries {
		if model.domainEntries == nil {
			model.domainEntries = make(map[ValueDomainEntryID]domainEntryRecord)
		}
		model.domainEntries[record.meta.id] = record
		entryIDs[i] = record.meta.id
	}
	domain, ok := model.valueDomainLocked(index)
	if !ok {
		panic("application: dynamic-domain publication missing canonical domain")
	}
	domain.entries = append([]ValueDomainEntryID(nil), entryIDs...)
	if model.valueDomainOverrides == nil {
		model.valueDomainOverrides = make(map[int]valueDomainRecord)
	}
	model.valueDomainOverrides[index] = domain
	for _, use := range domain.Uses() {
		record, ok := model.destinationLocked(use.destination)
		if !ok {
			panic("application: dynamic-domain destination missing from semantic view")
		}
		definition, ok := record.Definition()
		if !ok {
			panic("application: dynamic-domain destination lacks definition")
		}
		effective, ok := materializedValueDomainDefinition(definition, use.kind, publication.Language())
		if !ok {
			panic("application: invalid accepted dynamic-domain destination derivation")
		}
		if model.destinations == nil {
			model.destinations = make(map[ValueID]destinationRecord)
		}
		record.definition = effective
		model.destinations[record.meta.id] = record
	}
}

func (model *effectiveModel) ProviderForCommandEntry(entry CommandEntryID) (CommandProviderID, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	provider, ok := model.providerByCommandEntry[entry]
	return provider, ok
}

func (model *effectiveModel) ProvidersForParent(owner CommandID) []providerRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	result := model.base.ProvidersForParent(owner)
	return append(result, model.providersByOwner[owner]...)
}

func (model *effectiveModel) DestinationsForCommand(command CommandID) []destinationRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.destinationsForCommandLocked(command)
}

func (model *effectiveModel) destinationsForCommandsOrdered(commands []CommandID) []destinationRecord {
	model.mu.RLock()
	var result []destinationRecord
	for _, command := range commands {
		for _, record := range model.base.destinationsByCommand[command] {
			if effective, ok := model.destinations[record.meta.id]; ok {
				record = effective
			}
			result = append(result, record)
		}
		result = append(result, model.destinationsByCommand[command]...)
	}
	model.mu.RUnlock()

	slices.SortStableFunc(result, func(left, right destinationRecord) int {
		return cmp.Compare(left.meta.position, right.meta.position)
	})
	return result
}

func (model *effectiveModel) ValueDomainIndexesForDestination(destination ValueID) []int {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.valueDomainIndexesForDestinationLocked(destination)
}

func (model *effectiveModel) ValueDomain(index int) (valueDomainRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.valueDomainLocked(index)
}

func (model *effectiveModel) ProviderLocalizationForCommand(command CommandID) localizationSnapshot {
	model.mu.RLock()
	defer model.mu.RUnlock()
	current := command
	for current.present() {
		if localization, ok := model.providerLocalization[current]; ok {
			return localization
		}
		record, ok := model.commandLocked(current)
		if !ok {
			break
		}
		parent, ok := record.Parent()
		if !ok {
			break
		}
		current = parent
	}
	return localizationSnapshot{}
}

func (model *effectiveModel) ProviderLocalizationForCommandEntry(entry CommandEntryID) localizationSnapshot {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if localization, ok := model.providerEntryLocalization[entry]; ok {
		return localization
	}
	record, ok := model.commandEntryLocked(entry)
	if !ok {
		return localizationSnapshot{}
	}
	current := record.parent
	for current.present() {
		if localization, ok := model.providerLocalization[current]; ok {
			return localization
		}
		command, ok := model.commandLocked(current)
		if !ok {
			break
		}
		parent, ok := command.Parent()
		if !ok {
			break
		}
		current = parent
	}
	return localizationSnapshot{}
}

func appendProviderFamily[T any](base []T, publications []providerPublication, extract func(declarationDelta) []T) []T {
	result := base
	for _, publication := range publications {
		result = append(result, extract(publication.Declarations())...)
	}
	return result
}

func (model *effectiveModel) IntrinsicActions() (intrinsicActions, bool) {
	return model.base.IntrinsicActions()
}

func (model *effectiveModel) Commands() []commandRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.Commands(),
		model.providers,
		func(delta declarationDelta) []commandRecord { return delta.Commands() },
	)
}

func (model *effectiveModel) Destinations() []destinationRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	result := appendProviderFamily(
		model.base.Destinations(),
		model.providers,
		func(delta declarationDelta) []destinationRecord { return delta.Destinations() },
	)
	for i, record := range result {
		if effective, ok := model.destinations[record.meta.id]; ok {
			result[i] = effective
		}
	}
	return result
}

func (model *effectiveModel) Flags() []flagRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.Flags(),
		model.providers,
		func(delta declarationDelta) []flagRecord { return delta.Flags() },
	)
}

func (model *effectiveModel) FlagsForCommand(id CommandID) []flagRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.flagsForCommandLocked(id)
}

func (model *effectiveModel) FlagSpellings() []flagSpellingRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.FlagSpellings(),
		model.providers,
		func(delta declarationDelta) []flagSpellingRecord {
			return delta.FlagSpellings()
		},
	)
}

func (model *effectiveModel) ArgumentForms() []argumentFormRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.ArgumentForms(),
		model.providers,
		func(delta declarationDelta) []argumentFormRecord {
			return delta.ArgumentForms()
		},
	)
}

func (model *effectiveModel) ArgumentTerms() []argumentTermRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.ArgumentTerms(),
		model.providers,
		func(delta declarationDelta) []argumentTermRecord {
			return delta.ArgumentTerms()
		},
	)
}

func (model *effectiveModel) CommandEntries() []commandEntryRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.CommandEntries(),
		model.providers,
		func(delta declarationDelta) []commandEntryRecord {
			return delta.CommandEntries()
		},
	)
}

func (model *effectiveModel) ActionEntries() []actionEntryRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.ActionEntries(),
		model.providers,
		func(delta declarationDelta) []actionEntryRecord { return delta.ActionEntries() },
	)
}

func (model *effectiveModel) Defaults() []defaultRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.Defaults(),
		model.providers,
		func(delta declarationDelta) []defaultRecord { return delta.Defaults() },
	)
}

func (model *effectiveModel) DefaultsForParent(id CommandID) []defaultRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.defaultsForParentLocked(id)
}

func (model *effectiveModel) Providers() []providerRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.Providers(),
		model.providers,
		func(delta declarationDelta) []providerRecord { return delta.Providers() },
	)
}

func (model *effectiveModel) Rules() []ruleRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.Rules(),
		model.providers,
		func(delta declarationDelta) []ruleRecord { return delta.Rules() },
	)
}

func (model *effectiveModel) Advisories() []advisoryRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.Advisories(),
		model.providers,
		func(delta declarationDelta) []advisoryRecord { return delta.Advisories() },
	)
}

func (model *effectiveModel) RulesForCommand(id CommandID) []ruleRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.rulesForCommandLocked(id)
}

func (model *effectiveModel) AdvisoriesForCommand(id CommandID) []advisoryRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.advisoriesForCommandLocked(id)
}

func (model *effectiveModel) SignalPolicies() []signalPolicyRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.SignalPolicies(),
		model.providers,
		func(delta declarationDelta) []signalPolicyRecord {
			return delta.SignalPolicies()
		},
	)
}

func (model *effectiveModel) SignalPoliciesForCommand(id CommandID) []signalPolicyRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.signalPoliciesForCommandLocked(id)
}

func (model *effectiveModel) DomainEntries() []domainEntryRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	result := appendProviderFamily(
		model.base.DomainEntries(),
		model.providers,
		func(delta declarationDelta) []domainEntryRecord { return delta.DomainEntries() },
	)
	for _, publication := range model.domains {
		result = append(result, publication.Entries()...)
	}
	return result
}

func (model *effectiveModel) ValueDomains() []valueDomainRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	count := model.valueDomainCountLocked()
	result := make([]valueDomainRecord, 0, count)
	for i := range count {
		record, ok := model.valueDomainLocked(i)
		if ok {
			result = append(result, record)
		}
	}
	return result
}

func (model *effectiveModel) EnvironmentInputs() []environmentInputRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.EnvironmentInputs(),
		model.providers,
		func(delta declarationDelta) []environmentInputRecord {
			return delta.EnvironmentInputs()
		},
	)
}

func (model *effectiveModel) GrammarProfiles() []grammarProfileRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.GrammarProfiles(),
		model.providers,
		func(delta declarationDelta) []grammarProfileRecord {
			return delta.GrammarProfiles()
		},
	)
}

func (model *effectiveModel) GrammarProfilesForCommand(id CommandID) []grammarProfileRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.profilesForCommandLocked(id)
}

func (model *effectiveModel) EnvironmentInputsForCommand(id CommandID) []environmentInputRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.environmentForCommandLocked(id)
}

func (model *effectiveModel) SignalEndpoints() []signalEndpointRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.SignalEndpoints(),
		model.providers,
		func(delta declarationDelta) []signalEndpointRecord {
			return delta.SignalEndpoints()
		},
	)
}

func (model *effectiveModel) SignalCounters() []signalCounterRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.SignalCounters(),
		model.providers,
		func(delta declarationDelta) []signalCounterRecord {
			return delta.SignalCounters()
		},
	)
}

func (model *effectiveModel) ConditionPolicies() []conditionPolicyRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.ConditionPolicies(),
		model.providers,
		func(delta declarationDelta) []conditionPolicyRecord {
			return delta.ConditionPolicies()
		},
	)
}

func (model *effectiveModel) ConditionPoliciesForCommand(id CommandID) []conditionPolicyRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.conditionsForCommandLocked(id)
}

// ConditionPoliciesForCommandsOrdered projects only the selected command
// scopes while preserving the same base-then-provider observable order as the
// complete operation-effective condition-policy family. The immutable model
// retains the canonical static full-family ordinal in its scoped index.
// Provider declaration positions are provider-local, so the existing scoped
// overlay retains each accepted publication/declaration coordinate needed for
// cross-owner order.
func (model *effectiveModel) ConditionPoliciesForCommandsOrdered(ids []CommandID) []conditionPolicyRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()

	base := model.base.ConditionPoliciesForCommandsOrdered(ids)
	selected := make(map[CommandID]struct{}, len(ids))
	var overlay []providerConditionPolicyRecord
	for _, id := range ids {
		if _, duplicate := selected[id]; duplicate {
			continue
		}
		selected[id] = struct{}{}
		overlay = append(overlay, model.conditionsByCommand[id]...)
	}

	slices.SortStableFunc(overlay, func(left, right providerConditionPolicyRecord) int {
		return cmp.Or(
			cmp.Compare(left.publication, right.publication),
			cmp.Compare(left.declaration, right.declaration),
		)
	})

	result := base
	for _, indexed := range overlay {
		result = append(result, indexed.record)
	}
	return result
}

func (model *effectiveModel) HelpGroupsForCommand(id CommandID) []helpGroupRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.helpGroupsForCommandLocked(id)
}

func (model *effectiveModel) ReplacementsForCommand(id CommandID) []declarationReplacementRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.replacementsForCommandLocked(id)
}

func (model *effectiveModel) NoticeSuppressions() []noticeSuppressionRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.NoticeSuppressions(),
		model.providers,
		func(delta declarationDelta) []noticeSuppressionRecord {
			return delta.NoticeSuppressions()
		},
	)
}

func (model *effectiveModel) NoticeSuppressionsForCommand(id CommandID) []noticeSuppressionRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.noticeForCommandLocked(id)
}

// NoticeSuppressionsForCommandsOrdered projects only the selected command
// scopes while preserving the same base-then-provider observable order as the
// complete operation-effective suppression family.
func (model *effectiveModel) NoticeSuppressionsForCommandsOrdered(ids []CommandID) []noticeSuppressionRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()

	selected := make(map[CommandID]struct{}, len(ids))
	var overlay []providerNoticeSuppressionRecord
	var result []noticeSuppressionRecord
	for _, id := range ids {
		if _, duplicate := selected[id]; duplicate {
			continue
		}
		selected[id] = struct{}{}
		result = append(result, model.base.NoticeSuppressionsForCommand(id)...)
		overlay = append(overlay, model.noticeByCommand[id]...)
	}
	slices.SortStableFunc(result, func(left, right noticeSuppressionRecord) int { return cmp.Compare(left.position, right.position) })
	slices.SortStableFunc(overlay, func(left, right providerNoticeSuppressionRecord) int {
		return cmp.Or(
			cmp.Compare(left.publication, right.publication),
			cmp.Compare(left.declaration, right.declaration),
		)
	})
	for _, indexed := range overlay {
		result = append(result, indexed.record)
	}
	return result
}

func (model *effectiveModel) SignalPolicyScopes() []signalPolicyScope {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.SignalPolicyScopes(),
		model.providers,
		func(delta declarationDelta) []signalPolicyScope {
			return delta.SignalPolicyScopes()
		},
	)
}

func (model *effectiveModel) SignalPolicyScopesForCommand(id CommandID) []signalPolicyScope {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.signalScopesForCommandLocked(id)
}

func (model *effectiveModel) HelpGroups() []helpGroupRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.HelpGroups(),
		model.providers,
		func(delta declarationDelta) []helpGroupRecord { return delta.HelpGroups() },
	)
}

func (model *effectiveModel) Replacements() []declarationReplacementRecord {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return appendProviderFamily(
		model.base.Replacements(),
		model.providers,
		func(delta declarationDelta) []declarationReplacementRecord {
			return delta.Replacements()
		},
	)
}

// GrammarSnapshot returns the operation-local bounded grammar read facet and
// the matching accepted-provider generation. The facet owns no semantic state:
// every answer is derived from the immutable base GrammarModel plus accepted
// ProviderPublication GrammarDelta facts.
func (model *effectiveModel) GrammarSnapshot() (grammarView, int, bool) {
	if model == nil {
		return nil, 0, false
	}
	model.mu.RLock()
	defer model.mu.RUnlock()
	grammar, ok := model.effectiveGrammarLocked()
	if !ok {
		return nil, 0, false
	}
	return grammar, len(model.providers), true
}

func (model *effectiveModel) EffectiveGrammar() (grammarView, bool) {
	grammar, _, ok := model.GrammarSnapshot()
	return grammar, ok
}

func (model *effectiveModel) Command(id CommandID) (commandRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.commandLocked(id)
}

func (model *effectiveModel) Destination(id ValueID) (destinationRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.destinations[id]; ok {
		return record, true
	}
	return model.base.Destination(id)
}

func (model *effectiveModel) Flag(id FlagID) (flagRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.flags[id]; ok {
		return record, true
	}
	return model.base.Flag(id)
}

func (model *effectiveModel) FlagSpelling(id FlagSpellingID) (flagSpellingRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.flagSpellings[id]; ok {
		return record, true
	}
	return model.base.FlagSpelling(id)
}

func (model *effectiveModel) ArgumentForm(id ArgumentFormID) (argumentFormRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.argumentForms[id]; ok {
		return record, true
	}
	return model.base.ArgumentForm(id)
}

func (model *effectiveModel) ArgumentTerm(id ArgumentTermID) (argumentTermRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.argumentTerms[id]; ok {
		return record, true
	}
	return model.base.ArgumentTerm(id)
}

func (model *effectiveModel) CommandEntry(id CommandEntryID) (commandEntryRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	return model.commandEntryLocked(id)
}

func (model *effectiveModel) ActionEntry(id ActionEntryID) (actionEntryRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.actionEntries[id]; ok {
		return record, true
	}
	return model.base.ActionEntry(id)
}

func (model *effectiveModel) Default(id CommandDefaultID) (defaultRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.defaults[id]; ok {
		return record, true
	}
	return model.base.Default(id)
}

func (model *effectiveModel) Provider(id CommandProviderID) (providerRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.providerRecords[id]; ok {
		return record, true
	}
	return model.base.Provider(id)
}

func (model *effectiveModel) Rule(id RuleID) (ruleRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.rules[id]; ok {
		return record, true
	}
	return model.base.Rule(id)
}

func (model *effectiveModel) Advisory(id advisoryID) (advisoryRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.advisories[id]; ok {
		return record, true
	}
	return model.base.Advisory(id)
}

func (model *effectiveModel) SignalPolicy(id SignalPolicyID) (signalPolicyRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.signalPolicies[id]; ok {
		return record, true
	}
	return model.base.SignalPolicy(id)
}

func (model *effectiveModel) DomainEntry(id ValueDomainEntryID) (domainEntryRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.domainEntries[id]; ok {
		return record, true
	}
	return model.base.DomainEntry(id)
}

func (model *effectiveModel) EnvironmentInput(id EnvironmentInputID) (environmentInputRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.environmentInputs[id]; ok {
		return record, true
	}
	return model.base.EnvironmentInput(id)
}

func (model *effectiveModel) GrammarProfile(id GrammarProfileID) (grammarProfileRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.grammarProfiles[id]; ok {
		return record, true
	}
	return model.base.GrammarProfile(id)
}

func (model *effectiveModel) SignalEndpoint(id SignalEndpointID) (signalEndpointRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.signalEndpoints[id]; ok {
		return record, true
	}
	return model.base.SignalEndpoint(id)
}

func (model *effectiveModel) SignalCounter(id SignalCounterID) (signalCounterRecord, bool) {
	model.mu.RLock()
	defer model.mu.RUnlock()
	if record, ok := model.signalCounters[id]; ok {
		return record, true
	}
	return model.base.SignalCounter(id)
}

func (model *effectiveModel) commandLocked(id CommandID) (commandRecord, bool) {
	if record, ok := model.commands[id]; ok {
		return record, true
	}
	return model.base.Command(id)
}

func (model *effectiveModel) destinationLocked(id ValueID) (destinationRecord, bool) {
	if record, ok := model.destinations[id]; ok {
		return record, true
	}
	return model.base.Destination(id)
}

func (model *effectiveModel) flagLocked(id FlagID) (flagRecord, bool) {
	if record, ok := model.flags[id]; ok {
		return record, true
	}
	return model.base.Flag(id)
}

func (model *effectiveModel) argumentFormLocked(id ArgumentFormID) (argumentFormRecord, bool) {
	if record, ok := model.argumentForms[id]; ok {
		return record, true
	}
	return model.base.ArgumentForm(id)
}

func (model *effectiveModel) argumentTermLocked(id ArgumentTermID) (argumentTermRecord, bool) {
	if record, ok := model.argumentTerms[id]; ok {
		return record, true
	}
	return model.base.ArgumentTerm(id)
}

func (model *effectiveModel) commandEntryLocked(id CommandEntryID) (commandEntryRecord, bool) {
	if record, ok := model.commandEntries[id]; ok {
		return record, true
	}
	return model.base.CommandEntry(id)
}

func (model *effectiveModel) resetDerivedLocked() {
	model.grammar = providerGrammarIndex{}
	model.providerValueDomains = nil
	model.commands = nil
	model.destinations = nil
	model.flags = nil
	model.flagSpellings = nil
	model.argumentForms = nil
	model.argumentTerms = nil
	model.commandEntries = nil
	model.actionEntries = nil
	model.defaults = nil
	model.providerRecords = nil
	model.rules = nil
	model.advisories = nil
	model.signalPolicies = nil
	model.domainEntries = nil
	model.environmentInputs = nil
	model.grammarProfiles = nil
	model.signalEndpoints = nil
	model.signalCounters = nil
	model.valueDomainOverrides = nil
	model.providersByOwner = nil
	model.domainsByDestination = nil
	model.childrenByCommand = nil
	model.destinationsByCommand = nil
	model.flagsByCommand = nil
	model.flagSpellingsByFlag = nil
	model.formsByCommand = nil
	model.termsByForm = nil
	model.entriesByParent = nil
	model.actionsByParent = nil
	model.defaultsByParent = nil
	model.rulesByCommand = nil
	model.advisoriesByCommand = nil
	model.signalPoliciesByCommand = nil
	model.environmentByCommand = nil
	model.profilesByCommand = nil
	model.endpointsByCommand = nil
	model.countersByPolicy = nil
	model.conditionsByCommand = nil
	model.noticeByCommand = nil
	model.signalScopesByCommand = nil
	model.helpGroupsByCommand = nil
	model.replacementsByCommand = nil
	model.providerLocalization = nil
	model.providerEntryLocalization = nil
	model.providerByCommandEntry = nil
	for index, publication := range model.providers {
		model.applyProviderDerivedLocked(publication, index+1)
	}
	for _, publication := range model.domains {
		model.applyDynamicDomainDerivedLocked(publication)
	}
}

func (model *effectiveModel) indexDeclarationDeltaScopesLocked(delta declarationDelta, publication int) {
	for _, record := range delta.Commands() {
		if parent, ok := record.Parent(); ok {
			if model.childrenByCommand == nil {
				model.childrenByCommand = make(map[CommandID][]CommandID)
			}
			model.childrenByCommand[parent] = append(model.childrenByCommand[parent], record.meta.id)
		}
	}
	for _, record := range delta.Destinations() {
		if model.destinationsByCommand == nil {
			model.destinationsByCommand = make(map[CommandID][]destinationRecord)
		}
		model.destinationsByCommand[record.command] = append(model.destinationsByCommand[record.command], record)
	}
	for _, record := range delta.Flags() {
		if model.flagsByCommand == nil {
			model.flagsByCommand = make(map[CommandID][]flagRecord)
		}
		model.flagsByCommand[record.command] = append(model.flagsByCommand[record.command], record)
	}
	for _, record := range delta.FlagSpellings() {
		if model.flagSpellingsByFlag == nil {
			model.flagSpellingsByFlag = make(map[FlagID][]flagSpellingRecord)
		}
		model.flagSpellingsByFlag[record.flag] = append(model.flagSpellingsByFlag[record.flag], record)
	}
	for _, record := range delta.ArgumentForms() {
		if model.formsByCommand == nil {
			model.formsByCommand = make(map[CommandID][]argumentFormRecord)
		}
		model.formsByCommand[record.command] = append(model.formsByCommand[record.command], record)
	}
	for _, record := range delta.ArgumentTerms() {
		if model.termsByForm == nil {
			model.termsByForm = make(map[ArgumentFormID][]argumentTermRecord)
		}
		model.termsByForm[record.form] = append(model.termsByForm[record.form], record)
	}
	for _, record := range delta.CommandEntries() {
		if model.entriesByParent == nil {
			model.entriesByParent = make(map[CommandID][]commandEntryRecord)
		}
		model.entriesByParent[record.parent] = append(model.entriesByParent[record.parent], record)
	}
	for _, record := range delta.ActionEntries() {
		if model.actionsByParent == nil {
			model.actionsByParent = make(map[CommandID][]actionEntryRecord)
		}
		model.actionsByParent[record.parent] = append(model.actionsByParent[record.parent], record)
	}
	for _, record := range delta.Defaults() {
		if model.defaultsByParent == nil {
			model.defaultsByParent = make(map[CommandID][]defaultRecord)
		}
		model.defaultsByParent[record.parent] = append(model.defaultsByParent[record.parent], record)
	}
	for _, record := range delta.Rules() {
		if model.rulesByCommand == nil {
			model.rulesByCommand = make(map[CommandID][]ruleRecord)
		}
		model.rulesByCommand[record.command] = append(model.rulesByCommand[record.command], record)
	}
	for _, record := range delta.Advisories() {
		if model.advisoriesByCommand == nil {
			model.advisoriesByCommand = make(map[CommandID][]advisoryRecord)
		}
		model.advisoriesByCommand[record.command] = append(model.advisoriesByCommand[record.command], record)
	}
	for _, record := range delta.SignalPolicies() {
		if model.signalPoliciesByCommand == nil {
			model.signalPoliciesByCommand = make(map[CommandID][]signalPolicyRecord)
		}
		model.signalPoliciesByCommand[record.command] = append(
			model.signalPoliciesByCommand[record.command],
			record,
		)
	}
	for _, record := range delta.EnvironmentInputs() {
		if model.environmentByCommand == nil {
			model.environmentByCommand = make(map[CommandID][]environmentInputRecord)
		}
		model.environmentByCommand[record.command] = append(model.environmentByCommand[record.command], record)
	}
	for _, record := range delta.GrammarProfiles() {
		if model.profilesByCommand == nil {
			model.profilesByCommand = make(map[CommandID][]grammarProfileRecord)
		}
		model.profilesByCommand[record.command] = append(model.profilesByCommand[record.command], record)
	}
	for _, record := range delta.SignalEndpoints() {
		if model.endpointsByCommand == nil {
			model.endpointsByCommand = make(map[CommandID][]signalEndpointRecord)
		}
		model.endpointsByCommand[record.command] = append(model.endpointsByCommand[record.command], record)
	}
	for _, record := range delta.SignalCounters() {
		if model.countersByPolicy == nil {
			model.countersByPolicy = make(map[SignalPolicyID][]signalCounterRecord)
		}
		model.countersByPolicy[record.policy] = append(model.countersByPolicy[record.policy], record)
	}
	for declaration, record := range delta.ConditionPolicies() {
		if owner, ok := model.conditionOwnerLocked(record.target); ok {
			if model.conditionsByCommand == nil {
				model.conditionsByCommand = make(map[CommandID][]providerConditionPolicyRecord)
			}
			indexed := providerConditionPolicyRecord{
				record:      record,
				publication: publication,
				declaration: declaration,
			}
			model.conditionsByCommand[owner] = append(model.conditionsByCommand[owner], indexed)
		}
	}
	for declaration, record := range delta.NoticeSuppressions() {
		if model.noticeByCommand == nil {
			model.noticeByCommand = make(map[CommandID][]providerNoticeSuppressionRecord)
		}
		indexed := providerNoticeSuppressionRecord{
			record:      record,
			publication: publication,
			declaration: declaration,
		}
		command := record.command
		model.noticeByCommand[command] = append(model.noticeByCommand[command], indexed)
	}
	for _, scope := range delta.SignalPolicyScopes() {
		if model.signalScopesByCommand == nil {
			model.signalScopesByCommand = make(map[CommandID][]signalPolicyScope)
		}
		model.signalScopesByCommand[scope.Command()] = append(model.signalScopesByCommand[scope.Command()], scope)
	}
	for _, record := range delta.HelpGroups() {
		if model.helpGroupsByCommand == nil {
			model.helpGroupsByCommand = make(map[CommandID][]helpGroupRecord)
		}
		model.helpGroupsByCommand[record.command] = append(model.helpGroupsByCommand[record.command], record)
	}
	for _, replacement := range delta.Replacements() {
		seen := make(map[CommandID]struct{})
		for _, identity := range []declarationReplacementIdentity{replacement.source, replacement.target} {
			if command, ok := model.replacementIdentityCommandLocked(identity); ok {
				if _, duplicate := seen[command]; !duplicate {
					if model.replacementsByCommand == nil {
						model.replacementsByCommand = make(
							map[CommandID][]declarationReplacementRecord,
						)
					}
					model.replacementsByCommand[command] = append(model.replacementsByCommand[command], replacement)
					seen[command] = struct{}{}
				}
			}
		}
	}
}

func appendScoped[T any](base, delta []T) []T {
	return append(base, delta...)
}

func (model *effectiveModel) childrenForCommandLocked(id CommandID) []CommandID {
	return appendScoped(model.base.Children(id), model.childrenByCommand[id])
}

func (model *effectiveModel) destinationsForCommandLocked(id CommandID) []destinationRecord {
	result := model.base.DestinationsForCommand(id)
	for i, record := range result {
		if effective, ok := model.destinations[record.meta.id]; ok {
			result[i] = effective
		}
	}
	return append(result, model.destinationsByCommand[id]...)
}

func (model *effectiveModel) flagsForCommandLocked(id CommandID) []flagRecord {
	return appendScoped(model.base.FlagsForCommand(id), model.flagsByCommand[id])
}

func (model *effectiveModel) flagSpellingsForFlagLocked(id FlagID) []flagSpellingRecord {
	return appendScoped(model.base.FlagSpellingsForFlag(id), model.flagSpellingsByFlag[id])
}

func (model *effectiveModel) formsForCommandLocked(id CommandID) []argumentFormRecord {
	return appendScoped(model.base.ArgumentFormsForCommand(id), model.formsByCommand[id])
}

func (model *effectiveModel) termsForFormLocked(id ArgumentFormID) []argumentTermRecord {
	return appendScoped(model.base.ArgumentTermsForForm(id), model.termsByForm[id])
}

func (model *effectiveModel) commandEntriesForParentLocked(id CommandID) []commandEntryRecord {
	return appendScoped(model.base.CommandEntriesForParent(id), model.entriesByParent[id])
}

func (model *effectiveModel) actionEntriesForParentLocked(id CommandID) []actionEntryRecord {
	return appendScoped(model.base.ActionEntriesForParent(id), model.actionsByParent[id])
}

func (model *effectiveModel) defaultsForParentLocked(id CommandID) []defaultRecord {
	return appendScoped(model.base.DefaultsForParent(id), model.defaultsByParent[id])
}

func (model *effectiveModel) providersForParentLocked(id CommandID) []providerRecord {
	return appendScoped(model.base.ProvidersForParent(id), model.providersByOwner[id])
}

func (model *effectiveModel) rulesForCommandLocked(id CommandID) []ruleRecord {
	return appendScoped(model.base.RulesForCommand(id), model.rulesByCommand[id])
}

func (model *effectiveModel) advisoriesForCommandLocked(id CommandID) []advisoryRecord {
	return appendScoped(model.base.AdvisoriesForCommand(id), model.advisoriesByCommand[id])
}

func (model *effectiveModel) signalPoliciesForCommandLocked(id CommandID) []signalPolicyRecord {
	return appendScoped(model.base.SignalPoliciesForCommand(id), model.signalPoliciesByCommand[id])
}

func (model *effectiveModel) environmentForCommandLocked(id CommandID) []environmentInputRecord {
	return appendScoped(model.base.EnvironmentInputsForCommand(id), model.environmentByCommand[id])
}

func (model *effectiveModel) profilesForCommandLocked(id CommandID) []grammarProfileRecord {
	return appendScoped(model.base.GrammarProfilesForCommand(id), model.profilesByCommand[id])
}

func (model *effectiveModel) endpointsForCommandLocked(id CommandID) []signalEndpointRecord {
	return appendScoped(model.base.SignalEndpointsForCommand(id), model.endpointsByCommand[id])
}

func (model *effectiveModel) countersForPolicyLocked(id SignalPolicyID) []signalCounterRecord {
	return appendScoped(model.base.SignalCountersForPolicy(id), model.countersByPolicy[id])
}

func (model *effectiveModel) conditionsForCommandLocked(id CommandID) []conditionPolicyRecord {
	base := model.base.ConditionPoliciesForCommand(id)
	overlay := model.conditionsByCommand[id]
	result := base
	for _, indexed := range overlay {
		result = append(result, indexed.record)
	}
	return result
}

func (model *effectiveModel) noticeForCommandLocked(id CommandID) []noticeSuppressionRecord {
	result := model.base.NoticeSuppressionsForCommand(id)
	for _, indexed := range model.noticeByCommand[id] {
		result = append(result, indexed.record)
	}
	return result
}

func (model *effectiveModel) signalScopesForCommandLocked(id CommandID) []signalPolicyScope {
	return appendScoped(model.base.SignalPolicyScopesForCommand(id), model.signalScopesByCommand[id])
}

func (model *effectiveModel) helpGroupsForCommandLocked(id CommandID) []helpGroupRecord {
	return appendScoped(model.base.HelpGroupsForCommand(id), model.helpGroupsByCommand[id])
}

func (model *effectiveModel) valueDomainIndexesForDestinationLocked(id ValueID) []int {
	return appendScoped(model.base.ValueDomainIndexesForDestination(id), model.domainsByDestination[id])
}

func (model *effectiveModel) replacementsForCommandLocked(id CommandID) []declarationReplacementRecord {
	return appendScoped(model.base.ReplacementsForCommand(id), model.replacementsByCommand[id])
}

func (model *effectiveModel) domainEntryLocked(id ValueDomainEntryID) (domainEntryRecord, bool) {
	if r, ok := model.domainEntries[id]; ok {
		return r, true
	}
	return model.base.DomainEntry(id)
}

func (model *effectiveModel) flagSpellingLocked(id FlagSpellingID) (flagSpellingRecord, bool) {
	if r, ok := model.flagSpellings[id]; ok {
		return r, true
	}
	return model.base.FlagSpelling(id)
}

func (model *effectiveModel) environmentInputLocked(id EnvironmentInputID) (environmentInputRecord, bool) {
	if r, ok := model.environmentInputs[id]; ok {
		return r, true
	}
	return model.base.EnvironmentInput(id)
}

func (model *effectiveModel) replacementIdentityCommandLocked(identity declarationReplacementIdentity) (CommandID, bool) {
	switch identity.Kind() {
	case declarationReplacementKindCommandEntry:
		r, ok := model.commandEntryLocked(identity.CommandEntry())
		if !ok {
			return CommandID(0), false
		}
		return r.parent, true
	case declarationReplacementKindFlag:
		r, ok := model.flagLocked(identity.Flag())
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	case declarationReplacementKindFlagSpelling:
		r, ok := model.flagSpellingLocked(identity.FlagSpelling())
		if !ok {
			return CommandID(0), false
		}
		f, ok := model.flagLocked(r.flag)
		if !ok {
			return CommandID(0), false
		}
		return f.command, true
	case declarationReplacementKindArgumentForm:
		r, ok := model.argumentFormLocked(identity.ArgumentForm())
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	case declarationReplacementKindArgumentTerm:
		r, ok := model.argumentTermLocked(identity.ArgumentTerm())
		if !ok {
			return CommandID(0), false
		}
		f, ok := model.argumentFormLocked(r.form)
		if !ok {
			return CommandID(0), false
		}
		return f.command, true
	case declarationReplacementKindEnvironmentInput:
		r, ok := model.environmentInputLocked(identity.EnvironmentInput())
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	default:
		return CommandID(0), false
	}
}

func (model *effectiveModel) conditionOwnerLocked(target conditionTarget) (CommandID, bool) {
	switch target.Kind() {
	case conditionTargetCommand:
		return target.Command(), target.Command().present()
	case conditionTargetFlag:
		record, ok := model.flagLocked(target.Flag())
		if !ok {
			return CommandID(0), false
		}
		return record.command, true
	case conditionTargetArgumentForm:
		record, ok := model.argumentFormLocked(target.ArgumentForm())
		if !ok {
			return CommandID(0), false
		}
		return record.command, true
	case conditionTargetArgumentTerm:
		term, ok := model.argumentTermLocked(target.ArgumentTerm())
		if !ok {
			return CommandID(0), false
		}
		form, ok := model.argumentFormLocked(term.form)
		if !ok {
			return CommandID(0), false
		}
		return form.command, true
	default:
		return CommandID(0), false
	}
}

func (model *effectiveModel) applyDeclarationDeltaLocked(delta declarationDelta) {
	model.commands = putAll(
		model.commands,
		delta.Commands(),
		func(r commandRecord) CommandID { return r.meta.id },
	)
	model.destinations = putAll(
		model.destinations,
		delta.Destinations(),
		func(r destinationRecord) ValueID { return r.meta.id },
	)
	model.flags = putAll(
		model.flags,
		delta.Flags(),
		func(r flagRecord) FlagID { return r.meta.id },
	)
	model.flagSpellings = putAll(
		model.flagSpellings,
		delta.FlagSpellings(),
		func(r flagSpellingRecord) FlagSpellingID { return r.meta.id },
	)
	model.argumentForms = putAll(
		model.argumentForms,
		delta.ArgumentForms(),
		func(r argumentFormRecord) ArgumentFormID { return r.meta.id },
	)
	model.argumentTerms = putAll(
		model.argumentTerms,
		delta.ArgumentTerms(),
		func(r argumentTermRecord) ArgumentTermID { return r.meta.id },
	)
	model.commandEntries = putAll(
		model.commandEntries,
		delta.CommandEntries(),
		func(r commandEntryRecord) CommandEntryID { return r.meta.id },
	)
	model.actionEntries = putAll(
		model.actionEntries,
		delta.ActionEntries(),
		func(r actionEntryRecord) ActionEntryID { return r.meta.id },
	)
	model.defaults = putAll(
		model.defaults,
		delta.Defaults(),
		func(r defaultRecord) CommandDefaultID { return r.meta.id },
	)
	model.providerRecords = putAll(
		model.providerRecords,
		delta.Providers(),
		func(r providerRecord) CommandProviderID { return r.meta.id },
	)
	model.rules = putAll(
		model.rules,
		delta.Rules(),
		func(r ruleRecord) RuleID { return r.meta.id },
	)
	model.advisories = putAll(
		model.advisories,
		delta.Advisories(),
		func(r advisoryRecord) advisoryID { return r.meta.id },
	)
	model.signalPolicies = putAll(
		model.signalPolicies,
		delta.SignalPolicies(),
		func(r signalPolicyRecord) SignalPolicyID { return r.meta.id },
	)
	model.domainEntries = putAll(
		model.domainEntries,
		delta.DomainEntries(),
		func(r domainEntryRecord) ValueDomainEntryID { return r.meta.id },
	)
	model.environmentInputs = putAll(
		model.environmentInputs,
		delta.EnvironmentInputs(),
		func(r environmentInputRecord) EnvironmentInputID { return r.meta.id },
	)
	model.grammarProfiles = putAll(
		model.grammarProfiles,
		delta.GrammarProfiles(),
		func(r grammarProfileRecord) GrammarProfileID { return r.meta.id },
	)
	model.signalEndpoints = putAll(
		model.signalEndpoints,
		delta.SignalEndpoints(),
		func(r signalEndpointRecord) SignalEndpointID { return r.meta.id },
	)
	model.signalCounters = putAll(
		model.signalCounters,
		delta.SignalCounters(),
		func(r signalCounterRecord) SignalCounterID { return r.meta.id },
	)
}

func (model *effectiveModel) valueDomainCountLocked() int {
	return model.base.ValueDomainCount() + len(model.providerValueDomains)
}

func (model *effectiveModel) valueDomainLocked(index int) (valueDomainRecord, bool) {
	if record, ok := model.valueDomainOverrides[index]; ok {
		return cloneValueDomainRecord(record), true
	}
	baseCount := model.base.ValueDomainCount()
	if index >= 0 && index < baseCount {
		record, ok := model.base.ValueDomain(index)
		return record, ok
	}
	offset := index - baseCount
	if offset < 0 || offset >= len(model.providerValueDomains) {
		return valueDomainRecord{}, false
	}
	locator := model.providerValueDomains[offset]
	if locator.publication < 0 || locator.publication >= len(model.providers) {
		return valueDomainRecord{}, false
	}
	return model.providers[locator.publication].Declarations().ValueDomain(locator.domain)
}

func (model *effectiveModel) indexValueDomainsLocked(domains []valueDomainRecord, base int) {
	for offset, domain := range domains {
		for _, use := range domain.Uses() {
			if model.domainsByDestination == nil {
				model.domainsByDestination = make(map[ValueID][]int)
			}
			model.domainsByDestination[use.destination] = append(
				model.domainsByDestination[use.destination],
				base+offset,
			)
		}
	}
}

func putAll[I comparable, R any](dst map[I]R, records []R, id func(R) I) map[I]R {
	if len(records) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[I]R, len(records))
	}
	for _, record := range records {
		dst[id(record)] = record
	}
	return dst
}
