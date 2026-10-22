package cli

import "context"

// HelpRenderer renders an already-structured HelpPage. The page contains resolved presentation and
// semantic distinctions; a renderer does not select locales, perform translation lookup, or
// recover semantics by parsing formatted text.
type HelpRenderer func(context.Context, HelpPage) (string, error)

// HelpSection identifies one configurable section of Commandment's built-in Help presentation.
type HelpSection uint8

const (
	HelpSectionOverview HelpSection = iota + 1
	HelpSectionUsage
	HelpSectionCommands
	HelpSectionArguments
	HelpSectionFlags
	HelpSectionExamples
)

// DefaultHelpConfig configures Commandment's built-in Help renderer. An empty Sections slice selects all
// standard sections. A nonempty slice selects only the named presentation sections and does not
// change the underlying Help semantic model or HelpPage.
type DefaultHelpConfig struct {
	Sections []HelpSection
}

type helpSectionSet uint8

const standardHelpSectionCount = 6

func helpSections(config DefaultHelpConfig) helpSectionSet {
	if len(config.Sections) == 0 {
		return helpSectionSet((1 << standardHelpSectionCount) - 1)
	}
	var result helpSectionSet
	for _, section := range append([]HelpSection(nil), config.Sections...) {
		if section >= HelpSectionOverview && section <= HelpSectionExamples {
			result |= 1 << (section - 1)
		}
	}
	return result
}

func (set helpSectionSet) has(section HelpSection) bool {
	if section < HelpSectionOverview || section > HelpSectionExamples {
		return false
	}
	return set&(1<<(section-1)) != 0
}

// DefaultHelp returns Commandment's built-in HelpRenderer configured by config. Configuration is
// snapshotted when the renderer is constructed.
func DefaultHelp(config DefaultHelpConfig) HelpRenderer {
	sections := helpSections(config)
	return func(ctx context.Context, page HelpPage) (string, error) {
		return renderDefaultHelp(requireContext(ctx), page, sections)
	}
}

// HelpPageKind classifies the subject of a HelpPage. Its zero value is the unknown classification.
type HelpPageKind uint8

const (
	// HelpPageUnknown is the zero/unknown page kind.
	HelpPageUnknown HelpPageKind = iota
	// HelpPageApplication identifies application-level help.
	HelpPageApplication
	// HelpPageCommand identifies help for a selected command.
	HelpPageCommand
)

// HelpGroupKind classifies the source of a rendered help group. Group kind is presentation
// metadata and does not create a command, policy, provider, localization, or declaration scope.
type HelpGroupKind uint8

const (
	// HelpGroupUnknown is the zero/unknown group kind.
	HelpGroupUnknown HelpGroupKind = iota

	// HelpGroupDeclared identifies an application-declared help group.
	HelpGroupDeclared

	// HelpGroupDefault identifies a compiler-owned default group for otherwise ungrouped entries.
	HelpGroupDefault

	// HelpGroupInherited identifies a group inherited through the effective presentation model.
	HelpGroupInherited
)

// HelpGroupClassification preserves the non-localized declaration coordinates needed to keep a
// static presentation group semantically stable across Help and Completion projection. CommandPath
// is the canonical human-readable declaration scope. Profile and SourceTitle further identify
// application-declared groups; compiler-owned default and inherited classes are distinguished by
// HelpGroupKind and group family. SourceTitle is source-language declaration text, not the
// localized rendered title.
type HelpGroupClassification struct {
	CommandPath []string
	Profile     GrammarProfileID
	SourceTitle string
}

// HelpCommandKind classifies the exact command-entry presentation represented by a HelpCommand.
type HelpCommandKind uint8

const (
	// HelpCommandUnknown is the zero/unknown command-entry kind.
	HelpCommandUnknown HelpCommandKind = iota
	// HelpCommandCanonical identifies a canonical command entry.
	HelpCommandCanonical
	// HelpCommandAlias identifies a lexical command alias.
	HelpCommandAlias
	// HelpCommandShortcut identifies a command shortcut.
	HelpCommandShortcut
	// HelpCommandAction identifies an intrinsic-action command entry.
	HelpCommandAction
)

// HelpCommandGroup is one ordered command presentation group. The group preserves its semantic
// kind, resolved title, presentation ordering policy, and the relative order of surviving command
// entries.
type HelpCommandGroup struct {
	Kind           HelpGroupKind
	Classification HelpGroupClassification
	Title          string
	Order          HelpGroupOrder
	Commands       []HelpCommand
}

// HelpFlagGroup is one ordered flag presentation group. Filtering preserves the relative order of
// surviving entries, and empty groups are not rendered.
type HelpFlagGroup struct {
	Kind           HelpGroupKind
	Classification HelpGroupClassification
	Title          string
	Order          HelpGroupOrder
	Flags          []HelpFlag
}

// HelpPage is a detached structured presentation projection of the effective semantic model for
// one help subject. It preserves selection, lifecycle and visibility-derived presentation, ordered
// command/argument/flag relationships, resolved localization, labels, and page-owned exact-node
// maps that close flag-spelling and value-domain relationships. It is presentation data, not an
// alternate declaration registry or semantic authority; caller mutation does not change the
// Application.
type HelpPage struct {
	Kind      HelpPageKind
	Selection CommandSelection

	ApplicationName string
	Name            string
	Description     Description
	Version         string
	SelectedPath    []string
	// RequestedPath preserves the exact command-entry spellings used to reach this Help subject,
	// including lexical aliases and command shortcuts.
	RequestedPath    []string
	EvaluatedRelease string
	Release          ReleaseLifecycle
	LifecycleState   LifecycleState

	Runnable bool
	Usage    []string
	Examples []Example

	CommandGroups []HelpCommandGroup
	ArgumentForms []HelpArgumentForm
	FlagGroups    []HelpFlagGroup

	FlagSpellings      map[FlagSpellingID]HelpFlagSpelling
	ValueDomainEntries map[ValueDomainEntryID]HelpValueDomainEntry

	Labels HelpLabels
	Locale Locale
}

// HelpCommand records one rendered command, alias, shortcut, or action entry with its exact entry
// kind, declaration origin, target path, resolved description, visibility/runnability, and
// lifecycle state. Entry-local hidden state does not hide the canonical target.
type HelpCommand struct {
	Kind   HelpCommandKind
	Origin DeclarationOrigin

	CommandEntry CommandEntryID
	ActionEntry  ActionEntryID
	TargetPath   []CommandID

	Name        string
	Description Description

	Hidden         bool
	Default        bool
	Runnable       bool
	HasSubcommands bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
}

// HelpArgumentForm preserves positional argument presentation in canonical form and term order;
// positional terms are not regrouped into arbitrary help groups.
type HelpArgumentForm struct {
	ID    ArgumentFormID
	Terms []HelpArgumentTerm
}

// HelpArgumentTerm is one positional term presentation record. It preserves syntax, value/count
// rules, visibility and sensitivity, lifecycle, token-language syntax, and ordered finite-domain
// uses needed by the renderer.
type HelpArgumentTerm struct {
	ID   ArgumentTermID
	Kind ArgumentTermKind

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

	TokenLanguage  *TokenLanguageSyntax
	Release        ReleaseLifecycle
	LifecycleState LifecycleState

	ValueDomainUses []HelpValueDomainUse
}

// HelpFlag is one flag or action-flag presentation record. Ordinary semantic flags reference
// ordered exact FlagSpellingID nodes in HelpPage.FlagSpellings; action-entry flags instead carry
// inline ActionSpelling syntax and no independent spelling identity. ValueDomainUses retain
// ordered finite-domain relationships.
type HelpFlag struct {
	Origin DeclarationOrigin

	Flag        FlagID
	ActionEntry ActionEntryID

	FlagSpellings  []FlagSpellingID
	ActionSpelling FlagSpelling

	ValueName   string
	Description string
	OwnerPath   []string

	Occurrences         CountRange
	ValuesPerOccurrence CountRange
	Placement           FlagPlacement

	Required       bool
	Hidden         bool
	Sensitive      bool
	LocaleSelector bool

	Release        ReleaseLifecycle
	LifecycleState LifecycleState

	ValueDomainUses []HelpValueDomainUse
}

// HelpFlagSpelling is a page-owned exact flag-spelling node used by HelpFlag relationships and
// replacement closure. Its hidden/lifecycle/replacement state belongs to the exact spelling and
// does not automatically apply to sibling spellings or the parent flag.
type HelpFlagSpelling struct {
	ID       FlagSpellingID
	Spelling FlagSpelling

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    FlagSpellingID
}

// HelpValueDomainUse records one ordered finite-domain use; Entries resolve through
// HelpPage.ValueDomainEntries.
type HelpValueDomainUse struct {
	Kind    ValueDomainUseKind
	Entries []ValueDomainEntryID
}

// HelpValueDomainEntry is a page-owned exact finite-domain node used for ordered domain
// relationships and canonical/replacement closure. Closure-only nodes do not render merely because
// the map contains them.
type HelpValueDomainEntry struct {
	ID        ValueDomainEntryID
	Kind      ValueDomainEntryKind
	Name      string
	Canonical ValueDomainEntryID

	Description string

	Release        ReleaseLifecycle
	LifecycleState LifecycleState
	Replacement    ValueDomainEntryID
}

// HelpLabels contains the complete resolved localized framework-phrase input used by Help rendering.
// Built-in and custom HelpRenderer implementations receive the same detached labels; labels are
// presentation text and do not define semantic identity.
type HelpLabels struct {
	Usage       string
	Commands    string
	Arguments   string
	Flags       string
	GlobalFlags string
	Aliases     string
	Environment string
	Default     string
	Required    string
	Optional    string
	Deprecated  string

	Examples      string
	Alias         string
	Shortcut      string
	Values        string
	Keys          string
	Syntax        string
	AliasFor      string
	Use           string
	ImplicitValue string
	FixedValue    string
}
