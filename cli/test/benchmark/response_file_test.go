package benchmark

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

// BenchmarkInspectResponseFiles covers token replay, nesting depth, and provenance growth through public inspection.
func BenchmarkInspectResponseFiles(b *testing.B) {
	app := responseApplication(b)
	for _, tokens := range [...]int{1, 8, 32, 128, 512, 1024} {
		files, want := flatResponseFiles(tokens)
		request := cli.InspectionRequest{
			Args:          []string{"@flat"},
			ResponseFiles: responseAccess(files),
		}
		preflightResponseInspection(b, app, request, want, 1)
		b.Run(fmt.Sprintf("Tokens/N=%d", tokens), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
	for _, depth := range [...]int{1, 2, 4, 8, 16} {
		files := nestedResponseFiles(depth)
		request := cli.InspectionRequest{
			Args:          []string{"@d000"},
			ResponseFiles: responseAccess(files),
		}
		preflightResponseInspection(b, app, request, []string{"value"}, depth)
		b.Run(fmt.Sprintf("Depth/N=%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
	for _, entries := range [...]int{16, 64, 256, 1024} {
		files, want := provenanceResponseFiles(entries)
		request := cli.InspectionRequest{
			Args:          []string{"@root"},
			ResponseFiles: responseAccess(files),
		}
		preflightResponseInspection(b, app, request, want, 2)
		b.Run(fmt.Sprintf("ProvenanceEntries/N=%d/Depth=2", entries), func(b *testing.B) {
			b.ReportAllocs()
			clibench.Inspect(b, app, request)
		})
	}
}

func responseApplication(tb testing.TB) cli.Application {
	tb.Helper()
	definition := cli.Definition[inputConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[inputConfig]) {
			root.Dialect.ResponseFiles(true)
			values := root.Values.Sequence(&root.Config.Strings, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[inputConfig]) {
				form.Sequence(values, "value", cli.Count.ZeroOrMore()).Greedy()
			})
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("response benchmark compile: %v", err)
	}
	return app
}

func flatResponseFiles(tokens int) (map[string]cli.ResponseFile, []string) {
	values := make([]string, tokens)
	for index := range values {
		values[index] = fmt.Sprintf("v%05d", index)
	}
	return map[string]cli.ResponseFile{
		"flat": responseFile("flat", strings.Join(values, " ")),
	}, values
}

func nestedResponseFiles(depth int) map[string]cli.ResponseFile {
	files := make(map[string]cli.ResponseFile, depth)
	for index := range depth {
		name := fmt.Sprintf("d%03d", index)
		content := "value"
		if index+1 < depth {
			content = "@" + fmt.Sprintf("d%03d", index+1)
		}
		files[name] = responseFile(name, content)
	}
	return files
}

func provenanceResponseFiles(entries int) (map[string]cli.ResponseFile, []string) {
	children := entries / 2
	files := make(map[string]cli.ResponseFile, children+1)
	includes := make([]string, children)
	values := make([]string, children)
	for index := range children {
		name := fmt.Sprintf("child-%04d", index)
		value := fmt.Sprintf("value-%04d", index)
		includes[index] = "@" + name
		values[index] = value
		files[name] = responseFile(name, value)
	}
	files["root"] = responseFile("root", strings.Join(includes, " "))
	return files, values
}

func responseFile(name, content string) cli.ResponseFile {
	file := cli.ResponseFile{
		Key:      name,
		Revision: cli.Revision("r1"),
		Location: "memory:" + name,
		Content:  []byte(content),
	}
	return file
}

func responseAccess(files map[string]cli.ResponseFile) cli.ResponseFileAccess {
	access := cli.ResponseFileAccess{
		Load: func(_ context.Context, request cli.ResponseFileRequest) (cli.ResponseFile, error) {
			file, ok := files[request.Path]
			if !ok {
				return cli.ResponseFile{}, errors.New("response file not found")
			}
			return file, nil
		},
	}
	return access
}

func preflightResponseInspection(tb testing.TB, app cli.Application, request cli.InspectionRequest, want []string, minimumDepth int) {
	tb.Helper()
	proof := clitest.Inspect(tb.Context(), app, request)
	proof.Require(tb, clitest.InspectionReady())
	if tb.Failed() {
		return
	}
	inspection, _ := proof.Inspection()
	responseSources := make(map[cli.InputSourceID]struct{})
	for id, source := range inspection.Sources {
		if source.Kind == cli.InputKindResponseFile {
			responseSources[id] = struct{}{}
		}
	}
	if len(responseSources) == 0 {
		tb.Fatal("response inspection produced no response-file sources")
	}
	expanded := make([]string, 0, len(want))
	maxDepth := 0
	for _, token := range inspection.Tokens {
		if token.Role == cli.TokenRoleResponseFile {
			continue
		}
		origin, ok := inspection.Origins[token.Origin]
		if !ok {
			tb.Fatalf("response token origin %v missing", token.Origin)
		}
		if _, ok := responseSources[origin.Span.Source]; !ok {
			continue
		}
		expanded = append(expanded, token.Text)
		if depth := originDepth(tb, inspection, token.Origin); depth > maxDepth {
			maxDepth = depth
		}
	}
	if len(expanded) != len(want) {
		tb.Fatalf("expanded response tokens=%d want=%d", len(expanded), len(want))
	}
	for index := range want {
		if expanded[index] != want[index] {
			tb.Fatalf("expanded[%d]=%q want=%q", index, expanded[index], want[index])
		}
	}
	if maxDepth < minimumDepth {
		tb.Fatalf("response provenance depth=%d want at least %d", maxDepth, minimumDepth)
	}
}

func originDepth(tb testing.TB, inspection cli.Inspection, start cli.InputOriginID) int {
	tb.Helper()
	seen := make(map[cli.InputOriginID]struct{})
	current := start
	depth := 0
	for {
		if _, duplicate := seen[current]; duplicate {
			tb.Fatalf("response provenance cycle at %v", current)
		}
		seen[current] = struct{}{}
		origin, ok := inspection.Origins[current]
		if !ok {
			tb.Fatalf("response provenance origin %v missing", current)
		}
		if len(origin.Parents) == 0 {
			return depth
		}
		current = origin.Parents[0]
		depth++
	}
}
