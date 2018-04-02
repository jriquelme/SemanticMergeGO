package smgo

// File is the root of the declarations tree.
type File struct {
	LocationSpan  LocationSpan
	FooterSpan    RuneSpan
	Containers    []*Container
	Nodes         []*Node
	ParsingErrors []*ParsingError
}

type ContainerType int

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

const (
	PackageNode NodeType = iota
	FunctionNode
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

type LocationSpan struct {
	Start Location
	End   Location
}

type RuneSpan struct {
	Start int
	End   int
}
