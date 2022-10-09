package match

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/scan"
)

func Test__testdata_match_no_targets_is_empty(t *testing.T) {
	srcRoot := "testdata/x"

	scanX, err := scan.Run(srcRoot, scan.NoSkip, nil)
	require.NoError(t, err)

	res := BuildMatch(scanX, []Index{})
	assert.Empty(t, res)
}

func Test__testdata_match_single_target(t *testing.T) {
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
	res := BuildMatch(scanX, []Index{indexY})
	assert.Equal(t, want, res)
}

func Test__testdata_match_single_target_reversed(t *testing.T) {
	srcRoot := "testdata/y"
	targetRoot := "testdata/x"

	x := &Dir{
		Parent: nil,
		ScanDir: &scan.Dir{
			Name:  "x",
			Files: []*scan.File{testdata_x_a, testdata_x_b, testdata_x_c},
		},
	}

	scanY, err := scan.Run(srcRoot, scan.NoSkip, nil)
	require.NoError(t, err)
	scanX, err := scan.Run(targetRoot, scan.NoSkip, nil)
	require.NoError(t, err)

	indexX := BuildIndex(scanX)

	// Note that there's only a single match for files that are duplicated in the source ('y/a' and 'y/b' in this case).
	// This is (at least partially) the reason why we build the match mapping on hashes instead of individual files.
	want := Matches{
		620331299357648818: []*File{
			NewFile(x, testdata_x_a),
		},
	}
	res := BuildMatch(scanY, []Index{indexX})
	assert.Equal(t, want, res)
}

func Test__testdata_match_self(t *testing.T) {
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
	res := BuildMatch(scanX, []Index{indexX})
	assert.Equal(t, want, res)
}
