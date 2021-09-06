package events

import (
	"fmt"
	"strings"

	"github.com/axelarnetwork/utils/jobs"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/tm-events/pubsub"
)

// FilteredSubscriber filters events of a subscriber according to a predicate
type FilteredSubscriber struct {
	pubsub.Subscriber
	eventChan chan Event
	predicate func(event Event) bool
}

// NewFilteredSubscriber creates a FilteredSubscriber
func NewFilteredSubscriber(subscriber pubsub.Subscriber, predicate func(event Event) bool) FilteredSubscriber {
	s := FilteredSubscriber{Subscriber: subscriber, predicate: predicate, eventChan: make(chan Event)}

	go func() {
		for event := range s.Subscriber.Events() {
			switch e := event.(type) {
			case Event:
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
func (s FilteredSubscriber) Events() <-chan Event {
	return s.eventChan
}

// Consume processes all events from the given subscriber with the given function.
// Do not consume the same subscriber multiple times.
func Consume(subscriber FilteredSubscriber, process func(event Event) error) jobs.Job {
	return func(errChan chan<- error) {
		for {
			select {
			case e := <-subscriber.Events():
				go func() {
					defer recovery(errChan)
					if err := process(e); err != nil {
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
func OnlyBlockHeight(f func(int64)) func(event Event) error {
	return func(e Event) error { f(e.Height); return nil }
}

// QueryBuilder is a builder struct to create a pubsub.Query
type QueryBuilder struct {
	eventType string
	ands      []string
}

// NewTxEventQuery initializes a QueryBuilder for a query that matches the given transaction event type
func NewTxEventQuery(eventType string) QueryBuilder {
	return QueryBuilder{eventType: eventType}.Match(tm.EventTypeKey, tm.EventTx)
}

// NewBlockHeaderEventQuery initializes a QueryBuilder for a query that matches the given block header event type
func NewBlockHeaderEventQuery(eventType string) QueryBuilder {
	return QueryBuilder{eventType: eventType}.Match(tm.EventTypeKey, tm.EventNewBlockHeader)
}

// Match adds a predicate to match the given key and value exactly to the query
func (q QueryBuilder) Match(key, value string) QueryBuilder {
	q.ands = append(q.ands, fmt.Sprintf("%s='%s'", key, value))
	return q
}

// MatchAction adds a predicate to match the given action event attribute to the query
func (q QueryBuilder) MatchAction(action string) QueryBuilder {
	return q.MatchAttributes(sdk.Attribute{Key: sdk.AttributeKeyAction, Value: action})
}

// MatchModule adds a predicate to match the given module event attribute to the query
func (q QueryBuilder) MatchModule(module string) QueryBuilder {
	return q.MatchAttributes(sdk.Attribute{Key: sdk.AttributeKeyModule, Value: module})
}

// MatchAttributes adds a predicate to match all the given event attributes to the query
func (q QueryBuilder) MatchAttributes(attributes ...sdk.Attribute) QueryBuilder {
	for _, attribute := range attributes {
		q.ands = append(q.ands, fmt.Sprintf("%s='%s'", CompositeKey(q.eventType, attribute.Key), attribute.Value))
	}
	return q
}

// Build creates a pubsub.Query from all given predicates
func (q QueryBuilder) Build() tmpubsub.Query {
	return query.MustParse(strings.Join(q.ands, " AND "))
}

// Publisher can create a subscription for the given query
type Publisher interface {
	Subscribe(tmpubsub.Query) (pubsub.Subscriber, error)
}

// Query represents a query used to subscribe a FilteredSubscriber to an event
type Query struct {
	TMQuery   tmpubsub.Query
	Predicate func(event Event) bool
}

// QueryTxEventByAttributes creates a Query for a transaction event with the given attributes
func QueryTxEventByAttributes(eventType string, module string, attributes ...sdk.Attribute) Query {
	return Query{
		TMQuery: NewTxEventQuery(eventType).MatchModule(module).MatchAttributes(attributes...).Build(),
		Predicate: func(e Event) bool {
			return e.Type == eventType && e.Attributes[sdk.AttributeKeyModule] == module && matchAll(e, attributes...)
		},
	}
}

func matchAll(event Event, attributes ...sdk.Attribute) bool {
	for _, attribute := range attributes {
		if event.Attributes[attribute.Key] != attribute.Value {
			return false
		}
	}
	return true
}

// QueryBlockHeader creates a query that matches new block events once per block
func QueryBlockHeader() Query {
	blockHeight := int64(-1)
	return Query{
		TMQuery: QueryBuilder{}.Match(tm.EventTypeKey, tm.EventNewBlockHeader).Build(),
		Predicate: func(e Event) bool {
			if e.Height != blockHeight {
				blockHeight = e.Height
				return true
			}
			return false
		},
	}
}

// MustSubscribeWithAttributes panics if subscription to the transaction event fails
func MustSubscribeWithAttributes(pub Publisher, eventType string, module string, attributes ...sdk.Attribute) FilteredSubscriber {
	return MustSubscribe(pub, QueryTxEventByAttributes(eventType, module, attributes...),
		func(err error) error {
			return sdkerrors.Wrapf(err, "subscription to event {type %s, module %s, attributes %v} failed", eventType, module, attributes)
		})
}

// MustSubscribeBlockHeader panics if subscription to the block header event fails
func MustSubscribeBlockHeader(pub Publisher) FilteredSubscriber {
	q := QueryBlockHeader()
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
