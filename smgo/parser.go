package smgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/text/encoding/charmap"
)

var ErrUnsupportedEncoding = errors.New("Unsupported encoding")

// Parse parses the GO source code from src and returns a *smgo.File declarations tree.
func Parse(src io.Reader, encoding string) (*File, error) {
	encoding = strings.ToUpper(encoding)
	switch encoding {
	case "UTF-8":
	case "WINDOWS-1252":
		decoder := charmap.Windows1252.NewDecoder()
		src = decoder.Reader(src)
	default:
		return nil, ErrUnsupportedEncoding
	}

	srcBytes, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading src")
	}

	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, "", srcBytes, parser.ParseComments)
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

	// visit top-level declarations only
	v := newVisitor(fset, fileAST)
	for _, decl := range fileAST.Decls {
		ast.Walk(v, decl)
	}
	// fix file LocationSpan
	pos := v.FileSet.Position(token.Pos(1))
	end := v.FileSet.Position(token.Pos(len(srcBytes)))
	v.File.LocationSpan = LocationSpan{
		Start: Location{
			Line:   pos.Line,
			Column: pos.Column,
		},
		End: Location{
			Line:   end.Line,
			Column: end.Column,
		},
	}

	ffc := v.freeFloatingCommentsBefore(len(srcBytes))
	v.AddFFCToParentContainer(ffc...)
	//for _, c := range ffc {
	//	v.AddToParentContainer(c)
	//}

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

type commentSet map[*ast.CommentGroup]struct{}

type visitor struct {
	FileSet        *token.FileSet
	File           *File
	Comments       commentSet
	astStack       []ast.Node
	containerStack []parentNode
}

func newVisitor(fset *token.FileSet, srcAST *ast.File) *visitor {
	v := &visitor{
		FileSet: fset,
	}
	// save comments to insert free-floating comments in the resulting File as Comment nodes.
	v.Comments = make(commentSet, len(srcAST.Comments))
	for _, cg := range srcAST.Comments {
		v.Comments[cg] = struct{}{}
	}

	file := v.createFile(srcAST)
	v.File = file
	v.Push(srcAST, file)
	return v
}

func (v *visitor) Push(node ast.Node, container parentNode) {
	v.astStack = append(v.astStack, node)
	v.containerStack = append(v.containerStack, container)
}

func (v *visitor) Pop() (ast.Node, parentNode) {
	l := len(v.astStack) - 1
	n := v.astStack[l]
	pn := v.containerStack[l]
	v.astStack = v.astStack[:l]
	v.containerStack = v.containerStack[:l]
	return n, pn
}

func (v *visitor) Peek() (ast.Node, parentNode) {
	return v.astStack[len(v.astStack)-1], v.containerStack[len(v.containerStack)-1]
}

func (v *visitor) AddToParentContainer(node ...Node) {
	_, parentContainer := v.Peek()
	for _, n := range node {
		parentContainer.AddNode(n)
	}
}

// merges last container with a free-floating comment, in cases like:
// type (
// ...
// ) // free-floating comment...
//
// or:
// type MyStruct struct {
// ...
// } // free-floating comment...
func mergeFFCToPrevContainer(lc Node, ffc *Terminal) bool {
	switch lastChild := lc.(type) {
	case *Container:
		switch lastChild.Type {
		case StructNode, InterfaceNode:
			if ffc.LocationSpan.Start.Line == lastChild.LocationSpan.End.Line {
				lastChild.LocationSpan.End.Column = ffc.LocationSpan.End.Column
				lastChild.FooterSpan.End = ffc.Span.End
				return true
			}
		case ConstNode, ImportNode, TypeNode, VarNode:
			if ffc.LocationSpan.Start.Line == lastChild.LocationSpan.End.Line {
				lastChild.LocationSpan.End.Column = ffc.LocationSpan.End.Column
				lastChild.FooterSpan.End = ffc.Span.End
				return true
			}
		}
	}
	return false
}

func (v *visitor) AddFFCToParentContainer(ffc ...*Terminal) {
	if len(ffc) > 0 {
		_, parentContainer := v.Peek()
		switch pc := parentContainer.(type) {
		case *File:
			childrenLen := len(pc.Children)
			if childrenLen > 0 {
				lastChild := pc.Children[childrenLen-1]
				merged := mergeFFCToPrevContainer(lastChild, ffc[0])
				if merged {
					ffc = ffc[1:]
					if len(ffc) == 0 {
						return
					}
				}
			}
			// merge last ffc to file footer
			lastFFC := ffc[len(ffc)-1]
			if lastFFC.LocationSpan.End.Line == pc.LocationSpan.End.Line {
				pc.FooterSpan.Start = lastFFC.Span.Start
				ffc = ffc[:len(ffc)-1]
			}
		case *Container:
			childrenLen := len(pc.Children)
			switch {
			// cover cases like: type ( // free-floating comment...
			case childrenLen == 0 && ffc[0].LocationSpan.Start.Line == pc.LocationSpan.Start.Line:
				pc.HeaderSpan.End = ffc[0].Span.End
				ffc = ffc[1:]
				// cover cases like: ) // free-floating comment...
			case childrenLen > 0:
				lastChild := pc.Children[childrenLen-1]
				merged := mergeFFCToPrevContainer(lastChild, ffc[0])
				if merged {
					ffc = ffc[1:]
				}
			}
			if len(ffc) == 0 {
				return
			}
			// merge last ffc to parent container footer
			lastFFC := ffc[len(ffc)-1]
			if lastFFC.LocationSpan.End.Line+1 == pc.LocationSpan.End.Line {
				pc.FooterSpan.Start = lastFFC.Span.Start
				ffc = ffc[:len(ffc)-1]
			}
		}
		for _, n := range ffc {
			parentContainer.AddNode(n)
		}
	}
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case nil:
		astNode, _ := v.Peek()
		switch node := astNode.(type) {
		case *ast.GenDecl:
			if node.Rparen.IsValid() {
				position := v.FileSet.Position(node.Rparen)
				ffc := v.freeFloatingCommentsBefore(position.Offset)
				v.AddFFCToParentContainer(ffc...)
			}
		case *ast.StructType:
			position := v.FileSet.Position(node.End())
			ffc := v.freeFloatingCommentsBefore(position.Offset)
			v.AddFFCToParentContainer(ffc...)
		case *ast.InterfaceType:
			position := v.FileSet.Position(node.End())
			ffc := v.freeFloatingCommentsBefore(position.Offset)
			v.AddFFCToParentContainer(ffc...)
		}
		v.Pop()
		return v
	case *ast.GenDecl:
		if n.Lparen.IsValid() {
			switch n.Tok {
			case token.IMPORT:
				importGroup := v.createImportGroup(n)
				ffc := v.freeFloatingCommentsBefore(importGroup.HeaderSpan.Start)
				v.AddFFCToParentContainer(ffc...)
				v.AddToParentContainer(importGroup)
				v.Push(n, importGroup)
			case token.CONST:
				constGroup := v.createConstGroup(n)
				ffc := v.freeFloatingCommentsBefore(constGroup.HeaderSpan.Start)
				v.AddFFCToParentContainer(ffc...)
				v.AddToParentContainer(constGroup)
				v.Push(n, constGroup)
			case token.TYPE:
				typeGroup := v.createTypeGroup(n)
				ffc := v.freeFloatingCommentsBefore(typeGroup.HeaderSpan.Start)
				v.AddFFCToParentContainer(ffc...)
				v.AddToParentContainer(typeGroup)
				v.Push(n, typeGroup)
			case token.VAR:
				varGroup := v.createVarGroup(n)
				ffc := v.freeFloatingCommentsBefore(varGroup.HeaderSpan.Start)
				v.AddFFCToParentContainer(ffc...)
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
				importNode := v.createImport(n, is)
				ffc := v.freeFloatingCommentsBefore(importNode.Span.Start)
				v.AddFFCToParentContainer(ffc...)
				v.AddToParentContainer(importNode)
			case token.CONST:
				vs, ok := n.Specs[0].(*ast.ValueSpec)
				if !ok {
					panic("*ast.ValueSpec expected")
				}
				constNode := v.createConst(n, vs)
				ffc := v.freeFloatingCommentsBefore(constNode.Span.Start)
				v.AddFFCToParentContainer(ffc...)
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
				ffc := v.freeFloatingCommentsBefore(varNode.Span.Start)
				v.AddFFCToParentContainer(ffc...)
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
		case token.CONST:
			constNode := v.createConstInGroup(n)
			ffc := v.freeFloatingCommentsBefore(constNode.Span.Start)
			v.AddFFCToParentContainer(ffc...)
			parentContainer.AddNode(constNode)
		case token.VAR:
			varNode := v.createVarInGroup(n)
			ffc := v.freeFloatingCommentsBefore(varNode.Span.Start)
			v.AddFFCToParentContainer(ffc...)
			parentContainer.AddNode(varNode)
		}
		return nil
	case *ast.ImportSpec:
		importNode := v.createImportInGroup(n)
		ffc := v.freeFloatingCommentsBefore(importNode.Span.Start)
		v.AddFFCToParentContainer(ffc...)
		v.AddToParentContainer(importNode)
		return nil
	case *ast.FuncDecl:
		funcNode := v.createFunc(n)
		ffc := v.freeFloatingCommentsBefore(funcNode.Span.Start)
		v.AddFFCToParentContainer(ffc...)
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
			ffc := v.freeFloatingCommentsBefore(container.HeaderSpan.Start)
			if len(ffc) > 0 {

			}
			v.AddFFCToParentContainer(ffc...)
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
			ffc := v.freeFloatingCommentsBefore(container.HeaderSpan.Start)
			v.AddFFCToParentContainer(ffc...)
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
			ffc := v.freeFloatingCommentsBefore(terminal.Span.Start)
			v.AddFFCToParentContainer(ffc...)
			v.AddToParentContainer(terminal)
			return nil
		}
	case *ast.Field:
		fieldNode := v.createField(n)
		ffc := v.freeFloatingCommentsBefore(fieldNode.Span.Start)
		v.AddFFCToParentContainer(ffc...)
		v.AddToParentContainer(fieldNode)
		return nil
	default:
		_, container := v.Peek()
		v.Push(n, container)
		return v
	}
}

func (v *visitor) freeFloatingCommentsBefore(offset int) []*Terminal {
	cgNodes := make([]*ast.CommentGroup, 0, 5)
	for cg, _ := range v.Comments {
		cgPosition := v.FileSet.Position(cg.End())
		if cgPosition.Offset < offset {
			cgNodes = append(cgNodes, cg)
		}
	}
	if len(cgNodes) > 1 {
		sort.Slice(cgNodes, func(i, j int) bool {
			posi := v.FileSet.Position(cgNodes[i].Pos())
			posj := v.FileSet.Position(cgNodes[j].Pos())
			return posi.Offset < posj.Offset
		})
	}
	comments := make([]*Terminal, 0, len(cgNodes))
	for _, cg := range cgNodes {
		delete(v.Comments, cg)
		name := strings.TrimSpace(cg.Text())
		if len(name) > 10 {
			name = name[0:10] + "..."
		}
		comments = append(comments, &Terminal{
			Type:         Comment,
			Name:         name,
			LocationSpan: locationSpanFromNode(v.FileSet, cg),
			Span:         runeSpanFromNode(v.FileSet, cg),
		})
	}
	return comments
}

func (v *visitor) createFile(n *ast.File) *File {
	f := &File{
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		FooterSpan: RuneSpan{
			Start: 0,
			End:   -1,
		},
	}
	pos := n.Pos()
	if n.Doc != nil {
		pos = n.Doc.Pos()
		delete(v.Comments, n.Doc)
	}
	position := v.FileSet.Position(pos)
	ffc := v.freeFloatingCommentsBefore(position.Offset)
	for _, c := range ffc {
		f.AddNode(c)
	}
	end := n.Name.End()
	f.AddNode(&Terminal{
		Type:         PackageNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	})
	return f
}

func (v *visitor) createConst(gd *ast.GenDecl, n *ast.ValueSpec) *Terminal {
	if gd.Doc != nil {
		delete(v.Comments, gd.Doc)
	}
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := gd.Pos()
	end := gd.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
	return &Terminal{
		Type:         ConstNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createConstGroup(n *ast.GenDecl) *Container {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
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
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := n.Pos()
	end := n.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
	return &Terminal{
		Type:         ConstNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createFunc(n *ast.FuncDecl) *Terminal {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	return &Terminal{
		Type:         FunctionNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		Span:         runeSpanFromNode(v.FileSet, n),
	}
}

func (v *visitor) createImport(gd *ast.GenDecl, n *ast.ImportSpec) *Terminal {
	if gd.Doc != nil {
		delete(v.Comments, gd.Doc)
	}
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := n.Pos()
	end := n.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
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
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createImportGroup(n *ast.GenDecl) *Container {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	c := &Container{
		Type:         ImportNode,
		Name:         "import",
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, n.Pos(), n.Lparen),
		FooterSpan:   runeSpanFromPositions(v.FileSet, n.Rparen, n.End()),
	}
	if len(n.Specs) > 0 {
		c.Children = make([]Node, 0, len(n.Specs))
	}
	return c
}

func (v *visitor) createImportInGroup(n *ast.ImportSpec) *Terminal {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := n.Pos()
	end := n.End()
	if n.Comment != nil {
		end = n.End()
		delete(v.Comments, n.Comment)
	}
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
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createInterface(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) *Container {
	st, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		panic("*ast.InterfaceType expected")
	}
	if genDecl.Doc != nil {
		delete(v.Comments, genDecl.Doc)
	}
	if typeSpec.Doc != nil {
		delete(v.Comments, typeSpec.Doc)
	}
	pos := genDecl.Pos()
	end := genDecl.End()
	if typeSpec.Comment != nil {
		end = typeSpec.Comment.End()
		delete(v.Comments, typeSpec.Comment)
	}
	container := &Container{
		Type:         InterfaceNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, pos, st.Methods.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Methods.Closing, end),
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
	if typeSpec.Doc != nil {
		delete(v.Comments, typeSpec.Doc)
	}
	pos := typeSpec.Pos()
	end := st.Methods.Closing
	if typeSpec.Comment != nil {
		end = typeSpec.Comment.End()
		delete(v.Comments, typeSpec.Comment)
	}
	container := &Container{
		Type:         InterfaceNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, pos, st.Methods.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Methods.Closing, end),
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
	if genDecl.Doc != nil {
		delete(v.Comments, genDecl.Doc)
	}
	if typeSpec.Doc != nil {
		delete(v.Comments, typeSpec.Doc)
	}
	pos := genDecl.Pos()
	end := genDecl.End()
	if typeSpec.Comment != nil {
		end = typeSpec.Comment.End()
		delete(v.Comments, typeSpec.Comment)
	}
	container := &Container{
		Type:         StructNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, pos, st.Fields.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Fields.Closing, end),
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
	if typeSpec.Doc != nil {
		delete(v.Comments, typeSpec.Doc)
	}
	pos := typeSpec.Pos()
	end := st.Fields.Closing
	if typeSpec.Comment != nil {
		end = typeSpec.Comment.End()
		delete(v.Comments, typeSpec.Comment)
	}
	container := &Container{
		Type:         StructNode,
		Name:         typeSpec.Name.Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, pos, st.Fields.Opening),
		FooterSpan:   runeSpanFromPositions(v.FileSet, st.Fields.Closing, end),
	}
	if len(st.Fields.List) > 0 {
		container.Children = make([]Node, 0, len(st.Fields.List))
	}
	return container
}

func (v *visitor) createField(n *ast.Field) *Terminal {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := n.Pos()
	end := n.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
	return &Terminal{
		Type:         FieldNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createType(genDecl *ast.GenDecl, n *ast.TypeSpec) *Terminal {
	if genDecl.Doc != nil {
		delete(v.Comments, genDecl.Doc)
	}
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := genDecl.Pos()
	end := genDecl.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
	return &Terminal{
		Type:         TypeNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createTypeGroup(n *ast.GenDecl) *Container {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	c := &Container{
		Type:         TypeNode,
		Name:         "type",
		LocationSpan: locationSpanFromNode(v.FileSet, n),
		HeaderSpan:   runeSpanFromPositions(v.FileSet, n.Pos(), n.Lparen),
		FooterSpan:   runeSpanFromPositions(v.FileSet, n.Rparen, n.End()),
	}
	if len(n.Specs) > 0 {
		c.Children = make([]Node, 0, len(n.Specs))
	}
	return c
}

func (v *visitor) createTypeInGroup(n *ast.TypeSpec) *Terminal {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := n.Pos()
	end := n.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
	return &Terminal{
		Type:         TypeNode,
		Name:         n.Name.Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createVar(gd *ast.GenDecl, n *ast.ValueSpec) *Terminal {
	if gd.Doc != nil {
		delete(v.Comments, gd.Doc)
	}
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := gd.Pos()
	end := gd.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
	return &Terminal{
		Type:         VarNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func (v *visitor) createVarGroup(n *ast.GenDecl) *Container {
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
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
	if n.Doc != nil {
		delete(v.Comments, n.Doc)
	}
	pos := n.Pos()
	end := n.End()
	if n.Comment != nil {
		end = n.Comment.End()
		delete(v.Comments, n.Comment)
	}
	return &Terminal{
		Type:         VarNode,
		Name:         n.Names[0].Name,
		LocationSpan: locationSpanFromPositions(v.FileSet, pos, end),
		Span:         runeSpanFromPositions(v.FileSet, pos, end),
	}
}

func locationFromPosition(fset *token.FileSet, pos token.Pos) Location {
	return Location{
		Line:   fset.Position(pos).Line,
		Column: fset.Position(pos).Column,
	}
}

func locationSpanFromPositions(fset *token.FileSet, pos, end token.Pos) LocationSpan {
	return LocationSpan{
		Start: locationFromPosition(fset, pos),
		End:   locationFromPosition(fset, end),
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
