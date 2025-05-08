package testutil

import (
	"bytes"
	"fmt"
	"log"
)

// CollectLogs returns a collector of logs emitted with the [log] package.
func CollectLogs() fmt.Stringer {
	var buf bytes.Buffer
	log.SetFlags(0)
	log.SetOutput(&buf)
	return &buf
}
