// Command seed mengisi database dev dengan data contoh (5-10 baris per tabel)
// supaya admin FE dan customer FE punya data untuk ditampilkan.
//
// Jalankan dari root project SETELAH `go run ./cmd/migrate`:
//
//	SEED_PASSWORD=... go run ./cmd/seed
//
// SEED_PASSWORD dipakai sebagai password semua staff & customer contoh.
// Seed menolak jalan jika tabel stores sudah berisi data (hindari duplikasi).
// Semua insert dalam 1 transaction: gagal di tengah = tidak ada yang tersimpan.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

const actor = "seed"

func ptr[T any](v T) *T { return &v }

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// roundK membulatkan ke Rp 1.000 terdekat.
func roundK(v float64) float64 { return math.Round(v/1000) * 1000 }

func main() {
	for _, p := range []string{".env", "../.env", "../../.env"} {
		if godotenv.Load(p) == nil {
			break
		}
	}
	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		log.Fatal("SEED_PASSWORD wajib diset (password untuk semua staff & customer contoh)")
	}

	config.InitDB()
	db := config.DB

	var storeCount int64
	db.Model(&models.Store{}).Count(&storeCount)
	if storeCount > 0 {
		log.Fatalf("Tabel stores sudah berisi %d baris, seed dibatalkan", storeCount)
	}

	hash, err := utils.HashPassword(password)
	must(err)

	err = db.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		seed(tx, hash)
		return nil
	})
	if err != nil {
		log.Fatalf("Seed gagal, semua perubahan di-rollback: %v", err)
	}
	log.Println("Seed selesai")
}

func seed(tx *gorm.DB, hash string) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	must(err)
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	day := func(offset int) time.Time { return today.AddDate(0, 0, offset) }
	a := ptr(actor)

	// ── Roles & permissions ──────────────────────────────────
	// Super Admin (is_system) dibuat oleh cmd/migrate. Nama permission diambil
	// dari game_lounge_fe (src/views/role/RoleView.vue) — backend tidak mengeceknya.
	var superAdmin models.Role
	if tx.Where("is_system = ? AND deleted_at IS NULL", true).First(&superAdmin).Error != nil {
		superAdmin = models.Role{Name: "Super Admin", IsSystem: true, CreatedBy: a}
		must(tx.Create(&superAdmin).Error)
	}
	rolePerms := []struct {
		name  string
		perms []string
	}{
		{"Store Manager", []string{"dashboard.view", "bookings.view", "bookings.create", "bookings.edit", "rooms.view", "rooms.edit", "pricing.view", "pricing.edit", "schedule.view", "schedule.edit", "customers.view", "customers.edit", "play_credits.view", "promotion.view", "orders_fnb.view"}},
		{"Cashier", []string{"dashboard.view", "bookings.view", "bookings.create", "customers.view", "customers.create", "play_credits.view", "play_credits.create", "orders_fnb.view", "orders_fnb.create"}},
		{"Operator", []string{"dashboard.view", "bookings.view", "rooms.view", "schedule.view", "orders_fnb.view", "orders_fnb.edit"}},
		{"Marketing", []string{"dashboard.view", "promotion.view", "promotion.create", "promotion.edit", "membership.view", "membership.create", "membership.edit", "customers.view"}},
		{"Viewer", []string{"dashboard.view", "bookings.view", "rooms.view", "pricing.view", "customers.view"}},
	}
	roles := map[string]models.Role{}
	for _, rp := range rolePerms {
		r := models.Role{Name: rp.name, CreatedBy: a}
		must(tx.Create(&r).Error)
		roles[rp.name] = r
		for _, p := range rp.perms {
			must(tx.Create(&models.RolePermission{RoleID: r.ID, Permission: p}).Error)
		}
	}

	// ── Stores ───────────────────────────────────────────────
	type storeDef struct {
		name, address, postal, status string
	}
	storeDefs := []storeDef{
		{"Quantum Jelambar", "Jl. Jelambar Utama No. 12, Grogol Petamburan, Jakarta Barat", "11460", "active"},
		{"Quantum Kelapa Gading", "Jl. Boulevard Raya Blok QA No. 5, Kelapa Gading, Jakarta Utara", "14240", "active"},
		{"Quantum PIK", "Ruko Golf Island Blok C No. 8, Pantai Indah Kapuk, Jakarta Utara", "14460", "active"},
		{"Quantum BSD", "Jl. BSD Raya Utama No. 21, Serpong, Tangerang Selatan", "15345", "active"},
		{"Quantum Bekasi", "Jl. Ahmad Yani No. 88, Bekasi Selatan, Bekasi", "17141", "active"},
		{"Quantum Depok", "Jl. Margonda Raya No. 140, Beji, Depok", "16424", "draft"},
	}
	var stores []models.Store
	for i, d := range storeDefs {
		s := models.Store{
			ID:          uuid.NewString(),
			Name:        d.name,
			Address:     d.address,
			Whatsapp:    ptr(fmt.Sprintf("08128800%04d", i+1)),
			PostalCode:  ptr(d.postal),
			Description: ptr("Gaming lounge PlayStation 5 dengan ruangan privat ber-AC, cocok untuk main bareng teman dan keluarga."),
			LinkGmaps:   ptr("https://maps.google.com/?q=" + d.name),
			Status:      d.status,
			CreatedBy:   a,
		}
		must(tx.Create(&s).Error)
		stores = append(stores, s)
	}
	active := stores[:5]

	for _, s := range stores {
		must(tx.Create(&[]models.StoreOperatingHour{
			{StoreID: s.ID, DayType: "weekday", OpenTime: "10:00", CloseTime: "23:00", IsActive: true, CreatedBy: a},
			{StoreID: s.ID, DayType: "weekend", OpenTime: "09:00", CloseTime: "02:00", IsActive: true, CreatedBy: a},
		}).Error)
	}

	// ── Holidays ─────────────────────────────────────────────
	globalDefs := []struct {
		date time.Time
		name string
	}{
		{time.Date(2026, 12, 24, 0, 0, 0, 0, loc), "Cuti Bersama Natal"},
		{time.Date(2026, 12, 25, 0, 0, 0, 0, loc), "Hari Raya Natal"},
		{time.Date(2027, 1, 1, 0, 0, 0, 0, loc), "Tahun Baru Masehi"},
		{time.Date(2027, 2, 6, 0, 0, 0, 0, loc), "Tahun Baru Imlek"},
		{time.Date(2027, 3, 10, 0, 0, 0, 0, loc), "Hari Raya Idul Fitri"},
		{time.Date(2027, 3, 11, 0, 0, 0, 0, loc), "Hari Raya Idul Fitri (Hari Kedua)"},
	}
	var globals []models.GlobalHolidaySchedule
	for _, g := range globalDefs {
		h := models.GlobalHolidaySchedule{Date: g.date, Name: g.name, OpenTime: "09:00", CloseTime: "02:00", CreatedBy: a}
		must(tx.Create(&h).Error)
		globals = append(globals, h)
	}
	// Store holiday: turunan dari 1 global holiday + 1 manual per store aktif.
	for i, s := range active {
		must(tx.Create(&[]models.StoreHolidaySchedule{
			{StoreID: s.ID, Date: globals[1].Date, OpenTime: "09:00", CloseTime: "02:00", GlobalHolidayID: &globals[1].ID, CreatedBy: a},
			{StoreID: s.ID, Date: day(20 + i), OpenTime: "12:00", CloseTime: "22:00", CreatedBy: a},
		}).Error)
	}

	// ── Facilities & room templates ──────────────────────────
	catNames := []string{"Console", "Display", "Audio", "Furniture", "Amenities"}
	cats := map[string]uint{}
	for _, n := range catNames {
		c := models.FacilityCategory{Name: n, CreatedBy: a}
		must(tx.Create(&c).Error)
		cats[n] = c.ID
	}
	facDefs := []struct{ cat, name, desc string }{
		{"Console", "PlayStation 5", "PS5 Disc Edition dengan 2 DualSense controller"},
		{"Console", "Nintendo Switch OLED", "Switch OLED dengan 4 Joy-Con"},
		{"Console", "Extra Controller", "Tambahan 2 controller"},
		{"Display", "TV 55 inch 4K", "Smart TV 55 inch 4K HDR"},
		{"Display", "TV 75 inch 4K", "Smart TV 75 inch 4K HDR 120Hz"},
		{"Audio", "Soundbar", "Soundbar 2.1 dengan subwoofer"},
		{"Audio", "Gaming Headset", "Wireless gaming headset"},
		{"Furniture", "Sofa Bed", "Sofa bed 3 dudukan"},
		{"Furniture", "Bean Bag", "Bean bag besar"},
		{"Amenities", "AC & Wi-Fi", "Ruangan ber-AC dan Wi-Fi 100 Mbps"},
	}
	facs := map[string]uint{}
	for _, f := range facDefs {
		m := models.Facility{CategoryID: cats[f.cat], Name: f.name, Description: ptr(f.desc), IsActive: true, CreatedBy: a}
		must(tx.Create(&m).Error)
		facs[f.name] = m.ID
	}

	tplDefs := []struct {
		name       string
		min, max   uint
		desc       string
		hourly     float64
		facilities []string
	}{
		{"Regular Room", 1, 4, "Ruangan standar untuk main santai bersama teman.", 25000, []string{"PlayStation 5", "TV 55 inch 4K", "Bean Bag", "AC & Wi-Fi"}},
		{"Couple Room", 1, 2, "Ruangan privat untuk berdua dengan sofa bed.", 30000, []string{"PlayStation 5", "TV 55 inch 4K", "Sofa Bed", "AC & Wi-Fi"}},
		{"VIP Room", 2, 6, "Ruangan luas dengan TV 75 inch dan soundbar.", 45000, []string{"PlayStation 5", "TV 75 inch 4K", "Soundbar", "Sofa Bed", "AC & Wi-Fi"}},
		{"VVIP Room", 4, 10, "Ruangan premium dengan PS5, Switch, dan audio lengkap.", 70000, []string{"PlayStation 5", "Nintendo Switch OLED", "TV 75 inch 4K", "Soundbar", "Gaming Headset", "Sofa Bed", "AC & Wi-Fi"}},
		{"Family Room", 3, 8, "Ruangan keluarga dengan Switch dan controller tambahan.", 60000, []string{"PlayStation 5", "Nintendo Switch OLED", "Extra Controller", "TV 75 inch 4K", "AC & Wi-Fi"}},
	}
	type tplInfo struct {
		ID     uint
		Name   string
		Hourly float64
	}
	var tpls []tplInfo
	for _, t := range tplDefs {
		m := models.RoomTemplate{Name: t.name, CapacityMin: t.min, CapacityMax: t.max, Description: ptr(t.desc), IsActive: true, CreatedBy: a}
		must(tx.Create(&m).Error)
		tpls = append(tpls, tplInfo{m.ID, t.name, t.hourly})
		for _, f := range t.facilities {
			must(tx.Create(&models.RoomTemplateFacility{RoomTemplateID: m.ID, FacilityID: facs[f]}).Error)
		}
	}

	// ── Store rooms: 2 Regular, 1 Couple, 1 VIP, 1 VVIP per store aktif ──
	roomsByStore := map[string][]models.StoreRoom{}
	roomPlan := []struct {
		tpl   int
		units int
	}{{0, 2}, {1, 1}, {2, 1}, {3, 1}}
	for _, s := range stores {
		for _, rp := range roomPlan {
			for u := 1; u <= rp.units; u++ {
				r := models.StoreRoom{
					ID:             uuid.NewString(),
					StoreID:        s.ID,
					RoomTemplateID: tpls[rp.tpl].ID,
					UnitNumber:     uint(u),
					Name:           fmt.Sprintf("%s %d", tpls[rp.tpl].Name, u),
					IsActive:       true,
					CreatedBy:      a,
				}
				must(tx.Create(&r).Error)
				roomsByStore[s.ID] = append(roomsByStore[s.ID], r)
			}
		}
	}

	// ── Pricing ──────────────────────────────────────────────
	pkgPrice := func(hourly float64, hours uint) float64 {
		disc := map[uint]float64{1: 1, 3: 0.9, 5: 0.85, 8: 0.8, 10: 0.75}[hours]
		return roundK(hourly * float64(hours) * disc)
	}
	for i, s := range stores {
		must(tx.Create(&models.StorePricing{StoreID: s.ID, IsHappyHourEnabled: true, IsMixedTimeEnabled: true, EdgeCase2h: "two_x_1h", EdgeCase4h: "3h_plus_1h", CreatedBy: a}).Error)
		must(tx.Create(&models.StoreEventPrice{StoreID: s.ID, PricePerDay: float64(6000000 + i*500000), CreatedBy: a}).Error)
		must(tx.Create(&models.StoreHappyHourSchedule{StoreID: s.ID, StartTime: "10:00", EndTime: "14:00", CreatedBy: a}).Error)
		for _, t := range tpls {
			must(tx.Create(&models.StoreHappyHourPrice{StoreID: s.ID, RoomTemplateID: t.ID, PricePerHour: roundK(t.Hourly * 0.7), CreatedBy: a}).Error)
			for _, h := range []uint{1, 3, 5, 8, 10} {
				must(tx.Create(&models.StorePackagePrice{StoreID: s.ID, RoomTemplateID: t.ID, DurationHours: h, Price: pkgPrice(t.Hourly, h), CreatedBy: a}).Error)
			}
		}
	}
	// Happy hour malam untuk 2 store (contoh lintas tengah malam).
	for _, s := range stores[:2] {
		must(tx.Create(&models.StoreHappyHourSchedule{StoreID: s.ID, StartTime: "23:00", EndTime: "02:00", CreatedBy: a}).Error)
	}
	flashDefs := []struct {
		store, tpl int
		name       string
		from, to   int
		tFrom, tTo string
		price      float64
		active     bool
	}{
		{0, 2, "Flash Sale VIP Weekday", 0, 14, "13:00", "17:00", 30000, true},
		{1, 0, "Happy Monday Regular", 0, 30, "10:00", "15:00", 15000, true},
		{2, 3, "VVIP Night Deal", 3, 10, "20:00", "23:00", 50000, true},
		{3, 1, "Couple Week", 7, 14, "15:00", "19:00", 20000, true},
		{4, 0, "Grand Opening Bekasi", -30, -1, "10:00", "22:00", 15000, false},
	}
	for _, f := range flashDefs {
		fs := models.StoreFlashSale{
			ID: uuid.NewString(), StoreID: stores[f.store].ID, RoomTemplateID: tpls[f.tpl].ID, Name: f.name,
			Description: ptr("Promo terbatas, harga spesial per jam."), PricePerHour: f.price,
			DateFrom: day(f.from), DateTo: day(f.to), TimeFrom: f.tFrom, TimeTo: f.tTo, IsActive: f.active, CreatedBy: a,
		}
		must(tx.Create(&fs).Error)
	}

	// ── Staff ────────────────────────────────────────────────
	staffDefs := []struct {
		username, email, role string
		allStores             bool
		storeIdx              []int
	}{
		{"manager.jelambar", "manager.jelambar@quantum.local", "Store Manager", false, []int{0}},
		{"manager.utara", "manager.utara@quantum.local", "Store Manager", false, []int{1, 2}},
		{"kasir.jelambar", "kasir.jelambar@quantum.local", "Cashier", false, []int{0}},
		{"kasir.bsd", "kasir.bsd@quantum.local", "Cashier", false, []int{3}},
		{"operator.bekasi", "operator.bekasi@quantum.local", "Operator", false, []int{4}},
		{"marketing", "marketing@quantum.local", "Marketing", true, nil},
		{"viewer", "viewer@quantum.local", "Viewer", true, nil},
	}
	for i, d := range staffDefs {
		s := models.Staff{
			ID: uuid.NewString(), RoleID: roles[d.role].ID, Username: d.username, Email: d.email,
			Phone: ptr(fmt.Sprintf("08137700%04d", i+1)), PasswordHash: hash, IsAllStores: d.allStores, CreatedBy: a,
		}
		must(tx.Create(&s).Error)
		for _, si := range d.storeIdx {
			must(tx.Create(&models.StaffStore{StaffID: s.ID, StoreID: stores[si].ID}).Error)
		}
	}
	// Token reset password lama (sudah dipakai / kedaluwarsa) — hanya contoh riwayat.
	var admin models.Staff
	hasAdmin := tx.Where("role_id = ?", superAdmin.ID).First(&admin).Error == nil
	for i := 0; hasAdmin && i < 5; i++ {
		t := models.PasswordResetToken{StaffID: admin.ID, Token: randomHex(32), IPAddress: fmt.Sprintf("10.0.0.%d", i+10), ExpiresAt: now.AddDate(0, 0, -i-1)}
		if i%2 == 0 {
			t.UsedAt = ptr(now.AddDate(0, 0, -i-1).Add(-10 * time.Minute))
		}
		must(tx.Create(&t).Error)
	}

	// ── Customers ────────────────────────────────────────────
	custDefs := []struct {
		name, email, gender, occupation, typ, status string
		dob                                          string
	}{
		{"Viking Pratama", "viking.pratama@example.com", "male", "Mahasiswa", "member", "active", "2002-04-12"},
		{"Siti Rahmawati", "siti.rahmawati@example.com", "female", "Karyawan Swasta", "member", "active", "1998-09-03"},
		{"Budi Santoso", "budi.santoso@example.com", "male", "Wiraswasta", "regular", "active", "1990-01-25"},
		{"Dewi Lestari", "dewi.lestari@example.com", "female", "Desainer", "member", "active", "1996-06-18"},
		{"Andi Wijaya", "andi.wijaya@example.com", "male", "Programmer", "member", "active", "1995-11-30"},
		{"Rina Kusuma", "rina.kusuma@example.com", "female", "Mahasiswa", "regular", "active", "2003-02-14"},
		{"Kevin Halim", "kevin.halim@example.com", "male", "Pelajar", "regular", "active", "2007-08-08"},
		{"Maya Putri", "maya.putri@example.com", "female", "Guru", "member", "active", "1993-12-01"},
		{"Fajar Nugroho", "fajar.nugroho@example.com", "male", "Karyawan Swasta", "regular", "inactive", "1992-03-22"},
		{"Lina Tanoto", "lina.tanoto@example.com", "female", "Dokter", "member", "active", "1989-07-07"},
	}
	var customers []models.Customer
	for i, d := range custDefs {
		dob, err := time.ParseInLocation("2006-01-02", d.dob, loc)
		must(err)
		c := models.Customer{
			ID: uuid.NewString(), Name: d.name, Whatsapp: fmt.Sprintf("08571234%04d", i+1), Email: ptr(d.email),
			PasswordHash: ptr(hash), DateOfBirth: &dob, Gender: ptr(d.gender), Occupation: ptr(d.occupation),
			Type: d.typ, Status: d.status, CreatedBy: a,
		}
		must(tx.Create(&c).Error)
		customers = append(customers, c)
	}
	for i := 0; i < 8; i++ {
		must(tx.Create(&models.CustomerFavoriteRoomType{CustomerID: customers[i].ID, RoomTemplateID: tpls[i%len(tpls)].ID}).Error)
	}
	for i := 0; i < 5; i++ {
		r := models.CustomerPasswordReset{CustomerID: customers[i].ID, Token: randomHex(32), IPAddress: fmt.Sprintf("192.168.1.%d", i+20), ExpiresAt: now.AddDate(0, 0, -i-1)}
		if i%2 == 1 {
			r.UsedAt = ptr(now.AddDate(0, 0, -i-1).Add(-5 * time.Minute))
		}
		must(tx.Create(&r).Error)
	}

	// ── Play credits ─────────────────────────────────────────
	pkgDefs := []struct {
		name     string
		hours    float64
		price    float64
		validity uint
		allStore bool
		stores   []int
	}{
		{"Starter 5 Jam", 5, 110000, 30, true, nil},
		{"Gamer 10 Jam", 10, 200000, 60, true, nil},
		{"Pro 20 Jam", 20, 380000, 90, true, nil},
		{"Ultimate 50 Jam", 50, 900000, 180, false, []int{0, 1, 2}},
		{"Weekend Pass 8 Jam", 8, 170000, 14, false, []int{3, 4}},
		{"Legacy 3 Jam", 3, 70000, 30, true, nil},
	}
	var pkgs []models.PlayCreditsPackage
	for i, d := range pkgDefs {
		p := models.PlayCreditsPackage{
			ID: uuid.NewString(), Name: d.name, TotalHours: d.hours, Price: d.price, ValidityDays: d.validity,
			Description: ptr(fmt.Sprintf("Paket %v jam bermain, berlaku %d hari sejak pembelian.", d.hours, d.validity)),
			IsActive:    i != 5, ApplyToAllStores: d.allStore, CreatedBy: a,
		}
		must(tx.Create(&p).Error)
		pkgs = append(pkgs, p)
		for _, si := range d.stores {
			must(tx.Create(&models.PlayCreditsPackageStore{PackageID: p.ID, StoreID: stores[si].ID}).Error)
		}
	}
	var credits []models.CustomerPlayCredit
	for i := 0; i < 8; i++ {
		p := pkgs[i%5]
		purchased := now.AddDate(0, 0, -(i*4 + 1))
		used := float64(i % 3)
		method := "manual"
		var invoice *string
		if i%2 == 1 {
			method = "xendit"
			invoice = ptr("inv-mock-" + randomHex(6))
		}
		c := models.CustomerPlayCredit{
			ID: uuid.NewString(), CustomerID: customers[i].ID, PackageID: p.ID, TotalHours: p.TotalHours,
			RemainingHours: math.Max(p.TotalHours-used, 0), PurchasedAt: purchased,
			ExpiresAt: purchased.AddDate(0, 0, int(p.ValidityDays)), PaymentMethod: method,
			PaymentAmount: ptr(p.Price), XenditInvoiceID: invoice, IsActive: true, CreatedBy: a,
		}
		must(tx.Create(&c).Error)
		credits = append(credits, c)
	}
	intentStatus := []string{"paid", "paid", "pending", "expired", "failed", "paid"}
	for i, st := range intentStatus {
		p := pkgs[i%5]
		created := now.Add(-time.Duration(i*30+5) * time.Hour)
		in := models.PlayCreditsPurchaseIntent{
			ID: uuid.NewString(), CustomerID: customers[i].ID, StoreID: active[i%5].ID, PackageID: p.ID, Amount: p.Price,
			Status: st, XenditInvoiceID: ptr("inv-mock-" + randomHex(6)), ExpiresAt: created.Add(24 * time.Hour),
		}
		if st == "pending" {
			in.ExpiresAt = now.Add(20 * time.Hour)
		}
		if st == "paid" {
			in.PaidAt = ptr(created.Add(10 * time.Minute))
		}
		must(tx.Create(&in).Error)
	}

	// ── Vouchers ─────────────────────────────────────────────
	vDefs := []struct {
		name, code, typ, dtype string
		value                  float64
		maxDisc, minPurchase   *float64
		start, end             int
		channel                string
		allStores, allRooms    bool
		active                 bool
	}{
		{"Diskon Member 10%", "MEMBER10", "booking", "percentage", 10, ptr(50000.0), ptr(50000.0), -10, 60, "all", true, true, true},
		{"Potongan 25rb", "HEMAT25", "booking", "nominal", 25000, nil, ptr(100000.0), -5, 30, "whatsapp", true, true, true},
		{"VIP Weekend 15%", "VIPWKND", "booking", "percentage", 15, ptr(75000.0), nil, 0, 45, "email", false, false, true},
		{"Credits Bonus 20rb", "CREDIT20", "play_credits", "nominal", 20000, nil, ptr(150000.0), -3, 30, "all", true, true, true},
		{"Grand Opening", "OPENING50", "both", "percentage", 50, ptr(100000.0), nil, -60, -30, "all", false, true, false},
		{"Birthday Treat", "BDAY2026", "both", "nominal", 50000, nil, nil, -20, 70, "email", true, true, true},
	}
	var vouchers []models.Voucher
	for _, d := range vDefs {
		v := models.Voucher{
			ID: uuid.NewString(), Name: d.name, Code: d.code, Description: ptr("Voucher khusus member Quantum Gaming Center."),
			Type: d.typ, DiscountType: d.dtype, DiscountValue: d.value, MaxDiscount: d.maxDisc, MinPurchase: d.minPurchase,
			StartDate: day(d.start), EndDate: ptr(day(d.end)), SendChannel: ptr(d.channel), TotalSent: 6,
			IsAllStores: d.allStores, IsAllRoomTypes: d.allRooms, IsActive: d.active, CreatedBy: a,
		}
		must(tx.Create(&v).Error)
		vouchers = append(vouchers, v)
	}
	// VIPWKND: hanya store 0-2, hanya VIP & VVIP. OPENING50: hanya Bekasi.
	for _, si := range []int{0, 1, 2} {
		must(tx.Create(&models.VoucherStore{VoucherID: vouchers[2].ID, StoreID: stores[si].ID}).Error)
	}
	must(tx.Create(&models.VoucherStore{VoucherID: vouchers[4].ID, StoreID: stores[4].ID}).Error)
	for _, ti := range []int{2, 3} {
		must(tx.Create(&models.VoucherRoomTemplate{VoucherID: vouchers[2].ID, RoomTemplateID: tpls[ti].ID}).Error)
	}

	// ── Bookings ─────────────────────────────────────────────
	// Status disimpan "completed"/"cancelled" untuk masa lalu, "upcoming" untuk sisanya;
	// booking_service.computeStatus menghitung ongoing/completed dari jam WIB saat dibaca.
	type bkDef struct {
		offset         int
		store, room    int
		cust           int // -1 = walk-in tanpa akun
		start, end     string
		hours          uint
		status, method string
		voucher        int // -1 = tanpa voucher
	}
	bkDefs := []bkDef{
		{-7, 0, 0, 0, "14:00", "17:00", 3, "completed", "cash", -1},
		{-5, 1, 3, 1, "19:00", "22:00", 3, "completed", "cash", 0},
		{-3, 2, 4, 2, "15:00", "20:00", 5, "completed", "cash", -1},
		{-2, 0, 2, 3, "16:00", "17:00", 1, "cancelled", "cash", -1},
		{-1, 3, 1, 4, "18:00", "21:00", 3, "completed", "play_credits", -1},
		{0, 0, 1, 0, "15:00", "18:00", 3, "upcoming", "cash", -1},
		{0, 4, 0, -1, "19:00", "22:00", 3, "upcoming", "cash", -1},
		{1, 1, 4, 7, "13:00", "18:00", 5, "upcoming", "cash", 1},
		{2, 2, 3, 9, "20:00", "23:00", 3, "upcoming", "cash", -1},
		{4, 0, 0, 5, "14:00", "15:00", 1, "upcoming", "cash", -1},
	}
	var seq uint
	var bookings []models.Booking
	for _, d := range bkDefs {
		s := active[d.store]
		room := roomsByStore[s.ID][d.room]
		var tpl tplInfo
		for _, t := range tpls {
			if t.ID == room.RoomTemplateID {
				tpl = t
			}
		}
		base := pkgPrice(tpl.Hourly, d.hours)
		bd, _ := json.Marshal([]map[string]any{{
			"time_range": d.start + " - " + d.end, "type": "Normal Hour",
			"description": fmt.Sprintf("Paket %d Jam", d.hours), "amount": base,
		}})
		seq++
		date := day(d.offset)
		b := models.Booking{
			ID: uuid.NewString(), BookingCode: fmt.Sprintf("BK-%s-%04d", date.Format("060102"), seq),
			StoreID: s.ID, RoomID: room.ID, BookingDate: date, StartTime: d.start, EndTime: d.end,
			DurationHours: float64(d.hours), PriceBreakdown: ptr(string(bd)), BasePrice: base, TotalPrice: base,
			PaymentMethod: d.method, Status: d.status, CreatedBy: a,
		}
		if d.cust >= 0 {
			c := customers[d.cust]
			b.CustomerID, b.CustomerName, b.CustomerWhatsapp, b.CustomerEmail = &c.ID, c.Name, &c.Whatsapp, c.Email
		} else {
			b.CustomerName, b.CustomerWhatsapp = "Walk-in Customer", ptr("081299990001")
		}
		if d.method == "play_credits" {
			b.PlayCreditID = &credits[d.cust].ID
			b.TotalPrice = 0
		}
		if d.voucher >= 0 {
			v := vouchers[d.voucher]
			disc := v.DiscountValue
			if v.DiscountType == "percentage" {
				disc = math.Min(roundK(base*v.DiscountValue/100), *v.MaxDiscount)
			}
			b.VoucherID, b.VoucherCode, b.DiscountAmount, b.TotalPrice = &v.ID, &v.Code, disc, base-disc
		}
		if d.status == "cancelled" {
			b.CancelReason, b.CancelledAt, b.CancelledBy = ptr("Customer berhalangan hadir"), ptr(date.Add(10*time.Hour)), a
		}
		must(tx.Create(&b).Error)
		bookings = append(bookings, b)
	}
	must(tx.Model(&models.BookingSequence{}).Where("id = 1").Update("last_sequence", seq).Error)

	// Voucher usage: 2 dari booking ber-voucher + 3 pemakaian MEMBER10 lain.
	usageDefs := []struct {
		v, c    int
		booking *string
		disc    float64
	}{
		{0, 1, &bookings[1].ID, bookings[1].DiscountAmount},
		{1, 7, &bookings[7].ID, bookings[7].DiscountAmount},
		{0, 3, nil, 15000},
		{0, 4, nil, 20000},
		{5, 9, nil, 50000},
	}
	usedCount := map[int]uint{}
	for i, u := range usageDefs {
		must(tx.Create(&models.VoucherUsage{
			VoucherID: vouchers[u.v].ID, CustomerID: customers[u.c].ID, BookingID: u.booking,
			DiscountAmount: u.disc, UsedAt: now.AddDate(0, 0, -(i + 1)),
		}).Error)
		usedCount[u.v]++
	}
	for vi, n := range usedCount {
		must(tx.Model(&models.Voucher{}).Where("id = ?", vouchers[vi].ID).Update("used_count", n).Error)
	}

	// Booking hold yang sudah kedaluwarsa (tidak memblokir jadwal).
	for i := 0; i < 5; i++ {
		s := active[i]
		room := roomsByStore[s.ID][0]
		created := now.Add(-time.Duration(i+2) * time.Hour)
		base := pkgPrice(tpls[0].Hourly, 3)
		must(tx.Create(&models.BookingHold{
			ID: uuid.NewString(), CustomerID: customers[i].ID, StoreID: s.ID, RoomID: room.ID, RoomTemplateID: room.RoomTemplateID,
			BookingDate: day(3), StartTime: "10:00", EndTime: "13:00", DurationHours: 3, BasePrice: base, TotalPrice: base,
			PriceBreakdown: `[{"time_range":"10:00 - 13:00","type":"Normal Hour","description":"Paket 3 Jam","amount":` + fmt.Sprint(base) + `}]`,
			PaymentMethod:  "xendit", XenditInvoiceID: ptr("inv-mock-" + randomHex(6)), ExpiresAt: created.Add(15 * time.Minute), CreatedAt: created,
		}).Error)
	}

	// ── Event bookings ───────────────────────────────────────
	evDefs := []struct {
		store, offset  int
		name, customer string
		start, end     string
		hours          float64
		durType, scope string
		tplIDs         *string
		status         string
	}{
		{0, -14, "Turnamen FIFA Komunitas", "Komunitas FIFA Jakarta", "10:00", "18:00", 8, "hourly", "full_venue", nil, "completed"},
		{1, 6, "Ulang Tahun Raka", "Ibu Sari", "13:00", "17:00", 4, "hourly", "per_room_type", ptr(fmt.Sprintf("[%d,%d]", tpls[2].ID, tpls[3].ID)), "upcoming"},
		{2, 9, "Gathering Kantor PT Maju", "PT Maju Bersama", "09:00", "02:00", 17, "full_day", "full_venue", nil, "upcoming"},
		{3, 12, "Tekken Tournament", "BSD Fighting Club", "12:00", "22:00", 10, "hourly", "full_venue", nil, "upcoming"},
		{4, -4, "Kelas Esports Pelajar", "SMA Negeri 1 Bekasi", "10:00", "14:00", 4, "hourly", "per_room_type", ptr(fmt.Sprintf("[%d]", tpls[0].ID)), "cancelled"},
	}
	for i, d := range evDefs {
		perDay := float64(6000000 + d.store*500000)
		total := roundK(perDay / 24 * d.hours)
		if d.durType == "full_day" {
			total = perDay
		}
		e := models.EventBooking{
			ID: uuid.NewString(), StoreID: active[d.store].ID, EventName: d.name, Description: ptr("Acara privat di " + active[d.store].Name),
			CustomerName: d.customer, CustomerWhatsapp: ptr(fmt.Sprintf("08111222%04d", i+1)), BookingDate: day(d.offset),
			StartTime: d.start, EndTime: d.end, DurationHours: d.hours, PricePerDay: perDay, TotalPrice: total, Status: d.status,
			DurationType: d.durType, BookingScope: d.scope, SelectedRoomTemplateIDs: d.tplIDs, CreatedBy: a,
		}
		if d.status == "cancelled" {
			e.CancelReason, e.CancelledAt, e.CancelledBy = ptr("Jadwal sekolah berubah"), ptr(now.AddDate(0, 0, -6)), a
		}
		if i == 1 {
			c := customers[1]
			e.CustomerID, e.IsCustomerBooking, e.PaymentStatus, e.PaymentMethod = &c.ID, true, ptr("paid"), ptr("xendit")
			e.XenditInvoiceID = ptr("inv-mock-" + randomHex(6))
		}
		must(tx.Create(&e).Error)
	}

	// ── FnB ──────────────────────────────────────────────────
	fnbCats := []string{"Minuman Dingin", "Kopi & Teh", "Makanan Berat", "Snack", "Dessert"}
	var fnbCatIDs []uint
	for i, n := range fnbCats {
		c := models.FnbCategory{Name: n, SortOrder: i + 1, IsActive: true, CreatedBy: a}
		must(tx.Create(&c).Error)
		fnbCatIDs = append(fnbCatIDs, c.ID)
	}
	itemDefs := []struct {
		cat   int
		name  string
		price float64
		avail bool
	}{
		{0, "Es Teh Manis", 8000, true}, {0, "Coca-Cola", 12000, true}, {0, "Air Mineral", 6000, true},
		{1, "Kopi Susu Gula Aren", 22000, true}, {1, "Matcha Latte", 25000, false},
		{2, "Nasi Goreng Spesial", 35000, true}, {2, "Indomie Goreng Telur", 18000, true},
		{3, "Kentang Goreng", 20000, true}, {3, "Chicken Wings", 30000, true},
		{4, "Es Krim Vanilla", 15000, true},
	}
	var items []models.FnbItem
	for i, d := range itemDefs {
		it := models.FnbItem{CategoryID: fnbCatIDs[d.cat], Name: d.name, Description: ptr(d.name + " — favorit pengunjung."), Price: d.price, IsAvailable: d.avail, IsActive: true, SortOrder: i + 1, CreatedBy: a}
		must(tx.Create(&it).Error)
		items = append(items, it)
	}
	orderDefs := []struct {
		booking int
		status  string
		lines   [][2]int // {item index, qty}
	}{
		{0, "delivered", [][2]int{{0, 2}, {7, 1}}},
		{1, "delivered", [][2]int{{3, 2}, {8, 1}}},
		{2, "delivered", [][2]int{{5, 1}, {1, 3}}},
		{4, "cancelled", [][2]int{{6, 2}}},
		{5, "preparing", [][2]int{{0, 1}, {6, 1}}},
		{7, "pending", [][2]int{{9, 2}, {2, 2}}},
	}
	for _, d := range orderDefs {
		b := bookings[d.booking]
		o := models.FnbOrder{ID: uuid.NewString(), BookingID: b.ID, CustomerID: *b.CustomerID, StoreID: b.StoreID, RoomID: b.RoomID, Status: d.status}
		for _, l := range d.lines {
			o.TotalAmount += items[l[0]].Price * float64(l[1])
		}
		must(tx.Create(&o).Error)
		for _, l := range d.lines {
			it := items[l[0]]
			must(tx.Create(&models.FnbOrderItem{OrderID: o.ID, ItemID: it.ID, ItemName: it.Name, Quantity: l[1], Price: it.Price}).Error)
		}
	}

	// ── Notification templates (3 key yang dipakai code) ─────
	vars := func(kv ...string) string {
		var out []map[string]string
		for i := 0; i < len(kv); i += 2 {
			out = append(out, map[string]string{"key": kv[i], "label": kv[i+1]})
		}
		b, _ := json.Marshal(out)
		return string(b)
	}
	must(tx.Create(&[]models.NotificationTemplate{
		{
			NotificationKey: "customer_welcome", Name: "Selamat Datang Customer", Description: ptr("Dikirim saat akun customer dibuat."),
			EmailSubject:       ptr("Selamat datang di Quantum Gaming Center"),
			EmailBody:          ptr("Halo {{nama_customer}},\n\nAkun kamu sudah dibuat.\nEmail: {{email}}\nPassword: {{password}}\n\nSegera ganti password setelah login."),
			WhatsappBody:       ptr("Halo {{nama_customer}}! Akun Quantum kamu sudah aktif. Email: {{email}}, Password: {{password}}"),
			AvailableVariables: vars("nama_customer", "Nama Customer", "email", "Email", "password", "Password"),
			IsEmailActive:      true, IsWhatsappActive: true, CreatedBy: a,
		},
		{
			NotificationKey: "voucher_notification", Name: "Notifikasi Voucher", Description: ptr("Dikirim saat voucher baru dibuat untuk member."),
			EmailSubject:       ptr("Voucher baru untukmu: {{nama_voucher}}"),
			EmailBody:          ptr("Halo {{nama_customer}},\n\nKamu dapat voucher {{nama_voucher}} dengan kode {{kode_voucher}}.\n{{deskripsi_voucher}}\nBerlaku sampai {{berlaku_sampai}}."),
			WhatsappBody:       ptr("Halo {{nama_customer}}! Pakai kode {{kode_voucher}} ({{nama_voucher}}), berlaku s/d {{berlaku_sampai}}."),
			AvailableVariables: vars("nama_customer", "Nama Customer", "nama_voucher", "Nama Voucher", "kode_voucher", "Kode Voucher", "berlaku_sampai", "Berlaku Sampai", "deskripsi_voucher", "Deskripsi Voucher"),
			IsEmailActive:      true, IsWhatsappActive: true, CreatedBy: a,
		},
		{
			NotificationKey: "booking_confirmation", Name: "Konfirmasi Booking", Description: ptr("Dikirim setelah booking berhasil."),
			EmailSubject:       ptr("Booking {{kode_booking}} dikonfirmasi"),
			EmailBody:          ptr("Halo {{nama_customer}},\n\nBooking {{kode_booking}} untuk {{nama_ruangan}} pada {{tanggal}} jam {{jam_mulai}}-{{jam_selesai}} ({{durasi}} jam) sudah dikonfirmasi.\nTotal: Rp {{total_harga}}"),
			WhatsappBody:       ptr("Booking {{kode_booking}} dikonfirmasi: {{nama_ruangan}}, {{tanggal}} {{jam_mulai}}-{{jam_selesai}}. Total Rp {{total_harga}}."),
			AvailableVariables: vars("nama_customer", "Nama Customer", "kode_booking", "Kode Booking", "nama_ruangan", "Nama Ruangan", "tanggal", "Tanggal", "jam_mulai", "Jam Mulai", "jam_selesai", "Jam Selesai", "durasi", "Durasi (jam)", "total_harga", "Total Harga"),
			IsEmailActive:      true, IsWhatsappActive: false, CreatedBy: a,
		},
	}).Error)

	// ── Banners (image_url wajib NOT NULL → string kosong dulu) ──
	bannerDefs := []struct{ title, sub string }{
		{"Grand Opening Quantum Bekasi", "Diskon 50% semua ruangan"},
		{"Happy Hour Setiap Hari", "Jam 10.00 - 14.00 lebih hemat 30%"},
		{"Member Lebih Untung", "Voucher eksklusif tiap bulan"},
		{"Play Credits", "Beli jam main lebih murah"},
		{"Turnamen Tekken BSD", "Daftar sekarang, slot terbatas"},
	}
	for i, d := range bannerDefs {
		b := models.Banner{Title: d.title, Subtitle: ptr(d.sub), Description: ptr(d.sub + "."), ImageURL: "", SortOrder: uint(i + 1), IsActive: i != 4, CreatedBy: a}
		must(tx.Create(&b).Error)
	}
}
