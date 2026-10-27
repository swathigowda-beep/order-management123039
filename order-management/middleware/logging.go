package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func StructuredLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		reqID, _ := c.Get(RequestIDKey)
		errs := c.Errors.ByType(gin.ErrorTypePrivate).String()

		log.Printf("request_id=%v method=%s path=%s status=%d duration=%s error=%s",
			reqID, c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration, errs)
	}
}