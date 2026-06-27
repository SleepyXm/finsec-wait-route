package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

func APIKeyMiddleware(c *gin.Context) {
	key := c.GetHeader("x-api-key")
	if key != os.Getenv("xage47282") {
		c.JSON(401, gin.H{"error": "unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}
