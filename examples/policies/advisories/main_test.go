package main

import (
	"context"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestQuietSuppressesTheRecklessAdvisory(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	command := clitest.Command("approach")
	notice := clitest.Notice(cli.NoticeAdvisory).Advisory("reckless-approach")
	recklessRequest := cli.InspectionRequest{
		Args: []string{"approach", "--reckless", "earth"},
	}
	clitest.Inspect(t.Context(), app, recklessRequest).Require(t,
		clitest.InspectionReady(),
		command.Advisory("reckless-approach").Outcome(cli.AdvisoryOutcomeMatched),
		clitest.NoticesExactly(notice),
	)
	var delivered []string
	notify := func(_ context.Context, notice cli.Notice) error {
		delivered = append(delivered, notice.Message)
		return nil
	}
	runRequest := cli.RunRequest{
		Args:   []string{"approach", "--reckless", "earth"},
		Notify: notify,
	}
	if err := app.Run(t.Context(), runRequest); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(delivered) != 1 || delivered[0] != "reckless approach enabled" {
		t.Fatalf("delivered notices = %v", delivered)
	}

	quietRequest := cli.InspectionRequest{
		Args: []string{"approach", "--reckless", "--quiet", "earth"},
	}
	clitest.Inspect(t.Context(), app, quietRequest).Require(t,
		clitest.InspectionReady(),
		notice.Suppression(cli.NoticeSuppressionSuppressed),
	)
}
