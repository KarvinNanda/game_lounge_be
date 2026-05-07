package dto

type CreateStaffRequest struct {
	RoleID      uint     `json:"role_id" binding:"required"`
	Username    string   `json:"username" binding:"required,min=3,max=100"`
	Email       string   `json:"email" binding:"required,email"`
	Phone       string   `json:"phone"`
	Password    string   `json:"password" binding:"required,min=6"`
	IsAllStores bool     `json:"is_all_stores"`
	StoreIDs    []string `json:"store_ids"`
}

type UpdateStaffRequest struct {
	RoleID      uint     `json:"role_id" binding:"required"`
	Username    string   `json:"username" binding:"required,min=3,max=100"`
	Email       string   `json:"email" binding:"required,email"`
	Phone       string   `json:"phone"`
	IsAllStores bool     `json:"is_all_stores"`
	StoreIDs    []string `json:"store_ids"`
}

type StaffFilter struct {
	Search  string `form:"search"`
	RoleID  uint   `form:"role_id"`
	StoreID string `form:"store_id"`
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=10"`
}
