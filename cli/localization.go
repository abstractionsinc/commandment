package cli

import (
	"errors"
	"maps"
	"strings"
	"sync"
)

// frameworkPresentationState is the operation-local immutable-input context
// used to resolve Commandment-owned presentation after an effective locale is
// known. The compiled localization snapshot is detached before the operation
// begins; FinalizeLocale may establish the operation locale exactly once.
type frameworkPresentationState struct {
	mu           sync.RWMutex
	localization localizationSnapshot
	locale       string
	finalized    bool
}

func prepareFrameworkPresentationState(localization localizationSnapshot) *frameworkPresentationState {
	state := &frameworkPresentationState{
		localization: localization,
	}
	return state
}

// FinalizeLocale records the already-validated effective locale. Repeating the
// same value is harmless; a conflicting second value is rejected.
func (state *frameworkPresentationState) FinalizeLocale(locale string) bool {
	if state == nil {
		return false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.finalized {
		return state.locale == locale
	}
	state.locale = locale
	state.finalized = true
	return true
}

// Locale returns the finalized effective locale and whether it has been
// established for this operation.
func (state *frameworkPresentationState) Locale() (string, bool) {
	if state == nil {
		return "", false
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.locale, state.finalized
}

// Resolve resolves one finite Commandment-owned source phrase through the
// framework localization owner. Before locale finalization, source text is
// preserved rather than guessing an operation locale.
func (state *frameworkPresentationState) Resolve(source string) string {
	if state == nil || source == "" {
		return source
	}
	state.mu.RLock()
	locale, finalized := state.locale, state.finalized
	localization := state.localization
	state.mu.RUnlock()
	if !finalized {
		return source
	}
	resolved, _, err := localization.Resolve(locale, source)
	if err != nil {
		return source
	}
	return resolved
}

// FrameworkPresentation returns the operation-local framework presentation
// context. The returned state owns only compiled immutable localization input
// plus this operation's once-finalized effective locale.
func (operation *invocation) FrameworkPresentation() *frameworkPresentationState {
	if operation == nil {
		return nil
	}
	return operation.presentation
}

// FinalizeFrameworkPresentationLocale binds the already-selected effective
// locale to this operation's delayed Diagnostic presentation context.
func (operation *invocation) FinalizeFrameworkPresentationLocale(locale string) bool {
	if operation == nil {
		return false
	}
	return operation.presentation.FinalizeLocale(locale)
}

var (
	errInvalidLocale       = errors.New("semantic: invalid locale")
	errInvalidLocalization = errors.New("semantic: invalid localization")
)

const (
	posixLocale                = "POSIX"
	localeLanguageMinLength    = 2
	localeLanguageMaxLength    = 8
	localeExtlangMaxCount      = 3
	localeExtlangLength        = 3
	localeScriptLength         = 4
	localeAlphaRegionLength    = 2
	localeNumericRegionLength  = 3
	localeExtensionMinLength   = 2
	localeExtensionMaxLength   = 8
	localePrivateMinLength     = 1
	localePrivateMaxLength     = 8
	localeVariantMinLength     = 5
	localeVariantMaxLength     = 8
	localeNumericVariantLength = 4
	minimumDedupeLength        = 2
)

type translationBundleInput struct {
	Locale       string
	Translations map[string]string
}

type localizationSnapshot struct {
	enabled bool
	source  string
	bundles map[string]map[string]string
}

func cloneLocalizationSnapshot(localization localizationSnapshot) localizationSnapshot {
	clone := localizationSnapshot{
		enabled: localization.enabled,
		source:  localization.source,
	}
	if localization.bundles == nil {
		return clone
	}
	clone.bundles = make(map[string]map[string]string, len(localization.bundles))
	for locale, translations := range localization.bundles {
		clone.bundles[locale] = maps.Clone(translations)
	}
	return clone
}

func freezeLocalizationSnapshot(source string, bundles []translationBundleInput) (localizationSnapshot, error) {
	if source == "" && len(bundles) == 0 {
		return localizationSnapshot{}, nil
	}
	if source == "" {
		return localizationSnapshot{}, errInvalidLocalization
	}
	canonicalSource, err := canonicalStoredLocale(source)
	if err != nil {
		return localizationSnapshot{}, errInvalidLocalization
	}
	result := localizationSnapshot{
		enabled: true,
		source:  canonicalSource,
		bundles: make(map[string]map[string]string),
	}
	for _, bundle := range bundles {
		locale, err := canonicalStoredLocale(bundle.Locale)
		if err != nil || locale == canonicalSource {
			return localizationSnapshot{}, errInvalidLocalization
		}
		target := result.bundles[locale]
		if target == nil {
			target = make(map[string]string)
			result.bundles[locale] = target
		}
		for sourceText, translated := range bundle.Translations {
			if sourceText == "" {
				return localizationSnapshot{}, errInvalidLocalization
			}
			if _, duplicate := target[sourceText]; duplicate {
				return localizationSnapshot{}, errInvalidLocalization
			}
			target[sourceText] = translated
		}
	}
	return result, nil
}

func (localization localizationSnapshot) Enabled() bool { return localization.enabled }

func (localization localizationSnapshot) SourceLocale() string { return localization.source }

func (localization localizationSnapshot) Bundles() []translationBundleInput {
	if !localization.enabled {
		return nil
	}
	locales := make([]string, 0, len(localization.bundles))
	for locale := range localization.bundles {
		locales = append(locales, locale)
	}
	sortStrings(locales)
	result := make([]translationBundleInput, 0, len(locales))
	for _, locale := range locales {
		translations := make(map[string]string, len(localization.bundles[locale]))
		maps.Copy(translations, localization.bundles[locale])
		result = append(result, translationBundleInput{
			Locale:       locale,
			Translations: translations,
		})
	}
	return result
}

// Resolve performs exact-source translation lookup over structural locale
// parents. The bool reports translation presence, including a present empty
// target. Source fallback returns translated=false.
func (localization localizationSnapshot) Resolve(effectiveLocale, sourceText string) (string, bool, error) {
	if sourceText == "" {
		return "", false, nil
	}
	if effectiveLocale == "C" || effectiveLocale == posixLocale {
		return sourceText, false, nil
	}
	if !localization.enabled {
		return sourceText, false, nil
	}
	selected, err := canonicalSelectedLocale(effectiveLocale)
	if err != nil {
		return "", false, err
	}
	if selected == "C" || selected == "" {
		return sourceText, false, nil
	}
	lookup, err := localeLookupBase(selected)
	if err != nil {
		return "", false, err
	}
	for _, locale := range localeParents(lookup) {
		if locale == localization.source {
			return sourceText, false, nil
		}
		if bundle := localization.bundles[locale]; bundle != nil {
			if target, ok := bundle[sourceText]; ok {
				return target, true, nil
			}
		}
	}
	return sourceText, false, nil
}

// selectEffectiveLocale applies host -> resolved selector -> source -> empty
// precedence. Invalid nonempty higher-precedence input is an error and never
// falls through.
func selectEffectiveLocale(host, selector, source string) (string, error) {
	for _, candidate := range []string{host, selector} {
		if candidate == "" {
			continue
		}
		return canonicalSelectedLocale(candidate)
	}
	if source != "" {
		return canonicalStoredLocale(source)
	}
	return "", nil
}

var errInvalidResolvedLocaleSelector = errors.New("semantic: invalid resolved locale selector")

func resolvedLocaleSelector(model *effectiveModel, grammar grammarResult, values valueAnalysis) (Locale, error) {
	var selector Locale
	var found bool
	for _, command := range grammar.Selection.Path {
		for _, flag := range model.FlagsForCommand(command) {
			if !flag.localeSelector {
				continue
			}
			resolution, ok := values.Destinations[flag.destination]
			if !ok || resolution.State == destinationStateAbsent || resolution.State == destinationStateInactive {
				continue
			}
			if resolution.State != destinationStateResolved {
				return "", errInvalidResolvedLocaleSelector
			}
			var current Locale
			switch value := resolution.Value.(type) {
			case Locale:
				current = value
			case string:
				current = Locale(value)
			default:
				return "", errInvalidResolvedLocaleSelector
			}
			if found && current != selector {
				return "", errInvalidResolvedLocaleSelector
			}
			selector, found = current, true
		}
	}
	return selector, nil
}

type parsedLocale struct {
	language   string
	extlangs   []string
	script     string
	region     string
	variants   []string
	extensions []string
	private    []string
}

func canonicalStoredLocale(value string) (string, error) {
	if value == "" || value == "C" || value == posixLocale || value != strings.TrimSpace(value) ||
		strings.ContainsAny(value, "_.") {
		return "", errInvalidLocale
	}
	tag, ok := parseLocale(value)
	if !ok || len(tag.extensions) != 0 || len(tag.private) != 0 {
		return "", errInvalidLocale
	}
	return tag.canonical(false), nil
}

func canonicalSelectedLocale(value string) (string, error) {
	if value == "C" || value == posixLocale {
		return "C", nil
	}
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "_.") {
		return "", errInvalidLocale
	}
	tag, ok := parseLocale(value)
	if !ok {
		return "", errInvalidLocale
	}
	return tag.canonical(true), nil
}

func parseLocale(value string) (parsedLocale, bool) {
	var result parsedLocale
	parts := strings.Split(value, "-")
	if len(parts) == 0 || !alphaBetween(parts[0], localeLanguageMinLength, localeLanguageMaxLength) || strings.EqualFold(parts[0], "x") {
		return result, false
	}
	result.language = strings.ToLower(parts[0])
	i := 1
	if len(parts[0]) <= localeExtlangLength {
		for len(result.extlangs) < localeExtlangMaxCount && i < len(parts) && alphaBetween(parts[i], localeExtlangLength, localeExtlangLength) {
			result.extlangs = append(result.extlangs, strings.ToLower(parts[i]))
			i++
		}
	}
	if i < len(parts) && alphaBetween(parts[i], localeScriptLength, localeScriptLength) {
		part := strings.ToLower(parts[i])
		result.script = strings.ToUpper(part[:1]) + part[1:]
		i++
	}
	if i < len(parts) && (alphaBetween(parts[i], localeAlphaRegionLength, localeAlphaRegionLength) || digitBetween(parts[i], localeNumericRegionLength, localeNumericRegionLength)) {
		if alphaBetween(parts[i], localeAlphaRegionLength, localeAlphaRegionLength) {
			result.region = strings.ToUpper(parts[i])
		} else {
			result.region = parts[i]
		}
		i++
	}
	for i < len(parts) && validVariant(parts[i]) {
		result.variants = append(result.variants, strings.ToLower(parts[i]))
		i++
	}
	seenExtension := map[byte]struct{}{}
	for i < len(parts) && len(parts[i]) == 1 && asciiAlnum(parts[i][0]) && !strings.EqualFold(parts[i], "x") {
		singleton := strings.ToLower(parts[i])[0]
		if _, duplicate := seenExtension[singleton]; duplicate {
			return parsedLocale{}, false
		}
		seenExtension[singleton] = struct{}{}
		start := i
		i++
		count := 0
		for i < len(parts) && alnumBetween(parts[i], localeExtensionMinLength, localeExtensionMaxLength) {
			count++
			i++
		}
		if count == 0 {
			return parsedLocale{}, false
		}
		extension := strings.ToLower(strings.Join(parts[start:i], "-"))
		result.extensions = append(result.extensions, extension)
	}
	if i < len(parts) && strings.EqualFold(parts[i], "x") {
		i++
		if i == len(parts) {
			return parsedLocale{}, false
		}
		for i < len(parts) {
			if !alnumBetween(parts[i], localePrivateMinLength, localePrivateMaxLength) {
				return parsedLocale{}, false
			}
			result.private = append(result.private, strings.ToLower(parts[i]))
			i++
		}
	}
	if i != len(parts) {
		return parsedLocale{}, false
	}
	return result, true
}

func (tag parsedLocale) canonical(includeLookupIrrelevant bool) string {
	parts := []string{tag.language}
	parts = append(parts, tag.extlangs...)
	if tag.script != "" {
		parts = append(parts, tag.script)
	}
	if tag.region != "" {
		parts = append(parts, tag.region)
	}
	parts = append(parts, tag.variants...)
	if includeLookupIrrelevant {
		parts = append(parts, tag.extensions...)
		if len(tag.private) != 0 {
			parts = append(parts, "x")
			parts = append(parts, tag.private...)
		}
	}
	return strings.Join(parts, "-")
}

func localeLookupBase(value string) (string, error) {
	tag, ok := parseLocale(value)
	if !ok {
		return "", errInvalidLocale
	}
	return tag.canonical(false), nil
}

func localeParents(value string) []string {
	tag, ok := parseLocale(value)
	if !ok {
		return nil
	}
	var result []string
	appendCurrent := func() { result = append(result, tag.canonical(false)) }
	appendCurrent()
	for len(tag.variants) > 0 {
		tag.variants = tag.variants[:len(tag.variants)-1]
		appendCurrent()
	}
	if tag.region != "" {
		tag.region = ""
		appendCurrent()
	}
	if tag.script != "" {
		tag.script = ""
		appendCurrent()
	}
	// extlangs are part of the language identity for this bounded parser; they
	// are not stripped independently because doing so would invent fallback not
	// established by this implementation.
	return dedupeStrings(result)
}

func validVariant(value string) bool {
	if alnumBetween(value, localeVariantMinLength, localeVariantMaxLength) {
		return true
	}
	return len(value) == localeNumericVariantLength && value[0] >= '0' && value[0] <= '9' && alnumBetween(value, localeNumericVariantLength, localeNumericVariantLength)
}

func alphaBetween(value string, minimum, maximum int) bool {
	if len(value) < minimum || len(value) > maximum {
		return false
	}
	for i := range len(value) {
		if !asciiAlpha(value[i]) {
			return false
		}
	}
	return true
}

func digitBetween(value string, minimum, maximum int) bool {
	if len(value) < minimum || len(value) > maximum {
		return false
	}
	for i := range len(value) {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func alnumBetween(value string, minimum, maximum int) bool {
	if len(value) < minimum || len(value) > maximum {
		return false
	}
	for i := range len(value) {
		if !asciiAlnum(value[i]) {
			return false
		}
	}
	return true
}

func asciiAlpha(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

func asciiAlnum(value byte) bool { return asciiAlpha(value) || (value >= '0' && value <= '9') }

func dedupeStrings(values []string) []string {
	if len(values) < minimumDedupeLength {
		return values
	}
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

const (
	frameworkHelpLabelTokenDefault    = "default"
	frameworkHelpLabelTokenRequired   = "required"
	frameworkHelpLabelTokenDeprecated = "deprecated"
	frameworkHelpLabelTokenAlias      = "alias"
	frameworkHelpLabelTokenShortcut   = "shortcut"
)

func defaultHelpLabels(localization localizationSnapshot, locale Locale) HelpLabels {
	labels := frameworkHelpSourceLabels()
	translate := func(source string) string {
		translated, _, err := localization.Resolve(string(locale), source)
		if err != nil {
			return source
		}
		return translated
	}

	labels.Usage = translate(labels.Usage)
	labels.Commands = translate(labels.Commands)
	labels.Arguments = translate(labels.Arguments)
	labels.Flags = translate(labels.Flags)
	labels.GlobalFlags = translate(labels.GlobalFlags)
	labels.Aliases = translate(labels.Aliases)
	labels.Environment = translate(labels.Environment)
	labels.Default = translate(labels.Default)
	labels.Required = translate(labels.Required)
	labels.Optional = translate(labels.Optional)
	labels.Deprecated = translate(labels.Deprecated)
	labels.Examples = translate(labels.Examples)
	labels.Alias = translate(labels.Alias)
	labels.Shortcut = translate(labels.Shortcut)
	labels.Values = translate(labels.Values)
	labels.Keys = translate(labels.Keys)
	labels.Syntax = translate(labels.Syntax)
	labels.AliasFor = translate(labels.AliasFor)
	labels.Use = translate(labels.Use)
	labels.ImplicitValue = translate(labels.ImplicitValue)
	labels.FixedValue = translate(labels.FixedValue)

	return labels
}

func helpLabelsWithFrameworkFallbacks(labels HelpLabels) HelpLabels {
	fallback := frameworkHelpSourceLabels()
	if labels.Usage == "" {
		labels.Usage = fallback.Usage
	}
	if labels.Commands == "" {
		labels.Commands = fallback.Commands
	}
	if labels.Arguments == "" {
		labels.Arguments = fallback.Arguments
	}
	if labels.Flags == "" {
		labels.Flags = fallback.Flags
	}
	if labels.GlobalFlags == "" {
		labels.GlobalFlags = fallback.GlobalFlags
	}
	if labels.Aliases == "" {
		labels.Aliases = fallback.Aliases
	}
	if labels.Environment == "" {
		labels.Environment = fallback.Environment
	}
	if labels.Default == "" {
		labels.Default = fallback.Default
	}
	if labels.Required == "" {
		labels.Required = fallback.Required
	}
	if labels.Optional == "" {
		labels.Optional = fallback.Optional
	}
	if labels.Deprecated == "" {
		labels.Deprecated = fallback.Deprecated
	}
	if labels.Examples == "" {
		labels.Examples = fallback.Examples
	}
	if labels.Alias == "" {
		labels.Alias = fallback.Alias
	}
	if labels.Shortcut == "" {
		labels.Shortcut = fallback.Shortcut
	}
	if labels.Values == "" {
		labels.Values = fallback.Values
	}
	if labels.Keys == "" {
		labels.Keys = fallback.Keys
	}
	if labels.Syntax == "" {
		labels.Syntax = fallback.Syntax
	}
	if labels.AliasFor == "" {
		labels.AliasFor = fallback.AliasFor
	}
	if labels.Use == "" {
		labels.Use = fallback.Use
	}
	if labels.ImplicitValue == "" {
		labels.ImplicitValue = fallback.ImplicitValue
	}
	if labels.FixedValue == "" {
		labels.FixedValue = fallback.FixedValue
	}
	return labels
}

func frameworkHelpSourceLabels() HelpLabels {
	labels := HelpLabels{
		Usage:         "Usage",
		Commands:      "Commands",
		Arguments:     "Arguments",
		Flags:         "Flags",
		GlobalFlags:   "Global flags",
		Aliases:       "Aliases",
		Environment:   "Environment",
		Default:       frameworkHelpLabelTokenDefault,
		Required:      frameworkHelpLabelTokenRequired,
		Optional:      "optional",
		Deprecated:    frameworkHelpLabelTokenDeprecated,
		Examples:      "Examples",
		Alias:         frameworkHelpLabelTokenAlias,
		Shortcut:      frameworkHelpLabelTokenShortcut,
		Values:        "Values",
		Keys:          "Keys",
		Syntax:        "Syntax",
		AliasFor:      "alias for",
		Use:           "use",
		ImplicitValue: "Implicit value",
		FixedValue:    "Fixed value",
	}

	return labels
}

// Locale carries locale identity for localization configuration and operation presentation. Stored
// source and bundle locales use validated BCP 47 language identity; C and POSIX are
// source-presentation directives rather than stored bundle locales.
type Locale string

// Translations maps exact nonempty source strings to opaque final target strings for one locale.
// Source keys are not trimmed, case-folded, Unicode-normalized, or namespaced; map presence
// distinguishes a missing translation from an intentional empty translation.
type Translations map[string]string

// TranslationBundle contributes target translations for one locale. Disjoint fragments for the
// same canonical locale have no order-based precedence, while duplicate authority for the same
// exact source key is invalid even when target text agrees.
type TranslationBundle struct {
	Locale       Locale
	Translations Translations
}

// Localization declares the application source-first translation configuration. Its zero value is
// valid and disables configured translation processing; a nonempty DefaultLocale with no bundles
// is valid, while nonempty bundles require a valid nonempty default/source locale. Compilation
// snapshots retained bundle slices and translation maps, so later caller mutation does not change
// the compiled Application.
type Localization struct {
	DefaultLocale Locale
	Bundles       []TranslationBundle
}

// snapshotLocalization lowers the caller-owned public localization declaration
// into the one immutable semantic translation authority. Validation remains in
// the semantic owner so the facade cannot drift from locale or duplicate rules.
func snapshotLocalization(localization Localization) (localizationSnapshot, error) {
	bundles := make([]translationBundleInput, len(localization.Bundles))
	for i, bundle := range localization.Bundles {
		translations := make(map[string]string, len(bundle.Translations))
		maps.Copy(translations, bundle.Translations)
		bundles[i] = translationBundleInput{
			Locale:       string(bundle.Locale),
			Translations: translations,
		}
	}
	return freezeLocalizationSnapshot(string(localization.DefaultLocale), bundles)
}

const frameworkDefaultLocale Locale = "en"

// snapshotFrameworkLocalization snapshots caller-contributed target translations for
// Commandment-owned presentation. The framework source/default locale is owned by Commandment and
// intentionally has no public configuration field.
func snapshotFrameworkLocalization(bundles []TranslationBundle) (localizationSnapshot, error) {
	localization := Localization{
		DefaultLocale: frameworkDefaultLocale,
		Bundles:       bundles,
	}
	return snapshotLocalization(localization)
}

func publicLocalization(localization localizationSnapshot) Localization {
	if !localization.Enabled() {
		return Localization{}
	}
	bundles := localization.Bundles()
	result := Localization{
		DefaultLocale: Locale(localization.SourceLocale()),
		Bundles:       make([]TranslationBundle, len(bundles)),
	}
	for index, bundle := range bundles {
		translations := make(Translations, len(bundle.Translations))
		maps.Copy(translations, bundle.Translations)
		result.Bundles[index] = TranslationBundle{
			Locale:       Locale(bundle.Locale),
			Translations: translations,
		}
	}
	return result
}
