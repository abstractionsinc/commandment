package cli

// Release attaches lifecycle to this exact flag-spelling declaration and returns the declaration
// for continued fluent authoring; lifecycle belongs to the declaration rather than the destination
// or replacement relation.
func (declaration FlagSpellingDeclaration[C, T]) Release(lifecycle ReleaseLifecycle) FlagSpellingDeclaration[C, T] {
	return configureFlagSpellingRelease(declaration, lifecycle)
}

// Release attaches lifecycle to this exact environment-input declaration and returns it for
// continued fluent authoring; lifecycle does not alter environment provenance or destination
// identity.
func (declaration EnvironmentInputDeclaration[C, T]) Release(lifecycle ReleaseLifecycle) EnvironmentInputDeclaration[C, T] {
	stageEnvironmentLifecycle(declaration.state, declaration.producer, lifecycle)
	return declaration
}

// Release attaches lifecycle to this prefix environment-input declaration.
func (declaration EnvironmentPrefixDeclaration[C, K, V]) Release(lifecycle ReleaseLifecycle) EnvironmentPrefixDeclaration[C, K, V] {
	stageEnvironmentLifecycle(declaration.state, declaration.producer, lifecycle)
	return declaration
}

// Release attaches lifecycle to this exact argument-form declaration and returns it for continued
// fluent authoring.
func (declaration ArgumentFormDeclaration[C]) Release(lifecycle ReleaseLifecycle) ArgumentFormDeclaration[C] {
	stageArgumentFormLifecycle(declaration.state, declaration.form, lifecycle)
	return declaration
}

// Release attaches lifecycle to this exact argument-literal declaration and returns it for
// continued fluent authoring.
func (declaration ArgumentLiteralDeclaration[C]) Release(lifecycle ReleaseLifecycle) ArgumentLiteralDeclaration[C] {
	stageArgumentTermLifecycle(declaration.state, declaration.term, lifecycle)
	return declaration
}

// Release attaches lifecycle to this exact finite-domain value declaration and returns it for
// continued fluent authoring.
func (declaration DomainValueDeclaration[T]) Release(lifecycle ReleaseLifecycle) DomainValueDeclaration[T] {
	declaration.use(func(_ *valueDomainState, entry *valueDomainEntry) { entry.lifecycle = lifecycle })
	return declaration
}

// Release attaches lifecycle to this exact finite-domain alias declaration and returns it for
// continued fluent authoring.
func (declaration DomainAliasDeclaration[T]) Release(lifecycle ReleaseLifecycle) DomainAliasDeclaration[T] {
	declaration.use(func(_ *valueDomainState, entry *valueDomainEntry) { entry.lifecycle = lifecycle })
	return declaration
}

// ReleaseScheme is the immutable reusable application authority for comparing release identifiers.
// Its zero value means lifecycle comparison is unavailable.
type ReleaseScheme struct {
	_     [0]func()
	state *releaseSchemeImpl
}

type releaseSchemes struct{}

// Release provides constructors for application release-comparison schemes.
var Release releaseSchemes

// SemVer returns a release scheme using strict semantic-version precedence; build metadata does not
// affect precedence and invalid versions are rejected when the scheme is validated.
func (releaseSchemes) SemVer(current string) ReleaseScheme {
	return compileSemVerReleaseScheme(current)
}

// Ordered returns a finite release scheme whose first argument is the current release and whose
// remaining arguments define the complete strict order by list position, not lexical comparison.
// The caller-owned order slice is copied.
func (releaseSchemes) Ordered(current string, order ...string) ReleaseScheme {
	return compileOrderedReleaseScheme(current, order)
}

// ReleaseLifecycle declares introduction, deprecation, and removal boundaries on one exact
// declaration. Reason is ordinary source-first localizable text and replacement identity is a
// separate relationship.
type ReleaseLifecycle struct {
	IntroducedIn string
	DeprecatedIn string
	RemovedIn    string
	Reason       string
}

// LifecycleState is the evaluated state of one declaration at a release selected by the applicable
// ReleaseScheme.
type LifecycleState uint8

const (
	// LifecycleStateUnknown means no evaluated lifecycle state is available.
	LifecycleStateUnknown LifecycleState = iota
	// LifecycleStateNotIntroduced applies before the declaration's IntroducedIn boundary.
	LifecycleStateNotIntroduced
	// LifecycleStateCurrent applies after introduction and before deprecation or removal.
	LifecycleStateCurrent
	// LifecycleStateDeprecated applies at or after DeprecatedIn and before RemovedIn.
	LifecycleStateDeprecated
	// LifecycleStateRemoved applies at or after RemovedIn.
	LifecycleStateRemoved
)

type releaseSchemeKind uint8

const (
	releaseSchemeInvalid releaseSchemeKind = iota
	releaseSchemeSemVer
	releaseSchemeOrdered
)

type releaseSchemeImpl struct {
	kind    releaseSchemeKind
	current string
	order   []string
	index   map[string]int
}

func compileSemVerReleaseScheme(current string) ReleaseScheme {
	impl := &releaseSchemeImpl{
		kind:    releaseSchemeSemVer,
		current: current,
	}
	scheme := ReleaseScheme{
		state: impl,
	}
	return scheme
}

func compileOrderedReleaseScheme(current string, order []string) ReleaseScheme {
	impl := &releaseSchemeImpl{
		kind:    releaseSchemeOrdered,
		current: current,
		order:   append([]string(nil), order...),
	}
	scheme := ReleaseScheme{
		state: impl,
	}
	return scheme
}

func compileRelease(scheme ReleaseScheme) (ReleaseScheme, error) {
	if scheme.state == nil {
		return ReleaseScheme{}, nil
	}

	switch scheme.state.kind {
	case releaseSchemeSemVer:
		if _, ok := parseSemVer(scheme.state.current); !ok {
			return ReleaseScheme{}, errInvalidReleaseScheme
		}

		impl := &releaseSchemeImpl{
			kind:    releaseSchemeSemVer,
			current: scheme.state.current,
		}
		releaseScheme := ReleaseScheme{
			state: impl,
		}
		return releaseScheme, nil

	case releaseSchemeOrdered:
		current := scheme.state.current
		order := append([]string(nil), scheme.state.order...)
		if current == "" || len(order) == 0 {
			return ReleaseScheme{}, errInvalidReleaseScheme
		}

		index := make(map[string]int, len(order))
		for position, release := range order {
			if release == "" {
				return ReleaseScheme{}, errInvalidReleaseScheme
			}
			if _, duplicate := index[release]; duplicate {
				return ReleaseScheme{}, errInvalidReleaseScheme
			}
			index[release] = position
		}
		if _, ok := index[current]; !ok {
			return ReleaseScheme{}, errInvalidReleaseScheme
		}

		impl := &releaseSchemeImpl{
			kind:    releaseSchemeOrdered,
			current: current,
			order:   order,
			index:   index,
		}
		releaseScheme := ReleaseScheme{
			state: impl,
		}
		return releaseScheme, nil

	default:
		return ReleaseScheme{}, errInvalidReleaseScheme
	}
}

func (scheme ReleaseScheme) valid() bool {
	return scheme.state != nil &&
		(scheme.state.kind == releaseSchemeSemVer || scheme.state.kind == releaseSchemeOrdered)
}

func (scheme ReleaseScheme) current() string {
	if !scheme.valid() {
		return ""
	}
	return scheme.state.current
}

func (scheme ReleaseScheme) validate(release string) error {
	if !scheme.valid() || release == "" {
		return errInvalidRelease
	}

	switch scheme.state.kind {
	case releaseSchemeSemVer:
		if _, ok := parseSemVer(release); !ok {
			return errInvalidRelease
		}

	case releaseSchemeOrdered:
		if _, ok := scheme.state.index[release]; !ok {
			return errInvalidRelease
		}

	default:
		return errInvalidRelease
	}

	return nil
}

func (scheme ReleaseScheme) compare(left, right string) (int, error) {
	if err := scheme.validate(left); err != nil {
		return 0, err
	}
	if err := scheme.validate(right); err != nil {
		return 0, err
	}

	switch scheme.state.kind {
	case releaseSchemeOrdered:
		return compareInt(scheme.state.index[left], scheme.state.index[right]), nil

	case releaseSchemeSemVer:
		l, _ := parseSemVer(left)
		r, _ := parseSemVer(right)
		return compareSemVer(l, r), nil

	default:
		return 0, errInvalidReleaseScheme
	}
}
