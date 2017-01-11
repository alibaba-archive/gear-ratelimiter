Gear-Ratelimiter
=====
Smart rate limiter middleware for Gear, base on redis or memory limiter.

[![Build Status](http://img.shields.io/travis/teambition/gear-ratelimiter.svg?style=flat-square)](https://travis-ci.org/teambition/gear-ratelimiter)
[![Coverage Status](http://img.shields.io/coveralls/teambition/gear-ratelimiter.svg?style=flat-square)](https://coveralls.io/r/teambition/gear-ratelimiter)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/teambition/gear-ratelimiter/master/LICENSE)
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/teambition/gear-ratelimiter)

## Installation

```bash
go get github.com/teambition/gear-ratelimiter
```

```bash
import "github.com/teambition/gear-ratelimiter"
```

## Demo

```go
import (
  "github.com/teambition/gear-ratelimiter"
  redisClient "github.com/teambition/gear-ratelimiter/redis"
	redis "gopkg.in/redis.v5"
)

limiter := ratelimiter.New(&ratelimiter.Options{
  Client: redisClient.NewRedisClient(&redis.Options{Addr: "127.0.0.1:6379"})
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
})
app.UseHandler(limiter)
```

## API

### ratelimiter.New(ratelimiter.Options)

returns a Gear middleware handler.

- `options.Client`: *Optional*, a wrapped redis client. if omit, it will use memory limiter.
- `options.Max`: *Optional*, Type: `int`, The max count in duration and using it when limiter cannot found the appropriate policy, default to `100`.
- `options.Prefix`: *Optional*, Type: `String`, redis key namespace, default to `LIMIT`.
- `options.Duration`: *Optional*, {Number}, of limit in milliseconds, default to `3600000`
- `options.GetID`: *Optional*, {Function}, generate a identifier for requests, default to user's IP
- `options.Policy`: *Required*, {map[string][]int}, limit policy

## Example

Try into github.com/teambition/gear-ratelimiter directory:

```bash
go run example/main.go
```

## License
Gear-Ratelimiter is licensed under the [MIT](https://github.com/teambition/gear-ratelimiter/blob/master/LICENSE) license.
Copyright &copy; 2016 [Teambition](https://www.teambition.com).
