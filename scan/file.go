package scan

const useBinarySearch = true

// Dir represents a directory as a name and lists of contained files and subdirectories.
// All of these lists must be sorted to enable binary search.
type Dir struct {
	// Name of the directory relative to its parent.
	// For the root dir, this is the path that was passed to Run.
	Name string `json:"name"`
	// Sorted list of the subdirectories of the directory.
	Dirs []*Dir `json:"dirs,omitempty"`
	// Sorted list of non-empty files in the directory.
	Files []*File `json:"files,omitempty"`
	// Sorted list of empty files in the directory.
	EmptyFiles []string `json:"empty_files,omitempty"`
	// Sorted list of files in the directory that were skipped when scanning.
	SkippedFiles []string `json:"skipped_files,omitempty"`
	// Sorted list of subdirectories of the directory that were skipped when scanning.
	SkippedDirs []string `json:"skipped_dirs,omitempty"`
}

// NewDir constructs a Dir.
func NewDir(name string) *Dir {
	return &Dir{
		Name:  name,
		Dirs:  nil,
		Files: nil,
	}
}

// appendDir appends a Dir to the list of subdirectories.
// The usage pattern must ensure that this doesn't break the ordering constraint
// as the function doesn't ensure nor check this.
func (d *Dir) appendDir(s *Dir) {
	d.Dirs = append(d.Dirs, s)
}

// appendFile appends a File to the list of files.
// The usage pattern must ensure that this doesn't break the ordering constraint
// as the function doesn't ensure nor check this.
func (d *Dir) appendFile(f *File) {
	d.Files = append(d.Files, f)
}

// appendEmptyFile appends a file name to the list of empty files.
// The usage pattern must ensure that this doesn't break the ordering constraint
// as the function doesn't ensure nor check this.
func (d *Dir) appendEmptyFile(fileName string) {
	d.EmptyFiles = append(d.EmptyFiles, fileName)
}

// appendSkippedFile appends the file name to the list of files that were skipped by scan.
// The usage pattern must ensure that this doesn't break the ordering constraint
// as the function doesn't ensure nor check this.
func (d *Dir) appendSkippedFile(name string) {
	d.SkippedFiles = append(d.SkippedFiles, name)
}

// appendSkippedDir appends the dir name to the list of subdirectories that were skipped by scan.
// The usage pattern must ensure that this doesn't break the ordering constraint
// as the function doesn't ensure nor check this.
func (d *Dir) appendSkippedDir(dirName string) {
	d.SkippedDirs = append(d.SkippedDirs, dirName)
}

// TODO Add function for validating (or ensuring?) that the lists are indeed ordered correctly.

// File represents a file as a name, size, and fnv hash.
type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Hash uint64 `json:"hash,omitempty"`
}

// NewFile constructs a File.
func NewFile(name string, size int64, hash uint64) *File {
	return &File{
		Name: name,
		Size: size,
		Hash: hash,
	}
}

// safeFindDir looks for a Dir with the given name in the subdirectory list of the given Dir.
// Returns nil if the Dir is nil.
func safeFindDir(d *Dir, name string) (*Dir, int) {
	if d == nil {
		return nil, 0
	}

	if useBinarySearch {
		return findDir(d.Dirs, name, len(d.Dirs)/2)
	}

	for i, s := range d.Dirs {
		if s.Name == name {
			return s, i
		}
	}
	return nil, 0
}

// safeFindFile looks for a File with the given name in the file list of the given Dir.
// Returns nil if the Dir is nil.
func safeFindFile(d *Dir, name string) (*File, int) {
	if d == nil {
		return nil, 0
	}

	if useBinarySearch {
		return findFile(d.Files, name, len(d.Files)/2)
	}

	for i, f := range d.Files {
		if f.Name == name {
			return f, i
		}
	}
	return nil, 0
}

// findDir searches the provided list for a Dir with the provided name.
// The search is performed using binary search, so the input list must be sorted.
// TODO The provided index determines the first element to be considered.
//      This may be used to improve search performance if a good candidate is known in advance
//      (say, the element following the previous match).
// The implementation is exactly identical to findFile but it necessary to implement separately
// due to the lack of generics in Go.
func findDir(ds []*Dir, name string, idx int) (*Dir, int) {
	l, r := 0, len(ds)-1
	//if idx > r {
	//	idx = (l + r) / 2
	//}
	for l <= r {
		d := ds[idx]
		n := d.Name
		if n == name {
			return d, idx
		}
		if n < name {
			l = idx + 1
		} else {
			r = idx - 1
		}
		idx = (l + r) / 2
	}
	return nil, 0
}

// findDir searches the provided list for a File with the provided name.
// The search is performed using binary search, so the input list must be sorted.
// The implementation is exactly identical to findDir but it necessary to implement separately
// due to the lack of generics in Go.
func findFile(ds []*File, name string, idx int) (*File, int) {
	l, r := 0, len(ds)-1
	//if idx > r {
	//	idx = (l + r) / 2
	//}
	for l <= r {
		f := ds[idx]
		n := f.Name
		if n == name {
			return f, idx
		}
		if n < name {
			l = idx + 1
		} else {
			r = idx - 1
		}
		idx = (l + r) / 2
	}
	return nil, 0
}
