package benchmark

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clibench"
	"github.com/abstractionsinc/commandment/clitest"
)

// BenchmarkCompleteMatchingCandidates varies candidate cardinality independently for each public match mode.
func BenchmarkCompleteMatchingCandidates(b *testing.B) {
	const query = "aaaaaabc"
	for _, mode := range []cli.CompletionMatchMode{
		cli.CompletionMatchModePrefix,
		cli.CompletionMatchModeFuzzy,
		cli.CompletionMatchModeApproximate,
	} {
		for _, count := range [...]int{16, 64, 256, 1024, 4096} {
			candidates := completionCandidateCorpus(count, query, 32)
			app := completionCandidateApplication(b, candidates)
			request := completionCandidateRequest(query, mode, []cli.CompletionMatchField{cli.CompletionMatchFieldValue})
			preflightCompletionCandidateMatch(b, app, request, candidates[len(candidates)-1].Value)

			b.Run(fmt.Sprintf("%s/Candidates/N=%d", completionMatchModeName(mode), count), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Complete(b, app, request)
			})
		}
	}
}

// BenchmarkCompleteMatchingTextLength holds candidate count fixed while query and candidate text lengths grow.
func BenchmarkCompleteMatchingTextLength(b *testing.B) {
	const candidates = 64
	for _, mode := range []cli.CompletionMatchMode{
		cli.CompletionMatchModeSubstring,
		cli.CompletionMatchModeFuzzy,
		cli.CompletionMatchModeApproximate,
	} {
		for _, queryBytes := range [...]int{4, 8, 16, 32, 64} {
			fieldBytes := 4 * queryBytes
			query, text := completionTextAdversary(mode, queryBytes, fieldBytes)
			corpus := make([]cli.CompletionCandidate, candidates)
			for index := range corpus {
				corpus[index] = cli.CompletionCandidate{Value: text}
			}
			app := completionCandidateApplication(b, corpus)
			request := completionCandidateRequest(query, mode, []cli.CompletionMatchField{cli.CompletionMatchFieldValue})
			preflightCompletionItems(b, app, request, 0)

			b.Run(fmt.Sprintf("%s/QueryBytes=%d/FieldBytes=%d/Candidates=%d", completionMatchModeName(mode), queryBytes, fieldBytes, candidates), func(b *testing.B) {
				b.ReportAllocs()
				clibench.Complete(b, app, request)
			})
		}
	}
}

// BenchmarkCompleteMatchingFields isolates field-selection and search-term breadth at fixed candidate cardinality.
func BenchmarkCompleteMatchingFields(b *testing.B) {
	const (
		query      = "aaaaaabc"
		fieldBytes = 32
		candidates = 256
	)
	for _, layout := range completionFieldLayouts(query, fieldBytes) {
		corpus := make([]cli.CompletionCandidate, candidates)
		for index := range corpus {
			corpus[index] = cloneCompletionCandidate(layout.candidate)
		}
		app := completionCandidateApplication(b, corpus)
		request := completionCandidateRequest(query, cli.CompletionMatchModePrefix, layout.fields)
		preflightCompletionFieldLayout(b, app, request, layout)

		b.Run(layout.name, func(b *testing.B) {
			b.ReportAllocs()
			clibench.Complete(b, app, request)
		})
	}
}

// BenchmarkCompleteRanking varies candidate count, ranking mode, and result limit without timing fixture construction.
func BenchmarkCompleteRanking(b *testing.B) {
	for _, ranking := range []cli.CompletionRankingMode{
		cli.CompletionRankingModeSource,
		cli.CompletionRankingModeLexical,
		cli.CompletionRankingModeRelevance,
	} {
		for _, count := range [...]int{16, 64, 256, 1024, 4096} {
			candidates := completionRankingCandidates(count)
			app := completionCandidateApplication(b, candidates)
			for _, limit := range [...]int{0, 16} {
				request := completionCandidateRequest("", cli.CompletionMatchModeUnfiltered, nil)
				request.Client.Ranking = ranking
				request.Limit = limit
				want := count
				if limit != 0 && limit < want {
					want = limit
				}
				preflightCompletionRanking(b, app, request, candidates, ranking, want)

				b.Run(fmt.Sprintf("%s/Candidates/N=%d/Limit=%d", completionRankingModeName(ranking), count, limit), func(b *testing.B) {
					b.ReportAllocs()
					clibench.Complete(b, app, request)
				})
			}
		}
	}
}

func completionCandidateApplication(tb testing.TB, candidates []cli.CompletionCandidate) cli.Application {
	tb.Helper()
	definition := cli.Definition[inputConfig]{
		Name: "tool",
		Help: cli.HelpConfig{Disabled: true},
		Declare: func(root *cli.CommandBuilder[inputConfig]) {
			root.Flags.String(&root.Config.Value).
				CompleteBy(func(context.Context, *inputConfig, cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
					return cli.CompletionContribution{Candidates: candidates}, nil
				}).
				Long("value")
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		tb.Fatalf("completion matching compile: %v", err)
	}
	return app
}

func completionCandidateRequest(query string, mode cli.CompletionMatchMode, fields []cli.CompletionMatchField) cli.CompletionRequest {
	return cli.CompletionRequest{
		Args: []string{"--value", query},
		Cursor: cli.CompletionCursor{
			Argument: 1,
			Byte:     len(query),
		},
		Client: cli.CompletionClient{
			Match: cli.CompletionMatchOptions{
				Mode:   mode,
				Case:   cli.CompletionCaseSensitive,
				Fields: append([]cli.CompletionMatchField(nil), fields...),
			},
			Capabilities: cli.CompletionClientCapabilities{MatchRanges: true},
		},
	}
}

func preflightCompletionCandidateMatch(tb testing.TB, app cli.Application, request cli.CompletionRequest, expected string) {
	tb.Helper()
	proof := clitest.Complete(tb.Context(), app, request)
	proof.Require(tb,
		clitest.CompletionItemsExactly(clitest.CompletionItem(expected)),
		clitest.CompletionItemAt(0).Value(expected),
		clitest.CompletionItemAt(0).MatchField(cli.CompletionMatchFieldValue),
	)
}

func preflightCompletionRanking(tb testing.TB, app cli.Application, request cli.CompletionRequest, candidates []cli.CompletionCandidate, ranking cli.CompletionRankingMode, want int) {
	tb.Helper()
	proof := clitest.Complete(tb.Context(), app, request)
	proof.Require(tb)
	if tb.Failed() {
		return
	}
	completion, _ := proof.Completion()
	if len(completion.Items) != want {
		tb.Fatalf("Complete ranking preflight items=%d want=%d", len(completion.Items), want)
	}
	if want == 0 {
		return
	}
	expected := completionRankingFirstCandidate(candidates, ranking)
	proof.Require(tb,
		clitest.CompletionItemAt(0).Value(expected.Value),
		clitest.CompletionItemAt(0).Priority(expected.Priority),
		clitest.CompletionItemAt(0).SortText(expected.SortText),
	)
	if request.Limit != 0 && request.Limit < len(candidates) {
		proof.Require(tb, clitest.CompletionTruncated())
	}
}

func completionRankingFirstCandidate(candidates []cli.CompletionCandidate, ranking cli.CompletionRankingMode) cli.CompletionCandidate {
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if completionCandidatePrecedes(candidate, best, ranking) {
			best = candidate
		}
	}
	return best
}

func completionCandidatePrecedes(left, right cli.CompletionCandidate, ranking cli.CompletionRankingMode) bool {
	if ranking == cli.CompletionRankingModeSource {
		return false
	}
	if ranking == cli.CompletionRankingModeRelevance && left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	leftText := left.SortText
	if leftText == "" {
		leftText = left.Value
	}
	rightText := right.SortText
	if rightText == "" {
		rightText = right.Value
	}
	if leftText != rightText {
		return leftText < rightText
	}
	return left.Value < right.Value
}

func preflightCompletionItems(tb testing.TB, app cli.Application, request cli.CompletionRequest, want int) {
	tb.Helper()
	proof := clitest.Complete(tb.Context(), app, request)
	proof.Require(tb)
	if tb.Failed() {
		return
	}
	completion, _ := proof.Completion()
	if len(completion.Items) != want {
		tb.Fatalf("Complete preflight items=%d want=%d", len(completion.Items), want)
	}
}

func completionCandidateCorpus(count int, query string, fieldBytes int) []cli.CompletionCandidate {
	noMatch := strings.Repeat("a", fieldBytes)
	match := query + strings.Repeat("a", fieldBytes-len(query))
	candidates := make([]cli.CompletionCandidate, count)
	for index := range candidates {
		candidates[index] = cli.CompletionCandidate{Value: noMatch}
	}
	candidates[len(candidates)-1] = cli.CompletionCandidate{Value: match}
	return candidates
}

func completionTextAdversary(mode cli.CompletionMatchMode, queryBytes, fieldBytes int) (string, string) {
	text := strings.Repeat("a", fieldBytes)
	switch mode {
	case cli.CompletionMatchModeSubstring, cli.CompletionMatchModeApproximate:
		return strings.Repeat("a", queryBytes-2) + "bc", text
	case cli.CompletionMatchModeFuzzy:
		prefix := 4
		if prefix >= queryBytes {
			prefix = queryBytes - 1
		}
		return strings.Repeat("a", prefix) + "b" + strings.Repeat("a", queryBytes-prefix-1), text
	default:
		panic("unsupported completion match mode")
	}
}

type completionFieldLayout struct {
	name       string
	fields     []cli.CompletionMatchField
	candidate  cli.CompletionCandidate
	wantItems  int
	matchField cli.CompletionMatchField
}

func completionFieldLayouts(query string, fieldBytes int) []completionFieldLayout {
	noMatch := strings.Repeat("a", fieldBytes)
	match := query + strings.Repeat("a", fieldBytes-len(query))
	searchTerms := func(count int) []string {
		terms := make([]string, count)
		for index := range terms {
			digits := fieldBytes - len("miss-")
			terms[index] = fmt.Sprintf("miss-%0*d", digits, index)
		}
		if count != 0 {
			terms[count-1] = match
		}
		return terms
	}
	return []completionFieldLayout{
		{
			name:       "ValueOnly",
			fields:     []cli.CompletionMatchField{cli.CompletionMatchFieldValue},
			candidate:  cli.CompletionCandidate{Value: match},
			wantItems:  256,
			matchField: cli.CompletionMatchFieldValue,
		},
		{
			name: "DefaultFields",
			candidate: cli.CompletionCandidate{
				Value:       noMatch,
				DisplayText: noMatch,
				SearchTerms: searchTerms(2),
			},
			wantItems:  256,
			matchField: cli.CompletionMatchFieldSearchTerm,
		},
		{
			name:   "ExplicitFields/Count=5",
			fields: []cli.CompletionMatchField{cli.CompletionMatchFieldValue, cli.CompletionMatchFieldDisplayText, cli.CompletionMatchFieldDescription, cli.CompletionMatchFieldDetail, cli.CompletionMatchFieldSearchTerm},
			candidate: cli.CompletionCandidate{
				Value:       noMatch,
				DisplayText: noMatch,
				Description: noMatch,
				Detail:      noMatch,
				SearchTerms: searchTerms(1),
			},
			wantItems:  256,
			matchField: cli.CompletionMatchFieldSearchTerm,
		},
		{
			name:      "SearchTerms/N=0",
			fields:    []cli.CompletionMatchField{cli.CompletionMatchFieldSearchTerm},
			candidate: cli.CompletionCandidate{},
		},
		{
			name:   "SearchTerms/N=4",
			fields: []cli.CompletionMatchField{cli.CompletionMatchFieldSearchTerm},
			candidate: cli.CompletionCandidate{
				Value:       noMatch,
				SearchTerms: searchTerms(4),
			},
			wantItems:  256,
			matchField: cli.CompletionMatchFieldSearchTerm,
		},
		{
			name:   "SearchTerms/N=16",
			fields: []cli.CompletionMatchField{cli.CompletionMatchFieldSearchTerm},
			candidate: cli.CompletionCandidate{
				Value:       noMatch,
				SearchTerms: searchTerms(16),
			},
			wantItems:  256,
			matchField: cli.CompletionMatchFieldSearchTerm,
		},
		{
			name:   "SearchTerms/N=64",
			fields: []cli.CompletionMatchField{cli.CompletionMatchFieldSearchTerm},
			candidate: cli.CompletionCandidate{
				Value:       noMatch,
				SearchTerms: searchTerms(64),
			},
			wantItems:  256,
			matchField: cli.CompletionMatchFieldSearchTerm,
		},
		{
			name:   "DuplicateValueDisplay",
			fields: []cli.CompletionMatchField{cli.CompletionMatchFieldValue, cli.CompletionMatchFieldDisplayText},
			candidate: cli.CompletionCandidate{
				Value:       noMatch,
				DisplayText: noMatch,
			},
		},
	}
}

func preflightCompletionFieldLayout(tb testing.TB, app cli.Application, request cli.CompletionRequest, layout completionFieldLayout) {
	tb.Helper()
	proof := clitest.Complete(tb.Context(), app, request)
	proof.Require(tb)
	if tb.Failed() {
		return
	}
	completion, _ := proof.Completion()
	if len(completion.Items) != layout.wantItems {
		tb.Fatalf("Complete field-layout preflight items=%d want=%d", len(completion.Items), layout.wantItems)
	}
	if layout.wantItems == 0 {
		return
	}
	proof.Require(tb,
		clitest.CompletionItemAt(0).Value(layout.candidate.Value),
		clitest.CompletionItemAt(0).MatchField(layout.matchField),
	)
}

func cloneCompletionCandidate(candidate cli.CompletionCandidate) cli.CompletionCandidate {
	candidate.SearchTerms = append([]string(nil), candidate.SearchTerms...)
	return candidate
}

func completionRankingCandidates(count int) []cli.CompletionCandidate {
	candidates := make([]cli.CompletionCandidate, count)
	half := count / 2
	for index := range candidates {
		base := index % half
		sortIndex := (base * 37) % half
		valueIndex := (base * 53) % half
		sortText := fmt.Sprintf("sort-%05d", sortIndex)
		if base%4 == 0 {
			sortText = ""
		}
		candidates[index] = cli.CompletionCandidate{
			Value:    fmt.Sprintf("value-%05d-%05d", valueIndex, index),
			SortText: sortText,
			Priority: base % 7,
			Kind:     cli.CompletionCandidateValue,
		}
	}
	return candidates
}

func completionMatchModeName(mode cli.CompletionMatchMode) string {
	switch mode {
	case cli.CompletionMatchModePrefix:
		return "Prefix"
	case cli.CompletionMatchModeSubstring:
		return "Substring"
	case cli.CompletionMatchModeFuzzy:
		return "Fuzzy"
	case cli.CompletionMatchModeApproximate:
		return "Approximate"
	default:
		return fmt.Sprintf("Mode-%d", mode)
	}
}

func completionRankingModeName(mode cli.CompletionRankingMode) string {
	switch mode {
	case cli.CompletionRankingModeSource:
		return "Source"
	case cli.CompletionRankingModeLexical:
		return "Lexical"
	case cli.CompletionRankingModeRelevance:
		return "Relevance"
	default:
		return fmt.Sprintf("Ranking-%d", mode)
	}
}
