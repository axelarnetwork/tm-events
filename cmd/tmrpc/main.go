package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"
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
	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)).With("process", "tmrpc")

	cmd := cobra.Command{
		Use:              "tmrpc",
		Short:            "Event listener",
		TraverseChildren: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			resettableClient := tendermint.NewRobustClient(func() (client.Client, error) {
				cl, err := tendermint.StartClient(rpcURL, endpoint)
				if err == nil {
					logger.Info("connected tendermint client to " + rpcURL)
				}
				return cl, err
			})
			notifier := events.NewBlockNotifier(resettableClient, logger)
			blockSource := events.NewBlockSource(resettableClient, notifier, logger)
			eventBus = events.NewEventBus(blockSource, pubsub.NewBus[abci.Event](), logger)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&rpcURL, "address", tendermint.DefaultAddress, "tendermint RPC address")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", tendermint.DefaultWSEndpoint, "websocket endpoint")

	cmd.AddCommand(
		CmdWaitEvent(&eventBus, logger),
	)

	if err := cmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

// CmdWaitEvent returns a cli command to wait for a specific event
func CmdWaitEvent(busPtr **events.Bus, logger tmlog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-event <event type> <module> [action]",
		Short: "Wait for the next matching event",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventBus := *busPtr
			shutdownBus := startFetchingEvents(eventBus, logger)

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

			logger.Debug("waiting for event", logKeyVals...)
			sub := eventBus.Subscribe(funcs.Compose(events.Map, events.QueryEventByAttributes(eventType, module, attributes...)))
			once := sync.Once{}
			job := events.Consume(chans.Map(sub, events.Map), func(e events.Event) {
				once.Do(func() {
					logger.Debug(fmt.Sprintf("found event %v", e), logKeyVals...)
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

func startFetchingEvents(eventBus *events.Bus, logger tmlog.Logger) context.CancelFunc {
	ctx, shutdownBus := context.WithCancel(context.Background())
	errs := eventBus.FetchEvents(ctx)

	go func() {
		select {
		case err := <-errs:
			logger.Error(err.Error())
			shutdownBus()
		case <-ctx.Done():
			return
		}
	}()
	return shutdownBus
}
