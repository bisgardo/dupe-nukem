package hash

import (
	"hash/fnv"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/util"
)

// File computes the FNV-1a hash of the contents of the file at the provided path.
func File(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, errors.Wrap(util.SimplifyIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: cannot close file %q: %v\n", path, err)
		}
	}()
	return Reader(f)
}

// Reader computes the FNV-1a hash of the contents of the provided reader.
func Reader(r io.Reader) (uint64, error) {
	h := fnv.New64a() // is just a *uint64 internally
	n, err := io.Copy(h, r)
	return h.Sum64(), errors.Wrapf(err, "read error after %d bytes", n)
}
