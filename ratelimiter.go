package ratelimiter

import (
	"fmt"
	"strconv"
	"time"

	redis "gopkg.in/redis.v5"

	"github.com/teambition/gear"
	baselimiter "github.com/teambition/ratelimiter-go"
)

//Options ...
type Options struct {
	RedisAddr string
	Prefix    string
	Max       int
	Duration  time.Duration
	GetID     func(ctx *gear.Context) string
	Policy    map[string][]int
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
				return key, nil
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
	ctx.Set("X-Ratelimit-Limit", strconv.FormatInt(int64(res.Total), 10))
	ctx.Set("X-Ratelimit-Remaining", strconv.FormatInt(int64(res.Remaining), 10))
	ctx.Set("X-Ratelimit-Reset", strconv.FormatInt(res.Reset.Unix(), 10))
	if res.Remaining < 0 {
		after := int64(res.Reset.Sub(time.Now())) / 1e9
		ctx.Set("Retry-After", strconv.FormatInt(after, 10))
		return ctx.End(429, []byte(fmt.Sprintf("Rate limit exceeded, retry in %d seconds.\n", after)))
	}
	return nil
}

//New ...
func New(opts *Options) (l *RateLimiter) {
	if opts.GetID == nil {
		panic("getId function required")
	}
	client := redis.NewClient(&redis.Options{
		Addr: opts.RedisAddr,
	})
	limiter, err := baselimiter.New(&DefaultRedisClient{client}, baselimiter.Options{
		Prefix:   opts.Prefix,
		Max:      opts.Max,
		Duration: opts.Duration, // limit to 1000 requests in 1 minute.
	})
	if err != nil {
		panic(err)
	}
	l = &RateLimiter{
		limiter: limiter,
		options: opts,
	}
	return
}
