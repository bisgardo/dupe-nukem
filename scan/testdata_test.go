package scan

import (
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

// node is a common interface for files and directories to be written to disk as testdata.
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
		n.simulateScanFromParent(res, nodePath)
	}
	return res
}

func (d dir) writeTestdata(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0700); err != nil {
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
	err := os.WriteFile(path, data, 0600)
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
	// assert.Equal doesn't compare times correctly unless they're in the same time zone.
	ts = ts.Local()

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
		}
	}

	t.Run("writeTestdata", func(t *testing.T) {
		root := makeRoot()
		before := time.Now()
		rootPath := tempDir(t)
		root.writeTestdata(t, rootPath)
		after := time.Now()

		// Assert that root wasn't modified.
		require.Equal(t, makeRoot(), root)

		infos, err := readFileInfos(rootPath, []string{"e/f/g", "e/f/h"})
		require.NoError(t, err)

		want := map[string]fileInfo{
			"a":     {Name: "a", Mode: 0600},
			"b":     {Name: "b", Mode: os.ModeDir | 0700},
			"c":     {Name: "c", Contents: "y\n", Mode: 0600},
			"b/d":   {Name: "d", Contents: "x\n", Mode: 0600, ModTime: ts},
			"d":     {Name: "d", Mode: os.ModeDir | 0700},
			"e":     {Name: "e", Mode: os.ModeDir | 0700},
			"e/f":   {Name: "f", Mode: os.ModeDir | 0700},
			"e/f/a": {Name: "a", Contents: "z\n", Mode: 0600, ModTime: ts},
			"e/f/g": {Name: "g", Contents: "", Mode: 0},              // cannot read contents of inaccessible file
			"e/f/h": {Name: "h", Contents: "", Mode: 0, ModTime: ts}, // cannot read contents of inaccessible file
			"h":     {Name: "h", Contents: "q", Mode: 0600},
			"x":     {Name: "x", Mode: os.ModeDir | 0700},
		}

		require.Len(t, infos, len(want))

		for path, info := range infos {
			wantInfo, ok := want[filepath.ToSlash(path)] // Windows...
			require.True(t, ok)
			if wantInfo.ModTime.IsZero() {
				mt := info.ModTime
				// It appears that we cannot trust the computer to follow the laws of physics below second accuracy.
				assert.True(t, before.Add(-time.Second).Before(mt))
				assert.True(t, after.Add(time.Second).After(mt))
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
						{
							Name:    "d",
							Size:    2,
							ModTime: ts.Unix(),
							Hash:    644258871406045975, // actual hash
						},
					},
				},
				{
					Name: "e",
					Dirs: []*Dir{
						{
							Name: "f",
							Files: []*File{
								{
									Name:    "a",
									Size:    2,
									ModTime: ts.Unix(),
									Hash:    42, // "cached"
								},
								{
									Name:    "h",
									Size:    2,
									ModTime: ts.Unix(),
									Hash:    0, // cannot hash inaccessible file
								},
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
				{
					Name: "c",
					Size: 2,
					Hash: 53, // "cached"
				},
			},
			EmptyFiles:   []string{"a"},
			SkippedFiles: []string{"h"},
			SkippedDirs:  []string{"d"},
		}
		s.assertEqual(t, want)
	})
}

// TODO: Add test where inaccessible dir has contents.
// TODO: Add test where different keys have overlapping paths.

type fileInfo struct {
	Name     string
	Contents string
	Mode     os.FileMode
	ModTime  time.Time
}

func readFileInfos(rootPath string, inaccessiblePaths []string) (map[string]fileInfo, error) {
	inaccessiblePathsSet := make(map[string]struct{})
	for _, p := range inaccessiblePaths {
		// Running path through clean because Windows.
		inaccessiblePathsSet[filepath.Clean(p)] = struct{}{}
	}

	res := make(map[string]fileInfo)
	err := filepath.Walk(rootPath, func(absPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if absPath != rootPath {
			relPath, err := filepath.Rel(rootPath, absPath)
			if err != nil {
				return errors.WithStack(err)
			}

			var contents string
			if _, skip := inaccessiblePathsSet[relPath]; !skip && !info.IsDir() {
				bs, err := os.ReadFile(absPath)
				if err != nil {
					return errors.WithStack(err)
				}
				if int64(len(bs)) != info.Size() {
					panic("unexpected file size")
				}
				contents = string(bs)
			}

			res[relPath] = fileInfo{
				Name:     info.Name(),
				Contents: contents,
				Mode:     info.Mode(),
				ModTime:  info.ModTime(),
			}
		}
		return nil
	})
	return res, err
}

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
