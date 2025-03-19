package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/bisgardo/dupe-nukem/testutil"
)

func Test__parseSkipNames_empty_returns_nil(t *testing.T) {
	input := ""
	res, err := parseSkipNames(input)
	require.NoError(t, err)
	assert.Nil(t, res)
}

func Test__parseSkipNames_splits_on_comma(t *testing.T) {
	input := "a,b"
	want := []string{"a", "b"}
	res, err := parseSkipNames(input)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__parseSkipNames_with_at_prefix_splits_file_on_newline(t *testing.T) {
	input := "@testdata/skipnames"
	want := []string{"a", "b", "c"}
	res, err := parseSkipNames(input)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__parseSkipNames_file_with_length_255_is_allowed(t *testing.T) {
	f, err := os.CreateTemp("", "allowed-skipnames")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	line := strings.Repeat("x", maxSkipNameFileLineLen-1)
	n, err := f.WriteString(line)
	require.NoError(t, err)
	require.Equal(t, maxSkipNameFileLineLen-1, n)
	err = f.Close()
	assert.NoError(t, err)

	input := fmt.Sprintf("@%v", f.Name())
	want := []string{line}
	res, err := parseSkipNames(input)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

// The tests below use loadShouldSkip because validation is performed after parseSkipNames.
// The ones above use parseSkipNames because it returns a slice which is easier to assert against
// in the valid cases.

func Test__loadShouldSkip_file_with_length_256_fails(t *testing.T) {
	f, err := os.CreateTemp("", "long-skipnames")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString(strings.Repeat("x", maxSkipNameFileLineLen) + "\n") // let the 256'th character be a newline
	require.NoError(t, err)
	require.Equal(t, maxSkipNameFileLineLen+1, n)
	err = f.Close()
	assert.NoError(t, err)

	input := fmt.Sprintf("@%v", f.Name())
	_, err = loadShouldSkip(input)
	assert.EqualError(t, err, fmt.Sprintf("cannot read skip names from file %q: line 1 is longer than the max allowed length of 256 characters", f.Name()))
}

func Test__loadShouldSkip_file_with_invalid_line_fails(t *testing.T) {
	f, err := os.CreateTemp("", "invalid-skipnames")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString("with/slash")
	require.NoError(t, err)
	require.Equal(t, 10, n)
	err = f.Close()
	assert.NoError(t, err)

	input := fmt.Sprintf("@%v", f.Name())
	_, err = loadShouldSkip(input)
	assert.EqualError(t, err, `invalid skip name "with/slash": has invalid character '/'`)
}

func Test__loadShouldSkip_invalid_names_fail(t *testing.T) {
	tests := []struct {
		names   string
		wantErr string
	}{
		{names: " x", wantErr: `invalid skip name " x": has surrounding space`},
		{names: "x ", wantErr: `invalid skip name "x ": has surrounding space`},
		{names: ".", wantErr: `invalid skip name ".": current directory`},
		{names: "..", wantErr: `invalid skip name "..": parent directory`},
		{names: "/", wantErr: `invalid skip name "/": has invalid character '/'`},
		{names: "x,/y", wantErr: `invalid skip name "/y": has invalid character '/'`},
		{names: ",", wantErr: `invalid skip name "": empty`},
	}

	for _, test := range tests {
		t.Run(test.names, func(t *testing.T) {
			_, err := loadShouldSkip(test.names)
			assert.EqualError(t, err, test.wantErr)
		})
	}
}

func Test__Scan_wraps_skip_file_not_found_error(t *testing.T) {
	_, err := Scan("x", "@missing", "")
	assert.EqualError(t, err, `cannot process skip dirs expression "@missing": cannot read skip names from file "missing": cannot open file: not found`)
}

func Test__Scan_wraps_parse_error_of_skip_names(t *testing.T) {
	_, err := Scan("x", "valid, it's not", "")
	assert.EqualError(t, err, `cannot process skip dirs expression "valid, it's not": invalid skip name " it's not": has surrounding space`)
}

func Test__loadCacheDir_empty_loads_nil(t *testing.T) {
	res, err := loadScanDirCacheFile("")
	require.NoError(t, err)
	assert.Nil(t, res)
}

func Test__loadScanDirCacheFile_logs_file_before_and_after_loading(t *testing.T) {
	f := "testdata/cache2.json.gz"
	buf := testutil.LogBuffer()
	_, err := loadScanDirCacheFile(f)
	require.NoError(t, err)
	ls := strings.Split(buf.String(), "\n")
	assert.Len(t, ls, 3)
	assert.Equal(t, `loading scan cache file "testdata/cache2.json.gz"...`, ls[0])
	assert.Regexp(t, `^scan cache loaded successfully from "testdata/cache2.json.gz" in [\w.]+s$`, ls[1])
	assert.Empty(t, ls[2])
}

func Test__loadScanDirCacheFile_logs_nonexistent_file_before_loading(t *testing.T) {
	f := "testdata/nonexistent-cache"
	buf := testutil.LogBuffer()
	_, err := loadScanDirCacheFile(f)
	require.Error(t, err)
	ls := strings.Split(buf.String(), "\n")
	assert.Len(t, ls, 2)
	assert.Equal(t, `loading scan cache file "testdata/nonexistent-cache"...`, ls[0])
	assert.Empty(t, ls[1])
}

func Test__loadScanDirCacheFile_wraps_invalid_cache_error(t *testing.T) {
	f, err := os.CreateTemp("", "invalid")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	_, err = f.WriteString(`{"name":""}`)
	require.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	_, err = loadScanDirCacheFile(f.Name())
	assert.EqualError(t, err, `invalid cache contents: directory name is empty`)
}

func Test__Scan_wraps_invalid_dir_error(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)
	_, err = Scan(string([]byte{0}), "", "")
	want := fmt.Sprintf(`invalid root directory "%s/\x00": invalid argument (lstat)`, dir)
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		// On Windows this case actually test that Scan does *not* wrap "unable to resolve absolute path" error.
		want = `cannot resolve absolute path of "\x00": invalid argument`
	}
	assert.EqualError(t, err, want)
}

func Test__Scan_wraps_cache_file_not_found_error(t *testing.T) {
	_, err := Scan("x", "", "missing")
	assert.EqualError(t, err, `cannot load scan cache file "missing": cannot open file: not found`)
}

func Test__Scan_wraps_cache_file_not_accessible_error(t *testing.T) {
	f, err := os.CreateTemp("", "inaccessible")
	require.NoError(t, err)
	filename := f.Name()
	t.Cleanup(func() {
		err := os.Remove(filename)
		assert.NoError(t, err)
	})
	testutil.MakeInaccessibleT(t, filename)
	require.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)
	_, err = Scan("x", "", filename)
	assert.EqualError(t, err, fmt.Sprintf("cannot load scan cache file %q: cannot open file: access denied", filename))
}

func Test__Scan_wraps_cache_load_error(t *testing.T) {
	f, err := os.CreateTemp("", "malformed")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString("{")
	require.NoError(t, err)
	require.Equal(t, 1, n)
	err = f.Close()
	assert.NoError(t, err)

	_, err = Scan("x", "", f.Name())
	assert.EqualError(t, err, fmt.Sprintf("cannot load scan cache file %q: cannot decode file as JSON: unexpected EOF", f.Name()))
}

func Test__checkCache_rejects_unsorted_lists_for_nonempty_items(t *testing.T) {
	testdata := func() *scan.Dir {
		return &scan.Dir{
			Name: "x",
			Dirs: []*scan.Dir{
				{
					Name:       "y",
					Files:      []*scan.File{{Name: "a", Size: 1, Hash: 1}, {Name: "b", Size: 1, Hash: 1}},
					EmptyFiles: []string{"c", "d", "e"},
				},
				{
					Name: "z",
					Dirs: []*scan.Dir{{Name: "r"}, {Name: "s"}, {Name: "t"}},
				},
			},
			Files:      []*scan.File{{Name: "a", Size: 1, Hash: 1}, {Name: "b", Size: 1, Hash: 1}, {Name: "c", Size: 1, Hash: 1}},
			EmptyFiles: []string{"c", "d"},
		}
	}

	t.Run("correctly ordered", func(t *testing.T) {
		err := checkCache(testdata())
		assert.NoError(t, err)
	})
	t.Run("invalid order of empty files is accepted", func(t *testing.T) {
		d := testdata()
		d.EmptyFiles[0], d.EmptyFiles[1] = d.EmptyFiles[1], d.EmptyFiles[0]
		err := checkCache(d)
		assert.NoError(t, err)
	})
	t.Run("invalid order of non-empty files", func(t *testing.T) {
		d := testdata()
		d.Files[1], d.Files[2] = d.Files[2], d.Files[1]
		err := checkCache(d)
		assert.EqualError(t, err, `list of non-empty files in directory "x" is not sorted: "b" on index 2 should come before "c" on index 1`)
	})
	t.Run("invalid order of subdirs", func(t *testing.T) {
		d := testdata()
		d.Dirs[0], d.Dirs[1] = d.Dirs[1], d.Dirs[0]
		err := checkCache(d)
		assert.EqualError(t, err, `list of subdirectories of "x" is not sorted: "y" on index 1 should come before "z" on index 0`)
	})
	t.Run("invalid order of nested empty files is accepted", func(t *testing.T) {
		d := testdata()
		d0 := d.Dirs[0]
		d0.EmptyFiles[1], d0.EmptyFiles[2] = d0.EmptyFiles[2], d0.EmptyFiles[1]
		err := checkCache(d)
		assert.NoError(t, err)
	})
	t.Run("invalid order of nested files", func(t *testing.T) {
		d := testdata()
		d0 := d.Dirs[0]
		d0.Files[0], d0.Files[1] = d0.Files[1], d0.Files[0]
		err := checkCache(d)
		assert.EqualError(t, err, `in subdirectory "y" on index 0: list of non-empty files in directory "y" is not sorted: "a" on index 1 should come before "b" on index 0`)
	})
	t.Run("invalid order of nested subdirs", func(t *testing.T) {
		d := testdata()
		d1 := d.Dirs[1]
		d1.Dirs[1], d1.Dirs[2] = d1.Dirs[2], d1.Dirs[1]
		err := checkCache(d)
		assert.EqualError(t, err, `in subdirectory "z" on index 1: list of subdirectories of "z" is not sorted: "s" on index 2 should come before "t" on index 1`)
	})
}

func Test__checkCache_rejects_nonempty_file_with_size_0(t *testing.T) {
	err := checkCache(&scan.Dir{
		Name:  "x",
		Files: []*scan.File{{Name: "a", Size: 0, Hash: 1}},
	})
	assert.EqualError(t, err, `non-empty file "a" on index 0 has size 0`)
}

func Test__checkCache_rejects_nonempty_file_with_empty_name(t *testing.T) {
	err := checkCache(&scan.Dir{
		Name:  "x",
		Files: []*scan.File{{Name: "", Size: 1, Hash: 1}},
	})
	assert.EqualError(t, err, `name of non-empty file on index 0 is empty`)
}

func Test__checkCache_rejects_dir_with_empty_name(t *testing.T) {
	t.Run("root directory with empty name", func(t *testing.T) {
		testdata := &scan.Dir{Name: ""}
		err := checkCache(testdata)
		assert.EqualError(t, err, `directory name is empty`)
	})
	t.Run("nested directory name with empty name", func(t *testing.T) {
		testdata := &scan.Dir{
			Name: "x",
			Dirs: []*scan.Dir{
				{Name: "y"},
				{Name: "z", Dirs: []*scan.Dir{{Name: ""}}},
			},
		}
		err := checkCache(testdata)
		assert.EqualError(t, err, `in subdirectory "z" on index 1: in subdirectory "" on index 0: directory name is empty`)
	})
}

func Test__checkCache_logs_warning_on_hash_0(t *testing.T) {
	// We shouldn't reject this as it could theoretically come from a file that actually hashes to zero.
	buf := testutil.LogBuffer()
	err := checkCache(&scan.Dir{
		Name:  "x",
		Files: []*scan.File{{Name: "a", Size: 1, Hash: 0}},
	})
	require.NoError(t, err)
	assert.Equal(t, "warning: file \"a\" is cached with hash 0 - this hash will be ignored\n", buf.String())
}

func Test__scan_testdata(t *testing.T) {
	want := &scan.Dir{
		Name: "testdata",
		Files: []*scan.File{
			{Name: "cache1.json", Size: 232, Hash: 17698409774061682325, ModTime: modTime(t, "./testdata/cache1.json")},
			{Name: "cache2.json.gz", Size: 34, Hash: 11617732806245318878, ModTime: modTime(t, "./testdata/cache2.json.gz")},
			{Name: "skipnames", Size: 7, Hash: 10951817445047336725, ModTime: modTime(t, "./testdata/skipnames")},
		},
	}

	roots := []string{"testdata", "testdata/", "./testdata", "./testdata/"}
	for _, root := range roots {
		res, err := Scan(root, "", "")
		require.NoError(t, err)
		assert.Equal(t, want, res)
	}
}

func Test__scan_logs_absolute_path_of_relative_dir(t *testing.T) {
	dir := "testdata"
	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	buf := testutil.LogBuffer()
	_, err = Scan(dir, "", "")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("absolute path of %q resolved to %q\n", dir, absDir), buf.String())
}

func Test__scan_does_not_log_absolute_dir_path(t *testing.T) {
	absDir, err := filepath.Abs("testdata")
	require.NoError(t, err)
	buf := testutil.LogBuffer()
	_, err = Scan(absDir, "", "")
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

//goland:noinspection GoSnakeCaseUsage
func Test__scan_testdata_uses_provided_cache(t *testing.T) {
	modTime_cache1 := modTime(t, "./testdata/cache1.json")
	modTime_cache2 := modTime(t, "./testdata/cache2.json.gz")
	modTime_skipnames := modTime(t, "./testdata/skipnames")

	want := &scan.Dir{
		Name: "testdata",
		Files: []*scan.File{
			{Name: "cache1.json", Size: 232, Hash: 69, ModTime: modTime_cache1},                     // wrong hash loaded from cache
			{Name: "cache2.json.gz", Size: 34, Hash: 11617732806245318878, ModTime: modTime_cache2}, // computed as cache didn't match
			{Name: "skipnames", Size: 7, Hash: 10951817445047336725, ModTime: modTime_skipnames},    // computed as cache didn't match
		},
	}

	// Setup cache and write it to tmp file.
	cache := &scan.Dir{
		Name: "testdata",
		Files: []*scan.File{
			{Name: "cache1.json", Size: 232, Hash: 69, ModTime: modTime_cache1},   // correct size and mod time
			{Name: "cache2.json.gz", Size: 69, Hash: 69, ModTime: modTime_cache2}, // incorrect size
			{Name: "skipnames", Size: 7, Hash: 69, ModTime: 23},                   // incorrect mod time
		},
	}
	cachePath, err := os.CreateTemp("", "cache")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(cachePath.Name())
		assert.NoError(t, err)
	}()
	cacheBytes, err := json.MarshalIndent(cache, "", "  ")
	require.NoError(t, err)
	_, err = cachePath.Write(cacheBytes)
	require.NoError(t, err)

	res, err := Scan("testdata", "", cachePath.Name())
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

// UTILITIES

func modTime(t *testing.T, path string) int64 {
	// Load actual mod time of file.
	info, err := os.Lstat(path)
	require.NoError(t, err)
	return info.ModTime().Unix()
}
