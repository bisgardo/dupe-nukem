package main

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/match"
	"github.com/bisgardo/dupe-nukem/scan"
)

// Match computes the hash-based matches between the files recorded in the scan file located at the path srcScanFile
// and the files recorded in the scan files located at paths targetScanFiles.
func Match(srcScanFilePath string, targetScanFilePaths []string) (*match.Result, error) {
	time0 := time.Now()
	log.Printf("loading source scan file %q...\n", srcScanFilePath)
	srcRoot, err := loadScanDirFile(srcScanFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load source scan file %q", srcScanFilePath)
	}
	targets, err := loadTargets(targetScanFilePaths, srcScanFilePath, srcRoot)
	if err != nil {
		return nil, err
	}
	time1 := time.Now()
	log.Printf("all scan files loaded successfully in %v\n", timeBetween(time0, time1))
	res := match.Run(srcRoot, targets)
	time2 := time.Now()
	log.Printf("match completed successfully in %v\n", timeBetween(time1, time2))
	return res, nil
}

type targetKey struct {
	cleanPath string
	id        match.TargetID
}

func loadTargets(idPaths []string, srcScanFilePath string, srcScanDir *scan.Dir) ([]match.Target, error) {
	keys := make([]targetKey, len(idPaths))
	for i, p := range idPaths {
		id, path := extractId(p, '=')
		cleanPath := filepath.Clean(path)
		for _, k := range keys {
			if k.cleanPath == cleanPath {
				return nil, errors.Errorf("duplicate path %q on target #%d", id, i+1)
			}
		}
		keys = append(keys, targetKey{
			cleanPath: cleanPath,
			id: match.TargetID{
				ID: id,
			},
		})
	}

	// TODO [optimization] Load in parallel?
	res := make([]match.Target, len(idPaths))
	for i, k := range keys {
		scanDir, err := loadTargetScanDir(k.cleanPath, srcScanFilePath, srcScanDir)
		if err != nil {
			return nil, err
		}
		index := match.BuildIndex(scanDir)
		res[i] = match.Target{
			Index: index,
			ID:    k.id,
		}
	}
	return res, nil
}

func extractId(p string, splitter rune) (string, string) {
	if i := strings.IndexRune(p, splitter); i >= 0 {
		return p[0:i], p[i+1:]
	}
	return p, p
}

func loadTargetScanDir(path string, srcScanFilePath string, srcScanDir *scan.Dir) (*scan.Dir, error) {
	if filepath.Clean(path) == filepath.Clean(srcScanFilePath) {
		log.Printf("reusing loaded source scan file %q as a target\n", srcScanFilePath)
		return srcScanDir, nil
	}
	log.Printf("loading target scan file %q\n", srcScanFilePath)
	return loadScanDirFile(path)
}
