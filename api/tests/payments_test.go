package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestPayments(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	t.Run("pay table with order", func(t *testing.T) {
		table, menu := setupTableForPayment(t, app, admin.Token, "Pay1", "Água", 500)
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)

		history := testutils.DecodeJSON[models.PaymentHistory](t, w)
		if history.ID == "" {
			t.Fatal("expected history id")
		}
		if history.CompanyID != admin.Company.ID {
			t.Fatalf("company %s, want %s", history.CompanyID, admin.Company.ID)
		}
		if history.TableID != table.ID || history.TableNumber != "Pay1" {
			t.Fatalf("unexpected table on history: %+v", history)
		}
		if history.TotalCents != 500 {
			t.Fatalf("expected total 500, got %d", history.TotalCents)
		}
		if history.PaidAt.IsZero() || history.TableSessionID == "" {
			t.Fatalf("missing paid_at or session: %+v", history)
		}

		var snap []struct {
			Name     string `json:"name"`
			Quantity int    `json:"quantity"`
			Price    int64  `json:"price_cents"`
		}
		if err := json.Unmarshal([]byte(history.ItemsSnapshot), &snap); err != nil {
			t.Fatalf("snapshot: %v", err)
		}
		if len(snap) != 1 || snap[0].Name != "Água" || snap[0].Quantity != 1 || snap[0].Price != 500 {
			t.Fatalf("unexpected snapshot: %+v", snap)
		}

		histories := testutils.Get[[]models.PaymentHistory](t, app, "/admin/payment-histories", admin.Token)
		var stored *models.PaymentHistory
		for i := range histories {
			if histories[i].ID == history.ID {
				stored = &histories[i]
				break
			}
		}
		if stored == nil {
			t.Fatal("pagamento não apareceu no GET de históricos")
		}
		if stored.TotalCents != 500 || stored.TableID != table.ID || stored.TableNumber != "Pay1" {
			t.Fatalf("GET do histórico não bate: %+v", stored)
		}

		tableAfter := testutils.Get[models.Table](t, app, "/admin/tables/"+table.ID, admin.Token)
		if tableAfter.Status != models.TableStatusAvailable {
			t.Fatalf("expected table available after pay, got %s", tableAfter.Status)
		}

		var session models.TableSession
		if err := app.DB.First(&session, "id = ?", history.TableSessionID).Error; err != nil {
			t.Fatal(err)
		}
		if session.ClosedAt == nil {
			t.Fatal("expected session closed after pay")
		}

		var orders []models.TableOrder
		if err := app.DB.Where("table_session_id = ?", history.TableSessionID).Find(&orders).Error; err != nil {
			t.Fatal(err)
		}
		if len(orders) == 0 {
			t.Fatal("expected orders on session")
		}
		for _, o := range orders {
			if o.Status != models.TableOrderStatusCompleted {
				t.Fatalf("expected completed order, got %s", o.Status)
			}
		}

		var ev models.SystemEvent
		if err := app.DB.Where("type = ? AND resource_id = ?", "payment_completed", history.ID).First(&ev).Error; err != nil {
			t.Fatalf("payment event not found: %v", err)
		}
	})

	t.Run("pay sums multiple orders", func(t *testing.T) {
		table, menu := setupTableForPayment(t, app, admin.Token, "Pay2", "Refri", 800)
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 2)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		history := testutils.DecodeJSON[models.PaymentHistory](t, w)
		if history.TotalCents != 2400 {
			t.Fatalf("expected total 2400, got %d", history.TotalCents)
		}

		var snap []struct {
			Quantity int   `json:"quantity"`
			Price    int64 `json:"price_cents"`
		}
		if err := json.Unmarshal([]byte(history.ItemsSnapshot), &snap); err != nil {
			t.Fatalf("snapshot: %v", err)
		}
		if len(snap) != 2 {
			t.Fatalf("expected 2 snapshot items, got %d", len(snap))
		}
	})

	t.Run("pay occupied table without orders", func(t *testing.T) {
		table, _ := setupTableForPayment(t, app, admin.Token, "PayEmpty", "Suco", 700)
		testutils.OccupyTable(t, app, admin.Token, table.ID)

		w := app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		history := testutils.DecodeJSON[models.PaymentHistory](t, w)
		if history.TotalCents != 0 {
			t.Fatalf("expected total 0, got %d", history.TotalCents)
		}
	})

	t.Run("list histories", func(t *testing.T) {
		w := app.Request("GET", "/admin/payment-histories", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		before := testutils.DecodeJSON[[]models.PaymentHistory](t, w)

		table, menu := setupTableForPayment(t, app, admin.Token, "PayList", "Café", 300)
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)
		w = app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		paid := testutils.DecodeJSON[models.PaymentHistory](t, w)

		w = app.Request("GET", "/admin/payment-histories", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		list := testutils.DecodeJSON[[]models.PaymentHistory](t, w)
		if len(list) != len(before)+1 {
			t.Fatalf("expected %d histories, got %d", len(before)+1, len(list))
		}
		found := false
		for _, h := range list {
			if h.ID == paid.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("paid history missing from list")
		}
	})

	t.Run("cashier can pay and list histories", func(t *testing.T) {
		mail := "cashier-pay@test.com"
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Caixa Pay", "email": mail, "password": "secret123", "role": models.RoleCashier,
		})
		token := testutils.Login(t, app, mail, "secret123")

		table, menu := setupTableForPayment(t, app, admin.Token, "PayCashier", "Chá", 400)
		testutils.OccupyTable(t, app, token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, token)
		testutils.AssertCode(t, w, http.StatusOK)

		w = app.Request("GET", "/admin/payment-histories", nil, token)
		testutils.AssertCode(t, w, http.StatusOK)
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		w := app.Request("POST", "/admin/tables/00000000-0000-0000-0000-000000000000/pay", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)

		w = app.Request("GET", "/admin/payment-histories", nil, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("pay table not found", func(t *testing.T) {
		w := app.Request("POST", "/admin/tables/00000000-0000-0000-0000-000000000000/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "recurso não encontrado" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("pay table without open session", func(t *testing.T) {
		table, _ := setupTableForPayment(t, app, admin.Token, "PayNoSession", "Bolo", 900)
		w := app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "mesa sem sessão aberta" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("pay twice fails", func(t *testing.T) {
		table, menu := setupTableForPayment(t, app, admin.Token, "PayTwice", "Vinho", 1500)
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)

		w = app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "mesa sem sessão aberta" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("isolated between companies", func(t *testing.T) {
		other := app.RegisterCompany(t)
		table, menu := setupTableForPayment(t, app, admin.Token, "PayIso", "Iso", 200)
		testutils.OccupyTable(t, app, admin.Token, table.ID)
		testutils.AddPublicOrder(t, app, table, menu.ID, 1)

		w := app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, other.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "recurso não encontrado" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}

		w = app.Request("POST", "/admin/tables/"+table.ID+"/pay", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		paid := testutils.DecodeJSON[models.PaymentHistory](t, w)

		w = app.Request("GET", "/admin/payment-histories", nil, other.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		list := testutils.DecodeJSON[[]models.PaymentHistory](t, w)
		for _, h := range list {
			if h.ID == paid.ID {
				t.Fatal("other company should not see this history")
			}
		}
	})
}

func setupTableForPayment(t *testing.T, app *testutils.TestApp, token, tableNumber, itemName string, price int64) (models.Table, models.MenuItem) {
	t.Helper()
	table := testutils.CreateTable(t, app, token, tableNumber, 2)
	menu := testutils.CreateMenuItem(t, app, token, map[string]interface{}{
		"name": itemName, "price_cents": price, "active": true,
	})
	return table, menu
}
