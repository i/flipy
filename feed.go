package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const gdaxFeedURL = "wss://ws-feed.gdax.com"

type Feed struct {
	conn *websocket.Conn
	auth AuthInfo
}

type Message interface {
	MessageType() MessageType
}

func (f *Feed) Messages() (<-chan Message, error) {
	ch := make(chan Message)
	go func() {
		for {
			_, bb, err := f.conn.ReadMessage()
			if err != nil {
				log.Printf("can't read message: %v", err)
				return
			}
			msg, err := parseMessage(bb)
			if err != nil {
				log.Fatalf("Error parsing message: %v", err)
			}
			ch <- msg
		}
	}()
	return ch, nil
}

type FeedOption func(f *Feed)

func WithAuth(info AuthInfo) FeedOption {
	return func(f *Feed) {
		f.auth = info
	}
}

func NewFeed(opts ...FeedOption) (*Feed, error) {
	c, _, err := websocket.DefaultDialer.Dial(gdaxFeedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error dialing feed: %v", err)
	}

	feed := &Feed{conn: c}
	for _, o := range opts {
		o(feed)
	}

	if err := feed.Subscribe([]ProductID{BchUsd}); err != nil {
		return nil, err
	}
	return feed, nil
}

// TODO
type channelSpec interface{}

func (f *Feed) Subscribe(productIDs []ProductID) error {
	ts, sig := f.auth.signature("GET", "/users/self/verify", "")

	payload, err := json.Marshal(struct {
		Type       string        `json:"type"`
		ProductIDs []ProductID   `json:"product_ids"`
		Channels   []channelSpec `json:"channels"`
		// Authenticated fields
		Key        string `json:"key,omitempty"`
		Passphrase string `json:"passphrase,omitempty"`
		Timestamp  string `json:"timestamp,omitempty"`
		Signature  string `json:"signature,omitempty"`
	}{
		Type:       "subscribe",
		ProductIDs: productIDs,
		Channels: []channelSpec{
			"heartbeat",
			"ticker",
			"user",
		},
		Key:        f.auth.Key,
		Passphrase: f.auth.Passphrase,
		Timestamp:  strconv.FormatInt(ts.Unix(), 10),
		Signature:  sig,
	})
	if err != nil {
		return fmt.Errorf("error creating subscribe payload: %v", err)
	}

	if err := f.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		return fmt.Errorf("Unable to subscribe: %v", err)
	}

	return nil
}

type MessageType string

const (
	L2Update      MessageType = "l2update"
	Snapshot                  = "snapshot"
	Heartbeat                 = "heartbeat"
	Subscriptions             = "subscriptions"
	Error                     = "error"
	Ticker                    = "ticker"
	Received                  = "received"
	Open                      = "open"
	Done                      = "done"
	Match                     = "match"
)

func messageType(bb []byte) (MessageType, error) {
	t := new(struct {
		Type MessageType `json:"type"`
	})
	return t.Type, json.Unmarshal(bb, t)
}

func parseMessage(bb []byte) (Message, error) {
	mt, err := messageType(bb)
	if err != nil {
		log.Printf("invalid message: %v", err)
	}

	switch mt {
	case Snapshot:
		return parseSnapshot(bb)
	case L2Update:
		return parseL2Update(bb)
	case Heartbeat:
		return parseHeartbeat(bb)
	case Subscriptions:
		return parseSubscriptions(bb)
	case Ticker:
		return parseTicker(bb)
	case Received:
		return nil, nil
	case Open:
		return nil, nil
	case Match:
		return parseMatch(bb)
	case Done:
		return parseDone(bb)
	case Error:
		return nil, fmt.Errorf("received error: %v", string(bb))
	default:
		return nil, fmt.Errorf("unknown message type: %v\n%s", mt, string(bb))
	}
}

type SnapshotMessage struct {
	ProductID ProductID
	Bids      []*bookEntry
	Asks      []*bookEntry
}

func (s SnapshotMessage) MessageType() MessageType {
	return Snapshot
}

// dry this later
type L2UpdateMessage struct {
	ProductID ProductID
	Bids      []*bookEntry
	Asks      []*bookEntry
}

func (s L2UpdateMessage) MessageType() MessageType {
	return L2Update
}

type HeartbeatMessage struct {
}

func (s HeartbeatMessage) MessageType() MessageType {
	return Heartbeat
}

type SubscriptionsMessage struct {
}

func (s SubscriptionsMessage) MessageType() MessageType {
	return Subscriptions
}

func parseSnapshot(bb []byte) (SnapshotMessage, error) {
	p := new(struct {
		Type      string     `json:"type"` // always snapshot
		ProductID ProductID  `json:"product_id"`
		Asks      [][]string `json:"asks"`
		Bids      [][]string `json:"bids"`
	})
	if err := json.Unmarshal(bb, p); err != nil {
		return SnapshotMessage{}, err
	}
	//	fmt.Println(p)

	msg := SnapshotMessage{
		ProductID: p.ProductID,
		Asks:      make([]*bookEntry, 0, len(p.Asks)),
		Bids:      make([]*bookEntry, 0, len(p.Bids)),
	}

	for _, dd := range p.Asks {
		ask, err := parseSnapshotBookEntry(Sell, dd)
		if err != nil {
			return SnapshotMessage{}, err
		}
		msg.Asks = append(msg.Asks, ask)
	}

	for _, dd := range p.Bids {
		bid, err := parseSnapshotBookEntry(Buy, dd)
		if err != nil {
			return SnapshotMessage{}, err
		}
		msg.Bids = append(msg.Bids, bid)
	}

	return msg, nil
}

func parseSnapshotBookEntry(side Side, data []string) (*bookEntry, error) {
	price, err := MoneyFromString(data[0])
	if err != nil {
		return nil, fmt.Errorf("error parsing price: %v\ndata: %v", err, data)
	}

	size, err := strconv.ParseFloat(data[1], 64)
	if err != nil {
		return nil, err
	}

	return &bookEntry{
		Side:  side,
		Price: price,
		Size:  size,
	}, nil
}

var lastL2 time.Time

func parseL2Update(bb []byte) (L2UpdateMessage, error) {
	p := new(struct {
		Type      string    `json:"type"`
		ProductID ProductID `json:"product_id"`
		Time      time.Time `json:"time"`
		Changes   [][]string
	})
	if err := json.Unmarshal(bb, &p); err != nil {
		return L2UpdateMessage{}, err
	}

	fmt.Println(p.Time)
	if p.Time.Before(lastL2) {
		panic("out of order")
	}
	lastL2 = p.Time

	msg := L2UpdateMessage{
		ProductID: p.ProductID,
		Bids:      make([]*bookEntry, 0, 1024),
		Asks:      make([]*bookEntry, 0, 1024),
	}

	for _, c := range p.Changes {
		entry, err := parseL2BookEntry(c)
		if err != nil {
			return L2UpdateMessage{}, err
		}
		switch entry.Side {
		case Buy:
			msg.Bids = append(msg.Bids, entry)
		case Sell:
			msg.Asks = append(msg.Asks, entry)
		default:
			return L2UpdateMessage{}, fmt.Errorf("invalid side: %s", entry.Side)
		}
	}

	return msg, nil
}

func parseL2BookEntry(data []string) (*bookEntry, error) {
	side, err := parseSide(data[0])
	if err != nil {
		return nil, err
	}
	price, err := MoneyFromString(data[1])
	if err != nil {
		return nil, err
	}
	size, err := strconv.ParseFloat(data[2], 64)
	if err != nil {
		return nil, err
	}

	return &bookEntry{
		Side:  side,
		Price: price,
		Size:  size,
	}, nil
}

type TickerMessage struct {
	Message

	Seq       int
	ProductID ProductID
	BestBid   Money
	BestAsk   Money
	Spread    Money
}

type CancelMessage struct {
	Message

	OrderID OrderID
}

type FilledMessage struct {
	Message

	OrderID OrderID
	Side    Side
}

type MatchMessage struct {
	Message

	Side    Side
	OrderID OrderID
	Size    float64
}

func parseTicker(bb []byte) (Message, error) {
	p := new(struct {
		Type      MessageType `json:"type"`
		Sequence  int         `json:"sequence"`
		ProductID ProductID   `json:"product_id"`
		Price     string      `json:"price"`
		Open24H   string      `json:"open_24h"`
		Volume24H string      `json:"volume_24h"`
		Low24H    string      `json:"low_24h"`
		High24H   string      `json:"high_24h"`
		Volume30D string      `json:"volume_30d"`
		BestBid   Money       `json:"best_bid"`
		BestAsk   Money       `json:"best_ask"`
	})
	if err := json.Unmarshal(bb, p); err != nil {
		return nil, err
	}
	return TickerMessage{
		ProductID: p.ProductID,
		BestBid:   p.BestBid,
		BestAsk:   p.BestAsk,
		Spread:    p.BestAsk.Minus(p.BestBid),
	}, nil
}

func parseMatch(bb []byte) (Message, error) {
	p := new(struct {
		Type         string  `json:"type"`
		Side         Side    `json:"side"`
		MakerOrderID OrderID `json:"maker_order_id"`
		Size         string  `json:"size"`
		// unused below this line
		TradeID        int       `json:"trade_id"`
		TakerOrderID   string    `json:"taker_order_id"`
		Price          string    `json:"price"`
		ProductID      string    `json:"product_id"`
		MakerUserID    string    `json:"maker_user_id"`
		UserID         string    `json:"user_id"`
		MakerProfileID string    `json:"maker_profile_id"`
		ProfileID      string    `json:"profile_id"`
		Sequence       int       `json:"sequence"`
		Time           time.Time `json:"time"`
	})

	if err := json.Unmarshal(bb, p); err != nil {
		return nil, err
	}

	return MatchMessage{
		Side:    p.Side,
		OrderID: p.MakerOrderID,
	}, nil
}

func parseDone(bb []byte) (Message, error) {
	p := new(struct {
		Type          string    `json:"type"`
		Side          Side      `json:"side"`
		OrderID       OrderID   `json:"order_id"`
		Reason        string    `json:"reason"`
		ProductID     string    `json:"product_id"`
		Price         string    `json:"price"`
		RemainingSize string    `json:"remaining_size"`
		Sequence      int       `json:"sequence"`
		UserID        string    `json:"user_id"`
		ProfileID     string    `json:"profile_id"`
		Time          time.Time `json:"time"`
	})
	if err := json.Unmarshal(bb, p); err != nil {
		return nil, err
	}

	switch p.Reason {
	case "canceled":
		return CancelMessage{OrderID: p.OrderID}, nil
	case "filled":
		return FilledMessage{OrderID: p.OrderID, Side: p.Side}, nil
	default:
		return nil, fmt.Errorf("unknown reason: %v", p.Reason)
	}
}

func parseHeartbeat(bb []byte) (HeartbeatMessage, error) {
	return HeartbeatMessage{}, nil
}

func parseSubscriptions(bb []byte) (SubscriptionsMessage, error) {
	return SubscriptionsMessage{}, nil
}

func parseSide(s string) (Side, error) {
	switch s {
	case "buy":
		return Buy, nil
	case "sell":
		return Sell, nil
	default:
		return "", fmt.Errorf("invalid side: %s", s)
	}
}
