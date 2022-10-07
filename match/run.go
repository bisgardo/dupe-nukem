package match

import (
	"bytes"
	"sort"

	"github.com/bisgardo/dupe-nukem/scan"
)

// HashMatch is a hash value and the paths of the files in the target directories whose contents hash to this value.
type HashMatch struct {
	Hash  uint64
	Paths []string
}

// Run computes the hash-based matches between the files recorded in the scan file located at the path srcScanFile
// and the files recorded in the scan files located at paths targetScanFiles.
func Run(srcRoot *scan.Dir, targets []Index) ([]HashMatch, error) {
	matches := BuildMatchIndex(srcRoot, targets)
	return sortedHashMatches(matches), nil
}

func sortedHashMatches(m Index) []HashMatch {
	hashes := sortedHashes(m)
	res := make([]HashMatch, len(hashes))
	for i, h := range hashes {
		res[i] = HashMatch{
			Hash:  h,
			Paths: toFilePaths(m[h]),
		}
	}
	return res
}

func sortedHashes(m Index) []uint64 {
	hashes := make([]uint64, 0, len(m))
	for h := range m {
		hashes = append(hashes, h)
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	return hashes
}

func toFilePaths(files []*File) []string {
	res := make([]string, len(files))
	for i, f := range files {
		var buf bytes.Buffer
		writeFilePath(f, buf)
		res[i] = buf.String()
	}
	return res
}

func writeFilePath(f *File, buf bytes.Buffer) {
	writeDirPath(f.Dir, buf)
	buf.WriteRune('/')
	buf.WriteString(f.ScanFile.Name)
}

func writeDirPath(d *Dir, buf bytes.Buffer) {
	if d.Parent != nil {
		writeDirPath(d.Parent, buf)
		buf.WriteRune('/')
	}
	buf.WriteString(d.ScanDir.Name)
}
