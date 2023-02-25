package util

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

var (
	errNotFound     = fmt.Errorf("not found")
	errAccessDenied = fmt.Errorf("access denied")
)

// SimplifyIOError replaces "file does not exist" and "permission denied" errors with simpler, constant ones.
func SimplifyIOError(err error) error {
	cause := errors.Cause(err)
	if os.IsNotExist(cause) {
		return errNotFound
	}
	if os.IsPermission(cause) {
		return errAccessDenied
	}
	if pathErr, ok := cause.(*os.PathError); ok {
		return errors.Errorf("%v (%s)", pathErr.Err, pathErr.Op)
	}
	return err
}
