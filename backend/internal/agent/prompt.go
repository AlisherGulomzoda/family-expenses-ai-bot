package agent

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
	"time"
)

//go:embed prompt.tmpl
var promptText string

var promptTmpl = template.Must(template.New("prompt").Parse(promptText))

// promptData is computed in Go, not by the model: date arithmetic is a
// common source of LLM mistakes, so the boundaries of the usual periods
// are given ready-made.
type promptData struct {
	UserName      string
	Timezone      string
	Today         string
	Weekday       string
	Yesterday     string
	WeekStart     string
	WeekEnd       string
	MonthStart    string
	MonthEnd      string
	PrevMonthFrom string
	PrevMonthTo   string
	// LastWeek is "суббота: 2026-10-10" for each of the 7 days before today.
	LastWeek []string
}

var weekdays = [...]string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

// systemPrompt renders the system prompt for today (a date at midnight UTC).
func systemPrompt(userName, timezone string, today time.Time) (string, error) {
	const day = time.DateOnly
	// Monday-based week, as in Russia.
	offset := (int(today.Weekday()) + 6) % 7
	weekStart := today.AddDate(0, 0, -offset)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevMonthStart := monthStart.AddDate(0, -1, 0)

	data := promptData{
		UserName:      userName,
		Timezone:      timezone,
		Today:         today.Format(day),
		Weekday:       weekdays[today.Weekday()],
		Yesterday:     today.AddDate(0, 0, -1).Format(day),
		WeekStart:     weekStart.Format(day),
		WeekEnd:       weekStart.AddDate(0, 0, 6).Format(day),
		MonthStart:    monthStart.Format(day),
		MonthEnd:      monthStart.AddDate(0, 1, -1).Format(day),
		PrevMonthFrom: prevMonthStart.Format(day),
		PrevMonthTo:   monthStart.AddDate(0, 0, -1).Format(day),
	}
	for i := 7; i >= 1; i-- {
		d := today.AddDate(0, 0, -i)
		data.LastWeek = append(data.LastWeek, weekdays[d.Weekday()]+": "+d.Format(day))
	}
	var b bytes.Buffer
	if err := promptTmpl.Execute(&b, data); err != nil {
		return "", fmt.Errorf("render system prompt: %w", err)
	}
	return b.String(), nil
}
