package main

import (
	"fmt"
	"os"
)

var errFileNotFound = fmt.Errorf("file not found")

func cleanFileNotFoundError(err error) error {
	if os.IsNotExist(err) {
		return errFileNotFound
	}
	return err
}
