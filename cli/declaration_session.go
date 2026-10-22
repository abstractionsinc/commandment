package cli

import (
	"errors"
	"sync"
)

var errBuilderMisuse = errors.New("cli: builder misuse")

type declarationSessionState uint8

const (
	declarationSessionPrepared declarationSessionState = iota
	declarationSessionActive
	declarationSessionSuspended
	declarationSessionClosed
)

type declarationSession struct {
	mu   sync.Mutex
	uses sync.WaitGroup

	application *applicationDraft
	draft       *declarationDraft
	batch       *providerBatchDraft
	command     commandRef

	state declarationSessionState
	inUse bool
}

type builderAbort struct {
	cause error
}

func abortBuilder(cause error) {
	if cause == nil {
		panic("cli: nil builder abort")
	}
	panic(builderAbort{
		cause: cause,
	})
}

func panicBuilderMisuse() { abortBuilder(errBuilderMisuse) }

func (draft *applicationDraft) prepareSession(command commandRef) *declarationSession {
	if draft == nil {
		panic("cli: nil application draft")
	}
	if _, ok := draft.declarations.commands.resolve(&draft.declarations, command); !ok {
		panic("cli: invalid declaration command")
	}
	result := &declarationSession{
		application: draft,
		draft:       &draft.declarations,
		command:     command,
		state:       declarationSessionPrepared,
	}

	return result
}

func (session *declarationSession) prepareChild(command commandRef) *declarationSession {
	if session == nil {
		panic("cli: nil declaration session")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state != declarationSessionSuspended {
		panic("cli: invalid child declaration transition")
	}
	if session.draft == nil {
		panic("cli: suspended declaration session has no draft")
	}
	if _, ok := session.draft.commands.resolve(session.draft, command); !ok {
		panic("cli: invalid child declaration command")
	}
	result := &declarationSession{
		application: session.application,
		draft:       session.draft,
		batch:       session.batch,
		command:     command,
		state:       declarationSessionPrepared,
	}

	return result
}

func (session *declarationSession) activate() {
	if session == nil {
		panic("cli: nil declaration session")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state != declarationSessionPrepared {
		panic("cli: invalid declaration session transition")
	}
	if session.draft == nil {
		panic("cli: prepared declaration session has no draft")
	}
	session.state = declarationSessionActive
}

func (session *declarationSession) close() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	retainedUse := session.inUse
	session.state = declarationSessionClosed
	session.mu.Unlock()
	session.uses.Wait()
	session.mu.Lock()
	session.inUse = false
	session.application = nil
	session.draft = nil
	session.batch = nil
	session.command = commandRef{}
	session.mu.Unlock()
	return retainedUse
}

func (session *declarationSession) invoke(callback func()) (cause error) {
	session.activate()
	defer func() {
		recovered := recover()
		retainedUse := session.close()
		switch recovered := recovered.(type) {
		case nil:
		case builderAbort:
			cause = recovered.cause
		default:
			panic(recovered)
		}
		if cause == nil && retainedUse {
			cause = errBuilderMisuse
		}
	}()
	callback()
	return nil
}

func (session *declarationSession) acquire() (*declarationDraft, commandRef, func()) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	if session.state != declarationSessionActive || session.inUse {
		session.mu.Unlock()
		panicBuilderMisuse()
	}
	if session.draft == nil {
		session.mu.Unlock()
		panic("cli: active declaration session has no draft")
	}
	session.inUse = true
	session.uses.Add(1)
	draft := session.draft
	command := session.command
	session.mu.Unlock()
	release := func() {
		session.mu.Lock()
		session.inUse = false
		session.mu.Unlock()
		session.uses.Done()
	}
	return draft, command, release
}

func (session *declarationSession) use(callback func(*declarationDraft, commandRef)) {
	draft, command, release := session.acquire()
	defer release()
	callback(draft, command)
}

func (session *declarationSession) useCondition(callback func(*declarationDraft)) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	if (session.state != declarationSessionActive && session.state != declarationSessionSuspended) || session.inUse {
		session.mu.Unlock()
		panicBuilderMisuse()
	}
	if session.draft == nil {
		session.mu.Unlock()
		panic("cli: live declaration session has no draft")
	}
	session.inUse = true
	session.uses.Add(1)
	draft := session.draft
	session.mu.Unlock()
	defer func() {
		session.mu.Lock()
		session.inUse = false
		session.mu.Unlock()
		session.uses.Done()
	}()
	callback(draft)
}

func (session *declarationSession) useApplication(callback func(*applicationDraft, *declarationDraft, commandRef)) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	if session.state != declarationSessionActive || session.inUse {
		session.mu.Unlock()
		panicBuilderMisuse()
	}
	if session.application == nil || session.draft == nil {
		session.mu.Unlock()
		panicBuilderMisuse()
	}
	session.inUse = true
	session.uses.Add(1)
	application := session.application
	draft := session.draft
	command := session.command
	session.mu.Unlock()
	defer func() {
		session.mu.Lock()
		session.inUse = false
		session.mu.Unlock()
		session.uses.Done()
	}()
	callback(application, draft, command)
}

func (session *declarationSession) suspend() (*declarationDraft, commandRef, func()) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state != declarationSessionActive || session.inUse {
		panicBuilderMisuse()
	}
	if session.draft == nil {
		panic("cli: active declaration session has no draft")
	}
	session.state = declarationSessionSuspended
	draft := session.draft
	command := session.command
	resume := func() {
		session.mu.Lock()
		defer session.mu.Unlock()
		if session.state == declarationSessionSuspended {
			session.state = declarationSessionActive
		}
	}
	return draft, command, resume
}

func (draft *providerBatchDraft) prepareSession() *declarationSession {
	if draft == nil {
		panic("cli: nil provider batch draft")
	}
	if !draft.provider.present() || !draft.parent.present() || !draft.scope.valid() || draft.scope.operation == nil {
		panic("cli: invalid provider batch identity")
	}
	draft.declarations.scope = draft.scope
	if draft.declarations.identity == nil {
		draft.declarations.identity = &draftIdentity{}
	}
	result := &declarationSession{
		draft: &draft.declarations,
		batch: draft,
		state: declarationSessionPrepared,
	}

	return result
}

func (session *declarationSession) acquireProvider() (*declarationDraft, *providerBatchDraft, commandRef, func()) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	if session.state != declarationSessionActive || session.inUse {
		session.mu.Unlock()
		panicBuilderMisuse()
	}
	if session.draft == nil {
		session.mu.Unlock()
		panic("cli: active provider session has no declaration draft")
	}
	if session.batch == nil {
		session.mu.Unlock()
		panic("cli: active provider session has no provider batch")
	}
	if &session.batch.declarations != session.draft {
		session.mu.Unlock()
		panic("cli: provider session transaction mismatch")
	}
	session.inUse = true
	session.uses.Add(1)
	declarations := session.draft
	batch := session.batch
	command := session.command
	session.mu.Unlock()
	release := func() {
		session.mu.Lock()
		session.inUse = false
		session.mu.Unlock()
		session.uses.Done()
	}
	return declarations, batch, command, release
}

func (session *declarationSession) useProvider(callback func(*declarationDraft, *providerBatchDraft, commandRef)) {
	declarations, batch, command, release := session.acquireProvider()
	defer release()
	callback(declarations, batch, command)
}

func (session *declarationSession) suspendProvider() (*declarationDraft, *providerBatchDraft, func()) {
	if session == nil {
		panicBuilderMisuse()
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.state != declarationSessionActive || session.inUse {
		panicBuilderMisuse()
	}
	if session.draft == nil {
		panic("cli: active provider session has no declaration draft")
	}
	if session.batch == nil {
		panic("cli: active provider session has no provider batch")
	}
	if &session.batch.declarations != session.draft {
		panic("cli: provider session transaction mismatch")
	}
	session.state = declarationSessionSuspended
	declarations := session.draft
	batch := session.batch
	resume := func() {
		session.mu.Lock()
		defer session.mu.Unlock()
		if session.state == declarationSessionSuspended {
			session.state = declarationSessionActive
		}
	}
	return declarations, batch, resume
}
