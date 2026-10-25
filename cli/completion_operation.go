package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

var errInvalidCompletionRequest = errors.New("cli: invalid completion request")

type completionContextError struct {
	operation string
	cause     error
}

type completionOperation struct {
	invocation *invocation
	input      completionInput
}

func beginCompletion(model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalSources []compiledSignalSource, request CompletionRequest) (*completionOperation, error) {
	args, cursor, err := completionAnalysisInput(request)
	if err != nil {
		return nil, err
	}
	client := request.Client
	client.Match.Fields = append([]CompletionMatchField(nil), request.Client.Match.Fields...)
	completionInput := completionInput{
		cursor:                 cursor,
		analysisCursorArgument: cursor.Argument,
		trigger:                request.Trigger,
		visibility:             request.Visibility,
		client:                 client,
		limit:                  request.Limit,
	}
	if err := validateCompletionInput(args, completionInput); err != nil {
		return nil, err
	}

	input, err := snapshotInvocationInput(args, request.Env, request.WorkingDirectory, request.ResponseFiles, request.Defaults, request.Facts, request.Locale, request.Limits)
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
	operation := &completionOperation{
		invocation: invocation,
		input:      completionInput,
	}
	return operation, nil
}

func (err *completionContextError) Error() string {
	return "cli: " + err.operation + ": " + err.cause.Error()
}

func (err *completionContextError) Unwrap() error { return err.cause }

func (operation *completionOperation) completeIncompleteResponseFile(ctx context.Context, prefix grammarPrefixResult, result Completion, visibility ReferenceVisibility) (Completion, bool, error) {
	snapshot := operation
	path, ok := incompleteResponseFileCompletionPath(snapshot.invocation.Args()[snapshot.input.analysisCursorArgument])
	if !ok || snapshot.input.cursor.Byte <= 0 {
		return result, false, nil
	}
	grammar, ok := grammarOperationSource{
		operation: snapshot.invocation,
		analysis:  snapshot.invocation.analysis,
	}.scope(prefix.Grammar.SelectedProfiles)
	if !ok {
		panic("cli: accepted completion operation has no grammar scope")
	}
	dialect, ok := grammar.Dialect(prefix.Current)
	if !ok {
		panic("cli: accepted completion command has no dialect")
	}
	if !dialect.ResponseFiles() {
		return result, false, nil
	}
	referenceOptionsValue := ReferenceOptions{
		Locale:     snapshot.invocation.input.locale,
		Visibility: visibility,
		Order:      ReferenceOrderDeclaration,
	}
	reference, err := projectOperationReference(snapshot.invocation, referenceOptionsValue)
	if err != nil {
		return result, true, completionRequestFailure(snapshot.invocation, err)
	}
	result.Locale = reference.Locale
	completed, err := operation.completeResponseFileReference(ctx, reference, prefix, result, path)
	return completed, true, err
}

func (operation *completionOperation) completePendingFlagCompletion(ctx context.Context, prefix grammarPrefixResult, result Completion, shortcutAppendCount int, visibility ReferenceVisibility) (Completion, error) {
	snapshot := operation
	invocation := operation.invocation
	pendingPrefix := prefix
	if shortcutAppendCount != 0 {
		var viable bool
		pendingPrefix, viable = analyzeShortcutPendingFlagCompletion(ctx, snapshot, prefix, invocation)
		if !viable {
			return result, nil
		}
		providerNamespaces := invocation.providerNamespaces()
		providerDiagnostics, _ := providerMaterializationDiagnostics(providerNamespaces)
		result.Selection = projectGrammarSelection(pendingPrefix.Grammar.Selection)
		result.Diagnostics = append([]Diagnostic(nil), providerDiagnostics...)
		if len(providerDiagnostics) != 0 || len(pendingPrefix.Grammar.IncompleteNamespaces) != 0 {
			result.Partial = true
		}
	}
	effectiveModel := snapshot.invocation.analysis.effectiveModel
	referenceOptionsValue := ReferenceOptions{
		Locale:     snapshot.invocation.input.locale,
		Visibility: visibility,
		Order:      ReferenceOrderDeclaration,
	}
	reference, err := projectOperationReference(snapshot.invocation, referenceOptionsValue)
	if err != nil {
		return result, completionRequestFailure(snapshot.invocation, err)
	}
	result.Locale = reference.Locale
	if structuralPresentationNeedsValues(effectiveModel, pendingPrefix.Grammar) {
		if code, supplyErr := invocation.finalizeSupplies(ctx, pendingPrefix.Grammar.Selection); supplyErr != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation, code, "completion invocation value supply failed", supplyErr,
			)
		}
	}
	if !applyStructuralConditionsReference(ctx, &reference, effectiveModel, pendingPrefix.Grammar, invocation) {
		if code, supplyErr := invocation.supplyFailure(); supplyErr != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation, code, "completion invocation value supply failed", supplyErr,
			)
		}
		result.Partial = true
		return result, nil
	}
	snapshot.invocation.EnableShortcutDefaults()
	return operation.completePendingFlagValue(ctx, effectiveModel, reference, pendingPrefix, result)
}

func completeApplicationWithSignals(ctx context.Context, model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalSources []compiledSignalSource, request CompletionRequest) (Completion, error) {
	completion, err := completeApplicationWithSignalsUnpreserved(ctx, model, callbacks, identity, signalSources, request)
	return completion, preserveIndependentCallerCancellation(ctx, err)
}

func completeApplicationWithSignalsUnpreserved(ctx context.Context, model staticProgram, callbacks declarationCallbacks, identity *applicationIdentity, signalSources []compiledSignalSource, request CompletionRequest) (Completion, error) {
	operation, err := beginCompletion(model, callbacks, identity, signalSources, request)
	if err != nil {
		return Completion{}, projectDiagnosticFailure(DiagnosticInvalidCompletionRequest, "invalid completion request", err)
	}
	defer func() {
		if operation.invocation.analysis != nil {
			finalizeOperationFrameworkPresentation(operation.invocation, operation.invocation.input.locale, operation.invocation.analysis.effectiveModel)
		}
	}()
	snapshot := operation
	visibility := ReferenceVisibilityPublic
	if snapshot.input.visibility == CompletionVisibilityAllInvocable {
		visibility = ReferenceVisibilityAll
	}
	invocation := snapshot.invocation
	completionOp := snapshot
	snapshot.invocation.beginAnalysis()
	shortcutState, err := settleCompletionShortcutPrefix(ctx, completionOp, completionOp.invocation)
	shortcutAppendCount := shortcutState.trailingAppend
	if err != nil {
		if failedArgument, ok := responseExpansionFailureArgument(err); ok {
			failure := responseFileDiagnosticFailure(snapshot.invocation, err)
			prefix, captured := responseExpansionFailurePrefix(err)
			return completionPartialResultBeforeResponseFileFailure(
				ctx,
				snapshot,
				visibility,
				invocation.namespaceFor(ctx),
				invocation.conditionFor(ctx),
				invocation.defaultCommandFor(ctx),
				failedArgument,
				prefix,
				captured,
			), failure
		}
		return Completion{}, err
	}
	if code, err := invocation.supplyFailure(); err != nil {
		return Completion{}, operationDiagnosticFailure(snapshot.invocation, code, "invocation value supply failed", err)
	}
	prefix := analyzeGrammarPrefix(
		ctx,
		snapshot.invocation,
		snapshot.input.analysisCursorArgument,
		invocation.namespaceFor(ctx),
		invocation.conditionFor(ctx),
		invocation.defaultCommandFor(ctx),
	)
	if code, err := invocation.supplyFailure(); err != nil {
		return Completion{}, operationDiagnosticFailure(snapshot.invocation, code, "invocation value supply failed", err)
	}
	effectiveModel := snapshot.invocation.analysis.effectiveModel
	locale, err := completionResultLocale(effectiveModel, snapshot.invocation.input.locale)
	if err != nil {
		return Completion{}, completionRequestFailure(snapshot.invocation, err)
	}
	snapshot.invocation.FinalizeFrameworkPresentationLocale(string(locale))
	var reference Reference
	projectCurrentReference := func() error {
		var projectErr error
		referenceOptionsValue := ReferenceOptions{
			Locale:     snapshot.invocation.input.locale,
			Visibility: visibility,
			Order:      ReferenceOrderDeclaration,
		}
		reference, projectErr = projectOperationReference(snapshot.invocation, referenceOptionsValue)
		return projectErr
	}
	providerNamespaces := invocation.providerNamespaces()
	providerDiagnostics, _ := providerMaterializationDiagnostics(providerNamespaces)
	var providerCause error
	result := Completion{
		Locale:    locale,
		Selection: projectGrammarSelection(prefix.Grammar.Selection),
		Replace: CompletionSpan{
			Argument: snapshot.input.cursor.Argument,
			Start:    0,
			End:      len(snapshot.invocation.Args()[snapshot.input.analysisCursorArgument]),
		},
		Sources:     projectOperationSources(snapshot.invocation),
		Origins:     projectOperationOrigins(snapshot.invocation),
		Diagnostics: append([]Diagnostic(nil), providerDiagnostics...),
	}
	if len(providerDiagnostics) != 0 || len(prefix.Grammar.IncompleteNamespaces) != 0 {
		result.Partial = true
	}
	if err := ctx.Err(); err != nil {
		result.Partial = true
		failure := &completionContextError{
			operation: "begin completion",
			cause:     err,
		}
		return result, failure
	}
	if len(prefix.Grammar.Issues) != 0 {
		diagnostics, projectionTruncated := grammarDiagnostics(snapshot.invocation, prefix.Grammar, nil)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		result.DiagnosticsTruncated = prefix.Grammar.Truncated || projectionTruncated
		result.Partial = true
		return result, grammarDiagnosticFailure(
			snapshot.invocation,
			prefix.Grammar,
			diagnostics,
			result.DiagnosticsTruncated,
		)
	}
	if prefix.TerminalAction() {
		return result, nil
	}
	if responseResult, handled, responseErr := completionOp.completeIncompleteResponseFile(
		ctx, prefix, result, visibility,
	); handled {
		return responseResult, responseErr
	}
	if prefix.CommandPathOpen() &&
		completionCursorCanNameCommand(snapshot.invocation.Args()[snapshot.input.analysisCursorArgument]) {
		invocation.namespace(ctx, grammarNamespaceDemand{
			Command: prefix.Current,
			Grammar: prefix.Grammar,
		})
		providerNamespaces = invocation.providerNamespaces()
		providerDiagnostics, _ = providerMaterializationDiagnostics(providerNamespaces)
		result.Diagnostics = append([]Diagnostic(nil), providerDiagnostics...)
		if len(providerDiagnostics) != 0 {
			result.Partial = true
		}
	}
	if prefix.PendingFlagValue != nil {
		return completionOp.completePendingFlagCompletion(ctx, prefix, result, shortcutAppendCount, visibility)
	}

	// Preserve the pre-cursor explicit syntax window while independently
	// settling the branch that owns the cursor only when no explicit structural
	// owner recognizes it. Defaults consume no input, so this may advance into a
	// child command while parent command candidates remain legitimate edits.
	syntaxPrefixes := []grammarPrefixResult{prefix}
	completionPrefix := prefix
	defaultAnalysis := analyzeCompletionDefaultScope(
		ctx,
		snapshot.invocation,
		prefix,
		snapshot.input.analysisCursorArgument,
		snapshot.invocation.Args()[snapshot.input.analysisCursorArgument],
		invocation.namespaceFor(ctx),
		invocation.conditionFor(ctx),
		invocation.defaultCommandFor(ctx),
	)
	if !defaultAnalysis.Settled {
		result.Partial = true
	} else if !defaultAnalysis.StructuralOwner {
		completionPrefix = defaultAnalysis.Prefix
		if completionPrefix.Current != prefix.Current {
			syntaxPrefixes = append(syntaxPrefixes, completionPrefix)
		}
	}

	var shortcutTail *grammarPrefixResult
	shortcutTailApplied := false
	//nolint:nestif // Shortcut-tail settlement stays inline so cursor ownership and partiality remain in orchestration order.
	if shortcutAppendCount != 0 && defaultAnalysis.Settled {
		tailPrefix := analyzeGrammarCompletionTail(
			ctx,
			snapshot.invocation,
			snapshot.input.analysisCursorArgument,
			invocation.namespaceFor(ctx),
			invocation.conditionFor(ctx),
			invocation.defaultCommandFor(ctx),
		)
		if code, supplyErr := invocation.supplyFailure(); supplyErr != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation,
				code,
				"completion invocation value supply failed",
				supplyErr,
			)
		}

		if len(tailPrefix.Grammar.Issues) != 0 || tailPrefix.HelpPath() || tailPrefix.TerminalAction() {
			if !completionCursorMaySelectCommand(prefix, snapshot.invocation.Args()[snapshot.input.analysisCursorArgument]) {
				// A fixed suffix can be invalid only because the intentionally skipped
				// cursor is itself the earlier command-selection position. In that case
				// candidate-local canonical analysis below decides which insertions make
				// the unchanged suffix legal; the unresolved cursor is not partiality.
				result.Partial = true
			}
		} else {
			shortcutTail = &tailPrefix
			syntaxPrefixes = completionSyntaxPrefixesWithTailOccurrences(syntaxPrefixes, tailPrefix.FlagOccurrences)
			if !defaultAnalysis.StructuralOwner && tailPrefix.Current == completionPrefix.Current {
				// Cursor ownership stays at the original incomplete frontier. When the
				// fixed suffix remains in that same command scope, its complete flag and
				// positional ownership can also participate in cursor value analysis.
				tailPrefix.FlagsEnabled = completionPrefix.FlagsEnabled
				tailPrefix = adoptGrammarPrefixRoute(tailPrefix, completionPrefix)
				completionPrefix = tailPrefix
				shortcutTail = &tailPrefix
				shortcutTailApplied = true
			}
		}
	}

	// Default-scope settlement may demand and atomically accept another provider
	// batch. Observe the final operation-effective frontier before any
	// presentation or target callback observes the declaration graph. The common
	// static syntax path projects only candidates that can become Completion
	// output; branches that require structural/value/argument projection retain
	// the complete Reference path below.
	effectiveModel = snapshot.invocation.analysis.effectiveModel
	providerNamespaces = invocation.providerNamespaces()
	providerDiagnostics, providerCause = providerMaterializationDiagnostics(providerNamespaces)
	effectiveGrammar := completionPrefix.Grammar
	if shortcutTail != nil {
		effectiveGrammar = shortcutTail.Grammar
	}
	result.Selection = projectGrammarSelection(effectiveGrammar.Selection)
	result.Diagnostics = append([]Diagnostic(nil), providerDiagnostics...)
	if len(providerDiagnostics) != 0 || len(effectiveGrammar.IncompleteNamespaces) != 0 {
		result.Partial = true
	}

	// A settled static syntax completion with no active structural
	// presentation and no positional target does not need a whole detached
	// Reference as an intermediate. Project the final candidate records directly
	// from the canonical operation view and effective grammar. This keeps the
	// unavoidable detached Θ(output) result while preventing unrelated Schema /
	// Reference families from multiplying fixed-output Completion work.
	//nolint:nestif // Keep static projection inline so its settlement and presentation gates remain visible in one control flow.
	if defaultAnalysis.Settled && !completionStaticProjectionNeedsReference(effectiveModel, effectiveGrammar) {
		argumentAnalysis := argumentCompletionAnalysis{
			Grammar: effectiveGrammar,
			Settled: true,
		}
		fastPrefix := completionPrefix
		if !defaultAnalysis.StructuralOwner {
			switch {
			case shortcutTailApplied:
				argumentAnalysis = analyzeArgumentCompletionAt(
					snapshot.invocation,
					fastPrefix,
					snapshot.input.analysisCursorArgument,
					snapshot.invocation.Args()[snapshot.input.analysisCursorArgument],
					invocation.conditionFor(ctx),
				)
			case shortcutTail == nil || shortcutTail.Current == fastPrefix.Current:
				argumentAnalysis = analyzeArgumentCompletion(
					snapshot.invocation,
					fastPrefix,
					snapshot.invocation.Args()[snapshot.input.analysisCursorArgument],
					invocation.conditionFor(ctx),
				)
			}
			fastPrefix.Grammar = argumentAnalysis.Grammar
		}
		if argumentAnalysis.Settled && !argumentAnalysis.Ambiguous && len(argumentAnalysis.Targets) == 0 {
			if code, supplyErr := invocation.supplyFailure(); supplyErr != nil {
				result.Partial = true
				return result, operationDiagnosticFailure(
					snapshot.invocation,
					code,
					"completion invocation value supply failed",
					supplyErr,
				)
			}
			snapshot.invocation.EnableShortcutDefaults()
			projector, projectErr := prepareCompletionStaticProjector(snapshot.invocation, visibility, snapshot.invocation.input.locale)
			if projectErr == nil {
				var staticErr error
				result, staticErr = completionOp.appendStaticCompletionSyntax(
					ctx,
					result,
					projector,
					syntaxPrefixes,
					shortcutAppendCount,
					shortcutState.candidateSensitiveResponseFiles,
				)
				if staticErr != nil {
					result.Partial = true
				}
				finalizeCompletionItems(&result, snapshot.input)
				var completionErr error
				if len(providerDiagnostics) != 0 {
					completionErr = providerDiagnosticFailure(snapshot.invocation, providerNamespaces, providerCause)
				}
				return result, errors.Join(completionErr, staticErr)
			}
		}
	}

	if err := projectCurrentReference(); err != nil {
		return result, completionRequestFailure(snapshot.invocation, err)
	}
	result.Locale = reference.Locale
	if structuralPresentationNeedsValues(effectiveModel, effectiveGrammar) {
		if code, supplyErr := invocation.finalizeSupplies(ctx, effectiveGrammar.Selection); supplyErr != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation,
				code,
				"completion invocation value supply failed",
				supplyErr,
			)
		}
	}
	if !applyStructuralConditionsReference(ctx, &reference, effectiveModel, effectiveGrammar, invocation) {
		if code, supplyErr := invocation.supplyFailure(); supplyErr != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation,
				code,
				"completion invocation value supply failed",
				supplyErr,
			)
		}
		result.Partial = true
	}

	argumentAnalysis := argumentCompletionAnalysis{
		Grammar: effectiveGrammar,
		Settled: true,
	}
	if defaultAnalysis.Settled && !defaultAnalysis.StructuralOwner {
		if shortcutTailApplied {
			argumentAnalysis = analyzeArgumentCompletionAt(
				snapshot.invocation,
				completionPrefix,
				snapshot.input.analysisCursorArgument,
				snapshot.invocation.Args()[snapshot.input.analysisCursorArgument],
				invocation.conditionFor(ctx),
			)
		} else if shortcutTail == nil || shortcutTail.Current == completionPrefix.Current {
			argumentAnalysis = analyzeArgumentCompletion(
				snapshot.invocation,
				completionPrefix,
				snapshot.invocation.Args()[snapshot.input.analysisCursorArgument],
				invocation.conditionFor(ctx),
			)
		}
		completionPrefix.Grammar = argumentAnalysis.Grammar
		if !argumentAnalysis.Settled {
			result.Partial = true
		}
	}
	snapshot.invocation.EnableShortcutDefaults()

	transferExclusive := len(argumentAnalysis.Targets) == 1 &&
		(argumentAnalysis.Targets[0].Kind == ArgumentTermRemainder || argumentAnalysis.Targets[0].Kind == ArgumentTermPassThrough)

	var staticErr error
	if !transferExclusive {
		result, staticErr = completionOp.appendStaticCompletionSyntax(
			ctx,
			result,
			referenceStaticCompletionProjector{
				reference: reference,
			},
			syntaxPrefixes,
			shortcutAppendCount,
			shortcutState.candidateSensitiveResponseFiles,
		)
		if staticErr != nil {
			result.Partial = true
		}
	}

	var argumentErr error
	if argumentAnalysis.Settled && (len(argumentAnalysis.Targets) != 0 || argumentAnalysis.Ambiguous) {
		result, argumentErr = completionOp.completeArgumentCursor(
			ctx,
			effectiveModel,
			reference,
			completionPrefix,
			result,
			argumentAnalysis,
		)
	}

	finalizeCompletionItems(&result, snapshot.input)
	var completionErr error
	if len(providerDiagnostics) != 0 {
		completionErr = providerDiagnosticFailure(snapshot.invocation, providerNamespaces, providerCause)
	}
	return result, errors.Join(completionErr, staticErr, argumentErr)
}

func completionPartialResultBeforeResponseFileFailure(ctx context.Context, snapshot *completionOperation, visibility ReferenceVisibility, materialize grammarNamespaceDemandFunc, condition grammarConditionDemandFunc, defaults grammarDefaultDemandFunc, failedArgument int, prefix grammarPrefixResult, captured bool) Completion {
	if !captured {
		prefix = analyzeGrammarPrefix(
			ctx,
			snapshot.invocation,
			failedArgument,
			materialize,
			condition,
			defaults,
		)
	}
	replace := CompletionSpan{
		Argument: snapshot.input.cursor.Argument,
	}
	args := snapshot.invocation.Args()
	if snapshot.input.analysisCursorArgument >= 0 && snapshot.input.analysisCursorArgument < len(args) {
		replace.End = len(args[snapshot.input.analysisCursorArgument])
	}
	referenceOptionsValue := ReferenceOptions{
		Locale:     snapshot.invocation.input.locale,
		Visibility: visibility,
		Order:      ReferenceOrderDeclaration,
	}
	reference, err := projectOperationReference(snapshot.invocation, referenceOptionsValue)
	if err != nil {
		completion := Completion{
			Partial:   true,
			Selection: projectGrammarSelection(prefix.Grammar.Selection),
			Replace:   replace,
			Sources:   projectOperationSources(snapshot.invocation),
			Origins:   projectOperationOrigins(snapshot.invocation),
		}

		return completion
	}
	providerDiagnostics, _ := providerMaterializationDiagnostics(snapshot.invocation.analysis.providers.outcomes())
	completion := Completion{
		Partial:     true,
		Locale:      reference.Locale,
		Selection:   projectGrammarSelection(prefix.Grammar.Selection),
		Replace:     replace,
		Sources:     projectOperationSources(snapshot.invocation),
		Origins:     projectOperationOrigins(snapshot.invocation),
		Diagnostics: append([]Diagnostic(nil), providerDiagnostics...),
	}

	return completion
}

func completionCursorCanNameCommand(argument string) bool {
	return argument == "" || argument[0] != '-'
}

func completionCursorMaySelectCommand(prefix grammarPrefixResult, argument string) bool {
	return prefix.CommandPathOpen() && prefix.PendingFlagValue == nil && completionCursorCanNameCommand(argument)
}

func incompleteResponseFileCompletionPath(argument string) (string, bool) {
	if argument == "" || argument[0] != '@' {
		return "", false
	}
	if len(argument) > 1 && argument[1] == '@' {
		return "", false
	}
	return argument[1:], true
}

func (operation *completionOperation) completeResponseFileReference(ctx context.Context, reference Reference, prefix grammarPrefixResult, result Completion, path string) (Completion, error) {
	snapshot := operation
	command, ok := reference.Commands[prefix.Current]
	if !ok {
		panic("cli: accepted response-file completion command is absent from Reference")
	}
	target := CompletionTarget{
		Kind:        CompletionTargetResponseFile,
		Command:     command.ID,
		CommandPath: append([]string(nil), command.Path...),
	}
	result.Targets = append(result.Targets, target)
	result.Replace.Start = 1
	result.Replace.End = len(path) + 1
	if err := ctx.Err(); err != nil {
		result.Partial = true
		failure := &completionContextError{
			operation: "complete response file",
			cause:     err,
		}
		return result, failure
	}

	request := ResponseFileCompletionRequest{
		Path:   path,
		Base:   snapshot.invocation.input.workingDirectory,
		Cursor: snapshot.input.cursor.Byte - 1,
		Replace: CompletionRange{
			Start: 0,
			End:   len(path),
		},
		Trigger: snapshot.input.trigger,
		Limit:   snapshot.input.limit,
	}
	contribution, err := responseFileComplete(ctx, snapshot.invocation.input.responseFiles, request)
	if err != nil {
		result.Partial = true
		if errors.Is(err, errInvalidCompletionContribution) {
			return result, operationDiagnosticFailure(
				snapshot.invocation,
				DiagnosticInvalidCompletionContribution,
				"invalid response-file completion contribution",
				err,
			)
		}
		return result, operationDiagnosticFailure(
			snapshot.invocation,
			DiagnosticCompletionFailure,
			"response-file completion failed",
			err,
		)
	}
	result, err = applyResponseFileCompletionContribution(result, snapshot, contribution, path)
	if err != nil {
		result.Partial = true
		return result, operationDiagnosticFailure(
			snapshot.invocation,
			DiagnosticInvalidCompletionContribution,
			"invalid response-file completion contribution",
			err,
		)
	}
	finalizeCompletionItems(&result, snapshot.input)
	return result, nil
}

func (operation *completionOperation) completePendingFlagValue(ctx context.Context, model *effectiveModel, reference Reference, prefix grammarPrefixResult, result Completion) (Completion, error) {
	snapshot := operation
	invocation := operation.invocation
	pending := prefix.PendingFlagValue
	if pending == nil {
		panic("cli: missing pending flag value")
	}
	flag, ok := reference.Flags[pending.Flag]
	if !ok || !flag.Available {
		panic("cli: accepted pending flag is absent from completion Reference")
	}
	spelling, ok := reference.FlagSpellings[pending.Spelling]
	if !ok || !spelling.Available {
		panic("cli: accepted pending flag spelling is absent from completion Reference")
	}
	target := CompletionTarget{
		Kind:          CompletionTargetFlagValue,
		Command:       pending.Command,
		CommandPath:   append([]string(nil), reference.Commands[pending.Command].Path...),
		ValueID:       flag.Destination,
		Name:          flag.ValueName,
		Spelling:      spelling.Spelling.Text,
		FlagValueForm: publicFlagValueForm(pending.Form),
		Occurrence:    pending.Occurrence,
	}
	result.Targets = append(result.Targets, target)
	if err := ctx.Err(); err != nil {
		result.Partial = true
		failure := &completionContextError{
			operation: "complete pending flag value",
			cause:     err,
		}
		return result, failure
	}

	record, ok := model.Flag(pending.Flag)
	if !ok {
		panic("cli: accepted pending flag is absent from effective model")
	}
	if completionPendingValueNeedsInvocationSupplies(model, pending.Destination, record) {
		if code, err := invocation.finalizeSupplies(ctx, prefix.Grammar.Selection); err != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation,
				code,
				"completion invocation value supply failed",
				err,
			)
		}
	}

	effectiveModel, domainReference, domainOutcomes, domainErr := snapshot.settleCompletionTargetDomains(ctx, prefix.Grammar, pending.Destination)
	if domainErr != nil {
		diagnostics, _ := dynamicDomainMaterializationDiagnostics(domainOutcomes)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		result.Partial = true
		return result, domainErr
	}
	if domainReference.Commands != nil {
		model = effectiveModel
		reference = domainReference
		record, ok = model.Flag(pending.Flag)
		if !ok {
			panic("cli: materialized pending flag is absent from effective model")
		}
		result.Locale = reference.Locale
		result = appendCompletionDomainCandidates(result, snapshot, reference, flag.Destination)
	}

	completion, hasCompletion := record.Completion()
	if !hasCompletion {
		finalizeCompletionItems(&result, snapshot.input)
		return result, nil
	}
	switch completion.Kind() {
	case completionDeclarationKindNative:
		nativeKind, nativeMode, hasNative := completion.Native()
		if hasNative && completionNativeSupported(snapshot.input.client.Capabilities.Native, nativeKind) &&
			nativeCompletionTierActive(nativeMode, len(result.Items), 0) {
			request2 := CompletionNativeRequest{
				Kind:        nativeKind,
				Mode:        nativeMode,
				TargetIndex: 0,
			}
			result.NativeRequests = append(result.NativeRequests, request2)
		}
		finalizeCompletionItems(&result, snapshot.input)
		return result, nil
	case completionDeclarationKindValue, completionDeclarationKindTokenLanguage:
		callback, _ := snapshot.invocation.flagCompletionCallback(record.meta.id)
		if !callback.present() {
			panic("cli: accepted flag completion has no callback")
		}
		callbackCtx, callbackValues, ok := operation.completionCallbackContext(
			ctx,
			model,
			reference,
			prefix,
			result.Selection,
			completion.Dependencies(),
		)
		if !ok {
			result.Partial = true
			return result, completionDependencyFailure(snapshot.invocation, model, callbackValues, completion.Dependencies())
		}
		configuration, ok := snapshot.invocation.analysisConfiguration(record.command, callbackValues.Destinations)
		if !ok {
			panic("cli: accepted pending-flag completion callback has no analysis configuration")
		}
		currentText := snapshot.invocation.Args()[snapshot.input.analysisCursorArgument]
		currentReplace := CompletionRange{
			Start: 0,
			End:   len(currentText),
		}
		var contribution CompletionContribution
		var callbackErr error
		if completion.Kind() == completionDeclarationKindValue {
			request := ValueCompletionRequest{
				Target:           target,
				Selection:        cloneCommandSelection(result.Selection),
				Text:             currentText,
				Cursor:           snapshot.input.cursor.Byte,
				Replace:          currentReplace,
				WorkingDirectory: snapshot.invocation.input.workingDirectory,
				Trigger:          snapshot.input.trigger,
				Limit:            snapshot.input.limit,
			}
			contribution, callbackErr = callback.value(callbackCtx, configuration, request)
		} else {
			target.Kind = CompletionTargetTokenLanguage
			result.Targets[0] = target
			request := TokenLanguageCompletionRequest{
				Target:           target,
				Selection:        cloneCommandSelection(result.Selection),
				Tokens:           pendingFlagValueTokens(snapshot.invocation.Args(), pending),
				Text:             currentText,
				Cursor:           snapshot.input.cursor.Byte,
				Replace:          currentReplace,
				WorkingDirectory: snapshot.invocation.input.workingDirectory,
				Trigger:          snapshot.input.trigger,
				Limit:            snapshot.input.limit,
			}
			contribution, callbackErr = callback.tokenLanguage(callbackCtx, configuration, request)
		}
		if callbackErr != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation,
				DiagnosticCompletionFailure,
				"completion callback failed",
				callbackErr,
			)
		}
		result, applyErr := applyValueCompletionContribution(result, snapshot, contribution, completion)
		if applyErr != nil {
			result.Partial = true
			return result, operationDiagnosticFailure(
				snapshot.invocation,
				DiagnosticInvalidCompletionContribution,
				"invalid completion contribution",
				applyErr,
			)
		}
		finalizeCompletionItems(&result, snapshot.input)
		return result, nil
	default:
		panic("cli: accepted pending-flag completion declaration has invalid kind")
	}
}

func completionPendingValueNeedsInvocationSupplies(model *effectiveModel, destination ValueID, flag flagRecord) bool {
	if completion, ok := flag.Completion(); ok {
		switch completion.Kind() {
		case completionDeclarationKindValue, completionDeclarationKindTokenLanguage:
			return true
		case completionDeclarationKindUnknown,
			completionDeclarationKindDelegation,
			completionDeclarationKindNative:
		}
	}
	if completionDomainsNeedInvocationSupplies(model, destination) {
		return true
	}
	return false
}

func pendingFlagValueTokens(args []string, pending *flagOccurrence) []string {
	if pending == nil || len(pending.ValueIndices) == 0 {
		return nil
	}
	result := make([]string, 0, len(pending.ValueIndices))
	for _, index := range pending.ValueIndices {
		if index < 0 || index >= len(args) {
			panic("cli: pending flag value index out of range")
		}
		result = append(result, args[index])
	}
	return result
}

func completionNativeSupported(capabilities CompletionNativeCapabilities, kind CompletionNativeKind) bool {
	switch kind {
	case CompletionNativePaths:
		return capabilities.Paths
	case CompletionNativeFiles:
		return capabilities.Files
	case CompletionNativeDirectories:
		return capabilities.Directories
	case CompletionNativeExecutables:
		return capabilities.Executables
	case CompletionNativeUsers:
		return capabilities.Users
	case CompletionNativeHosts:
		return capabilities.Hosts
	case CompletionNativeEnvironmentVariables:
		return capabilities.EnvironmentVariables
	default:
		return false
	}
}

func completionRequestFailure(operation *invocation, cause error) error {
	return operationDiagnosticFailure(
		operation,
		DiagnosticInvalidCompletionRequest,
		"invalid completion request",
		cause,
	)
}

func validateCompletionInput(args []string, input completionInput) error {
	if input.cursor.Argument < 0 || input.cursor.Argument >= len(args) || input.cursor.Byte < 0 ||
		input.cursor.Byte > len(args[input.cursor.Argument]) {
		return errInvalidCompletionRequest
	}
	// The prefix recognizer consumes every preceding argument exactly once and
	// deliberately leaves the complete cursor-owned argument uncommitted. The
	// byte cursor may therefore sit anywhere inside that argument; target-local
	// matching and callbacks receive the full text plus the byte coordinate.
	if input.visibility != CompletionVisibilityPublic && input.visibility != CompletionVisibilityAllInvocable {
		return errInvalidCompletionRequest
	}
	if input.trigger.Kind < CompletionTriggerInvoked || input.trigger.Kind > CompletionTriggerRefresh {
		return errInvalidCompletionRequest
	}
	if input.trigger.Kind == CompletionTriggerCharacter && input.trigger.Character == 0 {
		return errInvalidCompletionRequest
	}
	if input.trigger.Kind != CompletionTriggerCharacter && input.trigger.Character != 0 {
		return errInvalidCompletionRequest
	}
	if input.client.Match.Mode < CompletionMatchModePrefix ||
		input.client.Match.Mode > CompletionMatchModeUnfiltered {
		return errInvalidCompletionRequest
	}
	if input.client.Match.Case < CompletionCaseSmart || input.client.Match.Case > CompletionCaseInsensitive {
		return errInvalidCompletionRequest
	}
	seenFields := make(map[CompletionMatchField]struct{}, len(input.client.Match.Fields))
	for _, field := range input.client.Match.Fields {
		if field < CompletionMatchFieldValue || field > CompletionMatchFieldDetail {
			return errInvalidCompletionRequest
		}
		if _, duplicate := seenFields[field]; duplicate {
			return errInvalidCompletionRequest
		}
		seenFields[field] = struct{}{}
	}
	if input.client.Ranking < CompletionRankingModeRelevance ||
		input.client.Ranking > CompletionRankingModeUnranked {
		return errInvalidCompletionRequest
	}
	if input.client.Shell.Family == "" && input.client.Shell.Version != "" {
		return errInvalidCompletionRequest
	}
	if input.limit < 0 {
		return errInvalidCompletionRequest
	}
	return nil
}

func completionSyntaxPrefixesWithTailOccurrences(prefixes []grammarPrefixResult, occurrences map[FlagID]int) []grammarPrefixResult {
	if len(prefixes) == 0 || len(occurrences) == 0 {
		return prefixes
	}
	result := append([]grammarPrefixResult(nil), prefixes...)
	for index := range result {
		result[index].FlagOccurrences = make(map[FlagID]int, len(occurrences))
		maps.Copy(result[index].FlagOccurrences, occurrences)
	}
	return result
}

type staticCompletionSyntaxProjector interface {
	commandPath(command CommandID) ([]string, bool)
	commandItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool)
	flagItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool)
}

type referenceStaticCompletionProjector struct {
	reference Reference
}

func (projector referenceStaticCompletionProjector) commandPath(command CommandID) ([]string, bool) {
	projected, ok := projector.reference.Commands[command]
	if !ok {
		return nil, false
	}
	return append([]string(nil), projected.Path...), true
}

func (projector referenceStaticCompletionProjector) commandItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	return staticCommandCompletionItem(projector.reference, client, candidate, targetIndex)
}

func (projector referenceStaticCompletionProjector) flagItem(client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	return staticFlagCompletionItem(projector.reference, client, candidate, targetIndex)
}

func (operation *completionOperation) appendStaticCompletionSyntax(ctx context.Context, result Completion, projector staticCompletionSyntaxProjector, prefixes []grammarPrefixResult, fixedTailCount int, candidateSensitiveResponseFiles bool) (Completion, error) {
	snapshot := operation
	var responseFileLoads completionResponseFileLoadCache
	seenCommands := make(map[CommandEntryID]struct{})
	seenCommandActions := make(map[ActionEntryID]struct{})
	seenFlags := make(map[FlagSpellingID]struct{})
	seenFlagActions := make(map[ActionEntryID]struct{})
	matcher := prepareCompletionMatcher(snapshot.input, completionMatchQuery(snapshot))
	scratch := completionMatchScratch{}
	var candidateErrors []error
	appendCandidates := func(
		prefix grammarPrefixResult,
		kind CompletionTargetKind,
		candidates []completionSyntaxCandidate,
		project func(CompletionClient, completionSyntaxCandidate, int) (CompletionItem, bool),
	) {
		if len(candidates) == 0 {
			return
		}
		path, ok := projector.commandPath(prefix.Current)
		if !ok {
			return
		}
		targetIndex := len(result.Targets)
		completionTargetValue := CompletionTarget{
			Kind:        kind,
			Command:     prefix.Current,
			CommandPath: path,
		}
		result.Targets = append(result.Targets, completionTargetValue)
		for _, candidate := range candidates {
			item, ok := project(snapshot.input.client, candidate, targetIndex)
			if ok {
				item, ok = matchCompletionItem(item, matcher, &scratch)
			}
			if ok && fixedTailCount != 0 {
				var branchErr error
				ok, branchErr = operation.completionSyntaxCandidatePreservesFixedTail(
					ctx,
					item.Value,
					fixedTailCount,
					candidateSensitiveResponseFiles,
					&responseFileLoads,
				)
				if branchErr != nil {
					candidateErrors = append(
						candidateErrors,
						responseFileDiagnosticFailure(snapshot.invocation, branchErr),
					)
				}
			}
			if !ok {
				continue
			}
			if snapshot.input.client.Capabilities.Groups {
				item.GroupID = staticCompletionGroupID(
					snapshot.invocation,
					prefix.Grammar,
					prefix.Current,
					path,
					candidate,
				)
				result = appendStaticCompletionGroup(result, item.GroupID)
			}
			result.Items = append(result.Items, item)
		}
	}
	for _, prefix := range prefixes {
		syntax := analyzeCompletionSyntax(snapshot.invocation, prefix)
		commandOrder := make([]completionSyntaxCandidate, 0, len(syntax.CommandOrder))
		for _, candidate := range syntax.CommandOrder {
			switch candidate.Kind {
			case completionSyntaxCandidateCommandEntry:
				if _, seen := seenCommands[candidate.CommandEntry]; seen {
					continue
				}
				seenCommands[candidate.CommandEntry] = struct{}{}
			case completionSyntaxCandidateCommandAction:
				if _, seen := seenCommandActions[candidate.ActionEntry]; seen {
					continue
				}
				seenCommandActions[candidate.ActionEntry] = struct{}{}
			default:
				continue
			}
			commandOrder = append(commandOrder, candidate)
		}
		appendCandidates(prefix, CompletionTargetCommand, commandOrder, projector.commandItem)

		flagOrder := make([]completionSyntaxCandidate, 0, len(syntax.FlagOrder))
		for _, candidate := range syntax.FlagOrder {
			switch candidate.Kind {
			case completionSyntaxCandidateFlagSpelling:
				if _, seen := seenFlags[candidate.FlagSpelling]; seen {
					continue
				}
				seenFlags[candidate.FlagSpelling] = struct{}{}
			case completionSyntaxCandidateFlagAction:
				if _, seen := seenFlagActions[candidate.ActionEntry]; seen {
					continue
				}
				seenFlagActions[candidate.ActionEntry] = struct{}{}
			default:
				continue
			}
			flagOrder = append(flagOrder, candidate)
		}
		appendCandidates(prefix, CompletionTargetFlag, flagOrder, projector.flagItem)
	}
	return result, errors.Join(candidateErrors...)
}

func (operation *completionOperation) completionSyntaxCandidatePreservesFixedTail(ctx context.Context, candidate string, fixedTailCount int, candidateSensitiveResponseFiles bool, responseFileLoads *completionResponseFileLoadCache) (bool, error) {
	snapshot := operation
	invocation := operation.invocation
	if candidate == "" || snapshot.input.analysisCursorArgument < 0 ||
		snapshot.input.analysisCursorArgument >= len(snapshot.invocation.Args()) {
		return false, nil
	}
	args := snapshot.invocation.Args()
	args[snapshot.input.analysisCursorArgument] = candidate
	if candidateSensitiveResponseFiles {
		var err error
		args, err = operation.expandCompletionShortcutCandidateTail(
			ctx,
			args,
			fixedTailCount,
			responseFileLoads,
		)
		if err != nil {
			return false, err
		}
	}
	var condition grammarConditionDemandFunc
	var defaults grammarDefaultDemandFunc
	if invocation != nil {
		condition = invocation.conditionFor(ctx)
		defaults = invocation.defaultCommandFor(ctx)
	}
	model := snapshot.invocation.analysis.effectiveModel
	var materializedNamespaces map[CommandID]providerNamespaceOutcome
	if invocation != nil {
		materializedNamespaces = invocation.providerNamespaces()
	}
	namespace := func(demand grammarNamespaceDemand) grammarNamespaceMaterialization {
		providers := effectiveProviderRecords(
			providerRecordsForOwner(model, demand.Command),
			demand.Command,
			demand.Grammar.SelectedProfiles,
		)
		if materialized, ok := materializedNamespaces[demand.Command]; ok &&
			materialized.matchesProviderSet(providers) {
			materialization := grammarNamespaceMaterialization{
				Complete: materialized.complete,
			}
			return materialization
		}
		materialization := grammarNamespaceMaterialization{
			Complete: len(providers) == 0,
		}
		return materialization
	}
	analysis := analyzeGrammarArguments(
		ctx,
		snapshot.invocation,
		args,
		namespace,
		condition,
		defaults,
		nil,
	)
	if len(analysis.Issues) == 0 {
		return true, nil
	}
	for _, issue := range analysis.Issues {
		// Only a defect attached to the substituted cursor token or an already
		// present complete token after it disproves this candidate. A requirement
		// reported at the end boundary can still be satisfied by later residual
		// input inserted before the immutable fixed suffix.
		if issue.TokenIndex < snapshot.input.analysisCursorArgument || issue.TokenIndex >= len(args) {
			continue
		}
		return false, nil
	}
	return true, nil
}

func (operation *completionOperation) expandCompletionShortcutCandidateTail(ctx context.Context, args []string, fixedTailCount int, responseFileLoads *completionResponseFileLoadCache) ([]string, error) {
	snapshot := operation
	invocation := operation.invocation
	if fixedTailCount <= 0 || fixedTailCount > len(args) || operation.invocation.analysis.responses == nil || responseFileLoads == nil {
		return args, nil
	}
	tailStart := len(args) - fixedTailCount
	var condition grammarConditionDemandFunc
	var defaults grammarDefaultDemandFunc
	if invocation != nil {
		condition = invocation.conditionFor(ctx)
		defaults = invocation.defaultCommandFor(ctx)
	}
	grammarCursor := prepareGrammarArgumentsPrefixCursor(
		ctx,
		snapshot.invocation,
		args[:tailStart],
		-1,
		nil,
		condition,
		defaults,
	)
	for index := tailStart; index < len(args); {
		if !grammarCursor.Ready() {
			return args, nil
		}
		dialect, ok := grammarCursor.Dialect()
		if !ok || !dialect.ResponseFiles() {
			grammarCursor.Append(ctx, args[index])
			index++
			continue
		}
		path, literal, include := recognizeResponseArgument(args[index])
		if !include {
			if literal != args[index] {
				args[index] = literal
			}
			grammarCursor.Append(ctx, literal)
			index++
			continue
		}
		recognition := grammarResponseStreamRecognition(ctx, grammarCursor)
		expansion, err := operation.invocation.analysis.responses.StageWithStreamRecognition(
			ctx,
			responseFileLoads.load(snapshot.invocation.input.responseFiles),
			responseReference{
				Path: path,
				Base: snapshot.invocation.input.workingDirectory,
			},
			recognition,
		)
		if err != nil {
			failure := &responseArgumentExpansionError{
				argument: index,
				err:      err,
			}
			return nil, failure
		}
		texts := expansion.TokenTexts()
		next := make([]string, 0, len(args)-1+len(texts))
		next = append(next, args[:index]...)
		next = append(next, texts...)
		next = append(next, args[index+1:]...)
		args = next
		index += len(texts)
	}
	return args, nil
}

func staticCommandCompletionItem(reference Reference, client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	var item CompletionItem
	switch candidate.Kind {
	case completionSyntaxCandidateCommandEntry:
		entry, ok := reference.CommandEntries[candidate.CommandEntry]
		if !ok || !entry.Visible {
			return CompletionItem{}, false
		}
		kind := CompletionCandidateCommand
		switch entry.Kind {
		case CommandEntryAlias:
			kind = CompletionCandidateCommandAlias
		case CommandEntryShortcut:
			kind = CompletionCandidateCommandShortcut
		case CommandEntryCanonical:
		default:
			return CompletionItem{}, false
		}
		item.CompletionCandidate = CompletionCandidate{
			Value:          entry.Name,
			DisplayText:    entry.Name,
			Kind:           kind,
			SortText:       entry.Name,
			LifecycleState: entry.LifecycleState,
		}
		item.Declaration = DeclarationIdentity{
			Kind:         DeclarationKindCommandEntry,
			CommandEntry: entry.ID,
		}
		if client.Capabilities.Descriptions {
			item.Description = entry.Description.Short
		}
	case completionSyntaxCandidateCommandAction:
		entry, ok := reference.ActionEntries[candidate.ActionEntry]
		if !ok || !entry.Visible || entry.Kind != ActionEntryCommand || entry.Name == "" {
			return CompletionItem{}, false
		}
		item.CompletionCandidate = CompletionCandidate{
			Value:          entry.Name,
			DisplayText:    entry.Name,
			Kind:           CompletionCandidateAction,
			SortText:       entry.Name,
			LifecycleState: entry.LifecycleState,
		}
		item.Declaration = DeclarationIdentity{
			Kind:        DeclarationKindActionEntry,
			ActionEntry: entry.ID,
		}
		if client.Capabilities.Descriptions {
			item.Description = entry.Description.Short
		}
	default:
		return CompletionItem{}, false
	}
	if !client.Capabilities.CandidateKinds {
		item.Kind = CompletionCandidateUnknown
	}
	item.TargetIndexes = []int{targetIndex}
	item.Match = emptyPrefixCompletionMatch(client)
	return item, true
}

func staticFlagCompletionItem(reference Reference, client CompletionClient, candidate completionSyntaxCandidate, targetIndex int) (CompletionItem, bool) {
	var item CompletionItem
	switch candidate.Kind {
	case completionSyntaxCandidateFlagSpelling:
		spelling, ok := reference.FlagSpellings[candidate.FlagSpelling]
		if !ok || !spelling.Visible {
			return CompletionItem{}, false
		}
		flag, ok := reference.Flags[spelling.Flag]
		if !ok || !flag.Visible {
			return CompletionItem{}, false
		}
		item.CompletionCandidate = CompletionCandidate{
			Value:          spelling.Spelling.Text,
			DisplayText:    spelling.Spelling.Text,
			Kind:           CompletionCandidateFlag,
			SortText:       spelling.Spelling.Text,
			LifecycleState: spelling.LifecycleState,
		}
		item.Declaration = DeclarationIdentity{
			Kind:         DeclarationKindFlagSpelling,
			FlagSpelling: spelling.ID,
		}
		if client.Capabilities.Descriptions {
			item.Description = flag.Description
		}
	case completionSyntaxCandidateFlagAction:
		entry, ok := reference.ActionEntries[candidate.ActionEntry]
		if !ok || !entry.Visible || entry.Kind != ActionEntryFlag {
			return CompletionItem{}, false
		}
		text := actionFlagText(entry)
		if text == "" {
			return CompletionItem{}, false
		}
		item.CompletionCandidate = CompletionCandidate{
			Value:          text,
			DisplayText:    text,
			Kind:           CompletionCandidateAction,
			SortText:       text,
			LifecycleState: entry.LifecycleState,
		}
		item.Declaration = DeclarationIdentity{
			Kind:        DeclarationKindActionEntry,
			ActionEntry: entry.ID,
		}
		if client.Capabilities.Descriptions {
			item.Description = entry.Description.Short
		}
	default:
		return CompletionItem{}, false
	}
	if !client.Capabilities.CandidateKinds {
		item.Kind = CompletionCandidateUnknown
	}
	item.TargetIndexes = []int{targetIndex}
	item.Match = emptyPrefixCompletionMatch(client)
	return item, true
}

func actionFlagText(entry ActionEntryReference) string {
	switch {
	case entry.Long != "":
		return entry.Long
	case entry.Short != 0:
		return "-" + string(entry.Short)
	case entry.Literal != "":
		return entry.Literal
	default:
		return ""
	}
}

func emptyPrefixCompletionMatch(CompletionClient) CompletionMatch {
	return CompletionMatch{}
}

func matchCompletionItem(item CompletionItem, matcher completionMatcher, scratch *completionMatchScratch) (CompletionItem, bool) {
	match, ok := matcher.matchCandidate(item.CompletionCandidate, scratch)
	if !ok {
		return CompletionItem{}, false
	}
	item.Match = match
	return item, true
}

func completionMatchQuery(snapshot *completionOperation) string {
	return snapshot.invocation.Args()[snapshot.input.analysisCursorArgument][:snapshot.input.cursor.Byte]
}

func completionItemCompare(left, right *CompletionItem, mode CompletionRankingMode) int {
	leftText, rightText := left.SortText, right.SortText
	if leftText == "" {
		leftText = left.Value
	}
	if rightText == "" {
		rightText = right.Value
	}
	lexical := func() int {
		if order := cmp.Compare(leftText, rightText); order != 0 {
			return order
		}
		return cmp.Compare(left.Value, right.Value)
	}
	switch mode {
	case CompletionRankingModeRelevance:
		if order := cmp.Compare(right.Priority, left.Priority); order != 0 {
			return order
		}
		return lexical()
	case CompletionRankingModeLexical:
		return lexical()
	default:
		return 0
	}
}

func completionItemIndexCompare(items []CompletionItem, mode CompletionRankingMode, left, right int) int {
	if order := completionItemCompare(&items[left], &items[right], mode); order != 0 {
		return order
	}
	return cmp.Compare(left, right)
}

func completionItemIndexLess(items []CompletionItem, mode CompletionRankingMode, left, right int) bool {
	return completionItemIndexCompare(items, mode, left, right) < 0
}

func applyStaticCompletionRanking(items []CompletionItem, mode CompletionRankingMode) {
	switch mode {
	case CompletionRankingModeRelevance, CompletionRankingModeLexical:
		slices.SortStableFunc(items, func(left, right CompletionItem) int { return completionItemCompare(&left, &right, mode) })
	case CompletionRankingModeSource, CompletionRankingModeUnranked:
	}
}

func canonicalizeCompletionResult(result *Completion) {
	if result == nil {
		return
	}
	if len(result.Targets) == 0 {
		result.Targets = nil
	}
	for i := range result.Targets {
		if len(result.Targets[i].CommandPath) == 0 {
			result.Targets[i].CommandPath = nil
		}
	}
	if len(result.Selection.Path) == 0 {
		result.Selection.Path = nil
	}
	if len(result.Selection.Steps) == 0 {
		result.Selection.Steps = nil
	}
	for i := range result.Selection.Steps {
		if len(result.Selection.Steps[i].Commands) == 0 {
			result.Selection.Steps[i].Commands = nil
		}
	}
	if len(result.Items) == 0 {
		result.Items = nil
	}
	for i := range result.Items {
		item := &result.Items[i]
		if len(item.Replacement) == 0 {
			item.Replacement = nil
		}
		if len(item.SearchTerms) == 0 {
			item.SearchTerms = nil
		}
		if len(item.CommitCharacters) == 0 {
			item.CommitCharacters = nil
		}
		if len(item.TargetIndexes) == 0 {
			item.TargetIndexes = nil
		}
		if len(item.Match.Ranges) == 0 {
			item.Match.Ranges = nil
		}
	}
	if len(result.Groups) == 0 {
		result.Groups = nil
	}
	if len(result.Messages) == 0 {
		result.Messages = nil
	}
	if len(result.NativeRequests) == 0 {
		result.NativeRequests = nil
	}
	if result.Delegation != nil && len(result.Delegation.Args) == 0 {
		result.Delegation.Args = nil
	}
	if len(result.Sources) == 0 {
		result.Sources = nil
	}
	if len(result.Origins) == 0 {
		result.Origins = nil
	} else {
		for id, origin := range result.Origins {
			if len(origin.Parents) == 0 {
				origin.Parents = nil
				result.Origins[id] = origin
			}
		}
	}
	if len(result.Diagnostics) == 0 {
		result.Diagnostics = nil
	}
	for i := range result.Diagnostics {
		diagnostic := &result.Diagnostics[i]
		if len(diagnostic.CommandPath) == 0 {
			diagnostic.CommandPath = nil
		}
		if len(diagnostic.Origins) == 0 {
			diagnostic.Origins = nil
		}
		if len(diagnostic.Suggestions) == 0 {
			diagnostic.Suggestions = nil
		}
		if len(diagnostic.Usage) == 0 {
			diagnostic.Usage = nil
		}
		if len(diagnostic.RepairIndexes) == 0 {
			diagnostic.RepairIndexes = nil
		}
		if diagnostic.Rule != nil {
			if len(diagnostic.Rule.Subjects) == 0 {
				diagnostic.Rule.Subjects = nil
			}
			if len(diagnostic.Rule.Operands) == 0 {
				diagnostic.Rule.Operands = nil
			}
		}
	}
	if len(result.Repairs) == 0 {
		result.Repairs = nil
	}
	for i := range result.Repairs {
		repair := &result.Repairs[i]
		if len(repair.Sources) == 0 {
			repair.Sources = nil
		}
		for j := range repair.Sources {
			source := &repair.Sources[j]
			if len(source.TextEdits) == 0 {
				source.TextEdits = nil
			}
			for k := range source.TextEdits {
				if len(source.TextEdits[k].Replacement) == 0 {
					source.TextEdits[k].Replacement = nil
				}
			}
			if len(source.EntryEdits) == 0 {
				source.EntryEdits = nil
			}
			for k := range source.EntryEdits {
				entryEdit := &source.EntryEdits[k]
				if len(entryEdit.Entries) == 0 {
					entryEdit.Entries = nil
				}
				for m := range entryEdit.Entries {
					if len(entryEdit.Entries[m]) == 0 {
						entryEdit.Entries[m] = nil
					}
				}
			}
		}
	}
}

type completionInput struct {
	cursor                 CompletionCursor
	analysisCursorArgument int
	trigger                CompletionTrigger
	visibility             CompletionVisibility
	client                 CompletionClient
	limit                  int
}

func completionAnalysisInput(request CompletionRequest) ([]string, CompletionCursor, error) {
	if request.ObservedInput == nil {
		return append([]string(nil), request.Args...), request.Cursor, nil
	}
	if request.Args != nil || request.Cursor != (CompletionCursor{}) {
		return nil, CompletionCursor{}, errInvalidCompletionRequest
	}
	input := request.ObservedInput
	if len(input.Words) == 0 || input.Cursor.Word < 0 || input.Cursor.Word >= len(input.Words) ||
		input.Cursor.Byte < 0 {
		return nil, CompletionCursor{}, errInvalidCompletionRequest
	}
	for _, word := range input.Words {
		if word.Fidelity != CompletionInputFidelityExact && word.Fidelity != CompletionInputFidelityLexical &&
			word.Fidelity != CompletionInputFidelityExpanded {
			return nil, CompletionCursor{}, errInvalidCompletionRequest
		}
	}
	exactPrefix := 0
	switch input.Topology {
	case CompletionArgvTopologyExact:
		exactPrefix = len(input.Words)
	case CompletionArgvTopologyPrefixExact:
		if input.ExactArgvPrefix < 0 || input.ExactArgvPrefix > len(input.Words) {
			return nil, CompletionCursor{}, errInvalidCompletionRequest
		}
		exactPrefix = input.ExactArgvPrefix
	case CompletionArgvTopologyUncertain:
		if input.ExactArgvPrefix != 0 {
			return nil, CompletionCursor{}, errInvalidCompletionRequest
		}
	default:
		return nil, CompletionCursor{}, errInvalidCompletionRequest
	}

	settledPrefix := exactPrefix
	for i := 0; i < settledPrefix; i++ {
		if input.Words[i].Fidelity != CompletionInputFidelityExact {
			settledPrefix = i
			break
		}
	}
	analysisWord := min(input.Cursor.Word, settledPrefix)
	if analysisWord >= len(input.Words) {
		return nil, CompletionCursor{}, errInvalidCompletionRequest
	}

	words := input.Words[:analysisWord+1]
	args := make([]string, len(words))
	for i, word := range words {
		args[i] = word.Value
	}
	byteOffset := input.Cursor.Byte
	if analysisWord != input.Cursor.Word {
		byteOffset = len(args[analysisWord])
	}
	if byteOffset > len(args[analysisWord]) {
		return nil, CompletionCursor{}, errInvalidCompletionRequest
	}
	switch input.Cursor.Space {
	case CompletionCursorSpaceArgumentBytes:
		if input.Words[input.Cursor.Word].Fidelity != CompletionInputFidelityExact || input.Cursor.Word >= exactPrefix {
			return nil, CompletionCursor{}, errInvalidCompletionRequest
		}
	case CompletionCursorSpaceObservedWordBytes:
	case CompletionCursorSpaceSourceBytes:
		if analysisWord != input.Cursor.Word || input.Cursor.Byte != len(input.Words[input.Cursor.Word].Value) {
			return nil, CompletionCursor{}, errInvalidCompletionRequest
		}
	default:
		return nil, CompletionCursor{}, errInvalidCompletionRequest
	}
	cursor := CompletionCursor{
		Argument: analysisWord,
		Byte:     byteOffset,
	}
	return args, cursor, nil
}

// staticCompletionGroupID projects the canonical static presentation classification for a
// declaration-backed Completion candidate. The returned identity is deliberately non-display:
// canonical command scope and source-language group titles participate in identity, while
// localized Help titles do not. Inherited flags retain the classification of their owning scope.
// Dynamic callback-authored Completion groups remain independent.
func staticCompletionGroupID(operation *invocation, grammar grammarResult, current CommandID, currentPath []string, candidate completionSyntaxCandidate) string {
	if operation == nil || !current.present() {
		return ""
	}
	model := operation.analysis.effectiveModel
	if !model.Valid() {
		return ""
	}

	switch candidate.Kind {
	case completionSyntaxCandidateCommandEntry:
		record, ok := model.CommandEntry(candidate.CommandEntry)
		if !ok {
			return ""
		}
		if group, ok := staticCommandHelpGroup(model, grammar, current, record.meta.id); ok {
			return encodeStaticCompletionGroupID(
				"command",
				HelpGroupDeclared,
				currentPath,
				group.profile,
				group.title,
			)
		}
		return encodeStaticCompletionGroupID("command", HelpGroupDefault, currentPath, 0, "")
	case completionSyntaxCandidateCommandAction:
		return encodeStaticCompletionGroupID("command", HelpGroupDefault, currentPath, 0, "")
	case completionSyntaxCandidateFlagSpelling:
		spelling, ok := model.FlagSpelling(candidate.FlagSpelling)
		if !ok {
			return ""
		}
		flag, ok := model.Flag(spelling.flag)
		if !ok {
			return ""
		}
		owner := flag.command
		ownerPath := currentPath
		if owner != current {
			ownerPath = commandPathNames(model, owner)
		}
		if group, ok := staticFlagHelpGroup(model, grammar, owner, flag.meta.id); ok {
			return encodeStaticCompletionGroupID(
				"flag",
				HelpGroupDeclared,
				ownerPath,
				group.profile,
				group.title,
			)
		}
		return encodeStaticCompletionGroupID("flag", HelpGroupDefault, ownerPath, 0, "")
	case completionSyntaxCandidateFlagAction:
		record, ok := model.ActionEntry(candidate.ActionEntry)
		if !ok {
			return ""
		}
		ownerPath := currentPath
		if record.parent != current {
			ownerPath = commandPathNames(model, record.parent)
		}
		return encodeStaticCompletionGroupID("flag", HelpGroupDefault, ownerPath, 0, "")
	default:
		return ""
	}
}

func staticCommandHelpGroup(model *effectiveModel, grammar grammarResult, command CommandID, entry CommandEntryID) (helpGroupRecord, bool) {
	provider, fromProvider := model.ProviderForCommandEntry(entry)
	for _, group := range model.HelpGroupsForCommand(command) {
		if group.command != command || group.family != helpGroupFamilyCommand ||
			!structuralProfileActive(grammar.SelectedProfiles, command, group.profile) {
			continue
		}
		for _, member := range group.CommandEntries() {
			switch member.Kind() {
			case HelpCommandGroupEntryCommandEntry:
				if member.CommandEntry() == entry {
					return group, true
				}
			case HelpCommandGroupEntryCommandProvider:
				if fromProvider && member.Provider() == provider {
					return group, true
				}
			case HelpCommandGroupEntryUnknown:
			}
		}
	}
	return helpGroupRecord{}, false
}

func staticFlagHelpGroup(model *effectiveModel, grammar grammarResult, command CommandID, flag FlagID) (helpGroupRecord, bool) {
	for _, group := range model.HelpGroupsForCommand(command) {
		if group.command != command || group.family != helpGroupFamilyFlag ||
			!structuralProfileActive(grammar.SelectedProfiles, command, group.profile) {
			continue
		}
		if slices.Contains(group.Flags(), flag) {
			return group, true
		}
	}
	return helpGroupRecord{}, false
}

func encodeStaticCompletionGroupID(family string, kind HelpGroupKind, scope []string, profile GrammarProfileID, sourceTitle string) string {
	var result strings.Builder
	fmt.Fprintf(&result, "commandment:static:%s:%d:%d:%d:", family, kind, profile, len(scope))
	for _, segment := range scope {
		fmt.Fprintf(&result, "%d:%s", len(segment), segment)
	}
	fmt.Fprintf(&result, "%d:%s", len(sourceTitle), sourceTitle)
	return result.String()
}

func appendStaticCompletionGroup(result Completion, id string) Completion {
	if id == "" {
		return result
	}
	for _, group := range result.Groups {
		if group.ID == id {
			return result
		}
	}
	result.Groups = append(result.Groups, CompletionGroup{
		ID: id,
	})
	return result
}
