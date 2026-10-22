package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const unknownDiagnosticText = "unknown"

const (
	diagnosticPresentationLimit       = 16
	diagnosticPresentationDetailLimit = 40
	diagnosticPresentationTextLimit   = 240
	diagnosticPresentationListLimit   = 6
)

// writeDiagnosticPresentation writes the canonical Commandment-owned human
// presentation for every structured Diagnostic reachable through err. It
// returns false when err carries no valid Commandment Diagnostic payload.
// The caller retains ownership of the destination and all process policy.
func writeDiagnosticPresentation(writer io.Writer, err error) bool {
	report, extractErr := extract(err)
	if extractErr != nil || len(report.Diagnostics) == 0 {
		return false
	}
	text, renderErr := renderDiagnosticReport(report, defaultProfile(destinationStderrDiagnostic))
	if renderErr != nil {
		return false
	}
	_, _ = io.WriteString(writer, text)
	return true
}

func renderDiagnosticReport(report report, profile profile) (string, error) {
	document := document{
		Destination: destinationStderrDiagnostic,
	}
	limit := min(len(report.Diagnostics), diagnosticPresentationLimit)
	for index := range limit {
		diagnostic := report.Diagnostics[index]
		title := report.ResolveFrameworkText("Commandment diagnostic")
		if len(report.Diagnostics) > 1 {
			title += " " + strconv.Itoa(index+1)
		}
		if code := boundedDiagnosticText(diagnostic.Code); code != "" {
			title += " [" + code + "]"
		}
		details := diagnosticPresentationDetails(report, diagnostic)
		blockValue := block{
			Kind:  blockDiagnosticGroup,
			Title: title,
			Items: details,
		}
		document.Blocks = append(document.Blocks, blockValue)
	}
	if omitted := len(report.Diagnostics) - limit; omitted > 0 {
		document.Blocks = append(document.Blocks, block{
			Kind: blockHint,
			Spans: []presentationSpan{
				{
					Text: fmt.Sprintf(
						report.ResolveFrameworkText(
							"Presentation limit reached: %d additional Commandment diagnostics are omitted from this human view.",
						),
						omitted,
					),
				},
			},
		})
	}
	if report.Truncated {
		document.Blocks = append(document.Blocks, block{
			Kind: blockHint,
			Spans: []presentationSpan{
				{
					Text: report.ResolveFrameworkText(
						"Analysis was truncated; additional eligible diagnostic or repair work may be absent from the structured report.",
					),
					Role: spanWarning,
				},
			},
		})
	}
	return render(context.Background(), document, profile)
}

type diagnosticDetailSink struct {
	report  report
	items   []string
	omitted int
}

func (sink *diagnosticDetailSink) framework(source string) string {
	return sink.report.ResolveFrameworkText(source)
}

func (sink *diagnosticDetailSink) format(source string, operands ...any) string {
	return fmt.Sprintf(sink.framework(source), operands...)
}

func (sink *diagnosticDetailSink) add(label, value string) {
	value = boundedDiagnosticText(value)
	if value == "" {
		return
	}
	if len(sink.items) >= diagnosticPresentationDetailLimit-1 {
		sink.omitted++
		return
	}
	if label != "" {
		value = sink.framework(label) + ": " + value
	}
	sink.items = append(sink.items, value)
}

func (sink *diagnosticDetailSink) finish() []string {
	if sink.omitted > 0 {
		sink.items = append(
			sink.items,
			sink.format("presentation: %d additional detail items omitted", sink.omitted),
		)
	}
	return sink.items
}

func diagnosticPresentationDetails(report report, diagnostic diagnostic) []string {
	sink := diagnosticDetailSink{
		report: report,
	}

	if diagnostic.Command.present() {
		sink.add("command id", identityText(diagnostic.Command.uint32()))
	}
	if len(diagnostic.CommandPath) != 0 {
		sink.add("command", boundedPath(&sink, diagnostic.CommandPath))
	}
	if diagnostic.Declaration.Kind != "" {
		sink.add("declaration", diagnosticDeclarationText(&sink, diagnostic.Declaration))
	}
	if diagnostic.Subject != "" && !diagnostic.Sensitive {
		sink.add("subject", diagnostic.Subject)
	}
	if diagnostic.Provider.present() {
		sink.add("provider", identityText(diagnostic.Provider.uint32()))
	}
	if diagnostic.Default.present() {
		sink.add("default", identityText(diagnostic.Default.uint32()))
	}
	if diagnostic.AdvisoryCode != "" {
		sink.add("advisory code", diagnostic.AdvisoryCode)
	}

	if diagnostic.Message != "" && !diagnostic.Sensitive {
		message := diagnostic.Message
		if diagnostic.FrameworkMessage {
			message = sink.framework(message)
		}
		sink.add("message", message)
	}
	if diagnostic.Reason != "" && !diagnostic.Sensitive {
		reason := diagnostic.Reason
		if diagnostic.FrameworkReason {
			reason = sink.framework(reason)
		}
		sink.add("reason", reason)
	}

	if diagnostic.Rule != nil {
		addRulePresentation(&sink, diagnostic.Sensitive, *diagnostic.Rule)
	}
	if diagnostic.ProviderOutput != nil {
		addProviderOutputPresentation(&sink, diagnostic.Sensitive, *diagnostic.ProviderOutput)
	}
	if diagnostic.Replacement != nil {
		addReplacementPresentation(&sink, diagnostic.Sensitive, *diagnostic.Replacement)
	}

	if diagnostic.InputRedacted || diagnostic.Sensitive {
		sink.add("input", sink.framework("<redacted>"))
	} else if diagnostic.Input != "" {
		sink.add("input", diagnostic.Input)
	} else if input, ok := diagnosticInputExcerpt(report, diagnostic); ok {
		sink.add("input", input)
	}
	if diagnostic.ProducerKind != "" {
		sink.add("producer kind", sink.framework(string(diagnostic.ProducerKind)))
	}
	if diagnostic.Tier != "" {
		sink.add("resolution tier", sink.framework(string(diagnostic.Tier)))
	}
	if diagnostic.Disposition != "" {
		sink.add("candidate disposition", sink.framework(string(diagnostic.Disposition)))
	}
	if diagnostic.ExitCode != 0 {
		sink.add("exit code", strconv.Itoa(diagnostic.ExitCode))
	}

	addSourcePresentation(&sink, report, diagnostic)
	addOriginsPresentation(&sink, report, diagnostic.Origins)
	addSuggestionsPresentation(&sink, diagnostic)
	addRepairsPresentation(&sink, report, diagnostic)
	addSignalPresentation(&sink, diagnostic)
	addNoticePresentation(&sink, diagnostic)

	if diagnostic.Hint != "" && !diagnostic.Sensitive {
		hint := diagnostic.Hint
		if diagnostic.FrameworkHint {
			hint = sink.framework(hint)
		}
		sink.add("hint", hint)
	}
	if len(sink.items) == 0 {
		sink.add("details", sink.framework("no additional structured presentation fields were available"))
	}
	return sink.finish()
}

func addRulePresentation(sink *diagnosticDetailSink, diagnosticSensitive bool, rule analysisDiagnosticRuleFailure) {
	identity := sink.framework(string(rule.Kind))
	if rule.ID != 0 {
		identity += " #" + strconv.FormatUint(uint64(rule.ID), 10)
	}
	if rule.Code != "" {
		identity += sink.format(" (code %s)", boundedDiagnosticText(rule.Code))
	}
	sink.add("rule", identity)
	if rule.ViolationKind != "" {
		sink.add("violation", sink.framework(string(rule.ViolationKind)))
	}
	if len(rule.Subjects) != 0 {
		values := make([]string, 0, min(len(rule.Subjects), diagnosticPresentationListLimit))
		for _, subject := range rule.Subjects[:min(len(rule.Subjects), diagnosticPresentationListLimit)] {
			values = append(values, sink.format("value #%d", subject))
		}
		if omitted := len(rule.Subjects) - len(values); omitted > 0 {
			values = append(values, sink.format("+%d more", omitted))
		}
		sink.add("rule subjects", strings.Join(values, ", "))
	}
	if len(rule.Operands) != 0 {
		values := make([]string, 0, min(len(rule.Operands), diagnosticPresentationListLimit))
		for _, operand := range rule.Operands[:min(len(rule.Operands), diagnosticPresentationListLimit)] {
			values = append(values, ruleOperandText(sink, operand, diagnosticSensitive))
		}
		if omitted := len(rule.Operands) - len(values); omitted > 0 {
			values = append(values, sink.format("+%d more", omitted))
		}
		sink.add("rule operands", strings.Join(values, ", "))
	}
	if rule.Pattern != "" {
		if diagnosticSensitive {
			sink.add("pattern", sink.framework("<redacted>"))
		} else {
			sink.add("pattern", rule.Pattern)
		}
	}
	if rule.CountValid {
		sink.add("count", diagnosticCountText(sink, rule))
	}
}

func ruleOperandText(sink *diagnosticDetailSink, operand diagnosticRuleOperand, diagnosticSensitive bool) string {
	if diagnosticSensitive || operand.Sensitive ||
		operand.Literal != nil && (operand.Literal.Sensitive || operand.Literal.Redacted) {
		return sink.framework("<redacted>")
	}
	if operand.Literal == nil {
		return sink.framework("<unrepresentable>")
	}
	kind := sink.framework(operand.Literal.Type.Kind)
	if operand.Literal.Type.Name != "" {
		kind = operand.Literal.Type.Name
	}
	text := boundedDiagnosticText(operand.Literal.Text)
	if text == "" {
		return kind
	}
	return kind + " " + strconv.Quote(text)
}

func diagnosticCountText(sink *diagnosticDetailSink, rule analysisDiagnosticRuleFailure) string {
	if rule.CountUnbounded {
		return sink.format("%d or more", rule.CountMinimum)
	}
	if rule.CountMinimum == rule.CountMaximum {
		return strconv.Itoa(rule.CountMinimum)
	}
	return fmt.Sprintf("%d..%d", rule.CountMinimum, rule.CountMaximum)
}

func addProviderOutputPresentation(sink *diagnosticDetailSink, diagnosticSensitive bool, output diagnosticProviderOutput) {
	value := sink.format("entry %d (%s)", output.Index, sink.framework(commandEntryKindText(output.Kind)))
	if output.Redacted || diagnosticSensitive {
		value += " " + sink.framework("<redacted>")
	} else {
		if output.Key != "" {
			value += sink.format(", key %s", boundedDiagnosticText(output.Key))
		}
		if output.Location != "" {
			value += sink.format(", location %s", boundedDiagnosticText(output.Location))
		}
	}
	sink.add("provider output", value)
}

func addReplacementPresentation(sink *diagnosticDetailSink, diagnosticSensitive bool, replacement diagnosticReplacement) {
	value := diagnosticDeclarationText(sink, replacement.Target)
	if replacement.Presentation != "" {
		if diagnosticSensitive {
			value += sink.format(" as %s", sink.framework("<redacted>"))
		} else {
			value += sink.format(" as %s", boundedDiagnosticText(replacement.Presentation))
		}
	}
	if replacement.TargetDeprecatedDuringMigration {
		value += sink.framework(" (target also deprecated during migration)")
	}
	sink.add("replacement", value)
}

func addSourcePresentation(sink *diagnosticDetailSink, report report, diagnostic diagnostic) {
	if diagnostic.Source == 0 && diagnostic.Span == nil {
		return
	}
	id := diagnostic.Source
	if id == 0 && diagnostic.Span != nil {
		id = diagnostic.Span.Source
	}
	source, ok := report.Sources[id]
	switch {
	case !ok:
		sink.add("source", sink.format("#%d (snapshot unavailable)", id))
	default:
		value := fmt.Sprintf("#%d %s", id, sink.framework(sourceKindText(source.Kind)))
		if source.Sensitive || diagnostic.Sensitive {
			value += " " + sink.framework("<protected metadata>")
		} else {
			if source.Key != "" {
				value += sink.format(", key %s", boundedDiagnosticText(source.Key))
			}
			if source.Location != "" {
				value += sink.format(", location %s", boundedDiagnosticText(source.Location))
			}
		}
		sink.add("source", value)
	}
	if diagnostic.Span != nil {
		sink.add("span", diagnosticSpanText(sink, *diagnostic.Span))
	}
}

func diagnosticInputExcerpt(report report, diagnostic diagnostic) (string, bool) {
	if diagnostic.Sensitive {
		return "", false
	}
	if diagnostic.Span != nil {
		if text, ok := safeSpanText(report, *diagnostic.Span, false); ok {
			return text, true
		}
	}
	for _, id := range diagnostic.Origins {
		origin, ok := report.Origins[id]
		if !ok || origin.Sensitive {
			continue
		}
		if text, ok := safeSpanText(report, origin.Span, false); ok {
			return text, true
		}
	}
	return "", false
}

func safeSpanText(report report, span span, allowSensitive bool) (string, bool) {
	source, ok := report.Sources[span.Source]
	if !ok || source.Sensitive && !allowSensitive {
		return "", false
	}
	if span.Start.Offset < 0 || span.End.Offset < span.Start.Offset {
		return "", false
	}
	if span.Entry == -1 {
		if source.FlatText == nil || span.End.Offset > len(*source.FlatText) {
			return "", false
		}
		return boundedDiagnosticText((*source.FlatText)[span.Start.Offset:span.End.Offset]), true
	}
	if span.Entry < 0 || span.Entry >= len(source.Entries) || source.FlatText != nil ||
		span.End.Offset > len(source.Entries[span.Entry]) {
		return "", false
	}
	return boundedDiagnosticText(source.Entries[span.Entry][span.Start.Offset:span.End.Offset]), true
}

func addOriginsPresentation(sink *diagnosticDetailSink, report report, ids []originID) {
	limit := min(len(ids), diagnosticPresentationListLimit)
	for index := range limit {
		id := ids[index]
		origin, ok := report.Origins[id]
		if !ok {
			sink.add("origin", sink.format("#%d (snapshot unavailable)", id))
			continue
		}
		if origin.Sensitive {
			sink.add("origin", sink.format("#%d %s", id, sink.framework("<protected provenance>")))
			continue
		}
		value := sink.format("#%d at %s", id, diagnosticSpanText(sink, origin.Span))
		if len(origin.Parents) != 0 {
			parents := make([]string, 0, min(len(origin.Parents), diagnosticPresentationListLimit))
			for _, parent := range origin.Parents[:min(len(origin.Parents), diagnosticPresentationListLimit)] {
				parents = append(parents, "#"+strconv.FormatUint(uint64(parent), 10))
			}
			if omitted := len(origin.Parents) - len(parents); omitted > 0 {
				parents = append(parents, sink.format("+%d more", omitted))
			}
			value += sink.format(", parents %s", strings.Join(parents, ", "))
		}
		sink.add("origin", value)
	}
	if omitted := len(ids) - limit; omitted > 0 {
		sink.add("origins", sink.format("%d additional origins omitted", omitted))
	}
}

func addSuggestionsPresentation(sink *diagnosticDetailSink, diagnostic diagnostic) {
	if diagnostic.Sensitive {
		if len(diagnostic.Suggestions) != 0 || len(diagnostic.Usage) != 0 {
			sink.add("suggestions", sink.framework("<redacted>"))
		}
		return
	}
	for index, suggestion := range diagnostic.Suggestions[:min(len(diagnostic.Suggestions), diagnosticPresentationListLimit)] {
		sink.add(sink.format("suggestion %d", index+1), suggestion)
	}
	if omitted := len(
		diagnostic.Suggestions,
	) - min(
		len(diagnostic.Suggestions),
		diagnosticPresentationListLimit,
	); omitted > 0 {
		sink.add("suggestions", sink.format("%d additional suggestions omitted", omitted))
	}
	for index, usage := range diagnostic.Usage[:min(len(diagnostic.Usage), diagnosticPresentationListLimit)] {
		sink.add(sink.format("usage %d", index+1), usage)
	}
	if omitted := len(diagnostic.Usage) - min(len(diagnostic.Usage), diagnosticPresentationListLimit); omitted > 0 {
		sink.add("usage", sink.format("%d additional usage forms omitted", omitted))
	}
}

func addRepairsPresentation(sink *diagnosticDetailSink, report report, diagnostic diagnostic) {
	limit := min(len(diagnostic.RepairIndexes), diagnosticPresentationListLimit)
	for index := range limit {
		repairIndex := diagnostic.RepairIndexes[index]
		if repairIndex < 0 || repairIndex >= len(report.Repairs) {
			sink.add("repair", sink.format("#%d unavailable", repairIndex))
			continue
		}
		repair := report.Repairs[repairIndex]
		if repair.Sensitive || diagnostic.Sensitive {
			sink.add("repair", sink.format("#%d %s", repairIndex, sink.framework("<redacted>")))
			continue
		}
		value := "#" + strconv.Itoa(repairIndex)
		if repair.Title != "" {
			value += " " + boundedDiagnosticText(repair.Title)
		}
		if repair.PreservesMeaning {
			value += sink.framework(" (preserves meaning)")
		}
		sink.add("repair", value)
		for sourceIndex, sourceRepair := range repair.Sources[:min(len(repair.Sources), diagnosticPresentationListLimit)] {
			addRepairSourcePresentation(sink, report, repairIndex, sourceIndex, sourceRepair)
		}
		if omitted := len(repair.Sources) - min(len(repair.Sources), diagnosticPresentationListLimit); omitted > 0 {
			sink.add(
				"repair sources",
				sink.format("repair #%d has %d additional source plans omitted", repairIndex, omitted),
			)
		}
	}
	if omitted := len(diagnostic.RepairIndexes) - limit; omitted > 0 {
		sink.add("repairs", sink.format("%d additional repair alternatives omitted", omitted))
	}
}

func addRepairSourcePresentation(sink *diagnosticDetailSink, report report, repairIndex, sourceIndex int, sourceRepair repairSource) {
	protected := false
	if source, ok := report.Sources[sourceRepair.Source]; ok && source.Sensitive {
		protected = true
	}
	value := sink.format(
		"repair #%d source %d (#%d): %d text edits, %d entry edits",
		repairIndex,
		sourceIndex+1,
		sourceRepair.Source,
		len(sourceRepair.TextEdits),
		len(sourceRepair.EntryEdits),
	)
	if protected {
		value += " " + sink.framework("<protected>")
	}
	sink.add("repair plan", value)
	if protected {
		return
	}
	for index, edit := range sourceRepair.TextEdits[:min(len(sourceRepair.TextEdits), diagnosticPresentationListLimit)] {
		sink.add(
			sink.format("text edit %d", index+1),
			diagnosticSpanText(sink, edit.Span)+" -> "+replacementText(sink, report, edit.Replacement),
		)
	}
	for index, edit := range sourceRepair.EntryEdits[:min(len(sourceRepair.EntryEdits), diagnosticPresentationListLimit)] {
		entries := make([]string, 0, min(len(edit.Entries), diagnosticPresentationListLimit))
		for _, replacement := range edit.Entries[:min(len(edit.Entries), diagnosticPresentationListLimit)] {
			entries = append(entries, replacementText(sink, report, replacement))
		}
		if omitted := len(edit.Entries) - len(entries); omitted > 0 {
			entries = append(entries, sink.format("+%d more", omitted))
		}
		sink.add(
			sink.format("entry edit %d", index+1),
			sink.format("entries %d..%d -> %s", edit.Start, edit.End, strings.Join(entries, ", ")),
		)
	}
}

func replacementText(sink *diagnosticDetailSink, report report, replacement replacement) string {
	parts := make([]string, 0, min(len(replacement), diagnosticPresentationListLimit))
	for _, part := range replacement[:min(len(replacement), diagnosticPresentationListLimit)] {
		switch {
		case part.Sensitive:
			parts = append(parts, sink.framework("<redacted>"))
		case part.Literal != nil:
			parts = append(parts, strconv.Quote(boundedDiagnosticText(*part.Literal)))
		case part.Copy != nil:
			if source, ok := report.Sources[part.Copy.Source]; ok && source.Sensitive {
				parts = append(parts, sink.framework("<copy from protected source>"))
			} else {
				parts = append(parts, sink.format("copy %s", diagnosticSpanText(sink, *part.Copy)))
			}
		default:
			parts = append(parts, sink.framework("<invalid replacement part>"))
		}
	}
	if omitted := len(replacement) - len(parts); omitted > 0 {
		parts = append(parts, sink.format("+%d more", omitted))
	}
	return strings.Join(parts, " + ")
}

func signalIdentityText(sink *diagnosticDetailSink, signal Signal) string {
	return sink.format("declared-signal-%d", signal.id)
}

func addSignalActionPresentation(sink *diagnosticDetailSink, diagnostic diagnostic, failure *diagnosticSignalActionFailure) {
	value := sink.format(
		"policy #%d action %d %s",
		failure.PolicyID.uint32(),
		failure.Index,
		sink.framework(string(failure.Action.Kind)),
	)
	if failure.Action.Name != "" && !failure.Action.Sensitive && !diagnostic.Sensitive {
		value += " " + boundedDiagnosticText(failure.Action.Name)
	}
	value += sink.format(", phase %s", sink.framework(signalPhaseText(failure.Phase)))
	sink.add("signal action", value)
	if len(failure.CommandPath) != 0 {
		sink.add("signal command", boundedPath(sink, failure.CommandPath))
	}
	if failure.Action.EndpointID.present() {
		sink.add("signal endpoint", identityText(failure.Action.EndpointID.uint32()))
	}
	if failure.Action.SignalRedacted || failure.Action.Sensitive || diagnostic.Sensitive {
		sink.add("action signal", sink.framework("<redacted>"))
	} else if failure.Action.Signal.valid() {
		sink.add("action signal", signalIdentityText(sink, failure.Action.Signal))
	}
	if failure.Action.Delayed {
		if failure.Action.DelayRedacted || failure.Action.Sensitive || diagnostic.Sensitive {
			sink.add("action delay", sink.framework("<redacted>"))
		} else {
			sink.add("action delay", failure.Action.Delay.String())
		}
	}
	if failure.Action.ExitCode != 0 {
		sink.add("action exit code", strconv.Itoa(failure.Action.ExitCode))
	}
	if failure.Event.Signal.valid() && !failure.Action.Sensitive && !diagnostic.Sensitive {
		sink.add("observed signal", signalIdentityText(sink, failure.Event.Signal))
	}
	sink.add("signal origin", sink.framework(signalOriginText(failure.Event.Origin)))
	sink.add("signal event phase", sink.framework(signalPhaseText(failure.Event.Phase)))
	if failure.Event.BatchSize != 0 || failure.Event.FirstOccurrence != 0 || failure.Event.LastOccurrence != 0 {
		sink.add(
			"signal occurrences",
			sink.format(
				"first %d, last %d, batch %d",
				failure.Event.FirstOccurrence,
				failure.Event.LastOccurrence,
				failure.Event.BatchSize,
			),
		)
	}
	if failure.Event.Sender != nil && !failure.Action.Sensitive && !diagnostic.Sensitive {
		sink.add(
			"signal sender",
			sink.format(
				"pid %d, uid %d, code %d",
				failure.Event.Sender.ProcessID,
				failure.Event.Sender.UserID,
				failure.Event.Sender.Code,
			),
		)
	}
}

func addSignalPresentation(sink *diagnosticDetailSink, diagnostic diagnostic) {
	if failure := diagnostic.SignalAction; failure != nil {
		addSignalActionPresentation(sink, diagnostic, failure)
	}
	if overflow := diagnostic.SignalOverflow; overflow != nil {
		value := sink.format(
			"%s boundary, capacity %d, buffered %d",
			sink.framework(string(overflow.Boundary)),
			overflow.Capacity,
			overflow.Buffered,
		)
		if overflow.EndpointID.present() {
			value += sink.format(", endpoint #%d", overflow.EndpointID.uint32())
		}
		sink.add("signal overflow", value)
	}
}

func addNoticePresentation(sink *diagnosticDetailSink, diagnostic diagnostic) {
	if failure := diagnostic.NoticeDelivery; failure != nil {
		value := sink.format(
			"attempt %d, %s, notice code %s",
			failure.Index,
			sink.framework(string(failure.Phase)),
			boundedDiagnosticText(failure.Code),
		)
		sink.add("notice delivery", value)
		if len(failure.CommandPath) != 0 {
			sink.add("notice command", boundedPath(sink, failure.CommandPath))
		}
	}
}

func diagnosticDeclarationText(sink *diagnosticDetailSink, identity analysisDiagnosticDeclarationIdentity) string {
	var id uint32
	switch identity.Kind {
	case diagnosticDeclarationCommandEntry:
		id = identity.CommandEntry.uint32()
	case diagnosticDeclarationActionEntry:
		id = identity.ActionEntry.uint32()
	case diagnosticDeclarationFlag:
		id = identity.Flag.uint32()
	case diagnosticDeclarationFlagSpelling:
		id = identity.FlagSpelling.uint32()
	case diagnosticDeclarationArgumentForm:
		id = identity.ArgumentForm.uint32()
	case diagnosticDeclarationArgumentTerm:
		id = identity.ArgumentTerm.uint32()
	case diagnosticDeclarationEnvironmentInput:
		id = identity.EnvironmentInput.uint32()
	case diagnosticDeclarationValueDomainEntry:
		id = identity.ValueDomainEntry.uint32()
	}
	if id == 0 {
		return sink.framework(string(identity.Kind))
	}
	return sink.format("%s #%d", sink.framework(string(identity.Kind)), id)
}

func diagnosticSpanText(sink *diagnosticDetailSink, span span) string {
	value := sink.format("source #%d", span.Source)
	if span.Entry >= 0 {
		value += sink.format(" entry %d", span.Entry)
	}
	value += sink.format(" bytes %d..%d", span.Start.Offset, span.End.Offset)
	if span.Start.Line > 0 && span.Start.Column > 0 {
		value += sink.format(" (line %d:%d", span.Start.Line, span.Start.Column)
		if span.End.Line > 0 && span.End.Column > 0 {
			value += fmt.Sprintf("..%d:%d", span.End.Line, span.End.Column)
		}
		value += ")"
	}
	return value
}

func sourceKindText(kind InputKind) string {
	switch kind {
	case InputKindArguments:
		return "arguments"
	case InputKindEnvironment:
		return "environment"
	case InputKindShortcut:
		return "shortcut"
	case InputKindResponseFile:
		return "response file"
	case InputKindGenerated:
		return "generated"
	default:
		return unknownDiagnosticText
	}
}

func commandEntryKindText(kind CommandEntryKind) string {
	switch kind {
	case CommandEntryCanonical:
		return "canonical"
	case CommandEntryAlias:
		return "alias"
	case CommandEntryShortcut:
		return "shortcut"
	default:
		return unknownDiagnosticText
	}
}

func signalPhaseText(phase SignalPhase) string {
	switch phase {
	case SignalPhaseResolution:
		return "resolution"
	case SignalPhaseExecution:
		return "execution"
	case SignalPhaseCleanup:
		return "cleanup"
	default:
		return unknownDiagnosticText
	}
}

func signalOriginText(origin SignalOrigin) string {
	switch origin {
	case SignalOriginTerminal:
		return "terminal"
	case SignalOriginProcess:
		return "process"
	case SignalOriginKernel:
		return "kernel"
	case SignalOriginChild:
		return "child"
	case SignalOriginRuntime:
		return "runtime"
	default:
		return unknownDiagnosticText
	}
}

func identityText(id uint32) string { return "#" + strconv.FormatUint(uint64(id), 10) }

func boundedPath(sink *diagnosticDetailSink, path []string) string {
	limit := min(len(path), diagnosticPresentationListLimit)
	parts := make([]string, 0, limit+1)
	for _, part := range path[:limit] {
		parts = append(parts, boundedDiagnosticText(part))
	}
	if omitted := len(path) - limit; omitted > 0 {
		parts = append(parts, sink.format("…(+%d)", omitted))
	}
	return strings.Join(parts, " ")
}

func boundedDiagnosticText(text string) string {
	count := 0
	for index := range text {
		if count == diagnosticPresentationTextLimit {
			return text[:index] + "…"
		}
		count++
	}
	return text
}
