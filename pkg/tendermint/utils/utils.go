package utils

import (
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/events"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
)

func WaitActionAsync(hub *events.Hub, eventType string, module string, action string) (events.WaitEventFunc, error) {
	sub, err := events.Subscribe(hub, eventType, module, action)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (types.Event, error) {
		fmt.Printf("Waiting for action '%s'... ", action)
		ev := <-evChan
		err := <-errChan
		sub.Close()

		return ev, err
	}, nil
}

func FilterActionAsync(hub *events.Hub, eventType string, module string, action string) (events.NextEventFunc, error) {
	sub, err := events.Subscribe(hub, eventType, module, action)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func(filter events.EventFilterFunc) (ev types.Event, err error) {
		fmt.Printf("Waiting for action '%s'... ", action)
		defer sub.Close()

		for {
			ev = <-evChan
			err = <-errChan
			if err != nil {
				return
			}

			if filter(ev) {
				return
			}
		}
	}, nil
}

func ConsumeFilteredSubscriptionEvents(sub events.FilteredSubscriber, evChan chan types.Event, errChan chan error) {
	for {
		select {
		case event := <-sub.Events():
			evChan <- event
			errChan <- nil
		case <-sub.Done():
			evChan <- types.Event{}
			errChan <- fmt.Errorf("subscription closed before event was detected")
			return
		}
	}
}

func NextActionAsync(hub *events.Hub, eventType string, module string, action string) (events.WaitEventFunc, events.FilteredSubscriber, error) {
	sub, err := events.Subscribe(hub, eventType, module, action)
	if err != nil {
		return nil, events.FilteredSubscriber{}, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (ev types.Event, err error) {
		fmt.Printf("Waiting for next action '%s'... ", action)

		return <-evChan, <-errChan
	}, sub, nil
}

func FilterNextAction(wait events.WaitEventFunc, sub events.FilteredSubscriber, filter events.EventFilterFunc) (types.Event, error) {
	for {
		ev, err := wait()
		if err != nil {
			return ev, err
		}

		if filter(ev) {
			sub.Close()
			return ev, nil
		}
	}
}
