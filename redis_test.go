package ratelimiter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear-ratelimiter"
	redis "gopkg.in/redis.v5"
)

func TestRedisClient(t *testing.T) {

	t.Run("RedisClient init should be", func(t *testing.T) {
		assert := assert.New(t)
		client := redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		})
		c := &ratelimiter.DefaultRedisClient{client}
		err := c.RateSet("testset", "1")

		assert.Nil(err)
	})
}
