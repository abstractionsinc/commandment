package main

import (
	"context"
	"fmt"
	"os"

	"github.com/abstractionsinc/commandment/cli"
)

type rootConfig struct {
	Verbose bool
}

type navigateConfig struct {
	Destination string
	Approach    string
	Token       string
}

var definition = cli.Definition[rootConfig]{
	Name: "alienctl",
	Description: cli.Description{
		Long: "Navigate expedition craft across nearby worlds.",
	},
	Release: cli.Release.Ordered("2", "1", "2"),
	Examples: []cli.Example{{
		Description: "Return the expedition home.",
		CommandLine: "alienctl home",
	}},
	Declare: func(cb *cli.CommandBuilder[rootConfig]) {
		cb.Flags.Bool(&cb.Config.Verbose).
			Description("Show navigation detail.").
			Placement(cli.FlagPlacementSubtree).
			Long("verbose")

		cb.Subcommands.HelpGroup("Navigation", func(commands *cli.Subcommands[rootConfig]) {
			cmd := cli.Command[navigateConfig]{
				Name:    "navigate",
				Aliases: cli.Aliases("travel"),
				Description: cli.Description{
					Long: "Navigate the expedition to a destination.",
				},
				Examples: []cli.Example{{
					Description: "Travel to Europa by warp corridor.",
					CommandLine: "alienctl navigate europa --approach=warp",
				}},
				Declare: func(cb *cli.CommandBuilder[navigateConfig]) {
					approaches := cli.Value.Domain(cli.Value.Assign[string](), func(values *cli.ValueDomainBuilder[string]) {
						orbit := values.Value("orbit", "orbit").Description("Enter a stable orbit.")
						legacyRelease := cli.ReleaseLifecycle{
							DeprecatedIn: "2",
							Reason:       "transporter navigation is being retired",
						}
						legacy := values.Value("teleport", "teleport").
							Description("Use the legacy transporter.").
							Release(legacyRelease)
						warp := values.Value("warp", "warp").Description("Use a controlled warp corridor.")
						legacy.ReplacedBy(warp)
						values.Alias("cruise", orbit)
					})
					approach := cb.Values.Value(&cb.Config.Approach, approaches)
					cb.Values.Default(approach, "orbit")
					cb.Flags.ValueFor(approach).
						ValueName("METHOD").
						Description("Choose the navigation approach.").
						Long("approach")
					cb.Flags.String(&cb.Config.Token).
						ValueName("TOKEN").
						Description("Authenticate the navigation link.").
						Long("token").
						ImplicitValue("classified", cli.Source.Sensitive()).
						Sensitive()
					cb.Arguments.String(&cb.Config.Destination, "destination").
						Description("World or station to navigate to.")

					shortcut := cli.CommandShortcut{
						Name: "home",
						Description: cli.Description{
							Short: "Navigate directly to Earth.",
						},
						Args: []string{"earth"},
					}
					cb.Shortcut(shortcut)
				},
				Run: func(_ context.Context, cfg *navigateConfig) error {
					fmt.Printf("navigating to %s by %s\n", cfg.Destination, cfg.Approach)
					return nil
				},
			}
			commands.Command(cmd)
		})
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
