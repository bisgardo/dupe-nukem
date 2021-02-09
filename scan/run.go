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

func SkipNameSet(names map[string]struct{}) ShouldSkipPath {
	return func(dir, name string) bool {
		_, ok := names[name]
		return ok
	}
}

// Run runs the "scan" command with all arguments provided.
// If the root is a symlink, this link is traversed recursively.
// The root name of the scan result keeps the name of the original symlink.
// The following sanity checks are performed:
// - The root directory must not be skipped.
// - If a cache is provided, it's root must have the same name as the provided root.
// - The root is an existing directory.
func Run(root string, shouldSkip ShouldSkipPath, cache *Dir) (*Dir, error) {
	rootName := filepath.Base(root)
	if shouldSkip(filepath.Dir(root), rootName) {
		return nil, fmt.Errorf("skipping root directory %q", root)
	}
	if cache != nil && cache.Name != rootName {
		// While there's no technical reason for this requirement,
		// it seems reasonable that differing root names would signal a mistake in most cases.
		return nil, fmt.Errorf("cache of directory %q cannot be used with root directory %q", cache.Name, rootName)
	}
	r, err := resolveRoot(root)
	if err != nil {
		return nil, errors.Wrapf(util.SimplifyIOError(err), "invalid root directory %q", root)
	}
	return run(rootName, r, shouldSkip, cache)
}

func resolveRoot(path string) (string, error) {
	// Follow symlink.
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if p != path {
		log.Printf("following root symlink %q to %q", path, p)
	}
	return p, validateRoot(p)
}

func validateRoot(path string) error {
	s, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !s.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

// run runs the "scan" command without any sanity checks.
// In particular, the directory must not have a trailing slash as that will cause the file walk to panic.
func run(rootName, root string, shouldSkip ShouldSkipPath, cache *Dir) (*Dir, error) {
	type walkContext struct {
		prev                *walkContext
		curDir              *Dir
		pathLen             int
		cacheDir            *Dir
		lastCacheDirHitIdx  int // has no meaning if cacheDir is nil
		lastCacheFileHitIdx int // has no meaning if cacheDir is nil
	}

	head := &walkContext{
		prev:               nil,
		curDir:             NewDir(rootName),
		pathLen:            len(root),
		cacheDir:           cache,
		lastCacheDirHitIdx: -1,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Propagate error and skip root.
		if err != nil || path == root {
			if os.IsPermission(err) {
				log.Printf("skipping inaccessible %v %q\n", util.FileModeName(info.Mode()), path)
				return nil
			}
			return err
		}
		parentPath := filepath.Dir(path)
		name := filepath.Base(path)
		if shouldSkip(parentPath, name) {
			if info.IsDir() {
				log.Printf("skipping directory %q based on skip list\n", path)
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

		if mode := info.Mode(); mode.IsDir() {
			dir := NewDir(name)
			head.curDir.appendDir(dir) // Walk visits in lexical order
			d, idx := safeFindDir(head.cacheDir, name)
			if d != nil {
				head.lastCacheDirHitIdx = idx
			}

			head = &walkContext{
				prev:               head,
				curDir:             dir,
				pathLen:            len(path),
				cacheDir:           d,
				lastCacheDirHitIdx: -1,
			}
		} else if !mode.IsRegular() {
			// File is a symlink, named pipe, socket, device, etc.
			// We start by not supporting any of that.
			// IDEA If symlink, print target (and if it exists).
			log.Printf("skipping %v %q during scan\n", util.FileModeName(mode), path)
		} else if size := info.Size(); size == 0 {
			head.curDir.appendEmptyFile(name) // Walk visits in lexical order
		} else {
			// IDEA Parallelize hash computation (via work queue for example).
			// IDEA Consider adding option to hash a limited number of bytes only
			//      (the reason being that if two files differ, the first 1MB or so probably differ too).
			var hash uint64
			f, idx := safeFindFile(head.cacheDir, name)
			if f != nil {
				head.lastCacheFileHitIdx = idx
				if f.Size == size {
					hash = f.Hash
				}
			}
			if hash == 0 {
				// File was not found in cache or its size didn't match.
				hash, err = hashFile(path)
				if err != nil {
					// Currently report error but keep going.
					log.Printf("error: cannot hash file %q: %v\n", path, err)
				}
			}
			// If the hash is cached with value 0, this is assumed to be a mistake and considered a cache miss:
			// There's no way to represent the cached value if it happens to be actually 0.
			head.curDir.appendFile(NewFile(name, size, hash)) // Walk visits in lexical order
		}
		return nil
	})
	for head.prev != nil {
		head = head.prev
	}
	return head.curDir, errors.Wrapf(simplifyFilepathWalkError(err), "cannot scan root directory %q", root)
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
