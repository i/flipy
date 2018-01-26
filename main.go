package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func authFromEnv() AuthInfo {
	return AuthInfo{
		Key:        os.Getenv("CB_KEY"),
		Passphrase: os.Getenv("CB_PASSPHRASE"),
		Secret:     os.Getenv("CB_SECRET"),
	}
}

func launchPprof() {
	go http.ListenAndServe(":3000", nil)
}

func main() {
	log.Printf("getting feed")

	launchPprof()

	auth := authFromEnv()

	feed, err := NewFeed(WithAuth(auth))
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

	client, err := NewGdaxClient(
		WithBaseURL("https://api.gdax.com"),
		//WithBaseURL("https://api-public.gdax.com"),
		WithAuthInfo(auth),
	)
	if err != nil {
		log.Fatalf("error creating gdax http client: %v", err)
	}

	_ = client
	//orderID, err := client.buy(BchUsd, NewMoney(140, 0), 0.01)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Println("orderID:", orderID)
	//return

	for m := range ch {
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
		spread, err := book.Spread()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("spread:", spread)
		fmt.Println("size:", book.Size())
	}

	b := book.Dump()
	for _, e := range b {
		fmt.Println(e)
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

var (
	debugEnabled = false
	warnEnabled  = false
)

func debug(s string, m ...interface{}) {
	if debugEnabled {
		log.Print("[debug] ")
		log.Printf(s, m...)
	}
}

func warn(s string, m ...interface{}) {
	if warnEnabled {
		log.Print("[warn] ")
		log.Printf(s, m...)
	}
}
