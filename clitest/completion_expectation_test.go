package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestOutcomeGraphExpectationsConsumeInspectionAndCompletionPublicGraphs(t *testing.T) {
	t.Parallel()
	sourceID := cli.InputSourceID{}
	originID := cli.InputOriginID{}
	source := cli.InputSource{
		ID:       sourceID,
		Kind:     cli.InputKindArguments,
		Key:      "argv",
		Location: "argument 1",
	}
	origin := cli.InputOrigin{
		ID: originID,
		Span: cli.InputSpan{
			Source: sourceID,
			Entry:  1,
		},
	}

	inspection := cli.Inspection{
		Sources: map[cli.InputSourceID]cli.InputSource{sourceID: source},
		Origins: map[cli.InputOriginID]cli.InputOrigin{originID: origin},
	}
	completion := cli.Completion{
		Sources: map[cli.InputSourceID]cli.InputSource{sourceID: source},
		Origins: map[cli.InputOriginID]cli.InputOrigin{originID: origin},
	}

	assertInspectionExpectation(t, InputSourceByID(sourceID).Present(), inspection)
	assertInspectionExpectation(t, InputSourceByID(sourceID).Kind(cli.InputKindArguments), inspection)
	assertInspectionExpectation(t, InputOriginByID(originID).Present(), inspection)
	assertInspectionExpectation(t, InputOriginByID(originID).Span(origin.Span), inspection)

	assertCompletionExpectation(t, InputSourceByID(sourceID).Present(), completion)
	assertCompletionExpectation(t, InputOriginByID(originID).Span(origin.Span), completion)
}

func TestInspectionAndCompletionObservationExpectations(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Status:    cli.InspectionStatusReady,
		Truncated: true,
	}
	ready, ok := InspectionReady().(inspectionStatusExpectation)
	if !ok {
		t.Fatal("InspectionReady must return inspectionStatusExpectation")
	}
	if got := ready.evaluateInspection(inspection, nil); got.status != evaluationProved {
		t.Fatalf("ready inspection = %#v", got)
	}
	rejected, ok := InspectionRejected().(inspectionStatusExpectation)
	if !ok {
		t.Fatal("InspectionRejected must return inspectionStatusExpectation")
	}
	if got := rejected.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
		t.Fatalf("wrong inspection status = %#v, want disproved", got)
	}
	truncatedInspection, ok := InspectionTruncated().(inspectionTruncatedExpectation)
	if !ok {
		t.Fatal("InspectionTruncated must return inspectionTruncatedExpectation")
	}
	if got := truncatedInspection.evaluateInspection(inspection, nil); got.status != evaluationProved {
		t.Fatalf("truncated inspection = %#v", got)
	}

	span := cli.CompletionSpan{
		Argument: 2,
		Start:    1,
		End:      3,
	}
	completion := cli.Completion{
		Replace:          span,
		RankingAlgorithm: cli.CompletionRankingAlgorithmRelevance,
		Spacing:          cli.CompletionSpacingAppendSpace,
		Partial:          true,
		Truncated:        true,
		RefreshOnInput:   true,
	}
	partial, ok := CompletionPartial().(completionBooleanExpectation)
	if !ok {
		t.Fatal("CompletionPartial must return completionBooleanExpectation")
	}
	truncatedCompletion, ok := CompletionTruncated().(completionBooleanExpectation)
	if !ok {
		t.Fatal("CompletionTruncated must return completionBooleanExpectation")
	}
	refreshOnInput, ok := CompletionRefreshOnInput().(completionBooleanExpectation)
	if !ok {
		t.Fatal("CompletionRefreshOnInput must return completionBooleanExpectation")
	}
	replace, ok := CompletionReplace(span).(completionReplaceExpectation)
	if !ok {
		t.Fatal("CompletionReplace must return completionReplaceExpectation")
	}
	ranking, ok := CompletionRankingAlgorithm(cli.CompletionRankingAlgorithmRelevance).(completionRankingExpectation)
	if !ok {
		t.Fatal("CompletionRankingAlgorithm must return completionRankingExpectation")
	}
	spacing, ok := CompletionSpacing(cli.CompletionSpacingAppendSpace).(completionSpacingExpectation)
	if !ok {
		t.Fatal("CompletionSpacing must return completionSpacingExpectation")
	}
	checks := []evaluationResult{
		partial.evaluateCompletion(completion, nil),
		truncatedCompletion.evaluateCompletion(completion, nil),
		refreshOnInput.evaluateCompletion(completion, nil),
		replace.evaluateCompletion(completion, nil),
		ranking.evaluateCompletion(completion, nil),
		spacing.evaluateCompletion(completion, nil),
	}
	for i, got := range checks {
		if got.status != evaluationProved {
			t.Fatalf("completion check %d = %#v", i, got)
		}
	}
}

func TestCompletionOperationBooleanExpectationsAreAsymmetric(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		result   cli.Completion
		matching CompletionExpectation
		siblings []CompletionExpectation
	}{
		{
			name: "partial",
			result: cli.Completion{
				Partial: true,
			},
			matching: CompletionPartial(),
			siblings: []CompletionExpectation{CompletionTruncated(), CompletionRefreshOnInput()},
		},
		{
			name: "truncated",
			result: cli.Completion{
				Truncated: true,
			},
			matching: CompletionTruncated(),
			siblings: []CompletionExpectation{CompletionPartial(), CompletionRefreshOnInput()},
		},
		{
			name: "refresh",
			result: cli.Completion{
				RefreshOnInput: true,
			},
			matching: CompletionRefreshOnInput(),
			siblings: []CompletionExpectation{CompletionPartial(), CompletionTruncated()},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := evaluateCompletionExpectation(
				t,
				test.matching,
				test.result,
			); got.status != evaluationProved {
				t.Fatalf("matching boolean expectation = %#v", got)
			}
			for i, sibling := range test.siblings {
				if got := evaluateCompletionExpectation(
					t,
					sibling,
					test.result,
				); got.status != evaluationDisproved {
					t.Fatalf("sibling boolean expectation %d = %#v, want disproved", i, got)
				}
			}
		})
	}
}

func TestCompletionCollectionAndSubjectExpectations(t *testing.T) {
	t.Parallel()
	result := cli.Completion{
		Targets: []cli.CompletionTarget{
			{
				Kind:        cli.CompletionTargetCommand,
				Command:     1,
				CommandPath: []string{"serve"},
				Name:        "serve",
			},
			{
				Kind:     cli.CompletionTargetFlagValue,
				Command:  1,
				ValueID:  7,
				Spelling: "--port",
			},
		},
		Items: []cli.CompletionItem{
			{
				Value:    "serve",
				Kind:     cli.CompletionCandidateCommand,
				GroupID:  "commands",
				Priority: 10,
				Declaration: cli.DeclarationIdentity{
					Kind:         cli.DeclarationKindCommandEntry,
					CommandEntry: 9,
				},
				TargetIndexes: []int{0},
				Required:      true,
			},
			{
				Value:         "8080",
				Replacement:   []string{"8080"},
				Kind:          cli.CompletionCandidateValue,
				GroupID:       "values",
				Sensitive:     true,
				TargetIndexes: []int{1},
				Conditional:   true,
			},
		},
		Groups: []cli.CompletionGroup{
			{
				ID:       "commands",
				Priority: 10,
			},
			{
				ID:        "values",
				Sensitive: true,
			},
		},
		NativeRequests: []cli.CompletionNativeRequest{
			{
				Kind:        cli.CompletionNativeFiles,
				Mode:        cli.CompletionNativeAlways,
				TargetIndex: 1,
				Required:    true,
			},
		},
		Delegation: &cli.CompletionDelegation{
			Args: []string{"child", "--"},
			Cursor: cli.CompletionCursor{
				Argument: 1,
				Byte:     2,
			},
			Sensitive: true,
		},
	}

	target0 := CompletionTargetAt(0)
	target1 := CompletionTargetAt(1)
	serve := CompletionItem("serve")
	port8080 := CompletionItem("8080")
	commands := CompletionGroup("commands")
	values := CompletionGroup("values")
	nativeRequest0 := CompletionNativeRequestAt(0)
	delegation := CompletionDelegation()

	assertCompletionExpectation(t, CompletionTargetsExactly(
		target0,
		target1,
	), result)
	assertCompletionExpectation(t, CompletionItemsExactly(
		serve,
		port8080,
	), result)
	assertCompletionExpectation(t, CompletionGroupsExactly(
		commands,
		values,
	), result)
	assertCompletionExpectation(t, CompletionNativeRequestsExactly(nativeRequest0), result)

	assertCompletionExpectation(t, CompletionTarget(cli.CompletionTargetCommand).On(CommandByID(1)).Name("serve"), result)
	assertCompletionExpectation(t, target1.On(ValueByID(7)).Spelling("--port"), result)
	assertCompletionExpectation(t, serve.Kind(cli.CompletionCandidateCommand), result)
	assertCompletionExpectation(t, serve.Declaration(CommandEntryByID(9)), result)
	assertCompletionExpectation(t, serve.Group(commands), result)
	assertCompletionExpectation(t, serve.TargetsExactly(target0), result)
	assertCompletionExpectation(t, serve.Required(), result)
	assertCompletionExpectation(t, port8080.ReplacementExactly("8080"), result)
	assertCompletionExpectation(t, port8080.Sensitive(), result)
	assertCompletionExpectation(t, port8080.Conditional(), result)
	assertCompletionExpectation(t, values.Sensitive(), result)
	assertCompletionExpectation(t, nativeRequest0.Kind(cli.CompletionNativeFiles), result)
	assertCompletionExpectation(t, nativeRequest0.Mode(cli.CompletionNativeAlways), result)
	assertCompletionExpectation(t, nativeRequest0.Required(), result)
	assertCompletionExpectation(t, nativeRequest0.Target().Spelling("--port"), result)
	assertCompletionExpectation(t, delegation.Present(), result)
	assertCompletionExpectation(t, delegation.ArgsExactly("child", "--"), result)
	completion := cli.CompletionCursor{
		Argument: 1,
		Byte:     2,
	}
	assertCompletionExpectation(t, delegation.Cursor(completion), result)
	assertCompletionExpectation(t, delegation.Sensitive(), result)
}

func TestCompletionExactAndNegativeExpectationsRespectTruncation(t *testing.T) {
	t.Parallel()
	result := cli.Completion{
		Items: []cli.CompletionItem{{
			Value: "known",
		}},
		Truncated: true,
	}

	outcome := evaluateCompletionExpectation(t, CompletionItemsExactly(CompletionItem("known")), result)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("exact collection status = %v, want not provable", outcome.status)
	}
	outcome = evaluateCompletionExpectation(t, CompletionItem("missing").Absent(), result)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("absence status = %v, want not provable", outcome.status)
	}
	outcome = evaluateCompletionExpectation(t, CompletionItem("known").Present(), result)
	if outcome.status != evaluationProved {
		t.Fatalf("positive presence status = %v, want proved", outcome.status)
	}
}

func TestSelectionExpectationsUseAuthoritativeSelection(t *testing.T) {
	t.Parallel()
	origin := cli.InputOriginID{}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:   1,
				Path: []string{},
			},
			2: {
				ID:   2,
				Path: []string{"serve"},
			},
		},
		Origins: map[cli.InputOriginID]cli.InputOrigin{origin: {ID: origin}},
		Selection: cli.InvocationSelection{
			Command: cli.CommandSelection{
				Root:     1,
				Selected: 2,
				Path:     []cli.CommandID{1, 2},
				Steps: []cli.CommandSelectionStep{
					{
						Kind:     cli.CommandSelectionStepEntry,
						Entry:    9,
						Input:    origin,
						Commands: []cli.CommandID{2},
					},
				},
			},
		},
	}

	assertInspectionExpectation(t, RootCommand().Selected(), inspection)
	assertInspectionExpectation(t, Command("serve").Selected(), inspection)
	assertInspectionExpectation(t, SelectionPathExactly(RootCommand(), Command("serve")), inspection)
	assertInspectionExpectation(t, SelectionStepsExactly(
		CommandEntryByID(9).SelectionStep().From(InputOriginByID(origin)).Selects(CommandByID(2)),
	), inspection)

	completion := cli.Completion{
		Selection: inspection.Selection.Command,
		Origins:   inspection.Origins,
	}
	assertCompletionExpectation(t, CommandByID(2).Selected(), completion)
	assertCompletionExpectation(t, SelectionPathExactly(CommandByID(1), CommandByID(2)), completion)
	assertCompletionExpectation(t, SelectionStepsExactly(
		CommandEntryByID(9).SelectionStep().From(InputOriginByID(origin)).Selects(CommandByID(2)),
	), completion)

	outcome := evaluateCompletionExpectation(t, Command("serve").Selected(), completion)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("path locator against completion status = %v, want not provable", outcome.status)
	}
}

func completionCapabilityFixture() cli.Completion {
	completion := cli.Completion{
		Targets: []cli.CompletionTarget{
			{
				Kind:        cli.CompletionTargetCommand,
				Command:     1,
				CommandPath: []string{"serve"},
				Name:        "serve",
			},
			{
				Kind:          cli.CompletionTargetFlagValue,
				Command:       1,
				CommandPath:   []string{},
				ValueID:       7,
				Name:          "port",
				Spelling:      "--port",
				FlagValueForm: cli.FlagValueFormSeparate,
				Occurrence:    2,
			},
			{
				Kind:           cli.CompletionTargetArgument,
				Command:        1,
				CommandPath:    []string{},
				ArgumentFormID: 12,
				TermIndex:      2,
				Name:           "input",
				Occurrence:     3,
			},
		},
		Items: []cli.CompletionItem{
			{
				Value:            "serve",
				Replacement:      []string{"serve"},
				DisplayText:      "Serve",
				Description:      "start the server",
				Detail:           "serve detail",
				Kind:             cli.CompletionCandidateCommand,
				GroupID:          "commands",
				SearchTerms:      []string{"serve", "start"},
				SortText:         "01-serve",
				Priority:         10,
				Recommended:      true,
				LifecycleState:   cli.LifecycleStateCurrent,
				Spacing:          cli.CompletionSpacingAppendSpace,
				CommitCharacters: []rune{' ', '/'},
				Declaration: cli.DeclarationIdentity{
					Kind:         cli.DeclarationKindCommandEntry,
					CommandEntry: 9,
				},
				TargetIndexes: []int{0},
				Match: cli.CompletionMatch{
					Kind:       cli.CompletionMatchKindPrefix,
					Field:      cli.CompletionMatchFieldSearchTerm,
					FieldIndex: 1,
					Score:      88,
					Ranges: []cli.CompletionRange{{
						Start: 0,
						End:   2,
					}},
					Edits: 1,
				},
				Required: true,
			},
			{
				Value:         "8080",
				Replacement:   []string{"8080"},
				DisplayText:   "8080",
				Description:   "port",
				Kind:          cli.CompletionCandidateValue,
				GroupID:       "values",
				FallbackOnly:  true,
				Sensitive:     true,
				TargetIndexes: []int{1},
				Conditional:   true,
			},
			{
				Value:         "stdin",
				DisplayText:   "stdin",
				Kind:          cli.CompletionCandidateValue,
				GroupID:       "values",
				TargetIndexes: []int{2},
			},
		},
		Groups: []cli.CompletionGroup{
			{
				ID:       "commands",
				Label:    "Commands",
				Priority: 10,
			},
			{
				ID:       "values",
				Label:    "Values",
				Priority: 5,
			},
		},
		Messages: []cli.CompletionMessage{
			{
				Kind: cli.CompletionMessageInformation,
				Text: "type to filter",
			},
			{
				Kind:      cli.CompletionMessageWarning,
				Sensitive: true,
			},
		},
		Replace: cli.CompletionSpan{
			Argument: 1,
			Start:    1,
			End:      3,
		},
	}
	return completion
}

func TestCompletionPropertyExpectationsMatchObservedCompletion(t *testing.T) {
	t.Parallel()
	result := completionCapabilityFixture()
	completionRange := cli.CompletionRange{
		Start: 0,
		End:   2,
	}
	mismatchRange := cli.CompletionRange{
		Start: 0,
		End:   1,
	}
	message := cli.CompletionMessage{
		Kind: cli.CompletionMessageInformation,
		Text: "other",
	}
	completionSpan := cli.CompletionSpan{
		Argument: 1,
		Start:    0,
		End:      3,
	}
	tests := []struct {
		name     string
		match    CompletionExpectation
		mismatch CompletionExpectation
	}{
		{
			"target command path",
			CompletionTargetAt(0).CommandPathExactly("serve"),
			CompletionTargetAt(0).CommandPathExactly("other"),
		},
		{"target term index", CompletionTargetAt(2).TermIndex(2), CompletionTargetAt(2).TermIndex(1)},
		{
			"target flag value form",
			CompletionTargetAt(1).FlagValueForm(cli.FlagValueFormSeparate),
			CompletionTargetAt(1).FlagValueForm(cli.FlagValueFormJoined),
		},
		{"target occurrence", CompletionTargetAt(1).Occurrence(2), CompletionTargetAt(1).Occurrence(1)},
		{
			"item display text",
			CompletionItem("serve").DisplayText("Serve"),
			CompletionItem("serve").DisplayText("Other"),
		},
		{
			"item description",
			CompletionItem("serve").Description("start the server"),
			CompletionItem("serve").Description("other"),
		},
		{"item detail", CompletionItem("serve").Detail("serve detail"), CompletionItem("serve").Detail("other")},
		{
			"item search terms",
			CompletionItem("serve").SearchTermsExactly("serve", "start"),
			CompletionItem("serve").SearchTermsExactly("start", "serve"),
		},
		{"item sort text", CompletionItem("serve").SortText("01-serve"), CompletionItem("serve").SortText("02-serve")},
		{
			"item commit characters",
			CompletionItem("serve").CommitCharactersExactly(' ', '/'),
			CompletionItem("serve").CommitCharactersExactly('/'),
		},
		{
			"match field",
			CompletionItem("serve").MatchField(cli.CompletionMatchFieldSearchTerm),
			CompletionItem("serve").MatchField(cli.CompletionMatchFieldValue),
		},
		{"match field index", CompletionItem("serve").MatchFieldIndex(1), CompletionItem("serve").MatchFieldIndex(0)},
		{"match score", CompletionItem("serve").MatchScore(88), CompletionItem("serve").MatchScore(87)},
		{"match ranges", CompletionItem("serve").MatchRangesExactly(completionRange), CompletionItem("serve").MatchRangesExactly(mismatchRange)},
		{"match edits", CompletionItem("serve").MatchEdits(1), CompletionItem("serve").MatchEdits(2)},
		{
			"item group",
			CompletionItem("serve").Group(CompletionGroup("commands")),
			CompletionItem("serve").Group(CompletionGroup("values")),
		},
		{"group label", CompletionGroup("commands").Label("Commands"), CompletionGroup("commands").Label("Other")},
		{
			"group membership",
			CompletionGroup("values").ItemsExactly(CompletionItem("8080"), CompletionItem("stdin")),
			CompletionGroup("values").ItemsExactly(CompletionItem("stdin"), CompletionItem("8080")),
		},
		{
			"group membership count",
			CompletionGroup("commands").ItemsExactly(CompletionItem("serve")),
			CompletionGroup("commands").ItemsExactly(),
		},
		{"messages", CompletionMessagesExactly(result.Messages...), CompletionMessagesExactly(message, result.Messages[1])},
		{"message count", CompletionMessagesExactly(result.Messages...), CompletionMessagesExactly(result.Messages[0])},
		{"operation replacement", CompletionReplace(result.Replace), CompletionReplace(completionSpan)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if outcome := evaluateCompletionExpectation(
				t,
				test.match,
				result,
			); outcome.status != evaluationProved {
				t.Fatalf("matching expectation = %#v", outcome)
			}
			if outcome := evaluateCompletionExpectation(
				t,
				test.mismatch,
				result,
			); outcome.status != evaluationDisproved {
				t.Fatalf("mismatch expectation = %#v, want disproved", outcome)
			}
		})
	}

	assertCompletionExpectation(t, CompletionTargetAt(0).On(CommandByID(1)).Present(), result)
	assertCompletionExpectation(t, CompletionTargetAt(0).On(Command("serve")).Present(), result)
	assertCompletionExpectation(t, CompletionTargetAt(1).On(ValueByID(7)).Present(), result)
	assertCompletionExpectation(t, CompletionTargetAt(2).On(ArgumentFormByID(12)).Present(), result)
	assertCompletionExpectation(t, CompletionItem("serve").Group(CompletionGroupAt(0)), result)
	assertCompletionExpectation(t, CompletionMessagesExactly(result.Messages...), result)
	assertCompletionExpectation(t, CompletionMessagesExactly(), cli.Completion{})
}

func TestCompletionFrontendMissingAmbiguousAndUnsupportedCoordinatesFailClosed(t *testing.T) {
	t.Parallel()
	complete := completionCapabilityFixture()

	missing := []CompletionExpectation{
		CompletionItem("missing").Description("anything"),
		CompletionItem("missing").Group(CompletionGroup("commands")),
		CompletionItem("serve").Group(CompletionGroup("missing")),
		CompletionGroup("missing").Label("anything"),
		CompletionGroup("missing").ItemsExactly(),
		CompletionGroup("values").ItemsExactly(CompletionItem("missing"), CompletionItem("stdin")),
		CompletionTarget(cli.CompletionTargetRemainder).Occurrence(0),
	}
	for index, expectation := range missing {
		if outcome := evaluateCompletionExpectation(
			t,
			expectation,
			complete,
		); outcome.status != evaluationDisproved {
			t.Fatalf("missing expectation %d = %#v, want disproved", index, outcome)
		}
	}

	ambiguous := completionCapabilityFixture()
	ambiguous.Items = append(ambiguous.Items, ambiguous.Items[0])
	ambiguous.Groups = append(ambiguous.Groups, ambiguous.Groups[0])
	target := cli.CompletionTarget{
		Kind:    cli.CompletionTargetCommand,
		Command: 2,
		Name:    "other",
	}
	ambiguous.Targets = append(ambiguous.Targets, target)
	checks := []CompletionExpectation{
		CompletionItem("serve").DisplayText("Serve"),
		CompletionItem("serve").Group(CompletionGroupAt(0)),
		CompletionItem("8080").Group(CompletionGroup("commands")),
		CompletionGroup("commands").Label("Commands"),
		CompletionGroup("commands").ItemsExactly(CompletionItem("serve")),
		CompletionTarget(cli.CompletionTargetCommand).Name("serve"),
	}
	for index, expectation := range checks {
		outcome := evaluateCompletionExpectation(t, expectation, ambiguous)
		if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.observed), "indexes") {
			t.Fatalf("ambiguous expectation %d = %#v, want indexed ambiguity", index, outcome)
		}
	}

	ambiguousMembership := completionCapabilityFixture()
	ambiguousMembership.Groups = append(ambiguousMembership.Groups, ambiguousMembership.Groups[0])
	for index, expectation := range []CompletionExpectation{
		CompletionItem("serve").Group(CompletionGroupAt(0)),
		CompletionGroupAt(0).ItemsExactly(CompletionItem("serve")),
	} {
		outcome := evaluateCompletionExpectation(t, expectation, ambiguousMembership)
		if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.observed), "ambiguous") {
			t.Fatalf("ambiguous group membership %d = %#v, want disproved ambiguity", index, outcome)
		}
	}

	unsupported := evaluateCompletionExpectation(t, CompletionTargetAt(1).On(FlagByID(99)).Present(), complete)
	if unsupported.status != evaluationNotProvable || !strings.Contains(renderEvaluationField(unsupported.observed), "does not expose") {
		t.Fatalf("unsupported target coordinate = %#v, want not provable", unsupported)
	}
	unsupported = evaluateCompletionExpectation(t, CompletionTargetAt(2).On(ArgumentTermByID(99)).Present(), complete)
	if unsupported.status != evaluationNotProvable {
		t.Fatalf("unsupported term target coordinate = %#v, want not provable", unsupported)
	}
}

func TestCompletionFrontendExactnessRefusesIncompleteObservation(t *testing.T) {
	t.Parallel()

	for _, incomplete := range []struct {
		name string
		set  func(*cli.Completion)
	}{
		{
			name: "partial",
			set:  func(result *cli.Completion) { result.Partial = true },
		},
		{
			name: "truncated",
			set:  func(result *cli.Completion) { result.Truncated = true },
		},
		{
			name: "partial-and-truncated",
			set:  func(result *cli.Completion) { result.Partial, result.Truncated = true, true },
		},
	} {
		t.Run(incomplete.name, func(t *testing.T) {
			t.Parallel()
			result := completionCapabilityFixture()
			incomplete.set(&result)

			exhaustive := []CompletionExpectation{
				CompletionMessagesExactly(result.Messages...),
				CompletionGroup("values").ItemsExactly(CompletionItem("8080"), CompletionItem("stdin")),
				CompletionItem("serve").TargetsExactly(CompletionTargetAt(0)),
				CompletionItemsExactly(CompletionItem("serve"), CompletionItem("8080"), CompletionItem("stdin")),
				CompletionGroupsExactly(CompletionGroup("commands"), CompletionGroup("values")),
				CompletionTargetsExactly(CompletionTargetAt(0), CompletionTargetAt(1), CompletionTargetAt(2)),
			}
			for index, expectation := range exhaustive {
				if outcome := evaluateCompletionExpectation(
					t,
					expectation,
					result,
				); outcome.status != evaluationNotProvable {
					t.Fatalf("incomplete exhaustive expectation %d = %#v, want not provable", index, outcome)
				}
			}

			negative := []CompletionExpectation{
				CompletionItem("missing").Absent(),
				CompletionGroup("missing").Absent(),
				CompletionTarget(cli.CompletionTargetRemainder).Absent(),
				CompletionDelegation().Absent(),
			}
			for index, expectation := range negative {
				if outcome := evaluateCompletionExpectation(
					t,
					expectation,
					result,
				); outcome.status != evaluationNotProvable {
					t.Fatalf("incomplete negative expectation %d = %#v, want not provable", index, outcome)
				}
			}

			if outcome := evaluateCompletionExpectation(
				t,
				CompletionItem("serve").Description("start the server"),
				result,
			); outcome.status != evaluationProved {
				t.Fatalf("positive retained item property = %#v, want proved", outcome)
			}
			if outcome := evaluateCompletionExpectation(
				t,
				CompletionItem("missing").Description("anything"),
				result,
			); outcome.status != evaluationNotProvable {
				t.Fatalf("missing item property = %#v, want not provable", outcome)
			}
		})
	}
}

func TestCompletionFrontendSnapshotsMutableExpectationInputs(t *testing.T) {
	t.Parallel()
	result := completionCapabilityFixture()

	path := []string{"serve"}
	terms := []string{"serve", "start"}
	characters := []rune{' ', '/'}
	ranges := []cli.CompletionRange{{
		Start: 0,
		End:   2,
	}}
	messages := append([]cli.CompletionMessage(nil), result.Messages...)
	member := CompletionItem("8080")

	expectations := []CompletionExpectation{
		CompletionTargetAt(0).CommandPathExactly(path...),
		CompletionItem("serve").SearchTermsExactly(terms...),
		CompletionItem("serve").CommitCharactersExactly(characters...),
		CompletionItem("serve").MatchRangesExactly(ranges...),
		CompletionMessagesExactly(messages...),
		CompletionGroup("values").ItemsExactly(member, CompletionItem("stdin")),
	}

	path[0] = "mutated"
	terms[0] = "mutated"
	characters[0] = '!'
	ranges[0].End = 99
	messages[0].Text = "mutated"

	for index, expectation := range expectations {
		if outcome := evaluateCompletionExpectation(t, expectation, result); outcome.status != evaluationProved {
			t.Fatalf("snapshotted expectation %d = %#v", index, outcome)
		}
	}
	if result.Items[0].SearchTerms[0] != "serve" || result.Messages[0].Text != "type to filter" {
		t.Fatalf("expectation evaluation mutated observation: %#v %#v", result.Items[0].SearchTerms, result.Messages)
	}
}

func TestCompletionFrontendSensitiveMismatchDiagnosticsAreRedacted(t *testing.T) {
	t.Parallel()
	result := completionCapabilityFixture()
	result.Items[1].Description = "actual-secret"
	outcome := evaluateCompletionExpectation(t, CompletionItem("8080").Description("expected-secret"), result)
	if outcome.status != evaluationDisproved || strings.Contains(renderEvaluationField(outcome.expected), "expected-secret") ||
		strings.Contains(renderEvaluationField(outcome.observed), "actual-secret") {
		t.Fatalf("sensitive item mismatch leaked text: %#v", outcome)
	}

	result.Groups[1].Sensitive = true
	result.Groups[1].Label = "actual-group-secret"
	outcome = evaluateCompletionExpectation(t, CompletionGroup("values").Label("expected-group-secret"), result)
	if outcome.status != evaluationDisproved || strings.Contains(renderEvaluationField(outcome.expected), "expected-group-secret") ||
		strings.Contains(renderEvaluationField(outcome.observed), "actual-group-secret") {
		t.Fatalf("sensitive group mismatch leaked text: %#v", outcome)
	}

	result.Messages[1].Text = "actual-message-secret"
	message := cli.CompletionMessage{
		Kind:      cli.CompletionMessageWarning,
		Text:      "expected-message-secret",
		Sensitive: true,
	}
	outcome = evaluateCompletionExpectation(t, CompletionMessagesExactly(
		result.Messages[0], message,
	), result)
	if outcome.status != evaluationDisproved || strings.Contains(renderEvaluationField(outcome.expected), "expected-message-secret") ||
		strings.Contains(renderEvaluationField(outcome.observed), "actual-message-secret") {
		t.Fatalf("sensitive message mismatch leaked text: %#v", outcome)
	}
}

func TestCompletionLocatorPropertyAndSelectionStepContracts(t *testing.T) {
	t.Parallel()
	result := completionCapabilityFixture()
	result.NativeRequests = []cli.CompletionNativeRequest{{
		Kind:        cli.CompletionNativeFiles,
		Mode:        cli.CompletionNativeAlways,
		TargetIndex: 1,
		Conditional: true,
	}}

	checks := []CompletionExpectation{
		CompletionTargetAt(0).Kind(cli.CompletionTargetCommand),
		CompletionItemAt(0).Value("serve"),
		CompletionItem("serve").Recommended(),
		CompletionItem("8080").FallbackOnly(),
		CompletionItem("serve").Lifecycle(cli.LifecycleStateCurrent),
		CompletionItem("serve").Match(cli.CompletionMatchKindPrefix),
		CompletionItem("serve").Priority(10),
		CompletionItem("serve").Spacing(cli.CompletionSpacingAppendSpace),
		CompletionItemAt(0).TargetAt(0).Name("serve"),
		CompletionGroupAt(0).Present(),
		CompletionGroupAt(0).ID("commands"),
		CompletionGroupAt(0).Priority(10),
		CompletionNativeRequestAt(0).Conditional(),
	}
	for index, expectation := range checks {
		if got := evaluateCompletionExpectation(t, expectation, result); got.status != evaluationProved {
			t.Fatalf("completion expectation %d = %#v", index, got)
		}
	}

	mismatches := []CompletionExpectation{
		CompletionTargetAt(0).Kind(cli.CompletionTargetArgument),
		CompletionItemAt(0).Value("other"),
		CompletionItem("stdin").Recommended(),
		CompletionItem("serve").FallbackOnly(),
		CompletionItem("serve").Lifecycle(cli.LifecycleStateDeprecated),
		CompletionItem("serve").Match(cli.CompletionMatchKindExact),
		CompletionItem("serve").Priority(9),
		CompletionItem("serve").Spacing(cli.CompletionSpacingNoSpace),
		CompletionGroupAt(0).ID("values"),
		CompletionGroupAt(0).Priority(5),
	}
	for index, expectation := range mismatches {
		if got := evaluateCompletionExpectation(t, expectation, result); got.status != evaluationDisproved {
			t.Fatalf("completion mismatch %d = %#v, want disproved", index, got)
		}
	}

	origin := cli.InputOriginID{}
	selection := cli.Completion{
		Origins: map[cli.InputOriginID]cli.InputOrigin{origin: {ID: origin}},
		Selection: cli.CommandSelection{
			Root:     1,
			Selected: 2,
			Path:     []cli.CommandID{1, 2},
			Steps: []cli.CommandSelectionStep{
				{
					Kind:     cli.CommandSelectionStepDefault,
					Default:  17,
					Input:    origin,
					Commands: []cli.CommandID{2},
				},
			},
		},
	}
	assertCompletionExpectation(t, SelectionStepsExactly(
		CommandDefaultByID(17).SelectionStep().From(InputOriginByID(origin)).Selects(CommandByID(2)),
	), selection)
}

func TestCompletionTargetPresencePreservesPresenceTruth(t *testing.T) {
	t.Parallel()

	complete := completionCapabilityFixture()
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		result := cli.Completion{
			Partial: true,
		}
		outcome := evaluateCompletionExpectation(t, CompletionTargetAt(0).Absent(), result)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("partial target absence = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionTargetAt(0).Absent(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("present target negative = %#v", outcome)
		}
	})
	t.Run("absent: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionTargetAt(99).Present(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("missing target positive = %#v", outcome)
		}
	})
}

func TestCompletionItemPresencePreservesPresenceTruth(t *testing.T) {
	t.Parallel()

	complete := completionCapabilityFixture()
	t.Run("present: semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionItem("missing").Present(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("missing item positive = %#v", outcome)
		}
	})
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		result := cli.Completion{
			Partial: true,
		}
		outcome := evaluateCompletionExpectation(t, CompletionItem("missing").Absent(), result)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("partial item absence = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionItem("serve").Absent(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("present item negative = %#v", outcome)
		}
	})
	t.Run("absent: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionItem("missing").Present(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("missing item positive = %#v", outcome)
		}
	})
}

func TestCompletionItemReplacementPreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	outcome := evaluateCompletionExpectation(t, CompletionItem("8080").ReplacementExactly("secret"), result)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.expected) != "redacted expected item property" ||
		renderEvaluationField(outcome.observed) != "redacted observed item property" {
		t.Fatalf("sensitive replacement mismatch = %#v", outcome)
	}
}

func TestCompletionItemKindPreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	outcome := evaluateCompletionExpectation(t, CompletionItem("8080").Kind(cli.CompletionCandidateCommand), result)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "redacted observed item property" {
		t.Fatalf("sensitive kind mismatch = %#v", outcome)
	}
}

func TestCompletionGroupMembershipIdentifiablePreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	result.Groups = append(result.Groups, result.Groups[0])
	if completionGroupMembershipIdentifiable(result.Groups[0], result) {
		t.Fatal("duplicate group ID reported uniquely identifiable")
	}
	if completionGroupMembershipIdentifiable(cli.CompletionGroup{}, result) {
		t.Fatal("empty group ID reported identifiable")
	}
}

func TestCompletionItemTargetsPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	complete := completionCapabilityFixture()
	t.Run("completionitemtargets unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(
			t,
			CompletionItem("missing").TargetsExactly(CompletionTargetAt(0)),
			complete,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("unresolved item targets = %#v", outcome)
		}
	})
	t.Run("targets: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(
			t,
			CompletionItem("serve").TargetsExactly(CompletionTargetAt(1)),
			complete,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("target index mismatch = %#v", outcome)
		}
	})
	t.Run("completionitemtargets unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		result := completionCapabilityFixture()
		result.Partial = true
		outcome := evaluateCompletionExpectation(
			t,
			CompletionItem("serve").TargetsExactly(CompletionTargetAt(0)),
			result,
		)
		if outcome.status != evaluationNotProvable {
			t.Fatalf("partial target collection = %#v", outcome)
		}
	})
}

func TestCompletionGroupPresencePreservesPresenceTruth(t *testing.T) {
	t.Parallel()

	complete := completionCapabilityFixture()
	t.Run("present: semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionGroup("missing").Present(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("missing group positive = %#v", outcome)
		}
	})
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionGroup("missing").Absent(), cli.Completion{
			Partial: true,
		})
		if outcome.status != evaluationNotProvable {
			t.Fatalf("partial group absence = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionGroup("commands").Absent(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("present group negative = %#v", outcome)
		}
	})
	t.Run("absent: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionGroup("missing").Present(), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("missing group positive = %#v", outcome)
		}
	})
}

func TestCompletionGroupSensitivePreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	result.Groups[1].Sensitive = true
	outcome := evaluateCompletionExpectation(t, CompletionGroup("values").Sensitive(), result)
	if outcome.status != evaluationProved {
		t.Fatalf("sensitive group = %#v", outcome)
	}
	outcome = evaluateCompletionExpectation(t, CompletionGroup("values").Priority(99), result)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "redacted observed group property" {
		t.Fatalf("sensitive group mismatch = %#v", outcome)
	}
}

func TestCompletionNativeRequestPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	result.NativeRequests = []cli.CompletionNativeRequest{
		{
			Kind:        cli.CompletionNativeFiles,
			Mode:        cli.CompletionNativeAlways,
			Conditional: true,
		},
	}
	t.Run("completionnativerequestproperty unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionNativeRequestAt(9).Conditional(), result)
		if outcome.status != evaluationDisproved {
			t.Fatalf("missing native request = %#v", outcome)
		}
	})
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionNativeRequestAt(0).Conditional(), result)
		if outcome.status != evaluationProved {
			t.Fatalf("native request conditional = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(
			t,
			CompletionNativeRequestAt(0).Kind(cli.CompletionNativeDirectories),
			result,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("native request mismatch = %#v", outcome)
		}
	})
}

func TestCompletionNativeRequestKindPreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	result.NativeRequests = []cli.CompletionNativeRequest{{Kind: cli.CompletionNativeFiles}}
	outcome := evaluateCompletionExpectation(
		t,
		CompletionNativeRequestAt(0).Kind(cli.CompletionNativeDirectories),
		result,
	)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
		t.Fatalf("native kind mismatch = %#v", outcome)
	}
}

func TestCompletionNativeRequestModePreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	result.NativeRequests = []cli.CompletionNativeRequest{{Mode: cli.CompletionNativeAlways}}
	outcome := evaluateCompletionExpectation(
		t,
		CompletionNativeRequestAt(0).Mode(cli.CompletionNativeOnNoMatch),
		result,
	)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
		t.Fatalf("native mode mismatch = %#v", outcome)
	}
}

func TestCompletionNativeRequestBoolPreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	result.NativeRequests = []cli.CompletionNativeRequest{{Conditional: false}}
	outcome := evaluateCompletionExpectation(t, CompletionNativeRequestAt(0).Conditional(), result)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) == "" {
		t.Fatalf("native bool mismatch = %#v", outcome)
	}
}

func TestCompletionDelegationPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	t.Run("semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Present(), cli.Completion{})
		if outcome.status != evaluationDisproved {
			t.Fatalf("absent delegation = %#v", outcome)
		}
	})
	t.Run("redacted value: semantic absence", func(t *testing.T) {
		t.Parallel()
		result := cli.Completion{
			Delegation: &cli.CompletionDelegation{
				Sensitive: true,
			},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().ArgsExactly("secret"), result)
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "redacted observed delegation property" {
			t.Fatalf("sensitive delegation = %#v", outcome)
		}
	})
	t.Run("delegation is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().ArgsExactly("x"), cli.Completion{})
		if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "delegation is absent" {
			t.Fatalf("absent delegation args = %#v", outcome)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		result := cli.Completion{
			Delegation: &cli.CompletionDelegation{
				Args: []string{"actual"},
			},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().ArgsExactly("expected"), result)
		if outcome.status != evaluationDisproved {
			t.Fatalf("delegation args mismatch = %#v", outcome)
		}
	})
	t.Run("completion delegation: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		result := cli.Completion{
			Partial: true,
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Present(), result)
		if outcome.status != evaluationNotProvable || renderEvaluationField(outcome.subject) != "completion delegation" {
			t.Fatalf("partial delegation = %#v", outcome)
		}
	})
}

func TestCompletionDelegationPresencePreservesRedaction(t *testing.T) {
	t.Parallel()

	result := cli.Completion{
		Delegation: &cli.CompletionDelegation{
			Sensitive: true,
		},
	}
	outcome := evaluateCompletionExpectation(t, CompletionDelegation().Absent(), result)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "redacted observed delegation property" {
		t.Fatalf("sensitive presence mismatch = %#v", outcome)
	}
}

func TestCompletionDelegationArgsPreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("semantic absence", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			Delegation: &cli.CompletionDelegation{
				Args: []string{"y"},
			},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().ArgsExactly("x"), completion)
		if outcome.status != evaluationDisproved {
			t.Fatalf("delegation args mismatch = %#v", outcome)
		}
	})
	t.Run("delegation is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().ArgsExactly("x"), cli.Completion{})
		if renderEvaluationField(outcome.observed) != "delegation is absent" {
			t.Fatalf("absent delegation args = %#v", outcome)
		}
	})
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			Delegation: &cli.CompletionDelegation{
				Args: []string{"x"},
			},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().ArgsExactly("x"), completion)
		if outcome.status != evaluationProved {
			t.Fatalf("delegation args = %#v", outcome)
		}
	})
}

func TestCompletionDelegationCursorPreservesRedaction(t *testing.T) {
	t.Parallel()

	cursor := cli.CompletionCursor{
		Argument: 1,
		Byte:     2,
	}
	t.Run("semantic absence", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			Delegation: &cli.CompletionDelegation{},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Cursor(cursor), completion)
		if outcome.status != evaluationDisproved {
			t.Fatalf("delegation cursor mismatch = %#v", outcome)
		}
	})
	t.Run("delegation is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Cursor(cursor), cli.Completion{})
		if renderEvaluationField(outcome.observed) != "delegation is absent" {
			t.Fatalf("absent delegation cursor = %#v", outcome)
		}
	})
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			Delegation: &cli.CompletionDelegation{
				Cursor: cursor,
			},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Cursor(cursor), completion)
		if outcome.status != evaluationProved {
			t.Fatalf("delegation cursor = %#v", outcome)
		}
	})
}

func TestCompletionDelegationSensitivePreservesRedaction(t *testing.T) {
	t.Parallel()

	t.Run("semantic absence", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			Delegation: &cli.CompletionDelegation{},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Sensitive(), completion)
		if outcome.status != evaluationDisproved {
			t.Fatalf("delegation sensitivity mismatch = %#v", outcome)
		}
	})
	t.Run("delegation is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Sensitive(), cli.Completion{})
		if renderEvaluationField(outcome.observed) != "delegation is absent" {
			t.Fatalf("absent delegation sensitivity = %#v", outcome)
		}
	})
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		completion := cli.Completion{
			Delegation: &cli.CompletionDelegation{
				Sensitive: true,
			},
		}
		outcome := evaluateCompletionExpectation(t, CompletionDelegation().Sensitive(), completion)
		if outcome.status != evaluationProved {
			t.Fatalf("delegation sensitive = %#v", outcome)
		}
	})
}

func TestCompletionCollectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	complete := completionCapabilityFixture()
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(t, CompletionTargetsExactly(CompletionTargetAt(0)), complete)
		if outcome.status != evaluationDisproved {
			t.Fatalf("collection cardinality mismatch = %#v", outcome)
		}
	})
	t.Run("completioncollection unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		outcome := evaluateCompletionExpectation(
			t,
			CompletionTargetsExactly(CompletionTargetAt(99), CompletionTargetAt(1), CompletionTargetAt(2)),
			complete,
		)
		if outcome.status != evaluationDisproved {
			t.Fatalf("unresolved collection member = %#v", outcome)
		}
	})
}

func TestCompletionItemDeclarationPreservesRedaction(t *testing.T) {
	t.Parallel()

	result := completionCapabilityFixture()
	outcome := evaluateCompletionExpectation(t, CompletionItem("8080").Declaration(CommandEntryByID(9)), result)
	if outcome.status != evaluationDisproved || renderEvaluationField(outcome.observed) != "redacted observed item property" {
		t.Fatalf("sensitive declaration mismatch = %#v", outcome)
	}
}

func TestCompletionExpectationInternalStateFailsClosed(t *testing.T) {
	t.Parallel()
	result := cli.Completion{
		Items: []cli.CompletionItem{{
			Value:         "serve",
			TargetIndexes: []int{1},
		}},
		Targets: []cli.CompletionTarget{{}},
	}
	expectation, ok := completionItemTargetsExactlyExpectation(
		CompletionItemAt(0),
		[]CompletionTargetLocator{CompletionTargetAt(0)},
	).(completionItemTargetsExpectationValue)
	if !ok {
		t.Fatal("completionItemTargetsExactlyExpectation must return completionItemTargetsExpectationValue")
	}
	if got := expectation.evaluateCompletion(
		result,
		nil,
	); got.status != evaluationDisproved ||
		renderEvaluationField(got.observed) != "item references unavailable target" {
		t.Fatalf("unavailable target = %#v", got)
	}
}

func TestCompletionExpectationsConserveNegativeAndExactBoundaryStates(t *testing.T) {
	t.Parallel()

	if got := (completionTargetPresenceExpectationValue{
		locator: CompletionTargetAt(0).state,
		present: false,
	}).evaluateCompletion(cli.Completion{}, nil); got.status != evaluationProved {
		t.Fatalf("absent target = %#v", got)
	}
	if got := (completionTargetPresenceExpectationValue{
		locator: &completionTargetLocatorState{},
		present: false,
	}).evaluateCompletion(cli.Completion{}, nil); got.status != evaluationDisproved {
		t.Fatalf("invalid target locator = %#v", got)
	}
	if got := (completionItemPresenceExpectationValue{
		locator: CompletionItemAt(0).state,
		present: false,
	}).evaluateCompletion(cli.Completion{}, nil); got.status != evaluationProved {
		t.Fatalf("absent item = %#v", got)
	}
	if got := (completionItemPresenceExpectationValue{
		locator: &completionItemLocatorState{},
		present: false,
	}).evaluateCompletion(cli.Completion{}, nil); got.status != evaluationDisproved {
		t.Fatalf("invalid item locator = %#v", got)
	}
	if got := (completionGroupPresenceExpectationValue{
		locator: CompletionGroupAt(0).state,
		present: false,
	}).evaluateCompletion(cli.Completion{}, nil); got.status != evaluationProved {
		t.Fatalf("absent group = %#v", got)
	}
	if got := (completionGroupPresenceExpectationValue{
		locator: &completionGroupLocatorState{},
		present: false,
	}).evaluateCompletion(cli.Completion{}, nil); got.status != evaluationDisproved {
		t.Fatalf("invalid group locator = %#v", got)
	}

	item := cli.CompletionItem{
		Value:       "item",
		Replacement: []string{"actual"},
		Kind:        cli.CompletionCandidateValue,
	}
	result := cli.Completion{
		Items: []cli.CompletionItem{item},
	}
	replacementExpectation, ok := completionItemReplacementExpectation(
		CompletionItemAt(0),
		[]string{"expected"},
	).(completionItemPropertyExpectation)
	if !ok {
		t.Fatal("completionItemReplacementExpectation must return completionItemPropertyExpectation")
	}
	if got := replacementExpectation.evaluateCompletion(result, nil); got.status != evaluationDisproved {
		t.Fatalf("replacement mismatch = %#v", got)
	}
	kindExpectation, ok := completionItemKindExpectation(
		CompletionItemAt(0),
		cli.CompletionCandidateCommand,
	).(completionItemPropertyExpectation)
	if !ok {
		t.Fatal("completionItemKindExpectation must return completionItemPropertyExpectation")
	}
	if got := kindExpectation.evaluateCompletion(result, nil); got.status != evaluationDisproved {
		t.Fatalf("kind mismatch = %#v", got)
	}
	declarationExpectation, ok := completionItemDeclarationExpectation(
		CompletionItemAt(0),
		CommandEntryByID(9),
	).(completionItemPropertyExpectation)
	if !ok {
		t.Fatal("completionItemDeclarationExpectation must return completionItemPropertyExpectation")
	}
	if got := declarationExpectation.evaluateCompletion(result, nil); got.status != evaluationDisproved {
		t.Fatalf("declaration mismatch = %#v", got)
	}

	groupResult := cli.Completion{
		Groups: []cli.CompletionGroup{{ID: "group"}},
	}
	groupSensitiveExpectation, ok := completionGroupSensitiveExpectation(
		CompletionGroupAt(0),
	).(completionGroupPropertyExpectation)
	if !ok {
		t.Fatal("completionGroupSensitiveExpectation must return completionGroupPropertyExpectation")
	}
	if got := groupSensitiveExpectation.evaluateCompletion(groupResult, nil); got.status != evaluationDisproved {
		t.Fatalf("group sensitivity mismatch = %#v", got)
	}

	itemWithTarget := cli.Completion{
		Items: []cli.CompletionItem{{
			Value:         "item",
			TargetIndexes: []int{0},
		}},
		Targets: []cli.CompletionTarget{{Kind: cli.CompletionTargetCommand}},
	}
	if got := (completionItemTargetsExpectationValue{
		item: CompletionItemAt(0).state,
	}).evaluateCompletion(itemWithTarget, nil); got.status != evaluationDisproved {
		t.Fatalf("target count mismatch = %#v", got)
	}
	if got := (completionItemTargetsExpectationValue{
		item:    CompletionItemAt(0).state,
		targets: []*completionTargetLocatorState{{}},
	}).evaluateCompletion(itemWithTarget, nil); got.status != evaluationDisproved {
		t.Fatalf("unresolved expected target = %#v", got)
	}
}
