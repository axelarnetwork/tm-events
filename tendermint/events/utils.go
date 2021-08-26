package events

import (
	"errors"
	"fmt"

	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"

	"github.com/axelarnetwork/tm-events/pubsub"
	"github.com/axelarnetwork/tm-events/tendermint/types"

	"github.com/axelarnetwork/tm-events/events"
)

type EventPredicateFunc func(event types.Event) bool
type NextEventFunc func() (types.Event, error)
type FilteredNextEventFunc func(filter EventPredicateFunc) (types.Event, error)

var InvalidEventErr = errors.New("invalid event")

const LargeBuffSize = 32768

func ProcessQuery(eventBus *events.Bus, query tmpubsub.Query, callback func(event types.Event)) (func() error, error) {
	sub, err := eventBus.Subscribe(query)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, LargeBuffSize)
	errChan := make(chan error, 1)
	catchChan := make(chan error, 1)

	go consumeSubscriptionEvents(sub, evChan, errChan)

	go func() {
		for ev := range evChan {
			callback(ev)
		}
	}()

	go func() {
		err := <-errChan
		if err != nil {
			catchChan <- err
		}
	}()

	catch := func() error {
		err := <-catchChan
		fmt.Printf("Report events failed: %s", err)
		sub.Close()
		close(evChan)
		return err
	}

	return catch, nil
}

func consumeSubscriptionEvents(sub pubsub.Subscriber, evChan chan types.Event, errChan chan error) {
	for {
		select {
		case ev := <-sub.Events():
			if txEvent, ok := ev.(types.Event); ok {
				evChan <- txEvent
				errChan <- nil
			}
		case <-sub.Done():
			evChan <- types.Event{}
			errChan <- fmt.Errorf("subscription closed before event was detected")
			return
		}
	}

}
