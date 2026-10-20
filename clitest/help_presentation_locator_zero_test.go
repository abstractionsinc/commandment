package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func helpLocatorZeroFixture() cli.HelpPage {
	const (
		spellingID cli.FlagSpellingID     = 1
		entryID    cli.ValueDomainEntryID = 2
	)
	use := cli.HelpValueDomainUse{
		Kind:    cli.ValueDomainUseDestination,
		Entries: []cli.ValueDomainEntryID{entryID},
	}
	page := cli.HelpPage{
		CommandGroups: []cli.HelpCommandGroup{{
			Title:    "",
			Commands: []cli.HelpCommand{{Name: ""}},
		}},
		ArgumentForms: []cli.HelpArgumentForm{
			{Terms: []cli.HelpArgumentTerm{{ValueDomainUses: []cli.HelpValueDomainUse{use}}}},
		},
		FlagGroups: []cli.HelpFlagGroup{
			{
				Title: "",
				Flags: []cli.HelpFlag{
					{
						FlagSpellings:   []cli.FlagSpellingID{spellingID},
						ValueDomainUses: []cli.HelpValueDomainUse{use},
					},
				},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.HelpFlagSpelling{
			spellingID: {
				ID: spellingID,
				Spelling: cli.FlagSpelling{
					Text: "",
				},
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.HelpValueDomainEntry{
			entryID: {
				ID:   entryID,
				Name: "",
			},
		},
	}
	return page
}

func TestHelpPageLocatorZeroValuesAreInvalid(t *testing.T) {
	t.Parallel()
	page := helpLocatorZeroFixture()
	cases := []struct {
		name    string
		resolve func(cli.HelpPage) locatorResolutionStatus
	}{
		{"command group", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommandGroup(HelpPageCommandGroupLocator{}, page)
			return s
		}},
		{"command", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommand(HelpPageCommandLocator{}, page)
			return s
		}},
		{"argument form", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpArgumentForm(HelpPageArgumentFormLocator{}, page)
			return s
		}},
		{"argument term", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpArgumentTerm(HelpPageArgumentTermLocator{}, page)
			return s
		}},
		{"flag group", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagGroup(HelpPageFlagGroupLocator{}, page)
			return s
		}},
		{"flag", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlag(HelpPageFlagLocator{}, page)
			return s
		}},
		{"flag spelling", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagSpelling(HelpPageFlagSpellingLocator{}, page)
			return s
		}},
		{"value-domain entry", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(HelpPageValueDomainEntryLocator{}, page)
			return s
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.resolve(page); got != locatorResolutionInvalid {
				t.Fatalf("zero locator resolution = %v, want invalid", got)
			}
		})
	}
}

func TestHelpPageLocatorExplicitZeroCoordinatesRemainValid(t *testing.T) {
	t.Parallel()
	page := helpLocatorZeroFixture()
	group := HelpCommandGroupAt(0)
	form := HelpArgumentFormAt(0)
	flagGroup := HelpFlagGroupAt(0)
	flag := flagGroup.FlagAt(0)
	term := form.TermAt(0)
	cases := []struct {
		name    string
		resolve func(cli.HelpPage) locatorResolutionStatus
	}{
		{"command group", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommandGroup(group, page)
			return s
		}},
		{"argument form", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpArgumentForm(form, page)
			return s
		}},
		{"flag group", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagGroup(flagGroup, page)
			return s
		}},
		{"command at", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommand(group.CommandAt(0), page)
			return s
		}},
		{"term at", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpArgumentTerm(term, page)
			return s
		}},
		{
			"flag at",
			func(page cli.HelpPage) locatorResolutionStatus { _, s, _ := resolveHelpFlag(flag, page); return s },
		},
		{"spelling at", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagSpelling(flag.SpellingAt(0), page)
			return s
		}},
		{"term domain entry at", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(term.ValueDomainEntryAt(cli.ValueDomainUseDestination, 0), page)
			return s
		}},
		{"flag domain entry at", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(flag.ValueDomainEntryAt(cli.ValueDomainUseDestination, 0), page)
			return s
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.resolve(page); got != locatorResolutionFound {
				t.Fatalf("explicit zero coordinate resolution = %v, want found", got)
			}
		})
	}
	for _, got := range []locatorResolutionStatus{
		func() locatorResolutionStatus {
			_, s, _ := resolveHelpCommandGroup(HelpCommandGroupAt(-1), page)
			return s
		}(),
		func() locatorResolutionStatus {
			_, s, _ := resolveHelpArgumentForm(HelpArgumentFormAt(-1), page)
			return s
		}(),
		func() locatorResolutionStatus { _, s, _ := resolveHelpFlagGroup(HelpFlagGroupAt(-1), page); return s }(),
	} {
		if got != locatorResolutionAbsent {
			t.Fatalf("explicit negative coordinate resolution = %v, want absent", got)
		}
	}
}

func TestHelpPageLocatorExplicitEmptyTextRemainsTextSelection(t *testing.T) {
	t.Parallel()
	page := helpLocatorZeroFixture()
	cases := []struct {
		name    string
		resolve func(cli.HelpPage) locatorResolutionStatus
	}{
		{"command group", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommandGroup(HelpCommandGroup(""), page)
			return s
		}},
		{"command", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommand(HelpCommand(""), page)
			return s
		}},
		{"flag group", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagGroup(HelpFlagGroup(""), page)
			return s
		}},
		{"flag spelling", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagSpelling(HelpFlagSpelling(""), page)
			return s
		}},
		{"value-domain entry", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(HelpValueDomainEntry(""), page)
			return s
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.resolve(page); got != locatorResolutionFound {
				t.Fatalf("explicit empty-text resolution = %v, want found", got)
			}
		})
	}
}

func TestHelpPageLocatorInvalidParentsCannotBeResurrected(t *testing.T) {
	t.Parallel()
	page := helpLocatorZeroFixture()
	cases := []struct {
		name    string
		resolve func(cli.HelpPage) locatorResolutionStatus
	}{
		{"command", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommand(HelpPageCommandGroupLocator{}.Command("x"), page)
			return s
		}},
		{"command at", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpCommand(HelpPageCommandGroupLocator{}.CommandAt(0), page)
			return s
		}},
		{"term", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpArgumentTerm(HelpPageArgumentFormLocator{}.TermAt(0), page)
			return s
		}},
		{"flag", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlag(HelpPageFlagGroupLocator{}.Flag("--x"), page)
			return s
		}},
		{"action flag", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlag(HelpPageFlagGroupLocator{}.ActionFlag("--x"), page)
			return s
		}},
		{"flag at", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlag(HelpPageFlagGroupLocator{}.FlagAt(0), page)
			return s
		}},
		{"spelling", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagSpelling(HelpPageFlagLocator{}.SpellingAt(0), page)
			return s
		}},
		{"term domain", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(
				HelpPageArgumentTermLocator{}.ValueDomainEntryAt(cli.ValueDomainUseDestination, 0),
				page,
			)
			return s
		}},
		{"flag domain", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(
				HelpPageFlagLocator{}.ValueDomainEntryAt(cli.ValueDomainUseDestination, 0),
				page,
			)
			return s
		}},
		{"spelling replacement", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpFlagSpelling(HelpPageFlagSpellingLocator{}.Replacement(), page)
			return s
		}},
		{"domain canonical", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(HelpPageValueDomainEntryLocator{}.Canonical(), page)
			return s
		}},
		{"domain replacement", func(page cli.HelpPage) locatorResolutionStatus {
			_, s, _ := resolveHelpValueDomainEntry(HelpPageValueDomainEntryLocator{}.Replacement(), page)
			return s
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.resolve(page); got != locatorResolutionInvalid {
				t.Fatalf("invalid-parent resolution = %v, want invalid", got)
			}
		})
	}
}

func TestHelpPageLocatorInvalidDoesNotCheckAbsence(t *testing.T) {
	t.Parallel()
	page := helpLocatorZeroFixture()
	for _, expectation := range []HelpPresentationExpectation{
		HelpPageCommandGroupLocator{}.Exists(),
		HelpPageCommandGroupLocator{}.DoesNotExist(),
	} {
		checkedOutcome, ok := expectation.(interface {
			evaluateHelp(page cli.HelpPage) evaluationResult
		})
		if !ok {
			t.Fatal("expectation must satisfy expected evaluator interface")
		}
		outcome := checkedOutcome.evaluateHelp(page)
		if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.observed), "invalid") {
			t.Fatalf("invalid locator expectation = %#v, want disproved invalid", outcome)
		}
	}
}
