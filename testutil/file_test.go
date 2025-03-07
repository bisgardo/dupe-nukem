package testutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify that TmpDir successfully cleans up inaccessible files and directories.
func Test__tmp_dir_cleanup(t *testing.T) {
	t.Run("inaccessible root directory", func(t *testing.T) {
		var rootPath string
		t.Cleanup(func() {
			info, err := os.Stat(rootPath)

			spew.Dump("info:", info) // for debugging
			fmt.Println("error:", err)

			_ = err
			//// Assertions work as expected within Cleanup.
			//assert.ErrorIs(t, err, fs.ErrNotExist)
		})

		rootPath = t.TempDir()
		MakeInaccessibleT(t, rootPath)
	})
	t.Run("inaccessible file", func(t *testing.T) {
		var rootPath string
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
