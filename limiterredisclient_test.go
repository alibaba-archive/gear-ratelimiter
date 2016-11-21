package smartlimiter_test

import (
	"testing"

	"github.com/teambition/gear-ratelimiter"
	redis "gopkg.in/redis.v5"
)

func Test_RedisSet(t *testing.T) {

	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	c := smartlimiter.DRedisClient{client}
	err := c.RateSet("testset", "1")
	if err != nil {
	}
}
