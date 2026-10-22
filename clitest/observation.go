package clitest

import (
	"maps"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
)

type valueNoRepresentationExpectation struct {
	locator *valueLocatorState
}

type observedCommandDoesNotExistExpectation struct {
	locator *commandLocatorState
}

type valueStateIfObservedExpectationValue struct {
	locator *valueLocatorState
	states  []cli.ResolutionState
}

func resolveObservedInspectionCommandLocator(state *commandLocatorState, commands map[cli.CommandID]cli.CommandInspection) locatorResolution[cli.CommandInspection] {
	label := commandLocatorLabel(state)
	resolution := locatorResolution[cli.CommandInspection]{
		subject: label,
	}
	if state == nil || state.mode == commandLocatorInvalid {
		resolution.status = locatorResolutionInvalid
		resolution.subject = semanticSubjectCommand
		resolution.detail = observed(semanticSubjectCommand, observationInvalidLocator)
		return resolution
	}

	switch state.mode {
	case commandLocatorID:
		command, ok := commands[state.id]
		if !ok {
			resolution.status = locatorResolutionAbsent
			resolution.detail = observed(semanticSubjectCommand, observationAbsent)
			return resolution
		}
		resolution.status = locatorResolutionFound
		resolution.value = command
		return resolution

	case commandLocatorPath:
		matches := make([]cli.CommandInspection, 0, 1)
		for _, command := range commands {
			if slices.Equal(command.Path, state.path) {
				matches = append(matches, command)
			}
		}
		return resolveUniqueObserved(
			matches,
			true,
			label,
			observed(semanticSubjectCommand, observationAbsent),
			evaluationText("command path domain is incomplete"),
			evaluationText("command path is ambiguous"),
		)

	case commandLocatorChild:
		parent := resolveObservedInspectionCommandLocator(state.parent, commands)
		if parent.status != locatorResolutionFound {
			resolution.status = parent.status
			resolution.detail = parent.detail
			return resolution
		}

		matches := make([]cli.CommandInspection, 0, 1)
		for _, id := range parent.value.Children {
			command, ok := commands[id]
			if ok && command.Name == state.name {
				matches = append(matches, command)
			}
		}
		return resolveUniqueObserved(
			matches,
			true,
			label,
			observed(semanticSubjectChildCommand, observationAbsent),
			evaluationText("child command domain is incomplete"),
			evaluationText("child command is ambiguous"),
		)

	default:
		resolution.status = locatorResolutionInvalid
		resolution.detail = observed(semanticSubjectCommand, observationInvalidLocator)
		return resolution
	}
}

func snapshotActionEntryInspection(value cli.ActionEntryInspection) cli.ActionEntryInspection {
	result := value
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotActionEntryReference(value cli.ActionEntryReference) cli.ActionEntryReference {
	result := value
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotActionEntrySchema(value cli.ActionEntrySchema) cli.ActionEntrySchema {
	result := value
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotAdvisoryInspection(value cli.AdvisoryInspection) cli.AdvisoryInspection {
	result := value
	if value.Activation != nil {
		item0 := snapshotConditionExpression(*value.Activation)
		result.Activation = &item0
	}
	result.Subjects = slices.Clone(value.Subjects)
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotAdvisoryReference(value cli.AdvisoryReference) cli.AdvisoryReference {
	result := value
	if value.Activation != nil {
		item0 := snapshotConditionExpression(*value.Activation)
		result.Activation = &item0
	}
	result.Subjects = slices.Clone(value.Subjects)
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotAdvisorySchema(value cli.AdvisorySchema) cli.AdvisorySchema {
	result := value
	if value.Activation != nil {
		item0 := snapshotConditionExpression(*value.Activation)
		result.Activation = &item0
	}
	result.Subjects = slices.Clone(value.Subjects)
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotArgumentFormInspection(value cli.ArgumentFormInspection) cli.ArgumentFormInspection {
	result := value
	result.Terms = slices.Clone(value.Terms)
	result.Issues = slices.Clone(value.Issues)
	for index0 := range result.Issues {
		result.Issues[index0] = snapshotArgumentFormIssue(result.Issues[index0])
	}
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotArgumentFormIssue(value cli.ArgumentFormIssue) cli.ArgumentFormIssue {
	result := value
	result.InputOrigins = slices.Clone(value.InputOrigins)
	return result
}

func snapshotArgumentFormReference(value cli.ArgumentFormReference) cli.ArgumentFormReference {
	result := value
	result.Terms = slices.Clone(value.Terms)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotArgumentFormSchema(value cli.ArgumentFormSchema) cli.ArgumentFormSchema {
	result := value
	result.Terms = slices.Clone(value.Terms)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotArgumentFormSelection(value cli.ArgumentFormSelection) cli.ArgumentFormSelection {
	result := value
	result.Matches = slices.Clone(value.Matches)
	result.Blocked = slices.Clone(value.Blocked)
	result.Failed = slices.Clone(value.Failed)
	return result
}

func snapshotArgumentTermInspection(value cli.ArgumentTermInspection) cli.ArgumentTermInspection {
	result := value
	if value.TokenLanguage != nil {
		item0 := snapshotTokenLanguageSyntax(*value.TokenLanguage)
		result.TokenLanguage = &item0
	}
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	result.InputOrigins = slices.Clone(value.InputOrigins)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotArgumentTermReference(value cli.ArgumentTermReference) cli.ArgumentTermReference {
	result := value
	if value.TokenLanguage != nil {
		item0 := snapshotTokenLanguageSyntax(*value.TokenLanguage)
		result.TokenLanguage = &item0
	}
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotArgumentTermSchema(value cli.ArgumentTermSchema) cli.ArgumentTermSchema {
	result := value
	if value.TokenLanguage != nil {
		item0 := snapshotTokenLanguageSyntax(*value.TokenLanguage)
		result.TokenLanguage = &item0
	}
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotCommandDefaultInspection(value cli.CommandDefaultInspection) cli.CommandDefaultInspection {
	result := value
	if value.Condition != nil {
		item0 := snapshotConditionExpression(*value.Condition)
		result.Condition = &item0
	}
	return result
}

func snapshotCommandDefaultReference(value cli.CommandDefaultReference) cli.CommandDefaultReference {
	result := value
	if value.Condition != nil {
		item0 := snapshotConditionExpression(*value.Condition)
		result.Condition = &item0
	}
	return result
}

func snapshotCommandDefaultSchema(value cli.CommandDefaultSchema) cli.CommandDefaultSchema {
	result := value
	if value.Condition != nil {
		item0 := snapshotConditionExpression(*value.Condition)
		result.Condition = &item0
	}
	return result
}

func snapshotCommandDialect(value cli.CommandDialect) cli.CommandDialect {
	result := value
	result.EndOfOptions = slices.Clone(value.EndOfOptions)
	return result
}

func snapshotCommandEntryInspection(value cli.CommandEntryInspection) cli.CommandEntryInspection {
	result := value
	result.TargetPath = slices.Clone(value.TargetPath)
	result.Examples = slices.Clone(value.Examples)
	result.Args = slices.Clone(value.Args)
	result.AppendArgs = slices.Clone(value.AppendArgs)
	result.Defaults = slices.Clone(value.Defaults)
	for index0 := range result.Defaults {
		result.Defaults[index0] = snapshotCommandShortcutDefaultInspection(result.Defaults[index0])
	}
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotCommandEntryReference(value cli.CommandEntryReference) cli.CommandEntryReference {
	result := value
	result.TargetPath = slices.Clone(value.TargetPath)
	result.Examples = slices.Clone(value.Examples)
	result.Args = slices.Clone(value.Args)
	result.AppendArgs = slices.Clone(value.AppendArgs)
	result.Defaults = slices.Clone(value.Defaults)
	for index0 := range result.Defaults {
		result.Defaults[index0] = snapshotCommandShortcutDefaultReference(result.Defaults[index0])
	}
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotCommandEntrySchema(value cli.CommandEntrySchema) cli.CommandEntrySchema {
	result := value
	result.TargetPath = slices.Clone(value.TargetPath)
	result.Examples = slices.Clone(value.Examples)
	result.Args = slices.Clone(value.Args)
	result.AppendArgs = slices.Clone(value.AppendArgs)
	result.Defaults = slices.Clone(value.Defaults)
	for index0 := range result.Defaults {
		result.Defaults[index0] = snapshotCommandShortcutDefaultSchema(result.Defaults[index0])
	}
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotCommandInspection(value cli.CommandInspection) cli.CommandInspection {
	result := value
	result.Children = slices.Clone(value.Children)
	result.Path = slices.Clone(value.Path)
	result.Examples = slices.Clone(value.Examples)
	if value.Dialect != nil {
		item0 := snapshotCommandDialect(*value.Dialect)
		result.Dialect = &item0
	}
	result.Access = snapshotExecutionAccess(value.Access)
	result.DefaultSelection = snapshotDefaultCommandSelection(value.DefaultSelection)
	result.ArgumentFormSelection = snapshotArgumentFormSelection(value.ArgumentFormSelection)
	result.CommandEntries = slices.Clone(value.CommandEntries)
	result.ActionEntries = slices.Clone(value.ActionEntries)
	result.CommandDefaults = slices.Clone(value.CommandDefaults)
	result.CommandProviders = slices.Clone(value.CommandProviders)
	result.Flags = slices.Clone(value.Flags)
	result.Facts = slices.Clone(value.Facts)
	result.ArgumentForms = slices.Clone(value.ArgumentForms)
	result.Values = slices.Clone(value.Values)
	result.Conditions = slices.Clone(value.Conditions)
	for index1 := range result.Conditions {
		result.Conditions[index1] = snapshotConditionInspection(result.Conditions[index1])
	}
	result.Rules = slices.Clone(value.Rules)
	for index2 := range result.Rules {
		result.Rules[index2] = snapshotRuleInspection(result.Rules[index2])
	}
	result.Advisories = slices.Clone(value.Advisories)
	for index3 := range result.Advisories {
		result.Advisories[index3] = snapshotAdvisoryInspection(result.Advisories[index3])
	}
	result.NoticeSuppressions = slices.Clone(value.NoticeSuppressions)
	for index4 := range result.NoticeSuppressions {
		result.NoticeSuppressions[index4] = snapshotNoticeSuppressionInspection(result.NoticeSuppressions[index4])
	}
	result.SignalPolicies = slices.Clone(value.SignalPolicies)
	return result
}

func snapshotCommandProviderInspection(value cli.CommandProviderInspection) cli.CommandProviderInspection {
	result := value
	result.Dependencies = slices.Clone(value.Dependencies)
	if value.Localization != nil {
		item0 := snapshotLocalization(*value.Localization)
		result.Localization = &item0
	}
	result.CommandEntries = slices.Clone(value.CommandEntries)
	return result
}

func snapshotCommandProviderReference(value cli.CommandProviderReference) cli.CommandProviderReference {
	result := value
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotCommandProviderSchema(value cli.CommandProviderSchema) cli.CommandProviderSchema {
	result := value
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotCommandReference(value cli.CommandReference) cli.CommandReference {
	result := value
	result.Children = slices.Clone(value.Children)
	result.Path = slices.Clone(value.Path)
	result.Examples = slices.Clone(value.Examples)
	if value.Dialect != nil {
		item0 := snapshotCommandDialect(*value.Dialect)
		result.Dialect = &item0
	}
	if value.Profiles != nil {
		item1 := snapshotGrammarProfileSet(*value.Profiles)
		result.Profiles = &item1
	}
	result.Usage = slices.Clone(value.Usage)
	result.ProfileUsage = maps.Clone(value.ProfileUsage)
	for key2, item := range result.ProfileUsage {
		item = slices.Clone(item)
		result.ProfileUsage[key2] = item
	}
	result.Help = snapshotHelpSchema(value.Help)
	result.Access = snapshotExecutionAccess(value.Access)
	result.Flags = slices.Clone(value.Flags)
	result.Facts = slices.Clone(value.Facts)
	result.ArgumentForms = slices.Clone(value.ArgumentForms)
	result.Values = slices.Clone(value.Values)
	result.Conditions = slices.Clone(value.Conditions)
	for index3 := range result.Conditions {
		result.Conditions[index3] = snapshotConditionSchema(result.Conditions[index3])
	}
	result.Rules = slices.Clone(value.Rules)
	for index4 := range result.Rules {
		result.Rules[index4] = snapshotRuleSchema(result.Rules[index4])
	}
	result.Advisories = slices.Clone(value.Advisories)
	for index5 := range result.Advisories {
		result.Advisories[index5] = snapshotAdvisoryReference(result.Advisories[index5])
	}
	result.NoticeSuppressions = slices.Clone(value.NoticeSuppressions)
	for index6 := range result.NoticeSuppressions {
		result.NoticeSuppressions[index6] = snapshotNoticeSuppressionSchema(result.NoticeSuppressions[index6])
	}
	result.SignalPolicies = slices.Clone(value.SignalPolicies)
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	result.CommandEntries = slices.Clone(value.CommandEntries)
	result.ActionEntries = slices.Clone(value.ActionEntries)
	result.CommandDefaults = slices.Clone(value.CommandDefaults)
	result.CommandProviders = slices.Clone(value.CommandProviders)
	return result
}

func snapshotCommandSchema(value cli.CommandSchema) cli.CommandSchema {
	result := value
	result.Children = slices.Clone(value.Children)
	result.Path = slices.Clone(value.Path)
	result.Examples = slices.Clone(value.Examples)
	if value.Dialect != nil {
		item0 := snapshotCommandDialect(*value.Dialect)
		result.Dialect = &item0
	}
	if value.Profiles != nil {
		item1 := snapshotGrammarProfileSet(*value.Profiles)
		result.Profiles = &item1
	}
	result.Help = snapshotHelpSchema(value.Help)
	result.Access = snapshotExecutionAccess(value.Access)
	result.Flags = slices.Clone(value.Flags)
	result.Facts = slices.Clone(value.Facts)
	result.ArgumentForms = slices.Clone(value.ArgumentForms)
	result.Values = slices.Clone(value.Values)
	result.Conditions = slices.Clone(value.Conditions)
	for index2 := range result.Conditions {
		result.Conditions[index2] = snapshotConditionSchema(result.Conditions[index2])
	}
	result.Rules = slices.Clone(value.Rules)
	for index3 := range result.Rules {
		result.Rules[index3] = snapshotRuleSchema(result.Rules[index3])
	}
	result.Advisories = slices.Clone(value.Advisories)
	for index4 := range result.Advisories {
		result.Advisories[index4] = snapshotAdvisorySchema(result.Advisories[index4])
	}
	result.NoticeSuppressions = slices.Clone(value.NoticeSuppressions)
	for index5 := range result.NoticeSuppressions {
		result.NoticeSuppressions[index5] = snapshotNoticeSuppressionSchema(result.NoticeSuppressions[index5])
	}
	result.SignalPolicies = slices.Clone(value.SignalPolicies)
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	result.CommandEntries = slices.Clone(value.CommandEntries)
	result.ActionEntries = slices.Clone(value.ActionEntries)
	result.CommandDefaults = slices.Clone(value.CommandDefaults)
	result.CommandProviders = slices.Clone(value.CommandProviders)
	return result
}

func snapshotCommandSelection(value cli.CommandSelection) cli.CommandSelection {
	result := value
	result.Path = slices.Clone(value.Path)
	result.Steps = slices.Clone(value.Steps)
	for index0 := range result.Steps {
		result.Steps[index0] = snapshotCommandSelectionStep(result.Steps[index0])
	}
	return result
}

func snapshotCommandSelectionStep(value cli.CommandSelectionStep) cli.CommandSelectionStep {
	result := value
	result.Commands = slices.Clone(value.Commands)
	return result
}

func snapshotCommandShortcutDefaultInspection(value cli.CommandShortcutDefaultInspection) cli.CommandShortcutDefaultInspection {
	result := value
	result.Values = slices.Clone(value.Values)
	return result
}

func snapshotCommandShortcutDefaultReference(value cli.CommandShortcutDefaultReference) cli.CommandShortcutDefaultReference {
	result := value
	result.Values = slices.Clone(value.Values)
	return result
}

func snapshotCommandShortcutDefaultSchema(value cli.CommandShortcutDefaultSchema) cli.CommandShortcutDefaultSchema {
	result := value
	result.Values = slices.Clone(value.Values)
	return result
}

func snapshotCompletion(value cli.Completion) cli.Completion {
	result := value
	result.Targets = slices.Clone(value.Targets)
	for index0 := range result.Targets {
		result.Targets[index0] = snapshotCompletionTarget(result.Targets[index0])
	}
	result.Selection = snapshotCommandSelection(value.Selection)
	result.Items = slices.Clone(value.Items)
	for index1 := range result.Items {
		result.Items[index1] = snapshotCompletionItem(result.Items[index1])
	}
	result.Groups = slices.Clone(value.Groups)
	result.Messages = slices.Clone(value.Messages)
	result.NativeRequests = slices.Clone(value.NativeRequests)
	if value.Delegation != nil {
		item2 := snapshotCompletionDelegation(*value.Delegation)
		result.Delegation = &item2
	}
	result.Sources = maps.Clone(value.Sources)
	result.Origins = maps.Clone(value.Origins)
	for key3, item := range result.Origins {
		item = snapshotInputOrigin(item)
		result.Origins[key3] = item
	}
	result.Diagnostics = slices.Clone(value.Diagnostics)
	for index4 := range result.Diagnostics {
		result.Diagnostics[index4] = snapshotDiagnostic(result.Diagnostics[index4])
	}
	result.Repairs = slices.Clone(value.Repairs)
	for index5 := range result.Repairs {
		result.Repairs[index5] = snapshotInputRepair(result.Repairs[index5])
	}
	return result
}

func snapshotCompletionCandidate(value cli.CompletionCandidate) cli.CompletionCandidate {
	result := value
	result.Replacement = slices.Clone(value.Replacement)
	result.SearchTerms = slices.Clone(value.SearchTerms)
	result.CommitCharacters = slices.Clone(value.CommitCharacters)
	return result
}

func snapshotCompletionDeclarationCapabilities(value cli.CompletionDeclarationCapabilities) cli.CompletionDeclarationCapabilities {
	result := value
	result.Dependencies = slices.Clone(value.Dependencies)
	if value.Native != nil {
		result.Native = new(*value.Native)
	}
	return result
}

func snapshotCompletionDelegation(value cli.CompletionDelegation) cli.CompletionDelegation {
	result := value
	result.Args = slices.Clone(value.Args)
	return result
}

func snapshotCompletionItem(value cli.CompletionItem) cli.CompletionItem {
	result := value
	result.CompletionCandidate = snapshotCompletionCandidate(value.CompletionCandidate)
	result.TargetIndexes = slices.Clone(value.TargetIndexes)
	result.Match = snapshotCompletionMatch(value.Match)
	return result
}

func snapshotCompletionMatch(value cli.CompletionMatch) cli.CompletionMatch {
	result := value
	result.Ranges = slices.Clone(value.Ranges)
	return result
}

func snapshotCompletionTarget(value cli.CompletionTarget) cli.CompletionTarget {
	result := value
	result.CommandPath = slices.Clone(value.CommandPath)
	return result
}

func snapshotConditionExpression(value cli.ConditionExpression) cli.ConditionExpression {
	result := value
	result.Operands = slices.Clone(value.Operands)
	result.ProducerKinds = slices.Clone(value.ProducerKinds)
	result.Children = slices.Clone(value.Children)
	for index0 := range result.Children {
		result.Children[index0] = snapshotConditionExpression(result.Children[index0])
	}
	return result
}

func snapshotConditionInspection(value cli.ConditionInspection) cli.ConditionInspection {
	result := value
	result.Expression = snapshotConditionExpression(value.Expression)
	return result
}

func snapshotConditionSchema(value cli.ConditionSchema) cli.ConditionSchema {
	result := value
	result.Expression = snapshotConditionExpression(value.Expression)
	return result
}

func snapshotDefaultCommandSelection(value cli.DefaultCommandSelection) cli.DefaultCommandSelection {
	result := value
	result.Candidates = slices.Clone(value.Candidates)
	result.Applicable = slices.Clone(value.Applicable)
	result.Blocked = slices.Clone(value.Blocked)
	return result
}

func snapshotDiagnostic(value cli.Diagnostic) cli.Diagnostic {
	result := value
	if value.Rule != nil {
		item0 := snapshotRuleFailure(*value.Rule)
		result.Rule = &item0
	}
	result.CommandPath = slices.Clone(value.CommandPath)
	if value.ProviderOutput != nil {
		result.ProviderOutput = new(*value.ProviderOutput)
	}
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	if value.Span != nil {
		result.Span = new(*value.Span)
	}
	result.Origins = slices.Clone(value.Origins)
	result.Suggestions = slices.Clone(value.Suggestions)
	result.Usage = slices.Clone(value.Usage)
	result.RepairIndexes = slices.Clone(value.RepairIndexes)
	if value.SignalAction != nil {
		item4 := snapshotSignalActionFailure(*value.SignalAction)
		result.SignalAction = &item4
	}
	if value.SignalOverflow != nil {
		result.SignalOverflow = new(*value.SignalOverflow)
	}
	if value.NoticeDelivery != nil {
		item6 := snapshotNoticeDeliveryFailure(*value.NoticeDelivery)
		result.NoticeDelivery = &item6
	}
	return result
}

func snapshotDiagnosticReport(value cli.DiagnosticReport) cli.DiagnosticReport {
	result := value
	result.Diagnostics = slices.Clone(value.Diagnostics)
	for index0 := range result.Diagnostics {
		result.Diagnostics[index0] = snapshotDiagnostic(result.Diagnostics[index0])
	}
	result.Sources = maps.Clone(value.Sources)
	result.Origins = maps.Clone(value.Origins)
	for key1, item := range result.Origins {
		item = snapshotInputOrigin(item)
		result.Origins[key1] = item
	}
	result.Repairs = slices.Clone(value.Repairs)
	for index2 := range result.Repairs {
		result.Repairs[index2] = snapshotInputRepair(result.Repairs[index2])
	}
	return result
}

func snapshotEnvironmentInputInspection(value cli.EnvironmentInputInspection) cli.EnvironmentInputInspection {
	result := value
	result.EnvironmentInputSchema = snapshotEnvironmentInputSchema(value.EnvironmentInputSchema)
	return result
}

func snapshotEnvironmentInputReference(value cli.EnvironmentInputReference) cli.EnvironmentInputReference {
	result := value
	result.EnvironmentInputSchema = snapshotEnvironmentInputSchema(value.EnvironmentInputSchema)
	return result
}

func snapshotEnvironmentInputSchema(value cli.EnvironmentInputSchema) cli.EnvironmentInputSchema {
	result := value
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotExecutionAccess(value cli.ExecutionAccess) cli.ExecutionAccess {
	result := value
	result.Values = slices.Clone(value.Values)
	result.OptionalValues = slices.Clone(value.OptionalValues)
	result.ValueStates = slices.Clone(value.ValueStates)
	return result
}

func snapshotExecutionAdmissionInspection(value cli.ExecutionAdmissionInspection) cli.ExecutionAdmissionInspection {
	result := value
	result.Middleware = slices.Clone(value.Middleware)
	result.SignalPolicies = slices.Clone(value.SignalPolicies)
	result.SignalEndpoints = slices.Clone(value.SignalEndpoints)
	result.BlockingDiagnosticIndexes = slices.Clone(value.BlockingDiagnosticIndexes)
	return result
}

func snapshotFactInspection(value cli.FactInspection) cli.FactInspection {
	result := value
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotFactReference(value cli.FactReference) cli.FactReference {
	result := value
	result.FactSchema = snapshotFactSchema(value.FactSchema)
	return result
}

func snapshotFactSchema(value cli.FactSchema) cli.FactSchema {
	result := value
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotFlagInspection(value cli.FlagInspection) cli.FlagInspection {
	result := value
	result.Spellings = slices.Clone(value.Spellings)
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotFlagReference(value cli.FlagReference) cli.FlagReference {
	result := value
	result.Spellings = slices.Clone(value.Spellings)
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotFlagSchema(value cli.FlagSchema) cli.FlagSchema {
	result := value
	result.Spellings = slices.Clone(value.Spellings)
	result.Completion = snapshotCompletionDeclarationCapabilities(value.Completion)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotFlagSpelling(value cli.FlagSpelling) cli.FlagSpelling {
	result := value
	result.ValueForms = slices.Clone(value.ValueForms)
	if value.FixedValue != nil {
		result.FixedValue = new(*value.FixedValue)
	}
	if value.ImplicitValue != nil {
		result.ImplicitValue = new(*value.ImplicitValue)
	}
	return result
}

func snapshotFlagSpellingInspection(value cli.FlagSpellingInspection) cli.FlagSpellingInspection {
	result := value
	result.Spelling = snapshotFlagSpelling(value.Spelling)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotFlagSpellingReference(value cli.FlagSpellingReference) cli.FlagSpellingReference {
	result := value
	result.Spelling = snapshotFlagSpelling(value.Spelling)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotFlagSpellingSchema(value cli.FlagSpellingSchema) cli.FlagSpellingSchema {
	result := value
	result.Spelling = snapshotFlagSpelling(value.Spelling)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotGrammarProfile(value cli.GrammarProfile) cli.GrammarProfile {
	result := value
	result.Dialect = snapshotCommandDialect(value.Dialect)
	return result
}

func snapshotGrammarProfileSet(value cli.GrammarProfileSet) cli.GrammarProfileSet {
	result := value
	result.Cases = slices.Clone(value.Cases)
	result.Profiles = maps.Clone(value.Profiles)
	for key0, item := range result.Profiles {
		item = snapshotGrammarProfile(item)
		result.Profiles[key0] = item
	}
	return result
}

func snapshotHelpCommandGroupSchema(value cli.HelpCommandGroupSchema) cli.HelpCommandGroupSchema {
	result := value
	result.Entries = slices.Clone(value.Entries)
	return result
}

func snapshotHelpFlagGroupSchema(value cli.HelpFlagGroupSchema) cli.HelpFlagGroupSchema {
	result := value
	result.Flags = slices.Clone(value.Flags)
	return result
}

func snapshotHelpSchema(value cli.HelpSchema) cli.HelpSchema {
	result := value
	result.CommandGroups = slices.Clone(value.CommandGroups)
	for index0 := range result.CommandGroups {
		result.CommandGroups[index0] = snapshotHelpCommandGroupSchema(result.CommandGroups[index0])
	}
	result.FlagGroups = slices.Clone(value.FlagGroups)
	for index1 := range result.FlagGroups {
		result.FlagGroups[index1] = snapshotHelpFlagGroupSchema(result.FlagGroups[index1])
	}
	return result
}

func snapshotInputEntryEdit(value cli.InputEntryEdit) cli.InputEntryEdit {
	result := value
	result.Entries = slices.Clone(value.Entries)
	for index0 := range result.Entries {
		result.Entries[index0] = snapshotInputReplacement(result.Entries[index0])
	}
	return result
}

func snapshotInputOrigin(value cli.InputOrigin) cli.InputOrigin {
	result := value
	result.Parents = slices.Clone(value.Parents)
	return result
}

func snapshotInputRepair(value cli.InputRepair) cli.InputRepair {
	result := value
	result.Sources = slices.Clone(value.Sources)
	for index0 := range result.Sources {
		result.Sources[index0] = snapshotInputRepairSource(result.Sources[index0])
	}
	return result
}

func snapshotInputRepairSource(value cli.InputRepairSource) cli.InputRepairSource {
	result := value
	result.TextEdits = slices.Clone(value.TextEdits)
	for index0 := range result.TextEdits {
		result.TextEdits[index0] = snapshotInputTextEdit(result.TextEdits[index0])
	}
	result.EntryEdits = slices.Clone(value.EntryEdits)
	for index1 := range result.EntryEdits {
		result.EntryEdits[index1] = snapshotInputEntryEdit(result.EntryEdits[index1])
	}
	return result
}

func snapshotInputReplacement(value cli.InputReplacement) cli.InputReplacement {
	result := slices.Clone(value)
	for index0 := range result {
		result[index0] = snapshotInputReplacementPart(result[index0])
	}
	return result
}

func snapshotInputReplacementPart(value cli.InputReplacementPart) cli.InputReplacementPart {
	result := value
	if value.Copy != nil {
		result.Copy = new(*value.Copy)
	}
	return result
}

func snapshotInputTextEdit(value cli.InputTextEdit) cli.InputTextEdit {
	result := value
	result.Replacement = snapshotInputReplacement(value.Replacement)
	return result
}

func snapshotInspection(value cli.Inspection) cli.Inspection {
	result := value
	result.Selection = snapshotInvocationSelection(value.Selection)
	result.CommandOrder = slices.Clone(value.CommandOrder)
	result.Commands = maps.Clone(value.Commands)
	for key0, item := range result.Commands {
		item = snapshotCommandInspection(item)
		result.Commands[key0] = item
	}
	result.CommandEntries = maps.Clone(value.CommandEntries)
	for key1, item := range result.CommandEntries {
		item = snapshotCommandEntryInspection(item)
		result.CommandEntries[key1] = item
	}
	result.ActionEntries = maps.Clone(value.ActionEntries)
	for key2, item := range result.ActionEntries {
		item = snapshotActionEntryInspection(item)
		result.ActionEntries[key2] = item
	}
	result.CommandDefaults = maps.Clone(value.CommandDefaults)
	for key3, item := range result.CommandDefaults {
		item = snapshotCommandDefaultInspection(item)
		result.CommandDefaults[key3] = item
	}
	result.CommandProviders = maps.Clone(value.CommandProviders)
	for key4, item := range result.CommandProviders {
		item = snapshotCommandProviderInspection(item)
		result.CommandProviders[key4] = item
	}
	result.Flags = maps.Clone(value.Flags)
	for key5, item := range result.Flags {
		item = snapshotFlagInspection(item)
		result.Flags[key5] = item
	}
	result.FlagSpellings = maps.Clone(value.FlagSpellings)
	for key6, item := range result.FlagSpellings {
		item = snapshotFlagSpellingInspection(item)
		result.FlagSpellings[key6] = item
	}
	result.ArgumentForms = maps.Clone(value.ArgumentForms)
	for key7, item := range result.ArgumentForms {
		item = snapshotArgumentFormInspection(item)
		result.ArgumentForms[key7] = item
	}
	result.ArgumentTerms = maps.Clone(value.ArgumentTerms)
	for key8, item := range result.ArgumentTerms {
		item = snapshotArgumentTermInspection(item)
		result.ArgumentTerms[key8] = item
	}
	result.Facts = maps.Clone(value.Facts)
	for key9, item := range result.Facts {
		item = snapshotFactInspection(item)
		result.Facts[key9] = item
	}
	result.EnvironmentInputs = maps.Clone(value.EnvironmentInputs)
	for key10, item := range result.EnvironmentInputs {
		item = snapshotEnvironmentInputInspection(item)
		result.EnvironmentInputs[key10] = item
	}
	result.ValueDomains = slices.Clone(value.ValueDomains)
	for index11 := range result.ValueDomains {
		result.ValueDomains[index11] = snapshotValueDomainInspection(result.ValueDomains[index11])
	}
	result.ValueDomainEntries = maps.Clone(value.ValueDomainEntries)
	for key12, item := range result.ValueDomainEntries {
		item = snapshotValueDomainEntryInspection(item)
		result.ValueDomainEntries[key12] = item
	}
	result.Values = maps.Clone(value.Values)
	for key13, item := range result.Values {
		item = snapshotValueInspection(item)
		result.Values[key13] = item
	}
	result.ValueCandidates = maps.Clone(value.ValueCandidates)
	for key14, item := range result.ValueCandidates {
		item = snapshotValueCandidateInspection(item)
		result.ValueCandidates[key14] = item
	}
	result.Sources = maps.Clone(value.Sources)
	result.Origins = maps.Clone(value.Origins)
	for key15, item := range result.Origins {
		item = snapshotInputOrigin(item)
		result.Origins[key15] = item
	}
	result.Tokens = slices.Clone(value.Tokens)
	result.SignalSource = snapshotSignalSourceInspection(value.SignalSource)
	result.SignalPolicies = maps.Clone(value.SignalPolicies)
	for key16, item := range result.SignalPolicies {
		item = snapshotSignalPolicyInspection(item)
		result.SignalPolicies[key16] = item
	}
	result.SignalEndpoints = maps.Clone(value.SignalEndpoints)
	for key17, item := range result.SignalEndpoints {
		item = snapshotSignalEndpointInspection(item)
		result.SignalEndpoints[key17] = item
	}
	result.Diagnostics = slices.Clone(value.Diagnostics)
	for index18 := range result.Diagnostics {
		result.Diagnostics[index18] = snapshotDiagnostic(result.Diagnostics[index18])
	}
	result.Repairs = slices.Clone(value.Repairs)
	for index19 := range result.Repairs {
		result.Repairs[index19] = snapshotInputRepair(result.Repairs[index19])
	}
	result.Notices = slices.Clone(value.Notices)
	for index20 := range result.Notices {
		result.Notices[index20] = snapshotNoticeInspection(result.Notices[index20])
	}
	result.ExecutionAdmission = snapshotExecutionAdmissionInspection(value.ExecutionAdmission)
	return result
}

func snapshotInvocationSelection(value cli.InvocationSelection) cli.InvocationSelection {
	result := value
	result.Command = snapshotCommandSelection(value.Command)
	return result
}

func snapshotLocalization(value cli.Localization) cli.Localization {
	result := value
	result.Bundles = slices.Clone(value.Bundles)
	for index0 := range result.Bundles {
		result.Bundles[index0] = snapshotTranslationBundle(result.Bundles[index0])
	}
	return result
}

func snapshotNotice(value cli.Notice) cli.Notice {
	result := value
	result.CommandPath = slices.Clone(value.CommandPath)
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotNoticeDeliveryFailure(value cli.NoticeDeliveryFailure) cli.NoticeDeliveryFailure {
	result := value
	result.CommandPath = slices.Clone(value.CommandPath)
	return result
}

func snapshotNoticeInspection(value cli.NoticeInspection) cli.NoticeInspection {
	result := value
	result.Notice = snapshotNotice(value.Notice)
	result.SuppressedBy = slices.Clone(value.SuppressedBy)
	result.RepairIndexes = slices.Clone(value.RepairIndexes)
	return result
}

func snapshotNoticeSuppressionInspection(value cli.NoticeSuppressionInspection) cli.NoticeSuppressionInspection {
	result := value
	result.Condition = snapshotConditionExpression(value.Condition)
	return result
}

func snapshotNoticeSuppressionSchema(value cli.NoticeSuppressionSchema) cli.NoticeSuppressionSchema {
	result := value
	result.Condition = snapshotConditionExpression(value.Condition)
	return result
}

func snapshotReference(value cli.Reference) cli.Reference {
	result := value
	result.CommandOrder = slices.Clone(value.CommandOrder)
	result.Commands = maps.Clone(value.Commands)
	for key0, item := range result.Commands {
		item = snapshotCommandReference(item)
		result.Commands[key0] = item
	}
	result.CommandEntries = maps.Clone(value.CommandEntries)
	for key1, item := range result.CommandEntries {
		item = snapshotCommandEntryReference(item)
		result.CommandEntries[key1] = item
	}
	result.ActionEntries = maps.Clone(value.ActionEntries)
	for key2, item := range result.ActionEntries {
		item = snapshotActionEntryReference(item)
		result.ActionEntries[key2] = item
	}
	result.CommandDefaults = maps.Clone(value.CommandDefaults)
	for key3, item := range result.CommandDefaults {
		item = snapshotCommandDefaultReference(item)
		result.CommandDefaults[key3] = item
	}
	result.CommandProviders = maps.Clone(value.CommandProviders)
	for key4, item := range result.CommandProviders {
		item = snapshotCommandProviderReference(item)
		result.CommandProviders[key4] = item
	}
	result.Flags = maps.Clone(value.Flags)
	for key5, item := range result.Flags {
		item = snapshotFlagReference(item)
		result.Flags[key5] = item
	}
	result.FlagSpellings = maps.Clone(value.FlagSpellings)
	for key6, item := range result.FlagSpellings {
		item = snapshotFlagSpellingReference(item)
		result.FlagSpellings[key6] = item
	}
	result.ArgumentForms = maps.Clone(value.ArgumentForms)
	for key7, item := range result.ArgumentForms {
		item = snapshotArgumentFormReference(item)
		result.ArgumentForms[key7] = item
	}
	result.ArgumentTerms = maps.Clone(value.ArgumentTerms)
	for key8, item := range result.ArgumentTerms {
		item = snapshotArgumentTermReference(item)
		result.ArgumentTerms[key8] = item
	}
	result.Values = maps.Clone(value.Values)
	for key9, item := range result.Values {
		item = snapshotValueReference(item)
		result.Values[key9] = item
	}
	result.Facts = maps.Clone(value.Facts)
	for key10, item := range result.Facts {
		item = snapshotFactReference(item)
		result.Facts[key10] = item
	}
	result.EnvironmentInputs = maps.Clone(value.EnvironmentInputs)
	for key11, item := range result.EnvironmentInputs {
		item = snapshotEnvironmentInputReference(item)
		result.EnvironmentInputs[key11] = item
	}
	result.ValueDomains = slices.Clone(value.ValueDomains)
	for index12 := range result.ValueDomains {
		result.ValueDomains[index12] = snapshotValueDomainReference(result.ValueDomains[index12])
	}
	result.ValueDomainEntries = maps.Clone(value.ValueDomainEntries)
	for key13, item := range result.ValueDomainEntries {
		item = snapshotValueDomainEntryReference(item)
		result.ValueDomainEntries[key13] = item
	}
	result.SignalPolicies = maps.Clone(value.SignalPolicies)
	for key14, item := range result.SignalPolicies {
		item = snapshotSignalPolicyReference(item)
		result.SignalPolicies[key14] = item
	}
	result.SignalEndpoints = maps.Clone(value.SignalEndpoints)
	for key15, item := range result.SignalEndpoints {
		item = snapshotSignalEndpoint(item)
		result.SignalEndpoints[key15] = item
	}
	return result
}

func snapshotRuleFailure(value cli.RuleFailure) cli.RuleFailure {
	result := value
	result.Subjects = slices.Clone(value.Subjects)
	result.Operands = slices.Clone(value.Operands)
	for index0 := range result.Operands {
		result.Operands[index0] = snapshotRuleOperand(result.Operands[index0])
	}
	return result
}

func snapshotRuleInspection(value cli.RuleInspection) cli.RuleInspection {
	result := value
	result.Subjects = slices.Clone(value.Subjects)
	result.Dependencies = slices.Clone(value.Dependencies)
	result.Operands = slices.Clone(value.Operands)
	for index0 := range result.Operands {
		result.Operands[index0] = snapshotRuleOperand(result.Operands[index0])
	}
	if value.Activation != nil {
		item1 := snapshotConditionExpression(*value.Activation)
		result.Activation = &item1
	}
	result.DiagnosticIndexes = slices.Clone(value.DiagnosticIndexes)
	return result
}

func snapshotRuleOperand(value cli.RuleOperand) cli.RuleOperand {
	result := value
	if value.Literal != nil {
		result.Literal = new(*value.Literal)
	}
	return result
}

func snapshotRuleSchema(value cli.RuleSchema) cli.RuleSchema {
	result := value
	result.Subjects = slices.Clone(value.Subjects)
	result.Dependencies = slices.Clone(value.Dependencies)
	result.Operands = slices.Clone(value.Operands)
	for index0 := range result.Operands {
		result.Operands[index0] = snapshotRuleOperand(result.Operands[index0])
	}
	if value.Activation != nil {
		item1 := snapshotConditionExpression(*value.Activation)
		result.Activation = &item1
	}
	return result
}

func snapshotSchema(value cli.Schema) cli.Schema {
	result := value
	result.Localization = snapshotLocalization(value.Localization)
	result.CommandOrder = slices.Clone(value.CommandOrder)
	result.Commands = maps.Clone(value.Commands)
	for key0, item := range result.Commands {
		item = snapshotCommandSchema(item)
		result.Commands[key0] = item
	}
	result.CommandEntries = maps.Clone(value.CommandEntries)
	for key1, item := range result.CommandEntries {
		item = snapshotCommandEntrySchema(item)
		result.CommandEntries[key1] = item
	}
	result.ActionEntries = maps.Clone(value.ActionEntries)
	for key2, item := range result.ActionEntries {
		item = snapshotActionEntrySchema(item)
		result.ActionEntries[key2] = item
	}
	result.CommandDefaults = maps.Clone(value.CommandDefaults)
	for key3, item := range result.CommandDefaults {
		item = snapshotCommandDefaultSchema(item)
		result.CommandDefaults[key3] = item
	}
	result.CommandProviders = maps.Clone(value.CommandProviders)
	for key4, item := range result.CommandProviders {
		item = snapshotCommandProviderSchema(item)
		result.CommandProviders[key4] = item
	}
	result.Flags = maps.Clone(value.Flags)
	for key5, item := range result.Flags {
		item = snapshotFlagSchema(item)
		result.Flags[key5] = item
	}
	result.FlagSpellings = maps.Clone(value.FlagSpellings)
	for key6, item := range result.FlagSpellings {
		item = snapshotFlagSpellingSchema(item)
		result.FlagSpellings[key6] = item
	}
	result.ArgumentForms = maps.Clone(value.ArgumentForms)
	for key7, item := range result.ArgumentForms {
		item = snapshotArgumentFormSchema(item)
		result.ArgumentForms[key7] = item
	}
	result.ArgumentTerms = maps.Clone(value.ArgumentTerms)
	for key8, item := range result.ArgumentTerms {
		item = snapshotArgumentTermSchema(item)
		result.ArgumentTerms[key8] = item
	}
	result.Values = maps.Clone(value.Values)
	for key9, item := range result.Values {
		item = snapshotValueSchema(item)
		result.Values[key9] = item
	}
	result.Facts = maps.Clone(value.Facts)
	for key10, item := range result.Facts {
		item = snapshotFactSchema(item)
		result.Facts[key10] = item
	}
	result.EnvironmentInputs = maps.Clone(value.EnvironmentInputs)
	for key11, item := range result.EnvironmentInputs {
		item = snapshotEnvironmentInputSchema(item)
		result.EnvironmentInputs[key11] = item
	}
	result.ValueDomains = slices.Clone(value.ValueDomains)
	for index12 := range result.ValueDomains {
		result.ValueDomains[index12] = snapshotValueDomainSchema(result.ValueDomains[index12])
	}
	result.ValueDomainEntries = maps.Clone(value.ValueDomainEntries)
	for key13, item := range result.ValueDomainEntries {
		item = snapshotValueDomainEntrySchema(item)
		result.ValueDomainEntries[key13] = item
	}
	result.SignalPolicies = maps.Clone(value.SignalPolicies)
	for key14, item := range result.SignalPolicies {
		item = snapshotSignalPolicySchema(item)
		result.SignalPolicies[key14] = item
	}
	result.SignalEndpoints = maps.Clone(value.SignalEndpoints)
	for key15, item := range result.SignalEndpoints {
		item = snapshotSignalEndpoint(item)
		result.SignalEndpoints[key15] = item
	}
	return result
}

func snapshotSignalActionFailure(value cli.SignalActionFailure) cli.SignalActionFailure {
	result := value
	result.CommandPath = slices.Clone(value.CommandPath)
	result.Event = snapshotSignalEvent(value.Event)
	return result
}

func snapshotSignalEndpoint(value cli.SignalEndpoint) cli.SignalEndpoint {
	result := value
	result.Policies = slices.Clone(value.Policies)
	return result
}

func snapshotSignalEndpointInspection(value cli.SignalEndpointInspection) cli.SignalEndpointInspection {
	result := value
	result.SignalEndpoint = snapshotSignalEndpoint(value.SignalEndpoint)
	result.EffectivePolicies = slices.Clone(value.EffectivePolicies)
	result.BlockedPolicies = slices.Clone(value.BlockedPolicies)
	return result
}

func snapshotSignalEvent(value cli.SignalEvent) cli.SignalEvent {
	result := value
	if value.Sender != nil {
		result.Sender = new(*value.Sender)
	}
	return result
}

func snapshotSignalPolicyInspection(value cli.SignalPolicyInspection) cli.SignalPolicyInspection {
	result := value
	result.Selector = snapshotSignalSelector(value.Selector)
	result.Actions = slices.Clone(value.Actions)
	if value.Activation != nil {
		item0 := snapshotConditionExpression(*value.Activation)
		result.Activation = &item0
	}
	return result
}

func snapshotSignalPolicyReference(value cli.SignalPolicyReference) cli.SignalPolicyReference {
	result := value
	result.SignalPolicySchema = snapshotSignalPolicySchema(value.SignalPolicySchema)
	return result
}

func snapshotSignalPolicySchema(value cli.SignalPolicySchema) cli.SignalPolicySchema {
	result := value
	result.Selector = snapshotSignalSelector(value.Selector)
	result.Actions = slices.Clone(value.Actions)
	if value.Activation != nil {
		item0 := snapshotConditionExpression(*value.Activation)
		result.Activation = &item0
	}
	return result
}

func snapshotSignalSelector(value cli.SignalSelector) cli.SignalSelector {
	result := value
	result.Signals = slices.Clone(value.Signals)
	result.ExcludedSignals = slices.Clone(value.ExcludedSignals)
	result.Origins = slices.Clone(value.Origins)
	result.ExcludedOrigins = slices.Clone(value.ExcludedOrigins)
	result.Phases = slices.Clone(value.Phases)
	return result
}

func snapshotSignalSourceInspection(value cli.SignalSourceInspection) cli.SignalSourceInspection {
	result := value
	result.ResolvedSignals = slices.Clone(value.ResolvedSignals)
	result.MissingSignals = slices.Clone(value.MissingSignals)
	return result
}

func snapshotTokenLanguageSyntax(value cli.TokenLanguageSyntax) cli.TokenLanguageSyntax {
	result := value
	result.Fragments = slices.Clone(value.Fragments)
	return result
}

func snapshotTranslationBundle(value cli.TranslationBundle) cli.TranslationBundle {
	result := value
	result.Translations = snapshotTranslations(value.Translations)
	return result
}

func snapshotTranslations(value cli.Translations) cli.Translations {
	result := maps.Clone(value)
	return result
}

func snapshotValueCandidateInspection(value cli.ValueCandidateInspection) cli.ValueCandidateInspection {
	result := value
	if value.Input != nil {
		result.Input = new(*value.Input)
	}
	if value.Representation != nil {
		item1 := snapshotValueRepresentation(*value.Representation)
		result.Representation = &item1
	}
	result.Origins = slices.Clone(value.Origins)
	result.Elements = slices.Clone(value.Elements)
	for index2 := range result.Elements {
		result.Elements[index2] = snapshotValueComponentInspection(result.Elements[index2])
	}
	result.Entries = slices.Clone(value.Entries)
	for index3 := range result.Entries {
		result.Entries[index3] = snapshotValueEntryInspection(result.Entries[index3])
	}
	result.SuppressedBy = slices.Clone(value.SuppressedBy)
	result.ConflictsWith = slices.Clone(value.ConflictsWith)
	result.DiagnosticIndexes = slices.Clone(value.DiagnosticIndexes)
	return result
}

func snapshotValueComponentInspection(value cli.ValueComponentInspection) cli.ValueComponentInspection {
	result := value
	if value.Input != nil {
		result.Input = new(*value.Input)
	}
	if value.Representation != nil {
		item1 := snapshotValueRepresentation(*value.Representation)
		result.Representation = &item1
	}
	result.Origins = slices.Clone(value.Origins)
	result.DiagnosticIndexes = slices.Clone(value.DiagnosticIndexes)
	return result
}

func snapshotValueDomainEntryInspection(value cli.ValueDomainEntryInspection) cli.ValueDomainEntryInspection {
	result := value
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotValueDomainEntryReference(value cli.ValueDomainEntryReference) cli.ValueDomainEntryReference {
	result := value
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotValueDomainEntrySchema(value cli.ValueDomainEntrySchema) cli.ValueDomainEntrySchema {
	result := value
	if value.Replacement != nil {
		result.Replacement = new(*value.Replacement)
	}
	return result
}

func snapshotValueDomainInspection(value cli.ValueDomainInspection) cli.ValueDomainInspection {
	result := value
	result.Dependencies = slices.Clone(value.Dependencies)
	result.Uses = slices.Clone(value.Uses)
	result.Entries = slices.Clone(value.Entries)
	return result
}

func snapshotValueDomainReference(value cli.ValueDomainReference) cli.ValueDomainReference {
	result := value
	result.Uses = slices.Clone(value.Uses)
	result.Entries = slices.Clone(value.Entries)
	return result
}

func snapshotValueDomainSchema(value cli.ValueDomainSchema) cli.ValueDomainSchema {
	result := value
	result.Uses = slices.Clone(value.Uses)
	result.Entries = slices.Clone(value.Entries)
	return result
}

func snapshotValueEntryInspection(value cli.ValueEntryInspection) cli.ValueEntryInspection {
	result := value
	result.Key = snapshotValueComponentInspection(value.Key)
	result.Value = snapshotValueComponentInspection(value.Value)
	return result
}

func snapshotValueInspection(value cli.ValueInspection) cli.ValueInspection {
	result := value
	result.Producers = slices.Clone(value.Producers)
	for index0 := range result.Producers {
		result.Producers[index0] = snapshotValueProducerInspection(result.Producers[index0])
	}
	result.Candidates = slices.Clone(value.Candidates)
	result.Contributors = slices.Clone(value.Contributors)
	result.BlockedBy = slices.Clone(value.BlockedBy)
	result.DiagnosticIndexes = slices.Clone(value.DiagnosticIndexes)
	if value.Representation != nil {
		item1 := snapshotValueRepresentation(*value.Representation)
		result.Representation = &item1
	}
	result.Elements = slices.Clone(value.Elements)
	result.Entries = slices.Clone(value.Entries)
	return result
}

func snapshotValueKeyedLiteral(value cli.ValueKeyedLiteral) cli.ValueKeyedLiteral {
	result := value
	result.Entries = slices.Clone(value.Entries)
	return result
}

func snapshotValueProducerInspection(value cli.ValueProducerInspection) cli.ValueProducerInspection {
	result := value
	result.Origins = slices.Clone(value.Origins)
	result.Dependencies = slices.Clone(value.Dependencies)
	result.BlockedBy = slices.Clone(value.BlockedBy)
	result.DiagnosticIndexes = slices.Clone(value.DiagnosticIndexes)
	return result
}

func snapshotValueReference(value cli.ValueReference) cli.ValueReference {
	result := value
	result.ValueSchema = snapshotValueSchema(value.ValueSchema)
	return result
}

func snapshotValueRepresentation(value cli.ValueRepresentation) cli.ValueRepresentation {
	result := value
	result.Tokens = slices.Clone(value.Tokens)
	return result
}

func snapshotValueSchema(value cli.ValueSchema) cli.ValueSchema {
	result := value
	result.EnvironmentInputs = slices.Clone(value.EnvironmentInputs)
	result.StaticDefault = snapshotValueStaticDefault(value.StaticDefault)
	result.Dependencies = slices.Clone(value.Dependencies)
	return result
}

func snapshotValueSequenceLiteral(value cli.ValueSequenceLiteral) cli.ValueSequenceLiteral {
	result := value
	result.Elements = slices.Clone(value.Elements)
	return result
}

func snapshotValueStaticDefault(value cli.ValueStaticDefault) cli.ValueStaticDefault {
	result := value
	if value.Scalar != nil {
		result.Scalar = new(*value.Scalar)
	}
	if value.Sequence != nil {
		item1 := snapshotValueSequenceLiteral(*value.Sequence)
		result.Sequence = &item1
	}
	if value.Keyed != nil {
		item2 := snapshotValueKeyedLiteral(*value.Keyed)
		result.Keyed = &item2
	}
	return result
}

func valueStateIfObservedExpectation(locator ValueLocator, states []cli.ResolutionState) InspectionExpectation {
	value := valueStateIfObservedExpectationValue{
		locator: locator.state,
		states:  slices.Clone(states),
	}
	return value
}

func (observedCommandDoesNotExistExpectation) inspectionExpectation() {}

func (expectation observedCommandDoesNotExistExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveObservedInspectionCommandLocator(expectation.locator, inspection.Commands)
	switch resolved.status {
	case locatorResolutionAbsent:
		return provedResult()
	case locatorResolutionFound:
		return disprovedResult(evaluationText("observed command"), evaluationText("no observed match"), evaluationText("matching command is observed"))
	case locatorResolutionAmbiguous:
		return disprovedResult(evaluationText("observed command"), evaluationText("no observed match"), resolved.detail)
	default:
		return disprovedResult(evaluationText("observed command"), evaluationText("valid command locator"), resolved.detail)
	}
}
