package main

import (
	"os"

	"github.com/bisgardo/dupe-nukem/scan"
)

// Hash computes and returns the FNV-1a hash of the contents of the file on the provided path.
// If the path is empty, the hash of the contents of stdin is computed instead.
func Hash(path string) (uint64, error) {
	if path == "" {
		return scan.Hash(os.Stdin)
	}
	return scan.HashFile(path)
}
