package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

const localizationAuditApplicationName = "localization-audit"

func TestLocalizationAuditUsesStaticSourcesParentFallbackRuntimeSourcesAndExactness(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: localizationAuditApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Description: cli.Description{
			Short: "shared",
			Long:  "root long",
		},
		Examples: []cli.Example{{
			Description: "shared",
			CommandLine: localizationAuditApplicationName,
		}},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{
				{
					Locale: "fr",
					Translations: cli.Translations{
						"shared":    "",
						"root long": "racine longue",
						"runtime":   "exécution",
					},
				},
			},
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	policy := LocalizationPolicy{
		RequireCompleteLocales:         []cli.Locale{"fr-CA"},
		RequireExactLocales:            []cli.Locale{"fr"},
		RuntimeSources:                 []string{"runtime"},
		RejectUnreferencedTranslations: true,
		RejectSharedSources:            true,
		AllowedSharedSources:           []string{"shared"},
	}
	if issues := auditLocalization(app, policy); len(issues) != 0 {
		t.Fatalf("audit issues = %#v", issues)
	}

	policy.RequireExactLocales = []cli.Locale{"fr-CA"}
	issues := auditLocalization(app, policy)
	if !containsIssue(issues, `locale "fr-CA" is not exact`) {
		t.Fatalf("exact-locale issues = %#v", issues)
	}
}

func TestLocalizationAuditRejectsInvalidRuntimeUnreferencedAndStaleSharedAllowance(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: "localization-invalid",
		Description: cli.Description{
			Short: "known",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"known": "connu",
					"old":   "ancien",
				},
			}},
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	policy := LocalizationPolicy{
		RuntimeSources:                 []string{"known", "runtime", "runtime"},
		RejectUnreferencedTranslations: true,
		AllowedSharedSources:           []string{"not-shared"},
	}
	issues := auditLocalization(app, policy)
	for _, want := range []string{
		`runtime source "known" is already statically discoverable`,
		`runtime source "runtime" is duplicated`,
		`contains unreferenced translation "old"`,
		`allowed shared source "not-shared" is not currently shared`,
	} {
		if !containsIssue(issues, want) {
			t.Fatalf("issues %q do not contain %q", issues, want)
		}
	}
}

func TestLocalizationSourceInventoryDoesNotDoubleCountCanonicalCommandProjection(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	entryID := cli.CommandEntryID(1)
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			commandID: {
				ID:     commandID,
				Origin: cli.DeclarationOriginApplication,
				Description: cli.Description{
					Short: "canonical",
				},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{
			entryID: {
				ID:     entryID,
				Origin: cli.DeclarationOriginApplication,
				Kind:   cli.CommandEntryCanonical,
				Description: cli.Description{
					Short: "canonical",
				},
			},
		},
	}
	inventory := collectLocalizationSources(*schema)
	if got := inventory.counts["canonical"]; got != 1 {
		t.Fatalf("canonical source count = %d, want 1", got)
	}
}

func TestCanonicalLocaleParentsPreservesExtlangBaseAndDropsLookupIrrelevantSuffixes(t *testing.T) {
	t.Parallel()
	got := canonicalLocaleParents("zh-cmn-Hans-CN-variant-u-co-phonebk-x-private")
	want := []string{"zh-cmn-Hans-CN-variant", "zh-cmn-Hans-CN", "zh-cmn-Hans", "zh-cmn"}
	if len(got) != len(want) {
		t.Fatalf("parents = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parents = %#v, want %#v", got, want)
		}
	}
}

func containsIssue(issues []evaluationField, fragment string) bool {
	for _, issue := range issues {
		if strings.Contains(renderEvaluationField(issue), fragment) {
			return true
		}
	}
	return false
}

func TestPresentationExpectationsObserveSourceResolvedCompletionAndHelpWithoutOwningLocalization(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	sourceDescription := cli.Description{
		Short: "source short",
		Long:  "source long",
	}
	presentedDescription := cli.Description{
		Short: "court",
		Long:  "longue",
	}
	sourceExamples := []cli.Example{{
		Description: "source example",
		CommandLine: "tool source",
	}}
	presentedExamples := []cli.Example{{
		Description: "exemple", //nolint:misspell // Intentional French localization fixture.
		CommandLine: "tool source",
	}}

	schema := cli.Schema{
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"source short": "court",
				},
			}},
		},
		Commands: map[cli.CommandID]cli.CommandSchema{
			commandID: {
				ID:          commandID,
				Path:        []string{},
				Description: sourceDescription,
				Examples:    sourceExamples,
			},
		},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			commandID: {
				ID:          commandID,
				Path:        []string{},
				Description: presentedDescription,
				Examples:    presentedExamples,
			},
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:          commandID,
				Path:        []string{},
				Description: presentedDescription,
				Examples:    presentedExamples,
			},
		},
	}
	completion := cli.Completion{
		Items: []cli.CompletionItem{{
			Value:       "serve",
			DisplayText: "Servir",
			Description: "commande",
		}},
	}
	help := cli.HelpPage{
		Description: presentedDescription,
		Examples:    presentedExamples,
	}

	localization := cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				"source short": "court",
			},
		}},
	}
	localizationExpectation := LocalizationExactly(localization)
	localizationEvaluator, ok := localizationExpectation.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("LocalizationExactly expectation does not expose schema evaluation")
	}
	localization.Bundles[0].Translations["source short"] = "mutated expectation input"
	for i := range 2 {
		if outcome := localizationEvaluator.evaluateSchema(&schema); outcome.status != evaluationProved {
			t.Fatalf("LocalizationExactly reuse %d outcome = %#v", i, outcome)
		}
	}

	sourceDescriptionExpectation := RootCommand().SourceDescription(sourceDescription)
	sourceDescriptionEvaluator, ok := sourceDescriptionExpectation.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("SourceDescription expectation does not expose schema evaluation")
	}
	if outcome := sourceDescriptionEvaluator.evaluateSchema(&schema); outcome.status != evaluationProved {
		t.Fatalf("SourceDescription outcome = %#v", outcome)
	}
	sourceExamplesExpectation := RootCommand().SourceExamplesExactly(sourceExamples...)
	sourceExamplesEvaluator, ok := sourceExamplesExpectation.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("SourceExamplesExactly expectation does not expose schema evaluation")
	}
	if outcome := sourceExamplesEvaluator.evaluateSchema(&schema); outcome.status != evaluationProved {
		t.Fatalf("SourceExamplesExactly outcome = %#v", outcome)
	}

	presentedDescriptionExpectation := RootCommand().PresentedDescription(presentedDescription)
	presentedDescriptionReferenceEvaluator, ok := presentedDescriptionExpectation.(interface {
		evaluateReference(reference cli.Reference, err error) evaluationResult
	})
	if !ok {
		t.Fatal("PresentedDescription expectation does not expose reference evaluation")
	}
	if outcome := presentedDescriptionReferenceEvaluator.evaluateReference(reference, nil); outcome.status != evaluationProved {
		t.Fatalf("PresentedDescription Reference outcome = %#v", outcome)
	}
	presentedDescriptionInspectionEvaluator, ok := presentedDescriptionExpectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("PresentedDescription expectation does not expose inspection evaluation")
	}
	if outcome := presentedDescriptionInspectionEvaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
		t.Fatalf("PresentedDescription Inspection outcome = %#v", outcome)
	}
	presentedExamplesExpectation := RootCommand().PresentedExamplesExactly(presentedExamples...)
	presentedExamplesReferenceEvaluator, ok := presentedExamplesExpectation.(interface {
		evaluateReference(reference cli.Reference, err error) evaluationResult
	})
	if !ok {
		t.Fatal("PresentedExamplesExactly expectation does not expose reference evaluation")
	}
	if outcome := presentedExamplesReferenceEvaluator.evaluateReference(reference, nil); outcome.status != evaluationProved {
		t.Fatalf("PresentedExamplesExactly Reference outcome = %#v", outcome)
	}
	presentedExamplesInspectionEvaluator, ok := presentedExamplesExpectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("PresentedExamplesExactly expectation does not expose inspection evaluation")
	}
	if outcome := presentedExamplesInspectionEvaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
		t.Fatalf("PresentedExamplesExactly Inspection outcome = %#v", outcome)
	}

	completionExpectation := CompletionItem("serve").Presentation("Servir", "commande")
	completionEvaluator, ok := completionExpectation.(interface {
		evaluateCompletion(completion cli.Completion, err error) evaluationResult
	})
	if !ok {
		t.Fatal("CompletionItem Presentation expectation does not expose completion evaluation")
	}
	for i := range 2 {
		if outcome := completionEvaluator.evaluateCompletion(completion, nil); outcome.status != evaluationProved {
			t.Fatalf("Completion Presentation reuse %d outcome = %#v", i, outcome)
		}
	}

	RequireHelpPresentation(t, help, HelpDescription(presentedDescription), HelpExamplesExactly(presentedExamples...))
}

func TestPresentationExpectationsAttributeFieldMismatchAndRefuseExactTruncatedObservation(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	schema := cli.Schema{
		Localization: cli.Localization{
			DefaultLocale: "fr",
		},
		Commands: map[cli.CommandID]cli.CommandSchema{
			commandID: {
				ID:   commandID,
				Path: []string{},
				Description: cli.Description{
					Short: "actual source",
				},
				Examples: []cli.Example{{
					Description: "actual source example",
				}},
			},
		},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			commandID: {
				ID:   commandID,
				Path: []string{},
				Description: cli.Description{
					Short: "actual",
				},
				Examples: []cli.Example{{
					Description: "actual example",
				}},
			},
		},
	}
	inspection := cli.Inspection{
		Truncated: true,
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:   commandID,
				Path: []string{},
				Description: cli.Description{
					Short: "actual",
				},
				Examples: []cli.Example{{
					Description: "seen",
				}},
			},
		},
	}

	localization := LocalizationExactly(cli.Localization{
		DefaultLocale: "en",
	})
	localizationEvaluator, ok := localization.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("localization expectation does not expose schema evaluation")
	}
	outcome := localizationEvaluator.evaluateSchema(&schema)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.subject) != "schema localization" ||
		!strings.Contains(renderEvaluationField(outcome.observed), `source locale "fr"`) {
		t.Fatalf("localization mismatch outcome = %#v", outcome)
	}

	sourceDescription := RootCommand().SourceDescription(cli.Description{
		Short: "expected source",
	})
	sourceDescriptionEvaluator, ok := sourceDescription.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("sourceDescription expectation does not expose schema evaluation")
	}
	outcome = sourceDescriptionEvaluator.evaluateSchema(&schema)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "source description short") ||
		renderEvaluationField(outcome.observed) != "actual source" {
		t.Fatalf("source description mismatch outcome = %#v", outcome)
	}

	sourceExamplesInput := []cli.Example{{
		Description: "expected source example",
	}}
	sourceExamples := RootCommand().SourceExamplesExactly(sourceExamplesInput...)
	sourceExamplesInput[0].Description = "mutated expectation input"
	sourceExamplesEvaluator, ok := sourceExamples.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("sourceExamples expectation does not expose schema evaluation")
	}
	outcome = sourceExamplesEvaluator.evaluateSchema(&schema)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "source examples") ||
		!strings.Contains(renderEvaluationField(outcome.expected), "index 0=") {
		t.Fatalf("source examples mismatch outcome = %#v", outcome)
	}

	description := RootCommand().PresentedDescription(cli.Description{
		Short: "expected",
	})
	descriptionEvaluator, ok := description.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("PresentedDescription expectation does not expose inspection evaluation")
	}
	outcome = descriptionEvaluator.evaluateInspection(inspection, nil)
	if outcome.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(outcome.subject), "presented description short") ||
		renderEvaluationField(outcome.observed) != "actual" {
		t.Fatalf("description mismatch outcome = %#v", outcome)
	}

	presentedExamples := RootCommand().PresentedExamplesExactly(cli.Example{
		Description: "expected example",
	})
	presentedExamplesReferenceEvaluator, ok := presentedExamples.(interface {
		evaluateReference(reference cli.Reference, err error) evaluationResult
	})
	if !ok {
		t.Fatal("PresentedExamplesExactly expectation does not expose reference evaluation")
	}
	outcome = presentedExamplesReferenceEvaluator.evaluateReference(reference, nil)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "presented examples") ||
		!strings.Contains(renderEvaluationField(outcome.observed), "index 0=") {
		t.Fatalf("presented examples mismatch outcome = %#v", outcome)
	}
	presentedExamplesInspectionEvaluator, ok := presentedExamples.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("PresentedExamplesExactly expectation does not expose inspection evaluation")
	}
	outcome = presentedExamplesInspectionEvaluator.evaluateInspection(inspection, nil)
	if outcome.status != evaluationNotProvable || !strings.Contains(renderEvaluationField(outcome.observed), "truncated") {
		t.Fatalf("truncated exact examples outcome = %#v", outcome)
	}

	completion := cli.Completion{
		Items: []cli.CompletionItem{{
			Value:       "serve",
			DisplayText: "actual",
			Description: "description",
		}},
	}
	presentation := CompletionItem("serve").Presentation("expected", "description")
	presentationEvaluator, ok := presentation.(interface {
		evaluateCompletion(completion cli.Completion, err error) evaluationResult
	})
	if !ok {
		t.Fatal("CompletionItem Presentation expectation does not expose completion evaluation")
	}
	outcome = presentationEvaluator.evaluateCompletion(completion, nil)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "presentation") ||
		!strings.Contains(renderEvaluationField(outcome.observed), `display "actual"`) {
		t.Fatalf("completion mismatch outcome = %#v", outcome)
	}

	help := cli.HelpPage{
		Description: cli.Description{
			Short: "actual help",
		},
		Examples: []cli.Example{{
			Description: "actual help example",
		}},
	}
	helpDescription := HelpDescription(cli.Description{
		Short: "expected help",
	})
	helpDescriptionEvaluator, ok := helpDescription.(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("HelpDescription expectation does not expose help evaluation")
	}
	outcome = helpDescriptionEvaluator.evaluateHelp(help)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "help description short") ||
		renderEvaluationField(outcome.observed) != "actual help" {
		t.Fatalf("help description mismatch outcome = %#v", outcome)
	}
	helpExamplesInput := []cli.Example{{
		Description: "expected help example",
	}}
	helpExamples := HelpExamplesExactly(helpExamplesInput...)
	helpExamplesInput[0].Description = "mutated expectation input"
	helpExamplesEvaluator, ok := helpExamples.(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("HelpExamplesExactly expectation does not expose help evaluation")
	}
	outcome = helpExamplesEvaluator.evaluateHelp(help)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "help examples") ||
		!strings.Contains(renderEvaluationField(outcome.expected), "index 0=") {
		t.Fatalf("help examples mismatch outcome = %#v", outcome)
	}
}

func TestLocalizationAuditAndSourceDescriptionExpectationContracts(t *testing.T) {
	t.Parallel()
	definition := cli.Definition[struct{}]{
		Name: localizationAuditApplicationName,
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Description: cli.Description{
			Short: "short",
			Long:  "long",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"short": "court",
					"long":  "longue",
				},
			}},
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// The public wrapper must execute the whole-application audit on a valid
	// policy without inventing a second audit model.
	validPolicy := LocalizationPolicy{
		RequireCompleteLocales: []cli.Locale{"fr"},
	}
	RequireLocalization(t, app, validPolicy)

	missingLocalePolicy := LocalizationPolicy{
		RequireCompleteLocales: []cli.Locale{"de"},
	}
	issues := auditLocalization(app, missingLocalePolicy)
	if len(issues) == 0 || !containsIssue(issues, `locale "de"`) {
		t.Fatalf("localization audit issues = %#v, want missing de locale", issues)
	}
	// A missing command drives the actual presentation expectation failure path
	// that owns formatDescription; both fields must survive into the diagnostic.
	description := cli.Description{
		Short: "expected short",
		Long:  "expected long",
	}
	expectation := CommandByID(999).SourceDescription(description)
	evaluator, ok := expectation.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("SourceDescription expectation does not expose schema evaluation")
	}
	outcome := evaluator.evaluateSchema(&cli.Schema{})
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.expected), `short="expected short"`) ||
		!strings.Contains(renderEvaluationField(outcome.expected), `long="expected long"`) {
		t.Fatalf("description failure diagnostic = %#v", outcome)
	}
}

type unsupportedHelpPresentationExpectation struct{}

func (unsupportedHelpPresentationExpectation) helpPresentationExpectation() {}

func TestHelpPresentationRejectsInvalidCoordinates(t *testing.T) {
	t.Parallel()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("unsupported help presentation expectation did not panic")
		}
		message, ok := recovered.(string)
		if !ok || !strings.Contains(message, "invalid HelpPresentationExpectation implementation") {
			t.Fatalf("panic = %#v", recovered)
		}
	}()
	RequireHelpPresentation(t, cli.HelpPage{}, unsupportedHelpPresentationExpectation{})
	t.Fatal("unsupported help presentation expectation returned")
}

func TestLocalizationSchemaPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	expectation, ok := LocalizationExactly(cli.Localization{}).(localizationSchemaExpectation)
	if !ok {
		t.Fatal("LocalizationExactly must return localizationSchemaExpectation")
	}
	outcome := expectation.evaluateSchema(nil)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "schema is nil" {
		t.Fatalf("nil schema outcome = %#v", outcome)
	}
}

func TestCommandPresentedDescriptionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("command is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		expectation, ok := RootCommand().PresentedDescription(cli.Description{
			Short: "expected",
		}).(commandPresentedDescriptionExpectation)
		if !ok {
			t.Fatal("PresentedDescription must return commandPresentedDescriptionExpectation")
		}
		outcome := expectation.evaluateReference(cli.Reference{}, nil)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "command is absent" {
			t.Fatalf("reference outcome = %#v", outcome)
		}
	})
	t.Run("commandpresenteddescription unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		expectation, ok := CommandByID(99).PresentedDescription(cli.Description{
			Short: "expected",
		}).(commandPresentedDescriptionExpectation)
		if !ok {
			t.Fatal("PresentedDescription must return commandPresentedDescriptionExpectation")
		}
		outcome := expectation.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status == evaluationProved {
			t.Fatalf("unresolved inspection outcome = %#v", outcome)
		}
	})
}

func TestCommandSourceExamplesPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	expectation, ok := CommandByID(99).SourceExamplesExactly(cli.Example{
		Description: "expected",
	}).(commandSourceExamplesExpectation)
	if !ok {
		t.Fatal("SourceExamplesExactly must return commandSourceExamplesExpectation")
	}
	outcome := expectation.evaluateSchema(&cli.Schema{})
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "command is absent" {
		t.Fatalf("schema outcome = %#v", outcome)
	}
}

func TestCommandPresentedExamplesPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("command is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		expectation, ok := CommandByID(99).PresentedExamplesExactly(cli.Example{
			Description: "expected",
		}).(commandPresentedExamplesExpectation)
		if !ok {
			t.Fatal("PresentedExamplesExactly must return commandPresentedExamplesExpectation")
		}
		outcome := expectation.evaluateReference(cli.Reference{}, nil)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "command is absent" {
			t.Fatalf("reference outcome = %#v", outcome)
		}
	})
	t.Run("commandpresentedexamples unresolved: incomplete observation", func(t *testing.T) {
		t.Parallel()
		expectation, ok := CommandByID(99).PresentedExamplesExactly(cli.Example{
			Description: "expected",
		}).(commandPresentedExamplesExpectation)
		if !ok {
			t.Fatal("PresentedExamplesExactly must return commandPresentedExamplesExpectation")
		}
		outcome := expectation.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil)
		if outcome.status == evaluationProved {
			t.Fatalf("unresolved inspection outcome = %#v", outcome)
		}
	})
	t.Run("presented examples: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expectation, ok := CommandByID(1).PresentedExamplesExactly(cli.Example{
			Description: "expected",
		}).(commandPresentedExamplesExpectation)
		if !ok {
			t.Fatal("PresentedExamplesExactly must return commandPresentedExamplesExpectation")
		}
		inspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{
				1: {
					ID: 1,
					Examples: []cli.Example{{
						Description: "actual",
					}},
				},
			},
		}
		outcome := expectation.evaluateInspection(inspection, nil)
		if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "presented examples") {
			t.Fatalf("mismatch outcome = %#v", outcome)
		}
	})
}

func TestCompletionItemPresentationPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("completionitempresentation unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		expectation, ok := CompletionItemAt(1).Presentation(
			"expected",
			"expected",
		).(completionItemPresentationExpectation)
		if !ok {
			t.Fatal("Presentation must return completionItemPresentationExpectation")
		}
		outcome := expectation.evaluateCompletion(
			cli.Completion{},
			nil,
		)
		if outcome.status == evaluationProved {
			t.Fatalf("unresolved completion outcome = %#v", outcome)
		}
	})
	t.Run("redacted value: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expectation, ok := CompletionItemAt(0).Presentation(
			"expected",
			"expected",
		).(completionItemPresentationExpectation)
		if !ok {
			t.Fatal("Presentation must return completionItemPresentationExpectation")
		}
		completion := cli.Completion{
			Items: []cli.CompletionItem{{
				DisplayText: "actual",
				Description: "actual",
				Sensitive:   true,
			}},
		}
		outcome := expectation.evaluateCompletion(completion, nil)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.expected) != "redacted expected presentation" ||
			renderEvaluationField(outcome.observed) != "redacted observed presentation" {
			t.Fatalf("redacted outcome = %#v", outcome)
		}
	})
}

func TestDescriptionPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()
	description := cli.Description{
		Short: "same",
		Long:  "expected",
	}
	actualDescription := cli.Description{
		Short: "same",
		Long:  "actual",
	}
	outcome := compareDescription(evaluationText("description"), description, actualDescription)
	if outcome.status != evaluationDisproved || !strings.HasSuffix(renderEvaluationField(outcome.subject), " long") {
		t.Fatalf("long mismatch outcome = %#v", outcome)
	}
}

func TestLocalizationPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			expected cli.Localization
			actual   cli.Localization
			fragment string
		}{
			{
				name: "bundle count",
				expected: cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
					}},
				},
				actual: cli.Localization{
					DefaultLocale: "en",
				},
				fragment: "0 bundles",
			},
			{
				name: "bundle locale",
				expected: cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
					}},
				},
				actual: cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "de",
					}},
				},
				fragment: `locale "de"`,
			},
			{
				name: "translation count",
				expected: cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"a": "A",
						},
					}},
				},
				actual: cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
					}},
				},
				fragment: "0 translations",
			},
			{
				name: "missing source",
				expected: cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"a": "A",
							"b": "B",
						},
					}},
				},
				actual: cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"a": "A",
							"c": "C",
						},
					}},
				},
				fragment: `missing source "b"`,
			},
		}
		for _, tc := range cases {
			ok, actual := compareLocalization(tc.expected, tc.actual)
			if ok || !strings.Contains(renderEvaluationField(actual), tc.fragment) {
				t.Fatalf("%s comparison = %v, %q", tc.name, ok, renderEvaluationField(actual))
			}
		}
	})
}

func TestAuditLocalizationPreservesSemanticResult(t *testing.T) {
	t.Parallel()

	definition := cli.Definition[struct{}]{
		Name: "localization-audit-residual",
		Description: cli.Description{
			Short: "shared",
			Long:  "shared",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"shared": "partage",
				},
			}},
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	t.Run("membership or order mismatch", func(t *testing.T) {
		t.Parallel()
		policy := LocalizationPolicy{
			RuntimeSources: []string{"", string([]byte{0xff})},
		}
		issues := auditLocalization(app, policy)
		if !containsIssue(issues, "RuntimeSources[0] is empty") ||
			!containsIssue(issues, "RuntimeSources[1] is not valid UTF-8") {
			t.Fatalf("issues = %#v", issues)
		}
	})
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		policy := LocalizationPolicy{
			RequireCompleteLocales:         []cli.Locale{"fr"},
			RequireExactLocales:            []cli.Locale{"en"},
			RejectUnreferencedTranslations: true,
			RejectSharedSources:            true,
		}
		issues := auditLocalization(app, policy)
		if !containsIssue(issues, `source locale "en" cannot be required as an exact target locale`) {
			t.Fatalf("issues = %#v", issues)
		}
	})
	t.Run("count mismatch: membership or order mismatch", func(t *testing.T) {
		t.Parallel()
		policy := LocalizationPolicy{
			RejectSharedSources: true,
		}
		issues := auditLocalization(app, policy)
		if !containsIssue(issues, `source "shared" is shared by 2 static presentation slots`) {
			t.Fatalf("issues = %#v", issues)
		}
	})
}

func TestAuditLocalizationRejectsInvalidCoordinates(t *testing.T) {
	t.Parallel()

	definition := cli.Definition[struct{}]{
		Name: "localization-invalid-locale",
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		policy := LocalizationPolicy{
			RequireCompleteLocales: []cli.Locale{"not_a_locale"},
			RequireExactLocales:    []cli.Locale{"also_not_a_locale"},
		}
		issues := auditLocalization(app, policy)
		if !containsIssue(issues, "required complete locale") || !containsIssue(issues, "required exact locale") {
			t.Fatalf("issues = %#v", issues)
		}
	})
}

func TestCollectLocalizationSourcesPreservesSemanticResult(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		schema := cli.Schema{
			Commands: map[cli.CommandID]cli.CommandSchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Description: cli.Description{
						Short: "command short",
						Long:  "command long",
					},
					Examples: []cli.Example{{
						Description: "command example",
					}},
					Help: cli.HelpSchema{
						CommandGroups: []cli.HelpCommandGroupSchema{{
							Kind:  cli.HelpGroupDeclared,
							Title: "command group",
						}},
						FlagGroups: []cli.HelpFlagGroupSchema{{
							Kind:  cli.HelpGroupDeclared,
							Title: "flag group",
						}},
					},
					Advisories: []cli.AdvisorySchema{{
						Message: "app advisory message",
						Reason:  "app advisory reason",
						Hint:    "app advisory hint",
						Origin:  cli.DeclarationOriginApplication,
					}, {
						Message: "frontend advisory",
						Origin:  cli.DeclarationOriginFrontend,
					}},
				},
				2: {
					ID:     2,
					Origin: cli.DeclarationOriginFrontend,
					Description: cli.Description{
						Short: "frontend command",
					},
				},
			},
			CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{
				1: {
					ID:     1,
					Kind:   cli.CommandEntryCanonical,
					Origin: cli.DeclarationOriginApplication,
					Description: cli.Description{
						Short: "canonical duplicate",
					},
					Release: cli.ReleaseLifecycle{
						Reason: "canonical release",
					},
				},
				2: {
					ID:     2,
					Kind:   cli.CommandEntryAlias,
					Origin: cli.DeclarationOriginApplication,
					Description: cli.Description{
						Short: "alias short",
						Long:  "alias long",
					},
					Examples: []cli.Example{{
						Description: "alias example",
					}},
					Release: cli.ReleaseLifecycle{
						Reason: "alias release",
					},
				},
				3: {
					ID:     3,
					Kind:   cli.CommandEntryAlias,
					Origin: cli.DeclarationOriginFrontend,
					Description: cli.Description{
						Short: "frontend entry",
					},
				},
			},
			ActionEntries: map[cli.ActionEntryID]cli.ActionEntrySchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Description: cli.Description{
						Short: "action short",
						Long:  "action long",
					},
					Release: cli.ReleaseLifecycle{
						Reason: "action release",
					},
				},
			},
			Flags: map[cli.FlagID]cli.FlagSchema{
				1: {
					ID:          1,
					Origin:      cli.DeclarationOriginApplication,
					Description: "flag description",
					Release: cli.ReleaseLifecycle{
						Reason: "flag release",
					},
				},
				2: {
					ID:          2,
					Origin:      cli.DeclarationOriginFrontend,
					Description: "frontend flag",
				},
			},
			FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Release: cli.ReleaseLifecycle{
						Reason: "spelling release",
					},
				},
			},
			ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Release: cli.ReleaseLifecycle{
						Reason: "form release",
					},
				},
			},
			ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
				1: {
					ID:          1,
					Origin:      cli.DeclarationOriginApplication,
					Description: "term description",
					Release: cli.ReleaseLifecycle{
						Reason: "term release",
					},
				},
				2: {
					ID:          2,
					Origin:      cli.DeclarationOriginFrontend,
					Description: "frontend term",
				},
			},
			EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Release: cli.ReleaseLifecycle{
						Reason: "environment release",
					},
				},
			},
			ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Description: cli.Description{
						Short: "domain short",
						Long:  "domain long",
					},
					Release: cli.ReleaseLifecycle{
						Reason: "domain release",
					},
				},
				2: {
					ID:     2,
					Origin: cli.DeclarationOriginFrontend,
					Description: cli.Description{
						Short: "frontend domain",
					},
				},
			},
			SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Hint:   "signal hint",
				},
			},
		}
		inventory := collectLocalizationSources(schema)
		for _, source := range []string{
			"command short",
			"command long",
			"command example",
			"command group",
			"flag group",
			"app advisory message",
			"app advisory reason",
			"app advisory hint",
			"canonical release",
			"alias short",
			"alias long",
			"alias example",
			"alias release",
			"action short",
			"action long",
			"action release",
			"flag description",
			"flag release",
			"spelling release",
			"form release",
			"term description",
			"term release",
			"environment release",
			"domain short",
			"domain long",
			"domain release",
			"signal hint",
		} {
			if inventory.counts[source] == 0 {
				t.Fatalf("source %q not collected: %#v", source, inventory.counts)
			}
		}
		if inventory.counts["canonical duplicate"] != 0 || inventory.counts["frontend advisory"] != 0 ||
			inventory.counts["frontend flag"] != 0 ||
			inventory.counts["frontend term"] != 0 ||
			inventory.counts["frontend domain"] != 0 {
			t.Fatalf("non-owning sources collected: %#v", inventory.counts)
		}
	})
}

func TestCollectLocalizationSourcesRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		schema := cli.Schema{
			Commands: map[cli.CommandID]cli.CommandSchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginApplication,
					Advisories: []cli.AdvisorySchema{{
						Message: "frontend advisory",
						Origin:  cli.DeclarationOriginFrontend,
					}},
				},
			},
			Flags: map[cli.FlagID]cli.FlagSchema{
				1: {
					ID:          1,
					Origin:      cli.DeclarationOriginFrontend,
					Description: "frontend flag",
				},
			},
			ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
				1: {
					ID:          1,
					Origin:      cli.DeclarationOriginFrontend,
					Description: "frontend term",
				},
			},
			ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{
				1: {
					ID:     1,
					Origin: cli.DeclarationOriginFrontend,
					Description: cli.Description{
						Short: "frontend domain",
					},
				},
			},
		}
		inventory := collectLocalizationSources(schema)
		if len(inventory.counts) != 0 {
			t.Fatalf("non-application sources collected: %#v", inventory.counts)
		}
	})
}

func TestCanonicalAuditLocalePreservesSemanticResult(t *testing.T) {
	t.Parallel()

	definition := cli.Definition[struct{}]{
		Name: "canonical-locale",
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := canonicalAuditLocale(app, "not_a_locale"); err == nil {
		t.Fatal("invalid locale unexpectedly canonicalized")
	}
}

func TestMissingCompleteSourcesPreservesSemanticResult(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		sources := map[string]struct{}{"a": {}, "b": {}}
		if missing := missingCompleteSources("en", "en", nil, sources); missing != nil {
			t.Fatalf("source-locale missing = %#v", missing)
		}
		value := map[string]cli.Translations{"fr": {"a": "A"}}
		missing := missingCompleteSources("fr-CA", "en", value, sources)
		if len(missing) != 1 || missing[0] != "b" {
			t.Fatalf("fallback missing = %#v", missing)
		}
	})
}

func TestCanonicalLocaleParentsPreservesSemanticResult(t *testing.T) {
	t.Parallel()

	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		if parents := canonicalLocaleParents(""); parents != nil {
			t.Fatalf("empty parents = %#v", parents)
		}
		parents := canonicalLocaleParents("zh-cmn-yue-Hans-CN-x-private")
		if len(parents) < 2 || parents[0] != "zh-cmn-yue-Hans-CN" || parents[len(parents)-1] != "zh-cmn-yue" {
			t.Fatalf("extlang parents = %#v", parents)
		}
	})
}

func TestAsciiLettersPreservesSemanticResult(t *testing.T) {
	t.Parallel()

	if asciiLetters("ab1") {
		t.Fatal("non-letter ASCII accepted")
	}
	if !asciiLetters("AbC") {
		t.Fatal("ASCII letters rejected")
	}
}

func TestMissingCompleteSourcesStopsAtDefaultLocaleBoundary(t *testing.T) {
	t.Parallel()
	value := map[string]struct{}{"missing": {}}
	missing := missingCompleteSources("en-US", "en", map[string]cli.Translations{}, value)
	if len(missing) != 1 || missing[0] != "missing" {
		t.Fatalf("missing sources = %v", missing)
	}
}
