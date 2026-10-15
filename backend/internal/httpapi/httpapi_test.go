package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/httpapi"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

const (
	alisher int64 = 100000001
	madina  int64 = 100000002
)

// 2026-10-16 10:00 in Moscow.
var now = clock.Fixed(time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC))

var contract validator.Validator

func TestMain(m *testing.M) {
	spec, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		panic(err)
	}
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		panic(err)
	}
	v, errs := validator.NewValidator(doc)
	if len(errs) > 0 {
		panic(errors.Join(errs...))
	}
	contract = v
	os.Exit(m.Run())
}

type pinger struct{ err error }

func (p pinger) Ping(context.Context) error { return p.err }

type env struct {
	handler http.Handler
	ids     map[string]int64 // seeded expenses by comment
}

func newEnv(t *testing.T) env {
	t.Helper()
	ctx := context.Background()
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	store := memstore.New(now)
	users := user.NewService(store.Users())
	for _, u := range []user.User{{TelegramID: alisher, Name: "Алишер"}, {TelegramID: madina, Name: "Мадина"}} {
		if _, err := users.Upsert(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	svc := expense.NewService(store.Expenses(), now, moscow)
	ids := map[string]int64{}
	add := func(owner int64, amount expense.Money, cat expense.Category, d int, comment string) {
		e, err := svc.ForUser(owner).Create(ctx, expense.Draft{
			Amount: amount, Category: cat, SpentAt: time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC), Comment: &comment,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids[comment] = e.ID
	}
	add(alisher, 35000, expense.CategoryFood, 15, "кофе")
	add(madina, 120000, expense.CategoryTransport, 14, "такси")
	add(alisher, 348050, expense.CategoryFood, 14, "продукты")

	return env{
		handler: httpapi.NewHandler(httpapi.Deps{
			Expenses: svc,
			Users:    users,
			DB:       pinger{},
			Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		}),
		ids: ids,
	}
}

type response struct {
	status int
	header http.Header
	body   map[string]any
	list   []any
}

// call performs a request and checks the response against the OpenAPI
// contract. Well-formed requests are checked against the contract too.
func call(t *testing.T, h http.Handler, method, path, body string, requestIsValid bool) response {
	t.Helper()
	newReq := func() *http.Request {
		r := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		return r
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newReq())
	raw := rec.Body.Bytes()

	res := rec.Result()
	res.Body = io.NopCloser(bytes.NewReader(raw))
	var ok bool
	var verrs []string
	if requestIsValid {
		ok, verrs = wrap(contract.ValidateHttpRequestResponse(newReq(), res))
	} else {
		ok, verrs = wrap(contract.ValidateHttpResponse(newReq(), res))
	}
	if !ok {
		for _, e := range verrs {
			t.Errorf("%s %s: contract violation: %s", method, path, e)
		}
	}

	out := response{status: rec.Code, header: rec.Header()}
	if len(raw) > 0 {
		if raw[0] == '[' {
			if err := json.Unmarshal(raw, &out.list); err != nil {
				t.Fatalf("decode %s: %v", raw, err)
			}
		} else if err := json.Unmarshal(raw, &out.body); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
	}
	return out
}

func wrap[E interface{ Error() string }](ok bool, errs []E) (bool, []string) {
	out := make([]string, len(errs))
	for i, e := range errs {
		out[i] = e.Error()
	}
	return ok, out
}

func fields(t *testing.T, r response) map[string]any {
	t.Helper()
	errObj, _ := r.body["error"].(map[string]any)
	f, _ := errObj["fields"].(map[string]any)
	return f
}

func wantError(t *testing.T, r response, status int, code string, field string) {
	t.Helper()
	if r.status != status {
		t.Fatalf("status = %d, want %d; body %v", r.status, status, r.body)
	}
	errObj, _ := r.body["error"].(map[string]any)
	if errObj["code"] != code {
		t.Errorf("code = %v, want %s", errObj["code"], code)
	}
	if msg, _ := errObj["message"].(string); msg == "" {
		t.Error("error message is empty")
	}
	if field != "" {
		if _, ok := fields(t, r)[field]; !ok {
			t.Errorf("fields = %v, want %q", fields(t, r), field)
		}
	}
}

func TestUsersAndCategories(t *testing.T) {
	e := newEnv(t)
	r := call(t, e.handler, "GET", "/api/users", "", true)
	if r.status != 200 || len(r.list) != 2 || r.list[0].(map[string]any)["name"] != "Алишер" {
		t.Errorf("users = %d %v", r.status, r.list)
	}
	r = call(t, e.handler, "GET", "/api/categories", "", true)
	if r.status != 200 || len(r.list) != 7 || r.list[0].(map[string]any)["code"] != "food" {
		t.Errorf("categories = %d %v", r.status, r.list)
	}
}

func TestListExpenses(t *testing.T) {
	e := newEnv(t)

	r := call(t, e.handler, "GET", "/api/expenses", "", true)
	if r.status != 200 || r.body["total"] != float64(3) {
		t.Fatalf("list = %d %v", r.status, r.body)
	}
	first := r.body["items"].([]any)[0].(map[string]any)
	if first["comment"] != "кофе" || first["amount"] != float64(350) || first["user_name"] != "Алишер" ||
		first["spent_at"] != "2026-10-15" || first["source"] != "bot" {
		t.Errorf("first item = %v", first)
	}

	r = call(t, e.handler, "GET", "/api/expenses?user_id=100000001&category=food&from=2026-10-14&to=2026-10-14&sort=amount&order=asc&limit=1&offset=0", "", true)
	items := r.body["items"].([]any)
	if r.status != 200 || r.body["total"] != float64(1) || len(items) != 1 || items[0].(map[string]any)["amount"] != 3480.5 {
		t.Errorf("filtered list = %d %v", r.status, r.body)
	}

	bad := []struct{ query, field string }{
		{"from=15.10.2026", "from"},
		{"to=yesterday", "to"},
		{"category=fun", "category"},
		{"user_id=abc", "user_id"},
		{"limit=abc", "limit"},
		{"limit=0", "limit"},
		{"limit=101", "limit"},
		{"offset=-1", "offset"},
		{"sort=id", "sort"},
		{"order=up", "order"},
		{"from=2026-10-16&to=2026-10-01", "from"},
	}
	for _, b := range bad {
		r := call(t, e.handler, "GET", "/api/expenses?"+b.query, "", false)
		wantError(t, r, 400, "bad_request", b.field)
	}
}

func TestSummary(t *testing.T) {
	e := newEnv(t)
	r := call(t, e.handler, "GET", "/api/expenses/summary?from=2026-10-01&to=2026-10-31", "", true)
	if r.status != 200 || r.body["total"] != 5030.5 || r.body["count"] != float64(3) {
		t.Fatalf("summary = %d %v", r.status, r.body)
	}
	byCat := r.body["by_category"].([]any)
	if len(byCat) != 2 || byCat[0].(map[string]any)["category"] != "food" || byCat[0].(map[string]any)["total"] != 3830.5 {
		t.Errorf("by_category = %v", byCat)
	}

	r = call(t, e.handler, "GET", "/api/expenses/summary?user_id=999", "", true)
	if r.status != 200 || r.body["total"] != float64(0) || len(r.body["by_category"].([]any)) != 0 {
		t.Errorf("empty summary = %v", r.body)
	}

	wantError(t, call(t, e.handler, "GET", "/api/expenses/summary?category=fun", "", false), 400, "bad_request", "category")
}

func TestCreateExpense(t *testing.T) {
	e := newEnv(t)
	body := `{"user_id": 100000002, "amount": 350.5, "category": "health", "spent_at": "2026-10-16", "comment": "  аптека "}`
	r := call(t, e.handler, "POST", "/api/expenses", body, true)
	if r.status != 201 {
		t.Fatalf("create = %d %v", r.status, r.body)
	}
	if r.body["amount"] != 350.5 || r.body["source"] != "web" || r.body["comment"] != "аптека" || r.body["user_name"] != "Мадина" {
		t.Errorf("created = %v", r.body)
	}
	id := int64(r.body["id"].(float64))
	if loc := r.header.Get("Location"); loc != "/api/expenses/"+jsonInt(id) {
		t.Errorf("Location = %q", loc)
	}
	got := call(t, e.handler, "GET", "/api/expenses/"+jsonInt(id), "", true)
	if got.status != 200 || got.body["amount"] != 350.5 {
		t.Errorf("get created = %d %v", got.status, got.body)
	}

	// Without comment and with null comment.
	for _, b := range []string{
		`{"user_id": 100000001, "amount": 100, "category": "other", "spent_at": "2026-10-16"}`,
		`{"user_id": 100000001, "amount": 100, "category": "other", "spent_at": "2026-10-16", "comment": null}`,
	} {
		r := call(t, e.handler, "POST", "/api/expenses", b, true)
		if r.status != 201 || r.body["comment"] != nil {
			t.Errorf("create %s = %d %v", b, r.status, r.body)
		}
	}
}

func TestCreateExpenseErrors(t *testing.T) {
	e := newEnv(t)
	valid := map[string]any{"user_id": 100000002, "amount": 350, "category": "food", "spent_at": "2026-10-16"}
	with := func(k string, v any) string {
		m := map[string]any{}
		for key, val := range valid {
			m[key] = val
		}
		m[k] = v
		b, _ := json.Marshal(m)
		return string(b)
	}

	unprocessable := []struct {
		name, body, field string
	}{
		{"amount as string", with("amount", "350"), "amount"},
		{"three decimals", strings.Replace(with("amount", 1), `"amount":1`, `"amount":350.555`, 1), "amount"},
		{"zero amount", with("amount", 0), "amount"},
		{"negative amount", with("amount", -5), "amount"},
		{"too large amount", with("amount", 100000001), "amount"},
		{"unknown category", with("category", "fun"), "category"},
		{"bad date", with("spent_at", "16.10.2026"), "spent_at"},
		{"future date", with("spent_at", "2026-12-31"), "spent_at"},
		{"unknown family member", with("user_id", 42), "user_id"},
		{"long comment", with("comment", strings.Repeat("ж", 501)), "comment"},
	}
	for _, tt := range unprocessable {
		t.Run(tt.name, func(t *testing.T) {
			wantError(t, call(t, e.handler, "POST", "/api/expenses", tt.body, false), 422, "validation_error", tt.field)
		})
	}

	t.Run("missing required fields", func(t *testing.T) {
		r := call(t, e.handler, "POST", "/api/expenses", `{}`, false)
		wantError(t, r, 422, "validation_error", "")
		for _, f := range []string{"user_id", "amount", "category", "spent_at"} {
			if _, ok := fields(t, r)[f]; !ok {
				t.Errorf("missing %q in %v", f, fields(t, r))
			}
		}
	})

	malformed := []struct {
		name, body, field string
	}{
		{"invalid JSON", `{"amount": `, "body"},
		{"empty body", ``, "body"},
		{"unknown field", strings.Replace(with("amount", 1), `{`, `{"source":"bot",`, 1), "body"},
		{"user_id as string", with("user_id", "alisher"), "user_id"},
		{"two objects", with("amount", 1) + `{}`, "body"},
	}
	for _, tt := range malformed {
		t.Run(tt.name, func(t *testing.T) {
			wantError(t, call(t, e.handler, "POST", "/api/expenses", tt.body, false), 400, "bad_request", tt.field)
		})
	}
}

func TestGetUpdateDelete(t *testing.T) {
	e := newEnv(t)
	coffee := "/api/expenses/" + jsonInt(e.ids["кофе"])

	if r := call(t, e.handler, "GET", coffee, "", true); r.status != 200 || r.body["comment"] != "кофе" {
		t.Errorf("get = %d %v", r.status, r.body)
	}
	wantError(t, call(t, e.handler, "GET", "/api/expenses/999999", "", true), 404, "not_found", "")
	wantError(t, call(t, e.handler, "GET", "/api/expenses/abc", "", false), 400, "bad_request", "id")
	wantError(t, call(t, e.handler, "GET", "/api/expenses/0", "", false), 400, "bad_request", "id")

	// Full replacement: moves the expense to Madina and clears the comment.
	body := `{"user_id": 100000002, "amount": 450, "category": "entertainment", "spent_at": "2026-10-10", "comment": null}`
	r := call(t, e.handler, "PUT", coffee, body, true)
	if r.status != 200 || r.body["amount"] != float64(450) || r.body["user_name"] != "Мадина" ||
		r.body["category"] != "entertainment" || r.body["spent_at"] != "2026-10-10" || r.body["comment"] != nil {
		t.Errorf("put = %d %v", r.status, r.body)
	}
	if r.body["source"] != "bot" {
		t.Errorf("source changed by the web panel: %v", r.body["source"])
	}

	wantError(t, call(t, e.handler, "PUT", "/api/expenses/999999", body, true), 404, "not_found", "")
	wantError(t, call(t, e.handler, "PUT", coffee, `{"amount": 1}`, false), 422, "validation_error", "user_id")
	wantError(t, call(t, e.handler, "PUT", coffee, strings.Replace(body, "100000002", "42", 1), false), 422, "validation_error", "user_id")

	if r := call(t, e.handler, "DELETE", coffee, "", true); r.status != 204 {
		t.Errorf("delete = %d %v", r.status, r.body)
	}
	wantError(t, call(t, e.handler, "DELETE", coffee, "", true), 404, "not_found", "")
	wantError(t, call(t, e.handler, "GET", coffee, "", true), 404, "not_found", "")
}

func TestHealthz(t *testing.T) {
	e := newEnv(t)
	if r := call(t, e.handler, "GET", "/healthz", "", true); r.status != 200 || r.body["status"] != "ok" {
		t.Errorf("healthz = %d %v", r.status, r.body)
	}

	down := httpapi.NewHandler(httpapi.Deps{
		DB: pinger{err: errors.New("connection refused")}, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if r := call(t, down, "GET", "/healthz", "", true); r.status != 503 || r.body["status"] != "unavailable" {
		t.Errorf("healthz down = %d %v", r.status, r.body)
	}
}

// brokenService fails or panics on every call.
type brokenService struct {
	httpapi.ExpenseService
	panics bool
}

func (b brokenService) List(context.Context, expense.Filter) (expense.Page, error) {
	if b.panics {
		panic("boom")
	}
	return expense.Page{}, errors.New("database is on fire: secret details")
}

func TestUnexpectedErrorsAreHidden(t *testing.T) {
	var logs bytes.Buffer
	for _, panics := range []bool{false, true} {
		h := httpapi.NewHandler(httpapi.Deps{
			Expenses: brokenService{panics: panics},
			DB:       pinger{},
			Log:      slog.New(slog.NewJSONHandler(&logs, nil)),
		})
		r := call(t, h, "GET", "/api/expenses", "", true)
		wantError(t, r, 500, "internal", "")
		if msg := r.body["error"].(map[string]any)["message"].(string); strings.Contains(msg, "secret") {
			t.Errorf("internal details leaked: %q", msg)
		}
		if r.header.Get("X-Request-ID") == "" {
			t.Error("no X-Request-ID")
		}
	}
	if !strings.Contains(logs.String(), "secret details") || !strings.Contains(logs.String(), "boom") {
		t.Errorf("errors were not logged: %s", logs.String())
	}
}

func TestRequestIDAndUnknownRoute(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/api/categories", nil)
	req.Header.Set("X-Request-ID", "from-nginx-42")
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "from-nginx-42" {
		t.Errorf("X-Request-ID = %q", got)
	}

	rec = httptest.NewRecorder()
	e.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), "GET", "/api/nope", nil))
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), `"not_found"`) {
		t.Errorf("unknown route = %d %s", rec.Code, rec.Body.String())
	}
}

func jsonInt(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}
