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

func TestParseCommentCases(t *testing.T) {
	t.Parallel()
	if testing.Verbose() {
		smgo.PrintBlocks = true
	}

	cases := []struct {
		Src          string
		ExpectedFile *smgo.File
	}{
		{
			Src: "comment_const.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 16, 12),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "commentconst",
						LocationSpan: newLocationSpan(1, 0, 1, 21),
						Span:         smgo.RuneSpan{0, 20},
					},
					&smgo.Container{
						Type:         smgo.ConstNode,
						Name:         "const",
						LocationSpan: newLocationSpan(2, 0, 9, 2),
						HeaderSpan:   smgo.RuneSpan{21, 50},
						FooterSpan:   smgo.RuneSpan{92, 93},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.ConstNode,
								Name:         "N",
								LocationSpan: newLocationSpan(5, 0, 5, 7),
								Span:         smgo.RuneSpan{51, 57},
							},
							&smgo.Terminal{
								Type:         smgo.ConstNode,
								Name:         "Name",
								LocationSpan: newLocationSpan(6, 0, 8, 20),
								Span:         smgo.RuneSpan{58, 91},
							},
						},
					},
					&smgo.Container{
						Type:         smgo.ConstNode,
						Name:         "const",
						LocationSpan: newLocationSpan(10, 0, 12, 9),
						HeaderSpan:   smgo.RuneSpan{94, 116},
						FooterSpan:   smgo.RuneSpan{117, 118},
						Children:     nil,
					},
					&smgo.Terminal{
						Type:         smgo.ConstNode,
						Name:         "X",
						LocationSpan: newLocationSpan(13, 0, 16, 12),
						Span:         smgo.RuneSpan{119, 158},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "comment_import.go_src",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 21, 17),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "commentimport",
						LocationSpan: newLocationSpan(1, 0, 1, 22),
						Span:         smgo.RuneSpan{0, 21},
					},
					&smgo.Container{
						Type:         smgo.ImportNode,
						Name:         "import",
						LocationSpan: newLocationSpan(2, 0, 13, 2),
						HeaderSpan:   smgo.RuneSpan{22, 63},
						FooterSpan:   smgo.RuneSpan{147, 148},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.ImportNode,
								Name:         "io",
								LocationSpan: newLocationSpan(6, 0, 7, 6),
								Span:         smgo.RuneSpan{64, 76},
							},
							&smgo.Terminal{
								Type:         smgo.ImportNode,
								Name:         "io/ioutil",
								LocationSpan: newLocationSpan(8, 0, 8, 13),
								Span:         smgo.RuneSpan{77, 89},
							},
							&smgo.Terminal{
								Type:         smgo.ImportNode,
								Name:         "github.com/pkg/errors",
								LocationSpan: newLocationSpan(9, 0, 12, 25),
								Span:         smgo.RuneSpan{90, 146},
							},
						},
					},
					&smgo.Container{
						Type:         smgo.ImportNode,
						Name:         "import",
						LocationSpan: newLocationSpan(14, 0, 16, 10),
						HeaderSpan:   smgo.RuneSpan{149, 179},
						FooterSpan:   smgo.RuneSpan{180, 181},
						Children:     nil,
					},
					&smgo.Terminal{
						Type:         smgo.ImportNode,
						Name:         "fmt",
						LocationSpan: newLocationSpan(17, 0, 18, 13),
						Span:         smgo.RuneSpan{182, 195},
					},
					&smgo.Terminal{
						Type:         smgo.ImportNode,
						Name:         "strings",
						LocationSpan: newLocationSpan(19, 0, 21, 17),
						Span:         smgo.RuneSpan{196, 228},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "comment_pkg.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 10, 12),
				FooterSpan:   smgo.RuneSpan{90, 111},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.Comment,
						Name:         "commentpkg...",
						LocationSpan: newLocationSpan(1, 0, 2, 15),
						Span:         smgo.RuneSpan{0, 34},
					},
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "commentpkg",
						LocationSpan: newLocationSpan(3, 0, 5, 19),
						Span:         smgo.RuneSpan{35, 69},
					},
					&smgo.Terminal{
						Type:         smgo.Comment,
						Name:         "another co...",
						LocationSpan: newLocationSpan(6, 0, 7, 19),
						Span:         smgo.RuneSpan{70, 89},
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "comment_type.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 50, 2),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "commenttype",
						LocationSpan: newLocationSpan(1, 0, 1, 20),
						Span:         smgo.RuneSpan{0, 19},
					},
					&smgo.Container{
						Type:         smgo.TypeNode,
						Name:         "type",
						LocationSpan: newLocationSpan(2, 0, 35, 13),
						HeaderSpan:   smgo.RuneSpan{20, 51},
						FooterSpan:   smgo.RuneSpan{449, 488},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.TypeNode,
								Name:         "String",
								LocationSpan: newLocationSpan(5, 0, 6, 23),
								Span:         smgo.RuneSpan{52, 90},
							},
							&smgo.Terminal{
								Type:         smgo.TypeNode,
								Name:         "StringAlias",
								LocationSpan: newLocationSpan(7, 0, 9, 25),
								Span:         smgo.RuneSpan{91, 129},
							},
							&smgo.Terminal{
								Type:         smgo.TypeNode,
								Name:         "Map",
								LocationSpan: newLocationSpan(10, 0, 11, 21),
								Span:         smgo.RuneSpan{130, 158},
							},
							&smgo.Terminal{
								Type:         smgo.TypeNode,
								Name:         "Array",
								LocationSpan: newLocationSpan(12, 0, 13, 14),
								Span:         smgo.RuneSpan{159, 182},
							},
							&smgo.Container{
								Type:         smgo.StructNode,
								Name:         "Person",
								LocationSpan: newLocationSpan(14, 0, 21, 14),
								HeaderSpan:   smgo.RuneSpan{183, 218},
								FooterSpan:   smgo.RuneSpan{262, 275},
								Children: []smgo.Node{
									&smgo.Terminal{
										Type:         smgo.FieldNode,
										Name:         "Name",
										LocationSpan: newLocationSpan(17, 0, 17, 14),
										Span:         smgo.RuneSpan{219, 232},
									},
									&smgo.Terminal{
										Type:         smgo.FieldNode,
										Name:         "Age",
										LocationSpan: newLocationSpan(18, 0, 20, 19),
										Span:         smgo.RuneSpan{233, 261},
									},
								},
							},
							&smgo.Container{
								Type:         smgo.InterfaceNode,
								Name:         "Figure",
								LocationSpan: newLocationSpan(22, 0, 31, 14),
								HeaderSpan:   smgo.RuneSpan{276, 335},
								FooterSpan:   smgo.RuneSpan{414, 448},
								Children: []smgo.Node{
									&smgo.Terminal{
										Type:         smgo.FieldNode,
										Name:         "Area",
										LocationSpan: newLocationSpan(25, 0, 26, 24),
										Span:         smgo.RuneSpan{336, 369},
									},
									&smgo.Terminal{
										Type:         smgo.FieldNode,
										Name:         "Perimeter",
										LocationSpan: newLocationSpan(27, 0, 28, 29),
										Span:         smgo.RuneSpan{370, 413},
									},
								},
							},
						},
					},
					&smgo.Container{
						Type:         smgo.TypeNode,
						Name:         "type",
						LocationSpan: newLocationSpan(36, 0, 38, 24),
						HeaderSpan:   smgo.RuneSpan{489, 515},
						FooterSpan:   smgo.RuneSpan{516, 533},
						Children:     nil,
					},
					&smgo.Terminal{
						Type:         smgo.TypeNode,
						Name:         "Chan",
						LocationSpan: newLocationSpan(39, 0, 41, 30),
						Span:         smgo.RuneSpan{534, 580},
					},
					&smgo.Container{
						Type:         smgo.StructNode,
						Name:         "AnotherStruct",
						LocationSpan: newLocationSpan(42, 0, 50, 2),
						HeaderSpan:   smgo.RuneSpan{581, 627},
						FooterSpan:   smgo.RuneSpan{706, 707},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.FieldNode,
								Name:         "Func",
								LocationSpan: newLocationSpan(45, 0, 46, 29),
								Span:         smgo.RuneSpan{628, 665},
							},
							&smgo.Terminal{
								Type:         smgo.FieldNode,
								Name:         "IntPointer",
								LocationSpan: newLocationSpan(47, 0, 49, 27),
								Span:         smgo.RuneSpan{666, 705},
							},
						},
					},
				},
				ParsingErrors: nil,
			},
		},
	}
	for _, testCase := range cases {
		name := testCase.Src[len("comment_"):strings.LastIndex(testCase.Src, ".")]
		t.Run(name, func(t *testing.T) {
			srcFile, err := os.Open("testdata/" + testCase.Src)
			require.Nil(t, err)
			defer srcFile.Close()

			file, err := smgo.Parse(srcFile, "UTF-8")
			assert.NotNil(t, file)
			assert.Nil(t, err)

			assert.Equal(t, testCase.ExpectedFile, file)
			if t.Failed() {
				spew.Dump(t.Name(), file)
			}
		})
	}

}
