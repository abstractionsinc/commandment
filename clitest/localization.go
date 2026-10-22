package clitest

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/abstractionsinc/commandment/cli"
)

// PresentedDeclarationExpectation is a closed presentation expectation applicable to
// detached Reference and Inspection observations.
//
//nolint:iface // Distinct closed public expectation roles are part of the accepted clitest API.
type PresentedDeclarationExpectation interface {
	ReferenceExpectation
	InspectionExpectation
}

// HelpPresentationExpectation is a closed presentation expectation over an
// already-observed public HelpPage. Use RequireHelpPresentation to evaluate it
// without running a Commandment operation or rendering golden text.
type HelpPresentationExpectation interface{ helpPresentationExpectation() }

// LocalizationPolicy configures the whole-application localization audit.
// RequireCompleteLocales permits normal locale fallback while requiring every
// known application source to resolve without source-text fallback;
// RequireExactLocales requires a direct translation in each canonical target
// locale. RuntimeSources adds exact application-owned strings that static
// declarations cannot reveal. The rejection fields optionally flag
// unreferenced translations and shared static sources, with AllowedSharedSources
// serving as the explicit exception set.
type LocalizationPolicy struct {
	// RequireCompleteLocales lists target locales that must resolve every known
	// application source without falling back to source text; normal locale fallback
	// remains allowed.
	RequireCompleteLocales []cli.Locale
	// RequireExactLocales lists canonical target locales that must contain a direct
	// translation for every known application source.
	RequireExactLocales []cli.Locale

	// RuntimeSources adds exact application-owned source strings that are not
	// statically discoverable. Entries must be unique, nonempty, valid UTF-8, and
	// absent from the static source inventory.
	RuntimeSources []string

	// RejectUnreferencedTranslations reports translations whose source is absent
	// from the combined static and RuntimeSources inventory.
	RejectUnreferencedTranslations bool
	// RejectSharedSources reports static source strings used by more than one
	// presentation slot unless allowed by AllowedSharedSources.
	RejectSharedSources bool
	// AllowedSharedSources lists currently shared static source strings exempted
	// from RejectSharedSources. Naming a source that is not currently shared is
	// itself an audit issue.
	AllowedSharedSources []string
}

// LocalizationExactly returns a Schema expectation that checks the complete
// public static localization snapshot. It snapshots expected so later caller
// mutation cannot change expectation evaluation.
func LocalizationExactly(expected cli.Localization) SchemaExpectation {
	expectation := localizationSchemaExpectation{
		expected: cloneLocalization(expected),
	}
	return expectation
}

// RequireHelpPresentation checks already-observed public HelpPage presentation.
// It deliberately does not run Commandment or render golden text.
func RequireHelpPresentation(t testing.TB, page cli.HelpPage, expectations ...HelpPresentationExpectation) {
	t.Helper()
	outcomes := make([]evaluationResult, 0, len(expectations))
	for _, expectation := range expectations {
		if expectation == nil {
			outcomes = append(outcomes, disprovedResult(evaluationText("help presentation"), evaluationText("valid expectation"), evaluationText("nil expectation")))
			continue
		}
		evaluator, ok := expectation.(interface {
			evaluateHelp(page cli.HelpPage) evaluationResult
		})
		if !ok {
			panic(fmt.Sprintf("clitest: invalid HelpPresentationExpectation implementation %T", expectation))
		}
		outcomes = append(outcomes, evaluator.evaluateHelp(page))
	}
	reportEvaluationResults(t, outcomes)
}

// HelpDescription returns a HelpPresentationExpectation that checks the public
// HelpPage description.
func HelpDescription(expected cli.Description) HelpPresentationExpectation {
	expectation := helpDescriptionExpectation{
		expected: expected,
	}
	return expectation
}

// HelpExamplesExactly returns a HelpPresentationExpectation that checks the
// complete ordered public HelpPage examples.
func HelpExamplesExactly(expected ...cli.Example) HelpPresentationExpectation {
	expectation := helpExamplesExpectation{
		expected: slices.Clone(expected),
	}
	return expectation
}

// RequireLocalization audits app against policy using Commandment's public
// localization artifacts and reports every discovered issue through t. It marks
// itself as a testing helper and does not introduce a localization-specific
// application operation or alternate localization semantics.
func RequireLocalization(t testing.TB, app cli.Application, policy LocalizationPolicy) {
	t.Helper()
	issues := auditLocalization(app, policy)
	outcomes := make([]evaluationResult, 0, len(issues))
	for _, issue := range issues {
		outcomes = append(outcomes, disprovedResult(evaluationText("localization"), evaluationText("policy satisfied"), issue))
	}
	reportEvaluationResults(t, outcomes)
}

func (LocalizationCase) conformanceCase() {}

const localizationSourceRecordParts = 2

type localizationReferenceError struct {
	locale cli.Locale
	cause  error
}

type localizationSchemaExpectation struct{ expected cli.Localization }

type commandSourceDescriptionExpectation struct {
	locator  *commandLocatorState
	expected cli.Description
}

type commandPresentedDescriptionExpectation struct {
	locator  *commandLocatorState
	expected cli.Description
}

type commandSourceExamplesExpectation struct {
	locator  *commandLocatorState
	expected []cli.Example
}

type commandPresentedExamplesExpectation struct {
	locator  *commandLocatorState
	expected []cli.Example
}

type completionItemPresentationExpectation struct {
	locator     *completionItemLocatorState
	displayText string
	description string
}

type helpDescriptionExpectation struct{ expected cli.Description }

type helpExamplesExpectation struct{ expected []cli.Example }

type localizationSourceInventory struct {
	counts map[string]int
}

type effectiveLocaleExpectationValue struct{ locale cli.Locale }

func compareDescription(subject evaluationField, expected, actual cli.Description) evaluationResult {
	if actual.Short != expected.Short {
		return disprovedResult(qualifySubject(subject, evaluationText("short")), evaluationText(expected.Short), evaluationText(actual.Short))
	}
	if actual.Long != expected.Long {
		return disprovedResult(qualifySubject(subject, evaluationText("long")), evaluationText(expected.Long), evaluationText(actual.Long))
	}
	return provedResult()
}

func cloneLocalization(input cli.Localization) cli.Localization {
	output := cli.Localization{
		DefaultLocale: input.DefaultLocale,
		Bundles:       make([]cli.TranslationBundle, len(input.Bundles)),
	}
	for i, bundle := range input.Bundles {
		translations := make(cli.Translations, len(bundle.Translations))
		maps.Copy(translations, bundle.Translations)
		output.Bundles[i] = cli.TranslationBundle{
			Locale:       bundle.Locale,
			Translations: translations,
		}
	}
	return output
}

func presentationField(display, description string) evaluationField {
	return qualifySubject(subjectNamed(evaluationText("display"), display), subjectNamed(evaluationText("description"), description))
}

func compareLocalization(expected, actual cli.Localization) (bool, evaluationField) {
	if actual.DefaultLocale != expected.DefaultLocale {
		return false, subjectNamed(evaluationText("source locale"), string(actual.DefaultLocale))
	}
	if len(actual.Bundles) != len(expected.Bundles) {
		return false, evaluationText(fmt.Sprintf("%d bundles", len(actual.Bundles)))
	}
	for i := range expected.Bundles {
		want, got := expected.Bundles[i], actual.Bundles[i]
		bundle := subjectNamed(evaluationText(fmt.Sprintf("bundle[%d] locale", i)), string(got.Locale))
		if got.Locale != want.Locale {
			return false, bundle
		}
		if len(got.Translations) != len(want.Translations) {
			return false, qualifySubject(bundle, evaluationText(fmt.Sprintf("has %d translations", len(got.Translations))))
		}
		keys := make([]string, 0, len(want.Translations))
		for source := range want.Translations {
			keys = append(keys, source)
		}
		slices.Sort(keys)
		for _, source := range keys {
			wantTarget := want.Translations[source]
			gotTarget, ok := got.Translations[source]
			if !ok {
				return false, qualifySubject(bundle, subjectNamed(evaluationText("missing source"), source))
			}
			if gotTarget != wantTarget {
				sourceTarget := qualifySubject(subjectNamed(evaluationText("source"), source), subjectNamed(evaluationText("target"), gotTarget))
				return false, qualifySubject(bundle, sourceTarget)
			}
		}
	}
	return true, formatLocalization(actual)
}

func formatLocalization(localization cli.Localization) evaluationField {
	locale := subjectNamed(evaluationText("source locale"), string(localization.DefaultLocale))
	return qualifySubject(locale, evaluationText(fmt.Sprintf("with %d bundles", len(localization.Bundles))))
}

func auditLocalization(app cli.Application, policy LocalizationPolicy) []evaluationField {
	schema := app.Schema()
	inventory := collectLocalizationSources(schema)
	knownSources := make(map[string]struct{}, len(inventory.counts)+len(policy.RuntimeSources))
	for source := range inventory.counts {
		knownSources[source] = struct{}{}
	}
	seenRuntime := make(map[string]struct{}, len(policy.RuntimeSources))

	var issues []evaluationField
	for i, source := range policy.RuntimeSources {
		switch {
		case source == "":
			issues = append(issues, evaluationText(fmt.Sprintf("RuntimeSources[%d] is empty", i)))
		case !utf8.ValidString(source):
			issues = append(issues, evaluationText(fmt.Sprintf("RuntimeSources[%d] is not valid UTF-8", i)))
		case inventory.counts[source] != 0:
			issues = append(issues, qualifySubject(subjectNamed(evaluationText("runtime source"), source), evaluationText("is already statically discoverable")))
		default:
			if _, duplicate := seenRuntime[source]; duplicate {
				issues = append(issues, qualifySubject(subjectNamed(evaluationText("runtime source"), source), evaluationText("is duplicated")))
				continue
			}
			seenRuntime[source] = struct{}{}
			knownSources[source] = struct{}{}
		}
	}

	bundles := make(map[string]cli.Translations, len(schema.Localization.Bundles))
	for _, bundle := range schema.Localization.Bundles {
		bundles[string(bundle.Locale)] = bundle.Translations
	}
	sourceLocale := string(schema.Localization.DefaultLocale)

	for _, requested := range policy.RequireCompleteLocales {
		canonical, err := canonicalAuditLocale(app, requested)
		if err != nil {
			issues = append(issues, qualifySubject(subjectNamed(evaluationText("required complete locale"), string(requested)), evaluationText("is invalid")))
			continue
		}
		missing := missingCompleteSources(canonical, sourceLocale, bundles, knownSources)
		if len(missing) != 0 {
			issue := qualifySubject(
				subjectNamed(evaluationText("locale"), canonical),
				withDetail("is incomplete; missing translations for ", stringCollectionField{
					values: missing,
				}),
			)
			issues = append(issues, issue)
		}
	}

	for _, requested := range policy.RequireExactLocales {
		canonical, err := canonicalAuditLocale(app, requested)
		if err != nil {
			issues = append(issues, qualifySubject(subjectNamed(evaluationText("required exact locale"), string(requested)), evaluationText("is invalid")))
			continue
		}
		if canonical == sourceLocale && canonical != "" {
			issues = append(issues, qualifySubject(subjectNamed(evaluationText("source locale"), canonical), evaluationText("cannot be required as an exact target locale")))
			continue
		}
		translations := bundles[canonical]
		missing := missingDirectSources(translations, knownSources)
		if len(missing) != 0 {
			issue := qualifySubject(
				subjectNamed(evaluationText("locale"), canonical),
				withDetail("is not exact; missing direct translations for ", stringCollectionField{
					values: missing,
				}),
			)
			issues = append(issues, issue)
		}
	}

	if policy.RejectUnreferencedTranslations {
		var unreferenced []string
		for locale, translations := range bundles {
			for source := range translations {
				if _, ok := knownSources[source]; !ok {
					unreferenced = append(unreferenced, locale+"\x00"+source)
				}
			}
		}
		slices.Sort(unreferenced)
		for _, item := range unreferenced {
			parts := strings.SplitN(item, "\x00", localizationSourceRecordParts)
			issues = append(issues, qualifySubject(
				subjectNamed(evaluationText("locale"), parts[0]),
				subjectNamed(evaluationText("contains unreferenced translation"), parts[1]),
			))
		}
	}

	shared := make(map[string]struct{})
	for source, count := range inventory.counts {
		if count > 1 {
			shared[source] = struct{}{}
		}
	}
	allowed := make(map[string]struct{}, len(policy.AllowedSharedSources))
	for _, source := range policy.AllowedSharedSources {
		if _, ok := shared[source]; !ok {
			issues = append(issues, qualifySubject(subjectNamed(evaluationText("allowed shared source"), source), evaluationText("is not currently shared")))
			continue
		}
		allowed[source] = struct{}{}
	}
	if policy.RejectSharedSources {
		var rejected []string
		for source := range shared {
			if _, ok := allowed[source]; !ok {
				rejected = append(rejected, source)
			}
		}
		slices.Sort(rejected)
		for _, source := range rejected {
			issues = append(issues, qualifySubject(
				subjectNamed(evaluationText("source"), source),
				evaluationText(fmt.Sprintf("is shared by %d static presentation slots", inventory.counts[source])),
			))
		}
	}
	return issues
}

func collectLocalizationSources(schema cli.Schema) localizationSourceInventory {
	inventory := localizationSourceInventory{
		counts: make(map[string]int),
	}
	for _, command := range schema.Commands {
		if command.Origin != cli.DeclarationOriginApplication {
			continue
		}
		inventory.addDescription(command.Description)
		inventory.addExamples(command.Examples)
		for _, group := range command.Help.CommandGroups {
			if group.Kind == cli.HelpGroupDeclared {
				inventory.add(group.Title)
			}
		}
		for _, group := range command.Help.FlagGroups {
			if group.Kind == cli.HelpGroupDeclared {
				inventory.add(group.Title)
			}
		}
		for _, advisory := range command.Advisories {
			if advisory.Origin != cli.DeclarationOriginApplication {
				continue
			}
			inventory.add(advisory.Message)
			inventory.add(advisory.Reason)
			inventory.add(advisory.Hint)
		}
	}
	for _, entry := range schema.CommandEntries {
		if entry.Origin != cli.DeclarationOriginApplication {
			continue
		}
		// Canonical command presentation is stored on CommandSchema as the
		// canonical source slot. The canonical entry repeats that presentation
		// for entry projection and must not become a second localization owner.
		if entry.Kind != cli.CommandEntryCanonical {
			inventory.addDescription(entry.Description)
			inventory.addExamples(entry.Examples)
		}
		inventory.addRelease(entry.Release)
	}
	for _, entry := range schema.ActionEntries {
		if entry.Origin != cli.DeclarationOriginApplication {
			continue
		}
		inventory.addDescription(entry.Description)
		inventory.addRelease(entry.Release)
	}
	for _, flag := range schema.Flags {
		if flag.Origin != cli.DeclarationOriginApplication {
			continue
		}
		inventory.add(flag.Description)
		inventory.addRelease(flag.Release)
	}
	for _, spelling := range schema.FlagSpellings {
		if spelling.Origin == cli.DeclarationOriginApplication {
			inventory.addRelease(spelling.Release)
		}
	}
	for _, form := range schema.ArgumentForms {
		if form.Origin == cli.DeclarationOriginApplication {
			inventory.addRelease(form.Release)
		}
	}
	for _, term := range schema.ArgumentTerms {
		if term.Origin != cli.DeclarationOriginApplication {
			continue
		}
		inventory.add(term.Description)
		inventory.addRelease(term.Release)
	}
	for _, input := range schema.EnvironmentInputs {
		if input.Origin == cli.DeclarationOriginApplication {
			inventory.addRelease(input.Release)
		}
	}
	for _, entry := range schema.ValueDomainEntries {
		if entry.Origin != cli.DeclarationOriginApplication {
			continue
		}
		inventory.addDescription(entry.Description)
		inventory.addRelease(entry.Release)
	}
	for _, policy := range schema.SignalPolicies {
		if policy.Origin == cli.DeclarationOriginApplication {
			inventory.add(policy.Hint)
		}
	}
	return inventory
}

func canonicalAuditLocale(app cli.Application, locale cli.Locale) (string, error) {
	options := cli.ReferenceOptions{
		Locale:     locale,
		Visibility: cli.ReferenceVisibilityAll,
		Order:      cli.ReferenceOrderDeclaration,
	}
	reference, err := app.Reference(options)
	if err != nil {
		failure := localizationReferenceError{
			locale: locale,
			cause:  err,
		}
		return "", failure
	}
	return string(reference.Locale), nil
}

func missingCompleteSources(locale, sourceLocale string, bundles map[string]cli.Translations, sources map[string]struct{}) []string {
	if locale == sourceLocale && locale != "" {
		return nil
	}
	parents := canonicalLocaleParents(locale)
	var missing []string
	for source := range sources {
		translated := false
		for _, parent := range parents {
			if parent == sourceLocale && sourceLocale != "" {
				break
			}
			if translations := bundles[parent]; translations != nil {
				if _, ok := translations[source]; ok {
					translated = true
					break
				}
			}
		}
		if !translated {
			missing = append(missing, source)
		}
	}
	slices.Sort(missing)
	return missing
}

func missingDirectSources(translations cli.Translations, sources map[string]struct{}) []string {
	var missing []string
	for source := range sources {
		if _, ok := translations[source]; !ok {
			missing = append(missing, source)
		}
	}
	slices.Sort(missing)
	return missing
}

// canonicalLocaleParents operates only on the canonical effective locale
// returned by the public Reference operation. It removes lookup-irrelevant
// extension/private-use suffixes and then walks the structural BCP-47 parent
// chain without inventing sibling-region fallback. Extended language subtags
// remain part of the base language identity.
func canonicalLocaleParents(locale string) []string {
	if locale == "" || locale == "C" || locale == "POSIX" {
		return nil
	}
	parts := strings.Split(locale, "-")
	baseEnd := len(parts)
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) == 1 {
			baseEnd = i
			break
		}
	}
	parts = parts[:baseEnd]
	minimum := 1
	if len(parts) > 1 && len(parts[0]) <= 3 && len(parts[1]) == 3 && asciiLetters(parts[1]) {
		minimum = 2
		for minimum < len(parts) && minimum <= 3 && len(parts[minimum]) == 3 && asciiLetters(parts[minimum]) {
			minimum++
		}
	}
	result := make([]string, 0, len(parts)-minimum+1)
	for end := len(parts); end >= minimum; end-- {
		result = append(result, strings.Join(parts[:end], "-"))
	}
	return result
}

func asciiLetters(value string) bool {
	for i := range len(value) {
		if (value[i] < 'A' || value[i] > 'Z') && (value[i] < 'a' || value[i] > 'z') {
			return false
		}
	}
	return value != ""
}

func (err localizationReferenceError) Error() string {
	return "clitest: resolve canonical audit locale " + string(err.locale) + ": " + err.cause.Error()
}

func (err localizationReferenceError) Unwrap() error { return err.cause }

func (localizationSchemaExpectation) schemaExpectation() {}

func (e localizationSchemaExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	if schema == nil {
		return disprovedResult(evaluationText("schema localization"), evaluationText("complete localization snapshot"), evaluationText("schema is nil"))
	}
	if ok, actual := compareLocalization(e.expected, schema.Localization); !ok {
		return disprovedResult(evaluationText("schema localization"), formatLocalization(e.expected), actual)
	}
	return provedResult()
}

func (inventory localizationSourceInventory) add(source string) {
	if source != "" {
		inventory.counts[source]++
	}
}

func (inventory localizationSourceInventory) addDescription(description cli.Description) {
	inventory.add(description.Short)
	inventory.add(description.Long)
}

func (inventory localizationSourceInventory) addExamples(examples []cli.Example) {
	for _, example := range examples {
		inventory.add(example.Description)
	}
}

func (inventory localizationSourceInventory) addRelease(release cli.ReleaseLifecycle) {
	inventory.add(release.Reason)
}

func (effectiveLocaleExpectationValue) referenceExpectation() {}

func (effectiveLocaleExpectationValue) inspectionExpectation() {}

func (effectiveLocaleExpectationValue) completionExpectation() {}

func (effectiveLocaleExpectationValue) helpPresentationExpectation() {}

func (expectation effectiveLocaleExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return compareLocale("reference locale", expectation.locale, reference.Locale)
}

func (expectation effectiveLocaleExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return compareLocale("inspection locale", expectation.locale, inspection.Locale)
}

func (expectation effectiveLocaleExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	return compareLocale("completion locale", expectation.locale, result.Locale)
}

func (expectation effectiveLocaleExpectationValue) evaluateHelp(page cli.HelpPage) evaluationResult {
	return compareLocale("help locale", expectation.locale, page.Locale)
}
