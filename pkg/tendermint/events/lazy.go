package events

import (
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
)

func WaitActionAsync(hub *Hub, eventType string, module string, action string) (WaitEventFunc, error) {
	sub, err := Subscribe(hub, eventType, module, action)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (types.Event, error) {
		ev := <-evChan
		err := <-errChan
		sub.Close()

		return ev, err
	}, nil
}

func NextActionAsync(hub *Hub, eventType string, module string, action string) (WaitEventFunc, FilteredSubscriber, error) {
	sub, err := Subscribe(hub, eventType, module, action)
	if err != nil {
		return nil, FilteredSubscriber{}, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (ev types.Event, err error) {
		//fmt.Printf("> Waiting for next action %s.%s.action='%s'...\n", module, eventType, action)

		return <-evChan, <-errChan
	}, sub, nil
}

func GetFilterableWaitActionAsync(hub *Hub, eventType string, module string, action string) (FilterableEventFunc, error) {
	sub, err := Subscribe(hub, eventType, module, action)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func(filter EventFilterFunc) (ev types.Event, err error) {
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

func FilterNextAction(wait WaitEventFunc, filter EventFilterFunc) (types.Event, error) {
	for {
		ev, err := wait()
		if err != nil {
			return ev, err
		}

		if filter(ev) {
			return ev, nil
		}
	}
}

func ConsumeFilteredSubscriptionEvents(sub FilteredSubscriber, evChan chan types.Event, errChan chan error) {
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
