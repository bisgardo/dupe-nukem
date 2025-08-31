package scantest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bisgardo/dupe-nukem/scan"
)

// AssertEqualDir asserts that the provided scan.Dir matches the provided expectation.
// The assertion works like assert.Equal except for a special rule explained in AssertEqualFile.
func AssertEqualDir(t *testing.T, d *scan.Dir, want *scan.Dir) {
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
		AssertEqualDir(t, d.Dirs[i], want.Dirs[i])
	}
	for i := 0; i < fileCount && !t.Failed(); i++ {
		AssertEqualFile(t, d.Files[i], want.Files[i])
	}
}

// AssertEqualFile asserts that the provided scan.File matches the provided expectation.
// The assertion works like assert.Equal with the special rule that mod times are assumed equal if the expected one is zero.
// This exception exists because we don't want to explicitly set the mod times of all generated test files,
// in which case they default to the time that the test is run.
// The solution of patching the expectation with the current time didn't work well and was replaced with this one.
// Now if only you could somehow specify how assert.Empty should test equality for a given type...
func AssertEqualFile(t *testing.T, f *scan.File, want *scan.File) {
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

// AssertEqualResult asserts that the provided scan.Result matches the provided expectation.
// The assertion works like assert.Equal except for a special rule explained in AssertEqualFile.
func AssertEqualResult(t *testing.T, r *scan.Result, want *scan.Result) {
	if r == nil {
		assert.Nil(t, want)
		return
	}
	assert.Equal(t, want.TypeVersion, r.TypeVersion)
	AssertEqualDir(t, r.Root, want.Root)
}
