package reindex

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"time"

	"github.com/evanj/csearch/codesearch/index"
	"github.com/evanj/csearch/grep"
)

const minQueryLength = 3

func Create(indexPath string) (*index.IndexWriter, error) {
	err := os.Remove(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ix := index.Create(indexPath)
	return ix, nil
}

func IndexTree(ix *index.IndexWriter, tree string, shouldIndex func(string, os.FileInfo) bool) error {
	ix.AddPaths([]string{tree})

	err := filepath.Walk(tree, func(path string, info os.FileInfo, err error) error {
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
		if !shouldIndex(path, info) {
			// strictly speaking: we should not return this for files, but it works
			return filepath.SkipDir
		}

		if info.Mode()&os.ModeType == 0 {
			ix.AddFile(path)
		}
		return nil
	})
	return err
}

// TODO: get path from writer? Don't include this at all? (it does make imports easier)
func FlushAndReopen(writer *index.IndexWriter, path string) *index.Index {
	writer.Flush()
	return index.Open(path)
}

// Returns matches that match qString and fileRegexp. Ignores files that exist in the
// index but cannot be opened. This usually indicates that the index is out of date.
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
	notFound := 0
	var results []*grep.Match
	for _, fileId := range postingList {
		name := ix.Name(fileId)
		if !fileRe.MatchString(name) {
			continue
		}
		fileMatches += 1

		matches, err := grep.Grep(re, name)
		if err != nil {
			if os.IsNotExist(err) {
				// TODO: Warn when file not found? Requires changing match structure?
				notFound += 1
			} else {
				return nil, err
			}
		}
		if len(matches) > 0 {
			realMatches += 1
		}
		results = append(results, matches...)
	}
	grepTime := time.Now()
	log.Printf("posting matches: %d; file matches: %d; real matches: %d (false positives: %d; not found: %d)",
		len(postingList), fileMatches, realMatches, fileMatches-realMatches-notFound, notFound)
	log.Printf("posting time: %f grep time: %f",
		postingTime.Sub(start).Seconds(), grepTime.Sub(postingTime).Seconds())
	return results, nil
}
