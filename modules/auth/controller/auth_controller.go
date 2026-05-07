package controller

import (
	"net/http"
	"game_lounge_be/modules/auth/dto"
	"game_lounge_be/modules/auth/repository"
	"game_lounge_be/modules/auth/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	resp, err := service.Login(req)
	if err != nil {
		utils.ResponseError(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Login berhasil", resp)
}

func Me(c *gin.Context) {
	staffID := c.GetString("staff_id")
	staff, err := repository.FindStaffByID(staffID)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Staff tidak ditemukan")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", staff)
}

func Logout(c *gin.Context) {
	utils.ResponseSuccess(c, http.StatusOK, "Logout berhasil", nil)
}
