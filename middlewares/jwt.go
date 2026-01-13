package middlewares

import (
	"net/http"
	"strings"
	"turtle/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		token := strings.Replace(auth, "Bearer ", "", 1)
		claims, err := jwt.VerifyToken(token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
