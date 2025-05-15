package testutil

import "bytes"

// Lines concatenates the provided strings after adding a newline after each of them.
func Lines(s ...string) string {
	var buf bytes.Buffer
	for _, l := range s {
		buf.WriteString(l)
		buf.WriteByte('\n')
	}
	return buf.String()
}
