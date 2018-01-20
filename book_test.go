package main

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ = require.Equal

var entries = []bookEntry{
	bookEntry{
		Side:  Sell,
		Price: Money{dollars: 5, cents: 0},
	},
}

func mkEntry(side Side, dollars int64) bookEntry {
	return bookEntry{
		Side:  side,
		Price: Money{dollars, 0},
	}
}

func TestBookHeap(t *testing.T) {
	for _, tt := range []struct {
		desc   string
		input  []int64
		output []int64
		side   Side
	}{
		{
			desc:   "single entry",
			input:  []int64{1},
			output: []int64{1},
			side:   Sell,
		},
		{
			desc:   "two entries in order",
			input:  []int64{1, 2},
			output: []int64{1, 2},
			side:   Sell,
		},
		{
			desc:   "two entries out of order",
			input:  []int64{2, 1},
			output: []int64{1, 2},
			side:   Sell,
		},
		{
			desc:   "three entries in order",
			input:  []int64{1, 2, 3},
			output: []int64{1, 2, 3},
			side:   Sell,
		},
		{
			desc:   "three entries shuffled",
			input:  []int64{3, 1, 2},
			output: []int64{1, 2, 3},
			side:   Sell,
		},
		{
			desc:   "three entries shuffled buy",
			input:  []int64{3, 1, 2},
			output: []int64{3, 2, 1},
			side:   Buy,
		},
	} {
		h := &bookEntryHeap{}
		heap.Init(h)

		for _, i := range tt.input {
			heap.Push(h, mkEntry(Sell, i))
		}

		for _, i := range tt.output {
			assert.Equal(t, heap.Pop(h), mkEntry(Sell, i), tt.desc)
		}

	}
}
