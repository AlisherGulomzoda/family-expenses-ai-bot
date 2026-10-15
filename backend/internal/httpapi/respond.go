package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the client is gone if this fails
}

func writeError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	writeJSON(w, status, errorDTO{Error: errorBody{Code: code, Message: message, Fields: fields}})
}

func badRequest(w http.ResponseWriter, fields map[string]string) {
	writeError(w, http.StatusBadRequest, "bad_request", "Некорректные параметры запроса", fields)
}

func invalidBody(w http.ResponseWriter, fields map[string]string) {
	writeError(w, http.StatusUnprocessableEntity, "validation_error", "Проверьте заполнение полей", fields)
}

// where an error came from decides how a validation error is reported:
// invalid query parameters are a malformed request (400), invalid body
// values are a well-formed request with bad data (422).
type origin int

const (
	fromQuery origin = iota
	fromBody
)

// fail maps a service error to a response. Unexpected errors are logged
// with the request ID and hidden from the client.
func (a *api) fail(w http.ResponseWriter, r *http.Request, err error, o origin) {
	var ve *expense.ValidationError
	switch {
	case errors.Is(err, expense.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Трата не найдена", nil)
	case errors.As(err, &ve) && o == fromQuery:
		badRequest(w, ve.Fields)
	case errors.As(err, &ve):
		invalidBody(w, ve.Fields)
	default:
		a.Log.ErrorContext(r.Context(), "request failed",
			"error", err, "method", r.Method, "path", r.URL.Path, "request_id", requestIDFrom(r.Context()))
		writeError(w, http.StatusInternalServerError, "internal", "Что-то пошло не так. Попробуйте ещё раз", nil)
	}
}
