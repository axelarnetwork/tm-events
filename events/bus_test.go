package events_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/stretchr/testify/assert"

	abci "github.com/cometbft/cometbft/abci/types"

	testutils "github.com/axelarnetwork/utils/test"

	"github.com/axelarnetwork/tm-events/events"
	"github.com/axelarnetwork/tm-events/events/mock"
	"github.com/axelarnetwork/tm-events/pubsub"
	"github.com/axelarnetwork/utils/slices"
	"github.com/axelarnetwork/utils/test/rand"
)

func TestBus_FetchEvents(t *testing.T) {
	t.Run("WHEN the event source throws an error THEN the bus returns error", func(t *testing.T) {
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
		bus := events.NewEventBus(source, pubsub.NewBus[events.ABCIEventWithHeight]())

		errChan := bus.FetchEvents(context.Background())

		errors <- fmt.Errorf("some error")

		err := <-errChan
		assert.Error(t, err)
	})

	t.Run("WHEN the block source block result channel closes THEN the bus shuts down", func(t *testing.T) {
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
		bus := events.NewEventBus(source, pubsub.NewBus[events.ABCIEventWithHeight]())

		bus.FetchEvents(context.Background())

		close(results)

		timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		newBlocks chan *coretypes.ResultBlockResults
	)

	setup := func() {
		newBlocks = make(chan *coretypes.ResultBlockResults, 10000)
		source := &mock.BlockSourceMock{BlockResultsFunc: func(ctx context.Context) (<-chan *coretypes.ResultBlockResults, <-chan error) {
			return newBlocks, nil
		}}

		bus = events.NewEventBus(source, pubsub.NewBus[events.ABCIEventWithHeight]())
	}

	repeats := 20
	t.Run("WHEN subscribing to block events THEN event height matches block height", testutils.Func(func(t *testing.T) {
		setup()

		bus.FetchEvents(context.Background())
		sub := bus.Subscribe(func(events.ABCIEventWithHeight) bool { return true })

		beginBlockEvents := slices.Map(randomEvents(rand.I64Between(0, 10)), func(e abci.Event) abci.Event {
			e.Attributes = append(e.Attributes, abci.EventAttribute{Key: "mode", Value: "BeginBlock"})
			return e
		})
		endBlockEvents := slices.Map(randomEvents(rand.I64Between(0, 10)), func(e abci.Event) abci.Event {
			e.Attributes = append(e.Attributes, abci.EventAttribute{Key: "mode", Value: "EndBlock"})
			return e
		})

		newBlock := &coretypes.ResultBlockResults{
			Height:              rand.PosI64(),
			FinalizeBlockEvents: append(beginBlockEvents, endBlockEvents...),
			TxsResults:          randomTxResults(rand.I64Between(1, 10)),
		}

		endMarkerBlock := &coretypes.ResultBlockResults{
			Height:              0,
			FinalizeBlockEvents: randomEvents(rand.I64Between(3, 10)),
			TxsResults:          randomTxResults(rand.I64Between(1, 10)),
		}
		newBlocks <- newBlock
		newBlocks <- endMarkerBlock

		timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		expectedEventCount := len(newBlock.FinalizeBlockEvents)
		for _, result := range newBlock.TxsResults {
			expectedEventCount += len(result.Events)
		}
		var eventCount int
		for {
			select {
			case <-timeout.Done():
				assert.FailNow(t, "timed out")
			case event := <-sub:
				actualHeight := event.Height
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
