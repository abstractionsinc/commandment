package conformance

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

var _ = errors.Is

type concurrencyModeRegionConfig struct {
	Mode   string
	Region string
}

type concurrencyModeConfig struct{ Mode string }

func TestMultiUseDynamicDomainSnapshotIsDetachedAcrossOperations(t *testing.T) {
	t.Parallel()
	type config = concurrencyModeRegionConfig
	calls := 0
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					calls++
					canonical := builder.Value("prod", "production")
					builder.Alias("p", canonical)
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			region := root.Values.Value(&root.Config.Region, domain)
			root.Flags.ValueFor(mode).Long("mode")
			root.Flags.ValueFor(region).Long("region")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}

	check := func(label string, inspection cli.Inspection, wantCalls int) {
		t.Helper()
		if calls != wantCalls || len(inspection.ValueDomains) != 1 {
			t.Fatalf("%s domain calls/state = %d %#v", label, calls, inspection.ValueDomains)
		}
		domain := inspection.ValueDomains[0]
		if domain.Outcome != cli.ValueDomainOutcomeReady || len(domain.Entries) != 2 || len(domain.Uses) != 2 {
			t.Fatalf("%s multi-use domain = %#v", label, domain)
		}
		destinations := map[string]cli.ValueID{}
		for _, flag := range inspection.Flags {
			for _, spellingID := range flag.Spellings {
				text := inspection.FlagSpellings[spellingID].Spelling.Text
				if text == "--mode" || text == "--region" {
					destinations[text] = flag.Destination
				}
			}
		}
		uses := map[cli.ValueID]struct{}{}
		for _, use := range domain.Uses {
			uses[use.Destination] = struct{}{}
		}
		for _, destination := range destinations {
			_, used := uses[destination]
			if destination == 0 || !used {
				t.Fatalf("%s domain uses = %#v destinations=%#v", label, domain.Uses, destinations)
			}
			value := inspection.Values[destination]
			if value.Representation == nil || value.Representation.Text != "prod" {
				t.Fatalf("%s resolved value %v = %#v", label, destination, value)
			}
		}
	}

	first, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--mode", "p", "--region", "prod"},
		})
		proof.Require(t, clitest.DiagnosticsExactly())
		return proof.Inspection()
	}()
	if err != nil {
		t.Fatal(err)
	}
	check("first", first, 1)
	first.ValueDomains[0].Uses[0].Destination = 0
	first.ValueDomains[0].Entries = nil
	for id, entry := range first.ValueDomainEntries {
		entry.Name = "mutated"
		first.ValueDomainEntries[id] = entry
		break
	}

	second, err := app.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"--mode", "prod", "--region", "p"},
	})
	if err != nil {
		t.Fatal(err)
	}
	check("second", second, 2)
}

func TestDynamicDomainFailedAndReadyEmptyArtifactsDetachAcrossOperations(t *testing.T) {
	t.Parallel()
	type config = concurrencyModeConfig
	failure := errors.New("domain unavailable")
	failedCalls := 0
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					failedCalls++
					return failure
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	failed, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	firstFailed, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), failed, cli.InspectionRequest{
			Args: []string{"--mode", "x"},
		})
		proof.Require(t, clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure).Present()))
		return proof.Inspection()
	}()
	if err == nil || !errors.Is(err, failure) || len(firstFailed.ValueDomains) != 1 ||
		firstFailed.ValueDomains[0].Outcome != cli.ValueDomainOutcomeFailed {
		t.Fatalf("first failed domain = %#v err=%v", firstFailed.ValueDomains, err)
	}
	firstFailed.ValueDomains[0].Outcome = cli.ValueDomainOutcomeReady
	secondFailed, err := failed.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"--mode", "x"},
	})
	if err == nil || !errors.Is(err, failure) || failedCalls != 2 || len(secondFailed.ValueDomains) != 1 ||
		secondFailed.ValueDomains[0].Outcome != cli.ValueDomainOutcomeFailed ||
		len(secondFailed.ValueDomains[0].Entries) != 0 {
		t.Fatalf("second failed domain = calls %d %#v err=%v", failedCalls, secondFailed.ValueDomains, err)
	}

	emptyCalls := 0
	definitionUnderTest := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					emptyCalls++
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	empty, err := cli.Compile(definitionUnderTest)
	if err != nil {
		t.Fatal(err)
	}
	firstEmpty, _ := empty.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"--mode", "x"},
	})
	if len(firstEmpty.ValueDomains) != 1 || firstEmpty.ValueDomains[0].Outcome != cli.ValueDomainOutcomeReady ||
		len(firstEmpty.ValueDomains[0].Entries) != 0 {
		t.Fatalf("first ready-empty domain = %#v", firstEmpty.ValueDomains)
	}
	firstEmpty.ValueDomains[0].Outcome = cli.ValueDomainOutcomeFailed
	secondEmpty, _ := empty.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"--mode", "x"},
	})
	if emptyCalls != 2 || len(secondEmpty.ValueDomains) != 1 ||
		secondEmpty.ValueDomains[0].Outcome != cli.ValueDomainOutcomeReady ||
		len(secondEmpty.ValueDomains[0].Entries) != 0 {
		t.Fatalf("second ready-empty domain = calls %d %#v", emptyCalls, secondEmpty.ValueDomains)
	}
}

const domainMaterializationToolName = "tool"

type domainMaterializationModeConfig struct{ Mode string }

func TestDynamicDomainMaterializationReady(t *testing.T) {
	t.Parallel()
	type config = domainMaterializationModeConfig
	var calls int
	definition := cli.Definition[config]{
		Name: domainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(ctx context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					calls++
					if ctx == nil {
						return errors.New("dynamic-domain context was not normalized")
					}
					canonical := builder.Value("prod", "production").Description("Production")
					builder.Alias("p", canonical)
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	value := clitest.RootCommand().Flag("--mode").Value()
	domain := value.Domain(cli.ValueDomainUseDestination)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "p"},
	})
	proof.Require(t,
		clitest.InspectionReady(), clitest.DiagnosticsExactly(),
		clitest.InspectionObservedValueDomainsExactly(domain),
		clitest.InspectionObservedValueDomainEntriesExactly(domain.Entry("prod"), domain.Entry("p")),
		domain.Outcome(cli.ValueDomainOutcomeReady), domain.InvocationScoped(),
		domain.EntriesExactly(domain.Entry("prod"), domain.Entry("p")),
		domain.Entry("p").Canonical(domain.Entry("prod")),
	)
	if calls != 1 {
		t.Fatalf("dynamic-domain calls = %d, want 1", calls)
	}
}

func TestDynamicDomainMaterializationFailure(t *testing.T) {
	t.Parallel()
	type config = domainMaterializationModeConfig
	sentinel := errors.New("domain unavailable")
	definition := cli.Definition[config]{
		Name: domainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					return sentinel
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "missing"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure).Present()),
		clitest.InspectionObservedValueDomainsExactly(domain),
		domain.Outcome(cli.ValueDomainOutcomeFailed),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Inspect() error = %v, want %v", err, sentinel)
	}
}

func TestDynamicDomainMaterializationRejectsInvalidDynamicDeclaration(t *testing.T) {
	t.Parallel()
	type config = domainMaterializationModeConfig
	definition := cli.Definition[config]{
		Name: domainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					builder.Value("", "invalid")
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "invalid"},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValueDomain).Present()),
		clitest.InspectionObservedValueDomainsExactly(domain),
		domain.Outcome(cli.ValueDomainOutcomeInvalid),
	)
}

func TestDynamicDomainDemandSkipsUnallocatedOptionalArgument(t *testing.T) {
	t.Parallel()
	type config = domainMaterializationModeConfig
	calls := 0
	definition := cli.Definition[config]{
		Name: domainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					calls++
					builder.Value("prod", "production")
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Arguments.Form(
				func(form *cli.ArgumentFormBuilder[config]) { form.ValueFor(mode, "mode").Optional() },
			)
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().ArgumentFormAt(0).Term("mode").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.InspectionReady(), clitest.DiagnosticsExactly(),
		clitest.InspectionObservedValueDomainsExactly(domain),
		domain.Outcome(cli.ValueDomainOutcomeNotNeeded),
	)
	if calls != 0 {
		t.Fatalf("dynamic-domain calls = %d, want 0", calls)
	}
}

const domainMembershipToolName = "tool"

type domainMembershipModeConfig struct{ Mode string }

func TestEffectiveDomainMembershipPreservesCanonicalAndAliasEntries(t *testing.T) {
	t.Parallel()
	type config = domainMembershipModeConfig
	definition := cli.Definition[config]{
		Name: domainMembershipToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					canonical := builder.Value("prod", "production").Description("Production")
					builder.Alias("p", canonical)
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	prod := domain.Entry("prod")
	alias := domain.Entry("p")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "p"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
		domain.Outcome(cli.ValueDomainOutcomeReady),
		domain.EntriesExactly(prod, alias),
		clitest.InspectionObservedValueDomainsExactly(domain),
		clitest.InspectionObservedValueDomainEntriesExactly(prod, alias),
	)
}

type dynamicDomainAuthoringModeConfig struct{ Mode string }

func TestNewDynamicValueDomainPreservesDynamicDomainAuthoringSemantics(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainAuthoringModeConfig
	callbackErr := errors.New("dynamic domain failed")
	definition := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					builder.Value("ready", "ready")
					return nil
				},
			)
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "ready"},
	}).Require(t, clitest.InspectionReady())

	definitionUnderTest := cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					return callbackErr
				},
			)
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
	}
	failed := clitest.Compile(t, definitionUnderTest)
	_, err := failed.Inspect(t.Context(), cli.InspectionRequest{
		Args: []string{"--mode", "ready"},
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("Inspect dynamic-domain failure = %v, want %v", err, callbackErr)
	}
}

const dynamicDomainMaterializationToolName = "tool"

type dynamicDomainMaterializationRootConfig struct{}

type dynamicDomainMaterializationModeUnrelatedTargetConfig struct {
	Mode      string
	Unrelated [64]string
}

type dynamicDomainMaterializationBulkConfig struct{}

type dynamicDomainMaterializationModeDecoyConfig struct{ Mode string }

type dynamicDomainMaterializationFirstSecondConfig struct{ First, Second string }

type dynamicDomainMaterializationModeConfig struct{ Mode string }

type dynamicDomainMaterializationRegionModeConfig struct{ Region, Mode string }

func TestDynamicDomainPublicationScopeClosesSiblingCommandTargetsWithoutImportingTheirOwnedRecords(t *testing.T) {
	t.Parallel()
	type rootConfig = dynamicDomainMaterializationRootConfig
	type targetConfig = dynamicDomainMaterializationModeUnrelatedTargetConfig
	type bulkConfig = dynamicDomainMaterializationBulkConfig
	type decoyConfig = dynamicDomainMaterializationModeDecoyConfig

	var targetCalls atomic.Int32
	var unrelatedCalls atomic.Int32
	var decoyCalls atomic.Int32
	definition := cli.Definition[rootConfig]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Subcommands.Command(cli.Command[targetConfig]{
				Name: "target",
				Declare: func(target *cli.CommandBuilder[targetConfig]) {
					domain := target.Values.Domain(
						func(_ context.Context, _ *targetConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
							targetCalls.Add(1)
							builder.Value("prod", "production")
							return nil
						},
					)
					mode := target.Values.Value(&target.Config.Mode, domain)
					target.Flags.ValueFor(mode).Long("domain")
					for index := range target.Config.Unrelated {
						unrelated := target.Values.Domain(
							func(_ context.Context, _ *targetConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
								unrelatedCalls.Add(1)
								builder.Value("unused", "unused")
								return nil
							},
						)
						target.Values.Value(&target.Config.Unrelated[index], unrelated)
					}
				},
			})
			root.Subcommands.Command(cli.Command[bulkConfig]{
				Name: "bulk",
				Declare: func(bulk *cli.CommandBuilder[bulkConfig]) {
					bulk.Subcommands.Command(cli.Command[decoyConfig]{
						Name: "decoy",
						Declare: func(decoy *cli.CommandBuilder[decoyConfig]) {
							domain := decoy.Values.Domain(
								func(_ context.Context, _ *decoyConfig, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
									decoyCalls.Add(1)
									builder.Value("unused", "unused")
									return nil
								},
							)
							mode := decoy.Values.Value(&decoy.Config.Mode, domain)
							decoy.Flags.ValueFor(mode).Long("domain")
						},
					})
				},
			})
		},
	}
	app := clitest.Compile(t, definition)

	target := clitest.Command("target")
	ready := target.Flag("--domain").Value().Domain(cli.ValueDomainUseDestination)
	decoy := clitest.Command("bulk", "decoy").Flag("--domain").Value().Domain(cli.ValueDomainUseDestination)
	domains := []clitest.ValueDomainLocator{ready}
	expectations := []clitest.InspectionExpectation{
		clitest.DiagnosticsExactly(),
		ready.Outcome(cli.ValueDomainOutcomeReady),
		ready.EntriesExactly(ready.Entry("prod")),
	}
	for index := 1; index <= len(targetConfig{}.Unrelated); index++ {
		domain := target.ValueAt(index).Domain(cli.ValueDomainUseDestination)
		domains = append(domains, domain)
		expectations = append(expectations, domain.Outcome(cli.ValueDomainOutcomeNotNeeded))
	}
	domains = append(domains, decoy)
	expectations = append(expectations,
		decoy.Outcome(cli.ValueDomainOutcomeNotNeeded),
		clitest.InspectionObservedValueDomainsExactly(domains...),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"target", "--domain", "prod"},
	}).Require(t, expectations...)
	if targetCalls.Load() != 1 || unrelatedCalls.Load() != 0 || decoyCalls.Load() != 0 {
		t.Fatalf(
			"dynamic-domain calls = target %d unrelated %d decoy %d, want 1/0/0",
			targetCalls.Load(),
			unrelatedCalls.Load(),
			decoyCalls.Load(),
		)
	}
}

func TestDynamicDomainSettlementConvergesConcurrentDemandOnOneOperation(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationFirstSecondConfig
	var calls atomic.Int32
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					calls.Add(1)
					builder.Value("one", "one")
					return nil
				},
			)
			first := root.Values.Value(&root.Config.First, domain)
			second := root.Values.Value(&root.Config.Second, domain)
			root.Flags.ValueFor(first).Long("first")
			root.Flags.ValueFor(second).Long("second")
		},
	}
	app := clitest.Compile(t, definition)
	first := clitest.RootCommand().Flag("--first").Value().Domain(cli.ValueDomainUseDestination)
	second := clitest.RootCommand().Flag("--second").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--first", "one", "--second", "one"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
		first.Outcome(cli.ValueDomainOutcomeReady),
		first.EntriesExactly(first.Entry("one")),
		first.UsesExactly(first, second),
		clitest.InspectionObservedValueDomainsExactly(first),
		clitest.InspectionObservedValueDomainEntriesExactly(first.Entry("one")),
	)
	if calls.Load() != 1 {
		t.Fatalf("one-operation shared-domain calls = %d, want 1", calls.Load())
	}
}

func TestDynamicDomainProviderBuilderExpiresAfterDemandedCallback(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationModeConfig
	var retained *cli.ValueDomainBuilder[string]
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					retained = builder
					builder.Value("one", "one")
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := func() (cli.Inspection, error) {
		proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
			Args: []string{"--mode", "one"},
		})
		proof.Require(t, clitest.DiagnosticsExactly())
		return proof.Inspection()
	}(); err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if retained == nil {
		t.Fatal("dynamic-domain provider did not receive builder")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("retained dynamic-domain provider builder remained usable after callback")
		}
	}()
	retained.Value("late", "late")
}

func TestDynamicDomainSettlementBlocksOnUnresolvedRequiredDependencyWithoutExecuting(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationRegionModeConfig
	var calls atomic.Int32
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			region := root.Facts.Require(&root.Config.Region)
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					calls.Add(1)
					return nil
				},
				region,
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--mode", "x"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
		domain.Outcome(cli.ValueDomainOutcomeBlocked),
		domain.EntriesExactly(),
		clitest.InspectionObservedValueDomainsExactly(domain),
		clitest.InspectionObservedValueDomainEntriesExactly(),
	)
	if calls.Load() != 0 {
		t.Fatalf("blocked dynamic domain executed %d times", calls.Load())
	}
}

func TestDynamicDomainSettlementCanceledDemandDoesNotExecuteOrCommit(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationModeConfig
	var calls atomic.Int32
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					calls.Add(1)
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	proof := clitest.Inspect(ctx, app, cli.InspectionRequest{
		Args: []string{"--mode", "x"},
	})
	proof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionCancellation).Present()),
		clitest.InspectionObservedValueDomainEntriesExactly(),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect() error = %v, want context cancellation", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("pre-canceled dynamic domain executed %d times", calls.Load())
	}
}

func TestDynamicDomainFailureAndSuccessfulEmptySnapshotRemainDistinct(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationModeConfig
	failure := errors.New("domain unavailable")
	failedCalls := 0
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					failedCalls++
					return failure
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	failed := clitest.Compile(t, definition)
	failedDomain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	failedProof := clitest.Inspect(t.Context(), failed, cli.InspectionRequest{
		Args: []string{"--mode", "x"},
	})
	failedProof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure).Present()),
		failedDomain.Outcome(cli.ValueDomainOutcomeFailed),
		failedDomain.EntriesExactly(),
		clitest.InspectionObservedValueDomainsExactly(failedDomain),
		clitest.InspectionObservedValueDomainEntriesExactly(),
	)
	_, err := failedProof.Inspection()
	if !errors.Is(err, failure) {
		t.Fatalf("Inspect(failed) error = %v", err)
	}
	if failedCalls != 1 {
		t.Fatalf("failed dynamic-domain calls = %d, want 1", failedCalls)
	}

	emptyCalls := 0
	definitionUnderTest := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					emptyCalls++
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	empty := clitest.Compile(t, definitionUnderTest)
	emptyDomain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Inspect(t.Context(), empty, cli.InspectionRequest{
		Args: []string{"--mode", "x"},
	}).Require(t,
		clitest.FailsWith(
			clitest.Diagnostic(cli.DiagnosticInvalidValueDomain).Absent(),
			clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure).Absent(),
		),
		emptyDomain.Outcome(cli.ValueDomainOutcomeReady),
		emptyDomain.EntriesExactly(),
		clitest.InspectionObservedValueDomainsExactly(emptyDomain),
		clitest.InspectionObservedValueDomainEntriesExactly(),
	)
	if emptyCalls != 1 {
		t.Fatalf("empty dynamic-domain calls = %d, want 1", emptyCalls)
	}
}

func TestDynamicDomainIndependentFailureRacingWithCancellationRemainsProvisionFailure(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationModeConfig
	failure := errors.New("independent dynamic-domain failure")
	callerCancellation := errors.New("caller cancellation")
	started := make(chan struct{})
	release := make(chan struct{})
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error {
					close(started)
					<-release
					return failure
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	ctx, cancel := context.WithCancelCause(t.Context())
	result := make(chan clitest.InspectionResult, 1)
	go func() {
		result <- clitest.Inspect(ctx, app, cli.InspectionRequest{
			Args: []string{"--mode", "x"},
		})
	}()
	<-started
	cancel(callerCancellation)
	<-ctx.Done()
	close(release)
	proof := <-result
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticsExactly(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure))),
		domain.Outcome(cli.ValueDomainOutcomeFailed),
		domain.EntriesExactly(),
		clitest.InspectionObservedValueDomainsExactly(domain),
		clitest.InspectionObservedValueDomainEntriesExactly(),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, failure) {
		t.Fatalf("Inspect() error = %v, want independent dynamic-domain failure", err)
	}
	if !errors.Is(err, callerCancellation) {
		t.Fatalf("Inspect() error = %v, want independent caller cancellation", err)
	}
}

func TestDynamicDomainResolvesDeclaredDependenciesBeforeDemandedCallback(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationRegionModeConfig
	calls := 0
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			region := root.Values.String(&root.Config.Region)
			root.Flags.ValueFor(region).Long("region")
			domain := root.Values.Domain(
				func(_ context.Context, cfg *config, dependencies cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					calls++
					value, ok := dependencies.Get(region)
					if !ok || value != "us" || cfg.Region != "us" {
						return errors.New("dynamic domain dependency unavailable")
					}
					canonical := builder.Value("us-prod", "production")
					builder.Alias("u", canonical)
					return nil
				},
				region,
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"--region", "us", "--mode", "u"},
	}).Require(t,
		clitest.DiagnosticsExactly(),
		domain.Outcome(cli.ValueDomainOutcomeReady),
		domain.DependenciesExactly(clitest.RootCommand().Flag("--region").Value()),
		domain.EntriesExactly(domain.Entry("us-prod"), domain.Entry("u")),
		clitest.InspectionObservedValueDomainsExactly(domain),
	)
	if calls != 1 {
		t.Fatalf("dynamic domain calls = %d, want 1", calls)
	}
}

func TestDynamicDomainRejectsTypedContributionOutsideAcceptedSnapshot(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationModeConfig
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					builder.Value("prod", "production")
					return nil
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Values.ComputedDefault(
				mode,
				cli.Value.ComputedBy(
					func(context.Context, *config, cli.Dependencies) (string, error) { return "other", nil },
				),
			)
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().ValueAt(0).Domain(cli.ValueDomainUseDestination)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{}).Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
		domain.Outcome(cli.ValueDomainOutcomeReady),
		domain.EntriesExactly(domain.Entry("prod")),
		clitest.InspectionObservedValueDomainsExactly(domain),
	)
}

func TestDynamicDomainCancellationAfterFinalValidationBeforeLinearizationLeavesNoAcceptedDomain(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationModeConfig
	contributionReady := make(chan struct{})
	definition := cli.Definition[config]{
		Name: dynamicDomainMaterializationToolName,
		Help: cli.HelpConfig{
			Disabled: true,
		},
		ShellCompletion: cli.ShellCompletion{
			Disabled: true,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(ctx context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
					builder.Value("prod", "production")
					close(contributionReady)
					<-ctx.Done()
					return ctx.Err()
				},
			)
			mode := root.Values.Value(&root.Config.Mode, domain)
			root.Flags.ValueFor(mode).Long("mode")
		},
	}
	app := clitest.Compile(t, definition)
	domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan clitest.InspectionResult, 1)
	go func() {
		result <- clitest.Inspect(ctx, app, cli.InspectionRequest{
			Args: []string{"--mode", "prod"},
		})
	}()
	<-contributionReady
	cancel()
	proof := <-result
	proof.Require(t,
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionCancellation).Present()),
		domain.Outcome(cli.ValueDomainOutcomeCanceled),
		domain.EntriesExactly(),
		clitest.InspectionObservedValueDomainsExactly(domain),
		clitest.InspectionObservedValueDomainEntriesExactly(),
	)
	_, err := proof.Inspection()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect() error = %v, want cancellation", err)
	}
}

func TestDynamicDomainCancellationDuringCallbackAndAfterLinearizedPublication(t *testing.T) {
	t.Parallel()
	type config = dynamicDomainMaterializationModeConfig

	t.Run("during callback", func(t *testing.T) {
		t.Parallel()
		started := make(chan struct{})
		definition := cli.Definition[config]{
			Name: dynamicDomainMaterializationToolName,
			Help: cli.HelpConfig{
				Disabled: true,
			},
			ShellCompletion: cli.ShellCompletion{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				domain := root.Values.Domain(
					func(ctx context.Context, _ *config, _ cli.Dependencies, _ *cli.ValueDomainBuilder[string]) error {
						close(started)
						<-ctx.Done()
						return ctx.Err()
					},
				)
				mode := root.Values.Value(&root.Config.Mode, domain)
				root.Flags.ValueFor(mode).Long("mode")
			},
		}
		app := clitest.Compile(t, definition)
		domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan clitest.InspectionResult, 1)
		go func() {
			result <- clitest.Inspect(ctx, app, cli.InspectionRequest{
				Args: []string{"--mode", "late"},
			})
		}()
		<-started
		cancel()
		proof := <-result
		proof.Require(t,
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionCancellation).Present()),
			domain.Outcome(cli.ValueDomainOutcomeCanceled),
			clitest.InspectionObservedValueDomainsExactly(domain),
			clitest.InspectionObservedValueDomainEntriesExactly(),
		)
		_, err := proof.Inspection()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("callback cancellation error = %v, want context.Canceled", err)
		}
	})

	t.Run("after successful publication", func(t *testing.T) {
		t.Parallel()
		definition := cli.Definition[config]{
			Name: dynamicDomainMaterializationToolName,
			Help: cli.HelpConfig{
				Disabled: true,
			},
			ShellCompletion: cli.ShellCompletion{
				Disabled: true,
			},
			Declare: func(root *cli.CommandBuilder[config]) {
				domain := root.Values.Domain(
					func(_ context.Context, _ *config, _ cli.Dependencies, builder *cli.ValueDomainBuilder[string]) error {
						builder.Value("prod", "production")
						return nil
					},
				)
				mode := root.Values.Value(&root.Config.Mode, domain)
				root.Flags.ValueFor(mode).Long("mode")
			},
		}
		app := clitest.Compile(t, definition)
		domain := clitest.RootCommand().Flag("--mode").Value().Domain(cli.ValueDomainUseDestination)
		entry := domain.Entry("prod")
		ctx, cancel := context.WithCancel(t.Context())
		proof := clitest.Inspect(ctx, app, cli.InspectionRequest{
			Args: []string{"--mode", "prod"},
		})
		cancel()
		proof.Require(t,
			clitest.InspectionReady(),
			domain.Outcome(cli.ValueDomainOutcomeReady),
			domain.EntriesExactly(entry),
			clitest.InspectionObservedValueDomainsExactly(domain),
			clitest.InspectionObservedValueDomainEntriesExactly(entry),
		)
	})
}

type operationPreparationRegionModeConfig struct{ Region, Mode string }

type operationPreparationModeConfig struct{ Mode string }

func TestRunAndInspectionDynamicDomainIncompleteRemainIncomplete(t *testing.T) {
	t.Parallel()
	type config = operationPreparationRegionModeConfig
	handlerCalled := false
	definition := cli.Definition[config]{
		Name: "domain-incomplete",
		Declare: func(root *cli.CommandBuilder[config]) {
			region := root.Facts.Require(&root.Config.Region)
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error { return nil },
				region,
			)
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
		Run: func(context.Context, *config) error {
			handlerCalled = true
			return nil
		},
	}
	app := clitest.Compile(t, definition)
	args := []string{"--mode", "x"}

	runErr := app.Run(t.Context(), cli.RunRequest{
		Args: args,
	})
	if runErr == nil {
		t.Fatal("Run() error = nil, want incomplete invocation to fail closed")
	}
	if handlerCalled {
		t.Fatal("Run() executed handler for incomplete invocation")
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: args,
	}).Require(t, clitest.InspectionIncomplete())
}

func TestRunAndInspectionDynamicDomainFailureRemainRejected(t *testing.T) {
	t.Parallel()
	type config = operationPreparationModeConfig
	cause := errors.New("domain failure")
	definition := cli.Definition[config]{
		Name: "domain-failure",
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := root.Values.Domain(
				func(context.Context, *config, cli.Dependencies, *cli.ValueDomainBuilder[string]) error { return cause },
			)
			root.Flags.Value(&root.Config.Mode, domain).Long("mode")
		},
		Run: func(context.Context, *config) error { return nil },
	}
	app := clitest.Compile(t, definition)
	args := []string{"--mode", "x"}

	if err := app.Run(t.Context(), cli.RunRequest{
		Args: args,
	}); !errors.Is(err, cause) {
		t.Fatalf("Run() error = %v, want %v", err, cause)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: args,
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticValueDomainProvisionFailure).Present()),
	)
	_, inspectErr := proof.Inspection()
	if !errors.Is(inspectErr, cause) {
		t.Fatalf("Inspect() error = %v, want %v", inspectErr, cause)
	}
}
