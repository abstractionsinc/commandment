package cli

import (
	"maps"
	"slices"
)

type valueProducerKey struct {
	destination ValueID
	index       int
}

type completionCallbacks struct {
	flags     map[FlagID]completionCallback
	arguments map[ArgumentTermID]completionCallback
}

type valueDomainProvider = dynamicDomainEvaluator

type declarationCallbacks struct {
	valueProducers   map[valueProducerKey]valueProducer
	completions      completionCallbacks
	valueDomains     []valueDomainProvider
	commandProviders map[CommandProviderID]commandProviderFunc
}

func cloneDeclarationCallbacks(callbacks declarationCallbacks) declarationCallbacks {
	result := declarationCallbacks{
		valueProducers: maps.Clone(callbacks.valueProducers),
		completions: completionCallbacks{
			flags:     maps.Clone(callbacks.completions.flags),
			arguments: maps.Clone(callbacks.completions.arguments),
		},
		valueDomains:     slices.Clone(callbacks.valueDomains),
		commandProviders: maps.Clone(callbacks.commandProviders),
	}

	return result
}

func (callbacks *declarationCallbacks) valueProducer(destination ValueID, index int) (valueProducer, bool) {
	if !destination.present() || index < 0 {
		return nil, false
	}
	producer, ok := callbacks.valueProducers[valueProducerKey{
		destination: destination,
		index:       index,
	}]
	return producer, ok && producer != nil
}

func (callbacks *declarationCallbacks) flagCompletion(id FlagID) (completionCallback, bool) {
	callback, ok := callbacks.completions.flags[id]
	return callback, ok && callback.present()
}

func (callbacks *declarationCallbacks) argumentCompletion(id ArgumentTermID) (completionCallback, bool) {
	callback, ok := callbacks.completions.arguments[id]
	return callback, ok && callback.present()
}

func (callbacks *declarationCallbacks) valueDomain(index int) (valueDomainProvider, bool) {
	if index < 0 || index >= len(callbacks.valueDomains) {
		return nil, false
	}
	provider := callbacks.valueDomains[index]
	return provider, provider != nil
}

func (callbacks *declarationCallbacks) commandProvider(id CommandProviderID) (commandProviderFunc, bool) {
	provider, ok := callbacks.commandProviders[id]
	return provider, ok && provider != nil
}

func (callbacks *declarationCallbacks) merge(contribution declarationCallbacks) {
	if callbacks == nil {
		panic("cli: nil operation callback custody")
	}
	if len(contribution.valueProducers) != 0 {
		if callbacks.valueProducers == nil {
			callbacks.valueProducers = make(map[valueProducerKey]valueProducer, len(contribution.valueProducers))
		}
		for key, callback := range contribution.valueProducers {
			if _, exists := callbacks.valueProducers[key]; exists {
				panic("cli: duplicate value producer callback custody")
			}
			callbacks.valueProducers[key] = callback
		}
	}
	if len(contribution.completions.flags) != 0 {
		if callbacks.completions.flags == nil {
			callbacks.completions.flags = make(map[FlagID]completionCallback, len(contribution.completions.flags))
		}
		for id, callback := range contribution.completions.flags {
			if _, exists := callbacks.completions.flags[id]; exists {
				panic("cli: duplicate flag completion callback custody")
			}
			callbacks.completions.flags[id] = callback
		}
	}
	if len(contribution.completions.arguments) != 0 {
		if callbacks.completions.arguments == nil {
			callbacks.completions.arguments = make(map[ArgumentTermID]completionCallback, len(contribution.completions.arguments))
		}
		for id, callback := range contribution.completions.arguments {
			if _, exists := callbacks.completions.arguments[id]; exists {
				panic("cli: duplicate argument completion callback custody")
			}
			callbacks.completions.arguments[id] = callback
		}
	}
	callbacks.valueDomains = append(callbacks.valueDomains, contribution.valueDomains...)
	if len(contribution.commandProviders) != 0 {
		if callbacks.commandProviders == nil {
			callbacks.commandProviders = make(map[CommandProviderID]commandProviderFunc, len(contribution.commandProviders))
		}
		for id, callback := range contribution.commandProviders {
			if _, exists := callbacks.commandProviders[id]; exists {
				panic("cli: duplicate command provider callback custody")
			}
			callbacks.commandProviders[id] = callback
		}
	}
}

func (analysis *analysisState) valueProducerCallback(destination ValueID, index int) (valueProducer, bool) {
	if analysis == nil {
		return nil, false
	}
	return analysis.callbacks.valueProducer(destination, index)
}

func (analysis *analysisState) flagCompletionCallback(id FlagID) (completionCallback, bool) {
	if analysis == nil {
		return completionCallback{}, false
	}
	return analysis.callbacks.flagCompletion(id)
}

func (analysis *analysisState) argumentCompletionCallback(id ArgumentTermID) (completionCallback, bool) {
	if analysis == nil {
		return completionCallback{}, false
	}
	return analysis.callbacks.argumentCompletion(id)
}

func (analysis *analysisState) valueDomainCallback(index int) (valueDomainProvider, bool) {
	if analysis == nil {
		return nil, false
	}
	return analysis.callbacks.valueDomain(index)
}

func (analysis *analysisState) commandProviderCallback(id CommandProviderID) (commandProviderFunc, bool) {
	if analysis == nil {
		return nil, false
	}
	return analysis.callbacks.commandProvider(id)
}

func (operation *invocation) flagCompletionCallback(id FlagID) (completionCallback, bool) {
	if operation == nil {
		return completionCallback{}, false
	}
	operation.publication.mu.RLock()
	defer operation.publication.mu.RUnlock()
	return operation.analysis.flagCompletionCallback(id)
}

func (operation *invocation) argumentCompletionCallback(id ArgumentTermID) (completionCallback, bool) {
	if operation == nil {
		return completionCallback{}, false
	}
	operation.publication.mu.RLock()
	defer operation.publication.mu.RUnlock()
	return operation.analysis.argumentCompletionCallback(id)
}

func staticCallbacksValid(acceptance *declarationCandidate) bool {
	if acceptance == nil || acceptance.callbacks == nil {
		return false
	}
	callbacks := acceptance.callbacks

	producerCount := 0
	for _, destination := range acceptance.input.Destinations {
		for index, producer := range destination.Producers() {
			if producer.kind == declaredProducerKindDefault {
				continue
			}
			producerCount++
			callback, ok := callbacks.valueProducer(destination.meta.id, index)
			if !ok || callback == nil {
				return false
			}
		}
	}
	if len(callbacks.valueProducers) != producerCount {
		return false
	}

	flagCompletionCount := 0
	for _, flag := range acceptance.input.Flags {
		completion, ok := flag.Completion()
		if !ok || completion.Kind() == completionDeclarationKindNative {
			continue
		}
		flagCompletionCount++
		callback, ok := callbacks.flagCompletion(flag.meta.id)
		if !ok || !callback.validFor(completion.Kind()) {
			return false
		}
	}
	if len(callbacks.completions.flags) != flagCompletionCount {
		return false
	}

	argumentCompletionCount := 0
	for _, term := range acceptance.input.ArgumentTerms {
		completion, ok := term.Completion()
		if !ok || completion.Kind() == completionDeclarationKindNative {
			continue
		}
		argumentCompletionCount++
		callback, ok := callbacks.argumentCompletion(term.meta.id)
		if !ok || !callback.validFor(completion.Kind()) {
			return false
		}
	}
	if len(callbacks.completions.arguments) != argumentCompletionCount {
		return false
	}

	if len(callbacks.valueDomains) != len(acceptance.input.ValueDomains) {
		return false
	}
	for index, domain := range acceptance.input.ValueDomains {
		_, dynamic := domain.DynamicDefinition()
		callback, present := callbacks.valueDomain(index)
		if dynamic != present || dynamic && callback == nil {
			return false
		}
	}

	if len(callbacks.commandProviders) != len(acceptance.input.Providers) {
		return false
	}
	for _, provider := range acceptance.input.Providers {
		callback, ok := callbacks.commandProvider(provider.meta.id)
		if !ok || callback == nil {
			return false
		}
	}

	return true
}
