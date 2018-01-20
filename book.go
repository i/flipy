package main

import "container/heap"

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
	if e.Side == Sell {
		heap.Push(b.asks, e)
	} else {
		heap.Push(b.bids, e)
	}
}

type bookEntry struct {
	Side  Side
	Price Money
	Size  float64
}

func (b bookEntry) priority() int64 {
	if b.Side == Sell {
		return b.Price.Int64()
	}
	return -b.Price.Int64()
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
