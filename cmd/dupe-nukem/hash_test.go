package main

import (
	"fmt"
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
	path := testutil.TempFile(t, "")
	testutil.MakeInaccessibleT(t, path)
	_, err := Hash(path)
	assert.EqualError(t, err, fmt.Sprintf("cannot hash file %q: cannot open file: access denied", path))
}
