package smartlimiter

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
)

//RedisClient ...
type RedisClient interface {
	RateDel(string) error
	RateEvalSha(string, []string, ...interface{}) (interface{}, error)
	RateScriptLoad(string) (string, error)
}

//Options ...
type Options struct {
	Redis    RedisClient
	Prefix   string
	Max      int
	Duration time.Duration
	GetID    func() string
	Policy   map[string][]int
}

//Limiter ...
type Limiter struct {
	sha1, prefix, duration, max string
	rc                          RedisClient
	opts                        Options
}

//Result ...
type Result struct {
	Total     int
	Remaining int
	Duration  time.Duration
	Reset     time.Time
}

//Get ...
func (l *Limiter) Get(id string, policy ...int) (Result, error) {
	var result Result

	length := len(policy)

	if odd := length % 2; odd == 1 {
		return result, errors.New("ratelimiter: must be paired values")
	}
	capacity := 3
	if length > 2 {
		capacity = length + 1
	}
	args := make([]interface{}, capacity, capacity)
	args[0] = genTimestamp()
	if length == 0 {
		args[1] = l.max
		args[2] = l.duration
	} else {
		for i, val := range policy {
			if val <= 0 {
				return result, errors.New("ratelimiter: must be positive integer")
			}
			args[i+1] = strconv.FormatInt(int64(val), 10)
		}
	}
	return l.buildResult(id, args)
}

// Remove remove limiter record for id
func (l *Limiter) Remove(id string) error {
	return l.rc.RateDel(l.prefix + id)
}

func (l *Limiter) buildResult(id string, args []interface{}) (Result, error) {

	var result Result
	keys := []string{l.prefix + id}

	res, err := l.getLimit(keys[0:1], args...)
	if err != nil {
		return result, err
	}
	arr := reflect.ValueOf(res)
	timestamp := arr.Index(3).Interface().(int64)
	sec := timestamp / 1000
	nsec := (timestamp - (sec * 1000)) * 1e6
	result = Result{
		Remaining: int(arr.Index(0).Interface().(int64)),
		Total:     int(arr.Index(1).Interface().(int64)),
		Duration:  time.Duration(arr.Index(2).Interface().(int64) * 1e6),
		Reset:     time.Unix(sec, nsec),
	}
	return result, err
}

func (l *Limiter) getLimit(keys []string, args ...interface{}) (res interface{}, err error) {
	res, err = l.rc.RateEvalSha(l.sha1, keys, args...)
	if err != nil && isNoScriptErr(err) {
		_, err = l.rc.RateScriptLoad(lua)
		if err == nil {
			res, err = l.rc.RateEvalSha(l.sha1, keys, args...)
		}
	}
	return
}

//New ...
func New(o Options) (*Limiter, error) {

	r := o.Redis

	var limiter *Limiter

	sha1, err := r.RateScriptLoad(lua)
	if err != nil {
		return limiter, err
	}

	prefix := o.Prefix
	if prefix == "" {
		prefix = "LIMIT:"
	}

	max := "100"
	if o.Max > 0 {
		max = strconv.FormatInt(int64(o.Max), 10)
	}

	duration := "60000"
	if o.Duration > 0 {
		duration = strconv.FormatInt(int64(o.Duration/time.Millisecond), 10)
	}

	limiter = &Limiter{rc: r, sha1: sha1, prefix: prefix, max: max, duration: duration}
	limiter.opts = o
	return limiter, nil
}
func genTimestamp() string {
	time := time.Now().UnixNano() / 1e6
	return strconv.FormatInt(time, 10)
}
func isNoScriptErr(err error) bool {
	var no bool
	s := err.Error()
	if strings.HasPrefix(s, "NOSCRIPT ") {
		no = true
	}
	return no
}

// copy from ./ratelimiter.lua
const lua string = `
local res = {}
local policyCount = (#ARGV - 1) / 2
local statusKey = '{' .. KEYS[1] .. '}:S'
local limit = redis.call('hmget', KEYS[1], 'ct', 'lt', 'dn', 'rt')

if limit[1] then

  res[1] = tonumber(limit[1]) - 1
  res[2] = tonumber(limit[2])
  res[3] = tonumber(limit[3]) or ARGV[3]
  res[4] = tonumber(limit[4])

  if res[1] >= 0 then
    redis.call('hincrby', KEYS[1], 'ct', -1)
  else
    res[1] = -1
  end

  if policyCount > 1 and res[1] == -1 then
    redis.call('incr', statusKey)
    redis.call('pexpire', statusKey, res[3] * 2)
  end

else

  local index = 1
  if policyCount > 1 then
    index = tonumber(redis.call('get', statusKey)) or 1
    if index > policyCount then
      index = policyCount
    end
  end

  local total = tonumber(ARGV[index * 2])
  res[1] = total - 1
  res[2] = total
  res[3] = tonumber(ARGV[index * 2 + 1])
  res[4] = tonumber(ARGV[1]) + res[3]

  redis.call('hmset', KEYS[1], 'ct', res[1], 'lt', res[2], 'dn', res[3], 'rt', res[4])
  redis.call('pexpire', KEYS[1], res[3])

  if policyCount > 1 then
    redis.call('set', statusKey, index)
    redis.call('pexpire', statusKey, res[3] * 2)
  end

end

return res
`
