package middleware

import (
	"net/http"

	"game_lounge_be/config"
	"game_lounge_be/models"
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

		// Status, tipe, dan token_version dibaca dari DB: customer yang logout,
		// ganti password, atau dinonaktifkan langsung kehilangan akses.
		access, err := loadCustomerAccess(claims.CustomerID)
		if err != nil || access.TokenVersion != claims.TokenVersion {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Sesi tidak valid atau sudah berakhir",
			})
			c.Abort()
			return
		}

		c.Set("customer_id", claims.CustomerID)
		c.Set("customer_type", access.Type)
		c.Next()
	}
}

// CustomerAccess adalah data customer yang dibaca ulang dari DB setiap request.
type CustomerAccess struct {
	Type         string
	TokenVersion uint
}

// loadCustomerAccess bisa diganti di unit test (tanpa DB).
var loadCustomerAccess = func(customerID string) (*CustomerAccess, error) {
	var cust models.Customer
	err := config.DB.Select("type", "token_version").
		Where("id = ? AND status = 'active' AND deleted_at IS NULL", customerID).
		First(&cust).Error
	if err != nil {
		return nil, err
	}
	return &CustomerAccess{Type: cust.Type, TokenVersion: cust.TokenVersion}, nil
}
