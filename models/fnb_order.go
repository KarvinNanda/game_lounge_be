package models

import "time"

// FnbOrder menyimpan pesanan FnB dari customer selama sesi bermain.
// ID dibuat via uuid.NewString() di service layer — tidak pakai default DB.
type FnbOrder struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	BookingID   string    `gorm:"type:char(36);not null;index" json:"booking_id"`
	CustomerID  string    `gorm:"type:char(36);not null;index" json:"customer_id"`
	StoreID     string    `gorm:"type:char(36);not null;index" json:"store_id"`
	RoomID      string    `gorm:"type:char(36);not null" json:"room_id"`
	Status      string    `gorm:"type:enum('pending','preparing','delivered','cancelled');default:'pending'" json:"status"`
	Notes       *string   `gorm:"type:text" json:"notes"`
	TotalAmount float64   `gorm:"type:decimal(12,2);default:0" json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Booking  Booking        `gorm:"foreignKey:BookingID"  json:"booking,omitempty"`
	Customer Customer       `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Store    Store          `gorm:"foreignKey:StoreID"    json:"store,omitempty"`
	Room     StoreRoom      `gorm:"foreignKey:RoomID"     json:"room,omitempty"`
	Items    []FnbOrderItem `gorm:"foreignKey:OrderID"    json:"items,omitempty"`
}

// FnbOrderItem menyimpan detail item dalam 1 order FnB.
type FnbOrderItem struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID   string    `gorm:"type:char(36);not null;index" json:"order_id"`
	ItemID    uint      `gorm:"not null;index" json:"item_id"`
	ItemName  string    `gorm:"size:150;not null" json:"item_name"` // snapshot nama saat order
	Quantity  int       `gorm:"not null;default:1" json:"quantity"`
	Price     float64   `gorm:"type:decimal(12,2);not null" json:"price"` // snapshot harga saat order
	Notes     *string   `gorm:"type:text" json:"notes"`
	CreatedAt time.Time `json:"created_at"`

	Item FnbItem `gorm:"foreignKey:ItemID" json:"item,omitempty"`
}
