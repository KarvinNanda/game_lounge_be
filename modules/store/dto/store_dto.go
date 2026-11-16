package dto

type OperatingHourInput struct {
	DayType   string `json:"day_type" binding:"required,oneof=weekday weekend"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsActive  bool   `json:"is_active"`
}

type HolidayInput struct {
	Date      string `json:"date" binding:"required,datetime=2006-01-02"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

type RoomSetupInput struct {
	RoomTemplateID uint `json:"room_template_id"`
	UnitCount      int  `json:"unit_count"`
}

type CreateStoreRequest struct {
	Name           string               `json:"name" binding:"required,min=2,max=100"`
	Address        string               `json:"address" binding:"required"`
	Whatsapp       string               `json:"whatsapp"`
	PostalCode     string               `json:"postal_code"`
	Description    string               `json:"description"`
	PhotoURL       string               `json:"photo_url"`
	LinkGmaps      string               `json:"link_gmaps"`
	Status         string               `json:"status"`
	OperatingHours []OperatingHourInput `json:"operating_hours" binding:"dive"`
	Holidays       []HolidayInput       `json:"holidays" binding:"dive"`
	Rooms          []RoomSetupInput     `json:"rooms"`
}

type UpdateStoreRequest struct {
	Name           string               `json:"name" binding:"required,min=2,max=100"`
	Address        string               `json:"address" binding:"required"`
	Whatsapp       string               `json:"whatsapp"`
	PostalCode     string               `json:"postal_code"`
	Description    string               `json:"description"`
	PhotoURL       string               `json:"photo_url"`
	LinkGmaps      string               `json:"link_gmaps"`
	Status         string               `json:"status"`
	OperatingHours []OperatingHourInput `json:"operating_hours" binding:"dive"`
	Holidays       []HolidayInput       `json:"holidays" binding:"dive"`
	Rooms          []RoomSetupInput     `json:"rooms"`
}

type StoreFilter struct {
	Search  string `form:"search"`
	Status  string `form:"status"`
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=10"`
}
