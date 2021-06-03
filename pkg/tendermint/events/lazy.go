package events

import (
	"fmt"

	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"
)

func WaitActionAsync(hub *Hub, eventType string, module string, action string) (WaitEventFunc, error) {
	q := Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}

	sub, err := Subscribe(hub, q)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (types.Event, error) {
		defer sub.Close()
		hub.Logger.Debug(fmt.Sprintf("waiting for action %s.%s.action='%s'", module, eventType, action), "module", module, "eventType", eventType, "action", action)
		ev := <-evChan
		err := <-errChan

		return ev, err
	}, nil
}

func NextActionAsync(hub *Hub, eventType string, module string, action string) (WaitEventFunc, FilteredSubscriber, error) {
	q := Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}

	sub, err := Subscribe(hub, q)
	if err != nil {
		return nil, FilteredSubscriber{}, err
	}

	evChan := make(chan types.Event, 1)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (ev types.Event, err error) {
		hub.Logger.Debug(fmt.Sprintf("waiting for next action %s.%s.action='%s'", module, eventType, action), "module", module, "eventType", eventType, "action", action)

		return <-evChan, <-errChan
	}, sub, nil
}

func WaitQueryAsync(hub *Hub, query tmpubsub.Query) (WaitEventFunc, error) {
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
		hub.Logger.Debug(fmt.Sprintf("waiting for event from query \"%s\"", query.String()))

		return <-evChan, <-errChan
	}, nil
}

func GetFilterableWaitActionAsync(hub *Hub, eventType string, module string, action string) (FilterableEventFunc, error) {
	q := Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}

	sub, err := Subscribe(hub, q)
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
