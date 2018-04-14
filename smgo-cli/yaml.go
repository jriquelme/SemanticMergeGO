package main

import "github.com/jriquelme/SemanticMergeGO/smgo"

type File struct {
	Type                  string           `yaml:"type"`
	Name                  string           `yaml:"name"`
	LocationSpan          map[string][]int `yaml:"locationSpan,flow"`
	FooterSpan            []int            `yaml:"footerSpan,flow"`
	ParsingErrorsDetected bool             `yaml:"parsingErrorsDetected"`
	Children              []interface{}    `yaml:"children,omitempty"`
	ParsingErrors         []*ParsingError  `yaml:"parsingErrors,omitempty"`
}

type Container struct {
	Type         string           `yaml:"type"`
	Name         string           `yaml:"name"`
	LocationSpan map[string][]int `yaml:"locationSpan,flow"`
	HeaderSpan   []int            `yaml:"headerSpan,flow"`
	FooterSpan   []int            `yaml:"footerSpan,flow"`
	Children     []interface{}    `yaml:"children,omitempty"`
}

type Terminal struct {
	Type         string           `yaml:"type"`
	Name         string           `yaml:"name"`
	LocationSpan map[string][]int `yaml:"locationSpan,flow"`
	Span         []int            `yaml:"span,flow"`
}

type ParsingError struct {
	Location []int  `yaml:"location,flow"`
	Message  string `yaml:"message"`
}

func toFile(dtFile *smgo.File) *File {
	f := &File{
		Type: "file",
		LocationSpan: map[string][]int{
			"start": {dtFile.LocationSpan.Start.Line, dtFile.LocationSpan.Start.Column},
			"end":   {dtFile.LocationSpan.End.Line, dtFile.LocationSpan.End.Column},
		},
		FooterSpan:            []int{dtFile.FooterSpan.Start, dtFile.FooterSpan.End},
		ParsingErrorsDetected: len(dtFile.ParsingErrors) > 0,
		Children:              make([]interface{}, 0, len(dtFile.Children)),
		ParsingErrors:         make([]*ParsingError, 0, len(dtFile.ParsingErrors)),
	}
	for _, child := range dtFile.Children {
		node := toNode(child)
		f.Children = append(f.Children, node)
	}
	for _, parsingError := range dtFile.ParsingErrors {
		f.ParsingErrors = append(f.ParsingErrors, &ParsingError{
			Location: []int{parsingError.Location.Line, parsingError.Location.Column},
			Message:  parsingError.Message,
		})
	}
	return f
}

func toNode(node smgo.Node) interface{} {
	switch n := node.(type) {
	case *smgo.Terminal:
		return &Terminal{
			Type: toType(n.Type),
			Name: n.Name,
			LocationSpan: map[string][]int{
				"start": {n.LocationSpan.Start.Line, n.LocationSpan.Start.Column},
				"end":   {n.LocationSpan.End.Line, n.LocationSpan.End.Column},
			},
			Span: []int{n.Span.Start, n.Span.End},
		}
	case *smgo.Container:
		c := &Container{
			Type: toType(n.Type),
			Name: n.Name,
			LocationSpan: map[string][]int{
				"start": {n.LocationSpan.Start.Line, n.LocationSpan.Start.Column},
				"end":   {n.LocationSpan.End.Line, n.LocationSpan.End.Column},
			},
			HeaderSpan: []int{n.HeaderSpan.Start, n.HeaderSpan.End},
			FooterSpan: []int{n.FooterSpan.Start, n.FooterSpan.End},
			Children:   make([]interface{}, 0, len(n.Children)),
		}
		for _, child := range n.Children {
			childNode := toNode(child)
			c.Children = append(c.Children, childNode)
		}
		return c
	default:
		panic("unknown node type")
	}
}

func toType(t smgo.NodeType) string {
	switch t {
	case smgo.PackageNode:
		return "Package"
	case smgo.FunctionNode:
		return "Function"
	case smgo.FieldNode:
		return "Field"
	case smgo.ImportNode:
		return "Import"
	case smgo.ConstNode:
		return "Constant"
	case smgo.VarNode:
		return "Variable"
	case smgo.TypeNode:
		return "Type"
	case smgo.StructNode:
		return "Struct"
	case smgo.InterfaceNode:
		return "Interface"
	default:
		return "Unknown"
	}
}
