package clitest

import (
	"slices"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
)

// ExecutionAccess declares the exact public execution-access relationships and
// facilities expected on one command. Value relationships use ValueLocator
// rather than raw IDs. The zero value expects no execution access.
type ExecutionAccess struct {
	Values         []ValueLocator
	OptionalValues []ValueLocator
	ValueStates    []ValueLocator

	Locale           bool
	CommandSelection bool
	WorkingDirectory bool

	Stdin  bool
	Stdout bool
	Stderr bool
}

// ExecutionAccessExactly snapshots expected and checks the exact command-owned
// execution-access declaration exposed by Schema, Reference, or Inspection.
func (locator CommandLocator) ExecutionAccessExactly(expected ExecutionAccess) DeclarationExpectation {
	expectation := commandExecutionAccessExact{
		locator:  locator.state,
		expected: snapshotExpectedExecutionAccess(expected),
	}
	return expectation
}

func snapshotExpectedExecutionAccess(expected ExecutionAccess) *expectedExecutionAccessState {
	state := &expectedExecutionAccessState{
		values:           cloneValueLocatorStates(expected.Values),
		optionalValues:   cloneValueLocatorStates(expected.OptionalValues),
		valueStates:      cloneValueLocatorStates(expected.ValueStates),
		locale:           expected.Locale,
		commandSelection: expected.CommandSelection,
		workingDirectory: expected.WorkingDirectory,
		stdin:            expected.Stdin,
		stdout:           expected.Stdout,
		stderr:           expected.Stderr,
	}
	return state
}

type expectedExecutionAccessState struct {
	values         []*valueLocatorState
	optionalValues []*valueLocatorState
	valueStates    []*valueLocatorState

	locale           bool
	commandSelection bool
	workingDirectory bool
	stdin            bool
	stdout           bool
	stderr           bool
}

type commandExecutionAccessExact struct {
	locator  *commandLocatorState
	expected *expectedExecutionAccessState
}

func expectedExecutionAccessSchema(state *expectedExecutionAccessState, schema *cli.Schema) (cli.ExecutionAccess, locatorResolutionStatus, evaluationField) {
	values, status, detail := expectedTypedValueIDsSchema(state.values, schema)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	optional, status, detail := expectedTypedValueIDsSchema(state.optionalValues, schema)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	valueStates, status, detail := expectedTypedValueIDsSchema(state.valueStates, schema)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	return executionAccessFromExpectedState(state, values, optional, valueStates), locatorResolutionFound, evaluationText("")
}

func expectedExecutionAccessReference(state *expectedExecutionAccessState, reference cli.Reference) (cli.ExecutionAccess, locatorResolutionStatus, evaluationField) {
	values, status, detail := expectedTypedValueIDsReference(state.values, reference)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	optional, status, detail := expectedTypedValueIDsReference(state.optionalValues, reference)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	valueStates, status, detail := expectedTypedValueIDsReference(state.valueStates, reference)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	return executionAccessFromExpectedState(state, values, optional, valueStates), locatorResolutionFound, evaluationText("")
}

func expectedExecutionAccessInspection(state *expectedExecutionAccessState, inspection cli.Inspection) (cli.ExecutionAccess, locatorResolutionStatus, evaluationField) {
	values, status, detail := expectedTypedValueIDsInspection(state.values, inspection)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	optional, status, detail := expectedTypedValueIDsInspection(state.optionalValues, inspection)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	valueStates, status, detail := expectedTypedValueIDsInspection(state.valueStates, inspection)
	if status != locatorResolutionFound {
		return cli.ExecutionAccess{}, status, detail
	}
	return executionAccessFromExpectedState(state, values, optional, valueStates), locatorResolutionFound, evaluationText("")
}

func executionAccessFromExpectedState(state *expectedExecutionAccessState, values, optional, valueStates []cli.ValueID) cli.ExecutionAccess {
	access := cli.ExecutionAccess{
		Values:           values,
		OptionalValues:   optional,
		ValueStates:      valueStates,
		Locale:           state.locale,
		CommandSelection: state.commandSelection,
		WorkingDirectory: state.workingDirectory,
		Stdin:            state.stdin,
		Stdout:           state.stdout,
		Stderr:           state.stderr,
	}

	return access
}

func compareExecutionAccess(label evaluationField, expected, actual cli.ExecutionAccess) evaluationResult {
	base := qualifySubject(label, evaluationText("execution access"))
	if !slices.Equal(expected.Values, actual.Values) {
		return orderedCollectionMismatch(qualifySubject(base, evaluationText("Values")), "values", expected.Values, actual.Values)
	}
	if !slices.Equal(expected.OptionalValues, actual.OptionalValues) {
		return orderedCollectionMismatch(qualifySubject(base, evaluationText("OptionalValues")), "optional-values", expected.OptionalValues, actual.OptionalValues)
	}
	if !slices.Equal(expected.ValueStates, actual.ValueStates) {
		return orderedCollectionMismatch(qualifySubject(base, evaluationText("ValueStates")), "value-states", expected.ValueStates, actual.ValueStates)
	}
	if result, mismatch := compareTypedField(base, "Locale", evaluationText(strconv.FormatBool(expected.Locale)), evaluationText(strconv.FormatBool(actual.Locale)), expected.Locale == actual.Locale); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "CommandSelection", evaluationText(strconv.FormatBool(expected.CommandSelection)), evaluationText(strconv.FormatBool(actual.CommandSelection)), expected.CommandSelection == actual.CommandSelection); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "WorkingDirectory", evaluationText(strconv.FormatBool(expected.WorkingDirectory)), evaluationText(strconv.FormatBool(actual.WorkingDirectory)), expected.WorkingDirectory == actual.WorkingDirectory); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "Stdin", evaluationText(strconv.FormatBool(expected.Stdin)), evaluationText(strconv.FormatBool(actual.Stdin)), expected.Stdin == actual.Stdin); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "Stdout", evaluationText(strconv.FormatBool(expected.Stdout)), evaluationText(strconv.FormatBool(actual.Stdout)), expected.Stdout == actual.Stdout); mismatch {
		return result
	}
	if result, mismatch := compareTypedField(base, "Stderr", evaluationText(strconv.FormatBool(expected.Stderr)), evaluationText(strconv.FormatBool(actual.Stderr)), expected.Stderr == actual.Stderr); mismatch {
		return result
	}
	return provedResult()
}

func (commandExecutionAccessExact) schemaExpectation() {}

func (commandExecutionAccessExact) referenceExpectation() {}

func (commandExecutionAccessExact) inspectionExpectation() {}

func (expectation commandExecutionAccessExact) evaluateSchema(schema *cli.Schema) evaluationResult {
	command := resolveSchemaCommandLocator(expectation.locator, schema)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("execution access"), command.detail)
	}
	access, status, detail := expectedExecutionAccessSchema(expectation.expected, schema)
	if status != locatorResolutionFound {
		return outcomeForValueStatus(status, qualifySubject(command.subject, evaluationText("execution access")), evaluationText("resolvable expected access"), detail)
	}
	return compareExecutionAccess(command.subject, access, command.value.Access)
}

func (expectation commandExecutionAccessExact) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	command := resolveReferenceCommandLocator(expectation.locator, reference)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("execution access"), command.detail)
	}
	access, status, detail := expectedExecutionAccessReference(expectation.expected, reference)
	if status != locatorResolutionFound {
		return outcomeForValueStatus(status, qualifySubject(command.subject, evaluationText("execution access")), evaluationText("resolvable expected access"), detail)
	}
	return compareExecutionAccess(command.subject, access, command.value.Access)
}

func (expectation commandExecutionAccessExact) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	command := resolveInspectionCommandLocator(expectation.locator, inspection)
	if command.status != locatorResolutionFound {
		return outcomeForValueStatus(command.status, command.subject, evaluationText("execution access"), command.detail)
	}
	access, status, detail := expectedExecutionAccessInspection(expectation.expected, inspection)
	if status != locatorResolutionFound {
		return outcomeForValueStatus(status, qualifySubject(command.subject, evaluationText("execution access")), evaluationText("resolvable expected access"), detail)
	}
	return compareExecutionAccess(command.subject, access, command.value.Access)
}
