package events

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/assert"
)

func TestEventMarshalling(t *testing.T) {
	actualEvent := ABCIEventWithHeight{
		Height: 120,
		Event: abci.Event{
			Type: "eventType",
			Attributes: []abci.EventAttribute{
				{Key: "key1", Value: "value1"},
			},
		},
	}

	bz, err := actualEvent.Marshal()
	assert.NoError(t, err)

	var unmarshalledEvent ABCIEventWithHeight

	assert.NoError(t, unmarshalledEvent.Unmarshal(bz))
	assert.Equal(t, actualEvent, unmarshalledEvent)
}
