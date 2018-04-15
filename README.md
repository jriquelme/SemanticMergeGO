# GO external parser for SemanticMerge [![Build Status](https://travis-ci.org/jriquelme/SemanticMergeGO.svg?branch=master)](https://travis-ci.org/jriquelme/SemanticMergeGO) [![Go Report Card](https://goreportcard.com/badge/github.com/jriquelme/SemanticMergeGO)](https://goreportcard.com/report/github.com/jriquelme/SemanticMergeGO) [![Coverage Status](https://coveralls.io/repos/github/jriquelme/SemanticMergeGO/badge.svg?branch=master)](https://coveralls.io/github/jriquelme/SemanticMergeGO?branch=master)

Check the [SemanticMerge Documentation](https://users.semanticmerge.com/documentation/external-parsers/external-parsers-guide.shtml)
for a detailed description.

**Work in progress.**

## Development notes

The package smgo-cli has some integration tests. Those tests run against the binary in `$GOPATH/bin/smgo-cli`; therefore
the package has to be installed before executing the tests. Additionally, the build tag *itest* is used to run the
integration tests explicitly:

```bash
$ go install ./...
$ go test -tags="itest" -v ./smgo-cli
```
