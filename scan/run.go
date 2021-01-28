package scan

import (
	"hash/fnv"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const computeHash = true // to become a parameter

// ShouldSkipPath is a function for determining if a given path should be skipped when walking a file tree.
type ShouldSkipPath func(dir, name string) bool

// NoSkip always returns false.
func NoSkip(string, string) bool {
	return false
}

// Run runs the "scan" command to walk the provided root directory.
// The directory is assumed to be "clean" in the sense that filepath.Clean is a no-op.
func Run(root string, shouldSkip ShouldSkipPath) (*Dir, error) {
	type walkContext struct {
		prev    *walkContext
		curDir  *Dir
		pathLen int
	}

	head := &walkContext{
		prev:    nil,
		curDir:  NewDir(filepath.Base(root)),
		pathLen: len(root),
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Propagate error and skip root.
		if err != nil || path == root {
			return err
		}
		parentPath := filepath.Dir(path)
		name := filepath.Base(path)
		if shouldSkip(parentPath, name) {
			if info.IsDir() {
				log.Printf("skipping dir %q\n", path)
				head.curDir.appendSkippedDir(name)
				return filepath.SkipDir
			} else {
				log.Printf("skipping file %q\n", path)
				head.curDir.appendSkippedFile(name)
				return nil
			}
		}

		// We don't get any signal that the walk has returned up the stack so have to detect it ourselves.
		for head.pathLen != len(parentPath) {
			head = head.prev
		}

		if info.IsDir() {
			dir := NewDir(name)
			head.curDir.appendDir(dir) // Walk visits in lexical order
			head = &walkContext{
				prev:    head,
				curDir:  dir,
				pathLen: len(path),
			}
		} else if size := info.Size(); size == 0 {
			head.curDir.appendEmptyFile(name) // Walk visits in lexical order
		} else {
			// IDEA Parallelize hash computation (via work queue for example).
			// IDEA Consider adding option to hash a limited number of bytes only
			//      (the reason being that if two files differ, the first 1MB or so probably differ too).
			var hash uint64
			if computeHash {
				var err error
				hash, err = hashFile(path)
				if err != nil {
					// Currently report error but keep going.
					log.Printf("error: cannot hash file %q: %v", path, err)
				}
			}
			head.curDir.appendFile(NewFile(name, size, hash)) // Walk visits in lexical order
		}
		return nil
	})
	for head.prev != nil {
		head = head.prev
	}
	return head.curDir, errors.Wrapf(cleanError(err), "cannot scan root directory %q", root)
}

func hashFile(path string) (uint64, error) {
	// Hasher is just a *uint64 so there's no point in thinking of reusing it.
	h := fnv.New64a()
	f, err := os.Open(path)
	if err != nil {
		return 0, errors.Wrap(err, "cannot open file")
	}
	n, err := io.Copy(h, f)
	return h.Sum64(), errors.Wrapf(err, "error reading file after around %d bytes", n)
}
