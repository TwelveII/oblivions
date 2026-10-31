package ratelimit

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const fileRateIdleTTL = 10 * time.Minute

type fileLimitKey struct {
	kind string
	id   string
}

type fileBucket struct {
	limiter     *rate.Limiter
	lastSeen    time.Time
	denied      int
	lastWarnLog time.Time
}

type FileRateLimiter struct {
	mu          sync.Mutex
	buckets     map[fileLimitKey]*fileBucket
	userPerMin  int
	ipPerMin    int
	userBurst   int
	ipBurst     int
	stopCleanup chan struct{}
	stopOnce    sync.Once
}

func NewFileRateLimiter(userPerMin, ipPerMin int) *FileRateLimiter {
	if userPerMin <= 0 {
		userPerMin = 600
	}
	if ipPerMin <= 0 {
		ipPerMin = 2000
	}

	rl := &FileRateLimiter{
		buckets:     make(map[fileLimitKey]*fileBucket),
		userPerMin:  userPerMin,
		ipPerMin:    ipPerMin,
		userBurst:   max(1, userPerMin/6),
		ipBurst:     max(1, ipPerMin/10),
		stopCleanup: make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *FileRateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stopCleanup)
	})
}

func (rl *FileRateLimiter) Allow(userID, ip string) (bool, int) {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if userID == "" && ip == "" {
		log.Printf("[ratelimit] file endpoint limit hit kind=unknown id=empty denied=1")
		return false, 60
	}

	var reservations []*rate.Reservation
	var deniedBucket *fileBucket
	var deniedKey fileLimitKey
	retryAfter := 0

	if userID != "" {
		key := fileLimitKey{kind: "user", id: userID}
		b := rl.bucketLocked(key, rl.userPerMin, rl.userBurst, now)
		res := b.limiter.ReserveN(now, 1)
		if !reservationAllowed(res, now) {
			retryAfter = reservationRetryAfter(res, now)
			res.CancelAt(now) // release the future token, else denied floods starve refill
			deniedBucket = b
			deniedKey = key
		} else {
			reservations = append(reservations, res)
		}
	}

	if deniedBucket == nil && ip != "" {
		key := fileLimitKey{kind: "ip", id: ip}
		b := rl.bucketLocked(key, rl.ipPerMin, rl.ipBurst, now)
		res := b.limiter.ReserveN(now, 1)
		if !reservationAllowed(res, now) {
			retryAfter = reservationRetryAfter(res, now)
			res.CancelAt(now) // release the future token, else denied floods starve refill
			deniedBucket = b
			deniedKey = key
		} else {
			reservations = append(reservations, res)
		}
	}

	if deniedBucket != nil {
		for _, res := range reservations {
			res.CancelAt(now)
		}
		rl.recordDeniedLocked(deniedBucket, deniedKey, now)
		return false, retryAfter
	}

	return true, 0
}

func (rl *FileRateLimiter) HandleLimit(w http.ResponseWriter, retryAfter int) {
	if retryAfter < 1 {
		retryAfter = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
}

func (rl *FileRateLimiter) bucketLocked(key fileLimitKey, perMin, burst int, now time.Time) *fileBucket {
	b := rl.buckets[key]
	if b == nil {
		b = &fileBucket{
			limiter:  rate.NewLimiter(rate.Limit(float64(perMin)/60.0), burst),
			lastSeen: now,
		}
		rl.buckets[key] = b
	}

	b.lastSeen = now
	return b
}

func reservationAllowed(res *rate.Reservation, now time.Time) bool {
	return res.OK() && res.DelayFrom(now) <= 0
}

func reservationRetryAfter(res *rate.Reservation, now time.Time) int {
	if !res.OK() {
		return 60
	}
	return int(math.Ceil(res.DelayFrom(now).Seconds()))
}

func (rl *FileRateLimiter) recordDeniedLocked(b *fileBucket, key fileLimitKey, now time.Time) {
	b.denied++
	if b.denied == 1 || now.Sub(b.lastWarnLog) >= 30*time.Second {
		log.Printf("[ratelimit] file endpoint limit hit kind=%s id=%s denied=%d", key.kind, key.id, b.denied)
		b.lastWarnLog = now
		b.denied = 0
	}
}

func (rl *FileRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

func (rl *FileRateLimiter) cleanup() {
	cutoff := time.Now().Add(-fileRateIdleTTL)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for key, b := range rl.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
}
