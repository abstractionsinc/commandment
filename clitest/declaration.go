package clitest

import (
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
)

// DeclarationValueLiteralLocator identifies a public ValueLiteral nested in declaration state.
// Literal expectations preserve Commandment sensitivity and redaction boundaries instead of treating
// presentation text as identity; the zero value is invalid.
type DeclarationValueLiteralLocator struct {
	_     [0]func()
	state *declarationValueLiteralLocatorState
}

// DeclarationExpectation is a declaration expectation whose underlying public fact
// is authoritative in Schema, Reference, and Inspection observations.
//
//nolint:iface // Distinct closed public expectation roles are part of the accepted clitest API.
type DeclarationExpectation interface {
	SchemaExpectation
	ReferenceExpectation
	InspectionExpectation
}

// StaticDeclarationExpectation is a declaration expectation applicable to the
// static Schema and detached Reference observations.
type StaticDeclarationExpectation interface {
	SchemaExpectation
	ReferenceExpectation
}

// EffectiveDeclarationExpectation is a declaration expectation applicable to the
// effective Reference and Inspection observations.
//
//nolint:iface // Distinct closed public expectation roles are part of the accepted clitest API.
type EffectiveDeclarationExpectation interface {
	ReferenceExpectation
	InspectionExpectation
}

// DeclarationIdentityLocator is the closed heterogeneous locator grammar for
// exact declaration identity, including Completion item declaration relations.
type DeclarationIdentityLocator interface{ declarationIdentityLocator() }

// Type returns a reusable declaration expectation that checks the public type exposed by the
// located declaration literal.
func (locator DeclarationValueLiteralLocator) Type(expected cli.ValueType) DeclarationExpectation {
	expectation := declarationValueLiteralExpectation{
		locator:   locator.state,
		property:  declarationValueLiteralExpectationType,
		valueType: expected,
	}
	return expectation
}

// Sensitive returns a reusable declaration expectation that checks the located declaration
// literal is marked sensitive. Expectation failures preserve Commandment redaction boundaries.
func (locator DeclarationValueLiteralLocator) Sensitive() DeclarationExpectation {
	expectation := declarationValueLiteralExpectation{
		locator:   locator.state,
		property:  declarationValueLiteralExpectationSensitive,
		sensitive: true,
	}
	return expectation
}

// Redacted returns a reusable declaration expectation that checks the located declaration
// literal is marked redacted without exposing the protected representation.
func (locator DeclarationValueLiteralLocator) Redacted() DeclarationExpectation {
	expectation := declarationValueLiteralExpectation{
		locator:  locator.state,
		property: declarationValueLiteralExpectationRedacted,
		redacted: true,
	}
	return expectation
}

// Text returns a declaration expectation that checks the public literal text when that text is
// safely observable. Failure reporting does not use the method to bypass Commandment
// sensitivity or redaction semantics.
func (locator DeclarationValueLiteralLocator) Text(expected string) DeclarationExpectation {
	expectation := declarationValueLiteralExpectation{
		locator:  locator.state,
		property: declarationValueLiteralExpectationText,
		text:     expected,
	}
	return expectation
}

const (
	commandEntryAmbiguityDetail = "matches entry-name IDs"
	actionEntryAmbiguityDetail  = "matches action-scoped selector IDs"
	flagAmbiguityDetail         = "matches flag-spelling flag IDs"
	flagSpellingAmbiguityDetail = "matches spelling-text IDs"
	argumentTermAmbiguityDetail = "matches term-locator IDs"
)

type declarationValueLiteralLocatorMode uint8

const (
	declarationValueLiteralLocatorInvalid declarationValueLiteralLocatorMode = iota
	declarationValueLiteralLocatorConditionOperand
	declarationValueLiteralLocatorFlagFixedValue
	declarationValueLiteralLocatorRuleOperand
	declarationValueLiteralLocatorShortcutDefaultValue
)

type declarationValueLiteralLocatorState struct {
	mode         declarationValueLiteralLocatorMode
	condition    *conditionExpressionLocatorState
	flagSpelling *flagSpellingLocatorState
	rule         *ruleLocatorState
	entry        *commandEntryLocatorState
	defaultIndex int
	index        int
}

type declarationValueLiteralExpectationProperty uint8

const (
	declarationValueLiteralExpectationInvalid declarationValueLiteralExpectationProperty = iota
	declarationValueLiteralExpectationType
	declarationValueLiteralExpectationSensitive
	declarationValueLiteralExpectationRedacted
	declarationValueLiteralExpectationText
)

type declarationValueLiteralExpectation struct {
	locator   *declarationValueLiteralLocatorState
	property  declarationValueLiteralExpectationProperty
	valueType cli.ValueType
	sensitive bool
	redacted  bool
	text      string
}

func declarationEffectiveBoolResult(status locatorResolutionStatus, subject, detail evaluationField, propertyName string, expected, actual bool) evaluationResult {
	expectedField := evaluationText(fmt.Sprintf("%s=%t", propertyName, expected))
	switch status {
	case locatorResolutionFound:
		if actual == expected {
			return provedResult()
		}
		return disprovedResult(subject, expectedField, evaluationText(fmt.Sprintf("%s=%t", propertyName, actual)))
	case locatorResolutionNotProvable:
		return notProvableResult(subject, expectedField, detail)
	case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subject, expectedField, detail)
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func equalFieldResult(subject, name, expected, actual evaluationField, equal bool) evaluationResult {
	if equal {
		return provedResult()
	}
	return disprovedResult(subject, property(name, expected), property(name, actual))
}

func missingResult(subject, expected, detail evaluationField, truncated bool) evaluationResult {
	if truncated {
		return notProvableResult(subject, expected, observed(semanticSubjectInspection, observationTruncated))
	}
	if emptyEvaluationField(detail) {
		detail = evaluationText("subject is absent")
	}
	return disprovedResult(subject, expected, detail)
}

func relationLocatorFailureOutcome(subject, expected, locator string, status locatorResolutionStatus) evaluationResult {
	switch status {
	case locatorResolutionAbsent:
		return disprovedResult(evaluationText(subject), evaluationText(expected), evaluationText(locator+" is absent"))
	case locatorResolutionNotProvable:
		return notProvableResult(evaluationText(subject), evaluationText(expected), evaluationText(locator+" is not provable"))
	case locatorResolutionAmbiguous:
		return disprovedResult(evaluationText(subject), evaluationText(expected), evaluationText(locator+" is ambiguous"))
	case locatorResolutionInvalid, locatorResolutionFound:
		return disprovedResult(evaluationText(subject), evaluationText(expected), evaluationText(locator+" is invalid"))
	default:
		return disprovedResult(evaluationText(subject), evaluationText(expected), evaluationText(locator+" did not resolve"))
	}
}

func safeDeclarationLiteralText(value cli.ValueLiteral, status locatorResolutionStatus, detail evaluationField, expected string, truncated bool) evaluationResult {
	if status != locatorResolutionFound {
		return missingResult(evaluationText("declaration literal"), evaluationText("text"), detail, truncated)
	}
	if value.Text == expected {
		return provedResult()
	}
	if value.Sensitive || value.Redacted {
		return disprovedResult(evaluationText("declaration literal"), evaluationText("matching text"), evaluationText("text=<redacted>"))
	}
	return disprovedResult(evaluationText("declaration literal"), property(evaluationText("text"), quotedTextField(expected)), property(evaluationText("text"), quotedTextField(value.Text)))
}

func resolveSequenceSchema(locator ValueSequenceLiteralLocator, schema *cli.Schema) (cli.ValueSequenceLiteral, locatorResolutionStatus, evaluationField) {
	v, s, d := resolveStaticDefaultSchema(locator.root, schema)
	if s != locatorResolutionFound {
		return cli.ValueSequenceLiteral{}, s, d
	}
	if v.Sequence == nil {
		return cli.ValueSequenceLiteral{}, locatorResolutionAbsent, evaluationText("sequence representation is absent")
	}
	return *v.Sequence, locatorResolutionFound, evaluationText("")
}

func resolveSequenceReference(locator ValueSequenceLiteralLocator, reference cli.Reference) (cli.ValueSequenceLiteral, locatorResolutionStatus, evaluationField) {
	v, s, d := resolveStaticDefaultReference(locator.root, reference)
	if s != locatorResolutionFound {
		return cli.ValueSequenceLiteral{}, s, d
	}
	if v.Sequence == nil {
		return cli.ValueSequenceLiteral{}, locatorResolutionAbsent, evaluationText("sequence representation is absent")
	}
	return *v.Sequence, locatorResolutionFound, evaluationText("")
}

func resolveKeyedSchema(locator ValueKeyedLiteralLocator, schema *cli.Schema) (cli.ValueKeyedLiteral, locatorResolutionStatus, evaluationField) {
	v, s, d := resolveStaticDefaultSchema(locator.root, schema)
	if s != locatorResolutionFound {
		return cli.ValueKeyedLiteral{}, s, d
	}
	if v.Keyed == nil {
		return cli.ValueKeyedLiteral{}, locatorResolutionAbsent, evaluationText("keyed representation is absent")
	}
	return *v.Keyed, locatorResolutionFound, evaluationText("")
}

func resolveKeyedReference(locator ValueKeyedLiteralLocator, reference cli.Reference) (cli.ValueKeyedLiteral, locatorResolutionStatus, evaluationField) {
	v, s, d := resolveStaticDefaultReference(locator.root, reference)
	if s != locatorResolutionFound {
		return cli.ValueKeyedLiteral{}, s, d
	}
	if v.Keyed == nil {
		return cli.ValueKeyedLiteral{}, locatorResolutionAbsent, evaluationText("keyed representation is absent")
	}
	return *v.Keyed, locatorResolutionFound, evaluationText("")
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveExpectedDeclarationIdentity(value DeclarationIdentityLocator, schema *cli.Schema, reference *cli.Reference, inspection cli.Inspection) (cli.DeclarationIdentity, locatorResolutionStatus, evaluationField) {
	var identity cli.DeclarationIdentity
	switch locator := value.(type) {
	case CommandEntryLocator:
		if locator.state == nil || locator.state.mode == commandEntryLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == commandEntryLocatorID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindCommandEntry,
				CommandEntry: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaCommandEntryLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindCommandEntry,
				CommandEntry: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceCommandEntryLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindCommandEntry,
				CommandEntry: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionCommandEntryLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind:         cli.DeclarationKindCommandEntry,
			CommandEntry: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail

	case ActionEntryLocator:
		if locator.state == nil || locator.state.mode == actionEntryLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == actionEntryLocatorID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind:        cli.DeclarationKindActionEntry,
				ActionEntry: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaActionEntryLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind:        cli.DeclarationKindActionEntry,
				ActionEntry: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceActionEntryLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind:        cli.DeclarationKindActionEntry,
				ActionEntry: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionActionEntryLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind:        cli.DeclarationKindActionEntry,
			ActionEntry: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail

	case FlagLocator:
		if locator.state == nil || locator.state.mode == flagLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == flagLocatorID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind: cli.DeclarationKindFlag,
				Flag: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaFlagLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind: cli.DeclarationKindFlag,
				Flag: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceFlagLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind: cli.DeclarationKindFlag,
				Flag: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionFlagLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind: cli.DeclarationKindFlag,
			Flag: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail

	case FlagSpellingLocator:
		if locator.state == nil || locator.state.mode == flagSpellingLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == flagSpellingLocatorID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindFlagSpelling,
				FlagSpelling: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaFlagSpellingLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindFlagSpelling,
				FlagSpelling: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceFlagSpellingLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindFlagSpelling,
				FlagSpelling: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionFlagSpellingLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind:         cli.DeclarationKindFlagSpelling,
			FlagSpelling: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail

	case ArgumentFormLocator:
		if locator.state == nil || locator.state.mode == argumentFormLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == argumentFormLocatorID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentForm,
				ArgumentForm: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaArgumentFormLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentForm,
				ArgumentForm: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceArgumentFormLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentForm,
				ArgumentForm: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionArgumentFormLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind:         cli.DeclarationKindArgumentForm,
			ArgumentForm: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail

	case ArgumentTermLocator:
		if locator.state == nil || locator.state.mode == argumentTermLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == argumentTermLocatorID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentTerm,
				ArgumentTerm: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaArgumentTermLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentTerm,
				ArgumentTerm: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceArgumentTermLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind:         cli.DeclarationKindArgumentTerm,
				ArgumentTerm: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionArgumentTermLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind:         cli.DeclarationKindArgumentTerm,
			ArgumentTerm: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail

	case EnvironmentInputLocator:
		if locator.state == nil || locator.state.mode == environmentInputLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == environmentInputLocatorID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindEnvironmentInput,
				EnvironmentInput: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaEnvironmentInputLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindEnvironmentInput,
				EnvironmentInput: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceEnvironmentInputLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindEnvironmentInput,
				EnvironmentInput: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionEnvironmentInputLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind:             cli.DeclarationKindEnvironmentInput,
			EnvironmentInput: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail

	case ValueDomainEntryLocator:
		if locator.state == nil || locator.state.mode == valueDomainEntryLocatorInvalid {
			return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
		}
		if locator.state.mode == valueDomainEntryLocatorByID {
			if locator.state.id == 0 {
				return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
			}
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindValueDomainEntry,
				ValueDomainEntry: locator.state.id,
			}
			return identity, locatorResolutionFound, evaluationText("")
		}
		if schema != nil {
			resolution := resolveSchemaValueDomainEntryLocator(locator.state, schema)
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindValueDomainEntry,
				ValueDomainEntry: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		if reference != nil {
			resolution := resolveReferenceValueDomainEntryLocator(locator.state, *reference)
			identity = cli.DeclarationIdentity{
				Kind:             cli.DeclarationKindValueDomainEntry,
				ValueDomainEntry: resolution.value.ID,
			}
			return identity, resolution.status, resolution.detail
		}
		resolution := resolveInspectionValueDomainEntryLocator(locator.state, inspection)
		identity = cli.DeclarationIdentity{
			Kind:             cli.DeclarationKindValueDomainEntry,
			ValueDomainEntry: resolution.value.ID,
		}
		return identity, resolution.status, resolution.detail
	default:
		return cli.DeclarationIdentity{}, locatorResolutionInvalid, observed(semanticSubjectDeclaration, observationLocatorDoesNotIdentify)
	}
}

func declarationResolvedComparison(status locatorResolutionStatus, subject, detail, expected, observed evaluationField, equal bool) evaluationResult {
	switch status {
	case locatorResolutionFound:
		if equal {
			return provedResult()
		}
		return disprovedResult(subject, expected, observed)
	case locatorResolutionNotProvable:
		return notProvableResult(subject, expected, detail)
	case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subject, expected, detail)
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func replacementAbsentField(replacement *cli.DeclarationReplacement) evaluationField {
	if replacement == nil {
		return evaluationText("replacement absent")
	}
	return evaluationText("replacement present")
}

func evaluateReplacementTargetDeprecated(status locatorResolutionStatus, subject, detail evaluationField, replacement *cli.DeclarationReplacement, expected bool) evaluationResult {
	expectedField := evaluationText(fmt.Sprintf("target deprecated during migration=%t", expected))
	if status != locatorResolutionFound {
		return declarationResolvedComparison(status, qualifySubject(subject, evaluationText("replacement target deprecated during migration")), detail, expectedField, evaluationText(""), false)
	}
	if replacement == nil {
		return disprovedResult(qualifySubject(subject, evaluationText("replacement target deprecated during migration")), expectedField, evaluationText("replacement is absent"))
	}
	actual := replacement.TargetDeprecatedDuringMigration
	return declarationResolvedComparison(locatorResolutionFound, qualifySubject(subject, evaluationText("replacement target deprecated during migration")), evaluationText(""), expectedField, evaluationText(fmt.Sprintf("target deprecated during migration=%t", actual)), actual == expected)
}

func declarationValueLiteralLabel(state *declarationValueLiteralLocatorState) evaluationField {
	if state == nil {
		return evaluationText("declaration literal")
	}
	switch state.mode {
	case declarationValueLiteralLocatorConditionOperand:
		return qualifySubject(conditionExpressionLabel(state.condition), subjectAt(semanticSubjectConditionOperand, state.index))
	case declarationValueLiteralLocatorFlagFixedValue:
		return qualifySubject(flagSpellingLocatorLabel(state.flagSpelling), semanticSubjectFixedValue)
	case declarationValueLiteralLocatorRuleOperand:
		return qualifySubject(ruleLocatorLabel(state.rule), subjectAt(semanticSubjectRuleOperandLiteral, state.index))
	case declarationValueLiteralLocatorShortcutDefaultValue:
		return qualifySubject(commandEntryLocatorLabel(state.entry), subjectAt(semanticSubjectShortcutDefaultValue, state.index))
	default:
		return evaluationText("declaration literal")
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveSchemaDeclarationValueLiteralLocator(state *declarationValueLiteralLocatorState, schema *cli.Schema) locatorResolution[cli.ValueLiteral] {
	var resolution locatorResolution[cli.ValueLiteral]
	label := declarationValueLiteralLabel(state)
	if state == nil || state.mode == declarationValueLiteralLocatorInvalid {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	switch state.mode {
	case declarationValueLiteralLocatorConditionOperand:
		parent := resolveSchemaConditionExpressionLocator(state.condition, schema)
		return resolveConditionOperandLiteral(state, parent)
	case declarationValueLiteralLocatorFlagFixedValue:
		resolved := resolveSchemaFlagSpellingLocator(state.flagSpelling, schema)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueLiteral]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		return resolveOptionalLiteral(label, resolved.value.Spelling.FixedValue, semanticSubjectFixedValue)
	case declarationValueLiteralLocatorRuleOperand:
		resolved := resolveSchemaRuleLocator(state.rule, schema)
		return resolveRuleOperandLiteral(state, resolved.status, resolved.detail, resolved.value.Operands)
	case declarationValueLiteralLocatorShortcutDefaultValue:
		locator := CommandShortcutDefaultLocator{
			entry: CommandEntryLocator{
				state: state.entry,
			},
			index: state.defaultIndex,
		}
		value, status, detail := resolveShortcutDefaultSchema(locator, schema)
		return resolveShortcutDefaultLiteral(state, label, status, detail, value.Values)
	default:
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveReferenceDeclarationValueLiteralLocator(state *declarationValueLiteralLocatorState, reference cli.Reference) locatorResolution[cli.ValueLiteral] {
	var resolution locatorResolution[cli.ValueLiteral]
	label := declarationValueLiteralLabel(state)
	if state == nil || state.mode == declarationValueLiteralLocatorInvalid {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	switch state.mode {
	case declarationValueLiteralLocatorConditionOperand:
		parent := resolveReferenceConditionExpressionLocator(state.condition, reference)
		return resolveConditionOperandLiteral(state, parent)
	case declarationValueLiteralLocatorFlagFixedValue:
		resolved := resolveReferenceFlagSpellingLocator(state.flagSpelling, reference)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueLiteral]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		return resolveOptionalLiteral(label, resolved.value.Spelling.FixedValue, semanticSubjectFixedValue)
	case declarationValueLiteralLocatorRuleOperand:
		resolved := resolveReferenceRuleLocator(state.rule, reference)
		return resolveRuleOperandLiteral(state, resolved.status, resolved.detail, resolved.value.Operands)
	case declarationValueLiteralLocatorShortcutDefaultValue:
		locator := CommandShortcutDefaultLocator{
			entry: CommandEntryLocator{
				state: state.entry,
			},
			index: state.defaultIndex,
		}
		value, status, detail := resolveShortcutDefaultReference(locator, reference)
		return resolveShortcutDefaultLiteral(state, label, status, detail, value.Values)
	default:
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
}

//nolint:dupl // Typed semantic-family code intentionally remains direct; generic callback or mode abstractions would recreate the rejected universal proof machinery.
func resolveInspectionDeclarationValueLiteralLocator(state *declarationValueLiteralLocatorState, inspection cli.Inspection) locatorResolution[cli.ValueLiteral] {
	var resolution locatorResolution[cli.ValueLiteral]
	label := declarationValueLiteralLabel(state)
	if state == nil || state.mode == declarationValueLiteralLocatorInvalid {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
	switch state.mode {
	case declarationValueLiteralLocatorConditionOperand:
		parent := resolveInspectionConditionExpressionLocator(state.condition, inspection)
		return resolveConditionOperandLiteral(state, parent)
	case declarationValueLiteralLocatorFlagFixedValue:
		resolved := resolveInspectionFlagSpellingLocator(state.flagSpelling, inspection)
		if resolved.status != locatorResolutionFound {
			resolution = locatorResolution[cli.ValueLiteral]{
				status:  resolved.status,
				subject: label,
				detail:  resolved.detail,
			}
			return resolution
		}
		return resolveOptionalLiteral(label, resolved.value.Spelling.FixedValue, semanticSubjectFixedValue)
	case declarationValueLiteralLocatorRuleOperand:
		resolved := resolveInspectionRuleLocator(state.rule, inspection)
		return resolveRuleOperandLiteral(state, resolved.status, resolved.detail, resolved.value.Operands)
	case declarationValueLiteralLocatorShortcutDefaultValue:
		locator := CommandShortcutDefaultLocator{
			entry: CommandEntryLocator{
				state: state.entry,
			},
			index: state.defaultIndex,
		}
		value, status, detail := resolveShortcutDefaultInspection(locator, inspection)
		return resolveShortcutDefaultLiteral(state, label, status, detail, value.Values)
	default:
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionInvalid,
			subject: label,
			detail:  observed(semanticSubjectLocator, observationInvalidLocator),
		}
		return resolution
	}
}

func resolveOptionalLiteral(label evaluationField, value *cli.ValueLiteral, subject semanticSubject) locatorResolution[cli.ValueLiteral] {
	var resolution locatorResolution[cli.ValueLiteral]
	if value == nil {
		resolution = locatorResolution[cli.ValueLiteral]{
			status:  locatorResolutionAbsent,
			subject: label,
			detail:  observed(subject, observationAbsent),
		}
		return resolution
	}
	resolution = locatorResolution[cli.ValueLiteral]{
		status:  locatorResolutionFound,
		value:   *value,
		subject: label,
	}
	return resolution
}

func declarationExistenceResult[T any](resolution locatorResolution[T], expected bool) evaluationResult {
	expectedField := evaluationText(fmt.Sprintf("exists=%t", expected))

	switch resolution.status {
	case locatorResolutionFound:
		if expected {
			return provedResult()
		}
		return disprovedResult(resolution.subject, expectedField, evaluationText("exists=true"))
	case locatorResolutionAbsent:
		if !expected {
			return provedResult()
		}
		return disprovedResult(resolution.subject, expectedField, resolution.detail)
	case locatorResolutionNotProvable:
		return notProvableResult(resolution.subject, expectedField, resolution.detail)
	case locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(resolution.subject, expectedField, resolution.detail)
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func resolvedBoolProperty(status locatorResolutionStatus, subject, detail, expectedField evaluationField, expected, actual bool) evaluationResult {
	return declarationResolvedComparison(
		status,
		subject,
		detail,
		expectedField,
		evaluationBool(actual),
		actual == expected,
	)
}

func replacementRelationResult(subject locatorResolutionStatus, subjectField, subjectDetail evaluationField, replacement *cli.DeclarationReplacement, expected DeclarationIdentityLocator, schema *cli.Schema, reference *cli.Reference, inspection cli.Inspection) evaluationResult {
	if subject != locatorResolutionFound {
		return declarationResolvedComparison(subject, subjectField, subjectDetail, semanticRequirementRequestedReplacement, evaluationText(""), false)
	}
	if replacement == nil {
		return disprovedResult(subjectField, semanticRequirementRequestedReplacement, evaluationText("replacement is absent"))
	}

	identity, status, detail := resolveExpectedDeclarationIdentity(expected, schema, reference, inspection)
	switch status {
	case locatorResolutionFound:
		if replacement.Target == identity {
			return provedResult()
		}
		return disprovedResult(subjectField, semanticRequirementRequestedReplacement, evaluationText("replacement points to a different declaration identity"))
	case locatorResolutionNotProvable:
		return notProvableResult(subjectField, semanticRequirementRequestedReplacement, detail)
	case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subjectField, semanticRequirementRequestedReplacement, detail)
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func relatedCommandResult(subject locatorResolutionStatus, subjectField, subjectDetail evaluationField, target locatorResolution[cli.CommandSchema], actual cli.CommandID) evaluationResult {
	if subject != locatorResolutionFound {
		return declarationResolvedComparison(subject, subjectField, subjectDetail, evaluationText("resolvable target command"), evaluationText(""), false)
	}
	switch target.status {
	case locatorResolutionFound:
		if actual == target.value.ID {
			return provedResult()
		}
		return disprovedResult(subjectField, evaluationText("requested target command"), evaluationText("entry targets a different command identity"))
	case locatorResolutionNotProvable:
		return notProvableResult(subjectField, evaluationText("resolvable target command"), target.detail)
	case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subjectField, evaluationText("resolvable target command"), target.detail)
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func relatedReferenceCommandResult(subject locatorResolutionStatus, subjectField, subjectDetail evaluationField, target locatorResolution[cli.CommandReference], actual cli.CommandID) evaluationResult {
	if subject != locatorResolutionFound {
		return declarationResolvedComparison(subject, subjectField, subjectDetail, evaluationText("resolvable target command"), evaluationText(""), false)
	}
	switch target.status {
	case locatorResolutionFound:
		if actual == target.value.ID {
			return provedResult()
		}
		return disprovedResult(subjectField, evaluationText("requested target command"), evaluationText("entry targets a different command identity"))
	case locatorResolutionNotProvable:
		return notProvableResult(subjectField, evaluationText("resolvable target command"), target.detail)
	case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subjectField, evaluationText("resolvable target command"), target.detail)
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func relatedInspectionCommandResult(subject locatorResolutionStatus, subjectField, subjectDetail evaluationField, target locatorResolution[cli.CommandInspection], actual cli.CommandID) evaluationResult {
	if subject != locatorResolutionFound {
		return declarationResolvedComparison(subject, subjectField, subjectDetail, evaluationText("resolvable target command"), evaluationText(""), false)
	}
	switch target.status {
	case locatorResolutionFound:
		if actual == target.value.ID {
			return provedResult()
		}
		return disprovedResult(subjectField, evaluationText("requested target command"), evaluationText("entry targets a different command identity"))
	case locatorResolutionNotProvable:
		return notProvableResult(subjectField, evaluationText("resolvable target command"), target.detail)
	case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subjectField, evaluationText("resolvable target command"), target.detail)
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func lastCommandID(path []cli.CommandID) cli.CommandID {
	if len(path) == 0 {
		return 0
	}
	return path[len(path)-1]
}

func policyInspectionNamespaceIncomplete(command cli.CommandInspection, inspection cli.Inspection) (evaluationField, bool) {
	if inspection.Truncated {
		return observed(semanticSubjectInspection, observationTruncated), true
	}
	if command.NamespaceCompleteness != cli.CommandNamespaceComplete {
		return evaluationText(fmt.Sprintf("command namespace is %v", command.NamespaceCompleteness)), true
	}
	return nil, false
}

func exactPolicyCollectionInspection(command locatorResolution[cli.CommandInspection], inspection cli.Inspection, name string) (evaluationField, evaluationResult, bool) {
	if command.status != locatorResolutionFound {
		return nil, outcomeForValueStatus(command.status, command.subject, evaluationText(name+" exactly"), command.detail), false
	}
	label := qualifySubject(command.subject, evaluationText(name))
	if inspection.Truncated {
		return label, collectionClosureFailure(label, name, observed(semanticSubjectInspection, observationTruncated)), false
	}
	if command.value.NamespaceCompleteness != cli.CommandNamespaceComplete {
		detail := evaluationText(fmt.Sprintf("command namespace is %v", command.value.NamespaceCompleteness))
		return label, collectionClosureFailure(label, name, detail), false
	}
	return label, evaluationResult{}, true
}

func validateSchemaPolicyParent(parent *commandLocatorState, command cli.CommandSchema, schema *cli.Schema) bool {
	resolution := resolveSchemaCommandLocator(parent, schema)
	return resolution.status == locatorResolutionFound && resolution.value.ID == command.ID
}

func validateReferencePolicyParent(parent *commandLocatorState, command cli.CommandReference, reference cli.Reference) bool {
	resolution := resolveReferenceCommandLocator(parent, reference)
	return resolution.status == locatorResolutionFound && resolution.value.ID == command.ID
}

func validateInspectionPolicyParent(parent *commandLocatorState, command cli.CommandInspection, inspection cli.Inspection, label, expected evaluationField) (evaluationResult, bool) {
	resolution := resolveInspectionCommandLocator(parent, inspection)
	if resolution.status == locatorResolutionNotProvable {
		return notProvableResult(label, expected, resolution.detail), false
	}
	if resolution.status != locatorResolutionFound || resolution.value.ID != command.ID {
		return disprovedResult(label, expected, evaluationText("policy locator belongs to a different or invalid command")), false
	}
	return evaluationResult{}, true
}

func compareSuppressionStatesSchema(command cli.CommandSchema, states []*noticeSuppressionLocatorState, schema *cli.Schema, label evaluationField) evaluationResult {
	if len(command.NoticeSuppressions) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("notice suppressions count %d", len(states))), evaluationText(fmt.Sprintf("notice suppressions count %d", len(command.NoticeSuppressions))))
	}
	for index, state := range states {
		if state == nil || state.index != index || !validateSchemaPolicyParent(state.command, command, schema) {
			return disprovedResult(label, evaluationText("ordered notice-suppression locators"), evaluationText(fmt.Sprintf("invalid notice-suppression locator at position %d", index)))
		}
	}
	return provedResult()
}

func compareSuppressionStatesReference(command cli.CommandReference, states []*noticeSuppressionLocatorState, reference cli.Reference, label evaluationField) evaluationResult {
	if len(command.NoticeSuppressions) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("notice suppressions count %d", len(states))), evaluationText(fmt.Sprintf("notice suppressions count %d", len(command.NoticeSuppressions))))
	}
	for index, state := range states {
		if state == nil || state.index != index || !validateReferencePolicyParent(state.command, command, reference) {
			return disprovedResult(label, evaluationText("ordered notice-suppression locators"), evaluationText(fmt.Sprintf("invalid notice-suppression locator at position %d", index)))
		}
	}
	return provedResult()
}

func compareSuppressionStatesInspection(command cli.CommandInspection, states []*noticeSuppressionLocatorState, inspection cli.Inspection, label evaluationField) evaluationResult {
	if len(command.NoticeSuppressions) != len(states) {
		return disprovedResult(label, evaluationText(fmt.Sprintf("notice suppressions count %d", len(states))), evaluationText(fmt.Sprintf("notice suppressions count %d", len(command.NoticeSuppressions))))
	}
	for index, state := range states {
		if state == nil || state.index != index {
			return disprovedResult(label, evaluationText("ordered notice-suppression locators"), evaluationText(fmt.Sprintf("invalid notice-suppression locator at position %d", index)))
		}
		if outcome, ok := validateInspectionPolicyParent(state.command, command, inspection, label, evaluationText("ordered notice-suppression locators")); !ok {
			return outcome
		}
	}
	return provedResult()
}

func expectedRelationFailure(status locatorResolutionStatus, subject, expected, detail evaluationField) evaluationResult {
	switch status {
	case locatorResolutionNotProvable:
		return notProvableResult(subject, expected, detail)
	case locatorResolutionAbsent, locatorResolutionAmbiguous, locatorResolutionInvalid:
		return disprovedResult(subject, expected, detail)
	case locatorResolutionFound:
		panic("clitest: found relation passed to failure helper")
	default:
		panic("clitest: invalid locator resolution status")
	}
}

func policyPropertyResult(status locatorResolutionStatus, subject, detail, property, expected, actual evaluationField, equal bool) evaluationResult {
	qualified := qualifySubject(subject, property)
	return declarationResolvedComparison(status, qualified, detail, expected, actual, equal)
}

func declarationIdentityRelationResult(status locatorResolutionStatus, subject, detail evaluationField, actual cli.DeclarationIdentity, expected DeclarationIdentityLocator, schema *cli.Schema, reference *cli.Reference, inspection cli.Inspection) evaluationResult {
	expectedField := evaluationText("valid target declaration")
	if status != locatorResolutionFound {
		return declarationResolvedComparison(status, subject, detail, expectedField, evaluationText(""), false)
	}
	identity, expectedStatus, expectedDetail := resolveExpectedDeclarationIdentity(expected, schema, reference, inspection)
	if expectedStatus != locatorResolutionFound {
		return expectedRelationFailure(expectedStatus, subject, expectedField, expectedDetail)
	}
	if actual == identity {
		return provedResult()
	}
	return disprovedResult(subject, expectedField, evaluationText("target has different declaration identity"))
}

func (declarationValueLiteralExpectation) schemaExpectation() {}

func (declarationValueLiteralExpectation) referenceExpectation() {}

func (declarationValueLiteralExpectation) inspectionExpectation() {}

func (expectation declarationValueLiteralExpectation) evaluateResolved(resolution locatorResolution[cli.ValueLiteral], truncated bool) evaluationResult {
	if expectation.property == declarationValueLiteralExpectationText {
		return safeDeclarationLiteralText(resolution.value, resolution.status, resolution.detail, expectation.text, truncated)
	}
	if resolution.status != locatorResolutionFound {
		return missingResult(evaluationText("declaration literal"), expectation.propertyLabel(), resolution.detail, truncated)
	}
	subject := evaluationText("declaration literal")
	switch expectation.property {
	case declarationValueLiteralExpectationType:
		field := valueTypeField{
			value: expectation.valueType,
		}
		observedField := valueTypeField{
			value: resolution.value.Type,
		}
		return equalFieldResult(subject, evaluationText("type"), field, observedField, resolution.value.Type == expectation.valueType)
	case declarationValueLiteralExpectationSensitive:
		return equalFieldResult(subject, semanticRequirementSensitive, evaluationBool(expectation.sensitive), evaluationBool(resolution.value.Sensitive), resolution.value.Sensitive == expectation.sensitive)
	case declarationValueLiteralExpectationRedacted:
		return equalFieldResult(subject, evaluationText("redacted"), evaluationBool(expectation.redacted), evaluationBool(resolution.value.Redacted), resolution.value.Redacted == expectation.redacted)
	default:
		panic("clitest: invalid declaration-literal expectation")
	}
}

func (expectation declarationValueLiteralExpectation) propertyLabel() evaluationField {
	switch expectation.property {
	case declarationValueLiteralExpectationType:
		return evaluationText("type")
	case declarationValueLiteralExpectationSensitive:
		return semanticRequirementSensitive
	case declarationValueLiteralExpectationRedacted:
		return evaluationText("redacted")
	case declarationValueLiteralExpectationText:
		return evaluationText("text")
	default:
		return evaluationText("declaration literal")
	}
}

func (expectation declarationValueLiteralExpectation) evaluateSchema(schema *cli.Schema) evaluationResult {
	return expectation.evaluateResolved(resolveSchemaDeclarationValueLiteralLocator(expectation.locator, schema), false)
}

func (expectation declarationValueLiteralExpectation) evaluateReference(reference cli.Reference, _ error) evaluationResult {
	return expectation.evaluateResolved(resolveReferenceDeclarationValueLiteralLocator(expectation.locator, reference), false)
}

func (expectation declarationValueLiteralExpectation) evaluateInspection(inspection cli.Inspection, _ error) evaluationResult {
	return expectation.evaluateResolved(resolveInspectionDeclarationValueLiteralLocator(expectation.locator, inspection), inspection.Truncated)
}

func (declarationIdentityField) evaluationField() {}
