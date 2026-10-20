package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestValueDomainRelationalLocatorsCanonicalizeAcrossArtifacts(t *testing.T) {
	t.Parallel()

	const (
		rootID      cli.CommandID          = 1
		flagID      cli.FlagID             = 2
		spellingID  cli.FlagSpellingID     = 3
		valueID     cli.ValueID            = 4
		referenceID cli.ValueDomainEntryID = 5
		modernID    cli.ValueDomainEntryID = 6
	)

	use := cli.ValueDomainUse{
		Destination: valueID,
		Kind:        cli.ValueDomainUseDestination,
	}
	replacement := &cli.DeclarationReplacement{
		Target: cli.DeclarationIdentity{
			Kind:             cli.DeclarationKindValueDomainEntry,
			ValueDomainEntry: modernID,
		},
	}

	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			rootID: {
				ID:    rootID,
				Path:  []string{},
				Flags: []cli.FlagID{flagID},
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{
			flagID: {
				ID:          flagID,
				Command:     rootID,
				Destination: valueID,
				Spellings:   []cli.FlagSpellingID{spellingID},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
			spellingID: {
				ID:   spellingID,
				Flag: flagID,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
		ValueDomains: []cli.ValueDomainSchema{
			{
				Uses:    []cli.ValueDomainUse{use},
				Entries: []cli.ValueDomainEntryID{referenceID, modernID},
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{
			referenceID: {
				ID:          referenceID,
				Name:        "reference",
				Kind:        cli.ValueDomainEntryAlias,
				Canonical:   modernID,
				Replacement: replacement,
			},
			modernID: {
				ID:        modernID,
				Name:      "modern",
				Kind:      cli.ValueDomainEntryCanonical,
				Canonical: modernID,
			},
		},
	}

	reference := cli.Reference{
		Commands:           map[cli.CommandID]cli.CommandReference{},
		Flags:              map[cli.FlagID]cli.FlagReference{},
		FlagSpellings:      map[cli.FlagSpellingID]cli.FlagSpellingReference{},
		ValueDomains:       []cli.ValueDomainReference{{Uses: []cli.ValueDomainUse{use}, Entries: []cli.ValueDomainEntryID{referenceID, modernID}}},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryReference{},
	}
	schemaCommand := schema.Commands[rootID]
	reference.Commands[rootID] = cli.CommandReference{
		ID:    schemaCommand.ID,
		Path:  schemaCommand.Path,
		Flags: schemaCommand.Flags,
	}
	schemaFlag := schema.Flags[flagID]
	reference.Flags[flagID] = cli.FlagReference{
		ID:          schemaFlag.ID,
		Command:     schemaFlag.Command,
		Destination: schemaFlag.Destination,
		Spellings:   schemaFlag.Spellings,
	}
	schemaSpelling := schema.FlagSpellings[spellingID]
	reference.FlagSpellings[spellingID] = cli.FlagSpellingReference{
		ID:       schemaSpelling.ID,
		Flag:     schemaSpelling.Flag,
		Spelling: schemaSpelling.Spelling,
	}
	for id, entry := range schema.ValueDomainEntries {
		reference.ValueDomainEntries[id] = cli.ValueDomainEntryReference{
			ID:          entry.ID,
			Name:        entry.Name,
			Kind:        entry.Kind,
			Canonical:   entry.Canonical,
			Replacement: entry.Replacement,
		}
	}

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				Flags:                 []cli.FlagID{flagID},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			flagID: {
				ID:          flagID,
				Command:     rootID,
				Destination: valueID,
				Spellings:   []cli.FlagSpellingID{spellingID},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			spellingID: {
				ID:   spellingID,
				Flag: flagID,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
		Values: map[cli.ValueID]cli.ValueInspection{
			valueID: {ID: valueID},
		},
		ValueDomains: []cli.ValueDomainInspection{
			{
				Uses:    []cli.ValueDomainUse{use},
				Entries: []cli.ValueDomainEntryID{referenceID, modernID},
				Outcome: cli.ValueDomainOutcomeReady,
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{
			referenceID: {
				ID:          referenceID,
				Name:        "reference",
				Kind:        cli.ValueDomainEntryAlias,
				Canonical:   modernID,
				Replacement: replacement,
			},
			modernID: {
				ID:        modernID,
				Name:      "modern",
				Kind:      cli.ValueDomainEntryCanonical,
				Canonical: modernID,
			},
		},
		Notices: []cli.NoticeInspection{{
			Code: cli.NoticeDeprecatedValue,
			Declaration: cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindValueDomainEntry,
				ValueDomainEntry: referenceID,
			},
			Replacement: replacement,
		}},
	}

	directDomain := ValueByID(valueID).Domain(cli.ValueDomainUseDestination)
	relationalDomain := RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	directEntry := directDomain.Entry("reference")
	relationalEntry := relationalDomain.Entry("reference")
	relationalEntryAt := relationalDomain.EntryAt(0)
	modernEntry := relationalDomain.Entry("modern")

	schemaDomain := resolveSchemaValueDomainLocator(relationalDomain.state, schema)
	if schemaDomain.status != locatorResolutionFound || len(schemaDomain.value.Entries) != 2 {
		t.Fatalf("schema relational domain = %#v, want found with two entries", schemaDomain)
	}
	referenceDomain := resolveReferenceValueDomainLocator(relationalDomain.state, reference)
	if referenceDomain.status != locatorResolutionFound || len(referenceDomain.value.Entries) != 2 {
		t.Fatalf("reference relational domain = %#v, want found with two entries", referenceDomain)
	}
	inspectionDomain := resolveInspectionValueDomainLocator(relationalDomain.state, inspection)
	if inspectionDomain.status != locatorResolutionFound || len(inspectionDomain.value.Entries) != 2 {
		t.Fatalf("inspection relational domain = %#v, want found with two entries", inspectionDomain)
	}

	for name, resolution := range map[string]locatorResolution[cli.ValueDomainEntrySchema]{
		"schema name":  resolveSchemaValueDomainEntryLocator(relationalEntry.state, &schema),
		"schema index": resolveSchemaValueDomainEntryLocator(relationalEntryAt.state, &schema),
	} {
		if resolution.status != locatorResolutionFound || resolution.value.ID != referenceID {
			t.Fatalf("%s = %#v, want entry %d", name, resolution, referenceID)
		}
	}
	for name, resolution := range map[string]locatorResolution[cli.ValueDomainEntryReference]{
		"reference name":  resolveReferenceValueDomainEntryLocator(relationalEntry.state, reference),
		"reference index": resolveReferenceValueDomainEntryLocator(relationalEntryAt.state, reference),
	} {
		if resolution.status != locatorResolutionFound || resolution.value.ID != referenceID {
			t.Fatalf("%s = %#v, want entry %d", name, resolution, referenceID)
		}
	}
	for name, resolution := range map[string]locatorResolution[cli.ValueDomainEntryInspection]{
		"inspection name":  resolveInspectionValueDomainEntryLocator(relationalEntry.state, inspection),
		"inspection index": resolveInspectionValueDomainEntryLocator(relationalEntryAt.state, inspection),
	} {
		if resolution.status != locatorResolutionFound || resolution.value.ID != referenceID {
			t.Fatalf("%s = %#v, want entry %d", name, resolution, referenceID)
		}
	}

	for _, test := range []struct {
		name        string
		expectation DeclarationExpectation
	}{
		{"domain exists", relationalDomain.Exists()},
		{"domain entries", relationalDomain.EntriesExactly(directEntry, modernEntry)},
		{"entry exists", relationalEntry.Exists()},
		{"entry index exists", relationalEntryAt.Exists()},
		{"entry replacement", relationalEntry.ReplacedBy(modernEntry)},
	} {
		if got := schemaExpectationOutcome(t, test.expectation, &schema); got.status != evaluationProved {
			t.Fatalf("schema %s = %#v, want satisfied", test.name, got)
		}
		if got := referenceExpectationOutcome(t, test.expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference %s = %#v, want satisfied", test.name, got)
		}
		if got := inspectionExpectationOutcome(t, test.expectation, inspection); got.status != evaluationProved {
			t.Fatalf("inspection %s = %#v, want satisfied", test.name, got)
		}
	}

	if got := inspectionExpectationOutcome(
		t,
		relationalDomain.Outcome(cli.ValueDomainOutcomeReady),
		inspection,
	); got.status != evaluationProved {
		t.Fatalf("inspection relational domain status = %#v, want satisfied", got)
	}
	deprecated := Notice(cli.NoticeDeprecatedValue).On(relationalEntry)
	if got := inspectionExpectationOutcome(t, deprecated.Present(), inspection); got.status != evaluationProved {
		t.Fatalf("notice relational subject = %#v, want satisfied", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		deprecated.ReplacedBy(modernEntry),
		inspection,
	); got.status != evaluationProved {
		t.Fatalf("notice relational replacement = %#v, want satisfied", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		NoticesExactly(deprecated),
		inspection,
	); got.status != evaluationProved {
		t.Fatalf("exact notice collection with relational subject = %#v, want satisfied", got)
	}

	directSchemaDomain := resolveSchemaValueDomainLocator(directDomain.state, schema)
	if directSchemaDomain.status != locatorResolutionFound || directSchemaDomain.value.Entries[0] != schemaDomain.value.Entries[0] {
		t.Fatalf("schema direct/relational domain mismatch: %#v vs %#v", directSchemaDomain, schemaDomain)
	}
	directReferenceDomain := resolveReferenceValueDomainLocator(directDomain.state, reference)
	if directReferenceDomain.status != locatorResolutionFound || directReferenceDomain.value.Entries[0] != referenceDomain.value.Entries[0] {
		t.Fatalf("reference direct/relational domain mismatch: %#v vs %#v", directReferenceDomain, referenceDomain)
	}
	directInspectionDomain := resolveInspectionValueDomainLocator(directDomain.state, inspection)
	if directInspectionDomain.status != locatorResolutionFound || directInspectionDomain.value.Entries[0] != inspectionDomain.value.Entries[0] {
		t.Fatalf("inspection direct/relational domain mismatch: %#v vs %#v", directInspectionDomain, inspectionDomain)
	}

	directSchemaEntry := resolveSchemaValueDomainEntryLocator(directEntry.state, &schema)
	relationalSchemaEntry := resolveSchemaValueDomainEntryLocator(relationalEntry.state, &schema)
	if directSchemaEntry.status != locatorResolutionFound || relationalSchemaEntry.status != locatorResolutionFound || directSchemaEntry.value.ID != relationalSchemaEntry.value.ID {
		t.Fatalf("schema direct/relational entry mismatch: %#v vs %#v", directSchemaEntry, relationalSchemaEntry)
	}
	directReferenceEntry := resolveReferenceValueDomainEntryLocator(directEntry.state, reference)
	relationalReferenceEntry := resolveReferenceValueDomainEntryLocator(relationalEntry.state, reference)
	if directReferenceEntry.status != locatorResolutionFound || relationalReferenceEntry.status != locatorResolutionFound || directReferenceEntry.value.ID != relationalReferenceEntry.value.ID {
		t.Fatalf("reference direct/relational entry mismatch: %#v vs %#v", directReferenceEntry, relationalReferenceEntry)
	}
	directInspectionEntry := resolveInspectionValueDomainEntryLocator(directEntry.state, inspection)
	relationalInspectionEntry := resolveInspectionValueDomainEntryLocator(relationalEntry.state, inspection)
	if directInspectionEntry.status != locatorResolutionFound || relationalInspectionEntry.status != locatorResolutionFound || directInspectionEntry.value.ID != relationalInspectionEntry.value.ID {
		t.Fatalf("inspection direct/relational entry mismatch: %#v vs %#v", directInspectionEntry, relationalInspectionEntry)
	}
}

func TestInspectionValueDomainRelationalLocatorPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	const rootID cli.CommandID = 1
	relational := RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)

	missing := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
	}
	if resolution := resolveInspectionValueDomainLocator(relational.state, missing); resolution.status != locatorResolutionAbsent {
		t.Fatalf("missing relational domain = %#v, want absent", resolution)
	}

	ambiguous := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				Flags:                 []cli.FlagID{2, 3},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			2: {
				ID:        2,
				Spellings: []cli.FlagSpellingID{4},
			},
			3: {
				ID:        3,
				Spellings: []cli.FlagSpellingID{5},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			4: {
				ID: 4,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
			5: {
				ID: 5,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
	}
	if resolution := resolveInspectionValueDomainLocator(relational.state, ambiguous); resolution.status != locatorResolutionAmbiguous {
		t.Fatalf("ambiguous relational domain = %#v, want ambiguous", resolution)
	}

	partial := cli.Inspection{
		Truncated: true,
	}
	if resolution := resolveInspectionValueDomainLocator(relational.state, partial); resolution.status != locatorResolutionNotProvable {
		t.Fatalf("partial relational domain = %#v, want not provable", resolution)
	}
}

func TestValueDomainNegativeExistencePreservesRelationalResolutionStatus(t *testing.T) {
	t.Parallel()

	const rootID cli.CommandID = 1
	relationalDomain := RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	relationalEntry := relationalDomain.Entry("legacy")

	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			rootID: {
				ID:    rootID,
				Path:  []string{},
				Flags: []cli.FlagID{2, 3},
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{
			2: {
				ID:          2,
				Command:     rootID,
				Destination: 4,
				Spellings:   []cli.FlagSpellingID{6},
			},
			3: {
				ID:          3,
				Command:     rootID,
				Destination: 5,
				Spellings:   []cli.FlagSpellingID{7},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
			6: {
				ID:   6,
				Flag: 2,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
			7: {
				ID:   7,
				Flag: 3,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			rootID: {
				ID:    rootID,
				Path:  []string{},
				Flags: []cli.FlagID{2, 3},
			},
		},
		Flags: map[cli.FlagID]cli.FlagReference{
			2: {
				ID:          2,
				Command:     rootID,
				Destination: 4,
				Spellings:   []cli.FlagSpellingID{6},
			},
			3: {
				ID:          3,
				Command:     rootID,
				Destination: 5,
				Spellings:   []cli.FlagSpellingID{7},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingReference{
			6: {
				ID:   6,
				Flag: 2,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
			7: {
				ID:   7,
				Flag: 3,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				Flags:                 []cli.FlagID{2, 3},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			2: {
				ID:          2,
				Command:     rootID,
				Destination: 4,
				Spellings:   []cli.FlagSpellingID{6},
			},
			3: {
				ID:          3,
				Command:     rootID,
				Destination: 5,
				Spellings:   []cli.FlagSpellingID{7},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			6: {
				ID:   6,
				Flag: 2,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
			7: {
				ID:   7,
				Flag: 3,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
	}

	for _, expectation := range []DeclarationExpectation{relationalDomain.DoesNotExist(), relationalEntry.DoesNotExist()} {
		if got := schemaExpectationOutcome(
			t,
			expectation,
			&schema,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "matches") {
			t.Fatalf("schema ambiguous %T = %#v, want disproved ambiguity", expectation, got)
		}
		if got := referenceExpectationOutcome(
			t,
			expectation,
			reference,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "matches") {
			t.Fatalf("reference ambiguous %T = %#v, want disproved ambiguity", expectation, got)
		}
		if got := inspectionExpectationOutcome(
			t,
			expectation,
			inspection,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "matches") {
			t.Fatalf("inspection ambiguous %T = %#v, want disproved ambiguity", expectation, got)
		}
	}

	invalidEntry := (ValueLocator{}).Domain(cli.ValueDomainUseDestination).Entry("legacy")
	for _, got := range []evaluationResult{
		schemaExpectationOutcome(t, invalidEntry.DoesNotExist(), &schema),
		referenceExpectationOutcome(t, invalidEntry.DoesNotExist(), reference),
		inspectionExpectationOutcome(t, invalidEntry.DoesNotExist(), inspection),
	} {
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "invalid") {
			t.Fatalf("invalid relational entry absence = %#v, want disproved invalid locator", got)
		}
	}
}

func TestValueDomainNegativeExistenceRejectsAmbiguousDomainTopology(t *testing.T) {
	t.Parallel()

	const valueID cli.ValueID = 4
	use := cli.ValueDomainUse{
		Destination: valueID,
		Kind:        cli.ValueDomainUseDestination,
	}
	domain := ValueByID(valueID).Domain(cli.ValueDomainUseDestination)
	entry := domain.Entry("legacy")

	schema := cli.Schema{
		ValueDomains: []cli.ValueDomainSchema{{Uses: []cli.ValueDomainUse{use}}, {Uses: []cli.ValueDomainUse{use}}},
	}
	reference := cli.Reference{
		ValueDomains: []cli.ValueDomainReference{{Uses: []cli.ValueDomainUse{use}}, {Uses: []cli.ValueDomainUse{use}}},
	}
	inspection := cli.Inspection{
		ValueDomains: []cli.ValueDomainInspection{{Uses: []cli.ValueDomainUse{use}}, {Uses: []cli.ValueDomainUse{use}}},
	}

	for _, expectation := range []DeclarationExpectation{domain.DoesNotExist(), entry.DoesNotExist()} {
		if got := schemaExpectationOutcome(
			t,
			expectation,
			&schema,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "matches value-domain indexes; count=2") {
			t.Fatalf("schema multi-domain %T = %#v, want disproved ambiguity", expectation, got)
		}
		if got := referenceExpectationOutcome(
			t,
			expectation,
			reference,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "matches value-domain indexes; count=2") {
			t.Fatalf("reference multi-domain %T = %#v, want disproved ambiguity", expectation, got)
		}
		if got := inspectionExpectationOutcome(
			t,
			expectation,
			inspection,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "matches value-domain indexes; count=2") {
			t.Fatalf("inspection multi-domain %T = %#v, want disproved ambiguity", expectation, got)
		}
	}
}

func TestValueDomainNegativeExistenceDistinguishesAbsenceFromIncompleteObservation(t *testing.T) {
	t.Parallel()

	const valueID cli.ValueID = 4
	domain := ValueByID(valueID).Domain(cli.ValueDomainUseDestination)
	entry := domain.Entry("missing")

	for _, expectation := range []DeclarationExpectation{domain.DoesNotExist(), entry.DoesNotExist()} {
		if got := schemaExpectationOutcome(t, expectation, &cli.Schema{}); got.status != evaluationProved {
			t.Fatalf("schema complete absence %T = %#v, want proved", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, cli.Reference{}); got.status != evaluationProved {
			t.Fatalf("reference complete absence %T = %#v, want proved", expectation, got)
		}
		if got := inspectionExpectationOutcome(t, expectation, cli.Inspection{}); got.status != evaluationProved {
			t.Fatalf("inspection complete absence %T = %#v, want proved", expectation, got)
		}
		if got := inspectionExpectationOutcome(t, expectation, cli.Inspection{
			Truncated: true,
		}); got.status != evaluationNotProvable {
			t.Fatalf("inspection truncated absence %T = %#v, want not provable", expectation, got)
		}
	}
}

func TestNoticeSubjectFilteringPreservesRelationalResolutionStatus(t *testing.T) {
	t.Parallel()

	const (
		rootID      cli.CommandID          = 1
		referenceID cli.ValueDomainEntryID = 8
		modernID    cli.ValueDomainEntryID = 9
	)
	replacement := &cli.DeclarationReplacement{
		Target: cli.DeclarationIdentity{
			Kind:             cli.DeclarationKindValueDomainEntry,
			ValueDomainEntry: modernID,
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			rootID: {
				ID:                    rootID,
				Path:                  []string{},
				Flags:                 []cli.FlagID{2, 3},
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			2: {
				ID:          2,
				Command:     rootID,
				Destination: 4,
				Spellings:   []cli.FlagSpellingID{6},
			},
			3: {
				ID:          3,
				Command:     rootID,
				Destination: 5,
				Spellings:   []cli.FlagSpellingID{7},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			6: {
				ID:   6,
				Flag: 2,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
			7: {
				ID:   7,
				Flag: 3,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
		Notices: []cli.NoticeInspection{{
			Code: cli.NoticeDeprecatedValue,
			Declaration: cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindValueDomainEntry,
				ValueDomainEntry: referenceID,
			},
			Replacement: replacement,
		}},
	}
	ambiguousEntry := RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination).Entry("legacy")
	ambiguousNotice := Notice(cli.NoticeDeprecatedValue).On(ambiguousEntry)
	modern := ValueDomainEntryByID(modernID)

	for _, expectation := range []InspectionExpectation{
		ambiguousNotice.Absent(),
		ambiguousNotice.Present(),
		ambiguousNotice.ReplacedBy(modern),
		NoticesExactly(ambiguousNotice),
	} {
		got := inspectionExpectationOutcome(t, expectation, inspection)
		if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "matches") {
			t.Fatalf("ambiguous notice subject %T = %#v, want disproved ambiguity", expectation, got)
		}
	}

	noCandidate := inspection
	noCandidate.Notices = nil
	if got := inspectionExpectationOutcome(
		t,
		ambiguousNotice.Absent(),
		noCandidate,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "matches") {
		t.Fatalf("ambiguous notice subject with no code candidate = %#v, want disproved ambiguity", got)
	}

	invalidEntry := (ValueLocator{}).Domain(cli.ValueDomainUseDestination).Entry("legacy")
	invalidNotice := Notice(cli.NoticeDeprecatedValue).On(invalidEntry)
	for _, observed := range []cli.Inspection{inspection, noCandidate} {
		if got := inspectionExpectationOutcome(
			t,
			invalidNotice.Absent(),
			observed,
		); got.status != evaluationDisproved ||
			!strings.Contains(renderEvaluationField(got.observed), "invalid") {
			t.Fatalf("invalid notice subject absence = %#v, want disproved invalid locator", got)
		}
	}

	complete := cli.Inspection{
		ValueDomains: []cli.ValueDomainInspection{{
			Uses: []cli.ValueDomainUse{{
				Destination: 4,
				Kind:        cli.ValueDomainUseDestination,
			}},
			Entries: []cli.ValueDomainEntryID{referenceID},
			Outcome: cli.ValueDomainOutcomeReady,
		}},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{
			referenceID: {
				ID:   referenceID,
				Name: "legacy",
			},
		},
		Notices: inspection.Notices,
	}
	missingEntry := ValueByID(4).Domain(cli.ValueDomainUseDestination).Entry("missing")
	if got := inspectionExpectationOutcome(
		t,
		Notice(cli.NoticeDeprecatedValue).On(missingEntry).Absent(),
		complete,
	); got.status != evaluationProved {
		t.Fatalf("complete missing notice subject = %#v, want proved absence", got)
	}

	truncated := cli.Inspection{
		Truncated: true,
		Notices:   inspection.Notices,
	}
	truncatedEntry := ValueByID(4).Domain(cli.ValueDomainUseDestination).Entry("missing")
	if got := inspectionExpectationOutcome(
		t,
		Notice(cli.NoticeDeprecatedValue).On(truncatedEntry).Absent(),
		truncated,
	); got.status != evaluationNotProvable {
		t.Fatalf("truncated notice subject absence = %#v, want not provable", got)
	}
}
