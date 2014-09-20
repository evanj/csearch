package grep

import (
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
	// if the are equal, we actually execute the significantly slower asciiToLower()
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

func isWordEnd(previous rune, next rune) bool {
	if !unicode.IsLetter(previous) {
		return true
	}
	if unicode.IsLower(previous) && !unicode.IsLower(next) {
		return true
	}
	return false
}

// Returns a ranking of how well query matches path. Values less than zero do not match at all.
// General order: Filename prefix match (case insensitive)
func FuzzyMatch(query string, path string) int {
	qRunes := toRunes(query)
	pRunes := toRunes(path)

	lastPathIndex := -1
	for i, c := range pRunes {
		if c == '/' {
			lastPathIndex = i
		}
	}

	score := 0
	qIndex := 0
	for i, pRune := range pRunes {
		qRuneLower := unicode.ToLower(qRunes[qIndex])
		pRuneLower := unicode.ToLower(pRune)
		if qRuneLower == pRuneLower {
			// if this rune is the beginning of a word it scores higher
			if i == 0 || isWordEnd(pRunes[i-1], pRune) {
				score += 1
			}
			if qIndex == 0 {
				if i == lastPathIndex+1 {
					// first match is the beginning of the filename: huge boost
					score += 100
				} else if i > lastPathIndex {
					// first match is in the filename: bost
					score += 50
				}
			}
			qIndex += 1
			if qIndex == len(qRunes) {
				break
			}
		}
	}
	if qIndex != len(qRunes) {
		return -1
	}
	return score
}
