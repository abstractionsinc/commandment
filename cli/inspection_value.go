package cli

import (
	"cmp"
	"slices"
	"strconv"
)

type inspectionCandidateProjection struct {
	owner *analysisState
	ids   map[valueCandidateRef]ValueCandidateID
}

func prepareInspectionCandidateProjection(analysis valueAnalysis) inspectionCandidateProjection {
	refs := make([]valueCandidateRef, 0, len(analysis.Candidates))
	var owner *analysisState
	for ref, candidate := range analysis.Candidates {
		if !ref.valid() || candidate.Meta.ID != ref {
			panic("cli: invalid value candidate analysis reference")
		}
		if owner == nil {
			owner = ref.owner
		} else if owner != ref.owner {
			panic("cli: value candidate analysis crosses snapshots")
		}
		refs = append(refs, ref)
	}
	slices.SortFunc(refs, func(left, right valueCandidateRef) int { return cmp.Compare(left.index, right.index) })
	projection := inspectionCandidateProjection{
		owner: owner,
		ids:   make(map[valueCandidateRef]ValueCandidateID, len(refs)),
	}
	for index, ref := range refs {
		if index > 0 && refs[index-1].index == ref.index {
			panic("cli: duplicate value candidate analysis reference")
		}
		projection.ids[ref] = ValueCandidateID(index + 1)
	}
	return projection
}

func (projection inspectionCandidateProjection) id(ref valueCandidateRef) ValueCandidateID {
	if !ref.valid() || projection.owner == nil || ref.owner != projection.owner {
		panic("cli: invalid inspection value candidate reference")
	}
	id, ok := projection.ids[ref]
	if !ok {
		panic("cli: inspection value candidate reference cannot resolve")
	}
	return id
}

// projectValueAnalysis detaches the canonical operation-local resolution graph
// into the frozen public Inspection representation. It never retains typed
// application storage or executable callbacks. A false return means the
// private operation knows that some selected value fact cannot yet be
// represented exactly by this projection; callers must preserve Incomplete
// rather than manufacturing Unknown state.
func projectValueAnalysis(result *Inspection, model *effectiveModel, analysis valueAnalysis) bool {
	projection := prepareInspectionCandidateProjection(analysis)
	complete := !analysis.Unsupported
	for destination, resolved := range analysis.Destinations {
		value, present, projected := projectDestinationValue(model, destination, resolved, projection.id)
		if !present {
			complete = false
			continue
		}
		if !projected {
			complete = false
		}
		result.Values[value.ID] = value
	}

	for candidateID, observed := range analysis.Candidates {
		candidate, ok := projectObservedValueCandidate(model, observed, projection.id)
		if !ok {
			complete = false
			continue
		}
		result.ValueCandidates[projection.id(candidateID)] = candidate
	}
	return complete
}

// projectDestinationValue owns the detached public representation of exactly one
// resolved destination. present reports whether enough semantic declaration
// state exists to name the value; complete reports whether every observable
// value fact could be represented without approximation.
func projectDestinationValue(model *effectiveModel, destination ValueID, resolved destinationResolution, candidateID func(valueCandidateRef) ValueCandidateID) (ValueInspection, bool, bool) {
	record, ok := model.Destination(destination)
	if !ok {
		return ValueInspection{}, false, false
	}
	definition, ok := record.Definition()
	if !ok {
		return ValueInspection{}, false, false
	}
	kind, ok := projectValueDestinationKind(definition.Kind())
	if !ok {
		return ValueInspection{}, false, false
	}
	semanticType, hasType := definition.ValueType()
	value := ValueInspection{
		ID:        destination,
		Kind:      kind,
		HasType:   hasType,
		State:     publicResolutionState(resolved.State),
		Sensitive: record.sensitive || resolved.Sensitive,
		BlockedBy: publicDestinationIDs(resolved.BlockedBy),
	}
	if candidateID != nil {
		value.Candidates = publicValueCandidateIDs(resolved.Candidates, candidateID)
		value.Contributors = publicValueCandidateIDs(resolved.Contributors, candidateID)
	}
	if hasType {
		value.Type = semanticType
	}
	for _, producer := range resolved.Producers {
		value.Producers = append(value.Producers, projectValueProducer(producer))
	}
	complete := true
	if resolved.State != destinationStateResolved {
		return value, true, true
	}

	switch definition.Kind() {
	case destinationKindScalar,
		destinationKindOpaque,
		destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		representation, ok := projectCanonicalValueRepresentation(definition, resolved.Value, value.Sensitive)
		if !ok {
			complete = false
		} else {
			value.Representation = representation
		}
	case destinationKindSequence:
		if candidateID != nil {
			for _, ref := range resolved.Elements {
				value.Elements = append(value.Elements, ValueElementReference{
					Candidate: candidateID(ref.Candidate),
					Index:     ref.Index,
				})
			}
		}
	case destinationKindKeyed:
		if candidateID != nil {
			for _, ref := range resolved.Entries {
				value.Entries = append(value.Entries, ValueEntryReference{
					Candidate: candidateID(ref.Candidate),
					Index:     ref.Index,
				})
			}
		}
	default:
		complete = false
	}
	return value, true, complete
}

func projectObservedValueCandidate(model *effectiveModel, observed valueCandidateObservation, candidateID func(valueCandidateRef) ValueCandidateID) (ValueCandidateInspection, bool) {
	record, ok := model.Destination(observed.Meta.Destination)
	if !ok {
		return ValueCandidateInspection{}, false
	}
	definition, ok := record.Definition()
	if !ok {
		return ValueCandidateInspection{}, false
	}
	return projectValueCandidate(record, definition, observed, candidateID)
}

func projectObservedValueRepresentation(definition destinationDefinition, observed valueCandidateObservation, sensitive bool) (*ValueRepresentation, bool) {
	if !observed.HasTokenRepresentation {
		return projectCanonicalValueRepresentation(definition, observed.Value, sensitive)
	}
	if sensitive {
		representation := &ValueRepresentation{
			Kind:     ValueRepresentationTokens,
			Redacted: true,
		}
		return representation, true
	}
	representation := &ValueRepresentation{
		Kind:   ValueRepresentationTokens,
		Tokens: append([]string(nil), observed.TokenRepresentation...),
	}
	return representation, true
}

func projectValueProducer(producer producerObservation) ValueProducerInspection {
	valueProducerInspection := ValueProducerInspection{
		Kind:         producer.ProducerKind,
		Tier:         producer.Tier,
		Outcome:      publicProducerOutcome(producer.Outcome),
		Sensitive:    producer.Sensitive,
		Origins:      publicOriginIDs(producer.Origins),
		Dependencies: publicDestinationIDs(producer.Dependencies),
		BlockedBy:    publicDestinationIDs(producer.BlockedBy),
	}

	return valueProducerInspection
}

func projectValueCandidate(record destinationRecord, definition destinationDefinition, observed valueCandidateObservation, candidateID func(valueCandidateRef) ValueCandidateID) (ValueCandidateInspection, bool) {
	meta := observed.Meta
	sensitive := record.sensitive || meta.Sensitive
	candidate := ValueCandidateInspection{
		Destination:   meta.Destination,
		ProducerIndex: observed.ProducerIndex,
		Disposition:   meta.Disposition,
		Contribution:  publicCandidateContribution(meta.Shape),
		Declaration:   publicCandidateDeclaration(observed.Declaration),
		Origins:       publicOriginIDs(meta.Origins),
		Sensitive:     sensitive,
		Suppression:   meta.Suppression,
	}
	if candidateID != nil {
		candidate.ID = candidateID(meta.ID)
		candidate.SuppressedBy = publicValueCandidateIDs(meta.SuppressedBy, candidateID)
		candidate.ConflictsWith = publicValueCandidateIDs(meta.ConflictsWith, candidateID)
	}

	switch definition.Kind() {
	case destinationKindScalar,
		destinationKindOpaque,
		destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		if observed.HasInput {
			candidate.Input = projectValueInput(observed.Input, sensitive)
		}
		if observed.HasValue {
			representation, ok := projectObservedValueRepresentation(definition, observed, sensitive)
			if !ok {
				return ValueCandidateInspection{}, false
			}
			candidate.Representation = representation
		}
	case destinationKindSequence:
		language, ok := definition.Language()
		if !ok {
			return ValueCandidateInspection{}, false
		}
		for _, element := range observed.Elements {
			component, ok := projectValueComponent(language, element, record.sensitive || element.Sensitive)
			if !ok {
				return ValueCandidateInspection{}, false
			}
			candidate.Elements = append(candidate.Elements, component)
		}
	case destinationKindKeyed:
		language, ok := definition.KeyedLanguage()
		if !ok {
			return ValueCandidateInspection{}, false
		}
		for _, observedEntry := range observed.Entries {
			key, ok := projectValueComponent(
				language.Key(),
				observedEntry.Key,
				record.sensitive || observedEntry.Key.Sensitive,
			)
			if !ok {
				return ValueCandidateInspection{}, false
			}
			value, ok := projectValueComponent(
				language.Value(),
				observedEntry.Value,
				record.sensitive || observedEntry.Value.Sensitive,
			)
			if !ok {
				return ValueCandidateInspection{}, false
			}
			candidate.Entries = append(candidate.Entries, ValueEntryInspection{
				Key:   key,
				Value: value,
			})
		}
	default:
		return ValueCandidateInspection{}, false
	}
	return candidate, true
}

func projectValueComponent(language valueLanguage, observed valueCandidateComponentObservation, sensitive bool) (ValueComponentInspection, bool) {
	component := ValueComponentInspection{
		Origins:   publicOriginIDs(observed.Origins),
		Sensitive: sensitive,
	}
	if observed.HasInput {
		component.Input = projectValueInput(observed.Input, sensitive)
	}
	if observed.HasValue {
		if sensitive {
			component.Representation = &ValueRepresentation{
				Kind:     ValueRepresentationText,
				Redacted: true,
			}
			return component, true
		}
		text, err := language.Format(observed.Value)
		if err != nil {
			return ValueComponentInspection{}, false
		}
		component.Representation = &ValueRepresentation{
			Kind: ValueRepresentationText,
			Text: text,
		}
	}
	return component, true
}

func projectCanonicalValueRepresentation(definition destinationDefinition, value any, sensitive bool) (*ValueRepresentation, bool) {
	if sensitive {
		representation := &ValueRepresentation{
			Kind:     ValueRepresentationText,
			Redacted: true,
		}
		return representation, true
	}
	if language, ok := definition.Language(); ok {
		text, err := language.Format(value)
		if err != nil {
			return nil, false
		}
		representation := &ValueRepresentation{
			Kind: ValueRepresentationText,
			Text: text,
		}
		return representation, true
	}
	if valueType, ok := definition.ValueType(); ok && valueType.Kind == ValueKindCount {
		count, ok := value.(int)
		if !ok {
			return nil, false
		}
		representation := &ValueRepresentation{
			Kind: ValueRepresentationText,
			Text: strconv.Itoa(count),
		}
		return representation, true
	}
	// Opaque, derived, and fact destinations can carry typed runtime values while
	// deliberately owning no public CLI value language. Absence of a detached
	// representation is complete semantic truth, not a projection failure.
	switch definition.Kind() {
	case destinationKindOpaque,
		destinationKindDerived,
		destinationKindOptionalDerived,
		destinationKindFactRequired,
		destinationKindFactResolved:
		return nil, true
	default:
		return nil, false
	}
}

func projectValueInput(text string, sensitive bool) *ValueInputInspection {
	if sensitive {
		inspection := &ValueInputInspection{
			Redacted: true,
		}
		return inspection
	}
	inspection := &ValueInputInspection{
		Text: text,
	}
	return inspection
}

func publicCandidateDeclaration(declaration valueCandidateDeclaration) DeclarationIdentity {
	switch {
	case declaration.FlagSpelling.present():
		identity := DeclarationIdentity{
			Kind:         DeclarationKindFlagSpelling,
			FlagSpelling: declaration.FlagSpelling,
		}
		return identity
	case declaration.Flag.present():
		identity := DeclarationIdentity{
			Kind: DeclarationKindFlag,
			Flag: declaration.Flag,
		}
		return identity
	case declaration.ArgumentTerm.present():
		identity := DeclarationIdentity{
			Kind:         DeclarationKindArgumentTerm,
			ArgumentTerm: declaration.ArgumentTerm,
		}
		return identity
	case declaration.EnvironmentInput.present():
		identity := DeclarationIdentity{
			Kind:             DeclarationKindEnvironmentInput,
			EnvironmentInput: declaration.EnvironmentInput,
		}
		return identity
	default:
		return DeclarationIdentity{}
	}
}

func publicProducerOutcome(outcome producerOutcome) ValueProducerOutcome {
	switch outcome {
	case producerOutcomeInactive:
		return ValueProducerOutcomeInactive
	case producerOutcomeSkipped:
		return ValueProducerOutcomeSkipped
	case producerOutcomeOmitted:
		return ValueProducerOutcomeOmitted
	case producerOutcomeContributed:
		return ValueProducerOutcomeContributed
	case producerOutcomeFailed:
		return ValueProducerOutcomeFailed
	case producerOutcomeBlocked:
		return ValueProducerOutcomeBlocked
	default:
		return ValueProducerOutcomeUnknown
	}
}

func publicResolutionState(state analysisDestinationState) ResolutionState {
	switch state {
	case destinationStateInactive:
		return ResolutionStateInactive
	case destinationStateAbsent:
		return ResolutionStateAbsent
	case destinationStateResolved:
		return ResolutionStateResolved
	case destinationStateFailed:
		return ResolutionStateFailed
	case destinationStateBlocked:
		return ResolutionStateBlocked
	default:
		return ResolutionStateUnknown
	}
}

func publicCandidateContribution(shape contributionShape) ValueCandidateContribution {
	switch shape {
	case contributionShapeWhole:
		return ValueCandidateContributionWhole
	case contributionShapeSequenceSegment:
		return ValueCandidateContributionSequenceSegment
	case contributionShapeKeyedEntry:
		return ValueCandidateContributionKeyedEntry
	case contributionShapeKeyedReplacement:
		return ValueCandidateContributionKeyedReplacement
	default:
		return ValueCandidateContributionUnknown
	}
}

func publicValueCandidateIDs(refs []valueCandidateRef, project func(valueCandidateRef) ValueCandidateID) []ValueCandidateID {
	if project == nil {
		return nil
	}
	result := make([]ValueCandidateID, len(refs))
	for i, ref := range refs {
		result[i] = project(ref)
	}
	return result
}

func publicDestinationIDs(ids []ValueID) []ValueID {
	return slices.Clone(ids)
}

func publicOriginIDs(ids []InputOriginID) []InputOriginID {
	return slices.Clone(ids)
}

func applyValueDiagnostics(result *Inspection, model *effectiveModel, analysis valueAnalysis) []Diagnostic {
	projection := prepareInspectionCandidateProjection(analysis)
	var diagnostics []Diagnostic
	for _, issue := range analysis.Issues {
		if issue.Kind == valueIssueUnsupported || issue.Kind == valueIssueProducerBlocked {
			continue
		}
		diagnostic, candidateRef := publicValueIssueDiagnostic(model, analysis, issue)
		index := len(result.Diagnostics) + len(diagnostics)
		diagnostics = append(diagnostics, diagnostic)
		valueID := issue.Destination
		if value, ok := result.Values[valueID]; ok {
			value.DiagnosticIndexes = append(value.DiagnosticIndexes, index)
			result.Values[valueID] = value
		}
		if candidateRef.valid() {
			attachValueCandidateDiagnostic(result, valueID, projection.id(candidateRef), index)
		}
	}
	return diagnostics
}

func attachValueCandidateDiagnostic(result *Inspection, valueID ValueID, candidateID ValueCandidateID, index int) {
	if candidateID == 0 {
		return
	}
	candidate, ok := result.ValueCandidates[candidateID]
	if !ok {
		return
	}
	candidate.DiagnosticIndexes = append(candidate.DiagnosticIndexes, index)
	result.ValueCandidates[candidateID] = candidate
	if candidate.ProducerIndex < 0 {
		return
	}
	value, ok := result.Values[valueID]
	if !ok || candidate.ProducerIndex >= len(value.Producers) {
		return
	}
	value.Producers[candidate.ProducerIndex].DiagnosticIndexes = append(
		value.Producers[candidate.ProducerIndex].DiagnosticIndexes,
		index,
	)
	result.Values[valueID] = value
}

func publicValueIssueDiagnostic(model *effectiveModel, analysis valueAnalysis, issue valueIssue) (Diagnostic, valueCandidateRef) {
	record, _ := model.Destination(issue.Destination)
	diagnostic := Diagnostic{
		Code:    DiagnosticInvalidValue,
		Command: record.command,
		Origins: publicOriginIDs(issue.Origins),
	}
	candidate, candidateRef, found := rejectedValueCandidate(analysis, issue.Destination)
	applyRejectedValueCandidateDiagnostic(&diagnostic, record, candidate, found)

	switch issue.Kind {
	case valueIssueInvalidInput:
		diagnostic.Code = classifyInputFailure(record, candidate, found, issue.Err)
		if found {
			if entry, state, ok := rejectedDomainLifecycleEntry(model, issue.Destination, candidate); ok {
				switch state {
				case LifecycleStateNotIntroduced:
					diagnostic.Code = DiagnosticNotIntroducedValue
				case LifecycleStateRemoved:
					diagnostic.Code = DiagnosticRemovedValue
				case LifecycleStateUnknown,
					LifecycleStateCurrent,
					LifecycleStateDeprecated:
				}
				diagnostic.Declaration = DeclarationIdentity{
					Kind:             DeclarationKindValueDomainEntry,
					ValueDomainEntry: entry.meta.id,
				}
			}
		}
	case valueIssueResolutionConflict:
		diagnostic.Code = DiagnosticInvalidValue
	case valueIssueProducerFailure:
		definition, _ := record.Definition()
		switch definition.Kind() {
		case destinationKindFactRequired, destinationKindFactResolved:
			diagnostic.Code = DiagnosticFactResolutionFailure
		default:
			diagnostic.Code = DiagnosticValueComputationFailure
		}
	case valueIssueNone, valueIssueProducerBlocked, valueIssueUnsupported:
	}
	return diagnostic, candidateRef
}

func applyRejectedValueCandidateDiagnostic(diagnostic *Diagnostic, record destinationRecord, candidate valueCandidateObservation, found bool) {
	if !found {
		return
	}
	diagnostic.Declaration = publicCandidateDeclaration(candidate.Declaration)
	diagnostic.InputSource = candidate.InputSource
	diagnostic.ProducerKind = candidate.Meta.ProducerKind
	diagnostic.Tier = candidate.Meta.Tier
	diagnostic.Disposition = candidate.Meta.Disposition
	if len(diagnostic.Origins) == 0 {
		diagnostic.Origins = publicOriginIDs(candidate.Meta.Origins)
	}
	input, ok := primaryRejectedInput(candidate)
	if !ok {
		return
	}
	sensitive := record.sensitive || candidate.Meta.Sensitive
	if sensitive {
		diagnostic.InputRedacted = true
		diagnostic.Sensitive = true
		return
	}
	diagnostic.Input = input
}

func rejectedDomainLifecycleEntry(model *effectiveModel, destination ValueID, candidate valueCandidateObservation) (domainEntryRecord, LifecycleState, bool) {
	for _, index := range model.ValueDomainIndexesForDestination(destination) {
		domain, ok := model.ValueDomain(index)
		if !ok {
			continue
		}

		for _, use := range domain.Uses() {
			if use.destination != destination {
				continue
			}
			inputs := rejectedDomainInputs(candidate, use.kind)
			if len(inputs) == 0 {
				continue
			}
			for _, id := range domain.Entries() {
				entry, ok := model.DomainEntry(id)
				if !ok || !slices.Contains(inputs, entry.name) {
					continue
				}
				state, err := evaluateLifecycle(model.Application().release, entry.lifecycle, "")
				if err != nil {
					continue
				}
				if state == LifecycleStateNotIntroduced || state == LifecycleStateRemoved {
					return entry, state, true
				}
			}
		}
	}
	return domainEntryRecord{}, LifecycleStateUnknown, false
}

func rejectedDomainInputs(candidate valueCandidateObservation, kind ValueDomainUseKind) []string {
	var result []string
	switch kind {
	case ValueDomainUseDestination:
		if candidate.HasInput && !candidate.HasValue {
			result = append(result, candidate.Input)
		}
	case ValueDomainUseSequenceElement:
		for _, element := range candidate.Elements {
			if element.HasInput && !element.HasValue {
				result = append(result, element.Input)
			}
		}
	case ValueDomainUseMapKey:
		for _, entry := range candidate.Entries {
			if entry.Key.HasInput && !entry.Key.HasValue {
				result = append(result, entry.Key.Input)
			}
		}
	case ValueDomainUseMapValue:
		for _, entry := range candidate.Entries {
			if entry.Value.HasInput && !entry.Value.HasValue {
				result = append(result, entry.Value.Input)
			}
		}
	case ValueDomainUseUnknown:
	}
	return result
}

func classifyInputFailure(record destinationRecord, candidate valueCandidateObservation, found bool, err error) DiagnosticCode {
	definition, ok := record.Definition()
	if !ok {
		return DiagnosticInputParserFailure
	}
	var kind valueLanguageKind
	switch definition.Kind() {
	case destinationKindScalar, destinationKindSequence:
		language, ok := definition.Language()
		if !ok {
			return DiagnosticInputParserFailure
		}
		kind = language.Kind()
	case destinationKindKeyed:
		// Keyed failures can originate in either component. Both component
		// languages are callback-capable, so classify by structured rejection
		// evidence below rather than guessing which callback failed.
		kind = valueLanguageKindCodec
	default:
		return DiagnosticInvalidValue
	}
	if kind == valueLanguageKindBuiltin || kind == valueLanguageKindDomain {
		return DiagnosticInvalidValue
	}
	if found {
		if input, ok := primaryRejectedInput(candidate); ok {
			if collectInputRejections(err, []string{input}) {
				return DiagnosticInvalidValue
			}
		}
	}
	return DiagnosticInputParserFailure
}

func rejectedValueCandidate(analysis valueAnalysis, destination ValueID) (valueCandidateObservation, valueCandidateRef, bool) {
	for id, candidate := range analysis.Candidates {
		if candidate.Meta.Destination == destination &&
			candidate.Meta.Disposition == ValueCandidateDispositionRejected {
			return candidate, id, true
		}
	}
	return valueCandidateObservation{}, valueCandidateRef{}, false
}

func primaryRejectedInput(candidate valueCandidateObservation) (string, bool) {
	if candidate.HasInput {
		return candidate.Input, true
	}
	for _, element := range candidate.Elements {
		if element.HasInput && !element.HasValue {
			return element.Input, true
		}
	}
	for _, entry := range candidate.Entries {
		if entry.Key.HasInput && !entry.Key.HasValue {
			return entry.Key.Input, true
		}
		if entry.Value.HasInput && !entry.Value.HasValue {
			return entry.Value.Input, true
		}
	}
	return "", false
}

func valueAnalysisIncomplete(analysis valueAnalysis) bool {
	if analysis.Unsupported {
		return true
	}
	for _, issue := range analysis.Issues {
		if issue.Kind == valueIssueUnsupported || issue.Kind == valueIssueProducerBlocked {
			return true
		}
	}
	return false
}

func valueDiagnosticFailure(operation *invocation, model *effectiveModel, analysis valueAnalysis, diagnostics []Diagnostic, truncated bool) error {
	payload := operationDiagnosticPayload(operation, truncated)
	payload.Diagnostics = make([]diagnostic, len(diagnostics))
	projectedIndex := 0
	for _, issue := range analysis.Issues {
		if issue.Kind == valueIssueUnsupported || issue.Kind == valueIssueProducerBlocked {
			continue
		}
		if projectedIndex >= len(diagnostics) {
			break
		}
		public := diagnostics[projectedIndex]
		private := diagnostic{
			Code:          string(public.Code),
			Command:       privateDestinationCommand(model, issue.Destination),
			Source:        sourceID(public.InputSource.value),
			Origins:       projectOriginIDs(issue.Origins),
			Input:         public.Input,
			InputRedacted: public.InputRedacted,
			Sensitive:     public.Sensitive,
			ProducerKind:  projectDiagnosticProducerKind(public.ProducerKind),
			Tier:          projectResolutionTier(public.Tier),
			Disposition:   projectCandidateDisposition(public.Disposition),
		}
		if candidate, _, ok := rejectedValueCandidate(analysis, issue.Destination); ok {
			private.Declaration = privateCandidateDiagnosticDeclaration(candidate.Declaration)
			if entry, _, lifecycle := rejectedDomainLifecycleEntry(model, issue.Destination, candidate); lifecycle {
				private.Declaration = analysisDiagnosticDeclarationIdentity{
					Kind:             diagnosticDeclarationValueDomainEntry,
					ValueDomainEntry: entry.meta.id,
				}
			}
		}
		if len(private.Origins) == 0 {
			if candidate, _, ok := rejectedValueCandidate(analysis, issue.Destination); ok {
				private.Origins = projectOriginIDs(candidate.Meta.Origins)
			}
		}
		payload.Diagnostics[projectedIndex] = private
		projectedIndex++
	}
	return retainDiagnosticFailure(payload, firstValueIssueCause(analysis))
}

func firstValueIssueCause(analysis valueAnalysis) error {
	for _, issue := range analysis.Issues {
		if issue.Kind != valueIssueUnsupported && issue.Kind != valueIssueProducerBlocked &&
			issue.Err != nil {
			return issue.Err
		}
	}
	return nil
}

func privateCandidateDiagnosticDeclaration(declaration valueCandidateDeclaration) analysisDiagnosticDeclarationIdentity {
	switch {
	case declaration.FlagSpelling.present():
		identity := analysisDiagnosticDeclarationIdentity{
			Kind:         diagnosticDeclarationFlagSpelling,
			FlagSpelling: declaration.FlagSpelling,
		}
		return identity
	case declaration.Flag.present():
		identity := analysisDiagnosticDeclarationIdentity{
			Kind: diagnosticDeclarationFlag,
			Flag: declaration.Flag,
		}
		return identity
	case declaration.ArgumentTerm.present():
		identity := analysisDiagnosticDeclarationIdentity{
			Kind:         diagnosticDeclarationArgumentTerm,
			ArgumentTerm: declaration.ArgumentTerm,
		}
		return identity
	case declaration.EnvironmentInput.present():
		identity := analysisDiagnosticDeclarationIdentity{
			Kind:             diagnosticDeclarationEnvironmentInput,
			EnvironmentInput: declaration.EnvironmentInput,
		}
		return identity
	default:
		return analysisDiagnosticDeclarationIdentity{}
	}
}

func privateDestinationCommand(model *effectiveModel, destination ValueID) CommandID {
	record, ok := model.Destination(destination)
	if !ok {
		return CommandID(0)
	}
	return record.command
}

func projectOriginIDs(ids []InputOriginID) []originID {
	result := make([]originID, len(ids))
	for i, id := range ids {
		result[i] = originID(id.uint32())
	}
	return result
}

func projectDiagnosticProducerKind(producerKind ValueProducerKind) analysisDiagnosticProducerKind {
	switch producerKind {
	case ValueProducerKindFlag:
		return diagnosticProducerKindFlag
	case ValueProducerKindArgument:
		return diagnosticProducerKindArgument
	case ValueProducerKindRemainder:
		return diagnosticProducerKindRemainder
	case ValueProducerKindPassThrough:
		return diagnosticProducerKindPassThrough
	case ValueProducerKindShortcutDefault:
		return diagnosticProducerKindShortcutDefault
	case ValueProducerKindEnvironment:
		return diagnosticProducerKindEnvironment
	case ValueProducerKindInvocationDefault:
		return diagnosticProducerKindInvocationDefault
	case ValueProducerKindComputedDefault:
		return diagnosticProducerKindComputedDefault
	case ValueProducerKindStaticDefault:
		return diagnosticProducerKindStaticDefault
	case ValueProducerKindInitial:
		return diagnosticProducerKindInitial
	case ValueProducerKindSuppliedFact:
		return diagnosticProducerKindSuppliedFact
	case ValueProducerKindResolvedFact:
		return diagnosticProducerKindResolvedFact
	case ValueProducerKindDerived:
		return diagnosticProducerKindDerived
	default:
		return ""
	}
}

func projectResolutionTier(tier ResolutionTier) analysisDiagnosticResolutionTier {
	switch tier {
	case ResolutionTierExplicit:
		return diagnosticResolutionTierExplicit
	case ResolutionTierShortcutDefault:
		return diagnosticResolutionTierShortcutDefault
	case ResolutionTierEnvironment:
		return diagnosticResolutionTierEnvironment
	case ResolutionTierInvocationDefault:
		return diagnosticResolutionTierInvocationDefault
	case ResolutionTierComputedDefault:
		return diagnosticResolutionTierComputedDefault
	case ResolutionTierStaticDefault:
		return diagnosticResolutionTierStaticDefault
	case ResolutionTierInitial:
		return diagnosticResolutionTierInitial
	default:
		return ""
	}
}

func projectCandidateDisposition(disposition ValueCandidateDisposition) analysisDiagnosticDisposition {
	switch disposition {
	case ValueCandidateDispositionEffective:
		return diagnosticDispositionEffective
	case ValueCandidateDispositionNoEffect:
		return diagnosticDispositionNoEffect
	case ValueCandidateDispositionSuppressed:
		return diagnosticDispositionSuppressed
	case ValueCandidateDispositionConflicting:
		return diagnosticDispositionConflicting
	case ValueCandidateDispositionRejected:
		return diagnosticDispositionRejected
	case ValueCandidateDispositionUnresolved:
		return diagnosticDispositionUnresolved
	default:
		return ""
	}
}
