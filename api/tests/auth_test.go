package tests

import (
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestAuth(t *testing.T) {
	app := testutils.SetupTestApp(t)

	t.Run("register success", func(t *testing.T) {
		email := testutils.UniqueEmail("reg")
		w := app.Request("POST", "/auth/register", map[string]string{
			"name": "Ana", "email": email, "password": "secret123", "role": "admin",
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
		var res struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		}
		res = testutils.DecodeJSON[struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		}](t, w)
		if res.Token == "" || res.User.ID == "" || res.User.Email != email || res.User.Role != models.RoleAdmin {
			t.Fatalf("unexpected register: %+v", res)
		}
		if res.User.PasswordHash != "" {
			t.Fatal("password hash não deve ir na resposta")
		}

		app.Request("POST", "/auth/logout", nil, res.Token)
		w = app.Request("POST", "/auth/login", map[string]string{
			"email": email, "password": "secret123",
		}, "")
		testutils.AssertCode(t, w, http.StatusOK)
		login := testutils.DecodeJSON[struct {
			User models.User `json:"user"`
		}](t, w)
		if login.User.ID != res.User.ID || login.User.Email != email || login.User.Name != "Ana" {
			t.Fatalf("login após register não bate com o usuário criado: %+v", login.User)
		}
	})

	t.Run("register cashier", func(t *testing.T) {
		w := app.Request("POST", "/auth/register", map[string]string{
			"name": "Caixa", "email": testutils.UniqueEmail("cash"), "password": "secret123", "role": "cashier",
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
	})

	t.Run("register missing fields", func(t *testing.T) {
		w := app.Request("POST", "/auth/register", map[string]string{
			"name": "X", "email": "", "password": "secret123", "role": "admin",
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "dados obrigatórios ausentes" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("register requires role", func(t *testing.T) {
		w := app.Request("POST", "/auth/register", map[string]string{
			"name": "X", "email": testutils.UniqueEmail("norole"), "password": "secret123",
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "role é obrigatória" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("register rejects dev role", func(t *testing.T) {
		w := app.Request("POST", "/auth/register", map[string]string{
			"name": "Dev", "email": testutils.UniqueEmail("dev"), "password": "secret123", "role": "dev",
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "role inválida; use admin ou cashier" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("register duplicate email", func(t *testing.T) {
		email := testutils.UniqueEmail("dup")
		w := app.Request("POST", "/auth/register", map[string]string{
			"name": "A", "email": email, "password": "secret123", "role": "admin",
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
		w = app.Request("POST", "/auth/register", map[string]string{
			"name": "B", "email": email, "password": "secret123", "role": "admin",
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "já existe um usuário com este e-mail" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("login success", func(t *testing.T) {
		email := testutils.UniqueEmail("login")
		w := app.Request("POST", "/auth/register", map[string]string{
			"name": "Login", "email": email, "password": "secret123", "role": "admin",
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
		token := testutils.DecodeJSON[struct {
			Token string `json:"token"`
		}](t, w).Token
		app.Request("POST", "/auth/logout", nil, token)

		w = app.Request("POST", "/auth/login", map[string]string{
			"email": email, "password": "secret123",
		}, "")
		testutils.AssertCode(t, w, http.StatusOK)
		var res struct {
			Token string `json:"token"`
		}
		res = testutils.DecodeJSON[struct {
			Token string `json:"token"`
		}](t, w)
		if res.Token == "" {
			t.Fatal("expected token")
		}
	})

	t.Run("login wrong password", func(t *testing.T) {
		email := testutils.UniqueEmail("badpwd")
		app.Request("POST", "/auth/register", map[string]string{
			"name": "X", "email": email, "password": "secret123", "role": "admin",
		}, "")
		w := app.Request("POST", "/auth/login", map[string]string{
			"email": email, "password": "wrong",
		}, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
		if testutils.ResponseError(t, w) != "credenciais inválidas" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("login unknown email", func(t *testing.T) {
		w := app.Request("POST", "/auth/login", map[string]string{
			"email": "nobody@test.com", "password": "secret123",
		}, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("login invalid payload", func(t *testing.T) {
		w := app.Request("POST", "/auth/login", "x", "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("logout invalidates token", func(t *testing.T) {
		admin := app.RegisterCompany(t)
		w := app.Request("POST", "/auth/logout", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)

		w = app.Request("GET", "/admin/users", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("admin without token", func(t *testing.T) {
		w := app.Request("GET", "/admin/users", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("logout without token", func(t *testing.T) {
		w := app.Request("POST", "/auth/logout", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})
}
