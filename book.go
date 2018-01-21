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
	askHeap *bookEntryHeap
	askMap  map[Money]*bookEntry

	bidHeap *bookEntryHeap
	bidMap  map[Money]*bookEntry
}

func NewBook() (*Book, error) {
	book := &Book{
		askHeap: &bookEntryHeap{},
		askMap:  make(map[Money]*bookEntry),
		bidHeap: &bookEntryHeap{},
		bidMap:  make(map[Money]*bookEntry),
	}
	heap.Init(book.askHeap)
	heap.Init(book.bidHeap)
	return book, nil
}

// removals can only happen when popping or peeking
func (b *Book) update(e *bookEntry) {
	var m map[Money]*bookEntry
	var h *bookEntryHeap

	if e.Side == Buy {
		m = b.bidMap
		h = b.bidHeap
	} else {
		m = b.askMap
		h = b.askHeap
	}

	if existing, ok := m[e.Price]; ok {
		existing.Size = e.Size
		return
	}

	m[e.Price] = e
	h.Push(e)
}

func (b *Book) Spread() (Money, error) {
	ask, ok := b.askHeap.Peek()
	if !ok {
		return Money{}, fmt.Errorf("todo")
	}
	bid, ok := b.bidHeap.Peek()
	if !ok {
		return Money{}, fmt.Errorf("todo")
	}
	return ask.Price.Minus(bid.Price), nil
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

type bookEntryHeap []*bookEntry

func (h bookEntryHeap) Len() int           { return len(h) }
func (h bookEntryHeap) Less(i, j int) bool { return h[i].priority() < h[j].priority() }
func (h bookEntryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *bookEntryHeap) Peek() (*bookEntry, bool) {
	for h.Len() > 0 {
		if (*h)[0].Size > 0 {
			return (*h)[0], true
		}
		h.Pop()
	}
	return nil, false
}

func (h *bookEntryHeap) Push(x interface{}) {
	entry := x.(*bookEntry)
	*h = append(*h, entry)
}

func (h *bookEntryHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
