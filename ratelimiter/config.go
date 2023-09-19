package ratelimiter

import (
	"context"

	"github.com/maxliu9403/common/logger"
	"golang.org/x/time/rate"
)

type LimiterConfig struct {
	RateLimit      rate.Limit `yaml:"rate_limit" env:"RateLimit" env-description:"令牌桶的产生Token的速率，每秒10个"`
	RateLimitBurst int        `yaml:"rate_limit_burst" env:"RateLimitBurst" env-description:"令牌桶的容量大小"`
}

func (c *LimiterConfig) initConfig() *LimiterConfig {
	if c.RateLimit == 0 {
		c.RateLimit = 200
	}
	if c.RateLimitBurst == 0 {
		c.RateLimitBurst = 20
	}
	return c
}

func (c *LimiterConfig) BuildRateLimiter(ctx context.Context) {
	logger.Debug("build rate limiter")

	if _rateLimiter != nil {
		return
	}

	_rateLimiter = &RateLimiter{ctx: ctx}
	_rateLimiter.rateLimiter = rate.NewLimiter(c.initConfig().RateLimit, c.initConfig().RateLimitBurst)
}
