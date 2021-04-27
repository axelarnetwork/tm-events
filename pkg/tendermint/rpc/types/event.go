package types

import (
	"fmt"
	"github.com/acrazing/cheapjson"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strings"
)

// @TODO use ABCI event
type Attribute struct {
	Type  string
	Key   string
	Value string
}

func ParseEventAttr(log *cheapjson.Value, msgType string, key string) *Attribute {
	path := fmt.Sprintf("%s.%s", msgType, key)
	strVal := log.Get(path, "0")
	if strVal != nil {
		return &Attribute{
			Key:   key,
			Value: strVal.String(),
		}
	}
	return nil
}

type Event struct {
	AttrByType map[string]map[string]string
}

func ExtractLogsAttribute(logs sdk.ABCIMessageLogs, key string) (val string) {
	for _, log := range logs {
		for _, event := range log.Events {
			for _, attr := range event.Attributes {
				if attr.Key == key {
					val = attr.Value
					return
				}
			}
		}
	}
	return
}

func PrintLogs(logs sdk.ABCIMessageLogs) (val string) {
	for _, log := range logs {
		for _, event := range log.Events {
			fmt.Printf("event %s\n", event.Type)
			for _, attr := range event.Attributes {
				fmt.Printf("\t%s = %s\n", attr.Key, attr.Value)
			}
		}
	}
	return
}

func ParseEventsFromLog(attributes map[string]*cheapjson.Value) Event {
	events := Event{AttrByType: make(map[string]map[string]string)}
	fmt.Printf("\n\tReceived event:\n")
	for label, value := range attributes {
		s := strings.Split(label, ".")
		eventType := s[0]
		key := s[1]
		val := value.Get("0").String()
		_, ok := events.AttrByType[eventType]
		if !ok {
			events.AttrByType[eventType] = make(map[string]string)
		}
		events.AttrByType[eventType][key] = val
		fmt.Printf("\t%s.%s = %s\n", eventType, key, val)
	}
	return events
}

func (es Event) GetAttr(eventType string, key string) (attr *Attribute) {
	//entry := es.attributes[fmt.Sprintf("%s.%s", eventType, key)]
	val := es.AttrByType[eventType][key]
	if len(val) > 0 {
		attr = &Attribute{Type: eventType, Key: key, Value: val}
	}
	return
}
