package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/bisgardo/dupe-nukem/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__hash_stdin(t *testing.T) {
	// Go wires stdin to '/dev/null' in tests, so the result is the hash of the empty string.
	res, err := Hash("")
	require.NoError(t, err)
	assert.Equal(t, uint64(14695981039346656037), res)
}

func Test__hash_file(t *testing.T) {
	res, err := Hash("testdata/skipnames")
	require.NoError(t, err)
	assert.Equal(t, uint64(10951817445047336725), res)
}

func Test__hash_dir_fails(t *testing.T) {
	_, err := Hash("testdata")
	assert.EqualError(t, err, `cannot hash directory "testdata"`)
}

func Test__hash_nonexisting_file_fails(t *testing.T) {
	_, err := Hash("nonexisting/file")
	assert.EqualError(t, err, `cannot stat "nonexisting/file": not found`)
}

func Test__hash_wraps_file_error(t *testing.T) {
	// This test is basically identical to 'Test__hash_inaccessible_file_fails' (in package 'hash'),
	// but, as indicated by the test name, the purpose is slightly different:
	// Hashing an inaccessible file just happens to be the easiest way to trigger an error.
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	filename := f.Name()
	t.Cleanup(func() {
		err := os.Remove(filename)
		require.NoError(t, err)
	})
	testutil.MakeInaccessibleT(t, filename)
	require.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)
	_, err = Hash(filename)
	assert.EqualError(t, err, fmt.Sprintf("cannot hash file %q: cannot open file: access denied", filename))
}
