package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"strconv"
	"time"

	"code.google.com/p/codesearch/index"
	// csregexp "code.google.com/p/codesearch/regexp"

	"./grep"
)

const repositoryPath = "/Users/ej/gpath/src"
const indexPath = "csearch_index"
const staticPath = "static"

const minQueryLength = 3

type csearchServer struct {
	ix *index.Index
}

const formPage = `<html>
<head><title>page</title>
</head>
<body>
<form action="/search" method="GET">
Query: <input type="text" name="q">
</form>
</body></html>`

const resultsTemplateString = `<html>
<head><title>results</title></head>
<body>
{{range .}}{{.}}<br>{{end}}
</body></html>`

var resultsTemplate = template.Must(template.New("results").Parse(resultsTemplateString))

func (server *csearchServer) handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(formPage))
}

func search(ix *index.Index, qString string) ([]*grep.Match, error) {
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

	start := time.Now()
	postingList := ix.PostingQuery(indexQuery)
	postingTime := time.Now()
	log.Printf("%d posting list matches", len(postingList))

	realMatches := 0
	var results []*grep.Match
	for _, fileId := range postingList {
		name := ix.Name(fileId)
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
	log.Printf("posting matches: %d; real matches: %d (false positives: %d)",
		len(postingList), realMatches, len(postingList)-realMatches)
	log.Printf("posting time: %f grep time: %f",
		postingTime.Sub(start).Seconds(), grepTime.Sub(postingTime).Seconds())
	return results, nil
}

func (server *csearchServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	results, err := search(server.ix, r.Form.Get("q"))
	if err != nil {
		panic(err)
	}
	err = resultsTemplate.Execute(w, results)
	if err != nil {
		panic(err)
	}
}

func favicon(w http.ResponseWriter, r *http.Request) {
	// TODO: Add favicon
	const cacheSeconds = 60 * 60
	w.Header().Add("Cache-Control", "max-age="+strconv.Itoa(cacheSeconds))
	http.Error(w, "not found", http.StatusNotFound)
}

func logRequests(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func indexTree(tree string, indexPath string) (*index.Index, error) {
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

func main() {
	fmt.Printf("Indexing %s ...\n", repositoryPath)
	start := time.Now()
	ix, err := indexTree(repositoryPath, indexPath)
	if err != nil {
		panic(err)
	}
	end := time.Now()
	fmt.Printf("Done (%f seconds)\n", end.Sub(start).Seconds())

	server := csearchServer{ix}

	http.HandleFunc("/favicon.ico", favicon)
	const staticPrefix = "/static/"
	staticHandler := http.StripPrefix(staticPrefix, http.FileServer(http.Dir(staticPath)))
	http.Handle("/static", http.NotFoundHandler())
	http.Handle(staticPrefix, staticHandler)
	http.Handle("/", http.HandlerFunc(server.handler))
	http.Handle("/search", http.HandlerFunc(server.searchHandler))

	fmt.Println("Listening on http://localhost:8080/")
	err = http.ListenAndServe(":8080", logRequests(http.DefaultServeMux))
	if err != nil {
		panic(err)
	}
}
