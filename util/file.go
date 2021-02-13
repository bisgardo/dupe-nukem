package util

import (
	"os"
)

// FileInfoModeName returns the name of the file's "mode".
func FileInfoModeName(i os.FileInfo) string {
	if i == nil {
		return "file or directory"
	}
	return FileModeName(i.Mode())
}

// FileModeName returns the name of the file's "mode".
func FileModeName(m os.FileMode) string {
	switch {
	case m.IsDir():
		return "directory"
	case m.IsRegular():
		return "file"
	case m&os.ModeSymlink != 0:
		return "symlink"
	case m&os.ModeNamedPipe != 0:
		return "named pipe file"
	case m&os.ModeSocket != 0:
		return "socket file"
	case m&os.ModeDevice != 0:
		return "device file"
	case m&os.ModeCharDevice != 0:
		return "char device file"
	case m&os.ModeIrregular != 0:
		return `"irregular" file`
	}
	return "<" + m.String() + ">"
}
