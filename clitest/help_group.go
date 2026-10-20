package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// HelpCommandGroupLocator identifies one static command help group in the owning
// command's public Schema or Reference metadata. Its zero value is invalid;
// obtain a value from CommandLocator.HelpCommandGroup or
// CommandLocator.HelpCommandGroupAt.
type HelpCommandGroupLocator struct {
	_     [0]func()
	state *helpCommandGroupLocatorState
}

// HelpFlagGroupLocator identifies one static flag help group in the owning
// command's public Schema or Reference metadata. Its zero value is invalid;
// obtain a value from CommandLocator.HelpFlagGroup or
// CommandLocator.HelpFlagGroupAt.
type HelpFlagGroupLocator struct {
	_     [0]func()
	state *helpFlagGroupLocatorState
}

// HelpCommandGroupEntry is the closed set of locators that may occupy one
// static command help group while preserving heterogeneous entry order.
type HelpCommandGroupEntry interface {
	helpCommandGroupEntry()
}

func (CommandEntryLocator) helpCommandGroupEntry() {}

func (CommandProviderLocator) helpCommandGroupEntry() {}

// HelpCommandGroup returns a locator for the unique command help group with title under the
// located command; duplicate titles remain ambiguous.
func (locator CommandLocator) HelpCommandGroup(title string) HelpCommandGroupLocator {
	return HelpCommandGroupLocator{
		state: &helpCommandGroupLocatorState{
			command: locator.state,
			mode:    helpGroupLocatorTitle,
			title:   title,
		},
	}
}

// HelpCommandGroupAt returns a locator for the command help group at index in the located
// command's authoritative public help-group order.
func (locator CommandLocator) HelpCommandGroupAt(index int) HelpCommandGroupLocator {
	return HelpCommandGroupLocator{
		state: &helpCommandGroupLocatorState{
			command: locator.state,
			mode:    helpGroupLocatorIndex,
			index:   index,
		},
	}
}

// HelpFlagGroup returns a locator for the unique flag help group with title under the located
// command; duplicate titles remain ambiguous.
func (locator CommandLocator) HelpFlagGroup(title string) HelpFlagGroupLocator {
	return HelpFlagGroupLocator{
		state: &helpFlagGroupLocatorState{
			command: locator.state,
			mode:    helpGroupLocatorTitle,
			title:   title,
		},
	}
}

// HelpFlagGroupAt returns a locator for the flag help group at index in the located command's
// authoritative public help-group order.
func (locator CommandLocator) HelpFlagGroupAt(index int) HelpFlagGroupLocator {
	return HelpFlagGroupLocator{
		state: &helpFlagGroupLocatorState{
			command: locator.state,
			mode:    helpGroupLocatorIndex,
			index:   index,
		},
	}
}

// Exists returns an expectation that checks the located command help group exists in the static
// public help metadata.
func (locator HelpCommandGroupLocator) Exists() StaticDeclarationExpectation {
	return staticHelpCommandGroupExpectation{
		locator:  cloneHelpCommandGroupLocatorState(locator.state),
		property: staticHelpCommandGroupExpectationExists,
		exists:   true,
	}
}

// DoesNotExist returns an expectation that checks no command help group matches this locator in
// the complete static public help metadata.
func (locator HelpCommandGroupLocator) DoesNotExist() StaticDeclarationExpectation {
	return staticHelpCommandGroupExpectation{
		locator:  cloneHelpCommandGroupLocatorState(locator.state),
		property: staticHelpCommandGroupExpectationExists,
	}
}

// Kind returns an expectation that checks the located command help group has the expected
// HelpGroupKind.
func (locator HelpCommandGroupLocator) Kind(expected cli.HelpGroupKind) StaticDeclarationExpectation {
	return staticHelpCommandGroupExpectation{
		locator:  cloneHelpCommandGroupLocatorState(locator.state),
		property: staticHelpCommandGroupExpectationKind,
		kind:     expected,
	}
}

// Title returns an expectation that checks the located command help group has exactly the expected
// public title.
func (locator HelpCommandGroupLocator) Title(expected string) StaticDeclarationExpectation {
	return staticHelpCommandGroupExpectation{
		locator:  cloneHelpCommandGroupLocatorState(locator.state),
		property: staticHelpCommandGroupExpectationTitle,
		title:    expected,
	}
}

// Order returns an expectation that checks the located command help group has the expected public
// HelpGroupOrder metadata.
func (locator HelpCommandGroupLocator) Order(expected cli.HelpGroupOrder) StaticDeclarationExpectation {
	return staticHelpCommandGroupExpectation{
		locator:  cloneHelpCommandGroupLocatorState(locator.state),
		property: staticHelpCommandGroupExpectationOrder,
		order:    expected,
	}
}

// Profile returns an expectation that checks the located command help group is related to the
// exact grammar profile identified by expected.
func (locator HelpCommandGroupLocator) Profile(expected GrammarProfileLocator) StaticDeclarationExpectation {
	return staticHelpCommandGroupExpectation{
		locator:  cloneHelpCommandGroupLocatorState(locator.state),
		property: staticHelpCommandGroupExpectationProfile,
		profile:  expected.state,
	}
}

// NoProfile returns an expectation that checks the located command help group has no
// grammar-profile relationship.
func (locator HelpCommandGroupLocator) NoProfile() StaticDeclarationExpectation {
	return staticHelpCommandGroupExpectation{
		locator:   cloneHelpCommandGroupLocatorState(locator.state),
		property:  staticHelpCommandGroupExpectationProfile,
		noProfile: true,
	}
}

// EntriesExactly returns an expectation that checks the command help group entries are exactly
// expected in authoritative public order.
func (locator HelpCommandGroupLocator) EntriesExactly(expected ...HelpCommandGroupEntry) StaticDeclarationExpectation {
	states := make([]*helpCommandGroupEntryState, len(expected))
	for i, entry := range expected {
		states[i] = snapshotHelpCommandGroupEntry(entry)
	}
	expectation := staticHelpCommandGroupExpectation{
		locator:  cloneHelpCommandGroupLocatorState(locator.state),
		property: staticHelpCommandGroupExpectationEntries,
		entries:  states,
	}
	return expectation
}

// Exists returns an expectation that checks the located flag help group exists in the static
// public help metadata.
func (locator HelpFlagGroupLocator) Exists() StaticDeclarationExpectation {
	return staticHelpFlagGroupExpectation{
		locator:  cloneHelpFlagGroupLocatorState(locator.state),
		property: staticHelpFlagGroupExpectationExists,
		exists:   true,
	}
}

// DoesNotExist returns an expectation that checks no flag help group matches this locator in the
// complete static public help metadata.
func (locator HelpFlagGroupLocator) DoesNotExist() StaticDeclarationExpectation {
	return staticHelpFlagGroupExpectation{
		locator:  cloneHelpFlagGroupLocatorState(locator.state),
		property: staticHelpFlagGroupExpectationExists,
	}
}

// Kind returns an expectation that checks the located flag help group has the expected
// HelpGroupKind.
func (locator HelpFlagGroupLocator) Kind(expected cli.HelpGroupKind) StaticDeclarationExpectation {
	return staticHelpFlagGroupExpectation{
		locator:  cloneHelpFlagGroupLocatorState(locator.state),
		property: staticHelpFlagGroupExpectationKind,
		kind:     expected,
	}
}

// Title returns an expectation that checks the located flag help group has exactly the expected
// public title.
func (locator HelpFlagGroupLocator) Title(expected string) StaticDeclarationExpectation {
	return staticHelpFlagGroupExpectation{
		locator:  cloneHelpFlagGroupLocatorState(locator.state),
		property: staticHelpFlagGroupExpectationTitle,
		title:    expected,
	}
}

// Order returns an expectation that checks the located flag help group has the expected public
// HelpGroupOrder metadata.
func (locator HelpFlagGroupLocator) Order(expected cli.HelpGroupOrder) StaticDeclarationExpectation {
	return staticHelpFlagGroupExpectation{
		locator:  cloneHelpFlagGroupLocatorState(locator.state),
		property: staticHelpFlagGroupExpectationOrder,
		order:    expected,
	}
}

// Profile returns an expectation that checks the located flag help group is related to the exact
// grammar profile identified by expected.
func (locator HelpFlagGroupLocator) Profile(expected GrammarProfileLocator) StaticDeclarationExpectation {
	return staticHelpFlagGroupExpectation{
		locator:  cloneHelpFlagGroupLocatorState(locator.state),
		property: staticHelpFlagGroupExpectationProfile,
		profile:  expected.state,
	}
}

// NoProfile returns an expectation that checks the located flag help group has no grammar-profile
// relationship.
func (locator HelpFlagGroupLocator) NoProfile() StaticDeclarationExpectation {
	return staticHelpFlagGroupExpectation{
		locator:   cloneHelpFlagGroupLocatorState(locator.state),
		property:  staticHelpFlagGroupExpectationProfile,
		noProfile: true,
	}
}

// FlagsExactly returns an expectation that checks the flag help group members are exactly expected
// in authoritative public order.
func (locator HelpFlagGroupLocator) FlagsExactly(expected ...FlagLocator) StaticDeclarationExpectation {
	states := make([]*flagLocatorState, len(expected))
	for i, flag := range expected {
		states[i] = flag.state
	}
	return staticHelpFlagGroupExpectation{
		locator:  cloneHelpFlagGroupLocatorState(locator.state),
		property: staticHelpFlagGroupExpectationFlags,
		flags:    states,
	}
}

type helpGroupLocatorMode uint8

const (
	helpGroupLocatorInvalid helpGroupLocatorMode = iota
	helpGroupLocatorTitle
	helpGroupLocatorIndex
)

type helpGroupLocatorState struct {
	command *commandLocatorState
	mode    helpGroupLocatorMode
	title   string
	index   int
}

type helpCommandGroupEntryKind uint8

type helpCommandGroupEntryState struct {
	kind     helpCommandGroupEntryKind
	entry    *commandEntryLocatorState
	provider *commandProviderLocatorState
}

type staticHelpCommandGroupExpectationProperty uint8

const (
	staticHelpCommandGroupExpectationInvalid staticHelpCommandGroupExpectationProperty = iota
	staticHelpCommandGroupExpectationExists
	staticHelpCommandGroupExpectationKind
	staticHelpCommandGroupExpectationTitle
	staticHelpCommandGroupExpectationOrder
	staticHelpCommandGroupExpectationProfile
	staticHelpCommandGroupExpectationEntries
)

type staticHelpCommandGroupExpectation struct {
	locator   *helpCommandGroupLocatorState
	property  staticHelpCommandGroupExpectationProperty
	exists    bool
	kind      cli.HelpGroupKind
	title     string
	order     cli.HelpGroupOrder
	profile   *grammarProfileLocatorState
	noProfile bool
	entries   []*helpCommandGroupEntryState
}

type staticHelpFlagGroupExpectationProperty uint8

const (
	staticHelpFlagGroupExpectationInvalid staticHelpFlagGroupExpectationProperty = iota
	staticHelpFlagGroupExpectationExists
	staticHelpFlagGroupExpectationKind
	staticHelpFlagGroupExpectationTitle
	staticHelpFlagGroupExpectationOrder
	staticHelpFlagGroupExpectationProfile
	staticHelpFlagGroupExpectationFlags
)

type staticHelpFlagGroupExpectation struct {
	locator   *helpFlagGroupLocatorState
	property  staticHelpFlagGroupExpectationProperty
	exists    bool
	kind      cli.HelpGroupKind
	title     string
	order     cli.HelpGroupOrder
	profile   *grammarProfileLocatorState
	noProfile bool
	flags     []*flagLocatorState
}

type helpGroupResolution[Group any] struct {
	group  Group
	status locatorResolutionStatus
	label  evaluationField
	actual evaluationField
}

func helpGroupComparableOutcome(status locatorResolutionStatus, label, detail evaluationField, field string, expected, actual evaluationField, equal bool) evaluationResult {
	subject := qualifySubject(label, evaluationText(field))
	expectedField := property(evaluationText(field), expected)
	if status != locatorResolutionFound {
		return outcomeForValueStatus(status, subject, expectedField, detail)
	}
	if equal {
		return provedResult()
	}
	return disprovedResult(subject, expectedField, property(evaluationText(field), actual))
}

func snapshotHelpCommandGroupEntry(entry HelpCommandGroupEntry) *helpCommandGroupEntryState {
	switch value := entry.(type) {
	case CommandEntryLocator:
		return &helpCommandGroupEntryState{
			kind:  helpCommandGroupEntryEntry,
			entry: value.state,
		}
	case *CommandEntryLocator:
		if value == nil {
			panic("clitest: nil help command group entry locator")
		}
		return &helpCommandGroupEntryState{
			kind:  helpCommandGroupEntryEntry,
			entry: value.state,
		}
	case CommandProviderLocator:
		return &helpCommandGroupEntryState{
			kind:     helpCommandGroupEntryProvider,
			provider: value.state,
		}
	case *CommandProviderLocator:
		if value == nil {
			panic("clitest: nil help command group provider locator")
		}
		return &helpCommandGroupEntryState{
			kind:     helpCommandGroupEntryProvider,
			provider: value.state,
		}
	default:
		panic(fmt.Sprintf("clitest: unsupported help command group entry %T", entry))
	}
}

func resolveSchemaHelpCommandGroup(state *helpCommandGroupLocatorState, schema *cli.Schema) helpCommandGroupResolution {
	var resolution helpCommandGroupResolution
	if state == nil || schema == nil {
		resolution = helpCommandGroupResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectHelpCommandGroup,
			actual: observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = helpCommandGroupResolution{
			status: locatorResolutionAbsent,
			label:  qualifySubject(command.subject, semanticSubjectHelpCommandGroup),
			actual: command.detail,
		}
		return resolution
	}
	return selectHelpCommandGroup(state, command.value.Help.CommandGroups, command.subject)
}

func resolveReferenceHelpCommandGroup(state *helpCommandGroupLocatorState, reference cli.Reference) helpCommandGroupResolution {
	var resolution helpCommandGroupResolution
	if state == nil {
		resolution = helpCommandGroupResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectHelpCommandGroup,
			actual: observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = helpCommandGroupResolution{
			status: locatorResolutionAbsent,
			label:  qualifySubject(command.subject, semanticSubjectHelpCommandGroup),
			actual: command.detail,
		}
		return resolution
	}
	return selectHelpCommandGroup(state, command.value.Help.CommandGroups, command.subject)
}

//nolint:dupl // Command and flag Help groups intentionally keep direct typed selection instead of a callback-driven generic selector.
func selectHelpCommandGroup(state *helpCommandGroupLocatorState, groups []cli.HelpCommandGroupSchema, commandLabel evaluationField) helpCommandGroupResolution {
	var resolution helpCommandGroupResolution
	switch state.mode {
	case helpGroupLocatorIndex:
		label := qualifySubject(commandLabel, subjectAt(semanticSubjectHelpCommandGroup, state.index))
		if state.index < 0 || state.index >= len(groups) {
			resolution = helpCommandGroupResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: observed(semanticSubjectHelpCommandGroup, observationAbsent),
			}
			return resolution
		}
		resolution = helpCommandGroupResolution{
			group:  groups[state.index],
			status: locatorResolutionFound,
			label:  label,
		}
		return resolution
	case helpGroupLocatorTitle:
		label := qualifySubject(commandLabel, subjectNamed(semanticSubjectHelpCommandGroup, state.title))
		matches := make([]int, 0, 1)
		for index, group := range groups {
			if group.Title == state.title {
				matches = append(matches, index)
			}
		}
		if len(matches) == 1 {
			resolution = helpCommandGroupResolution{
				group:  groups[matches[0]],
				status: locatorResolutionFound,
				label:  label,
			}
			return resolution
		}
		if len(matches) > 1 {
			resolution = helpCommandGroupResolution{
				status: locatorResolutionAmbiguous,
				label:  label,
				actual: collectionCardinalityField{
					name:  "matches title indexes",
					count: len(matches),
				},
			}
			return resolution
		}
		resolution = helpCommandGroupResolution{
			status: locatorResolutionAbsent,
			label:  label,
			actual: observed(semanticSubjectHelpCommandGroup, observationAbsent),
		}
		return resolution
	default:
		resolution = helpCommandGroupResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectHelpCommandGroup,
			actual: observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
}

func resolveSchemaHelpFlagGroup(state *helpFlagGroupLocatorState, schema *cli.Schema) helpFlagGroupResolution {
	var resolution helpFlagGroupResolution
	if state == nil || schema == nil {
		resolution = helpFlagGroupResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectHelpFlagGroup,
			actual: observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveSchemaCommandLocator(state.command, schema)
	if command.status != locatorResolutionFound {
		resolution = helpFlagGroupResolution{
			status: locatorResolutionAbsent,
			label:  qualifySubject(command.subject, semanticSubjectHelpFlagGroup),
			actual: command.detail,
		}
		return resolution
	}
	return selectHelpFlagGroup(state, command.value.Help.FlagGroups, command.subject)
}

func resolveReferenceHelpFlagGroup(state *helpFlagGroupLocatorState, reference cli.Reference) helpFlagGroupResolution {
	var resolution helpFlagGroupResolution
	if state == nil {
		resolution = helpFlagGroupResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectHelpFlagGroup,
			actual: observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	command := resolveReferenceCommandLocator(state.command, reference)
	if command.status != locatorResolutionFound {
		resolution = helpFlagGroupResolution{
			status: locatorResolutionAbsent,
			label:  qualifySubject(command.subject, semanticSubjectHelpFlagGroup),
			actual: command.detail,
		}
		return resolution
	}
	return selectHelpFlagGroup(state, command.value.Help.FlagGroups, command.subject)
}

//nolint:dupl // Command and flag Help groups intentionally keep direct typed selection instead of a callback-driven generic selector.
func selectHelpFlagGroup(state *helpFlagGroupLocatorState, groups []cli.HelpFlagGroupSchema, commandLabel evaluationField) helpFlagGroupResolution {
	var resolution helpFlagGroupResolution
	switch state.mode {
	case helpGroupLocatorIndex:
		label := qualifySubject(commandLabel, subjectAt(semanticSubjectHelpFlagGroup, state.index))
		if state.index < 0 || state.index >= len(groups) {
			resolution = helpFlagGroupResolution{
				status: locatorResolutionAbsent,
				label:  label,
				actual: observed(semanticSubjectHelpFlagGroup, observationAbsent),
			}
			return resolution
		}
		resolution = helpFlagGroupResolution{
			group:  groups[state.index],
			status: locatorResolutionFound,
			label:  label,
		}
		return resolution
	case helpGroupLocatorTitle:
		label := qualifySubject(commandLabel, subjectNamed(semanticSubjectHelpFlagGroup, state.title))
		matches := make([]int, 0, 1)
		for index, group := range groups {
			if group.Title == state.title {
				matches = append(matches, index)
			}
		}
		if len(matches) == 1 {
			resolution = helpFlagGroupResolution{
				group:  groups[matches[0]],
				status: locatorResolutionFound,
				label:  label,
			}
			return resolution
		}
		if len(matches) > 1 {
			resolution = helpFlagGroupResolution{
				status: locatorResolutionAmbiguous,
				label:  label,
				actual: collectionCardinalityField{
					name:  "matches title indexes",
					count: len(matches),
				},
			}
			return resolution
		}
		resolution = helpFlagGroupResolution{
			status: locatorResolutionAbsent,
			label:  label,
			actual: observed(semanticSubjectHelpFlagGroup, observationAbsent),
		}
		return resolution
	default:
		resolution = helpFlagGroupResolution{
			status: locatorResolutionInvalid,
			label:  semanticSubjectHelpFlagGroup,
			actual: observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
}

func helpGroupExistenceOutcome(status locatorResolutionStatus, label, actual evaluationField, expect string, expected bool) evaluationResult {
	if status == locatorResolutionAmbiguous || status == locatorResolutionInvalid {
		return disprovedResult(label, evaluationText(expect), actual)
	}
	found := status == locatorResolutionFound
	if found == expected {
		return provedResult()
	}
	return disprovedResult(label, evaluationText(expect), evaluationText(fmt.Sprintf("exists=%t", found)))
}

func compareHelpGroupProfileSchema(label evaluationField, actual cli.GrammarProfileID, expected *grammarProfileLocatorState, none bool, schema *cli.Schema) evaluationResult {
	if none {
		if actual == 0 {
			return provedResult()
		}
		return disprovedResult(qualifySubject(label, evaluationText("profile")), evaluationText("none"), evaluationText(fmt.Sprintf("profile %d", actual)))
	}
	id, status, detail := expectedSchemaGrammarProfileID(expected, schema)
	if status != locatorResolutionFound {
		return outcomeForValueStatus(status, qualifySubject(label, evaluationText("profile")), evaluationText("valid grammar profile"), detail)
	}
	if actual == id {
		return provedResult()
	}
	return disprovedResult(
		qualifySubject(label, evaluationText("profile")),
		evaluationText(fmt.Sprintf("profile %d", id)),
		evaluationText(fmt.Sprintf("profile %d", actual)),
	)
}

func compareHelpGroupProfileReference(label evaluationField, actual cli.GrammarProfileID, expected *grammarProfileLocatorState, none bool, reference cli.Reference) evaluationResult {
	if none {
		if actual == 0 {
			return provedResult()
		}
		return disprovedResult(qualifySubject(label, evaluationText("profile")), evaluationText("none"), evaluationText(fmt.Sprintf("profile %d", actual)))
	}
	id, status, detail := expectedReferenceGrammarProfileID(expected, reference)
	if status != locatorResolutionFound {
		return outcomeForValueStatus(status, qualifySubject(label, evaluationText("profile")), evaluationText("valid grammar profile"), detail)
	}
	if actual == id {
		return provedResult()
	}
	return disprovedResult(
		qualifySubject(label, evaluationText("profile")),
		evaluationText(fmt.Sprintf("profile %d", id)),
		evaluationText(fmt.Sprintf("profile %d", actual)),
	)
}

//nolint:dupl // Schema and Reference Help command-group entry proofs intentionally remain direct typed evaluators.
func compareHelpCommandGroupEntriesSchema(label evaluationField, profile cli.GrammarProfileID, actual []cli.HelpCommandGroupEntrySchema, expected []*helpCommandGroupEntryState, schema *cli.Schema) evaluationResult {
	if len(actual) != len(expected) {
		expectedField := orderedCollectionField{
			name:          semanticNameHelpCommandGroupEntries,
			length:        len(expected),
			mismatchIndex: -1,
			expected:      true,
		}
		observedField := orderedCollectionField{
			name:          semanticNameHelpCommandGroupEntries,
			length:        len(actual),
			mismatchIndex: -1,
		}
		return disprovedResult(
			qualifySubject(label, evaluationText("entries")),
			expectedField,
			observedField,
		)
	}
	for i := range expected {
		want, status, detail := resolveHelpCommandGroupEntrySchema(expected[i], schema)
		if status != locatorResolutionFound {
			return outcomeForValueStatus(status, qualifySubject(label, evaluationText(fmt.Sprintf("entry %d", i))), evaluationText("valid expected entry"), detail)
		}
		want.Profile = profile
		if actual[i] != want {
			expectedField := orderedCollectionField{
				name:          semanticNameHelpCommandGroupEntries,
				length:        len(expected),
				mismatchIndex: i,
				expected:      true,
			}
			observedField := orderedCollectionField{
				name:          semanticNameHelpCommandGroupEntries,
				length:        len(actual),
				mismatchIndex: i,
			}
			return disprovedResult(
				qualifySubject(label, subjectAt(evaluationText("entry"), i)),
				expectedField,
				observedField,
			)
		}
	}
	return provedResult()
}

//nolint:dupl // Schema and Reference Help command-group entry proofs intentionally remain direct typed evaluators.
func compareHelpCommandGroupEntriesReference(label evaluationField, profile cli.GrammarProfileID, actual []cli.HelpCommandGroupEntrySchema, expected []*helpCommandGroupEntryState, reference cli.Reference) evaluationResult {
	if len(actual) != len(expected) {
		expectedField := orderedCollectionField{
			name:          semanticNameHelpCommandGroupEntries,
			length:        len(expected),
			mismatchIndex: -1,
			expected:      true,
		}
		observedField := orderedCollectionField{
			name:          semanticNameHelpCommandGroupEntries,
			length:        len(actual),
			mismatchIndex: -1,
		}
		return disprovedResult(
			qualifySubject(label, evaluationText("entries")),
			expectedField,
			observedField,
		)
	}
	for i := range expected {
		want, status, detail := resolveHelpCommandGroupEntryReference(expected[i], reference)
		if status != locatorResolutionFound {
			return outcomeForValueStatus(status, qualifySubject(label, evaluationText(fmt.Sprintf("entry %d", i))), evaluationText("valid expected entry"), detail)
		}
		want.Profile = profile
		if actual[i] != want {
			expectedField := orderedCollectionField{
				name:          semanticNameHelpCommandGroupEntries,
				length:        len(expected),
				mismatchIndex: i,
				expected:      true,
			}
			observedField := orderedCollectionField{
				name:          semanticNameHelpCommandGroupEntries,
				length:        len(actual),
				mismatchIndex: i,
			}
			return disprovedResult(
				qualifySubject(label, subjectAt(evaluationText("entry"), i)),
				expectedField,
				observedField,
			)
		}
	}
	return provedResult()
}

func resolveHelpCommandGroupEntrySchema(state *helpCommandGroupEntryState, schema *cli.Schema) (cli.HelpCommandGroupEntrySchema, locatorResolutionStatus, evaluationField) {
	var groupEntry cli.HelpCommandGroupEntrySchema
	if state == nil {
		return cli.HelpCommandGroupEntrySchema{}, locatorResolutionInvalid, evaluationText("invalid help command-group entry")
	}

	switch state.kind {
	case helpCommandGroupEntryEntry:
		id, status, detail := expectedSchemaCommandEntryID(state.entry, schema)
		if status != locatorResolutionFound {
			return cli.HelpCommandGroupEntrySchema{}, status, detail
		}
		groupEntry = cli.HelpCommandGroupEntrySchema{
			Kind:         cli.HelpCommandGroupEntryCommandEntry,
			CommandEntry: id,
		}
		return groupEntry, locatorResolutionFound, evaluationText("")
	case helpCommandGroupEntryProvider:
		id, status, detail := expectedSchemaCommandProviderID(state.provider, schema)
		if status != locatorResolutionFound {
			return cli.HelpCommandGroupEntrySchema{}, status, detail
		}
		groupEntry = cli.HelpCommandGroupEntrySchema{
			Kind:     cli.HelpCommandGroupEntryCommandProvider,
			Provider: id,
		}
		return groupEntry, locatorResolutionFound, evaluationText("")
	default:
		return cli.HelpCommandGroupEntrySchema{}, locatorResolutionInvalid, evaluationText("invalid help command-group entry kind")
	}
}

func resolveHelpCommandGroupEntryReference(state *helpCommandGroupEntryState, reference cli.Reference) (cli.HelpCommandGroupEntrySchema, locatorResolutionStatus, evaluationField) {
	var groupEntry cli.HelpCommandGroupEntrySchema
	if state == nil {
		return cli.HelpCommandGroupEntrySchema{}, locatorResolutionInvalid, evaluationText("invalid help command-group entry")
	}

	switch state.kind {
	case helpCommandGroupEntryEntry:
		id, status, detail := expectedReferenceCommandEntryID(state.entry, reference)
		if status != locatorResolutionFound {
			return cli.HelpCommandGroupEntrySchema{}, status, detail
		}
		groupEntry = cli.HelpCommandGroupEntrySchema{
			Kind:         cli.HelpCommandGroupEntryCommandEntry,
			CommandEntry: id,
		}
		return groupEntry, locatorResolutionFound, evaluationText("")
	case helpCommandGroupEntryProvider:
		id, status, detail := expectedReferenceCommandProviderID(state.provider, reference)
		if status != locatorResolutionFound {
			return cli.HelpCommandGroupEntrySchema{}, status, detail
		}
		groupEntry = cli.HelpCommandGroupEntrySchema{
			Kind:     cli.HelpCommandGroupEntryCommandProvider,
			Provider: id,
		}
		return groupEntry, locatorResolutionFound, evaluationText("")
	default:
		return cli.HelpCommandGroupEntrySchema{}, locatorResolutionInvalid, evaluationText("invalid help command-group entry kind")
	}
}

func resolveHelpFlagIDsSchema(states []*flagLocatorState, schema *cli.Schema) ([]cli.FlagID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.FlagID, len(states))
	for i, state := range states {
		id, status, detail := expectedSchemaFlagID(state, schema)
		if status != locatorResolutionFound {
			return nil, status, detail
		}
		ids[i] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func resolveHelpFlagIDsReference(states []*flagLocatorState, reference cli.Reference) ([]cli.FlagID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.FlagID, len(states))
	for i, state := range states {
		id, status, detail := expectedReferenceFlagID(state, reference)
		if status != locatorResolutionFound {
			return nil, status, detail
		}
		ids[i] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func compareHelpFlagIDs(label evaluationField, expected, actual []cli.FlagID) evaluationResult {
	if slices.Equal(expected, actual) {
		return provedResult()
	}
	return orderedCollectionMismatch(qualifySubject(label, evaluationText("flags")), "flag IDs", expected, actual)
}

func cloneHelpCommandGroupLocatorState(state *helpCommandGroupLocatorState) *helpCommandGroupLocatorState {
	if state == nil {
		return nil
	}

	return new(*state)
}

func cloneHelpFlagGroupLocatorState(state *helpFlagGroupLocatorState) *helpFlagGroupLocatorState {
	if state == nil {
		return nil
	}

	return new(*state)
}

func (helpCommandGroupExpectation) helpPresentationExpectation() {}

//nolint:dupl // Command and flag Help groups intentionally keep direct typed proof logic.
func (expectation helpCommandGroupExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	group, status, detail := resolveHelpCommandGroup(expectation.locator, page)
	if expectation.property == helpCommandGroupExists {
		return evaluateHelpExistence(semanticSubjectHelpCommandGroup, expectation.exists, status)
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(semanticSubjectHelpCommandGroup, expectation.commandGroupExpected(), detail)
	}
	switch expectation.property {
	case helpCommandGroupKind:
		if group.Kind == expectation.kind {
			return provedResult()
		}
		return helpPropertyMismatch(semanticSubjectHelpCommandGroup, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(group.Kind)))
	case helpCommandGroupTitle:
		if group.Title == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(semanticSubjectHelpCommandGroup, "title", evaluationText(expectation.text), evaluationText(group.Title))
	case helpCommandGroupOrder:
		if group.Order == expectation.order {
			return provedResult()
		}
		return helpPropertyMismatch(semanticSubjectHelpCommandGroup, "order", evaluationText(fmt.Sprint(expectation.order)), evaluationText(fmt.Sprint(group.Order)))
	case helpCommandGroupCommandCount:
		if len(group.Commands) == expectation.count {
			return provedResult()
		}
		return helpPropertyMismatch(semanticSubjectHelpCommandGroup, "command-count", evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(group.Commands))))
	default:
		panic("clitest: invalid Help command-group expectation")
	}
}

func (expectation helpCommandGroupExpectation) commandGroupExpected() evaluationField {
	switch expectation.property {
	case helpCommandGroupKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)))
	case helpCommandGroupTitle:
		return property(evaluationText("title"), evaluationText(expectation.text))
	case helpCommandGroupOrder:
		return property(evaluationText("order"), evaluationText(fmt.Sprint(expectation.order)))
	case helpCommandGroupCommandCount:
		return property(evaluationText("command-count"), evaluationText(strconv.Itoa(expectation.count)))
	default:
		return evaluationText("valid command-group expectation")
	}
}

func (helpFlagGroupExpectation) helpPresentationExpectation() {}

//nolint:dupl // Command and flag Help groups intentionally keep direct typed proof logic.
func (expectation helpFlagGroupExpectation) evaluateHelp(page cli.HelpPage) evaluationResult {
	group, status, detail := resolveHelpFlagGroup(expectation.locator, page)
	subject := evaluationText("help flag group")
	if expectation.property == helpFlagGroupExists {
		return evaluateHelpExistence(subject, expectation.exists, status)
	}
	if status != locatorResolutionFound {
		return helpResolutionFailure(subject, expectation.flagGroupExpected(), detail)
	}
	switch expectation.property {
	case helpFlagGroupKind:
		if group.Kind == expectation.kind {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(group.Kind)))
	case helpFlagGroupTitle:
		if group.Title == expectation.text {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "title", evaluationText(expectation.text), evaluationText(group.Title))
	case helpFlagGroupOrder:
		if group.Order == expectation.order {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "order", evaluationText(fmt.Sprint(expectation.order)), evaluationText(fmt.Sprint(group.Order)))
	case helpFlagGroupFlagCount:
		if len(group.Flags) == expectation.count {
			return provedResult()
		}
		return helpPropertyMismatch(subject, "flag-count", evaluationText(strconv.Itoa(expectation.count)), evaluationText(strconv.Itoa(len(group.Flags))))
	default:
		panic("clitest: invalid Help flag-group expectation")
	}
}

func (expectation helpFlagGroupExpectation) flagGroupExpected() evaluationField {
	switch expectation.property {
	case helpFlagGroupKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)))
	case helpFlagGroupTitle:
		return property(evaluationText("title"), evaluationText(expectation.text))
	case helpFlagGroupOrder:
		return property(evaluationText("order"), evaluationText(fmt.Sprint(expectation.order)))
	case helpFlagGroupFlagCount:
		return property(evaluationText("flag-count"), evaluationText(strconv.Itoa(expectation.count)))
	default:
		return evaluationText("valid flag-group expectation")
	}
}

func (staticHelpCommandGroupExpectation) schemaExpectation() {}

func (staticHelpCommandGroupExpectation) referenceExpectation() {}

func (expectation staticHelpCommandGroupExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaHelpCommandGroup(expectation.locator, schema)
	return expectation.evaluateSchemaResolved(resolved, schema)
}

func (expectation staticHelpCommandGroupExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceHelpCommandGroup(expectation.locator, reference)
	return expectation.evaluateReferenceResolved(resolved, reference)
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation staticHelpCommandGroupExpectation) evaluateSchemaResolved(resolved helpCommandGroupResolution, schema *cli.Schema) evaluationResult {
	switch expectation.property {
	case staticHelpCommandGroupExpectationExists:
		return helpGroupExistenceOutcome(resolved.status, resolved.label, resolved.actual, fmt.Sprintf("exists=%t", expectation.exists), expectation.exists)
	case staticHelpCommandGroupExpectationKind:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(resolved.group.Kind)), resolved.group.Kind == expectation.kind)
	case staticHelpCommandGroupExpectationTitle:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "title", evaluationText(expectation.title), evaluationText(resolved.group.Title), resolved.group.Title == expectation.title)
	case staticHelpCommandGroupExpectationOrder:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "order", evaluationText(fmt.Sprint(expectation.order)), evaluationText(fmt.Sprint(resolved.group.Order)), resolved.group.Order == expectation.order)
	case staticHelpCommandGroupExpectationProfile:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("profile")), evaluationText("profile"), resolved.actual)
		}
		return compareHelpGroupProfileSchema(resolved.label, resolved.group.Profile, expectation.profile, expectation.noProfile, schema)
	case staticHelpCommandGroupExpectationEntries:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("entries")), evaluationText("exact entries"), resolved.actual)
		}
		return compareHelpCommandGroupEntriesSchema(resolved.label, resolved.group.Profile, resolved.group.Entries, expectation.entries, schema)
	default:
		return disprovedResult(semanticSubjectHelpCommandGroup, evaluationText("valid expectation"), observed(semanticSubjectLocator, observationInvalidLocator))
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation staticHelpCommandGroupExpectation) evaluateReferenceResolved(resolved helpCommandGroupResolution, reference cli.Reference) evaluationResult {
	switch expectation.property {
	case staticHelpCommandGroupExpectationExists:
		return helpGroupExistenceOutcome(resolved.status, resolved.label, resolved.actual, fmt.Sprintf("exists=%t", expectation.exists), expectation.exists)
	case staticHelpCommandGroupExpectationKind:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "kind", evaluationText(fmt.Sprint(expectation.kind)), evaluationText(fmt.Sprint(resolved.group.Kind)), resolved.group.Kind == expectation.kind)
	case staticHelpCommandGroupExpectationTitle:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "title", evaluationText(expectation.title), evaluationText(resolved.group.Title), resolved.group.Title == expectation.title)
	case staticHelpCommandGroupExpectationOrder:
		return helpGroupComparableOutcome(resolved.status, resolved.label, resolved.actual, "order", evaluationText(fmt.Sprint(expectation.order)), evaluationText(fmt.Sprint(resolved.group.Order)), resolved.group.Order == expectation.order)
	case staticHelpCommandGroupExpectationProfile:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("profile")), evaluationText("profile"), resolved.actual)
		}
		return compareHelpGroupProfileReference(resolved.label, resolved.group.Profile, expectation.profile, expectation.noProfile, reference)
	case staticHelpCommandGroupExpectationEntries:
		if resolved.status != locatorResolutionFound {
			return outcomeForValueStatus(resolved.status, qualifySubject(resolved.label, evaluationText("entries")), evaluationText("exact entries"), resolved.actual)
		}
		return compareHelpCommandGroupEntriesReference(resolved.label, resolved.group.Profile, resolved.group.Entries, expectation.entries, reference)
	default:
		return disprovedResult(semanticSubjectHelpCommandGroup, evaluationText("valid expectation"), observed(semanticSubjectLocator, observationInvalidLocator))
	}
}
