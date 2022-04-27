package events

import (
	"context"

	"github.com/tendermint/tendermint/libs/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/axelarnetwork/tm-events/pubsub"
)

// Bus represents an object that receives blocks from a tendermint server and manages queries for events in those blocks
type Bus struct {
	source BlockSource
	logger log.Logger
	done   chan struct{}
	bus    pubsub.Bus[ABCIEventWithHeight]
}

// NewEventBus returns a new event bus instance
func NewEventBus(source BlockSource, bus pubsub.Bus[ABCIEventWithHeight], logger log.Logger) *Bus {
	return &Bus{
		bus:    bus,
		source: source,
		logger: logger.With("publisher", "events"),
		done:   make(chan struct{}),
	}
}

// FetchEvents asynchronously queries the blockchain for new blocks and publishes all txs events in those blocks to the event manager's subscribers.
// Any occurring errors are pushed into the returned error channel.
func (b *Bus) FetchEvents(ctx context.Context) <-chan error {
	// either the block source or the event manager could push an error at the same time, so we need to make sure we don't block
	errChan := make(chan error, 2)

	ctx, shutdown := context.WithCancel(ctx)
	blockResults, blockErrs := b.source.BlockResults(ctx)

	go func() {
		defer b.logger.Info("shutting down")

		for {
			select {
			case block, ok := <-blockResults:
				if !ok {
					shutdown()
				} else if err := b.publish(block); err != nil {
					errChan <- err
					shutdown()
				}
			case err := <-blockErrs:
				errChan <- err
				shutdown()
			case <-ctx.Done():
				b.logger.Info("closing all subscriptions")
				b.bus.Close()

				<-b.bus.Done()
				<-b.source.Done()
				close(b.done)
				return
			}
		}
	}()

	return errChan
}

// Subscribe returns an event subscription based on the given query
func (b *Bus) Subscribe(predicate func(ABCIEventWithHeight) bool) <-chan ABCIEventWithHeight {
	return b.bus.Subscribe(predicate)
}

// Done returns a channel that gets closed when the Bus is done cleaning up
func (b *Bus) Done() <-chan struct{} {
	return b.done
}

func (b *Bus) publish(block *coretypes.ResultBlockResults) error {
	// beginBlock and endBlock events are published together as block events
	blockEvents := append(block.BeginBlockEvents, block.EndBlockEvents...)
	for _, event := range blockEvents {
		err := b.bus.Publish(ABCIEventWithHeight{
			Height: block.Height,
			Event:  event,
		})
		if err != nil {
			return err
		}
	}

	for _, txRes := range block.TxsResults {
		for _, event := range txRes.Events {
			err := b.bus.Publish(ABCIEventWithHeight{
				Height: block.Height,
				Event:  event,
			})
			if err != nil {
				return err
			}
		}
	}

	b.logger.Debug("published all events for block", "block_height", block.Height)
	return nil
}
