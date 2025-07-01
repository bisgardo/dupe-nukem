package main

import (
	"time"
)

func timeSince(start time.Time) time.Duration {
	return timeRounded(time.Since(start))
}

func timeRounded(t time.Duration) time.Duration {
	return t.Round(time.Millisecond)
}
