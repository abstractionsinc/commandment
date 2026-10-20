package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestValueInspectionExpectationGraph(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(7)
	blockedID := cli.ValueID(8)
	candidateID := cli.ValueCandidateID(11)
	originID := cli.InputOriginID{}
	domainEntryID := cli.ValueDomainEntryID(17)
	declaration := cli.DeclarationIdentity{
		Kind: cli.DeclarationKindFlag,
		Flag: cli.FlagID(19),
	}
	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{
			valueID: {
				ID:    valueID,
				State: cli.ResolutionStateResolved,
				Producers: []cli.ValueProducerInspection{
					{
						Kind:      cli.ValueProducerKindFlag,
						Tier:      cli.ResolutionTierExplicit,
						Outcome:   cli.ValueProducerOutcomeContributed,
						Origins:   []cli.InputOriginID{originID},
						BlockedBy: []cli.ValueID{blockedID},
					},
				},
				Candidates: []cli.ValueCandidateID{
					candidateID,
				},
				Contributors: []cli.ValueCandidateID{candidateID},
				BlockedBy:    []cli.ValueID{blockedID},
				Elements: []cli.ValueElementReference{{
					Candidate: candidateID,
					Index:     0,
				}},
			},
		},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			candidateID: {
				ID:            candidateID,
				Destination:   valueID,
				ProducerIndex: 0,
				Disposition:   cli.ValueCandidateDispositionEffective,
				Contribution:  cli.ValueCandidateContributionSequenceSegment,
				Declaration:   declaration,
				Origins:       []cli.InputOriginID{originID},
				Elements: []cli.ValueComponentInspection{{
					Representation: &cli.ValueRepresentation{
						Kind: cli.ValueRepresentationText,
						Text: "x",
					},
					DomainEntry: domainEntryID,
					Origins:     []cli.InputOriginID{originID},
				}},
			},
		},
		ValueDomains: []cli.ValueDomainInspection{
			{
				Uses: []cli.ValueDomainUse{{
					Destination: valueID,
					Kind:        cli.ValueDomainUseSequenceElement,
				}},
				Dependencies:     []cli.ValueID{blockedID},
				Outcome:          cli.ValueDomainOutcomeReady,
				InvocationScoped: true,
			},
		},
	}

	value := ValueByID(valueID)
	candidate := value.CandidateAt(0)
	producer := value.ProducerAt(0)
	element := candidate.ElementAt(0)

	expectations := []InspectionExpectation{
		value.Resolved(),
		value.ContributedBy(cli.ValueProducerKindFlag),
		value.BlockedBy(ValueByID(blockedID)),
		value.ProducersExactly(
			producer,
		),
		value.CandidatesExactly(candidate),
		value.ContributorsExactly(candidate),
		value.ElementsExactly(element),
		producer.Kind(
			cli.ValueProducerKindFlag,
		),
		producer.Tier(cli.ResolutionTierExplicit),
		producer.Outcome(cli.ValueProducerOutcomeContributed),
		producer.OriginsExactly(InputOriginByID(originID)),
		producer.BlockedByExactly(ValueByID(blockedID)),
		candidate.Disposition(
			cli.ValueCandidateDispositionEffective,
		),
		candidate.Contribution(cli.ValueCandidateContributionSequenceSegment),
		candidate.ProducedBy(producer),
		candidate.Declaration(FlagByID(cli.FlagID(19))),
		candidate.OriginsExactly(InputOriginByID(originID)),
		candidate.ElementCount(1),
		element.Text(
			"x",
		),
		element.DomainEntry(ValueDomainEntryByID(domainEntryID)),
		element.OriginsExactly(InputOriginByID(originID)),
		value.Domain(cli.ValueDomainUseSequenceElement).
			Outcome(cli.ValueDomainOutcomeReady),
		value.Domain(cli.ValueDomainUseSequenceElement).InvocationScoped(),
		value.Domain(cli.ValueDomainUseSequenceElement).DependenciesExactly(ValueByID(blockedID)),
	}
	for _, expectation := range expectations {
		evaluator, ok := expectation.(interface {
			evaluateInspection(inspection cli.Inspection, err error) evaluationResult
		})
		if !ok {
			t.Fatal("expectation must satisfy expected evaluator interface")
		}
		if outcome := evaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
			t.Fatalf("expectation %T did not pass: %#v", expectation, outcome)
		}
	}
}

func TestValueExactCollectionIsNotProvableWhenInspectionTruncated(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(1)
	inspection := cli.Inspection{
		Truncated: true,
		Values:    map[cli.ValueID]cli.ValueInspection{valueID: {ID: valueID}},
	}
	expectation := ValueByID(valueID).CandidatesExactly()
	checkedOutcome, ok := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("expectation must satisfy expected evaluator interface")
	}
	outcome := checkedOutcome.evaluateInspection(inspection, nil)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("status = %v, want not provable", outcome.status)
	}
}

func TestSensitiveValueMismatchRedactsExpectedAndObservedMaterial(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(1)
	candidateID := cli.ValueCandidateID(2)
	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{
			valueID: {
				ID:         valueID,
				Candidates: []cli.ValueCandidateID{candidateID},
			},
		},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			candidateID: {
				ID:          candidateID,
				Destination: valueID,
				Elements: []cli.ValueComponentInspection{{
					Sensitive: true,
					Representation: &cli.ValueRepresentation{
						Kind: cli.ValueRepresentationText,
						Text: "observed-secret",
					},
				}},
			},
		},
	}
	expectation := ValueByID(valueID).CandidateAt(0).ElementAt(0).Text("expected-secret")
	checkedGot, ok := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("expectation must satisfy expected evaluator interface")
	}
	got := checkedGot.evaluateInspection(inspection, nil)
	if got.status != evaluationDisproved {
		t.Fatalf("sensitive mismatch = %#v, want disproved", got)
	}
	for _, secret := range []string{"observed-secret", "expected-secret"} {
		if strings.Contains(renderEvaluationField(got.subject)+renderEvaluationField(got.expected)+renderEvaluationField(got.observed), secret) {
			t.Fatalf("sensitive mismatch leaked %q: %#v", secret, got)
		}
	}
}

func TestCommandAndValueDeclarationExpectationsUseTheirAuthoritativeObservations(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	valueID := cli.ValueID(2)
	commandSchema := cli.CommandSchema{
		ID:             commandID,
		Path:           []string{"serve"},
		Hidden:         true,
		Runnable:       true,
		LifecycleState: cli.LifecycleStateCurrent,
	}
	valueSchema := cli.ValueSchema{
		ID:        valueID,
		Required:  true,
		Sensitive: true,
	}
	schema := &cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{commandID: commandSchema},
		Values:   map[cli.ValueID]cli.ValueSchema{valueID: valueSchema},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{
			commandID: {
				ID:             commandID,
				Path:           []string{"serve"},
				Hidden:         true,
				Runnable:       true,
				LifecycleState: cli.LifecycleStateCurrent,
			},
		},
		Values: map[cli.ValueID]cli.ValueReference{
			valueID: {
				ValueSchema: valueSchema,
				Available:   true,
				Visible:     false,
			},
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:                    commandID,
				Path:                  []string{"serve"},
				Hidden:                true,
				Runnable:              true,
				LifecycleState:        cli.LifecycleStateCurrent,
				Available:             true,
				Visible:               false,
				NamespaceCompleteness: cli.CommandNamespaceComplete,
			},
		},
		Values: map[cli.ValueID]cli.ValueInspection{valueID: {
			ID:        valueID,
			Sensitive: true,
		}},
	}

	command := Command("serve")
	for _, expectation := range []DeclarationExpectation{
		command.Exists(),
		command.Hidden(),
		command.Runnable(),
		command.Lifecycle(cli.LifecycleStateCurrent),
	} {
		schemaEvaluator, ok := expectation.(interface {
			evaluateSchema(schema *cli.Schema) evaluationResult
		})
		if !ok {
			t.Fatal("expectation must satisfy expected evaluator interface")
		}
		if outcome := schemaEvaluator.evaluateSchema(schema); outcome.status != evaluationProved {
			t.Fatalf("schema %T: %#v", expectation, outcome)
		}
		referenceEvaluator, ok := expectation.(interface {
			evaluateReference(reference cli.Reference, err error) evaluationResult
		})
		if !ok {
			t.Fatal("expectation must satisfy expected evaluator interface")
		}
		if outcome := referenceEvaluator.evaluateReference(reference, nil); outcome.status != evaluationProved {
			t.Fatalf("reference %T: %#v", expectation, outcome)
		}
		inspectionEvaluator, ok := expectation.(interface {
			evaluateInspection(inspection cli.Inspection, err error) evaluationResult
		})
		if !ok {
			t.Fatal("expectation must satisfy expected evaluator interface")
		}
		if outcome := inspectionEvaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
			t.Fatalf("inspection %T: %#v", expectation, outcome)
		}
	}

	value := ValueByID(valueID)
	for _, expectation := range []StaticDeclarationExpectation{
		value.Exists(),
		value.Required(),
		value.DeclaredSensitive(),
	} {
		schemaEvaluator, ok := expectation.(interface {
			evaluateSchema(schema *cli.Schema) evaluationResult
		})
		if !ok {
			t.Fatal("expectation must satisfy expected evaluator interface")
		}
		if outcome := schemaEvaluator.evaluateSchema(schema); outcome.status != evaluationProved {
			t.Fatalf("static schema %T: %#v", expectation, outcome)
		}
		referenceEvaluator, ok := expectation.(interface {
			evaluateReference(reference cli.Reference, err error) evaluationResult
		})
		if !ok {
			t.Fatal("expectation must satisfy expected evaluator interface")
		}
		if outcome := referenceEvaluator.evaluateReference(reference, nil); outcome.status != evaluationProved {
			t.Fatalf("static reference %T: %#v", expectation, outcome)
		}
	}
	availableEvaluator, ok := value.Available().(interface {
		evaluateReference(reference cli.Reference, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Available must satisfy expected evaluator interface")
	}
	if outcome := availableEvaluator.evaluateReference(reference, nil); outcome.status != evaluationProved {
		t.Fatalf("available reference: %#v", outcome)
	}
	observedEvaluator, ok := value.Observed().(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Observed must satisfy expected evaluator interface")
	}
	if outcome := observedEvaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
		t.Fatalf("observed inspection: %#v", outcome)
	}
	sensitiveEvaluator, ok := value.Sensitive().(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Sensitive must satisfy expected evaluator interface")
	}
	if outcome := sensitiveEvaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
		t.Fatalf("effective sensitive inspection: %#v", outcome)
	}
	namespaceEvaluator, ok := command.NamespaceCompleteness(cli.CommandNamespaceComplete).(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Namespace must satisfy expected evaluator interface")
	}
	if outcome := namespaceEvaluator.evaluateInspection(inspection, nil); outcome.status != evaluationProved {
		t.Fatalf("namespace: %#v", outcome)
	}
}

func TestValueObservationStateAndConditionalStateRemainDistinct(t *testing.T) {
	t.Parallel()
	value := ValueByID(7)

	observedAbsent := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{
			7: {
				ID:    7,
				State: cli.ResolutionStateAbsent,
			},
		},
	}
	for name, expectation := range map[string]InspectionExpectation{
		"observed":     value.Observed(),
		"absent-state": value.State(cli.ResolutionStateAbsent),
	} {
		if got := inspectionExpectationOutcome(t, expectation, observedAbsent); got.status != evaluationProved {
			t.Fatalf("%s = %#v, want proved", name, got)
		}
	}
	if got := inspectionExpectationOutcome(
		t,
		value.NotObserved(),
		observedAbsent,
	); got.status != evaluationDisproved {
		t.Fatalf("NotObserved observed-absent = %#v, want disproved", got)
	}

	missing := cli.Inspection{}
	if got := inspectionExpectationOutcome(t, value.NotObserved(), missing); got.status != evaluationProved {
		t.Fatalf("NotObserved complete miss = %#v, want proved", got)
	}
	if got := inspectionExpectationOutcome(t, value.Observed(), missing); got.status != evaluationDisproved {
		t.Fatalf("Observed complete miss = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.State(cli.ResolutionStateAbsent),
		missing,
	); got.status != evaluationDisproved {
		t.Fatalf("State on unobserved value = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.StateIfObserved(cli.ResolutionStateAbsent),
		missing,
	); got.status != evaluationProved {
		t.Fatalf("StateIfObserved complete miss = %#v, want proved", got)
	}

	truncated := cli.Inspection{
		Truncated: true,
	}
	for name, expectation := range map[string]InspectionExpectation{
		"observed":          value.Observed(),
		"not-observed":      value.NotObserved(),
		"state":             value.State(cli.ResolutionStateAbsent),
		"state-if-observed": value.StateIfObserved(cli.ResolutionStateAbsent),
	} {
		if got := inspectionExpectationOutcome(t, expectation, truncated); got.status != evaluationNotProvable {
			t.Fatalf("%s truncated miss = %#v, want not provable", name, got)
		}
	}
}

func TestValueInactiveAndStateIfObservedBranches(t *testing.T) {
	t.Parallel()
	value := ValueByID(1)
	inactive := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{
			1: {
				ID:    1,
				State: cli.ResolutionStateInactive,
			},
		},
	}
	if got := inspectionExpectationOutcome(t, value.Inactive(), inactive); got.status != evaluationProved {
		t.Fatalf("Inactive = %#v, want proved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.State(cli.ResolutionStateInactive),
		inactive,
	); got.status != evaluationProved {
		t.Fatalf("State(Inactive) = %#v, want proved", got)
	}
	resolved := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{
			1: {
				ID:    1,
				State: cli.ResolutionStateResolved,
			},
		},
	}
	if got := inspectionExpectationOutcome(t, value.Inactive(), resolved); got.status != evaluationDisproved {
		t.Fatalf("Inactive mismatch = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.State(cli.ResolutionStateInactive),
		resolved,
	); got.status != evaluationDisproved {
		t.Fatalf("State(Inactive) mismatch = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.StateIfObserved(cli.ResolutionStateInactive, cli.ResolutionStateAbsent),
		inactive,
	); got.status != evaluationProved {
		t.Fatalf("StateIfObserved allowed = %#v, want proved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.StateIfObserved(cli.ResolutionStateFailed),
		inactive,
	); got.status != evaluationDisproved {
		t.Fatalf("StateIfObserved disallowed = %#v, want disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.StateIfObserved(),
		inactive,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "no allowed states") {
		t.Fatalf("StateIfObserved empty = %#v, want invalid/disproved", got)
	}

	states := []cli.ResolutionState{cli.ResolutionStateInactive}
	expectation := value.StateIfObserved(states...)
	states[0] = cli.ResolutionStateFailed
	if got := inspectionExpectationOutcome(t, expectation, inactive); got.status != evaluationProved {
		t.Fatalf("StateIfObserved retained caller slice = %#v, want proved", got)
	}
}

func TestValueObservationRejectsInvalidAndAmbiguousLocators(t *testing.T) {
	t.Parallel()
	if got := inspectionExpectationOutcome(
		t,
		ValueLocator{}.NotObserved(),
		cli.Inspection{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "invalid") {
		t.Fatalf("invalid locator = %#v, want invalid/disproved", got)
	}

	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:    1,
				Path:  nil,
				Flags: []cli.FlagID{10, 11},
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{
			10: {
				ID:          10,
				Command:     1,
				Destination: 20,
			},
			11: {
				ID:          11,
				Command:     1,
				Destination: 21,
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingInspection{
			100: {
				ID:   100,
				Flag: 10,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
			101: {
				ID:   101,
				Flag: 11,
				Spelling: cli.FlagSpelling{
					Text: "--mode",
				},
			},
		},
	}
	inspection.Flags[10] = cli.FlagInspection{
		ID:          10,
		Command:     1,
		Destination: 20,
		Spellings:   []cli.FlagSpellingID{100},
	}
	inspection.Flags[11] = cli.FlagInspection{
		ID:          11,
		Command:     1,
		Destination: 21,
		Spellings:   []cli.FlagSpellingID{101},
	}
	ambiguous := RootCommand().Flag("--mode").Value().Observed()
	got := inspectionExpectationOutcome(t, ambiguous, inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "matches") {
		t.Fatalf("ambiguous locator = %#v, want ambiguity/disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		RootCommand().Flag("--mode").Value().StateIfObserved(cli.ResolutionStateResolved),
		inspection,
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "matches") {
		t.Fatalf("StateIfObserved ambiguous locator = %#v, want ambiguity/disproved", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		ValueLocator{}.StateIfObserved(cli.ResolutionStateResolved),
		cli.Inspection{},
	); got.status != evaluationDisproved ||
		!strings.Contains(renderEvaluationField(got.observed), "invalid") {
		t.Fatalf("StateIfObserved invalid locator = %#v, want invalid/disproved", got)
	}
}

func TestValueRepresentationsEntriesAndSuppressionExpectations(t *testing.T) {
	t.Parallel()
	textValueID := cli.ValueID(31)
	tokenValueID := cli.ValueID(32)
	keyedValueID := cli.ValueID(33)
	textCandidateID := cli.ValueCandidateID(41)
	tokenCandidateID := cli.ValueCandidateID(42)
	domainEntryID := cli.ValueDomainEntryID(51)
	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{
			textValueID: {
				ID: textValueID,
				Representation: &cli.ValueRepresentation{
					Kind: cli.ValueRepresentationText,
					Text: "alpha",
				},
			},
			tokenValueID: {
				ID: tokenValueID,
				Representation: &cli.ValueRepresentation{
					Kind:   cli.ValueRepresentationTokens,
					Tokens: []string{"alpha", "beta"},
				},
			},
			keyedValueID: {
				ID:         keyedValueID,
				Candidates: []cli.ValueCandidateID{textCandidateID, tokenCandidateID},
				Entries: []cli.ValueEntryReference{{
					Candidate: textCandidateID,
					Index:     0,
				}},
			},
		},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			textCandidateID: {
				ID:          textCandidateID,
				Destination: keyedValueID,
				Representation: &cli.ValueRepresentation{
					Kind: cli.ValueRepresentationText,
					Text: "candidate",
				},
				DomainEntry:  domainEntryID,
				SuppressedBy: []cli.ValueCandidateID{tokenCandidateID},
				Entries: []cli.ValueEntryInspection{{
					Key: cli.ValueComponentInspection{
						Representation: &cli.ValueRepresentation{
							Kind:   cli.ValueRepresentationTokens,
							Tokens: []string{"key"},
						},
					},
					Value: cli.ValueComponentInspection{
						Representation: &cli.ValueRepresentation{
							Kind: cli.ValueRepresentationText,
							Text: "value",
						},
					},
				}},
			},
			tokenCandidateID: {
				ID:          tokenCandidateID,
				Destination: keyedValueID,
				Representation: &cli.ValueRepresentation{
					Kind:   cli.ValueRepresentationTokens,
					Tokens: []string{"candidate", "tokens"},
				},
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{
			domainEntryID: {ID: domainEntryID},
		},
	}

	keyed := ValueByID(keyedValueID)
	textCandidate := keyed.CandidateAt(0)
	tokenCandidate := keyed.CandidateAt(1)
	entry := textCandidate.EntryAt(0)
	(InspectionResult{state: &inspectionResultState{observation: inspection, err: nil}}).Require(t,
		ValueByID(textValueID).Text("alpha"),
		ValueByID(tokenValueID).Tokens("alpha", "beta"),
		keyed.EntriesExactly(entry),
		textCandidate.Text("candidate"),
		tokenCandidate.Tokens("candidate", "tokens"),
		textCandidate.DomainEntry(ValueDomainEntryByID(domainEntryID)),
		textCandidate.SuppressedByExactly(tokenCandidate),
		entry.Key().Tokens("key"),
		entry.Value().Text("value"))
}

func TestValueZeroHitRuntimeRedactionIdentityAndNegativeLaws(t *testing.T) {
	t.Parallel()
	valueID := cli.ValueID(70)
	blockedID := cli.ValueID(71)
	candidateID := cli.ValueCandidateID(80)
	conflictID := cli.ValueCandidateID(81)
	entryID := cli.ValueDomainEntryID(90)
	originID := cli.InputOriginID{}
	declaration := cli.DeclarationIdentity{
		Kind: cli.DeclarationKindFlag,
		Flag: cli.FlagID(100),
	}

	inspection := cli.Inspection{
		Facts: map[cli.ValueID]cli.FactInspection{valueID: {ID: valueID}},
		Values: map[cli.ValueID]cli.ValueInspection{
			valueID: {
				ID:        valueID,
				State:     cli.ResolutionStateBlocked,
				Sensitive: true,
				Representation: &cli.ValueRepresentation{
					Kind:     cli.ValueRepresentationText,
					Redacted: true,
				},
				Producers: []cli.ValueProducerInspection{
					{
						Kind:         cli.ValueProducerKindFlag,
						Tier:         cli.ResolutionTierExplicit,
						Outcome:      cli.ValueProducerOutcomeContributed,
						Sensitive:    true,
						Origins:      []cli.InputOriginID{originID},
						Dependencies: []cli.ValueID{blockedID},
						BlockedBy:    []cli.ValueID{blockedID},
					},
				},
				Candidates: []cli.ValueCandidateID{
					candidateID,
				},
				Contributors: []cli.ValueCandidateID{candidateID},
				BlockedBy:    []cli.ValueID{blockedID},
				Entries: []cli.ValueEntryReference{{
					Candidate: candidateID,
					Index:     0,
				}},
			},
		},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			candidateID: {
				ID:            candidateID,
				Destination:   valueID,
				ProducerIndex: 0,
				Disposition:   cli.ValueCandidateDispositionEffective,
				Contribution:  cli.ValueCandidateContributionKeyedEntry,
				Declaration:   declaration,
				Origins:       []cli.InputOriginID{originID},
				Sensitive:     true,
				Suppression:   cli.ValueCandidateSuppressionKeepLast,
				ConflictsWith: []cli.ValueCandidateID{conflictID},
				Entries: []cli.ValueEntryInspection{{
					Key: cli.ValueComponentInspection{
						Representation: &cli.ValueRepresentation{
							Kind:     cli.ValueRepresentationText,
							Redacted: true,
						},
						DomainEntry: entryID,
						Origins:     []cli.InputOriginID{originID},
						Sensitive:   true,
					},
					Value: cli.ValueComponentInspection{
						Representation: &cli.ValueRepresentation{
							Kind:   cli.ValueRepresentationTokens,
							Tokens: []string{"v"},
						},
						Origins:   []cli.InputOriginID{originID},
						Sensitive: true,
					},
				}},
			},
			conflictID: {
				ID:            conflictID,
				Destination:   valueID,
				ProducerIndex: 0,
			},
		},
		ValueDomains: []cli.ValueDomainInspection{{
			Uses: []cli.ValueDomainUse{{
				Destination: valueID,
				Kind:        cli.ValueDomainUseDestination,
			}},
			Dependencies: []cli.ValueID{blockedID},
			Outcome:      cli.ValueDomainOutcomeReady,
		}},
	}

	value := FactByID(valueID).Value()
	candidate := ValueCandidateByID(candidateID)
	producer := value.ProducerAt(0)
	contributor := value.ContributorAt(0)
	entry := candidate.EntryAt(0)
	key := entry.Key()
	entryValue := entry.Value()

	for name, expectation := range map[string]InspectionExpectation{
		"fact value redacted":        value.Redacted(),
		"value failed state":         value.Blocked(),
		"producer sensitive":         producer.Sensitive(),
		"producer dependencies":      producer.DependenciesExactly(ValueByID(blockedID)),
		"candidate direct sensitive": candidate.Sensitive(),
		"contributor sensitive":      contributor.Sensitive(),
		"candidate suppression":      candidate.Suppression(cli.ValueCandidateSuppressionKeepLast),
		"candidate conflicts":        candidate.ConflictsWithExactly(ValueCandidateByID(conflictID)),
		"candidate entry count":      candidate.EntryCount(1),
		"key redacted":               key.Redacted(),
		"key sensitive":              key.Sensitive(),
		"key domain entry":           key.DomainEntry(ValueDomainEntryByID(entryID)),
		"entry value sensitive":      entryValue.Sensitive(),
		"entry value tokens":         entryValue.Tokens("v"),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("%s = %#v, want proved", name, got)
		}
	}

	for name, expectation := range map[string]InspectionExpectation{
		"failed-state mismatch":             value.Failed(),
		"sensitive value text mismatch":     value.Text("secret"),
		"sensitive producer mismatch":       producer.Kind(cli.ValueProducerKindEnvironment),
		"sensitive candidate mismatch":      candidate.Suppression(cli.ValueCandidateSuppressionPrecedence),
		"sensitive component mismatch":      key.Text("secret-key"),
		"producer missing":                  value.ProducerAt(3).Sensitive(),
		"contributor missing":               value.ContributorAt(3).Sensitive(),
		"entry missing":                     candidate.EntryAt(3).Key().Sensitive(),
		"wrong producer kind":               value.ContributedBy(cli.ValueProducerKindEnvironment),
		"wrong blocker":                     value.BlockedBy(ValueByID(999)),
		"candidate producer cannot resolve": candidate.ProducedBy(value.ProducerAt(9)),
		"candidate entry count mismatch":    candidate.EntryCount(2),
		"value domain dependency mismatch": value.Domain(cli.ValueDomainUseDestination).DependenciesExactly(
			ValueByID(999),
		),
		"candidate conflict identity mismatch": candidate.ConflictsWithExactly(ValueCandidateByID(999)),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationDisproved {
			t.Fatalf("%s = %#v, want disproved", name, got)
		}
	}

	for _, expectation := range []InspectionExpectation{
		value.ProducersExactly(value.ProducerAt(0), value.ProducerAt(1)),
		value.CandidatesExactly(ValueCandidateByID(conflictID)),
		value.ContributorsExactly(ValueCandidateByID(conflictID)),
		value.EntriesExactly(candidate.EntryAt(1)),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationDisproved {
			t.Fatalf("exact identity mismatch %T = %#v, want disproved", expectation, got)
		}
	}

	ambiguous := inspection
	ambiguous.ValueDomains = append(
		append([]cli.ValueDomainInspection(nil), inspection.ValueDomains...),
		inspection.ValueDomains[0],
	)
	if got := inspectionExpectationOutcome(
		t,
		value.Domain(cli.ValueDomainUseDestination).Outcome(cli.ValueDomainOutcomeReady),
		ambiguous,
	); got.status != evaluationDisproved {
		t.Fatalf("ambiguous domain = %#v, want disproved", got)
	}

	truncated := inspection
	truncated.Truncated = true
	if got := inspectionExpectationOutcome(
		t,
		value.ProducerAt(3).Sensitive(),
		truncated,
	); got.status != evaluationNotProvable {
		t.Fatalf("truncated producer = %#v, want not provable", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		value.ContributorAt(3).Sensitive(),
		truncated,
	); got.status != evaluationNotProvable {
		t.Fatalf("truncated contributor = %#v, want not provable", got)
	}
	if got := inspectionExpectationOutcome(
		t,
		candidate.EntryAt(3).Key().Sensitive(),
		truncated,
	); got.status != evaluationNotProvable {
		t.Fatalf("truncated entry = %#v, want not provable", got)
	}

	var zeroValue ValueLocator
	if got := inspectionExpectationOutcome(
		t,
		zeroValue.Blocked(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("zero value locator = %#v, want deterministic disproval", got)
	}
	var zeroCandidate ValueCandidateLocator
	if got := inspectionExpectationOutcome(
		t,
		zeroCandidate.Sensitive(),
		inspection,
	); got.status != evaluationDisproved {
		t.Fatalf("zero candidate locator = %#v, want deterministic disproval", got)
	}
}

func TestFactAndValueZeroHitStaticNegativePublicSurface(t *testing.T) {
	t.Parallel()
	factID := cli.ValueID(7)
	valueID := cli.ValueID(8)
	schema := cli.Schema{
		Facts:  map[cli.ValueID]cli.FactSchema{factID: {ID: factID}},
		Values: map[cli.ValueID]cli.ValueSchema{valueID: {ID: valueID}},
	}
	reference := cli.Reference{
		Facts: map[cli.ValueID]cli.FactReference{factID: {
			FactSchema: cli.FactSchema{
				ID: factID,
			},
			Available: true,
		}},
		Values: map[cli.ValueID]cli.ValueReference{valueID: {
			ValueSchema: cli.ValueSchema{
				ID: valueID,
			},
			Available: true,
			Visible:   false,
		}},
	}

	for name, expectation := range map[string]SchemaExpectation{
		"fact exists":       FactByID(factID).Exists(),
		"fact missing":      FactByID(99).DoesNotExist(),
		"value domain miss": ValueByID(valueID).Domain(cli.ValueDomainUseDestination).DoesNotExist(),
		"domain entry miss": ValueDomainEntryByID(99).DoesNotExist(),
	} {
		if got := schemaExpectationOutcome(t, expectation, &schema); got.status != evaluationProved {
			t.Fatalf("schema %s = %#v, want proved", name, got)
		}
	}
	if got := referenceExpectationOutcome(
		t,
		FactByID(99).DoesNotExist(),
		reference,
	); got.status != evaluationProved {
		t.Fatalf("missing fact reference = %#v, want proved", got)
	}
	if got := referenceExpectationOutcome(
		t,
		FactByID(factID).Unavailable(),
		reference,
	); got.status != evaluationDisproved {
		t.Fatalf("fact unavailable mismatch = %#v, want disproved", got)
	}
	if got := referenceExpectationOutcome(
		t,
		ValueByID(valueID).Unavailable(),
		reference,
	); got.status != evaluationDisproved {
		t.Fatalf("value unavailable mismatch = %#v, want disproved", got)
	}
	if got := referenceExpectationOutcome(
		t,
		ValueByID(valueID).Visible(),
		reference,
	); got.status != evaluationDisproved {
		t.Fatalf("value visible mismatch = %#v, want disproved", got)
	}
	if got := referenceExpectationOutcome(
		t,
		ValueByID(valueID).NotVisible(),
		reference,
	); got.status != evaluationProved {
		t.Fatalf("value not-visible = %#v, want proved", got)
	}
}

func TestProducerResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("value producer: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveProducer(nil, cli.Inspection{})
		if got.status != locatorResolutionInvalid || renderEvaluationField(got.actual) != "invalid value-producer locator" {
			t.Fatalf("resolution = %#v", got)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		got := resolveProducer(ValueByID(9).ProducerAt(0).state, cli.Inspection{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("status = %v, want absent", got.status)
		}
	})
}

func TestCandidateResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("value candidate index: matching observation", func(t *testing.T) {
		t.Parallel()
		got := resolveCandidate(ValueByID(9).CandidateAt(0).state, cli.Inspection{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("status = %v, want absent", got.status)
		}
	})
	t.Run("value candidate index: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated: true,
			Values: map[cli.ValueID]cli.ValueInspection{1: {
				ID:         1,
				Candidates: []cli.ValueCandidateID{2},
			}},
		}
		got := resolveCandidate(ValueByID(1).CandidateAt(0).state, inspection)
		if got.status != locatorResolutionNotProvable || got.id != 2 {
			t.Fatalf("resolution = %#v", got)
		}
	})
}

func TestEntryResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("value entry: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		_, status, actual := resolveEntry(nil, cli.Inspection{})
		if status != locatorResolutionInvalid || renderEvaluationField(actual) != "invalid value-entry locator" {
			t.Fatalf("status/detail = %v, %q", status, actual)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		_, status, _ := resolveEntry(ValueCandidateByID(9).EntryAt(0).state, cli.Inspection{})
		if status != locatorResolutionAbsent {
			t.Fatalf("status = %v, want absent", status)
		}
	})
}

func TestComponentResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("value component: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveComponent(nil, cli.Inspection{})
		if got.status != locatorResolutionInvalid {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("value element index: matching observation", func(t *testing.T) {
		t.Parallel()
		got := resolveComponent(ValueByID(9).CandidateAt(0).ElementAt(0).state, cli.Inspection{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("status = %v", got.status)
		}
	})
	t.Run("value element index: incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated: true,
			Values: map[cli.ValueID]cli.ValueInspection{1: {
				ID:         1,
				Candidates: []cli.ValueCandidateID{2},
			}},
			ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{2: {
				ID:          2,
				Destination: 1,
			}},
		}
		got := resolveComponent(ValueByID(1).CandidateAt(0).ElementAt(0).state, inspection)
		if got.status != locatorResolutionNotProvable || renderEvaluationField(got.actual) != "inspection is truncated" {
			t.Fatalf("resolution = %#v", got)
		}
	})
	t.Run("value element index: semantic absence", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Values: map[cli.ValueID]cli.ValueInspection{1: {
				ID:         1,
				Candidates: []cli.ValueCandidateID{2},
			}},
			ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{2: {
				ID:          2,
				Destination: 1,
			}},
		}
		got := resolveComponent(ValueByID(1).CandidateAt(0).ElementAt(0).state, inspection)
		if got.status != locatorResolutionAbsent || renderEvaluationField(got.actual) != "element is absent" {
			t.Fatalf("resolution = %#v", got)
		}
	})
}

func TestDomainResolutionPreservesResolutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("value domain: invalid coordinates", func(t *testing.T) {
		t.Parallel()
		got := resolveInspectionValueDomainLocator(nil, cli.Inspection{})
		if got.status != locatorResolutionInvalid {
			t.Fatalf("resolution = %#v", got)
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		got := resolveInspectionValueDomainLocator(ValueByID(9).Domain(cli.ValueDomainUseKind(1)).state, cli.Inspection{})
		if got.status != locatorResolutionAbsent {
			t.Fatalf("resolution = %#v", got)
		}
	})
	t.Run("incomplete observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Truncated: true,
			Values:    map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
		}
		got := resolveInspectionValueDomainLocator(ValueByID(1).Domain(cli.ValueDomainUseKind(1)).state, inspection)
		if got.status != locatorResolutionNotProvable {
			t.Fatalf("resolution = %#v", got)
		}
	})
	t.Run("absent value domain: semantic absence", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
		}
		got := resolveInspectionValueDomainLocator(ValueByID(1).Domain(cli.ValueDomainUseKind(1)).state, inspection)
		if got.status != locatorResolutionAbsent {
			t.Fatalf("resolution = %#v", got)
		}
	})
}

func TestValueTextPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).Text("x"), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "redacted") {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueTokensPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).Tokens("x"), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "redacted") {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueContributedByRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:           1,
			Contributors: []cli.ValueCandidateID{2},
		}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).ContributedBy(cli.ValueProducerKind(1)), inspection)
	if got.status != evaluationDisproved {
		t.Fatalf("status = %v", got.status)
	}
}

func TestValueBlockedByRejectsInvalidCoordinates(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, valueBlockedByExpectation(ValueByID(1), ValueLocator{}), inspection)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid blocker locator" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestProducerPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{
			1: {
				ID:        1,
				Producers: []cli.ValueProducerInspection{{Kind: cli.ValueProducerKind(1)}},
			},
		},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).ProducerAt(0).Kind(cli.ValueProducerKind(2)), inspection)
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueProducerValuesExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:        1,
			Producers: []cli.ValueProducerInspection{{}},
		}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).ProducerAt(0).BlockedByExactly(ValueLocator{}), inspection)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid value locator in expected collection" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueProducerOriginsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:        1,
			Producers: []cli.ValueProducerInspection{{}},
		}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).ProducerAt(0).OriginsExactly(InputOriginLocator{}), inspection)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid origin locator in expected collection" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestCandidatePropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			1: {
				ID:          1,
				Disposition: cli.ValueCandidateDisposition(1),
			},
		},
	}
	got := inspectionExpectationOutcome(
		t,
		ValueCandidateByID(1).Disposition(cli.ValueCandidateDisposition(2)),
		inspection,
	)
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueCandidateProducedByRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{2: {
			ID:        2,
			Producers: []cli.ValueProducerInspection{{}},
		}},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{3: {
			ID:          3,
			Destination: 1,
		}},
	}
	got := inspectionExpectationOutcome(t, ValueCandidateByID(3).ProducedBy(ValueByID(2).ProducerAt(0)), inspection)
	if got.status != evaluationDisproved ||
		renderEvaluationField(got.expected) != "producer for value ID 2 at index 0" ||
		renderEvaluationField(got.observed) != "producer for value ID 1 at index 0" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueCandidateDeclarationRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueCandidateByID(1).Declaration(FlagLocator{}), inspection)
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueCandidateOriginsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueCandidateByID(1).OriginsExactly(InputOriginLocator{}), inspection)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid origin locator in expected collection" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueCandidateTextPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueCandidateByID(1).Text("x"), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "redacted") {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueCandidateTokensPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueCandidateByID(1).Tokens("x"), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "redacted") {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestComponentPropertyPreservesPropertyTruth(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:         1,
			Candidates: []cli.ValueCandidateID{2},
		}},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			2: {
				ID:          2,
				Destination: 1,
				Elements:    []cli.ValueComponentInspection{{}},
			},
		},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).CandidateAt(0).ElementAt(0).Sensitive(), inspection)
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueComponentTextPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:         1,
			Candidates: []cli.ValueCandidateID{2},
		}},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			2: {
				ID:          2,
				Destination: 1,
				Elements:    []cli.ValueComponentInspection{{}},
			},
		},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).CandidateAt(0).ElementAt(0).Text("x"), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "redacted") {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueComponentTokensPreservesIncompleteObservation(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:         1,
			Candidates: []cli.ValueCandidateID{2},
		}},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			2: {
				ID:          2,
				Destination: 1,
				Elements:    []cli.ValueComponentInspection{{}},
			},
		},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).CandidateAt(0).ElementAt(0).Tokens("x"), inspection)
	if got.status != evaluationDisproved || !strings.Contains(renderEvaluationField(got.observed), "redacted") {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueComponentOriginsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:         1,
			Candidates: []cli.ValueCandidateID{2},
		}},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			2: {
				ID:          2,
				Destination: 1,
				Elements:    []cli.ValueComponentInspection{{}},
			},
		},
	}
	got := inspectionExpectationOutcome(
		t,
		ValueByID(1).CandidateAt(0).ElementAt(0).OriginsExactly(InputOriginLocator{}),
		inspection,
	)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid origin locator in expected collection" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueCollectionPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	got := inspectionExpectationOutcome(t, ValueByID(9).CandidatesExactly(), cli.Inspection{})
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueProducersExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:        1,
			Producers: []cli.ValueProducerInspection{{}, {}},
		}},
	}
	got := inspectionExpectationOutcome(
		t,
		ValueByID(1).ProducersExactly(ValueByID(1).ProducerAt(1), ValueByID(1).ProducerAt(0)),
		inspection,
	)
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestCandidateLocatorIDRejectsInvalidCoordinate(t *testing.T) {
	t.Parallel()

	if id, ok := candidateLocatorID(nil, cli.Inspection{}); ok || id != 0 {
		t.Fatalf("candidateLocatorID = %v, %t", id, ok)
	}
}

func TestValueCandidatesExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).CandidatesExactly(ValueCandidateLocator{}), inspection)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "expected candidate locator cannot be resolved" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueContributorsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).ContributorsExactly(ValueCandidateLocator{}), inspection)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "expected contributor locator cannot be resolved" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestComponentReferenceFromElementLocatorRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	t.Run("matching observation", func(t *testing.T) {
		t.Parallel()
		if _, ok := componentReferenceFromElementLocator(ValueElementLocator{}, cli.Inspection{}); ok {
			t.Fatal("zero locator unexpectedly resolved")
		}
	})
	t.Run("expected subject unresolved: matching observation", func(t *testing.T) {
		t.Parallel()
		inspection := cli.Inspection{
			Values: map[cli.ValueID]cli.ValueInspection{1: {
				ID:         1,
				Candidates: []cli.ValueCandidateID{2},
			}},
			ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{2: {
				ID:          2,
				Destination: 1,
			}},
		}
		if _, ok := componentReferenceFromElementLocator(ValueByID(1).CandidateAt(0).ElementAt(0), inspection); ok {
			t.Fatal("missing element unexpectedly resolved")
		}
	})
}

func TestEntryReferenceFromLocatorRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	if _, ok := entryReferenceFromLocator(ValueEntryLocator{}, cli.Inspection{}); ok {
		t.Fatal("zero locator unexpectedly resolved")
	}
}

func TestValueElementsExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueByID(1).ElementsExactly(ValueElementLocator{}), inspection)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "expected element locator cannot be resolved" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueDomainDependenciesExactlyPreservesExactMembershipAndOrder(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {ID: 1}},
		ValueDomains: []cli.ValueDomainInspection{
			{Uses: []cli.ValueDomainUse{{
				Destination: 1,
				Kind:        cli.ValueDomainUseKind(1),
			}}},
		},
	}
	got := inspectionExpectationOutcome(
		t,
		ValueByID(1).Domain(cli.ValueDomainUseKind(1)).DependenciesExactly(ValueLocator{}),
		inspection,
	)
	if got.status != evaluationDisproved || renderEvaluationField(got.observed) != "invalid value locator in expected collection" {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestValueCandidateDomainEntryRejectsSemanticMismatch(t *testing.T) {
	t.Parallel()

	inspection := cli.Inspection{
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{1: {ID: 1}},
	}
	got := inspectionExpectationOutcome(t, ValueCandidateByID(1).DomainEntry(ValueDomainEntryLocator{}), inspection)
	if got.status != evaluationDisproved {
		t.Fatalf("outcome = %#v", got)
	}
}

func TestComponentDomainEntryPreservesSemanticStatus(t *testing.T) {
	t.Parallel()

	componentInspection := cli.Inspection{
		Values: map[cli.ValueID]cli.ValueInspection{1: {
			ID:         1,
			Candidates: []cli.ValueCandidateID{2},
		}},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			2: {
				ID:          2,
				Destination: 1,
				Elements:    []cli.ValueComponentInspection{{DomainEntry: 1}},
			},
		},
	}
	t.Run("componentdomainentry unresolved: incomplete observation", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			valueComponentDomainEntryExpectation(ValueComponentLocator{}, ValueDomainEntryByID(1)),
			cli.Inspection{},
		)
		if got.status != evaluationDisproved {
			t.Fatalf("invalid component outcome = %#v", got)
		}
		inspection := componentInspection
		inspection.Truncated = true
		got = inspectionExpectationOutcome(
			t,
			valueComponentDomainEntryExpectation(
				ValueByID(1).CandidateAt(0).ElementAt(0).ValueComponentLocator,
				ValueByID(9).Domain(cli.ValueDomainUseKind(1)).Entry("missing"),
			),
			inspection,
		)
		if got.status != evaluationNotProvable {
			t.Fatalf("not provable entry outcome = %#v", got)
		}
	})
	t.Run("componentdomainentry unresolved: semantic mismatch", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			valueComponentDomainEntryExpectation(
				ValueByID(1).CandidateAt(0).ElementAt(0).ValueComponentLocator,
				ValueDomainEntryLocator{},
			),
			componentInspection,
		)
		if got.status != evaluationDisproved {
			t.Fatalf("invalid entry outcome = %#v", got)
		}
	})
	t.Run("semantic mismatch", func(t *testing.T) {
		t.Parallel()
		got := inspectionExpectationOutcome(
			t,
			valueComponentDomainEntryExpectation(
				ValueByID(1).CandidateAt(0).ElementAt(0).ValueComponentLocator,
				ValueDomainEntryByID(2),
			),
			componentInspection,
		)
		if got.status != evaluationDisproved {
			t.Fatalf("mismatch outcome = %#v", got)
		}
	})
}

func TestValueInspectionInternalStateFailsClosed(t *testing.T) {
	t.Parallel()
	wrongCandidate := &valueCandidateLocatorState{mode: valueCandidateLocatorMode(255)}
	if got := resolveCandidate(wrongCandidate, cli.Inspection{}); got.status != locatorResolutionInvalid {
		t.Fatalf("candidate = %#v", got)
	}
	wrongComponent := &valueComponentLocatorState{mode: valueComponentLocatorMode(255)}
	if got := resolveComponent(wrongComponent, cli.Inspection{}); got.status != locatorResolutionInvalid {
		t.Fatalf("component = %#v", got)
	}

	inspection := cli.Inspection{
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			1: {
				ID:       1,
				Elements: []cli.ValueComponentInspection{{}},
			},
		},
	}
	declaration := FlagLocator{}
	declarationEvaluator, ok := valueCandidateDeclarationExpectation(
		ValueCandidateByID(1),
		declaration,
	).(inspectionEvaluator)
	if !ok {
		t.Fatal("valueCandidateDeclarationExpectation must satisfy inspectionEvaluator")
	}
	if got := declarationEvaluator.evaluateInspection(
		inspection,
		nil,
	); got.status != evaluationDisproved ||
		renderEvaluationField(got.observed) != "locator does not identify a declaration" {
		t.Fatalf("candidate declaration = %#v", got)
	}
	entry := ValueDomainEntryLocator{}
	domainEntryEvaluator, ok := valueCandidateDomainEntryExpectation(
		ValueCandidateByID(1),
		entry,
	).(inspectionEvaluator)
	if !ok {
		t.Fatal("valueCandidateDomainEntryExpectation must satisfy inspectionEvaluator")
	}
	if got := domainEntryEvaluator.evaluateInspection(
		inspection,
		nil,
	); got.status != evaluationDisproved ||
		renderEvaluationField(got.observed) != "invalid value-domain-entry locator" {
		t.Fatalf("candidate domain entry = %#v", got)
	}
	component := componentDomainEntryExpectation{
		locator: ValueCandidateByID(1).ElementAt(0).state,
	}
	if got := component.evaluateInspection(
		inspection,
		nil,
	); got.status != evaluationDisproved ||
		renderEvaluationField(got.observed) != "invalid value-domain-entry locator" {
		t.Fatalf("component domain entry = %#v", got)
	}
}

func TestInspectionValueRelationsResolveSemanticValueLocators(t *testing.T) {
	t.Parallel()

	const valueID cli.ValueID = 7
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:     1,
			Path:   nil,
			Values: []cli.ValueID{valueID},
		}},
		Values: map[cli.ValueID]cli.ValueInspection{
			valueID: {
				ID: valueID,
				Producers: []cli.ValueProducerInspection{
					{
						Kind:         cli.ValueProducerKindInvocationDefault,
						Dependencies: []cli.ValueID{valueID},
					},
				},
				Candidates: []cli.ValueCandidateID{9},
			},
		},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			9: {
				ID:            9,
				Destination:   valueID,
				ProducerIndex: 0,
			},
		},
	}
	value := RootCommand().ValueAt(0)
	producer := value.ProducerAt(0)
	candidate := value.CandidateAt(0)
	for _, expectation := range []InspectionExpectation{
		value.ProducersExactly(producer),
		producer.DependenciesExactly(value),
		candidate.ProducedBy(producer),
	} {
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("expectation outcome = %#v", got)
		}
	}
}

func TestInspectionValueDomainRelationResolvesDestinationIdentityWithoutValueDetails(t *testing.T) {
	t.Parallel()

	const valueID cli.ValueID = 7
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:     1,
			Path:   nil,
			Values: []cli.ValueID{valueID},
		}},
		ValueDomains: []cli.ValueDomainInspection{{
			Outcome: cli.ValueDomainOutcomeFailed,
			Uses: []cli.ValueDomainUse{{
				Destination: valueID,
				Kind:        cli.ValueDomainUseDestination,
			}},
		}},
	}
	domain := RootCommand().ValueAt(0).Domain(cli.ValueDomainUseDestination)
	if got := inspectionExpectationOutcome(
		t,
		domain.Outcome(cli.ValueDomainOutcomeFailed),
		inspection,
	); got.status != evaluationProved {
		t.Fatalf("domain relation outcome = %#v", got)
	}
}
