package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	diagnosticDebugLimit       = 16
	diagnosticDebugDetailLimit = 40
	diagnosticDebugTextLimit   = 240
	diagnosticDebugListLimit   = 6
	diagnosticDebugUnknown     = "unknown"
)

// writeDiagnosticDebug writes the bounded deterministic developer/debug
// projection used only by %+v on exact Commandment-owned Diagnostic errors.
// It deliberately does not select terminal-dependent presentation.
func writeDiagnosticDebug(writer io.Writer, err error) bool {
	report := Diagnostics(err)
	if len(report.Diagnostics) == 0 {
		return false
	}
	_, writeErr := io.WriteString(writer, renderDiagnosticDebug(report))
	return writeErr == nil
}

func renderDiagnosticDebug(report DiagnosticReport) string {
	var out strings.Builder
	limit := min(len(report.Diagnostics), diagnosticDebugLimit)
	for index, diagnostic := range report.Diagnostics[:limit] {
		if index != 0 {
			out.WriteByte('\n')
		}
		title := "Commandment diagnostic"
		if len(report.Diagnostics) > 1 {
			title += " " + strconv.Itoa(index+1)
		}
		if diagnostic.Code != "" {
			title += " [" + debugText(string(diagnostic.Code)) + "]"
		}
		out.WriteString(title)
		out.WriteByte('\n')
		for _, detail := range diagnosticDebugDetails(diagnostic) {
			out.WriteString("  ")
			out.WriteString(detail)
			out.WriteByte('\n')
		}
	}
	if omitted := len(report.Diagnostics) - limit; omitted > 0 {
		fmt.Fprintf(&out, "%d additional diagnostics omitted\n", omitted)
	}
	if report.Truncated {
		out.WriteString("analysis truncated\n")
	}
	return out.String()
}

func diagnosticDebugDetails(diagnostic Diagnostic) []string {
	details := make([]string, 0, diagnosticDebugDetailLimit)
	add := func(label, value string) {
		if len(details) >= diagnosticDebugDetailLimit {
			return
		}
		value = debugText(value)
		if value == "" {
			return
		}
		details = append(details, label+": "+value)
	}
	add("command id", debugID(uint64(diagnostic.Command)))
	if len(diagnostic.CommandPath) != 0 {
		add("command", strings.Join(diagnostic.CommandPath, " "))
	}
	if diagnostic.Declaration.Kind != DeclarationKindUnknown {
		add("declaration", debugDeclaration(diagnostic.Declaration))
	}
	if diagnostic.Subject != "" {
		add("subject", diagnostic.Subject)
	}
	add("provider", debugID(uint64(diagnostic.Provider)))
	add("default", debugID(uint64(diagnostic.Default)))
	if diagnostic.AdvisoryCode != "" {
		add("advisory code", string(diagnostic.AdvisoryCode))
	}
	if diagnostic.Sensitive { //nolint:nestif // One sensitivity gate keeps all secret-bearing debug fields behind the same branch.
		add("input", "<redacted>")
	} else {
		if diagnostic.Message != "" {
			add("message", diagnostic.Message)
		}
		if diagnostic.Reason != "" {
			add("reason", diagnostic.Reason)
		}
		if diagnostic.InputRedacted {
			add("input", "<redacted>")
		} else if diagnostic.Input != "" {
			add("input", diagnostic.Input)
		}
	}
	if diagnostic.Rejection != nil {
		add("rejection code", string(diagnostic.Rejection.Code))
		if diagnostic.Sensitive {
			add("rejection expected", "<redacted>")
		} else if len(diagnostic.Rejection.Expected) != 0 {
			add("rejection expected", debugList(diagnostic.Rejection.Expected))
		}
		if len(diagnostic.Rejection.Spans) != 0 {
			add("rejection spans", strconv.Itoa(len(diagnostic.Rejection.Spans)))
		}
	}
	if diagnostic.Rule != nil {
		add("rule", diagnosticDebugRule(diagnostic.Rule, diagnostic.Sensitive))
	}
	if diagnostic.ProviderOutput != nil {
		add("provider output", diagnosticDebugProviderOutput(diagnostic.ProviderOutput, diagnostic.Sensitive))
	}
	if diagnostic.Replacement != nil {
		add("replacement", debugDeclaration(diagnostic.Replacement.Target))
	}
	if diagnostic.InputSource.value != 0 {
		add("input source", debugID(uint64(diagnostic.InputSource.value)))
	}
	if diagnostic.Span != nil {
		add("span", debugSpan(*diagnostic.Span))
	}
	if len(diagnostic.Origins) != 0 {
		ids := make([]string, 0, min(len(diagnostic.Origins), diagnosticDebugListLimit))
		for _, id := range diagnostic.Origins[:min(len(diagnostic.Origins), diagnosticDebugListLimit)] {
			ids = append(ids, debugID(uint64(id.value)))
		}
		add("origins", strings.Join(ids, ", "))
	}
	if diagnostic.ProducerKind != ValueProducerKindUnknown {
		add("producer kind", string(projectDiagnosticProducerKind(diagnostic.ProducerKind)))
	}
	if diagnostic.Tier != ResolutionTierUnknown {
		add("resolution tier", string(projectResolutionTier(diagnostic.Tier)))
	}
	if diagnostic.Disposition != ValueCandidateDispositionUnknown {
		add("candidate disposition", string(projectCandidateDisposition(diagnostic.Disposition)))
	}
	if diagnostic.ExitCode != 0 {
		add("exit code", strconv.Itoa(diagnostic.ExitCode))
	}
	if len(diagnostic.Suggestions) != 0 && !diagnostic.Sensitive {
		add("suggestions", debugList(diagnostic.Suggestions))
	}
	if diagnostic.Hint != "" && !diagnostic.Sensitive {
		add("hint", diagnostic.Hint)
	}
	if len(diagnostic.Usage) != 0 && !diagnostic.Sensitive {
		add("usage", debugList(diagnostic.Usage))
	}
	if diagnostic.SignalAction != nil {
		add("signal action", string(diagnostic.SignalAction.Action.Kind))
	}
	if diagnostic.SignalOverflow != nil {
		add("signal overflow", string(diagnostic.SignalOverflow.Boundary))
	}
	if diagnostic.NoticeDelivery != nil {
		add("notice delivery", string(diagnostic.NoticeDelivery.Code))
	}
	if len(diagnostic.RepairIndexes) != 0 {
		values := make([]string, 0, min(len(diagnostic.RepairIndexes), diagnosticDebugListLimit))
		for _, index := range diagnostic.RepairIndexes[:min(len(diagnostic.RepairIndexes), diagnosticDebugListLimit)] {
			values = append(values, strconv.Itoa(index))
		}
		add("repairs", strings.Join(values, ", "))
	}
	if len(details) == 0 {
		details = append(details, "details: no additional structured fields")
	}
	return details
}

func diagnosticDebugRule(rule *RuleFailure, sensitive bool) string {
	value := string(diagnosticProjectRuleKind(rule.RuleKind))
	if rule.RuleID != 0 {
		value += " #" + strconv.FormatUint(uint64(rule.RuleID), 10)
	}
	if rule.RuleCode != "" {
		value += " (code " + debugText(string(rule.RuleCode)) + ")"
	}
	if sensitive {
		return value + " <redacted>"
	}
	if rule.ViolationKind != ViolationUnknown {
		value += ", violation " + string(diagnosticProjectViolationKind(rule.ViolationKind))
	}
	return value
}

func diagnosticDebugProviderOutput(output *CommandProviderOutputLocator, sensitive bool) string {
	value := fmt.Sprintf("entry %d (%s)", output.Index, debugCommandEntryKind(output.Kind))
	if sensitive || output.Redacted {
		return value + " <redacted>"
	}
	if output.Key != "" {
		value += ", key " + debugText(output.Key)
	}
	if output.Location != "" {
		value += ", location " + debugText(output.Location)
	}
	return value
}

func debugCommandEntryKind(kind CommandEntryKind) string {
	switch kind {
	case CommandEntryCanonical:
		return "canonical"
	case CommandEntryAlias:
		return "alias"
	case CommandEntryShortcut:
		return "shortcut"
	case CommandEntryUnknown:
		return diagnosticDebugUnknown
	default:
		return diagnosticDebugUnknown
	}
}

func debugDeclaration(identity DeclarationIdentity) string {
	var kind string
	var id uint64
	switch identity.Kind {
	case DeclarationKindCommandEntry:
		kind, id = "command-entry", uint64(identity.CommandEntry)
	case DeclarationKindActionEntry:
		kind, id = "action-entry", uint64(identity.ActionEntry)
	case DeclarationKindFlag:
		kind, id = "flag", uint64(identity.Flag)
	case DeclarationKindFlagSpelling:
		kind, id = "flag-spelling", uint64(identity.FlagSpelling)
	case DeclarationKindArgumentForm:
		kind, id = "argument-form", uint64(identity.ArgumentForm)
	case DeclarationKindArgumentTerm:
		kind, id = "argument-term", uint64(identity.ArgumentTerm)
	case DeclarationKindEnvironmentInput:
		kind, id = "environment-input", uint64(identity.EnvironmentInput)
	case DeclarationKindValueDomainEntry:
		kind, id = "value-domain-entry", uint64(identity.ValueDomainEntry)
	case DeclarationKindCommand:
		kind, id = "command", uint64(identity.Command)
	case DeclarationKindUnknown:
		return ""
	default:
		return diagnosticDebugUnknown
	}
	if id != 0 {
		kind += " #" + strconv.FormatUint(id, 10)
	}
	return kind
}

func debugSpan(span InputSpan) string {
	return fmt.Sprintf("source %d entry %d offsets %d..%d", span.Source.value, span.Entry, span.Start.Offset, span.End.Offset)
}

func debugList(values []string) string {
	limit := min(len(values), diagnosticDebugListLimit)
	result := make([]string, 0, limit+1)
	for _, value := range values[:limit] {
		result = append(result, debugText(value))
	}
	if omitted := len(values) - limit; omitted > 0 {
		result = append(result, fmt.Sprintf("+%d more", omitted))
	}
	return strings.Join(result, ", ")
}

func debugID(value uint64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatUint(value, 10)
}

func debugText(text string) string {
	var result strings.Builder
	count := 0
	for _, value := range text {
		if count == diagnosticDebugTextLimit {
			result.WriteRune('…')
			break
		}
		count++
		if !unicode.IsControl(value) {
			result.WriteRune(value)
			continue
		}
		if value <= unicode.MaxLatin1 {
			fmt.Fprintf(&result, `\x%02x`, value)
			continue
		}
		fmt.Fprintf(&result, `\u%04x`, value)
	}
	return result.String()
}
