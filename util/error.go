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

// IOError replaces "file does not exist" and "permission denied" errors with simpler, constant ones.
func IOError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return errNotFound
	}
	if errors.Is(err, os.ErrPermission) {
		return errAccessDenied
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
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
