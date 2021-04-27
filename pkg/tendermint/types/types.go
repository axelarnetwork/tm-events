package types

import (
	"errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EventTypeMessage = "message"
	EventAttrAction  = "action"
)

var (
	// ErrNotFound is the error with message "Not found"
	ErrNotFound = errors.New("Not found")
	// ErrUnknownType is the error with message "Unknown type"
	ErrUnknownType = errors.New("Unknown type")
	// ErrUnknownModule is the error with message "Unknown module"
	ErrUnknownModule = errors.New("Unknown module")
	// ErrUnknownAction is the error with message "Unknown action"
	ErrUnknownAction = errors.New("Unknown action")
	// ErrParsingBlockID indicates one of the uint parsers failed to convert a value.
	ErrParsingBlockID = errors.New("error parsing block id values")
	// ErrInvalidParseBlockIDInput indicates the splitting of block path failed.
	ErrInvalidParseBlockIDInput = errors.New("error parsing block id input string")
)

// Event stores type, module, action and attributes list of sdk
type Event struct {
	Type       string
	Module     string
	Action     string
	Attributes []sdk.Attribute
	Height     int64
}

type Param struct {
	Key   string
	Value string
}

// ParseEvent parses string to event
func ParseEvent(sev sdk.StringEvent) (Event, error) {

	ev := Event{Type: sev.Type, Attributes: sev.Attributes}
	var err error

	if ev.Module, err = GetAttrString(sev.Attributes, sdk.AttributeKeyModule); err != nil {
	}

	if ev.Action, err = GetAttrString(sev.Attributes, sdk.AttributeKeyAction); err != nil {
		//return ev, fmt.Errorf("failed to parse evevent: %w", err)
	}

	return ev, nil
}

// GetString take sdk attributes, key and returns key value. Returns error incase of failure.
func GetAttrString(attrs []sdk.Attribute, key string) (string, error) {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Value, nil
		}
	}
	return "", ErrNotFound
}
