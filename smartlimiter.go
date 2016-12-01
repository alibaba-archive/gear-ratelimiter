package smartlimiter

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	redis "gopkg.in/redis.v5"

	"github.com/teambition/gear"
	ratelimiter "github.com/teambition/ratelimiter-go"
)

//Options ...
type Options struct {
	RedisAddr string
	Prefix    string
	Max       int
	Duration  time.Duration
	GetID     func(req *http.Request) string
	Policy    map[string][]int
}

var limiter *ratelimiter.Limiter
var options *Options

func initLimiter(opts *Options) {
	options = opts
	var err error
	client := redis.NewClient(&redis.Options{
		Addr: options.RedisAddr,
	})
	limiter, err = ratelimiter.New(&DefaultRedisClient{client}, ratelimiter.Options{
		Prefix:   options.Prefix,
		Max:      options.Max,
		Duration: options.Duration, // limit to 1000 requests in 1 minute.
	})
	if err != nil {
		panic(err)
	}
}

func getArgs(ctx *gear.Context) (string, []int) {
	key := ctx.Method + " " + ctx.Path
	p, ok := options.Policy[key]
	if !ok {
		key = ctx.Path
		if p, ok = options.Policy[key]; !ok {
			key = ctx.Method
			if p, ok = options.Policy[key]; !ok {
				return key, make([]int, 0, 0)
			}
		}
	}
	if options.GetID != nil {
		key = options.GetID(ctx.Req) + key
	} else {
		key = ctx.IP().String() + key
	}
	return key, p
}

//NewLimiter ...
func NewLimiter(opts *Options) gear.Middleware {
	initLimiter(opts)
	return func(ctx *gear.Context) error {
		key, p := getArgs(ctx)
		if len(p) < 1 {
			return nil
		}
		res, err := limiter.Get(key, p...)
		if err != nil {
			return nil
		}
		header := ctx.Res.Header()
		header.Set("X-Ratelimit-Limit", strconv.FormatInt(int64(res.Total), 10))
		header.Set("X-Ratelimit-Remaining", strconv.FormatInt(int64(res.Remaining), 10))
		header.Set("X-Ratelimit-Reset", strconv.FormatInt(res.Reset.Unix(), 10))
		if res.Remaining < 0 {
			after := int64(res.Reset.Sub(time.Now())) / 1e9
			header.Set("Retry-After", strconv.FormatInt(after, 10))
			ctx.Res.Body = []byte(fmt.Sprintf("Rate limit exceeded, retry in %d seconds.\n", after))
			ctx.End(429)
		}
		return nil
	}
}
