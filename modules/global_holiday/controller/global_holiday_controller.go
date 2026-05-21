package controller

import (
	"net/http"
	"strconv"

	"game_lounge_be/modules/global_holiday/dto"
	"game_lounge_be/modules/global_holiday/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func GetAll(c *gin.Context) {
	var filter dto.GlobalHolidayFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	holidays, total, err := service.GetAll(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OK",
		"data":    holidays,
		"meta": utils.Meta{
			Page:      filter.Page,
			PerPage:   filter.PerPage,
			Total:     total,
			TotalPage: totalPage,
		},
	})
}

func Create(c *gin.Context) {
	var req dto.CreateGlobalHolidayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	h, err := service.Create(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Tanggal merah berhasil ditambahkan", h)
}

func Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req dto.UpdateGlobalHolidayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	h, err := service.Update(uint(id), req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Berhasil diupdate", h)
}

func Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	actorName := c.GetString("staff_username")
	if err := service.Delete(uint(id), actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Berhasil dihapus", nil)
}
