package ratelimit

import (
	"testing"
	"time"
)

func TestFileRateLimiterAllowsUserBurst(t *testing.T) {
	rl := NewFileRateLimiter(600, 2000)
	defer rl.Stop()

	for i := 0; i < 100; i++ {
		ok, retry := rl.Allow("u1", "203.0.113.10")
		if !ok {
			t.Fatalf("request %d rejected with retry=%d", i+1, retry)
		}
	}

	ok, retry := rl.Allow("u1", "203.0.113.10")
	if ok {
		t.Fatal("expected user burst to reject request 101")
	}
	if retry < 1 {
		t.Fatalf("retry after = %d, want >= 1", retry)
	}
}

func TestFileRateLimiterAppliesIPLimitWithoutUser(t *testing.T) {
	rl := NewFileRateLimiter(600, 60)
	defer rl.Stop()

	for i := 0; i < rl.ipBurst; i++ {
		ok, retry := rl.Allow("", "203.0.113.20")
		if !ok {
			t.Fatalf("ip request %d rejected with retry=%d", i+1, retry)
		}
	}

	ok, retry := rl.Allow("", "203.0.113.20")
	if ok {
		t.Fatal("expected ip burst to reject next request")
	}
	if retry < 1 {
		t.Fatalf("retry after = %d, want >= 1", retry)
	}
}

func TestFileRateLimiterDoesNotConsumeUserTokenWhenIPRejects(t *testing.T) {
	rl := NewFileRateLimiter(6, 6)
	defer rl.Stop()

	userID := "u1"
	for i := 0; i < rl.ipBurst; i++ {
		ok, retry := rl.Allow("other", "203.0.113.40")
		if !ok {
			t.Fatalf("priming ip request %d rejected with retry=%d", i+1, retry)
		}
	}

	ok, retry := rl.Allow(userID, "203.0.113.40")
	if ok {
		t.Fatal("expected shared IP bucket to reject")
	}
	if retry < 1 {
		t.Fatalf("retry after = %d, want >= 1", retry)
	}

	ok, retry = rl.Allow(userID, "203.0.113.41")
	if !ok {
		t.Fatalf("user bucket was consumed by rejected IP request; retry=%d", retry)
	}
}

func TestFileRateLimiterRefillsAfterFlood(t *testing.T) {
	// High IP limit so the user bucket is the only constraint.
	rl := NewFileRateLimiter(600, 1000000)
	defer rl.Stop()

	for i := 0; i < rl.userBurst; i++ {
		if ok, _ := rl.Allow("u1", "203.0.113.50"); !ok {
			t.Fatalf("burst request %d rejected", i+1)
		}
	}

	// Flood of denied requests must not consume future refill tokens.
	for i := 0; i < 500; i++ {
		rl.Allow("u1", "203.0.113.50")
	}

	// At 600/min the bucket refills 10 tokens/sec; 300ms must free at least one.
	time.Sleep(300 * time.Millisecond)
	if ok, retry := rl.Allow("u1", "203.0.113.50"); !ok {
		t.Fatalf("expected refill to allow request after flood; retry=%d", retry)
	}
}

func TestFileRateLimiterRejectsMissingIdentity(t *testing.T) {
	rl := NewFileRateLimiter(600, 2000)
	defer rl.Stop()

	ok, retry := rl.Allow("", "")
	if ok {
		t.Fatal("expected missing user and ip to reject")
	}
	if retry < 1 {
		t.Fatalf("retry after = %d, want >= 1", retry)
	}
}

func TestFileRateLimiterScalesUserBurstWithConfiguredRate(t *testing.T) {
	rl := NewFileRateLimiter(12, 2000)
	defer rl.Stop()

	if rl.userBurst != 2 {
		t.Fatalf("userBurst = %d, want 2", rl.userBurst)
	}
}

func TestFileRateLimiterCleanupEvictsIdleBuckets(t *testing.T) {
	rl := NewFileRateLimiter(600, 2000)
	defer rl.Stop()
	if ok, _ := rl.Allow("u1", "203.0.113.30"); !ok {
		t.Fatal("initial request rejected")
	}

	rl.mu.Lock()
	for _, b := range rl.buckets {
		b.lastSeen = time.Now().Add(-fileRateIdleTTL - time.Second)
	}
	rl.mu.Unlock()

	rl.cleanup()

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if len(rl.buckets) != 0 {
		t.Fatalf("bucket count = %d, want 0", len(rl.buckets))
	}
}
