package main

import (
	"os"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/hash"
	"github.com/bisgardo/dupe-nukem/util"
)

// Hash computes and returns the FNV-1a hash of the contents of the file on the provided path.
// If the path is empty, then the hash of the contents of stdin is computed instead.
func Hash(path string) (uint64, error) {
	if path == "" {
		return hash.Reader(os.Stdin)
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrapf(util.SimplifyIOError(err), "cannot stat %q", path)
	}
	if info.IsDir() {
		return 0, errors.Errorf("cannot hash directory %q", path)
	}
	res, err := hash.File(path)
	return res, errors.Wrapf(err, "cannot hash %v %q", util.FileModeName(info.Mode()), path)
}
