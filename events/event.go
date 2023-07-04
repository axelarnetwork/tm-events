package events

import (
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/json"
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

// Marshal extends the Marshal function of abci.Event to ABCIEventWithHeight
func (e *ABCIEventWithHeight) Marshal() (dAtA []byte, err error) {
	bz, err := e.Event.Marshal()
	if err != nil {
		return nil, err
	}

	data := struct {
		Height int64
		Event  []byte
	}{
		Height: e.Height,
		Event:  bz,
	}
	return json.Marshal(data)
}

// Unmarshal extends the Unmarshal function of abci.Event to ABCIEventWithHeight
func (e *ABCIEventWithHeight) Unmarshal(dAta []byte) error {
	data := struct {
		Height int64
		Event  []byte
	}{}

	if err := json.Unmarshal(dAta, &data); err != nil {
		return err
	}

	var event abci.Event
	if err := event.Unmarshal(data.Event); err != nil {
		return err
	}

	e.Height = data.Height
	e.Event = event
	return nil
}

// Map transforms the ABCIEventWithHeight into an Event
// Deprecated
func Map(event ABCIEventWithHeight) Event {
	e := Event{Type: event.Type, Attributes: make(map[string]string), Height: event.Height}

	for _, attribute := range event.Attributes {
		if len(attribute.Key) == 0 {
			continue
		}
		e.Attributes[string(attribute.Key)] = string(attribute.Value)
	}

	return e
}
