package main

import (
	"bytes"
	"compress/gzip"
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
	. "github.com/bisgardo/dupe-nukem/testutil"
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
	tests := []string{"@testdata/skipnames", "@testdata/skipnames_crlf"}
	want := []string{"a", "b", "c"}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			res, err := parseSkipNames(test)
			require.NoError(t, err)
			assert.Equal(t, want, res)
		})
	}
}

func Test__parseSkipNames_file_with_length_255_is_allowed(t *testing.T) {
	expr := strings.Repeat("x", maxSkipNameFileLineLen-1)
	path := TempStringFile(t, expr)
	input := fmt.Sprintf("@%v", path)
	want := []string{expr}
	res, err := parseSkipNames(input)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

// The tests below use loadShouldSkip because validation is performed after parseSkipNames.
// The ones above use parseSkipNames because it returns a slice which is easier to assert against
// in the valid cases.

func Test__loadShouldSkip_file_with_length_256_fails(t *testing.T) {
	expr := strings.Repeat("x", maxSkipNameFileLineLen) + "\n" // let the 256'th character be a newline
	path := TempStringFile(t, expr)
	input := fmt.Sprintf("@%v", path)
	_, err := loadShouldSkip(input)
	assert.EqualError(t,
		err,
		fmt.Sprintf(
			"cannot read skip names from file %q: line 1 is longer than the max allowed length of 256 characters",
			path,
		),
	)
}

func Test__loadShouldSkip_file_with_invalid_line_fails(t *testing.T) {
	expr := "with/slash"
	path := TempStringFile(t, expr)
	input := fmt.Sprintf("@%v", path)
	_, err := loadShouldSkip(input)
	assert.EqualError(t, err, fmt.Sprintf(`invalid skip name %q: invalid character '/'`, expr))
}

func Test__loadShouldSkip_invalid_names_fail(t *testing.T) {
	tests := []struct {
		names   string
		wantErr string
	}{
		{names: " x", wantErr: `invalid skip name " x": surrounding space`},
		{names: "x ", wantErr: `invalid skip name "x ": surrounding space`},
		{names: ".", wantErr: `invalid skip name ".": current directory`},
		{names: "..", wantErr: `invalid skip name "..": parent directory`},
		{names: "/", wantErr: `invalid skip name "/": invalid character '/'`},
		{names: "x,/y", wantErr: `invalid skip name "/y": invalid character '/'`},
		{names: ",", wantErr: `invalid skip name "": empty`},
	}

	for _, test := range tests {
		t.Run(test.names, func(t *testing.T) {
			_, err := loadShouldSkip(test.names)
			assert.EqualError(t, err, test.wantErr)
		})
	}
}

func Test__backslash_is_invalid_in_skip_name_on_windows_only(t *testing.T) {
	_, containsBackslash := invalidSkipNameChars['\\']
	assert.Equal(t, runtime.GOOS == "windows", containsBackslash)
	assert.Contains(t, invalidSkipNameChars, '/') // '/' is invalid on all systems
}

func Test__regex_characters_are_invalid_in_skip_name(t *testing.T) {
	assert.Contains(t, invalidSkipNameChars, '*')
	assert.Contains(t, invalidSkipNameChars, '?')
}

func Test__Scan_wraps_skip_file_not_found_error(t *testing.T) {
	_, err := Scan("x", "@missing", "")
	assert.EqualError(t, err, `cannot process skip dirs expression "@missing": cannot read skip names from file "missing": cannot open file: not found`)
}

func Test__Scan_wraps_parse_error_of_skip_names(t *testing.T) {
	_, err := Scan("x", "valid, it's not", "")
	assert.EqualError(t, err, `cannot process skip dirs expression "valid, it's not": invalid skip name " it's not": surrounding space`)
}

func Test__loadCacheDir_empty_loads_nil(t *testing.T) {
	res, err := loadScanCache("")
	require.NoError(t, err)
	assert.Nil(t, res)
}

func Test__loadScanDirCacheFile_logs_file_before_and_after_loading(t *testing.T) {
	f := "testdata/cache2.json.gz"
	logs := CaptureLogs(t)
	_, err := loadScanCache(f)
	require.NoError(t, err)
	ls := strings.Split(logs.String(), "\n")
	assert.Len(t, ls, 3)
	assert.Equal(t, `loading scan cache file "testdata/cache2.json.gz"...`, ls[0])
	assert.Regexp(t, `^scan cache loaded successfully from "testdata/cache2.json.gz" in [\w.]+s$`, ls[1])
	assert.Empty(t, ls[2])
}

func Test__loadScanDirCacheFile_logs_nonexistent_file_before_loading(t *testing.T) {
	f := "testdata/nonexistent-cache"
	logs := CaptureLogs(t)
	_, err := loadScanCache(f)
	assert.EqualError(t, err, "cannot open file: not found")
	assert.Equal(t,
		fmt.Sprintf(
			Lines("loading scan cache file %q..."),
			"testdata/nonexistent-cache",
		),
		logs.String(),
	)
}

func Test__loadScanDirResultFile_empty(t *testing.T) {
	path := TempStringFile(t, "")
	_, err := loadScanCache(path)
	assert.EqualError(t, err, `invalid JSON: EOF`)
}

func Test__loadScanCacheFile_error(t *testing.T) {
	type obj map[string]interface{}

	tests := []struct {
		name     string
		contents obj
		wantErr  string
	}{
		{
			name: "no version",
			contents: obj{
				"root": scan.Dir{Name: "xyz"},
			},
			wantErr: `schema version is missing`,
		}, {
			name: "unsupported version",
			contents: obj{
				"schema_version": scan.CurrentResultTypeVersion + 1,
				"root":           &scan.Dir{Name: "xyz"},
			},
			wantErr: fmt.Sprintf("unsupported schema version: %d", scan.CurrentResultTypeVersion+1),
		}, {
			name: "wrong version type",
			contents: obj{
				"schema_version": "xyz",
				"root":           &scan.Dir{Name: "xyz"},
			},
			wantErr: `cannot decode field "schema_version" of type "int" with value of type "string"`,
		}, {
			name: "no root",
			contents: obj{
				"schema_version": scan.CurrentResultTypeVersion,
			},
			wantErr: `no root`,
		}, {
			name: "null root",
			contents: obj{
				"schema_version": scan.CurrentResultTypeVersion,
				"root":           nil,
			},
			wantErr: `no root`,
		}, {
			name: "wrong root type",
			contents: obj{
				"schema_version": scan.CurrentResultTypeVersion,
				"root":           "xyz",
			},
			wantErr: `cannot decode field "root" of type "scan.Dir" with value of type "string"`,
		}, {
			name: "no root name",
			contents: obj{
				"schema_version": scan.CurrentResultTypeVersion,
				"root":           obj{},
			},
			wantErr: `invalid root: directory has no name`,
		}, {
			name: "empty root name",
			contents: obj{
				"schema_version": scan.CurrentResultTypeVersion,
				"root":           obj{"name": ""},
			},
			wantErr: `invalid root: directory has no name`,
		}, {
			name: "wrong root name type",
			contents: obj{
				"schema_version": scan.CurrentResultTypeVersion,
				"root":           obj{"name": 123},
			},
			wantErr: `cannot decode field "root.name" of type "string" with value of type "number"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bs, err := json.Marshal(test.contents)
			require.NoError(t, err)
			path := TempStringFile(t, string(bs))
			_, err = loadScanCache(path)
			assert.EqualError(t, err, test.wantErr)
		})
	}
}

func Test__loadScanDirCacheFile_wraps_invalid_cache_error(t *testing.T) {
	bs, err := json.Marshal(scan.Result{
		TypeVersion: scan.CurrentResultTypeVersion,
		Root:        &scan.Dir{Name: ""},
	})
	require.NoError(t, err)
	path := TempStringFile(t, string(bs))
	_, err = loadScanCache(path)
	assert.EqualError(t, err, `invalid root: directory has no name`)
}

func Test__Scan_wraps_invalid_dir_error(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)
	_, err = Scan(string([]byte{0}), "", "")
	want := fmt.Sprintf(`invalid root directory "%s/\x00": invalid argument (lstat)`, dir)
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		// On Windows this case actually tests that Scan does *not* wrap "unable to resolve absolute path" error.
		want = `cannot resolve absolute path of "\x00": invalid argument`
	}
	assert.EqualError(t, err, want)
}

func Test__Scan_wraps_cache_file_not_found_error(t *testing.T) {
	_, err := Scan("x", "", "missing")
	assert.EqualError(t, err, `cannot load scan cache file "missing": cannot open file: not found`)
}

func Test__Scan_wraps_cache_file_not_accessible_error(t *testing.T) {
	path := TempStringFile(t, "")
	MakeInaccessibleT(t, path)
	_, err := Scan("x", "", path)
	assert.EqualError(t, err, fmt.Sprintf("cannot load scan cache file %q: cannot open file: access denied", path))
}

func Test__Scan_wraps_cache_load_error(t *testing.T) {
	path := TempStringFile(t, "{")
	_, err := Scan("x", "", path)
	assert.EqualError(t, err, fmt.Sprintf("cannot load scan cache file %q: invalid JSON: unexpected EOF", path))
}

func Test__checkCache_rejects_unsorted_lists_for_nonempty_items(t *testing.T) {
	makeTestdata := func() *scan.Dir {
		return &scan.Dir{
			Name: "x",
			Dirs: []*scan.Dir{
				{
					Name:       "y",
					Files:      []*scan.File{{Name: "a", Size: 1, ModTime: 11, Hash: 1}, {Name: "b", Size: 1, ModTime: 21, Hash: 1}},
					EmptyFiles: []string{"c", "d", "e"},
				},
				{
					Name: "z",
					Dirs: []*scan.Dir{{Name: "r"}, {Name: "s"}, {Name: "t"}},
				},
			},
			Files:      []*scan.File{{Name: "a", Size: 1, ModTime: 42, Hash: 1}, {Name: "b", Size: 1, ModTime: 53, Hash: 1}, {Name: "c", Size: 1, ModTime: 69, Hash: 1}},
			EmptyFiles: []string{"c", "d"},
		}
	}

	t.Run("correctly ordered", func(t *testing.T) {
		err := checkCacheRoot(makeTestdata())
		assert.NoError(t, err)
	})
	t.Run("invalid order of empty files is accepted", func(t *testing.T) {
		d := makeTestdata()
		d.EmptyFiles[0], d.EmptyFiles[1] = d.EmptyFiles[1], d.EmptyFiles[0]
		err := checkCacheRoot(d)
		assert.NoError(t, err)
	})
	t.Run("invalid order of non-empty files", func(t *testing.T) {
		d := makeTestdata()
		d.Files[1], d.Files[2] = d.Files[2], d.Files[1]
		err := checkCacheRoot(d)
		assert.EqualError(t, err, `list of non-empty files in directory "x" is not sorted: "b" on index 2 should come before "c" on index 1`)
	})
	t.Run("invalid order of subdirs", func(t *testing.T) {
		d := makeTestdata()
		d.Dirs[0], d.Dirs[1] = d.Dirs[1], d.Dirs[0]
		err := checkCacheRoot(d)
		assert.EqualError(t, err, `list of subdirectories of "x" is not sorted: "y" on index 1 should come before "z" on index 0`)
	})
	t.Run("invalid order of nested empty files is accepted", func(t *testing.T) {
		d := makeTestdata()
		d0 := d.Dirs[0]
		d0.EmptyFiles[1], d0.EmptyFiles[2] = d0.EmptyFiles[2], d0.EmptyFiles[1]
		err := checkCacheRoot(d)
		assert.NoError(t, err)
	})
	t.Run("invalid order of nested files", func(t *testing.T) {
		d := makeTestdata()
		d0 := d.Dirs[0]
		d0.Files[0], d0.Files[1] = d0.Files[1], d0.Files[0]
		err := checkCacheRoot(d)
		assert.EqualError(t, err, `in subdirectory "y" on index 0: list of non-empty files in directory "y" is not sorted: "a" on index 1 should come before "b" on index 0`)
	})
	t.Run("invalid order of nested subdirs", func(t *testing.T) {
		d := makeTestdata()
		d1 := d.Dirs[1]
		d1.Dirs[1], d1.Dirs[2] = d1.Dirs[2], d1.Dirs[1]
		err := checkCacheRoot(d)
		assert.EqualError(t, err, `in subdirectory "z" on index 1: list of subdirectories of "z" is not sorted: "s" on index 2 should come before "t" on index 1`)
	})
}

func Test__checkCache_rejects_nonempty_file_with_size_0(t *testing.T) {
	err := checkCacheRoot(&scan.Dir{
		Name:  "x",
		Files: []*scan.File{{Name: "a", Size: 0, ModTime: 23, Hash: 1}},
	})
	assert.EqualError(t, err, `file "a" on index 0 has size 0, but is not listed as empty`)
}

func Test__checkCache_rejects_nonempty_file_with_empty_name(t *testing.T) {
	err := checkCacheRoot(&scan.Dir{
		Name:  "x",
		Files: []*scan.File{{Name: "", Size: 1, ModTime: 33, Hash: 1}},
	})
	assert.EqualError(t, err, `file on index 0 has no name`)
}

func Test__checkCache_rejects_dir_with_empty_name(t *testing.T) {
	t.Run("root directory with empty name", func(t *testing.T) {
		cache := &scan.Dir{Name: ""}
		err := checkCacheRoot(cache)
		assert.EqualError(t, err, `directory has no name`)
	})
	t.Run("nested directory name with empty name", func(t *testing.T) {
		cache := &scan.Dir{
			Name: "x",
			Dirs: []*scan.Dir{
				{Name: "y"},
				{Name: "z", Dirs: []*scan.Dir{{Name: ""}}},
			},
		}
		err := checkCacheRoot(cache)
		assert.EqualError(t, err, `in subdirectory "z" on index 1: in subdirectory "" on index 0: directory has no name`)
	})
}

func Test__checkCache_logs_warning_on_hash_0(t *testing.T) {
	// We shouldn't reject this as it could theoretically come from a file that actually hashes to zero.
	logs := CaptureLogs(t)
	err := checkCacheRoot(&scan.Dir{
		Name:  "x",
		Files: []*scan.File{{Name: "a", Size: 1, ModTime: 19, Hash: 0}},
	})
	require.NoError(t, err)
	assert.Equal(t, Lines("warning: file \"a\" is cached with hash 0 - this hash will be recomputed"), logs.String())
}

func Test__scan_testdata(t *testing.T) {
	absRootPath, err := filepath.Abs("testdata")
	require.NoError(t, err)

	want := &scan.Result{
		TypeVersion: scan.CurrentResultTypeVersion,
		Root: &scan.Dir{
			Name: absRootPath,
			Files: []*scan.File{
				{Name: ".gitattributes", Size: 8, ModTime: ModTime(t, "./testdata/.gitattributes"), Hash: 14181289122033052373},
				{Name: "cache1.json", Size: 297, ModTime: ModTime(t, "./testdata/cache1.json"), Hash: 4470884388509523918},
				{Name: "cache2.json.gz", Size: 80, ModTime: ModTime(t, "./testdata/cache2.json.gz"), Hash: 921101782703557466},
				{Name: "skipnames", Size: 7, ModTime: ModTime(t, "./testdata/skipnames"), Hash: 10951817445047336725},
				{Name: "skipnames_crlf", Size: 11, ModTime: ModTime(t, "./testdata/skipnames_crlf"), Hash: 15953509558814875971},
			},
		},
	}

	roots := map[string]struct{}{
		"testdata":    {},
		"testdata/":   {},
		"./testdata":  {},
		"./testdata/": {},
	}
	// Include OS-specific path separation.
	for root := range roots {
		// It's fine to modify the map while iterating it (https://go.dev/ref/spec#For_range):
		// added entries may or may not get visited by the loop but that doesn't matter
		// as they'll just re-add themselves.
		roots[filepath.FromSlash(root)] = struct{}{}
	}

	for root := range roots {
		res, err := Scan(root, "", "")
		require.NoError(t, err)
		assert.Equal(t, want, res)
	}
}

func Test__scan_logs_absolute_path_of_relative_dir(t *testing.T) {
	dir := "testdata"
	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	logs := CaptureLogs(t)
	_, err = Scan(dir, "", "")
	require.NoError(t, err)
	ls := strings.Split(logs.String(), "\n")
	assert.Len(t, ls, 3)
	assert.Equal(t, fmt.Sprintf("absolute path of %q resolved to %q", dir, absDir), ls[0])
	assert.Regexp(t, `^scan completed successfully in [\w.]+s$`, ls[1])
	assert.Empty(t, ls[2])
}

func Test__scan_does_not_log_absolute_dir_path(t *testing.T) {
	absDir, err := filepath.Abs("testdata")
	require.NoError(t, err)
	logs := CaptureLogs(t)
	_, err = Scan(absDir, "", "")
	require.NoError(t, err)
	ls := strings.Split(logs.String(), "\n")
	assert.Len(t, ls, 2)
	assert.Regexp(t, `^scan completed successfully in [\w.]+s$`, ls[0])
	assert.Empty(t, ls[1])
}

//goland:noinspection GoSnakeCaseUsage
func Test__scan_testdata_uses_provided_cache(t *testing.T) {
	modTime_gitattributes := ModTime(t, "./testdata/.gitattributes")
	modTime_cache1 := ModTime(t, "./testdata/cache1.json")
	modTime_cache2 := ModTime(t, "./testdata/cache2.json.gz")
	modTime_skipnames := ModTime(t, "./testdata/skipnames")
	modTime_skipnames_crlf := ModTime(t, "./testdata/skipnames_crlf")

	rootPath, err := filepath.Abs("./testdata")
	require.NoError(t, err)

	want := &scan.Result{
		TypeVersion: scan.CurrentResultTypeVersion,
		Root: &scan.Dir{
			Name: rootPath,
			Files: []*scan.File{
				{Name: ".gitattributes", Size: 8, ModTime: modTime_gitattributes, Hash: 14181289122033052373},   // not present in cache
				{Name: "cache1.json", Size: 297, ModTime: modTime_cache1, Hash: 69},                             // wrong hash loaded from cache
				{Name: "cache2.json.gz", Size: 80, ModTime: modTime_cache2, Hash: 921101782703557466},           // computed as cache didn't match
				{Name: "skipnames", Size: 7, ModTime: modTime_skipnames, Hash: 10951817445047336725},            // computed as cache didn't match
				{Name: "skipnames_crlf", Size: 11, ModTime: modTime_skipnames_crlf, Hash: 15953509558814875971}, // computed as cache didn't match (not actually present)
			},
		},
	}

	// Setup cache and write it to tmp file.
	cache := &scan.Result{
		TypeVersion: scan.CurrentResultTypeVersion,
		Root: &scan.Dir{
			Name: rootPath,
			Files: []*scan.File{
				// .gitattributes                                                              // not present
				{Name: "cache1.json", Size: 297, ModTime: modTime_cache1, Hash: 69},           // correct size and mod time
				{Name: "cache2.json.gz", Size: 666, ModTime: modTime_cache2, Hash: 69},        // incorrect size
				{Name: "skipnames", Size: 7, ModTime: 23, Hash: 69},                           // incorrect mod time
				{Name: "skipnames_clrs", Size: 11, ModTime: modTime_skipnames_crlf, Hash: 69}, // incorrect name
			},
		},
	}
	for _, compressCache := range []bool{false, true} {
		t.Run(fmt.Sprintf("compress:%t", compressCache), func(t *testing.T) {
			cacheBytes, err := json.MarshalIndent(cache, "", "  ")
			require.NoError(t, err)
			var pattern string
			if compressCache {
				pattern = "*.gz" // remove once 'resolveReader' uses magic number instead of relying on extension
				var buf bytes.Buffer
				w := gzip.NewWriter(&buf)
				_, err := w.Write(cacheBytes)
				require.NoError(t, err)
				err = w.Flush()
				require.NoError(t, err)
				cacheBytes = buf.Bytes()
			}
			cachePath := TempFileByPattern(t, pattern, cacheBytes)
			res, err := Scan(rootPath, "", cachePath)
			require.NoError(t, err)
			assert.Equal(t, want, res)
		})
	}
}
