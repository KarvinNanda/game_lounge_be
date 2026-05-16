package controller

import (
	"net/http"

	"game_lounge_be/modules/customer/dto"
	"game_lounge_be/modules/customer/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetAll mengambil semua customer dengan filter & paginasi.
func GetAll(c *gin.Context) {
	var filter dto.CustomerListFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	customers, total, err := service.GetAllCustomers(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data customer")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", customers, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}

// GetByID mengambil 1 customer berdasarkan ID.
func GetByID(c *gin.Context) {
	id := c.Param("id")
	customer, err := service.GetCustomerByID(id)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", customer)
}

// Create membuat customer baru.
func Create(c *gin.Context) {
	var req dto.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	customer, err := service.CreateCustomer(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Customer berhasil dibuat", customer)
}

// Update mengupdate data customer.
func Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	customer, err := service.UpdateCustomer(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Customer berhasil diupdate", customer)
}

// UpdateNotes mengupdate hanya field notes customer.
func UpdateNotes(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateCustomerNotesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	customer, err := service.UpdateCustomerNotes(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Catatan customer berhasil diupdate", customer)
}

// Delete soft-delete customer.
func Delete(c *gin.Context) {
	id := c.Param("id")
	actorName := c.GetString("staff_username")
	if err := service.DeleteCustomer(id, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Customer berhasil dihapus", nil)
}

// ResendPassword generate ulang password dan kirim via email.
func ResendPassword(c *gin.Context) {
	id := c.Param("id")
	if err := service.ResendPassword(id); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Password berhasil dikirim ulang ke email customer", nil)
}
