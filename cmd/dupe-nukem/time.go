package main

import (
	"time"
)

func timeSince(start time.Time) time.Duration {
	return time.Since(start).Round(time.Millisecond)
}
