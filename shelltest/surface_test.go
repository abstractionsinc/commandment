package shelltest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestExportedSurfaceIsExact(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	set := token.NewFileSet()
	exports := map[string]struct{}{}
	methods := map[string][]string{}
	fields := map[string][]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, filepath.Clean(entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			switch value := declaration.(type) {
			case *ast.GenDecl:
				for _, spec := range value.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !typeSpec.Name.IsExported() {
						continue
					}
					exports[typeSpec.Name.Name] = struct{}{}
					if structure, ok := typeSpec.Type.(*ast.StructType); ok {
						for _, field := range structure.Fields.List {
							for _, name := range field.Names {
								if name.IsExported() {
									fields[typeSpec.Name.Name] = append(fields[typeSpec.Name.Name], name.Name)
								}
							}
						}
					}
				}
			case *ast.FuncDecl:
				if !value.Name.IsExported() {
					continue
				}
				if value.Recv == nil {
					exports[value.Name.Name] = struct{}{}
					continue
				}
				receiver := receiverName(value.Recv.List[0].Type)
				if ast.IsExported(receiver) {
					methods[receiver] = append(methods[receiver], value.Name.Name)
				}
			}
		}
	}
	gotExports := make([]string, 0, len(exports))
	for name := range exports {
		gotExports = append(gotExports, name)
	}
	slices.Sort(gotExports)
	wantExports := []string{
		"Complete", "CompleteForGenerationTarget", "Input", "InterpretFunc", "Observation",
		"PrepareFunc", "Runtime", "RuntimeVersionParser", "Setup", "Target",
	}
	if !slices.Equal(gotExports, wantExports) {
		t.Fatalf("exported declarations = %#v, want %#v", gotExports, wantExports)
	}
	wantFields := map[string][]string{
		"Runtime": {"Executable", "ParseVersion", "VersionArgs"},
		"Target":  {"Interpret", "Prepare", "Provider", "Runtime"},
		"Input":   {"AfterCursor", "BeforeCursor"},
	}
	for name := range fields {
		slices.Sort(fields[name])
	}
	for name := range wantFields {
		slices.Sort(wantFields[name])
	}
	if !equalStringSliceMaps(fields, wantFields) {
		t.Fatalf("exported fields = %#v, want %#v", fields, wantFields)
	}
	for name := range methods {
		slices.Sort(methods[name])
	}
	wantMethods := map[string][]string{
		"Observation": {"EditorInput", "ScreenLines", "Version"},
		"Setup":       {"Args", "Env", "Executable", "File", "IntegrationFile", "Version"},
	}
	for name := range wantMethods {
		slices.Sort(wantMethods[name])
	}
	if !equalStringSliceMaps(methods, wantMethods) {
		t.Fatalf("exported methods = %#v, want %#v", methods, wantMethods)
	}
}

func equalStringSliceMaps(left, right map[string][]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValues := range left {
		rightValues, ok := right[key]
		if !ok || !slices.Equal(leftValues, rightValues) {
			return false
		}
	}
	return true
}

func receiverName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	case *ast.IndexExpr:
		return receiverName(value.X)
	case *ast.IndexListExpr:
		return receiverName(value.X)
	default:
		return ""
	}
}

func TestExportedSignaturesMatchAcceptedGrammar(t *testing.T) {
	t.Parallel()
	var _ RuntimeVersionParser = func(string) (string, error) { return "", nil }
	var _ PrepareFunc = func(Setup) error { return nil }
	var _ InterpretFunc[int] = func(Observation) (int, error) { return 0, nil }
	requireSignature[func(testing.TB, cli.Application, Target[int], Input) int](Complete[int])
	requireSignature[func(testing.TB, cli.Application, Target[int], string, Input) int](CompleteForGenerationTarget[int])
	requireSignature[func(Setup) string](Setup.Executable)
	requireSignature[func(Setup) string](Setup.Version)
	requireSignature[func(Setup, string) string](Setup.IntegrationFile)
	requireSignature[func(Setup, string, string) string](Setup.File)
	requireSignature[func(Setup, string, string)](Setup.Env)
	requireSignature[func(Setup, ...string)](Setup.Args)
	requireSignature[func(Observation) string](Observation.Version)
	requireSignature[func(Observation) (Input, bool)](Observation.EditorInput)
	requireSignature[func(Observation) []string](Observation.ScreenLines)

	_ = Runtime{
		Executable:   "runtime",
		VersionArgs:  []string{"--version"},
		ParseVersion: func(string) (string, error) { return "1", nil },
	}
	_ = Target[int]{
		Provider:  cli.Shell.Bash,
		Runtime:   Runtime{},
		Prepare:   nil,
		Interpret: nil,
	}
	_ = Input{
		BeforeCursor: "before",
		AfterCursor:  "after",
	}
}

func requireSignature[T any](_ T) {}
