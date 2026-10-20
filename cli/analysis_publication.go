package cli

import "sync"

type publicationBarrier struct {
	mu sync.RWMutex
}

type publicationOperationError struct {
	operation string
	cause     error
}

func (err *publicationOperationError) Error() string {
	return "invocation: " + err.operation + ": " + err.cause.Error()
}

func (err *publicationOperationError) Unwrap() error { return err.cause }

type providerCommit struct {
	analysis     *analysisState
	identities   operationIDs
	publication  providerPublication
	callbacks    declarationCallbacks
	destinations []destinationAcceptance
	outcome      providerNamespaceOutcome
}

type dynamicDomainCommit struct {
	analysis    *analysisState
	identities  operationIDs
	publication dynamicDomainPublication
	outcome     dynamicDomainOutcome
}

func (operation *invocation) prepareProviderPublication(analysis *analysisState, owner CommandID, external []ValueID) (operationIDs, *publicationAcceptance, bool) {
	if operation == nil || analysis == nil || analysis.effectiveModel == nil || !owner.present() {
		return operationIDs{}, nil, false
	}
	if analysis == operation.analysis {
		operation.publication.mu.RLock()
		defer operation.publication.mu.RUnlock()
		if analysis != operation.analysis {
			return operationIDs{}, nil, false
		}
	}
	identities := analysis.ids.ForkDeclarations()
	scope, ok := analysis.effectiveModel.ProviderPublicationScope(owner, external)
	if !ok {
		return operationIDs{}, nil, false
	}
	acceptance := preparePublicationAcceptance(scope.Records, scope.Grammar, scope.Application, identities)
	return identities, acceptance, true
}

func (operation *invocation) prepareDynamicDomainPublication(analysis *analysisState, index int) (operationIDs, *publicationAcceptance, int, bool) {
	if operation == nil || analysis == nil || analysis.effectiveModel == nil || index < 0 {
		return operationIDs{}, nil, 0, false
	}
	if analysis == operation.analysis {
		operation.publication.mu.RLock()
		defer operation.publication.mu.RUnlock()
		if analysis != operation.analysis {
			return operationIDs{}, nil, 0, false
		}
	}
	identities := analysis.ids.ForkDeclarations()
	scope, localIndex, ok := analysis.effectiveModel.DynamicDomainPublicationScope(index)
	if !ok {
		return operationIDs{}, nil, 0, false
	}
	acceptance := preparePublicationAcceptance(scope.Records, scope.Grammar, scope.Application, identities)
	return identities, acceptance, localIndex, true
}

func applyProviderCommit(analysis *analysisState, commit providerCommit) {
	if analysis == nil || analysis != commit.analysis || analysis.effectiveModel == nil {
		panic("invocation: invalid provider commit target")
	}
	analysis.ids.CommitDeclarations(commit.identities)
	analysis.effectiveModel.CommitProvider(commit.publication)
	analysis.callbacks.merge(commit.callbacks)
}

func applyDynamicDomainCommit(analysis *analysisState, commit dynamicDomainCommit) {
	if analysis == nil || analysis != commit.analysis || analysis.effectiveModel == nil {
		panic("invocation: invalid dynamic-domain commit target")
	}
	analysis.ids.CommitDeclarations(commit.identities)
	analysis.effectiveModel.CommitDynamicDomain(commit.publication)
}

func (operation *invocation) publishProviderCommit(slot *providerNamespaceSlot, commit providerCommit) {
	if operation == nil || slot == nil || commit.analysis == nil {
		panic("invocation: invalid provider commit")
	}
	operation.publication.mu.Lock()
	defer operation.publication.mu.Unlock()
	if operation.analysis != commit.analysis {
		panic("invocation: provider commit parent is no longer current")
	}
	applyProviderCommit(commit.analysis, commit)
	for _, destination := range commit.destinations {
		destination.publish()
	}
	settleProviderSlot(slot, commit.outcome, false, nil)
}

func (operation *invocation) publishProviderCommitPrivate(analysis *analysisState, slot *providerNamespaceSlot, commit providerCommit) {
	if analysis == nil || slot == nil {
		panic("invocation: invalid private provider commit")
	}
	applyProviderCommit(analysis, commit)
	analysis.pendingDestinationAcceptances = append(analysis.pendingDestinationAcceptances, commit.destinations...)
	settleProviderSlot(slot, commit.outcome, false, nil)
}

func (operation *invocation) publishDynamicDomainCommit(slot *dynamicDomainSlot, commit dynamicDomainCommit) {
	if operation == nil || slot == nil || commit.analysis == nil {
		panic("invocation: invalid dynamic-domain commit")
	}
	operation.publication.mu.Lock()
	defer operation.publication.mu.Unlock()
	if operation.analysis != commit.analysis {
		panic("invocation: dynamic-domain commit parent is no longer current")
	}
	applyDynamicDomainCommit(commit.analysis, commit)
	settleDynamicDomainSlot(slot, commit.outcome, false, nil)
}

func (operation *invocation) publishDynamicDomainCommitPrivate(analysis *analysisState, slot *dynamicDomainSlot, commit dynamicDomainCommit) {
	if analysis == nil || slot == nil {
		panic("invocation: invalid private dynamic-domain commit")
	}
	applyDynamicDomainCommit(analysis, commit)
	settleDynamicDomainSlot(slot, commit.outcome, false, nil)
}
