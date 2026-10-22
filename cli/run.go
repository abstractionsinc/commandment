package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type frontendActionError struct {
	operation string
	cause     error
}

func (err *frontendActionError) Error() string { return err.operation + ": " + err.cause.Error() }

func (err *frontendActionError) Unwrap() error { return err.cause }

func writeFrameworkOutput(operation *invocation, writer io.Writer, text, subject string) error {
	if _, err := io.WriteString(writer, text); err != nil {
		failure := &frontendActionError{
			operation: "write " + subject + " output",
			cause:     err,
		}
		return operationDiagnosticFailure(
			operation,
			DiagnosticOutputWriteFailure,
			subject+" output write failure",
			failure,
		)
	}
	return nil
}

// executeSelectedFrontendAction dispatches frontend-owned actions before
// value, policy, and execution preparation so they do not demand unrelated
// application runtime machinery.
func executeSelectedFrontendAction(ctx context.Context, snapshot *runState, invocation *invocation, model *effectiveModel, grammar grammarResult) error {
	switch grammar.Selection.ActionKind {
	case intrinsicActionHelp:
		return executeHelpAction(ctx, snapshot, invocation, model, grammar)
	case intrinsicActionVersion:
		return executeVersionAction(ctx, snapshot, model)
	default:
		panic("cli: selected frontend action has invalid intrinsic kind")
	}
}

func executeVersionAction(ctx context.Context, snapshot *runState, model *effectiveModel) error {
	actions, ok := model.IntrinsicActions()
	if !ok {
		panic("cli: selected Version action has no accepted intrinsic actions")
	}
	version := actions.Version()
	if !version.Present() || version.Disabled() {
		panic("cli: selected Version action is not enabled")
	}
	if snapshot.stdout == nil {
		panic("cli: Version selected without an output writer")
	}
	profile := profileForWriter(snapshot.stdout, destinationStdoutInformational)
	document := document{
		Destination: destinationStdoutInformational,
		Blocks: []block{{
			Kind:  blockLiteralLine,
			Lines: []string{version.Text()},
		}},
	}
	text, err := render(ctx, document, profile)
	if err != nil {
		failure := &frontendActionError{
			operation: "render version output",
			cause:     err,
		}
		return failure
	}
	return writeFrameworkOutput(snapshot.invocation, snapshot.stdout, text, "version")
}

func executeShellCompletionExport(snapshot *runState, model *effectiveModel, grammar grammarResult, catalog shellCompletionCatalog) error {
	if snapshot.stdout == nil {
		panic("cli: shell completion export selected without an output writer")
	}
	shell, ok := frontendDestinationToken(grammar, catalog.ShellDestination())
	if !ok {
		panic("cli: selected shell completion export has no shell destination value")
	}
	provider, ok := catalog.Provider(shell)
	if !ok {
		return operationDiagnosticFailure(
			snapshot.invocation,
			DiagnosticInvalidValue,
			"invalid shell completion provider",
			errInvalidShellCompletionProvider,
		)
	}
	version := ""
	for _, occurrence := range grammar.Occurrences {
		if occurrence.Flag != catalog.VersionFlag() {
			continue
		}
		if occurrence.Attached != "" {
			version = occurrence.Attached
			break
		}
		for _, index := range occurrence.ValueIndices {
			for _, token := range grammar.Tokens {
				if token.PhysicalIndex == index && token.Ownership.Flag == occurrence.Flag &&
					token.Ownership.Role == tokenRoleFlagValue {
					version = token.Text
					break
				}
			}
		}
	}
	text, err := generateCompiledShellCompletionScript(model.Application().name, provider, version)
	if err != nil {
		return operationDiagnosticFailure(
			snapshot.invocation,
			DiagnosticInvalidValue,
			"shell completion generation failed",
			err,
		)
	}
	return writeFrameworkOutput(snapshot.invocation, snapshot.stdout, text, "shell completion")
}

func frontendDestinationToken(grammar grammarResult, destination ValueID) (string, bool) {
	for _, token := range grammar.Tokens {
		if token.Ownership.Destination == destination && token.Ownership.Role == tokenRolePositional {
			return token.Text, true
		}
	}
	return "", false
}

func policyFailureDetails(model *effectiveModel, analysis policyAnalysis) ([]Diagnostic, error) {
	diagnostics := make([]Diagnostic, 0)
	causes := make([]error, 0)
	for _, id := range analysis.RuleOrder {
		runtime, ok := analysis.Rules[id]
		if !ok {
			continue
		}
		record, ok := model.Rule(id)
		if !ok {
			return nil, errInvalidModelTopology
		}
		definition := record.definition
		command := record.command
		committed := analysis.RuleDiagnosticCounts[id]
		switch runtime.Outcome.Kind {
		case ruleOutcomeViolated:
			for index, violation := range runtime.Outcome.Violations {
				if index >= committed {
					break
				}
				diagnostic, err := policyRuleDiagnostic(model, record.meta.id, definition, violation)
				if err != nil {
					return nil, err
				}
				diagnostic.Command = command
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
				diagnostic.Command = command
				diagnostics = append(diagnostics, diagnostic)
			}
			if runtime.Outcome.Err != nil {
				causes = append(causes, runtime.Outcome.Err)
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
				diagnostic.Command = command
				diagnostics = append(diagnostics, diagnostic)
			}
			if runtime.Outcome.Err != nil {
				causes = append(causes, runtime.Outcome.Err)
			}
		case ruleOutcomeInvalid, ruleOutcomeSatisfied, ruleOutcomeUpstreamBlocked:
		}
	}
	for _, id := range analysis.AdvisoryOrder {
		runtime, ok := analysis.Advisories[id]
		if !ok || runtime.Status != advisoryEvaluationFailed {
			continue
		}
		record, ok := model.Advisory(id)
		if !ok {
			return nil, errInvalidModelTopology
		}
		if _, ok := analysis.AdvisoryDiagnostics[id]; ok {
			diagnosticValue := Diagnostic{
				Code:         DiagnosticAdvisoryEvaluationFailure,
				AdvisoryCode: AdvisoryCode(record.definition.Code()),
				Command:      record.command,
			}
			diagnostics = append(diagnostics, diagnosticValue)
		}
		if runtime.Err != nil {
			causes = append(causes, runtime.Err)
		}
	}
	return diagnostics, errors.Join(causes...)
}

func runApplication(ctx context.Context, model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, shellCompletion shellCompletionCatalog, signalSources []compiledSignalSource, request RunRequest) (returned error) {
	defer func() {
		returned = preserveIndependentCallerCancellation(ctx, returned)
	}()
	run, err := beginRun(model, callbacks, identity, signalSources, request)
	if err != nil {
		return projectDiagnosticFailure(DiagnosticInvalidValue, "invalid invocation", err)
	}
	run.invocation.beginAnalysis()
	defer func() {
		// A nonempty request locale is already the highest-precedence locale and
		// can safely establish presentation context for failures that occur before
		// ordinary execution-access realization. With an empty request locale,
		// the application locale selector may still participate later; do not
		// guess a fallback locale for failures that prevent that selection.
		if run.invocation.input.locale != "" {
			finalizeOperationFrameworkPresentation(run.invocation, run.invocation.input.locale, run.invocation.analysis.effectiveModel)
		}
	}()
	// Shell-completion export is a frontend intrinsic over wholly static grammar.
	// Recognize the direct intrinsic before demanding operation semantics so
	// application Facts, defaults, command providers, dynamic domains, or Signal
	// runtime are not demanded merely to generate an adapter artifact.
	staticGrammar := analyzeGrammar(ctx, run.invocation)
	if shellCompletion.Selected(staticGrammar.Selection.Selected) &&
		len(staticGrammar.Issues) == 0 && len(staticGrammar.IncompleteNamespaces) == 0 {
		return executeShellCompletionExport(
			run, run.invocation.analysis.effectiveModel, staticGrammar, shellCompletion,
		)
	}
	grammar, err := analyzeCommittedGrammar(ctx, run.invocation)
	if err != nil {
		return responseFileDiagnosticFailure(run.invocation, err)
	}
	if code, err := run.invocation.supplyFailure(); err != nil {
		return operationDiagnosticFailure(run.invocation, code, "invocation value supply failed", err)
	}
	providerNamespaces := run.invocation.providerNamespaces()
	providerDiagnostics, providerCause := providerMaterializationDiagnostics(providerNamespaces)
	if len(providerDiagnostics) != 0 {
		return providerDiagnosticFailure(run.invocation, providerNamespaces, providerCause)
	}
	if len(grammar.IncompleteNamespaces) != 0 {
		dependencyValues := operationDependencySeed(run.invocation.analysis)
		if diagnostics := runValueDiagnostics(run.invocation.analysis.effectiveModel, dependencyValues); len(diagnostics) != 0 {
			return valueDiagnosticFailure(run.invocation, run.invocation.analysis.effectiveModel, dependencyValues, diagnostics, grammar.Truncated)
		}
		return runProviderNamespaceBlockedFailure(
			run.invocation,
			run.invocation.analysis.effectiveModel,
			dependencyValues,
			providerNamespaces,
			grammar.IncompleteNamespaces,
		)
	}
	if len(grammar.Issues) != 0 {
		return runGrammarDiagnosticFailure(run.invocation, grammar)
	}
	effectiveModel := run.invocation.analysis.effectiveModel
	if shellCompletion.Selected(grammar.Selection.Selected) {
		return executeShellCompletionExport(run, effectiveModel, grammar, shellCompletion)
	}
	if grammar.Selection.ActionSelected() {
		return executeSelectedFrontendAction(ctx, run, run.invocation, effectiveModel, grammar)
	}
	if code, err := run.invocation.finalizeSupplies(ctx, grammar.Selection); err != nil {
		return operationDiagnosticFailure(run.invocation, code, "invocation value supply failed", err)
	}

	demandedDomains := demandedDynamicDomainsForValueAnalysis(run.invocation, grammar)
	domainOutcomes := run.invocation.materializeDynamicDomains(ctx, grammar, demandedDomains)
	domainDiagnostics, domainCause := dynamicDomainMaterializationDiagnostics(domainOutcomes)
	if len(domainDiagnostics) != 0 {
		return dynamicDomainDiagnosticFailure(run.invocation, domainOutcomes, domainCause)
	}
	if !dynamicDomainOutcomesComplete(domainOutcomes, demandedDomains) {
		dependencyValues := operationDependencySeed(run.invocation.analysis)
		if diagnostics := runValueDiagnostics(run.invocation.analysis.effectiveModel, dependencyValues); len(diagnostics) != 0 {
			return valueDiagnosticFailure(run.invocation, run.invocation.analysis.effectiveModel, dependencyValues, diagnostics, grammar.Truncated)
		}
		return runDynamicDomainBlockedFailure(
			run.invocation,
			run.invocation.analysis.effectiveModel,
			dependencyValues,
			domainOutcomes,
			demandedDomains,
		)
	}
	effectiveModel = run.invocation.analysis.effectiveModel
	values := analyzeValues(ctx, run.invocation, grammar)
	enforceEffectiveDomainMembership(effectiveModel, &values)
	if diagnostics := runValueDiagnostics(effectiveModel, values); len(diagnostics) != 0 {
		return valueDiagnosticFailure(run.invocation, effectiveModel, values, diagnostics, grammar.Truncated)
	}
	if values.Unsupported {
		panic("cli: finalized Run value analysis is unsupported")
	}
	if blocked := runBlockedValueDestinations(values); len(blocked) != 0 {
		return blockedDependencyFailure(run.invocation, effectiveModel, values, blocked)
	}
	if !inspectionPostValueComplete(effectiveModel, grammar) {
		panic("cli: finalized Run selection is absent from effective model")
	}

	configurations, ok := prepareExecutionConfigurations(effectiveModel, grammar.Selection.Path)
	if !ok {
		panic("cli: finalized Run path cannot create execution configurations")
	}
	run.configurations = configurations
	signalPreparation, primaryCtx, signalCallbackCtx, signalErr := prepareRunSignals(
		ctx,
		run,
		run.signalSources,
		effectiveModel,
		grammar.Selection,
		grammar.SelectedProfiles,
		values,
	)
	if !signalPreparation.supported {
		panic("cli: finalized Run signal plan is unsupported")
	}
	if signalErr != nil {
		return signalErr
	}
	signalState := signalPreparation.state
	run.signals = signalState
	analysisCtx := ctx
	if signalState != nil {
		if err := signalState.runtime.Start(signalCallbackCtx); err != nil {
			return finalizeRunFailures(ctx, signalState, run.invocation, err, nil)
		}
		analysisCtx = primaryCtx
	}

	policy := analyzePolicy(
		analysisCtx,
		run.invocation,
		grammar.Selection,
		grammar.SelectedProfiles,
		values,
		policyContextProjector(effectiveModel, values),
	)
	if policy.Blocked {
		diagnostics, cause := policyFailureDetails(effectiveModel, policy)
		return finalizeRunFailures(
			ctx,
			signalState,
			run.invocation,
			policyDiagnosticFailure(run.invocation, effectiveModel, diagnostics, policy.Truncated, cause),
			nil,
		)
	}
	if signalState != nil && primaryCtx.Err() != nil {
		return finalizeRunFailures(ctx, signalState, run.invocation, context.Cause(primaryCtx), nil)
	}
	if policy.Unsupported {
		panic("cli: finalized Run policy is unsupported")
	}
	locale, _, _, err := realizeExecutionAccess(run, effectiveModel, grammar, values)
	if err != nil {
		return finalizeRunFailures(ctx, signalState, run.invocation, err, nil)
	}
	run.invocation.FinalizeFrameworkPresentationLocale(string(locale))
	if signalState != nil && primaryCtx.Err() != nil {
		return finalizeRunFailures(ctx, signalState, run.invocation, context.Cause(primaryCtx), nil)
	}
	attempts, err := buildAdvisoryNoticeDeliveryAttempts(run.invocation, effectiveModel, locale, policy)
	if err != nil {
		panic("cli: finalized advisory Notice projection is inconsistent")
	}
	lifecycleAttempts := buildLifecycleNoticeDeliveryAttempts(effectiveModel, grammar, values, policy)
	for index := range lifecycleAttempts {
		lifecycleAttempts[index].Index += len(policy.Notices)
	}
	attempts = append(attempts, lifecycleAttempts...)
	if signalState != nil && primaryCtx.Err() != nil {
		return finalizeRunFailures(ctx, signalState, run.invocation, context.Cause(primaryCtx), nil)
	}

	selected, ok := effectiveModel.Command(grammar.Selection.Selected)
	if !ok {
		panic("cli: finalized Run selection has no effective command")
	}
	handler, ok := selected.execution.Handler()
	if !ok {
		return finalizeRunFailures(
			ctx,
			signalState,
			run.invocation,
			notRunnableCommandDiagnosticFailure(run.invocation, grammar.Selection.Selected),
			nil,
		)
	}
	handlerConfiguration, ok := run.configurations.configuration(grammar.Selection.Selected)
	if !ok {
		panic("cli: selected Run command has no execution configuration")
	}

	middleware := make([]func(context.Context, func(context.Context) error) error, 0)
	for _, command := range grammar.Selection.Path {
		record, ok := effectiveModel.Command(command)
		if !ok {
			panic("cli: finalized Run path contains no effective command")
		}
		configuration, ok := run.configurations.configuration(command)
		if !ok {
			panic("cli: selected Run path command has no execution configuration")
		}
		for _, item := range record.execution.Middleware() {
			callback := item
			boundConfiguration := configuration
			middleware = append(middleware, func(callCtx context.Context, next func(context.Context) error) error {
				return callback(callCtx, boundConfiguration, next)
			})
		}
	}
	prepared := bindExecution(
		middleware,
		func(callCtx context.Context) error { return handler(callCtx, handlerConfiguration) },
		run.invocation.FrameworkPresentation(),
	)

	var preparedSignals preparedSignalExecution
	if signalState != nil {
		if primaryCtx.Err() != nil {
			return finalizeRunFailures(ctx, signalState, run.invocation, context.Cause(primaryCtx), nil)
		}
		preparedSignals, err = signalState.prepareExecutionTransition(primaryCtx, run)
		if err != nil {
			if primaryCtx.Err() != nil {
				return finalizeRunFailures(ctx, signalState, run.invocation, context.Cause(primaryCtx), nil)
			}
			signalState.mu.Lock()
			signalState.runtimeFailures = append(signalState.runtimeFailures, err)
			signalState.mu.Unlock()
			return finalizeRunFailures(ctx, signalState, run.invocation, nil, nil)
		}
	}
	if signalState != nil && primaryCtx.Err() != nil {
		return finalizeRunFailures(ctx, signalState, run.invocation, context.Cause(primaryCtx), nil)
	}

	run.configurations.handoff()
	if signalState != nil {
		signalState.enterExecutionPhase()
		preparedSignals.commit()
	}

	noticeCtx := ctx
	if signalState != nil {
		noticeCtx = primaryCtx
	}
	noticeFailures := deliverNotices(noticeCtx, run.invocation, run.notify, attempts)

	executionCtx := ctx
	var beginCleanup func()
	if signalState != nil {
		executionCtx = signalState.executionContext(primaryCtx)
		beginCleanup = func() { signalState.beginCleanup(ctx) }
	}
	if signalState != nil {
		err = executeSignaled(executionCtx, prepared, beginCleanup)
	} else {
		err = execute(executionCtx, prepared)
	}
	if signalState != nil {
		// No-middleware execution reaches cleanup through the handler wrapper;
		// this call is an idempotent safety net for pre-handler protocol exits.
		signalState.beginCleanup(ctx)
	}
	return finalizeRunFailures(ctx, signalState, run.invocation, err, noticeFailures)
}

func runProviderNamespaceBlockedFailure(operation *invocation, model *effectiveModel, values valueAnalysis, namespaces map[CommandID]providerNamespaceOutcome, incomplete []CommandID) error {
	blocked := make([]ValueID, 0)
	for _, owner := range incomplete {
		outcome, ok := namespaces[owner]
		if !ok {
			panic("cli: incomplete provider namespace has no settled outcome")
		}
		for _, provider := range outcome.results {
			if provider.status != providerOutcomeBlocked {
				continue
			}
			for _, destination := range provider.blockedBy {
				blocked = appendUniqueDestinationID(blocked, destination)
			}
		}
	}
	if len(blocked) != 0 {
		return blockedDependencyFailure(operation, model, values, blocked)
	}

	payload := operationDiagnosticPayload(operation, false)
	seen := make(map[CommandID]struct{}, len(incomplete))
	for _, command := range incomplete {
		if !command.present() {
			panic("cli: incomplete provider namespace has invalid command identity")
		}
		if _, duplicate := seen[command]; duplicate {
			continue
		}
		seen[command] = struct{}{}
		payload.Diagnostics = append(payload.Diagnostics, diagnostic{
			Code:    string(DiagnosticCommandTargetBlocked),
			Command: command,
		})
	}
	if len(payload.Diagnostics) == 0 {
		panic("cli: provider namespace is incomplete without a blocking cause")
	}
	return retainDiagnosticFailure(payload, nil)
}

func runDynamicDomainBlockedFailure(operation *invocation, model *effectiveModel, values valueAnalysis, outcomes map[int]dynamicDomainOutcome, demanded []int) error {
	blocked := make([]ValueID, 0)
	for _, index := range demanded {
		outcome, ok := outcomes[index]
		if !ok {
			panic("cli: demanded dynamic domain has no settled outcome")
		}
		if outcome.status != dynamicDomainOutcomeBlocked {
			panic("cli: demanded dynamic domain is incomplete without a blocked outcome")
		}
		for _, destination := range outcome.blockedBy {
			blocked = appendUniqueDestinationID(blocked, destination)
		}
	}
	if len(blocked) == 0 {
		panic("cli: blocked dynamic domain has no blocking dependency")
	}
	return blockedDependencyFailure(operation, model, values, blocked)
}

func runBlockedValueDestinations(values valueAnalysis) []ValueID {
	blocked := make([]ValueID, 0)
	for _, issue := range values.Issues {
		switch issue.Kind {
		case valueIssueNone, valueIssueInvalidInput, valueIssueResolutionConflict, valueIssueProducerFailure:
		case valueIssueProducerBlocked:
			blocked = appendUniqueDestinationID(blocked, issue.Destination)
		case valueIssueUnsupported:
			panic("cli: finalized Run value analysis contains unsupported state")
		}
	}
	for destination, resolution := range values.Destinations {
		if resolution.State == destinationStateBlocked && len(resolution.BlockedBy) == 0 {
			blocked = appendUniqueDestinationID(blocked, destination)
		}
	}
	return blocked
}

func blockedDependencyFailure(operation *invocation, model *effectiveModel, values valueAnalysis, blocked []ValueID) error {
	payload := operationDiagnosticPayload(operation, false)
	seen := make(map[ValueID]struct{})
	var visit func(ValueID)
	visit = func(destination ValueID) {
		if _, duplicate := seen[destination]; duplicate {
			return
		}
		seen[destination] = struct{}{}

		resolution, resolved := values.Destinations[destination]
		if resolved && resolution.State == destinationStateBlocked && len(resolution.BlockedBy) != 0 {
			for _, upstream := range resolution.BlockedBy {
				visit(upstream)
			}
			return
		}
		if resolved {
			switch resolution.State {
			case destinationStateResolved:
				return
			case destinationStateFailed, destinationStateInvalid:
				panic("cli: blocked Run dependency reached an unprojected failed destination")
			case destinationStateAbsent, destinationStateInactive, destinationStateBlocked:
			default:
				panic("cli: blocked Run dependency has invalid destination state")
			}
		}

		record, ok := model.Destination(destination)
		if !ok {
			panic("cli: blocked Run dependency is absent from effective model")
		}
		definition, ok := record.Definition()
		if !ok {
			panic("cli: blocked Run dependency has no destination definition")
		}
		code := DiagnosticMissingValue
		if definition.Kind() == destinationKindFactRequired {
			code = DiagnosticMissingFact
		}
		payload.Diagnostics = append(payload.Diagnostics, diagnostic{
			Code:    string(code),
			Command: record.command,
		})
	}
	for _, destination := range blocked {
		visit(destination)
	}
	if len(payload.Diagnostics) == 0 {
		panic("cli: blocked Run semantic state has no causal diagnostic")
	}
	return retainDiagnosticFailure(payload, nil)
}

func finalizeRunFailures(ctx context.Context, signals *runSignals, operation *invocation, primary error, noticeFailures []error) error {
	set := runFailures{
		primary:        primary,
		noticeFailures: append([]error(nil), noticeFailures...),
	}
	if signals != nil {
		set = signals.settle(ctx, operation, set.primary)
		set.noticeFailures = append([]error(nil), noticeFailures...)
	}
	return set.Error()
}

func runValueDiagnostics(model *effectiveModel, analysis valueAnalysis) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(analysis.Issues))
	for _, issue := range analysis.Issues {
		if issue.Kind == valueIssueUnsupported || issue.Kind == valueIssueProducerBlocked {
			continue
		}
		diagnostic, _ := publicValueIssueDiagnostic(model, analysis, issue)
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics
}

func notRunnableCommandDiagnosticFailure(operation *invocation, command CommandID) error {
	failurePayload := payload{
		Diagnostics: []diagnostic{{
			Code:             string(DiagnosticNotRunnableCommand),
			Message:          "selected command is not runnable",
			FrameworkMessage: true,
			Command:          command,
		}},
		Presentation: operation.FrameworkPresentation(),
	}
	return retainDiagnosticFailure(failurePayload, nil)
}

func runGrammarDiagnosticFailure(operation *invocation, grammar grammarResult) error {
	diagnostics := grammarDiagnostics(operation, grammar, nil)
	return grammarDiagnosticFailure(operation, grammar, diagnostics, grammar.Truncated)
}

// runFailures records already-settled returned failures in canonical
// cross-domain precedence. It is private implementation vocabulary, not a
// public business-failure model.
type runFailures struct {
	terminal                 error
	primary                  error
	noticeFailures           []error
	signalActionFailures     []error
	signalRuntimeFailures    []error
	lifecycleCleanupFailures []error
}

// Error returns nil, one original error, or a private ordered multi-error. The
// first branch is always the highest-precedence surviving branch.
func (set runFailures) Error() error {
	var ordered []error
	ordered = appendError(ordered, set.terminal)
	ordered = appendError(ordered, set.primary)
	ordered = appendErrors(ordered, set.noticeFailures)
	ordered = appendErrors(ordered, set.signalActionFailures)
	ordered = appendErrors(ordered, set.signalRuntimeFailures)
	ordered = appendErrors(ordered, set.lifecycleCleanupFailures)
	return joinOrdered(ordered...)
}

type orderedError struct {
	errors []error
}

func (err *orderedError) Error() string {
	if err == nil || len(err.errors) == 0 || err.errors[0] == nil {
		return ""
	}
	return err.errors[0].Error()
}

// Format keeps the aggregate's ordinary string form on its highest-precedence
// branch. %+v becomes Diagnostic presentation only when that leading branch is
// itself Commandment-owned; ordinary application/protocol primaries are never
// reclassified merely because a later branch carries Diagnostics.
func (err *orderedError) Format(state fmt.State, verb rune) {
	if verb == 'v' && state.Flag('+') && err != nil && len(err.errors) != 0 &&
		ownsDiagnosticPresentation(err.errors[0]) {
		if writeDiagnosticPresentation(state, err) {
			return
		}
	}
	formatOrderedErrorString(state, verb, err.Error())
}

func formatOrderedErrorString(state fmt.State, verb rune, text string) {
	_, _ = fmt.Fprintf(state, fmt.FormatString(state, verb), text)
}

func (err *orderedError) Unwrap() []error {
	if err == nil {
		return nil
	}
	return append([]error(nil), err.errors...)
}

func joinOrdered(errors ...error) error {
	filtered := make([]error, 0, len(errors))
	for _, err := range errors {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		failure := &orderedError{
			errors: filtered,
		}
		return failure
	}
}

func appendError(dst []error, err error) []error {
	if err != nil {
		dst = append(dst, err)
	}
	return dst
}

func appendErrors(dst, errors []error) []error {
	for _, err := range errors {
		dst = appendError(dst, err)
	}
	return dst
}

// suppressConsequentialCancellation removes a callback cancellation only when
// the execution owner can prove both sides of the causal relationship: the
// authoritative cancellation cause is already preserved, and the callback
// result contains no independent or compound failure branch. Custom errors.Is
// equivalence is intentionally insufficient proof.
func suppressConsequentialCancellation(primary, authoritativeCause, preservedCause error) error {
	if primary == nil || authoritativeCause == nil || preservedCause == nil {
		return primary
	}
	if !exactBranchContains(preservedCause, authoritativeCause) {
		return primary
	}
	if isOnlyCause(primary, authoritativeCause) || isOnlyCause(primary, context.Canceled) {
		return nil
	}
	return primary
}

// isOnlyCause reports whether err is an exact cause or a single-branch unwrap
// chain ending at that exact cause. Compound trees and custom errors.Is
// equivalence are intentionally insufficient causal proof.
func isOnlyCause(err, cause error) bool {
	const maximumErrorChainNodes = 4096
	for visited := 0; err != nil && visited < maximumErrorChainNodes; visited++ {
		if exactErrorIdentity(err, cause) {
			return true
		}
		if _, compound := err.(interface{ Unwrap() []error }); compound {
			return false
		}
		one, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = one.Unwrap()
	}
	return false
}

// preserveIndependentCallerCancellation returns caller cancellation when it is
// the sole returned failure, or appends it after an already-selected primary
// failure unless that exact cause is already reachable in the primary tree.
// The primary failure remains the highest-precedence branch.
func preserveIndependentCallerCancellation(ctx context.Context, primary error) error {
	if ctx == nil || ctx.Err() == nil {
		return primary
	}
	cause := context.Cause(ctx)
	if cause == nil {
		cause = ctx.Err()
	}
	if exactBranchContains(primary, cause) {
		return primary
	}
	primary = suppressConsequentialCancellation(primary, cause, cause)
	if primary == nil {
		return cause //nolint:wrapcheck // A sole caller cancellation cause must preserve exact identity.
	}
	if ordered, ok := errors.AsType[*orderedError](primary); ok {
		branches := ordered.Unwrap()
		branches = append(branches, cause)
		return joinOrdered(branches...)
	}
	return joinOrdered(primary, cause)
}

// runState owns one Run frontend lifetime. Semantic analysis remains owned by
// invocation.analysis; execution configurations and live Signals are created only
// after the selected frontend action has committed to business execution.
type runState struct {
	invocation *invocation

	configurations *executionConfigurations
	signals        *runSignals

	signalSources []compiledSignalSource

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	abort  func(int) error
	notify NoticeHandler
}

func beginRun(model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalSources []compiledSignalSource, request RunRequest) (*runState, error) {
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

	run := &runState{
		invocation:    invocation,
		signalSources: append([]compiledSignalSource(nil), signalSources...),
		stdin:         request.Stdin,
		stdout:        request.Stdout,
		stderr:        request.Stderr,
		abort:         request.Signals.Abort,
		notify:        request.Notify,
	}

	return run, nil
}
