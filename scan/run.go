package scan

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/util"
)

// ShouldSkipPath is a function for determining if a given path
// should be skipped when walking a file tree.
type ShouldSkipPath func(dir, name string) bool

// NoSkip always returns false.
func NoSkip(string, string) bool {
	return false
}

// SkipNameSet constructs a ShouldSkipPath which returns true
// when the base name matches any of the names in the provided set.
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
	if p != filepath.Clean(path) {
		log.Printf("following root symlink %q to %q\n", path, p)
	}
	return p, validateRoot(p)
}

func validateRoot(path string) error {
	i, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !i.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

// run runs the "scan" command without any sanity checks.
// In particular, the directory must not have a trailing slash as that will cause the file walk to panic.
func run(rootName, root string, shouldSkip ShouldSkipPath, cache *Dir) (*Dir, error) {
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
			modeName := util.FileInfoModeName(info)
			switch {
			case os.IsPermission(err):
				log.Printf("skipping inaccessible %v %q\n", modeName, path)
				return nil
			case os.IsNotExist(err):
				// TODO Test - can maybe only do on Windows (with too long path).
				log.Printf("error: %v %q not found!\n", modeName, path)
			}
			return errors.Wrapf(util.SimplifyIOError(err), "cannot walk %v %q", modeName, path)
		}
		parentPath := filepath.Dir(path)
		name := filepath.Base(path)
		if shouldSkip(parentPath, name) {
			log.Printf("skipping %v %q based on skip list\n", util.FileModeName(info.Mode()), path)
			if info.IsDir() {
				head.curDir.appendSkippedDir(name)
				return filepath.SkipDir
			}
			head.curDir.appendSkippedFile(name)
			return nil
		}

		// We don't get any signal that the walk has returned up the stack so have to detect it ourselves.
		for head.pathLen != len(parentPath) {
			head = head.prev
		}

		if mode := info.Mode(); mode.IsDir() {
			dir := NewDir(name)
			head.curDir.appendDir(dir) // Walk visits in lexical order
			head = &walkContext{
				prev:     head,
				curDir:   dir,
				pathLen:  len(path),
				cacheDir: safeFindDir(head.cacheDir, name),
			}
		} else if !mode.IsRegular() {
			// File is a symlink, named pipe, socket, device, etc.
			// We start by not supporting any of that.
			// IDEA If symlink, print target (and whether it exists).
			log.Printf("skipping %v %q during scan\n", util.FileModeName(mode), path)
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
				} else if hash == 0 {
					log.Printf("info: hash of file %q evaluated to 0 - this might result in warnings which can be safely ignored\n", path)
				}
			}
			head.curDir.appendFile(NewFile(name, size, hash)) // Walk visits in lexical order
		}
		return nil
	})
	for head.prev != nil {
		head = head.prev
	}
	return head.curDir, errors.Wrapf(err, "cannot scan root directory %q", root)
}

// hashFromCache looks up the content hash of the given file in the given cache dir.
// A cache miss is represented by the value 0.
// If the hash is cached with value 0, this is assumed to be a mistake (caused by 0 being the default value of uint64)
// and considered a cache miss:
// There's intentionally no way to cache the hash if it happens to be actually 0.
// This is deemed acceptable as this is expected to practically never happen.
// If the cached file size doesn't match the expected one, the cache is considered missed as well.
func hashFromCache(cacheDir *Dir, fileName string, fileSize int64) uint64 {
	f := safeFindFile(cacheDir, fileName)
	if f != nil && f.Size == fileSize {
		return f.Hash
	}
	return 0
}

// hashFile computes the FNV-1a hash of the contents of the file at the provided path.
func hashFile(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, errors.Wrap(util.SimplifyIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: cannot close file '%v': %v\n", path, err)
		}
	}()
	return hash(f)
}

// hash computes the FNV-1a hash of the contents of the provided reader.
func hash(r io.Reader) (uint64, error) {
	h := fnv.New64a() // is just a *uint64 internally
	n, err := io.Copy(h, r)
	return h.Sum64(), errors.Wrapf(err, "error reading file after approx. %d bytes", n)
}
