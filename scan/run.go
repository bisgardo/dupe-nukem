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

// Result is the result of calling Run.
type Result struct {
	// TypeVersion is the "version" of the [Result] type.
	// It's used to compare decoded values of external representations against [CurrentResultTypeVersion]
	// (which is the value that it's always encoded with).
	// In the context of JSON, the type implicitly defines the schema of the result,
	// so in that context the name is "schema_version".
	TypeVersion int `json:"schema_version"`
	// Root is the scanned directory data as a recursive data structure.
	Root *Dir `json:"root"`
}

// CurrentResultTypeVersion is the currently expected value of [Result.TypeVersion].
// The initial (and current) version is 1 to ensure that the default decode value of 0 can be assumed to mean that the field is missing.
// The value is not going to be bumped before the application reaches a stable, useful state,
// even if there are breaking changes to the format before then.
const CurrentResultTypeVersion = 1

// NoSkip doesn't skip any files.
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
// - If a cache is provided, its root must have the same name as the provided root (after following any symlinks).
// - The root is an existing directory.
func Run(root string, shouldSkip ShouldSkipPath, cache *Dir) (*Result, error) {
	rootPath, err := resolveRoot(root)
	if err != nil {
		return nil, errors.Wrapf(util.IOError(err), "invalid root directory %q", root)
	}
	if cache != nil && cache.Name != rootPath {
		// While there's no technical reason for this requirement,
		// it seems reasonable that differing root names would signal a mistake in most cases.
		// For now, we keep it simple and just require the paths to match (one can always edit the file manually).
		// In the future you could imagine this being relaxed in ways like:
		// - Allow remapping the name.
		// - Allow caches that only cover some subdirectory.
		//   Could even allow multiple such files (using the one of the closest parent).
		// - Bypass the check entirely.
		return nil, fmt.Errorf("cache of directory %q cannot be used with root directory %q", cache.Name, rootPath)
	}
	res, err := run(rootPath, shouldSkip, cache)
	return &Result{
		TypeVersion: CurrentResultTypeVersion,
		Root:        res,
	}, errors.Wrapf(err, "cannot scan root directory %q", rootPath) // cannot test
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
	if err := validateRoot(p); err != nil {
		return "", err
	}
	return filepath.Abs(p)
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
// In particular, the root path must not have a trailing slash as that would cause the file walk to panic.
func run(rootPath string, shouldSkip ShouldSkipPath, cache *Dir) (*Dir, error) {
	if shouldSkip(filepath.Dir(rootPath), filepath.Base(rootPath)) {
		log.Printf("not skipping root directory %q", rootPath)
	}

	type walkContext struct {
		prev     *walkContext
		curDir   *Dir
		pathLen  int
		cacheDir *Dir
	}

	res := NewDir(rootPath)
	head := &walkContext{
		prev:     nil,
		curDir:   res,
		pathLen:  len(rootPath),
		cacheDir: cache,
	}
	return res, filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		// Propagate error and skip root.
		if err != nil || path == rootPath {
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
			return errors.Wrapf(util.IOError(err), "cannot walk %v %q", modeName, path) // cannot test
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
			h, hit := hashFromCache(head.cacheDir, name, size, info.ModTime().Unix())
			// If the cache contains the actual hash value 0,
			// we assume that it's either caused by the file being inaccessible
			// or by a mistake resulting in unintended zero-initialization somewhere.
			// A warning to let the user know that the cache contains this value.
			// The fact that a file with hash 0 cannot be cached is deemed acceptable,
			// as this is expected to practically never happen for real data.
			// But even if it did, the only drawback is that the file's hash will get redundantly recomputed.
			if h == 0 {
				if hit {
					log.Printf("warning: cached hash value 0 of file %q ignored\n", path)
				}
				h, err = hash.File(path)
				if err != nil {
					// Currently report error but keep going (i.e. include the file with empty hash).
					log.Printf("error: cannot hash file %q: %v\n", path, err)
				} else if h == 0 {
					log.Printf("info: hash of file %q evaluated to 0 - this might result in warnings (which can be safely ignored) if the output is used as cache in future scans\n", path)
				}
			}
			head.curDir.appendFile(NewFile(name, size, info.ModTime().Unix(), h)) // Walk visits in lexical order
		}
		return nil
	})
}

// hashFromCache looks up the hash of the contents of the provided file in the provided cache dir.
// If the cached file size or modification time don't match that of the file being looked up, the cache is considered missed.
// A cache miss will always return hash value 0.
// The boolean return value indicates whether the hash was found in the cache or not.
func hashFromCache(cacheDir *Dir, fileName string, fileSize int64, modTimeUnix int64) (uint64, bool) {
	f := safeFindFile(cacheDir, fileName)
	if f != nil && f.Size == fileSize && f.ModTime == modTimeUnix {
		return f.Hash, true
	}
	return 0, false
}
