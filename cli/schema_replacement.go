package cli

func setGenericReplacement(result *Schema, source declarationReplacementIdentity, replacement *DeclarationReplacement) error {
	switch source.Kind() {
	case declarationReplacementKindCommandEntry:
		id := source.CommandEntry()
		value, ok := result.CommandEntries[id]
		if !ok {
			return errInvalidModelTopology
		}
		value.Replacement = replacement
		result.CommandEntries[id] = value
	case declarationReplacementKindFlag:
		id := source.Flag()
		value, ok := result.Flags[id]
		if !ok {
			return errInvalidModelTopology
		}
		value.Replacement = replacement
		result.Flags[id] = value
	case declarationReplacementKindFlagSpelling:
		id := source.FlagSpelling()
		value, ok := result.FlagSpellings[id]
		if !ok {
			return errInvalidModelTopology
		}
		value.Replacement = replacement
		result.FlagSpellings[id] = value
	case declarationReplacementKindArgumentForm:
		id := source.ArgumentForm()
		value, ok := result.ArgumentForms[id]
		if !ok {
			return errInvalidModelTopology
		}
		value.Replacement = replacement
		result.ArgumentForms[id] = value
	case declarationReplacementKindArgumentTerm:
		id := source.ArgumentTerm()
		value, ok := result.ArgumentTerms[id]
		if !ok {
			return errInvalidModelTopology
		}
		value.Replacement = replacement
		result.ArgumentTerms[id] = value
	case declarationReplacementKindEnvironmentInput:
		id := source.EnvironmentInput()
		value, ok := result.EnvironmentInputs[id]
		if !ok {
			return errInvalidModelTopology
		}
		value.Replacement = replacement
		result.EnvironmentInputs[id] = value
	default:
		return errInvalidModelTopology
	}
	return nil
}

func targetDeprecatedDuringMigration(scheme ReleaseScheme, source, target ReleaseLifecycle) bool {
	if !scheme.valid() || target.DeprecatedIn == "" {
		return false
	}
	start := source.DeprecatedIn
	if start == "" {
		start = source.RemovedIn
	}
	if start == "" {
		return false
	}
	cmpStart, err := scheme.compare(target.DeprecatedIn, start)
	if err != nil || cmpStart < 0 {
		return false
	}
	if source.RemovedIn == "" {
		return true
	}
	cmpEnd, err := scheme.compare(target.DeprecatedIn, source.RemovedIn)
	return err == nil && cmpEnd <= 0
}
