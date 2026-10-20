package cli

import (
	"context"
	"errors"
)

var errInvalidValueContribution = errors.New("cli: invalid value contribution")

// ValueContribution is the result of an optional value computation. ContributeValue records a
// present value, including a Go zero value or nil; OmitValue records semantic absence. Its zero
// value is invalid callback output.
type ValueContribution[T any] struct {
	state contributionState
	value T
}

type contributionState uint8

const (
	contributionInvalid contributionState = iota
	contributionValue
	contributionOmitted
)

// ContributeValue returns an optional-computation result containing value, including when value
// is a Go zero value or nil.
func ContributeValue[T any](value T) ValueContribution[T] {
	return ValueContribution[T]{
		state: contributionValue,
		value: value,
	}
}

// OmitValue returns an optional-computation result representing semantic absence.
func OmitValue[T any]() ValueContribution[T] {
	return ValueContribution[T]{
		state: contributionOmitted,
	}
}

// ValueComputation computes one required typed value from read-only configuration and declared
// dependencies during invocation analysis.
type ValueComputation[C, T any] func(context.Context, *C, Dependencies) (T, error)

// OptionalValueComputation computes either a typed contribution or semantic absence from read-only
// configuration and declared dependencies.
type OptionalValueComputation[C, T any] func(context.Context, *C, Dependencies) (ValueContribution[T], error)

// valueComputation is the immutable fluent carrier for one required value computation, source
// annotations, and explicitly declared dependencies.
type valueComputation[C, T any] struct {
	compute      ValueComputation[C, T]
	dependencies []Dependency
	options      []SourceOption
}

// optionalValueComputation is the immutable fluent carrier for one optional value computation,
// source annotations, and explicitly declared dependencies.
type optionalValueComputation[C, T any] struct {
	compute      OptionalValueComputation[C, T]
	dependencies []Dependency
	options      []SourceOption
}

// From replaces the dependencies declared for this required value computation.
func (computation valueComputation[C, T]) From(dependencies ...Dependency) valueComputation[C, T] {
	computation.dependencies = append([]Dependency(nil), dependencies...)
	return computation
}

// From replaces the dependencies declared for this optional value computation.
func (computation optionalValueComputation[C, T]) From(dependencies ...Dependency) optionalValueComputation[C, T] {
	computation.dependencies = append([]Dependency(nil), dependencies...)
	return computation
}

// ComputedBy starts a required value computation with the supplied source annotations.
func (valueConstructors) ComputedBy[C, T any](compute ValueComputation[C, T], options ...SourceOption) valueComputation[C, T] {
	return stageValueComputation(compute, options...)
}

// OptionallyComputedBy starts an optional value computation whose callback must return ContributeValue
// or OmitValue.
func (valueConstructors) OptionallyComputedBy[C, T any](compute OptionalValueComputation[C, T], options ...SourceOption) optionalValueComputation[C, T] {
	return stageOptionalValueComputation(compute, options...)
}

func stageValueComputation[C, T any](compute ValueComputation[C, T], options ...SourceOption) valueComputation[C, T] {
	return valueComputation[C, T]{
		compute: compute,
		options: append([]SourceOption(nil), options...),
	}
}

func stageOptionalValueComputation[C, T any](compute OptionalValueComputation[C, T], options ...SourceOption) optionalValueComputation[C, T] {
	return optionalValueComputation[C, T]{
		compute: compute,
		options: append([]SourceOption(nil), options...),
	}
}

func (values *Values[C]) ComputedDefault[T any](destination destinationLike[C, T], computation valueComputation[C, T]) {
	state := destinationHandleOf(destination)
	erased, ok := eraseValueComputation(computation)
	directive := valueDirectiveDraft{
		kind:        valueDirectiveComputedDefault,
		computation: erased,
		valid:       state != nil && computation.compute != nil && ok,
	}
	stageValueDirective(
		values.session,
		values.profile,
		directive,
		state,
	)
}

func (values *Values[C]) OptionalComputedDefault[T any](destination destinationLike[C, T], computation optionalValueComputation[C, T]) {
	state := destinationHandleOf(destination)
	erased, ok := eraseOptionalValueComputation(computation)
	directive := valueDirectiveDraft{
		kind:        valueDirectiveOptionalComputedDefault,
		computation: erased,
		valid:       state != nil && computation.compute != nil && ok,
	}
	stageValueDirective(
		values.session,
		values.profile,
		directive,
		state,
	)
}

func (values *Values[C]) Derive[T any](storage *T, computation valueComputation[C, T]) Destination[C, T] {
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	erased, ok := eraseValueComputation(computation)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationDerived,
		storage:            storage,
		computation:        erased,
		binding:            binding,
		definition:         typedAssignmentDestinationDefinition[T](destinationDeclarationDerived),
		definitionComplete: ok,
		valid:              storage != nil && bindingOK && computation.compute != nil && ok,
	}
	state := stageDestination(
		values.session,
		values.profile,
		destinationValue,
	)
	return Destination[C, T]{handle: state}
}

func (values *Values[C]) OptionalDerive[T any](storage *T, computation optionalValueComputation[C, T]) Destination[C, T] {
	binding, bindingOK := bindConfigurationStorage(values.Config, storage)
	erased, ok := eraseOptionalValueComputation(computation)
	destinationValue := destinationDraft{
		kind:               destinationDeclarationOptionalDerived,
		storage:            storage,
		computation:        erased,
		binding:            binding,
		definition:         typedAssignmentDestinationDefinition[T](destinationDeclarationOptionalDerived),
		definitionComplete: ok,
		valid:              storage != nil && bindingOK && computation.compute != nil && ok,
	}
	state := stageDestination(
		values.session,
		values.profile,
		destinationValue,
	)
	return Destination[C, T]{handle: state}
}

type authoringValueEvaluator func(context.Context, any, Dependencies) (any, bool, error)

type authoringValueComputation struct {
	pending      []destinationDependency
	dependencies []authoringDestinationDependency
	source       semanticProducerSource
	evaluate     authoringValueEvaluator
}

func freezeAuthoringValueComputation(draft *declarationDraft, computation authoringValueComputation) (authoringValueComputation, bool) {
	if len(computation.pending) == 0 {
		return computation, true
	}
	dependencies, ok := freezeAuthoringDestinationDependencies(draft, computation.pending)
	if !ok {
		return authoringValueComputation{}, false
	}
	computation.pending = nil
	computation.dependencies = dependencies
	return computation, true
}

func eraseValueComputation[C, T any](computation valueComputation[C, T]) (authoringValueComputation, bool) {
	dependencies, ok := dependencyDescriptors(computation.dependencies)
	source, sourceOK := producerSource(computation.options)
	if !ok || !sourceOK || computation.compute == nil {
		return authoringValueComputation{}, false
	}
	result := authoringValueComputation{
		pending: dependencies,
		source:  source,
		evaluate: func(ctx context.Context, raw any, dependencies Dependencies) (any, bool, error) {
			config, ok := raw.(*C)
			if !ok || config == nil {
				panic("cli: invalid value computation configuration")
			}
			value, err := computation.compute(ctx, config, dependencies)
			if err != nil {
				return nil, false, err
			}
			return value, true, nil
		},
	}
	return result, true
}

func eraseOptionalValueComputation[C, T any](computation optionalValueComputation[C, T]) (authoringValueComputation, bool) {
	dependencies, ok := dependencyDescriptors(computation.dependencies)
	source, sourceOK := producerSource(computation.options)
	if !ok || !sourceOK || computation.compute == nil {
		return authoringValueComputation{}, false
	}
	result := authoringValueComputation{
		pending: dependencies,
		source:  source,
		evaluate: func(ctx context.Context, raw any, dependencies Dependencies) (any, bool, error) {
			config, ok := raw.(*C)
			if !ok || config == nil {
				panic("cli: invalid value computation configuration")
			}
			contribution, err := computation.compute(ctx, config, dependencies)
			if err != nil {
				return nil, false, err
			}
			value, present, valid := contributionResult(contribution)
			if !valid {
				return nil, false, errInvalidValueContribution
			}
			if !present {
				return nil, false, nil
			}
			return value, true, nil
		},
	}
	return result, true
}
