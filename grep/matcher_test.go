package grep

import (
	"io/ioutil"
	"strings"
	"testing"
)

func loadData() []string {
	data, err := ioutil.ReadFile("benchfiles.txt")
	if err != nil {
		panic(err)
	}
	s := string(data)
	return strings.Split(s, "\n")
}

func runBenchmark(b *testing.B, query string, limit int) {
	lines := loadData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher := FuzzyMatcher{Query: query, Limit: limit}
		for _, l := range lines {
			matcher.Match(l)
		}
		matcher.Results()
		// fmt.Printf("%d results %d\n", len(r), matcher.TotalMatches())
		// StatsHack()
	}
}

func BenchmarkFuzzyMatcherUnlimitedLong(b *testing.B) {
	runBenchmark(b, "decoder", 0)
}

func BenchmarkFuzzyMatcherLimitedLong(b *testing.B) {
	runBenchmark(b, "decoder", 50)
}

func BenchmarkFuzzyMatcherUnlimitedShort(b *testing.B) {
	runBenchmark(b, "a", 0)
}

func BenchmarkFuzzyMatcherLimitedShort(b *testing.B) {
	runBenchmark(b, "a", 50)
}
