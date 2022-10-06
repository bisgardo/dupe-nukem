package match

import "github.com/bisgardo/dupe-nukem/scan"

// Index is a map from hash to the files with this hash.
type Index map[uint64][]*File

type Dir struct {
	parent  *Dir
	scanDir *scan.Dir
}

func NewDir(parent *Dir, scanDir *scan.Dir) *Dir {
	return &Dir{
		parent:  parent,
		scanDir: scanDir,
	}
}

type File struct {
	dir      *Dir
	scanFile *scan.File
}

func NewFile(dir *Dir, scanFile *scan.File) *File {
	return &File{
		dir:      dir,
		scanFile: scanFile,
	}
}

func buildIndexRecursively(scanDir *scan.Dir, parent *Dir, res Index) {
	dir := NewDir(parent, scanDir)
	for _, f := range scanDir.Files {
		res[f.Hash] = append(res[f.Hash], NewFile(dir, f))
	}
	for _, d := range scanDir.Dirs {
		buildIndexRecursively(d, dir, res)
	}
}

func BuildIndex(root *scan.Dir, parent *Dir) Index {
	res := make(Index)
	buildIndexRecursively(root, nil, res)
	return res
}
