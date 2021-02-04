package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__parsed_ShouldSkipPath_empty_always_returns_false(t *testing.T) {
	f, err := parseSkipNames("")
	require.NoError(t, err)

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
			skip := f(test.dirName, test.baseName)
			assert.False(t, skip)
		})
	}
}

func Test__parsed_ShouldSkipPath_nonempty_returns_true_on_basename_match(t *testing.T) {
	f, err := parseSkipNames("a,b")
	require.NoError(t, err)

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
			skip := f(test.dirName, test.baseName)
			assert.Equal(t, test.want, skip)
		})
	}
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
			_, err := parseSkipNames(test.names)
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
	assert.EqualError(t, err, `cannot load cache file "missing": cannot open file: file not found`)
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
	_, err = f.WriteString("{")
	require.NoError(t, err)

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
