package hash

import (
	"hash"
	"hash/fnv"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/util"
)

// New constructs a new FNV-1a hash function.
func New() hash.Hash64 {
	return fnv.New64a() // is just a *uint64 internally
}

// File computes the hash (of function constructed by New) of the contents of the file at the provided path.
func File(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, errors.Wrap(util.SimplifyIOError(err), "cannot open file") // caller wraps path
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: cannot close file %q: %v\n", path, err) // cannot test
		}
	}()
	return Reader(f)
}

// Reader computes the (of function constructed by New) hash of the contents of the provided reader.
func Reader(r io.Reader) (uint64, error) {
	h := New()
	n, err := io.Copy(h, r)
	return h.Sum64(), errors.Wrapf(err, "read error after %d bytes", n) // cannot test
}

// Bytes computes the hash (of function constructed by New) of the contents of the provided reader.
func Bytes(b []byte) uint64 {
	h := New()
	_, err := h.Write(b)
	if err != nil {
		// Docs of Hash states that Write never returns an error.
		panic(err)
	}
	return h.Sum64()
}
