package smgo

import (
	"bytes"
	"errors"
)

type blockType int

//go:generate stringer -type=blockType

const (
	nodeBlock blockType = iota
	containerHeader
	containerFooter
)

type block struct {
	Type      blockType
	Node      *Node
	Container *Container
}

func fixBlockBoundaries(file *File, blocks []block, src []byte) error {
	file.LocationSpan.Start.Column = 0

	var pos int
	switch blocks[0].Type {
	case nodeBlock:
		n := blocks[0].Node
		n.LocationSpan.Start.Column = 0 // FIXME
		pos = n.Span.End
	default:
		return errors.New("case not covered")
	}

	for _, b := range blocks[1:] {
		switch b.Type {
		case nodeBlock:
			n := b.Node
			n.LocationSpan.Start.Column = 0 // FIXME
			b := src[pos+1 : n.Span.Start]
			newLines := bytes.Count(b, []byte{0x0a})
			n.Span.Start = pos + 1
			n.LocationSpan.Start.Line = n.LocationSpan.Start.Line - newLines
			n.LocationSpan.Start.Column = 0
		case containerHeader:
			return errors.New("case not covered")
		case containerFooter:
			return errors.New("case not covered")
		default:
			return errors.New("case not covered")
		}
	}
	return nil
}
