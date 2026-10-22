package cli

import (
	"cmp"
	"slices"
)

// grammarProfilesOverlap reports whether two declaration profiles can be
// simultaneously active in one grammar branch. An unspecified profile is
// common to every branch.
func grammarProfilesOverlap(left, right GrammarProfileID) bool {
	return !left.present() || !right.present() || left == right
}

// conditionTargetProfile resolves the command/profile branch that owns a
// structural condition target.
func conditionTargetProfile(target conditionTarget, commands map[CommandID]commandRecord, flags map[FlagID]flagRecord, forms map[ArgumentFormID]argumentFormRecord, terms []argumentTermRecord) (CommandID, GrammarProfileID, bool) {
	switch target.Kind() {
	case conditionTargetCommand:
		_, ok := commands[target.Command()]
		return target.Command(), GrammarProfileID(0), ok
	case conditionTargetFlag:
		record, ok := flags[target.Flag()]
		return record.command, record.meta.profile, ok
	case conditionTargetArgumentForm:
		record, ok := forms[target.ArgumentForm()]
		return record.command, record.meta.profile, ok
	case conditionTargetArgumentTerm:
		for _, term := range terms {
			if term.meta.id != target.ArgumentTerm() {
				continue
			}
			form, ok := forms[term.form]
			return form.command, form.meta.profile, ok
		}
	case conditionTargetUnknown:
	}
	return CommandID(0), GrammarProfileID(0), false
}

// commandSameOrAncestor reports whether candidate is command or one of its
// ancestors in the validated command tree.
func commandSameOrAncestor(commands map[CommandID]commandRecord, candidate, command CommandID) bool {
	for current := command; current.present(); {
		if current == candidate {
			return true
		}
		record, ok := commands[current]
		if !ok {
			return false
		}
		parent, ok := record.Parent()
		if !ok {
			return false
		}
		current = parent
	}
	return false
}

func (draft *publicationDraft) validateCandidate(decision acceptanceDecision) (grammarModel, error) {
	if decision.Boundary() != acceptanceBoundaryProviderBatch {
		return grammarModel{}, errAcceptanceBoundary
	}
	rejections := append(decision.Rejections(), draft.preflight...)
	rejections = append(rejections, draft.validateReferences(true)...)
	if len(rejections) == 0 {
		rejections = append(rejections, validateProofInvariants(draft.input)...)
	}
	if len(rejections) != 0 {
		combined := acceptAcceptanceDecision(acceptanceBoundaryProviderBatch, rejections)
		failure := rejectionError{
			rejections: combined.Rejections(),
		}
		return grammarModel{}, failure
	}
	grammar, hasGrammar, grammarRejections := draft.buildGrammar()
	if len(grammarRejections) != 0 {
		combined := acceptAcceptanceDecision(acceptanceBoundaryProviderBatch, grammarRejections)
		failure := rejectionError{
			rejections: combined.Rejections(),
		}
		return grammarModel{}, failure
	}
	if !hasGrammar {
		failure := rejectionError{
			rejections: []rejection{{
				Kind: rejectionStructuralInvalidity,
			}},
		}
		return grammarModel{}, failure
	}
	return grammar, nil
}

type referenceValidationIndex struct {
	roots          int
	commands       map[CommandID]commandRecord
	destinations   map[ValueID]destinationRecord
	flags          map[FlagID]flagRecord
	forms          map[ArgumentFormID]argumentFormRecord
	entryGrammar   map[CommandEntryID]commandEntryGrammar
	commandEntries map[CommandEntryID]commandEntryRecord
	profiles       map[GrammarProfileID]grammarProfileRecord
	policies       map[SignalPolicyID]signalPolicyRecord
	endpoints      map[SignalEndpointID]signalEndpointRecord
	counters       map[SignalCounterID]signalCounterRecord
}

func (draft *declarationCandidate) validateReferences(provider bool) []rejection {
	index := referenceValidationIndex{
		commands:       make(map[CommandID]commandRecord, len(draft.input.Commands)),
		destinations:   make(map[ValueID]destinationRecord, len(draft.input.Destinations)),
		flags:          make(map[FlagID]flagRecord, len(draft.input.Flags)),
		forms:          make(map[ArgumentFormID]argumentFormRecord, len(draft.input.ArgumentForms)),
		entryGrammar:   make(map[CommandEntryID]commandEntryGrammar),
		commandEntries: make(map[CommandEntryID]commandEntryRecord, len(draft.input.CommandEntries)),
		profiles:       make(map[GrammarProfileID]grammarProfileRecord, len(draft.input.GrammarProfiles)),
		policies:       make(map[SignalPolicyID]signalPolicyRecord, len(draft.input.SignalPolicies)),
		endpoints:      make(map[SignalEndpointID]signalEndpointRecord, len(draft.input.SignalEndpoints)),
		counters:       make(map[SignalCounterID]signalCounterRecord, len(draft.input.SignalCounters)),
	}
	for _, record := range draft.input.Commands {
		index.commands[record.meta.id] = record
		if _, hasParent := record.Parent(); !hasParent {
			index.roots++
		}
	}
	for _, record := range draft.input.Destinations {
		index.destinations[record.meta.id] = record
	}
	for _, record := range draft.input.Flags {
		index.flags[record.meta.id] = record
	}
	for _, record := range draft.input.ArgumentForms {
		index.forms[record.meta.id] = record
	}
	if draft.grammar != nil {
		for _, entry := range draft.grammar.entries {
			index.entryGrammar[entry.id] = entry
		}
	}
	for _, record := range draft.input.CommandEntries {
		index.commandEntries[record.meta.id] = record
	}
	for _, record := range draft.input.GrammarProfiles {
		index.profiles[record.meta.id] = record
	}
	for _, record := range draft.input.SignalPolicies {
		index.policies[record.meta.id] = record
	}
	for _, record := range draft.input.SignalEndpoints {
		index.endpoints[record.meta.id] = record
	}
	for _, record := range draft.input.SignalCounters {
		index.counters[record.meta.id] = record
	}

	var rejections []rejection
	rejections = draft.validateStructuralReferences(index, rejections)
	rejections = draft.validateDeclarationRelationships(index, rejections)
	rejections = draft.validatePolicyReferences(index, rejections)
	rejections = draft.validateSignalReferences(index, rejections)
	return draft.validateValueAndProfileReferences(index, rejections, provider)
}

func (draft *declarationCandidate) validateStructuralReferences(index referenceValidationIndex, rejections []rejection) []rejection {
	commands := index.commands
	destinations := index.destinations
	flags := index.flags
	forms := index.forms
	if index.roots != 1 {
		rejections = append(rejections, rejection{
			Kind:  rejectionStructuralInvalidity,
			Order: 0,
		})
	}
	for _, record := range draft.input.Commands {
		if parent, ok := record.Parent(); ok {
			if _, exists := commands[parent]; !exists {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
	}

	for _, record := range draft.input.Destinations {
		if _, exists := commands[record.command]; !exists {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}
	for _, record := range draft.input.Commands {
		access := record.access
		if !access.Valid() {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		for _, binding := range access.Bindings() {
			for _, destination := range draft.input.Destinations {
				if destination.command == record.meta.id {
					if destinationBinding, ok := destination.Binding(); ok &&
						semanticConfigurationBindingsOverlap(binding, destinationBinding) {
						rejections = append(rejections, semanticRejection(record.meta.position))
					}
				}
			}
		}
		for _, relationship := range append(access.Values(), access.OptionalValues()...) {
			source, ok := destinations[relationship.Source()]
			if !ok || source.command == record.meta.id ||
				!commandSameOrAncestor(commands, source.command, record.meta.id) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		for _, relationship := range access.ValueStates() {
			source, ok := destinations[relationship.Source()]
			if !ok || !commandSameOrAncestor(commands, source.command, record.meta.id) {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
	}
	rejections = append(
		rejections,
		draft.validateDestinationProducers(commands, destinations, draft.input.Destinations)...,
	)
	for _, record := range draft.input.Flags {
		destination, destinationOK := destinations[record.destination]
		_, commandOK := commands[record.command]
		if !destinationOK || !commandOK || destination.command != record.command {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		if completion, present := record.Completion(); present {
			for _, dependency := range completion.Dependencies() {
				target, ok := destinations[dependency.Destination()]
				if !ok || !commandSameOrAncestor(commands, target.command, record.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
	}
	for _, record := range draft.input.FlagSpellings {
		if _, exists := flags[record.flag]; !exists {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}

	for _, record := range draft.input.ArgumentForms {
		if _, exists := commands[record.command]; !exists {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}
	for _, record := range draft.input.ArgumentTerms {
		form, formOK := forms[record.form]
		if !formOK {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		if ValueID, hasDestination := record.Destination(); hasDestination {
			destination, destinationOK := destinations[ValueID]
			if !destinationOK || destination.command != form.command {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
		}
		if completion, present := record.Completion(); present {
			for _, dependency := range completion.Dependencies() {
				target, ok := destinations[dependency.Destination()]
				if !ok || !commandSameOrAncestor(commands, target.command, form.command) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
	}

	return rejections
}

func (draft *declarationCandidate) validateDeclarationRelationships(index referenceValidationIndex, rejections []rejection) []rejection {
	commands := index.commands
	destinations := index.destinations
	flags := index.flags
	forms := index.forms
	entryGrammar := index.entryGrammar
	commandEntries := index.commandEntries
	for _, record := range draft.input.CommandEntries {
		if _, ok := commands[record.parent]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		if _, ok := commands[record.target]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		definition, hasShortcut := record.Shortcut()
		grammarEntry, hasGrammarEntry := entryGrammar[record.meta.id]
		switch {
		case hasShortcut:
			if !hasGrammarEntry || grammarEntry.kind != CommandEntryShortcut ||
				grammarEntry.parent != record.parent ||
				grammarEntry.target != record.target {
				rejections = append(rejections, semanticRejection(record.meta.position))
			}
			seenDestinations := make(map[ValueID]struct{})
			seenSpellings := make(map[string]struct{})
			for _, value := range definition.Defaults() {
				switch value.Kind() {
				case shortcutDefaultKindDestination:
					destination, ok := destinations[value.Destination()]
					if !ok || !commandSameOrAncestor(commands, destination.command, record.target) {
						rejections = append(rejections, semanticRejection(record.meta.position))
					}
					if _, duplicate := seenDestinations[value.Destination()]; duplicate {
						rejections = append(rejections, semanticRejection(record.meta.position))
					}
					seenDestinations[value.Destination()] = struct{}{}
				case shortcutDefaultKindFlag:
					if _, duplicate := seenSpellings[value.Spelling()]; duplicate {
						rejections = append(rejections, semanticRejection(record.meta.position))
					}
					seenSpellings[value.Spelling()] = struct{}{}
				default:
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		case hasGrammarEntry && grammarEntry.kind == CommandEntryShortcut:
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}
	for _, record := range draft.input.CommandEntries {
		replacement, hasReplacement := record.Replacement()
		if !hasReplacement {
			continue
		}
		target, ok := commandEntries[replacement]
		grammarEntry, grammarOK := entryGrammar[record.meta.id]
		targetGrammar, targetGrammarOK := entryGrammar[replacement]
		if !ok || replacement == record.meta.id || !grammarOK || !targetGrammarOK ||
			grammarEntry.kind != CommandEntryShortcut ||
			targetGrammar.kind != CommandEntryShortcut ||
			target.parent != record.parent {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}
	for _, start := range draft.input.CommandEntries {
		seen := make(map[CommandEntryID]struct{})
		current := start
		for {
			replacement, ok := current.Replacement()
			if !ok {
				break
			}
			if _, duplicate := seen[current.meta.id]; duplicate {
				rejections = append(rejections, semanticRejection(start.meta.position))
				break
			}
			seen[current.meta.id] = struct{}{}
			next, exists := commandEntries[replacement]
			if !exists {
				break
			}
			current = next
		}
	}
	for _, record := range draft.input.ActionEntries {
		if _, ok := commands[record.parent]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}
	defaultsByParent := make(map[CommandID][]defaultRecord)
	for _, record := range draft.input.Defaults {
		parent, parentOK := commands[record.parent]
		target, targetOK := commands[record.target]
		if !parentOK || !targetOK {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		targetParent, immediateChild := target.Parent()
		if !immediateChild || targetParent != parent.meta.id {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		if condition, conditional := record.Condition(); conditional {
			for _, dependency := range condition.Dependencies() {
				destination, ok := destinations[dependency]
				if !ok || !commandSameOrAncestor(commands, destination.command, parent.meta.id) {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
		defaultsByParent[parent.meta.id] = append(defaultsByParent[parent.meta.id], record)
	}
	for _, defaults := range defaultsByParent {
		unconditional := 0
		for _, record := range defaults {
			if _, conditional := record.Condition(); !conditional {
				unconditional++
			}
		}
		if unconditional > 1 {
			for _, record := range defaults {
				if _, conditional := record.Condition(); !conditional {
					rejections = append(rejections, semanticRejection(record.meta.position))
				}
			}
		}
	}

	// Generic exact-declaration replacement relationships are validated as one
	// bounded graph after every referenced declaration family exists. Domain,
	// action-entry, and shortcut replacement keep their family-specific records.
	if len(draft.input.Replacements) == 0 {
		return rejections
	}
	spellingByID := make(map[FlagSpellingID]flagSpellingRecord, len(draft.input.FlagSpellings))
	for _, record := range draft.input.FlagSpellings {
		spellingByID[record.meta.id] = record
	}
	termByID := make(map[ArgumentTermID]argumentTermRecord, len(draft.input.ArgumentTerms))
	for _, record := range draft.input.ArgumentTerms {
		termByID[record.meta.id] = record
	}
	environmentByID := make(map[EnvironmentInputID]environmentInputRecord, len(draft.input.EnvironmentInputs))
	for _, record := range draft.input.EnvironmentInputs {
		environmentByID[record.meta.id] = record
	}

	entryKind := func(id CommandEntryID) CommandEntryKind {
		if grammar, ok := entryGrammar[id]; ok {
			return grammar.kind
		}
		return CommandEntryUnknown
	}
	termKind := func(id ArgumentTermID) ArgumentTermKind {
		record, ok := termByID[id]
		if !ok || draft.grammar == nil {
			return ArgumentTermUnknown
		}
		for _, grammar := range draft.grammar.terms[record.form] {
			if grammar.id == id {
				return grammar.kind
			}
		}
		return ArgumentTermUnknown
	}
	subject := func(id declarationReplacementIdentity) (replacementSubject, bool) {
		switch id.Kind() {
		case declarationReplacementKindCommandEntry:
			record, ok := commandEntries[id.CommandEntry()]
			if !ok || entryKind(record.meta.id) != CommandEntryCanonical {
				return replacementSubject{}, false
			}
			value := replacementSubject{
				identity:  id,
				command:   record.parent,
				profile:   record.meta.profile,
				lifecycle: record.lifecycle,
				position:  record.meta.position,
			}
			return value, true
		case declarationReplacementKindFlag:
			record, ok := flags[id.Flag()]
			if !ok {
				return replacementSubject{}, false
			}
			value := replacementSubject{
				identity:  id,
				command:   record.command,
				profile:   record.meta.profile,
				lifecycle: record.lifecycle,
				position:  record.meta.position,
			}
			return value, true
		case declarationReplacementKindFlagSpelling:
			record, ok := spellingByID[id.FlagSpelling()]
			if !ok {
				return replacementSubject{}, false
			}
			flag, ok := flags[record.flag]
			if !ok {
				return replacementSubject{}, false
			}
			result := replacementSubject{
				identity:  id,
				command:   flag.command,
				flag:      record.flag,
				profile:   flag.meta.profile,
				lifecycle: record.lifecycle,
				position:  record.meta.position,
			}
			return result, true
		case declarationReplacementKindArgumentForm:
			record, ok := forms[id.ArgumentForm()]
			if !ok {
				return replacementSubject{}, false
			}
			value := replacementSubject{
				identity:  id,
				command:   record.command,
				profile:   record.meta.profile,
				lifecycle: record.lifecycle,
				position:  record.meta.position,
			}
			return value, true
		case declarationReplacementKindArgumentTerm:
			record, ok := termByID[id.ArgumentTerm()]
			if !ok {
				return replacementSubject{}, false
			}
			form, ok := forms[record.form]
			if !ok {
				return replacementSubject{}, false
			}
			result := replacementSubject{
				identity:  id,
				command:   form.command,
				termKind:  termKind(record.meta.id),
				profile:   form.meta.profile,
				lifecycle: record.lifecycle,
				position:  record.meta.position,
			}
			return result, true
		case declarationReplacementKindEnvironmentInput:
			record, ok := environmentByID[id.EnvironmentInput()]
			if !ok {
				return replacementSubject{}, false
			}
			value := replacementSubject{
				identity:  id,
				command:   record.command,
				profile:   record.meta.profile,
				lifecycle: record.lifecycle,
				position:  record.meta.position,
			}
			return value, true
		default:
			return replacementSubject{}, false
		}
	}
	seenSource := make(map[declarationReplacementIdentity]struct{})
	edges := make(map[declarationReplacementIdentity]declarationReplacementIdentity)
	scheme := draft.application.release
	for _, relation := range draft.input.Replacements {
		source, sourceOK := subject(relation.source)
		target, targetOK := subject(relation.target)
		valid := sourceOK && targetOK && source.identity != target.identity &&
			validReplacementProfile(source.profile, target.profile) &&
			validReplacementLifecycle(scheme, source.lifecycle, target.lifecycle)
		if _, duplicate := seenSource[relation.source]; duplicate {
			valid = false
		}
		seenSource[relation.source] = struct{}{}
		if valid {
			switch source.identity.Kind() {
			case declarationReplacementKindCommandEntry:
				valid = target.identity.Kind() == declarationReplacementKindCommandEntry &&
					source.command == target.command
			case declarationReplacementKindFlag:
				valid = source.command == target.command &&
					(target.identity.Kind() == declarationReplacementKindFlag || (target.identity.Kind() == declarationReplacementKindArgumentTerm && termKind(target.identity.ArgumentTerm()) != ArgumentTermLiteral))
			case declarationReplacementKindFlagSpelling:
				valid = target.identity.Kind() == declarationReplacementKindFlagSpelling && source.flag.present() &&
					source.flag == target.flag
			case declarationReplacementKindArgumentTerm:
				if source.termKind == ArgumentTermLiteral {
					valid = target.identity.Kind() == declarationReplacementKindArgumentTerm &&
						target.termKind == ArgumentTermLiteral &&
						source.command == target.command
				} else {
					valid = source.termKind != ArgumentTermUnknown && source.command == target.command &&
						(target.identity.Kind() == declarationReplacementKindFlag || (target.identity.Kind() == declarationReplacementKindArgumentTerm && target.termKind != ArgumentTermLiteral && target.termKind != ArgumentTermUnknown))
				}
			case declarationReplacementKindArgumentForm:
				valid = target.identity.Kind() == declarationReplacementKindArgumentForm &&
					source.command == target.command
			case declarationReplacementKindEnvironmentInput:
				valid = target.identity.Kind() == declarationReplacementKindEnvironmentInput &&
					source.command == target.command
			default:
				valid = false
			}
		}
		if !valid {
			rejections = append(rejections, semanticRejection(relation.position))
			continue
		}
		edges[relation.source] = relation.target
	}
	for start := range edges {
		seen := make(map[declarationReplacementIdentity]struct{})
		for current := start; ; {
			if _, duplicate := seen[current]; duplicate {
				rejections = append(rejections, rejection{
					Kind:  rejectionStructuralInvalidity,
					Order: uint64(subjectPosition(start, draft.input.Replacements)),
				})
				break
			}
			seen[current] = struct{}{}
			next, ok := edges[current]
			if !ok {
				break
			}
			current = next
		}
	}
	return rejections
}

func (draft *declarationCandidate) validateValueAndProfileReferences(index referenceValidationIndex, rejections []rejection, provider bool) []rejection {
	commands := index.commands
	destinations := index.destinations
	flags := index.flags
	forms := index.forms
	profiles := index.profiles
	policies := index.policies
	rejections = append(
		rejections,
		validateValueDomains(
			commands,
			destinations,
			flags,
			profiles,
			draft.input.DomainEntries,
			draft.input.ValueDomains,
		)...,
	)
	rejections = append(rejections, draft.validateLocaleSelector(destinations, flags, provider)...)
	environmentByDestination := make(map[ValueID][]environmentInputRecord)
	for _, record := range draft.input.EnvironmentInputs {
		destination, destinationOK := destinations[record.destination]
		_, commandOK := commands[record.command]
		if !destinationOK || !commandOK || destination.command != record.command {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
		if !validEnvironmentInputSyntax(record) {
			rejections = append(rejections, semanticRejection(record.meta.position))
			continue
		}
		for _, prior := range environmentByDestination[record.destination] {
			if grammarProfilesOverlap(prior.meta.profile, record.meta.profile) && environmentSelectorsOverlap(prior, record) {
				rejections = append(rejections, semanticRejection(record.meta.position))
				break
			}
		}
		environmentByDestination[record.destination] = append(environmentByDestination[record.destination], record)
	}
	rejections = append(rejections, draft.validateGrammarProfiles(commands, destinations, flags, forms)...)
	for _, record := range draft.input.SignalCounters {
		if _, ok := policies[record.policy]; !ok {
			rejections = append(rejections, semanticRejection(record.meta.position))
		}
	}
	return rejections
}

func semanticRejection(position declarationPosition) rejection {
	candidateRejection := rejection{
		Kind:  rejectionStructuralInvalidity,
		Order: uint64(position),
	}
	return candidateRejection
}

// rejectionKind is private implementation vocabulary for deterministic
// acceptance control. It is not a public DiagnosticCode family and does not
// create a generic compiler-finding semantic object.
type rejectionKind uint8

const (
	rejectionNone rejectionKind = iota
	rejectionForeignOrigin
	rejectionForeignScope
	rejectionIllegalScope
	rejectionUnavailablePhase
	rejectionDuplicateDependency
	rejectionDependencyCycle
	rejectionMissingDependencyTarget
	rejectionInvalidNormalization
	rejectionNormalizedCollision
	rejectionInvalidLifecycleInterval
	rejectionStructuralInvalidity
	rejectionProvedInvariantViolation
)

// rejection identifies one deterministic acceptance failure for an existing
// semantic subject/order coordinate supplied by the owning subsystem.
type rejection struct {
	Kind    rejectionKind
	Order   uint64
	Subject uint64
	Related uint64
}

func normalizeRejections(rejections []rejection) []rejection {
	result := append([]rejection(nil), rejections...)
	slices.SortStableFunc(result, func(left, right rejection) int {
		return cmp.Or(
			cmp.Compare(left.Order, right.Order),
			cmp.Compare(left.Kind, right.Kind),
			cmp.Compare(left.Subject, right.Subject),
			cmp.Compare(left.Related, right.Related),
		)
	})
	return result
}
