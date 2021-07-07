package events

import (
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"
)

// WaitActionAsync subscribes to events by eventType and module, filtering for results which have the provided action value.
// It returns a function which lazy loads the next event from the subscription then closes the subscription when called.
func WaitActionAsync(hub *Hub, eventType string, module string, action string) (NextEventFunc, error) {
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

// NextActionAsync subscribes to events by eventType and module, filtering for results which have the provided action value.
// It returns a function which lazy loads the next event from the subscription every time it is called.
func NextActionAsync(hub *Hub, eventType string, module string, action string) (NextEventFunc, FilteredSubscriber, error) {
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

// NextFilteredEventAsync subscribes to events by eventType and module, filtering for results which match the predicate.
// It returns a function which lazy loads the next event from the subscription every time it is called.
func NextFilteredEventAsync(hub *Hub, eventType string, module string, predicate func(event types.Event) bool) (NextEventFunc, FilteredSubscriber, error) {
	q := Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && predicate(e)
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
		hub.Logger.Debug(fmt.Sprintf("waiting for next event from query %s", q.TMQuery.String()), "module", module, "eventType", eventType)

		return <-evChan, <-errChan
	}, sub, nil
}

func WaitQueryAsync(hub *Hub, query tmpubsub.Query) (NextEventFunc, error) {
	// todo: test to confirm events are caught before NextEventFunc is called (lazy loading functioning correctly)
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

	return func(filter EventPredicateFunc) (ev types.Event, err error) {
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

func WaitForPredicateMatch(wait NextEventFunc, filter EventPredicateFunc) (types.Event, error) {
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

func MatchAttributesPredicate(expectedAttributes map[string]string) EventPredicateFunc {
	mask := make(map[string]bool)

	return func(event types.Event) bool {
		for _, attribute := range event.Attributes {
			if attribute.Value == expectedAttributes[attribute.Key] {
				mask[attribute.Key] = true
			}
		}

		if len(mask) != len(expectedAttributes) {
			// skip checking mask if incorrect number of attributes have been set
			return false
		}

		for key, _ := range expectedAttributes {
			if mask[key] == false {
				return false
			}
		}
		return true
	}
}
