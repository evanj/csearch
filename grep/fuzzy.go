package grep

import (
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

/* BRANCHLESS MAGIC: nearly 2X faster than the naive approach

HOW IT WORKS:

The first term is negative if c >= A
The second term is negative if c <= Z
ANDing them together: negative only if both are negative
SHIFT/AND: equals 32 if both terms are negative, zero otherwise

The shift can be anywhere from 26 (max) to 2
*/

func asciiToLower(b byte) byte {
	c := uint32(b)
	c += (((('A' - 1) - c) & (c - ('Z' + 1))) >> 26) & 32
	return byte(c)
}

// Returns true if s contains all the bytes in query in order, with any number of additional bytes
// between them. It matches ASCII in a case-insensitive way.
func containsBytesFuzzyInsensitive(s string, query string) bool {
	// this is ridiculously over-optimized
	// We unconditionally set the magicLowerBit for an approximate case-insensitive comparison
	// if they are equal, we execute asciiToLower(), which is signficantly slower
	// this is about ~25% faster than always calling asciiToLower
	const magicLowerBit = 0x20
	qIndex := 0
	qByteMaybeLower := query[qIndex] | magicLowerBit
	qByteLower := asciiToLower(query[qIndex])
	for i := 0; i < len(s); i++ {
		sByteMaybeLower := s[i] | magicLowerBit
		if sByteMaybeLower == qByteMaybeLower {
			// now ACTUALLY check lowercase
			if asciiToLower(s[i]) == qByteLower {
				qIndex += 1
				if qIndex == len(query) {
					return true
				}
				qByteMaybeLower = query[qIndex] | magicLowerBit
				qByteLower = asciiToLower(query[qIndex])
			}
		}
	}
	return false
}

func toRunes(s string) []rune {
	if !utf8.ValidString(s) {
		panic("invalid utf8 input")
	}
	l := utf8.RuneCountInString(s)
	runes := make([]rune, 0, l)
	for _, r := range s {
		runes = append(runes, r)
	}
	return runes
}

// A word begins when the previous rune is a non-letter (_example), or if there is a lowercase
// to uppercase transition (newWord). Uppercase does not count (Word) (HTTP)
func isWordStart(previous rune, current rune) bool {
	if !unicode.IsLetter(current) {
		return false
	}
	if !unicode.IsLetter(previous) {
		return true
	}
	if unicode.IsLower(previous) && !unicode.IsLower(current) {
		return true
	}
	return false
}

// pass into isWordStart to detect a word start at the beginning of a string
const wordStartInitialRune = '-'

// general order of file path scores
const fileNamePrefixScore = 500
const fileNameWordPrefixScore = 400
const fileNameSubstringScore = 300
const fileNameMatchScore = 100
const caseMatchBonus = 10

const maxLengthPenalty = 80

// Returns a ranking of how well query matches path. Values less than zero do not match at all.
// General order of scores:
// * Filename prefix match (with case match bonus)
// * Filename substring match (with entire word, word prefix, case match bonuses)
// * Path substring match (same bonuses)
// * Filename fuzzy match (all chars in filename)
// * path + filename match
func FuzzyMatch(filepath string, query string) int {
	// this filter is "incorrect":
	// * false positive: matches BYTES, so it can incorrectly match UTF-8 byte parts. (see unit test)
	// * false negative: doesn't lowercase non-ASCII
	if !containsBytesFuzzyInsensitive(filepath, query) {
		return -1
	}

	// file name prefix and substring match
	filename := path.Base(filepath)
	index := strings.Index(strings.ToLower(filename), strings.ToLower(query))
	if index >= 0 {
		lengthPenalty := len(filename) - len(query)
		if lengthPenalty > maxLengthPenalty {
			lengthPenalty = maxLengthPenalty
		}
		caseScore := 0
		if strings.HasPrefix(filename[index:], query) {
			caseScore = caseMatchBonus
		}
		if index == 0 {
			return fileNamePrefixScore + caseScore - lengthPenalty
		}
		prevRune, _ := utf8.DecodeLastRuneInString(filename[:index])
		firstMatchRune, _ := utf8.DecodeRuneInString(filename[index:])
		if prevRune == utf8.RuneError || firstMatchRune == utf8.RuneError {
			panic("unexpected rune error: invalid UTF-8 filename?")
		}
		if isWordStart(prevRune, firstMatchRune) {
			return fileNameWordPrefixScore + caseScore - lengthPenalty
		}
		return fileNameSubstringScore + caseScore - lengthPenalty
	}

	// file name fuzzy match
	if containsBytesFuzzyInsensitive(filename, query) {
		score := scoreFuzzyStrings(filename, query)
		if score >= 0 {
			return score + fileNameMatchScore
		}
	}

	// path fuzzy match
	return scoreFuzzyStrings(filepath, query)
}

// Returns the score for string matches, ignoring path-specific information
func scoreFuzzyStrings(s string, query string) int {
	qRunes := toRunes(query)

	score := 0
	qIndex := 0
	previousRune := wordStartInitialRune
	for _, sRune := range s {
		qRuneLower := unicode.ToLower(qRunes[qIndex])
		sRuneLower := unicode.ToLower(sRune)
		if qRuneLower == sRuneLower {
			// if this rune is the beginning of a word it scores higher
			if isWordStart(previousRune, sRune) {
				score += 1
			}
			qIndex += 1
			if qIndex == len(qRunes) {
				break
			}
		}
		previousRune = sRune
	}
	if qIndex != len(qRunes) {
		return -1
	}
	return score
}

type fuzzyMatch struct {
	score int
	value string
}

type fuzzyMatches []*fuzzyMatch

func (a fuzzyMatches) Len() int           { return len(a) }
func (a fuzzyMatches) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a fuzzyMatches) Less(i, j int) bool { return a[i].score > a[j].score }

type FuzzyMatcher struct {
	Query        string
	Limit        int
	results      []*fuzzyMatch
	totalMatches int
}

func assert(v bool) {
	if !v {
		panic("assertion failed")
	}
}

func (matcher *FuzzyMatcher) Match(value string) {
	score := FuzzyMatch(value, matcher.Query)
	if score < 0 {
		// no match
		return
	}

	matcher.totalMatches += 1
	if matcher.Limit <= 0 || len(matcher.results) < matcher.Limit {
		match := &fuzzyMatch{score, value}
		matcher.results = append(matcher.results, match)
	} else {
		assert(matcher.Limit > 0 && len(matcher.results) == matcher.Limit)
		// TODO: Make this faster? This is a naive n**2 algorithm
		for _, r := range matcher.results {
			if r.score < score {
				r.score = score
				r.value = value
				break
			}
		}
	}
}

func (matcher *FuzzyMatcher) Results() []string {
	sort.Sort(fuzzyMatches(matcher.results))
	out := make([]string, len(matcher.results))
	for i, match := range matcher.results {
		out[i] = match.value
	}
	return out
}

func (matcher *FuzzyMatcher) TotalMatches() int {
	return matcher.totalMatches
}
