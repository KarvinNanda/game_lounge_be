package controller

import (
	"encoding/json"
	"net/http"

	"game_lounge_be/modules/play_credits/dto"
	"game_lounge_be/modules/play_credits/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// ── Package ───────────────────────────────────────────────────────────────────

// GetAllPackages mengambil list paket dengan filter.
func GetAllPackages(c *gin.Context) {
	var filter dto.PackageFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	packages, total, err := service.GetAllPackages(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data paket")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", packages, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}

// GetActivePackages mengambil semua paket aktif (untuk dropdown).
func GetActivePackages(c *gin.Context) {
	packages, err := service.GetActivePackages()
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil paket aktif")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", packages)
}

// GetPackageByID mengambil detail 1 paket.
func GetPackageByID(c *gin.Context) {
	id := c.Param("id")
	pkg, err := service.GetPackageByID(id)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", pkg)
}

// CreatePackage membuat paket baru (multipart/form-data: mendukung upload icon).
func CreatePackage(c *gin.Context) {
	var req dto.CreatePackageRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	// store_ids bisa dikirim sebagai JSON string karena multipart
	if storeIDsJSON := c.PostForm("store_ids"); storeIDsJSON != "" {
		_ = json.Unmarshal([]byte(storeIDsJSON), &req.StoreIDs)
	}

	iconURL := ""
	if file, err := c.FormFile("icon"); err == nil {
		if url, err := utils.SaveFileToAssets(file, "play_credits"); err == nil {
			iconURL = url
		}
	}

	actorName := c.GetString("staff_username")
	pkg, err := service.CreatePackage(req, iconURL, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Paket berhasil dibuat", pkg)
}

// UpdatePackage mengupdate paket (multipart/form-data: mendukung upload icon baru).
func UpdatePackage(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdatePackageRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	// store_ids bisa dikirim sebagai JSON string karena multipart
	if storeIDsJSON := c.PostForm("store_ids"); storeIDsJSON != "" {
		_ = json.Unmarshal([]byte(storeIDsJSON), &req.StoreIDs)
	}

	iconURL := ""
	if file, err := c.FormFile("icon"); err == nil {
		if url, err := utils.SaveFileToAssets(file, "play_credits"); err == nil {
			iconURL = url
		}
	}

	actorName := c.GetString("staff_username")
	pkg, err := service.UpdatePackage(id, req, iconURL, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Paket berhasil diupdate", pkg)
}

// DeletePackage soft-delete paket.
func DeletePackage(c *gin.Context) {
	id := c.Param("id")
	actorName := c.GetString("staff_username")
	if err := service.DeletePackage(id, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Paket berhasil dihapus", nil)
}

// ── Member Credits ────────────────────────────────────────────────────────────

// GetAllMemberCredits mengambil list credits member.
func GetAllMemberCredits(c *gin.Context) {
	var filter dto.MemberCreditFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	credits, total, err := service.GetAllMemberCredits(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data credits")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", credits, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}

// GetCreditByID mengambil detail 1 credit.
func GetCreditByID(c *gin.Context) {
	id := c.Param("id")
	credit, err := service.GetCreditByID(id)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", credit)
}

// AssignCredit admin assign credits ke customer secara manual.
func AssignCredit(c *gin.Context) {
	var req dto.AssignCreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	credit, err := service.AssignCredit(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Credits berhasil di-assign", credit)
}

// AdjustCredit admin adjust jam dan/atau perpanjang masa berlaku.
func AdjustCredit(c *gin.Context) {
	id := c.Param("id")
	var req dto.AdjustCreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	credit, err := service.AdjustCredit(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Credits berhasil diupdate", credit)
}

// DeleteCredit soft-delete credit.
func DeleteCredit(c *gin.Context) {
	id := c.Param("id")
	actorName := c.GetString("staff_username")
	if err := service.DeleteCredit(id, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Credits berhasil dihapus", nil)
}
