package cli

func projectEffectiveSignalAction(model *effectiveModel, action signalActionDefinition) (SignalAction, error) {
	result, err := projectSignalActionBase(action)
	if err != nil {
		return SignalAction{}, err
	}
	if destination, present := action.SignalValue(); present {
		result.SignalValueID = destination
		sensitive, err := effectiveDestinationSensitive(model, destination)
		if err != nil {
			return SignalAction{}, err
		}
		result.SignalRedacted = sensitive
		result.Sensitive = result.Sensitive || sensitive
	}
	if destination, present := action.DelayValue(); present {
		result.DelayValueID = destination
		sensitive, err := effectiveDestinationSensitive(model, destination)
		if err != nil {
			return SignalAction{}, err
		}
		result.DelayRedacted = sensitive
		result.Sensitive = result.Sensitive || sensitive
	}
	return result, nil
}

func projectSignalActionBase(action signalActionDefinition) (SignalAction, error) {
	kind, ok := projectSignalActionKind(action.Kind())
	if !ok {
		return SignalAction{}, errInvalidModelTopology
	}
	result := SignalAction{
		Kind:      kind,
		Name:      action.Name(),
		Signal:    action.Signal(),
		Delayed:   action.Delayed(),
		Delay:     action.Delay(),
		ExitCode:  action.ExitCode(),
		Sensitive: action.Sensitive(),
	}
	if endpoint, present := action.Endpoint(); present {
		result.EndpointID = endpoint
	}
	return result, nil
}

func projectSignalSelector(selector signalSelectorDefinition) (SignalSelector, error) {
	kind, ok := projectSignalSelectorKind(selector.Kind())
	if !ok {
		return SignalSelector{}, errInvalidModelTopology
	}
	result := SignalSelector{
		Kind:            kind,
		Signals:         projectSemanticSignals(selector.Signals()),
		ExcludedSignals: projectSemanticSignals(selector.ExcludedSignals()),
		ExcludedSender:  selector.ExcludedSender(),
	}
	for _, origin := range selector.Origins() {
		if !origin.valid() {
			return SignalSelector{}, errInvalidModelTopology
		}
		result.Origins = append(result.Origins, origin)
	}
	for _, origin := range selector.ExcludedOrigins() {
		if !origin.valid() {
			return SignalSelector{}, errInvalidModelTopology
		}
		result.ExcludedOrigins = append(result.ExcludedOrigins, origin)
	}
	for _, phase := range selector.Phases() {
		if !phase.valid() {
			return SignalSelector{}, errInvalidModelTopology
		}
		result.Phases = append(result.Phases, phase)
	}
	return result, nil
}

func projectSignalSelectorKind(kind signalSelectorKind) (SignalSelectorKind, bool) {
	switch kind {
	case signalSelectorKindInterrupt:
		return SignalSelectorInterrupt, true
	case signalSelectorKindTerminate:
		return SignalSelectorTerminate, true
	case signalSelectorKindListed:
		return SignalSelectorListed, true
	case signalSelectorKindAsynchronous:
		return SignalSelectorAsynchronous, true
	case signalSelectorKindShutdown:
		return SignalSelectorShutdown, true
	default:
		return SignalSelectorUnknown, false
	}
}

func projectSignalActionKind(kind signalActionKind) (SignalActionKind, bool) {
	switch kind {
	case signalActionKindIgnore:
		return SignalActionIgnore, true
	case signalActionKindCancel:
		return SignalActionCancel, true
	case signalActionKindHandle:
		return SignalActionHandle, true
	case signalActionKindForward:
		return SignalActionForward, true
	case signalActionKindSend:
		return SignalActionSend, true
	case signalActionKindAbort:
		return SignalActionAbort, true
	case signalActionKindReraise:
		return SignalActionReraise, true
	case signalActionKindEmit:
		return SignalActionEmit, true
	default:
		return SignalActionUnknown, false
	}
}
