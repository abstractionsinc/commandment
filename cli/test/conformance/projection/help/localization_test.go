package conformance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type localizationConcurrencyConfig struct{}

func TestProviderLocalizationInspectionSnapshotDetachesAcrossOperations(t *testing.T) {
	t.Parallel()
	type config = localizationConcurrencyConfig
	calls := 0
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[config]) error {
				calls++
				batch.Localization(cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"remote command": "commande distante",
						},
					}},
				})
				batch.Command(cli.Command[config]{
					Name: "remote",
					Description: cli.Description{
						Short: "remote command",
					},
				})
				return nil
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	check := func(label string, inspection cli.Inspection, wantCalls int) {
		t.Helper()
		if calls != wantCalls {
			t.Fatalf("%s provider calls = %d, want %d", label, calls, wantCalls)
		}
		found := false
		for _, provider := range inspection.CommandProviders {
			if provider.Outcome == cli.CommandProviderOutcomeAccepted {
				found = true
				if provider.Localization == nil || len(provider.Localization.Bundles) != 1 ||
					provider.Localization.Bundles[0].Translations["remote command"] != "commande distante" {
					t.Fatalf("%s provider localization = %#v", label, provider.Localization)
				}
			}
		}
		if !found {
			t.Fatalf("%s accepted provider missing", label)
		}
	}
	first, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args:   []string{"remote"},
			Locale: "fr",
		})
		proof.Require(t, clitest.InspectionReady())
		return proof.Inspection()
	}()
	if err != nil {
		t.Fatal(err)
	}
	check("first", first, 1)
	for id, provider := range first.CommandProviders {
		if provider.Localization != nil {
			provider.Localization.Bundles[0].Translations["remote command"] = "mutated"
			first.CommandProviders[id] = provider
		}
	}
	second, err := app.Inspect(t.Context(), cli.InspectionRequest{
		Args:   []string{"remote"},
		Locale: "fr",
	})
	if err != nil {
		t.Fatal(err)
	}
	check("second", second, 2)
}

func TestApplicationTranslationsCannotRewriteFrameworkDiagnosticIdentity(t *testing.T) {
	t.Parallel()
	type config = localizationConcurrencyConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Description: cli.Description{
			Short: "tool short",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"tool short":   "outil court",
					"unknown-flag": "translated identity",
				},
			}},
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Locale: "fr",
		})
		proof.Require(t, clitest.InspectionReady(), clitest.EffectiveLocale("fr"))
		return proof.Inspection()
	}()
	if err != nil {
		t.Fatal(err)
	}
	root := ready.Commands[ready.Selection.Command.Root]
	if root.Description.Short != "outil court" {
		t.Fatalf("application presentation was not localized: %#v", root.Description)
	}
	rejected, err := app.Inspect(t.Context(), cli.InspectionRequest{
		Args:   []string{"--missing"},
		Locale: "fr",
	})
	if err == nil || len(rejected.Diagnostics) != 1 || rejected.Diagnostics[0].Code != cli.DiagnosticUnknownFlag {
		t.Fatalf("framework diagnostic identity was translated or lost: %#v err=%v", rejected.Diagnostics, err)
	}
}

func localizationMustCompilePublic[C any](t testing.TB, definition cli.Definition[C]) cli.Application {
	t.Helper()
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
}

func TestSnapshotLocalizationAcceptsZeroLocalization(t *testing.T) {
	t.Parallel()
	app := localizationMustCompilePublic(t, cli.Definition[struct{}]{
		Name: "tool",
	})
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.LocalizationExactly(cli.Localization{}),
	)
}

func TestSnapshotLocalizationAcceptsSourceOnlyLocalization(t *testing.T) {
	t.Parallel()
	app := localizationMustCompilePublic(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en-US",
		},
	})
	clitest.Schema(app).Require(t,
		clitest.RootCommand().Exists(),
		clitest.LocalizationExactly(cli.Localization{
			DefaultLocale: "en-US",
		}),
	)
}

func TestSnapshotLocalizationProjectsExactTranslationSourceIdentity(t *testing.T) {
	t.Parallel()
	translations := cli.Translations{
		"hello": "bonjour",
		"empty": "",
	}
	app := localizationMustCompilePublic(t, cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale:       "fr",
				Translations: translations,
			}},
		},
	})

	expected := cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				"hello": "bonjour",
				"empty": "",
			},
		}},
	}
	proof := clitest.Schema(app)
	proof.Require(t, clitest.LocalizationExactly(expected))

	translations["hello"] = "mutated caller"
	expected.Bundles[0].Translations["hello"] = "mutated expectation"
	proof.Require(t, clitest.LocalizationExactly(cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				"hello": "bonjour",
				"empty": "",
			},
		}},
	}))
}

func TestSnapshotLocalizationRejectsBundlesWithoutSourceLocale(t *testing.T) {
	t.Parallel()
	_, err := cli.Compile(cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	diagnosticProof := clitest.Diagnostics(err)
	diagnosticProof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestSnapshotLocalizationRejectsCanonicalSourceLocaleBundle(t *testing.T) {
	t.Parallel()
	_, err := cli.Compile(cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "EN",
				Translations: cli.Translations{
					"x": "y",
				},
			}},
		},
	})
	diagnosticProof := clitest.Diagnostics(err)
	diagnosticProof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

func TestSnapshotLocalizationRejectsDuplicateCanonicalBundleLocale(t *testing.T) {
	t.Parallel()
	_, err := cli.Compile(cli.Definition[struct{}]{
		Name: "tool",
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{
				{
					Locale: "fr",
					Translations: cli.Translations{
						"same": "un",
					},
				},
				{
					Locale: "FR",
					Translations: cli.Translations{
						"same": "deux",
					},
				},
			},
		},
	})
	diagnosticProof := clitest.Diagnostics(err)
	diagnosticProof.Require(t, clitest.Diagnostic(cli.DiagnosticInvalidDeclaration).Present())
}

type localizationSelectorConfig struct{ Locale string }

func TestInspectionHonorsApplicationLocaleSelectorPrecedence(t *testing.T) {
	t.Parallel()

	type config = localizationSelectorConfig
	declare := func(root *cli.CommandBuilder[config]) {
		locale := root.Values.String(&root.Config.Locale)
		root.Flags.ValueFor(locale).
			LocaleSelector().
			Occurrences(cli.Count.Optional()).
			Long("locale")
	}
	definition := cli.Definition[config]{
		Name: "alienctl",
		Description: cli.Description{
			Short: "short",
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"short": "court",
				},
			}},
		},
		Declare: declare,
	}
	app := clitest.Compile(t, definition)
	selectorRequest := cli.InspectionRequest{
		Args: []string{"--locale", "fr"},
	}
	hostRequest := cli.InspectionRequest{
		Args:   []string{"--locale", "fr"},
		Locale: "en",
	}
	defaultRequest := cli.InspectionRequest{}

	clitest.Inspect(t.Context(), app, selectorRequest).Require(t,
		clitest.InspectionReady(),
		clitest.EffectiveLocale("fr"),
		clitest.RootCommand().PresentedDescription(cli.Description{
			Short: "court",
		}),
	)
	clitest.Inspect(t.Context(), app, hostRequest).Require(t,
		clitest.InspectionReady(),
		clitest.EffectiveLocale("en"),
		clitest.RootCommand().PresentedDescription(cli.Description{
			Short: "short",
		}),
	)
	clitest.Inspect(t.Context(), app, defaultRequest).Require(t,
		clitest.InspectionReady(),
		clitest.EffectiveLocale("en"),
	)
}

func TestHelpHonorsApplicationLocaleSelectorPrecedence(t *testing.T) {
	t.Parallel()

	type config = localizationSelectorConfig
	var page cli.HelpPage
	definition := cli.Definition[config]{
		Name: "alienctl",
		Description: cli.Description{
			Short: "short",
		},
		Help: cli.HelpConfig{
			Renderer: func(_ context.Context, observed cli.HelpPage) (string, error) {
				page = observed
				return "help\n", nil
			},
		},
		Localization: cli.Localization{
			DefaultLocale: "en",
			Bundles: []cli.TranslationBundle{{
				Locale: "fr",
				Translations: cli.Translations{
					"short": "court",
				},
			}},
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			locale := root.Values.String(&root.Config.Locale)
			root.Flags.ValueFor(locale).
				LocaleSelector().
				Occurrences(cli.Count.Optional()).
				Long("locale")
		},
	}
	app := clitest.Compile(t, definition)
	tests := []struct {
		name        string
		request     cli.RunRequest
		wantLocale  cli.Locale
		wantSummary string
	}{
		{
			name:        "empty invocation uses application default",
			request:     cli.RunRequest{},
			wantLocale:  "en",
			wantSummary: "short",
		},
		{
			name: "help flag uses application default",
			request: cli.RunRequest{
				Args: []string{"--help"},
			},
			wantLocale:  "en",
			wantSummary: "short",
		},
		{
			name: "selector overrides application default",
			request: cli.RunRequest{
				Args: []string{"--locale", "fr", "--help"},
			},
			wantLocale:  "fr",
			wantSummary: "court",
		},
		{
			name: "host locale overrides selector",
			request: cli.RunRequest{
				Args:   []string{"--locale", "fr", "--help"},
				Locale: "en",
			},
			wantLocale:  "en",
			wantSummary: "short",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page = cli.HelpPage{}
			var output strings.Builder
			test.request.Stdout = &output
			if err := app.Run(t.Context(), test.request); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if output.String() != "help\n" {
				t.Fatalf("Help output = %q", output.String())
			}
			if page.Locale != test.wantLocale || page.Description.Short != test.wantSummary {
				t.Fatalf("Help locale/description = %q / %#v", page.Locale, page.Description)
			}
		})
	}
}
