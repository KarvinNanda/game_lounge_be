package middleware

import (
	"net/http"
	"os"
	"strings"

	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// csrfOriginAllowed adalah proteksi CSRF berbasis Origin header untuk
// request state-changing (POST/PUT/PATCH/DELETE). Karena auth memakai cookie
// yang dikirim otomatis oleh browser, situs jahat bisa memicu request atas
// nama user — cek Origin memblokirnya.
//
// - GET/HEAD/OPTIONS → selalu lolos (tidak state-changing)
// - Origin kosong → lolos (non-browser client: curl, mobile app, server-to-server)
// - ALLOWED_ORIGINS kosong → lolos (dev mode)
// - Origin ada tapi tidak terdaftar → DITOLAK
func csrfOriginAllowed(c *gin.Context) bool {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		return true
	}
	allowed := os.Getenv("ALLOWED_ORIGINS")
	if allowed == "" {
		return true
	}
	for _, o := range strings.Split(allowed, ",") {
		if strings.TrimSpace(o) == origin {
			return true
		}
	}
	return false
}

// AuthMiddleware memvalidasi staff JWT dari httpOnly cookie.
// Token TIDAK lagi diterima via Authorization header — cookie-only.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !csrfOriginAllowed(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Origin tidak diizinkan",
			})
			return
		}

		token, err := c.Cookie(utils.StaffCookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Autentikasi diperlukan",
			})
			return
		}

		claims, err := utils.ValidateJWT(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Sesi tidak valid atau sudah berakhir",
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
