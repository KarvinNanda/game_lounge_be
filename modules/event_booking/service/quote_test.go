package service

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/event_booking/dto"

	"github.com/google/uuid"
)

// newPricedStore: store acak (jam operasional fallback 10:00–02:00) dengan harga event.
func newPricedStore(t *testing.T, pricePerDay float64) string {
	t.Helper()
	storeID := uuid.NewString()
	if err := config.DB.Create(&models.StoreEventPrice{StoreID: storeID, PricePerDay: pricePerDay}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.DB.Where("store_id = ?", storeID).Delete(&models.StoreEventPrice{})
		config.DB.Where("store_id = ?", storeID).Delete(&models.EventBooking{})
	})
	return storeID
}

const quoteDate = "2099-07-01"

func TestQuoteCustomerEvent_HargaDibulatkanSepertiCreate(t *testing.T) {
	skipIfNoDB(t)
	storeID := newPricedStore(t, 1_000_000)

	// 1 jam = 1.000.000/24 = 41.666,67 → dibulatkan Rp1.000 → 42.000
	q, err := QuoteCustomerEvent(storeID, quoteDate, "20:00", "21:00")
	if err != nil {
		t.Fatal(err)
	}
	if q.TotalPrice != 42_000 || q.DurationHours != 1 || q.PricePerDay != 1_000_000 {
		t.Errorf("got %+v", q)
	}
	if !q.Available {
		t.Error("slot kosong harus available")
	}
}

// Quote harus sama dengan harga yang benar-benar ditagih saat customer booking.
func TestQuoteCustomerEvent_SamaDenganCreate(t *testing.T) {
	skipIfNoDB(t)
	storeID := newPricedStore(t, 2_350_000)

	q, err := QuoteCustomerEvent(storeID, quoteDate, "21:30", "02:00")
	if err != nil {
		t.Fatal(err)
	}
	created, err := Create(dto.CreateEventBookingRequest{
		StoreID: storeID, EventName: "TEST", CustomerName: "TEST", BookingDate: quoteDate,
		StartTime: "21:30", EndTime: "02:00", DurationType: "hourly", BookingScope: "full_venue",
	}, "TEST")
	if err != nil {
		t.Fatal(err)
	}
	if q.TotalPrice != created.TotalPrice || q.DurationHours != created.DurationHours {
		t.Errorf("quote %v/%vh != create %v/%vh", q.TotalPrice, q.DurationHours, created.TotalPrice, created.DurationHours)
	}
}

func TestQuoteCustomerEvent_BentrokTidakAvailable(t *testing.T) {
	skipIfNoDB(t)
	storeID := newPricedStore(t, 1_000_000)
	other := &models.EventBooking{
		ID: uuid.NewString(), StoreID: storeID, EventName: "TEST", CustomerName: "TEST",
		BookingDate: time.Date(2099, 7, 1, 0, 0, 0, 0, time.Local), StartTime: "22:00", EndTime: "01:00",
		DurationHours: 3, PricePerDay: 1, TotalPrice: 1, Status: "upcoming",
	}
	if err := config.DB.Create(other).Error; err != nil {
		t.Fatal(err)
	}

	q, err := QuoteCustomerEvent(storeID, quoteDate, "00:00", "02:00")
	if err != nil {
		t.Fatal(err)
	}
	if q.Available {
		t.Error("bentrok dengan event 22:00–01:00, harus tidak available")
	}
}

func TestQuoteCustomerEvent_InputTidakValid(t *testing.T) {
	skipIfNoDB(t)
	storeID := newPricedStore(t, 1_000_000)
	cases := []struct{ name, store, date, start, end string }{
		{"jam sama", storeID, quoteDate, "20:00", "20:00"},
		{"format jam salah", storeID, quoteDate, "8pm", "22:00"},
		{"format tanggal salah", storeID, "01-07-2099", "20:00", "22:00"},
		{"tanggal lampau", storeID, "2020-01-01", "20:00", "22:00"},
		{"harga belum dikonfigurasi", uuid.NewString(), quoteDate, "20:00", "22:00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := QuoteCustomerEvent(tc.store, tc.date, tc.start, tc.end); err == nil {
				t.Error("want error")
			}
		})
	}
}
