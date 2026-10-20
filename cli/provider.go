package cli

import (
	"context"
	"slices"
)

// CommandProvider materializes one operation-scoped batch of child-command contributions from the
// current command configuration and declared dependencies. The supplied CommandBatch is valid only
// for that evaluation; callback failure or invalid output rejects the complete batch rather than
// publishing a partial contribution.
type CommandProvider[C any] func(context.Context, *CommandBatch[C]) error

// CommandBatch is the callback-scoped declaration surface for one CommandProvider evaluation.
// Config and Dependencies describe that operation-local evaluation; Command, Alias, Shortcut, and
// Localization stage one atomic provider batch, so callback failure or invalid output commits none
// of it. The zero value and retained, concurrent, or post-callback use are invalid. CommandBatch
// values are noncomparable.
type CommandBatch[C any] struct {
	Config       *C
	Dependencies Dependencies

	session *declarationSession
}

// Command contributes cmd to the active provider batch and returns its declaration handle in the
// owning parent declaration scope.
func (batch *CommandBatch[C]) Command[T any](cmd Command[T], source ...SourceOption) CommandDeclaration[C] {
	return providerBatchCommand(batch, cmd, source)
}

// Localization contributes localization data to the active provider batch.
func (batch *CommandBatch[C]) Localization(localization Localization) {
	providerBatchLocalization(batch, localization)
}

// Alias contributes alias for target to the active provider batch.
func (batch *CommandBatch[C]) Alias(target CommandTarget, alias CommandAlias, source ...SourceOption) {
	providerBatchAlias(batch, target, alias, source)
}

// Shortcut contributes shortcut for target and returns a batch-scoped binding for defaults or
// replacement.
func (batch *CommandBatch[C]) Shortcut(target CommandTarget, shortcut CommandShortcut, source ...SourceOption) CommandShortcutBinding {
	return providerBatchShortcut(batch, target, shortcut, source)
}

func declareCommandProvider[C any](subcommands *Subcommands[C], provider CommandProvider[C], dependencies []Dependency) {
	if subcommands == nil || subcommands.session == nil {
		return
	}
	provide, providerOK := compileCommandProvider(provider)
	descriptors, dependenciesOK := dependencyDescriptors(dependencies)
	if !providerOK || !dependenciesOK {
		stageCommandProvider(subcommands.session, subcommands.profile, subcommands.helpGroup, nil, nil, false)
		return
	}
	stageCommandProvider(subcommands.session, subcommands.profile, subcommands.helpGroup, descriptors, provide, true)
}

func compileCommandProvider[C any](provider CommandProvider[C]) (commandProviderFunc, bool) {
	if provider == nil {
		return nil, false
	}
	provide := func(ctx context.Context, call providerCall) (providerBatch, error) {
		return captureProviderBatch(ctx, call, provider)
	}
	return provide, true
}

func stageCommandProvider(session *declarationSession, profile grammarProfileCaseRef, helpGroup helpGroupRef, pending []destinationDependency, provide commandProviderFunc, valid bool) {
	if session == nil {
		return
	}

	session.use(func(draft *declarationDraft, command commandRef) {
		dependencies, ok := freezeAuthoringDestinationDependencies(draft, pending)
		record := commandProviderDraft{
			position:     draft.nextPosition(),
			command:      command,
			profile:      profile,
			helpGroup:    helpGroup,
			dependencies: dependencies,
			provide:      provide,
			valid:        valid && ok,
		}
		draft.providers.append(draft, record)
	})
}

func stageStaticProviderDefinitions(draft *declarationCandidate, input declarationPlan, source *declarationDraft, state *staticAcceptanceState) error {
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordProvider {
			continue
		}
		provider := state.slots[index].provider
		value := source.providers.entries[record.familyIndex].value
		if !provider.ID().present() || !value.valid || value.provide == nil {
			panic("cli: validated provider has unavailable accepted state")
		}
		dependencies := make(
			[]dependencyRequirement,
			0,
			len(source.providers.entries[record.familyIndex].value.dependencies),
		)
		for _, dependency := range value.dependencies {
			id := state.mustValueID(dependency.destination)
			dependencies = append(dependencies, acceptDependencyRequirement(id, dependency.optional))
		}
		provide := value.provide
		definition := acceptProviderDefinition(dependencies)
		draft.ProviderDefinition(provider, definition, provide, input.positionFor(index))
	}
	return nil
}

type providerCall struct {
	provider CommandProviderID
	parent   CommandID
	scope    identityScope
	release  ReleaseScheme

	config       any
	dependencies Dependencies
}

type commandProviderFunc func(context.Context, providerCall) (providerBatch, error)

type providerDefinition struct {
	dependencies []dependencyRequirement
	valid        bool
}

func acceptProviderDefinition(dependencies []dependencyRequirement) providerDefinition {
	definition := providerDefinition{
		dependencies: append([]dependencyRequirement(nil), dependencies...),
		valid:        true,
	}

	return definition
}

func (definition providerDefinition) Valid() bool { return definition.valid }

func (definition providerDefinition) Dependencies() []dependencyRequirement {
	return append([]dependencyRequirement(nil), definition.dependencies...)
}

// providerLexicalDisposition is the acceptance result for one provider batch
// at the shared lexical-namespace boundary. A provider is first tested against
// static authority and its own staged batch; only providers that survive that
// intrinsic phase participate in peer collision settlement.
type providerLexicalDisposition uint8

const (
	providerLexicalAccepted providerLexicalDisposition = iota
	providerLexicalRejectedIntrinsic
	providerLexicalRejectedPeer
)

// providerLexicalEntry is one staged provider contribution to a single
// concrete lexical namespace. Provider is private caller identity used only to
// keep complete batches atomic; it is not semantic declaration identity.
type providerLexicalEntry[P comparable] struct {
	Provider P
	Name     string
}

const minimumConflictingProviders = 2

// settleProviderLexicalNamespace applies the same bounded normalization rule
// used by static acceptance, then performs the provider-specific atomic
// settlement required by materialization:
//
//   - collision with static authority rejects only the provider batch;
//   - duplicate normalized names inside one provider reject that batch before
//     peer settlement, so an intrinsically invalid batch cannot poison a peer;
//   - collisions between surviving provider batches reject every participating
//     provider, with no winner selected by declaration or completion order.
//
// The result contains an entry for every provider present in entries.
func settleProviderLexicalNamespace[P comparable](mode NameNormalization, static []string, entries []providerLexicalEntry[P]) map[P]providerLexicalDisposition {
	result := make(map[P]providerLexicalDisposition)
	staticNames := make(map[string]struct{}, len(static))
	for _, raw := range static {
		name, ok := normalizeName(mode, raw)
		if !ok || name == "" {
			continue
		}
		staticNames[name] = struct{}{}
	}

	type normalizedEntry struct {
		provider P
		name     string
	}
	prepared := make([]normalizedEntry, 0, len(entries))
	perProvider := make(map[P]map[string]struct{})
	for _, entry := range entries {
		if _, seen := result[entry.Provider]; !seen {
			result[entry.Provider] = providerLexicalAccepted
		}
		name, ok := normalizeName(mode, entry.Name)
		if !ok || name == "" {
			result[entry.Provider] = providerLexicalRejectedIntrinsic
			continue
		}
		seen := perProvider[entry.Provider]
		if seen == nil {
			seen = make(map[string]struct{})
			perProvider[entry.Provider] = seen
		}
		if _, duplicate := seen[name]; duplicate {
			result[entry.Provider] = providerLexicalRejectedIntrinsic
		}
		seen[name] = struct{}{}
		if _, collision := staticNames[name]; collision {
			result[entry.Provider] = providerLexicalRejectedIntrinsic
		}
		prepared = append(prepared, normalizedEntry{
			provider: entry.Provider,
			name:     name,
		})
	}

	byName := make(map[string][]P)
	for _, entry := range prepared {
		if result[entry.provider] != providerLexicalAccepted {
			continue
		}
		providers := byName[entry.name]
		already := slices.Contains(providers, entry.provider)
		if !already {
			byName[entry.name] = append(providers, entry.provider)
		}
	}
	for _, providers := range byName {
		if len(providers) < minimumConflictingProviders {
			continue
		}
		for _, provider := range providers {
			result[provider] = providerLexicalRejectedPeer
		}
	}
	return result
}

type providerRecord struct {
	meta       recordMeta[CommandProviderID]
	parent     CommandID
	definition providerDefinition
}

func acceptProviderRecord(id CommandProviderID, parent CommandID, position declarationPosition) providerRecord {
	mustCommand(parent)
	return providerRecord{
		meta:   acceptRecordMeta(id, position),
		parent: parent,
	}
}
