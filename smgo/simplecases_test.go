package smgo_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jriquelme/SemanticMergeGO/smgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestParseEmpty(t *testing.T) {
	t.Parallel()

	src := bytes.NewReader([]byte{})
	file, err := smgo.Parse(src, "UTF-8")
	assert.NotNil(t, file)
	assert.Nil(t, err)

	assert.Equal(t, &smgo.File{
		LocationSpan: newLocationSpan(1, 0, 1, 0),
		FooterSpan:   smgo.RuneSpan{0, -1},
		Containers:   nil,
		Nodes:        nil,
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

func TestParseSimpleFunc(t *testing.T) {
	t.Parallel()
	simpleMain, err := os.Open("testdata/simple_func.go")
	require.Nil(t, err)
	defer simpleMain.Close()

	file, err := smgo.Parse(simpleMain, "UTF-8")
	assert.NotNil(t, file)
	assert.Nil(t, err)

	assert.Equal(t, &smgo.File{
		LocationSpan: newLocationSpan(1, 0, 5, 2),
		FooterSpan:   smgo.RuneSpan{0, -1},
		Containers:   nil,
		Nodes: []*smgo.Node{
			{
				Type:         smgo.PackageNode,
				Name:         "simplefunc",
				LocationSpan: newLocationSpan(1, 0, 1, 19),
				Span:         smgo.RuneSpan{0, 18},
			},
			{
				Type:         smgo.FunctionNode,
				Name:         "Hi",
				LocationSpan: newLocationSpan(2, 0, 5, 2),
				Span:         smgo.RuneSpan{19, 47},
			},
		},
		ParsingErrors: nil,
	}, file)
	if t.Failed() {
		spew.Dump(t.Name(), file)
	}
}
