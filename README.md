# gear-ratelimiter
Smart rate limiter middleware for Gear.  
[![Build Status][travis-image]][travis-url]
##Requirements
 - [ratelimiter-go](https://github.com/teambition/ratelimiter-go)
 - Redis 3+ with  [gopkg.in/redis.v5](gopkg.in/redis.v5)

## Installation
    go get github.com/teambition/gear-ratelimiter

##API
    import "github.com/teambition/gear-ratelimiter"
### smartLimiter(Options)
	limiter := ratelimiter.New(&ratelimiter.Options{
		GetID: func(ctx *gear.Context) string {
			return "user-123465"
		},
		Max:      10,
		Duration: time.Minute, // limit to 1000 requests in 1 minute.
		Policy: map[string][]int{
			"/":      []int{16, 6 * 1000},
			"GET /a": []int{3, 5 * 1000, 10, 60 * 1000},
			"GET /b": []int{5, 60 * 1000},
			"/c":     []int{6, 60 * 1000},
		},
		RedisAddr: "127.0.0.1:6379",
	})
    app.Use(limiter.GetLimiter())
return a express gear middleware.

- `options.Max`: *Optional*, Type: `int`, The max count in duration and using it when limiter cannot found the appropriate policy, default to `100`.
- `options.Prefix`: *Optional*, Type: `String`, redis key namespace, default to `LIMIT`.
- `options.RedisAddr`: *Optional*, Redis address such as "127.0.0.1:6379"
- `options.Duration`: *Optional*, {Number}, of limit in milliseconds, default to `3600000`
- `options.GetID`: *Optional*, {Function}, generate a identifier for requests, default to user's IP
- `options.Policy`: *Required*, {map[string][]int}, limit policy

##Example
Try into github.com/teambition/gear-ratelimiter directory:  

    go run ratelimiter/main.go

## License
​MIT  

[travis-url]: https://travis-ci.org/teambition/gear-ratelimiter
[travis-image]: http://img.shields.io/travis/teambition/gear-ratelimiter.svg
