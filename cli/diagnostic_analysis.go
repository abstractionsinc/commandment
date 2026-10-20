package cli

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

var (
	errDiagnosticNegativeProviderOutputIndex           = errors.New("invocation: negative provider-output index")
	errDiagnosticInvalidProviderOutputCommandEntryKind = errors.New("invocation: invalid provider-output command-entry kind")
	errDiagnosticInvalidSignalActionFailure            = errors.New("invocation: invalid diagnostic signal action failure")
	errDiagnosticInvalidSignalOverflowFailure          = errors.New("invocation: invalid diagnostic signal overflow failure")
	errDiagnosticEndpointOverflowMissingIdentity       = errors.New("invocation: endpoint overflow lacks endpoint identity")
	errDiagnosticInvalidNoticeDeliveryFailure          = errors.New("invocation: invalid diagnostic notice delivery failure")
	errDiagnosticInvalidRuleIdentityKind               = errors.New("invocation: invalid diagnostic Rule identity/kind")
	errDiagnosticInvalidZeroRuleSubject                = errors.New("invocation: invalid zero Rule subject")
	errDiagnosticInvalidRuleOperandLiteral             = errors.New("invocation: invalid Rule operand literal")
	errDiagnosticInvalidRuleCount                      = errors.New("invocation: invalid Rule count")
	errDiagnosticDeclarationIdentityValueWithoutKind   = errors.New("invocation: diagnostic declaration identity has value without kind")
	errDiagnosticDeclarationIdentitySubjectCount       = errors.New("invocation: diagnostic declaration identity must name exactly one subject")
	errDiagnosticCommandEntryIdentityMissingID         = errors.New("invocation: command-entry diagnostic identity lacks command-entry ID")
	errDiagnosticActionEntryIdentityMissingID          = errors.New("invocation: action-entry diagnostic identity lacks action-entry ID")
	errDiagnosticFlagIdentityMissingID                 = errors.New("invocation: flag diagnostic identity lacks flag ID")
	errDiagnosticFlagSpellingIdentityMissingID         = errors.New("invocation: flag-spelling diagnostic identity lacks spelling ID")
	errDiagnosticArgumentFormIdentityMissingID         = errors.New("invocation: argument-form diagnostic identity lacks form ID")
	errDiagnosticArgumentTermIdentityMissingID         = errors.New("invocation: argument-term diagnostic identity lacks term ID")
	errDiagnosticEnvironmentInputIdentityMissingID     = errors.New("invocation: environment-input diagnostic identity lacks environment ID")
	errDiagnosticValueDomainEntryIdentityMissingID     = errors.New("invocation: value-domain-entry diagnostic identity lacks entry ID")
)

type diagnosticEnumError struct {
	field string
	value string
}

func (err diagnosticEnumError) Error() string {
	return fmt.Sprintf("invocation: invalid diagnostic %s %q", err.field, err.value)
}

type diagnosticReplacementTargetError struct {
	cause error
}

func (err diagnosticReplacementTargetError) Error() string {
	return "invocation: invalid diagnostic replacement target: " + err.cause.Error()
}

func (err diagnosticReplacementTargetError) Unwrap() error { return err.cause }

// SourceID and OriginID are report-local projection identities. Each input
// Payload owns its own namespace; Extract rebases all payload namespaces into
// one deterministic report-local namespace.
type (
	sourceID uint32
	originID uint32
)

type analysisPosition struct {
	Offset int
	Line   int
	Column int
}

type span struct {
	Source sourceID
	Entry  int
	Start  analysisPosition
	End    analysisPosition
}

type source struct {
	ID              sourceID
	Kind            InputKind
	Revision        string
	Key             string
	Location        string
	Sensitive       bool
	CommandShortcut CommandEntryID
	Entries         []string
	FlatText        *string
}

type origin struct {
	ID        originID
	Span      span
	Parents   []originID
	Sensitive bool
}

type replacementPart struct {
	Literal   *string
	Copy      *span
	Sensitive bool
}

type replacement []replacementPart

type textEdit struct {
	Span        span
	Replacement replacement
}

type entryEdit struct {
	Start   int
	End     int
	Entries []replacement
}

type repairSource struct {
	Source sourceID

	TextEdits  []textEdit
	EntryEdits []entryEdit
}

type repair struct {
	Title            string
	PreservesMeaning bool
	Sensitive        bool
	Sources          []repairSource
}

// analysisDiagnosticRuleKind is the projection-layer spelling of the public RuleKind
// vocabulary. It is a detached transport value, not a second Rule authority.
type analysisDiagnosticRuleKind string

const (
	diagnosticRuleCustom       analysisDiagnosticRuleKind = "custom"
	diagnosticRuleRequire      analysisDiagnosticRuleKind = "require"
	diagnosticRuleForbid       analysisDiagnosticRuleKind = "forbid"
	diagnosticRuleNotEmpty     analysisDiagnosticRuleKind = "not-empty"
	diagnosticRuleIn           analysisDiagnosticRuleKind = "in"
	diagnosticRuleNotIn        analysisDiagnosticRuleKind = "not-in"
	diagnosticRuleEqual        analysisDiagnosticRuleKind = "equal"
	diagnosticRuleNotEqual     analysisDiagnosticRuleKind = "not-equal"
	diagnosticRuleBetween      analysisDiagnosticRuleKind = "between"
	diagnosticRuleAtLeast      analysisDiagnosticRuleKind = "at-least"
	diagnosticRuleAtMost       analysisDiagnosticRuleKind = "at-most"
	diagnosticRuleGreaterThan  analysisDiagnosticRuleKind = "greater-than"
	diagnosticRuleLessThan     analysisDiagnosticRuleKind = "less-than"
	diagnosticRuleMatch        analysisDiagnosticRuleKind = "match"
	diagnosticRuleItemCount    analysisDiagnosticRuleKind = "item-count"
	diagnosticRuleContains     analysisDiagnosticRuleKind = "contains"
	diagnosticRuleEachIn       analysisDiagnosticRuleKind = "each-in"
	diagnosticRuleUnique       analysisDiagnosticRuleKind = "unique"
	diagnosticRuleEntryCount   analysisDiagnosticRuleKind = "entry-count"
	diagnosticRuleHasKeys      analysisDiagnosticRuleKind = "has-keys"
	diagnosticRuleKeysIn       analysisDiagnosticRuleKind = "keys-in"
	diagnosticRuleSubsetOf     analysisDiagnosticRuleKind = "subset-of"
	diagnosticRuleDisjoint     analysisDiagnosticRuleKind = "disjoint"
	diagnosticRuleKeysSubsetOf analysisDiagnosticRuleKind = "keys-subset-of"
	diagnosticRuleKeysDisjoint analysisDiagnosticRuleKind = "keys-disjoint"
	diagnosticRuleExactlyOne   analysisDiagnosticRuleKind = "exactly-one"
	diagnosticRuleAllOrNone    analysisDiagnosticRuleKind = "all-or-none"
	diagnosticRuleAtMostOne    analysisDiagnosticRuleKind = "at-most-one"
	diagnosticRuleAtLeastOne   analysisDiagnosticRuleKind = "at-least-one"
)

type analysisDiagnosticValueType struct {
	Kind string
	Name string
}

type diagnosticValueLiteral struct {
	Type analysisDiagnosticValueType
	Text string

	Sensitive bool
	Redacted  bool
}

type diagnosticRuleOperand struct {
	Sensitive bool
	Literal   *diagnosticValueLiteral
}

type analysisDiagnosticRuleFailure struct {
	ID            uint32
	Kind          analysisDiagnosticRuleKind
	Code          string
	ViolationKind analysisDiagnosticViolationKind

	Subjects []uint32
	Operands []diagnosticRuleOperand

	Pattern string

	CountMinimum   int
	CountMaximum   int
	CountUnbounded bool
	CountValid     bool
}

type analysisDiagnosticViolationKind string

const (
	diagnosticViolationInvalid   analysisDiagnosticViolationKind = "invalid"
	diagnosticViolationConflict  analysisDiagnosticViolationKind = "conflict"
	diagnosticViolationDuplicate analysisDiagnosticViolationKind = "duplicate"
)

type analysisDiagnosticProducerKind string

const (
	diagnosticProducerKindFlag              analysisDiagnosticProducerKind = "flag"
	diagnosticProducerKindArgument          analysisDiagnosticProducerKind = "argument"
	diagnosticProducerKindRemainder         analysisDiagnosticProducerKind = "remainder"
	diagnosticProducerKindPassThrough       analysisDiagnosticProducerKind = "pass-through"
	diagnosticProducerKindShortcutDefault   analysisDiagnosticProducerKind = "shortcut-default"
	diagnosticProducerKindEnvironment       analysisDiagnosticProducerKind = "environment"
	diagnosticProducerKindInvocationDefault analysisDiagnosticProducerKind = "invocation-default"
	diagnosticProducerKindComputedDefault   analysisDiagnosticProducerKind = "computed-default"
	diagnosticProducerKindStaticDefault     analysisDiagnosticProducerKind = "static-default"
	diagnosticProducerKindInitial           analysisDiagnosticProducerKind = "initial"
	diagnosticProducerKindSuppliedFact      analysisDiagnosticProducerKind = "supplied-fact"
	diagnosticProducerKindResolvedFact      analysisDiagnosticProducerKind = "resolved-fact"
	diagnosticProducerKindDerived           analysisDiagnosticProducerKind = "derived"
)

type analysisDiagnosticResolutionTier string

const (
	diagnosticResolutionTierExplicit          analysisDiagnosticResolutionTier = "explicit"
	diagnosticResolutionTierShortcutDefault   analysisDiagnosticResolutionTier = "shortcut-default"
	diagnosticResolutionTierEnvironment       analysisDiagnosticResolutionTier = "environment"
	diagnosticResolutionTierInvocationDefault analysisDiagnosticResolutionTier = "invocation-default"
	diagnosticResolutionTierComputedDefault   analysisDiagnosticResolutionTier = "computed-default"
	diagnosticResolutionTierStaticDefault     analysisDiagnosticResolutionTier = "static-default"
	diagnosticResolutionTierInitial           analysisDiagnosticResolutionTier = "initial"
)

type analysisDiagnosticDisposition string

const (
	diagnosticDispositionEffective   analysisDiagnosticDisposition = "effective"
	diagnosticDispositionNoEffect    analysisDiagnosticDisposition = "no-effect"
	diagnosticDispositionSuppressed  analysisDiagnosticDisposition = "suppressed"
	diagnosticDispositionConflicting analysisDiagnosticDisposition = "conflicting"
	diagnosticDispositionRejected    analysisDiagnosticDisposition = "rejected"
	diagnosticDispositionUnresolved  analysisDiagnosticDisposition = "unresolved"
)

type diagnosticDeclarationKind string

const (
	diagnosticDeclarationCommandEntry     diagnosticDeclarationKind = "command-entry"
	diagnosticDeclarationActionEntry      diagnosticDeclarationKind = "action-entry"
	diagnosticDeclarationFlag             diagnosticDeclarationKind = "flag"
	diagnosticDeclarationFlagSpelling     diagnosticDeclarationKind = "flag-spelling"
	diagnosticDeclarationArgumentForm     diagnosticDeclarationKind = "argument-form"
	diagnosticDeclarationArgumentTerm     diagnosticDeclarationKind = "argument-term"
	diagnosticDeclarationEnvironmentInput diagnosticDeclarationKind = "environment-input"
	diagnosticDeclarationValueDomainEntry diagnosticDeclarationKind = "value-domain-entry"
)

type analysisDiagnosticDeclarationIdentity struct {
	Kind diagnosticDeclarationKind

	CommandEntry CommandEntryID
	ActionEntry  ActionEntryID
	Flag         FlagID
	FlagSpelling FlagSpellingID
	ArgumentForm ArgumentFormID
	ArgumentTerm ArgumentTermID

	EnvironmentInput EnvironmentInputID
	ValueDomainEntry ValueDomainEntryID
}

type diagnosticProviderOutput struct {
	Index int
	Kind  CommandEntryKind

	Key      string
	Location string
	Redacted bool
}

type diagnosticReplacement struct {
	Target analysisDiagnosticDeclarationIdentity

	Presentation string

	TargetDeprecatedDuringMigration bool
}

// diagnosticSignalActionKind is the detached projection spelling of the
// public signal action vocabulary. It transports already-finalized failure
// context without becoming signal routing authority.
type diagnosticSignalActionKind string

const (
	diagnosticSignalActionIgnore  diagnosticSignalActionKind = "ignore"
	diagnosticSignalActionCancel  diagnosticSignalActionKind = "cancel"
	diagnosticSignalActionHandle  diagnosticSignalActionKind = "handle"
	diagnosticSignalActionForward diagnosticSignalActionKind = "forward"
	diagnosticSignalActionSend    diagnosticSignalActionKind = "send"
	diagnosticSignalActionAbort   diagnosticSignalActionKind = "abort"
	diagnosticSignalActionReraise diagnosticSignalActionKind = "reraise"
	diagnosticSignalActionEmit    diagnosticSignalActionKind = "emit"
)

type diagnosticSignalAction struct {
	Kind diagnosticSignalActionKind
	Name string

	EndpointID SignalEndpointID

	Signal         Signal
	SignalValue    ValueID
	SignalRedacted bool

	Delayed bool

	Delay         time.Duration
	DelayValue    ValueID
	DelayRedacted bool

	ExitCode int

	Sensitive bool
}

type diagnosticSignalEvent struct {
	Signal Signal
	Origin SignalOrigin
	Sender *SignalSender
	Phase  SignalPhase

	ObservedAt      time.Time
	FirstObservedAt time.Time
	LastObservedAt  time.Time

	FirstOccurrence int
	LastOccurrence  int
	BatchSize       int
}

type diagnosticSignalActionFailure struct {
	CommandPath []string
	PolicyID    SignalPolicyID
	Index       int
	Action      diagnosticSignalAction
	Event       diagnosticSignalEvent
	Phase       SignalPhase
}

type diagnosticSignalOverflowBoundary string

const (
	diagnosticSignalOverflowSource   diagnosticSignalOverflowBoundary = "source"
	diagnosticSignalOverflowRouting  diagnosticSignalOverflowBoundary = "routing"
	diagnosticSignalOverflowEndpoint diagnosticSignalOverflowBoundary = "endpoint"
)

type diagnosticSignalOverflowFailure struct {
	Boundary   diagnosticSignalOverflowBoundary
	EndpointID SignalEndpointID
	Capacity   int
	Buffered   int
}

type analysisDiagnosticNoticePhase string

const (
	diagnosticNoticePhaseAnalysis  analysisDiagnosticNoticePhase = "analysis"
	diagnosticNoticePhaseExecution analysisDiagnosticNoticePhase = "execution"
	diagnosticNoticePhaseCleanup   analysisDiagnosticNoticePhase = "cleanup"
)

type diagnosticNoticeDeliveryFailure struct {
	Index       int
	Phase       analysisDiagnosticNoticePhase
	Code        string
	CommandPath []string
}

type diagnostic struct {
	Code string

	Rule         *analysisDiagnosticRuleFailure
	AdvisoryCode string

	Message string
	Reason  string
	Hint    string

	FrameworkMessage bool
	FrameworkReason  bool
	FrameworkHint    bool

	Subject     string
	CommandPath []string

	Command     CommandID
	Declaration analysisDiagnosticDeclarationIdentity
	Provider    CommandProviderID
	Default     CommandDefaultID

	ProviderOutput *diagnosticProviderOutput
	Replacement    *diagnosticReplacement

	Source  sourceID
	Span    *span
	Origins []originID

	Input         string
	InputRedacted bool
	Suggestions   []string
	Usage         []string
	Sensitive     bool

	ExitCode     int
	ProducerKind analysisDiagnosticProducerKind
	Tier         analysisDiagnosticResolutionTier
	Disposition  analysisDiagnosticDisposition

	RepairIndexes []int

	SignalAction   *diagnosticSignalActionFailure
	SignalOverflow *diagnosticSignalOverflowFailure
	NoticeDelivery *diagnosticNoticeDeliveryFailure
}

// payload is one frontend-owned diagnostic namespace before standalone report
// extraction. Sources and Origins may contain unrelated operation state;
// extraction retains only the transitive closure needed by diagnostics/repairs.
type payload struct {
	Diagnostics  []diagnostic
	Sources      map[sourceID]source
	Origins      map[originID]origin
	Repairs      []repair
	Truncated    bool
	Presentation *frameworkPresentationState
}

type report struct {
	Diagnostics  []diagnostic
	Sources      map[sourceID]source
	Origins      map[originID]origin
	Repairs      []repair
	Truncated    bool
	presentation *frameworkPresentationState
}

// ResolveFrameworkText resolves a Commandment-owned finite source phrase
// through the operation-local framework presentation context carried by this
// report. Reports without such context preserve source text.
func (report report) ResolveFrameworkText(source string) string {
	if report.presentation == nil {
		return source
	}
	return report.presentation.Resolve(source)
}

type diagnosticFailureError struct {
	payload payload
	cause   error
}

func (failure *diagnosticFailureError) Error() string {
	if len(failure.payload.Diagnostics) != 0 && failure.payload.Diagnostics[0].Code != "" {
		return "commandment: " + failure.payload.Diagnostics[0].Code
	}
	return "commandment: frontend failure"
}

// Format preserves the concise ordinary error surface and reserves %+v for the
// canonical structured Diagnostic presentation. Destination/process ownership
// remains with the caller performing the formatting.
func (failure *diagnosticFailureError) Format(state fmt.State, verb rune) {
	if verb == 'v' && state.Flag('+') && writeDiagnosticPresentation(state, failure) {
		return
	}
	formatErrorString(state, verb, failure.Error())
}

func (failure *diagnosticFailureError) Unwrap() error { return failure.cause }

// ownsDiagnosticPresentation reports whether err itself is the private
// Commandment Diagnostic error that owns the canonical human projection. It
// deliberately does not unwrap arbitrary application errors.
func ownsDiagnosticPresentation(err error) bool {
	_, ok := err.(*diagnosticFailureError) //nolint:errorlint // This predicate classifies only the exact top-level Diagnostic owner; wrapped application errors must remain ordinary.
	return ok
}

func formatErrorString(state fmt.State, verb rune, text string) {
	_, _ = fmt.Fprintf(state, fmt.FormatString(state, verb), text)
}

// retainDiagnosticFailure retains a frontend-owned diagnostic payload with an
// optional underlying cause. The payload is detached so later caller mutation
// cannot alter extraction.
func retainDiagnosticFailure(payload payload, cause error) error {
	return &diagnosticFailureError{
		payload: clonePayload(payload),
		cause:   cause,
	}
}

// extract returns the deterministic source-closed report from every frontend
// payload reachable through ordinary wrapping or errors.Join. Ordinary errors
// contribute no synthetic diagnostics.
func extract(err error) (report, error) {
	if err == nil {
		return report{}, nil
	}
	payloads, truncated := collectPayloads(err)
	if len(payloads) == 0 && !truncated {
		return report{}, nil
	}
	builder := reportBuilder{
		report: report{
			Sources:   map[sourceID]source{},
			Origins:   map[originID]origin{},
			Truncated: truncated,
		},
		nextSource: 1,
		nextOrigin: 1,
	}
	for _, payload := range payloads {
		if err := builder.addPayload(payload); err != nil {
			return report{}, err
		}
	}
	return builder.report, nil
}

func collectPayloads(err error) ([]payload, bool) {
	const maximumErrorGraphNodes = 4096
	const initialErrorGraphCapacity = 16
	var result []payload
	seenFailures := make(map[*diagnosticFailureError]struct{}, initialErrorGraphCapacity)
	stack := []error{err}
	visited := 0
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}
		if visited == maximumErrorGraphNodes {
			return result, true
		}
		visited++
		if failure, ok := current.(*diagnosticFailureError); ok { //nolint:errorlint // The explicit walker records each exact diagnostic wrapper once before descending.
			if _, duplicate := seenFailures[failure]; duplicate {
				continue
			}
			seenFailures[failure] = struct{}{}
			result = append(result, clonePayload(failure.payload))
		}
		if joined, ok := current.(interface{ Unwrap() []error }); ok {
			children := joined.Unwrap()
			for _, child := range slices.Backward(children) {
				stack = append(stack, child)
			}
			continue
		}
		if wrapped, ok := current.(interface{ Unwrap() error }); ok {
			stack = append(stack, wrapped.Unwrap())
		}
	}
	return result, false
}

func exactErrorIdentity(left, right error) bool {
	if left == nil || right == nil {
		return false
	}
	return exactInterfaceIdentity(left, right)
}

func exactInterfaceIdentity(left, right any) bool {
	equal := false
	func() {
		defer func() { _ = recover() }()
		equal = left == right
	}()
	return equal
}

func validateDiagnosticMetadata(diagnostic diagnostic) error {
	if diagnostic.Rule != nil {
		if err := validateDiagnosticRuleFailure(*diagnostic.Rule); err != nil {
			return err
		}
	}
	if diagnostic.ProducerKind != "" && !validDiagnosticProducerKind(diagnostic.ProducerKind) {
		return diagnosticEnumError{
			field: "producer kind",
			value: string(diagnostic.ProducerKind),
		}
	}
	if diagnostic.Tier != "" && !validDiagnosticResolutionTier(diagnostic.Tier) {
		return diagnosticEnumError{
			field: "resolution tier",
			value: string(diagnostic.Tier),
		}
	}
	if diagnostic.Disposition != "" && !validDiagnosticDisposition(diagnostic.Disposition) {
		return diagnosticEnumError{
			field: "candidate disposition",
			value: string(diagnostic.Disposition),
		}
	}
	if err := validateDiagnosticDeclarationIdentity(diagnostic.Declaration); err != nil {
		return err
	}
	if diagnostic.Replacement != nil {
		if err := validateDiagnosticDeclarationIdentity(diagnostic.Replacement.Target); err != nil {
			return diagnosticReplacementTargetError{cause: err}
		}
	}
	if diagnostic.ProviderOutput != nil {
		if diagnostic.ProviderOutput.Index < 0 {
			return errDiagnosticNegativeProviderOutputIndex
		}
		switch diagnostic.ProviderOutput.Kind {
		case CommandEntryCanonical,
			CommandEntryAlias,
			CommandEntryShortcut:
		default:
			return errDiagnosticInvalidProviderOutputCommandEntryKind
		}
	}
	if diagnostic.SignalAction != nil {
		event := diagnostic.SignalAction.Event
		action := diagnostic.SignalAction.Action
		if !diagnostic.SignalAction.PolicyID.present() || diagnostic.SignalAction.Index < 0 ||
			!diagnostic.SignalAction.Phase.valid() || !event.Phase.valid() || !event.Origin.valid() ||
			(action.Signal.valid() && !validDiagnosticSignal(action.Signal)) || !validDiagnosticSignal(event.Signal) ||
			event.FirstOccurrence < 0 || event.LastOccurrence < 0 || event.BatchSize < 0 ||
			!validDiagnosticSignalActionKind(action.Kind) {
			return errDiagnosticInvalidSignalActionFailure
		}
	}
	if diagnostic.SignalOverflow != nil {
		if !validDiagnosticSignalOverflowBoundary(diagnostic.SignalOverflow.Boundary) ||
			diagnostic.SignalOverflow.Capacity < 0 || diagnostic.SignalOverflow.Buffered < 0 {
			return errDiagnosticInvalidSignalOverflowFailure
		}
		if diagnostic.SignalOverflow.Boundary == diagnosticSignalOverflowEndpoint &&
			!diagnostic.SignalOverflow.EndpointID.present() {
			return errDiagnosticEndpointOverflowMissingIdentity
		}
	}
	if diagnostic.NoticeDelivery != nil {
		if diagnostic.NoticeDelivery.Index < 0 || !validDiagnosticNoticePhase(diagnostic.NoticeDelivery.Phase) ||
			diagnostic.NoticeDelivery.Code == "" {
			return errDiagnosticInvalidNoticeDeliveryFailure
		}
	}
	return nil
}

func validateDiagnosticRuleFailure(rule analysisDiagnosticRuleFailure) error {
	if rule.ID == 0 || !validDiagnosticRuleKind(rule.Kind) {
		return errDiagnosticInvalidRuleIdentityKind
	}
	if rule.ViolationKind != "" && !validDiagnosticViolationKind(rule.ViolationKind) {
		return diagnosticEnumError{
			field: "violation kind",
			value: string(rule.ViolationKind),
		}
	}
	if slices.Contains(rule.Subjects, 0) {
		return errDiagnosticInvalidZeroRuleSubject
	}
	for _, operand := range rule.Operands {
		if operand.Literal == nil {
			continue
		}
		literal := operand.Literal
		if !validDiagnosticValueType(literal.Type) || (literal.Redacted && literal.Text != "") {
			return errDiagnosticInvalidRuleOperandLiteral
		}
	}
	if rule.CountValid {
		if rule.CountMinimum < 0 || (!rule.CountUnbounded && rule.CountMaximum < rule.CountMinimum) {
			return errDiagnosticInvalidRuleCount
		}
	}
	return nil
}

func validDiagnosticValueType(value analysisDiagnosticValueType) bool {
	switch value.Kind {
	case "bool", "count", "int", "uint", "string", "sequence", "keyed":
		return value.Name == ""
	case "custom":
		return true
	default:
		return false
	}
}

func validDiagnosticSignal(signal Signal) bool { return signal.valid() }

func validDiagnosticSignalActionKind(kind diagnosticSignalActionKind) bool {
	switch kind {
	case diagnosticSignalActionIgnore,
		diagnosticSignalActionCancel,
		diagnosticSignalActionHandle,
		diagnosticSignalActionForward,
		diagnosticSignalActionSend,
		diagnosticSignalActionAbort,
		diagnosticSignalActionReraise,
		diagnosticSignalActionEmit:
		return true
	default:
		return false
	}
}

func validDiagnosticSignalOverflowBoundary(boundary diagnosticSignalOverflowBoundary) bool {
	switch boundary {
	case diagnosticSignalOverflowSource, diagnosticSignalOverflowRouting, diagnosticSignalOverflowEndpoint:
		return true
	default:
		return false
	}
}

func validDiagnosticNoticePhase(phase analysisDiagnosticNoticePhase) bool {
	switch phase {
	case diagnosticNoticePhaseAnalysis, diagnosticNoticePhaseExecution, diagnosticNoticePhaseCleanup:
		return true
	default:
		return false
	}
}

func validDiagnosticRuleKind(kind analysisDiagnosticRuleKind) bool {
	switch kind {
	case diagnosticRuleCustom,
		diagnosticRuleRequire,
		diagnosticRuleForbid,
		diagnosticRuleNotEmpty,
		diagnosticRuleIn,
		diagnosticRuleNotIn,
		diagnosticRuleEqual,
		diagnosticRuleNotEqual,
		diagnosticRuleBetween,
		diagnosticRuleAtLeast,
		diagnosticRuleAtMost,
		diagnosticRuleGreaterThan,
		diagnosticRuleLessThan,
		diagnosticRuleMatch,
		diagnosticRuleItemCount,
		diagnosticRuleContains,
		diagnosticRuleEachIn,
		diagnosticRuleUnique,
		diagnosticRuleEntryCount,
		diagnosticRuleHasKeys,
		diagnosticRuleKeysIn,
		diagnosticRuleSubsetOf,
		diagnosticRuleDisjoint,
		diagnosticRuleKeysSubsetOf,
		diagnosticRuleKeysDisjoint,
		diagnosticRuleExactlyOne,
		diagnosticRuleAllOrNone,
		diagnosticRuleAtMostOne,
		diagnosticRuleAtLeastOne:
		return true
	default:
		return false
	}
}

func validDiagnosticViolationKind(kind analysisDiagnosticViolationKind) bool {
	switch kind {
	case diagnosticViolationInvalid, diagnosticViolationConflict, diagnosticViolationDuplicate:
		return true
	default:
		return false
	}
}

func validDiagnosticProducerKind(source analysisDiagnosticProducerKind) bool {
	switch source {
	case diagnosticProducerKindFlag,
		diagnosticProducerKindArgument,
		diagnosticProducerKindRemainder,
		diagnosticProducerKindPassThrough,
		diagnosticProducerKindShortcutDefault,
		diagnosticProducerKindEnvironment,
		diagnosticProducerKindInvocationDefault,
		diagnosticProducerKindComputedDefault,
		diagnosticProducerKindStaticDefault,
		diagnosticProducerKindInitial,
		diagnosticProducerKindSuppliedFact,
		diagnosticProducerKindResolvedFact,
		diagnosticProducerKindDerived:
		return true
	default:
		return false
	}
}

func validDiagnosticResolutionTier(tier analysisDiagnosticResolutionTier) bool {
	switch tier {
	case diagnosticResolutionTierExplicit,
		diagnosticResolutionTierShortcutDefault,
		diagnosticResolutionTierEnvironment,
		diagnosticResolutionTierInvocationDefault,
		diagnosticResolutionTierComputedDefault,
		diagnosticResolutionTierStaticDefault,
		diagnosticResolutionTierInitial:
		return true
	default:
		return false
	}
}

func validDiagnosticDisposition(disposition analysisDiagnosticDisposition) bool {
	switch disposition {
	case diagnosticDispositionEffective,
		diagnosticDispositionNoEffect,
		diagnosticDispositionSuppressed,
		diagnosticDispositionConflicting,
		diagnosticDispositionRejected,
		diagnosticDispositionUnresolved:
		return true
	default:
		return false
	}
}

func validateDiagnosticDeclarationIdentity(identity analysisDiagnosticDeclarationIdentity) error {
	count := 0
	if identity.CommandEntry.present() {
		count++
	}
	if identity.ActionEntry.present() {
		count++
	}
	if identity.Flag.present() {
		count++
	}
	if identity.FlagSpelling.present() {
		count++
	}
	if identity.ArgumentForm.present() {
		count++
	}
	if identity.ArgumentTerm.present() {
		count++
	}
	if identity.EnvironmentInput.present() {
		count++
	}
	if identity.ValueDomainEntry.present() {
		count++
	}

	if identity.Kind == "" {
		if count != 0 {
			return errDiagnosticDeclarationIdentityValueWithoutKind
		}
		return nil
	}
	if count != 1 {
		return errDiagnosticDeclarationIdentitySubjectCount
	}
	switch identity.Kind {
	case diagnosticDeclarationCommandEntry:
		if !identity.CommandEntry.present() {
			return errDiagnosticCommandEntryIdentityMissingID
		}
	case diagnosticDeclarationActionEntry:
		if !identity.ActionEntry.present() {
			return errDiagnosticActionEntryIdentityMissingID
		}
	case diagnosticDeclarationFlag:
		if !identity.Flag.present() {
			return errDiagnosticFlagIdentityMissingID
		}
	case diagnosticDeclarationFlagSpelling:
		if !identity.FlagSpelling.present() {
			return errDiagnosticFlagSpellingIdentityMissingID
		}
	case diagnosticDeclarationArgumentForm:
		if !identity.ArgumentForm.present() {
			return errDiagnosticArgumentFormIdentityMissingID
		}
	case diagnosticDeclarationArgumentTerm:
		if !identity.ArgumentTerm.present() {
			return errDiagnosticArgumentTermIdentityMissingID
		}
	case diagnosticDeclarationEnvironmentInput:
		if !identity.EnvironmentInput.present() {
			return errDiagnosticEnvironmentInputIdentityMissingID
		}
	case diagnosticDeclarationValueDomainEntry:
		if !identity.ValueDomainEntry.present() {
			return errDiagnosticValueDomainEntryIdentityMissingID
		}
	default:
		return diagnosticEnumError{
			field: "declaration kind",
			value: string(identity.Kind),
		}
	}
	return nil
}
