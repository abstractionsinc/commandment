package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestValueDomainEntryTypedResolution(t *testing.T) {
	t.Parallel()

	valueID := cli.ValueID(1)
	entryID := cli.ValueDomainEntryID(2)
	use := cli.ValueDomainUse{
		Destination: valueID,
		Kind:        cli.ValueDomainUseDestination,
	}
	domain := cli.ValueDomainSchema{
		Uses:    []cli.ValueDomainUse{use},
		Entries: []cli.ValueDomainEntryID{entryID},
	}
	schema := &cli.Schema{
		Values: map[cli.ValueID]cli.ValueSchema{valueID: {
			ID: valueID,
		}},
		ValueDomains: []cli.ValueDomainSchema{domain},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{entryID: {
			ID:   entryID,
			Name: "fast",
		}},
	}

	locator := ValueByID(valueID).Domain(cli.ValueDomainUseDestination).Entry("fast")
	resolution := resolveSchemaValueDomainEntryLocator(locator.state, schema)
	if resolution.status != locatorResolutionFound || resolution.value.ID != entryID {
		t.Fatalf("name resolution = %#v", resolution)
	}
	if resolution := resolveSchemaValueDomainEntryLocator(ValueDomainEntryByID(entryID).state, schema); resolution.status != locatorResolutionFound || resolution.value.ID != entryID {
		t.Fatalf("ID resolution = %#v", resolution)
	}
	if resolution := resolveSchemaValueDomainEntryLocator(ValueDomainEntryLocator{}.state, schema); resolution.status != locatorResolutionInvalid {
		t.Fatalf("zero locator = %#v", resolution)
	}
}

func TestValueDomainEntryExistenceAndExactOrder(t *testing.T) {
	t.Parallel()

	valueID := cli.ValueID(1)
	first := cli.ValueDomainEntryID(2)
	second := cli.ValueDomainEntryID(3)
	use := cli.ValueDomainUse{
		Destination: valueID,
		Kind:        cli.ValueDomainUseDestination,
	}
	schema := &cli.Schema{
		Values: map[cli.ValueID]cli.ValueSchema{valueID: {
			ID: valueID,
		}},
		ValueDomains: []cli.ValueDomainSchema{{
			Uses:    []cli.ValueDomainUse{use},
			Entries: []cli.ValueDomainEntryID{first, second},
		}},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{
			first: {
				ID:   first,
				Name: "first",
			},
			second: {
				ID:   second,
				Name: "second",
			},
		},
	}

	domain := ValueByID(valueID).Domain(cli.ValueDomainUseDestination)
	for _, expectation := range []SchemaExpectation{
		ValueDomainEntryByID(first).Exists(),
		domain.EntriesExactly(ValueDomainEntryByID(first), ValueDomainEntryByID(second)),
	} {
		got := requireSchemaEvaluator(t, expectation).evaluateSchema(schema)
		if got.status != evaluationProved {
			t.Fatalf("expectation = %#v", got)
		}
	}

	mismatch := requireSchemaEvaluator(t, domain.EntriesExactly(ValueDomainEntryByID(second), ValueDomainEntryByID(first))).evaluateSchema(schema)
	if mismatch.status != evaluationDisproved {
		t.Fatalf("order mismatch = %#v", mismatch)
	}
}

func TestValueDomainEntryInspectionAbsenceRequiresClosure(t *testing.T) {
	t.Parallel()

	entryID := cli.ValueDomainEntryID(9)
	expectation := requireInspectionEvaluator(t, ValueDomainEntryByID(entryID).DoesNotExist())
	if got := expectation.evaluateInspection(cli.Inspection{
		Truncated: true,
	}, nil); got.status != evaluationNotProvable {
		t.Fatalf("truncated absence = %#v", got)
	}
	if got := expectation.evaluateInspection(cli.Inspection{}, nil); got.status != evaluationProved {
		t.Fatalf("closed absence = %#v", got)
	}
}

func TestValueDomainEntryEffectiveProperties(t *testing.T) {
	t.Parallel()

	entryID := cli.ValueDomainEntryID(2)
	reference := cli.Reference{
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryReference{
			entryID: {
				ID:        entryID,
				Available: true,
				Visible:   false,
			},
		},
	}
	if got := requireReferenceEvaluator(t, ValueDomainEntryByID(entryID).Available()).evaluateReference(reference, nil); got.status != evaluationProved {
		t.Fatalf("available = %#v", got)
	}
	if got := requireReferenceEvaluator(t, ValueDomainEntryByID(entryID).NotVisible()).evaluateReference(reference, nil); got.status != evaluationProved {
		t.Fatalf("not visible = %#v", got)
	}
}
