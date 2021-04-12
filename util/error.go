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
	switch {
	case os.IsNotExist(cause):
		return errNotFound
	case os.IsPermission(cause):
		return errAccessDenied
	}
	return err
}
