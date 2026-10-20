package cli

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"
)

type shortClusterMember struct {
	flag     flagGrammar
	spelling flagSpellingGrammar
	start    int
	end      int
	suffix   string
}

type shortClusterMatch struct {
	recognized     bool
	members        []shortClusterMember
	issueAt        int
	ambiguous      []FlagSpellingID
	candidateFlags []FlagID
}

func matchShortCluster(model grammarView, path []CommandID, dialect dialect, token string) shortClusterMatch {
	frontier := buildSelectedPathFlagFrontier(model, path)
	return matchShortClusterFrontier(model, frontier, dialect, token)
}

func matchShortClusterFlags(flags []flagGrammar, dialect dialect, token string) shortClusterMatch {
	if len(token) < 3 || token[0] != '-' || token[1] == '-' {
		return shortClusterMatch{}
	}
	mode := dialect.FlagNormalization()
	cursor := 1
	var members []shortClusterMember
	for cursor < len(token) {
		r, size := utf8.DecodeRuneInString(token[cursor:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		spellingText := "-" + string(r)
		normalized, ok := normalizeName(mode, spellingText)
		if !ok {
			match := shortClusterMatch{
				recognized: len(members) != 0,
				members:    members,
				issueAt:    cursor,
			}

			return match
		}
		type candidate struct {
			flag     flagGrammar
			spelling flagSpellingGrammar
		}
		var candidates []candidate
		for _, flag := range flags {
			for _, spelling := range flag.Spellings() {
				if spelling.kind != FlagSpellingShort {
					continue
				}
				name, valid := normalizeName(mode, spelling.text)
				if valid && name == normalized {
					candidates = append(candidates, candidate{
						flag:     flag,
						spelling: spelling,
					})
				}
			}
		}
		if len(candidates) == 0 {
			if len(members) == 0 {
				return shortClusterMatch{}
			}
			match := shortClusterMatch{
				recognized: true,
				members:    members,
				issueAt:    cursor,
			}

			return match
		}
		if len(candidates) > 1 {
			ids := make([]FlagSpellingID, len(candidates))
			flags := make([]FlagID, len(candidates))
			for i, candidate := range candidates {
				ids[i] = candidate.spelling.id
				flags[i] = candidate.flag.id
			}
			match := shortClusterMatch{
				recognized:     true,
				members:        members,
				issueAt:        cursor,
				ambiguous:      ids,
				candidateFlags: flags,
			}

			return match
		}
		matched := candidates[0]
		end := cursor + size
		member := shortClusterMember{
			flag:     matched.flag,
			spelling: matched.spelling,
			start:    cursor,
			end:      end,
		}
		if matched.spelling.mode != FlagValueModeNone {
			member.suffix = token[end:]
			members = append(members, member)
			match := shortClusterMatch{
				recognized: true,
				members:    members,
				issueAt:    -1,
			}

			return match
		}
		members = append(members, member)
		cursor = end
	}
	result := shortClusterMatch{
		recognized: len(members) > 1,
		members:    members,
		issueAt:    -1,
	}

	return result
}

func acceptShortCluster(operation *invocation, analysis *analysisState, args []string, result *grammarResult, command CommandID, model grammarView, path []CommandID, dialect dialect, index int, cluster shortClusterMatch, occurrences *map[FlagID]int, argumentsStarted bool) (int, bool) {
	if len(cluster.ambiguous) != 0 {
		grammarIssueValue := grammarIssue{
			Kind:       grammarIssueAmbiguousFlag,
			TokenIndex: index,
			Command:    command,
			Flags:      cluster.ambiguous,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue)
		return index, false
	}
	if cluster.issueAt >= 0 {
		grammarIssueValue8 := grammarIssue{
			Kind:       grammarIssueUnknownFlag,
			TokenIndex: index,
			Command:    command,
		}
		appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue8)
		return index, false
	}
	if len(cluster.members) == 0 {
		return index, false
	}
	physicalOrigin, _ := analysis.argumentOrigin(index)
	observations := make([]tokenObservation, 0, len(cluster.members))
	next := index + 1
	for memberIndex, member := range cluster.members {
		if kind := flagSpellingLifecycleIssue(
			analysis,
			member.flag.id,
			member.spelling.id,
		); kind != grammarIssueNone {
			grammarIssueValue2 := grammarIssue{
				Kind:       kind,
				TokenIndex: index,
				Command:    command,
				Flags:      []FlagSpellingID{member.spelling.id},
			}
			appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue2)
			return index, false
		}
		if !member.flag.Enabled() {
			grammarIssueValue3 := grammarIssue{
				Kind:       grammarIssueUnavailableFlag,
				TokenIndex: index,
				Command:    command,
				Flags:      []FlagSpellingID{member.spelling.id},
			}
			appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue3)
			return index, false
		}
		count := (*occurrences)[member.flag.id]
		maximum, bounded := member.flag.occurrences.Maximum()
		if bounded && count >= maximum {
			grammarIssueValue4 := grammarIssue{
				Kind:       grammarIssueTooManyFlagOccurrences,
				TokenIndex: index,
				Command:    command,
				Flags:      []FlagSpellingID{member.spelling.id},
			}
			appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue4)
			return index, false
		}
		occurrence := flagOccurrence{
			Occurrence:      count,
			Flag:            member.flag.id,
			Spelling:        member.spelling.id,
			Command:         command,
			Destination:     member.flag.destination,
			TokenIndex:      index,
			TerminatorIndex: -1,
			Match:           nameMatchKindExact,
		}
		mode := member.spelling.mode
		if mode != FlagValueModeNone {
			switch {
			case member.suffix != "":
				occurrence.Form = flagValueFormJoined
				if !member.spelling.Allows(flagValueFormJoined) ||
					!countAllows(member.flag.valuesPerOccurrence, 1) {
					grammarIssueValue5 := grammarIssue{
						Kind:       grammarIssueMissingFlagValue,
						TokenIndex: index,
						Command:    command,
						Flags:      []FlagSpellingID{member.spelling.id},
					}
					appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue5)
					return index, false
				}
				occurrence.Attached = member.suffix
			case member.spelling.Allows(flagValueFormSeparate):
				occurrence.Form = flagValueFormSeparate
				match2 := flagMatch{
					flag:     member.flag,
					spelling: member.spelling,
					form:     flagValueFormSeparate,
					match:    nameMatchKindExact,
				}
				planned, plannedNext, issue, issueIndex := planFlagOccurrence(
					args,
					index,
					command,
					model,
					path,
					dialect,
					match2,
					argumentsStarted,
				)
				if issue != grammarIssueNone {
					grammarIssueValue6 := grammarIssue{
						Kind:       issue,
						TokenIndex: issueIndex,
						Command:    command,
						Flags:      []FlagSpellingID{member.spelling.id},
					}
					appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue6)
					return index, false
				}
				occurrence.ValueIndices = append([]int(nil), planned.ValueIndices...)
				occurrence.TerminatorIndex = planned.TerminatorIndex
				next = plannedNext
			default:
				grammarIssueValue7 := grammarIssue{
					Kind:       grammarIssueMissingFlagValue,
					TokenIndex: index,
					Command:    command,
					Flags:      []FlagSpellingID{member.spelling.id},
				}
				appendGrammarIssue(result, operation.input.limits.Diagnostics, grammarIssueValue7)
				return index, false
			}
		}
		ownership := tokenOwnership{
			Role:         tokenRoleFlag,
			Command:      command,
			Flag:         member.flag.id,
			FlagSpelling: member.spelling.id,
			Destination:  member.flag.destination,
			Match:        nameMatchKindExact,
			Form:         occurrence.Form,
			Occurrence:   count,
		}
		spanRefValue := spanRef{
			Source: operation.ArgumentSource(),
			Entry:  index,
			Start:  member.start,
			End:    member.end,
		}
		origin := operation.AddDerivedOrigin(spanRefValue, []originRef{physicalOrigin}, operation.originSensitive(physicalOrigin.ID()))
		ownership.Origin = origin.ID()
		observation := tokenObservation{
			Text:          member.spelling.text,
			Origin:        origin.ID(),
			PhysicalIndex: index,
			Start:         member.start,
			End:           member.end,
			Ownership:     ownership,
		}
		observations = append(observations, observation)
		if *occurrences == nil {
			*occurrences = make(map[FlagID]int)
		}
		(*occurrences)[member.flag.id] = count + 1
		result.Occurrences = append(result.Occurrences, occurrence)
		if memberIndex == 0 {
			physical := ownership
			physical.Origin = result.Ownership[index].Origin
			result.Ownership[index] = physical
		}
	}
	if result.tokenOverrides == nil {
		result.tokenOverrides = make(map[int][]tokenObservation)
	}
	result.tokenOverrides[index] = observations
	last := cluster.members[len(cluster.members)-1]
	if last.spelling.mode != FlagValueModeNone && last.suffix == "" {
		occurrence := result.Occurrences[len(result.Occurrences)-1]
		for element, valueIndex := range occurrence.ValueIndices {
			ownership2 := tokenOwnership{
				Role:         tokenRoleFlagValue,
				Command:      command,
				Flag:         last.flag.id,
				FlagSpelling: last.spelling.id,
				Destination:  last.flag.destination,
				Form:         occurrence.Form,
				Occurrence:   occurrence.Occurrence,
				Element:      element,
			}
			claimToken(result, valueIndex, ownership2)
		}
		if occurrence.TerminatorIndex >= 0 {
			ownership3 := tokenOwnership{
				Role:         tokenRoleFlagTerminator,
				Command:      command,
				Flag:         last.flag.id,
				FlagSpelling: last.spelling.id,
				Destination:  last.flag.destination,
				Form:         occurrence.Form,
				Occurrence:   occurrence.Occurrence,
			}
			claimToken(result, occurrence.TerminatorIndex, ownership3)
		}
	}
	return next, true
}

type childMatch struct {
	found          bool
	ambiguous      bool
	enabled        bool
	match          nameMatchKind
	entry          commandEntryGrammar
	action         ActionEntryID
	actionKind     intrinsicAction
	entries        []CommandEntryID
	actions        []ActionEntryID
	targets        []CommandID
	lifecycle      LifecycleState
	lifecycleKnown bool
}

func commandLexicalNamespaceFor(model grammarView, command CommandID) (*commandLexicalNamespace, bool) {
	switch grammar := model.(type) {
	case analysisOperationGrammarScope:
		return grammar.commandLexicalNamespace(command), true
	case availabilityGrammar:
		return commandLexicalNamespaceFor(grammar.base, command)
	default:
		return nil, false
	}
}

func commandEntryEnabled(model grammarView, entry commandEntryGrammar) bool {
	enabled := entry.enabled
	for {
		grammar, ok := model.(availabilityGrammar)
		if !ok {
			return enabled
		}
		if available, exists := grammar.commands[entry.target]; exists && !available {
			enabled = false
		}
		model = grammar.base
	}
}

func lexicalPrefixIndices(members []commandLexicalMember, normalized string) []int {
	var indices []int
	for index, member := range members {
		if strings.HasPrefix(member.name, normalized) {
			indices = append(indices, index)
		}
	}
	return indices
}

func matchCachedCommandChild(model grammarView, namespace *commandLexicalNamespace, normalized string, dialect dialect) childMatch {
	indices := namespace.exactEntries[normalized]
	kind := nameMatchKindExact
	if len(indices) == 0 && dialect.CommandMatching() == NameMatchingUniquePrefix {
		kind = nameMatchKindUniquePrefix
		indices = lexicalPrefixIndices(namespace.entryMembers, normalized)
	}
	if len(indices) == 0 {
		return childMatch{}
	}
	if len(indices) > 1 {
		result := childMatch{
			found:     true,
			ambiguous: true,
			match:     kind,
		}
		for _, index := range indices {
			entry := namespace.entryMembers[index].entry
			result.entries = append(result.entries, entry.id)
			result.targets = append(result.targets, entry.target)
		}
		return result
	}
	member := namespace.entryMembers[indices[0]]
	entry := member.entry
	value := childMatch{
		found:          true,
		match:          kind,
		entry:          entry,
		entries:        []CommandEntryID{entry.id},
		targets:        []CommandID{entry.target},
		enabled:        commandEntryEnabled(model, entry),
		lifecycle:      member.lifecycle,
		lifecycleKnown: true,
	}

	return value
}

func matchCachedChild(model grammarView, namespace *commandLexicalNamespace, normalized string, dialect dialect) childMatch {
	indices := namespace.exact[normalized]
	kind := nameMatchKindExact
	if len(indices) == 0 && dialect.CommandMatching() == NameMatchingUniquePrefix {
		kind = nameMatchKindUniquePrefix
		indices = lexicalPrefixIndices(namespace.members, normalized)
	}
	if len(indices) == 0 {
		return childMatch{}
	}
	if len(indices) > 1 {
		result := childMatch{
			found:     true,
			ambiguous: true,
			match:     kind,
		}
		for _, index := range indices {
			member := namespace.members[index]
			if member.entry.id.present() {
				result.entries = append(result.entries, member.entry.id)
				result.targets = append(result.targets, member.entry.target)
				continue
			}
			result.actions = append(result.actions, member.action.id)
		}
		return result
	}
	member := namespace.members[indices[0]]
	result := childMatch{
		found:          true,
		match:          kind,
		lifecycle:      member.lifecycle,
		lifecycleKnown: true,
	}
	if member.entry.id.present() {
		result.entry = member.entry
		result.entries = []CommandEntryID{member.entry.id}
		result.targets = []CommandID{member.entry.target}
		result.enabled = commandEntryEnabled(model, member.entry)
		return result
	}
	result.action = member.action.id
	result.actionKind = member.action.action
	result.actions = []ActionEntryID{member.action.id}
	result.enabled = member.action.enabled
	return result
}

func matchCommandChild(model grammarView, command CommandID, dialect dialect, token string) childMatch {
	mode := dialect.CommandNormalization()
	normalized, ok := normalizeName(mode, token)
	if !ok {
		return childMatch{}
	}
	if namespace, cached := commandLexicalNamespaceFor(model, command); cached {
		return matchCachedCommandChild(model, namespace, normalized, dialect)
	}
	type candidate struct {
		name  string
		entry commandEntryGrammar
	}
	var exact, prefix []candidate
	for _, entry := range model.CommandEntries(command) {
		name, _ := normalizeName(mode, entry.name)
		candidate := candidate{
			name:  name,
			entry: entry,
		}
		if normalized == name {
			exact = append(exact, candidate)
		} else if dialect.CommandMatching() == NameMatchingUniquePrefix && strings.HasPrefix(name, normalized) {
			prefix = append(prefix, candidate)
		}
	}
	candidates := exact
	kind := nameMatchKindExact
	if len(candidates) == 0 {
		candidates, kind = prefix, nameMatchKindUniquePrefix
	}
	if len(candidates) == 0 {
		return childMatch{}
	}
	if len(candidates) > 1 {
		result := childMatch{
			found:     true,
			ambiguous: true,
			match:     kind,
		}
		for _, candidate := range candidates {
			result.entries = append(result.entries, candidate.entry.id)
			result.targets = append(result.targets, candidate.entry.target)
		}
		return result
	}
	selected := candidates[0].entry
	value := childMatch{
		found:   true,
		match:   kind,
		entry:   selected,
		entries: []CommandEntryID{selected.id},
		targets: []CommandID{selected.target},
		enabled: selected.enabled,
	}

	return value
}

func matchChild(model grammarView, command CommandID, dialect dialect, token string) childMatch {
	mode := dialect.CommandNormalization()
	normalized, ok := normalizeName(mode, token)
	if !ok {
		return childMatch{}
	}
	if namespace, cached := commandLexicalNamespaceFor(model, command); cached {
		return matchCachedChild(model, namespace, normalized, dialect)
	}
	type candidate struct {
		name   string
		entry  *commandEntryGrammar
		action *actionCommandGrammar
	}
	var exact, prefix []candidate
	entries := model.CommandEntries(command)
	for i := range entries {
		name, _ := normalizeName(mode, entries[i].name)
		candidate := candidate{
			name:  name,
			entry: &entries[i],
		}
		if normalized == name {
			exact = append(exact, candidate)
		} else if dialect.CommandMatching() == NameMatchingUniquePrefix && strings.HasPrefix(name, normalized) {
			prefix = append(prefix, candidate)
		}
	}
	actions := model.ActionCommands(command)
	for i := range actions {
		name, _ := normalizeName(mode, actions[i].name)
		candidate := candidate{
			name:   name,
			action: &actions[i],
		}
		if normalized == name {
			exact = append(exact, candidate)
		} else if dialect.CommandMatching() == NameMatchingUniquePrefix && strings.HasPrefix(name, normalized) {
			prefix = append(prefix, candidate)
		}
	}
	candidates := exact
	kind := nameMatchKindExact
	if len(candidates) == 0 {
		candidates, kind = prefix, nameMatchKindUniquePrefix
	}
	if len(candidates) == 0 {
		return childMatch{}
	}
	if len(candidates) > 1 {
		result := childMatch{
			found:     true,
			ambiguous: true,
			match:     kind,
		}
		for _, candidate := range candidates {
			if candidate.entry != nil {
				result.entries = append(result.entries, candidate.entry.id)
				result.targets = append(result.targets, candidate.entry.target)
			} else {
				result.actions = append(result.actions, candidate.action.id)
			}
		}
		return result
	}
	selected := candidates[0]
	result := childMatch{
		found: true,
		match: kind,
	}
	if selected.entry != nil {
		result.entry = *selected.entry
		result.entries = []CommandEntryID{selected.entry.id}
		result.targets = []CommandID{selected.entry.target}
		result.enabled = selected.entry.enabled
	} else {
		result.action = selected.action.id
		result.actionKind = selected.action.action
		result.actions = []ActionEntryID{selected.action.id}
		result.enabled = selected.action.enabled
	}
	return result
}

type flagMatch struct {
	found          bool
	ambiguous      bool
	enabled        bool
	match          nameMatchKind
	form           flagValueForm
	attached       string
	flag           flagGrammar
	spelling       flagSpellingGrammar
	action         semanticActionFlagGrammar
	candidates     []FlagSpellingID
	candidateFlags []FlagID
	actions        []ActionEntryID
}

func matchFlag(model grammarView, path []CommandID, dialect dialect, token string) flagMatch {
	frontier := buildSelectedPathFlagFrontier(model, path)
	return matchFlagSetsFrontier(model, frontier, dialect, token)
}

func matchSelectedPathFlag(model grammarView, path []CommandID, dialect dialect, token string) flagMatch {
	flagSets := make([][]flagGrammar, len(path))
	flagCount := 0
	for index, command := range path {
		flagSets[index] = model.Flags(command)
		flagCount += len(flagSets[index])
	}
	flags := make([]flagGrammar, 0, flagCount)
	for _, set := range flagSets {
		flags = append(flags, set...)
	}
	return matchFlags(flags, dialect, token)
}

func matchFlags(flags []flagGrammar, dialect dialect, token string) flagMatch {
	type candidate struct {
		flag     flagGrammar
		spelling flagSpellingGrammar
		match    nameMatchKind
		form     flagValueForm
		attached string
	}
	var exactAttached, exactPlain, prefixPlain []candidate
	mode := dialect.FlagNormalization()
	normalizedToken, validNormalization := normalizeName(mode, token)
	// Accepted normalization modes preserve byte length, so token byte offsets
	// remain valid in normalizedToken for attached-value prefix equality.
	for _, flag := range flags {
		for _, spelling := range flag.Spellings() {
			text := spelling.text
			normalizedSpelling, _ := normalizeName(mode, text)
			if spelling.Allows(flagValueFormAssigned) && len(token) > len(text) && token[len(text)] == '=' {
				var normalizedBase string
				if validNormalization {
					normalizedBase = normalizedToken[:len(text)]
				} else {
					normalizedBase, _ = normalizeName(mode, token[:len(text)])
				}
				if normalizedBase == normalizedSpelling {
					candidateValue := candidate{
						flag:     flag,
						spelling: spelling,
						match:    nameMatchKindExact,
						form:     flagValueFormAssigned,
						attached: token[len(text)+1:],
					}
					exactAttached = append(exactAttached, candidateValue)
				}
			}
			if spelling.Allows(flagValueFormJoined) && len(token) > len(text) {
				var normalizedBase string
				if validNormalization {
					normalizedBase = normalizedToken[:len(text)]
				} else {
					normalizedBase, _ = normalizeName(mode, token[:len(text)])
				}
				if normalizedBase == normalizedSpelling {
					candidateValue2 := candidate{
						flag:     flag,
						spelling: spelling,
						match:    nameMatchKindExact,
						form:     flagValueFormJoined,
						attached: token[len(text):],
					}
					exactAttached = append(exactAttached, candidateValue2)
				}
			}
			if normalizedToken == normalizedSpelling {
				if spelling.mode == FlagValueModeNone ||
					spelling.Allows(flagValueFormSeparate) {
					candidateValue3 := candidate{
						flag:     flag,
						spelling: spelling,
						match:    nameMatchKindExact,
						form:     flagValueFormSeparate,
					}
					exactPlain = append(exactPlain, candidateValue3)
				}
			} else if spelling.kind == FlagSpellingLong && dialect.FlagMatching() == NameMatchingUniquePrefix && strings.HasPrefix(normalizedSpelling, normalizedToken) {
				if spelling.mode == FlagValueModeNone ||
					spelling.Allows(flagValueFormSeparate) {
					candidateValue4 := candidate{
						flag:     flag,
						spelling: spelling,
						match:    nameMatchKindUniquePrefix,
						form:     flagValueFormSeparate,
					}
					prefixPlain = append(prefixPlain, candidateValue4)
				}
			}
		}
	}
	candidates := exactAttached
	if len(candidates) == 0 {
		candidates = exactPlain
	}
	if len(candidates) == 0 {
		candidates = prefixPlain
	}
	if len(candidates) == 0 {
		return flagMatch{}
	}
	if len(candidates) > 1 {
		result := flagMatch{
			found:     true,
			ambiguous: true,
		}
		for _, candidate := range candidates {
			result.candidates = append(result.candidates, candidate.spelling.id)
			result.candidateFlags = append(result.candidateFlags, candidate.flag.id)
		}
		return result
	}
	selected := candidates[0]
	value := flagMatch{
		found:          true,
		enabled:        selected.flag.Enabled(),
		match:          selected.match,
		form:           selected.form,
		attached:       selected.attached,
		flag:           selected.flag,
		spelling:       selected.spelling,
		candidates:     []FlagSpellingID{selected.spelling.id},
		candidateFlags: []FlagID{selected.flag.id},
	}

	return value
}

func matchActionFlags(actions []semanticActionFlagGrammar, dialect dialect, token string) flagMatch {
	mode := dialect.FlagNormalization()
	normalizedToken, _ := normalizeName(mode, token)
	var exact, prefix []semanticActionFlagGrammar
	for _, action := range actions {
		name, _ := normalizeName(mode, action.text)
		if normalizedToken == name {
			exact = append(exact, action)
		} else if action.kind == FlagSpellingLong && dialect.FlagMatching() == NameMatchingUniquePrefix && strings.HasPrefix(name, normalizedToken) {
			prefix = append(prefix, action)
		}
	}
	candidates := exact
	match := nameMatchKindExact
	if len(candidates) == 0 {
		candidates, match = prefix, nameMatchKindUniquePrefix
	}
	if len(candidates) == 0 {
		return flagMatch{}
	}
	if len(candidates) > 1 {
		result := flagMatch{
			found:     true,
			ambiguous: true,
		}
		for _, candidate := range candidates {
			result.actions = append(result.actions, candidate.id)
		}
		return result
	}
	selected := candidates[0]
	value := flagMatch{
		found:   true,
		enabled: selected.enabled,
		match:   match,
		action:  selected,
		actions: []ActionEntryID{selected.id},
	}

	return value
}

type flagLexicalCandidate struct {
	flag     flagGrammar
	spelling flagSpellingGrammar
	name     string
}

type flagAttachedCandidate struct {
	candidate int
	form      flagValueForm
	order     int
}

type flagMatchIndex struct {
	flags      []flagGrammar
	candidates []flagLexicalCandidate
	exactPlain map[string][]int
	attached   map[string][]flagAttachedCandidate
	shortExact map[string][]int
}

type actionFlagLexicalCandidate struct {
	action semanticActionFlagGrammar
	name   string
}

type actionFlagMatchIndex struct {
	actions    []semanticActionFlagGrammar
	candidates []actionFlagLexicalCandidate
	exact      map[string][]int
}

type flagFrontierSnapshot struct {
	active        []flagGrammar
	actions       []semanticActionFlagGrammar
	selected      []flagGrammar
	activeIndex   flagMatchIndex
	actionIndex   actionFlagMatchIndex
	selectedIndex flagMatchIndex
}

type flagFrontierCacheKey struct {
	lexical lexicalNamespaceKey
	parent  *flagFrontierSnapshot
}

func indexFlagMatches(flags []flagGrammar, dialect dialect) flagMatchIndex {
	index := flagMatchIndex{
		flags:      flags,
		exactPlain: make(map[string][]int),
		attached:   make(map[string][]flagAttachedCandidate),
		shortExact: make(map[string][]int),
	}
	mode := dialect.FlagNormalization()
	attachedOrder := 0
	for _, flag := range flags {
		for _, spelling := range flag.Spellings() {
			name, valid := normalizeName(mode, spelling.text)
			if !valid {
				continue
			}
			candidateIndex := len(index.candidates)
			flagLexicalCandidateValue := flagLexicalCandidate{
				flag:     flag,
				spelling: spelling,
				name:     name,
			}
			index.candidates = append(index.candidates, flagLexicalCandidateValue)
			if spelling.mode == FlagValueModeNone || spelling.Allows(flagValueFormSeparate) {
				index.exactPlain[name] = append(index.exactPlain[name], candidateIndex)
			}
			if spelling.kind == FlagSpellingShort {
				index.shortExact[name] = append(index.shortExact[name], candidateIndex)
			}
			if spelling.Allows(flagValueFormAssigned) {
				attached := flagAttachedCandidate{
					candidate: candidateIndex,
					form:      flagValueFormAssigned,
					order:     attachedOrder,
				}
				index.attached[name] = append(index.attached[name], attached)
				attachedOrder++
			}
			if spelling.Allows(flagValueFormJoined) {
				attached := flagAttachedCandidate{
					candidate: candidateIndex,
					form:      flagValueFormJoined,
					order:     attachedOrder,
				}
				index.attached[name] = append(index.attached[name], attached)
				attachedOrder++
			}
		}
	}
	return index
}

func indexActionFlagMatches(actions []semanticActionFlagGrammar, dialect dialect) actionFlagMatchIndex {
	index := actionFlagMatchIndex{
		actions: actions,
		exact:   make(map[string][]int),
	}
	mode := dialect.FlagNormalization()
	for _, action := range actions {
		name, valid := normalizeName(mode, action.text)
		if !valid {
			continue
		}
		candidateIndex := len(index.candidates)
		index.candidates = append(index.candidates, actionFlagLexicalCandidate{
			action: action,
			name:   name,
		})
		index.exact[name] = append(index.exact[name], candidateIndex)
	}
	return index
}

func flagGrammarEnabled(model grammarView, flag flagGrammar) bool {
	enabled := flag.Enabled()
	for {
		grammar, ok := model.(availabilityGrammar)
		if !ok {
			return enabled
		}
		if available, exists := grammar.flags[flag.id]; exists && !available {
			enabled = false
		}
		model = grammar.base
	}
}

func matchFlagsIndexed(model grammarView, index flagMatchIndex, dialect dialect, token string) flagMatch {
	mode := dialect.FlagNormalization()
	normalizedToken, validNormalization := normalizeName(mode, token)
	if !validNormalization {
		result := matchFlags(index.flags, dialect, token)
		if result.found && !result.ambiguous && result.flag.id.present() {
			result.enabled = flagGrammarEnabled(model, result.flag)
		}
		return result
	}
	type attachedMatch struct {
		spec     flagAttachedCandidate
		attached string
	}
	var attached []attachedMatch
	for end := 1; end < len(normalizedToken); end++ {
		for _, spec := range index.attached[normalizedToken[:end]] {
			switch spec.form {
			case flagValueFormAssigned:
				if token[end] != '=' {
					continue
				}
				attached = append(attached, attachedMatch{
					spec:     spec,
					attached: token[end+1:],
				})
			case flagValueFormJoined:
				attached = append(attached, attachedMatch{
					spec:     spec,
					attached: token[end:],
				})
			case flagValueFormInvalid, flagValueFormSeparate:
				continue
			}
		}
	}
	if len(attached) > 1 {
		slices.SortFunc(attached, func(left, right attachedMatch) int { return cmp.Compare(left.spec.order, right.spec.order) })
	}
	if len(attached) != 0 {
		if len(attached) > 1 {
			result := flagMatch{
				found:     true,
				ambiguous: true,
			}
			for _, match := range attached {
				candidate := index.candidates[match.spec.candidate]
				result.candidates = append(result.candidates, candidate.spelling.id)
				result.candidateFlags = append(result.candidateFlags, candidate.flag.id)
			}
			return result
		}
		match := attached[0]
		candidate := index.candidates[match.spec.candidate]
		value := flagMatch{
			found:          true,
			enabled:        flagGrammarEnabled(model, candidate.flag),
			match:          nameMatchKindExact,
			form:           match.spec.form,
			attached:       match.attached,
			flag:           candidate.flag,
			spelling:       candidate.spelling,
			candidates:     []FlagSpellingID{candidate.spelling.id},
			candidateFlags: []FlagID{candidate.flag.id},
		}

		return value
	}
	indices := index.exactPlain[normalizedToken]
	kind := nameMatchKindExact
	if len(indices) == 0 && dialect.FlagMatching() == NameMatchingUniquePrefix {
		kind = nameMatchKindUniquePrefix
		for candidateIndex, candidate := range index.candidates {
			if candidate.spelling.kind == FlagSpellingLong &&
				strings.HasPrefix(candidate.name, normalizedToken) &&
				(candidate.spelling.mode == FlagValueModeNone || candidate.spelling.Allows(flagValueFormSeparate)) {
				indices = append(indices, candidateIndex)
			}
		}
	}
	if len(indices) == 0 {
		return flagMatch{}
	}
	if len(indices) > 1 {
		result := flagMatch{
			found:     true,
			ambiguous: true,
		}
		for _, candidateIndex := range indices {
			candidate := index.candidates[candidateIndex]
			result.candidates = append(result.candidates, candidate.spelling.id)
			result.candidateFlags = append(result.candidateFlags, candidate.flag.id)
		}
		return result
	}
	candidate := index.candidates[indices[0]]
	value := flagMatch{
		found:          true,
		enabled:        flagGrammarEnabled(model, candidate.flag),
		match:          kind,
		form:           flagValueFormSeparate,
		flag:           candidate.flag,
		spelling:       candidate.spelling,
		candidates:     []FlagSpellingID{candidate.spelling.id},
		candidateFlags: []FlagID{candidate.flag.id},
	}

	return value
}

func matchActionFlagsIndexed(index actionFlagMatchIndex, dialect dialect, token string) flagMatch {
	mode := dialect.FlagNormalization()
	normalizedToken, valid := normalizeName(mode, token)
	if !valid {
		return matchActionFlags(index.actions, dialect, token)
	}
	indices := index.exact[normalizedToken]
	kind := nameMatchKindExact
	if len(indices) == 0 && dialect.FlagMatching() == NameMatchingUniquePrefix {
		kind = nameMatchKindUniquePrefix
		for candidateIndex, candidate := range index.candidates {
			if candidate.action.kind == FlagSpellingLong &&
				strings.HasPrefix(candidate.name, normalizedToken) {
				indices = append(indices, candidateIndex)
			}
		}
	}
	if len(indices) == 0 {
		return flagMatch{}
	}
	if len(indices) > 1 {
		result := flagMatch{
			found:     true,
			ambiguous: true,
		}
		for _, candidateIndex := range indices {
			result.actions = append(result.actions, index.candidates[candidateIndex].action.id)
		}
		return result
	}
	action := index.candidates[indices[0]].action
	value := flagMatch{
		found:   true,
		enabled: action.enabled,
		match:   kind,
		action:  action,
		actions: []ActionEntryID{action.id},
	}

	return value
}

func matchShortClusterIndexed(model grammarView, index flagMatchIndex, dialect dialect, token string) shortClusterMatch {
	if len(token) < 3 || token[0] != '-' || token[1] == '-' {
		return shortClusterMatch{}
	}
	mode := dialect.FlagNormalization()
	cursor := 1
	var members []shortClusterMember
	for cursor < len(token) {
		r, size := utf8.DecodeRuneInString(token[cursor:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		spellingText := "-" + string(r)
		normalized, ok := normalizeName(mode, spellingText)
		if !ok {
			match := shortClusterMatch{
				recognized: len(members) != 0,
				members:    members,
				issueAt:    cursor,
			}

			return match
		}
		indices := index.shortExact[normalized]
		if len(indices) == 0 {
			if len(members) == 0 {
				return shortClusterMatch{}
			}
			match := shortClusterMatch{
				recognized: true,
				members:    members,
				issueAt:    cursor,
			}

			return match
		}
		if len(indices) > 1 {
			result := shortClusterMatch{
				recognized: true,
				members:    members,
				issueAt:    cursor,
			}
			for _, candidateIndex := range indices {
				candidate := index.candidates[candidateIndex]
				result.ambiguous = append(result.ambiguous, candidate.spelling.id)
				result.candidateFlags = append(result.candidateFlags, candidate.flag.id)
			}
			return result
		}
		candidate := index.candidates[indices[0]]
		flag := candidate.flag
		if !flagGrammarEnabled(model, flag) {
			flag.disabled = true
		}
		end := cursor + size
		member := shortClusterMember{
			flag:     flag,
			spelling: candidate.spelling,
			start:    cursor,
			end:      end,
		}
		if candidate.spelling.mode != FlagValueModeNone {
			member.suffix = token[end:]
			members = append(members, member)
			match := shortClusterMatch{
				recognized: true,
				members:    members,
				issueAt:    -1,
			}

			return match
		}
		members = append(members, member)
		cursor = end
	}
	match := shortClusterMatch{
		recognized: len(members) > 1,
		members:    members,
		issueAt:    -1,
	}

	return match
}

func operationGrammarScopeFor(model grammarView) (analysisOperationGrammarScope, bool) {
	for {
		switch grammar := model.(type) {
		case analysisOperationGrammarScope:
			return grammar, true
		case availabilityGrammar:
			model = grammar.base
		default:
			return analysisOperationGrammarScope{}, false
		}
	}
}

func (analysis *analysisState) cachedFlagFrontier(key flagFrontierCacheKey, build func() *flagFrontierSnapshot) *flagFrontierSnapshot {
	analysis.grammar.mu.RLock()
	cached := analysis.grammar.flagFrontiers[key]
	analysis.grammar.mu.RUnlock()
	if cached != nil {
		return cached
	}
	built := build()
	analysis.grammar.mu.Lock()
	if analysis.grammar.flagFrontiers == nil {
		analysis.grammar.flagFrontiers = make(map[flagFrontierCacheKey]*flagFrontierSnapshot)
	}
	if cached = analysis.grammar.flagFrontiers[key]; cached == nil {
		analysis.grammar.flagFrontiers[key] = built
		cached = built
	}
	analysis.grammar.mu.Unlock()
	return cached
}

func inheritedActionFlagPlacement(scope analysisOperationGrammarScope, action semanticActionFlagGrammar) FlagPlacement {
	if !action.defaultPlacement {
		return action.placement
	}
	dialect, ok := scope.Dialect(action.parent)
	if !ok {
		return action.placement
	}
	return dialect.DefaultFlagPlacement()
}

func buildFlagFrontierSnapshot(scope analysisOperationGrammarScope, parent *flagFrontierSnapshot, command CommandID) *flagFrontierSnapshot {
	namespace := scope.flagLexicalNamespace(command)
	snapshot := &flagFrontierSnapshot{}
	if parent == nil {
		snapshot.active = namespace.flags
		snapshot.selected = namespace.flags
		snapshot.actions = namespace.actions
	} else {
		snapshot.selected = make([]flagGrammar, 0, len(parent.selected)+len(namespace.flags))
		snapshot.selected = append(snapshot.selected, parent.selected...)
		snapshot.selected = append(snapshot.selected, namespace.flags...)
		snapshot.active = make([]flagGrammar, 0, len(parent.active)+len(namespace.flags))
		for _, flag := range parent.active {
			if flag.placement == FlagPlacementSubtree {
				snapshot.active = append(snapshot.active, flag)
			}
		}
		snapshot.active = append(snapshot.active, namespace.flags...)
		snapshot.actions = make([]semanticActionFlagGrammar, 0, len(parent.actions)+len(namespace.actions))
		for _, action := range parent.actions {
			if inheritedActionFlagPlacement(scope, action) == FlagPlacementSubtree {
				snapshot.actions = append(snapshot.actions, action)
			}
		}
		snapshot.actions = append(snapshot.actions, namespace.actions...)
	}
	if dialect, ok := scope.Dialect(command); ok {
		snapshot.activeIndex = indexFlagMatches(snapshot.active, dialect)
		snapshot.selectedIndex = indexFlagMatches(snapshot.selected, dialect)
		snapshot.actionIndex = indexActionFlagMatches(snapshot.actions, dialect)
	}
	return snapshot
}

func (scope analysisOperationGrammarScope) cachedFlagFrontier(parent *flagFrontierSnapshot, command CommandID) *flagFrontierSnapshot {
	lexical, cacheable := scope.lexicalNamespaceKey(command)
	build := func() *flagFrontierSnapshot {
		return buildFlagFrontierSnapshot(scope, parent, command)
	}
	if !cacheable {
		return build()
	}
	return scope.source.analysis.cachedFlagFrontier(flagFrontierCacheKey{
		lexical: lexical,
		parent:  parent,
	}, build)
}

type selectedPathFlagFrontier struct {
	active   []flagGrammar
	actions  []semanticActionFlagGrammar
	selected []flagGrammar
	snapshot *flagFrontierSnapshot
}

func selectedPathFlagFrontierFromSnapshot(snapshot *flagFrontierSnapshot) selectedPathFlagFrontier {
	if snapshot == nil {
		return selectedPathFlagFrontier{}
	}
	result := selectedPathFlagFrontier{
		active:   snapshot.active,
		actions:  snapshot.actions,
		selected: snapshot.selected,
		snapshot: snapshot,
	}

	return result
}

func buildSelectedPathFlagFrontier(model grammarView, path []CommandID) selectedPathFlagFrontier {
	if scope, ok := operationGrammarScopeFor(model); ok && len(path) != 0 {
		var snapshot *flagFrontierSnapshot
		for _, command := range path {
			snapshot = scope.cachedFlagFrontier(snapshot, command)
		}
		return selectedPathFlagFrontierFromSnapshot(snapshot)
	}
	var frontier selectedPathFlagFrontier
	for index, command := range path {
		last := index == len(path)-1
		flags := model.Flags(command)
		frontier.selected = append(frontier.selected, flags...)
		for _, flag := range flags {
			if last || flag.placement == FlagPlacementSubtree {
				frontier.active = append(frontier.active, flag)
			}
		}
		dialect, hasDialect := model.Dialect(command)
		for _, action := range model.ActionFlags(command) {
			placement := action.placement
			if action.defaultPlacement && hasDialect {
				placement = dialect.DefaultFlagPlacement()
			}
			if last || placement == FlagPlacementSubtree {
				frontier.actions = append(frontier.actions, action)
			}
		}
	}
	return frontier
}

func (frontier *selectedPathFlagFrontier) extend(model grammarView, command CommandID) {
	if scope, ok := operationGrammarScopeFor(model); ok && frontier.snapshot != nil {
		*frontier = selectedPathFlagFrontierFromSnapshot(scope.cachedFlagFrontier(frontier.snapshot, command))
		return
	}
	keptFlags := frontier.active[:0]
	for _, flag := range frontier.active {
		if flag.placement == FlagPlacementSubtree {
			keptFlags = append(keptFlags, flag)
		}
	}
	frontier.active = keptFlags
	flags := model.Flags(command)
	frontier.active = append(frontier.active, flags...)
	frontier.selected = append(frontier.selected, flags...)

	keptActions := frontier.actions[:0]
	for _, action := range frontier.actions {
		placement := action.placement
		if action.defaultPlacement {
			if dialect, ok := model.Dialect(action.parent); ok {
				placement = dialect.DefaultFlagPlacement()
			}
		}
		if placement == FlagPlacementSubtree {
			keptActions = append(keptActions, action)
		}
	}
	frontier.actions = keptActions
	frontier.actions = append(frontier.actions, model.ActionFlags(command)...)
}

func (frontier *selectedPathFlagFrontier) clone() selectedPathFlagFrontier {
	if frontier.snapshot != nil {
		return *frontier
	}
	result := selectedPathFlagFrontier{
		active:   append([]flagGrammar(nil), frontier.active...),
		actions:  append([]semanticActionFlagGrammar(nil), frontier.actions...),
		selected: append([]flagGrammar(nil), frontier.selected...),
	}

	return result
}

func matchFlagsFrontier(model grammarView, frontier selectedPathFlagFrontier, dialect dialect, token string) flagMatch {
	if frontier.snapshot != nil {
		return matchFlagsIndexed(model, frontier.snapshot.activeIndex, dialect, token)
	}
	return matchFlags(frontier.active, dialect, token)
}

func matchSelectedFlagsFrontier(model grammarView, frontier selectedPathFlagFrontier, dialect dialect, token string) flagMatch {
	if frontier.snapshot != nil {
		return matchFlagsIndexed(model, frontier.snapshot.selectedIndex, dialect, token)
	}
	return matchFlags(frontier.selected, dialect, token)
}

func matchActionFlagsFrontier(frontier selectedPathFlagFrontier, dialect dialect, token string) flagMatch {
	if frontier.snapshot != nil {
		return matchActionFlagsIndexed(frontier.snapshot.actionIndex, dialect, token)
	}
	return matchActionFlags(frontier.actions, dialect, token)
}

func matchFlagSetsFrontier(model grammarView, frontier selectedPathFlagFrontier, dialect dialect, token string) flagMatch {
	regular := matchFlagsFrontier(model, frontier, dialect, token)
	action := matchActionFlagsFrontier(frontier, dialect, token)
	if !regular.found {
		return action
	}
	if !action.found {
		return regular
	}
	match := flagMatch{
		found:          true,
		ambiguous:      true,
		candidates:     append(append([]FlagSpellingID(nil), regular.candidates...), action.candidates...),
		candidateFlags: append([]FlagID(nil), regular.candidateFlags...),
		actions:        append(append([]ActionEntryID(nil), regular.actions...), action.actions...),
	}

	return match
}

func matchShortClusterFrontier(model grammarView, frontier selectedPathFlagFrontier, dialect dialect, token string) shortClusterMatch {
	if frontier.snapshot != nil {
		return matchShortClusterIndexed(model, frontier.snapshot.activeIndex, dialect, token)
	}
	return matchShortClusterFlags(frontier.active, dialect, token)
}

func activeActionFlags(model grammarView, path []CommandID) []semanticActionFlagGrammar {
	var result []semanticActionFlagGrammar
	for index, command := range path {
		last := index == len(path)-1
		dialect, hasDialect := model.Dialect(command)
		for _, action := range model.ActionFlags(command) {
			placement := action.placement
			if action.defaultPlacement && hasDialect {
				placement = dialect.DefaultFlagPlacement()
			}
			if last || placement == FlagPlacementSubtree {
				result = append(result, action)
			}
		}
	}
	return result
}

func activeFlags(model grammarView, path []CommandID) []flagGrammar {
	var result []flagGrammar
	for index, command := range path {
		last := index == len(path)-1
		for _, flag := range model.Flags(command) {
			if last || flag.placement == FlagPlacementSubtree {
				result = append(result, flag)
			}
		}
	}
	return result
}

func planFlagOccurrence(args []string, index int, command CommandID, model grammarView, path []CommandID, dialect dialect, match flagMatch, argumentsStarted bool) (flagOccurrence, int, grammarIssueKind, int) {
	occurrence := flagOccurrence{
		Flag:            match.flag.id,
		Spelling:        match.spelling.id,
		Command:         command,
		Destination:     match.flag.destination,
		TokenIndex:      index,
		TerminatorIndex: -1,
		Attached:        match.attached,
		Match:           match.match,
		Form:            match.form,
	}
	mode := match.spelling.mode
	if mode == FlagValueModeNone {
		return occurrence, index + 1, grammarIssueNone, -1
	}
	if match.form == flagValueFormAssigned || match.form == flagValueFormJoined {
		if !countAllows(match.flag.valuesPerOccurrence, 1) {
			return occurrence, index + 1, grammarIssueMissingFlagValue, index
		}
		return occurrence, index + 1, grammarIssueNone, -1
	}
	count := match.flag.valuesPerOccurrence
	minimum := count.Minimum()
	maximum, bounded := count.Maximum()
	if !bounded {
		maximum = len(args) - index - 1
	}
	if maximum > len(args)-index-1 {
		maximum = len(args) - index - 1
	}
	if mode == FlagValueModeOptional && minimum == 0 {
		if index+1 >= len(args) {
			return occurrence, index + 1, grammarIssueNone, -1
		}
		if len(model.ArgumentForms(command)) != 0 || argumentsStarted {
			return occurrence, index + 1, grammarIssueAmbiguousOwnership, index
		}
	}
	terminator, hasTerminator := match.flag.Terminator()
	consumed := 0
	for consumed < maximum {
		next := index + 1 + consumed
		if next >= len(args) {
			break
		}
		token := args[next]
		if hasTerminator && token == terminator {
			if consumed < minimum {
				occurrence = occurrenceWithValues(occurrence, index, consumed)
				occurrence.TerminatorIndex = next
				return occurrence, next + 1, grammarIssuePrematureFlagTerminator, next
			}
			occurrence = occurrenceWithValues(occurrence, index, consumed)
			occurrence.TerminatorIndex = next
			return occurrence, next + 1, grammarIssueNone, -1
		}
		if lexicalBoundary(model, path, dialect, token) {
			break
		}
		if isHyphenLeadingToken(token) && !hyphenValueAllowed(match.flag.hyphenValues, token) {
			break
		}
		consumed++
	}
	if consumed < minimum {
		occurrence = occurrenceWithValues(occurrence, index, consumed)
		return occurrence, index + 1 + consumed, grammarIssueMissingFlagValue, index
	}
	if !bounded && !hasTerminator && len(model.ArgumentForms(command)) != 0 && consumed > minimum {
		return occurrence, index + 1, grammarIssueAmbiguousOwnership, index
	}
	return occurrenceWithValues(occurrence, index, consumed), index + 1 + consumed, grammarIssueNone, -1
}

func recoverableFlagOccurrenceIssue(issue grammarIssueKind, next int, args []string, model grammarView, path []CommandID, dialect dialect) bool {
	switch issue {
	case grammarIssuePrematureFlagTerminator:
		return true
	case grammarIssueMissingFlagValue:
		if next >= len(args) {
			return true
		}
		token := args[next]
		if isEndOfOptions(dialect, token) {
			return true
		}
		match := matchFlag(model, path, dialect, token)
		return match.found && !match.ambiguous
	default:
		return false
	}
}

func claimRejectedFlagOccurrence(result *grammarResult, index int, command CommandID, match flagMatch, occurrence flagOccurrence, count int) {
	ownership := tokenOwnership{
		Role:         tokenRoleFlag,
		Command:      command,
		Flag:         match.flag.id,
		FlagSpelling: match.spelling.id,
		Destination:  match.flag.destination,
		Match:        match.match,
		Occurrence:   count,
	}
	claimToken(result, index, ownership)
	for element, valueIndex := range occurrence.ValueIndices {
		ownership2 := tokenOwnership{
			Role:         tokenRoleFlagValue,
			Command:      command,
			Flag:         match.flag.id,
			FlagSpelling: match.spelling.id,
			Destination:  match.flag.destination,
			Occurrence:   count,
			Element:      element,
		}
		claimToken(result, valueIndex, ownership2)
	}
	if occurrence.TerminatorIndex >= 0 {
		ownership3 := tokenOwnership{
			Role:         tokenRoleFlagTerminator,
			Command:      command,
			Flag:         match.flag.id,
			FlagSpelling: match.spelling.id,
			Destination:  match.flag.destination,
			Occurrence:   count,
		}
		claimToken(result, occurrence.TerminatorIndex, ownership3)
	}
}

func occurrenceWithValues(occurrence flagOccurrence, flagIndex, consumed int) flagOccurrence {
	for offset := range consumed {
		occurrence.ValueIndices = append(occurrence.ValueIndices, flagIndex+1+offset)
	}
	return occurrence
}

func lexicalBoundary(model grammarView, path []CommandID, dialect dialect, token string) bool {
	if isEndOfOptions(dialect, token) {
		return true
	}
	return matchFlag(model, path, dialect, token).found
}

func positionalCanOwnHyphenToken(forms []argumentFormGrammar, token string) bool {
	for _, form := range forms {
		if !form.enabled {
			continue
		}
		terms := form.Terms()
		if len(terms) == 0 {
			continue
		}
		term := terms[0]
		switch term.kind {
		case ArgumentTermLiteral:
			if term.literal == token {
				return true
			}
		case ArgumentTermRemainder, ArgumentTermPassThrough:
			return true
		case ArgumentTermValue, ArgumentTermTokens:
			if hyphenValueAllowed(term.hyphenValues, token) {
				return true
			}
		case ArgumentTermUnknown, ArgumentTermSequence:
		}
	}
	return false
}

func isHyphenLeadingToken(token string) bool {
	return len(token) > 1 && token[0] == '-'
}

func isUnknownHyphenFlagToken(token string) bool {
	return isHyphenLeadingToken(token)
}

func isUnknownAssignedFlagToken(token string) bool {
	separator := strings.IndexByte(token, '=')
	return separator > 1
}

func isNegativeNumberToken(token string) bool {
	if len(token) < 2 || token[0] != '-' {
		return false
	}
	for _, char := range token[1:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func hyphenValueAllowed(mode HyphenValueMode, token string) bool {
	switch mode {
	case HyphenValueNegativeNumbers:
		return isNegativeNumberToken(token)
	case HyphenValueAllow:
		return true
	default:
		return false
	}
}

func countAllows(count semanticCountRange, value int) bool {
	if value < count.Minimum() {
		return false
	}
	if maximum, bounded := count.Maximum(); bounded && value > maximum {
		return false
	}
	return true
}
