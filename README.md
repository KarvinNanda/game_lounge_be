# Game Lounge Backend API

REST API untuk manajemen game lounge — mengelola store, staff, room template, fasilitas, pricing, booking, play credits, voucher, customer, dan laporan penjualan.

**Tech Stack:** Go · Gin · GORM · MySQL · JWT · SMTP

---

## Prasyarat

| Tool | Versi minimal |
|------|--------------|
| Go | 1.21+ |
| MySQL | 8.0+ |
| Git | — |

---

## Setup Project

### 1. Clone repository

```bash
git clone <repo-url>
cd game_lounge_be
```

### 2. Install dependencies

```bash
go mod tidy
```

### 3. Buat file `.env`

Buat file `.env` di **root folder project** (sejajar dengan folder `cmd/`):

```env
# App
APP_PORT=8080

# Database
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=game_lounge_db

# JWT
JWT_SECRET=your_super_secret_key
JWT_EXPIRED_HOURS=24

# SMTP (untuk kirim email password customer & notifikasi)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_FROM=noreply@gamelounge.com       # alamat pengirim (wajib)
SMTP_PASSWORD=your-smtp-app-password   # Gmail: gunakan App Password, bukan password biasa
SMTP_SENDER_NAME=Quantum Gaming Center # opsional — display name; fallback ke bagian sebelum @
```

> **Gmail App Password**: aktifkan 2-Step Verification di akun Google, lalu buat App Password di
> [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords).
> Jangan gunakan password Gmail biasa — akan ditolak SMTP.

### 4. Buat database MySQL

```sql
CREATE DATABASE game_lounge_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

> Migrasi tabel dilakukan otomatis oleh GORM saat server pertama kali dijalankan, atau import SQL schema secara manual sesuai kebutuhan.

### 5. Jalankan server

```bash
cd cmd
go run main.go
```

Server berjalan di: `http://localhost:8080`

---

## Struktur Folder

```
game_lounge_be/
│
├── .env                          # Konfigurasi environment (tidak di-commit)
├── .env.example                  # Template konfigurasi environment
│
├── cmd/
│   └── main.go                   # Entry point — inisialisasi DB, router, server
│
├── config/
│   ├── config.go                 # LoadEnv, GetEnv helper
│   └── database.go               # Koneksi GORM ke MySQL (config.DB)
│
├── middleware/
│   ├── auth.go                   # JWT auth middleware — set staff_id, staff_username, role_id, staff_role_id, is_system, permissions ke context
│   └── cors.go                   # CORS middleware (allow all origins)
│
├── models/                       # Definisi struct GORM (mapping ke tabel DB)
│   ├── role.go
│   ├── role_permission.go
│   ├── staff.go
│   ├── staff_store.go
│   ├── facility_category.go
│   ├── facility.go
│   ├── room_template.go
│   ├── room_template_facility.go
│   ├── store.go
│   ├── store_operating_hour.go
│   ├── store_holiday_schedule.go
│   ├── store_room.go
│   ├── pricing_config.go         # Konfigurasi harga per store
│   ├── happy_hour_schedule.go    # Jadwal happy hour
│   ├── happy_hour_price.go       # Harga happy hour per room type
│   ├── package_price.go          # Harga paket durasi
│   ├── flash_sale.go             # Flash sale (diskon sementara)
│   ├── play_credits_package.go   # Paket play credits
│   ├── play_credits_package_store.go  # Junction: package ↔ store
│   ├── customer_play_credit.go   # Kepemilikan play credits per customer
│   ├── voucher.go                # Voucher / promo
│   ├── voucher_store.go          # Junction: voucher ↔ store
│   ├── voucher_usage.go          # Riwayat pemakaian voucher
│   ├── customer.go               # Customer (member & walk-in)
│   ├── customer_favorite_room_type.go  # Preferensi room type customer
│   ├── booking.go                # Booking sesi bermain
│   ├── booking_sequence.go       # Counter auto-increment untuk kode booking
│   └── global_holiday_schedule.go  # Hari libur nasional/global (menonaktifkan happy hour)
│
├── modules/                      # Fitur-fitur API, masing-masing modul mandiri
│   │
│   ├── auth/
│   │   ├── controller/           # Login, Me, Logout
│   │   ├── dto/                  # LoginRequest, LoginResponse
│   │   ├── repository/           # FindStaffByUsername
│   │   └── service/              # Validasi password, generate JWT
│   │
│   ├── role/
│   │   ├── controller/           # CRUD role
│   │   ├── dto/                  # CreateRoleRequest, UpdateRoleRequest
│   │   ├── repository/           # FindAllRoles (+ Preload Permissions), SyncPermissions
│   │   └── service/              # RoleWithPermissions
│   │
│   ├── staff/
│   │   ├── controller/           # CRUD staff + reset-password (Super Admin only)
│   │   ├── dto/                  # CreateStaffRequest, UpdateStaffRequest, StaffFilter
│   │   ├── repository/           # FindAllStaffs (Preload Role, StaffStores.Store), UpdatePassword
│   │   └── service/              # Hash password, sync store assignments, ResetStaffPassword
│   │
│   ├── facility/
│   │   ├── controller/           # CRUD kategori + CRUD fasilitas
│   │   ├── dto/                  # CreateCategoryRequest, CreateFacilityRequest
│   │   ├── repository/           # FindAllFacilities (Preload Category)
│   │   └── service/              # GetAllCategories, GetAllFacilities
│   │
│   ├── room_template/
│   │   ├── controller/           # CRUD room template
│   │   ├── dto/                  # CreateRoomTemplateRequest (facility_ids, image_url)
│   │   ├── repository/           # Preload Facilities.Category, SyncFacilities
│   │   └── service/              # CreateRoomTemplate, UpdateRoomTemplate
│   │
│   ├── store/
│   │   ├── controller/           # CRUD store + GetOperatingHours
│   │   ├── dto/                  # CreateStoreRequest (operating_hours, holidays, rooms)
│   │   ├── repository/           # FindAllStores, FindStoreByID (full preload chain)
│   │   └── service/              # StoreListItem, UpsertOperatingHours/Holidays, GetEffectiveOperatingHours
│   │
│   ├── pricing/
│   │   ├── controller/           # CRUD config, happy hour, packages, flash sales, calculator
│   │   ├── dto/                  # PricingConfigRequest, CalculateRequest/Response
│   │   ├── repository/           # GetByStore, UpsertHappyHourPrices, UpsertPackagePrices, IsStoreHoliday
│   │   └── service/              # Calculate() — engine harga (flash sale overlap → HH/paket), isWeekdayForPricing
│   │
│   ├── play_credits/
│   │   ├── controller/           # CRUD paket + assign/adjust/delete member credits
│   │   ├── dto/                  # CreatePackageRequest, AssignCreditRequest, MemberCreditFilter
│   │   ├── repository/           # SyncPackageStores, DeductHours (negatif = refund)
│   │   └── service/              # MemberCreditResponse + computed fields (used_hours, days_left, dll)
│   │
│   ├── voucher/
│   │   ├── controller/           # CRUD voucher + generate-code + validate + customer-available + recipient-count
│   │   ├── dto/                  # CreateVoucherRequest (is_all_room_types, room_template_ids), ValidateVoucherRequest/Response
│   │   ├── repository/           # FindAvailableVouchersForCustomer, RedeemVoucher, SyncRoomTemplates, GetMembersForVoucher
│   │   └── service/              # VoucherWithStatus, sendVoucherNotification (targeted by room type, async)
│   │
│   ├── customer/
│   │   ├── controller/           # CRUD customer + update-notes + resend-password
│   │   ├── dto/                  # CreateCustomerRequest, UpdateCustomerRequest, CustomerListFilter
│   │   ├── repository/           # SyncFavoriteRoomTypes, FindCustomerByEmail/Whatsapp
│   │   └── service/              # Generate & hash password, kirim email (async)
│   │
│   ├── booking/
│   │   ├── controller/           # CRUD booking + dashboard + sessions-ending-soon + cancel/complete
│   │   ├── dto/                  # CreateBookingRequest, BookingFilter, BookingResponse
│   │   ├── repository/           # CheckOverlap, GenerateBookingCode, FindAvailableCredits, GetRoomNameByID
│   │   └── service/              # 13-step CreateBooking; DashboardData includes open/close time + holiday info
│   │
│   ├── sales/
│   │   ├── controller/           # GET summary, trend, transactions
│   │   ├── dto/                  # SalesFilter, SalesStats, TrendPoint, TransactionItem
│   │   ├── repository/           # BookingRevenue, CreditsRevenue, SalesTrend, GetTransactions
│   │   └── service/              # GetPeriodDates, changePercent, GetSalesSummary/Trend/Transactions
│   │
│   ├── global_holiday/
│   │   ├── controller/           # CRUD global holiday
│   │   ├── dto/                  # CreateGlobalHolidayRequest, UpdateGlobalHolidayRequest
│   │   ├── repository/           # FindAll, FindByID, FindByDate, Create, Update, SoftDelete
│   │   └── service/              # GetAll, Create, Update, Delete
│   │
│   ├── notification_template/
│   │   ├── controller/           # GET list, GET by key, PUT update, POST preview
│   │   ├── dto/                  # UpdateTemplateRequest, PreviewRequest
│   │   ├── repository/           # FindAll, FindByKey, UpdateByKey
│   │   └── service/              # GetRendered (dipakai customer/voucher/booking/play_credits service)
│   │
│   └── upload/
│       └── controller/           # POST /upload — simpan file ke ./assets/img/{folder}/
│
├── routes/
│   └── router.go                 # Daftar semua endpoint, grup public vs protected
│
├── utils/
│   ├── jwt.go                    # GenerateJWT, ValidateJWT, JWTClaims struct
│   ├── bcrypt.go                 # HashPassword, CheckPassword
│   ├── response.go               # ResponseSuccess, ResponseError, ResponseSuccessPaginate, Meta
│   ├── upload.go                 # SaveFileToAssets, InitAssetsDir, DeleteFile
│   ├── email.go                  # dispatch, SendEmail (plain text→HTML), SendHTMLEmail, SendCustomerPasswordEmail
│   ├── email_templates.go        # BuildBookingEmailHTML, BuildVoucherEmailHTML (fallback HTML berdesain)
│   ├── template_renderer.go      # RenderTemplate ({{var}} substitution), GetTemplateOrFallback
│   └── password_generator.go     # GeneratePasswordFromName (substitusi karakter + suffix #Gl)
│
├── assets/
│   └── img/                      # File gambar yang diupload (served sebagai static files)
│       ├── stores/
│       ├── facilities/
│       ├── room_templates/
│       ├── staffs/
│       ├── play_credits/
│       └── ...
│
├── uploads/                      # Legacy upload dir (backward compatibility)
│
├── go.mod
└── go.sum
```

---

## API Endpoints

Base URL: `/api/v1`

### 🔓 Public (tanpa auth)

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| POST | `/auth/login` | Login, mendapat JWT token |
| POST | `/upload` | Upload file gambar |

**Upload** — `multipart/form-data`:
- `file` *(required)* — file gambar (jpg, png, webp, svg, maks 2MB)
- `folder` *(optional)* — subfolder tujuan, misal `stores`, `facilities` (default: `img`)

Response:
```json
{ "status": "success", "data": { "url": "/assets/img/stores/uuid.jpg" } }
```

---

### 🔒 Protected (Bearer JWT)

#### Auth
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/auth/me` | Data staff yang sedang login |
| POST | `/auth/logout` | Logout |

#### Roles
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/roles` | List semua role (+ permissions) |
| POST | `/roles` | Buat role baru |
| GET | `/roles/:id` | Detail role |
| PUT | `/roles/:id` | Update role |
| DELETE | `/roles/:id` | Hapus role (soft delete) |

#### Staffs
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/staffs` | List staff (filter: search, role_id, store_id) |
| POST | `/staffs` | Buat staff baru |
| GET | `/staffs/:id` | Detail staff |
| PUT | `/staffs/:id` | Update staff |
| DELETE | `/staffs/:id` | Hapus staff (soft delete) |
| POST | `/staffs/:id/reset-password` | Reset password staff & kirim email (Super Admin only) |

#### Facility Categories
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/facility-categories` | List semua kategori |
| POST | `/facility-categories` | Buat kategori |
| PUT | `/facility-categories/:id` | Update kategori |
| DELETE | `/facility-categories/:id` | Hapus kategori |

#### Facilities
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/facilities` | List fasilitas (filter: search, category_id) |
| POST | `/facilities` | Buat fasilitas |
| GET | `/facilities/:id` | Detail fasilitas |
| PUT | `/facilities/:id` | Update fasilitas |
| DELETE | `/facilities/:id` | Hapus fasilitas |

#### Room Templates
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/room-templates` | List room template (+ facilities & category) |
| POST | `/room-templates` | Buat room template |
| GET | `/room-templates/:id` | Detail room template |
| PUT | `/room-templates/:id` | Update room template |
| DELETE | `/room-templates/:id` | Hapus room template |

#### Stores
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/stores` | List store (+ operating hours, holidays, room count) |
| POST | `/stores` | Buat store baru |
| GET | `/stores/operating-hours` | Jam operasional efektif store pada tanggal tertentu |
| GET | `/stores/:id` | Detail store (+ rooms → template → facilities → category) |
| PUT | `/stores/:id` | Update store |
| DELETE | `/stores/:id` | Hapus store (soft delete) |

**`GET /stores/operating-hours`** — query params: `store_id` *(required)*, `date` *(required, YYYY-MM-DD)*

Response:
```json
{
  "open_time": "10:00:00",
  "close_time": "22:00:00",
  "is_holiday": true,
  "holiday_name": "Hari Kemerdekaan Indonesia",
  "holiday_type": "global"
}
```
Priority: **Global Holiday** → **Store Holiday** → **Regular Hours** (weekday/weekend)

#### Pricing
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/pricing` | List semua pricing config |
| POST | `/pricing` | Buat pricing config baru |
| GET | `/pricing/:store_id` | Pricing config untuk store tertentu |
| PUT | `/pricing/:store_id` | Update pricing config |
| DELETE | `/pricing/:store_id` | Hapus pricing config |
| POST | `/pricing/calculate` | Hitung estimasi harga booking |
| POST | `/pricing/:store_id/happy-hour/schedules` | Tambah jadwal happy hour |
| DELETE | `/pricing/:store_id/happy-hour/schedules/:id` | Hapus jadwal happy hour |
| GET | `/pricing/:store_id/happy-hour/prices` | List harga happy hour per room type |
| PUT | `/pricing/:store_id/happy-hour/prices` | Upsert harga happy hour |
| GET | `/pricing/:store_id/packages` | List harga paket durasi |
| PUT | `/pricing/:store_id/packages` | Upsert harga paket durasi |
| DELETE | `/pricing/:store_id/packages/:id` | Hapus harga paket |
| GET | `/pricing/:store_id/flash-sales` | List flash sale |
| POST | `/pricing/:store_id/flash-sales` | Buat flash sale |
| PUT | `/pricing/flash-sales/:id` | Update flash sale |
| DELETE | `/pricing/flash-sales/:id` | Hapus flash sale |

**Calculate** — query params: `store_id`, `room_template_id`, `date`, `start_time`, `duration_hours`

#### Play Credits — Packages
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/play-credits/packages` | List semua paket |
| GET | `/play-credits/packages/active` | List paket aktif saja |
| POST | `/play-credits/packages` | Buat paket baru (`multipart/form-data`) |
| GET | `/play-credits/packages/:id` | Detail paket |
| PUT | `/play-credits/packages/:id` | Update paket (`multipart/form-data`) |
| DELETE | `/play-credits/packages/:id` | Hapus paket (soft delete) |

#### Play Credits — Member Credits
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/play-credits/members` | List kepemilikan credits (filter: customer_id, search, is_active) |
| POST | `/play-credits/members` | Assign credits ke customer |
| GET | `/play-credits/members/:id` | Detail member credit |
| PATCH | `/play-credits/members/:id/adjust` | Adjust jam credits (tambah / kurangi) |
| DELETE | `/play-credits/members/:id` | Hapus member credit (soft delete) |

#### Vouchers
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/vouchers` | List voucher (filter: search, type, status) |
| GET | `/vouchers/generate-code` | Generate kode voucher unik |
| GET | `/vouchers/customer-available` | Voucher tersedia untuk customer tertentu |
| GET | `/vouchers/recipient-count` | Preview jumlah member penerima berdasarkan room type |
| POST | `/vouchers/validate` | Validasi & cek kelayakan voucher |
| POST | `/vouchers` | Buat voucher baru |
| GET | `/vouchers/:id` | Detail voucher |
| PUT | `/vouchers/:id` | Update voucher |
| DELETE | `/vouchers/:id` | Hapus voucher (soft delete) |

**customer-available** — query params: `customer_id` *(required)*, `store_id` *(required)*, `type` (default: `booking`)

**recipient-count** — query params: `is_all_room_types` (`true`/`false`, default `true`), `room_template_ids[]` *(repeated uint)*

**Room type targeting pada voucher:**
| Field | Tipe | Keterangan |
|-------|------|------------|
| `is_all_room_types` | bool | `true` = berlaku semua room type; `false` = per room type |
| `room_template_ids` | `[]uint` | Daftar room template ID (wajib jika `is_all_room_types=false`) |

> Jika `is_all_room_types = false`, hanya member yang memiliki salah satu room type tersebut sebagai **favorit** yang akan menerima notifikasi. Validasi voucher saat booking juga akan cek room type yang dipilih.

#### Customers
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/customers` | List customer (filter: search, type, store_id) |
| POST | `/customers` | Buat customer baru |
| GET | `/customers/:id` | Detail customer |
| PUT | `/customers/:id` | Update customer |
| PATCH | `/customers/:id/notes` | Update catatan internal customer |
| DELETE | `/customers/:id` | Hapus customer (soft delete) |
| POST | `/customers/:id/resend-password` | Kirim ulang email password |

#### Bookings
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/bookings` | List booking (filter: store_id, date_from, date_to, status, search) |
| GET | `/bookings/dashboard` | Data dashboard booking per store (rooms + bookings + jam operasional efektif + info holiday) |
| GET | `/bookings/sessions-ending-soon` | Sesi yang akan berakhir dalam 5 menit ke depan |
| GET | `/bookings/available-credits` | Play credits yang tersedia untuk customer tertentu |
| POST | `/bookings` | Buat booking baru |
| GET | `/bookings/:id` | Detail booking |
| PATCH | `/bookings/:id/cancel` | Batalkan booking |
| PATCH | `/bookings/:id/complete` | Tandai booking selesai |

**available-credits** — query params: `customer_id` *(required)*, `store_id` *(required)*

#### Sales
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/sales/summary` | Dashboard utama — revenue, transaksi, breakdown per tipe/cabang/room |
| GET | `/sales/trend` | Data grafik tren penjualan |
| GET | `/sales/transactions` | Daftar transaksi (booking + play credits) dengan pagination |

#### Global Holidays
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/global-holidays` | List hari libur nasional (filter: year, search) |
| POST | `/global-holidays` | Tambah hari libur nasional |
| PUT | `/global-holidays/:id` | Update hari libur |
| DELETE | `/global-holidays/:id` | Hapus hari libur (soft delete) |

> Global holiday **menonaktifkan happy hour** di semua store pada tanggal tersebut — diprioritaskan sebelum store holiday dan pengecekan hari kerja.

**Body `POST /global-holidays`:**
```json
{
  "date": "2025-08-17",
  "name": "Hari Kemerdekaan Indonesia",
  "open_time": "10:00",
  "close_time": "22:00"
}
```

#### Notification Templates
| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/notification-templates` | List semua template notifikasi |
| GET | `/notification-templates/:key` | Detail template berdasarkan key |
| PUT | `/notification-templates/:key` | Update isi template (email & WhatsApp) |
| POST | `/notification-templates/preview` | Preview template dengan data contoh |

**notification_key** yang tersedia secara default:

| Key | Deskripsi | Variabel |
|-----|-----------|---------|
| `customer_welcome` | Email sambutan + password akun baru | `nama_customer`, `email`, `password` |
| `voucher_notification` | Notifikasi voucher ke seluruh member | `nama_customer`, `nama_voucher`, `kode_voucher`, `berlaku_sampai`, `deskripsi_voucher` |
| `booking_confirmation` | Konfirmasi booking ke customer | `nama_customer`, `kode_booking`, `nama_ruangan`, `tanggal`, `jam_mulai`, `jam_selesai`, `durasi`, `total_harga` |
| `play_credits_assigned` | Notifikasi assign play credits ke customer | `nama_customer`, `nama_paket`, `total_jam`, `sisa_jam`, `tanggal_kadaluwarsa` |

> Template disimpan di tabel `notification_templates`. Admin bisa mengubah konten email & WhatsApp tanpa deploy ulang. Variabel dinamis menggunakan format `{{nama_variabel}}`.

**Filter params (semua sales endpoint):**
| Param | Nilai | Default |
|-------|-------|---------|
| `period` | `today` \| `yesterday` \| `this_week` \| `this_month` \| `custom` | `today` |
| `store_id` | UUID store | — (semua store) |
| `date_from` | `YYYY-MM-DD` | — (wajib jika `period=custom`) |
| `date_to` | `YYYY-MM-DD` | — (wajib jika `period=custom`) |
| `granularity` | `daily` \| `weekly` \| `monthly` | `daily` (khusus `/trend`) |
| `type` | `all` \| `booking` \| `play_credits` | `all` (khusus `/transactions`) |
| `page` | integer | `1` (khusus `/transactions`) |
| `per_page` | integer | `20` (khusus `/transactions`) |

---

## Sistem Email

### Arsitektur pengiriman email

```
Trigger (create booking / create voucher / create customer)
  │
  ├─► ntService.GetRendered(key, vars)   ← ambil template dari DB
  │     │
  │     ├── Template ditemukan & aktif
  │     │     └─► utils.SendEmail()      ← plain text + generic card wrapper
  │     │
  │     └── Template tidak ada / error
  │           └─► utils.SendHTMLEmail()  ← HTML berdesain (BuildBookingEmailHTML / BuildVoucherEmailHTML)
  │
  └── [semua proses di goroutine — tidak memblokir HTTP response]
```

### Fungsi utilitas email

| Fungsi | Kapan dipakai |
|--------|--------------|
| `SendEmail(to, name, subject, plainText)` | Template DB (plain text dengan `\n`, auto-wrap card) |
| `SendHTMLEmail(to, name, subject, html)` | Template hardcode berdesain (booking & voucher fallback) |
| `SendCustomerPasswordEmail(to, name, pwd)` | Fallback welcome email jika DB template tidak ada |
| `BuildBookingEmailHTML(...)` | HTML email booking konfirmasi berdesain |
| `BuildVoucherEmailHTML(...)` | HTML email notifikasi voucher berdesain |

### Prioritas template

1. **DB template** (`notification_templates` table) — bisa dikustomisasi admin via `PUT /notification-templates/:key`
2. **Hardcode fallback** — HTML berdesain bawaan (booking & voucher), `SendCustomerPasswordEmail` (customer welcome)

### Konfigurasi SMTP (Gmail)

1. Aktifkan **2-Step Verification** di akun Google
2. Buat **App Password** di [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords)
3. Isi `SMTP_PASSWORD` dengan App Password tersebut (16 karakter tanpa spasi)

---

## Autentikasi

Semua endpoint protected menggunakan **Bearer JWT**.

```
Authorization: Bearer <token>
```

JWT berisi: `staff_id`, `username`, `role_id`, `is_system`, `permissions[]`

---

## Konvensi Audit Fields

Setiap record memiliki kolom:

| Kolom | Isi |
|-------|-----|
| `created_by` | Username staff yang membuat |
| `updated_by` | Username staff yang terakhir update |
| `deleted_by` | Username staff yang menghapus |
| `deleted_at` | Timestamp soft delete (null = aktif) |

> **Booking** tidak menggunakan soft delete — record bersifat permanen sebagai catatan keuangan.

---

## Logika Pricing Engine

Urutan penerapan harga saat `POST /pricing/calculate` atau membuat booking:

1. **Flash sale** — hitung overlap menit antara waktu booking dan window flash sale aktif; bagian yang overlap menggunakan `price_per_hour` flash sale (bukan diskon, langsung harga per jam)
2. **Sisa durasi** (non-flash) — cek `isWeekdayForPricing()`:
   - Global holiday → harga **normal** (happy hour nonaktif)
   - Store holiday → harga **normal** (happy hour nonaktif)
   - Senin–Kamis → cek jadwal **happy hour** (jika masuk → pakai harga HH, kalau tidak → cek package price → normal)
   - Jumat–Minggu → cek **package price** → normal
3. **Voucher** — diskon voucher diterapkan di atas `final_price` (hanya saat booking)

**Flash sale** menggunakan field `price_per_hour` (tarif per jam langsung, bukan diskon persen). Contoh: flash sale `price_per_hour = 50.000` aktif 19:00–21:00, booking 19:00–23:00 → 2 jam × 50.000 + 2 jam harga normal/HH.

---

## Alur Booking

```
POST /bookings
  ├── Cek overlap jadwal room
  ├── Hitung harga via pricing engine
  ├── Validasi voucher (opsional, khusus member)
  ├── Validasi play credits (opsional, khusus member)
  ├── Generate kode booking (BK-YYMMDD-XXXX)
  ├── Simpan record booking
  └── [goroutine] Potong play credits + redeem voucher + kirim email notifikasi
```

---

## Static Files

File yang diupload dapat diakses langsung via URL:

```
GET /assets/img/{folder}/{filename}
```

Contoh: `http://localhost:8080/assets/img/stores/abc123.jpg`
