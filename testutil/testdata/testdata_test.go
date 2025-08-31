package testdata

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/bisgardo/dupe-nukem/scan/scantest"
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
		rootPath := t.TempDir()
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

		want := &scan.Dir{
			Name: "root",
			Dirs: []*scan.Dir{
				{
					Name: "b",
					Files: []*scan.File{
						{Name: "d", Size: 2, ModTime: ts.Unix(), Hash: 644258871406045975}, // actual hash
					},
				},
				{
					Name: "e",
					Dirs: []*scan.Dir{
						{
							Name: "f",
							Files: []*scan.File{
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
			Files: []*scan.File{
				{Name: "c", Size: 2, Hash: 53}, // cached + no mod time
			},
			EmptyFiles:   []string{"a"},
			SkippedFiles: []string{"h"},
			SkippedDirs:  []string{"d"},
		}
		scantest.AssertEqualDir(t, s, want)
	})
}

func Test__node_with_overlapping_dirs(t *testing.T) {
	root := DirNode{
		"a/b": FileNode{C: "ab"},
		"a/c": FileNode{C: "ac"},
	}
	t.Run("WriteTestdata", func(t *testing.T) {
		before := time.Now()
		rootPath := t.TempDir()
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
		rootPath := t.TempDir()
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
	// Extend time by 1s in both directions as some file systems appear to use inexact timing.
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
