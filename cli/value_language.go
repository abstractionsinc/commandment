package cli

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
)

type conversionMemo struct {
	mu      sync.Mutex
	results map[conversionEvaluationKey]*frozenConversionEvaluation
}

type conversionEvaluationKind uint8

const (
	conversionEvaluationValue conversionEvaluationKind = iota + 1
	conversionEvaluationFlagTokenLanguage
	conversionEvaluationArgumentTokenLanguage
)

type conversionEvaluationKey struct {
	kind        conversionEvaluationKind
	destination ValueID
	flag        FlagID
	term        ArgumentTermID
	rangeKey    string
}

type conversionRangeCoordinate struct {
	origin        InputOriginID
	physicalIndex int
	start         int
	end           int
}

type frozenConversionEvaluation struct {
	done       chan struct{}
	outcome    conversionOutcome
	panicked   bool
	panicValue any
}

func (operation *invocation) evaluateConversion(analysis *analysisState, key conversionEvaluationKey, evaluate func() conversionOutcome) conversionOutcome {
	if operation == nil || analysis == nil || evaluate == nil {
		panic("invocation: invalid parser evaluation")
	}
	release := analysis.work.enter()
	defer release()

	analysis.conversions.mu.Lock()
	if analysis.conversions.results == nil {
		analysis.conversions.results = make(map[conversionEvaluationKey]*frozenConversionEvaluation)
	}
	frozen := analysis.conversions.results[key]
	leader := frozen == nil
	if leader {
		frozen = &frozenConversionEvaluation{
			done: make(chan struct{}),
		}
		analysis.conversions.results[key] = frozen
	}
	analysis.conversions.mu.Unlock()

	if leader {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					frozen.panicked = true
					frozen.panicValue = recovered
				}
				close(frozen.done)
			}()
			frozen.outcome = evaluate().Clone()
		}()
	} else {
		<-frozen.done
	}
	if frozen.panicked {
		panic(frozen.panicValue)
	}
	return frozen.outcome.Clone()
}

func conversionRangeKey(coordinates []conversionRangeCoordinate) string {
	var builder strings.Builder
	builder.WriteString(strconv.Itoa(len(coordinates)))
	builder.WriteByte('|')
	for _, coordinate := range coordinates {
		builder.WriteString(strconv.FormatUint(uint64(coordinate.origin.uint32()), 10))
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(coordinate.physicalIndex))
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(coordinate.start))
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(coordinate.end))
		builder.WriteByte('|')
	}
	return builder.String()
}

func (operation *invocation) argumentConversionRange(analysis *analysisState, indices []int) ([]conversionRangeCoordinate, bool) {
	if operation == nil || analysis == nil {
		return nil, false
	}
	coordinates := make([]conversionRangeCoordinate, len(indices))
	for i, index := range indices {
		if index < 0 || index >= len(analysis.args) {
			return nil, false
		}
		origin, ok := analysis.argumentOrigin(index)
		if !ok {
			return nil, false
		}
		coordinates[i] = conversionRangeCoordinate{
			origin:        origin.ID(),
			physicalIndex: index,
			start:         0,
			end:           len(analysis.args[index]),
		}
	}
	return coordinates, true
}

func (operation *invocation) flagConversionRange(analysis *analysisState, occurrence flagOccurrence) ([]conversionRangeCoordinate, bool) {
	if operation == nil || analysis == nil {
		return nil, false
	}
	if occurrence.Attached == "" {
		return operation.argumentConversionRange(analysis, occurrence.ValueIndices)
	}
	index := occurrence.TokenIndex
	if index < 0 || index >= len(analysis.args) || !strings.HasSuffix(analysis.args[index], occurrence.Attached) {
		return nil, false
	}
	origin, ok := analysis.argumentOrigin(index)
	if !ok {
		return nil, false
	}
	end := len(analysis.args[index])
	coordinate := conversionRangeCoordinate{
		origin:        origin.ID(),
		physicalIndex: index,
		start:         end - len(occurrence.Attached),
		end:           end,
	}
	return []conversionRangeCoordinate{coordinate}, true
}

func (operation *invocation) evaluateValueLanguage(analysis *analysisState, destination ValueID, language valueLanguage, text string, index int) conversionOutcome {
	coordinates, ok := operation.argumentConversionRange(analysis, []int{index})
	if !ok {
		return failedConversion(errInvalidValueLanguageInput)
	}
	key := conversionEvaluationKey{
		kind:        conversionEvaluationValue,
		destination: destination,
		rangeKey:    conversionRangeKey(coordinates),
	}
	return operation.evaluateConversion(analysis, key, func() conversionOutcome {
		outcome := language.Evaluate(text)
		return normalizeRejectedConversion(analysis, outcome, []string{text}, coordinates)
	})
}

func (operation *invocation) evaluateFlagTokenLanguage(ctx context.Context, analysis *analysisState, flag FlagID, language tokenLanguage, occurrence flagOccurrence, tokens []string) conversionOutcome {
	coordinates, ok := operation.flagConversionRange(analysis, occurrence)
	if !ok {
		return failedConversion(errInvalidValueLanguageInput)
	}
	key := conversionEvaluationKey{
		kind:     conversionEvaluationFlagTokenLanguage,
		flag:     flag,
		rangeKey: conversionRangeKey(coordinates),
	}
	return operation.evaluateConversion(
		analysis,
		key,
		func() conversionOutcome {
			outcome := language.Evaluate(ctx, tokens)
			return normalizeRejectedConversion(analysis, outcome, tokens, coordinates)
		},
	)
}

func (operation *invocation) evaluateArgumentTokenLanguage(ctx context.Context, analysis *analysisState, term ArgumentTermID, language tokenLanguage, indices []int, tokens []string) conversionOutcome {
	coordinates, ok := operation.argumentConversionRange(analysis, indices)
	if !ok {
		return failedConversion(errInvalidValueLanguageInput)
	}
	key := conversionEvaluationKey{
		kind:     conversionEvaluationArgumentTokenLanguage,
		term:     term,
		rangeKey: conversionRangeKey(coordinates),
	}
	return operation.evaluateConversion(
		analysis,
		key,
		func() conversionOutcome {
			outcome := language.Evaluate(ctx, tokens)
			return normalizeRejectedConversion(analysis, outcome, tokens, coordinates)
		},
	)
}

// ValueLanguage is a typed scalar language accepted by value-producing declarations. Built-in
// languages, ValueCodec values, and ValueDomain values implement this contract.
type ValueLanguage[T any] interface {
	valueLanguageFor(value *T)
}

// ValueCodec defines a named scalar language that parses one string into T, formats T back to
// text, and snapshots retained values. Its zero value is invalid; Name must be nonempty and Parse,
// Format, and Snapshot must be non-nil.
type ValueCodec[T any] struct {
	Name     string
	Parse    func(string) (T, error)
	Format   func(T) string
	Snapshot func(T) T
}

func (ValueCodec[T]) valueLanguageFor(*T) {}

type builtinValueLanguage[T any] struct{ name string }

func (builtinValueLanguage[T]) valueLanguageFor(*T) {}

// TokenLanguage defines a bounded token language that parses and formats one T. Its zero value is
// invalid; Name, a CountRange with minimum at least one, Parse, and Format are required.
type TokenLanguage[T any] struct {
	Name   string
	Count  CountRange
	Syntax []string
	Parse  func(context.Context, []string) (T, error)
	Format func(T) []string
}

// TokenLanguageCompleter computes completion contributions for a TokenLanguage using the current
// command configuration and token-language completion request.
type TokenLanguageCompleter[C any] func(context.Context, *C, TokenLanguageCompletionRequest) (CompletionContribution, error)

// TokenLanguageCompletionRequest describes one bounded-language completion call. Target identifies
// the exact producer; Tokens contains complete owned child tokens before the current token; Text,
// Cursor, and Replace describe the current token, while Selection, WorkingDirectory, Trigger, and
// Limit carry the surrounding completion context.
type TokenLanguageCompletionRequest struct {
	Target           CompletionTarget
	Selection        CommandSelection
	Tokens           []string
	Text             string
	Cursor           int
	Replace          CompletionRange
	WorkingDirectory string
	Trigger          CompletionTrigger
	Limit            int
}

// KeyValueLanguage describes keyed values parsed from a key language, value language, and
// separator, with optional key-only semantics. Construct one with Value.Keyed; its zero value is
// not a valid value language.
type KeyValueLanguage[K comparable, V any] struct {
	_     [0]func()
	state *keyValueLanguageState
}

type keyValueLanguageState struct {
	key          any
	value        any
	separator    string
	separatorSet bool
	keyOnly      bool
	keyOnlyValue any
	valid        bool
}

// KeyOnly returns a copy of the keyed language that accepts a key without a separator and maps
// that key to value. Applying KeyOnly more than once makes the language invalid at compilation.
func (language KeyValueLanguage[K, V]) KeyOnly(value V) KeyValueLanguage[K, V] {
	if language.state == nil {
		return language
	}
	state := *language.state
	if state.keyOnly {
		state.valid = false
	}
	state.keyOnly = true
	state.keyOnlyValue = value
	keyValueLanguage := KeyValueLanguage[K, V]{
		state: &state,
	}
	return keyValueLanguage
}

// SeparatedBy returns a copy of the keyed language using separator between encoded keys and values.
// An empty separator or repeated separator configuration makes the language invalid at compilation.
func (language KeyValueLanguage[K, V]) SeparatedBy(separator string) KeyValueLanguage[K, V] {
	if language.state == nil {
		return language
	}
	state := *language.state
	if separator == "" || state.separatorSet {
		state.valid = false
	} else {
		state.separator = separator
		state.separatorSet = true
	}
	keyValueLanguage := KeyValueLanguage[K, V]{
		state: &state,
	}
	return keyValueLanguage
}

var (
	errInvalidValueLanguage = errors.New("cli: invalid value language")
	errInvalidTokenLanguage = errors.New("cli: invalid token language")
)

type normalizedValueLanguageKind uint8

const (
	normalizedValueLanguageUnknown normalizedValueLanguageKind = iota
	normalizedValueLanguageBuiltin
	normalizedValueLanguageCodec
	normalizedValueLanguageDomain
)

type authoringDomainUse struct {
	state *valueDomainState
	kind  ValueDomainUseKind
}

type domainLanguageEntry[T any] struct {
	name      string
	value     T
	canonical bool
}

type normalizedValueLanguage[T any] struct {
	kind     normalizedValueLanguageKind
	name     string
	parse    func(string) (T, error)
	format   func(T) string
	snapshot func(T) T
	domain   *valueDomainState
	builtin  string
}

func normalizeValueLanguage[T any](language ValueLanguage[T]) (normalizedValueLanguage[T], error) {
	switch language := language.(type) {
	case builtinValueLanguage[T]:
		parse, format, ok := builtinValueLanguageFunctions[T](language.name)
		if !ok {
			return normalizedValueLanguage[T]{}, errInvalidValueLanguage
		}
		result := normalizedValueLanguage[T]{
			kind:     normalizedValueLanguageBuiltin,
			name:     language.name,
			builtin:  language.name,
			parse:    parse,
			format:   format,
			snapshot: func(value T) T { return value },
		}
		return result, nil
	case ValueCodec[T]:
		if language.Name == "" || language.Parse == nil || language.Format == nil || language.Snapshot == nil {
			return normalizedValueLanguage[T]{}, errInvalidValueLanguage
		}
		record := normalizedValueLanguage[T]{
			kind:     normalizedValueLanguageCodec,
			name:     language.Name,
			parse:    language.Parse,
			format:   language.Format,
			snapshot: language.Snapshot,
		}
		return record, nil
	default:
		domain, ok := any(language).(interface {
			valueDomainStateFor() *valueDomainState
			valueDomainFor(value *T)
		})
		if !ok {
			return normalizedValueLanguage[T]{}, errInvalidValueLanguage
		}
		state := domain.valueDomainStateFor()
		if !validValueDomainState(state) {
			return normalizedValueLanguage[T]{}, errInvalidValueLanguage
		}
		if state.snapshot == nil {
			return normalizedValueLanguage[T]{}, errInvalidValueLanguage
		}
		built := normalizedValueLanguage[T]{
			kind:   normalizedValueLanguageDomain,
			domain: state,
			snapshot: func(value T) T {
				snap, ok := state.snapshot(value)
				if !ok {
					return value
				}
				typed, ok := snap.(T)
				if !ok {
					return value
				}
				return typed
			},
		}
		return built, nil
	}
}

func builtinValueLanguageFunctions[T any](name string) (func(string) (T, error), func(T) string, bool) {
	parse := func(text string) (T, error) {
		var raw any
		var err error
		switch name {
		case builtinValueLanguageNameBool:
			raw, err = strconv.ParseBool(text)
		case builtinValueLanguageNameInt:
			raw, err = strconv.Atoi(text)
		case builtinValueLanguageNameUint:
			var parsed uint64
			parsed, err = strconv.ParseUint(text, 10, 0)
			if err == nil {
				raw = uint(parsed)
			}
		case builtinValueLanguageNameString:
			raw = text
		default:
			err = errInvalidValueLanguage
		}
		if err != nil {
			var zero T
			return zero, err
		}
		value, ok := raw.(T)
		if !ok {
			var zero T
			return zero, errInvalidValueLanguage
		}
		return value, nil
	}
	format := func(value T) string {
		switch typed := any(value).(type) {
		case bool:
			return strconv.FormatBool(typed)
		case int:
			return strconv.Itoa(typed)
		case uint:
			return strconv.FormatUint(uint64(typed), 10)
		case string:
			return typed
		default:
			return ""
		}
	}
	switch name {
	case builtinValueLanguageNameBool, builtinValueLanguageNameInt, builtinValueLanguageNameUint, builtinValueLanguageNameString:
		return parse, format, true
	default:
		return nil, nil, false
	}
}

// eraseEffectiveDomainValueLanguage snapshots only entries that are effective
// for the application's evaluated release. The immutable semantic language is
// therefore the same membership authority used by runtime parsing and
// canonical formatting; lifecycle-inactive names cannot leak through the
// authoring-time language snapshot.
func eraseEffectiveDomainValueLanguage(state *valueDomainState, scheme ReleaseScheme) (valueLanguage, bool) {
	if !validValueDomainState(state) {
		return valueLanguage{}, false
	}
	if state.dynamic != nil {
		return acceptDynamicDomainValueLanguage(), true
	}
	type entrySnapshot struct {
		name      string
		value     any
		canonical bool
	}
	effective := make([]bool, len(state.entries))
	for index, entry := range state.entries {
		lifecycle := entry.lifecycle
		LifecycleState, err := evaluateLifecycle(scheme, lifecycle, "")
		if err != nil {
			return valueLanguage{}, false
		}
		effective[index] = LifecycleState == LifecycleStateCurrent ||
			LifecycleState == LifecycleStateDeprecated
	}

	entries := make([]entrySnapshot, 0, len(state.entries))
	for index, entry := range state.entries {
		if !effective[index] {
			continue
		}
		switch entry.kind {
		case domainDeclarationValue:
			entrySnapshotValue := entrySnapshot{
				name:      entry.name,
				value:     entry.value,
				canonical: true,
			}
			entries = append(entries, entrySnapshotValue)
		case domainDeclarationAlias:
			if entry.target < 0 || entry.target >= len(state.entries) || !effective[entry.target] {
				return valueLanguage{}, false
			}
			canonical := state.entries[entry.target]
			if canonical.kind != domainDeclarationValue {
				return valueLanguage{}, false
			}
			entries = append(entries, entrySnapshot{
				name:  entry.name,
				value: canonical.value,
			})
		default:
			return valueLanguage{}, false
		}
	}

	language := acceptValueLanguage(
		valueLanguageKindDomain,
		"",
		func(text string) (any, error) {
			for _, entry := range entries {
				if entry.name == text {
					return entry.value, nil
				}
			}
			return nil, errInvalidValueLanguage
		},
		func(value any) (string, bool) {
			for _, entry := range entries {
				if entry.canonical && state.equal.Equal(entry.value, value) {
					return entry.name, true
				}
			}
			return "", false
		},
	)
	if state.snapshot == nil {
		return valueLanguage{}, false
	}
	language = language.WithSnapshot(state.snapshot)
	return language, language.Valid()
}

func eraseNormalizedValueLanguage[T any](language normalizedValueLanguage[T]) (valueLanguage, bool) {
	if language.kind == normalizedValueLanguageDomain {
		return eraseDomainValueLanguage[T](language.domain)
	}
	if language.kind != normalizedValueLanguageBuiltin && language.kind != normalizedValueLanguageCodec {
		return valueLanguage{}, false
	}
	if language.parse == nil || language.format == nil || language.snapshot == nil {
		return valueLanguage{}, false
	}
	kind := valueLanguageKindCodec
	if language.kind == normalizedValueLanguageBuiltin {
		kind = valueLanguageKindBuiltin
	}
	erased := acceptValueLanguageWithEvaluation(
		kind,
		language.name,
		func(text string) conversionOutcome {
			value, err := language.parse(text)
			if err != nil {
				if language.kind == normalizedValueLanguageBuiltin {
					return rejectedConversion(err)
				}
				if collectInputRejections(err, []string{text}) {
					return rejectedConversion(err)
				}
				return failedConversion(err)
			}
			return acceptedConversion(value)
		},
		func(value any) (string, bool) {
			typed, ok := value.(T)
			if !ok {
				return "", false
			}
			return language.format(typed), true
		},
	)
	erased = erased.WithSnapshot(func(value any) (any, bool) {
		typed, ok := value.(T)
		if !ok {
			return nil, false
		}
		return language.snapshot(typed), true
	})
	return erased, erased.Valid()
}

func eraseNormalizedTokenLanguage[T any](language normalizedTokenLanguage[T]) (tokenLanguage, bool) {
	if _, ok := prepareCountRange(
		language.count,
	); !ok || language.name == "" || language.parse == nil ||
		language.format == nil {
		return tokenLanguage{}, false
	}
	return compileTokenLanguage(
		language.name,
		func(ctx context.Context, tokens []string) conversionOutcome {
			value, err := language.parse(ctx, append([]string(nil), tokens...))
			if err != nil {
				if collectInputRejections(err, tokens) {
					return rejectedConversion(err)
				}
				return failedConversion(err)
			}
			// Parse owns the returned T. TokenLanguage has no caller-retained input value, so
			// its explicit snapshot policy is typed assignment rather than shape discovery.
			frozen := value
			return acceptedTokenConversion(frozen, func() []string {
				return append([]string(nil), language.format(frozen)...)
			})
		},
	), true
}

type normalizedTokenLanguage[T any] struct {
	name   string
	count  CountRange
	syntax []string
	parse  func(context.Context, []string) (T, error)
	format func(T) []string
}

func (language normalizedTokenLanguage[T]) tokenLanguageSyntax() []string {
	return append([]string(nil), language.syntax...)
}

func (language normalizedTokenLanguage[T]) erasedTokenLanguage() (tokenLanguage, bool) {
	return eraseNormalizedTokenLanguage(language)
}

func eraseTokenLanguageAny(raw any) (tokenLanguage, []string, bool) {
	language, ok := raw.(interface {
		erasedTokenLanguage() (tokenLanguage, bool)
		tokenLanguageSyntax() []string
	})
	if !ok {
		return tokenLanguage{}, nil, false
	}
	erased, valid := language.erasedTokenLanguage()
	if !valid {
		return tokenLanguage{}, nil, false
	}
	return erased, language.tokenLanguageSyntax(), true
}

func normalizeTokenLanguage[T any](language TokenLanguage[T]) (normalizedTokenLanguage[T], error) {
	if language.Name == "" || !language.Count.valid || language.Count.minimum < 1 || language.Parse == nil ||
		language.Format == nil {
		return normalizedTokenLanguage[T]{}, errInvalidTokenLanguage
	}
	result := normalizedTokenLanguage[T]{
		name:   language.Name,
		count:  language.Count,
		syntax: append([]string(nil), language.Syntax...),
		parse:  language.Parse,
		format: language.Format,
	}
	return result, nil
}

type conversionOutcomeKind uint8

const (
	conversionOutcomeInvalid conversionOutcomeKind = iota
	conversionOutcomeAccepted
	conversionOutcomeRejected
	conversionOutcomeFailed
)

type frozenTokenRepresentation struct {
	once   sync.Once
	format func() []string
	tokens []string
}

func freezeTokenRepresentation(format func() []string) *frozenTokenRepresentation {
	if format == nil {
		return nil
	}
	representation := &frozenTokenRepresentation{
		format: format,
	}
	return representation
}

func (representation *frozenTokenRepresentation) tokensSnapshot() []string {
	if representation == nil {
		return nil
	}
	representation.once.Do(func() {
		representation.tokens = append([]string(nil), representation.format()...)
		representation.format = nil
	})
	return append([]string(nil), representation.tokens...)
}

type conversionOutcome struct {
	kind                conversionOutcomeKind
	value               any
	err                 error
	tokenRepresentation *frozenTokenRepresentation
}

func acceptedConversion(value any) conversionOutcome {
	outcome := conversionOutcome{
		kind:  conversionOutcomeAccepted,
		value: snapshotMutableValue(value),
	}
	return outcome
}

func acceptedTokenConversion(value any, format func() []string) conversionOutcome {
	if format == nil {
		panic("semantic: nil token formatter")
	}
	result := conversionOutcome{
		kind:                conversionOutcomeAccepted,
		value:               snapshotMutableValue(value),
		tokenRepresentation: freezeTokenRepresentation(format),
	}

	return result
}

func rejectedConversion(err error) conversionOutcome {
	if err == nil {
		panic("semantic: nil conversion rejection")
	}
	outcome := conversionOutcome{
		kind: conversionOutcomeRejected,
		err:  err,
	}
	return outcome
}

func failedConversion(err error) conversionOutcome {
	if err == nil {
		panic("semantic: nil conversion failure")
	}
	outcome := conversionOutcome{
		kind: conversionOutcomeFailed,
		err:  err,
	}
	return outcome
}

func (outcome conversionOutcome) Kind() conversionOutcomeKind { return outcome.kind }

func (outcome conversionOutcome) Value() (any, bool) {
	if outcome.kind != conversionOutcomeAccepted {
		return nil, false
	}
	return snapshotMutableValue(outcome.value), true
}

func (outcome conversionOutcome) Err() error { return outcome.err }

func (outcome conversionOutcome) Clone() conversionOutcome {
	outcome.value = snapshotMutableValue(outcome.value)
	return outcome
}

// TokenRepresentation returns the canonical token spelling bound to the exact
// accepted typed TokenLanguage result. Count belongs to the owning grammar
// declaration, so callers supply that one structural authority for validation.
func (outcome conversionOutcome) TokenRepresentation(count semanticCountRange) ([]string, bool, error) {
	if outcome.kind != conversionOutcomeAccepted || outcome.tokenRepresentation == nil {
		return nil, false, nil
	}
	tokens := outcome.tokenRepresentation.tokensSnapshot()
	if !count.valid() || len(tokens) < count.Minimum() {
		return nil, true, errInvalidValueLanguageInput
	}
	if maximum, bounded := count.Maximum(); bounded && len(tokens) > maximum {
		return nil, true, errInvalidValueLanguageInput
	}
	return tokens, true, nil
}

// tokenLanguage is the immutable erased runtime form of one typed bounded
// token-language declaration. Structural cardinality remains on the owning
// FlagGrammar or ArgumentTermGrammar; this carrier owns executable conversion.
type tokenLanguage struct {
	name     string
	evaluate func(context.Context, []string) conversionOutcome
}

func compileTokenLanguage(name string, evaluate func(context.Context, []string) conversionOutcome) tokenLanguage {
	if name == "" || evaluate == nil {
		panic("semantic: invalid token language")
	}
	language := tokenLanguage{
		name:     name,
		evaluate: evaluate,
	}
	return language
}

func (language tokenLanguage) Valid() bool { return language.name != "" && language.evaluate != nil }

func (language tokenLanguage) Name() string { return language.name }

func (language tokenLanguage) Evaluate(ctx context.Context, tokens []string) conversionOutcome {
	if !language.Valid() {
		return failedConversion(errInvalidValueLanguageInput)
	}
	owned := append([]string(nil), tokens...)
	return language.evaluate(ctx, owned).Clone()
}
