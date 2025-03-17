package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/hash"
	"github.com/bisgardo/dupe-nukem/testutil"
)

// node is a common interface for files and directories to be written to disk as testdata.
type node interface {
	// simulateScanFromParent adds the "scan" result of the node to the Dir representing the parent node.
	// Because the implementation file might stat the actual file on disk (when no modification time has been specified),
	// writeTestdata should be called before this method (unless the equivalent test data already exists on the provided path).
	simulateScanFromParent(parent *Dir, parentPath, name string)

	// writeTestdata writes the directory structure rooted at the node to the provided path on disk.
	writeTestdata(t *testing.T, path string)
}

type dir map[string]node

func (d dir) simulateScanFromParent(parent *Dir, parentPath, name string) {
	s := d.simulateScanAux(parentPath, name)
	parent.appendDir(s)
}

func (d dir) simulateScanAux(parentPath, name string) *Dir {
	res := NewDir(name)
	// Iterate nodes in sorted order to respect ordering requirements of the append functions of Dir.
	nodePaths := make([]string, 0, len(d))
	for n := range d {
		nodePaths = append(nodePaths, n)
	}
	sort.Strings(nodePaths)
	for _, nodePath := range nodePaths {
		res := res // prevent overwriting for future iterations
		n := d[nodePath]
		path := filepath.Join(parentPath, name)
		for {
			// Handle name being a path.
			slashIdx := strings.IndexRune(nodePath, '/')
			if slashIdx == -1 {
				// nodePath is no longer a path, just a name.
				break
			}
			dirName := nodePath[0:slashIdx]  // extract dir name
			nodePath = nodePath[slashIdx+1:] // pop dir name
			path = filepath.Join(path, dirName)
			// TODO: Use existing dir if it's already there?
			//       Probably better to reject...
			r := NewDir(dirName)
			res.appendDir(r)
			res = r
		}
		n.simulateScanFromParent(res, path, nodePath)
	}
	return res
}

func (d dir) simulateScan(path string) *Dir {
	return d.simulateScanAux(filepath.Split(path))
}

func (d dir) writeTestdata(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		require.NoErrorf(t, err, "cannot create dir on path %q", path)
	}
	// No need for iterating in sorted order.
	for name, n := range d {
		nodePath := filepath.Join(path, name)
		if p := filepath.Dir(nodePath); p != path {
			// nodePath has multiple components: create intermediary directories.
			dir(nil).writeTestdata(t, p)
		}
		// Path may have arbitrary number of components.
		n.writeTestdata(t, nodePath)
	}
}

type dirExt struct {
	dir          dir
	skipped      bool
	inaccessible bool
}

func (d dirExt) simulateScanFromParent(parent *Dir, parentPath, name string) {
	if d.inaccessible {
		return
	}
	if d.skipped {
		parent.appendSkippedDir(name)
		return
	}
	d.dir.simulateScanFromParent(parent, parentPath, name)
}

func (d dirExt) writeTestdata(t *testing.T, path string) {
	d.dir.writeTestdata(t, path)
	if d.inaccessible {
		testutil.MakeInaccessibleT(t, path)
	}
}

type file struct {
	// The file's contents as a string.
	c string
	// The file's latest modification time (with second accuracy).
	ts time.Time
	// The file's hash as resolved from a cache file rather than being computed (if non-zero).
	// Should not be combined with makeInaccessible.
	hashFromCache uint64
	// Whether the file is expected to be skipped by Run.
	skipped bool
	// Whether the file is to be made makeInaccessible (and thus expecting Run to find it so).
	makeInaccessible bool
}

func (f file) simulateScanFromParent(parent *Dir, parentPath, name string) {
	if f.skipped {
		parent.appendSkippedFile(name)
		return
	}
	if len(f.c) == 0 {
		parent.appendEmptyFile(name)
		return
	}
	// Inaccessibility is handled in simulateScan (by hashing to 0).
	s := f.simulateScan(parentPath, name)
	parent.appendFile(s)
}

func (f file) simulateScan(parentPath, name string) *File {
	data := []byte(f.c)
	h := f.hashFromCache
	if h == 0 && !f.makeInaccessible {
		// If both hashFromCache and makeInaccessible are set, then the cached value is used.
		// As hashFromCache isn't used to actually derive a cache
		// (it just simulates that the file's hash originated from one),
		// the two inputs cannot be combined in any meaningful way.
		// And consequently there's no need for doing anything about it.
		// TODO: Add a test where a file that is cached is now makeInaccessible nonetheless.
		h = hash.Bytes(data)
	}
	modTime := f.ts
	if modTime.IsZero() {
		// Timestamp is zero: read it from file (and panic if it isn't there - could just log??).
		path := filepath.Join(parentPath, name)
		info, err := os.Lstat(path)
		if err != nil {
			panic(fmt.Errorf("cannot stat file %q: %w", path, err))
		}
		modTime = info.ModTime()
	}
	return NewFile(name, int64(len(data)), modTime.Unix(), h)
}

func (f file) writeTestdata(t *testing.T, path string) {
	data := []byte(f.c)
	err := os.WriteFile(path, data, 0644)
	require.NoErrorf(t, err, "cannot create file %q with contents %q", path, f.c)
	if !f.ts.IsZero() {
		err := os.Chtimes(path, time.Time{}, f.ts)
		require.NoErrorf(t, err, "cannot update modification time of file %q", path)
	}
	if f.makeInaccessible {
		testutil.MakeInaccessibleT(t, path)
	}
}

type symlink string // target path relative to own location

func (s symlink) simulateScanFromParent(*Dir, string, string) {
	// Symlinks are ignored.
}

func (s symlink) writeTestdata(t *testing.T, path string) {
	err := os.Symlink(string(s), path)
	require.NoErrorf(t, err, "cannot create symlink with value %q at path %q", s, path)
}

// Verify conformance to node interface.
var (
	_ node = dir{}
	_ node = dirExt{}
	_ node = file{}
	_ node = symlink("")
)

// TODO: Test the testers (implement timestamping first).

//// withSkippedNames constructs a copy of the Dir with the provided names being added to the "skipped" lists
//// instead of the regular ones.
//func (d *Dir) withSkippedNames(names map[string]struct{}) *Dir {
//	res := NewDir(d.Name)
//	res.EmptyFiles = append(res.EmptyFiles, d.EmptyFiles...)
//	res.SkippedDirs = append(res.SkippedDirs, d.SkippedDirs...)
//	res.SkippedFiles = append(res.SkippedFiles, d.SkippedFiles...)
//	// Iteration order assumes that the input fields already respect the sorting requirement.
//	// Except for the "skipped" lists, this automatically ensures that the order is maintained for the result also.
//	for _, n := range d.Dirs {
//		if _, ok := names[n.Name]; ok {
//			res.appendSkippedDir(n.Name)
//		} else {
//			res.appendDir(n)
//		}
//	}
//	for _, n := range d.Files {
//		if _, ok := names[n.Name]; ok {
//			res.appendSkippedFile(n.Name)
//		} else {
//			res.appendFile(n)
//		}
//	}
//	sort.Strings(res.SkippedDirs)
//	sort.Strings(res.SkippedFiles)
//	return res
//}
//
//func (d *Dir) findReplace(name string, replace func(*Dir) *Dir) *Dir {
//	rest := ""
//	idx := strings.IndexRune(name, '/')
//	if idx != -1 {
//		name, rest = name[0:idx], name[idx+1:]
//	}
//	for i, n := range d.Dirs {
//		if n.Name != name {
//			continue
//		}
//		if rest != "" {
//			return n.findReplace(rest, replace)
//		}
//		if replace == nil {
//			return n
//		}
//		n = replace(n)
//		d.Dirs[i] = n
//		return n
//	}
//	return nil
//}
