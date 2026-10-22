package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestPolicyAndProviderInspectionExpectations(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	providerID := cli.CommandProviderID(2)
	entryID := cli.CommandEntryID(3)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:               commandID,
				Path:             []string{"serve"},
				CommandProviders: []cli.CommandProviderID{providerID},
				Conditions: []cli.ConditionInspection{{
					Outcome:   cli.EvaluationOutcomeSatisfied,
					Sensitive: true,
				}},
				Rules: []cli.RuleInspection{
					{
						Code:              cli.RuleCode("required"),
						Outcome:           cli.EvaluationOutcomeUnsatisfied,
						ActivationOutcome: cli.EvaluationOutcomeSatisfied,
					},
				},
				Advisories: []cli.AdvisoryInspection{{
					Code:              cli.AdvisoryCode("warn"),
					Outcome:           cli.AdvisoryOutcomeMatched,
					ActivationOutcome: cli.EvaluationOutcomeSatisfied,
				}},
				NoticeSuppressions: []cli.NoticeSuppressionInspection{
					{
						ConditionOutcome: cli.EvaluationOutcomeUnsatisfied,
					},
				},
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:             providerID,
				Outcome:        cli.CommandProviderOutcomeAccepted,
				Localization:   &cli.Localization{},
				CommandEntries: []cli.CommandEntryID{entryID},
			},
		},
	}
	command := CommandByID(commandID)
	expectations := []InspectionExpectation{
		command.ProviderAt(0).Outcome(cli.CommandProviderOutcomeAccepted),
		command.ProviderAt(0).EntriesExactly(CommandEntryByID(entryID)),
		command.ProviderAt(0).HasLocalization(),
		command.ConditionAt(0).Outcome(cli.EvaluationOutcomeSatisfied),
		command.ConditionAt(0).Sensitive(),
		command.Rule(cli.RuleCode("required")).Outcome(cli.EvaluationOutcomeUnsatisfied),
		command.RuleAt(0).ActivationOutcome(cli.EvaluationOutcomeSatisfied),
		command.Advisory(cli.AdvisoryCode("warn")).Outcome(cli.AdvisoryOutcomeMatched),
		command.AdvisoryAt(0).ActivationOutcome(cli.EvaluationOutcomeSatisfied),
		command.NoticeSuppressionAt(0).ConditionOutcome(cli.EvaluationOutcomeUnsatisfied),
	}
	for _, expectation := range expectations {
		evaluator, ok := expectation.(interface {
			evaluateInspection(inspection cli.Inspection, err error) evaluationResult
		})
		if !ok {
			t.Fatalf("expectation %T does not expose inspection evaluation", expectation)
		}
		outcome := evaluator.evaluateInspection(inspection, nil)
		if outcome.status != evaluationProved {
			t.Fatalf("expectation %T did not pass: %#v", expectation, outcome)
		}
	}
}

func TestProviderLocalizationPayloadExpectationIsExactReusableAndCompleteObservationAware(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	providerID := cli.CommandProviderID(2)
	expected := cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				"remote command": "commande distante",
			},
		}},
	}
	expectation := CommandByID(commandID).ProviderAt(0).LocalizationExactly(expected)
	expected.Bundles[0].Translations["remote command"] = "mutated expectation input"

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:               commandID,
				CommandProviders: []cli.CommandProviderID{providerID},
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:      providerID,
				Outcome: cli.CommandProviderOutcomeAccepted,
				Localization: &cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"remote command": "commande distante",
						},
					}},
				},
			},
		},
	}
	evaluate := func(value InspectionExpectation, observed cli.Inspection) evaluationResult {
		t.Helper()
		evaluator, ok := value.(interface {
			evaluateInspection(inspection cli.Inspection, err error) evaluationResult
		})
		if !ok {
			t.Fatalf("expectation %T does not expose inspection evaluation", value)
		}
		return evaluator.evaluateInspection(observed, nil)
	}
	for i := range 2 {
		if outcome := evaluate(expectation, inspection); outcome.status != evaluationProved {
			t.Fatalf("reuse %d outcome = %#v", i, outcome)
		}
	}

	mismatch := inspection
	mismatch.CommandProviders = map[cli.CommandProviderID]cli.CommandProviderInspection{
		providerID: {
			ID:      providerID,
			Outcome: cli.CommandProviderOutcomeAccepted,
			Localization: &cli.Localization{
				DefaultLocale: "en",
				Bundles: []cli.TranslationBundle{{
					Locale: "fr",
					Translations: cli.Translations{
						"remote command": "wrong",
					},
				}},
			},
		},
	}
	outcome := evaluate(expectation, mismatch)
	if outcome.status != evaluationDisproved || !strings.Contains(renderEvaluationField(outcome.subject), "localization") ||
		!strings.Contains(renderEvaluationField(outcome.observed), `source "remote command" target "wrong"`) {
		t.Fatalf("payload mismatch outcome = %#v", outcome)
	}

	absent := inspection
	absent.CommandProviders = map[cli.CommandProviderID]cli.CommandProviderInspection{
		providerID: {
			ID:      providerID,
			Outcome: cli.CommandProviderOutcomeAccepted,
		},
	}
	if outcome = evaluate(
		CommandByID(commandID).ProviderAt(0).NoLocalization(),
		absent,
	); outcome.status != evaluationProved {
		t.Fatalf("NoLocalization complete outcome = %#v", outcome)
	}
	if outcome = evaluate(
		CommandByID(commandID).ProviderAt(0).HasLocalization(),
		absent,
	); outcome.status != evaluationDisproved {
		t.Fatalf("HasLocalization absent outcome = %#v", outcome)
	}
	if outcome = evaluate(
		expectation,
		absent,
	); outcome.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(outcome.observed), "absent") {
		t.Fatalf("LocalizationExactly absent outcome = %#v", outcome)
	}

	presentZero := inspection
	presentZero.CommandProviders = map[cli.CommandProviderID]cli.CommandProviderInspection{
		providerID: {
			ID:           providerID,
			Outcome:      cli.CommandProviderOutcomeAccepted,
			Localization: &cli.Localization{},
		},
	}
	if outcome = evaluate(
		CommandByID(commandID).ProviderAt(0).HasLocalization(),
		presentZero,
	); outcome.status != evaluationProved {
		t.Fatalf("HasLocalization present-zero outcome = %#v", outcome)
	}
	if outcome = evaluate(
		CommandByID(commandID).ProviderAt(0).NoLocalization(),
		presentZero,
	); outcome.status != evaluationDisproved {
		t.Fatalf("NoLocalization present-zero outcome = %#v", outcome)
	}
	if outcome = evaluate(
		CommandByID(commandID).ProviderAt(0).LocalizationExactly(cli.Localization{}),
		presentZero,
	); outcome.status != evaluationProved {
		t.Fatalf("LocalizationExactly present-zero outcome = %#v", outcome)
	}

	truncated := absent
	truncated.Truncated = true
	if outcome = evaluate(
		CommandByID(commandID).ProviderAt(0).NoLocalization(),
		truncated,
	); outcome.status != evaluationNotProvable {
		t.Fatalf("NoLocalization truncated outcome = %#v", outcome)
	}
	truncated = inspection
	truncated.Truncated = true
	if outcome = evaluate(expectation, truncated); outcome.status != evaluationNotProvable {
		t.Fatalf("LocalizationExactly truncated outcome = %#v", outcome)
	}
}

func TestPolicyMaterializationResolverNegativeAndPartialLaws(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	providerID := cli.CommandProviderID(2)
	base := cli.Inspection{
		CommandOrder: []cli.CommandID{commandID},
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				Path:                  []string{},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
				CommandProviders:      []cli.CommandProviderID{providerID},
				Conditions: []cli.ConditionInspection{{
					Outcome: cli.EvaluationOutcomeSatisfied,
				}},
				Rules: []cli.RuleInspection{
					{
						Code:    cli.RuleCode("required"),
						Outcome: cli.EvaluationOutcomeSatisfied,
					},
				},
				Advisories: []cli.AdvisoryInspection{{
					Code:    cli.AdvisoryCode("warn"),
					Outcome: cli.AdvisoryOutcomeMatched,
				}},
				NoticeSuppressions: []cli.NoticeSuppressionInspection{
					{
						ConditionOutcome: cli.EvaluationOutcomeSatisfied,
					},
				},
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:      providerID,
				Outcome: cli.CommandProviderOutcomeAccepted,
			},
		},
	}

	if resolved := resolveInspectionCommandLocator(CommandByID(99).state, base); resolved.status != locatorResolutionAbsent {
		t.Fatalf("missing command details = %#v", resolved)
	}
	if resolved := resolveInspectionCommandLocator(Command("missing").state, base); resolved.status != locatorResolutionAbsent {
		t.Fatalf("closed missing command path = %#v", resolved)
	}
	truncated := base
	truncated.Truncated = true
	if resolved := resolveInspectionCommandLocator(Command("missing").state, truncated); resolved.status != locatorResolutionNotProvable {
		t.Fatalf("truncated missing command = %#v", resolved)
	}
	if resolved := resolveInspectionCommandLocator(CommandByID(99).state, truncated); resolved.status != locatorResolutionNotProvable {
		t.Fatalf("truncated exact-ID command details = %#v", resolved)
	}
	if resolved := resolveInspectionCommandLocator(nil, base); resolved.status != locatorResolutionInvalid {
		t.Fatalf("invalid command = %#v", resolved)
	}

	providerCases := []struct {
		name       string
		state      *commandProviderLocatorState
		inspection cli.Inspection
		want       locatorResolutionStatus
	}{
		{
			name:       "invalid",
			state:      nil,
			inspection: base,
			want:       locatorResolutionInvalid,
		},
		{
			name:       "id-found",
			state:      CommandProviderByID(providerID).state,
			inspection: base,
			want:       locatorResolutionFound,
		},
		{
			name:       "id-absent",
			state:      CommandProviderByID(99).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "id-truncated",
			state:      CommandProviderByID(99).state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
		{
			name:       "index-absent",
			state:      CommandByID(commandID).ProviderAt(3).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "index-truncated",
			state:      CommandByID(commandID).ProviderAt(3).state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
		{
			name:       "missing-parent",
			state:      CommandByID(99).ProviderAt(0).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
	}
	for _, tc := range providerCases {
		t.Run("provider/"+tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveInspectionCommandProviderLocator(tc.state, tc.inspection); got.status != tc.want {
				t.Fatalf("status = %v, want %v: %#v", got.status, tc.want, got)
			}
		})
	}

	conditionCases := []struct {
		name       string
		state      *conditionLocatorState
		inspection cli.Inspection
		want       locatorResolutionStatus
	}{
		{
			name:       "invalid",
			state:      nil,
			inspection: base,
			want:       locatorResolutionInvalid,
		},
		{
			name:       "missing-parent",
			state:      CommandByID(99).ConditionAt(0).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "absent",
			state:      CommandByID(commandID).ConditionAt(3).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "truncated",
			state:      CommandByID(commandID).ConditionAt(3).state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
	}
	for _, tc := range conditionCases {
		t.Run("condition/"+tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveInspectionConditionLocator(tc.state, tc.inspection); got.status != tc.want {
				t.Fatalf("status = %v, want %v: %#v", got.status, tc.want, got)
			}
		})
	}

	ambiguousRules := base
	ambiguousRules.Commands = map[cli.CommandID]cli.CommandInspection{
		commandID: {
			ID:   commandID,
			Path: []string{"serve"},
			Rules: []cli.RuleInspection{{
				Code: "dup",
			}, {
				Code: "dup",
			}},
		},
	}
	ruleCases := []struct {
		name       string
		state      *ruleLocatorState
		inspection cli.Inspection
		want       locatorResolutionStatus
	}{
		{
			name:       "invalid",
			state:      nil,
			inspection: base,
			want:       locatorResolutionInvalid,
		},
		{
			name:       "missing-parent",
			state:      CommandByID(99).RuleAt(0).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "index-absent",
			state:      CommandByID(commandID).RuleAt(3).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "index-truncated",
			state:      CommandByID(commandID).RuleAt(3).state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
		{
			name:       "code-ambiguous",
			state:      CommandByID(commandID).Rule("dup").state,
			inspection: ambiguousRules,
			want:       locatorResolutionAmbiguous,
		},
		{
			name:       "code-absent",
			state:      CommandByID(commandID).Rule("missing").state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "code-truncated",
			state:      CommandByID(commandID).Rule("missing").state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
	}
	for _, tc := range ruleCases {
		t.Run("rule/"+tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveInspectionRuleLocator(tc.state, tc.inspection); got.status != tc.want {
				t.Fatalf("status = %v, want %v: %#v", got.status, tc.want, got)
			}
		})
	}

	ambiguousAdvisories := base
	ambiguousAdvisories.Commands = map[cli.CommandID]cli.CommandInspection{
		commandID: {
			ID:   commandID,
			Path: []string{"serve"},
			Advisories: []cli.AdvisoryInspection{{
				Code: "dup",
			}, {
				Code: "dup",
			}},
		},
	}
	advisoryCases := []struct {
		name       string
		state      *advisoryLocatorState
		inspection cli.Inspection
		want       locatorResolutionStatus
	}{
		{
			name:       "invalid",
			state:      nil,
			inspection: base,
			want:       locatorResolutionInvalid,
		},
		{
			name:       "index-absent",
			state:      CommandByID(commandID).AdvisoryAt(3).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "index-truncated",
			state:      CommandByID(commandID).AdvisoryAt(3).state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
		{
			name:       "code-ambiguous",
			state:      CommandByID(commandID).Advisory("dup").state,
			inspection: ambiguousAdvisories,
			want:       locatorResolutionAmbiguous,
		},
		{
			name:       "code-absent",
			state:      CommandByID(commandID).Advisory("missing").state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "code-truncated",
			state:      CommandByID(commandID).Advisory("missing").state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
	}
	for _, tc := range advisoryCases {
		t.Run("advisory/"+tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveInspectionAdvisoryLocator(tc.state, tc.inspection); got.status != tc.want {
				t.Fatalf("status = %v, want %v: %#v", got.status, tc.want, got)
			}
		})
	}

	suppressionCases := []struct {
		name       string
		state      *noticeSuppressionLocatorState
		inspection cli.Inspection
		want       locatorResolutionStatus
	}{
		{
			name:       "invalid",
			state:      nil,
			inspection: base,
			want:       locatorResolutionInvalid,
		},
		{
			name:       "absent",
			state:      CommandByID(commandID).NoticeSuppressionAt(3).state,
			inspection: base,
			want:       locatorResolutionAbsent,
		},
		{
			name:       "truncated",
			state:      CommandByID(commandID).NoticeSuppressionAt(3).state,
			inspection: truncated,
			want:       locatorResolutionNotProvable,
		},
	}
	for _, tc := range suppressionCases {
		t.Run("suppression/"+tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveInspectionNoticeSuppressionLocator(tc.state, tc.inspection); got.status != tc.want {
				t.Fatalf("status = %v, want %v: %#v", got.status, tc.want, got)
			}
		})
	}
}

func TestPolicyMaterializationMismatchRedactionAndExactnessLaws(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	providerID := cli.CommandProviderID(2)
	entryID := cli.CommandEntryID(3)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:               commandID,
				CommandProviders: []cli.CommandProviderID{providerID},
				Conditions: []cli.ConditionInspection{
					{
						Outcome:   cli.EvaluationOutcomeUnsatisfied,
						Sensitive: true,
					},
				},
				Rules: []cli.RuleInspection{
					{
						Code:      "required",
						Outcome:   cli.EvaluationOutcomeUnsatisfied,
						Sensitive: true,
					},
				},
				Advisories: []cli.AdvisoryInspection{{
					Code:    "warn",
					Outcome: cli.AdvisoryOutcomeMatched,
				}},
				NoticeSuppressions: []cli.NoticeSuppressionInspection{
					{
						ConditionOutcome: cli.EvaluationOutcomeUnsatisfied,
					},
				},
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:             providerID,
				Outcome:        cli.CommandProviderOutcomeAccepted,
				CommandEntries: []cli.CommandEntryID{entryID},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{entryID: {
			ID: entryID,
		}},
	}
	command := CommandByID(commandID)

	for _, test := range []struct {
		name        string
		expectation InspectionExpectation
	}{
		{
			name:        "provider-status",
			expectation: command.ProviderAt(0).Outcome(0),
		},
		{
			name:        "provider-entries",
			expectation: command.ProviderAt(0).EntriesExactly(CommandEntryByID(entryID + 1)),
		},
		{
			name:        "condition",
			expectation: command.ConditionAt(0).Outcome(cli.EvaluationOutcomeSatisfied),
		},
		{
			name:        "rule",
			expectation: command.Rule("required").Outcome(cli.EvaluationOutcomeSatisfied),
		},
		{
			name:        "advisory",
			expectation: command.Advisory("warn").Outcome(0),
		},
		{
			name:        "suppression",
			expectation: command.NoticeSuppressionAt(0).ConditionOutcome(cli.EvaluationOutcomeSatisfied),
		},
	} {
		name := test.name
		expectation := test.expectation
		got := inspectionExpectationOutcome(t, expectation, inspection)
		if got.status != evaluationDisproved {
			t.Fatalf("%s = %#v, want disproved", name, got)
		}
	}

	condition := inspectionExpectationOutcome(
		t,
		command.ConditionAt(0).Outcome(cli.EvaluationOutcomeSatisfied),
		inspection,
	)
	if !strings.Contains(renderEvaluationField(condition.expected), "redacted") || !strings.Contains(renderEvaluationField(condition.observed), "redacted") {
		t.Fatalf("sensitive condition mismatch leaked material: %#v", condition)
	}
	rule := inspectionExpectationOutcome(
		t,
		command.Rule("required").Outcome(cli.EvaluationOutcomeSatisfied),
		inspection,
	)
	if !strings.Contains(renderEvaluationField(rule.expected), "redacted") || !strings.Contains(renderEvaluationField(rule.observed), "redacted") {
		t.Fatalf("sensitive rule mismatch leaked material: %#v", rule)
	}

	for _, test := range []struct {
		name        string
		expectation InspectionExpectation
	}{
		{
			name:        "provider",
			expectation: CommandProviderLocator{}.Outcome(cli.CommandProviderOutcomeAccepted),
		},
		{
			name:        "condition",
			expectation: ConditionLocator{}.Outcome(cli.EvaluationOutcomeSatisfied),
		},
		{
			name:        "rule",
			expectation: RuleLocator{}.Outcome(cli.EvaluationOutcomeSatisfied),
		},
		{
			name:        "advisory",
			expectation: AdvisoryLocator{}.Outcome(cli.AdvisoryOutcomeMatched),
		},
		{
			name:        "suppression",
			expectation: NoticeSuppressionLocator{}.ConditionOutcome(cli.EvaluationOutcomeSatisfied),
		},
	} {
		name := test.name
		expectation := test.expectation
		got := inspectionExpectationOutcome(t, expectation, inspection)
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "invalid") {
			t.Fatalf("invalid %s locator = %#v", name, got)
		}
	}
}

func TestPolicyCommandResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	inspection := cli.Inspection{
		CommandOrder: []cli.CommandID{commandID},
		Commands: map[cli.CommandID]cli.CommandInspection{commandID: {
			ID:                    commandID,
			Path:                  []string{},
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		}},
	}
	t.Run("incomplete observation cannot prove absence", func(t *testing.T) {
		t.Parallel()
		truncated := inspection
		truncated.Truncated = true
		if resolved := resolveInspectionCommandLocator(CommandByID(9).state, truncated); resolved.status != locatorResolutionNotProvable {
			t.Fatalf("truncated exact command = %#v", resolved)
		}
	})
	t.Run("closed observation proves absence", func(t *testing.T) {
		t.Parallel()
		if resolved := resolveInspectionCommandLocator(CommandByID(9).state, inspection); resolved.status != locatorResolutionAbsent {
			t.Fatalf("missing exact command = %#v", resolved)
		}
		if resolved := resolveInspectionCommandLocator(CommandByID(commandID).state, inspection); resolved.status != locatorResolutionFound || resolved.value.ID != commandID {
			t.Fatalf("found command = %#v", resolved)
		}
		if resolved := resolveInspectionCommandLocator(nil, inspection); resolved.status != locatorResolutionInvalid {
			t.Fatalf("nil command = %#v", resolved)
		}
	})
}

func TestProviderLocalizationPresencePreservesPresenceTruth(t *testing.T) {
	t.Parallel()
	providerID := cli.CommandProviderID(2)
	present := cli.Localization{
		DefaultLocale: "en",
	}
	inspection := cli.Inspection{
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:           providerID,
				Localization: &present,
			},
		},
	}
	presentExpectation, ok := commandProviderLocalizationExpectation(
		CommandProviderByID(providerID),
		true,
	).(providerLocalizationPresenceExpectation)
	if !ok {
		t.Fatal("commandProviderLocalizationExpectation must return providerLocalizationPresenceExpectation")
	}
	absentExpectation, ok := commandProviderLocalizationExpectation(
		CommandProviderByID(providerID),
		false,
	).(providerLocalizationPresenceExpectation)
	if !ok {
		t.Fatal("commandProviderLocalizationExpectation must return providerLocalizationPresenceExpectation")
	}
	for _, test := range []struct {
		name        string
		expectation providerLocalizationPresenceExpectation
	}{
		{
			name:        "present",
			expectation: presentExpectation,
		},
		{
			name:        "absent",
			expectation: absentExpectation,
		},
	} {
		name := test.name
		expectation := test.expectation
		got := expectation.evaluateInspection(inspection, nil)
		if name == "present" && got.status != evaluationProved {
			t.Fatalf("present localization = %#v", got)
		}
		if name == "absent" && got.status != evaluationDisproved {
			t.Fatalf("unexpected localization = %#v", got)
		}
	}
	truncated := cli.Inspection{
		Truncated: true,
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{providerID: {
			ID: providerID,
		}},
	}
	truncatedExpectation, ok := commandProviderLocalizationExpectation(
		CommandProviderByID(providerID),
		false,
	).(providerLocalizationPresenceExpectation)
	if !ok {
		t.Fatal("commandProviderLocalizationExpectation must return providerLocalizationPresenceExpectation")
	}
	if got := truncatedExpectation.evaluateInspection(
		truncated,
		nil,
	); got.status != evaluationNotProvable {
		t.Fatalf("truncated absence = %#v", got)
	}
}

func TestProviderLocalizationPayloadPreservesSemanticStatus(t *testing.T) {
	t.Parallel()
	providerID := cli.CommandProviderID(2)
	expected := cli.Localization{
		DefaultLocale: "en",
		Bundles: []cli.TranslationBundle{{
			Locale: "fr",
			Translations: cli.Translations{
				"remote": "distant",
			},
		}},
	}
	expectation, ok := commandProviderLocalizationPayloadExpectation(
		CommandProviderByID(providerID),
		expected,
	).(providerLocalizationPayloadExpectation)
	if !ok {
		t.Fatal("commandProviderLocalizationPayloadExpectation must return providerLocalizationPayloadExpectation")
	}
	inspection := cli.Inspection{
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID: providerID,
				Localization: &cli.Localization{
					DefaultLocale: "en",
					Bundles: []cli.TranslationBundle{{
						Locale: "fr",
						Translations: cli.Translations{
							"remote": "distant",
						},
					}},
				},
			},
		},
	}
	if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationProved {
		t.Fatalf("matching localization = %#v", got)
	}
	inspection.CommandProviders[providerID] = cli.CommandProviderInspection{
		ID: providerID,
	}
	if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
		t.Fatalf("missing localization = %#v", got)
	}
	inspection.Truncated = true
	if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationNotProvable {
		t.Fatalf("truncated payload = %#v", got)
	}
}

func TestProviderEntriesExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()
	providerID := cli.CommandProviderID(2)
	first, second := cli.CommandEntryID(3), cli.CommandEntryID(4)
	provider := CommandProviderByID(providerID)
	entries := []CommandEntryLocator{CommandEntryByID(first), CommandEntryByID(second)}
	expectation, ok := commandProviderEntriesExactlyExpectation(provider, entries).(providerEntriesExactlyExpectation)
	if !ok {
		t.Fatal("commandProviderEntriesExactlyExpectation must return providerEntriesExactlyExpectation")
	}
	base := cli.Inspection{
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:             providerID,
				CommandEntries: []cli.CommandEntryID{first, second},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntryInspection{first: {
			ID: first,
		}, second: {
			ID: second,
		}},
	}
	if got := expectation.evaluateInspection(base, nil); got.status != evaluationProved {
		t.Fatalf("exact entries = %#v", got)
	}
	t.Run("providerentriesexactly unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		missingProvider, ok := commandProviderEntriesExactlyExpectation(
			CommandProviderByID(99),
			[]CommandEntryLocator{CommandEntryByID(first)},
		).(providerEntriesExactlyExpectation)
		if !ok {
			t.Fatal("commandProviderEntriesExactlyExpectation must return providerEntriesExactlyExpectation")
		}
		if got := missingProvider.evaluateInspection(base, nil); got.status != evaluationDisproved {
			t.Fatalf("missing provider = %#v", got)
		}
	})
	t.Run("exact provider command entries in authoritative order: incomplete observation", func(t *testing.T) {
		t.Parallel()
		truncated := base
		truncated.Truncated = true
		truncated.CommandEntries = map[cli.CommandEntryID]cli.CommandEntryInspection{first: {
			ID: first,
		}}
		if got := expectation.evaluateInspection(truncated, nil); got.status != evaluationNotProvable {
			t.Fatalf("truncated expected entry = %#v", got)
		}
	})
	t.Run("providerentriesexactly unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		mismatch := base
		mismatch.CommandProviders = map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:             providerID,
				CommandEntries: []cli.CommandEntryID{second, first},
			},
		}
		if got := expectation.evaluateInspection(mismatch, nil); got.status != evaluationDisproved {
			t.Fatalf("entry order mismatch = %#v", got)
		}
	})
}

func TestConditionInspectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
				Conditions: []cli.ConditionInspection{{
					Outcome: cli.EvaluationOutcomeSatisfied,
				}},
			},
		},
	}
	expectation := conditionOutcomeExpectation(
		CommandByID(commandID).ConditionAt(0),
		cli.EvaluationOutcomeSatisfied,
	)
	evaluator, ok := expectation.(inspectionEvaluator)
	if !ok {
		t.Fatal("condition outcome expectation must evaluate Inspection")
	}
	if got := evaluator.evaluateInspection(inspection, nil); got.status != evaluationProved {
		t.Fatalf("condition result = %#v", got)
	}
	mismatch := conditionOutcomeExpectation(
		CommandByID(commandID).ConditionAt(0),
		cli.EvaluationOutcomeUnsatisfied,
	)
	evaluator, ok = mismatch.(inspectionEvaluator)
	if !ok {
		t.Fatal("condition outcome expectation must evaluate Inspection")
	}
	if got := evaluator.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
		t.Fatalf("condition mismatch = %#v", got)
	}
	missing := conditionOutcomeExpectation(
		CommandByID(commandID).ConditionAt(2),
		cli.EvaluationOutcomeSatisfied,
	)
	evaluator, ok = missing.(inspectionEvaluator)
	if !ok {
		t.Fatal("condition outcome expectation must evaluate Inspection")
	}
	if got := evaluator.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
		t.Fatalf("missing condition = %#v", got)
	}
}

func TestRuleInspectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
				Rules: []cli.RuleInspection{{
					Code:    "required",
					Outcome: cli.EvaluationOutcomeSatisfied,
				}},
			},
		},
	}
	expectation := ruleOutcomeExpectation(
		CommandByID(commandID).Rule("required"),
		cli.EvaluationOutcomeSatisfied,
	)
	evaluator, ok := expectation.(inspectionEvaluator)
	if !ok {
		t.Fatal("rule outcome expectation must evaluate Inspection")
	}
	if got := evaluator.evaluateInspection(inspection, nil); got.status != evaluationProved {
		t.Fatalf("rule result = %#v", got)
	}
	mismatch := ruleOutcomeExpectation(
		CommandByID(commandID).Rule("required"),
		cli.EvaluationOutcomeUnsatisfied,
	)
	evaluator, ok = mismatch.(inspectionEvaluator)
	if !ok {
		t.Fatal("rule outcome expectation must evaluate Inspection")
	}
	if got := evaluator.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
		t.Fatalf("rule mismatch = %#v", got)
	}
	missing := ruleOutcomeExpectation(
		CommandByID(commandID).Rule("missing"),
		cli.EvaluationOutcomeSatisfied,
	)
	evaluator, ok = missing.(inspectionEvaluator)
	if !ok {
		t.Fatal("rule outcome expectation must evaluate Inspection")
	}
	if got := evaluator.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
		t.Fatalf("missing rule = %#v", got)
	}
}

func TestPolicyMaterializationInternalStateFailsClosed(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID: 1,
		}},
	}

	t.Run("provider kind", func(t *testing.T) {
		t.Parallel()
		if got := resolveInspectionCommandProviderLocator(&commandProviderLocatorState{}, inspection); got.status != locatorResolutionInvalid {
			t.Fatalf("provider = %#v, want invalid", got)
		}
	})
	t.Run("rule kind", func(t *testing.T) {
		t.Parallel()
		if got := resolveInspectionRuleLocator(&ruleLocatorState{
			command: CommandByID(1).state,
		}, inspection); got.status != locatorResolutionInvalid {
			t.Fatalf("rule = %#v, want invalid", got)
		}
	})
	t.Run("advisory kind", func(t *testing.T) {
		t.Parallel()
		if got := resolveInspectionAdvisoryLocator(&advisoryLocatorState{
			command: CommandByID(1).state,
		}, inspection); got.status != locatorResolutionInvalid {
			t.Fatalf("advisory = %#v, want invalid", got)
		}
	})
	t.Run("provider entry kind", func(t *testing.T) {
		t.Parallel()
		expectation := providerEntriesExactlyExpectation{
			locator: &commandProviderLocatorState{
				mode: commandProviderLocatorID,
				id:   2,
			},
			entries: []*commandEntryLocatorState{{}},
		}
		observed := cli.Inspection{
			CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{2: {
				ID: 2,
			}},
		}
		got := expectation.evaluateInspection(observed, nil)
		rendered := renderEvaluationField(got.observed)
		if got.status != evaluationDisproved || !strings.Contains(rendered, "command-entry") || !strings.Contains(rendered, "invalid") {
			t.Fatalf("provider entries = %#v, want invalid command-entry failure", got)
		}
	})
}

func TestPolicyMaterializationExpectationsConserveMissingResolution(t *testing.T) {
	t.Parallel()

	child := CommandByID(1).Command("missing").state
	closedInspection := cli.Inspection{
		CommandOrder: []cli.CommandID{1},
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:                    1,
			Path:                  []string{},
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		}},
	}
	if resolved := resolveInspectionCommandLocator(child, closedInspection); resolved.status != locatorResolutionAbsent {
		t.Fatalf("complete missing child = %#v", resolved)
	}
	truncatedInspection := cli.Inspection{
		Truncated: true,
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID: 1,
		}},
	}
	if resolved := resolveInspectionCommandLocator(child, truncatedInspection); resolved.status != locatorResolutionNotProvable {
		t.Fatalf("truncated missing child = %#v", resolved)
	}
	missingDetailInspection := cli.Inspection{
		Truncated: true,
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:                    1,
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		}},
	}
	if resolved := resolveInspectionCommandLocator(child, missingDetailInspection); resolved.status != locatorResolutionNotProvable {
		t.Fatalf("truncated complete missing child = %#v", resolved)
	}

	missingProvider := CommandByID(1).ProviderAt(0)
	presence := providerLocalizationPresenceExpectation{
		locator: missingProvider.state,
		present: true,
	}
	if got := presence.evaluateInspection(cli.Inspection{}, nil); got.status != evaluationNotProvable {
		t.Fatalf("open command domain provider localization presence = %#v", got)
	}
	payload := providerLocalizationPayloadExpectation{
		locator:  missingProvider.state,
		expected: cli.Localization{},
	}
	if got := payload.evaluateInspection(cli.Inspection{}, nil); got.status != evaluationNotProvable {
		t.Fatalf("open command domain provider localization payload = %#v", got)
	}

	commandID := cli.CommandID(1)
	providerID := cli.CommandProviderID(2)
	provider := cli.CommandProviderInspection{
		ID:             providerID,
		CommandEntries: []cli.CommandEntryID{3},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				CommandProviders:      []cli.CommandProviderID{providerID},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{providerID: provider},
	}
	exact := providerEntriesExactlyExpectation{
		locator: CommandByID(commandID).ProviderAt(0).state,
		entries: []*commandEntryLocatorState{CommandByID(commandID).Command("missing").CanonicalEntry().state},
	}
	if got := exact.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
		t.Fatalf("unresolved provider entry = %#v", got)
	}
	incomplete := inspection
	incomplete.Commands = map[cli.CommandID]cli.CommandInspection{
		commandID: {
			ID:               commandID,
			CommandProviders: []cli.CommandProviderID{providerID},
		},
	}
	exact = providerEntriesExactlyExpectation{
		locator: CommandByID(commandID).ProviderAt(0).state,
		entries: []*commandEntryLocatorState{CommandByID(commandID).Command("missing").CanonicalEntry().state},
	}
	if got := exact.evaluateInspection(incomplete, nil); got.status != evaluationNotProvable {
		t.Fatalf("not provable provider entry = %#v", got)
	}
}
