package cli

// SourceOption annotates the semantic provenance of a value producer without changing producer
// precedence or destination identity.
type SourceOption interface{ sourceOption() }

type sourceOptionKind uint8

const (
	sourceOptionUnknown sourceOptionKind = iota
	sourceOptionKey
	sourceOptionLocation
	sourceOptionText
)

type sourceOption struct {
	kind  sourceOptionKind
	value string
}

func (sourceOption) sourceOption() {}

type sourceSensitiveOption struct{}

func (sourceSensitiveOption) sourceOption() {}

// sourceConstructors is the stateless namespace behind Source.
type sourceConstructors struct{}

// Source provides source-annotation options for value, fact, completion, and related producers.
var Source sourceConstructors

// Key attaches a stable source key annotation to the producer.
func (sourceConstructors) Key(key string) SourceOption {
	option := sourceOption{
		kind:  sourceOptionKey,
		value: key,
	}
	return option
}

// Location attaches source-location text to the producer.
func (sourceConstructors) Location(location string) SourceOption {
	option := sourceOption{
		kind:  sourceOptionLocation,
		value: location,
	}
	return option
}

// Text attaches descriptive source text to the producer.
func (sourceConstructors) Text(text string) SourceOption {
	option := sourceOption{
		kind:  sourceOptionText,
		value: text,
	}
	return option
}

// Sensitive marks produced semantic values and provenance as sensitive so public projections
// preserve sensitivity while redacting representable protected value text where required.
func (sourceConstructors) Sensitive() sourceSensitiveOption { return sourceSensitiveOption{} }

func producerSource(sourceOptions []SourceOption) (semanticProducerSource, bool) {
	annotations := make([]sourceAnnotation, 0, len(sourceOptions))
	sensitive := false
	for _, option := range sourceOptions {
		switch option := option.(type) {
		case sourceOption:
			var kind sourceAnnotationKind
			switch option.kind {
			case sourceOptionKey:
				kind = sourceAnnotationKindKey
			case sourceOptionLocation:
				kind = sourceAnnotationKindLocation
			case sourceOptionText:
				kind = sourceAnnotationKindText
			default:
				return semanticProducerSource{}, false
			}
			annotations = append(annotations, acceptSourceAnnotation(kind, option.value))
		case sourceSensitiveOption:
			sensitive = true
		default:
			return semanticProducerSource{}, false
		}
	}
	return snapshotProducerSource(annotations, sensitive), true
}

func producerSourceWithSensitive(base semanticProducerSource, sensitive bool) semanticProducerSource {
	return snapshotProducerSource(base.Annotations(), base.Sensitive() || sensitive)
}
