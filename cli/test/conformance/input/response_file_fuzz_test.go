package conformance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func responseFileFuzzInspect(t *testing.T, app cli.Application, request cli.InspectionRequest) (cli.Inspection, error) {
	t.Helper()
	return clitest.Inspect(t.Context(), app, request).Inspection()
}

var _ = errors.Is

func responseFileFuzzBoundedFuzzText(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}

func responseFileFuzzFuzzASCIIFragment(value string, maximum int, preserveEmpty bool) string {
	value = responseFileFuzzBoundedFuzzText(value, maximum)
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

type responseFileFuzzValueConfig struct{ Value string }

func FuzzResponsePipelineProperties(f *testing.F) {
	type config = responseFileFuzzValueConfig
	definition := cli.Definition[config]{
		Name: "tool",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(true)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(&form.Config.Value, "value")
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		f.Fatalf("Compile(response) error = %v", err)
	}
	definitionUnderTest := cli.Definition[config]{
		Name: "literal-tool",
		Run:  func(context.Context, *config) error { return nil },
		Declare: func(root *cli.CommandBuilder[config]) {
			root.Dialect.ResponseFiles(false)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) {
				form.String(&form.Config.Value, "value")
			})
		},
	}
	literalApp, err := cli.Compile(definitionUnderTest)
	if err != nil {
		f.Fatalf("Compile(literal) error = %v", err)
	}
	f.Add("value", false, false)
	f.Add("nested", true, false)
	f.Add("budget", true, true)
	f.Fuzz(func(t *testing.T, text string, nested, breach bool) {
		text = responseFuzzToken(text)
		loads := 0
		limits := cli.AnalysisLimits{}
		if breach && nested {
			limits.ResponseFiles.TotalSourceBytes = len("@inner")
		}
		request := cli.InspectionRequest{
			Args:   []string{"@outer"},
			Limits: limits,
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
					loads++
					switch request.Path {
					case "outer":
						if nested {
							file := cli.ResponseFile{
								Key:         "outer",
								Revision:    cli.Revision("1"),
								IncludeBase: "root",
								Content:     []byte("@inner"),
							}
							return file, nil
						}
						file := cli.ResponseFile{
							Key:      "outer",
							Revision: cli.Revision("1"),
							Content:  []byte(text),
						}
						return file, nil
					case "inner":
						file := cli.ResponseFile{
							Key:      "inner",
							Revision: cli.Revision("1"),
							Content:  []byte(text),
						}
						return file, nil
					default:
						return cli.ResponseFile{}, errors.New("unexpected response path")
					}
				},
			},
		}
		inspection, inspectErr := responseFileFuzzInspect(t, app, request)
		if breach && nested {
			if inspectErr == nil || inspection.Status == cli.InspectionStatusReady {
				t.Fatalf("cumulative response budget breach committed: %#v", inspection)
			}
			if loads != 2 {
				t.Fatalf("budget breach loads = %d, want outer+inner", loads)
			}
			report := cli.Diagnostics(inspectErr)
			if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != cli.DiagnosticResponseFileLimitExceeded {
				t.Fatalf("budget breach Diagnostics = %#v", report)
			}
			return
		}
		if inspectErr != nil || inspection.Status != cli.InspectionStatusReady {
			t.Fatalf(
				"response pipeline rejected canonical bounded token %q: status=%v err=%v diagnostics=%#v",
				text,
				inspection.Status,
				inspectErr,
				inspection.Diagnostics,
			)
		}
		wantLoads := 1
		if nested {
			wantLoads = 2
		}
		if loads != wantLoads {
			t.Fatalf("response loads = %d, want %d", loads, wantLoads)
		}
		responseSources := 0
		for _, source := range inspection.Sources {
			if source.Kind == cli.InputKindResponseFile {
				responseSources++
			}
		}
		if responseSources != wantLoads {
			t.Fatalf("response source closure = %d, want %d: %#v", responseSources, wantLoads, inspection.Sources)
		}

		literalLoads := 0
		literal, literalErr := responseFileFuzzInspect(t, literalApp, cli.InspectionRequest{
			Args: []string{"@outer"},
			ResponseFiles: cli.ResponseFileAccess{
				Load: func(context.Context, cli.ResponseFileRequest) (cli.ResponseFile, error) {
					literalLoads++
					return cli.ResponseFile{}, errors.New("must not load")
				},
			},
		})
		if literalErr != nil || literal.Status != cli.InspectionStatusReady || literalLoads != 0 {
			t.Fatalf(
				"command-local response dialect ignored: loads=%d result=%#v err=%v",
				literalLoads,
				literal,
				literalErr,
			)
		}
	})
}

func responseFuzzToken(value string) string {
	value = responseFileFuzzFuzzASCIIFragment(value, 24, false)
	if value == "" {
		return "value"
	}
	return "v" + value
}
