package cli

import (
	"cmp"
	"errors"
	"slices"
)

type referenceSource struct {
	*staticArtifactSource
}

func prepareReferenceSource(valid bool, application applicationMetadata, records staticRecords, grammar grammarView) (*referenceSource, error) {
	source, err := prepareStaticArtifactSource(
		valid,
		application,
		records,
		grammar,
		analyzeRecordReachability(records),
	)
	if err != nil {
		return nil, err
	}
	reference := &referenceSource{
		staticArtifactSource: source,
	}
	return reference, nil
}

func prepareApplicationReferenceSource(model staticProgram, grammar grammarView) (*referenceSource, error) {
	records := model.Records()
	return prepareReferenceSource(model.Valid(), model.Application(), records, grammar)
}

func prepareEffectiveReferenceSource(model *effectiveModel) (*referenceSource, grammarView, error) {
	records, grammar, available := model.schemaSnapshot()
	if !available {
		return nil, nil, errInvalidModelTopology
	}
	source, err := prepareReferenceSource(model.Valid(), model.Application(), records, grammar)
	if err != nil {
		return nil, nil, err
	}
	return source, grammar, nil
}

func referenceCommandPath(source *referenceSource, ancestor, target CommandID) ([]canonicalCommandPathStep, error) {
	if source == nil || !ancestor.present() || !target.present() {
		return nil, errInvalidModelTopology
	}
	if _, ok := source.command(ancestor); !ok {
		return nil, errInvalidModelTopology
	}
	if ancestor == target {
		return nil, nil
	}
	seen := make(map[CommandID]struct{})
	var reversed []canonicalCommandPathStep
	current := target
	for current != ancestor {
		if _, duplicate := seen[current]; duplicate {
			return nil, errInvalidModelTopology
		}
		seen[current] = struct{}{}
		command, ok := source.command(current)
		if !ok {
			return nil, errInvalidModelTopology
		}
		parent, ok := command.Parent()
		if !ok {
			return nil, errInvalidModelTopology
		}
		entryID, ok := source.canonicalEntries[current]
		if !ok {
			return nil, errInvalidModelTopology
		}
		entry, ok := source.entryGrammar[entryID]
		if !ok || entry.kind != CommandEntryCanonical || entry.parent != parent || entry.target != current {
			return nil, errInvalidModelTopology
		}
		if _, ok := source.commandEntry(entryID); !ok {
			return nil, errInvalidModelTopology
		}
		reversed = append(reversed, canonicalCommandPathStep{
			Command: current,
			Entry:   entryID,
		})
		current = parent
	}
	return reverseCanonicalCommandPath(reversed), nil
}

func effectiveReferenceCommandPathProjection(model *effectiveModel, source *referenceSource, scheme ReleaseScheme, command CommandID, visibility ReferenceVisibility) (commandPathProjectionState, error) {
	root := source.root
	rootRecord, ok := source.command(root)
	if !ok {
		return commandPathProjectionState{}, errInvalidModelTopology
	}
	path, err := referenceCommandPath(source, root, command)
	if err != nil {
		return commandPathProjectionState{}, err
	}
	lifecycle, err := intersectCanonicalCommandPathLifecycle(model, scheme, path)
	if err != nil {
		return commandPathProjectionState{}, err
	}
	records := make([]commandRecord, len(path))
	for index, step := range path {
		record, ok := source.command(step.Command)
		if !ok {
			return commandPathProjectionState{}, errInvalidModelTopology
		}
		records[index] = record
	}
	return projectCommandPathState(lifecycle, visibility, rootRecord, records), nil
}

func effectiveReferenceCommandEntryProjection(model *effectiveModel, source *referenceSource, scheme ReleaseScheme, release string, visibility ReferenceVisibility, id CommandEntryID) (commandEntryProjectionState, error) {
	record, ok := source.commandEntry(id)
	if !ok {
		return commandEntryProjectionState{}, errInvalidModelTopology
	}
	entry, ok := source.entryGrammar[id]
	if !ok || entry.parent != record.parent {
		return commandEntryProjectionState{}, errInvalidModelTopology
	}
	parent, err := effectiveReferenceCommandPathProjection(model, source, scheme, record.parent, visibility)
	if err != nil {
		return commandEntryProjectionState{}, err
	}
	var target *commandPathProjectionState
	if entry.kind == CommandEntryAlias || entry.kind == CommandEntryShortcut {
		projected, err := effectiveReferenceCommandPathProjection(model, source, scheme, entry.target, visibility)
		if err != nil {
			return commandEntryProjectionState{}, err
		}
		target = &projected
	}
	return projectCommandEntryState(scheme, release, visibility, record, parent, target)
}

type effectiveReferenceProjector struct {
	referenceProjectionState

	model   *effectiveModel
	grammar grammarView
}

func projectEffectiveReference(model *effectiveModel, options ReferenceOptions, providerLocalization func(CommandID) localizationSnapshot, providerEntryLocalization func(CommandEntryID) localizationSnapshot) (referenceProjection, error) {
	source, grammar, err := prepareEffectiveReferenceSource(model)
	if err != nil {
		return referenceProjection{}, err
	}
	state, err := prepareApplicationReferenceProjectionState(options, source, model.Application(), providerLocalization, providerEntryLocalization)
	if err != nil {
		return referenceProjection{}, err
	}
	projector := effectiveReferenceProjector{
		model:                    model,
		grammar:                  grammar,
		referenceProjectionState: state,
	}

	return projector.project()
}

func (p *effectiveReferenceProjector) project() (referenceProjection, error) {
	if err := p.projectCommands(); err != nil {
		return referenceProjection{}, err
	}
	for _, id := range p.source.commandOrder {
		p.commandIsAvailable(id)
		p.visibleCommand(id)
	}
	p.result.CommandOrder = p.projectCommandTopology()
	if err := p.projectDeclarations(); err != nil {
		return referenceProjection{}, err
	}
	if err := p.projectValuesAndSignals(); err != nil {
		return referenceProjection{}, err
	}
	return p.result, nil
}

func (p *effectiveReferenceProjector) localizationForCommand(command CommandID) localizationSnapshot {
	private := command
	if p.providerLocalization != nil {
		if localization := p.providerLocalization(private); localization.Enabled() {
			return localization
		}
	}
	return p.applicationLocalization
}

func (p *effectiveReferenceProjector) localizationForEntry(entry CommandEntryID, parent CommandID) localizationSnapshot {
	if p.providerEntryLocalization != nil {
		if localization := p.providerEntryLocalization(entry); localization.Enabled() {
			return localization
		}
	}
	return p.localizationForCommand(parent)
}

func (p *effectiveReferenceProjector) projectCommands() error {
	return p.referenceProjectionState.projectCommands(p.localizationForCommand)
}

func (p *effectiveReferenceProjector) projectCommandTopology() []CommandID {
	return p.referenceProjectionState.projectCommandTopology(p.commandEntryVisible, p.helpForCommand, p.commandOrder)
}

func (p *effectiveReferenceProjector) projectDeclarations() error {
	projectCommandEntry := func(id CommandEntryID) (commandEntryProjectionState, error) {
		return effectiveReferenceCommandEntryProjection(p.model, p.source, p.scheme, p.release, p.options.Visibility, id)
	}
	return p.referenceProjectionState.projectDeclarations(projectCommandEntry, p.localizationForEntry, p.localizationForCommand)
}

func (p *effectiveReferenceProjector) projectValuesAndSignals() error {
	return p.referenceProjectionState.projectValuesAndSignals(p.localizationForCommand)
}

func (p *effectiveReferenceProjector) helpForCommand(commandID CommandID) HelpSchema {
	return projectReferenceCommandHelp(p.source, p.localizationForCommand(commandID), p.locale, commandID, p.visibleCommand, p.commandEntryVisible, p.flagVisible)
}

func (p *effectiveReferenceProjector) commandOrder() []CommandID {
	return referenceCommandOrder(p.source, p.visibleCommand, p.options.Order)
}

func (p *effectiveReferenceProjector) commandEntryVisible(id CommandEntryID) bool {
	state, err := effectiveReferenceCommandEntryProjection(
		p.model,
		p.source,
		p.scheme,
		p.release,
		p.options.Visibility,
		id,
	)
	return err == nil && state.visible
}

var errInvalidStaticProjectionInput = errors.New("cli: invalid reference options")

func projectApplicationReference(model staticProgram, options ReferenceOptions) (Reference, error) {
	projection, err := projectReference(model, options, nil, nil)
	if err != nil {
		return Reference{}, err
	}
	return projection.public(), nil
}

func projectOperationReference(operation *invocation, options ReferenceOptions) (Reference, error) {
	projection, err := projectOperationReferenceProjection(operation, options)
	if err != nil {
		return Reference{}, err
	}
	return projection.public(), nil
}

func projectOperationReferenceProjection(operation *invocation, options ReferenceOptions) (referenceProjection, error) {
	if operation == nil {
		return referenceProjection{}, errInvalidStaticProjectionInput
	}
	return projectEffectiveReference(
		operation.analysis.effectiveModel,
		options,
		operation.LocalizationForCommand,
		operation.LocalizationForCommandEntry,
	)
}

type referenceProjection struct {
	Locale           Locale
	EvaluatedRelease string

	Root         CommandID
	CommandOrder []CommandID
	Commands     map[CommandID]CommandReference

	CommandEntries   map[CommandEntryID]CommandEntryReference
	ActionEntries    map[ActionEntryID]ActionEntryReference
	CommandDefaults  map[CommandDefaultID]CommandDefaultReference
	CommandProviders map[CommandProviderID]CommandProviderReference

	Flags              map[FlagID]FlagReference
	FlagSpellings      map[FlagSpellingID]FlagSpellingReference
	ArgumentForms      map[ArgumentFormID]ArgumentFormReference
	ArgumentTerms      map[ArgumentTermID]ArgumentTermReference
	Values             map[ValueID]ValueReference
	Facts              map[ValueID]FactReference
	EnvironmentInputs  map[EnvironmentInputID]EnvironmentInputReference
	ValueDomains       []ValueDomainReference
	ValueDomainEntries map[ValueDomainEntryID]ValueDomainEntryReference

	SignalPolicies  map[SignalPolicyID]SignalPolicyReference
	SignalEndpoints map[SignalEndpointID]SignalEndpoint
}

func (p referenceProjection) public() Reference {
	return Reference(p)
}

type referenceProjectionState struct {
	options                   ReferenceOptions
	source                    *referenceSource
	scheme                    ReleaseScheme
	release                   string
	locale                    string
	providerLocalization      func(CommandID) localizationSnapshot
	providerEntryLocalization func(CommandEntryID) localizationSnapshot
	applicationLocalization   localizationSnapshot
	result                    referenceProjection
	ownCommandAvailable       map[CommandID]bool
	commandAvailable          map[CommandID]bool
	commandVisible            map[CommandID]bool
}

func (p *referenceProjectionState) commandIsAvailable(id CommandID) bool {
	if available, ok := p.commandAvailable[id]; ok {
		return available
	}
	record, ok := p.source.command(id)
	if !ok || !p.ownCommandAvailable[id] {
		p.commandAvailable[id] = false
		return false
	}
	parent, hasParent := record.Parent()
	if hasParent && !p.commandIsAvailable(parent) {
		p.commandAvailable[id] = false
		return false
	}
	p.commandAvailable[id] = true
	return true
}

func (p *referenceProjectionState) visibleCommand(id CommandID) bool {
	if visible, ok := p.commandVisible[id]; ok {
		return visible
	}
	record, ok := p.source.command(id)
	if !ok || !p.commandIsAvailable(id) ||
		(p.options.Visibility == ReferenceVisibilityPublic && record.presentation.hidden) {
		p.commandVisible[id] = false
		return false
	}
	parent, hasParent := record.Parent()
	if hasParent && !p.visibleCommand(parent) {
		p.commandVisible[id] = false
		return false
	}
	p.commandVisible[id] = true
	return true
}

func (p *referenceProjectionState) actionEntryVisible(id ActionEntryID) bool {
	record, ok := p.source.actionEntry(id)
	if !ok {
		return false
	}
	state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
	return err == nil && referenceAvailable(state) &&
		(p.options.Visibility == ReferenceVisibilityAll || !record.presentation.hidden)
}

func (p *referenceProjectionState) flagVisible(id FlagID) bool {
	record, ok := p.source.flag(id)
	if !ok {
		return false
	}
	state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
	return err == nil && referenceAvailable(state) &&
		(p.options.Visibility == ReferenceVisibilityAll || !record.presentation.hidden)
}

func (p *referenceProjectionState) flagSpellingVisible(id FlagSpellingID) bool {
	record, ok := p.source.flagSpelling(id)
	if !ok {
		return false
	}
	state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
	if err != nil || !referenceAvailable(state) {
		return false
	}
	flag, ok := p.source.flag(record.flag)
	return ok && (p.options.Visibility == ReferenceVisibilityAll || (!flag.presentation.hidden && !record.hidden))
}

func (p *referenceProjectionState) formVisible(id ArgumentFormID) bool {
	record, ok := p.source.argumentForm(id)
	if !ok {
		return false
	}
	state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
	return err == nil && referenceAvailable(state)
}

func (p *referenceProjectionState) argumentTermVisible(id ArgumentTermID) bool {
	record, ok := p.source.argumentTerm(id)
	if !ok {
		return false
	}
	state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
	return err == nil && referenceAvailable(state) &&
		(p.options.Visibility == ReferenceVisibilityAll || !record.presentation.hidden)
}

type referenceProjector struct {
	referenceProjectionState

	model   staticProgram
	grammar grammarView
}

func validateReferenceOptions(options ReferenceOptions) error {
	if options.Visibility != ReferenceVisibilityPublic && options.Visibility != ReferenceVisibilityAll {
		return errInvalidStaticProjectionInput
	}
	if options.Order != ReferenceOrderDeclaration && options.Order != ReferenceOrderName {
		return errInvalidStaticProjectionInput
	}
	return nil
}

func selectReferenceRelease(scheme ReleaseScheme, requested string) (string, error) {
	release := requested
	if release == "" && scheme.valid() {
		release = scheme.current()
	}
	if release != "" && (!scheme.valid() || scheme.validate(release) != nil) {
		return "", errInvalidStaticProjectionInput
	}
	return release, nil
}

func selectReferenceLocale(options ReferenceOptions, localization localizationSnapshot) (string, error) {
	locale, err := selectEffectiveLocale(string(options.Locale), "", localization.SourceLocale())
	if err != nil {
		return "", errInvalidStaticProjectionInput
	}
	return locale, nil
}

func prepareReferenceProjectionResult(locale, release string, source *referenceSource) referenceProjection {
	records := source.records
	result := referenceProjection{
		Locale:             Locale(locale),
		EvaluatedRelease:   release,
		Root:               source.root,
		Commands:           make(map[CommandID]CommandReference, len(records.commands)),
		CommandEntries:     make(map[CommandEntryID]CommandEntryReference, len(records.commandEntries)),
		ActionEntries:      make(map[ActionEntryID]ActionEntryReference, len(records.actionEntries)),
		CommandDefaults:    make(map[CommandDefaultID]CommandDefaultReference, len(records.defaults)),
		CommandProviders:   make(map[CommandProviderID]CommandProviderReference, len(records.providers)),
		Flags:              make(map[FlagID]FlagReference, len(records.flags)),
		FlagSpellings:      make(map[FlagSpellingID]FlagSpellingReference, len(records.flagSpellings)),
		ArgumentForms:      make(map[ArgumentFormID]ArgumentFormReference, len(records.argumentForms)),
		ArgumentTerms:      make(map[ArgumentTermID]ArgumentTermReference, len(records.argumentTerms)),
		Values:             make(map[ValueID]ValueReference, len(records.destinations)),
		Facts:              make(map[ValueID]FactReference),
		EnvironmentInputs:  make(map[EnvironmentInputID]EnvironmentInputReference, len(records.environmentInputs)),
		ValueDomains:       make([]ValueDomainReference, len(records.valueDomains)),
		ValueDomainEntries: make(map[ValueDomainEntryID]ValueDomainEntryReference, len(records.domainEntries)),
		SignalPolicies:     make(map[SignalPolicyID]SignalPolicyReference, len(records.signalPolicies)),
		SignalEndpoints:    make(map[SignalEndpointID]SignalEndpoint, len(records.signalEndpoints)),
	}
	return result
}

func prepareReferenceProjectionState(options ReferenceOptions, source *referenceSource, scheme ReleaseScheme, release, locale string, providerLocalization func(CommandID) localizationSnapshot, providerEntryLocalization func(CommandEntryID) localizationSnapshot, applicationLocalization localizationSnapshot) referenceProjectionState {
	state := referenceProjectionState{
		options:                   options,
		source:                    source,
		scheme:                    scheme,
		release:                   release,
		locale:                    locale,
		providerLocalization:      providerLocalization,
		providerEntryLocalization: providerEntryLocalization,
		applicationLocalization:   applicationLocalization,
		result:                    prepareReferenceProjectionResult(locale, release, source),
		ownCommandAvailable:       make(map[CommandID]bool, len(source.records.commands)),
		commandAvailable:          make(map[CommandID]bool, len(source.records.commands)),
		commandVisible:            make(map[CommandID]bool, len(source.records.commands)),
	}
	return state
}

func prepareApplicationReferenceProjectionState(options ReferenceOptions, source *referenceSource, application applicationMetadata, providerLocalization func(CommandID) localizationSnapshot, providerEntryLocalization func(CommandEntryID) localizationSnapshot) (referenceProjectionState, error) {
	if err := validateReferenceOptions(options); err != nil {
		return referenceProjectionState{}, err
	}
	scheme := application.release
	release, err := selectReferenceRelease(scheme, options.Release)
	if err != nil {
		return referenceProjectionState{}, err
	}
	localization := application.localization
	locale, err := selectReferenceLocale(options, localization)
	if err != nil {
		return referenceProjectionState{}, err
	}
	state := prepareReferenceProjectionState(options, source, scheme, release, locale, providerLocalization, providerEntryLocalization, localization)
	return state, nil
}

func projectReference(model staticProgram, options ReferenceOptions, providerLocalization func(CommandID) localizationSnapshot, providerEntryLocalization func(CommandEntryID) localizationSnapshot) (referenceProjection, error) {
	grammar, available := model.EffectiveGrammar()
	if !available {
		return referenceProjection{}, errInvalidModelTopology
	}
	source, err := prepareApplicationReferenceSource(model, grammar)
	if err != nil {
		return referenceProjection{}, err
	}
	state, err := prepareApplicationReferenceProjectionState(options, source, model.Application(), providerLocalization, providerEntryLocalization)
	if err != nil {
		return referenceProjection{}, err
	}
	projector := referenceProjector{
		model:                    model,
		grammar:                  grammar,
		referenceProjectionState: state,
	}
	return projector.project()
}

func (p *referenceProjector) project() (referenceProjection, error) {
	if err := p.projectCommands(); err != nil {
		return referenceProjection{}, err
	}
	for _, id := range p.source.commandOrder {
		p.commandIsAvailable(id)
		p.visibleCommand(id)
	}
	p.result.CommandOrder = p.projectCommandTopology()
	if err := p.projectDeclarations(); err != nil {
		return referenceProjection{}, err
	}
	if err := p.projectValuesAndSignals(); err != nil {
		return referenceProjection{}, err
	}
	return p.result, nil
}

func (p *referenceProjector) localizationForCommand(command CommandID) localizationSnapshot {
	private := command
	if p.providerLocalization != nil {
		if localization := p.providerLocalization(private); localization.Enabled() {
			return localization
		}
	}
	return p.applicationLocalization
}

func (p *referenceProjector) localizationForEntry(entry CommandEntryID, parent CommandID) localizationSnapshot {
	if p.providerEntryLocalization != nil {
		if localization := p.providerEntryLocalization(entry); localization.Enabled() {
			return localization
		}
	}
	return p.localizationForCommand(parent)
}

func (p *referenceProjector) projectCommands() error {
	return p.referenceProjectionState.projectCommands(p.localizationForCommand)
}

func (p *staticArtifactSource) genericReplacement(source declarationReplacementIdentity) (*DeclarationReplacement, bool, error) {
	for _, relation := range p.records.replacements {
		if relation.source != source {
			continue
		}
		target, targetLifecycle, err := p.projectReplacementIdentity(relation.target)
		if err != nil {
			return nil, false, err
		}
		_, sourceLifecycle, err := p.projectReplacementIdentity(relation.source)
		if err != nil {
			return nil, false, err
		}
		presentation, err := p.replacementPresentation(relation.target)
		if err != nil {
			return nil, false, err
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
		return replacement, true, nil
	}
	return nil, false, nil
}

func (p *staticArtifactSource) commandEntryReplacement(record commandEntryRecord) (*DeclarationReplacement, bool, error) {
	if targetID, present := record.Replacement(); present {
		target, ok := p.commandEntry(targetID)
		if !ok {
			return nil, false, errInvalidModelTopology
		}
		presentation, err := p.replacementPresentation(commandEntryReplacementIdentity(targetID))
		if err != nil {
			return nil, false, err
		}
		identity := DeclarationIdentity{
			Kind:         DeclarationKindCommandEntry,
			CommandEntry: targetID,
		}
		replacement := &DeclarationReplacement{
			Target:       identity,
			Presentation: presentation,
			TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
				p.application.release,
				record.lifecycle,
				target.lifecycle,
			),
		}
		return replacement, true, nil
	}
	return p.genericReplacement(commandEntryReplacementIdentity(record.meta.id))
}

func (p *staticArtifactSource) actionEntryReplacement(record actionEntryRecord) (*DeclarationReplacement, bool, error) {
	targetID, present := record.Replacement()
	if !present {
		return nil, false, nil
	}
	target, ok := p.actionEntry(targetID)
	if !ok {
		return nil, false, errInvalidModelTopology
	}
	presentation, err := p.actionEntryReplacementPresentation(target)
	if err != nil {
		return nil, false, err
	}
	identity := DeclarationIdentity{
		Kind:        DeclarationKindActionEntry,
		ActionEntry: targetID,
	}
	replacement := &DeclarationReplacement{
		Target:       identity,
		Presentation: presentation,
		TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
			p.application.release,
			record.lifecycle,
			target.lifecycle,
		),
	}
	return replacement, true, nil
}

func (p *staticArtifactSource) domainEntryReplacement(record domainEntryRecord) (*DeclarationReplacement, bool, error) {
	targetID, present := record.Replacement()
	if !present {
		return nil, false, nil
	}
	target, ok := p.domainEntry(targetID)
	if !ok {
		return nil, false, errInvalidModelTopology
	}
	presentation, err := p.domainReplacementPresentation(target)
	if err != nil {
		return nil, false, err
	}
	identity := DeclarationIdentity{
		Kind:             DeclarationKindValueDomainEntry,
		ValueDomainEntry: targetID,
	}
	replacement := &DeclarationReplacement{
		Target:       identity,
		Presentation: presentation,
		TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
			p.application.release,
			record.lifecycle,
			target.lifecycle,
		),
	}
	return replacement, true, nil
}

func (p *referenceSource) referenceCommandPolicies(commandID CommandID) ([]ConditionSchema, []RuleSchema, []AdvisorySchema, []NoticeSuppressionSchema, error) {
	conditions := make([]ConditionSchema, 0, len(p.conditionPoliciesByCommand[commandID]))
	for _, position := range p.conditionPoliciesByCommand[commandID] {
		record := p.records.conditionPolicies[position]
		identity, owner, err := p.projectConditionTarget(record.target)
		if err != nil || owner != commandID {
			return nil, nil, nil, nil, errInvalidModelTopology
		}
		expression, err := p.projectConditionExpression(record.condition)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		effect, ok := projectConditionEffect(record.effect)
		if !ok {
			return nil, nil, nil, nil, errInvalidModelTopology
		}
		condition := ConditionSchema{
			Target:       identity,
			Effect:       effect,
			Expression:   expression,
			Reachability: p.analysis.Condition(record.target, record.effect, record.position),
			Profile:      record.profile,
		}
		conditions = append(conditions, condition)
	}

	rules := make([]RuleSchema, 0, len(p.rulesByCommand[commandID]))
	for _, id := range p.rulesByCommand[commandID] {
		position, ok := p.rules[id]
		if !ok {
			return nil, nil, nil, nil, errInvalidModelTopology
		}
		projected, err := p.projectRule(p.records.rules[position])
		if err != nil {
			return nil, nil, nil, nil, err
		}
		rules = append(rules, projected)
	}

	advisories := make([]AdvisorySchema, 0, len(p.advisoriesByCommand[commandID]))
	for _, id := range p.advisoriesByCommand[commandID] {
		position, ok := p.advisories[id]
		if !ok {
			return nil, nil, nil, nil, errInvalidModelTopology
		}
		projected, err := p.projectAdvisory(p.records.advisories[position])
		if err != nil {
			return nil, nil, nil, nil, err
		}
		advisories = append(advisories, projected)
	}

	suppressions := make([]NoticeSuppressionSchema, 0, len(p.noticeSuppressionsByCommand[commandID]))
	for _, position := range p.noticeSuppressionsByCommand[commandID] {
		record := p.records.noticeSuppressions[position]
		expression, err := p.projectConditionExpression(record.condition)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		match, ok := projectNoticeMatch(record.match)
		if !ok {
			return nil, nil, nil, nil, errInvalidModelTopology
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
		suppressions = append(suppressions, suppression)
	}
	return conditions, rules, advisories, suppressions, nil
}

func (p *referenceProjectionState) projectCommands(localizationForCommand func(CommandID) localizationSnapshot) error {
	for _, id := range p.source.commandOrder {
		var facts, values []ValueID
		for _, destinationID := range p.source.destinationsByCommand[id] {
			destination, ok := p.source.destination(destinationID)
			if !ok {
				return errInvalidModelTopology
			}
			definition, ok := destination.Definition()
			if !ok {
				return errInvalidModelTopology
			}
			switch definition.Kind() {
			case destinationKindFactRequired, destinationKindFactResolved:
				facts = append(facts, destination.meta.id)
			default:
				values = append(values, destination.meta.id)
			}
		}
		if err := p.projectCommand(localizationForCommand, id, facts, values); err != nil {
			return err
		}
	}
	return nil
}

func (p *referenceProjectionState) projectCommand(localizationForCommand func(CommandID) localizationSnapshot, id CommandID, facts, values []ValueID) error {
	record, _ := p.source.command(id)
	path, err := p.source.commandPath(id)
	if err != nil {
		return err
	}
	name := p.source.application.name
	canonicalEntry := CommandEntryID(0)
	state := LifecycleStateCurrent
	if id != p.source.root {
		entryID, ok := p.source.canonicalEntries[id]
		if !ok {
			return errInvalidModelTopology
		}
		entry, _ := p.source.commandEntry(entryID)
		canonicalEntry = entryID
		name = p.source.entryGrammar[entryID].name
		state, err = referenceLifecycleState(p.scheme, entry.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
	}
	available := id == p.source.root || referenceAvailable(state)
	p.ownCommandAvailable[id] = available

	dialect, ok := p.source.grammar.Dialect(id)
	if !ok {
		return errInvalidModelTopology
	}
	publicDialect, err := projectPublicDialectRecord(dialect)
	if err != nil {
		return err
	}
	profiles, hasProfiles, err := p.source.projectGrammarProfileSet(id)
	if err != nil {
		return err
	}
	var profileSet *GrammarProfileSet
	if hasProfiles {
		profileSet = &profiles
	}

	conditions, rules, advisories, suppressions, err := p.source.referenceCommandPolicies(id)
	if err != nil {
		return err
	}
	presentation := record.presentation
	localization := localizationForCommand(id)
	p.result.Commands[id] = CommandReference{
		ID:                 id,
		CanonicalEntry:     canonicalEntry,
		Parent:             record.parent.id,
		Origin:             p.source.declarationOrigin(id),
		Name:               name,
		Path:               path,
		Description:        referenceDescription(localization, p.locale, presentation.description),
		Examples:           referenceExamples(localization, p.locale, presentation.Examples()),
		Hidden:             presentation.hidden,
		Runnable:           record.execution.Runnable(),
		Access:             projectExecutionAccessRecord(record.access),
		LifecycleState:     state,
		Dialect:            &publicDialect,
		Profiles:           profileSet,
		Help:               referenceHelp(localization, p.locale, p.source.projectHelp(id)),
		Facts:              facts,
		Values:             values,
		Conditions:         conditions,
		Rules:              rules,
		Advisories:         referenceAdvisories(localization, p.locale, advisories),
		NoticeSuppressions: suppressions,
		CommandDefaults:    slices.Clone(p.source.defaultsByParent[id]),
		CommandProviders:   slices.Clone(p.source.providersByParent[id]),
	}
	return nil
}

func (p *referenceProjector) projectCommandTopology() []CommandID {
	return p.referenceProjectionState.projectCommandTopology(p.commandEntryVisible, p.helpForCommand, p.commandOrder)
}

func (p *referenceProjectionState) projectCommandTopology(commandEntryVisible func(CommandEntryID) bool, helpForCommand func(CommandID) HelpSchema, commandOrder func() []CommandID) []CommandID {
	for id, command := range p.result.Commands {
		var flags []FlagID
		for _, flagID := range p.source.flagsByCommand[id] {
			if p.flagVisible(flagID) && p.visibleCommand(id) {
				flags = append(flags, flagID)
			}
		}
		var entries []CommandEntryID
		for _, entryID := range p.source.entriesByParent[id] {
			if commandEntryVisible(entryID) && p.visibleCommand(id) {
				entries = append(entries, entryID)
			}
		}
		p.projectCommandTopologyRecord(helpForCommand, id, command, flags, entries)
	}
	return commandOrder()
}

func (p *referenceProjectionState) projectCommandTopologyRecord(helpForCommand func(CommandID) HelpSchema, id CommandID, command CommandReference, flags []FlagID, entries []CommandEntryID) {
	for _, child := range p.source.childrenByCommand[id] {
		if p.visibleCommand(child) {
			command.Children = append(command.Children, child)
		}
	}
	command.Flags = flags
	for _, formID := range p.source.formsByCommand[id] {
		if p.formVisible(formID) && p.visibleCommand(id) {
			command.ArgumentForms = append(command.ArgumentForms, formID)
		}
	}
	command.SignalPolicies = slices.Clone(p.source.signalPoliciesByCommand[id])
	command.CommandEntries = entries
	for _, entryID := range p.source.actionEntriesByParent[id] {
		if p.actionEntryVisible(entryID) && p.visibleCommand(id) {
			command.ActionEntries = append(command.ActionEntries, entryID)
		}
	}
	command.CommandDefaults = filterCommandDefaultIDs(command.CommandDefaults, func(defaultID CommandDefaultID) bool {
		position, ok := p.source.defaults[defaultID]
		return ok && p.visibleCommand(p.source.records.defaults[position].target)
	})
	command.Help = helpForCommand(id)
	p.result.Commands[id] = command
}

func staticReferenceCommandPathProjection(model staticProgram, source *referenceSource, scheme ReleaseScheme, command CommandID, visibility ReferenceVisibility) (commandPathProjectionState, error) {
	root := source.root
	rootRecord, ok := source.command(root)
	if !ok {
		return commandPathProjectionState{}, errInvalidModelTopology
	}
	path, err := referenceCommandPath(source, root, command)
	if err != nil {
		return commandPathProjectionState{}, err
	}
	lifecycle, err := intersectStaticCommandPathLifecycle(model, scheme, path)
	if err != nil {
		return commandPathProjectionState{}, err
	}
	records := make([]commandRecord, len(path))
	for index, step := range path {
		record, ok := source.command(step.Command)
		if !ok {
			return commandPathProjectionState{}, errInvalidModelTopology
		}
		records[index] = record
	}
	return projectCommandPathState(lifecycle, visibility, rootRecord, records), nil
}

func staticReferenceCommandEntryProjection(model staticProgram, source *referenceSource, scheme ReleaseScheme, release string, visibility ReferenceVisibility, id CommandEntryID) (commandEntryProjectionState, error) {
	record, ok := source.commandEntry(id)
	if !ok {
		return commandEntryProjectionState{}, errInvalidModelTopology
	}
	entry, ok := source.entryGrammar[id]
	if !ok || entry.parent != record.parent {
		return commandEntryProjectionState{}, errInvalidModelTopology
	}
	parent, err := staticReferenceCommandPathProjection(model, source, scheme, record.parent, visibility)
	if err != nil {
		return commandEntryProjectionState{}, err
	}
	var target *commandPathProjectionState
	if entry.kind == CommandEntryAlias || entry.kind == CommandEntryShortcut {
		projected, err := staticReferenceCommandPathProjection(model, source, scheme, entry.target, visibility)
		if err != nil {
			return commandEntryProjectionState{}, err
		}
		target = &projected
	}
	return projectCommandEntryState(scheme, release, visibility, record, parent, target)
}

func (p *referenceProjector) projectDeclarations() error {
	projectCommandEntry := func(id CommandEntryID) (commandEntryProjectionState, error) {
		return staticReferenceCommandEntryProjection(p.model, p.source, p.scheme, p.release, p.options.Visibility, id)
	}
	return p.referenceProjectionState.projectDeclarations(projectCommandEntry, p.localizationForEntry, p.localizationForCommand)
}

func (p *referenceProjectionState) projectDeclarations(projectCommandEntry func(CommandEntryID) (commandEntryProjectionState, error), localizationForEntry func(CommandEntryID, CommandID) localizationSnapshot, localizationForCommand func(CommandID) localizationSnapshot) error {
	for _, record := range p.source.records.commandEntries {
		id := record.meta.id
		projection, err := projectCommandEntry(id)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		grammar, ok := p.source.entryGrammar[id]
		if !ok {
			return errInvalidModelTopology
		}
		targetPath := p.source.pathCommandIDs(record.target)
		if len(targetPath) == 0 {
			return errInvalidModelTopology
		}
		description := record.presentation.description
		descriptionLocalization := localizationForEntry(id, record.parent)
		if (grammar.kind == CommandEntryAlias || grammar.kind == CommandEntryShortcut) && description == (Description{}) {
			target, ok := p.source.command(record.target)
			if !ok {
				return errInvalidModelTopology
			}
			description = target.presentation.description
			descriptionLocalization = localizationForCommand(record.target)
		}
		replacement, _, err := p.source.commandEntryReplacement(record)
		if err != nil {
			return err
		}
		entry := CommandEntryReference{
			ID:               id,
			Parent:           record.parent,
			Kind:             grammar.kind,
			Origin:           p.source.declarationOrigin(record.target),
			DeclarationIndex: int(record.meta.position),
			Profile:          record.meta.profile,
			Name:             grammar.name,
			TargetPath:       targetPath,
			Description:      referenceDescription(descriptionLocalization, p.locale, description),
			Examples:         referenceExamples(localizationForEntry(id, record.parent), p.locale, record.presentation.Examples()),
			Hidden:           record.presentation.hidden,
			Available:        projection.available,
			Visible:          projection.visible,
			Release:          releaseLifecycle(record.lifecycle),
			LifecycleState:   projection.lifecycle,
			Replacement:      replacement,
		}
		if shortcut, ok := record.Shortcut(); ok {
			entry.Args = shortcut.Args()
			entry.AppendArgs = shortcut.AppendArgs()
			defaults, err := projectShortcutDefaults(shortcut, p.source.destination)
			if err != nil {
				return err
			}
			entry.Defaults = referenceShortcutDefaults(defaults)
		}
		p.result.CommandEntries[id] = entry
	}

	for _, record := range p.source.records.actionEntries {
		id := record.meta.id
		state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		entry := ActionEntryReference{
			ID:               id,
			Parent:           record.parent,
			Origin:           record.origin,
			DeclarationIndex: int(record.meta.position),
			Description:      referenceDescription(localizationForCommand(record.parent), p.locale, record.presentation.description),
			Hidden:           record.presentation.hidden,
			Available:        referenceAvailable(state) && p.commandIsAvailable(record.parent),
			Release:          releaseLifecycle(record.lifecycle),
			LifecycleState:   state,
		}
		entry.Visible = entry.Available && p.visibleCommand(record.parent) &&
			(p.options.Visibility == ReferenceVisibilityAll || !entry.Hidden)
		if grammar, ok := p.source.actionCommandGrammar[id]; ok {
			entry.Kind = ActionEntryCommand
			entry.Action = invocationAction(grammar.action)
			entry.Name = grammar.name
		} else if grammar, ok := p.source.actionFlagGrammar[id]; ok {
			entry.Kind = ActionEntryFlag
			entry.Action = invocationAction(grammar.action)
			entry.Placement = grammar.placement
			switch grammar.kind {
			case FlagSpellingLong:
				entry.Long = grammar.text
			case FlagSpellingShort:
				runes := []rune(grammar.text)
				if len(runes) != 2 || runes[0] != '-' {
					return errInvalidModelTopology
				}
				entry.Short = runes[1]
			case FlagSpellingLiteral:
				entry.Literal = grammar.text
			default:
				return errInvalidModelTopology
			}
		} else {
			return errInvalidModelTopology
		}
		replacement, _, err := p.source.actionEntryReplacement(record)
		entry.Replacement = replacement
		if err != nil {
			return err
		}
		p.result.ActionEntries[id] = entry
	}

	for _, record := range p.source.records.defaults {
		item := CommandDefaultReference{
			ID:               record.meta.id,
			Parent:           record.parent,
			Target:           record.target,
			Origin:           DeclarationOriginApplication,
			DeclarationIndex: int(record.meta.position),
			Profile:          record.meta.profile,
			Reachability:     p.source.analysis.Default(record.meta.id),
		}
		if condition, present := record.Condition(); present {
			projected, err := p.source.projectConditionExpression(condition)
			if err != nil {
				return err
			}
			item.Condition = &projected
		}
		p.result.CommandDefaults[item.ID] = item
	}
	for _, record := range p.source.records.providers {
		p.result.CommandProviders[record.meta.id] = CommandProviderReference{
			ID:               record.meta.id,
			Parent:           record.parent,
			DeclarationIndex: int(record.meta.position),
			Profile:          record.meta.profile,
			Dependencies:     dependencyIDs(record.definition.Dependencies()),
		}
	}

	for _, record := range p.source.records.flags {
		id := record.meta.id
		destination, ok := p.source.destination(record.destination)
		if !ok {
			return errInvalidModelTopology
		}
		definition, ok := destination.Definition()
		if !ok {
			return errInvalidModelTopology
		}
		valueType, ok := definition.ValueType()
		if !ok {
			return errInvalidModelTopology
		}
		state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		completion, present := record.Completion()
		publicCompletion, err := projectCompletion(completion, present)
		if err != nil {
			return err
		}
		grammar := p.source.flagGrammar[id]
		replacement, _, err := p.source.genericReplacement(flagReplacementIdentity(id))
		if err != nil {
			return err
		}
		available := referenceAvailable(state) && p.commandIsAvailable(record.command)
		visible := available && p.visibleCommand(record.command) &&
			(p.options.Visibility == ReferenceVisibilityAll || !record.presentation.hidden)
		p.result.Flags[id] = FlagReference{
			ID:                  id,
			Command:             record.command,
			Origin:              p.source.declarationOrigin(record.command),
			Profile:             record.meta.profile,
			Destination:         record.destination,
			Spellings:           filterFlagSpellingIDs(p.source.spellingsByFlag[id], p.flagSpellingVisible),
			ValueName:           record.valueName,
			Description:         referenceText(localizationForCommand(record.command), p.locale, record.presentation.description.Short),
			Type:                valueType,
			Occurrences:         countRange(grammar.occurrences),
			ValuesPerOccurrence: countRange(grammar.valuesPerOccurrence),
			Placement:           grammar.placement,
			Required:            record.required,
			Hidden:              record.presentation.hidden,
			Sensitive:           destination.sensitive,
			LocaleSelector:      record.localeSelector,
			Completion:          publicCompletion,
			Available:           available,
			Visible:             visible,
			Release:             releaseLifecycle(record.lifecycle),
			LifecycleState:      state,
			Replacement:         replacement,
		}
	}

	for _, record := range p.source.records.flagSpellings {
		id := record.meta.id
		state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		grammar := p.source.spellingGrammar[id]
		spelling := FlagSpelling{
			Kind:       grammar.kind,
			Text:       grammar.text,
			ValueMode:  grammar.mode,
			ValueForms: flagValueForms(grammar.Forms()),
			Sensitive:  grammar.sensitive,
		}
		flag, ok := p.source.flag(record.flag)
		if !ok {
			return errInvalidModelTopology
		}
		if value, present, sensitive := grammar.FixedValue(); present {
			destination, _ := p.source.destination(flag.destination)
			literal, err := projectValueLiteralRecord(destination, value, grammar.sensitive || sensitive)
			if err != nil {
				return err
			}
			spelling.FixedValue = &literal
		}
		if value, present, sensitive := grammar.ImplicitValue(); present {
			destination, _ := p.source.destination(flag.destination)
			literal, err := projectValueLiteralRecord(destination, value, grammar.sensitive || sensitive)
			if err != nil {
				return err
			}
			spelling.ImplicitValue = &literal
		}
		flagState, err := referenceLifecycleState(p.scheme, flag.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		replacement, _, err := p.source.genericReplacement(flagSpellingReplacementIdentity(id))
		if err != nil {
			return err
		}
		available := referenceAvailable(state) && referenceAvailable(flagState) && p.commandIsAvailable(flag.command)
		visible := available && p.visibleCommand(flag.command) &&
			(p.options.Visibility == ReferenceVisibilityAll || (!flag.presentation.hidden && !record.hidden))
		p.result.FlagSpellings[id] = FlagSpellingReference{
			ID:             id,
			Flag:           record.flag,
			Origin:         p.source.declarationOrigin(flag.command),
			Spelling:       spelling,
			Hidden:         record.hidden,
			Available:      available,
			Visible:        visible,
			Release:        releaseLifecycle(record.lifecycle),
			LifecycleState: state,
			Replacement:    replacement,
		}
	}

	for _, record := range p.source.records.argumentForms {
		id := record.meta.id
		state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		replacement, _, err := p.source.genericReplacement(argumentFormReplacementIdentity(id))
		if err != nil {
			return err
		}
		available := referenceAvailable(state) && p.commandIsAvailable(record.command)
		p.result.ArgumentForms[id] = ArgumentFormReference{
			ID:             id,
			Command:        record.command,
			Origin:         p.source.declarationOrigin(record.command),
			Profile:        record.meta.profile,
			Terms:          filterArgumentTermIDs(p.source.termsByForm[id], p.argumentTermVisible),
			Available:      available,
			Visible:        available && p.visibleCommand(record.command),
			Release:        releaseLifecycle(record.lifecycle),
			LifecycleState: state,
			Replacement:    replacement,
			Reachability:   p.source.analysis.ArgumentForm(id),
		}
	}

	for _, record := range p.source.records.argumentTerms {
		id := record.meta.id
		state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		form, ok := p.source.argumentForm(record.form)
		if !ok {
			return errInvalidModelTopology
		}
		formState, err := referenceLifecycleState(p.scheme, form.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		grammar := p.source.termGrammar[id]
		item := ArgumentTermReference{
			ID:              id,
			Form:            record.form,
			Origin:          p.source.declarationOrigin(form.command),
			Kind:            grammar.kind,
			Name:            record.name,
			Literal:         grammar.literal,
			Description:     referenceText(localizationForCommand(form.command), p.locale, record.presentation.description.Short),
			Count:           countRange(grammar.count),
			Optional:        grammar.count.Minimum() == 0,
			Greedy:          grammar.allocation == semanticArgumentAllocationGreedy,
			Minimal:         grammar.allocation == semanticArgumentAllocationMinimal,
			EndOfOptions:    grammar.endOfOptions,
			Hidden:          record.presentation.hidden,
			SourceSensitive: record.sourceSensitive,
			HyphenValues:    grammar.hyphenValues,
			Release:         releaseLifecycle(record.lifecycle),
			LifecycleState:  state,
		}
		if destinationID, present := record.Destination(); present {
			destination, _ := p.source.destination(destinationID)
			definition, _ := destination.Definition()
			valueType, ok := definition.ValueType()
			if !ok {
				return errInvalidModelTopology
			}
			item.Destination = destinationID
			item.Type = valueType
			item.Sensitive = destination.sensitive
			if item.Kind == ArgumentTermValue && valueType.Kind == ValueKindSequence {
				item.Kind = ArgumentTermSequence
			}
		}
		if grammar.tokenName != "" {
			item.TokenLanguage = &TokenLanguageSyntax{
				Fragments: grammar.TokenSyntax(),
			}
		}
		completion, present := record.Completion()
		item.Completion, err = projectCompletion(completion, present)
		if err != nil {
			return err
		}
		item.Available = referenceAvailable(state) && referenceAvailable(formState) && p.commandIsAvailable(form.command)
		item.Visible = item.Available && p.visibleCommand(form.command) &&
			(p.options.Visibility == ReferenceVisibilityAll || !item.Hidden)
		replacement, _, err := p.source.genericReplacement(argumentTermReplacementIdentity(id))
		item.Replacement = replacement
		if err != nil {
			return err
		}
		p.result.ArgumentTerms[id] = item
	}
	return nil
}

func (p *referenceProjector) projectValuesAndSignals() error {
	return p.referenceProjectionState.projectValuesAndSignals(p.localizationForCommand)
}

func (p *referenceProjectionState) projectValuesAndSignals(localizationForCommand func(CommandID) localizationSnapshot) error {
	for _, record := range p.source.records.destinations {
		value, fact, isFact, err := projectDestinationSchemaRecord(record, p.source.environmentByDestination[record.meta.id], p.source.declarationOrigin(record.command))
		if err != nil {
			return err
		}
		if isFact {
			p.result.Facts[record.meta.id] = FactReference{
				FactSchema: fact,
				Available:  p.commandIsAvailable(record.command),
			}
		}
		p.result.Values[record.meta.id] = ValueReference{
			ValueSchema: value,
			Available:   p.commandIsAvailable(record.command),
			Visible:     p.visibleCommand(record.command),
		}
	}

	for _, record := range p.source.records.environmentInputs {
		state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		artifactState, err := p.source.lifecycleState(record.lifecycle)
		if err != nil {
			return err
		}
		contribution, ok := valueCandidateContribution(record.contribution)
		if !ok {
			return errInvalidModelTopology
		}
		replacement, _, err := p.source.genericReplacement(environmentInputReplacementIdentity(record.meta.id))
		if err != nil {
			return err
		}
		environment := EnvironmentInputSchema{
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
			LifecycleState: artifactState,
			Replacement:    replacement,
		}
		p.result.EnvironmentInputs[record.meta.id] = EnvironmentInputReference{
			EnvironmentInputSchema: environment,
			Available:              referenceAvailable(state) && p.commandIsAvailable(record.command),
		}
	}

	domainEntryCommands := make(map[ValueDomainEntryID]CommandID)
	for i, record := range p.source.records.valueDomains {
		uses := record.Uses()
		for _, use := range uses {
			p.result.ValueDomains[i].Uses = append(p.result.ValueDomains[i].Uses, ValueDomainUse{
				Destination: use.destination,
				Kind:        use.kind,
			})
		}
		var owner CommandID
		for _, use := range uses {
			destination, ok := p.source.destination(use.destination)
			if !ok {
				continue
			}
			if owner == 0 {
				owner = destination.command
			} else if owner != destination.command {
				owner = 0
				break
			}
		}
		if owner != 0 {
			for _, entryID := range record.Entries() {
				domainEntryCommands[entryID] = owner
			}
		}
	}
	for _, record := range p.source.records.domainEntries {
		id := record.meta.id
		state, err := referenceLifecycleState(p.scheme, record.lifecycle, p.release)
		if err != nil {
			return errInvalidStaticProjectionInput
		}
		replacement, present, err := p.source.domainEntryReplacement(record)
		if err != nil {
			return err
		}
		if !present {
			replacement = nil
		}
		localization := p.applicationLocalization
		origin := DeclarationOriginApplication
		if owner, ok := domainEntryCommands[id]; ok {
			localization = localizationForCommand(owner)
			origin = p.source.declarationOrigin(owner)
		}
		available := referenceAvailable(state)
		p.result.ValueDomainEntries[id] = ValueDomainEntryReference{
			ID:        id,
			Kind:      valueDomainEntryKind(record.kind),
			Origin:    origin,
			Name:      record.name,
			Canonical: record.canonical,
			Description: referenceDescription(localization, p.locale, Description{
				Short: record.description,
			}),
			Hidden:         record.hidden,
			Available:      available,
			Visible:        available && (p.options.Visibility == ReferenceVisibilityAll || !record.hidden),
			Release:        releaseLifecycle(record.lifecycle),
			LifecycleState: state,
			Replacement:    replacement,
		}
	}
	for i, record := range p.source.records.valueDomains {
		for _, entryID := range record.Entries() {
			if entry, ok := p.result.ValueDomainEntries[entryID]; ok && entry.Visible {
				p.result.ValueDomains[i].Entries = append(p.result.ValueDomains[i].Entries, entryID)
			}
		}
	}

	for _, record := range p.source.records.signalPolicies {
		definition, present := record.Definition()
		if !present || !definition.Valid() {
			return errInvalidModelTopology
		}
		policy, err := p.source.projectSignalPolicy(record, definition, DeclarationOriginApplication)
		if err != nil {
			return err
		}
		policy.Hint = referenceText(localizationForCommand(record.command), p.locale, policy.Hint)
		p.result.SignalPolicies[record.meta.id] = SignalPolicyReference{
			SignalPolicySchema: policy,
			Available:          p.commandIsAvailable(record.command),
			Visible:            p.visibleCommand(record.command),
		}
	}
	for _, record := range p.source.records.signalEndpoints {
		p.result.SignalEndpoints[record.meta.id] = SignalEndpoint{
			ID:       record.meta.id,
			Command:  record.command,
			Origin:   DeclarationOriginApplication,
			Capacity: SignalBacklogCapacity,
			Policies: append([]SignalPolicyID(nil), p.source.endpointPolicies[record.meta.id]...),
		}
	}
	return nil
}

func referenceShortcutDefaults(source []CommandShortcutDefaultSchema) []CommandShortcutDefaultReference {
	out := make([]CommandShortcutDefaultReference, len(source))
	for i, item := range source {
		out[i] = CommandShortcutDefaultReference{
			Destination: item.Destination,
			Spelling:    item.Spelling,
			Values:      append([]ValueLiteral(nil), item.Values...),
			Sensitive:   item.Sensitive,
		}
	}
	return out
}

func referenceAdvisories(localization localizationSnapshot, locale string, source []AdvisorySchema) []AdvisoryReference {
	out := make([]AdvisoryReference, len(source))
	for i, item := range source {
		out[i] = AdvisoryReference{
			Code:             item.Code,
			Severity:         item.Severity,
			Message:          referenceText(localization, locale, item.Message),
			Reason:           referenceText(localization, locale, item.Reason),
			Hint:             referenceText(localization, locale, item.Hint),
			Command:          item.Command,
			Origin:           item.Origin,
			DeclarationIndex: item.DeclarationIndex,
			Profile:          item.Profile,
			Kind:             item.Kind,
			Activation:       item.Activation,
			Reachability:     item.Reachability,
			Subjects:         append([]ValueID(nil), item.Subjects...),
			Dependencies:     append([]ValueID(nil), item.Dependencies...),
		}
	}
	return out
}

func referenceLifecycleState(scheme ReleaseScheme, lifecycle ReleaseLifecycle, release string) (LifecycleState, error) {
	releaseLifecycleValue := ReleaseLifecycle{
		IntroducedIn: lifecycle.IntroducedIn,
		DeprecatedIn: lifecycle.DeprecatedIn,
		RemovedIn:    lifecycle.RemovedIn,
		Reason:       lifecycle.Reason,
	}
	state, err := evaluateLifecycle(scheme, releaseLifecycleValue, release)
	if err != nil {
		return LifecycleStateUnknown, err
	}
	switch state {
	case LifecycleStateNotIntroduced:
		return LifecycleStateNotIntroduced, nil
	case LifecycleStateCurrent:
		return LifecycleStateCurrent, nil
	case LifecycleStateDeprecated:
		return LifecycleStateDeprecated, nil
	case LifecycleStateRemoved:
		return LifecycleStateRemoved, nil
	default:
		return LifecycleStateUnknown, errInvalidStaticProjectionInput
	}
}

func referenceAvailable(state LifecycleState) bool {
	return state == LifecycleStateCurrent || state == LifecycleStateDeprecated
}

func referenceText(localization localizationSnapshot, locale, source string) string {
	text, _, err := localization.Resolve(locale, source)
	if err != nil {
		return source
	}
	return text
}

func referenceDescription(localization localizationSnapshot, locale string, source Description) Description {
	description := Description{
		Short: referenceText(localization, locale, source.Short),
		Long:  referenceText(localization, locale, source.Long),
	}
	return description
}

func referenceExamples(localization localizationSnapshot, locale string, source []Example) []Example {
	out := make([]Example, len(source))
	for i, example := range source {
		out[i] = Example{
			Description: referenceText(localization, locale, example.Description),
			CommandLine: example.CommandLine,
		}
	}
	return out
}

func referenceHelp(localization localizationSnapshot, locale string, source HelpSchema) HelpSchema {
	out := HelpSchema{
		CommandGroups: make([]HelpCommandGroupSchema, len(source.CommandGroups)),
		FlagGroups:    make([]HelpFlagGroupSchema, len(source.FlagGroups)),
	}
	for i, group := range source.CommandGroups {
		out.CommandGroups[i] = group
		out.CommandGroups[i].Title = referenceText(localization, locale, group.Title)
		out.CommandGroups[i].Entries = append([]HelpCommandGroupEntrySchema(nil), group.Entries...)
	}
	for i, group := range source.FlagGroups {
		out.FlagGroups[i] = group
		out.FlagGroups[i].Title = referenceText(localization, locale, group.Title)
		out.FlagGroups[i].Flags = append([]FlagID(nil), group.Flags...)
	}
	return out
}

func projectReferenceCommandHelp(source *referenceSource, localization localizationSnapshot, locale string, commandID CommandID, commandVisible func(CommandID) bool, commandEntryVisible func(CommandEntryID) bool, flagVisible func(FlagID) bool) HelpSchema {
	help := source.projectHelp(commandID)
	out := HelpSchema{}
	for _, group := range help.CommandGroups {
		projected := group
		projected.Title = referenceText(localization, locale, group.Title)
		projected.Entries = nil
		for _, entry := range group.Entries {
			keep := false
			switch entry.Kind {
			case HelpCommandGroupEntryCommandEntry:
				if record, ok := source.commandEntry(entry.CommandEntry); ok {
					keep = commandVisible(record.parent) && commandEntryVisible(entry.CommandEntry)
				}
			case HelpCommandGroupEntryCommandProvider:
				_, keep = source.providers[entry.Provider]
			case HelpCommandGroupEntryUnknown:
			}
			if keep {
				projected.Entries = append(projected.Entries, entry)
			}
		}
		out.CommandGroups = append(out.CommandGroups, projected)
	}
	for _, group := range help.FlagGroups {
		projected := group
		projected.Title = referenceText(localization, locale, group.Title)
		projected.Flags = filterFlagIDs(group.Flags, func(id FlagID) bool {
			flag, ok := source.flag(id)
			return ok && commandVisible(flag.command) && flagVisible(id)
		})
		out.FlagGroups = append(out.FlagGroups, projected)
	}
	return out
}

func referenceCommandOrder(source *referenceSource, visible func(CommandID) bool, order ReferenceOrder) []CommandID {
	if source.root == 0 || !visible(source.root) {
		return nil
	}
	if order == ReferenceOrderDeclaration {
		out := make([]CommandID, 0, len(source.commandOrder))
		for _, id := range source.commandOrder {
			if visible(id) {
				out = append(out, id)
			}
		}
		return out
	}
	name := func(id CommandID) string {
		if id == source.root {
			return source.application.name
		}
		entryID, ok := source.canonicalEntries[id]
		if !ok {
			return ""
		}
		return source.entryGrammar[entryID].name
	}
	out := make([]CommandID, 0, len(source.commandOrder))
	var visit func(CommandID)
	visit = func(id CommandID) {
		if !visible(id) {
			return
		}
		out = append(out, id)
		children := append([]CommandID(nil), source.childrenByCommand[id]...)
		slices.SortStableFunc(children, func(left, right CommandID) int { return cmp.Compare(name(left), name(right)) })
		for _, child := range children {
			visit(child)
		}
	}
	visit(source.root)
	return out
}

func (p *referenceProjector) helpForCommand(commandID CommandID) HelpSchema {
	return projectReferenceCommandHelp(p.source, p.localizationForCommand(commandID), p.locale, commandID, p.visibleCommand, p.commandEntryVisible, p.flagVisible)
}

func (p *referenceProjector) commandOrder() []CommandID {
	return referenceCommandOrder(p.source, p.visibleCommand, p.options.Order)
}

func (p *referenceProjector) commandEntryVisible(id CommandEntryID) bool {
	state, err := staticReferenceCommandEntryProjection(
		p.model,
		p.source,
		p.scheme,
		p.release,
		p.options.Visibility,
		id,
	)
	return err == nil && state.visible
}

func filterFlagIDs(in []FlagID, keep func(FlagID) bool) []FlagID {
	var out []FlagID
	for _, x := range in {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

func filterFlagSpellingIDs(in []FlagSpellingID, keep func(FlagSpellingID) bool) []FlagSpellingID {
	var out []FlagSpellingID
	for _, x := range in {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

func filterArgumentTermIDs(in []ArgumentTermID, keep func(ArgumentTermID) bool) []ArgumentTermID {
	var out []ArgumentTermID
	for _, x := range in {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

func filterCommandDefaultIDs(in []CommandDefaultID, keep func(CommandDefaultID) bool) []CommandDefaultID {
	var out []CommandDefaultID
	for _, x := range in {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}
