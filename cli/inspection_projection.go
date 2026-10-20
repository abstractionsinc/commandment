package cli

import (
	"cmp"
	"errors"
	"slices"
)

func projectInspectionDynamicDomainEntry(result *Inspection, operation *invocation, model *effectiveModel, owner CommandID, locale Locale, entry domainEntryRecord) (ValueDomainEntryInspection, bool) {
	state, err := projectEffectiveLifecycleState(model, entry.lifecycle)
	if err != nil {
		panic("cli: accepted dynamic domain entry has invalid lifecycle")
	}
	available := referenceAvailable(state)
	origin := DeclarationOriginApplication
	if command, ok := result.Commands[CommandID(owner.uint32())]; ok && command.Origin == DeclarationOriginProvider {
		origin = DeclarationOriginProvider
	}
	projected := ValueDomainEntryInspection{
		ID:             ValueDomainEntryID(entry.meta.id.uint32()),
		Kind:           valueDomainEntryKind(entry.kind),
		Origin:         origin,
		Name:           entry.name,
		Canonical:      ValueDomainEntryID(entry.canonical.uint32()),
		Description:    Description{Short: referenceText(operation.LocalizationForCommand(owner), string(locale), entry.description)},
		Hidden:         entry.hidden,
		Available:      available,
		Visible:        available && !entry.hidden,
		Release:        entry.lifecycle,
		LifecycleState: state,
	}
	if replacement, ok := entry.Replacement(); ok {
		projected.Replacement = inspectionDynamicDomainReplacement(model, entry, replacement)
	}
	return projected, available
}

func inspectionDynamicDomainReplacement(model *effectiveModel, source domainEntryRecord, replacement ValueDomainEntryID) *DeclarationReplacement {
	target, ok := model.DomainEntry(replacement)
	if !ok {
		panic("cli: accepted dynamic domain replacement target is absent")
	}
	presentation := target.name
	if target.kind == domainEntryKindAlias {
		canonical, ok := model.DomainEntry(target.canonical)
		if !ok || canonical.kind != domainEntryKindCanonical {
			panic("cli: accepted dynamic domain replacement canonical target is absent")
		}
		presentation = canonical.name
	}
	return &DeclarationReplacement{
		Target: DeclarationIdentity{
			Kind:             DeclarationKindValueDomainEntry,
			ValueDomainEntry: ValueDomainEntryID(replacement.uint32()),
		},
		Presentation: presentation,
		TargetDeprecatedDuringMigration: targetDeprecatedDuringMigration(
			model.Application().release,
			source.lifecycle,
			target.lifecycle,
		),
	}
}

func inspectionFromReference(reference referenceProjection) Inspection {
	result := Inspection{
		CommandOrder:       append([]CommandID(nil), reference.CommandOrder...),
		Commands:           make(map[CommandID]CommandInspection, len(reference.Commands)),
		CommandEntries:     make(map[CommandEntryID]CommandEntryInspection, len(reference.CommandEntries)),
		ActionEntries:      make(map[ActionEntryID]ActionEntryInspection, len(reference.ActionEntries)),
		CommandDefaults:    make(map[CommandDefaultID]CommandDefaultInspection, len(reference.CommandDefaults)),
		CommandProviders:   make(map[CommandProviderID]CommandProviderInspection, len(reference.CommandProviders)),
		Flags:              make(map[FlagID]FlagInspection, len(reference.Flags)),
		FlagSpellings:      make(map[FlagSpellingID]FlagSpellingInspection, len(reference.FlagSpellings)),
		ArgumentForms:      make(map[ArgumentFormID]ArgumentFormInspection, len(reference.ArgumentForms)),
		ArgumentTerms:      make(map[ArgumentTermID]ArgumentTermInspection, len(reference.ArgumentTerms)),
		Facts:              make(map[ValueID]FactInspection, len(reference.Facts)),
		EnvironmentInputs:  make(map[EnvironmentInputID]EnvironmentInputInspection, len(reference.EnvironmentInputs)),
		ValueDomains:       make([]ValueDomainInspection, len(reference.ValueDomains)),
		ValueDomainEntries: make(map[ValueDomainEntryID]ValueDomainEntryInspection, len(reference.ValueDomainEntries)),
		Values:             make(map[ValueID]ValueInspection),
		ValueCandidates:    make(map[ValueCandidateID]ValueCandidateInspection),
		SignalPolicies:     make(map[SignalPolicyID]SignalPolicyInspection),
		SignalEndpoints:    make(map[SignalEndpointID]SignalEndpointInspection),
	}
	for id, value := range reference.Commands {
		available := id == reference.Root || referenceAvailable(value.LifecycleState)
		conditions := make([]ConditionInspection, len(value.Conditions))
		for i, condition := range value.Conditions {
			conditions[i] = ConditionInspection{
				Outcome:      EvaluationOutcomeUnknown,
				Target:       condition.Target,
				Effect:       condition.Effect,
				Expression:   cloneConditionExpressionValue(condition.Expression),
				Reachability: condition.Reachability,
				Sensitive:    condition.Expression.Sensitive,
				Profile:      condition.Profile,
			}
		}
		rules := make([]RuleInspection, len(value.Rules))
		for i, rule := range value.Rules {
			rules[i] = RuleInspection{
				ID:                rule.ID,
				DeclarationIndex:  rule.DeclarationIndex,
				Profile:           rule.Profile,
				Kind:              rule.Kind,
				Code:              rule.Code,
				Outcome:           EvaluationOutcomeUnknown,
				Subjects:          append([]ValueID(nil), rule.Subjects...),
				Dependencies:      append([]ValueID(nil), rule.Dependencies...),
				Operands:          cloneRuleOperands(rule.Operands),
				Pattern:           rule.Pattern,
				Count:             rule.Count,
				Activation:        cloneConditionExpression(rule.Activation),
				ActivationOutcome: EvaluationOutcomeUnknown,
				Reachability:      rule.Reachability,
				Sensitive:         rule.Sensitive,
			}
		}
		advisories := make([]AdvisoryInspection, len(value.Advisories))
		for i, advisory := range value.Advisories {
			advisories[i] = AdvisoryInspection{
				Advisory:          advisory.Advisory,
				Command:           advisory.Command,
				Origin:            advisory.Origin,
				DeclarationIndex:  advisory.DeclarationIndex,
				Profile:           advisory.Profile,
				Kind:              advisory.Kind,
				Outcome:           AdvisoryOutcomeNotEvaluated,
				Activation:        cloneConditionExpression(advisory.Activation),
				ActivationOutcome: EvaluationOutcomeUnknown,
				Reachability:      advisory.Reachability,
				Subjects:          append([]ValueID(nil), advisory.Subjects...),
				Dependencies:      append([]ValueID(nil), advisory.Dependencies...),
			}
		}
		suppressions := make([]NoticeSuppressionInspection, len(value.NoticeSuppressions))
		for i, suppression := range value.NoticeSuppressions {
			suppressions[i] = NoticeSuppressionInspection{
				Command:          suppression.Command,
				Origin:           suppression.Origin,
				DeclarationIndex: suppression.DeclarationIndex,
				Profile:          suppression.Profile,
				Match:            suppression.Match,
				Condition:        cloneConditionExpressionValue(suppression.Condition),
				ConditionOutcome: EvaluationOutcomeUnknown,
				Reachability:     suppression.Reachability,
			}
		}
		result.Commands[id] = CommandInspection{
			ID:                    value.ID,
			CanonicalEntry:        value.CanonicalEntry,
			Parent:                value.Parent,
			Children:              append([]CommandID(nil), value.Children...),
			NamespaceCompleteness: namespaceStateForReference(value),
			Origin:                value.Origin,
			Name:                  value.Name,
			Path:                  append([]string(nil), value.Path...),
			Description:           value.Description,
			Examples:              append([]Example(nil), value.Examples...),
			Hidden:                value.Hidden,
			Available:             available,
			Visible:               available && !value.Hidden,
			Runnable:              value.Runnable,
			Access:                cloneExecutionAccess(value.Access),
			LifecycleState:        value.LifecycleState,
			Dialect:               cloneCommandDialect(value.Dialect),
			CommandEntries:        append([]CommandEntryID(nil), value.CommandEntries...),
			ActionEntries:         append([]ActionEntryID(nil), value.ActionEntries...),
			CommandDefaults:       append([]CommandDefaultID(nil), value.CommandDefaults...),
			CommandProviders:      append([]CommandProviderID(nil), value.CommandProviders...),
			Flags:                 append([]FlagID(nil), value.Flags...),
			Facts:                 append([]ValueID(nil), value.Facts...),
			ArgumentForms:         append([]ArgumentFormID(nil), value.ArgumentForms...),
			Values:                append([]ValueID(nil), value.Values...),
			Conditions:            conditions,
			Rules:                 rules,
			Advisories:            advisories,
			NoticeSuppressions:    suppressions,
			SignalPolicies:        append([]SignalPolicyID(nil), value.SignalPolicies...),
		}
		if value.Profiles != nil {
			command := result.Commands[id]
			command.ProfileSelector = value.Profiles.Selector
			result.Commands[id] = command
		}
	}
	for id, value := range reference.CommandEntries {
		defaults := make([]CommandShortcutDefaultInspection, len(value.Defaults))
		for i, item := range value.Defaults {
			defaults[i] = CommandShortcutDefaultInspection{
				Destination: item.Destination,
				Spelling:    item.Spelling,
				Values:      append([]ValueLiteral(nil), item.Values...),
				Redacted:    item.Sensitive,
				Active:      value.Available,
			}
		}
		result.CommandEntries[id] = CommandEntryInspection{
			ID:               value.ID,
			Parent:           value.Parent,
			Kind:             value.Kind,
			Origin:           value.Origin,
			DeclarationIndex: value.DeclarationIndex,
			Profile:          value.Profile,
			Name:             value.Name,
			TargetPath:       append([]CommandID(nil), value.TargetPath...),
			Description:      value.Description,
			Examples:         append([]Example(nil), value.Examples...),
			Args:             append([]string(nil), value.Args...),
			AppendArgs:       append([]string(nil), value.AppendArgs...),
			Defaults:         defaults,
			Hidden:           value.Hidden,
			Available:        value.Available,
			Visible:          value.Visible,
			Release:          value.Release,
			LifecycleState:   value.LifecycleState,
			Replacement:      cloneReplacement(value.Replacement),
		}
	}
	for id, value := range reference.ActionEntries {
		result.ActionEntries[id] = ActionEntryInspection{
			ID:               value.ID,
			Parent:           value.Parent,
			Kind:             value.Kind,
			Action:           value.Action,
			Origin:           value.Origin,
			DeclarationIndex: value.DeclarationIndex,
			Long:             value.Long,
			Short:            value.Short,
			Literal:          value.Literal,
			Name:             value.Name,
			Placement:        value.Placement,
			Description:      value.Description,
			Hidden:           value.Hidden,
			Available:        value.Available,
			Visible:          value.Visible,
			Release:          value.Release,
			LifecycleState:   value.LifecycleState,
			Replacement:      cloneReplacement(value.Replacement),
		}
	}
	for id, value := range reference.CommandDefaults {
		result.CommandDefaults[id] = CommandDefaultInspection{
			ID:               value.ID,
			Parent:           value.Parent,
			Target:           value.Target,
			Origin:           value.Origin,
			DeclarationIndex: value.DeclarationIndex,
			Profile:          value.Profile,
			Condition:        cloneConditionExpression(value.Condition),
			ConditionOutcome: EvaluationOutcomeUnknown,
			Reachability:     value.Reachability,
		}
	}
	for id, value := range reference.CommandProviders {
		result.CommandProviders[id] = CommandProviderInspection{
			ID:               value.ID,
			Parent:           value.Parent,
			DeclarationIndex: value.DeclarationIndex,
			Profile:          value.Profile,
			Dependencies:     append([]ValueID(nil), value.Dependencies...),
			Outcome:          CommandProviderOutcomeNotNeeded,
		}
	}
	for id, value := range reference.Flags {
		result.Flags[id] = FlagInspection{
			ID:                  value.ID,
			Command:             value.Command,
			Origin:              value.Origin,
			Profile:             value.Profile,
			Destination:         value.Destination,
			Spellings:           append([]FlagSpellingID(nil), value.Spellings...),
			ValueName:           value.ValueName,
			Description:         value.Description,
			Type:                value.Type,
			Occurrences:         value.Occurrences,
			ValuesPerOccurrence: value.ValuesPerOccurrence,
			Placement:           value.Placement,
			Required:            value.Required,
			Hidden:              value.Hidden,
			Sensitive:           value.Sensitive,
			LocaleSelector:      value.LocaleSelector,
			Completion:          value.Completion,
			Available:           value.Available,
			Visible:             value.Visible,
			Release:             value.Release,
			LifecycleState:      value.LifecycleState,
			Replacement:         cloneReplacement(value.Replacement),
		}
	}
	for id, value := range reference.FlagSpellings {
		flag := reference.Flags[value.Flag]
		result.FlagSpellings[id] = FlagSpellingInspection{
			ID:             value.ID,
			Flag:           value.Flag,
			Origin:         value.Origin,
			Spelling:       value.Spelling,
			Hidden:         value.Hidden,
			Available:      value.Available,
			Visible:        value.Visible && !flag.Hidden && !value.Hidden,
			Release:        value.Release,
			LifecycleState: value.LifecycleState,
			Replacement:    cloneReplacement(value.Replacement),
		}
	}
	for id, value := range reference.ArgumentForms {
		result.ArgumentForms[id] = ArgumentFormInspection{
			ID:             value.ID,
			Command:        value.Command,
			Origin:         value.Origin,
			Profile:        value.Profile,
			Terms:          append([]ArgumentTermID(nil), value.Terms...),
			Outcome:        ArgumentFormOutcomeNoMatch,
			Available:      value.Available,
			Visible:        value.Visible,
			Release:        value.Release,
			LifecycleState: value.LifecycleState,
			Replacement:    cloneReplacement(value.Replacement),
			Reachability:   value.Reachability,
		}
	}
	for id, value := range reference.ArgumentTerms {
		result.ArgumentTerms[id] = ArgumentTermInspection{
			ID:              value.ID,
			Form:            value.Form,
			Origin:          value.Origin,
			Kind:            value.Kind,
			Destination:     value.Destination,
			Name:            value.Name,
			Literal:         value.Literal,
			Description:     value.Description,
			Type:            value.Type,
			Count:           value.Count,
			Optional:        value.Optional,
			Greedy:          value.Greedy,
			Minimal:         value.Minimal,
			EndOfOptions:    value.EndOfOptions,
			Hidden:          value.Hidden,
			Sensitive:       value.Sensitive,
			SourceSensitive: value.SourceSensitive,
			HyphenValues:    value.HyphenValues,
			TokenLanguage:   cloneTokenLanguageSyntax(value.TokenLanguage),
			Completion:      value.Completion,
			Available:       value.Available,
			Visible:         value.Visible,
			Release:         value.Release,
			LifecycleState:  value.LifecycleState,
			Replacement:     cloneReplacement(value.Replacement),
		}
	}
	for id, value := range reference.Facts {
		result.Facts[id] = FactInspection{
			ID:                value.ID,
			HasResolver:       value.HasResolver,
			ResolverSensitive: value.ResolverSensitive,
			Dependencies:      append([]ValueID(nil), value.Dependencies...),
		}
	}
	for id, value := range reference.EnvironmentInputs {
		result.EnvironmentInputs[id] = EnvironmentInputInspection(value)
	}
	for index, value := range reference.ValueDomains {
		result.ValueDomains[index] = ValueDomainInspection{
			Uses:    append([]ValueDomainUse(nil), value.Uses...),
			Entries: append([]ValueDomainEntryID(nil), value.Entries...),
			Outcome: ValueDomainOutcomeReady,
		}
	}
	for id, value := range reference.ValueDomainEntries {
		result.ValueDomainEntries[id] = ValueDomainEntryInspection{
			ID:             value.ID,
			Kind:           value.Kind,
			Origin:         value.Origin,
			Name:           value.Name,
			Canonical:      value.Canonical,
			Description:    value.Description,
			Hidden:         value.Hidden,
			Available:      value.Available,
			Visible:        value.Visible && !value.Hidden,
			Release:        value.Release,
			LifecycleState: value.LifecycleState,
			Replacement:    cloneReplacement(value.Replacement),
		}
	}
	for id, value := range reference.SignalPolicies {
		result.SignalPolicies[id] = signalPolicyInspectionFromReference(value)
	}
	for id, value := range reference.SignalEndpoints {
		result.SignalEndpoints[id] = SignalEndpointInspection{
			SignalEndpoint: value,
			Status:         SignalEndpointStatusInactive,
		}
	}
	return result
}

func projectGrammarSelection(selection commandSelection) CommandSelection {
	result := CommandSelection{
		Root:     selection.Root,
		Selected: selection.Selected,
		Path:     slices.Clone(selection.Path),
		Steps:    make([]CommandSelectionStep, len(selection.Steps)),
	}
	for i, step := range selection.Steps {
		commands := slices.Clone(step.Commands)
		projected := CommandSelectionStep{
			Input:    step.Input,
			Commands: commands,
		}
		if step.Default.present() {
			projected.Kind = CommandSelectionStepDefault
			projected.Default = step.Default
		} else {
			projected.Kind = CommandSelectionStepEntry
			projected.Entry = step.Entry
		}
		result.Steps[i] = projected
	}
	return result
}

func projectGrammarActionSelection(grammar grammarResult, reference referenceProjection) InvocationActionSelection {
	if !grammar.Selection.ActionSelected() {
		return InvocationActionSelection{
			Selected: InvocationActionRun,
		}
	}
	if !grammar.Selection.Action.present() {
		trigger := InvocationActionTriggerUnknown
		switch grammar.Selection.ActionTrigger {
		case actionTriggerEmpty:
			trigger = InvocationActionTriggerEmpty
		case actionTriggerStructural:
			trigger = InvocationActionTriggerStructural
		case actionTriggerNone:
		}
		return InvocationActionSelection{
			Selected: InvocationActionHelp,
			Trigger:  trigger,
		}
	}
	entryID := grammar.Selection.Action
	entry, ok := reference.ActionEntries[entryID]
	if !ok {
		return InvocationActionSelection{}
	}
	trigger := InvocationActionTriggerEntry
	var input InputOriginID
	for _, token := range grammar.Tokens {
		if token.Ownership.ActionEntry == grammar.Selection.Action {
			input = token.Origin
			if entry.Kind == ActionEntryCommand {
				trigger = InvocationActionTriggerCommand
			}
			break
		}
	}
	invocationActionSelection := InvocationActionSelection{
		Selected: entry.Action,
		Trigger:  trigger,
		Entry:    entryID,
		Input:    input,
	}

	return invocationActionSelection
}

func projectOperationSources(operation *invocation) map[InputSourceID]InputSource {
	records := operation.Sources()
	result := make(map[InputSourceID]InputSource, len(records))
	for _, source := range records {
		id := source.id
		result[id] = InputSource{
			ID:   id,
			Kind: source.kind,
			Revision: InputRevision{
				value: source.revision,
			},
			Key:             source.key,
			Location:        source.location,
			Sensitive:       source.sensitive,
			CommandShortcut: source.commandShortcut,
		}
	}
	return result
}

func projectOperationOrigins(operation *invocation) map[InputOriginID]InputOrigin {
	records := operation.Origins()
	result := make(map[InputOriginID]InputOrigin, len(records))
	for _, origin := range records {
		span := origin.span
		parents := make([]InputOriginID, len(origin.Parents()))
		for i, parent := range origin.Parents() {
			parents[i] = parent
		}
		id := origin.id
		result[id] = InputOrigin{
			ID: id,
			Span: InputSpan{
				Source: span.Source,
				Entry:  span.Entry,
				Start: InputPosition{
					Offset: span.Start,
				},
				End: InputPosition{
					Offset: span.End,
				},
			},
			Parents:   parents,
			Sensitive: origin.sensitive,
		}
	}
	return result
}

func projectGrammarTokens(operation *invocation, grammar grammarResult, reference referenceProjection) []TokenInspection {
	includes := operation.ResponseIncludes()
	slices.SortStableFunc(includes, func(left, right responseIncludeObservation) int {
		return cmp.Compare(left.BeforeArgument, right.BeforeArgument)
	})
	result := make([]TokenInspection, 0, len(grammar.Tokens)+len(includes))
	includeIndex := 0
	for argument := 0; argument <= len(grammar.Tokens); argument++ {
		for includeIndex < len(includes) && includes[includeIndex].BeforeArgument == argument {
			include := includes[includeIndex]
			tokenInspectionValue := TokenInspection{
				Text:           include.Spelling,
				Origin:         include.Origin.ID(),
				Role:           TokenRoleResponseFile,
				DelegatedIndex: -1,
			}
			result = append(result, tokenInspectionValue)
			includeIndex++
		}
		if argument == len(grammar.Tokens) {
			break
		}
		result = append(result, projectGrammarToken(grammar.Tokens[argument], reference))
	}
	return result
}

func projectGrammarToken(token tokenObservation, reference referenceProjection) TokenInspection {
	ownership := token.Ownership
	item := TokenInspection{
		Text:            token.Text,
		Origin:          token.Origin,
		Role:            publicTokenRole(ownership.Role),
		Command:         ownership.Command,
		Destination:     ownership.Destination,
		Match:           publicNameMatch(ownership.Match),
		FlagValueForm:   publicFlagValueForm(ownership.Form),
		OccurrenceIndex: ownership.Occurrence,
		ElementIndex:    ownership.Element,
		DelegatedIndex:  -1,
	}
	if ownership.Flag.present() {
		item.Declaration = DeclarationIdentity{
			Kind: DeclarationKindFlag,
			Flag: ownership.Flag,
		}
	}
	if ownership.FlagSpelling.present() {
		spellingID := ownership.FlagSpelling
		item.Declaration = DeclarationIdentity{
			Kind:         DeclarationKindFlagSpelling,
			FlagSpelling: spellingID,
		}
		if spelling, ok := reference.FlagSpellings[spellingID]; ok {
			item.MatchedSpelling = spelling.Spelling.Text
		}
	}
	if ownership.CommandEntry.present() {
		item.Declaration = DeclarationIdentity{
			Kind:         DeclarationKindCommandEntry,
			CommandEntry: ownership.CommandEntry,
		}
	}
	if ownership.ActionEntry.present() {
		item.Declaration = DeclarationIdentity{
			Kind:        DeclarationKindActionEntry,
			ActionEntry: ownership.ActionEntry,
		}
	}
	if ownership.ArgumentTerm.present() {
		item.Declaration = DeclarationIdentity{
			Kind:         DeclarationKindArgumentTerm,
			ArgumentTerm: ownership.ArgumentTerm,
		}
	}
	return item
}

func applyGrammarPresence(result *Inspection, grammar grammarResult) {
	for _, occurrence := range grammar.Occurrences {
		flag := occurrence.Flag
		if value, ok := result.Flags[flag]; ok {
			value.Present = true
			result.Flags[flag] = value
		}
		spelling := occurrence.Spelling
		if value, ok := result.FlagSpellings[spelling]; ok {
			value.Present = true
			result.FlagSpellings[spelling] = value
		}
	}
	for _, token := range grammar.Tokens {
		term := token.Ownership.ArgumentTerm
		if !term.present() {
			continue
		}
		id := term
		value := result.ArgumentTerms[id]
		value.Present = true
		value.InputOrigins = append(value.InputOrigins, token.Origin)
		result.ArgumentTerms[id] = value
	}
}

func applyDefaultCommandSelections(result *Inspection, grammar grammarResult) {
	for commandIDPrivate, selection := range grammar.DefaultSelections {
		id := commandIDPrivate
		command, ok := result.Commands[id]
		if !ok {
			continue
		}
		command.DefaultSelection = DefaultCommandSelection{
			Outcome:    publicDefaultSelectionOutcome(selection.Status),
			Candidates: slices.Clone(selection.Candidates),
			Applicable: slices.Clone(selection.Applicable),
			Blocked:    slices.Clone(selection.Blocked),
			Selected:   selection.Selected,
		}
		result.Commands[id] = command
	}
	for defaultIDPrivate, outcome := range grammar.DefaultConditions {
		id := defaultIDPrivate
		item, ok := result.CommandDefaults[id]
		if !ok {
			continue
		}
		item.ConditionOutcome = publicConditionOutcome(outcome)
		result.CommandDefaults[id] = item
	}
}

func publicDefaultSelectionOutcome(status defaultSelectionStatus) DefaultCommandSelectionOutcome {
	switch status {
	case defaultSelectionNotEvaluated:
		return DefaultCommandSelectionNotEvaluated
	case defaultSelectionNone:
		return DefaultCommandSelectionNone
	case defaultSelectionSelected:
		return DefaultCommandSelectionSelected
	case defaultSelectionAmbiguous:
		return DefaultCommandSelectionAmbiguous
	case defaultSelectionBlocked:
		return DefaultCommandSelectionBlocked
	default:
		return DefaultCommandSelectionUnknown
	}
}

func publicConditionOutcome(outcome conditionOutcome) EvaluationOutcome {
	switch outcome {
	case conditionOutcomeSatisfied:
		return EvaluationOutcomeSatisfied
	case conditionOutcomeUnsatisfied:
		return EvaluationOutcomeUnsatisfied
	case conditionOutcomeBlocked, conditionOutcomeUnsupported:
		return EvaluationOutcomeFailed
	default:
		return EvaluationOutcomeUnknown
	}
}

func applyArgumentFormSelection(result *Inspection, grammar grammarResult) {
	selected := grammar.Arguments.Form
	selectedID := selected
	selectedCommandID := grammar.Selection.Selected
	command := result.Commands[selectedCommandID]
	if len(command.ArgumentForms) == 0 {
		command.ArgumentFormSelection = ArgumentFormSelection{
			Outcome: ArgumentFormSelectionNoForms,
		}
		result.Commands[selectedCommandID] = command
		return
	}

	var accepted, rejected, failed []ArgumentFormID
	for _, evaluation := range grammar.ArgumentForms {
		id := evaluation.Form
		observation, ok := result.ArgumentForms[id]
		if !ok {
			continue
		}
		switch evaluation.Status {
		case argumentFormEvaluationNoMatch:
			observation.Outcome = ArgumentFormOutcomeNoMatch
		case argumentFormEvaluationRejected, argumentFormEvaluationAmbiguousAllocation:
			observation.Outcome = ArgumentFormOutcomeRejected
			rejected = append(rejected, evaluation.Form)
		case argumentFormEvaluationAccepted:
			observation.Outcome = ArgumentFormOutcomeMatched
			accepted = append(accepted, evaluation.Form)
		case argumentFormEvaluationFailed:
			observation.Outcome = ArgumentFormOutcomeFailed
			failed = append(failed, evaluation.Form)
		case argumentFormEvaluationUnknown:
		}
		result.ArgumentForms[id] = observation
	}
	if selected.present() {
		setArgumentFormOutcomes(result, []ArgumentFormID{selected}, ArgumentFormOutcomeSelected)
		command.ArgumentFormSelection = ArgumentFormSelection{
			Outcome:  ArgumentFormSelectionSelected,
			Selected: selectedID,
			Matches:  []ArgumentFormID{selectedID},
		}
		result.Commands[selectedCommandID] = command
		return
	}

	for _, issue := range grammar.Issues {
		if issue.Command.present() && issue.Command != grammar.Selection.Selected {
			continue
		}
		switch issue.Kind {
		case grammarIssueArgumentFormFailure:
			if len(failed) == 0 {
				failed = append(failed, issue.Forms...)
			}
			command.ArgumentFormSelection = ArgumentFormSelection{
				Outcome: ArgumentFormSelectionFailed,
				Matches: slices.Clone(accepted),
				Failed:  slices.Clone(failed),
			}
			result.Commands[selectedCommandID] = command
			return
		case grammarIssueAmbiguousArgumentForm:
			if len(accepted) == 0 {
				accepted = append(accepted, issue.Forms...)
				setArgumentFormOutcomes(result, issue.Forms, ArgumentFormOutcomeMatched)
			}
			command.ArgumentFormSelection = ArgumentFormSelection{
				Outcome: ArgumentFormSelectionAmbiguous,
				Matches: slices.Clone(accepted),
			}
			result.Commands[selectedCommandID] = command
			return
		case grammarIssueRejectedArgumentForm, grammarIssueAmbiguousFormAllocation:
			if len(rejected) == 0 {
				rejected = append(rejected, issue.Forms...)
			}
			command.ArgumentFormSelection = ArgumentFormSelection{
				Outcome: ArgumentFormSelectionRejected,
				Matches: slices.Clone(rejected),
			}
			result.Commands[selectedCommandID] = command
			return
		case grammarIssueNoArgumentForm:
			command.ArgumentFormSelection = ArgumentFormSelection{
				Outcome: ArgumentFormSelectionNoMatch,
			}
			result.Commands[selectedCommandID] = command
			return
		case grammarIssueNone,
			grammarIssueMissingDialect,
			grammarIssueAmbiguousName,
			grammarIssueBlockedExplicitEntry,
			grammarIssueAmbiguousFlag,
			grammarIssueMisplacedFlag,
			grammarIssueUnknownFlag,
			grammarIssueMissingFlagValue,
			grammarIssuePrematureFlagTerminator,
			grammarIssueTooManyFlagOccurrences,
			grammarIssueMissingFlagOccurrence,
			grammarIssueAmbiguousOwnership,
			grammarIssueUnexpectedInput,
			grammarIssueProfileSelection,
			grammarIssueUnavailableCommand,
			grammarIssueUnavailableFlag,
			grammarIssueUnavailableArgument,
			grammarIssueNotIntroducedCommand,
			grammarIssueNotIntroducedFlag,
			grammarIssueNotIntroducedArgument,
			grammarIssueRemovedCommand,
			grammarIssueRemovedFlag,
			grammarIssueRemovedArgument,
			grammarIssueConditionEvaluation,
			grammarIssueAmbiguousDefault,
			grammarIssueBlockedDefault:
		}
	}

	command.ArgumentFormSelection = ArgumentFormSelection{
		Outcome: ArgumentFormSelectionNotEvaluated,
	}
	result.Commands[selectedCommandID] = command
}

func setArgumentFormOutcomes(result *Inspection, forms []ArgumentFormID, status ArgumentFormOutcome) {
	for _, form := range forms {
		id := form
		if observation, ok := result.ArgumentForms[id]; ok {
			observation.Outcome = status
			result.ArgumentForms[id] = observation
		}
	}
}

func inspectionPostValueComplete(model *effectiveModel, grammar grammarResult) bool {
	if len(grammar.Issues) != 0 {
		return true
	}
	for _, command := range grammar.Selection.Path {
		if _, ok := model.Command(command); !ok {
			return false
		}
	}
	return true
}

func projectStaticExecution(result *Inspection, model *effectiveModel, selection commandSelection) {
	if selection.Action.present() {
		result.ExecutionAdmission.Status = ExecutionAdmissionNotApplicable
		return
	}
	record, ok := model.Command(selection.Selected)
	if !ok {
		return
	}
	execution := record.execution
	if !execution.Runnable() {
		result.ExecutionAdmission.Status = ExecutionAdmissionNotRunnable
		return
	}
	result.ExecutionAdmission.Status = ExecutionAdmissionReady
	result.ExecutionAdmission.HandlerCommand = selection.Selected
	for _, command := range selection.Path {
		current, ok := model.Command(command)
		if !ok {
			continue
		}
		plan := current.execution
		for index := range plan.Middleware() {
			scope := MiddlewareScopeCommand
			if _, hasParent := current.Parent(); !hasParent {
				scope = MiddlewareScopeApplication
			}
			middlewareInspectionValue := MiddlewareInspection{
				Scope:            scope,
				Command:          command,
				DeclarationIndex: index,
			}
			result.ExecutionAdmission.Middleware = append(result.ExecutionAdmission.Middleware, middlewareInspectionValue)
		}
	}
}

func namespaceStateForReference(command CommandReference) CommandNamespaceCompleteness {
	if len(command.CommandProviders) != 0 {
		return CommandNamespaceUnresolved
	}
	return CommandNamespaceComplete
}

func signalPolicyInspectionFromReference(reference SignalPolicyReference) SignalPolicyInspection {
	schema := reference.SignalPolicySchema
	signalPolicyInspection := SignalPolicyInspection{
		ID:                         schema.ID,
		CounterID:                  schema.CounterID,
		Command:                    schema.Command,
		Origin:                     schema.Origin,
		Profile:                    schema.Profile,
		Selector:                   schema.Selector,
		Occurrence:                 schema.Occurrence,
		Aggregation:                schema.Aggregation,
		Deferred:                   schema.Deferred,
		ContinueMatching:           schema.ContinueMatching,
		Hint:                       schema.Hint,
		Actions:                    append([]SignalAction(nil), schema.Actions...),
		RequiredSourceCapabilities: schema.RequiredSourceCapabilities,
		RequiredHostCapabilities:   schema.RequiredHostCapabilities,
		Sensitive:                  schema.Sensitive,
		Activation:                 cloneConditionExpression(schema.Activation),
		ActivationOutcome:          EvaluationOutcomeUnknown,
		Reachability:               schema.Reachability,
		Status:                     SignalPolicyStatusInactive,
	}

	return signalPolicyInspection
}

func publicTokenRole(role tokenRole) TokenRole {
	switch role {
	case tokenRoleCommand:
		return TokenRoleCommand
	case tokenRoleAction:
		return TokenRoleAction
	case tokenRoleFlag:
		return TokenRoleFlag
	case tokenRoleFlagValue:
		return TokenRoleFlagValue
	case tokenRolePositional:
		return TokenRolePositional
	case tokenRoleLiteral:
		return TokenRoleLiteral
	case tokenRoleTokenLanguage:
		return TokenRoleTokenLanguage
	case tokenRoleEndOfOptions:
		return TokenRoleEndOfOptions
	case tokenRoleResponseFile:
		return TokenRoleResponseFile
	case tokenRoleRemainder:
		return TokenRoleRemainder
	case tokenRolePassThrough:
		return TokenRolePassThrough
	default:
		return TokenRoleUnknown
	}
}

func publicFlagValueForm(form flagValueForm) FlagValueForm {
	switch form {
	case flagValueFormSeparate:
		return FlagValueFormSeparate
	case flagValueFormJoined:
		return FlagValueFormJoined
	case flagValueFormAssigned:
		return FlagValueFormAssigned
	default:
		return FlagValueFormUnknown
	}
}

func publicNameMatch(match nameMatchKind) NameMatchKind {
	switch match {
	case nameMatchKindExact:
		return NameMatchKindExact
	case nameMatchKindUniquePrefix:
		return NameMatchKindUniquePrefix
	default:
		return NameMatchKindUnknown
	}
}

func cloneCommandDialect(input *CommandDialect) *CommandDialect {
	if input == nil {
		return nil
	}
	out := *input
	out.EndOfOptions = append([]string(nil), input.EndOfOptions...)
	return &out
}

func cloneReplacement(input *DeclarationReplacement) *DeclarationReplacement {
	if input == nil {
		return nil
	}
	out := *input
	return &out
}

func cloneConditionExpressionValue(input ConditionExpression) ConditionExpression {
	result := input
	result.Operands = append([]ValueLiteral(nil), input.Operands...)
	result.ProducerKinds = append([]ValueProducerKind(nil), input.ProducerKinds...)
	result.Children = make([]ConditionExpression, len(input.Children))
	for i, child := range input.Children {
		result.Children[i] = cloneConditionExpressionValue(child)
	}
	return result
}

func cloneConditionExpression(input *ConditionExpression) *ConditionExpression {
	if input == nil {
		return nil
	}
	out := cloneConditionExpressionValue(*input)
	return &out
}

func cloneTokenLanguageSyntax(input *TokenLanguageSyntax) *TokenLanguageSyntax {
	if input == nil {
		return nil
	}
	out := *input
	out.Fragments = append([]string(nil), input.Fragments...)
	return &out
}

func applyPolicyAnalysis(result *Inspection, operation *invocation, model *effectiveModel, locale Locale, analysis policyAnalysis) ([]Diagnostic, error) {
	if result == nil {
		return nil, nil
	}
	diagnostics := make([]Diagnostic, 0)
	causes := make([]error, 0)

	for _, id := range analysis.RuleOrder {
		record, ok := model.Rule(id)
		if !ok {
			return nil, errInvalidModelTopology
		}
		evaluation, ok := analysis.Rules[id]
		if !ok {
			continue
		}
		commandID := record.command
		command, ok := result.Commands[commandID]
		if !ok {
			continue
		}
		index := ruleInspectionIndex(command.Rules, int(record.meta.position))
		if index < 0 {
			continue
		}
		rule := command.Rules[index]
		rule.ActivationOutcome = policyEvaluationOutcome(evaluation.Activation)
		rule.Outcome = ruleEvaluationOutcome(evaluation)
		start := len(diagnostics)
		definition := record.definition
		committed := analysis.RuleDiagnosticCounts[record.meta.id]
		switch evaluation.Outcome.Kind {
		case ruleOutcomeViolated:
			for i, violation := range evaluation.Outcome.Violations {
				if i >= committed {
					break
				}
				diagnostic, err := policyRuleDiagnostic(model, record.meta.id, definition, violation)
				if err != nil {
					return nil, err
				}
				diagnostics = append(diagnostics, diagnostic)
			}
		case ruleOutcomeEvaluationFailed:
			if committed != 0 {
				diagnostic, err := policyRuleFailureDiagnostic(
					model,
					record.meta.id,
					DiagnosticRuleEvaluationFailure,
					definition,
				)
				if err != nil {
					return nil, err
				}
				diagnostics = append(diagnostics, diagnostic)
			}
			if evaluation.Outcome.Err != nil {
				causes = append(causes, evaluation.Outcome.Err)
			}
		case ruleOutcomeInvalidResult:
			if committed != 0 {
				diagnostic, err := policyRuleFailureDiagnostic(
					model,
					record.meta.id,
					DiagnosticInvalidRuleResult,
					definition,
				)
				if err != nil {
					return nil, err
				}
				diagnostics = append(diagnostics, diagnostic)
			}
			if evaluation.Outcome.Err != nil {
				causes = append(causes, evaluation.Outcome.Err)
			}
		case ruleOutcomeInvalid, ruleOutcomeSatisfied, ruleOutcomeUpstreamBlocked:
		}
		for i := start; i < len(diagnostics); i++ {
			diagnostics[i].Command = commandID
			rule.DiagnosticIndexes = append(rule.DiagnosticIndexes, len(result.Diagnostics)+i)
		}
		command.Rules[index] = rule
		result.Commands[commandID] = command
	}

	for _, id := range analysis.AdvisoryOrder {
		record, ok := model.Advisory(id)
		if !ok {
			return nil, errInvalidModelTopology
		}
		evaluation, ok := analysis.Advisories[id]
		if !ok {
			continue
		}
		commandID := record.command
		command, ok := result.Commands[commandID]
		if !ok {
			continue
		}
		index := advisoryInspectionIndex(command.Advisories, int(record.meta.position))
		if index < 0 {
			continue
		}
		advisory := command.Advisories[index]
		advisory.ActivationOutcome = policyEvaluationOutcome(evaluation.Activation)
		advisory.Outcome = advisoryEvaluationOutcome(evaluation.Status)
		if evaluation.Status == advisoryEvaluationFailed {
			if _, ok := analysis.AdvisoryDiagnostics[record.meta.id]; ok {
				diagnosticValue := Diagnostic{
					Code:         DiagnosticAdvisoryEvaluationFailure,
					AdvisoryCode: AdvisoryCode(record.definition.Code()),
					Command:      commandID,
				}
				diagnostics = append(diagnostics, diagnosticValue)
			}
			if evaluation.Err != nil {
				causes = append(causes, evaluation.Err)
			}
		}
		command.Advisories[index] = advisory
		result.Commands[commandID] = command
	}

	for _, record := range model.NoticeSuppressions() {
		key := noticeSuppressionKey{
			Command:  record.command,
			Position: record.position,
		}
		evaluation, ok := analysis.Suppressions[key]
		if !ok {
			continue
		}
		commandID := record.command
		command, ok := result.Commands[commandID]
		if !ok {
			continue
		}
		index := suppressionInspectionIndex(command.NoticeSuppressions, int(record.position))
		if index < 0 {
			continue
		}
		suppression := command.NoticeSuppressions[index]
		suppression.ConditionOutcome = policyEvaluationOutcome(evaluation.Condition)
		command.NoticeSuppressions[index] = suppression
		result.Commands[commandID] = command
	}

	result.Notices = make([]NoticeInspection, len(analysis.Notices))
	for i, notice := range analysis.Notices {
		record, ok := model.Advisory(notice.AdvisoryID)
		if !ok {
			continue
		}
		command := record.command
		path := append([]string(nil), result.Commands[command].Path...)
		public := NoticeInspection{
			Notice: projectAdvisoryNotice(operation, locale, record.command, notice, command, path),
		}
		if len(notice.SuppressedBy) == 0 {
			public.Suppression = NoticeSuppressionNotSuppressed
		} else {
			public.Suppression = NoticeSuppressionSuppressed
			public.SuppressedBy = make([]NoticeSuppressionLocator, len(notice.SuppressedBy))
			for j, item := range notice.SuppressedBy {
				public.SuppressedBy[j] = NoticeSuppressionLocator{
					Command:          item.Command,
					DeclarationIndex: int(item.Position),
				}
			}
		}
		result.Notices[i] = public
	}
	return diagnostics, errors.Join(causes...)
}

func ruleInspectionIndex(values []RuleInspection, declarationIndex int) int {
	for i := range values {
		if values[i].DeclarationIndex == declarationIndex {
			return i
		}
	}
	return -1
}

func advisoryInspectionIndex(values []AdvisoryInspection, declarationIndex int) int {
	for i := range values {
		if values[i].DeclarationIndex == declarationIndex {
			return i
		}
	}
	return -1
}

func suppressionInspectionIndex(values []NoticeSuppressionInspection, declarationIndex int) int {
	for i := range values {
		if values[i].DeclarationIndex == declarationIndex {
			return i
		}
	}
	return -1
}

func policyEvaluationOutcome(outcome conditionOutcome) EvaluationOutcome {
	switch outcome {
	case conditionOutcomeSatisfied:
		return EvaluationOutcomeSatisfied
	case conditionOutcomeUnsatisfied:
		return EvaluationOutcomeUnsatisfied
	case conditionOutcomeBlocked:
		return EvaluationOutcomeFailed
	default:
		return EvaluationOutcomeUnknown
	}
}

func ruleEvaluationOutcome(evaluation ruleEvaluation) EvaluationOutcome {
	if evaluation.Skipped {
		return EvaluationOutcomeSkipped
	}
	switch evaluation.Outcome.Kind {
	case ruleOutcomeSatisfied:
		return EvaluationOutcomeSatisfied
	case ruleOutcomeViolated:
		return EvaluationOutcomeUnsatisfied
	case ruleOutcomeEvaluationFailed, ruleOutcomeInvalidResult:
		return EvaluationOutcomeFailed
	case ruleOutcomeUpstreamBlocked:
		return EvaluationOutcomeSkipped
	default:
		return EvaluationOutcomeUnknown
	}
}

func advisoryEvaluationOutcome(status analysisAdvisoryEvaluationOutcome) AdvisoryOutcome {
	switch status {
	case advisoryEvaluationNotMatched:
		return AdvisoryOutcomeNotMatched
	case advisoryEvaluationMatched:
		return AdvisoryOutcomeMatched
	case advisoryEvaluationBlocked:
		return AdvisoryOutcomeBlocked
	case advisoryEvaluationFailed:
		return AdvisoryOutcomeFailed
	default:
		return AdvisoryOutcomeNotEvaluated
	}
}

// applySignalInspection projects the frozen non-subscribing Signal plan. The
// source description is used only to resolve concrete selector membership and
// compatibility; Inspection never subscribes, binds endpoints, starts timers,
// or executes application Signal callbacks.
func applySignalInspection(result *Inspection, operation *inspectionOperation, sources []compiledSignalSource, model *effectiveModel, selection commandSelection, selectedProfiles map[CommandID]GrammarProfileID, values valueAnalysis) ([]Diagnostic, bool, error) {
	if result == nil {
		return nil, false, nil
	}
	plan, ok := buildSignalPlan(model, selection, selectedProfiles, values)
	if !ok {
		return nil, false, nil
	}
	activePolicies := plan.activePolicyIDs()
	for _, policy := range plan.policies {
		publicID := policy.record.meta.id
		observation := result.SignalPolicies[publicID]
		observation.ActivationOutcome = policyEvaluationOutcome(policy.activation)
		if policy.active {
			observation.Status = SignalPolicyStatusReady
		} else {
			observation.Status = SignalPolicyStatusInactive
		}
		result.SignalPolicies[publicID] = observation
	}

	result.SignalSource.Required = len(activePolicies) != 0
	result.SignalSource.RequiredSourceCapabilities = plan.requiredSource
	result.SignalSource.RequiredHostCapabilities = plan.requiredHost
	availableHost := operation.signalCapabilities
	result.SignalSource.AvailableHostCapabilities = availableHost
	result.ExecutionAdmission.SignalPolicies = slices.Clone(activePolicies)
	if len(activePolicies) == 0 {
		result.SignalSource.Status = SignalIntegrationStatusNotRequired
		return nil, true, nil
	}
	if len(sources) == 0 {
		result.SignalSource.Status = SignalIntegrationStatusMissing
		result.SignalSource.MissingSignals = projectSemanticSignals(plan.requiredSignals)
		result.SignalSource.MissingSourceCapabilities = plan.requiredSource
		result.SignalSource.MissingHostCapabilities = plan.requiredHost &^ availableHost
		markActiveSignalPoliciesBlocked(result, activePolicies)
		return []Diagnostic{{Code: DiagnosticIncompatibleSignalSource}}, true, nil
	}

	result.SignalSource.Supplied = true
	description := operation.invocation.signalDescription
	result.SignalSource.Described = true
	resolved, err := resolveSignalPlan(description, plan, availableHost)
	if err != nil {
		result.SignalSource.Status = SignalIntegrationStatusFailed
		markActiveSignalPoliciesBlocked(result, activePolicies)
		return []Diagnostic{{Code: DiagnosticSignalSourceFailure}}, true, err
	}
	result.SignalSource.ResolvedSignals = projectSemanticSignals(resolved.resolvedSignals)
	result.SignalSource.MissingSignals = projectSemanticSignals(resolved.compatibility.missingSignals)
	result.SignalSource.MissingSourceCapabilities = resolved.compatibility.missingSourceCapabilities
	result.SignalSource.AvailableHostCapabilities = resolved.compatibility.availableHostCapabilities
	result.SignalSource.MissingHostCapabilities = resolved.compatibility.missingHostCapabilities
	projectResolvedSignalEndpoints(result, resolved, resolved.compatibility.compatible())
	if !resolved.compatibility.compatible() {
		result.SignalSource.Status = SignalIntegrationStatusIncompatible
		markActiveSignalPoliciesBlocked(result, activePolicies)
		return []Diagnostic{{Code: DiagnosticIncompatibleSignalSource}}, true, nil
	}
	result.SignalSource.Status = SignalIntegrationStatusReady
	return nil, true, nil
}

func projectResolvedSignalEndpoints(result *Inspection, plan resolvedSignalPlan, ready bool) {
	if result == nil {
		return
	}
	result.ExecutionAdmission.SignalEndpoints = make([]SignalEndpointID, 0, len(plan.endpoints))
	for _, endpointID := range plan.endpoints {
		publicID := endpointID
		result.ExecutionAdmission.SignalEndpoints = append(result.ExecutionAdmission.SignalEndpoints, publicID)
		observation := result.SignalEndpoints[publicID]
		if ready {
			observation.Status = SignalEndpointStatusReady
		} else {
			observation.Status = SignalEndpointStatusBlocked
		}
		for _, policy := range plan.policies {
			if !policy.active {
				continue
			}
			for _, action := range policy.definition.Actions() {
				endpoint, present := action.Endpoint()
				if !present || endpoint != endpointID {
					continue
				}
				policyID := policy.record.meta.id
				if ready {
					observation.EffectivePolicies = append(observation.EffectivePolicies, policyID)
				} else {
					observation.BlockedPolicies = append(observation.BlockedPolicies, policyID)
				}
			}
		}
		result.SignalEndpoints[publicID] = observation
	}
}

func effectiveListedSignals(selector signalSelectorDefinition) []Signal {
	excluded := make(map[Signal]struct{}, len(selector.ExcludedSignals()))
	for _, signal := range selector.ExcludedSignals() {
		excluded[signal] = struct{}{}
	}
	result := make([]Signal, 0, len(selector.Signals()))
	for _, signal := range selector.Signals() {
		if _, skip := excluded[signal]; skip {
			continue
		}
		result = append(result, signal)
	}
	return result
}

func availableSignalHostCapabilities(snapshot *runState) SignalHostCapabilities {
	var result SignalHostCapabilities
	if snapshot.abort != nil {
		result |= SignalHostCapabilityAbort
	}
	return result
}

func effectiveSignalPolicies(model *effectiveModel, selection commandSelection, selectedProfiles map[CommandID]GrammarProfileID) ([]signalPolicyRecord, bool) {
	var effective []signalPolicyRecord
	for _, command := range selection.Path {
		type event struct {
			position declarationPosition
			clear    bool
			policy   signalPolicyRecord
		}
		var events []event
		for _, policy := range model.SignalPoliciesForCommand(command) {
			if policy.meta.profile.present() && selectedProfiles[policy.command] != policy.meta.profile {
				continue
			}
			events = append(events, event{
				position: policy.meta.position,
				policy:   policy,
			})
		}
		for _, scope := range model.SignalPolicyScopesForCommand(command) {
			if scope.Profile().present() && selectedProfiles[scope.Command()] != scope.Profile() {
				continue
			}
			if scope.ClearInherited() {
				events = append(events, event{
					position: scope.Position(),
					clear:    true,
				})
			}
		}
		slices.SortStableFunc(events, func(left, right event) int { return cmp.Compare(left.position, right.position) })
		for _, item := range events {
			if item.clear {
				kept := effective[:0]
				for _, policy := range effective {
					if policy.command == command {
						kept = append(kept, policy)
					}
				}
				effective = kept
				continue
			}
			effective = append(effective, item.policy)
		}
	}
	return effective, true
}

func markActiveSignalPoliciesBlocked(result *Inspection, policies []SignalPolicyID) {
	for _, id := range policies {
		publicID := id
		observation := result.SignalPolicies[publicID]
		observation.Status = SignalPolicyStatusBlocked
		result.SignalPolicies[publicID] = observation
	}
}

func signalInspectionDiagnosticFailure(operation *invocation, diagnostics []Diagnostic, cause error) error {
	payload := operationDiagnosticPayload(operation, false)
	payload.Diagnostics = make([]diagnostic, len(diagnostics))
	for i, item := range diagnostics {
		payload.Diagnostics[i] = diagnostic{
			Code: string(item.Code),
		}
	}
	return retainDiagnosticFailure(payload, cause)
}

func projectSemanticSignals(values []Signal) []Signal {
	return slices.Clone(values)
}

// applyStructuralConditionInspection projects the already-authoritative
// availability decisions made by grammar and evaluates visibility only after
// value resolution. Visibility never changes token ownership.
func applyStructuralConditionInspection(result *Inspection, model *effectiveModel, grammar grammarResult, values valueAnalysis, valuesReady bool) bool {
	if result == nil {
		return false
	}
	complete := true
	selected := make(map[CommandID]struct{}, len(grammar.Selection.Path))
	for _, command := range grammar.Selection.Path {
		selected[command] = struct{}{}
	}
	availability := make(map[conditionTarget]conditionOutcome, len(grammar.Conditions))
	for _, observation := range grammar.Conditions {
		if observation.Effect == ConditionEffectAvailability {
			availability[observation.Target] = observation.Outcome
		}
	}

	for _, record := range model.ConditionPolicies() {
		owner := structuralConditionOwner(model, record.target)
		if !structuralConditionActive(model, selected, record.target) ||
			!structuralProfileActive(grammar.SelectedProfiles, owner, record.profile) {
			setConditionInspectionOutcome(result, model, record, EvaluationOutcomeSkipped)
			continue
		}

		var outcome conditionOutcome
		switch record.effect {
		case ConditionEffectAvailability:
			var ok bool
			outcome, ok = availability[record.target]
			if !ok && valuesReady {
				// An unused declaration reaches its first presentation frontier only
				// after value resolution. Evaluating here does not revise token
				// ownership because the declaration was never used by grammar.
				outcome = evaluateCondition(record.condition, values)
				ok = true
			}
			if !ok {
				complete = false
				setConditionInspectionOutcome(result, model, record, EvaluationOutcomeUnknown)
				continue
			}
		case ConditionEffectVisibility:
			if !valuesReady {
				complete = false
				setConditionInspectionOutcome(result, model, record, EvaluationOutcomeUnknown)
				continue
			}
			outcome = evaluateCondition(record.condition, values)
		default:
			complete = false
			continue
		}
		setConditionInspectionOutcome(result, model, record, policyEvaluationOutcome(outcome))
		switch outcome {
		case conditionOutcomeSatisfied:
			// Static lifecycle/presentation state remains authoritative.
		case conditionOutcomeUnsatisfied:
			applyStructuralTargetState(result, record.target, record.effect, false)
		case conditionOutcomeBlocked,
			conditionOutcomeUnsupported,
			conditionOutcomeUnknown:
			complete = false
		}
	}
	return complete
}

func structuralConditionActive(model *effectiveModel, selected map[CommandID]struct{}, target conditionTarget) bool {
	owner := structuralConditionOwner(model, target)
	if target.Kind() != conditionTargetCommand {
		_, ok := selected[owner]
		return ok
	}
	if _, ok := selected[owner]; ok {
		return true
	}
	command, ok := model.Command(owner)
	if !ok {
		return false
	}
	parent, ok := command.Parent()
	if !ok {
		return false
	}
	_, ok = selected[parent]
	return ok
}

func structuralProfileActive(selected map[CommandID]GrammarProfileID, owner CommandID, profile GrammarProfileID) bool {
	return grammarProfileApplies(selected, owner, profile)
}

func structuralConditionOwner(model *effectiveModel, target conditionTarget) CommandID {
	switch target.Kind() {
	case conditionTargetCommand:
		return target.Command()
	case conditionTargetFlag:
		record, ok := model.Flag(target.Flag())
		if ok {
			return record.command
		}
	case conditionTargetArgumentForm:
		record, ok := model.ArgumentForm(target.ArgumentForm())
		if ok {
			return record.command
		}
	case conditionTargetArgumentTerm:
		term, ok := model.ArgumentTerm(target.ArgumentTerm())
		if !ok {
			break
		}
		form, ok := model.ArgumentForm(term.form)
		if ok {
			return form.command
		}
	case conditionTargetUnknown:
	}
	return CommandID(0)
}

func setConditionInspectionOutcome(result *Inspection, model *effectiveModel, record conditionPolicyRecord, outcome EvaluationOutcome) {
	owner := structuralConditionOwner(model, record.target)
	command, ok := result.Commands[owner]
	if !ok {
		return
	}
	target := publicStructuralConditionTarget(record.target)
	profile := record.profile
	effect := ConditionEffectUnknown
	switch record.effect {
	case ConditionEffectAvailability:
		effect = ConditionEffectAvailability
	case ConditionEffectVisibility:
		effect = ConditionEffectVisibility
	case ConditionEffectUnknown:
	}
	for index := range command.Conditions {
		condition := &command.Conditions[index]
		if condition.Target == target && condition.Effect == effect && condition.Profile == profile {
			condition.Outcome = outcome
			break
		}
	}
	result.Commands[owner] = command
}

func publicStructuralConditionTarget(target conditionTarget) DeclarationIdentity {
	switch target.Kind() {
	case conditionTargetCommand:
		return DeclarationIdentity{
			Kind:    DeclarationKindCommand,
			Command: target.Command(),
		}
	case conditionTargetFlag:
		return DeclarationIdentity{
			Kind: DeclarationKindFlag,
			Flag: target.Flag(),
		}
	case conditionTargetArgumentForm:
		return DeclarationIdentity{
			Kind:         DeclarationKindArgumentForm,
			ArgumentForm: target.ArgumentForm(),
		}
	case conditionTargetArgumentTerm:
		return DeclarationIdentity{
			Kind:         DeclarationKindArgumentTerm,
			ArgumentTerm: target.ArgumentTerm(),
		}
	default:
		return DeclarationIdentity{}
	}
}

func applyStructuralTargetState(result *Inspection, target conditionTarget, effect ConditionEffect, state bool) {
	apply := func(available, visible *bool) {
		switch effect {
		case ConditionEffectAvailability:
			*available = *available && state
			if !*available {
				*visible = false
			}
		case ConditionEffectVisibility:
			*visible = *visible && state
		default:
			return
		}
	}
	switch target.Kind() {
	case conditionTargetCommand:
		id := target.Command()
		value, ok := result.Commands[id]
		if !ok {
			return
		}
		apply(&value.Available, &value.Visible)
		result.Commands[id] = value
	case conditionTargetFlag:
		id := target.Flag()
		value, ok := result.Flags[id]
		if !ok {
			return
		}
		apply(&value.Available, &value.Visible)
		result.Flags[id] = value
		if !value.Available || !value.Visible {
			for _, spellingID := range value.Spellings {
				spelling := result.FlagSpellings[spellingID]
				if !value.Available {
					spelling.Available = false
				}
				if !value.Visible {
					spelling.Visible = false
				}
				result.FlagSpellings[spellingID] = spelling
			}
		}
	case conditionTargetArgumentForm:
		id := target.ArgumentForm()
		value, ok := result.ArgumentForms[id]
		if !ok {
			return
		}
		apply(&value.Available, &value.Visible)
		result.ArgumentForms[id] = value
	case conditionTargetArgumentTerm:
		id := target.ArgumentTerm()
		value, ok := result.ArgumentTerms[id]
		if !ok {
			return
		}
		apply(&value.Available, &value.Visible)
		result.ArgumentTerms[id] = value
	case conditionTargetUnknown:
	}
}
