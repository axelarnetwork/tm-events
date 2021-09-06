package events

import (
	"context"
	"fmt"
	"sync"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tm "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/tm-events/pubsub"
)

// Bus represents an object that receives blocks from a tendermint server and manages queries for events in those blocks
type Bus struct {
	subscribeLock sync.RWMutex

	source BlockSource

	subscriptions map[string]struct {
		tmpubsub.Query
		pubsub.Bus
	}
	createBus func() pubsub.Bus
	logger    log.Logger

	done chan struct{}
}

// NewEventBus returns a new event bus instance
func NewEventBus(source BlockSource, pubsubFactory func() pubsub.Bus, logger log.Logger) *Bus {
	bus := &Bus{
		subscribeLock: sync.RWMutex{},
		source:        source,
		subscriptions: make(map[string]struct {
			tmpubsub.Query
			pubsub.Bus
		}),
		createBus: pubsubFactory,
		logger:    logger.With("publisher", "events"),
		done:      make(chan struct{}),
	}

	return bus
}

// FetchEvents asynchronously queries the blockchain for new blocks and publishes all txs events in those blocks to the event manager's subscribers.
// Any occurring errors are pushed into the returned error channel.
func (m *Bus) FetchEvents(ctx context.Context) <-chan error {
	// either the block source or the event manager could push an error at the same time, so we need to make sure we don't block
	errChan := make(chan error, 2)

	ctx, shutdown := context.WithCancel(ctx)
	blockResults, blockErrs := m.source.BlockResults(ctx)

	go func() {
		defer m.logger.Info("shutting down")

		for {
			select {
			case block, ok := <-blockResults:
				if !ok {
					shutdown()
				} else if err := m.publishEvents(block); err != nil {
					errChan <- err
					shutdown()
				}
			case err := <-blockErrs:
				errChan <- err
				shutdown()
			case <-ctx.Done():
				m.logger.Info("closing all subscriptions")

				m.subscribeLock.Lock()
				for _, sub := range m.subscriptions {
					sub.Close()
				}
				m.subscribeLock.Unlock()
				<-m.source.Done()
				close(m.done)
				return
			}
		}
	}()

	return errChan
}

// Subscribe returns an event subscription based on the given query
func (m *Bus) Subscribe(q tmpubsub.Query) (pubsub.Subscriber, error) {
	// map cannot deal with concurrent read/writes so we lock for the whole function.
	// Alternatively we would have to acquire a read lock first and then replace it with a write lock if the value doesn't exist.
	// We chose the simpler solution here.
	m.subscribeLock.Lock()
	defer m.subscribeLock.Unlock()

	subscription, ok := m.subscriptions[q.String()]
	if !ok {
		subscription = struct {
			tmpubsub.Query
			pubsub.Bus
		}{Query: q, Bus: m.createBus()}
		m.subscriptions[q.String()] = subscription
	}

	return subscription.Subscribe()
}

// Done returns a channel that gets closed when the Bus is done cleaning up
func (m *Bus) Done() <-chan struct{} {
	return m.done
}

func (m *Bus) publishEvents(block *coretypes.ResultBlockResults) error {
	// Publishing events and adding subscriptions are mutually exclusive operations.
	// This guarantees that a subscription sees all block events or none
	m.subscribeLock.RLock()
	defer m.subscribeLock.RUnlock()

	// beginBlock and endBlock events are published together as block events
	blockEvents := append(block.BeginBlockEvents, block.EndBlockEvents...)
	eventMap := Flatten(blockEvents)
	eventMap[tm.EventTypeKey] = append(eventMap[tm.EventTypeKey], tm.EventNewBlockHeader, tm.EventNewBlock)
	err := m.publishMatches(blockEvents, eventMap, block.Height)
	if err != nil {
		return err
	}

	for _, txRes := range block.TxsResults {
		eventMap = Flatten(txRes.Events)
		eventMap[tm.EventTypeKey] = append(eventMap[tm.EventTypeKey], tm.EventTx)
		err := m.publishMatches(txRes.Events, eventMap, block.Height)
		if err != nil {
			return err
		}
	}

	m.logger.Debug(fmt.Sprintf("published all events for block %d", block.Height))
	return nil
}

func (m *Bus) publishMatches(abciEvents []abci.Event, eventMap map[string][]string, blockHeight int64) error {
	for _, subscription := range m.subscriptions {
		match, err := subscription.Query.Matches(eventMap)
		if err != nil {
			return fmt.Errorf("failed to match against query %s: %w", subscription.Query.String(), err)
		}

		if !match {
			continue
		}

		for _, abciEvent := range abciEvents {
			event, err := Parse(abciEvent)
			if err != nil {
				return sdkerrors.Wrap(err, fmt.Sprintf("could not parse event %v", abciEvent))
			}
			event.Height = blockHeight

			if err := subscription.Publish(event); err != nil {
				return err
			}
		}
	}
	return nil
}