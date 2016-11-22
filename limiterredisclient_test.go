package smartlimiter_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/teambition/gear-ratelimiter"
	redis "gopkg.in/redis.v5"
)

// init Test
func TestLimiterRedisClientGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RatelimiterGo Suite")
}

var _ = Describe("RedisClient", func() {
	It("RedisClient init should be", func() {

		client := redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		})
		c := &smartlimiter.DefaultRedisClient{client}
		err := c.RateSet("testset", "1")
		Expect(err).ToNot(HaveOccurred())

	})
})
