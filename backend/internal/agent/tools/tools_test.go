package tools_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/jsonschema/v6"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

const (
	alisher int64 = 100000001
	madina  int64 = 100000002
)

// 2026-10-16 10:00 in Moscow.
var now = clock.Fixed(time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC))

type env struct {
	reg     *tools.Registry
	svc     *expense.Service
	alisher expense.UserScope
	madina  expense.UserScope
	schemas map[string]*jsonschema.Schema
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	store := memstore.New(now)
	for _, u := range []user.User{{TelegramID: alisher, Name: "Алишер"}, {TelegramID: madina, Name: "Мадина"}} {
		if _, err := store.Users().Upsert(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	svc := expense.NewService(store.Expenses(), now, moscow)
	e := &env{reg: tools.Default(), svc: svc, alisher: svc.ForUser(alisher), madina: svc.ForUser(madina)}
	e.schemas = compileSchemas(t, e.reg)
	return e
}

func compileSchemas(t *testing.T, reg *tools.Registry) map[string]*jsonschema.Schema {
	t.Helper()
	out := map[string]*jsonschema.Schema{}
	for _, tool := range reg.Tools() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(tool.Schema))
		if err != nil {
			t.Fatalf("%s: schema is not JSON: %v", tool.Name, err)
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource(tool.Name+".json", doc); err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		sch, err := c.Compile(tool.Name + ".json")
		if err != nil {
			t.Fatalf("%s: invalid JSON Schema: %v", tool.Name, err)
		}
		out[tool.Name] = sch
	}
	return out
}

// call runs a tool as the given user. Arguments of calls expected to
// succeed must also satisfy the published schema.
func (e *env) call(t *testing.T, scope expense.UserScope, name, args string) (map[string]any, tools.Result) {
	t.Helper()
	return e.run(t, scope, name, args, true)
}

// callLenient runs a call whose arguments deliberately break the schema
// (a number instead of a string, a limit above the maximum): the schema
// guides the model, the handler still tolerates such minor slips.
func (e *env) callLenient(t *testing.T, scope expense.UserScope, name, args string) (map[string]any, tools.Result) {
	t.Helper()
	return e.run(t, scope, name, args, false)
}

func (e *env) run(t *testing.T, scope expense.UserScope, name, args string, checkSchema bool) (map[string]any, tools.Result) {
	t.Helper()
	res := e.reg.Call(context.Background(), scope, name, args)
	if checkSchema && res.Err == nil && args != "" {
		inst, err := jsonschema.UnmarshalJSON(strings.NewReader(args))
		if err != nil {
			t.Fatalf("args are not JSON: %v", err)
		}
		if err := e.schemas[name].Validate(inst); err != nil {
			t.Errorf("%s accepted %s, which violates its own schema: %v", name, args, err)
		}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(res.Content), &out); err != nil {
		t.Fatalf("%s returned non-JSON %q: %v", name, res.Content, err)
	}
	return out, res
}

func wantToolError(t *testing.T, out map[string]any, res tools.Result, field string) {
	t.Helper()
	if res.Err == nil || res.Internal {
		t.Fatalf("want a model-facing error, got %+v", res)
	}
	if msg, _ := out["error"].(string); msg == "" {
		t.Errorf("no error message in %v", out)
	}
	if field != "" {
		f, _ := out["fields"].(map[string]any)
		if _, ok := f[field]; !ok {
			t.Errorf("fields = %v, want %q", f, field)
		}
	}
}

func TestSchemas(t *testing.T) {
	reg := tools.Default()
	compileSchemas(t, reg) // every schema is a valid JSON Schema

	names := map[string]bool{}
	for _, tool := range reg.Tools() {
		if names[tool.Name] {
			t.Errorf("duplicate tool %s", tool.Name)
		}
		names[tool.Name] = true
		if len(tool.Description) < 40 {
			t.Errorf("%s: description is too short to guide the model", tool.Name)
		}
		var s struct {
			Type       string                     `json:"type"`
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(tool.Schema, &s); err != nil || s.Type != "object" {
			t.Errorf("%s: schema must be an object: %v", tool.Name, err)
		}
		// Security: the model must have no way to address another user.
		for prop := range s.Properties {
			if strings.Contains(strings.ToLower(prop), "user") || strings.Contains(strings.ToLower(prop), "owner") {
				t.Errorf("%s: property %q lets the model choose a user", tool.Name, prop)
			}
		}
	}
	for _, want := range []string{"add_expense", "list_expenses", "summarize_expenses", "update_expense", "delete_expense"} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
	if defs := reg.Defs(); len(defs) != len(names) || defs[0].Name != "add_expense" || len(defs[0].Parameters) == 0 {
		t.Errorf("Defs = %+v", defs)
	}
}

func TestAddExpense(t *testing.T) {
	e := newEnv(t)

	out, res := e.call(t, e.alisher, "add_expense", `{"amount": "350", "category": "food", "comment": "кофе"}`)
	if res.Err != nil {
		t.Fatalf("add_expense: %v (%s)", res.Err, res.Content)
	}
	created := out["created"].(map[string]any)
	if created["amount"] != "350 ₽" || created["category"] != "food" || created["category_name"] != "Еда" ||
		created["spent_at"] != "2026-10-16" || created["comment"] != "кофе" || created["id"] == nil {
		t.Errorf("created = %v", created)
	}
	stored, err := e.alisher.Get(context.Background(), int64(created["id"].(float64)))
	if err != nil || stored.Source != expense.SourceBot || stored.Amount != 35000 {
		t.Errorf("stored = %+v, %v", stored, err)
	}

	// A number instead of a string and an explicit date.
	out, res = e.callLenient(t, e.alisher, "add_expense", `{"amount": 1200.5, "category": "transport", "spent_at": "2026-10-15"}`)
	if res.Err != nil || out["created"].(map[string]any)["amount"] != "1 200,50 ₽" {
		t.Errorf("numeric amount: %v %v", out, res.Err)
	}

	cases := []struct{ name, args, field string }{
		{"no amount", `{"category": "food"}`, ""},
		{"amount is text", `{"amount": "триста", "category": "food"}`, "amount"},
		{"three decimals", `{"amount": "1.005", "category": "food"}`, "amount"},
		{"zero", `{"amount": "0", "category": "food"}`, "amount"},
		{"unknown category", `{"amount": "100", "category": "coffee"}`, "category"},
		{"bad date", `{"amount": "100", "category": "food", "spent_at": "вчера"}`, "spent_at"},
		{"future date", `{"amount": "100", "category": "food", "spent_at": "2027-01-01"}`, "spent_at"},
		{"not an object", `["350"]`, ""},
		{"broken JSON", `{"amount": "350"`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, res := e.call(t, e.alisher, "add_expense", tc.args)
			wantToolError(t, out, res, tc.field)
		})
	}
}

func TestToolsCannotReachAnotherUser(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	comment := "такси"
	taxi, err := e.madina.Create(ctx, expense.Draft{Amount: 120000, Category: expense.CategoryTransport, Comment: &comment})
	if err != nil {
		t.Fatal(err)
	}
	id := jsonID(taxi.ID)

	// The model tries to write as Madina: the extra field is ignored.
	out, res := e.call(t, e.alisher, "add_expense", `{"amount": "100", "category": "food", "user_id": 100000002}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	created, _ := e.svc.Get(ctx, int64(out["created"].(map[string]any)["id"].(float64)))
	if created.UserID != alisher {
		t.Fatalf("expense was created for user %d", created.UserID)
	}

	// Lists and totals never include Madina's taxi.
	out, _ = e.call(t, e.alisher, "list_expenses", `{"query": "такси", "user_id": 100000002}`)
	if out["total"] != float64(0) {
		t.Errorf("Alisher sees Madina's expenses: %v", out)
	}
	out, _ = e.call(t, e.alisher, "summarize_expenses", `{}`)
	if out["total"] != "100 ₽" {
		t.Errorf("Alisher's total includes Madina: %v", out)
	}

	// Guessing the id of Madina's expense does not help.
	out, res = e.call(t, e.alisher, "update_expense", `{"id": `+id+`, "amount": "1"}`)
	wantToolError(t, out, res, "")
	out, res = e.call(t, e.alisher, "delete_expense", `{"id": `+id+`}`)
	wantToolError(t, out, res, "")
	if !strings.Contains(res.Content, "не найдена") {
		t.Errorf("foreign expense must look missing: %s", res.Content)
	}

	still, err := e.madina.Get(ctx, taxi.ID)
	if err != nil || still.Amount != 120000 {
		t.Errorf("Madina's taxi changed: %+v, %v", still, err)
	}
}

func seedAlisher(t *testing.T, e *env) map[string]int64 {
	t.Helper()
	ids := map[string]int64{}
	for _, s := range []struct {
		amount  expense.Money
		cat     expense.Category
		day     int
		comment string
	}{
		{35000, expense.CategoryFood, 15, "кофе"},
		{120000, expense.CategoryTransport, 15, "такси до работы"},
		{348050, expense.CategoryFood, 14, "продукты"},
		{50000, expense.CategoryTransport, 9, "Такси домой"},
	} {
		c := s.comment
		ex, err := e.alisher.Create(context.Background(), expense.Draft{
			Amount: s.amount, Category: s.cat, SpentAt: time.Date(2026, 10, s.day, 0, 0, 0, 0, time.UTC), Comment: &c,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids[s.comment] = ex.ID
	}
	return ids
}

func itemComments(out map[string]any) []string {
	var got []string
	for _, it := range out["items"].([]any) {
		got = append(got, it.(map[string]any)["comment"].(string))
	}
	return got
}

func TestListExpenses(t *testing.T) {
	e := newEnv(t)
	seedAlisher(t, e)

	out, res := e.call(t, e.alisher, "list_expenses", `{"query": "такси"}`)
	if res.Err != nil || out["total"] != float64(2) {
		t.Fatalf("query = %v %v", out, res.Err)
	}
	if got := itemComments(out); strings.Join(got, "|") != "такси до работы|Такси домой" {
		t.Errorf("order by date desc: %v", got)
	}

	// "Последнее" is the last recorded expense, not the latest date.
	out, _ = e.call(t, e.alisher, "list_expenses", `{"sort": "created_at", "limit": 1}`)
	if got := itemComments(out); len(got) != 1 || got[0] != "Такси домой" || out["total"] != float64(4) {
		t.Errorf("last recorded = %v", out)
	}

	out, _ = e.call(t, e.alisher, "list_expenses", `{"from": "2026-10-10", "to": "2026-10-16", "category": "food"}`)
	if got := itemComments(out); strings.Join(got, "|") != "кофе|продукты" {
		t.Errorf("period and category: %v", got)
	}

	// Limits above the maximum are capped instead of failing.
	if _, res := e.callLenient(t, e.alisher, "list_expenses", `{"limit": 500}`); res.Err != nil {
		t.Errorf("large limit: %v", res.Err)
	}
	// Empty arguments mean "latest expenses".
	if out, res := e.call(t, e.alisher, "list_expenses", ``); res.Err != nil || out["total"] != float64(4) {
		t.Errorf("no args: %v %v", out, res.Err)
	}

	for _, args := range []string{`{"sort": "amount"}`, `{"from": "15.10.2026"}`, `{"from": "2026-10-16", "to": "2026-10-01"}`} {
		out, res := e.call(t, e.alisher, "list_expenses", args)
		wantToolError(t, out, res, "")
	}
}

func TestSummarizeExpenses(t *testing.T) {
	e := newEnv(t)
	seedAlisher(t, e)

	out, res := e.call(t, e.alisher, "summarize_expenses", `{"from": "2026-10-01", "to": "2026-10-31"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if out["total"] != "5 530,50 ₽" || out["count"] != float64(4) || out["from"] != "2026-10-01" {
		t.Errorf("summary = %v", out)
	}
	byCat := out["by_category"].([]any)
	first := byCat[0].(map[string]any)
	if len(byCat) != 2 || first["category"] != "food" || first["category_name"] != "Еда" || first["total"] != "3 830,50 ₽" {
		t.Errorf("by_category = %v", byCat)
	}

	out, _ = e.call(t, e.alisher, "summarize_expenses", `{"from": "2026-10-15", "to": "2026-10-15", "category": "transport"}`)
	if out["total"] != "1 200 ₽" || out["count"] != float64(1) {
		t.Errorf("one day of transport = %v", out)
	}

	out, _ = e.call(t, e.alisher, "summarize_expenses", `{"from": "2025-01-01", "to": "2025-01-31"}`)
	if out["total"] != "0 ₽" || len(out["by_category"].([]any)) != 0 {
		t.Errorf("empty period = %v", out)
	}
}

func TestUpdateExpense(t *testing.T) {
	e := newEnv(t)
	ids := seedAlisher(t, e)
	coffee := jsonID(ids["кофе"])

	out, res := e.call(t, e.alisher, "update_expense", `{"id": `+coffee+`, "amount": "450"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	upd := out["updated"].(map[string]any)
	if upd["amount"] != "450 ₽" || upd["comment"] != "кофе" || upd["category"] != "food" {
		t.Errorf("updated = %v", upd)
	}

	out, _ = e.call(t, e.alisher, "update_expense", `{"id": `+coffee+`, "category": "entertainment", "spent_at": "2026-10-14", "comment": ""}`)
	upd = out["updated"].(map[string]any)
	if upd["category"] != "entertainment" || upd["spent_at"] != "2026-10-14" || upd["comment"] != nil {
		t.Errorf("updated = %v", upd)
	}

	cases := []struct{ name, args, field string }{
		{"nothing to change", `{"id": ` + coffee + `}`, ""},
		{"no id", `{"amount": "100"}`, ""},
		{"missing expense", `{"id": 999999, "amount": "100"}`, ""},
		{"bad amount", `{"id": ` + coffee + `, "amount": "-5"}`, "amount"},
		{"bad category", `{"id": ` + coffee + `, "category": "fun"}`, "category"},
		{"id as text", `{"id": "кофе"}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, res := e.call(t, e.alisher, "update_expense", tc.args)
			wantToolError(t, out, res, tc.field)
		})
	}
}

func TestDeleteExpense(t *testing.T) {
	e := newEnv(t)
	ids := seedAlisher(t, e)
	taxi := jsonID(ids["такси до работы"])

	out, res := e.call(t, e.alisher, "delete_expense", `{"id": `+taxi+`}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if d := out["deleted"].(map[string]any); d["comment"] != "такси до работы" || d["amount"] != "1 200 ₽" {
		t.Errorf("deleted = %v", d)
	}
	out, res = e.call(t, e.alisher, "delete_expense", `{"id": `+taxi+`}`)
	wantToolError(t, out, res, "")
	out, res = e.call(t, e.alisher, "delete_expense", `{}`)
	wantToolError(t, out, res, "")
}

// failingRepo breaks every read, like an unreachable database.
type failingRepo struct{ expense.Repository }

func (failingRepo) List(context.Context, expense.Filter) ([]expense.Expense, error) {
	return nil, errors.New("connection refused")
}

func TestRegistryErrors(t *testing.T) {
	e := newEnv(t)

	out, res := e.call(t, e.alisher, "transfer_money", `{}`)
	wantToolError(t, out, res, "")

	broken := expense.NewService(failingRepo{}, now, time.UTC).ForUser(alisher)
	res = e.reg.Call(context.Background(), broken, "list_expenses", `{}`)
	if !res.Internal || res.Err == nil || strings.Contains(res.Content, "connection refused") {
		t.Errorf("internal error = %+v (details must not reach the model)", res)
	}
}

func jsonID(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}
