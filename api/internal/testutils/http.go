package testutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"reservation/api/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func Get[T any](t *testing.T, app *TestApp, path, token string) T {
	t.Helper()
	w := app.Request("GET", path, nil, token)
	AssertCode(t, w, http.StatusOK)
	return DecodeJSON[T](t, w)
}

func AssertCode(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status %d, want %d: %s", w.Code, want, w.Body.String())
	}
}

func ResponseError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return DecodeJSON[struct {
		Error string `json:"error"`
	}](t, w).Error
}

func Login(t *testing.T, app *TestApp, email, password string) string {
	t.Helper()
	w := app.Request("POST", "/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	AssertCode(t, w, http.StatusOK)
	return DecodeJSON[struct {
		Token string `json:"token"`
	}](t, w).Token
}

func CreateUser(t *testing.T, app *TestApp, token string, body map[string]interface{}) models.User {
	t.Helper()
	w := app.Request("POST", "/admin/users", body, token)
	AssertCode(t, w, http.StatusCreated)
	return DecodeJSON[models.User](t, w)
}

func CashierToken(t *testing.T, app *TestApp, adminToken, email string) string {
	t.Helper()
	CreateUser(t, app, adminToken, map[string]interface{}{
		"name": "Caixa", "email": email, "password": "secret123", "role": models.RoleCashier,
	})
	return Login(t, app, email, "secret123")
}

func SeedDevToken(t *testing.T, app *TestApp, companyID string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	mail := fmt.Sprintf("seed-dev-%s@test.com", companyID[:8])
	companyIDCopy := companyID
	user := models.User{
		Name:         "Seed Dev",
		Email:        mail,
		PasswordHash: string(hash),
		Role:         models.RoleDev,
		CompanyID:    &companyIDCopy,
	}
	if err := app.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return Login(t, app, mail, "secret123")
}

func CreateTable(t *testing.T, app *TestApp, token, number string, capacity int) models.Table {
	t.Helper()
	w := app.Request("POST", "/admin/tables", map[string]interface{}{
		"table_number": number, "capacity": capacity,
	}, token)
	AssertCode(t, w, http.StatusCreated)
	return DecodeJSON[models.Table](t, w)
}

func CreateMenuItem(t *testing.T, app *TestApp, token string, body map[string]interface{}) models.MenuItem {
	t.Helper()
	w := app.Request("POST", "/admin/menu-items", body, token)
	AssertCode(t, w, http.StatusCreated)
	return DecodeJSON[models.MenuItem](t, w)
}

func OccupyTable(t *testing.T, app *TestApp, token, tableID string) {
	t.Helper()
	w := app.Request("PATCH", "/admin/tables/"+tableID+"/status", map[string]string{
		"status": string(models.TableStatusOccupied),
	}, token)
	AssertCode(t, w, http.StatusOK)
}

func AddPublicOrder(t *testing.T, app *TestApp, table models.Table, menuID string, qty int) {
	t.Helper()
	w := app.Request("POST", "/public/tables/"+table.PublicToken+"/table-orders", map[string]interface{}{
		"guest_name": "Cliente",
		"items":      []map[string]interface{}{{"menu_item_id": menuID, "quantity": qty}},
	}, "")
	AssertCode(t, w, http.StatusCreated)
}

func UpdateSchedule(t *testing.T, app *TestApp, token string, body map[string]interface{}) {
	t.Helper()
	w := app.Request("PATCH", "/admin/company/schedule", body, token)
	AssertCode(t, w, http.StatusOK)
}

func TomorrowAt(hour, min int) time.Time {
	slot := time.Now().Add(24 * time.Hour)
	return time.Date(slot.Year(), slot.Month(), slot.Day(), hour, min, 0, 0, slot.Location())
}

func UniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@test.com", prefix, time.Now().UnixNano())
}

func RegisterUser(t *testing.T, app *TestApp, name, email, password, role string) RegisterResult {
	t.Helper()
	w := app.Request("POST", "/auth/register", map[string]string{
		"name": name, "email": email, "password": password, "role": role,
	}, "")
	AssertCode(t, w, http.StatusCreated)
	return DecodeJSON[RegisterResult](t, w)
}

func CompanyPayload(name string) map[string]interface{} {
	doc := fmt.Sprintf("%014d", time.Now().UnixNano()%100000000000000)
	return map[string]interface{}{
		"name":          name,
		"document_type": "cnpj",
		"document":      doc,
		"email":         UniqueEmail("co"),
		"phone":         "11999999999",
		"address": map[string]string{
			"zip_code":     "01310100",
			"street":       "Av Paulista",
			"number":       "1000",
			"neighborhood": "Bela Vista",
			"city":         "São Paulo",
			"state":        "SP",
		},
	}
}
