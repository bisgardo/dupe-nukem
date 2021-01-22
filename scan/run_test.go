package scan

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Working dir for tests is the containing folder.

func Test__empty_dir(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	want := &Dir{Name: filepath.Base(dir)}

	res, err := Run(dir)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__nonexistent_dir(t *testing.T) {
	t.Run("without trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent")
		require.EqualError(t, err, "cannot scan root directory \"nonexistent\": file or directory \"nonexistent\" does not exist")
	})
	t.Run("with trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent/")
		require.EqualError(t, err, "cannot scan root directory \"nonexistent/\": file or directory \"nonexistent/\" does not exist")
	})
}

func Test__testdata_dir(t *testing.T) {
	root := "testdata"
	want := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name: "b",
				Dirs: nil,
				Files: []*File{
					{Name: "d", Size: 2},
				},
			},
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name: "f",
						Dirs: nil,
						Files: []*File{
							{Name: "a", Size: 2},
							{Name: "g", Size: 0},
						},
					},
				},
				Files: nil,
			},
		},
		Files: []*File{
			{Name: "a", Size: 2},
			{Name: "c", Size: 2},
		},
	}
	// Working dir for tests is the containing folder.
	res, err := Run(root)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_dir_with_trailing_slash_panics(t *testing.T) {
	// Doesn't happen in practice because cmd passes dir through filepath.Clean.
	require.Panics(t, func() {
		_, _ = Run("testdata/")
	})
}

/* UTILITIES */

func tmpDir(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir("", "empty")
	require.NoError(t, err)
	return dir, func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}
}
