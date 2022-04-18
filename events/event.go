package events

import (
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"
)

// Event stores type, module, action and attributes list of sdk
// Deprecated
type Event struct {
	Type       string
	Attributes map[string]string
	Height     int64
}

// Parse parses string to event
func Parse(event abci.Event) (Event, error) {
	e := Event{Type: event.Type, Attributes: make(map[string]string)}

	for _, attribute := range event.Attributes {
		if len(attribute.Key) == 0 {
			continue
		}
		e.Attributes[string(attribute.Key)] = string(attribute.Value)
	}

	return e, nil
}

// Flatten transforms all given events into a map in the form { eventType.attributeKey: attribute.Value }.
// In case of duplicate keys the last event wins.
func Flatten(events []abci.Event) map[string][]string {
	result := make(map[string][]string)
	for _, event := range events {
		if len(event.Type) == 0 {
			return nil
		}

		for _, attr := range event.Attributes {
			if len(attr.Key) == 0 {
				continue
			}

			compositeTag := CompositeKey(event.Type, string(attr.Key))
			result[compositeTag] = append(result[compositeTag], string(attr.Value))
		}
	}
	return result
}

// CompositeKey creates a key of the form eventType.attributeKey
func CompositeKey(eventType, attrKey string) string {
	return fmt.Sprintf("%s.%s", eventType, attrKey)
}
