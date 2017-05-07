package oauth

import (
	"log"
	"sync"
	"time"
)

type Bucket struct {
	tokens int
}

type RateLimiter struct {
	refillRate int
	bucket     *Bucket
	lastTaken  *time.Time
	lock       *sync.Mutex
}

type RateLimitResult struct {
	Limited   bool
	Remaining int
	Limit     int
}

func (r *RateLimiter) Take(n int) *RateLimitResult {
	r.lock.Lock()
	defer r.lock.Unlock()
	lastTaken := *r.lastTaken
	result := &RateLimitResult{
		Limit: r.refillRate,
	}

	now := time.Now()
	diff := now.Sub(lastTaken)
	// If it has been more than a second if must be at limit
	tokensToAdd := int64(r.refillRate)
	if diff.Seconds() < 1 {
		tokensToAdd = int64(r.refillRate) * diff.Nanoseconds() / time.Second.Nanoseconds()
	}

	tokens := r.refillRate
	// Safe case, should never be greater than 32 bits otherwise
	// you'd have a very large bucket!
	if r.refillRate > r.bucket.tokens+int(tokensToAdd) {
		tokens = r.bucket.tokens + int(tokensToAdd)
	}

	r.bucket.tokens = tokens

	if tokens < n {
		result.Limited = true
	} else {
		result.Limited = false
		r.lastTaken = &now
		r.bucket.tokens = r.bucket.tokens - n
		result.Remaining = r.bucket.tokens
	}

	return result
}

// RateLimiterPool manages a set of rate limiters for use with OAuth clients.
// Rate limites are provided by the ClientService.
type RateLimiterPool struct {
	rateLimiter   map[string]*RateLimiter
	clientService ClientService
}

// MakeRateLimiter constructs a new RateLimiterPool using a given ClientServce.
func MakeRateLimiter(clientService ClientService) *RateLimiterPool {
	return &RateLimiterPool{
		rateLimiter:   map[string]*RateLimiter{},
		clientService: clientService,
	}
}

// GetRateLimiter gets an existing ratelimiter or creates a new one and
// returns it.
func (l *RateLimiterPool) GetRateLimiter(clientID string) *RateLimiter {
	rateLimiter := l.rateLimiter[clientID]
	if rateLimiter == nil {
		rateLimiter = l.makeRateLimiter(clientID)
	}
	return rateLimiter
}

func (l *RateLimiterPool) makeRateLimiter(clientID string) *RateLimiter {
	client, err := l.clientService.GetClient(clientID)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	rateLimiter := &RateLimiter{
		refillRate: client.RateLimitPerSecond,
		lock:       &sync.Mutex{},
		lastTaken:  &now,
		bucket: &Bucket{
			tokens: client.RateLimitPerSecond,
		},
	}

	l.rateLimiter[clientID] = rateLimiter
	return rateLimiter
}
