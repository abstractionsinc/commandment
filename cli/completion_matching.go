package cli

import (
	"slices"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const completionComparisonDistanceBeyondOne = 2

type completionComparisonUnit struct {
	key   int64
	start int
	end   int
}

type completionMatcher struct {
	mode             CompletionMatchMode
	caseMode         CompletionCaseMode
	ignoreDiacritics bool
	fields           []CompletionMatchField
	matchRanges      bool
	acceptAll        bool
	rejectAll        bool
	queryKeys        []int64
	substringFailure []int
}

type completionMatchScratch struct {
	units  []completionComparisonUnit
	misses []string
}

func prepareCompletionMatcher(input completionInput, query string) completionMatcher {
	matcher := completionMatcher{
		mode:             input.client.Match.Mode,
		caseMode:         effectiveCompletionCaseMode(input.client.Match.Case, query),
		ignoreDiacritics: input.client.Match.IgnoreDiacritics,
		fields:           append([]CompletionMatchField(nil), input.client.Match.Fields...),
		matchRanges:      input.client.Capabilities.MatchRanges,
	}
	if query == "" || matcher.mode == CompletionMatchModeUnfiltered {
		matcher.acceptAll = true
		return matcher
	}
	queryUnits := completionComparisonUnits(query, matcher.caseMode, matcher.ignoreDiacritics, nil)
	if len(queryUnits) == 0 {
		matcher.rejectAll = true
		return matcher
	}
	matcher.queryKeys = make([]int64, len(queryUnits))
	for i := range queryUnits {
		matcher.queryKeys[i] = queryUnits[i].key
	}
	if matcher.mode == CompletionMatchModeSubstring || matcher.mode == CompletionMatchModeFuzzy ||
		matcher.mode == CompletionMatchModeApproximate {
		matcher.substringFailure = comparisonSubstringFailure(matcher.queryKeys)
	}
	return matcher
}

func (matcher completionMatcher) matchCandidate(candidate CompletionCandidate, scratch *completionMatchScratch) (CompletionMatch, bool) {
	if matcher.acceptAll {
		return CompletionMatch{}, true
	}
	if matcher.rejectAll {
		return CompletionMatch{}, false
	}
	if scratch == nil {
		scratch = &completionMatchScratch{}
	}
	clear(scratch.misses)
	scratch.misses = scratch.misses[:0]

	matchField := func(field CompletionMatchField, index int, text string) (CompletionMatch, bool) {
		if slices.Contains(scratch.misses, text) {
			return CompletionMatch{}, false
		}
		scratch.units = completionComparisonUnits(text, matcher.caseMode, matcher.ignoreDiacritics, scratch.units[:0])
		kind, start, end, edits, ok := matcher.matchUnits(scratch.units)
		if !ok {
			scratch.misses = append(scratch.misses, text)
			return CompletionMatch{}, false
		}
		match := CompletionMatch{
			Kind:       kind,
			Field:      field,
			FieldIndex: index,
			Edits:      edits,
		}
		if matcher.matchRanges {
			switch kind {
			case CompletionMatchKindFuzzy:
				positions, fuzzyOK := comparisonSubsequencePositions(scratch.units, matcher.queryKeys)
				if fuzzyOK {
					match.Ranges = comparisonPositionRanges(scratch.units, positions)
				}
			case CompletionMatchKindApproximate:
				match.Ranges = comparisonWindowRanges(scratch.units, start, end, text)
			default:
				match.Ranges = comparisonWindowRanges(scratch.units, start, end, text)
			}
		}
		return match, true
	}

	fields := matcher.fields
	if len(fields) == 0 {
		fields = []CompletionMatchField{
			CompletionMatchFieldValue,
			CompletionMatchFieldDisplayText,
			CompletionMatchFieldSearchTerm,
		}
	}
	for _, field := range fields {
		switch field {
		case CompletionMatchFieldValue:
			if match, ok := matchField(field, 0, candidate.Value); ok {
				return match, true
			}
		case CompletionMatchFieldDisplayText:
			if match, ok := matchField(field, 0, candidate.DisplayText); ok {
				return match, true
			}
		case CompletionMatchFieldSearchTerm:
			for index, term := range candidate.SearchTerms {
				if match, ok := matchField(field, index, term); ok {
					return match, true
				}
			}
		case CompletionMatchFieldDescription:
			if match, ok := matchField(field, 0, candidate.Description); ok {
				return match, true
			}
		case CompletionMatchFieldDetail:
			if match, ok := matchField(field, 0, candidate.Detail); ok {
				return match, true
			}
		case CompletionMatchFieldUnknown:
		}
	}
	return CompletionMatch{}, false
}

func (matcher completionMatcher) matchUnits(candidate []completionComparisonUnit) (CompletionMatchKind, int, int, int, bool) {
	query := matcher.queryKeys
	if equalComparisonUnitsKeys(candidate, query) {
		return CompletionMatchKindExact, 0, len(candidate), 0, true
	}
	if matcher.mode == CompletionMatchModeExact {
		return CompletionMatchKindUnknown, 0, 0, 0, false
	}
	if hasComparisonUnitPrefix(candidate, query) {
		return CompletionMatchKindPrefix, 0, len(query), 0, true
	}
	if matcher.mode == CompletionMatchModePrefix {
		return CompletionMatchKindUnknown, 0, 0, 0, false
	}
	if start, ok := comparisonSubstringUnits(candidate, query, matcher.substringFailure); ok {
		return CompletionMatchKindSubstring, start, start + len(query), 0, true
	}
	if matcher.mode == CompletionMatchModeSubstring {
		return CompletionMatchKindUnknown, 0, 0, 0, false
	}
	if comparisonSubsequenceUnits(candidate, query) {
		return CompletionMatchKindFuzzy, 0, 0, 0, true
	}
	if matcher.mode == CompletionMatchModeFuzzy {
		return CompletionMatchKindUnknown, 0, 0, 0, false
	}
	if matcher.mode != CompletionMatchModeApproximate {
		return CompletionMatchKindUnknown, 0, 0, 0, false
	}
	start, end, edits, ok := approximateComparisonUnitWindow(query, candidate)
	if !ok {
		return CompletionMatchKindUnknown, 0, 0, 0, false
	}
	return CompletionMatchKindApproximate, start, end, edits, true
}

func effectiveCompletionCaseMode(mode CompletionCaseMode, query string) CompletionCaseMode {
	if mode != CompletionCaseSmart {
		return mode
	}
	for len(query) != 0 {
		r, size := utf8.DecodeRuneInString(query)
		if r == utf8.RuneError && size == 1 {
			query = query[1:]
			continue
		}
		if unicode.IsUpper(r) || unicode.Is(unicode.Other_Uppercase, r) {
			return CompletionCaseSensitive
		}
		query = query[size:]
	}
	return CompletionCaseInsensitive
}

func completionComparisonUnits(text string, caseMode CompletionCaseMode, ignoreDiacritics bool, storage []completionComparisonUnit) []completionComparisonUnit {
	units := storage[:0]
	for offset := 0; offset < len(text); {
		r, size := utf8.DecodeRuneInString(text[offset:])
		if r == utf8.RuneError && size == 1 {
			completionComparisonUnitValue := completionComparisonUnit{
				key:   int64(utf8.MaxRune) + 1 + int64(text[offset]),
				start: offset,
				end:   offset + 1,
			}
			units = append(units, completionComparisonUnitValue)
			offset++
			continue
		}
		start, end := offset, offset+size
		decomposed := string(r)
		if ignoreDiacritics {
			decomposed = norm.NFD.String(decomposed)
		}
		for _, unitRune := range decomposed {
			if ignoreDiacritics && unicode.Is(unicode.Properties["Diacritic"], unitRune) {
				continue
			}
			if caseMode == CompletionCaseInsensitive {
				unitRune = simpleFoldRepresentative(unitRune)
			}
			completionComparisonUnitValue2 := completionComparisonUnit{
				key:   int64(unitRune),
				start: start,
				end:   end,
			}
			units = append(units, completionComparisonUnitValue2)
		}
		offset = end
	}
	return units
}

func simpleFoldRepresentative(r rune) rune {
	best := r
	for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
		if next < best {
			best = next
		}
	}
	return best
}

func equalComparisonUnitsKeys(candidate []completionComparisonUnit, query []int64) bool {
	if len(candidate) != len(query) {
		return false
	}
	for i := range query {
		if candidate[i].key != query[i] {
			return false
		}
	}
	return true
}

func hasComparisonUnitPrefix(candidate []completionComparisonUnit, query []int64) bool {
	if len(query) > len(candidate) {
		return false
	}
	for i := range query {
		if candidate[i].key != query[i] {
			return false
		}
	}
	return true
}

func comparisonSubstringFailure(query []int64) []int {
	failure := make([]int, len(query))
	for i, matched := 1, 0; i < len(query); {
		if query[i] == query[matched] {
			matched++
			failure[i] = matched
			i++
			continue
		}
		if matched != 0 {
			matched = failure[matched-1]
			continue
		}
		i++
	}
	return failure
}

func comparisonSubstringUnits(candidate []completionComparisonUnit, query []int64, failure []int) (int, bool) {
	if len(query) == 0 {
		return 0, true
	}
	if len(query) > len(candidate) {
		return 0, false
	}
	for i, matched := 0, 0; i < len(candidate); {
		if candidate[i].key == query[matched] {
			i++
			matched++
			if matched == len(query) {
				return i - matched, true
			}
			continue
		}
		if matched != 0 {
			matched = failure[matched-1]
			continue
		}
		i++
	}
	return 0, false
}

func comparisonSubsequenceUnits(candidate []completionComparisonUnit, query []int64) bool {
	at := 0
	for _, wanted := range query {
		for at < len(candidate) && candidate[at].key != wanted {
			at++
		}
		if at == len(candidate) {
			return false
		}
		at++
	}
	return true
}

func comparisonSubsequencePositions(candidate []completionComparisonUnit, query []int64) ([]int, bool) {
	positions := make([]int, 0, len(query))
	at := 0
	for _, wanted := range query {
		for at < len(candidate) && candidate[at].key != wanted {
			at++
		}
		if at == len(candidate) {
			return nil, false
		}
		positions = append(positions, at)
		at++
	}
	return positions, true
}

func approximateComparisonUnitWindow(query []int64, candidate []completionComparisonUnit) (int, int, int, bool) {
	minLength := max(len(query)-1, 0)
	maxLength := len(query) + 1
	for start := 0; start <= len(candidate); start++ {
		for length := minLength; length <= maxLength; length++ {
			end := start + length
			if end > len(candidate) {
				continue
			}
			edits := comparisonDistanceAtMostOne(query, candidate[start:end])
			if edits == 1 {
				return start, end, edits, true
			}
		}
	}
	return 0, 0, completionComparisonDistanceBeyondOne, false
}

func comparisonDistanceAtMostOne(left []int64, right []completionComparisonUnit) int {
	delta := len(left) - len(right)
	if delta > 1 || delta < -1 {
		return completionComparisonDistanceBeyondOne
	}
	if delta == 0 {
		return equalLengthComparisonDistanceAtMostOne(left, right)
	}
	if delta == 1 {
		return comparisonDeletionDistanceAtMostOne(left, right)
	}
	return comparisonInsertionDistanceAtMostOne(left, right)
}

func equalLengthComparisonDistanceAtMostOne(left []int64, right []completionComparisonUnit) int {
	first := -1
	for i := range left {
		if left[i] == right[i].key {
			continue
		}
		if first < 0 {
			first = i
			continue
		}
		if i == first+1 && left[first] == right[i].key && left[i] == right[first].key {
			for j := i + 1; j < len(left); j++ {
				if left[j] != right[j].key {
					return completionComparisonDistanceBeyondOne
				}
			}
			return 1
		}
		return completionComparisonDistanceBeyondOne
	}
	if first < 0 {
		return 0
	}
	return 1
}

func comparisonDeletionDistanceAtMostOne(left []int64, right []completionComparisonUnit) int {
	li, ri, edits := 0, 0, 0
	for li < len(left) && ri < len(right) {
		if left[li] == right[ri].key {
			li++
			ri++
			continue
		}
		edits++
		if edits > 1 {
			return completionComparisonDistanceBeyondOne
		}
		li++
	}
	if li < len(left) {
		edits++
	}
	if edits == 1 {
		return 1
	}
	return completionComparisonDistanceBeyondOne
}

func comparisonInsertionDistanceAtMostOne(left []int64, right []completionComparisonUnit) int {
	li, ri, edits := 0, 0, 0
	for li < len(left) && ri < len(right) {
		if left[li] == right[ri].key {
			li++
			ri++
			continue
		}
		edits++
		if edits > 1 {
			return completionComparisonDistanceBeyondOne
		}
		ri++
	}
	if ri < len(right) {
		edits++
	}
	if edits == 1 {
		return 1
	}
	return completionComparisonDistanceBeyondOne
}

func comparisonPositionRanges(units []completionComparisonUnit, positions []int) []CompletionRange {
	if len(positions) == 0 {
		return nil
	}
	ranges := make([]CompletionRange, 0, len(positions))
	for _, position := range positions {
		if position < 0 || position >= len(units) {
			continue
		}
		ranges = appendCoalescedCompletionRange(ranges, CompletionRange{
			Start: units[position].start,
			End:   units[position].end,
		})
	}
	return ranges
}

func comparisonWindowRanges(units []completionComparisonUnit, start, end int, text string) []CompletionRange {
	if start < 0 || end < start || end > len(units) {
		return nil
	}
	if start == end {
		byteOffset := len(text)
		if start < len(units) {
			byteOffset = units[start].start
		}
		return []CompletionRange{{
			Start: byteOffset,
			End:   byteOffset,
		}}
	}
	ranges := make([]CompletionRange, 0, end-start)
	for i := start; i < end; i++ {
		ranges = appendCoalescedCompletionRange(ranges, CompletionRange{
			Start: units[i].start,
			End:   units[i].end,
		})
	}
	return ranges
}

func appendCoalescedCompletionRange(ranges []CompletionRange, next CompletionRange) []CompletionRange {
	if len(ranges) != 0 && ranges[len(ranges)-1].End >= next.Start {
		if next.End > ranges[len(ranges)-1].End {
			ranges[len(ranges)-1].End = next.End
		}
		return ranges
	}
	return append(ranges, next)
}
