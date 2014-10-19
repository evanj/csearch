package reindex

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "search_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	tempDir2, err := ioutil.TempDir("", "search_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir2)

	f1Path := filepath.Join(tempDir, "f1")
	err = ioutil.WriteFile(f1Path, []byte("hello world f1\nfoo bar\n"), 0700)
	if err != nil {
		t.Fatal(err)
	}
	f2Path := filepath.Join(tempDir, "f2")
	err = ioutil.WriteFile(f2Path, []byte("hello world f2\nfoo bar\n"), 0700)
	if err != nil {
		t.Fatal(err)
	}
	f3Path := filepath.Join(tempDir2, "f3")
	err = ioutil.WriteFile(f3Path, []byte("hello world f3\nfoo bar\n"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	indexPath := filepath.Join(tempDir, ".index")
	writer, err := Create(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	err = IndexTree(writer, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	err = IndexTree(writer, tempDir2)
	if err != nil {
		t.Fatal(err)
	}
	writer.Flush()

	index := FlushAndReopen(writer, indexPath)

	results, err := Search(index, " f1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Error("bad results", results)
	}
	if !strings.HasSuffix(results[0].Path, "/f1") {
		t.Error(*results[0])
	}
	if results[0].LineNumber != 1 {
		t.Error(*results[0])
	}
	if results[0].Line[results[0].Start:results[0].End] != " f1" {
		t.Error(*results[0])
	}

	results, err = Search(index, "foo", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Error(results)
	}
	results, err = Search(index, "foo", "f1$")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Error(results)
	}
}
