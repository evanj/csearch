package grep

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func TestIsWordStart(t *testing.T) {
	strings := []string{
		"SampleWord",
		"sampleWord",
		"sample-word",
		"--sample---word---",
		"--SAMPLE---WORD---",
	}

	for _, s := range strings {
		previous := wordStartInitialRune
		for _, r := range s {
			lr := unicode.ToLower(r)
			if lr == 's' || lr == 'w' {
				if !isWordStart(previous, r) {
					t.Errorf("%s: expected %c -> %c to be a word", previous, r)
				}
			} else if isWordStart(previous, r) {
				t.Errorf("%s: expected %c -> %c to not be a word", previous, r)
			}
			previous = r
		}
	}
}

func TestFuzzyMatch(t *testing.T) {
	scoreOrder := []string{
		"a/b/c/TypeAheadHandler.java",
		"a/b/c/PlaceType.java",
		"a/b/type/Foo.java",
		"a/b/ctyped/Foo.java",
	}
	assertOrder(t, scoreOrder, "type")
}

func assertOrder(t *testing.T, scoreOrder []string, query string) {
	prevScore := 10000
	prevPath := ""
	for _, path := range scoreOrder {
		score := FuzzyMatch(path, query)
		// log.Printf("%s %d", path, score)
		if score >= prevScore {
			t.Errorf("%s (%d) should be less than %s (%d)", path, score, prevPath, prevScore)
		}

		prevScore = score
		prevPath = path
	}
	if prevScore < 0 {
		t.Errorf("%s (%d) should match", prevPath, prevScore)
	}
}

func TestFuzzyFilename(t *testing.T) {
	scoreOrder := []string{
		"f/i/l/File.java",
		"f/i/l/file.txt",
		"a/b/c/examplefile.txt",
		"a/f/i/le.txt",
	}
	assertOrder(t, scoreOrder, "File")
}

func TestAbbreviations(t *testing.T) {
	scoreOrder2 := []string{
		"science/src/thrift/com/twitter/ads/adserver/adserver_new_rpc.thrift",
		"science/tests/resources/com/twitter/ads/dataservice/validation/rules/card/ValidateLeadGenCardRulesIT.testDefaultCard_response_logs.txt",
	}
	assertOrder(t, scoreOrder2, "anr")
	scoreOrder := []string{
		"/Users/ejones/workspace/science/src/thrift/com/twitter/ads/adserver/adserver_new_rpc.thrift",
		"/Users/ejones/workspace/science/tests/resources/com/twitter/ads/dataservice/validation/rules/card/ValidateLeadGenCardRulesIT.testDefaultCard_response_logs.txt",
	}
	assertOrder(t, scoreOrder, "anr")
}

func TestWordMatch(t *testing.T) {
	scoreOrder := []string{
		// "foo/bar/viewer.thrift",
		"foo/harViewer.js",
		"foo/reviewer.html",
		// "foo/view_alerts.html",
		// "baz/har-viewer/something.js",
		// "path/CreativeWriterWithExisting_response.txt",
	}
	assertOrder(t, scoreOrder, "viewer")

	other := []string{
		"path/apt",
		"path/api_test",
	}
	assertOrder(t, other, "apt")
}

func TestShorterFilenamesScoreHigher(t *testing.T) {
	scoreOrder := []string{
		"zzzreally/longbutbetter/git",
		"zzzlonger/git.a",
		"zzlong/path/git.py",
		"ashort/git-foo.py",
	}
	assertOrder(t, scoreOrder, "git")
}

func matchValues(matcher *FuzzyMatcher) {
	values := []string{
		"he-match-llo",
		"helloworld",
		"goodbye",
		"hello",
	}
	for _, v := range values {
		matcher.Match(v)
	}
}

func TestFuzzyMatcherUnlimited(t *testing.T) {
	matcher := FuzzyMatcher{Query: "hello"}
	matchValues(&matcher)

	output := matcher.Results()
	expected := []string{
		"hello",
		"helloworld",
		"he-match-llo",
	}
	if !reflect.DeepEqual(expected, output) {
		t.Error("unexpected results", expected, output)
	}
}

func TestFuzzyMatcherLimited(t *testing.T) {
	matcher := FuzzyMatcher{Query: "hello", Limit: 2}
	matchValues(&matcher)
	output := matcher.Results()
	expected := []string{
		"hello",
		"helloworld",
	}
	if !reflect.DeepEqual(expected, output) {
		t.Error("unexpected results", expected, output)
	}
}

func TestContainsBytesFuzzy(t *testing.T) {
	const query = "abcde"

	bad := []string{
		"",
		"abcd",
		"----a-b-c-d----",
		"------",
		"Abcde",
	}
	good := []string{
		"abcde",
		"-a-b-c-d-e-",
	}
	for _, s := range bad {
		if containsBytesFuzzy(s, query) {
			t.Error(s + " must not contain " + query + " (returned true)")
		}
	}
	for _, s := range good {
		if !containsBytesFuzzy(s, query) {
			t.Error(s + " must contain " + query + " (returned false)")
		}
	}
}

func TestContainsBytesFuzzyInsensitive(t *testing.T) {
	const query = "abcde"

	bad := []string{
		// TODO: accent folding? unicode to lower?
		"abcdé",
	}
	good := []string{
		"ABCDE",
		"-A-B-C-D-E-",
	}
	for _, s := range bad {
		if containsBytesFuzzyInsensitive(s, query) {
			t.Error(s + " must not contain " + query + " (returned true)")
		}
	}
	for _, s := range good {
		if !containsBytesFuzzyInsensitive(s, query) {
			t.Error(s + " must contain " + query + " (returned false)")
		}
	}

	// Known false positive: this is a *byte* match, not a string match
	nbsp := "\u00a0"            // UTF-8: nbsp (c2 a0)
	poundSDot := "\u00a3\u2260" // UTF-8: pound (c2 a3) NOT EQUAL TO (e2 89 a0)
	if !containsBytesFuzzyInsensitive(poundSDot, nbsp) {
		t.Error("expected match (bytes match, even though runes do not)")
	}
}

func TestToLower(t *testing.T) {
	if asciiToLower('@') != '@' {
		t.Error("error")
	}
	if asciiToLower('A') != 'a' {
		t.Error("error")
	}
	if asciiToLower('Z') != 'z' {
		t.Error("error")
	}
	if asciiToLower('[') != '[' {
		t.Error("error")
	}
	for i := 0; i < 256; i++ {
		if asciiToLower(byte(i)) != asciiToLowerSlow(byte(i)) {
			t.Error("Wrong output for byte:", i)
		}
	}
}

const query = "abcde"
const s = "01234567890123456789a01234567890123456789b01234567890123456789c01234567890123456789d01234567890123456789e"

func containsBytesFuzzy(s string, query string) bool {
	qIndex := 0
	for i := 0; i < len(s); i++ {
		if s[i] == query[qIndex] {
			qIndex += 1
			if qIndex == len(query) {
				return true
			}
		}
	}
	return false
}

func BenchmarkContainsBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		containsBytesFuzzy(s, query)
	}
}

func containsBytesFuzzyInsensitiveToLower(s string, query string) bool {
	sLower := strings.ToLower(s)
	qLower := strings.ToLower(query)
	return containsBytesFuzzy(sLower, qLower)
}

func BenchmarkContainsBytesInsensitiveToLower(b *testing.B) {
	for i := 0; i < b.N; i++ {
		containsBytesFuzzyInsensitiveToLower(s, query)
	}
}

func BenchmarkContainsBytesInsensitive(b *testing.B) {
	for i := 0; i < b.N; i++ {
		containsBytesFuzzyInsensitive(s, query)
	}
}

func BenchmarkAsciiToLower(b *testing.B) {
	for i := 0; i < b.N; i++ {
		asciiToLower('a')
	}
}

func asciiToLowerSlow(b byte) byte {
	const diff = 'a' - 'A'
	if 'A' <= b && b <= 'Z' {
		b += diff
	}
	return b
}

func BenchmarkAsciiToLowerSlow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		asciiToLowerSlow('a')
	}
}
