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
	if err == nil {
		return false
	}
	const maximumErrorGraphNodes = 4096
	stack := []error{err}
	visited := 0
	leaves := 0
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			return false
		}
		visited++
		if visited > maximumErrorGraphNodes {
			return false
		}
		if many, ok := current.(interface{ Unwrap() []error }); ok {
			children := many.Unwrap()
			if len(children) == 0 {
				return false
			}
			for _, child := range slices.Backward(children) {
				stack = append(stack, child)
			}
			continue
		}
		if one, ok := current.(interface{ Unwrap() error }); ok {
			child := one.Unwrap()
			if child == nil {
				return false
			}
			stack = append(stack, child)
			continue
		}

		leaves++
		switch rejection := current.(type) { //nolint:errorlint // The walker already descends wrappers; leaf classification must inspect the exact current error.
		case InputRejection:
			if !validInputRejectionForTokens(rejection, tokens) {
				return false
			}
		case *InputRejection:
			if rejection == nil || !validInputRejectionForTokens(*rejection, tokens) {
				return false
			}
		default:
			return false
		}
	}
	return leaves > 0
}
