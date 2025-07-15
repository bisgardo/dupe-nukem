package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/bisgardo/dupe-nukem/testutil"
)

// TODO: Revise data setup of all tests using the new system (construct more targeted setups)
//       rather than just keep reconstructing the old testdata contents.

// TODO: Figure out how to test Windows-specific features (shortcuts, junctions).

func Test__empty_dir(t *testing.T) {
	rootDir := tempDir(t)

	want := &Result{
		TypeVersion: CurrentResultTypeVersion,
		Root:        &Dir{Name: rootDir},
	}

	tests := []struct {
		name       string
		shouldSkip ShouldSkipPath
	}{
		{name: "without skip", shouldSkip: NoSkip},
		{name: "skipping root", shouldSkip: makeSkip(rootDir)},
		{name: "skipping non-existing", shouldSkip: makeSkip("non-existing")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := Run(rootDir, test.shouldSkip, nil)
			require.NoError(t, err)
			res.assertEqual(t, want)
		})
	}
}

func Test__nonexistent_root_fails(t *testing.T) {
	t.Run("without trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent", NoSkip, nil)
		assert.EqualError(t, err, `invalid root directory "nonexistent": not found`)
	})
	t.Run("with trailing slash", func(t *testing.T) {
		_, err := Run("nonexistent/", NoSkip, nil)
		assert.EqualError(t, err, `invalid root directory "nonexistent/": not found`)
	})
	t.Run("skipping is not logged", func(t *testing.T) {
		rootPath := "nonexistent"
		logs := CaptureLogs(t)
		_, err := Run(rootPath, makeSkip(filepath.Base(rootPath)), nil)
		assert.EqualError(t, err, `invalid root directory "nonexistent": not found`)
		assert.Empty(t, logs.String())
	})
}

// SKIPPED on Windows unless running as administrator.
func Test__symlink_to_nonexistent_root_fails(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}

	symlinkName := "broken-root-symlink"
	rootPath := tempDir(t)
	symlinkPath := filepath.Join(rootPath, symlinkName)
	symlink("nonexistent").writeTestdata(t, symlinkPath)

	t.Run("without trailing slash", func(t *testing.T) {
		_, err := Run(symlinkPath, NoSkip, nil)
		assert.EqualError(t, err, fmt.Sprintf(`invalid root directory %q: not found`, symlinkPath))
	})
	t.Run("with trailing slash", func(t *testing.T) {
		root := symlinkPath + "/"
		_, err := Run(root, NoSkip, nil)
		assert.EqualError(t, err, fmt.Sprintf(`invalid root directory %q: not found`, root))
	})
	t.Run("skipping is not logged", func(t *testing.T) {
		logs := CaptureLogs(t)
		_, err := Run(symlinkPath, makeSkip(filepath.Base(rootPath)), nil)
		assert.EqualError(t, err, fmt.Sprintf(`invalid root directory %q: not found`, symlinkPath))
		assert.Empty(t, logs.String())
	})
}

func Test__file_root_fails(t *testing.T) {
	path := TempStringFile(t, "")
	_, err := Run(path, NoSkip, nil)
	assert.EqualError(t, err, fmt.Sprintf("invalid root directory %q: not a directory", path))
}

func Test__inaccessible_root_is_skipped_and_logged(t *testing.T) {
	rootPath := tempDir(t)
	MakeInaccessibleT(t, rootPath)
	want := &Result{
		TypeVersion: CurrentResultTypeVersion,
		Root:        &Dir{Name: rootPath},
	}
	logs := CaptureLogs(t)
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("skipping inaccessible directory %q"),
			rootPath,
		),
		logs.String(),
	)
}

func Test__no_skip(t *testing.T) {
	root := dir{
		"a":   file{c: "x\n"},
		"c":   file{c: "y\n"},
		"b/d": file{c: "x\n"},
		"e/f": dir{
			"a": file{c: "z\n"},
			"g": file{},
		},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
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

func Test__root_cannot_be_skipped(t *testing.T) {
	root := dir{"a": file{c: "x"}}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)
	logs := CaptureLogs(t)
	res, err := Run(rootPath, makeSkip(filepath.Base(rootPath)), nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(Lines("not skipping root directory %q"), rootPath),
		logs.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__symlink_root_cannot_be_skipped(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}
	root := dir{"a": file{c: "x"}}
	symlinkName := "root-symlink"

	rootName := "root"
	wrap := dir{
		rootName:    root,
		symlinkName: symlink(rootName),
	}
	wrapPath := tempDir(t)
	wrap.writeTestdata(t, wrapPath)
	symlinkPath := filepath.Join(wrapPath, symlinkName)
	rootPath := filepath.Join(wrapPath, rootName)

	want := simulateScan(root, rootPath)
	log1 := fmt.Sprintf("following root symlink %q to %q", symlinkPath, rootPath)
	log2 := fmt.Sprintf("not skipping root directory %q", rootPath)

	tests := []struct {
		name     string
		skipName string
		wantLogs string
	}{
		{
			name:     "skipping symlink name is not logged",
			skipName: symlinkName,
			wantLogs: Lines(log1),
		},
		{
			name:     "skipping resolved name is logged",
			skipName: rootName,
			wantLogs: Lines(log1, log2),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logs := CaptureLogs(t)
			res, err := Run(symlinkPath, makeSkip(test.skipName), nil)
			require.NoError(t, err)
			res.assertEqual(t, want)
			assert.Equal(t, test.wantLogs, logs.String())
		})
	}
}

func Test__skip_dir_without_subdirs_is_logged(t *testing.T) {
	root := dir{
		"a": file{c: "x\n"},
		"c": file{c: "y\n"},
		"b": dirExt{skipped: true},
		"e/f": dir{
			"a": file{c: "z\n"},
			"g": file{},
		},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootPath, makeSkip("b"), nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("skipping directory %q based on skip list"),
			filepath.Join(rootPath, "b"),
		),
		logs.String(),
	)
}

func Test__skip_dir_with_subdirs_is_logged(t *testing.T) {
	root := dir{
		"a":   file{c: "x\n"},
		"c":   file{c: "y\n"},
		"b/d": file{c: "x\n"},
		"e": dirExt{
			skipped: true,
			dir: dir{
				"f": dir{
					"a": file{c: "z\n"},
					"g": file{},
				},
			},
		},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootPath, makeSkip("e"), nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("skipping directory %q based on skip list"),
			filepath.Join(rootPath, "e"),
		),
		logs.String(),
	)
}

func Test__skip_nonempty_files_is_logged(t *testing.T) {
	root := dir{
		"a":   file{c: "x\n", skipped: true},
		"c":   file{c: "y\n"},
		"b/d": file{c: "x\n"},
		"e/f": dir{
			"a": file{c: "z\n", skipped: true},
			"g": file{},
		},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootPath, makeSkip("a"), nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines(
				"skipping file %q based on skip list",
				"skipping file %q based on skip list",
			),
			filepath.Join(rootPath, "a"),
			filepath.Join(rootPath, "e/f/a"),
		),
		logs.String(),
	)
}

func Test__skip_empty_file_is_logged(t *testing.T) {
	root := dir{
		"a": file{c: "z\n"},
		"g": file{skipped: true},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)
	logs := CaptureLogs(t)
	res, err := Run(rootPath, makeSkip("g"), nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("skipping file %q based on skip list"),
			filepath.Join(rootPath, "g"),
		),
		logs.String(),
	)
}

func Test__skip_symlink_is_logged(t *testing.T) {
	symlinkName := "symlink"
	root := dir{
		symlinkName: symlinkExt{
			symlink: symlink("skipped"),
			skipped: true,
		},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	symlinkPath := filepath.Join(rootPath, symlinkName)
	want := simulateScan(root, rootPath)
	logs := CaptureLogs(t)
	res, err := Run(rootPath, makeSkip(symlinkName), nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(Lines("skipping symlink %q based on skip list"), symlinkPath),
		logs.String(),
	)
}

func Test__skip_file_after_dir(t *testing.T) {
	// Added while fixing a bug where the skipped node was wrongfully added to the directory
	// that was just scanned because 'head' wasn't getting updated until *after* the skip check.
	root := dir{
		"a": dir{},
		"x": file{skipped: true},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)
	logs := CaptureLogs(t)
	res, err := Run(rootPath, makeSkip("x"), nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("skipping file %q based on skip list"),
			filepath.Join(rootPath, "x"),
		),
		logs.String(),
	)
}

func Test__trailing_slash_of_run_path_gets_removed(t *testing.T) {
	root := dir{
		"a": file{c: "z\n"},
		"g": file{},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	res, err := Run(rootPath+"/", NoSkip, nil) // note added '/'
	require.NoError(t, err)
	res.assertEqual(t, want)
}

// On Windows, this test only works if the repository is stored on an NTFS drive.
// TODO: Detect and skip based on the above (something like testutil.UsesInaccessible(t) - which we can then assert against).
//
//	... and maybe even call t.Parallel() automatically when certain features are not used?
func Test__inaccessible_internal_file_is_not_hashed_and_is_logged(t *testing.T) {
	root := dir{
		"a":            file{c: "z\n"},
		"g":            file{},
		"inaccessible": file{c: "53cR31_", inaccessible: true},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("error: cannot hash file %q: cannot open file: access denied"),
			filepath.Join(rootPath, "inaccessible"),
		),
		logs.String(),
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
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("skipping inaccessible directory %q"),
			filepath.Join(rootPath, "f/inaccessible"),
		),
		logs.String(),
	)
}

func Test__inaccessible_internal_empty_file_is_not_logged(t *testing.T) {
	root := dir{
		"a":                  file{c: "x"},
		"inaccessible+empty": file{inaccessible: true},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)
	logs := CaptureLogs(t)
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Empty(t, logs.String())
}

func Test__cache_root_name_check(t *testing.T) {
	t.Run("mismatching name is rejected", func(t *testing.T) {
		rootPath, err := os.Getwd()
		require.NoError(t, err)
		cache := &Dir{Name: "other-root"}
		_, err = Run(rootPath, NoSkip, cache)
		assert.EqualError(t, err, fmt.Sprintf("cache of directory %q cannot be used with root directory %q", "other-root", rootPath))
	})
	t.Run("cache name matches root", func(t *testing.T) {
		rootPath := tempDir(t)
		cache := &Dir{Name: rootPath}
		res, err := Run(rootPath, NoSkip, cache)
		require.NoError(t, err)
		assert.Equal(t, &Result{
			TypeVersion: CurrentResultTypeVersion,
			Root:        &Dir{Name: rootPath},
		}, res)
	})
	t.Run("cache name matches after resolving root symlink", func(t *testing.T) {
		rootName := "root"
		symlinkName := "symlink"
		wrap := dir{
			rootName:    dir{},
			symlinkName: symlink(rootName),
		}
		wrapPath := tempDir(t)
		wrap.writeTestdata(t, wrapPath)
		rootPath := filepath.Join(wrapPath, rootName)
		rootSymlinkPath := filepath.Join(wrapPath, symlinkName)

		cache := &Dir{Name: rootPath}
		res, err := Run(rootSymlinkPath, NoSkip, cache)
		require.NoError(t, err)
		assert.Equal(t, &Result{
			TypeVersion: CurrentResultTypeVersion,
			Root:        &Dir{Name: rootPath}, //
		}, res)
	})
	t.Run("cache name symlink is not followed", func(t *testing.T) {
		rootName := "root"
		cacheSymlinkName := "cache-symlink"
		wrap := dir{
			rootName:         dir{},
			cacheSymlinkName: symlink(rootName),
		}
		wrapPath := tempDir(t)
		wrap.writeTestdata(t, wrapPath)
		rootPath := filepath.Join(wrapPath, rootName)
		cacheSymlinkPath := filepath.Join(wrapPath, cacheSymlinkName)

		cache := &Dir{Name: cacheSymlinkPath}
		logs := CaptureLogs(t)
		_, err := Run(rootPath, NoSkip, cache)
		assert.EqualError(t, err, fmt.Sprintf("cache of directory %q cannot be used with root directory %q", cacheSymlinkPath, rootPath))
		assert.Empty(t, logs.String())
	})
}

func Test__hashes_from_cache_are_used(t *testing.T) {
	ts, err := time.Parse(time.Layout, time.Layout)
	require.NoError(t, err)

	root := dir{
		"a":   file{c: "x\n", ts: ts},
		"c":   file{c: "y\n", ts: ts, hashFromCache: 53},
		"b/d": file{c: "x\n", ts: ts},
		"e/f": dir{
			"a": file{c: "z\n", ts: ts, hashFromCache: 42},
			"g": file{ts: ts},
		},
		"h": file{c: "q\n", ts: ts},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	tsUnix := ts.Unix()
	cache := &Dir{
		Name: want.Root.Name,
		Dirs: []*Dir{
			{
				Name: "e",
				Dirs: []*Dir{
					{
						Name: "f",
						Files: []*File{
							{Name: "a", Size: 2, ModTime: tsUnix, Hash: 42}, // used
							{Name: "g", Size: 0, ModTime: tsUnix, Hash: 42}, // not used: size and time match, but file is empty
						},
					},
				},
				Files: []*File{
					{Name: "d", Size: 2, ModTime: tsUnix, Hash: 69}, // not used: file doesn't exist in testdata (but "b/d" does)
				},
			},
		},
		Files: []*File{
			// no entry for "a"
			{Name: "b", Size: 1, ModTime: tsUnix, Hash: 69}, // not used: "b" is a dir in testdata
			{Name: "c", Size: 2, ModTime: tsUnix, Hash: 53}, // used
			{Name: "d", Size: 2, ModTime: tsUnix, Hash: 69}, // not used: no such file in testdata
		},
	}
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	res.assertEqual(t, want)
}

func Test__cache_with_mismatching_file_size_is_not_used(t *testing.T) {
	ts, err := time.Parse(time.Layout, time.Layout)
	require.NoError(t, err)

	root := dir{
		"d": file{c: "x\n", ts: ts},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	cache := &Dir{
		Name: want.Root.Name,
		Files: []*File{
			{
				Name:    "d",
				Size:    69,        // size of "d" is 2,
				ModTime: ts.Unix(), // so even with correct mod time,
				Hash:    21,        // the cached hash value is not used
			},
		},
	}
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	res.assertEqual(t, want)
}

func Test__cache_with_mismatching_file_mod_time_is_not_used(t *testing.T) {
	ts, err := time.Parse(time.Layout, time.Layout)
	require.NoError(t, err)

	root := dir{
		"d": file{c: "x\n", ts: ts},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	cache := &Dir{
		Name: want.Root.Name,
		Files: []*File{
			{
				Name:    "d",
				Size:    2,             // size is correct,
				ModTime: ts.Unix() + 1, // but mod time isn't,
				Hash:    21,            // so the cached hash value is not used
			},
		},
	}
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	res.assertEqual(t, want)
}

func Test__hash_of_inaccessible_file_is_used(t *testing.T) {
	ts, err := time.Parse(time.Layout, time.Layout)
	require.NoError(t, err)

	root := dir{
		"a": file{c: "x", ts: ts, inaccessible: true, hashFromCache: 42},
	}

	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	cache := &Dir{
		Name: want.Root.Name,
		Files: []*File{
			{
				Name:    "a",
				Size:    1,
				ModTime: ts.Unix(),
				Hash:    42,
			},
		},
	}
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	res.assertEqual(t, want)
}

func Test__cache_entry_with_hash_0_is_ignored_and_logged(t *testing.T) {
	ts, err := time.Parse(time.Layout, time.Layout)
	require.NoError(t, err)

	root := dir{
		"d": file{c: "x\n", ts: ts},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	cache := &Dir{
		Name: want.Root.Name,
		Files: []*File{
			{
				Name:    "d",
				Size:    2,         // size is correct,
				ModTime: ts.Unix(), // time is correct,
				Hash:    0,         // but value 0 is explicitly ignored
			},
		},
	}
	logs := CaptureLogs(t)
	res, err := Run(rootPath, NoSkip, cache)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("warning: cached hash value 0 of file %q ignored"),
			filepath.Join(rootPath, "d"),
		),
		logs.String(),
	)
}

func Test__hash_computed_as_0_is_logged(t *testing.T) {
	root := dir{
		// Contents hash to 0 (https://md5hashing.net/hash/fnv1a64/0000000000000000).
		"hash0": file{c: "77kepQFQ8Kl"},
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := simulateScan(root, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines("info: hash of file %q evaluated to 0 - this might result in warnings (which can be safely ignored) if the output is used as cache in future scans"),
			filepath.Join(rootPath, "hash0"),
		),
		logs.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__root_symlink_is_followed_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}

	symlinkedDir := dir{
		"a": file{c: "z\n"},
		"g": file{},
	}
	symlinkName := "symlink"
	rootName := "data"
	wrap := dir{
		symlinkName: symlink(rootName),
		rootName:    symlinkedDir,
	}
	wrapPath := tempDir(t)
	wrap.writeTestdata(t, wrapPath)
	rootSymlinkPath := filepath.Join(wrapPath, symlinkName)
	rootPath := filepath.Join(wrapPath, rootName)
	want := simulateScan(symlinkedDir, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootSymlinkPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(Lines("following root symlink %q to %q"), rootSymlinkPath, rootPath),
		logs.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__root_indirect_symlink_is_followed_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}

	symlinkedDir := dir{
		"a": file{c: "z\n"},
		"g": file{},
	}
	symlinkName := "indirect-symlink"
	rootName := "data"
	wrap := dir{
		symlinkName: symlink("symlink"),
		"symlink":   symlink(rootName),
		rootName:    symlinkedDir,
	}
	wrapPath := tempDir(t)
	wrap.writeTestdata(t, wrapPath)
	rootSymlinkPath := filepath.Join(wrapPath, symlinkName)
	rootPath := filepath.Join(wrapPath, rootName)
	want := simulateScan(symlinkedDir, rootPath)

	logs := CaptureLogs(t)
	res, err := Run(rootSymlinkPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(Lines("following root symlink %q to %q"), rootSymlinkPath, rootPath),
		logs.String(),
	)
}

// SKIPPED on Windows unless running as administrator.
func Test__internal_symlink_is_skipped_and_logged(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}

	symlinkName := "symlink"

	tests := []struct {
		name string
		root dir
	}{
		{
			name: "existing target",
			root: dir{
				"a":         file{c: "z\n"},
				symlinkName: symlink("a"),
			},
		},
		{
			name: "non-existing target",
			root: dir{
				symlinkName: symlink("x"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rootPath := tempDir(t)
			test.root.writeTestdata(t, rootPath)
			want := simulateScan(test.root, rootPath)
			logs := CaptureLogs(t)
			res, err := Run(rootPath, NoSkip, nil)
			require.NoError(t, err)
			res.assertEqual(t, want)
			assert.Equal(t,
				fmt.Sprintf(
					Lines("skipping symlink %q during scan"),
					filepath.Join(rootPath, symlinkName),
				),
				logs.String(),
			)
		})
	}
}

// SKIPPED on Windows unless running as administrator.
func Test__root_symlink_to_ancestor_is_followed_but_skipped_and_logged_when_internal(t *testing.T) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" && !IsWindowsAdministrator() {
		t.Skip("Creating symlinks on Windows requires elevated privileges.")
	}
	symlinkName := "parent-symlink"
	symlinkedDir := dir{
		"a":         file{c: "z\n"},
		"g":         file{},
		symlinkName: symlink(".."), // points to root
	}
	symlinkTargetName := "f"
	root := dir{
		symlinkTargetName: symlinkedDir,
	}
	rootPath := tempDir(t)
	root.writeTestdata(t, rootPath)
	want := &Result{
		TypeVersion: CurrentResultTypeVersion,
		Root: &Dir{
			Name: rootPath,
			Dirs: []*Dir{
				symlinkedDir.simulateScan(symlinkTargetName),
			},
		},
	}

	logs := CaptureLogs(t)
	rootSymlinkPath := filepath.Join(rootPath, symlinkTargetName, symlinkName)
	res, err := Run(rootSymlinkPath, NoSkip, nil)
	require.NoError(t, err)
	res.assertEqual(t, want)
	assert.Equal(t,
		fmt.Sprintf(
			Lines(
				"following root symlink %q to %q",
				"skipping symlink %q during scan",
			),
			rootSymlinkPath,
			rootPath,
			rootSymlinkPath,
		),
		logs.String(),
	)
}

/* UTILITIES */

// tempDir constructs a new temporary directory and evaluates any symbolic links in the path.
// The directory is constructed using T.TempDir() to bind it to the lifetime of a test case.
// On some systems, the path returned by this function includes symlinks.
// This is the case for macOS and the Windows runners on GitHub Actions
// (the returned path 'C:\Users\RUNNER~1\AppData\Local\Temp\...'
// somehow resolves to 'C:\Users\runneradmin\AppData\Local\Temp\...').
// When passing such a path to Run, it will emit a log entry that the link has been followed
// and thus break tests that make assertions about log output.
// evaluating the links up front prevents this problem without breaking anything else.
func tempDir(t *testing.T) string {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return dir
}

func makeSkip(names ...string) ShouldSkipPath {
	return func(dir, name string) bool {
		for _, n := range names {
			if n == name {
				return true
			}
		}
		return false
	}
}
