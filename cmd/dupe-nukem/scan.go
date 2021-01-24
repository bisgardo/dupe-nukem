package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/pkg/errors"
)

func Scan(dir string, skip string) (*scan.Dir, error) {
	skipDirs, err := parseSkipNames(skip)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse skip dirs expression %q", skip)
	}
	// TODO Replace '.' with working dir.
	return scan.Run(filepath.Clean(dir), skipDirs)
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
