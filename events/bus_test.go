package events_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/axelarnetwork/tm-events/events"
	"github.com/axelarnetwork/tm-events/events/mock"
	"github.com/axelarnetwork/tm-events/pubsub"
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
		bus := events.NewEventBus(source, pubsub.NewBus[abci.Event](), log.TestingLogger())

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
		bus := events.NewEventBus(source, pubsub.NewBus[abci.Event](), log.TestingLogger())

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
