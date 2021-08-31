package events_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tm "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/utils/test"
	"github.com/axelarnetwork/utils/test/rand"

	"github.com/axelarnetwork/tm-events/events"
	"github.com/axelarnetwork/tm-events/events/mock"
	"github.com/axelarnetwork/tm-events/pubsub"
	pubsubMock "github.com/axelarnetwork/tm-events/pubsub/mock"
)

func TestBus_FetchEvents(t *testing.T) {

	t.Run("WHEN the event source throws an error THEN the bus returns error", func(t *testing.T) {
		busFactory := func() pubsub.Bus { return &pubsubMock.BusMock{} }
		errors := make(chan error, 1)
		source := &mock.BlockSourceMock{
			BlockResultsFunc: func(ctx context.Context) (<-chan *coretypes.ResultBlockResults, <-chan error) {
				return make(chan *coretypes.ResultBlockResults), errors
			},
			DoneFunc: func() <-chan struct{} {
				done := make(chan struct{})
				close(done)
				return done
			},
		}
		bus := events.NewEventBus(source, busFactory, log.TestingLogger())

		errChan := bus.FetchEvents(context.Background())

		errors <- fmt.Errorf("some error")

		err := <-errChan
		assert.Error(t, err)
	})

	t.Run("WHEN the block source block result channel closes THEN the bus shuts down", func(t *testing.T) {
		busMock := &pubsubMock.BusMock{
			SubscribeFunc: func() (pubsub.Subscriber, error) {
				return &pubsubMock.SubscriberMock{}, nil
			},
			CloseFunc: func() {},
		}
		busFactory := func() pubsub.Bus { return busMock }
		results := make(chan *coretypes.ResultBlockResults)
		source := &mock.BlockSourceMock{
			BlockResultsFunc: func(ctx context.Context) (<-chan *coretypes.ResultBlockResults, <-chan error) {
				return results, nil
			},
			DoneFunc: func() <-chan struct{} {
				done := make(chan struct{})
				close(done)
				return done
			},
		}
		bus := events.NewEventBus(source, busFactory, log.TestingLogger())

		bus.FetchEvents(context.Background())

		close(results)

		timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		select {
		case <-bus.Done():
			return
		case <-timeout.Done():
			assert.FailNow(t, "timed out")
		}
	})
}

func TestBus_Subscribe(t *testing.T) {
	var (
		bus       *events.Bus
		query     *mock.QueryMock
		newBlocks chan *coretypes.ResultBlockResults
	)

	setup := func() {
		newBlocks = make(chan *coretypes.ResultBlockResults, 10000)
		source := &mock.BlockSourceMock{BlockResultsFunc: func(ctx context.Context) (<-chan *coretypes.ResultBlockResults, <-chan error) {
			return newBlocks, nil
		}}

		actualEvents := make(chan pubsub.Event, 100000)
		busFactory := func() pubsub.Bus {
			return &pubsubMock.BusMock{
				PublishFunc: func(event pubsub.Event) error {
					actualEvents <- event
					return nil
				},
				SubscribeFunc: func() (pubsub.Subscriber, error) {
					return &pubsubMock.SubscriberMock{
						EventsFunc: func() <-chan pubsub.Event { return actualEvents },
					}, nil
				},
				CloseFunc: func() {
					close(actualEvents)
				},
			}
		}
		bus = events.NewEventBus(source, busFactory, log.TestingLogger())

		query = &mock.QueryMock{
			MatchesFunc: func(map[string][]string) (bool, error) { return true, nil },
			StringFunc:  func() string { return rand.StrBetween(1, 100) },
		}
	}

	repeats := 20
	t.Run("query block with txs", testutils.Func(func(t *testing.T) {
		setup()

		bus.FetchEvents(context.Background())
		sub, err := bus.Subscribe(query)
		assert.NoError(t, err)

		newBlock := &coretypes.ResultBlockResults{
			Height:           rand.PosI64(),
			BeginBlockEvents: randomEvents(rand.I64Between(0, 10)),
			TxsResults:       randomTxResults(rand.I64Between(1, 10)),
			EndBlockEvents:   randomEvents(rand.I64Between(0, 10)),
		}

		endMarkerBlock := &coretypes.ResultBlockResults{
			Height:           0,
			BeginBlockEvents: randomEvents(rand.I64Between(3, 10)),
			TxsResults:       randomTxResults(rand.I64Between(1, 10)),
			EndBlockEvents:   randomEvents(rand.I64Between(3, 10)),
		}
		newBlocks <- newBlock
		newBlocks <- endMarkerBlock

		timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		expectedEventCount := len(newBlock.BeginBlockEvents) + len(newBlock.EndBlockEvents)
		for _, result := range newBlock.TxsResults {
			expectedEventCount += len(result.Events)
		}
		var eventCount int
		for {
			select {
			case <-timeout.Done():
				assert.FailNow(t, "timed out")
			case event := <-sub.Events():
				assert.IsType(t, events.Event{}, event)
				actualHeight := event.(events.Event).Height
				if actualHeight == 0 {
					assert.Equal(t, expectedEventCount, eventCount)
					return
				}
				assert.Equal(t, newBlock.Height, actualHeight)
				eventCount++
			}
		}
	}).Repeat(repeats))

	t.Run("match tm.event=Tx", testutils.Func(func(t *testing.T) {
		setup()

		bus.FetchEvents(context.Background())
		query.MatchesFunc = func(events map[string][]string) (bool, error) {
			types := events[tm.EventTypeKey]

			for _, t := range types {
				if t == tm.EventTx {
					return true, nil
				}
			}
			return false, nil
		}
		sub, err := bus.Subscribe(query)
		assert.NoError(t, err)

		newBlock := &coretypes.ResultBlockResults{
			Height:           rand.PosI64(),
			BeginBlockEvents: randomEvents(rand.I64Between(0, 10)),
			TxsResults:       randomTxResults(rand.I64Between(1, 10)),
			EndBlockEvents:   randomEvents(rand.I64Between(0, 10)),
		}

		endMarkerBlock := &coretypes.ResultBlockResults{
			Height:           0,
			BeginBlockEvents: randomEvents(rand.I64Between(3, 10)),
			TxsResults:       randomTxResults(rand.I64Between(1, 10)),
			EndBlockEvents:   randomEvents(rand.I64Between(3, 10)),
		}
		newBlocks <- newBlock
		newBlocks <- endMarkerBlock

		timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		expectedEventCount := 0
		for _, result := range newBlock.TxsResults {
			expectedEventCount += len(result.Events)
		}
		var eventCount int
		for {
			select {
			case <-timeout.Done():
				assert.FailNow(t, "timed out")
			case event := <-sub.Events():
				assert.IsType(t, events.Event{}, event)
				actualHeight := event.(events.Event).Height
				if actualHeight == 0 {
					assert.Equal(t, expectedEventCount, eventCount)
					return
				}
				assert.Equal(t, newBlock.Height, actualHeight)
				eventCount++
			}
		}
	}).Repeat(repeats))
}
