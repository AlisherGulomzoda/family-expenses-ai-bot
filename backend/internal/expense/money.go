package expense

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Money is an amount in kopecks. Money never goes through float64:
// it is parsed from and formatted to decimal strings directly.
type Money int64

// MaxMoney is the largest accepted amount: 100 000 000 rubles.
const MaxMoney Money = 100_000_000 * 100

// ErrInvalidMoney is returned for strings that are not a ruble amount.
var ErrInvalidMoney = errors.New("invalid money amount")

// ParseRubles parses a ruble amount such as "350", "350.5", "349,90" or "1 200".
// Spaces (including non-breaking ones) are ignored, both "." and "," are
// accepted as the decimal separator, at most 2 decimal places are allowed.
// Signs, exponents and other characters are rejected. Zero parses fine:
// positivity is a validation rule, not a parsing one.
func ParseRubles(s string) (Money, error) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', ' ', ' ':
			return -1
		case ',':
			return '.'
		}
		return r
	}, s)

	whole, frac, hasFrac := strings.Cut(s, ".")
	if whole == "" || (hasFrac && (frac == "" || len(frac) > 2)) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidMoney, s)
	}
	if !isDigits(whole) || !isDigits(frac) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidMoney, s)
	}
	rubles, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || rubles > int64(MaxMoney/100) {
		return 0, fmt.Errorf("%w: %q is too large", ErrInvalidMoney, s)
	}
	kopecks := int64(0)
	if frac != "" {
		if len(frac) == 1 {
			frac += "0"
		}
		kopecks, _ = strconv.ParseInt(frac, 10, 64) // two digits, cannot fail
	}
	return Money(rubles*100 + kopecks), nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Rubles formats m as a plain decimal number of rubles for machines:
// "350", "350.5", "1250000.05". It is also the JSON representation.
func (m Money) Rubles() string {
	sign := ""
	if m < 0 {
		sign, m = "-", -m
	}
	whole, kop := int64(m)/100, int64(m)%100
	switch {
	case kop == 0:
		return fmt.Sprintf("%s%d", sign, whole)
	case kop%10 == 0:
		return fmt.Sprintf("%s%d.%d", sign, whole, kop/10)
	default:
		return fmt.Sprintf("%s%d.%02d", sign, whole, kop)
	}
}

// String formats m for people, the way the UI does: "1 200 ₽", "349,90 ₽".
// Groups are separated with a non-breaking space.
func (m Money) String() string {
	sign := ""
	if m < 0 {
		sign, m = "-", -m
	}
	whole, kop := int64(m)/100, int64(m)%100
	digits := strconv.FormatInt(whole, 10)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	if kop != 0 {
		fmt.Fprintf(&b, ",%02d", kop)
	}
	return sign + b.String() + " ₽"
}

// MarshalJSON encodes m as a JSON number of rubles.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.Rubles()), nil
}

// UnmarshalJSON decodes a JSON number of rubles. Strings, exponents
// and more than 2 decimal places are rejected.
func (m *Money) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || strings.ContainsAny(s, `" ,`) {
		return fmt.Errorf("%w: expected a JSON number", ErrInvalidMoney)
	}
	v, err := ParseRubles(s)
	if err != nil {
		return err
	}
	*m = v
	return nil
}
