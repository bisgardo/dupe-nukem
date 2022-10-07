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

func Test__testdata_match_single_target(t *testing.T) {
	srcRoot := "testdata/x"
	targetRoot := "testdata/y"

	scanX, err := scan.Run(srcRoot, scan.NoSkip, nil)
	require.NoError(t, err)
	scanY, err := scan.Run(targetRoot, scan.NoSkip, nil)
	require.NoError(t, err)

	indexY := BuildIndex(scanY)

	// Have to dig out files from index because comparison of maps with pointer keys relies on pointer identity.
	want := Matches{
		620331299357648818: FileSet{
			indexY[620331299357648818][0]: struct{}{},
			indexY[620331299357648818][1]: struct{}{},
		},
	}
	res := BuildMatches(scanX, []Index{indexY})
	assert.Equal(t, want, res)
}

func Test__testdata_match_single_target_reversed(t *testing.T) {
	srcRoot := "testdata/y"
	targetRoot := "testdata/x"

	scanY, err := scan.Run(srcRoot, scan.NoSkip, nil)
	require.NoError(t, err)
	scanX, err := scan.Run(targetRoot, scan.NoSkip, nil)
	require.NoError(t, err)

	indexX := BuildIndex(scanX)

	// Note that there's only a single match for files that are duplicated in the source ('y/a' and 'y/b' in this case).
	// This is (at least partially) the reason why we build the match mapping on hashes instead of individual files.
	want := Matches{
		620331299357648818: FileSet{
			indexX[620331299357648818][0]: struct{}{},
		},
	}
	res := BuildMatches(scanY, []Index{indexX})
	assert.Equal(t, want, res)
}

func Test__testdata_match_self(t *testing.T) {
	root := "testdata/x"

	scanX, err := scan.Run(root, scan.NoSkip, nil)
	require.NoError(t, err)

	indexX := BuildIndex(scanX)

	want := Matches{
		620331299357648818: FileSet{
			indexX[620331299357648818][0]: struct{}{},
		},
		623218616892763229: FileSet{
			indexX[623218616892763229][0]: struct{}{},
		},
		622257643729896040: FileSet{
			indexX[622257643729896040][0]: struct{}{},
		},
	}
	res := BuildMatches(scanX, []Index{indexX})
	assert.Equal(t, want, res)
}
