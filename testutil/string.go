package testutil

import "bytes"

func Lines(s ...string) string {
	var buf bytes.Buffer
	for _, l := range s {
		buf.WriteString(l)
		buf.WriteByte('\n')
	}
	return buf.String()
}
