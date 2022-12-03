package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/bisgardo/dupe-nukem/scan"
	"github.com/bisgardo/dupe-nukem/util"
)

func resolveReader(path string, f *os.File) (io.Reader, error) {
	// TODO Read magic number instead of extension.
	if strings.HasSuffix(path, ".gz") {
		return gzip.NewReader(f)
	}
	return f, nil
}

func loadScanDirFile(path string) (*scan.Dir, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(util.SimplifyIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error closing scan file %q: %v\n", path, err)
		}
	}()
	r, err := resolveReader(path, f)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve file reader")
	}
	var res scan.Dir
	err = json.NewDecoder(r).Decode(&res)
	return &res, errors.Wrapf(err, "cannot decode file as JSON")
}

func absPath(path string) (string, error) {
	a, err := filepath.Abs(path)
	if err != nil {
		return "", errors.Wrapf(util.SimplifyIOError(err), "cannot resolve absolute path of %q", path)
	}
	//if runtime.GOOS == "windows" {
	//	return `\\?\` + a, nil // hack to enable long paths on Windows
	//}
	return a, nil
}
