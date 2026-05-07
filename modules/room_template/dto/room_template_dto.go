package dto

type CreateRoomTemplateRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	CapacityMin uint   `json:"capacity_min" binding:"required,min=1"`
	CapacityMax uint   `json:"capacity_max" binding:"required,min=1"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	FacilityIDs []uint `json:"facility_ids"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateRoomTemplateRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	CapacityMin uint   `json:"capacity_min" binding:"required,min=1"`
	CapacityMax uint   `json:"capacity_max" binding:"required,min=1"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	FacilityIDs []uint `json:"facility_ids"`
	IsActive    *bool  `json:"is_active"`
}

type RoomTemplateFilter struct {
	Search  string `form:"search"`
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=10"`
}
