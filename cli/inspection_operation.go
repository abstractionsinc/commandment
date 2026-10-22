package cli

import (
	"cmp"
	"context"
	"slices"
)

func settleInspectionLocaleSelector(ctx context.Context, operation *invocation, grammar grammarResult) (Locale, error) {
	if operation == nil || operation.analysis == nil {
		panic("cli: Inspection locale selection requires an invocation analysis")
	}
	if operation.input.locale != "" || len(grammar.Issues) != 0 || len(grammar.IncompleteNamespaces) != 0 {
		return "", nil
	}

	model := operation.analysis.effectiveModel
	requirements := make([]dependencyRequirement, 0, 1)
	for _, command := range grammar.Selection.Path {
		for _, flag := range model.FlagsForCommand(command) {
			if !flag.localeSelector || !structuralProfileActive(grammar.SelectedProfiles, flag.command, flag.meta.profile) {
				continue
			}
			requirements = append(requirements, acceptDependencyRequirement(flag.destination, true))
		}
	}
	if len(requirements) == 0 {
		return "", nil
	}
	values, err := operation.settleDependencies(ctx, grammar, requirements)
	if err != nil {
		return "", err
	}
	selector, err := resolvedLocaleSelector(model, grammar, values)
	if err != nil {
		return "", operationDiagnosticFailure(
			operation,
			DiagnosticInvalidValue,
			"invalid locale selector",
			err,
		)
	}
	return selector, nil
}

// inspectApplicationWithSignals runs one Inspection over the canonical invocation and analysis owners.
// When a downstream semantic fact cannot be established authoritatively, Inspection reports
// InspectionStatusIncomplete rather than treating a zero or Unknown projection as Ready.
func inspectApplicationWithSignals(ctx context.Context, model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalSources []compiledSignalSource, request InspectionRequest) (Inspection, error) {
	inspection, err := inspectApplicationWithSignalsUnpreserved(ctx, model, callbacks, identity, signalSources, request)
	return inspection, preserveIndependentCallerCancellation(ctx, err)
}

func inspectApplicationWithSignalsUnpreserved(ctx context.Context, model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalSources []compiledSignalSource, request InspectionRequest) (Inspection, error) {
	operation, err := beginInspection(model, callbacks, identity, signalSources, request)
	if err != nil {
		return rejectedInspection(), projectDiagnosticFailure(DiagnosticInvalidValue, "invalid inspection request", err)
	}
	operation.invocation.beginAnalysis()
	defer func() {
		finalizeOperationFrameworkPresentation(operation.invocation, operation.invocation.input.locale, operation.invocation.analysis.effectiveModel)
	}()

	grammar, err := analyzeCommittedGrammar(ctx, operation.invocation)
	if err != nil {
		return rejectedInspection(), responseFileDiagnosticFailure(operation.invocation, err)
	}
	if code, supplyErr := operation.invocation.supplyFailure(); supplyErr != nil {
		return rejectedInspection(), operationDiagnosticFailure(operation.invocation, code, "invocation value supply failed", supplyErr)
	}

	selector, err := settleInspectionLocaleSelector(ctx, operation.invocation, grammar)
	if err != nil {
		return rejectedInspection(), err
	}
	locale, err := selectEffectiveLocale(
		string(operation.invocation.input.locale),
		string(selector),
		model.Application().localization.SourceLocale(),
	)
	if err != nil {
		return rejectedInspection(), operationDiagnosticFailure(
			operation.invocation,
			DiagnosticInvalidValue,
			"invalid invocation locale",
			err,
		)
	}
	operation.invocation.FinalizeFrameworkPresentationLocale(locale)

	referenceOptionsValue := ReferenceOptions{
		Locale:     Locale(locale),
		Visibility: ReferenceVisibilityAll,
		Order:      ReferenceOrderDeclaration,
	}
	reference, err := projectOperationReferenceProjection(operation.invocation, referenceOptionsValue)
	if err != nil {
		return Inspection{}, err
	}
	operation.result = inspectionFromReference(reference)
	result := &operation.result
	result.Locale = Locale(locale)
	result.EvaluatedRelease = reference.EvaluatedRelease
	result.Selection.Command = projectGrammarSelection(grammar.Selection)
	result.Selection.Action = projectGrammarActionSelection(grammar, reference)
	result.Sources = projectOperationSources(operation.invocation)
	result.Origins = projectOperationOrigins(operation.invocation)
	result.Tokens = projectGrammarTokens(operation.invocation, grammar, reference)
	result.Truncated = grammar.Truncated
	applyGrammarPresence(result, grammar)
	applyDefaultCommandSelections(result, grammar)
	applyArgumentFormSelection(result, grammar)
	effectiveModel := operation.invocation.analysis.effectiveModel
	applyStructuralConditionInspection(result, effectiveModel, grammar, valueAnalysis{}, false)

	providerNamespaces := operation.invocation.providerNamespaces()
	applyProviderMaterialization(result, providerNamespaces, operation.invocation.ProviderPublications())
	providerDiagnostics, providerCause := providerMaterializationDiagnostics(providerNamespaces)
	providerStart := len(result.Diagnostics)
	result.Diagnostics = append(result.Diagnostics, providerDiagnostics...)
	if len(providerDiagnostics) != 0 {
		operation.block(providerStart, len(result.Diagnostics))
		return operation.finish(false), providerDiagnosticFailure(operation.invocation, providerNamespaces, providerCause)
	}

	grammarStart := len(result.Diagnostics)
	diagnostics := grammarDiagnostics(operation.invocation, grammar, result.Tokens)
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	if len(grammar.Issues) != 0 {
		operation.block(grammarStart, len(result.Diagnostics))
		return operation.finish(false), grammarDiagnosticFailure(operation.invocation, grammar, result.Diagnostics, grammar.Truncated)
	}
	if len(grammar.IncompleteNamespaces) != 0 {
		return operation.finish(false), nil
	}

	switch result.Selection.Action.Selected {
	case InvocationActionHelp, InvocationActionVersion:
		result.ExecutionAdmission.Status = ExecutionAdmissionNotApplicable
		return operation.finish(true), nil
	case InvocationActionRun:
	case InvocationActionUnknown:
		return operation.finish(false), nil
	default:
		panic("cli: invalid Inspection action selection")
	}

	if code, supplyErr := operation.invocation.finalizeSupplies(ctx, grammar.Selection); supplyErr != nil {
		return operation.finish(false), operationDiagnosticFailure(
			operation.invocation,
			code,
			"invocation value supply failed",
			supplyErr,
		)
	}

	demandedDomains := demandedDynamicDomainsForValueAnalysis(operation.invocation, grammar)
	domainOutcomes := operation.invocation.materializeDynamicDomains(ctx, grammar, demandedDomains)
	applyDynamicDomainMaterialization(result, operation.invocation, effectiveModel, Locale(locale), domainOutcomes)
	domainDiagnostics, domainCause := dynamicDomainMaterializationDiagnostics(domainOutcomes)
	domainStart := len(result.Diagnostics)
	result.Diagnostics = append(result.Diagnostics, domainDiagnostics...)
	if len(domainDiagnostics) != 0 {
		operation.block(domainStart, len(result.Diagnostics))
		return operation.finish(false), dynamicDomainDiagnosticFailure(operation.invocation, domainOutcomes, domainCause)
	}
	if !dynamicDomainOutcomesComplete(domainOutcomes, demandedDomains) {
		return operation.finish(false), nil
	}

	valueAnalysis := analyzeValues(ctx, operation.invocation, grammar)
	enforceEffectiveDomainMembership(effectiveModel, &valueAnalysis)
	valuesComplete := projectValueAnalysis(result, effectiveModel, valueAnalysis) && !valueAnalysisIncomplete(valueAnalysis)
	valueStart := len(result.Diagnostics)
	valueDiagnostics := applyValueDiagnostics(result, effectiveModel, valueAnalysis)
	result.Diagnostics = append(result.Diagnostics, valueDiagnostics...)
	if len(valueDiagnostics) != 0 {
		operation.block(valueStart, len(result.Diagnostics))
		return operation.finish(false), valueDiagnosticFailure(
			operation.invocation,
			effectiveModel,
			valueAnalysis,
			valueDiagnostics,
			grammar.Truncated,
		)
	}
	if !valuesComplete {
		return operation.finish(false), nil
	}

	structuralComplete := applyStructuralConditionInspection(result, effectiveModel, grammar, valueAnalysis, true)
	if !structuralComplete || !inspectionPostValueComplete(effectiveModel, grammar) {
		return operation.finish(false), nil
	}

	policyAnalysis := analyzePolicy(
		ctx,
		operation.invocation,
		grammar.Selection,
		grammar.SelectedProfiles,
		valueAnalysis,
		policyContextProjector(effectiveModel, valueAnalysis),
	)
	policyStart := len(result.Diagnostics)
	policyDiagnostics, policyCause := applyPolicyAnalysis(
		result,
		operation.invocation,
		effectiveModel,
		Locale(locale),
		policyAnalysis,
	)
	result.Diagnostics = append(result.Diagnostics, policyDiagnostics...)
	result.Notices = append(result.Notices, projectLifecycleNoticeInspections(effectiveModel, grammar, valueAnalysis, policyAnalysis)...)
	result.Truncated = result.Truncated || policyAnalysis.Truncated
	if policyAnalysis.Unsupported {
		return operation.finish(false), nil
	}
	if policyAnalysis.Blocked || len(policyDiagnostics) != 0 {
		operation.block(policyStart, len(result.Diagnostics))
		return operation.finish(false), policyDiagnosticFailure(
			operation.invocation,
			effectiveModel,
			policyDiagnostics,
			policyAnalysis.Truncated,
			policyCause,
		)
	}

	signalStart := len(result.Diagnostics)
	signalDiagnostics, signalsComplete, signalCause := applySignalInspection(
		result,
		operation,
		signalSources,
		effectiveModel,
		grammar.Selection,
		grammar.SelectedProfiles,
		valueAnalysis,
	)
	result.Diagnostics = append(result.Diagnostics, signalDiagnostics...)
	if len(signalDiagnostics) != 0 {
		operation.block(signalStart, len(result.Diagnostics))
		return operation.finish(false), signalInspectionDiagnosticFailure(operation.invocation, signalDiagnostics, signalCause)
	}
	if !signalsComplete {
		return operation.finish(false), nil
	}

	projectStaticExecution(result, effectiveModel, grammar.Selection)
	return operation.finish(true), nil
}

func (operation *inspectionOperation) block(start, end int) {
	if operation == nil || operation.blocked || end <= start {
		return
	}
	operation.blocked = true
	operation.blockingStart = start
	operation.blockingEnd = end
}

func (operation *inspectionOperation) finish(ready bool) Inspection {
	if operation == nil {
		panic("cli: nil inspection operation")
	}
	if operation.blocked {
		operation.result.Status = InspectionStatusRejected
		operation.result.ExecutionAdmission.Status = ExecutionAdmissionBlocked
		operation.result.ExecutionAdmission.BlockingDiagnosticIndexes = operation.result.ExecutionAdmission.BlockingDiagnosticIndexes[:0]
		for index := operation.blockingStart; index < operation.blockingEnd; index++ {
			operation.result.ExecutionAdmission.BlockingDiagnosticIndexes = append(operation.result.ExecutionAdmission.BlockingDiagnosticIndexes, index)
		}
		return operation.result
	}
	if ready {
		operation.result.Status = InspectionStatusReady
		return operation.result
	}
	operation.result.Status = InspectionStatusIncomplete
	return operation.result
}

func rejectedInspection() Inspection {
	inspection := Inspection{
		Status: InspectionStatusRejected,
	}
	return inspection
}

func dynamicDomainOutcomesComplete(outcomes map[int]dynamicDomainOutcome, demanded []int) bool {
	for _, index := range demanded {
		outcome, ok := outcomes[index]
		if !ok || outcome.status != dynamicDomainOutcomeReady {
			return false
		}
	}
	return true
}

func providerMaterializationDiagnosticCode(status providerOutcomeStatus) (DiagnosticCode, bool) {
	switch status {
	case providerOutcomeBuilderMisuse:
		return DiagnosticBuilderMisuse, true
	case providerOutcomeRejected:
		return DiagnosticInvalidCommandBatch, true
	case providerOutcomeFailed:
		return DiagnosticCommandProviderFailure, true
	case providerOutcomeCanceled:
		return DiagnosticCommandProviderCancellation, true
	default:
		return "", false
	}
}

func providerMaterializationDiagnostics(namespaces map[CommandID]providerNamespaceOutcome) ([]Diagnostic, error) {
	owners := make([]CommandID, 0, len(namespaces))
	for owner := range namespaces {
		owners = append(owners, owner)
	}
	slices.SortFunc(owners, func(left, right CommandID) int { return cmp.Compare(left.uint32(), right.uint32()) })
	var diagnostics []Diagnostic
	var cause error
	for _, owner := range owners {
		for _, outcome := range namespaces[owner].results {
			code, ok := providerMaterializationDiagnosticCode(outcome.status)
			if !ok {
				continue
			}
			diagnosticValue := Diagnostic{
				Code:     code,
				Command:  CommandID(owner.uint32()),
				Provider: CommandProviderID(outcome.provider.uint32()),
			}
			diagnostics = append(diagnostics, diagnosticValue)
			if cause == nil && outcome.err != nil {
				cause = outcome.err
			}
		}
	}
	return diagnostics, cause
}

func providerDiagnosticFailure(operation *invocation, namespaces map[CommandID]providerNamespaceOutcome, cause error) error {
	owners := make([]CommandID, 0, len(namespaces))
	for owner := range namespaces {
		owners = append(owners, owner)
	}
	slices.SortFunc(owners, func(left, right CommandID) int { return cmp.Compare(left.uint32(), right.uint32()) })
	payload := operationDiagnosticPayload(operation, false)
	for _, owner := range owners {
		for _, outcome := range namespaces[owner].results {
			code, ok := providerMaterializationDiagnosticCode(outcome.status)
			if !ok {
				continue
			}
			diagnosticValue := diagnostic{
				Code:     string(code),
				Command:  owner,
				Provider: outcome.provider,
			}
			payload.Diagnostics = append(payload.Diagnostics, diagnosticValue)
		}
	}
	return retainDiagnosticFailure(payload, cause)
}

func applyProviderMaterialization(result *Inspection, namespaces map[CommandID]providerNamespaceOutcome, publications []providerPublication) {
	providerCommands := make(map[CommandID]struct{})
	acceptedEntries := acceptedProviderEntries(publications)
	for owner, materialized := range namespaces {
		publicOwner := CommandID(owner.uint32())
		if command, ok := result.Commands[publicOwner]; ok {
			if materialized.complete {
				command.NamespaceCompleteness = CommandNamespaceComplete
			} else {
				command.NamespaceCompleteness = CommandNamespaceIncomplete
			}
			result.Commands[publicOwner] = command
		}
		for _, outcome := range materialized.results {
			id := CommandProviderID(outcome.provider.uint32())
			provider, ok := result.CommandProviders[id]
			if !ok {
				continue
			}
			switch outcome.status {
			case providerOutcomeReady:
				provider.Outcome = CommandProviderOutcomeAccepted
			case providerOutcomeBuilderMisuse, providerOutcomeRejected:
				provider.Outcome = CommandProviderOutcomeRejected
			case providerOutcomeFailed:
				provider.Outcome = CommandProviderOutcomeFailed
			case providerOutcomeCanceled:
				provider.Outcome = CommandProviderOutcomeCanceled
			case providerOutcomeBlocked:
				provider.Outcome = CommandProviderOutcomeBlocked
			default:
				provider.Outcome = CommandProviderOutcomeUnknown
			}
			if outcome.batch.localization != nil {
				localization := publicLocalization(*outcome.batch.localization)
				provider.Localization = &localization
			}
			entries := acceptedEntries[outcome.provider]
			if outcome.status != providerOutcomeReady {
				entries = nil
			}
			provider.CommandEntries = make([]CommandEntryID, len(entries))
			for index, entry := range entries {
				publicEntry := CommandEntryID(entry.uint32())
				provider.CommandEntries[index] = publicEntry
				if projected, exists := result.CommandEntries[publicEntry]; exists {
					projected.Provider = id
					projected.Origin = DeclarationOriginProvider
					result.CommandEntries[publicEntry] = projected
					if projected.Kind == CommandEntryCanonical && len(projected.TargetPath) != 0 {
						providerCommands[projected.TargetPath[len(projected.TargetPath)-1]] = struct{}{}
					}
				}
			}
			result.CommandProviders[id] = provider
		}
	}
	markProviderDeclarationOrigins(result, providerCommands)
}

func markProviderDeclarationOrigins(result *Inspection, providerCommands map[CommandID]struct{}) {
	if result == nil || len(providerCommands) == 0 {
		return
	}
	changed := true
	for changed {
		changed = false
		for id, command := range result.Commands {
			if _, already := providerCommands[id]; already {
				continue
			}
			if _, parentDynamic := providerCommands[command.Parent]; parentDynamic {
				providerCommands[id] = struct{}{}
				changed = true
			}
		}
	}
	for id := range providerCommands {
		command, ok := result.Commands[id]
		if !ok {
			continue
		}
		command.Origin = DeclarationOriginProvider
		for index := range command.Advisories {
			command.Advisories[index].Origin = DeclarationOriginProvider
		}
		for index := range command.NoticeSuppressions {
			command.NoticeSuppressions[index].Origin = DeclarationOriginProvider
		}
		result.Commands[id] = command
	}
	for id, entry := range result.CommandEntries {
		if _, dynamic := providerCommands[entry.Parent]; dynamic {
			entry.Origin = DeclarationOriginProvider
			result.CommandEntries[id] = entry
		}
	}
	for id, item := range result.CommandDefaults {
		if _, dynamic := providerCommands[item.Parent]; dynamic {
			item.Origin = DeclarationOriginProvider
			result.CommandDefaults[id] = item
		}
	}
	providerFlags := make(map[FlagID]struct{})
	for id, flag := range result.Flags {
		if _, dynamic := providerCommands[flag.Command]; dynamic {
			flag.Origin = DeclarationOriginProvider
			result.Flags[id] = flag
			providerFlags[id] = struct{}{}
		}
	}
	for id, spelling := range result.FlagSpellings {
		if _, dynamic := providerFlags[spelling.Flag]; dynamic {
			spelling.Origin = DeclarationOriginProvider
			result.FlagSpellings[id] = spelling
		}
	}
	providerForms := make(map[ArgumentFormID]struct{})
	for id, form := range result.ArgumentForms {
		if _, dynamic := providerCommands[form.Command]; dynamic {
			form.Origin = DeclarationOriginProvider
			result.ArgumentForms[id] = form
			providerForms[id] = struct{}{}
		}
	}
	for id, term := range result.ArgumentTerms {
		if _, dynamic := providerForms[term.Form]; dynamic {
			term.Origin = DeclarationOriginProvider
			result.ArgumentTerms[id] = term
		}
	}
	for id, policy := range result.SignalPolicies {
		if _, dynamic := providerCommands[policy.Command]; dynamic {
			policy.Origin = DeclarationOriginProvider
			result.SignalPolicies[id] = policy
		}
	}
}

func cloneRuleOperands(input []RuleOperand) []RuleOperand {
	if input == nil {
		return nil
	}
	result := make([]RuleOperand, len(input))
	for i, operand := range input {
		result[i].Sensitive = operand.Sensitive
		if operand.Literal != nil {
			literal := *operand.Literal
			result[i].Literal = &literal
		}
	}
	return result
}

type inspectionOperation struct {
	invocation *invocation

	signalCapabilities SignalHostCapabilities

	result Inspection

	blocked       bool
	blockingStart int
	blockingEnd   int
}

func beginInspection(model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalSources []compiledSignalSource, request InspectionRequest) (*inspectionOperation, error) {
	input, err := snapshotInvocationInput(request.Args, request.Env, request.WorkingDirectory, request.ResponseFiles, request.Defaults, request.Facts, request.Locale, request.Limits)
	if err != nil {
		return nil, err
	}
	description, err := signalSourceDescription(signalSources)
	if err != nil {
		return nil, err
	}
	invocation, err := beginInvocation(model, callbacks, identity, description, input)
	if err != nil {
		return nil, err
	}
	operation := &inspectionOperation{
		invocation:         invocation,
		signalCapabilities: request.SignalHostCapabilities,
	}
	return operation, nil
}
