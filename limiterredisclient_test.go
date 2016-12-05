package ratelimiter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/teambition/gear-ratelimiter"
	redis "gopkg.in/redis.v5"
)

var _ = Describe("RedisClient", func() {
	It("RedisClient init should be", func() {

		client := redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		})
		c := &ratelimiter.DefaultRedisClient{client}
		err := c.RateSet("testset", "1")
		Expect(err).ToNot(HaveOccurred())

	})
})
