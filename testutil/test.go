package testutil

import "pgregory.net/rapid"

type T interface {
	rapid.TB
	Cleanup(func())
}
