package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Money struct {
	cents int64
}

func NewMoney(dollars, cents int64) Money {
	return Money{dollars * 100}
}
func MoneyFromString(s string) (Money, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		dollars, err := strconv.ParseInt(parts[0], 10, 64)
		return Money{cents: dollars * 100}, err
	}
	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("invalid money: %v", s)
	}
	cents, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("invalid money: %v", s)
	}
	return Money{dollars*100 + cents}, nil
}

func (m Money) String() string {
	return fmt.Sprintf("%d.%02d", m.cents/100, m.cents%100)
}

func (m Money) Int64() int64 {
	return m.cents
}

func (m Money) Plus(that Money) Money {
	return Money{m.cents + that.cents}
}

func (m Money) Minus(that Money) Money {
	return Money{m.cents - that.cents}
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}
func (m *Money) UnmarshalJSON(b []byte) error {
	return nil
}
