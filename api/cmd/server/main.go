package main

import (
	"log"
	"os"

	"reservation/api/internal/config"
	"reservation/api/internal/controllers"
	"reservation/api/internal/database"
	"reservation/api/internal/jobs"
	"reservation/api/internal/middleware"
	"reservation/api/internal/notify"
	"reservation/api/internal/routes"
	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	events := services.NewSystemEventService(db)
	authSvc := services.NewAuthService(db, cfg.JWTSecret, events)
	userSvc := services.NewUserService(db, events)
	companySvc := services.NewCompanyService(db, events)
	tableSvc := services.NewTableService(db, cfg, events)
	menuSvc := services.NewMenuItemService(db, events)
	reservationSvc := services.NewReservationService(db, events)
	tableOrderSvc := services.NewTableOrderService(db, events)
	paymentSvc := services.NewPaymentService(db, events)
	notifier := notify.NewSMTPNotifier(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	waitlistSvc := services.NewWaitlistService(db, cfg, notifier, events, tableSvc)

	cleanup := jobs.NewCleanupJob(db, reservationSvc)
	c := cron.New()
	if _, err := c.AddFunc(cfg.CronCleanupReservations, cleanup.Run); err != nil {
		log.Fatalf("cron: %v", err)
	}
	c.Start()

	authMw := middleware.NewAuthMiddleware(db, cfg.JWTSecret)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	routes.Register(r, routes.Handlers{
		Auth:           controllers.NewAuthController(authSvc),
		User:           controllers.NewUserController(userSvc),
		Company:        controllers.NewCompanyController(companySvc),
		Table:          controllers.NewTableController(tableSvc, waitlistSvc),
		MenuItem:       controllers.NewMenuItemController(menuSvc),
		Reservation:    controllers.NewReservationController(reservationSvc),
		TableOrder:     controllers.NewTableOrderController(tableOrderSvc),
		Payment:        controllers.NewPaymentController(paymentSvc),
		PublicTable:       controllers.NewPublicTableController(tableSvc, tableOrderSvc),
		PublicReservation: controllers.NewPublicReservationController(reservationSvc),
		PublicWaitlist:    controllers.NewPublicWaitlistController(waitlistSvc),
		SystemEvent:    controllers.NewSystemEventController(events),
		Waitlist:       controllers.NewWaitlistController(waitlistSvc),
		AuthMiddleware: authMw,
	})

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
