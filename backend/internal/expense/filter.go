package expense

import "time"

// SortField is a column the list can be sorted by.
type SortField string

// Sort fields. Only these reach SQL, so sorting cannot be used for injection.
const (
	SortSpentAt   SortField = "spent_at"
	SortAmount    SortField = "amount"
	SortCreatedAt SortField = "created_at"
)

// SortOrder is the sort direction.
type SortOrder string

// Sort directions.
const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

// Filter selects expenses. Zero values mean "no restriction" or "default".
type Filter struct {
	UserID   *int64
	From     *time.Time // inclusive
	To       *time.Time // inclusive
	Category *Category
	// Query is a case-insensitive substring of the comment.
	Query  string
	Sort   SortField
	Order  SortOrder
	Limit  int
	Offset int
}

// normalize fills defaults and checks f. Field keys match the REST query
// parameters, so the API can report them as they are.
func (f *Filter) normalize() error {
	errs := fieldErrors{}
	if f.Sort == "" {
		f.Sort = SortSpentAt
	}
	if f.Order == "" {
		f.Order = OrderDesc
	}
	if f.Limit == 0 {
		f.Limit = DefaultLimit
	}
	if f.From != nil {
		d := dateOnly(*f.From)
		f.From = &d
	}
	if f.To != nil {
		d := dateOnly(*f.To)
		f.To = &d
	}

	switch f.Sort {
	case SortSpentAt, SortAmount, SortCreatedAt:
	default:
		errs.add("sort", "Неизвестное поле сортировки")
	}
	switch f.Order {
	case OrderAsc, OrderDesc:
	default:
		errs.add("order", "Порядок сортировки — asc или desc")
	}
	if f.Limit < 1 || f.Limit > MaxLimit {
		errs.add("limit", "Размер страницы — от 1 до 100")
	}
	if f.Offset < 0 {
		errs.add("offset", "Смещение не может быть отрицательным")
	}
	if f.Category != nil && !f.Category.Valid() {
		errs.add("category", "Неизвестная категория")
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		errs.add("from", "Начало периода позже его конца")
	}
	return errs.err()
}

// Page is one page of a list and the number of all matching expenses.
type Page struct {
	Items []Expense
	Total int
}

// CategoryTotal is the sum of expenses in one category.
type CategoryTotal struct {
	Category Category
	Total    Money
	Count    int
}

// Summary aggregates all expenses matching a filter (pagination ignored).
// ByCategory holds only non-empty categories, largest total first.
type Summary struct {
	Total      Money
	Count      int
	ByCategory []CategoryTotal
}
