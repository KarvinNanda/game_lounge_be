package service

import (
	"encoding/json"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/customer/dto"

	"github.com/google/uuid"
)

func statsBooking(t *testing.T, customerID string, day int, status, method string, price int64) {
	t.Helper()
	b := &models.Booking{
		ID: uuid.NewString(), BookingCode: "T" + uuid.NewString()[:12], StoreID: uuid.NewString(),
		RoomID: uuid.NewString(), CustomerID: &customerID, CustomerName: "TEST", BookingDate: time.Now().AddDate(0, 0, day),
		StartTime: "12:00", EndTime: "13:00", DurationHours: 1, BasePrice: 1, Status: status, PaymentMethod: method,
	}
	if err := config.DB.Create(b).Error; err != nil {
		t.Fatal(err)
	}
	config.DB.Model(b).Update("total_price", price)
	t.Cleanup(func() { config.DB.Delete(&models.Booking{}, "id = ?", b.ID) })
}

// One customer: completed cash 50k (yesterday), upcoming cash 30k (tomorrow),
// cancelled cash 99k, play-credits booking yesterday, credit purchase 120k.
func TestCustomerStats(t *testing.T) {
	skipIfNoDB(t)
	cust := &models.Customer{ID: uuid.NewString(), Name: "TEST Stats " + uuid.NewString()[:6], Whatsapp: "0896" + uuid.NewString()[:8], Type: "member", Status: "active"}
	if err := config.DB.Create(cust).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.DB.Where("customer_id = ?", cust.ID).Delete(&models.CustomerPlayCredit{})
		config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", cust.ID)
	})
	statsBooking(t, cust.ID, -1, "completed", "cash", 50000)
	statsBooking(t, cust.ID, 1, "upcoming", "cash", 30000)
	statsBooking(t, cust.ID, -2, "cancelled", "cash", 99000)
	statsBooking(t, cust.ID, -1, "upcoming", "play_credits", 0) // stale status, but the day has passed
	pay := int64(120000)
	credit := &models.CustomerPlayCredit{ID: uuid.NewString(), CustomerID: cust.ID, PackageID: uuid.NewString(),
		TotalHours: 5, RemainingHours: 5, PurchasedAt: time.Now(), ExpiresAt: time.Now().AddDate(0, 1, 0), IsActive: true}
	if err := config.DB.Create(credit).Error; err != nil {
		t.Fatal(err)
	}
	config.DB.Model(credit).Update("payment_amount", pay)

	want := map[string]float64{"total_booking": 3, "total_visit": 2, "total_spent": 200000}
	check := func(where string, v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		var got map[string]any
		json.Unmarshal(b, &got)
		for k, w := range want {
			if got[k] != w {
				t.Errorf("%s: %s = %v, want %v", where, k, got[k], w)
			}
		}
		if got["name"] != cust.Name {
			t.Errorf("%s: customer fields must stay at the top level, got name %v", where, got["name"])
		}
	}

	list, _, err := GetAllCustomers(dto.CustomerListFilter{Search: cust.Name, Page: 1, PerPage: 10})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v, %d rows", err, len(list))
	}
	check("list", list[0])
	detail, err := GetCustomerByID(cust.ID)
	if err != nil {
		t.Fatal(err)
	}
	check("detail", detail)
}
