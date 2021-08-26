package events

import (
	"fmt"

	"github.com/axelarnetwork/utils/jobs"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/tm-events/pubsub"
	"github.com/axelarnetwork/tm-events/tendermint/types"
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

// Query represents a query used to subscribe a FilteredSubscriber to an event
type Query struct {
	TMQuery   *query.Query
	Predicate func(event types.Event) bool
}

// MustSubscribeTx panics if subscription to the transaction event fails
func MustSubscribeTx(pub Publisher, eventType string, module string, action string) FilteredSubscriber {
	q := Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tm.EventTypeKey, tm.EventTx, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e types.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}
	return MustSubscribe(pub, q,
		func(err error) error {
			return sdkerrors.Wrapf(err, "subscription to event {type %s, module %s, action %s} failed", eventType, module, action)
		})
}

// MustSubscribeNewBlockHeader panics if subscription to the block header event fails
func MustSubscribeNewBlockHeader(pub Publisher) FilteredSubscriber {
	blockHeight := int64(-1)
	q := Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s'", tm.EventTypeKey, tm.EventNewBlockHeader)),
		Predicate: func(e types.Event) bool {
			if e.Height != blockHeight {
				blockHeight = e.Height
				return true
			}
			return false
		},
	}
	return MustSubscribe(pub, q,
		func(err error) error { return sdkerrors.Wrapf(err, "subscription to block header failed") })
}

// MustSubscribe panics if the subscription to the given query fails
func MustSubscribe(pub Publisher, query Query, onFail func(error) error) FilteredSubscriber {
	subscriber, err := Subscribe(pub, query)
	if err != nil {
		panic(onFail(err))
	}
	return subscriber
}

// Subscribe returns a filtered subscriber that only streams events of the given type, module and action
func Subscribe(pub Publisher, q Query) (FilteredSubscriber, error) {
	subscriber, err := pub.Subscribe(q.TMQuery)
	if err != nil {
		return FilteredSubscriber{}, err
	}
	return NewFilteredSubscriber(subscriber, q.Predicate), nil
}

// Publisher can create a subscription for the given query
type Publisher interface {
	Subscribe(tmpubsub.Query) (pubsub.Subscriber, error)
}

// Consume processes all events from the given subscriber with the given function.
// Do not consume the same subscriber multiple times.
func Consume(subscriber FilteredSubscriber, process func(blockHeight int64, attributes []sdk.Attribute) error) jobs.Job {
	return func(errChan chan<- error) {
		for {
			select {
			case e := <-subscriber.Events():
				go func() {
					defer recovery(errChan)
					if err := process(e.Height, e.Attributes); err != nil {
						errChan <- err
					}
				}()
			case <-subscriber.Done():
				return
			}
		}
	}
}

func recovery(errChan chan<- error) {
	if r := recover(); r != nil {
		errChan <- fmt.Errorf("job panicked:%s", r)
	}
}

// OnlyBlockHeight wraps a function that only depends on block height and makes it compatible with the Consume function
func OnlyBlockHeight(f func(int64)) func(int64, []sdk.Attribute) error {
	return func(h int64, _ []sdk.Attribute) error { f(h); return nil }
}

// OnlyAttributes wraps a function that only depends on event attributes and makes it compatible with the Consume function
func OnlyAttributes(f func([]sdk.Attribute) error) func(int64, []sdk.Attribute) error {
	return func(_ int64, a []sdk.Attribute) error { return f(a) }
}
