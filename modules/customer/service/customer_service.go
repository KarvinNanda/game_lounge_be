package service

import (
	"errors"
	"log"
	"time"

	"game_lounge_be/models"
	"game_lounge_be/modules/customer/dto"
	"game_lounge_be/modules/customer/repository"
	ntService "game_lounge_be/modules/notification_template/service"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

// ── Get All ───────────────────────────────────────────────────────────────────

func GetAllCustomers(filter dto.CustomerListFilter) ([]models.Customer, int64, error) {
	return repository.FindAllCustomers(filter.Search, filter.Type, filter.Status, filter.Page, filter.PerPage)
}

// ── Get By ID ─────────────────────────────────────────────────────────────────

func GetCustomerByID(id string) (*models.Customer, error) {
	customer, err := repository.FindCustomerByID(id)
	if err != nil {
		return nil, errors.New("customer tidak ditemukan")
	}
	return customer, nil
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

	// Generate & hash password jika ada email
	var plainPassword string
	if req.Email != "" {
		plainPassword = utils.GeneratePasswordFromName(req.Name)
		hash, err := utils.HashPassword(plainPassword)
		if err != nil {
			return nil, errors.New("gagal memproses password")
		}
		customer.PasswordHash = &hash
	}

	if err := repository.CreateCustomer(customer); err != nil {
		return nil, errors.New("gagal membuat customer")
	}

	// Sync favorite room types
	if len(req.FavoriteRoomTypes) > 0 {
		_ = repository.SyncFavoriteRoomTypes(customer.ID, req.FavoriteRoomTypes)
	}

	// Kirim notifikasi via template dari DB (non-blocking)
	if req.Email != "" && plainPassword != "" {
		name := req.Name
		email := req.Email
		pwd := plainPassword
		wa := req.Whatsapp
		go func() {
			tmpl, err := ntService.GetRendered("customer_welcome", map[string]string{
				"nama_customer": name,
				"email":         email,
				"password":      pwd,
			})
			if err != nil {
				// Fallback ke fungsi email lama jika template belum ada di DB
				_ = utils.SendCustomerPasswordEmail(email, name, pwd)
				return
			}
			if tmpl.IsEmailActive {
				if sendErr := utils.SendEmail(email, name, tmpl.EmailSubject, tmpl.EmailBody); sendErr != nil {
					log.Printf("[Customer] Gagal kirim email ke %s: %v", email, sendErr)
				}
			}
			if tmpl.IsWhatsappActive && wa != "" {
				log.Printf("[Customer][WhatsApp placeholder] → %s: %s", wa, tmpl.WhatsappBody)
			}
		}()
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

	plainPassword := utils.GeneratePasswordFromName(customer.Name)
	hash, err := utils.HashPassword(plainPassword)
	if err != nil {
		return errors.New("gagal memproses password")
	}

	customer.PasswordHash = &hash
	if err := repository.UpdateCustomer(customer); err != nil {
		return errors.New("gagal menyimpan password baru")
	}

	email := *customer.Email
	name := customer.Name
	pwd := plainPassword
	tmpl, tmplErr := ntService.GetRendered("customer_welcome", map[string]string{
		"nama_customer": name,
		"email":         email,
		"password":      pwd,
	})
	if tmplErr != nil {
		// Fallback ke fungsi email lama
		if err := utils.SendCustomerPasswordEmail(email, name, pwd); err != nil {
			return errors.New("password diperbarui, namun gagal mengirim email: " + err.Error())
		}
	} else if tmpl.IsEmailActive {
		if err := utils.SendEmail(email, name, tmpl.EmailSubject, tmpl.EmailBody); err != nil {
			return errors.New("password diperbarui, namun gagal mengirim email: " + err.Error())
		}
	}

	return nil
}
