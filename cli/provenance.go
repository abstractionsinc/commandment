package cli

import "slices"

// sourceRef is an operation-local capability reference to one accepted input source.
type sourceRef struct {
	operation *invocation
	id        InputSourceID
}

func (ref sourceRef) ID() InputSourceID { return ref.id }

// originRef is an operation-local capability reference to one accepted input origin.
type originRef struct {
	operation *invocation
	id        InputOriginID
}

func (ref originRef) ID() InputOriginID { return ref.id }

// spanRef keeps mutable-operation origin validation separate from the detached
// semantic ID stored after acceptance.
type spanRef struct {
	Source sourceRef
	Entry  int
	Start  int
	End    int
}

type invocationProvenance struct {
	sources              []inputSourceRecord
	origins              []inputOriginRecord
	argumentSource       sourceRef
	environmentSource    sourceRef
	argumentOrigins      []originRef
	environmentOrigins   []originRef
	responseIncludes     []responseIncludeObservation
	shortcutInputEntries map[InputSourceID][]string
}

func (operation *invocation) initializeInputSources() {
	operation.analysis.provenance.argumentSource = operation.addSource(InputKindArguments, false)
	operation.analysis.provenance.argumentOrigins = make([]originRef, len(operation.analysis.args))
	for index, argument := range operation.analysis.args {
		spanRefValue := spanRef{
			Source: operation.analysis.provenance.argumentSource,
			Entry:  index,
			Start:  0,
			End:    len(argument),
		}
		operation.analysis.provenance.argumentOrigins[index] = operation.addOrigin(spanRefValue, nil, false)
	}

	operation.analysis.provenance.environmentSource = operation.addSource(InputKindEnvironment, false)
	operation.analysis.provenance.environmentOrigins = make([]originRef, len(operation.input.environment))
	for index, entry := range operation.input.environment {
		// The prepared Env shape has structured Name/Value fields rather than
		// caller-owned raw NAME=VALUE storage. The full prepared element extent
		// is represented deterministically as len(name)+1+len(value).
		spanRefValue2 := spanRef{
			Source: operation.analysis.provenance.environmentSource,
			Entry:  index,
			Start:  0,
			End:    len(entry.Name) + 1 + len(entry.Value),
		}
		operation.analysis.provenance.environmentOrigins[index] = operation.addOrigin(spanRefValue2, nil, false)
	}
}

func (analysis *analysisState) addSource(operation *invocation, kind InputKind, sensitive bool) sourceRef {
	if analysis == nil || operation == nil {
		panic("invocation: invalid input source owner")
	}
	id := analysis.ids.AllocateInputSourceID()
	analysis.provenance.sources = append(analysis.provenance.sources, acceptInputSourceRecord(id, kind, sensitive))
	ref := sourceRef{
		operation: operation,
		id:        id,
	}
	return ref
}

func (operation *invocation) addSource(kind InputKind, sensitive bool) sourceRef {
	if operation == nil {
		panic("invocation: nil operation")
	}
	return operation.analysis.addSource(operation, kind, sensitive)
}

func (analysis *analysisState) addShortcutSource(operation *invocation, entry CommandEntryID, tokens []string, sensitive bool) sourceRef {
	if analysis == nil || operation == nil {
		panic("invocation: invalid shortcut source owner")
	}
	id := analysis.ids.AllocateInputSourceID()
	analysis.provenance.sources = append(analysis.provenance.sources, acceptShortcutInputSourceRecord(id, entry, sensitive))
	if analysis.provenance.shortcutInputEntries == nil {
		analysis.provenance.shortcutInputEntries = make(map[InputSourceID][]string)
	}
	analysis.provenance.shortcutInputEntries[id] = append([]string(nil), tokens...)
	ref := sourceRef{
		operation: operation,
		id:        id,
	}
	return ref
}

func (operation *invocation) addShortcutSource(entry CommandEntryID, tokens []string, sensitive bool) sourceRef {
	if operation == nil {
		panic("invocation: nil operation")
	}
	return operation.analysis.addShortcutSource(operation, entry, tokens, sensitive)
}

func (analysis *analysisState) addResponseSource(operation *invocation, key, revision, location, includeBase string, sensitive bool) sourceRef {
	if analysis == nil || operation == nil {
		panic("invocation: invalid response source owner")
	}
	id := analysis.ids.AllocateInputSourceID()
	input := responseInputSourceInput{
		ID:          id,
		Key:         key,
		Revision:    revision,
		Location:    location,
		IncludeBase: includeBase,
		Sensitive:   sensitive,
	}
	analysis.provenance.sources = append(analysis.provenance.sources, acceptResponseInputSourceRecord(input))
	ref := sourceRef{
		operation: operation,
		id:        id,
	}
	return ref
}

func (analysis *analysisState) addOrigin(operation *invocation, span spanRef, parents []originRef, sensitive bool) originRef {
	if analysis == nil || operation == nil || span.Source.operation != operation || !span.Source.id.present() {
		panic("invocation: foreign or unknown input source")
	}
	semanticParents := make([]InputOriginID, len(parents))
	for index, parent := range parents {
		if parent.operation != operation || !parent.id.present() || !analysis.hasOrigin(parent.id) {
			panic("invocation: foreign or unknown input origin parent")
		}
		semanticParents[index] = parent.id
	}
	id := analysis.ids.AllocateInputOriginID()
	semanticSpan := semanticInputSpan{
		Source: span.Source.id,
		Entry:  span.Entry,
		Start:  span.Start,
		End:    span.End,
	}
	record := acceptInputOriginRecord(id, semanticSpan, semanticParents, sensitive)
	analysis.provenance.origins = append(analysis.provenance.origins, record)
	ref := originRef{
		operation: operation,
		id:        id,
	}
	return ref
}

func (operation *invocation) addOrigin(span spanRef, parents []originRef, sensitive bool) originRef {
	if operation == nil {
		panic("invocation: nil operation")
	}
	return operation.analysis.addOrigin(operation, span, parents, sensitive)
}

func (analysis *analysisState) originRecord(id InputOriginID) (inputOriginRecord, bool) {
	if analysis == nil || !id.present() {
		return inputOriginRecord{}, false
	}
	position := int(id.uint32()) - 1
	if position < 0 || position >= len(analysis.provenance.origins) {
		return inputOriginRecord{}, false
	}
	record := analysis.provenance.origins[position]
	if record.id != id {
		return inputOriginRecord{}, false
	}
	return record, true
}

func (operation *invocation) originRecord(id InputOriginID) (inputOriginRecord, bool) {
	if operation == nil {
		return inputOriginRecord{}, false
	}
	return operation.analysis.originRecord(id)
}

func (analysis *analysisState) hasOrigin(id InputOriginID) bool {
	_, ok := analysis.originRecord(id)
	return ok
}

func (analysis *analysisState) originSensitive(id InputOriginID) bool {
	record, ok := analysis.originRecord(id)
	return ok && record.sensitive
}

func (operation *invocation) originSensitive(id InputOriginID) bool {
	if operation == nil {
		return false
	}
	return operation.analysis.originSensitive(id)
}

func (analysis *analysisState) originsSensitive(ids []InputOriginID) bool {
	return slices.ContainsFunc(ids, analysis.originSensitive)
}

func (analysis *analysisState) commonOriginSource(ids []InputOriginID) InputSourceID {
	var source InputSourceID
	for _, id := range ids {
		record, ok := analysis.originRecord(id)
		if !ok {
			return InputSourceID{}
		}
		if !source.present() {
			source = record.span.Source
			continue
		}
		if source != record.span.Source {
			return InputSourceID{}
		}
	}
	return source
}

func (analysis *analysisState) sourcesSnapshot() []inputSourceRecord {
	if analysis == nil {
		return nil
	}
	return append([]inputSourceRecord(nil), analysis.provenance.sources...)
}

func (operation *invocation) Sources() []inputSourceRecord {
	if operation == nil {
		return nil
	}
	return operation.analysis.sourcesSnapshot()
}

func (analysis *analysisState) originsSnapshot() []inputOriginRecord {
	if analysis == nil {
		return nil
	}
	return append([]inputOriginRecord(nil), analysis.provenance.origins...)
}

func (operation *invocation) Origins() []inputOriginRecord {
	if operation == nil {
		return nil
	}
	return operation.analysis.originsSnapshot()
}

func (operation *invocation) ArgumentSource() sourceRef {
	return operation.analysis.provenance.argumentSource
}

func (operation *invocation) EnvironmentSource() sourceRef {
	return operation.analysis.provenance.environmentSource
}

func (analysis *analysisState) argumentOrigin(index int) (originRef, bool) {
	if analysis == nil || index < 0 || index >= len(analysis.provenance.argumentOrigins) {
		return originRef{}, false
	}
	return analysis.provenance.argumentOrigins[index], true
}

func (operation *invocation) ArgumentOrigin(index int) (originRef, bool) {
	if operation == nil {
		return originRef{}, false
	}
	return operation.analysis.argumentOrigin(index)
}

func (analysis *analysisState) environmentOrigin(index int) (originRef, bool) {
	if analysis == nil || index < 0 || index >= len(analysis.provenance.environmentOrigins) {
		return originRef{}, false
	}
	return analysis.provenance.environmentOrigins[index], true
}

func (operation *invocation) EnvironmentOrigin(index int) (originRef, bool) {
	if operation == nil {
		return originRef{}, false
	}
	return operation.analysis.environmentOrigin(index)
}

func (operation *invocation) Origin(id InputOriginID) (originRef, bool) {
	if _, ok := operation.originRecord(id); !ok {
		return originRef{}, false
	}
	ref := originRef{
		operation: operation,
		id:        id,
	}
	return ref, true
}

// AddDerivedOrigin records one generated provenance fragment from already
// accepted parents. Requiring committed parents at insertion makes ancestry a
// DAG by construction while preserving parent order.
func (operation *invocation) AddDerivedOrigin(span spanRef, parents []originRef, sensitive bool) originRef {
	if operation == nil {
		panic("invocation: nil operation")
	}
	return operation.addOrigin(span, append([]originRef(nil), parents...), sensitive)
}

// InputSourceKind classifies one operation-local input container. This is
// private semantic vocabulary; public projection is owned downstream.

// inputSourceRecord is one immutable operation-local source snapshot identity.
type inputSourceRecord struct {
	id              InputSourceID
	kind            InputKind
	revision        string
	key             string
	location        string
	includeBase     string
	commandShortcut CommandEntryID
	sensitive       bool
}

func acceptInputSourceRecord(id InputSourceID, kind InputKind, sensitive bool) inputSourceRecord {
	if !id.present() || kind == InputKindUnknown || kind == InputKindShortcut || kind == InputKindResponseFile {
		panic("semantic: invalid input source record")
	}
	record := inputSourceRecord{
		id:        id,
		kind:      kind,
		sensitive: sensitive,
	}

	return record
}

func acceptShortcutInputSourceRecord(id InputSourceID, shortcut CommandEntryID, sensitive bool) inputSourceRecord {
	if !id.present() || !shortcut.present() {
		panic("semantic: invalid shortcut input source record")
	}
	record := inputSourceRecord{
		id:              id,
		kind:            InputKindShortcut,
		commandShortcut: shortcut,
		sensitive:       sensitive,
	}

	return record
}

// responseInputSourceInput is the complete immutable payload of one response-file input source.
type responseInputSourceInput struct {
	ID          InputSourceID
	Key         string
	Revision    string
	Location    string
	IncludeBase string
	Sensitive   bool
}

func acceptResponseInputSourceRecord(input responseInputSourceInput) inputSourceRecord {
	if !input.ID.present() || input.Key == "" || input.Revision == "" {
		panic("semantic: invalid response input source record")
	}
	record := inputSourceRecord{
		id:          input.ID,
		kind:        InputKindResponseFile,
		revision:    input.Revision,
		key:         input.Key,
		location:    input.Location,
		includeBase: input.IncludeBase,
		sensitive:   input.Sensitive,
	}

	return record
}

// semanticInputSpan identifies one half-open byte range in one operation-local input
// source. Entry -1 denotes a flat textual source; nonnegative entries identify
// one element in an element-oriented source.
type semanticInputSpan struct {
	Source InputSourceID
	Entry  int
	Start  int
	End    int
}

func (span semanticInputSpan) Valid() bool {
	return span.Source.present() && span.Entry >= -1 && span.Start >= 0 && span.End >= span.Start
}

// inputOriginRecord is one immutable provenance fragment. Parent order is
// semantic generator order and is snapshotted on construction.
type inputOriginRecord struct {
	id        InputOriginID
	span      semanticInputSpan
	parents   []InputOriginID
	sensitive bool
}

func acceptInputOriginRecord(id InputOriginID, span semanticInputSpan, parents []InputOriginID, sensitive bool) inputOriginRecord {
	if !id.present() || !span.Valid() {
		panic("semantic: invalid input origin record")
	}
	seen := make(map[InputOriginID]struct{}, len(parents))
	for _, parent := range parents {
		if !parent.present() || parent == id {
			panic("semantic: invalid input origin parent")
		}
		if _, exists := seen[parent]; exists {
			panic("semantic: duplicate input origin parent")
		}
		seen[parent] = struct{}{}
	}
	record := inputOriginRecord{
		id:        id,
		span:      span,
		parents:   append([]InputOriginID(nil), parents...),
		sensitive: sensitive,
	}

	return record
}

func (record inputOriginRecord) Parents() []InputOriginID {
	return append([]InputOriginID(nil), record.parents...)
}
