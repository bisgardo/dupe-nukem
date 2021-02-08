package scan

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// INFO Working dir for tests is the directory containing the file.

// TODO Test logged output whenever it's relevant.

func Test__empty_dir(t *testing.T) {
	dir, err := ioutil.TempDir("", "empty")
	require.NoError(t, err)
	defer func() {
		err := os.RemoveAll(dir)
		assert.NoError(t, err)
	}()

	want := &Dir{Name: filepath.Base(dir)}

	tests := []struct {
		name       string
		shouldSkip ShouldSkipPath
	}{
		{name: "without skip", shouldSkip: NoSkip},
		{name: "skipping root", shouldSkip: skip(dir)},
		{name: "skipping non-existing", shouldSkip: skip("non-existing")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := Run(dir, test.shouldSkip, nil)
			require.NoError(t, err)
			assert.Equal(t, want, res)
		})
	}
}

func Test__nonexistent_dir_fails(t *testing.T) {
	t.Run("without trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent", NoSkip, nil)
		require.EqualError(t, err, `invalid root directory "nonexistent": not found`)
	})
	t.Run("with trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent/", NoSkip, nil)
		require.EqualError(t, err, `invalid root directory "nonexistent/": not found`)
	})
}

func Test__file_root_fails(t *testing.T) {
	f, err := ioutil.TempFile("", "root")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	_, err = Run(f.Name(), NoSkip, nil)
	assert.EqualError(t, err, fmt.Sprintf("invalid root directory %q: not a directory", f.Name()))
}

func Test__inaccessible_root_is_skipped(t *testing.T) {
	d, err := ioutil.TempDir("", "inaccessible")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(d)
		assert.NoError(t, err)
	}()
	err = os.Chmod(d, 0) // remove permissions
	require.NoError(t, err)

	buf := logBuffer()
	res, err := Run(d, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, &Dir{Name: filepath.Base(d)}, res)
	assert.Equal(t, fmt.Sprintf("skipping inaccessible directory %q\n", d), buf.String())

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
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_root_fails(t *testing.T) {
	root := "testdata"
	_, err := Run(root, skip(root), nil)
	assert.EqualError(t, err, `skipping root directory "testdata"`)
}

func Test__testdata_skip_symlinked_root_fails(t *testing.T) {
	symlinkName := "test_root-symlink"
	err := os.Symlink("testdata", symlinkName)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(symlinkName)
		assert.NoError(t, err)
	}()
	_, err = Run(symlinkName, skip(symlinkName), nil)
	assert.EqualError(t, err, `skipping root directory "test_root-symlink"`)
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
	res, err := Run(root, skip("b"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__SkipNameSet_empty_always_returns_false(t *testing.T) {
	shouldSkip := SkipNameSet(nil)

	tests := []struct {
		name     string
		dirName  string
		baseName string
	}{
		{name: "empty dir- and basename", dirName: "", baseName: ""},
		{name: "empty dirname and nonempty basename", dirName: "", baseName: "x"},
		{name: "nonempty dirname and empty basename", dirName: "x", baseName: ""},
		{name: "nonempty dir- and basename", dirName: "x", baseName: "y"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			skip := shouldSkip(test.dirName, test.baseName)
			assert.False(t, skip)
		})
	}
}

func Test__SkipNameSet_nonempty_returns_whether_basename_matches(t *testing.T) {
	shouldSkip := SkipNameSet(map[string]struct{}{"a": {}, "b": {}})

	tests := []struct {
		name     string
		dirName  string
		baseName string
		want     bool
	}{
		{name: "empty dir- and basename", dirName: "", baseName: "", want: false},
		{name: "empty dirname and matching basename", dirName: "", baseName: "a", want: true},
		{name: "empty dirname and nonmatching basename", dirName: "", baseName: "x", want: false},
		{name: "matching dirname and empty basename", dirName: "a", baseName: "", want: false},
		{name: "nonmatching dirname and empty basename", dirName: "x", baseName: "", want: false},

		{name: "nonmatching dirname and nonmatching basename", dirName: "x", baseName: "y", want: false},
		{name: "nonmatching dirname and matching basename", dirName: "x", baseName: "b", want: true},
		{name: "matching dirname and nonmatching basename", dirName: "a", baseName: "x", want: false},
		{name: "matching dir- and basename", dirName: "a", baseName: "b", want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			skip := shouldSkip(test.dirName, test.baseName)
			assert.Equal(t, test.want, skip)
		})
	}
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
	res, err := Run(root, skip("g"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_trailing_slash_gets_removed(t *testing.T) {
	root := "testdata/e/f/"
	want := &Dir{
		Name:       "f",
		Files:      []*File{testdata_e_f_a},
		EmptyFiles: []string{"g"},
	}
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__inaccessible_internal_file_is_not_hashed(t *testing.T) {
	f, err := ioutil.TempFile("testdata/e/f", "inaccessible")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString("53cR31_")
	require.NoError(t, err)
	require.Equal(t, 7, n)

	err = f.Chmod(0) // remove permissions
	require.NoError(t, err)

	root := "testdata/e/f"
	want := &Dir{
		Name: "f",
		Files: []*File{
			testdata_e_f_a,
			{
				Name: filepath.Base(f.Name()),
				Size: 7,
				Hash: 0,
			},
		},
		EmptyFiles: []string{"g"},
	}
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__inaccessible_internal_dir_fails(t *testing.T) {
	d, err := ioutil.TempDir("testdata/e/f", "inaccessible")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(d)
		assert.NoError(t, err)
	}()

	err = os.Chmod(d, 0) // remove permissions
	require.NoError(t, err)

	root := "testdata/e"
	want := &Dir{
		Name: "e",
		Dirs: []*Dir{
			{
				Name:       "f",
				Files:      []*File{testdata_e_f_a},
				EmptyFiles: []string{"g"},
			},
		},
	}
	buf := logBuffer()
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("skipping inaccessible directory %q\n", d), buf.String())
}

func Test__testdata_cache_with_mismatching_root_fails(t *testing.T) {
	root := "testdata"
	cache := &Dir{
		Name: "not-testdata",
	}
	_, err := Run(root, skip("a"), cache)
	assert.EqualError(t, err, `cache of directory "not-testdata" cannot be used with root directory "testdata"`)
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

func Test__root_symlink_is_followed(t *testing.T) {
	symlinkName := "test_root-symlink"

	err := os.Symlink("testdata/e/f", symlinkName)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(symlinkName)
		assert.NoError(t, err)
	}()
	want := &Dir{
		Name:       symlinkName,
		Files:      []*File{testdata_e_f_a},
		EmptyFiles: []string{"g"},
	}
	res, err := Run(symlinkName, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__root_indirect_symlink_is_followed(t *testing.T) {
	indirectSymlink := "test_indirect-root-symlink"
	symlink := "test-symlink"

	err := os.Symlink(symlink, indirectSymlink)
	require.NoError(t, err)
	err = os.Symlink("testdata/e/f", symlink)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(indirectSymlink)
		assert.NoError(t, err)
	}()
	defer func() {
		err := os.Remove(symlink)
		assert.NoError(t, err)
	}()
	want := &Dir{
		Name:       indirectSymlink,
		Files:      []*File{testdata_e_f_a},
		EmptyFiles: []string{"g"},
	}
	res, err := Run(indirectSymlink, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__internal_symlink_is_skipped(t *testing.T) {
	symlink := "testdata/e/f/test_internal-symlink"

	err := os.Symlink("testdata", symlink)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(symlink)
		assert.NoError(t, err)
	}()

	root := "testdata/e/f"
	want := &Dir{
		Name:       "f",
		Files:      []*File{testdata_e_f_a}, // doesn't include the symlink as file nor the dir it points at
		EmptyFiles: []string{"g"},
	}
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__root_symlink_to_ancestor_is_followed_but_skipped_when_internal(t *testing.T) {
	symlink := "testdata/e/f/test_ancestor-symlink" // points to "testdata/e"

	err := os.Symlink("..", symlink)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(symlink)
		assert.NoError(t, err)
	}()

	want := &Dir{
		Name: "test_ancestor-symlink",
		Dirs: []*Dir{
			{
				Name:       "f",
				Files:      []*File{testdata_e_f_a}, // doesn't include the symlink as file nor the dir it points at
				EmptyFiles: []string{"g"},
			},
		},
	}
	res, err := Run(symlink, NoSkip, nil)
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

func logBuffer() *bytes.Buffer {
	var buf bytes.Buffer
	log.SetFlags(0)
	log.SetOutput(&buf)
	return &buf
}
