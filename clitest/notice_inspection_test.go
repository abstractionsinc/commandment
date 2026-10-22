package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestNoticeOnPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()
	base := Notice(cli.NoticeDeprecatedFlag)
	t.Run("semantic result", func(t *testing.T) {
		t.Parallel()
		subjects := []NoticeSubject{
			CommandByID(1),
			CommandEntryByID(2),
			ActionEntryByID(3),
			FlagByID(4),
			FlagSpellingByID(5),
			ArgumentFormByID(6),
			ArgumentTermByID(7),
			EnvironmentInputByID(8),
			ValueByID(9).Domain(cli.ValueDomainUseDestination).Entry("old"),
		}
		for i, subject := range subjects {
			got := noticeOn(base, subject)
			if got.state == nil || !got.state.subjectSet || got.state.subject == nil {
				t.Fatalf("subject %d = %#v", i, got.state)
			}
		}
	})
	t.Run("invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := noticeOn(base, nil)
		if got.state == nil || !got.state.subjectSet {
			t.Fatalf("nil subject coordinate = %#v", got.state)
		}
		if _, status, _ := resolveNoticeSubject(got.state.subject, cli.Inspection{}); status != locatorResolutionInvalid {
			t.Fatalf("nil subject status = %v, want invalid", status)
		}
		if got := noticeOn(NoticeLocator{}, CommandByID(1)); got.state != nil {
			t.Fatalf("zero notice locator = %#v, want nil state", got.state)
		}
	})
}

func TestNoticeAdvisoryPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()
	got := noticeAdvisory(Notice(cli.NoticeAdvisory), cli.AdvisoryCode("warn"))
	if got.state == nil || !got.state.advisorySet || got.state.advisory != cli.AdvisoryCode("warn") {
		t.Fatalf("advisory locator = %#v", got.state)
	}
	if got := noticeAdvisory(NoticeLocator{}, cli.AdvisoryCode("warn")); got.state != nil {
		t.Fatalf("zero notice advisory = %#v, want nil state", got.state)
	}
}

func TestNoticeResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()
	flag := cli.FlagID(7)
	inspection := cli.Inspection{
		Flags: map[cli.FlagID]cli.FlagInspection{flag: {
			ID: flag,
		}},
		Notices: []cli.NoticeInspection{
			{
				Code: cli.NoticeDeprecatedFlag,
				Declaration: cli.DeclarationIdentity{
					Kind: cli.DeclarationKindFlag,
					Flag: flag,
				},
			},
			{
				Code: cli.NoticeDeprecatedFlag,
			},
		},
	}

	t.Run("notice: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		if got := resolveNotice(nil, inspection); got.status != locatorResolutionInvalid {
			t.Fatalf("nil locator = %#v", got)
		}
		state := &noticeLocatorState{}
		if got := resolveNotice(state, inspection); got.status != locatorResolutionInvalid {
			t.Fatalf("unsupported locator = %#v", got)
		}
	})
	t.Run("notice index: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := resolveNotice(NoticeAt(5).state, cli.Inspection{
			Truncated: true,
		})
		if got.status != locatorResolutionNotProvable {
			t.Fatalf("truncated index = %#v", got)
		}
	})
	t.Run("notice index: semantic absence", func(t *testing.T) {
		t.Parallel()
		got := resolveNotice(NoticeAt(5).state, cli.Inspection{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("complete missing index = %#v", got)
		}
	})
	t.Run("notice code: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveNotice(&noticeLocatorState{}, inspection)
		if got.status != locatorResolutionInvalid {
			t.Fatalf("wrong locator kind = %#v", got)
		}
	})
	t.Run("notice code: semantic result", func(t *testing.T) {
		t.Parallel()
		got := resolveNotice(Notice(cli.NoticeDeprecatedFlag).On(FlagByID(flag)).state, inspection)
		if got.status != locatorResolutionFound || got.index != 0 {
			t.Fatalf("typed subject = %#v", got)
		}
		commandInspection := cli.Inspection{
			Commands: map[cli.CommandID]cli.CommandInspection{1: {
				ID: 1,
			}, 2: {
				ID: 2,
			}},
			Notices: []cli.NoticeInspection{
				{
					Code:    cli.NoticeDeprecatedFlag,
					Command: 2,
				},
				{
					Code:    cli.NoticeDeprecatedFlag,
					Command: 1,
				},
			},
		}
		got = resolveNotice(Notice(cli.NoticeDeprecatedFlag).On(CommandByID(1)).state, commandInspection)
		if got.status != locatorResolutionFound || got.index != 1 {
			t.Fatalf("command subject filter = %#v", got)
		}
		if got := resolveNotice(
			Notice(cli.NoticeDeprecatedFlag).state,
			inspection,
		); got.status != locatorResolutionAmbiguous {
			t.Fatalf("duplicate code = %#v", got)
		}
		if got := resolveNotice(
			Notice(cli.NoticeDeprecatedFlag).Advisory(cli.AdvisoryCode("other")).state,
			inspection,
		); got.status != locatorResolutionAbsent {
			t.Fatalf("advisory filter = %#v", got)
		}
	})
	t.Run("notice code: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := resolveNotice(Notice(cli.NoticeSignalHint).state, cli.Inspection{
			Truncated: true,
		})
		if got.status != locatorResolutionNotProvable {
			t.Fatalf("truncated missing code = %#v", got)
		}
	})
	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := resolveNotice(Notice(cli.NoticeDeprecatedFlag).On(FlagByID(99)).state, cli.Inspection{
			Truncated: true,
		})
		if got.status != locatorResolutionNotProvable {
			t.Fatalf("truncated subject = %#v", got)
		}
	})
}

func TestNoticeLabelPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()
	if got := renderEvaluationField(noticeLabel(nil)); got != "notice" {
		t.Fatalf("nil label = %q", got)
	}
	if got := renderEvaluationField(noticeLabel(NoticeAt(3).state)); got != "notice at index 3" {
		t.Fatalf("index label = %q", got)
	}
	if got := renderEvaluationField(noticeLabel(Notice(cli.NoticeDeprecatedFlag).state)); got == "notice" {
		t.Fatalf("code label = %q", got)
	}
}

func TestNoticePropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()
	inspection := cli.Inspection{
		Notices: []cli.NoticeInspection{{
			Code:     cli.NoticeDeprecatedFlag,
			Severity: cli.NoticeSeverityWarning,
		}},
	}
	t.Run("resolution found: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		expectation, ok := noticeSeverityExpectation(
			Notice(cli.NoticeDeprecatedFlag),
			cli.NoticeSeverityInformation,
		).(noticeExpectation)
		if !ok {
			t.Fatal("noticeSeverityExpectation must satisfy noticeExpectation")
		}
		if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
			t.Fatalf("mismatched severity = %#v", got)
		}
		present, ok := noticePresentExpectation(Notice(cli.NoticeDeprecatedFlag)).(noticeExpectation)
		if !ok {
			t.Fatal("noticePresentExpectation must satisfy noticeExpectation")
		}
		if got := present.evaluateInspection(inspection, nil); got.status != evaluationProved {
			t.Fatalf("present notice = %#v", got)
		}
		presentButExpectedAbsent, ok := noticeAbsentExpectation(Notice(cli.NoticeDeprecatedFlag)).(noticeExpectation)
		if !ok {
			t.Fatal("noticeAbsentExpectation must satisfy noticeExpectation")
		}
		if got := presentButExpectedAbsent.evaluateInspection(
			inspection,
			nil,
		); got.status != evaluationDisproved {
			t.Fatalf("present notice expected absent = %#v", got)
		}
		absent, ok := noticeAbsentExpectation(Notice(cli.NoticeSignalHint)).(noticeExpectation)
		if !ok {
			t.Fatal("noticeAbsentExpectation must satisfy noticeExpectation")
		}
		if got := absent.evaluateInspection(inspection, nil); got.status != evaluationProved {
			t.Fatalf("absent notice = %#v", got)
		}
		if got := absent.evaluateInspection(cli.Inspection{
			Truncated: true,
		}, nil); got.status != evaluationNotProvable {
			t.Fatalf("truncated absence = %#v", got)
		}
	})
}

func TestNoticeReplacementPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()
	flag := cli.FlagID(1)
	replacement := cli.FlagID(2)
	locator := Notice(cli.NoticeDeprecatedFlag).On(FlagByID(flag))

	t.Run("expected subject unresolved: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		expectation, ok := noticeReplacementExpectation(locator, nil).(noticeExpectation)
		if !ok {
			t.Fatal("noticeReplacementExpectation must satisfy noticeExpectation")
		}
		inspection := cli.Inspection{
			Flags: map[cli.FlagID]cli.FlagInspection{flag: {
				ID: flag,
			}},
			Notices: []cli.NoticeInspection{{
				Code: cli.NoticeDeprecatedFlag,
				Declaration: cli.DeclarationIdentity{
					Kind: cli.DeclarationKindFlag,
					Flag: flag,
				},
			}},
		}
		if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
			t.Fatalf("invalid replacement locator = %#v", got)
		}
	})
	t.Run("expected subject unresolved: semantic absence", func(t *testing.T) {
		t.Parallel()
		expectation, ok := noticeReplacementExpectation(locator, FlagByID(replacement)).(noticeExpectation)
		if !ok {
			t.Fatal("noticeReplacementExpectation must satisfy noticeExpectation")
		}
		inspection := cli.Inspection{
			Flags: map[cli.FlagID]cli.FlagInspection{flag: {
				ID: flag,
			}},
			Notices: []cli.NoticeInspection{{
				Code: cli.NoticeDeprecatedFlag,
				Declaration: cli.DeclarationIdentity{
					Kind: cli.DeclarationKindFlag,
					Flag: flag,
				},
			}},
		}
		if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
			t.Fatalf("missing replacement subject = %#v", got)
		}
	})
	t.Run("replacement is absent: semantic absence", func(t *testing.T) {
		t.Parallel()
		expectation, ok := noticeReplacementExpectation(locator, FlagByID(replacement)).(noticeExpectation)
		if !ok {
			t.Fatal("noticeReplacementExpectation must satisfy noticeExpectation")
		}
		inspection := cli.Inspection{
			Flags: map[cli.FlagID]cli.FlagInspection{flag: {
				ID: flag,
			}, replacement: {
				ID: replacement,
			}},
			Notices: []cli.NoticeInspection{{
				Code: cli.NoticeDeprecatedFlag,
				Declaration: cli.DeclarationIdentity{
					Kind: cli.DeclarationKindFlag,
					Flag: flag,
				},
			}},
		}
		if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
			t.Fatalf("absent replacement = %#v", got)
		}
	})
}

func TestNoticeSuppressedByPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()
	command := cli.CommandID(1)
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			command: {
				ID: command,
				NoticeSuppressions: []cli.NoticeSuppressionInspection{{
					Command:          command,
					DeclarationIndex: 0,
				}},
			},
		},
		Notices: []cli.NoticeInspection{{
			Code: cli.NoticeAdvisory,
			SuppressedBy: []cli.NoticeSuppressionLocator{{
				Command:          command,
				DeclarationIndex: 0,
			}},
		}},
	}
	expectation, ok := noticeSuppressedByExpectation(
		Notice(cli.NoticeAdvisory),
		CommandByID(command).NoticeSuppressionAt(0),
	).(noticeExpectation)
	if !ok {
		t.Fatal("noticeSuppressedByExpectation must satisfy noticeExpectation")
	}
	if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationProved {
		t.Fatalf("matching suppressions = %#v", got)
	}
	bad, ok := noticeSuppressedByExpectation(
		Notice(cli.NoticeAdvisory),
		CommandByID(command).NoticeSuppressionAt(2),
	).(noticeExpectation)
	if !ok {
		t.Fatal("noticeSuppressedByExpectation must satisfy noticeExpectation")
	}
	if got := bad.evaluateInspection(inspection, nil); got.status != evaluationNotProvable {
		t.Fatalf("unresolvable suppression = %#v", got)
	}
}

func TestNoticesExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()
	first := NoticeAt(0)
	second := NoticeAt(1)
	expectation, ok := noticesExactlyExpectation(first, second).(noticesExactlyExpectationValue)
	if !ok {
		t.Fatal("noticesExactlyExpectation must satisfy noticesExactlyExpectationValue")
	}

	t.Run("truncated inspection: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated: true,
			Notices:   []cli.NoticeInspection{{}, {}},
		}
		if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationNotProvable {
			t.Fatalf("truncated exact notices = %#v", got)
		}
	})
	t.Run("notices: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Notices: []cli.NoticeInspection{{}, {}, {}},
		}
		if got := expectation.evaluateInspection(inspection, nil); got.status != evaluationDisproved {
			t.Fatalf("non-exact notices = %#v", got)
		}
		checkedGot, ok := noticesExactlyExpectation(first, second, NoticeAt(2)).(noticesExactlyExpectationValue)
		if !ok {
			t.Fatal("noticesExactlyExpectation must satisfy noticesExactlyExpectationValue")
		}
		if got := checkedGot.evaluateInspection(
			inspection,
			nil,
		); got.status != evaluationProved {
			t.Fatalf("exact notices = %#v", got)
		}
	})
}

func TestNoticeSubjectInternalStateFailsClosed(t *testing.T) {
	t.Parallel()

	if _, status, _ := resolveNoticeSubject(nil, cli.Inspection{}); status != locatorResolutionInvalid {
		t.Fatalf("nil subject status = %v, want invalid", status)
	}
	if _, status, _ := resolveNoticeSubject(CommandLocator{}, cli.Inspection{}); status != locatorResolutionInvalid {
		t.Fatalf("invalid declaration subject status = %v, want invalid", status)
	}
}

func TestNoticeInspectionExpectationsConserveFilteringAndReplacementFailures(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Notices: []cli.NoticeInspection{{
			Code: cli.NoticeDeprecatedFlag,
		}, {
			Code: cli.NoticeDeprecatedFlag,
		}},
	}
	state := *Notice(cli.NoticeDeprecatedFlag).state
	state.advisorySet = true
	state.advisory = cli.AdvisoryCode("other")
	if got := resolveNotice(&state, inspection); got.status != locatorResolutionAbsent {
		t.Fatalf("advisory-filtered notice = %#v", got)
	}

	property := noticeExpectation{
		locator:  NoticeAt(0).state,
		kind:     noticeExpectationSeverity,
		severity: cli.NoticeSeverityWarning,
	}
	propertyInspection := cli.Inspection{
		Notices: []cli.NoticeInspection{{}},
	}
	if got := property.evaluateInspection(propertyInspection, nil); got.status != evaluationDisproved {
		t.Fatalf("notice property mismatch = %#v", got)
	}

	replacement, ok := noticeReplacementExpectation(NoticeAt(0), FlagByID(9)).(noticeExpectation)
	if !ok {
		t.Fatal("noticeReplacementExpectation must satisfy noticeExpectation")
	}
	replacementInspection := cli.Inspection{
		Notices: []cli.NoticeInspection{{}},
	}
	if got := replacement.evaluateInspection(replacementInspection, nil); got.status != evaluationDisproved {
		t.Fatalf("missing replacement coordinate = %#v", got)
	}
}
