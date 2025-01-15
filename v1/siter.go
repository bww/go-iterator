// Pronounced "sitter", short for "streaming iterator"
package siter

import (
	"context"
	"errors"
)

var (
	ErrClosed   = errors.New("Iterator closed")
	ErrCanceled = errors.New("Iterator canceled")
)

type Meta struct {
	Total int
}

type Result[T any] struct {
	Elem  T
	Error error
}

type Iterator[T any] interface {
	Meta() Meta
	Next() (T, error)
	Close()
}

type Writer[T any] interface {
	Iterator[T]
	Write(T) error
	Cancel(error) error
}

func IsFinished(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, ErrClosed) || errors.Is(err, ErrCanceled)
}

type Visitor[T any] interface {
	Visit(T) error
}

type VisitorFunc[T any] func(T) error

func (f VisitorFunc[T]) Visit(v T) error {
	return f(v)
}

// Visit applies the provided function to every element in the iterator. The
// iterator is closed before returning if it is not fully consumed.
func Visit[T any](iter Iterator[T], visitor Visitor[T]) error {
	for {
		v, err := iter.Next()
		if IsFinished(err) {
			break // iterator is consumed or canceled
		} else if err != nil {
			return err
		}
		err = visitor.Visit(v)
		if err != nil {
			iter.Close()
			return err
		}
	}
	return nil
}

// Collect processes the entire iterator and appends each element produced to a
// slice, which is returned.
func Collect[T any](iter Iterator[T]) ([]T, error) {
	return CollectN(iter, -1)
}

// CollectN processes the iterator to the end or [limit] elements, whichever
// comes first, and appends each element produced to a slice, which is returned.
// If the limit is <= 0 no limit is imposed. The iterator is closed before
// returning if it is not fully consumed.
func CollectN[T any](iter Iterator[T], limit int) ([]T, error) {
	var res []T
	for i := 0; limit <= 0 || i < limit; i++ {
		e, err := iter.Next()
		if IsFinished(err) {
			break // iterator closed or context canceled; this is not unusual
		} else if err != nil {
			return res, err
		}
		res = append(res, e)
	}
	iter.Close()
	return res, nil
}

// CollectErr has the same functionality as Collect, except that it accepts an
// error as its second argument. If the error is non-nil it is simply returned
// instead of processing the iterator. This is intended to support a use cases
// where we pass the result of a function that creates an iterator into this
// collection function:
//
//	    func loadData() (Iterator[T], error) {}
//
//	    res, err := CollectErr(loadData())
//			 if err != nil {
//	      // ...
//			 }
func CollectErr[T any](iter Iterator[T], err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	return Collect(iter)
}
