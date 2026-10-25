package cli

import "slices"

// InputRejectionCode is an application-defined machine classification for parser
// rejection. Valid codes are nonempty lowercase kebab-case strings.
type InputRejectionCode string

// InputRejection is a structured parser rejection returned by an application input parser.
// Construct it with Rejection.Message. Invalid option combinations produce the zero/invalid
// value, which the Commandment treats as malformed parser output.
//
//nolint:errname // Current canonical public API fixes InputRejection; rename only with synchronized API authority.
type InputRejection struct {
	code      InputRejectionCode
	message   string
	sensitive bool

	rangeValue   inputRejectionRange
	rangePresent bool
	expected     []string
	suggestions  []string
	replacement  []string
	replaceSet   bool
	hint         string
	hintSet      bool
	valid        bool
}

type inputRejectionRange struct {
	firstToken int
	firstByte  int
	lastToken  int
	lastByte   int
}

// Error returns source-oriented parser rejection text. Sensitive rejections return only a safe
// generic message; locale-aware presentation is resolved later by the owning Commandment
// operation.
func (rejection InputRejection) Error() string {
	if rejection.sensitive {
		return "cli: input rejected"
	}
	return rejection.message
}

type rejectionOptionKind uint8

const (
	rejectionOptionUnknown rejectionOptionKind = iota
	rejectionOptionRange
	rejectionOptionExpected
	rejectionOptionSuggest
	rejectionOptionReplacement
	rejectionOptionHint
	rejectionOptionSensitive
)

type rejectionOption struct {
	kind       rejectionOptionKind
	inputRange inputRejectionRange
	values     []string
	text       string
}

// rejectionConstructors is the namespace behind Rejection. Its methods construct the
// validated, immutable option values accepted by Message.
type rejectionConstructors struct{}

// Rejection exposes constructors for structured InputRejection values and options.
var Rejection rejectionConstructors

// Message constructs an InputRejection with code, source message, and options. A
// malformed code, duplicate/incompatible option, empty nonsensitive message, or
// replacement without a range yields the zero/invalid rejection.
func (rejectionConstructors) Message(code InputRejectionCode, message string, options ...rejectionOption) InputRejection {
	rejection := InputRejection{
		code:    code,
		message: message,
		valid:   validInputRejectionCode(code),
	}
	if !rejection.valid {
		return InputRejection{}
	}

	for _, option := range options {
		switch option.kind {
		case rejectionOptionRange:
			if rejection.rangePresent || !validInputRejectionRange(option.inputRange) {
				return InputRejection{}
			}
			rejection.rangeValue = option.inputRange
			rejection.rangePresent = true
		case rejectionOptionExpected:
			if hasExactDuplicate(option.values) {
				return InputRejection{}
			}
			for _, existing := range rejection.expected {
				if slices.Contains(option.values, existing) {
					return InputRejection{}
				}
			}
			rejection.expected = append(rejection.expected, option.values...)
		case rejectionOptionSuggest:
			if hasExactDuplicate(option.values) {
				return InputRejection{}
			}
			for _, existing := range rejection.suggestions {
				if slices.Contains(option.values, existing) {
					return InputRejection{}
				}
			}
			rejection.suggestions = append(rejection.suggestions, option.values...)
		case rejectionOptionReplacement:
			if rejection.replaceSet {
				return InputRejection{}
			}
			rejection.replacement = append([]string(nil), option.values...)
			rejection.replaceSet = true
		case rejectionOptionHint:
			if rejection.hintSet || option.text == "" {
				return InputRejection{}
			}
			rejection.hint = option.text
			rejection.hintSet = true
		case rejectionOptionSensitive:
			if rejection.sensitive {
				return InputRejection{}
			}
			rejection.sensitive = true
		default:
			return InputRejection{}
		}
	}
	if rejection.replaceSet && !rejection.rangePresent {
		return InputRejection{}
	}
	if !rejection.sensitive && rejection.message == "" {
		return InputRejection{}
	}
	return rejection
}

// At attaches a half-open byte range [start, end) within one rejected token.
func (rejectionConstructors) At(token, start, end int) rejectionOption {
	inputRange := inputRejectionRange{
		firstToken: token,
		firstByte:  start,
		lastToken:  token,
		lastByte:   end,
	}
	option := rejectionOption{
		kind:       rejectionOptionRange,
		inputRange: inputRange,
	}
	return option
}

// Across attaches an ordered parser-local range that may span tokens. Token indexes
// and byte offsets are zero-based, and the ending byte offset is exclusive.
func (rejectionConstructors) Across(firstToken, firstByte, lastToken, lastByte int) rejectionOption {
	inputRange := inputRejectionRange{
		firstToken: firstToken,
		firstByte:  firstByte,
		lastToken:  lastToken,
		lastByte:   lastByte,
	}
	option := rejectionOption{
		kind:       rejectionOptionRange,
		inputRange: inputRange,
	}
	return option
}

// Expected records distinct values that would have satisfied the parser.
func (rejectionConstructors) Expected(values ...string) rejectionOption {
	option := rejectionOption{
		kind:   rejectionOptionExpected,
		values: append([]string(nil), values...),
	}
	return option
}

// Suggest records distinct alternative values suitable for presentation.
func (rejectionConstructors) Suggest(values ...string) rejectionOption {
	option := rejectionOption{
		kind:   rejectionOptionSuggest,
		values: append([]string(nil), values...),
	}
	return option
}

// ReplaceWith records replacement tokens for the rejection range. It is valid only
// when Message also receives At or Across.
func (rejectionConstructors) ReplaceWith(tokens ...string) rejectionOption {
	option := rejectionOption{
		kind:   rejectionOptionReplacement,
		values: append([]string(nil), tokens...),
	}
	return option
}

// Hint attaches one nonempty human-facing hint to the rejection.
func (rejectionConstructors) Hint(text string) rejectionOption {
	option := rejectionOption{
		kind: rejectionOptionHint,
		text: text,
	}
	return option
}

// Sensitive marks rejection presentation as protected. The Commandment substitutes safe
// presentation and excludes sensitive rejection message/hint text from localization lookup and
// audit surfaces.
func (rejectionConstructors) Sensitive() rejectionOption {
	option := rejectionOption{
		kind: rejectionOptionSensitive,
	}
	return option
}

func validInputRejectionRange(value inputRejectionRange) bool {
	if value.firstToken < 0 || value.lastToken < value.firstToken || value.firstByte < 0 || value.lastByte < 0 {
		return false
	}
	return value.firstToken != value.lastToken || value.firstByte <= value.lastByte
}

func hasExactDuplicate(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func validInputRejectionCode(code InputRejectionCode) bool {
	text := string(code)
	if text == "" {
		return false
	}
	segmentStart := true
	for i := range len(text) {
		c := text[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			segmentStart = false
		case c == '-':
			if segmentStart || i == len(text)-1 {
				return false
			}
			segmentStart = true
		default:
			return false
		}
	}
	return !segmentStart
}

func validInputRejectionForTokens(rejection InputRejection, tokens []string) bool {
	if !rejection.valid {
		return false
	}
	if rejection.rangePresent {
		rangeValue := rejection.rangeValue
		if rangeValue.firstToken >= len(tokens) || rangeValue.lastToken >= len(tokens) ||
			rangeValue.firstByte > len(tokens[rangeValue.firstToken]) ||
			rangeValue.lastByte > len(tokens[rangeValue.lastToken]) {
			return false
		}
	}
	return true
}

func collectInputRejections(err error, tokens []string) bool {
	_, ok := inputRejectionLeaves(err, tokens)
	return ok
}

func inputRejectionLeaves(err error, tokens []string) ([]InputRejection, bool) {
	if err == nil {
		return nil, false
	}
	const maximumErrorGraphNodes = 4096
	stack := []error{err}
	visited := 0
	var leaves []InputRejection
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			return nil, false
		}
		visited++
		if visited > maximumErrorGraphNodes {
			return nil, false
		}
		if many, ok := current.(interface{ Unwrap() []error }); ok {
			children := many.Unwrap()
			if len(children) == 0 {
				return nil, false
			}
			for _, child := range slices.Backward(children) {
				stack = append(stack, child)
			}
			continue
		}
		if one, ok := current.(interface{ Unwrap() error }); ok {
			child := one.Unwrap()
			if child == nil {
				return nil, false
			}
			stack = append(stack, child)
			continue
		}

		var rejection InputRejection
		switch value := current.(type) { //nolint:errorlint // The walker already descends wrappers; leaf classification must inspect the exact current error.
		case InputRejection:
			rejection = value
		case *InputRejection:
			if value == nil {
				return nil, false
			}
			rejection = *value
		default:
			return nil, false
		}
		if !validInputRejectionForTokens(rejection, tokens) {
			return nil, false
		}
		rejection.expected = slices.Clone(rejection.expected)
		rejection.suggestions = slices.Clone(rejection.suggestions)
		rejection.replacement = slices.Clone(rejection.replacement)
		leaves = append(leaves, rejection)
	}
	return leaves, len(leaves) != 0
}

type normalizedInputRejection struct {
	code        InputRejectionCode
	message     string
	hint        string
	sensitive   bool
	origins     []InputOriginID
	source      InputSourceID
	input       string
	primarySpan semanticInputSpan
	primarySet  bool
	expected    []string
	suggestions []string
	spans       []semanticInputSpan
	replacement []string
	replaceSet  bool
}

type normalizedInputRejectionError struct {
	cause      error
	rejections []normalizedInputRejection
}

func (err *normalizedInputRejectionError) Error() string { return err.cause.Error() }
func (err *normalizedInputRejectionError) Unwrap() error { return err.cause }

func normalizedInputRejections(err error) ([]normalizedInputRejection, bool) {
	for err != nil {
		if normalized, ok := err.(*normalizedInputRejectionError); ok { //nolint:errorlint // Exact private phase-boundary carrier is intentional.
			return cloneNormalizedInputRejections(normalized.rejections), len(normalized.rejections) != 0
		}
		one, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = one.Unwrap()
	}
	return nil, false
}

func cloneNormalizedInputRejections(values []normalizedInputRejection) []normalizedInputRejection {
	result := make([]normalizedInputRejection, len(values))
	for index, value := range values {
		result[index] = value
		result[index].origins = slices.Clone(value.origins)
		result[index].expected = slices.Clone(value.expected)
		result[index].suggestions = slices.Clone(value.suggestions)
		result[index].spans = slices.Clone(value.spans)
		result[index].replacement = slices.Clone(value.replacement)
	}
	return result
}

func normalizeRejectedConversion(analysis *analysisState, outcome conversionOutcome, tokens []string, coordinates []conversionRangeCoordinate) conversionOutcome {
	if outcome.Kind() != conversionOutcomeRejected {
		return outcome
	}
	leaves, ok := inputRejectionLeaves(outcome.Err(), tokens)
	if !ok {
		return outcome
	}
	if len(tokens) != len(coordinates) {
		return failedConversion(outcome.Err())
	}
	normalized := make([]normalizedInputRejection, len(leaves))
	for index, rejection := range leaves {
		value, valid := normalizeInputRejection(analysis, rejection, tokens, coordinates)
		if !valid {
			return failedConversion(outcome.Err())
		}
		normalized[index] = value
	}
	outcome.err = &normalizedInputRejectionError{
		cause:      outcome.Err(),
		rejections: normalized,
	}
	return outcome
}

func normalizeInputRejection(analysis *analysisState, rejection InputRejection, tokens []string, coordinates []conversionRangeCoordinate) (normalizedInputRejection, bool) {
	result := normalizedInputRejection{
		code:        rejection.code,
		message:     rejection.message,
		hint:        rejection.hint,
		sensitive:   rejection.sensitive,
		expected:    slices.Clone(rejection.expected),
		suggestions: slices.Clone(rejection.suggestions),
		replacement: slices.Clone(rejection.replacement),
		replaceSet:  rejection.replaceSet,
	}
	for _, coordinate := range coordinates {
		if coordinate.origin.present() && analysis.originSensitive(coordinate.origin) {
			result.sensitive = true
			break
		}
	}
	firstToken := 0
	lastToken := len(tokens) - 1
	if rejection.rangePresent {
		firstToken = rejection.rangeValue.firstToken
		lastToken = rejection.rangeValue.lastToken
	}
	for tokenIndex := firstToken; tokenIndex <= lastToken; tokenIndex++ {
		if tokenIndex < 0 || tokenIndex >= len(tokens) || tokenIndex >= len(coordinates) {
			return normalizedInputRejection{}, false
		}
		coordinate := coordinates[tokenIndex]
		if coordinate.origin.present() {
			result.origins = appendUniqueOrigin(result.origins, coordinate.origin)
		}
	}
	if result.sensitive {
		result.message = ""
		result.hint = ""
		result.expected = nil
		result.suggestions = nil
		result.replacement = nil
		result.replaceSet = false
	}
	if firstToken >= 0 && firstToken < len(tokens) && firstToken < len(coordinates) {
		coordinate := coordinates[firstToken]
		span, ok := normalizedInputRejectionSpan(
			analysis,
			tokens[firstToken],
			coordinate,
			0,
			len(tokens[firstToken]),
		)
		if !ok {
			return normalizedInputRejection{}, false
		}
		result.source = span.Source
		result.primarySpan = span
		result.primarySet = true
		if !result.sensitive {
			result.input = tokens[firstToken]
		}
	}
	if !rejection.rangePresent {
		return result, true
	}

	rangeValue := rejection.rangeValue
	for tokenIndex := rangeValue.firstToken; tokenIndex <= rangeValue.lastToken; tokenIndex++ {
		coordinate := coordinates[tokenIndex]
		start, end := 0, len(tokens[tokenIndex])
		if tokenIndex == rangeValue.firstToken {
			start = rangeValue.firstByte
		}
		if tokenIndex == rangeValue.lastToken {
			end = rangeValue.lastByte
		}
		if start < 0 || end < start || end > len(tokens[tokenIndex]) {
			return normalizedInputRejection{}, false
		}
		span, ok := normalizedInputRejectionSpan(analysis, tokens[tokenIndex], coordinate, start, end)
		if !ok {
			return normalizedInputRejection{}, false
		}
		result.spans = append(result.spans, span)
	}
	return result, true
}

func normalizedInputRejectionSpan(analysis *analysisState, token string, coordinate conversionRangeCoordinate, start, end int) (semanticInputSpan, bool) {
	origin, ok := analysis.originRecord(coordinate.origin)
	if !ok || coordinate.start < 0 || coordinate.end < coordinate.start || coordinate.end-coordinate.start != len(token) ||
		start < 0 || end < start || end > len(token) {
		return semanticInputSpan{}, false
	}
	mappedStart := origin.span.Start + coordinate.start + start
	mappedEnd := origin.span.Start + coordinate.start + end
	if mappedStart < origin.span.Start || mappedEnd > origin.span.End {
		return semanticInputSpan{}, false
	}
	result := semanticInputSpan{
		Source: origin.span.Source,
		Entry:  origin.span.Entry,
		Start:  mappedStart,
		End:    mappedEnd,
	}
	return result, true
}

func attachNormalizedInputRejectionRepair(payload *payload, diagnostic *diagnostic, rejection normalizedInputRejection) {
	if payload == nil || diagnostic == nil || diagnostic.Sensitive || !rejection.replaceSet || len(rejection.spans) == 0 {
		return
	}
	spans := make([]span, len(rejection.spans))
	for index, input := range rejection.spans {
		spans[index] = span{
			Source: sourceID(input.Source.uint32()),
			Entry:  input.Entry,
			Start: analysisPosition{
				Offset: input.Start,
			},
			End: analysisPosition{
				Offset: input.End,
			},
		}
	}
	candidate, ok := normalizedInputRejectionRepair(*payload, spans, rejection.replacement)
	if !ok {
		return
	}
	index := len(payload.Repairs)
	payload.Repairs = append(payload.Repairs, candidate)
	diagnostic.RepairIndexes = append(diagnostic.RepairIndexes, index)
}

func normalizedInputRejectionRepair(payload payload, spans []span, tokens []string) (repair, bool) {
	if len(spans) == 0 {
		return repair{}, false
	}
	firstSource := spans[0].Source
	inputSource, ok := payload.Sources[firstSource]
	if !ok || inputSource.Revision == "" {
		return repair{}, false
	}
	for _, item := range spans[1:] {
		if item.Source != firstSource {
			return repair{}, false
		}
	}

	sourceRepair := repairSource{
		Source: firstSource,
	}
	switch {
	case len(spans) == 1 && inputSource.FlatText != nil:
		if len(tokens) > 1 {
			return repair{}, false
		}
		sourceRepair.TextEdits = []textEdit{{
			Span:        spans[0],
			Replacement: singleInputReplacement(tokens),
		}}
	case len(spans) == 1:
		item := spans[0]
		if item.Entry < 0 || item.Entry >= len(inputSource.Entries) {
			return repair{}, false
		}
		switch {
		case len(tokens) <= 1:
			sourceRepair.TextEdits = []textEdit{{
				Span:        item,
				Replacement: singleInputReplacement(tokens),
			}}
		case item.Start.Offset == 0 && item.End.Offset == len(inputSource.Entries[item.Entry]):
			sourceRepair.EntryEdits = []entryEdit{{
				Start:   item.Entry,
				End:     item.Entry + 1,
				Entries: inputReplacementEntries(tokens),
			}}
		default:
			return repair{}, false
		}
	default:
		start, end, ok := contiguousInputEntries(inputSource, spans)
		if !ok {
			return repair{}, false
		}
		sourceRepair.EntryEdits = []entryEdit{{
			Start:   start,
			End:     end,
			Entries: inputReplacementEntries(tokens),
		}}
	}
	result := repair{
		Title:   "replace rejected input",
		Sources: []repairSource{sourceRepair},
	}
	return result, true
}

func contiguousInputEntries(inputSource source, spans []span) (int, int, bool) {
	if inputSource.FlatText != nil || len(spans) == 0 {
		return 0, 0, false
	}
	start := spans[0].Entry
	if start < 0 {
		return 0, 0, false
	}
	for index, item := range spans {
		entry := start + index
		if item.Entry != entry || entry >= len(inputSource.Entries) || item.Start.Offset != 0 ||
			item.End.Offset != len(inputSource.Entries[entry]) {
			return 0, 0, false
		}
	}
	return start, start + len(spans), true
}

func singleInputReplacement(tokens []string) replacement {
	if len(tokens) == 0 {
		return nil
	}
	literal := tokens[0]
	part := replacementPart{
		Literal: &literal,
	}
	return replacement{part}
}

func inputReplacementEntries(tokens []string) []replacement {
	result := make([]replacement, len(tokens))
	for index, token := range tokens {
		literal := token
		result[index] = replacement{{
			Literal: &literal,
		}}
	}
	return result
}
