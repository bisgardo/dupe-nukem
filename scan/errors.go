package scan

import (
	"errors"
	"fmt"
	"os"
)

func cleanError(err error) error {
	if pathErr, ok := err.(*os.PathError); ok && errors.Is(pathErr.Err, os.ErrNotExist) {
		return fmt.Errorf("file or directory %q does not exist", pathErr.Path)
	}
	return err
}
