package adapter

import "time"

// SystemClock is the production ports.Clock backed by time.Now.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
