package ratelimit

import (
    "testing"
    "time"
)

func TestIPLimiter_Allow(t *testing.T) {
    l := NewIPLimiter(60) // 60 per minute => 1 per second, capacity 60
    key := "1.2.3.4"

    // Burst should be allowed up to capacity immediately
    for i := 0; i < 10; i++ {
        if !l.Allow(key) {
            t.Fatalf("unexpected deny at i=%d", i)
        }
    }

    // If we burn allowance down to below 1, subsequent immediate call should deny
    for i := 0; i < 60; i++ { l.Allow(key) }
    if l.Allow(key) { t.Fatalf("expected deny when allowance < 1") }

    // Wait to replenish ~1 token
    time.Sleep(1100 * time.Millisecond)
    if !l.Allow(key) { t.Fatalf("expected allow after replenish") }

    // Different key should have independent bucket
    if !l.Allow("another") { t.Fatalf("expected allow for another key") }
}


