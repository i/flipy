package main

import (
	"container/heap"
	"fmt"
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

	for {
		fmt.Println("ok...")
		if _, ok := b.asks.peek(); !ok {
			break
		}
		ret = append(ret, b.asks.Pop().(*bookEntry))
	}

	return ret
}

func (b *Book) Size() int {
	return b.bids.Len() + b.asks.Len()
}

// removals can only happen when popping or peeking
func (b *Book) update(e *bookEntry) {
	var hm *heapMap
	if e.Side == Buy {
		hm = b.bids
	} else {
		hm = b.asks
	}
	hm.Push(e)
}

// TODO needs fixing
func (b *Book) Spread() (Money, error) {
	ask, ok := b.asks.peek()
	if !ok {
		return Money{}, fmt.Errorf("todo")
	}
	bid, ok := b.bids.peek()
	if !ok {
		return Money{}, fmt.Errorf("todo")
	}
	return ask.Price.Minus(bid.Price), nil

	return Money{}, nil
}

type bookEntry struct {
	Side  Side
	Price Money
	Size  float64
}

// low sells > high sells
// high buys > low buys
func (e bookEntry) priority() int64 {
	if e.Side == Sell {
		return e.Price.Int64()
	}
	return -e.Price.Int64()
}

type heapMap struct {
	h []*bookEntry
	m map[Money]*bookEntry
}

func (hm heapMap) Len() int           { return len(hm.h) }
func (hm heapMap) Less(i, j int) bool { return hm.h[i].priority() < hm.h[j].priority() }
func (hm heapMap) Swap(i, j int)      { hm.h[i], hm.h[j] = hm.h[j], hm.h[i] }

func (hm *heapMap) peek() (*bookEntry, bool) {
	debug("peeking")
	for hm.Len() > 0 {
		if hm.h[0].Size == 0 {
			hm.Pop()
		} else {
			return (hm.h)[0], true
		}
	}
	return nil, false
}

func (hm *heapMap) Push(x interface{}) {
	entry := x.(*bookEntry)

	if el, ok := hm.m[entry.Price]; ok {
		el.Size = entry.Size
		return
	}

	hm.m[entry.Price] = entry
	hm.h = append(hm.h, entry)
}

// todo popping also needs to remove
func (h *heapMap) Pop() interface{} {
	debug("pop")
	for e := h.popImpl(); ; {
		debug(e)
		if e.Size == 0 {
			return e
		}
	}
}

func (hm *heapMap) popImpl() *bookEntry {
	debug("popimpl")
	old := hm.h
	n := len(old)
	x := old[n-1]
	hm.h = old[0 : n-1]
	return x
}
