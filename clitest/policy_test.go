package clitest

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestStaticPolicyFamiliesUseCommandOwnedRecordsAcrossArtifacts(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	targetID := cli.CommandID(2)
	defaultID := cli.CommandDefaultID(10)
	providerID := cli.CommandProviderID(11)
	flagID := cli.FlagID(12)
	subjectID := cli.ValueID(20)
	dependencyID := cli.ValueID(21)
	ruleID := cli.RuleID(30)
	ruleCode := cli.RuleCode("required")
	advisoryCode := cli.AdvisoryCode("tip")
	match := cli.NoticeMatch{
		Code:     cli.NoticeDeprecatedFlag,
		Severity: cli.NoticeSeverityWarning,
	}

	conditionSchema := cli.ConditionSchema{
		Target: cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: flagID,
		},
		Effect:       cli.ConditionEffectAvailability,
		Reachability: cli.ReachabilityReachable,
	}
	ruleSchema := cli.RuleSchema{
		ID:           ruleID,
		Kind:         cli.RuleRequire,
		Code:         ruleCode,
		Reachability: cli.ReachabilityReachable,
		Sensitive:    true,
		Subjects:     []cli.ValueID{subjectID},
		Dependencies: []cli.ValueID{dependencyID},
	}
	advisorySchema := cli.AdvisorySchema{
		Code:         advisoryCode,
		Severity:     cli.NoticeSeverityWarning,
		Command:      commandID,
		Kind:         cli.AdvisoryKindConditional,
		Reachability: cli.ReachabilityReachable,
		Subjects:     []cli.ValueID{subjectID},
		Dependencies: []cli.ValueID{dependencyID},
	}
	suppressionSchema := cli.NoticeSuppressionSchema{
		Command:      commandID,
		Match:        match,
		Reachability: cli.ReachabilityReachable,
	}
	commandSchema := cli.CommandSchema{
		ID:                 commandID,
		CommandDefaults:    []cli.CommandDefaultID{defaultID},
		CommandProviders:   []cli.CommandProviderID{providerID},
		Conditions:         []cli.ConditionSchema{conditionSchema},
		Rules:              []cli.RuleSchema{ruleSchema},
		Advisories:         []cli.AdvisorySchema{advisorySchema},
		NoticeSuppressions: []cli.NoticeSuppressionSchema{suppressionSchema},
	}
	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{commandID: commandSchema, targetID: {
			ID: targetID,
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultSchema{
			defaultID: {
				ID:           defaultID,
				Parent:       commandID,
				Target:       targetID,
				Reachability: cli.ReachabilityReachable,
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderSchema{
			providerID: {
				ID:           providerID,
				Parent:       commandID,
				Dependencies: []cli.ValueID{dependencyID},
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{flagID: {
			ID: flagID,
		}},
		Values: map[cli.ValueID]cli.ValueSchema{
			subjectID: {
				ID: subjectID,
			},
			dependencyID: {
				ID: dependencyID,
			},
		},
	}

	commandReference := cli.CommandReference{
		ID:               commandID,
		CommandDefaults:  []cli.CommandDefaultID{defaultID},
		CommandProviders: []cli.CommandProviderID{providerID},
		Conditions:       []cli.ConditionSchema{conditionSchema},
		Rules:            []cli.RuleSchema{ruleSchema},
		Advisories: []cli.AdvisoryReference{
			{
				Advisory:     advisorySchema.Advisory,
				Command:      commandID,
				Kind:         advisorySchema.Kind,
				Reachability: advisorySchema.Reachability,
				Subjects:     append([]cli.ValueID(nil), advisorySchema.Subjects...),
				Dependencies: append([]cli.ValueID(nil), advisorySchema.Dependencies...),
			},
		},
		NoticeSuppressions: []cli.NoticeSuppressionSchema{suppressionSchema},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{commandID: commandReference, targetID: {
			ID: targetID,
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultReference{
			defaultID: {
				ID:           defaultID,
				Parent:       commandID,
				Target:       targetID,
				Reachability: cli.ReachabilityReachable,
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderReference{
			providerID: {
				ID:           providerID,
				Parent:       commandID,
				Dependencies: []cli.ValueID{dependencyID},
			},
		},
		Flags: map[cli.FlagID]cli.FlagReference{flagID: {
			ID: flagID,
		}},
		Values: map[cli.ValueID]cli.ValueReference{
			subjectID: {ValueSchema: cli.ValueSchema{
				ID: subjectID,
			}},
			dependencyID: {ValueSchema: cli.ValueSchema{
				ID: dependencyID,
			}},
		},
	}

	commandInspection := cli.CommandInspection{
		ID:               commandID,
		CommandDefaults:  []cli.CommandDefaultID{defaultID},
		CommandProviders: []cli.CommandProviderID{providerID},
		Conditions: []cli.ConditionInspection{
			{
				Target:       conditionSchema.Target,
				Effect:       conditionSchema.Effect,
				Reachability: conditionSchema.Reachability,
				Outcome:      cli.EvaluationOutcomeSatisfied,
			},
		},
		Rules: []cli.RuleInspection{
			{
				ID:                ruleID,
				Kind:              ruleSchema.Kind,
				Code:              ruleCode,
				Reachability:      ruleSchema.Reachability,
				Sensitive:         true,
				Subjects:          append([]cli.ValueID(nil), ruleSchema.Subjects...),
				Dependencies:      append([]cli.ValueID(nil), ruleSchema.Dependencies...),
				Outcome:           cli.EvaluationOutcomeSatisfied,
				ActivationOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
		Advisories: []cli.AdvisoryInspection{
			{
				Advisory:          advisorySchema.Advisory,
				Command:           commandID,
				Kind:              advisorySchema.Kind,
				Reachability:      advisorySchema.Reachability,
				Subjects:          append([]cli.ValueID(nil), advisorySchema.Subjects...),
				Dependencies:      append([]cli.ValueID(nil), advisorySchema.Dependencies...),
				Outcome:           cli.AdvisoryOutcomeMatched,
				ActivationOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
		NoticeSuppressions: []cli.NoticeSuppressionInspection{
			{
				Command:          commandID,
				Match:            match,
				Reachability:     cli.ReachabilityReachable,
				ConditionOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{commandID: commandInspection, targetID: {
			ID: targetID,
		}},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultInspection{
			defaultID: {
				ID:               defaultID,
				Parent:           commandID,
				Target:           targetID,
				Reachability:     cli.ReachabilityReachable,
				ConditionOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
		CommandProviders: map[cli.CommandProviderID]cli.CommandProviderInspection{
			providerID: {
				ID:           providerID,
				Parent:       commandID,
				Dependencies: []cli.ValueID{dependencyID},
			},
		},
		Flags: map[cli.FlagID]cli.FlagInspection{flagID: {
			ID: flagID,
		}},
		Values: map[cli.ValueID]cli.ValueInspection{
			subjectID: {
				ID: subjectID,
			},
			dependencyID: {
				ID: dependencyID,
			},
		},
	}

	command := CommandByID(commandID)
	defaultLocator := command.DefaultAt(0)
	providerLocator := command.ProviderAt(0)
	conditionLocator := command.ConditionAt(0)
	ruleLocator := command.Rule(ruleCode)
	advisoryLocator := command.Advisory(advisoryCode)
	suppressionLocator := command.NoticeSuppressionAt(0)

	expectations := []DeclarationExpectation{
		defaultLocator.Exists(),
		defaultLocator.Target(CommandByID(targetID)),
		defaultLocator.Reachability(cli.ReachabilityReachable),
		providerLocator.Exists(),
		providerLocator.DependenciesExactly(ValueByID(dependencyID)),
		conditionLocator.Exists(),
		conditionLocator.Effect(cli.ConditionEffectAvailability),
		conditionLocator.Reachability(cli.ReachabilityReachable),
		conditionLocator.Target(FlagByID(flagID)),
		ruleLocator.Exists(),
		ruleLocator.Kind(cli.RuleRequire),
		ruleLocator.Code(ruleCode),
		ruleLocator.Reachability(cli.ReachabilityReachable),
		ruleLocator.SubjectsExactly(ValueByID(subjectID)),
		ruleLocator.DependenciesExactly(ValueByID(dependencyID)),
		ruleLocator.Sensitive(),
		advisoryLocator.Exists(),
		advisoryLocator.Kind(cli.AdvisoryKindConditional),
		advisoryLocator.Code(advisoryCode),
		advisoryLocator.Severity(cli.NoticeSeverityWarning),
		advisoryLocator.Reachability(cli.ReachabilityReachable),
		advisoryLocator.SubjectsExactly(ValueByID(subjectID)),
		advisoryLocator.DependenciesExactly(ValueByID(dependencyID)),
		suppressionLocator.Exists(),
		suppressionLocator.Reachability(cli.ReachabilityReachable),
		suppressionLocator.Match(match),
	}
	for _, expectation := range expectations {
		(SchemaResult{
			state: &schemaResultState{
				observation: schema,
			},
		}).Require(t, expectation)
		(ReferenceResult{
			state: &referenceResultState{
				observation: reference,
				err:         nil,
			},
		}).Require(t, expectation)
		(InspectionResult{
			state: &inspectionResultState{
				observation: inspection,
				err:         nil,
			},
		}).Require(t, expectation)
	}
}

func TestStaticPolicyRuntimeExpectationsUseInspectionOnlyState(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	defaultID := cli.CommandDefaultID(10)
	ruleCode := cli.RuleCode("required")
	advisoryCode := cli.AdvisoryCode("tip")
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID:              commandID,
				CommandDefaults: []cli.CommandDefaultID{defaultID},
				Conditions: []cli.ConditionInspection{{
					Outcome: cli.EvaluationOutcomeSatisfied,
				}},
				Rules: []cli.RuleInspection{
					{
						Code:              ruleCode,
						Outcome:           cli.EvaluationOutcomeSatisfied,
						ActivationOutcome: cli.EvaluationOutcomeSatisfied,
					},
				},
				Advisories: []cli.AdvisoryInspection{{
					Code:              advisoryCode,
					Outcome:           cli.AdvisoryOutcomeMatched,
					ActivationOutcome: cli.EvaluationOutcomeSatisfied,
				}},
				NoticeSuppressions: []cli.NoticeSuppressionInspection{
					{
						ConditionOutcome: cli.EvaluationOutcomeSatisfied,
					},
				},
			},
		},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultInspection{
			defaultID: {
				ID:               defaultID,
				ConditionOutcome: cli.EvaluationOutcomeSatisfied,
			},
		},
	}
	command := CommandByID(commandID)
	(InspectionResult{
		state: &inspectionResultState{
			observation: inspection,
			err:         nil,
		},
	}).Require(t,
		command.DefaultAt(0).ConditionOutcome(cli.EvaluationOutcomeSatisfied),
		command.ConditionAt(0).Outcome(cli.EvaluationOutcomeSatisfied),
		command.Rule(ruleCode).Outcome(cli.EvaluationOutcomeSatisfied),
		command.Rule(ruleCode).ActivationOutcome(cli.EvaluationOutcomeSatisfied),
		command.Advisory(advisoryCode).Outcome(cli.AdvisoryOutcomeMatched),
		command.Advisory(advisoryCode).ActivationOutcome(cli.EvaluationOutcomeSatisfied),
		command.NoticeSuppressionAt(0).ConditionOutcome(cli.EvaluationOutcomeSatisfied))
}

func TestStaticPolicyExactCollectionsRefuseTruncatedMissingOwner(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	expectation := CommandByID(commandID).Rule(cli.RuleCode("required")).DependenciesExactly(ValueByID(2))
	inspection := cli.Inspection{
		Truncated: true,
		Commands: map[cli.CommandID]cli.CommandInspection{commandID: {
			ID: commandID,
		}},
	}
	evaluator, evaluatorOK := expectation.(interface {
		evaluateInspection(inspection cli.Inspection, err error) evaluationResult
	})
	if !evaluatorOK {
		t.Fatal("expectation does not expose inspection evaluation")
	}
	outcome := evaluator.evaluateInspection(inspection, nil)
	if outcome.status != evaluationNotProvable {
		t.Fatalf("truncated static-policy collection = %#v, want not provable", outcome)
	}
}

func TestStaticPolicyNegativeExistencePublicSurface(t *testing.T) {
	t.Parallel()
	commandID := cli.CommandID(1)
	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{commandID: {
			ID: commandID,
		}},
	}
	reference := cli.Reference{
		Commands: map[cli.CommandID]cli.CommandReference{commandID: {
			ID: commandID,
		}},
	}
	inspection := cli.Inspection{
		Commands: map[cli.CommandID]cli.CommandInspection{commandID: {
			ID:                    commandID,
			NamespaceCompleteness: cli.CommandNamespaceComplete,
		}},
	}
	command := CommandByID(commandID)
	expectations := []DeclarationExpectation{
		CommandDefaultByID(90).DoesNotExist(),
		CommandProviderByID(91).DoesNotExist(),
		command.ConditionAt(0).DoesNotExist(),
		command.Rule("missing").DoesNotExist(),
		command.Advisory("missing").DoesNotExist(),
		command.NoticeSuppressionAt(0).DoesNotExist(),
	}
	for _, expectation := range expectations {
		if got := schemaExpectationOutcome(t, expectation, &schema); got.status != evaluationProved {
			t.Fatalf("schema %T = %#v", expectation, got)
		}
		if got := referenceExpectationOutcome(t, expectation, reference); got.status != evaluationProved {
			t.Fatalf("reference %T = %#v", expectation, got)
		}
		if got := inspectionExpectationOutcome(t, expectation, inspection); got.status != evaluationProved {
			t.Fatalf("inspection %T = %#v", expectation, got)
		}
	}
}

func TestStaticPolicyTypedResolversPreserveInvalidAndClosureStates(t *testing.T) {
	t.Parallel()

	commandID := cli.CommandID(1)
	inspection := cli.Inspection{
		Truncated: true,
		Commands: map[cli.CommandID]cli.CommandInspection{
			commandID: {
				ID: commandID,
			},
		},
	}

	cases := []struct {
		name   string
		status locatorResolutionStatus
	}{
		{
			name:   "default negative index",
			status: resolveInspectionCommandDefaultLocator(CommandByID(commandID).DefaultAt(-1).state, inspection).status,
		},
		{
			name:   "provider negative index",
			status: resolveInspectionCommandProviderLocator(CommandByID(commandID).ProviderAt(-1).state, inspection).status,
		},
		{
			name:   "condition negative index",
			status: resolveInspectionConditionLocator(CommandByID(commandID).ConditionAt(-1).state, inspection).status,
		},
		{
			name:   "rule negative index",
			status: resolveInspectionRuleLocator(CommandByID(commandID).RuleAt(-1).state, inspection).status,
		},
		{
			name:   "advisory negative index",
			status: resolveInspectionAdvisoryLocator(CommandByID(commandID).AdvisoryAt(-1).state, inspection).status,
		},
		{
			name:   "suppression negative index",
			status: resolveInspectionNoticeSuppressionLocator(CommandByID(commandID).NoticeSuppressionAt(-1).state, inspection).status,
		},
	}
	for _, tc := range cases {
		if tc.status != locatorResolutionInvalid {
			t.Fatalf("%s status = %v, want invalid", tc.name, tc.status)
		}
	}

	if got := resolveInspectionRuleLocator(CommandByID(commandID).Rule("missing").state, inspection).status; got != locatorResolutionNotProvable {
		t.Fatalf("truncated rule lookup = %v, want not provable", got)
	}
	if got := resolveInspectionAdvisoryLocator(CommandByID(commandID).Advisory("missing").state, inspection).status; got != locatorResolutionNotProvable {
		t.Fatalf("truncated advisory lookup = %v, want not provable", got)
	}
}

func TestStaticPolicyTypedRelationsPreserveTargetAndMatchSemantics(t *testing.T) {
	t.Parallel()

	commandID := cli.CommandID(1)
	targetID := cli.CommandID(2)
	defaultID := cli.CommandDefaultID(3)
	flagID := cli.FlagID(4)
	match := cli.NoticeMatch{
		Code: cli.NoticeDeprecatedFlag,
	}

	schema := cli.Schema{
		Commands: map[cli.CommandID]cli.CommandSchema{
			commandID: {
				ID: commandID,
				Conditions: []cli.ConditionSchema{{
					Target: cli.DeclarationIdentity{
						Kind: cli.DeclarationKindFlag,
						Flag: flagID,
					},
				}},
				NoticeSuppressions: []cli.NoticeSuppressionSchema{{
					Match: match,
				}},
			},
			targetID: {
				ID: targetID,
			},
		},
		CommandDefaults: map[cli.CommandDefaultID]cli.CommandDefaultSchema{
			defaultID: {
				ID:     defaultID,
				Parent: commandID,
				Target: targetID,
			},
		},
		Flags: map[cli.FlagID]cli.FlagSchema{flagID: {
			ID: flagID,
		}},
	}

	expectations := []SchemaExpectation{
		CommandDefaultByID(defaultID).Target(CommandByID(targetID)),
		CommandByID(commandID).ConditionAt(0).Target(FlagByID(flagID)),
		CommandByID(commandID).NoticeSuppressionAt(0).Match(match),
	}
	for _, expectation := range expectations {
		if got := schemaExpectationOutcome(t, expectation, &schema); got.status != evaluationProved {
			t.Fatalf("%T = %#v", expectation, got)
		}
	}
}
