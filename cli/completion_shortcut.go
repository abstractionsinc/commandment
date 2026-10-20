package cli

import "context"

type completionShortcutSettlement struct {
	changed        bool
	processedDelta int
	trailingAppend int
}

type completionShortcutState struct {
	trailingAppend                  int
	candidateSensitiveResponseFiles bool
}

type completionResponseFileLoadKey struct {
	path string
	base string
}

type completionResponseFileLoadResult struct {
	snapshot responseFileSnapshot
	err      error
}

type completionResponseFileLoadCache struct {
	values map[completionResponseFileLoadKey]completionResponseFileLoadResult
}

func (cache *completionResponseFileLoadCache) load(access ResponseFileAccess) analysisResponseFileLoad {
	baseLoad := responseFileLoad(access)
	if baseLoad == nil {
		return nil
	}
	if cache.values == nil {
		cache.values = make(map[completionResponseFileLoadKey]completionResponseFileLoadResult)
	}
	return func(ctx context.Context, path, base string) (responseFileSnapshot, error) {
		key := completionResponseFileLoadKey{
			path: path,
			base: base,
		}
		if cached, ok := cache.values[key]; ok {
			return cloneCompletionResponseFileSnapshot(cached.snapshot), cached.err
		}
		snapshot, err := baseLoad(ctx, path, base)
		cache.values[key] = completionResponseFileLoadResult{
			snapshot: cloneCompletionResponseFileSnapshot(snapshot),
			err:      err,
		}
		return cloneCompletionResponseFileSnapshot(snapshot), err
	}
}

func cloneCompletionResponseFileSnapshot(snapshot responseFileSnapshot) responseFileSnapshot {
	snapshot.Content = append([]byte(nil), snapshot.Content...)
	return snapshot
}

// settleCompletionShortcutPrefix settles fixed shortcut input around the
// incomplete cursor without changing lexical order. Response references in
// ordinary/prepended input and in the final fixed append suffix share one
// operation-local expansion session; the cursor itself remains unowned while
// the suffix is expanded and later analyzed under canonical target grammar.
func settleCompletionShortcutPrefix(ctx context.Context, snapshot *completionOperation, invocation *invocation) (completionShortcutState, error) {
	if !completionPrefixNeedsShortcutSettlement(snapshot) {
		return completionShortcutState{}, nil
	}
	expanded := make(map[InputOriginID]struct{})
	trailingAppend := 0
	for processed := 0; processed < snapshot.input.analysisCursorArgument; {
		stop := processed + 1
		settledStop, err := settleResponseRange(ctx, snapshot.invocation, snapshot.invocation.analysis, processed, stop, -1)
		delta := settledStop - stop
		snapshot.input.analysisCursorArgument += delta
		if err != nil {
			return completionShortcutState{
				trailingAppend: trailingAppend,
			}, err
		}
		processed = settledStop

		for {
			settlement, err := settleNextCompletionShortcut(
				ctx, snapshot, invocation, expanded, trailingAppend, processed,
			)
			if err != nil {
				return completionShortcutState{
					trailingAppend: trailingAppend,
				}, err
			}
			trailingAppend = settlement.trailingAppend
			if !settlement.changed {
				break
			}
			processed += settlement.processedDelta
		}
	}

	state := completionShortcutState{
		trailingAppend: trailingAppend,
	}
	if trailingAppend == 0 {
		return state, nil
	}

	args := snapshot.invocation.Args()
	tailStart := len(args) - trailingAppend
	if !completionShortcutTailHasResponseFileSensitiveToken(args[tailStart:]) {
		return state, nil
	}

	if completionShortcutTailNeedsCandidateSensitiveResponseFiles(ctx, snapshot, invocation, trailingAppend) {
		state.candidateSensitiveResponseFiles = true
		return state, nil
	}

	settledStop, err := settleResponseRange(ctx, snapshot.invocation, snapshot.invocation.analysis, tailStart, len(args), snapshot.input.analysisCursorArgument)
	if err != nil {
		return state, err
	}
	state.trailingAppend = settledStop - tailStart
	return state, nil
}

func completionPrefixNeedsShortcutSettlement(snapshot *completionOperation) bool {
	if snapshot == nil || snapshot.invocation == nil || snapshot.input.analysisCursorArgument <= 0 {
		return false
	}
	args := snapshot.invocation.Args()
	cursor := min(snapshot.input.analysisCursorArgument, len(args))
	if completionShortcutTailHasResponseFileSensitiveToken(args[:cursor]) {
		return true
	}
	return shortcutSettlementPossible(snapshot.invocation)
}

func completionShortcutTailNeedsCandidateSensitiveResponseFiles(ctx context.Context, snapshot *completionOperation, invocation *invocation, trailingAppend int) bool {
	if snapshot == nil || snapshot.invocation == nil || trailingAppend <= 0 ||
		snapshot.input.analysisCursorArgument < 0 || snapshot.input.analysisCursorArgument >= len(snapshot.invocation.Args()) {
		return false
	}
	args := snapshot.invocation.Args()
	tailStart := len(args) - trailingAppend
	if tailStart < 0 || !completionShortcutTailHasResponseFileSensitiveToken(args[tailStart:]) {
		return false
	}
	prefix := analyzeGrammarPrefix(
		ctx,
		snapshot.invocation, snapshot.input.analysisCursorArgument,
		invocation.namespaceFor(ctx), invocation.conditionFor(ctx), invocation.defaultCommandFor(ctx),
	)
	cursor := args[snapshot.input.analysisCursorArgument]
	if completionCursorMaySelectCommand(prefix, cursor) {
		invocation.namespace(ctx, grammarNamespaceDemand{
			Command: prefix.Current,
			Grammar: prefix.Grammar,
		})
		prefix = analyzeGrammarPrefix(
			ctx,
			snapshot.invocation, snapshot.input.analysisCursorArgument,
			invocation.namespaceFor(ctx), invocation.conditionFor(ctx), invocation.defaultCommandFor(ctx),
		)
	}
	syntax := analyzeCompletionSyntax(snapshot.invocation, prefix)
	model := snapshot.invocation.analysis.effectiveModel
	grammar, ok := model.EffectiveGrammar()
	if !ok {
		return false
	}
	matcher := prepareCompletionMatcher(snapshot.input, completionMatchQuery(snapshot))
	matches := func(value string) bool {
		_, matched := matcher.matchCandidate(CompletionCandidate{
			Value: value,
		}, nil)
		return matched
	}
	for _, candidate := range syntax.CommandOrder {
		if candidate.Kind != completionSyntaxCandidateCommandEntry {
			continue
		}
		for _, entry := range grammar.CommandEntries(prefix.Current) {
			if entry.id == candidate.CommandEntry && matches(entry.name) {
				return true
			}
		}
	}
	commands := append([]CommandID(nil), prefix.Grammar.Selection.Path...)
	if len(commands) == 0 || commands[len(commands)-1] != prefix.Current {
		commands = append(commands, prefix.Current)
	}
	for _, candidate := range syntax.FlagOrder {
		switch candidate.Kind {
		case completionSyntaxCandidateFlagAction:
			for _, command := range commands {
				for _, action := range grammar.ActionFlags(command) {
					if action.id == candidate.ActionEntry && matches(action.text) {
						return true
					}
				}
			}
		case completionSyntaxCandidateFlagSpelling:
			for _, command := range commands {
				for _, flag := range grammar.Flags(command) {
					profileSelector := false
					for _, profile := range model.GrammarProfilesForCommand(flag.command) {
						selector, present := profile.Selector()
						if present && selector == flag.destination {
							profileSelector = true
							break
						}
					}
					if !profileSelector {
						continue
					}
					for _, spelling := range flag.Spellings() {
						if spelling.id == candidate.FlagSpelling && matches(spelling.text) {
							return true
						}
					}
				}
			}
		default:
			continue
		}
	}
	return false
}

func completionShortcutTailHasResponseFileSensitiveToken(args []string) bool {
	for _, argument := range args {
		if len(argument) >= 2 && (argument[0] == '@' || (argument[0] == '\\' && argument[1] == '@')) {
			return true
		}
	}
	return false
}

func analyzeShortcutPendingFlagCompletion(ctx context.Context, snapshot *completionOperation, prefix grammarPrefixResult, invocation *invocation) (grammarPrefixResult, bool) {
	if prefix.PendingFlagValue == nil || snapshot.input.analysisCursorArgument < 0 ||
		snapshot.input.analysisCursorArgument >= len(snapshot.invocation.Args()) {
		return prefix, true
	}
	pending := *prefix.PendingFlagValue
	args := snapshot.invocation.Args()
	// Grammar needs one complete representative token at the value-owned cursor
	// in order to consume the immutable fixed suffix. The representative is
	// never converted or published; the real pending occurrence is restored
	// before the completion callback observes the request.
	args[snapshot.input.analysisCursorArgument] = "shortcut-completion-value"
	var namespace grammarNamespaceDemandFunc
	var condition grammarConditionDemandFunc
	var defaults grammarDefaultDemandFunc
	if invocation != nil {
		namespace = invocation.namespaceFor(ctx)
		condition = invocation.conditionFor(ctx)
		defaults = invocation.defaultCommandFor(ctx)
	}
	branch := analyzeGrammarArgumentsPrefixSkipping(
		ctx,
		snapshot.invocation,
		args,
		-1,
		namespace,
		condition,
		defaults,
	)
	cursor := snapshot.input.analysisCursorArgument
	if cursor >= len(branch.Grammar.Ownership) || len(branch.Grammar.Issues) != 0 ||
		len(branch.Grammar.IncompleteNamespaces) != 0 {
		return branch, false
	}
	owner := branch.Grammar.Ownership[cursor]
	if owner.Role != tokenRoleFlagValue ||
		owner.Flag != pending.Flag || owner.FlagSpelling != pending.Spelling ||
		owner.Command != pending.Command || owner.Destination != pending.Destination ||
		owner.Form != pending.Form || owner.Occurrence != pending.Occurrence {
		return branch, false
	}
	branch.PendingFlagValue = &pending
	return branch, true
}

// settleNextCompletionShortcut applies the first not-yet-expanded shortcut
// selected within the already-processed lexical frontier. Prefixes are
// inspected through the ordinary grammar analyzer, so shortcut recognition and
// the canonical target remain the same semantic selection operation.
func settleNextCompletionShortcut(ctx context.Context, snapshot *completionOperation, invocation *invocation, expanded map[InputOriginID]struct{}, trailingAppend, processed int) (completionShortcutSettlement, error) {
	cursor := snapshot.input.analysisCursorArgument
	if processed > cursor {
		processed = cursor
	}
	for boundary := 1; boundary <= processed; boundary++ {
		prefix := analyzeGrammarPrefix(
			ctx,
			snapshot.invocation, boundary, invocation.namespaceFor(ctx), invocation.conditionFor(ctx), invocation.defaultCommandFor(ctx),
		)
		if code, err := invocation.supplyFailure(); err != nil {
			return completionShortcutSettlement{}, operationDiagnosticFailure(
				snapshot.invocation,
				code,
				"invocation value supply failed",
				err,
			)
		}
		selection, ok := selectedShortcutFromPrefix(snapshot.invocation, prefix, expanded, boundary, cursor)
		if !ok {
			continue
		}
		residualEnd := len(snapshot.invocation.Args()) - trailingAppend
		prepend := selection.definition.Args()
		appendArgs := selection.definition.AppendArgs()
		_, err := snapshot.invocation.ApplyShortcut(
			selection.selectorIndex,
			residualEnd,
			selection.entry,
			selection.target,
			prepend,
			appendArgs,
			selection.definition.Defaults(),
			selection.definition.Source().Sensitive(),
		)
		if err != nil {
			panic("cli: accepted completion shortcut state is inconsistent")
		}
		expanded[selection.input] = struct{}{}
		cursor += len(prepend)
		processedDelta := len(prepend)

		// Fixed prepend argv is lexically before every later residual token, so
		// expand that generated range immediately through the shared response
		// session. Fixed append argv remains physically after the residual cursor;
		// settleCompletionShortcutPrefix expands the final trailing range only
		// after all shortcut splices are complete, while leaving the cursor unowned.
		prependStart := selection.selectorIndex + 1
		prependStop := prependStart + len(prepend)
		if len(prepend) != 0 {
			settledStop, expandErr := settleResponseRange(ctx, snapshot.invocation, snapshot.invocation.analysis, prependStart, prependStop, -1)
			delta := settledStop - prependStop
			cursor += delta
			processedDelta += delta
			if expandErr != nil {
				snapshot.input.analysisCursorArgument = cursor
				return completionShortcutSettlement{}, expandErr
			}
		}
		snapshot.input.analysisCursorArgument = cursor
		result := completionShortcutSettlement{
			changed:        true,
			processedDelta: processedDelta,
			trailingAppend: trailingAppend + len(appendArgs),
		}
		return result, nil
	}
	return completionShortcutSettlement{
		trailingAppend: trailingAppend,
	}, nil
}
