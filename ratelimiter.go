package ratelimiter

import (
	"strconv"
	"time"

	"github.com/teambition/gear"
	baselimiter "github.com/teambition/ratelimiter-go"
)

// Version is ratelimiter's version
const Version = "1.0.1"

// Options for Limiter
type Options struct {
	// Key prefix, default is "LIMIT:".
	Prefix string
	// The max count in duration for no policy, default is 100.
	Max int
	// Count duration for no policy, default is 1 Minute.
	Duration time.Duration
	// Policy is a map of custom limiter policy.
	Policy map[string][]int
	// GetID returns limiter id for a request.
	GetID func(ctx *gear.Context) string
	// Use a redis client for limiter, if omit, it will use a memory limiter.
	Client baselimiter.RedisClient
}

//RateLimiter ...
type RateLimiter struct {
	options *Options
	limiter *baselimiter.Limiter
}

func (l *RateLimiter) getArgs(ctx *gear.Context) (key string, p []int) {
	id := l.options.GetID(ctx)
	if id == "" {
		return
	}
	key = ctx.Method + " " + ctx.Path
	var ok bool
	p, ok = l.options.Policy[key]
	if !ok {
		key = ctx.Path
		if p, ok = l.options.Policy[key]; !ok {
			key = ctx.Method
			if p, ok = l.options.Policy[key]; !ok {
				p = []int{} // It will use Options.Max and Options.Duration if no policy
			}
		}
	}
	key = id + key
	return
}

//Serve ...
func (l *RateLimiter) Serve(ctx *gear.Context) error {
	key, p := l.getArgs(ctx)
	if key == "" {
		return nil
	}
	if len(p) < 1 {
		return nil
	}
	res, err := l.limiter.Get(key, p...)
	if err != nil {
		return nil
	}
	ctx.Set("X-Ratelimit-Limit", strconv.Itoa(res.Total))
	ctx.Set("X-Ratelimit-Remaining", strconv.Itoa(res.Remaining))
	ctx.Set("X-Ratelimit-Reset", strconv.Itoa(int(res.Reset.Unix())))
	if res.Remaining < 0 {
		after := int(res.Reset.Sub(time.Now()).Seconds())
		ctx.Set("Retry-After", strconv.Itoa(after))
		return gear.ErrTooManyRequests.WithMsgf("Rate limit exceeded, retry in %d seconds.", after)
	}
	return nil
}

//New ...
func New(opts *Options) (l *RateLimiter) {
	if opts.GetID == nil {
		panic("getId function required")
	}

	limiter := baselimiter.New(baselimiter.Options{
		Prefix:   opts.Prefix,
		Max:      opts.Max,
		Duration: opts.Duration,
		Client:   opts.Client,
	})
	return &RateLimiter{options: opts, limiter: limiter}
}
