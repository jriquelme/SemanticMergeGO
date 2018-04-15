package smgo_test

import (
	"strings"
	"testing"

	"github.com/jriquelme/SemanticMergeGO/smgo"
	"github.com/stretchr/testify/assert"
)

func TestParseErrUnsupportedEncoding(t *testing.T) {
	t.Parallel()
	if testing.Verbose() {
		smgo.PrintBlocks = true
	}

	file, err := smgo.Parse(strings.NewReader("package main\n"), "ISO 8859-1")
	assert.Nil(t, file)
	assert.Equal(t, smgo.ErrUnsupportedEncoding, err)
}
