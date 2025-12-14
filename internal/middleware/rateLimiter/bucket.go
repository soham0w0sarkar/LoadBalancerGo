package ratelimiter

import (
	"sync"
	"time"
)

type Bucket struct {
	tokens     float64
	lastRefill time.Time
	mux        sync.RWMutex
}

func (b *Bucket) CheckAndConsumeToken(refillRate float64, capacity int) bool {
	b.mux.Lock()
	defer b.mux.Unlock()

	elapsed := time.Since(b.lastRefill)
	tokensToAdd := elapsed.Seconds() * refillRate
	b.tokens = min(tokensToAdd+b.tokens, float64(capacity))

	if b.tokens > 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}
