package tests

import (
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestWaitlist(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	var company models.Company
	if err := app.DB.First(&company, "id = ?", admin.Company.ID).Error; err != nil {
		t.Fatal(err)
	}

	t.Run("join and status", func(t *testing.T) {
		w := app.Request("POST", "/public/waitlist/"+company.WaitlistToken, map[string]interface{}{
			"guest_name": "Ana", "party_size": 2, "notify_via": "email", "contact": "ana@test.com",
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
		var first struct {
			Entry struct {
				ID string `json:"id"`
			} `json:"entry"`
			Position int `json:"position"`
		}
		first = testutils.DecodeJSON[struct {
			Entry struct {
				ID string `json:"id"`
			} `json:"entry"`
			Position int `json:"position"`
		}](t, w)
		if first.Position != 1 || first.Entry.ID == "" {
			t.Fatalf("unexpected join: %+v", first)
		}

		status := testutils.Get[struct {
			Entry    models.WaitlistEntry `json:"entry"`
			Position int                  `json:"position"`
		}](t, app, "/public/waitlist/"+company.WaitlistToken+"/"+first.Entry.ID, "")
		if status.Entry.GuestName != "Ana" || status.Entry.PartySize != 2 || status.Entry.Contact != "ana@test.com" || status.Position != 1 {
			t.Fatalf("GET não bate com o que foi enviado: %+v", status)
		}

		w = app.Request("POST", "/public/waitlist/"+company.WaitlistToken, map[string]interface{}{
			"guest_name": "Beto", "party_size": 2, "notify_via": "email", "contact": "beto@test.com",
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
		var second struct {
			Position int `json:"position"`
		}
		second = testutils.DecodeJSON[struct {
			Position int `json:"position"`
		}](t, w)
		if second.Position != 2 {
			t.Fatalf("expected position 2, got %d", second.Position)
		}

		w = app.Request("GET", "/public/waitlist/"+company.WaitlistToken+"/"+first.Entry.ID, nil, "")
		testutils.AssertCode(t, w, http.StatusOK)
	})

	t.Run("admin list and update", func(t *testing.T) {
		w := app.Request("GET", "/admin/waitlist", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		entries := testutils.DecodeJSON[[]models.WaitlistEntry](t, w)
		if len(entries) == 0 {
			t.Fatal("expected waitlist entries")
		}

		w = app.Request("PATCH", "/admin/waitlist/"+entries[0].ID, map[string]interface{}{
			"status": models.WaitlistStatusCalled,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		updated := testutils.Get[[]models.WaitlistEntry](t, app, "/admin/waitlist", admin.Token)
		found := false
		for _, e := range updated {
			if e.ID == entries[0].ID {
				found = true
				if e.Status != models.WaitlistStatusCalled || e.CalledAt == nil {
					t.Fatalf("GET após update não bate: %+v", e)
				}
			}
		}
		if !found {
			t.Fatal("entrada não apareceu no GET da fila após update")
		}
	})

	t.Run("join invalid token", func(t *testing.T) {
		w := app.Request("POST", "/public/waitlist/missing", map[string]interface{}{
			"guest_name": "X", "party_size": 2, "notify_via": "email", "contact": "x@test.com",
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("status not found", func(t *testing.T) {
		w := app.Request("GET", "/public/waitlist/"+company.WaitlistToken+"/00000000-0000-0000-0000-000000000000", nil, "")
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("invalid payload", func(t *testing.T) {
		w := app.Request("POST", "/public/waitlist/"+company.WaitlistToken, "x", "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("cashier cannot manage waitlist", func(t *testing.T) {
		token := testutils.CashierToken(t, app, admin.Token, testutils.UniqueEmail("cashier-wait"))
		w := app.Request("GET", "/admin/waitlist", nil, token)
		testutils.AssertCode(t, w, http.StatusForbidden)
	})

	t.Run("freeing table calls next guest", func(t *testing.T) {
		other := app.RegisterCompany(t)
		var otherCompany models.Company
		if err := app.DB.First(&otherCompany, "id = ?", other.Company.ID).Error; err != nil {
			t.Fatal(err)
		}
		testutils.CreateTable(t, app, other.Token, "W1", 4)

		w := app.Request("POST", "/public/waitlist/"+otherCompany.WaitlistToken, map[string]interface{}{
			"guest_name": "Carla", "party_size": 2, "notify_via": "email", "contact": "carla@test.com",
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)

		table := testutils.CreateTable(t, app, other.Token, "W2", 4)
		testutils.OccupyTable(t, app, other.Token, table.ID)
		w = app.Request("PATCH", "/admin/tables/"+table.ID+"/status", map[string]string{
			"status": string(models.TableStatusAvailable),
		}, other.Token)
		testutils.AssertCode(t, w, http.StatusOK)

		if len(app.Notifier.Emails) == 0 {
			t.Fatal("expected email when waitlist guest is called")
		}

		w = app.Request("GET", "/admin/waitlist", nil, other.Token)
		entries := testutils.DecodeJSON[[]models.WaitlistEntry](t, w)
		called := false
		for _, e := range entries {
			if e.GuestName == "Carla" && e.Status == models.WaitlistStatusCalled {
				called = true
			}
		}
		if !called {
			t.Fatal("expected Carla to be called")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := app.Request("GET", "/admin/waitlist", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})
}
