# csearch: A Code Search experiment

I was inspired by Russ Cox's [Regular Expression index article](https://swtch.com/~rsc/regexp/regexp4.html) and by Sublime Text's "find anything" feature to build a code search web tool. Ås with most things this is an incomplete proof of concept more than anything.


# Usage

* *Install*: `go get github.com/evanj/csearch`
* *Run*: `csearch (path to search)` e.g. `csearch $GOPATH/src`

In the "Query" box, type a regexp and click search. The results are ugly, sorry.

In the "file name live" box, start typing. It will display a "live" list of results. This is both ugly and the results are not high quality.


# codesearch fork

I've forked codesearch into this repository only to be able to read the file names from the index file. This is a bit of overkill but it works.