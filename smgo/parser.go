package smgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
)

var PrintBlocks bool

var ErrUnsupportedEncoding = errors.New("Unsupported encoding")

// Parse parses the GO source code from the src io.ReadSeeker and returns a declarations tree *smgo.File.
func Parse(src io.Reader, encoding string) (*File, error) {
	srcBytes, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading src")
	}
	encoding = strings.ToUpper(encoding)
	if encoding != "UTF-8" {
		return nil, ErrUnsupportedEncoding
	}

	fset := token.NewFileSet()
	srcAST, err := parser.ParseFile(fset, "", srcBytes, parser.ParseComments)
	if err != nil {
		file := &File{
			LocationSpan: LocationSpan{
				Start: Location{1, 0},
				End:   Location{1, 0},
			},
			FooterSpan: RuneSpan{0, -1},
			ParsingErrors: []*ParsingError{
				{
					Location: Location{1, 0},
					Message:  err.Error(),
				},
			},
		}
		return file, nil
	}

	fv := &fileVisitor{
		parserState: &parserState{
			FileSet: fset,
		},
	}
	ast.Walk(fv, srcAST)

	if PrintBlocks {
		printBlocks(fv.Blocks)
	}
	err = fixBlockBoundaries(fv.File, fv.Blocks, srcBytes)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading fixing boundaries")
	}
	if PrintBlocks {
		printBlocks(fv.Blocks)
	}

	return fv.File, nil
}

type parserState struct {
	FileSet *token.FileSet
	File    *File
	Blocks  []block
}

type fileVisitor struct {
	*parserState
}

func (v *fileVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.File:
		v.File = createFile(v.FileSet, n)
		v.Blocks = append(v.Blocks, block{
			Type:      nodeBlock,
			Node:      v.File.Nodes[0],
			Container: nil,
		})
		return v
	case *ast.GenDecl:
		return &genDeclVisitor{
			parserState: v.parserState,
			GenDecl:     n,
		}
	case *ast.FuncDecl:
		funcNode := createFunc(v.FileSet, n)
		v.File.Nodes = append(v.File.Nodes, funcNode)
		v.Blocks = append(v.Blocks, block{
			Type:      nodeBlock,
			Node:      funcNode,
			Container: nil,
		})
		return nil
	default:
		return nil
	}
}

type genDeclVisitor struct {
	*parserState
	GenDecl *ast.GenDecl
}

func (v *genDeclVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.TypeSpec:
		switch n.Type.(type) {
		case *ast.StructType:
			container, blocks := createStruct(v.FileSet, n)
			if v.GenDecl.Lparen == token.NoPos {
				container.LocationSpan.Start = locationFromPosition(v.FileSet, v.GenDecl.Pos())
				container.HeaderSpan.Start = v.FileSet.Position(v.GenDecl.Pos()).Offset
			}
			v.File.Containers = append(v.File.Containers, container)
			v.Blocks = append(v.Blocks, blocks...)
			return nil
		}
	case *ast.ImportSpec:
		importNode := createImport(v.FileSet, n)
		v.File.Nodes = append(v.File.Nodes, importNode)
		v.Blocks = append(v.Blocks, block{
			Type:      nodeBlock,
			Node:      importNode,
			Container: nil,
		})
		return nil
	case *ast.ValueSpec:
		switch v.GenDecl.Tok {
		case token.CONST:
			constNode := createConst(v.FileSet, n)
			v.File.Nodes = append(v.File.Nodes, constNode)
			v.Blocks = append(v.Blocks, block{
				Type:      nodeBlock,
				Node:      constNode,
				Container: nil,
			})
		}
		return nil
	}
	return nil
}

func createFile(fset *token.FileSet, n *ast.File) *File {
	return &File{
		LocationSpan: locationSpanFromNode(fset, n),
		FooterSpan: RuneSpan{
			Start: 0,
			End:   -1,
		},
		Nodes: []*Node{
			{
				Type: PackageNode,
				Name: n.Name.Name,
				LocationSpan: LocationSpan{
					Start: locationFromPosition(fset, n.Package),
					End:   locationFromPositions(fset, n.Name.Pos(), n.Name.End()),
				},
				Span: runeSpanFromPositions(fset, n.Package, n.Name.End()),
			},
		},
	}
}

func createConst(fset *token.FileSet, n *ast.ValueSpec) *Node {
	return &Node{
		Type:         ConstNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(fset, n),
		Span:         runeSpanFromNode(fset, n),
	}
}

func createFunc(fset *token.FileSet, n *ast.FuncDecl) *Node {
	return &Node{
		Type:         FunctionNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromNode(fset, n),
		Span:         runeSpanFromNode(fset, n),
	}
}

func createStruct(fset *token.FileSet, typeSpec *ast.TypeSpec) (*Container, []block) {
	st, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		panic("*ast.StructType expected")
	}

	blocks := make([]block, 0, len(st.Fields.List)+2)
	container := &Container{
		Type:         StructContainer,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromNode(fset, typeSpec),
		HeaderSpan:   runeSpanFromPositions(fset, typeSpec.Pos(), st.Fields.Opening),
		FooterSpan:   runeSpanFromPositions(fset, st.Fields.Closing, st.Fields.Closing),
		Containers:   nil,
		Nodes:        make([]*Node, 0, len(st.Fields.List)),
	}
	blocks = append(blocks, block{
		Type:      containerHeader,
		Container: container,
	})

	ast.Inspect(typeSpec.Type, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Field:
			field := &Node{
				Type:         FieldNode,
				Name:         n.Names[0].Name, // FIXME: won't work with anonymous fields
				LocationSpan: locationSpanFromNode(fset, n),
				Span:         runeSpanFromNode(fset, n),
			}
			container.Nodes = append(container.Nodes, field)
			blocks = append(blocks, block{
				Type: nodeBlock,
				Node: field,
			})
			return false
		default:
			return true
		}
	})

	blocks = append(blocks, block{
		Type:      containerFooter,
		Container: container,
	})

	return container, blocks
}

func createImport(fset *token.FileSet, n *ast.ImportSpec) *Node {
	var name string
	switch n.Path.Kind {
	case token.STRING:
		name = n.Path.Value[1 : len(n.Path.Value)-1]
	default:
		panic("Unknown token type for import Path")
	}
	return &Node{
		Type:         ImportNode,
		Name:         name,
		LocationSpan: locationSpanFromNode(fset, n),
		Span:         runeSpanFromNode(fset, n),
	}
}

func locationFromPosition(fset *token.FileSet, pos token.Pos) Location {
	return Location{
		Line:   fset.Position(pos).Line,
		Column: fset.Position(pos).Column,
	}
}

func locationFromPositions(fset *token.FileSet, pos1, pos2 token.Pos) Location {
	return Location{
		Line:   fset.Position(pos1).Line,
		Column: fset.Position(pos2).Column,
	}
}

func locationSpanFromPositions(fset *token.FileSet, pos1, pos2 token.Pos) LocationSpan {
	return LocationSpan{
		Start: locationFromPosition(fset, pos1),
		End:   locationFromPosition(fset, pos2),
	}
}

func locationSpanFromNode(fset *token.FileSet, n ast.Node) LocationSpan {
	return LocationSpan{
		Start: locationFromPosition(fset, n.Pos()),
		End:   locationFromPosition(fset, n.End()),
	}
}

func runeSpanFromNode(fset *token.FileSet, n ast.Node) RuneSpan {
	return RuneSpan{
		Start: fset.Position(n.Pos()).Offset,
		End:   fset.Position(n.End()).Offset,
	}
}

func runeSpanFromPositions(fset *token.FileSet, pos1, pos2 token.Pos) RuneSpan {
	return RuneSpan{
		Start: fset.Position(pos1).Offset,
		End:   fset.Position(pos2).Offset,
	}
}
