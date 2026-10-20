package cli

import (
	"cmp"
	"errors"
	"slices"
	"sync"
)

type analysisDestinationState uint8

const (
	destinationStateInvalid analysisDestinationState = iota
	destinationStateInactive
	destinationStateAbsent
	destinationStateResolved
	destinationStateFailed
	destinationStateBlocked
)

type contributionShape uint8

const (
	contributionShapeWhole contributionShape = iota + 1
	contributionShapeSequenceSegment
	contributionShapeKeyedEntry
	contributionShapeKeyedReplacement
)

type valueCandidateMeta struct {
	ID            valueCandidateRef
	Destination   ValueID
	ProducerKind  ValueProducerKind
	Tier          ResolutionTier
	Order         uint64
	Shape         contributionShape
	Sensitive     bool
	Origins       []InputOriginID
	Disposition   ValueCandidateDisposition
	Suppression   ValueCandidateSuppression
	SuppressedBy  []valueCandidateRef
	ConflictsWith []valueCandidateRef
}

func (meta valueCandidateMeta) clone() valueCandidateMeta {
	meta.Origins = append([]InputOriginID(nil), meta.Origins...)
	meta.SuppressedBy = append([]valueCandidateRef(nil), meta.SuppressedBy...)
	meta.ConflictsWith = append([]valueCandidateRef(nil), meta.ConflictsWith...)
	return meta
}

type valueCandidateInput struct {
	ProducerKind ValueProducerKind
	Tier         ResolutionTier
	Order        uint64
	Sensitive    bool
	Origins      []InputOriginID
}

func validateValueCandidateInput(input valueCandidateInput) {
	if input.ProducerKind == ValueProducerKindUnknown || input.Tier == ResolutionTierUnknown {
		panic("invocation: invalid candidate producer kind/tier")
	}
	for _, origin := range input.Origins {
		if !origin.present() {
			panic("invocation: invalid candidate origin")
		}
	}
}

type lazyValue[T any] struct {
	once  sync.Once
	fn    func() (T, error)
	value T
	err   error
}

func prepareLazyValue[T any](fn func() (T, error)) *lazyValue[T] {
	if fn == nil {
		panic("invocation: nil candidate evaluator")
	}
	return &lazyValue[T]{
		fn: fn,
	}
}

func (lazy *lazyValue[T]) get() (T, error) {
	lazy.once.Do(func() {
		lazy.value, lazy.err = lazy.fn()
		lazy.fn = nil
	})
	return lazy.value, lazy.err
}

type scalarValueCandidateInput[T any] struct {
	ProducerKind ValueProducerKind
	Tier         ResolutionTier
	Order        uint64
	Sensitive    bool
	Origins      []InputOriginID
	Evaluate     func() (T, error)
}

type scalarValueCandidate[T any] struct {
	meta  valueCandidateMeta
	value *lazyValue[T]
}

type scalarResolution[T any] struct {
	State      analysisDestinationState
	Value      T
	Sensitive  bool
	Candidates []valueCandidateMeta
	Err        error
}

type scalarResolver[T any] struct {
	analysis    *analysisState
	destination ValueID
	policy      duplicateResolutionPolicy
	strictTiers bool
	candidates  []scalarValueCandidate[T]
	once        sync.Once
	result      scalarResolution[T]
}

func prepareScalarResolver[T any](analysis *analysisState, destination ValueID, policy duplicateResolutionPolicy, strictTiers bool, inputs []scalarValueCandidateInput[T]) *scalarResolver[T] {
	if analysis == nil || !destination.present() {
		panic("invocation: invalid scalar resolver")
	}
	if policy > duplicateResolutionKeepLast {
		panic("invocation: invalid duplicate policy")
	}
	prepared := append([]scalarValueCandidateInput[T](nil), inputs...)
	slices.SortStableFunc(prepared, func(left, right scalarValueCandidateInput[T]) int {
		return cmp.Or(
			cmp.Compare(left.Tier, right.Tier),
			cmp.Compare(left.Order, right.Order),
		)
	})
	resolver := &scalarResolver[T]{
		analysis:    analysis,
		destination: destination,
		policy:      policy,
		strictTiers: strictTiers,
	}
	var previousTier ResolutionTier
	var previousOrder uint64
	for index, input := range prepared {
		input2 := valueCandidateInput{
			ProducerKind: input.ProducerKind,
			Tier:         input.Tier,
			Order:        input.Order,
			Sensitive:    input.Sensitive,
			Origins:      input.Origins,
		}
		validateValueCandidateInput(input2)
		if index > 0 && input.Tier == previousTier && input.Order == previousOrder {
			panic("invocation: duplicate candidate order")
		}
		previousTier, previousOrder = input.Tier, input.Order
		resolver.candidates = append(resolver.candidates, scalarValueCandidate[T]{
			meta: valueCandidateMeta{
				ID:           analysis.allocateValueCandidateRef(),
				Destination:  destination,
				ProducerKind: input.ProducerKind,
				Tier:         input.Tier,
				Order:        input.Order,
				Shape:        contributionShapeWhole,
				Sensitive:    input.Sensitive,
				Origins:      append([]InputOriginID(nil), input.Origins...),
				Disposition:  ValueCandidateDispositionUnresolved,
			},
			value: prepareLazyValue(input.Evaluate),
		})
	}
	return resolver
}

func (resolver *scalarResolver[T]) Resolve() scalarResolution[T] {
	if resolver == nil {
		panic("invocation: nil scalar resolver")
	}
	resolver.once.Do(resolver.resolve)
	return cloneScalarResolution(resolver.result)
}

func (resolver *scalarResolver[T]) resolve() {
	if len(resolver.candidates) == 0 {
		resolver.result.State = destinationStateAbsent
		return
	}
	if resolver.strictTiers {
		tiers := 0
		last := ResolutionTierUnknown
		for _, candidate := range resolver.candidates {
			if candidate.meta.Tier != last {
				tiers++
				last = candidate.meta.Tier
			}
		}
		if tiers > 1 {
			indices := make([]int, len(resolver.candidates))
			for i := range indices {
				indices[i] = i
			}
			markScalarConflicts(resolver.candidates, indices)
			resolver.result = scalarResolution[T]{
				State:      destinationStateFailed,
				Candidates: scalarMetas(resolver.candidates),
				Err:        errMultipleResolutionTiers,
			}
			return
		}
	}
	for start := 0; start < len(resolver.candidates); {
		end := start + 1
		for end < len(resolver.candidates) && resolver.candidates[end].meta.Tier == resolver.candidates[start].meta.Tier {
			end++
		}
		group := resolver.candidates[start:end]
		selected := 0
		switch resolver.policy {
		case duplicateResolutionDefault:
			if len(group) > 1 {
				indices := make([]int, 0, end-start)
				suppressors := make([]valueCandidateRef, 0, end-start)
				for i := start; i < end; i++ {
					indices = append(indices, i)
					suppressors = append(suppressors, resolver.candidates[i].meta.ID)
				}
				markScalarConflicts(resolver.candidates, indices)
				for i := end; i < len(resolver.candidates); i++ {
					markScalarSuppressed(resolver.candidates, i, ValueCandidateSuppressionPrecedence, suppressors...)
				}
				resolver.result = scalarResolution[T]{
					State:      destinationStateFailed,
					Candidates: scalarMetas(resolver.candidates),
					Err:        errSameTierConflict,
				}
				return
			}
		case duplicateResolutionKeepFirst:
			selected = 0
		case duplicateResolutionKeepLast:
			selected = len(group) - 1
		}
		selectedID := resolver.candidates[start+selected].meta.ID
		for i := range group {
			if i != selected {
				reason := ValueCandidateSuppressionKeepFirst
				if resolver.policy == duplicateResolutionKeepLast {
					reason = ValueCandidateSuppressionKeepLast
				}
				markScalarSuppressed(resolver.candidates, start+i, reason, selectedID)
			}
		}
		value, err := resolver.candidates[start+selected].value.get()
		if errors.Is(err, errValueCandidateOmitted) {
			resolver.candidates[start+selected].meta.Disposition = ValueCandidateDispositionNoEffect
			start = end
			continue
		}
		if err != nil {
			resolver.candidates[start+selected].meta.Disposition = ValueCandidateDispositionRejected
			for i := end; i < len(resolver.candidates); i++ {
				markScalarSuppressed(
					resolver.candidates,
					i,
					ValueCandidateSuppressionPrecedence,
					resolver.candidates[start+selected].meta.ID,
				)
			}
			resolver.result = scalarResolution[T]{
				State:      destinationStateFailed,
				Candidates: scalarMetas(resolver.candidates),
				Err:        err,
			}
			return
		}
		resolver.candidates[start+selected].meta.Disposition = ValueCandidateDispositionEffective
		for i := end; i < len(resolver.candidates); i++ {
			markScalarSuppressed(
				resolver.candidates,
				i,
				ValueCandidateSuppressionPrecedence,
				resolver.candidates[start+selected].meta.ID,
			)
		}
		resolver.result = scalarResolution[T]{
			State:      destinationStateResolved,
			Value:      value,
			Sensitive:  resolver.candidates[start+selected].meta.Sensitive,
			Candidates: scalarMetas(resolver.candidates),
		}
		return
	}
}

var (
	errSameTierConflict        = errors.New("same-tier candidate conflict")
	errMultipleResolutionTiers = errors.New("multiple active resolution tiers")
	errKeyedDuplicate          = errors.New("same-tier keyed duplicate")
	errValueCandidateOmitted   = errors.New("candidate omitted")
)

func markScalarSuppressed[T any](candidates []scalarValueCandidate[T], index int, reason ValueCandidateSuppression, suppressors ...valueCandidateRef) {
	candidates[index].meta.Disposition = ValueCandidateDispositionSuppressed
	candidates[index].meta.Suppression = reason
	candidates[index].meta.SuppressedBy = append([]valueCandidateRef(nil), suppressors...)
}

func markScalarConflicts[T any](candidates []scalarValueCandidate[T], indices []int) {
	for _, index := range indices {
		candidates[index].meta.Disposition = ValueCandidateDispositionConflicting
		for _, other := range indices {
			if other != index {
				candidates[index].meta.ConflictsWith = append(
					candidates[index].meta.ConflictsWith,
					candidates[other].meta.ID,
				)
			}
		}
	}
}

func scalarMetas[T any](candidates []scalarValueCandidate[T]) []valueCandidateMeta {
	result := make([]valueCandidateMeta, len(candidates))
	for i := range candidates {
		result[i] = candidates[i].meta.clone()
	}
	return result
}

func cloneScalarResolution[T any](result scalarResolution[T]) scalarResolution[T] {
	result.Candidates = append([]valueCandidateMeta(nil), result.Candidates...)
	for i := range result.Candidates {
		result.Candidates[i] = result.Candidates[i].clone()
	}
	return result
}

type sequenceContributionKind uint8

const (
	sequenceContributionSegment sequenceContributionKind = iota + 1
	sequenceContributionWhole
)

type sequenceValueCandidateInput[T any] struct {
	ProducerKind ValueProducerKind
	Tier         ResolutionTier
	Order        uint64
	Sensitive    bool
	Origins      []InputOriginID
	Kind         sequenceContributionKind
	Evaluate     func() ([]T, error)
}

type sequenceValueCandidate[T any] struct {
	meta  valueCandidateMeta
	kind  sequenceContributionKind
	value *lazyValue[[]T]
}

type sequenceResolution[T any] struct {
	State      analysisDestinationState
	Value      []T
	Sensitive  bool
	Candidates []valueCandidateMeta
	Err        error
}

type sequenceResolver[T any] struct {
	analysis    *analysisState
	destination ValueID
	combine     ResolutionTierOrder
	candidates  []sequenceValueCandidate[T]
	once        sync.Once
	result      sequenceResolution[T]
}

func prepareSequenceResolver[T any](analysis *analysisState, destination ValueID, combine ResolutionTierOrder, inputs []sequenceValueCandidateInput[T]) *sequenceResolver[T] {
	if analysis == nil || !destination.present() {
		panic("invocation: invalid sequence resolver")
	}
	if combine != resolutionTierOrderUnknown && combine != ResolutionTierLowerFirst &&
		combine != ResolutionTierHigherFirst {
		panic("invocation: invalid resolution tier order")
	}
	prepared := append([]sequenceValueCandidateInput[T](nil), inputs...)
	slices.SortStableFunc(prepared, func(left, right sequenceValueCandidateInput[T]) int {
		return cmp.Or(
			cmp.Compare(left.Tier, right.Tier),
			cmp.Compare(left.Order, right.Order),
		)
	})
	resolver := &sequenceResolver[T]{
		analysis:    analysis,
		destination: destination,
		combine:     combine,
	}
	for _, input := range prepared {
		input2 := valueCandidateInput{
			ProducerKind: input.ProducerKind,
			Tier:         input.Tier,
			Order:        input.Order,
			Sensitive:    input.Sensitive,
			Origins:      input.Origins,
		}
		validateValueCandidateInput(input2)
		if input.Kind != sequenceContributionSegment && input.Kind != sequenceContributionWhole {
			panic("invocation: invalid sequence contribution")
		}
		shape := contributionShapeSequenceSegment
		if input.Kind == sequenceContributionWhole {
			shape = contributionShapeWhole
		}
		composite := sequenceValueCandidate[T]{
			meta: valueCandidateMeta{
				ID:           analysis.allocateValueCandidateRef(),
				Destination:  destination,
				ProducerKind: input.ProducerKind,
				Tier:         input.Tier,
				Order:        input.Order,
				Shape:        shape,
				Sensitive:    input.Sensitive,
				Origins:      append([]InputOriginID(nil), input.Origins...),
				Disposition:  ValueCandidateDispositionUnresolved,
			},
			kind:  input.Kind,
			value: prepareLazyValue(input.Evaluate),
		}
		resolver.candidates = append(resolver.candidates, composite)
	}
	return resolver
}

func (resolver *sequenceResolver[T]) Resolve() sequenceResolution[T] {
	if resolver == nil {
		panic("invocation: nil sequence resolver")
	}
	resolver.once.Do(resolver.resolve)
	result := resolver.result
	result.Value = append([]T(nil), result.Value...)
	result.Candidates = append([]valueCandidateMeta(nil), result.Candidates...)
	for i := range result.Candidates {
		result.Candidates[i] = result.Candidates[i].clone()
	}
	return result
}

func (resolver *sequenceResolver[T]) resolve() {
	if len(resolver.candidates) == 0 {
		resolver.result.State = destinationStateAbsent
		return
	}
	if resolver.combine == resolutionTierOrderUnknown {
		resolver.resolveHighestTier()
		return
	}
	resolver.resolveCombined()
}

func (resolver *sequenceResolver[T]) resolveHighestTier() {
	for start := 0; start < len(resolver.candidates); {
		end := start + 1
		for end < len(resolver.candidates) && resolver.candidates[end].meta.Tier == resolver.candidates[start].meta.Tier {
			end++
		}
		var values []T
		effective := false
		sensitive := false
		for i := start; i < end; i++ {
			value, err := resolver.candidates[i].value.get()
			if errors.Is(err, errValueCandidateOmitted) {
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionNoEffect
				continue
			}
			if err != nil {
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionRejected
				for j := end; j < len(resolver.candidates); j++ {
					markSequenceSuppressed(resolver.candidates, j, resolver.candidates[i].meta.ID)
				}
				resolver.result = sequenceResolution[T]{
					State:      destinationStateFailed,
					Candidates: sequenceMetas(resolver.candidates),
					Err:        err,
				}
				return
			}
			if resolver.candidates[i].kind == sequenceContributionWhole || len(value) != 0 {
				effective = true
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionEffective
				values = append(values, value...)
				if resolver.candidates[i].meta.Sensitive {
					sensitive = true
				}
			} else {
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionNoEffect
			}
		}
		if effective {
			var suppressors []valueCandidateRef
			for i := start; i < end; i++ {
				if resolver.candidates[i].meta.Disposition == ValueCandidateDispositionEffective {
					suppressors = append(suppressors, resolver.candidates[i].meta.ID)
				}
			}
			for i := end; i < len(resolver.candidates); i++ {
				markSequenceSuppressed(resolver.candidates, i, suppressors...)
			}
			resolver.result = sequenceResolution[T]{
				State:      destinationStateResolved,
				Value:      values,
				Sensitive:  sensitive,
				Candidates: sequenceMetas(resolver.candidates),
			}
			return
		}
		start = end
	}
	resolver.result = sequenceResolution[T]{
		State:      destinationStateAbsent,
		Candidates: sequenceMetas(resolver.candidates),
	}
}

func (resolver *sequenceResolver[T]) resolveCombined() {
	type group struct{ start, end int }
	var groups []group
	for start := 0; start < len(resolver.candidates); {
		end := start + 1
		for end < len(resolver.candidates) && resolver.candidates[end].meta.Tier == resolver.candidates[start].meta.Tier {
			end++
		}
		groups = append(groups, group{start, end})
		start = end
	}
	if resolver.combine == ResolutionTierLowerFirst {
		for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
			groups[i], groups[j] = groups[j], groups[i]
		}
	}
	var values []T
	sensitive := false
	effective := false
	for _, g := range groups {
		for i := g.start; i < g.end; i++ {
			value, err := resolver.candidates[i].value.get()
			if errors.Is(err, errValueCandidateOmitted) {
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionNoEffect
				continue
			}
			if err != nil {
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionRejected
				resolver.result = sequenceResolution[T]{
					State:      destinationStateFailed,
					Candidates: sequenceMetas(resolver.candidates),
					Err:        err,
				}
				return
			}
			if resolver.candidates[i].kind == sequenceContributionWhole || len(value) != 0 {
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionEffective
				values = append(values, value...)
				effective = true
				if resolver.candidates[i].meta.Sensitive {
					sensitive = true
				}
			} else {
				resolver.candidates[i].meta.Disposition = ValueCandidateDispositionNoEffect
			}
		}
	}
	state := destinationStateAbsent
	if effective {
		state = destinationStateResolved
	}
	resolver.result = sequenceResolution[T]{
		State:      state,
		Value:      values,
		Sensitive:  sensitive,
		Candidates: sequenceMetas(resolver.candidates),
	}
}

func markSequenceSuppressed[T any](candidates []sequenceValueCandidate[T], index int, suppressors ...valueCandidateRef) {
	candidates[index].meta.Disposition = ValueCandidateDispositionSuppressed
	candidates[index].meta.Suppression = ValueCandidateSuppressionPrecedence
	candidates[index].meta.SuppressedBy = append([]valueCandidateRef(nil), suppressors...)
}

func sequenceMetas[T any](candidates []sequenceValueCandidate[T]) []valueCandidateMeta {
	result := make([]valueCandidateMeta, len(candidates))
	for i := range candidates {
		result[i] = candidates[i].meta.clone()
	}
	return result
}

// keyedBatchKind preserves the producer contribution boundary required by
// keyed resolution. Entry batches resolve per typed key but commit atomically;
// replacements are one destination-level contribution that may suppress all
// lower-precedence keyed input without forcing lower key conversion.
type keyedBatchKind uint8

const (
	keyedBatchEntries keyedBatchKind = iota + 1
	keyedBatchReplacement
)

type keyedBatchEntryInput struct {
	Key   func() (any, error)
	Value func() (any, error)
}

type keyedBatchInput struct {
	ProducerKind ValueProducerKind
	Tier         ResolutionTier
	Order        uint64
	Sensitive    bool
	Origins      []InputOriginID
	Kind         keyedBatchKind
	Entries      []keyedBatchEntryInput
	Whole        func() ([]keyedValueEntry, error)
}

type keyedBatchValueCandidateResult struct {
	Meta       valueCandidateMeta
	BatchIndex int
	EntryIndex int
	Key        any
	HasKey     bool
	Value      any
	HasValue   bool
	Entries    []keyedValueEntry
}

type keyedBatchEntryReference struct {
	CandidateIndex int
	EntryIndex     int
}

type keyedBatchResolution struct {
	State      analysisDestinationState
	Entries    []keyedValueEntry
	EntryRefs  []keyedBatchEntryReference
	Sensitive  bool
	Candidates []keyedBatchValueCandidateResult
	Err        error
}

type keyedResolutionBatch struct {
	input      keyedBatchInput
	inputIndex int
	candidates []int
	whole      *lazyValue[[]keyedValueEntry]
}

// resolveKeyedBatches is the operation-local keyed resolution kernel used by
// value analysis. It deliberately stages every selected keyed value before
// publishing any Effective disposition, preserving producer-batch atomicity.
func resolveKeyedBatches(analysis *analysisState, destination ValueID, policy duplicateResolutionPolicy, equalKey func(any, any) bool, inputs []keyedBatchInput) keyedBatchResolution {
	if analysis == nil || !destination.present() || equalKey == nil {
		panic("invocation: invalid keyed batch resolver")
	}
	inputsCopy := append([]keyedBatchInput(nil), inputs...)
	order := make([]int, len(inputsCopy))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(leftIndex, rightIndex int) int {
		left, right := inputsCopy[leftIndex], inputsCopy[rightIndex]
		return cmp.Or(
			cmp.Compare(left.Tier, right.Tier),
			cmp.Compare(left.Order, right.Order),
		)
	})

	batches := make([]keyedResolutionBatch, 0, len(inputs))
	var candidates []keyedBatchValueCandidateResult
	for _, inputIndex := range order {
		input := inputsCopy[inputIndex]
		input2 := valueCandidateInput{
			ProducerKind: input.ProducerKind,
			Tier:         input.Tier,
			Order:        input.Order,
			Sensitive:    input.Sensitive,
			Origins:      input.Origins,
		}
		validateValueCandidateInput(input2)
		batch := keyedResolutionBatch{
			input:      input,
			inputIndex: inputIndex,
		}
		switch input.Kind {
		case keyedBatchEntries:
			for entryIndex, entry := range input.Entries {
				if entry.Key == nil || entry.Value == nil {
					panic("invocation: invalid keyed entry batch")
				}
				candidateIndex := len(candidates)
				batch.candidates = append(batch.candidates, candidateIndex)
				keyedBatchValueCandidateResultValue := keyedBatchValueCandidateResult{
					Meta: valueCandidateMeta{
						ID:           analysis.allocateValueCandidateRef(),
						Destination:  destination,
						ProducerKind: input.ProducerKind,
						Tier:         input.Tier,
						Order:        keyedValueCandidateOrder(input.Order, entryIndex),
						Shape:        contributionShapeKeyedEntry,
						Sensitive:    input.Sensitive,
						Origins:      append([]InputOriginID(nil), input.Origins...),
						Disposition:  ValueCandidateDispositionUnresolved,
					},
					BatchIndex: inputIndex,
					EntryIndex: entryIndex,
				}
				candidates = append(candidates, keyedBatchValueCandidateResultValue)
			}
		case keyedBatchReplacement:
			if input.Whole == nil {
				panic("invocation: invalid keyed replacement batch")
			}
			candidateIndex := len(candidates)
			batch.candidates = []int{candidateIndex}
			batch.whole = prepareLazyValue(input.Whole)
			keyedBatchValueCandidateResultValue2 := keyedBatchValueCandidateResult{
				Meta: valueCandidateMeta{
					ID:           analysis.allocateValueCandidateRef(),
					Destination:  destination,
					ProducerKind: input.ProducerKind,
					Tier:         input.Tier,
					Order:        keyedValueCandidateOrder(input.Order, 0),
					Shape:        contributionShapeKeyedReplacement,
					Sensitive:    input.Sensitive,
					Origins:      append([]InputOriginID(nil), input.Origins...),
					Disposition:  ValueCandidateDispositionUnresolved,
				},
				BatchIndex: inputIndex,
				EntryIndex: -1,
			}
			candidates = append(candidates, keyedBatchValueCandidateResultValue2)
		default:
			panic("invocation: invalid keyed batch kind")
		}
		batches = append(batches, batch)
	}

	// Find the highest-precedence effective whole replacement. A higher whole
	// suppresses lower keyed input without converting its keys. Omitted callback
	// candidates are no-effect and allow the next tier to participate.
	boundaryBatch := -1
	boundaryTier := ResolutionTierUnknown
	for start := 0; start < len(batches); {
		end := start + 1
		for end < len(batches) && batches[end].input.Tier == batches[start].input.Tier {
			end++
		}
		var activeWholes []int
		entryCount := 0
		for i := start; i < end; i++ {
			batch := &batches[i]
			if batch.input.Kind == keyedBatchEntries {
				entryCount += len(batch.input.Entries)
				continue
			}
			values, err := batch.whole.get()
			candidate := &candidates[batch.candidates[0]]
			if errors.Is(err, errValueCandidateOmitted) {
				candidate.Meta.Disposition = ValueCandidateDispositionNoEffect
				continue
			}
			if err != nil {
				candidate.Meta.Disposition = ValueCandidateDispositionRejected
				result := keyedBatchResolution{
					State:      destinationStateFailed,
					Candidates: cloneKeyedBatchValueCandidates(candidates),
					Err:        err,
				}

				return result
			}
			if !validKeyedEntrySet(values, equalKey) {
				candidate.Meta.Disposition = ValueCandidateDispositionRejected
				result := keyedBatchResolution{
					State:      destinationStateFailed,
					Candidates: cloneKeyedBatchValueCandidates(candidates),
					Err:        errInvalidKeyedKey,
				}

				return result
			}
			candidate.Entries = cloneKeyedEntries(values)
			activeWholes = append(activeWholes, i)
		}
		if len(activeWholes) == 0 {
			start = end
			continue
		}
		if len(activeWholes) != 1 || entryCount != 0 {
			var conflictIDs []valueCandidateRef
			for i := start; i < end; i++ {
				for _, candidateIndex := range batches[i].candidates {
					if candidates[candidateIndex].Meta.Disposition == ValueCandidateDispositionNoEffect {
						continue
					}
					conflictIDs = append(conflictIDs, candidates[candidateIndex].Meta.ID)
				}
			}
			for i := start; i < end; i++ {
				for _, candidateIndex := range batches[i].candidates {
					candidate := &candidates[candidateIndex]
					if candidate.Meta.Disposition == ValueCandidateDispositionNoEffect {
						continue
					}
					candidate.Meta.Disposition = ValueCandidateDispositionConflicting
					for _, id := range conflictIDs {
						if id != candidate.Meta.ID {
							candidate.Meta.ConflictsWith = append(candidate.Meta.ConflictsWith, id)
						}
					}
				}
			}
			result := keyedBatchResolution{
				State:      destinationStateFailed,
				Candidates: cloneKeyedBatchValueCandidates(candidates),
				Err:        errSameTierConflict,
			}

			return result
		}
		boundaryBatch = activeWholes[0]
		boundaryTier = batches[boundaryBatch].input.Tier
		break
	}

	// Lower contributions are destination-level suppressed by the selected whole
	// and their keys/values remain unevaluated.
	if boundaryBatch >= 0 {
		wholeCandidate := &candidates[batches[boundaryBatch].candidates[0]]
		for i := boundaryBatch + 1; i < len(batches); i++ {
			if batches[i].input.Tier <= boundaryTier {
				continue
			}
			for _, candidateIndex := range batches[i].candidates {
				candidate := &candidates[candidateIndex]
				if candidate.Meta.Disposition != ValueCandidateDispositionUnresolved {
					continue
				}
				candidate.Meta.Disposition = ValueCandidateDispositionSuppressed
				candidate.Meta.Suppression = ValueCandidateSuppressionPrecedence
				candidate.Meta.SuppressedBy = []valueCandidateRef{wholeCandidate.Meta.ID}
			}
		}
	}

	// Resolve entry candidates above the whole replacement, or all entries when
	// there is no whole replacement. Key conversion is complete before per-key
	// suppression, and value evaluation is staged before Effective publication.
	entryLimit := len(batches)
	if boundaryBatch >= 0 {
		entryLimit = boundaryBatch
	}
	type entryRef struct {
		batchIndex     int
		candidateIndex int
		entryIndex     int
		key            any
	}
	var entries []entryRef
	for batchIndex := range entryLimit {
		batch := &batches[batchIndex]
		if batch.input.Kind != keyedBatchEntries {
			continue
		}
		for entryIndex, entry := range batch.input.Entries {
			key, err := entry.Key()
			candidateIndex := batch.candidates[entryIndex]
			candidate := &candidates[candidateIndex]
			if err != nil {
				candidate.Meta.Disposition = ValueCandidateDispositionRejected
				result := keyedBatchResolution{
					State:      destinationStateFailed,
					Candidates: cloneKeyedBatchValueCandidates(candidates),
					Err:        err,
				}

				return result
			}
			if !equalKey(key, key) {
				candidate.Meta.Disposition = ValueCandidateDispositionRejected
				result := keyedBatchResolution{
					State:      destinationStateFailed,
					Candidates: cloneKeyedBatchValueCandidates(candidates),
					Err:        errInvalidKeyedKey,
				}

				return result
			}
			candidate.Key, candidate.HasKey = snapshotMutableValue(key), true
			entryRefValue := entryRef{
				batchIndex:     batchIndex,
				candidateIndex: candidateIndex,
				entryIndex:     entryIndex,
				key:            key,
			}
			entries = append(entries, entryRefValue)
		}
	}

	groups := make([][]entryRef, 0)
	for _, entry := range entries {
		found := -1
		for i := range groups {
			if equalKey(groups[i][0].key, entry.key) {
				found = i
				break
			}
		}
		if found < 0 {
			groups = append(groups, []entryRef{entry})
		} else {
			groups[found] = append(groups[found], entry)
		}
	}

	selected := make([]entryRef, 0, len(groups))
	for _, group := range groups {
		bestTier := candidates[group[0].candidateIndex].Meta.Tier
		var same, lower []entryRef
		for _, entry := range group {
			if candidates[entry.candidateIndex].Meta.Tier == bestTier {
				same = append(same, entry)
			} else {
				lower = append(lower, entry)
			}
		}
		chosenIndex := 0
		if len(same) > 1 {
			switch policy {
			case duplicateResolutionDefault:
				var ids []valueCandidateRef
				for _, entry := range same {
					ids = append(ids, candidates[entry.candidateIndex].Meta.ID)
				}
				for _, entry := range same {
					candidate := &candidates[entry.candidateIndex]
					candidate.Meta.Disposition = ValueCandidateDispositionConflicting
					for _, id := range ids {
						if id != candidate.Meta.ID {
							candidate.Meta.ConflictsWith = append(candidate.Meta.ConflictsWith, id)
						}
					}
				}
				for _, entry := range lower {
					candidate := &candidates[entry.candidateIndex]
					candidate.Meta.Disposition = ValueCandidateDispositionSuppressed
					candidate.Meta.Suppression = ValueCandidateSuppressionPrecedence
					candidate.Meta.SuppressedBy = append([]valueCandidateRef(nil), ids...)
				}
				result := keyedBatchResolution{
					State:      destinationStateFailed,
					Candidates: cloneKeyedBatchValueCandidates(candidates),
					Err:        errKeyedDuplicate,
				}

				return result
			case duplicateResolutionKeepLast:
				chosenIndex = len(same) - 1
			case duplicateResolutionKeepFirst:
			}
		}
		chosen := same[chosenIndex]
		chosenID := candidates[chosen.candidateIndex].Meta.ID
		for i, entry := range same {
			if i == chosenIndex {
				continue
			}
			candidate := &candidates[entry.candidateIndex]
			candidate.Meta.Disposition = ValueCandidateDispositionSuppressed
			candidate.Meta.Suppression = ValueCandidateSuppressionKeepFirst
			if policy == duplicateResolutionKeepLast {
				candidate.Meta.Suppression = ValueCandidateSuppressionKeepLast
			}
			candidate.Meta.SuppressedBy = []valueCandidateRef{chosenID}
		}
		for _, entry := range lower {
			candidate := &candidates[entry.candidateIndex]
			candidate.Meta.Disposition = ValueCandidateDispositionSuppressed
			candidate.Meta.Suppression = ValueCandidateSuppressionPrecedence
			candidate.Meta.SuppressedBy = []valueCandidateRef{chosenID}
		}
		selected = append(selected, chosen)
	}

	// A selected higher entry can override one key of the effective whole map.
	// Stage all selected values before marking any candidate Effective.
	selectedValues := make([]keyedValueEntry, len(selected))
	for i, entry := range selected {
		value, err := batches[entry.batchIndex].input.Entries[entry.entryIndex].Value()
		candidate := &candidates[entry.candidateIndex]
		if err != nil {
			candidate.Meta.Disposition = ValueCandidateDispositionRejected
			result := keyedBatchResolution{
				State:      destinationStateFailed,
				Candidates: cloneKeyedBatchValueCandidates(candidates),
				Err:        err,
			}

			return result
		}
		candidate.Value, candidate.HasValue = snapshotMutableValue(value), true
		selectedValues[i] = keyedValueEntry{
			Key:   entry.key,
			Value: value,
		}
	}

	resultEntries := append([]keyedValueEntry(nil), selectedValues...)
	entryRefs := make([]keyedBatchEntryReference, 0, len(selectedValues))
	for _, entry := range selected {
		entryRefs = append(entryRefs, keyedBatchEntryReference{
			CandidateIndex: entry.candidateIndex,
			EntryIndex:     0,
		})
	}
	if boundaryBatch >= 0 {
		wholeCandidate := &candidates[batches[boundaryBatch].candidates[0]]
		for wholeEntryIndex, entry := range wholeCandidate.Entries {
			overridden := false
			for _, selectedEntry := range selectedValues {
				if equalKey(entry.Key, selectedEntry.Key) {
					overridden = true
					break
				}
			}
			if !overridden {
				resultEntries = append(resultEntries, entry)
				entryRefs = append(entryRefs, keyedBatchEntryReference{
					CandidateIndex: batches[boundaryBatch].candidates[0],
					EntryIndex:     wholeEntryIndex,
				})
			}
		}
		wholeCandidate.Meta.Disposition = ValueCandidateDispositionEffective
	}
	for _, entry := range selected {
		candidate := &candidates[entry.candidateIndex]
		candidate.Meta.Disposition = ValueCandidateDispositionEffective
	}

	state := destinationStateAbsent
	if boundaryBatch >= 0 || len(selected) != 0 {
		state = destinationStateResolved
	}
	sensitive := false
	for i := range candidates {
		if candidates[i].Meta.Disposition == ValueCandidateDispositionEffective && candidates[i].Meta.Sensitive {
			sensitive = true
		}
	}
	result := keyedBatchResolution{
		State:      state,
		Entries:    cloneKeyedEntries(resultEntries),
		EntryRefs:  append([]keyedBatchEntryReference(nil), entryRefs...),
		Sensitive:  sensitive,
		Candidates: cloneKeyedBatchValueCandidates(candidates),
	}

	return result
}

var errInvalidKeyedKey = errors.New("invalid keyed candidate key")

func keyedValueCandidateOrder(order uint64, entryIndex int) uint64 {
	const shift = 32
	return order<<shift | uint64(checkedUint32FromInt(entryIndex+1))
}

func validKeyedEntrySet(entries []keyedValueEntry, equalKey func(any, any) bool) bool {
	for _, entry := range entries {
		if !equalKey(entry.Key, entry.Key) {
			return false
		}
	}
	return true
}

func cloneKeyedEntries(entries []keyedValueEntry) []keyedValueEntry {
	result := make([]keyedValueEntry, len(entries))
	for i, entry := range entries {
		result[i] = keyedValueEntry{
			Key:   snapshotMutableValue(entry.Key),
			Value: snapshotMutableValue(entry.Value),
		}
	}
	return result
}

func cloneKeyedBatchValueCandidates(candidates []keyedBatchValueCandidateResult) []keyedBatchValueCandidateResult {
	result := make([]keyedBatchValueCandidateResult, len(candidates))
	for i, candidate := range candidates {
		result[i] = candidate
		result[i].Meta = candidate.Meta.clone()
		result[i].Key = snapshotMutableValue(candidate.Key)
		result[i].Value = snapshotMutableValue(candidate.Value)
		result[i].Entries = cloneKeyedEntries(candidate.Entries)
	}
	return result
}
