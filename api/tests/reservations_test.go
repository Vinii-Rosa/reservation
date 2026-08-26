package tests

import (
	"net/http"
	"testing"
	"time"

	"reservation/api/internal/jobs"
	"reservation/api/internal/models"
	"reservation/api/internal/services"
	"reservation/api/internal/testutils"
)

func TestReservations(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)
	testutils.CreateTable(t, app, admin.Token, "R1", 2)
	testutils.CreateTable(t, app, admin.Token, "R1b", 2)
	testutils.CreateTable(t, app, admin.Token, "R1c", 2)
	slot := testutils.TomorrowAt(18, 0)

	t.Run("public create and get", func(t *testing.T) {
		w := app.Request("POST", "/public/companies/"+admin.Company.ID+"/reservations", map[string]interface{}{
			"guest_name": "João", "guest_contact": "joao@test.com", "party_size": 2, "scheduled_at": slot,
		}, "")
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.Reservation](t, w)
		if created.ID == "" || created.PublicToken == "" {
			t.Fatal("create não retornou id/token")
		}

		got := testutils.Get[models.Reservation](t, app, "/public/reservations/"+created.PublicToken, "")
		if got.ID != created.ID || got.GuestName != "João" || got.GuestContact != "joao@test.com" || got.PartySize != 2 || got.Status != models.ReservationStatusPending {
			t.Fatalf("GET não bate com o que foi enviado: %+v", got)
		}
	})

	t.Run("admin create get list update check-in delete", func(t *testing.T) {
		adminSlot := testutils.TomorrowAt(18, 30)
		w := app.Request("POST", "/admin/reservations", map[string]interface{}{
			"guest_name": "Maria", "guest_contact": "maria@test.com", "party_size": 2, "scheduled_at": adminSlot,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.Reservation](t, w)
		if created.ID == "" {
			t.Fatal("create não retornou id")
		}

		res := testutils.Get[models.Reservation](t, app, "/admin/reservations/"+created.ID, admin.Token)
		if res.GuestName != "Maria" || res.GuestContact != "maria@test.com" || res.PartySize != 2 || res.Status != models.ReservationStatusPending {
			t.Fatalf("GET não bate com o que foi enviado: %+v", res)
		}

		list := testutils.Get[[]models.Reservation](t, app, "/admin/reservations", admin.Token)
		found := false
		for _, r := range list {
			if r.ID == res.ID {
				found = true
			}
		}
		if !found {
			t.Fatal("created reservation missing from list")
		}

		w = app.Request("PUT", "/admin/reservations/"+res.ID, map[string]interface{}{
			"guest_name": "Maria Silva",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		updated := testutils.Get[models.Reservation](t, app, "/admin/reservations/"+res.ID, admin.Token)
		if updated.GuestName != "Maria Silva" {
			t.Fatalf("GET após update não bate: %+v", updated)
		}

		w = app.Request("PATCH", "/admin/reservations/"+res.ID+"/check-in", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		checked := testutils.Get[models.Reservation](t, app, "/admin/reservations/"+res.ID, admin.Token)
		if checked.Status != models.ReservationStatusCompleted {
			t.Fatalf("GET após check-in não bate: %+v", checked)
		}

		w = app.Request("PATCH", "/admin/reservations/"+res.ID+"/check-in", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "só é possível dar baixa em reserva pendente" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}

		otherSlot := testutils.TomorrowAt(19, 0)
		w = app.Request("POST", "/admin/reservations", map[string]interface{}{
			"guest_name": "Apagar", "guest_contact": "a@test.com", "party_size": 2, "scheduled_at": otherSlot,
		}, admin.Token)
		toDelete := testutils.DecodeJSON[models.Reservation](t, w)
		w = app.Request("DELETE", "/admin/reservations/"+toDelete.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNoContent)
		w = app.Request("GET", "/admin/reservations/"+toDelete.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("availability", func(t *testing.T) {
		date := slot.Format("2006-01-02")
		w := app.Request("GET", "/public/companies/"+admin.Company.ID+"/availability?date="+date+"&party_size=2", nil, "")
		testutils.AssertCode(t, w, http.StatusOK)
		slots := testutils.DecodeJSON[[]services.AvailabilitySlot](t, w)
		if len(slots) == 0 {
			t.Fatal("expected availability slots")
		}

		w = app.Request("GET", "/public/companies/"+admin.Company.ID+"/availability?date="+date, nil, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "party_size é obrigatório" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("outside opening hours", func(t *testing.T) {
		w := app.Request("POST", "/public/companies/"+admin.Company.ID+"/reservations", map[string]interface{}{
			"guest_name": "Cedo", "guest_contact": "c@test.com", "party_size": 2, "scheduled_at": testutils.TomorrowAt(10, 0),
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "fora do horário de funcionamento" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("invalid slot", func(t *testing.T) {
		w := app.Request("POST", "/public/companies/"+admin.Company.ID+"/reservations", map[string]interface{}{
			"guest_name": "Slot", "guest_contact": "s@test.com", "party_size": 2, "scheduled_at": testutils.TomorrowAt(18, 10),
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "horário não corresponde a um slot válido" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("no matching tables", func(t *testing.T) {
		w := app.Request("POST", "/public/companies/"+admin.Company.ID+"/reservations", map[string]interface{}{
			"guest_name": "Grupo", "guest_contact": "g@test.com", "party_size": 8, "scheduled_at": testutils.TomorrowAt(20, 0),
		}, "")
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "não há mais mesas com aquela quantidade de lugares" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("public get not found", func(t *testing.T) {
		w := app.Request("GET", "/public/reservations/missing-token", nil, "")
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("cashier cannot create but can list and check-in", func(t *testing.T) {
		token := testutils.CashierToken(t, app, admin.Token, testutils.UniqueEmail("cashier-res"))
		w := app.Request("POST", "/admin/reservations", map[string]interface{}{
			"guest_name": "X", "guest_contact": "x@test.com", "party_size": 2, "scheduled_at": testutils.TomorrowAt(21, 0),
		}, token)
		testutils.AssertCode(t, w, http.StatusForbidden)

		w = app.Request("GET", "/admin/reservations", nil, token)
		testutils.AssertCode(t, w, http.StatusOK)

		create := app.Request("POST", "/admin/reservations", map[string]interface{}{
			"guest_name": "Baixa", "guest_contact": "b@test.com", "party_size": 2, "scheduled_at": testutils.TomorrowAt(21, 30),
		}, admin.Token)
		res := testutils.DecodeJSON[models.Reservation](t, create)
		w = app.Request("PATCH", "/admin/reservations/"+res.ID+"/check-in", nil, token)
		testutils.AssertCode(t, w, http.StatusOK)
	})

	t.Run("isolated between companies", func(t *testing.T) {
		other := app.RegisterCompany(t)
		testutils.CreateTable(t, app, other.Token, "R2", 2)
		w := app.Request("POST", "/admin/reservations", map[string]interface{}{
			"guest_name": "Iso", "guest_contact": "i@test.com", "party_size": 2, "scheduled_at": testutils.TomorrowAt(22, 0),
		}, admin.Token)
		res := testutils.DecodeJSON[models.Reservation](t, w)

		w = app.Request("GET", "/admin/reservations/"+res.ID, nil, other.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("cleanup expired pending", func(t *testing.T) {
		past := models.Reservation{
			CompanyID: admin.Company.ID, GuestName: "Past", GuestContact: "p@test.com",
			PartySize: 2, ScheduledAt: time.Now().Add(-48 * time.Hour), Status: models.ReservationStatusPending,
		}
		if err := app.DB.Create(&past).Error; err != nil {
			t.Fatal(err)
		}
		job := jobs.NewCleanupJob(app.DB, services.NewReservationService(app.DB, services.NewSystemEventService(app.DB)))
		job.Run()
		var count int64
		app.DB.Model(&models.Reservation{}).Where("id = ?", past.ID).Count(&count)
		if count != 0 {
			t.Fatal("expected past reservation deleted")
		}
	})
}
