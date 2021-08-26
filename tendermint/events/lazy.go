package events

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/tm-events/tendermint/types"

	"github.com/axelarnetwork/tm-events/events"
)

const LazyEventBufferSize = 1000

// WaitActionAsync subscribes to events by eventType and module, filtering for results which have the provided action value.
// It returns a function which lazy loads the next event from the subscription then closes the subscription when called.
func WaitActionAsync(eventBus *events.Bus, eventType string, module string, action string, logger log.Logger) (NextEventFunc, error) {
	q := events.Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}

	sub, err := events.Subscribe(eventBus, q)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, LazyEventBufferSize)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (types.Event, error) {
		defer sub.Close()
		logger.Debug(fmt.Sprintf("waiting for action %s.%s.action='%s'", module, eventType, action), "module", module, "eventType", eventType, "action", action)
		ev := <-evChan
		err := <-errChan

		return ev, err
	}, nil
}

// NextActionAsync subscribes to events by eventType and module, filtering for results which have the provided action value.
// It returns a function which lazy loads the next event from the subscription every time it is called.
func NextActionAsync(eventBus *events.Bus, eventType string, module string, action string, logger log.Logger) (NextEventFunc, events.FilteredSubscriber, error) {
	q := events.Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}

	sub, err := events.Subscribe(eventBus, q)
	if err != nil {
		return nil, events.FilteredSubscriber{}, err
	}

	evChan := make(chan types.Event, LazyEventBufferSize)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (ev types.Event, err error) {
		logger.Debug(fmt.Sprintf("waiting for next action %s.%s.action='%s'", module, eventType, action), "module", module, "eventType", eventType, "action", action)

		return <-evChan, <-errChan
	}, sub, nil
}

// NextFilteredEventAsync subscribes to events by eventType and module, filtering for results which match the predicate.
// It returns a function which lazy loads the next event from the subscription every time it is called.
func NextFilteredEventAsync(eventBus *events.Bus, eventType string, module string, predicate EventPredicateFunc, logger log.Logger) (NextEventFunc, events.FilteredSubscriber, error) {
	q := events.Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && predicate(e)
		},
	}

	sub, err := events.Subscribe(eventBus, q)
	if err != nil {
		return nil, events.FilteredSubscriber{}, err
	}

	evChan := make(chan types.Event, LazyEventBufferSize)
	errChan := make(chan error, 1)

	go ConsumeFilteredSubscriptionEvents(sub, evChan, errChan)

	return func() (ev types.Event, err error) {
		logger.Debug(fmt.Sprintf("waiting for next event from query %s", q.TMQuery.String()), "module", module, "eventType", eventType)

		return <-evChan, <-errChan
	}, sub, nil
}

// WaitActionAsync subscribes to events using an arbitrary tendermint query.
// It returns a function which lazy loads the next event from the subscription then closes the subscription when called.
func WaitQueryAsync(eventBus *events.Bus, query tmpubsub.Query, logger log.Logger) (NextEventFunc, error) {
	// todo: test to confirm events are caught and buffered before NextEventFunc is called (lazy loading functioning correctly)
	sub, err := eventBus.Subscribe(query)
	if err != nil {
		return nil, err
	}

	evChan := make(chan types.Event, LazyEventBufferSize)
	errChan := make(chan error, 1)

	go consumeSubscriptionEvents(sub, evChan, errChan)

	return func() (types.Event, error) {
		defer sub.Close()
		logger.Debug(fmt.Sprintf("waiting for event from query \"%s\"", query.String()))

		return <-evChan, <-errChan
	}, nil
}

// WaitFilteredEvent lazy loads events until the filter predicate evaluates to true.
func WaitFilteredEvent(next NextEventFunc, filter EventPredicateFunc) (types.Event, error) {
	for {
		ev, err := next()
		if err != nil {
			return ev, err
		}

		if filter(ev) {
			return ev, nil
		}
	}
}

// ConsumeFilteredSubscriptionEvents consumes events from a filter subscription, writing them to the event and error channels until the subscription is closed.
// It is intended to run in its own goroutine.
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

// MatchAttributesPredicate returns an event predicate which evaluates to false if an event does not contain the attributes with the exact values specified in expectedAttributes.
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

// HasAttributesPredicate returns an event predicate which evaluates to false if an event does not contain the attribute keys listed in expectedAttributes.
func HasAttributesPredicate(expectedAttributes []string) EventPredicateFunc {
	mask := make(map[string]bool)

	return func(event types.Event) bool {
		for _, attribute := range event.Attributes {
			mask[attribute.Key] = true
		}

		for _, key := range expectedAttributes {
			if mask[key] == false {
				return false
			}
		}
		return true
	}
}
