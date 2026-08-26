package tests

import (
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestMenuItems(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	t.Run("create success", func(t *testing.T) {
		w := app.Request("POST", "/admin/menu-items", map[string]interface{}{
			"name": "Cerveja", "description": "350ml", "price_cents": 1200, "active": true,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.MenuItem](t, w)
		if created.ID == "" {
			t.Fatal("create não retornou id")
		}

		item := testutils.Get[models.MenuItem](t, app, "/admin/menu-items/"+created.ID, admin.Token)
		if item.Name != "Cerveja" || item.Description != "350ml" || item.PriceCents != 1200 || !item.Active {
			t.Fatalf("GET não bate com o que foi enviado: %+v", item)
		}
		if item.CompanyID != admin.Company.ID {
			t.Fatalf("expected company %s, got %s", admin.Company.ID, item.CompanyID)
		}
	})

	t.Run("list includes items ordered by name", func(t *testing.T) {
		testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Zebra", "price_cents": 100, "active": true,
		})
		testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Azeitona", "price_cents": 200, "active": true,
		})

		w := app.Request("GET", "/admin/menu-items", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		items := testutils.DecodeJSON[[]models.MenuItem](t, w)
		if len(items) < 2 {
			t.Fatalf("expected at least 2 items, got %d", len(items))
		}
		for i := 1; i < len(items); i++ {
			if items[i-1].Name > items[i].Name {
				t.Fatalf("list not ordered by name: %s then %s", items[i-1].Name, items[i].Name)
			}
		}
	})

	t.Run("get by id", func(t *testing.T) {
		created := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Get Me", "description": "item", "price_cents": 500, "active": true,
		})
		w := app.Request("GET", "/admin/menu-items/"+created.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.DecodeJSON[models.MenuItem](t, w)
		if got.ID != created.ID || got.Name != "Get Me" {
			t.Fatalf("unexpected get: %+v", got)
		}
	})

	t.Run("update price keeps other fields", func(t *testing.T) {
		created := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Porção", "description": "grande", "price_cents": 1200, "active": true,
		})
		w := app.Request("PUT", "/admin/menu-items/"+created.ID, map[string]interface{}{
			"price_cents": 1500,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.Get[models.MenuItem](t, app, "/admin/menu-items/"+created.ID, admin.Token)
		if got.PriceCents != 1500 || got.Name != "Porção" || got.Description != "grande" || !got.Active {
			t.Fatalf("GET após update não bate: %+v", got)
		}
	})

	t.Run("update name description and active", func(t *testing.T) {
		created := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Old", "description": "old desc", "price_cents": 100, "active": true,
		})
		w := app.Request("PUT", "/admin/menu-items/"+created.ID, map[string]interface{}{
			"name": "New", "description": "new desc", "active": false,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.Get[models.MenuItem](t, app, "/admin/menu-items/"+created.ID, admin.Token)
		if got.Name != "New" || got.Description != "new desc" || got.Active || got.PriceCents != 100 {
			t.Fatalf("GET após update não bate: %+v", got)
		}
	})

	t.Run("update empty body keeps fields", func(t *testing.T) {
		created := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Keep", "description": "same", "price_cents": 333, "active": true,
		})
		w := app.Request("PUT", "/admin/menu-items/"+created.ID, map[string]interface{}{}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.Get[models.MenuItem](t, app, "/admin/menu-items/"+created.ID, admin.Token)
		if got.Name != "Keep" || got.Description != "same" || got.PriceCents != 333 || !got.Active {
			t.Fatalf("fields changed on empty update: %+v", got)
		}
	})

	t.Run("admin list includes inactive", func(t *testing.T) {
		created := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Inativo Admin", "price_cents": 10, "active": true,
		})
		w := app.Request("PUT", "/admin/menu-items/"+created.ID, map[string]interface{}{
			"active": false,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		w = app.Request("GET", "/admin/menu-items", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		items := testutils.DecodeJSON[[]models.MenuItem](t, w)
		found := false
		for _, it := range items {
			if it.ID == created.ID {
				found = true
				if it.Active {
					t.Fatal("expected inactive item")
				}
			}
		}
		if !found {
			t.Fatal("inactive item missing from admin list")
		}
	})

	t.Run("public table shows only active items", func(t *testing.T) {
		w := app.Request("POST", "/admin/tables", map[string]interface{}{
			"table_number": "MenuPublic", "capacity": 2,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		table := testutils.DecodeJSON[models.Table](t, w)

		active := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Ativo Público", "price_cents": 800, "active": true,
		})
		inactive := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Inativo Público", "price_cents": 900, "active": true,
		})
		w = app.Request("PUT", "/admin/menu-items/"+inactive.ID, map[string]interface{}{
			"active": false,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)

		w = app.Request("GET", "/public/tables/"+table.PublicToken, nil, "")
		testutils.AssertCode(t, w, http.StatusOK)
		var body struct {
			Menu []models.MenuItem `json:"menu"`
		}
		body = testutils.DecodeJSON[struct {
			Menu []models.MenuItem `json:"menu"`
		}](t, w)

		foundActive, foundInactive := false, false
		for _, it := range body.Menu {
			if it.ID == active.ID {
				foundActive = true
			}
			if it.ID == inactive.ID {
				foundInactive = true
			}
		}
		if !foundActive {
			t.Fatal("active item missing from public menu")
		}
		if foundInactive {
			t.Fatal("inactive item should not appear on public menu")
		}
	})

	t.Run("delete item", func(t *testing.T) {
		created := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Apagar", "price_cents": 50, "active": true,
		})
		w := app.Request("DELETE", "/admin/menu-items/"+created.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNoContent)

		w = app.Request("GET", "/admin/menu-items/"+created.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		w := app.Request("POST", "/admin/menu-items", map[string]interface{}{
			"name": "X", "price_cents": 1, "active": true,
		}, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)

		w = app.Request("GET", "/admin/menu-items", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("invalid payload", func(t *testing.T) {
		w := app.Request("POST", "/admin/menu-items", "não é um objeto", admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "payload inválido" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("get not found", func(t *testing.T) {
		w := app.Request("GET", "/admin/menu-items/00000000-0000-0000-0000-000000000000", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
		if testutils.ResponseError(t, w) != "item não encontrado" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("update not found", func(t *testing.T) {
		w := app.Request("PUT", "/admin/menu-items/00000000-0000-0000-0000-000000000000", map[string]interface{}{
			"name": "Nope",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("delete not found", func(t *testing.T) {
		w := app.Request("DELETE", "/admin/menu-items/00000000-0000-0000-0000-000000000000", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("cashier cannot manage menu", func(t *testing.T) {
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Caixa Menu", "email": "cashier-menu@test.com", "password": "secret123", "role": models.RoleCashier,
		})
		token := testutils.Login(t, app, "cashier-menu@test.com", "secret123")
		item := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Só Admin", "price_cents": 100, "active": true,
		})

		w := app.Request("POST", "/admin/menu-items", map[string]interface{}{
			"name": "Blocked", "price_cents": 1, "active": true,
		}, token)
		testutils.AssertCode(t, w, http.StatusForbidden)

		w = app.Request("GET", "/admin/menu-items", nil, token)
		testutils.AssertCode(t, w, http.StatusForbidden)

		w = app.Request("GET", "/admin/menu-items/"+item.ID, nil, token)
		testutils.AssertCode(t, w, http.StatusForbidden)

		w = app.Request("PUT", "/admin/menu-items/"+item.ID, map[string]interface{}{
			"price_cents": 2,
		}, token)
		testutils.AssertCode(t, w, http.StatusForbidden)

		w = app.Request("DELETE", "/admin/menu-items/"+item.ID, nil, token)
		testutils.AssertCode(t, w, http.StatusForbidden)
	})

	t.Run("isolated between companies", func(t *testing.T) {
		other := app.RegisterCompany(t)
		item := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Privado", "price_cents": 111, "active": true,
		})

		w := app.Request("GET", "/admin/menu-items/"+item.ID, nil, other.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)

		w = app.Request("PUT", "/admin/menu-items/"+item.ID, map[string]interface{}{
			"name": "Hack",
		}, other.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)

		w = app.Request("GET", "/admin/menu-items", nil, other.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		list := testutils.DecodeJSON[[]models.MenuItem](t, w)
		for _, it := range list {
			if it.ID == item.ID {
				t.Fatal("other company should not see this item")
			}
		}
	})
}
