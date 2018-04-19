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

func TestParseGroupedConst(t *testing.T) {
	t.Parallel()
	if testing.Verbose() {
		smgo.PrintBlocks = true
	}

	cases := []struct {
		Src          string
		ExpectedFile *smgo.File
	}{
		{
			Src: "grouped_const.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 8, 9),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "groupedconst",
						LocationSpan: newLocationSpan(1, 0, 1, 21),
						Span:         smgo.RuneSpan{0, 20},
					},
					&smgo.Container{
						Type:         smgo.ConstNode,
						Name:         "const",
						LocationSpan: newLocationSpan(2, 0, 6, 2),
						HeaderSpan:   smgo.RuneSpan{21, 29},
						FooterSpan:   smgo.RuneSpan{67, 68},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.ConstNode,
								Name:         "N",
								LocationSpan: newLocationSpan(4, 0, 4, 17),
								Span:         smgo.RuneSpan{30, 46},
							},
							&smgo.Terminal{
								Type:         smgo.ConstNode,
								Name:         "Name",
								LocationSpan: newLocationSpan(5, 0, 5, 20),
								Span:         smgo.RuneSpan{47, 66},
							},
						},
					},
					&smgo.Container{
						Type:         smgo.ConstNode,
						Name:         "const",
						LocationSpan: newLocationSpan(7, 0, 8, 9),
						HeaderSpan:   smgo.RuneSpan{69, 76},
						FooterSpan:   smgo.RuneSpan{77, 78},
						Children:     nil,
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "grouped_import.go_src",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 10, 10),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "groupedimport",
						LocationSpan: newLocationSpan(1, 0, 1, 22),
						Span:         smgo.RuneSpan{0, 21},
					},
					&smgo.Container{
						Type:         smgo.ImportNode,
						Name:         "import",
						LocationSpan: newLocationSpan(2, 0, 8, 2),
						HeaderSpan:   smgo.RuneSpan{22, 31},
						FooterSpan:   smgo.RuneSpan{77, 78},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.ImportNode,
								Name:         "io",
								LocationSpan: newLocationSpan(4, 0, 4, 6),
								Span:         smgo.RuneSpan{32, 37},
							},
							&smgo.Terminal{
								Type:         smgo.ImportNode,
								Name:         "io/ioutil",
								LocationSpan: newLocationSpan(5, 0, 5, 13),
								Span:         smgo.RuneSpan{38, 50},
							},
							&smgo.Terminal{
								Type:         smgo.ImportNode,
								Name:         "github.com/pkg/errors",
								LocationSpan: newLocationSpan(6, 0, 7, 25),
								Span:         smgo.RuneSpan{51, 76},
							},
						},
					},
					&smgo.Container{
						Type:         smgo.ImportNode,
						Name:         "import",
						LocationSpan: newLocationSpan(9, 0, 10, 10),
						HeaderSpan:   smgo.RuneSpan{79, 87},
						FooterSpan:   smgo.RuneSpan{88, 89},
						Children:     nil,
					},
				},
				ParsingErrors: nil,
			},
		},
		{
			Src: "grouped_var.go",
			ExpectedFile: &smgo.File{
				LocationSpan: newLocationSpan(1, 0, 9, 7),
				FooterSpan:   smgo.RuneSpan{0, -1},
				Children: []smgo.Node{
					&smgo.Terminal{
						Type:         smgo.PackageNode,
						Name:         "groupedvar",
						LocationSpan: newLocationSpan(1, 0, 1, 19),
						Span:         smgo.RuneSpan{0, 18},
					},
					&smgo.Container{
						Type:         smgo.VarNode,
						Name:         "var",
						LocationSpan: newLocationSpan(2, 0, 7, 2),
						HeaderSpan:   smgo.RuneSpan{19, 25},
						FooterSpan:   smgo.RuneSpan{52, 53},
						Children: []smgo.Node{
							&smgo.Terminal{
								Type:         smgo.VarNode,
								Name:         "X",
								LocationSpan: newLocationSpan(4, 0, 4, 7),
								Span:         smgo.RuneSpan{26, 32},
							},
							&smgo.Terminal{
								Type:         smgo.VarNode,
								Name:         "Z",
								LocationSpan: newLocationSpan(5, 0, 6, 18),
								Span:         smgo.RuneSpan{33, 51},
							},
						},
					},
					&smgo.Container{
						Type:         smgo.VarNode,
						Name:         "var",
						LocationSpan: newLocationSpan(8, 0, 9, 7),
						HeaderSpan:   smgo.RuneSpan{54, 59},
						FooterSpan:   smgo.RuneSpan{60, 61},
						Children:     nil,
					},
				},
				ParsingErrors: nil,
			},
		},
	}
	for _, testCase := range cases {
		name := testCase.Src[len("grouped_"):strings.LastIndex(testCase.Src, ".")]
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
