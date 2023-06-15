package main

import (
	"context"
	"os"
	"sync"

	"github.com/axelarnetwork/utils/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client"

	"github.com/axelarnetwork/tm-events/events"
	"github.com/axelarnetwork/tm-events/pubsub"
	"github.com/axelarnetwork/tm-events/tendermint"
	"github.com/axelarnetwork/utils/chans"
	"github.com/axelarnetwork/utils/funcs"
	"github.com/axelarnetwork/utils/jobs"
)

func main() {
	var (
		rpcURL   string
		endpoint string
		eventBus *events.Bus
	)
	log.Setup(tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)).With("process", "tmrpc"))

	cmd := cobra.Command{
		Use:              "tmrpc",
		Short:            "Event listener",
		TraverseChildren: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			resettableClient := tendermint.NewRobustClient(func() (client.Client, error) {
				cl, err := tendermint.StartClient(rpcURL, endpoint)
				if err == nil {
					log.FromCtx(cmd.Context()).Info("connected tendermint client to " + rpcURL)
				}
				return cl, err
			})
			notifier := events.NewBlockNotifier(resettableClient)
			blockSource := events.NewBlockSource(resettableClient, notifier)
			eventBus = events.NewEventBus(blockSource, pubsub.NewBus[events.ABCIEventWithHeight]())
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&rpcURL, "address", tendermint.DefaultAddress, "tendermint RPC address")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", tendermint.DefaultWSEndpoint, "websocket endpoint")

	cmd.AddCommand(
		CmdWaitEvent(&eventBus),
	)

	if err := cmd.Execute(); err != nil {
		log.FromCtx(cmd.Context()).Error(err.Error())
		os.Exit(1)
	}
}

// CmdWaitEvent returns a cli command to wait for a specific event
func CmdWaitEvent(busPtr **events.Bus) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-event <event type> <module> [action]",
		Short: "Wait for the next matching event",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventBus := *busPtr

			eventType := args[0]
			module := args[1]
			var logKeyVals []interface{}
			logKeyVals = append(logKeyVals, "module", module)
			logKeyVals = append(logKeyVals, "eventType", eventType)

			var attributes []sdk.Attribute
			if len(args) == 3 {
				action := args[2]
				attributes = append(attributes, sdk.Attribute{Key: sdk.AttributeKeyAction, Value: action})
				logKeyVals = append(logKeyVals, "action", action)
			}

			ctx := log.AppendKeyVals(cmd.Context(), logKeyVals)
			shutdownBus := startFetchingEvents(ctx, eventBus)

			log.FromCtx(ctx).Debug("waiting for event")
			sub := eventBus.Subscribe(funcs.Compose(events.Map, events.QueryEventByAttributes(eventType, module, attributes...)))
			once := sync.Once{}
			job := events.Consume(chans.Map(sub, events.Map), func(e events.Event) {
				once.Do(func() {
					log.FromCtx(ctx).Debugf("found event %v", e)
					shutdownBus()
					<-eventBus.Done()
				})
			})
			mgr := jobs.NewMgr(context.Background())
			mgr.AddJob(job)

			<-mgr.Done()

			return nil
		},
	}
	return cmd
}

func startFetchingEvents(ctx context.Context, eventBus *events.Bus) context.CancelFunc {
	ctx, shutdownBus := context.WithCancel(ctx)
	errs := eventBus.FetchEvents(ctx)

	go func(ctx context.Context) {
		select {
		case err := <-errs:
			log.FromCtx(ctx).Error(err.Error())
			shutdownBus()
		case <-ctx.Done():
			return
		}
	}(ctx)
	return shutdownBus
}
