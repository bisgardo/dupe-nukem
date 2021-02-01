package scan

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/bisgardo/dupe-nukem/util"
	"github.com/pkg/errors"
)

// ShouldSkipPath is a function for determining if a given path should be skipped when walking a file tree.
type ShouldSkipPath func(dir, name string) bool

// NoSkip always returns false.
func NoSkip(string, string) bool {
	return false
}

// Run runs the "scan" command with all arguments provided.
// The directory is assumed to be "clean" in the sense that filepath.Clean is a no-op.
func Run(root string, shouldSkip ShouldSkipPath, cache *Dir) (*Dir, error) {
	rootName := filepath.Base(root)
	if cache != nil && cache.Name != rootName {
		// While there's no technical reason for this requirement,
		// it seems reasonable that differing root names would signal a mistake in most cases.
		return nil, fmt.Errorf("cache of dir %q cannot be used with root dir %q", cache.Name, rootName)
	}

	type walkContext struct {
		prev     *walkContext
		curDir   *Dir
		pathLen  int
		cacheDir *Dir
	}

	head := &walkContext{
		prev:     nil,
		curDir:   NewDir(rootName),
		pathLen:  len(root),
		cacheDir: cache,
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
				log.Printf("skipping dir %q based on skip list\n", path)
				head.curDir.appendSkippedDir(name)
				return filepath.SkipDir
			} else {
				log.Printf("skipping file %q based on skip list\n", path)
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
				prev:     head,
				curDir:   dir,
				pathLen:  len(path),
				cacheDir: safeFindDir(head.cacheDir, name),
			}
		} else if size := info.Size(); size == 0 {
			head.curDir.appendEmptyFile(name) // Walk visits in lexical order
		} else {
			// IDEA Parallelize hash computation (via work queue for example).
			// IDEA Consider adding option to hash a limited number of bytes only
			//      (the reason being that if two files differ, the first 1MB or so probably differ too).
			hash := hashFromCache(head.cacheDir, name, size)
			if hash == 0 {
				hash, err = hashFile(path)
				if err != nil {
					// Currently report error but keep going.
					log.Printf("error: cannot hash file %q: %v\n", path, err)
				}
			}
			head.curDir.appendFile(NewFile(name, size, hash)) // Walk visits in lexical order
		}
		return nil
	})
	for head.prev != nil {
		head = head.prev
	}
	// TODO Can the error happen in other cases than the root dir not existing?
	//      If so, simplify the wrapped one such that the path is printed only once.
	// TODO At least test case where subdir on the walk path is inaccessible.
	return head.curDir, errors.Wrapf(simplifyFilepathWalkError(err), "cannot scan root directory %q", root)
}

// hashFromCache looks up the content hash of the given file in the given cache dir.
// A cache miss is represented by the value 0.
// If the hash is cached with value 0, this is assumed to be a mistake and considered a cache miss:
// There's no way to represent the cached value if it happens to be actually 0.
func hashFromCache(cacheDir *Dir, fileName string, fileSize int64) uint64 {
	f := safeFindFile(cacheDir, fileName)
	if f != nil && f.Size == fileSize {
		return f.Hash
	}
	return 0
}

// hashFile computes the FNV-1a hash of the file at the provided path.
func hashFile(path string) (uint64, error) {
	h := fnv.New64a() // is just a *uint64 internally
	f, err := os.Open(path)
	if err != nil {
		return 0, errors.Wrap(util.SimplifyIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: cannot close file '%v': %v\n", path, err)
		}
	}()
	n, err := io.Copy(h, f)
	return h.Sum64(), errors.Wrapf(err, "error reading file after approx. %d bytes", n)
}
