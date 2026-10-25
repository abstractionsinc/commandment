package main

import (
	"context"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

const scanDescription = "scan nearby worlds"

type rootConfig struct {
	Locale string
}

var definition = cli.Definition[rootConfig]{
	Name: "alienctl",
	Localization: cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{
			{
				Locale: "fr",
				Translations: cli.Translations{
					scanDescription: "analyser les mondes proches",
				},
			},
		},
	},
	Declare: func(cb *cli.CommandBuilder[rootConfig]) {
		locale := cb.Values.String(&cb.Config.Locale)
		cb.Flags.ValueFor(locale).LocaleSelector().Occurrences(cli.Count.Optional()).Long("locale")

		scanCmd := cli.Command[struct{}]{
			Name: "scan",
			Description: cli.Description{
				Short: scanDescription,
			},
		}
		cb.Subcommands.Command(scanCmd)
	},
}

func main() {
	app, err := cli.Compile(definition)
	if err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.RunFromProcess(context.Background()); err != nil {
		_ = cli.WriteError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
