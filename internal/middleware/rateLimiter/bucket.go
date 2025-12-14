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

func NewBucket(capacity uint) *Bucket {
	return &Bucket{
		tokens:     float64(capacity),
		lastRefill: time.Now(),
	}
}

func (b *Bucket) CheckAndConsumeToken(refillRate float64, capacity uint) bool {
	b.mux.Lock()
	defer b.mux.Unlock()

	elapsed := time.Since(b.lastRefill)
	tokensToAdd := elapsed.Seconds() * refillRate
	b.tokens = min(tokensToAdd+b.tokens, float64(capacity))
	b.lastRefill = time.Now()

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}
