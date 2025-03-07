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
	t.Run("inaccessible root directory", func(t *testing.T) {
		var rootPath string
		// Sanity check. Isn't actually required as TempDir failing to clean up will fail the test by itself.
		t.Cleanup(func() {
			_, err := os.Stat(rootPath)
			// Assertions work as expected within Cleanup.
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})

		rootPath = t.TempDir()
		MakeInaccessibleT(t, rootPath)
	})
	t.Run("inaccessible file", func(t *testing.T) {
		var rootPath string
		// Sanity check. Isn't actually required as TempDir failing to clean up will fail the test by itself.
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
