package utils

import "github.com/gin-gonic/gin"

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

func ResponseError(c *gin.Context, statusCode int, message string) {
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
