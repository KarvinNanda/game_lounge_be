# Game Lounge Backend API

REST API untuk manajemen game lounge — mengelola store, staff, room template, fasilitas, dan jadwal operasional.

**Tech Stack:** Go · Gin · GORM · MySQL · JWT

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
```

### 4. Buat database MySQL

```sql
CREATE DATABASE game_lounge_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

> Migrasi tabel dilakukan otomatis oleh GORM saat server pertama kali dijalankan *(pastikan AutoMigrate sudah dikonfigurasi)*, atau import SQL schema secara manual sesuai kebutuhan.

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
│
├── cmd/
│   └── main.go                   # Entry point — inisialisasi DB, router, server
│
├── config/
│   ├── config.go                 # LoadEnv, GetEnv helper
│   └── database.go               # Koneksi GORM ke MySQL (config.DB)
│
├── middleware/
│   ├── auth.go                   # JWT auth middleware — set staff_id, staff_username, dll ke context
│   └── cors.go                   # CORS middleware (allow all origins)
│
├── models/                       # Definisi struct GORM (mapping ke tabel DB)
│   ├── role.go                   # Role (id, name, is_system)
│   ├── role_permission.go        # RolePermission (role_id, permission)
│   ├── staff.go                  # Staff (uuid PK, role_id, username, email, avatar_url)
│   ├── staff_store.go            # StaffStore — junction table Staff ↔ Store
│   ├── facility_category.go      # FacilityCategory (id, name, icon_url)
│   ├── facility.go               # Facility (id, category_id, name, icon_url, is_active)
│   ├── room_template.go          # RoomTemplate (id, name, capacity_min/max, facilities M2M)
│   ├── room_template_facility.go # Junction table RoomTemplate ↔ Facility
│   ├── store.go                  # Store (uuid PK, name, address, status, photo_url)
│   ├── store_operating_hour.go   # StoreOperatingHour (store_id, day_type, open/close_time)
│   ├── store_holiday_schedule.go # StoreHolidaySchedule (store_id, date, open/close_time)
│   └── store_room.go             # StoreRoom (uuid PK, store_id, room_template_id, unit_number)
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
│   │   ├── dto/                  # CreateRoleRequest, UpdateRoleRequest, RoleFilter
│   │   ├── repository/           # FindAllRoles (+ Preload Permissions), SyncPermissions
│   │   └── service/              # RoleWithPermissions — ekstrak string permissions dari preload
│   │
│   ├── staff/
│   │   ├── controller/           # CRUD staff
│   │   ├── dto/                  # CreateStaffRequest, UpdateStaffRequest, StaffFilter
│   │   ├── repository/           # FindAllStaffs (Preload Role, StaffStores.Store)
│   │   └── service/              # Hash password, sync store assignments
│   │
│   ├── facility/
│   │   ├── controller/           # CRUD kategori + CRUD fasilitas
│   │   ├── dto/                  # CreateCategoryRequest, CreateFacilityRequest, dll
│   │   ├── repository/           # FindAllFacilities (Preload Category)
│   │   └── service/              # GetAllCategories, GetAllFacilities, dll
│   │
│   ├── room_template/
│   │   ├── controller/           # CRUD room template
│   │   ├── dto/                  # CreateRoomTemplateRequest (facility_ids, image_url)
│   │   ├── repository/           # Preload Facilities.Category, SyncFacilities (M2M replace)
│   │   └── service/              # CreateRoomTemplate, UpdateRoomTemplate + SyncFacilities
│   │
│   ├── store/
│   │   ├── controller/           # CRUD store
│   │   ├── dto/                  # CreateStoreRequest (operating_hours, holidays, rooms)
│   │   ├── repository/           # FindAllStores (Preload OpHours, Holidays)
│   │   │                         # FindStoreByID (Preload full chain → Rooms.RoomTemplate.Facilities.Category)
│   │   └── service/              # StoreListItem, CreateStore, UpdateStore, UpsertOperatingHours/Holidays
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
│   ├── response.go               # ResponseSuccess, ResponseError, Meta struct
│   └── upload.go                 # SaveFileToAssets, InitAssetsDir, DeleteFile
│
├── assets/
│   └── img/                      # File gambar yang diupload (served sebagai static files)
│       ├── stores/
│       ├── facilities/
│       ├── room_templates/
│       ├── staffs/
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
| GET | `/stores/:id` | Detail store (+ rooms → template → facilities → category) |
| PUT | `/stores/:id` | Update store |
| DELETE | `/stores/:id` | Hapus store (soft delete) |

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

---

## Static Files

File yang diupload dapat diakses langsung via URL:

```
GET /assets/img/{folder}/{filename}
```

Contoh: `http://localhost:8080/assets/img/stores/abc123.jpg`
