package cli

type commandEntryProjectionState struct {
	lifecycle LifecycleState
	available bool
	visible   bool
}

type commandPathProjectionState struct {
	lifecycle ReleaseLifecycle
	visible   bool
}

func projectCommandEntryState(scheme ReleaseScheme, release string, visibility ReferenceVisibility, record commandEntryRecord, parent commandPathProjectionState, target *commandPathProjectionState) (commandEntryProjectionState, error) {
	parentState, err := referenceLifecycleState(scheme, releaseLifecycle(parent.lifecycle), release)
	if err != nil {
		return commandEntryProjectionState{}, err
	}

	lifecycle := record.lifecycle
	targetVisible := true
	if target != nil {
		lifecycle = intersectEffectiveReleaseLifecycles(scheme, lifecycle, target.lifecycle)
		targetVisible = target.visible
	}
	state, err := referenceLifecycleState(scheme, releaseLifecycle(lifecycle), release)
	if err != nil {
		return commandEntryProjectionState{}, err
	}
	available := referenceAvailable(parentState) && referenceAvailable(state)
	visible := available && parent.visible && targetVisible && (visibility == ReferenceVisibilityAll || !record.presentation.hidden)
	result := commandEntryProjectionState{
		lifecycle: state,
		available: available,
		visible:   visible,
	}
	return result, nil
}

func projectCommandPathState(lifecycle ReleaseLifecycle, visibility ReferenceVisibility, root commandRecord, records []commandRecord) commandPathProjectionState {
	result := commandPathProjectionState{
		lifecycle: lifecycle,
		visible:   true,
	}
	if visibility == ReferenceVisibilityPublic && root.presentation.hidden {
		result.visible = false
	}
	for _, record := range records {
		if visibility == ReferenceVisibilityPublic && record.presentation.hidden {
			result.visible = false
		}
	}
	return result
}

func effectiveCommandEntryProjection(model *effectiveModel, grammar grammarView, scheme ReleaseScheme, release string, visibility ReferenceVisibility, id CommandEntryID) (commandEntryProjectionState, error) {
	record, ok := model.CommandEntry(id)
	if !ok {
		return commandEntryProjectionState{}, errInvalidModelTopology
	}
	entry, ok := commandEntryProjectionGrammar(grammar, record.parent, id)
	if !ok {
		return commandEntryProjectionState{}, errInvalidModelTopology
	}

	parent, err := effectiveCommandPathProjection(model, grammar, scheme, record.parent, visibility)
	if err != nil {
		return commandEntryProjectionState{}, err
	}
	var target *commandPathProjectionState
	if entry.kind == CommandEntryAlias || entry.kind == CommandEntryShortcut {
		projected, err := effectiveCommandPathProjection(model, grammar, scheme, entry.target, visibility)
		if err != nil {
			return commandEntryProjectionState{}, err
		}
		target = &projected
	}
	return projectCommandEntryState(scheme, release, visibility, record, parent, target)
}

func effectiveCommandPathProjection(model *effectiveModel, grammar grammarView, scheme ReleaseScheme, command CommandID, visibility ReferenceVisibility) (commandPathProjectionState, error) {
	root := grammar.Root()
	if !root.present() {
		return commandPathProjectionState{}, errInvalidModelTopology
	}
	rootRecord, ok := model.Command(root)
	if !ok {
		return commandPathProjectionState{}, errInvalidModelTopology
	}
	path, err := canonicalCommandPath(model, grammar, root, command)
	if err != nil {
		return commandPathProjectionState{}, err
	}
	lifecycle, err := intersectCanonicalCommandPathLifecycle(model, scheme, path)
	if err != nil {
		return commandPathProjectionState{}, err
	}
	records := make([]commandRecord, 0, len(path))
	for _, step := range path {
		record, ok := model.Command(step.Command)
		if !ok {
			return commandPathProjectionState{}, errInvalidModelTopology
		}
		records = append(records, record)
	}
	return projectCommandPathState(lifecycle, visibility, rootRecord, records), nil
}

func commandEntryProjectionGrammar(grammar grammarView, parent CommandID, id CommandEntryID) (commandEntryGrammar, bool) {
	for _, entry := range grammar.CommandEntries(parent) {
		if entry.id == id {
			return entry, true
		}
	}
	return commandEntryGrammar{}, false
}

func commandPathNames(model *effectiveModel, command CommandID) []string {
	grammar, ok := model.EffectiveGrammar()
	if !ok {
		panic("cli: command path projection cannot resolve required declaration state")
	}
	path, err := canonicalCommandPath(model, grammar, grammar.Root(), command)
	if err != nil {
		panic("cli: command path projection cannot resolve required declaration state")
	}
	if len(path) == 0 {
		return nil
	}
	names := make([]string, len(path))
	parent := grammar.Root()
	for i, step := range path {
		entry, ok := commandEntryProjectionGrammar(grammar, parent, step.Entry)
		if !ok {
			panic("cli: command path projection cannot resolve required declaration state")
		}
		names[i] = entry.name
		parent = step.Command
	}
	return names
}
