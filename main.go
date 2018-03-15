package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
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

	bids := make(map[OrderID]*OrderInfo)
	var lastTick int

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

		case TickerMessage:
			if msg.Seq < lastTick {
				panic("out of order ticks")
			}
			lastTick = msg.Seq

			for _, order := range bids {
				if msg.BestBid.GT(order.buyAt) || msg.BestAsk.LT(order.sellAt) {
					order.Cancel()
				}
			}

			if msg.Spread.GT(NewMoney(0, 50)) && len(bids) < 10 {
				fmt.Println("got a nice spread: ", msg.Spread)
				info, err := flip(client, msg)
				if err != nil {
					warn(err.Error())
					time.Sleep(time.Millisecond * 50)
				} else {
					bids[info.id] = info
				}
			}

		case FilledMessage:
			if msg.Side == Buy {
				info, ok := bids[msg.OrderID]
				if !ok {
					log.Fatalf("invalid order id: %v", msg.OrderID)
				}
				close(info.sell)
				delete(bids, info.id)
			} else {
				debug("nice flip, bruh")
			}

		case CancelMessage:
			if _, ok := bids[msg.OrderID]; !ok {
				fmt.Println(bids)
				warn("unknown order canceled: %v", msg.OrderID)
			} else {
				debug("order %v successfully canceled", msg.OrderID)
				delete(bids, msg.OrderID)
			}
		}
	}

	select {}
}

const orderSize = 0.00001

type OrderInfo struct {
	gdax *GdaxClient

	id            OrderID
	size          float64
	buyAt, sellAt Money
	sell, cancel  chan struct{}
}

func (o *OrderInfo) Cancel() {
	defer func() {
		if r := recover(); r != nil {
			debug("order was already canceled")
		}
	}()
	close(o.cancel)
}

// Flipping:
// 	goals:
//		- buy product at buyAt
//		- if price goes above buyAt, cancel the buy order
// 		- if price goes below buyAt, cancel the buy order
//		- sell product at sellAt price
//		- if price goes below buyAt while holding, keep the order
// algorithm:
//	0. on new ticker message:
//		a. if currentFlip exists
//	1. let M be a map[OrderID]struct{Order, buyAt , sellAt Money}
//	2. place order for product with size: s and price: p
//	3. put that order into M
//	4. on next ticker message, check current flip
//	5. if flip exists
var i int

func flip(
	c *GdaxClient,
	msg TickerMessage,
) (*OrderInfo, error) {
	buyAt := msg.BestBid.Plus(NewMoney(0, 4))
	sellAt := msg.BestAsk.Minus(NewMoney(0, 4))

	debug("flipping:\tbuy:%v\tsell:%v\n", buyAt, sellAt)

	orderID, err := c.Buy(msg.ProductID, buyAt, orderSize)
	if err != nil {
		return nil, err
	}

	info := &OrderInfo{
		id:     orderID,
		sell:   make(chan struct{}),
		cancel: make(chan struct{}),
		buyAt:  buyAt,
		sellAt: sellAt,
	}

	go func() {
		select {
		case <-info.sell:
			c.Sell(msg.ProductID, sellAt, orderSize)
		case <-info.cancel:
			debug("cancelling %v", orderID)
			c.Cancel(orderID)
		}
	}()

	return info, nil
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
	debugEnabled = true
	warnEnabled  = true
)

func debug(s string, m ...interface{}) {
	if debugEnabled {
		log.Printf("[debug] %s", fmt.Sprintf(s, m...))
	}
}

func warn(s string, m ...interface{}) {
	if warnEnabled {
		log.Printf("[warn] %s", fmt.Sprintf(s, m...))
	}
}
