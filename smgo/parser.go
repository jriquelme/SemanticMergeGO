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

var ErrUnsupportedEncoding = errors.New("Unsupported encoding")

// Parse parses the GO source code from src and returns a *smgo.File declarations tree.
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

	err = fixBlockBoundaries(fv.File, srcBytes)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading fixing boundaries")
	}

	return fv.File, nil
}

type parserState struct {
	FileSet *token.FileSet
	File    *File
}

type parentNode interface {
	AddNode(node Node)
	Nodes() []Node
}

type fileVisitor struct {
	*parserState
}

func (v *fileVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.File:
		v.File = createFile(v.FileSet, n)
		return v
	case *ast.GenDecl:
		var parent parentNode = v.File
		if n.Lparen != token.NoPos {
			switch n.Tok {
			case token.CONST:
				parent = &Container{
					Type:         ConstNode,
					Name:         "const",
					LocationSpan: locationSpanFromNode(v.FileSet, n),
					HeaderSpan:   runeSpanFromPositions(v.FileSet, n.Pos(), n.Lparen),
					FooterSpan:   runeSpanFromPositions(v.FileSet, n.Rparen, n.Rparen),
					Children:     make([]Node, 0, len(n.Specs)),
				}
				v.File.AddNode(parent)
			}
		}
		return &genDeclVisitor{
			parserState: v.parserState,
			GenDecl:     n,
			Parent:      parent,
		}
	case *ast.FuncDecl:
		funcNode := createFunc(v.FileSet, n)
		v.File.AddNode(funcNode)
		return nil
	default:
		return nil
	}
}

type genDeclVisitor struct {
	*parserState
	GenDecl *ast.GenDecl
	Parent  parentNode
}

func (v *genDeclVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.TypeSpec:
		switch n.Type.(type) {
		case *ast.InterfaceType:
			container := createInterface(v.FileSet, n)
			if v.GenDecl.Lparen == token.NoPos {
				container.LocationSpan.Start = locationFromPosition(v.FileSet, v.GenDecl.Pos())
				container.HeaderSpan.Start = v.FileSet.Position(v.GenDecl.Pos()).Offset
			}
			v.Parent.AddNode(container)
			return nil
		case *ast.StructType:
			container := createStruct(v.FileSet, n)
			if v.GenDecl.Lparen == token.NoPos {
				container.LocationSpan.Start = locationFromPosition(v.FileSet, v.GenDecl.Pos())
				container.HeaderSpan.Start = v.FileSet.Position(v.GenDecl.Pos()).Offset
			}
			v.Parent.AddNode(container)
			return nil
		default:
			typeNode := createType(v.FileSet, n)
			v.Parent.AddNode(typeNode)
			return nil
		}
	case *ast.ImportSpec:
		importNode := createImport(v.FileSet, n)
		v.Parent.AddNode(importNode)
		return nil
	case *ast.ValueSpec:
		switch v.GenDecl.Tok {
		case token.CONST:
			constNode := createConst(v.FileSet, n)
			v.Parent.AddNode(constNode)
		case token.VAR:
			varNode := createVar(v.FileSet, n)
			v.Parent.AddNode(varNode)
		}
		return nil
	}
	return nil
}

func createFile(fset *token.FileSet, n *ast.File) *File {
	f := &File{
		LocationSpan: locationSpanFromNode(fset, n),
		FooterSpan: RuneSpan{
			Start: 0,
			End:   -1,
		},
	}
	f.AddNode(&Terminal{
		Type: PackageNode,
		Name: n.Name.Name,
		LocationSpan: LocationSpan{
			Start: locationFromPosition(fset, n.Package),
			End:   locationFromPositions(fset, n.Name.Pos(), n.Name.End()),
		},
		Span: runeSpanFromPositions(fset, n.Package, n.Name.End()),
	})
	return f
}

func createConst(fset *token.FileSet, n *ast.ValueSpec) *Terminal {
	return &Terminal{
		Type:         ConstNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(fset, n),
		Span:         runeSpanFromNode(fset, n),
	}
}

func createFunc(fset *token.FileSet, n *ast.FuncDecl) *Terminal {
	return &Terminal{
		Type:         FunctionNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromNode(fset, n),
		Span:         runeSpanFromNode(fset, n),
	}
}

func createInterface(fset *token.FileSet, typeSpec *ast.TypeSpec) *Container {
	st, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		panic("*ast.InterfaceType expected")
	}

	container := &Container{
		Type:         InterfaceNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromNode(fset, typeSpec),
		HeaderSpan:   runeSpanFromPositions(fset, typeSpec.Pos(), st.Methods.Opening),
		FooterSpan:   runeSpanFromPositions(fset, st.Methods.Closing, st.Methods.Closing),
		Children:     make([]Node, 0, len(st.Methods.List)),
	}

	ast.Inspect(typeSpec.Type, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Field:
			field := &Terminal{
				Type:         FunctionNode,
				Name:         n.Names[0].Name, // FIXME: won't work with anonymous fields
				LocationSpan: locationSpanFromNode(fset, n),
				Span:         runeSpanFromNode(fset, n),
			}
			container.AddNode(field)
			return false
		default:
			return true
		}
	})

	return container
}

func createStruct(fset *token.FileSet, typeSpec *ast.TypeSpec) *Container {
	st, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		panic("*ast.StructType expected")
	}

	container := &Container{
		Type:         StructNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromNode(fset, typeSpec),
		HeaderSpan:   runeSpanFromPositions(fset, typeSpec.Pos(), st.Fields.Opening),
		FooterSpan:   runeSpanFromPositions(fset, st.Fields.Closing, st.Fields.Closing),
		Children:     make([]Node, 0, len(st.Fields.List)),
	}

	ast.Inspect(typeSpec.Type, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Field:
			field := &Terminal{
				Type:         FieldNode,
				Name:         n.Names[0].Name, // FIXME: won't work with anonymous fields
				LocationSpan: locationSpanFromNode(fset, n),
				Span:         runeSpanFromNode(fset, n),
			}
			container.AddNode(field)
			return false
		default:
			return true
		}
	})

	return container
}

func createType(fset *token.FileSet, n *ast.TypeSpec) *Terminal {
	return &Terminal{
		Type:         TypeNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromNode(fset, n),
		Span:         runeSpanFromNode(fset, n),
	}
}

func createVar(fset *token.FileSet, n *ast.ValueSpec) *Terminal {
	return &Terminal{
		Type:         VarNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(fset, n),
		Span:         runeSpanFromNode(fset, n),
	}
}

func createImport(fset *token.FileSet, n *ast.ImportSpec) *Terminal {
	var name string
	switch n.Path.Kind {
	case token.STRING:
		name = n.Path.Value[1 : len(n.Path.Value)-1]
	default:
		panic("Unknown token type for import Path")
	}
	return &Terminal{
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
