package service

import (
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/voucher/dto"
)

// ── ValidateVoucher ───────────────────────────────────────────

func TestValidateVoucher_Valid(t *testing.T) {
	skipIfNoDB(t)

	v, cleanup := createTestVoucher("TST-VALID-001", "percentage", 20)
	defer cleanup()

	resp, err := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       v.Code,
		StoreID:    "any-store",
		CustomerID: testCustomerID,
		Amount:     100000,
		UseType:    "booking",
	})
	if err != nil {
		t.Fatalf("ValidateVoucher error: %v", err)
	}
	if !resp.IsValid {
		t.Errorf("harus valid, got message: %q", resp.Message)
	}
	// 20% dari 100000 = 20000
	if resp.DiscountAmount != 20000 {
		t.Errorf("discount: want 20000, got %.0f", resp.DiscountAmount)
	}
}

func TestValidateVoucher_KodeTidakAda(t *testing.T) {
	skipIfNoDB(t)

	resp, err := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       "KODE-TIDAK-ADA-9999",
		StoreID:    "any-store",
		CustomerID: testCustomerID,
		Amount:     100000,
		UseType:    "booking",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsValid {
		t.Error("kode tidak ada harus invalid")
	}
}

func TestValidateVoucher_Expired(t *testing.T) {
	skipIfNoDB(t)

	v, cleanup := createTestVoucher("TST-EXP-001", "percentage", 10, withExpired())
	defer cleanup()

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       v.Code,
		CustomerID: testCustomerID,
		Amount:     100000,
		UseType:    "booking",
	})
	if resp.IsValid {
		t.Error("voucher expired harus invalid")
	}
}

func TestValidateVoucher_Inactive(t *testing.T) {
	skipIfNoDB(t)

	v, cleanup := createTestVoucher("TST-INACT-001", "nominal", 15000)
	defer cleanup()
	// GORM skip zero-value bool (false) saat INSERT jika ada default:true →
	// gunakan Update eksplisit setelah create.
	config.DB.Model(v).Update("is_active", false)

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       v.Code,
		CustomerID: testCustomerID,
		Amount:     100000,
		UseType:    "booking",
	})
	if resp.IsValid {
		t.Error("voucher inactive harus invalid")
	}
}

func TestValidateVoucher_SudahPernakDipakai(t *testing.T) {
	skipIfNoDB(t)

	v, cleanup := createTestVoucher("TST-USED-001", "nominal", 20000)
	defer cleanup()

	// Simulasi: customer ini sudah pakai voucher ini sebelumnya
	config.DB.Create(&models.VoucherUsage{
		VoucherID:      v.ID,
		CustomerID:     testCustomerID,
		DiscountAmount: 20000,
	})

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       v.Code,
		CustomerID: testCustomerID,
		Amount:     100000,
		UseType:    "booking",
	})
	if resp.IsValid {
		t.Error("voucher yang sudah pernah dipakai harus invalid")
	}
	if resp.Message != "Voucher sudah pernah digunakan" {
		t.Errorf("pesan: want 'Voucher sudah pernah digunakan', got %q", resp.Message)
	}
}

func TestValidateVoucher_MinPurchaseNotMet(t *testing.T) {
	skipIfNoDB(t)

	v, cleanup := createTestVoucher("TST-MIN-001", "nominal", 30000, withMinPurchase(200000))
	defer cleanup()

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       v.Code,
		CustomerID: testCustomerID,
		Amount:     150000, // kurang dari min purchase 200000
		UseType:    "booking",
	})
	if resp.IsValid {
		t.Error("amount di bawah min purchase harus invalid")
	}
}

func TestValidateVoucher_TipeTidakCocok(t *testing.T) {
	skipIfNoDB(t)

	// Voucher type="booking", dipakai untuk use_type="play_credits"
	v, cleanup := createTestVoucher("TST-TYPE-001", "nominal", 10000)
	defer cleanup()

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       v.Code,
		CustomerID: testCustomerID,
		Amount:     100000,
		UseType:    "play_credits", // berbeda dari voucher type
	})
	if resp.IsValid {
		t.Error("tipe tidak cocok harus invalid")
	}
}

func TestValidateVoucher_RoomTypeRestriction_Pass(t *testing.T) {
	skipIfNoDB(t)

	rt := models.RoomTemplate{Name: "TEST RT Voucher", CapacityMax: 4}
	config.DB.Create(&rt)
	defer config.DB.Unscoped().Delete(&rt)

	v, cleanup := createTestVoucher("TST-RT-PASS-001", "nominal", 25000)
	defer cleanup()
	// Explicitly set is_all_room_types = false (GORM skip zero-value bool dengan default:true)
	config.DB.Model(v).Update("is_all_room_types", false)

	config.DB.Create(&models.VoucherRoomTemplate{VoucherID: v.ID, RoomTemplateID: rt.ID})

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:           v.Code,
		CustomerID:     testCustomerID,
		Amount:         100000,
		UseType:        "booking",
		RoomTemplateID: rt.ID, // room yang diizinkan
	})
	if !resp.IsValid {
		t.Errorf("room type sesuai harus valid, got: %q", resp.Message)
	}
}

func TestValidateVoucher_RoomTypeRestriction_Fail(t *testing.T) {
	skipIfNoDB(t)

	rt := models.RoomTemplate{Name: "TEST RT Voucher Fail", CapacityMax: 4}
	config.DB.Create(&rt)
	defer config.DB.Unscoped().Delete(&rt)

	v, cleanup := createTestVoucher("TST-RT-FAIL-001", "nominal", 25000)
	defer cleanup()
	config.DB.Model(v).Update("is_all_room_types", false)

	config.DB.Create(&models.VoucherRoomTemplate{VoucherID: v.ID, RoomTemplateID: rt.ID})

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:           v.Code,
		CustomerID:     testCustomerID,
		Amount:         100000,
		UseType:        "booking",
		RoomTemplateID: 9999999, // room yang BUKAN di daftar
	})
	if resp.IsValid {
		t.Error("room type tidak sesuai harus invalid")
	}
}

// ── Nominal flat discount ─────────────────────────────────────

func TestValidateVoucher_NominalDiscount(t *testing.T) {
	skipIfNoDB(t)

	v, cleanup := createTestVoucher("TST-NOM-001", "nominal", 50000)
	defer cleanup()

	resp, _ := ValidateVoucher(dto.ValidateVoucherRequest{
		Code:       v.Code,
		CustomerID: testCustomerID,
		Amount:     200000,
		UseType:    "booking",
	})
	if !resp.IsValid {
		t.Fatalf("harus valid: %q", resp.Message)
	}
	if resp.DiscountAmount != 50000 {
		t.Errorf("discount nominal: want 50000, got %.0f", resp.DiscountAmount)
	}
}
