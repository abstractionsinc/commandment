package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestRelationalValueLocatorsResolveStaticAndReferenceValues(t *testing.T) {
	t.Parallel()

	flagID := cli.FlagID(11)
	termID := cli.ArgumentTermID(12)
	environmentID := cli.EnvironmentInputID(13)
	factID := cli.ValueID(14)
	flagValueID := cli.ValueID(21)
	termValueID := cli.ValueID(22)
	environmentValueID := cli.ValueID(23)

	values := map[cli.ValueID]cli.ValueSchema{
		flagValueID: {
			ID:        flagValueID,
			Required:  true,
			Sensitive: true,
		},
		termValueID: {
			ID:        termValueID,
			Required:  true,
			Sensitive: true,
		},
		environmentValueID: {
			ID:        environmentValueID,
			Required:  true,
			Sensitive: true,
		},
		factID: {
			ID:        factID,
			Required:  true,
			Sensitive: true,
		},
	}
	schema := &cli.Schema{
		Flags: map[cli.FlagID]cli.FlagSchema{flagID: {
			ID:          flagID,
			Destination: flagValueID,
		}},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
			termID: {
				ID:          termID,
				Destination: termValueID,
			},
		},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{
			environmentID: {
				ID:          environmentID,
				Destination: environmentValueID,
			},
		},
		Values: values,
	}
	referenceValues := make(map[cli.ValueID]cli.ValueReference, len(values))
	for id, value := range values {
		referenceValues[id] = cli.ValueReference{
			ValueSchema: value,
			Available:   true,
			Visible:     true,
		}
	}
	reference := cli.Reference{
		Flags: map[cli.FlagID]cli.FlagReference{
			flagID: {
				ID:          flagID,
				Destination: flagValueID,
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermReference{
			termID: {
				ID:          termID,
				Destination: termValueID,
			},
		},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputReference{
			environmentID: {EnvironmentInputSchema: cli.EnvironmentInputSchema{
				ID:          environmentID,
				Destination: environmentValueID,
			}},
		},
		Values: referenceValues,
	}

	cases := []struct {
		name    string
		locator ValueLocator
		valueID cli.ValueID
	}{
		{
			name:    "flag",
			locator: FlagByID(flagID).Value(),
			valueID: flagValueID,
		},
		{
			name:    "argument term",
			locator: ArgumentTermByID(termID).Value(),
			valueID: termValueID,
		},
		{
			name:    "environment",
			locator: EnvironmentInputByID(environmentID).Value(),
			valueID: environmentValueID,
		},
		{
			name:    "fact",
			locator: FactByID(factID).Value(),
			valueID: factID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolvedSchema := resolveSchemaValueLocator(tc.locator.state, schema)
			if resolvedSchema.status != locatorResolutionFound || resolvedSchema.value.ID != tc.valueID {
				t.Fatalf("schema value = (%+v, %v, %q), want ID %d", resolvedSchema.value, resolvedSchema.status, resolvedSchema.subject, tc.valueID)
			}
			resolvedReference := resolveReferenceValueLocator(tc.locator.state, reference)
			if resolvedReference.status != locatorResolutionFound || resolvedReference.value.ID != tc.valueID {
				t.Fatalf("reference value = (%+v, %v, %q), want ID %d", resolvedReference.value, resolvedReference.status, resolvedReference.subject, tc.valueID)
			}
			for _, expectation := range []StaticDeclarationExpectation{
				tc.locator.Exists(),
				tc.locator.Required(),
				tc.locator.DeclaredSensitive(),
			} {
				if got := schemaExpectationOutcome(t, expectation, schema); got.status != evaluationProved {
					t.Fatalf("schema %T = %+v", expectation, got)
				}
				if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
					t.Fatalf("reference %T = %+v", expectation, got)
				}
			}
			for _, expectation := range []ReferenceExpectation{tc.locator.Available(), tc.locator.Visible()} {
				if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
					t.Fatalf("reference %T = %+v", expectation, got)
				}
			}
		})
	}
}

func TestRelationalValueLocatorsReportMissingDestinationValues(t *testing.T) {
	t.Parallel()

	flagID := cli.FlagID(31)
	termID := cli.ArgumentTermID(32)
	environmentID := cli.EnvironmentInputID(33)
	factID := cli.ValueID(34)
	schema := &cli.Schema{
		Flags: map[cli.FlagID]cli.FlagSchema{flagID: {
			ID:          flagID,
			Destination: 41,
		}},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{termID: {
			ID:          termID,
			Destination: 42,
		}},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{
			environmentID: {
				ID:          environmentID,
				Destination: 43,
			},
		},
		Values: map[cli.ValueID]cli.ValueSchema{},
	}
	reference := cli.Reference{
		Flags: map[cli.FlagID]cli.FlagReference{flagID: {
			ID:          flagID,
			Destination: 41,
		}},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermReference{termID: {
			ID:          termID,
			Destination: 42,
		}},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputReference{
			environmentID: {EnvironmentInputSchema: cli.EnvironmentInputSchema{
				ID:          environmentID,
				Destination: 43,
			}},
		},
		Values: map[cli.ValueID]cli.ValueReference{},
	}
	for _, locator := range []ValueLocator{
		FlagByID(flagID).Value(),
		ArgumentTermByID(termID).Value(),
		EnvironmentInputByID(environmentID).Value(),
		FactByID(factID).Value(),
	} {
		if got := schemaExpectationOutcome(t, locator.DoesNotExist(), schema); got.status != evaluationProved {
			t.Fatalf("schema missing %v = %+v", locator.state.mode, got)
		}
		if got := referenceExpectationOutcome(
			t,
			locator.DoesNotExist(),
			reference,
		); got.status != evaluationProved {
			t.Fatalf("reference missing %v = %+v", locator.state.mode, got)
		}
	}
}
