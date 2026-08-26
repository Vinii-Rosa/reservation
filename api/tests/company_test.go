package tests

import (
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestCompany(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	t.Run("get company", func(t *testing.T) {
		w := app.Request("GET", "/admin/company", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		company := testutils.DecodeJSON[models.Company](t, w)
		if company.ID != admin.Company.ID || company.Name == "" {
			t.Fatalf("unexpected company: %+v", company)
		}
	})

	t.Run("update name", func(t *testing.T) {
		w := app.Request("PATCH", "/admin/company", map[string]interface{}{
			"name": "Novo Nome",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		company := testutils.Get[models.Company](t, app, "/admin/company", admin.Token)
		if company.Name != "Novo Nome" {
			t.Fatalf("GET após update não bate: %+v", company)
		}
	})

	t.Run("user already in company cannot create another", func(t *testing.T) {
		w := app.Request("POST", "/admin/company", testutils.CompanyPayload("Outra"), admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "usuário já está vinculado a uma companhia" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("duplicate document", func(t *testing.T) {
		w := app.Request("GET", "/admin/company", nil, admin.Token)
		existing := testutils.DecodeJSON[models.Company](t, w)
		other := testutils.RegisterUser(t, app, "Other", testutils.UniqueEmail("other-co"), "secret123", "admin")
		payload := testutils.CompanyPayload("Dup Doc")
		payload["document"] = existing.Document
		w = app.Request("POST", "/admin/company", payload, other.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "já existe uma companhia com este CPF/CNPJ" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("invalid cnpj", func(t *testing.T) {
		user := testutils.RegisterUser(t, app, "Bad", testutils.UniqueEmail("bad-cnpj"), "secret123", "admin")
		payload := testutils.CompanyPayload("Bad CNPJ")
		payload["document"] = "123"
		w := app.Request("POST", "/admin/company", payload, user.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "CNPJ deve ter 14 dígitos" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("incomplete address", func(t *testing.T) {
		user := testutils.RegisterUser(t, app, "Addr", testutils.UniqueEmail("addr"), "secret123", "admin")
		w := app.Request("POST", "/admin/company", map[string]interface{}{
			"name": "X", "document_type": "cnpj", "document": "11222333000181",
			"email": testutils.UniqueEmail("addr-co"), "phone": "11999999999",
			"address": map[string]string{"street": "Rua"},
		}, user.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("list and update configs", func(t *testing.T) {
		w := app.Request("GET", "/admin/company/configs", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		list := testutils.DecodeJSON[[]map[string]interface{}](t, w)
		if len(list) == 0 {
			t.Fatal("expected configs")
		}

		w = app.Request("PATCH", "/admin/company/configs/allow_larger_tables", map[string]interface{}{
			"value": true,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)

		w = app.Request("PATCH", "/admin/company/configs/unknown_key", map[string]interface{}{
			"value": true,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "configuração desconhecida" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("cashier cannot update company", func(t *testing.T) {
		token := testutils.CashierToken(t, app, admin.Token, testutils.UniqueEmail("cashier-co"))
		w := app.Request("PATCH", "/admin/company", map[string]interface{}{
			"name": "Hack",
		}, token)
		testutils.AssertCode(t, w, http.StatusForbidden)
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := app.Request("GET", "/admin/company", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})
}
