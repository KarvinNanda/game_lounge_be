package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"

	"game_lounge_be/config"

	"github.com/gin-gonic/gin"
)

// Scope store diisi AuthMiddleware. Staff dengan is_all_stores (atau Super Admin)
// boleh semua store; staff lain hanya store di staff_stores.
const (
	ctxStoreAll = "store_scope_all"
	ctxStoreIDs = "store_scope_ids"
)

// StoreScope mengembalikan cabang yang boleh diakses staff pada request ini
// (diisi AuthMiddleware). all=true → semua cabang, ids diabaikan.
func StoreScope(c *gin.Context) (all bool, ids []string) {
	ids = c.GetStringSlice(ctxStoreIDs)
	if ids == nil {
		ids = []string{}
	}
	return c.GetBool(ctxStoreAll), ids
}

// CanAccessStore: apakah staff pada request ini boleh mengakses store tersebut.
func CanAccessStore(c *gin.Context, storeID string) bool {
	if c.GetBool(ctxStoreAll) {
		return true
	}
	return storeID != "" && slices.Contains(c.GetStringSlice(ctxStoreIDs), storeID)
}

func forbidStore(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": "Anda tidak punya akses ke cabang ini",
	})
}

// ScopeStoreQuery untuk endpoint list dengan filter ?store_id=.
// Staff 1 cabang tanpa filter → filter dipaksa ke cabangnya. Staff beberapa
// cabang wajib memilih cabang (tanpa filter = data semua cabang).
func ScopeStoreQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool(ctxStoreAll) {
			c.Next()
			return
		}
		ids := c.GetStringSlice(ctxStoreIDs)
		// Baca dari URL langsung, bukan c.Query(): c.Query() meng-cache query
		// sehingga perubahan RawQuery di bawah tidak terlihat oleh handler.
		storeID := c.Request.URL.Query().Get("store_id")
		switch {
		case storeID != "":
			if !CanAccessStore(c, storeID) {
				forbidStore(c)
				return
			}
		case len(ids) == 1:
			q := c.Request.URL.Query()
			q.Set("store_id", ids[0])
			c.Request.URL.RawQuery = q.Encode()
		case len(ids) == 0:
			forbidStore(c)
			return
		default:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Pilih cabang (store_id) terlebih dahulu",
			})
			return
		}
		c.Next()
	}
}

// RequireStoreParam untuk route dengan store ID di path (mis. /pricing/:store_id).
func RequireStoreParam(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CanAccessStore(c, c.Param(param)) {
			forbidStore(c)
			return
		}
		c.Next()
	}
}

// RequireStoreInBody untuk route yang menerima store_id di JSON body.
// Body dibaca lalu dikembalikan supaya handler tetap bisa bind seperti biasa.
func RequireStoreInBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool(ctxStoreAll) {
			c.Next()
			return
		}
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "message": "Body tidak valid"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		var body struct {
			StoreID string `json:"store_id"`
		}
		_ = json.Unmarshal(raw, &body)
		if !CanAccessStore(c, body.StoreID) {
			forbidStore(c)
			return
		}
		c.Next()
	}
}

// RequireEntityStore untuk route by-ID (booking, event, room, dst.): store
// milik entity dicari dulu. Entity tidak ditemukan → diteruskan (handler 404).
func RequireEntityStore(param string, lookup func(id string) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool(ctxStoreAll) {
			c.Next()
			return
		}
		storeID, err := lookup(c.Param(param))
		if err == nil && !CanAccessStore(c, storeID) {
			forbidStore(c)
			return
		}
		c.Next()
	}
}

// StoreOf mengembalikan lookup store_id untuk tabel yang punya kolom id & store_id.
func StoreOf(table string) func(id string) (string, error) {
	return func(id string) (string, error) {
		var storeID string
		res := config.DB.Table(table).Select("store_id").Where("id = ?", id).Limit(1).Scan(&storeID)
		if res.Error != nil {
			return "", res.Error
		}
		if res.RowsAffected == 0 {
			return "", errStoreLookupNotFound
		}
		return storeID, nil
	}
}

var errStoreLookupNotFound = errors.New("entity tidak ditemukan")
