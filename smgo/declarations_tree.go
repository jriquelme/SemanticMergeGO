package smgo

import "fmt"

// File is the root of the declarations tree.
type File struct {
	LocationSpan  LocationSpan
	FooterSpan    RuneSpan
	Containers    []*Container
	Nodes         []*Node
	ParsingErrors []*ParsingError
}

type ContainerType int

//go:generate stringer -type=ContainerType

const (
	StructContainer ContainerType = iota
)

type Container struct {
	Type         ContainerType
	Name         string
	LocationSpan LocationSpan
	HeaderSpan   RuneSpan
	FooterSpan   RuneSpan
	Containers   []*Container
	Nodes        []*Node
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
)

type Node struct {
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
