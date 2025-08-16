package handlers

import (
	"sync"
	"time"
)

type bucket struct {
	allowance float64
	lastCheck time.Time
}

type IPLimiter struct {
	mu       sync.Mutex
	rate     float64 // tokens per second
	capacity float64 // max tokens
	buckets  map[string]*bucket
}

func NewIPLimiter(reqPerMinute int) *IPLimiter {
	return &IPLimiter{
		rate:     float64(reqPerMinute) / 60.0,
		capacity: float64(reqPerMinute),
		buckets:  make(map[string]*bucket),
	}
}

func (l *IPLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{allowance: l.capacity, lastCheck: time.Now()}
		l.buckets[key] = b
	}
	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.lastCheck = now
	b.allowance += elapsed * l.rate
	if b.allowance > l.capacity {
		b.allowance = l.capacity
	}
	if b.allowance < 1.0 {
		return false
	}
	b.allowance -= 1.0
	return true
}