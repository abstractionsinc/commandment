package cli

import "time"

// ArgumentTermKind classifies positional-term ownership grammar. Its zero value is the
// unknown/unset classification.
type ArgumentTermKind uint8

const (
	// ArgumentTermUnknown means no positional-term kind is specified.
	ArgumentTermUnknown ArgumentTermKind = iota
	// ArgumentTermValue is a scalar value-producing positional term.
	ArgumentTermValue
	// ArgumentTermSequence is a repeated/sequence value-producing term whose owned elements preserve
	// input order.
	ArgumentTermSequence
	// ArgumentTermLiteral is a required literal grammar term and creates no value destination.
	ArgumentTermLiteral
	// ArgumentTermTokens is a bounded token-language term described by TokenLanguage syntax.
	ArgumentTermTokens
	// ArgumentTermRemainder owns the remaining positional input at the form's transfer boundary.
	ArgumentTermRemainder
	// ArgumentTermPassThrough owns pass-through positional input at the form's transfer boundary.
	ArgumentTermPassThrough
)

// Schema is the detached static semantic projection of a compiled Application. It contains
// only statically compiled information: it does not execute command providers or
// invocation-scoped value-domain providers. Commandment-produced snapshots are referentially
// closed, share no caller-mutable state with the Application or other snapshots, and may be
// freely mutated by the caller without changing application semantics. Identity-bearing
// subjects live in maps, ordered relationships live in slices, and CommandOrder is the
// canonical whole-schema command traversal order.
type Schema struct {
	EvaluatedRelease string
	Localization     Localization

	Root         CommandID
	CommandOrder []CommandID
	Commands     map[CommandID]CommandSchema

	CommandEntries   map[CommandEntryID]CommandEntrySchema
	ActionEntries    map[ActionEntryID]ActionEntrySchema
	CommandDefaults  map[CommandDefaultID]CommandDefaultSchema
	CommandProviders map[CommandProviderID]CommandProviderSchema

	Flags              map[FlagID]FlagSchema
	FlagSpellings      map[FlagSpellingID]FlagSpellingSchema
	ArgumentForms      map[ArgumentFormID]ArgumentFormSchema
	ArgumentTerms      map[ArgumentTermID]ArgumentTermSchema
	Values             map[ValueID]ValueSchema
	Facts              map[ValueID]FactSchema
	EnvironmentInputs  map[EnvironmentInputID]EnvironmentInputSchema
	ValueDomains       []ValueDomainSchema
	ValueDomainEntries map[ValueDomainEntryID]ValueDomainEntrySchema

	SignalPolicies  map[SignalPolicyID]SignalPolicySchema
	SignalEndpoints map[SignalEndpointID]SignalEndpoint
}

// CommandID identifies one command in the compiled application artifact model. Static command IDs
// remain stable across detached artifacts from the same compiled Application.
type CommandID uint32

// CommandEntryID identifies one command-entry declaration in the compiled application artifact
// model. Zero means no such relationship.
type CommandEntryID uint32

// CommandDefaultID identifies one default-command declaration in the compiled application artifact
// model. Zero means no such relationship.
type CommandDefaultID uint32

// ActionEntryID identifies one intrinsic-action entry declaration in the compiled application
// artifact model. Zero means no such relationship.
type ActionEntryID uint32

// CommandProviderID identifies one command-provider declaration in the compiled application
// artifact model. Zero means no such relationship.
type CommandProviderID uint32

// FlagID identifies one flag declaration in the compiled application artifact model. Zero means no
// such relationship.
type FlagID uint32

// FlagSpellingID identifies one exact flag-spelling declaration in the compiled application
// artifact model. Zero means no such relationship.
type FlagSpellingID uint32

// ArgumentFormID identifies one argument-form declaration in the compiled application artifact
// model. Zero means no such relationship.
type ArgumentFormID uint32

// ArgumentTermID identifies one argument-term declaration in the compiled application artifact
// model. Zero means no such relationship.
type ArgumentTermID uint32

// EnvironmentInputID identifies one environment-input declaration in the compiled application
// artifact model. Zero means no such relationship.
type EnvironmentInputID uint32

// ValueDomainEntryID identifies one finite-domain entry declaration in the compiled application
// artifact model. Zero means no such relationship.
type ValueDomainEntryID uint32

// GrammarProfileID identifies one grammar-profile branch in the compiled application artifact
// model. Zero also denotes declarations common to every profile branch.
type GrammarProfileID uint32

// ValueCandidateID identifies one candidate within its owning detached invocation result. It is
// result-local identity; zero means no candidate relationship.
type ValueCandidateID uint32

// CommandEntryKind classifies how a command entry relates to its target command. Its zero value is
// the unknown/unset classification.
type CommandEntryKind uint8

const (
	// CommandEntryUnknown means no command-entry relationship kind is specified.
	CommandEntryUnknown CommandEntryKind = iota
	// CommandEntryCanonical identifies the canonical lexical entry of its target command.
	CommandEntryCanonical
	// CommandEntryAlias identifies an alternate lexical entry for its target command.
	CommandEntryAlias
	// CommandEntryShortcut identifies an entry that selects a target command and may inject declared
	// arguments/defaults.
	CommandEntryShortcut
)

// ActionEntryKind classifies the lexical shape of an intrinsic-action entry. Its zero value is the
// unknown/unset classification.
type ActionEntryKind uint8

const (
	// ActionEntryUnknown means no intrinsic-action entry kind is specified.
	ActionEntryUnknown ActionEntryKind = iota
	// ActionEntryFlag identifies an intrinsic action triggered by flag syntax.
	ActionEntryFlag
	// ActionEntryCommand identifies an intrinsic action triggered by command syntax.
	ActionEntryCommand
)

// NameMatchKind classifies the lexical name-match result. Its zero value is the unknown/unset
// classification.
type NameMatchKind uint8

const (
	// NameMatchKindUnknown means no name-match classification is available.
	NameMatchKindUnknown NameMatchKind = iota
	// NameMatchKindExact means the analyzed spelling exactly matched the declaration under its
	// normalization rules.
	NameMatchKindExact
	// NameMatchKindUniquePrefix means the spelling selected one declaration by an unambiguous accepted
	// prefix.
	NameMatchKindUniquePrefix
)

// CommandSelectionStepKind classifies the relationship that caused a command-selection step. Its
// zero value is the unknown/unset classification.
type CommandSelectionStepKind uint8

const (
	// CommandSelectionStepUnknown means no causal selection relationship is specified.
	CommandSelectionStepUnknown CommandSelectionStepKind = iota
	// CommandSelectionStepEntry records command selection through an explicit command entry.
	CommandSelectionStepEntry
	// CommandSelectionStepDefault records command selection through a default-command declaration.
	CommandSelectionStepDefault
)

// CommandSelectionStep records one causal command-selection relationship. Kind selects Entry versus
// Default; Input is the source origin when selection was input-caused, and Commands preserves the
// ordered command segment produced by the step.
type CommandSelectionStep struct {
	Kind CommandSelectionStepKind

	Entry   CommandEntryID
	Default CommandDefaultID
	Input   InputOriginID

	Commands []CommandID
}

// CommandSelection records canonical command selection: Root, Selected, canonical Path, and ordered
// causal Steps. It does not duplicate lexical spelling or match observations.
type CommandSelection struct {
	Root     CommandID
	Selected CommandID
	Path     []CommandID
	Steps    []CommandSelectionStep
}

// DefaultCommandSelectionOutcome classifies the aggregate default-command-selection outcome. Its
// zero value is the unknown/unset classification.
type DefaultCommandSelectionOutcome uint8

const (
	// DefaultCommandSelectionUnknown means the aggregate selection state is unavailable.
	DefaultCommandSelectionUnknown DefaultCommandSelectionOutcome = iota
	// DefaultCommandSelectionNotEvaluated means default selection was bypassed or had not been
	// semantically evaluated.
	DefaultCommandSelectionNotEvaluated
	// DefaultCommandSelectionNone means default selection was evaluated and no default applied.
	DefaultCommandSelectionNone
	// DefaultCommandSelectionSelected means exactly one applicable default selected a command.
	DefaultCommandSelectionSelected
	// DefaultCommandSelectionAmbiguous means multiple applicable defaults prevented a unique
	// selection.
	DefaultCommandSelectionAmbiguous
	// DefaultCommandSelectionBlocked means otherwise relevant default selection was prevented by
	// blocking semantics.
	DefaultCommandSelectionBlocked
)

// DefaultCommandSelection is the aggregate result of default-child selection at one
// command-selection point. Status distinguishes not evaluated from evaluated-with-none, selected,
// ambiguous, and blocked outcomes; the ID slices preserve candidate/applicable/blocked order.
type DefaultCommandSelection struct {
	Outcome DefaultCommandSelectionOutcome

	Candidates []CommandDefaultID
	Applicable []CommandDefaultID
	Blocked    []CommandDefaultID
	Selected   CommandDefaultID
}

// ValueID identifies one value destination in the compiled application artifact model. Zero means
// no such relationship.
type ValueID uint32

// ValueKind classifies the public value representation shape. Its zero value is the unknown/unset
// classification.
type ValueKind uint8

const (
	// ValueKindUnknown means no public value representation kind is specified.
	ValueKindUnknown ValueKind = iota
	// ValueKindBool represents a boolean value.
	ValueKindBool
	// ValueKindCount represents a count value.
	ValueKindCount
	// ValueKindInt represents a signed integer value.
	ValueKindInt
	// ValueKindUint represents an unsigned integer value.
	ValueKindUint
	// ValueKindString represents a string value.
	ValueKindString
	// ValueKindSequence represents an ordered sequence value.
	ValueKindSequence
	// ValueKindKeyed represents a keyed value with structural key/value entries.
	ValueKindKeyed
	// ValueKindCustom represents a named application-defined semantic value type.
	ValueKindCustom
)

// ValueType describes the public semantic type of a destination or literal. Kind selects the
// built-in shape; Name distinguishes named/custom public types when the kind requires it.
type ValueType struct {
	Kind ValueKind
	Name string
}

// DeclarationKind classifies the exact typed identity carried by DeclarationIdentity. Its zero
// value is the unknown/unset classification.
type DeclarationKind uint8

const (
	// DeclarationKindUnknown means DeclarationIdentity carries no exact declaration kind.
	DeclarationKindUnknown DeclarationKind = iota
	// DeclarationKindCommandEntry means DeclarationIdentity carries a CommandEntryID.
	DeclarationKindCommandEntry
	// DeclarationKindActionEntry means DeclarationIdentity carries an ActionEntryID.
	DeclarationKindActionEntry
	// DeclarationKindFlag means DeclarationIdentity carries a FlagID.
	DeclarationKindFlag
	// DeclarationKindFlagSpelling means DeclarationIdentity carries a FlagSpellingID.
	DeclarationKindFlagSpelling
	// DeclarationKindArgumentForm means DeclarationIdentity carries an ArgumentFormID.
	DeclarationKindArgumentForm
	// DeclarationKindArgumentTerm means DeclarationIdentity carries an ArgumentTermID.
	DeclarationKindArgumentTerm
	// DeclarationKindEnvironmentInput means DeclarationIdentity carries an EnvironmentInputID.
	DeclarationKindEnvironmentInput
	// DeclarationKindValueDomainEntry means DeclarationIdentity carries a ValueDomainEntryID.
	DeclarationKindValueDomainEntry
	// DeclarationKindCommand means DeclarationIdentity carries a CommandID.
	DeclarationKindCommand
)

// DeclarationIdentity is the closed heterogeneous carrier for exact declaration identity. It is not
// a universal declaration-ID namespace: exactly one typed ID field is nonzero and must agree with
// Kind. Presentation strings never substitute for this machine identity.
type DeclarationIdentity struct {
	Kind DeclarationKind

	Command      CommandID
	CommandEntry CommandEntryID
	ActionEntry  ActionEntryID

	Flag         FlagID
	FlagSpelling FlagSpellingID

	ArgumentForm ArgumentFormID
	ArgumentTerm ArgumentTermID

	EnvironmentInput EnvironmentInputID
	ValueDomainEntry ValueDomainEntryID
}

// DeclarationReplacement describes one direct source-to-target migration relationship. Target is
// authoritative exact identity; Presentation is derived display text and is not replacement
// authority. TargetDeprecatedDuringMigration is the relationship-specific static fact about the
// target during the source migration interval; chains remain ordinary direct relationships.
type DeclarationReplacement struct {
	Target DeclarationIdentity

	Presentation string

	TargetDeprecatedDuringMigration bool
}

// Reachability classifies a static reachability conclusion independently of lifecycle. Its zero
// value is the unknown/unset classification.
type Reachability uint8

const (
	// ReachabilityUnknown means static analysis has no reachability conclusion for the effect.
	ReachabilityUnknown Reachability = iota
	// ReachabilityReachable means the semantic effect is statically reachable.
	ReachabilityReachable
	// ReachabilityUnreachable means the semantic effect is independently impossible.
	ReachabilityUnreachable
	// ReachabilityInherited means the effect is unreachable only because an enclosing semantic effect
	// is unreachable.
	ReachabilityInherited
)

// CommandDialect describes the effective lexical grammar attached to a command or grammar profile,
// including matching, normalization, flag placement syntax, end-of-options markers, and
// response-file recognition.
type CommandDialect struct {
	FlagOrder FlagOrder

	ShortFlagClustering ShortFlagClustering

	FlagMatching    NameMatching
	CommandMatching NameMatching

	FlagNormalization    NameNormalization
	CommandNormalization NameNormalization

	DashlessFlags DashlessFlagMode

	EndOfOptions  []string
	ResponseFiles bool
}

// GrammarProfile describes one statically compiled mutually exclusive grammar branch. A
// profile changes Commandment grammar/declarations for its command without changing that
// command's semantic identity, handler, middleware, lifecycle, or presentation identity.
type GrammarProfile struct {
	ID GrammarProfileID

	Value  string
	Origin DeclarationOrigin

	Reachability Reachability
	Dialect      CommandDialect
}

// GrammarProfileSet describes a command's finite grammar-profile switch. Selector chooses among
// Cases, Default is the fallback branch, Profiles resolves branch IDs, and DefaultRedundant records
// the static fact that the default can never affect selection. Zero profile IDs on declarations
// mean common to every branch.
type GrammarProfileSet struct {
	Selector ValueID

	Cases    []GrammarProfileID
	Default  GrammarProfileID
	Profiles map[GrammarProfileID]GrammarProfile

	DefaultRedundant bool
}

// HelpCommandGroupEntryKind classifies the declaration relationship stored in a help command-group
// entry. Its zero value is the unknown/unset classification.
type HelpCommandGroupEntryKind uint8

const (
	// HelpCommandGroupEntryUnknown means no help-group entry relationship is specified.
	HelpCommandGroupEntryUnknown HelpCommandGroupEntryKind = iota
	// HelpCommandGroupEntryCommandEntry relates the help row to an exact CommandEntry declaration.
	HelpCommandGroupEntryCommandEntry
	// HelpCommandGroupEntryCommandProvider relates the help row to a command-provider slot; provider
	// output itself is not materialized by Schema.
	HelpCommandGroupEntryCommandProvider
)

// HelpCommandGroupEntrySchema is one ordered member of a static command help group. Kind determines
// whether CommandEntry or Provider is authoritative; Profile records profile ownership.
type HelpCommandGroupEntrySchema struct {
	Kind HelpCommandGroupEntryKind

	CommandEntry CommandEntryID
	Provider     CommandProviderID
	Profile      GrammarProfileID
}

// HelpCommandGroupSchema describes one static command help group. Entries preserve the group's
// deterministic declaration/rendering relationship order.
type HelpCommandGroupSchema struct {
	Kind    HelpGroupKind
	Title   string
	Order   HelpGroupOrder
	Profile GrammarProfileID
	Entries []HelpCommandGroupEntrySchema
}

// HelpFlagGroupSchema describes one static flag help group. Flags is the deterministic ordered
// relationship to the group's flag declarations.
type HelpFlagGroupSchema struct {
	Kind    HelpGroupKind
	Title   string
	Order   HelpGroupOrder
	Profile GrammarProfileID
	Flags   []FlagID
}

// HelpSchema is the static help-group projection owned by a command. It records grouping and
// ordering relationships without becoming a separate declaration registry.
type HelpSchema struct {
	CommandGroups []HelpCommandGroupSchema
	FlagGroups    []HelpFlagGroupSchema
}

// CommandShortcutDefaultSchema describes one value default injected by a command shortcut.
// Destination is the semantic value target, Values preserve authored order, and Sensitive
// marks the default as requiring Commandment redaction discipline.
type CommandShortcutDefaultSchema struct {
	Destination ValueID
	Spelling    string
	Values      []ValueLiteral
	Sensitive   bool
}

// CommandEntrySchema describes one exact command-entry declaration: canonical entry, alias, or
// shortcut. ID is stable for the compiled Application; Parent owns the entry. TargetPath is the
// canonical command relationship, shortcut Args/AppendArgs/Defaults preserve declared input order,
// and Replacement points from this declaration to its direct replacement. A zero-value record is
// not a published declaration.
type CommandEntrySchema struct {
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
	Defaults   []CommandShortcutDefaultSchema

	Hidden bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ActionEntrySchema describes one exact intrinsic-action entry declaration. Kind selects flag
// versus command shape, Action is the intrinsic action triggered by the entry, and Replacement
// points from this declaration to its direct replacement. A zero-value record is not a published
// declaration.
type ActionEntrySchema struct {
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

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// CommandDefaultSchema describes one static default-child selection declaration. Condition and
// Reachability describe the declaration's static applicability; aggregate runtime/default-selection
// outcomes are represented separately by DefaultCommandSelection.
type CommandDefaultSchema struct {
	ID     CommandDefaultID
	Parent CommandID
	Target CommandID
	Origin DeclarationOrigin

	DeclarationIndex int
	Profile          GrammarProfileID

	Condition    *ConditionExpression
	Reachability Reachability
}

// CommandProviderSchema describes one statically declared command-provider slot. The provider
// itself is not executed by Schema; Dependencies are the values required before materialization,
// and Profile records branch ownership. Accepted provider output receives operation-local identity
// only when a materializing operation commits it.
type CommandProviderSchema struct {
	ID     CommandProviderID
	Parent CommandID

	DeclarationIndex int
	Profile          GrammarProfileID

	Dependencies []ValueID
}

// CommandSchema is the canonical static record for one command. CanonicalEntry, Parent, Children,
// and the owner-local ID slices form the command/declaration graph; Children and declaration slices
// preserve their contractual order. Access is the statically declared execution-access projection.
// Provider output and invocation-scoped dynamic declarations are absent from Schema until another
// operation materializes them.
type CommandSchema struct {
	ID             CommandID
	CanonicalEntry CommandEntryID
	Parent         CommandID
	Children       []CommandID

	Origin DeclarationOrigin

	Name        string
	Path        []string
	Description Description
	Examples    []Example

	Hidden         bool
	Runnable       bool
	LifecycleState LifecycleState

	Dialect  *CommandDialect
	Profiles *GrammarProfileSet
	Help     HelpSchema
	Access   ExecutionAccess

	Flags         []FlagID
	Facts         []ValueID
	ArgumentForms []ArgumentFormID
	Values        []ValueID

	Conditions         []ConditionSchema
	Rules              []RuleSchema
	Advisories         []AdvisorySchema
	NoticeSuppressions []NoticeSuppressionSchema
	SignalPolicies     []SignalPolicyID

	Completion CompletionDeclarationCapabilities

	CommandEntries   []CommandEntryID
	ActionEntries    []ActionEntryID
	CommandDefaults  []CommandDefaultID
	CommandProviders []CommandProviderID
}

// FlagSpellingKind classifies one flag spelling as long, short, or exact literal syntax. Its zero
// value is the unknown/unset classification.
type FlagSpellingKind uint8

const (
	// FlagSpellingUnknown means no flag-spelling syntax kind is specified.
	FlagSpellingUnknown FlagSpellingKind = iota
	// FlagSpellingLong represents long-name flag syntax.
	FlagSpellingLong
	// FlagSpellingShort represents short-rune flag syntax.
	FlagSpellingShort
	// FlagSpellingLiteral represents an exact literal flag spelling.
	FlagSpellingLiteral
)

// FlagSpelling describes the lexical syntax of one flag spelling. FixedValue and ImplicitValue are
// detached typed literals; Sensitive marks the spelling/value relationship for safe presentation
// and redaction.
type FlagSpelling struct {
	Kind FlagSpellingKind

	Text string

	ValueMode  FlagValueMode
	ValueForms []FlagValueForm

	Sensitive bool

	FixedValue    *ValueLiteral
	ImplicitValue *ValueLiteral
}

// FlagSchema describes one exact semantic flag declaration. Spellings is the ordered relationship
// to independently identified spelling declarations; Destination identifies the value it
// contributes to. Lifecycle and Replacement belong to the flag declaration itself, while
// spelling-local lifecycle/replacement lives on FlagSpellingSchema.
type FlagSchema struct {
	ID      FlagID
	Command CommandID
	Origin  DeclarationOrigin
	Profile GrammarProfileID

	Destination ValueID
	Spellings   []FlagSpellingID

	ValueName string

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

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// FlagSpellingSchema describes one exact spelling declaration of a flag. ID independently
// identifies the spelling, Flag identifies its owning semantic flag, and Replacement is
// deliberately spelling-local and therefore narrower than flag replacement.
type FlagSpellingSchema struct {
	ID     FlagSpellingID
	Flag   FlagID
	Origin DeclarationOrigin

	Spelling FlagSpelling
	Hidden   bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// FactSchema describes a statically declared fact destination. HasResolver distinguishes
// unresolved-required facts from resolver-backed facts; Dependencies are the resolver's declared
// value dependencies, and ResolverSensitive records sensitivity of resolver-produced data.
type FactSchema struct {
	ID ValueID

	HasResolver       bool
	ResolverSensitive bool
	Dependencies      []ValueID
}

// TokenLanguageSyntax is the detached structural syntax description for a bounded token language.
// Fragments preserve the authored fragment order.
type TokenLanguageSyntax struct {
	Fragments []string
}

// ArgumentFormSchema describes one exact positional argument form. Terms is the ordered ownership
// grammar for that form; Reachability is static analysis separate from lifecycle, and Replacement
// is the declaration's direct migration relationship.
type ArgumentFormSchema struct {
	ID      ArgumentFormID
	Command CommandID
	Origin  DeclarationOrigin
	Profile GrammarProfileID

	Terms []ArgumentTermID

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
	Reachability   Reachability
}

// ArgumentTermSchema describes one exact positional term within an argument form. Kind selects
// its ownership grammar, Destination is present only for value-producing terms, Count and the
// grammar booleans describe token ownership, and Sensitive/SourceSensitive preserve
// Commandment redaction requirements. TokenLanguage and Completion are optional capability
// projections for the term.
type ArgumentTermSchema struct {
	ID     ArgumentTermID
	Form   ArgumentFormID
	Origin DeclarationOrigin

	Kind        ArgumentTermKind
	Destination ValueID

	Name    string
	Literal string

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

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ValueDestinationKind classifies a value destination by storage/production role. Its zero value is
// the unknown/unset classification.
type ValueDestinationKind uint8

const (
	// ValueDestinationUnknown means no destination role is specified.
	ValueDestinationUnknown ValueDestinationKind = iota
	// ValueDestinationScalar is a scalar destination populated directly by contributions.
	ValueDestinationScalar
	// ValueDestinationSequence is an ordered sequence destination whose contributions compose in
	// semantic order.
	ValueDestinationSequence
	// ValueDestinationKeyed is a keyed destination with structural key/value contribution semantics.
	ValueDestinationKeyed
	// ValueDestinationOpaque is an application-defined destination without a public ValueType
	// representation.
	ValueDestinationOpaque
	// ValueDestinationDerived is a required derived destination computed from declared dependencies.
	ValueDestinationDerived
	// ValueDestinationOptionalDerived is a derived destination that may validly remain absent.
	ValueDestinationOptionalDerived
	// ValueDestinationFactRequired is a fact that must be supplied by the applicable invocation
	// contract.
	ValueDestinationFactRequired
	// ValueDestinationFactResolved is a typed invocation fact established by a declared resolver and dependencies.
	ValueDestinationFactResolved
)

// ValueSchema describes one statically declared destination and its production contract. Kind
// distinguishes scalar/sequence/keyed/opaque/derived/fact roles; HasType separates
// destinations with a public ValueType from opaque destinations. EnvironmentInputs and
// Dependencies are identity relationships. The Has* fields preserve absence versus an explicit
// zero/empty default, and sensitivity flags describe Commandment redaction of the
// corresponding production path.
type ValueSchema struct {
	ID      ValueID
	Command CommandID
	Origin  DeclarationOrigin
	Profile GrammarProfileID

	Kind    ValueDestinationKind
	HasType bool
	Type    ValueType

	Required  bool
	Derived   bool
	Sensitive bool

	EnvironmentInputs []EnvironmentInputID

	HasStaticDefault   bool
	StaticDefault      ValueStaticDefault
	HasComputedDefault bool
	HasInitialValue    bool

	ComputedDefaultSensitive bool
	DerivationSensitive      bool
	Dependencies             []ValueID
}

// EnvironmentInputSchema describes one host-environment contribution declaration. Name or prefix
// matching contributes to Destination using Contribution and optional Delimiter; Sensitive marks
// the host value as sensitive. It is static declaration data only and does not read the process
// environment.
type EnvironmentInputSchema struct {
	ID          EnvironmentInputID
	Command     CommandID
	Destination ValueID
	Origin      DeclarationOrigin
	Profile     GrammarProfileID

	Name   string
	Prefix bool

	Contribution ValueCandidateContribution
	Delimiter    string
	Sensitive    bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ValueDomainEntryKind classifies a finite-domain entry as canonical or alias. Its zero value is
// the unknown/unset classification.
type ValueDomainEntryKind uint8

const (
	// ValueDomainEntryUnknown means no finite-domain entry kind is specified.
	ValueDomainEntryUnknown ValueDomainEntryKind = iota
	// ValueDomainEntryCanonical is a canonical finite-domain value and may own description metadata.
	ValueDomainEntryCanonical
	// ValueDomainEntryAlias is an alternate name whose Canonical relationship identifies the canonical
	// entry.
	ValueDomainEntryAlias
)

// ValueDomainUseKind classifies which component of a destination consumes a finite value domain.
// Its zero value is the unknown/unset classification.
type ValueDomainUseKind uint8

const (
	// ValueDomainUseUnknown means no finite-domain consumer role is specified.
	ValueDomainUseUnknown ValueDomainUseKind = iota
	// ValueDomainUseDestination applies the domain to the destination value itself.
	ValueDomainUseDestination
	// ValueDomainUseSequenceElement applies the domain to each ordered sequence element.
	ValueDomainUseSequenceElement
	// ValueDomainUseMapKey applies the domain to keyed-value keys.
	ValueDomainUseMapKey
	// ValueDomainUseMapValue applies the domain to keyed-value values.
	ValueDomainUseMapValue
)

// ValueDomainUse identifies one semantic use of a finite value domain. Kind determines whether
// Destination itself, a sequence element, map key, or map value consumes the domain.
type ValueDomainUse struct {
	Destination ValueID
	Kind        ValueDomainUseKind
}

// ValueDomainSchema describes one static finite value domain without inventing a domain ID. Uses
// identifies every semantic consumer and Entries is the deterministic ordered relationship to exact
// ValueDomainEntryID declarations.
type ValueDomainSchema struct {
	Uses    []ValueDomainUse
	Entries []ValueDomainEntryID
}

// ValueDomainEntrySchema describes one exact finite-domain value or alias. Canonical entries own
// Description; aliases point to their canonical entry by ID and own no independent description.
// Hidden is presentation state, not availability, and Replacement is the direct declaration
// migration relationship.
type ValueDomainEntrySchema struct {
	ID     ValueDomainEntryID
	Kind   ValueDomainEntryKind
	Origin DeclarationOrigin

	Name      string
	Canonical ValueDomainEntryID

	Description Description
	Hidden      bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ConditionEffect classifies whether a condition controls availability or visibility. Its zero
// value is the unknown/unset classification.
type ConditionEffect uint8

const (
	// ConditionEffectUnknown means no condition effect is specified.
	ConditionEffectUnknown ConditionEffect = iota
	// ConditionEffectAvailability controls whether the target declaration can participate
	// semantically.
	ConditionEffectAvailability
	// ConditionEffectVisibility controls target presentation visibility without substituting for
	// availability.
	ConditionEffectVisibility
)

// ConditionExpressionKind classifies the operation represented by a ConditionExpression node. Its
// zero value is the unknown/unset classification.
type ConditionExpressionKind uint8

const (
	// ConditionExpressionUnknown means no condition operation is specified.
	ConditionExpressionUnknown ConditionExpressionKind = iota
	// ConditionExpressionResolved tests whether Destination has resolved semantic state.
	ConditionExpressionResolved
	// ConditionExpressionEqual compares Destination with the declared operand.
	ConditionExpressionEqual
	// ConditionExpressionNotEqual tests inequality with the declared operand.
	ConditionExpressionNotEqual
	// ConditionExpressionIn tests membership in the declared operand set.
	ConditionExpressionIn
	// ConditionExpressionNotIn tests non-membership in the declared operand set.
	ConditionExpressionNotIn
	// ConditionExpressionGreaterThan tests Destination against an exclusive lower bound.
	ConditionExpressionGreaterThan
	// ConditionExpressionAtLeast tests Destination against an inclusive lower bound.
	ConditionExpressionAtLeast
	// ConditionExpressionLessThan tests Destination against an exclusive upper bound.
	ConditionExpressionLessThan
	// ConditionExpressionAtMost tests Destination against an inclusive upper bound.
	ConditionExpressionAtMost
	// ConditionExpressionMatch tests Destination against Pattern.
	ConditionExpressionMatch
	// ConditionExpressionContributedBy tests Destination provenance against Sources.
	ConditionExpressionContributedBy
	// ConditionExpressionAll requires every child condition to hold.
	ConditionExpressionAll
	// ConditionExpressionAny requires at least one child condition to hold.
	ConditionExpressionAny
	// ConditionExpressionNot negates its child condition.
	ConditionExpressionNot
)

// ValueLiteral is a detached typed literal used by static semantic artifacts. Sensitive marks
// content requiring Commandment redaction discipline; Redacted states that Text is already a
// safe redacted representation rather than the underlying sensitive value.
type ValueLiteral struct {
	Type ValueType
	Text string

	Sensitive bool
	Redacted  bool
}

// ValueStaticDefault is the detached structural representation of an explicit static default.
// HasRepresentation distinguishes a representable default from an intentionally unavailable
// representation; Scalar, Sequence, and Keyed are mutually shaped by destination type. Sensitive
// and Redacted preserve safe-display state.
type ValueStaticDefault struct {
	Sensitive         bool
	HasRepresentation bool
	Redacted          bool

	Scalar   *ValueLiteral
	Sequence *ValueSequenceLiteral
	Keyed    *ValueKeyedLiteral
}

// ValueSequenceLiteral is an ordered detached sequence literal. Elements preserves semantic element
// order.
type ValueSequenceLiteral struct {
	ElementType ValueType
	Elements    []ValueLiteral
}

// ValueKeyedLiteral is a detached keyed literal with explicit key and value types. Entries
// preserves deterministic semantic entry order rather than map iteration order.
type ValueKeyedLiteral struct {
	KeyType   ValueType
	ValueType ValueType
	Entries   []ValueKeyedEntry
}

// ValueKeyedEntry is one ordered key/value pair in a ValueKeyedLiteral.
type ValueKeyedEntry struct {
	Key   ValueLiteral
	Value ValueLiteral
}

// ConditionExpression is a detached condition tree. Kind determines which of Destination, Pattern,
// Operands, Sources, or Children are meaningful. Sensitive marks the condition as containing or
// depending on sensitive semantic data.
type ConditionExpression struct {
	Kind          ConditionExpressionKind
	Destination   ValueID
	Pattern       string
	Operands      []ValueLiteral
	ProducerKinds []ValueProducerKind
	Children      []ConditionExpression

	Sensitive bool
}

// ConditionSchema attaches one availability or visibility condition to an exact
// DeclarationIdentity. Reachability is the static conclusion for this condition effect and remains
// distinct from declaration lifecycle.
type ConditionSchema struct {
	Target DeclarationIdentity
	Effect ConditionEffect

	Expression   ConditionExpression
	Reachability Reachability
	Profile      GrammarProfileID
}

// RuleKind classifies the built-in or custom validation semantics of a RuleSchema. Its zero value
// is the unknown/unset classification.
type RuleKind uint8

func (kind RuleKind) valid() bool { return kind >= RuleCustom && kind <= RuleAtLeastOne }

const (
	// RuleUnknown means no rule semantics are specified.
	RuleUnknown RuleKind = iota
	// RuleCustom means validation is performed by the declared custom rule callback.
	RuleCustom
	// RuleRequire means the subject is required.
	RuleRequire
	// RuleForbid means the subject is forbidden.
	RuleForbid
	// RuleNotEmpty means the subject must not be empty.
	RuleNotEmpty
	// RuleIn means the subject must be in the operand set.
	RuleIn
	// RuleNotIn means the subject must not be in the operand set.
	RuleNotIn
	// RuleEqual means the subject must equal the operand.
	RuleEqual
	// RuleNotEqual means the subject must differ from the operand.
	RuleNotEqual
	// RuleBetween means the subject must lie within the declared bounds.
	RuleBetween
	// RuleAtLeast means the subject must be at least the lower bound.
	RuleAtLeast
	// RuleAtMost means the subject must be at most the upper bound.
	RuleAtMost
	// RuleGreaterThan means the subject must exceed the operand.
	RuleGreaterThan
	// RuleLessThan means the subject must be below the operand.
	RuleLessThan
	// RuleMatch means the subject must match Pattern.
	RuleMatch
	// RuleItemCount means a sequence-like subject must satisfy the inclusive Count range.
	RuleItemCount
	// RuleContains means the subject must contain the operand.
	RuleContains
	// RuleEachIn means every element must belong to the operand set.
	RuleEachIn
	// RuleUnique means the subject elements must be unique.
	RuleUnique
	// RuleEntryCount means a keyed subject must satisfy the inclusive Count range.
	RuleEntryCount
	// RuleHasKeys means the keyed subject must contain the required keys.
	RuleHasKeys
	// RuleKeysIn means every key must belong to the operand set.
	RuleKeysIn
	// RuleSubsetOf means the subject set must be a subset of the operand set.
	RuleSubsetOf
	// RuleDisjoint means the subject set must be disjoint from the operand set.
	RuleDisjoint
	// RuleKeysSubsetOf means the keyed subject's keys must be a subset of the operand set.
	RuleKeysSubsetOf
	// RuleKeysDisjoint means the keyed subject's keys must be disjoint from the operand set.
	RuleKeysDisjoint
	// RuleExactlyOne means exactly one subject in the declared group must satisfy the group condition.
	RuleExactlyOne
	// RuleAllOrNone means either all grouped subjects or none of them must satisfy the group
	// condition.
	RuleAllOrNone
	// RuleAtMostOne means no more than one grouped subject may satisfy the group condition.
	RuleAtMostOne
	// RuleAtLeastOne means at least one grouped subject must satisfy the group condition.
	RuleAtLeastOne
)

// RuleOperand is one detached rule operand. Literal is present when the rule has a representable
// operand; Sensitive requires safe presentation of that operand.
type RuleOperand struct {
	Sensitive bool
	Literal   *ValueLiteral
}

// RuleSchema is the static declaration projection of one validation rule. Subjects and Dependencies
// are semantic destination relationships; Operands, Pattern, and Count are kind-specific rule
// metadata. Activation and Reachability describe conditional/static applicability, and Sensitive
// governs safe rule presentation.
type RuleSchema struct {
	ID               RuleID
	DeclarationIndex int
	Profile          GrammarProfileID

	Kind RuleKind
	Code RuleCode

	Subjects     []ValueID
	Dependencies []ValueID
	Operands     []RuleOperand
	Pattern      string
	Count        CountRange

	Activation   *ConditionExpression
	Reachability Reachability
	Sensitive    bool
}

// AdvisoryKind classifies how an advisory is produced or activated. Its zero value is the
// unknown/unset classification.
type AdvisoryKind uint8

func (kind AdvisoryKind) valid() bool {
	return kind >= AdvisoryKindAlways && kind <= AdvisoryKindDetector
}

const (
	// AdvisoryKindUnknown means no advisory production kind is specified.
	AdvisoryKindUnknown AdvisoryKind = iota
	// AdvisoryKindAlways is an unconditional declared advisory.
	AdvisoryKindAlways
	// AdvisoryKindConditional is a declared advisory gated by Activation.
	AdvisoryKindConditional
	// AdvisoryKindDetector is an advisory produced by a detector during a materializing analysis
	// operation; Schema records the declaration without executing it.
	AdvisoryKindDetector
)

// AdvisorySchema is the static declaration projection of one advisory. It embeds the advisory's
// public presentation/identity data and adds owning command, declaration/profile ownership,
// activation, reachability, subjects, and dependencies. Detector advisories record dependencies
// without executing their detector in Schema.
type AdvisorySchema struct {
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

// NoticeSuppressionSchema describes one static notice-suppression declaration. Match identifies the
// structured notice class/subject to suppress, Condition gates suppression, and Reachability
// records the static applicability conclusion; Schema does not deliver or suppress runtime notices.
type NoticeSuppressionSchema struct {
	Command          CommandID
	Origin           DeclarationOrigin
	DeclarationIndex int
	Profile          GrammarProfileID

	Match     NoticeMatch
	Condition ConditionExpression

	Reachability Reachability
}

// SignalPolicyID identifies one signal-policy declaration in the compiled application artifact
// model. Zero means no such relationship.
type SignalPolicyID uint32

// SignalCounterID identifies one signal occurrence counter in the compiled application artifact
// model. Zero means no such relationship.
type SignalCounterID uint32

// SignalEndpointID identifies one signal endpoint in the compiled application artifact model. Zero
// means no such relationship.
type SignalEndpointID uint32

// SignalEndpoint describes one statically declared application signal-delivery endpoint. Policies
// lists policies sharing the endpoint, and Capacity is endpoint-local pending application-delivery
// capacity; it does not describe source, controller, host, or callback buffering.
type SignalEndpoint struct {
	ID       SignalEndpointID
	Command  CommandID
	Origin   DeclarationOrigin
	Capacity int
	Policies []SignalPolicyID
}

// SignalSelectorKind classifies the signal set selected by a policy. Its zero value is the
// unknown/unset classification.
type SignalSelectorKind uint8

const (
	// SignalSelectorUnknown means no signal selector is specified.
	SignalSelectorUnknown SignalSelectorKind = iota
	// SignalSelectorInterrupt selects the framework's interrupt signal class.
	SignalSelectorInterrupt
	// SignalSelectorTerminate selects the framework's termination signal class.
	SignalSelectorTerminate
	// SignalSelectorListed selects the explicitly listed signal set, subject to selector exclusions.
	SignalSelectorListed
	// SignalSelectorAsynchronous selects the framework's supported asynchronous signal class.
	SignalSelectorAsynchronous
	// SignalSelectorShutdown selects the framework's shutdown signal class.
	SignalSelectorShutdown
)

// SignalSelector is the detached match predicate for a signal policy. Kind selects the selector
// family; inclusion/exclusion slices constrain signals, origins, senders, and phases without
// subscribing to a host source.
type SignalSelector struct {
	Kind SignalSelectorKind

	Signals         []Signal
	ExcludedSignals []Signal

	Origins         []SignalOrigin
	ExcludedOrigins []SignalOrigin
	ExcludedSender  string

	Phases []SignalPhase
}

// SignalOccurrenceRange is the inclusive occurrence interval in which a signal policy matches.
// Maximum uses the signal authoring contract's unbounded representation when applicable.
type SignalOccurrenceRange struct {
	Minimum int
	Maximum int
}

// SignalActionKind classifies one declared signal-policy action. Its zero value is the
// unknown/unset classification.
type SignalActionKind uint8

const (
	// SignalActionUnknown means no signal action is specified.
	SignalActionUnknown SignalActionKind = iota
	// SignalActionIgnore performs no application or host action for this policy action step.
	SignalActionIgnore
	// SignalActionCancel requests cancellation of the admitted run according to signal runtime
	// semantics.
	SignalActionCancel
	// SignalActionHandle invokes the declared application signal handler action.
	SignalActionHandle
	// SignalActionForward forwards the matched signal through the declared host-facing forwarding
	// behavior.
	SignalActionForward
	// SignalActionSend sends the configured signal, including dynamic signal selection when declared.
	SignalActionSend
	// SignalActionAbort terminates the signal action sequence with the declared abort/exit behavior.
	SignalActionAbort
	// SignalActionReraise re-raises the matched signal through the host integration when supported.
	SignalActionReraise
	// SignalActionEmit emits the signal event to the declared SignalEndpoint.
	SignalActionEmit
)

// SignalAction is one declared action in a signal policy's ordered action sequence. Kind determines
// which endpoint, signal, delay, or exit fields are meaningful. SignalRedacted and DelayRedacted
// distinguish safe redacted projections from underlying values, and Sensitive governs safe
// presentation of the action.
type SignalAction struct {
	Kind SignalActionKind
	Name string

	EndpointID SignalEndpointID

	Signal         Signal
	SignalValueID  ValueID
	SignalRedacted bool

	Delayed bool

	Delay         time.Duration
	DelayValueID  ValueID
	DelayRedacted bool

	ExitCode int

	Sensitive bool
}

// SignalPolicySchema is the static semantic projection of one signal policy. CounterID owns
// occurrence counting for the policy; Selector, Occurrence, and Aggregation define matching,
// Actions preserves execution order, and required source/host capabilities state environmental
// prerequisites. Deferred and ContinueMatching affect dispatch/matching semantics; Schema records
// them without subscribing or executing actions.
type SignalPolicySchema struct {
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

	Activation   *ConditionExpression
	Reachability Reachability
}

// CompletionDeclarationCapabilities describes completion behavior attached to a declaration.
// Dynamic and Delegated are distinct completion modes; SourceSensitive marks completion that
// depends on sensitive source material, Dependencies is the declared value dependency order, and
// Native is present only for native-shell delegation intent. The zero value means no declared
// completion capability.
type CompletionDeclarationCapabilities struct {
	Dynamic   bool
	Delegated bool

	SourceSensitive bool

	Dependencies []ValueID

	Native *CompletionNativeIntent
}
