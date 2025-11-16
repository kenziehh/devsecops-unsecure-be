package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/go-redis/redis/v8"
)

func RateLimiter(redis *redis.Client, limit int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := fmt.Sprintf("rate:%s:%s", c.Path(), c.IP())

		count, err := redis.Incr(c.Context(), key).Result()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Redis error")
		}

		if count == 1 {
			redis.Expire(c.Context(), key, window)
		}

		if count > int64(limit) {
			return c.Status(429).JSON(fiber.Map{
				"success": false,
				"message": "Too many requests. Please try again later.",
			})
		}

		return c.Next()
	}
}
