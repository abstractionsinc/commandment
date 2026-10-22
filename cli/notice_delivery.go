package cli

import "context"

// runNoticeDeliveryAttempt is one already-admitted Notice delivery operation.
// Notice construction and suppression remain upstream; Index is the canonical
// delivery index supplied by the owning invocation pipeline.
type runNoticeDeliveryAttempt[T any] struct {
	Index  int
	Notice T
}

// noticeDeliveryFailure preserves one failed fallible Notice attempt without
// preventing later delivery attempts.
type noticeDeliveryFailure[T any] struct {
	Index  int
	Notice T
	Err    error
}

// runDeliverNotices performs already-admitted Notice delivery synchronously in
// caller order. A nil handler means that no delivery attempt exists. Failed
// fallible attempts are preserved in deterministic attempt order and do not
// stop later attempts.
func runDeliverNotices[T any](ctx context.Context, attempts []runNoticeDeliveryAttempt[T], handler func(context.Context, T) error) []noticeDeliveryFailure[T] {
	if handler == nil || len(attempts) == 0 {
		return nil
	}

	var failures []noticeDeliveryFailure[T]
	for _, attempt := range attempts {
		if err := handler(ctx, attempt.Notice); err != nil {
			composite := noticeDeliveryFailure[T]{
				Index:  attempt.Index,
				Notice: attempt.Notice,
				Err:    err,
			}
			failures = append(failures, composite)
		}
	}
	return failures
}

// noticeDeliveryAttempt is an already-unsuppressed Notice delivery operation.
// Its Index and Phase are finalized by the owning invocation pipeline before
// application execution or cleanup delivery begins.
type noticeDeliveryAttempt struct {
	Index  int
	Phase  NoticePhase
	Notice Notice
}

type noticeDeliveryPayload struct {
	phase  NoticePhase
	notice Notice
}

// deliverNotices adapts the frozen public NoticeHandler onto the private
// synchronous delivery engine. Suppression is intentionally absent here:
// callers supply only already-admitted delivery attempts.
func deliverNotices(ctx context.Context, operation *invocation, handler NoticeHandler, attempts []noticeDeliveryAttempt) []error {
	if handler == nil || len(attempts) == 0 {
		return nil
	}

	prepared := make([]runNoticeDeliveryAttempt[noticeDeliveryPayload], len(attempts))
	for i, attempt := range attempts {
		if attempt.Index < 0 || !validNoticePhase(attempt.Phase) || attempt.Notice.Code == "" {
			panic("cli: invalid accepted Notice delivery attempt")
		}
		prepared[i] = runNoticeDeliveryAttempt[noticeDeliveryPayload]{
			Index: attempt.Index,
			Notice: noticeDeliveryPayload{
				phase:  attempt.Phase,
				notice: cloneNotice(attempt.Notice),
			},
		}
	}

	failures := runDeliverNotices(ctx, prepared, func(callCtx context.Context, state noticeDeliveryPayload) error {
		return handler(callCtx, cloneNotice(state.notice))
	})
	if len(failures) == 0 {
		return nil
	}

	result := make([]error, len(failures))
	for i, failure := range failures {
		result[i] = wrapNoticeDeliveryFailure(
			operation,
			failure.Index,
			failure.Notice.phase,
			failure.Notice.notice,
			failure.Err,
		)
	}
	return result
}

func wrapNoticeDeliveryFailure(operation *invocation, index int, phase NoticePhase, notice Notice, cause error) error {
	wrapped := &integrationFailureError{
		message: noticeDeliveryErrorText(index),
		err:     cause,
		policy:  integrationExitPolicyGenericFailure,
	}

	detail := diagnosticNoticeDeliveryFailure{
		Index:       index,
		Phase:       diagnosticNoticePhase(phase),
		Code:        string(notice.Code),
		CommandPath: append([]string(nil), notice.CommandPath...),
	}
	failurePayload := payload{
		Diagnostics: []diagnostic{{
			Code:           string(DiagnosticNoticeDeliveryFailure),
			CommandPath:    append([]string(nil), notice.CommandPath...),
			NoticeDelivery: &detail,
		}},
		Presentation: operation.FrameworkPresentation(),
	}
	return retainDiagnosticFailure(failurePayload, wrapped)
}

func diagnosticNoticePhase(phase NoticePhase) analysisDiagnosticNoticePhase {
	switch phase {
	case NoticePhaseAnalysis:
		return diagnosticNoticePhaseAnalysis
	case NoticePhaseExecution:
		return diagnosticNoticePhaseExecution
	case NoticePhaseCleanup:
		return diagnosticNoticePhaseCleanup
	default:
		return ""
	}
}

func validNoticePhase(phase NoticePhase) bool {
	switch phase {
	case NoticePhaseAnalysis, NoticePhaseExecution, NoticePhaseCleanup:
		return true
	default:
		return false
	}
}

func cloneNotice(notice Notice) Notice {
	cloned := notice
	cloned.CommandPath = append([]string(nil), notice.CommandPath...)
	if notice.Replacement != nil {
		replacement := *notice.Replacement
		cloned.Replacement = &replacement
	}
	return cloned
}

func buildAdvisoryNoticeDeliveryAttempts(operation *invocation, model *effectiveModel, locale Locale, analysis policyAnalysis) ([]noticeDeliveryAttempt, error) {
	if len(analysis.Notices) == 0 {
		return nil, nil
	}
	attempts := make([]noticeDeliveryAttempt, 0, len(analysis.Notices))
	for index, notice := range analysis.Notices {
		if len(notice.SuppressedBy) != 0 {
			continue
		}
		record, ok := model.Advisory(notice.AdvisoryID)
		if !ok {
			return nil, errInvalidModelTopology
		}
		command := record.command
		path := commandPathNames(model, command)
		noticeDeliveryAttemptValue := noticeDeliveryAttempt{
			Index:  index,
			Phase:  NoticePhaseAnalysis,
			Notice: projectAdvisoryNotice(operation, locale, record.command, notice, command, path),
		}
		attempts = append(attempts, noticeDeliveryAttemptValue)
	}
	return attempts, nil
}

func projectAdvisoryNotice(operation *invocation, locale Locale, owner CommandID, notice advisoryNotice, command CommandID, path []string) Notice {
	severity, _ := projectNoticeSeverity(notice.Severity)
	localization := operation.LocalizationForCommand(owner)
	result := Notice{
		Code:         NoticeAdvisory,
		AdvisoryCode: AdvisoryCode(notice.AdvisoryCode),
		Severity:     severity,
		Message:      referenceText(localization, string(locale), notice.Message),
		Reason:       referenceText(localization, string(locale), notice.Reason),
		Hint:         referenceText(localization, string(locale), notice.Hint),
		Command:      command,
		CommandPath:  append([]string(nil), path...),
	}

	return result
}
