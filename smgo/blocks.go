package smgo

import (
	"go/token"

	"github.com/davecgh/go-spew/spew"
)

var PrintBlocks bool

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

func addBlocksFrom(parent parentNode, blocks *[]block) {
	for _, child := range parent.Nodes() {
		switch n := child.(type) {
		case *Container:
			*blocks = append(*blocks, block{
				Type: containerHeader,
				Node: n,
			})
			addBlocksFrom(n, blocks)
			*blocks = append(*blocks, block{
				Type: containerFooter,
				Node: n,
			})
		case *Terminal:
			*blocks = append(*blocks, block{
				Type: nodeBlock,
				Node: n,
			})
		}
	}
}

func fixBlockBoundaries(fileSet *token.FileSet, file *File, src []byte) error {
	var blocks []block
	addBlocksFrom(file, &blocks)

	if PrintBlocks {
		printBlocks("original blocks", blocks)
	}

	file.LocationSpan.Start.Column = 0

	offset := 0
	for i := 0; i < len(blocks); i++ {
		b := blocks[i]
		switch b.Type {
		case nodeBlock:
			n := b.Terminal()
			n.Span.Start = offset
			newPos := fileSet.Position(token.Pos(n.Span.Start + 1))
			n.LocationSpan.Start.Line = newPos.Line
			n.LocationSpan.Start.Column = newPos.Column - 1
			offset = n.Span.End + 1
		case containerHeader:
			n := b.Container()
			n.HeaderSpan.Start = offset
			newPos := fileSet.Position(token.Pos(n.HeaderSpan.Start + 1))
			n.LocationSpan.Start.Line = newPos.Line
			n.LocationSpan.Start.Column = newPos.Column - 1
			if (src[n.HeaderSpan.End] == '(' || src[n.HeaderSpan.End] == '{') && src[n.HeaderSpan.End+1] == '\n' {
				n.HeaderSpan.End++
			}
			offset = n.HeaderSpan.End + 1
		case containerFooter:
			n := b.Container()
			n.FooterSpan.Start = offset
			if (src[n.FooterSpan.End] == ')' || src[n.FooterSpan.End] == '}') && src[n.FooterSpan.End+1] == '\n' {
				n.FooterSpan.End++
			}
			newPos := fileSet.Position(token.Pos(n.FooterSpan.End + 1))
			n.LocationSpan.End.Line = newPos.Line
			n.LocationSpan.End.Column = newPos.Column
			offset = n.FooterSpan.End + 1
		default:
			panic("impossibru!")
		}
	}

	// any remaining space is part of the footer
	if offset < len(src) {
		file.FooterSpan = RuneSpan{offset, len(src) - 1}
	}

	if PrintBlocks {
		printBlocks("fixed blocks", blocks)
	}
	return nil
}

type debugBlock struct {
	BlockType    blockType
	Name         string
	LocationSpan LocationSpan
	Span         RuneSpan
}

func printBlocks(title string, blocks []block) {
	debugBlocks := make([]debugBlock, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case nodeBlock:
			debugBlocks = append(debugBlocks, debugBlock{
				BlockType:    nodeBlock,
				Name:         b.Terminal().Name,
				LocationSpan: b.Terminal().LocationSpan,
				Span:         b.Terminal().Span,
			})
		case containerHeader:
			debugBlocks = append(debugBlocks, debugBlock{
				BlockType:    containerHeader,
				Name:         b.Container().Name,
				LocationSpan: b.Container().LocationSpan,
				Span:         b.Container().HeaderSpan,
			})
		case containerFooter:
			debugBlocks = append(debugBlocks, debugBlock{
				BlockType:    containerFooter,
				Name:         b.Container().Name,
				LocationSpan: b.Container().LocationSpan,
				Span:         b.Container().FooterSpan,
			})
		default:
			panic("impossibru!")
		}
	}
	spew.Printf("----------%s----------\n", title)
	spew.Dump(debugBlocks)
	spew.Printf("--------------------\n")
}
