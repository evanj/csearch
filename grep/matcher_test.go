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

func indexedBenchmark(b *testing.B, query string, limit int) int {
	lines := loadData()
	indexed := IndexedMatcher{}
	for _, l := range lines {
		indexed.Add(l)
	}
	b.ResetTimer()

	totalMatches := 0
	for i := 0; i < b.N; i++ {
		matches := indexed.Match(query, limit)
		totalMatches += len(matches)
	}
	return totalMatches
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

func BenchmarkIndexedMatcherUnlimitedLong(b *testing.B) {
	indexedBenchmark(b, "decoder", 0)
}

func BenchmarkIndexedMatcherLimitedLong(b *testing.B) {
	indexedBenchmark(b, "decoder", 50)
}

func BenchmarkIndexedMatcherUnlimitedShort(b *testing.B) {
	indexedBenchmark(b, "a", 0)
}

func BenchmarkIndexedMatcherLimitedShort(b *testing.B) {
	indexedBenchmark(b, "a", 50)
}
