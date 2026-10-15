package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
)

func (a *api) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.Users.List(r.Context())
	if err != nil {
		a.fail(w, r, err, fromQuery)
		return
	}
	out := make([]userDTO, len(users))
	for i, u := range users {
		out[i] = toUserDTO(u)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) listCategories(w http.ResponseWriter, _ *http.Request) {
	cats := expense.Categories()
	out := make([]categoryDTO, len(cats))
	for i, c := range cats {
		out[i] = categoryDTO(c)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) listExpenses(w http.ResponseWriter, r *http.Request) {
	f, fields := parseFilter(r.URL.Query(), true)
	if len(fields) > 0 {
		badRequest(w, fields)
		return
	}
	page, err := a.Expenses.List(r.Context(), f)
	if err != nil {
		a.fail(w, r, err, fromQuery)
		return
	}
	out := expenseListDTO{Items: make([]expenseDTO, len(page.Items)), Total: page.Total}
	for i, e := range page.Items {
		out.Items[i] = toExpenseDTO(e)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) summarizeExpenses(w http.ResponseWriter, r *http.Request) {
	f, fields := parseFilter(r.URL.Query(), false)
	if len(fields) > 0 {
		badRequest(w, fields)
		return
	}
	sum, err := a.Expenses.Summary(r.Context(), f)
	if err != nil {
		a.fail(w, r, err, fromQuery)
		return
	}
	writeJSON(w, http.StatusOK, toSummaryDTO(sum))
}

func (a *api) getExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	e, err := a.Expenses.Get(r.Context(), id)
	if err != nil {
		a.fail(w, r, err, fromQuery)
		return
	}
	writeJSON(w, http.StatusOK, toExpenseDTO(e))
}

func (a *api) createExpense(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	e, err := a.Expenses.Create(r.Context(), expense.CreateInput{
		UserID: *in.UserID,
		Draft: expense.Draft{
			Amount:   *in.Amount,
			Category: expense.Category(*in.Category),
			SpentAt:  in.spentAt,
			Comment:  in.Comment,
		},
	})
	if err != nil {
		a.fail(w, r, err, fromBody)
		return
	}
	w.Header().Set("Location", "/api/expenses/"+strconv.FormatInt(e.ID, 10))
	writeJSON(w, http.StatusCreated, toExpenseDTO(e))
}

func (a *api) updateExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	// PUT replaces every editable field; a missing or null comment clears it.
	comment := ""
	if in.Comment != nil {
		comment = *in.Comment
	}
	category := expense.Category(*in.Category)
	e, err := a.Expenses.Update(r.Context(), id, expense.UpdateInput{
		UserID: in.UserID,
		Patch: expense.Patch{
			Amount:   in.Amount,
			Category: &category,
			SpentAt:  &in.spentAt,
			Comment:  &comment,
		},
	})
	if err != nil {
		a.fail(w, r, err, fromBody)
		return
	}
	writeJSON(w, http.StatusOK, toExpenseDTO(e))
}

func (a *api) deleteExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.Expenses.Delete(r.Context(), id); err != nil {
		a.fail(w, r, err, fromQuery)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		badRequest(w, map[string]string{"id": "Идентификатор траты — положительное целое число"})
		return 0, false
	}
	return id, true
}

// parsedInput is a decoded body with required fields present and the date parsed.
type parsedInput struct {
	expenseInput
	spentAt time.Time
}

// readInput decodes the body of POST/PUT. Malformed JSON and unknown fields
// are a 400; missing required fields and unparsable values are a 422.
// Range and category checks are left to the domain.
func readInput(w http.ResponseWriter, r *http.Request) (parsedInput, bool) {
	var in parsedInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	err := dec.Decode(&in.expenseInput)
	if err == nil && dec.More() {
		err = errors.New("unexpected data after the JSON object")
	}

	var typeErr *json.UnmarshalTypeError
	var maxErr *http.MaxBytesError
	switch {
	case err == nil:
	case errors.Is(err, expense.ErrInvalidMoney):
		invalidBody(w, map[string]string{"amount": "Сумма — число, не больше двух знаков после запятой"})
		return in, false
	case errors.As(err, &typeErr):
		badRequest(w, map[string]string{typeErr.Field: "Неверный тип значения"})
		return in, false
	case errors.As(err, &maxErr):
		writeError(w, http.StatusRequestEntityTooLarge, "bad_request", "Слишком большой запрос", nil)
		return in, false
	case errors.Is(err, io.EOF):
		badRequest(w, map[string]string{"body": "Пустое тело запроса"})
		return in, false
	default:
		badRequest(w, map[string]string{"body": "Некорректный JSON: " + jsonErrorText(err)})
		return in, false
	}

	fields := map[string]string{}
	if in.UserID == nil {
		fields["user_id"] = "Выберите члена семьи"
	}
	if in.Amount == nil {
		fields["amount"] = "Укажите сумму"
	}
	if in.Category == nil {
		fields["category"] = "Выберите категорию"
	}
	if in.SpentAt == nil {
		fields["spent_at"] = "Укажите дату"
	} else if d, err := time.Parse(dateLayout, *in.SpentAt); err != nil {
		fields["spent_at"] = "Дата в формате ГГГГ-ММ-ДД"
	} else {
		in.spentAt = d
	}
	if len(fields) > 0 {
		invalidBody(w, fields)
		return in, false
	}
	return in, true
}

// jsonErrorText keeps decoder messages short and free of Go type names.
func jsonErrorText(err error) string {
	msg := err.Error()
	if strings.HasPrefix(msg, "json: unknown field ") {
		return "неизвестное поле " + strings.TrimPrefix(msg, "json: unknown field ")
	}
	return strings.TrimPrefix(msg, "json: ")
}

// parseFilter reads list/summary query parameters. Format errors are
// returned per parameter; value rules (ranges, from <= to) are checked
// by the domain and reported the same way.
func parseFilter(q url.Values, withPaging bool) (expense.Filter, map[string]string) {
	var f expense.Filter
	fields := map[string]string{}

	if v := q.Get("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			fields["user_id"] = "Ожидается Telegram ID (целое число)"
		} else {
			f.UserID = &id
		}
	}
	for _, p := range []struct {
		name string
		dst  **time.Time
	}{{"from", &f.From}, {"to", &f.To}} {
		if v := q.Get(p.name); v != "" {
			d, err := time.Parse(dateLayout, v)
			if err != nil {
				fields[p.name] = "Ожидается дата в формате ГГГГ-ММ-ДД"
				continue
			}
			*p.dst = &d
		}
	}
	if v := q.Get("category"); v != "" {
		c, err := expense.ParseCategory(v)
		if err != nil {
			fields["category"] = "Неизвестная категория"
		} else {
			f.Category = &c
		}
	}
	if withPaging {
		f.Sort = expense.SortField(q.Get("sort"))
		f.Order = expense.SortOrder(q.Get("order"))
		for _, p := range []struct {
			name string
			dst  *int
		}{{"limit", &f.Limit}, {"offset", &f.Offset}} {
			if v := q.Get(p.name); v != "" {
				n, err := strconv.Atoi(v)
				if err != nil {
					fields[p.name] = fmt.Sprintf("Ожидается целое число, получено %q", v)
					continue
				}
				*p.dst = n
			}
		}
		// limit=0 would silently mean "default"; reject it explicitly.
		if q.Get("limit") == "0" {
			fields["limit"] = "Размер страницы — от 1 до 100"
		}
	}
	return f, fields
}
