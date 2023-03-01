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

	res := BuildMatch(scanX, nil)
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
		key{size: 2, hash: 620331299357648818}: []Match{
			{
				TargetIndex: 0,
				File:        NewFile(y, testdata_y_a),
			},
			{
				TargetIndex: 0,
				File:        NewFile(y, testdata_y_b),
			},
		},
	}
	res := BuildMatch(scanX, []Target{{ID: TargetID{ID: ""}, Index: indexY}})
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
		key{
			size: 2,
			hash: 620331299357648818,
		}: []Match{
			{
				TargetIndex: 0,
				File:        NewFile(x, testdata_x_a),
			},
		},
	}
	res := BuildMatch(scanY, []Target{{ID: TargetID{ID: ""}, Index: indexX}})
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
		key{
			size: 2,
			hash: 620331299357648818,
		}: []Match{
			{
				TargetIndex: 0,
				File:        NewFile(x, testdata_x_a),
			},
		},
		key{
			size: 2,
			hash: 623218616892763229,
		}: []Match{
			{
				TargetIndex: 0,
				File:        NewFile(x, testdata_x_b),
			},
		},
		key{
			size: 2,
			hash: 622257643729896040,
		}: []Match{
			{
				TargetIndex: 0,
				File:        NewFile(x, testdata_x_c),
			},
		},
	}
	res := BuildMatch(scanX, []Target{{ID: TargetID{ID: ""}, Index: indexX}})
	assert.Equal(t, want, res)
}
