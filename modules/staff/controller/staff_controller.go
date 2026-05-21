package controller

import (
	"net/http"
	"game_lounge_be/modules/staff/dto"
	"game_lounge_be/modules/staff/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func GetAll(c *gin.Context) {
	var filter dto.StaffFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	staffs, total, err := service.GetAllStaffs(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data staff")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", staffs, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}

func GetByID(c *gin.Context) {
	id := c.Param("id")
	staff, err := service.GetStaffByID(id)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", staff)
}

func Create(c *gin.Context) {
	var req dto.CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	actorName := c.GetString("staff_username")
	staff, err := service.CreateStaff(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusCreated, "Staff berhasil dibuat", staff)
}

func Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	actorName := c.GetString("staff_username")
	staff, err := service.UpdateStaff(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Staff berhasil diupdate", staff)
}

func Delete(c *gin.Context) {
	id := c.Param("id")
	currentStaffID := c.GetString("staff_id")       // UUID — used to prevent self-deletion
	actorName := c.GetString("staff_username")       // username — stored in deleted_by

	if id == currentStaffID {
		utils.ResponseError(c, http.StatusBadRequest, "Tidak dapat menghapus akun sendiri")
		return
	}

	if err := service.DeleteStaff(id, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Staff berhasil dihapus", nil)
}

// ResetPassword mereset password staff dan mengirimkan password baru ke email staff.
// Hanya bisa dilakukan oleh Super Admin (is_system = true atau role_id = 1).
func ResetPassword(c *gin.Context) {
	staffID := c.Param("id")

	// Validasi: hanya Super Admin atau is_system yang boleh reset
	isSystem := c.GetBool("is_system")
	roleID := c.GetUint("role_id")
	if !isSystem && roleID != 1 {
		utils.ResponseError(c, http.StatusForbidden, "Hanya Super Admin yang bisa reset password staff")
		return
	}

	result, err := service.ResetStaffPassword(staffID)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Password berhasil direset dan dikirim ke email staff", result)
}
