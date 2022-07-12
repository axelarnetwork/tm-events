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

// Map transforms the ABCIEventWithHeight into an Event
// Deprecated
func Map(event abci.Event) Event {
	e := Event{Type: event.Type, Attributes: make(map[string]string)}

	for _, attribute := range event.Attributes {
		if len(attribute.Key) == 0 {
			continue
		}
		e.Attributes[string(attribute.Key)] = string(attribute.Value)
	}

	return e
}
