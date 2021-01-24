package scan

import (
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

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
		} else {
			size := info.Size()
			head.curDir.appendFile(NewFile(name, size)) // Walk visits in lexical order
		}
		return nil
	})
	for head.prev != nil {
		head = head.prev
	}
	return head.curDir, errors.Wrapf(cleanError(err), "cannot scan root directory %q", root)
}
