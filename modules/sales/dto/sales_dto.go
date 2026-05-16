package dto

// SalesFilter parameter filter untuk semua endpoint sales.
// period: today | yesterday | this_week | this_month | custom
// granularity (untuk trend): daily | weekly | monthly
type SalesFilter struct {
	Period      string `form:"period,default=today"`      // today | yesterday | this_week | this_month | custom
	StoreID     string `form:"store_id"`
	Type        string `form:"type"`                      // all | booking | play_credits
	DateFrom    string `form:"date_from"`                 // wajib jika period=custom
	DateTo      string `form:"date_to"`                   // wajib jika period=custom
	Granularity string `form:"granularity,default=daily"` // daily | weekly | monthly
	Page        int    `form:"page,default=1"`
	PerPage     int    `form:"per_page,default=20"`
}

// SalesStats adalah ringkasan metrik utama.
type SalesStats struct {
	TotalRevenue      float64 `json:"total_revenue"`
	BookingRevenue    float64 `json:"booking_revenue"`
	CreditsRevenue    float64 `json:"credits_revenue"`
	TotalTransactions int64   `json:"total_transactions"`
	AvgPerTransaction float64 `json:"avg_per_transaction"`

	// Perubahan vs periode sebelumnya (null jika custom atau tidak ada data sebelumnya)
	TotalRevenueChange   *float64 `json:"total_revenue_change"`
	BookingRevenueChange *float64 `json:"booking_revenue_change"`
	CreditsRevenueChange *float64 `json:"credits_revenue_change"`
	TransactionsChange   *float64 `json:"transactions_change"`
	AvgChange            *float64 `json:"avg_change"`
}

// RevenueByType breakdown pendapatan booking vs credits.
type RevenueByType struct {
	Type       string  `json:"type"`       // "booking" | "play_credits"
	Label      string  `json:"label"`      // "Booking" | "Play Credits"
	Revenue    float64 `json:"revenue"`
	Percentage float64 `json:"percentage"` // 0–100, 1 desimal
}

// RevenueByBranch pendapatan per cabang.
type RevenueByBranch struct {
	StoreID    string  `json:"store_id"`
	StoreName  string  `json:"store_name"`
	Revenue    float64 `json:"revenue"`
	Percentage float64 `json:"percentage"`
}

// RevenueByRoomType pendapatan per tipe ruangan.
type RevenueByRoomType struct {
	RoomTemplateID uint    `json:"room_template_id"`
	Name           string  `json:"name"`
	Revenue        float64 `json:"revenue"`
	Percentage     float64 `json:"percentage"`
}

// TrendPoint satu titik data untuk grafik trend.
type TrendPoint struct {
	Label          string  `json:"label"`
	TotalRevenue   float64 `json:"total_revenue"`
	BookingRevenue float64 `json:"booking_revenue"`
	CreditsRevenue float64 `json:"credits_revenue"`
}

// TransactionItem satu baris transaksi untuk modal detail / export.
type TransactionItem struct {
	ID           string  `json:"id"`
	Date         string  `json:"date"`
	Time         string  `json:"time"`
	Type         string  `json:"type"`        // "booking" | "play_credits"
	CustomerName string  `json:"customer_name"`
	Description  string  `json:"description"` // nama room atau nama paket
	StoreName    string  `json:"store_name"`
	Amount       float64 `json:"amount"`
	Status       string  `json:"status"`
}

// SalesSummaryResponse response utama dashboard sales.
type SalesSummaryResponse struct {
	Period            string              `json:"period"`
	DateFrom          string              `json:"date_from"`
	DateTo            string              `json:"date_to"`
	PrevDateFrom      string              `json:"prev_date_from"`
	PrevDateTo        string              `json:"prev_date_to"`
	Stats             SalesStats          `json:"stats"`
	RevenueByType     []RevenueByType     `json:"revenue_by_type"`
	RevenueByBranch   []RevenueByBranch   `json:"revenue_by_branch"`
	RevenueByRoomType []RevenueByRoomType `json:"revenue_by_room_type"`
}
