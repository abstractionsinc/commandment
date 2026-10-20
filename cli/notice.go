package cli

import (
	"context"
	"slices"
)

type noticeCapability[C any] struct {
	session *declarationSession
	profile grammarProfileCaseRef
}

// SuppressWhen declares Notice suppression evaluated from finalized invocation semantics when
// condition and match both apply; it does not reread application-mutated execution configuration.
func (noticeCapability noticeCapability[C]) SuppressWhen(condition Condition, match NoticeMatch) {
	stageNoticeSuppression(noticeCapability.session, noticeCapability.profile, condition, match)
}

func stageNoticeSuppression(session *declarationSession, profile grammarProfileCaseRef, condition Condition, match NoticeMatch) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		suppression := noticeSuppressionDraft{
			condition: condition,
			match:     match,
			valid:     condition.state != nil && condition.state.valid && condition.state.owner == draft.identity && validNoticeMatch(match),
		}
		record := noticeSuppressionDraft{
			position:  draft.nextPosition(),
			command:   command,
			profile:   profile,
			condition: suppression.condition,
			match:     suppression.match,
			valid:     suppression.valid,
		}
		draft.noticeSuppressions.append(draft, record)
	})
}

func validNoticeMatch(match NoticeMatch) bool {
	if match.Severity != NoticeSeverityUnknown && !match.Severity.valid() {
		return false
	}
	noticeMatchDeclarationValue := noticeMatchDeclaration{
		All:           match.All,
		Severity:      match.Severity,
		FrameworkCode: string(match.Code),
		AdvisoryCode:  string(match.AdvisoryCode),
	}
	return validateNoticeMatch(noticeMatchDeclarationValue)
}

type noticeMatch struct {
	All          bool
	Severity     uint8
	Code         string
	AdvisoryCode string
}

func (match noticeMatch) Valid() bool {
	if NoticeSeverity(match.Severity) > NoticeSeverityWarning {
		return false
	}
	narrow := match.Severity != 0 || match.Code != "" || match.AdvisoryCode != ""
	if match.All {
		return !narrow
	}
	return narrow
}

type noticeSuppressionRecord struct {
	command   CommandID
	profile   GrammarProfileID
	position  declarationPosition
	match     noticeMatch
	condition condition
}

func acceptNoticeSuppressionRecord(command CommandID, position declarationPosition, match noticeMatch, condition condition) noticeSuppressionRecord {
	if !command.present() || position == 0 || !match.Valid() || !condition.Valid() {
		panic("semantic: invalid Notice suppression record")
	}
	record := noticeSuppressionRecord{
		command:   command,
		position:  position,
		match:     match,
		condition: condition,
	}

	return record
}

type lifecycleNoticeCandidate struct {
	notice Notice
}

// projectLifecycleNoticeInspections projects declaration lifecycle warnings from the
// exact declarations exercised by this invocation. It deliberately reuses the
// existing Notice suppression contract; lifecycle state remains owned by the
// declaration and this function only projects an operation-local observation.
func projectLifecycleNoticeInspections(model *effectiveModel, grammar grammarResult, values valueAnalysis, policy policyAnalysis) []NoticeInspection {
	candidates := collectLifecycleNoticeCandidates(model, grammar, values)
	result := make([]NoticeInspection, 0, len(candidates))
	for _, candidate := range candidates {
		item := NoticeInspection{
			Notice:      cloneNotice(candidate.notice),
			Suppression: NoticeSuppressionNotSuppressed,
		}
		for _, suppression := range lifecycleNoticeSuppressions(model, policy, candidate.notice) {
			item.Suppression = NoticeSuppressionSuppressed
			item.SuppressedBy = append(item.SuppressedBy, suppression)
		}
		result = append(result, item)
	}
	return result
}

func buildLifecycleNoticeDeliveryAttempts(model *effectiveModel, grammar grammarResult, values valueAnalysis, policy policyAnalysis) []noticeDeliveryAttempt {
	inspections := projectLifecycleNoticeInspections(model, grammar, values, policy)
	result := make([]noticeDeliveryAttempt, 0, len(inspections))
	for index, item := range inspections {
		if item.Suppression == NoticeSuppressionSuppressed {
			continue
		}
		noticeDeliveryAttemptValue := noticeDeliveryAttempt{
			Index:  index,
			Phase:  NoticePhaseAnalysis,
			Notice: cloneNotice(item.Notice),
		}
		result = append(result, noticeDeliveryAttemptValue)
	}
	return result
}

func collectLifecycleNoticeCandidates(model *effectiveModel, grammar grammarResult, values valueAnalysis) []lifecycleNoticeCandidate {
	records, effectiveGrammar, available := model.schemaSnapshot()
	if !available {
		panic("cli: lifecycle Notice projection unavailable")
	}
	source, err := prepareStaticArtifactSource(
		model.Valid(),
		model.Application(),
		records,
		effectiveGrammar,
		analyzeRecordReachability(records),
	)
	if err != nil {
		panic("cli: lifecycle Notice projection invalid")
	}
	var seen map[lifecycleNoticeKey]struct{}
	var result []lifecycleNoticeCandidate

	for _, occurrence := range grammar.Occurrences {
		spelling, _ := model.FlagSpelling(occurrence.Spelling)
		state, _ := evaluateLifecycle(model.Application().release, spelling.lifecycle, "")
		if state != LifecycleStateDeprecated {
			continue
		}
		flag, _ := model.Flag(spelling.flag)
		id := spelling.meta.id
		key := lifecycleNoticeKey{
			kind: DeclarationKindFlagSpelling,
			id:   uint32(id),
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		if seen == nil {
			seen = make(map[lifecycleNoticeKey]struct{})
		}
		seen[key] = struct{}{}
		spellingGrammar, ok := source.spellingGrammar[id]
		if !ok {
			panic("cli: lifecycle Notice spelling projection missing")
		}
		replacement, _, err := source.genericReplacement(flagSpellingReplacementIdentity(id))
		if err != nil {
			panic("cli: lifecycle Notice replacement projection invalid")
		}
		command := flag.command
		result = append(result, lifecycleNoticeCandidate{
			notice: Notice{
				Code:        NoticeDeprecatedFlag,
				Severity:    NoticeSeverityWarning,
				Reason:      spelling.lifecycle.Reason,
				Subject:     spellingGrammar.text,
				Command:     command,
				CommandPath: commandPathNames(model, command),
				Declaration: DeclarationIdentity{
					Kind:         DeclarationKindFlagSpelling,
					FlagSpelling: id,
				},
				Replacement: cloneReplacement(replacement),
			},
		})
	}

	for destination, resolution := range values.Destinations {
		if resolution.State != destinationStateResolved {
			continue
		}
		record, _ := model.Destination(destination)
		command := record.command
		for _, candidateID := range resolution.Contributors {
			candidate := values.Candidates[candidateID]
			entries := acceptedDeprecatedDomainEntries(model, destination, candidate)
			if len(entries) == 0 {
				continue
			}
			for _, entry := range entries {
				id := entry.meta.id
				key := lifecycleNoticeKey{
					kind: DeclarationKindValueDomainEntry,
					id:   uint32(id),
				}
				if _, duplicate := seen[key]; duplicate {
					continue
				}
				if seen == nil {
					seen = make(map[lifecycleNoticeKey]struct{})
				}
				seen[key] = struct{}{}
				replacement, present, err := source.domainEntryReplacement(entry)
				if err != nil {
					panic("cli: lifecycle Notice domain replacement projection invalid")
				}
				if !present {
					replacement = nil
				}
				sensitive := record.sensitive || candidate.Meta.Sensitive
				subject := entry.name
				if sensitive {
					subject = ""
				}
				result = append(result, lifecycleNoticeCandidate{
					notice: Notice{
						Code:        NoticeDeprecatedValue,
						Severity:    NoticeSeverityWarning,
						Reason:      entry.lifecycle.Reason,
						Subject:     subject,
						Command:     command,
						CommandPath: commandPathNames(model, command),
						Declaration: DeclarationIdentity{
							Kind:             DeclarationKindValueDomainEntry,
							ValueDomainEntry: id,
						},
						Replacement: cloneReplacement(replacement),
						Sensitive:   sensitive,
					},
				})
			}
		}
	}
	return result
}

type lifecycleNoticeKey struct {
	kind DeclarationKind
	id   uint32
}

func acceptedDeprecatedDomainEntries(model *effectiveModel, destination ValueID, candidate valueCandidateObservation) []domainEntryRecord {
	seen := make(map[ValueDomainEntryID]struct{})
	result := make([]domainEntryRecord, 0)
	for _, index := range model.ValueDomainIndexesForDestination(destination) {
		domain, ok := model.ValueDomain(index)
		if !ok {
			continue
		}

		for _, use := range domain.Uses() {
			if use.destination != destination {
				continue
			}
			inputs := acceptedDomainInputs(candidate, use.kind)
			if len(inputs) == 0 {
				continue
			}
			for _, id := range domain.Entries() {
				entry, _ := model.DomainEntry(id)
				if !slices.Contains(inputs, entry.name) {
					continue
				}
				state, _ := evaluateLifecycle(model.Application().release, entry.lifecycle, "")
				if state != LifecycleStateDeprecated {
					continue
				}
				if _, duplicate := seen[entry.meta.id]; duplicate {
					continue
				}
				seen[entry.meta.id] = struct{}{}
				result = append(result, entry)
			}
		}
	}
	return result
}

func acceptedDomainInputs(candidate valueCandidateObservation, kind ValueDomainUseKind) []string {
	var result []string
	switch kind {
	case ValueDomainUseDestination:
		if candidate.HasInput && candidate.HasValue {
			result = append(result, candidate.Input)
		}
	case ValueDomainUseSequenceElement:
		for _, element := range candidate.Elements {
			if element.HasInput && element.HasValue {
				result = append(result, element.Input)
			}
		}
	case ValueDomainUseMapKey:
		for _, entry := range candidate.Entries {
			if entry.Key.HasInput && entry.Key.HasValue {
				result = append(result, entry.Key.Input)
			}
		}
	case ValueDomainUseMapValue:
		for _, entry := range candidate.Entries {
			if entry.Value.HasInput && entry.Value.HasValue {
				result = append(result, entry.Value.Input)
			}
		}
	case ValueDomainUseUnknown:
	}
	return result
}

func lifecycleNoticeSuppressions(model *effectiveModel, policy policyAnalysis, notice Notice) []NoticeSuppressionLocator {
	var result []NoticeSuppressionLocator
	for _, record := range model.NoticeSuppressions() {
		key := noticeSuppressionKey{
			Command:  record.command,
			Position: record.position,
		}
		runtime, ok := policy.Suppressions[key]
		if !ok || !runtime.Matched || !lifecycleNoticeMatches(notice, record.match) {
			continue
		}
		result = append(result, NoticeSuppressionLocator{
			Command:          record.command,
			DeclarationIndex: int(record.position),
		})
	}
	return result
}

func lifecycleNoticeMatches(notice Notice, match noticeMatch) bool {
	if match.All {
		return true
	}
	if match.Severity != 0 && uint8(notice.Severity) != match.Severity {
		return false
	}
	if match.Code != "" && string(notice.Code) != match.Code {
		return false
	}
	if match.AdvisoryCode != "" && string(notice.AdvisoryCode) != match.AdvisoryCode {
		return false
	}
	return true
}

// NoticeHandler synchronously receives one deliverable Notice for an invocation. A
// returned error is recorded as notice-delivery failure but does not stop later
// Notice delivery or the application execution pipeline.
type NoticeHandler func(context.Context, Notice) error

// NoticeCode is the stable machine classification of a normalized nonfatal Notice. Exact
// subjects remain in structured identity fields such as Declaration and AdvisoryCode.
type NoticeCode string

const (
	// NoticeAdvisory classifies a normalized application Advisory; AdvisoryCode retains its application classification.
	NoticeAdvisory NoticeCode = "advisory"
	// NoticeDeprecatedCommand reports invocation of a deprecated command declaration.
	NoticeDeprecatedCommand NoticeCode = "deprecated-command"
	// NoticeDeprecatedFlag reports invocation of a deprecated flag spelling/declaration.
	NoticeDeprecatedFlag NoticeCode = "deprecated-flag"
	// NoticeDeprecatedArgument reports use of a deprecated argument declaration.
	NoticeDeprecatedArgument NoticeCode = "deprecated-argument"
	// NoticeDeprecatedValue reports use of a deprecated finite-domain value or alias.
	NoticeDeprecatedValue NoticeCode = "deprecated-value"
	// NoticeSignalHint carries nonfatal signal-related guidance from the Commandment runtime.
	NoticeSignalHint NoticeCode = "signal-hint"
)

// NoticeSeverity is presentation severity for a Notice; it does not change semantic
// validity, execution eligibility, or delivery ordering.
type NoticeSeverity uint8

func (severity NoticeSeverity) valid() bool {
	return severity == NoticeSeverityInformation || severity == NoticeSeverityWarning
}

const (
	// NoticeSeverityUnknown is the zero value and is not a valid deliverable severity.
	NoticeSeverityUnknown NoticeSeverity = iota
	// NoticeSeverityInformation marks informational presentation.
	NoticeSeverityInformation
	// NoticeSeverityWarning marks warning presentation.
	NoticeSeverityWarning
)

// NoticeMatch selects Notices for declarative delivery suppression. Nonzero criteria
// are conjunctive. The zero value is invalid; All explicitly matches every Notice
// and cannot be combined with narrower criteria.
type NoticeMatch struct {
	All          bool
	Severity     NoticeSeverity
	Code         NoticeCode
	AdvisoryCode AdvisoryCode
}

// Notice is structured nonfatal semantic information produced by analysis or runtime
// lifecycle. Suppression affects delivery only: the Notice remains semantically
// present for applicable artifacts. Sensitive marks presentation that must be
// redacted by library-generated output.
type Notice struct {
	Code         NoticeCode
	AdvisoryCode AdvisoryCode
	Severity     NoticeSeverity

	Message string
	Reason  string
	Hint    string

	Subject     string
	CommandPath []string

	Command     CommandID
	Declaration DeclarationIdentity
	Replacement *DeclarationReplacement

	Sensitive bool
}

// NoticePhase identifies the invocation phase in which Notice delivery was attempted.
type NoticePhase uint8

const (
	// NoticePhaseUnknown is the zero value and does not identify a delivery phase.
	NoticePhaseUnknown NoticePhase = iota
	// NoticePhaseAnalysis identifies Notice delivery associated with invocation analysis.
	NoticePhaseAnalysis
	// NoticePhaseExecution identifies Notice delivery associated with application execution.
	NoticePhaseExecution
	// NoticePhaseCleanup identifies Notice delivery associated with Commandment cleanup.
	NoticePhaseCleanup
)

// NoticeDeliveryFailure identifies one failed delivery attempt by deterministic
// delivery index, phase, code, and command path.
type NoticeDeliveryFailure struct {
	Index       int
	Phase       NoticePhase
	Code        NoticeCode
	CommandPath []string
}
