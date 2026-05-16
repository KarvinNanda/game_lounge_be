package routes

import (
	"game_lounge_be/middleware"
	authCtrl "game_lounge_be/modules/auth/controller"
	bookingCtrl "game_lounge_be/modules/booking/controller"
	customerCtrl "game_lounge_be/modules/customer/controller"
	facilityCtrl "game_lounge_be/modules/facility/controller"
	voucherCtrl "game_lounge_be/modules/voucher/controller"
	playCreditsCtrl "game_lounge_be/modules/play_credits/controller"
	pricingCtrl "game_lounge_be/modules/pricing/controller"
	salesCtrl "game_lounge_be/modules/sales/controller"
	roleCtrl "game_lounge_be/modules/role/controller"
	roomTemplateCtrl "game_lounge_be/modules/room_template/controller"
	staffCtrl "game_lounge_be/modules/staff/controller"
	storeCtrl "game_lounge_be/modules/store/controller"
	uploadCtrl "game_lounge_be/modules/upload/controller"

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
		protected.GET("stores", storeCtrl.GetAll)
		protected.POST("stores", storeCtrl.Create)
		protected.GET("stores/:id", storeCtrl.GetByID)
		protected.PUT("stores/:id", storeCtrl.Update)
		protected.DELETE("stores/:id", storeCtrl.Delete)

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
		// Note: "generate-code" & "validate" didaftarkan SEBELUM /:id
		// agar Gin tidak menganggap path segment tersebut sebagai ID.
		protected.GET("vouchers", voucherCtrl.GetAll)
		protected.GET("vouchers/generate-code", voucherCtrl.GenerateCode)
		protected.GET("vouchers/customer-available", voucherCtrl.GetCustomerAvailable)
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
	}

	return r
}
