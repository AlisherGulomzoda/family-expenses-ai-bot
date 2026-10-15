package expense

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParseRubles(t *testing.T) {
	tests := []struct {
		in   string
		want Money
	}{
		{"350", 35000},
		{"350.5", 35050},
		{"350,5", 35050},
		{"349,90", 34990},
		{"0.01", 1},
		{"0", 0},
		{"007", 700},
		{"1 200", 120000},
		{"1 250 000,50", 125000050},
		{"1 200", 120000},
		{"100000000", MaxMoney},
	}
	for _, tt := range tests {
		got, err := ParseRubles(tt.in)
		if err != nil {
			t.Errorf("ParseRubles(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRubles(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseRublesInvalid(t *testing.T) {
	for _, in := range []string{
		"", " ", "-350", "+350", "350.", ".5", "350.555", "3.5e2", "350р",
		"abc", "1.2.3", "1,2,3", "100000001", "99999999999999999999999",
	} {
		if _, err := ParseRubles(in); !errors.Is(err, ErrInvalidMoney) {
			t.Errorf("ParseRubles(%q) error = %v, want ErrInvalidMoney", in, err)
		}
	}
}

func TestMoneyFormatting(t *testing.T) {
	tests := []struct {
		m           Money
		rubles, str string
	}{
		{35000, "350", "350 ₽"},
		{35050, "350.5", "350,50 ₽"},
		{34990, "349.9", "349,90 ₽"},
		{1, "0.01", "0,01 ₽"},
		{120000, "1200", "1 200 ₽"},
		{125000050, "1250000.5", "1 250 000,50 ₽"},
		{125000005, "1250000.05", "1 250 000,05 ₽"},
	}
	for _, tt := range tests {
		if got := tt.m.Rubles(); got != tt.rubles {
			t.Errorf("Money(%d).Rubles() = %q, want %q", tt.m, got, tt.rubles)
		}
		if got := tt.m.String(); got != tt.str {
			t.Errorf("Money(%d).String() = %q, want %q", tt.m, got, tt.str)
		}
	}
}

func TestMoneyJSON(t *testing.T) {
	var v struct {
		Amount Money `json:"amount"`
	}
	if err := json.Unmarshal([]byte(`{"amount": 350.5}`), &v); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if v.Amount != 35050 {
		t.Fatalf("Amount = %d, want 35050", v.Amount)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(out) != `{"amount":350.5}` {
		t.Fatalf("Marshal = %s", out)
	}

	for _, bad := range []string{`"350"`, `3.505e2`, `350.123`, `-1`, `null`} {
		if err := json.Unmarshal([]byte(`{"amount": `+bad+`}`), &v); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want error", bad)
		}
	}
}
