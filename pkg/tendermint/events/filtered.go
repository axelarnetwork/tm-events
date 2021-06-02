package events

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
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

type Query struct {
	TMQuery           *query.Query
	DetailedPredicate func(event types.Event) bool
}

// MustSubscribeTx panics if subscription to the transaction event fails
func MustSubscribeTx(hub *Hub, eventType string, module string, action string) FilteredSubscriber {
	q := Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		DetailedPredicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}
	return MustSubscribe(hub, q,
		func(err error) error {
			return sdkerrors.Wrapf(err, "subscription to event {type %s, module %s, action %s} failed", eventType, module, action)
		})
}

// MustSubscribeNewBlockHeader panics if subscription to the block header event fails
func MustSubscribeNewBlockHeader(hub *Hub) FilteredSubscriber {
	q := Query{
		TMQuery:           query.MustParse(fmt.Sprintf("%s='%s'", tm.EventTypeKey, tm.EventNewBlockHeader)),
		DetailedPredicate: func(e types.Event) bool { return e.Type == minttypes.EventTypeMint },
	}
	return MustSubscribe(hub, q,
		func(err error) error { return sdkerrors.Wrapf(err, "subscription to block header failed") })
}

// MustSubscribe panics if the subscription to the given query fails
func MustSubscribe(hub *Hub, query Query, onFail func(error) error) FilteredSubscriber {
	subscriber, err := Subscribe(hub, query)
	if err != nil {
		panic(onFail(err))
	}
	return subscriber
}

// Subscribe returns a filtered subscriber that only streams events of the given type, module and action
func Subscribe(hub *Hub, q Query) (FilteredSubscriber, error) {
	subscriber, err := hub.Subscribe(q.TMQuery)
	if err != nil {
		return FilteredSubscriber{}, err
	}
	return NewFilteredSubscriber(subscriber, q.DetailedPredicate), nil
}
