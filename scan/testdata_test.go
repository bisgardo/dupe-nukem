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

// node is a content item of a dir, used for building test data for Run.
type node interface {
	// simulateScanFromParent adds the "scan" result of the node to the Dir representing the parent node.
	simulateScanFromParent(parent *Dir, name string)

	// writeTestdata writes the directory structure rooted at the node to the provided path on disk.
	writeTestdata(t *testing.T, path string)
}

// dir is a directory node, implemented as a mapping to entry nodes by relative path.
// That is, the keys may contain '/' characters to implicitly define nested dir nodes.
// Different keys must not define any subdirectory more than once (i.e. paths must not overlap).
type dir map[string]node

func (d dir) simulateScanFromParent(parent *Dir, name string) {
	s := d.simulateScan(name)
	parent.appendDir(s)
}

func (d dir) simulateScan(name string) *Dir {
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
		n.simulateScanFromParent(res, nodePath)
	}
	return res
}

func (d dir) writeTestdata(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0700); err != nil { // permissions chosen to be unaffected by umask
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

// dirExt is an extension of dir that adds the ability
// to expect the directory to be skipped or made inaccessible.
type dirExt struct {
	dir
	skipped      bool
	inaccessible bool
}

func (d dirExt) simulateScanFromParent(parent *Dir, name string) {
	if d.inaccessible {
		return
	}
	if d.skipped {
		parent.appendSkippedDir(name)
		return
	}
	d.dir.simulateScanFromParent(parent, name)
}

func (d dirExt) writeTestdata(t *testing.T, path string) {
	d.dir.writeTestdata(t, path)
	if d.inaccessible {
		MakeInaccessibleT(t, path)
	}
}

// file is a file node.
type file struct {
	// The file's contents as a string.
	c string
	// The file's latest modification time (with second accuracy).
	ts time.Time
	// The file's hash as resolved from a cache file rather than being computed (if non-zero).
	// Should not be combined with inaccessible.
	hashFromCache uint64
	// Whether the file is expected to be skipped by Run.
	skipped bool
	// Whether the file is to be made inaccessible (and thus expecting Run to find it so).
	inaccessible bool
}

func (f file) simulateScanFromParent(parent *Dir, name string) {
	if f.skipped {
		parent.appendSkippedFile(name)
		return
	}
	if len(f.c) == 0 {
		parent.appendEmptyFile(name)
		return
	}
	// Inaccessibility is handled in simulateScan (by hashing to 0).
	// We don't have to check whether the file is already there,
	// as that cannot be expressed without duplicating dir (which is already checked).
	s := f.simulateScan(name)
	parent.appendFile(s)
}

func (f file) simulateScan(name string) *File {
	data := []byte(f.c)
	h := f.hashFromCache
	if h == 0 && !f.inaccessible {
		// If both hashFromCache and inaccessible are set, then the cached value is used.
		// This represents the situation that the file has become inaccessible since the run that produced the cache:
		// As the hash is cached, we make no attempts of opening the file, and thus don't notice that it's inaccessible.
		h = hash.Bytes(data)
	}
	var unixTime int64
	if !f.ts.IsZero() {
		unixTime = f.ts.Unix()
	}
	return NewFile(name, int64(len(data)), unixTime, h)
}

func (f file) writeTestdata(t *testing.T, path string) {
	data := []byte(f.c)
	_, err := os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("duplicate file %q", filepath.Base(path)))
	}
	require.ErrorIs(t, err, os.ErrNotExist)
	err = os.WriteFile(path, data, 0600) // permissions chosen to be unaffected by umask
	require.NoErrorf(t, err, "cannot create file %q with contents %q", path, f.c)
	if !f.ts.IsZero() {
		err := os.Chtimes(path, time.Time{}, f.ts)
		require.NoErrorf(t, err, "cannot update modification time of file %q", path)
	}
	if f.inaccessible {
		MakeInaccessibleT(t, path)
	}
}

type symlink string // target path relative to own location

func (s symlink) simulateScanFromParent(*Dir, string) {
	// Symlinks are ignored.
}

func (s symlink) writeTestdata(t *testing.T, path string) {
	err := os.Symlink(string(s), path)
	require.NoErrorf(t, err, "cannot create symlink with value %q at path %q", s, path)
}

type symlinkExt struct {
	symlink
	skipped bool
}

func (s symlinkExt) simulateScanFromParent(dir *Dir, name string) {
	if s.skipped {
		dir.appendSkippedFile(name)
	}
	s.symlink.simulateScanFromParent(dir, name)
}

// Verify conformance to node interface.
var (
	_ node = dir{}
	_ node = dirExt{}
	_ node = file{}
	_ node = symlink("")
	_ node = symlinkExt{}
)

func simulateScan(d dir, rootPath string) *Result {
	return &Result{
		TypeVersion: CurrentResultTypeVersion,
		Root:        d.simulateScan(rootPath),
	}
}

func Test__node(t *testing.T) {
	ts, err := time.Parse(time.Layout, time.Layout)
	require.NoError(t, err)
	ts = ts.Local() // assert.Equal only deems times equal if they're in the same time zone

	makeRoot := func() dir {
		return dir{
			"a":   file{},
			"b/d": file{c: "x\n", ts: ts},
			"c":   file{c: "y\n", hashFromCache: 53},
			"d":   dirExt{skipped: true},
			"e/f": dir{
				"a": file{c: "z\n", ts: ts, hashFromCache: 42},
				"g": file{inaccessible: true},
				"h": file{c: "h\n", ts: ts, inaccessible: true},
			},
			"h": file{c: "q", skipped: true},
			"x": dir{},
			"y": dirExt{inaccessible: true, dir: dir{"z": file{c: "zzz"}}},
		}
	}

	t.Run("writeTestdata", func(t *testing.T) {
		before := time.Now()
		root := makeRoot()
		rootPath := tempDir(t)
		root.writeTestdata(t, rootPath)
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

	t.Run("simulateScan", func(t *testing.T) {
		root := makeRoot()
		s := root.simulateScan("root")

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
	root := dir{
		"a/b": file{c: "ab"},
		"a/c": file{c: "ac"},
	}
	t.Run("writeTestdata", func(t *testing.T) {
		before := time.Now()
		rootPath := tempDir(t)
		root.writeTestdata(t, rootPath)
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

	t.Run("simulateScan", func(t *testing.T) {
		assert.PanicsWithError(t, "duplicate dir \"a\"", func() {
			root.simulateScan("root")
		})
	})
}

func Test__node_with_overlapping_files(t *testing.T) {
	root := dir{
		"a":   dir{"b": file{c: "x"}},
		"a/b": file{c: "y"},
	}
	t.Run("writeTestdata", func(t *testing.T) {
		rootPath := tempDir(t)
		assert.PanicsWithError(t, "duplicate file \"b\"", func() {
			root.writeTestdata(t, rootPath)
		})
	})

	t.Run("simulateScan", func(t *testing.T) {
		// Duplicate file implies duplicate dir.
		assert.PanicsWithError(t, "duplicate dir \"a\"", func() {
			root.simulateScan("root")
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

// TODO: Should name 'assertCompatible'?

// assertEqual asserts that this Dir equals the provided expectation.
// The assertion works like assert.Equal with the special rule that mod times are assumed equal if the expected one is zero.
// This exception exists because we don't want to explicitly set the mod times of all generated test files,
// in which case they default to the time that the test is run.
// The solution of patching the expectation with the current time didn't work well and was replaced with this one.
func (d *Dir) assertEqual(t *testing.T, want *Dir) {
	if d == nil {
		assert.Nil(t, want)
		return
	}
	assert.Equal(t, want.Name, d.Name)
	assert.Equal(t, want.EmptyFiles, d.EmptyFiles)
	assert.Equal(t, want.SkippedFiles, d.SkippedFiles)
	assert.Equal(t, want.SkippedDirs, d.SkippedDirs)

	dirCount := len(want.Dirs)
	fileCount := len(want.Files)
	assert.Len(t, d.Dirs, dirCount)
	assert.Len(t, d.Files, fileCount)
	// Avoid recursing if assertions already failed.
	for i := 0; i < dirCount && !t.Failed(); i++ {
		d.Dirs[i].assertEqual(t, want.Dirs[i])
	}
	for i := 0; i < fileCount && !t.Failed(); i++ {
		d.Files[i].assertEqual(t, want.Files[i])
	}
}

func (f *File) assertEqual(t *testing.T, want *File) {
	if f == nil {
		assert.Nil(t, want)
		return
	}
	assert.Equal(t, want.Name, f.Name)
	assert.Equal(t, want.Size, f.Size)
	if want.ModTime != 0 {
		assert.Equal(t, want.ModTime, f.ModTime)
	}
	assert.Equal(t, want.Hash, f.Hash)
}

func (r *Result) assertEqual(t *testing.T, want *Result) {
	if r == nil {
		assert.Nil(t, want)
		return
	}
	assert.Equal(t, want.TypeVersion, r.TypeVersion)
	r.Root.assertEqual(t, want.Root)
}
