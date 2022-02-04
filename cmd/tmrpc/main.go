package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/axelarnetwork/utils/jobs"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub/query"

	"github.com/axelarnetwork/tm-events/events"
	"github.com/axelarnetwork/tm-events/pubsub"
	"github.com/axelarnetwork/tm-events/tendermint"
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
			cl, err := tendermint.StartClient(rpcURL, endpoint, logger)
			if err != nil {
				return sdkerrors.Wrap(err, "failed to create tendermint client")
			}

			notifier := events.NewBlockNotifier(events.NewBlockClient(cl), logger)
			blockSource := events.NewBlockSource(cl, notifier)
			eventBus = events.NewEventBus(blockSource, pubsub.NewBus, logger)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&rpcURL, "address", tendermint.DefaultAddress, "tendermint RPC address")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", tendermint.DefaultWSEndpoint, "websocket endpoint")

	cmd.AddCommand(
		CmdWaitEvent(&eventBus, logger),
		CmdWaitQuery(&eventBus, logger),
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
			sub := events.MustSubscribeWithAttributes(eventBus, eventType, module, attributes...)
			once := sync.Once{}
			job := events.Consume(sub, func(e events.Event) {
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

// CmdWaitQuery returns a cli command to wait for the first event matching the given query
func CmdWaitQuery(busPtr **events.Bus, logger tmlog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-query <query string>",
		Short: "Wait for the next event matching the query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventBus := *busPtr
			shutdownBus := startFetchingEvents(eventBus, logger)

			q := events.Query{TMQuery: query.MustParse(args[0]), Predicate: func(events.Event) bool { return true }}
			logger.Debug(fmt.Sprintf("waiting for event matching query '%s'", q.TMQuery.String()))
			sub := events.MustSubscribe(eventBus, q, func(err error) error { return sdkerrors.Wrap(err, "query subscription failed") })
			once := sync.Once{}
			job := events.Consume(sub, func(e events.Event) {
				once.Do(func() {
					logger.Debug(fmt.Sprintf("found event matching query '%s': %v", q.TMQuery.String(), e))
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
