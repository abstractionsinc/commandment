package cli

import "context"

// AdvisoryDeclaration identifies one staged application Advisory. Subject grants structured
// presentation and provenance authority; DependsOn is valid only for detector dependencies. The
// zero value is invalid; copies identify the same Advisory and are valid only in the owning
// declaration callback.
type AdvisoryDeclaration[C any] struct {
	state    *declarationSession
	advisory advisoryDraftRef
}

// Subject declares that destination is a semantic subject of this Advisory, authorizing
// presentation/provenance and declared-state observation for that destination.
func (declaration AdvisoryDeclaration[C]) Subject[D, T any](destination destinationLike[D, T]) AdvisoryDeclaration[C] {
	stageAdvisorySubjectDirective(declaration.state, declaration.advisory, destinationHandleOf(destination))
	return declaration
}

// DependsOn declares a detector dependency; it is valid only for an Advisory created by Detect.
func (declaration AdvisoryDeclaration[C]) DependsOn(dependency Dependency) AdvisoryDeclaration[C] {
	stageAdvisoryDirective(declaration.state, declaration.advisory, advisoryDirectiveDependency, dependency)
	return declaration
}

type advisoryCapability[C any] struct {
	session *declarationSession
	profile grammarProfileCaseRef
}

// Always declares an Advisory that applies whenever its owning command reaches Advisory evaluation.
func (advisoryCapability advisoryCapability[C]) Always(advisory Advisory) AdvisoryDeclaration[C] {
	return stageAlwaysAdvisory[C](advisoryCapability.session, advisoryCapability.profile, advisory)
}

// When declares an Advisory controlled by the structural condition.
func (advisoryCapability advisoryCapability[C]) When(condition Condition, advisory Advisory) AdvisoryDeclaration[C] {
	valid := advisoryCapability.session != nil && condition.state != nil && condition.state.valid &&
		condition.state.owner != nil
	return stageConditionalAdvisory[C](advisoryCapability.session, advisoryCapability.profile, condition, advisory, valid)
}

// Detect declares an Advisory whose read-only analysis callback decides applicability; true, nil
// matches and any non-nil error is Advisory evaluation failure.
func (advisoryCapability advisoryCapability[C]) Detect(predicate AdvisoryPredicate[C], advisory Advisory) AdvisoryDeclaration[C] {
	callback := eraseAdvisoryPredicate(predicate)
	return stageDetectorAdvisory[C](advisoryCapability.session, advisoryCapability.profile, advisory, callback)
}

func stageAdvisory[C any](session *declarationSession, profile grammarProfileCaseRef, value advisoryDraft) AdvisoryDeclaration[C] {
	if session == nil {
		return AdvisoryDeclaration[C]{}
	}

	var ref advisoryDraftRef
	session.use(func(draft *declarationDraft, command commandRef) {
		if value.condition.state != nil {
			value.valid = value.valid && value.condition.state.valid && value.condition.state.owner == draft.identity
		}
		record := advisoryDraft{
			position:  draft.nextPosition(),
			command:   command,
			profile:   profile,
			kind:      value.kind,
			advisory:  value.advisory,
			condition: value.condition,
			callback:  value.callback,
			valid:     value.valid,
		}
		ref = draft.advisories.append(draft, record)
	})
	return AdvisoryDeclaration[C]{
		state:    session,
		advisory: ref,
	}
}

func stageAlwaysAdvisory[C any](session *declarationSession, profile grammarProfileCaseRef, advisory Advisory) AdvisoryDeclaration[C] {
	value := advisoryDraft{
		kind:     AdvisoryKindAlways,
		advisory: advisory,
		valid:    true,
	}

	return stageAdvisory[C](session, profile, value)
}

func stageConditionalAdvisory[C any](session *declarationSession, profile grammarProfileCaseRef, condition Condition, advisory Advisory, valid bool) AdvisoryDeclaration[C] {
	value := advisoryDraft{
		kind:      AdvisoryKindConditional,
		advisory:  advisory,
		condition: condition,
		valid:     valid,
	}

	return stageAdvisory[C](session, profile, value)
}

func stageDetectorAdvisory[C any](session *declarationSession, profile grammarProfileCaseRef, advisory Advisory, callback authoringAdvisoryCallback) AdvisoryDeclaration[C] {
	value := advisoryDraft{
		kind:     AdvisoryKindDetector,
		advisory: advisory,
		callback: callback,
		valid:    callback != nil,
	}

	return stageAdvisory[C](session, profile, value)
}

func eraseAdvisoryPredicate[C any](predicate AdvisoryPredicate[C]) authoringAdvisoryCallback {
	if predicate == nil {
		return nil
	}
	return func(ctx context.Context, raw any, dependencies Dependencies) (bool, error) {
		config, ok := raw.(*C)
		if !ok || config == nil {
			panic("cli: invalid advisory callback configuration")
		}
		return predicate(ctx, config, dependencies)
	}
}

func stageAdvisorySubjectDirective(session *declarationSession, advisory advisoryDraftRef, destination *destinationHandle) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		owner, ownerOK := draft.advisories.resolve(draft, advisory)
		directive := advisoryDirectiveDraft{
			advisory: advisory,
			kind:     advisoryDirectiveSubject,
		}
		if ownerOK && destination != nil {
			directive.destination, directive.valid = captureAuthoringDestinationIdentity(draft, destination)
		}
		profile := grammarProfileCaseRef{}
		if ownerOK {
			profile = owner.profile
		}
		record := advisoryDirectiveDraft{
			position:    draft.nextPosition(),
			command:     command,
			profile:     profile,
			advisory:    advisory,
			kind:        directive.kind,
			destination: directive.destination,
			optional:    directive.optional,
			valid:       directive.valid,
		}
		draft.advisoryDirectives.append(draft, record)
	})
}

func stageAdvisoryDirective(session *declarationSession, advisory advisoryDraftRef, kind advisoryDirectiveKind, dependency Dependency) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		owner, ownerOK := draft.advisories.resolve(draft, advisory)
		directive := advisoryDirectiveDraft{
			advisory: advisory,
			kind:     kind,
		}
		if ownerOK {
			destination, optional := dependencyTarget(dependency)
			directive.optional = optional
			directive.destination, directive.valid = captureAuthoringDestinationIdentity(draft, destination)
		}
		profile := grammarProfileCaseRef{}
		if ownerOK {
			profile = owner.profile
		}
		record := advisoryDirectiveDraft{
			position:    draft.nextPosition(),
			command:     command,
			profile:     profile,
			advisory:    advisory,
			kind:        directive.kind,
			destination: directive.destination,
			optional:    directive.optional,
			valid:       directive.valid,
		}
		draft.advisoryDirectives.append(draft, record)
	})
}

type authoringAdvisoryCallback func(context.Context, any, Dependencies) (bool, error)

type advisoryDirectiveKind uint8

const (
	advisoryDirectiveInvalid advisoryDirectiveKind = iota
	advisoryDirectiveSubject
	advisoryDirectiveDependency
)

type advisoryCallback func(context.Context, any, Dependencies) (bool, error)

type advisoryDefinition struct {
	kind         AdvisoryKind
	code         string
	severity     uint8
	message      string
	reason       string
	hint         string
	activation   *condition
	subjects     []ValueID
	dependencies []dependencyRequirement
	callback     advisoryCallback
}

type advisoryContent struct {
	Code     string
	Severity uint8
	Message  string
	Reason   string
	Hint     string
}

func acceptAlwaysAdvisoryDefinition(content advisoryContent, subjects []ValueID) advisoryDefinition {
	definition := advisoryDefinition{
		kind:     AdvisoryKindAlways,
		code:     content.Code,
		severity: content.Severity,
		message:  content.Message,
		reason:   content.Reason,
		hint:     content.Hint,
		subjects: subjects,
	}
	return finalizeAdvisoryDefinition(definition)
}

func acceptConditionalAdvisoryDefinition(content advisoryContent, activation condition, subjects []ValueID) advisoryDefinition {
	definition := advisoryDefinition{
		kind:       AdvisoryKindConditional,
		code:       content.Code,
		severity:   content.Severity,
		message:    content.Message,
		reason:     content.Reason,
		hint:       content.Hint,
		activation: &activation,
		subjects:   subjects,
	}
	return finalizeAdvisoryDefinition(definition)
}

func acceptDetectorAdvisoryDefinition(content advisoryContent, subjects []ValueID, dependencies []dependencyRequirement, callback advisoryCallback) advisoryDefinition {
	definition := advisoryDefinition{
		kind:         AdvisoryKindDetector,
		code:         content.Code,
		severity:     content.Severity,
		message:      content.Message,
		reason:       content.Reason,
		hint:         content.Hint,
		subjects:     subjects,
		dependencies: dependencies,
		callback:     callback,
	}
	return finalizeAdvisoryDefinition(definition)
}

func finalizeAdvisoryDefinition(definition advisoryDefinition) advisoryDefinition {
	definition.subjects = append([]ValueID(nil), definition.subjects...)
	definition.dependencies = append([]dependencyRequirement(nil), definition.dependencies...)
	if definition.activation != nil {
		activationCopy := *definition.activation
		definition.activation = &activationCopy
	}
	if !definition.Valid() {
		panic("semantic: invalid Advisory definition")
	}
	return definition
}

func (definition advisoryDefinition) Valid() bool {
	if definition.kind < AdvisoryKindAlways || definition.kind > AdvisoryKindDetector || definition.code == "" ||
		(definition.severity != 1 && definition.severity != 2) ||
		definition.message == "" {
		return false
	}
	for _, id := range definition.subjects {
		if !id.present() {
			return false
		}
	}
	for _, dependency := range definition.dependencies {
		if !dependency.Destination().present() {
			return false
		}
	}
	if definition.kind == AdvisoryKindConditional {
		return definition.activation != nil && definition.activation.Valid() && definition.callback == nil &&
			len(definition.dependencies) == 0
	}
	if definition.kind == AdvisoryKindDetector {
		return definition.activation == nil && definition.callback != nil
	}
	return definition.activation == nil && definition.callback == nil && len(definition.dependencies) == 0
}

func (definition advisoryDefinition) Kind() AdvisoryKind { return definition.kind }

func (definition advisoryDefinition) Code() string { return definition.code }

func (definition advisoryDefinition) Severity() uint8 { return definition.severity }

func (definition advisoryDefinition) Message() string { return definition.message }

func (definition advisoryDefinition) Reason() string { return definition.reason }

func (definition advisoryDefinition) Hint() string { return definition.hint }

func (definition advisoryDefinition) Activation() (condition, bool) {
	if definition.activation == nil {
		return condition{}, false
	}
	return *definition.activation, true
}

func (definition advisoryDefinition) Subjects() []ValueID {
	return append([]ValueID(nil), definition.subjects...)
}

func (definition advisoryDefinition) Dependencies() []dependencyRequirement {
	return append([]dependencyRequirement(nil), definition.dependencies...)
}

func (definition advisoryDefinition) Evaluate(ctx context.Context, config any, dependencies Dependencies) (bool, error) {
	if definition.callback == nil {
		panic("semantic: Advisory has no callback")
	}
	return definition.callback(ctx, config, dependencies)
}

type advisoryRecord struct {
	meta       recordMeta[advisoryID]
	command    CommandID
	definition advisoryDefinition
}

func acceptAdvisoryRecord(id advisoryID, command CommandID, position declarationPosition) advisoryRecord {
	mustCommand(command)
	return advisoryRecord{
		meta:    acceptRecordMeta(id, position),
		command: command,
	}
}

// AdvisoryCode is the application-defined machine classification of an Advisory.
// It remains distinct from the normalized NoticeCode used for delivery.
type AdvisoryCode string

// Advisory describes valid but noteworthy invocation semantics. Message is required;
// Severity is Information or Warning; Reason and Hint are optional source-first
// presentation. An Advisory normalizes to NoticeAdvisory while retaining Code.
type Advisory struct {
	Code     AdvisoryCode
	Severity NoticeSeverity
	Message  string
	Reason   string
	Hint     string
}

// AdvisoryPredicate evaluates an advisory detector over read-only analysis state.
// true, nil matches; false, nil does not; any non-nil error becomes
// DiagnosticAdvisoryEvaluationFailure.
type AdvisoryPredicate[C any] func(context.Context, *C, Dependencies) (bool, error)
