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
	dir, err := ioutil.TempDir("", "empty")
	require.NoError(t, err)
	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	want := &Dir{Name: filepath.Base(dir)}

	tests := []struct {
		name     string
		skipFunc ShouldSkipPath
	}{
		{name: "without skip", skipFunc: NoSkip},
		{name: "skipping root", skipFunc: skip(dir)},
		{name: "skipping non-existing", skipFunc: skip("non-existing")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := Run(dir, test.skipFunc, nil)
			require.NoError(t, err)
			assert.Equal(t, want, res)
		})
	}
}

func Test__nonexistent_dir_fails(t *testing.T) {
	t.Run("without trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent", NoSkip, nil)
		require.EqualError(t, err, "cannot scan root directory \"nonexistent\": file or directory \"nonexistent\" does not exist")
	})
	t.Run("with trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent/", NoSkip, nil)
		require.EqualError(t, err, "cannot scan root directory \"nonexistent/\": file or directory \"nonexistent/\" does not exist")
	})
}

//goland:noinspection GoSnakeCaseUsage
var (
	testdata_b_d   = &File{Name: "d", Size: 2, Hash: 644258871406045975} // contents: "x"
	testdata_e_f_a = &File{Name: "a", Size: 2, Hash: 646158827499216133} // contents: "z"
	testdata_a     = &File{Name: "a", Size: 2, Hash: 644258871406045975} // contents: "x"
	testdata_c     = &File{Name: "c", Size: 2, Hash: 643306694336204474} // contents: "y"
)

func Test__testdata_no_skip(t *testing.T) {
	root := "testdata"
	want := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name:  "b",
				Files: []*File{testdata_b_d},
			},
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name:       "f",
						Files:      []*File{testdata_e_f_a},
						EmptyFiles: []string{"g"},
					},
				},
			},
		},
		Files: []*File{testdata_a, testdata_c},
	}
	// Working dir for tests is the containing folder.
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_root(t *testing.T) {
	root := "testdata"
	want := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name:  "b",
				Files: []*File{testdata_b_d},
			},
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name:       "f",
						Files:      []*File{testdata_e_f_a},
						EmptyFiles: []string{"g"},
					},
				},
			},
		},
		Files: []*File{testdata_a, testdata_c},
	}
	// Working dir for tests is the containing folder.
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_dir_without_subdirs(t *testing.T) {
	root := "testdata"
	want := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name:       "f",
						Files:      []*File{testdata_e_f_a},
						EmptyFiles: []string{"g"},
					},
				},
			},
		},
		Files:       []*File{testdata_a, testdata_c},
		SkippedDirs: []string{"b"},
	}
	// Working dir for tests is the containing folder.
	res, err := Run(root, skip("b"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_dir_with_subdirs(t *testing.T) {
	root := "testdata"
	want := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name:  "b",
				Files: []*File{testdata_b_d},
			},
		},
		Files:       []*File{testdata_a, testdata_c},
		SkippedDirs: []string{"e"},
	}
	// Working dir for tests is the containing folder.
	res, err := Run(root, skip("e"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_nonempty_file(t *testing.T) {
	root := "testdata"
	want := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name:  "b",
				Files: []*File{testdata_b_d},
			},
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name:         "f",
						EmptyFiles:   []string{"g"},
						SkippedFiles: []string{"a"},
					},
				},
			},
		},
		Files:        []*File{testdata_c},
		SkippedFiles: []string{"a"},
	}
	// Working dir for tests is the containing folder.
	res, err := Run(root, skip("a"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_subdir_skip_empty_file(t *testing.T) {
	root := "testdata/e/f"
	want := &Dir{
		Name:         "f",
		Files:        []*File{testdata_e_f_a},
		SkippedFiles: []string{"g"},
	}
	// Working dir for tests is the containing folder.
	res, err := Run(root, skip("g"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_dir_with_trailing_slash_panics(t *testing.T) {
	// Doesn't happen in practice because cmd passes dir through filepath.Clean.
	require.Panics(t, func() {
		_, _ = Run("testdata/", NoSkip, nil)
	})
}

func Test__testdata_cache_with_mismatching_root_fails(t *testing.T) {
	root := "testdata"
	cache := &Dir{
		Name: "not-testdata",
	}
	_, err := Run(root, skip("a"), cache)
	assert.EqualError(t, err, "cache of dir \"not-testdata\" cannot be used with root dir \"testdata\"")
}

func Test__testdata_with_hashes_from_cache(t *testing.T) {
	root := "testdata"
	cache := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name: "f",
						Files: []*File{
							{Name: "a", Size: 2, Hash: 42}, // used
						},
					},
				},
				Files: []*File{
					{Name: "d", Size: 2, Hash: 66}, // not used
				},
			},
		},
		Files: []*File{
			{Name: "c", Size: 2, Hash: 53}, // used
			{Name: "d", Size: 2, Hash: 66}, // not used
		},
	}
	want := &Dir{
		Name: "testdata",
		Dirs: []*Dir{
			{
				Name:  "b",
				Files: []*File{testdata_b_d}, // not cached
			},
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name: "f",
						Files: []*File{
							{Name: "a", Size: 2, Hash: 42}, // cached
						},
						EmptyFiles: []string{"g"},
					},
				},
			},
		},
		Files: []*File{
			testdata_a,                     // not cached
			{Name: "c", Size: 2, Hash: 53}, // cached
		},
	}
	res, err := Run(root, NoSkip, cache)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_subdir_cache_not_used_for_different_file_size(t *testing.T) {
	root := "testdata/b"
	cache := &Dir{
		Name: "b",
		Files: []*File{
			{
				Name: "d",
				Size: 1,  // size of "testdata/b/d" is 2
				Hash: 21, // so the cached hash value is not read
			},
		},
	}
	want := &Dir{
		Name:  "b",
		Files: []*File{testdata_b_d},
	}
	res, err := Run(root, NoSkip, cache)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

/* UTILITIES */

func skip(names ...string) ShouldSkipPath {
	return func(dir, name string) bool {
		for _, n := range names {
			if n == name {
				return true
			}
		}
		return false
	}
}
