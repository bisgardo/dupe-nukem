package match

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/scan"
)

//goland:noinspection GoSnakeCaseUsage
var (
	testdata_x_a = scan.NewFile("a", 2, 620331299357648818) // contents: "a"
	testdata_x_b = scan.NewFile("b", 2, 623218616892763229) // contents: "b"
	testdata_x_c = scan.NewFile("c", 2, 622257643729896040) // contents: "c"
	testdata_y_a = scan.NewFile("a", 2, 620331299357648818) // contents: "a"
	testdata_y_b = scan.NewFile("b", 2, 620331299357648818) // contents: "a"
	testdata_y_c = scan.NewFile("c", 2, 617474768148124315) // contents: "d"
)

func Test__testdata_index(t *testing.T) {
	root := "testdata"

	scanX := &scan.Dir{
		Name:  "x",
		Files: []*scan.File{testdata_x_a, testdata_x_b, testdata_x_c},
	}
	scanY := &scan.Dir{
		Name:  "y",
		Files: []*scan.File{testdata_y_a, testdata_y_b, testdata_y_c},
	}

	testdata := &Dir{
		Parent: nil,
		ScanDir: &scan.Dir{
			Name: "testdata",
			Dirs: []*scan.Dir{scanX, scanY},
		},
	}
	x := &Dir{
		Parent:  testdata,
		ScanDir: scanX,
	}
	y := &Dir{
		Parent:  testdata,
		ScanDir: scanY,
	}

	want := Index{
		620331299357648818: []*File{
			NewFile(x, testdata_x_a),
			NewFile(y, testdata_y_a),
			NewFile(y, testdata_y_b),
		},
		623218616892763229: []*File{
			NewFile(x, testdata_x_b),
		},
		622257643729896040: []*File{
			NewFile(x, testdata_x_c),
		},
		617474768148124315: []*File{
			NewFile(y, testdata_y_c),
		},
	}
	scanRoot, err := scan.Run(root, scan.NoSkip, nil)
	require.NoError(t, err)

	res := BuildIndex(scanRoot)
	assert.Equal(t, want, res)
}
