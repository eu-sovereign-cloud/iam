package ports

import "time"

// Clock is injected so services are deterministic in tests.
type Clock interface {
	Now() time.Time
}
