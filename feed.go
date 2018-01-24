package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

type AuthInfo struct {
	Key        string
	Passphrase string
	Secret     string
}

func (a AuthInfo) signature() (time.Time, string) {
	ts := time.Now()
	what := []byte(strconv.FormatInt(ts.Unix(), 10) + "GET" + "/users/self/verify")
	secret, err := base64.StdEncoding.DecodeString(a.Secret)
	if err != nil {
		panic(err)
	}
	h := hmac.New(sha256.New, secret)
	h.Write(what)
	return ts, base64.StdEncoding.EncodeToString(h.Sum(nil))
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

	ts, sig := f.auth.signature()

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
			"level2",
			"full",
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
	case Error:
		return nil, fmt.Errorf("received error: %v", string(bb))
	default:
		return nil, fmt.Errorf("unknown message type: %v", mt)
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

	msg := L2UpdateMessage{
		ProductID: p.ProductID,
		Bids:      make([]*bookEntry, 0, 1),
		Asks:      make([]*bookEntry, 0, 1),
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
