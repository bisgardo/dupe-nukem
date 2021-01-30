package scan

import (
	"errors"
	"fmt"
	"os"
)

func errFileOrDirectoryDoesNotExist(path string) error {
	return fmt.Errorf("file or directory %q does not exist", path)
}

func cleanFilepathWalkError(err error) error {
	if pathErr, ok := err.(*os.PathError); ok && errors.Is(pathErr.Err, os.ErrNotExist) {
		return errFileOrDirectoryDoesNotExist(pathErr.Path)
	}
	return err
}
