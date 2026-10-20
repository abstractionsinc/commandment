package shelltest

import (
	"slices"
	"sync"
)

type observationState struct {
	mu sync.Mutex

	active     bool
	activeUses int

	version     string
	screenLines []string
}

// Version returns the verified runtime version for this settled observation.
func (observation Observation) Version() string {
	state := observation.acquire()
	version := state.version
	observation.release(state)
	return version
}

// EditorInput returns the settled editor buffer when the engine can establish one truthfully. The current engine has no independent settled-editor-buffer oracle.
func (observation Observation) EditorInput() (Input, bool) {
	state := observation.acquire()
	observation.release(state)
	return Input{}, false
}

// ScreenLines returns a defensive copy of normalized settled semantic screen lines. Nil means unavailable; nonnil empty means an observed empty screen.
func (observation Observation) ScreenLines() []string {
	state := observation.acquire()
	lines := slices.Clone(state.screenLines)
	observation.release(state)
	return lines
}

func (observation Observation) acquire() *observationState {
	if observation.state == nil {
		panic("shelltest: invalid Observation")
	}
	state := observation.state
	state.mu.Lock()
	if !state.active {
		state.mu.Unlock()
		panic("shelltest: Observation used outside Interpret callback")
	}
	state.activeUses++
	state.mu.Unlock()
	return state
}

func (Observation) release(state *observationState) {
	state.mu.Lock()
	state.activeUses--
	state.mu.Unlock()
}

func (state *observationState) expire() bool {
	state.mu.Lock()
	state.active = false
	crossedBoundary := state.activeUses != 0
	state.mu.Unlock()
	return crossedBoundary
}
