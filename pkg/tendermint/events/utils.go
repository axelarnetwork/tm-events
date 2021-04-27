package events

import (
	"errors"
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
)

func WaitAction(bus *Hub, action string) (types.Event, error) {
	sub, err := bus.SubscribeMessage(action)
	if err != nil {
		return types.Event{}, err
	}

	for {
		select {
		case ev := <-sub.Events():
			if txEvent, ok := ev.(types.Event); ok {
				fmt.Printf("%+v\n", ev)
				return txEvent, nil
			}
		case <-sub.Done():
			return types.Event{}, fmt.Errorf("subscription closed before '%s' action was detected", action)
		}
	}
}

type EventFilterFunc func(event types.Event) bool
type WaitEventFunc func() (types.Event, error)
type NextEventFunc func(filterFunc EventFilterFunc) (types.Event, error)

var InvalidEventErr = errors.New("invalid event")

func WaitEventAsync(hub *Hub, query tmpubsub.Query) (WaitEventFunc, error) {
	// todo: test events are caught before WaitEventFunc is called
	sub, err := hub.Subscribe(query)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go consumeSubscriptionEvents(sub, evChan, errChan)

	return func() (types.Event, error) {
		defer sub.Close()
		fmt.Printf("> Waiting for event from query \"%s\"...\n", query)

		return <-evChan, <-errChan
	}, nil
}

func ProcessQuery(hub *Hub, query tmpubsub.Query, callback func(event types.Event)) (func() error, error) {
	sub, err := hub.Subscribe(query)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, 1)
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
