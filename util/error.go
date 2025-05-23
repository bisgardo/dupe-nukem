package util

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

var (
	errNotFound     = fmt.Errorf("not found")
	errAccessDenied = fmt.Errorf("access denied")
)

// SimplifyIOError replaces "file does not exist" and "permission denied" errors with simpler, constant ones.
// TODO: Use errors.Is/As and rename to just IOError.
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

func JSONError(err error) error {
	var jsonErr *json.UnmarshalTypeError
	if errors.As(err, &jsonErr) {
		return errors.Errorf(
			"cannot decode value of type %q into field %q of type %q",
			jsonErr.Value,
			jsonErr.Field,
			jsonErr.Type,
		)
	}
	return errors.Wrap(err, "invalid JSON")
}
