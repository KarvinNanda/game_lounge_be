package dto

type CreateCategoryRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	IconURL string `json:"icon_url"`
}

type UpdateCategoryRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	IconURL string `json:"icon_url"`
}

type CreateFacilityRequest struct {
	CategoryID  uint   `json:"category_id" binding:"required"`
	Name        string `json:"name" binding:"required,min=2,max=100"`
	IconURL     string `json:"icon_url"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateFacilityRequest struct {
	CategoryID  uint   `json:"category_id" binding:"required"`
	Name        string `json:"name" binding:"required,min=2,max=100"`
	IconURL     string `json:"icon_url"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type FacilityFilter struct {
	Search     string `form:"search"`
	CategoryID uint   `form:"category_id"`
	IsActive   *bool  `form:"is_active"`
	Page       int    `form:"page,default=1"`
	PerPage    int    `form:"per_page,default=10"`
}
