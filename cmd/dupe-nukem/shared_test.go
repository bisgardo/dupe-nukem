package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/bisgardo/dupe-nukem/scan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__resolveReader_rejects_invalid_compressed_scan_file(t *testing.T) {
	f, err := ioutil.TempFile("", "invalid-*.gz") // the '*' is swapped out for gibberish instead of it being appended after the '.gz'
	require.NoError(t, err)
	defer func() {
		err := f.Close()
		assert.NoError(t, err)
		err = os.Remove(f.Name())
		assert.NoError(t, err)
	}()
	_, err = f.WriteString("totally legit compression")
	require.NoError(t, err)
	_, err = f.Seek(0, 0)
	require.NoError(t, err)

	_, err = resolveReader(f)
	assert.EqualError(t, err, "gzip: invalid header")
}

func Test__loadScanDirFile_loads_scan_file(t *testing.T) {
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
	res, err := loadScanDirFile(f)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__loadScanDirFile_loads_compressed_scan_file(t *testing.T) {
	f := "testdata/cache2.json.gz"
	want := &scan.Dir{Name: "y"}
	res, err := loadScanDirFile(f)
	require.NoError(t, err)
	assert.Equal(t, want, res)
}

func Test__loadScanDirFile_wraps_scan_file_error(t *testing.T) {
	f, err := ioutil.TempFile("", "invalid-*.gz") // the '*' is swapped out for gibberish instead of it being appended after the '.gz'
	require.NoError(t, err)
	defer func() {
		err := f.Close()
		assert.NoError(t, err)
		err = os.Remove(f.Name())
		assert.NoError(t, err)
	}()

	_, err = loadScanDirFile(f.Name())
	assert.EqualError(t, err, "cannot resolve file reader: EOF")
}
