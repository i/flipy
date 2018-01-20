package main

import "container/heap"

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
	asks *bookEntryHeap
	bids *bookEntryHeap
}

func NewBook() (*Book, error) {
	book := &Book{
		asks: &bookEntryHeap{},
		bids: &bookEntryHeap{},
	}
	heap.Init(book.asks)
	heap.Init(book.bids)
	return book, nil
}

func (b *Book) update(e bookEntry) {
	if e.Side == Buy {
		heap.Push(b.bids, e)
	} else {
		heap.Push(b.asks, e)
	}
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
		return -e.Price.Int64()
	}
	return e.Price.Int64()
}

type bookEntryHeap []bookEntry

func (h bookEntryHeap) Len() int           { return len(h) }
func (h bookEntryHeap) Less(i, j int) bool { return h[i].priority() < h[j].priority() }
func (h bookEntryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h bookEntryHeap) Peek() bookEntry    { return h[len(h)-1] }

func (h *bookEntryHeap) Push(x interface{}) {
	entry := x.(bookEntry)
	*h = append(*h, entry)
}

func (h *bookEntryHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
