package cli

import (
	"math"
	"regexp"
	"slices"
)

// truth is the private three-valued proof result. It is intentionally not a
// public semantic enum.
type truth uint8

const (
	unknown truth = iota
	proved
	disproved
)

func (truth truth) Not() truth {
	switch truth {
	case proved:
		return disproved
	case disproved:
		return proved
	default:
		return unknown
	}
}

// scalar is the bounded literal vocabulary understood by the mandatory proof
// kernel. Values that cannot be lowered soundly into this vocabulary remain
// outside proof rather than being guessed through reflection or formatting.
type scalar struct {
	kind scalarKind
	i    int64
	u    uint64
	s    string
	b    bool
}

type scalarKind uint8

const (
	scalarInvalid scalarKind = iota
	scalarBool
	scalarInt
	scalarUint
	scalarFloat
	scalarString
)

func scalarBoolValue(value bool) scalar {
	return scalar{
		kind: scalarBool,
		b:    value,
	}
}

func scalarIntValue(value int64) scalar {
	return scalar{
		kind: scalarInt,
		i:    value,
	}
}

func scalarUintValue(value uint64) scalar {
	return scalar{
		kind: scalarUint,
		u:    value,
	}
}

func scalarStringValue(value string) scalar {
	return scalar{
		kind: scalarString,
		s:    value,
	}
}

func scalarFloatValue(value float64) (scalar, bool) {
	if math.IsNaN(value) {
		return scalar{}, false
	}
	return scalar{
		kind: scalarFloat,
		u:    math.Float64bits(value),
	}, true
}

func (value scalar) valid() bool { return value.kind != scalarInvalid }

func (value scalar) compare(other scalar) (int, bool) {
	if !value.valid() || value.kind != other.kind {
		return 0, false
	}
	switch value.kind {
	case scalarBool:
		if value.b == other.b {
			return 0, true
		}
		if !value.b {
			return -1, true
		}
		return 1, true
	case scalarInt:
		return compareOrdered(value.i, other.i), true
	case scalarUint:
		return compareOrdered(value.u, other.u), true
	case scalarFloat:
		return compareOrdered(math.Float64frombits(value.u), math.Float64frombits(other.u)), true
	case scalarString:
		return compareOrdered(value.s, other.s), true
	default:
		return 0, false
	}
}

func compareOrdered[T ~int64 | ~uint64 | ~float64 | ~string](left, right T) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

type proofConditionKind uint8

const (
	conditionInvalid proofConditionKind = iota
	conditionTrue
	conditionFalse
	conditionAtom
	semanticConditionAll
	semanticConditionAny
	semanticConditionNot
)

type atomKind uint8

const (
	atomInvalid atomKind = iota
	atomResolved
	atomEqual
	atomNotEqual
	atomIn
	atomNotIn
	atomGreaterThan
	atomAtLeast
	atomLessThan
	atomAtMost
	atomMatches
)

type atom struct {
	kind        atomKind
	destination ValueID
	values      []scalar
	pattern     string
}

// proofCondition is a private immutable structured theorem input. There is no
// callback/predicate escape hatch: opaque application behavior cannot enter
// this proof vocabulary.
type proofCondition struct {
	kind     proofConditionKind
	atom     atom
	children []proofCondition
}

func always() proofCondition {
	return proofCondition{
		kind: conditionTrue,
	}
}

func resolved(destination ValueID) proofCondition {
	mustProofDestination(destination)
	return proofCondition{
		kind: conditionAtom,
		atom: atom{
			kind:        atomResolved,
			destination: destination,
		},
	}
}

func equal(destination ValueID, value scalar) proofCondition {
	return scalarAtom(atomEqual, destination, []scalar{value})
}

func notEqual(destination ValueID, value scalar) proofCondition {
	return scalarAtom(atomNotEqual, destination, []scalar{value})
}

func in(destination ValueID, values ...scalar) proofCondition {
	return scalarAtom(atomIn, destination, values)
}

func notIn(destination ValueID, values ...scalar) proofCondition {
	return scalarAtom(atomNotIn, destination, values)
}

func greaterThan(destination ValueID, value scalar) proofCondition {
	return scalarAtom(atomGreaterThan, destination, []scalar{value})
}

func atLeast(destination ValueID, value scalar) proofCondition {
	return scalarAtom(atomAtLeast, destination, []scalar{value})
}

func lessThan(destination ValueID, value scalar) proofCondition {
	return scalarAtom(atomLessThan, destination, []scalar{value})
}

func atMost(destination ValueID, value scalar) proofCondition {
	return scalarAtom(atomAtMost, destination, []scalar{value})
}

func matches(destination ValueID, pattern string) (proofCondition, bool) {
	mustProofDestination(destination)
	if _, err := regexp.Compile(pattern); err != nil {
		return proofCondition{}, false
	}
	return proofCondition{
		kind: conditionAtom,
		atom: atom{
			kind:        atomMatches,
			destination: destination,
			pattern:     pattern,
		},
	}, true
}

func scalarAtom(kind atomKind, destination ValueID, values []scalar) proofCondition {
	mustProofDestination(destination)
	for _, value := range values {
		if !value.valid() {
			panic("accept: invalid proof scalar")
		}
	}
	return proofCondition{
		kind: conditionAtom,
		atom: atom{
			kind:        kind,
			destination: destination,
			values:      append([]scalar(nil), values...),
		},
	}
}

func all(first proofCondition, rest ...proofCondition) proofCondition {
	children := make([]proofCondition, 0, 1+len(rest))
	children = append(children, cloneCondition(first))
	for _, child := range rest {
		children = append(children, cloneCondition(child))
	}
	return proofCondition{
		kind:     semanticConditionAll,
		children: children,
	}
}

func anyCondition(first proofCondition, rest ...proofCondition) proofCondition {
	children := make([]proofCondition, 0, 1+len(rest))
	children = append(children, cloneCondition(first))
	for _, child := range rest {
		children = append(children, cloneCondition(child))
	}
	return proofCondition{
		kind:     semanticConditionAny,
		children: children,
	}
}

func not(condition proofCondition) proofCondition {
	return proofCondition{
		kind:     semanticConditionNot,
		children: []proofCondition{cloneCondition(condition)},
	}
}

func cloneCondition(condition proofCondition) proofCondition {
	result := condition
	result.atom.values = append([]scalar(nil), condition.atom.values...)
	result.children = make([]proofCondition, len(condition.children))
	for i, child := range condition.children {
		result.children[i] = cloneCondition(child)
	}
	return result
}

// destinationUniverse contains only phase-authoritative structured facts.
// DomainKnown means Domain is the complete finite domain for this destination.
type destinationUniverse struct {
	Destination ValueID
	Resolved    truth
	DomainKnown bool
	Domain      []scalar
}

// universe is an immutable acceptance-phase theorem universe.
type universe struct {
	destinations map[ValueID]semanticDestinationUniverse
}

type semanticDestinationUniverse struct {
	resolved    truth
	domainKnown bool
	domain      []scalar
}

func prepareProofUniverse(destinations []destinationUniverse) universe {
	result := universe{
		destinations: make(map[ValueID]semanticDestinationUniverse, len(destinations)),
	}
	for _, destination := range destinations {
		mustProofDestination(destination.Destination)
		if _, exists := result.destinations[destination.Destination]; exists {
			panic("accept: duplicate proof destination")
		}
		domain := append([]scalar(nil), destination.Domain...)
		for _, value := range domain {
			if !value.valid() {
				panic("accept: invalid proof domain scalar")
			}
		}
		result.destinations[destination.Destination] = semanticDestinationUniverse{
			resolved:    destination.Resolved,
			domainKnown: destination.DomainKnown,
			domain:      domain,
		}
	}
	return result
}

// satisfiable classifies whether the proof kernel can establish at least one
// state in universe for which condition is true, can establish that no such
// state exists, or cannot soundly decide existence. This is distinct from
// Prove: an open condition may be neither universally true nor universally
// false while still having a constructible witness. Acceptance invariants such
// as conditional-default overlap use this existential result rather than
// treating mere non-disproof as proof of overlap.
func satisfiable(condition proofCondition, universe universe) truth {
	if !condition.valid() {
		panic("accept: malformed proof condition")
	}
	form := toDNF(condition, false)
	if !semanticSatisfiable(form, universe) {
		return disproved
	}
	hasUnknown := false
	for _, clause := range form {
		switch clauseWitness(clause, universe) {
		case proved:
			return proved
		case unknown:
			hasUnknown = true
		case disproved:
		}
	}
	if hasUnknown {
		return unknown
	}
	return disproved
}

func (condition proofCondition) valid() bool {
	switch condition.kind {
	case conditionTrue, conditionFalse:
		return true
	case conditionAtom:
		return condition.atom.valid()
	case semanticConditionAll, semanticConditionAny:
		if len(condition.children) == 0 {
			return false
		}
		for _, child := range condition.children {
			if !child.valid() {
				return false
			}
		}
		return true
	case semanticConditionNot:
		return len(condition.children) == 1 && condition.children[0].valid()
	default:
		return false
	}
}

func (a atom) valid() bool {
	if !a.destination.present() {
		return false
	}
	switch a.kind {
	case atomResolved:
		return true
	case atomEqual, atomNotEqual, atomGreaterThan, atomAtLeast, atomLessThan, atomAtMost:
		return len(a.values) == 1 && a.values[0].valid()
	case atomIn, atomNotIn:
		for _, value := range a.values {
			if !value.valid() {
				return false
			}
		}
		return true
	case atomMatches:
		_, err := regexp.Compile(a.pattern)
		return err == nil
	default:
		return false
	}
}

type literal struct {
	atom    atom
	negated bool
}

type (
	clause []literal
	dnf    []clause
)

func toDNF(condition proofCondition, negated bool) dnf {
	switch condition.kind {
	case conditionTrue:
		if negated {
			return nil
		}
		return dnf{clause{}}
	case conditionFalse:
		if negated {
			return dnf{clause{}}
		}
		return nil
	case conditionAtom:
		return dnf{clause{{
			atom:    condition.atom,
			negated: negated,
		}}}
	case semanticConditionNot:
		return toDNF(condition.children[0], !negated)
	case semanticConditionAll:
		if negated {
			var result dnf
			for _, child := range condition.children {
				result = append(result, toDNF(child, true)...)
			}
			return result
		}
		result := dnf{clause{}}
		for _, child := range condition.children {
			result = product(result, toDNF(child, false))
		}
		return result
	case semanticConditionAny:
		if negated {
			result := dnf{clause{}}
			for _, child := range condition.children {
				result = product(result, toDNF(child, true))
			}
			return result
		}
		var result dnf
		for _, child := range condition.children {
			result = append(result, toDNF(child, false)...)
		}
		return result
	default:
		panic("accept: invalid condition kind")
	}
}

func product(left, right dnf) dnf {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	result := make(dnf, 0, len(left)*len(right))
	for _, a := range left {
		for _, b := range right {
			combined := make(clause, 0, len(a)+len(b))
			combined = append(combined, a...)
			combined = append(combined, b...)
			result = append(result, combined)
		}
	}
	return result
}

func semanticSatisfiable(form dnf, universe universe) bool {
	for _, clause := range form {
		if clauseSatisfiable(clause, universe) {
			return true
		}
	}
	return false
}

type bound struct {
	value     scalar
	inclusive bool
	present   bool
}

type destinationConstraint struct {
	resolved *bool
	kind     scalarKind
	allowed  map[scalar]struct{}
	excluded map[scalar]struct{}
	lower    bound
	upper    bound
	patterns []patternConstraint
}

type patternConstraint struct {
	pattern *regexp.Regexp
	negated bool
}

func clauseSatisfiable(clause clause, universe universe) bool {
	constraints := make(map[ValueID]*destinationConstraint)
	for _, literal := range clause {
		constraint := constraints[literal.atom.destination]
		if constraint == nil {
			constraint = prepareProofConstraint(universe.destinations[literal.atom.destination])
			constraints[literal.atom.destination] = constraint
		}
		if !constraint.apply(literal) {
			return false
		}
	}
	for _, constraint := range constraints {
		if !constraint.finish() {
			return false
		}
	}
	return true
}

func prepareProofConstraint(universe semanticDestinationUniverse) *destinationConstraint {
	constraint := &destinationConstraint{
		excluded: make(map[scalar]struct{}),
	}
	switch universe.resolved {
	case proved:
		value := true
		constraint.resolved = &value
	case disproved:
		value := false
		constraint.resolved = &value
	case unknown:
	}
	if universe.domainKnown {
		constraint.allowed = make(map[scalar]struct{}, len(universe.domain))
		for _, value := range universe.domain {
			constraint.allowed[value] = struct{}{}
			constraint.noteKind(value.kind)
		}
	}
	return constraint
}

func (constraint *destinationConstraint) noteKind(kind scalarKind) bool {
	if kind == scalarInvalid {
		return false
	}
	if constraint.kind == scalarInvalid {
		constraint.kind = kind
		return true
	}
	return constraint.kind == kind
}

func (constraint *destinationConstraint) apply(literal literal) bool {
	a := literal.atom
	negated := literal.negated
	if a.kind == atomResolved {
		wanted := !negated
		if constraint.resolved != nil && *constraint.resolved != wanted {
			return false
		}
		constraint.resolved = &wanted
		return true
	}

	// Scalar predicates imply a resolved value.
	resolved := true
	if constraint.resolved != nil && !*constraint.resolved {
		return false
	}
	constraint.resolved = &resolved

	switch a.kind {
	case atomEqual:
		if negated {
			return constraint.exclude(a.values[0])
		}
		return constraint.intersect([]scalar{a.values[0]})
	case atomNotEqual:
		if negated {
			return constraint.intersect([]scalar{a.values[0]})
		}
		return constraint.exclude(a.values[0])
	case atomIn:
		if negated {
			for _, value := range a.values {
				if !constraint.exclude(value) {
					return false
				}
			}
			return true
		}
		return constraint.intersect(a.values)
	case atomNotIn:
		if negated {
			return constraint.intersect(a.values)
		}
		for _, value := range a.values {
			if !constraint.exclude(value) {
				return false
			}
		}
		return true
	case atomGreaterThan, atomAtLeast, atomLessThan, atomAtMost:
		kind := a.kind
		if negated {
			switch kind {
			case atomGreaterThan:
				kind = atomAtMost
			case atomAtLeast:
				kind = atomLessThan
			case atomLessThan:
				kind = atomAtLeast
			case atomAtMost:
				kind = atomGreaterThan
			case atomInvalid, atomResolved, atomEqual, atomNotEqual, atomIn, atomNotIn, atomMatches:
			}
		}
		return constraint.applyBound(kind, a.values[0])
	case atomMatches:
		pattern, err := regexp.Compile(a.pattern)
		if err != nil {
			return false
		}
		if !constraint.noteKind(scalarString) {
			return false
		}
		constraint.patterns = append(constraint.patterns, patternConstraint{
			pattern: pattern,
			negated: negated,
		})
		return true
	default:
		return false
	}
}

func (constraint *destinationConstraint) intersect(values []scalar) bool {
	candidate := make(map[scalar]struct{}, len(values))
	for _, value := range values {
		if !constraint.noteKind(value.kind) {
			return false
		}
		candidate[value] = struct{}{}
	}
	if constraint.allowed == nil {
		constraint.allowed = candidate
		return true
	}
	for value := range constraint.allowed {
		if _, ok := candidate[value]; !ok {
			delete(constraint.allowed, value)
		}
	}
	return len(constraint.allowed) > 0
}

func (constraint *destinationConstraint) exclude(value scalar) bool {
	if !constraint.noteKind(value.kind) {
		return false
	}
	constraint.excluded[value] = struct{}{}
	return true
}

func (constraint *destinationConstraint) applyBound(kind atomKind, value scalar) bool {
	if value.kind == scalarBool || !constraint.noteKind(value.kind) {
		return false
	}
	switch kind {
	case atomGreaterThan:
		return constraint.setLower(value, false)
	case atomAtLeast:
		return constraint.setLower(value, true)
	case atomLessThan:
		return constraint.setUpper(value, false)
	case atomAtMost:
		return constraint.setUpper(value, true)
	default:
		return false
	}
}

func (constraint *destinationConstraint) setLower(value scalar, inclusive bool) bool {
	if !constraint.lower.present {
		constraint.lower = bound{
			value:     value,
			inclusive: inclusive,
			present:   true,
		}
		return true
	}
	comparison, ok := value.compare(constraint.lower.value)
	if !ok {
		return false
	}
	if comparison > 0 || comparison == 0 && !inclusive && constraint.lower.inclusive {
		constraint.lower = bound{
			value:     value,
			inclusive: inclusive,
			present:   true,
		}
	}
	return true
}

func (constraint *destinationConstraint) setUpper(value scalar, inclusive bool) bool {
	if !constraint.upper.present {
		constraint.upper = bound{
			value:     value,
			inclusive: inclusive,
			present:   true,
		}
		return true
	}
	comparison, ok := value.compare(constraint.upper.value)
	if !ok {
		return false
	}
	if comparison < 0 || comparison == 0 && !inclusive && constraint.upper.inclusive {
		constraint.upper = bound{
			value:     value,
			inclusive: inclusive,
			present:   true,
		}
	}
	return true
}

func (constraint *destinationConstraint) finish() bool {
	if constraint.resolved != nil && !*constraint.resolved {
		// Unresolved satisfies no scalar predicate. If only Resolved(false) was
		// required, no scalar state was recorded and this remains satisfiable.
		return constraint.kind == scalarInvalid && constraint.allowed == nil && len(constraint.excluded) == 0 &&
			!constraint.lower.present &&
			!constraint.upper.present &&
			len(constraint.patterns) == 0
	}
	if constraint.lower.present && constraint.upper.present {
		comparison, ok := constraint.lower.value.compare(constraint.upper.value)
		if !ok || comparison > 0 || comparison == 0 && (!constraint.lower.inclusive || !constraint.upper.inclusive) {
			return false
		}
	}
	if constraint.allowed != nil {
		for value := range constraint.allowed {
			if constraint.valueAllowed(value) {
				return true
			}
		}
		return false
	}
	// Without a finite/equality candidate set, exclusions and regex predicates
	// over an open universe cannot establish exhaustion. Bounds were checked
	// structurally above, so a witness exists for the supported ordered types.
	return true
}

func (constraint *destinationConstraint) valueAllowed(value scalar) bool {
	if _, excluded := constraint.excluded[value]; excluded {
		return false
	}
	if constraint.lower.present {
		comparison, ok := value.compare(constraint.lower.value)
		if !ok || comparison < 0 || comparison == 0 && !constraint.lower.inclusive {
			return false
		}
	}
	if constraint.upper.present {
		comparison, ok := value.compare(constraint.upper.value)
		if !ok || comparison > 0 || comparison == 0 && !constraint.upper.inclusive {
			return false
		}
	}
	if len(constraint.patterns) > 0 {
		if value.kind != scalarString {
			return false
		}
		for _, pattern := range constraint.patterns {
			matched := pattern.pattern.MatchString(value.s)
			if matched == pattern.negated {
				return false
			}
		}
	}
	return true
}

func clauseWitness(clause clause, universe universe) truth {
	constraints := make(map[ValueID]*destinationConstraint)
	for _, literal := range clause {
		constraint := constraints[literal.atom.destination]
		if constraint == nil {
			constraint = prepareProofConstraint(universe.destinations[literal.atom.destination])
			constraints[literal.atom.destination] = constraint
		}
		if !constraint.apply(literal) {
			return disproved
		}
	}
	result := proved
	for _, constraint := range constraints {
		if !constraint.finish() {
			return disproved
		}
		switch constraint.witness() {
		case disproved:
			return disproved
		case unknown:
			result = unknown
		case proved:
		}
	}
	return result
}

func (constraint *destinationConstraint) witness() truth {
	if constraint.resolved != nil && !*constraint.resolved {
		if constraint.kind == scalarInvalid && constraint.allowed == nil && len(constraint.excluded) == 0 &&
			!constraint.lower.present &&
			!constraint.upper.present &&
			len(constraint.patterns) == 0 {
			return proved
		}
		return disproved
	}
	if constraint.allowed != nil {
		for value := range constraint.allowed {
			if constraint.valueAllowed(value) {
				return proved
			}
		}
		return disproved
	}
	if constraint.kind == scalarInvalid {
		return proved
	}
	if len(constraint.patterns) != 0 {
		for i, left := range constraint.patterns {
			for _, right := range constraint.patterns[i+1:] {
				if left.negated != right.negated && left.pattern.String() == right.pattern.String() {
					return disproved
				}
			}
		}
		// General regexp intersection/complement over an open string universe is
		// deliberately outside the mandatory theorem vocabulary. A finite or
		// equality candidate set was handled above.
		return unknown
	}
	switch constraint.kind {
	case scalarBool:
		if slices.ContainsFunc([]scalar{scalarBoolValue(false), scalarBoolValue(true)}, constraint.valueAllowed) {
			return proved
		}
		return disproved
	case scalarInt:
		return constraint.intWitness()
	case scalarUint:
		return constraint.uintWitness()
	case scalarFloat:
		return constraint.floatWitness()
	case scalarString:
		return constraint.stringWitness()
	default:
		return unknown
	}
}

func (constraint *destinationConstraint) intWitness() truth {
	lower := int64(math.MinInt64)
	upper := int64(math.MaxInt64)
	if constraint.lower.present {
		if constraint.lower.value.kind != scalarInt {
			return disproved
		}
		lower = constraint.lower.value.i
		if !constraint.lower.inclusive {
			if lower == math.MaxInt64 {
				return disproved
			}
			lower++
		}
	}
	if constraint.upper.present {
		if constraint.upper.value.kind != scalarInt {
			return disproved
		}
		upper = constraint.upper.value.i
		if !constraint.upper.inclusive {
			if upper == math.MinInt64 {
				return disproved
			}
			upper--
		}
	}
	if lower > upper {
		return disproved
	}
	candidate := lower
	for attempts := 0; attempts <= len(constraint.excluded); attempts++ {
		value := scalarIntValue(candidate)
		if constraint.valueAllowed(value) {
			return proved
		}
		if candidate == upper || candidate == math.MaxInt64 {
			return disproved
		}
		candidate++
	}
	// We tried one more in-range value than there are exact exclusions.
	return proved
}

func (constraint *destinationConstraint) uintWitness() truth {
	lower := uint64(0)
	upper := uint64(math.MaxUint64)
	if constraint.lower.present {
		if constraint.lower.value.kind != scalarUint {
			return disproved
		}
		lower = constraint.lower.value.u
		if !constraint.lower.inclusive {
			if lower == math.MaxUint64 {
				return disproved
			}
			lower++
		}
	}
	if constraint.upper.present {
		if constraint.upper.value.kind != scalarUint {
			return disproved
		}
		upper = constraint.upper.value.u
		if !constraint.upper.inclusive {
			if upper == 0 {
				return disproved
			}
			upper--
		}
	}
	if lower > upper {
		return disproved
	}
	candidate := lower
	for attempts := 0; attempts <= len(constraint.excluded); attempts++ {
		value := scalarUintValue(candidate)
		if constraint.valueAllowed(value) {
			return proved
		}
		if candidate == upper || candidate == math.MaxUint64 {
			return disproved
		}
		candidate++
	}
	return proved
}

const witnessProbeSlack = 2

func (constraint *destinationConstraint) floatWitness() truth {
	candidates := []float64{0, -1, 1, math.Inf(-1), math.Inf(1)}
	if constraint.lower.present {
		value := math.Float64frombits(constraint.lower.value.u)
		candidates = append(candidates, value)
		if !constraint.lower.inclusive {
			candidates = append(candidates, math.Nextafter(value, math.Inf(1)))
		}
	}
	if constraint.upper.present {
		value := math.Float64frombits(constraint.upper.value.u)
		candidates = append(candidates, value)
		if !constraint.upper.inclusive {
			candidates = append(candidates, math.Nextafter(value, math.Inf(-1)))
		}
	}
	for i := 0; i <= len(constraint.excluded)+witnessProbeSlack; i++ {
		candidates = append(candidates, float64(i+witnessProbeSlack), -float64(i+witnessProbeSlack))
	}
	for _, candidate := range candidates {
		if math.IsNaN(candidate) {
			continue
		}
		value, ok := scalarFloatValue(candidate)
		if ok && constraint.valueAllowed(value) {
			return proved
		}
	}
	// The representable float domain is discrete. If our bounded candidate
	// probes cannot construct a witness without exhaustively enumerating a
	// potentially large interval, retain Unknown rather than guessing.
	return unknown
}

func (constraint *destinationConstraint) stringWitness() truth {
	candidates := []string{"", "\x00", "a"}
	if constraint.lower.present {
		value := constraint.lower.value.s
		candidates = append(candidates, value, value+"\x00")
	}
	if constraint.upper.present {
		candidates = append(candidates, constraint.upper.value.s)
	}
	for i := 0; i <= len(constraint.excluded)+witnessProbeSlack; i++ {
		candidates = append(candidates, string(make([]byte, i+witnessProbeSlack)))
	}
	for _, candidate := range candidates {
		if constraint.valueAllowed(scalarStringValue(candidate)) {
			return proved
		}
	}
	return unknown
}

func mustProofDestination(destination ValueID) {
	if !destination.present() {
		panic("accept: invalid destination identity")
	}
}

// validateProofInvariants consumes the mandatory three-valued proof kernel only
// after ordinary referential/structural validation has established that every
// theorem input is a valid Draft-local semantic relationship. Consequence
// remains invariant-owned: today default overlap is the rejection-capable
// static/provider invariant that consumes ProofCondition satisfiability directly.
func validateProofInvariants(input declarationRecordInput) []rejection {
	universe := proofUniverseFromSlices(input.DomainEntries, input.ValueDomains)
	profiles := make(map[GrammarProfileID]grammarProfileRecord, len(input.GrammarProfiles))
	profilesByCommand := make(map[CommandID][]grammarProfileRecord)
	for _, profile := range input.GrammarProfiles {
		profiles[profile.meta.id] = profile
		profilesByCommand[profile.command] = append(profilesByCommand[profile.command], profile)
	}

	defaultsByParent := make(map[CommandID][]defaultRecord)
	for _, record := range input.Defaults {
		defaultsByParent[record.parent] = append(defaultsByParent[record.parent], record)
	}
	var rejections []rejection
	for _, defaults := range defaultsByParent {
		for index, current := range defaults {
			for _, prior := range defaults[:index] {
				if current.meta.profile.present() && prior.meta.profile.present() && current.meta.profile != prior.meta.profile {
					// Profile branches are structurally mutually exclusive.
					continue
				}
				left, leftOK := defaultProofCondition(prior, profiles, profilesByCommand)
				right, rightOK := defaultProofCondition(current, profiles, profilesByCommand)
				if !leftOK || !rightOK {
					// Unsupported theorem input is Unknown, never invalidity.
					continue
				}
				if satisfiable(all(left, right), universe) != proved {
					continue
				}
				rejectionValue := rejection{
					Kind:    rejectionProvedInvariantViolation,
					Order:   uint64(current.meta.position),
					Subject: uint64(current.meta.id.uint32()),
					Related: uint64(prior.meta.id.uint32()),
				}
				rejections = append(rejections, rejectionValue)
			}
		}
	}
	return rejections
}

const defaultProofConditionCapacity = 2

func defaultProofCondition(record defaultRecord, profiles map[GrammarProfileID]grammarProfileRecord, profilesByCommand map[CommandID][]grammarProfileRecord) (proofCondition, bool) {
	parts := make([]proofCondition, 0, defaultProofConditionCapacity)
	if profileID := record.meta.profile; profileID.present() {
		profile, ok := profiles[profileID]
		if !ok {
			return proofCondition{}, false
		}
		condition, ok := grammarProfileCondition(profile, profilesByCommand[profile.command])
		if !ok {
			return proofCondition{}, false
		}
		parts = append(parts, condition)
	}
	if condition, conditional := record.Condition(); conditional {
		lowered, ok := semanticLowerCondition(condition)
		if !ok {
			return proofCondition{}, false
		}
		parts = append(parts, lowered)
	}
	switch len(parts) {
	case 0:
		return always(), true
	case 1:
		return parts[0], true
	default:
		return all(parts[0], parts[1:]...), true
	}
}

func grammarProfileCondition(profile grammarProfileRecord, siblings []grammarProfileRecord) (proofCondition, bool) {
	selector, ok := profile.Selector()
	if !ok {
		return proofCondition{}, false
	}
	if !profile.defaultCase {
		value, ok := profile.Value()
		if !ok {
			return proofCondition{}, false
		}
		scalar, ok := lowerScalar(value)
		if !ok {
			return proofCondition{}, false
		}
		return equal(selector, scalar), true
	}
	values := make([]scalar, 0, len(siblings))
	for _, sibling := range siblings {
		if sibling.meta.id == profile.meta.id || sibling.defaultCase {
			continue
		}
		value, ok := sibling.Value()
		if !ok {
			return proofCondition{}, false
		}
		scalar, ok := lowerScalar(value)
		if !ok {
			return proofCondition{}, false
		}
		values = append(values, scalar)
	}
	return all(resolved(selector), notIn(selector, values...)), true
}

func semanticLowerCondition(condition condition) (proofCondition, bool) {
	if !condition.Valid() {
		return proofCondition{}, false
	}
	scalarOperands := func() ([]scalar, bool) {
		operands := condition.Operands()
		result := make([]scalar, len(operands))
		for index, operand := range operands {
			scalar, ok := lowerScalar(operand)
			if !ok {
				return nil, false
			}
			result[index] = scalar
		}
		return result, true
	}
	switch condition.Kind() {
	case conditionKindResolved:
		return resolved(condition.Destination()), true
	case conditionKindEqual, conditionKindNotEqual,
		conditionKindIn, conditionKindNotIn,
		conditionKindGreaterThan, conditionKindAtLeast,
		conditionKindLessThan, conditionKindAtMost:
		operands, ok := scalarOperands()
		if !ok {
			return proofCondition{}, false
		}
		switch condition.Kind() {
		case conditionKindEqual:
			return equal(condition.Destination(), operands[0]), true
		case conditionKindNotEqual:
			return notEqual(condition.Destination(), operands[0]), true
		case conditionKindIn:
			return in(condition.Destination(), operands...), true
		case conditionKindNotIn:
			return notIn(condition.Destination(), operands...), true
		case conditionKindGreaterThan:
			return greaterThan(condition.Destination(), operands[0]), true
		case conditionKindAtLeast:
			return atLeast(condition.Destination(), operands[0]), true
		case conditionKindLessThan:
			return lessThan(condition.Destination(), operands[0]), true
		case conditionKindAtMost:
			return atMost(condition.Destination(), operands[0]), true
		case conditionKindUnknown,
			conditionKindResolved,
			conditionKindMatches,
			conditionKindAll,
			conditionKindAny,
			conditionKindNot:
		}
	case conditionKindMatches:
		return matches(condition.Destination(), condition.Pattern())
	case conditionKindAll, conditionKindAny:
		children := condition.Children()
		lowered := make([]proofCondition, len(children))
		for index, child := range children {
			value, ok := semanticLowerCondition(child)
			if !ok {
				return proofCondition{}, false
			}
			lowered[index] = value
		}
		if condition.Kind() == conditionKindAll {
			return all(lowered[0], lowered[1:]...), true
		}
		return anyCondition(lowered[0], lowered[1:]...), true
	case conditionKindNot:
		child, ok := semanticLowerCondition(condition.Children()[0])
		if !ok {
			return proofCondition{}, false
		}
		return not(child), true
	case conditionKindUnknown:
	}
	return proofCondition{}, false
}

func lowerScalar(value any) (scalar, bool) {
	switch value := value.(type) {
	case bool:
		return scalarBoolValue(value), true
	case int:
		return scalarIntValue(int64(value)), true
	case int8:
		return scalarIntValue(int64(value)), true
	case int16:
		return scalarIntValue(int64(value)), true
	case int32:
		return scalarIntValue(int64(value)), true
	case int64:
		return scalarIntValue(value), true
	case uint:
		return scalarUintValue(uint64(value)), true
	case uint8:
		return scalarUintValue(uint64(value)), true
	case uint16:
		return scalarUintValue(uint64(value)), true
	case uint32:
		return scalarUintValue(uint64(value)), true
	case uint64:
		return scalarUintValue(value), true
	case uintptr:
		return scalarUintValue(uint64(value)), true
	case float32:
		return scalarFloatValue(float64(value))
	case float64:
		return scalarFloatValue(value)
	case string:
		return scalarStringValue(value), true
	default:
		return scalar{}, false
	}
}

func proofUniverseFromSlices(domainEntries []domainEntryRecord, valueDomains []valueDomainRecord) universe {
	entries := make(map[ValueDomainEntryID]domainEntryRecord, len(domainEntries))
	for _, entry := range domainEntries {
		entries[entry.meta.id] = entry
	}
	type domainState struct {
		seen  bool
		known bool
		items []scalar
	}
	byDestination := make(map[ValueID]domainState)
	for _, domain := range valueDomains {
		if _, dynamic := domain.DynamicDefinition(); dynamic {
			continue
		}
		values := make([]scalar, 0, len(domain.Entries()))
		known := true
		seenValues := make(map[scalar]struct{})
		for _, entryID := range domain.Entries() {
			entry, ok := entries[entryID]
			if !ok {
				known = false
				break
			}
			if entry.kind == domainEntryKindAlias {
				entry = entries[entry.canonical]
			}
			value, ok := entry.Value()
			if !ok {
				known = false
				break
			}
			scalar, ok := lowerScalar(value)
			if !ok {
				known = false
				break
			}
			if _, duplicate := seenValues[scalar]; !duplicate {
				seenValues[scalar] = struct{}{}
				values = append(values, scalar)
			}
		}
		for _, use := range domain.Uses() {
			if use.kind != ValueDomainUseDestination {
				continue
			}
			state := byDestination[use.destination]
			if state.seen {
				// Multiple direct domains do not become theorem input by guessed
				// intersection semantics.
				state.known = false
				state.items = nil
				byDestination[use.destination] = state
				continue
			}
			state.seen = true
			state.known = known
			if known {
				state.items = append([]scalar(nil), values...)
			}
			byDestination[use.destination] = state
		}
	}
	facts := make([]destinationUniverse, 0, len(byDestination))
	for destination, state := range byDestination {
		if !state.known {
			continue
		}
		destinationUniverseValue := destinationUniverse{
			Destination: destination,
			Resolved:    unknown,
			DomainKnown: true,
			Domain:      state.items,
		}
		facts = append(facts, destinationUniverseValue)
	}
	return prepareProofUniverse(facts)
}
