package reindex

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"time"

	"code.google.com/p/codesearch/index"
	"github.com/evanj/csearch/grep"
)

const minQueryLength = 3

func IndexTree(tree string, indexPath string) (*index.Index, error) {
	err := os.Remove(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ix := index.Create(indexPath)
	ix.AddPaths([]string{tree})

	err = filepath.Walk(tree, func(path string, info os.FileInfo, err error) error {
		if _, elem := filepath.Split(path); elem != "" {
			// Skip various temporary or "hidden" files or directories.
			if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if err != nil {
			log.Printf("%s: %s", path, err)
			return nil
		}
		if info != nil && info.Mode()&os.ModeType == 0 {
			ix.AddFile(path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	ix.Flush()

	return index.Open(indexPath), nil
}

func Search(ix *index.Index, qString string, fileRegexp string) ([]*grep.Match, error) {
	start := time.Now()

	if len(qString) < minQueryLength {
		return nil, errors.New("query string too short")
	}
	qSyntax, err := syntax.Parse(qString, syntax.Perl)
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(qString)
	if err != nil {
		return nil, err
	}
	indexQuery := index.RegexpQuery(qSyntax)
	fileRe, err := regexp.Compile(fileRegexp)
	if err != nil {
		return nil, err
	}

	postingList := ix.PostingQuery(indexQuery)
	postingTime := time.Now()
	log.Printf("%d posting list matches", len(postingList))

	realMatches := 0
	fileMatches := 0
	var results []*grep.Match
	for _, fileId := range postingList {
		name := ix.Name(fileId)
		if !fileRe.MatchString(name) {
			continue
		}
		fileMatches += 1

		matches, err := grep.Grep(re, name)
		if err != nil {
			return nil, err
		}
		if len(matches) > 0 {
			realMatches += 1
		}
		results = append(results, matches...)
	}
	grepTime := time.Now()
	log.Printf("posting matches: %d; file matches: %d; real matches: %d (false positives: %d)",
		len(postingList), fileMatches, realMatches, fileMatches-realMatches)
	log.Printf("posting time: %f grep time: %f",
		postingTime.Sub(start).Seconds(), grepTime.Sub(postingTime).Seconds())
	return results, nil
}
