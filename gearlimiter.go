package smartlimiter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/teambition/gear"
)

//NewLimiter ...
func NewLimiter(limiter *Limiter) gear.Middleware {
	return func(ctx *gear.Context) error {
		o := limiter.opts
		key := ctx.Method + " " + ctx.Path
		p, ok := o.Policy[key]
		if !ok {
			key = ctx.Path
			if p, ok = o.Policy[key]; !ok {
				key = ctx.Method
				if p, ok = o.Policy[key]; !ok {
					return nil
				}
			}
		}
		res, err := limiter.Get(ctx.IP().String(), p...)
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
