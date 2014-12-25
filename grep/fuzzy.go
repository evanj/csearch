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
func fuzzyMatchPathAndFile(filepath string, filename string, query string) int {
	// this filter is "incorrect":
	// * false positive: matches BYTES, so it can incorrectly match UTF-8 byte parts. (see unit test)
	// * false negative: doesn't lowercase non-ASCII
	if !containsBytesFuzzyInsensitive(filepath, query) {
		return -1
	}

	score := fuzzyMatchFile(filename, query)
	if score >= 0 {
		return score
	}

	// path fuzzy match
	return scoreFuzzyStrings(filepath, query)
}

func fuzzyMatchFile(filename string, query string) int {
	// file name prefix and substring match
	// do a fast test before slow scoring (minor performance win)
	if !containsBytesFuzzyInsensitive(filename, query) {
		return -1
	}

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
	score := scoreFuzzyStrings(filename, query)
	if score >= 0 {
		return score + fileNameMatchScore
	}
	return -1
}

func FuzzyMatchPath(filepath string, query string) int {
	filename := path.Base(filepath)
	return fuzzyMatchPathAndFile(filepath, filename, query)
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

func (matcher *FuzzyMatcher) addResult(filepath string, score int) {
	assert(score >= 0)
	matcher.totalMatches += 1
	if matcher.Limit <= 0 || len(matcher.results) < matcher.Limit {
		match := &fuzzyMatch{score, filepath}
		matcher.results = append(matcher.results, match)
	} else {
		assert(matcher.Limit > 0 && len(matcher.results) == matcher.Limit)
		// TODO: Make this faster? This is a naive n**2 algorithm
		for _, r := range matcher.results {
			if r.score < score {
				r.score = score
				r.value = filepath
				break
			}
		}
	}
}

func (matcher *FuzzyMatcher) matchFile(filepath string, filename string) bool {
	score := fuzzyMatchFile(filename, matcher.Query)
	if score >= 0 {
		matcher.addResult(filepath, score)
		return true
	}
	return false
}

func (matcher *FuzzyMatcher) matchPathAndFile(filepath string, filename string) {
	score := fuzzyMatchPathAndFile(filepath, filename, matcher.Query)
	if score >= 0 {
		matcher.addResult(filepath, score)
	}
}

func (matcher *FuzzyMatcher) Match(filepath string) {
	filename := path.Base(filepath)
	matcher.matchPathAndFile(filepath, filename)
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

type indexedPath struct {
	filepath string
	filename string
}

type IndexedMatcher struct {
	// TODO: Use two separate arrays for slightly better cache locality?
	// TODO: ~35% of file paths are lowercase only (slightly more file names); could
	// avoid calling .ToLower for paths that are known lowercase?
	paths []*indexedPath
}

func (matcher *IndexedMatcher) Add(filepath string) {
	filename := path.Base(filepath)
	indexed := indexedPath{filepath, filename}
	matcher.paths = append(matcher.paths, &indexed)
}

func (matcher *IndexedMatcher) Match(query string, limit int) []string {
	// match file names first, then add results with patch matches
	fuzzy := FuzzyMatcher{Query: query, Limit: limit}
	matched := map[int]struct{}{}
	for i, indexed := range matcher.paths {
		if fuzzy.matchFile(indexed.filepath, indexed.filename) {
			matched[i] = struct{}{}
		}
	}
	if fuzzy.Limit > 0 && len(fuzzy.results) == fuzzy.Limit {
		// found a full set of filename matches: we are done
		return fuzzy.Results()
	}

	for i, indexed := range matcher.paths {
		if _, ok := matched[i]; ok {
			// already matched
			continue
		}
		fuzzy.matchPathAndFile(indexed.filepath, indexed.filename)
		if fuzzy.Limit > 0 && len(fuzzy.results) == fuzzy.Limit {
			// don't care about finding the "best" path matches: this is probably enough
			break
		}
	}

	return fuzzy.Results()
}
