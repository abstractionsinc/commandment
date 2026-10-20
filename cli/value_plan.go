package cli

import (
	"context"
	"errors"
	"slices"
	"strings"
)

type valueCandidateComponentInput struct {
	text    string
	present bool
	origins []InputOriginID
}

type keyedEntryPlan struct {
	keyInput   valueCandidateComponentInput
	valueInput valueCandidateComponentInput
	key        func() (any, error)
	value      func() (any, error)
}

type valuePlanBuilder struct {
	state    *analysisState
	profiles map[CommandID]GrammarProfileID
	selected map[CommandID]struct{}
	include  map[ValueID]struct{}
	plans    map[ValueID][]valueCandidatePlan
	analysis *valueAnalysis
}

type valueCandidatePlan struct {
	producerKind        ValueProducerKind
	tier                ResolutionTier
	order               uint64
	sensitive           bool
	origins             []InputOriginID
	declaration         valueCandidateDeclaration
	input               string
	hasInput            bool
	inputSource         InputSourceID
	producerIndex       int
	tracker             *producerTracker
	tokenLanguage       tokenLanguage
	tokenRepresentation func() ([]string, bool, error)

	elementInputs    []valueCandidateComponentInput
	sequenceWhole    bool
	keyInput         valueCandidateComponentInput
	valueInput       valueCandidateComponentInput
	keyedBatch       bool
	keyedEntries     []keyedEntryPlan
	keyedReplacement bool

	whole    func() (any, error)
	sequence func() ([]any, error)
	key      func() (any, error)
	value    func() (any, error)
}

func (builder *valuePlanBuilder) appendGrammarValueCandidatePlans(ctx context.Context, operation *invocation, model grammarView, grammar grammarResult) {
	builder.appendGrammarValueCandidatePlansFromArgs(ctx, operation, model, builder.state.argsSnapshot(), grammar)
}

func (builder *valuePlanBuilder) appendGrammarValueCandidatePlansFromArgs(ctx context.Context, operation *invocation, model grammarView, args []string, grammar grammarResult) {
	flags := make(map[FlagID]flagGrammar)
	spellings := make(map[FlagSpellingID]flagSpellingGrammar)
	for _, command := range grammar.Selection.Path {
		for _, flag := range model.Flags(command) {
			flags[flag.id] = flag
			for _, spelling := range flag.Spellings() {
				spellings[spelling.id] = spelling
			}
		}
	}

	forms := make(map[ArgumentFormID]argumentFormGrammar)
	terms := make(map[ArgumentTermID]argumentTermGrammar)
	for _, form := range model.ArgumentForms(grammar.Selection.Selected) {
		forms[form.id] = form
		for _, term := range form.Terms() {
			terms[term.id] = term
		}
	}

	builder.appendFlagValueCandidatePlans(ctx, operation, args, grammar, flags, spellings, 0)
	flagOrderCount := uint64(len(grammar.Occurrences))
	selectionCount := builder.appendArgumentFormValueCandidatePlans(grammar, forms, flagOrderCount)
	builder.appendArgumentTermValueCandidatePlans(
		ctx,
		operation,
		args,
		grammar,
		terms,
		flagOrderCount+checkedUint64FromInt(selectionCount),
	)
}

func (builder *valuePlanBuilder) appendFlagValueCandidatePlans(ctx context.Context, operation *invocation, args []string, grammar grammarResult, flags map[FlagID]flagGrammar, spellings map[FlagSpellingID]flagSpellingGrammar, orderBase uint64) {
	// Count is one aggregated candidate per flag declaration, not one scalar
	// candidate per occurrence.
	countOrigins := make(map[FlagID][]InputOriginID)
	countFirstOrder := make(map[FlagID]uint64)
	for index, occurrence := range grammar.Occurrences {
		flag, ok := flags[occurrence.Flag]
		if !ok {
			builder.analysis.Unsupported = true
			continue
		}
		if _, ok := builder.selected[flag.command]; !ok ||
			!analysisGrammarProfileApplies(grammar.SelectedProfiles, flag.command, flag.profile) {
			continue
		}
		if builder.include != nil {
			if _, wanted := builder.include[occurrence.Destination]; !wanted {
				continue
			}
		}
		definition, ok := analysisDestinationDefinition(builder.state, occurrence.Destination)
		if !ok {
			builder.analysis.Unsupported = true
			continue
		}
		spelling, ok := spellings[occurrence.Spelling]
		if !ok {
			builder.analysis.Unsupported = true
			continue
		}
		originList := occurrenceOrigins(builder.state, grammar, occurrence)
		valueType, hasType := definition.ValueType()
		order := orderBase + uint64(index) + 1
		if hasType && valueType.Kind == ValueKindCount {
			countOrigins[occurrence.Flag] = append(countOrigins[occurrence.Flag], originList...)
			if _, exists := countFirstOrder[occurrence.Flag]; !exists {
				countFirstOrder[occurrence.Flag] = order
			}
			continue
		}
		plan, ok := flagValueCandidatePlan(
			ctx,
			operation,
			builder.state,
			args,
			occurrence,
			flag,
			spelling,
			definition,
			order,
			originList,
		)
		if !ok {
			builder.analysis.Unsupported = true
			continue
		}
		builder.plans[occurrence.Destination] = append(builder.plans[occurrence.Destination], plan)
	}
	for flagID, origins := range countOrigins {
		flag := flags[flagID]
		if builder.include != nil {
			if _, wanted := builder.include[flag.destination]; !wanted {
				continue
			}
		}
		count := len(origins)
		plan := valueCandidatePlan{
			producerIndex: -1,
			producerKind:  ValueProducerKindFlag,
			tier:          ResolutionTierExplicit,
			order:         countFirstOrder[flagID],
			origins:       append([]InputOriginID(nil), origins...),
			declaration: valueCandidateDeclaration{
				Flag: flagID,
			},
			whole: func() (any, error) { return count, nil },
		}
		destination := flag.destination
		builder.plans[destination] = append(builder.plans[destination], plan)
	}
}

func (builder *valuePlanBuilder) appendArgumentFormValueCandidatePlans(grammar grammarResult, forms map[ArgumentFormID]argumentFormGrammar, orderBase uint64) int {
	selectedForm, ok := forms[grammar.Arguments.Form]
	if grammar.Arguments.Form.present() &&
		(!ok || !analysisGrammarProfileApplies(grammar.SelectedProfiles, selectedForm.command, selectedForm.profile)) {
		builder.analysis.Unsupported = true
	}
	if !grammar.Arguments.Form.present() || !ok {
		return 0
	}
	form, present := builder.state.effectiveModel.ArgumentForm(grammar.Arguments.Form)
	if !present {
		builder.analysis.Unsupported = true
		return 0
	}
	selections := form.Selections()
	for selectionIndex, selection := range selections {
		destination := selection.destination
		if builder.include != nil {
			if _, wanted := builder.include[destination]; !wanted {
				continue
			}
		}
		definition, found := analysisDestinationDefinition(builder.state, destination)
		if !found {
			builder.analysis.Unsupported = true
			continue
		}
		plan2 := valueCandidatePlan{
			producerIndex: -1,
			producerKind:  ValueProducerKindArgument,
			tier:          ResolutionTierExplicit,
			order:         orderBase + uint64(selectionIndex) + 1,
		}
		plan, valid := typedFixedPlan(plan2, definition, selection.value)
		if !valid {
			builder.analysis.Unsupported = true
			continue
		}
		builder.plans[destination] = append(builder.plans[destination], plan)
	}
	return len(selections)
}

func (builder *valuePlanBuilder) appendArgumentTermValueCandidatePlans(ctx context.Context, operation *invocation, args []string, grammar grammarResult, terms map[ArgumentTermID]argumentTermGrammar, orderBase uint64) {
	for allocationIndex, allocation := range grammar.Arguments.Terms {
		term, ok := terms[allocation.Term]
		if !ok {
			builder.analysis.Unsupported = true
			continue
		}
		destination, hasDestination := term.Destination()
		if !hasDestination {
			continue
		}
		if builder.include != nil {
			if _, wanted := builder.include[destination]; !wanted {
				continue
			}
		}
		definition, ok := analysisDestinationDefinition(builder.state, destination)
		if !ok {
			builder.analysis.Unsupported = true
			continue
		}
		origins := indicesOrigins(builder.state, allocation.Indices)
		values := make([]string, len(allocation.Indices))
		for i, inputIndex := range allocation.Indices {
			if inputIndex >= 0 && inputIndex < len(args) {
				values[i] = args[inputIndex]
			}
		}
		plan := valueCandidatePlan{
			producerKind: ValueProducerKindArgument,
			tier:         ResolutionTierExplicit,
			order:        orderBase + uint64(allocationIndex) + 1,
			origins:      origins,
			declaration: valueCandidateDeclaration{
				ArgumentTerm: term.id,
			},
			inputSource: builder.state.commonOriginSource(origins),
		}
		switch term.kind {
		case ArgumentTermValue:
			if len(values) == 0 {
				continue
			}
			language, languageOK := definition.Language()
			if !languageOK {
				builder.analysis.Unsupported = true
				continue
			}
			switch {
			case definition.Kind() == destinationKindSequence:
				texts := append([]string(nil), values...)
				owned := append([]int(nil), allocation.Indices...)
				plan.sequence = func() ([]any, error) {
					return parseSequenceFrozen(operation, builder.state, destination, language, texts, owned)
				}
				plan.elementInputs = componentInputs(texts, allocation.Indices, builder.state)
			case len(values) == 1:
				text := values[0]
				index := allocation.Indices[0]
				plan.input, plan.hasInput = text, true
				plan.whole = func() (any, error) {
					return conversionValue(operation.evaluateValueLanguage(builder.state, destination, language, text, index))
				}
			default:
				builder.analysis.Unsupported = true
				continue
			}
		case ArgumentTermRemainder:
			plan.producerKind = ValueProducerKindRemainder
			texts := append([]string(nil), values...)
			plan.elementInputs = componentInputs(texts, allocation.Indices, builder.state)
			plan.sequence = func() ([]any, error) {
				items := make([]any, len(texts))
				for i := range texts {
					items[i] = texts[i]
				}
				return items, nil
			}
		case ArgumentTermPassThrough:
			plan.producerKind = ValueProducerKindPassThrough
			texts := append([]string(nil), values...)
			plan.elementInputs = componentInputs(texts, allocation.Indices, builder.state)
			plan.sequence = func() ([]any, error) {
				items := make([]any, len(texts))
				for i := range texts {
					items[i] = texts[i]
				}
				return items, nil
			}
		case ArgumentTermTokens:
			language, languageOK := term.TokenLanguage()
			if !languageOK {
				builder.analysis.Unsupported = true
				continue
			}
			texts := append([]string(nil), values...)
			owned := append([]int(nil), allocation.Indices...)
			plan.tokenLanguage = language
			plan.whole = func() (any, error) {
				return conversionValue(operation.evaluateArgumentTokenLanguage(ctx, builder.state, term.id, language, owned, texts))
			}
			plan.tokenRepresentation = func() ([]string, bool, error) {
				outcome := operation.evaluateArgumentTokenLanguage(ctx, builder.state, term.id, language, owned, texts)
				return outcome.TokenRepresentation(term.count)
			}
		default:
			continue
		}
		builder.plans[destination] = append(builder.plans[destination], plan)
	}
}

func flagValueCandidatePlan(ctx context.Context, operation *invocation, state *analysisState, args []string, occurrence flagOccurrence, flag flagGrammar, spelling flagSpellingGrammar, definition destinationDefinition, order uint64, origins []InputOriginID) (valueCandidatePlan, bool) {
	plan := valueCandidatePlan{
		producerIndex: -1,
		producerKind:  ValueProducerKindFlag,
		tier:          ResolutionTierExplicit,
		order:         order,
		sensitive:     spelling.sensitive,
		origins:       append([]InputOriginID(nil), origins...),
		declaration: valueCandidateDeclaration{
			Flag:         flag.id,
			FlagSpelling: spelling.id,
		},
		inputSource: flagCandidateInputSource(state, occurrence),
	}
	if value, ok, sensitive := spelling.FixedValue(); ok {
		plan.sensitive = plan.sensitive || sensitive
		return typedFixedPlan(plan, definition, value)
	}
	if occurrence.Attached == "" && len(occurrence.ValueIndices) == 0 {
		if value, ok, sensitive := spelling.ImplicitValue(); ok {
			plan.sensitive = plan.sensitive || sensitive
			return typedFixedPlan(plan, definition, value)
		}
		return valueCandidatePlan{}, false
	}

	texts := occurrenceValueTexts(args, occurrence)
	if len(texts) == 0 {
		return valueCandidatePlan{}, false
	}
	plan.input, plan.hasInput = strings.Join(texts, "\x00"), true
	if language, ok := flag.TokenLanguage(); ok {
		owned := append([]string(nil), texts...)
		plan.tokenLanguage = language
		plan.whole = func() (any, error) {
			return conversionValue(operation.evaluateFlagTokenLanguage(ctx, state, flag.id, language, occurrence, owned))
		}
		plan.tokenRepresentation = func() ([]string, bool, error) {
			outcome := operation.evaluateFlagTokenLanguage(ctx, state, flag.id, language, occurrence, owned)
			return outcome.TokenRepresentation(flag.valuesPerOccurrence)
		}
		return plan, true
	}
	switch definition.Kind() {
	case destinationKindScalar:
		if len(texts) != 1 {
			return valueCandidatePlan{}, false
		}
		language, ok := definition.Language()
		if !ok {
			return valueCandidatePlan{}, false
		}
		text := texts[0]
		plan.whole = func() (any, error) { return language.Parse(text) }
		return plan, true
	case destinationKindSequence:
		language, ok := definition.Language()
		if !ok {
			return valueCandidatePlan{}, false
		}
		copied := append([]string(nil), texts...)
		plan.elementInputs = flagComponentInputs(state, occurrence, copied)
		plan.sequence = func() ([]any, error) { return parseSequence(language, copied) }
		return plan, true
	case destinationKindKeyed:
		language, ok := definition.KeyedLanguage()
		if !ok {
			return valueCandidatePlan{}, false
		}
		inputs := flagComponentInputs(state, occurrence, texts)
		entries, ok := keyedEntryPlans(language, inputs)
		if !ok {
			return valueCandidatePlan{}, false
		}
		plan.keyedBatch = true
		plan.keyedEntries = entries
		return plan, true
	default:
		return valueCandidatePlan{}, false
	}
}

func typedFixedPlan(plan valueCandidatePlan, definition destinationDefinition, value any) (valueCandidatePlan, bool) {
	switch definition.Kind() {
	case destinationKindScalar, destinationKindOpaque:
		plan.whole = func() (any, error) {
			snapshot, ok := definition.Snapshot(value)
			if !ok {
				return nil, errValueLanguageTypeMismatch
			}
			return snapshot, nil
		}
	case destinationKindSequence:
		items, ok := definition.UnpackSequence(value)
		if !ok {
			return valueCandidatePlan{}, false
		}
		plan.sequence = func() ([]any, error) { return append([]any(nil), items...), nil }
	case destinationKindKeyed:
		entries, ok := definition.UnpackKeyed(value)
		if !ok {
			return valueCandidatePlan{}, false
		}
		plan.keyedBatch = true
		for _, entry := range entries {
			plan.keyedEntries = append(plan.keyedEntries, keyedEntryPlan{
				key:   func() (any, error) { return snapshotMutableValue(entry.Key), nil },
				value: func() (any, error) { return snapshotMutableValue(entry.Value), nil },
			})
		}
	default:
		return valueCandidatePlan{}, false
	}
	return plan, true
}

func (builder *valuePlanBuilder) appendEnvironmentValueCandidatePlans(operation *invocation) {
	profiles := builder.profiles
	selected := builder.selected
	include := builder.include
	plans := builder.plans
	analysis := builder.analysis
	matches, err := operation.matchEnvironmentProducersAt(builder.state)
	if err != nil {
		analysis.Unsupported = true
		analysis.Issues = append(analysis.Issues, valueIssue{
			Kind: valueIssueUnsupported,
			Err:  err,
		})
		return
	}
	for index, match := range matches {
		record, ok := builder.state.effectiveModel.EnvironmentInput(match.Producer)
		if !ok {
			analysis.Unsupported = true
			continue
		}
		if _, ok := selected[record.command]; !ok ||
			!analysisGrammarProfileApplies(profiles, record.command, record.meta.profile) {
			continue
		}
		if include != nil {
			if _, wanted := include[match.Destination]; !wanted {
				continue
			}
		}
		definition, ok := analysisDestinationDefinition(builder.state, match.Destination)
		if !ok {
			analysis.Unsupported = true
			continue
		}
		plan := valueCandidatePlan{
			producerIndex: -1,
			producerKind:  ValueProducerKindEnvironment,
			tier:          ResolutionTierEnvironment,
			order:         uint64(index + 1),
			sensitive:     match.Sensitive,
			origins:       []InputOriginID{match.Origin},
			declaration: valueCandidateDeclaration{
				EnvironmentInput: match.Producer,
			},
			input:       match.ValueText,
			hasInput:    true,
			inputSource: operation.EnvironmentSource().ID(),
		}
		switch match.Contribution {
		case environmentContributionScalar:
			language, ok := definition.Language()
			if !ok {
				analysis.Unsupported = true
				continue
			}
			text := match.ValueText
			plan.whole = func() (any, error) { return language.Parse(text) }
		case environmentContributionSequenceElements:
			language, ok := definition.Language()
			if !ok {
				analysis.Unsupported = true
				continue
			}
			texts := environmentCollectionParts(match.ValueText, match.Delimiter)
			plan.elementInputs = repeatedOriginComponentInputs(texts, match.Origin)
			plan.sequence = func() ([]any, error) { return parseSequence(language, texts) }
		case environmentContributionWholeSequence:
			plan.sequenceWhole = true
			language, ok := definition.Language()
			if !ok {
				analysis.Unsupported = true
				continue
			}
			texts := environmentCollectionParts(match.ValueText, match.Delimiter)
			plan.elementInputs = repeatedOriginComponentInputs(texts, match.Origin)
			plan.sequence = func() ([]any, error) { return parseSequence(language, texts) }
		case environmentContributionKeyedEntries:
			language, ok := definition.KeyedLanguage()
			if !ok {
				analysis.Unsupported = true
				continue
			}
			plan.keyedBatch = true
			if record.prefix {
				plan.keyedEntries = []keyedEntryPlan{
					keyedDirectEntryPlan(
						language,
						match.KeyText,
						match.ValueText,
						[]InputOriginID{match.Origin},
					),
				}
			} else if match.ValueText != "" {
				raw := repeatedOriginComponentInputs(splitContribution(match.ValueText, match.Delimiter), match.Origin)
				entries, ok := keyedEntryPlans(language, raw)
				if !ok {
					analysis.Unsupported = true
					continue
				}
				plan.keyedEntries = entries
			}
		case environmentContributionWholeKeyed:
			language, ok := definition.KeyedLanguage()
			if !ok {
				analysis.Unsupported = true
				continue
			}
			plan.keyedReplacement = true
			if match.ValueText != "" {
				raw := repeatedOriginComponentInputs(splitContribution(match.ValueText, match.Delimiter), match.Origin)
				entries, ok := keyedEntryPlans(language, raw)
				if !ok {
					analysis.Unsupported = true
					continue
				}
				plan.keyedEntries = entries
			}
		default:
			analysis.Unsupported = true
			continue
		}
		plans[match.Destination] = append(plans[match.Destination], plan)
	}
}

func analysisDestinationDefinition(state *analysisState, destination ValueID) (destinationDefinition, bool) {
	record, ok := state.effectiveModel.Destination(destination)
	if !ok {
		return destinationDefinition{}, false
	}
	return record.Definition()
}

func occurrenceOrigins(state *analysisState, grammar grammarResult, occurrence flagOccurrence) []InputOriginID {
	var origins []InputOriginID
	for _, token := range grammar.Tokens {
		if token.Ownership.Flag == occurrence.Flag && token.Ownership.Occurrence == occurrence.Occurrence {
			origins = append(origins, token.Origin)
		}
	}
	for _, index := range occurrence.ValueIndices {
		origin, ok := state.argumentOrigin(index)
		if ok {
			origins = appendUniqueOrigin(origins, origin.ID())
		}
	}
	return origins
}

func indicesOrigins(state *analysisState, indices []int) []InputOriginID {
	result := make([]InputOriginID, 0, len(indices))
	for _, index := range indices {
		if origin, ok := state.argumentOrigin(index); ok {
			result = append(result, origin.ID())
		}
	}
	return result
}

func occurrenceValueTexts(args []string, occurrence flagOccurrence) []string {
	if occurrence.Attached != "" {
		return []string{occurrence.Attached}
	}
	result := make([]string, 0, len(occurrence.ValueIndices))
	for _, index := range occurrence.ValueIndices {
		if index >= 0 && index < len(args) {
			result = append(result, args[index])
		}
	}
	return result
}

func componentInputs(texts []string, indices []int, state *analysisState) []valueCandidateComponentInput {
	result := make([]valueCandidateComponentInput, len(texts))
	for i, text := range texts {
		result[i] = valueCandidateComponentInput{
			text:    text,
			present: true,
		}
		if i < len(indices) {
			if origin, ok := state.argumentOrigin(indices[i]); ok {
				result[i].origins = []InputOriginID{origin.ID()}
			}
		}
	}
	return result
}

func repeatedOriginComponentInputs(texts []string, origin InputOriginID) []valueCandidateComponentInput {
	result := make([]valueCandidateComponentInput, len(texts))
	for i, text := range texts {
		result[i] = valueCandidateComponentInput{
			text:    text,
			present: true,
			origins: []InputOriginID{origin},
		}
	}
	return result
}

func flagCandidateInputSource(state *analysisState, occurrence flagOccurrence) InputSourceID {
	origins := flagValueComponentOrigins(state, occurrence)
	if len(origins) == 0 {
		if origin, ok := state.argumentOrigin(occurrence.TokenIndex); ok {
			origins = []InputOriginID{origin.ID()}
		}
	}
	return state.commonOriginSource(origins)
}

func flagComponentInputs(state *analysisState, occurrence flagOccurrence, texts []string) []valueCandidateComponentInput {
	if occurrence.Attached != "" {
		origins := flagValueComponentOrigins(state, occurrence)
		return []valueCandidateComponentInput{{
			text:    occurrence.Attached,
			present: true,
			origins: origins,
		}}
	}
	return componentInputs(texts, occurrence.ValueIndices, state)
}

func flagValueComponentOrigins(state *analysisState, occurrence flagOccurrence) []InputOriginID {
	if occurrence.Attached != "" {
		if origin, ok := state.argumentOrigin(occurrence.TokenIndex); ok {
			return []InputOriginID{origin.ID()}
		}
	}
	return indicesOrigins(state, occurrence.ValueIndices)
}

func keyedEntryPlans(language keyedValueLanguage, inputs []valueCandidateComponentInput) ([]keyedEntryPlan, bool) {
	result := make([]keyedEntryPlan, 0, len(inputs))
	for _, input := range inputs {
		keyText, valueText, hasValue := strings.Cut(input.text, language.Separator())
		entry := keyedEntryPlan{
			keyInput: valueCandidateComponentInput{
				text:    keyText,
				present: true,
				origins: append([]InputOriginID(nil), input.origins...),
			},
		}
		if !hasValue {
			keyOnly, ok := language.KeyOnly()
			if !ok {
				return nil, false
			}
			entry.key = func() (any, error) { return language.Key().Parse(keyText) }
			entry.value = func() (any, error) { return snapshotMutableValue(keyOnly), nil }
		} else {
			entry.valueInput = valueCandidateComponentInput{
				text:    valueText,
				present: true,
				origins: append([]InputOriginID(nil), input.origins...),
			}
			entry.key = func() (any, error) { return language.Key().Parse(keyText) }
			entry.value = func() (any, error) { return language.Value().Parse(valueText) }
		}
		result = append(result, entry)
	}
	return result, true
}

func keyedDirectEntryPlan(language keyedValueLanguage, keyText, valueText string, origins []InputOriginID) keyedEntryPlan {
	plan := keyedEntryPlan{
		keyInput: valueCandidateComponentInput{
			text:    keyText,
			present: true,
			origins: append([]InputOriginID(nil), origins...),
		},
		valueInput: valueCandidateComponentInput{
			text:    valueText,
			present: true,
			origins: append([]InputOriginID(nil), origins...),
		},
		key:   func() (any, error) { return language.Key().Parse(keyText) },
		value: func() (any, error) { return language.Value().Parse(valueText) },
	}

	return plan
}

func conversionValue(outcome conversionOutcome) (any, error) {
	if value, ok := outcome.Value(); ok {
		return value, nil
	}
	return nil, outcome.Err()
}

func parseSequenceFrozen(operation *invocation, analysis *analysisState, destination ValueID, language valueLanguage, texts []string, indices []int) ([]any, error) {
	values := make([]any, len(texts))
	for i, text := range texts {
		index := -1
		if i < len(indices) {
			index = indices[i]
		}
		value, err := conversionValue(operation.evaluateValueLanguage(analysis, destination, language, text, index))
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return values, nil
}

func parseSequence(language valueLanguage, texts []string) ([]any, error) {
	values := make([]any, len(texts))
	for i, text := range texts {
		value, err := language.Parse(text)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return values, nil
}

func splitContribution(text, delimiter string) []string {
	if delimiter == "" {
		return []string{text}
	}
	return strings.Split(text, delimiter)
}

// environmentCollectionParts reserves the exact empty environment value
// for an empty segment/collection. Interior empty fields in a nonempty value
// remain real element/entry texts and are therefore still split normally.
func environmentCollectionParts(text, delimiter string) []string {
	if text == "" {
		return nil
	}
	return splitContribution(text, delimiter)
}

func resolutionIssueKind(err error) valueIssueKind {
	if errors.Is(err, errSameTierConflict) || errors.Is(err, errMultipleResolutionTiers) ||
		errors.Is(err, errKeyedDuplicate) {
		return valueIssueResolutionConflict
	}
	return valueIssueInvalidInput
}

func appendUniqueOrigin(values []InputOriginID, value InputOriginID) []InputOriginID {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func findPlanIndex(plans []valueCandidatePlan, producerKind ValueProducerKind, tier ResolutionTier, order uint64) int {
	for i, plan := range plans {
		if plan.producerKind == producerKind && plan.tier == tier && plan.order == order {
			return i
		}
	}
	return -1
}

type memoValue struct {
	fn    func() (any, error)
	done  bool
	value any
	err   error
}

func memoizeValue(fn func() (any, error)) *memoValue {
	return &memoValue{
		fn: fn,
	}
}

func (memo *memoValue) get() (any, error) {
	if !memo.done {
		memo.value, memo.err = memo.fn()
		memo.done = true
		memo.fn = nil
	}
	return memo.value, memo.err
}

type memoSequence struct {
	fn    func() ([]any, error)
	done  bool
	value []any
	err   error
}

func memoizeSequence(fn func() ([]any, error)) *memoSequence {
	return &memoSequence{
		fn: fn,
	}
}

func (memo *memoSequence) get() ([]any, error) {
	if !memo.done {
		memo.value, memo.err = memo.fn()
		memo.done = true
		memo.fn = nil
	}
	return append([]any(nil), memo.value...), memo.err
}
