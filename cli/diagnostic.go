package cli

// DiagnosticCode is the stable machine classification of a Commandment diagnostic. It
// identifies failure class, not the exact semantic subject; structured fields on Diagnostic
// carry declaration, Rule, source, provider, signal, and other identity.
type DiagnosticCode string

const (
	// DiagnosticInvalidDeclaration identifies the stable Commandment diagnostic class
	// `invalid-declaration`.
	DiagnosticInvalidDeclaration DiagnosticCode = "invalid-declaration"

	// DiagnosticBuilderMisuse identifies the stable Commandment diagnostic class `builder-misuse`.
	DiagnosticBuilderMisuse DiagnosticCode = "builder-misuse"
	// DiagnosticCompletionProtocolMismatch identifies the stable Commandment diagnostic class
	// `completion-protocol-mismatch`.
	DiagnosticCompletionProtocolMismatch DiagnosticCode = "completion-protocol-mismatch"

	// DiagnosticUnknownCommand identifies the stable Commandment diagnostic class
	// `unknown-command`.
	DiagnosticUnknownCommand DiagnosticCode = "unknown-command"
	// DiagnosticAmbiguousCommand identifies the stable Commandment diagnostic class
	// `ambiguous-command`.
	DiagnosticAmbiguousCommand DiagnosticCode = "ambiguous-command"

	// DiagnosticUnknownFlag identifies the stable Commandment diagnostic class `unknown-flag`.
	DiagnosticUnknownFlag DiagnosticCode = "unknown-flag"
	// DiagnosticAmbiguousFlag identifies the stable Commandment diagnostic class `ambiguous-flag`.
	DiagnosticAmbiguousFlag DiagnosticCode = "ambiguous-flag"
	// DiagnosticMisplacedFlag identifies the stable Commandment diagnostic class `misplaced-flag`.
	DiagnosticMisplacedFlag DiagnosticCode = "misplaced-flag"
	// DiagnosticRepeatedFlag identifies the stable Commandment diagnostic class `repeated-flag`.
	DiagnosticRepeatedFlag DiagnosticCode = "repeated-flag"
	// DiagnosticMissingValue identifies the stable Commandment diagnostic class `missing-value`.
	DiagnosticMissingValue DiagnosticCode = "missing-value"
	// DiagnosticInvalidValue identifies the stable Commandment diagnostic class `invalid-value`.
	DiagnosticInvalidValue DiagnosticCode = "invalid-value"

	// DiagnosticUnexpectedArgument identifies the stable Commandment diagnostic class
	// `unexpected-argument`.
	DiagnosticUnexpectedArgument DiagnosticCode = "unexpected-argument"
	// DiagnosticAmbiguousArgumentForm identifies the stable Commandment diagnostic class
	// `ambiguous-argument-form`.
	DiagnosticAmbiguousArgumentForm DiagnosticCode = "ambiguous-argument-form"
	// DiagnosticInvalidArgumentSyntax identifies the stable Commandment diagnostic class
	// `invalid-argument-syntax`.
	DiagnosticInvalidArgumentSyntax DiagnosticCode = "invalid-argument-syntax"

	// DiagnosticResponseFileAccessUnavailable identifies the stable Commandment diagnostic class
	// `response-file-access-unavailable`.
	DiagnosticResponseFileAccessUnavailable DiagnosticCode = "response-file-access-unavailable"
	// DiagnosticResponseFileLoadFailure identifies the stable Commandment diagnostic class
	// `response-file-load-failure`.
	DiagnosticResponseFileLoadFailure DiagnosticCode = "response-file-load-failure"
	// DiagnosticInvalidResponseFile identifies the stable Commandment diagnostic class
	// `invalid-response-file`.
	DiagnosticInvalidResponseFile DiagnosticCode = "invalid-response-file"
	// DiagnosticInvalidResponseFileSource identifies the stable Commandment diagnostic class
	// `invalid-response-file-source`.
	DiagnosticInvalidResponseFileSource DiagnosticCode = "invalid-response-file-source"
	// DiagnosticResponseFileCycle identifies the stable Commandment diagnostic class
	// `response-file-cycle`.
	DiagnosticResponseFileCycle DiagnosticCode = "response-file-cycle"
	// DiagnosticResponseFileLimitExceeded identifies the stable Commandment diagnostic class
	// `response-file-limit-exceeded`.
	DiagnosticResponseFileLimitExceeded DiagnosticCode = "response-file-limit-exceeded"
	// DiagnosticResponseFileCancellation identifies the stable Commandment diagnostic class
	// `response-file-cancellation`.
	DiagnosticResponseFileCancellation DiagnosticCode = "response-file-cancellation"

	// DiagnosticCommandProviderFailure identifies the stable Commandment diagnostic class
	// `command-provider-failure`.
	DiagnosticCommandProviderFailure DiagnosticCode = "command-provider-failure"
	// DiagnosticInvalidCommandBatch identifies the stable Commandment diagnostic class
	// `invalid-command-batch`.
	DiagnosticInvalidCommandBatch DiagnosticCode = "invalid-command-batch"
	// DiagnosticCommandTargetBlocked identifies the stable Commandment diagnostic class
	// `command-target-blocked`.
	DiagnosticCommandTargetBlocked DiagnosticCode = "command-target-blocked"
	// DiagnosticCommandProviderCancellation identifies the stable Commandment diagnostic class
	// `command-provider-cancellation`.
	DiagnosticCommandProviderCancellation DiagnosticCode = "command-provider-cancellation"

	// DiagnosticValueDomainProvisionFailure identifies the stable Commandment diagnostic class
	// `value-domain-provision-failure`.
	DiagnosticValueDomainProvisionFailure DiagnosticCode = "value-domain-provision-failure"
	// DiagnosticInvalidValueDomain identifies the stable Commandment diagnostic class
	// `invalid-value-domain`.
	DiagnosticInvalidValueDomain DiagnosticCode = "invalid-value-domain"
	// DiagnosticValueDomainProvisionCancellation identifies the stable Commandment diagnostic
	// class `value-domain-provision-cancellation`.
	DiagnosticValueDomainProvisionCancellation DiagnosticCode = "value-domain-provision-cancellation"
	// DiagnosticAmbiguousCommandDefault identifies the stable Commandment diagnostic class
	// `ambiguous-command-default`.
	DiagnosticAmbiguousCommandDefault DiagnosticCode = "ambiguous-command-default"
	// DiagnosticBlockedCommandDefault identifies the stable Commandment diagnostic class
	// `blocked-command-default`.
	DiagnosticBlockedCommandDefault DiagnosticCode = "blocked-command-default"
	// DiagnosticActionConflict identifies the stable Commandment diagnostic class
	// `action-conflict`.
	DiagnosticActionConflict DiagnosticCode = "action-conflict"
	// DiagnosticNotIntroducedValue identifies the stable Commandment diagnostic class
	// `not-introduced-value`.
	DiagnosticNotIntroducedValue DiagnosticCode = "not-introduced-value"
	// DiagnosticRemovedValue identifies the stable Commandment diagnostic class `removed-value`.
	DiagnosticRemovedValue DiagnosticCode = "removed-value"

	// DiagnosticMissingFact identifies the stable Commandment diagnostic class `missing-fact`.
	DiagnosticMissingFact DiagnosticCode = "missing-fact"
	// DiagnosticFactSupplyFailure identifies the stable Commandment diagnostic class
	// `fact-supply-failure`.
	DiagnosticFactSupplyFailure DiagnosticCode = "fact-supply-failure"
	// DiagnosticFactResolutionFailure identifies the stable Commandment diagnostic class
	// `fact-resolution-failure`.
	DiagnosticFactResolutionFailure DiagnosticCode = "fact-resolution-failure"

	// DiagnosticDefaultSupplyFailure identifies the stable Commandment diagnostic class
	// `default-supply-failure`.
	DiagnosticDefaultSupplyFailure DiagnosticCode = "default-supply-failure"
	// DiagnosticValueComputationFailure identifies the stable Commandment diagnostic class
	// `value-computation-failure`.
	DiagnosticValueComputationFailure DiagnosticCode = "value-computation-failure"
	// DiagnosticInputParserFailure identifies the stable Commandment diagnostic class
	// `input-parser-failure`.
	DiagnosticInputParserFailure DiagnosticCode = "input-parser-failure"

	// DiagnosticRuleViolation identifies the stable Commandment diagnostic class `rule-violation`.
	DiagnosticRuleViolation DiagnosticCode = "rule-violation"
	// DiagnosticRuleEvaluationFailure identifies the stable Commandment diagnostic class
	// `rule-evaluation-failure`.
	DiagnosticRuleEvaluationFailure DiagnosticCode = "rule-evaluation-failure"
	// DiagnosticInvalidRuleResult identifies the stable Commandment diagnostic class
	// `invalid-rule-result`.
	DiagnosticInvalidRuleResult DiagnosticCode = "invalid-rule-result"
	// DiagnosticAdvisoryEvaluationFailure identifies the stable Commandment diagnostic class
	// `advisory-evaluation-failure`.
	DiagnosticAdvisoryEvaluationFailure DiagnosticCode = "advisory-evaluation-failure"

	// DiagnosticNotIntroducedCommand identifies the stable Commandment diagnostic class
	// `not-introduced-command`.
	DiagnosticNotIntroducedCommand DiagnosticCode = "not-introduced-command"
	// DiagnosticNotIntroducedFlag identifies the stable Commandment diagnostic class
	// `not-introduced-flag`.
	DiagnosticNotIntroducedFlag DiagnosticCode = "not-introduced-flag"
	// DiagnosticNotIntroducedArgument identifies the stable Commandment diagnostic class
	// `not-introduced-argument`.
	DiagnosticNotIntroducedArgument DiagnosticCode = "not-introduced-argument"

	// DiagnosticRemovedCommand identifies the stable Commandment diagnostic class
	// `removed-command`.
	DiagnosticRemovedCommand DiagnosticCode = "removed-command"
	// DiagnosticRemovedFlag identifies the stable Commandment diagnostic class `removed-flag`.
	DiagnosticRemovedFlag DiagnosticCode = "removed-flag"
	// DiagnosticRemovedArgument identifies the stable Commandment diagnostic class
	// `removed-argument`.
	DiagnosticRemovedArgument DiagnosticCode = "removed-argument"

	// DiagnosticUnavailableCommand identifies the stable Commandment diagnostic class
	// `unavailable-command`.
	DiagnosticUnavailableCommand DiagnosticCode = "unavailable-command"
	// DiagnosticUnavailableFlag identifies the stable Commandment diagnostic class
	// `unavailable-flag`.
	DiagnosticUnavailableFlag DiagnosticCode = "unavailable-flag"
	// DiagnosticUnavailableArgument identifies the stable Commandment diagnostic class
	// `unavailable-argument`.
	DiagnosticUnavailableArgument DiagnosticCode = "unavailable-argument"

	// DiagnosticCompletionFailure identifies the stable Commandment diagnostic class
	// `completion-failure`.
	DiagnosticCompletionFailure DiagnosticCode = "completion-failure"
	// DiagnosticInvalidCompletionRequest identifies the stable Commandment diagnostic class
	// `invalid-completion-request`.
	DiagnosticInvalidCompletionRequest DiagnosticCode = "invalid-completion-request"
	// DiagnosticAmbiguousCompletionTarget identifies the stable Commandment diagnostic class
	// `ambiguous-completion-target`.
	DiagnosticAmbiguousCompletionTarget DiagnosticCode = "ambiguous-completion-target"
	// DiagnosticInvalidCompletionContribution identifies the stable Commandment diagnostic class
	// `invalid-completion-contribution`.
	DiagnosticInvalidCompletionContribution DiagnosticCode = "invalid-completion-contribution"
	// DiagnosticUnsupportedCompletionDelegation identifies the stable Commandment diagnostic class
	// `unsupported-completion-delegation`.
	DiagnosticUnsupportedCompletionDelegation DiagnosticCode = "unsupported-completion-delegation"
	// DiagnosticHelpRenderingFailure identifies the stable Commandment diagnostic class
	// `help-rendering-failure`.
	DiagnosticHelpRenderingFailure DiagnosticCode = "help-rendering-failure"
	// DiagnosticCompletionScriptGenerationFailure identifies the stable Commandment diagnostic
	// class `completion-script-generation-failure`.
	DiagnosticCompletionScriptGenerationFailure DiagnosticCode = "completion-script-generation-failure"
	// DiagnosticOutputWriteFailure identifies the stable Commandment diagnostic class
	// `output-write-failure`.
	DiagnosticOutputWriteFailure DiagnosticCode = "output-write-failure"

	// DiagnosticNotRunnableCommand identifies the stable Commandment diagnostic class
	// `not-runnable-command`.
	DiagnosticNotRunnableCommand DiagnosticCode = "not-runnable-command"

	// DiagnosticMiddlewareProtocolViolation identifies the stable Commandment diagnostic class
	// `middleware-protocol-violation`.
	DiagnosticMiddlewareProtocolViolation DiagnosticCode = "middleware-protocol-violation"

	// DiagnosticNoticeDeliveryFailure identifies the stable Commandment diagnostic class
	// `notice-delivery-failure`.
	DiagnosticNoticeDeliveryFailure DiagnosticCode = "notice-delivery-failure"

	// DiagnosticIncompatibleSignalSource identifies the stable Commandment diagnostic class
	// `incompatible-signal-source`.
	DiagnosticIncompatibleSignalSource DiagnosticCode = "incompatible-signal-source"
	// DiagnosticSignalSourceFailure identifies the stable Commandment diagnostic class
	// `signal-source-failure`.
	DiagnosticSignalSourceFailure DiagnosticCode = "signal-source-failure"
	// DiagnosticSignalActionFailure identifies the stable Commandment diagnostic class
	// `signal-action-failure`.
	DiagnosticSignalActionFailure DiagnosticCode = "signal-action-failure"
	// DiagnosticSignalOverflow identifies the stable Commandment diagnostic class
	// `signal-overflow`.
	DiagnosticSignalOverflow DiagnosticCode = "signal-overflow"
)

// SignalOverflowBoundary identifies the signal pipeline boundary at which bounded
// buffering overflowed. The zero value is unspecified.
type SignalOverflowBoundary uint8

const (
	// SignalOverflowBoundaryUnknown is the zero value and does not identify an overflow boundary.
	SignalOverflowBoundaryUnknown SignalOverflowBoundary = iota
	// SignalOverflowBoundarySource identifies overflow at the signal-source boundary.
	SignalOverflowBoundarySource
	// SignalOverflowBoundaryRouting identifies overflow while routing accepted signal events.
	SignalOverflowBoundaryRouting
	// SignalOverflowBoundaryEndpoint identifies overflow at a bounded signal endpoint.
	SignalOverflowBoundaryEndpoint
)

// SignalOverflowFailure records structured signal overflow state. EndpointID is
// meaningful for endpoint overflow; Capacity and Buffered report the bounded queue
// state observed at failure.
type SignalOverflowFailure struct {
	Boundary   SignalOverflowBoundary
	EndpointID SignalEndpointID
	Capacity   int
	Buffered   int
}

// RuleFailure is the complete Rule-specific structure for a Rule diagnostic. RuleID
// is exact Rule identity; RuleCode is classification only; Subjects and Operands
// retain canonical order. Evaluation failure does not invent ViolationKind, and
// redaction preserves safe Rule structure while suppressing protected literals.
type RuleFailure struct {
	RuleID        RuleID
	RuleKind      RuleKind
	RuleCode      RuleCode
	ViolationKind ViolationKind

	Subjects []ValueID
	Operands []RuleOperand

	Pattern string
	Count   CountRange
}

// InputRejectionDiagnostic is the normalized parser-rejection structure retained by a
// Diagnostic. Code is the parser-local semantic classification, Expected preserves parser
// order, and Spans is the complete globalized parser range when one was supplied.
type InputRejectionDiagnostic struct {
	Code     InputRejectionCode
	Expected []string
	Spans    []InputSpan
}

// Diagnostic is one structured Commandment failure or analysis finding. Code classifies the
// failure while typed identity/provenance fields identify its exact subject. InputRedacted and
// Sensitive distinguish protected presentation from safe structured facts; Message, Reason,
// and Hint are presentation rather than semantic identity.
type Diagnostic struct {
	Code DiagnosticCode

	Rule         *RuleFailure
	Rejection    *InputRejectionDiagnostic
	AdvisoryCode AdvisoryCode

	Message string
	Reason  string
	Hint    string

	Subject     string
	CommandPath []string

	Command     CommandID
	Declaration DeclarationIdentity
	Provider    CommandProviderID
	Default     CommandDefaultID

	ProviderOutput *CommandProviderOutputLocator
	Replacement    *DeclarationReplacement

	InputSource InputSourceID
	Span        *InputSpan
	Origins     []InputOriginID

	Input         string
	InputRedacted bool
	Suggestions   []string
	Usage         []string

	Sensitive bool

	ExitCode     int
	ProducerKind ValueProducerKind
	Tier         ResolutionTier
	Disposition  ValueCandidateDisposition

	RepairIndexes []int

	SignalAction   *SignalActionFailure
	SignalOverflow *SignalOverflowFailure
	NoticeDelivery *NoticeDeliveryFailure
}

// DiagnosticReport is a detached, source-closed collection of Commandment diagnostics.
// Diagnostics are in deterministic order; Sources, Origins, and Repairs contain the complete
// report-local relationships they reference. The zero value is an empty report. Truncated is
// true only when the analysis bound actually omitted eligible diagnostic or recoverable work.
// Human publication is a separate writer-aware concern owned by WriteError. Error() / %v remain
// concise ordinary Go identity; %+v on exact Commandment Diagnostic owners is a bounded
// deterministic developer/debug projection. The caller retains destination and process-policy
// ownership.
type DiagnosticReport struct {
	Diagnostics []Diagnostic

	Sources map[InputSourceID]InputSource
	Origins map[InputOriginID]InputOrigin

	Repairs []InputRepair

	Truncated bool
}

// Diagnostics extracts structured Commandment diagnostic payload from err, including wrapped
// or joined Commandment failures, and rebases their report-local provenance into one coherent
// report. Nil or an ordinary error with no Commandment diagnostics returns the zero/empty
// report.
func Diagnostics(err error) DiagnosticReport {
	return diagnosticReportFromError(err)
}
