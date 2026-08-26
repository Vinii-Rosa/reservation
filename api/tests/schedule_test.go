package tests

import (
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestSchedule(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	t.Run("update range schedule", func(t *testing.T) {
		w := app.Request("PATCH", "/admin/company/schedule", map[string]interface{}{
			"reservation_mode":      "range",
			"opens_at":              "18:00",
			"closes_at":             "22:00",
			"slot_interval_minutes": 60,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		company := testutils.Get[models.Company](t, app, "/admin/company", admin.Token)
		if company.ReservationMode != models.ReservationModeRange || company.OpensAt != "18:00" || company.ClosesAt != "22:00" || company.SlotIntervalMinutes != 60 {
			t.Fatalf("GET após update da agenda não bate: %+v", company)
		}
	})

	t.Run("public get schedule", func(t *testing.T) {
		w := app.Request("GET", "/public/companies/"+admin.Company.ID+"/schedule", nil, "")
		testutils.AssertCode(t, w, http.StatusOK)
		var body struct {
			ReservationMode     string `json:"reservation_mode"`
			OpensAt             string `json:"opens_at"`
			ClosesAt            string `json:"closes_at"`
			SlotIntervalMinutes int    `json:"slot_interval_minutes"`
		}
		body = testutils.DecodeJSON[struct {
			ReservationMode     string `json:"reservation_mode"`
			OpensAt             string `json:"opens_at"`
			ClosesAt            string `json:"closes_at"`
			SlotIntervalMinutes int    `json:"slot_interval_minutes"`
		}](t, w)
		if body.ReservationMode != "range" || body.OpensAt != "18:00" || body.ClosesAt != "22:00" || body.SlotIntervalMinutes != 60 {
			t.Fatalf("unexpected public schedule: %+v", body)
		}
	})

	t.Run("update fixed schedule", func(t *testing.T) {
		w := app.Request("PATCH", "/admin/company/schedule", map[string]interface{}{
			"reservation_mode": "fixed",
			"fixed_time":       "19:00",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		company := testutils.DecodeJSON[models.Company](t, w)
		if company.ReservationMode != models.ReservationModeFixed || company.FixedTime != "19:00" {
			t.Fatalf("unexpected fixed schedule: %+v", company)
		}

		w = app.Request("GET", "/public/companies/"+admin.Company.ID+"/schedule", nil, "")
		testutils.AssertCode(t, w, http.StatusOK)
		var body struct {
			ReservationMode string `json:"reservation_mode"`
			FixedTime       string `json:"fixed_time"`
		}
		body = testutils.DecodeJSON[struct {
			ReservationMode string `json:"reservation_mode"`
			FixedTime       string `json:"fixed_time"`
		}](t, w)
		if body.ReservationMode != "fixed" || body.FixedTime != "19:00" {
			t.Fatalf("unexpected public fixed: %+v", body)
		}
	})

	t.Run("invalid reservation mode", func(t *testing.T) {
		w := app.Request("PATCH", "/admin/company/schedule", map[string]interface{}{
			"reservation_mode": "whatever",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "reservation_mode inválido" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("public schedule not found", func(t *testing.T) {
		w := app.Request("GET", "/public/companies/00000000-0000-0000-0000-000000000000/schedule", nil, "")
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("cashier cannot update schedule", func(t *testing.T) {
		token := testutils.CashierToken(t, app, admin.Token, testutils.UniqueEmail("cashier-sched"))
		w := app.Request("PATCH", "/admin/company/schedule", map[string]interface{}{
			"opens_at": "10:00",
		}, token)
		testutils.AssertCode(t, w, http.StatusForbidden)
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := app.Request("PATCH", "/admin/company/schedule", map[string]interface{}{
			"opens_at": "10:00",
		}, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})
}
