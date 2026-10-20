package cli

// HelpGroupOrder selects the deterministic order used for entries within a help group.
type HelpGroupOrder uint8

const (
	// HelpGroupOrderDeclaration keeps help-group entries in declaration order.
	HelpGroupOrderDeclaration HelpGroupOrder = iota

	// HelpGroupOrderName orders help-group entries by name.
	HelpGroupOrderName
)

type helpGroupDeclaration struct {
	session *declarationSession
	group   helpGroupRef
}

func (declaration helpGroupDeclaration) OrderByName() helpGroupDeclaration {
	if declaration.session == nil || !declaration.group.valid() {
		return declaration
	}
	declaration.session.use(func(draft *declarationDraft, _ commandRef) {
		record, ok := draft.helpGroups.resolve(draft, declaration.group)
		if !ok {
			panic("cli: issued help-group draft cannot resolve")
		}
		record.order = HelpGroupOrderName
	})
	return declaration
}

func stageHelpGroup(session *declarationSession, profile grammarProfileCaseRef, current helpGroupRef, family helpGroupFamily, title string, callbackPresent bool) (helpGroupRef, bool) {
	if session == nil {
		return helpGroupRef{}, false
	}

	var ref helpGroupRef
	valid := false
	session.use(func(draft *declarationDraft, command commandRef) {
		valid = callbackPresent && family.Valid() && title != "" && !current.valid()
		record := helpGroupDraft{
			position: draft.nextPosition(),
			command:  command,
			profile:  profile,
			family:   family,
			title:    title,
			order:    HelpGroupOrderDeclaration,
			valid:    valid,
		}
		ref = draft.helpGroups.append(draft, record)
		if !valid {
			draft.reject(record.position, errInvalidAuthoringTransaction)
		}
	})
	return ref, valid
}

func declareSubcommandHelpGroup[C any](subcommands *Subcommands[C], title string, declare func(*Subcommands[C])) helpGroupDeclaration {
	if subcommands == nil || subcommands.session == nil {
		return helpGroupDeclaration{}
	}
	group, valid := stageHelpGroup(subcommands.session, subcommands.profile, subcommands.helpGroup, helpGroupFamilyCommand, title, declare != nil)
	if valid && declare != nil {
		grouped := *subcommands
		grouped.helpGroup = group
		declare(&grouped)
	}
	return helpGroupDeclaration{
		session: subcommands.session,
		group:   group,
	}
}

func declareFlagHelpGroup[C any](flags *Flags[C], title string, declare func(*Flags[C])) helpGroupDeclaration {
	if flags == nil || flags.session == nil {
		return helpGroupDeclaration{}
	}
	group, valid := stageHelpGroup(flags.session, flags.profile, flags.helpGroup, helpGroupFamilyFlag, title, declare != nil)
	if valid && declare != nil {
		grouped := *flags
		grouped.helpGroup = group
		declare(&grouped)
	}
	return helpGroupDeclaration{
		session: flags.session,
		group:   group,
	}
}

type helpGroupAcceptanceKey struct {
	command CommandID
	profile GrammarProfileID
	family  helpGroupFamily
	title   string
}

type helpGroupAcceptanceState struct {
	order    HelpGroupOrder
	position declarationPosition
	entries  []helpCommandGroupEntry
	flags    []FlagID
}

func publishHelpGroups(draft *declarationCandidate, order []helpGroupAcceptanceKey, groups map[helpGroupAcceptanceKey]*helpGroupAcceptanceState) {
	for _, key := range order {
		group := groups[key]
		if group == nil {
			panic("cli: accepted help group is unavailable")
		}
		switch key.family {
		case helpGroupFamilyCommand:
			input := helpCommandGroupInput{
				Command:  key.command,
				Profile:  key.profile,
				Title:    key.title,
				Order:    group.order,
				Entries:  group.entries,
				Position: group.position,
			}
			draft.HelpGroup(acceptHelpCommandGroupRecord(input))
		case helpGroupFamilyFlag:
			input := helpFlagGroupInput{
				Command:  key.command,
				Profile:  key.profile,
				Title:    key.title,
				Order:    group.order,
				Flags:    group.flags,
				Position: group.position,
			}
			draft.HelpGroup(acceptHelpFlagGroupRecord(input))
		default:
			panic("cli: accepted help group has invalid family")
		}
	}
}

func stageStaticHelpGroups(draft *declarationCandidate, input declarationPlan, source *declarationDraft, state *staticAcceptanceState) error {
	groups := make(map[helpGroupAcceptanceKey]*helpGroupAcceptanceState)
	groupOrder := make([]helpGroupAcceptanceKey, 0)
	for index := range len(input.records) {
		record := input.records[index]
		if record.kind != declarationRecordHelpGroup {
			continue
		}
		if !source.helpGroups.entries[record.familyIndex].value.valid {
			panic("cli: validated help group is invalid")
		}
		command := state.mustCommand(record.commandIndex)
		profileID := GrammarProfileID(0)
		if record.profileIndex >= 0 {
			profileID = state.mustProfile(record.profileIndex).ID()
		}
		key := helpGroupAcceptanceKey{
			command: command.ID(),
			profile: profileID,
			family:  source.helpGroups.entries[record.familyIndex].value.family,
			title:   source.helpGroups.entries[record.familyIndex].value.title,
		}
		state.slots[index].helpGroupKey = key
		state.slots[index].helpGroupSet = true
		if existing := groups[key]; existing != nil {
			if existing.order != source.helpGroups.entries[record.familyIndex].value.order {
				return errStaticDeclarationRejected
			}
			continue
		}
		groups[key] = &helpGroupAcceptanceState{
			order:    source.helpGroups.entries[record.familyIndex].value.order,
			position: input.positionFor(index),
		}
		groupOrder = append(groupOrder, key)
	}

	for index := range len(input.records) {
		record := input.records[index]
		if err := stageStaticCommandHelpGroupMembership(index, record, input, state, groups); err != nil {
			return err
		}
		if err := stageStaticFlagHelpGroupMembership(index, record, input, state, groups); err != nil {
			return err
		}
	}

	publishHelpGroups(draft, groupOrder, groups)
	return nil
}

func stageStaticCommandHelpGroupMembership(index int, record declarationPlanRecord, input declarationPlan, state *staticAcceptanceState, groups map[helpGroupAcceptanceKey]*helpGroupAcceptanceState) error {
	if record.commandHelpGroupIndex < 0 {
		return nil
	}
	groupIndex := record.commandHelpGroupIndex
	if groupIndex >= len(input.records) || !state.slots[groupIndex].helpGroupSet {
		panic("cli: command help-group coordinate is unavailable")
	}
	key := state.slots[groupIndex].helpGroupKey
	group := groups[key]
	if group == nil || key.family != helpGroupFamilyCommand {
		panic("cli: command help-group coordinate has invalid family")
	}
	switch record.kind {
	case declarationRecordCommand:
		entries := state.slots[index].commandEntries
		if len(entries) == 0 {
			panic("cli: accepted command has no help entry identity")
		}
		for _, entry := range entries {
			if !entry.ID().present() {
				panic("cli: accepted command help entry identity is unavailable")
			}
			group.entries = append(group.entries, acceptHelpCommandEntry(entry.ID()))
		}
	case declarationRecordProvider:
		provider := state.slots[index].provider
		if !provider.ID().present() {
			panic("cli: accepted provider identity is unavailable")
		}
		group.entries = append(group.entries, acceptHelpProviderEntry(provider.ID()))
	case declarationRecordHelpGroup:
	default:
		panic("cli: declaration cannot belong to a command help group")
	}
	return nil
}

func stageStaticFlagHelpGroupMembership(index int, record declarationPlanRecord, input declarationPlan, state *staticAcceptanceState, groups map[helpGroupAcceptanceKey]*helpGroupAcceptanceState) error {
	if record.flagHelpGroupIndex < 0 {
		return nil
	}
	groupIndex := record.flagHelpGroupIndex
	if groupIndex >= len(input.records) || !state.slots[groupIndex].helpGroupSet {
		panic("cli: flag help-group coordinate is unavailable")
	}
	key := state.slots[groupIndex].helpGroupKey
	group := groups[key]
	if group == nil || key.family != helpGroupFamilyFlag {
		panic("cli: flag help-group coordinate has invalid family")
	}
	switch record.kind {
	case declarationRecordFlag:
		flag := state.slots[index].flag
		if !flag.ID().present() {
			panic("cli: accepted flag identity is unavailable")
		}
		group.flags = append(group.flags, flag.ID())
	case declarationRecordFlagSpelling, declarationRecordFlagSpellingDirective, declarationRecordHelpGroup:
	default:
		panic("cli: declaration cannot belong to a flag help group")
	}
	return nil
}

type helpGroupFamily uint8

const (
	helpGroupFamilyUnknown helpGroupFamily = iota
	helpGroupFamilyCommand
	helpGroupFamilyFlag
)

func (family helpGroupFamily) Valid() bool {
	return family == helpGroupFamilyCommand || family == helpGroupFamilyFlag
}

func (order HelpGroupOrder) valid() bool {
	return order == HelpGroupOrderDeclaration || order == HelpGroupOrderName
}

type helpCommandGroupEntry struct {
	kind     HelpCommandGroupEntryKind
	entry    CommandEntryID
	provider CommandProviderID
}

func acceptHelpCommandEntry(entry CommandEntryID) helpCommandGroupEntry {
	if !entry.present() {
		panic("semantic: invalid help command-entry identity")
	}
	return helpCommandGroupEntry{
		kind:  HelpCommandGroupEntryCommandEntry,
		entry: entry,
	}
}

func acceptHelpProviderEntry(provider CommandProviderID) helpCommandGroupEntry {
	if !provider.present() {
		panic("semantic: invalid help provider identity")
	}
	return helpCommandGroupEntry{
		kind:     HelpCommandGroupEntryCommandProvider,
		provider: provider,
	}
}

func (entry helpCommandGroupEntry) Kind() HelpCommandGroupEntryKind { return entry.kind }

func (entry helpCommandGroupEntry) CommandEntry() CommandEntryID { return entry.entry }

func (entry helpCommandGroupEntry) Provider() CommandProviderID { return entry.provider }

func (entry helpCommandGroupEntry) Valid() bool {
	switch entry.kind {
	case HelpCommandGroupEntryCommandEntry:
		return entry.entry.present() && !entry.provider.present()
	case HelpCommandGroupEntryCommandProvider:
		return entry.provider.present() && !entry.entry.present()
	default:
		return false
	}
}

type helpGroupRecord struct {
	command        CommandID
	profile        GrammarProfileID
	family         helpGroupFamily
	title          string
	order          HelpGroupOrder
	commandEntries []helpCommandGroupEntry
	flags          []FlagID
	position       declarationPosition
}

// helpCommandGroupInput is the complete immutable payload of one command help group.
type helpCommandGroupInput struct {
	Command  CommandID
	Profile  GrammarProfileID
	Title    string
	Order    HelpGroupOrder
	Entries  []helpCommandGroupEntry
	Position declarationPosition
}

func acceptHelpCommandGroupRecord(input helpCommandGroupInput) helpGroupRecord {
	mustCommand(input.Command)
	record := helpGroupRecord{
		command:        input.Command,
		profile:        input.Profile,
		family:         helpGroupFamilyCommand,
		title:          input.Title,
		order:          input.Order,
		commandEntries: append([]helpCommandGroupEntry(nil), input.Entries...),
		position:       input.Position,
	}

	return record
}

// helpFlagGroupInput is the complete immutable payload of one flag help group.
type helpFlagGroupInput struct {
	Command  CommandID
	Profile  GrammarProfileID
	Title    string
	Order    HelpGroupOrder
	Flags    []FlagID
	Position declarationPosition
}

func acceptHelpFlagGroupRecord(input helpFlagGroupInput) helpGroupRecord {
	mustCommand(input.Command)
	record := helpGroupRecord{
		command:  input.Command,
		profile:  input.Profile,
		family:   helpGroupFamilyFlag,
		title:    input.Title,
		order:    input.Order,
		flags:    append([]FlagID(nil), input.Flags...),
		position: input.Position,
	}

	return record
}

func (record helpGroupRecord) Valid() bool {
	if !record.command.present() || !record.family.Valid() || !record.order.valid() || record.title == "" {
		return false
	}
	if record.family == helpGroupFamilyCommand {
		if len(record.flags) != 0 {
			return false
		}
		for _, entry := range record.commandEntries {
			if !entry.Valid() {
				return false
			}
		}
		return true
	}
	if len(record.commandEntries) != 0 {
		return false
	}
	for _, flag := range record.flags {
		if !flag.present() {
			return false
		}
	}
	return true
}

func (record helpGroupRecord) CommandEntries() []helpCommandGroupEntry {
	return append([]helpCommandGroupEntry(nil), record.commandEntries...)
}

func (record helpGroupRecord) Flags() []FlagID {
	return append([]FlagID(nil), record.flags...)
}
