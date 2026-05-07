package controller

import (
	"net/http"
	"strconv"
	"game_lounge_be/modules/room_template/dto"
	"game_lounge_be/modules/room_template/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func GetAll(c *gin.Context) {
	var filter dto.RoomTemplateFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	templates, total, stats, err := service.GetAllRoomTemplates(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data room template")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true, "message": "OK",
		"data":  templates,
		"meta":  utils.Meta{Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage},
		"stats": stats,
	})
}

func GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	tmpl, err := service.GetRoomTemplateByID(uint(id))
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", tmpl)
}

func Create(c *gin.Context) {
	var req dto.CreateRoomTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	tmpl, err := service.CreateRoomTemplate(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Room template berhasil dibuat", tmpl)
}

func Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	var req dto.UpdateRoomTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	tmpl, err := service.UpdateRoomTemplate(uint(id), req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Room template berhasil diupdate", tmpl)
}

func Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	actorName := c.GetString("staff_username")
	if err := service.DeleteRoomTemplate(uint(id), actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Room template berhasil dihapus", nil)
}
