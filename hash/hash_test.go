package hash

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/bisgardo/dupe-nukem/testutil"

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

// TODO: Test other kinds of (broken) files.
func Test__hash_dir_fails(t *testing.T) {
	_, err := File("testdata")

	// The function should never be called with a directory (all current callers check this beforehand),
	// so it doesn't really matter that the error message sucks.
	//goland:noinspection GoBoolExpressions
	wantReason := "is a directory"
	if runtime.GOOS == "windows" {
		if runtime.Version() < "go1.20" {
			wantReason = "The handle is invalid."
		} else {
			// Incredible that they managed to change the message into an even less accurate one.
			wantReason = "Incorrect function."
		}
	}
	assert.EqualError(t, err, fmt.Sprintf("read error after 0 bytes: read testdata: %s", wantReason))
}

func Test__hash_inaccessible_file_fails(t *testing.T) {
	// This test is basically identical to 'Test__hash_wraps_file_error' (in package 'main'),
	// but the purpose is slightly different (as indicated by the test name).
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	filename := f.Name()
	defer func() {
		err := os.Remove(filename)
		require.NoError(t, err)
	}()
	err = testutil.MakeFileInaccessible(f)
	require.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)
	_, err = File(filename)
	assert.EqualError(t, err, "cannot open file: access denied")
}
