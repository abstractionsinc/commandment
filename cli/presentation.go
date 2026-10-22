package cli

import (
	"context"
	"io"
	"os"

	"golang.org/x/term"
)

// Description contains short and long presentation text for an application, command, flag, action
// entry, or other authoring subject.
type Description struct {
	Short string

	Long string
}

// Example describes one consumer-facing command-line example and its optional explanatory text.
type Example struct {
	Description string
	CommandLine string
}

// presentation is declaration-owned immutable source presentation. Hidden is
// policy metadata; examples remain opaque strings rather than executable input.
type presentation struct {
	description Description
	examples    []Example
	hidden      bool
}

func snapshotPresentation(description Description, examples []Example, hidden bool) presentation {
	snapshot := presentation{
		description: description,
		examples:    append([]Example(nil), examples...),
		hidden:      hidden,
	}
	return snapshot
}

func (presentation presentation) Examples() []Example {
	return append([]Example(nil), presentation.examples...)
}

const (
	fallbackWidth        = 80
	maximumTerminalWidth = 120
)

type presentationDestination uint8

const (
	destinationUnknown presentationDestination = iota
	destinationStdoutInformational
	destinationStderrDiagnostic
)

type spanRole uint8

const (
	spanOrdinary spanRole = iota
	spanHeading
	spanError
	spanWarning
	spanLiteral
	spanPlaceholder
)

type presentationSpan struct {
	Text string
	Role spanRole
}

type blockKind uint8

const (
	blockParagraph blockKind = iota + 1
	blockSection
	blockUsageLines
	blockEntryList
	blockBulletList
	blockDiagnosticGroup
	blockHint
	blockLiteralLine
)

type presentationEntry struct {
	Label       string
	Description string
}

type block struct {
	Kind    blockKind
	Title   string
	Spans   []presentationSpan
	Lines   []string
	Entries []presentationEntry
	Items   []string
}

type document struct {
	Destination presentationDestination
	Blocks      []block
}

type profile struct {
	Width       int
	TTY         bool
	Styled      bool
	Destination presentationDestination
}

type renderProfileKey struct{}

func defaultProfile(destination presentationDestination) profile {
	defaultProfile := profile{
		Width:       fallbackWidth,
		Destination: destination,
	}
	return defaultProfile
}

func withProfile(ctx context.Context, profile profile) context.Context {
	return context.WithValue(ctx, renderProfileKey{}, normalizeProfile(profile))
}

func profileFromContext(ctx context.Context, destination presentationDestination) profile {
	if ctx != nil {
		if profile, ok := ctx.Value(renderProfileKey{}).(profile); ok {
			if profile.Destination == destinationUnknown {
				profile.Destination = destination
			}
			return normalizeProfile(profile)
		}
	}
	return defaultProfile(destination)
}

func profileForWriter(writer io.Writer, destination presentationDestination) profile {
	profile := defaultProfile(destination)
	fdw, ok := writer.(interface{ Fd() uintptr })
	if !ok {
		return profile
	}
	fd := int(fdw.Fd())
	if !term.IsTerminal(fd) {
		return profile
	}
	profile.TTY = true
	if width, _, err := term.GetSize(fd); err == nil && width > 0 {
		profile.Width = width
	}
	profile.Styled = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	return profile
}

func normalizeProfile(profile profile) profile {
	if profile.Width <= 0 {
		profile.Width = fallbackWidth
	}
	if !profile.TTY {
		profile.Styled = false
		profile.Width = fallbackWidth
	} else if profile.Width > maximumTerminalWidth {
		profile.Width = maximumTerminalWidth
	}
	return profile
}
