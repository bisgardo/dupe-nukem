package testutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify that TmpDir successfully cleans up inaccessible files and directories.
func Test__tmp_dir_cleanup(t *testing.T) {
	t.Run("inaccessible tmp directory", func(t *testing.T) {
		var rootPath string
		// TempDir failing to clean up will fail the test by itself,
		// but a little extra sanity check never hurt anyone.
		t.Cleanup(func() {
			_, err := os.Stat(rootPath)
			// Assertions work as expected within Cleanup.
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
		rootPath = t.TempDir()
		MakeInaccessibleT(t, rootPath)
	})

	t.Run("inaccessible file in tmp directory", func(t *testing.T) {
		var rootPath string
		// TempDir failing to clean up will fail the test by itself,
		// but a little extra sanity check never hurt anyone.
		t.Cleanup(func() {
			_, err := os.Stat(rootPath)
			// Assertions work as expected within Cleanup.
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
		rootPath = t.TempDir()
		filePath := filepath.Join(rootPath, "x")
		err := os.WriteFile(filePath, []byte("blah"), 0644)
		require.NoError(t, err)
		MakeInaccessibleT(t, filePath)
	})
}
