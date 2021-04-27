package events

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/client"
	"github.com/axelarnetwork/tm-events/test/fake"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

func createRpcClient(address string) (*client.RPCClient, error) {
	conf := client.Config{
		Address:  address,
		Endpoint: "/websocket",
	}

	cl, err := client.NewClientFromConfig(conf)
	if err != nil {
		return nil, err
	}

	err = cl.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to connect tendermint client at %s%s: %w", conf.Address, conf.Endpoint, err)
	}
	fmt.Printf("Connected tendermint client at %s\n", conf.Address)

	return cl, nil
}

func TestEventHub(t *testing.T) {
	cdc := fake.MakeCodec()
	ec := fake.NewEventsClient(context.Background(), cdc)
	ec.SourceFn = ec.TickerEventSource(time.Millisecond)
	server := fake.NewWSServerWithEventsClient(cdc, &ec)
	defer server.Close()

	rpcClient, err := createRpcClient(server.URL)
	require.NoError(t, err)

	hub := NewHub(rpcClient)

	go func() {
		<-time.After(time.Second)
		panic("time out")
	}()

	ev, err := WaitAction(&hub, "link")
	require.NoError(t, err)

	fmt.Printf("\tmodule: %+v\n", ev.Module)
	fmt.Printf("\taction: %+v\n", ev.Action)
	fmt.Printf("\ttype: %+v\n", ev.Type)
	fmt.Printf("\tattributes: %+v\n", ev.Attributes)
}

func TestMultipleSubscribers(t *testing.T) {
	sourceFn, push := fake.PushEventSource()
	cdc := fake.MakeCodec()
	ec := fake.NewEventsClient(context.Background(), cdc)
	ec.SourceFn = sourceFn
	server := fake.NewWSServerWithEventsClient(cdc, &ec)
	defer server.Close()

	rpcClient, err := createRpcClient(server.URL)
	require.NoError(t, err)

	// create identical queries with different memory addresses
	key := "message.fake"
	val := "fake"
	q1 := tmquery.MustParse(fmt.Sprintf("%s='%s'", key, val))
	q2 := tmquery.MustParse(q1.String())
	events := make(map[string][]string)
	events[key] = []string{val}

	next := eventGenerator(q1.String(), events)

	// set up relays
	hub := NewHub(rpcClient)
	relay1, err := hub.Subscribe(q1)
	require.NoError(t, err)

	relay2, err := hub.Subscribe(q2)
	require.NoError(t, err)

	// todo: test same relay is returned
	//assert.Equal(t, relay1, relay2)

	s1, err := relay1.Subscribe()
	require.NoError(t, err)
	s2, err := relay2.Subscribe()
	require.NoError(t, err)

	// push a dummy event to the query subscribers
	ev := next()
	eventDataTx := ev.Data.(types.EventDataTx)
	expected, ok := ProcessEvent(eventDataTx.Result.GetEvents()[0])
	expected.Height = eventDataTx.Height
	require.True(t, ok)

	push <- ev

	for _, sub := range []pubsub.Subscriber{s1, s2} {
		select {
		case newEv := <-sub.Events():
			assert.Equal(t, expected, newEv)
		case <-pubsub.AfterThreadStart(t):
			assert.Fail(t, "time out")
		}
	}
}

func TestMultipleSubscribersToSameRelay(t *testing.T) {
	sourceFn, push := fake.PushEventSource()
	cdc := fake.MakeCodec()
	ec := fake.NewEventsClient(context.Background(), cdc)
	ec.SourceFn = sourceFn
	server := fake.NewWSServerWithEventsClient(cdc, &ec)
	defer server.Close()

	rpcClient, err := createRpcClient(server.URL)
	require.NoError(t, err)

	key := "message.fake"
	val := "fake"
	q := tmquery.MustParse(fmt.Sprintf("%s='%s'", key, val))
	events := make(map[string][]string)
	events[key] = []string{val}
	next := eventGenerator(q.String(), events)

	hub := NewHub(rpcClient)

	relay, err := hub.Subscribe(q)
	require.NoError(t, err)

	// each subscriber to the relays should receive a copy of the same event
	s1, err := relay.Subscribe()
	require.NoError(t, err)
	s2, err := relay.Subscribe()
	require.NoError(t, err)
	s3, err := relay.Subscribe()
	require.NoError(t, err)

	// push a dummy event to the query subscribers
	ev := next()
	eventDataTx := ev.Data.(types.EventDataTx)
	expected, ok := ProcessEvent(eventDataTx.Result.GetEvents()[0])
	expected.Height = eventDataTx.Height
	require.True(t, ok)

	push <- ev

	for _, sub := range []pubsub.Subscriber{s1, s2, s3} {
		select {
		case newEv := <-sub.Events():
			assert.Equal(t, expected, newEv)
		case <-pubsub.AfterThreadStart(t):
			assert.Fail(t, "time out")
		}
	}
}

func eventGenerator(query string, eventsMap map[string][]string) func() coretypes.ResultEvent {
	var events []abci.Event
	for key, ev := range eventsMap {
		events = append(events, abci.Event{
			Type: key,
			Attributes: []abci.EventAttribute{{
				Key:   []byte(key),
				Value: []byte(ev[0]),
			}},
		})
	}

	var i int64 = 0
	return func() coretypes.ResultEvent {
		i++
		return coretypes.ResultEvent{
			Query:  query,
			Events: eventsMap,
			Data: types.EventDataTx{
				TxResult: abci.TxResult{
					Height: i,
					Index:  0,
					Tx:     nil,
					Result: abci.ResponseDeliverTx{
						Code:      0,
						Data:      nil,
						Log:       "",
						Info:      "",
						GasWanted: 0,
						GasUsed:   0,
						Events:    events,
						Codespace: "",
					},
				},
			},
		}
	}
}
