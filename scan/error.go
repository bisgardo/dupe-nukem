package scan

import (
	"errors"
	"os"

	"github.com/bisgardo/dupe-nukem/util"
)

func simplifyFilepathWalkError(err error) error {
	pathErr, ok := err.(*os.PathError)
	if !ok {
		return err
	}
	switch {
	case errors.Is(pathErr.Err, os.ErrNotExist):
		return util.ErrFileOrDirectoryDoesNotExist(pathErr.Path)
	}
	return err
}
