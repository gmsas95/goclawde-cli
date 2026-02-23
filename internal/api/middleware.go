package api

import (
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func (s *Server) authMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing authorization header"})
		}

		tokenString := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.Security.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		return c.Next()
	}
}

func (s *Server) securityHeadersMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' ws: wss:;")
		return c.Next()
	}
}

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = valid
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	times := rl.requests[ip]

	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[ip] = valid
		return false
	}

	rl.requests[ip] = append(valid, now)
	return true
}

func (s *Server) rateLimitMiddleware(limit int, window time.Duration) fiber.Handler {
	rl := newRateLimiter(limit, window)
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if !rl.isAllowed(ip) {
			return c.Status(429).JSON(fiber.Map{
				"error": "rate limit exceeded",
			})
		}
		return c.Next()
	}
}

func (s *Server) validateRequestMiddleware(maxBodySize int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Context().SetUserValue("max_body_size", maxBodySize)

		contentLength := c.Request().Header.ContentLength()
		if contentLength > maxBodySize {
			return c.Status(413).JSON(fiber.Map{
				"error": "request body too large",
			})
		}

		return c.Next()
	}
}

func (s *Server) requestSizeLimitMiddleware(maxSize int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Context().SetUserValue("max_body_size", maxSize)

		contentLength := c.Request().Header.ContentLength()
		if contentLength > maxSize {
			return c.Status(413).JSON(fiber.Map{
				"error": "request body too large",
			})
		}

		return c.Next()
	}
}
