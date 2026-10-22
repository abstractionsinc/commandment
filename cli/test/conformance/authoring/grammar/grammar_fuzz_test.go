package conformance

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

// Source lineage: authoring_acceptance_test.go

var _ = errors.Is

func FuzzPositionalFormSelectionClassificationIsOrderIndependent(f *testing.F) {
	f.Add(uint8(0), uint8(1))
	f.Add(uint8(0), uint8(0))
	f.Add(uint8(1), uint8(1))
	f.Add(uint8(0), uint8(2))
	f.Fuzz(func(t *testing.T, aRaw, bRaw uint8) {
		a := grammarFuzzPositionalFormOutcome(aRaw % 3)
		b := grammarFuzzPositionalFormOutcome(bRaw % 3)
		var priorOutcome cli.ArgumentFormSelectionOutcome
		var priorWinner uint8
		for order, reverse := range []bool{false, true} {
			fixture := grammarFuzzNewFormSelectionFixture(t, a, b, reverse)
			inspection, err := fixture.app.Inspect(t.Context(), cli.InspectionRequest{
				Args: []string{"value"},
			})
			selection := inspection.Commands[inspection.Selection.Command.Selected].ArgumentFormSelection
			if fixture.callsA.Load() != 1 || fixture.callsB.Load() != 1 {
				t.Fatalf("order %d conversion calls = A:%d B:%d", order, fixture.callsA.Load(), fixture.callsB.Load())
			}
			formA := argumentFormForDestination(t, inspection, fixture.valueAID)
			formB := argumentFormForDestination(t, inspection, fixture.valueBID)
			failed := a == grammarFuzzPositionalFormFail || b == grammarFuzzPositionalFormFail
			accepted := 0
			if a == grammarFuzzPositionalFormAccept {
				accepted++
			}
			if b == grammarFuzzPositionalFormAccept {
				accepted++
			}
			switch {
			case failed:
				if selection.Outcome != cli.ArgumentFormSelectionFailed || err == nil {
					t.Fatalf("order %d failed classification = %#v err %v", order, selection, err)
				}
				if inspectionResolvedText(inspection, fixture.valueAID) {
					t.Fatalf("order %d failed form A leaked a resolved value", order)
				}
				if inspectionResolvedText(inspection, fixture.valueBID) {
					t.Fatalf("order %d failed form B leaked a resolved value", order)
				}
			case accepted == 1:
				wantForm := formA
				wantDestination := fixture.valueAID
				loseDestination := fixture.valueBID
				winner := uint8(1)
				if b == grammarFuzzPositionalFormAccept {
					wantForm = formB
					wantDestination, loseDestination = fixture.valueBID, fixture.valueAID
					winner = 2
				}
				if selection.Outcome != cli.ArgumentFormSelectionSelected || err != nil ||
					selection.Selected != wantForm {
					t.Fatalf(
						"order %d unique classification = %#v err %v want form %v",
						order,
						selection,
						err,
						wantForm,
					)
				}
				if !inspectionResolvedText(inspection, wantDestination) {
					t.Fatalf("order %d selected destination %v was not published", order, wantDestination)
				}
				if inspectionResolvedText(inspection, loseDestination) {
					t.Fatalf("order %d losing destination %v leaked a resolved value", order, loseDestination)
				}
				if order == 0 {
					priorWinner = winner
				} else if winner != priorWinner {
					t.Fatalf(
						"declaration order changed selected semantic form: forward=%d reverse=%d",
						priorWinner,
						winner,
					)
				}
			case accepted > 1:
				if selection.Outcome != cli.ArgumentFormSelectionAmbiguous || err == nil {
					t.Fatalf("order %d ambiguity classification = %#v err %v", order, selection, err)
				}
			default:
				if selection.Outcome != cli.ArgumentFormSelectionRejected || err == nil {
					t.Fatalf("order %d rejection classification = %#v err %v", order, selection, err)
				}
			}
			if order == 0 {
				priorOutcome = selection.Outcome
			} else if selection.Outcome != priorOutcome {
				t.Fatalf(
					"declaration order changed semantic classification: forward=%v reverse=%v",
					priorOutcome,
					selection.Outcome,
				)
			}
		}
	})
}

type grammarFuzzTokenLanguageRuntimeABConfig struct {
	A string
	B string
}

type grammarFuzzPositionalFormOutcome uint8

const (
	grammarFuzzPositionalFormAccept grammarFuzzPositionalFormOutcome = iota
	grammarFuzzPositionalFormReject
	grammarFuzzPositionalFormFail
)

func grammarFuzzFormSelectionCodec(name string, outcome grammarFuzzPositionalFormOutcome, failure error, calls *atomic.Int32) cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: name,
		Parse: func(text string) (string, error) {
			calls.Add(1)
			switch outcome {
			case grammarFuzzPositionalFormAccept:
				return name + "=" + text, nil
			case grammarFuzzPositionalFormReject:
				return "", cli.Rejection.Message(
					"form-reject",
					"form rejected input",
					cli.Rejection.At(0, 0, len(text)),
				)
			case grammarFuzzPositionalFormFail:
				return "", failure
			default:
				panic("unknown form outcome")
			}
		},
		Format:   func(value string) string { return value },
		Snapshot: cli.Value.Assign[string](),
	}
	return codec
}

type grammarFuzzFormSelectionFixture struct {
	app      cli.Application
	valueAID cli.ValueID
	valueBID cli.ValueID
	callsA   *atomic.Int32
	callsB   *atomic.Int32
	failure  error
}

func grammarFuzzNewFormSelectionFixture(t *testing.T, a, b grammarFuzzPositionalFormOutcome, reverse bool) grammarFuzzFormSelectionFixture {
	t.Helper()
	type config = grammarFuzzTokenLanguageRuntimeABConfig
	callsA := &atomic.Int32{}
	callsB := &atomic.Int32{}
	failure := errors.New("form-b operational failure")
	var valueA cli.Destination[config, string]
	var valueB cli.Destination[config, string]
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Declare: func(root *cli.CommandBuilder[config]) {
			valueA = root.Values.Value(&root.Config.A, grammarFuzzFormSelectionCodec("A", a, failure, callsA))
			valueB = root.Values.Value(&root.Config.B, grammarFuzzFormSelectionCodec("B", b, failure, callsB))
			declareA := func() {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(valueA, "a") })
			}
			declareB := func() {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(valueB, "b") })
			}
			if reverse {
				declareB()
				declareA()
			} else {
				declareA()
				declareB()
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	schema := app.Schema()
	var valueAID, valueBID cli.ValueID
	for _, term := range schema.ArgumentTerms {
		switch term.Name {
		case "a":
			valueAID = term.Destination
		case "b":
			valueBID = term.Destination
		}
	}
	if valueAID == 0 || valueBID == 0 || valueAID == valueBID {
		t.Fatalf("argument destination identities a=%v b=%v terms=%#v", valueAID, valueBID, schema.ArgumentTerms)
	}
	if _, ok := schema.Values[valueAID]; !ok {
		t.Fatalf("argument a destination %v missing from schema values", valueAID)
	}
	if _, ok := schema.Values[valueBID]; !ok {
		t.Fatalf("argument b destination %v missing from schema values", valueBID)
	}
	fixture := grammarFuzzFormSelectionFixture{
		app:      app,
		valueAID: valueAID,
		valueBID: valueBID,
		callsA:   callsA,
		callsB:   callsB,
		failure:  failure,
	}
	return fixture
}

func inspectionResolvedText(inspection cli.Inspection, id cli.ValueID) bool {
	value, ok := inspection.Values[id]
	return ok && value.State == cli.ResolutionStateResolved && value.Representation != nil
}

func argumentFormForDestination(t *testing.T, inspection cli.Inspection, destination cli.ValueID) cli.ArgumentFormID {
	t.Helper()
	var matched cli.ArgumentFormID
	for formID, form := range inspection.ArgumentForms {
		for _, termID := range form.Terms {
			term, ok := inspection.ArgumentTerms[termID]
			if !ok || term.Destination != destination {
				continue
			}
			if matched != cli.ArgumentFormID(0) && matched != formID {
				t.Fatalf("destination %v appears in multiple argument forms: %v and %v", destination, matched, formID)
			}
			matched = formID
		}
	}
	if matched == cli.ArgumentFormID(0) {
		t.Fatalf("destination %v has no argument-form term in inspection", destination)
	}
	return matched
}

func boundedFuzzToken(value string) string {
	if len(value) > 24 {
		value = value[:24]
	}
	return value
}

func positionalFuzzToken(value string) string {
	value = boundedFuzzToken(value)
	if strings.HasPrefix(value, "-") {
		return "value" + value
	}
	return value
}

func fuzzNameSuffix(value string) string {
	var b strings.Builder
	for _, r := range boundedFuzzToken(value) {
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
		if b.Len() >= 8 {
			break
		}
	}
	if b.Len() == 0 {
		return "x"
	}
	return b.String()
}

func inspectSnapshot(t *testing.T, app cli.Application, args ...string) (cli.Inspection, error) {
	t.Helper()
	request := cli.InspectionRequest{
		Args: args,
	}
	return clitest.Inspect(t.Context(), app, request).Inspection()
}

func argumentTermByName(t *testing.T, inspection cli.Inspection, name string) cli.ArgumentTermInspection {
	t.Helper()
	for _, term := range inspection.ArgumentTerms {
		if term.Name == name {
			return term
		}
	}
	t.Fatalf("argument term %q not found", name)
	return cli.ArgumentTermInspection{}
}

func FuzzPositionalAllocationAndFormProperties(f *testing.F) {
	type allocationConfig struct{ First, Second []string }
	definition := cli.Definition[allocationConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[allocationConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderBeforeArguments)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[allocationConfig]) {
				form.Strings(&form.Config.First, "first", cli.Count.Optional()).Greedy()
				form.Strings(&form.Config.Second, "second", cli.Count.Optional())
			})
		},
	}
	greedy := clitest.Compile(f, definition)
	definitionUnderTest := cli.Definition[allocationConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[allocationConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderBeforeArguments)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[allocationConfig]) {
				form.Strings(&form.Config.First, "first", cli.Count.Optional()).Minimal()
				form.Strings(&form.Config.Second, "second", cli.Count.Optional())
			})
		},
	}
	minimal := clitest.Compile(f, definitionUnderTest)
	definitionUnderTest2 := cli.Definition[allocationConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[allocationConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderBeforeArguments)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[allocationConfig]) {
				form.Strings(&form.Config.First, "first", cli.Count.ZeroOrMore())
				form.Strings(&form.Config.Second, "second", cli.Count.ZeroOrMore())
			})
		},
	}
	ambiguous := clitest.Compile(f, definitionUnderTest2)
	type boundedConfig struct{ Values []string }
	definitionUnderTest3 := cli.Definition[boundedConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[boundedConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderBeforeArguments)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[boundedConfig]) {
				form.Strings(&form.Config.Values, "value", cli.Count.Between(0, 2))
			})
		},
	}
	bounded := clitest.Compile(f, definitionUnderTest3)
	f.Add("x")
	f.Add("")
	f.Add("tail")
	f.Fuzz(func(t *testing.T, raw string) {
		payload := positionalFuzzToken(raw)
		gi, err := inspectSnapshot(t, greedy, payload)
		if err != nil {
			t.Fatalf("greedy inspect: %v", err)
		}
		mi, err := inspectSnapshot(t, minimal, payload)
		if err != nil {
			t.Fatalf("minimal inspect: %v", err)
		}
		if got := len(argumentTermByName(t, gi, "first").InputOrigins); got != 1 {
			t.Fatalf("greedy first origins=%d, want 1", got)
		}
		if got := len(argumentTermByName(t, mi, "first").InputOrigins); got != 0 {
			t.Fatalf("minimal first origins=%d, want 0", got)
		}
		ai, aerr := inspectSnapshot(t, ambiguous, payload)
		if aerr == nil {
			t.Fatalf("ambiguous allocation accepted: %#v", ai)
		}
		clitest.Diagnostics(aerr).Require(t, clitest.Diagnostic(cli.DiagnosticAmbiguousArgumentForm).Present())
		if len(ai.Tokens) != 1 || ai.Tokens[0].Role != cli.TokenRoleUnknown {
			t.Fatalf("ambiguous token ownership=%#v", ai.Tokens)
		}
		bi, err := inspectSnapshot(t, bounded, payload, payload)
		if err != nil {
			t.Fatalf("bounded two rejected: status=%v err=%v", bi.Status, err)
		}
		oi, oerr := inspectSnapshot(t, bounded, payload, payload, payload)
		if oerr == nil {
			t.Fatalf("bounded overflow accepted: %#v", oi)
		}
		clitest.Diagnostics(oerr).Require(t, clitest.Diagnostic(cli.DiagnosticUnexpectedArgument).Present())

		firstLiteral := "alpha" + fuzzNameSuffix(payload)
		secondLiteral := "beta" + fuzzNameSuffix(payload)
		definition := cli.Definition[struct{}]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			ShellCompletion: cli.ShellCompletion{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[struct{}]) { form.Literal(firstLiteral) })
				root.Arguments.Form(func(form *cli.ArgumentFormBuilder[struct{}]) { form.Literal(secondLiteral) })
			},
		}
		formApp := clitest.Compile(t, definition)
		clitest.Inspect(t.Context(), formApp, cli.InspectionRequest{
			Args: []string{firstLiteral},
		}).Require(t, clitest.RootCommand().ArgumentFormAt(0).Selected())
		clitest.Inspect(t.Context(), formApp, cli.InspectionRequest{
			Args: []string{secondLiteral},
		}).Require(t, clitest.RootCommand().ArgumentFormAt(1).Selected())
	})
}

func grammarDataFuzzToken(value string) string {
	value = boundedFuzzToken(value)
	var b strings.Builder
	b.WriteByte('v')
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func FuzzGeneralGrammarLexicalProperties(f *testing.F) {
	type flagConfig struct {
		Mode    string
		Verbose bool
	}
	definition := cli.Definition[flagConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[flagConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Flags.String(&root.Config.Mode).Long("mode").ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormAssigned)
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
		},
	}
	flagApp := clitest.Compile(f, definition)
	type remainderConfig struct {
		Rest []string
		Pos  []string
	}
	definitionUnderTest := cli.Definition[remainderConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[remainderConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderBeforeArguments)
			root.Subcommands.Command(cli.Command[remainderConfig]{
				Name: "child",
			})
			rest := root.Values.Sequence(&root.Config.Rest, cli.Value.String())
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[remainderConfig]) {
				form.Remainder(rest, "rest", cli.Count.ZeroOrMore())
			})
		},
	}
	remainderApp := clitest.Compile(f, definitionUnderTest)
	definitionUnderTest2 := cli.Definition[remainderConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[remainderConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderBeforeArguments)
			root.Subcommands.Command(cli.Command[remainderConfig]{
				Name: "child",
			})
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[remainderConfig]) {
				form.Strings(&form.Config.Pos, "arg", cli.Count.Between(1, 2))
			})
		},
	}
	positionalApp := clitest.Compile(f, definitionUnderTest2)
	type terminatorConfig struct {
		Items   []string
		Verbose bool
	}
	definitionUnderTest3 := cli.Definition[terminatorConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[terminatorConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Flags.Strings(&root.Config.Items).Occurrences(cli.Count.Between(0, 2)).ValuesPerOccurrence(cli.Count.Between(2, 3)).TerminatedBy(";").Long("item").ValueForms(cli.FlagValueFormSeparate)
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
		},
	}
	terminatorApp := clitest.Compile(f, definitionUnderTest3)
	type clusterConfig struct {
		A       bool
		Include string
	}
	definitionUnderTest4 := cli.Definition[clusterConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[clusterConfig]) {
			root.Dialect.ShortFlagClustering(cli.ShortFlagClusteringEnabled)
			root.Flags.Bool(&root.Config.A).Short('a')
			root.Flags.String(&root.Config.Include).Short('I').ValueForms(cli.FlagValueFormJoined)
		},
	}
	clusterApp := clitest.Compile(f, definitionUnderTest4)
	type numberConfig struct{ Number int }
	definitionUnderTest5 := cli.Definition[numberConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[numberConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Flags.Int(&root.Config.Number).HyphenValues(cli.HyphenValueNegativeNumbers).Long("number").ValueForms(cli.FlagValueFormSeparate)
		},
	}
	numberApp := clitest.Compile(f, definitionUnderTest5)
	type actionConfig struct{ Arg string }
	definitionUnderTest6 := cli.Definition[actionConfig]{
		Name: "tool",
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[actionConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderBeforeArguments)
			root.Arguments.Form(func(form *cli.ArgumentFormBuilder[actionConfig]) { form.String(&form.Config.Arg, "arg") })
		},
	}
	actionApp := clitest.Compile(f, definitionUnderTest6)
	type recoveryConfig struct{ Verbose bool }
	definitionUnderTest7 := cli.Definition[recoveryConfig]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[recoveryConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			root.Flags.Bool(&root.Config.Verbose).Long("verbose")
		},
	}
	recoveryApp := clitest.Compile(f, definitionUnderTest7)

	f.Add("value")
	f.Add("")
	f.Add("MiXeD")
	f.Add("a,b,,c")
	f.Fuzz(func(t *testing.T, raw string) {
		payload := grammarDataFuzzToken(raw)
		base := "serve" + fuzzNameSuffix(payload)
		definition := cli.Definition[struct{}]{
			Name: "tool",
			Help: cli.HelpConfig{
				Disabled: true,
			},
			ShellCompletion: cli.ShellCompletion{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[struct{}]) {
				root.Dialect.CommandMatching(cli.NameMatchingUniquePrefix)
				root.Dialect.CommandNormalization(cli.NameNormalizationASCIICaseFold)
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: base,
				})
				root.Subcommands.Command(cli.Command[struct{}]{
					Name: base + "r",
				})
			},
		}
		nameApp := clitest.Compile(t, definition)
		exact := clitest.Command(base)
		clitest.Inspect(t.Context(), nameApp, cli.InspectionRequest{
			Args: []string{strings.ToUpper(base)},
		}).Require(t, exact.Selected(), clitest.TokenAt(0).Role(cli.TokenRoleCommand))
		prefix := base[:len(base)-1]
		ambiguous := clitest.Inspect(t.Context(), nameApp, cli.InspectionRequest{
			Args: []string{prefix},
		})
		ambiguous.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticAmbiguousCommand).Present()))
		ai, _ := ambiguous.Inspection()
		if len(ai.Tokens) != 1 || ai.Tokens[0].Role != cli.TokenRoleUnknown {
			t.Fatalf("ambiguous name token=%#v", ai.Tokens)
		}

		separate := clitest.Inspect(t.Context(), flagApp, cli.InspectionRequest{
			Args: []string{"--mode", payload},
		})
		separate.Require(t, clitest.TokenAt(0).Role(cli.TokenRoleFlag), clitest.TokenAt(1).Role(cli.TokenRoleFlagValue))
		assigned := clitest.Inspect(t.Context(), flagApp, cli.InspectionRequest{
			Args: []string{"--mode=" + payload},
		})
		assigned.Require(t, clitest.TokenAt(0).Role(cli.TokenRoleFlag))
		missing := clitest.Inspect(t.Context(), flagApp, cli.InspectionRequest{
			Args: []string{"--mode", "--verbose"},
		})
		missing.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticMissingValue).Present()), clitest.TokenAt(1).Role(cli.TokenRoleFlag))
		owned := clitest.Inspect(t.Context(), remainderApp, cli.InspectionRequest{
			Args: []string{payload, "child", "--help"},
		})
		owned.Require(t, clitest.SelectionPathExactly(clitest.RootCommand()), clitest.TokenAt(0).Role(cli.TokenRoleRemainder), clitest.TokenAt(1).Role(cli.TokenRoleRemainder), clitest.TokenAt(2).Role(cli.TokenRoleRemainder))
		afterEnd := clitest.Inspect(t.Context(), positionalApp, cli.InspectionRequest{
			Args: []string{"--", payload},
		})
		afterEnd.Require(t, clitest.TokenAt(0).Role(cli.TokenRoleEndOfOptions), clitest.TokenAt(1).Role(cli.TokenRolePositional))
		if payload != "child" {
			handoff := clitest.Inspect(t.Context(), positionalApp, cli.InspectionRequest{
				Args: []string{payload, "child"},
			})
			handoff.Require(t, clitest.SelectionPathExactly(clitest.RootCommand()), clitest.TokenAt(0).Role(cli.TokenRolePositional), clitest.TokenAt(1).Role(cli.TokenRolePositional))
		}

		premature := clitest.Inspect(t.Context(), terminatorApp, cli.InspectionRequest{
			Args: []string{"--item", payload, ";", "--verbose"},
		})
		premature.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticMissingValue).Present()), clitest.TokenAt(3).Role(cli.TokenRoleFlag))
		closed := clitest.Inspect(t.Context(), terminatorApp, cli.InspectionRequest{
			Args: []string{"--item", payload, payload, ";"},
		})
		if _, err := closed.Inspection(); err != nil {
			t.Fatalf("closed variable occurrence rejected: %v", err)
		}

		clusterValue := payload
		if clusterValue == "" {
			clusterValue = "x"
		}
		cluster := clitest.Inspect(t.Context(), clusterApp, cli.InspectionRequest{
			Args: []string{"-aI" + clusterValue},
		})
		ci, cerr := cluster.Inspection()
		if cerr != nil {
			t.Fatalf("cluster error: %v", cerr)
		}
		if len(ci.Tokens) != 2 || ci.Tokens[0].Text != "-a" || ci.Tokens[1].Text != "-I" {
			t.Fatalf("cluster tokens=%#v", ci.Tokens)
		}

		number := "-" + string(rune('1'+len(payload)%8))
		if _, err := inspectSnapshot(t, numberApp, "--number", number); err != nil {
			t.Fatalf("negative number rejected: %v", err)
		}
		bad := "-" + fuzzNameSuffix(payload) + "x"
		badResult := clitest.Inspect(t.Context(), numberApp, cli.InspectionRequest{
			Args: []string{"--number", bad},
		})
		badResult.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticMissingValue).Present()))

		action := clitest.Inspect(t.Context(), actionApp, cli.InspectionRequest{
			Args: []string{"--help"},
		})
		action.Require(t, clitest.InvocationAction(cli.InvocationActionHelp).Selected(), clitest.TokenAt(0).Role(cli.TokenRoleAction))
		actionAfterEnd := clitest.Inspect(t.Context(), actionApp, cli.InspectionRequest{
			Args: []string{"--", "--help"},
		})
		actionAfterEnd.Require(t, clitest.TokenAt(1).Role(cli.TokenRolePositional))

		rec := clitest.Inspect(t.Context(), recoveryApp, cli.InspectionRequest{
			Args: []string{"--unknown=" + payload, "--verbose"},
		})
		rec.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnknownFlag).Present()), clitest.TokenAt(1).Role(cli.TokenRoleFlag))
		bare := clitest.Inspect(t.Context(), recoveryApp, cli.InspectionRequest{
			Args: []string{"--unknown", payload, "--verbose"},
		})
		bare.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticUnknownFlag).Present()))
		bi, _ := bare.Inspection()
		if len(bi.Tokens) != 3 || bi.Tokens[2].Role != cli.TokenRoleUnknown {
			t.Fatalf("bare recovery tokens=%#v", bi.Tokens)
		}
		limited := clitest.Inspect(t.Context(), recoveryApp, cli.InspectionRequest{
			Args: []string{"--unknown=" + payload, "--another=" + payload},
			Limits: cli.AnalysisLimits{
				Diagnostics: 1,
			},
		})
		limited.Require(t, clitest.FailsWith(clitest.DiagnosticAt(0).Code(cli.DiagnosticUnknownFlag), clitest.DiagnosticsTruncated()))
	})
}
