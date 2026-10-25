package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	diagnosticPresentationLimit      = 16
	diagnosticPresentationTextLimit  = 240
	diagnosticPresentationListLimit  = 6
	diagnosticPresentationUsageLimit = 2
	ruleBetweenOperandCount          = 2
)

var (
	errNilErrorWriter                = errors.New("cli: nil error writer")
	errMissingDiagnosticPresentation = errors.New("cli: Commandment presentation owner has no diagnostics")
)

// WriteError writes err to writer using Commandment's canonical human Diagnostic
// presentation only when err owns Commandment presentation. Otherwise it preserves
// the error's ordinary %v / Error() presentation. It returns rendering or publication
// failure and never owns process exit.
func WriteError(writer io.Writer, err error) error {
	if err == nil {
		return nil
	}
	if writer == nil {
		return errNilErrorWriter
	}

	commandment, owner := errorPresentationOwner(err)
	if commandment {
		return writeDiagnosticPresentation(writer, err)
	}
	_, writeErr := fmt.Fprintf(writer, "%v\n", owner)
	if writeErr != nil {
		return writeErr //nolint:wrapcheck // WriteError preserves the exact destination failure identity.
	}
	return nil
}

func errorPresentationOwner(err error) (bool, error) {
	switch value := err.(type) { //nolint:errorlint // Presentation ownership depends on the exact top-level private owner.
	case *diagnosticFailureError:
		return true, err
	case *orderedError:
		if len(value.errors) == 0 || value.errors[0] == nil {
			return false, err
		}
		if ownsDiagnosticPresentation(value.errors[0]) {
			return true, err
		}
		return false, value.errors[0]
	default:
		return false, err
	}
}

func writeDiagnosticPresentation(writer io.Writer, err error) error {
	report, extractErr := extract(err)
	if extractErr != nil {
		return extractErr
	}
	if len(report.Diagnostics) == 0 {
		return errMissingDiagnosticPresentation
	}
	text, renderErr := renderDiagnosticReport(report, profileForWriter(writer, destinationStderrDiagnostic))
	if renderErr != nil {
		return renderErr
	}
	text = strings.TrimRight(text, "\n") + "\n"
	written, writeErr := io.WriteString(writer, text)
	if writeErr != nil {
		return writeErr //nolint:wrapcheck // WriteError preserves the exact destination failure identity.
	}
	if written != len(text) {
		return io.ErrShortWrite
	}
	return nil
}

func renderDiagnosticReport(report report, profile profile) (string, error) {
	document := document{
		Destination: destinationStderrDiagnostic,
	}
	limit := min(len(report.Diagnostics), diagnosticPresentationLimit)
	diagnostics := report.Diagnostics[:limit]
	for index := 0; index < len(diagnostics); {
		diagnostic := diagnostics[index]
		if DiagnosticCode(diagnostic.Code) == DiagnosticActionConflict {
			end := index + 1
			for end < len(diagnostics) && DiagnosticCode(diagnostics[end].Code) == DiagnosticActionConflict {
				end++
			}
			document.Blocks = append(document.Blocks, actionConflictPresentationBlock(report, diagnostics[index:end]))
			index = end
			continue
		}
		document.Blocks = append(document.Blocks, diagnosticPresentationBlock(report, diagnostic))
		index++
	}
	if omitted := len(report.Diagnostics) - limit; omitted > 0 {
		omittedBlock := block{
			Kind: blockHint,
			Spans: []presentationSpan{{
				Text: fmt.Sprintf(
					report.ResolveFrameworkText("%d additional diagnostics were omitted from this human view."),
					omitted,
				),
			}},
		}
		document.Blocks = append(document.Blocks, omittedBlock)
	}
	if report.Truncated {
		truncatedBlock := block{
			Kind: blockHint,
			Spans: []presentationSpan{{
				Text: report.ResolveFrameworkText("Analysis was truncated; additional diagnostic information may be unavailable."),
				Role: spanWarning,
			}},
		}
		document.Blocks = append(document.Blocks, truncatedBlock)
	}
	return render(context.Background(), document, profile)
}

func actionConflictPresentationBlock(report report, diagnostics []diagnostic) block {
	type participant struct {
		subject string
		count   int
	}

	participants := make([]participant, 0, len(diagnostics))
	participantIndex := make(map[ActionEntryID]int, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Declaration.Kind != diagnosticDeclarationActionEntry || !diagnostic.Declaration.ActionEntry.present() {
			continue
		}
		id := diagnostic.Declaration.ActionEntry
		index, ok := participantIndex[id]
		if !ok {
			index = len(participants)
			participantIndex[id] = index
			participants = append(participants, participant{})
		}
		participants[index].count++
		if participants[index].subject != "" {
			continue
		}
		subject := diagnosticSubject(diagnostic)
		if subject == "" {
			subject = diagnosticInputText(report, diagnostic)
		}
		participants[index].subject = subject
	}

	problem := report.ResolveFrameworkText("conflicting actions")
	switch {
	case len(participants) > 1:
		subjects := make([]string, 0, min(len(participants), diagnosticPresentationListLimit))
		seen := make(map[string]struct{}, len(participants))
		for _, participant := range participants {
			if participant.subject == "" {
				continue
			}
			if _, duplicate := seen[participant.subject]; duplicate {
				continue
			}
			seen[participant.subject] = struct{}{}
			subjects = append(subjects, participant.subject)
			if len(subjects) == diagnosticPresentationListLimit {
				break
			}
		}
		if len(subjects) > 1 {
			problem = fmt.Sprintf(
				report.ResolveFrameworkText("conflicting actions: %s"),
				strings.Join(subjects, " and "),
			)
		}
	case len(participants) == 1 && participants[0].count > 1 && participants[0].subject != "":
		problem = fmt.Sprintf(
			report.ResolveFrameworkText("action %s was specified more than once"),
			participants[0].subject,
		)
	}

	prefix := boundedDiagnosticText(report.ApplicationName())
	if prefix == "" {
		prefix = "commandment"
	}
	result := block{
		Kind:  blockDiagnosticGroup,
		Title: prefix + ": " + problem,
	}
	return result
}

func diagnosticPresentationBlock(report report, diagnostic diagnostic) block {
	problem := diagnosticProblem(report, diagnostic)
	prefix := boundedDiagnosticText(report.ApplicationName())
	if prefix == "" {
		prefix = "commandment"
	}
	title := prefix + ": " + problem
	if diagnosticShowsCode(diagnostic) && diagnostic.Code != "" {
		title += " [" + boundedDiagnosticText(diagnostic.Code) + "]"
	}
	value := block{
		Kind:  blockDiagnosticGroup,
		Title: title,
		Items: diagnosticPresentationItems(report, diagnostic),
	}
	return value
}

func diagnosticProblem(report report, diagnostic diagnostic) string {
	if diagnostic.Sensitive {
		return sensitiveDiagnosticProblem(report, diagnostic)
	}
	if diagnostic.Code == string(DiagnosticRuleViolation) && diagnostic.Rule != nil {
		return ruleDiagnosticProblem(report, diagnostic)
	}
	if problem, ok := grammarDiagnosticProblem(report, diagnostic); ok {
		return problem
	}
	if diagnostic.Code == string(DiagnosticInvalidValue) {
		return invalidValueProblem(report, diagnostic)
	}
	if diagnostic.Message != "" {
		message := diagnostic.Message
		if diagnostic.FrameworkMessage {
			message = report.ResolveFrameworkText(message)
		}
		return boundedDiagnosticText(message)
	}
	return diagnosticCodeProblem(report, diagnostic.Code)
}

func sensitiveDiagnosticProblem(report report, diagnostic diagnostic) string {
	subject := diagnosticSubject(diagnostic)
	switch DiagnosticCode(diagnostic.Code) {
	case DiagnosticInvalidValue:
		if subject != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("invalid value for %s"), subject)
		}
		return report.ResolveFrameworkText("invalid value")
	case DiagnosticRuleViolation:
		if subject != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("%s is invalid"), subject)
		}
		return report.ResolveFrameworkText("input is invalid")
	default:
		return diagnosticCodeProblem(report, diagnostic.Code)
	}
}

func grammarDiagnosticProblem(report report, diagnostic diagnostic) (string, bool) {
	input := diagnosticInputText(report, diagnostic)
	subject := diagnosticSubject(diagnostic)
	switch DiagnosticCode(diagnostic.Code) {
	case DiagnosticUnknownCommand:
		if input != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("unknown command %s"), strconv.Quote(input)), true
		}
		return report.ResolveFrameworkText("unknown command"), true
	case DiagnosticAmbiguousCommand:
		if input != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("ambiguous command %s"), strconv.Quote(input)), true
		}
		return report.ResolveFrameworkText("ambiguous command"), true
	case DiagnosticUnknownFlag:
		if input != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("unknown flag %s"), input), true
		}
		return report.ResolveFrameworkText("unknown flag"), true
	case DiagnosticAmbiguousFlag:
		if input != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("ambiguous flag %s"), input), true
		}
		return report.ResolveFrameworkText("ambiguous flag"), true
	case DiagnosticMisplacedFlag:
		if input != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("flag %s is not allowed here"), input), true
		}
		return report.ResolveFrameworkText("flag is not allowed here"), true
	case DiagnosticRepeatedFlag:
		name := subject
		if name == "" {
			name = input
		}
		if name != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("flag %s has an invalid number of occurrences"), name), true
		}
		return report.ResolveFrameworkText("flag has an invalid number of occurrences"), true
	case DiagnosticMissingValue:
		switch {
		case strings.HasPrefix(subject, "-"):
			return fmt.Sprintf(report.ResolveFrameworkText("flag %s requires a value"), subject), true
		case strings.HasPrefix(subject, "<"):
			return fmt.Sprintf(report.ResolveFrameworkText("missing required argument %s"), subject), true
		case subject != "":
			return fmt.Sprintf(report.ResolveFrameworkText("%s is required"), subject), true
		default:
			return report.ResolveFrameworkText("missing required value"), true
		}
	case DiagnosticUnexpectedArgument:
		if input != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("unexpected argument %s"), strconv.Quote(input)), true
		}
		return report.ResolveFrameworkText("unexpected argument"), true
	case DiagnosticAmbiguousArgumentForm:
		return report.ResolveFrameworkText("arguments match more than one valid form"), true
	case DiagnosticInvalidArgumentSyntax:
		return report.ResolveFrameworkText("invalid argument syntax"), true
	case DiagnosticActionConflict:
		return report.ResolveFrameworkText("conflicting actions"), true
	case DiagnosticNotIntroducedCommand:
		return lifecycleDiagnosticProblem(report, "command", input, subject, "is not available in this release"), true
	case DiagnosticRemovedCommand:
		return lifecycleDiagnosticProblem(report, "command", input, subject, "was removed"), true
	case DiagnosticUnavailableCommand:
		return lifecycleDiagnosticProblem(report, "command", input, subject, "is unavailable"), true
	case DiagnosticNotIntroducedFlag:
		return lifecycleDiagnosticProblem(report, "flag", input, subject, "is not available in this release"), true
	case DiagnosticRemovedFlag:
		return lifecycleDiagnosticProblem(report, "flag", input, subject, "was removed"), true
	case DiagnosticUnavailableFlag:
		return lifecycleDiagnosticProblem(report, "flag", input, subject, "is unavailable"), true
	case DiagnosticNotIntroducedArgument:
		return lifecycleDiagnosticProblem(report, "argument", input, subject, "is not available in this release"), true
	case DiagnosticRemovedArgument:
		return lifecycleDiagnosticProblem(report, "argument", input, subject, "was removed"), true
	case DiagnosticUnavailableArgument:
		return lifecycleDiagnosticProblem(report, "argument", input, subject, "is unavailable"), true
	case DiagnosticNotRunnableCommand:
		return report.ResolveFrameworkText("selected command is not runnable"), true
	default:
		return "", false
	}
}

func lifecycleDiagnosticProblem(report report, kind, input, subject, state string) string {
	name := subject
	if name == "" {
		name = input
	}
	if name == "" {
		return report.ResolveFrameworkText(kind + " " + state)
	}
	if kind == "command" || kind == "argument" {
		name = strconv.Quote(name)
	}
	return fmt.Sprintf(report.ResolveFrameworkText("%s %s %s"), kind, name, state)
}

func invalidValueProblem(report report, diagnostic diagnostic) string {
	if diagnostic.Rejection != nil && diagnostic.Message != "" {
		return boundedDiagnosticText(diagnostic.Message)
	}
	input := diagnosticInputText(report, diagnostic)
	subject := diagnosticSubject(diagnostic)
	var problem string
	switch {
	case input != "" && subject != "":
		problem = fmt.Sprintf(report.ResolveFrameworkText("invalid value %s for %s"), strconv.Quote(input), subject)
	case input != "":
		problem = fmt.Sprintf(report.ResolveFrameworkText("invalid value %s"), strconv.Quote(input))
	case subject != "":
		problem = fmt.Sprintf(report.ResolveFrameworkText("invalid value for %s"), subject)
	default:
		problem = report.ResolveFrameworkText("invalid value")
	}
	if len(diagnostic.Expected) == 1 {
		problem += ": " + fmt.Sprintf(report.ResolveFrameworkText("expected %s"), diagnostic.Expected[0])
	}
	return problem
}

func ruleDiagnosticProblem(report report, diagnostic diagnostic) string {
	subject := diagnosticSubject(diagnostic)
	if subject == "" {
		subject = report.ResolveFrameworkText("value")
	}
	rule := diagnostic.Rule
	operands := rulePresentationOperands(diagnostic.Sensitive, *rule)
	switch rule.Kind {
	case diagnosticRuleRequire:
		return fmt.Sprintf(report.ResolveFrameworkText("%s is required"), subject)
	case diagnosticRuleForbid:
		return fmt.Sprintf(report.ResolveFrameworkText("%s is not allowed"), subject)
	case diagnosticRuleNotEmpty:
		return fmt.Sprintf(report.ResolveFrameworkText("%s must not be empty"), subject)
	case diagnosticRuleBetween:
		if len(operands) >= ruleBetweenOperandCount {
			return fmt.Sprintf(report.ResolveFrameworkText("%s must be between %s and %s"), subject, operands[0], operands[1])
		}
	case diagnosticRuleAtLeast:
		if len(operands) != 0 {
			return fmt.Sprintf(report.ResolveFrameworkText("%s must be at least %s"), subject, operands[0])
		}
	case diagnosticRuleAtMost:
		if len(operands) != 0 {
			return fmt.Sprintf(report.ResolveFrameworkText("%s must be at most %s"), subject, operands[0])
		}
	case diagnosticRuleGreaterThan:
		if len(operands) != 0 {
			return fmt.Sprintf(report.ResolveFrameworkText("%s must be greater than %s"), subject, operands[0])
		}
	case diagnosticRuleLessThan:
		if len(operands) != 0 {
			return fmt.Sprintf(report.ResolveFrameworkText("%s must be less than %s"), subject, operands[0])
		}
	case diagnosticRuleEqual:
		if len(operands) != 0 {
			return fmt.Sprintf(report.ResolveFrameworkText("%s must equal %s"), subject, operands[0])
		}
	case diagnosticRuleNotEqual:
		if len(operands) != 0 {
			return fmt.Sprintf(report.ResolveFrameworkText("%s must not equal %s"), subject, operands[0])
		}
	case diagnosticRuleMatch:
		if rule.Pattern != "" {
			return fmt.Sprintf(report.ResolveFrameworkText("%s has an invalid format"), subject)
		}
	case diagnosticRuleCustom,
		diagnosticRuleIn,
		diagnosticRuleNotIn,
		diagnosticRuleItemCount,
		diagnosticRuleContains,
		diagnosticRuleEachIn,
		diagnosticRuleUnique,
		diagnosticRuleEntryCount,
		diagnosticRuleHasKeys,
		diagnosticRuleKeysIn,
		diagnosticRuleSubsetOf,
		diagnosticRuleDisjoint,
		diagnosticRuleKeysSubsetOf,
		diagnosticRuleKeysDisjoint,
		diagnosticRuleExactlyOne,
		diagnosticRuleAllOrNone,
		diagnosticRuleAtMostOne,
		diagnosticRuleAtLeastOne:
	}
	return fmt.Sprintf(report.ResolveFrameworkText("%s does not satisfy the required rule"), subject)
}

func rulePresentationOperands(diagnosticSensitive bool, rule analysisDiagnosticRuleFailure) []string {
	if diagnosticSensitive {
		return nil
	}
	result := make([]string, 0, min(len(rule.Operands), diagnosticPresentationListLimit))
	for _, operand := range rule.Operands[:min(len(rule.Operands), diagnosticPresentationListLimit)] {
		if operand.Sensitive || operand.Literal == nil || operand.Literal.Sensitive || operand.Literal.Redacted {
			return nil
		}
		text := boundedDiagnosticText(operand.Literal.Text)
		if text == "" {
			continue
		}
		result = append(result, text)
	}
	return result
}

func diagnosticPresentationItems(report report, diagnostic diagnostic) []string {
	if diagnostic.Sensitive {
		return nil
	}
	items := make([]string, 0, diagnosticPresentationListLimit)
	input := diagnosticInputText(report, diagnostic)
	if diagnostic.Rejection != nil && diagnostic.Message != "" && input != "" {
		items = append(items, report.ResolveFrameworkText("input")+": "+input)
	}
	expected := diagnosticExpectedPresentation(diagnostic)
	if len(expected) > 1 || len(expected) == 1 && diagnostic.Rejection != nil {
		items = append(items, report.ResolveFrameworkText("expected")+": "+strings.Join(expected, ", "))
	}
	if len(diagnostic.Suggestions) == 1 {
		items = append(items, report.ResolveFrameworkText("suggestion")+": "+boundedDiagnosticText(diagnostic.Suggestions[0]))
	} else if len(diagnostic.Suggestions) > 1 {
		limit := min(len(diagnostic.Suggestions), diagnosticPresentationListLimit)
		values := make([]string, 0, limit)
		for _, suggestion := range diagnostic.Suggestions[:limit] {
			values = append(values, boundedDiagnosticText(suggestion))
		}
		items = append(items, report.ResolveFrameworkText("suggestions")+": "+strings.Join(values, ", "))
	}
	if diagnostic.Reason != "" {
		reason := diagnostic.Reason
		if diagnostic.FrameworkReason {
			reason = report.ResolveFrameworkText(reason)
		}
		items = append(items, report.ResolveFrameworkText("reason")+": "+boundedDiagnosticText(reason))
	}
	if diagnostic.Hint != "" {
		hint := diagnostic.Hint
		if diagnostic.FrameworkHint {
			hint = report.ResolveFrameworkText(hint)
		}
		items = append(items, report.ResolveFrameworkText("hint")+": "+boundedDiagnosticText(hint))
	}
	for _, usage := range diagnostic.Usage[:min(len(diagnostic.Usage), diagnosticPresentationUsageLimit)] {
		items = append(items, report.ResolveFrameworkText("usage")+": "+boundedDiagnosticText(usage))
	}
	if diagnosticShowsCode(diagnostic) && len(diagnostic.CommandPath) != 0 {
		items = append(items, report.ResolveFrameworkText("command")+": "+boundedDiagnosticText(strings.Join(diagnostic.CommandPath, " ")))
	}
	if location := diagnosticSourceLocation(report, diagnostic); location != "" {
		items = append(items, report.ResolveFrameworkText("source")+": "+location)
	}
	if failure := diagnostic.NoticeDelivery; failure != nil {
		items = append(items, report.ResolveFrameworkText("notice")+": "+boundedDiagnosticText(failure.Code))
	}
	if overflow := diagnostic.SignalOverflow; overflow != nil {
		items = append(items, report.ResolveFrameworkText("signal boundary")+": "+report.ResolveFrameworkText(string(overflow.Boundary)))
	}
	return items
}

func diagnosticExpectedPresentation(diagnostic diagnostic) []string {
	if diagnostic.Sensitive {
		return nil
	}
	if diagnostic.Rejection != nil && len(diagnostic.Rejection.Expected) != 0 {
		return boundedDiagnosticList(diagnostic.Rejection.Expected)
	}
	return boundedDiagnosticList(diagnostic.Expected)
}

func boundedDiagnosticList(values []string) []string {
	limit := min(len(values), diagnosticPresentationListLimit)
	result := make([]string, 0, limit)
	for _, value := range values[:limit] {
		value = boundedDiagnosticText(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func diagnosticSourceLocation(report report, diagnostic diagnostic) string {
	if diagnostic.Sensitive || diagnostic.Source == 0 {
		return ""
	}
	source, ok := report.Sources[diagnostic.Source]
	if !ok || source.Sensitive {
		return ""
	}
	return boundedDiagnosticText(source.Location)
}

func diagnosticShowsCode(diagnostic diagnostic) bool {
	switch DiagnosticCode(diagnostic.Code) {
	case DiagnosticUnknownCommand,
		DiagnosticAmbiguousCommand,
		DiagnosticUnknownFlag,
		DiagnosticAmbiguousFlag,
		DiagnosticMisplacedFlag,
		DiagnosticRepeatedFlag,
		DiagnosticMissingValue,
		DiagnosticInvalidValue,
		DiagnosticUnexpectedArgument,
		DiagnosticAmbiguousArgumentForm,
		DiagnosticInvalidArgumentSyntax,
		DiagnosticActionConflict,
		DiagnosticNotIntroducedValue,
		DiagnosticRemovedValue,
		DiagnosticNotIntroducedCommand,
		DiagnosticNotIntroducedFlag,
		DiagnosticNotIntroducedArgument,
		DiagnosticRemovedCommand,
		DiagnosticRemovedFlag,
		DiagnosticRemovedArgument,
		DiagnosticUnavailableCommand,
		DiagnosticUnavailableFlag,
		DiagnosticUnavailableArgument,
		DiagnosticRuleViolation,
		DiagnosticNotRunnableCommand:
		return false
	default:
		return diagnostic.Code != ""
	}
}

func diagnosticCodeProblem(report report, code string) string {
	text := strings.ReplaceAll(boundedDiagnosticText(code), "-", " ")
	switch {
	case strings.HasSuffix(text, " failure"):
		text = strings.TrimSuffix(text, " failure") + " failed"
	case strings.HasSuffix(text, " cancellation"):
		text = strings.TrimSuffix(text, " cancellation") + " was cancelled"
	}
	if text == "" {
		text = "frontend failure"
	}
	return report.ResolveFrameworkText(text)
}

func diagnosticInputText(report report, diagnostic diagnostic) string {
	if diagnostic.Sensitive || diagnostic.InputRedacted {
		return ""
	}
	if diagnostic.Input != "" {
		return boundedDiagnosticText(diagnostic.Input)
	}
	if input, ok := diagnosticInputExcerpt(report, diagnostic); ok {
		return input
	}
	return ""
}

func diagnosticSubject(diagnostic diagnostic) string {
	return boundedDiagnosticText(diagnostic.Subject)
}

func diagnosticInputExcerpt(report report, diagnostic diagnostic) (string, bool) {
	if diagnostic.Sensitive {
		return "", false
	}
	if diagnostic.Span != nil {
		if text, ok := safeSpanText(report, *diagnostic.Span); ok {
			return text, true
		}
	}
	for _, id := range diagnostic.Origins {
		origin, ok := report.Origins[id]
		if !ok || origin.Sensitive {
			continue
		}
		if text, ok := safeSpanText(report, origin.Span); ok {
			return text, true
		}
	}
	return "", false
}

func safeSpanText(report report, span span) (string, bool) {
	source, ok := report.Sources[span.Source]
	if !ok || source.Sensitive || span.Start.Offset < 0 || span.End.Offset < span.Start.Offset {
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
