package service

import (
	"errors"
	"log"
	"time"

	"game_lounge_be/models"
	"game_lounge_be/modules/customer/dto"
	"game_lounge_be/modules/customer/repository"
	"game_lounge_be/money"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

// ── Get All ───────────────────────────────────────────────────────────────────

// CustomerWithStats is a customer plus the counters the admin app shows
// (see repository.CustomerStats for the definitions). Customer fields stay at
// the top level of the JSON.
type CustomerWithStats struct {
	models.Customer
	TotalVisit   int64        `json:"total_visit"`
	TotalBooking int64        `json:"total_booking"`
	TotalSpent   money.Rupiah `json:"total_spent"`
}

func withStats(customers []models.Customer) ([]CustomerWithStats, error) {
	ids := make([]string, len(customers))
	for i, c := range customers {
		ids[i] = c.ID
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	stats, err := repository.FindCustomerStats(ids, time.Now().In(loc).Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	out := make([]CustomerWithStats, len(customers))
	for i, c := range customers {
		s := stats[c.ID]
		out[i] = CustomerWithStats{Customer: c, TotalVisit: s.TotalVisit, TotalBooking: s.TotalBooking, TotalSpent: s.TotalSpent}
	}
	return out, nil
}

func GetAllCustomers(filter dto.CustomerListFilter) ([]CustomerWithStats, int64, error) {
	customers, total, err := repository.FindAllCustomers(filter.Search, filter.Type, filter.Status, filter.Page, filter.PerPage)
	if err != nil {
		return nil, 0, err
	}
	out, err := withStats(customers)
	return out, total, err
}

// ── Get By ID ─────────────────────────────────────────────────────────────────

func GetCustomerByID(id string) (*CustomerWithStats, error) {
	customer, err := repository.FindCustomerByID(id)
	if err != nil {
		return nil, errors.New("customer tidak ditemukan")
	}
	out, err := withStats([]models.Customer{*customer})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func CreateCustomer(req dto.CreateCustomerRequest, createdBy string) (*models.Customer, error) {
	// Validasi whatsapp unik
	if _, err := repository.FindCustomerByWhatsapp(req.Whatsapp); err == nil {
		return nil, errors.New("nomor WhatsApp sudah terdaftar")
	}

	// Validasi email unik (jika diisi)
	if req.Email != "" {
		if _, err := repository.FindCustomerByEmail(req.Email); err == nil {
			return nil, errors.New("email sudah terdaftar")
		}
	}

	customerType := req.Type
	if customerType == "" {
		customerType = "regular"
	}

	customer := &models.Customer{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Whatsapp:  req.Whatsapp,
		Type:      customerType,
		Status:    "active",
		CreatedBy: &createdBy,
	}

	if req.Email != "" {
		customer.Email = &req.Email
	}
	if req.Gender != "" {
		customer.Gender = &req.Gender
	}
	if req.Occupation != "" {
		customer.Occupation = &req.Occupation
	}
	if req.Notes != "" {
		customer.Notes = &req.Notes
	}
	if req.DateOfBirth != "" {
		dob, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err == nil {
			customer.DateOfBirth = &dob
		}
	}

	// No password is generated: the customer sets one through the link in the
	// welcome message. Until then password_hash is NULL and login is refused.
	if err := repository.CreateCustomer(customer); err != nil {
		return nil, errors.New("gagal membuat customer")
	}

	// Sync favorite room types
	if len(req.FavoriteRoomTypes) > 0 {
		_ = repository.SyncFavoriteRoomTypes(customer.ID, req.FavoriteRoomTypes)
	}

	// Welcome message with a one-time set-password link (non-blocking).
	// The customer is already saved, so a failure here is only logged: the admin
	// can send a new link with "resend-password".
	if req.Email != "" {
		link, err := issueSetPasswordLink(customer.ID)
		if err != nil {
			log.Printf("[Customer] gagal membuat link set password untuk customer %s: %v", customer.ID, err)
		} else {
			msg := welcomeMessage{Email: req.Email, Name: req.Name, Whatsapp: req.Whatsapp, Link: link}
			utils.SafeGo(func() {
				if err := sendWelcome(msg); err != nil {
					log.Printf("[Customer] gagal kirim welcome ke customer %s: %v", customer.ID, err)
				}
			})
		}
	}

	return repository.FindCustomerByID(customer.ID)
}

// ── Update ────────────────────────────────────────────────────────────────────

func UpdateCustomer(id string, req dto.UpdateCustomerRequest, updatedBy string) (*models.Customer, error) {
	customer, err := repository.FindCustomerByID(id)
	if err != nil {
		return nil, errors.New("customer tidak ditemukan")
	}

	// Validasi whatsapp unik (kecuali diri sendiri)
	if existing, err := repository.FindCustomerByWhatsapp(req.Whatsapp); err == nil && existing.ID != id {
		return nil, errors.New("nomor WhatsApp sudah terdaftar")
	}

	// Validasi email unik (kecuali diri sendiri)
	if req.Email != "" {
		if existing, err := repository.FindCustomerByEmail(req.Email); err == nil && existing.ID != id {
			return nil, errors.New("email sudah terdaftar")
		}
	}

	customer.Name = req.Name
	customer.Whatsapp = req.Whatsapp
	customer.UpdatedBy = &updatedBy

	if req.Email != "" {
		customer.Email = &req.Email
	} else {
		customer.Email = nil
	}
	if req.Gender != "" {
		customer.Gender = &req.Gender
	} else {
		customer.Gender = nil
	}
	if req.Occupation != "" {
		customer.Occupation = &req.Occupation
	} else {
		customer.Occupation = nil
	}
	if req.Notes != "" {
		customer.Notes = &req.Notes
	} else {
		customer.Notes = nil
	}
	if req.Type != "" {
		customer.Type = req.Type
	}
	if req.Status != "" {
		customer.Status = req.Status
	}
	if req.DateOfBirth != "" {
		dob, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err == nil {
			customer.DateOfBirth = &dob
		}
	} else {
		customer.DateOfBirth = nil
	}

	if err := repository.UpdateCustomer(customer); err != nil {
		return nil, errors.New("gagal update customer")
	}

	// Sync favorite room types
	_ = repository.SyncFavoriteRoomTypes(customer.ID, req.FavoriteRoomTypes)

	return repository.FindCustomerByID(customer.ID)
}

// ── Update Notes ──────────────────────────────────────────────────────────────

func UpdateCustomerNotes(id string, req dto.UpdateCustomerNotesRequest, updatedBy string) (*models.Customer, error) {
	customer, err := repository.FindCustomerByID(id)
	if err != nil {
		return nil, errors.New("customer tidak ditemukan")
	}

	if req.Notes != "" {
		customer.Notes = &req.Notes
	} else {
		customer.Notes = nil
	}
	customer.UpdatedBy = &updatedBy

	if err := repository.UpdateCustomer(customer); err != nil {
		return nil, errors.New("gagal update catatan customer")
	}

	return repository.FindCustomerByID(customer.ID)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func DeleteCustomer(id string, deletedBy string) error {
	customer, err := repository.FindCustomerByID(id)
	if err != nil {
		return errors.New("customer tidak ditemukan")
	}
	now := time.Now()
	customer.DeletedAt = &now
	return repository.SoftDeleteCustomer(customer, deletedBy)
}

// ── Resend Password ───────────────────────────────────────────────────────────

func ResendPassword(id string) error {
	customer, err := repository.FindCustomerByID(id)
	if err != nil {
		return errors.New("customer tidak ditemukan")
	}
	if customer.Email == nil || *customer.Email == "" {
		return errors.New("customer tidak memiliki email, tidak dapat mengirim password")
	}

	// Sends a new set-password link. The current password and sessions stay as they
	// are until the customer uses the link (ResetPassword then ends old sessions).
	link, err := issueSetPasswordLink(customer.ID)
	if err != nil {
		return errors.New("gagal membuat link set password")
	}
	if err := sendWelcome(welcomeMessage{Email: *customer.Email, Name: customer.Name, Whatsapp: customer.Whatsapp, Link: link}); err != nil {
		return errors.New("link set password dibuat, namun gagal mengirim email: " + err.Error())
	}
	return nil
}
