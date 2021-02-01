package util

import (
	"fmt"
	"os"
)

var (
	errFileNotFound = fmt.Errorf("file not found")
	errAccessDenied = fmt.Errorf("access denied")
)

// SimplifyIOError replaces "file does not exist" and "permission denied" errors with constant ones.
func SimplifyIOError(err error) error {
	switch {
	case os.IsNotExist(err):
		return errFileNotFound
	case os.IsPermission(err):
		return errAccessDenied
	}
	return err
}

// ErrFileOrDirectoryDoesNotExist constructs a "file or directory does not exist" error which includes the path name.
func ErrFileOrDirectoryDoesNotExist(path string) error {
	return fmt.Errorf("file or directory %q does not exist", path)
}

// ErrFileOrDirectoryAccessDenied constructs a "file or directory access denied" error which includes the path name.
func ErrFileOrDirectoryAccessDenied(path string) error {
	return fmt.Errorf("access denied to %v %q", fileModeNameFromPath(path), path)
}

func fileModeNameFromPath(path string) string {
	s, err := os.Stat(path)
	if err != nil {
		return "file or directory (stat failed)"
	}
	return FileModeName(s.Mode())
}
