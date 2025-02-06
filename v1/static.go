package siter

import (
	"context"
	"sync/atomic"
)

// A static, slice-backed implementation; this is suitable for situations where
// the entire input set is already in memory
type sliceIter[T any] struct {
	meta   Meta
	res    []T
	cursor int64
	closed int32
}

func NewWithSlice[T any](_ context.Context, t []T) Iterator[T] {
	return &sliceIter[T]{
		meta: Meta{Total: len(t)},
		res:  t,
	}
}

func (t *sliceIter[T]) Meta() Meta {
	return t.meta
}

func (t *sliceIter[T]) Next() (T, error) {
	var zero T
	if atomic.LoadInt32(&t.closed) > 0 {
		return zero, ErrClosed
	}
	n := atomic.AddInt64(&t.cursor, 1)
	if x := n - 1; x < int64(len(t.res)) {
		return t.res[x], nil
	} else {
		return zero, ErrClosed
	}
}

func (t *sliceIter[T]) Close() {
	if t == nil {
		return
	}
	atomic.AddInt32(&t.closed, 1)
}
