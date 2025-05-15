package testutil

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureLogs returns a collector of logs emitted with the [log] package.
// The logger is reverted to its previous state upon completion of the test.
// As the logger is global, this function will fail if called from a parallel test.
// Note that t.Parallel will panic with a message like "t.Parallel called after t.Setenv"
// if called after this function.
func CaptureLogs(t *testing.T) fmt.Stringer {
	require.False(t, isParallel(t), "logs cannot be captured parallel tests")
	f, w := log.Flags(), log.Writer()
	t.Cleanup(func() {
		log.SetFlags(f)
		log.SetOutput(w)
	})
	var buf bytes.Buffer
	log.SetFlags(0)     // don't include timestamps
	log.SetOutput(&buf) // log to buffer
	return &buf
}

// isParallel returns whether [testing.T.Parallel] has already been called on t.
func isParallel(t *testing.T) (res bool) {
	defer func() {
		if recover() != nil {
			res = true
		}
	}()
	// Setenv panics for parallel tests.
	// Setting PATH is a no-op on all platforms.
	t.Setenv("PATH", os.Getenv("PATH"))
	return
}
