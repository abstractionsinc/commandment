package cli

import (
	"cmp"
	"context"
	"errors"
	"slices"
)

var (
	errRuleEvaluation     = errors.New("rule evaluation failed")
	errInvalidRuleResult  = errors.New("invalid rule result")
	errAdvisoryEvaluation = errors.New("advisory evaluation failed")
)

type ruleOutcomeKind uint8

const (
	ruleOutcomeInvalid ruleOutcomeKind = iota
	ruleOutcomeSatisfied
	ruleOutcomeViolated
	ruleOutcomeEvaluationFailed
	ruleOutcomeInvalidResult
	ruleOutcomeUpstreamBlocked
)

type violation struct {
	Kind    ruleViolationKind
	Origins []InputOriginID
}

func (violation violation) clone() violation {
	violation.Origins = append([]InputOriginID(nil), violation.Origins...)
	return violation
}

type ruleOutcome struct {
	Kind       ruleOutcomeKind
	Violations []violation
	Err        error
}

func satisfiedRule() ruleOutcome {
	outcome := ruleOutcome{
		Kind: ruleOutcomeSatisfied,
	}
	return outcome
}

func violatedRule(violations ...violation) ruleOutcome {
	copied := make([]violation, len(violations))
	for i, violation := range violations {
		copied[i] = violation.clone()
	}
	outcome := ruleOutcome{
		Kind:       ruleOutcomeViolated,
		Violations: copied,
	}
	return outcome
}

func upstreamBlockedRule() ruleOutcome {
	outcome := ruleOutcome{
		Kind: ruleOutcomeUpstreamBlocked,
	}
	return outcome
}

func ruleDiagnosticCount(outcome ruleOutcome) int {
	switch outcome.Kind {
	case ruleOutcomeSatisfied, ruleOutcomeUpstreamBlocked:
		return 0
	case ruleOutcomeViolated:
		return len(outcome.Violations)
	case ruleOutcomeEvaluationFailed, ruleOutcomeInvalidResult:
		return 1
	default:
		panic("invocation: invalid Rule outcome")
	}
}

type ruleEvaluation struct {
	ID         RuleID
	Activation conditionOutcome
	Outcome    ruleOutcome
	Skipped    bool
}

type analysisAdvisoryEvaluationOutcome uint8

const (
	advisoryEvaluationUnknown analysisAdvisoryEvaluationOutcome = iota
	advisoryEvaluationNotMatched
	advisoryEvaluationMatched
	advisoryEvaluationBlocked
	advisoryEvaluationFailed
	advisoryEvaluationUnsupported
)

type advisoryEvaluation struct {
	ID         advisoryID
	Activation conditionOutcome
	Status     analysisAdvisoryEvaluationOutcome
	Err        error
}

type noticeSuppressionKey struct {
	Command  CommandID
	Position declarationPosition
}

type noticeSuppressionEvaluation struct {
	Key       noticeSuppressionKey
	Condition conditionOutcome
	Matched   bool
}

type advisoryNotice struct {
	AdvisoryID   advisoryID
	AdvisoryCode string
	Severity     uint8
	Message      string
	Reason       string
	Hint         string
	SuppressedBy []noticeSuppressionKey
}

type policyAnalysis struct {
	Rules                map[RuleID]ruleEvaluation
	Advisories           map[advisoryID]advisoryEvaluation
	Suppressions         map[noticeSuppressionKey]noticeSuppressionEvaluation
	Notices              []advisoryNotice
	RuleOrder            []RuleID
	AdvisoryOrder        []advisoryID
	RuleDiagnosticCounts map[RuleID]int
	AdvisoryDiagnostics  map[advisoryID]struct{}
	Blocked              bool
	Truncated            bool
	Unsupported          bool
}

// analysisPolicyContextProjector prepares the phase-authorized analysis context for one
// callback from the exact destinations its declaration may observe.
type analysisPolicyContextProjector func(context.Context, []ValueID) context.Context

// analyzePolicy evaluates the operation-effective policy
// stream after grammar-profile selection has committed. Common declarations and
// declarations in the selected branch participate; inactive branch declarations
// are not evaluated and cannot contribute diagnostics or notices.
func analyzePolicy(ctx context.Context, operation *invocation, selection commandSelection, selectedProfiles map[CommandID]GrammarProfileID, values valueAnalysis, projector analysisPolicyContextProjector) policyAnalysis {
	if operation == nil {
		panic("invocation: nil policy operation")
	}
	result := policyAnalysis{}
	if values.Unsupported || valueAnalysisHasBlockingIssue(values) {
		result.Unsupported = values.Unsupported
		result.Blocked = valueAnalysisHasBlockingIssue(values)
		return result
	}
	// Selected-path traversal establishes the observable command order. Each
	// detached command-scoped family can be position-sorted in place, avoiding a
	// second selected-record copy while preserving declaration order.

	diagnostics := 0
	limit := operation.DiagnosticLimit()
	// Primary cancellation is a semantic frontier before each dependent policy
	// evaluation. Already-recorded results remain authoritative; no later
	// application callback or suppression condition starts after cancellation.
	for _, command := range selection.Path {
		rules := operation.analysis.effectiveModel.RulesForCommand(command)
		slices.SortStableFunc(rules, func(left, right ruleRecord) int { return cmp.Compare(left.meta.position, right.meta.position) })
		for _, record := range rules {
			if ctx.Err() != nil {
				return result
			}
			if !analysisGrammarProfileApplies(selectedProfiles, record.command, record.meta.profile) {
				continue
			}
			evaluation := evaluateRuleRecord(ctx, operation, record, values, projector)
			if result.Rules == nil {
				result.Rules = make(map[RuleID]ruleEvaluation)
			}
			result.Rules[record.meta.id] = evaluation
			result.RuleOrder = append(result.RuleOrder, record.meta.id)
			if evaluation.Outcome.Kind == ruleOutcomeInvalid {
				result.Unsupported = true
				continue
			}
			needed := ruleDiagnosticCount(evaluation.Outcome)
			committed := needed
			if limit >= 0 && diagnostics+needed > limit {
				committed = max(limit-diagnostics, 0)
				result.Truncated = true
			}
			diagnostics += committed
			if committed != 0 {
				if result.RuleDiagnosticCounts == nil {
					result.RuleDiagnosticCounts = make(map[RuleID]int)
				}
				result.RuleDiagnosticCounts[record.meta.id] = committed
			}
			switch evaluation.Outcome.Kind {
			case ruleOutcomeViolated, ruleOutcomeEvaluationFailed, ruleOutcomeInvalidResult:
				result.Blocked = true
			case ruleOutcomeInvalid, ruleOutcomeSatisfied, ruleOutcomeUpstreamBlocked:
			}
			if result.Truncated {
				result.Blocked = true
				break
			}
		}
		if result.Blocked {
			break
		}
	}
	if result.Blocked || result.Unsupported || ctx.Err() != nil {
		return result
	}

	for _, command := range selection.Path {
		advisories := operation.analysis.effectiveModel.AdvisoriesForCommand(command)
		slices.SortStableFunc(advisories, func(left, right advisoryRecord) int { return cmp.Compare(left.meta.position, right.meta.position) })
		for _, record := range advisories {
			if ctx.Err() != nil {
				return result
			}
			if !analysisGrammarProfileApplies(selectedProfiles, record.command, record.meta.profile) {
				continue
			}
			evaluation, notice := evaluateAdvisoryRecord(ctx, operation, record, values, projector)
			if result.Advisories == nil {
				result.Advisories = make(map[advisoryID]advisoryEvaluation)
			}
			result.Advisories[record.meta.id] = evaluation
			result.AdvisoryOrder = append(result.AdvisoryOrder, record.meta.id)
			if evaluation.Status == advisoryEvaluationUnsupported {
				result.Unsupported = true
				continue
			}
			if evaluation.Status == advisoryEvaluationFailed {
				if limit >= 0 && diagnostics+1 > limit {
					result.Truncated = true
					result.Blocked = true
					return result
				}
				if result.AdvisoryDiagnostics == nil {
					result.AdvisoryDiagnostics = make(map[advisoryID]struct{})
				}
				result.AdvisoryDiagnostics[record.meta.id] = struct{}{}
				result.Blocked = true
				return result
			}
			if notice != nil {
				result.Notices = append(result.Notices, *notice)
			}
		}
	}
	if result.Unsupported || ctx.Err() != nil {
		return result
	}

	suppressions := operation.analysis.effectiveModel.NoticeSuppressionsForCommandsOrdered(selection.Path)
	for _, record := range suppressions {
		if ctx.Err() != nil {
			return result
		}
		if !analysisGrammarProfileApplies(selectedProfiles, record.command, record.profile) {
			continue
		}
		key := noticeSuppressionKey{
			Command:  record.command,
			Position: record.position,
		}
		outcome := evaluateCondition(record.condition, values)
		entry := noticeSuppressionEvaluation{
			Key:       key,
			Condition: outcome,
		}
		if result.Suppressions == nil {
			result.Suppressions = make(map[noticeSuppressionKey]noticeSuppressionEvaluation)
		}
		if outcome == conditionOutcomeUnsupported {
			result.Unsupported = true
			result.Suppressions[key] = entry
			continue
		}
		if outcome == conditionOutcomeSatisfied {
			entry.Matched = true
			for index := range result.Notices {
				if advisoryNoticeMatches(result.Notices[index], record.match) {
					result.Notices[index].SuppressedBy = append(result.Notices[index].SuppressedBy, key)
				}
			}
		}
		result.Suppressions[key] = entry
	}
	return result
}

func valueAnalysisHasBlockingIssue(values valueAnalysis) bool {
	for _, issue := range values.Issues {
		if issue.Kind != valueIssueUnsupported && issue.Kind != valueIssueProducerBlocked {
			return true
		}
	}
	return false
}

func evaluateRuleRecord(ctx context.Context, operation *invocation, record ruleRecord, values valueAnalysis, projector analysisPolicyContextProjector) ruleEvaluation {
	definition := record.definition
	evaluation := ruleEvaluation{
		ID:         record.meta.id,
		Activation: conditionOutcomeSatisfied,
	}
	if activation, ok := definition.Activation(); ok {
		evaluation.Activation = evaluatePolicyCondition(activation, values)
		switch evaluation.Activation {
		case conditionOutcomeUnsatisfied:
			evaluation.Skipped = true
			evaluation.Outcome = satisfiedRule()
			return evaluation
		case conditionOutcomeBlocked:
			evaluation.Skipped = true
			evaluation.Outcome = upstreamBlockedRule()
			return evaluation
		case conditionOutcomeUnsupported:
			return evaluation
		case conditionOutcomeUnknown, conditionOutcomeSatisfied:
		}
	}
	if definition.Kind() != RuleCustom {
		evaluation.Outcome = evaluateBuiltinRule(definition, values)
		return evaluation
	}
	if projector == nil {
		return evaluation
	}
	if blockedByRequiredDependency(definition.Dependencies(), values) {
		evaluation.Outcome = upstreamBlockedRule()
		return evaluation
	}
	configuration, ok := operation.analysisConfiguration(record.command, values.Destinations)
	if !ok {
		return evaluation
	}
	authorized := policyAuthorizedDestinations(definition.Subjects(), definition.Dependencies())
	dependencies := dependenciesFromAuthorizedValueAnalysis(operation.scope, operation.analysis.effectiveModel, authorized, values)
	callbackContext := projector(ctx, authorized)
	switch definition.CallbackKind() {
	case ruleCallbackCheck:
		result, err := definition.Evaluate(callbackContext, configuration, dependencies)
		if err != nil {
			evaluation.Outcome = ruleOutcome{
				Kind: ruleOutcomeEvaluationFailed,
				Err:  errors.Join(errRuleEvaluation, err),
			}
			return evaluation
		}
		if len(result.Violations) != 0 {
			evaluation.Outcome = ruleOutcome{
				Kind: ruleOutcomeInvalidResult,
				Err:  errInvalidRuleResult,
			}
			return evaluation
		}
		if result.Check {
			evaluation.Outcome = satisfiedRule()
		} else {
			evaluation.Outcome = violatedRule(violation{
				Kind: ruleViolationInvalid,
			})
		}
		return evaluation
	case ruleCallbackValidate:
		result, err := definition.Evaluate(callbackContext, configuration, dependencies)
		if err != nil {
			evaluation.Outcome = ruleOutcome{
				Kind: ruleOutcomeEvaluationFailed,
				Err:  errors.Join(errRuleEvaluation, err),
			}
			return evaluation
		}
		violations, ok := projectRuleViolations(operation, definition.Subjects(), values, result.Violations)
		if !ok {
			evaluation.Outcome = ruleOutcome{
				Kind: ruleOutcomeInvalidResult,
				Err:  errInvalidRuleResult,
			}
			return evaluation
		}
		if len(violations) == 0 {
			evaluation.Outcome = satisfiedRule()
		} else {
			evaluation.Outcome = violatedRule(violations...)
		}
		return evaluation
	default:
		return evaluation
	}
}

func evaluateBuiltinRule(definition ruleDefinition, values valueAnalysis) ruleOutcome {
	subjects := definition.Subjects()
	states := make([]destinationResolution, len(subjects))
	for i, destination := range subjects {
		resolution, ok := values.Destinations[destination]
		if !ok {
			return ruleOutcome{}
		}
		states[i] = resolution
	}
	kind := definition.Kind()
	if kind == RuleRequire || kind == RuleForbid {
		if len(states) != 1 {
			return ruleOutcome{}
		}
		switch states[0].State {
		case destinationStateFailed, destinationStateBlocked:
			return upstreamBlockedRule()
		case destinationStateResolved:
			if kind == RuleRequire {
				return satisfiedRule()
			}
			ruleViolation := violation{
				Kind: ruleViolationInvalid,
			}
			return violatedRule(ruleViolation)
		case destinationStateAbsent, destinationStateInactive:
			if kind == RuleForbid {
				return satisfiedRule()
			}
			ruleViolation := violation{
				Kind: ruleViolationInvalid,
			}
			return violatedRule(ruleViolation)
		default:
			return ruleOutcome{}
		}
	}
	if isPresenceGroupRule(kind) {
		present := 0
		for _, state := range states {
			switch state.State {
			case destinationStateResolved:
				present++
			case destinationStateAbsent, destinationStateInactive:
			case destinationStateFailed, destinationStateBlocked:
				return upstreamBlockedRule()
			default:
				return ruleOutcome{}
			}
		}
		matched := false
		switch kind {
		case RuleExactlyOne:
			matched = present == 1
		case RuleAllOrNone:
			matched = present == 0 || present == len(states)
		case RuleAtMostOne:
			matched = present <= 1
		case RuleAtLeastOne:
			matched = present >= 1
		case RuleUnknown,
			RuleCustom,
			RuleRequire,
			RuleForbid,
			RuleNotEmpty,
			RuleIn,
			RuleNotIn,
			RuleEqual,
			RuleNotEqual,
			RuleBetween,
			RuleAtLeast,
			RuleAtMost,
			RuleGreaterThan,
			RuleLessThan,
			RuleMatch,
			RuleItemCount,
			RuleContains,
			RuleEachIn,
			RuleUnique,
			RuleEntryCount,
			RuleHasKeys,
			RuleKeysIn,
			RuleSubsetOf,
			RuleDisjoint,
			RuleKeysSubsetOf,
			RuleKeysDisjoint:
		}
		if matched {
			return satisfiedRule()
		}
		ruleViolation := violation{
			Kind: ruleViolationConflict,
		}
		return violatedRule(ruleViolation)
	}
	for _, state := range states {
		switch state.State {
		case destinationStateFailed, destinationStateBlocked:
			return upstreamBlockedRule()
		case destinationStateAbsent, destinationStateInactive:
			return satisfiedRule()
		case destinationStateResolved:
		default:
			return ruleOutcome{}
		}
	}
	if kind == RuleItemCount || kind == RuleEntryCount {
		if len(states) != 1 {
			return ruleOutcome{}
		}
		count := len(states[0].Elements)
		if kind == RuleEntryCount {
			count = len(states[0].Entries)
		}
		rangeValue := definition.Count()
		maximum, bounded := rangeValue.Maximum()
		if count >= rangeValue.Minimum() && (!bounded || count <= maximum) {
			return satisfiedRule()
		}
		ruleViolation := violation{
			Kind: ruleViolationInvalid,
		}
		return violatedRule(ruleViolation)
	}
	predicate, ok := definition.BuiltinPredicate()
	if !ok {
		return ruleOutcome{}
	}
	raw := make([]any, len(states))
	for i := range states {
		raw[i] = states[i].Value
	}
	if predicate(raw) {
		return satisfiedRule()
	}
	ruleViolation := violation{
		Kind: ruleViolationInvalid,
	}
	return violatedRule(ruleViolation)
}

func isPresenceGroupRule(kind RuleKind) bool {
	switch kind {
	case RuleExactlyOne,
		RuleAllOrNone,
		RuleAtMostOne,
		RuleAtLeastOne:
		return true
	default:
		return false
	}
}

func evaluateAdvisoryRecord(ctx context.Context, operation *invocation, record advisoryRecord, values valueAnalysis, projector analysisPolicyContextProjector) (advisoryEvaluation, *advisoryNotice) {
	definition := record.definition
	evaluation := advisoryEvaluation{
		ID:         record.meta.id,
		Activation: conditionOutcomeSatisfied,
	}
	var matched bool
	switch definition.Kind() {
	case AdvisoryKindAlways:
		matched = true
	case AdvisoryKindConditional:
		condition, ok := definition.Activation()
		if !ok {
			evaluation.Status = advisoryEvaluationUnsupported
			return evaluation, nil
		}
		evaluation.Activation = evaluateCondition(condition, values)
		switch evaluation.Activation {
		case conditionOutcomeSatisfied:
			matched = true
		case conditionOutcomeUnsatisfied:
			evaluation.Status = advisoryEvaluationNotMatched
			return evaluation, nil
		case conditionOutcomeBlocked:
			evaluation.Status = advisoryEvaluationBlocked
			return evaluation, nil
		default:
			evaluation.Status = advisoryEvaluationUnsupported
			return evaluation, nil
		}
	case AdvisoryKindDetector:
		if projector == nil {
			evaluation.Status = advisoryEvaluationUnsupported
			return evaluation, nil
		}
		if blockedByRequiredDependency(definition.Dependencies(), values) {
			evaluation.Status = advisoryEvaluationBlocked
			return evaluation, nil
		}
		configuration, ok := operation.analysisConfiguration(record.command, values.Destinations)
		if !ok {
			evaluation.Status = advisoryEvaluationUnsupported
			return evaluation, nil
		}
		authorized := policyAuthorizedDestinations(definition.Subjects(), definition.Dependencies())
		dependencies := dependenciesFromAuthorizedValueAnalysis(operation.scope, operation.analysis.effectiveModel, authorized, values)
		var err error
		matched, err = definition.Evaluate(projector(ctx, authorized), configuration, dependencies)
		if err != nil {
			evaluation.Status = advisoryEvaluationFailed
			evaluation.Err = errors.Join(errAdvisoryEvaluation, err)
			return evaluation, nil
		}
	default:
		evaluation.Status = advisoryEvaluationUnsupported
		return evaluation, nil
	}
	if !matched {
		evaluation.Status = advisoryEvaluationNotMatched
		return evaluation, nil
	}
	evaluation.Status = advisoryEvaluationMatched
	notice := &advisoryNotice{
		AdvisoryID:   record.meta.id,
		AdvisoryCode: definition.Code(),
		Severity:     definition.Severity(),
		Message:      definition.Message(),
		Reason:       definition.Reason(),
		Hint:         definition.Hint(),
	}
	return evaluation, notice
}

func blockedByRequiredDependency(requirements []dependencyRequirement, values valueAnalysis) bool {
	for _, requirement := range requirements {
		resolution, ok := values.Destinations[requirement.Destination()]
		if !ok || resolution.State != destinationStateResolved {
			if !requirement.Optional() {
				return true
			}
		}
	}
	return false
}

func policyAuthorizedDestinations(subjects []ValueID, dependencies []dependencyRequirement) []ValueID {
	result := make([]ValueID, 0, len(subjects)+len(dependencies))
	seen := make(map[ValueID]struct{}, len(subjects)+len(dependencies))
	appendID := func(id ValueID) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	for _, id := range subjects {
		appendID(id)
	}
	for _, dependency := range dependencies {
		appendID(dependency.Destination())
	}
	return result
}

func projectRuleViolations(operation *invocation, subjects []ValueID, values valueAnalysis, source []ruleViolation) ([]violation, bool) {
	allowed := make(map[uint32]InputOriginID)
	for _, subject := range subjects {
		resolution, ok := values.Destinations[subject]
		if !ok {
			continue
		}
		for _, candidateID := range resolution.Candidates {
			candidate, ok := values.Candidates[candidateID]
			if !ok {
				continue
			}
			collectAuthorizedOrigins(allowed, candidate.Meta.Origins)
			for _, component := range candidate.Elements {
				collectAuthorizedOrigins(allowed, component.Origins)
			}
			for _, entry := range candidate.Entries {
				collectAuthorizedOrigins(allowed, entry.Key.Origins)
				collectAuthorizedOrigins(allowed, entry.Value.Origins)
			}
		}
		for _, producer := range resolution.Producers {
			collectAuthorizedOrigins(allowed, producer.Origins)
		}
	}

	result := make([]violation, len(source))
	for index, item := range source {
		var kind ruleViolationKind
		switch item.Kind {
		case ruleViolationInvalid:
			kind = ruleViolationInvalid
		case ruleViolationConflict:
			kind = ruleViolationConflict
		case ruleViolationDuplicate:
			kind = ruleViolationDuplicate
		default:
			return nil, false
		}
		if kind == ruleViolationDuplicate && len(item.Origins) != 2 {
			return nil, false
		}
		origins := make([]InputOriginID, len(item.Origins))
		seen := make(map[InputOriginID]struct{}, len(item.Origins))
		for i, raw := range item.Origins {
			origin, ok := allowed[raw]
			if !ok {
				return nil, false
			}
			if _, duplicate := seen[origin]; duplicate {
				return nil, false
			}
			if _, ok := operation.Origin(origin); !ok {
				return nil, false
			}
			seen[origin] = struct{}{}
			origins[i] = origin
		}
		result[index] = violation{
			Kind:    kind,
			Origins: origins,
		}
	}
	return result, true
}

func collectAuthorizedOrigins(target map[uint32]InputOriginID, origins []InputOriginID) {
	for _, origin := range origins {
		if origin.present() {
			target[origin.uint32()] = origin
		}
	}
}

func advisoryNoticeMatches(notice advisoryNotice, match noticeMatch) bool {
	if match.All {
		return true
	}
	if match.Severity != 0 && notice.Severity != match.Severity {
		return false
	}
	if match.Code != "" && match.Code != "advisory" {
		return false
	}
	if match.AdvisoryCode != "" && notice.AdvisoryCode != match.AdvisoryCode {
		return false
	}
	return true
}
