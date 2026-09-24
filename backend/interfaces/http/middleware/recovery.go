package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery 捕获 panic 并返回统一错误响应。
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered", "panic", r, "path", c.Request.URL.Path)
				c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 1999, "msg": "internal server error"})
			}
		}()
		c.Next()
	}
}
