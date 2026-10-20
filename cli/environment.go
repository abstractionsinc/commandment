package cli

import (
	"errors"
	"strings"
)

var errIncompleteEnvironmentProducer = errors.New("incomplete environment producer")

// environmentProducerMatch is one exact invocation-environment contribution
// selected by one accepted static environment producer. Matches preserve
// producer declaration order first and prepared environment source order
// second. Conversion and destination resolution remain downstream phases.
type environmentProducerMatch struct {
	Producer     EnvironmentInputID
	Destination  ValueID
	Contribution environmentContribution
	Delimiter    string
	EntryIndex   int
	Name         string
	KeyText      string
	ValueText    string
	Origin       InputOriginID
	Sensitive    bool
}

// MatchEnvironmentProducers selects raw environment contributions from the
// immutable static producer declarations and this operation's detached Env
// snapshot. It performs no conversion, candidate reduction, or precedence
// selection.
func (operation *invocation) MatchEnvironmentProducers() ([]environmentProducerMatch, error) {
	if operation == nil {
		panic("invocation: nil operation")
	}
	return operation.matchEnvironmentProducersAt(operation.analysis)
}

func (operation *invocation) matchEnvironmentProducersAt(analysis *analysisState) ([]environmentProducerMatch, error) {
	if operation == nil || analysis == nil {
		panic("invocation: nil operation analysis")
	}

	producers := analysis.effectiveModel.EnvironmentInputs()
	if len(producers) == 0 {
		return nil, nil
	}
	if len(producers) == 1 && !producers[0].prefix {
		producer := producers[0]
		if producer.selector == "" || producer.contribution == environmentContributionInvalid {
			return nil, errIncompleteEnvironmentProducer
		}
		var matches []environmentProducerMatch
		for index, entry := range operation.input.environment {
			keyText, matched := matchEnvironmentSelector(producer, entry.Name)
			if !matched {
				continue
			}
			origin, ok := analysis.environmentOrigin(index)
			if !ok {
				panic("invocation: missing prepared environment origin")
			}
			match := environmentProducerMatch{
				Producer:     producer.meta.id,
				Destination:  producer.destination,
				Contribution: producer.contribution,
				Delimiter:    producer.delimiter,
				EntryIndex:   index,
				Name:         entry.Name,
				KeyText:      keyText,
				ValueText:    entry.Value,
				Origin:       origin.ID(),
				Sensitive:    producer.sensitive,
			}
			matches = append(matches, match)
		}
		return matches, nil
	}
	hasExact := false
	for _, producer := range producers {
		if producer.selector == "" || producer.contribution == environmentContributionInvalid {
			return nil, errIncompleteEnvironmentProducer
		}
		if !producer.prefix {
			hasExact = true
		}
	}

	type exactEnvironmentEntries struct {
		first    int
		overflow []int
	}
	var exactEntries map[string]exactEnvironmentEntries
	if hasExact && len(operation.input.environment) != 0 {
		exactEntries = make(map[string]exactEnvironmentEntries, len(operation.input.environment))
		for index, entry := range operation.input.environment {
			indexed, exists := exactEntries[entry.Name]
			if !exists {
				exactEntries[entry.Name] = exactEnvironmentEntries{first: index}
				continue
			}
			indexed.overflow = append(indexed.overflow, index)
			exactEntries[entry.Name] = indexed
		}
	}

	var matches []environmentProducerMatch
	appendMatch := func(producer environmentInputRecord, index int, keyText string) {
		entry := operation.input.environment[index]
		origin, ok := analysis.environmentOrigin(index)
		if !ok {
			panic("invocation: missing prepared environment origin")
		}
		match := environmentProducerMatch{
			Producer:     producer.meta.id,
			Destination:  producer.destination,
			Contribution: producer.contribution,
			Delimiter:    producer.delimiter,
			EntryIndex:   index,
			Name:         entry.Name,
			KeyText:      keyText,
			ValueText:    entry.Value,
			Origin:       origin.ID(),
			Sensitive:    producer.sensitive,
		}
		matches = append(matches, match)
	}
	for _, producer := range producers {
		if !producer.prefix {
			indexed, exists := exactEntries[producer.selector]
			if !exists {
				continue
			}
			appendMatch(producer, indexed.first, "")
			for _, index := range indexed.overflow {
				appendMatch(producer, index, "")
			}
			continue
		}
		for index, entry := range operation.input.environment {
			keyText, matched := matchEnvironmentSelector(producer, entry.Name)
			if matched {
				appendMatch(producer, index, keyText)
			}
		}
	}
	return matches, nil
}

func matchEnvironmentSelector(producer environmentInputRecord, name string) (string, bool) {
	if producer.prefix {
		if !strings.HasPrefix(name, producer.selector) {
			return "", false
		}
		return name[len(producer.selector):], true
	}
	return "", name == producer.selector
}

type environmentInputSyntax struct {
	selector     string
	prefix       bool
	contribution environmentContribution
	delimiter    string
	sensitive    bool
}

func prepareEnvironmentInputSyntax(directive environmentInputAcceptanceInput, destination destinationDefinition) (environmentInputSyntax, bool) {
	if directive.name == "" || strings.ContainsAny(directive.name, "=\x00") {
		return environmentInputSyntax{}, false
	}
	syntax := environmentInputSyntax{
		selector:  directive.name,
		prefix:    directive.kind == valueDirectiveEnvironmentPrefix,
		delimiter: directive.delimiter,
		sensitive: directive.sensitive,
	}
	adapterSet := directive.environmentShape != environmentInputShapeDefault
	if adapterSet && directive.delimiter == "" {
		return environmentInputSyntax{}, false
	}
	switch directive.environmentShape {
	case environmentInputShapeDefault:
	case environmentInputShapeElements:
		syntax.contribution = environmentContributionSequenceElements
	case environmentInputShapeWholeSequence:
		syntax.contribution = environmentContributionWholeSequence
	case environmentInputShapeKeyedEntries:
		syntax.contribution = environmentContributionKeyedEntries
	case environmentInputShapeWholeKeyed:
		syntax.contribution = environmentContributionWholeKeyed
	default:
		return environmentInputSyntax{}, false
	}
	if syntax.prefix {
		if destination.Kind() != destinationKindKeyed || adapterSet {
			return environmentInputSyntax{}, false
		}
		if language, ok := destination.KeyedLanguage(); !ok || !language.Valid() {
			return environmentInputSyntax{}, false
		}
		syntax.contribution = environmentContributionKeyedEntries
		return syntax, true
	}
	switch destination.Kind() {
	case destinationKindScalar:
		language, ok := destination.Language()
		if !ok || !language.Valid() || adapterSet {
			return environmentInputSyntax{}, false
		}
		syntax.contribution = environmentContributionScalar
	case destinationKindSequence:
		language, ok := destination.Language()
		if !ok || !language.Valid() || !adapterSet ||
			(syntax.contribution != environmentContributionSequenceElements &&
				syntax.contribution != environmentContributionWholeSequence) {
			return environmentInputSyntax{}, false
		}
	case destinationKindKeyed:
		language, ok := destination.KeyedLanguage()
		if !ok || !language.Valid() || !adapterSet ||
			(syntax.contribution != environmentContributionKeyedEntries &&
				syntax.contribution != environmentContributionWholeKeyed) {
			return environmentInputSyntax{}, false
		}
	default:
		return environmentInputSyntax{}, false
	}
	return syntax, true
}

type environmentInputAcceptanceInput struct {
	kind             valueDirectiveKind
	name             string
	sensitive        bool
	environmentShape environmentInputShape
	delimiter        string
}

func staticEnvironmentInputAcceptanceInput(value valueDirectiveDraft) (environmentInputAcceptanceInput, bool) {
	if !value.valid {
		return environmentInputAcceptanceInput{}, false
	}
	input := environmentInputAcceptanceInput{
		kind:             value.kind,
		name:             value.name,
		sensitive:        value.sensitive,
		environmentShape: value.environmentShape,
		delimiter:        value.delimiter,
	}

	return input, true
}

// EnvironmentInputDeclaration identifies one staged environment-input producer for a destination.
// It may participate in replacement relationships within its owning scope. The zero value is
// invalid; copies identify the same declaration, and retained, concurrent, or cross-scope use is
// builder lifecycle misuse.
type EnvironmentInputDeclaration[C, T any] struct {
	state    *declarationSession
	producer valueDirectiveRef
}

// EnvironmentPrefixDeclaration identifies a prefix-matched keyed environment producer.
type EnvironmentPrefixDeclaration[C any, K comparable, V any] struct {
	state    *declarationSession
	producer valueDirectiveRef
}

type environmentInputShape uint8

const (
	environmentInputShapeDefault environmentInputShape = iota
	environmentInputShapeElements
	environmentInputShapeWholeSequence
	environmentInputShapeKeyedEntries
	environmentInputShapeWholeKeyed
)

// Environment declares an exact environment-variable producer for destination. Environment input
// participates in ordinary value resolution and preserves environment-entry provenance.
func (values *Values[C]) Environment[T any](destination destinationLike[C, T], name string) EnvironmentInputDeclaration[C, T] {
	state := destinationHandleOf(destination)
	directive := valueDirectiveDraft{
		kind:  valueDirectiveEnvironment,
		name:  name,
		valid: state != nil && name != "",
	}
	producer := stageValueDirective(values.session, values.profile, directive, state)
	return EnvironmentInputDeclaration[C, T]{
		state:    values.session,
		producer: producer,
	}
}

// EnvironmentPrefix declares a prefix-matched environment producer for a keyed destination.
func (values *Values[C]) EnvironmentPrefix[K comparable, V any](destination destinationLike[C, map[K]V], prefix string) EnvironmentPrefixDeclaration[C, K, V] {
	state := destinationHandleOf(destination)
	directive := valueDirectiveDraft{
		kind:  valueDirectiveEnvironmentPrefix,
		name:  prefix,
		valid: state != nil && prefix != "",
	}
	producer := stageValueDirective(values.session, values.profile, directive, state)
	return EnvironmentPrefixDeclaration[C, K, V]{
		state:    values.session,
		producer: producer,
	}
}

func configureEnvironmentShape[C, T any](declaration EnvironmentInputDeclaration[C, T], shape environmentInputShape, delimiter string) EnvironmentInputDeclaration[C, T] {
	mutateValueDirective(declaration.state, declaration.producer, func(producer *valueDirectiveDraft) {
		if producer.environmentShape != environmentInputShapeDefault || delimiter == "" {
			producer.valid = false
			return
		}
		producer.environmentShape = shape
		producer.delimiter = delimiter
	})
	return declaration
}

// Elements treats delimiter-separated environment text as sequence elements.
func (declaration EnvironmentInputDeclaration[C, T]) Elements(delimiter string) EnvironmentInputDeclaration[C, T] {
	return configureEnvironmentShape(declaration, environmentInputShapeElements, delimiter)
}

// WholeSequence treats the complete environment entry as one sequence contribution.
func (declaration EnvironmentInputDeclaration[C, T]) WholeSequence(delimiter string) EnvironmentInputDeclaration[C, T] {
	return configureEnvironmentShape(declaration, environmentInputShapeWholeSequence, delimiter)
}

// KeyedEntries treats delimiter-separated environment text as separate keyed entries.
func (declaration EnvironmentInputDeclaration[C, T]) KeyedEntries(delimiter string) EnvironmentInputDeclaration[C, T] {
	return configureEnvironmentShape(declaration, environmentInputShapeKeyedEntries, delimiter)
}

// WholeKeyed treats the complete environment entry as one keyed contribution.
func (declaration EnvironmentInputDeclaration[C, T]) WholeKeyed(delimiter string) EnvironmentInputDeclaration[C, T] {
	return configureEnvironmentShape(declaration, environmentInputShapeWholeKeyed, delimiter)
}

func (declaration EnvironmentInputDeclaration[C, T]) Sensitive() EnvironmentInputDeclaration[C, T] {
	mutateValueDirective(declaration.state, declaration.producer, func(producer *valueDirectiveDraft) {
		producer.sensitive = true
	})
	return declaration
}

func (declaration EnvironmentPrefixDeclaration[C, K, V]) Sensitive() EnvironmentPrefixDeclaration[C, K, V] {
	mutateValueDirective(declaration.state, declaration.producer, func(producer *valueDirectiveDraft) {
		producer.sensitive = true
	})
	return declaration
}

func stageEnvironmentLifecycle(session *declarationSession, target valueDirectiveRef, lifecycle ReleaseLifecycle) {
	if session == nil || !target.valid() {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		profile := grammarProfileCaseRef{}
		if source, ok := draft.valueDirectives.resolve(draft, target); ok {
			profile = source.profile
		}
		record := lifecycleDirectiveDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  profile,
			target: authoringEnvironmentLifecycleTarget{
				reference: target,
			},
			lifecycle: lifecycle,
			valid:     target.ownedBy(draft.identity),
		}
		draft.lifecycleDirectives.append(draft, record)
	})
}

func validEnvironmentInputSyntax(record environmentInputRecord) bool {
	if record.selector == "" || strings.ContainsAny(record.selector, "=\x00") {
		return false
	}
	switch record.contribution {
	case environmentContributionScalar:
		return !record.prefix && record.delimiter == ""
	case environmentContributionSequenceElements,
		environmentContributionWholeSequence,
		environmentContributionWholeKeyed:
		return !record.prefix && record.delimiter != ""
	case environmentContributionKeyedEntries:
		if record.prefix {
			return record.delimiter == ""
		}
		return record.delimiter != ""
	default:
		return false
	}
}

func environmentSelectorsOverlap(left, right environmentInputRecord) bool {
	if left.prefix && right.prefix {
		return strings.HasPrefix(left.selector, right.selector) ||
			strings.HasPrefix(right.selector, left.selector)
	}
	if left.prefix {
		return strings.HasPrefix(right.selector, left.selector)
	}
	if right.prefix {
		return strings.HasPrefix(left.selector, right.selector)
	}
	return left.selector == right.selector
}

type environmentContribution uint8

const (
	environmentContributionInvalid environmentContribution = iota
	environmentContributionScalar
	environmentContributionSequenceElements
	environmentContributionWholeSequence
	environmentContributionKeyedEntries
	environmentContributionWholeKeyed
)

type environmentInputRecord struct {
	meta         recordMeta[EnvironmentInputID]
	command      CommandID
	destination  ValueID
	selector     string
	prefix       bool
	contribution environmentContribution
	delimiter    string
	sensitive    bool
	lifecycle    ReleaseLifecycle
}

// environmentInputRecordInput is the complete immutable payload of one environment input record.
type environmentInputRecordInput struct {
	ID           EnvironmentInputID
	Command      CommandID
	Destination  ValueID
	Position     declarationPosition
	Selector     string
	Prefix       bool
	Contribution environmentContribution
	Delimiter    string
	Sensitive    bool
}

func acceptEnvironmentInputRecordWithSyntax(input environmentInputRecordInput) environmentInputRecord {
	mustCommand(input.Command)
	mustDestination(input.Destination)
	record := environmentInputRecord{
		meta:         acceptRecordMeta(input.ID, input.Position),
		command:      input.Command,
		destination:  input.Destination,
		selector:     input.Selector,
		prefix:       input.Prefix,
		contribution: input.Contribution,
		delimiter:    input.Delimiter,
		sensitive:    input.Sensitive,
	}

	return record
}
