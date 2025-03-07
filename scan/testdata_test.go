package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/hash"
	"github.com/bisgardo/dupe-nukem/testutil"
)

type node interface {
	appendTo(parent *Dir, name string)
	writeTo(t *testing.T, path string)
}

type dir map[string]node

func (d dir) appendTo(parent *Dir, name string) {
	s := d.toScanDir(name)
	parent.appendDir(s)
}

func (d dir) toScanDir(name string) *Dir {
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
		for {
			// Handle name being a path.
			slashIdx := strings.IndexRune(nodePath, '/')
			if slashIdx == -1 {
				// nodePath is no longer a path, just a name.
				break
			}
			dirName := nodePath[0:slashIdx]  // extract dir name
			nodePath = nodePath[slashIdx+1:] // pop dir name
			// TODO: Use existing dir if it's already there?
			//       Probably better to reject...
			r := NewDir(dirName)
			res.appendDir(r)
			res = r
		}
		n.appendTo(res, nodePath)
	}
	return res
}

func (d dir) writeTo(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		require.NoErrorf(t, err, "cannot create dir on path %q", path)
	}
	// No need for iterating in sorted order.
	for name, n := range d {
		nodePath := filepath.Join(path, name)
		if p := filepath.Dir(nodePath); p != path {
			// nodePath has multiple components: create intermediary directories.
			dir(nil).writeTo(t, p)
		}
		// Path may have arbitrary number of components.
		n.writeTo(t, nodePath)
	}
}

//func (d dir) ext() dirExt {
//	return dirExt{dir: d}
//}

type dirExt struct {
	dir          dir
	skipped      bool
	inaccessible bool
}

func (d dirExt) appendTo(parent *Dir, name string) {
	if d.inaccessible {
		return
	}
	if d.skipped {
		parent.appendSkippedDir(name)
		return
	}
	d.dir.appendTo(parent, name)
}

func (d dirExt) writeTo(t *testing.T, path string) {
	d.dir.writeTo(t, path)
	if d.inaccessible {
		testutil.MakeInaccessibleT(t, path)
	}
}

type file struct {
	// The file's contents as a string.
	c string
	// The file's latest modification time (not yet implemented).
	ts int64
	// The file's hash as resolved from a cache file rather than being computed (if non-zero).
	// Should not be combined with inaccessible.
	cachedHash uint64
	// Whether the file is expected to be skipped by Run.
	skipped bool
	// Whether the file is to be made inaccessible (and thus expecting Run to find it so).
	inaccessible bool
}

func (f file) appendTo(parent *Dir, name string) {
	if f.skipped {
		parent.appendSkippedFile(name)
		return
	}
	if len(f.c) == 0 {
		parent.appendEmptyFile(name)
		return
	}
	// Inaccessibility is handled in toScanFile by hashing to 0.
	parent.appendFile(f.toScanFile(name))
}

func (f file) toScanFile(name string) *File {
	if f.ts != 0 {
		// TODO: Implement...
		panic("custom timestamp is not yet implemented")
	}
	data := []byte(f.c)
	h := f.cachedHash
	if h == 0 && !f.inaccessible {
		// If both cachedHash and inaccessible are set, then the cached value is used.
		// As cachedHash isn't used to actually derive a cache
		// (it just simulates that the file's hash originated from one),
		// the two inputs cannot be combined in any meaningful way.
		// And consequently there's no need for doing anything about it.
		// TODO: Add a test where a file that is cached is now inaccessible nonetheless.
		h = hash.Bytes(data)
	}
	return NewFile(name, int64(len(data)), h)
}

func (f file) writeTo(t *testing.T, path string) {
	if f.ts != 0 {
		// TODO: Implement...
		panic("custom timestamp is not yet implemented")
	}
	data := []byte(f.c)
	err := os.WriteFile(path, data, 0644)
	require.NoErrorf(t, err, "cannot create file %q with contents %q", path, f.c)
	if f.inaccessible {
		testutil.MakeInaccessibleT(t, path)
	}
}

type symlink string // target path relative to own location

func (s symlink) appendTo(*Dir, string) {
	// Symlinks are ignored.
}

func (s symlink) writeTo(t *testing.T, path string) {
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

// TODO: Test the testers.

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
