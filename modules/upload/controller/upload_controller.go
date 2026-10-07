package controller

import (
	"net/http"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// Upload handles POST /api/upload
// Content-Type: multipart/form-data
// Fields:
//   - file   : image file (required)
//   - folder : target subfolder, e.g. "stores", "facilities" (optional, default "img")
func Upload(c *gin.Context) {
	// ── Get folder (optional, default "img") ─────────────────
	folder := c.PostForm("folder")
	if folder == "" {
		folder = "img"
	}

	// ── Get file ─────────────────────────────────────────────
	file, err := c.FormFile("file")
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "File tidak ditemukan, pastikan field bernama 'file'")
		return
	}

	// ── Save to ./assets/img/{folder}/ ───────────────────────
	url, err := utils.SaveFileToAssets(file, folder)
	if err != nil {
		utils.ResponseError(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// ── Return public path ────────────────────────────────────
	utils.ResponseSuccess(c, http.StatusOK, "Upload berhasil", gin.H{"url": url})
}
