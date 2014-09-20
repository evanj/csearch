// Package grep contains utilities to mimic grep on files
package grep

import (
	"bufio"
	"os"
	"regexp"
)

type Match struct {
	Path       string
	LineNumber int
	Line       string
	Start      int
	End        int
}

func Grep(re *regexp.Regexp, path string) ([]*Match, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	line := 1
	scanner := bufio.NewScanner(f)
	var matches []*Match
	for scanner.Scan() {
		r := re.FindIndex(scanner.Bytes())
		if r != nil {
			match := &Match{path, line, string(scanner.Bytes()), r[0], r[1]}
			matches = append(matches, match)
		}
		line += 1
	}
	return matches, scanner.Err()
}
