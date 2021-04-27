package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/axelarnetwork/axelar-core/app"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	tmrpctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/rpc/jsonrpc/server"
	tmtypes "github.com/tendermint/tendermint/types"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"time"

	amino "github.com/tendermint/go-amino"

	"github.com/tendermint/tendermint/libs/log"
	types "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

func MakeCodec() *codec.LegacyAmino {
	var cdc = codec.New()
	tmrpctypes.RegisterAmino(cdc)
	app.ModuleBasics.RegisterCodec(cdc)
	vesting.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	//codec.RegisterCrypto(cdc)
	return cdc
}

// github.com/tendermint/tendermint@v0.33.7/rpc/jsonrpc/server/ws_handler_test.go
func NewWSServer() *httptest.Server {
	cdc := MakeCodec()

	ec := NewEventsClient(context.Background(), cdc)
	wsClient := NewWSClient(&ec)

	funcMap := map[string]*server.RPCFunc{
		"subscribe": server.NewWSRPCFunc(wsClient.SubscribeWS, "query"),
	}
	wm := server.NewWebsocketManager(funcMap, cdc)
	wm.SetLogger(log.TestingLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", wm.WebsocketHandler)

	return httptest.NewServer(mux)
}

func NewWSServerWithEventsClient(cdc *codec.LegacyAmino, ec *EventsClient) *httptest.Server {
	wsClient := NewWSClient(ec)

	funcMap := map[string]*server.RPCFunc{
		"subscribe": server.NewWSRPCFunc(wsClient.SubscribeWS, "query"),
	}
	wm := server.NewWebsocketManager(funcMap, cdc)
	wm.SetLogger(log.TestingLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", wm.WebsocketHandler)

	return httptest.NewServer(mux)
}

// Fake websocket client
type WSClient struct {
	EventsClient *EventsClient
	cdc          *amino.Codec
}

func NewWSClient(ec *EventsClient) WSClient {
	return WSClient{
		EventsClient: ec,
	}
}

func (w WSClient) SubscribeWS(ctx *types.Context, query string) (*tmrpctypes.ResultSubscribe, error) {
	out, err := w.EventsClient.Subscribe(context.Background(), ctx.RemoteAddr(), query)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case resultEvent := <-out:
				ctx.WSConn.TryWriteRPCResponse(
					types.NewRPCSuccessResponse(
						ctx.WSConn.Codec(),
						ctx.JSONReq.ID,
						resultEvent,
					))
			case <-w.EventsClient.Quit():
				return
			}
		}
	}()

	return &tmrpctypes.ResultSubscribe{}, nil
}

type EventsClient struct {
	ctx           context.Context
	subscriptions map[string]chan tmrpctypes.ResultEvent
	cdc           *amino.Codec
	SourceFn      EventSourceFn
}

func NewEventsClient(ctx context.Context, cdc *amino.Codec) EventsClient {
	ec := EventsClient{
		subscriptions: make(map[string]chan tmrpctypes.ResultEvent),
		ctx:           ctx,
		cdc:           cdc,
	}
	ec.SourceFn = ec.TickerEventSource(time.Second)
	return ec
}

func NewEventsClientWithSource(ctx context.Context, cdc *amino.Codec, sourceFn EventSourceFn) EventsClient {
	return EventsClient{
		subscriptions: make(map[string]chan tmrpctypes.ResultEvent),
		ctx:           ctx,
		cdc:           cdc,
		SourceFn:      sourceFn,
	}
}

const buffsize = 100

type EventSourceFn func(ctx context.Context, out chan<- tmrpctypes.ResultEvent, query string)

func (c *EventsClient) TickerEventSource(tick time.Duration) EventSourceFn {
	return func(ctx context.Context, out chan<- tmrpctypes.ResultEvent, query string) {
		dummy, err := readTxResultEventJson(c.cdc, eventDataPath("event_votepubkey.json"))
		if err != nil {
			panic(err)
		}

		dummy.Query = query

		ticker := time.NewTicker(tick)
		go func() {
			for {
				select {
				case <-ctx.Done():
				case <-ticker.C:
					out <- dummy
				}
			}
		}()
	}
}

// PushEventSource returns a channel which can be used to manually push events to the websocket
func PushEventSource() (EventSourceFn, chan<- tmrpctypes.ResultEvent) {
	in := make(chan tmrpctypes.ResultEvent)

	return func(ctx context.Context, out chan<- tmrpctypes.ResultEvent, query string) {
		go func() {
			for {
				select {
				case <-ctx.Done():
				case ev := <-in:
					ev.Query = query
					out <- ev
				}
			}
		}()
	}, in
}

func (c *EventsClient) getSubscription(query string, sourceFn EventSourceFn) (<-chan tmrpctypes.ResultEvent, error) {
	if _, ok := c.subscriptions[query]; ok {
		return nil, fmt.Errorf("subscription already exists for query %s", query)
	}

	out := make(chan tmrpctypes.ResultEvent, buffsize)
	c.subscriptions[query] = out

	sourceFn(c.ctx, out, query)
	return out, nil
}

// More: https://docs.tendermint.com/master/rpc/#/Websocket/subscribe
// github.com/tendermint/tendermint@v0.33.7/rpc/core/filtered.go
func (c *EventsClient) Subscribe(ctx context.Context, subscriber, query string, outCapacity ...int) (<-chan tmrpctypes.ResultEvent, error) {
	out, err := c.getSubscription(query, c.SourceFn)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *EventsClient) Quit() <-chan struct{} {
	return c.ctx.Done()
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

func eventDataPath(fname string) string {
	return basepath + "/../data/" + fname
}

func readTxResultEventJson(cdc *amino.Codec, path string) (ev tmrpctypes.ResultEvent, err error) {
	// @todo use simple action event
	jsonFile, err := os.Open(path)
	if err != nil {
		return
	}
	defer jsonFile.Close()

	bs, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	//var tmp tmrpctypes.ResultEvent
	err = json.Unmarshal(bs, &ev)
	if err != nil {
		return
	}

	if data, ok := ev.Data.(map[string]interface{}); ok {
		val := data["value"]

		bsTxResult, _err := json.Marshal(val)
		if _err != nil {
			err = _err
			return
		}

		var result tmtypes.TxResult
		_err = cdc.UnmarshalJSON(bsTxResult, &result)
		if _err != nil {
			err = _err
			return
		}

		nd := tmtypes.EventDataTx{result}
		ev.Data = nd
	}
	return
}
