package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
