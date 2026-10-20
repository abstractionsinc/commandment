package conformance

import (
	"errors"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func valueFuzzInspect(t *testing.T, app cli.Application, request cli.InspectionRequest) (cli.Inspection, error) {
	t.Helper()
	return clitest.Inspect(t.Context(), app, request).Inspection()
}

var _ = errors.Is

func valueFuzzBoundedFuzzText(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}

func valueFuzzFuzzASCIIFragment(value string, maximum int, preserveEmpty bool) string {
	value = valueFuzzBoundedFuzzText(value, maximum)
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

type valueFuzzRawModeLabelsConfig struct {
	Raw    string
	Mode   string
	Labels map[string]string
}

func FuzzValueLanguageAndFiniteDomainProperties(f *testing.F) {
	type config = valueFuzzRawModeLabelsConfig
	codec := cli.ValueCodec[string]{
		Name:     "canonical-lower",
		Parse:    func(text string) (string, error) { return strings.ToUpper(text), nil },
		Format:   strings.ToLower,
		Snapshot: cli.Value.Assign[string](),
	}
	app, err := cli.Compile(cli.Definition[config]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[config]) {
			domain := cli.Value.Domain(cli.Value.Assign[string](), func(builder *cli.ValueDomainBuilder[string]) {
				canonical := builder.Value("prod", "production")
				builder.Alias("p", canonical)
			})
			raw := root.Values.Value(&root.Config.Raw, codec)
			mode := root.Values.Value(&root.Config.Mode, domain)
			labels := root.Values.Keyed(
				&root.Config.Labels,
				cli.Value.Keyed(cli.Value.String(), cli.Value.String()).KeyOnly("implicit"),
			)
			root.Flags.ValueFor(raw).Long("raw")
			root.Flags.ValueFor(mode).Long("mode")
			root.Values.Environment(labels, "TOOL_LABELS").KeyedEntries(",")
		},
	})
	if err != nil {
		f.Fatalf("Compile() error = %v", err)
	}
	schema := clitest.Schema(app).Schema()
	var rawID, modeID, labelsID cli.ValueID
	for _, spelling := range schema.FlagSpellings {
		switch spelling.Spelling.Text {
		case "--raw":
			rawID = schema.Flags[spelling.Flag].Destination
		case "--mode":
			modeID = schema.Flags[spelling.Flag].Destination
		}
	}
	for _, input := range schema.EnvironmentInputs {
		if input.Name == "TOOL_LABELS" {
			labelsID = input.Destination
		}
	}
	if rawID == 0 || modeID == 0 || labelsID == 0 {
		f.Fatalf("fuzz destination identities missing: raw=%v mode=%v labels=%v", rawID, modeID, labelsID)
	}

	f.Add("MiXeD", "prod", "name", "value", false)
	f.Add("value", "p", "enabled", "ignored", true)
	f.Add("value", "unknown", "key", "value", false)
	// Keep the representation-collision case deterministic: raw and mode may
	// legitimately canonicalize to the same text while remaining distinct values.
	f.Add("prod", "prod", "collision", "value", false)
	f.Add("value", "prod", "", "ignored", true)
	f.Fuzz(func(t *testing.T, raw, mode, key, keyedValue string, keyOnly bool) {
		raw = valueFuzzFuzzASCIIFragment(raw, 32, true)
		mode = valueFuzzFuzzASCIIFragment(mode, 16, true)
		key = valueFuzzFuzzASCIIFragment(key, 20, true)
		keyedValue = valueFuzzFuzzASCIIFragment(keyedValue, 20, true)
		// An empty environment value encodes an empty keyed collection, not a
		// key-only entry with an empty key. Keep this property inside the
		// representable key-only-entry domain.
		if keyOnly && key == "" {
			key = "key"
		}
		entryText := key + "=" + keyedValue
		wantKeyedValue := keyedValue
		if keyOnly {
			entryText = key
			wantKeyedValue = "implicit"
		}
		inspection, inspectErr := valueFuzzInspect(t, app, cli.InspectionRequest{
			Args: []string{"--raw", raw, "--mode", mode},
			Env: cli.Env{{
				Name:  "TOOL_LABELS",
				Value: entryText,
			}},
		})
		if mode != "prod" && mode != "p" {
			if inspectErr == nil && inspection.Status == cli.InspectionStatusReady {
				t.Fatalf("finite domain accepted unknown spelling %q", mode)
			}
			return
		}
		if inspectErr != nil || inspection.Status != cli.InspectionStatusReady {
			t.Fatalf("accepted domain spelling %q rejected: status=%v err=%v", mode, inspection.Status, inspectErr)
		}
		rawObserved, rawOK := inspection.Values[rawID]
		seenRaw := rawOK && rawObserved.Representation != nil && !rawObserved.Representation.Redacted &&
			rawObserved.Representation.Text == strings.ToLower(raw)

		modeObserved, modeOK := inspection.Values[modeID]
		seenMode := modeOK && modeObserved.Representation != nil && !modeObserved.Representation.Redacted &&
			modeObserved.Representation.Text == "prod"

		labelsObserved, labelsOK := inspection.Values[labelsID]
		seenKeyed := false
		if labelsOK {
			for _, ref := range labelsObserved.Entries {
				candidate, ok := inspection.ValueCandidates[ref.Candidate]
				if !ok || ref.Index < 0 || ref.Index >= len(candidate.Entries) {
					t.Fatalf("broken keyed entry reference: %#v", ref)
				}
				entry := candidate.Entries[ref.Index]
				if entry.Key.Representation != nil && entry.Value.Representation != nil &&
					entry.Key.Representation.Text == key &&
					entry.Value.Representation.Text == wantKeyedValue {
					seenKeyed = true
				}
			}
		}
		if !seenRaw || !seenMode || !seenKeyed {
			t.Fatalf(
				"canonical formatter/domain/keyed authority missing: raw=%v mode=%v keyed=%v values=%#v",
				seenRaw,
				seenMode,
				seenKeyed,
				inspection.Values,
			)
		}
	})
}
