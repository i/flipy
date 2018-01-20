package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Money struct {
	dollars int64
	cents   int64
}

func NewMoney(dollars, cents int64) Money {
	return Money{dollars * 100, cents}
}
func MoneyFromString(s string) (Money, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		dollars, err := strconv.ParseInt(parts[0], 10, 64)
		return Money{dollars: dollars}, err
	}
	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("invalid money: %v", s)
	}
	cents, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("invalid money: %v", s)
	}
	return Money{dollars, cents}, nil
}

func (m Money) String() string {
	return fmt.Sprintf("%d.%d", m.dollars, m.cents)
}

func (m Money) Int64() int64 {
	return m.dollars*100 + m.cents
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}
func (m *Money) UnmarshalJSON(b []byte) error {
	return nil
}
