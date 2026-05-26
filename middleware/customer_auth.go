package middleware

import (
	"net/http"
	"strings"

	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// CustomerAuth middleware untuk endpoint yang butuh customer JWT.
// Set "customer_id" dan "customer_type" ke context.
func CustomerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Token tidak ditemukan",
			})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ValidateCustomerJWT(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Token tidak valid atau sudah kadaluwarsa",
			})
			c.Abort()
			return
		}

		c.Set("customer_id", claims.CustomerID)
		c.Set("customer_type", claims.CustomerType)
		c.Next()
	}
}
