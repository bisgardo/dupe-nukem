package scan

// Dir represents a directory as a name and lists of contained files and subdirectories.
// All of these lists must be sorted to enable binary search.
type Dir struct {
	// Name of the directory relative to its parent.
	// For the root dir, this is the path that was passed to Run.
	Name string `json:"name"`
	// Sorted list of the subdirectories of the directory.
	Dirs []*Dir `json:"dirs,omitempty"`
	// Sorted list of files in the directory.
	Files []*File `json:"files,omitempty"`
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

// TODO Add function for validating (or ensuring?) that the lists are indeed ordered correctly.

// File represents a file as a name, size, and fnv hash.
type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// NewFile constructs a File.
func NewFile(name string, size int64) *File {
	return &File{
		Name: name,
		Size: size,
	}
}
