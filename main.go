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

	book, err := NewBook()
	if err != nil {
		log.Fatalf("error creating book: %v", err)
	}

	var i int
	for m := range ch {
		i++
		if i > 10 {
			break
		}

		fmt.Printf("%s: %v\n", m.MessageType(), m)

		switch msg := m.(type) {
		case SnapshotMessage:
			for _, ask := range msg.Asks {
				book.update(ask)
			}
			for _, bid := range msg.Bids {
				book.update(bid)
			}
		case L2UpdateMessage:
			for _, ask := range msg.Asks {
				book.update(ask)
			}
			for _, bid := range msg.Bids {
				book.update(bid)
			}
		}
	}

	fmt.Println(book)

	bid := book.bids.Peek()
	ask := book.asks.Peek()

	spread := ask.Price.Minus(bid.Price)

	fmt.Printf("lowest  ask: %v\n", ask)
	fmt.Printf("highest bid: %v\n", bid)

	fmt.Printf("spread: %v\n", spread)

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
