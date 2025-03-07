package testutil

import (
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Consider using build flags instead of checking OS on runtime (would allow using 'golang.org/x/sys/windows').
// TODO: Would it be more idiomatic to return a cleanup function for reverting the change rather than doing it explicitly?

// MakeInaccessible makes the file or directory at the provided path non-readable to the user
// running the test.
// On Unix, this is done by zeroing out the permission bits.
// On Windows, that method can only be used to control the "write" flag (https://golang.org/pkg/os/#Chmod),
// so we invoke 'icacls' instead to deny access (https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/icacls).
// The function is only intended to be used on temporary files that
// get deleted as part of cleaning up after the test.
// Files and (except on Windows) directories being inaccessible
// don't prevent their deletion on any of the tested platforms.
//
// This function used to have separate variants for files and directories along with counterparts for reverting the change.
// These were deemed unnecessary after the introduction of testing.T.TempDir() (in Go 1.15),
// as that seems able to clean up properly without it.
func MakeInaccessible(path string) (func() error, error) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		u, err := user.Current()
		if err != nil {
			return nil, errors.Wrapf(err, "cannot resolve current Windows user")
		}
		// Deprecated variant - kept here in case we ever encounter a box that's unable to use 'icacls'.
		//cmd := exec.Command("cacls", path, "/e", "/d", u.Username)
		cmd := exec.Command("icacls", path, "/deny", u.Username+":r")
		cleanup := func() error {
			// Deprecated variant - kept here in case we ever encounter a box that's unable to use 'icacls'.
			//cmd := exec.Command("cacls", dirPath, "/e", "/g", u.Username+":f")
			cmd := exec.Command("icacls", path, "/grant", u.Username+":f")
			return runCommand(cmd)
		}
		return cleanup, runCommand(cmd)
	}

	// Not Windows.
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	mode := info.Mode()
	cleanup := func() error {
		return os.Chmod(path, mode)
	}
	return cleanup, os.Chmod(path, 0)
}

// MakeInaccessibleT wraps MakeInaccessible with error handling and automatic cleanup using the provided T object.
func MakeInaccessibleT(t *testing.T, path string) {
	cleanup, err := MakeInaccessible(path)
	require.NoErrorf(t, err, "cannot make file or directory %q inaccessible", path)
	if cleanup != nil {
		t.Cleanup(func() {
			err := cleanup()
			assert.NoErrorf(t, err, "cannot restore accessibility of file or directory %q", path)
		})
	}
}

func runCommand(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	return errors.Wrapf(err, string(out))
}
