package cli

type applicationIdentity struct {
	_ byte
}

type operationIdentity struct {
	_ byte
}

type identityScope struct {
	application *applicationIdentity
	operation   *operationIdentity
}

func (scope identityScope) valid() bool {
	return scope.application != nil
}

func (scope identityScope) includes(other identityScope) bool {
	if !scope.valid() || !other.valid() || other.application != scope.application {
		return false
	}
	if other.operation == nil {
		return true
	}
	return scope.operation == other.operation
}
