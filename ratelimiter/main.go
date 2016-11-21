package main

import (
	"os"
	"time"

	redis "gopkg.in/redis.v5"

	"github.com/teambition/gear"
	"github.com/teambition/gear-ratelimiter"
	"github.com/teambition/gear/middleware"
)

var limiter *smartlimiter.Limiter

func init() {

	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	redis := smartlimiter.DRedisClient{client}
	var err error
	limiter, err = smartlimiter.New(smartlimiter.Options{
		Redis:    &redis,
		Max:      10,
		Duration: time.Minute, // limit to 1000 requests in 1 minute.
		Policy: map[string][]int{
			"GET /a": []int{3, 5000},
			"GET /b": []int{5, 100000},
		},
	})
	if err != nil {
		panic(err)
	}
}
func main() {
	app := gear.New()
	// Add app middleware
	logger := &middleware.DefaultLogger{W: os.Stdout}
	app.Use(middleware.NewLogger(logger))
	app.Use(smartlimiter.NewLimiter(limiter))

	// Add router middleware
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
	app.UseHandler(router)
	app.Error(app.Listen(":3000"))
}
