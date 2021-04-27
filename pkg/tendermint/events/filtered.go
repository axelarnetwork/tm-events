package events

import (
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"
)

// FilteredSubscriber filters events of a subscriber according to a predicate
type FilteredSubscriber struct {
	pubsub.Subscriber
	eventChan chan types.Event
	predicate func(event types.Event) bool
}

// NewFilteredSubscriber creates a FilteredSubscriber
func NewFilteredSubscriber(subscriber pubsub.Subscriber, predicate func(event types.Event) bool) FilteredSubscriber {
	s := FilteredSubscriber{Subscriber: subscriber, predicate: predicate, eventChan: make(chan types.Event)}

	go func() {
		for event := range s.Subscriber.Events() {
			switch e := event.(type) {
			case types.Event:
				if predicate(e) {
					s.eventChan <- e
				}
			default:
				panic(fmt.Sprintf("unexpected event type %t", event))
			}
		}
	}()
	return s
}

// Events returns a channel of filtered events
func (s FilteredSubscriber) Events() <-chan types.Event {
	return s.eventChan
}

// MustSubscribe panics if Subscribe fails
func MustSubscribe(hub *Hub, eventType string, module string, action string) FilteredSubscriber {
	subscriber, err := Subscribe(hub, eventType, module, action)
	if err != nil {
		panic(sdkerrors.Wrapf(err, "subscription to event {type %s, module %s, action %s} failed", eventType, module, action))
	}
	return subscriber
}

// Subscribe returns a filtered subscriber that only streams events of the given type, module and action
func Subscribe(hub *Hub, eventType string, module string, action string) (FilteredSubscriber, error) {
	qString := fmt.Sprintf("%s='%s' AND %s.%s='%s'",
		tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)
	subscriber, err := hub.Subscribe(query.MustParse(qString))
	if err != nil {
		return FilteredSubscriber{}, err
	}
	return NewFilteredSubscriber(
		subscriber,
		func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	), nil
}
