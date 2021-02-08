package util

import "os"

// FileModeName returns the name of the file's "mode".
func FileModeName(m os.FileMode) string {
	switch {
	case m&os.ModeDir != 0:
		return "directory"
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
	return "file"
}
