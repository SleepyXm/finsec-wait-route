package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/redis"
)

func NewRedisStore(rdb *redisclient.Client) limiter.Store {
	store, err := redis.NewStoreWithOptions(rdb, limiter.StoreOptions{
		Prefix: "waitlist_rl",
	})
	if err != nil {
		panic(err)
	}
	return store
}

// IPRateLimiter limits each IP to 5 signups per hour
func IPRateLimiter(store limiter.Store) func(http.Handler) http.Handler {
	rate := limiter.Rate{
		Period: time.Hour,
		Limit:  5,
	}
	lmt := limiter.New(store, rate)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)

			// Skip rate limiting for local development
			if (ip == "127.0.0.1" || ip == "::1") && os.Getenv("ENV") != "production" {
				next.ServeHTTP(w, r)
				return
			}

			ctx, err := lmt.Get(r.Context(), ip)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Set standard rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(ctx.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(ctx.Remaining, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(ctx.Reset, 10))

			if ctx.Reached {
				w.Header().Set("Retry-After", strconv.FormatInt(ctx.Reset-time.Now().Unix(), 10))
				http.Error(w, "Slow down!", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GlobalRateLimiter limits total signups to 100 per minute (burst protection)
func GlobalRateLimiter(store limiter.Store) func(http.Handler) http.Handler {
	rate := limiter.Rate{
		Period: time.Minute,
		Limit:  100,
	}
	lmt := limiter.New(store, rate)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := lmt.Get(r.Context(), "global")
			if err != nil || ctx.Reached {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIP extracts the actual client IP, respecting proxies
func realIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" { // Cloudflare
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

// WrapGin adapts a standard http middleware to a Gin HandlerFunc
func WrapGin(m func(http.Handler) http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
		if c.Writer.Written() {
			c.Abort() // 👈 tells Gin not to call the next handler
		}
	}
}
