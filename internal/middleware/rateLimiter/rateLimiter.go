package ratelimiter

import (
	"fmt"
	"net/http"
	"sync"
)

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type RateLimiter struct {
	BucketList map[string]*Bucket
	capacity   uint
	refillRate float64
	next       Handler
	mux        sync.RWMutex
}

func NewRateLimiter(capacity uint, refillRate float64, next Handler) *RateLimiter {
	return &RateLimiter{
		BucketList: make(map[string]*Bucket),
		capacity:   capacity,
		refillRate: refillRate,
		next:       next,
	}
}

func (rl *RateLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIp := r.Header.Get("x-api-key")

	fmt.Printf("Hey there: %s\n", clientIp)

	clientBucket := rl.BucketList[clientIp]

	if clientBucket != nil {
		if !clientBucket.CheckAndConsumeToken(rl.refillRate, rl.capacity) {
			http.Error(w, "Rate Limited this IP", http.StatusTooManyRequests)
			return
		}
	} else {
		bucketToAdd := NewBucket(rl.capacity - 1)
		rl.addBucket(bucketToAdd, clientIp)
	}
	rl.next.ServeHTTP(w, r)
}

func (rl *RateLimiter) addBucket(bucket *Bucket, clientIp string) {
	rl.mux.Lock()

	rl.BucketList[clientIp] = bucket

	rl.mux.Unlock()
}
