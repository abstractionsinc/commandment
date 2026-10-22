package clitest

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

func TestHelpPresentationLocatorsPreserveExactSelectionAndRelations(t *testing.T) {
	t.Parallel()

	page := cli.HelpPage{
		Usage: []string{"app", "app child"},
		CommandGroups: []cli.HelpCommandGroup{
			{
				Title: "commands",
				Commands: []cli.HelpCommand{{
					Name:        "child",
					ActionEntry: 7,
				}},
			},
		},
		FlagGroups: []cli.HelpFlagGroup{
			{
				Title: "flags",
				Flags: []cli.HelpFlag{{
					ActionEntry: 9,
					ActionSpelling: cli.FlagSpelling{
						Text: "--act",
					},
				}},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.HelpFlagSpelling{
			1: {
				ID: 1,
				Spelling: cli.FlagSpelling{
					Text: "--old",
				},
				Replacement: 2,
			},
			2: {
				ID: 2,
				Spelling: cli.FlagSpelling{
					Text: "--new",
				},
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.HelpValueDomainEntry{
			1: {
				ID:          1,
				Name:        "alias",
				Canonical:   2,
				Replacement: 3,
			},
			2: {
				ID:   2,
				Name: "canonical",
			},
			3: {
				ID:   3,
				Name: "replacement",
			},
		},
	}

	cases := []struct {
		name        string
		expectation HelpPresentationExpectation
		status      evaluationStatus
	}{
		{
			name:        "ordered usage",
			expectation: HelpUsageExactly("app", "app child"),
			status:      evaluationProved,
		},
		{
			name:        "out of range command",
			expectation: HelpCommandGroup("commands").CommandAt(1).Exists(),
			status:      evaluationDisproved,
		},
		{
			name: "flag replacement edge",
			expectation: HelpFlagSpelling("--old").Replacement().Spelling(cli.FlagSpelling{
				Text: "--new",
			}),
			status: evaluationProved,
		},
		{
			name:        "domain canonical edge",
			expectation: HelpValueDomainEntry("alias").Canonical().Name("canonical"),
			status:      evaluationProved,
		},
		{
			name:        "domain replacement edge",
			expectation: HelpValueDomainEntry("alias").Replacement().Name("replacement"),
			status:      evaluationProved,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			evaluator, ok := tc.expectation.(interface {
				evaluateHelp(page cli.HelpPage) evaluationResult
			})
			if !ok {
				t.Fatalf("expectation %T does not expose help evaluation", tc.expectation)
			}
			if outcome := evaluator.evaluateHelp(page); outcome.status != tc.status {
				t.Fatalf("status = %v, want %v: %+v", outcome.status, tc.status, outcome)
			}
		})
	}

	ambiguous := page
	ambiguous.CommandGroups = append(append([]cli.HelpCommandGroup(nil), page.CommandGroups...), cli.HelpCommandGroup{
		Title: "commands",
	})
	expectation, ok := HelpCommandGroup("commands").Exists().(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("HelpCommandGroup Exists expectation does not expose help evaluation")
	}
	outcome := expectation.evaluateHelp(ambiguous)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.observed), "ambiguous") {
		t.Fatalf("ambiguous selector outcome = %+v", outcome)
	}
}

func TestHelpPresentationActionEntryRequiresExactBoundIdentity(t *testing.T) {
	t.Parallel()

	page := cli.HelpPage{
		CommandGroups: []cli.HelpCommandGroup{{Commands: []cli.HelpCommand{{
			Name:        "run",
			ActionEntry: 7,
		}}}},
		FlagGroups: []cli.HelpFlagGroup{
			{
				Title: "flags",
				Flags: []cli.HelpFlag{{
					ActionEntry: 9,
					ActionSpelling: cli.FlagSpelling{
						Text: "--act",
					},
				}},
			},
		},
	}

	command := HelpCommand("run")
	goodExpectation, ok := command.ActionEntry(ActionEntryByID(7)).(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("ActionEntry expectation does not expose help evaluation")
	}
	good := goodExpectation.evaluateHelp(page)
	if good.status != evaluationProved {
		t.Fatalf("matching command ActionEntry = %+v", good)
	}
	wrongExpectation, ok := command.ActionEntry(ActionEntryByID(8)).(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("ActionEntry expectation does not expose help evaluation")
	}
	wrong := wrongExpectation.evaluateHelp(page)
	if wrong.status != evaluationDisproved {
		t.Fatalf("wrong nonzero command ActionEntry = %+v", wrong)
	}

	flag := HelpFlagGroup("flags").ActionFlag("--act")
	goodFlagExpectation, ok := flag.ActionEntry(ActionEntryByID(9)).(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("ActionEntry expectation does not expose help evaluation")
	}
	good = goodFlagExpectation.evaluateHelp(page)
	if good.status != evaluationProved {
		t.Fatalf("matching flag ActionEntry = %+v", good)
	}
	wrongFlagExpectation, ok := flag.ActionEntry(ActionEntryByID(10)).(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("ActionEntry expectation does not expose help evaluation")
	}
	wrong = wrongFlagExpectation.evaluateHelp(page)
	if wrong.status != evaluationDisproved {
		t.Fatalf("wrong nonzero flag ActionEntry = %+v", wrong)
	}

	unresolvedExpectation, ok := command.ActionEntry(RootCommand().ActionEntryAt(0)).(interface {
		evaluateHelp(page cli.HelpPage) evaluationResult
	})
	if !ok {
		t.Fatal("ActionEntry expectation does not expose help evaluation")
	}
	unresolved := unresolvedExpectation.evaluateHelp(page)
	if unresolved.status != evaluationDisproved || !strings.Contains(renderEvaluationField(unresolved.observed), "exact-identity-bound") {
		t.Fatalf("unresolved expected identity = %+v", unresolved)
	}
}

func TestIdentityRelationFailuresDoNotExposeRawIDs(t *testing.T) {
	t.Parallel()

	const (
		rootID                   cli.CommandID          = 701000001
		actualCommandActionID    cli.ActionEntryID      = 701000002
		expectedCommandActionID  cli.ActionEntryID      = 701000003
		actualFlagActionID       cli.ActionEntryID      = 701000004
		expectedFlagActionID     cli.ActionEntryID      = 701000005
		flagSpellingID           cli.FlagSpellingID     = 701000006
		flagReplacementID        cli.FlagSpellingID     = 701000007
		domainEntryID            cli.ValueDomainEntryID = 701000008
		domainCanonicalID        cli.ValueDomainEntryID = 701000009
		domainReplacementID      cli.ValueDomainEntryID = 701000010
		actualConditionValueID   cli.ValueID            = 702000001
		expectedConditionValueID cli.ValueID            = 702000002
		policyID                 cli.SignalPolicyID     = 703000001
		actualEndpointID         cli.SignalEndpointID   = 703000002
		expectedEndpointID       cli.SignalEndpointID   = 703000003
		actualSignalValueID      cli.ValueID            = 703000004
		expectedSignalValueID    cli.ValueID            = 703000005
		actualDelayValueID       cli.ValueID            = 703000006
		expectedDelayValueID     cli.ValueID            = 703000007
		shortcutEntryID          cli.CommandEntryID     = 705000001
		actualShortcutValueID    cli.ValueID            = 705000002
		expectedShortcutValueID  cli.ValueID            = 705000003
		argumentFormID           cli.ArgumentFormID     = 706000001
		actualFormProfileID      cli.GrammarProfileID   = 706000002
		expectedFormProfileID    cli.GrammarProfileID   = 706000003
		actualDefaultProfileID   cli.GrammarProfileID   = 706000004
		expectedDefaultProfileID cli.GrammarProfileID   = 706000005
		actualCaseProfileID      cli.GrammarProfileID   = 706000006
		expectedCaseProfileID    cli.GrammarProfileID   = 706000007
		overflowEndpointID       cli.SignalEndpointID   = 707000001
	)

	help := cli.HelpPage{
		CommandGroups: []cli.HelpCommandGroup{
			{Commands: []cli.HelpCommand{{
				Name:        "run",
				ActionEntry: actualCommandActionID,
			}}},
		},
		FlagGroups: []cli.HelpFlagGroup{
			{
				Title: "flags",
				Flags: []cli.HelpFlag{{
					ActionEntry: actualFlagActionID,
					ActionSpelling: cli.FlagSpelling{
						Text: "--act",
					},
				}},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.HelpFlagSpelling{
			flagSpellingID: {
				ID: flagSpellingID,
				Spelling: cli.FlagSpelling{
					Text: "--old",
				},
				Replacement: flagReplacementID,
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.HelpValueDomainEntry{
			domainEntryID: {
				ID:          domainEntryID,
				Name:        "alias",
				Canonical:   domainCanonicalID,
				Replacement: domainReplacementID,
			},
		},
	}

	schema := cli.Schema{
		Root: rootID,
		Commands: map[cli.CommandID]cli.CommandSchema{
			rootID: {
				ID: rootID,
				Conditions: []cli.ConditionSchema{{Expression: cli.ConditionExpression{
					Destination: actualConditionValueID,
				}}},
				Profiles: &cli.GrammarProfileSet{
					Default: actualDefaultProfileID,
					Cases:   []cli.GrammarProfileID{actualCaseProfileID},
				},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{
			shortcutEntryID: {
				ID: shortcutEntryID,
				Defaults: []cli.CommandShortcutDefaultSchema{{
					Destination: actualShortcutValueID,
				}},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{
			argumentFormID: {
				ID:      argumentFormID,
				Profile: actualFormProfileID,
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{
			policyID: {
				ID: policyID,
				Actions: []cli.SignalAction{{
					EndpointID:    actualEndpointID,
					SignalValueID: actualSignalValueID,
					DelayValueID:  actualDelayValueID,
				}},
			},
		},
	}

	helpOutcome := func(expectation HelpPresentationExpectation) evaluationResult {
		evaluator, ok := expectation.(interface {
			evaluateHelp(page cli.HelpPage) evaluationResult
		})
		if !ok {
			t.Fatalf("expectation %T does not expose help evaluation", expectation)
		}
		return evaluator.evaluateHelp(help)
	}

	overflowExpectation, ok := DiagnosticAt(0).SignalOverflow().NoEndpoint().(diagnosticEvaluator)
	if !ok {
		t.Fatal("SignalOverflow NoEndpoint expectation must satisfy diagnosticEvaluator")
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{SignalOverflow: &cli.SignalOverflowFailure{
			EndpointID: overflowEndpointID,
		}}},
	}
	overflowOutcome := overflowExpectation.evaluateDiagnosticReport(report)

	cases := []struct {
		name      string
		outcome   evaluationResult
		semantic  string
		forbidden []string
	}{
		{
			name:     "help command action entry",
			outcome:  helpOutcome(HelpCommand("run").ActionEntry(ActionEntryByID(expectedCommandActionID))),
			semantic: "different identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualCommandActionID), 10),
				strconv.FormatUint(uint64(expectedCommandActionID), 10),
			},
		},
		{
			name: "help flag action entry",
			outcome: helpOutcome(
				HelpFlagGroup("flags").ActionFlag("--act").ActionEntry(ActionEntryByID(expectedFlagActionID)),
			),
			semantic: "different identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualFlagActionID), 10),
				strconv.FormatUint(uint64(expectedFlagActionID), 10),
			},
		},
		{
			name:      "help flag spelling replacement",
			outcome:   helpOutcome(HelpFlagSpelling("--old").NotReplaced()),
			semantic:  "replacement relation is present",
			forbidden: []string{strconv.FormatUint(uint64(flagReplacementID), 10)},
		},
		{
			name:      "help value-domain canonical",
			outcome:   helpOutcome(HelpValueDomainEntry("alias").NoCanonical()),
			semantic:  "canonical relation is present",
			forbidden: []string{strconv.FormatUint(uint64(domainCanonicalID), 10)},
		},
		{
			name:      "help value-domain replacement",
			outcome:   helpOutcome(HelpValueDomainEntry("alias").NotReplaced()),
			semantic:  "replacement relation is present",
			forbidden: []string{strconv.FormatUint(uint64(domainReplacementID), 10)},
		},
		{
			name: "condition destination",
			outcome: schemaExpectationOutcome(
				t,
				RootCommand().ConditionAt(0).Expression().Destination(ValueByID(expectedConditionValueID)),
				&schema,
			),
			semantic: "different value identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualConditionValueID), 10),
				strconv.FormatUint(uint64(expectedConditionValueID), 10),
			},
		},
		{
			name: "signal action endpoint",
			outcome: schemaExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).Endpoint(SignalEndpointByID(expectedEndpointID)),
				&schema,
			),
			semantic: "different signal-endpoint identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualEndpointID), 10),
				strconv.FormatUint(uint64(expectedEndpointID), 10),
			},
		},
		{
			name: "signal action signal value",
			outcome: schemaExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).SignalValue(ValueByID(expectedSignalValueID)),
				&schema,
			),
			semantic: "different value identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualSignalValueID), 10),
				strconv.FormatUint(uint64(expectedSignalValueID), 10),
			},
		},
		{
			name: "signal action delay value",
			outcome: schemaExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).DelayValue(ValueByID(expectedDelayValueID)),
				&schema,
			),
			semantic: "different value identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualDelayValueID), 10),
				strconv.FormatUint(uint64(expectedDelayValueID), 10),
			},
		},
		{
			name: "shortcut destination",
			outcome: schemaExpectationOutcome(
				t,
				CommandEntryByID(shortcutEntryID).ShortcutDefaultAt(0).Destination(ValueByID(expectedShortcutValueID)),
				&schema,
			),
			semantic: "different value identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualShortcutValueID), 10),
				strconv.FormatUint(uint64(expectedShortcutValueID), 10),
			},
		},
		{
			name: "argument-form grammar profile",
			outcome: schemaExpectationOutcome(
				t,
				ArgumentFormByID(argumentFormID).GrammarProfile(GrammarProfileByID(expectedFormProfileID)),
				&schema,
			),
			semantic: "different identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualFormProfileID), 10),
				strconv.FormatUint(uint64(expectedFormProfileID), 10),
			},
		},
		{
			name: "command default grammar profile",
			outcome: schemaExpectationOutcome(
				t,
				CommandByID(rootID).DefaultGrammarProfile(GrammarProfileByID(expectedDefaultProfileID)),
				&schema,
			),
			semantic: "different identity",
			forbidden: []string{
				strconv.FormatUint(uint64(actualDefaultProfileID), 10),
				strconv.FormatUint(uint64(expectedDefaultProfileID), 10),
			},
		},
		{
			name: "command grammar-profile sequence",
			outcome: schemaExpectationOutcome(
				t,
				CommandByID(rootID).GrammarProfilesExactly(GrammarProfileByID(expectedCaseProfileID)),
				&schema,
			),
			semantic: "different identity sequence",
			forbidden: []string{
				strconv.FormatUint(uint64(actualCaseProfileID), 10),
				strconv.FormatUint(uint64(expectedCaseProfileID), 10),
			},
		},
		{
			name:      "diagnostic overflow endpoint absence",
			outcome:   overflowOutcome,
			semantic:  "endpoint relation is present",
			forbidden: []string{strconv.FormatUint(uint64(overflowEndpointID), 10)},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.outcome.status != evaluationDisproved {
				t.Fatalf("identity relation mismatch = %+v", tc.outcome)
			}
			serialized := renderEvaluationField(tc.outcome.subject) + " " + renderEvaluationField(tc.outcome.expected) + " " + renderEvaluationField(tc.outcome.observed)
			if !strings.Contains(serialized, tc.semantic) {
				t.Fatalf("failure output lost semantic mismatch wording %q: %+v", tc.semantic, tc.outcome)
			}
			for _, rawID := range tc.forbidden {
				if strings.Contains(serialized, rawID) {
					t.Fatalf("failure output exposed raw ID %q: %+v", rawID, tc.outcome)
				}
			}
		})
	}
}

func TestRelationFailuresRenderAcrossCarriers(t *testing.T) {
	t.Parallel()

	const (
		rootID cli.CommandID = 711000001

		ambiguousFlagA     cli.FlagID         = 711000010
		ambiguousFlagB     cli.FlagID         = 711000011
		ambiguousSpellingA cli.FlagSpellingID = 711000012
		ambiguousSpellingB cli.FlagSpellingID = 711000013
		ambiguousValueA    cli.ValueID        = 711000014
		ambiguousValueB    cli.ValueID        = 711000015

		actualConditionValue   cli.ValueID          = 712000001
		expectedConditionValue cli.ValueID          = 712000002
		policyID               cli.SignalPolicyID   = 713000001
		actualEndpoint         cli.SignalEndpointID = 713000002
		expectedEndpoint       cli.SignalEndpointID = 713000003
		actualSignalValue      cli.ValueID          = 713000004
		expectedSignalValue    cli.ValueID          = 713000005
		actualDelayValue       cli.ValueID          = 713000006
		expectedDelayValue     cli.ValueID          = 713000007
		shortcutEntryID        cli.CommandEntryID   = 714000001
		actualShortcutValue    cli.ValueID          = 714000002
		expectedShortcutValue  cli.ValueID          = 714000003
		argumentFormID         cli.ArgumentFormID   = 715000001
		actualFormProfile      cli.GrammarProfileID = 715000002
		expectedFormProfile    cli.GrammarProfileID = 715000003
		actualDefaultProfile   cli.GrammarProfileID = 715000004
		expectedDefaultProfile cli.GrammarProfileID = 715000005
		caseProfileA           cli.GrammarProfileID = 715000006
		caseProfileB           cli.GrammarProfileID = 715000007
		ambiguousProfileA      cli.GrammarProfileID = 715000008
		ambiguousProfileB      cli.GrammarProfileID = 715000009
	)

	profiles := &cli.GrammarProfileSet{
		Default: actualDefaultProfile,
		Cases:   []cli.GrammarProfileID{caseProfileA, caseProfileB},
		Profiles: map[cli.GrammarProfileID]cli.GrammarProfile{
			actualFormProfile: {
				ID:    actualFormProfile,
				Value: "form",
			},
			actualDefaultProfile: {
				ID:    actualDefaultProfile,
				Value: "default",
			},
			caseProfileA: {
				ID:    caseProfileA,
				Value: "case-a",
			},
			caseProfileB: {
				ID:    caseProfileB,
				Value: "case-b",
			},
			ambiguousProfileA: {
				ID:    ambiguousProfileA,
				Value: "dup",
			},
			ambiguousProfileB: {
				ID:    ambiguousProfileB,
				Value: "dup",
			},
		},
	}

	schema := cli.Schema{
		Root: rootID,
		Commands: map[cli.CommandID]cli.CommandSchema{
			rootID: {
				ID:    rootID,
				Flags: []cli.FlagID{ambiguousFlagA, ambiguousFlagB},
				Conditions: []cli.ConditionSchema{{Expression: cli.ConditionExpression{
					Destination: actualConditionValue,
				}}},
				Profiles: profiles,
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{
			ambiguousFlagA: {
				ID:          ambiguousFlagA,
				Command:     rootID,
				Destination: ambiguousValueA,
				Spellings:   []cli.FlagSpellingID{ambiguousSpellingA},
			},
			ambiguousFlagB: {
				ID:          ambiguousFlagB,
				Command:     rootID,
				Destination: ambiguousValueB,
				Spellings:   []cli.FlagSpellingID{ambiguousSpellingB},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
			ambiguousSpellingA: {
				ID:   ambiguousSpellingA,
				Flag: ambiguousFlagA,
				Spelling: cli.FlagSpelling{
					Text: "--dup",
				},
			},
			ambiguousSpellingB: {
				ID:   ambiguousSpellingB,
				Flag: ambiguousFlagB,
				Spelling: cli.FlagSpelling{
					Text: "--dup",
				},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{
			shortcutEntryID: {
				ID: shortcutEntryID,
				Defaults: []cli.CommandShortcutDefaultSchema{{
					Destination: actualShortcutValue,
				}},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormSchema{
			argumentFormID: {
				ID:      argumentFormID,
				Profile: actualFormProfile,
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{
			policyID: {
				ID: policyID,
				Actions: []cli.SignalAction{
					{
						EndpointID:    actualEndpoint,
						SignalValueID: actualSignalValue,
						DelayValueID:  actualDelayValue,
					},
				},
			},
		},
	}

	reference := cli.Reference{
		Root: rootID,
		Commands: map[cli.CommandID]cli.CommandReference{
			rootID: {
				ID:    rootID,
				Flags: []cli.FlagID{ambiguousFlagA, ambiguousFlagB},
				Conditions: []cli.ConditionSchema{{Expression: cli.ConditionExpression{
					Destination: actualConditionValue,
				}}},
				Profiles: profiles,
			},
		},
		Flags: map[cli.FlagID]cli.FlagReference{
			ambiguousFlagA: {
				ID:          ambiguousFlagA,
				Command:     rootID,
				Destination: ambiguousValueA,
				Spellings:   []cli.FlagSpellingID{ambiguousSpellingA},
			},
			ambiguousFlagB: {
				ID:          ambiguousFlagB,
				Command:     rootID,
				Destination: ambiguousValueB,
				Spellings:   []cli.FlagSpellingID{ambiguousSpellingB},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingReference{
			ambiguousSpellingA: {
				ID:   ambiguousSpellingA,
				Flag: ambiguousFlagA,
				Spelling: cli.FlagSpelling{
					Text: "--dup",
				},
			},
			ambiguousSpellingB: {
				ID:   ambiguousSpellingB,
				Flag: ambiguousFlagB,
				Spelling: cli.FlagSpelling{
					Text: "--dup",
				},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryReference{
			shortcutEntryID: {
				ID: shortcutEntryID,
				Defaults: []cli.CommandShortcutDefaultReference{{
					Destination: actualShortcutValue,
				}},
			},
		},
		ArgumentForms: map[cli.ArgumentFormID]cli.ArgumentFormReference{
			argumentFormID: {
				ID:      argumentFormID,
				Profile: actualFormProfile,
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{
			policyID: {SignalPolicySchema: cli.SignalPolicySchema{
				ID: policyID,
				Actions: []cli.SignalAction{
					{
						EndpointID:    actualEndpoint,
						SignalValueID: actualSignalValue,
						DelayValueID:  actualDelayValue,
					},
				},
			}},
		},
	}

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
				Flags:                 []cli.FlagID{ambiguousFlagA, ambiguousFlagB},
				Conditions: []cli.ConditionInspection{{Expression: cli.ConditionExpression{
					Destination: actualConditionValue,
				}}},
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			ambiguousFlagA: {
				ID:          ambiguousFlagA,
				Command:     rootID,
				Destination: ambiguousValueA,
				Spellings:   []cli.FlagSpellingID{ambiguousSpellingA},
			},
			ambiguousFlagB: {
				ID:          ambiguousFlagB,
				Command:     rootID,
				Destination: ambiguousValueB,
				Spellings:   []cli.FlagSpellingID{ambiguousSpellingB},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			ambiguousSpellingA: {
				ID:   ambiguousSpellingA,
				Flag: ambiguousFlagA,
				Spelling: cli.FlagSpelling{
					Text: "--dup",
				},
			},
			ambiguousSpellingB: {
				ID:   ambiguousSpellingB,
				Flag: ambiguousFlagB,
				Spelling: cli.FlagSpelling{
					Text: "--dup",
				},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{
			shortcutEntryID: {
				ID: shortcutEntryID,
				Defaults: []cli.CommandShortcutDefaultInspection{{
					Destination: actualShortcutValue,
				}},
			},
		},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{
			policyID: {
				ID: policyID,
				Actions: []cli.SignalAction{
					{
						EndpointID:    actualEndpoint,
						SignalValueID: actualSignalValue,
						DelayValueID:  actualDelayValue,
					},
				},
			},
		},
	}

	assertFailure := func(name string, outcome evaluationResult, semantic string, forbidden ...string) {
		t.Helper()
		if outcome.status != evaluationDisproved {
			t.Fatalf("%s status = %v, want unsatisfied: %+v", name, outcome.status, outcome)
		}
		serialized := renderEvaluationField(outcome.subject) + " " + renderEvaluationField(outcome.expected) + " " + renderEvaluationField(outcome.observed)
		if !strings.Contains(serialized, semantic) {
			t.Fatalf("%s lost semantic wording %q: %+v", name, semantic, outcome)
		}
		for _, raw := range forbidden {
			if strings.Contains(serialized, raw) {
				t.Fatalf("%s exposed raw ID %q: %+v", name, raw, outcome)
			}
		}
	}
	idText := func(id uint32) string { return strconv.FormatUint(uint64(id), 10) }

	directCases := []struct {
		name      string
		outcome   evaluationResult
		semantic  string
		forbidden []string
	}{
		{
			name: "condition reference",
			outcome: referenceExpectationOutcome(
				t,
				RootCommand().ConditionAt(0).Expression().Destination(ValueByID(expectedConditionValue)),
				reference,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualConditionValue)), idText(uint32(expectedConditionValue))},
		},
		{
			name: "condition inspection",
			outcome: inspectionExpectationOutcome(
				t,
				RootCommand().ConditionAt(0).Expression().Destination(ValueByID(expectedConditionValue)),
				inspection,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualConditionValue)), idText(uint32(expectedConditionValue))},
		},
		{
			name: "endpoint reference",
			outcome: referenceExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).Endpoint(SignalEndpointByID(expectedEndpoint)),
				reference,
			),
			semantic:  "different signal-endpoint identity",
			forbidden: []string{idText(uint32(actualEndpoint)), idText(uint32(expectedEndpoint))},
		},
		{
			name: "endpoint inspection",
			outcome: inspectionExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).Endpoint(SignalEndpointByID(expectedEndpoint)),
				inspection,
			),
			semantic:  "different signal-endpoint identity",
			forbidden: []string{idText(uint32(actualEndpoint)), idText(uint32(expectedEndpoint))},
		},
		{
			name: "signal value reference",
			outcome: referenceExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).SignalValue(ValueByID(expectedSignalValue)),
				reference,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualSignalValue)), idText(uint32(expectedSignalValue))},
		},
		{
			name: "signal value inspection",
			outcome: inspectionExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).SignalValue(ValueByID(expectedSignalValue)),
				inspection,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualSignalValue)), idText(uint32(expectedSignalValue))},
		},
		{
			name: "delay value reference",
			outcome: referenceExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).DelayValue(ValueByID(expectedDelayValue)),
				reference,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualDelayValue)), idText(uint32(expectedDelayValue))},
		},
		{
			name: "delay value inspection",
			outcome: inspectionExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).DelayValue(ValueByID(expectedDelayValue)),
				inspection,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualDelayValue)), idText(uint32(expectedDelayValue))},
		},
		{
			name: "shortcut reference",
			outcome: referenceExpectationOutcome(
				t,
				CommandEntryByID(shortcutEntryID).ShortcutDefaultAt(0).Destination(ValueByID(expectedShortcutValue)),
				reference,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualShortcutValue)), idText(uint32(expectedShortcutValue))},
		},
		{
			name: "shortcut inspection",
			outcome: inspectionExpectationOutcome(
				t,
				CommandEntryByID(shortcutEntryID).ShortcutDefaultAt(0).Destination(ValueByID(expectedShortcutValue)),
				inspection,
			),
			semantic:  "different value identity",
			forbidden: []string{idText(uint32(actualShortcutValue)), idText(uint32(expectedShortcutValue))},
		},
		{
			name: "argument profile reference",
			outcome: referenceExpectationOutcome(
				t,
				ArgumentFormByID(argumentFormID).GrammarProfile(GrammarProfileByID(expectedFormProfile)),
				reference,
			),
			semantic:  "different identity",
			forbidden: []string{idText(uint32(actualFormProfile)), idText(uint32(expectedFormProfile))},
		},
		{
			name: "default profile reference",
			outcome: referenceExpectationOutcome(
				t,
				CommandByID(rootID).DefaultGrammarProfile(GrammarProfileByID(expectedDefaultProfile)),
				reference,
			),
			semantic:  "different identity",
			forbidden: []string{idText(uint32(actualDefaultProfile)), idText(uint32(expectedDefaultProfile))},
		},
		{
			name: "profile sequence reference",
			outcome: referenceExpectationOutcome(
				t,
				CommandByID(
					rootID,
				).GrammarProfilesExactly(GrammarProfileByID(caseProfileB), GrammarProfileByID(caseProfileA)),
				reference,
			),
			semantic:  "different identity sequence",
			forbidden: []string{idText(uint32(caseProfileA)), idText(uint32(caseProfileB))},
		},
	}
	for _, tc := range directCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertFailure(tc.name, tc.outcome, tc.semantic, tc.forbidden...)
		})
	}

	ambiguousValue := RootCommand().Flag("--dup").Value()
	ambiguousProfile := RootCommand().GrammarProfile("dup")
	ambiguousFlagIDs := []string{idText(uint32(ambiguousFlagA)), idText(uint32(ambiguousFlagB))}
	ambiguousProfileIDs := []string{idText(uint32(ambiguousProfileA)), idText(uint32(ambiguousProfileB))}

	ambiguousCases := []struct {
		name      string
		outcome   evaluationResult
		forbidden []string
	}{
		{
			name:      "condition schema ambiguous expected value",
			outcome:   schemaExpectationOutcome(t, RootCommand().ConditionAt(0).Expression().Destination(ambiguousValue), &schema),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "condition reference ambiguous expected value",
			outcome: referenceExpectationOutcome(
				t,
				RootCommand().ConditionAt(0).Expression().Destination(ambiguousValue),
				reference,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "condition inspection ambiguous expected value",
			outcome: inspectionExpectationOutcome(
				t,
				RootCommand().ConditionAt(0).Expression().Destination(ambiguousValue),
				inspection,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name:      "signal value schema ambiguous expected value",
			outcome:   schemaExpectationOutcome(t, SignalPolicyByID(policyID).ActionAt(0).SignalValue(ambiguousValue), &schema),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "signal value reference ambiguous expected value",
			outcome: referenceExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).SignalValue(ambiguousValue),
				reference,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "signal value inspection ambiguous expected value",
			outcome: inspectionExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).SignalValue(ambiguousValue),
				inspection,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name:      "delay value schema ambiguous expected value",
			outcome:   schemaExpectationOutcome(t, SignalPolicyByID(policyID).ActionAt(0).DelayValue(ambiguousValue), &schema),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "delay value reference ambiguous expected value",
			outcome: referenceExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).DelayValue(ambiguousValue),
				reference,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "delay value inspection ambiguous expected value",
			outcome: inspectionExpectationOutcome(
				t,
				SignalPolicyByID(policyID).ActionAt(0).DelayValue(ambiguousValue),
				inspection,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "shortcut schema ambiguous expected value",
			outcome: schemaExpectationOutcome(
				t,
				CommandEntryByID(shortcutEntryID).ShortcutDefaultAt(0).Destination(ambiguousValue),
				&schema,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "shortcut reference ambiguous expected value",
			outcome: referenceExpectationOutcome(
				t,
				CommandEntryByID(shortcutEntryID).ShortcutDefaultAt(0).Destination(ambiguousValue),
				reference,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name: "shortcut inspection ambiguous expected value",
			outcome: inspectionExpectationOutcome(
				t,
				CommandEntryByID(shortcutEntryID).ShortcutDefaultAt(0).Destination(ambiguousValue),
				inspection,
			),
			forbidden: ambiguousFlagIDs,
		},
		{
			name:      "argument profile schema ambiguous expected profile",
			outcome:   schemaExpectationOutcome(t, ArgumentFormByID(argumentFormID).GrammarProfile(ambiguousProfile), &schema),
			forbidden: ambiguousProfileIDs,
		},
		{
			name: "argument profile reference ambiguous expected profile",
			outcome: referenceExpectationOutcome(
				t,
				ArgumentFormByID(argumentFormID).GrammarProfile(ambiguousProfile),
				reference,
			),
			forbidden: ambiguousProfileIDs,
		},
		{
			name:      "default profile schema ambiguous expected profile",
			outcome:   schemaExpectationOutcome(t, CommandByID(rootID).DefaultGrammarProfile(ambiguousProfile), &schema),
			forbidden: ambiguousProfileIDs,
		},
		{
			name:      "default profile reference ambiguous expected profile",
			outcome:   referenceExpectationOutcome(t, CommandByID(rootID).DefaultGrammarProfile(ambiguousProfile), reference),
			forbidden: ambiguousProfileIDs,
		},
		{
			name:      "profile sequence schema ambiguous expected profile",
			outcome:   schemaExpectationOutcome(t, CommandByID(rootID).GrammarProfilesExactly(ambiguousProfile), &schema),
			forbidden: ambiguousProfileIDs,
		},
		{
			name:      "profile sequence reference ambiguous expected profile",
			outcome:   referenceExpectationOutcome(t, CommandByID(rootID).GrammarProfilesExactly(ambiguousProfile), reference),
			forbidden: ambiguousProfileIDs,
		},
	}
	for _, tc := range ambiguousCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertFailure(tc.name, tc.outcome, "ambiguous", tc.forbidden...)
		})
	}

	sequenceSchema := schema
	sequenceRoot := sequenceSchema.Commands[rootID]
	sequenceProfiles := *sequenceRoot.Profiles
	sequenceProfiles.Cases = []cli.GrammarProfileID{caseProfileA, caseProfileA, caseProfileB}
	sequenceRoot.Profiles = &sequenceProfiles
	sequenceSchema.Commands = map[cli.CommandID]cli.CommandSchema{rootID: sequenceRoot}

	sequenceReference := reference
	sequenceReferenceRoot := sequenceReference.Commands[rootID]
	sequenceReferenceProfiles := *sequenceReferenceRoot.Profiles
	sequenceReferenceProfiles.Cases = []cli.GrammarProfileID{caseProfileA, caseProfileA, caseProfileB}
	sequenceReferenceRoot.Profiles = &sequenceReferenceProfiles
	sequenceReference.Commands = map[cli.CommandID]cli.CommandReference{rootID: sequenceReferenceRoot}

	exactDuplicate := CommandByID(
		rootID,
	).GrammarProfilesExactly(
		GrammarProfileByID(caseProfileA),
		GrammarProfileByID(caseProfileA),
		GrammarProfileByID(caseProfileB),
	)
	if got := schemaExpectationOutcome(t, exactDuplicate, &sequenceSchema); got.status != evaluationProved {
		t.Fatalf("schema duplicate-sensitive exact sequence = %+v", got)
	}
	if got := referenceExpectationOutcome(t, exactDuplicate, sequenceReference); got.status != evaluationProved {
		t.Fatalf("reference duplicate-sensitive exact sequence = %+v", got)
	}
	missingDuplicate := CommandByID(
		rootID,
	).GrammarProfilesExactly(GrammarProfileByID(caseProfileA), GrammarProfileByID(caseProfileB))
	assertFailure(
		"schema missing duplicate",
		schemaExpectationOutcome(t, missingDuplicate, &sequenceSchema),
		"different identity sequence",
		idText(uint32(caseProfileA)),
		idText(uint32(caseProfileB)),
	)
	assertFailure(
		"reference missing duplicate",
		referenceExpectationOutcome(t, missingDuplicate, sequenceReference),
		"different identity sequence",
		idText(uint32(caseProfileA)),
		idText(uint32(caseProfileB)),
	)
	wrongOrder := CommandByID(
		rootID,
	).GrammarProfilesExactly(
		GrammarProfileByID(caseProfileA),
		GrammarProfileByID(caseProfileB),
		GrammarProfileByID(caseProfileA),
	)
	assertFailure(
		"schema ordered mismatch",
		schemaExpectationOutcome(t, wrongOrder, &sequenceSchema),
		"different identity sequence",
		idText(uint32(caseProfileA)),
		idText(uint32(caseProfileB)),
	)
	assertFailure(
		"reference ordered mismatch",
		referenceExpectationOutcome(t, wrongOrder, sequenceReference),
		"different identity sequence",
		idText(uint32(caseProfileA)),
		idText(uint32(caseProfileB)),
	)
}

func TestDeclarationLiteralTextFailureRedactsSensitivePayload(t *testing.T) {
	t.Parallel()

	const root cli.CommandID = 1
	literal := cli.ValueLiteral{
		Text:      "super-secret",
		Sensitive: true,
		Redacted:  true,
	}
	schema := cli.Schema{
		Root: root,
		Commands: map[cli.CommandID]cli.CommandSchema{
			root: {
				ID: root,
				Conditions: []cli.ConditionSchema{{
					Expression: cli.ConditionExpression{
						Operands: []cli.ValueLiteral{literal},
					},
				}},
			},
		},
	}
	locator := RootCommand().ConditionAt(0).Expression().OperandAt(0)
	expectation, ok := locator.Text("different").(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("Text expectation does not expose schema evaluation")
	}
	outcome := expectation.evaluateSchema(&schema)
	if outcome.status != evaluationDisproved {
		t.Fatalf("literal text mismatch = %+v", outcome)
	}
	if strings.Contains(renderEvaluationField(outcome.observed), literal.Text) || strings.Contains(renderEvaluationField(outcome.expected), literal.Text) {
		t.Fatalf("sensitive literal leaked: %+v", outcome)
	}
	if !strings.Contains(renderEvaluationField(outcome.observed), "<redacted>") {
		t.Fatalf("redaction marker missing: %+v", outcome)
	}
}

func TestDiagnosticGraphCompletenessAndRepairOrder(t *testing.T) {
	t.Parallel()

	closedEvaluator, ok := DiagnosticInputGraphClosed().(diagnosticEvaluator)
	if !ok {
		t.Fatal("DiagnosticInputGraphClosed must satisfy diagnosticEvaluator")
	}
	closed := closedEvaluator.evaluateDiagnosticReport(cli.DiagnosticReport{})
	if closed.status != evaluationProved {
		t.Fatalf("empty graph = %+v", closed)
	}
	openEvaluator, ok := DiagnosticInputGraphClosed().(diagnosticEvaluator)
	if !ok {
		t.Fatal("DiagnosticInputGraphClosed must satisfy diagnosticEvaluator")
	}
	openReport := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{{
			RepairIndexes: []int{0},
		}},
	}
	open := openEvaluator.evaluateDiagnosticReport(openReport)
	if open.status != evaluationDisproved {
		t.Fatalf("dangling repair edge = %+v", open)
	}

	report := cli.DiagnosticReport{
		Repairs: []cli.InputRepair{{}, {}},
	}
	exactEvaluator, ok := RepairsExactly(RepairAt(0), RepairAt(1)).(diagnosticEvaluator)
	if !ok {
		t.Fatal("RepairsExactly must satisfy diagnosticEvaluator")
	}
	exact := exactEvaluator.evaluateDiagnosticReport(report)
	if exact.status != evaluationProved {
		t.Fatalf("exact repair order = %+v", exact)
	}
	wrongEvaluator, ok := RepairsExactly(RepairAt(1), RepairAt(0)).(diagnosticEvaluator)
	if !ok {
		t.Fatal("RepairsExactly must satisfy diagnosticEvaluator")
	}
	wrong := wrongEvaluator.evaluateDiagnosticReport(report)
	if wrong.status != evaluationDisproved {
		t.Fatalf("wrong repair order = %+v", wrong)
	}
}

func TestResolveDeclarationExactBindingIsIndependent(t *testing.T) {
	t.Parallel()
	const actionID cli.ActionEntryID = 7
	schema := cli.Schema{
		Root: 1,
		Commands: map[cli.CommandID]cli.CommandSchema{
			1: {
				ID:            1,
				ActionEntries: []cli.ActionEntryID{actionID},
			},
		},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntrySchema{
			actionID: {
				ID:     actionID,
				Parent: 1,
			},
		},
	}
	result := SchemaResult{
		state: &schemaResultState{
			observation: schema,
		},
	}

	exact := ResolveDeclaration(t, result, ActionEntryByID(actionID))
	if exact.state == nil || exact.state.mode != actionEntryLocatorID || exact.state.id != actionID {
		t.Fatalf("exact locator not preserved: %+v", exact.state)
	}

	human := RootCommand().ActionEntryAt(0)
	one := ResolveDeclaration(t, result, human)
	two := ResolveDeclaration(t, result, human)
	if one.state == two.state {
		t.Fatal("resolved locators share mutable binding state")
	}
	one.state.id = actionID + 1
	if two.state.id != actionID {
		t.Fatalf("binding mutation leaked: got %v", two.state.id)
	}

	for range 4 {
		t.Run("parallel", func(t *testing.T) {
			t.Parallel()
			got := ResolveDeclaration(t, result, human)
			if got.state == nil || got.state.mode != actionEntryLocatorID || got.state.id != actionID {
				t.Fatalf("parallel binding = %+v", got.state)
			}
		})
	}
}

func TestResolveDeclarationFailureModes(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"missing", "ambiguous"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			commandCtx, commandCancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer commandCancel()
			//nolint:gosec // The test deliberately launches the current test binary with test-owned arguments.
			cmd := exec.CommandContext(commandCtx, os.Args[0], "-test.run=^TestResolveDeclarationFailureProbe$")
			cmd.Env = append(os.Environ(), "CLITEST_RESOLVE_DECLARATION_FAILURE="+mode)
			output, err := cmd.CombinedOutput()
			if commandCtx.Err() != nil {
				t.Fatalf("ResolveDeclaration %s case timed out: %v: %s", mode, commandCtx.Err(), output)
			}
			if err == nil {
				t.Fatalf("ResolveDeclaration %s case unexpectedly succeeded", mode)
			}
			text := string(output)
			switch mode {
			case "missing":
				if strings.Contains(text, "704000099") {
					t.Fatalf("ResolveDeclaration failure exposed raw declaration ID: %s", text)
				}
				if !strings.Contains(text, "resolved identity is absent from schema") {
					t.Fatalf("ResolveDeclaration failure lost semantic mismatch wording: %s", text)
				}
			case "ambiguous":
				for _, rawID := range []string{"704000101", "704000102"} {
					if strings.Contains(text, rawID) {
						t.Fatalf("ResolveDeclaration ambiguity exposed raw declaration ID %q: %s", rawID, text)
					}
				}
				if !strings.Contains(text, "locator is ambiguous") {
					t.Fatalf("ResolveDeclaration ambiguity lost semantic mismatch wording: %s", text)
				}
			}
		})
	}
}

func TestResolveDeclarationFailureProbe(t *testing.T) {
	t.Parallel()

	switch os.Getenv("CLITEST_RESOLVE_DECLARATION_FAILURE") {
	case "":
		return
	case "missing":
		schemaResult := SchemaResult{
			state: &schemaResultState{
				observation: cli.Schema{},
			},
		}
		ResolveDeclaration(t, schemaResult, ActionEntryByID(704000099))
	case "ambiguous":
		const root cli.CommandID = 1
		schema := cli.Schema{
			Root: root,
			Commands: map[cli.CommandID]cli.CommandSchema{
				root: {
					ID:    root,
					Flags: []cli.FlagID{704000101, 704000102},
				},
			},
			Flags: map[cli.FlagID]cli.FlagSchema{
				704000101: {
					ID:        704000101,
					Command:   root,
					Spellings: []cli.FlagSpellingID{704000201},
				},
				704000102: {
					ID:        704000102,
					Command:   root,
					Spellings: []cli.FlagSpellingID{704000202},
				},
			},
			FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
				704000201: {
					ID:   704000201,
					Flag: 704000101,
					Spelling: cli.FlagSpelling{
						Text: "--same",
					},
				},
				704000202: {
					ID:   704000202,
					Flag: 704000102,
					Spelling: cli.FlagSpelling{
						Text: "--same",
					},
				},
			},
		}
		schemaResult := SchemaResult{
			state: &schemaResultState{
				observation: schema,
			},
		}
		ResolveDeclaration(t, schemaResult, RootCommand().Flag("--same"))
	default:
		t.Fatalf("unknown probe mode")
	}
}

func TestStaticDefaultAndConditionLocatorsPreserveOrderAndType(t *testing.T) {
	t.Parallel()

	const (
		root    cli.CommandID = 1
		valueID cli.ValueID   = 11
	)
	schema := cli.Schema{
		Root: root,
		Commands: map[cli.CommandID]cli.CommandSchema{
			root: {
				ID: root,
				Conditions: []cli.ConditionSchema{{
					Expression: cli.ConditionExpression{
						Kind: cli.ConditionExpressionAll,
						Children: []cli.ConditionExpression{{
							Kind:        cli.ConditionExpressionEqual,
							Destination: valueID,
							Operands: []cli.ValueLiteral{{
								Type: cli.ValueType{
									Kind: cli.ValueKindString,
								},
								Text: "prod",
							}},
						}},
					},
				}},
			},
		},
		Values: map[cli.ValueID]cli.ValueSchema{
			valueID: {
				ID:      valueID,
				Command: root,
				Kind:    cli.ValueDestinationKeyed,
				HasType: true,
				Type: cli.ValueType{
					Kind: cli.ValueKindString,
				},
				HasStaticDefault: true,
				StaticDefault: cli.ValueStaticDefault{
					HasRepresentation: true,
					Keyed: &cli.ValueKeyedLiteral{
						KeyType: cli.ValueType{
							Kind: cli.ValueKindString,
						},
						ValueType: cli.ValueType{
							Kind: cli.ValueKindString,
						},
						Entries: []cli.ValueKeyedEntry{
							{
								Key: cli.ValueLiteral{
									Type: cli.ValueType{
										Kind: cli.ValueKindString,
									},
									Text: "a",
								},
								Value: cli.ValueLiteral{
									Type: cli.ValueType{
										Kind: cli.ValueKindString,
									},
									Text: "one",
								},
							},
							{
								Key: cli.ValueLiteral{
									Type: cli.ValueType{
										Kind: cli.ValueKindString,
									},
									Text: "z",
								},
								Value: cli.ValueLiteral{
									Type: cli.ValueType{
										Kind: cli.ValueKindString,
									},
									Text: "two",
								},
							},
						},
					},
				},
			},
		},
	}

	staticExpectations := []SchemaExpectation{
		ValueByID(valueID).Kind(cli.ValueDestinationKeyed),
		ValueByID(valueID).Type(cli.ValueType{
			Kind: cli.ValueKindString,
		}),
		ValueByID(valueID).StaticDefault().Keyed().EntryCount(2),
		ValueByID(valueID).StaticDefault().Keyed().EntryAt(0).Key().Text("a"),
		ValueByID(valueID).StaticDefault().Keyed().EntryAt(1).Key().Text("z"),
		RootCommand().ConditionAt(0).Expression().ChildCount(1),
		RootCommand().ConditionAt(0).Expression().ChildAt(0).Kind(cli.ConditionExpressionEqual),
		RootCommand().ConditionAt(0).Expression().ChildAt(0).Destination(ValueByID(valueID)),
		RootCommand().ConditionAt(0).Expression().ChildAt(0).OperandAt(0).Text("prod"),
	}
	for _, expectation := range staticExpectations {
		evaluator, ok := expectation.(interface {
			evaluateSchema(schema *cli.Schema) evaluationResult
		})
		if !ok {
			t.Fatalf("static expectation %T does not expose schema evaluation", expectation)
		}
		if outcome := evaluator.evaluateSchema(&schema); outcome.status != evaluationProved {
			t.Fatalf("static expectation %T = %+v", expectation, outcome)
		}
	}

	outOfRangeExpectation, ok := ValueByID(valueID).StaticDefault().Keyed().EntryAt(2).Key().Text("missing").(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("key Text expectation does not expose schema evaluation")
	}
	outOfRange := outOfRangeExpectation.evaluateSchema(&schema)
	if outOfRange.status != evaluationDisproved {
		t.Fatalf("out-of-range keyed entry = %+v", outOfRange)
	}
}

func TestSignalAndDiagnosticIntegrationExpectationsPreserveTypedFacts(t *testing.T) {
	t.Parallel()

	const policyID cli.SignalPolicyID = 3
	schema := cli.Schema{
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{
			policyID: {
				ID:               policyID,
				Occurrence:       cli.SignalOccurrenceRange{},
				Aggregation:      cli.SignalAggregationEach,
				Deferred:         true,
				ContinueMatching: true,
				Hint:             "interrupt",
				Selector: cli.SignalSelector{
					Origins: []cli.SignalOrigin{cli.SignalOriginRuntime},
				},
				Actions: []cli.SignalAction{{
					Kind:     cli.SignalActionAbort,
					Name:     "abort",
					ExitCode: 17,
				}},
			},
		},
	}
	for _, expectation := range []SchemaExpectation{
		SignalPolicyByID(policyID).Deferred(),
		SignalPolicyByID(policyID).ContinuesMatching(),
		SignalPolicyByID(policyID).Hint("interrupt"),
		SignalPolicyByID(policyID).Selector().OriginsExactly(cli.SignalOriginRuntime),
		SignalPolicyByID(policyID).ActionCount(1),
		SignalPolicyByID(policyID).ActionAt(0).Kind(cli.SignalActionAbort),
		SignalPolicyByID(policyID).ActionAt(0).ExitCode(17),
	} {
		evaluator, ok := expectation.(interface {
			evaluateSchema(schema *cli.Schema) evaluationResult
		})
		if !ok {
			t.Fatalf("signal expectation %T does not expose schema evaluation", expectation)
		}
		outcome := evaluator.evaluateSchema(&schema)
		if outcome.status != evaluationProved {
			t.Fatalf("signal expectation %T = %+v", expectation, outcome)
		}
	}

	diagnostic := cli.Diagnostic{
		SignalAction: &cli.SignalActionFailure{
			PolicyID:    policyID,
			Index:       0,
			CommandPath: []string{"root", "child"},
			Action: cli.SignalAction{
				Kind: cli.SignalActionAbort,
				Name: "abort",
			},
			Event: cli.SignalEvent{
				Origin:          cli.SignalOriginRuntime,
				Phase:           cli.SignalPhaseExecution,
				FirstOccurrence: 2,
				LastOccurrence:  3,
				BatchSize:       2,
			},
			Phase: cli.SignalPhaseExecution,
		},
	}
	report := cli.DiagnosticReport{
		Diagnostics: []cli.Diagnostic{diagnostic},
	}
	for _, expectation := range []DiagnosticReportExpectation{
		DiagnosticAt(0).SignalAction().PolicyIdentityPresent(),
		DiagnosticAt(0).SignalAction().CommandPathExactly("root", "child"),
		DiagnosticAt(0).SignalAction().Action().Kind(cli.SignalActionAbort),
		DiagnosticAt(0).SignalAction().Action().Name("abort"),
		DiagnosticAt(0).SignalAction().Event().Origin(cli.SignalOriginRuntime),
		DiagnosticAt(0).SignalAction().Event().Phase(cli.SignalPhaseExecution),
		DiagnosticAt(0).SignalAction().Event().FirstOccurrence(2),
		DiagnosticAt(0).SignalAction().Event().LastOccurrence(3),
		DiagnosticAt(0).SignalAction().Event().BatchSize(2),
	} {
		evaluator, ok := expectation.(diagnosticEvaluator)
		if !ok {
			t.Fatalf("diagnostic integration expectation %T does not satisfy diagnosticEvaluator", expectation)
		}
		outcome := evaluator.evaluateDiagnosticReport(report)
		if outcome.status != evaluationProved {
			t.Fatalf("diagnostic integration expectation %T = %+v", expectation, outcome)
		}
	}
}

func TestHelpValueDomainEntryLocatorsAggregateSameKindUsesInOrder(t *testing.T) {
	t.Parallel()

	const (
		firstA  cli.ValueDomainEntryID = 1
		firstB  cli.ValueDomainEntryID = 2
		ignored cli.ValueDomainEntryID = 3
		laterA  cli.ValueDomainEntryID = 4
		laterB  cli.ValueDomainEntryID = 5
	)

	uses := []cli.HelpValueDomainUse{
		{
			Kind:    cli.ValueDomainUseDestination,
			Entries: []cli.ValueDomainEntryID{firstA, firstB},
		},
		{
			Kind:    cli.ValueDomainUseMapKey,
			Entries: []cli.ValueDomainEntryID{ignored},
		},
		{
			Kind:    cli.ValueDomainUseDestination,
			Entries: []cli.ValueDomainEntryID{laterA, laterB},
		},
	}
	page := cli.HelpPage{
		ArgumentForms: []cli.HelpArgumentForm{{
			Terms: []cli.HelpArgumentTerm{{
				ValueDomainUses: uses,
			}},
		}},
		FlagGroups: []cli.HelpFlagGroup{{
			Title: "flags",
			Flags: []cli.HelpFlag{{
				FlagSpellings:   []cli.FlagSpellingID{1},
				ValueDomainUses: uses,
			}},
		}},
		FlagSpellings: map[cli.FlagSpellingID]cli.HelpFlagSpelling{
			1: {
				ID: 1,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.HelpValueDomainEntry{
			firstA: {
				ID:   firstA,
				Name: "first-a",
			},
			firstB: {
				ID:   firstB,
				Name: "first-b",
			},
			ignored: {
				ID:   ignored,
				Name: "ignored-map-key",
			},
			laterA: {
				ID:   laterA,
				Name: "later-a",
			},
			laterB: {
				ID:   laterB,
				Name: "later-b",
			},
		},
	}

	term := HelpArgumentFormAt(0).TermAt(0)
	flag := HelpFlagGroup("flags").Flag("--mode")
	RequireHelpPresentation(t, page,
		term.ValueDomainEntryCount(cli.ValueDomainUseDestination, 4),
		term.ValueDomainEntryAt(cli.ValueDomainUseDestination, 0).Name("first-a"),
		term.ValueDomainEntryAt(cli.ValueDomainUseDestination, 1).Name("first-b"),
		term.ValueDomainEntryAt(cli.ValueDomainUseDestination, 2).Name("later-a"),
		term.ValueDomainEntryAt(cli.ValueDomainUseDestination, 3).Name("later-b"),
		term.ValueDomainEntryCount(cli.ValueDomainUseMapKey, 1),
		term.ValueDomainEntryAt(cli.ValueDomainUseMapKey, 0).Name("ignored-map-key"),
		flag.ValueDomainEntryCount(cli.ValueDomainUseDestination, 4),
		flag.ValueDomainEntryAt(cli.ValueDomainUseDestination, 0).Name("first-a"),
		flag.ValueDomainEntryAt(cli.ValueDomainUseDestination, 1).Name("first-b"),
		flag.ValueDomainEntryAt(cli.ValueDomainUseDestination, 2).Name("later-a"),
		flag.ValueDomainEntryAt(cli.ValueDomainUseDestination, 3).Name("later-b"),
		flag.ValueDomainEntryCount(cli.ValueDomainUseMapKey, 1),
		flag.ValueDomainEntryAt(cli.ValueDomainUseMapKey, 0).Name("ignored-map-key"),
	)
}
