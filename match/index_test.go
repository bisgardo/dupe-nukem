package match

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/scan"
)

func Test__testdata_index(t *testing.T) {
	root := "testdata"

	testdata := &Dir{
		Parent: nil,
		ScanDir: &scan.Dir{
			Name: "testdata",
			Dirs: []*scan.Dir{testdata_x, testdata_y},
		},
	}
	x := &Dir{
		Parent:  testdata,
		ScanDir: testdata_x,
	}
	y := &Dir{
		Parent:  testdata,
		ScanDir: testdata_y,
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
