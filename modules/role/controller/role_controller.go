package controller

import (
	"net/http"
	"strconv"
	"game_lounge_be/modules/role/dto"
	"game_lounge_be/modules/role/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func GetAll(c *gin.Context) {
	var filter dto.RoleFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Parameter tidak valid", err.Error())
		return
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	roles, total, err := service.GetAllRoles(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data role")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", roles, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}

func GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}

	role, err := service.GetRoleByID(uint(id))
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", role)
}

func Create(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	actorName := c.GetString("staff_username")
	role, err := service.CreateRole(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusCreated, "Role berhasil dibuat", role)
}

func Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	actorName := c.GetString("staff_username")
	role, err := service.UpdateRole(uint(id), req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Role berhasil diupdate", role)
}

func Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}

	actorName := c.GetString("staff_username")
	if err := service.DeleteRole(uint(id), actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Role berhasil dihapus", nil)
}
