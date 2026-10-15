package expense

import "fmt"

// Category is a stable English category code stored in the database.
type Category string

// Categories in their fixed display order.
const (
	CategoryFood          Category = "food"
	CategoryTransport     Category = "transport"
	CategoryHousing       Category = "housing"
	CategoryEntertainment Category = "entertainment"
	CategoryHealth        Category = "health"
	CategoryShopping      Category = "shopping"
	CategoryOther         Category = "other"
)

// CategoryInfo is a category with its Russian label.
type CategoryInfo struct {
	Code Category
	Name string
}

var categories = []CategoryInfo{
	{CategoryFood, "Еда"},
	{CategoryTransport, "Транспорт"},
	{CategoryHousing, "Жильё"},
	{CategoryEntertainment, "Развлечения"},
	{CategoryHealth, "Здоровье"},
	{CategoryShopping, "Покупки"},
	{CategoryOther, "Другое"},
}

// Categories returns all categories in display order.
func Categories() []CategoryInfo {
	return append([]CategoryInfo(nil), categories...)
}

// ParseCategory converts a code into a Category.
func ParseCategory(s string) (Category, error) {
	c := Category(s)
	if !c.Valid() {
		return "", fmt.Errorf("unknown category %q", s)
	}
	return c, nil
}

// Valid reports whether c is a known category.
func (c Category) Valid() bool {
	for _, info := range categories {
		if info.Code == c {
			return true
		}
	}
	return false
}

// Name returns the Russian label of c, or the code itself if c is unknown.
func (c Category) Name() string {
	for _, info := range categories {
		if info.Code == c {
			return info.Name
		}
	}
	return string(c)
}

// Source tells where an expense was created.
type Source string

// Expense sources.
const (
	SourceBot Source = "bot"
	SourceWeb Source = "web"
)
