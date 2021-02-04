package util

import (
	"fmt"
	"os"
)

var (
	errNotFound     = fmt.Errorf("not found")
	errAccessDenied = fmt.Errorf("access denied")
)

// SimplifyIOError replaces "file does not exist" and "permission denied" errors with simpler, constant ones.
func SimplifyIOError(err error) error {
	switch {
	case os.IsNotExist(err):
		return errNotFound
	case os.IsPermission(err):
		return errAccessDenied
	}
	return err
}

// ErrFileOrDirectoryNotFound constructs a "file or directory not found" error which includes the path name.
func ErrFileOrDirectoryNotFound(path string) error {
	return fmt.Errorf("file or directory %q not found", path)
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
