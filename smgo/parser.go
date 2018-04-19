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

	v := &visitor{
		FileSet: fset,
	}
	ast.Walk(v, srcAST)

	err = fixBlockBoundaries(fset, v.File, srcBytes)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading fixing boundaries")
	}

	return v.File, nil
}

type parentNode interface {
	AddNode(node Node)
	Nodes() []Node
}

type visitor struct {
	FileSet        *token.FileSet
	File           *File
	astStack       []ast.Node
	containerStack []parentNode
}

func (v *visitor) Push(node ast.Node, container parentNode) {
	v.astStack = append(v.astStack, node)
	v.containerStack = append(v.containerStack, container)
}

func (v *visitor) Pop() {
	v.astStack = v.astStack[:len(v.astStack)-1]
	v.containerStack = v.containerStack[:len(v.containerStack)-1]
}

func (v *visitor) Peek() (ast.Node, parentNode) {
	return v.astStack[len(v.astStack)-1], v.containerStack[len(v.containerStack)-1]
}

func (v *visitor) AddToParentContainer(node Node) {
	_, parentContainer := v.Peek()
	parentContainer.AddNode(node)
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case nil:
		v.Pop()
		return v
	case *ast.File:
		file := v.createFile(n)
		v.File = file
		v.Push(n, file)
		return v
	case *ast.GenDecl:
		if n.Lparen.IsValid() {
			switch n.Tok {
			case token.IMPORT:
			case token.CONST:
				constGroup := v.createConstGroup(n)
				v.AddToParentContainer(constGroup)
				v.Push(n, constGroup)
			case token.TYPE:
			case token.VAR:
				varGroup := v.createVarGroup(n)
				v.AddToParentContainer(varGroup)
				v.Push(n, varGroup)
			}
			return v
		} else {
			switch n.Tok {
			case token.IMPORT:
				is, ok := n.Specs[0].(*ast.ImportSpec)
				if !ok {
					panic("*ast.ValueSpec expected")
				}
				importNode := v.createImport(is)
				v.AddToParentContainer(importNode)
			case token.CONST:
				vs, ok := n.Specs[0].(*ast.ValueSpec)
				if !ok {
					panic("*ast.ValueSpec expected")
				}
				constNode := v.createConst(n, vs)
				v.AddToParentContainer(constNode)
			case token.TYPE:
				_, parentContainer := v.Peek()
				v.Push(n, parentContainer)
				return v
			case token.VAR:
				vs, ok := n.Specs[0].(*ast.ValueSpec)
				if !ok {
					panic("*ast.ValueSpec expected")
				}
				varNode := v.createVar(n, vs)
				v.AddToParentContainer(varNode)
			}
			return nil
		}
	case *ast.ValueSpec:
		parentASTNode, parentContainer := v.Peek()
		gd, ok := parentASTNode.(*ast.GenDecl)
		if !ok {
			panic("*ast.GenDecl expected")
		}
		switch gd.Tok {
		case token.IMPORT:
		case token.CONST:
			constNode := v.createConstInGroup(n)
			parentContainer.AddNode(constNode)
		case token.TYPE:
		case token.VAR:
			varNode := v.createVarInGroup(n)
			parentContainer.AddNode(varNode)
		}
		return nil
	case *ast.FuncDecl:
		funcNode := v.createFunc(n)
		v.AddToParentContainer(funcNode)
		return nil
	case *ast.TypeSpec:
		parentASTNode, _ := v.Peek()
		gd, ok := parentASTNode.(*ast.GenDecl)
		if !ok {
			panic("*ast.GenDecl expected")
		}
		switch n.Type.(type) {
		case *ast.InterfaceType:
			var container *Container
			if gd.Lparen.IsValid() {
				container = v.createInterfaceInGroup(n)
			} else {
				container = v.createInterface(gd, n)
			}
			v.AddToParentContainer(container)
			v.Push(n, container)
			return v
		case *ast.StructType:
			var container *Container
			if gd.Lparen.IsValid() {
				container = v.createStructInGroup(n)
			} else {
				container = v.createStruct(gd, n)
			}
			v.AddToParentContainer(container)
			v.Push(n, container)
			return v
		default:
			var terminal *Terminal
			if gd.Lparen.IsValid() {
				terminal = v.createTypeInGroup(n)
			} else {
				terminal = v.createType(gd, n)
			}
			v.AddToParentContainer(terminal)
			return nil
		}
	case *ast.Field:
		fieldNode := v.createField(n)
		v.AddToParentContainer(fieldNode)
		return nil
	default:
		_, container := v.Peek()
		v.Push(n, container)
		return v
	}
}

func (v *visitor) createFile(n *ast.File) *File {
	f := &File{
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		FooterSpan: RuneSpan{
			Start: 0,
			End:   -1,
		},
	}
	f.AddNode(&Terminal{
		Type: PackageNode,
		Name: n.Name.Name,
		LocationSpan: LocationSpan{
			Start: locationFromPosition(v.FileSet, n.Package),
			End:   locationFromPositions(v.FileSet, n.Name.Pos(), n.Name.End()),
		},
		Span: runeSpanFromPositions(v.FileSet, n.Package, n.Name.End()),
	})
	return f
}

func (v *visitor) createConst(gd *ast.GenDecl, n *ast.ValueSpec) *Terminal {
	return &Terminal{
		Type:         ConstNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(v.FileSet, gd),
		Span:         runeSpanFromNode(v.FileSet, gd),
	}
}

func (v *visitor) createConstGroup(n *ast.GenDecl) *Container {
	c := &Container{
		Type:         ConstNode,
		Name:         "const",
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, n.Pos(), n.Lparen),
		FooterSpan:   runeSpanFromPositions(v.FileSet, n.Rparen, n.End()),
	}
	if len(n.Specs) > 0 {
		c.Children = make([]Node, 0, len(n.Specs))
	}
	return c
}

func (v *visitor) createConstInGroup(n *ast.ValueSpec) *Terminal {
	return &Terminal{
		Type:         ConstNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		Span:         runeSpanFromNode(v.FileSet, n),
	}
}

func (v *visitor) createFunc(n *ast.FuncDecl) *Terminal {
	return &Terminal{
		Type:         FunctionNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		Span:         runeSpanFromNode(v.FileSet, n),
	}
}

func (v *visitor) createImport(n *ast.ImportSpec) *Terminal {
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
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		Span:         runeSpanFromNode(v.FileSet, n),
	}
}

func (v *visitor) createInterface(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) *Container {
	st, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		panic("*ast.InterfaceType expected")
	}

	container := &Container{
		Type:         InterfaceNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, genDecl),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, genDecl.Pos(), st.Methods.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Methods.Closing, genDecl.End()),
	}
	if len(st.Methods.List) > 0 {
		container.Children = make([]Node, 0, len(st.Methods.List))
	}
	return container
}

func (v *visitor) createInterfaceInGroup(typeSpec *ast.TypeSpec) *Container {
	st, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		panic("*ast.InterfaceType expected")
	}

	container := &Container{
		Type:         InterfaceNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, typeSpec),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, typeSpec.Pos(), st.Methods.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Methods.Closing, st.Methods.Closing),
	}
	if len(st.Methods.List) > 0 {
		container.Children = make([]Node, 0, len(st.Methods.List))
	}
	return container
}

func (v *visitor) createStruct(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) *Container {
	st, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		panic("*ast.StructType expected")
	}

	container := &Container{
		Type:         StructNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, genDecl),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, genDecl.Pos(), st.Fields.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Fields.Closing, genDecl.End()),
	}
	if len(st.Fields.List) > 0 {
		container.Children = make([]Node, 0, len(st.Fields.List))
	}
	return container
}

func (v *visitor) createStructInGroup(typeSpec *ast.TypeSpec) *Container {
	st, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		panic("*ast.StructType expected")
	}

	container := &Container{
		Type:         StructNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, typeSpec),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, typeSpec.Pos(), st.Fields.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Fields.Closing, st.Fields.Closing),
	}
	if len(st.Fields.List) > 0 {
		container.Children = make([]Node, 0, len(st.Fields.List))
	}
	return container
}

func (v *visitor) createField(n *ast.Field) *Terminal {
	return &Terminal{
		Type:         FieldNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		Span:         runeSpanFromNode(v.FileSet, n),
	}
}

func (v *visitor) createType(genDecl *ast.GenDecl, n *ast.TypeSpec) *Terminal {
	return &Terminal{
		Type:         TypeNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, genDecl),
		Span:         runeSpanFromNode(v.FileSet, genDecl),
	}
}

func (v *visitor) createTypeInGroup(n *ast.TypeSpec) *Terminal {
	return &Terminal{
		Type:         TypeNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		Span:         runeSpanFromNode(v.FileSet, n),
	}
}

func (v *visitor) createVar(gd *ast.GenDecl, n *ast.ValueSpec) *Terminal {
	return &Terminal{
		Type:         VarNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(v.FileSet, gd),
		Span:         runeSpanFromNode(v.FileSet, gd),
	}
}

func (v *visitor) createVarGroup(n *ast.GenDecl) *Container {
	c := &Container{
		Type:         VarNode,
		Name:         "var",
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, n.Pos(), n.Lparen),
		FooterSpan:   runeSpanFromPositions(v.FileSet, n.Rparen, n.End()),
	}
	if len(n.Specs) > 0 {
		c.Children = make([]Node, 0, len(n.Specs))
	}
	return c
}

func (v *visitor) createVarInGroup(n *ast.ValueSpec) *Terminal {
	return &Terminal{
		Type:         VarNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		Span:         runeSpanFromNode(v.FileSet, n),
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
