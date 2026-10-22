package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestTypedCommandResolutionDistinguishesAllStatuses(t *testing.T) {
	t.Parallel()

	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			1: {
				ID:       1,
				Children: []cli.CommandID{2, 3},
			},
			2: {
				ID:   2,
				Name: "child",
			},
			3: {
				ID:   3,
				Name: "child",
			},
		},
	}

	if resolution := resolveSchemaCommandLocator(CommandByID(1).state, &schema); resolution.status != locatorResolutionFound || resolution.value.ID != 1 {
		t.Fatalf("found resolution = %+v", resolution)
	}
	if resolution := resolveSchemaCommandLocator(CommandByID(99).state, &schema); resolution.status != locatorResolutionAbsent {
		t.Fatalf("absent resolution = %+v", resolution)
	}
	if resolution := resolveSchemaCommandLocator(CommandByID(1).Command("child").state, &schema); resolution.status != locatorResolutionAmbiguous {
		t.Fatalf("ambiguous resolution = %+v", resolution)
	}
	if resolution := resolveSchemaCommandLocator(nil, &schema); resolution.status != locatorResolutionInvalid {
		t.Fatalf("invalid resolution = %+v", resolution)
	}

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID: 1,
			},
		},
	}
	if resolution := resolveInspectionCommandLocator(CommandByID(1).Command("missing").state, inspection); resolution.status != locatorResolutionNotProvable {
		t.Fatalf("not-provable resolution = %+v", resolution)
	}
}

func TestResolveUniqueObservedRequiresClosureOnlyForAbsence(t *testing.T) {
	t.Parallel()

	subject := evaluationText("subject")
	absent := evaluationText("absent")
	incomplete := evaluationText("incomplete")
	ambiguous := evaluationText("ambiguous")

	found := resolveUniqueObserved([]int{7}, false, subject, absent, incomplete, ambiguous)
	if found.status != locatorResolutionFound || found.value != 7 {
		t.Fatalf("found = %+v", found)
	}

	notProvable := resolveUniqueObserved([]int(nil), false, subject, absent, incomplete, ambiguous)
	if notProvable.status != locatorResolutionNotProvable || renderEvaluationField(notProvable.detail) != "incomplete" {
		t.Fatalf("not provable = %+v", notProvable)
	}

	missing := resolveUniqueObserved([]int(nil), true, subject, absent, incomplete, ambiguous)
	if missing.status != locatorResolutionAbsent || renderEvaluationField(missing.detail) != "absent" {
		t.Fatalf("absent = %+v", missing)
	}

	multipleMatches := []int{1, 2}
	multiple := resolveUniqueObserved(multipleMatches, false, subject, absent, incomplete, ambiguous)
	if multiple.status != locatorResolutionAmbiguous || renderEvaluationField(multiple.detail) != "ambiguous" {
		t.Fatalf("ambiguous = %+v", multiple)
	}
}
