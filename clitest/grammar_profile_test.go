package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestGrammarProfileTypedResolution(t *testing.T) {
	t.Parallel()

	commandID := cli.CommandID(1)
	profileID := cli.GrammarProfileID(2)
	profile := cli.GrammarProfile{
		ID:           profileID,
		Value:        "strict",
		Reachability: cli.Reachability(1),
	}
	set := &cli.GrammarProfileSet{
		Cases:    []cli.GrammarProfileID{profileID},
		Profiles: map[cli.GrammarProfileID]cli.GrammarProfile{profileID: profile},
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{commandID: {
			ID:       commandID,
			Profiles: set,
		}},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{commandID: {
			ID:       commandID,
			Profiles: set,
		}},
	}

	for _, test := range []struct {
		name       string
		resolution locatorResolution[cli.GrammarProfile]
	}{
		{
			name:       "schema by ID",
			resolution: resolveSchemaGrammarProfileLocator(GrammarProfileByID(profileID).state, schema),
		},
		{
			name:       "schema by value",
			resolution: resolveSchemaGrammarProfileLocator(RootCommand().GrammarProfile("strict").state, schema),
		},
		{
			name:       "reference by ID",
			resolution: resolveReferenceGrammarProfileLocator(GrammarProfileByID(profileID).state, reference),
		},
	} {
		name := test.name
		resolution := test.resolution
		if resolution.status != locatorResolutionFound || resolution.value.ID != profileID {
			t.Fatalf("%s = %#v", name, resolution)
		}
	}

	if resolution := resolveSchemaGrammarProfileLocator(GrammarProfileLocator{}.state, schema); resolution.status != locatorResolutionInvalid {
		t.Fatalf("zero locator = %#v", resolution)
	}
	if resolution := resolveSchemaGrammarProfileLocator(RootCommand().GrammarProfileAt(-1).state, schema); resolution.status != locatorResolutionInvalid {
		t.Fatalf("negative index = %#v", resolution)
	}
}

func TestGrammarProfileTypedExpectations(t *testing.T) {
	t.Parallel()

	profileID := cli.GrammarProfileID(2)
	reachability := cli.Reachability(1)
	set := &cli.GrammarProfileSet{
		Cases: []cli.GrammarProfileID{profileID},
		Profiles: map[cli.GrammarProfileID]cli.GrammarProfile{
			profileID: {
				ID:           profileID,
				Reachability: reachability,
			},
		},
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{1: {
			ID:       1,
			Profiles: set,
		}},
	}

	for _, expectation := range []SchemaExpectation{
		GrammarProfileByID(profileID).Exists(),
		GrammarProfileByID(profileID).Reachability(reachability),
	} {
		evaluator := requireSchemaEvaluator(t, expectation)
		if got := evaluator.evaluateSchema(schema); got.status != evaluationProved {
			t.Fatalf("expectation = %#v", got)
		}
	}
}
