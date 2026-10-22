package clitest

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

// DiagnosticRuleOperandLocator identifies one Rule operand carried by a public Diagnostic's
// structured Rule failure. The index is report-local expectation structure, not a new semantic
// identity, and the zero value is invalid.
type DiagnosticRuleOperandLocator struct {
	diagnostic DiagnosticLocator
	index      int
}

// DiagnosticValueLiteralLocator identifies a public literal carried by a Diagnostic Rule
// operand. Text mismatch reporting preserves Commandment sensitivity and redaction boundaries;
// the zero value is invalid.
type DiagnosticValueLiteralLocator struct {
	diagnostic DiagnosticLocator
	operand    int
}

// DiagnosticSignalActionFailureLocator identifies structured signal-action failure detail on a
// public Diagnostic. It exposes only the failure relationships and host facts Commandment
// reports; the zero value is invalid.
type DiagnosticSignalActionFailureLocator struct{ diagnostic DiagnosticLocator }

// DiagnosticSignalActionLocator identifies the action embedded in a structured signal-action
// Diagnostic. It is a expectation view of that public failure detail, not an executable action or
// independent semantic object; the zero value is invalid.
type DiagnosticSignalActionLocator struct{ diagnostic DiagnosticLocator }

// DiagnosticSignalEventLocator identifies the signal event embedded in structured Diagnostic
// failure detail. It checks the event facts Commandment retained, including ordered observation
// and occurrence coordinates, without synthesizing runtime state; the zero value is invalid.
type DiagnosticSignalEventLocator struct{ diagnostic DiagnosticLocator }

// DiagnosticSignalOverflowFailureLocator identifies structured endpoint-overflow failure detail
// on a public Diagnostic. Endpoint presence, capacity, and buffered counts are proved only from
// the reported failure payload; the zero value is invalid.
type DiagnosticSignalOverflowFailureLocator struct{ diagnostic DiagnosticLocator }

// DiagnosticNoticeDeliveryFailureLocator identifies structured Notice-delivery failure detail
// on a public Diagnostic. It preserves the reported delivery index, phase, Notice code, and
// command path without reinterpreting the failed operation; the zero value is invalid.
type DiagnosticNoticeDeliveryFailureLocator struct{ diagnostic DiagnosticLocator }

// DiagnosticSubject is the closed set of semantic subjects that can identify a
// structured diagnostic expectation target.
type DiagnosticSubject interface{ diagnosticSubject() }

// DiagnosticLocator identifies one structured Diagnostic for clitest semantic
// expectation. Its zero value is invalid.
type DiagnosticLocator struct {
	_     [0]func()
	state *diagnosticLocatorState
}

// DiagnosticInputGraphClosed returns a DiagnosticReport expectation that checks every
// Diagnostic, origin, span, repair, edit, and replacement-copy reference closes over the
// report-local Sources, Origins, and Repairs collections.
func DiagnosticInputGraphClosed() DiagnosticReportExpectation {
	return diagnosticInputGraphClosedExpectation{}
}

// RepairsExactly returns a DiagnosticReport expectation that checks the report-local Repairs
// collection has exactly the supplied locators in authoritative report order. The expected
// locator list is captured as reusable expectation configuration, not as repair authority.
func RepairsExactly(expected ...RepairLocator) DiagnosticReportExpectation {
	states := make([]*repairLocatorState, len(expected))
	for index, locator := range expected {
		states[index] = locator.state
	}
	expectation := repairsExactlyExpectation{
		states: states,
	}
	return expectation
}

// DiagnosticsTruncated checks that structured diagnostics are truncated on an
// applicable Completion or Diagnostics observation.
func DiagnosticsTruncated() DiagnosticTruncationExpectation { return diagnosticsTruncatedExpectation{} }

// DiagnosticsExactly checks the complete structured Diagnostic collection in
// authoritative Commandment order. Exactness is not inferred from a truncated
// diagnostic domain.
func DiagnosticsExactly(locators ...DiagnosticLocator) OutcomeExpectation {
	return diagnosticsExactlyExpectation(locators)
}

// Diagnostic returns an expected coordinate for the unique structured Diagnostic with code. If multiple diagnostics
// share the code the coordinate is ambiguous; truncation prevents code-only uniqueness or absence from being assumed.
func Diagnostic(code cli.DiagnosticCode) DiagnosticLocator { return diagnosticByCode(code) }

// DiagnosticAt returns an expected coordinate for the Diagnostic at index in DiagnosticReport.Diagnostics authoritative
// order.
func DiagnosticAt(index int) DiagnosticLocator { return diagnosticByIndex(index) }

// TokenAt returns an expected coordinate for the token at index in Inspection.Tokens authoritative order.
func TokenAt(index int) TokenLocator { return tokenIndexLocator(index) }

// RuleIdentityPresent returns an outcome expectation that checks the Diagnostic carries a
// nonzero structured Rule identity. The Rule's classification code is not substituted for that
// identity.
func (locator DiagnosticLocator) RuleIdentityPresent() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyRuleIdentityPresent,
	}
	return expectation
}

// RuleKind returns a reusable outcome expectation that checks the public Rule kind exposed by
// the located Diagnostic.
func (locator DiagnosticLocator) RuleKind(expected cli.RuleKind) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyRuleKind,
		ruleKind: expected,
	}
	return expectation
}

// RuleOperandCount returns a reusable outcome expectation that checks the Rule operand count
// exposed by the located Diagnostic.
func (locator DiagnosticLocator) RuleOperandCount(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyRuleOperandCount,
		integer:  expected,
	}
	return expectation
}

// RuleOperandAt returns a locator for the rule operand at the supplied public ordered index of
// the located Diagnostic. It does not search for or silently choose a first match.
func (locator DiagnosticLocator) RuleOperandAt(index int) DiagnosticRuleOperandLocator {
	diagnosticRuleOperandLocator := DiagnosticRuleOperandLocator{
		diagnostic: locator,
		index:      index,
	}
	return diagnosticRuleOperandLocator
}

// SignalAction returns a locator that follows the public signal action relationship from the
// located Diagnostic. The returned locator carries locator coordinates, not observed state.
func (locator DiagnosticLocator) SignalAction() DiagnosticSignalActionFailureLocator {
	diagnosticSignalActionFailureLocator := DiagnosticSignalActionFailureLocator{
		diagnostic: locator,
	}
	return diagnosticSignalActionFailureLocator
}

// SignalOverflow returns a locator that follows the public signal overflow relationship from
// the located Diagnostic. The returned locator carries locator coordinates, not observed state.
func (locator DiagnosticLocator) SignalOverflow() DiagnosticSignalOverflowFailureLocator {
	diagnosticSignalOverflowFailureLocator := DiagnosticSignalOverflowFailureLocator{
		diagnostic: locator,
	}
	return diagnosticSignalOverflowFailureLocator
}

// NoticeDelivery returns a locator that follows the public notice delivery relationship from
// the located Diagnostic. The returned locator carries locator coordinates, not observed state.
func (locator DiagnosticLocator) NoticeDelivery() DiagnosticNoticeDeliveryFailureLocator {
	diagnosticNoticeDeliveryFailureLocator := DiagnosticNoticeDeliveryFailureLocator{
		diagnostic: locator,
	}
	return diagnosticNoticeDeliveryFailureLocator
}

// SpanSource returns an InputSourceLocator for the public source referenced by the located Diagnostic span.
func (locator DiagnosticLocator) SpanSource() InputSourceLocator {
	return diagnosticSpanSourceLocator(locator)
}

// SpanPositioned returns an outcome expectation that the located Diagnostic has a span whose start line and column are
// both positive.
func (locator DiagnosticLocator) SpanPositioned() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertySpanPositioned,
	}
	return expectation
}

// Reason returns an outcome expectation that the located Diagnostic Reason exactly equals reason.
func (locator DiagnosticLocator) Reason(reason string) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.state,
		property: diagnosticPropertyReason,
		text:     reason,
	}
	return expectation
}

// On refines a Diagnostic coordinate by the structured public subject identity carried by the diagnostic observation.
// It does not derive subject identity from localized or display text.
func (locator DiagnosticLocator) On(subject DiagnosticSubject) DiagnosticLocator {
	return diagnosticOn(locator, subject)
}

// RepairAt returns the repair at index in the located Diagnostic's ordered RepairIndexes relationship, preserving the
// repair identity carried by the same DiagnosticReport.
func (locator DiagnosticLocator) RepairAt(index int) RepairLocator {
	return diagnosticRepairAt(locator, index)
}

// Present returns an outcome expectation that the Diagnostic coordinate resolves in the structured diagnostic report.
func (locator DiagnosticLocator) Present() OutcomeExpectation {
	return diagnosticPresentExpectation(locator)
}

// Code returns an outcome expectation that the located Diagnostic carries exactly code.
func (locator DiagnosticLocator) Code(code cli.DiagnosticCode) OutcomeExpectation {
	return diagnosticCodeExpectation(locator, code)
}

// Absent returns an outcome expectation that the Diagnostic coordinate is absent from a complete DiagnosticReport. A
// truncated report cannot establish absence, and a code-only match in a truncated report cannot establish uniqueness.
func (locator DiagnosticLocator) Absent() OutcomeExpectation {
	return diagnosticAbsentExpectation(locator)
}

// Rule refines the Diagnostic coordinate to the exact RuleCode relationship carried by the structured Diagnostic.
func (locator DiagnosticLocator) Rule(code cli.RuleCode) DiagnosticLocator {
	return diagnosticRule(locator, code)
}

// Advisory refines the Diagnostic coordinate to the exact AdvisoryCode relationship carried by the structured
// Diagnostic.
func (locator DiagnosticLocator) Advisory(code cli.AdvisoryCode) DiagnosticLocator {
	return diagnosticAdvisory(locator, code)
}

// Violation returns an outcome expectation that the located Diagnostic has Rule failure detail with exactly kind as its
// ViolationKind.
func (locator DiagnosticLocator) Violation(kind cli.ViolationKind) OutcomeExpectation {
	return diagnosticViolationExpectation(locator, kind)
}

// ProducerKind returns an outcome expectation that the located Diagnostic carries exactly the expected producer kind.
func (locator DiagnosticLocator) ProducerKind(source cli.ValueProducerKind) OutcomeExpectation {
	return diagnosticProducerKindExpectation(locator, source)
}

// Tier returns an outcome expectation that the located Diagnostic carries exactly tier as its resolution tier.
func (locator DiagnosticLocator) Tier(tier cli.ResolutionTier) OutcomeExpectation {
	return diagnosticTierExpectation(locator, tier)
}

// Disposition returns an outcome expectation that the located Diagnostic carries exactly disposition as its
// value-candidate disposition.
func (locator DiagnosticLocator) Disposition(disposition cli.ValueCandidateDisposition) OutcomeExpectation {
	return diagnosticDispositionExpectation(locator, disposition)
}

// ExitCode returns an outcome expectation that the located Diagnostic carries exactly code as its structured exit-code
// fact.
func (locator DiagnosticLocator) ExitCode(code int) OutcomeExpectation {
	return diagnosticExitCodeExpectation(locator, code)
}

// Sensitive returns an outcome expectation that the structured Diagnostic is marked sensitive. Sensitivity remains a
// semantic fact of the Diagnostic; it is not inferred from message text.
func (locator DiagnosticLocator) Sensitive() OutcomeExpectation {
	return diagnosticSensitiveExpectation(locator)
}

// InputRedacted returns an outcome expectation that the Diagnostic explicitly marks its input as redacted. It checks
// the structured redaction fact without attempting to reconstruct protected input.
func (locator DiagnosticLocator) InputRedacted() OutcomeExpectation {
	return diagnosticInputRedactedExpectation(locator)
}

// InputSource returns an outcome expectation that the Diagnostic's InputSource identity equals the structured input
// source selected by source in the same diagnostic report.
func (locator DiagnosticLocator) InputSource(source InputSourceLocator) OutcomeExpectation {
	return diagnosticInputSourceExpectation(locator, source)
}

// Span returns an outcome expectation that the Diagnostic carries a span and that it exactly equals span.
func (locator DiagnosticLocator) Span(span cli.InputSpan) OutcomeExpectation {
	return diagnosticSpanExpectation(locator, span)
}

// ProviderOutput returns an outcome expectation that the Diagnostic carries exactly output as its structured
// command-provider output locator. Sensitive or redacted provider-output mismatches do not expose protected details.
func (locator DiagnosticLocator) ProviderOutput(output cli.CommandProviderOutputLocator) OutcomeExpectation {
	return diagnosticProviderOutputExpectation(locator, output)
}

// OriginsExactly returns an outcome expectation that the Diagnostic's origin identities exactly equal origins in
// authoritative order. Expected origins are resolved only within the same diagnostic report.
func (locator DiagnosticLocator) OriginsExactly(origins ...InputOriginLocator) OutcomeExpectation {
	return diagnosticOriginsExactlyExpectation(locator, origins)
}

// ReplacedBy returns an outcome expectation that the Diagnostic's structured Replacement.Target exactly equals the
// declaration identity already carried by replacement. It does not treat caller-authored locator shape as checknance or
// perform a separate declaration lookup.
func (locator DiagnosticLocator) ReplacedBy(replacement DeclarationIdentityLocator) OutcomeExpectation {
	return diagnosticReplacementExpectation(locator, replacement)
}

// RepairCount returns an outcome expectation that the Diagnostic carries exactly count repair relationships.
func (locator DiagnosticLocator) RepairCount(count int) OutcomeExpectation {
	return diagnosticRepairCountExpectation(locator, count)
}

// SignalActionFailure returns an outcome expectation that the Diagnostic carries structured SignalAction failure detail
// exactly equal to failure, including its copied command path and signal-event data.
func (locator DiagnosticLocator) SignalActionFailure(failure cli.SignalActionFailure) OutcomeExpectation {
	return diagnosticSignalActionFailureExpectation(locator, failure)
}

// SignalOverflowFailure returns an outcome expectation that the Diagnostic carries structured SignalOverflow failure
// detail exactly equal to failure.
func (locator DiagnosticLocator) SignalOverflowFailure(failure cli.SignalOverflowFailure) OutcomeExpectation {
	return diagnosticSignalOverflowFailureExpectation(locator, failure)
}

// NoticeDeliveryFailure returns an outcome expectation that the Diagnostic carries structured NoticeDelivery failure
// detail equal to failure, including exact CommandPath order.
func (locator DiagnosticLocator) NoticeDeliveryFailure(failure cli.NoticeDeliveryFailure) OutcomeExpectation {
	return diagnosticNoticeDeliveryFailureExpectation(locator, failure)
}

// Sensitive returns a reusable outcome expectation that checks the located Diagnostic Rule
// operand is marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator DiagnosticRuleOperandLocator) Sensitive() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertyRuleOperandSensitive,
		operand:  locator.index,
	}
	return expectation
}

// Literal returns a locator that follows the public literal relationship from the located
// Diagnostic Rule operand. The returned locator carries locator coordinates, not observed state.
func (locator DiagnosticRuleOperandLocator) Literal() DiagnosticValueLiteralLocator {
	diagnosticValueLiteralLocator := DiagnosticValueLiteralLocator{
		diagnostic: locator.diagnostic,
		operand:    locator.index,
	}
	return diagnosticValueLiteralLocator
}

// NoLiteral returns a reusable outcome expectation that checks the located Diagnostic Rule
// operand explicitly has no literal.
func (locator DiagnosticRuleOperandLocator) NoLiteral() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertyRuleOperandNoLiteral,
		operand:  locator.index,
	}
	return expectation
}

// Type returns a reusable outcome expectation that checks the public type exposed by the
// located Diagnostic Rule literal.
func (locator DiagnosticValueLiteralLocator) Type(expected cli.ValueType) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:   locator.diagnostic.state,
		property:  diagnosticPropertyLiteralType,
		operand:   locator.operand,
		valueType: expected,
	}
	return expectation
}

// Text returns an outcome expectation that checks the Diagnostic literal text when safely
// observable. A mismatch reports redacted context instead of revealing sensitive or redacted
// literal material.
func (locator DiagnosticValueLiteralLocator) Text(expected string) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertyLiteralText,
		operand:  locator.operand,
		text:     expected,
	}
	return expectation
}

// Sensitive returns a reusable outcome expectation that checks the located Diagnostic Rule
// literal is marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator DiagnosticValueLiteralLocator) Sensitive() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertyLiteralSensitive,
		operand:  locator.operand,
	}
	return expectation
}

// Redacted returns a reusable outcome expectation that checks the located Diagnostic Rule
// literal is marked redacted without exposing the protected representation.
func (locator DiagnosticValueLiteralLocator) Redacted() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertyLiteralRedacted,
		operand:  locator.operand,
	}
	return expectation
}

// PolicyIdentityPresent returns an outcome expectation that checks the signal-action failure
// carries a nonzero public signal-policy identity.
func (locator DiagnosticSignalActionFailureLocator) PolicyIdentityPresent() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalActionPolicyIdentityPresent,
	}
	return expectation
}

// Index returns a reusable outcome expectation that checks the public index exposed by the
// located signal-action failure.
func (locator DiagnosticSignalActionFailureLocator) Index(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalActionIndex,
		integer:  expected,
	}
	return expectation
}

// Phase returns a reusable outcome expectation that checks the public phase exposed by the
// located signal-action failure.
func (locator DiagnosticSignalActionFailureLocator) Phase(expected cli.SignalPhase) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:     locator.diagnostic.state,
		property:    diagnosticPropertySignalActionPhase,
		signalPhase: expected,
	}
	return expectation
}

// CommandPathExactly returns a reusable outcome expectation that checks the complete command
// path exactly in authoritative public order.
func (locator DiagnosticSignalActionFailureLocator) CommandPathExactly(expected ...string) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalActionCommandPath,
		strings:  slices.Clone(expected),
	}
	return expectation
}

// Action returns a locator that follows the public action relationship from the located
// signal-action failure. The returned locator carries locator coordinates, not observed state.
func (locator DiagnosticSignalActionFailureLocator) Action() DiagnosticSignalActionLocator {
	return DiagnosticSignalActionLocator(locator)
}

// Event returns a locator that follows the public event relationship from the located
// signal-action failure. The returned locator carries locator coordinates, not observed state.
func (locator DiagnosticSignalActionFailureLocator) Event() DiagnosticSignalEventLocator {
	return DiagnosticSignalEventLocator(locator)
}

// Kind returns a reusable outcome expectation that checks the public kind exposed by the
// located failed signal action.
func (locator DiagnosticSignalActionLocator) Kind(expected cli.SignalActionKind) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:          locator.diagnostic.state,
		property:         diagnosticPropertySignalActionKind,
		signalActionKind: expected,
	}
	return expectation
}

// Name returns a reusable outcome expectation that checks the public name exposed by the
// located failed signal action.
func (locator DiagnosticSignalActionLocator) Name(expected string) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalActionName,
		text:     expected,
	}
	return expectation
}

// Signal returns a reusable outcome expectation that checks the public signal exposed by the
// located signal event.
func (locator DiagnosticSignalEventLocator) Signal(expected cli.Signal) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventSignal,
		signal:   expected,
	}
	return expectation
}

// Origin returns a reusable outcome expectation that checks the public origin exposed by the
// located signal event.
func (locator DiagnosticSignalEventLocator) Origin(expected cli.SignalOrigin) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:      locator.diagnostic.state,
		property:     diagnosticPropertySignalEventOrigin,
		signalOrigin: expected,
	}
	return expectation
}

// Sender returns a reusable outcome expectation that checks the public sender exposed by the
// located signal event.
func (locator DiagnosticSignalEventLocator) Sender(expected cli.SignalSender) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:      locator.diagnostic.state,
		property:     diagnosticPropertySignalEventSender,
		signalSender: expected,
	}
	return expectation
}

// NoSender returns a reusable outcome expectation that checks the located signal event
// explicitly has no sender.
func (locator DiagnosticSignalEventLocator) NoSender() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventNoSender,
	}
	return expectation
}

// Phase returns a reusable outcome expectation that checks the public phase exposed by the
// located signal event.
func (locator DiagnosticSignalEventLocator) Phase(expected cli.SignalPhase) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:     locator.diagnostic.state,
		property:    diagnosticPropertySignalEventPhase,
		signalPhase: expected,
	}
	return expectation
}

// ObservedAt returns a reusable outcome expectation that checks the public observed time
// exposed by the located signal event.
func (locator DiagnosticSignalEventLocator) ObservedAt(expected time.Time) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventObservedAt,
		at:       expected,
	}
	return expectation
}

// FirstObservedAt returns a reusable outcome expectation that checks the public first-observed
// time exposed by the located signal event.
func (locator DiagnosticSignalEventLocator) FirstObservedAt(expected time.Time) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventFirstObservedAt,
		at:       expected,
	}
	return expectation
}

// LastObservedAt returns a reusable outcome expectation that checks the public last-observed
// time exposed by the located signal event.
func (locator DiagnosticSignalEventLocator) LastObservedAt(expected time.Time) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventLastObservedAt,
		at:       expected,
	}
	return expectation
}

// FirstOccurrence returns a reusable outcome expectation that checks the public first
// occurrence exposed by the located signal event.
func (locator DiagnosticSignalEventLocator) FirstOccurrence(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventFirstOccurrence,
		integer:  expected,
	}
	return expectation
}

// LastOccurrence returns a reusable outcome expectation that checks the public last occurrence
// exposed by the located signal event.
func (locator DiagnosticSignalEventLocator) LastOccurrence(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventLastOccurrence,
		integer:  expected,
	}
	return expectation
}

// BatchSize returns a reusable outcome expectation that checks the public batch size exposed by
// the located signal event.
func (locator DiagnosticSignalEventLocator) BatchSize(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalEventBatchSize,
		integer:  expected,
	}
	return expectation
}

// Boundary returns a reusable outcome expectation that checks the public overflow boundary
// exposed by the located signal-overflow failure.
func (locator DiagnosticSignalOverflowFailureLocator) Boundary(expected cli.SignalOverflowBoundary) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:          locator.diagnostic.state,
		property:         diagnosticPropertySignalOverflowBoundary,
		overflowBoundary: expected,
	}
	return expectation
}

// EndpointIdentityPresent returns an outcome expectation that checks the overflow failure
// carries a nonzero public signal-endpoint identity.
func (locator DiagnosticSignalOverflowFailureLocator) EndpointIdentityPresent() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalOverflowEndpointPresent,
	}
	return expectation
}

// NoEndpoint returns an outcome expectation that checks the overflow failure explicitly carries
// no endpoint identity; it does not infer absence from unrelated Diagnostic state.
func (locator DiagnosticSignalOverflowFailureLocator) NoEndpoint() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalOverflowNoEndpoint,
	}
	return expectation
}

// Capacity returns a reusable outcome expectation that checks the public capacity exposed by
// the located signal-overflow failure.
func (locator DiagnosticSignalOverflowFailureLocator) Capacity(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalOverflowCapacity,
		integer:  expected,
	}
	return expectation
}

// Buffered returns a reusable outcome expectation that checks the public buffered count exposed
// by the located signal-overflow failure.
func (locator DiagnosticSignalOverflowFailureLocator) Buffered(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalOverflowBuffered,
		integer:  expected,
	}
	return expectation
}

// DoesNotExist returns an outcome expectation that the located Diagnostic has no structured SignalOverflow failure
// payload.
// It tests absence of that diagnostic detail, not absence of the Diagnostic itself.
func (locator DiagnosticSignalOverflowFailureLocator) DoesNotExist() OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertySignalOverflowAbsent,
	}
	return expectation
}

// Index returns a reusable outcome expectation that checks the public index exposed by the
// located Notice-delivery failure.
func (locator DiagnosticNoticeDeliveryFailureLocator) Index(expected int) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertyNoticeIndex,
		integer:  expected,
	}
	return expectation
}

// Phase returns a reusable outcome expectation that checks the public phase exposed by the
// located Notice-delivery failure.
func (locator DiagnosticNoticeDeliveryFailureLocator) Phase(expected cli.NoticePhase) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:     locator.diagnostic.state,
		property:    diagnosticPropertyNoticePhase,
		noticePhase: expected,
	}
	return expectation
}

// Code returns a reusable outcome expectation that checks the public Notice code exposed by the
// located Notice-delivery failure.
func (locator DiagnosticNoticeDeliveryFailureLocator) Code(expected cli.NoticeCode) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:    locator.diagnostic.state,
		property:   diagnosticPropertyNoticeCode,
		noticeCode: expected,
	}
	return expectation
}

// CommandPathExactly returns a reusable outcome expectation that checks the complete command
// path exactly in authoritative public order.
func (locator DiagnosticNoticeDeliveryFailureLocator) CommandPathExactly(expected ...string) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:  locator.diagnostic.state,
		property: diagnosticPropertyNoticeCommandPath,
		strings:  slices.Clone(expected),
	}
	return expectation
}

type diagnosticInputGraphClosedExpectation struct{}

type repairsExactlyExpectation struct {
	states []*repairLocatorState
}

type diagnosticProperty uint8

const (
	diagnosticPropertyInvalid diagnosticProperty = iota
	diagnosticPropertyPresent
	diagnosticPropertyAbsent
	diagnosticPropertyCode
	diagnosticPropertyViolation
	diagnosticPropertyProducerKind
	diagnosticPropertyTier
	diagnosticPropertyDisposition
	diagnosticPropertyExitCode
	diagnosticPropertySensitive
	diagnosticPropertyInputRedacted
	diagnosticPropertySpan
	diagnosticPropertySpanPositioned
	diagnosticPropertyReason
	diagnosticPropertyRepairCount
	diagnosticPropertyProviderOutput
	diagnosticPropertySignalActionFailure
	diagnosticPropertySignalOverflowFailure
	diagnosticPropertyNoticeDeliveryFailure
	diagnosticPropertyRuleIdentityPresent
	diagnosticPropertyRuleKind
	diagnosticPropertyRuleOperandCount
	diagnosticPropertyRuleOperandSensitive
	diagnosticPropertyRuleOperandNoLiteral
	diagnosticPropertyLiteralType
	diagnosticPropertyLiteralText
	diagnosticPropertyLiteralSensitive
	diagnosticPropertyLiteralRedacted
	diagnosticPropertySignalActionPolicyIdentityPresent
	diagnosticPropertySignalActionIndex
	diagnosticPropertySignalActionPhase
	diagnosticPropertySignalActionCommandPath
	diagnosticPropertySignalActionKind
	diagnosticPropertySignalActionName
	diagnosticPropertySignalEventSignal
	diagnosticPropertySignalEventOrigin
	diagnosticPropertySignalEventSender
	diagnosticPropertySignalEventNoSender
	diagnosticPropertySignalEventPhase
	diagnosticPropertySignalEventObservedAt
	diagnosticPropertySignalEventFirstObservedAt
	diagnosticPropertySignalEventLastObservedAt
	diagnosticPropertySignalEventFirstOccurrence
	diagnosticPropertySignalEventLastOccurrence
	diagnosticPropertySignalEventBatchSize
	diagnosticPropertySignalOverflowBoundary
	diagnosticPropertySignalOverflowEndpointPresent
	diagnosticPropertySignalOverflowNoEndpoint
	diagnosticPropertySignalOverflowCapacity
	diagnosticPropertySignalOverflowBuffered
	diagnosticPropertySignalOverflowAbsent
	diagnosticPropertyNoticeIndex
	diagnosticPropertyNoticePhase
	diagnosticPropertyNoticeCode
	diagnosticPropertyNoticeCommandPath
)

type diagnosticExpectation struct {
	locator  *diagnosticLocatorState
	property diagnosticProperty

	operand int
	integer int
	text    string
	strings []string

	code           cli.DiagnosticCode
	violation      cli.ViolationKind
	producerKind   cli.ValueProducerKind
	tier           cli.ResolutionTier
	disposition    cli.ValueCandidateDisposition
	span           cli.InputSpan
	providerOutput cli.CommandProviderOutputLocator

	ruleKind  cli.RuleKind
	valueType cli.ValueType

	signalActionFailure   cli.SignalActionFailure
	signalOverflowFailure cli.SignalOverflowFailure
	noticeDeliveryFailure cli.NoticeDeliveryFailure
	signalPhase           cli.SignalPhase
	signalActionKind      cli.SignalActionKind
	signal                cli.Signal
	signalOrigin          cli.SignalOrigin
	signalSender          cli.SignalSender
	at                    time.Time
	overflowBoundary      cli.SignalOverflowBoundary
	noticePhase           cli.NoticePhase
	noticeCode            cli.NoticeCode
}

const failureSubjectMaxLocatorDepth = 24

type diagnosticInputSourceExpectationValue struct {
	diagnostic *diagnosticLocatorState
	source     *inputSourceLocatorState
}

type diagnosticOriginsExpectationValue struct {
	diagnostic *diagnosticLocatorState
	origins    []*inputOriginLocatorState
}

type diagnosticReplacementExpectationValue struct {
	diagnostic  *diagnosticLocatorState
	replacement DeclarationIdentityLocator
}

type valueDiagnosticsExactlyExpectation struct {
	locator  *valueLocatorState
	expected []DiagnosticLocator
}

type candidateDiagnosticsExactlyExpectation struct {
	locator  *valueCandidateLocatorState
	expected []DiagnosticLocator
}

type diagnosticSubjectKind uint8

const (
	diagnosticSubjectInvalid diagnosticSubjectKind = iota
	diagnosticSubjectCommandPath
	diagnosticSubjectCommand
	diagnosticSubjectDeclaration
	diagnosticSubjectDefault
	diagnosticSubjectProvider
)

type resolvedDiagnosticSubject struct {
	kind        diagnosticSubjectKind
	commandPath []string
	command     cli.CommandID
	declaration cli.DeclarationIdentity
	defaultID   cli.CommandDefaultID
	providerID  cli.CommandProviderID
}

type diagnosticLocatorMode uint8

const (
	diagnosticLocatorInvalid diagnosticLocatorMode = iota
	diagnosticLocatorCode
	diagnosticLocatorIndex
)

type diagnosticLocatorState struct {
	mode  diagnosticLocatorMode
	code  cli.DiagnosticCode
	index int

	ruleSet     bool
	rule        cli.RuleCode
	advisorySet bool
	advisory    cli.AdvisoryCode
	subjectSet  bool
	subject     DiagnosticSubject
}

func resolveDiagnosticRuleOperand(d cli.Diagnostic, index int) (cli.RuleOperand, bool, evaluationField) {
	if d.Rule == nil {
		return cli.RuleOperand{}, false, observed(semanticSubjectDiagnostic, observationNoRuleFailure)
	}
	if index < 0 || index >= len(d.Rule.Operands) {
		return cli.RuleOperand{}, false, observed(semanticSubjectRuleOperand, observationAbsent)
	}
	return d.Rule.Operands[index], true, evaluationText("")
}

func resolveDiagnosticLiteral(d cli.Diagnostic, index int) (cli.ValueLiteral, bool, evaluationField) {
	operand, ok, detail := resolveDiagnosticRuleOperand(d, index)
	if !ok {
		return cli.ValueLiteral{}, false, detail
	}
	if operand.Literal == nil {
		return cli.ValueLiteral{}, false, observed(semanticSubjectRuleOperandLiteral, observationAbsent)
	}
	return *operand.Literal, true, evaluationText("")
}

func sourcePresent(report cli.DiagnosticReport, id cli.InputSourceID) bool {
	if id == (cli.InputSourceID{}) {
		return true
	}
	_, ok := report.Sources[id]
	return ok
}

func originPresent(report cli.DiagnosticReport, id cli.InputOriginID) bool {
	if id == (cli.InputOriginID{}) {
		return true
	}
	_, ok := report.Origins[id]
	return ok
}

func diagnosticInputClosureDetail(report cli.DiagnosticReport) string {
	for index, diagnostic := range report.Diagnostics {
		if !sourcePresent(report, diagnostic.InputSource) {
			return fmt.Sprintf("diagnostic %d references missing input source", index)
		}
		if diagnostic.Span != nil && !sourcePresent(report, diagnostic.Span.Source) {
			return fmt.Sprintf("diagnostic %d span references missing input source", index)
		}
		for _, id := range diagnostic.Origins {
			if !originPresent(report, id) {
				return fmt.Sprintf("diagnostic %d references missing input origin", index)
			}
		}
		for _, repair := range diagnostic.RepairIndexes {
			if repair < 0 || repair >= len(report.Repairs) {
				return fmt.Sprintf("diagnostic %d references unavailable repair", index)
			}
		}
	}
	for _, origin := range report.Origins {
		if !sourcePresent(report, origin.Span.Source) {
			return "input origin span references missing source"
		}
		for _, parent := range origin.Parents {
			if !originPresent(report, parent) {
				return "input origin references missing parent"
			}
		}
	}
	for repairIndex, repair := range report.Repairs {
		for sourceIndex, source := range repair.Sources {
			if !sourcePresent(report, source.Source) {
				return fmt.Sprintf("repair %d source %d references missing input source", repairIndex, sourceIndex)
			}
			for _, edit := range source.TextEdits {
				if !sourcePresent(report, edit.Span.Source) {
					return fmt.Sprintf("repair %d text edit references missing input source", repairIndex)
				}
				for _, part := range edit.Replacement {
					if part.Copy != nil && !sourcePresent(report, part.Copy.Source) {
						return fmt.Sprintf("repair %d text replacement references missing input source", repairIndex)
					}
				}
			}
			for _, edit := range source.EntryEdits {
				for _, replacement := range edit.Entries {
					for _, part := range replacement {
						if part.Copy != nil && !sourcePresent(report, part.Copy.Source) {
							return fmt.Sprintf(
								"repair %d entry replacement references missing input source",
								repairIndex,
							)
						}
					}
				}
			}
		}
	}
	return ""
}

func valueProducerKindFailureLabel(kind cli.ValueProducerKind) string {
	switch kind {
	case cli.ValueProducerKindUnknown:
		return "unknown"
	case cli.ValueProducerKindFlag:
		return "flag"
	case cli.ValueProducerKindArgument:
		return "argument"
	case cli.ValueProducerKindRemainder:
		return "remainder"
	case cli.ValueProducerKindPassThrough:
		return "pass-through"
	case cli.ValueProducerKindShortcutDefault:
		return "shortcut-default"
	case cli.ValueProducerKindEnvironment:
		return semanticNameEnvironment
	case cli.ValueProducerKindInvocationDefault:
		return "invocation-default"
	case cli.ValueProducerKindComputedDefault:
		return "computed-default"
	case cli.ValueProducerKindStaticDefault:
		return "static-default"
	case cli.ValueProducerKindInitial:
		return "initial"
	case cli.ValueProducerKindSuppliedFact:
		return "supplied-fact"
	case cli.ValueProducerKindResolvedFact:
		return "resolved-fact"
	case cli.ValueProducerKindDerived:
		return "derived"
	default:
		return fmt.Sprintf("producer-kind(%d)", kind)
	}
}

func diagnosticReportFromInspection(inspection cli.Inspection) cli.DiagnosticReport {
	report := cli.DiagnosticReport{
		Diagnostics: inspection.Diagnostics,
		Sources:     inspection.Sources,
		Origins:     inspection.Origins,
		Repairs:     inspection.Repairs,
		Truncated:   inspection.Truncated,
	}

	return report
}

func diagnosticInputSourceExpectation(locator DiagnosticLocator, source InputSourceLocator) OutcomeExpectation {
	value := diagnosticInputSourceExpectationValue{
		diagnostic: locator.state,
		source:     source.state,
	}
	return value
}

func diagnosticProviderOutputExpectation(locator DiagnosticLocator, output cli.CommandProviderOutputLocator) OutcomeExpectation {
	expectation := diagnosticExpectation{
		locator:        locator.state,
		property:       diagnosticPropertyProviderOutput,
		providerOutput: output,
	}
	return expectation
}

func diagnosticOriginsExactlyExpectation(locator DiagnosticLocator, origins []InputOriginLocator) OutcomeExpectation {
	states := make([]*inputOriginLocatorState, len(origins))
	for i, origin := range origins {
		states[i] = origin.state
	}
	value := diagnosticOriginsExpectationValue{
		diagnostic: locator.state,
		origins:    states,
	}
	return value
}

func diagnosticReplacementExpectation(locator DiagnosticLocator, replacement DeclarationIdentityLocator) OutcomeExpectation {
	value := diagnosticReplacementExpectationValue{
		diagnostic:  locator.state,
		replacement: replacement,
	}
	return value
}

func evaluateDiagnosticIndexesExactly(subject evaluationField, indexes []int, expected []DiagnosticLocator, inspection cli.Inspection) evaluationResult {
	expectation := evaluationText("exact attached diagnostics in public order")
	if len(indexes) != len(expected) {
		return disprovedResult(subject, expectation, evaluationText(fmt.Sprintf("%d attached diagnostics", len(indexes))))
	}
	for index, locator := range expected {
		resolved := resolveDiagnosticStateForInspection(locator.state, inspection)
		if resolved.status == locatorResolutionNotProvable {
			return notProvableResult(subject, expectation, nonempty(resolved.detail, evaluationText(fmt.Sprintf("expected diagnostic at position %d is not provable", index))))
		}
		if resolved.status != locatorResolutionFound {
			return disprovedResult(subject, expectation, nonempty(resolved.detail, evaluationText(fmt.Sprintf("expected diagnostic at position %d is not uniquely resolvable", index))))
		}
		if indexes[index] != resolved.value.index {
			return disprovedResult(subject, expectation, evaluationText(fmt.Sprintf("different diagnostic at position %d", index)))
		}
	}
	return provedResult()
}

func diagnosticSpanSourceLocator(diagnostic DiagnosticLocator) InputSourceLocator {
	state := &inputSourceLocatorState{
		mode:       inputSourceLocatorDiagnosticSpan,
		diagnostic: diagnostic.state,
	}
	locator := InputSourceLocator{
		state: state,
	}
	return locator
}

func diagnosticSubjectForReport(subject DiagnosticSubject) (resolvedDiagnosticSubject, locatorResolutionStatus, evaluationField) {
	switch subject := subject.(type) {
	case CommandLocator:
		if subject.state == nil {
			return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
		}
		switch subject.state.mode {
		case commandLocatorID:
			resolvedSubject := resolvedDiagnosticSubject{
				kind:    diagnosticSubjectCommand,
				command: subject.state.id,
			}
			return resolvedSubject, locatorResolutionFound, evaluationText("")
		case commandLocatorPath, commandLocatorChild:
			path, ok := commandPathCoordinate(subject.state)
			if !ok {
				return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
			}
			resolvedSubject := resolvedDiagnosticSubject{
				kind:        diagnosticSubjectCommandPath,
				commandPath: path,
			}
			return resolvedSubject, locatorResolutionFound, evaluationText("")
		default:
			return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
		}
	case CommandEntryLocator, ActionEntryLocator, FlagLocator, FlagSpellingLocator, ArgumentFormLocator, ArgumentTermLocator, EnvironmentInputLocator, ValueDomainEntryLocator:
		identityLocator, ok := subject.(DeclarationIdentityLocator)
		if !ok {
			panic("clitest: declaration diagnostic subject lacks declaration identity")
		}
		identity, ok := locatorDeclarationIdentityValue(identityLocator)
		if !ok {
			return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
		}
		resolvedSubject := resolvedDiagnosticSubject{
			kind:        diagnosticSubjectDeclaration,
			declaration: identity,
		}
		return resolvedSubject, locatorResolutionFound, evaluationText("")
	case CommandDefaultLocator:
		if subject.state == nil || subject.state.mode != commandDefaultLocatorID {
			return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
		}
		resolvedSubject := resolvedDiagnosticSubject{
			kind:      diagnosticSubjectDefault,
			defaultID: subject.state.id,
		}
		return resolvedSubject, locatorResolutionFound, evaluationText("")
	case CommandProviderLocator:
		if subject.state == nil || subject.state.mode != commandProviderLocatorID {
			return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
		}
		resolvedSubject := resolvedDiagnosticSubject{
			kind:       diagnosticSubjectProvider,
			providerID: subject.state.id,
		}
		return resolvedSubject, locatorResolutionFound, evaluationText("")
	default:
		return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
	}
}

func diagnosticSubjectForInspection(subject DiagnosticSubject, inspection cli.Inspection) (resolvedDiagnosticSubject, locatorResolutionStatus, evaluationField) {
	switch subject := subject.(type) {
	case CommandLocator:
		resolved := resolveInspectionCommandLocator(subject.state, inspection)
		if resolved.status != locatorResolutionFound {
			return resolvedDiagnosticSubject{}, resolved.status, resolved.detail
		}
		resolvedSubject := resolvedDiagnosticSubject{
			kind:    diagnosticSubjectCommand,
			command: resolved.value.ID,
		}
		return resolvedSubject, locatorResolutionFound, evaluationText("")
	case CommandEntryLocator, ActionEntryLocator, FlagLocator, FlagSpellingLocator, ArgumentFormLocator, ArgumentTermLocator, EnvironmentInputLocator, ValueDomainEntryLocator:
		identityLocator, ok := subject.(DeclarationIdentityLocator)
		if !ok {
			panic("clitest: declaration diagnostic subject lacks declaration identity")
		}
		identity, status, detail := resolveExpectedDeclarationIdentity(identityLocator, nil, nil, inspection)
		if status != locatorResolutionFound {
			return resolvedDiagnosticSubject{}, status, detail
		}
		resolvedSubject := resolvedDiagnosticSubject{
			kind:        diagnosticSubjectDeclaration,
			declaration: identity,
		}
		return resolvedSubject, locatorResolutionFound, evaluationText("")
	case CommandDefaultLocator:
		resolved := resolveInspectionCommandDefaultLocator(subject.state, inspection)
		if resolved.status != locatorResolutionFound {
			return resolvedDiagnosticSubject{}, resolved.status, resolved.detail
		}
		resolvedSubject := resolvedDiagnosticSubject{
			kind:      diagnosticSubjectDefault,
			defaultID: resolved.value.ID,
		}
		return resolvedSubject, locatorResolutionFound, evaluationText("")
	case CommandProviderLocator:
		resolved := resolveInspectionCommandProviderLocator(subject.state, inspection)
		if resolved.status != locatorResolutionFound {
			return resolvedDiagnosticSubject{}, resolved.status, resolved.detail
		}
		resolvedSubject := resolvedDiagnosticSubject{
			kind:       diagnosticSubjectProvider,
			providerID: resolved.value.ID,
		}
		return resolvedSubject, locatorResolutionFound, evaluationText("")
	default:
		return resolvedDiagnosticSubject{}, locatorResolutionInvalid, observed(semanticSubjectLocator, observationInvalidLocator)
	}
}

func diagnosticLabel(state *diagnosticLocatorState) evaluationField {
	if state == nil {
		return evaluationText("diagnostic")
	}
	if state.mode == diagnosticLocatorIndex {
		return evaluationText(fmt.Sprintf("diagnostic at index %d", state.index))
	}
	return subjectNamed(evaluationText("diagnostic"), string(state.code))
}

func diagnosticMatchesPremise(state *diagnosticLocatorState, subject *resolvedDiagnosticSubject, diagnostic cli.Diagnostic) bool {
	if state.ruleSet && (diagnostic.Rule == nil || diagnostic.Rule.RuleCode != state.rule) {
		return false
	}
	if state.advisorySet && diagnostic.AdvisoryCode != state.advisory {
		return false
	}

	return subject == nil || subject.matches(diagnostic)
}

func locateDiagnosticState(state *diagnosticLocatorState, report cli.DiagnosticReport, subject *resolvedDiagnosticSubject) locatorResolution[diagnosticLocation] {
	var resolution locatorResolution[diagnosticLocation]
	label := diagnosticLabel(state)
	if state == nil || (state.mode != diagnosticLocatorCode && state.mode != diagnosticLocatorIndex) {
		resolution = locatorResolution[diagnosticLocation]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectDiagnostic, observationInvalidLocator),
		}
		return resolution
	}
	if state.mode == diagnosticLocatorIndex && state.index < 0 {
		resolution = locatorResolution[diagnosticLocation]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectDiagnostic, observationInvalidLocator),
		}
		return resolution
	}

	if state.mode == diagnosticLocatorIndex {
		if state.index < len(report.Diagnostics) {
			diagnostic := report.Diagnostics[state.index]
			if diagnosticMatchesPremise(state, subject, diagnostic) {
				resolution = locatorResolution[diagnosticLocation]{
					status:  locatorResolutionFound,
					subject: label,
					value: diagnosticLocation{
						diagnostic: diagnostic,
						index:      state.index,
						count:      1,
						subject:    renderEvaluationField(label),
					},
				}
				return resolution
			}
			resolution = locatorResolution[diagnosticLocation]{
				status:  locatorResolutionAbsent,
				subject: label,
				detail:  evaluationText("diagnostic premise is absent"),
			}
			return resolution
		}
		if report.Truncated {
			resolution = locatorResolution[diagnosticLocation]{
				status:  locatorResolutionNotProvable,
				subject: label,
				detail:  observed(semanticSubjectDiagnosticReport, observationTruncated),
			}
			return resolution
		}
		resolution = locatorResolution[diagnosticLocation]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  evaluationText("diagnostic premise is absent"),
		}
		return resolution
	}

	locations := make([]diagnosticLocation, 0, 1)
	for index, diagnostic := range report.Diagnostics {
		if diagnostic.Code == state.code && diagnosticMatchesPremise(state, subject, diagnostic) {
			location := diagnosticLocation{
				diagnostic: diagnostic,
				index:      index,
				count:      1,
				subject:    renderEvaluationField(label),
			}
			locations = append(locations, location)
		}
	}
	if len(locations) > 1 {
		location := locations[0]
		location.count = len(locations)
		resolution = locatorResolution[diagnosticLocation]{
			status:  locatorResolutionAmbiguous,
			subject: label,
			value:   location,
			detail:  evaluationText(diagnosticAmbiguityActual(location)),
		}
		return resolution
	}
	if len(locations) == 1 {
		resolution = locatorResolution[diagnosticLocation]{
			status:  locatorResolutionFound,
			subject: label,
			value:   locations[0],
		}
		return resolution
	}
	if report.Truncated {
		resolution = locatorResolution[diagnosticLocation]{
			status:  locatorResolutionNotProvable,
			subject: label,
			detail:  evaluationText("diagnostic report is truncated and premise is not present"),
		}
		return resolution
	}
	resolution = locatorResolution[diagnosticLocation]{
		status:  locatorResolutionAbsent,
		subject: label,
		detail:  evaluationText("diagnostic premise is absent"),
	}
	return resolution
}

func resolveDiagnosticStateForReport(state *diagnosticLocatorState, report cli.DiagnosticReport) locatorResolution[diagnosticLocation] {
	var resolution locatorResolution[diagnosticLocation]
	if state == nil {
		return locateDiagnosticState(nil, report, nil)
	}
	var subject *resolvedDiagnosticSubject
	if state.subjectSet {
		resolved, status, detail := diagnosticSubjectForReport(state.subject)
		if status != locatorResolutionFound {
			resolution = locatorResolution[diagnosticLocation]{
				status:  status,
				subject: evaluationText("diagnostic subject"),
				detail:  detail,
			}
			return resolution
		}
		subject = &resolved
	}
	return locateDiagnosticState(state, report, subject)
}

func resolveDiagnosticStateForInspection(state *diagnosticLocatorState, inspection cli.Inspection) locatorResolution[diagnosticLocation] {
	var resolution locatorResolution[diagnosticLocation]
	if state == nil {
		return locateDiagnosticState(nil, diagnosticReportFromInspection(inspection), nil)
	}
	var subject *resolvedDiagnosticSubject
	if state.subjectSet {
		resolved, status, detail := diagnosticSubjectForInspection(state.subject, inspection)
		if status != locatorResolutionFound {
			resolution = locatorResolution[diagnosticLocation]{
				status:  status,
				subject: evaluationText("diagnostic subject"),
				detail:  detail,
			}
			return resolution
		}
		subject = &resolved
	}
	return locateDiagnosticState(state, diagnosticReportFromInspection(inspection), subject)
}

func tokenOriginLocator(parent TokenLocator) InputOriginLocator {
	state := &inputOriginLocatorState{
		mode:  inputOriginLocatorToken,
		token: parent.state,
	}
	locator := InputOriginLocator{
		state: state,
	}
	return locator
}

func (diagnosticInputGraphClosedExpectation) diagnosticReportExpectation() {}

func (diagnosticInputGraphClosedExpectation) evaluateDiagnosticReport(report cli.DiagnosticReport) evaluationResult {
	if detail := diagnosticInputClosureDetail(report); detail != "" {
		return disprovedResult(evaluationText("diagnostic input relationships"), evaluationText("all source/origin/repair references closed"), evaluationText(detail))
	}
	return provedResult()
}

func (diagnosticExpectation) inspectionExpectation() {}

func (diagnosticExpectation) completionExpectation() {}

func (diagnosticExpectation) diagnosticReportExpectation() {}

func (expectation diagnosticExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return expectation.evaluateResolved(resolveDiagnosticStateForInspection(expectation.locator, inspection))
}

func (expectation diagnosticExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	report := cli.DiagnosticReport{
		Diagnostics: result.Diagnostics,
		Sources:     result.Sources,
		Origins:     result.Origins,
		Repairs:     result.Repairs,
		Truncated:   result.DiagnosticsTruncated,
	}
	return expectation.evaluateResolved(resolveDiagnosticStateForReport(expectation.locator, report))
}

func (expectation diagnosticExpectation) evaluateDiagnosticReport(report cli.DiagnosticReport) evaluationResult {
	return expectation.evaluateResolved(resolveDiagnosticStateForReport(expectation.locator, report))
}

func (expectation diagnosticExpectation) evaluateResolved(resolution locatorResolution[diagnosticLocation]) evaluationResult {
	if expectation.property == diagnosticPropertyInvalid {
		if resolution.status != locatorResolutionFound {
			return outcomeForValueStatus(resolution.status, resolution.subject, evaluationText("valid diagnostic expectation"), resolution.detail)
		}
		panic("clitest: invalid diagnostic expectation reached resolved value")
	}
	if expectation.property == diagnosticPropertyAbsent {
		switch resolution.status {
		case locatorResolutionAbsent:
			return provedResult()
		case locatorResolutionNotProvable:
			return notProvableResult(resolution.subject, semanticRequirementAbsent, resolution.detail)
		case locatorResolutionFound:
			return disprovedResult(resolution.subject, semanticRequirementAbsent, evaluationText("present"))
		default:
			return disprovedResult(resolution.subject, semanticRequirementAbsent, resolution.detail)
		}
	}
	if expectation.property == diagnosticPropertyPresent && resolution.status == locatorResolutionNotProvable && resolution.value.count == 1 {
		return provedResult()
	}
	expected := expectation.expected()
	if resolution.status != locatorResolutionFound {
		return outcomeForValueStatus(resolution.status, resolution.subject, expected, resolution.detail)
	}
	if expectation.property == diagnosticPropertyPresent {
		return provedResult()
	}
	matched, actual := expectation.match(resolution.value.diagnostic)
	if matched {
		return provedResult()
	}
	return disprovedResult(resolution.subject, expected, actual)
}

func (expectation diagnosticExpectation) expected() evaluationField {
	switch expectation.property {
	case diagnosticPropertyPresent:
		return evaluationText("present")
	case diagnosticPropertyAbsent:
		return semanticRequirementAbsent
	case diagnosticPropertyCode:
		return comparison(evaluationText("code"), evaluationText(fmt.Sprint(expectation.code)))
	case diagnosticPropertyViolation:
		return comparison(evaluationText("violation"), evaluationText(fmt.Sprint(expectation.violation)))
	case diagnosticPropertyProducerKind:
		return evaluationText("producer kind = " + valueProducerKindFailureLabel(expectation.producerKind))
	case diagnosticPropertyTier:
		return comparison(evaluationText("tier"), evaluationText(fmt.Sprint(expectation.tier)))
	case diagnosticPropertyDisposition:
		return comparison(evaluationText("disposition"), evaluationText(fmt.Sprint(expectation.disposition)))
	case diagnosticPropertyExitCode:
		return comparison(evaluationText("exit code"), evaluationText(strconv.Itoa(expectation.integer)))
	case diagnosticPropertySensitive:
		return comparison(semanticRequirementSensitive, evaluationText("true"))
	case diagnosticPropertyInputRedacted:
		return comparison(evaluationText("input redacted"), evaluationText("true"))
	case diagnosticPropertySpan:
		field := inputSpanField{
			value: expectation.span,
		}
		return property(evaluationText("span"), field)
	case diagnosticPropertySpanPositioned:
		return evaluationText("positioned span")
	case diagnosticPropertyReason:
		return comparison(evaluationText("reason"), evaluationText(expectation.text))
	case diagnosticPropertyRepairCount:
		return comparison(evaluationText("repair count"), evaluationText(strconv.Itoa(expectation.integer)))
	case diagnosticPropertyProviderOutput:
		return evaluationText("diagnostic provider output")
	case diagnosticPropertySignalActionFailure:
		return evaluationText("exact signal action failure")
	case diagnosticPropertySignalOverflowFailure:
		return evaluationText("exact signal overflow failure")
	case diagnosticPropertyNoticeDeliveryFailure:
		return evaluationText("exact Notice delivery failure")
	case diagnosticPropertyRuleIdentityPresent:
		return evaluationText("nonzero rule identity")
	case diagnosticPropertyRuleKind:
		return comparison(evaluationText("rule kind"), evaluationText(fmt.Sprint(expectation.ruleKind)))
	case diagnosticPropertyRuleOperandCount:
		return comparison(evaluationText("rule operand count"), evaluationText(strconv.Itoa(expectation.integer)))
	case diagnosticPropertyRuleOperandSensitive:
		return evaluationText(fmt.Sprintf("rule operand %d sensitive", expectation.operand))
	case diagnosticPropertyRuleOperandNoLiteral:
		return evaluationText(fmt.Sprintf("rule operand %d literal absent", expectation.operand))
	case diagnosticPropertyLiteralType:
		field := valueTypeField{
			value: expectation.valueType,
		}
		return qualifySubject(evaluationText(fmt.Sprintf("rule operand %d literal", expectation.operand)), property(evaluationText("type"), field))
	case diagnosticPropertyLiteralText:
		return evaluationText(fmt.Sprintf("rule operand %d literal text", expectation.operand))
	case diagnosticPropertyLiteralSensitive:
		return evaluationText(fmt.Sprintf("rule operand %d literal sensitive", expectation.operand))
	case diagnosticPropertyLiteralRedacted:
		return evaluationText(fmt.Sprintf("rule operand %d literal redacted", expectation.operand))
	case diagnosticPropertySignalActionPolicyIdentityPresent:
		return evaluationText("signal-action policy identity present")
	case diagnosticPropertySignalActionIndex:
		return evaluationText(fmt.Sprintf("signal-action index=%d", expectation.integer))
	case diagnosticPropertySignalActionPhase:
		return evaluationText(fmt.Sprintf("signal-action phase=%v", expectation.signalPhase))
	case diagnosticPropertySignalActionCommandPath:
		field := stringCollectionField{
			values: expectation.strings,
		}
		return property(evaluationText("signal-action command-path"), field)
	case diagnosticPropertySignalActionKind:
		return evaluationText(fmt.Sprintf("signal action kind=%v", expectation.signalActionKind))
	case diagnosticPropertySignalActionName:
		return property(evaluationText("signal action name"), quotedTextField(expectation.text))
	case diagnosticPropertySignalEventSignal:
		return evaluationText("exact signal identity")
	case diagnosticPropertySignalEventOrigin:
		return evaluationText(fmt.Sprintf("origin=%v", expectation.signalOrigin))
	case diagnosticPropertySignalEventSender:
		field := signalSenderField{
			value: expectation.signalSender,
		}
		return property(evaluationText("sender"), field)
	case diagnosticPropertySignalEventNoSender:
		return evaluationText("sender absent")
	case diagnosticPropertySignalEventPhase:
		return evaluationText(fmt.Sprintf("phase=%v", expectation.signalPhase))
	case diagnosticPropertySignalEventObservedAt:
		field := timeField{
			value: expectation.at,
		}
		return property(evaluationText("observed-at"), field)
	case diagnosticPropertySignalEventFirstObservedAt:
		field := timeField{
			value: expectation.at,
		}
		return property(evaluationText("first-observed-at"), field)
	case diagnosticPropertySignalEventLastObservedAt:
		field := timeField{
			value: expectation.at,
		}
		return property(evaluationText("last-observed-at"), field)
	case diagnosticPropertySignalEventFirstOccurrence:
		return evaluationText(fmt.Sprintf("first-occurrence=%d", expectation.integer))
	case diagnosticPropertySignalEventLastOccurrence:
		return evaluationText(fmt.Sprintf("last-occurrence=%d", expectation.integer))
	case diagnosticPropertySignalEventBatchSize:
		return evaluationText(fmt.Sprintf("batch-size=%d", expectation.integer))
	case diagnosticPropertySignalOverflowBoundary:
		return evaluationText(fmt.Sprintf("overflow boundary=%v", expectation.overflowBoundary))
	case diagnosticPropertySignalOverflowEndpointPresent:
		return evaluationText("overflow endpoint identity present")
	case diagnosticPropertySignalOverflowNoEndpoint:
		return evaluationText("overflow endpoint absent")
	case diagnosticPropertySignalOverflowCapacity:
		return evaluationText(fmt.Sprintf("overflow capacity=%d", expectation.integer))
	case diagnosticPropertySignalOverflowBuffered:
		return evaluationText(fmt.Sprintf("overflow buffered=%d", expectation.integer))
	case diagnosticPropertySignalOverflowAbsent:
		return evaluationText("signal-overflow payload absent")
	case diagnosticPropertyNoticeIndex:
		return evaluationText(fmt.Sprintf("notice index=%d", expectation.integer))
	case diagnosticPropertyNoticePhase:
		return evaluationText(fmt.Sprintf("notice phase=%v", expectation.noticePhase))
	case diagnosticPropertyNoticeCode:
		return property(evaluationText("notice code"), quotedTextField(string(expectation.noticeCode)))
	case diagnosticPropertyNoticeCommandPath:
		field := stringCollectionField{
			values: expectation.strings,
		}
		return property(evaluationText("notice command-path"), field)
	default:
		panic("clitest: invalid diagnostic property")
	}
}

func (expectation diagnosticExpectation) match(diagnostic cli.Diagnostic) (bool, evaluationField) {
	switch expectation.property {
	case diagnosticPropertyCode:
		return diagnostic.Code == expectation.code, evaluationText(fmt.Sprint(diagnostic.Code))
	case diagnosticPropertyViolation:
		if diagnostic.Rule == nil {
			return false, observed(semanticSubjectDiagnostic, observationNoRuleFailure)
		}
		return diagnostic.Rule.ViolationKind == expectation.violation, evaluationText(fmt.Sprint(diagnostic.Rule.ViolationKind))
	case diagnosticPropertyProducerKind:
		return diagnostic.ProducerKind == expectation.producerKind, evaluationText(valueProducerKindFailureLabel(diagnostic.ProducerKind))
	case diagnosticPropertyTier:
		return diagnostic.Tier == expectation.tier, evaluationText(fmt.Sprint(diagnostic.Tier))
	case diagnosticPropertyDisposition:
		return diagnostic.Disposition == expectation.disposition, evaluationText(fmt.Sprint(diagnostic.Disposition))
	case diagnosticPropertyExitCode:
		return diagnostic.ExitCode == expectation.integer, evaluationText(strconv.Itoa(diagnostic.ExitCode))
	case diagnosticPropertySensitive:
		return diagnostic.Sensitive, evaluationText(strconv.FormatBool(diagnostic.Sensitive))
	case diagnosticPropertyInputRedacted:
		return diagnostic.InputRedacted, evaluationText(strconv.FormatBool(diagnostic.InputRedacted))
	case diagnosticPropertySpan:
		if diagnostic.Span == nil {
			return false, observed(semanticSubjectDiagnosticSpan, observationAbsent)
		}
		return *diagnostic.Span == expectation.span, evaluationText(fmt.Sprint(*diagnostic.Span))
	case diagnosticPropertySpanPositioned:
		if diagnostic.Span == nil {
			return false, observed(semanticSubjectDiagnosticSpan, observationAbsent)
		}
		if diagnostic.Span.Start.Line > 0 && diagnostic.Span.Start.Column > 0 {
			return true, evaluationText("")
		}
		return false, evaluationText("diagnostic span has no positive line/column position")
	case diagnosticPropertyReason:
		return diagnostic.Reason == expectation.text, evaluationText(diagnostic.Reason)
	case diagnosticPropertyRepairCount:
		return len(diagnostic.RepairIndexes) == expectation.integer, evaluationText(strconv.Itoa(len(diagnostic.RepairIndexes)))
	case diagnosticPropertyProviderOutput:
		if diagnostic.ProviderOutput != nil && *diagnostic.ProviderOutput == expectation.providerOutput {
			return true, evaluationText("")
		}
		redacted := diagnostic.Sensitive || (diagnostic.ProviderOutput != nil && diagnostic.ProviderOutput.Redacted)
		return false, redactedOutcomeField(redacted, evaluationText("provider output differs"))
	case diagnosticPropertySignalActionFailure:
		if diagnostic.SignalAction == nil {
			return false, observed(semanticSubjectStructuredFailureDetail, observationAbsent)
		}
		if equalSignalActionFailure(*diagnostic.SignalAction, expectation.signalActionFailure) {
			return true, evaluationText("")
		}
		return false, observed(semanticSubjectStructuredFailureDetail, observationDifferent)
	case diagnosticPropertySignalOverflowFailure:
		if diagnostic.SignalOverflow == nil {
			return false, observed(semanticSubjectStructuredFailureDetail, observationAbsent)
		}
		if *diagnostic.SignalOverflow == expectation.signalOverflowFailure {
			return true, evaluationText("")
		}
		return false, observed(semanticSubjectStructuredFailureDetail, observationDifferent)
	case diagnosticPropertyNoticeDeliveryFailure:
		if diagnostic.NoticeDelivery == nil {
			return false, observed(semanticSubjectStructuredFailureDetail, observationAbsent)
		}
		actual := *diagnostic.NoticeDelivery
		expected := expectation.noticeDeliveryFailure
		matched := actual.Index == expected.Index && actual.Phase == expected.Phase && actual.Code == expected.Code && slices.Equal(actual.CommandPath, expected.CommandPath)
		if matched {
			return true, evaluationText("")
		}
		return false, observed(semanticSubjectStructuredFailureDetail, observationDifferent)
	case diagnosticPropertyRuleIdentityPresent:
		if diagnostic.Rule == nil {
			return false, observed(semanticSubjectDiagnostic, observationNoRuleFailure)
		}
		if diagnostic.Rule.RuleID != 0 {
			return true, evaluationText("")
		}
		return false, evaluationText("rule identity is zero")
	case diagnosticPropertyRuleKind:
		if diagnostic.Rule == nil {
			return false, observed(semanticSubjectDiagnostic, observationNoRuleFailure)
		}
		return diagnostic.Rule.RuleKind == expectation.ruleKind, evaluationText(fmt.Sprint(diagnostic.Rule.RuleKind))
	case diagnosticPropertyRuleOperandCount:
		if diagnostic.Rule == nil {
			return false, observed(semanticSubjectDiagnostic, observationNoRuleFailure)
		}
		return len(diagnostic.Rule.Operands) == expectation.integer, evaluationText(strconv.Itoa(len(diagnostic.Rule.Operands)))
	case diagnosticPropertyRuleOperandSensitive, diagnosticPropertyRuleOperandNoLiteral:
		operand, ok, detail := resolveDiagnosticRuleOperand(diagnostic, expectation.operand)
		if !ok {
			return false, detail
		}
		if expectation.property == diagnosticPropertyRuleOperandSensitive {
			return operand.Sensitive, evaluationText(fmt.Sprintf("sensitive=%t", operand.Sensitive))
		}
		if operand.Literal == nil {
			return true, evaluationText("")
		}
		return false, evaluationText("literal is present")
	case diagnosticPropertyLiteralType, diagnosticPropertyLiteralText, diagnosticPropertyLiteralSensitive, diagnosticPropertyLiteralRedacted:
		literal, ok, detail := resolveDiagnosticLiteral(diagnostic, expectation.operand)
		if !ok {
			return false, detail
		}
		switch expectation.property {
		case diagnosticPropertyLiteralType:
			field := valueTypeField{
				value: literal.Type,
			}
			return literal.Type == expectation.valueType, property(evaluationText("type"), field)
		case diagnosticPropertyLiteralText:
			if literal.Text == expectation.text {
				return true, evaluationText("")
			}
			if literal.Sensitive || literal.Redacted {
				return false, evaluationText("text=<redacted>")
			}
			return false, property(evaluationText("text"), quotedTextField(literal.Text))
		case diagnosticPropertyLiteralSensitive:
			return literal.Sensitive, evaluationText(fmt.Sprintf("sensitive=%t", literal.Sensitive))
		default:
			return literal.Redacted, evaluationText(fmt.Sprintf("redacted=%t", literal.Redacted))
		}
	case diagnosticPropertySignalActionPolicyIdentityPresent, diagnosticPropertySignalActionIndex, diagnosticPropertySignalActionPhase, diagnosticPropertySignalActionCommandPath, diagnosticPropertySignalActionKind, diagnosticPropertySignalActionName, diagnosticPropertySignalEventSignal, diagnosticPropertySignalEventOrigin, diagnosticPropertySignalEventSender, diagnosticPropertySignalEventNoSender, diagnosticPropertySignalEventPhase, diagnosticPropertySignalEventObservedAt, diagnosticPropertySignalEventFirstObservedAt, diagnosticPropertySignalEventLastObservedAt, diagnosticPropertySignalEventFirstOccurrence, diagnosticPropertySignalEventLastOccurrence, diagnosticPropertySignalEventBatchSize:
		if diagnostic.SignalAction == nil {
			return false, observed(semanticSubjectSignalActionFailure, observationAbsent)
		}
		failure := diagnostic.SignalAction
		switch expectation.property {
		case diagnosticPropertySignalActionPolicyIdentityPresent:
			if failure.PolicyID != 0 {
				return true, evaluationText("")
			}
			return false, evaluationText("policy identity is zero")
		case diagnosticPropertySignalActionIndex:
			return failure.Index == expectation.integer, evaluationText(fmt.Sprintf("index=%d", failure.Index))
		case diagnosticPropertySignalActionPhase:
			return failure.Phase == expectation.signalPhase, evaluationText(fmt.Sprintf("phase=%v", failure.Phase))
		case diagnosticPropertySignalActionCommandPath:
			field := stringCollectionField{
				values: failure.CommandPath,
			}
			return slices.Equal(failure.CommandPath, expectation.strings), property(evaluationText("command-path"), field)
		case diagnosticPropertySignalActionKind:
			return failure.Action.Kind == expectation.signalActionKind, evaluationText(fmt.Sprintf("kind=%v", failure.Action.Kind))
		case diagnosticPropertySignalActionName:
			return failure.Action.Name == expectation.text, property(evaluationText("name"), quotedTextField(failure.Action.Name))
		default:
		}
		event := failure.Event
		switch expectation.property {
		case diagnosticPropertySignalEventSignal:
			return event.Signal == expectation.signal, evaluationText("different signal identity")
		case diagnosticPropertySignalEventOrigin:
			return event.Origin == expectation.signalOrigin, evaluationText(fmt.Sprintf("origin=%v", event.Origin))
		case diagnosticPropertySignalEventSender:
			if event.Sender == nil {
				return false, evaluationText("sender is absent")
			}
			field := signalSenderField{
				value: *event.Sender,
			}
			return *event.Sender == expectation.signalSender, property(evaluationText("sender"), field)
		case diagnosticPropertySignalEventNoSender:
			if event.Sender == nil {
				return true, evaluationText("")
			}
			return false, evaluationText("sender is present")
		case diagnosticPropertySignalEventPhase:
			return event.Phase == expectation.signalPhase, evaluationText(fmt.Sprintf("phase=%v", event.Phase))
		case diagnosticPropertySignalEventObservedAt:
			field := timeField{
				value: event.ObservedAt,
			}
			return event.ObservedAt.Equal(expectation.at), property(evaluationText("observed-at"), field)
		case diagnosticPropertySignalEventFirstObservedAt:
			field := timeField{
				value: event.FirstObservedAt,
			}
			return event.FirstObservedAt.Equal(expectation.at), property(evaluationText("first-observed-at"), field)
		case diagnosticPropertySignalEventLastObservedAt:
			field := timeField{
				value: event.LastObservedAt,
			}
			return event.LastObservedAt.Equal(expectation.at), property(evaluationText("last-observed-at"), field)
		case diagnosticPropertySignalEventFirstOccurrence:
			return event.FirstOccurrence == expectation.integer, evaluationText(fmt.Sprintf("first-occurrence=%d", event.FirstOccurrence))
		case diagnosticPropertySignalEventLastOccurrence:
			return event.LastOccurrence == expectation.integer, evaluationText(fmt.Sprintf("last-occurrence=%d", event.LastOccurrence))
		default:
			return event.BatchSize == expectation.integer, evaluationText(fmt.Sprintf("batch-size=%d", event.BatchSize))
		}
	case diagnosticPropertySignalOverflowBoundary, diagnosticPropertySignalOverflowEndpointPresent, diagnosticPropertySignalOverflowNoEndpoint, diagnosticPropertySignalOverflowCapacity, diagnosticPropertySignalOverflowBuffered:
		if diagnostic.SignalOverflow == nil {
			return false, evaluationText("signal-overflow failure is absent")
		}
		overflow := *diagnostic.SignalOverflow
		switch expectation.property {
		case diagnosticPropertySignalOverflowBoundary:
			return overflow.Boundary == expectation.overflowBoundary, evaluationText(fmt.Sprintf("boundary=%v", overflow.Boundary))
		case diagnosticPropertySignalOverflowEndpointPresent:
			if overflow.EndpointID != 0 {
				return true, evaluationText("")
			}
			return false, evaluationText("endpoint identity is zero")
		case diagnosticPropertySignalOverflowNoEndpoint:
			if overflow.EndpointID == 0 {
				return true, evaluationText("")
			}
			return false, evaluationText("endpoint relation is present")
		case diagnosticPropertySignalOverflowCapacity:
			return overflow.Capacity == expectation.integer, evaluationText(fmt.Sprintf("capacity=%d", overflow.Capacity))
		default:
			return overflow.Buffered == expectation.integer, evaluationText(fmt.Sprintf("buffered=%d", overflow.Buffered))
		}
	case diagnosticPropertySignalOverflowAbsent:
		if diagnostic.SignalOverflow == nil {
			return true, evaluationText("")
		}
		return false, evaluationText(fmt.Sprintf("unexpected signal-overflow payload present (boundary=%v capacity=%d buffered=%d)", diagnostic.SignalOverflow.Boundary, diagnostic.SignalOverflow.Capacity, diagnostic.SignalOverflow.Buffered))
	case diagnosticPropertyNoticeIndex, diagnosticPropertyNoticePhase, diagnosticPropertyNoticeCode, diagnosticPropertyNoticeCommandPath:
		if diagnostic.NoticeDelivery == nil {
			return false, evaluationText("notice-delivery failure is absent")
		}
		notice := *diagnostic.NoticeDelivery
		switch expectation.property {
		case diagnosticPropertyNoticeIndex:
			return notice.Index == expectation.integer, evaluationText(fmt.Sprintf("index=%d", notice.Index))
		case diagnosticPropertyNoticePhase:
			return notice.Phase == expectation.noticePhase, evaluationText(fmt.Sprintf("phase=%v", notice.Phase))
		case diagnosticPropertyNoticeCode:
			return notice.Code == expectation.noticeCode, property(evaluationText("code"), quotedTextField(string(notice.Code)))
		default:
			field := stringCollectionField{
				values: notice.CommandPath,
			}
			return slices.Equal(notice.CommandPath, expectation.strings), property(evaluationText("command-path"), field)
		}
	default:
		panic("clitest: invalid diagnostic property")
	}
}

func (diagnosticCodeCollectionField) evaluationField() {}

func (diagnosticsTruncatedExpectation) completionExpectation() {}

func (diagnosticsTruncatedExpectation) diagnosticReportExpectation() {}

func (diagnosticsTruncatedExpectation) evaluateDiagnosticReport(report cli.DiagnosticReport) evaluationResult {
	if report.Truncated {
		return provedResult()
	}
	return disprovedResult(evaluationText("diagnostic report"), evaluationText("truncated"), evaluationText("complete"))
}

func (diagnosticsTruncatedExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if result.DiagnosticsTruncated {
		return provedResult()
	}
	return disprovedResult(evaluationText("completion diagnostics"), evaluationText("truncated"), evaluationText("complete"))
}

func (diagnosticsExactlyExpectationValue) inspectionExpectation() {}

func (diagnosticsExactlyExpectationValue) completionExpectation() {}

func (diagnosticsExactlyExpectationValue) diagnosticReportExpectation() {}

func (expectation diagnosticsExactlyExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	if inspection.Truncated {
		return notProvableResult(evaluationText("diagnostic report"), evaluationText("exact diagnostic collection"), observed(semanticSubjectDiagnosticReport, observationTruncated))
	}
	if len(inspection.Diagnostics) != len(expectation.locators) {
		return disprovedResult(
			evaluationText("diagnostic report"),
			evaluationText(fmt.Sprintf("exactly %d diagnostics", len(expectation.locators))),
			evaluationText(fmt.Sprintf("%d diagnostics", len(inspection.Diagnostics))),
		)
	}
	for index, state := range expectation.locators {
		resolved := resolveDiagnosticStateForInspection(state, inspection)
		if resolved.status == locatorResolutionNotProvable {
			return notProvableResult(
				evaluationText("diagnostic report"),
				evaluationText("exact diagnostics in authoritative order"),
				resolved.detail,
			)
		}
		if resolved.status != locatorResolutionFound {
			return disprovedResult(
				evaluationText("diagnostic report"),
				evaluationText("exact diagnostics in authoritative order"),
				resolved.detail,
			)
		}
		if resolved.value.index != index {
			return disprovedResult(
				evaluationText("diagnostic report"),
				evaluationText("exact diagnostics in authoritative order"),
				evaluationText(fmt.Sprintf("different diagnostic at index %d", index)),
			)
		}
	}

	return provedResult()
}

func (expectation diagnosticsExactlyExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	report := diagnosticReportFromCompletion(result)

	return expectation.evaluateDiagnosticReport(report)
}

func (expectation diagnosticsExactlyExpectationValue) evaluateDiagnosticReport(report cli.DiagnosticReport) evaluationResult {
	if report.Truncated {
		return notProvableResult(evaluationText("diagnostic report"), evaluationText("exact diagnostic collection"), observed(semanticSubjectDiagnosticReport, observationTruncated))
	}
	if len(report.Diagnostics) != len(expectation.locators) {
		return disprovedResult(
			evaluationText("diagnostic report"),
			evaluationText(fmt.Sprintf("exactly %d diagnostics", len(expectation.locators))),
			evaluationText(fmt.Sprintf("%d diagnostics", len(report.Diagnostics))),
		)
	}
	for index, state := range expectation.locators {
		resolved := resolveDiagnosticStateForReport(state, report)
		if resolved.status == locatorResolutionNotProvable {
			return notProvableResult(
				evaluationText("diagnostic report"),
				evaluationText("exact diagnostics in authoritative order"),
				resolved.detail,
			)
		}
		if resolved.status != locatorResolutionFound {
			return disprovedResult(
				evaluationText("diagnostic report"),
				evaluationText("exact diagnostics in authoritative order"),
				resolved.detail,
			)
		}
		if resolved.value.index != index {
			return disprovedResult(
				evaluationText("diagnostic report"),
				evaluationText("exact diagnostics in authoritative order"),
				evaluationText(fmt.Sprintf("different diagnostic at index %d", index)),
			)
		}
	}

	return provedResult()
}

func (diagnosticInputSourceExpectationValue) inspectionExpectation() {}

func (diagnosticInputSourceExpectationValue) completionExpectation() {}

func (diagnosticInputSourceExpectationValue) diagnosticReportExpectation() {}

func (e diagnosticInputSourceExpectationValue) evaluate(report cli.DiagnosticReport, inspection *cli.Inspection) evaluationResult {
	var diagnostic locatorResolution[diagnosticLocation]
	var source sourceResolution
	if inspection != nil {
		diagnostic = resolveDiagnosticStateForInspection(e.diagnostic, *inspection)
		source = resolveSourceInspection(e.source, *inspection)
	} else {
		diagnostic = resolveDiagnosticStateForReport(e.diagnostic, report)
		source = resolveSource(e.source, report)
	}
	expected := evaluationText("diagnostic input source")
	if diagnostic.status != locatorResolutionFound {
		return outcomeForValueStatus(diagnostic.status, diagnostic.subject, expected, diagnostic.detail)
	}
	if source.status != locatorResolutionFound {
		return outcomeForValueStatus(source.status, source.label, expected, source.actual)
	}
	if diagnostic.value.diagnostic.InputSource == source.value.ID {
		return provedResult()
	}
	return disprovedResult(diagnostic.subject, expected, redactedOutcomeField(diagnostic.value.diagnostic.Sensitive || source.value.Sensitive, evaluationText(fmt.Sprint(diagnostic.value.diagnostic.InputSource))))
}

func (e diagnosticInputSourceExpectationValue) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluate(diagnosticReportFromInspection(i), &i)
}

func (e diagnosticInputSourceExpectationValue) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	return e.evaluate(diagnosticReportFromCompletion(c), nil)
}

func (e diagnosticInputSourceExpectationValue) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluate(r, nil)
}

func (diagnosticOriginsExpectationValue) inspectionExpectation() {}

func (diagnosticOriginsExpectationValue) completionExpectation() {}

func (diagnosticOriginsExpectationValue) diagnosticReportExpectation() {}

func (e diagnosticOriginsExpectationValue) evaluate(report cli.DiagnosticReport, inspection *cli.Inspection) evaluationResult {
	var diagnostic locatorResolution[diagnosticLocation]
	if inspection != nil {
		diagnostic = resolveDiagnosticStateForInspection(e.diagnostic, *inspection)
	} else {
		diagnostic = resolveDiagnosticStateForReport(e.diagnostic, report)
	}
	expectedField := evaluationText("exact diagnostic origins in authoritative order")
	if diagnostic.status != locatorResolutionFound {
		return outcomeForValueStatus(diagnostic.status, diagnostic.subject, expectedField, diagnostic.detail)
	}
	expected, outcome, ok := resolveExpectedOrigins(e.origins, report, inspection)
	if !ok {
		return outcome
	}
	if slices.Equal(diagnostic.value.diagnostic.Origins, expected) {
		return provedResult()
	}
	if diagnostic.value.diagnostic.Sensitive {
		return disprovedResult(diagnostic.subject, evaluationText("exact diagnostic origins in authoritative order"), evaluationText("value differs (redacted)"))
	}
	return orderedCollectionMismatch(diagnostic.subject, "origins", expected, diagnostic.value.diagnostic.Origins)
}

func (e diagnosticOriginsExpectationValue) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluate(diagnosticReportFromInspection(i), &i)
}

func (e diagnosticOriginsExpectationValue) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	return e.evaluate(diagnosticReportFromCompletion(c), nil)
}

func (e diagnosticOriginsExpectationValue) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluate(r, nil)
}

func (diagnosticReplacementExpectationValue) inspectionExpectation() {}

func (diagnosticReplacementExpectationValue) completionExpectation() {}

func (diagnosticReplacementExpectationValue) diagnosticReportExpectation() {}

func (e diagnosticReplacementExpectationValue) evaluate(report cli.DiagnosticReport, inspection *cli.Inspection) evaluationResult {
	var diagnostic locatorResolution[diagnosticLocation]
	if inspection != nil {
		diagnostic = resolveDiagnosticStateForInspection(e.diagnostic, *inspection)
	} else {
		diagnostic = resolveDiagnosticStateForReport(e.diagnostic, report)
	}
	expectedField := evaluationText("diagnostic replacement target")
	if diagnostic.status != locatorResolutionFound {
		return outcomeForValueStatus(diagnostic.status, diagnostic.subject, expectedField, diagnostic.detail)
	}
	var identity cli.DeclarationIdentity
	var ok bool
	if inspection != nil {
		var status locatorResolutionStatus
		var detail evaluationField
		identity, status, detail = resolveExpectedDeclarationIdentity(e.replacement, nil, nil, *inspection)
		if status != locatorResolutionFound {
			return outcomeForValueStatus(status, diagnostic.subject, expectedField, detail)
		}
		ok = true
	} else {
		identity, ok = locatorDeclarationIdentityValue(e.replacement)
	}
	if !ok {
		return disprovedResult(diagnostic.subject, evaluationText("valid replacement declaration"), observed(semanticSubjectDeclaration, observationInvalidLocator))
	}
	if diagnostic.value.diagnostic.Replacement != nil && diagnostic.value.diagnostic.Replacement.Target == identity {
		return provedResult()
	}
	return disprovedResult(diagnostic.subject, evaluationText("expected replacement declaration identity"), redactedOutcomeField(diagnostic.value.diagnostic.Sensitive, evaluationText("replacement differs")))
}

func (e diagnosticReplacementExpectationValue) evaluateInspection(i cli.Inspection, _ error) evaluationResult {
	return e.evaluate(diagnosticReportFromInspection(i), &i)
}

func (e diagnosticReplacementExpectationValue) evaluateCompletion(c cli.Completion, _ error) evaluationResult {
	return e.evaluate(diagnosticReportFromCompletion(c), nil)
}

func (e diagnosticReplacementExpectationValue) evaluateDiagnosticReport(r cli.DiagnosticReport) evaluationResult {
	return e.evaluate(r, nil)
}

func (candidateDiagnosticsExactlyExpectation) inspectionExpectation() {}

func (expectation candidateDiagnosticsExactlyExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveCandidate(expectation.locator, inspection)
	if resolved.status != locatorResolutionFound {
		return outcomeForValueStatus(resolved.status, resolved.label, evaluationText("exact attached diagnostics in public order"), resolved.actual)
	}
	return evaluateDiagnosticIndexesExactly(resolved.label, resolved.value.DiagnosticIndexes, expectation.expected, inspection)
}

func (subject resolvedDiagnosticSubject) matches(diagnostic cli.Diagnostic) bool {
	switch subject.kind {
	case diagnosticSubjectCommandPath:
		return slices.Equal(diagnostic.CommandPath, subject.commandPath)
	case diagnosticSubjectCommand:
		return diagnostic.Command == subject.command
	case diagnosticSubjectDeclaration:
		return diagnostic.Declaration == subject.declaration
	case diagnosticSubjectDefault:
		return diagnostic.Default == subject.defaultID
	case diagnosticSubjectProvider:
		return diagnostic.Provider == subject.providerID
	default:
		return false
	}
}
