package main

import (
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub/query"

	"github.com/axelarnetwork/tm-events/events"
	"github.com/axelarnetwork/tm-events/pubsub"
	"github.com/axelarnetwork/tm-events/tendermint"
	events2 "github.com/axelarnetwork/tm-events/tendermint/events"
	"github.com/axelarnetwork/tm-events/tendermint/types"
)

func main() {
	var (
		rpcURL   string
		endpoint string
		eventBus *events.Bus
	)
	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))

	cmd := cobra.Command{
		Use:              "tmrpc",
		Short:            "Event listener",
		TraverseChildren: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cl, err := tendermint.NewClient(rpcURL, endpoint, logger)
			if err != nil {
				return nil
			}

			blockSource := events.NewBlockSource(cl, events.NewBlockNotifier(events.NewBlockClient(cl), logger))
			eventBus = events.NewEventBus(blockSource, pubsub.NewBus, logger)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&rpcURL, "address", tendermint.DefaultAddress, "tendermint RPC address")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", tendermint.DefaultWSEndpoint, "websocket endpoint")

	cmd.AddCommand(
		CmdQuery(eventBus),
		CmdWaitQuery(eventBus, logger),
		CmdWaitTxEvent(eventBus, logger),
		CmdWaitAction(eventBus, logger),
	)

	if err := cmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func CmdWaitAction(eventBus *events.Bus, logger tmlog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-action <event type> <module> <action>",
		Short: "Wait for the next matching event",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventType := args[0]
			module := args[1]
			action := args[2]

			wait, err := events2.WaitActionAsync(eventBus, eventType, module, action, logger)
			if err != nil {
				return err
			}

			ev, err := wait()
			if err != nil {
				return err
			}

			reportEvent(ev)
			return nil
		},
	}
	return cmd
}

func CmdWaitQuery(eventBus *events.Bus, logger tmlog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-query <query string>",
		Short: "Wait for the next event matching the query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait, err := events2.WaitQueryAsync(eventBus, query.MustParse(args[0]), logger)
			if err != nil {
				return err
			}

			ev, err := wait()
			if err != nil {
				return err
			}

			reportEvent(ev)
			return nil
		},
	}
	return cmd
}

func CmdWaitTxEvent(eventBus *events.Bus, logger tmlog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-tx-event <event type> <module> [action]",
		Short: "Wait for the next matching tx event",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventType := args[0]
			module := args[1]

			params := []types.Param{{Key: sdk.AttributeKeyModule, Value: module}}

			if len(args) == 3 {
				action := args[2]
				params = append(params, types.Param{Key: sdk.AttributeKeyAction, Value: action})
			}

			wait, err := events2.WaitQueryAsync(eventBus, types.MakeTxEventQuery(eventType, params...), logger)
			if err != nil {
				return err
			}

			ev, err := wait()
			if err != nil {
				return err
			}

			reportEvent(ev)
			return nil
		},
	}
	return cmd
}

func CmdQuery(eventBus *events.Bus) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query <query string>",
		Short: "Print the next event matching the query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			catch, err := events2.ProcessQuery(eventBus, query.MustParse(args[0]), reportEventVerbose)
			if err != nil {
				return err
			}

			return catch()
		},
	}
	return cmd
}

func reportEventVerbose(event types.Event) {
	reportEvent(event)
	fmt.Printf("%+v\n", event)
}

func reportEvent(event types.Event) {
	fmt.Printf("\tmodule: %+v\n", event.Module)
	fmt.Printf("\taction: %+v\n", event.Action)
	fmt.Printf("\ttype: %+v\n", event.Type)
	fmt.Printf("\tattributes: %+v\n", event.Attributes)
}
