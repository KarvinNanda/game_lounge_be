package utils

import (
	"log"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

type Meta struct {
	Page      int   `json:"page"`
	PerPage   int   `json:"per_page"`
	Total     int64 `json:"total"`
	TotalPage int   `json:"total_page"`
}

func ResponseSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"message": message,
		"data":    data,
	})
}

func ResponseSuccessPaginate(c *gin.Context, statusCode int, message string, data interface{}, meta Meta) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"message": message,
		"data":    data,
		"meta":    meta,
	})
}

// dbErrorPattern mengenali pesan error mentah dari driver MySQL / database/sql.
var dbErrorPattern = regexp.MustCompile(`Error \d{4} \(|^sql: |database is closed|connection refused`)

// ResponseError mengirim error ke client. Error 5xx dan pesan mentah dari DB
// tidak pernah dikirim apa adanya (bisa membocorkan nama tabel, kolom, index,
// alamat host): pesan asli dicatat di log, client mendapat pesan generik.
func ResponseError(c *gin.Context, statusCode int, message string) {
	if statusCode >= http.StatusInternalServerError || dbErrorPattern.MatchString(message) {
		method := ""
		if c.Request != nil {
			method = c.Request.Method
		}
		log.Printf("[ERROR] %s %s → %d: %s", method, c.FullPath(), statusCode, message)
		if statusCode >= http.StatusInternalServerError {
			message = "Terjadi kesalahan pada server. Silakan coba lagi."
		} else {
			message = "Permintaan tidak dapat diproses"
		}
	}
	c.JSON(statusCode, gin.H{
		"success": false,
		"message": message,
		"data":    nil,
	})
}

func ResponseValidationError(c *gin.Context, statusCode int, message string, errors interface{}) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"message": message,
		"errors":  errors,
	})
}
