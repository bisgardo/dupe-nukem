package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/bisgardo/dupe-nukem/util"
)

// maxSkipNameFileLineLen is the size in bytes allocated for reading a skip file.
// As the file is read line by line, this is the maximum allowed line length.
const maxSkipNameFileLineLen = 256

// Scan parses the skip expression and cache path passed from the command line
// and then runs scan.Run with the resulting values.
func Scan(dir, skipExpr, cachePath string) (*scan.Dir, error) {
	shouldSkip, err := loadShouldSkip(skipExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot process skip dirs expression %q", skipExpr)
	}
	cacheDir, err := loadScanDirCacheFile(cachePath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load scan cache file %q", cachePath)
	}
	absDir, err := absPath(dir)
	if err != nil {
		return nil, err
	}
	if absDir != dir {
		log.Printf("absolute path of %q resolved to %q\n", dir, absDir)
	}
	return scan.Run(absDir, shouldSkip, cacheDir)
}

func loadShouldSkip(expr string) (scan.ShouldSkipPath, error) {
	names, err := parseSkipNames(expr)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return scan.NoSkip, nil
	}
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		if err := validateSkipName(n); err != nil {
			return nil, errors.Wrapf(err, "invalid skip name %q", n)
		}
		set[n] = struct{}{}
	}
	return scan.SkipNameSet(set), nil
}

func parseSkipNames(input string) ([]string, error) {
	if len(input) == 0 {
		return nil, nil
	}
	if input[0] == '@' {
		f := input[1:]
		res, err := parseSkipNameFile(f)
		return res, errors.Wrapf(err, "cannot read skip names from file %q", f)
	}
	return strings.Split(input, ","), nil
}

func parseSkipNameFile(path string) ([]string, error) {
	// TODO: Pass 'open' function (see comment in 'loadScanDirFile').
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(util.SimplifyIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: cannot close skip name file %q: %v\n", path, err) // cannot test
		}
	}()
	r := bufio.NewReaderSize(f, maxSkipNameFileLineLen)
	var names []string
	i := 0
	for {
		i++
		l, isNotSuffix, err := r.ReadLine()
		if err == io.EOF {
			return names, nil
		}
		if err != nil {
			return nil, err
		}
		if isNotSuffix {
			return nil, fmt.Errorf("line %d is longer than the max allowed length of %d characters", i, maxSkipNameFileLineLen)
		}
		if n := strings.TrimSpace(string(l)); n != "" {
			names = append(names, n)
		}
	}
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

func loadScanDirCacheFile(path string) (*scan.Dir, error) {
	if path == "" {
		return nil, nil
	}
	log.Printf("loading scan cache file %q...\n", path)
	start := time.Now()
	scanDir, err := loadScanDirFile(path)
	if err != nil {
		return nil, err
	}
	// TODO: Unless it's too expensive, just sort lists instead of only validating?
	if err := checkCache(scanDir); err != nil {
		return nil, errors.Wrap(err, "invalid cache contents")
	}
	log.Printf("scan cache loaded successfully from %q in %v\n", path, timeSince(start))
	return scanDir, nil
}

func checkCache(dir *scan.Dir) error {
	// Check subdirs.
	var ld *scan.Dir
	for i, d := range dir.Dirs {
		// Check that list is sorted.
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
		// Check that list is sorted.
		if lf != nil && lf.Name > f.Name {
			return fmt.Errorf("list of non-empty files in directory %q is not sorted: %q on index %d should come before %q on index %d", dir.Name, f.Name, i, lf.Name, i-1)
		}
		lf = f

		// Check that name is non-empty.
		if f.Name == "" {
			return fmt.Errorf("name of non-empty file on index %d is empty", i)
		}

		// Check that size is non-zero.
		if f.Size == 0 {
			return fmt.Errorf("non-empty file %q on index %d has size 0", f.Name, i)
		}

		// Check if hash is zero.
		if f.Hash == 0 {
			log.Printf("warning: file %q is cached with hash 0 - this hash will be ignored\n", f.Name)
		}
	}
	// Check empty files.
	var lef string
	for i, ef := range dir.EmptyFiles {
		// Check that list is sorted.
		if lef > ef {
			return fmt.Errorf("list of empty files in directory %q is not sorted: %q on index %d should come before %q on index %d", dir.Name, ef, i, lef, i-1)
		}
		lef = ef

		// Check that name is non-empty.
		if ef == "" {
			return fmt.Errorf("name of empty file on index %d is empty", i)
		}
	}
	return nil
}
