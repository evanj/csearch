package main

import (
	"fmt"
	"os"
	"regexp"

	"../../grep"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "ggrep (query) (path)\n")
		os.Exit(1)
	}
	query := os.Args[1]
	path := os.Args[2]

	q, err := regexp.Compile(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compiling expression '%s': %s", query, err.Error())
		os.Exit(1)
	}

	matches, err := grep.Grep(q, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
	for _, match := range matches {
		l := match.Line[:match.Start] + "#" + match.Line[match.Start:match.End] + "#" + match.Line[match.End:]
		fmt.Printf("%s:%d: %s\n", match.Path, match.LineNumber, l)
	}
}
