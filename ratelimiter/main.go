package main

import (
	"time"

	"github.com/teambition/gear"
	"github.com/teambition/gear-ratelimiter"
	"github.com/teambition/gear/logging"
)

func main() {
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
	app := gear.New()
	// Use a default logger middleware
	app.UseHandler(logging.Default())
	// Add rate limiter middleware
	app.UseHandler(limiter)

	router := gear.NewRouter()
	router.Get("/", func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
	})
	router.Get("/a", func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear! /a</h1>")
	})
	router.Get("/b", func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear! /b</h1>")
	})
	router.Get("/c", func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear! /c</h1>")
	})
	app.UseHandler(router)
	app.Error(app.Listen(":3000"))
}
