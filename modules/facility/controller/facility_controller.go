package controller

import (
	"net/http"
	"strconv"
	"game_lounge_be/modules/facility/dto"
	"game_lounge_be/modules/facility/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// ── Category ──────────────────────────────────────────────────

func GetAllCategories(c *gin.Context) {
	cats, err := service.GetAllCategories()
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil kategori")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", cats)
}

func CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	cat, err := service.CreateCategory(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Kategori berhasil dibuat", cat)
}

func UpdateCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	cat, err := service.UpdateCategory(uint(id), req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Kategori berhasil diupdate", cat)
}

func DeleteCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	actorName := c.GetString("staff_username")
	if err := service.DeleteCategory(uint(id), actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Kategori berhasil dihapus", nil)
}

// ── Facility ──────────────────────────────────────────────────

func GetAll(c *gin.Context) {
	var filter dto.FacilityFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	facilities, total, stats, err := service.GetAllFacilities(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data fasilitas")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true, "message": "OK",
		"data":  facilities,
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
	facility, err := service.GetFacilityByID(uint(id))
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", facility)
}

func Create(c *gin.Context) {
	var req dto.CreateFacilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	facility, err := service.CreateFacility(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Fasilitas berhasil dibuat", facility)
}

func Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	var req dto.UpdateFacilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	facility, err := service.UpdateFacility(uint(id), req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Fasilitas berhasil diupdate", facility)
}

func Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}
	actorName := c.GetString("staff_username")
	if err := service.DeleteFacility(uint(id), actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Fasilitas berhasil dihapus", nil)
}
