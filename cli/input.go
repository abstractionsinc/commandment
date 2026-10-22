package cli

// InputKind classifies the semantic source of invocation input.
type InputKind uint8

const (
	// InputKindUnknown is the zero value and does not identify an input-source class.
	InputKindUnknown InputKind = iota
	// InputKindArguments identifies explicit invocation argument input.
	InputKindArguments
	// InputKindEnvironment identifies input obtained from the invocation environment snapshot.
	InputKindEnvironment
	// InputKindShortcut identifies Commandment-generated input contributed by a command shortcut.
	InputKindShortcut
	// InputKindResponseFile identifies input decoded from an accepted response-file source.
	InputKindResponseFile
	// InputKindGenerated identifies other Commandment-generated input with retained provenance.
	InputKindGenerated
)

// InputSourceID identifies one source within the owning artifact or operation. The
// zero value is no source; it is not a cross-operation identity.
type InputSourceID struct{ value uint32 }

// InputRevision is an opaque revision token for verifying an exact input snapshot.
// The zero value is unspecified and cannot establish a repair precondition.
type InputRevision struct{ value string }

// Revision constructs an opaque InputRevision from token.
func Revision(token string) InputRevision {
	revision := InputRevision{
		value: token,
	}
	return revision
}

// InputOriginID identifies one provenance origin within the owning artifact. The
// zero value is no origin.
type InputOriginID struct{ value uint32 }

// InputPosition identifies a byte offset and, when available, one-based line and
// column presentation coordinates within an input source.
type InputPosition struct {
	Offset int
	Line   int
	Column int
}

// InputSpan is a half-open range in one input source entry. Source and Entry select
// the original snapshot; Start and End delimit the represented bytes.
type InputSpan struct {
	Source InputSourceID
	Entry  int
	Start  InputPosition
	End    InputPosition
}

// InputSource describes one accepted invocation input snapshot. ID is artifact-local;
// Revision is the sole repair/staleness precondition; Key and Location are source
// metadata. Sensitive marks source content that library-generated presentation must
// not expose unsafely.
type InputSource struct {
	ID       InputSourceID
	Kind     InputKind
	Revision InputRevision

	CommandShortcut CommandEntryID

	Key       string
	Location  string
	Sensitive bool
}

// InputOrigin records one provenance occurrence and its parent ancestry. Parents
// preserve transformations such as nested response-file inclusion as a DAG rather
// than flattening source history.
type InputOrigin struct {
	ID        InputOriginID
	Span      InputSpan
	Parents   []InputOriginID
	Sensitive bool
}

// InputRepair is one complete optional repair alternative over verified original
// source snapshots. Commandment does not apply it. PreservesMeaning is true only
// when semantic equivalence is proven; Sensitive warns generic clients that verbatim
// preview may be unsafe.
type InputRepair struct {
	Title string

	PreservesMeaning bool
	Sensitive        bool

	Sources []InputRepairSource
}

// InputRepairSource groups all edits and copy preconditions for one original input
// source. A source may appear with no edits when it is required only for copying.
type InputRepairSource struct {
	Source InputSourceID

	TextEdits  []InputTextEdit
	EntryEdits []InputEntryEdit
}

// InputTextEdit replaces one non-overlapping span of an original source snapshot.
type InputTextEdit struct {
	Span        InputSpan
	Replacement InputReplacement
}

// InputEntryEdit replaces the half-open entry range [Start, End) with Entries.
type InputEntryEdit struct {
	Start int
	End   int

	Entries []InputReplacement
}

// InputReplacement is an ordered replacement assembled from safe literals and/or
// verbatim copies from verified original input spans.
type InputReplacement []InputReplacementPart

// InputReplacementPart contributes either Literal text or a verbatim Copy span to
// an InputReplacement. Copy does not request semantic decode/re-encode.
type InputReplacementPart struct {
	Literal string
	Copy    *InputSpan
}
