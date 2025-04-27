package backoff

import "time"

// Backoff implements a simple exponential backoff strategy that caps the
// calculated delay at a configured maximum. It is intentionally free of
// external dependencies so it can be reused across packages.
type Backoff struct {
	base    time.Duration // starting delay
	max     time.Duration // maximum delay cap
	attempt int           // current attempt counter
}

// New creates a new backoff helper with base and max durations.
func New(base, max time.Duration) *Backoff {
	if base <= 0 {
		base = time.Second
	}
	if max < base {
		max = base
	}
	return &Backoff{
		base: base,
		max:  max,
	}
}

// Next returns the delay for the current attempt and increments the internal
// counter so that each subsequent call produces an exponentially longer delay
// until the configured maximum is reached.
func (b *Backoff) Next() time.Duration {
	// Calculate delay: base * 2^attempt.
	delay := b.base << uint(b.attempt) // shift multiplies by powers of two.
	if delay > b.max {
		delay = b.max
	} else {
		b.attempt++
	}
	return delay
}

// Reset sets the attempt counter back to zero so that the next call to Next
// returns the base delay again. This should be called after a successful
// operation to restart the back-off sequence.
func (b *Backoff) Reset() {
	b.attempt = 0
}
