package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/evanj/csearch/codesearch/index"
	"github.com/evanj/csearch/grep"
	"github.com/evanj/csearch/reindex"
)

const indexPath = "csearch_index"
const staticPath = "static"

const maxFileMatches = 200

type csearchServer struct {
	ix          *index.Index
	fileMatcher *grep.IndexedMatcher
	stripPrefix string
}

const formPage = `<html>
<head><title>codesearch</title>
<script>
var attachTypeahead = function() {
	var typeaheadInput = document.getElementById('typeahead_in');
	var typeaheadOutput = document.getElementById('typeahead_out')
	var generation = 0;

	var typeaheadError = function(e) {
		console.log('typeahead error:', e);
	}

	var onInput = function(e) {
		console.log('typeahead!', e, typeaheadInput.value);
		if (typeaheadInput.value == '') {
			typeaheadOutput.innerHTML = '';
			return;
		}

		// attempt to guard against fast typing/slow queries
		generation += 1;
		var g = generation;
		var typeaheadSuccess = function(result) {
			if (g != generation) {
				console.log('ignored typeahead:', g, generation);
				return;
			}

			typeaheadOutput.innerHTML = result;
		}
		ajax('/type?q=' + typeaheadInput.value, typeaheadSuccess, typeaheadError);
	}
	typeaheadInput.addEventListener('input', onInput);
}

var ajax = function(path, onSuccess, onError) {
	var request = new XMLHttpRequest();
	request.open('GET', path, true);

	request.onload = function() {
		if (request.status == 200){
			onSuccess(request.responseText);
		} else {
			onError(new Error('unexpected status: ' + request.status));
		}
	};

	request.onerror = onError;
	request.send();
}

window.addEventListener('load', attachTypeahead);
</script>
</head>
<body>
<form action="/search" method="GET">
Query: <input type="text" name="q" autofocus> file filter: <input type="text" name="f"> <input type="submit" value="Search">
</form>

<form>
File name live: <input id="typeahead_in" type="text" name="q" width="50">
<div id="typeahead_out"></div>
</form>
</body></html>`

type formattedResult struct {
	*grep.Match
	TruncatedPath string
}

func (f *formattedResult) HTMLLine() template.HTML {
	beforeMatch := template.HTMLEscapeString(f.Line[:f.Start])
	matched := template.HTMLEscapeString(f.Line[f.Start:f.End])
	afterMatch := template.HTMLEscapeString(f.Line[f.End:])

	return template.HTML(beforeMatch + `<span class="m">` + matched + `</span>` + afterMatch)
}

const resultsTemplateString = `<html>
<head><title>results</title>

<style type="text/css">
.results {
  font-family: Consolas, Courier, monospace;
}

.m {
  font-weight: bold;
}
</style>
</head>
<body>

<table>
{{range .}}
<tr><td><a href="/open?path={{.Path}}&linenum={{.LineNumber}}">{{.TruncatedPath}}:{{.LineNumber}}</a></td><td class="results"><code>{{.HTMLLine}}</code></td></tr>
{{end}}
</table>
</body></html>`

var resultsTemplate = template.Must(template.New("results").Parse(resultsTemplateString))

func (server *csearchServer) handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Write([]byte(formPage))
}

func (server *csearchServer) typeaheadHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	q := r.Form.Get("q")
	if q == "" {
		// 200 OK: Empty body (no results)
		return
	}

	// search for matching files!
	results := server.fileMatcher.Match(q, maxFileMatches)
	for _, path := range results {
		w.Write([]byte("<div>"))
		template.HTMLEscape(w, []byte(path))
		w.Write([]byte("</div>"))
	}
	end := time.Now()
	log.Printf("typeahead query len: %d; paths: %d; limited matches: %d; %f seconds",
		len(q), server.ix.NumNames(), len(results), end.Sub(start).Seconds())
}

func (server *csearchServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	results, err := reindex.Search(server.ix, r.Form.Get("q"), r.Form.Get("f"))
	if err != nil {
		panic(err)
	}

	formattedResults := make([]*formattedResult, len(results))
	for i, r := range results {
		formattedPath := r.Path
		if strings.HasPrefix(formattedPath, server.stripPrefix) {
			formattedPath = formattedPath[len(server.stripPrefix):]
		}
		formattedResults[i] = &formattedResult{r, formattedPath}
	}
	err = resultsTemplate.Execute(w, formattedResults)
	if err != nil {
		panic(err)
	}
}

func (server *csearchServer) openHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	path := r.FormValue("path")
	if _, err = os.Stat(path); err != nil {
		panic(path + " does not exist? " + err.Error())
	}
	lineNumberString := r.FormValue("linenum")
	_, err = strconv.Atoi(lineNumberString)
	if err != nil {
		panic(err)
	}

	argument := path + ":" + lineNumberString
	fmt.Println("running subl " + argument)
	c := exec.Command("subl", argument)
	err = c.Run()
	if err != nil {
		panic(err)
	}
	w.Write([]byte("OK!"))
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

func main() {
	skipIndexing := flag.Bool("skipIndexing", false, "do not index the source trees (uses existing index)")
	port := flag.Int("port", 8080, "HTTP listening port")
	stripPrefix := flag.String("stripPrefix", "", "Prefix to remove when displaying results")
	skipPathsFlag := flag.String("skipPaths", "", "Subpaths to not index separated by :")

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: csearch (source tree) [source tree*]")
		flag.Usage()
		os.Exit(1)
	}
	sourcePaths := flag.Args()

	skipPathSet := map[string]struct{}{}
	for _, v := range strings.Split(*skipPathsFlag, ":") {
		skipPathSet[v] = struct{}{}
	}
	shouldIndex := func(path string, info os.FileInfo) bool {
		if _, contained := skipPathSet[path]; contained {
			fmt.Println("!!! WTF SKIPPING", path)
			return false
		}
		return true
	}

	var ix *index.Index
	if !*skipIndexing {
		fmt.Printf("Indexing %s ...\n", strings.Join(sourcePaths, ", "))
		start := time.Now()
		writer, err := reindex.Create(indexPath)
		if err != nil {
			panic(err)
		}
		for _, path := range sourcePaths {
			err = reindex.IndexTree(writer, path, shouldIndex)
			if err != nil {
				panic(err)
			}
		}
		ix = reindex.FlushAndReopen(writer, indexPath)
		writer = nil
		end := time.Now()
		fmt.Printf("Done (%f seconds)\n", end.Sub(start).Seconds())
	} else {
		ix = index.Open(indexPath)
	}

	indexedMatcher := grep.IndexedMatcher{}
	for i := 0; i < ix.NumNames(); i++ {
		path := ix.Name(uint32(i))
		indexedMatcher.Add(path)
	}
	server := csearchServer{ix, &indexedMatcher, *stripPrefix}

	http.HandleFunc("/favicon.ico", favicon)
	const staticPrefix = "/static/"
	staticHandler := http.StripPrefix(staticPrefix, http.FileServer(http.Dir(staticPath)))
	http.Handle("/static", http.NotFoundHandler())
	http.Handle(staticPrefix, staticHandler)
	http.Handle("/", http.HandlerFunc(server.handler))
	http.Handle("/search", http.HandlerFunc(server.searchHandler))
	http.Handle("/type", http.HandlerFunc(server.typeaheadHandler))
	http.Handle("/open", http.HandlerFunc(server.openHandler))

	portString := "localhost:" + strconv.Itoa(*port)
	fmt.Printf("Listening on http://%s/\n", portString)
	err := http.ListenAndServe(portString, logRequests(http.DefaultServeMux))
	if err != nil {
		panic(err)
	}
}
