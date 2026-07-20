package middleware

import (
	"net/http"

	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// CustomerAuth middleware untuk endpoint yang butuh customer JWT.
// Token dibaca dari httpOnly cookie (cookie-only, bukan Authorization header).
// Set "customer_id" dan "customer_type" ke context.
func CustomerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !csrfOriginAllowed(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Origin tidak diizinkan",
			})
			return
		}

		token, err := c.Cookie(utils.CustomerCookieName)
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Autentikasi diperlukan",
			})
			c.Abort()
			return
		}

		claims, err := utils.ValidateCustomerJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Sesi tidak valid atau sudah berakhir",
			})
			c.Abort()
			return
		}

		c.Set("customer_id", claims.CustomerID)
		c.Set("customer_type", claims.CustomerType)
		c.Next()
	}
}
