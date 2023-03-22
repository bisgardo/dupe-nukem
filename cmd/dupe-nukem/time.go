package main

import (
	"time"
)

func timeSince(start time.Time) time.Duration {
	return round(time.Since(start))
}

func timeBetween(start time.Time, end time.Time) time.Duration {
	return round(end.Sub(start))
}

func round(t time.Duration) time.Duration {
	return t.Round(time.Millisecond)
}
