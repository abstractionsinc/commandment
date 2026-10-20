package conformance

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func inputFuzzInspect(t *testing.T, app cli.Application, request cli.InspectionRequest) (cli.Inspection, error) {
	t.Helper()
	return clitest.Inspect(t.Context(), app, request).Inspection()
}

var _ = errors.Is

func inputFuzzBoundedFuzzText(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}

func inputFuzzFuzzASCIIFragment(value string, maximum int, preserveEmpty bool) string {
	value = inputFuzzBoundedFuzzText(value, maximum)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 && !preserveEmpty {
		return "value"
	}
	return b.String()
}

type inputFuzzValueConfig struct{ Value string }

type inputFuzzVerboseLabelsTextConfig struct {
	Verbose int
	Labels  map[string]string
	Text    string
}

func FuzzTokenLanguageRuntimeProperties(f *testing.F) {
	f.Add("accepted", false)
	f.Add("rejected", true)
	f.Fuzz(func(t *testing.T, raw string, reject bool) {
		text := "v" + inputFuzzFuzzASCIIFragment(inputFuzzBoundedFuzzText(raw, 32), 32, true)
		type config = inputFuzzValueConfig
		calls := 0
		var received []string
		app, err := cli.Compile(cli.Definition[config]{
			Name: "tool",
			Declare: func(root *cli.CommandBuilder[config]) {
				value := root.Values.String(&root.Config.Value)
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
					form.Tokens(value, "value", cli.TokenLanguage[string]{
						Name:  "fuzz-token",
						Count: cli.Count.Exactly(1),
						Parse: func(_ context.Context, tokens []string) (string, error) {
							calls++
							received = append([]string(nil), tokens...)
							tokens[0] = "mutated"
							if reject {
								return "", cli.Rejection.Message(
									"fuzz-reject",
									"rejected",
									cli.Rejection.At(0, 0, len(text)),
								)
							}
							return strings.ToUpper(text), nil
						},
						Format: func(value string) []string { return []string{"canon-" + value} },
					})
					form.Literal("tail")
				})
			},
		})
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		inspection, inspectErr := inputFuzzInspect(t, app, cli.InspectionRequest{
			Args: []string{text, "tail"},
		})
		if calls != 1 || !slices.Equal(received, []string{text}) {
			t.Fatalf("parser execution = calls %d range %#v, want one exact owned token %q", calls, received, text)
		}
		if reject {
			if inspectErr == nil || !inputFuzzInspectionHasDiagnosticCode(inspection, cli.DiagnosticInvalidValue) {
				t.Fatalf("rejected Inspect() = err %v diagnostics %#v", inspectErr, inspection.Diagnostics)
			}
			return
		}
		if inspectErr != nil || inspection.Status != cli.InspectionStatusReady {
			t.Fatalf(
				"accepted Inspect() = status %v err %v diagnostics %#v",
				inspection.Status,
				inspectErr,
				inspection.Diagnostics,
			)
		}
		candidate := inputFuzzOnlyInspectionCandidate(t, inspection)
		if candidate.Representation == nil || candidate.Representation.Kind != cli.ValueRepresentationTokens ||
			!slices.Equal(candidate.Representation.Tokens, []string{"canon-" + strings.ToUpper(text)}) {
			t.Fatalf("candidate representation = %#v", candidate.Representation)
		}
	})
}

func FuzzInputProvenanceAndSemanticTokenProperties(f *testing.F) {
	type config = inputFuzzVerboseLabelsTextConfig
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Run: func(context.Context, *config) error {
			return nil
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ShortFlagClustering(cli.ShortFlagClusteringEnabled)
			root.Flags.Count(&root.Config.Verbose).Occurrences(cli.Count.ZeroOrMore()).Short('v')
			labels := root.Values.Keyed(&root.Config.Labels, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			root.Values.EnvironmentPrefix(labels, "APP_LABEL_")
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(&form.Config.Text, "text").Optional()
			})
		},
	})
	if err != nil {
		f.Fatalf("Compile() error = %v", err)
	}
	f.Add(3, "name", "alice")
	f.Add(0, "", "")
	f.Add(0, "0", "")
	f.Fuzz(func(t *testing.T, repeats int, suffix, value string) {
		repeats %= 9
		if repeats < 0 {
			repeats = -repeats
		}
		suffix = inputFuzzFuzzASCIIFragment(suffix, 24, true)
		value = inputFuzzFuzzASCIIFragment(value, 32, true)
		args := []string{}
		if repeats > 0 {
			args = append(args, "-"+strings.Repeat("v", repeats), "arg-"+suffix)
		}
		before := append([]string{}, args...)
		env := cli.Env{{
			Name:  "APP_LABEL_" + suffix,
			Value: value,
		}}
		if suffix != "" {
			env = append(env, cli.EnvEntry{
				Name:  "APP_LABEL_",
				Value: "empty",
			})
		}
		inspection, _ := inputFuzzInspect(t, app, cli.InspectionRequest{
			Args: args,
			Env:  env,
		})
		if !slices.Equal(args, before) {
			t.Fatalf("Inspect() mutated caller args: got %#v want %#v", args, before)
		}
		for _, token := range inspection.Tokens {
			origin, ok := inspection.Origins[token.Origin]
			if !ok || token.Origin == (cli.InputOriginID{}) {
				t.Fatalf("token lost authoritative origin: %#v", token)
			}
			if origin.Span.Source == (cli.InputSourceID{}) {
				t.Fatalf("origin has zero source: %#v", origin)
			}
			if _, ok := inspection.Sources[origin.Span.Source]; !ok {
				t.Fatalf("origin source %v missing from public closure", origin.Span.Source)
			}
			if origin.Span.Start.Offset < 0 || origin.Span.End.Offset < origin.Span.Start.Offset {
				t.Fatalf("invalid half-open byte span: %#v", origin.Span)
			}
		}
		var textTerm cli.ArgumentTermInspection
		for _, term := range inspection.ArgumentTerms {
			if term.Name == "text" {
				textTerm = term
				break
			}
		}
		if textTerm.ID == 0 {
			t.Fatal("optional argument term missing from Inspection")
		}
		if repeats == 0 {
			if len(textTerm.InputOrigins) != 0 {
				t.Fatalf("absent optional term owns input origins: %#v", textTerm.InputOrigins)
			}
		} else {
			if len(textTerm.InputOrigins) != 1 {
				t.Fatalf("selected argument term origins = %#v, want one", textTerm.InputOrigins)
			}
			if _, ok := inspection.Origins[textTerm.InputOrigins[0]]; !ok {
				t.Fatalf("selected term origin %v missing from public origin closure", textTerm.InputOrigins[0])
			}
		}

		if repeats > 1 {
			flagTokens := 0
			for _, token := range inspection.Tokens {
				if token.Role == cli.TokenRoleFlag && token.Text == "-v" {
					flagTokens++
				}
			}
			if flagTokens != repeats {
				t.Fatalf(
					"cluster produced %d semantic flag tokens, want %d: %#v",
					flagTokens,
					repeats,
					inspection.Tokens,
				)
			}
		}
		entries := map[string]string{}
		for _, candidate := range inspection.ValueCandidates {
			if candidate.Contribution != cli.ValueCandidateContributionKeyedEntry ||
				candidate.Disposition != cli.ValueCandidateDispositionEffective ||
				len(candidate.Entries) != 1 {
				continue
			}
			entry := candidate.Entries[0]
			if entry.Key.Input == nil || entry.Value.Input == nil || entry.Key.Representation == nil ||
				entry.Value.Representation == nil {
				t.Fatalf("keyed input lost exact component observation: %#v", entry)
			}
			entries[entry.Key.Representation.Text] = entry.Value.Representation.Text
		}
		got, ok := entries[suffix]
		if !ok || got != value {
			t.Fatalf("environment prefix lost suffix contribution %q=%q: %#v", suffix, value, entries)
		}
		if suffix != "" {
			got, ok := entries[""]
			if !ok || got != "empty" {
				t.Fatalf("environment prefix lost independent empty-key contribution: %#v", entries)
			}
		}
	})
}

func inputFuzzOnlyInspectionCandidate(t *testing.T, inspection cli.Inspection) cli.ValueCandidateInspection {
	t.Helper()
	if len(inspection.ValueCandidates) != 1 {
		t.Fatalf("Inspection.ValueCandidates = %#v, want exactly one candidate", inspection.ValueCandidates)
	}
	for _, candidate := range inspection.ValueCandidates {
		return candidate
	}
	panic("unreachable")
}

func inputFuzzInspectionHasDiagnosticCode(inspection cli.Inspection, code cli.DiagnosticCode) bool {
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
