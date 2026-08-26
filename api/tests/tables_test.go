package tests

import (
	"net/http"
	"strings"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestTables(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	t.Run("create list get", func(t *testing.T) {
		created := testutils.CreateTable(t, app, admin.Token, "M1", 4)
		if created.ID == "" {
			t.Fatal("create não retornou id")
		}

		table := testutils.Get[models.Table](t, app, "/admin/tables/"+created.ID, admin.Token)
		if table.TableNumber != "M1" || table.Capacity != 4 || table.Status != models.TableStatusAvailable || table.PublicToken == "" {
			t.Fatalf("GET não bate com o que foi enviado: %+v", table)
		}

		list := testutils.Get[[]models.Table](t, app, "/admin/tables", admin.Token)
		if len(list) < 1 {
			t.Fatal("expected tables in list")
		}
	})

	t.Run("update", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "M2", 2)
		w := app.Request("PUT", "/admin/tables/"+table.ID, map[string]interface{}{
			"table_number": "M2B", "capacity": 6,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.Get[models.Table](t, app, "/admin/tables/"+table.ID, admin.Token)
		if got.TableNumber != "M2B" || got.Capacity != 6 {
			t.Fatalf("GET após update não bate: %+v", got)
		}
	})

	t.Run("occupy and free", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "M3", 2)
		testutils.OccupyTable(t, app, admin.Token, table.ID)

		w := app.Request("GET", "/admin/tables/"+table.ID, nil, admin.Token)
		got := testutils.DecodeJSON[models.Table](t, w)
		if got.Status != models.TableStatusOccupied {
			t.Fatalf("expected occupied, got %s", got.Status)
		}

		w = app.Request("PATCH", "/admin/tables/"+table.ID+"/status", map[string]string{
			"status": string(models.TableStatusAvailable),
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
	})

	t.Run("cannot free table with pending orders", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "M4", 2)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Pedido Mesa", "price_cents": 100, "active": true,
		})
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("PATCH", "/admin/tables/"+table.ID+"/status", map[string]string{
			"status": string(models.TableStatusAvailable),
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "mesa possui pedidos em aberto" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "M5", 2)
		w := app.Request("PATCH", "/admin/tables/"+table.ID+"/status", map[string]string{
			"status": "reserved",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "status inválido" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("qr code", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "M6", 2)
		w := app.Request("GET", "/admin/tables/"+table.ID+"/qr", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		var body struct {
			URL      string `json:"url"`
			QRBase64 string `json:"qr_base64"`
		}
		body = testutils.DecodeJSON[struct {
			URL      string `json:"url"`
			QRBase64 string `json:"qr_base64"`
		}](t, w)
		if !strings.Contains(body.URL, table.PublicToken) || body.QRBase64 == "" {
			t.Fatalf("unexpected qr: %+v", body)
		}
	})

	t.Run("public get table", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "M7", 2)
		w := app.Request("GET", "/public/tables/"+table.PublicToken, nil, "")
		testutils.AssertCode(t, w, http.StatusOK)
		var body struct {
			Table models.Table `json:"table"`
		}
		body = testutils.DecodeJSON[struct {
			Table models.Table `json:"table"`
		}](t, w)
		if body.Table.ID != table.ID {
			t.Fatalf("unexpected public table: %+v", body.Table)
		}

		w = app.Request("GET", "/public/tables/missing-token", nil, "")
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "M8", 2)
		w := app.Request("DELETE", "/admin/tables/"+table.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNoContent)
		w = app.Request("GET", "/admin/tables/"+table.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("get not found", func(t *testing.T) {
		w := app.Request("GET", "/admin/tables/00000000-0000-0000-0000-000000000000", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("cashier cannot create or delete", func(t *testing.T) {
		token := testutils.CashierToken(t, app, admin.Token, testutils.UniqueEmail("cashier-table"))
		w := app.Request("POST", "/admin/tables", map[string]interface{}{
			"table_number": "X", "capacity": 2,
		}, token)
		testutils.AssertCode(t, w, http.StatusForbidden)

		table := testutils.CreateTable(t, app, admin.Token, "M9", 2)
		w = app.Request("GET", "/admin/tables", nil, token)
		testutils.AssertCode(t, w, http.StatusOK)

		w = app.Request("DELETE", "/admin/tables/"+table.ID, nil, token)
		testutils.AssertCode(t, w, http.StatusForbidden)
	})

	t.Run("isolated between companies", func(t *testing.T) {
		other := app.RegisterCompany(t)
		table := testutils.CreateTable(t, app, admin.Token, "M10", 2)
		w := app.Request("GET", "/admin/tables/"+table.ID, nil, other.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := app.Request("GET", "/admin/tables", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})
}
