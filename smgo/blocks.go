package smgo

import (
	"bytes"
	"errors"

	"github.com/davecgh/go-spew/spew"
)

type blockType int

//go:generate stringer -type=blockType

const (
	nodeBlock blockType = iota
	containerHeader
	containerFooter
)

type block struct {
	Type blockType
	Node Node
}

func (b *block) Terminal() *Terminal {
	return b.Node.(*Terminal)
}

func (b *block) Container() *Container {
	return b.Node.(*Container)
}

func fixBlockBoundaries(file *File, blocks []block, src []byte) error {
	file.LocationSpan.Start.Column = 0

	var pos int
	switch blocks[0].Type {
	case nodeBlock:
		n := blocks[0].Terminal()
		n.LocationSpan.Start.Column = 0 // FIXME
		pos = n.Span.End
	default:
		return errors.New("case not covered")
	}

	for _, b := range blocks[1:] {
		switch b.Type {
		case nodeBlock:
			n := b.Terminal()
			n.LocationSpan.Start.Column = 0 // FIXME
			b := src[pos+1 : n.Span.Start]
			newLines := bytes.Count(b, []byte{0x0a})
			n.Span.Start = pos + 1
			n.LocationSpan.Start.Line = n.LocationSpan.Start.Line - newLines
			n.LocationSpan.Start.Column = 0
			pos = n.Span.End
		case containerHeader:
			n := b.Container()
			n.LocationSpan.Start.Column = 0 // FIXME
			b := src[pos+1 : n.HeaderSpan.Start]
			newLines := bytes.Count(b, []byte{0x0a})
			n.HeaderSpan.Start = pos + 1
			n.HeaderSpan.End = n.HeaderSpan.End + 1
			n.LocationSpan.Start.Line = n.LocationSpan.Start.Line - newLines
			n.LocationSpan.Start.Column = 0
			pos = n.HeaderSpan.End
		case containerFooter:
			n := b.Container()
			n.LocationSpan.Start.Column = 0 // FIXME
			b := src[pos+1 : n.FooterSpan.Start]
			newLines := bytes.Count(b, []byte{0x0a})
			n.FooterSpan.Start = pos + 1
			n.FooterSpan.End = n.FooterSpan.End + 1
			n.LocationSpan.Start.Line = n.LocationSpan.Start.Line - newLines
			n.LocationSpan.Start.Column = 0
			pos = n.FooterSpan.End
		default:
			panic("impossibru!")
		}
	}
	return nil
}

type debugBlock struct {
	Name         string
	LocationSpan LocationSpan
	Span         RuneSpan
}

func printBlocks(blocks []block) {
	debugBlocks := make([]debugBlock, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case nodeBlock:
			debugBlocks = append(debugBlocks, debugBlock{
				Name:         b.Terminal().Name,
				LocationSpan: b.Terminal().LocationSpan,
				Span:         b.Terminal().Span,
			})
		case containerHeader:
			debugBlocks = append(debugBlocks, debugBlock{
				Name:         b.Container().Name,
				LocationSpan: b.Container().LocationSpan,
				Span:         b.Container().HeaderSpan,
			})
		case containerFooter:
			debugBlocks = append(debugBlocks, debugBlock{
				Name:         b.Container().Name,
				LocationSpan: b.Container().LocationSpan,
				Span:         b.Container().FooterSpan,
			})
		default:
			panic("impossibru!")
		}
	}
	spew.Printf("=========\n")
	spew.Dump("Debug Blocks", debugBlocks)
	spew.Printf("=========\n")
}
