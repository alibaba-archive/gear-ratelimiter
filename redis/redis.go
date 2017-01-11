package redis

import (
	"time"

	baselimiter "github.com/teambition/ratelimiter-go"
	redis "gopkg.in/redis.v5"
)

// NewRedisClient returns a new RedisClient with redis cluster options.
func NewRedisClient(opts *redis.Options) baselimiter.RedisClient {
	client := redis.NewClient(opts)
	return &DefaultRedisClient{client}
}

// NewClusterClient returns a new RedisClient with redis cluster options.
func NewClusterClient(opts *redis.ClusterOptions) baselimiter.RedisClient {
	client := redis.NewClusterClient(opts)
	return &DefaultClusterClient{client}
}

// DefaultRedisClient implements RedisClient interface by default
type DefaultRedisClient struct {
	*redis.Client
}

// RateDel ...
func (c *DefaultRedisClient) RateDel(key string) error {
	return c.Del(key).Err()
}

// RateEvalSha ...
func (c *DefaultRedisClient) RateEvalSha(sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return c.EvalSha(sha1, keys, args...).Result()
}

// RateScriptLoad ...
func (c *DefaultRedisClient) RateScriptLoad(script string) (string, error) {
	return c.ScriptLoad(script).Result()
}

// RateSet ...
func (c *DefaultRedisClient) RateSet(key string, val string) error {
	return c.Set(key, val, time.Hour).Err()
}

// DefaultClusterClient implements Cluster RedisClient interface
type DefaultClusterClient struct {
	*redis.ClusterClient
}

// RateDel ...
func (c *DefaultClusterClient) RateDel(key string) error {
	return c.Del(key).Err()
}

// RateEvalSha ...
func (c *DefaultClusterClient) RateEvalSha(sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return c.EvalSha(sha1, keys, args...).Result()
}

// RateScriptLoad ...
func (c *DefaultClusterClient) RateScriptLoad(script string) (string, error) {
	var sha1 string
	err := c.ForEachMaster(func(client *redis.Client) error {
		res, err := client.ScriptLoad(script).Result()
		if err == nil {
			sha1 = res
		}
		return err
	})
	return sha1, err
}
