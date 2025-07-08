package testutil

import "pgregory.net/rapid"

// T is an abstraction over [testing.T] that's compatible with [rapid.T].
type T interface {
	rapid.TB
	Cleanup(func())
}
