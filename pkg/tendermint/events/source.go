package events

import (
	"context"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/rpc/client"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"golang.org/x/sync/errgroup"
)

type Source interface {
	Close()
	Start(context.Context, *errgroup.Group, SinkFunc)
}

type ResultEventSource struct {
	events <-chan coretypes.ResultEvent
	query  pubsub.Query
	client client.EventsClient

	ctx context.Context
	eg  *errgroup.Group
}

const subscriber = "axelar-message"

func NewResultEventSource(events <-chan coretypes.ResultEvent, client client.EventsClient, query pubsub.Query) ResultEventSource {
	return ResultEventSource{
		events: events,
		query:  query,
		client: client,
	}
}

func (es ResultEventSource) Close() {
	es.eg.Go(func() error {
		// closes events channel
		err := es.client.Unsubscribe(es.ctx, subscriber, es.query.String())
		if err != nil {
			return err
		}
		return nil
	})
}

func (es ResultEventSource) Start(ctx context.Context, eg *errgroup.Group, sinkFunc SinkFunc) {
	es.ctx = ctx
	es.eg = eg
	startProcessingResultEvents(ctx, eg, es.events, sinkFunc)
}

// startProcessingResultEvents is a sink pipeline stage for parsing events from a tendermint subscription
func startProcessingResultEvents(ctx context.Context, eg *errgroup.Group, inCh <-chan coretypes.ResultEvent, sinkFunc SinkFunc) {
	eg.Go(func() error {
		for ed := range inCh {
			// extract event attributes
			switch evt := ed.Data.(type) {
			case tmtypes.EventDataTx:
				if !evt.Result.IsOK() {
					continue
				}
				err := processTxEventData(evt, sinkFunc)
				if err != nil {
					return err
				}
			case tmtypes.EventDataNewBlockHeader:
				err := processBlockHeaderEventData(evt, sinkFunc)
				if err != nil {
					return err
				}
			default:
			}
		}
		return nil
	})
}

func processTxEventData(data tmtypes.EventDataTx, sinkFunc SinkFunc) error {
	events := data.Result.GetEvents()
	for _, ev := range events {
		if mev, ok := ProcessEvent(ev); ok {
			mev.Height = data.Height
			if err := sinkFunc(mev); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

func processBlockHeaderEventData(data tmtypes.EventDataNewBlockHeader, sinkFunc SinkFunc) error {
	events := append(data.ResultEndBlock.Events, data.ResultBeginBlock.Events...)
	for _, ev := range events {
		if mev, ok := ProcessEvent(ev); ok {
			mev.Height = data.Header.Height
			if err := sinkFunc(mev); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

func ProcessEvent(bev abci.Event) (types.Event, bool) {
	// todo: parse cmd based on module
	ev, err := types.ParseEvent(sdk.StringifyEvent(bev))
	if err != nil {
		return types.Event{}, false
	}

	return ev, true
}
