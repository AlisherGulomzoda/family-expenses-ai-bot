package httpapi

import (
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// JSON shapes of api/openapi.yaml. Money marshals itself as rubles.

const dateLayout = time.DateOnly // YYYY-MM-DD

type userDTO struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Username *string `json:"username"`
}

func toUserDTO(u user.User) userDTO {
	return userDTO{ID: u.TelegramID, Name: u.Name, Username: u.Username}
}

type categoryDTO struct {
	Code expense.Category `json:"code"`
	Name string           `json:"name"`
}

type expenseDTO struct {
	ID        int64            `json:"id"`
	UserID    int64            `json:"user_id"`
	UserName  string           `json:"user_name"`
	Amount    expense.Money    `json:"amount"`
	Category  expense.Category `json:"category"`
	SpentAt   string           `json:"spent_at"`
	Comment   *string          `json:"comment"`
	Source    expense.Source   `json:"source"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func toExpenseDTO(e expense.Expense) expenseDTO {
	return expenseDTO{
		ID:        e.ID,
		UserID:    e.UserID,
		UserName:  e.UserName,
		Amount:    e.Amount,
		Category:  e.Category,
		SpentAt:   e.SpentAt.Format(dateLayout),
		Comment:   e.Comment,
		Source:    e.Source,
		CreatedAt: e.CreatedAt.UTC(),
		UpdatedAt: e.UpdatedAt.UTC(),
	}
}

type expenseListDTO struct {
	Items []expenseDTO `json:"items"`
	Total int          `json:"total"`
}

type categoryTotalDTO struct {
	Category expense.Category `json:"category"`
	Total    expense.Money    `json:"total"`
	Count    int              `json:"count"`
}

type summaryDTO struct {
	Total      expense.Money      `json:"total"`
	Count      int                `json:"count"`
	ByCategory []categoryTotalDTO `json:"by_category"`
}

func toSummaryDTO(s expense.Summary) summaryDTO {
	out := summaryDTO{Total: s.Total, Count: s.Count, ByCategory: make([]categoryTotalDTO, len(s.ByCategory))}
	for i, c := range s.ByCategory {
		out.ByCategory[i] = categoryTotalDTO(c)
	}
	return out
}

// expenseInput is the body of POST and PUT. Pointers tell a missing
// field from a zero value, so required fields can be reported.
type expenseInput struct {
	UserID   *int64         `json:"user_id"`
	Amount   *expense.Money `json:"amount"`
	Category *string        `json:"category"`
	SpentAt  *string        `json:"spent_at"`
	Comment  *string        `json:"comment"`
}

type healthDTO struct {
	Status string `json:"status"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type errorDTO struct {
	Error errorBody `json:"error"`
}
