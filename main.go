package main

import (
	"fmt"
	"log"
)

func main() {
	log.Printf("getting feed")

	feed, err := NewFeed()
	if err != nil {
		log.Fatalf("error creating feed: %v", err)
	}

	ch, err := feed.Messages()
	if err != nil {
		log.Fatalf("error getting messages from feed: %v", err)
	}

	for m := range ch {
		fmt.Printf("%s: %v\n", m.MessageType(), m)
	}

	// 	select {}
}

type ProductID string

const (
	EthUsd ProductID = "ETH-USD"
	BchUsd           = "BCH-USD"
	BtcUsd           = "BTC-USD"
	LtcUsd           = "LTC-USD"
)

type App struct {
	feed Feed
	book Book
}

type OrderID string

type OrderType string

// OrderType
const (
	Limit  OrderType = "limit"
	Market           = "Market"
	Stop             = "Stop"
)

type Side string

const (
	Buy  Side = "buy"
	Sell      = "sell"
)

var debugEnabled = true

func debug(s string, m ...interface{}) {
	if debugEnabled {
		log.Printf(s, m...)
	}
}
