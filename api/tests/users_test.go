package tests

import (
	"fmt"
	"net/http"
	"testing"

	"reservation/api/internal/models"
	"reservation/api/internal/testutils"
)

func TestUsers(t *testing.T) {
	app := testutils.SetupTestApp(t)
	admin := app.RegisterCompany(t)

	seq := 0
	email := func(prefix string) string {
		seq++
		return fmt.Sprintf("%s-%d@test.com", prefix, seq)
	}

	t.Run("create cashier success", func(t *testing.T) {
		mail := email("cashier")
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Caixa", "email": mail, "password": "secret123", "role": models.RoleCashier,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.User](t, w)
		if created.ID == "" {
			t.Fatal("create não retornou id")
		}

		user := testutils.Get[models.User](t, app, "/admin/users/"+created.ID, admin.Token)
		if user.Name != "Caixa" || user.Email != mail || user.Role != models.RoleCashier {
			t.Fatalf("GET não bate com o que foi enviado: %+v", user)
		}
		if user.PasswordHash != "" {
			t.Fatal("password hash não deve ir na resposta")
		}
		if user.CompanyID == nil || *user.CompanyID != admin.Company.ID {
			t.Fatalf("expected company %s, got %v", admin.Company.ID, user.CompanyID)
		}
	})

	t.Run("create without role defaults to cashier", func(t *testing.T) {
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Sem Role", "email": email("norole"), "password": "secret123",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.User](t, w)
		user := testutils.Get[models.User](t, app, "/admin/users/"+created.ID, admin.Token)
		if user.Name != "Sem Role" || user.Role != models.RoleCashier {
			t.Fatalf("GET não bate com o enviado: %+v", user)
		}
	})

	t.Run("create admin success", func(t *testing.T) {
		mail := email("admin")
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Outro Admin", "email": mail, "password": "secret123", "role": models.RoleAdmin,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.User](t, w)
		user := testutils.Get[models.User](t, app, "/admin/users/"+created.ID, admin.Token)
		if user.Name != "Outro Admin" || user.Email != mail || user.Role != models.RoleAdmin {
			t.Fatalf("GET não bate com o enviado: %+v", user)
		}
	})

	t.Run("create normalizes email", func(t *testing.T) {
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Norm", "email": "  Foo.Bar@Test.COM  ", "password": "secret123", "role": models.RoleCashier,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		created := testutils.DecodeJSON[models.User](t, w)
		user := testutils.Get[models.User](t, app, "/admin/users/"+created.ID, admin.Token)
		if user.Email != "foo.bar@test.com" {
			t.Fatalf("expected normalized email, got %s", user.Email)
		}
	})

	t.Run("created user can login", func(t *testing.T) {
		mail := email("login")
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Login", "email": mail, "password": "secret123", "role": models.RoleCashier,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
		_ = testutils.Login(t, app, mail, "secret123")
	})

	t.Run("get by id", func(t *testing.T) {
		created := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Get Me", "email": email("get"), "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("GET", "/admin/users/"+created.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.DecodeJSON[models.User](t, w)
		if got.ID != created.ID || got.Name != "Get Me" {
			t.Fatalf("unexpected get: %+v", got)
		}
	})

	t.Run("list includes created users", func(t *testing.T) {
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Listed", "email": email("list"), "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("GET", "/admin/users", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		users := testutils.DecodeJSON[[]models.User](t, w)
		if len(users) < 2 {
			t.Fatalf("expected at least 2 users, got %d", len(users))
		}
	})

	t.Run("update name and email", func(t *testing.T) {
		created := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Old", "email": email("upd"), "password": "secret123", "role": models.RoleCashier,
		})
		newMail := email("upd-new")
		w := app.Request("PUT", "/admin/users/"+created.ID, map[string]interface{}{
			"name": "New Name", "email": newMail,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.Get[models.User](t, app, "/admin/users/"+created.ID, admin.Token)
		if got.Name != "New Name" || got.Email != newMail {
			t.Fatalf("GET após update não bate: %+v", got)
		}
	})

	t.Run("update empty body keeps fields", func(t *testing.T) {
		mail := email("keep")
		created := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Keep", "email": mail, "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("PUT", "/admin/users/"+created.ID, map[string]interface{}{}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
		got := testutils.Get[models.User](t, app, "/admin/users/"+created.ID, admin.Token)
		if got.Name != "Keep" || got.Email != mail || got.Role != models.RoleCashier {
			t.Fatalf("fields changed on empty update: %+v", got)
		}
		_ = testutils.Login(t, app, mail, "secret123")
	})

	t.Run("update password only", func(t *testing.T) {
		mail := email("pwd")
		created := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Pwd", "email": mail, "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("PUT", "/admin/users/"+created.ID, map[string]interface{}{
			"password": "novaSenha123",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)

		w = app.Request("POST", "/auth/login", map[string]string{
			"email": mail, "password": "secret123",
		}, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)

		_ = testutils.Login(t, app, mail, "novaSenha123")
	})

	t.Run("update email to own email succeeds", func(t *testing.T) {
		mail := email("own")
		created := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Own", "email": mail, "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("PUT", "/admin/users/"+created.ID, map[string]interface{}{
			"email": mail,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusOK)
	})

	t.Run("delete user", func(t *testing.T) {
		created := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Gone", "email": email("del"), "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("DELETE", "/admin/users/"+created.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNoContent)

		w = app.Request("GET", "/admin/users/"+created.ID, nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "X", "email": email("noauth"), "password": "secret123", "role": models.RoleCashier,
		}, "")
		testutils.AssertCode(t, w, http.StatusUnauthorized)
	})

	t.Run("invalid payload", func(t *testing.T) {
		w := app.Request("POST", "/admin/users", "não é um objeto", admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "payload inválido" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		mail := email("dup")
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "A", "email": mail, "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "B", "email": mail, "password": "secret123", "role": models.RoleCashier,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "já existe um usuário com este e-mail nesta companhia" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("duplicate email case insensitive", func(t *testing.T) {
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "A", "email": "casedup@test.com", "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "B", "email": "CaseDup@TEST.com", "password": "secret123", "role": models.RoleCashier,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("get not found", func(t *testing.T) {
		w := app.Request("GET", "/admin/users/00000000-0000-0000-0000-000000000000", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("update not found", func(t *testing.T) {
		w := app.Request("PUT", "/admin/users/00000000-0000-0000-0000-000000000000", map[string]interface{}{
			"name": "Nope",
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
	})

	t.Run("delete not found", func(t *testing.T) {
		w := app.Request("DELETE", "/admin/users/00000000-0000-0000-0000-000000000000", nil, admin.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)
	})

	t.Run("update email taken", func(t *testing.T) {
		a := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "A", "email": email("taken-a"), "password": "secret123", "role": models.RoleCashier,
		})
		b := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "B", "email": email("taken-b"), "password": "secret123", "role": models.RoleCashier,
		})
		w := app.Request("PUT", "/admin/users/"+b.ID, map[string]interface{}{
			"email": a.Email,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "já existe um usuário com este e-mail nesta companhia" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("cashier cannot manage users", func(t *testing.T) {
		mail := email("cashier-role")
		testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Cashier", "email": mail, "password": "secret123", "role": models.RoleCashier,
		})
		token := testutils.Login(t, app, mail, "secret123")

		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Blocked", "email": email("blocked"), "password": "secret123", "role": models.RoleCashier,
		}, token)
		testutils.AssertCode(t, w, http.StatusForbidden)

		w = app.Request("GET", "/admin/users", nil, token)
		testutils.AssertCode(t, w, http.StatusForbidden)
	})

	t.Run("admin cannot create or promote to dev", func(t *testing.T) {
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Dev", "email": email("dev-create"), "password": "secret123", "role": models.RoleDev,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "apenas dev pode criar usuário dev" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}

		target := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Promote", "email": email("promote"), "password": "secret123", "role": models.RoleCashier,
		})
		w = app.Request("PUT", "/admin/users/"+target.ID, map[string]interface{}{
			"role": models.RoleDev,
		}, admin.Token)
		testutils.AssertCode(t, w, http.StatusBadRequest)
		if testutils.ResponseError(t, w) != "apenas dev pode atribuir role dev" {
			t.Fatalf("unexpected error: %s", w.Body.String())
		}
	})

	t.Run("dev can create another dev", func(t *testing.T) {
		devToken := testutils.SeedDevToken(t, app, admin.Company.ID)
		w := app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Dev 2", "email": email("dev2"), "password": "secret123", "role": models.RoleDev,
		}, devToken)
		testutils.AssertCode(t, w, http.StatusCreated)
		user := testutils.DecodeJSON[models.User](t, w)
		if user.Role != models.RoleDev {
			t.Fatalf("expected dev, got %s", user.Role)
		}
	})

	t.Run("isolated between companies", func(t *testing.T) {
		other := app.RegisterCompany(t)
		mail := email("iso")
		created := testutils.CreateUser(t, app, admin.Token, map[string]interface{}{
			"name": "Iso", "email": mail, "password": "secret123", "role": models.RoleCashier,
		})

		w := app.Request("GET", "/admin/users/"+created.ID, nil, other.Token)
		testutils.AssertCode(t, w, http.StatusNotFound)

		w = app.Request("POST", "/admin/users", map[string]interface{}{
			"name": "Same Email Other Co", "email": mail, "password": "secret123", "role": models.RoleCashier,
		}, other.Token)
		testutils.AssertCode(t, w, http.StatusCreated)
	})
}
