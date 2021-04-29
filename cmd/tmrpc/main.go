package main

import (
	"flag"
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/client"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/events"
	tmtypes "github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/tendermint/tendermint/libs/pubsub/query"
	"os"
	"strings"
)

var address = flag.String("address", client.DefaultAddress, "tendermint RPC address")
var endpoint = flag.String("endpoint", client.DefaultWSEndpoint, "websocket endpoint")

type Cmd struct {
	Name    string
	Usage   string
	MinArgs uint
	Target  func(*Context, []string) error
}

var commandList = []Cmd{CmdWaitAction, CmdWaitQuery, CmdWaitTxEvent, CmdQuery}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	var usages []string
	for _, c := range commandList {
		usages = append(usages, strings.Join([]string{"  ", c.Name, c.Usage}, " "))
	}

	fmt.Printf("Usage: tmrpc [flags] COMMAND\n\nCommands:\n")
	fmt.Printf(strings.Join(usages, "\n"))
	fmt.Printf("\n\nFlags:\n")
	flag.PrintDefaults()
}

func makeContext(rpcUrl, endpoint string) (*Context, error) {
	cl, err := client.NewClient(rpcUrl, endpoint)
	if err != nil {
		return nil, err
	}

	hub := events.NewHub(cl)
	return &Context{
		TM:  cl,
		Hub: &hub,
	}, nil
}

func run() error {
	flag.Parse()
	commandName := flag.Arg(0)

	var args []string
	if flag.NArg() > 1 {
		args = flag.Args()[1:]
	}

	rpcUrl := *address
	endpoint := *endpoint

	commands := make(map[string]Cmd)
	for _, cmd := range commandList {
		commands[cmd.Name] = cmd
	}

	if cmd, ok := commands[commandName]; ok {
		if len(args) < int(cmd.MinArgs) {
			return newErrInsufficientArgs(cmd.MinArgs)
		}

		ctx, err := makeContext(rpcUrl, endpoint)
		if err != nil {
			return err
		}

		return cmd.Target(ctx, args)
	}

	usage()
	return nil
}

func getSupportedModules() map[string]struct{} {
	modules := []string{"tss", "snapshot", "broadcast", "bridge", "bridge"}

	set := make(map[string]struct{}, len(modules))
	for _, m := range modules {
		set[m] = struct{}{}
	}
	return set
}

type Context struct {
	TM  *client.RPCClient
	Hub *events.Hub
}

var CmdWaitAction = Cmd{
	Name:    "wait-action",
	Usage:   "<event type> <module> <action>",
	MinArgs: 3,
	Target: func(c *Context, args []string) error {
		eventType := args[0]
		module := args[1]
		action := args[2]

		wait, err := events.WaitActionAsync(c.Hub, eventType, module, action)
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
var CmdWaitQuery = Cmd{
	Name:    "wait-query",
	Usage:   "<query>",
	MinArgs: 1,
	Target: func(c *Context, args []string) error {
		wait, err := events.WaitQueryAsync(c.Hub, query.MustParse(args[0]))
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

var CmdWaitTxEvent = Cmd{
	Name:    "wait-tx-event",
	Usage:   "<event type> <module> [action]",
	MinArgs: 2,
	Target: func(c *Context, args []string) error {
		if len(args) < 2 {
			return newErrInsufficientArgs(2)
		}
		eventType := args[0]
		module := args[1]

		params := []tmtypes.Param{tmtypes.Param{Key: sdk.AttributeKeyModule, Value: module}}

		if len(args) > 2 {
			action := args[2]
			params = append(params, tmtypes.Param{Key: sdk.AttributeKeyAction, Value: action})
		}

		wait, err := events.WaitQueryAsync(c.Hub, tmtypes.MakeTxEventQuery(eventType, params...))
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

var CmdQuery = Cmd{
	Name:    "query",
	Usage:   "<query>",
	MinArgs: 1,
	Target: func(c *Context, args []string) error {
		catch, err := events.ProcessQuery(c.Hub, query.MustParse(args[0]), reportEventVerbose)
		if err != nil {
			return err
		}

		return catch()
	},
}

func newErrInsufficientArgs(num uint) error {
	return fmt.Errorf("expected %v arguments", num)
}

func reportEventVerbose(event tmtypes.Event) {
	reportEvent(event)
	fmt.Printf("%+v\n", event)
}

func reportEvent(event tmtypes.Event) {
	fmt.Printf("\tmodule: %+v\n", event.Module)
	fmt.Printf("\taction: %+v\n", event.Action)
	fmt.Printf("\ttype: %+v\n", event.Type)
	fmt.Printf("\tattributes: %+v\n", event.Attributes)
}
