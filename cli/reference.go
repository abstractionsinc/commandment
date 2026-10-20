package cli

import "context"

// ReferenceVisibility selects which declarations participate in ordinary visible Reference
// relationships. Canonical records needed to close relationships may remain in the detached
// artifact even when omitted from visible relationships.
type ReferenceVisibility uint8

const (
	// ReferenceVisibilityPublic includes only declarations eligible for public visible relationships,
	// while still permitting hidden canonical closure records when required by an included
	// relationship.
	ReferenceVisibilityPublic ReferenceVisibility = iota
	// ReferenceVisibilityAll includes hidden and public declarations in ordinary visible
	// relationships, subject to release availability.
	ReferenceVisibilityAll
)

// ReferenceOrder selects the deterministic presentation order of commands in
// Reference.CommandOrder without changing semantic identity or topology.
type ReferenceOrder uint8

const (
	// ReferenceOrderDeclaration orders CommandOrder by semantic declaration order.
	ReferenceOrderDeclaration ReferenceOrder = iota
	// ReferenceOrderName traverses the command tree deterministically with sibling commands ordered
	// by name.
	ReferenceOrderName
)

// ReferenceOptions selects locale, release, visibility, and command ordering for a Reference
// projection. The zero value selects public visibility and declaration ordering; empty locale and
// release use the operation defaults. Invalid selections are rejected rather than silently
// normalized.
type ReferenceOptions struct {
	Locale     Locale
	Visibility ReferenceVisibility
	Order      ReferenceOrder
	Release    string
}

// DeclarationOrigin identifies which declaration layer established a projected declaration:
// application authoring, Commandment synthesis, or accepted provider materialization.
type DeclarationOrigin uint8

func (origin DeclarationOrigin) valid() bool {
	return origin == DeclarationOriginApplication || origin == DeclarationOriginFrontend || origin == DeclarationOriginProvider
}

const (
	// DeclarationOriginUnknown means the declaration origin is not authoritatively known.
	DeclarationOriginUnknown DeclarationOrigin = iota
	// DeclarationOriginApplication means the declaration originated in application authoring.
	DeclarationOriginApplication
	// DeclarationOriginFrontend means Commandment synthesized the declaration rather than
	// receiving it from application authoring or accepted provider output.
	DeclarationOriginFrontend
	// DeclarationOriginProvider means the declaration was committed from accepted provider output.
	DeclarationOriginProvider
)

// Reference is a detached, generation-ready static presentation of one Application for a
// selected locale and release. It does not execute providers, preserves the Schema
// identity/topology family, and may retain non-visible canonical records needed to close
// included relationships. The zero value is not a valid published snapshot.
type Reference struct {
	Locale           Locale
	EvaluatedRelease string

	Root         CommandID
	CommandOrder []CommandID
	Commands     map[CommandID]CommandReference

	CommandEntries   map[CommandEntryID]CommandEntryReference
	ActionEntries    map[ActionEntryID]ActionEntryReference
	CommandDefaults  map[CommandDefaultID]CommandDefaultReference
	CommandProviders map[CommandProviderID]CommandProviderReference

	Flags              map[FlagID]FlagReference
	FlagSpellings      map[FlagSpellingID]FlagSpellingReference
	ArgumentForms      map[ArgumentFormID]ArgumentFormReference
	ArgumentTerms      map[ArgumentTermID]ArgumentTermReference
	Values             map[ValueID]ValueReference
	Facts              map[ValueID]FactReference
	EnvironmentInputs  map[EnvironmentInputID]EnvironmentInputReference
	ValueDomains       []ValueDomainReference
	ValueDomainEntries map[ValueDomainEntryID]ValueDomainEntryReference

	SignalPolicies  map[SignalPolicyID]SignalPolicyReference
	SignalEndpoints map[SignalEndpointID]SignalEndpoint
}

// CommandShortcutDefaultReference describes one generated shortcut default in a Reference,
// including its destination, spelling, values, and sensitivity.
type CommandShortcutDefaultReference struct {
	Destination ValueID
	Spelling    string
	Values      []ValueLiteral
	Sensitive   bool
}

// CommandEntryReference is the detached static presentation of a command entry, preserving its
// stable identity, declaration position, lifecycle, target relationship, and visibility state.
type CommandEntryReference struct {
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
	Defaults   []CommandShortcutDefaultReference

	Hidden    bool
	Available bool
	Visible   bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ActionEntryReference is the detached static presentation of an action entry, preserving its
// stable identity, invocation action, spelling/presentation, lifecycle, and visibility state.
type ActionEntryReference struct {
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

// CommandDefaultReference describes one statically declared default-command relationship,
// including its parent, target, condition, profile, declaration position, and reachability.
type CommandDefaultReference struct {
	ID     CommandDefaultID
	Parent CommandID
	Target CommandID
	Origin DeclarationOrigin

	DeclarationIndex int
	Profile          GrammarProfileID

	Condition    *ConditionExpression
	Reachability Reachability
}

// CommandProviderReference describes one statically declared command provider and its value
// dependencies. Reference does not execute the provider or contain unmaterialized provider output.
type CommandProviderReference struct {
	ID     CommandProviderID
	Parent CommandID

	DeclarationIndex int
	Profile          GrammarProfileID

	Dependencies []ValueID
}

// CommandReference is the detached static presentation of one command. Its ID relationships use
// the same semantic command topology as Schema, while text and lifecycle visibility reflect the
// Reference locale/release projection.
type CommandReference struct {
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

	Dialect      *CommandDialect
	Profiles     *GrammarProfileSet
	Usage        []string
	ProfileUsage map[GrammarProfileID][]string
	Help         HelpSchema
	Access       ExecutionAccess

	Flags         []FlagID
	Facts         []ValueID
	ArgumentForms []ArgumentFormID
	Values        []ValueID

	Conditions         []ConditionSchema
	Rules              []RuleSchema
	Advisories         []AdvisoryReference
	NoticeSuppressions []NoticeSuppressionSchema
	SignalPolicies     []SignalPolicyID

	Completion CompletionDeclarationCapabilities

	CommandEntries   []CommandEntryID
	ActionEntries    []ActionEntryID
	CommandDefaults  []CommandDefaultID
	CommandProviders []CommandProviderID
}

// FlagReference is the detached static presentation of one flag, including exact spelling
// relationships, destination, lifecycle, visibility, completion, and sensitivity facts.
type FlagReference struct {
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

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// FlagSpellingReference is the exact detached spelling record for a flag. Spelling-local
// hidden/lifecycle/replacement facts remain distinct from the parent flag.
type FlagSpellingReference struct {
	ID     FlagSpellingID
	Flag   FlagID
	Origin DeclarationOrigin

	Spelling FlagSpelling
	Hidden   bool

	Available bool
	Visible   bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ArgumentFormReference is the detached static presentation of one argument form and its ordered
// term relationship, lifecycle, visibility, and reachability.
type ArgumentFormReference struct {
	ID      ArgumentFormID
	Command CommandID
	Origin  DeclarationOrigin
	Profile GrammarProfileID

	Terms []ArgumentTermID

	Available bool
	Visible   bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
	Reachability   Reachability
}

// ArgumentTermReference is the detached static presentation of one argument term, including
// destination, grammar, sensitivity, completion, lifecycle, and visibility facts.
type ArgumentTermReference struct {
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
	Available     bool
	Visible       bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    *DeclarationReplacement
}

// ValueReference is the static Reference projection of a value declaration, preserving ValueSchema
// semantics while adding release/visibility availability for the selected projection.
type ValueReference struct {
	ValueSchema

	Available bool
	Visible   bool
}

// FactReference is the static Reference projection of a fact declaration and records whether it is
// available for the selected release.
type FactReference struct {
	FactSchema

	Available bool
}

// EnvironmentInputReference is the static Reference projection of a declared environment input and
// records whether it is available for the selected release.
type EnvironmentInputReference struct {
	EnvironmentInputSchema

	Available bool
}

// ValueDomainReference is one ordered static value-domain projection. Domains have no public
// domain ID; exact entries are related by ValueDomainEntryID.
type ValueDomainReference struct {
	Uses    []ValueDomainUse
	Entries []ValueDomainEntryID
}

// ValueDomainEntryReference is the exact detached static presentation of a value-domain entry,
// including canonical-alias identity, lifecycle, replacement, and visibility.
type ValueDomainEntryReference struct {
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

// SignalPolicyReference is the static Reference projection of a signal policy, preserving its
// policy identity and requirements while adding release/visibility availability.
type SignalPolicyReference struct {
	SignalPolicySchema

	Available bool
	Visible   bool
}

// structuralPresentationNeedsValues reports whether the selected presentation
// frontier contains a visibility condition. Availability is intentionally
// excluded: grammar owns its earlier scheduling and may use only phase-legal
// early inputs/Facts.
func structuralPresentationNeedsValues(model *effectiveModel, grammar grammarResult) bool {
	selected := make(map[CommandID]struct{}, len(grammar.Selection.Path))
	for _, command := range grammar.Selection.Path {
		selected[command] = struct{}{}
	}
	for _, record := range model.ConditionPolicies() {
		if record.effect != ConditionEffectVisibility ||
			!structuralConditionActive(model, selected, record.target) {
			continue
		}
		owner := structuralConditionOwner(model, record.target)
		if structuralProfileActive(grammar.SelectedProfiles, owner, record.profile) {
			return true
		}
	}
	return false
}

// applyStructuralConditionsReference projects operation-effective structural
// availability and visibility onto a detached Reference. It never changes
// grammar ownership. Availability reuses the grammar's single-assignment
// observation when present; a declaration first demanded only for presentation
// is evaluated at that presentation frontier.
func applyStructuralConditionsReference(ctx context.Context, reference *Reference, model *effectiveModel, grammar grammarResult, invocation *invocation) bool {
	if reference == nil || invocation == nil {
		return false
	}
	selected := make(map[CommandID]struct{}, len(grammar.Selection.Path))
	for _, command := range grammar.Selection.Path {
		selected[command] = struct{}{}
	}
	availability := make(map[conditionTarget]conditionOutcome, len(grammar.Conditions))
	for _, observation := range grammar.Conditions {
		if observation.Effect == ConditionEffectAvailability {
			availability[observation.Target] = observation.Outcome
		}
	}
	complete := true
	for _, record := range model.ConditionPolicies() {
		if !structuralConditionActive(model, selected, record.target) {
			continue
		}
		owner := structuralConditionOwner(model, record.target)
		if !structuralProfileActive(grammar.SelectedProfiles, owner, record.profile) {
			continue
		}
		var outcome conditionOutcome
		switch record.effect {
		case ConditionEffectAvailability:
			if observed, ok := availability[record.target]; ok {
				outcome = observed
			} else {
				outcome = invocation.condition(ctx, grammarConditionDemand{
					Policy:  record,
					Grammar: grammar,
				})
			}
		case ConditionEffectVisibility:
			outcome = invocation.condition(ctx, grammarConditionDemand{
				Policy:  record,
				Grammar: grammar,
			})
		default:
			complete = false
			continue
		}
		switch outcome {
		case conditionOutcomeSatisfied:
		case conditionOutcomeUnsatisfied:
			applyStructuralReferenceTargetState(reference, record.target, record.effect)
		default:
			complete = false
		}
	}
	return complete
}

func applyStructuralReferenceTargetState(reference *Reference, target conditionTarget, effect ConditionEffect) {
	apply := func(available, visible *bool) {
		switch effect {
		case ConditionEffectAvailability:
			*available = false
			*visible = false
		case ConditionEffectVisibility:
			*visible = false
		default:
			return
		}
	}
	switch target.Kind() {
	case conditionTargetCommand:
		id := target.Command()
		command, ok := reference.Commands[id]
		if !ok {
			return
		}
		// CommandReference does not carry explicit Available/Visible booleans;
		// its ordinary public membership is represented by its entries. Disable
		// all lexical entries targeting the command at this projection boundary.
		for entryID, entry := range reference.CommandEntries {
			if len(entry.TargetPath) == 0 || entry.TargetPath[len(entry.TargetPath)-1] != id {
				continue
			}
			if effect == ConditionEffectAvailability {
				entry.Available = false
			}
			entry.Visible = false
			reference.CommandEntries[entryID] = entry
		}
		_ = command
	case conditionTargetFlag:
		id := target.Flag()
		flag, ok := reference.Flags[id]
		if !ok {
			return
		}
		apply(&flag.Available, &flag.Visible)
		reference.Flags[id] = flag
		for _, spellingID := range flag.Spellings {
			spelling, ok := reference.FlagSpellings[spellingID]
			if !ok {
				continue
			}
			if effect == ConditionEffectAvailability {
				spelling.Available = false
			}
			spelling.Visible = false
			reference.FlagSpellings[spellingID] = spelling
		}
	case conditionTargetArgumentForm:
		id := target.ArgumentForm()
		form, ok := reference.ArgumentForms[id]
		if !ok {
			return
		}
		apply(&form.Available, &form.Visible)
		reference.ArgumentForms[id] = form
	case conditionTargetArgumentTerm:
		id := target.ArgumentTerm()
		term, ok := reference.ArgumentTerms[id]
		if !ok {
			return
		}
		apply(&term.Available, &term.Visible)
		reference.ArgumentTerms[id] = term
	case conditionTargetUnknown:
	}
}
