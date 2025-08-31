package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/bisgardo/dupe-nukem/util"
)

// maxSkipNameFileLineLen is the size in bytes allocated for reading a skip file.
// As the file is read line by line, this is the maximum allowed line length.
const maxSkipNameFileLineLen = 256

// invalidSkipNameChars is a sequence of the Unicode code points that a valid skipname is not allowed to contain.
// The characters are deemed invalid to avoid giving the impression that the expression supports nesting or regex.
var invalidSkipNameChars = map[rune]struct{}{'/': {}, '*': {}, '?': {}}

func init() {
	// Ensure that path separator is included on systems where it isn't '/' (i.e. Windows).
	invalidSkipNameChars[filepath.Separator] = struct{}{}
}

// Scan parses the skip expression and cache path passed from the command line
// and then runs scan.Run with the resulting values.
func Scan(dir, skipExpr, cachePath string) (*scan.Result, error) {
	shouldSkip, err := loadShouldSkip(skipExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot process skip dirs expression %q", skipExpr)
	}
	cache, err := loadScanCache(cachePath)
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
	runStart := time.Now()
	run, err := scan.Run(absDir, shouldSkip, cache)
	if err != nil {
		return nil, err
	}
	log.Printf("scan completed successfully in %v\n", timeSince(runStart))
	return run, nil
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
		return nil, errors.Wrapf(util.CleanIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: closing skip name file %q failed: %v\n", path, err) // cannot test
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
		return fmt.Errorf("surrounding space")
	}
	switch name {
	case "":
		return fmt.Errorf("empty")
	case ".":
		return fmt.Errorf("current directory")
	case "..":
		return fmt.Errorf("parent directory")
	}
	for i, c := range name {
		if _, ok := invalidSkipNameChars[c]; ok {
			return fmt.Errorf("invalid character '%c'", name[i])
		}
	}
	return nil
}

func loadScanCache(path string) (*scan.Dir, error) {
	if path == "" {
		return nil, nil
	}
	log.Printf("loading scan cache file %q...\n", path)
	start := time.Now()
	cacheRoot, err := loadScanCacheResultRoot(path)
	if err != nil {
		return nil, err
	}
	if cacheRoot == nil {
		return nil, errors.Errorf("no root")
	}
	// Could just sort lists instead of (only) validating,
	// but it appears to be a needless complication for something that should never happen.
	// So if it does, it probably indicates a problem that's worth alarming the user about.
	if err := checkCacheRoot(cacheRoot); err != nil {
		return nil, errors.Wrap(err, "invalid root") // caller wraps path
	}
	log.Printf("scan cache loaded successfully from %q in %v\n", path, timeSince(start))
	return cacheRoot, nil
}

func loadScanCacheResultRoot(path string) (*scan.Dir, error) {
	res, err := loadScanResultFile(path)
	if err != nil {
		return nil, err
	}
	return res.Root, checkResultTypeVersion(res.TypeVersion)
}

func checkResultTypeVersion(v int) error {
	if v == 0 {
		return errors.Errorf("schema version is missing")
	}
	if v != scan.CurrentResultTypeVersion {
		return errors.Errorf("unsupported schema version: %d", v)
	}
	return nil
}

func checkCacheRoot(root *scan.Dir) error {
	// Require non-empty name.
	if root.Name == "" {
		return fmt.Errorf("directory has no name")
	}

	// Check subdirs.
	var ld *scan.Dir
	for i, d := range root.Dirs {
		// Require lexical order.
		if ld != nil && ld.Name > d.Name {
			return fmt.Errorf(
				"list of subdirectories of %q is not sorted: %q on index %d should come before %q on index %d",
				root.Name, d.Name, i, ld.Name, i-1,
			)
		}
		ld = d

		// Recurse.
		if err := checkCacheRoot(d); err != nil {
			return errors.Wrapf(err, "in subdirectory %q on index %d", d.Name, i)
		}
	}
	// Check non-empty files (empty files/dirs aren't used for caching).
	var lf *scan.File
	for i, f := range root.Files {
		// Require lexical order.
		if lf != nil && lf.Name > f.Name {
			return fmt.Errorf(
				"list of non-empty files in directory %q is not sorted: %q on index %d should come before %q on index %d",
				root.Name, f.Name, i, lf.Name, i-1,
			)
		}
		lf = f

		// Require non-empty name and non-zero size.
		if f.Name == "" {
			return fmt.Errorf("file on index %d has no name", i)
		}
		if f.Size == 0 {
			return fmt.Errorf("file %q on index %d has size 0, but is not listed as empty", f.Name, i)
		}
		// Log warning if hash is zero.
		if f.Hash == 0 {
			log.Printf("warning: file %q is cached with hash 0 - this hash will be recomputed\n", f.Name)
		}
		// Timestamps are used by the cache, but any value is valid, so there's nothing to check.
	}
	return nil
}
