package ratelimiter

import (
	"context"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	rateLimiter *rate.Limiter
	ctx         context.Context
}

var (
	_rateLimiter *RateLimiter
)

func GetRateLimiter() *RateLimiter {
	if _rateLimiter == nil {
		return &RateLimiter{}
	}

	return _rateLimiter
}

func (rl *RateLimiter) Limiter() *rate.Limiter {
	if rl == nil {
		return nil
	}
	return rl.rateLimiter
}

func (rl *RateLimiter) SetRateLimit(limit int64) {
	if rl == nil {
		return
	}
	rl.rateLimiter.SetLimit(rate.Limit(limit))
}

func (rl *RateLimiter) SetRateLimitBurst(burst int) {
	if rl == nil {
		return
	}
	rl.rateLimiter.SetBurst(burst)
}
