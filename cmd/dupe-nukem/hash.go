package main

import (
	"io/fs"
	"os"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/hash"
)

// Hash computes and returns the FNV-1a hash of the contents of the file on the provided path.
// If the path is empty, then the hash of the contents of stdin is computed instead.
func Hash(path string) (uint64, error) {
	if path == "" {
		return hash.Reader(os.Stdin)
	}
	res, err := hash.File(path)
	if cause := errors.Cause(err); cause != nil {
		if pathErr, ok := cause.(*fs.PathError); ok {
			err = pathErr.Err
		}
	}
	return res, errors.Wrapf(err, "cannot hash file %q", path)
}
