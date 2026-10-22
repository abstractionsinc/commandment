package clitest

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/abstractionsinc/commandment/cli"
)

const (
	failureReportMaxOutcomes   = 32
	failureReportMaxFieldBytes = 768
	failureReportMaxBytes      = 32 * 1024
)

const (
	failureReportMaxOutcomeBytes   = 3*failureReportMaxFieldBytes + 512
	failureReportTruncationReserve = 64
	failureReportNestedReserve     = 96
	failureReportNestedItemScale   = 2
)

type boundedReportWriter struct {
	out          strings.Builder
	hardLimit    int
	contentLimit int
	omittedBytes int
}

type failureExpectationValue struct {
	expectations []DiagnosticReportExpectation
}

func reportEvaluationResults(t testing.TB, outcomes []evaluationResult) {
	t.Helper()
	report := formatEvaluationReport(outcomes)
	if report == "" {
		return
	}
	t.Errorf("%s", report)
}

func failEvaluationResults(t testing.TB, outcomes ...evaluationResult) {
	t.Helper()
	reportEvaluationResults(t, outcomes)
	t.FailNow()
}

func formatEvaluationReport(outcomes []evaluationResult) string {
	totalFailures := 0
	for _, outcome := range outcomes {
		switch outcome.status {
		case evaluationProved:
		case evaluationDisproved, evaluationNotProvable:
			totalFailures++
		default:
			panic("clitest: invalid evaluation status")
		}
	}
	if totalFailures == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString("clitest: expectation failed")

	const omissionReserve = 112
	rootBudget := failureReportMaxBytes - omissionReserve
	written := 0
	for _, outcome := range outcomes {
		if outcome.status == evaluationProved {
			continue
		}
		if written == failureReportMaxOutcomes {
			break
		}

		failure := formatEvaluationFailure(written+1, outcome)
		if out.Len()+len(failure) > rootBudget {
			break
		}
		out.WriteString(failure)
		written++
	}

	if omitted := totalFailures - written; omitted > 0 {
		fmt.Fprintf(&out, "\n  ... %d additional failure(s) omitted to keep the report bounded", omitted)
	}
	if out.Len() > failureReportMaxBytes {
		panic("clitest: failure report budget invariant violated")
	}

	return out.String()
}

func boundedReportWriterForLimit(limit int) *boundedReportWriter {
	if limit < 0 {
		limit = 0
	}
	contentLimit := limit
	if contentLimit > failureReportTruncationReserve {
		contentLimit -= failureReportTruncationReserve
	} else {
		contentLimit = 0
	}

	writer := &boundedReportWriter{
		hardLimit:    limit,
		contentLimit: contentLimit,
	}
	return writer
}

func renderEvaluationField(field evaluationField) string {
	writer := boundedReportWriterForLimit(failureReportMaxFieldBytes)
	renderEvaluationFieldInto(writer, field)
	return writer.finish()
}

func renderEvaluationFieldInto(writer *boundedReportWriter, field evaluationField) {
	switch field := field.(type) {
	case nil:
	case evaluationText:
		writer.writeText(string(field))
	case quotedTextField:
		writer.writeLiteral("\"")
		writer.writeText(string(field))
		writer.writeLiteral("\"")
	case evaluationBool:
		if field {
			writer.writeLiteral("true")
		} else {
			writer.writeLiteral("false")
		}
	case releaseLifecycleField:
		renderReleaseLifecycleField(writer, field.value)
	case descriptionField:
		renderDescriptionField(writer, field.value)
	case helpLabelsField:
		renderHelpLabelsField(writer, field.value)
	case valueTypeField:
		renderValueTypeField(writer, field.value)
	case countRangeField:
		renderCountRangeField(writer, field.value)
	case flagSpellingField:
		renderFlagSpellingField(writer, field.value)
	case stringCollectionField:
		renderStringCollectionField(writer, field.values)
	case collectionCardinalityField:
		renderCollectionCardinalityField(writer, field)
	case orderedCollectionField:
		renderOrderedCollectionField(writer, field)
	case declarationIdentityField:
		renderDeclarationIdentityField(writer, field.value)
	case completionRangeCollectionField:
		renderCompletionRangeCollectionField(writer, field.values)
	case runeCollectionField:
		renderRuneCollectionField(writer, field.values)
	case completionMessageField:
		renderCompletionMessageField(writer, field)
	case diagnosticCodeCollectionField:
		renderDiagnosticCodeCollectionField(writer, field.values)
	case resolutionStateCollectionField:
		renderResolutionStateCollectionField(writer, field.values)
	case completionCursorField:
		renderCompletionCursorField(writer, field.value)
	case commandPathField:
		renderCommandPathField(writer, field.values)
	case commandIDField:
		writer.writeLiteral("command id ")
		writer.writeLiteral(strconv.FormatUint(uint64(field.value), 10))
	case inputSpanField:
		renderInputSpanField(writer, field.value)
	case completionSpanField:
		renderCompletionSpanField(writer, field.value)
	case signalSenderField:
		renderSignalSenderField(writer, field.value)
	case timeField:
		writer.writeLiteral(field.value.Format(time.RFC3339Nano))
	case noticeMatchField:
		renderNoticeMatchField(writer, field.value)
	case semanticSubject:
		writer.writeLiteral(semanticSubjectText(field))
	case semanticRequirement:
		writer.writeLiteral(semanticRequirementText(field))
	case semanticObservation:
		writer.writeLiteral(semanticObservationText(field))
	case semanticProperty:
		renderEvaluationFieldInto(writer, field.name)
		writer.writeLiteral("=")
		renderEvaluationFieldInto(writer, field.value)
	case qualifiedSubject:
		if !emptyEvaluationField(field.base) {
			renderEvaluationFieldInto(writer, field.base)
			writer.writeLiteral(" ")
		}
		renderEvaluationFieldInto(writer, field.qualifier)
	case indexedRelationSubject:
		renderEvaluationFieldInto(writer, field.noun)
		writer.writeLiteral(" at index ")
		writer.writeLiteral(strconv.Itoa(field.index))
		writer.writeLiteral(" of ")
		renderEvaluationFieldInto(writer, field.parent)
	case namedSubject:
		renderEvaluationFieldInto(writer, field.base)
		writer.writeLiteral(" \"")
		writer.writeText(field.name)
		writer.writeLiteral("\"")
	case indexedSubjectField:
		renderEvaluationFieldInto(writer, field.base)
		writer.writeLiteral(" at index ")
		writer.writeLiteral(strconv.Itoa(field.index))
	case prefixedDetail:
		writer.writeText(field.prefix)
		renderEvaluationFieldInto(writer, field.detail)
	case comparisonField:
		renderEvaluationFieldInto(writer, field.name)
		writer.writeText(field.separator)
		renderEvaluationFieldInto(writer, field.value)
	case evaluationSummary:
		renderEvaluationSummaryInto(writer, field)
	default:
		panic(fmt.Sprintf("clitest: unsupported evaluation field %T", field))
	}
}

func renderReleaseLifecycleField(writer *boundedReportWriter, value cli.ReleaseLifecycle) {
	writer.writeLiteral("introduced=")
	writer.writeText(value.IntroducedIn)
	writer.writeLiteral("; deprecated=")
	writer.writeText(value.DeprecatedIn)
	writer.writeLiteral("; removed=")
	writer.writeText(value.RemovedIn)
	writer.writeLiteral("; reason=")
	writer.writeText(value.Reason)
}

func renderDescriptionField(writer *boundedReportWriter, value cli.Description) {
	writer.writeLiteral("short=\"")
	writer.writeText(value.Short)
	writer.writeLiteral("\"; long=\"")
	writer.writeText(value.Long)
	writer.writeLiteral("\"")
}

func renderHelpLabelsField(writer *boundedReportWriter, value cli.HelpLabels) {
	labels := [...]struct {
		name string
		text string
	}{
		{
			name: "usage",
			text: value.Usage,
		},
		{
			name: "commands",
			text: value.Commands,
		},
		{
			name: "arguments",
			text: value.Arguments,
		},
		{
			name: "flags",
			text: value.Flags,
		},
		{
			name: "global-flags",
			text: value.GlobalFlags,
		},
		{
			name: "aliases",
			text: value.Aliases,
		},
		{
			name: semanticNameEnvironment,
			text: value.Environment,
		},
		{
			name: "default",
			text: value.Default,
		},
		{
			name: "required",
			text: value.Required,
		},
		{
			name: "optional",
			text: value.Optional,
		},
		{
			name: "deprecated",
			text: value.Deprecated,
		},
	}
	for index, label := range labels {
		if index > 0 {
			writer.writeLiteral("; ")
		}
		writer.writeLiteral(label.name)
		writer.writeLiteral("=")
		writer.writeText(label.text)
	}
}

func renderValueTypeField(writer *boundedReportWriter, value cli.ValueType) {
	writer.writeLiteral("kind=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.Kind), 10))
	writer.writeLiteral("; name=")
	writer.writeText(value.Name)
}

func renderCountRangeField(writer *boundedReportWriter, value cli.CountRange) {
	writer.writeLiteral("minimum=")
	writer.writeLiteral(strconv.Itoa(value.Minimum()))
	if maximum, bounded := value.Maximum(); bounded {
		writer.writeLiteral("; maximum=")
		writer.writeLiteral(strconv.Itoa(maximum))
	} else {
		writer.writeLiteral("; maximum=<unbounded>")
	}
}

func renderFlagSpellingField(writer *boundedReportWriter, value cli.FlagSpelling) {
	writer.writeLiteral("kind=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.Kind), 10))
	writer.writeLiteral("; text=")
	writer.writeText(value.Text)
	writer.writeLiteral("; value-mode=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.ValueMode), 10))
	writer.writeLiteral("; value-forms=[")
	for index, form := range value.ValueForms {
		if index > 0 {
			writer.writeLiteral(",")
		}
		writer.writeLiteral(strconv.FormatUint(uint64(form), 10))
	}
	writer.writeLiteral("]; sensitive=")
	if value.Sensitive {
		writer.writeLiteral("true")
	} else {
		writer.writeLiteral("false")
	}
	writer.writeLiteral("; fixed-value=")
	renderLiteralPresence(writer, value.FixedValue)
	writer.writeLiteral("; implicit-value=")
	renderLiteralPresence(writer, value.ImplicitValue)
}

func renderLiteralPresence(writer *boundedReportWriter, value *cli.ValueLiteral) {
	if value == nil {
		writer.writeLiteral("absent")
		return
	}
	writer.writeLiteral("present(type=")
	renderValueTypeField(writer, value.Type)
	writer.writeLiteral("; sensitive=")
	if value.Sensitive {
		writer.writeLiteral("true")
	} else {
		writer.writeLiteral("false")
	}
	writer.writeLiteral("; redacted=")
	if value.Redacted {
		writer.writeLiteral("true")
	} else {
		writer.writeLiteral("false")
	}
	writer.writeLiteral(")")
}

func renderStringCollectionField(writer *boundedReportWriter, values []string) {
	writer.writeLiteral("[")
	for index, value := range values {
		if index > 0 {
			writer.writeLiteral(", ")
		}
		writer.writeLiteral("\"")
		writer.writeText(value)
		writer.writeLiteral("\"")
	}
	writer.writeLiteral("]")
}

func renderCollectionCardinalityField(writer *boundedReportWriter, field collectionCardinalityField) {
	writer.writeText(field.name)
	writer.writeLiteral("; count=")
	writer.writeLiteral(strconv.Itoa(field.count))
}

func renderOrderedCollectionField(writer *boundedReportWriter, field orderedCollectionField) {
	writer.writeText(field.name)
	if field.expected {
		writer.writeLiteral(" exact ordered collection")
	}
	writer.writeLiteral("; length=")
	writer.writeLiteral(strconv.Itoa(field.length))
	if field.mismatchIndex >= 0 {
		writer.writeLiteral("; index ")
		writer.writeLiteral(strconv.Itoa(field.mismatchIndex))
		writer.writeLiteral("=")
		if field.expected {
			writer.writeLiteral("<expected element>")
		} else {
			writer.writeLiteral("<different observed element>")
		}
	}
}

func renderDeclarationIdentityField(writer *boundedReportWriter, value cli.DeclarationIdentity) {
	writer.writeLiteral("kind=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.Kind), 10))
	writer.writeLiteral("; command=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.Command), 10))
	writer.writeLiteral("; command-entry=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.CommandEntry), 10))
	writer.writeLiteral("; action-entry=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.ActionEntry), 10))
	writer.writeLiteral("; flag=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.Flag), 10))
	writer.writeLiteral("; flag-spelling=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.FlagSpelling), 10))
	writer.writeLiteral("; argument-form=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.ArgumentForm), 10))
	writer.writeLiteral("; argument-term=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.ArgumentTerm), 10))
	writer.writeLiteral("; environment-input=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.EnvironmentInput), 10))
	writer.writeLiteral("; value-domain-entry=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.ValueDomainEntry), 10))
}

func renderCompletionRangeCollectionField(writer *boundedReportWriter, values []cli.CompletionRange) {
	writer.writeLiteral("[")
	for index, value := range values {
		if index > 0 {
			writer.writeLiteral(", ")
		}
		writer.writeLiteral("{")
		writer.writeLiteral(strconv.Itoa(value.Start))
		writer.writeLiteral(":")
		writer.writeLiteral(strconv.Itoa(value.End))
		writer.writeLiteral("}")
	}
	writer.writeLiteral("]")
}

func renderRuneCollectionField(writer *boundedReportWriter, values []rune) {
	writer.writeLiteral("[")
	for index, value := range values {
		if index > 0 {
			writer.writeLiteral(", ")
		}
		writer.writeLiteral(strconv.QuoteRune(value))
	}
	writer.writeLiteral("]")
}

func renderCompletionMessageField(writer *boundedReportWriter, field completionMessageField) {
	writer.writeLiteral("message ")
	writer.writeLiteral(strconv.Itoa(field.index))
	writer.writeLiteral(" {kind=")
	writer.writeLiteral(strconv.FormatUint(uint64(field.value.Kind), 10))
	writer.writeLiteral("; sensitive=")
	if field.value.Sensitive {
		writer.writeLiteral("true; text=<redacted>")
	} else {
		writer.writeLiteral("false; text=")
		writer.writeText(field.value.Text)
	}
	writer.writeLiteral("}")
}

func renderDiagnosticCodeCollectionField(writer *boundedReportWriter, values []cli.DiagnosticCode) {
	writer.writeLiteral("[")
	for index, value := range values {
		if index > 0 {
			writer.writeLiteral(", ")
		}
		writer.writeText(string(value))
	}
	writer.writeLiteral("]")
}

func renderResolutionStateCollectionField(writer *boundedReportWriter, values []cli.ResolutionState) {
	writer.writeLiteral("[")
	for index, value := range values {
		if index > 0 {
			writer.writeLiteral(", ")
		}
		writer.writeLiteral(strconv.FormatUint(uint64(value), 10))
	}
	writer.writeLiteral("]")
}

func renderCompletionCursorField(writer *boundedReportWriter, value cli.CompletionCursor) {
	writer.writeLiteral("argument=")
	writer.writeLiteral(strconv.Itoa(value.Argument))
	writer.writeLiteral("; byte=")
	writer.writeLiteral(strconv.Itoa(value.Byte))
}

func renderCommandPathField(writer *boundedReportWriter, values []string) {
	writer.writeLiteral("command path")
	for _, value := range values {
		writer.writeLiteral(" \"")
		writer.writeText(value)
		writer.writeLiteral("\"")
	}
}

func renderInputSpanField(writer *boundedReportWriter, value cli.InputSpan) {
	writer.writeLiteral("source=")
	if value.Source == (cli.InputSourceID{}) {
		writer.writeLiteral("none")
	} else {
		writer.writeLiteral("present")
	}
	writer.writeLiteral("; entry=")
	writer.writeLiteral(strconv.Itoa(value.Entry))
	writer.writeLiteral("; start={offset=")
	writer.writeLiteral(strconv.Itoa(value.Start.Offset))
	writer.writeLiteral("; line=")
	writer.writeLiteral(strconv.Itoa(value.Start.Line))
	writer.writeLiteral("; column=")
	writer.writeLiteral(strconv.Itoa(value.Start.Column))
	writer.writeLiteral("}; end={offset=")
	writer.writeLiteral(strconv.Itoa(value.End.Offset))
	writer.writeLiteral("; line=")
	writer.writeLiteral(strconv.Itoa(value.End.Line))
	writer.writeLiteral("; column=")
	writer.writeLiteral(strconv.Itoa(value.End.Column))
	writer.writeLiteral("}")
}

func renderCompletionSpanField(writer *boundedReportWriter, value cli.CompletionSpan) {
	writer.writeLiteral("argument=")
	writer.writeLiteral(strconv.Itoa(value.Argument))
	writer.writeLiteral("; start=")
	writer.writeLiteral(strconv.Itoa(value.Start))
	writer.writeLiteral("; end=")
	writer.writeLiteral(strconv.Itoa(value.End))
}

func renderSignalSenderField(writer *boundedReportWriter, value cli.SignalSender) {
	writer.writeLiteral("process-id=")
	writer.writeLiteral(strconv.Itoa(value.ProcessID))
	writer.writeLiteral("; user-id=")
	writer.writeLiteral(strconv.Itoa(value.UserID))
	writer.writeLiteral("; code=")
	writer.writeLiteral(strconv.Itoa(value.Code))
}

func renderNoticeMatchField(writer *boundedReportWriter, value cli.NoticeMatch) {
	writer.writeLiteral("all=")
	writer.writeLiteral(strconv.FormatBool(value.All))
	writer.writeLiteral("; severity=")
	writer.writeLiteral(strconv.FormatUint(uint64(value.Severity), 10))
	writer.writeLiteral("; code=\"")
	writer.writeText(string(value.Code))
	writer.writeLiteral("\"; advisory-code=\"")
	writer.writeText(string(value.AdvisoryCode))
	writer.writeLiteral("\"")
}

func renderEvaluationSummaryInto(writer *boundedReportWriter, summary evaluationSummary) {
	total := 0
	for _, outcome := range summary.outcomes {
		switch outcome.status {
		case evaluationProved:
		case evaluationDisproved, evaluationNotProvable:
			total++
		default:
			panic("clitest: invalid evaluation status in summary")
		}
	}

	written := 0
	for _, outcome := range summary.outcomes {
		if outcome.status == evaluationProved {
			continue
		}
		if writer.remaining() <= failureReportNestedReserve {
			break
		}

		limit := min(writer.remaining()-failureReportNestedReserve, failureReportMaxFieldBytes/failureReportNestedItemScale)
		item := boundedReportWriterForLimit(limit)
		if written > 0 {
			item.writeLiteral("; ")
		}
		renderEvaluationFieldInto(item, outcome.subject)
		item.writeLiteral(": ")
		renderEvaluationFieldInto(item, outcome.observed)
		writer.writeLiteral(item.finish())
		written++
	}

	if omitted := total - written; omitted > 0 {
		if written > 0 {
			writer.writeLiteral("; ")
		}
		writer.writeLiteral(fmt.Sprintf("... %d nested outcome(s) omitted", omitted))
	}
}

func semanticSubjectText(subject semanticSubject) string {
	switch subject {
	case semanticSubjectArgumentForm:
		return "argument form"
	case semanticSubjectDelegation:
		return "delegation"
	case semanticSubjectDeclaration:
		return "declaration"
	case semanticSubjectValue:
		return "value"
	case semanticSubjectFlagSpelling:
		return "flag spelling"
	case semanticSubjectCommandDefault:
		return "command default"
	case semanticSubjectCondition:
		return "condition"
	case semanticSubjectRule:
		return "rule"
	case semanticSubjectAdvisory:
		return "advisory"
	case semanticSubjectNoticeSuppression:
		return "notice suppression"
	case semanticSubjectSignalPolicy:
		return "signal policy"
	case semanticSubjectConditionChild:
		return "condition child"
	case semanticSubjectConditionOperand:
		return "condition operand"
	case semanticSubjectFixedValue:
		return "fixed value"
	case semanticSubjectKeyedEntry:
		return "keyed entry"
	case semanticSubjectRuleOperand:
		return "rule operand"
	case semanticSubjectRuleOperandLiteral:
		return "rule operand literal"
	case semanticSubjectSignalAction:
		return "signal action"
	case semanticSubjectActionEntry:
		return "action entry"
	case semanticSubjectCommandEntry:
		return "command entry"
	case semanticSubjectShortcutDefault:
		return "shortcut default"
	case semanticSubjectShortcutDefaultValue:
		return "shortcut default value"
	case semanticSubjectValueDomainUse:
		return "value-domain use"
	case semanticSubjectTokenLanguage:
		return "token language"
	case semanticSubjectCompletionParent:
		return "completion parent"
	case semanticSubjectCommand:
		return "command"
	case semanticSubjectDiagnostic:
		return "diagnostic"
	case semanticSubjectSignalActionFailure:
		return "signal-action failure"
	case semanticSubjectRepairSource:
		return "repair source"
	case semanticSubjectDiagnosticReport:
		return "diagnostic report"
	case semanticSubjectDiagnosticSpanSource:
		return "diagnostic span source"
	case semanticSubjectDiagnosticSpan:
		return "diagnostic span"
	case semanticSubjectInputSource:
		return "input source"
	case semanticSubjectInputOrigin:
		return "input origin"
	case semanticSubjectHelpCommandGroup:
		return "help command group"
	case semanticSubjectLocator:
		return "locator"
	case semanticSubjectStaticLiteral:
		return "static literal"
	case semanticSubjectHelpFlagGroup:
		return "help flag group"
	case semanticSubjectArgumentTerm:
		return "argument term"
	case semanticSubjectValueDomainEntry:
		return "value-domain entry"
	case semanticSubjectInspection:
		return "inspection"
	case semanticSubjectSignalEndpoint:
		return "signal-endpoint"
	case semanticSubjectRawInput:
		return "raw input"
	case semanticSubjectNotice:
		return semanticLabelNotice
	case semanticSubjectCommandProvider:
		return "command provider"
	case semanticSubjectStructuredFailureDetail:
		return "structured failure detail"
	case semanticSubjectGrammarProfileSet:
		return "grammar profile set"
	case semanticSubjectGrammarProfile:
		return "grammar profile"
	case semanticSubjectFlag:
		return "flag"
	case semanticSubjectEnvironmentInput:
		return "environment input"
	case semanticSubjectValueDomain:
		return "value domain"
	case semanticSubjectParentCommand:
		return "parent command"
	case semanticSubjectChildCommand:
		return "child command"
	case semanticSubjectFact:
		return "fact"
	case semanticSubjectRelatedSignalEndpoint:
		return "related signal endpoint"
	case semanticSubjectOrigin:
		return "origin"
	default:
		return ""
	}
}

func semanticRequirementText(requirement semanticRequirement) string {
	switch requirement {
	case semanticRequirementSensitive:
		return "sensitive"
	case semanticRequirementAbsent:
		return "absent"
	case semanticRequirementRequestedReplacement:
		return "requested replacement"
	case semanticRequirementNoCanonicalRepresentation:
		return "no canonical representation"
	case semanticRequirementExactCanonicalTokens:
		return "exact canonical tokens"
	case semanticRequirementExactOrigins:
		return "exact origins"
	default:
		return ""
	}
}

func semanticObservationText(observation semanticObservation) string {
	subject := semanticSubjectText(observation.subject)
	switch observation.condition {
	case observationAbsent:
		return subject + " is absent"
	case observationInvalidLocator:
		if observation.subject == semanticSubjectLocator {
			return "invalid locator"
		}
		return "invalid " + semanticLocatorSubjectText(observation.subject) + " locator"
	case observationInvalidOrForeignLocator:
		return "invalid or foreign " + semanticLocatorSubjectText(observation.subject) + " locator"
	case observationLocatorDoesNotIdentify:
		return "locator does not identify a " + subject
	case observationTruncated:
		return subject + " is truncated"
	case observationNotDeclaration:
		return subject + " is not a declaration"
	case observationMustBeFlagOrArgumentTerm:
		return subject + " must be flag or argument term"
	case observationNoRuleFailure:
		return subject + " has no Rule failure"
	case observationAbsentOrAmbiguous:
		return subject + " is absent or ambiguous"
	case observationDifferent:
		return "different " + subject
	case observationNoSignalEndpointRelation:
		return subject + " has no signal-endpoint relation"
	case observationInvalidLocatorInExpectedCollection:
		return "invalid " + subject + " locator in expected collection"
	default:
		return ""
	}
}

func semanticLocatorSubjectText(subject semanticSubject) string {
	switch subject {
	case semanticSubjectArgumentForm:
		return "argument-form"
	case semanticSubjectCommandEntry:
		return "command-entry"
	case semanticSubjectEnvironmentInput:
		return semanticNameEnvironment
	case semanticSubjectFlagSpelling:
		return "flag-spelling"
	default:
		return semanticSubjectText(subject)
	}
}

func formatEvaluationFailure(index int, outcome evaluationResult) string {
	if outcome.status != evaluationDisproved && outcome.status != evaluationNotProvable {
		panic("clitest: invalid failure outcome")
	}

	subject := renderEvaluationField(outcome.subject)
	if subject == "" {
		subject = "expectation subject"
	}
	expected := renderEvaluationField(outcome.expected)
	if expected == "" {
		expected = "specified expectation"
	}
	observed := renderEvaluationField(outcome.observed)
	if observed == "" {
		observed = "no additional detail"
	}

	writer := boundedReportWriterForLimit(failureReportMaxOutcomeBytes)
	writer.writeLiteral("\n  ")
	writer.writeLiteral(strconv.Itoa(index))
	writer.writeLiteral(". ")
	writer.writeLiteral(subject)
	writer.writeLiteral("\n     expected: ")
	writer.writeLiteral(expected)
	if outcome.status == evaluationNotProvable {
		writer.writeLiteral("\n     not provable: ")
	} else {
		writer.writeLiteral("\n     observed: ")
	}
	writer.writeLiteral(observed)
	return writer.finish()
}

func orderedCollectionMismatch[T comparable](subject evaluationField, name string, expected, observed []T) evaluationResult {
	limit := min(len(expected), len(observed))
	for index := range limit {
		if expected[index] == observed[index] {
			continue
		}
		expectedField := orderedCollectionField{
			name:          name,
			length:        len(expected),
			mismatchIndex: index,
			expected:      true,
		}
		observedField := orderedCollectionField{
			name:          name,
			length:        len(observed),
			mismatchIndex: index,
		}
		return disprovedResult(subject, expectedField, observedField)
	}
	expectedField := orderedCollectionField{
		name:          name,
		length:        len(expected),
		mismatchIndex: -1,
		expected:      true,
	}
	observedField := orderedCollectionField{
		name:          name,
		length:        len(observed),
		mismatchIndex: -1,
	}
	if limit > 0 {
		observedField.name = fmt.Sprintf("%s; common prefix length=%d", name, limit)
	}
	return disprovedResult(subject, expectedField, observedField)
}

func encodeReportRune(firstByte byte, r rune, size int) string {
	if r == utf8.RuneError && size == 1 {
		return fmt.Sprintf("\\x%02x", firstByte)
	}
	switch r {
	case '\n':
		return `\n`
	case '\r':
		return `\r`
	case '\t':
		return `\t`
	case '\x1b':
		return `\x1b`
	}
	if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
		if r <= unicode.MaxLatin1 {
			return fmt.Sprintf("\\x%02x", r)
		}
		return fmt.Sprintf("\\u%04x", r)
	}
	return string(r)
}

func failureExpectation(expectations []DiagnosticReportExpectation) FailureExpectation {
	value := failureExpectationValue{
		expectations: slices.Clone(expectations),
	}
	return value
}

func (writer *boundedReportWriter) remaining() int {
	if writer == nil || writer.contentLimit <= writer.out.Len() {
		return 0
	}
	return writer.contentLimit - writer.out.Len()
}

func (writer *boundedReportWriter) writeLiteral(text string) {
	if writer == nil || text == "" {
		return
	}

	remaining := writer.remaining()
	if remaining == 0 {
		writer.omittedBytes += len(text)
		return
	}
	if len(text) <= remaining {
		writer.out.WriteString(text)
		return
	}

	writer.out.WriteString(text[:remaining])
	writer.omittedBytes += len(text) - remaining
}

func (writer *boundedReportWriter) writeText(text string) {
	if writer == nil || text == "" {
		return
	}

	for offset := 0; offset < len(text); {
		r, size := utf8.DecodeRuneInString(text[offset:])
		encoded := encodeReportRune(text[offset], r, size)
		writer.writeLiteral(encoded)
		offset += size
	}
}

func (writer *boundedReportWriter) finish() string {
	if writer == nil {
		return ""
	}
	if writer.omittedBytes == 0 {
		return writer.out.String()
	}

	marker := fmt.Sprintf("... <%d rendered bytes omitted>", writer.omittedBytes)
	if writer.out.Len()+len(marker) > writer.hardLimit {
		available := writer.hardLimit - writer.out.Len()
		if available <= 0 {
			return writer.out.String()
		}
		if len(marker) > available {
			marker = marker[:available]
		}
	}
	writer.out.WriteString(marker)
	return writer.out.String()
}

func (failureExpectationValue) inspectionExpectation() {}

func (failureExpectationValue) completionExpectation() {}

func (failureExpectationValue) referenceExpectation() {}

func (failureExpectationValue) expectsOperationFailure() {}

func (expectation failureExpectationValue) evaluateInspection(_ cli.Inspection, err error) evaluationResult {
	return expectation.evaluateFailure("inspection", err)
}

func (expectation failureExpectationValue) evaluateCompletion(_ cli.Completion, err error) evaluationResult {
	return expectation.evaluateFailure("completion", err)
}

func (expectation failureExpectationValue) evaluateReference(_ cli.Reference, err error) evaluationResult {
	return expectation.evaluateFailure("reference", err)
}

func (expectation failureExpectationValue) evaluateFailure(operation string, err error) evaluationResult {
	if err == nil {
		return disprovedResult(evaluationText(operation), evaluationText("Commandment-owned structured failure"), evaluationText("operation succeeded"))
	}
	if len(expectation.expectations) == 0 {
		return disprovedResult(
			evaluationText(operation),
			evaluationText("structured diagnostic expectations for expected failure"),
			evaluationText("FailsWith has no diagnostic expectations"),
		)
	}

	report := cli.Diagnostics(err)
	if len(report.Diagnostics) == 0 {
		return disprovedResult(
			evaluationText(operation),
			evaluationText("Commandment-owned structured diagnostic failure"),
			evaluationText("returned error exposes no diagnostics"),
		)
	}

	outcomes := make([]evaluationResult, 0, len(expectation.expectations))
	for _, nested := range expectation.expectations {
		if nested == nil {
			outcomes = append(outcomes, disprovedResult(evaluationText("diagnostic report"), evaluationText("valid expectation"), evaluationText("nil expectation")))
			continue
		}
		evaluator, ok := nested.(diagnosticEvaluator)
		if !ok {
			panic(fmt.Sprintf("clitest: invalid DiagnosticReportExpectation implementation %T", nested))
		}
		outcomes = append(outcomes, evaluator.evaluateDiagnosticReport(report))
	}

	status := evaluationProved
	for _, outcome := range outcomes {
		if outcome.status == evaluationDisproved {
			status = evaluationDisproved
			break
		}
		if outcome.status == evaluationNotProvable {
			status = evaluationNotProvable
		}
	}
	if status == evaluationProved {
		return provedResult()
	}
	actual := summarizeOutcomes(outcomes)
	if status == evaluationNotProvable {
		return notProvableResult(evaluationText(operation), evaluationText("expected structured failure diagnostics"), actual)
	}
	return disprovedResult(evaluationText(operation), evaluationText("expected structured failure diagnostics"), actual)
}
