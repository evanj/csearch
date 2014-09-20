package grep

import (
	"unicode"
	"unicode/utf8"
)

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
