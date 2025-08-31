package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/hash"
	. "github.com/bisgardo/dupe-nukem/testutil"
)

// Node is a content item of a DirNode, used for building test data for Run.
type Node interface {
	// SimulateScanFromParent adds the "scan" result of the Node to the Dir representing the parent Node.
	SimulateScanFromParent(parent *Dir, name string)

	// WriteTestdata writes the directory structure rooted at the Node to the provided path on disk.
	WriteTestdata(t *testing.T, path string)
}

// DirNode is a directory Node, implemented as a mapping to entry nodes by a relative path.
// Components in this path are separated by forward slash characters regardless of the platform we're running on.
// That is, the keys may include '/' to implicitly define nested DirNode (as long as it forms a valid path).
// Different keys must not define any subdirectory more than once (i.e. paths must not overlap).
type DirNode map[string]Node

func (d DirNode) SimulateScanFromParent(parent *Dir, name string) {
	s := d.SimulateScan(name)
	parent.appendDir(s)
}

func (d DirNode) SimulateScan(name string) *Dir {
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
			if safeFindDir(res, dirName) != nil {
				panic(fmt.Errorf("duplicate dir %q", dirName))
			}
			r := NewDir(dirName)
			res.appendDir(r)
			res = r
		}
		n.SimulateScanFromParent(res, nodePath)
	}
	return res
}

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

func (d DirNodeExt) SimulateScanFromParent(parent *Dir, name string) {
	if d.Inaccessible {
		return
	}
	if d.Skipped {
		parent.appendSkippedDir(name)
		return
	}
	d.Dir.SimulateScanFromParent(parent, name)
}

func (d DirNodeExt) WriteTestdata(t *testing.T, path string) {
	d.Dir.WriteTestdata(t, path)
	if d.Inaccessible {
		MakeInaccessibleT(t, path)
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
	// Whether SimulateScan should expect the file to be skipped by Run.
	Skipped bool
	// Whether WriteTestdata is to make the file inaccessible (and thus expecting Run to find it so).
	Inaccessible bool
}

func (f FileNode) SimulateScanFromParent(parent *Dir, name string) {
	if f.Skipped {
		parent.appendSkippedFile(name)
		return
	}
	if len(f.C) == 0 {
		parent.appendEmptyFile(name)
		return
	}
	// Inaccessibility is handled in SimulateScan (by hashing to 0).
	// We don't have to check whether the file is already there,
	// as that cannot be expressed without duplicating dir (which is already checked).
	s := f.SimulateScan(name)
	parent.appendFile(s)
}

func (f FileNode) SimulateScan(name string) *File {
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
	return NewFile(name, int64(len(data)), unixTime, h)
}

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
		MakeInaccessibleT(t, path)
	}
}

// SymlinkNode is a Node that represents a symlink.
// The underlying value is the target path relative to the symlink's own location.
type SymlinkNode string

func (s SymlinkNode) SimulateScanFromParent(*Dir, string) {
	// Symlinks are ignored.
}

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

func (s SymlinkExtNode) SimulateScanFromParent(dir *Dir, name string) {
	if s.Skipped {
		dir.appendSkippedFile(name)
	}
	s.Symlink.SimulateScanFromParent(dir, name)
}

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

func Test__node(t *testing.T) {
	ts, err := time.Parse(time.Layout, time.Layout)
	require.NoError(t, err)
	ts = ts.Local() // assert.Equal only deems times equal if they're in the same time zone

	makeRoot := func() DirNode {
		return DirNode{
			"a":   FileNode{},
			"b/d": FileNode{C: "x\n", Ts: ts},
			"c":   FileNode{C: "y\n", HashFromCache: 53},
			"d":   DirNodeExt{Skipped: true},
			"e/f": DirNode{
				"a": FileNode{C: "z\n", Ts: ts, HashFromCache: 42},
				"g": FileNode{Inaccessible: true},
				"h": FileNode{C: "h\n", Ts: ts, Inaccessible: true},
			},
			"h": FileNode{C: "q", Skipped: true},
			"x": DirNode{},
			"y": DirNodeExt{Inaccessible: true, Dir: DirNode{"z": FileNode{C: "zzz"}}},
		}
	}

	t.Run("WriteTestdata", func(t *testing.T) {
		before := time.Now()
		root := makeRoot()
		rootPath := tempDir(t)
		root.WriteTestdata(t, rootPath)
		after := time.Now()

		// Check that root wasn't modified.
		require.Equal(t, makeRoot(), root)

		p := filepath.FromSlash // because Windows...
		infos, err := readInfoTree(rootPath, []string{p("e/f/g"), p("e/f/h"), p("y")})
		require.NoError(t, err)

		want := map[string]fileInfo{
			p("a"):     {Name: "a", Mode: 0600},
			p("b"):     {Name: "b", Mode: os.ModeDir | 0700},
			p("c"):     {Name: "c", Contents: "y\n", Mode: 0600},
			p("b/d"):   {Name: "d", Contents: "x\n", Mode: 0600, ModTime: ts},
			p("d"):     {Name: "d", Mode: os.ModeDir | 0700},
			p("e"):     {Name: "e", Mode: os.ModeDir | 0700},
			p("e/f"):   {Name: "f", Mode: os.ModeDir | 0700},
			p("e/f/a"): {Name: "a", Contents: "z\n", Mode: 0600, ModTime: ts},
			p("e/f/g"): {Name: "g", Contents: "", Mode: 0},              // cannot read contents of inaccessible file
			p("e/f/h"): {Name: "h", Contents: "", Mode: 0, ModTime: ts}, // cannot read contents of inaccessible file
			p("h"):     {Name: "h", Contents: "q", Mode: 0600},
			p("x"):     {Name: "x", Mode: os.ModeDir | 0700},
			p("y"):     {Name: "y", Mode: os.ModeDir}, // not seeing contained file "z" (it is there, but we'd have to be root to see it)
		}

		assertCompatibleInfos(t, want, infos, before, after)
	})

	t.Run("SimulateScan", func(t *testing.T) {
		root := makeRoot()
		s := root.SimulateScan("root")

		// Assert that root wasn't modified.
		require.Equal(t, makeRoot(), root)

		want := &Dir{
			Name: "root",
			Dirs: []*Dir{
				{
					Name: "b",
					Files: []*File{
						{Name: "d", Size: 2, ModTime: ts.Unix(), Hash: 644258871406045975}, // actual hash
					},
				},
				{
					Name: "e",
					Dirs: []*Dir{
						{
							Name: "f",
							Files: []*File{
								{Name: "a", Size: 2, ModTime: ts.Unix(), Hash: 42}, // cached
								{Name: "h", Size: 2, ModTime: ts.Unix(), Hash: 0},  // cannot hash inaccessible file
							},
							EmptyFiles: []string{"g"},
						},
					},
				},
				{
					Name: "x",
				},
			},
			Files: []*File{
				{Name: "c", Size: 2, Hash: 53}, // cached + no mod time
			},
			EmptyFiles:   []string{"a"},
			SkippedFiles: []string{"h"},
			SkippedDirs:  []string{"d"},
		}
		s.assertEqual(t, want)
	})
}

func Test__node_with_overlapping_dirs(t *testing.T) {
	root := DirNode{
		"a/b": FileNode{C: "ab"},
		"a/c": FileNode{C: "ac"},
	}
	t.Run("WriteTestdata", func(t *testing.T) {
		before := time.Now()
		rootPath := tempDir(t)
		root.WriteTestdata(t, rootPath)
		after := time.Now()

		p := filepath.FromSlash // because Windows...
		infos, err := readInfoTree(rootPath, nil)
		require.NoError(t, err)

		want := map[string]fileInfo{
			p("a"):   {Name: "a", Mode: os.ModeDir | 0700},
			p("a/b"): {Name: "b", Contents: "ab", Mode: 0600},
			p("a/c"): {Name: "c", Contents: "ac", Mode: 0600},
		}

		assertCompatibleInfos(t, want, infos, before, after)
	})

	t.Run("SimulateScan", func(t *testing.T) {
		assert.PanicsWithError(t, "duplicate dir \"a\"", func() {
			root.SimulateScan("root")
		})
	})
}

func Test__node_with_overlapping_files(t *testing.T) {
	root := DirNode{
		"a":   DirNode{"b": FileNode{C: "x"}},
		"a/b": FileNode{C: "y"},
	}
	t.Run("WriteTestdata", func(t *testing.T) {
		rootPath := tempDir(t)
		assert.PanicsWithError(t, "duplicate file \"b\"", func() {
			root.WriteTestdata(t, rootPath)
		})
	})

	t.Run("SimulateScan", func(t *testing.T) {
		// Duplicate file implies duplicate dir.
		assert.PanicsWithError(t, "duplicate dir \"a\"", func() {
			root.SimulateScan("root")
		})
	})
}

func assertCompatibleInfos(t *testing.T, want, infos map[string]fileInfo, before, after time.Time) {
	// Extend time by 1s in both directions as some file systems appear use inexact timing.
	before = before.Add(-1 * time.Second)
	after = after.Add(1 * time.Second)

	require.Len(t, infos, len(want))
	for path, info := range infos {
		wantInfo, ok := want[path]
		require.True(t, ok)
		if wantInfo.ModTime.IsZero() {
			mt := info.ModTime
			assert.True(t, before.Before(mt))
			assert.True(t, after.After(mt))
			wantInfo.ModTime = mt
		}
		// Windows...
		if runtime.GOOS == "windows" {
			if wantInfo.Mode.IsDir() {
				wantInfo.Mode |= 0777
			} else {
				wantInfo.Mode |= 0666
			}
		}
		assert.Equal(t, wantInfo, info)
	}
}

type fileInfo struct {
	Name     string
	Contents string
	Mode     os.FileMode
	ModTime  time.Time
}

func readInfoTree(rootPath string, inaccessiblePaths []string) (map[string]fileInfo, error) {
	inaccessiblePathsSet := make(map[string]struct{})
	for _, p := range inaccessiblePaths {
		inaccessiblePathsSet[p] = struct{}{}
	}
	res := make(map[string]fileInfo)
	err := filepath.Walk(rootPath, func(absPath string, info os.FileInfo, err error) error {
		if err != nil && !errors.Is(err, fs.ErrPermission) || absPath == rootPath {
			return err
		}
		relPath, err := filepath.Rel(rootPath, absPath)
		if err != nil {
			return err
		}
		_, expectInaccessible := inaccessiblePathsSet[relPath]
		contents, err := readPath(absPath, info, expectInaccessible)
		if err != nil {
			return err
		}
		res[relPath] = fileInfo{
			Name:     info.Name(),
			Contents: contents,
			Mode:     info.Mode(),
			ModTime:  info.ModTime(),
		}
		return nil
	})
	return res, err
}

func readPath(path string, info os.FileInfo, expectInaccessible bool) (string, error) {
	if expectInaccessible {
		_, err := os.Open(path)
		if !errors.Is(err, os.ErrPermission) {
			return "", errors.Wrapf(err, "expected path %q to exist but be inaccessible", path)
		}
	} else if !info.IsDir() {
		bs, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if int64(len(bs)) != info.Size() {
			panic("unexpected file size") // sanity check, should never happen
		}
		return string(bs), nil
	}
	return "", nil
}
