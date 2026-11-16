package controller

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

type voucherItem struct {
	ID            string     `json:"voucher_id"`
	ReservedUntil *time.Time `json:"reserved_until"`
	HoldID        *string    `json:"hold_id"`
}

func newTestVoucher(t *testing.T) *models.Voucher {
	t.Helper()
	v := &models.Voucher{
		ID: uuid.NewString(), Name: "TEST", Code: "T" + uuid.NewString()[:12], Type: "booking",
		DiscountType: "percentage", DiscountValue: 10, StartDate: time.Now().AddDate(0, 0, -1),
		IsAllStores: true, IsAllRoomTypes: true, IsActive: true,
	}
	if err := config.DB.Create(v).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.DB.Where("voucher_id = ?", v.ID).Delete(&models.VoucherUsage{})
		config.DB.Unscoped().Delete(&models.Voucher{}, "id = ?", v.ID)
	})
	return v
}

// voucherFixture: member customer with three vouchers:
// free (never used), reserved (usage without booking + hold that expired 10 minutes ago),
// used (usage linked to a booking).
type voucherFixture struct {
	customerID           string
	free, reserved, used *models.Voucher
	hold                 *models.BookingHold
}

func newVoucherFixture(t *testing.T) voucherFixture {
	t.Helper()
	cust := &models.Customer{ID: uuid.NewString(), Name: "TEST", Whatsapp: "0", Type: "member", Status: "active"}
	if err := config.DB.Create(cust).Error; err != nil {
		t.Fatal(err)
	}
	f := voucherFixture{customerID: cust.ID, free: newTestVoucher(t), reserved: newTestVoucher(t), used: newTestVoucher(t)}

	f.hold = &models.BookingHold{
		ID: uuid.NewString(), CustomerID: cust.ID, StoreID: uuid.NewString(), RoomID: uuid.NewString(),
		RoomTemplateID: 1, BookingDate: time.Now().AddDate(0, 0, 3), StartTime: "12:00", EndTime: "13:00",
		DurationHours: 1, BasePrice: 1, TotalPrice: 1,
		PriceBreakdown: fmt.Sprintf(`{"voucher_id":%q}`, f.reserved.ID),
		ExpiresAt:      time.Now().Add(-10 * time.Minute).Truncate(time.Millisecond),
	}
	if err := config.DB.Create(f.hold).Error; err != nil {
		t.Fatal(err)
	}
	bookingID := uuid.NewString()
	for _, u := range []*models.VoucherUsage{
		{VoucherID: f.reserved.ID, CustomerID: cust.ID},
		{VoucherID: f.used.ID, CustomerID: cust.ID, BookingID: &bookingID},
	} {
		if err := config.DB.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		config.DB.Delete(&models.BookingHold{}, "id = ?", f.hold.ID)
		config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", cust.ID)
	})
	return f
}

func getVouchers(t *testing.T, customerID, query string) map[string]voucherItem {
	t.Helper()
	w := serve(http.MethodGet, "/vouchers/available"+query, customerID, "", GetMyVouchers)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var items []voucherItem
	decodeData(t, w.Body.Bytes(), &items)
	byID := map[string]voucherItem{}
	for _, it := range items {
		byID[it.ID] = it
	}
	return byID
}

func TestGetMyVouchers_Default_HidesReservedAndUsed(t *testing.T) {
	skipIfNoDB(t)
	f := newVoucherFixture(t)

	got := getVouchers(t, f.customerID, "")

	free, ok := got[f.free.ID]
	if !ok {
		t.Fatal("free voucher missing")
	}
	if free.ReservedUntil != nil || free.HoldID != nil {
		t.Errorf("free voucher must have null reserved_until/hold_id, got %+v", free)
	}
	if _, ok := got[f.reserved.ID]; ok {
		t.Error("reserved voucher must stay hidden without include_reserved")
	}
	if _, ok := got[f.used.ID]; ok {
		t.Error("used voucher must be hidden")
	}
}

func TestGetMyVouchers_IncludeReserved(t *testing.T) {
	skipIfNoDB(t)
	f := newVoucherFixture(t)

	got := getVouchers(t, f.customerID, "?include_reserved=true")

	if it, ok := got[f.free.ID]; !ok || it.ReservedUntil != nil {
		t.Errorf("free voucher: present=%v item=%+v", ok, it)
	}
	if _, ok := got[f.used.ID]; ok {
		t.Error("used voucher must stay hidden even with include_reserved")
	}
	res, ok := got[f.reserved.ID]
	if !ok {
		t.Fatal("reserved voucher missing with include_reserved=true")
	}
	if res.HoldID == nil || *res.HoldID != f.hold.ID {
		t.Errorf("hold_id = %v, want %s", res.HoldID, f.hold.ID)
	}
	want := f.hold.ExpiresAt.Add(time.Hour)
	if res.ReservedUntil == nil || !res.ReservedUntil.Equal(want) {
		t.Errorf("reserved_until = %v, want %v (hold expires_at + 1h grace)", res.ReservedUntil, want)
	}
}
