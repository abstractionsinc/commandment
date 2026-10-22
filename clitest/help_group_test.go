package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestStaticHelpGroupExpectationsObserveSchemaAndReference(t *testing.T) {
	t.Parallel()

	commandGroups := []cli.HelpCommandGroupSchema{{
		Kind:    cli.HelpGroupDeclared,
		Title:   "Commands",
		Order:   cli.HelpGroupOrderName,
		Profile: 7,
		Entries: []cli.HelpCommandGroupEntrySchema{
			{
				Kind:         cli.HelpCommandGroupEntryCommandEntry,
				CommandEntry: 2,
				Profile:      7,
			},
			{
				Kind:     cli.HelpCommandGroupEntryCommandProvider,
				Provider: 3,
				Profile:  7,
			},
		},
	}}
	flagGroups := []cli.HelpFlagGroupSchema{{
		Kind:  cli.HelpGroupDeclared,
		Title: "Options",
		Order: cli.HelpGroupOrderDeclaration,
		Flags: []cli.FlagID{4, 5},
	}}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID: 1,
			Help: cli.HelpSchema{
				CommandGroups: commandGroups,
				FlagGroups:    flagGroups,
			},
		}},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{1: {
			ID: 1,
			Help: cli.HelpSchema{
				CommandGroups: commandGroups,
				FlagGroups:    flagGroups,
			},
		}},
	}

	commandGroup := CommandByID(1).HelpCommandGroup("Commands")
	flagGroup := CommandByID(1).HelpFlagGroupAt(0)
	expectations := []StaticDeclarationExpectation{
		commandGroup.Exists(),
		commandGroup.Kind(cli.HelpGroupDeclared),
		commandGroup.Title("Commands"),
		commandGroup.Order(cli.HelpGroupOrderName),
		commandGroup.Profile(GrammarProfileByID(7)),
		commandGroup.EntriesExactly(
			CommandEntryByID(2),
			CommandByID(1).ProviderAt(0),
		),
		flagGroup.Exists(),
		flagGroup.Kind(cli.HelpGroupDeclared),
		flagGroup.Title("Options"),
		flagGroup.Order(cli.HelpGroupOrderDeclaration),
		flagGroup.NoProfile(),
		flagGroup.FlagsExactly(FlagByID(4), FlagByID(5)),
		CommandByID(1).HelpFlagGroup("Missing").DoesNotExist(),
	}

	// ProviderAt(0) resolves through the command's provider collection. Populate
	// that collection only after constructing the retained public observations.
	root := schema.Commands[1]
	root.CommandProviders = []cli.CommandProviderID{3}
	schema.Commands[1] = root
	refRoot := reference.Commands[1]
	refRoot.CommandProviders = []cli.CommandProviderID{3}
	reference.Commands[1] = refRoot

	for _, expectation := range expectations {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("schema %T = %+v", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference %T = %+v", expectation, got)
		}
	}
}

func TestStaticHelpGroupExpectationsSnapshotEntriesAndRejectAmbiguity(t *testing.T) {
	t.Parallel()

	entries := []HelpCommandGroupEntry{CommandEntryByID(2)}
	flags := []FlagLocator{FlagByID(4)}
	expectation := CommandByID(1).HelpCommandGroup("Commands").EntriesExactly(entries...)
	flagExpectation := CommandByID(1).HelpFlagGroup("Options").FlagsExactly(flags...)
	entries[0] = CommandEntryByID(99)
	flags[0] = FlagByID(99)

	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID: 1,
			Help: cli.HelpSchema{
				CommandGroups: []cli.HelpCommandGroupSchema{
					{
						Kind:  cli.HelpGroupDeclared,
						Title: "Commands",
						Entries: []cli.HelpCommandGroupEntrySchema{
							{
								Kind:         cli.HelpCommandGroupEntryCommandEntry,
								CommandEntry: 2,
							},
						},
					},
				},
				FlagGroups: []cli.HelpFlagGroupSchema{
					{
						Kind:  cli.HelpGroupDeclared,
						Title: "Options",
						Flags: []cli.FlagID{4},
					},
				},
			},
		}},
	}
	if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
		t.Fatalf("entry expectation borrowed caller slice: %+v", got)
	}
	if got := schemaExpectationOutcome(t, flagExpectation, schema); got.status != evaluationProved {
		t.Fatalf("flag-entry expectation borrowed caller slice: %+v", got)
	}

	root := schema.Commands[1]
	group := cli.HelpCommandGroupSchema{
		Kind:  cli.HelpGroupDeclared,
		Title: "Commands",
	}
	root.Help.CommandGroups = append(root.Help.CommandGroups, group)
	schema.Commands[1] = root
	got := schemaExpectationOutcome(t, CommandByID(1).HelpCommandGroup("Commands").Exists(), schema)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "indexes") {
		t.Fatalf("ambiguous help group = %+v", got)
	}
}

func TestStaticHelpGroupMismatchNamesOwningField(t *testing.T) {
	t.Parallel()

	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID: 1,
			Help: cli.HelpSchema{
				FlagGroups: []cli.HelpFlagGroupSchema{{
					Kind:  cli.HelpGroupDeclared,
					Title: "Options",
					Order: cli.HelpGroupOrderDeclaration,
				}},
			},
		}},
	}
	got := schemaExpectationOutcome(t, CommandByID(1).HelpFlagGroup("Options").Order(cli.HelpGroupOrderName), schema)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "order") {
		t.Fatalf("help-group mismatch attribution = %+v", got)
	}
}

func TestStaticHelpGroupExactnessRejectsWrongProfilesEntriesAndIndexes(t *testing.T) {
	t.Parallel()

	groups := []cli.HelpCommandGroupSchema{
		{
			Kind:    cli.HelpGroupDeclared,
			Title:   "Commands",
			Order:   cli.HelpGroupOrderDeclaration,
			Profile: 7,
			Entries: []cli.HelpCommandGroupEntrySchema{
				{
					Kind:         cli.HelpCommandGroupEntryCommandEntry,
					CommandEntry: 2,
					Profile:      7,
				},
			},
		},
	}
	flags := []cli.HelpFlagGroupSchema{
		{
			Kind:    cli.HelpGroupDeclared,
			Title:   "Options",
			Order:   cli.HelpGroupOrderDeclaration,
			Profile: 7,
			Flags:   []cli.FlagID{4},
		},
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID: 1,
			Help: cli.HelpSchema{
				CommandGroups: groups,
				FlagGroups:    flags,
			},
		}},
	}

	checks := []struct {
		name        string
		expectation StaticDeclarationExpectation
		contains    string
	}{
		{
			name:        "command group index absent",
			expectation: CommandByID(1).HelpCommandGroupAt(1).Exists(),
			contains:    "exists=false",
		},
		{
			name:        "flag group index absent",
			expectation: CommandByID(1).HelpFlagGroupAt(1).Exists(),
			contains:    "exists=false",
		},
		{
			name:        "command group no-profile mismatch",
			expectation: CommandByID(1).HelpCommandGroup("Commands").NoProfile(),
			contains:    "profile",
		},
		{
			name:        "command group profile mismatch",
			expectation: CommandByID(1).HelpCommandGroup("Commands").Profile(GrammarProfileByID(8)),
			contains:    "profile",
		},
		{
			name:        "flag group profile mismatch",
			expectation: CommandByID(1).HelpFlagGroup("Options").Profile(GrammarProfileByID(8)),
			contains:    "profile",
		},
		{
			name:        "entry count mismatch",
			expectation: CommandByID(1).HelpCommandGroup("Commands").EntriesExactly(),
			contains:    "entries",
		},
		{
			name:        "entry identity mismatch",
			expectation: CommandByID(1).HelpCommandGroup("Commands").EntriesExactly(CommandEntryByID(3)),
			contains:    "entry",
		},
		{
			name:        "flag identity mismatch",
			expectation: CommandByID(1).HelpFlagGroup("Options").FlagsExactly(FlagByID(5)),
			contains:    "flag",
		},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			t.Parallel()
			got := schemaExpectationOutcome(t, check.expectation, schema)
			if got.status != evaluationDisproved ||
				(!strings.Contains(renderEvaluationField(got.subject), check.contains) && !strings.Contains(renderEvaluationField(got.observed), check.contains)) {
				t.Fatalf("outcome = %+v", got)
			}
		})
	}

	profileCorrupt := *schema
	profileCorrupt.Commands = map[cli.CommandID]cli.CommandSchema{1: schema.Commands[1]}
	root := profileCorrupt.Commands[1]
	root.Help.CommandGroups = append([]cli.HelpCommandGroupSchema(nil), root.Help.CommandGroups...)
	root.Help.CommandGroups[0].Entries = append(
		[]cli.HelpCommandGroupEntrySchema(nil),
		root.Help.CommandGroups[0].Entries...,
	)
	root.Help.CommandGroups[0].Entries[0].Profile = 9
	profileCorrupt.Commands[1] = root
	got := schemaExpectationOutcome(
		t,
		CommandByID(1).HelpCommandGroup("Commands").EntriesExactly(CommandEntryByID(2)),
		&profileCorrupt,
	)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "entry") {
		t.Fatalf("entry profile mismatch = %+v", got)
	}
}

func TestSchemaHelpCommandGroupResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	got := resolveSchemaHelpCommandGroup(RootCommand().HelpCommandGroup("missing").state, &cli.Schema{})
	if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "command is absent" {
		t.Fatalf("resolution = %#v", got)
	}
}

func TestReferenceHelpCommandGroupResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	got := resolveReferenceHelpCommandGroup(RootCommand().HelpCommandGroup("missing").state, cli.Reference{})
	if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "command is absent" {
		t.Fatalf("resolution = %#v", got)
	}
}

func TestHelpCommandGroupPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	state := &helpCommandGroupLocatorState{
		mode:  helpGroupLocatorIndex,
		index: 0,
	}
	groups := []cli.HelpCommandGroupSchema{{
		Title: "commands",
	}}
	got := selectHelpCommandGroup(state, groups, evaluationText("root"))
	if got.status != locatorResolutionFound || got.group.Title != "commands" {
		t.Fatalf("resolution = %#v", got)
	}
}

func TestSchemaHelpFlagGroupResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	got := resolveSchemaHelpFlagGroup(RootCommand().HelpFlagGroup("missing").state, &cli.Schema{})
	if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "command is absent" {
		t.Fatalf("resolution = %#v", got)
	}
}

func TestReferenceHelpFlagGroupResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	got := resolveReferenceHelpFlagGroup(RootCommand().HelpFlagGroup("missing").state, cli.Reference{})
	if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "command is absent" {
		t.Fatalf("resolution = %#v", got)
	}
}

func TestHelpFlagGroupPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	state := &helpFlagGroupLocatorState{
		mode:  helpGroupLocatorTitle,
		title: "flags",
	}
	groups := []cli.HelpFlagGroupSchema{{
		Title: "flags",
	}, {
		Title: "flags",
	}}
	got := selectHelpFlagGroup(state, groups, evaluationText("root"))
	if got.status != locatorResolutionAmbiguous {
		t.Fatalf("resolution = %#v", got)
	}
}

func TestHelpCommandGroupPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		expectation := RootCommand().HelpCommandGroup("missing").Title("wanted")
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationDisproved {
			t.Fatalf("schema outcome = %#v", got)
		}
		if got := referenceExpectationOutcome(
			t,
			expectation,
			cli.Reference{},
		); got.status != evaluationDisproved {
			t.Fatalf("reference outcome = %#v", got)
		}
	})
	t.Run("subject mismatch: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		schema := &cli.Schema{
			Commands: map[cli.CommandID]cli.CommandSchema{1: {
				ID:   1,
				Path: []string{},
				Help: cli.HelpSchema{
					CommandGroups: []cli.HelpCommandGroupSchema{{
						Title: "actual",
					}},
				},
			}},
		}
		reference := cli.Reference{
			Commands: map[cli.CommandID]cli.CommandReference{1: {
				ID:   1,
				Path: []string{},
				Help: cli.HelpSchema{
					CommandGroups: []cli.HelpCommandGroupSchema{{
						Title: "actual",
					}},
				},
			}},
		}
		expectation := RootCommand().HelpCommandGroupAt(0).Title("wanted")
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationDisproved {
			t.Fatalf("schema outcome = %#v", got)
		}
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationDisproved {
			t.Fatalf("reference outcome = %#v", got)
		}
	})
}

func TestHelpFlagGroupPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		expectation := RootCommand().HelpFlagGroup("missing").Title("wanted")
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationDisproved {
			t.Fatalf("schema outcome = %#v", got)
		}
		if got := referenceExpectationOutcome(
			t,
			expectation,
			cli.Reference{},
		); got.status != evaluationDisproved {
			t.Fatalf("reference outcome = %#v", got)
		}
	})
	t.Run("subject mismatch: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		reference := cli.Reference{
			Commands: map[cli.CommandID]cli.CommandReference{1: {
				ID:   1,
				Path: []string{},
				Help: cli.HelpSchema{
					FlagGroups: []cli.HelpFlagGroupSchema{{
						Title: "actual",
					}},
				},
			}},
		}
		expectation := RootCommand().HelpFlagGroupAt(0).Title("wanted")
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationDisproved {
			t.Fatalf("reference outcome = %#v", got)
		}
	})
}

func TestHelpCommandGroupProfileRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		expectation := RootCommand().HelpCommandGroup("missing").Profile(GrammarProfileByID(1))
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationDisproved {
			t.Fatalf("schema outcome = %#v", got)
		}
		if got := referenceExpectationOutcome(
			t,
			expectation,
			cli.Reference{},
		); got.status != evaluationDisproved {
			t.Fatalf("reference outcome = %#v", got)
		}
	})
}

func TestHelpFlagGroupProfileRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		expectation := RootCommand().HelpFlagGroup("missing").Profile(GrammarProfileByID(1))
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationDisproved {
			t.Fatalf("schema outcome = %#v", got)
		}
		if got := referenceExpectationOutcome(
			t,
			expectation,
			cli.Reference{},
		); got.status != evaluationDisproved {
			t.Fatalf("reference outcome = %#v", got)
		}
	})
}

func TestHelpGroupProfileSchemaPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	got := compareHelpGroupProfileSchema(evaluationText("group"), 1, nil, false, &cli.Schema{})
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestHelpGroupProfileReferencePreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	t.Run("profile: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		if got := compareHelpGroupProfileReference(
			evaluationText("group"),
			1,
			nil,
			true,
			cli.Reference{},
		); got.status != evaluationDisproved {
			t.Fatalf("none mismatch = %#v", got)
		}
		if got := compareHelpGroupProfileReference(
			evaluationText("group"),
			1,
			GrammarProfileByID(2).state,
			false,
			cli.Reference{},
		); got.status != evaluationDisproved {
			t.Fatalf("profile mismatch = %#v", got)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		got := compareHelpGroupProfileReference(evaluationText("group"), 1, nil, false, cli.Reference{})
		if got.status != evaluationDisproved {
			t.Fatalf("outcome = %#v", got)
		}
	})
}

func TestHelpCommandGroupEntriesRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		expectation := RootCommand().HelpCommandGroup("missing").EntriesExactly()
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationDisproved {
			t.Fatalf("schema outcome = %#v", got)
		}
		if got := referenceExpectationOutcome(
			t,
			expectation,
			cli.Reference{},
		); got.status != evaluationDisproved {
			t.Fatalf("reference outcome = %#v", got)
		}
	})
}

func TestHelpCommandGroupEntriesSchemaPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	entries := []cli.HelpCommandGroupEntrySchema{{}}
	members := []*helpCommandGroupEntryState{nil}
	got := compareHelpCommandGroupEntriesSchema(evaluationText("group"), 0, entries, members, &cli.Schema{})
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestHelpCommandGroupEntriesReferencePreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	t.Run("entries: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		members := []*helpCommandGroupEntryState{{}}
		got := compareHelpCommandGroupEntriesReference(evaluationText("group"), 0, nil, members, cli.Reference{})
		if got.status != evaluationDisproved {
			t.Fatalf("outcome = %#v", got)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		entries := []cli.HelpCommandGroupEntrySchema{{}}
		members := []*helpCommandGroupEntryState{nil}
		got := compareHelpCommandGroupEntriesReference(evaluationText("group"), 0, entries, members, cli.Reference{})
		if got.status != evaluationDisproved {
			t.Fatalf("outcome = %#v", got)
		}
	})
	t.Run("expected subject unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expected := snapshotHelpCommandGroupEntry(CommandEntryByID(2))
		actual := []cli.HelpCommandGroupEntrySchema{{
			Kind:         cli.HelpCommandGroupEntryCommandEntry,
			CommandEntry: 1,
		}}
		got := compareHelpCommandGroupEntriesReference(
			evaluationText("group"),
			0,
			actual,
			[]*helpCommandGroupEntryState{expected},
			cli.Reference{},
		)
		if got.status != evaluationDisproved {
			t.Fatalf("outcome = %#v", got)
		}
	})
}

func TestHelpCommandGroupEntrySchemaResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	_, status, _ := resolveHelpCommandGroupEntrySchema(&helpCommandGroupEntryState{
		kind: helpCommandGroupEntryEntry,
	}, &cli.Schema{})
	if status != locatorResolutionInvalid {
		t.Fatalf("status = %v", status)
	}
}

func TestHelpCommandGroupEntryReferenceResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	_, status, _ := resolveHelpCommandGroupEntryReference(&helpCommandGroupEntryState{
		kind: helpCommandGroupEntryEntry,
	}, cli.Reference{})
	if status != locatorResolutionInvalid {
		t.Fatalf("status = %v", status)
	}
}

func TestHelpFlagGroupFlagsRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		missing := RootCommand().HelpFlagGroup("missing").FlagsExactly()
		if got := schemaExpectationOutcome(t, missing, &cli.Schema{}); got.status != evaluationDisproved {
			t.Fatalf("missing schema = %#v", got)
		}
		if got := referenceExpectationOutcome(t, missing, cli.Reference{}); got.status != evaluationDisproved {
			t.Fatalf("missing reference = %#v", got)
		}

		schema := &cli.Schema{
			Commands: map[cli.CommandID]cli.CommandSchema{1: {
				ID:   1,
				Path: []string{},
				Help: cli.HelpSchema{
					FlagGroups: []cli.HelpFlagGroupSchema{{
						Title: "flags",
					}},
				},
			}},
		}
		reference := cli.Reference{
			Commands: map[cli.CommandID]cli.CommandReference{1: {
				ID:   1,
				Path: []string{},
				Help: cli.HelpSchema{
					FlagGroups: []cli.HelpFlagGroupSchema{{
						Title: "flags",
					}},
				},
			}},
		}
		invalid := RootCommand().HelpFlagGroupAt(0).FlagsExactly(FlagLocator{})
		if got := schemaExpectationOutcome(t, invalid, schema); got.status != evaluationDisproved {
			t.Fatalf("invalid schema flag = %#v", got)
		}
		if got := referenceExpectationOutcome(t, invalid, reference); got.status != evaluationDisproved {
			t.Fatalf("invalid reference flag = %#v", got)
		}
	})
}

func TestHelpFlagIDsSchemaResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	_, status, _ := resolveHelpFlagIDsSchema([]*flagLocatorState{nil}, &cli.Schema{})
	if status != locatorResolutionInvalid {
		t.Fatalf("status = %v", status)
	}
}

func TestHelpFlagIDsReferenceResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	_, status, _ := resolveHelpFlagIDsReference([]*flagLocatorState{nil}, cli.Reference{})
	if status != locatorResolutionInvalid {
		t.Fatalf("status = %v", status)
	}
}

func TestHelpFlagIDsPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	got := compareHelpFlagIDs(evaluationText("group"), []cli.FlagID{1}, nil)
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestHelpGroupInternalStateFailsClosed(t *testing.T) {
	t.Parallel()
	wrongEntry := &commandEntryLocatorState{
		mode: commandEntryLocatorMode(255),
	}

	if got := resolveSchemaHelpCommandGroup(nil, &cli.Schema{}); got.status != locatorResolutionInvalid {
		t.Fatalf("schema command group = %#v", got)
	}
	if got := resolveReferenceHelpCommandGroup(nil, cli.Reference{}); got.status != locatorResolutionInvalid {
		t.Fatalf("reference command group = %#v", got)
	}
	if got := selectHelpCommandGroup(&helpCommandGroupLocatorState{
		mode: helpGroupLocatorInvalid,
	}, nil, evaluationText("command")); got.status != locatorResolutionInvalid {
		t.Fatalf("command selector = %#v", got)
	}
	if got := resolveSchemaHelpFlagGroup(nil, &cli.Schema{}); got.status != locatorResolutionInvalid {
		t.Fatalf("schema flag group = %#v", got)
	}
	if got := resolveReferenceHelpFlagGroup(nil, cli.Reference{}); got.status != locatorResolutionInvalid {
		t.Fatalf("reference flag group = %#v", got)
	}
	if got := selectHelpFlagGroup(&helpFlagGroupLocatorState{
		mode: helpGroupLocatorInvalid,
	}, nil, evaluationText("command")); got.status != locatorResolutionInvalid {
		t.Fatalf("flag selector = %#v", got)
	}

	for _, test := range []struct {
		name  string
		state *helpCommandGroupEntryState
	}{
		{
			name:  "nil",
			state: nil,
		},
		{
			name: "entry invalid locator",
			state: &helpCommandGroupEntryState{
				kind:  helpCommandGroupEntryEntry,
				entry: wrongEntry,
			},
		},
		{
			name: "provider invalid locator",
			state: &helpCommandGroupEntryState{
				kind:     helpCommandGroupEntryProvider,
				provider: &commandProviderLocatorState{},
			},
		},
		{
			name: "unknown entry kind",
			state: &helpCommandGroupEntryState{
				kind:  helpCommandGroupEntryInvalid,
				entry: wrongEntry,
			},
		},
	} {
		name := test.name
		state := test.state
		t.Run("schema entry "+name, func(t *testing.T) {
			t.Parallel()
			if _, status, _ := resolveHelpCommandGroupEntrySchema(
				state,
				&cli.Schema{},
			); status != locatorResolutionInvalid {
				t.Fatalf("status = %v, want invalid", status)
			}
		})
		t.Run("reference entry "+name, func(t *testing.T) {
			t.Parallel()
			if _, status, _ := resolveHelpCommandGroupEntryReference(
				state,
				cli.Reference{},
			); status != locatorResolutionInvalid {
				t.Fatalf("status = %v, want invalid", status)
			}
		})
	}

	invalidFlags := []*flagLocatorState{{
		mode: flagLocatorMode(255),
	}}
	if _, status, _ := resolveHelpFlagIDsSchema(invalidFlags, &cli.Schema{}); status != locatorResolutionInvalid {
		t.Fatalf("schema flags = %v", status)
	}
	if _, status, _ := resolveHelpFlagIDsReference(invalidFlags, cli.Reference{}); status != locatorResolutionInvalid {
		t.Fatalf("reference flags = %v", status)
	}
	if cloneHelpCommandGroupLocatorState(nil) != nil {
		t.Fatalf("nil command group clone became nonnil")
	}
	if cloneHelpFlagGroupLocatorState(nil) != nil {
		t.Fatalf("nil flag group clone became nonnil")
	}
}

func TestHelpCommandGroupAbsenceIsPreservedAcrossStaticObservations(t *testing.T) {
	t.Parallel()

	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID: 1,
		}},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{1: {
			ID: 1,
		}},
	}
	expectation := CommandByID(1).HelpCommandGroup("missing").DoesNotExist()
	if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
		t.Fatalf("schema missing help group = %+v", got)
	}
	if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
		t.Fatalf("reference missing help group = %+v", got)
	}
}

func TestNonemptyPreservesResolverDetailAndSuppliesTruncationFallback(t *testing.T) {
	t.Parallel()

	if got := nonempty(evaluationText("resolver detail"), evaluationText("inspection is truncated")); got != evaluationText("resolver detail") {
		t.Fatalf("nonempty detail = %q", got)
	}
	if got := nonempty(evaluationText(""), evaluationText("inspection is truncated")); got != evaluationText("inspection is truncated") {
		t.Fatalf("empty detail fallback = %q", got)
	}
}
