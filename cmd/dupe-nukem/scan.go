package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/bisgardo/dupe-nukem/util"
	"github.com/pkg/errors"
)

func Scan(dir, skip, cache string) (*scan.Dir, error) {
	skipDirs, err := parseSkipNames(skip)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse skip dirs expression %q", skip)
	}
	cacheDir, err := loadCacheDir(cache)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load cache file %q", cache)
	}
	// TODO Replace '.' with working dir?
	return scan.Run(filepath.Clean(dir), skipDirs, cacheDir)
}

func parseSkipNames(input string) (scan.ShouldSkipPath, error) {
	if input == "" {
		return scan.NoSkip, nil
	}
	names := strings.Split(input, ",")
	res := make(map[string]struct{}, len(names))
	for _, n := range names {
		if err := validateSkipName(n); err != nil {
			return nil, errors.Wrapf(err, "invalid skip name %q", n)
		}
		res[n] = struct{}{}
	}
	return func(dir, name string) bool {
		_, ok := res[name]
		return ok
	}, nil
}

func validateSkipName(name string) error {
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("has surrounding space")
	}
	switch name {
	case "":
		return fmt.Errorf("empty")
	case ".":
		return fmt.Errorf("current directory")
	case "..":
		return fmt.Errorf("parent directory")
	}
	if i := strings.IndexAny(name, "/"); i != -1 {
		return fmt.Errorf("has invalid character '%c'", name[i])
	}
	return nil
}

func loadCacheDir(path string) (*scan.Dir, error) {
	if path == "" {
		return nil, nil
	}
	start := time.Now()
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(util.SimplifyIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: cannot close cache file '%v': %v\n", path, err)
		}
	}()
	var cacheDir scan.Dir
	// TODO Decompress file if gzipped.
	if err := json.NewDecoder(f).Decode(&cacheDir); err != nil {
		return nil, errors.Wrap(err, "cannot decode file as JSON")
	}

	// TODO Unless it's too expensive, just sorts lists instead of validating.
	if err := checkCache(&cacheDir); err != nil {
		return nil, errors.Wrap(err, "invalid cache contents")
	}
	log.Printf("cache loaded from %q in %v\n", path, timeSince(start))
	return &cacheDir, nil
}

func checkCache(dir *scan.Dir) error {
	// Check subdirs.
	var ld *scan.Dir
	for i, d := range dir.Dirs {
		if ld != nil && ld.Name > d.Name {
			return fmt.Errorf("list of subdirectories of %q is not sorted: %q on index %d should come before %q on index %d", dir.Name, d.Name, i, ld.Name, i-1)
		}
		ld = d

		// Recurse.
		if err := checkCache(d); err != nil {
			return errors.Wrapf(err, "in subdirectory %q on index %d", d.Name, i)
		}
	}
	// Check non-empty files.
	var lf *scan.File
	for i, f := range dir.Files {
		if lf != nil && lf.Name > f.Name {
			return fmt.Errorf("list of non-empty files in directory %q is not sorted: %q on index %d should come before %q on index %d", dir.Name, f.Name, i, lf.Name, i-1)
		}
		lf = f
	}
	// Check empty files.
	var lef string
	for i, ef := range dir.EmptyFiles {
		if lef > ef {
			return fmt.Errorf("list of empty files in directory %q is not sorted: %q on index %d should come before %q on index %d", dir.Name, ef, i, lef, i-1)
		}
		lef = ef
	}
	return nil
}
