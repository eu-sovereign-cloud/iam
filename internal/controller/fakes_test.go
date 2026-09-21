package controller_test

import "time"

// fakeClock is a deterministic, fixed-time stub — unlike a store or
// signer double, there's no shared adapter for this (system.Clock always
// returns real wall time), so it stays local to these tests.
type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }
