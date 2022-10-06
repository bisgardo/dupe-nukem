package match

import (
	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func Test__testdata_match(t *testing.T) {
	y_a := scan.NewFile("a", 2, 620331299357648818)
	y_b := scan.NewFile("b", 2, 620331299357648818)
	y_c := scan.NewFile("c", 2, 617474768148124315)

	y2 := &Dir{
		parent: nil,
		scanDir: &scan.Dir{
			Name:  "y",
			Files: []*scan.File{y_a, y_b, y_c},
		},
	}

	scanX, err := scan.Run("testdata/x", scan.NoSkip, nil)
	require.NoError(t, err)
	scanY, err := scan.Run("testdata/y", scan.NoSkip, nil)
	require.NoError(t, err)

	indexY := BuildIndex(scanY, nil)

	want := Matches{
		620331299357648818: []*File{
			NewFile(y2, y_a),
			NewFile(y2, y_b),
		},
	}
	matches := BuildMatches(scanX, indexY)
	assert.Equal(t, want, matches)
}
