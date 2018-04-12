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
				container.LocationSpan.Start = Location{
					Line:   v.FileSet.Position(v.GenDecl.Pos()).Line,
					Column: v.FileSet.Position(v.GenDecl.Pos()).Column,
				}
				container.HeaderSpan.Start = v.FileSet.Position(v.GenDecl.Pos()).Offset
			}
			v.File.Containers = append(v.File.Containers, container)
			v.Blocks = append(v.Blocks, blocks...)
			return nil
		}
	}
	return nil
}

func createFile(fset *token.FileSet, n *ast.File) *File {
	return &File{
		LocationSpan: LocationSpan{
			Start: Location{
				Line:   fset.Position(n.Pos()).Line,
				Column: fset.Position(n.Pos()).Column,
			},
			End: Location{
				Line:   fset.Position(n.End()).Line,
				Column: fset.Position(n.End()).Column,
			},
		},
		FooterSpan: RuneSpan{
			Start: 0,
			End:   -1,
		},
		Nodes: []*Node{
			{
				Type: PackageNode,
				Name: n.Name.Name,
				LocationSpan: LocationSpan{
					Start: Location{
						Line:   fset.Position(n.Package).Line,
						Column: fset.Position(n.Package).Column,
					},
					End: Location{
						Line:   fset.Position(n.Name.Pos()).Line,
						Column: fset.Position(n.Name.End()).Column,
					},
				},
				Span: RuneSpan{
					Start: fset.Position(n.Package).Offset,
					End:   fset.Position(n.Name.End()).Offset,
				},
			},
		},
	}
}

func createFunc(fset *token.FileSet, n *ast.FuncDecl) *Node {
	return &Node{
		Type: FunctionNode,
		Name: n.Name.Name,
		LocationSpan: LocationSpan{
			Start: Location{
				Line:   fset.Position(n.Pos()).Line,
				Column: fset.Position(n.Pos()).Column,
			},
			End: Location{
				Line:   fset.Position(n.End()).Line,
				Column: fset.Position(n.End()).Column,
			},
		},
		Span: RuneSpan{
			Start: fset.Position(n.Pos()).Offset,
			End:   fset.Position(n.End()).Offset,
		},
	}
}

func createStruct(fset *token.FileSet, typeSpec *ast.TypeSpec) (*Container, []block) {
	st, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		panic("*ast.StructType expected")
	}

	blocks := make([]block, 0, len(st.Fields.List)+2)
	container := &Container{
		Type: StructContainer,
		Name: typeSpec.Name.Name,
		LocationSpan: LocationSpan{
			Start: Location{
				Line:   fset.Position(typeSpec.Pos()).Line,
				Column: fset.Position(typeSpec.Pos()).Column,
			},
			End: Location{
				Line:   fset.Position(typeSpec.End()).Line,
				Column: fset.Position(typeSpec.End()).Column,
			},
		},
		HeaderSpan: RuneSpan{
			Start: fset.Position(typeSpec.Pos()).Offset,
			End:   fset.Position(st.Fields.Opening).Offset,
		},
		FooterSpan: RuneSpan{
			Start: fset.Position(st.Fields.Closing).Offset,
			End:   fset.Position(st.Fields.Closing).Offset,
		},
		Containers: nil,
		Nodes:      make([]*Node, 0, len(st.Fields.List)),
	}
	blocks = append(blocks, block{
		Type:      containerHeader,
		Container: container,
	})

	ast.Inspect(typeSpec.Type, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Field:
			field := &Node{
				Type: FieldNode,
				Name: n.Names[0].Name, // FIXME: won't work with anonymous fields
				LocationSpan: LocationSpan{
					Start: Location{
						Line:   fset.Position(n.Pos()).Line,
						Column: fset.Position(n.Pos()).Column,
					},
					End: Location{
						Line:   fset.Position(n.End()).Line,
						Column: fset.Position(n.End()).Column,
					},
				},
				Span: RuneSpan{
					Start: fset.Position(n.Pos()).Offset,
					End:   fset.Position(n.End()).Offset,
				},
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
