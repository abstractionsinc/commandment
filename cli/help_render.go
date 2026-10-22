package cli

import (
	"context"
	"strings"
)

const (
	helpFlagDetailCapacity      = 4
	helpCompactValueDomainLimit = 4
)

func buildHelpUsage(page HelpPage) []string {
	base := page.ApplicationName
	if len(page.SelectedPath) != 0 {
		base += " " + strings.Join(page.SelectedPath, " ")
	}
	if len(page.FlagGroups) != 0 {
		base += " [flags]"
	}
	if len(page.ArgumentForms) == 0 {
		if len(page.CommandGroups) != 0 {
			if page.Runnable {
				return []string{base, base + " [command]"}
			}
			return []string{base + " <command>"}
		}
		return []string{base}
	}
	result := make([]string, 0, len(page.ArgumentForms)+1)
	for _, form := range page.ArgumentForms {
		var line strings.Builder
		line.WriteString(base)
		for _, term := range form.Terms {
			line.WriteString(" " + renderHelpUsageTerm(term))
		}
		result = append(result, line.String())
	}
	if len(page.CommandGroups) != 0 && page.Runnable {
		result = append(result, base+" [command]")
	}
	return result
}

func renderHelpUsageTerm(term HelpArgumentTerm) string {
	if term.Literal != "" {
		return term.Literal
	}
	name := term.Name
	if name == "" {
		name = "VALUE"
	}
	if maximum, bounded := term.Count.Maximum(); !bounded || maximum > 1 {
		name += "..."
	}
	if term.Optional {
		return "[" + name + "]"
	}
	return "<" + name + ">"
}

func renderDefaultHelp(ctx context.Context, page HelpPage, sections helpSectionSet) (string, error) {
	if err := ctx.Err(); err != nil {
		failure := &helpRenderingError{
			cause: err,
		}
		return "", failure
	}
	page.Labels = helpLabelsWithFrameworkFallbacks(page.Labels)
	document := buildHelpPresentationDocument(page, sections)
	profile := profileFromContext(ctx, destinationStdoutInformational)
	return render(ctx, document, profile)
}

func buildHelpPresentationDocument(page HelpPage, sections helpSectionSet) document {
	document := document{
		Destination: destinationStdoutInformational,
	}
	if sections.has(HelpSectionOverview) {
		description := page.Description.Long
		if description == "" {
			description = page.Description.Short
		}
		if description != "" {
			document.Blocks = append(document.Blocks, block{
				Kind: blockParagraph,
				Spans: []presentationSpan{{
					Text: description,
				}},
			})
		}
		if lifecycle := helpLifecycleAnnotation(page.Labels, page.Release, page.LifecycleState); lifecycle != "" {
			document.Blocks = append(document.Blocks, block{
				Kind: blockHint,
				Spans: []presentationSpan{{
					Text: lifecycle,
					Role: spanWarning,
				}},
			})
		}
	}
	if sections.has(HelpSectionUsage) && len(page.Usage) != 0 {
		document.Blocks = append(document.Blocks,
			block{
				Kind:  blockSection,
				Title: page.Labels.Usage,
			},
			block{
				Kind:  blockUsageLines,
				Lines: append([]string(nil), page.Usage...),
			},
		)
	}
	if sections.has(HelpSectionCommands) {
		appendHelpCommandBlocks(&document, page)
	}
	if sections.has(HelpSectionArguments) {
		appendHelpArgumentBlocks(&document, page)
	}
	if sections.has(HelpSectionFlags) {
		appendHelpFlagBlocks(&document, page)
	}
	if sections.has(HelpSectionExamples) && len(page.Examples) != 0 {
		document.Blocks = append(document.Blocks, block{
			Kind:  blockSection,
			Title: page.Labels.Examples,
		})
		for _, example := range page.Examples {
			if example.Description != "" {
				document.Blocks = append(document.Blocks, block{
					Kind: blockParagraph,
					Spans: []presentationSpan{{
						Text: example.Description,
					}},
				})
			}
			document.Blocks = append(document.Blocks, block{
				Kind:  blockLiteralLine,
				Lines: []string{"  " + example.CommandLine},
			})
		}
	}
	return document
}

func appendHelpCommandBlocks(document *document, page HelpPage) {
	for _, group := range page.CommandGroups {
		if len(group.Commands) == 0 {
			continue
		}
		title := group.Title
		if title == "" {
			title = page.Labels.Commands
		}
		entries := make([]presentationEntry, 0, len(group.Commands))
		for _, command := range group.Commands {
			description := command.Description.Short
			if description == "" {
				description = command.Description.Long
			}
			var annotations []string
			switch command.Kind {
			case HelpCommandAlias:
				annotations = append(annotations, page.Labels.Alias)
			case HelpCommandShortcut:
				annotations = append(annotations, page.Labels.Shortcut)
			case HelpCommandUnknown, HelpCommandCanonical, HelpCommandAction:
			}
			if command.Default {
				annotations = append(annotations, page.Labels.Default)
			}
			if lifecycle := helpLifecycleAnnotation(
				page.Labels,
				command.Release,
				command.LifecycleState,
			); lifecycle != "" {
				annotations = append(annotations, lifecycle)
			}
			entries = append(entries, presentationEntry{
				Label:       command.Name,
				Description: helpDescription(description, helpAnnotation(annotations)),
			})
		}
		document.Blocks = append(document.Blocks,
			block{
				Kind:  blockSection,
				Title: title,
			},
			block{
				Kind:    blockEntryList,
				Entries: entries,
			},
		)
	}
}

func appendHelpFlagBlocks(document *document, page HelpPage) {
	for _, group := range page.FlagGroups {
		if len(group.Flags) == 0 {
			continue
		}
		title := group.Title
		if title == "" {
			if group.Kind == HelpGroupInherited {
				title = page.Labels.GlobalFlags
			} else {
				title = page.Labels.Flags
			}
		}
		entries := make([]presentationEntry, 0, len(group.Flags))
		for _, flag := range group.Flags {
			if flag.Sensitive {
				continue
			}
			label := helpFlagLabel(page, flag)
			if label == "" {
				continue
			}
			var annotations []string
			if flag.Required {
				annotations = append(annotations, page.Labels.Required)
			}
			if lifecycle := helpLifecycleAnnotation(
				page.Labels,
				flag.Release,
				flag.LifecycleState,
			); lifecycle != "" {
				annotations = append(annotations, lifecycle)
			}
			details := make([]string, 0, helpFlagDetailCapacity)
			details = append(details, flag.Description, helpAnnotation(annotations))
			details = append(details, helpFlagStaticValueDetails(page, flag)...)
			domain := buildHelpValueDomainPresentation(page, flag.ValueDomainUses, flag.ValueName, flag.Sensitive)
			details = append(details, domain.descriptions...)
			entries = append(entries, presentationEntry{
				Label:       label,
				Description: helpDescription(details...),
				Lists:       domain.lists,
			})
		}
		if len(entries) == 0 {
			continue
		}
		document.Blocks = append(document.Blocks,
			block{
				Kind:  blockSection,
				Title: title,
			},
			block{
				Kind:    blockEntryList,
				Entries: entries,
			},
		)
	}
}

func appendHelpArgumentBlocks(document *document, page HelpPage) {
	nonemptyForms := 0
	for _, form := range page.ArgumentForms {
		if len(form.Terms) != 0 {
			nonemptyForms++
		}
	}
	if nonemptyForms == 0 {
		return
	}
	document.Blocks = append(document.Blocks, block{
		Kind:  blockSection,
		Title: page.Labels.Arguments,
	})
	formIndex := 0
	for _, form := range page.ArgumentForms {
		if len(form.Terms) == 0 {
			continue
		}
		if nonemptyForms > 1 && len(form.Terms) > 1 {
			document.Blocks = append(document.Blocks, block{
				Kind:  blockUsageLines,
				Lines: []string{helpArgumentFormUsage(form)},
			})
		}
		entries := make([]presentationEntry, 0, len(form.Terms))
		for _, term := range form.Terms {
			var annotations []string
			if lifecycle := helpLifecycleAnnotation(
				page.Labels,
				term.Release,
				term.LifecycleState,
			); lifecycle != "" {
				annotations = append(annotations, lifecycle)
			}
			details := []string{term.Description, helpAnnotation(annotations)}
			domain := buildHelpValueDomainPresentation(page, term.ValueDomainUses, term.Name, term.Sensitive)
			details = append(details, domain.descriptions...)
			if term.TokenLanguage != nil && len(term.TokenLanguage.Fragments) != 0 {
				details = append(details, page.Labels.Syntax+": "+strings.Join(term.TokenLanguage.Fragments, " "))
			}
			entries = append(entries, presentationEntry{
				Label:       renderHelpUsageTerm(term),
				Description: helpDescription(details...),
				Lists:       domain.lists,
			})
		}
		document.Blocks = append(document.Blocks, block{
			Kind:    blockEntryList,
			Entries: entries,
		})
		formIndex++
		if nonemptyForms > 1 && formIndex < nonemptyForms {
			document.Blocks = append(document.Blocks, block{
				Kind: blockParagraph,
			})
		}
	}
}

func helpArgumentFormUsage(form HelpArgumentForm) string {
	var line strings.Builder
	for index, term := range form.Terms {
		if index != 0 {
			line.WriteByte(' ')
		}
		line.WriteString(renderHelpUsageTerm(term))
	}
	return line.String()
}

func helpFlagLabel(page HelpPage, flag HelpFlag) string {
	if flag.ActionEntry != 0 {
		return strings.Join(helpFlagSpellingForms(flag.ActionSpelling, flag.ValueName), ", ")
	}
	visible := make(map[FlagSpellingID]struct{}, len(flag.FlagSpellings))
	for _, id := range flag.FlagSpellings {
		visible[id] = struct{}{}
	}
	var labels []string
	for _, id := range flag.FlagSpellings {
		node, ok := page.FlagSpellings[id]
		if !ok {
			continue
		}
		forms := helpFlagSpellingForms(node.Spelling, flag.ValueName)
		for index := range forms {
			var annotations []string
			if lifecycle := helpLifecycleAnnotation(
				page.Labels,
				node.Release,
				node.LifecycleState,
			); lifecycle != "" {
				annotations = append(annotations, lifecycle)
			}
			if node.Replacement != 0 {
				if _, shown := visible[node.Replacement]; shown {
					if replacement, exists := page.FlagSpellings[node.Replacement]; exists {
						annotations = append(annotations, page.Labels.Use+" "+replacement.Spelling.Text)
					}
				}
			}
			if annotation := helpAnnotation(annotations); annotation != "" {
				forms[index] += " " + annotation
			}
		}
		labels = append(labels, forms...)
	}
	return strings.Join(labels, ", ")
}

func helpFlagSpellingForms(spelling FlagSpelling, valueName string) []string {
	if spelling.Text == "" {
		return nil
	}
	if spelling.ValueMode == FlagValueModeNone || spelling.ValueMode == FlagValueModeUnknown {
		return []string{spelling.Text}
	}
	if valueName == "" {
		valueName = "VALUE"
	}
	forms := spelling.ValueForms
	if len(forms) == 0 {
		if spelling.ValueMode == FlagValueModeOptional {
			return []string{spelling.Text + " [" + valueName + "]"}
		}
		return []string{spelling.Text + " " + valueName}
	}
	result := make([]string, 0, len(forms))
	for _, form := range forms {
		switch form {
		case FlagValueFormSeparate:
			if spelling.ValueMode == FlagValueModeOptional {
				result = append(result, spelling.Text+" ["+valueName+"]")
			} else {
				result = append(result, spelling.Text+" "+valueName)
			}
		case FlagValueFormJoined:
			if spelling.ValueMode == FlagValueModeOptional {
				result = append(result, spelling.Text+"["+valueName+"]")
			} else {
				result = append(result, spelling.Text+valueName)
			}
		case FlagValueFormAssigned:
			if spelling.ValueMode == FlagValueModeOptional {
				result = append(result, spelling.Text+"[="+valueName+"]")
			} else {
				result = append(result, spelling.Text+"="+valueName)
			}
		case FlagValueFormUnknown:
		}
	}
	if len(result) == 0 {
		if spelling.ValueMode == FlagValueModeOptional {
			return []string{spelling.Text + " [" + valueName + "]"}
		}
		return []string{spelling.Text + " " + valueName}
	}
	return result
}

func helpFlagStaticValueDetails(page HelpPage, flag HelpFlag) []string {
	if flag.Sensitive || flag.ActionEntry != 0 {
		return nil
	}
	var details []string
	for _, id := range flag.FlagSpellings {
		node, ok := page.FlagSpellings[id]
		if !ok || node.Spelling.Sensitive {
			continue
		}
		spelling := node.Spelling
		if literal := spelling.ImplicitValue; literal != nil && !literal.Sensitive && !literal.Redacted &&
			literal.Text != "" {
			details = append(details, page.Labels.ImplicitValue+" for "+spelling.Text+": "+literal.Text)
		}
		if literal := spelling.FixedValue; literal != nil && !literal.Sensitive && !literal.Redacted &&
			literal.Text != "" &&
			literal.Type.Kind != ValueKindBool {
			details = append(details, page.Labels.FixedValue+" for "+spelling.Text+": "+literal.Text)
		}
	}
	return details
}

type helpValueDomainPresentation struct {
	descriptions []string
	lists        []presentationList
}

func buildHelpValueDomainPresentation(page HelpPage, uses []HelpValueDomainUse, valueName string, sensitive bool) helpValueDomainPresentation {
	if sensitive {
		return helpValueDomainPresentation{}
	}
	var result helpValueDomainPresentation
	for _, use := range uses {
		if len(use.Entries) == 0 {
			continue
		}
		visible := make(map[ValueDomainEntryID]struct{}, len(use.Entries))
		for _, id := range use.Entries {
			visible[id] = struct{}{}
		}
		entries := make([]string, 0, len(use.Entries))
		listItems := make([]string, 0, len(use.Entries))
		hasMemberDetails := false
		for _, id := range use.Entries {
			entry, ok := page.ValueDomainEntries[id]
			if !ok {
				continue
			}
			text := entry.Name
			if entry.Kind == ValueDomainEntryAlias && entry.Canonical != 0 {
				if _, shown := visible[entry.Canonical]; shown {
					if canonical, exists := page.ValueDomainEntries[entry.Canonical]; exists {
						text += " (" + page.Labels.AliasFor + " " + canonical.Name + ")"
						hasMemberDetails = true
					}
				}
			}
			var annotations []string
			if lifecycle := helpLifecycleAnnotation(
				page.Labels,
				entry.Release,
				entry.LifecycleState,
			); lifecycle != "" {
				annotations = append(annotations, lifecycle)
				hasMemberDetails = true
			}
			if entry.Replacement != 0 {
				if _, shown := visible[entry.Replacement]; shown {
					if replacement, exists := page.ValueDomainEntries[entry.Replacement]; exists {
						annotations = append(annotations, page.Labels.Use+" "+replacement.Name)
						hasMemberDetails = true
					}
				}
			}
			annotation := helpAnnotation(annotations)
			expanded := text
			if entry.Kind == ValueDomainEntryCanonical && entry.Description != "" {
				expanded += " - " + entry.Description
				hasMemberDetails = true
			}
			if annotation != "" {
				expanded += "\n" + annotation
				text += " " + annotation
			}
			if entry.Kind == ValueDomainEntryCanonical && entry.Description != "" {
				text += " - " + entry.Description
			}
			entries = append(entries, text)
			listItems = append(listItems, expanded)
		}
		if len(entries) == 0 {
			continue
		}
		label := helpValueDomainLabel(use.Kind, valueName, page.Labels)
		if len(entries) > helpCompactValueDomainLimit || len(entries) > 1 && hasMemberDetails {
			list := presentationList{
				Label: label,
				Items: listItems,
			}
			result.lists = append(result.lists, list)
			continue
		}
		result.descriptions = append(result.descriptions, label+": "+strings.Join(entries, ", "))
	}
	return result
}

func helpValueDomainLabel(kind ValueDomainUseKind, valueName string, labels HelpLabels) string {
	switch kind {
	case ValueDomainUseMapKey:
		return labels.Keys
	case ValueDomainUseMapValue:
		return labels.Values
	case ValueDomainUseUnknown, ValueDomainUseDestination, ValueDomainUseSequenceElement:
	}
	if valueName != "" {
		return valueName
	}
	return labels.Values
}

func helpLifecycleAnnotation(labels HelpLabels, lifecycle ReleaseLifecycle, state LifecycleState) string {
	if state != LifecycleStateDeprecated {
		return ""
	}
	text := labels.Deprecated
	if lifecycle.DeprecatedIn != "" {
		text += " " + lifecycle.DeprecatedIn
	}
	if lifecycle.Reason != "" {
		text += ": " + lifecycle.Reason
	}
	return text
}

func helpAnnotation(values []string) string {
	values = compactHelpText(values)
	if len(values) == 0 {
		return ""
	}
	return "[" + strings.Join(values, "; ") + "]"
}

func helpDescription(values ...string) string {
	return strings.Join(compactHelpText(values), "\n")
}

func compactHelpText(values []string) []string {
	result := values[:0]
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
