package cli

import "slices"

const (
	diagnosticValueKindTokenBool     = "bool"
	diagnosticValueKindTokenCount    = "count"
	diagnosticValueKindTokenInt      = "int"
	diagnosticValueKindTokenUint     = "uint"
	diagnosticValueKindTokenString   = "string"
	diagnosticValueKindTokenSequence = "sequence"
	diagnosticValueKindTokenKeyed    = "keyed"
	diagnosticValueKindTokenCustom   = "custom"
)

func diagnosticReportFromError(err error) DiagnosticReport {
	report, extractionErr := extract(err)
	if extractionErr != nil {
		panic("cli: invalid diagnostic payload: " + extractionErr.Error())
	}
	return projectDiagnosticReport(report)
}

func projectDiagnosticFailure(code DiagnosticCode, message string, cause error) error {
	return retainDiagnosticFailure(payload{
		Diagnostics: []diagnostic{{
			Code:             string(code),
			Message:          message,
			FrameworkMessage: true,
		}},
	}, cause)
}

func operationDiagnosticFailure(operation *invocation, code DiagnosticCode, message string, cause error) error {
	return retainDiagnosticFailure(payload{
		Diagnostics: []diagnostic{{
			Code:             string(code),
			Message:          message,
			FrameworkMessage: true,
		}},
		Presentation: operation.FrameworkPresentation(),
	}, cause)
}

func operationDiagnosticPayload(operation *invocation, truncated bool) payload {
	payload := payload{
		Sources:      map[sourceID]source{},
		Origins:      map[originID]origin{},
		Truncated:    truncated,
		Presentation: operation.FrameworkPresentation(),
	}
	args := operation.Args()
	env := operation.Env()
	for _, inputSource := range operation.Sources() {
		id := sourceID(inputSource.id.uint32())
		projected := source{
			ID:              id,
			Kind:            inputSource.kind,
			Revision:        inputSource.revision,
			Key:             inputSource.key,
			Location:        inputSource.location,
			Sensitive:       inputSource.sensitive,
			CommandShortcut: inputSource.commandShortcut,
		}
		switch inputSource.id {
		case operation.ArgumentSource().ID():
			projected.Entries = append([]string(nil), args...)
		case operation.EnvironmentSource().ID():
			projected.Entries = make([]string, len(env))
			for i, entry := range env {
				projected.Entries[i] = entry.Name + "=" + entry.Value
			}
		default:
			if inputSource.kind == InputKindShortcut {
				projected.Entries = operation.ShortcutInputEntries(inputSource.id)
			}
		}
		payload.Sources[id] = projected
	}
	for _, inputOrigin := range operation.Origins() {
		id := originID(inputOrigin.id.uint32())
		originSpan := inputOrigin.span
		parents := make([]originID, len(inputOrigin.Parents()))
		for i, parent := range inputOrigin.Parents() {
			parents[i] = originID(parent.uint32())
		}
		payload.Origins[id] = origin{
			ID: id,
			Span: span{
				Source: sourceID(originSpan.Source.uint32()),
				Entry:  originSpan.Entry,
				Start: analysisPosition{
					Offset: originSpan.Start,
				},
				End: analysisPosition{
					Offset: originSpan.End,
				},
			},
			Parents:   parents,
			Sensitive: inputOrigin.sensitive,
		}
	}
	return payload
}

func projectDiagnosticReport(input report) DiagnosticReport {
	if len(input.Diagnostics) == 0 && len(input.Sources) == 0 && len(input.Origins) == 0 && len(input.Repairs) == 0 &&
		!input.Truncated {
		return DiagnosticReport{}
	}
	result := DiagnosticReport{
		Diagnostics: make([]Diagnostic, len(input.Diagnostics)),
		Sources:     make(map[InputSourceID]InputSource, len(input.Sources)),
		Origins:     make(map[InputOriginID]InputOrigin, len(input.Origins)),
		Repairs:     make([]InputRepair, len(input.Repairs)),
		Truncated:   input.Truncated,
	}
	for id, source := range input.Sources {
		publicID := inputSourceID(id)
		result.Sources[publicID] = InputSource{
			ID:   publicID,
			Kind: source.Kind,
			Revision: InputRevision{
				value: source.Revision,
			},
			Key:             source.Key,
			Location:        source.Location,
			Sensitive:       source.Sensitive,
			CommandShortcut: source.CommandShortcut,
		}
	}
	for id, origin := range input.Origins {
		publicID := inputOriginID(id)
		parents := make([]InputOriginID, len(origin.Parents))
		for i, parent := range origin.Parents {
			parents[i] = inputOriginID(parent)
		}
		result.Origins[publicID] = InputOrigin{
			ID:        publicID,
			Span:      inputSpan(origin.Span),
			Parents:   parents,
			Sensitive: origin.Sensitive,
		}
	}
	for i, repair := range input.Repairs {
		result.Repairs[i] = inputRepair(repair)
	}
	for i, diagnostic := range input.Diagnostics {
		message := diagnostic.Message
		if diagnostic.FrameworkMessage {
			message = input.ResolveFrameworkText(message)
		}
		reason := diagnostic.Reason
		if diagnostic.FrameworkReason {
			reason = input.ResolveFrameworkText(reason)
		}
		hint := diagnostic.Hint
		if diagnostic.FrameworkHint {
			hint = input.ResolveFrameworkText(hint)
		}
		result.Diagnostics[i] = Diagnostic{
			Code:          DiagnosticCode(diagnostic.Code),
			Rule:          diagnosticRuleFailure(diagnostic.Rule),
			AdvisoryCode:  AdvisoryCode(diagnostic.AdvisoryCode),
			Message:       message,
			Reason:        reason,
			Hint:          hint,
			Subject:       diagnostic.Subject,
			CommandPath:   append([]string(nil), diagnostic.CommandPath...),
			Command:       diagnostic.Command,
			Declaration:   diagnosticDeclarationIdentity(diagnostic.Declaration),
			Provider:      diagnostic.Provider,
			Default:       diagnostic.Default,
			InputSource:   inputSourceID(diagnostic.Source),
			Origins:       inputOriginIDs(diagnostic.Origins),
			Input:         diagnostic.Input,
			InputRedacted: diagnostic.InputRedacted,
			Suggestions:   append([]string(nil), diagnostic.Suggestions...),
			Usage:         append([]string(nil), diagnostic.Usage...),
			Sensitive:     diagnostic.Sensitive,
			ExitCode:      diagnostic.ExitCode,
			ProducerKind:  diagnosticProducerKind(diagnostic.ProducerKind),
			Tier:          diagnosticResolutionTier(diagnostic.Tier),
			Disposition:   diagnosticDisposition(diagnostic.Disposition),
			RepairIndexes: append([]int(nil), diagnostic.RepairIndexes...),
		}
		if diagnostic.ProviderOutput != nil {
			result.Diagnostics[i].ProviderOutput = &CommandProviderOutputLocator{
				Index:    diagnostic.ProviderOutput.Index,
				Kind:     diagnostic.ProviderOutput.Kind,
				Key:      diagnostic.ProviderOutput.Key,
				Location: diagnostic.ProviderOutput.Location,
				Redacted: diagnostic.ProviderOutput.Redacted,
			}
		}
		if diagnostic.Replacement != nil {
			result.Diagnostics[i].Replacement = &DeclarationReplacement{
				Target:                          diagnosticDeclarationIdentity(diagnostic.Replacement.Target),
				Presentation:                    diagnostic.Replacement.Presentation,
				TargetDeprecatedDuringMigration: diagnostic.Replacement.TargetDeprecatedDuringMigration,
			}
		}
		if diagnostic.Span != nil {
			span := inputSpan(*diagnostic.Span)
			result.Diagnostics[i].Span = &span
		}
		if diagnostic.SignalAction != nil {
			action := projectDiagnosticSignalActionFailure(*diagnostic.SignalAction)
			result.Diagnostics[i].SignalAction = &action
		}
		if diagnostic.SignalOverflow != nil {
			overflow := projectDiagnosticSignalOverflowFailure(*diagnostic.SignalOverflow)
			result.Diagnostics[i].SignalOverflow = &overflow
		}
		if diagnostic.NoticeDelivery != nil {
			notice := projectDiagnosticNoticeDeliveryFailure(*diagnostic.NoticeDelivery)
			result.Diagnostics[i].NoticeDelivery = &notice
		}
	}
	return result
}

func diagnosticRuleFailure(input *analysisDiagnosticRuleFailure) *RuleFailure {
	if input == nil {
		return nil
	}
	result := &RuleFailure{
		RuleID:        RuleID(input.ID),
		RuleKind:      diagnosticRuleKind(input.Kind),
		RuleCode:      RuleCode(input.Code),
		ViolationKind: diagnosticViolationKind(input.ViolationKind),
		Subjects:      make([]ValueID, len(input.Subjects)),
		Operands:      make([]RuleOperand, len(input.Operands)),
		Pattern:       input.Pattern,
	}
	for i, subject := range input.Subjects {
		result.Subjects[i] = ValueID(subject)
	}
	for i, operand := range input.Operands {
		result.Operands[i].Sensitive = operand.Sensitive
		if operand.Literal != nil {
			result.Operands[i].Literal = &ValueLiteral{
				Type:      diagnosticValueType(operand.Literal.Type),
				Text:      operand.Literal.Text,
				Sensitive: operand.Literal.Sensitive,
				Redacted:  operand.Literal.Redacted,
			}
		}
	}
	if input.CountValid {
		result.Count = CountRange{
			minimum:   input.CountMinimum,
			maximum:   input.CountMaximum,
			unbounded: input.CountUnbounded,
			valid:     true,
		}
	}
	return result
}

func diagnosticValueType(input analysisDiagnosticValueType) ValueType {
	var kind ValueKind
	switch input.Kind {
	case diagnosticValueKindTokenBool:
		kind = ValueKindBool
	case diagnosticValueKindTokenCount:
		kind = ValueKindCount
	case diagnosticValueKindTokenInt:
		kind = ValueKindInt
	case diagnosticValueKindTokenUint:
		kind = ValueKindUint
	case diagnosticValueKindTokenString:
		kind = ValueKindString
	case diagnosticValueKindTokenSequence:
		kind = ValueKindSequence
	case diagnosticValueKindTokenKeyed:
		kind = ValueKindKeyed
	case diagnosticValueKindTokenCustom:
		kind = ValueKindCustom
	}
	return ValueType{
		Kind: kind,
		Name: input.Name,
	}
}

func projectDiagnosticSignalActionFailure(input diagnosticSignalActionFailure) SignalActionFailure {
	failure := SignalActionFailure{
		CommandPath: append([]string(nil), input.CommandPath...),
		PolicyID:    SignalPolicyID(input.PolicyID.uint32()),
		Index:       input.Index,
		Action:      projectDiagnosticSignalAction(input.Action),
		Event:       projectDiagnosticSignalEvent(input.Event),
		Phase:       input.Phase,
	}

	return failure
}

func projectDiagnosticSignalAction(input diagnosticSignalAction) SignalAction {
	action := SignalAction{
		Kind:           projectDiagnosticSignalActionKind(input.Kind),
		Name:           input.Name,
		EndpointID:     SignalEndpointID(input.EndpointID.uint32()),
		Signal:         input.Signal,
		SignalValueID:  ValueID(input.SignalValue.uint32()),
		SignalRedacted: input.SignalRedacted,
		Delayed:        input.Delayed,
		Delay:          input.Delay,
		DelayValueID:   ValueID(input.DelayValue.uint32()),
		DelayRedacted:  input.DelayRedacted,
		ExitCode:       input.ExitCode,
		Sensitive:      input.Sensitive,
	}

	return action
}

func projectDiagnosticSignalActionKind(kind diagnosticSignalActionKind) SignalActionKind {
	switch kind {
	case diagnosticSignalActionIgnore:
		return SignalActionIgnore
	case diagnosticSignalActionCancel:
		return SignalActionCancel
	case diagnosticSignalActionHandle:
		return SignalActionHandle
	case diagnosticSignalActionForward:
		return SignalActionForward
	case diagnosticSignalActionSend:
		return SignalActionSend
	case diagnosticSignalActionAbort:
		return SignalActionAbort
	case diagnosticSignalActionReraise:
		return SignalActionReraise
	case diagnosticSignalActionEmit:
		return SignalActionEmit
	default:
		return SignalActionUnknown
	}
}

func projectDiagnosticSignalEvent(input diagnosticSignalEvent) SignalEvent {
	var sender *SignalSender
	if input.Sender != nil {
		sender = &SignalSender{
			ProcessID: input.Sender.ProcessID,
			UserID:    input.Sender.UserID,
			Code:      input.Sender.Code,
		}
	}
	event := SignalEvent{
		Signal:          input.Signal,
		Origin:          input.Origin,
		Sender:          sender,
		Phase:           input.Phase,
		ObservedAt:      input.ObservedAt,
		FirstObservedAt: input.FirstObservedAt,
		LastObservedAt:  input.LastObservedAt,
		FirstOccurrence: input.FirstOccurrence,
		LastOccurrence:  input.LastOccurrence,
		BatchSize:       input.BatchSize,
	}

	return event
}

func projectDiagnosticSignalOverflowFailure(input diagnosticSignalOverflowFailure) SignalOverflowFailure {
	failure := SignalOverflowFailure{
		Boundary:   projectDiagnosticSignalOverflowBoundary(input.Boundary),
		EndpointID: SignalEndpointID(input.EndpointID.uint32()),
		Capacity:   input.Capacity,
		Buffered:   input.Buffered,
	}

	return failure
}

func projectDiagnosticSignalOverflowBoundary(boundary diagnosticSignalOverflowBoundary) SignalOverflowBoundary {
	switch boundary {
	case diagnosticSignalOverflowSource:
		return SignalOverflowBoundarySource
	case diagnosticSignalOverflowRouting:
		return SignalOverflowBoundaryRouting
	case diagnosticSignalOverflowEndpoint:
		return SignalOverflowBoundaryEndpoint
	default:
		return SignalOverflowBoundaryUnknown
	}
}

func projectDiagnosticNoticeDeliveryFailure(input diagnosticNoticeDeliveryFailure) NoticeDeliveryFailure {
	failure := NoticeDeliveryFailure{
		Index:       input.Index,
		Phase:       projectDiagnosticNoticePhase(input.Phase),
		Code:        NoticeCode(input.Code),
		CommandPath: append([]string(nil), input.CommandPath...),
	}

	return failure
}

func projectDiagnosticNoticePhase(phase analysisDiagnosticNoticePhase) NoticePhase {
	switch phase {
	case diagnosticNoticePhaseAnalysis:
		return NoticePhaseAnalysis
	case diagnosticNoticePhaseExecution:
		return NoticePhaseExecution
	case diagnosticNoticePhaseCleanup:
		return NoticePhaseCleanup
	default:
		return NoticePhaseUnknown
	}
}

func diagnosticRuleKind(kind analysisDiagnosticRuleKind) RuleKind {
	switch kind {
	case diagnosticRuleCustom:
		return RuleCustom
	case diagnosticRuleRequire:
		return RuleRequire
	case diagnosticRuleForbid:
		return RuleForbid
	case diagnosticRuleNotEmpty:
		return RuleNotEmpty
	case diagnosticRuleIn:
		return RuleIn
	case diagnosticRuleNotIn:
		return RuleNotIn
	case diagnosticRuleEqual:
		return RuleEqual
	case diagnosticRuleNotEqual:
		return RuleNotEqual
	case diagnosticRuleBetween:
		return RuleBetween
	case diagnosticRuleAtLeast:
		return RuleAtLeast
	case diagnosticRuleAtMost:
		return RuleAtMost
	case diagnosticRuleGreaterThan:
		return RuleGreaterThan
	case diagnosticRuleLessThan:
		return RuleLessThan
	case diagnosticRuleMatch:
		return RuleMatch
	case diagnosticRuleItemCount:
		return RuleItemCount
	case diagnosticRuleContains:
		return RuleContains
	case diagnosticRuleEachIn:
		return RuleEachIn
	case diagnosticRuleUnique:
		return RuleUnique
	case diagnosticRuleEntryCount:
		return RuleEntryCount
	case diagnosticRuleHasKeys:
		return RuleHasKeys
	case diagnosticRuleKeysIn:
		return RuleKeysIn
	case diagnosticRuleSubsetOf:
		return RuleSubsetOf
	case diagnosticRuleDisjoint:
		return RuleDisjoint
	case diagnosticRuleKeysSubsetOf:
		return RuleKeysSubsetOf
	case diagnosticRuleKeysDisjoint:
		return RuleKeysDisjoint
	case diagnosticRuleExactlyOne:
		return RuleExactlyOne
	case diagnosticRuleAllOrNone:
		return RuleAllOrNone
	case diagnosticRuleAtMostOne:
		return RuleAtMostOne
	case diagnosticRuleAtLeastOne:
		return RuleAtLeastOne
	default:
		return RuleUnknown
	}
}

func diagnosticViolationKind(kind analysisDiagnosticViolationKind) ViolationKind {
	switch kind {
	case diagnosticViolationInvalid:
		return ViolationInvalid
	case diagnosticViolationConflict:
		return ViolationConflict
	case diagnosticViolationDuplicate:
		return ViolationDuplicate
	default:
		return ViolationUnknown
	}
}

func diagnosticProducerKind(producerKind analysisDiagnosticProducerKind) ValueProducerKind {
	switch producerKind {
	case diagnosticProducerKindFlag:
		return ValueProducerKindFlag
	case diagnosticProducerKindArgument:
		return ValueProducerKindArgument
	case diagnosticProducerKindRemainder:
		return ValueProducerKindRemainder
	case diagnosticProducerKindPassThrough:
		return ValueProducerKindPassThrough
	case diagnosticProducerKindShortcutDefault:
		return ValueProducerKindShortcutDefault
	case diagnosticProducerKindEnvironment:
		return ValueProducerKindEnvironment
	case diagnosticProducerKindInvocationDefault:
		return ValueProducerKindInvocationDefault
	case diagnosticProducerKindComputedDefault:
		return ValueProducerKindComputedDefault
	case diagnosticProducerKindStaticDefault:
		return ValueProducerKindStaticDefault
	case diagnosticProducerKindInitial:
		return ValueProducerKindInitial
	case diagnosticProducerKindSuppliedFact:
		return ValueProducerKindSuppliedFact
	case diagnosticProducerKindResolvedFact:
		return ValueProducerKindResolvedFact
	case diagnosticProducerKindDerived:
		return ValueProducerKindDerived
	default:
		return ValueProducerKindUnknown
	}
}

func diagnosticResolutionTier(tier analysisDiagnosticResolutionTier) ResolutionTier {
	switch tier {
	case diagnosticResolutionTierExplicit:
		return ResolutionTierExplicit
	case diagnosticResolutionTierShortcutDefault:
		return ResolutionTierShortcutDefault
	case diagnosticResolutionTierEnvironment:
		return ResolutionTierEnvironment
	case diagnosticResolutionTierInvocationDefault:
		return ResolutionTierInvocationDefault
	case diagnosticResolutionTierComputedDefault:
		return ResolutionTierComputedDefault
	case diagnosticResolutionTierStaticDefault:
		return ResolutionTierStaticDefault
	case diagnosticResolutionTierInitial:
		return ResolutionTierInitial
	default:
		return ResolutionTierUnknown
	}
}

func diagnosticDisposition(disposition analysisDiagnosticDisposition) ValueCandidateDisposition {
	switch disposition {
	case diagnosticDispositionEffective:
		return ValueCandidateDispositionEffective
	case diagnosticDispositionNoEffect:
		return ValueCandidateDispositionNoEffect
	case diagnosticDispositionSuppressed:
		return ValueCandidateDispositionSuppressed
	case diagnosticDispositionConflicting:
		return ValueCandidateDispositionConflicting
	case diagnosticDispositionRejected:
		return ValueCandidateDispositionRejected
	case diagnosticDispositionUnresolved:
		return ValueCandidateDispositionUnresolved
	default:
		return ValueCandidateDispositionUnknown
	}
}

func diagnosticDeclarationIdentity(identity analysisDiagnosticDeclarationIdentity) DeclarationIdentity {
	result := DeclarationIdentity{}
	switch identity.Kind {
	case diagnosticDeclarationCommandEntry:
		result.Kind = DeclarationKindCommandEntry
		result.CommandEntry = identity.CommandEntry
	case diagnosticDeclarationActionEntry:
		result.Kind = DeclarationKindActionEntry
		result.ActionEntry = identity.ActionEntry
	case diagnosticDeclarationFlag:
		result.Kind = DeclarationKindFlag
		result.Flag = identity.Flag
	case diagnosticDeclarationFlagSpelling:
		result.Kind = DeclarationKindFlagSpelling
		result.FlagSpelling = identity.FlagSpelling
	case diagnosticDeclarationArgumentForm:
		result.Kind = DeclarationKindArgumentForm
		result.ArgumentForm = identity.ArgumentForm
	case diagnosticDeclarationArgumentTerm:
		result.Kind = DeclarationKindArgumentTerm
		result.ArgumentTerm = identity.ArgumentTerm
	case diagnosticDeclarationEnvironmentInput:
		result.Kind = DeclarationKindEnvironmentInput
		result.EnvironmentInput = identity.EnvironmentInput
	case diagnosticDeclarationValueDomainEntry:
		result.Kind = DeclarationKindValueDomainEntry
		result.ValueDomainEntry = identity.ValueDomainEntry
	}
	return result
}

func inputSourceID(id sourceID) InputSourceID {
	return InputSourceID{
		value: uint32(id),
	}
}

func inputOriginID(id originID) InputOriginID {
	return InputOriginID{
		value: uint32(id),
	}
}

func inputOriginIDs(ids []originID) []InputOriginID {
	if ids == nil {
		return nil
	}
	result := make([]InputOriginID, len(ids))
	for i, id := range ids {
		result[i] = inputOriginID(id)
	}
	return result
}

func inputPosition(position analysisPosition) InputPosition {
	return InputPosition(position)
}

func inputSpan(span span) InputSpan {
	inputSpan := InputSpan{
		Source: inputSourceID(span.Source),
		Entry:  span.Entry,
		Start:  inputPosition(span.Start),
		End:    inputPosition(span.End),
	}

	return inputSpan
}

func inputRepair(repair repair) InputRepair {
	result := InputRepair{
		Title:            repair.Title,
		PreservesMeaning: repair.PreservesMeaning,
		Sensitive:        repair.Sensitive,
		Sources:          make([]InputRepairSource, len(repair.Sources)),
	}
	for i, source := range repair.Sources {
		result.Sources[i] = InputRepairSource{
			Source:     inputSourceID(source.Source),
			TextEdits:  make([]InputTextEdit, len(source.TextEdits)),
			EntryEdits: make([]InputEntryEdit, len(source.EntryEdits)),
		}
		for j, edit := range source.TextEdits {
			result.Sources[i].TextEdits[j] = InputTextEdit{
				Span:        inputSpan(edit.Span),
				Replacement: inputReplacement(edit.Replacement),
			}
		}
		for j, edit := range source.EntryEdits {
			entries := make([]InputReplacement, len(edit.Entries))
			for k, replacement := range edit.Entries {
				entries[k] = inputReplacement(replacement)
			}
			result.Sources[i].EntryEdits[j] = InputEntryEdit{
				Start:   edit.Start,
				End:     edit.End,
				Entries: entries,
			}
		}
	}
	return result
}

func inputReplacement(replacement replacement) InputReplacement {
	result := make(InputReplacement, len(replacement))
	for i, part := range replacement {
		if part.Literal != nil {
			result[i].Literal = *part.Literal
		}
		if part.Copy != nil {
			span := inputSpan(*part.Copy)
			result[i].Copy = &span
		}
	}
	return result
}

func finalizeOperationFrameworkPresentation(operation *invocation, requested Locale, model *effectiveModel) {
	if operation == nil || !model.Valid() {
		return
	}
	if _, finalized := operation.FrameworkPresentation().Locale(); finalized {
		return
	}
	locale, err := selectEffectiveLocale(
		string(requested),
		"",
		model.Application().localization.SourceLocale(),
	)
	if err != nil {
		return
	}
	operation.FinalizeFrameworkPresentationLocale(locale)
}

func grammarDiagnostics(operation *invocation, grammar grammarResult, tokens []TokenInspection) []Diagnostic {
	if operation == nil || operation.analysis == nil {
		panic("cli: grammar Diagnostic projection requires an invocation analysis")
	}
	result := make([]Diagnostic, len(grammar.Issues))
	model := operation.analysis.effectiveModel
	for i, issue := range grammar.Issues {
		code := grammarDiagnosticCode(issue.Kind)
		diagnostic := Diagnostic{
			Code:        code,
			CommandPath: commandPathNames(model, issue.Command),
			Command:     issue.Command,
			Declaration: grammarIssueDeclaration(issue),
			Suggestions: grammarIssueSuggestions(model, issue),
		}
		if issue.Kind == grammarIssueRejectedArgumentForm {
			diagnostic.ProducerKind = ValueProducerKindArgument
			diagnostic.Tier = ResolutionTierExplicit
			diagnostic.Disposition = ValueCandidateDispositionRejected
		}
		if issue.TokenIndex >= 0 {
			diagnostic.Origins, diagnostic.Span = grammarIssueInputContext(operation, issue.TokenIndex)
			markGrammarIssueTokenRejection(tokens, grammar.Tokens, issue.TokenIndex, code)
		}
		result[i] = diagnostic
	}
	return result
}

func grammarIssueInputContext(operation *invocation, tokenIndex int) ([]InputOriginID, *InputSpan) {
	origin, ok := operation.ArgumentOrigin(tokenIndex)
	if !ok {
		return nil, nil
	}
	origins := []InputOriginID{origin.ID()}
	record, ok := operation.originRecord(origin.ID())
	if !ok {
		return origins, nil
	}
	span := &InputSpan{
		Source: record.span.Source,
		Entry:  record.span.Entry,
		Start:  InputPosition{Offset: record.span.Start},
		End:    InputPosition{Offset: record.span.End},
	}
	return origins, span
}

func markGrammarIssueTokenRejection(tokens []TokenInspection, observed []tokenObservation, tokenIndex int, code DiagnosticCode) {
	for index, token := range observed {
		if token.PhysicalIndex != tokenIndex || index >= len(tokens) {
			continue
		}
		if tokens[index].Role == TokenRoleUnknown {
			tokens[index].RejectionCode = code
		}
		return
	}
}

func grammarIssueSuggestions(model *effectiveModel, issue grammarIssue) []string {
	grammar, ok := model.EffectiveGrammar()
	if !ok {
		return nil
	}
	var suggestions []string
	appendSuggestion := func(value string) {
		if value == "" || slices.Contains(suggestions, value) {
			return
		}
		suggestions = append(suggestions, value)
	}
	for _, id := range issue.Entries {
		for _, entry := range grammar.CommandEntries(issue.Command) {
			if entry.id == id {
				appendSuggestion(entry.name)
				break
			}
		}
	}
	for _, id := range issue.Actions {
		for _, action := range grammar.ActionCommands(issue.Command) {
			if action.id == id {
				appendSuggestion(action.name)
				break
			}
		}
		for _, action := range grammar.ActionFlags(issue.Command) {
			if action.id == id {
				appendSuggestion(action.text)
				break
			}
		}
	}
	for _, id := range issue.Flags {
		for _, flag := range grammar.Flags(issue.Command) {
			for _, spelling := range flag.Spellings() {
				if spelling.id == id {
					appendSuggestion(spelling.text)
					break
				}
			}
		}
	}
	return suggestions
}

func grammarIssueDeclaration(issue grammarIssue) DeclarationIdentity {
	switch issue.Kind {
	case grammarIssueNotIntroducedFlag, grammarIssueRemovedFlag:
		if len(issue.Flags) == 1 && len(issue.Actions) == 0 {
			return DeclarationIdentity{
				Kind:         DeclarationKindFlagSpelling,
				FlagSpelling: issue.Flags[0],
			}
		}
		if len(issue.Actions) == 1 && len(issue.Flags) == 0 {
			return DeclarationIdentity{
				Kind:        DeclarationKindActionEntry,
				ActionEntry: issue.Actions[0],
			}
		}
	case grammarIssueRejectedArgumentForm:
		if issue.ArgumentTerm.present() {
			return DeclarationIdentity{
				Kind:         DeclarationKindArgumentTerm,
				ArgumentTerm: issue.ArgumentTerm,
			}
		}
	default:
		return DeclarationIdentity{}
	}
	return DeclarationIdentity{}
}

func privateGrammarIssueDeclaration(issue grammarIssue) analysisDiagnosticDeclarationIdentity {
	switch issue.Kind {
	case grammarIssueNotIntroducedFlag, grammarIssueRemovedFlag:
		if len(issue.Flags) == 1 && len(issue.Actions) == 0 {
			return analysisDiagnosticDeclarationIdentity{
				Kind:         diagnosticDeclarationFlagSpelling,
				FlagSpelling: issue.Flags[0],
			}
		}
		if len(issue.Actions) == 1 && len(issue.Flags) == 0 {
			return analysisDiagnosticDeclarationIdentity{
				Kind:        diagnosticDeclarationActionEntry,
				ActionEntry: issue.Actions[0],
			}
		}
	case grammarIssueRejectedArgumentForm:
		if issue.ArgumentTerm.present() {
			return analysisDiagnosticDeclarationIdentity{
				Kind:         diagnosticDeclarationArgumentTerm,
				ArgumentTerm: issue.ArgumentTerm,
			}
		}
	default:
		return analysisDiagnosticDeclarationIdentity{}
	}
	return analysisDiagnosticDeclarationIdentity{}
}

func grammarDiagnosticCode(kind grammarIssueKind) DiagnosticCode {
	switch kind {
	case grammarIssueAmbiguousName:
		return DiagnosticAmbiguousCommand
	case grammarIssueBlockedExplicitEntry, grammarIssueUnavailableCommand:
		return DiagnosticUnavailableCommand
	case grammarIssueUnavailableFlag:
		return DiagnosticUnavailableFlag
	case grammarIssueUnavailableArgument:
		return DiagnosticUnavailableArgument
	case grammarIssueNotIntroducedCommand:
		return DiagnosticNotIntroducedCommand
	case grammarIssueNotIntroducedFlag:
		return DiagnosticNotIntroducedFlag
	case grammarIssueNotIntroducedArgument:
		return DiagnosticNotIntroducedArgument
	case grammarIssueRemovedCommand:
		return DiagnosticRemovedCommand
	case grammarIssueRemovedFlag:
		return DiagnosticRemovedFlag
	case grammarIssueRemovedArgument:
		return DiagnosticRemovedArgument
	case grammarIssueConditionEvaluation:
		return DiagnosticInvalidValue
	case grammarIssueAmbiguousDefault:
		return DiagnosticAmbiguousCommandDefault
	case grammarIssueBlockedDefault:
		return DiagnosticBlockedCommandDefault
	case grammarIssueAmbiguousFlag:
		return DiagnosticAmbiguousFlag
	case grammarIssueMisplacedFlag:
		return DiagnosticMisplacedFlag
	case grammarIssueUnknownFlag:
		return DiagnosticUnknownFlag
	case grammarIssueMissingFlagValue, grammarIssuePrematureFlagTerminator:
		return DiagnosticMissingValue
	case grammarIssueTooManyFlagOccurrences, grammarIssueMissingFlagOccurrence:
		return DiagnosticRepeatedFlag
	case grammarIssueAmbiguousOwnership,
		grammarIssueAmbiguousArgumentForm,
		grammarIssueAmbiguousFormAllocation:
		return DiagnosticAmbiguousArgumentForm
	case grammarIssueRejectedArgumentForm:
		return DiagnosticInvalidValue
	case grammarIssueArgumentFormFailure:
		return DiagnosticInputParserFailure
	case grammarIssueNoArgumentForm, grammarIssueUnexpectedInput:
		return DiagnosticUnexpectedArgument
	default:
		return DiagnosticInvalidArgumentSyntax
	}
}

func grammarDiagnosticFailure(operation *invocation, grammar grammarResult, diagnostics []Diagnostic, truncated bool) error {
	payload := operationDiagnosticPayload(operation, truncated)
	payload.Diagnostics = make([]diagnostic, len(diagnostics))
	for i, item := range diagnostics {
		origins := make([]originID, len(item.Origins))
		for j, sourceOrigin := range item.Origins {
			origins[j] = originID(sourceOrigin.value)
		}
		projected := diagnostic{
			Code:        string(item.Code),
			CommandPath: append([]string(nil), item.CommandPath...),
			Origins:     origins,
			Suggestions: append([]string(nil), item.Suggestions...),
		}
		if item.Span != nil {
			projected.Span = &span{
				Source: sourceID(item.Span.Source.value),
				Entry:  item.Span.Entry,
				Start: analysisPosition{
					Offset: item.Span.Start.Offset,
					Line:   item.Span.Start.Line,
					Column: item.Span.Start.Column,
				},
				End: analysisPosition{
					Offset: item.Span.End.Offset,
					Line:   item.Span.End.Line,
					Column: item.Span.End.Column,
				},
			}
		}
		if i < len(grammar.Issues) {
			projected.Command = grammar.Issues[i].Command
			projected.Declaration = privateGrammarIssueDeclaration(grammar.Issues[i])
			if grammar.Issues[i].Kind == grammarIssueRejectedArgumentForm {
				projected.ProducerKind = projectDiagnosticProducerKind(ValueProducerKindArgument)
				projected.Tier = projectResolutionTier(ResolutionTierExplicit)
				projected.Disposition = projectCandidateDisposition(ValueCandidateDispositionRejected)
			}
		}
		payload.Diagnostics[i] = projected
	}
	var cause error
	for _, issue := range grammar.Issues {
		if issue.Err != nil {
			cause = issue.Err
			break
		}
	}
	return retainDiagnosticFailure(payload, cause)
}

const (
	policyValueKindTokenBool     = "bool"
	policyValueKindTokenCount    = "count"
	policyValueKindTokenInt      = "int"
	policyValueKindTokenUint     = "uint"
	policyValueKindTokenString   = "string"
	policyValueKindTokenSequence = "sequence"
	policyValueKindTokenKeyed    = "keyed"
	policyValueKindTokenCustom   = "custom"
)

func policyRuleDiagnostic(model *effectiveModel, id RuleID, definition ruleDefinition, violation violation) (Diagnostic, error) {
	diagnostic, err := policyRuleFailureDiagnostic(model, id, DiagnosticRuleViolation, definition)
	if err != nil {
		return Diagnostic{}, err
	}
	diagnostic.Rule.ViolationKind = policyViolationKind(violation.Kind)
	diagnostic.Origins = make([]InputOriginID, len(violation.Origins))
	copy(diagnostic.Origins, violation.Origins)
	return diagnostic, nil
}

func policyRuleFailureDiagnostic(model *effectiveModel, id RuleID, code DiagnosticCode, definition ruleDefinition) (Diagnostic, error) {
	rule, err := projectEffectiveRuleFailure(model, id, definition)
	if err != nil {
		return Diagnostic{}, err
	}
	return Diagnostic{
		Code: code,
		Rule: &rule,
	}, nil
}

func policyViolationKind(kind ruleViolationKind) ViolationKind {
	switch kind {
	case ruleViolationInvalid:
		return ViolationInvalid
	case ruleViolationConflict:
		return ViolationConflict
	case ruleViolationDuplicate:
		return ViolationDuplicate
	default:
		return ViolationUnknown
	}
}

func policyDiagnosticFailure(operation *invocation, model *effectiveModel, diagnostics []Diagnostic, truncated bool, cause error) error {
	payload := operationDiagnosticPayload(operation, truncated)
	payload.Diagnostics = make([]diagnostic, len(diagnostics))
	for i, item := range diagnostics {
		origins := make([]originID, len(item.Origins))
		for j, sourceOrigin := range item.Origins {
			origins[j] = originID(sourceOrigin.value)
		}
		payload.Diagnostics[i] = diagnostic{
			Code:         string(item.Code),
			Rule:         diagnosticProjectRuleFailure(item.Rule),
			AdvisoryCode: string(item.AdvisoryCode),
			Command:      privatePolicyCommandID(model, item.Command),
			Origins:      origins,
		}
	}
	return retainDiagnosticFailure(payload, cause)
}

func diagnosticProjectRuleFailure(rule *RuleFailure) *analysisDiagnosticRuleFailure {
	if rule == nil {
		return nil
	}
	result := &analysisDiagnosticRuleFailure{
		ID:            uint32(rule.RuleID),
		Kind:          diagnosticProjectRuleKind(rule.RuleKind),
		Code:          string(rule.RuleCode),
		ViolationKind: diagnosticProjectViolationKind(rule.ViolationKind),
		Subjects:      make([]uint32, len(rule.Subjects)),
		Operands:      make([]diagnosticRuleOperand, len(rule.Operands)),
		Pattern:       rule.Pattern,
	}
	for i, subject := range rule.Subjects {
		result.Subjects[i] = uint32(subject)
	}
	for i, operand := range rule.Operands {
		result.Operands[i].Sensitive = operand.Sensitive
		if operand.Literal != nil {
			result.Operands[i].Literal = &diagnosticValueLiteral{
				Type:      diagnosticProjectValueType(operand.Literal.Type),
				Text:      operand.Literal.Text,
				Sensitive: operand.Literal.Sensitive,
				Redacted:  operand.Literal.Redacted,
			}
		}
	}
	if rule.Count.valid {
		result.CountMinimum = rule.Count.minimum
		result.CountMaximum = rule.Count.maximum
		result.CountUnbounded = rule.Count.unbounded
		result.CountValid = true
	}
	return result
}

func diagnosticProjectValueType(value ValueType) analysisDiagnosticValueType {
	var kind string
	switch value.Kind {
	case ValueKindBool:
		kind = policyValueKindTokenBool
	case ValueKindCount:
		kind = policyValueKindTokenCount
	case ValueKindInt:
		kind = policyValueKindTokenInt
	case ValueKindUint:
		kind = policyValueKindTokenUint
	case ValueKindString:
		kind = policyValueKindTokenString
	case ValueKindSequence:
		kind = policyValueKindTokenSequence
	case ValueKindKeyed:
		kind = policyValueKindTokenKeyed
	case ValueKindCustom:
		kind = policyValueKindTokenCustom
	case ValueKindUnknown:
	}
	return analysisDiagnosticValueType{
		Kind: kind,
		Name: value.Name,
	}
}

func privatePolicyCommandID(model *effectiveModel, id CommandID) CommandID {
	for _, record := range model.Commands() {
		if record.meta.id == id {
			return record.meta.id
		}
	}
	return CommandID(0)
}

func diagnosticProjectRuleKind(kind RuleKind) analysisDiagnosticRuleKind {
	switch kind {
	case RuleCustom:
		return diagnosticRuleCustom
	case RuleRequire:
		return diagnosticRuleRequire
	case RuleForbid:
		return diagnosticRuleForbid
	case RuleNotEmpty:
		return diagnosticRuleNotEmpty
	case RuleIn:
		return diagnosticRuleIn
	case RuleNotIn:
		return diagnosticRuleNotIn
	case RuleEqual:
		return diagnosticRuleEqual
	case RuleNotEqual:
		return diagnosticRuleNotEqual
	case RuleBetween:
		return diagnosticRuleBetween
	case RuleAtLeast:
		return diagnosticRuleAtLeast
	case RuleAtMost:
		return diagnosticRuleAtMost
	case RuleGreaterThan:
		return diagnosticRuleGreaterThan
	case RuleLessThan:
		return diagnosticRuleLessThan
	case RuleMatch:
		return diagnosticRuleMatch
	case RuleItemCount:
		return diagnosticRuleItemCount
	case RuleContains:
		return diagnosticRuleContains
	case RuleEachIn:
		return diagnosticRuleEachIn
	case RuleUnique:
		return diagnosticRuleUnique
	case RuleEntryCount:
		return diagnosticRuleEntryCount
	case RuleHasKeys:
		return diagnosticRuleHasKeys
	case RuleKeysIn:
		return diagnosticRuleKeysIn
	case RuleSubsetOf:
		return diagnosticRuleSubsetOf
	case RuleDisjoint:
		return diagnosticRuleDisjoint
	case RuleKeysSubsetOf:
		return diagnosticRuleKeysSubsetOf
	case RuleKeysDisjoint:
		return diagnosticRuleKeysDisjoint
	case RuleExactlyOne:
		return diagnosticRuleExactlyOne
	case RuleAllOrNone:
		return diagnosticRuleAllOrNone
	case RuleAtMostOne:
		return diagnosticRuleAtMostOne
	case RuleAtLeastOne:
		return diagnosticRuleAtLeastOne
	default:
		return ""
	}
}

func diagnosticProjectViolationKind(kind ViolationKind) analysisDiagnosticViolationKind {
	switch kind {
	case ViolationInvalid:
		return diagnosticViolationInvalid
	case ViolationConflict:
		return diagnosticViolationConflict
	case ViolationDuplicate:
		return diagnosticViolationDuplicate
	default:
		return ""
	}
}
