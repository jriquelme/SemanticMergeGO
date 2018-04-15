package smgo

import "fmt"

// Node is a Container or a Terminal instance.
type Node interface{}

// File is the root of the declarations tree.
type File struct {
	LocationSpan  LocationSpan
	FooterSpan    RuneSpan
	Children      []Node
	ParsingErrors []*ParsingError
}

func (f *File) AddNode(node Node) {
	f.Children = append(f.Children, node)
}

func (f *File) Nodes() []Node {
	return f.Children
}

type NodeType int

//go:generate stringer -type=NodeType

const (
	PackageNode NodeType = iota
	FunctionNode
	FieldNode
	ImportNode
	ConstNode
	VarNode
	TypeNode
	StructNode
	InterfaceNode
)

type Container struct {
	Type         NodeType
	Name         string
	LocationSpan LocationSpan
	HeaderSpan   RuneSpan
	FooterSpan   RuneSpan
	Children     []Node
}

func (c *Container) AddNode(node Node) {
	c.Children = append(c.Children, node)
}

func (c *Container) Nodes() []Node {
	return c.Children
}

type Terminal struct {
	Type         NodeType
	Name         string
	LocationSpan LocationSpan
	Span         RuneSpan
}

type ParsingError struct {
	Location Location
	Message  string
}

type Location struct {
	Line   int
	Column int
}

func (l Location) String() string {
	return fmt.Sprintf("[L:%d C:%d]", l.Line, l.Column)
}

type LocationSpan struct {
	Start Location
	End   Location
}

func (ls LocationSpan) String() string {
	return fmt.Sprintf("S:%s E:%s", ls.Start, ls.End)
}

type RuneSpan struct {
	Start int
	End   int
}

func (rs RuneSpan) String() string {
	return fmt.Sprintf("[%d, %d]", rs.Start, rs.End)
}
