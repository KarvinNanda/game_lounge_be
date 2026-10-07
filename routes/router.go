package routes

import (
	"game_lounge_be/middleware"
	recoveryCtrl "game_lounge_be/modules/admin_recovery/controller"
	authCtrl "game_lounge_be/modules/auth/controller"
	bannerCtrl "game_lounge_be/modules/banner/controller"
	bookingCtrl "game_lounge_be/modules/booking/controller"
	customerCtrl "game_lounge_be/modules/customer/controller"
	customerAppCtrl "game_lounge_be/modules/customer_app/controller"
	bookingCustomerCtrl "game_lounge_be/modules/customer_booking/controller"
	eventCtrl "game_lounge_be/modules/event_booking/controller"
	facilityCtrl "game_lounge_be/modules/facility/controller"
	fnbCtrl "game_lounge_be/modules/fnb/controller"
	globalHolidayCtrl "game_lounge_be/modules/global_holiday/controller"
	ntCtrl "game_lounge_be/modules/notification_template/controller"
	playCreditsCtrl "game_lounge_be/modules/play_credits/controller"
	pricingCtrl "game_lounge_be/modules/pricing/controller"
	roleCtrl "game_lounge_be/modules/role/controller"
	roomTemplateCtrl "game_lounge_be/modules/room_template/controller"
	salesCtrl "game_lounge_be/modules/sales/controller"
	staffCtrl "game_lounge_be/modules/staff/controller"
	storeCtrl "game_lounge_be/modules/store/controller"
	uploadCtrl "game_lounge_be/modules/upload/controller"
	voucherCtrl "game_lounge_be/modules/voucher/controller"
	"game_lounge_be/utils"

	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// ── Trusted proxies ───────────────────────────────────────
	// Default: tidak percaya proxy manapun → ClientIP() tidak bisa dispoof
	// via X-Forwarded-For. Jika di belakang reverse proxy (Coolify/nginx),
	// set TRUSTED_PROXIES="10.0.0.0/8" (comma-separated CIDR/IP).
	if tp := os.Getenv("TRUSTED_PROXIES"); tp != "" {
		r.SetTrustedProxies(strings.Split(tp, ","))
	} else {
		r.SetTrustedProxies(nil)
	}

	// Batas memori parsing multipart (file upload)
	r.MaxMultipartMemory = 8 << 20 // 8 MB

	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.MaxBodySize(10 << 20)) // 10 MB max request body
	r.Use(middleware.ClampPerPage(100))     // per_page maksimal 100
	r.Use(middleware.SVGAsAttachment())     // SVG upload tidak dirender saat dibuka langsung

	// Rate limiter untuk endpoint sensitif (login, forgot password):
	// max 10 request per menit per IP per endpoint.
	authLimiter := middleware.RateLimit(10, time.Minute)

	// ── Health check ──────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── Static file serving ───────────────────────────────────
	r.Static("/assets/img", "./assets/img") // primary asset dir (used by /upload)
	r.Static("/uploads", "./uploads")       // legacy upload dir

	api := r.Group("/api")

	// ── Admin app ─────────────────────────────────────────────
	// SEMUA rute yang dipakai admin FE hidup di bawah /api/admin.
	// Ini memungkinkan cookie staff_token di-scope ke Path=/api/admin
	// sehingga tidak pernah terkirim ke rute customer (/api/customer/*).
	// Admin FE cukup mengganti baseURL: /api → /api/admin.
	api.POST("/admin/auth/login", authLimiter, authCtrl.Login)

	// ── Admin Recovery (forgot password super admin) ──────────
	api.POST("/admin/admin-recovery/request", authLimiter, recoveryCtrl.Request)
	api.GET("/admin/admin-recovery/:token/validate", recoveryCtrl.Validate)
	api.POST("/admin/admin-recovery/:token/reset", authLimiter, recoveryCtrl.Reset)

	// ── Protected endpoints (staff JWT via cookie) ────────────
	protected := api.Group("/admin")
	protected.Use(middleware.AuthMiddleware())
	{
		// Auth
		protected.GET("auth/me", authCtrl.Me)
		protected.POST("auth/logout", authCtrl.Logout)

		// Upload butuh staff JWT — endpoint upload publik berisiko
		// disalahgunakan (disk filling, hosting konten berbahaya).
		protected.POST("upload", uploadCtrl.Upload)

		// Roles
		protected.GET("roles", middleware.RequirePermission("settings.staff_role"), roleCtrl.GetAll)
		protected.POST("roles", middleware.RequirePermission("settings.staff_role"), roleCtrl.Create)
		protected.GET("roles/:id", middleware.RequirePermission("settings.staff_role"), roleCtrl.GetByID)
		protected.PUT("roles/:id", middleware.RequirePermission("settings.staff_role"), roleCtrl.Update)
		protected.DELETE("roles/:id", middleware.RequirePermission("settings.staff_role"), roleCtrl.Delete)

		// Staffs
		protected.GET("staffs", middleware.RequirePermission("settings.staff_role"), staffCtrl.GetAll)
		protected.POST("staffs", middleware.RequirePermission("settings.staff_role"), staffCtrl.Create)
		protected.GET("staffs/:id", middleware.RequirePermission("settings.staff_role"), staffCtrl.GetByID)
		protected.PUT("staffs/:id", middleware.RequirePermission("settings.staff_role"), staffCtrl.Update)
		protected.DELETE("staffs/:id", middleware.RequirePermission("settings.staff_role"), staffCtrl.Delete)
		protected.POST("staffs/:id/reset-password", middleware.RequireSuperAdmin(), staffCtrl.ResetPassword)

		// Facility Categories
		protected.GET("facility-categories", facilityCtrl.GetAllCategories)
		protected.POST("facility-categories", middleware.RequirePermission("settings.branches"), facilityCtrl.CreateCategory)
		protected.PUT("facility-categories/:id", middleware.RequirePermission("settings.branches"), facilityCtrl.UpdateCategory)
		protected.DELETE("facility-categories/:id", middleware.RequirePermission("settings.branches"), facilityCtrl.DeleteCategory)

		// Facilities
		protected.GET("facilities", facilityCtrl.GetAll)
		protected.POST("facilities", middleware.RequirePermission("settings.branches"), facilityCtrl.Create)
		protected.GET("facilities/:id", facilityCtrl.GetByID)
		protected.PUT("facilities/:id", middleware.RequirePermission("settings.branches"), facilityCtrl.Update)
		protected.DELETE("facilities/:id", middleware.RequirePermission("settings.branches"), facilityCtrl.Delete)

		// Room Templates
		protected.GET("room-templates", roomTemplateCtrl.GetAll)
		protected.POST("room-templates", middleware.RequirePermission("rooms.create"), roomTemplateCtrl.Create)
		protected.GET("room-templates/:id", roomTemplateCtrl.GetByID)
		protected.PUT("room-templates/:id", middleware.RequirePermission("rooms.edit"), roomTemplateCtrl.Update)
		protected.DELETE("room-templates/:id", middleware.RequirePermission("rooms.delete"), roomTemplateCtrl.Delete)

		// Stores
		// Note: rute statis (operating-hours) didaftarkan SEBELUM /:id
		// supaya Gin tidak mengira "operating-hours" adalah store_id.
		protected.GET("stores", storeCtrl.GetAll)
		protected.POST("stores", middleware.RequirePermission("settings.branches"), storeCtrl.Create)
		protected.GET("stores/operating-hours", storeCtrl.GetOperatingHours)
		protected.GET("stores/:id", storeCtrl.GetByID)
		protected.PUT("stores/:id", middleware.RequirePermission("settings.branches"), middleware.RequireStoreParam("id"), storeCtrl.Update)
		protected.DELETE("stores/:id", middleware.RequirePermission("settings.branches"), middleware.RequireStoreParam("id"), storeCtrl.Delete)

		// Store Rooms
		protected.PATCH("store-rooms/:id/toggle", middleware.RequirePermission("settings.branches"), middleware.RequireEntityStore("id", middleware.StoreOf("store_rooms")), storeCtrl.ToggleRoomActive)

		// ── Pricing ───────────────────────────────────────────
		// Note: rute statis (calculate, flash-sales) didaftarkan SEBELUM /:store_id
		// supaya Gin tidak mengira "calculate" adalah store_id.

		// Calculator (POST — tidak bentrok dengan GET /:store_id)
		protected.POST("pricing/calculate", middleware.RequirePermission("pricing.view", "bookings.create"), pricingCtrl.Calculate)

		// Flash Sales (tanpa store_id prefix — untuk PUT/DELETE by flash-sale ID)
		protected.PUT("pricing/flash-sales/:id", middleware.RequirePermission("pricing.edit"), middleware.RequireEntityStore("id", middleware.StoreOf("store_flash_sales")), pricingCtrl.UpdateFlashSale)
		protected.DELETE("pricing/flash-sales/:id", middleware.RequirePermission("pricing.edit"), middleware.RequireEntityStore("id", middleware.StoreOf("store_flash_sales")), pricingCtrl.DeleteFlashSale)

		// Pricing config
		protected.GET("pricing", middleware.RequirePermission("pricing.view"), pricingCtrl.GetAll)
		protected.POST("pricing", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreInBody(), pricingCtrl.Create)
		protected.GET("pricing/:store_id", middleware.RequirePermission("pricing.view"), middleware.RequireStoreParam("store_id"), pricingCtrl.GetByStore)
		protected.PUT("pricing/:store_id", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.UpdateConfig)
		protected.DELETE("pricing/:store_id", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.Delete)

		// Happy Hour — Schedules
		protected.POST("pricing/:store_id/happy-hour/schedules", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.AddSchedule)
		protected.DELETE("pricing/:store_id/happy-hour/schedules/:id", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.DeleteSchedule)

		// Happy Hour — Prices
		protected.GET("pricing/:store_id/happy-hour/prices", middleware.RequirePermission("pricing.view"), middleware.RequireStoreParam("store_id"), pricingCtrl.GetHappyHourPrices)
		protected.PUT("pricing/:store_id/happy-hour/prices", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.UpsertHappyHourPrices)

		// Package Prices
		protected.GET("pricing/:store_id/packages", middleware.RequirePermission("pricing.view"), middleware.RequireStoreParam("store_id"), pricingCtrl.GetPackagePrices)
		protected.PUT("pricing/:store_id/packages", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.UpsertPackagePrices)
		protected.DELETE("pricing/:store_id/packages/:id", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.DeletePackagePrice)

		// Flash Sales (list + create dengan store_id)
		protected.GET("pricing/:store_id/flash-sales", middleware.RequirePermission("pricing.view"), middleware.RequireStoreParam("store_id"), pricingCtrl.GetFlashSales)
		protected.POST("pricing/:store_id/flash-sales", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("store_id"), pricingCtrl.CreateFlashSale)

		// ── Play Credits — Packages ───────────────────────────
		// Note: "active" didaftarkan sebelum /:id agar tidak dianggap sebagai ID.
		protected.GET("play-credits/packages", playCreditsCtrl.GetAllPackages)
		protected.GET("play-credits/packages/active", playCreditsCtrl.GetActivePackages)
		protected.POST("play-credits/packages", middleware.RequirePermission("play_credits.create"), playCreditsCtrl.CreatePackage)
		protected.GET("play-credits/packages/:id", playCreditsCtrl.GetPackageByID)
		protected.PUT("play-credits/packages/:id", middleware.RequirePermission("play_credits.edit"), playCreditsCtrl.UpdatePackage)
		protected.DELETE("play-credits/packages/:id", middleware.RequirePermission("play_credits.edit"), playCreditsCtrl.DeletePackage)

		// ── Play Credits — Member Credits ─────────────────────
		protected.GET("play-credits/members", middleware.RequirePermission("play_credits.view"), playCreditsCtrl.GetAllMemberCredits)
		protected.POST("play-credits/members", middleware.RequirePermission("play_credits.create"), playCreditsCtrl.AssignCredit)
		protected.GET("play-credits/members/:id", middleware.RequirePermission("play_credits.view"), playCreditsCtrl.GetCreditByID)
		protected.PATCH("play-credits/members/:id/adjust", middleware.RequirePermission("play_credits.edit"), playCreditsCtrl.AdjustCredit)
		protected.DELETE("play-credits/members/:id", middleware.RequirePermission("play_credits.edit"), playCreditsCtrl.DeleteCredit)

		// ── Sales Summary ─────────────────────────────────────
		protected.GET("sales/summary", middleware.RequirePermission("dashboard.view"), middleware.ScopeStoreQuery(), salesCtrl.GetSummary)
		protected.GET("sales/trend", middleware.RequirePermission("dashboard.view"), middleware.ScopeStoreQuery(), salesCtrl.GetTrend)
		protected.GET("sales/transactions", middleware.RequirePermission("dashboard.view"), middleware.ScopeStoreQuery(), salesCtrl.GetTransactions)

		// ── Notification Templates ─────────────────────────────
		// Note: rute statis (preview) didaftarkan SEBELUM /:key.
		protected.GET("notification-templates", middleware.RequirePermission("settings.staff_role"), ntCtrl.GetAll)
		protected.POST("notification-templates/preview", middleware.RequirePermission("settings.staff_role"), ntCtrl.Preview)
		protected.GET("notification-templates/:key", middleware.RequirePermission("settings.staff_role"), ntCtrl.GetByKey)
		protected.PUT("notification-templates/:key", middleware.RequirePermission("settings.staff_role"), ntCtrl.Update)

		// ── Bookings ──────────────────────────────────────────
		// Note: rute statis (dashboard, sessions-ending-soon, available-credits)
		// didaftarkan SEBELUM /:id agar tidak dianggap sebagai ID oleh Gin.
		protected.GET("bookings", middleware.RequirePermission("bookings.view"), middleware.ScopeStoreQuery(), bookingCtrl.GetAll)
		protected.GET("bookings/dashboard", middleware.RequirePermission("bookings.view"), middleware.ScopeStoreQuery(), bookingCtrl.GetDashboard)
		protected.GET("bookings/sessions-ending-soon", middleware.RequirePermission("bookings.view"), middleware.ScopeStoreQuery(), bookingCtrl.GetSessionsEndingSoon)
		protected.GET("bookings/available-credits", middleware.RequirePermission("bookings.create"), middleware.ScopeStoreQuery(), bookingCtrl.GetAvailableCredits)
		protected.POST("bookings", middleware.RequirePermission("bookings.create"), middleware.RequireStoreInBody(), bookingCtrl.Create)
		protected.GET("bookings/:id", middleware.RequirePermission("bookings.view"), middleware.RequireEntityStore("id", middleware.StoreOf("bookings")), bookingCtrl.GetByID)
		protected.PATCH("bookings/:id/cancel", middleware.RequirePermission("bookings.cancel"), middleware.RequireEntityStore("id", middleware.StoreOf("bookings")), bookingCtrl.Cancel)
		protected.PATCH("bookings/:id/complete", middleware.RequirePermission("bookings.edit"), middleware.RequireEntityStore("id", middleware.StoreOf("bookings")), bookingCtrl.Complete)

		// ── Vouchers ──────────────────────────────────────────
		// Note: rute statis didaftarkan SEBELUM /:id
		// agar Gin tidak menganggap path segment tersebut sebagai ID.
		protected.GET("vouchers", middleware.RequirePermission("promotion.view"), voucherCtrl.GetAll)
		protected.GET("vouchers/generate-code", middleware.RequirePermission("promotion.create"), voucherCtrl.GenerateCode)
		protected.GET("vouchers/customer-available", middleware.RequirePermission("bookings.create", "promotion.view"), voucherCtrl.GetCustomerAvailable)
		protected.GET("vouchers/recipient-count", middleware.RequirePermission("promotion.create", "promotion.edit"), voucherCtrl.GetRecipientCount)
		protected.POST("vouchers/validate", middleware.RequirePermission("bookings.create"), voucherCtrl.Validate)
		protected.POST("vouchers", middleware.RequirePermission("promotion.create"), voucherCtrl.Create)
		protected.GET("vouchers/:id", middleware.RequirePermission("promotion.view"), voucherCtrl.GetByID)
		protected.PUT("vouchers/:id", middleware.RequirePermission("promotion.edit"), voucherCtrl.Update)
		protected.DELETE("vouchers/:id", middleware.RequirePermission("promotion.edit"), voucherCtrl.Delete)

		// ── Customers ─────────────────────────────────────────
		protected.GET("customers", middleware.RequirePermission("customers.view", "bookings.create"), customerCtrl.GetAll)
		protected.POST("customers", middleware.RequirePermission("customers.create"), customerCtrl.Create)
		protected.GET("customers/:id", middleware.RequirePermission("customers.view", "bookings.create"), customerCtrl.GetByID)
		protected.PUT("customers/:id", middleware.RequirePermission("customers.edit"), customerCtrl.Update)
		protected.PATCH("customers/:id/notes", middleware.RequirePermission("customers.edit"), customerCtrl.UpdateNotes)
		protected.DELETE("customers/:id", middleware.RequirePermission("customers.edit"), customerCtrl.Delete)
		protected.POST("customers/:id/resend-password", middleware.RequirePermission("customers.edit"), customerCtrl.ResendPassword)

		// ── Global Holidays ───────────────────────────────────
		protected.GET("global-holidays", globalHolidayCtrl.GetAll)
		protected.POST("global-holidays", middleware.RequirePermission("settings.branches"), globalHolidayCtrl.Create)
		protected.PUT("global-holidays/:id", middleware.RequirePermission("settings.branches"), globalHolidayCtrl.Update)
		protected.DELETE("global-holidays/:id", middleware.RequirePermission("settings.branches"), globalHolidayCtrl.Delete)

		// ── Event Bookings ────────────────────────────────────
		// Note: rute statis (dashboard, preview-price) didaftarkan SEBELUM /:id
		// agar Gin tidak menganggap path segment tersebut sebagai ID.
		protected.GET("event-bookings", middleware.RequirePermission("bookings.view"), middleware.ScopeStoreQuery(), eventCtrl.GetAll)
		protected.POST("event-bookings", middleware.RequirePermission("bookings.create"), middleware.RequireStoreInBody(), eventCtrl.Create)
		protected.GET("event-bookings/dashboard", middleware.RequirePermission("bookings.view"), middleware.ScopeStoreQuery(), eventCtrl.GetForDashboard)
		protected.GET("event-bookings/preview-price", middleware.RequirePermission("bookings.create"), middleware.ScopeStoreQuery(), eventCtrl.PreviewPrice)
		protected.GET("event-bookings/:id", middleware.RequirePermission("bookings.view"), middleware.RequireEntityStore("id", middleware.StoreOf("event_bookings")), eventCtrl.GetByID)
		protected.PATCH("event-bookings/:id/cancel", middleware.RequirePermission("bookings.cancel"), middleware.RequireEntityStore("id", middleware.StoreOf("event_bookings")), eventCtrl.Cancel)

		// ── Event Pricing (per store) ─────────────────────────
		// Pakai :id (sama dengan stores/:id) agar tidak conflict di Gin router tree.
		protected.GET("stores/:id/event-price", middleware.RequireStoreParam("id"), eventCtrl.GetEventPrice)
		protected.PUT("stores/:id/event-price", middleware.RequirePermission("pricing.edit"), middleware.RequireStoreParam("id"), eventCtrl.UpsertEventPrice)

		// ── Banners (admin) ───────────────────────────────────
		// Note: rute statis (admin, reorder) didaftarkan SEBELUM /:id
		// agar Gin tidak menganggap "admin" atau "reorder" sebagai ID.
		// GET public banners ada di /public group di bawah.
		protected.GET("banners/admin", middleware.RequirePermission("settings.branches"), bannerCtrl.GetAllAdmin)
		protected.PATCH("banners/reorder", middleware.RequirePermission("settings.branches"), bannerCtrl.Reorder)
		protected.POST("banners", middleware.RequirePermission("settings.branches"), bannerCtrl.Create)
		protected.PUT("banners/:id", middleware.RequirePermission("settings.branches"), bannerCtrl.Update)
		protected.DELETE("banners/:id", middleware.RequirePermission("settings.branches"), bannerCtrl.Delete)
		protected.PATCH("banners/:id/toggle", middleware.RequirePermission("settings.branches"), bannerCtrl.ToggleActive)

		// ── FnB — Admin ───────────────────────────────────────────────
		// Note: rute statis (sync-moka) didaftarkan SEBELUM /:id
		// agar Gin tidak menganggapnya sebagai category/item ID.
		protected.GET("fnb/categories", middleware.RequirePermission("orders_fnb.view"), fnbCtrl.AdminGetCategories)
		protected.POST("fnb/categories", middleware.RequirePermission("orders_fnb.edit"), fnbCtrl.AdminCreateCategory)
		protected.PUT("fnb/categories/:id", middleware.RequirePermission("orders_fnb.edit"), fnbCtrl.AdminUpdateCategory)
		protected.DELETE("fnb/categories/:id", middleware.RequirePermission("orders_fnb.edit"), fnbCtrl.AdminDeleteCategory)
		protected.GET("fnb/items", middleware.RequirePermission("orders_fnb.view"), fnbCtrl.AdminGetItems)
		protected.POST("fnb/items", middleware.RequirePermission("orders_fnb.edit"), fnbCtrl.AdminCreateItem)
		protected.PUT("fnb/items/:id", middleware.RequirePermission("orders_fnb.edit"), fnbCtrl.AdminUpdateItem)
		protected.GET("fnb/orders", middleware.RequirePermission("orders_fnb.view"), middleware.ScopeStoreQuery(), fnbCtrl.AdminGetOrders)
		protected.PUT("fnb/orders/:id/status", middleware.RequirePermission("orders_fnb.edit"), middleware.RequireEntityStore("id", middleware.StoreOf("fnb_orders")), fnbCtrl.AdminUpdateOrderStatus)
		protected.POST("fnb/sync-moka", middleware.RequirePermission("orders_fnb.edit"), fnbCtrl.AdminSyncMoka)
	}

	// ── Customer auth (tidak perlu JWT) ──────────────────────────
	customerAuth := api.Group("/customer")
	{
		customerAuth.POST("/login", authLimiter, customerAppCtrl.Login)
		customerAuth.POST("/forgot-password", authLimiter, customerAppCtrl.ForgotPassword)
		customerAuth.GET("/reset-password/:token/validate", customerAppCtrl.ValidateResetToken)
		customerAuth.POST("/reset-password/:token", authLimiter, customerAppCtrl.ResetPassword)
	}

	// ── Customer protected (perlu customer JWT) ───────────────────
	customerProtected := api.Group("/customer")
	customerProtected.Use(middleware.CustomerAuth())
	{
		customerProtected.GET("/me", customerAppCtrl.Me)
		customerProtected.GET("/room-recommendations", customerAppCtrl.RoomRecommendations)
		customerProtected.POST("/logout", customerAppCtrl.Logout)
		customerProtected.PUT("/profile", customerAppCtrl.UpdateProfile)
		customerProtected.PUT("/change-password", customerAppCtrl.ChangePassword)
		customerProtected.GET("/credits/expiring", customerAppCtrl.CreditsExpiring)

		// ── Customer Bookings ─────────────────────────────────────────────────
		// Note: rute statis (initiate) didaftarkan SEBELUM /:id agar tidak
		// dianggap sebagai hold_id/booking_id oleh Gin.
		customerProtected.POST("/bookings/initiate", bookingCustomerCtrl.InitiateBooking)
		customerProtected.GET("/bookings", bookingCustomerCtrl.GetMyBookings)
		customerProtected.GET("/bookings/:id", bookingCustomerCtrl.GetMyBookingByID)
		// Mock payment — hanya aktif saat XENDIT_SECRET_KEY kosong
		if utils.XenditMockMode() {
			customerProtected.POST("/bookings/:hold_id/mock-confirm", bookingCustomerCtrl.MockConfirm)
		}

		// ── Play Credits Purchase ─────────────────────────────────────────────
		customerProtected.POST("/play-credits/purchase/initiate", customerAppCtrl.InitiatePlayCreditsPurchase)
		if utils.XenditMockMode() {
			customerProtected.POST("/play-credits/purchase/:intent_id/mock-confirm", customerAppCtrl.MockConfirmPlayCredits)
		}

		// ── My Data ───────────────────────────────────────────────────────────
		customerProtected.GET("/my-credits", customerAppCtrl.GetMyCredits)
		customerProtected.GET("/vouchers/available", customerAppCtrl.GetMyVouchers)

		// ── Customer Event Booking ────────────────────────────────────────────
		// Note: rute statis (initiate) didaftarkan SEBELUM /:event_id
		customerProtected.POST("/event-bookings/initiate", customerAppCtrl.InitiateEventBooking)
		if utils.XenditMockMode() {
			customerProtected.POST("/event-bookings/:event_id/mock-confirm", customerAppCtrl.MockConfirmEventBooking)
		}

		// ── FnB — Customer ────────────────────────────────────────────────────
		customerProtected.POST("/fnb/orders", fnbCtrl.CustomerCreateOrder)
		customerProtected.GET("/fnb/orders", fnbCtrl.CustomerGetMyOrders)
	}

	// ── Public (tanpa auth, untuk customer web) ───────────────────
	publicGroup := api.Group("/public")
	{
		publicGroup.GET("/banners", bannerCtrl.GetAllPublic)
		publicGroup.GET("/banners/:id", bannerCtrl.GetByID)
		publicGroup.GET("/stores", customerAppCtrl.PublicGetStores)
		publicGroup.GET("/stores/:id", customerAppCtrl.PublicGetStoreByID)
		publicGroup.GET("/room-templates", customerAppCtrl.PublicGetRoomTemplates)
		publicGroup.GET("/room-templates/:id", customerAppCtrl.PublicGetRoomTemplateByID)
		publicGroup.GET("/booking/availability", bookingCustomerCtrl.GetAvailability)
		publicGroup.GET("/booking/quote", bookingCustomerCtrl.GetQuote)
		publicGroup.GET("/booking/slots", customerAppCtrl.GetBookingSlots)
		publicGroup.GET("/play-credits/packages", customerAppCtrl.PublicGetPlayCreditsPackages)
		publicGroup.GET("/event-booking/availability", customerAppCtrl.CheckEventAvailability)
		publicGroup.GET("/fnb/menu", fnbCtrl.GetPublicMenu)
	}

	// ── Xendit Webhook (tanpa auth, verifikasi via x-callback-token) ──────────
	r.POST("/webhook/xendit", bookingCustomerCtrl.XenditWebhook)

	return r
}
