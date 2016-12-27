package ratelimiter_test

import (
	"compress/gzip"
	"compress/zlib"
	"crypto/rand"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/teambition/gear"
	"github.com/teambition/gear-ratelimiter"
)

// ------Helpers for help test --------
var DefaultClient = &http.Client{}

type GearResponse struct {
	*http.Response
}

func RequestBy(method, url string) (*GearResponse, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func DefaultClientDo(req *http.Request) (*GearResponse, error) {
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func DefaultClientDoWithCookies(req *http.Request, cookies map[string]string) (*http.Response, error) {
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	return DefaultClient.Do(req)
}
func NewRequst(method, url string) (*http.Request, error) {
	return http.NewRequest(method, url, nil)
}

func (resp *GearResponse) OK() bool {
	return resp.StatusCode < 400
}
func (resp *GearResponse) Content() (val []byte, err error) {
	var b []byte
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		if reader, err = gzip.NewReader(resp.Body); err != nil {
			return nil, err
		}
	case "deflate":
		if reader, err = zlib.NewReader(resp.Body); err != nil {
			return nil, err
		}
	default:
		reader = resp.Body
	}

	defer reader.Close()
	if b, err = ioutil.ReadAll(reader); err != nil {
		return nil, err
	}
	return b, err
}

func (resp *GearResponse) Text() (val string, err error) {
	b, err := resp.Content()
	if err != nil {
		return "", err
	}
	return string(b), err
}
func genID() string {
	buf := make([]byte, 12)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

//--------- End ---------
func TestRateLimiter(t *testing.T) {

	t.Run("RateLimiter with not GetID func should be", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("getId function required")
			}
		}()
		ratelimiter.New(&ratelimiter.Options{
			RedisAddr: "127.0.0.1:6379",
		})
	})
	t.Run("RateLimiter with  GetID()=empty should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return ""
			},
			RedisAddr: "127.0.0.1:6379",
		})
		app := gear.New()
		app.UseHandler(limiter)
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String())

		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("", res.Header.Get("X-Ratelimit-Remaining"))
	})
	t.Run("RateLimiter with default Options should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
		})
		app := gear.New()
		app.UseHandler(limiter)
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String())

		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})
	t.Run("RateLimiter with get /a path should be", func(t *testing.T) {
		assert := assert.New(t)

		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"GET /a": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/a", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/a")

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("", res.Header.Get("Retry-After"))
		assert.NotEqual("", res.Header.Get("X-Ratelimit-Reset"))
		assert.Equal("6", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("5", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})
	t.Run("RateLimiter with post /a path should be", func(t *testing.T) {
		assert := assert.New(t)

		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"POST /a": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Post("/a", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("POST", "http://"+srv.Addr().String()+"/a")
		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("6", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("5", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})
	t.Run("RateLimiter with / path should be", func(t *testing.T) {
		assert := assert.New(t)

		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			Policy: map[string][]int{
				"/": []int{6, 5 * 1000},
			},
			RedisAddr: "127.0.0.1:6379",
		})
		app := gear.New()
		app.UseHandler(limiter)
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String())

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("6", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("5", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})

	t.Run("RateLimiter with /a path should be", func(t *testing.T) {
		assert := assert.New(t)

		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/a": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/a", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/a")

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("6", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("5", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})

	t.Run("RateLimiter with GET path should be", func(t *testing.T) {
		assert := assert.New(t)

		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"GET": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/b", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/b")

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("6", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("5", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})
	t.Run("ratelimiter with GET path and twice request should be", func(t *testing.T) {
		assert := assert.New(t)
		id := genID()
		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return id
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"GET": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/c", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		RequestBy("GET", "http://"+srv.Addr().String()+"/c")
		RequestBy("GET", "http://"+srv.Addr().String()+"/c")
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/c")

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("6", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("3", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})
	t.Run("ratelimiter with /d and the request exceeds the limiter that should be", func(t *testing.T) {
		assert := assert.New(t)

		id := genID()
		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return id
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/d": []int{3, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/d", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		RequestBy("GET", "http://"+srv.Addr().String()+"/d")
		RequestBy("GET", "http://"+srv.Addr().String()+"/d")
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/d")

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("3", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("0", res.Header.Get("X-Ratelimit-Remaining"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/d")
		assert.Equal(429, res.StatusCode)
		assert.Nil(err)
		assert.Equal("3", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))
		assert.NotEqual("", res.Header.Get("Retry-After"))
		res.Body.Close()
	})

	t.Run("RateLimiter with GetID func request should be", func(t *testing.T) {
		assert := assert.New(t)

		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/e": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/e", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		RequestBy("GET", "http://"+srv.Addr().String()+"/e")
		RequestBy("GET", "http://"+srv.Addr().String()+"/e")
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/e")

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("6", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("5", res.Header.Get("X-Ratelimit-Remaining"))
		res.Body.Close()
	})
	t.Run("RateLimiter with two policys that should be", func(t *testing.T) {
		assert := assert.New(t)

		id := genID()
		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return id
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/f": []int{2, 300, 1, 200},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/f", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("1", res.Header.Get("X-Ratelimit-Remaining"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal(200, res.StatusCode)
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("0", res.Header.Get("X-Ratelimit-Remaining"))
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal(429, res.StatusCode)
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))

		time.Sleep(300 * time.Millisecond)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal(200, res.StatusCode)
		assert.Equal("1", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("0", res.Header.Get("X-Ratelimit-Remaining"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal(429, res.StatusCode)
		assert.Equal("1", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))

		time.Sleep(2 * 200 * time.Millisecond)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal(200, res.StatusCode)
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("1", res.Header.Get("X-Ratelimit-Remaining"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("0", res.Header.Get("X-Ratelimit-Remaining"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		assert.Equal(429, res.StatusCode)
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))

		res.Body.Close()
	})
	t.Run("ratelimiter with multi-policy that should be", func(t *testing.T) {
		assert := assert.New(t)

		id := genID()
		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return id
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/g": []int{2, 2 * 100, 1, 1 * 100, 3, 1 * 100, 4, 5 * 100},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/g", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))

		RequestBy("GET", "http://"+srv.Addr().String()+"/g")

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal(429, res.StatusCode)
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))

		time.Sleep(200 * time.Millisecond)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal("1", res.Header.Get("X-Ratelimit-Limit"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal(429, res.StatusCode)
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))

		time.Sleep(100 * time.Millisecond)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal("3", res.Header.Get("X-Ratelimit-Limit"))

		RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		RequestBy("GET", "http://"+srv.Addr().String()+"/g")

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal("-1", res.Header.Get("X-Ratelimit-Remaining"))

		time.Sleep(200 * time.Millisecond)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal("2", res.Header.Get("X-Ratelimit-Limit"))

		res.Body.Close()
	})
	t.Run("ratelimiter with wrong multi-policy that should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/g": []int{2, 2 * 1000, 1 * 1000, 3, 1 * 1000, 4, 10 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/g", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("", res.Header.Get("X-Ratelimit-Limit"))
	})

	t.Run("RateLimiter without limited should be", func(t *testing.T) {
		assert := assert.New(t)

		limiter := ratelimiter.New(&ratelimiter.Options{
			GetID: func(ctx *gear.Context) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/h": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.UseHandler(limiter)
		router := gear.NewRouter()
		router.Get("/g", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/g")

		assert.Equal(200, res.StatusCode)
		assert.Nil(err)
		assert.Equal("", res.Header.Get("X-Ratelimit-Limit"))
		res.Body.Close()
	})

}
