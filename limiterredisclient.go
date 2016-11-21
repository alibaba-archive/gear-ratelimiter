package smartlimiter

import (
	"time"

	redis "gopkg.in/redis.v5"
)

/*
DRedisClient for implement RedisClient interface by default
*/
type DRedisClient struct {
	*redis.Client
}

//RateDel ...
func (c *DRedisClient) RateDel(key string) error {
	return c.Del(key).Err()
}

//RateEvalSha ...
func (c *DRedisClient) RateEvalSha(sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return c.EvalSha(sha1, keys, args...).Result()
}

//RateScriptLoad ...
func (c *DRedisClient) RateScriptLoad(script string) (string, error) {
	return c.ScriptLoad(script).Result()
}

//RateSet ...
func (c *DRedisClient) RateSet(key string, val string) error {
	return c.Set(key, val, time.Hour).Err()
}

//DClusterClient for implement Cluster RedisClient interface by default
type DClusterClient struct {
	*redis.Client
}
