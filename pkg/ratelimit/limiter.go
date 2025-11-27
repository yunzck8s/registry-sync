package ratelimit

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// Limiter provides rate limiting functionality
type Limiter struct {
	limiter *rate.Limiter
	enabled bool
}

// NewLimiter creates a new rate limiter
// qps: queries per second, 0 means no limit
func NewLimiter(qps int) *Limiter {
	if qps <= 0 {
		return &Limiter{enabled: false}
	}

	return &Limiter{
		limiter: rate.NewLimiter(rate.Limit(qps), qps),
		enabled: true,
	}
}

// Wait blocks until the limiter permits an event
func (l *Limiter) Wait(ctx context.Context) error {
	if !l.enabled {
		return nil
	}
	return l.limiter.Wait(ctx)
}

// Allow reports whether an event may happen now
func (l *Limiter) Allow() bool {
	if !l.enabled {
		return true
	}
	return l.limiter.Allow()
}

// Reserve returns a Reservation that indicates how long the caller must wait
func (l *Limiter) Reserve() *rate.Reservation {
	if !l.enabled {
		return nil
	}
	return l.limiter.Reserve()
}

// SetQPS updates the rate limit
func (l *Limiter) SetQPS(qps int) {
	if qps <= 0 {
		l.enabled = false
		return
	}

	if l.limiter == nil {
		l.limiter = rate.NewLimiter(rate.Limit(qps), qps)
	} else {
		l.limiter.SetLimit(rate.Limit(qps))
		l.limiter.SetBurst(qps)
	}
	l.enabled = true
}

// GetQPS returns the current QPS limit
func (l *Limiter) GetQPS() int {
	if !l.enabled {
		return 0
	}
	return int(l.limiter.Limit())
}

// MultiLimiter combines multiple rate limiters
type MultiLimiter struct {
	limiters []*Limiter
}

// NewMultiLimiter creates a new multi-limiter
func NewMultiLimiter(limiters ...*Limiter) *MultiLimiter {
	return &MultiLimiter{limiters: limiters}
}

// Wait blocks until all limiters permit an event
func (m *MultiLimiter) Wait(ctx context.Context) error {
	for _, limiter := range m.limiters {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

// WaitN blocks until all limiters permit N events
func (m *MultiLimiter) WaitN(ctx context.Context, n int) error {
	for _, limiter := range m.limiters {
		if limiter.enabled {
			if err := limiter.limiter.WaitN(ctx, n); err != nil {
				return err
			}
		}
	}
	return nil
}

// SleepLimiter provides simple sleep-based rate limiting
type SleepLimiter struct {
	interval time.Duration
	lastCall time.Time
}

// NewSleepLimiter creates a new sleep-based limiter
func NewSleepLimiter(qps int) *SleepLimiter {
	if qps <= 0 {
		return &SleepLimiter{}
	}
	return &SleepLimiter{
		interval: time.Second / time.Duration(qps),
	}
}

// Wait sleeps if necessary to maintain the rate limit
func (s *SleepLimiter) Wait(ctx context.Context) error {
	if s.interval == 0 {
		return nil
	}

	elapsed := time.Since(s.lastCall)
	if elapsed < s.interval {
		select {
		case <-time.After(s.interval - elapsed):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	s.lastCall = time.Now()
	return nil
}
