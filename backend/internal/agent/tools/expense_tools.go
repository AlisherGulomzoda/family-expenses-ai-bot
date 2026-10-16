package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
)

// Tool descriptions are part of the prompt: they tell the model when and
// how to use each tool. Keep them precise and keep examples in Russian,
// the language of the users.

const categoryEnum = `["food", "transport", "housing", "entertainment", "health", "shopping", "other"]`

const categoryHelp = `Category code: food (еда, кофе, продукты, кафе, ресторан, доставка еды), ` +
	`transport (такси, метро, автобус, бензин, парковка), housing (аренда, коммуналка, ремонт, интернет), ` +
	`entertainment (кино, концерт, игры, подписки, хобби), health (аптека, врач, анализы, спортзал), ` +
	`shopping (одежда, обувь, техника, товары для дома, подарки), other (всё остальное).`

const (
	maxListLimit     = 50
	defaultListLimit = 10
)

func addExpense() Tool {
	return Tool{
		Name: "add_expense",
		Description: `Record a new expense of the user. Call it once per expense; ` +
			`for "кофе 350, такси 500" call it twice. Never invent the amount: if the user did not give one, ask instead.`,
		Schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "amount":   {"type": "string", "description": "Amount in rubles as a decimal string: \"350\", \"349.90\". At most 2 decimal places."},
    "category": {"type": "string", "enum": ` + categoryEnum + `, "description": "` + categoryHelp + `"},
    "spent_at": {"type": "string", "format": "date", "description": "Date of the expense, YYYY-MM-DD. Omit for today. Resolve words like \"вчера\" against today's date from the system prompt."},
    "comment":  {"type": "string", "description": "What the money was spent on, in the user's words: \"кофе\", \"такси до работы\"."}
  },
  "required": ["amount", "category"]
}`),
		Handler: func(ctx context.Context, scope expense.UserScope, raw []byte) (any, error) {
			var args struct {
				Amount   flexAmount `json:"amount"`
				Category string     `json:"category"`
				SpentAt  string     `json:"spent_at"`
				Comment  string     `json:"comment"`
			}
			if err := decode(raw, &args); err != nil {
				return nil, err
			}
			if !args.Amount.set {
				return nil, badArgs("Не указана сумма (amount). Спроси сумму у пользователя.")
			}
			draft := expense.Draft{Amount: args.Amount.value, Category: expense.Category(args.Category)}
			if args.SpentAt != "" {
				d, err := parseDate("spent_at", args.SpentAt)
				if err != nil {
					return nil, err
				}
				draft.SpentAt = d
			}
			if args.Comment != "" {
				draft.Comment = &args.Comment
			}
			e, err := scope.Create(ctx, draft)
			if err != nil {
				return nil, err
			}
			return struct {
				Created expenseView `json:"created"`
			}{view(e)}, nil
		},
	}
}

func listExpenses() Tool {
	return Tool{
		Name: "list_expenses",
		Description: `Find the user's expenses, newest first. Use it to show expenses ("покажи траты за неделю") ` +
			`and to find the expense to change or delete: by comment text ("удали такси" → query "такси") ` +
			`or the last recorded one ("последнее было не 350, а 450" → sort "created_at", limit 1). ` +
			`It only reads: to change or delete the expense you found, call update_expense or delete_expense with its id.`,
		Schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "from":     {"type": "string", "format": "date", "description": "Period start, inclusive, YYYY-MM-DD."},
    "to":       {"type": "string", "format": "date", "description": "Period end, inclusive, YYYY-MM-DD."},
    "category": {"type": "string", "enum": ` + categoryEnum + `},
    "query":    {"type": "string", "description": "Case-insensitive text to look for in the comment: \"такси\"."},
    "sort":     {"type": "string", "enum": ["spent_at", "created_at"], "description": "spent_at (default): by expense date; created_at: by when it was recorded, for \"последнее\"."},
    "limit":    {"type": "integer", "minimum": 1, "maximum": 50, "description": "How many expenses to return, default 10."}
  }
}`),
		Handler: func(ctx context.Context, scope expense.UserScope, raw []byte) (any, error) {
			var args struct {
				From     string `json:"from"`
				To       string `json:"to"`
				Category string `json:"category"`
				Query    string `json:"query"`
				Sort     string `json:"sort"`
				Limit    int    `json:"limit"`
			}
			if err := decode(raw, &args); err != nil {
				return nil, err
			}
			f, err := periodFilter(args.From, args.To, args.Category)
			if err != nil {
				return nil, err
			}
			f.Query = strings.TrimSpace(args.Query)
			f.Order = expense.OrderDesc
			switch args.Sort {
			case "", string(expense.SortSpentAt):
				f.Sort = expense.SortSpentAt
			case string(expense.SortCreatedAt):
				f.Sort = expense.SortCreatedAt
			default:
				return nil, badArgs(`sort должен быть "spent_at" или "created_at"`)
			}
			f.Limit = args.Limit
			if f.Limit <= 0 {
				f.Limit = defaultListLimit
			}
			f.Limit = min(f.Limit, maxListLimit)

			page, err := scope.List(ctx, f)
			if err != nil {
				return nil, err
			}
			items := make([]expenseView, len(page.Items))
			for i, e := range page.Items {
				items[i] = view(e)
			}
			return struct {
				Items []expenseView `json:"items"`
				Total int           `json:"total"`
				// The eval caught the model reporting "удалил" after only listing:
				// the reminder sits right next to the ids it needs.
				Note string `json:"note"`
			}{items, page.Total, "Это только поиск, ничего не изменено. Чтобы изменить или удалить трату, вызови update_expense или delete_expense с её id."}, nil
		},
	}
}

func summarizeExpenses() Tool {
	return Tool{
		Name: "summarize_expenses",
		Description: `Get exact totals of the user's expenses: the sum, the count and the sum per category. ` +
			`Use it for every "сколько потратил..." question instead of adding numbers yourself.`,
		Schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "from":     {"type": "string", "format": "date", "description": "Period start, inclusive, YYYY-MM-DD. Omit both from and to for all time."},
    "to":       {"type": "string", "format": "date", "description": "Period end, inclusive, YYYY-MM-DD."},
    "category": {"type": "string", "enum": ` + categoryEnum + `}
  }
}`),
		Handler: func(ctx context.Context, scope expense.UserScope, raw []byte) (any, error) {
			var args struct {
				From     string `json:"from"`
				To       string `json:"to"`
				Category string `json:"category"`
			}
			if err := decode(raw, &args); err != nil {
				return nil, err
			}
			f, err := periodFilter(args.From, args.To, args.Category)
			if err != nil {
				return nil, err
			}
			sum, err := scope.Summary(ctx, f)
			if err != nil {
				return nil, err
			}
			type catTotal struct {
				Category     expense.Category `json:"category"`
				CategoryName string           `json:"category_name"`
				Total        string           `json:"total"`
				Count        int              `json:"count"`
			}
			out := struct {
				From       string     `json:"from,omitempty"`
				To         string     `json:"to,omitempty"`
				Total      string     `json:"total"`
				Count      int        `json:"count"`
				ByCategory []catTotal `json:"by_category"`
			}{From: args.From, To: args.To, Total: sum.Total.String(), Count: sum.Count, ByCategory: []catTotal{}}
			for _, c := range sum.ByCategory {
				out.ByCategory = append(out.ByCategory, catTotal{c.Category, c.Category.Name(), c.Total.String(), c.Count})
			}
			return out, nil
		},
	}
}

func updateExpense() Tool {
	return Tool{
		Name: "update_expense",
		Description: `Change an expense of the user by id. Pass only the fields to change. ` +
			`Find the id with list_expenses first; if several expenses fit the request, ask the user which one.`,
		Schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "id":       {"type": "integer", "description": "Expense id from list_expenses or add_expense."},
    "amount":   {"type": "string", "description": "New amount in rubles: \"450\", \"449.90\"."},
    "category": {"type": "string", "enum": ` + categoryEnum + `},
    "spent_at": {"type": "string", "format": "date", "description": "New date, YYYY-MM-DD."},
    "comment":  {"type": "string", "description": "New comment; empty string removes it."}
  },
  "required": ["id"]
}`),
		Handler: func(ctx context.Context, scope expense.UserScope, raw []byte) (any, error) {
			var args struct {
				ID       int64      `json:"id"`
				Amount   flexAmount `json:"amount"`
				Category *string    `json:"category"`
				SpentAt  *string    `json:"spent_at"`
				Comment  *string    `json:"comment"`
			}
			if err := decode(raw, &args); err != nil {
				return nil, err
			}
			if args.ID <= 0 {
				return nil, badArgs("Не указан id траты. Найди его через list_expenses.")
			}
			var p expense.Patch
			if args.Amount.set {
				p.Amount = &args.Amount.value
			}
			if args.Category != nil {
				c := expense.Category(*args.Category)
				p.Category = &c
			}
			if args.SpentAt != nil {
				d, err := parseDate("spent_at", *args.SpentAt)
				if err != nil {
					return nil, err
				}
				p.SpentAt = &d
			}
			p.Comment = args.Comment
			if p.IsEmpty() {
				return nil, badArgs("Не указано, что изменить: передай amount, category, spent_at или comment.")
			}
			e, err := scope.Update(ctx, args.ID, p)
			if err != nil {
				return nil, err
			}
			return struct {
				Updated expenseView `json:"updated"`
			}{view(e)}, nil
		},
	}
}

func deleteExpense() Tool {
	return Tool{
		Name: "delete_expense",
		Description: `Delete an expense of the user by id. Find the id with list_expenses first. ` +
			`If exactly one expense fits the request, delete it and tell the user which one was deleted; ` +
			`if several fit, list them and ask which one to delete.`,
		Schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "id": {"type": "integer", "description": "Expense id from list_expenses."}
  },
  "required": ["id"]
}`),
		Handler: func(ctx context.Context, scope expense.UserScope, raw []byte) (any, error) {
			var args struct {
				ID int64 `json:"id"`
			}
			if err := decode(raw, &args); err != nil {
				return nil, err
			}
			if args.ID <= 0 {
				return nil, badArgs("Не указан id траты. Найди его через list_expenses.")
			}
			e, err := scope.Delete(ctx, args.ID)
			if err != nil {
				return nil, err
			}
			return struct {
				Deleted expenseView `json:"deleted"`
			}{view(e)}, nil
		},
	}
}

// expenseView is the compact form of an expense shown to the model.
type expenseView struct {
	ID           int64            `json:"id"`
	Amount       string           `json:"amount"`
	Category     expense.Category `json:"category"`
	CategoryName string           `json:"category_name"`
	SpentAt      string           `json:"spent_at"`
	Comment      string           `json:"comment,omitempty"`
}

func view(e expense.Expense) expenseView {
	v := expenseView{
		ID:           e.ID,
		Amount:       e.Amount.String(),
		Category:     e.Category,
		CategoryName: e.Category.Name(),
		SpentAt:      e.SpentAt.Format(time.DateOnly),
	}
	if e.Comment != nil {
		v.Comment = *e.Comment
	}
	return v
}

// decode parses tool arguments. Unknown fields are ignored on purpose:
// a model adding "user_id" or "currency" changes nothing, and failing
// would only cost another round trip.
func decode(raw []byte, dst any) error {
	err := json.Unmarshal(raw, dst)
	var ae *argError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ae): // a field decoder (flexAmount) already explained it
		return ae
	default:
		return badArgs("Аргументы должны быть JSON-объектом по схеме инструмента: " + err.Error())
	}
}

func parseDate(field, s string) (time.Time, error) {
	d, err := time.Parse(time.DateOnly, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, &argError{msg: "Дата в формате ГГГГ-ММ-ДД", fields: map[string]string{field: "ожидается ГГГГ-ММ-ДД, получено " + strconv.Quote(s)}}
	}
	return d, nil
}

func periodFilter(from, to, category string) (expense.Filter, error) {
	var f expense.Filter
	if from != "" {
		d, err := parseDate("from", from)
		if err != nil {
			return f, err
		}
		f.From = &d
	}
	if to != "" {
		d, err := parseDate("to", to)
		if err != nil {
			return f, err
		}
		f.To = &d
	}
	if category != "" {
		c := expense.Category(category)
		f.Category = &c
	}
	return f, nil
}

// flexAmount accepts the amount as a string ("350.5", the schema) or as a
// number (350.5, what weaker models sometimes send). Either way it is
// parsed from text, never through float64.
type flexAmount struct {
	value expense.Money
	set   bool
}

func (a *flexAmount) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if string(data) == "null" {
		return nil
	}
	text := string(data)
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
	}
	m, err := expense.ParseRubles(text)
	if err != nil {
		return &argError{
			msg:    "Сумма — число рублей, не больше двух знаков после запятой",
			fields: map[string]string{"amount": "получено " + string(data)},
		}
	}
	a.value, a.set = m, true
	return nil
}
