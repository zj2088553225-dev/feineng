package middleware

import (
	"backend/models/res"
	"github.com/gin-gonic/gin"
	"strings"
)

func LocalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP != "127.0.0.1" && clientIP != "::1" && !strings.HasPrefix(clientIP, "127.") {
			res.FailWithMessage("仅允许本地访问", c)
			c.Abort()
			return
		}
		c.Next()
	}
}
