package util

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

var (
	// ErrNotFound indicates that a file doesn't exist.
	ErrNotFound = fmt.Errorf("not found")
	// ErrAccessDenied indicates that a file isn't accessible.
	ErrAccessDenied = fmt.Errorf("access denied")
)

// CleanIOError rewrites "file does not exist" and "permission denied" errors with simpler ones.
func CleanIOError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	if errors.Is(err, os.ErrPermission) {
		return ErrAccessDenied
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return errors.Errorf("%v (%s)", pathErr.Err, pathErr.Op)
	}
	return err
}

// CleanJSONError rewrites JSON decoding errors into more concise, platform-independent ones.
func CleanJSONError(err error) error {
	var jsonErr *json.UnmarshalTypeError
	if errors.As(err, &jsonErr) {
		return errors.Errorf(
			"cannot decode field %q of type %q with value of type %q",
			jsonErr.Field,
			jsonErr.Type,
			jsonErr.Value,
		)
	}
	return errors.Wrap(err, "invalid JSON")
}
