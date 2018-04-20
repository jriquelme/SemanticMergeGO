package smgo_test

import (
	"os"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jriquelme/SemanticMergeGO/smgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimpleCases(t *testing.T) {
	t.Parallel()
	if testing.Verbose() {
		smgo.PrintBlocks = true
	}

	simpleCases := []struct {
		Src          string
		ExpectedFile *smgo.File
	}{
		{
			Src: "simple_const.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 5, 25),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simpleconst",
						LocationSpan: newLocationSpan(1, 0, 1, 20),
						Span:         smgo.RuneSpan{0, 19},
					},
					&smgo.Terminal{
						Type:         smgo.ConstNode,
						Name:         "N",
						LocationSpan: newLocationSpan(2, 0, 3, 12),
						Span:         smgo.RuneSpan{20, 32},
					},
					&smgo.Terminal{
						Type:         smgo.ConstNode,
						Name:         "Name",
						LocationSpan: newLocationSpan(4, 0, 5, 25),
						Span:         smgo.RuneSpan{33, 58},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "simple_footer.go_src",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 8, 1),
				FooterSpan:   smgo.RuneSpan{51, 53},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simplefooter",
						LocationSpan: newLocationSpan(1, 0, 1, 21),
						Span:         smgo.RuneSpan{0, 20},
					},
					&smgo.Terminal{
						Type:         smgo.FunctionNode,
						Name:         "asdf",
						LocationSpan: newLocationSpan(2, 0, 5, 2),
						Span:         smgo.RuneSpan{21, 50},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "simple_func.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 5, 2),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simplefunc",
						LocationSpan: newLocationSpan(1, 0, 1, 19),
						Span:         smgo.RuneSpan{0, 18},
					},
					&smgo.Terminal{
						Type:         smgo.FunctionNode,
						Name:         "Hi",
						LocationSpan: newLocationSpan(2, 0, 5, 2),
						Span:         smgo.RuneSpan{19, 47},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "simple_import.go_src",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 3, 13),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simpleimport",
						LocationSpan: newLocationSpan(1, 0, 1, 21),
						Span:         smgo.RuneSpan{0, 20},
					},
					&smgo.Terminal{
						Type:         smgo.ImportNode,
						Name:         "fmt",
						LocationSpan: newLocationSpan(2, 0, 3, 13),
						Span:         smgo.RuneSpan{21, 34},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "simple_interface.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 5, 2),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simpleinterface",
						LocationSpan: newLocationSpan(1, 0, 1, 24),
						Span:         smgo.RuneSpan{0, 23},
					},
					&smgo.Container{
						Type:         smgo.InterfaceNode,
						Name:         "Figure",
						LocationSpan: newLocationSpan(2, 0, 5, 2),
						HeaderSpan:   smgo.RuneSpan{24, 48},
						FooterSpan:   smgo.RuneSpan{65, 66},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.FieldNode,
								Name:         "Area",
								LocationSpan: newLocationSpan(4, 0, 4, 16),
								Span:         smgo.RuneSpan{49, 64},
							},
						},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "simple_struct.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 9, 2),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simplestruct",
						LocationSpan: newLocationSpan(1, 0, 1, 21),
						Span:         smgo.RuneSpan{0, 20},
					},
					&smgo.Container{
						Type:         smgo.StructNode,
						Name:         "Person",
						LocationSpan: newLocationSpan(2, 0, 5, 2),
						HeaderSpan:   smgo.RuneSpan{21, 42},
						FooterSpan:   smgo.RuneSpan{56, 57},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.FieldNode,
								Name:         "Name",
								LocationSpan: newLocationSpan(4, 0, 4, 13),
								Span:         smgo.RuneSpan{43, 55},
							},
						},
					},
					&smgo.Terminal{
						Type:         smgo.FunctionNode,
						Name:         "SayHi",
						LocationSpan: newLocationSpan(6, 0, 9, 2),
						Span:         smgo.RuneSpan{58, 115},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "simple_types.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 21, 22),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simpletypes",
						LocationSpan: newLocationSpan(1, 0, 1, 20),
						Span:         smgo.RuneSpan{0, 19},
					},
					&smgo.Terminal{
						Type:         smgo.ImportNode,
						Name:         "io",
						LocationSpan: newLocationSpan(2, 0, 3, 12),
						Span:         smgo.RuneSpan{20, 32},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "String",
						LocationSpan: newLocationSpan(4, 0, 5, 19),
						Span:         smgo.RuneSpan{33, 52},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "StringAlias",
						LocationSpan: newLocationSpan(6, 0, 7, 26),
						Span:         smgo.RuneSpan{53, 79},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "Map",
						LocationSpan: newLocationSpan(8, 0, 9, 25),
						Span:         smgo.RuneSpan{80, 105},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "Array",
						LocationSpan: newLocationSpan(10, 0, 11, 18),
						Span:         smgo.RuneSpan{106, 124},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "Chan",
						LocationSpan: newLocationSpan(12, 0, 13, 21),
						Span:         smgo.RuneSpan{125, 146},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "Func",
						LocationSpan: newLocationSpan(14, 0, 15, 23),
						Span:         smgo.RuneSpan{147, 170},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "IntPointer",
						LocationSpan: newLocationSpan(16, 0, 17, 21),
						Span:         smgo.RuneSpan{171, 192},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "RedundantPar",
						LocationSpan: newLocationSpan(18, 0, 19, 25),
						Span:         smgo.RuneSpan{193, 218},
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "Reader",
						LocationSpan: newLocationSpan(20, 0, 21, 22),
						Span:         smgo.RuneSpan{219, 241},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "simple_var.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 5, 21),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "simplevar",
						LocationSpan: newLocationSpan(1, 0, 1, 18),
						Span:         smgo.RuneSpan{0, 17},
					},
					&smgo.Terminal{
						Type:         smgo.VarNode,
						Name:         "X",
						LocationSpan: newLocationSpan(2, 0, 3, 10),
						Span:         smgo.RuneSpan{18, 28},
					},
					&smgo.Terminal{
						Type:         smgo.VarNode,
						Name:         "Z",
						LocationSpan: newLocationSpan(4, 0, 5, 21),
						Span:         smgo.RuneSpan{29, 50},
					},
				},
				ParsingErrors: nil,
			},
		},
	}
	for _, simpleCase := range simpleCases {
		name := simpleCase.Src[len("simple_"):strings.LastIndex(simpleCase.Src, ".")]
		t.Run(name, func(t *testing.T) {
			srcFile, err := os.Open("testdata/" + simpleCase.Src)
			require.Nil(t, err)
			defer srcFile.Close()

			file, err := smgo.Parse(srcFile, "UTF-8")
			assert.NotNil(t, file)
			assert.Nil(t, err)

			assert.Equal(t, simpleCase.ExpectedFile, file)
			if t.Failed() {
				spew.Dump(t.Name(), file)
			}
		})
	}
}
