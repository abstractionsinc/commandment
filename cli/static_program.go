package cli

import (
	"cmp"
	"slices"
)

// staticProgram is the immutable application-lifetime semantic substrate
// published by successful static compilation. It contains accepted canonical
// records plus private derived indexes. It owns no declaration-staging state.
type staticProgram struct {
	application applicationMetadata
	records     staticRecords
	baseline    staticIdentityBaseline
	grammar     grammarModel

	commands          map[CommandID]commandRecord
	destinations      map[ValueID]destinationRecord
	flags             map[FlagID]flagRecord
	flagSpellings     map[FlagSpellingID]flagSpellingRecord
	argumentForms     map[ArgumentFormID]argumentFormRecord
	argumentTerms     map[ArgumentTermID]argumentTermRecord
	commandEntries    map[CommandEntryID]commandEntryRecord
	actionEntries     map[ActionEntryID]actionEntryRecord
	defaults          map[CommandDefaultID]defaultRecord
	providers         map[CommandProviderID]providerRecord
	rules             map[RuleID]ruleRecord
	advisories        map[advisoryID]advisoryRecord
	signalPolicies    map[SignalPolicyID]signalPolicyRecord
	domainEntries     map[ValueDomainEntryID]domainEntryRecord
	environmentInputs map[EnvironmentInputID]environmentInputRecord
	grammarProfiles   map[GrammarProfileID]grammarProfileRecord
	signalEndpoints   map[SignalEndpointID]signalEndpointRecord
	signalCounters    map[SignalCounterID]signalCounterRecord

	childrenByCommand       map[CommandID][]CommandID
	destinationsByCommand   map[CommandID][]destinationRecord
	flagsByCommand          map[CommandID][]flagRecord
	flagSpellingsByFlag     map[FlagID][]flagSpellingRecord
	formsByCommand          map[CommandID][]argumentFormRecord
	termsByForm             map[ArgumentFormID][]argumentTermRecord
	entriesByParent         map[CommandID][]commandEntryRecord
	actionsByParent         map[CommandID][]actionEntryRecord
	defaultsByParent        map[CommandID][]defaultRecord
	providersByParent       map[CommandID][]providerRecord
	rulesByCommand          map[CommandID][]ruleRecord
	advisoriesByCommand     map[CommandID][]advisoryRecord
	signalPoliciesByCommand map[CommandID][]signalPolicyRecord
	environmentByCommand    map[CommandID][]environmentInputRecord
	profilesByCommand       map[CommandID][]grammarProfileRecord
	endpointsByCommand      map[CommandID][]signalEndpointRecord
	countersByPolicy        map[SignalPolicyID][]signalCounterRecord
	conditionsByCommand     map[CommandID][]staticConditionPolicyRecord
	noticeByCommand         map[CommandID][]noticeSuppressionRecord
	signalScopesByCommand   map[CommandID][]signalPolicyScope
	helpGroupsByCommand     map[CommandID][]helpGroupRecord
	domainsByDestination    map[ValueID][]int
	replacementsByCommand   map[CommandID][]declarationReplacementRecord
}

// freezeStaticProgram publishes accepted static records, grammar, and application metadata
// as one immutable application-lifetime semantic view. Zero grammar or metadata means that
// optional accepted partition was not declared.
func freezeStaticProgram(records staticRecords, baseline staticIdentityBaseline, grammar grammarModel, application applicationMetadata) staticProgram {
	if !baseline.valid {
		panic("semantic: invalid static identity baseline")
	}
	model := staticProgram{
		application: application,
		records:     records,
		baseline:    baseline,
		grammar:     grammar,
		commands: indexRecords(
			records.commands,
			func(record commandRecord) CommandID { return record.meta.id },
		),
		destinations: indexRecords(
			records.destinations,
			func(record destinationRecord) ValueID { return record.meta.id },
		),
		flags: indexRecords(records.flags, func(record flagRecord) FlagID { return record.meta.id }),
		flagSpellings: indexRecords(
			records.flagSpellings,
			func(record flagSpellingRecord) FlagSpellingID { return record.meta.id },
		),
		argumentForms: indexRecords(
			records.argumentForms,
			func(record argumentFormRecord) ArgumentFormID { return record.meta.id },
		),
		argumentTerms: indexRecords(
			records.argumentTerms,
			func(record argumentTermRecord) ArgumentTermID { return record.meta.id },
		),
		commandEntries: indexRecords(
			records.commandEntries,
			func(record commandEntryRecord) CommandEntryID { return record.meta.id },
		),
		actionEntries: indexRecords(
			records.actionEntries,
			func(record actionEntryRecord) ActionEntryID { return record.meta.id },
		),
		defaults: indexRecords(
			records.defaults,
			func(record defaultRecord) CommandDefaultID { return record.meta.id },
		),
		providers: indexRecords(
			records.providers,
			func(record providerRecord) CommandProviderID { return record.meta.id },
		),
		rules: indexRecords(records.rules, func(record ruleRecord) RuleID { return record.meta.id }),
		advisories: indexRecords(
			records.advisories,
			func(record advisoryRecord) advisoryID { return record.meta.id },
		),
		signalPolicies: indexRecords(
			records.signalPolicies,
			func(record signalPolicyRecord) SignalPolicyID { return record.meta.id },
		),
		domainEntries: indexRecords(
			records.domainEntries,
			func(record domainEntryRecord) ValueDomainEntryID { return record.meta.id },
		),
		environmentInputs: indexRecords(
			records.environmentInputs,
			func(record environmentInputRecord) EnvironmentInputID { return record.meta.id },
		),
		grammarProfiles: indexRecords(
			records.grammarProfiles,
			func(record grammarProfileRecord) GrammarProfileID { return record.meta.id },
		),
		signalEndpoints: indexRecords(
			records.signalEndpoints,
			func(record signalEndpointRecord) SignalEndpointID { return record.meta.id },
		),
		signalCounters: indexRecords(
			records.signalCounters,
			func(record signalCounterRecord) SignalCounterID { return record.meta.id },
		),
	}
	buildDeclarationScopeIndexes(&model)
	return model
}

func indexRecords[I comparable, R any](records []R, identity func(R) I) map[I]R {
	result := make(map[I]R, len(records))
	for _, record := range records {
		id := identity(record)
		if _, exists := result[id]; exists {
			panic("semantic: duplicate record identity")
		}
		result[id] = record
	}
	return result
}

func (model staticProgram) Application() applicationMetadata { return model.application }

func (model staticProgram) Valid() bool { return model.baseline.valid }

func (model staticProgram) Records() staticRecords { return model.records }

func (model staticProgram) IdentityBaseline() staticIdentityBaseline { return model.baseline }

func (model staticProgram) Grammar() (grammarModel, bool) {
	return model.grammar, model.grammar.Valid()
}

// EffectiveGrammar returns the immutable grammar visible at this reader's
// current frontier. A static view has no operation-local additions, so its
// compiled GrammarModel is already the effective reader.
func (model staticProgram) EffectiveGrammar() (grammarView, bool) {
	grammar, ok := model.Grammar()
	if !ok {
		return nil, false
	}
	return grammar, true
}

func (model staticProgram) Command(id CommandID) (commandRecord, bool) {
	record, ok := model.commands[id]
	return record, ok
}

func (model staticProgram) Destination(id ValueID) (destinationRecord, bool) {
	record, ok := model.destinations[id]
	return record, ok
}

func (model staticProgram) Flag(id FlagID) (flagRecord, bool) {
	record, ok := model.flags[id]
	return record, ok
}

func (model staticProgram) FlagSpelling(id FlagSpellingID) (flagSpellingRecord, bool) {
	record, ok := model.flagSpellings[id]
	return record, ok
}

func (model staticProgram) ArgumentForm(id ArgumentFormID) (argumentFormRecord, bool) {
	record, ok := model.argumentForms[id]
	return record, ok
}

func (model staticProgram) ArgumentTerm(id ArgumentTermID) (argumentTermRecord, bool) {
	record, ok := model.argumentTerms[id]
	return record, ok
}

func (model staticProgram) CommandEntry(id CommandEntryID) (commandEntryRecord, bool) {
	record, ok := model.commandEntries[id]
	return record, ok
}

func (model staticProgram) ActionEntry(id ActionEntryID) (actionEntryRecord, bool) {
	record, ok := model.actionEntries[id]
	return record, ok
}

func (model staticProgram) Default(id CommandDefaultID) (defaultRecord, bool) {
	record, ok := model.defaults[id]
	return record, ok
}

func (model staticProgram) Provider(id CommandProviderID) (providerRecord, bool) {
	record, ok := model.providers[id]
	return record, ok
}

func (model staticProgram) Rule(id RuleID) (ruleRecord, bool) {
	record, ok := model.rules[id]
	return record, ok
}

func (model staticProgram) Advisory(id advisoryID) (advisoryRecord, bool) {
	record, ok := model.advisories[id]
	return record, ok
}

func (model staticProgram) SignalPolicy(id SignalPolicyID) (signalPolicyRecord, bool) {
	record, ok := model.signalPolicies[id]
	return record, ok
}

func (model staticProgram) DomainEntry(id ValueDomainEntryID) (domainEntryRecord, bool) {
	record, ok := model.domainEntries[id]
	return record, ok
}

func (model staticProgram) EnvironmentInput(id EnvironmentInputID) (environmentInputRecord, bool) {
	record, ok := model.environmentInputs[id]
	return record, ok
}

func (model staticProgram) GrammarProfile(id GrammarProfileID) (grammarProfileRecord, bool) {
	record, ok := model.grammarProfiles[id]
	return record, ok
}

func (model staticProgram) SignalEndpoint(id SignalEndpointID) (signalEndpointRecord, bool) {
	record, ok := model.signalEndpoints[id]
	return record, ok
}

func (model staticProgram) SignalCounter(id SignalCounterID) (signalCounterRecord, bool) {
	record, ok := model.signalCounters[id]
	return record, ok
}

func (model staticProgram) IntrinsicActions() (intrinsicActions, bool) {
	return model.records.IntrinsicActions()
}

func (model staticProgram) Commands() []commandRecord { return model.records.Commands() }

func (model staticProgram) Destinations() []destinationRecord { return model.records.Destinations() }

func (model staticProgram) Flags() []flagRecord { return model.records.Flags() }

func (model staticProgram) FlagSpellings() []flagSpellingRecord {
	return model.records.FlagSpellings()
}

func (model staticProgram) ArgumentForms() []argumentFormRecord {
	return model.records.ArgumentForms()
}

func (model staticProgram) ArgumentTerms() []argumentTermRecord {
	return model.records.ArgumentTerms()
}

func (model staticProgram) CommandEntries() []commandEntryRecord {
	return model.records.CommandEntries()
}

func (model staticProgram) ActionEntries() []actionEntryRecord {
	return model.records.ActionEntries()
}

func (model staticProgram) Defaults() []defaultRecord { return model.records.Defaults() }

func (model staticProgram) Providers() []providerRecord { return model.records.Providers() }

func (model staticProgram) Rules() []ruleRecord { return model.records.Rules() }

func (model staticProgram) Advisories() []advisoryRecord { return model.records.Advisories() }

func (model staticProgram) SignalPolicies() []signalPolicyRecord {
	return model.records.SignalPolicies()
}

func (model staticProgram) DomainEntries() []domainEntryRecord {
	return model.records.DomainEntries()
}

func (model staticProgram) ValueDomains() []valueDomainRecord { return model.records.ValueDomains() }

func (model staticProgram) ValueDomainCount() int { return len(model.records.valueDomains) }

func (model staticProgram) ValueDomain(index int) (valueDomainRecord, bool) {
	if index < 0 || index >= len(model.records.valueDomains) {
		return valueDomainRecord{}, false
	}
	record := model.records.valueDomains[index]
	if dynamic, ok := record.DynamicDefinition(); ok {
		result := acceptDynamicValueDomainRecord(dynamic, record.Uses())
		result.entries = record.Entries()
		return result, true
	}
	return acceptValueDomainRecord(record.Entries(), record.Uses()), true
}

func (model staticProgram) EnvironmentInputs() []environmentInputRecord {
	return model.records.EnvironmentInputs()
}

func (model staticProgram) GrammarProfiles() []grammarProfileRecord {
	return model.records.GrammarProfiles()
}

func (model staticProgram) SignalEndpoints() []signalEndpointRecord {
	return model.records.SignalEndpoints()
}

func (model staticProgram) SignalCounters() []signalCounterRecord {
	return model.records.SignalCounters()
}

func (model staticProgram) ConditionPolicies() []conditionPolicyRecord {
	return model.records.ConditionPolicies()
}

func (model staticProgram) NoticeSuppressions() []noticeSuppressionRecord {
	return model.records.NoticeSuppressions()
}

func (model staticProgram) SignalPolicyScopes() []signalPolicyScope {
	return model.records.SignalPolicyScopes()
}

func (model staticProgram) HelpGroups() []helpGroupRecord { return model.records.HelpGroups() }

func (model staticProgram) Replacements() []declarationReplacementRecord {
	return model.records.Replacements()
}

// present reports whether a CommandID contains a nonzero canonical identity;
// sibling semantic identity types use the same private representation rule.
func (id CommandID) present() bool { return id != 0 }

func (id ValueID) present() bool { return id != 0 }

func (id FlagID) present() bool { return id != 0 }

func (id FlagSpellingID) present() bool { return id != 0 }

func (id ArgumentFormID) present() bool { return id != 0 }

func (id ArgumentTermID) present() bool { return id != 0 }

func (id CommandEntryID) present() bool { return id != 0 }

func (id ActionEntryID) present() bool { return id != 0 }

func (id CommandDefaultID) present() bool { return id != 0 }

func (id CommandProviderID) present() bool { return id != 0 }

func (id RuleID) present() bool { return id != 0 }

func (id SignalPolicyID) present() bool { return id != 0 }

func (id ValueDomainEntryID) present() bool { return id != 0 }

func (id EnvironmentInputID) present() bool { return id != 0 }

func (id GrammarProfileID) present() bool { return id != 0 }

func (id SignalEndpointID) present() bool { return id != 0 }

func (id SignalCounterID) present() bool { return id != 0 }

func (id ValueCandidateID) present() bool { return id != 0 }

func (id InputSourceID) present() bool { return id.value != 0 }

func (id InputOriginID) present() bool { return id.value != 0 }

func (id CommandID) uint32() uint32 { return uint32(id) }

func (id ValueID) uint32() uint32 { return uint32(id) }

func (id FlagID) uint32() uint32 { return uint32(id) }

func (id FlagSpellingID) uint32() uint32 { return uint32(id) }

func (id ArgumentFormID) uint32() uint32 { return uint32(id) }

func (id ArgumentTermID) uint32() uint32 { return uint32(id) }

func (id CommandEntryID) uint32() uint32 { return uint32(id) }

func (id ActionEntryID) uint32() uint32 { return uint32(id) }

func (id CommandDefaultID) uint32() uint32 { return uint32(id) }

func (id CommandProviderID) uint32() uint32 { return uint32(id) }

func (id RuleID) uint32() uint32 { return uint32(id) }

func (id SignalPolicyID) uint32() uint32 { return uint32(id) }

func (id ValueDomainEntryID) uint32() uint32 { return uint32(id) }

func (id EnvironmentInputID) uint32() uint32 { return uint32(id) }

func (id GrammarProfileID) uint32() uint32 { return uint32(id) }

func (id SignalEndpointID) uint32() uint32 { return uint32(id) }

func (id InputSourceID) uint32() uint32 { return id.value }

func (id InputOriginID) uint32() uint32 { return id.value }

type advisoryID uint32

func (id advisoryID) present() bool { return id != 0 }

type identityCounter struct {
	last uint32
}

func (counter *identityCounter) allocate() uint32 {
	if counter.last == ^uint32(0) {
		panic("semantic: identity family exhausted")
	}
	counter.last++
	return counter.last
}

type declarationIdentityState struct {
	sealed bool

	command          identityCounter
	destination      identityCounter
	flag             identityCounter
	flagSpelling     identityCounter
	argumentForm     identityCounter
	argumentTerm     identityCounter
	commandEntry     identityCounter
	actionEntry      identityCounter
	defaultCommand   identityCounter
	provider         identityCounter
	rule             identityCounter
	advisory         identityCounter
	signalPolicy     identityCounter
	domainEntry      identityCounter
	environmentInput identityCounter
	grammarProfile   identityCounter
	signalEndpoint   identityCounter
	signalCounter    identityCounter
}

type declarationIdentities struct {
	state *declarationIdentityState
}

func allocateDeclarationIdentities() declarationIdentities {
	return declarationIdentities{state: &declarationIdentityState{}}
}

func restoreDeclarationIdentities(baseline staticIdentityBaseline) declarationIdentities {
	if !baseline.valid {
		panic("semantic: invalid static identity baseline")
	}
	state := baseline.state
	state.sealed = false
	return declarationIdentities{state: &state}
}

func (ids declarationIdentities) mustState() *declarationIdentityState {
	if ids.state == nil {
		panic("semantic: invalid identity scope")
	}
	return ids.state
}

func (ids declarationIdentities) mutableState() *declarationIdentityState {
	state := ids.mustState()
	if state.sealed {
		panic("semantic: sealed identity scope")
	}
	return state
}

// AllocateCommandID and the sibling allocation methods are serialized
// commit-time operations. Deterministic semantic commit order, not goroutine
// scheduling, decides final IDs.
func (ids declarationIdentities) AllocateCommandID() CommandID {
	return CommandID(ids.mutableState().command.allocate())
}

func (ids declarationIdentities) AllocateDestinationID() ValueID {
	return ValueID(ids.mutableState().destination.allocate())
}

func (ids declarationIdentities) AllocateFlagID() FlagID {
	return FlagID(ids.mutableState().flag.allocate())
}

func (ids declarationIdentities) AllocateFlagSpellingID() FlagSpellingID {
	return FlagSpellingID(ids.mutableState().flagSpelling.allocate())
}

func (ids declarationIdentities) AllocateArgumentFormID() ArgumentFormID {
	return ArgumentFormID(ids.mutableState().argumentForm.allocate())
}

func (ids declarationIdentities) AllocateArgumentTermID() ArgumentTermID {
	return ArgumentTermID(ids.mutableState().argumentTerm.allocate())
}

func (ids declarationIdentities) AllocateCommandEntryID() CommandEntryID {
	return CommandEntryID(ids.mutableState().commandEntry.allocate())
}

func (ids declarationIdentities) AllocateActionEntryID() ActionEntryID {
	return ActionEntryID(ids.mutableState().actionEntry.allocate())
}

func (ids declarationIdentities) AllocateDefaultID() CommandDefaultID {
	return CommandDefaultID(ids.mutableState().defaultCommand.allocate())
}

func (ids declarationIdentities) AllocateProviderID() CommandProviderID {
	return CommandProviderID(ids.mutableState().provider.allocate())
}

func (ids declarationIdentities) AllocateRuleID() RuleID {
	return RuleID(ids.mutableState().rule.allocate())
}

func (ids declarationIdentities) AllocateAdvisoryID() advisoryID {
	return advisoryID(ids.mutableState().advisory.allocate())
}

func (ids declarationIdentities) AllocateSignalPolicyID() SignalPolicyID {
	return SignalPolicyID(ids.mutableState().signalPolicy.allocate())
}

func (ids declarationIdentities) AllocateDomainEntryID() ValueDomainEntryID {
	return ValueDomainEntryID(ids.mutableState().domainEntry.allocate())
}

func (ids declarationIdentities) AllocateEnvironmentInputID() EnvironmentInputID {
	return EnvironmentInputID(ids.mutableState().environmentInput.allocate())
}

func (ids declarationIdentities) AllocateGrammarProfileID() GrammarProfileID {
	return GrammarProfileID(ids.mutableState().grammarProfile.allocate())
}

func (ids declarationIdentities) AllocateSignalEndpointID() SignalEndpointID {
	return SignalEndpointID(ids.mutableState().signalEndpoint.allocate())
}

func (ids declarationIdentities) AllocateSignalCounterID() SignalCounterID {
	return SignalCounterID(ids.mutableState().signalCounter.allocate())
}

type staticIdentityScope struct {
	declarationIdentities
}

func allocateStaticIdentityScope() staticIdentityScope {
	return staticIdentityScope{declarationIdentities: allocateDeclarationIdentities()}
}

type staticIdentityBaseline struct {
	valid bool
	state declarationIdentityState
}

func (ids staticIdentityScope) Seal() staticIdentityBaseline {
	state := ids.mustState()
	state.sealed = true
	return staticIdentityBaseline{
		valid: true,
		state: *state,
	}
}

// declarationPosition is the source-program position of one committed
// declaration contribution inside its owning declaration stream. Position is
// separate from semantic identity and does not encode precedence.
type declarationPosition uint32

type recordMeta[I interface{ present() bool }] struct {
	id       I
	position declarationPosition
	profile  GrammarProfileID
}

func acceptRecordMeta[I interface{ present() bool }](id I, position declarationPosition) recordMeta[I] {
	if !id.present() {
		panic("semantic: invalid record identity")
	}
	return recordMeta[I]{
		id:       id,
		position: position,
	}
}

// declarationRecordInput is an acceptance-boundary carrier for exact typed record
// families. It is not semantic authority after construction; NewStaticRecords
// snapshots each slice into immutable canonical storage.
type declarationRecordInput struct {
	IntrinsicActions   intrinsicActions
	Commands           []commandRecord
	Destinations       []destinationRecord
	Flags              []flagRecord
	FlagSpellings      []flagSpellingRecord
	ArgumentForms      []argumentFormRecord
	ArgumentTerms      []argumentTermRecord
	CommandEntries     []commandEntryRecord
	ActionEntries      []actionEntryRecord
	Defaults           []defaultRecord
	Providers          []providerRecord
	Rules              []ruleRecord
	Advisories         []advisoryRecord
	SignalPolicies     []signalPolicyRecord
	DomainEntries      []domainEntryRecord
	ValueDomains       []valueDomainRecord
	EnvironmentInputs  []environmentInputRecord
	GrammarProfiles    []grammarProfileRecord
	SignalEndpoints    []signalEndpointRecord
	SignalCounters     []signalCounterRecord
	ConditionPolicies  []conditionPolicyRecord
	NoticeSuppressions []noticeSuppressionRecord
	SignalPolicyScopes []signalPolicyScope
	HelpGroups         []helpGroupRecord
	Replacements       []declarationReplacementRecord
}

// staticRecords is immutable canonical storage for accepted static semantic
// records. Exact typed families remain separate; there is no universal
// declaration ID, registry key, or string lookup authority.
type staticRecords struct {
	intrinsicActions   intrinsicActions
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

func freezeStaticRecords(input declarationRecordInput) staticRecords {
	result := staticRecords{
		intrinsicActions:   input.IntrinsicActions,
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
		valueDomains:       append([]valueDomainRecord(nil), input.ValueDomains...),
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

func (records staticRecords) IntrinsicActions() (intrinsicActions, bool) {
	return records.intrinsicActions, records.intrinsicActions.Valid()
}

func (records staticRecords) Commands() []commandRecord {
	return append([]commandRecord(nil), records.commands...)
}

func (records staticRecords) Destinations() []destinationRecord {
	return append([]destinationRecord(nil), records.destinations...)
}

func (records staticRecords) Flags() []flagRecord { return append([]flagRecord(nil), records.flags...) }

func (records staticRecords) FlagSpellings() []flagSpellingRecord {
	return append([]flagSpellingRecord(nil), records.flagSpellings...)
}

func (records staticRecords) ArgumentForms() []argumentFormRecord {
	return append([]argumentFormRecord(nil), records.argumentForms...)
}

func (records staticRecords) ArgumentTerms() []argumentTermRecord {
	return append([]argumentTermRecord(nil), records.argumentTerms...)
}

func (records staticRecords) CommandEntries() []commandEntryRecord {
	return append([]commandEntryRecord(nil), records.commandEntries...)
}

func (records staticRecords) ActionEntries() []actionEntryRecord {
	return append([]actionEntryRecord(nil), records.actionEntries...)
}

func (records staticRecords) Defaults() []defaultRecord {
	return append([]defaultRecord(nil), records.defaults...)
}

func (records staticRecords) Providers() []providerRecord {
	return append([]providerRecord(nil), records.providers...)
}

func (records staticRecords) Rules() []ruleRecord { return append([]ruleRecord(nil), records.rules...) }

func (records staticRecords) Advisories() []advisoryRecord {
	return append([]advisoryRecord(nil), records.advisories...)
}

func (records staticRecords) SignalPolicies() []signalPolicyRecord {
	return append([]signalPolicyRecord(nil), records.signalPolicies...)
}

func (records staticRecords) DomainEntries() []domainEntryRecord {
	return append([]domainEntryRecord(nil), records.domainEntries...)
}

func (records staticRecords) ValueDomains() []valueDomainRecord {
	result := make([]valueDomainRecord, len(records.valueDomains))
	for i, record := range records.valueDomains {
		if dynamic, ok := record.DynamicDefinition(); ok {
			copyRecord := acceptDynamicValueDomainRecord(dynamic, record.Uses())
			copyRecord.entries = record.Entries()
			result[i] = copyRecord
		} else {
			result[i] = acceptValueDomainRecord(record.Entries(), record.Uses())
		}
	}
	return result
}

func (records staticRecords) EnvironmentInputs() []environmentInputRecord {
	return append([]environmentInputRecord(nil), records.environmentInputs...)
}

func (records staticRecords) GrammarProfiles() []grammarProfileRecord {
	return append([]grammarProfileRecord(nil), records.grammarProfiles...)
}

func (records staticRecords) SignalEndpoints() []signalEndpointRecord {
	return append([]signalEndpointRecord(nil), records.signalEndpoints...)
}

func (records staticRecords) SignalCounters() []signalCounterRecord {
	return append([]signalCounterRecord(nil), records.signalCounters...)
}

func (records staticRecords) ConditionPolicies() []conditionPolicyRecord {
	return append([]conditionPolicyRecord(nil), records.conditionPolicies...)
}

func (records staticRecords) NoticeSuppressions() []noticeSuppressionRecord {
	return append([]noticeSuppressionRecord(nil), records.noticeSuppressions...)
}

func (records staticRecords) SignalPolicyScopes() []signalPolicyScope {
	return append([]signalPolicyScope(nil), records.signalPolicyScopes...)
}

func (records staticRecords) HelpGroups() []helpGroupRecord {
	return append([]helpGroupRecord(nil), records.helpGroups...)
}

func (records staticRecords) Replacements() []declarationReplacementRecord {
	return append([]declarationReplacementRecord(nil), records.replacements...)
}

func mustCommand(id CommandID) {
	if !id.present() {
		panic("semantic: invalid command identity")
	}
}

func mustDestination(id ValueID) {
	if !id.present() {
		panic("semantic: invalid destination identity")
	}
}

type staticConditionPolicyRecord struct {
	record  conditionPolicyRecord
	ordinal int
}

const replacementCommandCapacity = 2

// buildDeclarationScopeIndexes derives immutable static scope indexes once when
// the application-lifetime declarations are published. Operation-local models
// use these indexes as their base and store only publication overlays.
func buildDeclarationScopeIndexes(model *staticProgram) {
	model.childrenByCommand = make(map[CommandID][]CommandID)
	model.destinationsByCommand = make(map[CommandID][]destinationRecord)
	model.flagsByCommand = make(map[CommandID][]flagRecord)
	model.flagSpellingsByFlag = make(map[FlagID][]flagSpellingRecord)
	model.formsByCommand = make(map[CommandID][]argumentFormRecord)
	model.termsByForm = make(map[ArgumentFormID][]argumentTermRecord)
	model.entriesByParent = make(map[CommandID][]commandEntryRecord)
	model.actionsByParent = make(map[CommandID][]actionEntryRecord)
	model.defaultsByParent = make(map[CommandID][]defaultRecord)
	model.providersByParent = make(map[CommandID][]providerRecord)
	model.rulesByCommand = make(map[CommandID][]ruleRecord)
	model.advisoriesByCommand = make(map[CommandID][]advisoryRecord)
	model.signalPoliciesByCommand = make(map[CommandID][]signalPolicyRecord)
	model.environmentByCommand = make(map[CommandID][]environmentInputRecord)
	model.profilesByCommand = make(map[CommandID][]grammarProfileRecord)
	model.endpointsByCommand = make(map[CommandID][]signalEndpointRecord)
	model.countersByPolicy = make(map[SignalPolicyID][]signalCounterRecord)
	model.conditionsByCommand = make(map[CommandID][]staticConditionPolicyRecord)
	model.noticeByCommand = make(map[CommandID][]noticeSuppressionRecord)
	model.signalScopesByCommand = make(map[CommandID][]signalPolicyScope)
	model.helpGroupsByCommand = make(map[CommandID][]helpGroupRecord)
	model.domainsByDestination = make(map[ValueID][]int)
	model.replacementsByCommand = make(map[CommandID][]declarationReplacementRecord)

	for _, r := range model.records.commands {
		if p, ok := r.Parent(); ok {
			model.childrenByCommand[p] = append(model.childrenByCommand[p], r.meta.id)
		}
	}
	for _, r := range model.records.destinations {
		model.destinationsByCommand[r.command] = append(model.destinationsByCommand[r.command], r)
	}
	for _, r := range model.records.flags {
		model.flagsByCommand[r.command] = append(model.flagsByCommand[r.command], r)
	}
	for _, r := range model.records.flagSpellings {
		model.flagSpellingsByFlag[r.flag] = append(model.flagSpellingsByFlag[r.flag], r)
	}
	for _, r := range model.records.argumentForms {
		model.formsByCommand[r.command] = append(model.formsByCommand[r.command], r)
	}
	for _, r := range model.records.argumentTerms {
		model.termsByForm[r.form] = append(model.termsByForm[r.form], r)
	}
	for _, r := range model.records.commandEntries {
		model.entriesByParent[r.parent] = append(model.entriesByParent[r.parent], r)
	}
	for _, r := range model.records.actionEntries {
		model.actionsByParent[r.parent] = append(model.actionsByParent[r.parent], r)
	}
	for _, r := range model.records.defaults {
		model.defaultsByParent[r.parent] = append(model.defaultsByParent[r.parent], r)
	}
	for _, r := range model.records.providers {
		model.providersByParent[r.parent] = append(model.providersByParent[r.parent], r)
	}
	for _, r := range model.records.rules {
		model.rulesByCommand[r.command] = append(model.rulesByCommand[r.command], r)
	}
	for _, r := range model.records.advisories {
		model.advisoriesByCommand[r.command] = append(model.advisoriesByCommand[r.command], r)
	}
	for _, r := range model.records.signalPolicies {
		model.signalPoliciesByCommand[r.command] = append(model.signalPoliciesByCommand[r.command], r)
	}
	for _, r := range model.records.environmentInputs {
		model.environmentByCommand[r.command] = append(model.environmentByCommand[r.command], r)
	}
	for _, r := range model.records.grammarProfiles {
		model.profilesByCommand[r.command] = append(model.profilesByCommand[r.command], r)
	}
	for _, r := range model.records.signalEndpoints {
		model.endpointsByCommand[r.command] = append(model.endpointsByCommand[r.command], r)
	}
	for _, r := range model.records.signalCounters {
		model.countersByPolicy[r.policy] = append(model.countersByPolicy[r.policy], r)
	}
	for ordinal, r := range model.records.conditionPolicies {
		if c, ok := model.conditionTargetCommand(r.target); ok {
			model.conditionsByCommand[c] = append(model.conditionsByCommand[c], staticConditionPolicyRecord{
				record:  r,
				ordinal: ordinal,
			})
		}
	}
	for _, r := range model.records.noticeSuppressions {
		model.noticeByCommand[r.command] = append(model.noticeByCommand[r.command], r)
	}
	for _, r := range model.records.signalPolicyScopes {
		model.signalScopesByCommand[r.Command()] = append(model.signalScopesByCommand[r.Command()], r)
	}
	for _, r := range model.records.helpGroups {
		model.helpGroupsByCommand[r.command] = append(model.helpGroupsByCommand[r.command], r)
	}
	for i, domain := range model.records.ValueDomains() {
		for _, use := range domain.Uses() {
			model.domainsByDestination[use.destination] = append(model.domainsByDestination[use.destination], i)
		}
	}
	for _, r := range model.records.replacements {
		seen := make(map[CommandID]struct{}, replacementCommandCapacity)
		for _, id := range []declarationReplacementIdentity{r.source, r.target} {
			if c, ok := model.replacementIdentityCommand(id); ok {
				if _, dup := seen[c]; !dup {
					model.replacementsByCommand[c] = append(model.replacementsByCommand[c], r)
					seen[c] = struct{}{}
				}
			}
		}
	}
}

func (model staticProgram) conditionTargetCommand(target conditionTarget) (CommandID, bool) {
	switch target.Kind() {
	case conditionTargetCommand:
		return target.Command(), target.Command().present()
	case conditionTargetFlag:
		r, ok := model.flags[target.Flag()]
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	case conditionTargetArgumentForm:
		r, ok := model.argumentForms[target.ArgumentForm()]
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	case conditionTargetArgumentTerm:
		t, ok := model.argumentTerms[target.ArgumentTerm()]
		if !ok {
			return CommandID(0), false
		}
		f, ok := model.argumentForms[t.form]
		if !ok {
			return CommandID(0), false
		}
		return f.command, true
	default:
		return CommandID(0), false
	}
}

func (model staticProgram) replacementIdentityCommand(id declarationReplacementIdentity) (CommandID, bool) {
	switch id.Kind() {
	case declarationReplacementKindCommandEntry:
		r, ok := model.commandEntries[id.CommandEntry()]
		if !ok {
			return CommandID(0), false
		}
		return r.parent, true
	case declarationReplacementKindFlag:
		r, ok := model.flags[id.Flag()]
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	case declarationReplacementKindFlagSpelling:
		r, ok := model.flagSpellings[id.FlagSpelling()]
		if !ok {
			return CommandID(0), false
		}
		f, ok := model.flags[r.flag]
		if !ok {
			return CommandID(0), false
		}
		return f.command, true
	case declarationReplacementKindArgumentForm:
		r, ok := model.argumentForms[id.ArgumentForm()]
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	case declarationReplacementKindArgumentTerm:
		r, ok := model.argumentTerms[id.ArgumentTerm()]
		if !ok {
			return CommandID(0), false
		}
		f, ok := model.argumentForms[r.form]
		if !ok {
			return CommandID(0), false
		}
		return f.command, true
	case declarationReplacementKindEnvironmentInput:
		r, ok := model.environmentInputs[id.EnvironmentInput()]
		if !ok {
			return CommandID(0), false
		}
		return r.command, true
	default:
		return CommandID(0), false
	}
}

func cloneSlice[T any](in []T) []T { return append([]T(nil), in...) }

func (model staticProgram) Children(command CommandID) []CommandID {
	return cloneSlice(model.childrenByCommand[command])
}

func (model staticProgram) DestinationsForCommand(command CommandID) []destinationRecord {
	return cloneSlice(model.destinationsByCommand[command])
}

func (model staticProgram) FlagsForCommand(command CommandID) []flagRecord {
	return cloneSlice(model.flagsByCommand[command])
}

func (model staticProgram) FlagSpellingsForFlag(flag FlagID) []flagSpellingRecord {
	return cloneSlice(model.flagSpellingsByFlag[flag])
}

func (model staticProgram) ArgumentFormsForCommand(command CommandID) []argumentFormRecord {
	return cloneSlice(model.formsByCommand[command])
}

func (model staticProgram) ArgumentTermsForForm(form ArgumentFormID) []argumentTermRecord {
	return cloneSlice(model.termsByForm[form])
}

func (model staticProgram) CommandEntriesForParent(command CommandID) []commandEntryRecord {
	return cloneSlice(model.entriesByParent[command])
}

func (model staticProgram) ActionEntriesForParent(command CommandID) []actionEntryRecord {
	return cloneSlice(model.actionsByParent[command])
}

func (model staticProgram) DefaultsForParent(command CommandID) []defaultRecord {
	return cloneSlice(model.defaultsByParent[command])
}

func (model staticProgram) ProvidersForParent(command CommandID) []providerRecord {
	return cloneSlice(model.providersByParent[command])
}

func (model staticProgram) RulesForCommand(command CommandID) []ruleRecord {
	return cloneSlice(model.rulesByCommand[command])
}

func (model staticProgram) AdvisoriesForCommand(command CommandID) []advisoryRecord {
	return cloneSlice(model.advisoriesByCommand[command])
}

func (model staticProgram) SignalPoliciesForCommand(command CommandID) []signalPolicyRecord {
	return cloneSlice(model.signalPoliciesByCommand[command])
}

func (model staticProgram) EnvironmentInputsForCommand(command CommandID) []environmentInputRecord {
	return cloneSlice(model.environmentByCommand[command])
}

func (model staticProgram) GrammarProfilesForCommand(command CommandID) []grammarProfileRecord {
	return cloneSlice(model.profilesByCommand[command])
}

func (model staticProgram) SignalEndpointsForCommand(command CommandID) []signalEndpointRecord {
	return cloneSlice(model.endpointsByCommand[command])
}

func (model staticProgram) SignalCountersForPolicy(policy SignalPolicyID) []signalCounterRecord {
	return cloneSlice(model.countersByPolicy[policy])
}

func (model staticProgram) ConditionPoliciesForCommand(command CommandID) []conditionPolicyRecord {
	indexed := model.conditionsByCommand[command]
	result := make([]conditionPolicyRecord, 0, len(indexed))
	for _, entry := range indexed {
		result = append(result, entry.record)
	}
	return result
}

// ConditionPoliciesForCommandsOrdered returns only the requested command scopes
// in the canonical full-family ordinal established by StaticRecords. The
// ordinal is retained in the immutable scoped index so callers do not need to
// scan unrelated static conditions or infer cross-owner order from declaration
// positions.
func (model staticProgram) ConditionPoliciesForCommandsOrdered(commands []CommandID) []conditionPolicyRecord {
	selected := make(map[CommandID]struct{}, len(commands))
	var indexed []staticConditionPolicyRecord
	for _, command := range commands {
		if _, duplicate := selected[command]; duplicate {
			continue
		}
		selected[command] = struct{}{}
		indexed = append(indexed, model.conditionsByCommand[command]...)
	}
	slices.SortFunc(indexed, func(left, right staticConditionPolicyRecord) int { return cmp.Compare(left.ordinal, right.ordinal) })
	result := make([]conditionPolicyRecord, 0, len(indexed))
	for _, entry := range indexed {
		result = append(result, entry.record)
	}
	return result
}

func (model staticProgram) NoticeSuppressionsForCommand(command CommandID) []noticeSuppressionRecord {
	return cloneSlice(model.noticeByCommand[command])
}

func (model staticProgram) SignalPolicyScopesForCommand(command CommandID) []signalPolicyScope {
	return cloneSlice(model.signalScopesByCommand[command])
}

func (model staticProgram) HelpGroupsForCommand(command CommandID) []helpGroupRecord {
	return cloneSlice(model.helpGroupsByCommand[command])
}

func (model staticProgram) ValueDomainIndexesForDestination(destination ValueID) []int {
	return cloneSlice(model.domainsByDestination[destination])
}

func (model staticProgram) ReplacementsForCommand(command CommandID) []declarationReplacementRecord {
	return cloneSlice(model.replacementsByCommand[command])
}
