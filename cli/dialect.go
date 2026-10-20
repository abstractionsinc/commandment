package cli

import "slices"

type dialectDirectiveDefinition struct {
	kind          dialectDirectiveKind
	flagOrder     FlagOrder
	clustering    ShortFlagClustering
	matching      NameMatching
	normalization NameNormalization
	dashless      DashlessFlagMode
	tokens        []string
	enabled       bool
	placement     FlagPlacement
}

func staticDialectDirectiveDefinition(value dialectDirectiveDraft) (dialectDirectiveDefinition, bool) {
	if !value.valid {
		return dialectDirectiveDefinition{}, false
	}
	directive := dialectDirectiveDefinition{
		kind:          value.kind,
		flagOrder:     value.flagOrder,
		clustering:    value.clustering,
		matching:      value.matching,
		normalization: value.normalization,
		dashless:      value.dashless,
		tokens:        value.tokens,
		enabled:       value.enabled,
		placement:     value.placement,
	}

	return directive, true
}

// FlagOrder selects whether flags must precede positional arguments or may be interspersed with
// them.
type FlagOrder uint8

func (order FlagOrder) valid() bool {
	return order == FlagOrderBeforeArguments || order == FlagOrderInterspersed
}

const (
	// FlagOrderUnknown is the unspecified flag-order value.
	FlagOrderUnknown FlagOrder = iota

	// FlagOrderBeforeArguments requires flags to appear before positional arguments in a command.
	//
	FlagOrderBeforeArguments

	// FlagOrderInterspersed allows flags and positional arguments to be interspersed.
	FlagOrderInterspersed
)

// ShortFlagClustering selects whether adjacent short flag spellings may be parsed as a cluster.
type ShortFlagClustering uint8

func (clustering ShortFlagClustering) valid() bool {
	return clustering == ShortFlagClusteringDisabled || clustering == ShortFlagClusteringEnabled
}

const (
	// ShortFlagClusteringUnknown is the unspecified short-flag clustering mode.
	ShortFlagClusteringUnknown ShortFlagClustering = iota

	// ShortFlagClusteringDisabled requires short flags to be parsed individually.
	ShortFlagClusteringDisabled

	// ShortFlagClusteringEnabled allows adjacent short flag spellings to be parsed as one cluster.
	ShortFlagClusteringEnabled
)

// NameMatching selects how command or flag names are matched after normalization.
type NameMatching uint8

func (matching NameMatching) valid() bool {
	return matching == NameMatchingExact || matching == NameMatchingUniquePrefix
}

const (
	// NameMatchingUnknown is the unspecified name-matching mode.
	NameMatchingUnknown NameMatching = iota

	// NameMatchingExact requires the normalized input to equal the normalized declared name.
	NameMatchingExact

	// NameMatchingUniquePrefix accepts a normalized input prefix only when it selects one declared
	// name uniquely.
	NameMatchingUniquePrefix
)

// NameNormalization selects how command or flag names are normalized before matching.
type NameNormalization uint8

func (normalization NameNormalization) valid() bool {
	return normalization == NameNormalizationExact || normalization == NameNormalizationASCIICaseFold
}

const (
	// NameNormalizationUnknown is the unspecified name-normalization mode.
	NameNormalizationUnknown NameNormalization = iota
	// NameNormalizationExact preserves declared name text for matching.
	NameNormalizationExact
	// NameNormalizationASCIICaseFold matches names after ASCII case folding.
	NameNormalizationASCIICaseFold
)

// DashlessFlagMode selects whether command grammar recognizes dashless short-flag clusters.
type DashlessFlagMode uint8

func (mode DashlessFlagMode) valid() bool {
	return mode == DashlessFlagModeDisabled || mode == DashlessFlagModeInitialCluster
}

const (
	// DashlessFlagModeUnknown is the unspecified dashless-flag mode.
	DashlessFlagModeUnknown DashlessFlagMode = iota

	// DashlessFlagModeDisabled disables dashless short-flag recognition.
	DashlessFlagModeDisabled

	// DashlessFlagModeInitialCluster allows an initial cluster of short flags without the usual
	// leading dash.
	DashlessFlagModeInitialCluster
)

// FlagPlacement selects whether a flag belongs only to its declaring command or is visible
// throughout that command subtree.
type FlagPlacement uint8

const (
	// FlagPlacementUnknown is the unspecified flag-placement value.
	FlagPlacementUnknown FlagPlacement = iota

	// FlagPlacementCommand limits the flag spelling to its declaring command.
	FlagPlacementCommand

	// FlagPlacementSubtree makes the flag spelling available throughout the declaring command subtree.
	FlagPlacementSubtree
)

// Dialect declares command grammar such as flag ordering, name matching and normalization,
// end-of-options tokens, response files, and default flag placement. It is valid only during the
// owning command declaration callback; its zero value and retained, concurrent, or post-callback
// use are invalid.
type Dialect struct {
	session *declarationSession
	profile grammarProfileCaseRef
}

// FlagOrder sets the command dialect's positional relationship between flags and arguments.
func (dialect *Dialect) FlagOrder(order FlagOrder) {
	directive := dialectDirectiveDraft{
		kind:      dialectDirectiveFlagOrder,
		flagOrder: order,
		valid:     true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// ShortFlagClustering sets whether short flag spellings may cluster.
func (dialect *Dialect) ShortFlagClustering(clustering ShortFlagClustering) {
	directive := dialectDirectiveDraft{
		kind:       dialectDirectiveShortFlagClustering,
		clustering: clustering,
		valid:      true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// FlagMatching sets how normalized flag names are matched.
func (dialect *Dialect) FlagMatching(matching NameMatching) {
	directive := dialectDirectiveDraft{
		kind:     dialectDirectiveFlagMatching,
		matching: matching,
		valid:    true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// CommandMatching sets how normalized command names are matched.
func (dialect *Dialect) CommandMatching(matching NameMatching) {
	directive := dialectDirectiveDraft{
		kind:     dialectDirectiveCommandMatching,
		matching: matching,
		valid:    true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// FlagNormalization sets name normalization for flags before matching.
func (dialect *Dialect) FlagNormalization(normalization NameNormalization) {
	directive := dialectDirectiveDraft{
		kind:          dialectDirectiveFlagNormalization,
		normalization: normalization,
		valid:         true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// CommandNormalization sets name normalization for commands before matching.
func (dialect *Dialect) CommandNormalization(normalization NameNormalization) {
	directive := dialectDirectiveDraft{
		kind:          dialectDirectiveCommandNormalization,
		normalization: normalization,
		valid:         true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// DashlessFlags sets the command's dashless short-flag policy.
func (dialect *Dialect) DashlessFlags(mode DashlessFlagMode) {
	directive := dialectDirectiveDraft{
		kind:     dialectDirectiveDashlessFlags,
		dashless: mode,
		valid:    true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// EndOfOptions replaces the dialect's recognized end-of-options tokens with tokens in the supplied
// order.
func (dialect *Dialect) EndOfOptions(tokens ...string) {
	directive := dialectDirectiveDraft{
		kind:   dialectDirectiveEndOfOptions,
		tokens: append([]string(nil), tokens...),
		valid:  true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// ResponseFiles enables or disables response-file expansion for the command dialect.
func (dialect *Dialect) ResponseFiles(enabled bool) {
	directive := dialectDirectiveDraft{
		kind:    dialectDirectiveResponseFiles,
		enabled: enabled,
		valid:   true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

// DefaultFlagPlacement sets the placement used by flags that do not declare an explicit placement
// option.
func (dialect *Dialect) DefaultFlagPlacement(placement FlagPlacement) {
	directive := dialectDirectiveDraft{
		kind:      dialectDirectiveDefaultFlagPlacement,
		placement: placement,
		valid:     true,
	}
	stageDialectDirective(dialect.session, dialect.profile, directive)
}

type dialectDirectiveKind uint8

const (
	dialectDirectiveInvalid dialectDirectiveKind = iota
	dialectDirectiveFlagOrder
	dialectDirectiveShortFlagClustering
	dialectDirectiveFlagMatching
	dialectDirectiveCommandMatching
	dialectDirectiveFlagNormalization
	dialectDirectiveCommandNormalization
	dialectDirectiveDashlessFlags
	dialectDirectiveEndOfOptions
	dialectDirectiveResponseFiles
	dialectDirectiveDefaultFlagPlacement
)

func stageDialectDirective(session *declarationSession, profile grammarProfileCaseRef, directive dialectDirectiveDraft) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		record := dialectDirectiveDraft{
			position:      draft.nextPosition(),
			command:       command,
			profile:       profile,
			kind:          directive.kind,
			flagOrder:     directive.flagOrder,
			clustering:    directive.clustering,
			matching:      directive.matching,
			normalization: directive.normalization,
			dashless:      directive.dashless,
			tokens:        append([]string(nil), directive.tokens...),
			enabled:       directive.enabled,
			placement:     directive.placement,
			valid:         directive.valid,
		}
		draft.dialectDirectives.append(draft, record)
	})
}

func applyStaticDialectDirectives(initial dialectInput, input declarationPlan, source *declarationDraft, commandIndex, profileIndex int) (dialectInput, error) {
	result := initial
	endOfOptionsConfigured := false
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordDialectDirective ||
			record.commandIndex != commandIndex || record.profileIndex != profileIndex {
			continue
		}
		directive, ok := staticDialectDirectiveDefinition(source.dialectDirectives.entries[record.familyIndex].value)
		if !ok {
			return dialectInput{}, errStaticDeclarationRejected
		}
		result, endOfOptionsConfigured, ok = applyDialectDirective(result, directive, endOfOptionsConfigured)
		if !ok {
			return dialectInput{}, errStaticDeclarationRejected
		}
	}
	if !validSemanticDialectInput(result) {
		return dialectInput{}, errStaticDeclarationRejected
	}
	return result, nil
}

// NameNormalization is the private lexical normalization policy used by
// compiled grammar. It is locale-independent and narrower than completion matching.

// NameMatching is the private command/flag recognition policy.

// dialect is the fully inherited grammar dialect for one command. Acceptance
// owns inheritance; invocation analysis consumes only this frozen effective value.
type dialect struct {
	commandMatching      NameMatching
	commandNormalization NameNormalization
	flagMatching         NameMatching
	flagNormalization    NameNormalization
	flagOrder            FlagOrder
	shortClustering      ShortFlagClustering
	dashlessFlags        DashlessFlagMode
	defaultFlagPlacement FlagPlacement
	endOfOptions         []string
	responseFiles        bool
}

type dialectInput struct {
	CommandMatching      NameMatching
	CommandNormalization NameNormalization
	FlagMatching         NameMatching
	FlagNormalization    NameNormalization
	FlagOrder            FlagOrder
	ShortClustering      ShortFlagClustering
	// DashlessFlags is the already-resolved effective dashless recognition
	// mode for this command. Public declaration inheritance and the root
	// initial value are resolved by the owning cli acceptance boundary before
	// this private semantic value is constructed.
	DashlessFlags        DashlessFlagMode
	DefaultFlagPlacement FlagPlacement
	EndOfOptions         []string

	// ResponseFiles is the already-resolved effective recognition state for
	// this command. Public declaration inheritance and the root initial value
	// are resolved by the owning cli acceptance boundary before this private
	// semantic value is constructed.
	ResponseFiles bool
}

func validDialectInput(input dialectInput) bool {
	if input.CommandMatching == NameMatchingUnknown || input.CommandNormalization == NameNormalizationUnknown ||
		input.FlagMatching == NameMatchingUnknown || input.FlagNormalization == NameNormalizationUnknown ||
		input.FlagOrder == FlagOrderUnknown || input.ShortClustering == ShortFlagClusteringUnknown ||
		input.DashlessFlags == DashlessFlagModeUnknown || input.DefaultFlagPlacement == FlagPlacementUnknown {
		return false
	}
	return !slices.Contains(input.EndOfOptions, "")
}

func acceptDialect(input dialectInput) dialect {
	if !validDialectInput(input) {
		panic("semantic: invalid grammar dialect")
	}
	end := append([]string(nil), input.EndOfOptions...)
	result := dialect{
		commandMatching:      input.CommandMatching,
		commandNormalization: input.CommandNormalization,
		flagMatching:         input.FlagMatching,
		flagNormalization:    input.FlagNormalization,
		flagOrder:            input.FlagOrder,
		shortClustering:      input.ShortClustering,
		dashlessFlags:        input.DashlessFlags,
		defaultFlagPlacement: input.DefaultFlagPlacement,
		endOfOptions:         end,
		responseFiles:        input.ResponseFiles,
	}

	return result
}

func (dialect dialect) CommandMatching() NameMatching { return dialect.commandMatching }

func (dialect dialect) CommandNormalization() NameNormalization {
	return dialect.commandNormalization
}

func (dialect dialect) FlagMatching() NameMatching { return dialect.flagMatching }

func (dialect dialect) FlagNormalization() NameNormalization {
	return dialect.flagNormalization
}

func (dialect dialect) FlagOrder() FlagOrder { return dialect.flagOrder }

func (dialect dialect) ShortClustering() ShortFlagClustering { return dialect.shortClustering }

func (dialect dialect) DashlessFlags() DashlessFlagMode { return dialect.dashlessFlags }

func (dialect dialect) DefaultFlagPlacement() FlagPlacement {
	return dialect.defaultFlagPlacement
}

func (dialect dialect) EndOfOptions() []string { return append([]string(nil), dialect.endOfOptions...) }

func (dialect dialect) ResponseFiles() bool { return dialect.responseFiles }

// Equal reports semantic equality of two fully resolved dialects.
func (dialect dialect) Equal(other dialect) bool {
	return dialect.commandMatching == other.commandMatching &&
		dialect.commandNormalization == other.commandNormalization &&
		dialect.flagMatching == other.flagMatching &&
		dialect.flagNormalization == other.flagNormalization &&
		dialect.flagOrder == other.flagOrder &&
		dialect.shortClustering == other.shortClustering &&
		dialect.dashlessFlags == other.dashlessFlags &&
		dialect.defaultFlagPlacement == other.defaultFlagPlacement &&
		slices.Equal(dialect.endOfOptions, other.endOfOptions) &&
		dialect.responseFiles == other.responseFiles
}
