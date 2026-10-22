package cli

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

var (
	errDiagnosticGraphZeroOriginReference      = errors.New("invocation: zero origin reference")
	errDiagnosticGraphZeroRepairSource         = errors.New("invocation: zero repair source")
	errDiagnosticGraphInvalidReplacementPart   = errors.New("invocation: replacement part must contain exactly one literal or copy")
	errDiagnosticGraphSensitiveRepairLiteral   = errors.New("invocation: sensitive semantic content cannot be retained as repair literal")
	errDiagnosticGraphFlatTextEntryEdit        = errors.New("invocation: entry edits are invalid for flat text source")
	errDiagnosticGraphForeignRepairSource      = errors.New("invocation: text edit references foreign repair source")
	errDiagnosticGraphOverlappingTextEdits     = errors.New("invocation: overlapping text edits")
	errDiagnosticGraphInvalidEntryEditRange    = errors.New("invocation: invalid entry edit range")
	errDiagnosticGraphOverlappingEntryEdits    = errors.New("invocation: overlapping entry edits")
	errDiagnosticGraphEditTargetsReplacedEntry = errors.New("invocation: text edit targets replaced entry")
	errDiagnosticGraphInvalidSourceSpan        = errors.New("invocation: invalid source span")
)

type diagnosticGraphReferenceKind uint8

const (
	diagnosticGraphRepairIndexOutOfRange diagnosticGraphReferenceKind = iota
	diagnosticGraphUnresolvedInputSource
	diagnosticGraphUnresolvedOriginParent
	diagnosticGraphUnresolvedDiagnosticSource
	diagnosticGraphUnresolvedDiagnosticOrigin
	diagnosticGraphUnresolvedDiagnosticRepair
	diagnosticGraphOriginAncestryCycle
	diagnosticGraphUnresolvedInputOrigin
	diagnosticGraphDuplicateRepairSource
	diagnosticGraphRepairSourceMissingRevision
	diagnosticGraphCopySourceMissingRevision
	diagnosticGraphUnresolvedSpanSource
	diagnosticGraphUnresolvedRepairSource
)

type diagnosticGraphReferenceError struct {
	kind   diagnosticGraphReferenceKind
	source sourceID
	origin originID
	index  int
}

func (err diagnosticGraphReferenceError) Error() string {
	switch err.kind {
	case diagnosticGraphRepairIndexOutOfRange:
		return fmt.Sprintf("invocation: diagnostic repair index %d out of range", err.index)
	case diagnosticGraphUnresolvedInputSource:
		return fmt.Sprintf("invocation: unresolved input source %d", err.source)
	case diagnosticGraphUnresolvedOriginParent:
		return fmt.Sprintf("invocation: unresolved origin parent %d", err.origin)
	case diagnosticGraphUnresolvedDiagnosticSource:
		return fmt.Sprintf("invocation: unresolved diagnostic source %d", err.source)
	case diagnosticGraphUnresolvedDiagnosticOrigin:
		return fmt.Sprintf("invocation: unresolved diagnostic origin %d", err.origin)
	case diagnosticGraphUnresolvedDiagnosticRepair:
		return fmt.Sprintf("invocation: unresolved diagnostic repair %d", err.index)
	case diagnosticGraphOriginAncestryCycle:
		return fmt.Sprintf("invocation: origin ancestry cycle at %d", err.origin)
	case diagnosticGraphUnresolvedInputOrigin:
		return fmt.Sprintf("invocation: unresolved input origin %d", err.origin)
	case diagnosticGraphDuplicateRepairSource:
		return fmt.Sprintf("invocation: duplicate repair source %d", err.source)
	case diagnosticGraphRepairSourceMissingRevision:
		return fmt.Sprintf("invocation: repair source %d lacks verified revision", err.source)
	case diagnosticGraphCopySourceMissingRevision:
		return fmt.Sprintf("invocation: copy source %d lacks verified revision", err.source)
	case diagnosticGraphUnresolvedSpanSource:
		return fmt.Sprintf("invocation: unresolved span source %d", err.source)
	case diagnosticGraphUnresolvedRepairSource:
		return fmt.Sprintf("invocation: unresolved repair source %d", err.source)
	default:
		return "invocation: invalid diagnostic graph reference"
	}
}

type reportBuilder struct {
	report               report
	nextSource           sourceID
	nextOrigin           originID
	presentationConflict bool
}

func (builder *reportBuilder) addPayload(payload payload) error {
	if payload.Presentation != nil && !builder.presentationConflict {
		if builder.report.presentation == nil {
			builder.report.presentation = payload.Presentation
		} else if builder.report.presentation != payload.Presentation {
			builder.report.presentation = nil
			builder.presentationConflict = true
		}
	}
	sourceMap := map[sourceID]sourceID{}
	originMap := map[originID]originID{}

	neededSources := map[sourceID]struct{}{}
	neededOrigins := map[originID]struct{}{}
	neededRepairs := map[int]struct{}{}

	for _, diagnostic := range payload.Diagnostics {
		if err := validateDiagnosticMetadata(diagnostic); err != nil {
			return err
		}
		if diagnostic.Source != 0 {
			neededSources[diagnostic.Source] = struct{}{}
		}
		if diagnostic.Span != nil {
			neededSources[diagnostic.Span.Source] = struct{}{}
		}
		for _, origin := range diagnostic.Origins {
			neededOrigins[origin] = struct{}{}
		}
		for _, index := range diagnostic.RepairIndexes {
			if index < 0 || index >= len(payload.Repairs) {
				failure := diagnosticGraphReferenceError{
					kind:  diagnosticGraphRepairIndexOutOfRange,
					index: index,
				}
				return failure
			}
			neededRepairs[index] = struct{}{}
		}
	}
	for index := range neededRepairs {
		if err := collectRepairClosure(payload, payload.Repairs[index], neededSources); err != nil {
			return err
		}
	}
	if err := collectOriginClosure(payload, neededOrigins, neededSources); err != nil {
		return err
	}

	sourceIDs := sortedSourceIDs(neededSources)
	for _, old := range sourceIDs {
		source, ok := payload.Sources[old]
		if !ok || source.ID != old || source.Kind == InputKindUnknown {
			failure := diagnosticGraphReferenceError{
				kind:   diagnosticGraphUnresolvedInputSource,
				source: old,
			}
			return failure
		}
		newID := builder.nextSource
		builder.nextSource++
		sourceMap[old] = newID
		source.ID = newID
		source.Entries = append([]string(nil), source.Entries...)
		if source.FlatText != nil {
			text := *source.FlatText
			source.FlatText = &text
		}
		builder.report.Sources[newID] = source
	}

	originIDs := sortedOriginIDs(neededOrigins)
	for _, old := range originIDs {
		newID := builder.nextOrigin
		builder.nextOrigin++
		originMap[old] = newID
	}
	for _, old := range originIDs {
		origin := payload.Origins[old]
		origin.ID = originMap[old]
		var err error
		origin.Span, err = rebaseSpan(origin.Span, sourceMap)
		if err != nil {
			return err
		}
		parents := make([]originID, len(origin.Parents))
		for i, parent := range origin.Parents {
			mapped, ok := originMap[parent]
			if !ok {
				failure := diagnosticGraphReferenceError{
					kind:   diagnosticGraphUnresolvedOriginParent,
					origin: parent,
				}
				return failure
			}
			parents[i] = mapped
		}
		origin.Parents = parents
		builder.report.Origins[origin.ID] = origin
	}

	repairMap := map[int]int{}
	repairIndexes := make([]int, 0, len(neededRepairs))
	for index := range neededRepairs {
		repairIndexes = append(repairIndexes, index)
	}
	slices.Sort(repairIndexes)
	for _, oldIndex := range repairIndexes {
		repair, err := rebaseRepair(payload.Repairs[oldIndex], sourceMap)
		if err != nil {
			return err
		}
		newIndex := len(builder.report.Repairs)
		repairMap[oldIndex] = newIndex
		builder.report.Repairs = append(builder.report.Repairs, repair)
	}

	for _, diagnostic := range payload.Diagnostics {
		diagnosticCopy := cloneDiagnostic(diagnostic)
		if diagnosticCopy.Source != 0 {
			mapped, ok := sourceMap[diagnosticCopy.Source]
			if !ok {
				failure := diagnosticGraphReferenceError{
					kind:   diagnosticGraphUnresolvedDiagnosticSource,
					source: diagnosticCopy.Source,
				}
				return failure
			}
			diagnosticCopy.Source = mapped
		}
		if diagnosticCopy.Span != nil {
			span, err := rebaseSpan(*diagnosticCopy.Span, sourceMap)
			if err != nil {
				return err
			}
			diagnosticCopy.Span = &span
		}
		for i, origin := range diagnosticCopy.Origins {
			mapped, ok := originMap[origin]
			if !ok {
				failure := diagnosticGraphReferenceError{
					kind:   diagnosticGraphUnresolvedDiagnosticOrigin,
					origin: origin,
				}
				return failure
			}
			diagnosticCopy.Origins[i] = mapped
		}
		for i, index := range diagnosticCopy.RepairIndexes {
			mapped, ok := repairMap[index]
			if !ok {
				failure := diagnosticGraphReferenceError{
					kind:  diagnosticGraphUnresolvedDiagnosticRepair,
					index: index,
				}
				return failure
			}
			diagnosticCopy.RepairIndexes[i] = mapped
		}
		if diagnosticCopy.Sensitive {
			diagnosticCopy.Input = ""
			diagnosticCopy.Suggestions = nil
			diagnosticCopy.InputRedacted = true
		}
		builder.report.Diagnostics = append(builder.report.Diagnostics, diagnosticCopy)
	}
	builder.report.Truncated = builder.report.Truncated || payload.Truncated
	return nil
}

const (
	originVisitActive uint8 = iota + 1
	originVisitComplete
)

func collectOriginClosure(payload payload, neededOrigins map[originID]struct{}, neededSources map[sourceID]struct{}) error {
	state := map[originID]uint8{}
	var visit func(originID) error
	visit = func(id originID) error {
		if id == 0 {
			return errDiagnosticGraphZeroOriginReference
		}
		switch state[id] {
		case originVisitActive:
			failure := diagnosticGraphReferenceError{
				kind:   diagnosticGraphOriginAncestryCycle,
				origin: id,
			}
			return failure
		case originVisitComplete:
			return nil
		}
		origin, ok := payload.Origins[id]
		if !ok || origin.ID != id {
			failure := diagnosticGraphReferenceError{
				kind:   diagnosticGraphUnresolvedInputOrigin,
				origin: id,
			}
			return failure
		}
		state[id] = originVisitActive
		neededOrigins[id] = struct{}{}
		neededSources[origin.Span.Source] = struct{}{}
		for _, parent := range origin.Parents {
			if err := visit(parent); err != nil {
				return err
			}
		}
		state[id] = originVisitComplete
		return nil
	}
	initial := make([]originID, 0, len(neededOrigins))
	for id := range neededOrigins {
		initial = append(initial, id)
	}
	for _, id := range initial {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func collectRepairClosure(payload payload, repair repair, neededSources map[sourceID]struct{}) error {
	seen := map[sourceID]struct{}{}
	for _, sourceRepair := range repair.Sources {
		if sourceRepair.Source == 0 {
			return errDiagnosticGraphZeroRepairSource
		}
		if _, duplicate := seen[sourceRepair.Source]; duplicate {
			failure := diagnosticGraphReferenceError{
				kind:   diagnosticGraphDuplicateRepairSource,
				source: sourceRepair.Source,
			}
			return failure
		}
		seen[sourceRepair.Source] = struct{}{}
		source, ok := payload.Sources[sourceRepair.Source]
		if !ok || source.ID != sourceRepair.Source || source.Revision == "" {
			failure := diagnosticGraphReferenceError{
				kind:   diagnosticGraphRepairSourceMissingRevision,
				source: sourceRepair.Source,
			}
			return failure
		}
		neededSources[sourceRepair.Source] = struct{}{}
		if err := validateRepairSource(sourceRepair, source); err != nil {
			return err
		}
		for _, edit := range sourceRepair.TextEdits {
			if err := collectReplacementSources(payload, edit.Replacement, neededSources); err != nil {
				return err
			}
		}
		for _, edit := range sourceRepair.EntryEdits {
			for _, replacement := range edit.Entries {
				if err := collectReplacementSources(payload, replacement, neededSources); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func collectReplacementSources(payload payload, replacement replacement, neededSources map[sourceID]struct{}) error {
	for _, part := range replacement {
		if (part.Literal == nil) == (part.Copy == nil) {
			return errDiagnosticGraphInvalidReplacementPart
		}
		if part.Sensitive && part.Literal != nil {
			return errDiagnosticGraphSensitiveRepairLiteral
		}
		if part.Copy != nil {
			if err := validateSpan(*part.Copy, payload.Sources); err != nil {
				return err
			}
			source := payload.Sources[part.Copy.Source]
			if source.Revision == "" {
				failure := diagnosticGraphReferenceError{
					kind:   diagnosticGraphCopySourceMissingRevision,
					source: part.Copy.Source,
				}
				return failure
			}
			neededSources[part.Copy.Source] = struct{}{}
		}
	}
	return nil
}

func validateRepairSource(repair repairSource, repairTarget source) error {
	if repairTarget.FlatText != nil && len(repair.EntryEdits) != 0 {
		return errDiagnosticGraphFlatTextEntryEdit
	}
	text := append([]textEdit(nil), repair.TextEdits...)
	slices.SortStableFunc(text, func(left, right textEdit) int {
		return cmp.Or(
			cmp.Compare(left.Span.Entry, right.Span.Entry),
			cmp.Compare(left.Span.Start.Offset, right.Span.Start.Offset),
		)
	})
	for i, edit := range text {
		if edit.Span.Source != repair.Source {
			return errDiagnosticGraphForeignRepairSource
		}
		if err := validateSpan(edit.Span, map[sourceID]source{repairTarget.ID: repairTarget}); err != nil {
			return err
		}
		if i > 0 && text[i-1].Span.Entry == edit.Span.Entry && text[i-1].Span.End.Offset > edit.Span.Start.Offset {
			return errDiagnosticGraphOverlappingTextEdits
		}
	}
	entries := append([]entryEdit(nil), repair.EntryEdits...)
	slices.SortStableFunc(entries, func(left, right entryEdit) int {
		return cmp.Or(
			cmp.Compare(left.Start, right.Start),
			cmp.Compare(left.End, right.End),
		)
	})
	for i, edit := range entries {
		if edit.Start < 0 || edit.End < edit.Start || edit.End > len(repairTarget.Entries) {
			return errDiagnosticGraphInvalidEntryEditRange
		}
		if i > 0 {
			prev := entries[i-1]
			if prev.End > edit.Start || (prev.Start == prev.End && edit.Start == edit.End && prev.Start == edit.Start) {
				// Same-boundary insertions are legal but are coalesced during rebasing.
				if prev.Start != prev.End || edit.Start != edit.End || prev.Start != edit.Start {
					return errDiagnosticGraphOverlappingEntryEdits
				}
			}
		}
	}
	for _, edit := range text {
		for _, entryEdit := range entries {
			if entryEdit.Start <= edit.Span.Entry && edit.Span.Entry < entryEdit.End {
				return errDiagnosticGraphEditTargetsReplacedEntry
			}
		}
	}
	return nil
}

func validateSpan(span span, sources map[sourceID]source) error {
	source, ok := sources[span.Source]
	if !ok || source.ID != span.Source {
		failure := diagnosticGraphReferenceError{
			kind:   diagnosticGraphUnresolvedSpanSource,
			source: span.Source,
		}
		return failure
	}
	if span.Start.Offset < 0 || span.End.Offset < span.Start.Offset {
		return errDiagnosticGraphInvalidSourceSpan
	}
	if span.Entry == -1 {
		if source.FlatText == nil || span.End.Offset > len(*source.FlatText) {
			return errDiagnosticGraphInvalidSourceSpan
		}
		return nil
	}
	if span.Entry < 0 || span.Entry >= len(source.Entries) || source.FlatText != nil ||
		span.End.Offset > len(source.Entries[span.Entry]) {
		return errDiagnosticGraphInvalidSourceSpan
	}
	return nil
}

func rebaseRepair(value repair, sourceMap map[sourceID]sourceID) (repair, error) {
	repairCopy := repair{
		Title:            value.Title,
		PreservesMeaning: value.PreservesMeaning,
		Sensitive:        value.Sensitive,
	}
	for _, sourceRepair := range value.Sources {
		mapped, ok := sourceMap[sourceRepair.Source]
		if !ok {
			failure := diagnosticGraphReferenceError{
				kind:   diagnosticGraphUnresolvedRepairSource,
				source: sourceRepair.Source,
			}
			return repair{}, failure
		}
		out := repairSource{
			Source: mapped,
		}
		for _, edit := range sourceRepair.TextEdits {
			span, err := rebaseSpan(edit.Span, sourceMap)
			if err != nil {
				return repair{}, err
			}
			replacement, err := rebaseReplacement(edit.Replacement, sourceMap)
			if err != nil {
				return repair{}, err
			}
			out.TextEdits = append(out.TextEdits, textEdit{
				Span:        span,
				Replacement: replacement,
			})
		}

		type orderedEntryEdit struct {
			edit  entryEdit
			order int
		}
		ordered := make([]orderedEntryEdit, len(sourceRepair.EntryEdits))
		for i, edit := range sourceRepair.EntryEdits {
			ordered[i] = orderedEntryEdit{
				edit:  edit,
				order: i,
			}
		}
		slices.SortStableFunc(ordered, func(left, right orderedEntryEdit) int {
			return cmp.Or(
				cmp.Compare(left.edit.Start, right.edit.Start),
				cmp.Compare(left.edit.End, right.edit.End),
				cmp.Compare(left.order, right.order),
			)
		})
		for _, orderedEdit := range ordered {
			edit := orderedEdit.edit
			entries := make([]replacement, len(edit.Entries))
			for i, replacement := range edit.Entries {
				rebased, err := rebaseReplacement(replacement, sourceMap)
				if err != nil {
					return repair{}, err
				}
				entries[i] = rebased
			}
			if n := len(
				out.EntryEdits,
			); n != 0 && edit.Start == edit.End && out.EntryEdits[n-1].Start == edit.Start &&
				out.EntryEdits[n-1].End == edit.End {
				out.EntryEdits[n-1].Entries = append(out.EntryEdits[n-1].Entries, entries...)
			} else {
				entryEditValue := entryEdit{
					Start:   edit.Start,
					End:     edit.End,
					Entries: entries,
				}
				out.EntryEdits = append(out.EntryEdits, entryEditValue)
			}
		}
		repairCopy.Sources = append(repairCopy.Sources, out)
	}
	return repairCopy, nil
}

func rebaseReplacement(value replacement, sourceMap map[sourceID]sourceID) (replacement, error) {
	result := make(replacement, len(value))
	for i, part := range value {
		result[i] = part
		if part.Literal != nil {
			literal := *part.Literal
			result[i].Literal = &literal
		}
		if part.Copy != nil {
			span, err := rebaseSpan(*part.Copy, sourceMap)
			if err != nil {
				return nil, err
			}
			result[i].Copy = &span
		}
	}
	return result, nil
}

func rebaseSpan(value span, sourceMap map[sourceID]sourceID) (span, error) {
	mapped, ok := sourceMap[value.Source]
	if !ok {
		failure := diagnosticGraphReferenceError{
			kind:   diagnosticGraphUnresolvedSpanSource,
			source: value.Source,
		}
		return span{}, failure
	}
	value.Source = mapped
	return value, nil
}

func sortedSourceIDs(set map[sourceID]struct{}) []sourceID {
	ids := make([]sourceID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func sortedOriginIDs(set map[originID]struct{}) []originID {
	ids := make([]originID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func clonePayload(value payload) payload {
	result := payload{
		Truncated:    value.Truncated,
		Presentation: value.Presentation,
	}
	result.Diagnostics = make([]diagnostic, len(value.Diagnostics))
	for i, diagnostic := range value.Diagnostics {
		result.Diagnostics[i] = cloneDiagnostic(diagnostic)
	}
	result.Sources = make(map[sourceID]source, len(value.Sources))
	for id, source := range value.Sources {
		source.Entries = append([]string(nil), source.Entries...)
		if source.FlatText != nil {
			text := *source.FlatText
			source.FlatText = &text
		}
		result.Sources[id] = source
	}
	result.Origins = make(map[originID]origin, len(value.Origins))
	for id, origin := range value.Origins {
		origin.Parents = append([]originID(nil), origin.Parents...)
		result.Origins[id] = origin
	}
	result.Repairs = make([]repair, len(value.Repairs))
	for i, repair := range value.Repairs {
		result.Repairs[i] = cloneRepair(repair)
	}
	return result
}

func cloneDiagnostic(diagnostic diagnostic) diagnostic {
	diagnosticCopy := diagnostic
	diagnosticCopy.CommandPath = append([]string(nil), diagnostic.CommandPath...)
	diagnosticCopy.Origins = append([]originID(nil), diagnostic.Origins...)
	diagnosticCopy.Suggestions = append([]string(nil), diagnostic.Suggestions...)
	diagnosticCopy.Usage = append([]string(nil), diagnostic.Usage...)
	diagnosticCopy.RepairIndexes = append([]int(nil), diagnostic.RepairIndexes...)
	if diagnostic.Rule != nil {
		rule := *diagnostic.Rule
		rule.Subjects = append([]uint32(nil), diagnostic.Rule.Subjects...)
		rule.Operands = make([]diagnosticRuleOperand, len(diagnostic.Rule.Operands))
		for i, operand := range diagnostic.Rule.Operands {
			rule.Operands[i] = operand
			if operand.Literal != nil {
				literal := *operand.Literal
				rule.Operands[i].Literal = &literal
			}
		}
		diagnosticCopy.Rule = &rule
	}
	if diagnostic.ProviderOutput != nil {
		providerOutput := *diagnostic.ProviderOutput
		diagnosticCopy.ProviderOutput = &providerOutput
	}
	if diagnostic.Replacement != nil {
		replacement := *diagnostic.Replacement
		diagnosticCopy.Replacement = &replacement
	}
	if diagnostic.Span != nil {
		span := *diagnostic.Span
		diagnosticCopy.Span = &span
	}
	if diagnostic.SignalAction != nil {
		action := *diagnostic.SignalAction
		action.CommandPath = append([]string(nil), diagnostic.SignalAction.CommandPath...)
		if diagnostic.SignalAction.Event.Sender != nil {
			sender := *diagnostic.SignalAction.Event.Sender
			action.Event.Sender = &sender
		}
		diagnosticCopy.SignalAction = &action
	}
	if diagnostic.SignalOverflow != nil {
		overflow := *diagnostic.SignalOverflow
		diagnosticCopy.SignalOverflow = &overflow
	}
	if diagnostic.NoticeDelivery != nil {
		notice := *diagnostic.NoticeDelivery
		notice.CommandPath = append([]string(nil), diagnostic.NoticeDelivery.CommandPath...)
		diagnosticCopy.NoticeDelivery = &notice
	}
	return diagnosticCopy
}

func cloneRepair(value repair) repair {
	repairCopy := repair{
		Title:            value.Title,
		PreservesMeaning: value.PreservesMeaning,
		Sensitive:        value.Sensitive,
		Sources:          make([]repairSource, len(value.Sources)),
	}
	for i, source := range value.Sources {
		out := repairSource{
			Source: source.Source,
		}
		out.TextEdits = make([]textEdit, len(source.TextEdits))
		for j, edit := range source.TextEdits {
			out.TextEdits[j] = textEdit{
				Span:        edit.Span,
				Replacement: analysisCloneReplacement(edit.Replacement),
			}
		}
		out.EntryEdits = make([]entryEdit, len(source.EntryEdits))
		for j, edit := range source.EntryEdits {
			entries := make([]replacement, len(edit.Entries))
			for k, replacement := range edit.Entries {
				entries[k] = analysisCloneReplacement(replacement)
			}
			out.EntryEdits[j] = entryEdit{
				Start:   edit.Start,
				End:     edit.End,
				Entries: entries,
			}
		}
		repairCopy.Sources[i] = out
	}
	return repairCopy
}

func analysisCloneReplacement(value replacement) replacement {
	result := make(replacement, len(value))
	for i, part := range value {
		result[i] = part
		if part.Literal != nil {
			literal := *part.Literal
			result[i].Literal = &literal
		}
		if part.Copy != nil {
			span := *part.Copy
			result[i].Copy = &span
		}
	}
	return result
}
