package grep

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	cregexp "github.com/google/codesearch/regexp"
)

type grepper interface {
	setup(query string, output io.Writer) error
	grep(path string) error
}

type csearchGrep struct {
	g *cregexp.Grep
}

func (c *csearchGrep) setup(query string, output io.Writer) error {
	re, err := cregexp.Compile(query)
	if err != nil {
		return err
	}
	c.g = &cregexp.Grep{}
	c.g.Regexp = re
	c.g.Stdout = output
	c.g.N = true

	// HACK to initialize the internal buffer
	f, err := ioutil.TempFile("", "grep_test")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	f.Write([]byte("data"))
	c.g.File(f.Name())

	return nil
}

func (c *csearchGrep) grep(path string) error {
	c.g.File(path)
	return nil
}

type scannerGrep struct {
	q      *regexp.Regexp
	output io.Writer
}

func (s *scannerGrep) setup(query string, output io.Writer) error {
	s.output = output
	var err error
	s.q, err = regexp.Compile(query)
	return err
}

func (s *scannerGrep) grep(path string) error {
	matches, err := Grep(s.q, path)
	if err != nil {
		return err
	}

	for _, match := range matches {
		fmt.Fprintf(s.output, "%s:%d: %s\n", match.Path, match.LineNumber, match.Line)
	}
	return nil
}

func makeFile(f *os.File, lineLength int, lines int) error {
	const tenChars = "0123456789"
	line := ""
	for len(line) < lineLength {
		line += tenChars
	}
	line += "\n"
	b := []byte(line)

	for i := 0; i < lines; i++ {
		_, err := f.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func benchmark(b *testing.B, query string, lineLength int, lines int, grep grepper) {
	f, err := ioutil.TempFile("", "grep_test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	err = makeFile(f, lineLength, lines)
	if err != nil {
		b.Fatal(err)
	}

	err = grep.setup(query, ioutil.Discard)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err = grep.grep(f.Name())
		if err != nil {
			b.Fatal(err)
		}
	}
}

const lineLength = 80
const tinyLines = 100
const smallLines = 1000
const largeLines = 70000 // ~5.5 MB file

const match = "90[0-9].*90"
const noMatch = "5notmatch"

func BenchmarkCsearchTinyMatch(b *testing.B) {
	benchmark(b, match, lineLength, tinyLines, &csearchGrep{})
}

func BenchmarkCsearchSmall(b *testing.B) {
	benchmark(b, noMatch, lineLength, smallLines, &csearchGrep{})
}

func BenchmarkCsearchLarge(b *testing.B) {
	benchmark(b, noMatch, lineLength, largeLines, &csearchGrep{})
}

func BenchmarkScannerTinyMatch(b *testing.B) {
	benchmark(b, match, lineLength, tinyLines, &csearchGrep{})
}

func BenchmarkScannerSmall(b *testing.B) {
	benchmark(b, noMatch, lineLength, smallLines, &csearchGrep{})
}

func BenchmarkScannerLarge(b *testing.B) {
	benchmark(b, noMatch, lineLength, largeLines, &csearchGrep{})
}
