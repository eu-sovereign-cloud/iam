// Package system implements the production ports.Clock, backed by
// time.Now — one of possibly several driven adapters under
// internal/adapter.
package system

import "time"

// Clock is the production ports.Clock backed by time.Now.
type Clock struct{}

func (Clock) Now() time.Time { return time.Now() }
