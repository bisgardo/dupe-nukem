package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/testutil"
)

// Working dir for tests is the directory containing the file.

// TODO: Figure out how to test symlinks on Windows:
//       - Create specialized tests that need to be run manually as administrator.
//       - Handle/test Windows-specific features: shortcuts, junctions.
// TODO: Test logged output whenever it's relevant.

func Test__empty_dir(t *testing.T) {
	dir, err := os.MkdirTemp("", "empty")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(dir)
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
	f, err := os.CreateTemp("", "root")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	err = f.Close()
	assert.NoError(t, err)
	_, err = Run(f.Name(), NoSkip, nil)
	assert.EqualError(t, err, fmt.Sprintf("invalid root directory %q: not a directory", f.Name()))
}

// DISABLED on Windows and macOS (on GitHub): This test only works as expected on the Linux setup of GitHub Actions.
// The reason is that on the other platforms, the project is checked out on a symlinked path which
// gets resolved by dupe-nukem.
// This causes another log entry to be emitted and also the symlinked path (which is not known in advance) to be used.
// We cannot (or shouldn't at least) control how GitHub Actions sets up the project as that is internal
// to their infrastructure.
// It's also possible that we can/should avoid resolving symlinks in cases like this...
// The fact that the test passes on Linux and Windows (tested locally) gives reasonable confidence
// that the functionality really is correct on all platforms - the test just needs to be set up some other way.
// But the encountered case (that inaccessible symlinked root should be skipped) should be properly tested as well!
func Test__inaccessible_root_is_skipped(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if testutil.CI() == "github" && runtime.GOOS != "linux" {
		return // skip test
	}
	d, err := os.MkdirTemp("", "inaccessible")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(d)
		assert.NoError(t, err)
	}()
	err = testutil.MakeDirInaccessible(d)
	require.NoError(t, err)
	defer func() {
		err := testutil.MakeDirAccessible(d)
		assert.NoError(t, err)
	}()

	buf := testutil.LogBuffer()
	res, err := Run(d, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, &Dir{Name: filepath.Base(d)}, res)
	assert.Equal(t, fmt.Sprintf("skipping inaccessible directory %q\n", d), buf.String())
}

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

// DISABLED on Windows: Creating symlinks requires elevated privileges.
func Test__testdata_skip_symlinked_root_fails(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		return // skip test
	}
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
		{name: "empty dirname and non-matching basename", dirName: "", baseName: "x", want: false},
		{name: "matching dirname and empty basename", dirName: "a", baseName: "", want: false},
		{name: "non-matching dirname and empty basename", dirName: "x", baseName: "", want: false},

		{name: "non-matching dirname and non-matching basename", dirName: "x", baseName: "y", want: false},
		{name: "non-matching dirname and matching basename", dirName: "x", baseName: "b", want: true},
		{name: "matching dirname and non-matching basename", dirName: "a", baseName: "x", want: false},
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

func Test__testdata_skip_files_is_logged(t *testing.T) {
	root := "testdata"
	buf := testutil.LogBuffer()
	_, err := Run(root, skip("a"), nil)
	require.NoError(t, err)
	want := fmt.Sprintf(
		"skipping file %q based on skip list\nskipping file %q based on skip list\n",
		filepath.Clean("testdata/a"),
		filepath.Clean("testdata/e/f/a"),
	)
	assert.Equal(t, want, buf.String())
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

// On Windows, this test only works if the repository is stored on an NTFS drive.
func Test__inaccessible_internal_file_is_not_hashed_and_is_logged(t *testing.T) {
	f, err := os.CreateTemp("testdata/e/f", "inaccessible")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString("53cR31_")
	require.NoError(t, err)
	require.Equal(t, 7, n)

	err = testutil.MakeFileInaccessible(f)
	require.NoError(t, err)

	err = f.Close()
	assert.NoError(t, err)

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
	buf := testutil.LogBuffer()
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("error: cannot hash file %q: cannot open file: access denied\n", filepath.Clean(f.Name())), buf.String())
}

// On Windows, this test only works if the repository is stored on an NTFS drive.
func Test__inaccessible_internal_dir_is_logged(t *testing.T) {
	d, err := os.MkdirTemp("testdata/e/f", "inaccessible")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(d)
		assert.NoError(t, err)
	}()

	err = testutil.MakeDirInaccessible(d)
	require.NoError(t, err)
	defer func() {
		err := testutil.MakeDirAccessible(d)
		assert.NoError(t, err)
	}()

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
	buf := testutil.LogBuffer()
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("skipping inaccessible directory %q\n", filepath.Clean(d)), buf.String())
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

func Test__testdata_subdir_cache_not_used_for_mismatching_file_size(t *testing.T) {
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

func Test__cache_entry_with_hash_0_is_ignored(t *testing.T) {
	root := "testdata/b"
	cache := &Dir{
		Name: "b",
		Files: []*File{
			{
				Name: "d",
				Size: 2,
				Hash: 0, // value 0 is intentionally ignored
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

// DISABLED on Windows and macOS (on GitHub) for the same reasons as 'Test__inaccessible_root_is_skipped'.
// TODO: Why is this test so complex? Does/should it test more than just logging?
func Test__hash_computed_as_0_is_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if testutil.CI() == "github" && runtime.GOOS != "linux" {
		return // skip test
	}

	v := "77kepQFQ8Kl" // from 'https://md5hashing.net/hash/fnv1a64/0000000000000000'

	d, err := os.MkdirTemp("", "hash0")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(d)
		assert.NoError(t, err)
	}()

	f, err := os.CreateTemp(d, "hash0")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	_, err = f.WriteString(v)
	require.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)
	buf := testutil.LogBuffer()
	res, err := Run(d, NoSkip, nil)
	want := &Dir{
		Name: filepath.Base(d),
		Files: []*File{{
			Name: filepath.Base(f.Name()),
			Size: 11,
			Hash: 0,
		}},
	}
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("info: hash of file %q evaluated to 0 - this might result in warnings which can be safely ignored\n", f.Name()), buf.String())
}

// DISABLED on Windows: Creating symlinks require elevated privileges.
func Test__root_symlink_is_followed_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		return // skip test
	}
	symlinkTarget := "testdata/e/f"
	symlink := "test_root-symlink"

	err := os.Symlink(symlinkTarget, symlink)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(symlink)
		assert.NoError(t, err)
	}()
	want := &Dir{
		Name:       symlink,
		Files:      []*File{testdata_e_f_a},
		EmptyFiles: []string{"g"},
	}
	buf := testutil.LogBuffer()
	res, err := Run(symlink, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("following root symlink %q to %q\n", symlink, symlinkTarget), buf.String())
}

// DISABLED on Windows: Creating symlinks require elevated privileges.
func Test__root_indirect_symlink_is_followed_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		return // skip test
	}
	symlinkTarget := "testdata/e/f"
	indirectSymlink := "test_indirect-root-symlink"
	symlink := "test-symlink"

	err := os.Symlink(symlinkTarget, symlink)
	require.NoError(t, err)
	err = os.Symlink(symlink, indirectSymlink)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(indirectSymlink)
		assert.NoError(t, err)
		err = os.Remove(symlink)
		assert.NoError(t, err)
	}()
	want := &Dir{
		Name:       indirectSymlink,
		Files:      []*File{testdata_e_f_a},
		EmptyFiles: []string{"g"},
	}
	buf := testutil.LogBuffer()
	res, err := Run(indirectSymlink, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("following root symlink %q to %q\n", indirectSymlink, symlinkTarget), buf.String())
}

// DISABLED on Windows: Creating symlinks require elevated privileges.
func Test__internal_symlink_is_skipped_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		return // skip test
	}
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
	buf := testutil.LogBuffer()
	res, err := Run(root, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("skipping symlink %q during scan\n", symlink), buf.String())
}

// DISABLED on Windows: Creating symlinks require elevated privileges.
func Test__root_symlink_to_ancestor_is_followed_but_skipped_when_internal(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		return // skip test
	}
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
