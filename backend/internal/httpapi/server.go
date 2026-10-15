// Package httpapi is the REST API of the web panel (see api/openapi.yaml).
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// ExpenseService is the part of expense.Service the API uses.
type ExpenseService interface {
	List(ctx context.Context, f expense.Filter) (expense.Page, error)
	Summary(ctx context.Context, f expense.Filter) (expense.Summary, error)
	Get(ctx context.Context, id int64) (expense.Expense, error)
	Create(ctx context.Context, in expense.CreateInput) (expense.Expense, error)
	Update(ctx context.Context, id int64, in expense.UpdateInput) (expense.Expense, error)
	Delete(ctx context.Context, id int64) error
}

// UserService is the part of user.Service the API uses.
type UserService interface {
	List(ctx context.Context) ([]user.User, error)
}

// Pinger checks that the database is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps are the dependencies of the API handler.
type Deps struct {
	Expenses ExpenseService
	Users    UserService
	DB       Pinger
	Log      *slog.Logger
}

type api struct {
	Deps
}

// maxBodyBytes limits request bodies; an expense is well under 1 KiB.
const maxBodyBytes = 64 << 10

// healthTimeout bounds the database ping of /healthz.
const healthTimeout = 2 * time.Second

// NewHandler returns the HTTP handler with all routes and middleware.
func NewHandler(d Deps) http.Handler {
	a := &api{Deps: d}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/users", a.listUsers)
	mux.HandleFunc("GET /api/categories", a.listCategories)
	mux.HandleFunc("GET /api/expenses", a.listExpenses)
	mux.HandleFunc("GET /api/expenses/summary", a.summarizeExpenses)
	mux.HandleFunc("POST /api/expenses", a.createExpense)
	mux.HandleFunc("GET /api/expenses/{id}", a.getExpense)
	mux.HandleFunc("PUT /api/expenses/{id}", a.updateExpense)
	mux.HandleFunc("DELETE /api/expenses/{id}", a.deleteExpense)
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "Неизвестный адрес API", nil)
	})

	return requestID(accessLog(d.Log, recoverPanic(d.Log, mux)))
}

func (a *api) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()
	if err := a.DB.Ping(ctx); err != nil {
		a.Log.WarnContext(r.Context(), "health check failed", "error", err, "request_id", requestIDFrom(r.Context()))
		writeJSON(w, http.StatusServiceUnavailable, healthDTO{Status: "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, healthDTO{Status: "ok"})
}
