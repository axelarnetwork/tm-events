package pubsub

import (
	"context"
	"errors"
	"sync"

	"github.com/smallnest/chanx"
)

// ErrNotRunning is the error with message "not running"
var ErrNotRunning = errors.New("not running")

// Bus provides a general purpose interface for message passing
type Bus[T any] interface {
	Publish(T) error
	Subscribe(filter func(T) bool) <-chan T
	Close()
	Done() <-chan struct{}
}

type bus[T any] struct {
	subMutex      *sync.Mutex
	subscriptions []*subscriber[T]
	buffer        *chanx.UnboundedChan[T]
	bufferCap     int // capacity of the in and out channels of buffer
	running       context.Context
	closing       context.CancelFunc
	cleanedUp     chan struct{}
	once          *sync.Once
}

// NewBus runs a new bus and returns bus details
func NewBus[T any]() Bus[T] {
	running, closing := context.WithCancel(context.Background())
	bufferCap := 1000
	b := &bus[T]{
		bufferCap: bufferCap,
		buffer:    chanx.NewUnboundedChan[T](bufferCap),
		running:   running,
		closing:   closing,
		subMutex:  &sync.Mutex{},
		cleanedUp: make(chan struct{}),
		once:      &sync.Once{},
	}

	go b.run()

	return b
}

func (b *bus[T]) Publish(item T) error {
	select {
	case <-b.running.Done():
		return ErrNotRunning
	default:
		b.buffer.In <- item
		return nil
	}
}

func (b *bus[T]) Subscribe(filter func(T) bool) <-chan T {
	// the mutex needs to be locked before the select, otherwise it could happen that we wait for the lock in the default case
	// and in the meantime the running context gets cancelled. This would lead to unpredictable interplay between Subscribe() and run()
	b.subMutex.Lock()
	defer b.subMutex.Unlock()

	select {
	case <-b.running.Done():
		ch := make(chan T)
		close(ch)
		return ch
	default:
		subscription := &subscriber[T]{
			filter: filter,
			buffer: chanx.NewUnboundedChan[T](b.bufferCap),
		}
		b.subscriptions = append(b.subscriptions, subscription)

		return subscription.buffer.Out
	}
}

func (b *bus[T]) Close() {
	b.once.Do(func() {
		b.closing()
		close(b.buffer.In)
	})
}

func (b *bus[T]) Done() <-chan struct{} {
	return b.cleanedUp
}

func (b *bus[T]) run() {
	for item := range b.buffer.Out {
		b.subMutex.Lock()
		for _, sub := range b.subscriptions {
			if sub.filter(item) {
				sub.buffer.In <- item
			}
		}
		b.subMutex.Unlock()
	}

	b.subMutex.Lock()
	for _, sub := range b.subscriptions {
		close(sub.buffer.In)
	}
	b.subMutex.Unlock()

	close(b.cleanedUp)
}

type subscriber[T any] struct {
	buffer *chanx.UnboundedChan[T]
	filter func(T) bool
}
