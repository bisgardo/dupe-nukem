package testdata

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/scan"

	"github.com/bisgardo/dupe-nukem/hash"
	"github.com/bisgardo/dupe-nukem/testutil"
)

// Node is a content item of a DirNode, used for building test data for scan.Run.
type Node interface {
	// SimulateScanFromParent adds the "scan" result of the Node to the Dir representing the parent Node.
	SimulateScanFromParent(parent *scan.Dir, name string)

	// WriteTestdata writes the directory structure rooted at the Node to the provided path on disk.
	WriteTestdata(t *testing.T, path string)
}

// DirNode is a directory Node, implemented as a mapping to entry nodes by a relative path.
// Components in this path are separated by forward slash characters regardless of the platform we're running on.
// That is, the keys may include '/' to implicitly define nested DirNode (as long as it forms a valid path).
// Different keys must not define any subdirectory more than once (i.e. paths must not overlap).
type DirNode map[string]Node

// SimulateScanFromParent implements Node.SimulateScanFromParent.
func (d DirNode) SimulateScanFromParent(parent *scan.Dir, name string) {
	s := d.SimulateScan(name)
	parent.AppendDir(s)
}

// SimulateScan returns the result of simulating a scan of the directory.
func (d DirNode) SimulateScan(name string) *scan.Dir {
	res := scan.NewDir(name)
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
			// Handle name being a (Unix) path.
			slashIdx := strings.IndexRune(nodePath, '/')
			if slashIdx == -1 {
				// nodePath is no longer a path, just a name.
				break
			}
			dirName := nodePath[0:slashIdx]  // extract dir name
			nodePath = nodePath[slashIdx+1:] // pop dir name

			// Reject duplicated dir.
			// Note that we only have to check nested paths (e.g. "x/y") because the key sorting above ensures
			// that any non-nested case (e.g. "x") is always processed first.
			if scan.SafeFindDir(res, dirName) != nil {
				panic(fmt.Errorf("duplicate dir %q", dirName))
			}
			r := scan.NewDir(dirName)
			res.AppendDir(r)
			res = r
		}
		n.SimulateScanFromParent(res, nodePath)
	}
	return res
}

// WriteTestdata implements Node.WriteTestdata.
func (d DirNode) WriteTestdata(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0700); err != nil { // permissions chosen to be unaffected by umask
		require.NoErrorf(t, err, "cannot create dir on path %q", path)
	}
	// No need for iterating in sorted order.
	for name, n := range d {
		nodePath := filepath.Join(path, name)
		if p := filepath.Dir(nodePath); p != path {
			// nodePath has multiple components: create intermediary directories.
			DirNode(nil).WriteTestdata(t, p)
		}
		// Path may have arbitrary number of components.
		n.WriteTestdata(t, nodePath)
	}
}

// DirNodeExt is an extension of DirNode that adds the ability
// to expect the directory to be skipped or made inaccessible.
type DirNodeExt struct {
	Dir          DirNode
	Skipped      bool
	Inaccessible bool
}

// SimulateScanFromParent implements Node.SimulateScanFromParent.
func (d DirNodeExt) SimulateScanFromParent(parent *scan.Dir, name string) {
	if d.Inaccessible {
		return
	}
	if d.Skipped {
		parent.AppendSkippedDir(name)
		return
	}
	d.Dir.SimulateScanFromParent(parent, name)
}

// WriteTestdata implements Node.WriteTestdata.
func (d DirNodeExt) WriteTestdata(t *testing.T, path string) {
	d.Dir.WriteTestdata(t, path)
	if d.Inaccessible {
		testutil.MakeInaccessibleT(t, path)
	}
}

// FileNode is a Node that represents a file.
type FileNode struct {
	// The file's contents as a string.
	C string
	// The file's latest modification time (with second accuracy).
	Ts time.Time
	// The file's simulated hash.
	// If the value is non-zero (or Inaccessible is true), then SimulateScan will expect the hash to resolve to this value
	// (as if it was read from a cache file) instead of explicitly computing it.
	HashFromCache uint64
	// Whether SimulateScan should expect the file to be skipped by scan.Run.
	Skipped bool
	// Whether WriteTestdata is to make the file inaccessible (and thus expecting scan.Run to find it so).
	Inaccessible bool
}

// SimulateScanFromParent implements Node.SimulateScanFromParent.
func (f FileNode) SimulateScanFromParent(parent *scan.Dir, name string) {
	if f.Skipped {
		parent.AppendSkippedFile(name)
		return
	}
	if len(f.C) == 0 {
		parent.AppendEmptyFile(name)
		return
	}
	// Inaccessibility is handled in SimulateScan (by hashing to 0).
	// We don't have to check whether the file is already there,
	// as that cannot be expressed without duplicating dir (which is already checked).
	s := f.SimulateScan(name)
	parent.AppendFile(s)
}

// SimulateScan returns the result of simulating a scan of the file.
func (f FileNode) SimulateScan(name string) *scan.File {
	data := []byte(f.C)
	h := f.HashFromCache
	if h == 0 && !f.Inaccessible {
		// If both HashFromCache and Inaccessible are set, then the cached value is used.
		// This represents the situation that the file has become inaccessible since the run that produced the cache:
		// As the hash is cached, we make no attempts of opening the file, and thus won't notice it being inaccessible.
		h = hash.Bytes(data)
	}
	var unixTime int64
	if !f.Ts.IsZero() {
		unixTime = f.Ts.Unix()
	}
	return scan.NewFile(name, int64(len(data)), unixTime, h)
}

// WriteTestdata implements Node.WriteTestdata.
func (f FileNode) WriteTestdata(t *testing.T, path string) {
	data := []byte(f.C)
	_, err := os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("duplicate file %q", filepath.Base(path)))
	}
	require.ErrorIs(t, err, os.ErrNotExist)
	err = os.WriteFile(path, data, 0600) // permissions chosen to be unaffected by umask
	require.NoErrorf(t, err, "cannot create file %q with contents %q", path, f.C)
	if !f.Ts.IsZero() {
		err := os.Chtimes(path, time.Time{}, f.Ts)
		require.NoErrorf(t, err, "cannot update modification time of file %q", path)
	}
	if f.Inaccessible {
		testutil.MakeInaccessibleT(t, path)
	}
}

// SymlinkNode is a Node that represents a symlink.
// The underlying value is the target path relative to the symlink's own location.
type SymlinkNode string

// SimulateScanFromParent implements Node.SimulateScanFromParent.
func (s SymlinkNode) SimulateScanFromParent(*scan.Dir, string) {
	// Symlinks are ignored.
}

// WriteTestdata implements Node.WriteTestdata.
func (s SymlinkNode) WriteTestdata(t *testing.T, path string) {
	err := os.Symlink(string(s), path)
	require.NoErrorf(t, err, "cannot create symlink with value %q at path %q", s, path)
}

// SymlinkExtNode is an extension of SymlinkNode that adds the ability
// to expect the directory to be skipped.
type SymlinkExtNode struct {
	Symlink SymlinkNode
	Skipped bool
}

// SimulateScanFromParent implements Node.SimulateScanFromParent.
func (s SymlinkExtNode) SimulateScanFromParent(dir *scan.Dir, name string) {
	if s.Skipped {
		dir.AppendSkippedFile(name)
	}
	s.Symlink.SimulateScanFromParent(dir, name)
}

// WriteTestdata implements Node.WriteTestdata.
func (s SymlinkExtNode) WriteTestdata(t *testing.T, path string) {
	s.Symlink.WriteTestdata(t, path)
}

// Verify conformance to Node interface.
var (
	_ Node = DirNode{}
	_ Node = DirNodeExt{}
	_ Node = FileNode{}
	_ Node = SymlinkNode("")
	_ Node = SymlinkExtNode{}
)
