package cli

import "errors"

var (
	errDraftClosed        = errors.New("application draft closed")
	errAcceptanceBoundary = errors.New("invalid declaration acceptance boundary")
)

// rejectionError is private declaration-acceptance control state. Public
// diagnostic projection is owned downstream; this type is not a public failure vocabulary.
type rejectionError struct{ rejections []rejection }

func (err rejectionError) Error() string { return "declaration program rejected" }

func (err rejectionError) Rejections() []rejection {
	return append([]rejection(nil), err.rejections...)
}

// publicationRecordCounts captures the accepted-record slice frontiers at the
// start of one operation-local publication transaction.
type publicationRecordCounts struct {
	commands, destinations, flags, flagSpellings, argumentForms, argumentTerms int
	commandEntries, actionEntries, defaults, providers, rules, advisories      int
	signalPolicies, domainEntries, valueDomains, environmentInputs             int
	grammarProfiles, signalEndpoints, signalCounters, conditionPolicies        int
	noticeSuppressions, signalPolicyScopes, helpGroups, replacements           int
}

type publicationGrammarCounts struct {
	commandEntries, actionCommands, actionFlags, flags, argumentForms int
}

type lifecycleAssignment struct {
	family uint8
	id     uint32
}

type declarationCandidate struct {
	owner                stagingOwner
	closed               bool
	ids                  declarationIdentities
	base                 publicationRecordCounts
	baseGrammar          publicationGrammarCounts
	baseGrammarModel     grammarModel
	input                declarationRecordInput
	preflight            []rejection
	grammar              *grammarDraft
	application          applicationMetadata
	hasApplication       bool
	lifecycleAssignments map[lifecycleAssignment]struct{}
	callbacks            *declarationCallbacks
}

type publicationDraft struct {
	declarationCandidate
}

// publicationDelta is the immutable accepted output of one disposable staging
// transaction. It contains only newly accepted declarations and grammar; it is
// never a flattened effective application model.
type publicationDelta struct {
	declarations declarationDelta
	grammar      grammarDelta
}

func (result publicationDelta) Declarations() declarationDelta { return result.declarations }

func (result publicationDelta) Grammar() grammarDelta { return result.grammar }

// publicationPreview is a read-only candidate-local relationship view used
// before final publication. It is intentionally narrower than declarations:
// only destination lookup and grammar path resolution are exposed.
type publicationPreview struct {
	destinations map[ValueID]destinationRecord
	grammar      grammarModel
}

func (preview publicationPreview) Destination(id ValueID) (destinationRecord, bool) {
	record, ok := preview.destinations[id]
	return record, ok
}

func (preview publicationPreview) Grammar() (grammarModel, bool) {
	return preview.grammar, preview.grammar.Valid()
}

// prepareStaticCandidate creates the application-lifetime acceptance candidate
// and the exact static identity scope that will be sealed with it.
func prepareStaticCandidate() (*declarationCandidate, staticIdentityScope) {
	identities := allocateStaticIdentityScope()
	callbacks := declarationCallbacks{
		valueProducers: make(map[valueProducerKey]valueProducer),
		completions: completionCallbacks{
			flags:     make(map[FlagID]completionCallback),
			arguments: make(map[ArgumentTermID]completionCallback),
		},
		commandProviders: make(map[CommandProviderID]commandProviderFunc),
	}
	candidate := &declarationCandidate{
		owner:     allocateStagingOwner(),
		ids:       identities.declarationIdentities,
		callbacks: &callbacks,
	}

	return candidate, identities
}

// preparePublicationDraft creates one disposable operation-local declaration
// staging transaction from the already-selected validation scope. The scope is
// copied as candidate input only; no declarations or StaticRecords is built.
func preparePublicationDraft(input declarationRecordInput, grammar grammarModel, application applicationMetadata, ids operationIDs) *publicationDraft {
	if !grammar.Valid() {
		panic("application: invalid publication draft grammar")
	}
	input = snapshotPublicationInput(input)
	draft := new(publicationDraft)
	draft.declarationCandidate = declarationCandidate{
		owner:            allocateStagingOwner(),
		ids:              ids.declarationIdentities,
		base:             publicationCounts(input),
		baseGrammar:      publicationGrammarBaseCounts(grammar),
		baseGrammarModel: grammar,
		input:            input,
		application:      application,
		hasApplication:   true,
	}
	draft.grammar = grammarDraftFromModel(&draft.declarationCandidate, grammar)
	return draft
}

// Application records the already-validated canonical application-root
// metadata. Public snapshotting and semantic validation are facade/acceptance
// responsibilities; this Draft stores only the accepted immutable value.
func (draft *declarationCandidate) Application(metadata applicationMetadata) {
	draft.mustOpen()
	if draft.hasApplication {
		draft.preflight = append(draft.preflight, rejection{
			Kind: rejectionStructuralInvalidity,
		})
		return
	}
	draft.application = metadata
	draft.hasApplication = true
}

func (draft *declarationCandidate) IntrinsicActions(actions intrinsicActions) {
	draft.mustOpen()
	if !actions.Valid() || draft.input.IntrinsicActions.Valid() {
		draft.preflight = append(draft.preflight, rejection{
			Kind: rejectionStructuralInvalidity,
		})
		return
	}
	draft.input.IntrinsicActions = actions
}

func (draft *declarationCandidate) mustOpen() {
	if draft == nil || draft.closed {
		panic(errDraftClosed)
	}
}

func (draft *declarationCandidate) reject(position declarationPosition) {
	draft.preflight = append(draft.preflight, rejection{
		Kind:  rejectionForeignOrigin,
		Order: uint64(position),
	})
}

func (draft *declarationCandidate) rejectStructural(position declarationPosition) {
	draft.preflight = append(draft.preflight, semanticRejection(position))
}

// acceptStaticDeclarations closes the static acceptance candidate and returns
// immutable application declarations. Operation-local publication uses the same
// candidate mechanics but a distinct transaction and terminal product.
func acceptStaticDeclarations(draft *declarationCandidate, identities staticIdentityScope, decision acceptanceDecision) (staticProgram, error) {
	if draft == nil || draft.closed {
		return staticProgram{}, errDraftClosed
	}
	if decision.Boundary() != acceptanceBoundaryStatic {
		return staticProgram{}, errAcceptanceBoundary
	}
	draft.closed = true
	defer draft.clear()

	rejections := append(decision.Rejections(), draft.preflight...)
	rejections = append(rejections, draft.validateReferences(false)...)
	if len(rejections) == 0 {
		rejections = append(rejections, validateProofInvariants(draft.input)...)
	}
	if len(rejections) != 0 {
		combined := acceptAcceptanceDecision(acceptanceBoundaryStatic, rejections)
		return staticProgram{}, rejectionError{
			rejections: combined.Rejections(),
		}
	}

	records := freezeStaticRecords(draft.input)
	grammar, _, grammarRejections := draft.buildGrammar()
	if len(grammarRejections) != 0 {
		combined := acceptAcceptanceDecision(acceptanceBoundaryStatic, grammarRejections)
		return staticProgram{}, rejectionError{
			rejections: combined.Rejections(),
		}
	}
	if err := validateAcceptedTopology(records, grammar); err != nil {
		return staticProgram{}, err
	}
	baseline := identities.Seal()
	return freezeStaticProgram(records, baseline, grammar, draft.application), nil
}

// preview validates the current operation-local transaction and returns the
// narrow candidate-local relationship view needed for provider target/default
// settlement. It does not close the Draft or publish declaration identity.
func (draft *publicationDraft) preview(decision acceptanceDecision) (publicationPreview, error) {
	if draft == nil || draft.closed {
		return publicationPreview{}, errDraftClosed
	}
	grammar, err := draft.validateCandidate(decision)
	if err != nil {
		return publicationPreview{}, err
	}
	destinations := make(map[ValueID]destinationRecord, len(draft.input.Destinations))
	for _, record := range draft.input.Destinations {
		destinations[record.meta.id] = record
	}
	return publicationPreview{
		destinations: destinations,
		grammar:      grammar,
	}, nil
}

// acceptPublication closes one operation-local acceptance transaction and returns only
// its immutable typed deltas. The caller commits the forked operation identity
// scope atomically with declarations publication after this method succeeds.
func (draft *publicationDraft) acceptPublication(decision acceptanceDecision) (publicationDelta, error) {
	if draft == nil || draft.closed {
		return publicationDelta{}, errDraftClosed
	}
	draft.closed = true
	defer draft.clear()
	grammar, err := draft.validateCandidate(decision)
	if err != nil {
		return publicationDelta{}, err
	}
	if err := validateAcceptedTopology(freezeStaticRecords(draft.input), grammar); err != nil {
		return publicationDelta{}, err
	}
	return publicationDelta{
		declarations: draft.declarationDelta(),
		grammar:      draft.grammarDelta(grammar),
	}, nil
}

func (draft *declarationCandidate) clear() {
	draft.input = declarationRecordInput{}
	draft.owner = stagingOwner{}
	draft.ids = declarationIdentities{}
	draft.base = publicationRecordCounts{}
	draft.baseGrammar = publicationGrammarCounts{}
	draft.preflight = nil
	draft.grammar = nil
	draft.application = applicationMetadata{}
	draft.hasApplication = false
}

// CommandByID imports one already-accepted command identity into this
// Draft as a relationship capability. It never allocates or copies a
// declaration.
func (draft *declarationCandidate) CommandByID(id CommandID) acceptanceCommandRef {
	draft.mustOpen()
	if !id.present() {
		draft.reject(0)
		return issueStagingRef(draft.owner, id)
	}
	for index, record := range draft.input.Commands {
		if record.meta.id == id {
			return issueIndexedStagingRef(draft.owner, id, index)
		}
	}
	draft.reject(0)
	return issueStagingRef(draft.owner, id)
}

// DestinationByID imports one already-accepted destination identity into this
// Draft as a relationship capability.
func (draft *declarationCandidate) DestinationByID(id ValueID) acceptanceDestinationRef {
	draft.mustOpen()
	if !id.present() {
		draft.reject(0)
		return issueStagingRef(draft.owner, id)
	}
	for index, record := range draft.input.Destinations {
		if record.meta.id == id {
			return issueIndexedStagingRef(draft.owner, id, index)
		}
	}
	draft.reject(0)
	return issueStagingRef(draft.owner, id)
}

// ExistingDestinationDefinition returns the accepted definition for an
// already-present destination without creating or mutating a declaration.
func (draft *declarationCandidate) ExistingDestinationDefinition(id ValueID) (destinationDefinition, bool) {
	draft.mustOpen()
	if !id.present() {
		return destinationDefinition{}, false
	}
	for _, record := range draft.input.Destinations {
		if record.meta.id == id {
			return record.Definition()
		}
	}
	return destinationDefinition{}, false
}

func (draft *declarationCandidate) commandRecordForRef(command acceptanceCommandRef) (int, commandRecord, bool) {
	if index, ok := command.RecordIndex(); ok && index < len(draft.input.Commands) {
		record := draft.input.Commands[index]
		if record.meta.id == command.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.Commands {
		if record.meta.id == command.ID() {
			return index, record, true
		}
	}
	return 0, commandRecord{}, false
}

func (draft *declarationCandidate) destinationRecordForRef(destination acceptanceDestinationRef) (int, destinationRecord, bool) {
	if index, ok := destination.RecordIndex(); ok && index < len(draft.input.Destinations) {
		record := draft.input.Destinations[index]
		if record.meta.id == destination.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.Destinations {
		if record.meta.id == destination.ID() {
			return index, record, true
		}
	}
	return 0, destinationRecord{}, false
}

func (draft *declarationCandidate) flagRecordForRef(flag acceptanceFlagRef) (int, flagRecord, bool) {
	if index, ok := flag.RecordIndex(); ok && index < len(draft.input.Flags) {
		record := draft.input.Flags[index]
		if record.meta.id == flag.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.Flags {
		if record.meta.id == flag.ID() {
			return index, record, true
		}
	}
	return 0, flagRecord{}, false
}

func (draft *declarationCandidate) flagSpellingRecordForRef(spelling acceptanceFlagSpellingRef) (int, flagSpellingRecord, bool) {
	if index, ok := spelling.RecordIndex(); ok && index < len(draft.input.FlagSpellings) {
		record := draft.input.FlagSpellings[index]
		if record.meta.id == spelling.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.FlagSpellings {
		if record.meta.id == spelling.ID() {
			return index, record, true
		}
	}
	return 0, flagSpellingRecord{}, false
}

func (draft *declarationCandidate) argumentFormRecordForRef(form argumentFormRef) (int, argumentFormRecord, bool) {
	if index, ok := form.RecordIndex(); ok && index < len(draft.input.ArgumentForms) {
		record := draft.input.ArgumentForms[index]
		if record.meta.id == form.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.ArgumentForms {
		if record.meta.id == form.ID() {
			return index, record, true
		}
	}
	return 0, argumentFormRecord{}, false
}

func (draft *declarationCandidate) argumentTermRecordForRef(term argumentTermRef) (int, argumentTermRecord, bool) {
	if index, ok := term.RecordIndex(); ok && index < len(draft.input.ArgumentTerms) {
		record := draft.input.ArgumentTerms[index]
		if record.meta.id == term.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.ArgumentTerms {
		if record.meta.id == term.ID() {
			return index, record, true
		}
	}
	return 0, argumentTermRecord{}, false
}

func (draft *declarationCandidate) commandEntryRecordForRef(entry commandEntryRef) (int, commandEntryRecord, bool) {
	if index, ok := entry.RecordIndex(); ok && index < len(draft.input.CommandEntries) {
		record := draft.input.CommandEntries[index]
		if record.meta.id == entry.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.CommandEntries {
		if record.meta.id == entry.ID() {
			return index, record, true
		}
	}
	return 0, commandEntryRecord{}, false
}

func (draft *declarationCandidate) actionEntryRecordForRef(entry actionEntryRef) (int, actionEntryRecord, bool) {
	if index, ok := entry.RecordIndex(); ok && index < len(draft.input.ActionEntries) {
		record := draft.input.ActionEntries[index]
		if record.meta.id == entry.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.ActionEntries {
		if record.meta.id == entry.ID() {
			return index, record, true
		}
	}
	return 0, actionEntryRecord{}, false
}

func (draft *declarationCandidate) defaultRecordForRef(defaultRef defaultRef) (int, defaultRecord, bool) {
	if index, ok := defaultRef.RecordIndex(); ok && index < len(draft.input.Defaults) {
		record := draft.input.Defaults[index]
		if record.meta.id == defaultRef.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.Defaults {
		if record.meta.id == defaultRef.ID() {
			return index, record, true
		}
	}
	return 0, defaultRecord{}, false
}

func (draft *declarationCandidate) providerRecordForRef(provider providerRef) (int, providerRecord, bool) {
	if index, ok := provider.RecordIndex(); ok && index < len(draft.input.Providers) {
		record := draft.input.Providers[index]
		if record.meta.id == provider.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.Providers {
		if record.meta.id == provider.ID() {
			return index, record, true
		}
	}
	return 0, providerRecord{}, false
}

func (draft *declarationCandidate) ruleRecordForRef(rule ruleRef) (int, ruleRecord, bool) {
	if index, ok := rule.RecordIndex(); ok && index < len(draft.input.Rules) {
		record := draft.input.Rules[index]
		if record.meta.id == rule.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.Rules {
		if record.meta.id == rule.ID() {
			return index, record, true
		}
	}
	return 0, ruleRecord{}, false
}

func (draft *declarationCandidate) advisoryRecordForRef(advisory advisoryRef) (int, advisoryRecord, bool) {
	if index, ok := advisory.RecordIndex(); ok && index < len(draft.input.Advisories) {
		record := draft.input.Advisories[index]
		if record.meta.id == advisory.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.Advisories {
		if record.meta.id == advisory.ID() {
			return index, record, true
		}
	}
	return 0, advisoryRecord{}, false
}

func (draft *declarationCandidate) signalPolicyRecordForRef(policy signalPolicyRef) (int, signalPolicyRecord, bool) {
	if index, ok := policy.RecordIndex(); ok && index < len(draft.input.SignalPolicies) {
		record := draft.input.SignalPolicies[index]
		if record.meta.id == policy.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.SignalPolicies {
		if record.meta.id == policy.ID() {
			return index, record, true
		}
	}
	return 0, signalPolicyRecord{}, false
}

func (draft *declarationCandidate) domainEntryRecordForRef(entry domainEntryRef) (int, domainEntryRecord, bool) {
	if index, ok := entry.RecordIndex(); ok && index < len(draft.input.DomainEntries) {
		record := draft.input.DomainEntries[index]
		if record.meta.id == entry.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.DomainEntries {
		if record.meta.id == entry.ID() {
			return index, record, true
		}
	}
	return 0, domainEntryRecord{}, false
}

func (draft *declarationCandidate) environmentInputRecordForRef(input environmentInputRef) (int, environmentInputRecord, bool) {
	if index, ok := input.RecordIndex(); ok && index < len(draft.input.EnvironmentInputs) {
		record := draft.input.EnvironmentInputs[index]
		if record.meta.id == input.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.EnvironmentInputs {
		if record.meta.id == input.ID() {
			return index, record, true
		}
	}
	return 0, environmentInputRecord{}, false
}

func (draft *declarationCandidate) grammarProfileRecordForRef(profile grammarProfileRef) (int, grammarProfileRecord, bool) {
	if index, ok := profile.RecordIndex(); ok && index < len(draft.input.GrammarProfiles) {
		record := draft.input.GrammarProfiles[index]
		if record.meta.id == profile.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.GrammarProfiles {
		if record.meta.id == profile.ID() {
			return index, record, true
		}
	}
	return 0, grammarProfileRecord{}, false
}

func (draft *declarationCandidate) signalEndpointRecordForRef(endpoint signalEndpointRef) (int, signalEndpointRecord, bool) {
	if index, ok := endpoint.RecordIndex(); ok && index < len(draft.input.SignalEndpoints) {
		record := draft.input.SignalEndpoints[index]
		if record.meta.id == endpoint.ID() {
			return index, record, true
		}
	}
	for index, record := range draft.input.SignalEndpoints {
		if record.meta.id == endpoint.ID() {
			return index, record, true
		}
	}
	return 0, signalEndpointRecord{}, false
}

func (draft *declarationCandidate) existingDestinationDefinitionForRef(destination acceptanceDestinationRef) (destinationDefinition, bool) {
	_, record, ok := draft.destinationRecordForRef(destination)
	if !ok {
		return destinationDefinition{}, false
	}
	return record.Definition()
}

func (draft *declarationCandidate) RootCommand(position declarationPosition, configuration semanticCommandConfiguration) acceptanceCommandRef {
	draft.mustOpen()
	id := draft.ids.AllocateCommandID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Commands))
	if !configuration.Valid() {
		draft.reject(position)
		return result
	}
	draft.input.Commands = append(draft.input.Commands, acceptRootCommandRecord(id, position, configuration))
	return result
}

func (draft *declarationCandidate) Command(parent acceptanceCommandRef, position declarationPosition, configuration semanticCommandConfiguration) acceptanceCommandRef {
	draft.mustOpen()
	id := draft.ids.AllocateCommandID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Commands))
	if !parent.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.Commands = append(
		draft.input.Commands,
		acceptCommandRecord(id, parent.ID(), position, configuration),
	)
	return result
}

func (draft *declarationCandidate) Destination(command acceptanceCommandRef, position declarationPosition, definition destinationDefinition, binding configurationBinding) acceptanceDestinationRef {
	draft.mustOpen()
	id := draft.ids.AllocateDestinationID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Destinations))
	if !command.OwnedBy(draft.owner) || !definition.Valid() {
		draft.reject(position)
		return result
	}
	draft.input.Destinations = append(
		draft.input.Destinations,
		acceptDestinationRecord(id, command.ID(), position, definition, binding),
	)
	return result
}

func (draft *declarationCandidate) DestinationProducer(destination acceptanceDestinationRef, producer valueProducerRecord, callback valueProducer, position declarationPosition) {
	draft.mustOpen()
	if !destination.OwnedBy(draft.owner) || !producer.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.destinationRecordForRef(destination)
	if !ok {
		draft.reject(position)
		return
	}
	if draft.callbacks != nil && callback != nil {
		producerIndex := len(record.producers)
		key := valueProducerKey{
			destination: destination.ID(),
			index:       producerIndex,
		}
		if _, exists := draft.callbacks.valueProducers[key]; exists {
			panic("cli: duplicate static value producer callback custody")
		}
		draft.callbacks.valueProducers[key] = callback
	}
	record.producers = append(append([]valueProducerRecord(nil), record.producers...), producer)
	draft.input.Destinations[index] = record
}

func (draft *declarationCandidate) DestinationDefinition(destination acceptanceDestinationRef, definition destinationDefinition, position declarationPosition) {
	draft.mustOpen()
	if !destination.OwnedBy(draft.owner) || !definition.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.destinationRecordForRef(destination)
	if !ok {
		draft.reject(position)
		return
	}
	record.definition = definition
	draft.input.Destinations[index] = record
}

func (draft *declarationCandidate) DestinationResolution(destination acceptanceDestinationRef, addition semanticResolutionPolicy, position declarationPosition) {
	draft.mustOpen()
	if !destination.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.destinationRecordForRef(destination)
	if !ok {
		draft.reject(position)
		return
	}
	definition, ok := record.Definition()
	if !ok {
		draft.reject(position)
		return
	}
	merged, ok := mergeResolutionPolicies(definition.Resolution(), addition)
	if !ok {
		rejection := rejection{
			Kind:  rejectionStructuralInvalidity,
			Order: uint64(position),
		}
		draft.preflight = append(draft.preflight, rejection)
		return
	}
	definition.resolution = merged
	record.definition = definition
	draft.input.Destinations[index] = record
}

func (draft *declarationCandidate) Flag(command acceptanceCommandRef, destination acceptanceDestinationRef, position declarationPosition) acceptanceFlagRef {
	draft.mustOpen()
	id := draft.ids.AllocateFlagID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Flags))
	if !command.OwnedBy(draft.owner) || !destination.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.Flags = append(draft.input.Flags, acceptFlagRecord(id, command.ID(), destination.ID(), position))
	return result
}

func (draft *declarationCandidate) CommandAccess(command acceptanceCommandRef, access executionAccess, position declarationPosition) {
	draft.mustOpen()
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandRecordForRef(command)
	if !ok {
		draft.reject(position)
		return
	}
	record.access = access
	draft.input.Commands[index] = record
}

func (draft *declarationCandidate) CommandExecution(command acceptanceCommandRef, execution commandExecution, position declarationPosition) {
	draft.mustOpen()
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandRecordForRef(command)
	if !ok {
		draft.reject(position)
		return
	}
	record.execution = execution
	draft.input.Commands[index] = record
}

func (draft *declarationCandidate) CommandPresentation(command acceptanceCommandRef, presentation presentation, position declarationPosition) {
	draft.mustOpen()
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandRecordForRef(command)
	if !ok {
		draft.reject(position)
		return
	}
	record.presentation = presentation
	draft.input.Commands[index] = record
}

func (draft *declarationCandidate) CommandEntryPresentation(entry commandEntryRef, presentation presentation, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	record.presentation = presentation
	draft.input.CommandEntries[index] = record
}

func (draft *declarationCandidate) ActionEntryPresentation(entry actionEntryRef, presentation presentation, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.actionEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	record.presentation = presentation
	draft.input.ActionEntries[index] = record
}

func (draft *declarationCandidate) DestinationSensitive(destination acceptanceDestinationRef, sensitive bool, position declarationPosition) {
	draft.mustOpen()
	if !destination.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.destinationRecordForRef(destination)
	if !ok {
		draft.reject(position)
		return
	}
	record.sensitive = sensitive
	draft.input.Destinations[index] = record
}

func (draft *declarationCandidate) FlagPresentation(flag acceptanceFlagRef, presentation presentation, valueName string, required, localeSelector bool, position declarationPosition) {
	draft.mustOpen()
	if !flag.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.flagRecordForRef(flag)
	if !ok {
		draft.reject(position)
		return
	}
	record.presentation = presentation
	record.valueName = valueName
	record.required = required
	record.localeSelector = localeSelector
	draft.input.Flags[index] = record
}

func (draft *declarationCandidate) FlagCompletion(flag acceptanceFlagRef, completion completionDeclaration, callback completionCallback, position declarationPosition) {
	draft.mustOpen()
	if !flag.OwnedBy(draft.owner) || !completion.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.flagRecordForRef(flag)
	if !ok {
		draft.reject(position)
		return
	}
	if _, exists := record.Completion(); exists {
		draft.reject(position)
		return
	}
	if draft.callbacks != nil && callback.present() {
		if _, exists := draft.callbacks.completions.flags[flag.ID()]; exists {
			panic("cli: duplicate static flag completion callback custody")
		}
		draft.callbacks.completions.flags[flag.ID()] = callback
	}
	record.completion = completion
	draft.input.Flags[index] = record
}

func (draft *declarationCandidate) ArgumentTermCompletion(term argumentTermRef, completion completionDeclaration, callback completionCallback, position declarationPosition) {
	draft.mustOpen()
	if !term.OwnedBy(draft.owner) || !completion.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.argumentTermRecordForRef(term)
	if !ok {
		draft.reject(position)
		return
	}
	if _, exists := record.Completion(); exists {
		draft.reject(position)
		return
	}
	if draft.callbacks != nil && callback.present() {
		if _, exists := draft.callbacks.completions.arguments[term.ID()]; exists {
			panic("cli: duplicate static argument completion callback custody")
		}
		draft.callbacks.completions.arguments[term.ID()] = callback
	}
	record.completion = completion
	draft.input.ArgumentTerms[index] = record
}

func (draft *declarationCandidate) ArgumentTermPresentation(term argumentTermRef, presentation presentation, name string, sourceSensitive bool, position declarationPosition) {
	draft.mustOpen()
	if !term.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.argumentTermRecordForRef(term)
	if !ok {
		draft.reject(position)
		return
	}
	record.presentation = presentation
	record.name = name
	record.sourceSensitive = sourceSensitive
	draft.input.ArgumentTerms[index] = record
}

func (draft *declarationCandidate) FlagLifecycle(flag acceptanceFlagRef, lifecycle ReleaseLifecycle, position declarationPosition) {
	draft.mustOpen()
	if !flag.OwnedBy(draft.owner) || !draft.validLifecycle(lifecycle) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.flagRecordForRef(flag)
	if !ok {
		draft.reject(position)
		return
	}
	record.lifecycle = lifecycle
	draft.input.Flags[index] = record
}

func (draft *declarationCandidate) FlagSpelling(flag acceptanceFlagRef, position declarationPosition) acceptanceFlagSpellingRef {
	draft.mustOpen()
	id := draft.ids.AllocateFlagSpellingID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.FlagSpellings))
	if !flag.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.FlagSpellings = append(draft.input.FlagSpellings, acceptFlagSpellingRecord(id, flag.ID(), position))
	return result
}

func (draft *declarationCandidate) FlagSpellingHidden(spelling acceptanceFlagSpellingRef, position declarationPosition) {
	draft.mustOpen()
	if !spelling.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.flagSpellingRecordForRef(spelling)
	if !ok {
		draft.reject(position)
		return
	}
	record.hidden = true
	draft.input.FlagSpellings[index] = record
}

func (draft *declarationCandidate) FlagSpellingLifecycle(spelling acceptanceFlagSpellingRef, lifecycle ReleaseLifecycle, position declarationPosition) {
	draft.mustOpen()
	if !spelling.OwnedBy(draft.owner) || !draft.validLifecycle(lifecycle) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.flagSpellingRecordForRef(spelling)
	if !ok {
		draft.reject(position)
		return
	}
	record.lifecycle = lifecycle
	draft.input.FlagSpellings[index] = record
}

func (draft *declarationCandidate) ArgumentForm(command acceptanceCommandRef, position declarationPosition) argumentFormRef {
	draft.mustOpen()
	id := draft.ids.AllocateArgumentFormID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.ArgumentForms))
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.ArgumentForms = append(draft.input.ArgumentForms, acceptArgumentFormRecord(id, command.ID(), position))
	return result
}

func (draft *declarationCandidate) ArgumentFormSelection(form argumentFormRef, destination acceptanceDestinationRef, value any, position declarationPosition) {
	draft.mustOpen()
	if !form.OwnedBy(draft.owner) || !destination.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	formIndex, formRecord, ok := draft.argumentFormRecordForRef(form)
	if !ok {
		draft.reject(position)
		return
	}
	_, destinationRecord, ok := draft.destinationRecordForRef(destination)
	if !ok || destinationRecord.command != formRecord.command {
		draft.reject(position)
		return
	}
	for _, selection := range formRecord.Selections() {
		if selection.destination != destination.ID() {
			continue
		}
		rejection := rejection{
			Kind:  rejectionStructuralInvalidity,
			Order: uint64(position),
		}
		draft.preflight = append(draft.preflight, rejection)
		return
	}
	formRecord.selections = append(
		append([]argumentFormSelection(nil), formRecord.selections...),
		acceptArgumentFormSelection(destination.ID(), value),
	)
	draft.input.ArgumentForms[formIndex] = formRecord
}

func (draft *declarationCandidate) ArgumentFormLifecycle(form argumentFormRef, lifecycle ReleaseLifecycle, position declarationPosition) {
	draft.mustOpen()
	if !form.OwnedBy(draft.owner) || !draft.validLifecycle(lifecycle) {
		draft.reject(position)
		return
	}
	if !draft.claimLifecycle(lifecycleFamilyArgumentForm, form.ID().uint32(), position) {
		return
	}
	index, record, ok := draft.argumentFormRecordForRef(form)
	if !ok {
		draft.reject(position)
		return
	}
	record.lifecycle = lifecycle
	draft.input.ArgumentForms[index] = record
}

func (draft *declarationCandidate) ArgumentTerm(form argumentFormRef, position declarationPosition) argumentTermRef {
	draft.mustOpen()
	id := draft.ids.AllocateArgumentTermID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.ArgumentTerms))
	if !form.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.ArgumentTerms = append(draft.input.ArgumentTerms, acceptArgumentTermRecord(id, form.ID(), position))
	return result
}

const (
	lifecycleFamilyArgumentForm uint8 = iota + 1
	lifecycleFamilyArgumentTerm
	lifecycleFamilyEnvironmentInput
)

func (draft *declarationCandidate) ArgumentTermLifecycle(term argumentTermRef, lifecycle ReleaseLifecycle, position declarationPosition) {
	draft.mustOpen()
	if !term.OwnedBy(draft.owner) || !draft.validLifecycle(lifecycle) {
		draft.reject(position)
		return
	}
	if !draft.claimLifecycle(lifecycleFamilyArgumentTerm, term.ID().uint32(), position) {
		return
	}
	index, record, ok := draft.argumentTermRecordForRef(term)
	if !ok {
		draft.reject(position)
		return
	}
	record.lifecycle = lifecycle
	draft.input.ArgumentTerms[index] = record
}

func (draft *declarationCandidate) ArgumentValueTerm(form argumentFormRef, destination acceptanceDestinationRef, position declarationPosition) argumentTermRef {
	draft.mustOpen()
	id := draft.ids.AllocateArgumentTermID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.ArgumentTerms))
	if !form.OwnedBy(draft.owner) || !destination.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.ArgumentTerms = append(
		draft.input.ArgumentTerms,
		acceptArgumentValueTermRecord(id, form.ID(), destination.ID(), position),
	)
	return result
}

func (draft *declarationCandidate) CommandEntry(parent, target acceptanceCommandRef, position declarationPosition) commandEntryRef {
	draft.mustOpen()
	id := draft.ids.AllocateCommandEntryID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.CommandEntries))
	if !parent.OwnedBy(draft.owner) || !target.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.CommandEntries = append(
		draft.input.CommandEntries,
		acceptCommandEntryRecord(id, parent.ID(), target.ID(), position),
	)
	return result
}

func (draft *declarationCandidate) CommandEntryShortcut(entry commandEntryRef, definition shortcutDefinition, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	shortcut := acceptShortcutDefinition(
		definition.Args(),
		definition.AppendArgs(),
		definition.Defaults(),
		definition.Source(),
	)
	record.shortcut = &shortcut
	draft.input.CommandEntries[index] = record
}

func (draft *declarationCandidate) CommandEntryReplacement(entry, replacement commandEntryRef, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) || !replacement.OwnedBy(draft.owner) || entry.ID() == replacement.ID() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	record.replacement = replacement.ID()
	draft.input.CommandEntries[index] = record
}

func (draft *declarationCandidate) CommandEntryLifecycle(entry commandEntryRef, lifecycle ReleaseLifecycle, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) || !draft.validLifecycle(lifecycle) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.commandEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	record.lifecycle = lifecycle
	draft.input.CommandEntries[index] = record
}

func (draft *declarationCandidate) ActionEntry(parent acceptanceCommandRef, position declarationPosition) actionEntryRef {
	draft.mustOpen()
	id := draft.ids.AllocateActionEntryID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.ActionEntries))
	if !parent.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.ActionEntries = append(draft.input.ActionEntries, acceptActionEntryRecord(id, parent.ID(), position))
	return result
}

func (draft *declarationCandidate) ActionEntryOrigin(entry actionEntryRef, origin DeclarationOrigin, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) || !origin.valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.actionEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	record.origin = origin
	draft.input.ActionEntries[index] = record
}

func (draft *declarationCandidate) ActionEntryLifecycle(entry actionEntryRef, lifecycle ReleaseLifecycle, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) || !draft.validLifecycle(lifecycle) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.actionEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	record.lifecycle = lifecycle
	draft.input.ActionEntries[index] = record
}

func (draft *declarationCandidate) ActionEntryReplacement(entry, replacement actionEntryRef, position declarationPosition) {
	draft.mustOpen()
	if !entry.OwnedBy(draft.owner) || !replacement.OwnedBy(draft.owner) || entry.ID() == replacement.ID() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.actionEntryRecordForRef(entry)
	if !ok {
		draft.reject(position)
		return
	}
	record.replacement = replacement.ID()
	draft.input.ActionEntries[index] = record
}

func (draft *declarationCandidate) Default(parent, target acceptanceCommandRef, position declarationPosition) defaultRef {
	draft.mustOpen()
	id := draft.ids.AllocateDefaultID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Defaults))
	if !parent.OwnedBy(draft.owner) || !target.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.Defaults = append(draft.input.Defaults, acceptDefaultRecord(id, parent.ID(), target.ID(), position))
	return result
}

func (draft *declarationCandidate) DefaultCondition(defaultRef defaultRef, condition condition, position declarationPosition) {
	draft.mustOpen()
	if !defaultRef.OwnedBy(draft.owner) || !condition.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.defaultRecordForRef(defaultRef)
	if !ok {
		draft.reject(position)
		return
	}
	record.condition = condition
	draft.input.Defaults[index] = record
}

func (draft *declarationCandidate) CommandCondition(command acceptanceCommandRef, effect ConditionEffect, condition condition, position declarationPosition) {
	draft.conditionPolicy(
		command.OwnedBy(draft.owner),
		commandConditionTarget(command.ID()),
		effect,
		condition,
		position,
	)
}

func (draft *declarationCandidate) FlagCondition(flag acceptanceFlagRef, effect ConditionEffect, condition condition, position declarationPosition) {
	draft.conditionPolicy(flag.OwnedBy(draft.owner), flagConditionTarget(flag.ID()), effect, condition, position)
}

func (draft *declarationCandidate) ArgumentFormCondition(form argumentFormRef, effect ConditionEffect, condition condition, position declarationPosition) {
	draft.conditionPolicy(
		form.OwnedBy(draft.owner),
		argumentFormConditionTarget(form.ID()),
		effect,
		condition,
		position,
	)
}

func (draft *declarationCandidate) ArgumentTermCondition(term argumentTermRef, effect ConditionEffect, condition condition, position declarationPosition) {
	draft.conditionPolicy(
		term.OwnedBy(draft.owner),
		argumentTermConditionTarget(term.ID()),
		effect,
		condition,
		position,
	)
}

func (draft *declarationCandidate) conditionPolicy(owned bool, target conditionTarget, effect ConditionEffect, condition condition, position declarationPosition) {
	draft.mustOpen()
	if !owned || !target.Valid() || !condition.Valid() ||
		(effect != ConditionEffectAvailability && effect != ConditionEffectVisibility) {
		draft.reject(position)
		return
	}
	draft.input.ConditionPolicies = append(
		draft.input.ConditionPolicies,
		acceptConditionPolicyRecord(target, effect, condition, position),
	)
}

func (draft *declarationCandidate) ConditionPolicyProfile(index int, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if !profile.OwnedBy(draft.owner) || index < 0 || index >= len(draft.input.ConditionPolicies) {
		draft.reject(position)
		return
	}
	record := draft.input.ConditionPolicies[index]
	if !draft.conditionTargetBelongsToCommand(record.target, draft.grammarProfileCommand(profile)) {
		draft.reject(position)
		return
	}
	record.profile = profile.ID()
	draft.input.ConditionPolicies[index] = record
}

func (draft *declarationCandidate) conditionTargetBelongsToCommand(target conditionTarget, command CommandID) bool {
	switch target.Kind() {
	case conditionTargetCommand:
		return target.Command() == command
	case conditionTargetFlag:
		for _, record := range draft.input.Flags {
			if record.meta.id == target.Flag() {
				return record.command == command
			}
		}
	case conditionTargetArgumentForm:
		for _, record := range draft.input.ArgumentForms {
			if record.meta.id == target.ArgumentForm() {
				return record.command == command
			}
		}
	case conditionTargetArgumentTerm:
		for _, term := range draft.input.ArgumentTerms {
			if term.meta.id == target.ArgumentTerm() {
				for _, form := range draft.input.ArgumentForms {
					if form.meta.id == term.form {
						return form.command == command
					}
				}
			}
		}
	case conditionTargetUnknown:
	}
	return false
}

func (draft *declarationCandidate) Provider(parent acceptanceCommandRef, position declarationPosition) providerRef {
	draft.mustOpen()
	id := draft.ids.AllocateProviderID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Providers))
	if !parent.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.Providers = append(draft.input.Providers, acceptProviderRecord(id, parent.ID(), position))
	return result
}

func (draft *declarationCandidate) ProviderDefinition(provider providerRef, definition providerDefinition, callback commandProviderFunc, position declarationPosition) {
	draft.mustOpen()
	if !provider.OwnedBy(draft.owner) || !definition.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.providerRecordForRef(provider)
	if !ok {
		draft.reject(position)
		return
	}
	if draft.callbacks != nil && callback != nil {
		if _, exists := draft.callbacks.commandProviders[provider.ID()]; exists {
			panic("cli: duplicate static command provider callback custody")
		}
		draft.callbacks.commandProviders[provider.ID()] = callback
	}
	record.definition = definition
	draft.input.Providers[index] = record
}

func (draft *declarationCandidate) Rule(command acceptanceCommandRef, position declarationPosition) ruleRef {
	draft.mustOpen()
	id := draft.ids.AllocateRuleID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Rules))
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.Rules = append(draft.input.Rules, acceptRuleRecord(id, command.ID(), position))
	return result
}

func (draft *declarationCandidate) RuleDefinition(rule ruleRef, definition ruleDefinition, position declarationPosition) {
	draft.mustOpen()
	if !rule.OwnedBy(draft.owner) || !definition.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.ruleRecordForRef(rule)
	if !ok {
		draft.reject(position)
		return
	}
	record.definition = definition
	draft.input.Rules[index] = record
}

func (draft *declarationCandidate) AdvisoryDefinition(advisory advisoryRef, definition advisoryDefinition, position declarationPosition) {
	draft.mustOpen()
	if !advisory.OwnedBy(draft.owner) || !definition.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.advisoryRecordForRef(advisory)
	if !ok {
		draft.reject(position)
		return
	}
	record.definition = definition
	draft.input.Advisories[index] = record
}

func (draft *declarationCandidate) NoticeSuppression(command acceptanceCommandRef, match noticeMatch, condition condition, position declarationPosition) int {
	draft.mustOpen()
	if !command.OwnedBy(draft.owner) || !match.Valid() || !condition.Valid() {
		draft.reject(position)
		return -1
	}
	draft.input.NoticeSuppressions = append(
		draft.input.NoticeSuppressions,
		acceptNoticeSuppressionRecord(command.ID(), position, match, condition),
	)
	return len(draft.input.NoticeSuppressions) - 1
}

func (draft *declarationCandidate) NoticeSuppressionProfile(index int, profile grammarProfileRef, position declarationPosition) {
	draft.mustOpen()
	if index < 0 || index >= len(draft.input.NoticeSuppressions) || !profile.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	record := draft.input.NoticeSuppressions[index]
	if record.command != draft.grammarProfileCommand(profile) {
		draft.reject(position)
		return
	}
	record.profile = profile.ID()
	draft.input.NoticeSuppressions[index] = record
}

func (draft *declarationCandidate) Advisory(command acceptanceCommandRef, position declarationPosition) advisoryRef {
	draft.mustOpen()
	id := draft.ids.AllocateAdvisoryID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.Advisories))
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.Advisories = append(draft.input.Advisories, acceptAdvisoryRecord(id, command.ID(), position))
	return result
}

func (draft *declarationCandidate) SignalPolicy(command acceptanceCommandRef, position declarationPosition) signalPolicyRef {
	draft.mustOpen()
	id := draft.ids.AllocateSignalPolicyID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.SignalPolicies))
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.SignalPolicies = append(draft.input.SignalPolicies, acceptSignalPolicyRecord(id, command.ID(), position))
	return result
}

func (draft *declarationCandidate) SignalPolicyDefinition(policy signalPolicyRef, definition signalPolicyDefinition, position declarationPosition) {
	draft.mustOpen()
	if !policy.OwnedBy(draft.owner) || !definition.Valid() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.signalPolicyRecordForRef(policy)
	if !ok {
		draft.reject(position)
		return
	}
	if _, exists := record.Definition(); exists {
		rejection := rejection{
			Kind:  rejectionStructuralInvalidity,
			Order: uint64(position),
		}
		draft.preflight = append(draft.preflight, rejection)
		return
	}
	record.definition = definition
	draft.input.SignalPolicies[index] = record
}

func (draft *declarationCandidate) HelpGroup(record helpGroupRecord) {
	draft.mustOpen()
	if !record.Valid() {
		draft.reject(record.position)
		return
	}
	draft.input.HelpGroups = append(draft.input.HelpGroups, record)
}

func (draft *declarationCandidate) SignalPolicyScope(command acceptanceCommandRef, profile grammarProfileRef, clearInherited bool, position declarationPosition) {
	draft.mustOpen()
	if !command.OwnedBy(draft.owner) || (profile.ID().present() && !profile.OwnedBy(draft.owner)) {
		draft.reject(position)
		return
	}
	profileID := GrammarProfileID(0)
	if profile.ID().present() {
		profileID = profile.ID()
	}
	draft.input.SignalPolicyScopes = append(
		draft.input.SignalPolicyScopes,
		acceptSignalPolicyScope(command.ID(), profileID, clearInherited, position),
	)
}

func (draft *declarationCandidate) DomainEntry(position declarationPosition) domainEntryRef {
	draft.mustOpen()
	id := draft.ids.AllocateDomainEntryID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.DomainEntries))
	draft.input.DomainEntries = append(draft.input.DomainEntries, acceptDomainEntryRecord(id, position))
	return result
}

func (draft *declarationCandidate) DomainValue(position declarationPosition, name string, value any, description string, hidden bool, lifecycle ReleaseLifecycle) domainEntryRef {
	draft.mustOpen()
	id := draft.ids.AllocateDomainEntryID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.DomainEntries))
	input2 := domainValueInput{
		ID:          id,
		Position:    position,
		Name:        name,
		Value:       value,
		Description: description,
		Hidden:      hidden,
		Lifecycle:   lifecycle,
	}
	draft.input.DomainEntries = append(
		draft.input.DomainEntries,
		acceptDomainValueRecord(input2),
	)
	return result
}

func (draft *declarationCandidate) DomainAlias(position declarationPosition, name string, canonical domainEntryRef, hidden bool, lifecycle ReleaseLifecycle) domainEntryRef {
	draft.mustOpen()
	id := draft.ids.AllocateDomainEntryID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.DomainEntries))
	if !canonical.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	input2 := domainAliasInput{
		ID:        id,
		Position:  position,
		Name:      name,
		Canonical: canonical.ID(),
		Hidden:    hidden,
		Lifecycle: lifecycle,
	}
	draft.input.DomainEntries = append(
		draft.input.DomainEntries,
		acceptDomainAliasRecord(input2),
	)
	return result
}

func (draft *declarationCandidate) DomainReplacement(source, target domainEntryRef, position declarationPosition) {
	draft.mustOpen()
	if !source.OwnedBy(draft.owner) || !target.OwnedBy(draft.owner) || source.ID() == target.ID() {
		draft.reject(position)
		return
	}
	index, record, ok := draft.domainEntryRecordForRef(source)
	if !ok {
		draft.reject(position)
		return
	}
	record.replacement = target.ID()
	draft.input.DomainEntries[index] = record
}

func (draft *declarationCandidate) ValueDomain(entries []domainEntryRef) valueDomainRef {
	draft.mustOpen()
	ids := make([]ValueDomainEntryID, len(entries))
	for i, entry := range entries {
		if !entry.OwnedBy(draft.owner) {
			draft.reject(declarationPosition(i + 1))
			return issueValueDomainRef(draft.owner, -1)
		}
		ids[i] = entry.ID()
	}
	index := len(draft.input.ValueDomains)
	draft.input.ValueDomains = append(draft.input.ValueDomains, acceptValueDomainRecord(ids, nil))
	if draft.callbacks != nil {
		draft.callbacks.valueDomains = append(draft.callbacks.valueDomains, nil)
	}
	return issueValueDomainRef(draft.owner, index)
}

func (draft *declarationCandidate) ValueDomainByIndex(index int) valueDomainRef {
	draft.mustOpen()
	if index < 0 || index >= len(draft.input.ValueDomains) {
		draft.reject(0)
		return issueValueDomainRef(draft.owner, -1)
	}
	return issueValueDomainRef(draft.owner, index)
}

func (draft *declarationCandidate) DynamicValueDomain(definition dynamicDomainDefinition, callback dynamicDomainEvaluator) valueDomainRef {
	draft.mustOpen()
	if !definition.Valid() {
		draft.reject(0)
		return issueValueDomainRef(draft.owner, -1)
	}
	if draft.callbacks != nil {
		draft.callbacks.valueDomains = append(draft.callbacks.valueDomains, callback)
	}
	index := len(draft.input.ValueDomains)
	draft.input.ValueDomains = append(draft.input.ValueDomains, acceptDynamicValueDomainRecord(definition, nil))
	return issueValueDomainRef(draft.owner, index)
}

func (draft *declarationCandidate) ValueDomainEntries(domain valueDomainRef, entries []domainEntryRef, position declarationPosition) {
	draft.mustOpen()
	if !domain.OwnedBy(draft.owner, len(draft.input.ValueDomains)) {
		draft.reject(position)
		return
	}
	ids := make([]ValueDomainEntryID, len(entries))
	for index, entry := range entries {
		if !entry.OwnedBy(draft.owner) {
			draft.reject(position)
			return
		}
		ids[index] = entry.ID()
	}
	record := draft.input.ValueDomains[domain.Index()]
	record.entries = append([]ValueDomainEntryID(nil), ids...)
	draft.input.ValueDomains[domain.Index()] = record
}

func (draft *declarationCandidate) ValueDomainUse(domain valueDomainRef, destination acceptanceDestinationRef, kind ValueDomainUseKind, position declarationPosition) {
	draft.mustOpen()
	if !domain.OwnedBy(draft.owner, len(draft.input.ValueDomains)) || !destination.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	use := acceptValueDomainUseRecord(destination.ID(), kind)
	record := draft.input.ValueDomains[domain.Index()]
	record.uses = append(append([]valueDomainUseRecord(nil), record.uses...), use)
	draft.input.ValueDomains[domain.Index()] = record
}

func (draft *declarationCandidate) EnvironmentInputWithSyntax(command acceptanceCommandRef, destination acceptanceDestinationRef, position declarationPosition, selector string, prefix bool, contribution environmentContribution, delimiter string, sensitive bool) environmentInputRef {
	draft.mustOpen()
	id := draft.ids.AllocateEnvironmentInputID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.EnvironmentInputs))
	if !command.OwnedBy(draft.owner) || !destination.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	input2 := environmentInputRecordInput{
		ID:           id,
		Command:      command.ID(),
		Destination:  destination.ID(),
		Position:     position,
		Selector:     selector,
		Prefix:       prefix,
		Contribution: contribution,
		Delimiter:    delimiter,
		Sensitive:    sensitive,
	}
	draft.input.EnvironmentInputs = append(
		draft.input.EnvironmentInputs,
		acceptEnvironmentInputRecordWithSyntax(input2),
	)
	return result
}

func (draft *declarationCandidate) EnvironmentInputLifecycle(input environmentInputRef, lifecycle ReleaseLifecycle, position declarationPosition) {
	draft.mustOpen()
	if !input.OwnedBy(draft.owner) || !draft.validLifecycle(lifecycle) {
		draft.reject(position)
		return
	}
	if !draft.claimLifecycle(lifecycleFamilyEnvironmentInput, input.ID().uint32(), position) {
		return
	}
	index, record, ok := draft.environmentInputRecordForRef(input)
	if !ok {
		draft.reject(position)
		return
	}
	record.lifecycle = lifecycle
	draft.input.EnvironmentInputs[index] = record
}

func (draft *declarationCandidate) claimLifecycle(family uint8, id uint32, position declarationPosition) bool {
	if draft.lifecycleAssignments == nil {
		draft.lifecycleAssignments = make(map[lifecycleAssignment]struct{})
	}
	key := lifecycleAssignment{
		family: family,
		id:     id,
	}
	if _, duplicate := draft.lifecycleAssignments[key]; duplicate {
		draft.reject(position)
		return false
	}
	draft.lifecycleAssignments[key] = struct{}{}
	return true
}

func (draft *declarationCandidate) validLifecycle(lifecycle ReleaseLifecycle) bool {
	if !draft.hasApplication {
		return false
	}
	return validateLifecycle(draft.application.release, lifecycle) == nil
}

func (draft *declarationCandidate) GrammarProfile(command acceptanceCommandRef, position declarationPosition) grammarProfileRef {
	draft.mustOpen()
	id := draft.ids.AllocateGrammarProfileID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.GrammarProfiles))
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.GrammarProfiles = append(
		draft.input.GrammarProfiles,
		acceptGrammarProfileRecord(id, command.ID(), position),
	)
	return result
}

func (draft *declarationCandidate) SignalEndpoint(command acceptanceCommandRef, position declarationPosition) signalEndpointRef {
	draft.mustOpen()
	id := draft.ids.AllocateSignalEndpointID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.SignalEndpoints))
	if !command.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.SignalEndpoints = append(
		draft.input.SignalEndpoints,
		acceptSignalEndpointRecord(id, command.ID(), position),
	)
	return result
}

func (draft *declarationCandidate) SignalEndpointBinding(endpoint signalEndpointRef, binding configurationBinding, adapt signalEndpointAdapter, position declarationPosition) {
	draft.mustOpen()
	if !endpoint.OwnedBy(draft.owner) || !binding.Valid() || adapt == nil {
		draft.reject(position)
		return
	}
	index, record, ok := draft.signalEndpointRecordForRef(endpoint)
	if !ok {
		draft.reject(position)
		return
	}
	if _, exists := record.Binding(); exists {
		rejection := rejection{
			Kind:  rejectionStructuralInvalidity,
			Order: uint64(position),
		}
		draft.preflight = append(draft.preflight, rejection)
		return
	}
	record.binding = binding
	record.adapt = adapt
	draft.input.SignalEndpoints[index] = record
}

func (draft *declarationCandidate) SignalCounter(policy signalPolicyRef, position declarationPosition) signalCounterRef {
	draft.mustOpen()
	id := draft.ids.AllocateSignalCounterID()
	result := issueIndexedStagingRef(draft.owner, id, len(draft.input.SignalCounters))
	if !policy.OwnedBy(draft.owner) {
		draft.reject(position)
		return result
	}
	draft.input.SignalCounters = append(draft.input.SignalCounters, acceptSignalCounterRecord(id, policy.ID(), position))
	draft.SignalPolicyCounter(policy, result, position)
	return result
}

func (draft *declarationCandidate) SignalPolicyCounter(policy signalPolicyRef, counter signalCounterRef, position declarationPosition) {
	draft.mustOpen()
	if !policy.OwnedBy(draft.owner) || !counter.OwnedBy(draft.owner) {
		draft.reject(position)
		return
	}
	index, record, ok := draft.signalPolicyRecordForRef(policy)
	if !ok {
		draft.reject(position)
		return
	}
	if _, exists := record.Counter(); exists {
		rejection := rejection{
			Kind:  rejectionStructuralInvalidity,
			Order: uint64(position),
		}
		draft.preflight = append(draft.preflight, rejection)
		return
	}
	record.counter = counter.ID()
	draft.input.SignalPolicies[index] = record
}

func (draft *declarationCandidate) DeclarationReplacement(source, target replacementRef, position declarationPosition) {
	draft.mustOpen()
	if !source.OwnedBy(draft.owner) || !target.OwnedBy(draft.owner) || source.Identity() == target.Identity() {
		draft.reject(position)
		return
	}
	draft.input.Replacements = append(
		draft.input.Replacements,
		acceptDeclarationReplacementRecord(source.Identity(), target.Identity(), position),
	)
}

// stagingOwner is an opaque transaction-local ownership token for declaration
// references. It is implementation identity only and never becomes semantic
// identity.
type stagingOwner struct{ token *uint64 }

func allocateStagingOwner() stagingOwner {
	return stagingOwner{
		token: new(uint64),
	}
}

func (owner stagingOwner) Valid() bool { return owner.token != nil }

// stagingRef is a transaction-local typed relationship reference. The semantic
// ID is application/operation scoped; the owner token prevents foreign or stale
// staging references from being accepted merely because numeric IDs coincide.
// recordCoordinate is an optional one-based transaction-local record coordinate.
// Zero means that no coordinate is known. Candidate record families are append-only
// while references are live, so the coordinate remains stable. Record lookup validates
// it against the semantic ID and falls back to ID lookup for reconstructed references.
type stagingRef[I interface{ present() bool }] struct {
	owner            stagingOwner
	id               I
	recordCoordinate uint32
}

func issueStagingRef[I interface{ present() bool }](owner stagingOwner, id I) stagingRef[I] {
	ref := stagingRef[I]{
		owner: owner,
		id:    id,
	}
	return ref
}

func issueIndexedStagingRef[I interface{ present() bool }](owner stagingOwner, id I, recordIndex int) stagingRef[I] {
	if recordIndex < 0 {
		panic("cli: staging record coordinate out of range")
	}
	coordinate := uint64(recordIndex) + 1
	if coordinate > uint64(^uint32(0)) {
		panic("cli: staging record coordinate out of range")
	}
	ref := stagingRef[I]{
		owner:            owner,
		id:               id,
		recordCoordinate: uint32(coordinate),
	}
	return ref
}

func (ref stagingRef[I]) ID() I { return ref.id }

func (ref stagingRef[I]) RecordIndex() (int, bool) {
	if !ref.owner.Valid() || !ref.id.present() || ref.recordCoordinate == 0 {
		return 0, false
	}
	return int(ref.recordCoordinate - 1), true
}

func (ref stagingRef[I]) OwnedBy(owner stagingOwner) bool {
	return owner.Valid() && ref.owner.token == owner.token && ref.id.present()
}

func (ref stagingRef[I]) Owner() stagingOwner { return ref.owner }

type (
	acceptanceCommandRef      = stagingRef[CommandID]
	acceptanceDestinationRef  = stagingRef[ValueID]
	acceptanceFlagRef         = stagingRef[FlagID]
	acceptanceFlagSpellingRef = stagingRef[FlagSpellingID]
	argumentFormRef           = stagingRef[ArgumentFormID]
	argumentTermRef           = stagingRef[ArgumentTermID]
	commandEntryRef           = stagingRef[CommandEntryID]
	actionEntryRef            = stagingRef[ActionEntryID]
	defaultRef                = stagingRef[CommandDefaultID]
	providerRef               = stagingRef[CommandProviderID]
	ruleRef                   = stagingRef[RuleID]
	advisoryRef               = stagingRef[advisoryID]
	signalPolicyRef           = stagingRef[SignalPolicyID]
	domainEntryRef            = stagingRef[ValueDomainEntryID]
	environmentInputRef       = stagingRef[EnvironmentInputID]
	grammarProfileRef         = stagingRef[GrammarProfileID]
	signalEndpointRef         = stagingRef[SignalEndpointID]
	signalCounterRef          = stagingRef[SignalCounterID]
)

type replacementRef struct {
	owner    stagingOwner
	identity declarationReplacementIdentity
}

func issueReplacementRef(owner stagingOwner, identity declarationReplacementIdentity) replacementRef {
	return replacementRef{
		owner:    owner,
		identity: identity,
	}
}

func (ref replacementRef) OwnedBy(owner stagingOwner) bool {
	return owner.Valid() && ref.owner.token == owner.token && ref.identity.Valid()
}

func (ref replacementRef) Identity() declarationReplacementIdentity { return ref.identity }

func replacementCommandEntry(ref commandEntryRef) replacementRef {
	return issueReplacementRef(ref.Owner(), commandEntryReplacementIdentity(ref.ID()))
}

func replacementFlag(ref acceptanceFlagRef) replacementRef {
	return issueReplacementRef(ref.Owner(), flagReplacementIdentity(ref.ID()))
}

func replacementFlagSpelling(ref acceptanceFlagSpellingRef) replacementRef {
	return issueReplacementRef(ref.Owner(), flagSpellingReplacementIdentity(ref.ID()))
}

func replacementArgumentForm(ref argumentFormRef) replacementRef {
	return issueReplacementRef(ref.Owner(), argumentFormReplacementIdentity(ref.ID()))
}

func replacementArgumentTerm(ref argumentTermRef) replacementRef {
	return issueReplacementRef(ref.Owner(), argumentTermReplacementIdentity(ref.ID()))
}

func replacementEnvironmentInput(ref environmentInputRef) replacementRef {
	return issueReplacementRef(ref.Owner(), environmentInputReplacementIdentity(ref.ID()))
}

type valueDomainRef struct {
	owner stagingOwner
	index int
}

func issueValueDomainRef(owner stagingOwner, index int) valueDomainRef {
	return valueDomainRef{
		owner: owner,
		index: index,
	}
}

func (ref valueDomainRef) OwnedBy(owner stagingOwner, count int) bool {
	return owner.Valid() && ref.owner.token == owner.token && ref.index >= 0 && ref.index < count
}

func (ref valueDomainRef) Index() int { return ref.index }
