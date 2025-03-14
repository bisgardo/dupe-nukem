package scan

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/hash"
	"github.com/bisgardo/dupe-nukem/util"
)

// ShouldSkipPath is a function for determining whether a given path
// should be skipped when walking a file tree.
type ShouldSkipPath func(dir, name string) bool

// NoSkip always returns false.
func NoSkip(string, string) bool {
	return false
}

var _ ShouldSkipPath = NoSkip // declare that NoSkip conforms to ShouldSkipPath

// SkipNameSet constructs a ShouldSkipPath which returns true
// if the base name matches any of the names in the provided set.
func SkipNameSet(names map[string]struct{}) ShouldSkipPath {
	return func(dir, name string) bool {
		_, ok := names[name]
		return ok
	}
}

// Run runs the "scan" command with all arguments provided.
// If the root is a symlink, then this link is traversed recursively.
// The root name of the scan result keeps the name of the original symlink.
// The following sanity checks are performed:
// - The root directory must not be skipped.
// - If a cache is provided, its root must have the same name as the provided root.
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

	rootDir := NewDir(rootName)
	head := &walkContext{
		prev:     nil,
		curDir:   rootDir,
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
				// TODO: Should return 'filepath.SkipDir' to skip children?
				return nil
			case os.IsNotExist(err):
				// TODO: Can maybe test on Windows (with too long path)?
				log.Printf("error: %v %q not found\n", modeName, path) // cannot test
			}
			// TODO: Should be able to test
			//       - creating a file with an invalid timestamp (use 'os.Chtimes(filename, time.Unix(0, 0), time.Unix(0, 0))')
			//       - setting the file descriptor limit to a value lower than the number of files in a directory
			//         (use 'syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)')
			//       These approaches should also be useful for testing other things currently deemed "cannot test".
			return errors.Wrapf(util.SimplifyIOError(err), "cannot walk %v %q", modeName, path) // cannot test
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

		// Detect that the walk has returned up the stack, as we aren't given any information about that.
		// Checking just the length of the path works because directories are guaranteed to be visited
		// before the files that they contain.
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
			// We don't currently support any of that.
			// IDEA: If symlink, print target (and whether it exists).
			log.Printf("skipping %v %q during scan\n", util.FileModeName(mode), path)
		} else if size := info.Size(); size == 0 {
			head.curDir.appendEmptyFile(name) // Walk visits in lexical order
		} else {
			// IDEA: Parallelize hash computation (via work queue for example).
			// IDEA: Consider adding option to hash a limited number of bytes only
			//       (the reason being that if two files differ, the first 1MB or so probably differ too).
			h, hit := hashFromCache(head.cacheDir, name, size)
			if h == 0 {
				if hit {
					log.Printf("warning: cached hash value 0 of file %q ignored\n", path)
				}
				h, err = hash.File(path)
				if err != nil {
					// Currently report error but keep going.
					log.Printf("error: cannot hash file %q: %v\n", path, err)
				} else if h == 0 {
					// TODO: What warnings? And why?
					log.Printf("info: hash of file %q evaluated to 0 - this might result in warnings which can be safely ignored\n", path)
				}
			}
			head.curDir.appendFile(NewFile(name, size, h)) // Walk visits in lexical order
		}
		return nil
	})
	return rootDir, errors.Wrapf(err, "cannot scan root directory %q", root) // cannot test
}

// hashFromCache looks up the hash of the contents of the provided file in the provided cache dir.
// The boolean return value indicates whether the hash was found in the cache or not.
// A cache miss will always return hash with value 0.
// A cached hash value of 0 is assumed to be a mistake (caused by 0 being the default value of uint64),
// and will cause a warning to be logged from the caller (which knows the full path):
// There's intentionally no way to cache the hash if it happens to be actually 0.
// This is deemed acceptable as this is expected to practically never happen.
// If the cached file size doesn't match the expected one, the cache is considered missed as well.
// TODO: Also check timestamp.
func hashFromCache(cacheDir *Dir, fileName string, fileSize int64) (uint64, bool) {
	f := safeFindFile(cacheDir, fileName)
	if f != nil && f.Size == fileSize {
		return f.Hash, true
	}
	return 0, false
}
