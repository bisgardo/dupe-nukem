package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/scan"
	. "github.com/bisgardo/dupe-nukem/testutil"
)

func Test__resolveReader_rejects_invalid_compressed_scan_file(t *testing.T) {
	path := TempFileByPattern(t,
		"invalid-*.gz",                      // the '*' is swapped out for gibberish instead of it being appended after ".gz"
		[]byte("totally legit compression"), // spoiler alert: it's not!
	)
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() {
		err := f.Close()
		assert.NoError(t, err)
	}()
	_, err = resolveReader(f)
	assert.EqualError(t, err, "gzip: invalid header")
}

func Test__loadScanDirFile_loads_scan_file(t *testing.T) {
	f := "testdata/cache1.json"
	want := &scan.Result{
		TypeVersion: scan.CurrentResultTypeVersion,
		Root: &scan.Dir{
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
		},
	}
	res, err := loadScanResultFile(f)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__loadScanDirFile_loads_compressed_scan_file(t *testing.T) {
	f := "testdata/cache2.json.gz" // fun fact: uses CRLF when uncompressed (while cache1.json uses LF)
	want := &scan.Result{
		TypeVersion: scan.CurrentResultTypeVersion,
		Root:        &scan.Dir{Name: "y"},
	}
	res, err := loadScanResultFile(f)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__loadScanDirFile_wraps_scan_file_error(t *testing.T) {
	path := TempFileByPattern(t,
		"invalid-*.gz", // the '*' is swapped out for gibberish instead of it being appended after ".gz"
		nil,
	)
	_, err := loadScanResultFile(path)
	assert.EqualError(t, err, "cannot resolve file reader: EOF")
}
