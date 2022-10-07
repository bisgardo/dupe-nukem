package main

import (
	"bytes"
	"sort"

	"github.com/bisgardo/dupe-nukem/match"
	"github.com/bisgardo/dupe-nukem/scan"
)

type HashMatches struct {
	Hash    uint64
	Matches []string
}

func Match(srcScanFile string, targetScanFiles []string) ([]HashMatches, error) {
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
	// TODO Optimization: If a target is also the source there's no need for loading the file again.
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

func sortedHashMatches(m match.Matches) []HashMatches {
	hashes := sortedHashes(m)
	res := make([]HashMatches, len(hashes))
	for i, h := range hashes {
		res[i] = HashMatches{
			Hash:    h,
			Matches: toFilePaths(m[h]),
		}
	}
	return res
}

func sortedHashes(m match.Matches) []uint64 {
	hashes := make([]uint64, 0, len(m))
	for h := range m {
		hashes = append(hashes, h)
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	return hashes
}

func toFilePaths(files match.FileSet) []string {
	res := make([]string, 0, len(files))
	for f := range files {
		var buf bytes.Buffer
		writeFilePath(f, buf)
		res = append(res, buf.String())
	}
	sort.Strings(res)
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
