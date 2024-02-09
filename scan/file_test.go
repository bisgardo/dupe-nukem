package scan

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__safeFindDir_nil_returns_nil(t *testing.T) {
	assert.Nil(t, safeFindDir(nil, "x"))
}

func Test__safeFindFile_nil_returns_nil(t *testing.T) {
	assert.Nil(t, safeFindFile(nil, "x"))
}

//goland:noinspection GoSnakeCaseUsage
var (
	testDir_x = &Dir{
		Name: "x",
		Dirs: []*Dir{testDir_y, testDir_r},
		Files: []*File{
			{Name: "a", Size: 1},
			{Name: "b", Size: 2},
			{Name: "c", Size: 3},
		},
	}
	testDir_y = &Dir{
		Name: "y",
		Dirs: []*Dir{testDir_z},
		Files: []*File{
			{Name: "r", Size: 4},
			{Name: "s", Size: 5},
		},
	}
	testDir_z = &Dir{
		Name: "z",
		Files: []*File{
			{Name: "a", Size: 6},
			{Name: "b", Size: 7},
		},
	}
	testDir_r = &Dir{
		Name: "r",
		Dirs: []*Dir{
			{Name: "s"},
			{
				Name: "t",
				Files: []*File{
					{Name: "c", Size: 8},
				},
			},
		},
	}
)

func Test__safeFindDir_finds_subdir(t *testing.T) {
	tests := []struct {
		dir  *Dir
		name string
		want *Dir
	}{
		{dir: testDir_x, name: "", want: nil},                // doesn't find garbage
		{dir: testDir_x, name: "nonexistent", want: nil},     // doesn't find other garbage
		{dir: testDir_x, name: "x", want: nil},               // doesn't find dir
		{dir: testDir_x, name: "y", want: testDir_y},         // finds subdir 1/2
		{dir: testDir_x, name: "r", want: testDir_r},         // finds subdir 2/2
		{dir: testDir_x, name: "z", want: nil},               // doesn't find nested subdir
		{dir: testDir_x, name: "a", want: nil},               // doesn't find file
		{dir: testDir_x, name: "s", want: nil},               // doesn't find nested file (in "r")
		{dir: testDir_y, name: "x", want: nil},               // doesn't find parent
		{dir: testDir_y, name: "z", want: testDir_z},         // finds single subdir
		{dir: testDir_z, name: "", want: nil},                // has no subdirs
		{dir: testDir_z, name: "", want: nil},                // has no subdirs
		{dir: testDir_r, name: "s", want: testDir_r.Dirs[0]}, // finds subdir 1/2 (empty)
		{dir: testDir_r, name: "t", want: testDir_r.Dirs[1]}, // finds subdir 2/2 (no subdirs)
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%v/%v", test.dir.Name, test.name), func(t *testing.T) {
			dir := safeFindDir(test.dir, test.name)
			assert.True(t, dir == test.want)
		})
	}
}

func Test__safeFindFile_finds_file(t *testing.T) {
	tests := []struct {
		dir  *Dir
		name string
		want *File
	}{
		{dir: testDir_x, name: "", want: nil},                 // doesn't find garbage
		{dir: testDir_x, name: "nonexistent", want: nil},      // doesn't find other garbage
		{dir: testDir_x, name: "x", want: nil},                // doesn't find dir
		{dir: testDir_x, name: "y", want: nil},                // doesn't find subdir
		{dir: testDir_x, name: "a", want: testDir_x.Files[0]}, // finds file 1/3
		{dir: testDir_x, name: "b", want: testDir_x.Files[1]}, // finds file 2/3
		{dir: testDir_x, name: "c", want: testDir_x.Files[2]}, // finds file 3/3
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%v/%v", test.dir.Name, test.name), func(t *testing.T) {
			dir := safeFindFile(test.dir, test.name)
			assert.True(t, dir == test.want)
		})
	}
}
