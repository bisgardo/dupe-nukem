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
	"github.com/bisgardo/dupe-nukem/testutil"
)

// node is an interface for files and directories to be written to disk as testdata.
type node interface {
	// simulateScanFromParent adds the "scan" result of the node to the Dir representing the parent node.
	simulateScanFromParent(parent *Dir, name string)

	// writeTestdata writes the directory structure rooted at the node to the provided path on disk.
	writeTestdata(t *testing.T, path string)
}

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
			// that we always process any non-nested (e.g. "y") case first.
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

type dirExt struct {
	dir              dir
	skipped          bool
	makeInaccessible bool
}

func (d dirExt) simulateScanFromParent(parent *Dir, name string) {
	if d.makeInaccessible {
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
	if d.makeInaccessible {
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
	// Note that we cannot express a situation where the file is duplicated
	// as that would imply duplicated dir which is already rejected.
	s := f.simulateScan(name)
	parent.appendFile(s)
}

func (f file) simulateScan(name string) *File {
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
	var unixTime int64
	if !f.ts.IsZero() {
		unixTime = f.ts.Unix()
	}
	return NewFile(name, int64(len(data)), unixTime, h)
}

func (f file) writeTestdata(t *testing.T, path string) {
	data := []byte(f.c)
	err := os.WriteFile(path, data, 0600) // permissions chosen to be unaffected by umask
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

func (s symlink) simulateScanFromParent(*Dir, string) {
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

func Test_node(t *testing.T) {
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
				"g": file{makeInaccessible: true},
				"h": file{c: "h\n", ts: ts, makeInaccessible: true},
			},
			"h": file{c: "q", skipped: true},
			"x": dir{},
			"y": dirExt{makeInaccessible: true, dir: dir{"z": file{c: "zzz"}}},
		}
	}

	t.Run("writeTestdata", func(t *testing.T) {
		before := time.Now().Add(-1 * time.Second) // extend backwards by 1s to accommodate for the fact that the FS appears to be lazy around updating time
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

func Test_node_with_overlapping_dirs(t *testing.T) {
	root := dir{
		"a/b": file{c: "ab"},
		"a/c": file{c: "ac"},
	}
	t.Run("writeTestdata", func(t *testing.T) {
		before := time.Now().Add(-1 * time.Second) // extend backwards by 1s to accommodate for the fact that the FS appears to be lazy around updating time
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

func Test_node_with_overlapping_files(t *testing.T) {
	root := dir{
		"a":   dir{"b": file{c: "x"}},
		"a/b": file{c: "y"},
	}
	t.Run("writeTestdata", func(t *testing.T) {
		before := time.Now()
		rootPath := tempDir(t)
		root.writeTestdata(t, rootPath)
		after := time.Now()

		p := filepath.FromSlash // because Windows...
		infos, err := readInfoTree(rootPath, nil)
		require.NoError(t, err)

		// The file is just overwritten...
		want := map[string]fileInfo{
			p("a"):   {Name: "a", Mode: os.ModeDir | 0700},
			p("a/b"): {Name: "b", Contents: "y", Mode: 0600},
		}

		// Extend expected time backwards by 1s to accommodate for the fact that the FS appears to be lazy around updating time.
		assertCompatibleInfos(t, want, infos, before.Add(-time.Second), after)
	})

	t.Run("simulateScan", func(t *testing.T) {
		assert.PanicsWithError(t, "duplicate dir \"a\"", func() {
			root.simulateScan("root")
		})
	})
}

func assertCompatibleInfos(t *testing.T, want, infos map[string]fileInfo, before, after time.Time) {
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
		var expectedMode os.FileMode
		if info.IsDir() {
			expectedMode = os.ModeDir
		}
		if info.Mode() != expectedMode {
			return "", errors.Errorf("expected path %q to be inaccessible", path)
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

// TODO: Should name 'assertCompatible'??

// assertEqual asserts that this Dir equals the provided expectation.
// The assertion works like assert.Equal with the special rule that mod times are assumed equal if the expected one is zero.
// This exception exists because we don't want to explicitly set the mod times of all generated test files,
// in which case they default to the time that the test is run.
// The solution of patching the expectation with the current time didn't work well and was replaced with this one.
func (d *Dir) assertEqual(t *testing.T, expected *Dir) {
	assert.Equal(t, d.Name, expected.Name)
	assert.Equal(t, expected.EmptyFiles, d.EmptyFiles)
	assert.Equal(t, expected.SkippedFiles, d.SkippedFiles)
	assert.Equal(t, expected.SkippedDirs, d.SkippedDirs)

	dirCount := len(expected.Dirs)
	fileCount := len(expected.Files)
	assert.Len(t, d.Dirs, dirCount)
	assert.Len(t, d.Files, fileCount)
	// Avoid recursing if assertions already failed.
	for i := 0; i < dirCount && !t.Failed(); i++ {
		d.Dirs[i].assertEqual(t, expected.Dirs[i])
	}
	for i := 0; i < fileCount && !t.Failed(); i++ {
		d.Files[i].assertEqual(t, expected.Files[i])
	}
}

func (f *File) assertEqual(t *testing.T, expected *File) {
	assert.Equal(t, expected.Name, f.Name)
	assert.Equal(t, expected.Size, f.Size)
	if expected.ModTime != 0 {
		assert.Equal(t, expected.ModTime, f.ModTime)
	}
	assert.Equal(t, expected.Hash, f.Hash)
}
