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

func resolveReader(f *os.File) (io.Reader, error) {
	// TODO: Read magic number instead of extension (then update test 'Test__scan_testdata_uses_provided_cache').
	if strings.HasSuffix(f.Name(), ".gz") {
		return gzip.NewReader(f)
	}
	return f, nil
}

func loadScanResultFile(path string) (*scan.Result, error) {
	// TODO: Pass in 'open' function to enable tests to return error, create fake file, disallow closing, etc.
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(util.SimplifyIOError(err), "cannot open file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error: closing scan file %q failed: %v\n", path, err) // cannot test
		}
	}()
	r, err := resolveReader(f)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve file reader")
	}
	return decodeScanResult(r)
}

func decodeScanResult(r io.Reader) (*scan.Result, error) {
	var res scan.Result
	err := json.NewDecoder(r).Decode(&res)
	return &res, util.JSONError(err)
}

func absPath(path string) (string, error) {
	a, err := filepath.Abs(path)
	if err != nil {
		return "", errors.Wrapf(util.SimplifyIOError(err), "cannot resolve absolute path of %q", path) // only tested on Windows ('Test__Scan_wraps_invalid_dir_error')
	}
	//if runtime.GOOS == "windows" {
	//	return `\\?\` + a, nil // hack to enable long paths on Windows
	//}
	return a, nil
}
