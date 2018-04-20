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
