# Contributing to Commandment

Thanks for contributing to Commandment.

## Development setup

Commandment requires Go 1.27.1. Clone the repository, then run `make` to see the available development targets. The normal quality and module-hygiene targets are:

| Command | Purpose |
| --- | --- |
| `make fmt` | Mutate Go source to the canonical repository format. |
| `make lint` | Run the authoritative non-mutating mechanical quality gate; hard findings must be zero. |
| `make lint-review` | Run the non-gating diagnostic review surface; findings require human review rather than establishing correctness failures. |
| `make test` | Run the complete ordinary repository test suite. |
| `make test-coverage` | Report separate coverage profiles for the production owners `cli`, `clitest`, `clibench`, and `shelltest`. |
| `make test-race` | Run the broad race campaign. |
| `make bench` | Screen the permanent CLI benchmark suite in `cli/test/benchmark` with the repository's bounded settings; this is not decision-grade performance evidence. |
| `make tidy` | Tidy the main module and the tooling module. |

Before submitting a change, run the commands that are relevant to the files and behavior you changed. Changes that affect concurrency should include appropriate race evidence; during iteration that may begin with a focused package race run before the broad `make test-race` campaign. Performance-sensitive changes may use `make bench` for broad screening; consequential performance conclusions should use the focused protocol appropriate to the decision.

## Public API changes

Commandment follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html). Within a stable major line, public API and behavioral contracts that Commandment promises to preserve for that major version remain backward compatible unless a change is explicitly being developed for the next major version. The minimum supported Go version follows the separate release-significance rule below. Other declared consumer environment and build-mode support is part of the stable compatibility contract unless an already-published support policy explicitly permits that support window to evolve within the major line. Pre-release iterations may change before their target release, but the eventual stable release must satisfy the compatibility rules of its version.

A change to an exported name, signature, behavior, or documented public contract should include the tests and public documentation needed to make the new contract clear. Avoid unrelated public API changes in the same contribution.

## Commit messages

Commandment uses [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) as its base grammar, with a closed repository-defined vocabulary. Normal commits that enter the permanent `main` history use one of these forms:

```text
type: subject
type(scope): subject
type!: subject
type(scope)!: subject
```

Reverts use the dedicated grammar in [Reverts](#reverts).

Type and scope are separate classification axes; the full header elements have distinct jobs:

| Element | Meaning |
| --- | --- |
| `type` | Why the commit exists. |
| `scope` | The primary durable Commandment responsibility that owns the change. |
| `!` | The commit intentionally breaks a public compatibility contract that Commandment promises to preserve within the current major version. |
| `subject` | A concise imperative description of the resulting change. |

Examples:

```text
feat(cli): add command replacement inspection
fix(clitest): preserve ambiguous locator results
refactor(shelltest): simplify bridge request ownership
perf(cli): reduce completion candidate allocations
test(cli): cover conditional command defaults
docs(examples): add dynamic command provider scenario
build: require Go 1.28
chore(repo): upgrade golangci-lint
```

### Types

Use exactly one normal type.

| Type | Use when the commit... | Typical example |
| --- | --- | --- |
| `feat` | adds or materially extends supported product behavior or a public capability | `feat(clitest): add command default expectations` |
| `fix` | corrects supported product or consumer build behavior that does not match its intended contract | `fix(cli): preserve provider command ordering` |
| `refactor` | changes production representation, structure, ownership, or supported product contract shape without adding a capability or correcting incorrect behavior; this includes public lifecycle changes, and incompatibility requires `!` | `refactor(shelltest): simplify bridge request ownership` |
| `perf` | improves product performance as the primary reason while otherwise preserving supported behavior; any necessary incompatibility requires `!` | `perf(cli): reduce schema projection allocations` |
| `test` | changes verification artifacts, required coverage, or the claims they encode, including tests, conformance suites, benchmarks, or fixtures, without changing product behavior | `test(cli): cover provider command ordering` |
| `docs` | changes documentation surfaces only while describing or projecting existing product behavior, implementation, public contract, or repository policy without changing the underlying behavior, contract, or policy; this includes README/guides, GoDoc and source comments, contributor documentation, and the executable examples corpus | `docs(examples): correct command provider scenario` |
| `build` | changes or maintains the consumer build contract of the module or a shipped package, such as minimum supported Go version, consumer-relevant module metadata, or consumer dependency requirements, when the commit is not correcting a defect or delivering product behavior | `build: require Go 1.28` |
| `chore` | maintains repository-owned infrastructure, metadata, policy, contributor workflow, or quality-gate orchestration, or performs a deliberate repository-wide mechanical normalization, without changing released product behavior, the consumer build contract, consumer-facing product documentation semantics, or required product verification coverage or claims | `chore(repo): upgrade golangci-lint` |

### Release significance

Commit classification records release significance alongside change intent. Release significance may come from the type, compatibility or lifecycle metadata, consumer requirement changes, or—when reverting released work—the resulting consumer-visible state. It does not require Commandment to publish a release for every qualifying commit, and Commandment does not derive release versions automatically from commit history. Beyond the base `fix`, `feat`, and breaking-change mapping, the `perf`, deprecation, and consumer-support signals below are deliberate Commandment release policy layered on the pinned standards.

| Commit signal | Minimum significance when included in a release |
| --- | --- |
| `fix` | Patch. |
| `perf` | Patch. |
| `feat` | Minor. |
| `DEPRECATED:` footer | Minor, regardless of the ordinary type. |
| Change the minimum supported Go version | Minor, regardless of the ordinary type. |
| Expand declared consumer environment or build-mode support | Minor, regardless of the ordinary type. |
| Narrow or remove declared consumer environment or build-mode support other than the minimum supported Go version | Major unless an already-published support policy explicitly permits that change within the current major version; when it does, follow that policy's release rule. |
| Increase an existing consumer dependency requirement within its patch line | Patch, regardless of the ordinary type. |
| Add a consumer dependency or raise an existing requirement across a minor or major line | Minor, regardless of the ordinary type. |
| Lower or remove a consumer dependency requirement | No independent release level by itself. |
| `!` with the required `BREAKING CHANGE:` footer | Major, regardless of the ordinary type or other signals. |
| Ordinary `refactor`, `test`, `docs`, `chore`, or routine `build` | Patch. |

Within a commit and across the changes that survive into a release, use the highest resulting significance: major outranks minor, and minor outranks patch. Breaking significance is independent of type. In particular, changing Commandment's minimum supported Go version is normally a `build` change and does not by itself require a new module major version. Narrowing other declared environment or build-mode support is breaking unless the already-published support policy for that support surface explicitly permits the change within the current major version; when it does, follow that policy's stated release treatment. Use `!` whenever the resulting change breaks a compatibility contract that Commandment promises to preserve within the current major version.

Release significance follows consumer impact even when the commit type is determined by another primary reason. For example, a `fix(cli)` that necessarily changes the minimum supported Go version remains a `fix(cli)` but carries at least minor release significance because the Go requirement also changed. Consumer dependency movement is likewise an independent release signal: a patch-line requirement increase may remain patch-significant, while adding a dependency or raising an existing requirement across a minor or major line is at least minor-significant. Lowering or removing a dependency requirement does not force a higher release level by itself. These consumer-requirement rules apply to the shipped main module and its declared support contract. Development-only requirements in `tools.mod`, including its `go` directive and tool dependencies, are repository workflow and do not create product release significance.

Release significance determines the version floor only when a release is actually cut; no individual commit requires immediate publication. Repository-only or verification-only work may wait for a later product release. If a release is cut and no surviving change establishes a minor or major floor, use a patch release.

Release significance applies to changes that survive into the released state. If a change is fully reverted before any release includes it, neither the reverted change nor the restoring revert forces a release level solely from those transient commits. A partial revert is classified from the resulting state that does remain in the release.

`chore` is not a fallback for work that another type describes more precisely. In normal permanent history, `chore` requires the `repo` scope.

The following classifications are deliberately excluded:

| Do not use | Use instead |
| --- | --- |
| `style` | Fold formatting into the logical change; use `chore(repo)` only for a deliberate repository-wide mechanical normalization. |
| `cleanup` | `refactor` for production cleanup; `chore(repo)` for repository-only cleanup. |
| `ci` | Classify CI-file changes by semantics: `chore(repo)` for orchestration, or `test` under the truthful owner when required verification coverage or claims change. |

`revert` is a special operation rather than a normal type; see [Reverts](#reverts).

### Scopes

Scopes are a closed ownership vocabulary, not free-form directory or subsystem labels.

| Scope | Owns |
| --- | --- |
| `cli` | The `cli` package and the Commandment CLI model it implements. |
| `clitest` | The `clitest` semantic testing frontend. |
| `clibench` | The `clibench` benchmark helper package. |
| `shelltest` | The `shelltest` real-shell testing package. |
| `examples` | The executable public examples corpus. |
| `repo` | Repository-owned infrastructure, metadata, policy, development tooling, contributor workflow, and repository checks. |

Use a scope whenever one allowed scope clearly owns the logical change. Omit the scope only when the change genuinely belongs to Commandment as a whole or no single allowed scope truthfully owns an inseparable cross-cutting change.

| Situation | Classification |
| --- | --- |
| One allowed scope clearly owns the logical change | Use that scope. |
| Several files/packages change only because they support one owning change | Use the owning scope. |
| Several public packages are equal participants in one indivisible product change | Omit the scope. |
| The change is genuinely repository-wide | Omit the scope unless `repo` specifically owns the responsibility. |
| The work contains unrelated changes with different owners | Split the commit. |

The scope is an ownership label, not a directory list. A CLI correction that also updates conformance tests, `clitest` usage, examples, and documentation remains:

```text
fix(cli): preserve provider command ordering
```

Do not invent scopes from implementation details, semantic subsystems, or directories:

| Avoid | Use |
| --- | --- |
| `feat(completion): add native delegation` | `feat(cli): add native completion delegation` |
| `feat(cli/completion): add native delegation` | `feat(cli): add native completion delegation` |
| `fix(cli,clitest): ...` | Use the primary owning scope, or omit the scope only if both are equal owners. |
| `fix(cli+clitest): ...` | Use the primary owning scope, or omit the scope only if both are equal owners. |
| `fix(cli/clitest): ...` | Use the primary owning scope, or omit the scope only if both are equal owners. |
| any `misc` scope | Omit the scope for a genuinely cross-cutting change or split unrelated work. |

### Type and scope compatibility

Allowed tokens do not make every type/scope pair meaningful. Use only the combinations listed below for normal commits. Types that describe shipped product changes belong to shipped product responsibilities; repository-only work stays repository-only.

| Scope | Valid normal types |
| --- | --- |
| `cli` | `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build` |
| `clitest` | `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build` |
| `clibench` | `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build` |
| `shelltest` | `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build` |
| `examples` | `docs`, `test` |
| `repo` | `docs`, `test`, `chore` |
| no scope | Product-wide or inseparable cross-package `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, or `build` when no single allowed owner exists |

Repository infrastructure does not become a product feature or fix merely because it was broken or newly added. Use `chore(repo)` for repository automation, development tooling, lint/build targets, and similar maintainer infrastructure. Likewise, `chore(cli)`, `fix(repo)`, `feat(repo)`, and `perf(repo)` are not normal classifications. If one appears necessary, reconsider the primary reason and durable owner before committing.

### Type and scope selection

Classify a commit by its primary logical change, not by the set of files it modifies. **The reason for the change outranks the mechanism used to implement it.** Supporting tests, documentation, examples, generated projections, dependency changes, and mechanical adaptations should normally travel with the product change they prove or enable.

When more than one type appears plausible, choose the type from the primary reason the commit exists:

| Primary reason the commit exists | Type |
| --- | --- |
| Correct supported product or consumer build behavior that violated its intended contract | `fix` |
| Add or materially extend a supported capability | `feat` |
| Improve performance while preserving correct behavior | `perf` |
| Change production representation, structure, ownership, or supported product contract shape without adding or correcting behavior | `refactor` |
| Change verification artifacts, required coverage, or the claims they encode without changing product behavior | `test` |
| Change documentation surfaces only while describing or projecting existing product behavior, implementation, public contract, or repository policy | `docs` |
| Change or maintain the consumed module or package build contract, including minimum supported Go version, consumer-relevant module metadata, or consumer dependency requirements, without correcting a defect or another product reason dominating | `build` |
| Maintain repository-owned infrastructure, metadata, policy, contributor workflow, or quality-gate orchestration, including repository build automation | `chore` |

A dependency upgrade used to correct supported product or consumer build behavior is therefore normally part of `fix`, not automatically `build`. A large structural rewrite whose purpose is lower allocation or latency is `perf`, not `refactor`. Compatibility impact is recorded separately with `!` when required.

The examples below assume the described change is the primary logical change unless the row explicitly says it is a supporting projection of another change.

| Change | Correct classification | Why |
| --- | --- | --- |
| Add a test later for already-supported `cli` behavior | `test(cli): ...` | Verification is the whole change. |
| Add a new public `clitest` capability | `feat(clitest): ...` | `clitest` itself is the product being extended. |
| Add a new public `clibench` helper | `feat(clibench): ...` | The helper is a new package capability, not merely a benchmark file. |
| Optimize `cli` and update the benchmark proving it | `perf(cli): ...` | The benchmark supports the performance change. |
| Correct only an executable example | `docs(examples): ...` | The examples corpus is executable public documentation. |
| Update an example because a `cli` feature changed | `feat(cli): ...` | The example is a supporting projection of the owning feature. |
| Update an example because a `cli` bug was fixed | `fix(cli): ...` | The example is a supporting projection of the owning correction. |
| Repair or add repository-only automation | `chore(repo): ...` | Repository infrastructure does not acquire product release significance merely because it was broken or newly added. |
| Change contributor workflow, contribution policy, or permanent-history policy | `chore(repo): ...` | The repository policy itself changed; use `docs(repo)` only to explain an already-established policy. |
| Clarify existing contributor workflow or repository policy without changing it | `docs(repo): ...` | Documentation projects an already-established repository policy. |
| Change repository quality-gate orchestration without changing required verification coverage or claims | `chore(repo): ...` | Scheduling and gate wiring are repository policy; changes to required tests, conformance coverage or claims, benchmarks, or fixtures use `test` under their truthful owner. |
| Update development-only tooling | `chore(repo): ...` | The released product and consumer build contract are unchanged. |
| Update a shipped-module dependency solely to deliver a `cli` product fix | `fix(cli): ...` | The dependency is the mechanism; the CLI correction is why the commit exists. |
| Maintain a dependency whose shipped product use is owned only by `shelltest`, with no product reason or independent support-policy change | `build(shelltest): ...` | The consumed module requirement changed, but no product behavior reason dominates. |
| Change repository Make targets or build scripts | `chore(repo): ...` | Repository build automation is maintainer infrastructure, not the consumed module build contract. |
| Change only the preferred development Go toolchain without changing the minimum supported Go version | `chore(repo): ...` | A preferred development toolchain is repository workflow, not a consumer requirement. |
| Change only the tooling module's Go requirement | `chore(repo): ...` | The `tools.mod` requirement governs repository tooling, not consumers of Commandment. |
| Change the minimum supported Go version of the main module when that requirement change is the primary reason | `build: ...` | The consumer Go requirement changed; a product feature or fix that necessarily changes the Go floor keeps its product type. Use `!` only if Commandment separately promises that requirement for the current major version. |
| Deprecate an existing public API without adding or fixing behavior | `refactor(cli): ...` | Public lifecycle changed even though the capability remains available; a `DEPRECATED:` footer is required. |

Do not split one coherent correction into implementation, test, and documentation commits merely because different file classes are involved.

`docs` is appropriate only when documentation describes or projects existing product behavior, implementation, supported contract, or repository policy. If normative prose or GoDoc changes the supported public contract itself, classify the commit by that product impact even when the diff contains only documentation. If contributor documentation changes repository policy itself, use `chore(repo)`. File type never overrides semantic effect.

### Subject

These rules apply to normal commit subjects. Reverts preserve the original commit header under the dedicated grammar below.

Write the subject as a concise imperative description of the resulting change.

| Rule | Requirement |
| --- | --- |
| Voice | Use an imperative verb: `add`, `preserve`, `remove`, `simplify`, `document`. |
| Opening | Begin the sentence with a lowercase verb. |
| Content | Describe the resulting change, not the work performed or the implementation history. |
| Terminology | Use canonical Commandment terminology where applicable. |
| Punctuation | Do not end with a period. |
| Tracking metadata | Keep issue numbers, pull-request or task identifiers, and other tracking metadata out of the subject. Versions may appear when they are part of the durable change. |
| Length | Keep a normal commit header at or below 100 characters. Reverts have the narrow exact-preservation exception defined below. |
| Proper names | Preserve normal spelling for proper nouns and Go identifiers such as `PowerShell`, `Go`, and exported names. |

| Prefer | Avoid |
| --- | --- |
| `fix(cli): preserve command identity during replacement` | `fix(cli): fixed command identity` |
| `feat(clitest): add provenance expectations` | `feat(clitest): adds provenance expectations` |
| `refactor(shelltest): simplify bridge request ownership` | `refactor(shelltest): refactoring bridge code` |
| `fix(cli): preserve value resolution ordering` | `fix(cli): change value_resolution.go` |

Permanent history should describe the durable source or product change, not the engineering campaign that discovered or authorized it. A subject should remain useful to a maintainer who knows nothing about the issue, audit, milestone, or release program that produced the commit.

| Usually avoid in subjects | Prefer |
| --- | --- |
| `close`, `closure`, `finalize`, `complete` | Name the concrete behavior, representation, documentation, or repository change that reached completion. |
| `converge`, `recompose`, `remediate` | Name the resulting structure or behavior. |
| `audit`, `review`, `follow-up`, `phase` | Name the source change produced by that work. |
| `pre-v1`, `post-freeze`, milestone or task identifiers | Describe the durable repository state instead of the temporary program context. |
| generic `improve`, `update`, `change`, `cleanup` | State what became different and in what way. |

Use the diff-independence test: the subject should identify a meaningful changed concept and the nature of the change without requiring the reader to open the diff before the header says anything useful.

| Prefer | Avoid |
| --- | --- |
| `refactor(cli): preserve completion request and result types` | `refactor(cli): improve completion callbacks` |
| `docs(examples): use direct declaration callbacks` | `docs(examples): update example structure` |
| `chore(repo): remove obsolete lint exceptions` | `chore(repo): clean up linting` |
| `perf(cli): avoid repeated reference projection scans` | `perf(cli): improve reference performance` |

### Body

Separate the body from the subject with a blank line.

A Commandment commit normally has a body. Breaking changes, newly introduced deprecations, and reverts always require one. For every other commit, omit the body only when the header completely explains the change and no consequential reason, invariant, tradeoff, compatibility effect, or design decision would be lost.

| Commit case | Body expectation |
| --- | --- |
| `feat` | Expected. Explain the purpose or capability boundary that makes the new behavior belong. |
| `fix` | Expected. Record the failure mode or violated invariant and why the correction is right. |
| `refactor` | Expected. Explain why the representation, structure, ownership, or supported product contract shape changed and what behavior or compatibility remains preserved. |
| `perf` | Expected. Identify the avoidable work removed and any consequential tradeoff or preserved behavior. |
| `build` | Expected when consequential, especially for minimum supported Go versions, consumer-relevant module metadata, and consumer dependencies. |
| `test` | Optional when the missing proof is fully clear from the header. |
| `docs` | Optional when the documentation change is self-explanatory. |
| `chore` | Optional for routine repository maintenance; expected when changing contributor policy, repository gates, or other durable workflow. |
| Breaking change | Required. Explain the design reason in the body and the consumer migration in the required `BREAKING CHANGE:` footer. |
| New deprecation | Required. Explain why the surface is being deprecated; put the consumer migration in the required `DEPRECATED:` footer. |
| `revert` | Required by the revert form; identify the reverted commit by its full object ID and state why the revert is necessary. |

The body has a different job from the diff. The header says **what changed**. The body preserves **why the change exists, which invariant matters, or which non-obvious consequence a future maintainer must understand**. Write that rationale in durable product or repository terms; audit names, task identifiers, phases, and release-campaign context do not substitute for the reason. Do not enumerate files or narrate implementation steps that are already obvious from the patch.

Wrap body and footer prose to at most 100 characters per line when practical. Do not break URLs, identifiers, or other indivisible tokens merely to satisfy the width.

Before omitting the body, ask whether a future maintainer can understand why the change was made from the header and diff alone without reconstructing issue history, review discussion, or project context. If adding a body would only restate the header or patch, omit it.

```text
fix(cli): preserve provider completion ordering

Provider-produced candidates were reordered after normalization, which made
provider declaration order differ from static declaration order. Preserve the
accepted declaration order through candidate projection.
```

### Breaking changes

An intentionally incompatible change to a public compatibility contract that Commandment promises to preserve within the current major version requires **both** the `!` marker and a `BREAKING CHANGE:` footer.

```text
feat(cli)!: expose reusable resolution policies

Resolution policies need to be nameable and reusable in ordinary typed
application code. Replace the legacy construction carrier with concrete
ResolutionPolicy values so one policy model can be stored and passed directly.

BREAKING CHANGE: Remove the legacy resolution policy constructors.
Consumers must construct ResolutionPolicy using the supported policy
constructors.
```

Choose the type independently from compatibility impact:

| Reason for the change | Example breaking header |
| --- | --- |
| New capability with an incompatible contract | `feat(cli)!: expose reusable resolution policies` |
| Correctness fix that necessarily breaks a contract promised for the current major version | `fix(cli)!: correct invalid replacement semantics` |
| API simplification without new behavior | `refactor(cli)!: remove obsolete declaration carrier` |
| Performance work that necessarily breaks a protected public contract | `perf(cli)!: replace legacy completion callback contract` |
| A consumer build contract explicitly promised for the current major version is broken | `build!: remove supported legacy build mode` |

Do not use `!` because a change is large, invasive, or difficult to implement. It denotes a compatibility break. `perf` may take `!` when performance is genuinely the primary reason and an incompatible protected contract change is necessary to achieve it. Because `test`, `docs`, and `chore` are defined here as not changing the supported product contract, they do not take `!`; if one appears to need it, reclassify the commit by the actual product, build, or contract reason for the incompatibility.

### Lifecycle, issues, and other footers

Use footers for compatibility, lifecycle, issue, and other durable metadata rather than putting that information in the subject.

Separate the footer block from the body, or directly from the subject when the body is omitted, with a blank line. Use the canonical semantic and issue tokens below exactly as shown. When multiple footers are present, keep them in one contiguous footer block; start each new footer on a new line without an intervening blank line. Other standard Git-trailer-style footers may appear for durable authorship, compliance, or tooling metadata, but do not invent additional project-semantic footer tokens ad hoc.

| Footer | Use |
| --- | --- |
| `BREAKING CHANGE:` | State the incompatible public contract and the required consumer migration. Required with `!`. |
| `DEPRECATED:` | Identify consumer-facing public API or supported product usage that remains available but is now deprecated; state the supported migration or replacement when one exists. |
| `Fixes #...` | Link an issue closed by the commit. |
| `Refs #...` | Link relevant issue context without claiming the commit closes it. |

`DEPRECATED:` is reserved for a deprecation introduced by the commit; do not repeat it merely because changed documentation mentions a surface that was already deprecated. Any commit that newly deprecates consumer-facing public API or supported product usage requires a `DEPRECATED:` footer. That footer gives the commit at least minor release significance because it changes the supported public lifecycle even when the ordinary type would otherwise carry no higher release level. It must name the deprecated surface and the supported migration clearly enough that a future maintainer does not need the originating issue or pull request to understand what consumers should do. If there is no direct replacement, say so and state how consumers should stop relying on the deprecated surface.

```text
feat(cli): add direct resolution policy construction

Expose ResolutionPolicy values so policies can be named, stored, and reused
in ordinary typed application code.

DEPRECATED: legacy resolution policy constructors
Use the ResolutionPolicy constructors instead. The legacy constructors remain
supported for the current major version.
```

A pure deprecation of product API or supported product usage uses `refactor` when the public lifecycle changes without adding a capability or correcting behavior. If the deprecated surface is specifically part of the consumer build contract, use `build` instead because that type describes the changed contract more precisely:

```text
refactor(cli): deprecate legacy policy constructors

Legacy resolution policy construction remains supported, but ResolutionPolicy is now the
canonical reusable representation. Mark the legacy constructors deprecated so
consumers can migrate before any future removal.

DEPRECATED: legacy resolution policy constructors
Use the ResolutionPolicy constructors instead.
```

Issue metadata follows the same footer model:

```text
fix(cli): reject conflicting command defaults

A command declaration may establish only one default relationship. Reject
repeated conditional or unconditional declarations instead of allowing the
later declaration to replace the first.

Fixes #123
```

### Reverts

`revert` is reserved for reversing one complete prior commit. If only part of a prior commit is being undone, classify the new causal change under the normal type and scope rules instead of using the revert grammar.

A revert preserves the complete header of the reverted commit, identifies it by its full commit object ID, and always states why the revert is necessary. Because the `revert: ` or `revert!: ` prefix is added to that preserved header, a revert may exceed the normal 100-character header limit only by the amount required for exact preservation:

```text
revert: feat(cli): add command replacement inspection

This reverts commit 0123456789abcdef0123456789abcdef01234567.

The replacement inspection path retains state beyond the operation that owns
it, so restore the previous behavior until the lifecycle can be corrected
without extending application-wide retention.
```

A revert derives release significance from the consumer-visible released state it changes, not from the `revert` operation itself. Reverting work that never appeared in a release has no release significance by itself. Reverting an already-released non-breaking product correction or optimization normally carries patch release significance; reverting repository-only, test-only, or documentation-projection work normally has no release significance. If reverting already-released behavior breaks a compatibility contract that Commandment promises to preserve within the current major version, use `revert!:` and the normal breaking footer:

| Revert situation | Minimum release significance |
| --- | --- |
| Revert work that never appeared in a release | None by itself. |
| Revert an already-released non-breaking product correction or optimization | Normally patch. |
| Revert released functionality while preserving compatibility | Determine from the resulting product change. |
| Remove an already-released supported capability | Major; use `revert!:` and `BREAKING CHANGE:`. |
| Revert repository-only, test-only, or documentation-projection work | None unless the supported public contract changes. |

```text
revert!: feat(cli): add command replacement inspection

This reverts commit 0123456789abcdef0123456789abcdef01234567.

The replacement inspection implementation retains application-wide state beyond
the operation that owns it, and that lifetime cannot be corrected compatibly.
Remove the capability rather than preserve an observation with unsound custody.

BREAKING CHANGE: Remove command replacement inspection introduced by the
reverted commit. Consumers using replacement inspection must remove that
dependency before upgrading.
```

Do not classify a revert with another ordinary type or invent a scoped revert form. `revert:` and `revert!:` are the complete revert headers.

### Commit boundaries

Every commit that enters `main` must represent one coherent completed change and leave the repository in a valid state suitable for review, history, and bisection.

Atomicity is causal, not mechanical. Changes belong in one commit when they exist for one causal reason and form one sensible rollback unit. Sharing a type, scope, audit, workstream, implementation session, or release milestone does not make otherwise independent changes atomic.

| Question | Keep together when... | Split when... |
| --- | --- | --- |
| Why do the changes exist? | They have one causal reason. | They solve independent problems. |
| Would one normally be reverted without the other? | No; reverting one alone would leave the change incomplete or misleading. | Yes; each can be undone independently. |
| Are tests, docs, examples, or generated projections involved? | They exist to prove, explain, or project the owning change. | They are independently valuable changes with their own reason to exist. |
| Can one precise header describe the whole commit? | Yes, without joining unrelated claims. | No; an honest header requires `and` between independent changes or becomes vague. |
| Were the changes found by the same audit, milestone, or workstream? | That fact is irrelevant if one causal change owns them. | Shared discovery context is the only reason they are grouped. |
| Are several packages or directories touched? | One durable owner still causes the supporting changes. | The packages contain independent changes with different reasons or rollback units. |

| If... | Then... |
| --- | --- |
| Tests, docs, examples, or mechanical projections are required to complete one logical change | Include them in the same commit. |
| Two changes are independently complete and would deserve different subjects, types, owners, or rollback decisions | Split them. |
| A large change is still one coherent semantic change | Keep it together; size alone is not a reason to split. |
| The only distinction is package, directory, phase, or file type | Do not split on that basis alone. |
| A local scratch commit will be rebased or squashed before merge | It does not need to follow the permanent-history convention. |
| A commit will enter permanent `main` history | It must comply with this convention. |
| A pull request is squash-merged and its title becomes the commit subject | The PR title must follow the same header grammar, and the resulting squash commit must include any body or footers required by this convention. |

### Permanent history integration

Permanent `main` history is linear. Do not introduce merge commits; mechanical merge messages such as `Merge pull request #...` therefore never enter `main`.

| Contribution shape | Integration rule |
| --- | --- |
| One already-complete permanent-history commit | Fast-forward it when possible or rebase it when necessary; preserve its logical content and commit message when they remain accurate after integration. |
| Several independently coherent commits that each satisfy this convention | Rebase them and preserve the useful boundaries. |
| WIP, fixup, review, or construction commits | Squash or rewrite them into the causal commit or commits they belong to. |
| Pull request whose branch history is not itself worth preserving | Squash into the smallest honest set of causal commits. |
| Merge commit | Do not add it to `main`. |

Do not force every pull request into one commit. Preserve independently useful commits; remove construction history that has no durable value. The permanent history, not the contributor's scratch workflow, is the governed artifact.

### Initial publication and releases

| Situation | Rule |
| --- | --- |
| Initial repository-wide publication | Use an unscoped product commit such as `feat: introduce Commandment`. |
| A release requires no repository change | Tag the release; do not create an empty release commit. |
| A release requires an actual repository change | Classify the commit by the change itself. |
| A change exists only because a release is near | That timing does not change its type or scope. |

Releases are represented by Git tags. There is deliberately no `release` type or scope. Release-significance rules classify changes relative to an existing stable release; the initial `v1.0.0` publication establishes the baseline and is not derived from the root commit's `feat` signal.
