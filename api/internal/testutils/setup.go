package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"reservation/api/internal/config"
	"reservation/api/internal/controllers"
	"reservation/api/internal/database"
	"reservation/api/internal/middleware"
	"reservation/api/internal/notify"
	"reservation/api/internal/routes"
	"reservation/api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TestApp struct {
	DB       *gorm.DB
	Router   *gin.Engine
	Notifier *notify.FakeNotifier
	Config   config.Config
}

func SetupTestApp(t *testing.T) *TestApp {
	t.Helper()
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	pg, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = pg.Terminate(ctx)
	})

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := gorm.Open(postgresdriver.Open(connStr), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := config.Config{
		JWTSecret:                  "test-secret",
		AppPublicURL:               "http://localhost:8080",
		WaitlistCallTimeoutMinutes: 10,
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
	fakeNotifier := &notify.FakeNotifier{}
	waitlistSvc := services.NewWaitlistService(db, cfg, fakeNotifier, events, tableSvc)
	authMw := middleware.NewAuthMiddleware(db, cfg.JWTSecret)

	r := gin.New()
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

	return &TestApp{DB: db, Router: r, Notifier: fakeNotifier, Config: cfg}
}

func (a *TestApp) Request(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	return w
}

type RegisterResult struct {
	Token   string `json:"token"`
	Company struct {
		ID string `json:"id"`
	} `json:"company"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
}

func (a *TestApp) RegisterCompany(t *testing.T) RegisterResult {
	t.Helper()
	email := fmt.Sprintf("admin-%d@test.com", time.Now().UnixNano())
	w := a.Request("POST", "/auth/register", map[string]string{
		"name":     "Admin",
		"email":    email,
		"password": "secret123",
		"role":     "admin",
	}, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", w.Code, w.Body.String())
	}
	var res RegisterResult
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	doc := fmt.Sprintf("%014d", time.Now().UnixNano()%100000000000000)
	w = a.Request("POST", "/admin/company", map[string]interface{}{
		"name":          "Test Co",
		"document_type": "cnpj",
		"document":      doc,
		"email":         fmt.Sprintf("co-%d@test.com", time.Now().UnixNano()),
		"phone":         "11999999999",
		"address": map[string]string{
			"zip_code":     "01310100",
			"street":       "Av Paulista",
			"number":       "1000",
			"neighborhood": "Bela Vista",
			"city":         "São Paulo",
			"state":        "SP",
		},
	}, res.Token)
	if w.Code != http.StatusCreated {
		t.Fatalf("create company status %d: %s", w.Code, w.Body.String())
	}
	var company struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &company)
	res.Company.ID = company.ID
	return res
}

func DecodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode json: %v body=%s", err, w.Body.String())
	}
	return v
}
