package main

import (
	"log"
	"time"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/match"
	"github.com/bisgardo/dupe-nukem/scan"
)

// Match computes the hash-based matches between the files recorded in the scan file located at the path srcScanFile
// and the files recorded in the scan files located at paths targetScanFiles.
func Match(srcScanFile string, targetScanFiles []string) (*match.Result, error) {
	time0 := time.Now()
	srcRoot, err := loadScanDir(srcScanFile)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load source scan file %q", srcScanFile)
	}
	targets, err := loadTargets(targetScanFiles)
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

func loadScanDir(path string) (*scan.Dir, error) {
	// TODO Expect 'path' to actually be '<ID>=<path>' and prefix matches by the ID.
	log.Printf("loading scan file %q...\n", path)
	return loadScanDirFile(path)
}

func loadTargets(paths []string) ([]match.Target, error) {
	// TODO [optimization] If a target is also the source there's no need for loading the file again.
	// TODO [optimization] Load files in parallel?
	res := make([]match.Target, len(paths))
	ids := make(map[string]struct{})
	for i, path := range paths {
		scanDir, err := loadScanDir(path)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot load target #%d scan file %q", i+1, path)
		}
		index := match.BuildIndex(scanDir)
		id := path // TODO should be overridable
		if _, ok := ids[id]; ok {
			return nil, errors.Errorf("duplicate ID %q for target #%d", id, i+1)
		}
		ids[id] = struct{}{}
		res[i] = match.Target{
			Index: index,
			ID: match.TargetID{
				ID: id,
			},
		}
	}
	return res, nil
}
