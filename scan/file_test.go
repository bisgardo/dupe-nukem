package scan

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__safeFindDir_nil_returns_nil(t *testing.T) {
	res, _ := safeFindDir(nil, "x")
	assert.Nil(t, res)
}

func Test__safeFindFile_nil_returns_nil(t *testing.T) {
	res, _ := safeFindFile(nil, "x")
	assert.Nil(t, res)
}

//goland:noinspection GoSnakeCaseUsage
var (
	testDir_x = &Dir{
		Name: "x",
		Dirs: []*Dir{testDir_r, testDir_y},
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
		{dir: testDir_x, name: "x", want: nil},
		{dir: testDir_x, name: "y", want: testDir_y},
		{dir: testDir_x, name: "r", want: testDir_r},
		{dir: testDir_y, name: "y", want: nil},
		{dir: testDir_y, name: "z", want: testDir_z},
		{dir: testDir_z, name: "", want: nil},
		{dir: testDir_r, name: "s", want: testDir_r.Dirs[0]},
		{dir: testDir_r, name: "t", want: testDir_r.Dirs[1]},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%v/%v", test.dir.Name, test.name), func(t *testing.T) {
			res, _ := safeFindDir(test.dir, test.name)
			assert.Equal(t, test.want, res)
			assert.True(t, test.want == res)
		})
	}
}

func Test__safeFindFile_finds_file(t *testing.T) {
	tests := []struct {
		dir  *Dir
		name string
		want *File
	}{
		{dir: testDir_x, name: "x", want: nil},
		{dir: testDir_x, name: "a", want: testDir_x.Files[0]},
		{dir: testDir_x, name: "b", want: testDir_x.Files[1]},
		{dir: testDir_x, name: "c", want: testDir_x.Files[2]},

		{dir: testDir_y, name: "y", want: nil},
		{dir: testDir_y, name: "r", want: testDir_y.Files[0]},
		{dir: testDir_y, name: "s", want: testDir_y.Files[1]},

		{dir: testDir_z, name: "", want: nil},
		{dir: testDir_z, name: "a", want: testDir_z.Files[0]},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%v/%v", test.dir.Name, test.name), func(t *testing.T) {
			res, _ := safeFindFile(test.dir, test.name)
			assert.Equal(t, test.want, res)
			assert.True(t, test.want == res)
		})
	}
}
