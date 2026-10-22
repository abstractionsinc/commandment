package clitest

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// CompletionTargetLocator identifies one completion target in a Completion
// result for clitest semantic expectation. Its coordinates are interpreted only
// against the Completion observation being evaluated. Its zero value is invalid.
type CompletionTargetLocator struct {
	_     [0]func()
	state *completionTargetLocatorState
}

// CompletionItemLocator identifies one completion item in a Completion result
// for clitest semantic expectation. Its coordinates are interpreted only against the
// Completion observation being evaluated. Its zero value is invalid.
type CompletionItemLocator struct {
	_     [0]func()
	state *completionItemLocatorState
}

// CompletionGroupLocator identifies one completion group in a Completion result
// for clitest semantic expectation. Its coordinates are interpreted only against the
// Completion observation being evaluated. Its zero value is invalid.
type CompletionGroupLocator struct {
	_     [0]func()
	state *completionGroupLocatorState
}

// CompletionNativeRequestLocator identifies one native-completion request in a
// Completion result for clitest semantic expectation. Its coordinates are interpreted
// only against the Completion observation being evaluated. Its zero value is
// invalid.
type CompletionNativeRequestLocator struct {
	_     [0]func()
	state *completionNativeRequestLocatorState
}

// CompletionDelegationLocator identifies the delegated-completion relationship
// in a Completion result for clitest semantic expectation. Its coordinates are
// interpreted only against the Completion observation being evaluated. Its zero
// value is invalid.
type CompletionDelegationLocator struct {
	_     [0]func()
	state *completionDelegationLocatorState
}

// CompletionDeclarationLocator identifies the public completion capabilities attached to a flag
// or argument term. It checks declaration-time completion facts across the observations that
// expose them; the zero value is invalid and it is not a completion operation result.
type CompletionDeclarationLocator struct {
	_     [0]func()
	state *completionDeclarationLocatorState
}

// CompletionTargetSubject is the closed set of semantic subjects that can be
// named as a completion target.
type CompletionTargetSubject interface{ completionTargetSubject() }

// CompletionTarget returns an expected coordinate for the unique target of kind in a Completion observation. A missing
// match in a partial or truncated Completion does not establish absence.
func CompletionTarget(kind cli.CompletionTargetKind) CompletionTargetLocator {
	return completionTargetKindLocator(kind)
}

// CompletionTargetAt returns an expected coordinate for the target at index in Completion.Targets authoritative order.
// An out-of-range coordinate in a partial or truncated Completion is not treated as expectation of absence.
func CompletionTargetAt(index int) CompletionTargetLocator {
	return completionTargetIndexLocator(index)
}

// CompletionItem returns an expected coordinate for the unique item whose observed Value exactly equals value. It never
// chooses among duplicate values, and an unmatched value in an incomplete Completion is not expectation of absence.
func CompletionItem(value string) CompletionItemLocator { return completionItemValueLocator(value) }

// CompletionItemAt returns an expected coordinate for the item at index in Completion.Items authoritative order. An
// out-of-range coordinate in a partial or truncated Completion remains not provable.
func CompletionItemAt(index int) CompletionItemLocator { return completionItemIndexLocator(index) }

// CompletionGroup returns an expected coordinate for the unique group whose observed ID exactly equals id. Duplicate
// IDs are ambiguous, and an unmatched ID in an incomplete Completion is not expectation of absence.
func CompletionGroup(id string) CompletionGroupLocator { return completionGroupIDLocator(id) }

// CompletionGroupAt returns an expected coordinate for the group at index in Completion.Groups authoritative order. An
// out-of-range coordinate in a partial or truncated Completion remains not provable.
func CompletionGroupAt(index int) CompletionGroupLocator { return completionGroupIndexLocator(index) }

// CompletionNativeRequestAt returns an expected coordinate for the native request at index in Completion.NativeRequests
// authoritative order. An out-of-range coordinate in an incomplete Completion remains not provable.
func CompletionNativeRequestAt(index int) CompletionNativeRequestLocator {
	return completionNativeRequestIndexLocator(index)
}

// CompletionDelegation returns the expected coordinate for the optional Completion delegation relationship. When an
// incomplete Completion carries no delegation, its absence and dependent properties are not provable.
func CompletionDelegation() CompletionDelegationLocator { return completionDelegationLocator() }

// CompletionObservedTargetsExactly returns an expectation that the targets present in the returned Completion snapshot
// are exactly the supplied locators in the same order.
// This is observed-snapshot exactness: it remains evaluable when Completion is partial or truncated and does not claim
// that omitted, unobserved subjects do not exist.
func CompletionObservedTargetsExactly(locators ...CompletionTargetLocator) CompletionExpectation {
	states := make([]*completionTargetLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionObservedTargetsExpectationValue{
		states: states,
	}
	return value
}

// CompletionObservedItemsExactly returns an expectation that the items present in the returned Completion snapshot are
// exactly the supplied locators in the same order.
// This is observed-snapshot exactness: it remains evaluable when Completion is partial or truncated and does not claim
// that omitted, unobserved subjects do not exist.
func CompletionObservedItemsExactly(locators ...CompletionItemLocator) CompletionExpectation {
	states := make([]*completionItemLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionObservedItemsExpectationValue{
		states: states,
	}
	return value
}

// CompletionObservedGroupsExactly returns an expectation that the groups present in the returned Completion snapshot
// are exactly the supplied locators in the same order.
// This is observed-snapshot exactness: it remains evaluable when Completion is partial or truncated and does not claim
// that omitted, unobserved subjects do not exist.
func CompletionObservedGroupsExactly(locators ...CompletionGroupLocator) CompletionExpectation {
	states := make([]*completionGroupLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionObservedGroupsExpectationValue{
		states: states,
	}
	return value
}

// CompletionObservedNativeRequestsExactly returns an expectation that the native requests present in the returned
// Completion snapshot are exactly the supplied locators in the same order.
// This is observed-snapshot exactness: it remains evaluable when Completion is partial or truncated and does not claim
// that omitted, unobserved subjects do not exist.
func CompletionObservedNativeRequestsExactly(locators ...CompletionNativeRequestLocator) CompletionExpectation {
	states := make([]*completionNativeRequestLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionObservedNativeRequestsExpectationValue{
		states: states,
	}
	return value
}

// CompletionObservedMessagesExactly returns an expectation that the messages present in the returned Completion
// snapshot exactly equal the supplied messages in order.
// This is observed-snapshot exactness, so partial or truncated Completion does not by itself make the expectation
// not provable. Failure reporting does not expose sensitive message text.
func CompletionObservedMessagesExactly(messages ...cli.CompletionMessage) CompletionExpectation {
	return completionObservedMessagesExpectation(slices.Clone(messages))
}

// CompletionNotPartial returns an expectation that the returned Completion does not carry the Partial marker.
func CompletionNotPartial() CompletionExpectation {
	expectation := completionBooleanExpectation{
		field:   completionBooleanPartial,
		want:    false,
		subject: "completion Partial",
	}
	return expectation
}

// CompletionNotTruncated returns an expectation that the returned Completion does not carry the Truncated marker.
func CompletionNotTruncated() CompletionExpectation {
	expectation := completionBooleanExpectation{
		field:   completionBooleanTruncated,
		want:    false,
		subject: "completion Truncated",
	}
	return expectation
}

// CompletionNoRefreshOnInput returns an expectation that the returned Completion does not request refresh on further
// input.
func CompletionNoRefreshOnInput() CompletionExpectation {
	expectation := completionBooleanExpectation{
		field:   completionBooleanRefreshOnInput,
		want:    false,
		subject: "completion RefreshOnInput",
	}
	return expectation
}

// CompletionObservedInputSourcesExactly returns a Completion expectation that the sources of kind present in the
// returned Completion source map have exactly the supplied distinct semantic members.
// This is observed-snapshot membership, not an assertion that a partial or truncated Completion contains every source
// that could exist. The map has no semantic iteration order.
func CompletionObservedInputSourcesExactly(kind cli.InputKind, sources ...InputSourceLocator) CompletionExpectation {
	expectation := completionObservedInputSourcesExpectation{
		kind:    kind,
		sources: slices.Clone(sources),
	}
	return expectation
}

// CompletionPartial checks that the Completion is partial.
func CompletionPartial() CompletionExpectation {
	expectation := completionBooleanExpectation{
		field:   completionBooleanPartial,
		want:    true,
		subject: "completion partiality",
	}
	return expectation
}

// CompletionTruncated checks that the Completion is truncated.
func CompletionTruncated() CompletionExpectation {
	expectation := completionBooleanExpectation{
		field:   completionBooleanTruncated,
		want:    true,
		subject: "completion truncation",
	}
	return expectation
}

// CompletionRefreshOnInput checks that the Completion requests refresh when
// input changes.
func CompletionRefreshOnInput() CompletionExpectation {
	expectation := completionBooleanExpectation{
		field:   completionBooleanRefreshOnInput,
		want:    true,
		subject: "completion refresh-on-input",
	}
	return expectation
}

// CompletionReplace checks the exact replacement span reported by Completion.
func CompletionReplace(span cli.CompletionSpan) CompletionExpectation {
	expectation := completionReplaceExpectation{
		span: span,
	}
	return expectation
}

// CompletionItemsExactly checks the complete Completion item collection in
// authoritative Commandment order. Exactness is not provable from an incomplete
// or truncated relevant domain.
func CompletionItemsExactly(locators ...CompletionItemLocator) CompletionExpectation {
	return completionItemsExactlyExpectation(locators)
}

// CompletionGroupsExactly checks the complete Completion group collection in
// authoritative Commandment order. Exactness requires a complete observation.
func CompletionGroupsExactly(locators ...CompletionGroupLocator) CompletionExpectation {
	return completionGroupsExactlyExpectation(locators)
}

// CompletionMessagesExactly checks the complete ordered Completion message
// collection. Exactness requires a complete observation.
func CompletionMessagesExactly(messages ...cli.CompletionMessage) CompletionExpectation {
	return completionMessagesExactlyExpectation(messages)
}

// CompletionNativeRequestsExactly checks the complete ordered native-request
// collection. Exactness requires a complete observation.
func CompletionNativeRequestsExactly(locators ...CompletionNativeRequestLocator) CompletionExpectation {
	return completionNativeRequestsExactlyExpectation(locators)
}

// CompletionRankingAlgorithm checks the ranking algorithm reported by
// Completion.
func CompletionRankingAlgorithm(algorithm cli.CompletionRankingAlgorithm) CompletionExpectation {
	expectation := completionRankingExpectation{
		algorithm: algorithm,
	}
	return expectation
}

// CompletionSpacing checks the spacing policy reported by Completion.
func CompletionSpacing(spacing cli.CompletionSpacing) CompletionExpectation {
	expectation := completionSpacingExpectation{
		spacing: spacing,
	}
	return expectation
}

// CompletionTargetsExactly checks the complete ordered Completion target
// collection. Exactness requires a complete observation.
func CompletionTargetsExactly(locators ...CompletionTargetLocator) CompletionExpectation {
	return completionTargetsExactlyExpectation(locators)
}

// On refines the target coordinate to the same Completion target as the supplied public subject coordinate. The
// relationship is proved only from subject identity exposed by the Completion observation; an unavailable subject
// coordinate remains not provable.
func (locator CompletionTargetLocator) On(subject CompletionTargetSubject) CompletionTargetLocator {
	return completionTargetOn(locator, subject)
}

// Present returns an expectation that the target coordinate resolves in the returned Completion.
func (locator CompletionTargetLocator) Present() CompletionExpectation {
	return completionTargetPresenceExpectation(locator, true)
}

// Kind returns an expectation that the located target has the exact CompletionTarget.Kind value.
func (locator CompletionTargetLocator) Kind(kind cli.CompletionTargetKind) CompletionExpectation {
	return completionTargetKindExpectation(locator, kind)
}

// Absent returns an expectation that the target coordinate is absent from a complete Completion observation. Partial or
// truncated observations cannot establish that absence.
func (locator CompletionTargetLocator) Absent() CompletionExpectation {
	return completionTargetPresenceExpectation(locator, false)
}

// Name returns an expectation that the located target carries exactly the observed completion-target name; it does not
// infer declaration identity from display text.
func (locator CompletionTargetLocator) Name(name string) CompletionExpectation {
	return completionTargetNameExpectation(locator, name)
}

// Spelling returns an expectation that the located target carries exactly the observed completion-target spelling.
func (locator CompletionTargetLocator) Spelling(spelling string) CompletionExpectation {
	return completionTargetSpellingExpectation(locator, spelling)
}

// CommandPathExactly returns an expectation that the located target carries exactly path, in order, as its Completion
// command path.
func (locator CompletionTargetLocator) CommandPathExactly(path ...string) CompletionExpectation {
	return completionTargetCommandPathExpectation(locator, path)
}

// TermIndex returns an expectation that the located target carries exactly index as its argument-term index.
func (locator CompletionTargetLocator) TermIndex(index int) CompletionExpectation {
	return completionTargetTermIndexExpectation(locator, index)
}

// FlagValueForm returns an expectation that the located target carries exactly form as its flag-value form.
func (locator CompletionTargetLocator) FlagValueForm(form cli.FlagValueForm) CompletionExpectation {
	return completionTargetFlagValueFormExpectation(locator, form)
}

// Occurrence returns an expectation that the located target carries exactly index as its occurrence coordinate.
func (locator CompletionTargetLocator) Occurrence(index int) CompletionExpectation {
	return completionTargetOccurrenceExpectation(locator, index)
}

// OnSelectedCommand returns an expectation that the located target carries a command identity equal to the Completion
// selection's selected command identity.
// The comparison uses identities from the same Completion observation, including operation-local materialized command
// identities; it does not require a Schema lookup.
func (locator CompletionTargetLocator) OnSelectedCommand() CompletionExpectation {
	expectation := completionTargetIdentityExpectation{
		locator: locator.state,
		kind:    uint8(completionTargetIdentitySelectedCommand),
	}
	return expectation
}

// CommandIdentityPresent returns an expectation that the located completion target carries a nonzero command identity.
func (locator CompletionTargetLocator) CommandIdentityPresent() CompletionExpectation {
	expectation := completionTargetIdentityExpectation{
		locator: locator.state,
		kind:    uint8(completionTargetIdentityCommand),
	}
	return expectation
}

// ValueIdentityPresent returns an expectation that the located completion target carries a nonzero value identity.
func (locator CompletionTargetLocator) ValueIdentityPresent() CompletionExpectation {
	expectation := completionTargetIdentityExpectation{
		locator: locator.state,
		kind:    uint8(completionTargetIdentityValue),
	}
	return expectation
}

// TargetAt returns the target at index in the located item's ordered TargetIndexes relationship, preserving the target
// identity carried by the same Completion observation.
func (locator CompletionItemLocator) TargetAt(index int) CompletionTargetLocator {
	return completionItemTargetAt(locator, index)
}

// Present returns an expectation that the item coordinate resolves in the returned Completion.
func (locator CompletionItemLocator) Present() CompletionExpectation {
	return completionItemPresenceExpectation(locator, true)
}

// Value returns an expectation that the located item's semantic insertion value exactly equals value.
func (locator CompletionItemLocator) Value(value string) CompletionExpectation {
	return completionItemValueExpectation(locator, value)
}

// DisplayText returns an expectation that the located item's observed presentation text exactly equals text.
func (locator CompletionItemLocator) DisplayText(text string) CompletionExpectation {
	return completionItemDisplayTextExpectation(locator, text)
}

// Description returns an expectation that the located item's observed description exactly equals description.
func (locator CompletionItemLocator) Description(description string) CompletionExpectation {
	return completionItemDescriptionExpectation(locator, description)
}

// Detail returns an expectation that the located item's observed detail exactly equals detail.
func (locator CompletionItemLocator) Detail(detail string) CompletionExpectation {
	return completionItemDetailExpectation(locator, detail)
}

// SearchTermsExactly returns an expectation that the located item's search terms exactly equal terms in authoritative
// order.
func (locator CompletionItemLocator) SearchTermsExactly(terms ...string) CompletionExpectation {
	return completionItemSearchTermsExpectation(locator, terms)
}

// SortText returns an expectation that the located item's sort text exactly equals text.
func (locator CompletionItemLocator) SortText(text string) CompletionExpectation {
	return completionItemSortTextExpectation(locator, text)
}

// CommitCharactersExactly returns an expectation that the located item's commit characters exactly equal characters in
// order.
func (locator CompletionItemLocator) CommitCharactersExactly(characters ...rune) CompletionExpectation {
	return completionItemCommitCharactersExpectation(locator, characters)
}

// ReplacementExactly returns an expectation that the located item's replacement sequence exactly equals values in
// order.
func (locator CompletionItemLocator) ReplacementExactly(values ...string) CompletionExpectation {
	return completionItemReplacementExpectation(locator, values)
}

// Absent returns an expectation that the item coordinate is absent from a complete Completion observation. Partial or
// truncated observations cannot establish that absence.
func (locator CompletionItemLocator) Absent() CompletionExpectation {
	return completionItemPresenceExpectation(locator, false)
}

// Kind returns an expectation that the located item has the exact CompletionCandidateKind.
func (locator CompletionItemLocator) Kind(kind cli.CompletionCandidateKind) CompletionExpectation {
	return completionItemKindExpectation(locator, kind)
}

// Recommended returns an expectation that the located item is marked recommended.
func (locator CompletionItemLocator) Recommended() CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyRecommended,
	}
	return expectation
}

// FallbackOnly returns an expectation that the located item is marked fallback-only.
func (locator CompletionItemLocator) FallbackOnly() CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyFallbackOnly,
	}
	return expectation
}

// Lifecycle returns an expectation that the located item carries exactly the observed lifecycle state. It checks
// Completion output, not declaration lifecycle through a separate artifact.
func (locator CompletionItemLocator) Lifecycle(state cli.LifecycleState) CompletionExpectation {
	return completionItemLifecycleExpectation(locator, state)
}

// Sensitive returns an expectation that the located item is marked sensitive. Once an item is sensitive, clitest
// redacts expected and observed item-property details on mismatches rather than exposing the protected value.
func (locator CompletionItemLocator) Sensitive() CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertySensitive,
	}
	return expectation
}

// Required returns an expectation that the located item is marked required.
func (locator CompletionItemLocator) Required() CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyRequired,
	}
	return expectation
}

// Conditional returns an expectation that the located item is marked conditional.
func (locator CompletionItemLocator) Conditional() CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyConditional,
	}
	return expectation
}

// Match returns an expectation that the located item carries exactly kind as its Completion match kind.
func (locator CompletionItemLocator) Match(kind cli.CompletionMatchKind) CompletionExpectation {
	return completionItemMatchExpectation(locator, kind)
}

// MatchField returns an expectation that the located item's match metadata identifies exactly field.
func (locator CompletionItemLocator) MatchField(field cli.CompletionMatchField) CompletionExpectation {
	return completionItemMatchFieldExpectation(locator, field)
}

// MatchFieldIndex returns an expectation that the located item's match metadata carries exactly index as the
// matched-field index.
func (locator CompletionItemLocator) MatchFieldIndex(index int) CompletionExpectation {
	return completionItemMatchFieldIndexExpectation(locator, index)
}

// MatchScore returns an expectation that the located item's match metadata carries exactly score.
func (locator CompletionItemLocator) MatchScore(score int64) CompletionExpectation {
	return completionItemMatchScoreExpectation(locator, score)
}

// MatchRangesExactly returns an expectation that the located item's match ranges exactly equal ranges in authoritative
// order.
func (locator CompletionItemLocator) MatchRangesExactly(ranges ...cli.CompletionRange) CompletionExpectation {
	return completionItemMatchRangesExpectation(locator, ranges)
}

// MatchEdits returns an expectation that the located item's match metadata carries exactly edits as its edit count.
func (locator CompletionItemLocator) MatchEdits(edits int) CompletionExpectation {
	return completionItemMatchEditsExpectation(locator, edits)
}

// Declaration returns an expectation that CompletionItem.Declaration exactly equals the declaration identity carried by
// v. The expected locator must already be an exact declaration identity, such as one rebound by ResolveDeclaration;
// this expectation does not perform an independent declaration lookup.
func (locator CompletionItemLocator) Declaration(v DeclarationIdentityLocator) CompletionExpectation {
	return completionItemDeclarationExpectation(locator, v)
}

// Group returns an expectation that the located item belongs to the uniquely identifiable Completion group selected by
// group. Group membership is not inferred when the observed group ID is empty or non-unique.
func (locator CompletionItemLocator) Group(group CompletionGroupLocator) CompletionExpectation {
	return completionItemGroupExpectation(locator, group)
}

// TargetsExactly returns an expectation that the located item's target relationships exactly equal targets in the same
// order. Exactness is not provable from a partial or truncated Completion.
func (locator CompletionItemLocator) TargetsExactly(targets ...CompletionTargetLocator) CompletionExpectation {
	return completionItemTargetsExactlyExpectation(locator, targets)
}

// Priority returns an expectation that the located item carries exactly priority.
func (locator CompletionItemLocator) Priority(priority int) CompletionExpectation {
	return completionItemPriorityExpectation(locator, priority)
}

// Spacing returns an expectation that the located item carries exactly spacing as its completion spacing policy.
func (locator CompletionItemLocator) Spacing(spacing cli.CompletionSpacing) CompletionExpectation {
	return completionItemSpacingExpectation(locator, spacing)
}

// DeclarationKind returns an expectation that the located completion item has the requested declaration kind and the
// corresponding typed declaration identity is nonzero.
// It tests the exact heterogeneous CompletionItem.Declaration relation; it does not substitute a parent or related
// declaration identity.
func (locator CompletionItemLocator) DeclarationKind(expected cli.DeclarationKind) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:         locator.state,
		property:        completionItemPropertyDeclarationKind,
		declarationKind: expected,
	}
	return expectation
}

// Ungrouped returns an expectation that the located completion item has no completion-group relation.
func (locator CompletionItemLocator) Ungrouped() CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyUngrouped,
	}
	return expectation
}

// Presentation returns a Completion expectation that checks the located item's
// display text and description. It does not define completion ranking or
// matching semantics.
func (locator CompletionItemLocator) Presentation(displayText, description string) CompletionExpectation {
	expectation := completionItemPresentationExpectation{
		locator:     locator.state,
		displayText: displayText,
		description: description,
	}
	return expectation
}

// Target returns the target referenced by the located native request's TargetIndex in the same Completion observation.
func (locator CompletionNativeRequestLocator) Target() CompletionTargetLocator {
	return completionNativeRequestTarget(locator)
}

// Kind returns an expectation that the located native request has exactly kind.
func (locator CompletionNativeRequestLocator) Kind(kind cli.CompletionNativeKind) CompletionExpectation {
	expectation := completionNativeRequestPropertyExpectation{
		locator:  locator.state,
		property: completionNativeRequestPropertyKind,
		kind:     kind,
	}
	return expectation
}

// Mode returns an expectation that the located native request has exactly mode.
func (locator CompletionNativeRequestLocator) Mode(mode cli.CompletionNativeMode) CompletionExpectation {
	expectation := completionNativeRequestPropertyExpectation{
		locator:  locator.state,
		property: completionNativeRequestPropertyMode,
		mode:     mode,
	}
	return expectation
}

// Required returns an expectation that the located native request is marked required.
func (locator CompletionNativeRequestLocator) Required() CompletionExpectation {
	expectation := completionNativeRequestPropertyExpectation{
		locator:  locator.state,
		property: completionNativeRequestPropertyRequired,
	}
	return expectation
}

// Conditional returns an expectation that the located native request is marked conditional.
func (locator CompletionNativeRequestLocator) Conditional() CompletionExpectation {
	expectation := completionNativeRequestPropertyExpectation{
		locator:  locator.state,
		property: completionNativeRequestPropertyConditional,
	}
	return expectation
}

// Present returns an expectation that the group coordinate resolves in the returned Completion.
func (locator CompletionGroupLocator) Present() CompletionExpectation {
	return completionGroupPresenceExpectation(locator, true)
}

// ID returns an expectation that the located group's observed ID exactly equals id.
func (locator CompletionGroupLocator) ID(id string) CompletionExpectation {
	return completionGroupIDExpectation(locator, id)
}

// Absent returns an expectation that the group coordinate is absent from a complete Completion observation. Partial or
// truncated observations cannot establish that absence.
func (locator CompletionGroupLocator) Absent() CompletionExpectation {
	return completionGroupPresenceExpectation(locator, false)
}

// Priority returns an expectation that the located group carries exactly priority.
func (locator CompletionGroupLocator) Priority(priority int) CompletionExpectation {
	return completionGroupPriorityExpectation(locator, priority)
}

// Label returns an expectation that the located group's observed presentation label exactly equals label.
func (locator CompletionGroupLocator) Label(label string) CompletionExpectation {
	return completionGroupLabelExpectation(locator, label)
}

// ItemsExactly returns an expectation that the located group contains exactly items in Completion.Items authoritative
// order. Exactness is not provable from a partial or truncated Completion.
func (locator CompletionGroupLocator) ItemsExactly(items ...CompletionItemLocator) CompletionExpectation {
	return completionGroupItemsExactlyExpectation(locator, items)
}

// Sensitive returns an expectation that the located group is marked sensitive. Mismatched group-property diagnostics
// redact expected and observed property details when the observed group is sensitive.
func (locator CompletionGroupLocator) Sensitive() CompletionExpectation {
	return completionGroupSensitiveExpectation(locator)
}

// Present returns an expectation that the returned Completion carries a delegation relationship.
func (locator CompletionDelegationLocator) Present() CompletionExpectation {
	expectation := completionDelegationExpectation{
		locator:  locator.state,
		property: completionDelegationPropertyPresent,
		present:  true,
	}
	return expectation
}

// Absent returns an expectation that a complete Completion carries no delegation relationship. A missing delegation in
// a partial or truncated Completion is not expectation of absence.
func (locator CompletionDelegationLocator) Absent() CompletionExpectation {
	expectation := completionDelegationExpectation{
		locator:  locator.state,
		property: completionDelegationPropertyPresent,
		present:  false,
	}
	return expectation
}

// ArgsExactly returns an expectation that the delegation arguments exactly equal args in order.
func (locator CompletionDelegationLocator) ArgsExactly(args ...string) CompletionExpectation {
	expectation := completionDelegationExpectation{
		locator:  locator.state,
		property: completionDelegationPropertyArgs,
		args:     slices.Clone(args),
	}
	return expectation
}

// Cursor returns an expectation that the delegation carries exactly cursor.
func (locator CompletionDelegationLocator) Cursor(cursor cli.CompletionCursor) CompletionExpectation {
	expectation := completionDelegationExpectation{
		locator:  locator.state,
		property: completionDelegationPropertyCursor,
		cursor:   cursor,
	}
	return expectation
}

// Sensitive returns an expectation that the delegation is present and marked sensitive. When a sensitive delegation
// mismatches, clitest redacts both expected and observed delegation-property details.
func (locator CompletionDelegationLocator) Sensitive() CompletionExpectation {
	expectation := completionDelegationExpectation{
		locator:  locator.state,
		property: completionDelegationPropertySensitive,
	}
	return expectation
}

func (CompletionCase) conformanceCase() {}

// Dynamic returns a reusable declaration expectation that checks the public dynamic completion
// capability exposed by the located completion declaration.
func (locator CompletionDeclarationLocator) Dynamic() DeclarationExpectation {
	value := completionDeclarationExpectationValue{
		locator:  locator.state,
		property: completionDeclarationExpectationDynamic,
	}
	return value
}

// Delegated returns a reusable declaration expectation that checks the public delegated
// completion capability exposed by the located completion declaration.
func (locator CompletionDeclarationLocator) Delegated() DeclarationExpectation {
	value := completionDeclarationExpectationValue{
		locator:  locator.state,
		property: completionDeclarationExpectationDelegated,
	}
	return value
}

// Native returns a declaration expectation that checks the located completion declaration
// exposes the expected native completion kind and mode as one public capability.
func (locator CompletionDeclarationLocator) Native(kind cli.CompletionNativeKind, mode cli.CompletionNativeMode) DeclarationExpectation {
	value := completionDeclarationExpectationValue{
		locator:    locator.state,
		property:   completionDeclarationExpectationNative,
		nativeKind: kind,
		nativeMode: mode,
	}
	return value
}

// DependenciesExactly returns a declaration expectation that checks the completion
// declaration's complete dependency list by public value identity and authoritative order.
func (locator CompletionDeclarationLocator) DependenciesExactly(values ...ValueLocator) DeclarationExpectation {
	dependencies := make([]*valueLocatorState, len(values))
	for index, value := range values {
		dependencies[index] = value.state
	}
	expectation := completionDeclarationExpectationValue{
		locator:      locator.state,
		property:     completionDeclarationExpectationDependencies,
		dependencies: dependencies,
	}
	return expectation
}

const completionTargetLabel = "completion target"

type completionTargetResolution struct {
	status locatorResolutionStatus
	value  cli.CompletionTarget
	index  int
	label  evaluationField
	actual evaluationField
}

type completionEntryResolution[Entry any] struct {
	status locatorResolutionStatus
	value  Entry
	index  int
	label  evaluationField
	actual evaluationField
}

type (
	completionItemResolution  = completionEntryResolution[cli.CompletionItem]
	completionGroupResolution = completionEntryResolution[cli.CompletionGroup]
)

type completionNativeRequestResolution struct {
	status locatorResolutionStatus
	value  cli.CompletionNativeRequest
	index  int
	label  evaluationField
	actual evaluationField
}

type completionTargetLocatorMode uint8

const (
	completionTargetLocatorInvalid completionTargetLocatorMode = iota
	completionTargetLocatorKind
	completionTargetLocatorIndex
	completionTargetLocatorItemIndex
	completionTargetLocatorNativeRequest
)

type completionTargetLocatorState struct {
	mode  completionTargetLocatorMode
	kind  cli.CompletionTargetKind
	index int

	item   *completionItemLocatorState
	native *completionNativeRequestLocatorState

	subjectSet bool
	subject    CompletionTargetSubject
}

type completionItemLocatorMode uint8

const (
	completionItemLocatorInvalid completionItemLocatorMode = iota
	completionItemLocatorValue
	completionItemLocatorIndex
)

type completionItemLocatorState struct {
	mode  completionItemLocatorMode
	value string
	index int
}

type completionGroupLocatorMode uint8

const (
	completionGroupLocatorInvalid completionGroupLocatorMode = iota
	completionGroupLocatorID
	completionGroupLocatorIndex
)

type completionGroupLocatorState struct {
	mode  completionGroupLocatorMode
	id    string
	index int
}

type completionNativeRequestLocatorState struct {
	valid bool
	index int
}

type completionDelegationLocatorState struct {
	valid bool
}

type completionDeclarationLocatorMode uint8

const (
	completionDeclarationLocatorInvalid completionDeclarationLocatorMode = iota
	completionDeclarationLocatorFlag
	completionDeclarationLocatorArgumentTerm
)

type completionDeclarationLocatorState struct {
	mode completionDeclarationLocatorMode
	flag *flagLocatorState
	term *argumentTermLocatorState
}

type completionDeclarationExpectationProperty uint8

const (
	completionDeclarationExpectationInvalid completionDeclarationExpectationProperty = iota
	completionDeclarationExpectationDynamic
	completionDeclarationExpectationDelegated
	completionDeclarationExpectationNative
	completionDeclarationExpectationDependencies
)

type completionDeclarationExpectationValue struct {
	locator      *completionDeclarationLocatorState
	property     completionDeclarationExpectationProperty
	nativeKind   cli.CompletionNativeKind
	nativeMode   cli.CompletionNativeMode
	dependencies []*valueLocatorState
}

type completionTargetProperty uint8

const (
	completionTargetPropertyInvalid completionTargetProperty = iota
	completionTargetPropertyKind
	completionTargetPropertyName
	completionTargetPropertySpelling
	completionTargetPropertyCommandPath
	completionTargetPropertyTermIndex
	completionTargetPropertyFlagValueForm
	completionTargetPropertyOccurrence
)

type completionTargetPropertyExpectation struct {
	locator   *completionTargetLocatorState
	property  completionTargetProperty
	kind      cli.CompletionTargetKind
	text      string
	path      []string
	index     int
	valueForm cli.FlagValueForm
}

type completionTargetPresenceExpectationValue struct {
	locator *completionTargetLocatorState
	present bool
}

type completionItemPresenceExpectationValue struct {
	locator *completionItemLocatorState
	present bool
}

type completionItemProperty uint8

const (
	completionItemPropertyInvalid completionItemProperty = iota
	completionItemPropertyValue
	completionItemPropertyReplacement
	completionItemPropertyKind
	completionItemPropertyRecommended
	completionItemPropertyFallbackOnly
	completionItemPropertyLifecycle
	completionItemPropertySensitive
	completionItemPropertyRequired
	completionItemPropertyConditional
	completionItemPropertyMatchKind
	completionItemPropertyPriority
	completionItemPropertySpacing
	completionItemPropertyDeclaration
	completionItemPropertyDisplayText
	completionItemPropertyDescription
	completionItemPropertyDetail
	completionItemPropertySearchTerms
	completionItemPropertySortText
	completionItemPropertyCommitCharacters
	completionItemPropertyMatchField
	completionItemPropertyMatchFieldIndex
	completionItemPropertyMatchScore
	completionItemPropertyMatchRanges
	completionItemPropertyMatchEdits
	completionItemPropertyDeclarationKind
	completionItemPropertyUngrouped
)

type completionItemPropertyExpectation struct {
	locator          *completionItemLocatorState
	property         completionItemProperty
	text             string
	strings          []string
	runes            []rune
	kind             cli.CompletionCandidateKind
	lifecycle        cli.LifecycleState
	matchKind        cli.CompletionMatchKind
	integer          int
	spacing          cli.CompletionSpacing
	declaration      cli.DeclarationIdentity
	declarationValid bool
	matchField       cli.CompletionMatchField
	declarationKind  cli.DeclarationKind
	score            int64
	ranges           []cli.CompletionRange
}

type completionItemGroupExpectationValue struct {
	item  *completionItemLocatorState
	group *completionGroupLocatorState
}

type completionItemTargetsExpectationValue struct {
	item    *completionItemLocatorState
	targets []*completionTargetLocatorState
}

type completionGroupPresenceExpectationValue struct {
	locator *completionGroupLocatorState
	present bool
}

type completionGroupProperty uint8

const (
	completionGroupPropertyInvalid completionGroupProperty = iota
	completionGroupPropertyID
	completionGroupPropertyPriority
	completionGroupPropertySensitive
	completionGroupPropertyLabel
)

type completionGroupPropertyExpectation struct {
	locator  *completionGroupLocatorState
	property completionGroupProperty
	text     string
	priority int
}

type completionNativeRequestProperty uint8

const (
	completionNativeRequestPropertyInvalid completionNativeRequestProperty = iota
	completionNativeRequestPropertyKind
	completionNativeRequestPropertyMode
	completionNativeRequestPropertyRequired
	completionNativeRequestPropertyConditional
)

type completionNativeRequestPropertyExpectation struct {
	locator  *completionNativeRequestLocatorState
	property completionNativeRequestProperty
	kind     cli.CompletionNativeKind
	mode     cli.CompletionNativeMode
}

type completionDelegationProperty uint8

const (
	completionDelegationPropertyInvalid completionDelegationProperty = iota
	completionDelegationPropertyPresent
	completionDelegationPropertyArgs
	completionDelegationPropertyCursor
	completionDelegationPropertySensitive
)

type completionDelegationExpectation struct {
	locator  *completionDelegationLocatorState
	property completionDelegationProperty
	present  bool
	args     []string
	cursor   cli.CompletionCursor
}

type completionTargetsExactlyExpectationValue struct {
	states []*completionTargetLocatorState
}

type completionItemsExactlyExpectationValue struct {
	states []*completionItemLocatorState
}

type completionGroupsExactlyExpectationValue struct {
	states []*completionGroupLocatorState
}

type completionNativeRequestsExactlyExpectationValue struct {
	states []*completionNativeRequestLocatorState
}

type completionGroupItemsExpectation struct {
	group *completionGroupLocatorState
	items []*completionItemLocatorState
}

type completionMessagesExpectation struct {
	messages []cli.CompletionMessage
}

type completionObservedTargetsExpectationValue struct {
	states []*completionTargetLocatorState
}

type completionObservedItemsExpectationValue struct {
	states []*completionItemLocatorState
}

type completionObservedGroupsExpectationValue struct {
	states []*completionGroupLocatorState
}

type completionObservedNativeRequestsExpectationValue struct {
	states []*completionNativeRequestLocatorState
}

type completionObservedMessagesExpectation []cli.CompletionMessage

type completionTargetIdentityExpectation struct {
	locator *completionTargetLocatorState
	kind    uint8
}

const (
	completionTargetIdentityCommand completionTargetIdentityExpectationKind = iota + 1
	completionTargetIdentityValue
	completionTargetIdentitySelectedCommand
)

type completionTargetIdentityExpectationKind uint8

type completionObservedInputSourcesExpectation struct {
	kind    cli.InputKind
	sources []InputSourceLocator
}

type completionBooleanField uint8

const (
	completionBooleanInvalid completionBooleanField = iota
	completionBooleanPartial
	completionBooleanTruncated
	completionBooleanRefreshOnInput
)

type completionBooleanExpectation struct {
	field   completionBooleanField
	want    bool
	subject string
}

type completionReplaceExpectation struct{ span cli.CompletionSpan }

type completionRankingExpectation struct {
	algorithm cli.CompletionRankingAlgorithm
}

type completionSpacingExpectation struct{ spacing cli.CompletionSpacing }

func completionTargetKindLocator(kind cli.CompletionTargetKind) CompletionTargetLocator {
	state := &completionTargetLocatorState{
		mode: completionTargetLocatorKind,
		kind: kind,
	}
	locator := CompletionTargetLocator{
		state: state,
	}
	return locator
}

func completionTargetIndexLocator(index int) CompletionTargetLocator {
	state := &completionTargetLocatorState{
		mode:  completionTargetLocatorIndex,
		index: index,
	}
	locator := CompletionTargetLocator{
		state: state,
	}
	return locator
}

func completionItemValueLocator(value string) CompletionItemLocator {
	state := &completionItemLocatorState{
		mode:  completionItemLocatorValue,
		value: value,
	}
	locator := CompletionItemLocator{
		state: state,
	}
	return locator
}

func completionItemIndexLocator(index int) CompletionItemLocator {
	state := &completionItemLocatorState{
		mode:  completionItemLocatorIndex,
		index: index,
	}
	locator := CompletionItemLocator{
		state: state,
	}
	return locator
}

func completionGroupIDLocator(id string) CompletionGroupLocator {
	state := &completionGroupLocatorState{
		mode: completionGroupLocatorID,
		id:   id,
	}
	locator := CompletionGroupLocator{
		state: state,
	}
	return locator
}

func completionGroupIndexLocator(index int) CompletionGroupLocator {
	state := &completionGroupLocatorState{
		mode:  completionGroupLocatorIndex,
		index: index,
	}
	locator := CompletionGroupLocator{
		state: state,
	}
	return locator
}

func completionNativeRequestIndexLocator(index int) CompletionNativeRequestLocator {
	state := &completionNativeRequestLocatorState{
		valid: true,
		index: index,
	}
	locator := CompletionNativeRequestLocator{
		state: state,
	}
	return locator
}

func completionDelegationLocator() CompletionDelegationLocator {
	state := &completionDelegationLocatorState{
		valid: true,
	}
	locator := CompletionDelegationLocator{
		state: state,
	}
	return locator
}

func commandEntrySelectionStepLocator(entry CommandEntryLocator) SelectionStepLocator {
	if entry.state == nil {
		return SelectionStepLocator{}
	}

	state := &selectionStepLocatorState{
		kind:  selectionStepLocatorEntry,
		entry: entry.state,
	}

	locator := SelectionStepLocator{
		state: state,
	}
	return locator
}

func commandDefaultSelectionStepLocator(commandDefault CommandDefaultLocator) SelectionStepLocator {
	if commandDefault.state == nil {
		return SelectionStepLocator{}
	}

	step := &selectionStepLocatorState{
		kind:           selectionStepLocatorDefault,
		commandDefault: commandDefault.state,
	}

	locator := SelectionStepLocator{
		state: step,
	}
	return locator
}

func selectionStepFrom(step SelectionStepLocator, origin InputOriginLocator) SelectionStepLocator {
	if step.state == nil {
		return SelectionStepLocator{}
	}

	state := *step.state
	state.commands = slices.Clone(step.state.commands)
	state.origin = origin.state
	if state.origin == nil {
		return SelectionStepLocator{}
	}

	locator := SelectionStepLocator{
		state: &state,
	}
	return locator
}

func selectionStepSelects(step SelectionStepLocator, commands []CommandLocator) SelectionStepLocator {
	if step.state == nil {
		return SelectionStepLocator{}
	}

	states := make([]*commandLocatorState, len(commands))
	for index, command := range commands {
		if command.state == nil {
			return SelectionStepLocator{}
		}
		states[index] = command.state
	}

	state := *step.state
	state.commands = states

	locator := SelectionStepLocator{
		state: &state,
	}
	return locator
}

func completionTargetOn(target CompletionTargetLocator, subject CompletionTargetSubject) CompletionTargetLocator {
	if target.state == nil {
		return CompletionTargetLocator{}
	}
	state := *target.state
	state.subjectSet = true
	state.subject = subject
	locator := CompletionTargetLocator{
		state: &state,
	}
	return locator
}

func completionItemTargetAt(item CompletionItemLocator, index int) CompletionTargetLocator {
	if item.state == nil {
		return CompletionTargetLocator{}
	}
	state := &completionTargetLocatorState{
		mode:  completionTargetLocatorItemIndex,
		item:  item.state,
		index: index,
	}
	locator := CompletionTargetLocator{
		state: state,
	}
	return locator
}

func completionNativeRequestTarget(request CompletionNativeRequestLocator) CompletionTargetLocator {
	if request.state == nil {
		return CompletionTargetLocator{}
	}
	state := &completionTargetLocatorState{
		mode:   completionTargetLocatorNativeRequest,
		native: request.state,
	}
	locator := CompletionTargetLocator{
		state: state,
	}
	return locator
}

func completionObservationIncomplete(result cli.Completion) bool {
	return result.Partial || result.Truncated
}

func completionObservationIncompleteReason(result cli.Completion) string {
	switch {
	case result.Partial && result.Truncated:
		return "completion observation is partial and truncated"
	case result.Partial:
		return "completion observation is partial"
	case result.Truncated:
		return "completion observation is truncated"
	default:
		return "completion observation is complete"
	}
}

func completeMiss(incomplete bool, label evaluationField, absent string) (locatorResolutionStatus, evaluationField) {
	if incomplete {
		return locatorResolutionNotProvable, qualifySubject(label, evaluationText("is not present in an incomplete completion observation"))
	}
	return locatorResolutionAbsent, evaluationText(absent)
}

func resolveCompletionTarget(state *completionTargetLocatorState, result cli.Completion) completionTargetResolution {
	var resolution completionTargetResolution
	if state == nil || state.mode == completionTargetLocatorInvalid {
		resolution = completionTargetResolution{
			status: locatorResolutionInvalid,
			label:  evaluationText(completionTargetLabel),
			actual: evaluationText("invalid completion-target locator"),
		}
		return resolution
	}

	switch state.mode {
	case completionTargetLocatorItemIndex:
		if state.item == nil || state.index < 0 {
			resolution = completionTargetResolution{
				status: locatorResolutionInvalid,
				label:  evaluationText(completionTargetLabel),
				actual: evaluationText("invalid item-target coordinate"),
			}
			return resolution
		}
		item := resolveCompletionItem(state.item, result)
		label := indexedSubject(evaluationText("target"), state.index, item.label)
		if item.status != locatorResolutionFound {
			resolution = completionTargetResolution{
				status: item.status,
				label:  label,
				actual: item.actual,
			}
			return resolution
		}
		if state.index >= len(item.value.TargetIndexes) {
			status, actual := completeMiss(completionObservationIncomplete(result), label, "item target relationship is absent")
			resolution = completionTargetResolution{
				status: status,
				label:  label,
				actual: actual,
			}
			return resolution
		}
		absolute := item.value.TargetIndexes[state.index]
		if absolute < 0 || absolute >= len(result.Targets) {
			resolution = completionTargetResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("item references an unavailable completion target"),
			}
			return resolution
		}
		resolution = completionTargetResolution{
			status: locatorResolutionFound,
			value:  result.Targets[absolute],
			index:  absolute,
			label:  label,
		}
		return resolution

	case completionTargetLocatorNativeRequest:
		if state.native == nil {
			resolution = completionTargetResolution{
				status: locatorResolutionInvalid,
				label:  evaluationText(completionTargetLabel),
				actual: evaluationText("invalid native-request target coordinate"),
			}
			return resolution
		}
		request := resolveCompletionNativeRequest(state.native, result)
		label := qualifySubject(evaluationText("target of"), request.label)
		if request.status != locatorResolutionFound {
			resolution = completionTargetResolution{
				status: request.status,
				label:  label,
				actual: request.actual,
			}
			return resolution
		}
		absolute := request.value.TargetIndex
		if absolute < 0 || absolute >= len(result.Targets) {
			resolution = completionTargetResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("native request references an unavailable completion target"),
			}
			return resolution
		}
		resolution = completionTargetResolution{
			status: locatorResolutionFound,
			value:  result.Targets[absolute],
			index:  absolute,
			label:  label,
		}
		return resolution
	default:
	}

	matches := make([]int, 0, 1)
	var label evaluationField = evaluationText(completionTargetLabel)
	switch state.mode {
	case completionTargetLocatorIndex:
		label = subjectAt(evaluationText("completion target"), state.index)
		if state.index < 0 {
			resolution = completionTargetResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("invalid completion-target index"),
			}
			return resolution
		}
		if state.index < len(result.Targets) {
			matches = append(matches, state.index)
		}
	case completionTargetLocatorKind:
		label = evaluationText(fmt.Sprintf("completion target kind %v", state.kind))
		for index, target := range result.Targets {
			if target.Kind == state.kind {
				matches = append(matches, index)
			}
		}
	default:
		resolution = completionTargetResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid completion-target locator"),
		}
		return resolution
	}

	if state.subjectSet {
		filtered := matches[:0]
		for _, index := range matches {
			matched, status := completionTargetMatchesSubject(result.Targets[index], state.subject)
			if status != locatorResolutionFound {
				detail := evaluationText("completion observation does not expose the requested target-subject coordinate")
				if status == locatorResolutionInvalid {
					detail = evaluationText("invalid completion-target subject locator")
				}
				resolution = completionTargetResolution{
					status: status,
					label:  label,
					actual: detail,
				}
				return resolution
			}
			if matched {
				filtered = append(filtered, index)
			}
		}
		matches = filtered
	}

	if len(matches) == 1 {
		resolution = completionTargetResolution{
			status: locatorResolutionFound,
			value:  result.Targets[matches[0]],
			index:  matches[0],
			label:  label,
		}
		return resolution
	}
	if len(matches) > 1 {
		resolution = completionTargetResolution{
			status: locatorResolutionAmbiguous,
			label:  label,
			actual: collectionCardinalityField{
				name:  "matches completion-target indexes",
				count: len(matches),
			},
		}
		return resolution
	}
	status, actual := completeMiss(completionObservationIncomplete(result), label, "completion target is absent")
	resolution = completionTargetResolution{
		status: status,
		label:  label,
		actual: actual,
	}
	return resolution
}

func completionTargetMatchesSubject(target cli.CompletionTarget, subject CompletionTargetSubject) (bool, locatorResolutionStatus) {
	switch subject := subject.(type) {
	case CommandLocator:
		if subject.state == nil {
			return false, locatorResolutionInvalid
		}
		switch subject.state.mode {
		case commandLocatorID:
			return target.Command == subject.state.id, locatorResolutionFound
		case commandLocatorPath, commandLocatorChild:
			path, ok := commandPathCoordinate(subject.state)
			if !ok {
				return false, locatorResolutionInvalid
			}
			return slices.Equal(target.CommandPath, path), locatorResolutionFound
		default:
			return false, locatorResolutionInvalid
		}
	case ArgumentFormLocator:
		if subject.state == nil {
			return false, locatorResolutionInvalid
		}
		if subject.state.mode != argumentFormLocatorID {
			return false, locatorResolutionNotProvable
		}
		return target.ArgumentFormID == subject.state.id, locatorResolutionFound
	case ValueLocator:
		if subject.state == nil {
			return false, locatorResolutionInvalid
		}
		if subject.state.mode != valueLocatorID {
			return false, locatorResolutionNotProvable
		}
		return target.ValueID == subject.state.id, locatorResolutionFound
	case FlagLocator, ArgumentTermLocator:
		return false, locatorResolutionNotProvable
	default:
		return false, locatorResolutionInvalid
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveCompletionItem(state *completionItemLocatorState, result cli.Completion) completionItemResolution {
	var resolution completionItemResolution
	if state == nil || state.mode == completionItemLocatorInvalid {
		resolution = completionItemResolution{
			status: locatorResolutionInvalid,
			label:  evaluationText("completion item"),
			actual: evaluationText("invalid completion-item locator"),
		}
		return resolution
	}
	matches := make([]int, 0, 1)
	var label evaluationField = evaluationText("completion item")
	switch state.mode {
	case completionItemLocatorIndex:
		label = subjectAt(evaluationText("completion item"), state.index)
		if state.index < 0 {
			resolution = completionItemResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("invalid completion-item index"),
			}
			return resolution
		}
		if state.index < len(result.Items) {
			matches = append(matches, state.index)
		}
	case completionItemLocatorValue:
		label = subjectNamed(evaluationText("completion item"), state.value)
		for index, item := range result.Items {
			if item.Value == state.value {
				matches = append(matches, index)
			}
		}
	default:
		resolution = completionItemResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid completion-item locator"),
		}
		return resolution
	}
	if len(matches) == 1 {
		resolution = completionItemResolution{
			status: locatorResolutionFound,
			value:  result.Items[matches[0]],
			index:  matches[0],
			label:  label,
		}
		return resolution
	}
	if len(matches) > 1 {
		resolution = completionItemResolution{
			status: locatorResolutionAmbiguous,
			label:  label,
			actual: collectionCardinalityField{
				name:  "matches completion-item indexes",
				count: len(matches),
			},
		}
		return resolution
	}
	status, actual := completeMiss(completionObservationIncomplete(result), label, "completion item is absent")
	resolution = completionItemResolution{
		status: status,
		label:  label,
		actual: actual,
	}
	return resolution
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveCompletionGroup(state *completionGroupLocatorState, result cli.Completion) completionGroupResolution {
	var resolution completionGroupResolution
	if state == nil || state.mode == completionGroupLocatorInvalid {
		resolution = completionGroupResolution{
			status: locatorResolutionInvalid,
			label:  evaluationText("completion group"),
			actual: evaluationText("invalid completion-group locator"),
		}
		return resolution
	}
	matches := make([]int, 0, 1)
	var label evaluationField = evaluationText("completion group")
	switch state.mode {
	case completionGroupLocatorIndex:
		label = subjectAt(evaluationText("completion group"), state.index)
		if state.index < 0 {
			resolution = completionGroupResolution{
				status: locatorResolutionInvalid,
				label:  label,
				actual: evaluationText("invalid completion-group index"),
			}
			return resolution
		}
		if state.index < len(result.Groups) {
			matches = append(matches, state.index)
		}
	case completionGroupLocatorID:
		label = subjectNamed(evaluationText("completion group"), state.id)
		for index, group := range result.Groups {
			if group.ID == state.id {
				matches = append(matches, index)
			}
		}
	default:
		resolution = completionGroupResolution{
			status: locatorResolutionInvalid,
			label:  label,
			actual: evaluationText("invalid completion-group locator"),
		}
		return resolution
	}
	if len(matches) == 1 {
		resolution = completionGroupResolution{
			status: locatorResolutionFound,
			value:  result.Groups[matches[0]],
			index:  matches[0],
			label:  label,
		}
		return resolution
	}
	if len(matches) > 1 {
		resolution = completionGroupResolution{
			status: locatorResolutionAmbiguous,
			label:  label,
			actual: collectionCardinalityField{
				name:  "matches completion-group indexes",
				count: len(matches),
			},
		}
		return resolution
	}
	status, actual := completeMiss(completionObservationIncomplete(result), label, "completion group is absent")
	resolution = completionGroupResolution{
		status: status,
		label:  label,
		actual: actual,
	}
	return resolution
}

func resolveCompletionNativeRequest(state *completionNativeRequestLocatorState, result cli.Completion) completionNativeRequestResolution {
	var resolution completionNativeRequestResolution
	if state == nil || !state.valid || state.index < 0 {
		resolution = completionNativeRequestResolution{
			status: locatorResolutionInvalid,
			label:  evaluationText("completion native request"),
			actual: evaluationText("invalid completion native-request locator"),
		}
		return resolution
	}
	label := subjectAt(evaluationText("completion native request"), state.index)
	if state.index < len(result.NativeRequests) {
		resolution = completionNativeRequestResolution{
			status: locatorResolutionFound,
			value:  result.NativeRequests[state.index],
			index:  state.index,
			label:  label,
		}
		return resolution
	}
	status, actual := completeMiss(completionObservationIncomplete(result), label, "completion native request is absent")
	resolution = completionNativeRequestResolution{
		status: status,
		label:  label,
		actual: actual,
	}
	return resolution
}

func resolutionOutcome(status locatorResolutionStatus, subject, expected, actual evaluationField) evaluationResult {
	switch status {
	case locatorResolutionNotProvable:
		return notProvableResult(subject, expected, actual)
	case locatorResolutionFound:
		return provedResult()
	default:
		return disprovedResult(subject, expected, actual)
	}
}

func completionDeclarationLabel(state *completionDeclarationLocatorState) evaluationField {
	if state == nil {
		return evaluationText("completion declaration")
	}
	switch state.mode {
	case completionDeclarationLocatorFlag:
		return qualifySubject(flagLocatorLabel(state.flag), evaluationText("completion"))
	case completionDeclarationLocatorArgumentTerm:
		return qualifySubject(argumentTermLocatorLabel(state.term), evaluationText("completion"))
	default:
		return evaluationText("completion declaration")
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveSchemaCompletionDeclaration(state *completionDeclarationLocatorState, schema *cli.Schema) locatorResolution[cli.CompletionDeclarationCapabilities] {
	var resolution locatorResolution[cli.CompletionDeclarationCapabilities]
	label := completionDeclarationLabel(state)
	if state == nil {
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCompletionParent, observationInvalidLocator),
		}
		return resolution
	}
	switch state.mode {
	case completionDeclarationLocatorFlag:
		resolved := resolveSchemaFlagLocator(state.flag, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionFound,
			value:   resolved.value.Completion,
			subject: label,
		}
		return resolution
	case completionDeclarationLocatorArgumentTerm:
		resolved := resolveSchemaArgumentTermLocator(state.term, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionFound,
			value:   resolved.value.Completion,
			subject: label,
		}
		return resolution
	default:
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCompletionParent, observationMustBeFlagOrArgumentTerm),
		}
		return resolution
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveReferenceCompletionDeclaration(state *completionDeclarationLocatorState, reference cli.Reference) locatorResolution[cli.CompletionDeclarationCapabilities] {
	var resolution locatorResolution[cli.CompletionDeclarationCapabilities]
	label := completionDeclarationLabel(state)
	if state == nil {
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCompletionParent, observationInvalidLocator),
		}
		return resolution
	}
	switch state.mode {
	case completionDeclarationLocatorFlag:
		resolved := resolveReferenceFlagLocator(state.flag, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionFound,
			value:   resolved.value.Completion,
			subject: label,
		}
		return resolution
	case completionDeclarationLocatorArgumentTerm:
		resolved := resolveReferenceArgumentTermLocator(state.term, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionFound,
			value:   resolved.value.Completion,
			subject: label,
		}
		return resolution
	default:
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCompletionParent, observationMustBeFlagOrArgumentTerm),
		}
		return resolution
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveInspectionCompletionDeclaration(state *completionDeclarationLocatorState, inspection cli.Inspection) locatorResolution[cli.CompletionDeclarationCapabilities] {
	var resolution locatorResolution[cli.CompletionDeclarationCapabilities]
	label := completionDeclarationLabel(state)
	if state == nil {
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCompletionParent, observationInvalidLocator),
		}
		return resolution
	}
	switch state.mode {
	case completionDeclarationLocatorFlag:
		resolved := resolveInspectionFlagLocator(state.flag, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionFound,
			value:   resolved.value.Completion,
			subject: label,
		}
		return resolution
	case completionDeclarationLocatorArgumentTerm:
		resolved := resolveInspectionArgumentTermLocator(state.term, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionFound,
			value:   resolved.value.Completion,
			subject: label,
		}
		return resolution
	default:
		resolution = locatorResolution[cli.CompletionDeclarationCapabilities]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectCompletionParent, observationMustBeFlagOrArgumentTerm),
		}
		return resolution
	}
}

func resolveSchemaValueStates(states []*valueLocatorState, schema *cli.Schema) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		resolved := resolveSchemaValueLocator(state, schema)
		if resolved.status != locatorResolutionFound {
			return nil, resolved.status, resolved.detail
		}
		ids[index] = resolved.value.ID
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func resolveReferenceValueStates(states []*valueLocatorState, reference cli.Reference) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		resolved := resolveReferenceValueLocator(state, reference)
		if resolved.status != locatorResolutionFound {
			return nil, resolved.status, resolved.detail
		}
		ids[index] = resolved.value.ID
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func resolveInspectionValueStates(states []*valueLocatorState, inspection cli.Inspection) ([]cli.ValueID, locatorResolutionStatus, evaluationField) {
	ids := make([]cli.ValueID, len(states))
	for index, state := range states {
		id, status, detail := resolveInspectionValueID(state, inspection)
		if status != locatorResolutionFound {
			return nil, status, detail
		}
		ids[index] = id
	}
	return ids, locatorResolutionFound, evaluationText("")
}

func completionTargetPresenceExpectation(locator CompletionTargetLocator, present bool) CompletionExpectation {
	value := completionTargetPresenceExpectationValue{
		locator: locator.state,
		present: present,
	}
	return value
}

func completionTargetKindExpectation(locator CompletionTargetLocator, kind cli.CompletionTargetKind) CompletionExpectation {
	expectation := completionTargetPropertyExpectation{
		locator:  locator.state,
		property: completionTargetPropertyKind,
		kind:     kind,
	}
	return expectation
}

func completionTargetNameExpectation(locator CompletionTargetLocator, name string) CompletionExpectation {
	expectation := completionTargetPropertyExpectation{
		locator:  locator.state,
		property: completionTargetPropertyName,
		text:     name,
	}
	return expectation
}

func completionTargetSpellingExpectation(locator CompletionTargetLocator, spelling string) CompletionExpectation {
	expectation := completionTargetPropertyExpectation{
		locator:  locator.state,
		property: completionTargetPropertySpelling,
		text:     spelling,
	}
	return expectation
}

func completionItemPresenceExpectation(locator CompletionItemLocator, present bool) CompletionExpectation {
	value := completionItemPresenceExpectationValue{
		locator: locator.state,
		present: present,
	}
	return value
}

func completionItemValueExpectation(locator CompletionItemLocator, value string) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyValue,
		text:     value,
	}
	return expectation
}

func completionItemReplacementExpectation(locator CompletionItemLocator, values []string) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyReplacement,
		strings:  slices.Clone(values),
	}
	return expectation
}

func completionItemKindExpectation(locator CompletionItemLocator, kind cli.CompletionCandidateKind) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyKind,
		kind:     kind,
	}
	return expectation
}

func completionItemLifecycleExpectation(locator CompletionItemLocator, state cli.LifecycleState) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:   locator.state,
		property:  completionItemPropertyLifecycle,
		lifecycle: state,
	}
	return expectation
}

func completionItemMatchExpectation(locator CompletionItemLocator, kind cli.CompletionMatchKind) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:   locator.state,
		property:  completionItemPropertyMatchKind,
		matchKind: kind,
	}
	return expectation
}

func completionItemPriorityExpectation(locator CompletionItemLocator, priority int) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyPriority,
		integer:  priority,
	}
	return expectation
}

func completionItemSpacingExpectation(locator CompletionItemLocator, spacing cli.CompletionSpacing) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertySpacing,
		spacing:  spacing,
	}
	return expectation
}

func completionItemGroupExpectation(locator CompletionItemLocator, group CompletionGroupLocator) CompletionExpectation {
	value := completionItemGroupExpectationValue{
		item:  locator.state,
		group: group.state,
	}
	return value
}

func completionGroupMembershipIdentifiable(group cli.CompletionGroup, result cli.Completion) bool {
	if group.ID == "" {
		return false
	}
	matches := 0
	for _, candidate := range result.Groups {
		if candidate.ID == group.ID {
			matches++
		}
	}
	return matches == 1
}

func completionItemTargetsExactlyExpectation(item CompletionItemLocator, targets []CompletionTargetLocator) CompletionExpectation {
	states := make([]*completionTargetLocatorState, len(targets))
	for i, target := range targets {
		states[i] = target.state
	}
	value := completionItemTargetsExpectationValue{
		item:    item.state,
		targets: states,
	}
	return value
}

func completionGroupPresenceExpectation(locator CompletionGroupLocator, present bool) CompletionExpectation {
	value := completionGroupPresenceExpectationValue{
		locator: locator.state,
		present: present,
	}
	return value
}

func completionGroupIDExpectation(locator CompletionGroupLocator, id string) CompletionExpectation {
	expectation := completionGroupPropertyExpectation{
		locator:  locator.state,
		property: completionGroupPropertyID,
		text:     id,
	}
	return expectation
}

func completionGroupPriorityExpectation(locator CompletionGroupLocator, priority int) CompletionExpectation {
	expectation := completionGroupPropertyExpectation{
		locator:  locator.state,
		property: completionGroupPropertyPriority,
		priority: priority,
	}
	return expectation
}

func completionGroupSensitiveExpectation(locator CompletionGroupLocator) CompletionExpectation {
	expectation := completionGroupPropertyExpectation{
		locator:  locator.state,
		property: completionGroupPropertySensitive,
	}
	return expectation
}

func completionTargetsExactlyExpectation(locators []CompletionTargetLocator) CompletionExpectation {
	states := make([]*completionTargetLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionTargetsExactlyExpectationValue{
		states: states,
	}
	return value
}

func evaluateExactCompletionTargets(states []*completionTargetLocatorState, result cli.Completion) evaluationResult {
	const subject = "completion targets"
	if completionObservationIncomplete(result) {
		return notProvableResult(evaluationText(subject), evaluationText("exact authoritative collection"), evaluationText(completionObservationIncompleteReason(result)))
	}
	if len(result.Targets) != len(states) {
		return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("%d entries", len(states))), evaluationText(fmt.Sprintf("%d entries", len(result.Targets))))
	}
	for index, state := range states {
		resolved := resolveCompletionTarget(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("locator entry %d at authoritative index %d", index, index)), evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)))
		}
	}
	return provedResult()
}

func completionItemsExactlyExpectation(locators []CompletionItemLocator) CompletionExpectation {
	states := make([]*completionItemLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionItemsExactlyExpectationValue{
		states: states,
	}
	return value
}

func completionGroupsExactlyExpectation(locators []CompletionGroupLocator) CompletionExpectation {
	states := make([]*completionGroupLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionGroupsExactlyExpectationValue{
		states: states,
	}
	return value
}

func completionNativeRequestsExactlyExpectation(locators []CompletionNativeRequestLocator) CompletionExpectation {
	states := make([]*completionNativeRequestLocatorState, len(locators))
	for index, locator := range locators {
		states[index] = locator.state
	}
	value := completionNativeRequestsExactlyExpectationValue{
		states: states,
	}
	return value
}

func completionItemDeclarationExpectation(locator CompletionItemLocator, declaration DeclarationIdentityLocator) CompletionExpectation {
	expected, valid := locatorDeclarationIdentityValue(declaration)
	expectation := completionItemPropertyExpectation{
		locator:          locator.state,
		property:         completionItemPropertyDeclaration,
		declaration:      expected,
		declarationValid: valid,
	}
	return expectation
}

func completionTargetCommandPathExpectation(locator CompletionTargetLocator, path []string) CompletionExpectation {
	expectation := completionTargetPropertyExpectation{
		locator:  locator.state,
		property: completionTargetPropertyCommandPath,
		path:     slices.Clone(path),
	}
	return expectation
}

func completionTargetTermIndexExpectation(locator CompletionTargetLocator, index int) CompletionExpectation {
	expectation := completionTargetPropertyExpectation{
		locator:  locator.state,
		property: completionTargetPropertyTermIndex,
		index:    index,
	}
	return expectation
}

func completionTargetFlagValueFormExpectation(locator CompletionTargetLocator, form cli.FlagValueForm) CompletionExpectation {
	expectation := completionTargetPropertyExpectation{
		locator:   locator.state,
		property:  completionTargetPropertyFlagValueForm,
		valueForm: form,
	}
	return expectation
}

func completionTargetOccurrenceExpectation(locator CompletionTargetLocator, index int) CompletionExpectation {
	expectation := completionTargetPropertyExpectation{
		locator:  locator.state,
		property: completionTargetPropertyOccurrence,
		index:    index,
	}
	return expectation
}

func completionItemDisplayTextExpectation(locator CompletionItemLocator, text string) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyDisplayText,
		text:     text,
	}
	return expectation
}

func completionItemDescriptionExpectation(locator CompletionItemLocator, description string) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyDescription,
		text:     description,
	}
	return expectation
}

func completionItemDetailExpectation(locator CompletionItemLocator, detail string) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyDetail,
		text:     detail,
	}
	return expectation
}

func completionItemSearchTermsExpectation(locator CompletionItemLocator, terms []string) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertySearchTerms,
		strings:  slices.Clone(terms),
	}
	return expectation
}

func completionItemSortTextExpectation(locator CompletionItemLocator, text string) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertySortText,
		text:     text,
	}
	return expectation
}

func completionItemCommitCharactersExpectation(locator CompletionItemLocator, characters []rune) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyCommitCharacters,
		runes:    slices.Clone(characters),
	}
	return expectation
}

func completionItemMatchFieldExpectation(locator CompletionItemLocator, field cli.CompletionMatchField) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:    locator.state,
		property:   completionItemPropertyMatchField,
		matchField: field,
	}
	return expectation
}

func completionItemMatchFieldIndexExpectation(locator CompletionItemLocator, index int) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyMatchFieldIndex,
		integer:  index,
	}
	return expectation
}

func completionItemMatchScoreExpectation(locator CompletionItemLocator, score int64) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyMatchScore,
		score:    score,
	}
	return expectation
}

func completionItemMatchRangesExpectation(locator CompletionItemLocator, ranges []cli.CompletionRange) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyMatchRanges,
		ranges:   slices.Clone(ranges),
	}
	return expectation
}

func completionItemMatchEditsExpectation(locator CompletionItemLocator, edits int) CompletionExpectation {
	expectation := completionItemPropertyExpectation{
		locator:  locator.state,
		property: completionItemPropertyMatchEdits,
		integer:  edits,
	}
	return expectation
}

func completionGroupLabelExpectation(locator CompletionGroupLocator, label string) CompletionExpectation {
	expectation := completionGroupPropertyExpectation{
		locator:  locator.state,
		property: completionGroupPropertyLabel,
		text:     label,
	}
	return expectation
}

func completionGroupItemsExactlyExpectation(group CompletionGroupLocator, items []CompletionItemLocator) CompletionExpectation {
	states := make([]*completionItemLocatorState, len(items))
	for index, item := range items {
		states[index] = item.state
	}
	expectation := completionGroupItemsExpectation{
		group: group.state,
		items: states,
	}
	return expectation
}

func completionMessagesExactlyExpectation(messages []cli.CompletionMessage) CompletionExpectation {
	expectation := completionMessagesExpectation{
		messages: slices.Clone(messages),
	}
	return expectation
}

func declarationIdentityKindPresent(identity cli.DeclarationIdentity, expected cli.DeclarationKind) bool {
	if identity.Kind != expected {
		return false
	}
	switch expected {
	case cli.DeclarationKindCommand:
		return identity.Command != 0
	case cli.DeclarationKindCommandEntry:
		return identity.CommandEntry != 0
	case cli.DeclarationKindActionEntry:
		return identity.ActionEntry != 0
	case cli.DeclarationKindFlag:
		return identity.Flag != 0
	case cli.DeclarationKindFlagSpelling:
		return identity.FlagSpelling != 0
	case cli.DeclarationKindArgumentForm:
		return identity.ArgumentForm != 0
	case cli.DeclarationKindArgumentTerm:
		return identity.ArgumentTerm != 0
	case cli.DeclarationKindEnvironmentInput:
		return identity.EnvironmentInput != 0
	case cli.DeclarationKindValueDomainEntry:
		return identity.ValueDomainEntry != 0
	default:
		return false
	}
}

func diagnosticReportFromCompletion(result cli.Completion) cli.DiagnosticReport {
	report := cli.DiagnosticReport{
		Diagnostics: result.Diagnostics,
		Sources:     result.Sources,
		Origins:     result.Origins,
		Repairs:     result.Repairs,
		Truncated:   result.DiagnosticsTruncated,
	}

	return report
}

func (completionDeclarationExpectationValue) schemaExpectation() {}

func (completionDeclarationExpectationValue) referenceExpectation() {}

func (completionDeclarationExpectationValue) inspectionExpectation() {}

func (expectation completionDeclarationExpectationValue) evaluate(value cli.CompletionDeclarationCapabilities, status locatorResolutionStatus, detail evaluationField, dependencies []cli.ValueID, dependencyStatus locatorResolutionStatus, dependencyDetail evaluationField, truncated bool) evaluationResult {
	expect := expectation.expectedText()
	if status != locatorResolutionFound {
		return declarationMissing(status, evaluationText("completion declaration"), evaluationText(expect), detail, truncated)
	}
	switch expectation.property {
	case completionDeclarationExpectationDynamic:
		if value.Dynamic {
			return provedResult()
		}
		return disprovedResult(evaluationText("completion declaration"), evaluationText(expect), evaluationText(fmt.Sprintf("dynamic=%t", value.Dynamic)))
	case completionDeclarationExpectationDelegated:
		if value.Delegated {
			return provedResult()
		}
		return disprovedResult(evaluationText("completion declaration"), evaluationText(expect), evaluationText(fmt.Sprintf("delegated=%t", value.Delegated)))
	case completionDeclarationExpectationNative:
		if value.Native == nil {
			return disprovedResult(evaluationText("completion declaration"), evaluationText(expect), evaluationText("native completion is absent"))
		}
		if value.Native.Kind == expectation.nativeKind && value.Native.Mode == expectation.nativeMode {
			return provedResult()
		}
		return disprovedResult(evaluationText("completion declaration"), evaluationText(expect), evaluationText(fmt.Sprintf("native kind=%v mode=%v", value.Native.Kind, value.Native.Mode)))
	case completionDeclarationExpectationDependencies:
		if dependencyStatus != locatorResolutionFound {
			return declarationMissing(dependencyStatus, evaluationText("completion declaration"), evaluationText(expect), dependencyDetail, truncated)
		}
		if slices.Equal(value.Dependencies, dependencies) {
			return provedResult()
		}
		return disprovedResult(evaluationText("completion declaration"), evaluationText(expect), evaluationText("different semantic dependency sequence"))
	default:
		panic("clitest: invalid completion-declaration expectation")
	}
}

func (expectation completionDeclarationExpectationValue) expectedText() string {
	switch expectation.property {
	case completionDeclarationExpectationDynamic:
		return "dynamic=true"
	case completionDeclarationExpectationDelegated:
		return "delegated=true"
	case completionDeclarationExpectationNative:
		return fmt.Sprintf("native kind=%v mode=%v", expectation.nativeKind, expectation.nativeMode)
	case completionDeclarationExpectationDependencies:
		return "exact semantic completion dependencies in order"
	default:
		return "valid completion declaration expectation"
	}
}

func (expectation completionDeclarationExpectationValue) evaluateSchema(schema *cli.Schema) evaluationResult {
	resolved := resolveSchemaCompletionDeclaration(expectation.locator, schema)
	dependencies, status, detail := resolveSchemaValueStates(expectation.dependencies, schema)
	return expectation.evaluate(resolved.value, resolved.status, resolved.detail, dependencies, status, detail, false)
}

func (expectation completionDeclarationExpectationValue) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	resolved := resolveReferenceCompletionDeclaration(expectation.locator, reference)
	dependencies, status, detail := resolveReferenceValueStates(expectation.dependencies, reference)
	return expectation.evaluate(resolved.value, resolved.status, resolved.detail, dependencies, status, detail, false)
}

func (expectation completionDeclarationExpectationValue) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	resolved := resolveInspectionCompletionDeclaration(expectation.locator, inspection)
	dependencies, status, detail := resolveInspectionValueStates(expectation.dependencies, inspection)
	return expectation.evaluate(resolved.value, resolved.status, resolved.detail, dependencies, status, detail, inspection.Truncated)
}

func (completionTargetPropertyExpectation) completionExpectation() {}

func (expectation completionTargetPropertyExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionTarget(expectation.locator, result)
	expected := expectation.expected()
	if resolved.status != locatorResolutionFound {
		return resolutionOutcome(resolved.status, resolved.label, expected, resolved.actual)
	}
	if expectation.matches(resolved.value) {
		return provedResult()
	}
	return disprovedResult(resolved.label, expected, expectation.observed(resolved.value))
}

func (expectation completionTargetPropertyExpectation) expected() evaluationField {
	switch expectation.property {
	case completionTargetPropertyKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)))
	case completionTargetPropertyName:
		return property(evaluationText("name"), evaluationText(expectation.text))
	case completionTargetPropertySpelling:
		return property(evaluationText("spelling"), evaluationText(expectation.text))
	case completionTargetPropertyCommandPath:
		field := stringCollectionField{
			values: expectation.path,
		}
		return property(evaluationText("command path"), field)
	case completionTargetPropertyTermIndex:
		return evaluationText(fmt.Sprintf("term index %d", expectation.index))
	case completionTargetPropertyFlagValueForm:
		return property(evaluationText("flag value form"), evaluationText(fmt.Sprint(expectation.valueForm)))
	case completionTargetPropertyOccurrence:
		return evaluationText(fmt.Sprintf("occurrence %d", expectation.index))
	default:
		panic("clitest: invalid completion target property")
	}
}

func (expectation completionTargetPropertyExpectation) observed(target cli.CompletionTarget) evaluationField {
	switch expectation.property {
	case completionTargetPropertyKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(target.Kind)))
	case completionTargetPropertyName:
		return property(evaluationText("name"), evaluationText(target.Name))
	case completionTargetPropertySpelling:
		return property(evaluationText("spelling"), evaluationText(target.Spelling))
	case completionTargetPropertyCommandPath:
		field := stringCollectionField{
			values: target.CommandPath,
		}
		return property(evaluationText("command path"), field)
	case completionTargetPropertyTermIndex:
		return evaluationText(fmt.Sprintf("term index %d", target.TermIndex))
	case completionTargetPropertyFlagValueForm:
		return property(evaluationText("flag value form"), evaluationText(fmt.Sprint(target.FlagValueForm)))
	case completionTargetPropertyOccurrence:
		return evaluationText(fmt.Sprintf("occurrence %d", target.Occurrence))
	default:
		panic("clitest: invalid completion target property")
	}
}

func (expectation completionTargetPropertyExpectation) matches(target cli.CompletionTarget) bool {
	switch expectation.property {
	case completionTargetPropertyKind:
		return target.Kind == expectation.kind
	case completionTargetPropertyName:
		return target.Name == expectation.text
	case completionTargetPropertySpelling:
		return target.Spelling == expectation.text
	case completionTargetPropertyCommandPath:
		return slices.Equal(target.CommandPath, expectation.path)
	case completionTargetPropertyTermIndex:
		return target.TermIndex == expectation.index
	case completionTargetPropertyFlagValueForm:
		return target.FlagValueForm == expectation.valueForm
	case completionTargetPropertyOccurrence:
		return target.Occurrence == expectation.index
	default:
		panic("clitest: invalid completion target property")
	}
}

func (completionTargetPresenceExpectationValue) completionExpectation() {}

func (expectation completionTargetPresenceExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionTarget(expectation.locator, result)
	if expectation.present {
		if resolved.status == locatorResolutionFound {
			return provedResult()
		}
		return resolutionOutcome(resolved.status, resolved.label, evaluationText("present"), resolved.actual)
	}
	if resolved.status == locatorResolutionAbsent {
		return provedResult()
	}
	if resolved.status == locatorResolutionNotProvable {
		return notProvableResult(resolved.label, semanticRequirementAbsent, resolved.actual)
	}
	if resolved.status == locatorResolutionFound {
		return disprovedResult(resolved.label, semanticRequirementAbsent, evaluationText("completion target is present"))
	}
	return disprovedResult(resolved.label, semanticRequirementAbsent, resolved.actual)
}

func (completionItemPresenceExpectationValue) completionExpectation() {}

func (expectation completionItemPresenceExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionItem(expectation.locator, result)
	if expectation.present {
		if resolved.status == locatorResolutionFound {
			return provedResult()
		}
		return resolutionOutcome(resolved.status, resolved.label, evaluationText("present"), resolved.actual)
	}
	if resolved.status == locatorResolutionAbsent {
		return provedResult()
	}
	if resolved.status == locatorResolutionNotProvable {
		return notProvableResult(resolved.label, semanticRequirementAbsent, resolved.actual)
	}
	if resolved.status == locatorResolutionFound {
		return disprovedResult(resolved.label, semanticRequirementAbsent, evaluationText("completion item is present"))
	}
	return disprovedResult(resolved.label, semanticRequirementAbsent, resolved.actual)
}

func (completionItemPropertyExpectation) completionExpectation() {}

func (expectation completionItemPropertyExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionItem(expectation.locator, result)
	expected := expectation.expected()
	if resolved.status != locatorResolutionFound {
		return resolutionOutcome(resolved.status, resolved.label, expected, resolved.actual)
	}
	if expectation.matches(resolved.value) {
		return provedResult()
	}
	if resolved.value.Sensitive {
		return disprovedResult(resolved.label, evaluationText("redacted expected item property"), evaluationText("redacted observed item property"))
	}
	return disprovedResult(resolved.label, expected, expectation.observed(resolved.value))
}

func (expectation completionItemPropertyExpectation) expected() evaluationField {
	switch expectation.property {
	case completionItemPropertyValue:
		return property(evaluationText("value"), evaluationText(expectation.text))
	case completionItemPropertyReplacement:
		field := stringCollectionField{
			values: expectation.strings,
		}
		return property(evaluationText("replacement"), field)
	case completionItemPropertyKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(expectation.kind)))
	case completionItemPropertyRecommended:
		return evaluationText("recommended")
	case completionItemPropertyFallbackOnly:
		return evaluationText("fallback-only")
	case completionItemPropertyLifecycle:
		return property(evaluationText("lifecycle"), evaluationText(fmt.Sprint(expectation.lifecycle)))
	case completionItemPropertySensitive:
		return evaluationText("sensitive")
	case completionItemPropertyRequired:
		return evaluationText("required")
	case completionItemPropertyConditional:
		return evaluationText("conditional")
	case completionItemPropertyMatchKind:
		return property(evaluationText("match kind"), evaluationText(fmt.Sprint(expectation.matchKind)))
	case completionItemPropertyPriority:
		return evaluationText(fmt.Sprintf("priority %d", expectation.integer))
	case completionItemPropertySpacing:
		return property(evaluationText("spacing"), evaluationText(fmt.Sprint(expectation.spacing)))
	case completionItemPropertyDeclaration:
		field := declarationIdentityField{
			value: expectation.declaration,
		}
		return field
	case completionItemPropertyDisplayText:
		return property(evaluationText("display text"), evaluationText(expectation.text))
	case completionItemPropertyDescription:
		return property(evaluationText("description"), evaluationText(expectation.text))
	case completionItemPropertyDetail:
		return property(evaluationText("detail"), evaluationText(expectation.text))
	case completionItemPropertySearchTerms:
		field := stringCollectionField{
			values: expectation.strings,
		}
		return property(evaluationText("search terms"), field)
	case completionItemPropertySortText:
		return property(evaluationText("sort text"), evaluationText(expectation.text))
	case completionItemPropertyCommitCharacters:
		field := runeCollectionField{
			values: expectation.runes,
		}
		return property(evaluationText("commit characters"), field)
	case completionItemPropertyMatchField:
		return property(evaluationText("match field"), evaluationText(fmt.Sprint(expectation.matchField)))
	case completionItemPropertyMatchFieldIndex:
		return evaluationText(fmt.Sprintf("match field index %d", expectation.integer))
	case completionItemPropertyMatchScore:
		return evaluationText(fmt.Sprintf("match score %d", expectation.score))
	case completionItemPropertyMatchRanges:
		field := completionRangeCollectionField{
			values: expectation.ranges,
		}
		return property(evaluationText("match ranges"), field)
	case completionItemPropertyMatchEdits:
		return evaluationText(fmt.Sprintf("match edits %d", expectation.integer))
	case completionItemPropertyDeclarationKind:
		return evaluationText(fmt.Sprintf("declaration kind %v with identity present", expectation.declarationKind))
	case completionItemPropertyUngrouped:
		return evaluationText("no completion group relation")
	default:
		panic("clitest: invalid completion item property")
	}
}

func (expectation completionItemPropertyExpectation) matches(item cli.CompletionItem) bool {
	switch expectation.property {
	case completionItemPropertyValue:
		return item.Value == expectation.text
	case completionItemPropertyReplacement:
		return slices.Equal(item.Replacement, expectation.strings)
	case completionItemPropertyKind:
		return item.Kind == expectation.kind
	case completionItemPropertyRecommended:
		return item.Recommended
	case completionItemPropertyFallbackOnly:
		return item.FallbackOnly
	case completionItemPropertyLifecycle:
		return item.LifecycleState == expectation.lifecycle
	case completionItemPropertySensitive:
		return item.Sensitive
	case completionItemPropertyRequired:
		return item.Required
	case completionItemPropertyConditional:
		return item.Conditional
	case completionItemPropertyMatchKind:
		return item.Match.Kind == expectation.matchKind
	case completionItemPropertyPriority:
		return item.Priority == expectation.integer
	case completionItemPropertySpacing:
		return item.Spacing == expectation.spacing
	case completionItemPropertyDeclaration:
		return expectation.declarationValid && item.Declaration == expectation.declaration
	case completionItemPropertyDisplayText:
		return item.DisplayText == expectation.text
	case completionItemPropertyDescription:
		return item.Description == expectation.text
	case completionItemPropertyDetail:
		return item.Detail == expectation.text
	case completionItemPropertySearchTerms:
		return slices.Equal(item.SearchTerms, expectation.strings)
	case completionItemPropertySortText:
		return item.SortText == expectation.text
	case completionItemPropertyCommitCharacters:
		return slices.Equal(item.CommitCharacters, expectation.runes)
	case completionItemPropertyMatchField:
		return item.Match.Field == expectation.matchField
	case completionItemPropertyMatchFieldIndex:
		return item.Match.FieldIndex == expectation.integer
	case completionItemPropertyMatchScore:
		return item.Match.Score == expectation.score
	case completionItemPropertyMatchRanges:
		return slices.Equal(item.Match.Ranges, expectation.ranges)
	case completionItemPropertyMatchEdits:
		return item.Match.Edits == expectation.integer
	case completionItemPropertyDeclarationKind:
		return declarationIdentityKindPresent(item.Declaration, expectation.declarationKind)
	case completionItemPropertyUngrouped:
		return item.GroupID == ""
	default:
		panic("clitest: invalid completion item property")
	}
}

func (expectation completionItemPropertyExpectation) observed(item cli.CompletionItem) evaluationField {
	switch expectation.property {
	case completionItemPropertyValue:
		return property(evaluationText("value"), evaluationText(item.Value))
	case completionItemPropertyReplacement:
		field := stringCollectionField{
			values: item.Replacement,
		}
		return property(evaluationText("replacement"), field)
	case completionItemPropertyKind:
		return property(evaluationText("kind"), evaluationText(fmt.Sprint(item.Kind)))
	case completionItemPropertyRecommended:
		return property(evaluationText("recommended"), evaluationBool(item.Recommended))
	case completionItemPropertyFallbackOnly:
		return property(evaluationText("fallback-only"), evaluationBool(item.FallbackOnly))
	case completionItemPropertyLifecycle:
		return property(evaluationText("lifecycle"), evaluationText(fmt.Sprint(item.LifecycleState)))
	case completionItemPropertySensitive:
		return property(evaluationText("sensitive"), evaluationBool(item.Sensitive))
	case completionItemPropertyRequired:
		return property(evaluationText("required"), evaluationBool(item.Required))
	case completionItemPropertyConditional:
		return property(evaluationText("conditional"), evaluationBool(item.Conditional))
	case completionItemPropertyMatchKind:
		return property(evaluationText("match kind"), evaluationText(fmt.Sprint(item.Match.Kind)))
	case completionItemPropertyPriority:
		return evaluationText(fmt.Sprintf("priority %d", item.Priority))
	case completionItemPropertySpacing:
		return property(evaluationText("spacing"), evaluationText(fmt.Sprint(item.Spacing)))
	case completionItemPropertyDeclaration:
		field := declarationIdentityField{
			value: item.Declaration,
		}
		return field
	case completionItemPropertyDisplayText:
		return property(evaluationText("display text"), evaluationText(item.DisplayText))
	case completionItemPropertyDescription:
		return property(evaluationText("description"), evaluationText(item.Description))
	case completionItemPropertyDetail:
		return property(evaluationText("detail"), evaluationText(item.Detail))
	case completionItemPropertySearchTerms:
		field := stringCollectionField{
			values: item.SearchTerms,
		}
		return property(evaluationText("search terms"), field)
	case completionItemPropertySortText:
		return property(evaluationText("sort text"), evaluationText(item.SortText))
	case completionItemPropertyCommitCharacters:
		field := runeCollectionField{
			values: item.CommitCharacters,
		}
		return property(evaluationText("commit characters"), field)
	case completionItemPropertyMatchField:
		return property(evaluationText("match field"), evaluationText(fmt.Sprint(item.Match.Field)))
	case completionItemPropertyMatchFieldIndex:
		return evaluationText(fmt.Sprintf("match field index %d", item.Match.FieldIndex))
	case completionItemPropertyMatchScore:
		return evaluationText(fmt.Sprintf("match score %d", item.Match.Score))
	case completionItemPropertyMatchRanges:
		field := completionRangeCollectionField{
			values: item.Match.Ranges,
		}
		return property(evaluationText("match ranges"), field)
	case completionItemPropertyMatchEdits:
		return evaluationText(fmt.Sprintf("match edits %d", item.Match.Edits))
	case completionItemPropertyDeclarationKind:
		if item.Declaration.Kind == expectation.declarationKind {
			return evaluationText(fmt.Sprintf("declaration kind %v with identity absent", item.Declaration.Kind))
		}
		return evaluationText(fmt.Sprintf("declaration kind %v", item.Declaration.Kind))
	case completionItemPropertyUngrouped:
		return evaluationText("completion group relation is present")
	default:
		panic("clitest: invalid completion item property")
	}
}

func (completionItemGroupExpectationValue) completionExpectation() {}

func (expectation completionItemGroupExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	item := resolveCompletionItem(expectation.item, result)
	if item.status != locatorResolutionFound {
		return resolutionOutcome(item.status, item.label, evaluationText("configured completion group"), item.actual)
	}
	group := resolveCompletionGroup(expectation.group, result)
	if group.status != locatorResolutionFound {
		return resolutionOutcome(group.status, qualifySubject(item.label, evaluationText("group")), evaluationText("configured completion group"), group.actual)
	}
	if !completionGroupMembershipIdentifiable(group.value, result) {
		return disprovedResult(
			qualifySubject(item.label, evaluationText("group")),
			evaluationText("uniquely identifiable completion group"),
			evaluationText("group membership is ambiguous because the group ID is empty or non-unique"),
		)
	}
	if item.value.GroupID == group.value.ID {
		return provedResult()
	}
	return disprovedResult(
		qualifySubject(item.label, evaluationText("group")),
		evaluationText("configured completion group"),
		evaluationText("item belongs to a different completion group"),
	)
}

func (completionItemTargetsExpectationValue) completionExpectation() {}

func (expectation completionItemTargetsExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	item := resolveCompletionItem(expectation.item, result)
	if item.status != locatorResolutionFound {
		return resolutionOutcome(item.status, item.label, evaluationText("exact targets"), item.actual)
	}
	if completionObservationIncomplete(result) {
		return notProvableResult(
			qualifySubject(item.label, evaluationText("targets")),
			evaluationText("exact target collection"),
			evaluationText(completionObservationIncompleteReason(result)),
		)
	}
	if len(item.value.TargetIndexes) != len(expectation.targets) {
		return disprovedResult(
			qualifySubject(item.label, evaluationText("targets")),
			evaluationText(fmt.Sprintf("%d targets", len(expectation.targets))),
			evaluationText(fmt.Sprintf("%d targets", len(item.value.TargetIndexes))),
		)
	}
	for i, expected := range expectation.targets {
		actualIndex := item.value.TargetIndexes[i]
		if actualIndex < 0 || actualIndex >= len(result.Targets) {
			return disprovedResult(
				qualifySubject(item.label, evaluationText("targets")),
				evaluationText("valid authoritative target relationship"),
				evaluationText("item references unavailable target"),
			)
		}
		resolved := resolveCompletionTarget(expected, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(
				resolved.status,
				qualifySubject(item.label, evaluationText("targets")),
				evaluationText(fmt.Sprintf("target at position %d", i)),
				resolved.actual,
			)
		}
		if resolved.index != actualIndex {
			return disprovedResult(
				qualifySubject(item.label, evaluationText("targets")),
				evaluationText(fmt.Sprintf("target index %d at position %d", resolved.index, i)),
				evaluationText(fmt.Sprintf("target index %d", actualIndex)),
			)
		}
	}
	return provedResult()
}

func (completionGroupPresenceExpectationValue) completionExpectation() {}

func (expectation completionGroupPresenceExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionGroup(expectation.locator, result)
	if expectation.present {
		if resolved.status == locatorResolutionFound {
			return provedResult()
		}
		return resolutionOutcome(resolved.status, resolved.label, evaluationText("present"), resolved.actual)
	}
	if resolved.status == locatorResolutionAbsent {
		return provedResult()
	}
	if resolved.status == locatorResolutionNotProvable {
		return notProvableResult(resolved.label, semanticRequirementAbsent, resolved.actual)
	}
	if resolved.status == locatorResolutionFound {
		return disprovedResult(resolved.label, semanticRequirementAbsent, evaluationText("completion group is present"))
	}
	return disprovedResult(resolved.label, semanticRequirementAbsent, resolved.actual)
}

func (completionGroupPropertyExpectation) completionExpectation() {}

func (expectation completionGroupPropertyExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionGroup(expectation.locator, result)
	expected := expectation.expected()
	if resolved.status != locatorResolutionFound {
		return resolutionOutcome(resolved.status, resolved.label, expected, resolved.actual)
	}
	if expectation.matches(resolved.value) {
		return provedResult()
	}
	if resolved.value.Sensitive {
		return disprovedResult(resolved.label, evaluationText("redacted expected group property"), evaluationText("redacted observed group property"))
	}
	return disprovedResult(resolved.label, expected, expectation.observed(resolved.value))
}

func (expectation completionGroupPropertyExpectation) expected() evaluationField {
	switch expectation.property {
	case completionGroupPropertyID:
		return subjectNamed(evaluationText("ID"), expectation.text)
	case completionGroupPropertyPriority:
		return evaluationText(fmt.Sprintf("priority %d", expectation.priority))
	case completionGroupPropertySensitive:
		return semanticRequirementSensitive
	case completionGroupPropertyLabel:
		return subjectNamed(evaluationText("label"), expectation.text)
	default:
		panic("clitest: invalid completion group property")
	}
}

func (expectation completionGroupPropertyExpectation) matches(group cli.CompletionGroup) bool {
	switch expectation.property {
	case completionGroupPropertyID:
		return group.ID == expectation.text
	case completionGroupPropertyPriority:
		return group.Priority == expectation.priority
	case completionGroupPropertySensitive:
		return group.Sensitive
	case completionGroupPropertyLabel:
		return group.Label == expectation.text
	default:
		panic("clitest: invalid completion group property")
	}
}

func (expectation completionGroupPropertyExpectation) observed(group cli.CompletionGroup) evaluationField {
	var observed string
	switch expectation.property {
	case completionGroupPropertyID:
		observed = renderEvaluationField(subjectNamed(evaluationText("ID"), group.ID))
	case completionGroupPropertyPriority:
		observed = fmt.Sprintf("priority %d", group.Priority)
	case completionGroupPropertySensitive:
		observed = fmt.Sprintf("sensitive=%t", group.Sensitive)
	case completionGroupPropertyLabel:
		observed = renderEvaluationField(subjectNamed(evaluationText("label"), group.Label))
	default:
		panic("clitest: invalid completion group property")
	}
	return evaluationText(observed)
}

func (completionNativeRequestPropertyExpectation) completionExpectation() {}

func (expectation completionNativeRequestPropertyExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionNativeRequest(expectation.locator, result)
	if resolved.status != locatorResolutionFound {
		return resolutionOutcome(resolved.status, resolved.label, evaluationText("native request property"), resolved.actual)
	}
	subject := resolved.label
	switch expectation.property {
	case completionNativeRequestPropertyKind:
		expected := evaluationText(fmt.Sprintf("kind %v", expectation.kind))
		if resolved.value.Kind == expectation.kind {
			return provedResult()
		}
		return disprovedResult(subject, expected, evaluationText(fmt.Sprintf("kind %v", resolved.value.Kind)))
	case completionNativeRequestPropertyMode:
		expected := evaluationText(fmt.Sprintf("mode %v", expectation.mode))
		if resolved.value.Mode == expectation.mode {
			return provedResult()
		}
		return disprovedResult(subject, expected, evaluationText(fmt.Sprintf("mode %v", resolved.value.Mode)))
	case completionNativeRequestPropertyRequired:
		if resolved.value.Required {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("required"), evaluationText("required=false"))
	case completionNativeRequestPropertyConditional:
		if resolved.value.Conditional {
			return provedResult()
		}
		return disprovedResult(subject, evaluationText("conditional"), evaluationText("conditional=false"))
	default:
		panic("clitest: invalid completion native-request expectation")
	}
}

func (completionDelegationExpectation) completionExpectation() {}

func (expectation completionDelegationExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if expectation.locator == nil || !expectation.locator.valid {
		return disprovedResult(evaluationText("completion delegation"), evaluationText("valid delegation coordinate"), observed(semanticSubjectDelegation, observationInvalidLocator))
	}
	if result.Delegation == nil && completionObservationIncomplete(result) {
		return notProvableResult(evaluationText("completion delegation"), expectation.expectedField(), evaluationText(completionObservationIncompleteReason(result)))
	}
	if expectation.matches(result.Delegation) {
		return provedResult()
	}
	if result.Delegation != nil && result.Delegation.Sensitive {
		return disprovedResult(evaluationText("completion delegation"), evaluationText("redacted expected delegation property"), evaluationText("redacted observed delegation property"))
	}
	return disprovedResult(evaluationText("completion delegation"), expectation.expectedField(), expectation.actualField(result.Delegation))
}

func (expectation completionDelegationExpectation) expectedField() evaluationField {
	switch expectation.property {
	case completionDelegationPropertyPresent:
		if expectation.present {
			return evaluationText("present")
		}
		return semanticRequirementAbsent
	case completionDelegationPropertyArgs:
		field := stringCollectionField{
			values: expectation.args,
		}
		return property(evaluationText("args"), field)
	case completionDelegationPropertyCursor:
		field := completionCursorField{
			value: expectation.cursor,
		}
		return field
	case completionDelegationPropertySensitive:
		return semanticRequirementSensitive
	default:
		return evaluationText("valid delegation expectation")
	}
}

func (expectation completionDelegationExpectation) matches(delegation *cli.CompletionDelegation) bool {
	switch expectation.property {
	case completionDelegationPropertyPresent:
		return (delegation != nil) == expectation.present
	case completionDelegationPropertyArgs:
		return delegation != nil && slices.Equal(delegation.Args, expectation.args)
	case completionDelegationPropertyCursor:
		return delegation != nil && delegation.Cursor == expectation.cursor
	case completionDelegationPropertySensitive:
		return delegation != nil && delegation.Sensitive
	default:
		panic("clitest: invalid completion delegation expectation")
	}
}

func (expectation completionDelegationExpectation) actualField(delegation *cli.CompletionDelegation) evaluationField {
	if delegation == nil {
		return observed(semanticSubjectDelegation, observationAbsent)
	}
	switch expectation.property {
	case completionDelegationPropertyPresent:
		return evaluationText("present")
	case completionDelegationPropertyArgs:
		field := stringCollectionField{
			values: delegation.Args,
		}
		return property(evaluationText("args"), field)
	case completionDelegationPropertyCursor:
		field := completionCursorField{
			value: delegation.Cursor,
		}
		return field
	case completionDelegationPropertySensitive:
		return evaluationText(fmt.Sprintf("sensitive=%t", delegation.Sensitive))
	default:
		return evaluationText("delegation property did not match")
	}
}

func (completionTargetsExactlyExpectationValue) completionExpectation() {}

func (expectation completionTargetsExactlyExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	return evaluateExactCompletionTargets(expectation.states, result)
}

func (completionItemsExactlyExpectationValue) completionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation completionItemsExactlyExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	const subject = "completion items"
	if completionObservationIncomplete(result) {
		return notProvableResult(evaluationText(subject), evaluationText("exact authoritative collection"), evaluationText(completionObservationIncompleteReason(result)))
	}
	if len(result.Items) != len(expectation.states) {
		return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("%d entries", len(expectation.states))), evaluationText(fmt.Sprintf("%d entries", len(result.Items))))
	}
	for index, state := range expectation.states {
		resolved := resolveCompletionItem(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("locator entry %d at authoritative index %d", index, index)), evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)))
		}
	}
	return provedResult()
}

func (completionGroupsExactlyExpectationValue) completionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation completionGroupsExactlyExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	const subject = "completion groups"
	if completionObservationIncomplete(result) {
		return notProvableResult(evaluationText(subject), evaluationText("exact authoritative collection"), evaluationText(completionObservationIncompleteReason(result)))
	}
	if len(result.Groups) != len(expectation.states) {
		return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("%d entries", len(expectation.states))), evaluationText(fmt.Sprintf("%d entries", len(result.Groups))))
	}
	for index, state := range expectation.states {
		resolved := resolveCompletionGroup(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("locator entry %d at authoritative index %d", index, index)), evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)))
		}
	}
	return provedResult()
}

func (completionNativeRequestsExactlyExpectationValue) completionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation completionNativeRequestsExactlyExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	const subject = "completion native requests"
	if completionObservationIncomplete(result) {
		return notProvableResult(evaluationText(subject), evaluationText("exact authoritative collection"), evaluationText(completionObservationIncompleteReason(result)))
	}
	if len(result.NativeRequests) != len(expectation.states) {
		return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("%d entries", len(expectation.states))), evaluationText(fmt.Sprintf("%d entries", len(result.NativeRequests))))
	}
	for index, state := range expectation.states {
		resolved := resolveCompletionNativeRequest(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(evaluationText(subject), evaluationText(fmt.Sprintf("locator entry %d at authoritative index %d", index, index)), evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)))
		}
	}
	return provedResult()
}

func (completionGroupItemsExpectation) completionExpectation() {}

func (expectation completionGroupItemsExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	group := resolveCompletionGroup(expectation.group, result)
	if group.status != locatorResolutionFound {
		return resolutionOutcome(group.status, group.label, evaluationText("exact group membership"), group.actual)
	}
	if completionObservationIncomplete(result) {
		return notProvableResult(
			qualifySubject(group.label, evaluationText("membership")),
			evaluationText("exact authoritative membership"),
			evaluationText(completionObservationIncompleteReason(result)),
		)
	}
	if !completionGroupMembershipIdentifiable(group.value, result) {
		return disprovedResult(
			qualifySubject(group.label, evaluationText("membership")),
			evaluationText("uniquely identifiable completion group"),
			evaluationText("group membership is ambiguous because the group ID is empty or non-unique"),
		)
	}

	actualIndexes := make([]int, 0, len(expectation.items))
	for index, item := range result.Items {
		if item.GroupID == group.value.ID {
			actualIndexes = append(actualIndexes, index)
		}
	}
	if len(actualIndexes) != len(expectation.items) {
		return disprovedResult(
			qualifySubject(group.label, evaluationText("membership")),
			evaluationText(fmt.Sprintf("%d items", len(expectation.items))),
			evaluationText(fmt.Sprintf("%d items", len(actualIndexes))),
		)
	}
	for position, state := range expectation.items {
		resolved := resolveCompletionItem(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(
				resolved.status,
				qualifySubject(group.label, evaluationText("membership")),
				evaluationText(fmt.Sprintf("item at position %d", position)),
				resolved.actual,
			)
		}
		if resolved.index != actualIndexes[position] {
			return disprovedResult(
				qualifySubject(group.label, evaluationText("membership")),
				evaluationText(fmt.Sprintf("item index %d at position %d", resolved.index, position)),
				evaluationText(fmt.Sprintf("item index %d", actualIndexes[position])),
			)
		}
	}
	return provedResult()
}

func (completionMessagesExpectation) completionExpectation() {}

func (expectation completionMessagesExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if completionObservationIncomplete(result) {
		return notProvableResult(
			evaluationText("completion messages"),
			evaluationText("exact authoritative messages"),
			evaluationText(completionObservationIncompleteReason(result)),
		)
	}
	if len(result.Messages) != len(expectation.messages) {
		return disprovedResult(
			evaluationText("completion messages"),
			evaluationText(fmt.Sprintf("%d messages", len(expectation.messages))),
			evaluationText(fmt.Sprintf("%d messages", len(result.Messages))),
		)
	}
	for index, expected := range expectation.messages {
		actual := result.Messages[index]
		if actual == expected {
			continue
		}
		field := completionMessageField{
			index: index,
			value: expected,
		}
		completionMessageField := completionMessageField{
			index: index,
			value: actual,
		}
		return disprovedResult(
			evaluationText("completion messages"),
			field,
			completionMessageField,
		)
	}
	return provedResult()
}

func (completionObservedTargetsExpectationValue) completionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation completionObservedTargetsExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	const subject = "observed completion targets"
	if len(result.Targets) != len(expectation.states) {
		return disprovedResult(
			evaluationText(fmt.Sprintf("%s snapshot (partial=%t truncated=%t)", subject, result.Partial, result.Truncated)),
			evaluationText(fmt.Sprintf("observed count=%d", len(expectation.states))),
			evaluationText(fmt.Sprintf("observed count=%d", len(result.Targets))),
		)
	}
	for index, state := range expectation.states {
		resolved := resolveCompletionTarget(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("observed entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(
				evaluationText(subject),
				evaluationText(fmt.Sprintf("observed locator at index %d", index)),
				evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)),
			)
		}
	}
	return provedResult()
}

func (completionObservedItemsExpectationValue) completionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation completionObservedItemsExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	const subject = "observed completion items"
	if len(result.Items) != len(expectation.states) {
		return disprovedResult(
			evaluationText(fmt.Sprintf("%s snapshot (partial=%t truncated=%t)", subject, result.Partial, result.Truncated)),
			evaluationText(fmt.Sprintf("observed count=%d", len(expectation.states))),
			evaluationText(fmt.Sprintf("observed count=%d", len(result.Items))),
		)
	}
	for index, state := range expectation.states {
		resolved := resolveCompletionItem(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("observed entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(
				evaluationText(subject),
				evaluationText(fmt.Sprintf("observed locator at index %d", index)),
				evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)),
			)
		}
	}
	return provedResult()
}

func (completionObservedGroupsExpectationValue) completionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation completionObservedGroupsExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	const subject = "observed completion groups"
	if len(result.Groups) != len(expectation.states) {
		return disprovedResult(
			evaluationText(fmt.Sprintf("%s snapshot (partial=%t truncated=%t)", subject, result.Partial, result.Truncated)),
			evaluationText(fmt.Sprintf("observed count=%d", len(expectation.states))),
			evaluationText(fmt.Sprintf("observed count=%d", len(result.Groups))),
		)
	}
	for index, state := range expectation.states {
		resolved := resolveCompletionGroup(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("observed entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(
				evaluationText(subject),
				evaluationText(fmt.Sprintf("observed locator at index %d", index)),
				evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)),
			)
		}
	}
	return provedResult()
}

func (completionObservedNativeRequestsExpectationValue) completionExpectation() {}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func (expectation completionObservedNativeRequestsExpectationValue) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	const subject = "observed completion native requests"
	if len(result.NativeRequests) != len(expectation.states) {
		return disprovedResult(
			evaluationText(fmt.Sprintf("%s snapshot (partial=%t truncated=%t)", subject, result.Partial, result.Truncated)),
			evaluationText(fmt.Sprintf("observed count=%d", len(expectation.states))),
			evaluationText(fmt.Sprintf("observed count=%d", len(result.NativeRequests))),
		)
	}
	for index, state := range expectation.states {
		resolved := resolveCompletionNativeRequest(state, result)
		if resolved.status != locatorResolutionFound {
			return resolutionOutcome(resolved.status, evaluationText(subject), evaluationText(fmt.Sprintf("observed entry %d", index)), resolved.actual)
		}
		if resolved.index != index {
			return disprovedResult(
				evaluationText(subject),
				evaluationText(fmt.Sprintf("observed locator at index %d", index)),
				evaluationText(fmt.Sprintf("resolved to index %d", resolved.index)),
			)
		}
	}
	return provedResult()
}

func (completionObservedMessagesExpectation) completionExpectation() {}

func (expected completionObservedMessagesExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if slices.Equal([]cli.CompletionMessage(expected), result.Messages) {
		return provedResult()
	}
	subject := fmt.Sprintf(
		"observed completion messages snapshot (partial=%t truncated=%t)",
		result.Partial,
		result.Truncated,
	)
	for _, message := range expected {
		if message.Sensitive {
			return disprovedResult(
				evaluationText(subject),
				evaluationText(fmt.Sprintf("count=%d with sensitive text <redacted>", len(expected))),
				evaluationText(fmt.Sprintf("count=%d", len(result.Messages))),
			)
		}
	}
	for _, message := range result.Messages {
		if message.Sensitive {
			return disprovedResult(
				evaluationText(subject),
				evaluationText(fmt.Sprintf("count=%d", len(expected))),
				evaluationText(fmt.Sprintf("count=%d with sensitive text <redacted>", len(result.Messages))),
			)
		}
	}
	if len(expected) != len(result.Messages) {
		field := collectionCardinalityField{
			name:  "expected completion messages",
			count: len(expected),
		}
		collectionCardinalityField := collectionCardinalityField{
			name:  "observed completion messages",
			count: len(result.Messages),
		}
		return disprovedResult(
			evaluationText(subject),
			field,
			collectionCardinalityField,
		)
	}
	for index := range expected {
		if expected[index] != result.Messages[index] {
			field := completionMessageField{
				index: index,
				value: expected[index],
			}
			completionMessageField := completionMessageField{
				index: index,
				value: result.Messages[index],
			}
			return disprovedResult(
				evaluationText(subject),
				field,
				completionMessageField,
			)
		}
	}
	panic("clitest: completion message mismatch lost")
}

func (completionTargetIdentityExpectation) completionExpectation() {}

func (expectation completionTargetIdentityExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionTarget(expectation.locator, result)
	if resolved.status != locatorResolutionFound {
		return resolutionOutcome(resolved.status, resolved.label, evaluationText("target identity relation"), resolved.actual)
	}
	kind := completionTargetIdentityExpectationKind(expectation.kind)
	switch kind {
	case completionTargetIdentityCommand:
		if resolved.value.Command != 0 {
			return provedResult()
		}
		return disprovedResult(resolved.label, evaluationText("command identity present"), evaluationText("command identity is absent"))
	case completionTargetIdentityValue:
		if resolved.value.ValueID != 0 {
			return provedResult()
		}
		return disprovedResult(resolved.label, evaluationText("value identity present"), evaluationText("value identity is absent"))
	case completionTargetIdentitySelectedCommand:
		if resolved.value.Command == 0 {
			return disprovedResult(resolved.label, evaluationText("owned by selected command"), evaluationText("target command identity is absent"))
		}
		if result.Selection.Selected == 0 {
			return disprovedResult(resolved.label, evaluationText("owned by selected command"), evaluationText("selected-command identity is absent"))
		}
		if resolved.value.Command == result.Selection.Selected {
			return provedResult()
		}
		return disprovedResult(
			resolved.label,
			evaluationText("owned by selected command"),
			evaluationText("target is not owned by the selected command"),
		)
	default:
		return disprovedResult(
			resolved.label,
			evaluationText("valid target identity expectation"),
			evaluationText("invalid private identity expectation"),
		)
	}
}

func (completionRangeCollectionField) evaluationField() {}

func (completionMessageField) evaluationField() {}

func (completionCursorField) evaluationField() {}

func (completionSpanField) evaluationField() {}

func (completionObservedInputSourcesExpectation) completionExpectation() {}

func (expectation completionObservedInputSourcesExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	report := cli.DiagnosticReport{
		Sources:     result.Sources,
		Diagnostics: result.Diagnostics,
		Origins:     result.Origins,
		Repairs:     result.Repairs,
		Truncated:   result.DiagnosticsTruncated,
	}
	return observedInputSourcesExactly(report, expectation.kind, expectation.sources)
}

func (completionItemPresentationExpectation) completionExpectation() {}

func (e completionItemPresentationExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	resolved := resolveCompletionItem(e.locator, result)
	if resolved.status != locatorResolutionFound {
		return resolutionOutcome(
			resolved.status,
			resolved.label,
			presentationField(e.displayText, e.description),
			resolved.actual,
		)
	}
	if resolved.value.DisplayText == e.displayText && resolved.value.Description == e.description {
		return provedResult()
	}
	if resolved.value.Sensitive {
		return disprovedResult(
			qualifySubject(resolved.label, evaluationText("presentation")),
			evaluationText("redacted expected presentation"),
			evaluationText("redacted observed presentation"),
		)
	}
	return disprovedResult(
		qualifySubject(resolved.label, evaluationText("presentation")),
		presentationField(e.displayText, e.description),
		presentationField(resolved.value.DisplayText, resolved.value.Description),
	)
}

func (completionBooleanExpectation) completionExpectation() {}

func (expectation completionBooleanExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	var actual bool
	switch expectation.field {
	case completionBooleanPartial:
		actual = result.Partial
	case completionBooleanTruncated:
		actual = result.Truncated
	case completionBooleanRefreshOnInput:
		actual = result.RefreshOnInput
	default:
		panic("clitest: invalid completion boolean expectation")
	}
	if actual == expectation.want {
		return provedResult()
	}
	return disprovedResult(evaluationText(expectation.subject), evaluationText(strconv.FormatBool(expectation.want)), evaluationText(strconv.FormatBool(actual)))
}

func (completionReplaceExpectation) completionExpectation() {}

func (expectation completionReplaceExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if result.Replace == expectation.span {
		return provedResult()
	}
	field := completionSpanField{
		value: expectation.span,
	}
	completionSpanField := completionSpanField{
		value: result.Replace,
	}
	return disprovedResult(
		evaluationText("completion replacement"),
		field,
		completionSpanField,
	)
}

func (completionRankingExpectation) completionExpectation() {}

func (expectation completionRankingExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if result.RankingAlgorithm == expectation.algorithm {
		return provedResult()
	}
	return disprovedResult(
		evaluationText("completion ranking"),
		quotedTextField(expectation.algorithm),
		quotedTextField(result.RankingAlgorithm),
	)
}

func (completionSpacingExpectation) completionExpectation() {}

func (expectation completionSpacingExpectation) evaluateCompletion(result cli.Completion, _ error) evaluationResult {
	if result.Spacing == expectation.spacing {
		return provedResult()
	}
	return disprovedResult(
		evaluationText("completion spacing"),
		evaluationText(fmt.Sprintf("%v", expectation.spacing)),
		evaluationText(fmt.Sprintf("%v", result.Spacing)),
	)
}
