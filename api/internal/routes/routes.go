package routes

import (
	"reservation/api/internal/controllers"
	"reservation/api/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Auth              *controllers.AuthController
	User              *controllers.UserController
	Company           *controllers.CompanyController
	Table             *controllers.TableController
	MenuItem          *controllers.MenuItemController
	Reservation       *controllers.ReservationController
	TableOrder        *controllers.TableOrderController
	Payment           *controllers.PaymentController
	PublicTable       *controllers.PublicTableController
	PublicReservation *controllers.PublicReservationController
	PublicWaitlist    *controllers.PublicWaitlistController
	SystemEvent       *controllers.SystemEventController
	Waitlist          *controllers.WaitlistController
	AuthMiddleware    *middleware.AuthMiddleware
}

func Register(r *gin.Engine, h Handlers) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	public := r.Group("/public")
	admin := r.Group("/admin")
	admin.Use(h.AuthMiddleware.RequireAuth())

	registerAuth(r, h)
	registerCompany(admin, h)

	staff := admin.Group("")
	staff.Use(middleware.RequireCompany())
	registerUser(staff, h)
	registerTable(staff, public, h)
	registerMenuItem(staff, h)
	registerReservation(staff, public, h)
	registerTableOrder(staff, h)
	registerPayment(staff, h)
	registerWaitlist(staff, public, h)
	registerSystemEvent(staff, h)
}
