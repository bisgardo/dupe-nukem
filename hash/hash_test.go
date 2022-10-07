package hash

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__hash_reader(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("x\n")
	res, err := Reader(&buf)
	require.NoError(t, err)
	assert.Equal(t, uint64(644258871406045975), res)
}

func Test__hash_file(t *testing.T) {
	res, err := File("testdata/a")
	require.NoError(t, err)
	assert.Equal(t, uint64(644258871406045975), res)
}

// TODO Test other kinds of (broken) files.
func Test__hash_dir_fails(t *testing.T) {
	_, err := File("testdata")
	assert.EqualError(t, err, "read error after 0 bytes: read testdata: is a directory")
}
