package cli

import "slices"

type lexicalNamespaceKey struct {
	frontier int
	command  CommandID
	profile  GrammarProfileID
}

type commandLexicalMember struct {
	name      string
	entry     commandEntryGrammar
	action    actionCommandGrammar
	lifecycle LifecycleState
}

type commandLexicalNamespace struct {
	entries      []commandEntryGrammar
	actions      []actionCommandGrammar
	entryMembers []commandLexicalMember
	members      []commandLexicalMember
	exactEntries map[string][]int
	exact        map[string][]int
}

type flagLexicalNamespace struct {
	flags   []flagGrammar
	actions []semanticActionFlagGrammar
}

type grammarOperationSource struct {
	operation *invocation
	analysis  *analysisState
}

type analysisOperationGrammarScope struct {
	source     grammarOperationSource
	base       grammarView
	generation int
	selected   map[CommandID]GrammarProfileID
}

func analysisEffectiveGrammar(source grammarOperationSource, base grammarView, generation int, selected map[CommandID]GrammarProfileID) grammarView {
	result := analysisOperationGrammarScope{
		source:     source,
		base:       base,
		generation: generation,
		selected:   selected,
	}

	return result
}

func (scope analysisOperationGrammarScope) lexicalNamespaceKey(command CommandID) (lexicalNamespaceKey, bool) {
	if scope.source.operation == nil {
		return lexicalNamespaceKey{}, false
	}
	result := lexicalNamespaceKey{
		frontier: scope.generation,
		command:  command,
		profile:  scope.selected[command],
	}
	return result, true
}

func (analysis *analysisState) cachedCommandLexicalNamespace(key lexicalNamespaceKey, build func() *commandLexicalNamespace) *commandLexicalNamespace {
	if analysis == nil {
		panic("cli: nil analysis grammar cache")
	}
	analysis.grammar.mu.RLock()
	cached := analysis.grammar.commandLexicalNamespaces[key]
	analysis.grammar.mu.RUnlock()
	if cached != nil {
		return cached
	}
	built := build()
	analysis.grammar.mu.Lock()
	if analysis.grammar.commandLexicalNamespaces == nil {
		analysis.grammar.commandLexicalNamespaces = make(map[lexicalNamespaceKey]*commandLexicalNamespace)
	}
	if cached = analysis.grammar.commandLexicalNamespaces[key]; cached == nil {
		analysis.grammar.commandLexicalNamespaces[key] = built
		cached = built
	}
	analysis.grammar.mu.Unlock()
	return cached
}

func (analysis *analysisState) cachedFlagLexicalNamespace(key lexicalNamespaceKey, build func() *flagLexicalNamespace) *flagLexicalNamespace {
	if analysis == nil {
		panic("cli: nil analysis grammar cache")
	}
	analysis.grammar.mu.RLock()
	cached := analysis.grammar.flagLexicalNamespaces[key]
	analysis.grammar.mu.RUnlock()
	if cached != nil {
		return cached
	}
	built := build()
	analysis.grammar.mu.Lock()
	if analysis.grammar.flagLexicalNamespaces == nil {
		analysis.grammar.flagLexicalNamespaces = make(map[lexicalNamespaceKey]*flagLexicalNamespace)
	}
	if cached = analysis.grammar.flagLexicalNamespaces[key]; cached == nil {
		analysis.grammar.flagLexicalNamespaces[key] = built
		cached = built
	}
	analysis.grammar.mu.Unlock()
	return cached
}

func (scope analysisOperationGrammarScope) commandLexicalNamespace(parent CommandID) *commandLexicalNamespace {
	build := func() *commandLexicalNamespace {
		entries, actions := lifecycleOwnedCommandGrammar(scope.source.analysis, scope.base, scope.selected, parent)
		dialect, ok := scope.Dialect(parent)
		if !ok {
			namespace := &commandLexicalNamespace{
				entries: entries,
				actions: actions,
			}
			return namespace
		}
		mode := dialect.CommandNormalization()
		canonicalLifecycles := canonicalCommandEntryLifecycles(scope.source.analysis, scope.base.CommandEntries(parent))
		namespace := &commandLexicalNamespace{
			entries:      entries,
			actions:      actions,
			exactEntries: make(map[string][]int),
			exact:        make(map[string][]int),
		}
		for _, entry := range entries {
			name, valid := normalizeName(mode, entry.name)
			if !valid {
				continue
			}
			life := effectiveCommandEntryLifecycleWithCanonical(scope.source.analysis, scope.base, entry, canonicalLifecycles)
			state, _ := evaluateLifecycle(scope.source.operation.program.Application().release, life, "")
			member := commandLexicalMember{
				name:      name,
				entry:     entry,
				lifecycle: state,
			}
			namespace.entryMembers = append(namespace.entryMembers, member)
			namespace.members = append(namespace.members, member)
			namespace.exactEntries[name] = append(namespace.exactEntries[name], len(namespace.entryMembers)-1)
			namespace.exact[name] = append(namespace.exact[name], len(namespace.members)-1)
		}
		for _, action := range actions {
			name, valid := normalizeName(mode, action.name)
			if !valid {
				continue
			}
			state, _ := evaluateLifecycle(
				scope.source.operation.program.Application().release,
				actionLifecycle(scope.source.analysis, action.id),
				"",
			)
			member := commandLexicalMember{
				name:      name,
				action:    action,
				lifecycle: state,
			}
			namespace.members = append(namespace.members, member)
			namespace.exact[name] = append(namespace.exact[name], len(namespace.members)-1)
		}
		return namespace
	}
	if key, ok := scope.lexicalNamespaceKey(parent); ok {
		return scope.source.analysis.cachedCommandLexicalNamespace(key, build)
	}
	return build()
}

func (scope analysisOperationGrammarScope) flagLexicalNamespace(command CommandID) *flagLexicalNamespace {
	build := func() *flagLexicalNamespace {
		flags, actions := lifecycleOwnedFlagGrammar(scope.source.analysis, scope.base, scope.selected, command)
		namespace := &flagLexicalNamespace{
			flags:   flags,
			actions: actions,
		}
		return namespace
	}
	if key, ok := scope.lexicalNamespaceKey(command); ok {
		return scope.source.analysis.cachedFlagLexicalNamespace(key, build)
	}
	return build()
}

func (scope analysisOperationGrammarScope) Root() CommandID { return scope.base.Root() }

func (scope analysisOperationGrammarScope) Dialect(command CommandID) (dialect, bool) {
	if profile := scope.selected[command]; profile.present() {
		if dialect, ok := scope.base.ProfileDialect(profile); ok {
			return dialect, true
		}
	}
	return scope.base.Dialect(command)
}

func (scope analysisOperationGrammarScope) ProfileDialect(profile GrammarProfileID) (dialect, bool) {
	return scope.base.ProfileDialect(profile)
}

func (scope analysisOperationGrammarScope) Profiles(command CommandID) []GrammarProfileID {
	return scope.base.Profiles(command)
}

func (scope analysisOperationGrammarScope) CommandEntries(parent CommandID) []commandEntryGrammar {
	namespace := scope.commandLexicalNamespace(parent)
	return append([]commandEntryGrammar(nil), namespace.entries...)
}

func (scope analysisOperationGrammarScope) ActionCommands(parent CommandID) []actionCommandGrammar {
	namespace := scope.commandLexicalNamespace(parent)
	return append([]actionCommandGrammar(nil), namespace.actions...)
}

func (scope analysisOperationGrammarScope) ActionFlags(command CommandID) []semanticActionFlagGrammar {
	namespace := scope.flagLexicalNamespace(command)
	return append([]semanticActionFlagGrammar(nil), namespace.actions...)
}

func (scope analysisOperationGrammarScope) Flags(command CommandID) []flagGrammar {
	namespace := scope.flagLexicalNamespace(command)
	return append([]flagGrammar(nil), namespace.flags...)
}

func (scope analysisOperationGrammarScope) ArgumentForms(command CommandID) []argumentFormGrammar {
	forms := scope.base.ArgumentForms(command)
	result := make([]argumentFormGrammar, 0, len(forms))
	for _, form := range forms {
		if form.profile.present() && scope.selected[command] != form.profile {
			continue
		}
		state := argumentFormLifecycleState(scope.source.analysis, form.id)
		if state == LifecycleStateNotIntroduced || state == LifecycleStateRemoved {
			form.enabled = false
			result = append(result, form)
			continue
		}
		terms := form.Terms()
		for i := range terms {
			state := argumentTermLifecycleState(scope.source.analysis, terms[i].id)
			if state == LifecycleStateNotIntroduced || state == LifecycleStateRemoved {
				terms[i].disabled = true
			}
		}
		form.terms = append([]argumentTermGrammar(nil), terms...)
		result = append(result, form)
	}
	return result
}

type availabilityGrammar struct {
	base          grammarView
	commands      map[CommandID]bool
	flags         map[FlagID]bool
	argumentForms map[ArgumentFormID]bool
	argumentTerms map[ArgumentTermID]bool
}

func (grammar availabilityGrammar) Root() CommandID { return grammar.base.Root() }

func (grammar availabilityGrammar) Dialect(command CommandID) (dialect, bool) {
	return grammar.base.Dialect(command)
}

func (grammar availabilityGrammar) ProfileDialect(profile GrammarProfileID) (dialect, bool) {
	return grammar.base.ProfileDialect(profile)
}

func (grammar availabilityGrammar) Profiles(command CommandID) []GrammarProfileID {
	return grammar.base.Profiles(command)
}

func (grammar availabilityGrammar) CommandEntries(parent CommandID) []commandEntryGrammar {
	entries := grammar.base.CommandEntries(parent)
	if len(grammar.commands) == 0 {
		return entries
	}
	for i := range entries {
		if enabled, ok := grammar.commands[entries[i].target]; ok && !enabled {
			entries[i].enabled = false
		}
	}
	return entries
}

func (grammar availabilityGrammar) ActionCommands(parent CommandID) []actionCommandGrammar {
	return grammar.base.ActionCommands(parent)
}

func (grammar availabilityGrammar) ActionFlags(command CommandID) []semanticActionFlagGrammar {
	return grammar.base.ActionFlags(command)
}

func (grammar availabilityGrammar) Flags(command CommandID) []flagGrammar {
	flags := grammar.base.Flags(command)
	if len(grammar.flags) == 0 {
		return flags
	}
	for i := range flags {
		if enabled, ok := grammar.flags[flags[i].id]; ok && !enabled {
			flags[i].disabled = true
		}
	}
	return flags
}

func (grammar availabilityGrammar) ArgumentForms(command CommandID) []argumentFormGrammar {
	forms := grammar.base.ArgumentForms(command)
	if len(grammar.argumentForms) == 0 && len(grammar.argumentTerms) == 0 {
		return forms
	}
	for i := range forms {
		if enabled, ok := grammar.argumentForms[forms[i].id]; ok && !enabled {
			forms[i].enabled = false
		}
		terms := forms[i].Terms()
		changed := false
		for j := range terms {
			if enabled, ok := grammar.argumentTerms[terms[j].id]; ok && !enabled {
				terms[j].disabled = true
				changed = true
			}
		}
		if changed {
			forms[i].terms = append([]argumentTermGrammar(nil), terms...)
		}
	}
	return forms
}

// scope returns the bounded operation grammar read surface for
// the supplied selected-profile state. The returned reader is a lightweight
// view; it owns no publication or parser state.
func (source grammarOperationSource) scope(selected map[CommandID]GrammarProfileID) (grammarView, bool) {
	if source.operation == nil || source.analysis == nil || source.analysis.effectiveModel == nil {
		return nil, false
	}
	base, generation, ok := source.analysis.effectiveModel.GrammarSnapshot()
	if !ok {
		return nil, false
	}
	return analysisEffectiveGrammar(source, base, generation, selected), true
}

const (
	keyedDestinationDomainUseCapacity = 2

	builtinValueLanguageNameBool   = "bool"
	builtinValueLanguageNameInt    = "int"
	builtinValueLanguageNameUint   = "uint"
	builtinValueLanguageNameString = "string"
)

// CountRange describes a bounded or unbounded count accepted by argument and flag grammar.
// Construct valid ranges with Count; the zero value is not a declared count range.
type CountRange struct {
	minimum   int
	maximum   int
	unbounded bool
	valid     bool
}

// countConstructors is the stateless namespace behind Count and constructs validated count ranges.
type countConstructors struct{}

// Count provides constructors for valid CountRange values.
var Count countConstructors

// Exactly returns a range that accepts exactly n items.
func (countConstructors) Exactly(n int) CountRange {
	return boundedCountRange(n, n)
}

// Between returns an inclusive bounded range from minimum through maximum.
func (countConstructors) Between(minimum, maximum int) CountRange {
	return boundedCountRange(minimum, maximum)
}

// AtLeast returns a range with minimum and no upper bound.
func (countConstructors) AtLeast(minimum int) CountRange {
	return unboundedCountRange(minimum)
}

// AtMost returns a range from zero through maximum.
func (countConstructors) AtMost(maximum int) CountRange {
	return boundedCountRange(0, maximum)
}

// Optional returns the zero-or-one range.
func (countConstructors) Optional() CountRange { return boundedCountRange(0, 1) }

// OneOrMore returns a range requiring at least one item.
func (countConstructors) OneOrMore() CountRange { return unboundedCountRange(1) }

// ZeroOrMore returns an unbounded range beginning at zero.
func (countConstructors) ZeroOrMore() CountRange { return unboundedCountRange(0) }

func boundedCountRange(minimum, maximum int) CountRange {
	if minimum < 0 || maximum < 0 || maximum < minimum {
		return CountRange{}
	}
	countRange := CountRange{
		minimum: minimum,
		maximum: maximum,
		valid:   true,
	}

	return countRange
}

func unboundedCountRange(minimum int) CountRange {
	if minimum < 0 {
		return CountRange{}
	}
	countRange := CountRange{
		minimum:   minimum,
		unbounded: true,
		valid:     true,
	}

	return countRange
}

// Minimum returns the inclusive minimum count.
func (count CountRange) Minimum() int { return count.minimum }

// Maximum returns the inclusive maximum count and reports false when the range is unbounded.
func (count CountRange) Maximum() (int, bool) {
	if count.unbounded {
		return 0, false
	}
	return count.maximum, true
}

// Unbounded reports whether the range has no maximum.
func (count CountRange) Unbounded() bool { return count.unbounded }

// semanticCountRange is a validated bounded-or-unbounded structural cardinality.
type semanticCountRange struct {
	minimum   int
	maximum   int
	unbounded bool
}

func boundedCount(minimum, maximum int) semanticCountRange {
	if minimum < 0 || maximum < minimum {
		panic("semantic: invalid bounded count")
	}
	count := semanticCountRange{
		minimum: minimum,
		maximum: maximum,
	}
	return count
}

func unboundedCount(minimum int) semanticCountRange {
	if minimum < 0 {
		panic("semantic: invalid unbounded count")
	}
	count := semanticCountRange{
		minimum:   minimum,
		unbounded: true,
	}
	return count
}

func (count semanticCountRange) Minimum() int { return count.minimum }

func (count semanticCountRange) Maximum() (int, bool) { return count.maximum, !count.unbounded }

func (count semanticCountRange) valid() bool {
	return count.minimum >= 0 && (count.unbounded || count.maximum >= count.minimum)
}

// grammarInput is immutable-constructor input for the static compiled grammar.
// The grammar is a semantic partition of the model, not a second command tree.
type grammarInput struct {
	Root            CommandID
	Dialects        map[CommandID]dialect
	ProfileDialects map[GrammarProfileID]dialect
	ProfileCommands map[GrammarProfileID]CommandID
	ProfileOrder    []GrammarProfileID
	CommandEntries  []commandEntryGrammar
	ActionCommands  []actionCommandGrammar
	ActionFlags     []semanticActionFlagGrammar
	Flags           []flagGrammar
	ArgumentForms   []argumentFormGrammar
}

// grammarView is the bounded read contract shared by immutable static
// grammar and operation-local effective grammar facets. It intentionally omits
// whole-family accessors so bounded consumers do not require whole-state
// reconstruction.
type grammarView interface {
	Root() CommandID
	Dialect(command CommandID) (dialect, bool)
	ProfileDialect(profile GrammarProfileID) (dialect, bool)
	Profiles(command CommandID) []GrammarProfileID
	CommandEntries(command CommandID) []commandEntryGrammar
	ActionCommands(command CommandID) []actionCommandGrammar
	ActionFlags(command CommandID) []semanticActionFlagGrammar
	Flags(command CommandID) []flagGrammar
	ArgumentForms(command CommandID) []argumentFormGrammar
}

type grammarModel struct {
	root                 CommandID
	dialects             map[CommandID]dialect
	profileDialects      map[GrammarProfileID]dialect
	profilesByCommand    map[CommandID][]GrammarProfileID
	commandEntries       []commandEntryGrammar
	actionCommands       []actionCommandGrammar
	actionFlags          []semanticActionFlagGrammar
	flags                []flagGrammar
	argumentForms        []argumentFormGrammar
	entriesByParent      map[CommandID][]commandEntryGrammar
	actionsByParent      map[CommandID][]actionCommandGrammar
	actionFlagsByCommand map[CommandID][]semanticActionFlagGrammar
	flagsByCommand       map[CommandID][]flagGrammar
	formsByCommand       map[CommandID][]argumentFormGrammar
}

func acceptGrammarModel(input grammarInput) grammarModel {
	if !input.Root.present() {
		panic("semantic: invalid grammar root")
	}
	model := grammarModel{
		root:                 input.Root,
		dialects:             make(map[CommandID]dialect, len(input.Dialects)),
		profileDialects:      make(map[GrammarProfileID]dialect, len(input.ProfileDialects)),
		profilesByCommand:    make(map[CommandID][]GrammarProfileID),
		commandEntries:       append([]commandEntryGrammar(nil), input.CommandEntries...),
		actionCommands:       append([]actionCommandGrammar(nil), input.ActionCommands...),
		actionFlags:          append([]semanticActionFlagGrammar(nil), input.ActionFlags...),
		flags:                append([]flagGrammar(nil), input.Flags...),
		argumentForms:        append([]argumentFormGrammar(nil), input.ArgumentForms...),
		entriesByParent:      make(map[CommandID][]commandEntryGrammar),
		actionsByParent:      make(map[CommandID][]actionCommandGrammar),
		actionFlagsByCommand: make(map[CommandID][]semanticActionFlagGrammar),
		flagsByCommand:       make(map[CommandID][]flagGrammar),
		formsByCommand:       make(map[CommandID][]argumentFormGrammar),
	}
	for id, dialect := range input.Dialects {
		if !id.present() {
			panic("semantic: invalid dialect command identity")
		}
		model.dialects[id] = cloneDialect(dialect)
	}
	for id, dialect := range input.ProfileDialects {
		if !id.present() {
			panic("semantic: invalid grammar-profile identity")
		}
		model.profileDialects[id] = cloneDialect(dialect)
	}
	seenProfiles := make(map[CommandID]map[GrammarProfileID]struct{})
	addProfile := func(command CommandID, profile GrammarProfileID) {
		if !profile.present() {
			return
		}
		seen := seenProfiles[command]
		if seen == nil {
			seen = make(map[GrammarProfileID]struct{})
			seenProfiles[command] = seen
		}
		if _, exists := seen[profile]; exists {
			return
		}
		seen[profile] = struct{}{}
		model.profilesByCommand[command] = append(model.profilesByCommand[command], profile)
	}
	seenProfileOrder := make(map[GrammarProfileID]struct{}, len(input.ProfileOrder))
	for _, profile := range input.ProfileOrder {
		command, ok := input.ProfileCommands[profile]
		if !profile.present() || !ok || !command.present() {
			panic("semantic: invalid grammar-profile command")
		}
		if _, ok := model.profileDialects[profile]; !ok {
			panic("semantic: grammar-profile command without dialect")
		}
		if _, duplicate := seenProfileOrder[profile]; duplicate {
			panic("semantic: duplicate grammar-profile order")
		}
		seenProfileOrder[profile] = struct{}{}
		addProfile(command, profile)
	}
	if len(seenProfileOrder) != len(input.ProfileCommands) || len(seenProfileOrder) != len(input.ProfileDialects) {
		panic("semantic: incomplete grammar-profile order")
	}
	for _, entry := range model.commandEntries {
		model.entriesByParent[entry.parent] = append(model.entriesByParent[entry.parent], entry)
		addProfile(entry.parent, entry.profile)
	}
	for _, action := range model.actionCommands {
		model.actionsByParent[action.parent] = append(model.actionsByParent[action.parent], action)
	}
	for _, action := range model.actionFlags {
		model.actionFlagsByCommand[action.parent] = append(model.actionFlagsByCommand[action.parent], action)
	}
	for _, flag := range model.flags {
		model.flagsByCommand[flag.command] = append(model.flagsByCommand[flag.command], flag)
		addProfile(flag.command, flag.profile)
	}
	for _, form := range model.argumentForms {
		model.formsByCommand[form.command] = append(model.formsByCommand[form.command], form)
		addProfile(form.command, form.profile)
	}
	return model
}

func cloneDialect(dialect dialect) dialect {
	result := dialect
	result.endOfOptions = append([]string(nil), dialect.endOfOptions...)
	return result
}

func (model grammarModel) Valid() bool { return model.root.present() }

func (model grammarModel) Root() CommandID { return model.root }

func (model grammarModel) Dialect(command CommandID) (dialect, bool) {
	dialect, ok := model.dialects[command]
	return cloneDialect(dialect), ok
}

func (model grammarModel) ProfileDialect(profile GrammarProfileID) (dialect, bool) {
	dialect, ok := model.profileDialects[profile]
	return cloneDialect(dialect), ok
}

func (model grammarModel) Profiles(command CommandID) []GrammarProfileID {
	return append([]GrammarProfileID(nil), model.profilesByCommand[command]...)
}

func (model grammarModel) CommandEntries(parent CommandID) []commandEntryGrammar {
	return append([]commandEntryGrammar(nil), model.entriesByParent[parent]...)
}

func (model grammarModel) ActionCommands(parent CommandID) []actionCommandGrammar {
	return append([]actionCommandGrammar(nil), model.actionsByParent[parent]...)
}

func (model grammarModel) ActionFlags(command CommandID) []semanticActionFlagGrammar {
	return append([]semanticActionFlagGrammar(nil), model.actionFlagsByCommand[command]...)
}

func (model grammarModel) Flags(command CommandID) []flagGrammar {
	return append([]flagGrammar(nil), model.flagsByCommand[command]...)
}

func (model grammarModel) ArgumentForms(command CommandID) []argumentFormGrammar {
	return append([]argumentFormGrammar(nil), model.formsByCommand[command]...)
}

func (model grammarModel) AllCommandEntries() []commandEntryGrammar {
	return append([]commandEntryGrammar(nil), model.commandEntries...)
}

func (model grammarModel) AllActionCommands() []actionCommandGrammar {
	return append([]actionCommandGrammar(nil), model.actionCommands...)
}

func (model grammarModel) AllActionFlags() []semanticActionFlagGrammar {
	return append([]semanticActionFlagGrammar(nil), model.actionFlags...)
}

func (model grammarModel) AllFlags() []flagGrammar { return append([]flagGrammar(nil), model.flags...) }

func (model grammarModel) AllArgumentForms() []argumentFormGrammar {
	return append([]argumentFormGrammar(nil), model.argumentForms...)
}

type providerGrammarValue[T any] struct {
	frontier int
	value    T
}

type providerGrammarIndex struct {
	dialects        map[CommandID][]providerGrammarValue[dialect]
	profileDialects map[GrammarProfileID][]providerGrammarValue[dialect]
	profiles        map[CommandID][]providerGrammarValue[GrammarProfileID]
	commandEntries  map[CommandID][]providerGrammarValue[commandEntryGrammar]
	actionCommands  map[CommandID][]providerGrammarValue[actionCommandGrammar]
	actionFlags     map[CommandID][]providerGrammarValue[semanticActionFlagGrammar]
	flags           map[CommandID][]providerGrammarValue[flagGrammar]
	argumentForms   map[CommandID][]providerGrammarValue[argumentFormGrammar]
}

func (index *providerGrammarIndex) empty() bool {
	return index.dialects == nil &&
		index.profileDialects == nil &&
		index.profiles == nil &&
		index.commandEntries == nil &&
		index.actionCommands == nil &&
		index.actionFlags == nil &&
		index.flags == nil &&
		index.argumentForms == nil
}

func appendProviderGrammarValue[K comparable, T any](index *map[K][]providerGrammarValue[T], key K, frontier int, value T) {
	if *index == nil {
		*index = make(map[K][]providerGrammarValue[T])
	}
	(*index)[key] = append((*index)[key], providerGrammarValue[T]{
		frontier: frontier,
		value:    value,
	})
}

func (index *providerGrammarIndex) add(delta grammarDelta, frontier int) {
	if index == nil || frontier <= 0 || delta.Empty() {
		return
	}
	input := delta.Input()
	for command, dialect := range input.Dialects {
		appendProviderGrammarValue(&index.dialects, command, frontier, dialect)
	}
	for profile, dialect := range input.ProfileDialects {
		appendProviderGrammarValue(&index.profileDialects, profile, frontier, dialect)
	}
	for _, profile := range input.ProfileOrder {
		command, ok := input.ProfileCommands[profile]
		if ok {
			appendProviderGrammarValue(&index.profiles, command, frontier, profile)
		}
	}
	for _, entry := range input.CommandEntries {
		appendProviderGrammarValue(&index.commandEntries, entry.parent, frontier, entry)
	}
	for _, action := range input.ActionCommands {
		appendProviderGrammarValue(&index.actionCommands, action.parent, frontier, action)
	}
	for _, action := range input.ActionFlags {
		appendProviderGrammarValue(&index.actionFlags, action.parent, frontier, action)
	}
	for _, flag := range input.Flags {
		appendProviderGrammarValue(&index.flags, flag.command, frontier, flag)
	}
	for _, form := range input.ArgumentForms {
		appendProviderGrammarValue(&index.argumentForms, form.command, frontier, form)
	}
}

func providerGrammarLatest[T any](values []providerGrammarValue[T], frontier int) (T, bool) {
	for _, value := range slices.Backward(values) {
		if value.frontier <= frontier {
			return value.value, true
		}
	}
	var zero T
	return zero, false
}

func providerGrammarList[T any](base []T, values []providerGrammarValue[T], frontier int) []T {
	result := base
	for _, entry := range values {
		if entry.frontier > frontier {
			break
		}
		result = append(result, entry.value)
	}
	return result
}

// effectiveGrammar is an immutable operation-frontier read facet over the
// compiled static GrammarModel plus accepted provider grammar facts visible
// when the facet is obtained. Canonical provider publications remain authority;
// the effective-declarations index is rebuildable acceleration keyed only by query scope.
type effectiveGrammar struct {
	base               grammarModel
	state              *effectiveModel
	frontier           int
	hasProviderGrammar bool
}

func (model *effectiveModel) effectiveGrammarLocked() (effectiveGrammar, bool) {
	base, ok := model.base.Grammar()
	if !ok || !base.Valid() {
		return effectiveGrammar{}, false
	}
	result := effectiveGrammar{
		base:               base,
		state:              model,
		frontier:           len(model.providers),
		hasProviderGrammar: !model.grammar.empty(),
	}
	return result, true
}

func (model *effectiveModel) GrammarGeneration(grammar grammarView) (int, bool) {
	if model == nil {
		return 0, false
	}
	effective, ok := grammar.(effectiveGrammar)
	if !ok || effective.state != model {
		return 0, false
	}
	return effective.frontier, true
}

func (grammar effectiveGrammar) Root() CommandID { return grammar.base.Root() }

func (grammar effectiveGrammar) Dialect(command CommandID) (dialect, bool) {
	if !grammar.hasProviderGrammar {
		return grammar.base.Dialect(command)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	if dialect, ok := providerGrammarLatest(grammar.state.grammar.dialects[command], grammar.frontier); ok {
		return dialect, true
	}
	return grammar.base.Dialect(command)
}

func (grammar effectiveGrammar) ProfileDialect(profile GrammarProfileID) (dialect, bool) {
	if !grammar.hasProviderGrammar {
		return grammar.base.ProfileDialect(profile)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	if dialect, ok := providerGrammarLatest(
		grammar.state.grammar.profileDialects[profile],
		grammar.frontier,
	); ok {
		return dialect, true
	}
	return grammar.base.ProfileDialect(profile)
}

func (grammar effectiveGrammar) Profiles(command CommandID) []GrammarProfileID {
	if !grammar.hasProviderGrammar {
		return grammar.base.Profiles(command)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	return providerGrammarList(
		grammar.base.Profiles(command),
		grammar.state.grammar.profiles[command],
		grammar.frontier,
	)
}

func (grammar effectiveGrammar) CommandEntries(parent CommandID) []commandEntryGrammar {
	if !grammar.hasProviderGrammar {
		return grammar.base.CommandEntries(parent)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	return providerGrammarList(
		grammar.base.CommandEntries(parent),
		grammar.state.grammar.commandEntries[parent],
		grammar.frontier,
	)
}

func (grammar effectiveGrammar) ActionCommands(parent CommandID) []actionCommandGrammar {
	if !grammar.hasProviderGrammar {
		return grammar.base.ActionCommands(parent)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	return providerGrammarList(
		grammar.base.ActionCommands(parent),
		grammar.state.grammar.actionCommands[parent],
		grammar.frontier,
	)
}

func (grammar effectiveGrammar) ActionFlags(command CommandID) []semanticActionFlagGrammar {
	if !grammar.hasProviderGrammar {
		return grammar.base.ActionFlags(command)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	return providerGrammarList(
		grammar.base.ActionFlags(command),
		grammar.state.grammar.actionFlags[command],
		grammar.frontier,
	)
}

func (grammar effectiveGrammar) Flags(command CommandID) []flagGrammar {
	if !grammar.hasProviderGrammar {
		return grammar.base.Flags(command)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	return providerGrammarList(
		grammar.base.Flags(command),
		grammar.state.grammar.flags[command],
		grammar.frontier,
	)
}

func (grammar effectiveGrammar) ArgumentForms(command CommandID) []argumentFormGrammar {
	if !grammar.hasProviderGrammar {
		return grammar.base.ArgumentForms(command)
	}
	grammar.state.mu.RLock()
	defer grammar.state.mu.RUnlock()
	return providerGrammarList(
		grammar.base.ArgumentForms(command),
		grammar.state.grammar.argumentForms[command],
		grammar.frontier,
	)
}
