# Game Lounge Backend API

REST API for game lounge management: stores, staff, room templates, facilities, pricing, bookings, play credits, vouchers, customers, and sales reports.

**Tech Stack:** Go · Gin · GORM · MySQL · JWT · SMTP

**Main features:** Auth · Roles & Staff · Store & Room · Pricing Engine (HH/Package/Flash Sale) · Booking · Event Booking · Play Credits · Voucher · Customer · Sales Report · Admin Recovery (forgot password)

---

## Prerequisites

| Tool | Minimum version |
|------|-----------------|
| Go | 1.21+ |
| MySQL | 8.0+ |
| Git | — |

---

## Project Setup

### 1. Clone the repository

```bash
git clone <repo-url>
cd game_lounge_be
```

### 2. Install dependencies

```bash
go mod tidy
```

### 3. Create the `.env` file

Create `.env` in the **project root** (next to the `cmd/` folder):

```env
# App
APP_PORT=8080
COOKIE_SECURE=false          # local dev over http; production: remove it (default true)
# APP_ENV=production         # production: the server refuses to start if security config is empty (see .env.example)

# Database
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=game_lounge_db

# JWT
JWT_SECRET=replace_with_output_of_openssl_rand_hex_32   # at least 32 characters; the server rejects short secrets
JWT_EXPIRED_HOURS=24

# SMTP (customer password emails & notifications)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_FROM=noreply@gamelounge.com       # sender address (required)
SMTP_PASSWORD=your-smtp-app-password   # Gmail: use an App Password, not the normal password
SMTP_SENDER_NAME=Quantum Gaming Center # optional display name; falls back to the part before @
```

> **Gmail App Password**: enable 2-Step Verification on the Google account, then create an App Password at
> [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords).
> Do not use the normal Gmail password. SMTP will reject it.

### 4. Create the MySQL database

```sql
CREATE DATABASE game_lounge_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

Create the tables (from the project root). The server does **not** run migrations automatically:

```bash
go run ./cmd/migrate                      # create/update tables from models
SEED_PASSWORD=... go run ./cmd/seed       # optional: sample dev data (5-10 rows per table)
```

First super admin: set `SEED_ADMIN_USERNAME`, `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD` when running `cmd/migrate`.
The full DDL is in `database/schema.sql`.

> `cmd/migrate` is for dev only. Do not run it against production: AutoMigrate can run `ALTER` statements you do not need. Production schema changes are done with explicit SQL.

### 5. Run the server

```bash
cd cmd
go run main.go
```

The server runs at: `http://localhost:8080`

---

## Folder Structure

```
game_lounge_be/
│
├── .env                          # Environment config (not committed)
├── .env.example                  # Environment config template
│
├── cmd/
│   └── main.go                   # Entry point: init DB, router, server
│
├── config/
│   ├── config.go                 # LoadEnv, GetEnv helper
│   └── database.go               # GORM connection to MySQL (config.DB)
│
├── middleware/
│   ├── auth.go                   # JWT auth middleware: sets staff_id, staff_username, role_id, staff_role_id, is_system, permissions in the context
│   └── cors.go                   # CORS middleware (allow all origins)
│
├── models/                       # GORM struct definitions (mapped to DB tables)
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
│   ├── store_pricing.go          # Pricing config per store
│   ├── store_happy_hour_schedule.go  # Happy hour schedule
│   ├── store_happy_hour_price.go     # Happy hour price per room type
│   ├── store_package_price.go        # Duration package price
│   ├── store_flash_sale.go           # Flash sale (temporary price)
│   ├── store_event_price.go          # Event price per day per store
│   ├── play_credits_package.go   # Play credits package
│   ├── play_credits_package_store.go  # Junction: package ↔ store
│   ├── customer_play_credit.go   # Play credits owned per customer
│   ├── voucher.go                # Voucher / promo
│   ├── voucher_store.go          # Junction: voucher ↔ store
│   ├── voucher_room_template.go  # Junction: voucher ↔ room template (room type targeting)
│   ├── voucher_usage.go          # Voucher usage history
│   ├── customer.go               # Customer (member & walk-in)
│   ├── customer_favorite_room_type.go  # Customer room type preferences
│   ├── booking.go                # Play session booking per room
│   ├── booking_sequence.go       # Global counter for booking codes (BK- / EV-)
│   ├── event_booking.go          # Whole-venue booking for events (birthdays, corporate, etc.)
│   ├── global_holiday_schedule.go  # National/global holidays (disable happy hour)
│   ├── notification_template.go  # Customizable email & WhatsApp templates
│   └── password_reset_token.go   # One-time token for super admin password reset
│
├── modules/                      # API features; each module is self-contained
│   │
│   ├── auth/
│   │   ├── controller/           # Login, Me, Logout
│   │   ├── dto/                  # LoginRequest, LoginResponse
│   │   ├── repository/           # FindStaffByUsername
│   │   └── service/              # Password check, generate JWT
│   │
│   ├── role/
│   │   ├── controller/           # Role CRUD
│   │   ├── dto/                  # CreateRoleRequest, UpdateRoleRequest
│   │   ├── repository/           # FindAllRoles (+ Preload Permissions), SyncPermissions
│   │   └── service/              # RoleWithPermissions
│   │
│   ├── staff/
│   │   ├── controller/           # Staff CRUD + reset-password (Super Admin only)
│   │   ├── dto/                  # CreateStaffRequest, UpdateStaffRequest, StaffFilter
│   │   ├── repository/           # FindAllStaffs (Preload Role, StaffStores.Store), UpdatePassword
│   │   └── service/              # Hash password, sync store assignments, ResetStaffPassword
│   │
│   ├── facility/
│   │   ├── controller/           # Category CRUD + facility CRUD
│   │   ├── dto/                  # CreateCategoryRequest, CreateFacilityRequest
│   │   ├── repository/           # FindAllFacilities (Preload Category)
│   │   └── service/              # GetAllCategories, GetAllFacilities
│   │
│   ├── room_template/
│   │   ├── controller/           # Room template CRUD
│   │   ├── dto/                  # CreateRoomTemplateRequest (facility_ids, image_url)
│   │   ├── repository/           # Preload Facilities.Category, SyncFacilities
│   │   └── service/              # CreateRoomTemplate, UpdateRoomTemplate
│   │
│   ├── store/
│   │   ├── controller/           # Store CRUD + GetOperatingHours
│   │   ├── dto/                  # CreateStoreRequest (operating_hours, holidays, rooms)
│   │   ├── repository/           # FindAllStores, FindStoreByID (full preload chain)
│   │   └── service/              # StoreListItem, UpsertOperatingHours/Holidays, GetEffectiveOperatingHours
│   │
│   ├── pricing/
│   │   ├── controller/           # Config CRUD, happy hour, packages, flash sales, calculator
│   │   ├── dto/                  # PricingConfigRequest, CalculateRequest/Response
│   │   ├── repository/           # GetByStore, UpsertHappyHourPrices, UpsertPackagePrices, IsStoreHoliday
│   │   └── service/              # Calculate(): pricing engine (flash sale overlap → HH/package), isWeekdayForPricing
│   │
│   ├── play_credits/
│   │   ├── controller/           # Package CRUD + assign/adjust/delete member credits
│   │   ├── dto/                  # CreatePackageRequest, AssignCreditRequest, MemberCreditFilter
│   │   ├── repository/           # SyncPackageStores, DeductHours (negative = refund)
│   │   └── service/              # MemberCreditResponse + computed fields (used_hours, days_left, etc.)
│   │
│   ├── voucher/
│   │   ├── controller/           # Voucher CRUD + generate-code + validate + customer-available + recipient-count
│   │   ├── dto/                  # CreateVoucherRequest (is_all_room_types, room_template_ids), ValidateVoucherRequest/Response
│   │   ├── repository/           # FindAvailableVouchersForCustomer, RedeemVoucher, SyncRoomTemplates, GetMembersForVoucher
│   │   └── service/              # VoucherWithStatus, sendVoucherNotification (targeted by room type, async)
│   │
│   ├── customer/
│   │   ├── controller/           # Customer CRUD + update-notes + resend-password
│   │   ├── dto/                  # CreateCustomerRequest, UpdateCustomerRequest, CustomerListFilter
│   │   ├── repository/           # SyncFavoriteRoomTypes, FindCustomerByEmail/Whatsapp
│   │   └── service/              # Generate & hash password, send email (async)
│   │
│   ├── booking/
│   │   ├── controller/           # Booking CRUD + dashboard + sessions-ending-soon + cancel/complete
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
│   │   ├── controller/           # Global holiday CRUD
│   │   ├── dto/                  # CreateGlobalHolidayRequest, UpdateGlobalHolidayRequest
│   │   ├── repository/           # FindAll, FindByID, FindByDate, Create, Update, SoftDelete
│   │   └── service/              # GetAll, Create, Update, Delete
│   │
│   ├── notification_template/
│   │   ├── controller/           # GET list, GET by key, PUT update, POST preview
│   │   ├── dto/                  # UpdateTemplateRequest, PreviewRequest
│   │   ├── repository/           # FindAll, FindByKey, UpdateByKey
│   │   └── service/              # GetRendered (used by the customer/voucher/booking/play_credits services)
│   │
│   ├── event_booking/
│   │   ├── controller/           # Event booking CRUD + dashboard + preview-price + cancel
│   │   ├── dto/                  # CreateEventBookingRequest, EventBookingFilter, CancelEventBookingRequest
│   │   ├── repository/           # CRUD + CheckOverlapWithRegular/Event + BatchUpdateStatus + UpsertEventPrice
│   │   └── service/              # calculateTotalPrice (price/day÷24×hours), computeStatus, PreviewPrice
│   │
│   ├── admin_recovery/
│   │   ├── controller/           # Request, Validate, Reset: always the same response (security)
│   │   ├── repository/           # GenerateToken, FindSuperAdminByEmail, CountRequestsFromIP, MarkTokenUsed
│   │   └── service/              # RequestReset (rate limit 3/hour/IP, async email), ValidateToken, ResetPassword
│   │
│   └── upload/
│       └── controller/           # POST /upload: saves the file to ./assets/img/{folder}/
│
├── routes/
│   └── router.go                 # All endpoints, public vs protected groups
│
├── utils/
│   ├── jwt.go                    # GenerateJWT, ValidateJWT, JWTClaims struct
│   ├── bcrypt.go                 # HashPassword, CheckPassword
│   ├── response.go               # ResponseSuccess, ResponseError, ResponseSuccessPaginate, Meta
│   ├── upload.go                 # SaveFileToAssets, InitAssetsDir, DeleteFile
│   ├── email.go                  # dispatch, SendEmail (plain text→HTML), SendHTMLEmail, SendCustomerPasswordEmail
│   ├── email_templates.go        # BuildBookingEmailHTML, BuildVoucherEmailHTML (designed HTML fallback)
│   ├── template_renderer.go      # RenderTemplate ({{var}} substitution), GetTemplateOrFallback
│   └── password_generator.go     # GeneratePasswordFromName (character substitution + #Gl suffix)
│
├── assets/
│   └── img/                      # Uploaded images (served as static files)
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

| Base URL | Used by | Auth |
|----------|---------|------|
| `/api/admin` | Admin FE (staff) | Cookie `staff_token` (Path=`/api/admin`) |
| `/api/customer` | Customer web | Cookie `customer_token` (Path=`/api/customer`) |
| `/api/public` | Customer web | None |
| `/webhook/xendit` | Xendit | Header `x-callback-token` |

The staff tables below are relative to `/api/admin` (example: `/stores` = `/api/admin/stores`).

### 🔓 Staff — no auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Login; the server sets the `staff_token` cookie |
| POST | `/admin-recovery/request` | Send a password reset link to the super admin email |
| GET | `/admin-recovery/:token/validate` | Validate the token before the reset form is shown |
| POST | `/admin-recovery/:token/reset` | Reset the super admin password with a valid token |

> **Admin Recovery**: the `/request` response is always the same (200 OK), so an attacker cannot tell whether an email is registered. Rate limit: max 3 requests/hour per IP. The token is valid for 15 minutes and can be used only once.

**Upload** (`POST /upload`, requires staff login), `multipart/form-data`:
- `file` *(required)*: image file (jpg, png, webp, svg, max 2MB)
- `folder` *(optional)*: target subfolder, e.g. `stores`, `facilities` (default: `img`)

Response:
```json
{ "status": "success", "data": { "url": "/assets/img/stores/uuid.jpg" } }
```

---

### 🔒 Staff — protected (cookie `staff_token`)

#### Auth
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/auth/me` | Logged-in staff data |
| POST | `/auth/logout` | Logout |

#### Roles
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/roles` | List all roles (+ permissions) |
| POST | `/roles` | Create a role |
| GET | `/roles/:id` | Role detail |
| PUT | `/roles/:id` | Update a role |
| DELETE | `/roles/:id` | Delete a role (soft delete) |

#### Staffs
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/staffs` | List staff (filter: search, role_id, store_id) |
| POST | `/staffs` | Create staff |
| GET | `/staffs/:id` | Staff detail |
| PUT | `/staffs/:id` | Update staff |
| DELETE | `/staffs/:id` | Delete staff (soft delete) |
| POST | `/staffs/:id/reset-password` | Reset a staff password and send an email (Super Admin only) |

#### Facility Categories
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/facility-categories` | List all categories |
| POST | `/facility-categories` | Create a category |
| PUT | `/facility-categories/:id` | Update a category |
| DELETE | `/facility-categories/:id` | Delete a category |

#### Facilities
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/facilities` | List facilities (filter: search, category_id) |
| POST | `/facilities` | Create a facility |
| GET | `/facilities/:id` | Facility detail |
| PUT | `/facilities/:id` | Update a facility |
| DELETE | `/facilities/:id` | Delete a facility |

#### Room Templates
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/room-templates` | List room templates (+ facilities & category) |
| POST | `/room-templates` | Create a room template |
| GET | `/room-templates/:id` | Room template detail |
| PUT | `/room-templates/:id` | Update a room template |
| DELETE | `/room-templates/:id` | Delete a room template |

#### Stores
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/stores` | List stores (+ operating hours, holidays, room count) |
| POST | `/stores` | Create a store |
| GET | `/stores/operating-hours` | Effective store operating hours on a given date |
| GET | `/stores/:id` | Store detail (+ rooms → template → facilities → category) |
| PUT | `/stores/:id` | Update a store |
| DELETE | `/stores/:id` | Delete a store (soft delete) |
| GET | `/stores/:id/event-price` | Get the event price per day for this store |
| PUT | `/stores/:id/event-price` | Set/update the event price per day for this store |

**`GET /stores/operating-hours`**: query params `store_id` *(required)*, `date` *(required, YYYY-MM-DD)*

Response:
```json
{
  "open_time": "10:00:00",
  "close_time": "22:00:00",
  "is_holiday": true,
  "holiday_name": "Indonesian Independence Day",
  "holiday_type": "global"
}
```
Priority: **Global Holiday** → **Store Holiday** → **Regular Hours** (weekday/weekend)

#### Pricing
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/pricing` | List all pricing configs |
| POST | `/pricing` | Create a pricing config |
| GET | `/pricing/:store_id` | Pricing config for a store |
| PUT | `/pricing/:store_id` | Update a pricing config |
| DELETE | `/pricing/:store_id` | Delete a pricing config |
| POST | `/pricing/calculate` | Calculate an estimated booking price |
| POST | `/pricing/:store_id/happy-hour/schedules` | Add a happy hour schedule |
| DELETE | `/pricing/:store_id/happy-hour/schedules/:id` | Delete a happy hour schedule |
| GET | `/pricing/:store_id/happy-hour/prices` | List happy hour prices per room type |
| PUT | `/pricing/:store_id/happy-hour/prices` | Upsert happy hour prices |
| GET | `/pricing/:store_id/packages` | List duration package prices |
| PUT | `/pricing/:store_id/packages` | Upsert duration package prices |
| DELETE | `/pricing/:store_id/packages/:id` | Delete a package price |
| GET | `/pricing/:store_id/flash-sales` | List flash sales |
| POST | `/pricing/:store_id/flash-sales` | Create a flash sale |
| PUT | `/pricing/flash-sales/:id` | Update a flash sale |
| DELETE | `/pricing/flash-sales/:id` | Delete a flash sale |

**Calculate**: query params `store_id`, `room_template_id`, `date`, `start_time`, `duration_hours`

#### Play Credits — Packages
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/play-credits/packages` | List all packages |
| GET | `/play-credits/packages/active` | List active packages only |
| POST | `/play-credits/packages` | Create a package (`multipart/form-data`) |
| GET | `/play-credits/packages/:id` | Package detail |
| PUT | `/play-credits/packages/:id` | Update a package (`multipart/form-data`) |
| DELETE | `/play-credits/packages/:id` | Delete a package (soft delete) |

#### Play Credits — Member Credits
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/play-credits/members` | List credits ownership (filter: customer_id, search, is_active) |
| POST | `/play-credits/members` | Assign credits to a customer |
| GET | `/play-credits/members/:id` | Member credit detail |
| PATCH | `/play-credits/members/:id/adjust` | Adjust credit hours (add / subtract) |
| DELETE | `/play-credits/members/:id` | Delete a member credit (soft delete) |

#### Vouchers
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/vouchers` | List vouchers (filter: search, type, status) |
| GET | `/vouchers/generate-code` | Generate a unique voucher code |
| GET | `/vouchers/customer-available` | Vouchers available to a given customer |
| GET | `/vouchers/recipient-count` | Preview the number of member recipients by room type |
| POST | `/vouchers/validate` | Validate a voucher and check eligibility |
| POST | `/vouchers` | Create a voucher |
| GET | `/vouchers/:id` | Voucher detail |
| PUT | `/vouchers/:id` | Update a voucher |
| DELETE | `/vouchers/:id` | Delete a voucher (soft delete) |

**customer-available**: query params `customer_id` *(required)*, `store_id` *(required)*, `type` (default: `booking`)

**recipient-count**: query params `is_all_room_types` (`true`/`false`, default `true`), `room_template_ids[]` *(repeated uint)*

**Room type targeting on vouchers:**
| Field | Type | Description |
|-------|------|-------------|
| `is_all_room_types` | bool | `true` = valid for all room types; `false` = per room type |
| `room_template_ids` | `[]uint` | Room template IDs (required when `is_all_room_types=false`) |

> When `is_all_room_types = false`, only members who have one of those room types as a **favorite** receive the notification. Voucher validation at booking time also checks the selected room type.

#### Customers
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/customers` | List customers (filter: search, type, store_id) |
| POST | `/customers` | Create a customer |
| GET | `/customers/:id` | Customer detail |
| PUT | `/customers/:id` | Update a customer |
| PATCH | `/customers/:id/notes` | Update internal customer notes |
| DELETE | `/customers/:id` | Delete a customer (soft delete) |
| POST | `/customers/:id/resend-password` | Resend the password email |

#### Bookings
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/bookings` | List bookings (filter: store_id, date_from, date_to, status, search) |
| GET | `/bookings/dashboard` | Booking dashboard data per store (rooms + bookings + effective operating hours + holiday info) |
| GET | `/bookings/sessions-ending-soon` | Sessions ending in the next 5 minutes |
| GET | `/bookings/available-credits` | Play credits available to a given customer |
| POST | `/bookings` | Create a booking |
| GET | `/bookings/:id` | Booking detail |
| PATCH | `/bookings/:id/cancel` | Cancel a booking |
| PATCH | `/bookings/:id/complete` | Mark a booking as completed |

**available-credits**: query params `customer_id` *(required)*, `store_id` *(required)*

#### Sales
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/sales/summary` | Main dashboard: revenue, transactions, breakdown per type/branch/room |
| GET | `/sales/trend` | Sales trend chart data |
| GET | `/sales/transactions` | Transaction list (bookings + play credits) with pagination |

#### Global Holidays
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/global-holidays` | List national holidays (filter: year, search) |
| POST | `/global-holidays` | Add a national holiday |
| PUT | `/global-holidays/:id` | Update a holiday |
| DELETE | `/global-holidays/:id` | Delete a holiday (soft delete) |

> A global holiday **disables happy hour** in all stores on that date. It takes priority over store holidays and the weekday check.

**Body `POST /global-holidays`:**
```json
{
  "date": "2025-08-17",
  "name": "Indonesian Independence Day",
  "open_time": "10:00",
  "close_time": "22:00"
}
```

#### Event Bookings
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/event-bookings` | List event bookings (filter: store_id, status, date_from, date_to) |
| POST | `/event-bookings` | Create an event booking |
| GET | `/event-bookings/dashboard` | Active event bookings for the calendar grid (query: `store_id`, `date`) |
| GET | `/event-bookings/preview-price` | Estimated price preview for admin (query: `store_id`, `start_time`, `end_time`). The customer web uses `/api/public/event-booking/quote` |
| GET | `/event-bookings/:id` | Event booking detail |
| PATCH | `/event-bookings/:id/cancel` | Cancel an event booking |

**Body `POST /event-bookings`:**
```json
{
  "store_id": "uuid-store",
  "event_name": "Andi's Birthday",
  "customer_name": "Budi Santoso",
  "customer_whatsapp": "08123456789",
  "customer_email": "budi@example.com",
  "booking_date": "2025-12-25",
  "start_time": "10:00",
  "end_time": "22:00",
  "notes": "Please prepare decorations"
}
```

> - `duration_hours` is **calculated automatically** from `start_time` and `end_time`. Do not send it.
> - `total_price` = `round((price_per_day ÷ 24 × duration_hours) / 1000) × 1000` (rounded to the nearest Rp 1,000).
> - The event price per store is configured with `PUT /stores/:id/event-price`.
> - An event booking **blocks the whole store**. Regular bookings that overlap are rejected automatically.
> - Overlap uses absolute time per operating day: a time before the store opening hour counts as after midnight. Example (store 10:00–02:00): a booking 00:00–01:00 on the same `booking_date` conflicts with an event 20:00–02:00.
> - If a regular booking already exists in that time range, creating the event booking fails with an error message.

#### Notification Templates
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/notification-templates` | List all notification templates |
| GET | `/notification-templates/:key` | Template detail by key |
| PUT | `/notification-templates/:key` | Update template content (email & WhatsApp) |
| POST | `/notification-templates/preview` | Preview a template with sample data |

Default **notification_key** values:

| Key | Description | Variables |
|-----|-------------|-----------|
| `customer_welcome` | Welcome email + new account password | `nama_customer`, `email`, `password` |
| `voucher_notification` | Voucher notification to all members | `nama_customer`, `nama_voucher`, `kode_voucher`, `berlaku_sampai`, `deskripsi_voucher` |
| `booking_confirmation` | Booking confirmation to the customer | `nama_customer`, `kode_booking`, `nama_ruangan`, `tanggal`, `jam_mulai`, `jam_selesai`, `durasi`, `total_harga` |
| `play_credits_assigned` | Play credits assigned to the customer | `nama_customer`, `nama_paket`, `total_jam`, `sisa_jam`, `tanggal_kadaluwarsa` |

> Templates are stored in the `notification_templates` table. Admins can change email & WhatsApp content without a redeploy. Dynamic variables use the `{{variable_name}}` format.

**Filter params (all sales endpoints):**
| Param | Values | Default |
|-------|--------|---------|
| `period` | `today` \| `yesterday` \| `this_week` \| `this_month` \| `custom` | `today` |
| `store_id` | Store UUID | — (all stores) |
| `date_from` | `YYYY-MM-DD` | — (required when `period=custom`) |
| `date_to` | `YYYY-MM-DD` | — (required when `period=custom`) |
| `granularity` | `daily` \| `weekly` \| `monthly` | `daily` (`/trend` only) |
| `type` | `all` \| `booking` \| `play_credits` | `all` (`/transactions` only) |
| `page` | integer | `1` (`/transactions` only) |
| `per_page` | integer | `20` (`/transactions` only) |

---

### 🌐 Public — customer web (`/api/public`, no auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/banners` | Active banners, ordered by `sort_order` |
| GET | `/banners/:id` | Active banner detail (inactive banner → 404) |
| GET | `/stores` | List stores |
| GET | `/stores/:id` | Store detail |
| GET | `/room-templates` | Active room templates + `min_price` (optional query: `store_id`) |
| GET | `/room-templates/:id` | Room template detail + active `facilities` + `min_price` |
| GET | `/booking/availability` | Available slots (query: `store_id`, `room_template_id`, `date`, `duration_hours`) |
| GET | `/booking/slots` | Hourly slots + remaining units + price |
| GET | `/booking/quote` | Final price for the selected slots without creating a hold (query: `store_id`, `room_template_id`, `booking_date`, `selected_slots[]`) |
| GET | `/play-credits/packages` | Play credits packages |
| GET | `/event-booking/availability` | Taken time ranges + `event_price` (query: `store_id`, `date`, optional `start_time`, `end_time`) |
| GET | `/event-booking/quote` | Customer event price (query: `store_id`, `booking_date`, `start_time`, `end_time`) |
| GET | `/fnb/menu` | FnB menu |

> - Banner and room template responses use a field whitelist. Audit columns (`created_by`, `updated_by`, `deleted_by`, `deleted_at`) and `is_active` are not sent, because `created_by`/`updated_by` contain staff usernames.
> - `/event-booking/quote` → `{store_id, booking_date, start_time, end_time, duration_hours, price_per_day, total_price, available}`. `total_price` is exactly what `/customer/event-bookings/initiate` charges (rounded to Rp 1,000). `available` = no conflict with regular bookings or other events right now.

### 👤 Customer (`/api/customer`, cookie `customer_token`)

**No auth:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/login` | Login; the server sets the `customer_token` cookie |
| POST | `/forgot-password` | Send a password reset link |
| GET | `/reset-password/:token/validate` | Validate a reset token |
| POST | `/reset-password/:token` | Reset the password |

**Login required:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/me` | Logged-in customer data |
| POST | `/logout` | Logout (all older tokens are rejected) |
| PUT | `/profile` | Update the profile |
| PUT | `/change-password` | Change the password. Wrong old password → **400** (not 401) |
| GET | `/room-recommendations` | Room recommendations |
| GET | `/credits/expiring` | Play credits that will expire soon |
| GET | `/my-credits` | The customer's play credits |
| GET | `/vouchers/available` | Usable vouchers |
| POST | `/bookings/initiate` | Create a hold + Xendit invoice |
| GET | `/bookings` | List bookings (query: `page`, `per_page` max 50, `status`) |
| GET | `/bookings/by-hold/:hold_id` | Payment status of a hold, for the payment success page |
| GET | `/bookings/:id` | Booking detail |
| POST | `/play-credits/purchase/initiate` | Buy a play credits package (Xendit invoice) |
| POST | `/event-bookings/initiate` | Create an event booking + Xendit invoice |
| POST | `/fnb/orders` | Order FnB |
| GET | `/fnb/orders` | FnB order history |

The `.../mock-confirm` endpoints (booking, play credits, event) are registered only when `XENDIT_SECRET_KEY` is empty (dev mock mode).

**`GET /bookings`**: optional `status`, comma-separated: `upcoming`, `ongoing`, `completed`, `cancelled` (example `?status=upcoming,ongoing`). Any other value → 400. The status stored in the DB can be stale, because the bulk sync only runs when the admin dashboard is opened. So before the query, the customer's booking statuses are synced against the current WIB time.

**`GET /bookings/by-hold/:hold_id`**: `data` is `{status, booking_code?, booking_id?}`:

| `status` | Meaning |
|----------|---------|
| `pending` | The hold is still valid; the webhook has not been processed yet |
| `confirmed` | The booking exists. `booking_code` and `booking_id` are set |
| `expired` | The hold passed `expires_at`. Not final: a late webhook can still confirm it if the slot is free |

A hold owned by another customer, an unknown ID, and a hold that was already cleaned up (about 1 hour after expiry) all return **404** with the same body, so this endpoint cannot be used to check whether an ID exists.

**Rate limits** (in-memory, per IP + route):

| Endpoint | Limit |
|----------|-------|
| `/admin/auth/login`, `/admin/admin-recovery/request`, `/admin/admin-recovery/:token/reset` | 10/minute |
| `/customer/login`, `/customer/forgot-password`, `/customer/reset-password/:token`, `/customer/change-password` | 10/minute |
| `/customer/bookings/by-hold/:hold_id` | 30/minute |

Over the limit → **429**. Behind a reverse proxy, `TRUSTED_PROXIES` must be set. Without it, `c.ClientIP()` is the proxy IP, so all users share one quota.

---

## Email System

### Email delivery architecture

```
Trigger (create booking / create voucher / create customer)
  │
  ├─► ntService.GetRendered(key, vars)   ← load the template from the DB
  │     │
  │     ├── Template found & active
  │     │     └─► utils.SendEmail()      ← plain text + generic card wrapper
  │     │
  │     └── Template missing / error
  │           └─► utils.SendHTMLEmail()  ← designed HTML (BuildBookingEmailHTML / BuildVoucherEmailHTML)
  │
  └── [everything runs in a goroutine; it does not block the HTTP response]
```

### Email utility functions

| Function | When it is used |
|----------|-----------------|
| `SendEmail(to, name, subject, plainText)` | DB template (plain text with `\n`, auto-wrapped in a card) |
| `SendHTMLEmail(to, name, subject, html)` | Hardcoded designed template (booking & voucher fallback) |
| `SendCustomerPasswordEmail(to, name, pwd)` | Fallback welcome email when the DB template does not exist |
| `BuildBookingEmailHTML(...)` | Designed booking confirmation HTML |
| `BuildVoucherEmailHTML(...)` | Designed voucher notification HTML |

### Template priority

1. **DB template** (`notification_templates` table): admins can customize it via `PUT /notification-templates/:key`
2. **Hardcoded fallback**: built-in designed HTML (booking & voucher), `SendCustomerPasswordEmail` (customer welcome)

### SMTP configuration (Gmail)

1. Enable **2-Step Verification** on the Google account
2. Create an **App Password** at [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords)
3. Put that App Password in `SMTP_PASSWORD` (16 characters, no spaces)

---

## Authentication

The JWT is sent in an **HttpOnly cookie**, not in the `Authorization` header (the header is not accepted).

| App | Cookie | Path |
|-----|--------|------|
| Admin (staff) | `staff_token` | `/api/admin` |
| Customer web | `customer_token` | `/api/customer` |

The cookie paths are separate so a staff token is never sent to customer routes, and the reverse. State-changing requests (POST/PUT/PATCH/DELETE) with an `Origin` header that is not listed in `ALLOWED_ORIGINS` are rejected (CSRF protection).

The staff JWT contains: `staff_id`, `username`, `role_id`, `is_system`, `permissions[]`

---

## Audit Field Convention

Every record has these columns:

| Column | Content |
|--------|---------|
| `created_by` | Username of the staff who created it |
| `updated_by` | Username of the staff who last updated it |
| `deleted_by` | Username of the staff who deleted it |
| `deleted_at` | Soft delete timestamp (null = active) |

> **Bookings** do not use soft delete. Records are permanent because they are financial records.

---

## Pricing Engine Logic

Order of price rules for `POST /pricing/calculate` or when creating a booking:

1. **Flash sale**: compute the minute overlap between the booking time and active flash sale windows; the overlapping part uses the flash sale `price_per_hour` (a direct hourly rate, not a discount)
2. **Remaining duration** (non-flash): check `isWeekdayForPricing()`:
   - Global holiday → **normal** price (happy hour disabled)
   - Store holiday → **normal** price (happy hour disabled)
   - Monday–Thursday → check the **happy hour** schedule (inside → HH price, otherwise → package price → normal)
   - Friday–Sunday → check the **package price** → normal
3. **Voucher**: the voucher discount is applied on top of `final_price` (booking only)

**Flash sale** uses the `price_per_hour` field (a direct hourly rate, not a percentage discount). Example: flash sale `price_per_hour = 50,000` active 19:00–21:00, booking 19:00–23:00 → 2 hours × 50,000 + 2 hours at normal/HH price.

---

## Booking Flow

```
POST /bookings
  ├── Check room schedule overlap (regular bookings in the same room)
  ├── Check event booking overlap (active events in the same store at that time)
  ├── Calculate the price with the pricing engine
  ├── Validate the voucher (optional, members only)
  ├── Validate play credits (optional, members only)
  ├── Generate the booking code (BK-YYMMDD-XXXX)
  ├── Save the booking record
  └── [goroutine] Deduct play credits + redeem voucher + send notification email
```

## Online Booking Flow (Customer)

```
POST /customer/bookings/initiate
  ├── Calculate the price (same as /public/booking/quote)
  ├── Create booking_hold (locks a room unit, valid for 15 minutes) + Xendit invoice
  └── Xendit redirects → {APP_URL}/payment/success?hold_id=<id>

POST /webhook/xendit (status PAID)
  ├── Hold expired + slot already taken → log [PAYMENT ORPHAN], manual refund needed
  ├── Transaction: delete the hold + create the booking (bookings.hold_id = hold id)
  └── [goroutine] Send the confirmation email

GET /customer/bookings/by-hold/:hold_id   ← polled by the payment success page
  └── pending → confirmed (booking_code) / expired
```

## Event Booking Flow

```
POST /event-bookings
  ├── Calculate the duration from start_time - end_time
  ├── Overlap check: regular booking in the store at that time? → reject (includes the previous/next date)
  ├── Overlap check: another event booking in the store at that time? → reject
  ├── Load the event price from store_event_prices
  ├── Calculate the total: round((price_per_day ÷ 24 × hours) / 1000) × 1000
  └── Save the event booking (status: upcoming)
```

---
