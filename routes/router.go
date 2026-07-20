package routes

import (
	"game_lounge_be/middleware"
	recoveryCtrl "game_lounge_be/modules/admin_recovery/controller"
	authCtrl "game_lounge_be/modules/auth/controller"
	bannerCtrl "game_lounge_be/modules/banner/controller"
	bookingCtrl "game_lounge_be/modules/booking/controller"
	bookingCustomerCtrl "game_lounge_be/modules/customer_booking/controller"
	customerAppCtrl "game_lounge_be/modules/customer_app/controller"
	customerCtrl "game_lounge_be/modules/customer/controller"
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
	api.POST("/admin/admin-recovery/request",          authLimiter, recoveryCtrl.Request)
	api.GET("/admin/admin-recovery/:token/validate",    recoveryCtrl.Validate)
	api.POST("/admin/admin-recovery/:token/reset",      authLimiter, recoveryCtrl.Reset)

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
		protected.GET("roles", roleCtrl.GetAll)
		protected.POST("roles", roleCtrl.Create)
		protected.GET("roles/:id", roleCtrl.GetByID)
		protected.PUT("roles/:id", roleCtrl.Update)
		protected.DELETE("roles/:id", roleCtrl.Delete)

		// Staffs
		protected.GET("staffs", staffCtrl.GetAll)
		protected.POST("staffs", staffCtrl.Create)
		protected.GET("staffs/:id", staffCtrl.GetByID)
		protected.PUT("staffs/:id", staffCtrl.Update)
		protected.DELETE("staffs/:id", staffCtrl.Delete)
		protected.POST("staffs/:id/reset-password", staffCtrl.ResetPassword)

		// Facility Categories
		protected.GET("facility-categories", facilityCtrl.GetAllCategories)
		protected.POST("facility-categories", facilityCtrl.CreateCategory)
		protected.PUT("facility-categories/:id", facilityCtrl.UpdateCategory)
		protected.DELETE("facility-categories/:id", facilityCtrl.DeleteCategory)

		// Facilities
		protected.GET("facilities", facilityCtrl.GetAll)
		protected.POST("facilities", facilityCtrl.Create)
		protected.GET("facilities/:id", facilityCtrl.GetByID)
		protected.PUT("facilities/:id", facilityCtrl.Update)
		protected.DELETE("facilities/:id", facilityCtrl.Delete)

		// Room Templates
		protected.GET("room-templates", roomTemplateCtrl.GetAll)
		protected.POST("room-templates", roomTemplateCtrl.Create)
		protected.GET("room-templates/:id", roomTemplateCtrl.GetByID)
		protected.PUT("room-templates/:id", roomTemplateCtrl.Update)
		protected.DELETE("room-templates/:id", roomTemplateCtrl.Delete)

		// Stores
		// Note: rute statis (operating-hours) didaftarkan SEBELUM /:id
		// supaya Gin tidak mengira "operating-hours" adalah store_id.
		protected.GET("stores", storeCtrl.GetAll)
		protected.POST("stores", storeCtrl.Create)
		protected.GET("stores/operating-hours", storeCtrl.GetOperatingHours)
		protected.GET("stores/:id", storeCtrl.GetByID)
		protected.PUT("stores/:id", storeCtrl.Update)
		protected.DELETE("stores/:id", storeCtrl.Delete)

		// Store Rooms
		protected.PATCH("store-rooms/:id/toggle", storeCtrl.ToggleRoomActive)

		// ── Pricing ───────────────────────────────────────────
		// Note: rute statis (calculate, flash-sales) didaftarkan SEBELUM /:store_id
		// supaya Gin tidak mengira "calculate" adalah store_id.

		// Calculator (POST — tidak bentrok dengan GET /:store_id)
		protected.POST("pricing/calculate", pricingCtrl.Calculate)

		// Flash Sales (tanpa store_id prefix — untuk PUT/DELETE by flash-sale ID)
		protected.PUT("pricing/flash-sales/:id", pricingCtrl.UpdateFlashSale)
		protected.DELETE("pricing/flash-sales/:id", pricingCtrl.DeleteFlashSale)

		// Pricing config
		protected.GET("pricing", pricingCtrl.GetAll)
		protected.POST("pricing", pricingCtrl.Create)
		protected.GET("pricing/:store_id", pricingCtrl.GetByStore)
		protected.PUT("pricing/:store_id", pricingCtrl.UpdateConfig)
		protected.DELETE("pricing/:store_id", pricingCtrl.Delete)

		// Happy Hour — Schedules
		protected.POST("pricing/:store_id/happy-hour/schedules", pricingCtrl.AddSchedule)
		protected.DELETE("pricing/:store_id/happy-hour/schedules/:id", pricingCtrl.DeleteSchedule)

		// Happy Hour — Prices
		protected.GET("pricing/:store_id/happy-hour/prices", pricingCtrl.GetHappyHourPrices)
		protected.PUT("pricing/:store_id/happy-hour/prices", pricingCtrl.UpsertHappyHourPrices)

		// Package Prices
		protected.GET("pricing/:store_id/packages", pricingCtrl.GetPackagePrices)
		protected.PUT("pricing/:store_id/packages", pricingCtrl.UpsertPackagePrices)
		protected.DELETE("pricing/:store_id/packages/:id", pricingCtrl.DeletePackagePrice)

		// Flash Sales (list + create dengan store_id)
		protected.GET("pricing/:store_id/flash-sales", pricingCtrl.GetFlashSales)
		protected.POST("pricing/:store_id/flash-sales", pricingCtrl.CreateFlashSale)

		// ── Play Credits — Packages ───────────────────────────
		// Note: "active" didaftarkan sebelum /:id agar tidak dianggap sebagai ID.
		protected.GET("play-credits/packages", playCreditsCtrl.GetAllPackages)
		protected.GET("play-credits/packages/active", playCreditsCtrl.GetActivePackages)
		protected.POST("play-credits/packages", playCreditsCtrl.CreatePackage)
		protected.GET("play-credits/packages/:id", playCreditsCtrl.GetPackageByID)
		protected.PUT("play-credits/packages/:id", playCreditsCtrl.UpdatePackage)
		protected.DELETE("play-credits/packages/:id", playCreditsCtrl.DeletePackage)

		// ── Play Credits — Member Credits ─────────────────────
		protected.GET("play-credits/members", playCreditsCtrl.GetAllMemberCredits)
		protected.POST("play-credits/members", playCreditsCtrl.AssignCredit)
		protected.GET("play-credits/members/:id", playCreditsCtrl.GetCreditByID)
		protected.PATCH("play-credits/members/:id/adjust", playCreditsCtrl.AdjustCredit)
		protected.DELETE("play-credits/members/:id", playCreditsCtrl.DeleteCredit)

		// ── Sales Summary ─────────────────────────────────────
		protected.GET("sales/summary", salesCtrl.GetSummary)
		protected.GET("sales/trend", salesCtrl.GetTrend)
		protected.GET("sales/transactions", salesCtrl.GetTransactions)

		// ── Notification Templates ─────────────────────────────
		// Note: rute statis (preview) didaftarkan SEBELUM /:key.
		protected.GET("notification-templates", ntCtrl.GetAll)
		protected.POST("notification-templates/preview", ntCtrl.Preview)
		protected.GET("notification-templates/:key", ntCtrl.GetByKey)
		protected.PUT("notification-templates/:key", ntCtrl.Update)

		// ── Bookings ──────────────────────────────────────────
		// Note: rute statis (dashboard, sessions-ending-soon, available-credits)
		// didaftarkan SEBELUM /:id agar tidak dianggap sebagai ID oleh Gin.
		protected.GET("bookings", bookingCtrl.GetAll)
		protected.GET("bookings/dashboard", bookingCtrl.GetDashboard)
		protected.GET("bookings/sessions-ending-soon", bookingCtrl.GetSessionsEndingSoon)
		protected.GET("bookings/available-credits", bookingCtrl.GetAvailableCredits)
		protected.POST("bookings", bookingCtrl.Create)
		protected.GET("bookings/:id", bookingCtrl.GetByID)
		protected.PATCH("bookings/:id/cancel", bookingCtrl.Cancel)
		protected.PATCH("bookings/:id/complete", bookingCtrl.Complete)

		// ── Vouchers ──────────────────────────────────────────
		// Note: rute statis didaftarkan SEBELUM /:id
		// agar Gin tidak menganggap path segment tersebut sebagai ID.
		protected.GET("vouchers", voucherCtrl.GetAll)
		protected.GET("vouchers/generate-code", voucherCtrl.GenerateCode)
		protected.GET("vouchers/customer-available", voucherCtrl.GetCustomerAvailable)
		protected.GET("vouchers/recipient-count", voucherCtrl.GetRecipientCount)
		protected.POST("vouchers/validate", voucherCtrl.Validate)
		protected.POST("vouchers", voucherCtrl.Create)
		protected.GET("vouchers/:id", voucherCtrl.GetByID)
		protected.PUT("vouchers/:id", voucherCtrl.Update)
		protected.DELETE("vouchers/:id", voucherCtrl.Delete)

		// ── Customers ─────────────────────────────────────────
		protected.GET("customers", customerCtrl.GetAll)
		protected.POST("customers", customerCtrl.Create)
		protected.GET("customers/:id", customerCtrl.GetByID)
		protected.PUT("customers/:id", customerCtrl.Update)
		protected.PATCH("customers/:id/notes", customerCtrl.UpdateNotes)
		protected.DELETE("customers/:id", customerCtrl.Delete)
		protected.POST("customers/:id/resend-password", customerCtrl.ResendPassword)

		// ── Global Holidays ───────────────────────────────────
		protected.GET("global-holidays", globalHolidayCtrl.GetAll)
		protected.POST("global-holidays", globalHolidayCtrl.Create)
		protected.PUT("global-holidays/:id", globalHolidayCtrl.Update)
		protected.DELETE("global-holidays/:id", globalHolidayCtrl.Delete)

		// ── Event Bookings ────────────────────────────────────
		// Note: rute statis (dashboard, preview-price) didaftarkan SEBELUM /:id
		// agar Gin tidak menganggap path segment tersebut sebagai ID.
		protected.GET("event-bookings", eventCtrl.GetAll)
		protected.POST("event-bookings", eventCtrl.Create)
		protected.GET("event-bookings/dashboard", eventCtrl.GetForDashboard)
		protected.GET("event-bookings/preview-price", eventCtrl.PreviewPrice)
		protected.GET("event-bookings/:id", eventCtrl.GetByID)
		protected.PATCH("event-bookings/:id/cancel", eventCtrl.Cancel)

		// ── Event Pricing (per store) ─────────────────────────
		// Pakai :id (sama dengan stores/:id) agar tidak conflict di Gin router tree.
		protected.GET("stores/:id/event-price", eventCtrl.GetEventPrice)
		protected.PUT("stores/:id/event-price", eventCtrl.UpsertEventPrice)

		// ── Banners (admin) ───────────────────────────────────
		// Note: rute statis (admin, reorder) didaftarkan SEBELUM /:id
		// agar Gin tidak menganggap "admin" atau "reorder" sebagai ID.
		// GET public banners ada di /public group di bawah.
		protected.GET("banners/admin", bannerCtrl.GetAllAdmin)
		protected.PATCH("banners/reorder", bannerCtrl.Reorder)
		protected.POST("banners", bannerCtrl.Create)
		protected.PUT("banners/:id", bannerCtrl.Update)
		protected.DELETE("banners/:id", bannerCtrl.Delete)
		protected.PATCH("banners/:id/toggle", bannerCtrl.ToggleActive)

		// ── FnB — Admin ───────────────────────────────────────────────
		// Note: rute statis (sync-moka) didaftarkan SEBELUM /:id
		// agar Gin tidak menganggapnya sebagai category/item ID.
		protected.GET("fnb/categories",          fnbCtrl.AdminGetCategories)
		protected.POST("fnb/categories",         fnbCtrl.AdminCreateCategory)
		protected.PUT("fnb/categories/:id",      fnbCtrl.AdminUpdateCategory)
		protected.DELETE("fnb/categories/:id",   fnbCtrl.AdminDeleteCategory)
		protected.GET("fnb/items",               fnbCtrl.AdminGetItems)
		protected.POST("fnb/items",              fnbCtrl.AdminCreateItem)
		protected.PUT("fnb/items/:id",           fnbCtrl.AdminUpdateItem)
		protected.GET("fnb/orders",              fnbCtrl.AdminGetOrders)
		protected.PUT("fnb/orders/:id/status",   fnbCtrl.AdminUpdateOrderStatus)
		protected.POST("fnb/sync-moka",          fnbCtrl.AdminSyncMoka)
	}

	// ── Customer auth (tidak perlu JWT) ──────────────────────────
	customerAuth := api.Group("/customer")
	{
		customerAuth.POST("/login",                          authLimiter, customerAppCtrl.Login)
		customerAuth.POST("/forgot-password",                authLimiter, customerAppCtrl.ForgotPassword)
		customerAuth.GET("/reset-password/:token/validate",  customerAppCtrl.ValidateResetToken)
		customerAuth.POST("/reset-password/:token",          authLimiter, customerAppCtrl.ResetPassword)
	}

	// ── Customer protected (perlu customer JWT) ───────────────────
	customerProtected := api.Group("/customer")
	customerProtected.Use(middleware.CustomerAuth())
	{
		customerProtected.GET("/me",                    customerAppCtrl.Me)
		customerProtected.GET("/room-recommendations",  customerAppCtrl.RoomRecommendations)
		customerProtected.POST("/logout",               customerAppCtrl.Logout)
		customerProtected.PUT("/profile",               customerAppCtrl.UpdateProfile)
		customerProtected.PUT("/change-password",       customerAppCtrl.ChangePassword)
		customerProtected.GET("/credits/expiring",      customerAppCtrl.CreditsExpiring)

		// ── Customer Bookings ─────────────────────────────────────────────────
		// Note: rute statis (initiate) didaftarkan SEBELUM /:id agar tidak
		// dianggap sebagai hold_id/booking_id oleh Gin.
		customerProtected.POST("/bookings/initiate",             bookingCustomerCtrl.InitiateBooking)
		customerProtected.GET("/bookings",                       bookingCustomerCtrl.GetMyBookings)
		customerProtected.GET("/bookings/:id",                   bookingCustomerCtrl.GetMyBookingByID)
		// Mock payment — hanya aktif saat XENDIT_SECRET_KEY kosong
		customerProtected.POST("/bookings/:hold_id/mock-confirm",              bookingCustomerCtrl.MockConfirm)

		// ── Play Credits Purchase ─────────────────────────────────────────────
		customerProtected.POST("/play-credits/purchase/initiate",                customerAppCtrl.InitiatePlayCreditsPurchase)
		customerProtected.POST("/play-credits/purchase/:intent_id/mock-confirm", customerAppCtrl.MockConfirmPlayCredits)

		// ── My Data ───────────────────────────────────────────────────────────
		customerProtected.GET("/my-credits",                                     customerAppCtrl.GetMyCredits)
		customerProtected.GET("/vouchers/available",                             customerAppCtrl.GetMyVouchers)

		// ── Customer Event Booking ────────────────────────────────────────────
		// Note: rute statis (initiate) didaftarkan SEBELUM /:event_id
		customerProtected.POST("/event-bookings/initiate",                       customerAppCtrl.InitiateEventBooking)
		customerProtected.POST("/event-bookings/:event_id/mock-confirm",         customerAppCtrl.MockConfirmEventBooking)

		// ── FnB — Customer ────────────────────────────────────────────────────
		customerProtected.POST("/fnb/orders",  fnbCtrl.CustomerCreateOrder)
		customerProtected.GET("/fnb/orders",   fnbCtrl.CustomerGetMyOrders)
	}

	// ── Public (tanpa auth, untuk customer web) ───────────────────
	publicGroup := api.Group("/public")
	{
		publicGroup.GET("/banners",                     bannerCtrl.GetAllPublic)
		publicGroup.GET("/banners/:id",                 bannerCtrl.GetByID)
		publicGroup.GET("/stores",                      customerAppCtrl.PublicGetStores)
		publicGroup.GET("/stores/:id",                  customerAppCtrl.PublicGetStoreByID)
		publicGroup.GET("/room-templates",              customerAppCtrl.PublicGetRoomTemplates)
		publicGroup.GET("/room-templates/:id",          customerAppCtrl.PublicGetRoomTemplateByID)
		publicGroup.GET("/booking/availability",          bookingCustomerCtrl.GetAvailability)
		publicGroup.GET("/booking/slots",                 customerAppCtrl.GetBookingSlots)
		publicGroup.GET("/play-credits/packages",         customerAppCtrl.PublicGetPlayCreditsPackages)
		publicGroup.GET("/event-booking/availability",    customerAppCtrl.CheckEventAvailability)
		publicGroup.GET("/fnb/menu",                      fnbCtrl.GetPublicMenu)
	}

	// ── Xendit Webhook (tanpa auth, verifikasi via x-callback-token) ──────────
	r.POST("/webhook/xendit", bookingCustomerCtrl.XenditWebhook)

	return r
}
