package testutil

import (
	"bytes"
	"log"
)

// LogBuffer redirects output from the builtin logger to a new buffer and returns this buffer.
func LogBuffer() *bytes.Buffer {
	var buf bytes.Buffer
	log.SetFlags(0)
	log.SetOutput(&buf)
	return &buf
}
