package scan

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
	if name == "" {
		panic("directory name cannot be empty")
	}
	return &Dir{Name: name}
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

// TODO: Add function for validating (or ensuring?) that the lists are indeed ordered correctly.

// File represents a file as a name, size, modification time, and fnv hash.
type File struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"ts"`
	Hash    uint64 `json:"hash"`
}

// NewFile constructs a File.
func NewFile(name string, size int64, modTime int64, hash uint64) *File {
	if name == "" {
		panic("file name cannot be empty")
	}
	return &File{
		Name:    name,
		Size:    size,
		ModTime: modTime,
		Hash:    hash,
	}
}

// safeFindDir looks for a Dir with the given name in the subdirectory list of the given Dir.
// Returns nil if the Dir is nil or doesn't have a subdirectory with that name.
func safeFindDir(d *Dir, name string) *Dir {
	if d == nil {
		return nil
	}
	// TODO: Use binary search.
	for _, s := range d.Dirs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// safeFindFile looks for a File with the given name in the file list of the given Dir.
// Returns nil if the Dir is nil or doesn't have a file with that name.
func safeFindFile(d *Dir, name string) *File {
	if d == nil {
		return nil
	}
	// TODO: Use binary search.
	for _, f := range d.Files {
		if f.Name == name {
			return f
		}
	}
	return nil
}
