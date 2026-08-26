package tests

import (
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestSystemEvents(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	t.Run("list events after user created", func(t *testing.T) {
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Setup Event", "email": testutils.UniqueEmail("ev-setup"), "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("GET", "/admin/system-events", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		events := testutils.DecodeJSON[[]models.SystemEvent](t, w)
		if len(events) == 0 {
			t.Fatal("expected system events")
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Event User", "email": testutils.UniqueEmail("ev-user"), "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("GET", "/admin/system-events?type=user_created", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		events := testutils.DecodeJSON[[]models.SystemEvent](t, w)
		if len(events) == 0 {
			t.Fatal("expected user_created events")
		}
		for _, e := range events {
			if e.Type != "user_created" {
				t.Fatalf("expected user_created, got %s", e.Type)
			}
		}
	})

	t.Run("filter by actor type", func(t *testing.T) {
		table := testutils.CreateTable(t, app, admin.Token, "E1", 2)
		menu := testutils.CreateMenuItem(t, app, admin.Token, map[string]interface{}{
			"name": "Evento Item", "price_cents": 100, "active": true,
		})
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("GET", "/admin/system-events?actor_type=client", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		events := testutils.DecodeJSON[[]models.SystemEvent](t, w)
		found := false
		for _, e := range events {
			if e.ActorType == models.ActorTypeClient {
				found = true
			}
		}
		if !found {
			t.Fatal("expected client actor event")
		}
	})

	t.Run("cashier cannot list events", func(t *testing.T) {
		token := testutils.CashierToken(t, app, admin.Token, testutils.UniqueEmail("cashier-ev"))
		w := app.Request("GET", "/admin/system-events", nil, token)
		testutils.AssertCode(t, w, http.StatusForbidden)
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := app.Request("GET", "/admin/system-events", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("isolated between companies", func(t *testing.T) {
		other := app.RegisterCompany(t)
		w := app.Request("GET", "/admin/system-events?type=user_created", nil, other.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		events := testutils.DecodeJSON[[]models.SystemEvent](t, w)
		for _, e := range events {
			if e.CompanyID != nil && *e.CompanyID == admin.Company.ID {
				t.Fatal("other company should not see these events")
			}
		}
	})
}
