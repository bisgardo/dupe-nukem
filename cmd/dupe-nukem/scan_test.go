package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	f, err := ioutil.TempFile("", "allowed-skipnames")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	line := strings.Repeat("x", maxSkipNameFileLineLen-1)
	n, err := f.WriteString(line)
	require.NoError(t, err)
	require.Equal(t, maxSkipNameFileLineLen-1, n)

	input := fmt.Sprintf("@%v", f.Name())
	want := []string{line}
	res, err := parseSkipNames(input)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__loadShouldSkip_file_with_length_256_fails(t *testing.T) {
	f, err := ioutil.TempFile("", "long-skipnames")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString(strings.Repeat("x", maxSkipNameFileLineLen) + "\n") // let the 256'th character be a newline
	require.NoError(t, err)
	require.Equal(t, maxSkipNameFileLineLen+1, n)

	input := fmt.Sprintf("@%v", f.Name())
	_, err = loadShouldSkip(input)
	assert.EqualError(t, err, "line 1 is longer than the max allowed length of 256 characters")
}

func Test__loadShouldSkip_file_with_invalid_line_fails(t *testing.T) {
	f, err := ioutil.TempFile("", "invalid-skipnames")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString("with/slash")
	require.NoError(t, err)
	require.Equal(t, 10, n)

	input := fmt.Sprintf("@%v", f.Name())
	_, err = loadShouldSkip(input)
	assert.EqualError(t, err, `invalid skip name "with/slash": has invalid character '/'`)
}

func Test__cannot_parse_invalid_skip_names(t *testing.T) {
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

func Test__Scan_wraps_parse_error_of_skip_names(t *testing.T) {
	_, err := Scan("x", "valid, it's not", "")
	assert.EqualError(t, err, `cannot parse skip dirs expression "valid, it's not": invalid skip name " it's not": has surrounding space`)
}

func Test__loadCacheDir_empty_loads_nil(t *testing.T) {
	res, err := loadCacheDir("")
	require.NoError(t, err)
	assert.Nil(t, res)
}

func Test__loadCacheDir_loads_scan_format(t *testing.T) {
	f := "testdata/cache1.json"
	want := &scan.Dir{
		Name: "x",
		Dirs: []*scan.Dir{
			{
				Name: "y",
				Files: []*scan.File{
					{Name: "a", Size: 21, Hash: 42},
					{Name: "b", Size: 53, Hash: 0},
				},
			},
		},
		Files: []*scan.File{
			{Name: "c", Size: 11, Hash: 11},
		},
	}
	res, err := loadCacheDir(f)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__loadCacheDir_loads_compressed_scan_format(t *testing.T) {
	f := "testdata/cache2.json.gz"
	want := &scan.Dir{Name: "y"}
	res, err := loadCacheDir(f)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__Scan_wraps_cache_file_not_found_error(t *testing.T) {
	_, err := Scan("x", "", "missing")
	assert.EqualError(t, err, `cannot load cache file "missing": cannot open file: not found`)
}

func Test__Scan_wraps_cache_file_not_accessible_error(t *testing.T) {
	f, err := ioutil.TempFile("", "inaccessible")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	err = f.Chmod(0) // remove permissions
	require.NoError(t, err)
	_, err = Scan("x", "", f.Name())
	assert.EqualError(t, err, fmt.Sprintf("cannot load cache file %q: cannot open file: access denied", f.Name()))
}

func Test__Scan_wraps_cache_load_error(t *testing.T) {
	f, err := ioutil.TempFile("", "malformed")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	n, err := f.WriteString("{")
	require.NoError(t, err)
	require.Equal(t, 1, n)

	_, err = Scan("x", "", f.Name())
	assert.EqualError(t, err, fmt.Sprintf("cannot load cache file %q: cannot decode file as JSON: unexpected EOF", f.Name()))
}

func Test__checkCache(t *testing.T) {
	testdata := func() *scan.Dir {
		return &scan.Dir{
			Name: "x",
			Dirs: []*scan.Dir{
				{
					Name:       "y",
					Files:      []*scan.File{{Name: "a"}, {Name: "b"}},
					EmptyFiles: []string{"c", "d", "e"},
				},
				{
					Name: "z",
					Dirs: []*scan.Dir{{Name: "r"}, {Name: "s"}, {Name: "t"}},
				},
			},
			Files:      []*scan.File{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			EmptyFiles: []string{"c", "d"},
		}
	}

	t.Run("correctly ordered", func(t *testing.T) {
		err := checkCache(testdata())
		assert.NoError(t, err)
	})
	t.Run("invalid order of empty files", func(t *testing.T) {
		d := testdata()
		d.EmptyFiles[0], d.EmptyFiles[1] = d.EmptyFiles[1], d.EmptyFiles[0]
		err := checkCache(d)
		assert.EqualError(t, err, `list of empty files in directory "x" is not sorted: "c" on index 1 should come before "d" on index 0`)
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
	t.Run("invalid order of nested empty files", func(t *testing.T) {
		d := testdata()
		d0 := d.Dirs[0]
		d0.EmptyFiles[1], d0.EmptyFiles[2] = d0.EmptyFiles[2], d0.EmptyFiles[1]
		err := checkCache(d)
		assert.EqualError(t, err, `in subdirectory "y" on index 0: list of empty files in directory "y" is not sorted: "d" on index 2 should come before "e" on index 1`)
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

func Test__scan_testdata(t *testing.T) {
	root := "testdata"
	want := &scan.Dir{
		Name: "testdata",
		Files: []*scan.File{
			{Name: "cache1.json", Size: 232, Hash: 17698409774061682325},
			{Name: "cache2.json.gz", Size: 34, Hash: 11617732806245318878},
			{Name: "skipnames", Size: 7, Hash: 10951817445047336725},
		},
	}
	res, err := Scan(root, "", "")
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__scan_testdata_with_trailing_slash(t *testing.T) {
	root := "testdata/"
	want := &scan.Dir{
		Name: "testdata",
		Files: []*scan.File{
			{Name: "cache1.json", Size: 232, Hash: 17698409774061682325},
			{Name: "cache2.json.gz", Size: 34, Hash: 11617732806245318878},
			{Name: "skipnames", Size: 7, Hash: 10951817445047336725},
		},
	}
	res, err := Scan(root, "", "")
	require.NoError(t, err)
	assert.Equal(t, want, res)
}
