package cli

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

type authoringLifecycleTarget interface {
	authoringLifecycleTarget()
}

type authoringEnvironmentLifecycleTarget struct {
	reference valueDirectiveRef
}

type authoringArgumentFormLifecycleTarget struct {
	reference argumentFormDraftRef
}

type authoringArgumentTermLifecycleTarget struct {
	reference argumentTermDraftRef
}

func (authoringEnvironmentLifecycleTarget) authoringLifecycleTarget() {}

func (authoringArgumentFormLifecycleTarget) authoringLifecycleTarget() {}

func (authoringArgumentTermLifecycleTarget) authoringLifecycleTarget() {}

func stageStaticLifecycles(input declarationPlan, source *declarationDraft, draft *declarationCandidate, state *staticAcceptanceState) error {
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordLifecycleDirective {
			continue
		}
		directive := source.lifecycleDirectives.entries[record.familyIndex].value
		position := input.positionFor(index)
		lifecycle := directive.lifecycle
		targetIndex, indexed := input.lifecycleTargetIndex(directive.target)
		if !indexed {
			return errStaticDeclarationRejected
		}
		switch directive.target.(type) {
		case authoringEnvironmentLifecycleTarget:
			slot := state.mustSlot(targetIndex)
			if !slot.environment.ID().present() {
				panic("cli: lifecycle target environment identity is unavailable")
			}
			draft.EnvironmentInputLifecycle(slot.environment, lifecycle, position)
		case authoringArgumentFormLifecycleTarget:
			slot := state.mustSlot(targetIndex)
			if !slot.argumentForm.ID().present() {
				panic("cli: lifecycle target argument-form identity is unavailable")
			}
			draft.ArgumentFormLifecycle(slot.argumentForm, lifecycle, position)
		case authoringArgumentTermLifecycleTarget:
			slot := state.mustSlot(targetIndex)
			if !slot.argumentTerm.ID().present() {
				panic("cli: lifecycle target argument-term identity is unavailable")
			}
			draft.ArgumentTermLifecycle(slot.argumentTerm, lifecycle, position)
		default:
			return errStaticDeclarationRejected
		}
	}
	return nil
}

// validateLifecycleLexicalCollisions joins declaration lifecycle authority to
// the shared normalization/collision proof for any static declaration transaction.
// Common declarations are checked in every concrete profile branch because a
// profile is a namespace partition, while availability is deliberately not.
func validateLifecycleLexicalCollisions(scheme ReleaseScheme, input declarationRecordInput, model grammarModel) []rejection {
	ranks, rankRejections := lifecycleBoundaryRanks(scheme, input)
	if len(rankRejections) != 0 {
		return rankRejections
	}
	commandRecords := make(map[CommandEntryID]commandEntryRecord, len(input.CommandEntries))
	canonicalByTarget := make(map[CommandID]ReleaseLifecycle)
	grammarEntries := make(map[CommandEntryID]commandEntryGrammar, len(model.AllCommandEntries()))
	for _, entry := range model.AllCommandEntries() {
		grammarEntries[entry.id] = entry
	}
	for _, record := range input.CommandEntries {
		commandRecords[record.meta.id] = record
		if grammar, ok := grammarEntries[record.meta.id]; ok && grammar.kind == CommandEntryCanonical {
			canonicalByTarget[record.target] = record.lifecycle
		}
	}
	actionRecords := make(map[ActionEntryID]actionEntryRecord, len(input.ActionEntries))
	for _, record := range input.ActionEntries {
		actionRecords[record.meta.id] = record
	}
	flagRecords := make(map[FlagID]flagRecord, len(input.Flags))
	for _, record := range input.Flags {
		flagRecords[record.meta.id] = record
	}
	spellingRecords := make(map[FlagSpellingID]flagSpellingRecord, len(input.FlagSpellings))
	for _, record := range input.FlagSpellings {
		spellingRecords[record.meta.id] = record
	}

	var lexical []lexicalEntry
	commands := map[CommandID]struct{}{model.Root(): {}}
	for _, entry := range model.AllCommandEntries() {
		commands[entry.parent] = struct{}{}
	}
	for _, action := range model.AllActionCommands() {
		commands[action.parent] = struct{}{}
	}
	for command := range commands {
		for _, profile := range grammarBranches(model, command) {
			dialect, ok := grammarBranchDialect(model, command, profile)
			if !ok {
				return []rejection{{Kind: rejectionStructuralInvalidity}}
			}
			mode := dialect.CommandNormalization()
			if !mode.valid() {
				return []rejection{{Kind: rejectionInvalidNormalization}}
			}
			for _, entry := range model.CommandEntries(command) {
				if !grammarBranchContains(profile, entry.profile) {
					continue
				}
				record, ok := commandRecords[entry.id]
				if !ok {
					return []rejection{{
						Kind:  rejectionStructuralInvalidity,
						Order: uint64(entry.position),
					}}
				}
				life := record.lifecycle
				if entry.kind != CommandEntryCanonical {
					life = intersectReleaseLifecycles(scheme, life, canonicalByTarget[entry.target])
				}
				interval, valid := acceptanceLifecycleInterval(life, ranks)
				if !valid {
					return []rejection{
						{
							Kind:    rejectionInvalidLifecycleInterval,
							Order:   uint64(entry.position),
							Subject: uint64(entry.id.uint32()),
						},
					}
				}
				entry2 := lexicalEntry{
					Name:      entry.name,
					Mode:      mode,
					Namespace: uint64(command.uint32()),
					Profile:   uint64(profile.uint32()),
					Lifecycle: interval,
					Order:     uint64(entry.position),
					Subject:   uint64(entry.id.uint32()),
				}
				lexical = append(lexical, entry2)
			}
			for _, action := range model.ActionCommands(command) {
				record, ok := actionRecords[action.id]
				if !ok {
					return []rejection{{
						Kind:  rejectionStructuralInvalidity,
						Order: uint64(action.position),
					}}
				}
				interval, valid := acceptanceLifecycleInterval(record.lifecycle, ranks)
				if !valid {
					return []rejection{
						{
							Kind:    rejectionInvalidLifecycleInterval,
							Order:   uint64(action.position),
							Subject: 1<<32 | uint64(action.id.uint32()),
						},
					}
				}
				entry3 := lexicalEntry{
					Name:      action.name,
					Mode:      mode,
					Namespace: uint64(command.uint32()),
					Profile:   uint64(profile.uint32()),
					Lifecycle: interval,
					Order:     uint64(action.position),
					Subject:   1<<32 | uint64(action.id.uint32()),
				}
				lexical = append(lexical, entry3)
			}
		}
	}

	flagCommands := make(map[CommandID]struct{})
	for _, flag := range model.AllFlags() {
		flagCommands[flag.command] = struct{}{}
	}
	for _, action := range model.AllActionFlags() {
		flagCommands[action.parent] = struct{}{}
	}
	for command := range flagCommands {
		for _, profile := range grammarBranches(model, command) {
			dialect, ok := grammarBranchDialect(model, command, profile)
			if !ok {
				return []rejection{{Kind: rejectionStructuralInvalidity}}
			}
			mode := dialect.FlagNormalization()
			if !mode.valid() {
				return []rejection{{Kind: rejectionInvalidNormalization}}
			}
			for _, flag := range model.Flags(command) {
				if !grammarBranchContains(profile, flag.profile) {
					continue
				}
				flagRecord, ok := flagRecords[flag.id]
				if !ok {
					return []rejection{{
						Kind:  rejectionStructuralInvalidity,
						Order: uint64(flag.position),
					}}
				}
				for _, spelling := range flag.Spellings() {
					record, ok := spellingRecords[spelling.id]
					if !ok {
						return []rejection{{
							Kind:  rejectionStructuralInvalidity,
							Order: uint64(spelling.position),
						}}
					}
					life := intersectReleaseLifecycles(scheme, flagRecord.lifecycle, record.lifecycle)
					interval, valid := acceptanceLifecycleInterval(life, ranks)
					if !valid {
						return []rejection{
							{
								Kind:    rejectionInvalidLifecycleInterval,
								Order:   uint64(spelling.position),
								Subject: 2<<32 | uint64(spelling.id.uint32()),
							},
						}
					}
					entry4 := lexicalEntry{
						Name:      spelling.text,
						Mode:      mode,
						Namespace: uint64(command.uint32()) | 1<<63,
						Profile:   uint64(profile.uint32()),
						Lifecycle: interval,
						Order:     uint64(spelling.position),
						Subject:   2<<32 | uint64(spelling.id.uint32()),
					}
					lexical = append(lexical, entry4)
				}
			}
			for _, action := range model.ActionFlags(command) {
				record, ok := actionRecords[action.id]
				if !ok {
					return []rejection{{
						Kind:  rejectionStructuralInvalidity,
						Order: uint64(action.position),
					}}
				}
				interval, valid := acceptanceLifecycleInterval(record.lifecycle, ranks)
				if !valid {
					return []rejection{
						{
							Kind:    rejectionInvalidLifecycleInterval,
							Order:   uint64(action.position),
							Subject: 3<<32 | uint64(action.id.uint32()),
						},
					}
				}
				entry5 := lexicalEntry{
					Name:      action.text,
					Mode:      mode,
					Namespace: uint64(command.uint32()) | 1<<63,
					Profile:   uint64(profile.uint32()),
					Lifecycle: interval,
					Order:     uint64(action.position),
					Subject:   3<<32 | uint64(action.id.uint32()),
				}
				lexical = append(lexical, entry5)
			}
		}
	}
	return validateLexicalCollisions(lexical)
}

func lifecycleBoundaryRanks(scheme ReleaseScheme, input declarationRecordInput) (map[string]int64, []rejection) {
	set := make(map[string]struct{})
	add := func(l ReleaseLifecycle) {
		if l.IntroducedIn != "" {
			set[l.IntroducedIn] = struct{}{}
		}
		if l.RemovedIn != "" {
			set[l.RemovedIn] = struct{}{}
		}
	}
	for _, r := range input.CommandEntries {
		add(r.lifecycle)
	}
	for _, r := range input.ActionEntries {
		add(r.lifecycle)
	}
	for _, r := range input.Flags {
		add(r.lifecycle)
	}
	for _, r := range input.FlagSpellings {
		add(r.lifecycle)
	}
	if len(set) == 0 {
		return map[string]int64{}, nil
	}
	if !scheme.valid() {
		return nil, []rejection{{Kind: rejectionInvalidLifecycleInterval}}
	}
	values := make([]string, 0, len(set))
	for value := range set {
		if scheme.validate(value) != nil {
			return nil, []rejection{{Kind: rejectionInvalidLifecycleInterval}}
		}
		values = append(values, value)
	}
	slices.SortFunc(values, func(left, right string) int {
		order, _ := scheme.compare(left, right)
		return order
	})
	ranks := make(map[string]int64, len(values))
	for i, value := range values {
		ranks[value] = int64(i + 1)
	}
	return ranks, nil
}

func acceptanceLifecycleInterval(l ReleaseLifecycle, ranks map[string]int64) (lifecycleInterval, bool) {
	var interval lifecycleInterval
	if l.IntroducedIn != "" {
		interval.Start, interval.HasStart = ranks[l.IntroducedIn], true
	}
	if l.RemovedIn != "" {
		interval.End, interval.HasEnd = ranks[l.RemovedIn], true
	}
	return interval, interval.Valid()
}

func intersectReleaseLifecycles(scheme ReleaseScheme, left, right ReleaseLifecycle) ReleaseLifecycle {
	if right == (ReleaseLifecycle{}) {
		return left
	}
	if left == (ReleaseLifecycle{}) {
		return right
	}
	result := left
	if result.IntroducedIn == "" ||
		(right.IntroducedIn != "" && releaseAfter(scheme, right.IntroducedIn, result.IntroducedIn)) {
		result.IntroducedIn = right.IntroducedIn
	}
	if result.RemovedIn == "" || (right.RemovedIn != "" && releaseBefore(scheme, right.RemovedIn, result.RemovedIn)) {
		result.RemovedIn = right.RemovedIn
	}
	// Deprecation is presentation state, not active-interval ownership. Preserve
	// the later applicable marker when both declarations carry one.
	if result.DeprecatedIn == "" ||
		(right.DeprecatedIn != "" && releaseAfter(scheme, right.DeprecatedIn, result.DeprecatedIn)) {
		result.DeprecatedIn = right.DeprecatedIn
	}
	return result
}

func releaseAfter(scheme ReleaseScheme, left, right string) bool {
	if right == "" {
		return left != ""
	}
	cmp, err := scheme.compare(left, right)
	return err == nil && cmp > 0
}

func releaseBefore(scheme ReleaseScheme, left, right string) bool {
	if right == "" {
		return false
	}
	cmp, err := scheme.compare(left, right)
	return err == nil && cmp < 0
}

var (
	errInvalidReleaseScheme = errors.New("semantic: invalid release scheme")
	errInvalidRelease       = errors.New("semantic: invalid release")
	errInvalidLifecycle     = errors.New("semantic: invalid release lifecycle")
)

const semVerCorePartCount = 3

func validateLifecycle(scheme ReleaseScheme, lifecycle ReleaseLifecycle) error {
	if lifecycle == (ReleaseLifecycle{}) {
		return nil
	}
	if !scheme.valid() {
		return errInvalidLifecycle
	}
	for _, release := range []string{lifecycle.IntroducedIn, lifecycle.DeprecatedIn, lifecycle.RemovedIn} {
		if release != "" && scheme.validate(release) != nil {
			return errInvalidLifecycle
		}
	}
	if lifecycle.IntroducedIn != "" && lifecycle.DeprecatedIn != "" {
		cmp, _ := scheme.compare(lifecycle.IntroducedIn, lifecycle.DeprecatedIn)
		if cmp > 0 {
			return errInvalidLifecycle
		}
	}
	if lifecycle.IntroducedIn != "" && lifecycle.RemovedIn != "" {
		cmp, _ := scheme.compare(lifecycle.IntroducedIn, lifecycle.RemovedIn)
		if cmp >= 0 {
			return errInvalidLifecycle
		}
	}
	if lifecycle.DeprecatedIn != "" && lifecycle.RemovedIn != "" {
		cmp, _ := scheme.compare(lifecycle.DeprecatedIn, lifecycle.RemovedIn)
		if cmp >= 0 {
			return errInvalidLifecycle
		}
	}
	return nil
}

type canonicalCommandPathStep struct {
	Command CommandID
	Entry   CommandEntryID
}

func canonicalGrammarEntry(grammar grammarView, parent, target CommandID) (commandEntryGrammar, bool) {
	var canonical commandEntryGrammar
	found := false
	for _, entry := range grammar.CommandEntries(parent) {
		if entry.kind != CommandEntryCanonical || entry.target != target {
			continue
		}
		if found {
			return commandEntryGrammar{}, false
		}
		canonical = entry
		found = true
	}
	return canonical, found
}

func reverseCanonicalCommandPath(reversed []canonicalCommandPathStep) []canonicalCommandPathStep {
	path := make([]canonicalCommandPathStep, len(reversed))
	for i := range reversed {
		path[len(reversed)-1-i] = reversed[i]
	}
	return path
}

func canonicalCommandPathWithLookups(valid bool, grammar grammarView, ancestor, target CommandID, commandLookup func(CommandID) (commandRecord, bool), entryLookup func(CommandEntryID) (commandEntryRecord, bool)) ([]canonicalCommandPathStep, error) {
	if !valid || grammar == nil || !ancestor.present() || !target.present() {
		return nil, errInvalidModelTopology
	}
	if _, ok := commandLookup(ancestor); !ok {
		return nil, errInvalidModelTopology
	}
	if ancestor == target {
		return nil, nil
	}

	seen := make(map[CommandID]struct{})
	var reversed []canonicalCommandPathStep
	current := target
	for current != ancestor {
		if _, duplicate := seen[current]; duplicate {
			return nil, errInvalidModelTopology
		}
		seen[current] = struct{}{}

		command, ok := commandLookup(current)
		if !ok {
			return nil, errInvalidModelTopology
		}
		parent, ok := command.Parent()
		if !ok {
			return nil, errInvalidModelTopology
		}

		canonical, found := canonicalGrammarEntry(grammar, parent, current)
		if !found {
			return nil, errInvalidModelTopology
		}
		if _, ok := entryLookup(canonical.id); !ok {
			return nil, errInvalidModelTopology
		}
		step := canonicalCommandPathStep{
			Command: current,
			Entry:   canonical.id,
		}
		reversed = append(reversed, step)
		current = parent
	}

	return reverseCanonicalCommandPath(reversed), nil
}

func canonicalCommandPath(model *effectiveModel, grammar grammarView, ancestor, target CommandID) ([]canonicalCommandPathStep, error) {
	return canonicalCommandPathWithLookups(model.Valid(), grammar, ancestor, target, model.Command, model.CommandEntry)
}

func intersectCanonicalCommandPathLifecycle(model *effectiveModel, scheme ReleaseScheme, path []canonicalCommandPathStep) (ReleaseLifecycle, error) {
	lifecycle := ReleaseLifecycle{}
	for _, step := range path {
		record, ok := model.CommandEntry(step.Entry)
		if !ok {
			return ReleaseLifecycle{}, errInvalidModelTopology
		}
		lifecycle = intersectEffectiveReleaseLifecycles(scheme, lifecycle, record.lifecycle)
	}
	return lifecycle, nil
}

func intersectEffectiveReleaseLifecycles(scheme ReleaseScheme, left, right ReleaseLifecycle) ReleaseLifecycle {
	result := ReleaseLifecycle{}
	result.IntroducedIn = laterReleaseBoundary(scheme, left.IntroducedIn, right.IntroducedIn)
	result.RemovedIn = earlierReleaseEndBoundary(scheme, left.RemovedIn, right.RemovedIn)
	result.DeprecatedIn = laterReleaseBoundary(scheme, left.DeprecatedIn, right.DeprecatedIn)
	if result.DeprecatedIn != "" && result.RemovedIn != "" {
		if cmp, err := scheme.compare(result.DeprecatedIn, result.RemovedIn); err == nil && cmp >= 0 {
			result.DeprecatedIn = ""
		}
	}
	return result
}

func laterReleaseBoundary(scheme ReleaseScheme, left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if cmp, err := scheme.compare(left, right); err == nil && cmp < 0 {
		return right
	}
	return left
}

func earlierReleaseEndBoundary(scheme ReleaseScheme, left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if cmp, err := scheme.compare(left, right); err == nil && cmp > 0 {
		return right
	}
	return left
}

func evaluateLifecycle(scheme ReleaseScheme, lifecycle ReleaseLifecycle, release string) (LifecycleState, error) {
	if err := validateLifecycle(scheme, lifecycle); err != nil {
		return LifecycleStateUnknown, err
	}
	if release == "" {
		switch {
		case scheme.valid():
			release = scheme.current()
		case lifecycle == (ReleaseLifecycle{}):
			return LifecycleStateCurrent, nil
		default:
			return LifecycleStateUnknown, errInvalidLifecycle
		}
	}
	if !scheme.valid() {
		return LifecycleStateUnknown, errInvalidReleaseScheme
	}
	if err := scheme.validate(release); err != nil {
		return LifecycleStateUnknown, err
	}
	if lifecycle.IntroducedIn != "" {
		cmp, _ := scheme.compare(release, lifecycle.IntroducedIn)
		if cmp < 0 {
			return LifecycleStateNotIntroduced, nil
		}
	}
	if lifecycle.RemovedIn != "" {
		cmp, _ := scheme.compare(release, lifecycle.RemovedIn)
		if cmp >= 0 {
			return LifecycleStateRemoved, nil
		}
	}
	if lifecycle.DeprecatedIn != "" {
		cmp, _ := scheme.compare(release, lifecycle.DeprecatedIn)
		if cmp >= 0 {
			return LifecycleStateDeprecated, nil
		}
	}
	return LifecycleStateCurrent, nil
}

type semVer struct {
	major uint64
	minor uint64
	patch uint64
	pre   []semVerIdentifier
}

type semVerIdentifier struct {
	text    string
	numeric bool
	number  uint64
}

func parseSemVer(value string) (semVer, bool) {
	var result semVer
	if value == "" || strings.HasPrefix(value, "v") {
		return result, false
	}
	coreAndPre := value
	if plus := strings.IndexByte(coreAndPre, '+'); plus >= 0 {
		build := coreAndPre[plus+1:]
		if build == "" || !validDotIdentifiers(build, false) {
			return result, false
		}
		coreAndPre = coreAndPre[:plus]
	}
	core := coreAndPre
	pre := ""
	if dash := strings.IndexByte(coreAndPre, '-'); dash >= 0 {
		pre = coreAndPre[dash+1:]
		core = coreAndPre[:dash]
		if pre == "" {
			return result, false
		}
	}
	parts := strings.Split(core, ".")
	if len(parts) != semVerCorePartCount {
		return result, false
	}
	var ok bool
	if result.major, ok = parseCoreNumber(parts[0]); !ok {
		return semVer{}, false
	}
	if result.minor, ok = parseCoreNumber(parts[1]); !ok {
		return semVer{}, false
	}
	if result.patch, ok = parseCoreNumber(parts[2]); !ok {
		return semVer{}, false
	}
	if pre == "" {
		return result, true
	}
	for part := range strings.SplitSeq(pre, ".") {
		if part == "" || !validIdentifier(part) {
			return semVer{}, false
		}
		identifier := semVerIdentifier{text: part}
		if allDigits(part) {
			if len(part) > 1 && part[0] == '0' {
				return semVer{}, false
			}
			n, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				return semVer{}, false
			}
			identifier.numeric = true
			identifier.number = n
		}
		result.pre = append(result.pre, identifier)
	}
	return result, true
}

func parseCoreNumber(value string) (uint64, bool) {
	if value == "" || (len(value) > 1 && value[0] == '0') || !allDigits(value) {
		return 0, false
	}
	n, err := strconv.ParseUint(value, 10, 64)
	return n, err == nil
}

func validDotIdentifiers(value string, prerelease bool) bool {
	for part := range strings.SplitSeq(value, ".") {
		if part == "" || !validIdentifier(part) {
			return false
		}
		if prerelease && allDigits(part) && len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}

func validIdentifier(value string) bool {
	for i := range len(value) {
		b := value[i]
		if (b < '0' || b > '9') && (b < 'A' || b > 'Z') && (b < 'a' || b > 'z') && b != '-' {
			return false
		}
	}
	return value != ""
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := range len(value) {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func compareSemVer(left, right semVer) int {
	if cmp := compareUint64(left.major, right.major); cmp != 0 {
		return cmp
	}
	if cmp := compareUint64(left.minor, right.minor); cmp != 0 {
		return cmp
	}
	if cmp := compareUint64(left.patch, right.patch); cmp != 0 {
		return cmp
	}
	if len(left.pre) == 0 && len(right.pre) == 0 {
		return 0
	}
	if len(left.pre) == 0 {
		return 1
	}
	if len(right.pre) == 0 {
		return -1
	}
	limit := min(len(right.pre), len(left.pre))
	for i := range limit {
		l, r := left.pre[i], right.pre[i]
		if l.numeric && r.numeric {
			if cmp := compareUint64(l.number, r.number); cmp != 0 {
				return cmp
			}
			continue
		}
		if l.numeric != r.numeric {
			if l.numeric {
				return -1
			}
			return 1
		}
		if l.text < r.text {
			return -1
		}
		if l.text > r.text {
			return 1
		}
	}
	return compareInt(len(left.pre), len(right.pre))
}

func compareInt(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareUint64(left, right uint64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func intersectStaticCommandPathLifecycle(model staticProgram, scheme ReleaseScheme, path []canonicalCommandPathStep) (ReleaseLifecycle, error) {
	lifecycle := ReleaseLifecycle{}
	for _, step := range path {
		record, ok := model.CommandEntry(step.Entry)
		if !ok {
			return ReleaseLifecycle{}, errInvalidModelTopology
		}
		lifecycle = intersectEffectiveReleaseLifecycles(scheme, lifecycle, record.lifecycle)
	}
	return lifecycle, nil
}
