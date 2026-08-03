package middleware

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

const (
	SMSVerificationRateLimitMark = "SV"
	SMSVerificationMaxRequests   = 2  // 30秒内最多2次
	SMSVerificationDuration      = 30 // 30秒时间窗口
)

func redisSMSVerificationRateLimiter(c *gin.Context) {
	allowed, _, ttlSeconds, err := redisFixedWindowTake(
		c.Request.Context(),
		redisIPRateLimitKey(SMSVerificationRateLimitMark, c.ClientIP()),
		SMSVerificationMaxRequests,
		SMSVerificationDuration,
	)
	if err != nil {
		memorySMSVerificationRateLimiter(c)
		return
	}
	if allowed {
		c.Next()
		return
	}

	waitSeconds := int64(SMSVerificationDuration)
	if ttlSeconds > 0 {
		waitSeconds = ttlSeconds
	}

	c.JSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": fmt.Sprintf("发送过于频繁，请等待 %d 秒后再试", waitSeconds),
	})
	c.Abort()
}

func memorySMSVerificationRateLimiter(c *gin.Context) {
	key := SMSVerificationRateLimitMark + ":" + c.ClientIP()

	if !inMemoryRateLimiter.Request(key, SMSVerificationMaxRequests, SMSVerificationDuration) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "发送过于频繁，请稍后再试",
		})
		c.Abort()
		return
	}

	c.Next()
}

func SMSVerificationRateLimit() gin.HandlerFunc {
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		if common.RedisEnabled {
			redisSMSVerificationRateLimiter(c)
		} else {
			memorySMSVerificationRateLimiter(c)
		}
	}
}
