package events

import (
	abci "github.com/tendermint/tendermint/abci/types"
)

// Event stores type, module, action and attributes list of sdk
// Deprecated
type Event struct {
	Type       string
	Attributes map[string]string
	Height     int64
}

// ABCIEventWithHeight adds a height field to abci.Event
type ABCIEventWithHeight struct {
	Height int64
	abci.Event
}

// Parse parses string to event
func Parse(event ABCIEventWithHeight) Event {
	e := Event{Type: event.Type, Attributes: make(map[string]string), Height: event.Height}

	for _, attribute := range event.Attributes {
		if len(attribute.Key) == 0 {
			continue
		}
		e.Attributes[string(attribute.Key)] = string(attribute.Value)
	}

	return e
}
