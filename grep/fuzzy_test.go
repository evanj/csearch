package grep

import (
	"strings"
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	first := FuzzyMatch("type", "a/b/c/TypeAheadHandler.java")
	secondA := FuzzyMatch("type", "a/b/c/OBJ_TYPE.txt")
	secondB := FuzzyMatch("type", "a/b/c/PlaceType.java")
	third := FuzzyMatch("type", "a/b/type/Foo.java")
	fourth := FuzzyMatch("type", "a/b/ctyped/Foo.java")

	if first <= secondA {
		t.Error("wrong order:", first, secondA)
	}
	if secondA != secondB {
		t.Error("wrong order:", secondA, secondB)
	}
	if secondB <= third {
		t.Error("wrong order:", secondB, third)
	}
	if third <= fourth {
		t.Error("wrong order:", third, fourth)
	}
	if fourth < 0 {
		t.Error("last must match:", fourth)
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
