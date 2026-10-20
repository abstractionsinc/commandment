package cli

// InspectionStatus classifies the completeness of one analyzed invocation. Rejected and incomplete
// artifacts may still contain valid semantic observations established before analysis stopped.
type InspectionStatus uint8

const (
	// InspectionStatusUnknown means no authoritative inspection completion state is available.
	InspectionStatusUnknown InspectionStatus = iota
	// InspectionStatusReady means analysis completed sufficiently to establish execution admission
	// and the requested invocation observation.
	InspectionStatusReady
	// InspectionStatusRejected means analysis established a semantic rejection; earlier valid
	// observations remain available in the artifact.
	InspectionStatusRejected
	// InspectionStatusIncomplete means analysis could not establish all required semantic phases;
	// preserved observations remain valid, but missing facts do not prove absence.
	InspectionStatusIncomplete
)

// InvocationAction identifies the Commandment action selected for an analyzed invocation;
// Inspection observes the selection but does not execute the action.
type InvocationAction uint8

const (
	// InvocationActionUnknown means no Commandment action has been authoritatively selected.
	InvocationActionUnknown InvocationAction = iota
	// InvocationActionRun selects normal application command execution; Inspection records that
	// selection without running the handler.
	InvocationActionRun
	// InvocationActionHelp selects help presentation for the invocation without making Inspection
	// execute a help renderer.
	InvocationActionHelp
	// InvocationActionVersion selects version presentation for the.
	InvocationActionVersion
)

// InvocationActionTrigger identifies the semantic cause that selected an InvocationAction,
// separate from later rendering, execution, or integration outcomes.
type InvocationActionTrigger uint8

const (
	// InvocationActionTriggerUnknown means the action-selection cause is not authoritatively known.
	InvocationActionTriggerUnknown InvocationActionTrigger = iota
	// InvocationActionTriggerCommand means command selection established the action through a command
	// action entry.
	InvocationActionTriggerCommand
	// InvocationActionTriggerEntry means an explicit action entry established the action.
	InvocationActionTriggerEntry
	// InvocationActionTriggerEmpty means the empty invocation path established the action.
	InvocationActionTriggerEmpty
	// InvocationActionTriggerStructural means Commandment structure rather than a command/entry
	// token established the action.
	InvocationActionTriggerStructural
)

// InvocationActionSelection records the selected Commandment action, its trigger class, exact
// action entry when applicable, and causal input origin.
type InvocationActionSelection struct {
	Selected InvocationAction
	Trigger  InvocationActionTrigger
	Entry    ActionEntryID
	Input    InputOriginID
}

// InvocationSelection records the command-selection and Commandment-action outcomes
// established for one analyzed.
type InvocationSelection struct {
	Command CommandSelection
	Action  InvocationActionSelection
}

// EvaluationOutcome records whether an invocation-time semantic evaluation was skipped, satisfied,
// unsatisfied, failed, or not authoritatively known.
type EvaluationOutcome uint8

const (
	// EvaluationOutcomeUnknown means the evaluation has no authoritative outcome.
	EvaluationOutcomeUnknown EvaluationOutcome = iota
	// EvaluationOutcomeSkipped means evaluation was intentionally not applied, such as an inactive
	// condition.
	EvaluationOutcomeSkipped
	// EvaluationOutcomeSatisfied means the evaluated predicate or condition held.
	EvaluationOutcomeSatisfied
	// EvaluationOutcomeUnsatisfied means the evaluated predicate or condition did not hold.
	EvaluationOutcomeUnsatisfied
	// EvaluationOutcomeFailed means evaluation could not produce a normal boolean outcome because the
	// evaluation itself failed.
	EvaluationOutcomeFailed
)

// TokenRole identifies the final semantic ownership role assigned to an analyzed lexical token.
// Ownership rejection is represented by TokenRoleUnknown together with a nonzero
// TokenInspection.RejectionCode.
type TokenRole uint8

const (
	// TokenRoleUnknown means no semantic ownership role was established; a nonzero RejectionCode may
	// explain an ownership rejection.
	TokenRoleUnknown TokenRole = iota
	// TokenRoleCommand marks a token owned by command selection.
	TokenRoleCommand
	// TokenRoleAction marks a token owned by Commandment action selection.
	TokenRoleAction
	// TokenRoleFlag marks a token as a flag spelling.
	TokenRoleFlag
	// TokenRoleFlagValue marks a token as a value supplied to a flag occurrence.
	TokenRoleFlagValue
	// TokenRolePositional marks a token as positional argument input.
	TokenRolePositional
	// TokenRoleLiteral marks a token owned by a literal argument term.
	TokenRoleLiteral
	// TokenRoleTokenLanguage marks a token owned by a token-language argument term.
	TokenRoleTokenLanguage
	// TokenRoleEndOfOptions marks the structural end-of-options delimiter.
	TokenRoleEndOfOptions
	// TokenRoleResponseFile marks a successfully recognized response-file include directive; included
	// tokens receive their ordinary semantic roles.
	TokenRoleResponseFile
	// TokenRoleRemainder marks input owned by a remainder argument term.
	TokenRoleRemainder
	// TokenRolePassThrough marks input owned by pass-through argument handling.
	TokenRolePassThrough
	// TokenRoleDelegated marks input delegated to another semantic consumer with DelegatedIndex
	// identifying its ordered position.
	TokenRoleDelegated
)

// TokenInspection records the smallest analyzed lexical unit with independent semantic ownership,
// including decoded/redacted text, provenance, role, exact declaration ownership, destination,
// lexical match facts, and semantic indexes.
type TokenInspection struct {
	Text     string
	Redacted bool
	Origin   InputOriginID

	Role          TokenRole
	RejectionCode DiagnosticCode

	Command     CommandID
	Declaration DeclarationIdentity
	Destination ValueID

	MatchedSpelling string
	Match           NameMatchKind
	FlagValueForm   FlagValueForm

	OccurrenceIndex int
	ElementIndex    int
	DelegatedIndex  int
}

// ArgumentFormOutcome records the invocation-time outcome for one argument form; inactive static
// forms belong to Schema/Reference rather than a synthetic not-applicable inspection state.
type ArgumentFormOutcome uint8

const (
	// ArgumentFormOutcomeUnknown means no authoritative per-form outcome is available.
	ArgumentFormOutcomeUnknown ArgumentFormOutcome = iota
	// ArgumentFormOutcomeNoMatch means the form was considered but did not match the analyzed input.
	ArgumentFormOutcomeNoMatch
	// ArgumentFormOutcomeRejected means the form matched enough to be considered but was rejected by
	// argument-form law.
	ArgumentFormOutcomeRejected
	// ArgumentFormOutcomeMatched means the form matched as a candidate but was not the selected
	// effective form.
	ArgumentFormOutcomeMatched
	// ArgumentFormOutcomeSelected means the form is the effective argument form.
	ArgumentFormOutcomeSelected
	// ArgumentFormOutcomeBlocked means evaluation could not proceed because a prerequisite semantic
	// dependency was blocked.
	ArgumentFormOutcomeBlocked
	// ArgumentFormOutcomeFailed means form evaluation itself failed.
	ArgumentFormOutcomeFailed
)

// ArgumentFormSelectionOutcome classifies the aggregate argument-form selection result
// independently of each form record.
type ArgumentFormSelectionOutcome uint8

const (
	// ArgumentFormSelectionUnknown means no authoritative aggregate selection outcome is available.
	ArgumentFormSelectionUnknown ArgumentFormSelectionOutcome = iota
	// ArgumentFormSelectionNotEvaluated means argument-form selection was not required or reached for
	// this.
	ArgumentFormSelectionNotEvaluated
	// ArgumentFormSelectionNoForms means selection was evaluated and the effective command had no
	// argument forms.
	ArgumentFormSelectionNoForms
	// ArgumentFormSelectionNoMatch means selection was evaluated and no form matched.
	ArgumentFormSelectionNoMatch
	// ArgumentFormSelectionRejected means selection produced a semantic rejection rather than a
	// selected form.
	ArgumentFormSelectionRejected
	// ArgumentFormSelectionSelected means exactly one effective form was selected.
	ArgumentFormSelectionSelected
	// ArgumentFormSelectionAmbiguous means more than one applicable form prevented unique selection.
	ArgumentFormSelectionAmbiguous
	// ArgumentFormSelectionBlocked means aggregate selection was blocked by prerequisite semantic
	// state.
	ArgumentFormSelectionBlocked
	// ArgumentFormSelectionFailed means aggregate form selection itself failed.
	ArgumentFormSelectionFailed
)

// ArgumentFormIssue records one structured argument-form issue and the input origins relevant to
// it.
type ArgumentFormIssue struct {
	Code         DiagnosticCode
	InputOrigins []InputOriginID
}

// ArgumentFormSelection records the aggregate argument-form selection outcome and the selected,
// matching, blocked, or failed form relationships that support it.
type ArgumentFormSelection struct {
	Outcome ArgumentFormSelectionOutcome

	Selected ArgumentFormID
	Matches  []ArgumentFormID
	Blocked  []ArgumentFormID
	Failed   []ArgumentFormID
}

// CommandProviderOutcome classifies one command-provider materialization attempt, distinguishing
// non-demand, accepted zero/output, rejected settlement, invalid staged output, operational
// failure, dependency blocking, and cancellation.
type CommandProviderOutcome uint8

const (
	// CommandProviderOutcomeUnknown means no authoritative provider materialization outcome is
	// available.
	CommandProviderOutcomeUnknown CommandProviderOutcome = iota
	// CommandProviderOutcomeNotNeeded means the provider was not demanded; this is distinct from a
	// successful provider that returned zero entries.
	CommandProviderOutcomeNotNeeded
	// CommandProviderOutcomeAccepted means the provider completed successfully and its valid staged
	// batch was committed, including the valid zero-output case.
	CommandProviderOutcomeAccepted
	// CommandProviderOutcomeRejected means a valid staged batch was rejected during provider
	// settlement and therefore received no committed output identities.
	CommandProviderOutcomeRejected
	// CommandProviderOutcomeInvalid means the callback returned successfully but its staged
	// declarations violated provider declaration law.
	CommandProviderOutcomeInvalid
	// CommandProviderOutcomeFailed means provider execution returned an operational error.
	CommandProviderOutcomeFailed
	// CommandProviderOutcomeBlocked means provider materialization could not run because a required
	// dependency was unavailable.
	CommandProviderOutcomeBlocked
	// CommandProviderOutcomeCanceled means provider materialization stopped because its operation
	// context was canceled.
	CommandProviderOutcomeCanceled
)

// CommandProviderOutputLocator identifies rejected uncommitted provider output within its staged
// batch. It is a diagnostic locator, not a committed semantic declaration ID.
type CommandProviderOutputLocator struct {
	Index int
	Kind  CommandEntryKind

	Key      string
	Location string
	Redacted bool
}

// ValueDomainOutcome classifies invocation-scoped value-domain materialization. A Ready domain may
// validly contain zero entries; a non-ready domain exposes no partial entries.
type ValueDomainOutcome uint8

const (
	// ValueDomainOutcomeUnknown means no authoritative dynamic-domain outcome is available.
	ValueDomainOutcomeUnknown ValueDomainOutcome = iota
	// ValueDomainOutcomeNotNeeded means the invocation-scoped domain was not demanded.
	ValueDomainOutcomeNotNeeded
	// ValueDomainOutcomeReady means materialization succeeded; zero committed entries is a valid
	// authoritative result.
	ValueDomainOutcomeReady
	// ValueDomainOutcomeInvalid means a successful callback produced an invalid staged domain result.
	ValueDomainOutcomeInvalid
	// ValueDomainOutcomeFailed means domain materialization returned an operational error.
	ValueDomainOutcomeFailed
	// ValueDomainOutcomeBlocked means domain materialization could not run because a dependency was
	// unavailable.
	ValueDomainOutcomeBlocked
	// ValueDomainOutcomeCanceled means domain materialization stopped because its operation context
	// was canceled.
	ValueDomainOutcomeCanceled
)

// ExecutionAdmissionStatus classifies declarative execution admission observed by Inspection; it is not an
// execution trace or executable plan.
type ExecutionAdmissionStatus uint8

const (
	// ExecutionAdmissionUnknown means Inspection could not authoritatively establish execution
	// admission.
	ExecutionAdmissionUnknown ExecutionAdmissionStatus = iota
	// ExecutionAdmissionNotApplicable means the selected Commandment action does not require
	// application command execution.
	ExecutionAdmissionNotApplicable
	// ExecutionAdmissionNotRunnable means the selected command is not executable even though invocation
	// analysis otherwise identified it.
	ExecutionAdmissionNotRunnable
	// ExecutionAdmissionBlocked means diagnostics or policy prevent execution admission.
	ExecutionAdmissionBlocked
	// ExecutionAdmissionReady means Inspection established that the selected command is admitted for
	// execution; no handler has run yet.
	ExecutionAdmissionReady
)

// MiddlewareScope identifies whether an effective middleware observation is application-wide or
// command-owned.
type MiddlewareScope uint8

const (
	// MiddlewareScopeUnknown means middleware ownership scope is not authoritatively known.
	MiddlewareScopeUnknown MiddlewareScope = iota
	// MiddlewareScopeApplication means the middleware is application-scoped in the effective
	// execution order.
	MiddlewareScopeApplication
	// MiddlewareScopeCommand means the middleware is owned by the identified command.
	MiddlewareScopeCommand
)

// MiddlewareInspection records one effective middleware position in execution order without
// executing the middleware.
type MiddlewareInspection struct {
	Scope MiddlewareScope

	Command          CommandID
	DeclarationIndex int
}

// ExecutionAdmissionInspection records declarative execution-admission state: handler command, effective
// middleware order, signal policy/endpoint relationships, and blocking diagnostics. Inspection
// never runs middleware, handlers, signal callbacks, cleanup, or notice delivery.
type ExecutionAdmissionInspection struct {
	Status ExecutionAdmissionStatus

	HandlerCommand CommandID
	Middleware     []MiddlewareInspection

	SignalPolicies  []SignalPolicyID
	SignalEndpoints []SignalEndpointID

	BlockingDiagnosticIndexes []int
}

// AdvisoryOutcome records the invocation-time disposition of an advisory evaluation.
type AdvisoryOutcome uint8

const (
	// AdvisoryOutcomeUnknown means no authoritative advisory evaluation outcome is available.
	AdvisoryOutcomeUnknown AdvisoryOutcome = iota
	// AdvisoryOutcomeNotEvaluated means the advisory was not evaluated for this.
	AdvisoryOutcomeNotEvaluated
	// AdvisoryOutcomeNotMatched means the advisory was evaluated and did not apply.
	AdvisoryOutcomeNotMatched
	// AdvisoryOutcomeMatched means the advisory applies to the analyzed.
	AdvisoryOutcomeMatched
	// AdvisoryOutcomeBlocked means advisory evaluation could not proceed because a dependency was
	// blocked.
	AdvisoryOutcomeBlocked
	// AdvisoryOutcomeFailed means advisory evaluation itself failed.
	AdvisoryOutcomeFailed
)

// NoticeSuppressionDisposition records whether a notice remains deliverable or was suppressed by
// delivery policy during analysis.
type NoticeSuppressionDisposition uint8

const (
	// NoticeSuppressionUnknown means no authoritative suppression disposition is available.
	NoticeSuppressionUnknown NoticeSuppressionDisposition = iota
	// NoticeSuppressionNotSuppressed means delivery policy did not suppress the notice.
	NoticeSuppressionNotSuppressed
	// NoticeSuppressionSuppressed means delivery policy suppressed the notice; the Notice
	// remains present as an inspected semantic observation.
	NoticeSuppressionSuppressed
)

// AdvisoryReference is the static/reference-facing projection of an advisory declaration,
// including origin, profile, activation, reachability, subjects, and dependencies.
type AdvisoryReference struct {
	Advisory

	Command          CommandID
	Origin           DeclarationOrigin
	DeclarationIndex int
	Profile          GrammarProfileID
	Kind             AdvisoryKind

	Activation   *ConditionExpression
	Reachability Reachability

	Subjects     []ValueID
	Dependencies []ValueID
}

// NoticeSuppressionLocator identifies the declaration location of a notice-suppression rule that
// affected an inspected notice.
type NoticeSuppressionLocator struct {
	Command          CommandID
	DeclarationIndex int
}

// Inspection is a detached semantic observation of one analyzed invocation. It may materialize
// phase-legal values, policies, providers, domains, diagnostics, notices, and execution admission,
// but never executes middleware/handlers or host termination. After analysis failure it preserves
// the most complete valid partial artifact, so absence from an incomplete or truncated observation
// does not by itself prove semantic absence. The zero value has InspectionStatusUnknown and is not
// evidence of completed analysis.
type Inspection struct {
	Status           InspectionStatus
	Selection        InvocationSelection
	EvaluatedRelease string
	Locale           Locale

	CommandOrder []CommandID
	Commands     map[CommandID]CommandInspection

	CommandEntries   map[CommandEntryID]CommandEntryInspection
	ActionEntries    map[ActionEntryID]ActionEntryInspection
	CommandDefaults  map[CommandDefaultID]CommandDefaultInspection
	CommandProviders map[CommandProviderID]CommandProviderInspection

	Flags              map[FlagID]FlagInspection
	FlagSpellings      map[FlagSpellingID]FlagSpellingInspection
	ArgumentForms      map[ArgumentFormID]ArgumentFormInspection
	ArgumentTerms      map[ArgumentTermID]ArgumentTermInspection
	Facts              map[ValueID]FactInspection
	EnvironmentInputs  map[EnvironmentInputID]EnvironmentInputInspection
	ValueDomains       []ValueDomainInspection
	ValueDomainEntries map[ValueDomainEntryID]ValueDomainEntryInspection

	Values          map[ValueID]ValueInspection
	ValueCandidates map[ValueCandidateID]ValueCandidateInspection

	Sources map[InputSourceID]InputSource
	Origins map[InputOriginID]InputOrigin
	Tokens  []TokenInspection

	SignalSource    SignalSourceInspection
	SignalPolicies  map[SignalPolicyID]SignalPolicyInspection
	SignalEndpoints map[SignalEndpointID]SignalEndpointInspection

	Diagnostics []Diagnostic
	Repairs     []InputRepair
	Notices     []NoticeInspection

	ExecutionAdmission ExecutionAdmissionInspection

	Truncated bool
}

// CommandNamespaceCompleteness records whether a command namespace was resolved completely for this
// Inspection demand graph. Inspection does not materialize a namespace merely for hypothetical
// later consumers.
type CommandNamespaceCompleteness uint8

const (
	// CommandNamespaceUnknown means no authoritative namespace resolution state is available.
	CommandNamespaceUnknown CommandNamespaceCompleteness = iota
	// CommandNamespaceUnresolved means the namespace was not resolved by the Inspection demand graph.
	CommandNamespaceUnresolved
	// CommandNamespaceComplete means the demanded namespace was resolved completely.
	CommandNamespaceComplete
	// CommandNamespaceIncomplete means namespace materialization began but analysis preserved only a
	// valid partial namespace; omission within it does not prove absence.
	CommandNamespaceIncomplete
)

// CommandInspection is the invocation-time projection of one effective command, preserving stable
// command identity/topology while adding namespace, availability, presence, selected-profile,
// provider, and invocation-sensitive facts.
type CommandInspection struct {
	ID             CommandID
	CanonicalEntry CommandEntryID
	Parent         CommandID
	Children       []CommandID

	NamespaceCompleteness CommandNamespaceCompleteness

	Origin DeclarationOrigin

	Name        string
	Path        []string
	Description Description
	Examples    []Example

	Hidden    bool
	Available bool
	Visible   bool
	Runnable  bool

	LifecycleState LifecycleState

	Dialect         *CommandDialect
	ProfileSelector ValueID
	Profile         GrammarProfileID
	Access          ExecutionAccess

	DefaultSelection      DefaultCommandSelection
	ArgumentFormSelection ArgumentFormSelection

	CommandEntries   []CommandEntryID
	ActionEntries    []ActionEntryID
	CommandDefaults  []CommandDefaultID
	CommandProviders []CommandProviderID

	Flags         []FlagID
	Facts         []ValueID
	ArgumentForms []ArgumentFormID
	Values        []ValueID

	Conditions         []ConditionInspection
	Rules              []RuleInspection
	Advisories         []AdvisoryInspection
	NoticeSuppressions []NoticeSuppressionInspection
	SignalPolicies     []SignalPolicyID
}

// CommandShortcutDefaultInspection records one effective shortcut default in an analyzed
// invocation, including destination, generated values, sensitivity, and causal input origins.
type CommandShortcutDefaultInspection struct {
	Destination ValueID
	Spelling    string

	Values   []ValueLiteral
	Redacted bool
	Active   bool

	Key      string
	Location string
}

// CommandEntryInspection is the invocation-time projection of a command entry, including stable
// identity, lifecycle/visibility, presence, provider provenance, and provider-local key/location
// when materialized dynamically.
type CommandEntryInspection struct {
	ID     CommandEntryID
	Parent CommandID
	Kind   CommandEntryKind
	Origin DeclarationOrigin

	DeclarationIndex int
	Profile          GrammarProfileID

	Name       string
	TargetPath []CommandID

	Description Description
	Examples    []Example

	Args       []string
	AppendArgs []string
	Defaults   []CommandShortcutDefaultInspection

	Hidden    bool
	Available bool
	Visible   bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement

	Provider CommandProviderID
	Key      string
	Location string
}

// ActionEntryInspection is the invocation-time projection of an action entry, including stable
// identity, selected action semantics, presentation, lifecycle, and visibility.
type ActionEntryInspection struct {
	ID     ActionEntryID
	Parent CommandID
	Kind   ActionEntryKind
	Action InvocationAction
	Origin DeclarationOrigin

	DeclarationIndex int

	Long    string
	Short   rune
	Literal string
	Name    string

	Placement FlagPlacement

	Description Description
	Hidden      bool
	Available   bool
	Visible     bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// CommandDefaultInspection records one default-command declaration together with its
// invocation-time condition outcome and reachability; aggregate selection is recorded separately.
type CommandDefaultInspection struct {
	ID     CommandDefaultID
	Parent CommandID
	Target CommandID
	Origin DeclarationOrigin

	DeclarationIndex int
	Profile          GrammarProfileID

	Condition        *ConditionExpression
	ConditionOutcome EvaluationOutcome
	Reachability     Reachability
}

// CommandProviderInspection records one provider declaration and its materialization outcome,
// dependencies, retained localization snapshot when independently configured, and committed
// CommandEntries. Rejected staged output is not assigned committed declaration identity.
type CommandProviderInspection struct {
	ID     CommandProviderID
	Parent CommandID

	DeclarationIndex int
	Profile          GrammarProfileID

	Dependencies []ValueID
	Outcome      CommandProviderOutcome

	Localization *Localization

	CommandEntries []CommandEntryID
}

// FlagInspection is the invocation-time projection of one flag, preserving stable declaration
// relationships while adding effective availability/visibility and whether the flag is present in
// the analyzed input.
type FlagInspection struct {
	ID      FlagID
	Command CommandID
	Origin  DeclarationOrigin
	Profile GrammarProfileID

	Destination ValueID
	Spellings   []FlagSpellingID

	ValueName   string
	Description string
	Type        ValueType

	Occurrences         CountRange
	ValuesPerOccurrence CountRange
	Placement           FlagPlacement

	Required       bool
	Hidden         bool
	Sensitive      bool
	LocaleSelector bool

	Completion CompletionDeclarationCapabilities
	Available  bool
	Visible    bool
	Present    bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// FlagSpellingInspection is the exact invocation-time spelling record, preserving spelling-local
// lifecycle/hidden state and whether that spelling is present in the analyzed input.
type FlagSpellingInspection struct {
	ID     FlagSpellingID
	Flag   FlagID
	Origin DeclarationOrigin

	Spelling FlagSpelling
	Hidden   bool

	Available bool
	Visible   bool
	Present   bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// FactInspection records invocation-time fact resolver availability, sensitivity, and
// dependencies; the resolved value itself is observed through the value-resolution model.
type FactInspection struct {
	ID ValueID

	HasResolver       bool
	ResolverSensitive bool
	Dependencies      []ValueID
}

// ArgumentFormInspection is the invocation-time projection of one argument form, including its
// selection outcome, issues, lifecycle/visibility, reachability, and ordered term identities.
type ArgumentFormInspection struct {
	ID      ArgumentFormID
	Command CommandID
	Origin  DeclarationOrigin
	Profile GrammarProfileID

	Terms []ArgumentTermID

	Outcome ArgumentFormOutcome
	Issues  []ArgumentFormIssue

	Available bool
	Visible   bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
	Reachability   Reachability
}

// ArgumentTermInspection is the invocation-time projection of one argument term. InputOrigins may
// retain attempted allocation for rejected candidates, while authoritative selected ownership
// remains in Inspection.Tokens.
type ArgumentTermInspection struct {
	ID     ArgumentTermID
	Form   ArgumentFormID
	Origin DeclarationOrigin

	Kind        ArgumentTermKind
	Destination ValueID

	Name        string
	Literal     string
	Description string
	Type        ValueType
	Count       CountRange

	Optional        bool
	Greedy          bool
	Minimal         bool
	EndOfOptions    bool
	Hidden          bool
	Sensitive       bool
	SourceSensitive bool
	HyphenValues    HyphenValueMode

	TokenLanguage *TokenLanguageSyntax
	Completion    CompletionDeclarationCapabilities
	Available     bool
	Visible       bool
	Present       bool

	InputOrigins []InputOriginID

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// EnvironmentInputInspection is the invocation-time projection of a declared environment input and
// its effective availability.
type EnvironmentInputInspection struct {
	EnvironmentInputSchema

	Available bool
}

// ValueDomainEntryInspection is the exact invocation-time projection of a committed value-domain
// entry, including canonical identity, lifecycle, replacement, and visibility.
type ValueDomainEntryInspection struct {
	ID     ValueDomainEntryID
	Kind   ValueDomainEntryKind
	Origin DeclarationOrigin

	Name      string
	Canonical ValueDomainEntryID

	Description Description
	Hidden      bool
	Available   bool
	Visible     bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ValueDomainInspection records one static or invocation-scoped domain use, its
// dependencies/materialization outcome, ordered uses, and committed entry identities.
type ValueDomainInspection struct {
	InvocationScoped bool
	Command          CommandID

	Dependencies []ValueID
	Outcome      ValueDomainOutcome

	Uses    []ValueDomainUse
	Entries []ValueDomainEntryID
}

// ValueRepresentationKind identifies whether a safe canonical value representation is scalar text
// or decoded token entries.
type ValueRepresentationKind uint8

const (
	// ValueRepresentationUnknown means no authoritative canonical representation is available.
	ValueRepresentationUnknown ValueRepresentationKind = iota
	// ValueRepresentationText means the canonical safe representation is scalar text.
	ValueRepresentationText
	// ValueRepresentationTokens means the canonical safe representation is an ordered decoded token
	// sequence.
	ValueRepresentationTokens
)

// ValueRepresentation carries canonical safe text or decoded token entries for an inspected value.
// It is not shell syntax, replay argv, or arbitrary application memory, and Redacted marks
// representation withheld for sensitivity.
type ValueRepresentation struct {
	Kind ValueRepresentationKind

	Text   string
	Tokens []string

	Redacted bool
}

// ValueInputInspection records observed candidate input text when safe to expose and marks
// redacted sensitive input explicitly.
type ValueInputInspection struct {
	Text     string
	Redacted bool
}

// ValueComponentInspection records one candidate-owned value component with input, canonical
// representation, domain identity, provenance, sensitivity, and diagnostic relationships.
type ValueComponentInspection struct {
	Input          *ValueInputInspection
	Representation *ValueRepresentation
	DomainEntry    ValueDomainEntryID

	Origins []InputOriginID

	Sensitive bool

	DiagnosticIndexes []int
}

// ValueEntryInspection records one candidate-owned keyed entry as separate key and value
// components so provenance, sensitivity, and diagnostics remain component-specific.
type ValueEntryInspection struct {
	Key   ValueComponentInspection
	Value ValueComponentInspection
}

// ValueCandidateContribution records the semantic granularity of a candidate contribution; it is
// declared resolution meaning and is not inferred from Go container shape.
type ValueCandidateContribution uint8

const (
	// ValueCandidateContributionUnknown means contribution granularity is not authoritatively known.
	ValueCandidateContributionUnknown ValueCandidateContribution = iota
	// ValueCandidateContributionWhole means the candidate contributes or attempts to contribute the
	// complete destination value.
	ValueCandidateContributionWhole
	// ValueCandidateContributionSequenceSegment means the candidate contributes one ordered segment
	// to a sequence destination.
	ValueCandidateContributionSequenceSegment
	// ValueCandidateContributionKeyedEntry means the candidate contributes one keyed entry.
	ValueCandidateContributionKeyedEntry
	// ValueCandidateContributionKeyedReplacement means the candidate contributes a complete keyed
	// replacement rather than one entry.
	ValueCandidateContributionKeyedReplacement
)

// ValueCandidateSuppression records why a candidate was suppressed so precedence and same-tier
// keep-first/keep-last behavior remain explicit.
type ValueCandidateSuppression uint8

const (
	// ValueCandidateSuppressionUnknown means the suppression cause is not authoritatively known.
	ValueCandidateSuppressionUnknown ValueCandidateSuppression = iota
	// ValueCandidateSuppressionPrecedence means a higher-precedence candidate suppressed this
	// candidate.
	ValueCandidateSuppressionPrecedence
	// ValueCandidateSuppressionKeepFirst means same-tier resolution retained the earlier candidate
	// for the affected semantic unit.
	ValueCandidateSuppressionKeepFirst
	// ValueCandidateSuppressionKeepLast means same-tier resolution retained the later candidate for
	// the affected semantic unit.
	ValueCandidateSuppressionKeepLast
)

// ValueProducerInspection records one value producer outcome with source tier, sensitivity,
// origins, dependencies/blockers, and diagnostic relationships.
type ValueProducerInspection struct {
	Kind    ValueProducerKind
	Tier    ResolutionTier
	Outcome ValueProducerOutcome

	Sensitive bool

	Origins []InputOriginID

	Dependencies []ValueID
	BlockedBy    []ValueID

	DiagnosticIndexes []int
}

// ValueCandidateInspection is the single result-local record for one value candidate. It preserves
// producer order, disposition, contribution granularity, input/representation, provenance,
// sensitivity, component structure, suppression/conflict relationships, and diagnostics.
type ValueCandidateInspection struct {
	ID          ValueCandidateID
	Destination ValueID

	ProducerIndex int

	Disposition  ValueCandidateDisposition
	Contribution ValueCandidateContribution
	Declaration  DeclarationIdentity

	Input          *ValueInputInspection
	Representation *ValueRepresentation
	DomainEntry    ValueDomainEntryID

	Origins   []InputOriginID
	Sensitive bool

	Elements []ValueComponentInspection
	Entries  []ValueEntryInspection

	Suppression   ValueCandidateSuppression
	SuppressedBy  []ValueCandidateID
	ConflictsWith []ValueCandidateID

	DiagnosticIndexes []int
}

// ValueElementReference points from a resolved sequence element to the candidate-owned component
// that established it.
type ValueElementReference struct {
	Candidate ValueCandidateID
	Index     int
}

// ValueEntryReference points from a resolved keyed entry to the candidate-owned entry that
// established it.
type ValueEntryReference struct {
	Candidate ValueCandidateID
	Index     int
}

// ValueInspection records one destination resolution outcome. Candidate order is deterministic,
// Contributors names the candidates composing the final result, and source/tier truth remains on
// producer/candidate history rather than a synthetic singular source.
type ValueInspection struct {
	ID      ValueID
	Kind    ValueDestinationKind
	HasType bool
	Type    ValueType

	State     ResolutionState
	Sensitive bool

	Producers []ValueProducerInspection

	Candidates   []ValueCandidateID
	Contributors []ValueCandidateID

	BlockedBy []ValueID

	DiagnosticIndexes []int

	Representation *ValueRepresentation

	Elements []ValueElementReference
	Entries  []ValueEntryReference
}

// ConditionInspection records one evaluated condition effect against an exact declaration target,
// including outcome, expression, reachability, sensitivity, and profile.
type ConditionInspection struct {
	Outcome EvaluationOutcome

	Target DeclarationIdentity
	Effect ConditionEffect

	Expression   ConditionExpression
	Reachability Reachability
	Sensitive    bool
	Profile      GrammarProfileID
}

// RuleInspection records one rule evaluation with stable RuleID, declaration position,
// subjects/dependencies, operands, activation/outcomes, reachability, sensitivity, and diagnostic
// relationships.
type RuleInspection struct {
	ID               RuleID
	DeclarationIndex int
	Profile          GrammarProfileID

	Kind    RuleKind
	Code    RuleCode
	Outcome EvaluationOutcome

	Subjects     []ValueID
	Dependencies []ValueID
	Operands     []RuleOperand
	Pattern      string
	Count        CountRange

	Activation        *ConditionExpression
	ActivationOutcome EvaluationOutcome
	Reachability      Reachability

	DiagnosticIndexes []int

	Sensitive bool
}

// AdvisoryInspection records one advisory evaluation, preserving declaration identity context,
// activation/outcome, reachability, subjects, and dependencies. Inspection does not deliver the
// advisory through RunRequest.Notify.
type AdvisoryInspection struct {
	Advisory

	Command          CommandID
	Origin           DeclarationOrigin
	DeclarationIndex int
	Profile          GrammarProfileID
	Kind             AdvisoryKind
	Outcome          AdvisoryOutcome

	Activation        *ConditionExpression
	ActivationOutcome EvaluationOutcome
	Reachability      Reachability

	Subjects     []ValueID
	Dependencies []ValueID
}

// NoticeSuppressionInspection records one notice-suppression declaration and its evaluated
// condition/reachability; suppression is delivery policy, not notice semantic deletion.
type NoticeSuppressionInspection struct {
	Command          CommandID
	Origin           DeclarationOrigin
	DeclarationIndex int
	Profile          GrammarProfileID

	Match     NoticeMatch
	Condition ConditionExpression

	ConditionOutcome EvaluationOutcome
	Reachability     Reachability
}

// NoticeInspection records an observed Notice together with suppression disposition, exact
// suppressing declaration locations, and ordered references to canonical Inspection.Repairs.
type NoticeInspection struct {
	Notice

	Suppression  NoticeSuppressionDisposition
	SuppressedBy []NoticeSuppressionLocator

	RepairIndexes []int
}

// SignalIntegrationStatus classifies whether the analyzed invocation requires signal integration
// and whether the supplied source/host capabilities can satisfy it.
type SignalIntegrationStatus uint8

const (
	// SignalIntegrationStatusUnknown means signal integration compatibility was not authoritatively
	// established.
	SignalIntegrationStatusUnknown SignalIntegrationStatus = iota
	// SignalIntegrationStatusNotRequired means the analyzed invocation has no effective
	// signal-integration requirement.
	SignalIntegrationStatusNotRequired
	// SignalIntegrationStatusMissing means required signal integration is absent from the supplied
	// invocation host state.
	SignalIntegrationStatusMissing
	// SignalIntegrationStatusReady means the supplied source and host capabilities satisfy the
	// effective signal requirements.
	SignalIntegrationStatusReady
	// SignalIntegrationStatusIncompatible means signal integration is supplied but lacks required
	// signals or capabilities.
	SignalIntegrationStatusIncompatible
	// SignalIntegrationStatusFailed means inspection of the supplied signal integration failed.
	SignalIntegrationStatusFailed
)

// SignalPolicyStatus records whether an effective signal policy is inactive, ready, or blocked for
// the analyzed.
type SignalPolicyStatus uint8

const (
	// SignalPolicyStatusUnknown means no authoritative effective policy status is available.
	SignalPolicyStatusUnknown SignalPolicyStatus = iota
	// SignalPolicyStatusInactive means the policy does not apply to this analyzed.
	SignalPolicyStatusInactive
	// SignalPolicyStatusReady means the policy is effective and its prerequisites are satisfied.
	SignalPolicyStatusReady
	// SignalPolicyStatusBlocked means the policy applies but a required signal/source/host
	// prerequisite prevents readiness.
	SignalPolicyStatusBlocked
)

// SignalEndpointStatus records whether a signal endpoint is inactive, ready, or blocked for the
// analyzed.
type SignalEndpointStatus uint8

const (
	// SignalEndpointStatusUnknown means no authoritative endpoint status is available.
	SignalEndpointStatusUnknown SignalEndpointStatus = iota
	// SignalEndpointStatusInactive means no effective policy requires the endpoint for this.
	//
	SignalEndpointStatusInactive
	// SignalEndpointStatusReady means the endpoint is required and all effective policy prerequisites
	// are satisfied.
	SignalEndpointStatusReady
	// SignalEndpointStatusBlocked means an effective policy targets the endpoint but prerequisites
	// prevent readiness.
	SignalEndpointStatusBlocked
)

// SignalEndpointInspection records one endpoint together with its invocation-time status and the
// effective or blocked policy relationships that target it.
type SignalEndpointInspection struct {
	SignalEndpoint

	Status SignalEndpointStatus

	EffectivePolicies []SignalPolicyID
	BlockedPolicies   []SignalPolicyID
}

// SignalSourceInspection records the analyzed signal-integration contract: whether a source is
// required/supplied/described, resolved or missing signals, and required/available/missing source
// and host capabilities. Inspection does not subscribe to the source.
type SignalSourceInspection struct {
	Status SignalIntegrationStatus

	Required  bool
	Supplied  bool
	Described bool

	ResolvedSignals []Signal
	MissingSignals  []Signal

	RequiredSourceCapabilities SignalSourceCapabilities
	MissingSourceCapabilities  SignalSourceCapabilities

	RequiredHostCapabilities  SignalHostCapabilities
	AvailableHostCapabilities SignalHostCapabilities
	MissingHostCapabilities   SignalHostCapabilities
}

// SignalPolicyInspection is the invocation-time projection of one signal policy, preserving
// identity, selector/occurrence/actions, capabilities, sensitivity, activation/reachability, and
// effective status without executing signal callbacks.
type SignalPolicyInspection struct {
	ID        SignalPolicyID
	CounterID SignalCounterID

	Command CommandID
	Origin  DeclarationOrigin
	Profile GrammarProfileID

	Selector    SignalSelector
	Occurrence  SignalOccurrenceRange
	Aggregation SignalAggregation

	Deferred         bool
	ContinueMatching bool

	Hint string

	Actions []SignalAction

	RequiredSourceCapabilities SignalSourceCapabilities
	RequiredHostCapabilities   SignalHostCapabilities

	Sensitive bool

	Activation        *ConditionExpression
	ActivationOutcome EvaluationOutcome
	Reachability      Reachability

	Status SignalPolicyStatus
}
