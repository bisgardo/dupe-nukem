package util

import (
	"fmt"
	"os"
)

var (
	errFileNotFound = fmt.Errorf("file not found")
)

// SimplifyIOError replaces "file does not exist" and "permission denied" errors with constant ones.
func SimplifyIOError(err error) error {
	switch {
	case os.IsNotExist(err):
		return errFileNotFound
	}
	return err
}

// ErrFileOrDirectoryDoesNotExist constructs a "file or directory does not exist" error which includes the path name.
func ErrFileOrDirectoryDoesNotExist(path string) error {
	return fmt.Errorf("file or directory %q does not exist", path)
}
