package service

import (
	"math"
	"time"

	"game_lounge_be/modules/sales/dto"
	"game_lounge_be/modules/sales/repository"
)

// jakartaLoc timezone WIB.
var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.UTC
	}
}

// GetPeriodDates menghitung range tanggal current dan previous berdasarkan period.
// Untuk period=custom tidak ada pembanding (prevFrom & prevTo dikembalikan kosong).
func GetPeriodDates(period, dateFrom, dateTo string) (curFrom, curTo, prevFrom, prevTo string) {
	now := time.Now().In(jakartaLoc)
	today := now.Format("2006-01-02")

	switch period {
	case "today":
		curFrom = today
		curTo = today
		prev := now.AddDate(0, 0, -1)
		prevFrom = prev.Format("2006-01-02")
		prevTo = prev.Format("2006-01-02")

	case "yesterday":
		yest := now.AddDate(0, 0, -1)
		curFrom = yest.Format("2006-01-02")
		curTo = yest.Format("2006-01-02")
		dayBefore := now.AddDate(0, 0, -2)
		prevFrom = dayBefore.Format("2006-01-02")
		prevTo = dayBefore.Format("2006-01-02")

	case "this_week":
		weekday := int(now.Weekday()) // 0=Sunday
		if weekday == 0 {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -(weekday - 1))
		curFrom = monday.Format("2006-01-02")
		curTo = today
		prevMonday := monday.AddDate(0, 0, -7)
		prevSunday := monday.AddDate(0, 0, -1)
		prevFrom = prevMonday.Format("2006-01-02")
		prevTo = prevSunday.Format("2006-01-02")

	case "this_month":
		firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, jakartaLoc)
		curFrom = firstDay.Format("2006-01-02")
		curTo = today
		prevFirst := firstDay.AddDate(0, -1, 0)
		prevLast := firstDay.AddDate(0, 0, -1)
		prevFrom = prevFirst.Format("2006-01-02")
		prevTo = prevLast.Format("2006-01-02")

	case "custom":
		curFrom = dateFrom
		curTo = dateTo
		prevFrom = ""
		prevTo = ""

	default: // fallback ke today
		curFrom = today
		curTo = today
		prev := now.AddDate(0, 0, -1)
		prevFrom = prev.Format("2006-01-02")
		prevTo = prev.Format("2006-01-02")
	}
	return
}

// changePercent menghitung persentase perubahan (current vs prev), 1 desimal.
// Kembalikan nil jika tidak ada data pembanding (prev == 0).
func changePercent(current, prev float64) *float64 {
	if prev == 0 {
		return nil
	}
	pct := math.Round((current-prev)/prev*1000) / 10
	return &pct
}

func changePercentInt(current, prev int64) *float64 {
	return changePercent(float64(current), float64(prev))
}

func pct1dp(num, total float64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(num/total*1000) / 10
}

// ── GetSalesSummary ───────────────────────────────────────────────────────────

// GetSalesSummary mengambil semua data untuk dashboard utama sales.
func GetSalesSummary(filter dto.SalesFilter) (*dto.SalesSummaryResponse, error) {
	curFrom, curTo, prevFrom, prevTo := GetPeriodDates(filter.Period, filter.DateFrom, filter.DateTo)

	// Current period
	bRev := repository.BookingRevenue(curFrom, curTo, filter.StoreID)
	cRev := repository.CreditsRevenue(curFrom, curTo, filter.StoreID)
	bCount := repository.BookingCount(curFrom, curTo, filter.StoreID)
	cCount := repository.CreditsCount(curFrom, curTo, filter.StoreID)

	totalRev := bRev + cRev
	totalTx := bCount + cCount
	avgTx := 0.0
	if totalTx > 0 {
		avgTx = totalRev / float64(totalTx)
	}

	stats := dto.SalesStats{
		TotalRevenue:      totalRev,
		BookingRevenue:    bRev,
		CreditsRevenue:    cRev,
		TotalTransactions: totalTx,
		AvgPerTransaction: avgTx,
	}

	// Previous period (hanya untuk period bukan custom)
	if prevFrom != "" && prevTo != "" {
		pBRev := repository.BookingRevenue(prevFrom, prevTo, filter.StoreID)
		pCRev := repository.CreditsRevenue(prevFrom, prevTo, filter.StoreID)
		pBCount := repository.BookingCount(prevFrom, prevTo, filter.StoreID)
		pCCount := repository.CreditsCount(prevFrom, prevTo, filter.StoreID)
		pTotalRev := pBRev + pCRev
		pTotalTx := pBCount + pCCount
		pAvg := 0.0
		if pTotalTx > 0 {
			pAvg = pTotalRev / float64(pTotalTx)
		}

		stats.TotalRevenueChange = changePercent(totalRev, pTotalRev)
		stats.BookingRevenueChange = changePercent(bRev, pBRev)
		stats.CreditsRevenueChange = changePercent(cRev, pCRev)
		stats.TransactionsChange = changePercentInt(totalTx, pTotalTx)
		stats.AvgChange = changePercent(avgTx, pAvg)
	}

	// Revenue by type
	revenueByType := []dto.RevenueByType{
		{
			Type:       "booking",
			Label:      "Booking",
			Revenue:    bRev,
			Percentage: pct1dp(bRev, totalRev),
		},
		{
			Type:       "play_credits",
			Label:      "Play Credits",
			Revenue:    cRev,
			Percentage: pct1dp(cRev, totalRev),
		},
	}

	revenueByBranch := repository.RevenueByBranch(curFrom, curTo)
	revenueByRoomType := repository.RevenueByRoomType(curFrom, curTo, filter.StoreID)

	// Pastikan slice tidak nil untuk JSON response
	if revenueByBranch == nil {
		revenueByBranch = []dto.RevenueByBranch{}
	}
	if revenueByRoomType == nil {
		revenueByRoomType = []dto.RevenueByRoomType{}
	}

	return &dto.SalesSummaryResponse{
		Period:            filter.Period,
		DateFrom:          curFrom,
		DateTo:            curTo,
		PrevDateFrom:      prevFrom,
		PrevDateTo:        prevTo,
		Stats:             stats,
		RevenueByType:     revenueByType,
		RevenueByBranch:   revenueByBranch,
		RevenueByRoomType: revenueByRoomType,
	}, nil
}

// ── GetSalesTrend ─────────────────────────────────────────────────────────────

// GetSalesTrend mengambil data trend untuk grafik garis.
func GetSalesTrend(filter dto.SalesFilter) ([]dto.TrendPoint, error) {
	curFrom, curTo, _, _ := GetPeriodDates(filter.Period, filter.DateFrom, filter.DateTo)
	trend := repository.SalesTrend(curFrom, curTo, filter.StoreID, filter.Granularity)
	if trend == nil {
		trend = []dto.TrendPoint{}
	}
	return trend, nil
}

// ── GetTransactions ───────────────────────────────────────────────────────────

// GetTransactions mengambil daftar transaksi untuk modal detail & export.
func GetTransactions(filter dto.SalesFilter) ([]dto.TransactionItem, int64, error) {
	curFrom, curTo, _, _ := GetPeriodDates(filter.Period, filter.DateFrom, filter.DateTo)
	items, total := repository.GetTransactions(curFrom, curTo, filter.StoreID, filter.Type, filter.Page, filter.PerPage)
	if items == nil {
		items = []dto.TransactionItem{}
	}
	return items, total, nil
}
