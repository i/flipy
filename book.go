package main

import (
	"container/heap"
	"errors"
	"fmt"
	"log"
)

/*
sell 5.30
sell 5.22
sell 5.20 <- min sell/ask
=========
buy  5.14 <- max buy/bid
buy  5.10
buy  5.00
*/
type Book struct {
	asks *heapMap
	bids *heapMap
}

type tinybook struct {
	maxBuy  Money
	minSell Money
}

func NewBook() (*Book, error) {
	return &Book{
		asks: newHeapMap(),
		bids: newHeapMap(),
	}, nil
}

func newHeapMap() *heapMap {
	hm := &heapMap{
		h: make([]*bookEntry, 0, 100),
		m: make(map[Money]*bookEntry, 100),
	}
	heap.Init(hm)
	return hm
}

func (b *Book) Dump() []*bookEntry {
	var ret []*bookEntry

	for b.asks.Len() > 0 {
		ret = append(ret, heap.Pop(b.asks).(*bookEntry))
	}
	return ret
}

func (b *Book) Size() int {
	return b.bids.Len() + b.asks.Len()
}

var MaxPrice = NewMoney(10000, 0)

func validate(e *bookEntry) error {
	if e.Price.GT(MaxPrice) {
		return errors.New("price too high")
	}
	return nil
}

func checkDouble(b *Book, e *bookEntry) {
	_, buyOK := b.bids.m[e.Price]
	_, sellOK := b.asks.m[e.Price]

	if buyOK && sellOK {
		log.Fatalf("fuck: %v", e.Price)
	}
}

func (b *Book) update(e *bookEntry) {
	if err := validate(e); err != nil {
		return
	}

	checkDouble(b, e)

	var hm *heapMap
	switch e.Side {
	case Buy:
		hm = b.bids
	case Sell:
		hm = b.asks
	default:
		log.Fatalf("invalid side: %v", e.Side)
	}
	hm.update(e)
}

func (b *Book) Spread() (Money, error) {
	ask, ok := b.asks.peek()
	if !ok {
		return Money{}, fmt.Errorf("can't calculate spread without any asks")
	}

	bid, ok := b.bids.peek()
	if !ok {
		return Money{}, fmt.Errorf("can't calculate spread without any bids")
	}

	//fmt.Printf("spread ask price/size: %v / %f\n", ask.Price, ask.Size)
	//fmt.Printf("spread bid price/size: %v / %f\n", bid.Price, bid.Size)
	return ask.Price.Minus(bid.Price), nil
}

type bookEntry struct {
	Side  Side
	Price Money
	Size  float64

	// position in heap array. -1 indicates it should not exist
	index int
}

// low sells > high sells
// high buys > low buys
func (e bookEntry) priority() int64 {
	if e.Side == Sell {
		return e.Price.Int64()
	} else if e.Side == Buy {
		return -e.Price.Int64()
	} else {
		panic(fmt.Sprintf("invalid side", e.Side))
	}
}

type heapMap struct {
	h []*bookEntry
	m map[Money]*bookEntry
}

func (hm heapMap) Len() int {
	if len(hm.h) != len(hm.m) {
		panic("lens don't match")
	}
	return len(hm.h)
}
func (hm heapMap) Less(i, j int) bool { return hm.h[i].priority() < hm.h[j].priority() }
func (hm *heapMap) Swap(i, j int) {
	hm.h[i], hm.h[j] = hm.h[j], hm.h[i]
	hm.h[i].index = i
	hm.h[j].index = j
}

func (hm *heapMap) peek() (*bookEntry, bool) {
	if len(hm.h) > 0 {
		if hm.h[0].Size == 0 {
			panic("size 0 found")
		}
		return hm.h[0], true
	}
	return nil, false
}

func (hm *heapMap) update(entry *bookEntry) {
	e, ok := hm.m[entry.Price]

	if entry.Size == 0 {
		if !ok {
			panic("removing invalid item")
		}
		heap.Remove(hm, e.index)
		delete(hm.m, e.Price)
		e.index = -1
		return
	}

	if ok {
		e.Size = entry.Size
	} else {
		heap.Push(hm, entry)
	}
}

func (hm *heapMap) Push(x interface{}) {
	entry := x.(*bookEntry)
	if entry.Size == 0 {
		panic("inserting empty?")
	}
	entry.index = len(hm.h)
	if _, ok := hm.m[entry.Price]; ok {
		panic("inserting something that's already there")
	}
	fmt.Printf("inserting %v\n", entry)
	hm.m[entry.Price] = entry
	hm.h = append(hm.h, entry)
}

func (hm *heapMap) Pop() interface{} {
	fmt.Println("pop")
	old := hm.h
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	(*hm).h = old[0 : n-1]
	return item
}
