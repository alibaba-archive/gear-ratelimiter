package smartlimiter_test

import (
	"encoding/hex"
	"math/rand"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/teambition/gear-ratelimiter"
	redis "gopkg.in/redis.v5"
)

// init Test
func TestRatelimiterGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RatelimiterGo Suite")
}

var client *redis.Client
var limiter *smartlimiter.Limiter

var _ = BeforeSuite(func() {
	client = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	pong, err := client.Ping().Result()
	Expect(pong).To(Equal("PONG"))
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := client.Close()
	Expect(err).ShouldNot(HaveOccurred())
})
var _ = Describe("smartlimiter", func() {
	Describe("smartlimiter.New, With default options", func() {
		var limiter *smartlimiter.Limiter
		var id string = genID()
		It("ratelimiter.New should be", func() {
			res, err := smartlimiter.New(&smartlimiter.DRedisClient{client}, smartlimiter.Options{})
			Expect(err).ToNot(HaveOccurred())
			limiter = res
		})
		It("limiter.Get should be", func() {
			res, err := limiter.Get(id)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Total).To(Equal(100))
			Expect(res.Remaining).To(Equal(99))
			Expect(res.Duration).To(Equal(time.Duration(60 * 1e9)))
			Expect(res.Reset.UnixNano() > time.Now().UnixNano()).To(Equal(true))
		})
	})
})

func genID() string {
	buf := make([]byte, 12)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
