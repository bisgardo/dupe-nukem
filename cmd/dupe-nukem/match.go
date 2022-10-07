package main

import (
	"github.com/bisgardo/dupe-nukem/match"
	"github.com/bisgardo/dupe-nukem/scan"
)

// Match computes the hash-based matches between the files recorded in the scan file located at the path srcScanFile
// and the files recorded in the scan files located at paths targetScanFiles.
func Match(srcScanFile string, targetScanFiles []string) ([]match.HashMatch, error) {
	srcRoot, err := loadSourceDir(srcScanFile)
	if err != nil {
		return nil, err
	}
	targetIndexes, err := loadTargetIndexes(targetScanFiles)
	if err != nil {
		return nil, err
	}
	return match.Run(srcRoot, targetIndexes)
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
