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

func NewFeed() (*Feed, error) {
	c, _, err := websocket.DefaultDialer.Dial(gdaxFeedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error dialing feed: %v", err)
	}

	feed := &Feed{conn: c}
	if err := feed.Subscribe([]ProductID{BchUsd}); err != nil {
		return nil, err
	}
	return feed, nil
}

// TODO
type channelSpec interface{}

func (f *Feed) Subscribe(productIDs []ProductID) error {
	payload, err := json.Marshal(struct {
		Type       string        `json:"type"`
		ProductIDs []ProductID   `json:"product_ids"`
		Channels   []channelSpec `json:"channels"`
	}{
		Type:       "subscribe",
		ProductIDs: productIDs,
		Channels: []channelSpec{
			"heartbeat",
			"level2",
		},
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
	default:
		return nil, fmt.Errorf("unknown message type: %v", mt)
	}
}

type SnapshotMessage struct {
	ProductID ProductID
	Bids      []bookEntry
	Asks      []bookEntry
}

func (s SnapshotMessage) MessageType() MessageType {
	return Snapshot
}

// dry this later
type L2UpdateMessage struct {
	ProductID ProductID
	Bids      []bookEntry
	Asks      []bookEntry
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
		Asks:      make([]bookEntry, 0, len(p.Asks)),
		Bids:      make([]bookEntry, 0, len(p.Bids)),
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

func parseSnapshotBookEntry(side Side, data []string) (bookEntry, error) {
	price, err := MoneyFromString(data[0])
	if err != nil {
		return bookEntry{}, fmt.Errorf("error parsing price: %v\ndata: %v", err, data)
	}

	size, err := strconv.ParseFloat(data[1], 64)
	if err != nil {
		return bookEntry{}, err
	}

	return bookEntry{
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
		Bids:      make([]bookEntry, 0, 1),
		Asks:      make([]bookEntry, 0, 1),
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

func parseL2BookEntry(data []string) (bookEntry, error) {
	side, err := parseSide(data[0])
	if err != nil {
		return bookEntry{}, err
	}
	price, err := MoneyFromString(data[1])
	if err != nil {
		return bookEntry{}, err
	}
	size, err := strconv.ParseFloat(data[2], 64)
	if err != nil {
		return bookEntry{}, err
	}

	return bookEntry{
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
