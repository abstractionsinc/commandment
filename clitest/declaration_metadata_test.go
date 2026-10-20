package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func declarationMetadataFixtures(origin cli.DeclarationOrigin, release cli.ReleaseLifecycle, lifecycle cli.LifecycleState, replacement *cli.DeclarationReplacement) (*cli.Schema, cli.Reference, cli.Inspection) {
	schema := &cli.Schema{
		CommandEntries:     map[cli.CommandEntryID]cli.CommandEntrySchema{},
		ActionEntries:      map[cli.ActionEntryID]cli.ActionEntrySchema{},
		Flags:              map[cli.FlagID]cli.FlagSchema{},
		FlagSpellings:      map[cli.FlagSpellingID]cli.FlagSpellingSchema{},
		ArgumentForms:      map[cli.ArgumentFormID]cli.ArgumentFormSchema{},
		ArgumentTerms:      map[cli.ArgumentTermID]cli.ArgumentTermSchema{},
		EnvironmentInputs:  map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{},
	}
	reference := cli.Reference{
		CommandEntries:     map[cli.CommandEntryID]cli.CommandEntryReference{},
		ActionEntries:      map[cli.ActionEntryID]cli.ActionEntryReference{},
		Flags:              map[cli.FlagID]cli.FlagReference{},
		FlagSpellings:      map[cli.FlagSpellingID]cli.FlagSpellingReference{},
		ArgumentForms:      map[cli.ArgumentFormID]cli.ArgumentFormReference{},
		ArgumentTerms:      map[cli.ArgumentTermID]cli.ArgumentTermReference{},
		EnvironmentInputs:  map[cli.EnvironmentInputID]cli.EnvironmentInputReference{},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryReference{},
	}
	inspection := cli.Inspection{
		CommandEntries:     map[cli.CommandEntryID]cli.CommandEntryInspection{},
		ActionEntries:      map[cli.ActionEntryID]cli.ActionEntryInspection{},
		Flags:              map[cli.FlagID]cli.FlagInspection{},
		FlagSpellings:      map[cli.FlagSpellingID]cli.FlagSpellingInspection{},
		ArgumentForms:      map[cli.ArgumentFormID]cli.ArgumentFormInspection{},
		ArgumentTerms:      map[cli.ArgumentTermID]cli.ArgumentTermInspection{},
		EnvironmentInputs:  map[cli.EnvironmentInputID]cli.EnvironmentInputInspection{},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{},
	}

	schema.CommandEntries[1] = cli.CommandEntrySchema{ID: 1, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	reference.CommandEntries[1] = cli.CommandEntryReference{ID: 1, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	inspection.CommandEntries[1] = cli.CommandEntryInspection{ID: 1, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}

	schema.ActionEntries[2] = cli.ActionEntrySchema{ID: 2, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	reference.ActionEntries[2] = cli.ActionEntryReference{ID: 2, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	inspection.ActionEntries[2] = cli.ActionEntryInspection{ID: 2, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}

	schema.Flags[3] = cli.FlagSchema{ID: 3, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	reference.Flags[3] = cli.FlagReference{ID: 3, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	inspection.Flags[3] = cli.FlagInspection{ID: 3, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}

	schema.FlagSpellings[4] = cli.FlagSpellingSchema{ID: 4, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	reference.FlagSpellings[4] = cli.FlagSpellingReference{ID: 4, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	inspection.FlagSpellings[4] = cli.FlagSpellingInspection{ID: 4, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}

	schema.ArgumentForms[5] = cli.ArgumentFormSchema{ID: 5, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	reference.ArgumentForms[5] = cli.ArgumentFormReference{ID: 5, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	inspection.ArgumentForms[5] = cli.ArgumentFormInspection{ID: 5, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}

	schema.ArgumentTerms[6] = cli.ArgumentTermSchema{ID: 6, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	reference.ArgumentTerms[6] = cli.ArgumentTermReference{ID: 6, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	inspection.ArgumentTerms[6] = cli.ArgumentTermInspection{ID: 6, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}

	environment := cli.EnvironmentInputSchema{ID: 7, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	schema.EnvironmentInputs[7] = environment
	reference.EnvironmentInputs[7] = cli.EnvironmentInputReference{EnvironmentInputSchema: environment}
	inspection.EnvironmentInputs[7] = cli.EnvironmentInputInspection{EnvironmentInputSchema: environment}

	schema.ValueDomainEntries[8] = cli.ValueDomainEntrySchema{ID: 8, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	reference.ValueDomainEntries[8] = cli.ValueDomainEntryReference{ID: 8, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}
	inspection.ValueDomainEntries[8] = cli.ValueDomainEntryInspection{ID: 8, Origin: origin, Release: release, LifecycleState: lifecycle, Replacement: replacement}

	return schema, reference, inspection
}

func TestDeclarationMetadataCoversAllReleaseBearingDeclarationKinds(t *testing.T) {
	t.Parallel()

	release := cli.ReleaseLifecycle{
		IntroducedIn: "1.0",
		DeprecatedIn: "2.0",
		RemovedIn:    "3.0",
		Reason:       "migration",
	}
	const origin = cli.DeclarationOriginProvider

	schema, reference, inspection := declarationMetadataFixtures(
		origin,
		release,
		cli.LifecycleStateDeprecated,
		nil,
	)

	groups := [][]DeclarationExpectation{
		{
			CommandEntryByID(1).Origin(origin),
			CommandEntryByID(1).ReleaseLifecycle(release),
			CommandEntryByID(1).NotReplaced(),
		},
		{
			ActionEntryByID(2).Origin(origin),
			ActionEntryByID(2).ReleaseLifecycle(release),
			ActionEntryByID(2).NotReplaced(),
		},
		{FlagByID(3).Origin(origin), FlagByID(3).ReleaseLifecycle(release), FlagByID(3).NotReplaced()},
		{
			FlagSpellingByID(4).Origin(origin),
			FlagSpellingByID(4).ReleaseLifecycle(release),
			FlagSpellingByID(4).NotReplaced(),
		},
		{
			ArgumentFormByID(5).Origin(origin),
			ArgumentFormByID(5).ReleaseLifecycle(release),
			ArgumentFormByID(5).NotReplaced(),
		},
		{
			ArgumentTermByID(6).Origin(origin),
			ArgumentTermByID(6).ReleaseLifecycle(release),
			ArgumentTermByID(6).NotReplaced(),
		},
		{
			EnvironmentInputByID(7).Origin(origin),
			EnvironmentInputByID(7).ReleaseLifecycle(release),
			EnvironmentInputByID(7).NotReplaced(),
		},
		{
			ValueDomainEntryByID(8).Origin(origin),
			ValueDomainEntryByID(8).ReleaseLifecycle(release),
			ValueDomainEntryByID(8).NotReplaced(),
		},
	}
	for _, group := range groups {
		for _, expectation := range group {
			if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
				t.Fatalf("schema %T = %+v", expectation, got)
			}
			if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
				t.Fatalf("reference %T = %+v", expectation, got)
			}
			if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
				t.Fatalf("inspection %T = %+v", expectation, got)
			}
		}
	}

	for _, expectation := range []DeclarationExpectation{
		CommandEntryByID(1).Lifecycle(cli.LifecycleStateDeprecated),
		ActionEntryByID(2).Lifecycle(cli.LifecycleStateDeprecated),
		FlagByID(3).Lifecycle(cli.LifecycleStateDeprecated),
		FlagSpellingByID(4).Lifecycle(cli.LifecycleStateDeprecated),
		ArgumentFormByID(5).Lifecycle(cli.LifecycleStateDeprecated),
		ArgumentTermByID(6).Lifecycle(cli.LifecycleStateDeprecated),
		EnvironmentInputByID(7).Lifecycle(cli.LifecycleStateDeprecated),
		ValueDomainEntryByID(8).Lifecycle(cli.LifecycleStateDeprecated),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("lifecycle %T = %+v", expectation, got)
		}
	}
}

func TestValueDomainEntryReplacementUsesTheSamePublicDeclarationLaw(t *testing.T) {
	t.Parallel()

	target := cli.DeclarationIdentity{
		Kind:             cli.DeclarationKindValueDomainEntry,
		ValueDomainEntry: 9,
	}
	replacement := &cli.DeclarationReplacement{
		Target:       target,
		Presentation: "new",
	}
	schema := &cli.Schema{
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{
			8: {
				ID:          8,
				Replacement: replacement,
			}, 9: {ID: 9},
		},
	}
	reference := cli.Reference{
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryReference{
			8: {
				ID:          8,
				Replacement: replacement,
			}, 9: {ID: 9},
		},
	}
	inspection := cli.Inspection{
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{
			8: {
				ID:          8,
				Replacement: replacement,
			}, 9: {ID: 9},
		},
	}
	expectation := ValueDomainEntryByID(8).ReplacedBy(ValueDomainEntryByID(9))
	if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
		t.Fatalf("schema replacement = %+v", got)
	}
	if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
		t.Fatalf("reference replacement = %+v", got)
	}
	if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
		t.Fatalf("inspection replacement = %+v", got)
	}
}

func TestDeclarationMetadataRefusesFalseAbsenceOnTruncatedInspection(t *testing.T) {
	t.Parallel()

	truncated := cli.Inspection{
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{},
		Truncated:          true,
	}
	for _, expectation := range []DeclarationExpectation{
		ValueDomainEntryByID(44).NotReplaced(),
		ValueDomainEntryByID(44).ReleaseLifecycle(cli.ReleaseLifecycle{}),
		ValueDomainEntryByID(44).ReplacementPresentation("replacement"),
		ValueDomainEntryByID(44).ReplacementTargetDeprecatedDuringMigration(false),
	} {
		got := inspectionExpectationOutcome(t, expectation, truncated)
		if got.status != evaluationNotProvable || !strings.Contains(renderEvaluationField(got.observed), "truncated") {
			t.Fatalf("truncated declaration absence for %T = %+v", expectation, got)
		}
	}

	inspection := cli.Inspection{
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{
			44: {
				ID:     44,
				Origin: cli.DeclarationOriginFrontend,
			},
		},
	}
	got := inspectionExpectationOutcome(t, ValueDomainEntryByID(44).Origin(cli.DeclarationOriginProvider), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "origin") {
		t.Fatalf("origin mismatch attribution = %+v", got)
	}
}

func TestReplacementMetadataCoversEveryReleaseBearingDeclarationKind(t *testing.T) {
	t.Parallel()

	replacement := &cli.DeclarationReplacement{
		Presentation:                    "use replacement",
		TargetDeprecatedDuringMigration: true,
	}
	schema, reference, inspection := declarationMetadataFixtures(
		cli.DeclarationOriginUnknown,
		cli.ReleaseLifecycle{},
		cli.LifecycleStateUnknown,
		replacement,
	)

	expectations := []DeclarationExpectation{
		CommandEntryByID(
			1,
		).ReplacementPresentation("use replacement"),
		CommandEntryByID(1).ReplacementTargetDeprecatedDuringMigration(true),
		ActionEntryByID(
			2,
		).ReplacementPresentation("use replacement"),
		ActionEntryByID(2).ReplacementTargetDeprecatedDuringMigration(true),
		FlagByID(
			3,
		).ReplacementPresentation("use replacement"),
		FlagByID(3).ReplacementTargetDeprecatedDuringMigration(true),
		FlagSpellingByID(
			4,
		).ReplacementPresentation("use replacement"),
		FlagSpellingByID(4).ReplacementTargetDeprecatedDuringMigration(true),
		ArgumentFormByID(
			5,
		).ReplacementPresentation("use replacement"),
		ArgumentFormByID(5).ReplacementTargetDeprecatedDuringMigration(true),
		ArgumentTermByID(
			6,
		).ReplacementPresentation("use replacement"),
		ArgumentTermByID(6).ReplacementTargetDeprecatedDuringMigration(true),
		EnvironmentInputByID(
			7,
		).ReplacementPresentation("use replacement"),
		EnvironmentInputByID(7).ReplacementTargetDeprecatedDuringMigration(true),
		ValueDomainEntryByID(
			8,
		).ReplacementPresentation("use replacement"),
		ValueDomainEntryByID(8).ReplacementTargetDeprecatedDuringMigration(true),
	}
	for _, expectation := range expectations {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("schema %T = %+v", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference %T = %+v", expectation, got)
		}
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("inspection %T = %+v", expectation, got)
		}
	}

	got := inspectionExpectationOutcome(t, FlagByID(3).ReplacementPresentation("other"), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "replacement presentation") {
		t.Fatalf("replacement-presentation mismatch attribution = %+v", got)
	}
}

func TestDeclarationMetadataRejectsPresentReplacementAbsentMetadataAndReleaseMismatch(t *testing.T) {
	t.Parallel()

	replacement := &cli.DeclarationReplacement{
		Presentation: "replacement",
	}
	inspection := cli.Inspection{
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{
			8: {
				ID: 8,
				Release: cli.ReleaseLifecycle{
					IntroducedIn: "1.0",
				},
				Replacement: replacement,
			},
			9: {ID: 9},
		},
	}
	if got := inspectionExpectationOutcome(
		t,
		ValueDomainEntryByID(8).NotReplaced(),
		inspection,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "replacement") ||
		!strings.Contains(renderEvaluationField(got.observed), "present") {
		t.Fatalf("present replacement dischecks NotReplaced = %+v", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		ValueDomainEntryByID(9).ReplacementPresentation("replacement"),
		inspection,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "absent") {
		t.Fatalf("replacement metadata without replacement = %+v", got)
	}
	if got := inspectionExpectationOutcome(t, ValueDomainEntryByID(8).ReleaseLifecycle(cli.ReleaseLifecycle{
		IntroducedIn: "2.0",
	}), inspection); got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "release lifecycle") {
		t.Fatalf("release mismatch = %+v", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		ValueDomainEntryByID(8).ReplacementTargetDeprecatedDuringMigration(true),
		inspection,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.subject), "deprecated during migration") {
		t.Fatalf("replacement migration mismatch = %+v", got)
	}
}

func TestOriginCoversAllPublicOriginBearingStaticAndInspectionSubjects(t *testing.T) {
	t.Parallel()

	const origin = cli.DeclarationOriginProvider
	profile := cli.GrammarProfile{
		ID:     22,
		Value:  "prod",
		Origin: origin,
	}
	advisorySchema := cli.AdvisorySchema{
		Code:    "warn",
		Command: 1,
		Origin:  origin,
	}
	advisoryReference := cli.AdvisoryReference{
		Code:    "warn",
		Command: 1,
		Origin:  origin,
	}
	advisoryInspection := cli.AdvisoryInspection{
		Code:    "warn",
		Command: 1,
		Origin:  origin,
	}
	suppressionSchema := cli.NoticeSuppressionSchema{
		Command: 1,
		Origin:  origin,
	}
	suppressionInspection := cli.NoticeSuppressionInspection{
		Command: 1,
		Origin:  origin,
	}

	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID: 1,
			Profiles: &cli.GrammarProfileSet{
				Profiles: map[cli.GrammarProfileID]cli.GrammarProfile{22: profile},
			},
			Advisories:         []cli.AdvisorySchema{advisorySchema},
			NoticeSuppressions: []cli.NoticeSuppressionSchema{suppressionSchema},
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultSchema{21: {
			ID:     21,
			Parent: 1,
			Origin: origin,
		}},
		Values: map[cli.ValueID]cli.ValueSchema{23: {
			ID:      23,
			Command: 1,
			Origin:  origin,
		}},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicySchema{24: {
			ID:      24,
			Command: 1,
			Origin:  origin,
		}},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{25: {
			ID:      25,
			Command: 1,
			Origin:  origin,
		}},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{1: {
			ID: 1,
			Profiles: &cli.GrammarProfileSet{
				Profiles: map[cli.GrammarProfileID]cli.GrammarProfile{22: profile},
			},
			Advisories:         []cli.AdvisoryReference{advisoryReference},
			NoticeSuppressions: []cli.NoticeSuppressionSchema{suppressionSchema},
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultReference{21: {
			ID:     21,
			Parent: 1,
			Origin: origin,
		}},
		Values: map[cli.ValueID]cli.ValueReference{23: {ValueSchema: cli.ValueSchema{
			ID:      23,
			Command: 1,
			Origin:  origin,
		}}},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyReference{
			24: {SignalPolicySchema: cli.SignalPolicySchema{
				ID:      24,
				Command: 1,
				Origin:  origin,
			}},
		},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpoint{25: {
			ID:      25,
			Command: 1,
			Origin:  origin,
		}},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:                 1,
			Advisories:         []cli.AdvisoryInspection{advisoryInspection},
			NoticeSuppressions: []cli.NoticeSuppressionInspection{suppressionInspection},
			SignalPolicies:     []cli.SignalPolicyID{24},
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultInspection{21: {
			ID:     21,
			Parent: 1,
			Origin: origin,
		}},
		SignalPolicies: map[cli.SignalPolicyID]cli.SignalPolicyInspection{24: {
			ID:      24,
			Command: 1,
			Origin:  origin,
		}},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{25: {SignalEndpoint: cli.SignalEndpoint{
			ID:      25,
			Command: 1,
			Origin:  origin,
		}}},
	}

	for _, expectation := range []StaticDeclarationExpectation{
		GrammarProfileByID(22).Origin(origin),
		ValueByID(23).Origin(origin),
	} {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("schema static origin %T = %+v", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference static origin %T = %+v", expectation, got)
		}
	}

	for _, expectation := range []DeclarationExpectation{
		CommandDefaultByID(21).Origin(origin),
		CommandByID(1).AdvisoryAt(0).Origin(origin),
		CommandByID(1).NoticeSuppressionAt(0).Origin(origin),
		SignalPolicyByID(24).Origin(origin),
		SignalEndpointByID(25).Origin(origin),
	} {
		if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
			t.Fatalf("schema origin %T = %+v", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference origin %T = %+v", expectation, got)
		}
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("inspection origin %T = %+v", expectation, got)
		}
	}

	for _, expectation := range []StaticDeclarationExpectation{
		GrammarProfileByID(22).Origin(cli.DeclarationOriginApplication),
		ValueByID(23).Origin(cli.DeclarationOriginApplication),
	} {
		got := schemaExpectationOutcome(t, expectation, schema)
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "origin") {
			t.Fatalf("static origin mismatch attribution for %T = %+v", expectation, got)
		}
	}

	for _, expectation := range []DeclarationExpectation{
		CommandDefaultByID(21).Origin(cli.DeclarationOriginApplication),
		CommandByID(1).AdvisoryAt(0).Origin(cli.DeclarationOriginApplication),
		CommandByID(1).NoticeSuppressionAt(0).Origin(cli.DeclarationOriginApplication),
		SignalPolicyByID(24).Origin(cli.DeclarationOriginApplication),
		SignalEndpointByID(25).Origin(cli.DeclarationOriginApplication),
	} {
		got := inspectionExpectationOutcome(t, expectation, inspection)
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.subject), "origin") {
			t.Fatalf("origin mismatch attribution for %T = %+v", expectation, got)
		}
	}

	truncated := cli.Inspection{
		Truncated: true,
	}
	for _, expectation := range []DeclarationExpectation{
		CommandDefaultByID(21).Origin(origin),
		CommandByID(1).AdvisoryAt(0).Origin(origin),
		CommandByID(1).NoticeSuppressionAt(0).Origin(origin),
		SignalPolicyByID(24).Origin(origin),
		SignalEndpointByID(25).Origin(origin),
	} {
		got := inspectionExpectationOutcome(t, expectation, truncated)
		if got.status != evaluationNotProvable || !strings.Contains(renderEvaluationField(got.observed), "truncated") {
			t.Fatalf("truncated origin absence for %T = %+v", expectation, got)
		}
	}
}

func TestDeclarationMetadataInternalStateFailsClosed(t *testing.T) {
	t.Parallel()

	t.Run("nil schema property lookup", func(t *testing.T) {
		t.Parallel()

		expectations := []SchemaExpectation{
			CommandDefaultByID(1).Origin(cli.DeclarationOriginApplication),
			GrammarProfileByID(1).Origin(cli.DeclarationOriginApplication),
			CommandByID(1).AdvisoryAt(0).Origin(cli.DeclarationOriginApplication),
			CommandByID(1).NoticeSuppressionAt(0).Origin(cli.DeclarationOriginApplication),
			SignalPolicyByID(1).Origin(cli.DeclarationOriginApplication),
			SignalEndpointByID(1).Origin(cli.DeclarationOriginApplication),
		}
		for _, expectation := range expectations {
			if got := schemaExpectationOutcome(t, expectation, nil); got.status != evaluationDisproved {
				t.Fatalf("%T outcome = %#v, want unsatisfied", expectation, got)
			}
		}
	})
}

func TestDeclarationReplacementMetadataPreservesMissingAndMismatchOutcomes(t *testing.T) {
	t.Parallel()

	replacement := &cli.DeclarationReplacement{
		Presentation:                    "actual",
		TargetDeprecatedDuringMigration: false,
	}
	schema := &cli.Schema{
		Flags: map[cli.FlagID]cli.FlagSchema{
			1: {ID: 1},
			2: {
				ID:          2,
				Replacement: replacement,
			},
		},
	}
	reference := cli.Reference{
		Flags: map[cli.FlagID]cli.FlagReference{
			1: {ID: 1},
			2: {
				ID:          2,
				Replacement: replacement,
			},
		},
	}

	for _, tc := range []struct {
		name        string
		expectation DeclarationExpectation
		contains    string
	}{
		{
			name:        "missing declaration",
			expectation: FlagByID(99).ReplacementPresentation("wanted"),
			contains:    "absent",
		},
		{
			name:        "replacement absent",
			expectation: FlagByID(1).ReplacementPresentation("wanted"),
			contains:    "replacement is absent",
		},
		{
			name:        "presentation mismatch",
			expectation: FlagByID(2).ReplacementPresentation("wanted"),
			contains:    "actual",
		},
		{
			name:        "migration mismatch",
			expectation: FlagByID(2).ReplacementTargetDeprecatedDuringMigration(true),
			contains:    "false",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := schemaExpectationOutcome(
				t,
				tc.expectation,
				schema,
			); got.status != evaluationDisproved ||
				!strings.Contains(renderEvaluationField(got.observed), tc.contains) {
				t.Fatalf("schema = %+v", got)
			}
			if got := referenceExpectationOutcome(
				t,
				tc.expectation,
				reference,
			); got.status != evaluationDisproved ||
				!strings.Contains(renderEvaluationField(got.observed), tc.contains) {
				t.Fatalf("reference = %+v", got)
			}
		})
	}
}

func TestDeclarationOriginMetadataPreservesMissingAndMismatchOutcomes(t *testing.T) {
	t.Parallel()

	schema := &cli.Schema{
		Flags: map[cli.FlagID]cli.FlagSchema{1: {
			ID:     1,
			Origin: cli.DeclarationOriginProvider,
		}},
	}
	reference := cli.Reference{
		Flags: map[cli.FlagID]cli.FlagReference{1: {
			ID:     1,
			Origin: cli.DeclarationOriginProvider,
		}},
	}

	if got := schemaExpectationOutcome(
		t,
		FlagByID(99).Origin(cli.DeclarationOriginProvider),
		schema,
	); got.status != evaluationDisproved {
		t.Fatalf("schema missing origin = %+v", got)
	}
	if got := referenceExpectationOutcome(
		t,
		FlagByID(99).Origin(cli.DeclarationOriginProvider),
		reference,
	); got.status != evaluationDisproved {
		t.Fatalf("reference missing origin = %+v", got)
	}
	if got := schemaExpectationOutcome(
		t,
		FlagByID(1).Origin(cli.DeclarationOriginApplication),
		schema,
	); got.status != evaluationDisproved {
		t.Fatalf("schema origin mismatch = %+v", got)
	}
	if got := referenceExpectationOutcome(
		t,
		FlagByID(1).Origin(cli.DeclarationOriginApplication),
		reference,
	); got.status != evaluationDisproved {
		t.Fatalf("reference origin mismatch = %+v", got)
	}
}

func TestRelationalValueOriginUsesResolvedDestinationIdentity(t *testing.T) {
	t.Parallel()

	const expected = cli.DeclarationOriginProvider
	flagID := cli.FlagID(1)
	valueID := cli.ValueID(2)
	schema := &cli.Schema{
		Flags: map[cli.FlagID]cli.FlagSchema{flagID: {
			ID:          flagID,
			Destination: valueID,
		}},
		Values: map[cli.ValueID]cli.ValueSchema{valueID: {
			ID:     valueID,
			Origin: expected,
		}},
	}
	reference := cli.Reference{
		Flags: map[cli.FlagID]cli.FlagReference{flagID: {
			ID:          flagID,
			Destination: valueID,
		}},
		Values: map[cli.ValueID]cli.ValueReference{valueID: {ValueSchema: cli.ValueSchema{
			ID:     valueID,
			Origin: expected,
		}}},
	}
	expectation := FlagByID(flagID).Value().Origin(expected)
	if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
		t.Fatalf("schema relational value origin = %+v", got)
	}
	if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
		t.Fatalf("reference relational value origin = %+v", got)
	}

	missing := FlagByID(99).Value().Origin(expected)
	if got := schemaExpectationOutcome(t, missing, schema); got.status != evaluationDisproved {
		t.Fatalf("schema missing relational value origin = %+v", got)
	}
	if got := referenceExpectationOutcome(t, missing, reference); got.status != evaluationDisproved {
		t.Fatalf("reference missing relational value origin = %+v", got)
	}
}
