package tests

import (
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/services"
	"reservation/api/internal/testutils"
)

func TestTableOrders(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	t.Run("public create order", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P1", 4)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Refrigerante", "price_cents": 800, "active": true,
		})
		testutils.OccupyTable(t, app, admin.Token, table.ID)

		w := app.Request("POST", "/public/tables/"+table.PublicToken+"/table-orders", map[string]interface{}{
			"guest_name": "Maria",
			"items":      []map[string]interface{}{{"menu_item_id": menu.ID, "quantity": 2}},
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.TableOrder](t, w)
		if created.ID == "" {
			t.Fatal("create não retornou id")
		}

		pending := testutils.Get[[]services.TableOrderWithTable](t, app, "/admin/table-orders/pending", admin.Token)
		var order *services.TableOrderWithTable
		for i := range pending {
			if pending[i].ID == created.ID {
				order = &pending[i]
				break
			}
		}
		if order == nil {
			t.Fatal("pedido não apareceu no GET de pendentes")
		}
		if order.Status != models.TableOrderStatusPending || len(order.Items) != 1 {
			t.Fatalf("GET não bate com o enviado: %+v", order)
		}
		if order.Items[0].Quantity != 2 || order.Items[0].UnitPrice != 800 || order.Items[0].MenuItemName != "Refrigerante" {
			t.Fatalf("GET item não bate: %+v", order.Items[0])
		}
	})

	t.Run("list pending and summary", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P2", 4)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Porção", "price_cents": 1500, "active": true,
		})
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 2)

		w := app.Request("GET", "/admin/table-orders/pending", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		pending := testutils.DecodeJSON[[]services.TableOrderWithTable](t, w)
		found := false
		for _, o := range pending {
			if o.TableID == table.ID {
				found = true
				if o.TableNumber != "P2" {
					t.Fatalf("expected table number P2, got %s", o.TableNumber)
				}
			}
		}
		if !found {
			t.Fatal("pending order missing")
		}

		w = app.Request("GET", "/admin/tables/"+table.ID+"/order-summary", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		var summary struct {
			TotalCents int64 `json:"total_cents"`
		}
		summary = testutils.DecodeJSON[struct {
			TotalCents int64 `json:"total_cents"`
		}](t, w)
		if summary.TotalCents != 3000 {
			t.Fatalf("expected total 3000, got %d", summary.TotalCents)
		}
	})

	t.Run("public order occupies available table", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P3", 2)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Água", "price_cents": 400, "active": true,
		})
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("GET", "/admin/tables/"+table.ID, nil, admin.Token)
		got := testutils.DecodeJSON[models.Table](t, w)
		if got.Status != models.TableStatusOccupied {
			t.Fatalf("expected occupied after public order, got %s", got.Status)
		}
	})

	t.Run("empty order", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P4", 2)
		w := app.Request("POST", "/public/tables/"+table.PublicToken+"/table-orders", map[string]interface{}{
			"guest_name": "X", "items": []map[string]interface{}{},
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "pedido vazio" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("invalid menu item", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P5", 2)
		w := app.Request("POST", "/public/tables/"+table.PublicToken+"/table-orders", map[string]interface{}{
			"guest_name": "X",
			"items":      []map[string]interface{}{{"menu_item_id": "00000000-0000-0000-0000-000000000000", "quantity": 1}},
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "item do menu inválido" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("invalid quantity", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P6", 2)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Qtd", "price_cents": 100, "active": true,
		})
		w := app.Request("POST", "/public/tables/"+table.PublicToken+"/table-orders", map[string]interface{}{
			"guest_name": "X",
			"items":      []map[string]interface{}{{"menu_item_id": menu.ID, "quantity": 0}},
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "quantidade inválida" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("inactive menu item", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P7", 2)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Inativo Pedido", "price_cents": 100, "active": true,
		})
		w := app.Request("PUT", "/admin/menu-items/"+menu.ID, map[string]interface{}{
			"active": false,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		w = app.Request("POST", "/public/tables/"+table.PublicToken+"/table-orders", map[string]interface{}{
			"guest_name": "X",
			"items":      []map[string]interface{}{{"menu_item_id": menu.ID, "quantity": 1}},
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "item do menu inválido" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("unknown table token", func(t *testing.T) {
		w := app.Request("POST", "/public/tables/missing/table-orders", map[string]interface{}{
			"guest_name": "X",
			"items":      []map[string]interface{}{{"menu_item_id": "x", "quantity": 1}},
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("summary without session", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "P8", 2)
		w := app.Request("GET", "/admin/tables/"+table.ID+"/order-summary", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		var summary struct {
			TotalCents int64 `json:"total_cents"`
		}
		summary = testutils.DecodeJSON[struct {
			TotalCents int64 `json:"total_cents"`
		}](t, w)
		if summary.TotalCents != 0 {
			t.Fatalf("expected 0, got %d", summary.TotalCents)
		}
	})

	t.Run("unauthorized pending list", func(t *testing.T) {
		w := app.Request("GET", "/admin/table-orders/pending", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("isolated pending between companies", func(t *testing.T) {
		other := app.RegisterCompany(t)
		table := testutils.CreateTable(t, app, admin.Token, "P9", 2)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Iso Pedido", "price_cents": 100, "active": true,
		})
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("GET", "/admin/table-orders/pending", nil, other.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		pending := testutils.DecodeJSON[[]services.TableOrderWithTable](t, w)
		for _, o := range pending {
			if o.TableID == table.ID {
				t.Fatal("other company should not see this order")
			}
		}
	})
}
