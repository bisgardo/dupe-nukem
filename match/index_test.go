package match

import (
	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func Test__testdata_index(t *testing.T) {
	root := "testdata"

	x_a := scan.NewFile("a", 2, 620331299357648818)
	x_b := scan.NewFile("b", 2, 623218616892763229)
	x_c := scan.NewFile("c", 2, 622257643729896040)
	y_a := scan.NewFile("a", 2, 620331299357648818)
	y_b := scan.NewFile("b", 2, 620331299357648818)
	y_c := scan.NewFile("c", 2, 617474768148124315)

	x := &scan.Dir{
		Name:  "x",
		Files: []*scan.File{x_a, x_b, x_c},
	}
	y := &scan.Dir{
		Name:  "y",
		Files: []*scan.File{y_a, y_b, y_c},
	}
	testdata := &Dir{
		parent: nil,
		scanDir: &scan.Dir{
			Name: "testdata",
			Dirs: []*scan.Dir{x, y},
		},
	}

	x2 := &Dir{
		parent:  testdata,
		scanDir: x,
	}
	y2 := &Dir{
		parent:  testdata,
		scanDir: y,
	}

	want := Index{
		620331299357648818: []*File{
			NewFile(x2, x_a),
			NewFile(y2, y_a),
			NewFile(y2, y_b),
		},
		623218616892763229: []*File{
			NewFile(x2, x_b),
		},
		622257643729896040: []*File{
			NewFile(x2, x_c),
		},
		617474768148124315: []*File{
			NewFile(y2, y_c),
		},
	}
	scanRoot, err := scan.Run(root, scan.NoSkip, nil)
	require.NoError(t, err)

	res := BuildIndex(scanRoot, nil)
	assert.Equal(t, want, res)
}
