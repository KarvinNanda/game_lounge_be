package repository

import (
	"fmt"
	"sort"

	"game_lounge_be/config"
	"game_lounge_be/modules/sales/dto"
)

// ── Revenue Aggregation ───────────────────────────────────────────────────────

// BookingRevenue menghitung total pendapatan dari cash booking (non-cancelled).
func BookingRevenue(dateFrom, dateTo, storeID string) float64 {
	var result float64
	q := config.DB.Table("bookings").
		Select("COALESCE(SUM(total_price), 0)").
		Where("status != 'cancelled' AND payment_method = 'cash'").
		Where("booking_date BETWEEN ? AND ?", dateFrom, dateTo)
	if storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	q.Scan(&result)
	return result
}

// BookingCount menghitung jumlah cash booking (non-cancelled).
func BookingCount(dateFrom, dateTo, storeID string) int64 {
	var result int64
	q := config.DB.Table("bookings").
		Where("status != 'cancelled' AND payment_method = 'cash'").
		Where("booking_date BETWEEN ? AND ?", dateFrom, dateTo)
	if storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	q.Count(&result)
	return result
}

// CreditsRevenue menghitung total pendapatan dari penjualan play credits.
func CreditsRevenue(dateFrom, dateTo, storeID string) float64 {
	var result float64
	q := config.DB.Table("customer_play_credits").
		Select("COALESCE(SUM(payment_amount), 0)").
		Where("deleted_at IS NULL AND payment_amount IS NOT NULL").
		Where("DATE(purchased_at) BETWEEN ? AND ?", dateFrom, dateTo)
	if storeID != "" {
		q = q.Joins("JOIN play_credits_packages p ON p.id = customer_play_credits.package_id").
			Where(`p.apply_to_all_stores = true OR EXISTS (
				SELECT 1 FROM play_credits_package_stores pcps
				WHERE pcps.package_id = p.id AND pcps.store_id = ?
			)`, storeID)
	}
	q.Scan(&result)
	return result
}

// CreditsCount menghitung jumlah pembelian play credits.
func CreditsCount(dateFrom, dateTo, storeID string) int64 {
	var result int64
	q := config.DB.Table("customer_play_credits").
		Where("deleted_at IS NULL AND payment_amount IS NOT NULL").
		Where("DATE(purchased_at) BETWEEN ? AND ?", dateFrom, dateTo)
	if storeID != "" {
		q = q.Joins("JOIN play_credits_packages p ON p.id = customer_play_credits.package_id").
			Where(`p.apply_to_all_stores = true OR EXISTS (
				SELECT 1 FROM play_credits_package_stores pcps
				WHERE pcps.package_id = p.id AND pcps.store_id = ?
			)`, storeID)
	}
	q.Count(&result)
	return result
}

// ── Revenue by Branch ─────────────────────────────────────────────────────────

// RevenueByBranch menghitung pendapatan booking per cabang.
func RevenueByBranch(dateFrom, dateTo string) []dto.RevenueByBranch {
	var results []struct {
		StoreID   string
		StoreName string
		Revenue   float64
	}

	config.DB.Table("bookings b").
		Select("b.store_id, s.name as store_name, COALESCE(SUM(b.total_price), 0) as revenue").
		Joins("JOIN stores s ON s.id = b.store_id AND s.deleted_at IS NULL").
		Where("b.status != 'cancelled' AND b.booking_date BETWEEN ? AND ?", dateFrom, dateTo).
		Group("b.store_id, s.name").
		Order("revenue DESC").
		Scan(&results)

	var total float64
	for _, r := range results {
		total += r.Revenue
	}

	out := make([]dto.RevenueByBranch, 0, len(results))
	for _, r := range results {
		pct := 0.0
		if total > 0 {
			pct = r.Revenue / total * 100
		}
		out = append(out, dto.RevenueByBranch{
			StoreID:    r.StoreID,
			StoreName:  r.StoreName,
			Revenue:    r.Revenue,
			Percentage: pct,
		})
	}
	return out
}

// ── Revenue by Room Type ──────────────────────────────────────────────────────

// RevenueByRoomType menghitung pendapatan per tipe ruangan.
func RevenueByRoomType(dateFrom, dateTo, storeID string) []dto.RevenueByRoomType {
	var results []struct {
		RoomTemplateID uint
		Name           string
		Revenue        float64
	}

	q := config.DB.Table("bookings b").
		Select("rt.id as room_template_id, rt.name, COALESCE(SUM(b.total_price), 0) as revenue").
		Joins("JOIN store_rooms sr ON sr.id = b.room_id").
		Joins("JOIN room_templates rt ON rt.id = sr.room_template_id AND rt.deleted_at IS NULL").
		Where("b.status != 'cancelled' AND b.booking_date BETWEEN ? AND ?", dateFrom, dateTo)

	if storeID != "" {
		q = q.Where("b.store_id = ?", storeID)
	}

	q.Group("rt.id, rt.name").Order("revenue DESC").Scan(&results)

	var total float64
	for _, r := range results {
		total += r.Revenue
	}

	out := make([]dto.RevenueByRoomType, 0, len(results))
	for _, r := range results {
		pct := 0.0
		if total > 0 {
			pct = r.Revenue / total * 100
		}
		out = append(out, dto.RevenueByRoomType{
			RoomTemplateID: r.RoomTemplateID,
			Name:           r.Name,
			Revenue:        r.Revenue,
			Percentage:     pct,
		})
	}
	return out
}

// ── Sales Trend ───────────────────────────────────────────────────────────────

// SalesTrend mengambil data trend per granularitas (daily/weekly/monthly).
// Merge booking + credits ke dalam urutan kronologis.
func SalesTrend(dateFrom, dateTo, storeID, granularity string) []dto.TrendPoint {
	var bGroupExpr, bLabelExpr, cGroupExpr, cLabelExpr string

	switch granularity {
	case "weekly":
		bGroupExpr = "YEARWEEK(booking_date, 1)"
		bLabelExpr = "CONCAT('W', WEEK(booking_date, 1), ' ', YEAR(booking_date))"
		cGroupExpr = "YEARWEEK(purchased_at, 1)"
		cLabelExpr = "CONCAT('W', WEEK(purchased_at, 1), ' ', YEAR(purchased_at))"
	case "monthly":
		bGroupExpr = "DATE_FORMAT(booking_date, '%Y-%m')"
		bLabelExpr = "DATE_FORMAT(booking_date, '%b %Y')"
		cGroupExpr = "DATE_FORMAT(purchased_at, '%Y-%m')"
		cLabelExpr = "DATE_FORMAT(purchased_at, '%b %Y')"
	default: // daily
		bGroupExpr = "DATE(booking_date)"
		bLabelExpr = "DATE_FORMAT(booking_date, '%d %b %Y')"
		cGroupExpr = "DATE(purchased_at)"
		cLabelExpr = "DATE_FORMAT(purchased_at, '%d %b %Y')"
	}

	// Booking trend
	type trendRow struct {
		Label   string
		GroupBy string
		Revenue float64
	}
	var bookingTrend []trendRow

	bq := config.DB.Table("bookings").
		Select(fmt.Sprintf("%s as label, %s as group_by, COALESCE(SUM(total_price), 0) as revenue",
			bLabelExpr, bGroupExpr)).
		Where("status != 'cancelled' AND payment_method = 'cash'").
		Where("booking_date BETWEEN ? AND ?", dateFrom, dateTo)
	if storeID != "" {
		bq = bq.Where("store_id = ?", storeID)
	}
	bq.Group(bGroupExpr).Order(bGroupExpr).Scan(&bookingTrend)

	// Credits trend
	var creditsTrend []trendRow

	cq := config.DB.Table("customer_play_credits").
		Select(fmt.Sprintf("%s as label, %s as group_by, COALESCE(SUM(payment_amount), 0) as revenue",
			cLabelExpr, cGroupExpr)).
		Where("deleted_at IS NULL AND payment_amount IS NOT NULL").
		Where("DATE(purchased_at) BETWEEN ? AND ?", dateFrom, dateTo)
	if storeID != "" {
		cq = cq.Joins("JOIN play_credits_packages p ON p.id = customer_play_credits.package_id").
			Where(`p.apply_to_all_stores = true OR EXISTS (
				SELECT 1 FROM play_credits_package_stores pcps
				WHERE pcps.package_id = p.id AND pcps.store_id = ?
			)`, storeID)
	}
	cq.Group(cGroupExpr).Order(cGroupExpr).Scan(&creditsTrend)

	// Merge ke map[groupByKey] → TrendPoint, lalu sort secara kronologis
	type mergedEntry struct {
		key   string
		label string
		bRev  float64
		cRev  float64
	}
	entryMap := make(map[string]*mergedEntry)
	var keys []string

	for _, b := range bookingTrend {
		if _, ok := entryMap[b.GroupBy]; !ok {
			entryMap[b.GroupBy] = &mergedEntry{key: b.GroupBy, label: b.Label}
			keys = append(keys, b.GroupBy)
		}
		entryMap[b.GroupBy].bRev = b.Revenue
	}
	for _, c := range creditsTrend {
		if _, ok := entryMap[c.GroupBy]; !ok {
			entryMap[c.GroupBy] = &mergedEntry{key: c.GroupBy, label: c.Label}
			keys = append(keys, c.GroupBy)
		}
		if entryMap[c.GroupBy].label == "" {
			entryMap[c.GroupBy].label = c.Label
		}
		entryMap[c.GroupBy].cRev = c.Revenue
	}

	// Sort kunci secara leksikografis (YEARWEEK, DATE, DATE_FORMAT '%Y-%m' semua sortable)
	sort.Strings(keys)

	trend := make([]dto.TrendPoint, 0, len(keys))
	for _, k := range keys {
		e := entryMap[k]
		trend = append(trend, dto.TrendPoint{
			Label:          e.label,
			TotalRevenue:   e.bRev + e.cRev,
			BookingRevenue: e.bRev,
			CreditsRevenue: e.cRev,
		})
	}
	return trend
}

// ── Transaction List ──────────────────────────────────────────────────────────

// safeTime mengambil "HH:MM" dari string waktu "HH:MM" atau "HH:MM:SS".
func safeTime(t string) string {
	if len(t) >= 5 {
		return t[:5]
	}
	return t
}

// GetTransactions mengambil daftar transaksi (booking + credits) dengan filter.
func GetTransactions(dateFrom, dateTo, storeID, txType string, page, perPage int) ([]dto.TransactionItem, int64) {
	offset := (page - 1) * perPage
	items := make([]dto.TransactionItem, 0)
	var total int64

	includeBooking := txType == "" || txType == "all" || txType == "booking"
	includeCredits := txType == "" || txType == "all" || txType == "play_credits"

	if includeBooking {
		var bookings []struct {
			ID           string
			BookingDate  string
			StartTime    string
			CustomerName string
			RoomName     string
			StoreName    string
			TotalPrice   float64
			Status       string
		}

		bq := config.DB.Table("bookings b").
			Select("b.id, b.booking_date, b.start_time, b.customer_name, sr.name as room_name, s.name as store_name, b.total_price, b.status").
			Joins("JOIN store_rooms sr ON sr.id = b.room_id").
			Joins("JOIN stores s ON s.id = b.store_id").
			Where("b.status != 'cancelled' AND b.payment_method = 'cash'").
			Where("b.booking_date BETWEEN ? AND ?", dateFrom, dateTo)
		if storeID != "" {
			bq = bq.Where("b.store_id = ?", storeID)
		}
		bq.Order("b.booking_date DESC, b.start_time DESC").
			Limit(perPage).Offset(offset).Scan(&bookings)

		for _, b := range bookings {
			items = append(items, dto.TransactionItem{
				ID:           b.ID,
				Date:         b.BookingDate,
				Time:         safeTime(b.StartTime),
				Type:         "booking",
				CustomerName: b.CustomerName,
				Description:  b.RoomName,
				StoreName:    b.StoreName,
				Amount:       b.TotalPrice,
				Status:       b.Status,
			})
		}
	}

	if includeCredits {
		var credits []struct {
			ID            string
			PurchasedAt   string
			CustomerName  string
			PackageName   string
			PaymentAmount float64
		}

		cq := config.DB.Table("customer_play_credits cpc").
			Select("cpc.id, DATE_FORMAT(cpc.purchased_at, '%Y-%m-%d') as purchased_at, c.name as customer_name, p.name as package_name, cpc.payment_amount").
			Joins("JOIN customers c ON c.id = cpc.customer_id AND c.deleted_at IS NULL").
			Joins("JOIN play_credits_packages p ON p.id = cpc.package_id AND p.deleted_at IS NULL").
			Where("cpc.deleted_at IS NULL AND cpc.payment_amount IS NOT NULL").
			Where("DATE(cpc.purchased_at) BETWEEN ? AND ?", dateFrom, dateTo)
		if storeID != "" {
			cq = cq.Where(`p.apply_to_all_stores = true OR EXISTS (
				SELECT 1 FROM play_credits_package_stores pcps
				WHERE pcps.package_id = p.id AND pcps.store_id = ?
			)`, storeID)
		}
		cq.Order("cpc.purchased_at DESC").
			Limit(perPage).Offset(offset).Scan(&credits)

		for _, c := range credits {
			items = append(items, dto.TransactionItem{
				ID:           c.ID,
				Date:         c.PurchasedAt,
				Time:         "",
				Type:         "play_credits",
				CustomerName: c.CustomerName,
				Description:  c.PackageName,
				StoreName:    "Semua Cabang",
				Amount:       c.PaymentAmount,
				Status:       "completed",
			})
		}
	}

	// Hitung total untuk pagination
	switch {
	case txType == "booking":
		bq := config.DB.Table("bookings").
			Where("status != 'cancelled' AND payment_method = 'cash' AND booking_date BETWEEN ? AND ?", dateFrom, dateTo)
		if storeID != "" {
			bq = bq.Where("store_id = ?", storeID)
		}
		bq.Count(&total)
	case txType == "play_credits":
		cq := config.DB.Table("customer_play_credits").
			Where("deleted_at IS NULL AND payment_amount IS NOT NULL AND DATE(purchased_at) BETWEEN ? AND ?", dateFrom, dateTo)
		if storeID != "" {
			cq = cq.Joins("JOIN play_credits_packages p ON p.id = customer_play_credits.package_id").
				Where(`p.apply_to_all_stores = true OR EXISTS (
					SELECT 1 FROM play_credits_package_stores pcps
					WHERE pcps.package_id = p.id AND pcps.store_id = ?
				)`, storeID)
		}
		cq.Count(&total)
	default: // all
		var bc, cc int64
		config.DB.Table("bookings").
			Where("status != 'cancelled' AND payment_method = 'cash' AND booking_date BETWEEN ? AND ?", dateFrom, dateTo).
			Count(&bc)
		config.DB.Table("customer_play_credits").
			Where("deleted_at IS NULL AND payment_amount IS NOT NULL AND DATE(purchased_at) BETWEEN ? AND ?", dateFrom, dateTo).
			Count(&cc)
		total = bc + cc
	}

	return items, total
}
