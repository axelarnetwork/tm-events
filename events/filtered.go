package events

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tm "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/utils/jobs"

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
func Consume(subscriber FilteredSubscriber, process func(event Event)) jobs.Job {
	return func(ctx context.Context) error {
		errs := make(chan error, 1)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errs:
				return err
			case e := <-subscriber.Events():
				go func() {
					defer recovery(errs)
					process(e)
				}()
			case <-subscriber.Done():
				return nil
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
func OnlyBlockHeight(f func(int64) error) func(event Event) error {
	return func(e Event) error { return f(e.Height) }
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

// AttributeValueSet represents a set of possible values for an Attribute key
type AttributeValueSet struct {
	key    string
	values map[string]struct{}
}

// NewAttributeValueSet creates a set of possible values for an Attribute key from a list of strings
func NewAttributeValueSet(key string, values ...string) AttributeValueSet {
	valMap := make(map[string]struct{})

	for _, v := range values {
		valMap[v] = struct{}{}
	}

	return AttributeValueSet{
		key:    key,
		values: valMap,
	}
}

// Match checks whether the passed event contains an attribute whose value is contained by the set
func (s AttributeValueSet) Match(e Event) bool {
	if key, ok := e.Attributes[s.key]; ok {
		if _, ok := s.values[key]; ok {
			return true
		}
	}
	return false
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

// QueryBlockEventByAttributes creates a Query for a block event with the given attributes
func QueryBlockEventByAttributes(eventType string, module string, attributes ...sdk.Attribute) Query {
	return Query{
		TMQuery: NewBlockHeaderEventQuery(eventType).MatchModule(module).MatchAttributes(attributes...).Build(),
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

// QueryTxEventByValueSets creates a Query for a transaction event with at least one attribute value
// contained in every provided attribute value set.
func QueryTxEventByValueSets(eventType string, module string, sets ...AttributeValueSet) Query {
	return Query{
		TMQuery: NewTxEventQuery(eventType).MatchModule(module).Build(),
		Predicate: func(e Event) bool {
			return e.Type == eventType && e.Attributes[sdk.AttributeKeyModule] == module && matchAllValueSets(e, sets...)
		},
	}
}

// QueryBlockEventByValueSets creates a Query for a block event with at least one attribute value
// contained in every provided attribute value set.
func QueryBlockEventByValueSets(eventType string, module string, sets ...AttributeValueSet) Query {
	return Query{
		TMQuery: NewBlockHeaderEventQuery(eventType).MatchModule(module).Build(),
		Predicate: func(e Event) bool {
			return e.Type == eventType && e.Attributes[sdk.AttributeKeyModule] == module && matchAllValueSets(e, sets...)
		},
	}
}

func matchAllValueSets(event Event, sets ...AttributeValueSet) bool {
	for _, s := range sets {
		if s.Match(event) {
			return true
		}
	}
	return false
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

// MustSubscribeWithValueSets panics if subscription to the transaction event fails
func MustSubscribeWithValueSets(pub Publisher, eventType string, module string, sets ...AttributeValueSet) FilteredSubscriber {
	return MustSubscribe(pub, QueryTxEventByValueSets(eventType, module, sets...),
		func(err error) error {
			return sdkerrors.Wrapf(err, "subscription to event {type %s, module %s, attributes %v} failed", eventType, module, sets)
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
