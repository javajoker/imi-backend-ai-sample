// internal/middleware/rate_limit.go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	visitors map[string]*visitor
	mtx      sync.Mutex
	rate     rate.Limit
	burst    int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    b,
	}

	// Clean up old visitors every minute
	go rl.cleanupVisitors()

	return rl
}

func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		rl.mtx.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mtx.Unlock()
	}
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mtx.Lock()
	defer rl.mtx.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getVisitor(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Default rate limiters
var (
	generalLimiter = NewRateLimiter(rate.Every(time.Second), 10) // 10 requests per second
	authLimiter    = NewRateLimiter(rate.Every(time.Minute), 5)  // 5 auth requests per minute
	uploadLimiter  = NewRateLimiter(rate.Every(time.Minute), 10) // 10 uploads per minute
)

func GeneralRateLimit() gin.HandlerFunc {
	return generalLimiter.Middleware()
}

func AuthRateLimit() gin.HandlerFunc {
	return authLimiter.Middleware()
}

func UploadRateLimit() gin.HandlerFunc {
	return uploadLimiter.Middleware()
}
