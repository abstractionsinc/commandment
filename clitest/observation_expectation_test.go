package clitest

import (
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestCompletionObservationExpectationsPreserveSnapshotsAndIdentity(t *testing.T) {
	t.Parallel()

	completion := cli.Completion{
		Targets: []cli.CompletionTarget{{
			Kind:    cli.CompletionTargetFlag,
			Command: 41,
			ValueID: 51,
		}},
		Selection: cli.CommandSelection{
			Selected: 41,
		},
		Items: []cli.CompletionItem{{
			Value: "candidate",
			Declaration: cli.DeclarationIdentity{
				Kind: cli.DeclarationKindFlag,
				Flag: 7,
			},
		}},
		Groups: []cli.CompletionGroup{{ID: "group"}},
		NativeRequests: []cli.CompletionNativeRequest{{
			Kind:        cli.CompletionNativeFiles,
			TargetIndex: 0,
		}},
		Messages: []cli.CompletionMessage{{
			Kind: cli.CompletionMessageWarning,
			Text: "warning",
		}},
		Partial:   true,
		Truncated: true,
	}
	// Observed-snapshot exactness remains meaningful on partial/truncated results.
	message := cli.CompletionMessage{
		Kind: cli.CompletionMessageWarning,
		Text: "warning",
	}
	for _, expectation := range []CompletionExpectation{
		CompletionObservedTargetsExactly(CompletionTargetAt(0)),
		CompletionObservedItemsExactly(CompletionItemAt(0)),
		CompletionObservedGroupsExactly(CompletionGroupAt(0)),
		CompletionObservedNativeRequestsExactly(CompletionNativeRequestAt(0)),
		CompletionObservedMessagesExactly(message),
		CompletionItemAt(0).DeclarationKind(cli.DeclarationKindFlag),
		CompletionItemAt(0).Ungrouped(),
		CompletionTargetAt(0).OnSelectedCommand(),
		CompletionTargetAt(0).CommandIdentityPresent(),
		CompletionTargetAt(0).ValueIdentityPresent(),
	} {
		assertCompletionExpectation(t, expectation, completion)
	}

	if outcome := evaluateCompletionExpectation(
		t,
		CompletionTargetsExactly(CompletionTargetAt(0)),
		completion,
	); outcome.status != evaluationNotProvable {
		t.Fatalf("authoritative-complete target exactness on partial/truncated completion = %#v", outcome)
	}

	complete := completion
	complete.Partial = false
	complete.Truncated = false
	complete.Items[0].GroupID = ""
	for _, expectation := range []CompletionExpectation{
		CompletionNotPartial(),
		CompletionNotTruncated(),
		CompletionNoRefreshOnInput(),
	} {
		assertCompletionExpectation(t, expectation, complete)
	}

	for name, c := range map[string]cli.Completion{
		"zero-zero": {Targets: []cli.CompletionTarget{{Kind: cli.CompletionTargetFlag}}},
		"zero-nonzero": {
			Targets: []cli.CompletionTarget{{Kind: cli.CompletionTargetFlag}},
			Selection: cli.CommandSelection{
				Selected: 42,
			},
		},
		"nonzero-zero": {Targets: []cli.CompletionTarget{{
			Kind:    cli.CompletionTargetFlag,
			Command: 41,
		}}},
		"nonzero-unequal": {
			Targets: []cli.CompletionTarget{{
				Kind:    cli.CompletionTargetFlag,
				Command: 41,
			}},
			Selection: cli.CommandSelection{
				Selected: 42,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			outcome := evaluateCompletionExpectation(t, CompletionTargetAt(0).OnSelectedCommand(), c)
			if outcome.status != evaluationDisproved {
				t.Fatalf("OnSelectedCommand = %#v", outcome)
			}
			serialized := renderEvaluationField(outcome.subject) + renderEvaluationField(outcome.expected) + renderEvaluationField(outcome.observed)
			for _, rawID := range []string{"41", "42", "51"} {
				if strings.Contains(serialized, rawID) {
					t.Fatalf("failure output exposed raw ID %q: %#v", rawID, outcome)
				}
			}
		})
	}

	// Provider/materialization identity is operation-local: no static Schema is involved.
	providerProduced := cli.Completion{
		Targets: []cli.CompletionTarget{{
			Kind:    cli.CompletionTargetCommand,
			Command: 9001,
		}},
		Selection: cli.CommandSelection{
			Selected: 9001,
		},
	}
	assertCompletionExpectation(t, CompletionTargetAt(0).OnSelectedCommand(), providerProduced)

	if outcome := evaluateCompletionExpectation(
		t,
		CompletionTargetAt(1).OnSelectedCommand(),
		providerProduced,
	); outcome.status != evaluationDisproved {
		t.Fatalf("missing target did not fail closed: %#v", outcome)
	}
	ambiguous := cli.Completion{
		Targets: []cli.CompletionTarget{
			{
				Kind:    cli.CompletionTargetFlag,
				Command: 1,
			},
			{
				Kind:    cli.CompletionTargetFlag,
				Command: 1,
			},
		},
		Selection: cli.CommandSelection{
			Selected: 1,
		},
	}
	if outcome := evaluateCompletionExpectation(
		t,
		CompletionTarget(cli.CompletionTargetFlag).OnSelectedCommand(),
		ambiguous,
	); outcome.status != evaluationDisproved {
		t.Fatalf("ambiguous target did not fail closed: %#v", outcome)
	}

	commandOnly := cli.Completion{
		Targets: []cli.CompletionTarget{{
			Kind:    cli.CompletionTargetFlag,
			Command: 1,
		}},
	}
	assertCompletionExpectation(t, CompletionTargetAt(0).CommandIdentityPresent(), commandOnly)
	if outcome := evaluateCompletionExpectation(
		t,
		CompletionTargetAt(0).ValueIdentityPresent(),
		commandOnly,
	); outcome.status != evaluationDisproved {
		t.Fatalf("value identity unexpectedly present: %#v", outcome)
	}
	valueOnly := cli.Completion{
		Targets: []cli.CompletionTarget{{
			Kind:    cli.CompletionTargetFlag,
			ValueID: 1,
		}},
	}
	assertCompletionExpectation(t, CompletionTargetAt(0).ValueIdentityPresent(), valueOnly)
	if outcome := evaluateCompletionExpectation(
		t,
		CompletionTargetAt(0).CommandIdentityPresent(),
		valueOnly,
	); outcome.status != evaluationDisproved {
		t.Fatalf("command identity unexpectedly present: %#v", outcome)
	}
}

func TestInspectionObservationExpectationsPreserveLexicalDiagnosticNoticeSourceAndSignalFacts(t *testing.T) {
	t.Parallel()

	var sourceID cli.InputSourceID
	span := cli.InputSpan{
		Source: sourceID,
		Start: cli.InputPosition{
			Line:   3,
			Column: 5,
		},
	}
	sensitivePayload := "do-not-print-this-sensitive-payload"
	inspection := cli.Inspection{
		Status: cli.InspectionStatusReady,
		Tokens: []cli.TokenInspection{{
			Text:     "--raw",
			Redacted: false,
		}},
		Values: map[cli.ValueID]cli.ValueInspection{
			1: {
				ID:                1,
				Candidates:        []cli.ValueCandidateID{11},
				DiagnosticIndexes: []int{0},
			},
			2: {ID: 2},
		},
		ValueCandidates: map[cli.ValueCandidateID]cli.ValueCandidateInspection{
			11: {
				ID:          11,
				Destination: 1,
				Input: &cli.ValueInputInspection{
					Text:     "raw",
					Redacted: true,
				},
				Representation: &cli.ValueRepresentation{
					Kind:     cli.ValueRepresentationText,
					Text:     "canonical",
					Redacted: true,
				},
				Sensitive: true,
				Elements: []cli.ValueComponentInspection{{Input: &cli.ValueInputInspection{
					Text: "part",
				}}},
				DiagnosticIndexes: []int{0},
			},
		},
		Sources: map[cli.InputSourceID]cli.InputSource{
			sourceID: {
				ID:   sourceID,
				Kind: cli.InputKindArguments,
				Key:  "argv",
			},
		},
		Diagnostics: []cli.Diagnostic{{
			Code:   cli.DiagnosticInvalidValue,
			Reason: "because",
			Span:   &span,
		}},
		Notices: []cli.NoticeInspection{{
			Code:      cli.NoticeAdvisory,
			Message:   "message",
			Reason:    "reason",
			Hint:      "hint",
			Sensitive: true,
		}},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{7: {ID: 7}},
		ValueDomains: []cli.ValueDomainInspection{
			{Uses: []cli.ValueDomainUse{{
				Destination: 1,
				Kind:        cli.ValueDomainUseDestination,
			}}},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntryInspection{9: {
			ID:   9,
			Name: "entry",
		}},
		Commands: map[cli.CommandID]cli.CommandInspection{1: {
			ID:   1,
			Name: "root",
		}},
		SignalEndpoints: map[cli.SignalEndpointID]cli.SignalEndpointInspection{
			1: {
				SignalEndpoint: cli.SignalEndpoint{
					ID: 1,
				},
				Status: cli.SignalEndpointStatusReady,
			},
			2: {
				SignalEndpoint: cli.SignalEndpoint{
					ID: 2,
				},
				Status: cli.SignalEndpointStatusBlocked,
			},
		},
		SignalSource: cli.SignalSourceInspection{
			Status:                    cli.SignalIntegrationStatusReady,
			Described:                 true,
			AvailableHostCapabilities: cli.SignalHostCapabilityAbort,
		},
	}

	for _, expectation := range []InspectionExpectation{
		TokenAt(0).Text("--raw"),
		ValueCandidateByID(11).InputText("raw"),
		ValueCandidateByID(11).InputRedacted(),
		ValueCandidateByID(11).Redacted(),
		ValueCandidateByID(11).ElementAt(0).InputText("part"),
		ValueByID(1).DiagnosticsExactly(Diagnostic(cli.DiagnosticInvalidValue)),
		ValueCandidateByID(11).DiagnosticsExactly(Diagnostic(cli.DiagnosticInvalidValue)),
		ValueByID(2).NoRepresentation(),
		Notice(cli.NoticeAdvisory).Message("message"),
		Notice(cli.NoticeAdvisory).Reason("reason"),
		Notice(cli.NoticeAdvisory).Hint("hint"),
		Notice(cli.NoticeAdvisory).SubjectRedacted(),
		InspectionObservedInputSourcesExactly(cli.InputKindArguments, InputSource(cli.InputKindArguments)),
		InspectionObservedCommandProvidersExactly(CommandProviderByID(7)),
		InspectionObservedValueDomainsExactly(ValueByID(1).Domain(cli.ValueDomainUseDestination)),
		InspectionObservedValueDomainEntriesExactly(ValueDomainEntryByID(9)),
		CommandByID(2).ObservedDoesNotExist(),
		InspectionObservedSignalEndpointCount(2),
		InspectionSignalEndpoint(cli.SignalEndpointStatusBlocked).Status(cli.SignalEndpointStatusBlocked),
		SignalIntegration().AvailableHostCapabilities(cli.SignalHostCapabilityAbort),
		Diagnostic(cli.DiagnosticInvalidValue).SpanPositioned(),
		Diagnostic(cli.DiagnosticInvalidValue).Reason("because"),
		Diagnostic(cli.DiagnosticInvalidValue).SpanSource().Kind(cli.InputKindArguments),
		Diagnostic(cli.DiagnosticInvalidValue).SignalOverflow().DoesNotExist(),
	} {
		assertInspectionExpectation(t, expectation, inspection)
	}

	checkedObservedCommandOutcome, ok := CommandByID(1).ObservedDoesNotExist().(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("ObservedDoesNotExist must satisfy expected evaluator interface")
	}
	observedCommandOutcome := checkedObservedCommandOutcome.evaluateInspection(inspection, nil)
	if observedCommandOutcome.status != evaluationDisproved {
		t.Fatalf("observed command unexpectedly satisfied absence expectation: %#v", observedCommandOutcome)
	}

	incomplete := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			1: {
				ID:                    1,
				Path:                  nil,
				NamespaceCompleteness: cli.CommandNamespaceIncomplete,
			},
		},
	}
	for _, command := range []CommandLocator{RootCommand().Command("rejected"), Command("rejected")} {
		checkedOutcome, ok := command.ObservedDoesNotExist().(interface {
			evaluateInspection(inspection cli.Inspection, err error) evaluationResult
		})
		if !ok {
			t.Fatal("ObservedDoesNotExist must satisfy expected evaluator interface")
		}
		outcome := checkedOutcome.evaluateInspection(incomplete, nil)
		if outcome.status != evaluationProved {
			t.Fatalf("observed absence under incomplete namespace = %#v", outcome)
		}
	}

	completion := cli.Completion{
		Diagnostics: inspection.Diagnostics,
		Sources:     inspection.Sources,
	}
	assertCompletionExpectation(
		t,
		CompletionObservedInputSourcesExactly(cli.InputKindArguments, InputSourceByKey(cli.InputKindArguments, "argv")),
		completion,
	)
	assertCompletionExpectation(t, Diagnostic(cli.DiagnosticInvalidValue).Reason("because"), completion)

	missingSource := InputSource(cli.InputKindEnvironment)
	checkedOutcome, ok := missingSource.Kind(cli.InputKindEnvironment).(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Kind must satisfy expected evaluator interface")
	}
	outcome := checkedOutcome.evaluateInspection(inspection, nil)
	if outcome.status == evaluationProved {
		t.Fatalf("missing input-source locator unexpectedly satisfied: %#v", outcome)
	}

	// Sensitive failure paths do not disclose the caller-controlled payload.
	sensitiveInspection := inspection
	sensitiveInspection.Tokens = []cli.TokenInspection{{
		Text:     sensitivePayload,
		Redacted: true,
	}}
	checkedTokenOutcome, ok := TokenAt(0).Text("different").(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Text must satisfy expected evaluator interface")
	}
	tokenOutcome := checkedTokenOutcome.evaluateInspection(sensitiveInspection, nil)
	if strings.Contains(renderEvaluationField(tokenOutcome.subject)+renderEvaluationField(tokenOutcome.expected)+renderEvaluationField(tokenOutcome.observed), sensitivePayload) {
		t.Fatalf("token failure leaked sensitive text: %#v", tokenOutcome)
	}
	checkedCandidateOutcome, ok := ValueCandidateByID(11).InputText("different").(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("InputText must satisfy expected evaluator interface")
	}
	candidateOutcome := checkedCandidateOutcome.evaluateInspection(inspection, nil)
	if strings.Contains(renderEvaluationField(candidateOutcome.subject)+renderEvaluationField(candidateOutcome.expected)+renderEvaluationField(candidateOutcome.observed), "raw") {
		t.Fatalf("candidate failure leaked sensitive input: %#v", candidateOutcome)
	}

	// A semantic status selector is exact-one and must reject ambiguity instead of map iteration order.
	ambiguous := inspection
	ambiguous.SignalEndpoints = map[cli.SignalEndpointID]cli.SignalEndpointInspection{
		1: {
			SignalEndpoint: cli.SignalEndpoint{
				ID: 1,
			},
			Status: cli.SignalEndpointStatusBlocked,
		},
		2: {
			SignalEndpoint: cli.SignalEndpoint{
				ID: 2,
			},
			Status: cli.SignalEndpointStatusBlocked,
		},
	}
	checkedEndpointOutcome, ok := InspectionSignalEndpoint(cli.SignalEndpointStatusBlocked).Status(
		cli.SignalEndpointStatusBlocked,
	).(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("Status must satisfy expected evaluator interface")
	}
	endpointOutcome := checkedEndpointOutcome.evaluateInspection(
		ambiguous,
		nil,
	)
	if endpointOutcome.status == evaluationProved {
		t.Fatalf("ambiguous signal endpoint selector unexpectedly satisfied: %#v", endpointOutcome)
	}

	withOverflow := inspection
	withOverflow.Diagnostics = []cli.Diagnostic{
		{
			Code: cli.DiagnosticInvalidValue,
			SignalOverflow: &cli.SignalOverflowFailure{
				Boundary: cli.SignalOverflowBoundaryRouting,
				Capacity: 4,
				Buffered: 4,
			},
		},
	}
	checkedOverflowOutcome, ok := Diagnostic(cli.DiagnosticInvalidValue).SignalOverflow().DoesNotExist().(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !ok {
		t.Fatal("DoesNotExist must satisfy expected evaluator interface")
	}
	overflowOutcome := checkedOverflowOutcome.evaluateInspection(withOverflow, nil)
	if overflowOutcome.status != evaluationDisproved {
		t.Fatalf("signal-overflow structured presence was not disproved: %#v", overflowOutcome)
	}
}

func TestDeclarationSchemaExpectationsCoverStaticDeclarationFamilies(t *testing.T) {
	t.Parallel()

	root := cli.CommandID(1)
	flagID := cli.FlagID(2)
	spellingID := cli.FlagSpellingID(3)
	termID := cli.ArgumentTermID(4)
	entryID := cli.CommandEntryID(5)
	envID := cli.EnvironmentInputID(6)
	actionLong := cli.ActionEntryID(7)
	actionShort := cli.ActionEntryID(8)
	actionLiteral := cli.ActionEntryID(9)
	actionCommand := cli.ActionEntryID(10)
	factID := cli.ValueID(20)
	dependencyA := cli.ValueID(21)
	dependencyB := cli.ValueID(22)
	valueID := cli.ValueID(23)
	domainEntryID := cli.ValueDomainEntryID(30)
	native := cli.CompletionNativeIntent{
		Kind: cli.CompletionNativeFiles,
		Mode: cli.CompletionNativeAlways,
	}

	schema := &cli.Schema{
		Root: root,
		Commands: map[cli.CommandID]cli.CommandSchema{
			root: {
				ID:            root,
				Name:          "root",
				ActionEntries: []cli.ActionEntryID{actionLong, actionShort, actionLiteral, actionCommand},
			},
		},
		CommandEntries: map[cli.CommandEntryID]cli.CommandEntrySchema{
			entryID: {
				ID:     entryID,
				Parent: root,
				Description: cli.Description{
					Short: "source command entry",
				},
			},
		},
		ActionEntries: map[cli.ActionEntryID]cli.ActionEntrySchema{
			actionLong: {
				ID:        actionLong,
				Parent:    root,
				Kind:      cli.ActionEntryFlag,
				Action:    cli.InvocationActionHelp,
				Long:      "help",
				Placement: cli.FlagPlacementCommand,
				Description: cli.Description{
					Short: "long help",
				},
			},
			actionShort: {
				ID:        actionShort,
				Parent:    root,
				Kind:      cli.ActionEntryFlag,
				Action:    cli.InvocationActionHelp,
				Short:     'h',
				Placement: cli.FlagPlacementCommand,
				Description: cli.Description{
					Short: "short help",
				},
			},
			actionLiteral: {
				ID:        actionLiteral,
				Parent:    root,
				Kind:      cli.ActionEntryFlag,
				Action:    cli.InvocationActionHelp,
				Literal:   "?",
				Placement: cli.FlagPlacementCommand,
				Description: cli.Description{
					Short: "literal help",
				},
			},
			actionCommand: {
				ID:     actionCommand,
				Parent: root,
				Kind:   cli.ActionEntryCommand,
				Action: cli.InvocationActionHelp,
				Name:   "help",
				Description: cli.Description{
					Short: "command help",
				},
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{
			flagID: {
				ID:                  flagID,
				Command:             root,
				Spellings:           []cli.FlagSpellingID{spellingID},
				ValueName:           "VALUE",
				Description:         "flag description",
				Occurrences:         cli.Count.Between(1, 2),
				ValuesPerOccurrence: cli.Count.Exactly(1),
				Placement:           cli.FlagPlacementCommand,
				LocaleSelector:      true,
				Completion: cli.CompletionDeclarationCapabilities{
					Dynamic:      true,
					Delegated:    true,
					Native:       &native,
					Dependencies: []cli.ValueID{dependencyA, dependencyB},
				},
			},
		},
		FlagSpellings: map[cli.FlagSpellingID]cli.FlagSpellingSchema{
			spellingID: {
				ID:   spellingID,
				Flag: flagID,
				Spelling: cli.FlagSpelling{
					Kind:       cli.FlagSpellingLong,
					Text:       "flag",
					ValueForms: []cli.FlagValueForm{cli.FlagValueFormSeparate, cli.FlagValueFormAssigned},
				},
			},
		},
		ArgumentTerms: map[cli.ArgumentTermID]cli.ArgumentTermSchema{
			termID: {
				ID:              termID,
				Name:            "arg",
				Description:     "argument description",
				HyphenValues:    cli.HyphenValueAllow,
				Minimal:         true,
				EndOfOptions:    true,
				SourceSensitive: true,
				TokenLanguage: &cli.TokenLanguageSyntax{
					Fragments: []string{"a", "b"},
				},
				Completion: cli.CompletionDeclarationCapabilities{
					Dynamic:      true,
					Dependencies: []cli.ValueID{dependencyA},
				},
			},
		},
		EnvironmentInputs: map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{
			envID: {
				ID:        envID,
				Command:   root,
				Name:      "APP_",
				Prefix:    true,
				Delimiter: ":",
			},
		},
		Values: map[cli.ValueID]cli.ValueSchema{
			dependencyA: {ID: dependencyA},
			dependencyB: {ID: dependencyB},
			valueID: {
				ID:           valueID,
				Dependencies: []cli.ValueID{dependencyA, dependencyB},
			},
			factID: {ID: factID},
		},
		Facts: map[cli.ValueID]cli.FactSchema{
			factID: {
				ID:           factID,
				Dependencies: []cli.ValueID{dependencyA, dependencyB},
			},
		},
		ValueDomains: []cli.ValueDomainSchema{
			{
				Uses: []cli.ValueDomainUse{
					{
						Destination: valueID,
						Kind:        cli.ValueDomainUseDestination,
					},
					{
						Destination: dependencyA,
						Kind:        cli.ValueDomainUseMapKey,
					},
				},
				Entries: []cli.ValueDomainEntryID{domainEntryID},
			},
		},
		ValueDomainEntries: map[cli.ValueDomainEntryID]cli.ValueDomainEntrySchema{
			domainEntryID: {
				ID:   domainEntryID,
				Name: "entry",
			},
		},
	}

	for _, expectation := range []DeclarationExpectation{
		FlagByID(flagID).LocaleSelector(),
		CommandByID(root).Name("root"),
		CommandByID(root).EnvironmentInputsExactly(EnvironmentInputByID(envID)),
		EnvironmentInputByID(envID).Prefix(),
		EnvironmentInputByID(envID).Delimiter(":"),
		FlagByID(flagID).Occurrences(cli.Count.Between(1, 2)),
		FlagByID(flagID).ValuesPerOccurrence(cli.Count.Exactly(1)),
		FlagByID(flagID).Placement(cli.FlagPlacementCommand),
		FlagByID(flagID).ValueName("VALUE"),
		FlagByID(flagID).Description("flag description"),
		ArgumentTermByID(termID).HyphenValues(cli.HyphenValueAllow),
		ArgumentTermByID(termID).Description("argument description"),
		ArgumentTermByID(termID).Minimal(),
		ArgumentTermByID(termID).EndOfOptions(),
		ArgumentTermByID(termID).SourceSensitive(),
		ArgumentTermByID(termID).HasTokenLanguage(),
		ArgumentTermByID(termID).TokenLanguageFragmentsExactly("a", "b"),
		FlagSpellingByID(spellingID).ValueFormsExactly(cli.FlagValueFormSeparate, cli.FlagValueFormAssigned),
		FlagByID(flagID).Completion().Dynamic(),
		FlagByID(flagID).Completion().Delegated(),
		FlagByID(flagID).Completion().Native(cli.CompletionNativeFiles, cli.CompletionNativeAlways),
		FlagByID(flagID).Completion().DependenciesExactly(ValueByID(dependencyA), ValueByID(dependencyB)),
		ArgumentTermByID(termID).Completion().Dynamic(),
		ArgumentTermByID(termID).Completion().DependenciesExactly(ValueByID(dependencyA)),
		FactByID(factID).DependenciesExactly(ValueByID(dependencyA), ValueByID(dependencyB)),
		CommandByID(root).ActionEntries(cli.InvocationActionHelp).Exactly(
			CommandByID(root).ActionEntries(cli.InvocationActionHelp).Long("help"),
			CommandByID(root).ActionEntries(cli.InvocationActionHelp).Short('h'),
			CommandByID(root).ActionEntries(cli.InvocationActionHelp).Literal("?"),
			CommandByID(root).ActionEntries(cli.InvocationActionHelp).Command("help"),
		),
		CommandByID(root).ActionEntries(cli.InvocationActionHelp).Long("help").Placement(cli.FlagPlacementCommand),
	} {
		if outcome := schemaExpectationOutcome(t, expectation, schema); outcome.status != evaluationProved {
			t.Fatalf("expectation %T = %#v", expectation, outcome)
		}
	}

	// Exact domain uses preserve use-kind and same-domain identity.
	domainUses := ValueByID(valueID).Domain(cli.ValueDomainUseDestination).UsesExactly(
		ValueByID(valueID).Domain(cli.ValueDomainUseDestination),
		ValueByID(dependencyA).Domain(cli.ValueDomainUseMapKey),
	)
	if outcome := schemaExpectationOutcome(t, domainUses, schema); outcome.status != evaluationProved {
		t.Fatalf("exact domain uses = %#v", outcome)
	}

	for _, expectation := range []SchemaExpectation{
		ValueByID(valueID).DependenciesExactly(ValueByID(dependencyA), ValueByID(dependencyB)),
		CommandEntryByID(entryID).SourceDescription(cli.Description{
			Short: "source command entry",
		}),
		CommandByID(root).ActionEntries(cli.InvocationActionHelp).Long("help").HasSourceShortDescription(),
		CommandByID(root).ActionEntries(cli.InvocationActionHelp).Long("help").SourceDescription(cli.Description{
			Short: "long help",
		}),
	} {
		if outcome := schemaExpectationOutcome(t, expectation, schema); outcome.status != evaluationProved {
			t.Fatalf("expectation %T = %#v", expectation, outcome)
		}
	}

	// Prefix polarity is independently established.
	notPrefixSchema := *schema
	notPrefixSchema.EnvironmentInputs = map[cli.EnvironmentInputID]cli.EnvironmentInputSchema{
		envID: {
			ID:      envID,
			Command: root,
			Name:    "APP_ENV",
			Prefix:  false,
		},
	}
	if outcome := schemaExpectationOutcome(
		t,
		EnvironmentInputByID(envID).NotPrefix(),
		&notPrefixSchema,
	); outcome.status != evaluationProved {
		t.Fatalf("not-prefix expectation = %#v", outcome)
	}

	// Action-entry lexical locators are exact-one; duplicate semantic subjects fail closed.
	ambiguous := *schema
	ambiguous.Commands = map[cli.CommandID]cli.CommandSchema{
		root: {
			ID:            root,
			Name:          "root",
			ActionEntries: []cli.ActionEntryID{actionLong, 99},
		},
	}
	ambiguous.ActionEntries = map[cli.ActionEntryID]cli.ActionEntrySchema{
		actionLong: {
			ID:     actionLong,
			Parent: root,
			Kind:   cli.ActionEntryFlag,
			Action: cli.InvocationActionHelp,
			Long:   "help",
		},
		99: {
			ID:     99,
			Parent: root,
			Kind:   cli.ActionEntryFlag,
			Action: cli.InvocationActionHelp,
			Long:   "help",
		},
	}
	ambiguousExpectation := CommandByID(
		root,
	).ActionEntries(cli.InvocationActionHelp).
		Long("help").
		Placement(cli.FlagPlacementCommand)
	checkedOutcome, ok := ambiguousExpectation.(interface {
		evaluateSchema(schema *cli.Schema) evaluationResult
	})
	if !ok {
		t.Fatal("ambiguousExpectation must satisfy expected evaluator interface")
	}
	outcome := checkedOutcome.evaluateSchema(&ambiguous)
	if outcome.status == evaluationProved {
		t.Fatalf("ambiguous action entry selector unexpectedly satisfied: %#v", outcome)
	}
}
