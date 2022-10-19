package events

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-errors/errors"
	"github.com/gogo/protobuf/proto"

	"github.com/axelarnetwork/utils/jobs"
)

// Filter returns true if an event is of the given generic type, false otherwise
func Filter[T proto.Message]() func(e ABCIEventWithHeight) bool {
	return func(e ABCIEventWithHeight) bool {
		typedEvent, err := sdk.ParseTypedEvent(e.Event)
		if err != nil {
			return false
		}

		return proto.MessageName(typedEvent) == proto.MessageName(*new(T))
	}
}

// Consume processes all events from the given subscriber with the given function.
// Do not consume the same subscriber multiple times.
func Consume[T any](subscriber <-chan T, process func(event T)) jobs.Job {
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
		err := fmt.Errorf("job panicked: %s\n%s", r, errors.Wrap(r, 1).Stack())
		errChan <- err
	}
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
