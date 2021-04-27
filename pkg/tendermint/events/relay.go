package events

import (
	"context"
	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	"golang.org/x/sync/errgroup"
)

type SinkFunc = func(event pubsub.Event) error

type Relay struct {
	pubsub.Bus
	source Source

	ctx    context.Context
	cancel context.CancelFunc
	eg     *errgroup.Group
}

func NewRelay(ctx context.Context, source Source) Relay {
	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)

	return Relay{
		Bus:    pubsub.NewBus(),
		source: source,
		ctx:    ctx,
		cancel: cancel,
		eg:     eg,
	}
}

func (r Relay) Start() {
	r.source.Start(r.ctx, r.eg, r.Publish)
}

func (r Relay) Close() {
	// Clean up go routines
	r.source.Close()
	r.Bus.Close()
	r.cancel()
}

func (r Relay) ErrGroup() *errgroup.Group {
	return r.eg
}

func (r Relay) CatchAsync(cb func(error)) {
	go func() {
		err := r.eg.Wait()
		if err != nil {
			cb(err)
		}
	}()
}
