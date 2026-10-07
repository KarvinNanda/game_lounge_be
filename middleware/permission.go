package middleware

import (
	"net/http"
	"slices"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/gin-gonic/gin"
)

// StaffAccess adalah hak akses staff yang dibaca ulang dari DB setiap request.
// Permission di JWT TIDAK dipakai untuk otorisasi: JWT berlaku 24 jam, sedangkan
// role bisa diubah atau staff dihapus kapan saja.
type StaffAccess struct {
	IsSystem     bool
	RoleID       uint
	Permissions  []string
	TokenVersion uint
	AllStores    bool     // is_all_stores atau Super Admin
	StoreIDs     []string // dari staff_stores, dipakai jika !AllStores
}

// loadStaffAccess bisa diganti di unit test (tanpa DB).
var loadStaffAccess = dbLoadStaffAccess

func dbLoadStaffAccess(staffID string) (*StaffAccess, error) {
	var staff models.Staff
	err := config.DB.Preload("Role", "deleted_at IS NULL").Preload("Role.Permissions").
		Where("id = ? AND deleted_at IS NULL", staffID).
		First(&staff).Error
	if err != nil {
		return nil, err
	}
	// Role terhapus → Preload mengembalikan Role kosong (ID 0) → anggap tanpa akses.
	access := &StaffAccess{IsSystem: staff.Role.ID != 0 && staff.Role.IsSystem, RoleID: staff.RoleID, TokenVersion: staff.TokenVersion}
	for _, p := range staff.Role.Permissions {
		access.Permissions = append(access.Permissions, p.Permission)
	}
	access.AllStores = access.IsSystem || staff.IsAllStores
	if !access.AllStores {
		if err := config.DB.Model(&models.StaffStore{}).Where("staff_id = ?", staffID).
			Pluck("store_id", &access.StoreIDs).Error; err != nil {
			return nil, err
		}
	}
	return access, nil
}

// RequirePermission meloloskan request jika staff adalah Super Admin (is_system)
// atau punya SALAH SATU permission yang disebut. Harus dipasang setelah AuthMiddleware.
func RequirePermission(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("is_system") {
			c.Next()
			return
		}
		owned := c.GetStringSlice("permissions")
		for _, p := range perms {
			if slices.Contains(owned, p) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Anda tidak punya akses untuk aksi ini",
		})
	}
}

// RequireSuperAdmin hanya meloloskan staff dengan role is_system.
func RequireSuperAdmin() gin.HandlerFunc {
	return RequirePermission()
}
