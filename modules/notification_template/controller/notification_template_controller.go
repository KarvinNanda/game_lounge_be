package controller

import (
	"net/http"

	"game_lounge_be/modules/notification_template/dto"
	"game_lounge_be/modules/notification_template/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetAll mengambil semua template notifikasi.
func GetAll(c *gin.Context) {
	templates, err := service.GetAll()
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil template")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", templates)
}

// GetByKey mengambil satu template berdasarkan key.
func GetByKey(c *gin.Context) {
	key := c.Param("key")
	t, err := service.GetByKey(key)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", t)
}

// Update menyimpan perubahan template.
func Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	t, err := service.Update(key, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Template berhasil disimpan", t)
}

// Preview merender template dengan data contoh.
func Preview(c *gin.Context) {
	var req dto.PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	rendered, err := service.Preview(req)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", gin.H{"rendered": rendered})
}
