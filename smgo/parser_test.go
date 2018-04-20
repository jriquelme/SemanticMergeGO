package smgo_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jriquelme/SemanticMergeGO/smgo"
	"github.com/stretchr/testify/assert"
)

func newLocationSpan(startLine, startColumn, endLine, endColumn int) smgo.LocationSpan {
	return smgo.LocationSpan{
		Start: smgo.Location{
			Line:   startLine,
			Column: startColumn,
		},
		End: smgo.Location{
			Line:   endLine,
			Column: endColumn,
		},
	}
}

func TestParseErrUnsupportedEncoding(t *testing.T) {
	t.Parallel()
	if testing.Verbose() {
		smgo.PrintBlocks = true
	}

	file, err := smgo.Parse(strings.NewReader("package main\n"), "ISO 8859-1")
	assert.Nil(t, file)
	assert.Equal(t, smgo.ErrUnsupportedEncoding, err)
}

func TestParseEmpty(t *testing.T) {
	t.Parallel()
	if testing.Verbose() {
		smgo.PrintBlocks = true
	}

	src := bytes.NewReader([]byte{})
	file, err := smgo.Parse(src, "UTF-8")
	assert.NotNil(t, file)
	assert.Nil(t, err)

	assert.Equal(t, &smgo.File{
		LocationSpan: newLocationSpan(1, 0, 1, 0),
		FooterSpan:   smgo.RuneSpan{0, -1},
		Children:     nil,
		ParsingErrors: []*smgo.ParsingError{
			{
				Location: smgo.Location{1, 0},
				Message:  "1:1: expected 'package', found 'EOF'",
			},
		},
	}, file)
	if t.Failed() {
		spew.Dump(t.Name(), file)
	}
}
