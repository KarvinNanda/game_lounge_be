package routes

import (
	"game_lounge_be/middleware"
	recoveryCtrl "game_lounge_be/modules/admin_recovery/controller"
	authCtrl "game_lounge_be/modules/auth/controller"
	bookingCtrl "game_lounge_be/modules/booking/controller"
	customerCtrl "game_lounge_be/modules/customer/controller"
	eventCtrl "game_lounge_be/modules/event_booking/controller"
	facilityCtrl "game_lounge_be/modules/facility/controller"
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

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// ── Static file serving ───────────────────────────────────
	r.Static("/assets/img", "./assets/img") // primary asset dir (used by /upload)
	r.Static("/uploads", "./uploads")       // legacy upload dir

	api := r.Group("/api/v1")

	// ── Public endpoints (no auth) ────────────────────────────
	api.POST("/auth/login", authCtrl.Login)
	api.POST("/upload", uploadCtrl.Upload) // file upload, returns path

	// ── Admin Recovery (forgot password super admin) ──────────
	api.POST("/admin-recovery/request",          recoveryCtrl.Request)
	api.GET("/admin-recovery/:token/validate",    recoveryCtrl.Validate)
	api.POST("/admin-recovery/:token/reset",      recoveryCtrl.Reset)

	// ── Protected endpoints ───────────────────────────────────
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Auth
		protected.GET("auth/me", authCtrl.Me)
		protected.POST("auth/logout", authCtrl.Logout)

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
	}

	return r
}
