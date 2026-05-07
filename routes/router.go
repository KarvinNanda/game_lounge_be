package routes

import (
	"game_lounge_be/middleware"
	authCtrl "game_lounge_be/modules/auth/controller"
	facilityCtrl "game_lounge_be/modules/facility/controller"
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
	}

	return r
}
