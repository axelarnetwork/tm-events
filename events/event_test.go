package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestEventMarshalling(t *testing.T) {
	actualEvent := ABCIEventWithHeight{
		Height: 120,
		Event: abci.Event{
			Type: "eventType",
			Attributes: []abci.EventAttribute{
				{Key: []byte("key1"), Value: []byte("value1")},
			},
		},
	}

	bz, err := actualEvent.Marshal()
	assert.NoError(t, err)

	var unmarshalledEvent ABCIEventWithHeight

	assert.NoError(t, unmarshalledEvent.Unmarshal(bz))
	assert.Equal(t, actualEvent, unmarshalledEvent)
}
