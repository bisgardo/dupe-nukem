package main

import (
	"time"
)

func timeSince(start time.Time) time.Duration {
	return round(time.Since(start))
}

func round(t time.Duration) time.Duration {
	return t.Round(time.Millisecond)
}
