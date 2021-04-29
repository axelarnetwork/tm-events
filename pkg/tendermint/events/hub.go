package events

import (
	"context"
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/rpc/client"
	"log"
	"sync"
)

// Hub allows many subscribers to share a single subscription source non-exclusively. Consuming an event on a subscriber does not consume the event on other subscribers.
// Hub is a one-to-one mapping of tendermint queries to relays. It proxy's subscriptions to queries by returning a relay if one already exists.
type Hub struct {
	evBus *client.EventsClient
	ctx   context.Context

	relays map[string]Relay
	logger *log.Logger

	mtx sync.RWMutex
}

func NewHub(events client.EventsClient) Hub {
	return Hub{
		evBus:  &events,
		ctx:    context.Background(),
		relays: make(map[string]Relay),
		//todo: properly instantiate this logger
		logger: &log.Logger{},
	}
}

// Subscribe to messages of a particular action
func (h *Hub) SubscribeMessage(action string) (pubsub.Subscriber, error) {
	query := types.MsgQueryAction(action)
	sub, err := h.Subscribe(query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to subscribe to query '%s'", query.String())
	}

	return sub, nil
}

// Subscribe to a query, connect it to a bus, then return a subscriber to that bus.
// If already subscribed, return the subscriber
// Subscribe is thread safe.
func (h *Hub) Subscribe(query tmpubsub.Query) (pubsub.Subscriber, error) {
	relay, ok := h.relays[query.String()]
	if !ok {
		h.mtx.Lock()
		defer h.mtx.Unlock()

		ctx := context.Background()

		sourceCh, err := SubscribeTendermintQuery(ctx, *h.evBus, query)
		if err != nil {
			return nil, err
		}

		source := NewResultEventSource(sourceCh, *h.evBus, query)

		relay = NewRelay(ctx, source)
		relay.Start()
		relay.CatchAsync(relayErrorHandler(query.String()))

		h.relays[query.String()] = relay
	}

	return relay.Subscribe()
}

func relayErrorHandler(query string) func(error) {
	// todo: allow error handler to be passed at hub instantiation
	return func(err error) {
		fmt.Printf("Relay failed (query %s): %s", query, err)
	}
}

func (h *Hub) Unsubscribe() error {
	// todo: implement
	return nil
}

func (h *Hub) UnsubscribeAll() error {
	// todo: implement
	return nil
}
