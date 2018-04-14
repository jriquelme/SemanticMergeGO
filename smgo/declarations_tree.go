package smgo

import "fmt"

type Node interface {
	IsTerminal() bool
}

// File is the root of the declarations tree.
type File struct {
	LocationSpan  LocationSpan
	FooterSpan    RuneSpan
	Children      []Node
	ParsingErrors []*ParsingError
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

func (c *Container) IsTerminal() bool {
	return false
}

type Terminal struct {
	Type         NodeType
	Name         string
	LocationSpan LocationSpan
	Span         RuneSpan
}

func (t *Terminal) IsTerminal() bool {
	return true
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
