package match

import (
	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func Test__testdata_match_no_targets_is_empty(t *testing.T) {
	srcRoot := "testdata/x"

	scanX, err := scan.Run(srcRoot, scan.NoSkip, nil)
	require.NoError(t, err)

	res := BuildMatches(scanX, []Index{})
	assert.Empty(t, res)
}

func Test__testdata_match_single_target_not_self(t *testing.T) {
	srcRoot := "testdata/x"
	targetRoot := "testdata/y"

	y := &Dir{
		Parent: nil,
		ScanDir: &scan.Dir{
			Name:  "y",
			Files: []*scan.File{testdata_y_a, testdata_y_b, testdata_y_c},
		},
	}

	scanX, err := scan.Run(srcRoot, scan.NoSkip, nil)
	require.NoError(t, err)
	scanY, err := scan.Run(targetRoot, scan.NoSkip, nil)
	require.NoError(t, err)

	indexY := BuildIndex(scanY)

	want := Matches{
		620331299357648818: []*File{
			NewFile(y, testdata_y_a),
			NewFile(y, testdata_y_b),
		},
	}
	res := BuildMatches(scanX, []Index{indexY})
	assert.Equal(t, want, res)
}

func Test__testdata_match_self_target(t *testing.T) {
	root := "testdata/x"

	x := &Dir{
		Parent: nil,
		ScanDir: &scan.Dir{
			Name:  "x",
			Files: []*scan.File{testdata_x_a, testdata_x_b, testdata_x_c},
		},
	}

	scanX, err := scan.Run(root, scan.NoSkip, nil)
	require.NoError(t, err)

	indexX := BuildIndex(scanX)

	want := Matches{
		620331299357648818: []*File{
			NewFile(x, testdata_x_a),
		},
		623218616892763229: []*File{
			NewFile(x, testdata_x_b),
		},
		622257643729896040: []*File{
			NewFile(x, testdata_x_c),
		},
	}
	res := BuildMatches(scanX, []Index{indexX})
	assert.Equal(t, want, res)
}
