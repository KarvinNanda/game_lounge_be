package dto

type CreateGlobalHolidayRequest struct {
	Date      string `json:"date" binding:"required"`       // YYYY-MM-DD
	Name      string `json:"name" binding:"required,min=2"`
	OpenTime  string `json:"open_time" binding:"required"`  // HH:MM
	CloseTime string `json:"close_time" binding:"required"` // HH:MM
}

type UpdateGlobalHolidayRequest struct {
	Name      string `json:"name" binding:"required,min=2"`
	OpenTime  string `json:"open_time" binding:"required"`
	CloseTime string `json:"close_time" binding:"required"`
}

type GlobalHolidayFilter struct {
	Search  string `form:"search"`           // cari berdasarkan nama holiday
	Year    int    `form:"year"`             // filter tahun (e.g. 2025)
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=20"`
}
