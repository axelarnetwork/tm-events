package events

import (
	"context"
	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	abci "github.com/tendermint/tendermint/abci/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/rpc/client"
	tmrpctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const WebsocketQueueSize = 32768

func SubscribeTendermintQuery(ctx context.Context, client client.EventsClient, query tmpubsub.Query) (<-chan tmrpctypes.ResultEvent, error) {

	// Subscriber param is ignored because Tendermint will override it with remote IP anyway.
	msgCh, err := client.Subscribe(ctx, "", query.String(), WebsocketQueueSize)
	if err != nil {
		return nil, err
	}

	return msgCh, nil
}

func processACBIEvents(bus pubsub.Bus, events []abci.Event) {
	for _, ev := range events {
		if mev, ok := ProcessEvent(ev); ok {
			if err := bus.Publish(mev); err != nil {
				bus.Close()
				return
			}
			continue
		}
	}
}
