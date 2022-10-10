package match

import (
	"bytes"
	"sort"

	"github.com/bisgardo/dupe-nukem/scan"
)

// HashMatch is a hash value and the paths of the files in the target directories whose contents hash to this value.
// TODO As the target's roots could have the same name, the matched paths must be prefixed by a target ID.
//      An ok initial solution could be to reject targets with similar roots.
//      Alternatively, we could use the scan file paths as ID.
//      Or have 'scan' require an ID to be provided which is recorded into the scan file (could default to root name),
//      and reject targets with the same ID - or allowing overwrite on the command line!
//      That sounds like the scan file should have a root type (which could also record the absolute dir path...).
type HashMatch struct {
	Hash  uint64   `json:"hash"`
	Paths []string `json:"paths"`
}

// Run computes the hash-based matches between the files recorded in the scan file located at the path srcScanFile
// and the files recorded in the scan files located at paths targetScanFiles.
func Run(srcRoot *scan.Dir, targets []Index) ([]HashMatch, error) {
	matches := BuildMatch(srcRoot, targets)
	return sortedHashMatches(matches), nil
}

func sortedHashMatches(m Matches) []HashMatch {
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

func sortedHashes(m Matches) []uint64 {
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
		writeFilePath(f, &buf)
		res[i] = buf.String()
	}
	return res
}

func writeFilePath(f *File, buf *bytes.Buffer) {
	writeDirPath(f.Dir, buf)
	buf.WriteRune('/')
	buf.WriteString(f.ScanFile.Name)
}

func writeDirPath(d *Dir, buf *bytes.Buffer) {
	if d.Parent != nil {
		writeDirPath(d.Parent, buf)
		buf.WriteRune('/')
	}
	buf.WriteString(d.ScanDir.Name)
}
