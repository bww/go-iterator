package siter

import (
	"context"
	"sync"
)

// A channel-backed implementation; this is suitable for normal, streaming or
// unbounded applications.
type channelIter[T any] struct {
	cxt      context.Context
	meta     Meta
	res      chan Result[T]
	done     chan struct{}
	canceler sync.Once
}

func New[T any](res chan Result[T]) Writer[T] {
	return NewWithContext(context.Background(), res)
}

func NewWithContext[T any](cxt context.Context, res chan Result[T]) Writer[T] {
	return NewWithMeta(cxt, res, Meta{})
}

func NewWithMeta[T any](cxt context.Context, res chan Result[T], meta Meta) Writer[T] {
	return &channelIter[T]{
		cxt:  cxt,
		res:  res,
		done: make(chan struct{}),
		meta: meta,
	}
}

func (t *channelIter[T]) Meta() Meta {
	return t.meta
}

func (t *channelIter[T]) Next() (T, error) {
	var zero T
	select {
	case <-t.cxt.Done():
		return zero, ErrCanceled
	default:
		v, ok := <-t.res
		if ok {
			return v.Elem, v.Error
		} else {
			return zero, ErrClosed
		}
	}
}

func (t *channelIter[T]) Write(res T) error {
	select {
	case <-t.done:
		return ErrClosed
	case <-t.cxt.Done():
		return ErrCanceled
	default:
		t.res <- Result[T]{Elem: res}
		return nil
	}
}

func (t *channelIter[T]) Close() {
	select {
	case <-t.done:
		return // already finalized
	default:
		t.canceler.Do(func() {
			close(t.done)
			close(t.res)
		})
	}
}

func (t *channelIter[T]) Cancel(err error) error {
	if err != nil {
		select {
		case <-t.done:
			return ErrClosed
		case <-t.cxt.Done():
			return ErrCanceled
		default:
			t.res <- Result[T]{Error: err}
		}
	}
	t.Close()
	return err
}
