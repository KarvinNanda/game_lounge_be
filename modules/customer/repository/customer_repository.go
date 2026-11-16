package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/money"
)

// ── Find All ──────────────────────────────────────────────────────────────────

func FindAllCustomers(search, customerType, status string, page, perPage int) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64

	query := config.DB.Model(&models.Customer{}).Where("deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR whatsapp LIKE ? OR email LIKE ?", like, like, like)
	}
	if customerType != "" {
		query = query.Where("type = ?", customerType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	err := query.
		Preload("FavoriteRoomTypes").
		Preload("FavoriteRoomTypes.RoomTemplate").
		Order("created_at DESC").
		Limit(perPage).Offset(offset).
		Find(&customers).Error

	return customers, total, err
}

// ── Find One ──────────────────────────────────────────────────────────────────

func FindCustomerByID(id string) (*models.Customer, error) {
	var customer models.Customer
	err := config.DB.
		Preload("FavoriteRoomTypes").
		Preload("FavoriteRoomTypes.RoomTemplate").
		Preload("BookingHistory").
		Preload("BookingHistory.Room").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func FindCustomerByEmail(email string) (*models.Customer, error) {
	var customer models.Customer
	err := config.DB.Where("email = ? AND deleted_at IS NULL", email).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func FindCustomerByWhatsapp(whatsapp string) (*models.Customer, error) {
	var customer models.Customer
	err := config.DB.Where("whatsapp = ? AND deleted_at IS NULL", whatsapp).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// ── Write ─────────────────────────────────────────────────────────────────────

func CreateCustomer(customer *models.Customer) error {
	return config.DB.Create(customer).Error
}

func UpdateCustomer(customer *models.Customer) error {
	return config.DB.Save(customer).Error
}

func SoftDeleteCustomer(customer *models.Customer, deletedBy string) error {
	customer.DeletedBy = &deletedBy
	return config.DB.Save(customer).Error
}

// ── Favorite Room Types ───────────────────────────────────────────────────────

// SyncFavoriteRoomTypes menghapus semua favorit lama lalu insert yang baru.
func SyncFavoriteRoomTypes(customerID string, roomTemplateIDs []uint) error {
	// Hapus lama
	if err := config.DB.Where("customer_id = ?", customerID).
		Delete(&models.CustomerFavoriteRoomType{}).Error; err != nil {
		return err
	}
	if len(roomTemplateIDs) == 0 {
		return nil
	}

	var records []models.CustomerFavoriteRoomType
	for _, rtID := range roomTemplateIDs {
		records = append(records, models.CustomerFavoriteRoomType{
			CustomerID:     customerID,
			RoomTemplateID: rtID,
		})
	}
	return config.DB.Create(&records).Error
}

// ── Stats ─────────────────────────────────────────────────────────────────────

// CustomerStats are the admin list/detail counters for one customer.
//   - TotalBooking: bookings that are not cancelled (any payment method).
//   - TotalVisit: of those, sessions that already happened: status completed, or
//     booking_date before today (statuses in the DB can be stale).
//   - TotalSpent: same definition as the sales report revenue: non-cancelled
//     cash bookings plus play credit purchases. Event bookings are not included,
//     as in the sales report.
type CustomerStats struct {
	TotalBooking int64
	TotalVisit   int64
	TotalSpent   money.Rupiah
}

// FindCustomerStats returns stats for the given customers in 2 queries.
// today is YYYY-MM-DD in WIB. Customers without data get zero stats.
func FindCustomerStats(ids []string, today string) (map[string]CustomerStats, error) {
	out := make(map[string]CustomerStats, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	var bookingRows []struct {
		CustomerID   string
		TotalBooking int64
		TotalVisit   int64
		Spent        money.Rupiah
	}
	if err := config.DB.Table("bookings").
		Select(`customer_id,
			COUNT(*) AS total_booking,
			COALESCE(SUM(status = 'completed' OR booking_date < ?), 0) AS total_visit,
			COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN total_price ELSE 0 END), 0) AS spent`, today).
		Where("customer_id IN ? AND status != 'cancelled'", ids).
		Group("customer_id").Scan(&bookingRows).Error; err != nil {
		return nil, err
	}
	for _, r := range bookingRows {
		out[r.CustomerID] = CustomerStats{TotalBooking: r.TotalBooking, TotalVisit: r.TotalVisit, TotalSpent: r.Spent}
	}

	var creditRows []struct {
		CustomerID string
		Spent      money.Rupiah
	}
	if err := config.DB.Table("customer_play_credits").
		Select("customer_id, COALESCE(SUM(payment_amount), 0) AS spent").
		Where("customer_id IN ? AND deleted_at IS NULL AND payment_amount IS NOT NULL", ids).
		Group("customer_id").Scan(&creditRows).Error; err != nil {
		return nil, err
	}
	for _, r := range creditRows {
		s := out[r.CustomerID]
		s.TotalSpent += r.Spent
		out[r.CustomerID] = s
	}
	return out, nil
}
