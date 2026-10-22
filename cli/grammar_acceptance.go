package cli

import (
	"slices"
	"strings"
)

type flagGrammarSemantics struct {
	placement         FlagPlacement
	occurrences       semanticCountRange
	values            semanticCountRange
	delimiter         string
	terminator        string
	hyphenValues      HyphenValueMode
	tokenLanguageName string
	tokenLanguage     tokenLanguage
}

func prepareFlagGrammar(value flagAcceptanceInput, destination destinationAcceptanceInput, dialect dialectInput) (flagGrammarSemantics, bool) {
	occurrences := Count.ZeroOrMore()
	if value.occurrencesSet {
		occurrences = value.occurrences
	}
	var values CountRange
	switch value.producer {
	case flagProducerBool, flagProducerCount:
		values = Count.Exactly(0)
	case flagProducerScalar, flagProducerSequence, flagProducerKeyed:
		values = Count.Exactly(1)
	case flagProducerTokens:
		values = value.tokenCount
	default:
		return flagGrammarSemantics{}, false
	}
	if value.valuesSet {
		if value.producer != flagProducerSequence && value.producer != flagProducerKeyed {
			return flagGrammarSemantics{}, false
		}
		values = value.valuesPerOccurrence
	}
	placement := dialect.DefaultFlagPlacement
	if value.placementSet {
		mapped, ok := prepareFlagPlacement(value.placement)
		if !ok {
			return flagGrammarSemantics{}, false
		}
		placement = mapped
	}
	hyphenValues := HyphenValueDisallow
	if value.hyphenValuesSet {
		mapped, ok := prepareHyphenValueMode(value.hyphenValues)
		if !ok {
			return flagGrammarSemantics{}, false
		}
		hyphenValues = mapped
		if mapped == HyphenValueNegativeNumbers && !destinationUsesBuiltinSignedInteger(destination) {
			return flagGrammarSemantics{}, false
		}
	}
	delimiter := ""
	if value.delimiterSet {
		if value.delimiter == "" ||
			value.producer != flagProducerSequence && value.producer != flagProducerKeyed {
			return flagGrammarSemantics{}, false
		}
		delimiter = value.delimiter
	}
	terminator := ""
	if value.terminatorSet {
		if value.terminator == "" || strings.ContainsRune(value.terminator, '\x00') {
			return flagGrammarSemantics{}, false
		}
		terminator = value.terminator
	}
	semanticOccurrences, ok := prepareCountRange(occurrences)
	if !ok {
		return flagGrammarSemantics{}, false
	}
	semanticValues, ok := prepareCountRange(values)
	if !ok {
		return flagGrammarSemantics{}, false
	}
	if value.producer == flagProducerTokens && values.minimum < 1 {
		return flagGrammarSemantics{}, false
	}
	if (value.producer == flagProducerSequence || value.producer == flagProducerKeyed) && values.minimum < 1 {
		return flagGrammarSemantics{}, false
	}
	if delimiter != "" && semanticValues.Minimum() < 1 {
		return flagGrammarSemantics{}, false
	}
	var tokenLanguageName string
	var tokenLanguage tokenLanguage
	if value.producer == flagProducerTokens {
		tokenLanguage, _, ok = eraseTokenLanguageAny(value.tokenLanguage)
		if !ok || !tokenLanguage.Valid() {
			return flagGrammarSemantics{}, false
		}
		tokenLanguageName = tokenLanguage.Name()
	}
	grammar := flagGrammarSemantics{
		placement:         placement,
		occurrences:       semanticOccurrences,
		values:            semanticValues,
		delimiter:         delimiter,
		terminator:        terminator,
		hyphenValues:      hyphenValues,
		tokenLanguageName: tokenLanguageName,
		tokenLanguage:     tokenLanguage,
	}
	return grammar, true
}

func prepareArgumentTermGrammar(draft *declarationCandidate, term argumentTermRef, value argumentTermAcceptanceInput, destination destinationAcceptanceInput) bool {
	count := value.count
	allocation := semanticArgumentAllocationUnspecified
	if value.optionalSet {
		if value.kind != argumentTermValue || !value.optional {
			return false
		}
		count = Count.Optional()
	}
	if value.allocationSet {
		switch value.allocation {
		case argumentAllocationGreedy:
			allocation = semanticArgumentAllocationGreedy
		case argumentAllocationMinimal:
			allocation = semanticArgumentAllocationMinimal
		default:
			return false
		}
		semanticCount, ok := prepareCountRange(count)
		if !ok {
			return false
		}
		maximum, bounded := semanticCount.Maximum()
		if bounded && maximum == semanticCount.Minimum() {
			return false
		}
	}
	hyphenValues := HyphenValueDisallow
	if value.hyphenValuesSet {
		mapped, ok := prepareHyphenValueMode(value.hyphenValues)
		if !ok {
			return false
		}
		hyphenValues = mapped
		if mapped == HyphenValueNegativeNumbers && !destinationUsesBuiltinSignedInteger(destination) {
			return false
		}
	}

	input := argumentTermGrammarInput{
		Allocation:   allocation,
		HyphenValues: hyphenValues,
	}
	switch value.kind {
	case argumentTermValue, argumentTermSequence, argumentTermRemainder, argumentTermPassThrough:
		semanticCount, ok := prepareCountRange(count)
		if !ok {
			return false
		}
		input.Count = semanticCount
		switch value.kind {
		case argumentTermValue, argumentTermSequence:
			input.Kind = ArgumentTermValue
		case argumentTermRemainder:
			input.Kind = ArgumentTermRemainder
		case argumentTermPassThrough:
			input.Kind = ArgumentTermPassThrough
		case argumentTermInvalid, argumentTermTokens, argumentTermLiteral:
			panic("cli: unreachable argument term kind")
		}
	case argumentTermTokens:
		semanticCount, ok := prepareCountRange(count)
		if !ok {
			return false
		}
		language, syntax, ok := eraseTokenLanguageAny(value.tokenLanguage)
		if !ok || !language.Valid() {
			return false
		}
		input.Kind = ArgumentTermTokens
		input.Count = semanticCount
		input.TokenName = language.Name()
		input.TokenSyntax = syntax
		input.TokenLanguage = language
	case argumentTermLiteral:
		if value.literal == "" || value.allocationSet || value.hyphenValuesSet || value.optionalSet {
			return false
		}
		input.Kind = ArgumentTermLiteral
		input.Count = boundedCount(1, 1)
		input.Literal = value.literal
		input.EndOfOptions = value.endOfOptions
	default:
		return false
	}
	draft.ArgumentTermGrammar(term, input)
	return true
}

func semanticPresentation(description Description, examples []Example, hidden bool) presentation {
	return snapshotPresentation(description, examples, hidden)
}

func flagPresentation(value flagAcceptanceInput) (presentation, string, bool, bool) {
	description := Description{
		Short: value.description,
	}
	return snapshotPresentation(
		description,
		nil,
		value.hidden,
	), value.valueName, value.required, value.localeSelector
}

func argumentPresentation(value argumentTermAcceptanceInput) presentation {
	description := Description{
		Short: value.description,
	}
	return snapshotPresentation(
		description,
		nil,
		value.hidden,
	)
}

func flagReleaseLifecycle(value flagAcceptanceInput) ReleaseLifecycle {
	return value.release
}

func argumentReleaseLifecycle(value argumentTermAcceptanceInput) (ReleaseLifecycle, bool) {
	return value.release, value.releaseSet
}

func applySpellingDirective(config spellingAcceptanceConfig, directive spellingDirectiveAcceptanceInput) (spellingAcceptanceConfig, bool) {
	switch directive.kind {
	case flagSpellingDirectiveValueForms:
		config.forms = append([]FlagValueForm(nil), directive.forms...)
		config.formsSet = true
	case flagSpellingDirectiveFixedValue:
		config.fixed = true
		config.fixedValue = directive.value
		source, ok := producerSource(directive.sourceOptions)
		if !ok {
			return spellingAcceptanceConfig{}, false
		}
		config.fixedSensitive = source.Sensitive()
	case flagSpellingDirectiveImplicitValue:
		config.implicit = true
		config.implicitVal = directive.value
		source, ok := producerSource(directive.sourceOptions)
		if !ok {
			return spellingAcceptanceConfig{}, false
		}
		config.implicitSensitive = source.Sensitive()
	case flagSpellingDirectiveSensitive:
		config.sensitive = true
	case flagSpellingDirectiveHidden:
		config.hidden = true
	case flagSpellingDirectiveRelease:
		if config.hasRelease {
			return spellingAcceptanceConfig{}, false
		}
		config.hasRelease = true
		config.lifecycle = directive.lifecycle
	default:
		return spellingAcceptanceConfig{}, false
	}
	return config, true
}

func prepareFlagSpellingGrammar(kind flagSpellingKind, producer flagProducerKind, config spellingAcceptanceConfig) (FlagValueMode, []flagValueForm, bool, bool) {
	if producer == flagProducerCount && (config.formsSet || config.fixed || config.implicit) {
		return FlagValueModeUnknown, nil, false, false
	}
	if producer == flagProducerBool && (config.formsSet || config.implicit) {
		return FlagValueModeUnknown, nil, false, false
	}
	if config.fixed && config.formsSet {
		return FlagValueModeUnknown, nil, false, false
	}
	if producer == flagProducerBool || producer == flagProducerCount || config.fixed {
		return FlagValueModeNone, nil, config.sensitive, true
	}
	forms := config.forms
	if !config.formsSet {
		forms = defaultFlagValueForms(kind, config.implicit)
	}
	semanticForms := make([]flagValueForm, 0, len(forms))
	for _, form := range forms {
		mapped, ok := prepareFlagValueForm(form)
		if !ok || (kind == flagSpellingLong && mapped == flagValueFormJoined) {
			return FlagValueModeUnknown, nil, false, false
		}
		semanticForms = append(semanticForms, mapped)
	}
	if len(semanticForms) == 0 {
		return FlagValueModeUnknown, nil, false, false
	}
	mode := FlagValueModeRequired
	if config.implicit {
		if slices.Contains(semanticForms, flagValueFormSeparate) {
			return FlagValueModeUnknown, nil, false, false
		}
		mode = FlagValueModeOptional
	}
	return mode, semanticForms, config.sensitive, true
}

func defaultFlagValueForms(kind flagSpellingKind, implicit bool) []FlagValueForm {
	if implicit {
		switch kind {
		case flagSpellingLong:
			return []FlagValueForm{FlagValueFormAssigned}
		case flagSpellingShort:
			return []FlagValueForm{FlagValueFormJoined}
		case flagSpellingLiteral:
			return nil
		default:
			return nil
		}
	}
	switch kind {
	case flagSpellingLong:
		return []FlagValueForm{FlagValueFormSeparate, FlagValueFormAssigned}
	case flagSpellingShort:
		return []FlagValueForm{FlagValueFormSeparate, FlagValueFormJoined}
	case flagSpellingLiteral:
		return []FlagValueForm{FlagValueFormSeparate}
	default:
		return nil
	}
}

func prepareCountRange(count CountRange) (semanticCountRange, bool) {
	if !count.valid || count.minimum < 0 {
		return semanticCountRange{}, false
	}
	if count.unbounded {
		return unboundedCount(count.minimum), true
	}
	if count.maximum < count.minimum {
		return semanticCountRange{}, false
	}
	return boundedCount(count.minimum, count.maximum), true
}

func prepareFlagPlacement(value FlagPlacement) (FlagPlacement, bool) {
	switch value {
	case FlagPlacementCommand:
		return FlagPlacementCommand, true
	case FlagPlacementSubtree:
		return FlagPlacementSubtree, true
	default:
		return FlagPlacementUnknown, false
	}
}

func prepareFlagSpellingKind(value flagSpellingKind) (FlagSpellingKind, bool) {
	switch value {
	case flagSpellingLong:
		return FlagSpellingLong, true
	case flagSpellingShort:
		return FlagSpellingShort, true
	case flagSpellingLiteral:
		return FlagSpellingLiteral, true
	default:
		return FlagSpellingUnknown, false
	}
}

func prepareFlagValueForm(value FlagValueForm) (flagValueForm, bool) {
	switch value {
	case FlagValueFormSeparate:
		return flagValueFormSeparate, true
	case FlagValueFormJoined:
		return flagValueFormJoined, true
	case FlagValueFormAssigned:
		return flagValueFormAssigned, true
	default:
		return flagValueFormInvalid, false
	}
}

func prepareHyphenValueMode(value HyphenValueMode) (HyphenValueMode, bool) {
	switch value {
	case HyphenValueDisallow:
		return HyphenValueDisallow, true
	case HyphenValueNegativeNumbers:
		return HyphenValueNegativeNumbers, true
	case HyphenValueAllow:
		return HyphenValueAllow, true
	default:
		return HyphenValueUnknown, false
	}
}

func applyDialectDirective(result dialectInput, directive dialectDirectiveDefinition, endOfOptionsConfigured bool) (dialectInput, bool, bool) {
	switch directive.kind {
	case dialectDirectiveFlagOrder:
		value, ok := prepareFlagOrder(directive.flagOrder)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.FlagOrder = value
	case dialectDirectiveShortFlagClustering:
		value, ok := prepareShortFlagClustering(directive.clustering)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.ShortClustering = value
	case dialectDirectiveFlagMatching:
		value, ok := prepareNameMatching(directive.matching)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.FlagMatching = value
	case dialectDirectiveCommandMatching:
		value, ok := prepareNameMatching(directive.matching)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.CommandMatching = value
	case dialectDirectiveFlagNormalization:
		value, ok := prepareNameNormalization(directive.normalization)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.FlagNormalization = value
	case dialectDirectiveCommandNormalization:
		value, ok := prepareNameNormalization(directive.normalization)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.CommandNormalization = value
	case dialectDirectiveDashlessFlags:
		value, ok := prepareDashlessFlagMode(directive.dashless)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.DashlessFlags = value
	case dialectDirectiveEndOfOptions:
		if endOfOptionsConfigured || !validEndOfOptionsTokens(directive.tokens) {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		endOfOptionsConfigured = true
		result.EndOfOptions = append([]string(nil), directive.tokens...)
	case dialectDirectiveResponseFiles:
		result.ResponseFiles = directive.enabled
	case dialectDirectiveDefaultFlagPlacement:
		value, ok := prepareFlagPlacement(directive.placement)
		if !ok {
			return dialectInput{}, endOfOptionsConfigured, false
		}
		result.DefaultFlagPlacement = value
	default:
		return dialectInput{}, endOfOptionsConfigured, false
	}
	return result, endOfOptionsConfigured, true
}

func validSemanticDialectInput(input dialectInput) bool {
	return validDialectInput(input)
}

func validEndOfOptionsTokens(tokens []string) bool {
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if token == "" || strings.ContainsRune(token, '\x00') {
			return false
		}
		if _, exists := seen[token]; exists {
			return false
		}
		seen[token] = struct{}{}
	}
	return true
}

func prepareFlagOrder(value FlagOrder) (FlagOrder, bool) {
	switch value {
	case FlagOrderBeforeArguments:
		return FlagOrderBeforeArguments, true
	case FlagOrderInterspersed:
		return FlagOrderInterspersed, true
	default:
		return FlagOrderUnknown, false
	}
}

func prepareShortFlagClustering(value ShortFlagClustering) (ShortFlagClustering, bool) {
	switch value {
	case ShortFlagClusteringDisabled:
		return ShortFlagClusteringDisabled, true
	case ShortFlagClusteringEnabled:
		return ShortFlagClusteringEnabled, true
	default:
		return ShortFlagClusteringUnknown, false
	}
}

func prepareNameMatching(value NameMatching) (NameMatching, bool) {
	switch value {
	case NameMatchingExact:
		return NameMatchingExact, true
	case NameMatchingUniquePrefix:
		return NameMatchingUniquePrefix, true
	default:
		return NameMatchingUnknown, false
	}
}

func prepareNameNormalization(value NameNormalization) (NameNormalization, bool) {
	switch value {
	case NameNormalizationExact:
		return NameNormalizationExact, true
	case NameNormalizationASCIICaseFold:
		return NameNormalizationASCIICaseFold, true
	default:
		return NameNormalizationUnknown, false
	}
}

func prepareDashlessFlagMode(value DashlessFlagMode) (DashlessFlagMode, bool) {
	switch value {
	case DashlessFlagModeDisabled:
		return DashlessFlagModeDisabled, true
	case DashlessFlagModeInitialCluster:
		return DashlessFlagModeInitialCluster, true
	default:
		return DashlessFlagModeUnknown, false
	}
}

type grammarFlagDraft struct {
	ref               acceptanceFlagRef
	placement         FlagPlacement
	occurrences       semanticCountRange
	values            semanticCountRange
	delimiter         string
	terminator        string
	hyphenValues      HyphenValueMode
	tokenLanguageName string
	tokenLanguage     tokenLanguage
	position          declarationPosition
}

type formDraft struct {
	ref      argumentFormRef
	enabled  bool
	position declarationPosition
}

type grammarDraft struct {
	dialects    map[CommandID]dialect
	entries     []commandEntryGrammar
	actions     []actionCommandGrammar
	actionFlags []semanticActionFlagGrammar
	flags       map[FlagID]grammarFlagDraft
	spellings   map[FlagID][]flagSpellingGrammar
	forms       map[ArgumentFormID]formDraft
	terms       map[ArgumentFormID][]argumentTermGrammar
}

func (draft *declarationCandidate) grammarState() *grammarDraft {
	draft.mustOpen()
	if draft.grammar == nil {
		draft.grammar = &grammarDraft{
			dialects:  make(map[CommandID]dialect),
			flags:     make(map[FlagID]grammarFlagDraft),
			spellings: make(map[FlagID][]flagSpellingGrammar),
			forms:     make(map[ArgumentFormID]formDraft),
			terms:     make(map[ArgumentFormID][]argumentTermGrammar),
		}
	}
	return draft.grammar
}

// Dialect records the already-effective dialect for one command. Public
// declaration inheritance is resolved by cli acceptance before this call; the Draft
// stores only the effective semantic value.
func (draft *declarationCandidate) Dialect(command acceptanceCommandRef, input dialectInput) {
	state := draft.grammarState()
	if !command.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	if !validDialectInput(input) {
		draft.rejectStructural(0)
		return
	}
	state.dialects[command.ID()] = acceptDialect(input)
}

func (draft *declarationCandidate) CommandEntryGrammar(entry commandEntryRef, kind CommandEntryKind, name string, enabled bool) {
	state := draft.grammarState()
	if !entry.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.commandEntryRecordForRef(entry)
	if !ok {
		draft.reject(0)
		return
	}
	input := commandEntryGrammarInput{
		ID:       entry.ID(),
		Parent:   record.parent,
		Target:   record.target,
		Kind:     kind,
		Name:     name,
		Position: record.meta.position,
		Enabled:  enabled,
	}
	if !validCommandEntryGrammarInput(input) {
		draft.rejectStructural(record.meta.position)
		return
	}
	state.entries = append(state.entries, acceptCommandEntryGrammar(input))
}

func (draft *declarationCandidate) ActionCommandGrammar(entry actionEntryRef, action intrinsicAction, name string, enabled bool) {
	state := draft.grammarState()
	if !entry.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.actionEntryRecordForRef(entry)
	if !ok {
		draft.reject(0)
		return
	}
	input := actionCommandGrammarInput{
		ID:       entry.ID(),
		Parent:   record.parent,
		Action:   action,
		Name:     name,
		Position: record.meta.position,
		Enabled:  enabled,
	}
	if !validActionCommandGrammarInput(input) {
		draft.rejectStructural(record.meta.position)
		return
	}
	state.actions = append(state.actions, acceptActionCommandGrammar(input))
}

func (draft *declarationCandidate) ActionFlagGrammar(entry actionEntryRef, action intrinsicAction, kind FlagSpellingKind, text string, placement FlagPlacement, enabled bool) {
	state := draft.grammarState()
	if !entry.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.actionEntryRecordForRef(entry)
	if !ok {
		draft.reject(0)
		return
	}
	input := actionFlagGrammarInput{
		ID:        entry.ID(),
		Parent:    record.parent,
		Action:    action,
		Kind:      kind,
		Text:      text,
		Placement: placement,
		Position:  record.meta.position,
		Enabled:   enabled,
	}
	if !validActionFlagGrammarInput(input) {
		draft.rejectStructural(record.meta.position)
		return
	}
	state.actionFlags = append(state.actionFlags, acceptActionFlagGrammar(input))
}

func (draft *declarationCandidate) ActionFlagGrammarDefaulted(entry actionEntryRef, action intrinsicAction, kind FlagSpellingKind, text string, resolvedPlacement FlagPlacement, enabled bool) {
	state := draft.grammarState()
	if !entry.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.actionEntryRecordForRef(entry)
	if !ok {
		draft.reject(0)
		return
	}
	input := actionFlagGrammarInput{
		ID:        entry.ID(),
		Parent:    record.parent,
		Action:    action,
		Kind:      kind,
		Text:      text,
		Placement: resolvedPlacement,
		Position:  record.meta.position,
		Enabled:   enabled,
	}
	if !validActionFlagGrammarInput(input) {
		draft.rejectStructural(record.meta.position)
		return
	}
	state.actionFlags = append(state.actionFlags, acceptDefaultedActionFlagGrammar(input))
}

func (draft *declarationCandidate) FlagGrammar(flag acceptanceFlagRef, grammar flagGrammarSemantics) {
	state := draft.grammarState()
	if !flag.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.flagRecordForRef(flag)
	if !ok {
		draft.reject(0)
		return
	}
	if _, exists := state.flags[flag.ID()]; exists {
		draft.preflight = append(draft.preflight, semanticRejection(record.meta.position))
		return
	}
	state.flags[flag.ID()] = grammarFlagDraft{
		ref:               flag,
		placement:         grammar.placement,
		occurrences:       grammar.occurrences,
		values:            grammar.values,
		delimiter:         grammar.delimiter,
		terminator:        grammar.terminator,
		hyphenValues:      grammar.hyphenValues,
		tokenLanguageName: grammar.tokenLanguageName,
		tokenLanguage:     grammar.tokenLanguage,
		position:          record.meta.position,
	}
}

func (draft *declarationCandidate) FlagSpellingGrammar(spelling acceptanceFlagSpellingRef, text string, mode FlagValueMode, forms []flagValueForm, sensitive bool) {
	draft.FlagSpellingGrammarWithKind(spelling, FlagSpellingLiteral, text, mode, forms, sensitive)
}

func (draft *declarationCandidate) FlagSpellingGrammarWithKind(spelling acceptanceFlagSpellingRef, kind FlagSpellingKind, text string, mode FlagValueMode, forms []flagValueForm, sensitive bool) {
	state := draft.grammarState()
	if !spelling.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.flagSpellingRecordForRef(spelling)
	if !ok {
		draft.reject(0)
		return
	}
	input := flagSpellingGrammarInput{
		ID:        spelling.ID(),
		Flag:      record.flag,
		Kind:      kind,
		Text:      text,
		Mode:      mode,
		Forms:     forms,
		Sensitive: sensitive,
		Position:  record.meta.position,
	}
	if !validFlagSpellingGrammarInput(input) {
		draft.rejectStructural(record.meta.position)
		return
	}
	state.spellings[record.flag] = append(state.spellings[record.flag], acceptFlagSpellingGrammarWithKind(input))
}

func (draft *declarationCandidate) FlagSpellingFixedValue(spelling acceptanceFlagSpellingRef, value any, sensitive bool) {
	draft.flagSpellingValue(spelling, value, sensitive, flagSpellingDirectiveFixedValue)
}

func (draft *declarationCandidate) FlagSpellingImplicitValue(spelling acceptanceFlagSpellingRef, value any, sensitive bool) {
	draft.flagSpellingValue(spelling, value, sensitive, flagSpellingDirectiveImplicitValue)
}

func (draft *declarationCandidate) flagSpellingValue(spelling acceptanceFlagSpellingRef, value any, sensitive bool, kind flagSpellingDirectiveKind) {
	state := draft.grammarState()
	if !spelling.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.flagSpellingRecordForRef(spelling)
	if !ok {
		draft.reject(0)
		return
	}
	flag, flagOK := draft.flagRecord(record.flag)
	if !flagOK {
		draft.reject(record.meta.position)
		return
	}
	definition, definitionOK := draft.ExistingDestinationDefinition(flag.destination)
	if !definitionOK {
		draft.reject(record.meta.position)
		return
	}
	snapshot := definition.snapshot
	items := state.spellings[record.flag]
	for index := range items {
		if items[index].id != spelling.ID() {
			continue
		}
		spellingGrammar := items[index]
		switch kind {
		case flagSpellingDirectiveFixedValue:
			if spellingGrammar.hasFixedValue || spellingGrammar.hasImplicitValue || spellingGrammar.mode != FlagValueModeNone {
				draft.rejectStructural(record.meta.position)
				return
			}
			if snapshot == nil {
				panic("semantic: invalid fixed flag-spelling snapshot")
			}
			frozen, ok := snapshot(value)
			if !ok {
				panic("semantic: invalid fixed flag-spelling value")
			}
			spellingGrammar.fixedValue = frozen
			spellingGrammar.hasFixedValue = true
			spellingGrammar.fixedSensitive = sensitive
			spellingGrammar.fixedSnapshot = snapshot
		case flagSpellingDirectiveImplicitValue:
			if spellingGrammar.hasFixedValue || spellingGrammar.hasImplicitValue || spellingGrammar.mode != FlagValueModeOptional {
				draft.rejectStructural(record.meta.position)
				return
			}
			if snapshot == nil {
				panic("semantic: invalid implicit flag-spelling snapshot")
			}
			frozen, ok := snapshot(value)
			if !ok {
				panic("semantic: invalid implicit flag-spelling value")
			}
			spellingGrammar.implicitValue = frozen
			spellingGrammar.hasImplicitValue = true
			spellingGrammar.implicitSensitive = sensitive
			spellingGrammar.implicitSnapshot = snapshot
		default:
			panic("semantic: invalid flag-spelling value directive")
		}
		items[index] = spellingGrammar
		state.spellings[record.flag] = items
		return
	}
	draft.reject(record.meta.position)
}

func (draft *declarationCandidate) ArgumentFormGrammar(form argumentFormRef) {
	state := draft.grammarState()
	if !form.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.argumentFormRecordForRef(form)
	if !ok {
		draft.reject(0)
		return
	}
	if _, exists := state.forms[form.ID()]; exists {
		draft.preflight = append(draft.preflight, semanticRejection(record.meta.position))
		return
	}
	state.forms[form.ID()] = formDraft{
		ref:      form,
		enabled:  true,
		position: record.meta.position,
	}
}

func (draft *declarationCandidate) ArgumentTermGrammar(term argumentTermRef, input argumentTermGrammarInput) {
	state := draft.grammarState()
	if !term.OwnedBy(draft.owner) {
		draft.reject(0)
		return
	}
	_, record, ok := draft.argumentTermRecordForRef(term)
	if !ok {
		draft.reject(0)
		return
	}
	input.ID = term.ID()
	input.Position = record.meta.position
	destination, hasDestination := record.Destination()
	input.Destination = destination
	input.HasDestination = hasDestination
	if input.Kind == ArgumentTermLiteral && hasDestination {
		draft.preflight = append(draft.preflight, semanticRejection(record.meta.position))
		return
	}
	if input.Kind != ArgumentTermLiteral && !hasDestination {
		draft.preflight = append(draft.preflight, semanticRejection(record.meta.position))
		return
	}
	if !validArgumentTermGrammarInput(input) {
		draft.rejectStructural(record.meta.position)
		return
	}
	state.terms[record.form] = append(state.terms[record.form], acceptArgumentTermGrammar(input))
}

func (draft *declarationCandidate) buildGrammar() (grammarModel, bool, []rejection) {
	if draft.grammar == nil {
		return grammarModel{}, false, nil
	}
	state := draft.grammar
	root, ok := draft.rootCommand()
	if !ok {
		return grammarModel{}, true, []rejection{{
			Kind: rejectionStructuralInvalidity,
		}}
	}
	input := grammarInput{
		Root:            root,
		Dialects:        state.dialects,
		ProfileDialects: make(map[GrammarProfileID]dialect, len(draft.input.GrammarProfiles)),
		ProfileCommands: make(map[GrammarProfileID]CommandID, len(draft.input.GrammarProfiles)),
		ProfileOrder:    make([]GrammarProfileID, 0, len(draft.input.GrammarProfiles)),
		ActionCommands:  state.actions,
		ActionFlags:     state.actionFlags,
	}
	entryProfiles := make(map[CommandEntryID]GrammarProfileID, len(draft.input.CommandEntries))
	for _, record := range draft.input.CommandEntries {
		entryProfiles[record.meta.id] = record.meta.profile
	}
	for _, entry := range state.entries {
		if profile := entryProfiles[entry.id]; profile.present() {
			entry.profile = profile
		}
		input.CommandEntries = append(input.CommandEntries, entry)
	}
	for _, profile := range draft.input.GrammarProfiles {
		dialect, ok := profile.Dialect()
		if !ok {
			return grammarModel{}, true, []rejection{semanticRejection(profile.meta.position)}
		}
		input.ProfileDialects[profile.meta.id] = dialect
		input.ProfileCommands[profile.meta.id] = profile.command
		input.ProfileOrder = append(input.ProfileOrder, profile.meta.id)
	}
	for _, record := range draft.input.Flags {
		draft, ok := state.flags[record.meta.id]
		if !ok {
			return grammarModel{}, true, []rejection{semanticRejection(record.meta.position)}
		}
		spellings := state.spellings[record.meta.id]
		flagInput := flagGrammarInput{
			ID:                  record.meta.id,
			Command:             record.command,
			Destination:         record.destination,
			Placement:           draft.placement,
			Occurrences:         draft.occurrences,
			ValuesPerOccurrence: draft.values,
			Delimiter:           draft.delimiter,
			Terminator:          draft.terminator,
			HyphenValues:        draft.hyphenValues,
			TokenLanguageName:   draft.tokenLanguageName,
			TokenLanguage:       draft.tokenLanguage,
			Spellings:           spellings,
			Position:            draft.position,
		}
		if !validFlagGrammarInput(flagInput) {
			return grammarModel{}, true, []rejection{semanticRejection(record.meta.position)}
		}
		flag := acceptFlagGrammar(flagInput)
		if profile := record.meta.profile; profile.present() {
			flag.profile = profile
		}
		input.Flags = append(input.Flags, flag)
	}
	for _, record := range draft.input.ArgumentForms {
		draft, ok := state.forms[record.meta.id]
		if !ok {
			return grammarModel{}, true, []rejection{semanticRejection(record.meta.position)}
		}
		input2 := argumentFormGrammarInput{
			ID:       record.meta.id,
			Command:  record.command,
			Terms:    state.terms[record.meta.id],
			Position: draft.position,
			Enabled:  draft.enabled,
		}
		form := acceptArgumentFormGrammar(input2)
		if profile := record.meta.profile; profile.present() {
			form.profile = profile
		}
		input.ArgumentForms = append(input.ArgumentForms, form)
	}
	model := acceptGrammarModel(input)
	rejections := validateGrammarStructure(model)
	rejections = append(
		rejections,
		validateLifecycleLexicalCollisions(draft.application.release, draft.input, model)...,
	)
	return model, true, acceptAcceptanceDecision(acceptanceBoundaryProviderBatch, rejections).Rejections()
}

func (draft *declarationCandidate) rootCommand() (CommandID, bool) {
	var root CommandID
	for _, record := range draft.input.Commands {
		if _, parent := record.Parent(); parent {
			continue
		}
		if root.present() {
			return CommandID(0), false
		}
		root = record.meta.id
	}
	return root, root.present()
}

func (draft *declarationCandidate) flagRecord(id FlagID) (flagRecord, bool) {
	for _, record := range draft.input.Flags {
		if record.meta.id == id {
			return record, true
		}
	}
	return flagRecord{}, false
}

func grammarDraftFromModel(draft *declarationCandidate, model grammarModel) *grammarDraft {
	state := &grammarDraft{
		dialects:    make(map[CommandID]dialect),
		entries:     model.AllCommandEntries(),
		actions:     model.AllActionCommands(),
		actionFlags: model.AllActionFlags(),
		flags:       make(map[FlagID]grammarFlagDraft),
		spellings:   make(map[FlagID][]flagSpellingGrammar),
		forms:       make(map[ArgumentFormID]formDraft),
		terms:       make(map[ArgumentFormID][]argumentTermGrammar),
	}
	for _, record := range draft.input.Commands {
		if dialect, ok := model.Dialect(record.meta.id); ok {
			state.dialects[record.meta.id] = dialect
		}
	}
	for _, flag := range model.AllFlags() {
		tokenLanguage, _ := flag.TokenLanguage()
		state.flags[flag.id] = grammarFlagDraft{
			ref:               issueStagingRef(draft.owner, flag.id),
			placement:         flag.placement,
			occurrences:       flag.occurrences,
			values:            flag.valuesPerOccurrence,
			hyphenValues:      flag.hyphenValues,
			tokenLanguageName: flag.tokenLanguageName,
			tokenLanguage:     tokenLanguage,
			position:          flag.position,
		}
		if delimiter, ok := flag.Delimiter(); ok {
			flagGrammar := state.flags[flag.id]
			flagGrammar.delimiter = delimiter
			state.flags[flag.id] = flagGrammar
		}
		if terminator, ok := flag.Terminator(); ok {
			flagGrammar := state.flags[flag.id]
			flagGrammar.terminator = terminator
			state.flags[flag.id] = flagGrammar
		}
		state.spellings[flag.id] = flag.Spellings()
	}
	for _, form := range model.AllArgumentForms() {
		state.forms[form.id] = formDraft{
			ref:      issueStagingRef(draft.owner, form.id),
			enabled:  form.enabled,
			position: form.position,
		}
		state.terms[form.id] = form.Terms()
	}
	return state
}

func grammarProfileCanSee(owner, target GrammarProfileID) bool {
	return !target.present() || owner == target
}

func (draft *declarationCandidate) validateGrammarProfiles(commands map[CommandID]commandRecord, destinations map[ValueID]destinationRecord, flags map[FlagID]flagRecord, forms map[ArgumentFormID]argumentFormRecord) []rejection {
	profiles := make(map[GrammarProfileID]grammarProfileRecord, len(draft.input.GrammarProfiles))
	profilesByCommand := make(map[CommandID][]grammarProfileRecord)
	var rejections []rejection
	for _, record := range draft.input.GrammarProfiles {
		if _, ok := commands[record.command]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		selector, selectorOK := record.Selector()
		_, dialectOK := record.Dialect()
		if !selectorOK || !dialectOK {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		selectorRecord, ok := destinations[selector]
		if !ok || selectorRecord.command != record.command || selectorRecord.meta.profile.present() {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		profiles[record.meta.id] = record
		profilesByCommand[record.command] = append(profilesByCommand[record.command], record)
	}

	validProfileOwner := func(profile GrammarProfileID, command CommandID, position declarationPosition) bool {
		if !profile.present() {
			return true
		}
		record, ok := profiles[profile]
		if !ok || record.command != command {
			rejections = append(rejections, semanticRejection(position))
			return false
		}
		return true
	}

	activationProfiles := draft.commandActivationProfiles(commands)
	referenceVisible := func(ownerCommand CommandID, ownerProfile GrammarProfileID, target destinationRecord) bool {
		if !target.meta.profile.present() || ownerProfile == target.meta.profile {
			return true
		}
		return slices.Contains(activationProfiles[ownerCommand], target.meta.profile)
	}

	// Every profile-bearing declaration belongs to one branch of its owning command.
	for _, record := range draft.input.Destinations {
		validProfileOwner(record.meta.profile, record.command, record.meta.position)
	}
	for _, record := range draft.input.Flags {
		if validProfileOwner(record.meta.profile, record.command, record.meta.position) {
			if destination, ok := destinations[record.destination]; ok &&
				!referenceVisible(record.command, record.meta.profile, destination) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
	}
	for _, record := range draft.input.ArgumentForms {
		validProfileOwner(record.meta.profile, record.command, record.meta.position)
		for _, selection := range record.Selections() {
			destination, ok := destinations[selection.destination]
			if !ok || !referenceVisible(record.command, record.meta.profile, destination) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
	}
	for _, term := range draft.input.ArgumentTerms {
		form, ok := forms[term.form]
		if !ok {
			continue
		}
		if ValueID, present := term.Destination(); present {
			destination, ok := destinations[ValueID]
			if !ok || !referenceVisible(form.command, form.meta.profile, destination) {
				rejections = append(rejections, semanticRejection(term.meta.position))
			}
		}
	}
	for _, record := range draft.input.CommandEntries {
		validProfileOwner(record.meta.profile, record.parent, record.meta.position)
	}
	for _, record := range draft.input.Defaults {
		if validProfileOwner(record.meta.profile, record.parent, record.meta.position) && draft.grammar != nil {
			if !draft.profileDefaultTargetVisible(record, record.meta.profile) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		if condition, ok := record.Condition(); ok {
			for _, dependency := range condition.Dependencies() {
				destination, exists := destinations[dependency]
				if !exists || !referenceVisible(record.parent, record.meta.profile, destination) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
	}
	for _, record := range draft.input.EnvironmentInputs {
		if validProfileOwner(record.meta.profile, record.command, record.meta.position) {
			destination, ok := destinations[record.destination]
			if !ok || !referenceVisible(record.command, record.meta.profile, destination) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
	}
	for _, record := range draft.input.Providers {
		validProfileOwner(record.meta.profile, record.parent, record.meta.position)
	}
	for _, record := range draft.input.Rules {
		validProfileOwner(record.meta.profile, record.command, record.meta.position)
	}
	for _, record := range draft.input.Advisories {
		validProfileOwner(record.meta.profile, record.command, record.meta.position)
	}
	for _, record := range draft.input.SignalPolicies {
		validProfileOwner(record.meta.profile, record.command, record.meta.position)
	}
	for _, record := range draft.input.ConditionPolicies {
		ownerCommand, targetProfile, ok := conditionTargetProfile(
			record.target,
			commands,
			flags,
			forms,
			draft.input.ArgumentTerms,
		)
		if !ok || !validProfileOwner(record.profile, ownerCommand, record.position) ||
			!grammarProfileCanSee(record.profile, targetProfile) {
			rejections = append(rejections, semanticRejection(record.position))
			continue
		}
		if record.target.Kind() == conditionTargetCommand && record.profile.present() {
			// The owning command's availability/visibility is established before profile selection.
			rejections = append(rejections, semanticRejection(record.position))
		}
		for _, dependency := range record.condition.Dependencies() {
			destination, exists := destinations[dependency]
			if !exists || !referenceVisible(ownerCommand, record.profile, destination) {
				rejections = append(rejections, semanticRejection(record.position))
			}
		}
	}
	for _, destination := range draft.input.Destinations {
		for _, producer := range destination.Producers() {
			ownerProfile := producer.profile
			if !ownerProfile.present() {
				ownerProfile = destination.meta.profile
			}
			if !validProfileOwner(ownerProfile, destination.command, destination.meta.position) {
				continue
			}
			for _, dependency := range producer.Dependencies() {
				target, ok := destinations[dependency.Destination()]
				if !ok || !referenceVisible(destination.command, ownerProfile, target) {
					rejections = append(rejections, semanticRejection(destination.meta.position))
				}
			}
		}
	}

	for command, commandProfiles := range profilesByCommand {
		var selector ValueID
		defaultCount := 0
		explicitProfiles := make([]grammarProfileRecord, 0, len(commandProfiles))
		for _, profile := range commandProfiles {
			candidate, _ := profile.Selector()
			if !selector.present() {
				selector = candidate
			} else if selector != candidate {
				rejections = append(rejections, semanticRejection(profile.meta.position))
			}
			if profile.defaultCase {
				defaultCount++
				if defaultCount > 1 {
					rejections = append(rejections, semanticRejection(profile.meta.position))
				}
				continue
			}
			value, ok := profile.Value()
			if !ok {
				rejections = append(rejections, semanticRejection(profile.meta.position))
				continue
			}
			for _, prior := range explicitProfiles {
				priorValue, priorOK := prior.Value()
				if priorOK && profile.ValuesEqual(priorValue, value) {
					rejections = append(rejections, semanticRejection(profile.meta.position))
					break
				}
			}
			explicitProfiles = append(explicitProfiles, profile)
		}
		if selector.present() {
			rejections = append(
				rejections,
				draft.validateGrammarProfileSelector(
					command,
					selector,
					commandProfiles,
					defaultCount != 0,
					destinations,
					flags,
					forms,
				)...,
			)
		}
	}
	return rejections
}

func (draft *declarationCandidate) commandActivationProfiles(commands map[CommandID]commandRecord) map[CommandID][]GrammarProfileID {
	result := make(map[CommandID][]GrammarProfileID, len(commands))
	if draft.grammar == nil {
		return result
	}
	entryProfiles := make(map[CommandEntryID]GrammarProfileID, len(draft.input.CommandEntries))
	for _, record := range draft.input.CommandEntries {
		entryProfiles[record.meta.id] = record.meta.profile
	}
	direct := make(map[CommandID]GrammarProfileID)
	for _, entry := range draft.grammar.entries {
		if entry.kind == CommandEntryCanonical {
			direct[entry.target] = entryProfiles[entry.id]
		}
	}
	var resolve func(CommandID) []GrammarProfileID
	resolving := make(map[CommandID]struct{})
	resolved := make(map[CommandID]struct{})
	resolve = func(command CommandID) []GrammarProfileID {
		if _, ok := resolved[command]; ok {
			return result[command]
		}
		if _, ok := resolving[command]; ok {
			return nil
		}
		resolving[command] = struct{}{}
		record, ok := commands[command]
		if ok {
			if parent, hasParent := record.Parent(); hasParent {
				result[command] = append(result[command], resolve(parent)...)
			}
		}
		if profile := direct[command]; profile.present() {
			result[command] = append(result[command], profile)
		}
		delete(resolving, command)
		resolved[command] = struct{}{}
		return result[command]
	}
	for command := range commands {
		resolve(command)
	}
	return result
}

func (draft *declarationCandidate) profileDefaultTargetVisible(record defaultRecord, profile GrammarProfileID) bool {
	if draft.grammar == nil {
		return false
	}
	entryProfiles := make(map[CommandEntryID]GrammarProfileID, len(draft.input.CommandEntries))
	for _, entry := range draft.input.CommandEntries {
		entryProfiles[entry.meta.id] = entry.meta.profile
	}
	for _, entry := range draft.grammar.entries {
		if entry.kind != CommandEntryCanonical || entry.parent != record.parent ||
			entry.target != record.target {
			continue
		}
		return grammarProfileCanSee(profile, entryProfiles[entry.id])
	}
	return false
}

func (draft *declarationCandidate) validateLocaleSelector(destinations map[ValueID]destinationRecord, flags map[FlagID]flagRecord, provider bool) []rejection {
	var root CommandID
	for _, command := range draft.input.Commands {
		if _, hasParent := command.Parent(); !hasParent {
			root = command.meta.id
			break
		}
	}

	selectorFlags := make([]flagRecord, 0, 1)
	for _, flag := range draft.input.Flags {
		if flag.localeSelector {
			selectorFlags = append(selectorFlags, flag)
		}
	}
	if len(selectorFlags) == 0 {
		return nil
	}

	var rejections []rejection
	for index, flag := range selectorFlags {
		if index > 0 {
			rejections = append(rejections, semanticRejection(flag.meta.position))
		}
		if !root.present() || flag.command != root {
			rejections = append(rejections, semanticRejection(flag.meta.position))
		}
		destination, ok := destinations[flag.destination]
		if !ok {
			continue
		}
		if destination.command != root {
			rejections = append(rejections, semanticRejection(destination.meta.position))
		}
		rejections = append(rejections, draft.validateEarlySelector(root, destination.meta.id, destinations)...)

		if destination.sensitive {
			rejections = append(rejections, semanticRejection(destination.meta.position))
		}
		for _, producer := range destination.Producers() {
			if producer.source.Sensitive() {
				rejections = append(rejections, semanticRejection(destination.meta.position))
			}
		}
		for _, input := range draft.input.EnvironmentInputs {
			if input.destination == destination.meta.id && input.sensitive {
				rejections = append(rejections, semanticRejection(input.meta.position))
			}
		}
		if draft.grammar != nil {
			for _, spelling := range draft.grammar.spellings[flag.meta.id] {
				if spelling.sensitive {
					rejections = append(rejections, semanticRejection(spelling.position))
				}
			}
		}
		for _, policy := range draft.input.ConditionPolicies {
			target := policy.target
			if target.Kind() == conditionTargetFlag && target.Flag() == flag.meta.id {
				rejections = append(rejections, semanticRejection(policy.position))
			}
		}
	}

	if provider {
		for index := draft.base.flags; index < len(draft.input.Flags); index++ {
			flag := draft.input.Flags[index]
			if flag.localeSelector {
				rejections = append(rejections, semanticRejection(flag.meta.position))
			}
		}
		for index := draft.base.replacements; index < len(draft.input.Replacements); index++ {
			relation := draft.input.Replacements[index]
			source := relation.source
			if source.Kind() != declarationReplacementKindFlag {
				continue
			}
			flag, ok := flags[source.Flag()]
			if ok && flag.localeSelector {
				rejections = append(rejections, semanticRejection(relation.position))
			}
		}
	}
	return rejections
}

func (draft *declarationCandidate) validateEarlySelectorFlag(command CommandID, flag flagRecord) []rejection {
	if flag.meta.profile.present() || draft.grammar == nil {
		return []rejection{semanticRejection(flag.meta.position)}
	}
	dialect, dialectOK := draft.grammar.dialects[command]
	flagGrammar, grammarOK := draft.grammar.flags[flag.meta.id]
	if !dialectOK || !grammarOK || dialect.FlagMatching() != NameMatchingExact {
		return []rejection{semanticRejection(flag.meta.position)}
	}
	var rejections []rejection
	maximum, bounded := flagGrammar.occurrences.Maximum()
	valueMaximum, valueBounded := flagGrammar.values.Maximum()
	if !bounded || maximum > 1 || flagGrammar.values.Minimum() != 1 || !valueBounded || valueMaximum != 1 ||
		flagGrammar.delimiter != "" || flagGrammar.terminator != "" {
		rejections = append(rejections, semanticRejection(flag.meta.position))
	}
	for _, spelling := range draft.grammar.spellings[flag.meta.id] {
		if spelling.mode != FlagValueModeRequired {
			rejections = append(rejections, semanticRejection(spelling.position))
			continue
		}
		for _, form := range spelling.Forms() {
			if form != flagValueFormSeparate && form != flagValueFormAssigned {
				rejections = append(rejections, semanticRejection(spelling.position))
			}
		}
		if spelling.kind == FlagSpellingShort &&
			(dialect.ShortClustering() != ShortFlagClusteringDisabled || dialect.DashlessFlags() != DashlessFlagModeDisabled) {
			rejections = append(rejections, semanticRejection(spelling.position))
		}
	}
	return rejections
}

func (draft *declarationCandidate) validateEarlySelector(command CommandID, selector ValueID, destinations map[ValueID]destinationRecord) []rejection {
	selectorRecord, ok := destinations[selector]
	if !ok {
		return []rejection{{
			Kind: rejectionStructuralInvalidity,
		}}
	}
	position := selectorRecord.meta.position
	var rejections []rejection
	if selectorRecord.meta.profile.present() {
		rejections = append(rejections, semanticRejection(position))
	}

	staticDefaults := 0
	for _, producer := range selectorRecord.Producers() {
		if producer.profile.present() || producer.kind != declaredProducerKindDefault {
			rejections = append(rejections, semanticRejection(position))
			continue
		}
		staticDefaults++
	}
	if staticDefaults > 1 {
		rejections = append(rejections, semanticRejection(position))
	}

	var selectorFlags []flagRecord
	for _, flag := range draft.input.Flags {
		if flag.destination == selector {
			selectorFlags = append(selectorFlags, flag)
		}
	}
	switch len(selectorFlags) {
	case 0:
	case 1:
		rejections = append(rejections, draft.validateEarlySelectorFlag(command, selectorFlags[0])...)
	default:
		rejections = append(rejections, semanticRejection(position))
	}

	for _, term := range draft.input.ArgumentTerms {
		destination, present := term.Destination()
		if present && destination == selector {
			rejections = append(rejections, semanticRejection(term.meta.position))
		}
	}
	for _, input := range draft.input.EnvironmentInputs {
		if input.destination != selector {
			continue
		}
		if input.meta.profile.present() || input.contribution != environmentContributionScalar || input.prefix ||
			input.delimiter != "" {
			rejections = append(rejections, semanticRejection(input.meta.position))
		}
	}
	return rejections
}

func (draft *declarationCandidate) validateGrammarProfileSelector(command CommandID, selector ValueID, profiles []grammarProfileRecord, hasDefault bool, destinations map[ValueID]destinationRecord, flags map[FlagID]flagRecord, forms map[ArgumentFormID]argumentFormRecord) []rejection {
	selectorRecord, ok := destinations[selector]
	if !ok {
		return []rejection{{
			Kind: rejectionStructuralInvalidity,
		}}
	}
	position := selectorRecord.meta.position
	var rejections []rejection
	definition, ok := selectorRecord.Definition()
	language, languageOK := definition.Language()
	if !ok || definition.Kind() != destinationKindScalar || !languageOK || language.Kind() != valueLanguageKindDomain {
		rejections = append(rejections, semanticRejection(position))
	}

	var selectorDomain *valueDomainRecord
	for index := range draft.input.ValueDomains {
		domain := &draft.input.ValueDomains[index]
		for _, use := range domain.Uses() {
			if use.destination == selector && use.kind == ValueDomainUseDestination {
				if selectorDomain != nil {
					rejections = append(rejections, semanticRejection(position))
				} else {
					selectorDomain = domain
				}
			}
		}
	}
	if selectorDomain == nil {
		rejections = append(rejections, semanticRejection(position))
		return rejections
	}

	entryByID := make(map[ValueDomainEntryID]domainEntryRecord, len(draft.input.DomainEntries))
	for _, entry := range draft.input.DomainEntries {
		entryByID[entry.meta.id] = entry
	}
	explicitCases := make([]grammarProfileRecord, 0, len(profiles))
	explicitValues := make([]any, 0, len(profiles))
	for _, profile := range profiles {
		if profile.defaultCase {
			continue
		}
		value, ok := profile.Value()
		if !ok {
			continue
		}
		explicitCases = append(explicitCases, profile)
		explicitValues = append(explicitValues, value)
	}
	canonicalValues := make([]any, 0)
	effectiveValues := make([]any, 0)
	for _, id := range selectorDomain.Entries() {
		entry, ok := entryByID[id]
		if !ok || entry.kind != domainEntryKindCanonical {
			continue
		}
		value, ok := entry.Value()
		if !ok {
			continue
		}
		canonicalValues = append(canonicalValues, value)
		state, err := evaluateLifecycle(draft.application.release, entry.lifecycle, "")
		if err != nil {
			rejections = append(rejections, semanticRejection(entry.meta.position))
			continue
		}
		if state == LifecycleStateCurrent || state == LifecycleStateDeprecated {
			effectiveValues = append(effectiveValues, value)
		}
	}
	for index, value := range explicitValues {
		found := false
		for _, canonical := range canonicalValues {
			if explicitCases[index].ValuesEqual(canonical, value) {
				found = true
				break
			}
		}
		if !found {
			rejections = append(rejections, semanticRejection(explicitCases[index].meta.position))
		}
	}
	if !hasDefault {
		for _, value := range effectiveValues {
			covered := false
			for index, explicit := range explicitValues {
				if explicitCases[index].ValuesEqual(explicit, value) {
					covered = true
					break
				}
			}
			if !covered {
				rejections = append(rejections, semanticRejection(position))
				break
			}
		}
	}

	rejections = append(rejections, draft.validateEarlySelector(command, selector, destinations)...)

	_ = flags
	_ = forms
	return rejections
}

// validateGrammarStructure validates static grammar shape. Lexical collision
// ownership is lifecycle-aware and lives in ValidateLifecycleLexicalCollisions.
func validateGrammarStructure(model grammarModel) []rejection {
	if !model.Valid() {
		return []rejection{{
			Kind: rejectionStructuralInvalidity,
		}}
	}
	commandRejections := validateCommandNamespaceStructure(model)
	flagRejections := validateFlagNamespaceStructure(model)
	argumentRejections := validateArgumentForms(model)
	rejections := make([]rejection, 0, len(commandRejections)+len(flagRejections)+len(argumentRejections))
	rejections = append(rejections, commandRejections...)
	rejections = append(rejections, flagRejections...)
	rejections = append(rejections, argumentRejections...)
	return normalizeRejections(rejections)
}

func validateCommandNamespaceStructure(model grammarModel) []rejection {
	var rejections []rejection
	commands := map[CommandID]struct{}{model.Root(): {}}
	for _, entry := range model.AllCommandEntries() {
		commands[entry.parent] = struct{}{}
		commands[entry.target] = struct{}{}
	}
	for _, action := range model.AllActionCommands() {
		commands[action.parent] = struct{}{}
	}
	for command := range commands {
		for _, profile := range grammarBranches(model, command) {
			dialect, ok := grammarBranchDialect(model, command, profile)
			if !ok {
				rejections = append(rejections, rejection{
					Kind: rejectionStructuralInvalidity,
				})
				continue
			}
			mode := dialect.CommandNormalization()
			if !mode.valid() {
				rejections = append(rejections, rejection{
					Kind: rejectionInvalidNormalization,
				})
				continue
			}
			for _, entry := range model.CommandEntries(command) {
				if !grammarBranchContains(profile, entry.profile) {
					continue
				}
				name, valid := normalizeName(mode, entry.name)
				if !valid || name == "" {
					rejections = append(rejections, rejection{
						Kind:  rejectionInvalidNormalization,
						Order: uint64(entry.position),
					})
				}
			}
			for _, action := range model.ActionCommands(command) {
				name, valid := normalizeName(mode, action.name)
				if !valid || name == "" {
					rejections = append(rejections, rejection{
						Kind:  rejectionInvalidNormalization,
						Order: uint64(action.position),
					})
				}
			}
		}
	}
	return rejections
}

func validateFlagNamespaceStructure(model grammarModel) []rejection {
	var rejections []rejection
	commands := map[CommandID]struct{}{}
	for _, flag := range model.AllFlags() {
		commands[flag.command] = struct{}{}
	}
	for _, action := range model.AllActionFlags() {
		commands[action.parent] = struct{}{}
	}
	for command := range commands {
		for _, profile := range grammarBranches(model, command) {
			dialect, ok := grammarBranchDialect(model, command, profile)
			if !ok {
				rejections = append(rejections, rejection{
					Kind: rejectionStructuralInvalidity,
				})
				continue
			}
			mode := dialect.FlagNormalization()
			if !mode.valid() {
				rejections = append(rejections, rejection{
					Kind: rejectionInvalidNormalization,
				})
				continue
			}
			for _, flag := range model.Flags(command) {
				if !grammarBranchContains(profile, flag.profile) {
					continue
				}
				if terminator, present := flag.Terminator(); present {
					if terminator == "" || strings.ContainsRune(terminator, '\x00') ||
						containsExact(dialect.EndOfOptions(), terminator) {
						rejections = append(rejections, rejection{
							Kind:  rejectionStructuralInvalidity,
							Order: uint64(flag.position),
						})
					}
					for _, spelling := range flag.Spellings() {
						if spelling.text == terminator {
							rejectionValue := rejection{
								Kind:    rejectionStructuralInvalidity,
								Order:   uint64(flag.position),
								Related: uint64(spelling.position),
							}
							rejections = append(rejections, rejectionValue)
						}
					}
				}
				for _, spelling := range flag.Spellings() {
					name, valid := normalizeName(mode, spelling.text)
					if !valid || name == "" {
						rejections = append(rejections, rejection{
							Kind:  rejectionInvalidNormalization,
							Order: uint64(spelling.position),
						})
					}
				}
			}
			for _, action := range model.ActionFlags(command) {
				name, valid := normalizeName(mode, action.text)
				if !valid || name == "" {
					rejections = append(rejections, rejection{
						Kind:  rejectionInvalidNormalization,
						Order: uint64(action.position),
					})
				}
			}
		}
	}
	return rejections
}

func validateArgumentForms(model grammarModel) []rejection {
	var rejections []rejection
	seenRejections := make(map[rejection]struct{})
	appendRejection := func(rejection rejection) {
		if _, duplicate := seenRejections[rejection]; duplicate {
			return
		}
		seenRejections[rejection] = struct{}{}
		rejections = append(rejections, rejection)
	}
	commands := make(map[CommandID]struct{})
	for _, form := range model.AllArgumentForms() {
		commands[form.command] = struct{}{}
	}
	for command := range commands {
		for _, profile := range grammarBranches(model, command) {
			dialect, dialectOK := grammarBranchDialect(model, command, profile)
			if !dialectOK {
				appendRejection(rejection{
					Kind: rejectionStructuralInvalidity,
				})
			}
			for _, form := range model.ArgumentForms(command) {
				if !grammarBranchContains(profile, form.profile) {
					continue
				}
				seenTerms := make(map[ArgumentTermID]struct{})
				terms := form.Terms()
				endOfOptionsSeen := false
				for index, term := range terms {
					if _, exists := seenTerms[term.id]; exists {
						appendRejection(rejection{
							Kind:  rejectionStructuralInvalidity,
							Order: uint64(term.position),
						})
					}
					seenTerms[term.id] = struct{}{}
					if term.allocation != semanticArgumentAllocationUnspecified {
						maximum, bounded := term.count.Maximum()
						if bounded && maximum == term.count.Minimum() {
							appendRejection(rejection{
								Kind:  rejectionStructuralInvalidity,
								Order: uint64(term.position),
							})
						}
					}
					if term.endOfOptions {
						if endOfOptionsSeen || term.kind != ArgumentTermLiteral ||
							(dialectOK && containsExact(dialect.EndOfOptions(), term.literal)) {
							appendRejection(rejection{
								Kind:  rejectionStructuralInvalidity,
								Order: uint64(term.position),
							})
						}
						endOfOptionsSeen = true
					}
					switch term.kind {
					case ArgumentTermRemainder, ArgumentTermPassThrough:
						if index != len(terms)-1 {
							appendRejection(rejection{
								Kind:  rejectionStructuralInvalidity,
								Order: uint64(term.position),
							})
						}
					case ArgumentTermValue, ArgumentTermLiteral, ArgumentTermTokens:
					default:
						appendRejection(rejection{
							Kind:  rejectionStructuralInvalidity,
							Order: uint64(term.position),
						})
					}
				}
			}
		}
	}
	return rejections
}

func grammarBranches(model grammarModel, command CommandID) []GrammarProfileID {
	profiles := model.Profiles(command)
	if len(profiles) == 0 {
		return []GrammarProfileID{0}
	}
	return profiles
}

func grammarBranchDialect(model grammarModel, command CommandID, profile GrammarProfileID) (dialect, bool) {
	if profile.present() {
		return model.ProfileDialect(profile)
	}
	return model.Dialect(command)
}

func grammarBranchContains(branch, declaration GrammarProfileID) bool {
	return !declaration.present() || declaration == branch
}

func containsExact(values []string, target string) bool {
	return slices.Contains(values, target)
}

// normalizeName applies the locale-independent recognition normalization used
// before lexical collision checks.
func normalizeName(mode NameNormalization, value string) (string, bool) {
	switch mode {
	case NameNormalizationExact:
		return value, true
	case NameNormalizationASCIICaseFold:
		firstUpper := 0
		// Byte subtraction is uint8: only ASCII A-Z falls inside this range.
		for firstUpper < len(value) && value[firstUpper]-'A' > 'Z'-'A' {
			firstUpper++
		}
		if firstUpper == len(value) {
			return value, true
		}
		bytes := []byte(value)
		for i := firstUpper; i < len(bytes); i++ {
			if bytes[i]-'A' <= 'Z'-'A' {
				bytes[i] += 'a' - 'A'
			}
		}
		return string(bytes), true
	default:
		return "", false
	}
}

// lifecycleInterval is one normalized half-open active lexical interval. Rank
// values are supplied by the owning release-scheme adapter; zero HasStart or
// HasEnd means unbounded in that direction.
type lifecycleInterval struct {
	Start    int64
	End      int64
	HasStart bool
	HasEnd   bool
}

func (interval lifecycleInterval) Valid() bool {
	return !interval.HasStart || !interval.HasEnd || interval.Start < interval.End
}

func lifecycleIntervalsOverlap(left, right lifecycleInterval) bool {
	if !left.Valid() || !right.Valid() {
		return false
	}
	if left.HasEnd && right.HasStart && left.End <= right.Start {
		return false
	}
	if right.HasEnd && left.HasStart && right.End <= left.Start {
		return false
	}
	return true
}

// lexicalEntry describes one already-authoritative concrete namespace/profile
// member for collision validation. Availability is intentionally absent: it is
// not a namespace partition.
type lexicalEntry struct {
	Name      string
	Mode      NameNormalization
	Namespace uint64
	Profile   uint64
	Lifecycle lifecycleInterval
	Order     uint64
	Subject   uint64
}

// validateLexicalCollisions enforces normalization-before-collision and allows
// coexistence only in distinct concrete profile partitions or disjoint active
// lifecycle intervals.
func validateLexicalCollisions(entries []lexicalEntry) []rejection {
	type normalizedEntry struct {
		lexicalEntry

		normalized string
	}
	prepared := make([]normalizedEntry, 0, len(entries))
	var rejections []rejection
	for _, entry := range entries {
		name, ok := normalizeName(entry.Mode, entry.Name)
		if !ok {
			rejectionValue2 := rejection{
				Kind:    rejectionInvalidNormalization,
				Order:   entry.Order,
				Subject: entry.Subject,
			}
			rejections = append(rejections, rejectionValue2)
			continue
		}
		if !entry.Lifecycle.Valid() {
			rejectionValue3 := rejection{
				Kind:    rejectionInvalidLifecycleInterval,
				Order:   entry.Order,
				Subject: entry.Subject,
			}
			rejections = append(rejections, rejectionValue3)
			continue
		}
		prepared = append(prepared, normalizedEntry{
			lexicalEntry: entry,
			normalized:   name,
		})
	}
	for i := range prepared {
		for j := i + 1; j < len(prepared); j++ {
			left, right := prepared[i], prepared[j]
			if left.Namespace != right.Namespace || left.Profile != right.Profile ||
				left.normalized != right.normalized {
				continue
			}
			if !lifecycleIntervalsOverlap(left.Lifecycle, right.Lifecycle) {
				continue
			}
			rejectionValue := rejection{
				Kind:    rejectionNormalizedCollision,
				Order:   right.Order,
				Subject: right.Subject,
				Related: left.Subject,
			}
			rejections = append(rejections, rejectionValue)
		}
	}
	return normalizeRejections(rejections)
}
