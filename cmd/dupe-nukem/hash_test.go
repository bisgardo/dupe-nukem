package main

import (
	"testing"

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
	assert.EqualError(t, err, "cannot hash file \"testdata\": is a directory")
}
