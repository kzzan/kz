package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"example/pkg/cache"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func RateLimit(c cache.Cache, logger *zerolog.Logger, maxReqs int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key      := fmt.Sprintf("rate:%s", ctx.ClientIP())
		cacheCtx := context.Background()

		val, err := c.Get(cacheCtx, key)
		if err != nil {
			logger.Warn().Err(err).Msg("rate limit cache error")
			ctx.Next()
			return
		}

		if val == "" {
			_ = c.Set(cacheCtx, key, "1", time.Minute)
			ctx.Next()
			return
		}

		count, _ := strconv.Atoi(val)
		if count >= maxReqs {
			logger.Warn().
				Str("ip", ctx.ClientIP()).
				Int("count", count).
				Msg("rate limit exceeded")
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}

		_ = c.Set(cacheCtx, key, strconv.Itoa(count+1), time.Minute)
		ctx.Next()
	}
}