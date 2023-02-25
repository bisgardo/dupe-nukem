package testutil

import (
	"os"
	"os/exec"
	"os/user"
	"runtime"

	"github.com/pkg/errors"
)

// TODO Use build flags instead of checking OS on runtime?
// TODO Should ideally return cleanup function for reverting the change?

// MakeFileInaccessible makes the provided file non-readable to the user
// running the test.
// On Unix, this is done by zeroing the permission bits.
// On Windows, this method can only be used to control the "write" flag
// (https://golang.org/pkg/os/#Chmod), so we invoke 'icacls' to deny access.
// The function is only intended to be used on temporary files that
// get deleted as part of cleaning up after the test.
// On none of the tested platforms does it prevent deletion of the file
// that it has been made inaccessible.
func MakeFileInaccessible(f *os.File) error {
	if runtime.GOOS == "windows" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		// Deprecated - kept in case some box doesn't have icacls.
		//cmd := exec.Command("cacls", f.Name(), "/e", "/d", u.Username)
		cmd := exec.Command("icacls", f.Name(), "/deny", u.Username+":r")
		return run(cmd)
	}
	return f.Chmod(0)
}

// MakeDirInaccessible makes the directory at the provided path
// non-readable to the user running the test.
// On Unix, this is done by zeroing the permission bits.
// On Windows, this method can only be used to control the "write" flag
// (https://golang.org/pkg/os/#Chmod), so we invoke 'icacls' to deny access.
// The function is only intended to be used on temporary directories that
// get deleted as part of cleaning up after the test.
// Use MakeDirAccessible to make the directory deletable.
func MakeDirInaccessible(dirPath string) error {
	if runtime.GOOS == "windows" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		// Deprecated - kept in case some box doesn't have icacls.
		//cmd := exec.Command("cacls", dirPath, "/e", "/d", u.Username)
		cmd := exec.Command("icacls", dirPath, "/deny", u.Username+":r")
		return run(cmd)
	}
	return os.Chmod(dirPath, 0)
}

// MakeDirAccessible is a noop except when running on Windows.
// On Windows, it makes the directory on the provided path fully accessible
// to the user running the test by invoking 'icacls'.
// This is necessary to allow the test to delete the temporary directory after
// having made it inaccessible using MakeDirInaccessible.
func MakeDirAccessible(dirPath string) error {
	if runtime.GOOS == "windows" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		// Deprecated - kept in case some box doesn't have icacls.
		//cmd := exec.Command("cacls", dirPath, "/e", "/g", u.Username+":f")
		cmd := exec.Command("icacls", dirPath, "/grant", u.Username+":f")
		return run(cmd)
	}
	return nil
}

func run(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	return errors.Wrapf(err, string(out))
}
