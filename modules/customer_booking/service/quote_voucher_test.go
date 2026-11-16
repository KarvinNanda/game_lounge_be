package service

import (
	"fmt"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/customer_booking/repository"

	"github.com/google/uuid"
)

// quoteFixture: a member customer, a 15% voucher valid everywhere, and a slot
// that quotes successfully on an active seed store (skips if the DB has none).
type quoteFixture struct {
	customerID, voucherID, storeID, date string
	templateID                           uint
	slots                                []string
}

func newQuoteFixture(t *testing.T) quoteFixture {
	t.Helper()
	var f quoteFixture

	var stores []models.Store
	config.DB.Where("status = 'active' AND deleted_at IS NULL").Find(&stores)
	found := false
	for day := 7; day < 21 && !found; day++ {
		date := time.Now().AddDate(0, 0, day).Format("2006-01-02")
		for _, s := range stores {
			for _, tmpl := range []uint{8, 9, 10, 11} {
				for _, h := range []int{14, 15, 16} {
					slots := []string{fmt.Sprintf("%02d:00", h), fmt.Sprintf("%02d:00", h+1)}
					if q, err := quoteSlots(s.ID, tmpl, date, slots); err == nil && q.Available && q.Price.FinalPrice > 0 {
						f.storeID, f.templateID, f.date, f.slots, found = s.ID, tmpl, date, slots, true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
	}
	if !found {
		t.Skip("skip: no active store with a quotable slot in the DB")
	}

	cust := &models.Customer{ID: uuid.NewString(), Name: "TEST", Whatsapp: "0897" + uuid.NewString()[:8], Type: "member", Status: "active"}
	if err := config.DB.Create(cust).Error; err != nil {
		t.Fatal(err)
	}
	v := &models.Voucher{
		ID: uuid.NewString(), Name: "TEST 15", Code: "T" + uuid.NewString()[:12], Type: "booking",
		DiscountType: "percentage", DiscountValue: 15, StartDate: time.Now().AddDate(0, 0, -1),
		IsAllStores: true, IsAllRoomTypes: true, IsActive: true,
	}
	if err := config.DB.Create(v).Error; err != nil {
		t.Fatal(err)
	}
	f.customerID, f.voucherID = cust.ID, v.ID
	t.Cleanup(func() {
		var holds []models.BookingHold
		config.DB.Where("customer_id = ?", cust.ID).Find(&holds)
		for i := range holds {
			_ = repository.ReleaseHold(&holds[i])
		}
		config.DB.Where("voucher_id = ?", v.ID).Delete(&models.VoucherUsage{})
		config.DB.Unscoped().Delete(&models.Voucher{}, "id = ?", v.ID)
		config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", cust.ID)
	})
	return f
}

func TestQuoteBookingForCustomer_MatchesInitiateExactly(t *testing.T) {
	skipIfNoDB(t)
	f := newQuoteFixture(t)

	quote, err := QuoteBookingForCustomer(f.customerID, f.storeID, f.templateID, f.date, f.slots, f.voucherID)
	if err != nil {
		t.Fatal(err)
	}
	var usages int64
	config.DB.Model(&models.VoucherUsage{}).Where("voucher_id = ?", f.voucherID).Count(&usages)
	if usages != 0 {
		t.Fatal("a quote must not reserve the voucher")
	}

	init, err := InitiateBooking(InitiateBookingRequest{
		StoreID: f.storeID, RoomTemplateID: f.templateID, BookingDate: f.date,
		SelectedSlots: f.slots, PaymentMethod: "xendit", VoucherID: f.voucherID,
	}, f.customerID)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range [][2]string{{"total_price", "base_price"}, {"discount_amount", "discount_amount"}, {"final_price", "total_price"}} {
		if fmt.Sprint(quote[k[0]]) != fmt.Sprint(init[k[1]]) {
			t.Errorf("quote %s = %v, initiate %s = %v", k[0], quote[k[0]], k[1], init[k[1]])
		}
	}
	if fmt.Sprint(quote["discount_amount"]) == "0" {
		t.Error("15% voucher must give a discount")
	}
}

func TestQuoteBookingForCustomer_WithoutVoucher(t *testing.T) {
	skipIfNoDB(t)
	f := newQuoteFixture(t)
	q, err := QuoteBookingForCustomer(f.customerID, f.storeID, f.templateID, f.date, f.slots, "")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(q["discount_amount"]) != "0" || fmt.Sprint(q["final_price"]) != fmt.Sprint(q["total_price"]) {
		t.Errorf("no voucher: discount 0 and final = total, got %v / %v / %v", q["discount_amount"], q["final_price"], q["total_price"])
	}
}

func TestQuoteBookingForCustomer_InvalidVoucherSameErrorAsInitiate(t *testing.T) {
	skipIfNoDB(t)
	f := newQuoteFixture(t)
	_, qErr := QuoteBookingForCustomer(f.customerID, f.storeID, f.templateID, f.date, f.slots, uuid.NewString())
	_, iErr := InitiateBooking(InitiateBookingRequest{
		StoreID: f.storeID, RoomTemplateID: f.templateID, BookingDate: f.date,
		SelectedSlots: f.slots, PaymentMethod: "xendit", VoucherID: uuid.NewString(),
	}, f.customerID)
	if qErr == nil || iErr == nil || qErr.Error() != iErr.Error() {
		t.Errorf("want the same error, quote %v, initiate %v", qErr, iErr)
	}
}
