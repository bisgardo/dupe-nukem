package main

import (
	"bytes"
	"sort"

	"github.com/bisgardo/dupe-nukem/match"
	"github.com/bisgardo/dupe-nukem/scan"
)

// HashMatch is a hash value and the paths of the files in the target directories whose contents hash to this value.
type HashMatch struct {
	Hash  uint64
	Paths []string
}

// Match computes the hash-based matches between the files recorded in the scan file located at the path srcScanFile
// and the files recorded in the scan files located at paths targetScanFiles.
func Match(srcScanFile string, targetScanFiles []string) ([]HashMatch, error) {
	srcRoot, err := loadSourceDir(srcScanFile)
	if err != nil {
		return nil, err
	}
	targets, err := loadTargetIndexes(targetScanFiles)
	if err != nil {
		return nil, err
	}
	res := match.BuildMatches(srcRoot, targets)
	return sortedHashMatches(res), nil
}

func loadSourceDir(path string) (*scan.Dir, error) {
	return loadScanFile(path)
}

func loadTargetIndexes(paths []string) ([]match.Index, error) {
	// TODO [optimization] If a target is also the source there's no need for loading the file again.
	res := make([]match.Index, len(paths))
	for i, path := range paths {
		scanDir, err := loadScanFile(path)
		if err != nil {
			return nil, err
		}
		res[i] = match.BuildIndex(scanDir)
	}
	return res, nil
}

func sortedHashMatches(m match.Index) []HashMatch {
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

func sortedHashes(m match.Index) []uint64 {
	hashes := make([]uint64, 0, len(m))
	for h := range m {
		hashes = append(hashes, h)
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	return hashes
}

func toFilePaths(files []*match.File) []string {
	res := make([]string, len(files))
	for i, f := range files {
		var buf bytes.Buffer
		writeFilePath(f, buf)
		res[i] = buf.String()
	}
	return res
}

func writeFilePath(f *match.File, buf bytes.Buffer) {
	writeDirPath(f.Dir, buf)
	buf.WriteRune('/')
	buf.WriteString(f.ScanFile.Name)
}

func writeDirPath(d *match.Dir, buf bytes.Buffer) {
	if d.Parent != nil {
		writeDirPath(d.Parent, buf)
		buf.WriteRune('/')
	}
	buf.WriteString(d.ScanDir.Name)
}
