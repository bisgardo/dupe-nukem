package match

import (
	"github.com/bisgardo/dupe-nukem/scan"
)

// Index is a map from hash to the files whose contents hash to this hash.
type Index map[uint64][]*File

type Dir struct {
	Parent  *Dir
	ScanDir *scan.Dir
}

func NewDir(parent *Dir, scanDir *scan.Dir) *Dir {
	return &Dir{
		Parent:  parent,
		ScanDir: scanDir,
	}
}

type File struct {
	Dir      *Dir
	ScanFile *scan.File
}

func NewFile(dir *Dir, scanFile *scan.File) *File {
	return &File{
		Dir:      dir,
		ScanFile: scanFile,
	}
}

func innerBuildIndex(scanDir *scan.Dir, parent *Dir, res Index) {
	dir := NewDir(parent, scanDir)
	for _, f := range scanDir.Files {
		res[f.Hash] = append(res[f.Hash], NewFile(dir, f))
	}
	for _, d := range scanDir.Dirs {
		innerBuildIndex(d, dir, res)
	}
}

func BuildIndex(root *scan.Dir) Index {
	res := make(Index)
	innerBuildIndex(root, nil, res)
	return res
}
