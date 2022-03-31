package events

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axelarnetwork/utils/jobs"
)

// Consume processes all events from the given subscriber with the given function.
// Do not consume the same subscriber multiple times.
func Consume(subscriber <-chan Event, process func(event Event)) jobs.Job {
	return func(ctx context.Context) error {
		errs := make(chan error, 1)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errs:
				return err
			case e, ok := <-subscriber:
				if !ok {
					return nil
				}
				go func() {
					defer recovery(errs)
					process(e)
				}()
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

// QueryEventByAttributes creates a Query for an event with the given attributes
func QueryEventByAttributes(eventType string, module string, attributes ...sdk.Attribute) func(event Event) bool {
	return func(e Event) bool {
		return e.Type == eventType && e.Attributes[sdk.AttributeKeyModule] == module && matchAll(e, attributes...)
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

// QueryEventByValueSets creates a Query for an event with at least one attribute value
// contained in every provided attribute value set.
func QueryEventByValueSets(eventType string, module string, sets ...AttributeValueSet) func(event Event) bool {
	return func(e Event) bool {
		return e.Type == eventType && e.Attributes[sdk.AttributeKeyModule] == module && matchAllValueSets(e, sets...)
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
func QueryBlockHeader() func(event Event) bool {
	blockHeight := int64(-1)
	return func(e Event) bool {
		if e.Height != blockHeight {
			blockHeight = e.Height
			return true
		}
		return false
	}
}
