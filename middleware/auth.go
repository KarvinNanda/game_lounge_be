package middleware

import (
	"net/http"
	"strings"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid authorization format",
			})
			return
		}

		claims, err := utils.ValidateJWT(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid or expired token",
			})
			return
		}

		c.Set("staff_id", claims.StaffID)
		c.Set("staff_username", claims.Username)
		c.Set("role_id", claims.RoleID)
		c.Set("staff_role_id", claims.RoleID) // alias eksplisit untuk kontrol akses
		c.Set("is_system", claims.IsSystem)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}
