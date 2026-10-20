package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/clitest"
)

func TestRangeFlagHasShortAndLongSpellings(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, definition)
	flag := clitest.Command("scan").Flag("--range")
	clitest.Schema(app).Require(t,
		flag.SpellingsExactly(flag.Spelling("--range"), flag.Spelling("-r")),
	)
}
