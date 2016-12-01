package smartlimiter_test

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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/teambition/gear-ratelimiter"

	"github.com/teambition/gear"
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
func TestSmartLimiterGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SmartLimiterGo Suite")
}

var _ = BeforeSuite(func() {

})

var _ = AfterSuite(func() {

})
var _ = Describe("smartLimiter", func() {
	It("smartLimiter with default Options should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			RedisAddr: "127.0.0.1:6379",
		})
		app := gear.New()
		app.Use(limiter)
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String())

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal(""))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal(""))
		res.Body.Close()
	})
	It("smartLimiter with get /a path should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"GET /a": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Get("/a", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/a")

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("Retry-After")).To(Equal(""))
		Expect(res.Header.Get("X-Ratelimit-Reset")).ToNot(Equal(""))
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("6"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("5"))
		res.Body.Close()
	})
	It("smartLimiter with post /a path should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"POST /a": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Post("/a", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("POST", "http://"+srv.Addr().String()+"/a")
		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("6"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("5"))
		res.Body.Close()
	})
	It("smartLimiter with / path should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			Policy: map[string][]int{
				"/": []int{6, 5 * 1000},
			},
			RedisAddr: "127.0.0.1:6379",
		})
		app := gear.New()
		app.Use(limiter)
		app.Use(func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String())

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("6"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("5"))
		res.Body.Close()
	})

	It("smartLimiter with /a path should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/a": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Get("/a", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/a")

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("6"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("5"))
		res.Body.Close()
	})

	It("smartLimiter with GET path should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"GET": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Get("/b", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/b")

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("6"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("5"))
		res.Body.Close()
	})
	It("smartLimiter with GET path and twice request should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"GET": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Get("/c", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		RequestBy("GET", "http://"+srv.Addr().String()+"/c")
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/c")

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("6"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("3"))
		res.Body.Close()
	})
	It("smartLimiter with /d and the request exceeds the limiter that should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/d": []int{3, 5 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
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

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("3"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("0"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/d")
		Expect(res.StatusCode).To(Equal(429))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("3"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))
		Expect(res.Header.Get("Retry-After")).ToNot(Equal(""))
		res.Body.Close()
	})

	It("smartLimiter with GetID func request should be", func() {

		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			GetID: func(req *http.Request) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/e": []int{6, 5 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
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

		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("6"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("5"))
		res.Body.Close()
	})
	It("smartLimiter with two policys that should be", func() {

		id := genID()
		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			GetID: func(req *http.Request) string {
				return id
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/f": []int{2, 2 * 1000, 1, 1 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Get("/f", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("1"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.StatusCode).To(Equal(200))
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("0"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.StatusCode).To(Equal(429))
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))

		time.Sleep(2 * time.Second)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.StatusCode).To(Equal(200))
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("1"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("0"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.StatusCode).To(Equal(429))
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("1"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))

		time.Sleep(2 * time.Second)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.StatusCode).To(Equal(200))
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("1"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("0"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/f")
		Expect(res.StatusCode).To(Equal(429))
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))

		res.Body.Close()
	})
	It("smartLimiter with multi-policy that should be", func() {

		id := genID()
		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			GetID: func(req *http.Request) string {
				return id
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/g": []int{2, 2 * 1000, 1, 1 * 1000, 3, 1 * 1000, 4, 10 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Get("/g", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))

		RequestBy("GET", "http://"+srv.Addr().String()+"/g")

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.StatusCode).To(Equal(429))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))

		time.Sleep(2 * time.Second)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("1"))

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.StatusCode).To(Equal(429))
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))

		time.Sleep(1 * time.Second)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("3"))

		RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		RequestBy("GET", "http://"+srv.Addr().String()+"/g")

		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.Header.Get("X-Ratelimit-Remaining")).To(Equal("-1"))

		time.Sleep(2 * time.Second)
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal("2"))

		res.Body.Close()
	})
	It("smartLimiter with wrong multi-policy that should be", func() {
		limiter := smartlimiter.NewLimiter(&smartlimiter.Options{
			GetID: func(req *http.Request) string {
				return genID()
			},
			RedisAddr: "127.0.0.1:6379",
			Policy: map[string][]int{
				"/g": []int{2, 2 * 1000, 1 * 1000, 3, 1 * 1000, 4, 10 * 1000},
			},
		})
		app := gear.New()
		app.Use(limiter)
		router := gear.NewRouter()
		router.Get("/g", func(ctx *gear.Context) error {
			return ctx.HTML(200, "")
		})
		app.UseHandler(router)

		srv := app.Start()
		defer srv.Close()
		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/g")
		Expect(res.StatusCode).To(Equal(200))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Header.Get("X-Ratelimit-Limit")).To(Equal(""))
	})
})
