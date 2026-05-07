package dto

type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required,min=2,max=100"`
	Permissions []string `json:"permissions"`
}

type UpdateRoleRequest struct {
	Name        string   `json:"name" binding:"required,min=2,max=100"`
	Permissions []string `json:"permissions"`
}

type RoleFilter struct {
	Search  string `form:"search"`
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=10"`
}
