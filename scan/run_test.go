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

func Test__empty_dir(t *testing.T) {
	root := t.TempDir()

	want := &Dir{Name: filepath.Base(root)}

	tests := []struct {
		name       string
		shouldSkip ShouldSkipPath
	}{
		{name: "without skip", shouldSkip: NoSkip},
		{name: "skipping root", shouldSkip: skip(root)},
		{name: "skipping non-existing", shouldSkip: skip("non-existing")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := Run(root, test.shouldSkip, nil)
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

func Test__inaccessible_root_is_skipped(t *testing.T) {
	// Resolve symlink to prevent a log entry from being emitted on macOS where tmp paths are symlinked.
	// On GitHub Actions, this is also necessary for the Windows runner because the provided dir path
	// 'C:\Users\RUNNER~1\AppData\Local\Temp\...' somehow resolves as a symlink to 'C:\Users\runneradmin\AppData\Local\Temp\...'.
	rootPath, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	err = testutil.MakeInaccessible(rootPath)
	require.NoError(t, err)
	want := &Dir{Name: filepath.Base(rootPath)}

	buf := testutil.LogBuffer()
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t, fmt.Sprintf("skipping inaccessible directory %q\n", rootPath), buf.String())
}

func Test__testdata_no_skip(t *testing.T) {
	root := dir{
		"a":   file{c: "x\n"},
		"c":   file{c: "y\n"},
		"b/d": file{c: "x\n"},
		"e/f": dir{
			"a": file{c: "z\n"},
			"g": file{},
		},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_root_fails(t *testing.T) {
	tests := []struct {
		name     string
		rootPath string
	}{
		{name: "existing", rootPath: t.TempDir()},
		{name: "non-existing", rootPath: "non-existing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Run(test.rootPath, skip(filepath.Base(test.rootPath)), nil)
			assert.EqualError(t, err, fmt.Sprintf("skipping root directory %q", test.rootPath))
		})
	}
}

// SKIPPED on Windows unless running as administrator.
func Test__testdata_skip_symlinked_root_fails(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !testutil.IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
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
	root := dir{
		"a": file{c: "x\n"},
		"c": file{c: "y\n"},
		"b": dirExt{
			skipped: true,
			dir: dir{
				"d": file{c: "x\n"},
			},
		},
		"e/f": dir{
			"a": file{c: "z\n"},
			"g": file{},
		},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	res, err := Run(rootPath, skip("b"), nil)
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
	root := dir{
		"a":   file{c: "x\n"},
		"c":   file{c: "y\n"},
		"b/d": file{c: "x\n"},
		"e": dirExt{
			skipped: true,
			dir: dir{
				"f": dir{
					"a": file{c: "z\n", skipped: true},
					"g": file{},
				},
			},
		},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	res, err := Run(rootPath, skip("e"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_nonempty_file(t *testing.T) {
	root := dir{
		"a":   file{c: "x\n", skipped: true},
		"c":   file{c: "y\n"},
		"b/d": file{c: "x\n"},
		"e/f": dir{
			"a": file{c: "z\n", skipped: true},
			"g": file{},
		},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	res, err := Run(rootPath, skip("a"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_subdir_skip_empty_file(t *testing.T) {
	root := dir{
		"a": file{c: "z\n"},
		"g": file{skipped: true},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))
	res, err := Run(rootPath, skip("g"), nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_skip_files_is_logged(t *testing.T) {
	root := dir{
		"a":   file{c: "x\n"},
		"c":   file{c: "y\n"},
		"b/d": file{c: "x\n"},
		"e/f": dir{
			"a": file{c: "z\n"},
			"g": file{},
		},
	}
	// Resolving symlink for the same reason as described in a comment of 'Test__inaccessible_root_is_skipped'.
	rootPath, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	err = root.writeTo(rootPath)
	require.NoError(t, err)

	buf := testutil.LogBuffer()
	_, err = Run(rootPath, skip("a"), nil)
	require.NoError(t, err)
	want := fmt.Sprintf(
		"skipping file %q based on skip list\nskipping file %q based on skip list\n",
		filepath.Join(rootPath, "a"),
		filepath.Join(rootPath, "e/f/a"),
	)
	assert.Equal(t, want, buf.String())
}

func Test__testdata_trailing_slash_gets_removed(t *testing.T) {
	root := dir{
		"a": file{c: "z\n"},
		"g": file{},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	// Note added '/'.
	res, err := Run(rootPath+"/", NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

// On Windows, this test only works if the repository is stored on an NTFS drive.
func Test__inaccessible_internal_file_is_not_hashed_and_is_logged(t *testing.T) {
	root := dir{
		"a":            file{c: "z\n"},
		"g":            file{},
		"inaccessible": file{c: "53cR31_", inaccessible: true},
	}
	// Resolving symlink for the same reason as described in a comment of 'Test__inaccessible_root_is_skipped'.
	rootPath, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	err = root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	buf := testutil.LogBuffer()
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t,
		fmt.Sprintf(
			"error: cannot hash file %q: cannot open file: access denied\n",
			filepath.Join(rootPath, "inaccessible"),
		),
		buf.String(),
	)
}

// On Windows, this test only works if the repository is stored on a filesystem
// that supports the command 'icacls' (such as NTFS).
func Test__inaccessible_internal_dir_is_logged(t *testing.T) {
	root := dir{
		"f": dir{
			"a":            file{c: "z\n"},
			"g":            file{},
			"inaccessible": dirExt{inaccessible: true},
		},
	}
	// Resolving symlink for the same reason as described in a comment of 'Test__inaccessible_root_is_skipped'.
	rootPath, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	err = root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	buf := testutil.LogBuffer()
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t,
		fmt.Sprintf(
			"skipping inaccessible directory %q\n",
			filepath.Join(rootPath, "f/inaccessible"),
		),
		buf.String(),
	)
}

func Test__testdata_cache_with_mismatching_root_fails(t *testing.T) {
	rootPath := "some-root"
	cache := &Dir{Name: "other-root"}
	_, err := Run(rootPath, NoSkip, cache)
	assert.EqualError(t, err, `cache of directory "other-root" cannot be used with root directory "some-root"`)
}

func Test__testdata_with_hashes_from_cache(t *testing.T) {
	root := dir{
		"a":   file{c: "x\n"},
		"c":   file{c: "y\n", cachedHash: 53},
		"b/d": file{c: "x\n"},
		"e/f": dir{
			"a": file{c: "z\n", cachedHash: 42},
			"g": file{},
		},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	cache := &Dir{
		Name: want.Name,
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
					{Name: "d", Size: 2, Hash: 69}, // not used
				},
			},
		},
		Files: []*File{
			{Name: "c", Size: 2, Hash: 53}, // used
			{Name: "d", Size: 2, Hash: 69}, // not used
		},
	}
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__testdata_subdir_cache_not_used_for_mismatching_file_size(t *testing.T) {
	root := dir{
		"d": file{c: "x\n"},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	cache := &Dir{
		Name: want.Name,
		Files: []*File{
			{
				Name: "d",
				Size: 1,  // size of "d" is 2
				Hash: 21, // so the cached hash value is not read
			},
		},
	}
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__cache_entry_with_hash_0_is_ignored(t *testing.T) {
	root := dir{
		"d": file{c: "x\n"},
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	cache := &Dir{
		Name: want.Name,
		Files: []*File{
			{
				Name: "d",
				Size: 2,
				Hash: 0, // value 0 is specifically ignored
			},
		},
	}
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__hash_computed_as_0_is_logged(t *testing.T) {
	root := dir{
		// Contents hash to 0 (https://md5hashing.net/hash/fnv1a64/0000000000000000).
		"hash0": file{c: "77kepQFQ8Kl"},
	}
	// Resolving symlink for the same reason as described in a comment of 'Test__inaccessible_root_is_skipped'.
	rootPath, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	err = root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	buf := testutil.LogBuffer()
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t,
		fmt.Sprintf(
			"info: hash of file %q evaluated to 0 - this might result in warnings which can be safely ignored\n",
			filepath.Join(rootPath, "hash0"),
		),
		buf.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__root_symlink_is_followed_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !testutil.IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}

	symlinkedDir := dir{
		"a": file{c: "z\n"},
		"g": file{},
	}
	symlinkName := "symlink"
	root := dir{
		symlinkName: symlink("data"),
		"data":      symlinkedDir,
	}

	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)

	symlinkPath := filepath.Join(rootPath, symlinkName)
	want := symlinkedDir.toScanDir(symlinkName)

	buf := testutil.LogBuffer()
	res, err := Run(symlinkPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t,
		fmt.Sprintf(
			"following root symlink %q to %q\n",
			symlinkPath,
			filepath.Join(rootPath, "data"), // Clean replaces '/' with '\' on Windows
		),
		buf.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__root_indirect_symlink_is_followed_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !testutil.IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}

	symlinkedDir := dir{
		"a": file{c: "z\n"},
		"g": file{},
	}
	symlinkName := "indirect-symlink"
	testDir := dir{
		symlinkName: symlink("symlink"),
		"symlink":   symlink("data"),
		"data":      symlinkedDir,
	}

	rootPath := t.TempDir()
	err := testDir.writeTo(rootPath)
	require.NoError(t, err)

	want := symlinkedDir.toScanDir(symlinkName)
	buf := testutil.LogBuffer()
	indirectSymlinkPath := filepath.Join(rootPath, symlinkName)
	res, err := Run(indirectSymlinkPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t,
		fmt.Sprintf(
			"following root symlink %q to %q\n",
			indirectSymlinkPath,
			filepath.Clean(filepath.Join(rootPath, "data")), // Clean replaces '/' with '\' on Windows
		),
		buf.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__internal_symlink_is_skipped_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !testutil.IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}
	symlinkName := "symlink"
	root := dir{
		"a":         file{c: "z\n"},
		symlinkName: symlink("a"), // TODO: add test where target doesn't exist?
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	buf := testutil.LogBuffer()
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	assert.Equal(t, want, res)
	assert.Equal(t,
		fmt.Sprintf(
			"skipping symlink %q during scan\n",
			filepath.Join(rootPath, symlinkName),
		),
		buf.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__root_symlink_to_ancestor_is_followed_but_skipped_when_internal(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !testutil.IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}
	symlinkName := "internal-symlink"
	symlinkedDir := dir{
		"a":         file{c: "z\n"},
		"g":         file{},
		symlinkName: symlink(".."), // points to "e"
	}
	root := dir{
		"e/f": symlinkedDir,
	}
	rootPath := t.TempDir()
	err := root.writeTo(rootPath)
	require.NoError(t, err)
	want := root.toScanDir(filepath.Base(rootPath))

	res, err := Run(rootPath, NoSkip, nil)
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
