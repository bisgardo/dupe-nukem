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
		{names: " x", wantErr: "invalid skip name \" x\": has surrounding space"},
		{names: "x ", wantErr: "invalid skip name \"x \": has surrounding space"},
		{names: ".", wantErr: "invalid skip name \".\": current directory"},
		{names: "..", wantErr: "invalid skip name \"..\": parent directory"},
		{names: "/", wantErr: "invalid skip name \"/\": has invalid character '/'"},
		{names: "x,/y", wantErr: "invalid skip name \"/y\": has invalid character '/'"},
		{names: ",", wantErr: "invalid skip name \"\": empty"},
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
	assert.EqualError(t, err, "cannot parse skip dirs expression \"valid, it's not\": invalid skip name \" it's not\": has surrounding space")
}

func Test__loadCacheDir_empty_loads_nil(t *testing.T) {
	res, err := loadCacheDir("")
	require.NoError(t, err)
	assert.Nil(t, res)
}

func Test__loadCacheDir_loads_scan_format(t *testing.T) {
	f := "testdata/cache.json"
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

func Test__Scan_wraps_cache_file_not_found_error(t *testing.T) {
	_, err := Scan("x", "", "missing")
	assert.EqualError(t, err, "cannot load cache file \"missing\": file not found")
}

func Test__Scan_wraps_cache_load_error(t *testing.T) {
	f, err := ioutil.TempFile("", "malformed")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(f.Name())
		require.NoError(t, err)
	}()
	_, err = f.WriteString("{")
	require.NoError(t, err)

	_, err = Scan("x", "", f.Name())
	assert.EqualError(t, err, fmt.Sprintf("cannot load cache file %q: cannot decode cache file as JSON: unexpected EOF", f.Name()))
}
